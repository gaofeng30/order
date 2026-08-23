package subscription

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
)

func TestRecordConsentUsesOrderLockAndWritesReceiptLast(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 24, 16, 30, 0, 0, time.UTC)
	db, script := openScriptDB(t,
		queryStep("FROM notification_consents", []string{"order_id", "kind", "decision", "grant_sequence", "template_config_version", "decided_at"}),
		beginStep(),
		queryStep("FROM orders WHERE id=? FOR UPDATE", []string{"user_id"}, []driver.Value{uint64(7)}),
		queryStep("ORDER BY grant_sequence DESC LIMIT 1 FOR UPDATE", []string{"grant_sequence"}),
		execStep("INSERT INTO notification_consents"),
		execStep("INSERT INTO action_audits"),
		commitStep(),
	)
	service := New(db, NewFakeProvider())
	service.now = func() time.Time { return now }

	got, err := service.RecordConsent(context.Background(), WriteMeta{
		ActorUserID: 7, IdempotencyKey: "ready-42", RequestID: "request-ready-42",
	}, ConsentInput{OrderID: 42, Kind: KindReady, Decision: DecisionAccepted, TemplateConfigVersion: 3})
	if err != nil {
		t.Fatalf("RecordConsent() error = %v", err)
	}
	if got.GrantSequence != 1 || !got.Available || !got.DecidedAt.Equal(now) {
		t.Fatalf("RecordConsent() = %#v", got)
	}
	script.assertDone(t)
}

func TestRecordConsentReplaysFirstDecisionAndRejectsKeyReuse(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 24, 16, 45, 0, 0, time.UTC)
	row := []driver.Value{uint64(42), "READY", "ACCEPTED", uint64(1), uint64(3), now}
	t.Run("same command", func(t *testing.T) {
		db, script := openScriptDB(t,
			queryStep("FROM notification_consents", []string{"order_id", "kind", "decision", "grant_sequence", "template_config_version", "decided_at"}, row),
		)
		service := New(db, NewFakeProvider())
		got, err := service.RecordConsent(context.Background(), WriteMeta{ActorUserID: 7, IdempotencyKey: "ready-42", RequestID: "replay-ready-42"}, ConsentInput{OrderID: 42, Kind: KindReady, Decision: DecisionAccepted, TemplateConfigVersion: 3})
		if err != nil || got.GrantSequence != 1 || !got.Available {
			t.Fatalf("RecordConsent() = %#v, %v", got, err)
		}
		script.assertDone(t)
	})
	t.Run("different command", func(t *testing.T) {
		db, script := openScriptDB(t,
			queryStep("FROM notification_consents", []string{"order_id", "kind", "decision", "grant_sequence", "template_config_version", "decided_at"}, row),
		)
		service := New(db, NewFakeProvider())
		_, err := service.RecordConsent(context.Background(), WriteMeta{ActorUserID: 7, IdempotencyKey: "ready-42", RequestID: "conflict-ready-42"}, ConsentInput{OrderID: 42, Kind: KindReady, Decision: DecisionRejected, TemplateConfigVersion: 3})
		if !errors.Is(err, ErrIdempotencyConflict) {
			t.Fatalf("RecordConsent() error = %v, want ErrIdempotencyConflict", err)
		}
		script.assertDone(t)
	})
}

func TestRecordConsentRetriesOneWholeTransactionOnDeadlock(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 24, 16, 50, 0, 0, time.UTC)
	db, script := openScriptDB(t,
		queryStep("FROM notification_consents", []string{"order_id", "kind", "decision", "grant_sequence", "template_config_version", "decided_at"}),
		beginStep(),
		queryErrorStep("FROM orders WHERE id=? FOR UPDATE", &mysql.MySQLError{Number: 1213, Message: "synthetic deadlock"}),
		rollbackStep(),
		beginStep(),
		queryStep("FROM orders WHERE id=? FOR UPDATE", []string{"user_id"}, []driver.Value{uint64(7)}),
		queryStep("ORDER BY grant_sequence DESC LIMIT 1 FOR UPDATE", []string{"grant_sequence"}),
		execStep("INSERT INTO notification_consents"),
		execStep("INSERT INTO action_audits"),
		commitStep(),
	)
	service := New(db, NewFakeProvider())
	service.now = func() time.Time { return now }
	got, err := service.RecordConsent(context.Background(), WriteMeta{ActorUserID: 7, IdempotencyKey: "deadlock-ready-42", RequestID: "deadlock-request-42"}, ConsentInput{OrderID: 42, Kind: KindReady, Decision: DecisionAccepted, TemplateConfigVersion: 3})
	if err != nil || got.GrantSequence != 1 {
		t.Fatalf("RecordConsent() = %#v, %v", got, err)
	}
	script.assertDone(t)
}

func TestEnqueueInTxLocksOrderConsentOutboxAndConsumesAcceptedConsent(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 24, 17, 0, 0, 0, time.UTC)
	db, script := openScriptDB(t,
		beginStep(),
		queryStep("FROM orders WHERE id=? FOR UPDATE", []string{"user_id"}, []driver.Value{uint64(7)}),
		queryStep("FROM notification_consents", []string{"id", "user_id", "template_config_version", "consumed_at"}, []driver.Value{uint64(9), uint64(7), uint64(3), nil}),
		queryStep("FROM notification_outbox", []string{"consent_id", "recipient_user_id", "template_config_version", "immutable_message_json"}),
		execInspectStep("INSERT INTO notification_outbox", func(arguments []driver.NamedValue) error {
			if len(arguments) != 8 {
				return fmt.Errorf("insert arguments = %d, want 8", len(arguments))
			}
			message, ok := arguments[4].Value.([]byte)
			if !ok {
				return fmt.Errorf("message argument type = %T", arguments[4].Value)
			}
			want := `{"order_number":"ORDER-42","pickup_date":"2026-08-25","pickup_time":"12:00","pickup_point":"North gate"}`
			if string(message) != want {
				return fmt.Errorf("immutable message = %s", message)
			}
			return nil
		}),
		execStep("UPDATE notification_consents SET consumed_at=?"),
		commitStep(),
	)
	transaction, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	service := New(db, NewFakeProvider())
	service.now = func() time.Time { return now }
	intent := NotificationIntent{
		OrderID: 42, RecipientUserID: 7, Kind: KindReady, AvailableAt: now,
		Message: Message{OrderNumber: "ORDER-42", PickupDate: "2026-08-25", PickupTime: "12:00", PickupPoint: "North gate"},
	}
	if err := service.EnqueueInTx(context.Background(), transaction, intent); err != nil {
		_ = transaction.Rollback()
		t.Fatalf("EnqueueInTx() error = %v", err)
	}
	if err := transaction.Commit(); err != nil {
		t.Fatal(err)
	}
	script.assertDone(t)
}

func TestEnqueueInTxWithoutAcceptedConsentIsNoOp(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 24, 17, 15, 0, 0, time.UTC)
	db, script := openScriptDB(t,
		beginStep(),
		queryStep("FROM orders WHERE id=? FOR UPDATE", []string{"user_id"}, []driver.Value{uint64(7)}),
		queryStep("FROM notification_consents", []string{"id", "user_id", "template_config_version", "consumed_at"}),
		commitStep(),
	)
	transaction, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	service := New(db, NewFakeProvider())
	service.now = func() time.Time { return now }
	intent := NotificationIntent{
		OrderID: 42, RecipientUserID: 7, Kind: KindReady, AvailableAt: now,
		Message: Message{OrderNumber: "ORDER-42", PickupDate: "2026-08-25", PickupTime: "12:00", PickupPoint: "North gate"},
	}
	if err := service.EnqueueInTx(context.Background(), transaction, intent); err != nil {
		_ = transaction.Rollback()
		t.Fatalf("EnqueueInTx() without accepted consent error = %v", err)
	}
	if err := transaction.Commit(); err != nil {
		t.Fatal(err)
	}
	script.assertDone(t)
}

func TestRunDueCommitsLeaseBeforeProviderAndMarksSentWithCAS(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 24, 18, 30, 0, 0, time.UTC)
	message := []byte(`{"order_number":"ORDER-42","pickup_date":"2026-08-25","pickup_time":"12:00","pickup_point":"North gate"}`)
	db, script := openScriptDB(t,
		beginStep(),
		queryStep("FOR UPDATE SKIP LOCKED", []string{"id", "order_id", "recipient_user_id", "kind", "immutable_message_json", "template_config_version", "attempt_count", "record_version"}, []driver.Value{uint64(11), uint64(42), uint64(7), "READY", message, uint64(3), int64(0), uint64(1)}),
		execStep("SET state='IN_FLIGHT'"),
		commitStep(),
		execStep("SET state='SENT'"),
	)
	provider := NewFakeProvider()
	service := New(db, provider)
	service.owner = func() ([16]byte, error) { return [16]byte{1}, nil }

	got, err := service.RunDue(context.Background(), now, 1)
	if err != nil {
		t.Fatalf("RunDue() error = %v", err)
	}
	if got != (RunResult{Claimed: 1, Sent: 1}) {
		t.Fatalf("RunDue() = %#v", got)
	}
	if deliveries := provider.Deliveries(); len(deliveries) != 1 || deliveries[0].AttemptCount != 1 {
		t.Fatalf("provider deliveries = %#v", deliveries)
	}
	script.assertDone(t)
}

type scriptStep struct {
	kind     string
	contains string
	columns  []string
	rows     [][]driver.Value
	validate func([]driver.NamedValue) error
	err      error
}

type dbScript struct {
	mu    sync.Mutex
	steps []scriptStep
}

func (script *dbScript) pop(kind, query string) (scriptStep, error) {
	script.mu.Lock()
	defer script.mu.Unlock()
	if len(script.steps) == 0 {
		return scriptStep{}, fmt.Errorf("unexpected %s: %s", kind, query)
	}
	step := script.steps[0]
	if step.kind != kind || (step.contains != "" && !strings.Contains(compactSQL(query), step.contains)) {
		return scriptStep{}, fmt.Errorf("got %s %q, want %s containing %q", kind, compactSQL(query), step.kind, step.contains)
	}
	script.steps = script.steps[1:]
	return step, nil
}

func (script *dbScript) assertDone(t *testing.T) {
	t.Helper()
	script.mu.Lock()
	defer script.mu.Unlock()
	if len(script.steps) != 0 {
		t.Fatalf("unconsumed database steps = %#v", script.steps)
	}
}

func queryStep(contains string, columns []string, rows ...[]driver.Value) scriptStep {
	return scriptStep{kind: "query", contains: contains, columns: columns, rows: rows}
}
func queryErrorStep(contains string, err error) scriptStep {
	return scriptStep{kind: "query", contains: contains, err: err}
}
func execStep(contains string) scriptStep { return scriptStep{kind: "exec", contains: contains} }
func execInspectStep(contains string, validate func([]driver.NamedValue) error) scriptStep {
	return scriptStep{kind: "exec", contains: contains, validate: validate}
}
func beginStep() scriptStep    { return scriptStep{kind: "begin"} }
func commitStep() scriptStep   { return scriptStep{kind: "commit"} }
func rollbackStep() scriptStep { return scriptStep{kind: "rollback"} }

var scriptDriverSequence atomic.Uint64

func openScriptDB(t *testing.T, steps ...scriptStep) (*sql.DB, *dbScript) {
	t.Helper()
	script := &dbScript{steps: append([]scriptStep(nil), steps...)}
	name := fmt.Sprintf("subscription-script-%d", scriptDriverSequence.Add(1))
	sql.Register(name, &scriptDriver{script: script})
	db, err := sql.Open(name, "")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db, script
}

type scriptDriver struct{ script *dbScript }

func (driver *scriptDriver) Open(string) (driver.Conn, error) {
	return &scriptConn{script: driver.script}, nil
}

type scriptConn struct{ script *dbScript }

func (*scriptConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare unsupported")
}
func (*scriptConn) Close() error { return nil }
func (connection *scriptConn) Begin() (driver.Tx, error) {
	return connection.BeginTx(context.Background(), driver.TxOptions{})
}
func (connection *scriptConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	if _, err := connection.script.pop("begin", ""); err != nil {
		return nil, err
	}
	return &scriptTx{script: connection.script}, nil
}
func (connection *scriptConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	step, err := connection.script.pop("query", query)
	if err != nil {
		return nil, err
	}
	if step.err != nil {
		return nil, step.err
	}
	return &scriptRows{columns: step.columns, rows: step.rows}, nil
}
func (connection *scriptConn) ExecContext(_ context.Context, query string, arguments []driver.NamedValue) (driver.Result, error) {
	step, err := connection.script.pop("exec", query)
	if err != nil {
		return nil, err
	}
	if step.err != nil {
		return nil, step.err
	}
	if step.validate != nil {
		if err := step.validate(arguments); err != nil {
			return nil, err
		}
	}
	return driver.RowsAffected(1), nil
}

type scriptTx struct{ script *dbScript }

func (transaction *scriptTx) Commit() error {
	_, err := transaction.script.pop("commit", "")
	return err
}
func (transaction *scriptTx) Rollback() error {
	_, err := transaction.script.pop("rollback", "")
	return err
}

type scriptRows struct {
	columns []string
	rows    [][]driver.Value
	index   int
}

func (rows *scriptRows) Columns() []string { return rows.columns }
func (*scriptRows) Close() error           { return nil }
func (rows *scriptRows) Next(destination []driver.Value) error {
	if rows.index >= len(rows.rows) {
		return io.EOF
	}
	copy(destination, rows.rows[rows.index])
	rows.index++
	return nil
}

func compactSQL(query string) string { return strings.Join(strings.Fields(query), " ") }
