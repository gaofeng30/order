package staffdiscount

import (
	"bytes"
	"context"
	"database/sql"
	"math"
	"strings"
	"unicode"

	"github.com/gaofeng30/order/services/api/internal/quotepricing"
	"github.com/gaofeng30/order/services/api/internal/staffidentity"
	"golang.org/x/text/unicode/norm"
)

type pricingSnapshot struct {
	Unbound            bool
	PrimaryPhone       string
	DefaultContactName string
	Extra              *staffidentity.ExtraClaim
	RatePercent        int64
	WhitelistVersion   uint64
	Entries            []staffidentity.Entry
}

type pricingSnapshotLoader interface {
	Load(context.Context, uint64) (pricingSnapshot, error)
}

// PricingApplication resolves display-only current staff prices. Quote remains
// authoritative and repeats the same current-fact resolution transactionally.
type PricingApplication struct{ snapshots pricingSnapshotLoader }

func NewPostgreSQLPricing(db *sql.DB) *PricingApplication {
	return newPricingApplication(&postgresPricingSnapshotLoader{db: db})
}

func newPricingApplication(loader pricingSnapshotLoader) *PricingApplication {
	return &PricingApplication{snapshots: loader}
}

func (application *PricingApplication) ResolvePrices(ctx context.Context, userID uint64, originals []uint32) ([]*uint32, error) {
	if application == nil || application.snapshots == nil || userID == 0 {
		return nil, ErrUnavailable
	}
	result := make([]*uint32, len(originals))
	if len(originals) == 0 {
		return result, nil
	}
	for _, original := range originals {
		if original == 0 {
			return nil, ErrUnavailable
		}
	}
	snapshot, err := application.snapshots.Load(ctx, userID)
	if err != nil {
		return nil, ErrUnavailable
	}
	if snapshot.Unbound {
		if snapshot.PrimaryPhone != "" || snapshot.Extra != nil || snapshot.RatePercent != 0 || snapshot.WhitelistVersion != 0 || len(snapshot.Entries) != 0 {
			return nil, ErrUnavailable
		}
		return result, nil
	}
	if snapshot.RatePercent < 1 || snapshot.RatePercent > 100 {
		return nil, ErrUnavailable
	}
	resolved, err := staffidentity.Resolve(staffidentity.Input{
		PrimaryPhone: snapshot.PrimaryPhone, Extra: snapshot.Extra, DefaultContactName: snapshot.DefaultContactName,
		WhitelistVersion: snapshot.WhitelistVersion, CandidateEntries: snapshot.Entries,
	})
	if err != nil {
		return nil, ErrUnavailable
	}
	if resolved.Kind == staffidentity.KindVisitor {
		return result, nil
	}
	if resolved.Kind != staffidentity.KindStaff {
		return nil, ErrUnavailable
	}
	lines := make([]quotepricing.Line, 0, len(originals))
	for _, original := range originals {
		lines = append(lines, quotepricing.Line{UnitPriceCents: int64(original), Quantity: 1})
	}
	pricing, err := quotepricing.Calculate(quotepricing.Input{RatePercent: snapshot.RatePercent, Lines: lines})
	if err != nil || len(pricing.Lines) != len(originals) {
		return nil, ErrUnavailable
	}
	for index, line := range pricing.Lines {
		if line.DiscountedUnitPriceCents < 0 || line.DiscountedUnitPriceCents > math.MaxUint32 {
			return nil, ErrUnavailable
		}
		price := uint32(line.DiscountedUnitPriceCents)
		result[index] = &price
	}
	return result, nil
}

type postgresPricingSnapshotLoader struct{ db *sql.DB }

func (loader *postgresPricingSnapshotLoader) Load(ctx context.Context, userID uint64) (result pricingSnapshot, resultErr error) {
	if loader == nil || loader.db == nil || userID == 0 {
		return pricingSnapshot{}, ErrUnavailable
	}
	transaction, err := loader.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return pricingSnapshot{}, ErrUnavailable
	}
	committed := false
	defer func() {
		if !committed {
			_ = transaction.Rollback()
		}
	}()

	var primaryPhone, extraPhone, extraName sql.NullString
	var extraNameKey []byte
	err = transaction.QueryRowContext(ctx, `SELECT
  convert_from(primary_phone,'UTF8'), convert_from(extra_phone,'UTF8'), extra_name, extra_name_key
FROM miniprogram_users
WHERE id = $1`, userID).Scan(&primaryPhone, &extraPhone, &extraName, &extraNameKey)
	if err != nil {
		return pricingSnapshot{}, ErrUnavailable
	}
	if extraName.Valid != (extraNameKey != nil) {
		return pricingSnapshot{}, ErrUnavailable
	}
	if extraName.Valid {
		if !bytes.Equal(extraNameKey, canonicalPricingNameKey(extraName.String)) {
			return pricingSnapshot{}, ErrUnavailable
		}
		result.DefaultContactName = extraName.String
	}
	if !primaryPhone.Valid {
		if extraPhone.Valid {
			return pricingSnapshot{}, ErrUnavailable
		}
		if err := transaction.Commit(); err != nil {
			return pricingSnapshot{}, ErrUnavailable
		}
		committed = true
		return pricingSnapshot{Unbound: true}, nil
	}
	result.PrimaryPhone = primaryPhone.String
	if extraPhone.Valid {
		result.Extra = &staffidentity.ExtraClaim{Phone: extraPhone.String}
	}
	if err := transaction.QueryRowContext(ctx, `SELECT rate_percent, whitelist_version
FROM discount_settings
WHERE id = 1`).Scan(&result.RatePercent, &result.WhitelistVersion); err != nil || result.RatePercent < 1 || result.RatePercent > 100 || result.WhitelistVersion == 0 {
		return pricingSnapshot{}, ErrUnavailable
	}

	phones := []any{[]byte(result.PrimaryPhone)}
	query := `SELECT convert_from(phone,'UTF8'), name, name_key, enabled
FROM staff_whitelist
WHERE phone = $1`
	if result.Extra != nil && result.Extra.Phone != result.PrimaryPhone {
		query += ` OR phone = $2`
		phones = append(phones, []byte(result.Extra.Phone))
	}
	query += ` ORDER BY id`
	rows, err := transaction.QueryContext(ctx, query, phones...)
	if err != nil {
		return pricingSnapshot{}, ErrUnavailable
	}
	for rows.Next() {
		var entry staffidentity.Entry
		var nameKey []byte
		if err := rows.Scan(&entry.Phone, &entry.Name, &nameKey, &entry.Enabled); err != nil || !bytes.Equal(nameKey, canonicalPricingNameKey(entry.Name)) {
			rows.Close()
			return pricingSnapshot{}, ErrUnavailable
		}
		result.Entries = append(result.Entries, entry)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return pricingSnapshot{}, ErrUnavailable
	}
	if err := rows.Close(); err != nil {
		return pricingSnapshot{}, ErrUnavailable
	}
	if err := transaction.Commit(); err != nil {
		return pricingSnapshot{}, ErrUnavailable
	}
	committed = true
	return result, nil
}

func canonicalPricingNameKey(value string) []byte {
	value = strings.TrimSpace(norm.NFKC.String(value))
	key := strings.Map(func(character rune) rune {
		if unicode.IsSpace(character) {
			return -1
		}
		return character
	}, value)
	if key == "" || len([]byte(key)) > 400 {
		return nil
	}
	return []byte(key)
}
