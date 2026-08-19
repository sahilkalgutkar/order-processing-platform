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

	"github.com/skalgutkar/order-processing-platform/services/inventory-service/internal/config"
	"github.com/skalgutkar/order-processing-platform/services/inventory-service/internal/consumer"
	"github.com/skalgutkar/order-processing-platform/services/inventory-service/internal/repository"
)

// main is pure process wiring: load config, dial dependencies built by the
// helpers in bootstrap.go, start the consumer and HTTP server, and block
// until a signal asks for a graceful shutdown. There's no branching left to
// assert on here once connectMongo/newSQSClient (the parts with real
// retry/config logic) are factored out — what remains is exercised
// end-to-end by `docker compose up`, not a unit test. See the codecov.yml
// `ignore:` entry for this file.
func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	mongoClient, err := connectMongo(ctx, cfg.MongoURI, logger)
	if err != nil {
		logger.Error("failed to connect to mongo", "error", err)
		os.Exit(1)
	}
	defer mongoClient.Disconnect(context.Background())

	sqsClient, err := newSQSClient(ctx, cfg)
	if err != nil {
		logger.Error("failed to configure sqs client", "error", err)
		os.Exit(1)
	}

	repo := repository.NewReservationRepository(mongoClient.Database(cfg.MongoDBName))
	stock := consumer.FixedStock{Default: 100} // TODO: back with a real stock table
	c := consumer.New(sqsClient, cfg.SQSQueueURL, repo, stock, logger)

	go c.Run(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.Handle("GET /metrics", promhttp.Handler())

	srv := &http.Server{Addr: ":" + cfg.HTTPPort, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		logger.Info("inventory-service http listening", "port", cfg.HTTPPort)
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
