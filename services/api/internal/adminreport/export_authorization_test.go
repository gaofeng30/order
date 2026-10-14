package adminreport

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
)

// The driver revokes OWNER immediately after the first export page.
// It exercises the real database/sql adapter, without external services.
type revokingDB struct{ authReads, pages int }

func (d *revokingDB) Connect(context.Context) (driver.Conn, error) { return d, nil }
func (d *revokingDB) Driver() driver.Driver                        { return d }
func (d *revokingDB) Open(string) (driver.Conn, error)             { return d, nil }
func (d *revokingDB) Prepare(string) (driver.Stmt, error) {
	return nil, fmt.Errorf("unexpected prepare")
}
func (d *revokingDB) Close() error              { return nil }
func (d *revokingDB) Begin() (driver.Tx, error) { return nil, fmt.Errorf("unexpected transaction") }
func (d *revokingDB) QueryContext(_ context.Context, q string, _ []driver.NamedValue) (driver.Rows, error) {
	if strings.Contains(q, "FROM merchant_accounts") {
		d.authReads++
		rows := [][]driver.Value{}
		if d.pages == 0 {
			rows = append(rows, []driver.Value{int64(1)})
		}
		return &exportRows{[]string{"id"}, rows}, nil
	}
	if !strings.Contains(q, "FROM orders") {
		return nil, fmt.Errorf("unexpected query")
	}
	d.pages++
	rows := [][]driver.Value{}
	if d.pages == 1 {
		for i := 1; i <= 100; i++ {
			rows = append(rows, []driver.Value{int64(i), fmt.Sprint(i), "0001", time.Unix(1791878400, 0), "tx", int64(1000)})
		}
	}
	return &exportRows{[]string{"id", "order_no", "pickup_number", "paid_at", "transaction_id", "payable_cents"}, rows}, nil
}

type exportRows struct {
	columns []string
	rows    [][]driver.Value
}

func (r *exportRows) Columns() []string { return r.columns }
func (r *exportRows) Close() error      { return nil }
func (r *exportRows) Next(dest []driver.Value) error {
	if len(r.rows) == 0 {
		return io.EOF
	}
	copy(dest, r.rows[0])
	r.rows = r.rows[1:]
	return nil
}
func TestNativeExportStopsAfterOwnerRevocation(t *testing.T) {
	driver := &revokingDB{}
	db := sql.OpenDB(driver)
	defer db.Close()
	app := NewPostgreSQLApplication(db, nil)
	result, err := app.ExportCSV(context.Background(), 7, BillingRange{"2026-10-13", "2026-10-13"})
	if result != nil {
		result.Close()
	}
	if err != ErrForbidden || result != nil || driver.authReads != 2 || driver.pages != 1 {
		t.Fatalf("revoked owner got export: err=%v auth=%d pages=%d", err, driver.authReads, driver.pages)
	}
}
