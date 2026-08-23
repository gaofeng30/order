package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"

	"github.com/gaofeng30/order/services/api/internal/config"
	"github.com/gaofeng30/order/services/api/internal/fulfillment"
)

func composeRedemptionTokenCipher(environment config.Environment, region string) (*fulfillment.AESGCMTokenCipher, error) {
	if environment != config.Production {
		key := sha256.Sum256([]byte("order-local-redemption-token-key-v1"))
		return fulfillment.NewAESGCMTokenCipher(map[uint16][]byte{1: key[:]}, 1, rand.Reader)
	}
	material, err := config.LoadProductionRedemptionTokenMaterial(context.Background(), region)
	if err != nil {
		return nil, err
	}
	key := material.Key()
	defer clear(key)
	return fulfillment.NewAESGCMTokenCipher(map[uint16][]byte{material.Version(): key}, material.Version(), rand.Reader)
}
