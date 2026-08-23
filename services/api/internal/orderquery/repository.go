package orderquery

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gaofeng30/order/services/api/internal/merchantidentity"
)

type MerchantIdentity interface {
	Identity(context.Context, uint64) (merchantidentity.Identity, error)
}

type TokenOpener interface {
	Open(context.Context, uint16, []byte) (string, error)
}

type Repository struct {
	db       *sql.DB
	merchant MerchantIdentity
	tokens   TokenOpener
	now      func() time.Time
}

func NewRepository(db *sql.DB, merchant MerchantIdentity, tokens TokenOpener, now func() time.Time) *Repository {
	return &Repository{db: db, merchant: merchant, tokens: tokens, now: now}
}

func (repository *Repository) ListUser(ctx context.Context, userID uint64, query UserQuery) (Page, error) {
	if repository == nil || repository.db == nil || repository.now == nil || userID == 0 || !validUserQuery(query) {
		return Page{}, ErrInvalidInput
	}
	where := []string{"user_id=?"}
	args := []any{userID}
	if query.State != "" {
		where, args = append(where, "state=?"), append(args, query.State)
	}
	if query.Active {
		where = append(where, "state IN ('RESERVED','PREPARING','READY_FOR_PICKUP')")
	}
	if query.AfterID > 0 {
		where, args = append(where, "id<?"), append(args, query.AfterID)
	}
	return repository.list(ctx, where, args, query.Limit, false)
}

func (repository *Repository) SearchMerchant(ctx context.Context, userID uint64, query MerchantQuery) (Page, error) {
	if repository == nil || repository.db == nil || repository.now == nil || userID == 0 || !validMerchantQuery(query) {
		return Page{}, ErrInvalidInput
	}
	if err := repository.authorizeMerchant(ctx, userID); err != nil {
		return Page{}, err
	}
	where := make([]string, 0, 4)
	args := make([]any, 0, 8)
	if query.State != "" {
		where, args = append(where, "state=?"), append(args, query.State)
	}
	if query.Date != "" {
		where, args = append(where, "pickup_date=?"), append(args, query.Date)
	}
	if query.Search != "" {
		where = append(where, `(CONVERT(order_no USING utf8mb4)=? OR LPAD(CAST(pickup_number AS CHAR),4,'0')=? OR CONVERT(contact_phone_snapshot USING utf8mb4) LIKE ? ESCAPE '=')`)
		args = append(args, query.Search, query.Search, "%"+escapeLike(query.Search))
	}
	if query.AfterID > 0 {
		where, args = append(where, "id<?"), append(args, query.AfterID)
	}
	return repository.list(ctx, where, args, query.Limit, true)
}

func (repository *Repository) list(ctx context.Context, where []string, args []any, limit uint16, merchant bool) (Page, error) {
	query := `SELECT id,order_no,state,DATE_FORMAT(pickup_date,'%Y-%m-%d'),TIME_FORMAT(pickup_time,'%H:%i'),pickup_point_snapshot,pickup_number,payable_cents,materialized_at,pickup_at FROM orders`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY id DESC LIMIT ?"
	args = append(args, uint64(limit)+1)
	rows, err := repository.db.QueryContext(ctx, query, args...)
	if err != nil {
		return Page{}, ErrUnavailable
	}
	defer rows.Close()
	now := repository.now().UTC()
	if now.IsZero() {
		return Page{}, ErrUnavailable
	}
	orders := make([]Summary, 0, limit)
	for rows.Next() {
		summary, pickupAt, err := scanSummary(rows)
		if err != nil {
			return Page{}, ErrUnavailable
		}
		if merchant {
			summary.AvailableActions = merchantActions(summary.State)
		} else {
			summary.AvailableActions = userActions(summary.State, now, pickupAt)
		}
		if !validSummary(summary) {
			return Page{}, ErrUnavailable
		}
		orders = append(orders, summary)
	}
	if rows.Err() != nil {
		return Page{}, ErrUnavailable
	}
	page := Page{Orders: orders}
	if len(orders) > int(limit) {
		page.Orders = orders[:limit]
		page.NextAfterID = page.Orders[len(page.Orders)-1].ID
	}
	return page, nil
}

func (repository *Repository) GetUser(ctx context.Context, userID, orderID uint64) (Detail, error) {
	if repository == nil || repository.db == nil || repository.now == nil || userID == 0 || orderID == 0 {
		return Detail{}, ErrInvalidInput
	}
	return repository.get(ctx, "id=? AND user_id=?", []any{orderID, userID}, false, "")
}

func (repository *Repository) GetMerchant(ctx context.Context, userID, orderID uint64) (Detail, error) {
	return repository.GetMerchantAtState(ctx, userID, orderID, "")
}

func (repository *Repository) GetMerchantAtState(ctx context.Context, userID, orderID uint64, state State) (Detail, error) {
	if repository == nil || repository.db == nil || repository.now == nil || userID == 0 || orderID == 0 || (state != "" && !validState(state)) {
		return Detail{}, ErrInvalidInput
	}
	if err := repository.authorizeMerchant(ctx, userID); err != nil {
		return Detail{}, err
	}
	return repository.get(ctx, "id=?", []any{orderID}, true, state)
}

func (repository *Repository) authorizeMerchant(ctx context.Context, userID uint64) error {
	if repository.merchant == nil {
		return ErrUnavailable
	}
	projection, err := repository.merchant.Identity(ctx, userID)
	if err != nil {
		return ErrUnavailable
	}
	if projection.Merchant == nil || (projection.Merchant.Role != merchantidentity.RoleOwner && projection.Merchant.Role != merchantidentity.RoleSubaccount) || projection.Merchant.AuthVersion == 0 {
		return ErrForbidden
	}
	return nil
}

const detailSelect = `SELECT id,order_no,state,DATE_FORMAT(pickup_date,'%Y-%m-%d'),TIME_FORMAT(pickup_time,'%H:%i'),pickup_point_snapshot,pickup_number,payable_cents,materialized_at,pickup_at,contact_name_snapshot,contact_phone_snapshot,identity_kind,discount_rate_percent,transaction_id,paid_at,order_note,item_count,preparing_at,ready_at,completed_at,refunding_at,refunded_at,redemption_token_ciphertext,redemption_token_hash,redemption_key_version,redemption_issued_at,redeemed_by_account_id,redeemed_at FROM orders WHERE `

type orderRecord struct {
	summary                 Summary
	pickupAt                time.Time
	contactName             string
	contactPhone            []byte
	identityKind            string
	discountRate            uint8
	transactionID           []byte
	paidAt                  time.Time
	orderNote               string
	itemCount               uint16
	preparingAt, readyAt    sql.NullTime
	completedAt             sql.NullTime
	refundingAt, refundedAt sql.NullTime
	redemptionCiphertext    []byte
	redemptionHash          []byte
	redemptionVersion       sql.NullInt64
	redemptionIssuedAt      sql.NullTime
	redeemedByAccountID     sql.NullInt64
	redeemedAt              sql.NullTime
}

func (repository *Repository) get(ctx context.Context, where string, args []any, merchant bool, override State) (Detail, error) {
	record, err := scanRecord(repository.db.QueryRowContext(ctx, detailSelect+where+" LIMIT 1", args...))
	if errors.Is(err, sql.ErrNoRows) {
		return Detail{}, ErrNotFound
	}
	if err != nil || !validRecord(record) {
		return Detail{}, ErrUnavailable
	}
	items, err := repository.readItems(ctx, record.summary.ID, record.itemCount, record.summary.PayableCents)
	if err != nil {
		return Detail{}, err
	}
	state := record.summary.State
	if override != "" {
		if !historicalStateAllowed(record, override) {
			return Detail{}, ErrUnavailable
		}
		state = override
	}
	now := repository.now().UTC()
	if now.IsZero() {
		return Detail{}, ErrUnavailable
	}
	record.summary.State = state
	if merchant {
		record.summary.AvailableActions = merchantActions(state)
	} else {
		record.summary.AvailableActions = userActions(state, now, record.pickupAt)
	}
	maskedPhone, ok := maskPhone(string(record.contactPhone))
	if !ok {
		return Detail{}, ErrUnavailable
	}
	detail := Detail{
		Summary: record.summary, Contact: Contact{Name: record.contactName, MaskedPhone: maskedPhone},
		Identity: Identity{Kind: record.identityKind}, Discount: Discount{RatePercent: record.discountRate},
		Items: items, TransactionID: string(record.transactionID), PaidAt: record.paidAt,
		TransitionTimes: transitionTimesFor(record, state), NotificationOptions: notificationOptions(state, merchant), OrderNote: record.orderNote,
	}
	if !merchant && state == StateReadyForPickup {
		detail.RedemptionToken, err = openRedemption(ctx, repository.tokens, uint16(record.redemptionVersion.Int64), record.redemptionCiphertext, record.redemptionHash)
		if err != nil {
			return Detail{}, err
		}
	}
	if !validDetail(detail) {
		return Detail{}, ErrUnavailable
	}
	return detail, nil
}

type rowScanner interface{ Scan(...any) error }

func scanSummary(scanner rowScanner) (Summary, time.Time, error) {
	var summary Summary
	var orderNo []byte
	var state string
	var pickupAt time.Time
	err := scanner.Scan(
		&summary.ID, &orderNo, &state, &summary.PickupDate, &summary.PickupTime, &summary.PickupPoint,
		&summary.PickupNumber, &summary.PayableCents, &summary.MaterializedAt, &pickupAt,
	)
	if err != nil {
		return Summary{}, time.Time{}, err
	}
	if !utf8.Valid(orderNo) {
		return Summary{}, time.Time{}, ErrUnavailable
	}
	summary.OrderNo, summary.State = string(orderNo), State(state)
	if !validSummaryFacts(summary) || pickupAt.IsZero() {
		return Summary{}, time.Time{}, ErrUnavailable
	}
	return summary, pickupAt, nil
}

func scanRecord(scanner rowScanner) (orderRecord, error) {
	var record orderRecord
	var orderNo, state string
	err := scanner.Scan(
		&record.summary.ID, &orderNo, &state, &record.summary.PickupDate, &record.summary.PickupTime,
		&record.summary.PickupPoint, &record.summary.PickupNumber, &record.summary.PayableCents,
		&record.summary.MaterializedAt, &record.pickupAt, &record.contactName, &record.contactPhone,
		&record.identityKind, &record.discountRate, &record.transactionID, &record.paidAt, &record.orderNote,
		&record.itemCount, &record.preparingAt, &record.readyAt, &record.completedAt, &record.refundingAt,
		&record.refundedAt, &record.redemptionCiphertext, &record.redemptionHash, &record.redemptionVersion,
		&record.redemptionIssuedAt, &record.redeemedByAccountID, &record.redeemedAt,
	)
	if err != nil {
		return orderRecord{}, err
	}
	record.summary.OrderNo, record.summary.State = orderNo, State(state)
	return record, nil
}

func (repository *Repository) readItems(ctx context.Context, orderID uint64, itemCount uint16, payableCents uint64) ([]Item, error) {
	rows, err := repository.db.QueryContext(ctx, `
		SELECT line_number,product_id,product_name_snapshot,discounted_unit_price_cents,quantity,payable_subtotal_cents,flavors_json,line_note
		FROM order_items WHERE order_id=? ORDER BY line_number
	`, orderID)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer rows.Close()
	items := make([]Item, 0, itemCount)
	var total uint64
	for rows.Next() {
		var line uint16
		var item Item
		var raw []byte
		if err := rows.Scan(&line, &item.ProductID, &item.Name, &item.UnitPriceCents, &item.Quantity, &item.LineTotalCents, &raw, &item.Note); err != nil {
			return nil, ErrUnavailable
		}
		if line != uint16(len(items)+1) || item.ProductID == 0 || strings.TrimSpace(item.Name) == "" || !utf8.ValidString(item.Name) ||
			item.Quantity == 0 || item.UnitPriceCents > math.MaxUint64/item.Quantity || item.UnitPriceCents*item.Quantity != item.LineTotalCents || !json.Valid(raw) {
			return nil, ErrUnavailable
		}
		if err := json.Unmarshal(raw, &item.Flavors); err != nil || item.Flavors == nil || !validStrings(item.Flavors) || !utf8.ValidString(item.Note) {
			return nil, ErrUnavailable
		}
		if total > math.MaxUint64-item.LineTotalCents {
			return nil, ErrUnavailable
		}
		total += item.LineTotalCents
		items = append(items, item)
	}
	if rows.Err() != nil || len(items) != int(itemCount) || total != payableCents {
		return nil, ErrUnavailable
	}
	return items, nil
}

func validRecord(record orderRecord) bool {
	if !validSummaryFacts(record.summary) || record.pickupAt.IsZero() || strings.TrimSpace(record.contactName) == "" ||
		!utf8.ValidString(record.contactName) || len(record.transactionID) == 0 || len(record.transactionID) > 64 || !utf8.Valid(record.transactionID) ||
		record.paidAt.IsZero() || record.summary.MaterializedAt.Before(record.paidAt) || record.itemCount == 0 ||
		(record.identityKind != "STAFF" && record.identityKind != "VISITOR") || record.discountRate < 1 || record.discountRate > 100 ||
		!validTransitionHistory(record) {
		return false
	}
	hasCipher := len(record.redemptionCiphertext) > 0
	hasHash := len(record.redemptionHash) == sha256.Size
	hasVersion := record.redemptionVersion.Valid && record.redemptionVersion.Int64 > 0 && record.redemptionVersion.Int64 <= math.MaxUint16
	hasIssued := record.redemptionIssuedAt.Valid && !record.redemptionIssuedAt.Time.IsZero()
	switch record.summary.State {
	case StateReserved:
		return !record.preparingAt.Valid && !record.readyAt.Valid && !record.completedAt.Valid && !record.refundingAt.Valid && !record.refundedAt.Valid &&
			!hasCipher && len(record.redemptionHash) == 0 && !record.redemptionVersion.Valid && !record.redemptionIssuedAt.Valid && !record.redeemedByAccountID.Valid && !record.redeemedAt.Valid
	case StatePreparing:
		return record.preparingAt.Valid && !record.readyAt.Valid && !record.completedAt.Valid && !record.refundingAt.Valid && !record.refundedAt.Valid &&
			!hasCipher && len(record.redemptionHash) == 0 && !record.redemptionVersion.Valid && !record.redemptionIssuedAt.Valid && !record.redeemedByAccountID.Valid && !record.redeemedAt.Valid
	case StateReadyForPickup:
		return record.preparingAt.Valid && record.readyAt.Valid && !record.completedAt.Valid && !record.refundingAt.Valid && !record.refundedAt.Valid &&
			hasCipher && hasHash && hasVersion && hasIssued && !record.redeemedByAccountID.Valid && !record.redeemedAt.Valid
	case StateCompleted:
		return record.preparingAt.Valid && record.readyAt.Valid && record.completedAt.Valid && !record.refundingAt.Valid && !record.refundedAt.Valid &&
			!hasCipher && hasHash && !record.redemptionVersion.Valid && hasIssued && record.redeemedByAccountID.Valid && record.redeemedByAccountID.Int64 > 0 && record.redeemedAt.Valid
	case StateRefunding:
		return record.refundingAt.Valid && !record.refundedAt.Valid && !hasCipher && !record.redemptionVersion.Valid && validHistoricRedemption(record, hasHash, hasIssued)
	case StateRefunded:
		return record.refundingAt.Valid && record.refundedAt.Valid && !hasCipher && !record.redemptionVersion.Valid && validHistoricRedemption(record, hasHash, hasIssued)
	default:
		return false
	}
}

func validTransitionHistory(record orderRecord) bool {
	if record.preparingAt.Valid && (record.preparingAt.Time.IsZero() || record.preparingAt.Time.Before(record.summary.MaterializedAt)) {
		return false
	}
	if record.readyAt.Valid && (!record.preparingAt.Valid || record.readyAt.Time.IsZero() || record.readyAt.Time.Before(record.preparingAt.Time)) {
		return false
	}
	if record.redemptionIssuedAt.Valid && (!record.readyAt.Valid || record.redemptionIssuedAt.Time.IsZero() || record.redemptionIssuedAt.Time.Before(record.readyAt.Time)) {
		return false
	}
	if record.redeemedAt.Valid && (!record.redemptionIssuedAt.Valid || record.redeemedAt.Time.IsZero() || record.redeemedAt.Time.Before(record.redemptionIssuedAt.Time)) {
		return false
	}
	if record.completedAt.Valid && (!record.readyAt.Valid || record.completedAt.Time.IsZero() || record.completedAt.Time.Before(record.readyAt.Time)) {
		return false
	}
	if record.refundingAt.Valid && (record.refundingAt.Time.IsZero() || record.refundingAt.Time.Before(record.summary.MaterializedAt)) {
		return false
	}
	return !record.refundedAt.Valid || (record.refundingAt.Valid && !record.refundedAt.Time.IsZero() && !record.refundedAt.Time.Before(record.refundingAt.Time))
}

func validHistoricRedemption(record orderRecord, hasHash, hasIssued bool) bool {
	if hasHash != hasIssued {
		return false
	}
	completed := record.completedAt.Valid
	redeemed := record.redeemedByAccountID.Valid && record.redeemedByAccountID.Int64 > 0 && record.redeemedAt.Valid
	return completed == redeemed
}

func validSummaryFacts(summary Summary) bool {
	return summary.ID > 0 && summary.OrderNo != "" && utf8.ValidString(summary.OrderNo) && validState(summary.State) &&
		strictDate(summary.PickupDate) && strictPickupTime(summary.PickupTime) && strings.TrimSpace(summary.PickupPoint) != "" &&
		utf8.ValidString(summary.PickupPoint) && summary.PickupNumber >= 1 && summary.PickupNumber <= 9999 &&
		summary.PayableCents > 0 && !summary.MaterializedAt.IsZero()
}

func strictPickupTime(value string) bool {
	if len(value) != 5 || value[2] != ':' {
		return false
	}
	parsed, err := time.Parse("15:04", value)
	return err == nil && parsed.Format("15:04") == value
}

func validStrings(values []string) bool {
	for _, value := range values {
		if !utf8.ValidString(value) {
			return false
		}
	}
	return true
}

func historicalStateAllowed(record orderRecord, state State) bool {
	if state == record.summary.State {
		return true
	}
	switch state {
	case StateReadyForPickup:
		return record.readyAt.Valid
	case StateCompleted:
		return record.completedAt.Valid
	default:
		return false
	}
}

func transitionTimesFor(record orderRecord, state State) TransitionTimes {
	result := TransitionTimes{}
	if state != StateReserved && record.preparingAt.Valid {
		result.PreparingAt = record.preparingAt.Time
	}
	if (state == StateReadyForPickup || state == StateCompleted || state == StateRefunding || state == StateRefunded) && record.readyAt.Valid {
		result.ReadyAt = record.readyAt.Time
	}
	if (state == StateCompleted || state == StateRefunding || state == StateRefunded) && record.completedAt.Valid {
		result.CompletedAt = record.completedAt.Time
	}
	if (state == StateRefunding || state == StateRefunded) && record.refundingAt.Valid {
		result.RefundingAt = record.refundingAt.Time
	}
	if state == StateRefunded && record.refundedAt.Valid {
		result.RefundedAt = record.refundedAt.Time
	}
	return result
}

func notificationOptions(state State, merchant bool) []string {
	if merchant {
		return []string{}
	}
	switch state {
	case StateReadyForPickup:
		return []string{"READY"}
	case StateRefunding:
		return []string{"REFUND_RESULT"}
	default:
		return []string{}
	}
}

func userActions(state State, now, pickupAt time.Time) []Action {
	if state == StateReserved && !now.IsZero() && !pickupAt.IsZero() && pickupAt.Sub(now) > 30*time.Minute {
		return []Action{ActionCancel}
	}
	return []Action{}
}

func merchantActions(state State) []Action {
	switch state {
	case StatePreparing:
		return []Action{ActionReady}
	case StateReadyForPickup:
		return []Action{ActionRedeem}
	default:
		return []Action{}
	}
}

func maskPhone(phone string) (string, bool) {
	if len(phone) < 6 || len(phone) > 16 || phone[0] != '+' || phone[1] < '1' || phone[1] > '9' {
		return "", false
	}
	for index := 2; index < len(phone); index++ {
		if phone[index] < '0' || phone[index] > '9' {
			return "", false
		}
	}
	return "+" + strings.Repeat("*", len(phone)-5) + phone[len(phone)-4:], true
}

func openRedemption(ctx context.Context, opener TokenOpener, version uint16, ciphertext, storedHash []byte) (string, error) {
	if opener == nil || version == 0 || len(ciphertext) == 0 || len(storedHash) != sha256.Size {
		return "", ErrUnavailable
	}
	token, err := opener.Open(ctx, version, ciphertext)
	if err != nil || token == "" || len(token) > 256 || !utf8.ValidString(token) || strings.ContainsAny(token, " \t\r\n") {
		return "", ErrUnavailable
	}
	hash := sha256.Sum256([]byte(token))
	if subtle.ConstantTimeCompare(hash[:], storedHash) != 1 {
		return "", ErrUnavailable
	}
	return token, nil
}

func validUserQuery(query UserQuery) bool {
	return query.Limit >= 1 && query.Limit <= 100 && (query.State == "" || validState(query.State)) && !(query.Active && query.State != "")
}

func validMerchantQuery(query MerchantQuery) bool {
	return query.Limit >= 1 && query.Limit <= 100 && (query.State == "" || validState(query.State)) &&
		(query.Date == "" || strictDate(query.Date)) && utf8.ValidString(query.Search) && len(query.Search) <= 64 && strings.TrimSpace(query.Search) == query.Search
}

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, "=", "==")
	value = strings.ReplaceAll(value, "%", "=%")
	return strings.ReplaceAll(value, "_", "=_")
}
