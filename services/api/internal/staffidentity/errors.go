package staffidentity

// ErrorKind is a stable, non-sensitive identity resolution failure category.
type ErrorKind string

const (
	ErrorInvalidPrimaryPhone      ErrorKind = "INVALID_PRIMARY_PHONE"
	ErrorInvalidExtraClaim        ErrorKind = "INVALID_EXTRA_CLAIM"
	ErrorInvalidWhitelistSnapshot ErrorKind = "INVALID_WHITELIST_SNAPSHOT"
)

// Error exposes only a stable identity resolution failure category.
type Error struct {
	kind ErrorKind
}

func (err *Error) Error() string { return "staffidentity: " + string(err.kind) }

// Kind returns the stable failure category.
func (err *Error) Kind() ErrorKind { return err.kind }

func newError(kind ErrorKind) error { return &Error{kind: kind} }
