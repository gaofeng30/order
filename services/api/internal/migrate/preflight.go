package migrate

import (
	"context"
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

func (connection *mysqlRunnerConnection) preflight(ctx context.Context, migration Migration) error {
	var table string
	switch migration.Version {
	case 20:
		table = "categories"
	case 23:
		table = "products"
	default:
		return nil
	}

	rows, err := connection.conn.QueryContext(ctx, "SELECT id,name FROM "+table+" ORDER BY id")
	if err != nil {
		return err
	}
	defer rows.Close()

	names := make([]legacyCatalogName, 0)
	for rows.Next() {
		var row legacyCatalogName
		if err := rows.Scan(&row.id, &row.name); err != nil {
			return err
		}
		names = append(names, row)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return validateLegacyCatalogNames(names)
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
