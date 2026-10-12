package main

import (
	"context"
	"embed"
	"encoding/hex"
	"errors"
	"strings"
)

//go:embed sharedpg_bootstrap.sql
var sharedPGBootstrapSource embed.FS

// sharedPGBootstrapSQL accepts only the five native bootstrap inputs. It is
// management SQL, never an ordinary-user RPC or an API startup operation.
func sharedPGBootstrapSQL(input bootstrapInput) (string, error) {
	fields := []struct{ marker, field, value string }{
		{"__ORDER_OWNER_PHONE_HEX__", ownerPhoneEnvironment, input.OwnerPhone},
		{"__ORDER_OWNER_NAME_HEX__", ownerNameEnvironment, input.OwnerName},
		{"__ORDER_STORE_NAME_HEX__", storeNameEnvironment, input.StoreName},
		{"__ORDER_STORE_ADDRESS_HEX__", storeAddressEnvironment, input.StoreAddress},
		{"__ORDER_PICKUP_POINT_HEX__", pickupPointEnvironment, input.PickupPoint},
	}
	raw, err := sharedPGBootstrapSource.ReadFile("sharedpg_bootstrap.sql")
	if err != nil {
		return "", errBootstrapUnavailable
	}
	statement := string(raw)
	for _, field := range fields {
		if !validRequiredText(field.value) || (field.field == ownerPhoneEnvironment && !canonicalPhone(field.value)) {
			return "", inputError{field: field.field}
		}
		if strings.Count(statement, field.marker) != 1 {
			return "", errBootstrapUnavailable
		}
		statement = strings.Replace(statement, field.marker, hex.EncodeToString([]byte(field.value)), 1)
	}
	return statement, nil
}

// The management implementation executes the complete fixed batch, returns
// its typed outcome and sanitised metadata, and rolls back any failed batch.
// Credentials and the management SDK remain outside this business module.
type sharedPGBootstrapManagement interface {
	InstallBootstrap(context.Context, string) (bootstrapOutcome, error)
}

func sharedPGBootstrap(ctx context.Context, manager sharedPGBootstrapManagement, input bootstrapInput) (bootstrapOutcome, error) {
	statement, err := sharedPGBootstrapSQL(input)
	if err != nil {
		return "", err
	}
	if manager == nil {
		return "", errBootstrapUnavailable
	}
	for attempt := 0; attempt < 2; attempt++ {
		outcome, err := manager.InstallBootstrap(ctx, statement)
		if err == nil {
			if outcome != outcomeCreated && outcome != outcomeUnchanged {
				return "", errBootstrapUnavailable
			}
			return outcome, nil
		}
		var pg sharedPGBootstrapFailure
		if errors.As(err, &pg) && pg.SQLState == "P1002" {
			return "", errBootstrapConflict
		}
		// The control-plane error carries the same bounded native retry metadata.
		if attempt == 0 && errors.As(err, &pg) && (pg.SQLState == "40P01" || pg.SQLState == "40001" || pg.SQLState == "55P03" || (pg.SQLState == "23505" && (pg.ConstraintName == "storefront_settings_pkey" || pg.ConstraintName == "discount_settings_pkey" || pg.ConstraintName == "uq_merchant_accounts_phone"))) {
			continue
		}
		return "", errBootstrapUnavailable
	}
	return "", errBootstrapUnavailable
}

// sharedPGBootstrapFailure carries only management-side transaction metadata.
// The provider body, SQL text and customer inputs never enter its message.
type sharedPGBootstrapFailure struct{ SQLState, ConstraintName string }

func (sharedPGBootstrapFailure) Error() string { return "bootstrap management storage unavailable" }
