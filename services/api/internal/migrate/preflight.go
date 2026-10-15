package migrate

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

type legacyCatalogName struct {
	id   uint64
	name string
}

func validateLegacyCatalogNames(rows []legacyCatalogName) error {
	seen := make(map[string]uint64, len(rows))
	for _, row := range rows {
		if !utf8.ValidString(row.name) {
			return fmt.Errorf("invalid legacy name at id %d", row.id)
		}
		canonical := strings.TrimFunc(norm.NFKC.String(row.name), unicode.IsSpace)
		if canonical != row.name || len(canonical) == 0 || len([]byte(canonical)) > 400 {
			return fmt.Errorf("unsafe legacy name at id %d", row.id)
		}
		if previous, exists := seen[canonical]; exists {
			return fmt.Errorf("duplicate legacy name at ids %d and %d", previous, row.id)
		}
		seen[canonical] = row.id
	}
	return nil
}
