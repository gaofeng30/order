package config

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	redemptionTokenSecretName = "order-production-redemption-token-key"
	maxRedemptionSecretSize   = 256
)

type RedemptionTokenMaterial struct {
	version uint16
	key     [32]byte
}

func (material RedemptionTokenMaterial) Version() uint16 { return material.version }

func (material RedemptionTokenMaterial) Key() []byte {
	return append([]byte(nil), material.key[:]...)
}

type redemptionTokenSecret struct {
	Version   string `json:"version"`
	KeyBase64 string `json:"key_base64"`
}

func LoadRedemptionTokenMaterial(ctx context.Context, source SecretSource) (RedemptionTokenMaterial, error) {
	if ctx == nil || source == nil {
		return RedemptionTokenMaterial{}, loadError{reason: "production_redemption_secret_unavailable"}
	}
	loadContext, cancel := context.WithTimeout(ctx, secretLoadTimeout)
	defer cancel()
	encoded, err := source.Get(loadContext, redemptionTokenSecretName)
	if err != nil {
		return RedemptionTokenMaterial{}, loadError{reason: "production_redemption_secret_unavailable"}
	}
	if encoded == "" || len(encoded) > maxRedemptionSecretSize || !utf8.ValidString(encoded) {
		return RedemptionTokenMaterial{}, loadError{reason: "production_redemption_secret_invalid"}
	}
	decoder := json.NewDecoder(strings.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var secret redemptionTokenSecret
	if decoder.Decode(&secret) != nil || redemptionJSONEnd(decoder) != nil {
		return RedemptionTokenMaterial{}, loadError{reason: "production_redemption_secret_invalid"}
	}
	version, err := strconv.ParseUint(secret.Version, 10, 16)
	decoded, decodeErr := base64.StdEncoding.DecodeString(secret.KeyBase64)
	if err != nil || version == 0 || decodeErr != nil || len(decoded) != 32 || base64.StdEncoding.EncodeToString(decoded) != secret.KeyBase64 {
		return RedemptionTokenMaterial{}, loadError{reason: "production_redemption_secret_invalid"}
	}
	var key [32]byte
	copy(key[:], decoded)
	clear(decoded)
	return RedemptionTokenMaterial{version: uint16(version), key: key}, nil
}

func LoadProductionRedemptionTokenMaterial(ctx context.Context, region string) (RedemptionTokenMaterial, error) {
	if !validTencentRegion(region) {
		return RedemptionTokenMaterial{}, loadError{reason: "production_redemption_secret_unavailable"}
	}
	return LoadRedemptionTokenMaterial(ctx, newProductionSecretSource(region))
}

func redemptionJSONEnd(decoder *json.Decoder) error {
	var trailing any
	err := decoder.Decode(&trailing)
	if err == nil {
		return loadError{reason: "production_redemption_secret_invalid"}
	}
	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
}
