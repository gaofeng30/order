package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gaofeng30/order/services/api/internal/app"
	"github.com/gaofeng30/order/services/api/internal/catalog"
	"github.com/gaofeng30/order/services/api/internal/config"
	"github.com/gaofeng30/order/services/api/internal/database"
	"github.com/gaofeng30/order/services/api/internal/httpapi"
	"github.com/gaofeng30/order/services/api/internal/menu"
	"github.com/gaofeng30/order/services/api/internal/migrate"
	"github.com/gaofeng30/order/services/api/migrations"
)

func main() {
	os.Exit(run())
}

func run() int {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration error", "reason", config.Reason(err))
		return 1
	}
	db, err := database.Open(cfg.Database)
	if err != nil {
		logger.Error("database configuration error", "reason", database.Reason(err))
		return 1
	}
	defer db.Close()
	migrationSet, err := migrate.Load(migrations.FS)
	if err != nil {
		logger.Error("migration assets invalid", "reason", migrate.Reason(err))
		return 1
	}
	readiness := func(ctx context.Context) httpapi.ReadinessResult {
		checkContext, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		state := migrate.Check(checkContext, db, migrationSet)
		return httpapi.ReadinessResult{Ready: state.Ready, Reason: state.Reason}
	}
	catalogHandler := catalog.NewHandler(catalog.NewRepository(db))
	menuHandler := menu.NewHandler(menu.NewRepository(db), time.Now)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, cfg, httpapi.NewRouter(logger, readiness, catalogHandler, menuHandler), logger, net.Listen); err != nil {
		logger.Error("order-api stopped with error", "error", err)
		return 1
	}
	return 0
}
