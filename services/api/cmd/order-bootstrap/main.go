package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/gaofeng30/order/services/api/internal/config"
	"github.com/gaofeng30/order/services/api/internal/database"
	"github.com/gaofeng30/order/services/api/internal/migrate"
	"github.com/gaofeng30/order/services/api/migrations"
)

const (
	maxBootstrapTextBytes   = 65535
	ownerPhoneEnvironment   = "ORDER_BOOTSTRAP_OWNER_PHONE"
	ownerNameEnvironment    = "ORDER_BOOTSTRAP_OWNER_NAME"
	storeNameEnvironment    = "ORDER_BOOTSTRAP_STORE_NAME"
	storeAddressEnvironment = "ORDER_BOOTSTRAP_STORE_ADDRESS"
	pickupPointEnvironment  = "ORDER_BOOTSTRAP_PICKUP_POINT"
)

const (
	reasonBootstrapInputInvalid = "bootstrap_input_invalid"
	reasonBootstrapConflict     = "bootstrap_conflict"
	reasonBootstrapFailed       = "bootstrap_failed"
)

type bootstrapOutcome string

const (
	outcomeCreated   bootstrapOutcome = "created"
	outcomeUnchanged bootstrapOutcome = "unchanged"
)

type commandFunc func(context.Context) (bootstrapOutcome, error)

type commandError struct {
	reason string
	cause  error
}

func (err commandError) Error() string { return err.reason }

func main() {
	os.Exit(execute(os.Args[1:], os.Stdout, os.Stderr, runBootstrapCommand))
}

type bootstrapInput struct {
	OwnerPhone   string
	OwnerName    string
	StoreName    string
	StoreAddress string
	PickupPoint  string
}

type inputError struct{ field string }

func (err inputError) Error() string { return fmt.Sprintf("invalid bootstrap field: %s", err.field) }

func loadBootstrapInput(lookup func(string) (string, bool)) (bootstrapInput, error) {
	fields := []struct {
		key    string
		target *string
	}{
		{ownerPhoneEnvironment, nil},
		{ownerNameEnvironment, nil},
		{storeNameEnvironment, nil},
		{storeAddressEnvironment, nil},
		{pickupPointEnvironment, nil},
	}
	input := bootstrapInput{}
	fields[0].target = &input.OwnerPhone
	fields[1].target = &input.OwnerName
	fields[2].target = &input.StoreName
	fields[3].target = &input.StoreAddress
	fields[4].target = &input.PickupPoint
	for _, field := range fields {
		value, present := lookup(field.key)
		if !present || !validRequiredText(value) {
			return bootstrapInput{}, inputError{field: field.key}
		}
		*field.target = value
	}
	if !canonicalPhone(input.OwnerPhone) {
		return bootstrapInput{}, inputError{field: ownerPhoneEnvironment}
	}
	return input, nil
}

func validRequiredText(value string) bool {
	return utf8.ValidString(value) && value != "" && len(value) <= maxBootstrapTextBytes && strings.TrimSpace(value) == value
}

func canonicalPhone(phone string) bool {
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

func execute(args []string, stdout, stderr io.Writer, command commandFunc) int {
	if len(args) != 0 {
		_, _ = io.WriteString(stderr, "usage: order-bootstrap\n")
		return 2
	}
	outcome, err := command(context.Background())
	logger := slog.New(slog.NewJSONHandler(stderr, nil))
	if err != nil {
		reason := reasonBootstrapFailed
		if safe, ok := err.(commandError); ok {
			reason = safe.reason
		}
		logger.Error("bootstrap failed", "event", "bootstrap_failed", "reason", reason)
		return 1
	}
	logger.Info("bootstrap completed", "event", "bootstrap_complete", "outcome", outcome)
	return 0
}

func runBootstrapCommand(ctx context.Context) (bootstrapOutcome, error) {
	input, err := loadBootstrapInput(os.LookupEnv)
	if err != nil {
		return "", commandError{reason: reasonBootstrapInputInvalid, cause: err}
	}
	cfg, err := config.Load()
	if err != nil {
		return "", commandError{reason: config.Reason(err), cause: err}
	}
	db, err := database.Open(cfg.Database)
	if err != nil {
		return "", commandError{reason: database.Reason(err), cause: err}
	}
	defer closeBootstrapDatabase(db)
	set, err := migrate.Load(migrations.FS)
	if err != nil {
		return "", commandError{reason: reasonBootstrapFailed, cause: err}
	}
	state := migrate.Check(ctx, db, set)
	if !state.Ready {
		return "", commandError{reason: state.Reason, cause: errors.New(state.Reason)}
	}
	outcome, err := bootstrap(ctx, db, input)
	if errors.Is(err, errBootstrapConflict) {
		return "", commandError{reason: reasonBootstrapConflict, cause: err}
	}
	if err != nil {
		return "", commandError{reason: "database_unavailable", cause: err}
	}
	return outcome, nil
}

func closeBootstrapDatabase(db *sql.DB) { _ = db.Close() }
