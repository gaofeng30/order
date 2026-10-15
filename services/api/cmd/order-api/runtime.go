package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gaofeng30/order/services/api/internal/app"
	"github.com/gaofeng30/order/services/api/internal/config"
)

// runWithWorkers owns the API and all workers sharing its database pool.
func runWithWorkers(ctx context.Context, cfg config.Config, handler http.Handler, logger *slog.Logger, listen app.ListenFunc, workers ...func(context.Context)) error {
	workerContext, cancelWorkers := context.WithCancel(ctx)
	defer cancelWorkers()
	var active sync.WaitGroup
	for _, worker := range workers {
		active.Add(1)
		go func() { defer active.Done(); worker(workerContext) }()
	}
	result := app.Run(ctx, cfg, handler, logger, listen)
	// Listener/HTTP failures must stop workers too; main closes their shared pool
	// only after this drain completes (or reports a bounded shutdown failure).
	cancelWorkers()
	drained := make(chan struct{})
	go func() { active.Wait(); close(drained) }()
	timer := time.NewTimer(cfg.ShutdownTimeout)
	defer timer.Stop()
	select {
	case <-drained:
		logger.Info("background workers stopped")
		return result
	case <-timer.C:
		return errors.Join(result, errors.New("background worker shutdown timed out"))
	}
}
