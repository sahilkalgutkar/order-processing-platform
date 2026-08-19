package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/skalgutkar/order-processing-platform/services/notification-service/internal/config"
	"github.com/skalgutkar/order-processing-platform/services/notification-service/internal/consumer"
	"github.com/skalgutkar/order-processing-platform/services/notification-service/internal/notifier"
)

// main is pure process wiring: load config, dial the dependency built by
// newSQSClient in bootstrap.go, start the consumer and HTTP server, and
// block until a signal asks for a graceful shutdown. There's no branching
// left to assert on here once newSQSClient (the part with real config
// logic) is factored out — what remains is exercised end-to-end by
// `docker compose up`, not a unit test. See the codecov.yml `ignore:` entry
// for this file.
func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	sqsClient, err := newSQSClient(ctx, cfg)
	if err != nil {
		logger.Error("failed to configure sqs client", "error", err)
		os.Exit(1)
	}

	n := notifier.NewLogNotifier(logger)
	c := consumer.New(sqsClient, cfg.SQSQueueURL, n, logger)

	go c.Run(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.Handle("GET /metrics", promhttp.Handler())

	srv := &http.Server{Addr: ":" + cfg.HTTPPort, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		logger.Info("notification-service http listening", "port", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http server error", "error", err)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
