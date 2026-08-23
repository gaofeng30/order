package adminreport

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
)

var adminScriptSequence atomic.Uint64

func TestGetOrderRequiresOwnerAndProjectsItems(t *testing.T) {
	paidAt := time.Date(2026, 8, 25, 1, 0, 0, 0, time.UTC)
	db, script := openAdminScriptDB(t,
		adminQueryStep{"FROM merchant_accounts", []string{"id"}, [][]driver.Value{{uint64(3)}}},
		adminQueryStep{"FROM orders WHERE id=?", []string{
			"id", "order_no", "state", "pickup_date", "pickup_time", "meal_period", "pickup_point", "pickup_number",
			"payable", "subtotal", "discount", "discount_rate", "contact_name", "phone", "transaction_id", "paid_at", "materialized_at",
		}, [][]driver.Value{{uint64(41), "ORDER-41", "PREPARING", "2026-08-25", "12:30", "lunch", "北门", "0012", uint64(1800), uint64(2000), uint64(200), uint64(90), "顾客", "+8613800000001", "TX-41", paidAt, paidAt}}},
		adminQueryStep{"FROM order_items WHERE order_id IN (?)", []string{"order_id", "product_id", "name", "quantity", "line_total"}, [][]driver.Value{{uint64(41), uint64(9), "红烧肉", uint64(2), uint64(1800)}}},
	)
	application := NewMySQLApplication(db, nil)
	order, err := application.GetOrder(context.Background(), 7, 41)
	if err != nil || order.ID != 41 || order.State != "制作中" || order.MaskedPhone != "138****0001" || len(order.Items) != 1 || order.Items[0].ProductID != 9 || len(order.AvailableActions) != 2 {
		t.Fatalf("GetOrder() = %#v, %v", order, err)
	}
	script.assertDone(t)
}

func TestGetOrderDoesNotReadOrderForNonOwner(t *testing.T) {
	db, script := openAdminScriptDB(t, adminQueryStep{"FROM merchant_accounts", []string{"id"}, nil})
	_, err := NewMySQLApplication(db, nil).GetOrder(context.Background(), 7, 41)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("GetOrder() error = %v, want forbidden", err)
	}
	script.assertDone(t)
}

type adminQueryStep struct {
	contains string
	columns  []string
	rows     [][]driver.Value
}

type adminScript struct {
	mu    sync.Mutex
	steps []adminQueryStep
}

func (script *adminScript) query(query string) (driver.Rows, error) {
	script.mu.Lock()
	defer script.mu.Unlock()
	if len(script.steps) == 0 {
		return nil, fmt.Errorf("unexpected query: %s", query)
	}
	step := script.steps[0]
	if !strings.Contains(strings.Join(strings.Fields(query), " "), step.contains) {
		return nil, fmt.Errorf("query %q does not contain %q", query, step.contains)
	}
	script.steps = script.steps[1:]
	return &adminRows{columns: step.columns, rows: step.rows}, nil
}

func (script *adminScript) assertDone(t *testing.T) {
	t.Helper()
	script.mu.Lock()
	defer script.mu.Unlock()
	if len(script.steps) != 0 {
		t.Fatalf("unconsumed SQL steps = %d", len(script.steps))
	}
}

type adminDriver struct{ script *adminScript }

func (value adminDriver) Open(string) (driver.Conn, error) {
	return &adminConn{script: value.script}, nil
}

type adminConn struct{ script *adminScript }

func (*adminConn) Prepare(string) (driver.Stmt, error) { return nil, driver.ErrSkip }
func (*adminConn) Close() error                        { return nil }
func (*adminConn) Begin() (driver.Tx, error)           { return nil, driver.ErrSkip }
func (conn *adminConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	return conn.script.query(query)
}

type adminRows struct {
	columns []string
	rows    [][]driver.Value
	index   int
}

func (rows *adminRows) Columns() []string { return rows.columns }
func (*adminRows) Close() error           { return nil }
func (rows *adminRows) Next(dest []driver.Value) error {
	if rows.index >= len(rows.rows) {
		return io.EOF
	}
	copy(dest, rows.rows[rows.index])
	rows.index++
	return nil
}

func openAdminScriptDB(t *testing.T, steps ...adminQueryStep) (*sql.DB, *adminScript) {
	t.Helper()
	script := &adminScript{steps: append([]adminQueryStep(nil), steps...)}
	name := fmt.Sprintf("admin-report-script-%d", adminScriptSequence.Add(1))
	sql.Register(name, adminDriver{script: script})
	db, err := sql.Open(name, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, script
}
