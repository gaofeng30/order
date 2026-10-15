// Package postgresql holds the immutable runtime PostgreSQL migration chain.
package postgresql

import (
	"embed"
	"io/fs"
)

//go:embed *.sql
var embedded embed.FS

// FS is the forward-only PostgreSQL migration set, with checksums of PG SQL.
var FS fs.FS = embedded
