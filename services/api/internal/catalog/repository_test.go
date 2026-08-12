package catalog

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

func TestRepositoryListUsesOneExplicitSnapshotQueryAndFoldsRows(t *testing.T) {
	var queryCount int
	db := openScriptedDB(t, func(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
		queryCount++
		if len(args) != 0 {
			t.Fatalf("list args = %v, want none", args)
		}
		normalized := strings.Join(strings.Fields(query), " ")
		for _, want := range []string{
			"SELECT c.id, c.name, p.id, p.category_id, p.name, p.description, p.specification, p.price_cents",
			"FROM categories AS c LEFT JOIN products AS p ON p.category_id = c.id AND p.is_listed = TRUE",
			"WHERE c.is_active = TRUE",
			"ORDER BY c.sort_order ASC, c.id ASC, p.sort_order ASC, p.id ASC",
		} {
			if !strings.Contains(normalized, want) {
				t.Fatalf("list query missing %q: %s", want, normalized)
			}
		}
		if strings.Contains(strings.ToUpper(normalized), "SELECT *") {
			t.Fatalf("list query uses SELECT *: %s", normalized)
		}
		return &scriptedRows{
			columns: []string{"category_id", "category_name", "product_id", "product_category_id", "product_name", "description", "specification", "price_cents"},
			values: [][]driver.Value{
				{uint64(2), "Empty", nil, nil, nil, nil, nil, nil},
				{uint64(5), "Meals", uint64(10), uint64(5), "Rice", "", "", uint64(1250)},
				{uint64(5), "Meals", uint64(11), uint64(5), "Soup", "Warm", "Large", uint64(300)},
			},
		}, nil
	})

	got, err := NewRepository(db).List(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	want := []Category{
		{ID: 2, Name: "Empty", Products: []Product{}},
		{ID: 5, Name: "Meals", Products: []Product{
			{ID: 10, CategoryID: 5, Name: "Rice", Description: "", Specification: "", PriceCents: 1250},
			{ID: 11, CategoryID: 5, Name: "Soup", Description: "Warm", Specification: "Large", PriceCents: 300},
		}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("List = %#v, want %#v", got, want)
	}
	if got == nil || got[0].Products == nil || queryCount != 1 {
		t.Fatalf("list nil/query invariant: categories_nil=%t products_nil=%t query_count=%d", got == nil, got[0].Products == nil, queryCount)
	}
}

func TestRepositoryListReturnsEmptySliceAndPropagatesFailures(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		db := openScriptedDB(t, func(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
			return &scriptedRows{columns: make([]string, 8)}, nil
		})
		got, err := NewRepository(db).List(context.Background())
		if err != nil || got == nil || len(got) != 0 {
			t.Fatalf("empty List = %#v, %v", got, err)
		}
	})

	for _, test := range []struct {
		name      string
		responder scriptedResponder
	}{
		{name: "query", responder: func(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
			return nil, errors.New("query canary")
		}},
		{name: "scan", responder: func(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
			return &scriptedRows{columns: make([]string, 8), values: [][]driver.Value{{"invalid", "Category", nil, nil, nil, nil, nil, nil}}}, nil
		}},
		{name: "rows", responder: func(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
			return &scriptedRows{columns: make([]string, 8), terminalErr: errors.New("rows canary")}, nil
		}},
		{name: "invariant", responder: func(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
			return &scriptedRows{columns: make([]string, 8), values: [][]driver.Value{{uint64(1), "Category", nil, uint64(1), "partial", nil, nil, nil}}}, nil
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			db := openScriptedDB(t, test.responder)
			if _, err := NewRepository(db).List(context.Background()); err == nil || errors.Is(err, ErrProductNotFound) {
				t.Fatalf("List error = %v, want non-not-found failure", err)
			}
		})
	}
}

func TestRepositoryGetProductUsesVisibilityJoinAndMapsNoRows(t *testing.T) {
	var queryCount int
	db := openScriptedDB(t, func(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
		queryCount++
		normalized := strings.Join(strings.Fields(query), " ")
		for _, want := range []string{
			"SELECT p.id, p.category_id, p.name, p.description, p.specification, p.price_cents",
			"FROM products AS p INNER JOIN categories AS c ON c.id = p.category_id",
			"WHERE p.id = ? AND p.is_listed = TRUE AND c.is_active = TRUE LIMIT 1",
		} {
			if !strings.Contains(normalized, want) {
				t.Fatalf("detail query missing %q: %s", want, normalized)
			}
		}
		if len(args) != 1 || args[0].Value != int64(42) {
			t.Fatalf("detail args = %#v, want 42", args)
		}
		return &scriptedRows{
			columns: []string{"id", "category_id", "name", "description", "specification", "price_cents"},
			values:  [][]driver.Value{{uint64(42), uint64(7), "Noodles", "", "Large", uint64(999)}},
		}, nil
	})

	got, err := NewRepository(db).GetProduct(context.Background(), 42)
	want := Product{ID: 42, CategoryID: 7, Name: "Noodles", Description: "", Specification: "Large", PriceCents: 999}
	if err != nil || got != want || queryCount != 1 {
		t.Fatalf("GetProduct = %#v, %v, queries=%d", got, err, queryCount)
	}

	emptyDB := openScriptedDB(t, func(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
		return &scriptedRows{columns: make([]string, 6)}, nil
	})
	if _, err := NewRepository(emptyDB).GetProduct(context.Background(), 99); !errors.Is(err, ErrProductNotFound) {
		t.Fatalf("GetProduct no rows error = %v, want ErrProductNotFound", err)
	}

	failingDB := openScriptedDB(t, func(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
		return nil, errors.New("detail query canary")
	})
	if _, err := NewRepository(failingDB).GetProduct(context.Background(), 99); err == nil || errors.Is(err, ErrProductNotFound) {
		t.Fatalf("GetProduct query error = %v", err)
	}

	invalidDB := openScriptedDB(t, func(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
		return &scriptedRows{columns: make([]string, 6), values: [][]driver.Value{{"invalid", uint64(1), "name", "", "", uint64(1)}}}, nil
	})
	if _, err := NewRepository(invalidDB).GetProduct(context.Background(), 99); err == nil || errors.Is(err, ErrProductNotFound) {
		t.Fatalf("GetProduct scan error = %v", err)
	}
}

func TestRepositoryHonorsCanceledContext(t *testing.T) {
	var queryCount int
	db := openScriptedDB(t, func(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
		queryCount++
		return &scriptedRows{columns: make([]string, 8)}, nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewRepository(db).List(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("List canceled error = %v", err)
	}
	if queryCount != 0 {
		t.Fatalf("canceled List reached driver %d times", queryCount)
	}
}

type scriptedResponder func(context.Context, string, []driver.NamedValue) (driver.Rows, error)

type scriptedDriver struct{ responder scriptedResponder }

func (value scriptedDriver) Open(string) (driver.Conn, error) {
	return &scriptedConnection{responder: value.responder}, nil
}

type scriptedConnection struct{ responder scriptedResponder }

func (connection *scriptedConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare not supported")
}
func (connection *scriptedConnection) Close() error { return nil }
func (connection *scriptedConnection) Begin() (driver.Tx, error) {
	return nil, errors.New("begin not supported")
}
func (connection *scriptedConnection) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	return connection.responder(ctx, query, args)
}

type scriptedRows struct {
	columns     []string
	values      [][]driver.Value
	index       int
	terminalErr error
	errReturned bool
}

func (rows *scriptedRows) Columns() []string { return rows.columns }
func (rows *scriptedRows) Close() error      { return nil }
func (rows *scriptedRows) Next(destination []driver.Value) error {
	if rows.index < len(rows.values) {
		copy(destination, rows.values[rows.index])
		rows.index++
		return nil
	}
	if rows.terminalErr != nil && !rows.errReturned {
		rows.errReturned = true
		return rows.terminalErr
	}
	return io.EOF
}

var scriptedDriverSequence atomic.Uint64

func openScriptedDB(t *testing.T, responder scriptedResponder) *sql.DB {
	t.Helper()
	name := fmt.Sprintf("catalog-scripted-%d", scriptedDriverSequence.Add(1))
	sql.Register(name, scriptedDriver{responder: responder})
	db := sql.OpenDB(scriptedConnector{name: name, driver: scriptedDriver{responder: responder}})
	t.Cleanup(func() { db.Close() })
	return db
}

type scriptedConnector struct {
	name   string
	driver scriptedDriver
}

func (connector scriptedConnector) Connect(context.Context) (driver.Conn, error) {
	return connector.driver.Open(connector.name)
}
func (connector scriptedConnector) Driver() driver.Driver { return connector.driver }
