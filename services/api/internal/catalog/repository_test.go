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

func TestBrowseReadsV24ImagesInStoredOrder(t *testing.T) {
	db := openCatalogScriptedDB(t, func(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
		normalized := strings.Join(strings.Fields(query), " ")
		for _, fragment := range []string{"p.meal_period", "p.images_json", "p.is_listed", "p.price_cents", "p.is_listed = TRUE"} {
			if !strings.Contains(normalized, fragment) {
				t.Fatalf("browse query missing %q: %s", fragment, normalized)
			}
		}
		if len(args) != 0 {
			t.Fatalf("browse args = %v", args)
		}
		return &catalogRows{columns: make([]string, 11), values: [][]driver.Value{
			{uint64(2), "餐品", uint64(7), uint64(2), "无图", "", "份", "all", []byte(`[]`), true, uint64(200)},
			{uint64(2), "餐品", uint64(8), uint64(2), "一图", "", "份", "lunch", []byte(`[{"object_key":"p/1.png"}]`), true, uint64(300)},
			{uint64(2), "餐品", uint64(9), uint64(2), "三图", "", "份", "dinner", []byte(`[{"object_key":"p/a.png"},{"object_key":"p/b.png"},{"object_key":"p/c.png"}]`), true, uint64(400)},
		}}, nil
	})

	got, err := NewRepository(db).Browse(context.Background())
	if err != nil || len(got) != 1 || len(got[0].Products) != 3 {
		t.Fatalf("Browse() = %#v, %v", got, err)
	}
	wantKeys := [][]string{{}, {"p/1.png"}, {"p/a.png", "p/b.png", "p/c.png"}}
	for index, product := range got[0].Products {
		if !reflect.DeepEqual(product.ImageObjectKeys, wantKeys[index]) {
			t.Fatalf("product %d images = %#v, want %#v", index, product.ImageObjectKeys, wantKeys[index])
		}
	}
}

func TestBrowseFailsClosedForMalformedOrDuplicateImages(t *testing.T) {
	for _, raw := range [][]byte{
		[]byte(`null`), []byte(`{}`), []byte(`[{"object_key":""}]`),
		[]byte(`[{"object_key":"same"},{"object_key":"same"}]`),
		[]byte(`[{"object_key":"a","url":"not-stored"}]`),
		[]byte(`[{"object_key":"a"},{"object_key":"b"},{"object_key":"c"},{"object_key":"d"}]`),
	} {
		db := openCatalogScriptedDB(t, func(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
			return &catalogRows{columns: make([]string, 11), values: [][]driver.Value{{
				uint64(2), "餐品", uint64(7), uint64(2), "坏图", "", "份", "all", raw, true, uint64(200),
			}}}, nil
		})
		if _, err := NewRepository(db).Browse(context.Background()); err == nil {
			t.Fatalf("Browse accepted images %s", raw)
		}
	}
}

func TestDetailReadsDateStoreScheduleProductAndSoldOutInOneTransaction(t *testing.T) {
	queryCount := 0
	db := openCatalogScriptedDB(t, func(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
		queryCount++
		normalized := strings.Join(strings.Fields(query), " ")
		switch queryCount {
		case 1:
			if !strings.Contains(normalized, "service_dates") || len(args) != 1 || args[0].Value != "2026-08-25" {
				t.Fatalf("current facts query = %s args=%#v", normalized, args)
			}
			return &catalogRows{columns: make([]string, 3), values: [][]driver.Value{{"open", "2026-08-25", true}}}, nil
		case 2:
			return &catalogRows{columns: make([]string, 5), values: [][]driver.Value{
				{"lunch", "11:30:00", "11:30:00", "13:30:00", int64(30)},
				{"dinner", "17:00:00", "17:00:00", "19:00:00", int64(30)},
			}}, nil
		case 3:
			if !strings.Contains(normalized, "product_sold_out_dates") || len(args) != 2 || args[0].Value != "2026-08-25" || args[1].Value != int64(7) {
				t.Fatalf("detail product query = %s args=%#v", normalized, args)
			}
			return &catalogRows{columns: make([]string, 10), values: [][]driver.Value{{
				uint64(7), uint64(2), "红烧肉", "慢炖", "份", "dinner", []byte(`[{"object_key":"p/1.png"}]`), true, uint64(1800), true,
			}}}, nil
		default:
			t.Fatalf("unexpected query %d", queryCount)
			return nil, errors.New("unexpected query")
		}
	})
	product, facts, err := NewRepository(db).Detail(context.Background(), 7, "2026-08-25")
	if err != nil || !product.SoldOut || !facts.ServiceDatePresent || !facts.ServiceDateOpen || len(facts.MealPeriods) != 2 || queryCount != 3 {
		t.Fatalf("Detail() = %#v %#v %v queries=%d", product, facts, err, queryCount)
	}
}

type catalogResponder func(context.Context, string, []driver.NamedValue) (driver.Rows, error)
type catalogDriver struct{ responder catalogResponder }
type catalogConnection struct{ responder catalogResponder }
type catalogTx struct{}

func (value catalogDriver) Open(string) (driver.Conn, error) {
	return &catalogConnection{responder: value.responder}, nil
}
func (connection *catalogConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare not supported")
}
func (connection *catalogConnection) Close() error { return nil }
func (connection *catalogConnection) Begin() (driver.Tx, error) {
	return catalogTx{}, nil
}
func (connection *catalogConnection) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return catalogTx{}, nil
}
func (connection *catalogConnection) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	return connection.responder(ctx, query, args)
}
func (catalogTx) Commit() error   { return nil }
func (catalogTx) Rollback() error { return nil }

type catalogRows struct {
	columns []string
	values  [][]driver.Value
	index   int
}

func (rows *catalogRows) Columns() []string { return rows.columns }
func (rows *catalogRows) Close() error      { return nil }
func (rows *catalogRows) Next(destination []driver.Value) error {
	if rows.index >= len(rows.values) {
		return io.EOF
	}
	copy(destination, rows.values[rows.index])
	rows.index++
	return nil
}

var catalogScriptedSequence atomic.Uint64

func openCatalogScriptedDB(t *testing.T, responder catalogResponder) *sql.DB {
	t.Helper()
	driverValue := catalogDriver{responder: responder}
	name := fmt.Sprintf("catalog-frozen-%d", catalogScriptedSequence.Add(1))
	sql.Register(name, driverValue)
	db := sql.OpenDB(catalogConnector{driver: driverValue})
	t.Cleanup(func() { _ = db.Close() })
	return db
}

type catalogConnector struct{ driver catalogDriver }

func (connector catalogConnector) Connect(context.Context) (driver.Conn, error) {
	return connector.driver.Open("")
}
func (connector catalogConnector) Driver() driver.Driver { return connector.driver }
