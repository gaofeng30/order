package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gaofeng30/order/services/api/internal/adminreport"
	"github.com/gaofeng30/order/services/api/internal/audit"
	"github.com/gaofeng30/order/services/api/internal/catalog"
	"github.com/gaofeng30/order/services/api/internal/httpapi"
	"github.com/gaofeng30/order/services/api/internal/identity"
	"github.com/gaofeng30/order/services/api/internal/importbatch"
	"github.com/gaofeng30/order/services/api/internal/merchantidentity"
	"github.com/gaofeng30/order/services/api/internal/objectstore"
	"github.com/gaofeng30/order/services/api/internal/staffdiscount"
	"github.com/gaofeng30/order/services/api/internal/storefront"
	"net/http/httptest"
)

// TestPCRemainingUI1Server owns one random fresh-v44 schema and exposes only a
// private loopback API for the exact Chrome candidate selector.
func TestPCRemainingUI1Server(t *testing.T) {
	if os.Getenv("ORDER_PC_REMAINING_SERVE") != "YES" {
		t.Skip("PC remaining UI1 server not requested")
	}
	infoFile := os.Getenv("ORDER_PC_REMAINING_INFO_FILE")
	stopFile := os.Getenv("ORDER_PC_REMAINING_STOP_FILE")
	if !strings.HasPrefix(infoFile, "/private/tmp/order-pc-remaining-") || !strings.HasPrefix(stopFile, "/private/tmp/order-pc-remaining-") {
		t.Fatal("PC remaining harness paths must be exact private/tmp files")
	}

	db := acceptanceFreshMySQL(t)
	acceptanceSeedSharedFacts(t, db)
	identityRepository := identity.NewRepository(db)
	loginProvider := pcRemainingLoginProvider{}
	phoneProvider := acceptancePhoneProvider{phones: map[string]string{"pc-remaining-owner-openid": acceptanceOwnerPhone}}
	sessions := identity.NewService(loginProvider, identityRepository)
	phoneService := identity.NewPhoneService(phoneProvider, identityRepository)
	merchantRepository := merchantidentity.NewRepository(db)
	merchantService := merchantidentity.NewService(merchantRepository, phoneProvider)
	merchantAdmin := merchantidentity.NewMySQLAdminApplication(db, merchantService)

	objectRoot := t.TempDir()
	fileAdapter, err := objectstore.NewFileAdapter(objectRoot, "/api/v1/objects")
	if err != nil {
		t.Fatal("compose private file object adapter", err)
	}
	objectService := objectstore.NewService(fileAdapter)
	objectRoutes, err := newLocalObjectRoutes(objectRoot)
	if err != nil {
		t.Fatal("compose private object routes", err)
	}
	imports := importbatch.NewHandler(importbatch.NewMySQLApplication(db))
	adminFeatures := []adminGroupRegistrar{
		catalog.NewAdminHandler(catalog.NewMySQLAdminApplication(db, objectService)),
		storefront.NewAdminHandler(storefront.NewMySQLAdminApplication(db)),
		staffdiscount.NewHandler(staffdiscount.NewMySQLApplication(db)),
		imports,
		adminreport.NewHandler(adminreport.NewMySQLApplication(db, nil)),
		audit.NewHandler(audit.NewMySQLSearcher(db)),
	}
	router := httpapi.NewRouter(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		func(context.Context) httpapi.ReadinessResult { return httpapi.ReadinessResult{Ready: true} },
		identity.NewHandler(sessions), identity.NewPhoneHandler(sessions, phoneService),
		merchantidentity.NewHandler(sessions, merchantService),
		storefront.NewHandler(storefront.NewRepository(db), objectService),
		newAdminRoutes(sessions, merchantAdmin, merchantidentity.NewAdminHandler(merchantAdmin), adminFeatures, []adminGroupRegistrar{objectstore.NewHandler(objectService)}, imports),
		objectRoutes,
	)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	var schema string
	if err := db.QueryRowContext(t.Context(), `SELECT DATABASE()`).Scan(&schema); err != nil || !strings.HasPrefix(schema, "order_acceptance_") {
		t.Fatal("read private UI1 schema")
	}
	info := fmt.Sprintf("{\"origin\":%q,\"schema\":%q,\"object_root\":%q,\"login_code\":\"pc-remaining-login\",\"phone_code\":\"pc-remaining-phone\"}\n", server.URL, schema, objectRoot)
	if err := os.WriteFile(infoFile, []byte(info), 0o600); err != nil {
		t.Fatal("publish private UI1 harness info", err)
	}
	t.Cleanup(func() { _ = os.Remove(infoFile) })

	deadline := time.NewTimer(10 * time.Minute)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer deadline.Stop()
	defer ticker.Stop()
	for {
		select {
		case <-deadline.C:
			t.Fatal("PC remaining UI1 harness timed out")
		case <-ticker.C:
			if _, err := os.Stat(stopFile); err == nil {
				_ = os.Remove(stopFile)
				return
			}
		}
	}
}

type pcRemainingLoginProvider struct{}

func (pcRemainingLoginProvider) Exchange(_ context.Context, code string) (string, error) {
	if strings.TrimSpace(code) == "" {
		return "", fmt.Errorf("empty private UI1 login code")
	}
	return "pc-remaining-owner-openid", nil
}
