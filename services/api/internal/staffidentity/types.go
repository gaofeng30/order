package staffidentity

// Input is a complete candidate whitelist view for one primary and optional extra claim.
type Input struct {
	PrimaryPhone     string
	Extra            *ExtraClaim
	WhitelistVersion uint64
	CandidateEntries []Entry
}

// ExtraClaim is the single optional phone/name claim supplied by the caller.
type ExtraClaim struct {
	Phone string
	Name  string
}

// Entry is one candidate whitelist record supplied by the caller.
type Entry struct {
	Phone   string
	Name    string
	Enabled bool
}

// Kind is the resolved identity category.
type Kind string

const (
	KindVisitor Kind = "VISITOR"
	KindStaff   Kind = "STAFF"
)

// Snapshot is the resolved identity and source whitelist version.
type Snapshot struct {
	Kind             Kind
	WhitelistVersion uint64
}
