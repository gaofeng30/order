package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/gaofeng30/order/services/api/internal/config"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 15 * time.Second
	writeTimeout      = 15 * time.Second
	idleTimeout       = 60 * time.Second
)

// ListenFunc allows lifecycle tests to supply a deterministic listener.
type ListenFunc func(network, address string) (net.Listener, error)

// Run listens, serves requests, and drains active requests when ctx is canceled.
func Run(
	ctx context.Context,
	cfg config.Config,
	handler http.Handler,
	logger *slog.Logger,
	listen ListenFunc,
) error {
	listener, err := listen("tcp", cfg.HTTPAddr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", cfg.HTTPAddr, err)
	}

	server := newServer(cfg.HTTPAddr, handler)
	serveResult := make(chan error, 1)
	go func() {
		err := server.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serveResult <- err
	}()

	logger.InfoContext(
		ctx,
		"http server started",
		"addr", listener.Addr().String(),
		"shutdown_timeout", cfg.ShutdownTimeout,
	)

	select {
	case err := <-serveResult:
		if err != nil {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	case <-ctx.Done():
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		_ = server.Close()
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}
	if err := <-serveResult; err != nil {
		return fmt.Errorf("serve HTTP during shutdown: %w", err)
	}

	logger.Info("http server stopped")
	return nil
}

func newServer(address string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
}
