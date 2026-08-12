package migrations

import (
	"embed"
	"io/fs"
)

//go:embed *.sql
var embedded embed.FS

// FS contains the immutable forward-only migration set.
var FS fs.FS = embedded
