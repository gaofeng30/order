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

func TestMenuRepositoryUsesTwoFixedQueriesAndFoldsSoldOutRows(t *testing.T) {
	queryCount := 0
	db := openMenuScriptedDB(t, func(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
		queryCount++
		normalized := strings.Join(strings.Fields(query), " ")
		switch queryCount {
		case 1:
			if len(args) != 0 || !strings.Contains(normalized, "SELECT code, cutoff_time, pickup_start_time, pickup_end_time, interval_minutes FROM meal_periods ORDER BY code") {
				t.Fatalf("configuration query = %q args=%v", normalized, args)
			}
			return &menuScriptedRows{
				columns: []string{"code", "cutoff_time", "pickup_start_time", "pickup_end_time", "interval_minutes"},
				values: [][]driver.Value{
					{"lunch", "10:45:00", "11:00:00", "12:00:00", int64(20)},
					{"dinner", "17:00:00", "17:00:00", "19:00:00", int64(30)},
				},
			}, nil
		case 2:
			for _, fragment := range []string{
				"FROM categories AS c INNER JOIN products AS p",
				"p.is_listed = TRUE",
				"p.meal_period IN ('all', ?)",
				"LEFT JOIN product_sold_out_dates AS s ON s.product_id = p.id AND s.service_date = ?",
				"WHERE c.is_active = TRUE",
				"ORDER BY c.sort_order ASC, c.id ASC, p.sort_order ASC, p.id ASC",
			} {
				if !strings.Contains(normalized, fragment) {
					t.Fatalf("menu query missing %q: %s", fragment, normalized)
				}
			}
			if len(args) != 2 || args[0].Value != "lunch" || args[1].Value != "2026-08-20" {
				t.Fatalf("menu args = %#v", args)
			}
			return &menuScriptedRows{
				columns: []string{"category_id", "category_name", "product_id", "product_category_id", "product_name", "description", "specification", "price_cents", "sold_out"},
				values: [][]driver.Value{
					{uint64(5), "Meals", uint64(10), uint64(5), "Rice", "", "Large", uint64(1250), false},
					{uint64(5), "Meals", uint64(11), uint64(5), "Soup", "Warm", "", uint64(300), true},
				},
			}, nil
		default:
			t.Fatalf("unexpected query %d: %s", queryCount, normalized)
			return nil, errors.New("unexpected query")
		}
	})

	repository := NewRepository(db)
	periods, err := repository.MealPeriods(context.Background())
	wantPeriods := []MealPeriodRecord{
		{Code: "lunch", CutoffTime: "10:45:00", PickupStartTime: "11:00:00", PickupEndTime: "12:00:00", IntervalMinutes: 20},
		{Code: "dinner", CutoffTime: "17:00:00", PickupStartTime: "17:00:00", PickupEndTime: "19:00:00", IntervalMinutes: 30},
	}
	if err != nil || !reflect.DeepEqual(periods, wantPeriods) {
		t.Fatalf("MealPeriods() = %#v, %v", periods, err)
	}

	categories, err := repository.List(context.Background(), "2026-08-20", MealLunch)
	wantCategories := []Category{{ID: 5, Name: "Meals", Products: []Product{
		{ID: 10, CategoryID: 5, Name: "Rice", Description: "", Specification: "Large", PriceCents: 1250, SoldOut: false},
		{ID: 11, CategoryID: 5, Name: "Soup", Description: "Warm", Specification: "", PriceCents: 300, SoldOut: true},
	}}}
	if err != nil || !reflect.DeepEqual(categories, wantCategories) || queryCount != 2 {
		t.Fatalf("List() = %#v, %v queries=%d", categories, err, queryCount)
	}
}

func TestMenuRepositoryReturnsNonNilEmptySlicesAndFailsClosed(t *testing.T) {
	emptyDB := openMenuScriptedDB(t, func(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
		columns := []string{"code", "cutoff_time", "pickup_start_time", "pickup_end_time", "interval_minutes"}
		if strings.Contains(query, "FROM categories") {
			columns = []string{"category_id", "category_name", "product_id", "product_category_id", "product_name", "description", "specification", "price_cents", "sold_out"}
		}
		return &menuScriptedRows{columns: columns}, nil
	})
	repository := NewRepository(emptyDB)
	if got, err := repository.MealPeriods(context.Background()); err != nil || got == nil || len(got) != 0 {
		t.Fatalf("empty MealPeriods() = %#v, %v", got, err)
	}
	if got, err := repository.List(context.Background(), "2026-08-21", MealDinner); err != nil || got == nil || len(got) != 0 {
		t.Fatalf("empty List() = %#v, %v", got, err)
	}

	for _, test := range []struct {
		name string
		rows *menuScriptedRows
	}{
		{name: "configuration-scan", rows: &menuScriptedRows{columns: make([]string, 5), values: [][]driver.Value{{"lunch", "bad", "11:00:00", "12:00:00", "not-number"}}}},
		{name: "product-scan", rows: &menuScriptedRows{columns: make([]string, 9), values: [][]driver.Value{{"bad", "Meals", uint64(1), uint64(1), "Rice", "", "", uint64(100), false}}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			db := openMenuScriptedDB(t, func(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
				if test.name == "product-scan" && !strings.Contains(query, "FROM categories") {
					return &menuScriptedRows{columns: make([]string, 5)}, nil
				}
				return test.rows, nil
			})
			repository := NewRepository(db)
			var err error
			if test.name == "configuration-scan" {
				_, err = repository.MealPeriods(context.Background())
			} else {
				_, err = repository.List(context.Background(), "2026-08-20", MealLunch)
			}
			if err == nil {
				t.Fatal("repository accepted invalid row")
			}
		})
	}
}

type menuScriptedResponder func(context.Context, string, []driver.NamedValue) (driver.Rows, error)

type menuScriptedDriver struct{ responder menuScriptedResponder }

func (value menuScriptedDriver) Open(string) (driver.Conn, error) {
	return &menuScriptedConnection{responder: value.responder}, nil
}

type menuScriptedConnection struct{ responder menuScriptedResponder }

func (connection *menuScriptedConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare not supported")
}
func (connection *menuScriptedConnection) Close() error { return nil }
func (connection *menuScriptedConnection) Begin() (driver.Tx, error) {
	return nil, errors.New("begin not supported")
}
func (connection *menuScriptedConnection) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	return connection.responder(ctx, query, args)
}

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
	name := fmt.Sprintf("menu-scripted-%d", menuScriptedSequence.Add(1))
	driverValue := menuScriptedDriver{responder: responder}
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
