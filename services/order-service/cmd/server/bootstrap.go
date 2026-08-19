package main

import (
	"context"
	"log/slog"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/skalgutkar/order-processing-platform/services/order-service/internal/config"
)

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
