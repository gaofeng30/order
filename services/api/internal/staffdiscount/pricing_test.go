package staffdiscount

import (
	"context"
	"testing"

	"github.com/gaofeng30/order/services/api/internal/staffidentity"
)

type pricingSnapshotLoaderStub struct {
	snapshot pricingSnapshot
	err      error
}

func (stub pricingSnapshotLoaderStub) Load(context.Context, uint64) (pricingSnapshot, error) {
	return stub.snapshot, stub.err
}

func TestPublicPricingUsesPrimaryAndExtraNameIdentityRules(t *testing.T) {
	staffPrice := uint32(1530)
	for _, test := range []struct {
		name     string
		snapshot pricingSnapshot
		want     []*uint32
	}{
		{
			name: "primary enabled",
			snapshot: pricingSnapshot{PrimaryPhone: "+8613712345678", RatePercent: 85, WhitelistVersion: 7,
				Entries: []staffidentity.Entry{{Phone: "+8613712345678", Name: "张三", Enabled: true}}},
			want: []*uint32{&staffPrice},
		},
		{
			name: "extra phone and normalized name",
			snapshot: pricingSnapshot{PrimaryPhone: "+8613712345678", Extra: &staffidentity.ExtraClaim{Phone: "+8613912345678", Name: "李 四"}, RatePercent: 85, WhitelistVersion: 8,
				Entries: []staffidentity.Entry{{Phone: "+8613912345678", Name: "李四", Enabled: true}}},
			want: []*uint32{&staffPrice},
		},
		{
			name: "extra name mismatch stays visitor",
			snapshot: pricingSnapshot{PrimaryPhone: "+8613712345678", Extra: &staffidentity.ExtraClaim{Phone: "+8613912345678", Name: "王五"}, RatePercent: 85, WhitelistVersion: 9,
				Entries: []staffidentity.Entry{{Phone: "+8613912345678", Name: "李四", Enabled: true}}},
			want: []*uint32{nil},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			application := newPricingApplication(pricingSnapshotLoaderStub{snapshot: test.snapshot})
			got, err := application.ResolvePrices(context.Background(), 42, []uint32{1800})
			if err != nil || len(got) != 1 {
				t.Fatalf("ResolvePrices() = %#v, %v", got, err)
			}
			if test.want[0] == nil {
				if got[0] != nil {
					t.Fatalf("visitor price = %v, want absent", *got[0])
				}
			} else if got[0] == nil || *got[0] != *test.want[0] {
				t.Fatalf("staff price = %v, want %d", got[0], *test.want[0])
			}
		})
	}
}

func TestPublicPricingAllowsAuthenticatedUnboundVisitor(t *testing.T) {
	application := newPricingApplication(pricingSnapshotLoaderStub{snapshot: pricingSnapshot{Unbound: true}})
	got, err := application.ResolvePrices(context.Background(), 42, []uint32{1800, 200})
	if err != nil || len(got) != 2 || got[0] != nil || got[1] != nil {
		t.Fatalf("unbound ResolvePrices() = %#v, %v", got, err)
	}
}
