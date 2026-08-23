package storefront

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"testing"
)

func TestRepositoryReadsV27ObjectKeyAndFlavorFacts(t *testing.T) {
	db := openStorefrontScriptedDB(t, func(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
		normalized := strings.Join(strings.Fields(query), " ")
		for _, fragment := range []string{"launch_image_object_key", "flavor_options_json", "FROM storefront_settings", "WHERE id = 1"} {
			if !strings.Contains(normalized, fragment) {
				t.Fatalf("v27 query missing %q: %s", fragment, normalized)
			}
		}
		if len(args) != 0 || strings.Contains(normalized, "launch_png_url") {
			t.Fatalf("v27 query uses legacy fact or args: %s %#v", normalized, args)
		}
		return &storefrontRows{values: [][]driver.Value{{
			"绥安食品", "地址", "取餐点", "公告", "open", "launch/a.png", .5, .4, .8, 1.5, []byte(`["少盐","不辣"]`),
		}}}, nil
	})
	got, err := NewRepository(db).Get(context.Background())
	if err != nil || got.LaunchLayer == nil || got.LaunchLayer.ImageObjectKey != "launch/a.png" || len(got.Flavors) != 2 {
		t.Fatalf("Get() = %#v, %v", got, err)
	}
}

func TestRepositoryRejectsMalformedFlavorFacts(t *testing.T) {
	for _, raw := range [][]byte{[]byte(`null`), []byte(`["少盐","少盐"]`), []byte(`[""]`), []byte(`{"not":"array"}`)} {
		db := openStorefrontScriptedDB(t, func(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
			return &storefrontRows{values: [][]driver.Value{{"店", "地址", "取餐点", "", "open", nil, nil, nil, nil, nil, raw}}}, nil
		})
		if _, err := NewRepository(db).Get(context.Background()); err == nil {
			t.Fatalf("Get accepted flavor JSON %s", raw)
		}
	}
}

type storefrontResponder func(context.Context, string, []driver.NamedValue) (driver.Rows, error)
type storefrontDriver struct{ responder storefrontResponder }
type storefrontConnection struct{ responder storefrontResponder }
type storefrontRows struct {
	values [][]driver.Value
	index  int
}

func (value storefrontDriver) Open(string) (driver.Conn, error) {
	return &storefrontConnection{responder: value.responder}, nil
}
func (connection *storefrontConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare not supported")
}
func (connection *storefrontConnection) Close() error { return nil }
func (connection *storefrontConnection) Begin() (driver.Tx, error) {
	return nil, errors.New("begin not supported")
}
func (connection *storefrontConnection) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	return connection.responder(ctx, query, args)
}
func (rows *storefrontRows) Columns() []string { return make([]string, 11) }
func (rows *storefrontRows) Close() error      { return nil }
func (rows *storefrontRows) Next(destination []driver.Value) error {
	if rows.index >= len(rows.values) {
		return io.EOF
	}
	copy(destination, rows.values[rows.index])
	rows.index++
	return nil
}

var storefrontScriptedSequence atomic.Uint64

func openStorefrontScriptedDB(t *testing.T, responder storefrontResponder) *sql.DB {
	t.Helper()
	driverValue := storefrontDriver{responder: responder}
	name := fmt.Sprintf("storefront-v27-%d", storefrontScriptedSequence.Add(1))
	sql.Register(name, driverValue)
	db := sql.OpenDB(storefrontConnector{driver: driverValue})
	t.Cleanup(func() { _ = db.Close() })
	return db
}

type storefrontConnector struct{ driver storefrontDriver }

func (connector storefrontConnector) Connect(context.Context) (driver.Conn, error) {
	return connector.driver.Open("")
}
func (connector storefrontConnector) Driver() driver.Driver { return connector.driver }
