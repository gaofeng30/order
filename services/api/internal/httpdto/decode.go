package httpdto

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"unicode/utf8"
)

var (
	ErrInvalidJSON  = errors.New("invalid_json")
	ErrBodyTooLarge = errors.New("body_too_large")
)

// ID is the public JSON representation for a positive MySQL identifier.
type ID uint64

// ParseID accepts only the canonical positive decimal representation used by public paths.
func ParseID(encoded string) (uint64, bool) {
	if encoded == "" || encoded[0] == '0' {
		return 0, false
	}
	value, err := strconv.ParseUint(encoded, 10, 64)
	if err != nil || value == 0 || strconv.FormatUint(value, 10) != encoded {
		return 0, false
	}
	return value, true
}

func (id ID) MarshalJSON() ([]byte, error) {
	if id == 0 {
		return nil, ErrInvalidJSON
	}
	return json.Marshal(strconv.FormatUint(uint64(id), 10))
}

func (id *ID) UnmarshalJSON(data []byte) error {
	var encoded string
	if err := json.Unmarshal(data, &encoded); err != nil || encoded == "" || encoded[0] == '0' {
		return ErrInvalidJSON
	}
	value, err := strconv.ParseUint(encoded, 10, 64)
	if err != nil || value == 0 || strconv.FormatUint(value, 10) != encoded {
		return ErrInvalidJSON
	}
	*id = ID(value)
	return nil
}

// DecodeStrict reads exactly one bounded JSON document, rejecting duplicate or unknown fields.
func DecodeStrict(reader io.Reader, limit int64, target any) error {
	if limit <= 0 {
		return ErrBodyTooLarge
	}
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return ErrInvalidJSON
	}
	if int64(len(data)) > limit {
		return ErrBodyTooLarge
	}
	if len(data) == 0 || !utf8.Valid(data) || !hasUniqueObjectKeys(data) {
		return ErrInvalidJSON
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return ErrInvalidJSON
	}
	if _, err := decoder.Token(); err != io.EOF {
		return ErrInvalidJSON
	}
	return nil
}

func hasUniqueObjectKeys(data []byte) bool {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := consumeValue(decoder); err != nil {
		return false
	}
	_, err := decoder.Token()
	return err == io.EOF
}

func consumeValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, isDelimiter := token.(json.Delim)
	if !isDelimiter {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return ErrInvalidJSON
			}
			if _, exists := seen[key]; exists {
				return ErrInvalidJSON
			}
			seen[key] = struct{}{}
			if err := consumeValue(decoder); err != nil {
				return err
			}
		}
	case '[':
		for decoder.More() {
			if err := consumeValue(decoder); err != nil {
				return err
			}
		}
	default:
		return ErrInvalidJSON
	}
	expectedEnd := json.Delim('}')
	if delimiter == '[' {
		expectedEnd = ']'
	}
	end, err := decoder.Token()
	if err != nil || end != expectedEnd {
		return ErrInvalidJSON
	}
	return nil
}
