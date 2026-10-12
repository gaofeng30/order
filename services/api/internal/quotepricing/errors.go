package quotepricing

// ErrorKind is a stable, non-sensitive calculation failure category.
type ErrorKind string

const (
	ErrorEmptyLines      ErrorKind = "EMPTY_LINES"
	ErrorInvalidRate     ErrorKind = "INVALID_RATE"
	ErrorInvalidPrice    ErrorKind = "INVALID_PRICE"
	ErrorInvalidQuantity ErrorKind = "INVALID_QUANTITY"
	ErrorOverflow        ErrorKind = "OVERFLOW"
)

// Error exposes only a stable calculation failure category.
type Error struct {
	kind ErrorKind
}

func (err *Error) Error() string { return "quotepricing: " + string(err.kind) }

// Kind returns the stable failure category.
func (err *Error) Kind() ErrorKind { return err.kind }

func newError(kind ErrorKind) error { return &Error{kind: kind} }
