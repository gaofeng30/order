package staffdiscount

import (
	"bytes"
	"context"
	"encoding/hex"
	"strconv"

	"github.com/gaofeng30/order/services/api/internal/staffidentity"
)

type sharedPGExecutor interface {
	CallRPC(context.Context, string, map[string]any, any) error
}
type sharedPGPricingLoader struct{ client sharedPGExecutor }

func NewSharedPGPricing(client sharedPGExecutor) *PricingApplication {
	return newPricingApplication(&sharedPGPricingLoader{client})
}
func (l *sharedPGPricingLoader) Load(ctx context.Context, user uint64) (pricingSnapshot, error) {
	if l == nil || l.client == nil || ctx == nil || user == 0 {
		return pricingSnapshot{}, ErrUnavailable
	}
	var f struct {
		Outcome string `json:"outcome"`
		User    struct {
			Primary    *string `json:"primary_phone"`
			ExtraPhone *string `json:"extra_phone"`
			ExtraName  *string `json:"extra_name"`
			ExtraKey   *string `json:"extra_name_key_hex"`
		} `json:"user"`
		Settings *struct {
			Rate    int64  `json:"rate_percent"`
			Version string `json:"whitelist_version"`
		} `json:"settings"`
		Entries []struct {
			Phone   string `json:"phone"`
			Name    string `json:"name"`
			Key     string `json:"name_key_hex"`
			Enabled bool   `json:"enabled"`
		} `json:"entries"`
	}
	if l.client.CallRPC(ctx, "order_display_pricing_facts_v1", map[string]any{"p_user": strconv.FormatUint(user, 10)}, &f) != nil || f.Outcome != "SUCCEEDED" || f.Entries == nil {
		return pricingSnapshot{}, ErrUnavailable
	}
	if (f.User.ExtraName != nil) != (f.User.ExtraKey != nil) {
		return pricingSnapshot{}, ErrUnavailable
	}
	name := ""
	if f.User.ExtraName != nil {
		key, err := hex.DecodeString(*f.User.ExtraKey)
		if err != nil || !bytes.Equal(key, canonicalPricingNameKey(*f.User.ExtraName)) {
			return pricingSnapshot{}, ErrUnavailable
		}
		name = *f.User.ExtraName
	}
	if f.User.Primary == nil {
		if f.User.ExtraPhone != nil || f.Settings != nil || len(f.Entries) != 0 {
			return pricingSnapshot{}, ErrUnavailable
		}
		return pricingSnapshot{Unbound: true}, nil
	}
	if f.Settings == nil {
		return pricingSnapshot{}, ErrUnavailable
	}
	version, err := strconv.ParseUint(f.Settings.Version, 10, 64)
	if err != nil || version == 0 || strconv.FormatUint(version, 10) != f.Settings.Version || f.Settings.Rate < 1 || f.Settings.Rate > 100 {
		return pricingSnapshot{}, ErrUnavailable
	}
	result := pricingSnapshot{DefaultContactName: name, PrimaryPhone: *f.User.Primary, RatePercent: f.Settings.Rate, WhitelistVersion: version}
	if f.User.ExtraPhone != nil {
		result.Extra = &staffidentity.ExtraClaim{Phone: *f.User.ExtraPhone}
	}
	for _, entry := range f.Entries {
		key, err := hex.DecodeString(entry.Key)
		if err != nil || !bytes.Equal(key, canonicalPricingNameKey(entry.Name)) {
			return pricingSnapshot{}, ErrUnavailable
		}
		result.Entries = append(result.Entries, staffidentity.Entry{Phone: entry.Phone, Name: entry.Name, Enabled: entry.Enabled})
	}
	return result, nil
}
