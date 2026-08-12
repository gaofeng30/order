package migrate

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

var migrationNamePattern = regexp.MustCompile(`^([0-9]{6})_([a-z0-9_]+)\.sql$`)

// Migration is one validated, immutable, single-statement migration.
type Migration struct {
	Version  uint64
	Name     string
	SQL      []byte
	Checksum [sha256.Size]byte
}

// Load validates and loads a root-level forward-only migration set.
func Load(source fs.FS) ([]Migration, error) {
	entries, err := fs.ReadDir(source, ".")
	if err != nil {
		return nil, fmt.Errorf("migration_set_invalid")
	}
	migrations := make([]Migration, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			return nil, fmt.Errorf("migration_set_invalid")
		}
		matches := migrationNamePattern.FindStringSubmatch(entry.Name())
		if matches == nil || strings.Contains(matches[2], "down") || strings.Contains(matches[2], "seed") {
			return nil, fmt.Errorf("migration_set_invalid")
		}
		version, _ := strconv.ParseUint(matches[1], 10, 64)
		data, err := fs.ReadFile(source, entry.Name())
		if err != nil || !validMigrationBytes(data) {
			return nil, fmt.Errorf("migration_set_invalid")
		}
		migrations = append(migrations, Migration{Version: version, Name: entry.Name(), SQL: append([]byte(nil), data...), Checksum: sha256.Sum256(data)})
	}
	sort.Slice(migrations, func(i, j int) bool { return migrations[i].Version < migrations[j].Version })
	if len(migrations) == 0 {
		return nil, fmt.Errorf("migration_set_invalid")
	}
	for index, migration := range migrations {
		if migration.Version != uint64(index+1) {
			return nil, fmt.Errorf("migration_set_invalid")
		}
	}
	return migrations, nil
}

func validMigrationBytes(data []byte) bool {
	if len(data) == 0 || !utf8.Valid(data) || strings.HasPrefix(string(data), "\ufeff") || !strings.HasSuffix(string(data), "\n") || strings.Contains(string(data), "\r") {
		return false
	}
	trimmed := strings.TrimSpace(string(data))
	if !strings.HasSuffix(trimmed, ";") || strings.Count(trimmed, ";") != 1 {
		return false
	}
	upper := strings.ToUpper(trimmed)
	for _, forbidden := range []string{"DELIMITER", "SOURCE ", "LOAD DATA"} {
		if strings.Contains(upper, forbidden) {
			return false
		}
	}
	return true
}
