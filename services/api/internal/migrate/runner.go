package migrate

// Result is the sanitized migration outcome.
type Result struct {
	FromVersion  uint64
	ToVersion    uint64
	AppliedCount int
}

type migrateError struct {
	reason  string
	version uint64
}

func (err migrateError) Error() string { return err.reason }

// Reason returns a stable non-sensitive migration failure category.
func Reason(err error) string {
	if value, ok := err.(migrateError); ok {
		return value.reason
	}
	return ReasonMigrationFailed
}

// Version returns the migration version associated with a failure, if any.
func Version(err error) uint64 {
	if value, ok := err.(migrateError); ok {
		return value.version
	}
	return 0
}
