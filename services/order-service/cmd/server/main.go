package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/skalgutkar/order-processing-platform/services/order-service/internal/api"
	"github.com/skalgutkar/order-processing-platform/services/order-service/internal/config"
	"github.com/skalgutkar/order-processing-platform/services/order-service/internal/events"
	"github.com/skalgutkar/order-processing-platform/services/order-service/internal/repository"
)

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

// connectPostgres retries briefly so `docker compose up` doesn't require a
// manual restart while Postgres is still initializing.
func connectPostgres(ctx context.Context, dsn string, logger *slog.Logger) (*pgxpool.Pool, error) {
	var pool *pgxpool.Pool
	var err error

	for attempt := 1; attempt <= 10; attempt++ {
		pool, err = pgxpool.New(ctx, dsn)
		if err == nil {
			if pingErr := pool.Ping(ctx); pingErr == nil {
				return pool, nil
			} else {
				err = pingErr
			}
		}
		logger.Warn("postgres not ready yet, retrying", "attempt", attempt, "error", err)
		time.Sleep(2 * time.Second)
	}
	return nil, err
}

func newSNSClient(ctx context.Context, cfg config.Config) (*sns.Client, error) {
	optFns := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.AWSRegion),
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, optFns...)
	if err != nil {
		return nil, err
	}

	return sns.NewFromConfig(awsCfg, func(o *sns.Options) {
		if cfg.AWSEndpoint != "" {
			o.BaseEndpoint = &cfg.AWSEndpoint
		}
	}), nil
}
