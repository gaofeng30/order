package menu

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
)

func TestMenuRepositoryReadsOneTransactionalCurrentSnapshot(t *testing.T) {
	queryCount := 0
	db := openMenuScriptedDB(t, func(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
		queryCount++
		normalized := strings.Join(strings.Fields(query), " ")
		switch queryCount {
		case 1:
			if !strings.Contains(normalized, "FROM storefront_settings AS settings LEFT JOIN service_dates AS dates") || len(args) != 1 || args[0].Value != "2026-08-25" {
				t.Fatalf("facts query = %s args=%#v", normalized, args)
			}
			return &menuScriptedRows{columns: make([]string, 3), values: [][]driver.Value{{"open", "2026-08-25", true}}}, nil
		case 2:
			if !strings.Contains(normalized, "FROM meal_periods") {
				t.Fatalf("meal query = %s", normalized)
			}
			return &menuScriptedRows{columns: make([]string, 5), values: [][]driver.Value{
				{"lunch", "11:30:00", "11:30:00", "13:30:00", int64(30)},
				{"dinner", "17:00:00", "17:00:00", "19:00:00", int64(30)},
			}}, nil
		case 3:
			for _, fragment := range []string{"p.meal_period", "p.images_json", "p.is_listed", "product_sold_out_dates", "sold.service_date = ?"} {
				if !strings.Contains(normalized, fragment) {
					t.Fatalf("menu query missing %q: %s", fragment, normalized)
				}
			}
			return &menuScriptedRows{columns: make([]string, 12), values: [][]driver.Value{
				{uint64(2), "晚餐", uint64(7), uint64(2), "红烧肉", "慢炖", "份", "dinner", []byte(`[{"object_key":"p/1.png"}]`), true, uint64(1800), false},
			}}, nil
		default:
			t.Fatalf("unexpected query %d", queryCount)
			return nil, errors.New("unexpected query")
		}
	})

	got, err := NewRepository(db).ReadMenu(context.Background(), "2026-08-25")
	want := MenuSnapshot{
		BusinessStatus: "open", ServiceDatePresent: true, ServiceDateOpen: true,
		MealPeriods: defaultMealPeriodRecords(),
		Categories: []Category{{ID: 2, Name: "晚餐", Products: []Product{{
			ID: 7, CategoryID: 2, Name: "红烧肉", Description: "慢炖", Specification: "份", MealPeriod: "dinner",
			ImageObjectKeys: []string{"p/1.png"}, Listed: true, OriginalUnitPriceCents: 1800,
		}}}},
	}
	if err != nil || !reflect.DeepEqual(got, want) || queryCount != 3 {
		t.Fatalf("ReadMenu() = %#v, %v queries=%d", got, err, queryCount)
	}
}

func TestMenuRepositoryReadsMissingDateAsClosed(t *testing.T) {
	queryCount := 0
	db := openMenuScriptedDB(t, func(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
		queryCount++
		switch queryCount {
		case 1:
			return &menuScriptedRows{columns: make([]string, 3), values: [][]driver.Value{{"open", nil, nil}}}, nil
		case 2:
			return &menuScriptedRows{columns: make([]string, 5), values: [][]driver.Value{
				{"lunch", "11:30:00", "11:30:00", "13:30:00", int64(30)},
				{"dinner", "17:00:00", "17:00:00", "19:00:00", int64(30)},
			}}, nil
		default:
			return &menuScriptedRows{columns: make([]string, 12)}, nil
		}
	})
	got, err := NewRepository(db).ReadMenu(context.Background(), "2026-08-25")
	if err != nil || got.ServiceDatePresent || got.ServiceDateOpen {
		t.Fatalf("missing date = %#v, %v", got, err)
	}
}

func TestMenuRepositoryFailsClosedForMalformedImages(t *testing.T) {
	queryCount := 0
	db := openMenuScriptedDB(t, func(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
		queryCount++
		switch queryCount {
		case 1:
			return &menuScriptedRows{columns: make([]string, 3), values: [][]driver.Value{{"open", "2026-08-25", true}}}, nil
		case 2:
			return &menuScriptedRows{columns: make([]string, 5), values: [][]driver.Value{
				{"lunch", "11:30:00", "11:30:00", "13:30:00", int64(30)},
				{"dinner", "17:00:00", "17:00:00", "19:00:00", int64(30)},
			}}, nil
		default:
			return &menuScriptedRows{columns: make([]string, 12), values: [][]driver.Value{{
				uint64(2), "晚餐", uint64(7), uint64(2), "红烧肉", "", "份", "dinner", []byte(`[{"object_key":"same"},{"object_key":"same"}]`), true, uint64(1800), false,
			}}}, nil
		}
	})
	if _, err := NewRepository(db).ReadMenu(context.Background(), "2026-08-25"); err == nil {
		t.Fatal("ReadMenu accepted malformed images")
	}
}

type menuScriptedResponder func(context.Context, string, []driver.NamedValue) (driver.Rows, error)
type menuScriptedDriver struct{ responder menuScriptedResponder }
type menuScriptedConnection struct{ responder menuScriptedResponder }
type menuScriptedTx struct{}

func (value menuScriptedDriver) Open(string) (driver.Conn, error) {
	return &menuScriptedConnection{responder: value.responder}, nil
}
func (connection *menuScriptedConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare not supported")
}
func (connection *menuScriptedConnection) Close() error              { return nil }
func (connection *menuScriptedConnection) Begin() (driver.Tx, error) { return menuScriptedTx{}, nil }
func (connection *menuScriptedConnection) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return menuScriptedTx{}, nil
}
func (connection *menuScriptedConnection) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	return connection.responder(ctx, query, args)
}
func (menuScriptedTx) Commit() error   { return nil }
func (menuScriptedTx) Rollback() error { return nil }

type menuScriptedRows struct {
	columns []string
	values  [][]driver.Value
	index   int
}

func (rows *menuScriptedRows) Columns() []string { return rows.columns }
func (rows *menuScriptedRows) Close() error      { return nil }
func (rows *menuScriptedRows) Next(destination []driver.Value) error {
	if rows.index >= len(rows.values) {
		return io.EOF
	}
	copy(destination, rows.values[rows.index])
	rows.index++
	return nil
}

var menuScriptedSequence atomic.Uint64

func openMenuScriptedDB(t *testing.T, responder menuScriptedResponder) *sql.DB {
	t.Helper()
	driverValue := menuScriptedDriver{responder: responder}
	name := fmt.Sprintf("menu-frozen-%d", menuScriptedSequence.Add(1))
	sql.Register(name, driverValue)
	db := sql.OpenDB(menuScriptedConnector{driver: driverValue})
	t.Cleanup(func() { _ = db.Close() })
	return db
}

type menuScriptedConnector struct{ driver menuScriptedDriver }

func (connector menuScriptedConnector) Connect(context.Context) (driver.Conn, error) {
	return connector.driver.Open("")
}
func (connector menuScriptedConnector) Driver() driver.Driver { return connector.driver }
