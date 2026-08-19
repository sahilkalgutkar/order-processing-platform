package main

import (
	"context"
	"log/slog"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/skalgutkar/order-processing-platform/services/inventory-service/internal/config"
)

func connectMongo(ctx context.Context, uri string, logger *slog.Logger) (*mongo.Client, error) {
	var client *mongo.Client
	var err error

	for attempt := 1; attempt <= 10; attempt++ {
		client, err = mongo.Connect(ctx, options.Client().ApplyURI(uri))
		if err == nil {
			if pingErr := client.Ping(ctx, nil); pingErr == nil {
				return client, nil
			} else {
				err = pingErr
			}
		}
		logger.Warn("mongo not ready yet, retrying", "attempt", attempt, "error", err)
		time.Sleep(2 * time.Second)
	}
	return nil, err
}

func newSQSClient(ctx context.Context, cfg config.Config) (*sqs.Client, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.AWSRegion))
	if err != nil {
		return nil, err
	}
	return sqs.NewFromConfig(awsCfg, func(o *sqs.Options) {
		if cfg.AWSEndpoint != "" {
			o.BaseEndpoint = &cfg.AWSEndpoint
		}
	}), nil
}
