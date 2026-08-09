package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/gaofeng30/order/services/api/internal/app"
	"github.com/gaofeng30/order/services/api/internal/config"
	"github.com/gaofeng30/order/services/api/internal/httpapi"
)

func main() {
	os.Exit(run())
}

func run() int {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration error", "error", err)
		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, cfg, httpapi.NewRouter(logger), logger, net.Listen); err != nil {
		logger.Error("order-api stopped with error", "error", err)
		return 1
	}
	return 0
}
