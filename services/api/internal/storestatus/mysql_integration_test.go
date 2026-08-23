package storestatus

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/merchantidentity"
	"github.com/gaofeng30/order/services/api/internal/storefront"
)

func TestApplyRejectsDisabledAccountWithoutMutation(t *testing.T) {
	withStoreStatusSchema(t, func(db *sql.DB) {
		now := time.Date(2026, time.August, 23, 10, 0, 0, 123456000, time.UTC)
		userID := insertStoreStatusUser(t, db, "opaque-store-status-disabled", now)
		insertStoreStatusAccount(t, db, userID, merchantidentity.RoleOwner, false, now)
		insertStorefrontSettings(t, db, string(storefront.BusinessOpen))

		result, err := New(db, merchantidentity.NewRepository(db), func() time.Time { return now }).Apply(context.Background(), Command{
			UserID: userID, DesiredStatus: storefront.BusinessClosed,
			IdempotencyKey: "disabled-command", RequestID: "disabled-request",
		})

		if !errors.Is(err, merchantidentity.ErrMerchantAccountNotAvailable) || result != (Result{}) {
			t.Fatalf("Apply() = %#v, %v; want zero account-not-available", result, err)
		}
		if got := readBusinessStatus(t, db); got != string(storefront.BusinessOpen) {
			t.Fatalf("business_status = %q, want open", got)
		}
		if got := countStoreStatusAudits(t, db); got != 0 {
			t.Fatalf("audit count = %d, want 0", got)
		}
	})
}

func TestApplyOwnerChangesOnlyBusinessStatus(t *testing.T) {
	withStoreStatusSchema(t, func(db *sql.DB) {
		now := time.Date(2026, time.August, 23, 10, 5, 0, 234567000, time.UTC)
		userID := insertStoreStatusUser(t, db, "opaque-store-status-owner", now)
		insertStoreStatusAccount(t, db, userID, merchantidentity.RoleOwner, true, now)
		insertStorefrontSettings(t, db, string(storefront.BusinessOpen))
		beforeColumns := readStorefrontNonStatus(t, db)

		result, err := New(db, merchantidentity.NewRepository(db), func() time.Time { return now }).Apply(context.Background(), Command{
			UserID: userID, DesiredStatus: storefront.BusinessClosed,
			IdempotencyKey: "owner-command", RequestID: "owner-request",
		})

		want := Result{Before: storefront.BusinessOpen, After: storefront.BusinessClosed, Changed: true}
		if err != nil || result != want {
			t.Fatalf("Apply() = %#v, %v; want %#v", result, err, want)
		}
		if got := readBusinessStatus(t, db); got != string(storefront.BusinessClosed) {
			t.Fatalf("business_status = %q, want closed", got)
		}
		if afterColumns := readStorefrontNonStatus(t, db); afterColumns != beforeColumns {
			t.Fatalf("non-status storefront columns changed: before=%#v after=%#v", beforeColumns, afterColumns)
		}
		if got := countStoreStatusAudits(t, db); got != 1 {
			t.Fatalf("audit count = %d, want 1", got)
		}
	})
}

func TestApplySubaccountChangesOnlyBusinessStatus(t *testing.T) {
	withStoreStatusSchema(t, func(db *sql.DB) {
		now := time.Date(2026, time.August, 23, 10, 10, 0, 345678000, time.UTC)
		userID := insertStoreStatusUser(t, db, "opaque-store-status-subaccount", now)
		insertStoreStatusAccount(t, db, userID, merchantidentity.RoleSubaccount, true, now)
		insertStorefrontSettings(t, db, string(storefront.BusinessClosed))
		beforeColumns := readStorefrontNonStatus(t, db)

		result, err := New(db, merchantidentity.NewRepository(db), func() time.Time { return now }).Apply(context.Background(), Command{
			UserID: userID, DesiredStatus: storefront.BusinessCutoff,
			IdempotencyKey: "subaccount-command", RequestID: "subaccount-request",
		})

		want := Result{Before: storefront.BusinessClosed, After: storefront.BusinessCutoff, Changed: true}
		if err != nil || result != want {
			t.Fatalf("Apply() = %#v, %v; want %#v", result, err, want)
		}
		if got := readBusinessStatus(t, db); got != string(storefront.BusinessCutoff) {
			t.Fatalf("business_status = %q, want cutoff", got)
		}
		if afterColumns := readStorefrontNonStatus(t, db); afterColumns != beforeColumns {
			t.Fatalf("non-status storefront columns changed: before=%#v after=%#v", beforeColumns, afterColumns)
		}
		if got := countStoreStatusAudits(t, db); got != 1 {
			t.Fatalf("audit count = %d, want 1", got)
		}
	})
}

func TestApplyWritesExactSuccessAudit(t *testing.T) {
	withStoreStatusSchema(t, func(db *sql.DB) {
		now := time.Date(2026, time.August, 23, 10, 15, 0, 456789000, time.UTC)
		userID := insertStoreStatusUser(t, db, "opaque-store-status-exact-audit", now)
		accountID := insertStoreStatusAccount(t, db, userID, merchantidentity.RoleSubaccount, true, now)
		insertStorefrontSettings(t, db, string(storefront.BusinessOpen))
		key := "exact-audit-command"

		result, err := New(db, merchantidentity.NewRepository(db), func() time.Time { return now }).Apply(context.Background(), Command{
			UserID: userID, DesiredStatus: storefront.BusinessCutoff,
			IdempotencyKey: key, RequestID: "exact-audit-request",
		})
		if err != nil || result != (Result{Before: storefront.BusinessOpen, After: storefront.BusinessCutoff, Changed: true}) {
			t.Fatalf("Apply() = %#v, %v", result, err)
		}

		var liveAccountID, snapshotID, snapshotAuth, actorUserID, targetID uint64
		var role merchantidentity.Role
		var action, auditResult, reason, targetType, stateBefore, stateAfter string
		var requestID, keyHash []byte
		var occurredAt time.Time
		if err := db.QueryRowContext(context.Background(), `
			SELECT merchant_account_id,account_id_snapshot,role_snapshot,auth_version_snapshot,
			       actor_user_id,action,result,reason,target_type,target_id,request_id,
			       idempotency_key_hash,state_before,state_after,occurred_at
			FROM merchant_action_audits WHERE actor_user_id=?
		`, userID).Scan(
			&liveAccountID, &snapshotID, &role, &snapshotAuth, &actorUserID, &action, &auditResult, &reason,
			&targetType, &targetID, &requestID, &keyHash, &stateBefore, &stateAfter, &occurredAt,
		); err != nil {
			t.Fatal("read exact store status audit failed")
		}
		wantHash := sha256.Sum256([]byte(key))
		if liveAccountID != accountID || snapshotID != accountID || role != merchantidentity.RoleSubaccount || snapshotAuth != 1 ||
			actorUserID != userID || action != string(merchantidentity.ActionStoreStatusWrite) || auditResult != "SUCCEEDED" ||
			reason != "OPERATING_STATUS_CHANGED" || targetType != "storefront_settings" || targetID != 1 ||
			!bytes.Equal(requestID, []byte("exact-audit-request")) || !bytes.Equal(keyHash, wantHash[:]) ||
			stateBefore != string(storefront.BusinessOpen) || stateAfter != string(storefront.BusinessCutoff) || !occurredAt.Equal(now) {
			t.Fatalf("audit facts were not exact")
		}
	})
}

func TestApplyUsesFrozenAuthorizationActionAndTarget(t *testing.T) {
	withStoreStatusSchema(t, func(db *sql.DB) {
		now := time.Date(2026, time.August, 23, 10, 17, 0, 456789000, time.UTC)
		userID := insertStoreStatusUser(t, db, "opaque-store-status-authorization-contract", now)
		insertStoreStatusAccount(t, db, userID, merchantidentity.RoleOwner, true, now)
		insertStorefrontSettings(t, db, string(storefront.BusinessOpen))
		authorizer := &recordingAuthorizer{delegate: merchantidentity.NewRepository(db)}

		result, err := New(db, authorizer, func() time.Time { return now }).Apply(context.Background(), Command{
			UserID: userID, DesiredStatus: storefront.BusinessClosed,
			IdempotencyKey: "authorization-contract-command", RequestID: "authorization-contract-request",
		})
		if err != nil || result.After != storefront.BusinessClosed || !result.Changed {
			t.Fatalf("Apply() = %#v, %v", result, err)
		}
		if authorizer.calls != 1 || authorizer.action != merchantidentity.ActionStoreStatusWrite ||
			authorizer.target != (merchantidentity.Target{Type: "storefront_settings", ID: 1}) {
			t.Fatalf("authorization contract = calls %d action %q target %#v", authorizer.calls, authorizer.action, authorizer.target)
		}
	})
}

func TestApplyNoOpReturnsUnchangedAndAuditsOnce(t *testing.T) {
	withStoreStatusSchema(t, func(db *sql.DB) {
		now := time.Date(2026, time.August, 23, 10, 20, 0, 567890000, time.UTC)
		userID := insertStoreStatusUser(t, db, "opaque-store-status-noop", now)
		insertStoreStatusAccount(t, db, userID, merchantidentity.RoleOwner, true, now)
		insertStorefrontSettings(t, db, string(storefront.BusinessOpen))
		beforeColumns := readStorefrontNonStatus(t, db)

		result, err := New(db, merchantidentity.NewRepository(db), func() time.Time { return now }).Apply(context.Background(), Command{
			UserID: userID, DesiredStatus: storefront.BusinessOpen,
			IdempotencyKey: "noop-command", RequestID: "noop-request",
		})
		want := Result{Before: storefront.BusinessOpen, After: storefront.BusinessOpen, Changed: false}
		if err != nil || result != want {
			t.Fatalf("Apply() = %#v, %v; want %#v", result, err, want)
		}
		if got := readBusinessStatus(t, db); got != string(storefront.BusinessOpen) || readStorefrontNonStatus(t, db) != beforeColumns {
			t.Fatal("no-op changed persisted storefront settings")
		}
		var count int
		var reason, before, after string
		if err := db.QueryRowContext(context.Background(), `
			SELECT COUNT(*),MIN(reason),MIN(state_before),MIN(state_after)
			FROM merchant_action_audits WHERE actor_user_id=? AND action=?
		`, userID, merchantidentity.ActionStoreStatusWrite).Scan(&count, &reason, &before, &after); err != nil {
			t.Fatal("read no-op audit failed")
		}
		if count != 1 || reason != "OPERATING_STATUS_UNCHANGED" || before != "open" || after != "open" {
			t.Fatalf("no-op audit = %d/%s/%s/%s", count, reason, before, after)
		}
	})
}

func TestApplyReplayReturnsFirstResultWithoutRewrite(t *testing.T) {
	withStoreStatusSchema(t, func(db *sql.DB) {
		now := time.Date(2026, time.August, 23, 10, 25, 0, 678901000, time.UTC)
		userID := insertStoreStatusUser(t, db, "opaque-store-status-replay", now)
		insertStoreStatusAccount(t, db, userID, merchantidentity.RoleOwner, true, now)
		insertStorefrontSettings(t, db, string(storefront.BusinessOpen))
		core := New(db, merchantidentity.NewRepository(db), func() time.Time { return now })

		first, err := core.Apply(context.Background(), Command{
			UserID: userID, DesiredStatus: storefront.BusinessClosed,
			IdempotencyKey: "replayed-command", RequestID: "replay-first-request",
		})
		if err != nil || first != (Result{Before: storefront.BusinessOpen, After: storefront.BusinessClosed, Changed: true}) {
			t.Fatalf("first Apply() = %#v, %v", first, err)
		}
		if _, err := core.Apply(context.Background(), Command{
			UserID: userID, DesiredStatus: storefront.BusinessOpen,
			IdempotencyKey: "intervening-command", RequestID: "replay-intervening-request",
		}); err != nil {
			t.Fatalf("intervening Apply() error = %v", err)
		}

		replayed, err := core.Apply(context.Background(), Command{
			UserID: userID, DesiredStatus: storefront.BusinessClosed,
			IdempotencyKey: "replayed-command", RequestID: "replay-later-request",
		})
		if err != nil || replayed != first {
			t.Fatalf("replayed Apply() = %#v, %v; want first %#v", replayed, err, first)
		}
		if got := readBusinessStatus(t, db); got != string(storefront.BusinessOpen) {
			t.Fatalf("replay rewrote business_status to %q", got)
		}
		if got := countStoreStatusAudits(t, db); got != 2 {
			t.Fatalf("audit count = %d, want 2", got)
		}
	})
}

func TestApplyRejectsSameKeyWithDifferentDesiredStatus(t *testing.T) {
	withStoreStatusSchema(t, func(db *sql.DB) {
		now := time.Date(2026, time.August, 23, 10, 30, 0, 789012000, time.UTC)
		userID := insertStoreStatusUser(t, db, "opaque-store-status-conflict", now)
		insertStoreStatusAccount(t, db, userID, merchantidentity.RoleOwner, true, now)
		insertStorefrontSettings(t, db, string(storefront.BusinessOpen))
		authorizer := &recordingAuthorizer{delegate: merchantidentity.NewRepository(db)}
		core := New(db, authorizer, func() time.Time { return now })
		command := Command{
			UserID: userID, DesiredStatus: storefront.BusinessClosed,
			IdempotencyKey: "conflicting-command", RequestID: "conflict-first-request",
		}
		if _, err := core.Apply(context.Background(), command); err != nil {
			t.Fatalf("first Apply() error = %v", err)
		}

		command.DesiredStatus = storefront.BusinessCutoff
		command.RequestID = "conflict-second-request"
		result, err := core.Apply(context.Background(), command)
		if !errors.Is(err, ErrIdempotencyConflict) || result != (Result{}) {
			t.Fatalf("conflicting Apply() = %#v, %v; want zero conflict", result, err)
		}
		if got := readBusinessStatus(t, db); got != string(storefront.BusinessClosed) {
			t.Fatalf("conflict changed business_status to %q", got)
		}
		if got := countStoreStatusAudits(t, db); got != 1 {
			t.Fatalf("audit count = %d, want 1", got)
		}
		if authorizer.calls != 2 {
			t.Fatalf("authorization calls = %d, want one per command without conflict retry", authorizer.calls)
		}
	})
}

func TestApplyConcurrentSameKeyConvergesToOneResultAndAudit(t *testing.T) {
	withStoreStatusSchema(t, func(db *sql.DB) {
		now := time.Date(2026, time.August, 23, 10, 35, 0, 890123000, time.UTC)
		userID := insertStoreStatusUser(t, db, "opaque-store-status-concurrent-replay", now)
		insertStoreStatusAccount(t, db, userID, merchantidentity.RoleOwner, true, now)
		insertStorefrontSettings(t, db, string(storefront.BusinessOpen))
		authorizer := &twoCallBarrierAuthorizer{
			delegate: merchantidentity.NewRepository(db), ready: make(chan struct{}),
		}
		core := New(db, authorizer, func() time.Time { return now })

		type applyResult struct {
			result Result
			err    error
		}
		results := make(chan applyResult, 2)
		for index := range 2 {
			requestID := "concurrent-replay-request-" + string(rune('a'+index))
			go func() {
				result, err := core.Apply(context.Background(), Command{
					UserID: userID, DesiredStatus: storefront.BusinessClosed,
					IdempotencyKey: "concurrent-replayed-command", RequestID: requestID,
				})
				results <- applyResult{result: result, err: err}
			}()
		}
		want := Result{Before: storefront.BusinessOpen, After: storefront.BusinessClosed, Changed: true}
		for range 2 {
			select {
			case got := <-results:
				if got.err != nil || got.result != want {
					t.Fatalf("concurrent Apply() = %#v, %v; want %#v", got.result, got.err, want)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("concurrent same-key Apply did not complete")
			}
		}
		if got := readBusinessStatus(t, db); got != string(storefront.BusinessClosed) {
			t.Fatalf("business_status = %q, want closed", got)
		}
		if got := countStoreStatusAudits(t, db); got != 1 {
			t.Fatalf("audit count = %d, want 1", got)
		}
	})
}

func TestApplyConcurrentSameKeyNoOpRequiresSingletonLock(t *testing.T) {
	withStoreStatusSchema(t, func(db *sql.DB) {
		now := time.Date(2026, time.August, 23, 10, 37, 0, 890123000, time.UTC)
		userID := insertStoreStatusUser(t, db, "opaque-store-status-concurrent-noop", now)
		insertStoreStatusAccount(t, db, userID, merchantidentity.RoleOwner, true, now)
		insertStorefrontSettings(t, db, string(storefront.BusinessOpen))
		authorizer := &twoCallBarrierAuthorizer{delegate: merchantidentity.NewRepository(db), ready: make(chan struct{})}
		core := New(db, authorizer, func() time.Time { return now })

		type applyResult struct {
			result Result
			err    error
		}
		results := make(chan applyResult, 2)
		for index := range 2 {
			requestID := "concurrent-noop-request-" + string(rune('a'+index))
			go func() {
				result, err := core.Apply(context.Background(), Command{
					UserID: userID, DesiredStatus: storefront.BusinessOpen,
					IdempotencyKey: "concurrent-noop-command", RequestID: requestID,
				})
				results <- applyResult{result: result, err: err}
			}()
		}
		want := Result{Before: storefront.BusinessOpen, After: storefront.BusinessOpen, Changed: false}
		for range 2 {
			select {
			case got := <-results:
				if got.err != nil || got.result != want {
					t.Fatalf("concurrent no-op Apply() = %#v, %v; want %#v", got.result, got.err, want)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("concurrent same-key no-op Apply did not complete")
			}
		}
		if readBusinessStatus(t, db) != "open" || countStoreStatusAudits(t, db) != 1 {
			t.Fatal("concurrent same-key no-op was not serialized before replay")
		}
	})
}

func TestApplyConcurrentSameKeyDifferentDesiredHasOneWinner(t *testing.T) {
	withStoreStatusSchema(t, func(db *sql.DB) {
		now := time.Date(2026, time.August, 23, 10, 40, 0, 901234000, time.UTC)
		userID := insertStoreStatusUser(t, db, "opaque-store-status-concurrent-conflict", now)
		insertStoreStatusAccount(t, db, userID, merchantidentity.RoleOwner, true, now)
		insertStorefrontSettings(t, db, string(storefront.BusinessOpen))
		authorizer := &twoCallBarrierAuthorizer{delegate: merchantidentity.NewRepository(db), ready: make(chan struct{})}
		core := New(db, authorizer, func() time.Time { return now })

		type applyResult struct {
			result Result
			err    error
		}
		results := make(chan applyResult, 2)
		for index, desired := range []storefront.BusinessStatus{storefront.BusinessClosed, storefront.BusinessCutoff} {
			requestID := "concurrent-conflict-request-" + string(rune('a'+index))
			go func() {
				result, err := core.Apply(context.Background(), Command{
					UserID: userID, DesiredStatus: desired,
					IdempotencyKey: "concurrent-conflicting-command", RequestID: requestID,
				})
				results <- applyResult{result: result, err: err}
			}()
		}
		var winner Result
		successes, conflicts := 0, 0
		for range 2 {
			select {
			case got := <-results:
				switch {
				case got.err == nil:
					successes++
					winner = got.result
				case errors.Is(got.err, ErrIdempotencyConflict) && got.result == (Result{}):
					conflicts++
				default:
					t.Fatalf("concurrent conflicting Apply() = %#v, %v", got.result, got.err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("concurrent conflicting Apply did not complete")
			}
		}
		if successes != 1 || conflicts != 1 || winner.Before != storefront.BusinessOpen || !winner.Changed {
			t.Fatalf("successes=%d conflicts=%d winner=%#v", successes, conflicts, winner)
		}
		if got := readBusinessStatus(t, db); got != string(winner.After) {
			t.Fatalf("business_status = %q, want winner %q", got, winner.After)
		}
		if got := countStoreStatusAudits(t, db); got != 1 {
			t.Fatalf("audit count = %d, want 1", got)
		}
	})
}

func TestApplyConcurrentDifferentKeysFormSerializedChain(t *testing.T) {
	withStoreStatusSchema(t, func(db *sql.DB) {
		now := time.Date(2026, time.August, 23, 10, 45, 0, 12340000, time.UTC)
		userID := insertStoreStatusUser(t, db, "opaque-store-status-concurrent-chain", now)
		insertStoreStatusAccount(t, db, userID, merchantidentity.RoleOwner, true, now)
		insertStorefrontSettings(t, db, string(storefront.BusinessOpen))
		authorizer := &twoCallBarrierAuthorizer{delegate: merchantidentity.NewRepository(db), ready: make(chan struct{})}
		core := New(db, authorizer, func() time.Time { return now })

		errorsFound := make(chan error, 2)
		for index, desired := range []storefront.BusinessStatus{storefront.BusinessClosed, storefront.BusinessCutoff} {
			key := "concurrent-chain-key-" + string(rune('a'+index))
			requestID := "concurrent-chain-request-" + string(rune('a'+index))
			go func() {
				_, err := core.Apply(context.Background(), Command{
					UserID: userID, DesiredStatus: desired, IdempotencyKey: key, RequestID: requestID,
				})
				errorsFound <- err
			}()
		}
		for range 2 {
			select {
			case err := <-errorsFound:
				if err != nil {
					t.Fatalf("concurrent distinct-key Apply() error = %v", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("concurrent distinct-key Apply did not complete")
			}
		}
		rows, err := db.QueryContext(context.Background(), `
			SELECT state_before,state_after FROM merchant_action_audits
			WHERE actor_user_id=? AND action=? ORDER BY id
		`, userID, merchantidentity.ActionStoreStatusWrite)
		if err != nil {
			t.Fatal("read serialized audit chain failed")
		}
		defer rows.Close()
		chain := make([][2]string, 0, 2)
		for rows.Next() {
			var pair [2]string
			if err := rows.Scan(&pair[0], &pair[1]); err != nil {
				t.Fatal("scan serialized audit chain failed")
			}
			chain = append(chain, pair)
		}
		if err := rows.Err(); err != nil {
			t.Fatal("iterate serialized audit chain failed")
		}
		if len(chain) != 2 || chain[0][0] != "open" || chain[1][0] != chain[0][1] || chain[0][1] == chain[1][1] {
			t.Fatalf("serialized audit chain = %#v", chain)
		}
		if got := readBusinessStatus(t, db); got != chain[1][1] {
			t.Fatalf("business_status = %q, want final chain state %q", got, chain[1][1])
		}
	})
}

func TestApplyRejectsDeletedAndInvalidRoleAccounts(t *testing.T) {
	t.Run("deleted", func(t *testing.T) {
		withStoreStatusSchema(t, func(db *sql.DB) {
			now := time.Date(2026, time.August, 23, 10, 50, 0, 123456000, time.UTC)
			userID := insertStoreStatusUser(t, db, "opaque-store-status-deleted", now)
			accountID := insertStoreStatusAccount(t, db, userID, merchantidentity.RoleOwner, true, now)
			insertStorefrontSettings(t, db, string(storefront.BusinessOpen))
			if _, err := db.ExecContext(context.Background(), "DELETE FROM merchant_accounts WHERE id=?", accountID); err != nil {
				t.Fatal("delete merchant account fixture failed")
			}
			result, err := New(db, merchantidentity.NewRepository(db), func() time.Time { return now }).Apply(context.Background(), Command{
				UserID: userID, DesiredStatus: storefront.BusinessClosed,
				IdempotencyKey: "deleted-command", RequestID: "deleted-request",
			})
			if !errors.Is(err, merchantidentity.ErrMerchantAccountNotAvailable) || result != (Result{}) {
				t.Fatalf("deleted Apply() = %#v, %v", result, err)
			}
			if readBusinessStatus(t, db) != "open" || countStoreStatusAudits(t, db) != 0 {
				t.Fatal("deleted account caused a side effect")
			}
		})
	})

	t.Run("invalid role drift", func(t *testing.T) {
		withStoreStatusSchema(t, func(db *sql.DB) {
			now := time.Date(2026, time.August, 23, 10, 55, 0, 234567000, time.UTC)
			userID := insertStoreStatusUser(t, db, "opaque-store-status-invalid-role", now)
			accountID := insertStoreStatusAccount(t, db, userID, merchantidentity.RoleOwner, true, now)
			insertStorefrontSettings(t, db, string(storefront.BusinessOpen))
			if _, err := db.ExecContext(context.Background(), "ALTER TABLE merchant_accounts MODIFY role VARCHAR(32) NOT NULL"); err != nil {
				t.Fatal("prepare invalid role drift failed")
			}
			if _, err := db.ExecContext(context.Background(), "UPDATE merchant_accounts SET role='VIEWER' WHERE id=?", accountID); err != nil {
				t.Fatal("persist invalid role drift failed")
			}
			result, err := New(db, merchantidentity.NewRepository(db), func() time.Time { return now }).Apply(context.Background(), Command{
				UserID: userID, DesiredStatus: storefront.BusinessClosed,
				IdempotencyKey: "invalid-role-command", RequestID: "invalid-role-request",
			})
			if !errors.Is(err, merchantidentity.ErrUnavailable) || result != (Result{}) {
				t.Fatalf("invalid-role Apply() = %#v, %v", result, err)
			}
			if readBusinessStatus(t, db) != "open" || countStoreStatusAudits(t, db) != 0 {
				t.Fatal("invalid-role account caused a side effect")
			}
		})
	})
}

func TestApplySerializesRoleChangeAndAuditsLiveRole(t *testing.T) {
	withStoreStatusSchema(t, func(db *sql.DB) {
		now := time.Date(2026, time.August, 23, 11, 0, 0, 345678000, time.UTC)
		userID := insertStoreStatusUser(t, db, "opaque-store-status-role-order", now)
		accountID := insertStoreStatusAccount(t, db, userID, merchantidentity.RoleOwner, true, now)
		insertStorefrontSettings(t, db, string(storefront.BusinessOpen))
		authorizer := &pausingAuthorizer{
			delegate: merchantidentity.NewRepository(db), authorized: make(chan struct{}), release: make(chan struct{}),
		}
		core := New(db, authorizer, func() time.Time { return now })
		applyDone := make(chan error, 1)
		go func() {
			_, err := core.Apply(context.Background(), Command{
				UserID: userID, DesiredStatus: storefront.BusinessClosed,
				IdempotencyKey: "role-owner-command", RequestID: "role-owner-request",
			})
			applyDone <- err
		}()
		select {
		case <-authorizer.authorized:
		case <-time.After(5 * time.Second):
			t.Fatal("Apply did not reach live authorization")
		}

		roleChangeDone := make(chan error, 1)
		go func() {
			transaction, err := db.BeginTx(context.Background(), nil)
			if err != nil {
				roleChangeDone <- err
				return
			}
			defer transaction.Rollback()
			if _, err := transaction.ExecContext(context.Background(), `
				UPDATE merchant_accounts
				SET role='SUBACCOUNT',record_version=record_version+1,auth_version=auth_version+1,updated_at=?
				WHERE id=?
			`, now.Add(time.Minute), accountID); err != nil {
				roleChangeDone <- err
				return
			}
			roleChangeDone <- transaction.Commit()
		}()
		waitForStoreStatusLock(t, db, "merchant_accounts")
		select {
		case err := <-roleChangeDone:
			t.Fatalf("role change completed before authorized transaction: %v", err)
		default:
		}
		close(authorizer.release)
		select {
		case err := <-applyDone:
			if err != nil {
				t.Fatalf("OWNER Apply() error = %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("OWNER Apply did not complete")
		}
		select {
		case err := <-roleChangeDone:
			if err != nil {
				t.Fatalf("role change error = %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("role change did not resume")
		}

		if _, err := New(db, merchantidentity.NewRepository(db), func() time.Time { return now.Add(2 * time.Minute) }).Apply(context.Background(), Command{
			UserID: userID, DesiredStatus: storefront.BusinessCutoff,
			IdempotencyKey: "role-subaccount-command", RequestID: "role-subaccount-request",
		}); err != nil {
			t.Fatalf("SUBACCOUNT Apply() error = %v", err)
		}
		rows, err := db.QueryContext(context.Background(), `
			SELECT role_snapshot,auth_version_snapshot FROM merchant_action_audits
			WHERE actor_user_id=? AND action=? ORDER BY id
		`, userID, merchantidentity.ActionStoreStatusWrite)
		if err != nil {
			t.Fatal("read role-order audits failed")
		}
		defer rows.Close()
		type roleVersion struct {
			role    merchantidentity.Role
			version uint64
		}
		got := make([]roleVersion, 0, 2)
		for rows.Next() {
			var value roleVersion
			if err := rows.Scan(&value.role, &value.version); err != nil {
				t.Fatal("scan role-order audit failed")
			}
			got = append(got, value)
		}
		if len(got) != 2 || got[0] != (roleVersion{role: merchantidentity.RoleOwner, version: 1}) ||
			got[1] != (roleVersion{role: merchantidentity.RoleSubaccount, version: 2}) {
			t.Fatalf("role-order audit facts = %#v", got)
		}
	})
}

type pausingAuthorizer struct {
	delegate   merchantidentity.Authorizer
	authorized chan struct{}
	release    chan struct{}
}

type recordingAuthorizer struct {
	delegate merchantidentity.Authorizer
	calls    int
	action   merchantidentity.Action
	target   merchantidentity.Target
}

func (authorizer *recordingAuthorizer) AuthorizeInTx(
	ctx context.Context,
	transaction *sql.Tx,
	userID uint64,
	action merchantidentity.Action,
	target merchantidentity.Target,
) (merchantidentity.Authorization, error) {
	authorizer.calls++
	authorizer.action = action
	authorizer.target = target
	return authorizer.delegate.AuthorizeInTx(ctx, transaction, userID, action, target)
}

func (authorizer *pausingAuthorizer) AuthorizeInTx(
	ctx context.Context,
	transaction *sql.Tx,
	userID uint64,
	action merchantidentity.Action,
	target merchantidentity.Target,
) (merchantidentity.Authorization, error) {
	authorization, err := authorizer.delegate.AuthorizeInTx(ctx, transaction, userID, action, target)
	if err != nil {
		return merchantidentity.Authorization{}, err
	}
	close(authorizer.authorized)
	select {
	case <-authorizer.release:
		return authorization, nil
	case <-ctx.Done():
		return merchantidentity.Authorization{}, ctx.Err()
	}
}

func waitForStoreStatusLock(t *testing.T, db *sql.DB, table string) {
	t.Helper()
	var schemaName string
	if err := db.QueryRowContext(context.Background(), "SELECT DATABASE()").Scan(&schemaName); err != nil {
		t.Fatal("read store status schema failed")
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var waits int
		if err := db.QueryRowContext(context.Background(), `
			SELECT COUNT(*)
			FROM performance_schema.data_lock_waits AS w
			JOIN performance_schema.data_locks AS requested
			  ON requested.ENGINE_LOCK_ID=w.REQUESTING_ENGINE_LOCK_ID
			WHERE requested.OBJECT_SCHEMA=? AND requested.OBJECT_NAME=?
		`, schemaName, table).Scan(&waits); err != nil {
			t.Fatal("inspect store status lock wait failed")
		}
		if waits > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("no lock wait observed for %s", table)
}

type twoCallBarrierAuthorizer struct {
	delegate merchantidentity.Authorizer
	ready    chan struct{}
	mu       sync.Mutex
	calls    int
}

func (authorizer *twoCallBarrierAuthorizer) AuthorizeInTx(
	ctx context.Context,
	transaction *sql.Tx,
	userID uint64,
	action merchantidentity.Action,
	target merchantidentity.Target,
) (merchantidentity.Authorization, error) {
	authorization, err := authorizer.delegate.AuthorizeInTx(ctx, transaction, userID, action, target)
	if err != nil {
		return merchantidentity.Authorization{}, err
	}
	authorizer.mu.Lock()
	authorizer.calls++
	if authorizer.calls == 2 {
		close(authorizer.ready)
	}
	authorizer.mu.Unlock()
	select {
	case <-authorizer.ready:
		return authorization, nil
	case <-ctx.Done():
		return merchantidentity.Authorization{}, ctx.Err()
	}
}
