package staffidentity_test

import "github.com/gaofeng30/order/services/api/internal/staffidentity"

var _ func(staffidentity.Input) (staffidentity.Snapshot, error) = staffidentity.Resolve

var (
	_ = staffidentity.Input{
		PrimaryPhone:     "+1234567890",
		Extra:            &staffidentity.ExtraClaim{Phone: "+1987654321", Name: "Test Name"},
		WhitelistVersion: 1,
		CandidateEntries: []staffidentity.Entry{{Phone: "+1234567890", Name: "Test Name", Enabled: true}},
	}
	_ staffidentity.Kind     = staffidentity.KindVisitor
	_ staffidentity.Snapshot = staffidentity.Snapshot{Kind: staffidentity.KindStaff, WhitelistVersion: 1}
	_ staffidentity.ErrorKind
	_ interface {
		Kind() staffidentity.ErrorKind
	} = (*staffidentity.Error)(nil)
)
