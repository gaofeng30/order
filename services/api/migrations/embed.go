package migrations

import (
	"embed"
	"io/fs"
)

//go:embed *.sql
var embedded embed.FS

// FS preserves the immutable historical MySQL migration set. Runtime commands use postgresql.FS.
var FS fs.FS = embedded
