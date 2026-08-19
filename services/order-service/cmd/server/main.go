package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/skalgutkar/order-processing-platform/services/order-service/internal/api"
	"github.com/skalgutkar/order-processing-platform/services/order-service/internal/config"
	"github.com/skalgutkar/order-processing-platform/services/order-service/internal/events"
	"github.com/skalgutkar/order-processing-platform/services/order-service/internal/repository"
)

// main is pure process wiring: load config, dial dependencies built by the
// helpers in bootstrap.go, start the HTTP server, and block until a signal
// asks for a graceful shutdown. There's no branching left to assert on here
// once connectPostgres/newSNSClient (the parts with real retry/config logic)
// are factored out — what remains is exercised end-to-end by `docker compose
// up`, not a unit test. See the codecov.yml `ignore:` entry for this file.
func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := connectPostgres(ctx, cfg.DatabaseURL, logger)
	if err != nil {
		logger.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	snsClient, err := newSNSClient(ctx, cfg)
	if err != nil {
		logger.Error("failed to configure sns client", "error", err)
		os.Exit(1)
	}

	repo := repository.NewOrderRepository(pool)
	publisher := events.NewSNSPublisher(snsClient, cfg.SNSTopicARN)
	handler := api.NewHandler(repo, publisher, logger)

	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           handler.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("order-service listening", "port", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
}
