package wechatpay

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
)

// NewMerchantNumber generates a merchant number for a cloud payment or refund.
func NewMerchantNumber(environment, kind string) (string, error) {
	return newMerchantNumber(environment, kind, rand.Reader)
}

func newMerchantNumber(environment, kind string, reader io.Reader) (string, error) {
	var prefix string
	switch {
	case environment == "uat" && kind == "payment":
		prefix = "OU1P"
	case environment == "uat" && kind == "refund":
		prefix = "OU1R"
	case environment == "production" && kind == "payment":
		prefix = "OP1P"
	case environment == "production" && kind == "refund":
		prefix = "OP1R"
	default:
		return "", fmt.Errorf("wechatpay: invalid merchant number environment or kind")
	}
	var random [14]byte
	if _, err := io.ReadFull(reader, random[:]); err != nil {
		return "", fmt.Errorf("wechatpay: generate merchant number: %w", err)
	}
	return prefix + hex.EncodeToString(random[:]), nil
}

// ParseMerchantNumber reports protocol evidence only; an unmatched number may be legacy.
func ParseMerchantNumber(value string) (environment, kind string, ok bool) {
	if len(value) != 32 {
		return "", "", false
	}
	for index := 4; index < len(value); index++ {
		character := value[index]
		if !(character >= '0' && character <= '9') && !(character >= 'a' && character <= 'f') {
			return "", "", false
		}
	}
	switch value[:4] {
	case "OU1P":
		return "uat", "payment", true
	case "OU1R":
		return "uat", "refund", true
	case "OP1P":
		return "production", "payment", true
	case "OP1R":
		return "production", "refund", true
	default:
		return "", "", false
	}
}
