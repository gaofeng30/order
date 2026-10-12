package orderquery

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/gaofeng30/order/services/api/internal/cloudbasepg"
	"github.com/gaofeng30/order/services/api/internal/merchantidentity"
)

type sharedPGQueryExecutor interface {
	CallRPC(context.Context, string, map[string]any, any) error
}

// SharedPGRepository implements only the existing order read boundary. Every
// call uses a reviewed fixed RPC; domain projections remain in Go.
type SharedPGRepository struct {
	client   sharedPGQueryExecutor
	merchant MerchantIdentity
	tokens   TokenOpener
	now      func() time.Time
}

func NewSharedPGRepository(client sharedPGQueryExecutor, tokens TokenOpener, now func() time.Time) *SharedPGRepository {
	return &SharedPGRepository{client: client, merchant: merchantidentity.NewSharedPGIdentityReader(client), tokens: tokens, now: now}
}

func (r *SharedPGRepository) ListUser(ctx context.Context, userID uint64, q UserQuery) (Page, error) {
	if ctx == nil || r == nil || r.client == nil || r.now == nil || userID == 0 || !validUserQuery(q) {
		return Page{}, ErrInvalidInput
	}
	return r.list(ctx, "order_query_list_user_v1", map[string]any{"p_user": strconv.FormatUint(userID, 10), "p_state": string(q.State), "p_active": q.Active, "p_after": strconv.FormatUint(q.AfterID, 10), "p_limit": q.Limit}, q.Limit, false)
}

func (r *SharedPGRepository) SearchMerchant(ctx context.Context, userID uint64, q MerchantQuery) (Page, error) {
	if ctx == nil || r == nil || r.client == nil || r.now == nil || userID == 0 || !validMerchantQuery(q) {
		return Page{}, ErrInvalidInput
	}
	if err := authorizeMerchant(ctx, r.merchant, userID); err != nil {
		return Page{}, err
	}
	return r.list(ctx, "order_query_search_merchant_v1", map[string]any{"p_user": strconv.FormatUint(userID, 10), "p_state": string(q.State), "p_date": q.Date, "p_search": q.Search, "p_after": strconv.FormatUint(q.AfterID, 10), "p_limit": q.Limit}, q.Limit, true)
}

func (r *SharedPGRepository) list(ctx context.Context, name string, args map[string]any, limit uint16, merchant bool) (Page, error) {
	var facts []sharedPGSummaryFacts
	if err := r.client.CallRPC(ctx, name, args, &facts); err != nil {
		return Page{}, queryRPCError(err)
	}
	now := r.now().UTC()
	if now.IsZero() || facts == nil || len(facts) > int(limit)+1 {
		return Page{}, ErrUnavailable
	}
	orders := make([]Summary, 0, limit)
	for _, fact := range facts {
		summary, pickupAt, err := fact.summary()
		if err != nil {
			return Page{}, ErrUnavailable
		}
		if len(orders) > 0 && summary.ID >= orders[len(orders)-1].ID {
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
	page := Page{Orders: orders}
	if len(orders) > int(limit) {
		page.Orders = orders[:limit]
		page.NextAfterID = page.Orders[len(page.Orders)-1].ID
	}
	return page, nil
}

func (r *SharedPGRepository) GetUser(ctx context.Context, userID, orderID uint64) (Detail, error) {
	if ctx == nil || r == nil || r.client == nil || r.now == nil || userID == 0 || orderID == 0 {
		return Detail{}, ErrInvalidInput
	}
	return r.get(ctx, "order_query_get_user_v1", map[string]any{"p_user": strconv.FormatUint(userID, 10), "p_order": strconv.FormatUint(orderID, 10)}, false, "")
}

func (r *SharedPGRepository) GetMerchant(ctx context.Context, userID, orderID uint64) (Detail, error) {
	return r.GetMerchantAtState(ctx, userID, orderID, "")
}

func (r *SharedPGRepository) GetMerchantAtState(ctx context.Context, userID, orderID uint64, state State) (Detail, error) {
	if ctx == nil || r == nil || r.client == nil || r.now == nil || userID == 0 || orderID == 0 || (state != "" && !validState(state)) {
		return Detail{}, ErrInvalidInput
	}
	if err := authorizeMerchant(ctx, r.merchant, userID); err != nil {
		return Detail{}, err
	}
	return r.get(ctx, "order_query_get_merchant_v1", map[string]any{"p_user": strconv.FormatUint(userID, 10), "p_order": strconv.FormatUint(orderID, 10), "p_state": string(state)}, true, state)
}

func (r *SharedPGRepository) get(ctx context.Context, name string, args map[string]any, merchant bool, override State) (Detail, error) {
	var facts *sharedPGDetailFacts
	if err := r.client.CallRPC(ctx, name, args, &facts); err != nil {
		return Detail{}, queryRPCError(err)
	}
	if facts == nil {
		return Detail{}, ErrNotFound
	}
	record, err := facts.Order.record()
	if err != nil || !validRecord(record) {
		return Detail{}, ErrUnavailable
	}
	records := make([]itemRecord, len(facts.Items))
	for i, item := range facts.Items {
		records[i] = itemRecord{line: item.Line, item: Item{ProductID: item.ProductID, Name: item.Name, Quantity: item.Quantity, UnitPriceCents: item.UnitPriceCents, LineTotalCents: item.LineTotalCents, Note: item.Note}, raw: item.Flavors}
	}
	items, err := projectItems(records, record.itemCount, record.summary.PayableCents)
	if err != nil {
		return Detail{}, err
	}
	return projectOrderDetail(ctx, record, items, merchant, override, r.tokens, r.now)
}

func queryRPCError(err error) error {
	var rpc *cloudbasepg.Error
	if errors.As(err, &rpc) && rpc.SQLState == "P1002" {
		return ErrForbidden
	}
	return ErrUnavailable
}

type sharedPGSummaryFacts struct {
	ID             uint64    `json:"id,string"`
	OrderNoHex     string    `json:"order_no_hex"`
	State          State     `json:"state"`
	PickupDate     string    `json:"pickup_date"`
	PickupTime     string    `json:"pickup_time"`
	PickupPoint    string    `json:"pickup_point"`
	PickupNumber   uint16    `json:"pickup_number,string"`
	PayableCents   uint64    `json:"payable_cents,string"`
	MaterializedAt time.Time `json:"materialized_at"`
	PickupAt       time.Time `json:"pickup_at"`
}

func (f sharedPGSummaryFacts) summary() (Summary, time.Time, error) {
	raw, err := hex.DecodeString(f.OrderNoHex)
	if err != nil || !utf8.Valid(raw) {
		return Summary{}, time.Time{}, ErrUnavailable
	}
	s := Summary{ID: f.ID, OrderNo: string(raw), State: f.State, PickupDate: f.PickupDate, PickupTime: f.PickupTime, PickupPoint: f.PickupPoint, PickupNumber: f.PickupNumber, PayableCents: f.PayableCents, MaterializedAt: f.MaterializedAt}
	if !validSummaryFacts(s) || f.PickupAt.IsZero() {
		return Summary{}, time.Time{}, ErrUnavailable
	}
	return s, f.PickupAt, nil
}

type sharedPGOrderFacts struct {
	sharedPGSummaryFacts
	ContactName         string     `json:"contact_name"`
	ContactPhoneHex     string     `json:"contact_phone_hex"`
	IdentityKind        string     `json:"identity_kind"`
	DiscountRate        uint8      `json:"discount_rate,string"`
	TransactionIDHex    string     `json:"transaction_id_hex"`
	PaidAt              time.Time  `json:"paid_at"`
	OrderNote           string     `json:"order_note"`
	ItemCount           uint16     `json:"item_count,string"`
	PreparingAt         *time.Time `json:"preparing_at"`
	ReadyAt             *time.Time `json:"ready_at"`
	CompletedAt         *time.Time `json:"completed_at"`
	RefundingAt         *time.Time `json:"refunding_at"`
	RefundedAt          *time.Time `json:"refunded_at"`
	RedemptionCipherHex *string    `json:"redemption_cipher_hex"`
	RedemptionHashHex   *string    `json:"redemption_hash_hex"`
	RedemptionVersion   *int64     `json:"redemption_version,string"`
	RedemptionIssuedAt  *time.Time `json:"redemption_issued_at"`
	RedeemedBy          *uint64    `json:"redeemed_by,string"`
	RedeemedAt          *time.Time `json:"redeemed_at"`
}

func (f sharedPGOrderFacts) record() (orderRecord, error) {
	var r orderRecord
	var err error
	r.summary, r.pickupAt, err = f.summary()
	if err != nil {
		return r, err
	}
	r.contactName = f.ContactName
	r.identityKind = f.IdentityKind
	r.discountRate = f.DiscountRate
	r.paidAt = f.PaidAt
	r.orderNote = f.OrderNote
	r.itemCount = f.ItemCount
	if r.contactPhone, err = hex.DecodeString(f.ContactPhoneHex); err != nil {
		return r, err
	}
	if r.transactionID, err = hex.DecodeString(f.TransactionIDHex); err != nil {
		return r, err
	}
	if f.RedemptionCipherHex != nil {
		if r.redemptionCiphertext, err = hex.DecodeString(*f.RedemptionCipherHex); err != nil {
			return r, err
		}
	}
	if f.RedemptionHashHex != nil {
		if r.redemptionHash, err = hex.DecodeString(*f.RedemptionHashHex); err != nil {
			return r, err
		}
	}
	asTime := func(t *time.Time) sql.NullTime {
		if t == nil {
			return sql.NullTime{}
		}
		return sql.NullTime{Time: *t, Valid: true}
	}
	r.preparingAt = asTime(f.PreparingAt)
	r.readyAt = asTime(f.ReadyAt)
	r.completedAt = asTime(f.CompletedAt)
	r.refundingAt = asTime(f.RefundingAt)
	r.refundedAt = asTime(f.RefundedAt)
	r.redemptionIssuedAt = asTime(f.RedemptionIssuedAt)
	r.redeemedAt = asTime(f.RedeemedAt)
	if f.RedemptionVersion != nil {
		r.redemptionVersion = sql.NullInt64{Int64: *f.RedemptionVersion, Valid: true}
	}
	if f.RedeemedBy != nil {
		r.redeemedByAccountID = sql.Null[uint64]{V: *f.RedeemedBy, Valid: true}
	}
	return r, nil
}

type sharedPGItemFacts struct {
	Line           uint16          `json:"line,string"`
	ProductID      uint64          `json:"product_id,string"`
	Name           string          `json:"name"`
	UnitPriceCents uint64          `json:"unit_price_cents,string"`
	Quantity       uint64          `json:"quantity,string"`
	LineTotalCents uint64          `json:"line_total_cents,string"`
	Flavors        json.RawMessage `json:"flavors"`
	Note           string          `json:"note"`
}
type sharedPGDetailFacts struct {
	Order sharedPGOrderFacts  `json:"order"`
	Items []sharedPGItemFacts `json:"items"`
}
