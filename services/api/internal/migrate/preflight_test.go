package migrate

import "testing"

func TestValidateLegacyCatalogNamesAcceptsFrozenCanonicalBytes(t *testing.T) {
	rows := []legacyCatalogName{{id: 1, name: "午餐"}, {id: 2, name: "Chef Special"}, {id: 3, name: "A餐"}}
	if err := validateLegacyCatalogNames(rows); err != nil {
		t.Fatalf("validateLegacyCatalogNames() error = %v", err)
	}
}

func TestValidateLegacyCatalogNamesRejectsUnsafeBackfill(t *testing.T) {
	tooLong := make([]byte, 401)
	for index := range tooLong {
		tooLong[index] = 'a'
	}
	for _, test := range []struct {
		name string
		rows []legacyCatalogName
	}{
		{name: "invalid utf8", rows: []legacyCatalogName{{id: 1, name: string([]byte{0xff})}}},
		{name: "normalization changes bytes", rows: []legacyCatalogName{{id: 1, name: "A\u030a"}}},
		{name: "unicode trim changes bytes", rows: []legacyCatalogName{{id: 1, name: "\u3000午餐"}}},
		{name: "empty", rows: []legacyCatalogName{{id: 1, name: ""}}},
		{name: "over 400 bytes", rows: []legacyCatalogName{{id: 1, name: string(tooLong)}}},
		{name: "byte exact duplicate", rows: []legacyCatalogName{{id: 1, name: "午餐"}, {id: 2, name: "午餐"}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := validateLegacyCatalogNames(test.rows); err == nil {
				t.Fatal("validateLegacyCatalogNames() error = nil, want rejection")
			}
		})
	}
}
