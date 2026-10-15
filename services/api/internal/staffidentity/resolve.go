package staffidentity

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// Resolve resolves a versioned identity snapshot from a complete candidate view.
func Resolve(input Input) (Snapshot, error) {
	if !isCanonicalPhone(input.PrimaryPhone) {
		return Snapshot{}, newError(ErrorInvalidPrimaryPhone)
	}
	if input.Extra != nil && !isCanonicalPhone(input.Extra.Phone) {
		return Snapshot{}, newError(ErrorInvalidExtraClaim)
	}
	if !utf8.ValidString(input.DefaultContactName) {
		return Snapshot{}, newError(ErrorInvalidDefaultName)
	}
	if input.WhitelistVersion == 0 {
		return Snapshot{}, newError(ErrorInvalidWhitelistSnapshot)
	}
	seenPhones := make(map[string]struct{}, len(input.CandidateEntries))
	for _, entry := range input.CandidateEntries {
		if !isCanonicalPhone(entry.Phone) {
			return Snapshot{}, newError(ErrorInvalidWhitelistSnapshot)
		}
		if !utf8.ValidString(entry.Name) {
			return Snapshot{}, newError(ErrorInvalidWhitelistSnapshot)
		}
		if normalizeName(entry.Name) == "" {
			return Snapshot{}, newError(ErrorInvalidWhitelistSnapshot)
		}
		if entry.Phone != input.PrimaryPhone && (input.Extra == nil || entry.Phone != input.Extra.Phone) {
			return Snapshot{}, newError(ErrorInvalidWhitelistSnapshot)
		}
		if _, duplicate := seenPhones[entry.Phone]; duplicate {
			return Snapshot{}, newError(ErrorInvalidWhitelistSnapshot)
		}
		seenPhones[entry.Phone] = struct{}{}
	}

	name := normalizeName(input.DefaultContactName)
	for _, entry := range input.CandidateEntries {
		if name != "" && entry.Enabled && normalizeName(entry.Name) == name &&
			(entry.Phone == input.PrimaryPhone || (input.Extra != nil && entry.Phone == input.Extra.Phone)) {
			return Snapshot{Kind: KindStaff, WhitelistVersion: input.WhitelistVersion}, nil
		}
	}
	return Snapshot{Kind: KindVisitor, WhitelistVersion: input.WhitelistVersion}, nil
}

func isCanonicalPhone(phone string) bool {
	if len(phone) < 2 || len(phone) > 16 || phone[0] != '+' || phone[1] < '1' || phone[1] > '9' {
		return false
	}
	for index := 2; index < len(phone); index++ {
		if phone[index] < '0' || phone[index] > '9' {
			return false
		}
	}
	return true
}

func normalizeName(name string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, norm.NFKC.String(name))
}
