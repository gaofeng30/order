package fulfillment

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"time"

	"github.com/gaofeng30/order/services/api/internal/audit"
	"github.com/gaofeng30/order/services/api/internal/merchantidentity"
	"github.com/gaofeng30/order/services/api/internal/orderproduction"
	"github.com/gaofeng30/order/services/api/internal/orderquery"
	"github.com/gaofeng30/order/services/api/internal/subscription"
	mysqlDriver "github.com/go-sql-driver/mysql"
)

const (
	actionMarkReady             = "fulfillment.mark_ready"
	actionRedeemToken           = "fulfillment.redeem_token"
	actionRedeemCurrentDateCode = "fulfillment.redeem_current_date_code"
	actionRedeemOrder           = "fulfillment.redeem_order"
)

var fulfillmentLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

// MySQLApplication owns fulfillment state changes and verification lookups.
// Every transaction locks the current merchant account before one order and
// appends its command receipt last.
type MySQLApplication struct {
	db            *sql.DB
	authorizer    merchantidentity.Authorizer
	receipts      *audit.ReceiptStore
	cipher        TokenCipher
	notifications NotificationEnqueuer
	random        io.Reader
	now           func() time.Time
	location      *time.Location
}

var _ Application = (*MySQLApplication)(nil)

// NotificationEnqueuer appends a subscription intent inside the fulfillment transaction.
type NotificationEnqueuer interface {
	EnqueueInTx(context.Context, *sql.Tx, subscription.NotificationIntent) error
}

func NewMySQLApplication(db *sql.DB, authorizer merchantidentity.Authorizer, cipher TokenCipher, notifications NotificationEnqueuer) *MySQLApplication {
	return &MySQLApplication{
		db: db, authorizer: authorizer, receipts: audit.NewReceiptStore(db), cipher: cipher,
		notifications: notifications, random: rand.Reader, now: time.Now, location: fulfillmentLocation,
	}
}

type lockedOrder struct {
	id             uint64
	userID         uint64
	orderNumber    string
	state          orderquery.State
	pickupDate     string
	pickupTime     string
	pickupPoint    string
	pickupNumber   uint16
	preparingAt    sql.NullTime
	readyAt        sql.NullTime
	completedAt    sql.NullTime
	redemptionHash []byte
	ciphertext     []byte
	keyVersion     sql.NullInt64
	issuedAt       sql.NullTime
	redeemedBy     sql.NullInt64
	redeemedAt     sql.NullTime
	recordVersion  uint64
}

type receiptRequest struct {
	Kind         CommandKind `json:"kind"`
	OrderID      uint64      `json:"order_id,omitempty"`
	Token        string      `json:"token,omitempty"`
	PickupNumber string      `json:"pickup_number,omitempty"`
}

type receiptResponse struct {
	OrderID uint64           `json:"order_id"`
	State   orderquery.State `json:"state"`
	Changed bool             `json:"changed"`
}

type commandLocator struct {
	live         bool
	tokenHash    [sha256.Size]byte
	pickupDate   string
	pickupNumber uint16
}

func (application *MySQLApplication) Execute(ctx context.Context, meta WriteMeta, command Command) (Result, error) {
	if !validWrite(meta) || !validCommand(command) || ctx == nil {
		return Result{}, ErrInvalidInput
	}
	if !validApplication(application) {
		return Result{}, ErrUnavailable
	}
	action, identityAction, ok := commandActions(command.Kind)
	if !ok {
		return Result{}, ErrInvalidInput
	}
	request := receiptRequestFor(command)
	command, locator, err := application.resolveCommand(ctx, meta, command, request, action)
	if err != nil {
		return Result{}, err
	}

	for attempt := 0; attempt < 2; attempt++ {
		result, err := application.executeOnce(ctx, meta, command, request, locator, action, identityAction)
		if err == nil {
			return result, nil
		}
		if errors.Is(err, audit.ErrDuplicateReceipt) {
			for replayAttempt := 0; replayAttempt < 2; replayAttempt++ {
				replay, replayErr := application.replayAfterDuplicate(ctx, meta, command, request, action, identityAction)
				if replayErr == nil {
					return replay, nil
				}
				if !retryableTransaction(replayErr) || replayAttempt == 1 {
					return Result{}, mapApplicationError(replayErr)
				}
			}
		}
		if !retryableTransaction(err) || attempt == 1 {
			return Result{}, mapApplicationError(err)
		}
	}
	return Result{}, ErrUnavailable
}

func (application *MySQLApplication) executeOnce(
	ctx context.Context,
	meta WriteMeta,
	command Command,
	request receiptRequest,
	locator commandLocator,
	action string,
	identityAction merchantidentity.Action,
) (Result, error) {
	transaction, err := application.db.BeginTx(ctx, nil)
	if err != nil {
		return Result{}, err
	}
	defer transaction.Rollback()

	authorization, err := application.authorizer.AuthorizeInTx(ctx, transaction, meta.ActorUserID, identityAction, merchantidentity.Target{Type: "ORDER", ID: command.OrderID})
	if err != nil {
		return Result{}, err
	}
	if !validAuthorization(authorization) {
		return Result{}, ErrUnavailable
	}
	order, err := lockOrder(ctx, transaction, command.OrderID)
	if err != nil {
		return Result{}, err
	}
	if replay, found, err := replayReceiptInTx(ctx, transaction, meta, authorization, action, request, command.OrderID); err != nil {
		return Result{}, err
	} else if found {
		if err := transaction.Commit(); err != nil {
			return Result{}, err
		}
		return Result{OrderID: replay.OrderID, State: replay.State, Changed: replay.Changed, Replay: true}, nil
	}
	if !locator.live || !locatorMatches(order, command.Kind, locator) {
		return Result{}, ErrRedemptionInvalid
	}

	now := application.now().UTC().Truncate(time.Microsecond)
	if now.IsZero() {
		return Result{}, ErrUnavailable
	}
	if (command.Kind == CommandRedeemToken || command.Kind == CommandRedeemCurrentDateCode) && validCompletedOrder(order, now) {
		role, ok := receiptRole(authorization.Actor)
		if !ok {
			return Result{}, ErrUnavailable
		}
		response := receiptResponse{OrderID: order.id, State: orderquery.StateCompleted, Changed: true}
		if err := application.receipts.AppendInTx(ctx, transaction, audit.CommandMeta{
			ActorUserID: meta.ActorUserID, ActorAccountID: authorization.MerchantAccountID,
			ActorRole: role, ActorAuthVersion: authorization.AuthVersion,
			IdempotencyKey: meta.IdempotencyKey, RequestID: meta.RequestID,
		}, action, "ORDER", command.OrderID, request, response); err != nil {
			return Result{}, err
		}
		if err := transaction.Commit(); err != nil {
			return Result{}, err
		}
		return Result{OrderID: response.OrderID, State: response.State, Changed: response.Changed, Replay: true}, nil
	}
	var tokenHash [sha256.Size]byte
	var keyVersion uint16
	var ciphertext []byte
	if command.Kind == CommandMarkReady {
		plain := make([]byte, 32)
		if _, err := io.ReadFull(application.random, plain); err != nil {
			return Result{}, ErrUnavailable
		}
		token := base64.RawURLEncoding.EncodeToString(plain)
		tokenHash = sha256.Sum256([]byte(token))
		keyVersion, ciphertext, err = application.cipher.Seal(ctx, token)
		if err != nil || keyVersion == 0 || len(ciphertext) == 0 || len(ciphertext) > 192 {
			return Result{}, ErrUnavailable
		}
	}
	if command.Kind != CommandMarkReady && order.state == orderquery.StateReadyForPickup {
		if err := application.validateCurrentRedemption(ctx, order, now); err != nil {
			return Result{}, err
		}
	}
	response, err := applyTransition(ctx, transaction, authorization, command, order, now, tokenHash, keyVersion, ciphertext)
	if err != nil {
		return Result{}, err
	}
	if command.Kind == CommandMarkReady {
		if err := application.notifications.EnqueueInTx(ctx, transaction, subscription.NotificationIntent{
			OrderID:         order.id,
			RecipientUserID: order.userID,
			Kind:            subscription.KindReady,
			Message: subscription.Message{
				OrderNumber: order.orderNumber,
				PickupDate:  order.pickupDate,
				PickupTime:  order.pickupTime,
				PickupPoint: order.pickupPoint,
			},
			AvailableAt: now,
		}); err != nil {
			return Result{}, err
		}
	}
	role, ok := receiptRole(authorization.Actor)
	if !ok {
		return Result{}, ErrUnavailable
	}
	if err := application.receipts.AppendInTx(ctx, transaction, audit.CommandMeta{
		ActorUserID: meta.ActorUserID, ActorAccountID: authorization.MerchantAccountID,
		ActorRole: role, ActorAuthVersion: authorization.AuthVersion,
		IdempotencyKey: meta.IdempotencyKey, RequestID: meta.RequestID,
	}, action, "ORDER", command.OrderID, request, response); err != nil {
		return Result{}, err
	}
	if err := transaction.Commit(); err != nil {
		return Result{}, err
	}
	return Result{OrderID: response.OrderID, State: response.State, Changed: response.Changed}, nil
}

func applyTransition(
	ctx context.Context,
	transaction *sql.Tx,
	authorization merchantidentity.Authorization,
	command Command,
	order lockedOrder,
	now time.Time,
	tokenHash [sha256.Size]byte,
	keyVersion uint16,
	ciphertext []byte,
) (receiptResponse, error) {
	trigger := orderproduction.TriggerMerchantMarkReady
	if command.Kind != CommandMarkReady {
		trigger = orderproduction.TriggerRedeemSucceeded
	}
	decision, err := orderproduction.Transition(orderproduction.TransitionInput{Current: order.state, Trigger: trigger})
	if err != nil {
		return receiptResponse{}, ErrTransitionNotAllowed
	}
	if !decision.Changed {
		return receiptResponse{}, ErrTransitionNotAllowed
	}

	switch command.Kind {
	case CommandMarkReady:
		if !validPreparingOrder(order, now) || keyVersion == 0 || len(ciphertext) == 0 || len(ciphertext) > 192 {
			return receiptResponse{}, ErrUnavailable
		}
		result, err := transaction.ExecContext(ctx, `UPDATE orders SET state='READY_FOR_PICKUP',ready_at=?,redemption_token_ciphertext=?,redemption_token_hash=?,redemption_key_version=?,redemption_issued_at=?,record_version=record_version+1,updated_at=? WHERE id=? AND state='PREPARING' AND record_version=?`, now, ciphertext, tokenHash[:], keyVersion, now, now, order.id, order.recordVersion)
		if err != nil {
			return receiptResponse{}, err
		}
		if !exactlyOne(result) {
			return receiptResponse{}, ErrUnavailable
		}
	case CommandRedeemToken, CommandRedeemCurrentDateCode, CommandRedeemOrder:
		if !validReadyOrder(order, now) || authorization.MerchantAccountID == 0 {
			return receiptResponse{}, ErrUnavailable
		}
		result, err := transaction.ExecContext(ctx, `UPDATE orders SET state='COMPLETED',completed_at=?,redemption_token_ciphertext=NULL,redemption_key_version=NULL,redeemed_by_account_id=?,redeemed_at=?,record_version=record_version+1,updated_at=? WHERE id=? AND state='READY_FOR_PICKUP' AND record_version=?`, now, authorization.MerchantAccountID, now, now, order.id, order.recordVersion)
		if err != nil {
			return receiptResponse{}, err
		}
		if !exactlyOne(result) {
			return receiptResponse{}, ErrUnavailable
		}
	default:
		return receiptResponse{}, ErrInvalidInput
	}
	return receiptResponse{OrderID: order.id, State: decision.State, Changed: true}, nil
}

func (application *MySQLApplication) replayAfterDuplicate(ctx context.Context, meta WriteMeta, command Command, request receiptRequest, action string, identityAction merchantidentity.Action) (Result, error) {
	transaction, err := application.db.BeginTx(ctx, nil)
	if err != nil {
		return Result{}, err
	}
	defer transaction.Rollback()
	authorization, err := application.authorizer.AuthorizeInTx(ctx, transaction, meta.ActorUserID, identityAction, merchantidentity.Target{Type: "ORDER", ID: command.OrderID})
	if err != nil {
		return Result{}, err
	}
	if !validAuthorization(authorization) {
		return Result{}, ErrUnavailable
	}
	if _, err := lockOrder(ctx, transaction, command.OrderID); err != nil {
		return Result{}, err
	}
	replay, found, err := replayReceiptInTx(ctx, transaction, meta, authorization, action, request, command.OrderID)
	if err != nil || !found {
		if err != nil {
			return Result{}, err
		}
		return Result{}, ErrUnavailable
	}
	if err := transaction.Commit(); err != nil {
		return Result{}, err
	}
	return Result{OrderID: replay.OrderID, State: replay.State, Changed: replay.Changed, Replay: true}, nil
}

func (application *MySQLApplication) resolveCommand(ctx context.Context, meta WriteMeta, command Command, request receiptRequest, action string) (Command, commandLocator, error) {
	locator := commandLocator{live: true}
	switch command.Kind {
	case CommandMarkReady, CommandRedeemOrder:
		return command, locator, nil
	case CommandRedeemToken:
		locator.tokenHash = sha256.Sum256([]byte(command.Token))
		var orderID uint64
		err := application.db.QueryRowContext(ctx, `SELECT id FROM orders WHERE redemption_token_hash=? LIMIT 1`, locator.tokenHash[:]).Scan(&orderID)
		if err == nil && orderID > 0 {
			command.OrderID = orderID
			return command, locator, nil
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return Command{}, commandLocator{}, ErrUnavailable
		}
		locator.live = false
	case CommandRedeemCurrentDateCode:
		now := application.now()
		if now.IsZero() {
			return Command{}, commandLocator{}, ErrUnavailable
		}
		locator.pickupDate = now.In(application.location).Format("2006-01-02")
		number, _ := strconv.ParseUint(command.PickupNumber, 10, 16)
		locator.pickupNumber = uint16(number)
		var orderID uint64
		err := application.db.QueryRowContext(ctx, `SELECT id FROM orders WHERE pickup_date=? AND pickup_number=? LIMIT 1`, locator.pickupDate, locator.pickupNumber).Scan(&orderID)
		if err == nil && orderID > 0 {
			command.OrderID = orderID
			return command, locator, nil
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return Command{}, commandLocator{}, ErrUnavailable
		}
		locator.live = false
	default:
		return Command{}, commandLocator{}, ErrInvalidInput
	}

	orderID, err := application.locateReceiptTarget(ctx, meta, action, request)
	if err != nil {
		return Command{}, commandLocator{}, err
	}
	command.OrderID = orderID
	return command, locator, nil
}

func (application *MySQLApplication) locateReceiptTarget(ctx context.Context, meta WriteMeta, action string, request receiptRequest) (uint64, error) {
	var accountID uint64
	err := application.db.QueryRowContext(ctx, `SELECT id FROM merchant_accounts WHERE bound_user_id=? AND enabled=TRUE AND deleted_at IS NULL LIMIT 1`, meta.ActorUserID).Scan(&accountID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrForbidden
	}
	if err != nil || accountID == 0 {
		return 0, ErrUnavailable
	}
	scope := merchantScopeHash(meta.ActorUserID, accountID)
	operation := sha256.Sum256([]byte("operation:" + meta.IdempotencyKey))
	rows, err := application.db.QueryContext(ctx, `SELECT target_id,before_state_json,response_json FROM action_audits WHERE entry_kind='COMMAND_RECEIPT' AND actor_kind='MERCHANT' AND actor_scope_hash=? AND action=? AND operation_key_hash=? ORDER BY id LIMIT 2`, scope[:], action, operation[:])
	if err != nil {
		return 0, ErrUnavailable
	}
	defer rows.Close()
	var count int
	var orderID uint64
	for rows.Next() {
		count++
		var targetID uint64
		var evidence, responseRaw []byte
		if rows.Scan(&targetID, &evidence, &responseRaw) != nil {
			return 0, ErrUnavailable
		}
		requestRaw, marshalErr := json.Marshal(request)
		if marshalErr != nil {
			return 0, ErrUnavailable
		}
		digest := sha256.Sum256(requestRaw)
		if matchErr := matchReceiptEvidence(evidence, digest); matchErr != nil {
			return 0, matchErr
		}
		response, ok := decodeReceiptResponse(responseRaw)
		if !ok || response.OrderID != targetID || response.State != orderquery.StateCompleted {
			return 0, ErrUnavailable
		}
		orderID = targetID
	}
	if rows.Err() != nil || count > 1 {
		return 0, ErrUnavailable
	}
	if count == 0 || orderID == 0 {
		return 0, ErrRedemptionInvalid
	}
	return orderID, nil
}

const lockedOrderColumns = `id,user_id,order_no,state,DATE_FORMAT(pickup_date,'%Y-%m-%d'),TIME_FORMAT(pickup_time,'%H:%i'),pickup_point_snapshot,pickup_number,preparing_at,ready_at,completed_at,redemption_token_hash,redemption_token_ciphertext,redemption_key_version,redemption_issued_at,redeemed_by_account_id,redeemed_at,record_version`

func lockOrder(ctx context.Context, transaction *sql.Tx, orderID uint64) (lockedOrder, error) {
	return scanLockedOrder(transaction.QueryRowContext(ctx, `SELECT `+lockedOrderColumns+` FROM orders WHERE id=? FOR UPDATE`, orderID))
}

func scanLockedOrder(row *sql.Row) (lockedOrder, error) {
	var order lockedOrder
	var state string
	err := row.Scan(&order.id, &order.userID, &order.orderNumber, &state, &order.pickupDate, &order.pickupTime, &order.pickupPoint, &order.pickupNumber, &order.preparingAt, &order.readyAt, &order.completedAt, &order.redemptionHash, &order.ciphertext, &order.keyVersion, &order.issuedAt, &order.redeemedBy, &order.redeemedAt, &order.recordVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return lockedOrder{}, ErrNotFound
	}
	if err != nil {
		return lockedOrder{}, err
	}
	order.state = orderquery.State(state)
	if order.id == 0 || order.recordVersion == 0 || !validDate(order.pickupDate) || order.pickupNumber < 1 || order.pickupNumber > 9999 {
		return lockedOrder{}, ErrUnavailable
	}
	return order, nil
}

func validPreparingOrder(order lockedOrder, now time.Time) bool {
	return order.state == orderquery.StatePreparing && order.preparingAt.Valid && !order.preparingAt.Time.After(now) &&
		!order.readyAt.Valid && !order.completedAt.Valid && len(order.redemptionHash) == 0 && len(order.ciphertext) == 0 &&
		!order.keyVersion.Valid && !order.issuedAt.Valid && !order.redeemedBy.Valid && !order.redeemedAt.Valid
}

func validReadyOrder(order lockedOrder, now time.Time) bool {
	return order.state == orderquery.StateReadyForPickup && order.preparingAt.Valid && order.readyAt.Valid &&
		!order.readyAt.Time.Before(order.preparingAt.Time) && !order.readyAt.Time.After(now) && !order.completedAt.Valid &&
		len(order.redemptionHash) == sha256.Size && len(order.ciphertext) > 0 && len(order.ciphertext) <= 192 &&
		order.keyVersion.Valid && order.keyVersion.Int64 > 0 && order.keyVersion.Int64 <= 65535 && order.issuedAt.Valid &&
		!order.issuedAt.Time.Before(order.readyAt.Time) && !order.issuedAt.Time.After(now) && !order.redeemedBy.Valid && !order.redeemedAt.Valid
}

func validCompletedOrder(order lockedOrder, now time.Time) bool {
	return order.state == orderquery.StateCompleted && order.preparingAt.Valid && order.readyAt.Valid && order.completedAt.Valid &&
		!order.readyAt.Time.Before(order.preparingAt.Time) && !order.completedAt.Time.Before(order.readyAt.Time) && !order.completedAt.Time.After(now) &&
		len(order.redemptionHash) == sha256.Size && len(order.ciphertext) == 0 && !order.keyVersion.Valid && order.issuedAt.Valid &&
		!order.issuedAt.Time.Before(order.readyAt.Time) && !order.issuedAt.Time.After(order.completedAt.Time) &&
		order.redeemedBy.Valid && order.redeemedBy.Int64 > 0 && order.redeemedAt.Valid && order.redeemedAt.Time.Equal(order.completedAt.Time)
}

func (application *MySQLApplication) validateCurrentRedemption(ctx context.Context, order lockedOrder, now time.Time) error {
	if !validReadyOrder(order, now) {
		return ErrUnavailable
	}
	token, err := application.cipher.Open(ctx, uint16(order.keyVersion.Int64), order.ciphertext)
	if err != nil || !validPlainToken(token) {
		return ErrUnavailable
	}
	digest := sha256.Sum256([]byte(token))
	if subtle.ConstantTimeCompare(digest[:], order.redemptionHash) != 1 {
		return ErrUnavailable
	}
	return nil
}

func replayReceiptInTx(ctx context.Context, transaction *sql.Tx, meta WriteMeta, authorization merchantidentity.Authorization, action string, request receiptRequest, targetOrderID uint64) (receiptResponse, bool, error) {
	scope := merchantScopeHash(meta.ActorUserID, authorization.MerchantAccountID)
	operation := sha256.Sum256([]byte("operation:" + meta.IdempotencyKey))
	var evidence, responseRaw []byte
	err := transaction.QueryRowContext(ctx, `SELECT before_state_json,response_json FROM action_audits WHERE entry_kind='COMMAND_RECEIPT' AND actor_kind='MERCHANT' AND actor_scope_hash=? AND action=? AND operation_key_hash=? LIMIT 1 FOR SHARE`, scope[:], action, operation[:]).Scan(&evidence, &responseRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return receiptResponse{}, false, nil
	}
	if err != nil {
		return receiptResponse{}, false, err
	}
	requestRaw, err := json.Marshal(request)
	if err != nil {
		return receiptResponse{}, false, ErrUnavailable
	}
	digest := sha256.Sum256(requestRaw)
	if err := matchReceiptEvidence(evidence, digest); err != nil {
		return receiptResponse{}, false, err
	}
	response, ok := decodeReceiptResponse(responseRaw)
	if !ok || response.OrderID != targetOrderID || response.State != expectedReceiptState(request.Kind) {
		return receiptResponse{}, false, ErrUnavailable
	}
	return response, true, nil
}

func expectedReceiptState(kind CommandKind) orderquery.State {
	if kind == CommandMarkReady {
		return orderquery.StateReadyForPickup
	}
	return orderquery.StateCompleted
}

func merchantScopeHash(userID, accountID uint64) [sha256.Size]byte {
	return sha256.Sum256([]byte("merchant:" + strconv.FormatUint(userID, 10) + ":" + strconv.FormatUint(accountID, 10)))
}

func matchReceiptEvidence(raw []byte, digest [sha256.Size]byte) error {
	if len(raw) == 0 || len(raw) > 256 || !json.Valid(raw) {
		return ErrUnavailable
	}
	var evidence struct {
		RequestDigest string `json:"request_digest"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&evidence) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return ErrUnavailable
	}
	stored, err := hex.DecodeString(evidence.RequestDigest)
	if err != nil || len(stored) != sha256.Size {
		return ErrUnavailable
	}
	if subtle.ConstantTimeCompare(stored, digest[:]) != 1 {
		return ErrIdempotencyConflict
	}
	return nil
}

func decodeReceiptResponse(raw []byte) (receiptResponse, bool) {
	if len(raw) == 0 || len(raw) > 1024 || !json.Valid(raw) {
		return receiptResponse{}, false
	}
	var response receiptResponse
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&response) != nil || decoder.Decode(&struct{}{}) != io.EOF || response.OrderID == 0 || !response.Changed {
		return receiptResponse{}, false
	}
	switch response.State {
	case orderquery.StateReadyForPickup, orderquery.StateCompleted:
		return response, true
	default:
		return receiptResponse{}, false
	}
}

func validApplication(application *MySQLApplication) bool {
	return application != nil && application.db != nil && application.authorizer != nil && application.receipts != nil &&
		application.cipher != nil && application.notifications != nil && application.random != nil && application.now != nil && application.location != nil
}

func validWrite(meta WriteMeta) bool {
	return meta.ActorUserID > 0 && printable(meta.IdempotencyKey, 128) && printable(meta.RequestID, 64)
}

func validCommand(command Command) bool {
	switch command.Kind {
	case CommandMarkReady, CommandRedeemOrder:
		return command.OrderID > 0 && command.Token == "" && command.PickupNumber == ""
	case CommandRedeemToken:
		return command.OrderID == 0 && validPlainToken(command.Token) && command.PickupNumber == ""
	case CommandRedeemCurrentDateCode:
		if command.OrderID != 0 || command.Token != "" || len(command.PickupNumber) != 4 || !digits(command.PickupNumber) {
			return false
		}
		number, err := strconv.ParseUint(command.PickupNumber, 10, 16)
		return err == nil && number >= 1 && number <= 9999
	default:
		return false
	}
}

func receiptRequestFor(command Command) receiptRequest {
	request := receiptRequest{Kind: command.Kind}
	switch command.Kind {
	case CommandMarkReady, CommandRedeemOrder:
		request.OrderID = command.OrderID
	case CommandRedeemToken:
		request.Token = command.Token
	case CommandRedeemCurrentDateCode:
		request.PickupNumber = command.PickupNumber
	}
	return request
}

func commandActions(kind CommandKind) (string, merchantidentity.Action, bool) {
	switch kind {
	case CommandMarkReady:
		return actionMarkReady, merchantidentity.ActionOrderMarkReady, true
	case CommandRedeemToken:
		return actionRedeemToken, merchantidentity.ActionOrderRedeem, true
	case CommandRedeemCurrentDateCode:
		return actionRedeemCurrentDateCode, merchantidentity.ActionOrderRedeem, true
	case CommandRedeemOrder:
		return actionRedeemOrder, merchantidentity.ActionOrderRedeem, true
	default:
		return "", "", false
	}
}

func locatorMatches(order lockedOrder, kind CommandKind, locator commandLocator) bool {
	switch kind {
	case CommandMarkReady, CommandRedeemOrder:
		return true
	case CommandRedeemToken:
		return len(order.redemptionHash) == sha256.Size && subtle.ConstantTimeCompare(order.redemptionHash, locator.tokenHash[:]) == 1
	case CommandRedeemCurrentDateCode:
		return order.pickupDate == locator.pickupDate && order.pickupNumber == locator.pickupNumber
	default:
		return false
	}
}

func receiptRole(actor merchantidentity.Actor) (string, bool) {
	switch actor {
	case merchantidentity.ActorMerchantOwner:
		return string(merchantidentity.RoleOwner), true
	case merchantidentity.ActorMerchantSubaccount:
		return string(merchantidentity.RoleSubaccount), true
	default:
		return "", false
	}
}

func validAuthorization(value merchantidentity.Authorization) bool {
	_, roleOK := receiptRole(value.Actor)
	return roleOK && value.MerchantAccountID > 0 && value.RecordVersion > 0 && value.AuthVersion > 0
}

func printable(value string, maximum int) bool {
	if value == "" || len(value) > maximum {
		return false
	}
	for _, character := range []byte(value) {
		if character < 0x21 || character > 0x7e {
			return false
		}
	}
	return true
}

func digits(value string) bool {
	for _, character := range []byte(value) {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func validDate(value string) bool {
	parsed, err := time.Parse("2006-01-02", value)
	return err == nil && parsed.Format("2006-01-02") == value
}

func exactlyOne(result sql.Result) bool {
	if result == nil {
		return false
	}
	rows, err := result.RowsAffected()
	return err == nil && rows == 1
}

func retryableTransaction(err error) bool {
	if errors.Is(err, merchantidentity.ErrUnavailable) {
		return true
	}
	var mysqlError *mysqlDriver.MySQLError
	return errors.As(err, &mysqlError) && (mysqlError.Number == 1205 || mysqlError.Number == 1213)
}

func mapApplicationError(err error) error {
	switch {
	case err == nil:
		return ErrUnavailable
	case errors.Is(err, ErrInvalidInput), errors.Is(err, ErrTransitionNotAllowed), errors.Is(err, ErrRedemptionInvalid), errors.Is(err, ErrIdempotencyConflict), errors.Is(err, ErrForbidden), errors.Is(err, ErrNotFound):
		return err
	case errors.Is(err, audit.ErrIdempotencyConflict):
		return ErrIdempotencyConflict
	case errors.Is(err, merchantidentity.ErrMerchantAccountNotAvailable), errors.Is(err, merchantidentity.ErrForbidden):
		return ErrForbidden
	default:
		return ErrUnavailable
	}
}
