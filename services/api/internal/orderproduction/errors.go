package orderproduction

// ErrorKind is a stable, non-sensitive policy failure category.
type ErrorKind string

const (
	ErrorInvalidTime  ErrorKind = "INVALID_TIME"
	ErrorInvalidState ErrorKind = "INVALID_STATE"
)

// Error exposes only a stable policy failure category.
type Error struct {
	kind ErrorKind
}

func (err *Error) Error() string { return "orderproduction: " + string(err.kind) }

// Kind returns the stable failure category.
func (err *Error) Kind() ErrorKind { return err.kind }
