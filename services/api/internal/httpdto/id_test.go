package httpdto

import "testing"

func TestParseIDAcceptsOnlyCanonicalPositiveDecimal(t *testing.T) {
	if got, ok := ParseID("18446744073709551615"); !ok || got != ^uint64(0) {
		t.Fatalf("ParseID(max) = %d, %v", got, ok)
	}
	for _, input := range []string{"", "0", "00", "01", "+1", "-1", " 1", "1 ", "18446744073709551616"} {
		if got, ok := ParseID(input); ok || got != 0 {
			t.Fatalf("ParseID(%q) = %d, %v", input, got, ok)
		}
	}
}
