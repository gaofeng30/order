package paymentorder

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
)

func TestMySQLDeadlockRetriesWholeApplyExactlyOnce(t *testing.T) {
	state := &faultDriverState{beginErrors: []error{
		&mysql.MySQLError{Number: 1213, Message: "synthetic deadlock"},
		errors.New("synthetic disconnected transaction"),
	}}
	db := openFaultDB(t, state)
	service := NewMySQLApplication(db, &fixedQuoteSource{}, NewFakeProvider(), testServiceConfig())
	if _, _, err := service.applyReady(context.Background(), 31, false, 0); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("applyReady(deadlock then disconnect) = %v", err)
	}
	if got := state.beginCount.Load(); got != 2 {
		t.Fatalf("transaction attempts = %d, want 2", got)
	}
}

func TestDisconnectedMySQLFailsClosedBeforeProviderCreate(t *testing.T) {
	state := &faultDriverState{queryErr: errors.New("synthetic disconnected pool")}
	db := openFaultDB(t, state)
	provider := NewFakeProvider()
	service := NewMySQLApplication(db, &fixedQuoteSource{}, provider, testServiceConfig())
	_, err := service.Prepare(context.Background(), WriteMeta{ActorUserID: 42, IdempotencyKey: "prepare-91", RequestID: "request-91"}, 91)
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Prepare(disconnected) = %v", err)
	}
	if got := provider.CreateCount("anything"); got != 0 {
		t.Fatalf("provider CreateCount = %d", got)
	}
}

func TestMySQLRetryClassificationDoesNotRetryOtherProviderOrSQLErrors(t *testing.T) {
	if !isRetryableMySQL(&mysql.MySQLError{Number: 1205}) || !isRetryableMySQL(&mysql.MySQLError{Number: 1213}) {
		t.Fatal("1205/1213 must be retryable")
	}
	for _, err := range []error{&mysql.MySQLError{Number: 1062}, errors.New("network"), ErrUnavailable} {
		if isRetryableMySQL(err) {
			t.Fatalf("unexpected retryable error: %v", err)
		}
	}
}

func testServiceConfig() Config {
	return Config{AppID: "wx-local", MerchantID: "mch-local", Description: "预约点餐", PaymentNotifyURL: "https://local.invalid/notify", LeaseDuration: time.Second, ReconcileInterval: time.Second}
}

var faultDriverSequence atomic.Uint64

type faultDriverState struct {
	mu          sync.Mutex
	beginErrors []error
	queryErr    error
	beginCount  atomic.Uint64
}

type faultDriver struct{ state *faultDriverState }

func (driverValue faultDriver) Open(string) (driver.Conn, error) {
	return &faultConnection{state: driverValue.state}, nil
}

type faultConnection struct{ state *faultDriverState }

func (*faultConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}
func (*faultConnection) Close() error { return nil }
func (connection *faultConnection) Begin() (driver.Tx, error) {
	return connection.BeginTx(context.Background(), driver.TxOptions{})
}
func (connection *faultConnection) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	index := int(connection.state.beginCount.Add(1)) - 1
	connection.state.mu.Lock()
	defer connection.state.mu.Unlock()
	if index < len(connection.state.beginErrors) {
		return nil, connection.state.beginErrors[index]
	}
	return &faultTransaction{}, nil
}
func (connection *faultConnection) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	if connection.state.queryErr != nil {
		return nil, connection.state.queryErr
	}
	return nil, errors.New("unexpected query")
}

type faultTransaction struct{}

func (*faultTransaction) Commit() error   { return nil }
func (*faultTransaction) Rollback() error { return nil }

func openFaultDB(t *testing.T, state *faultDriverState) *sql.DB {
	t.Helper()
	name := fmt.Sprintf("paymentorder-fault-%d", faultDriverSequence.Add(1))
	sql.Register(name, faultDriver{state: state})
	db, err := sql.Open(name, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
