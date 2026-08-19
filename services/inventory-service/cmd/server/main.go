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
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/skalgutkar/order-processing-platform/services/inventory-service/internal/config"
	"github.com/skalgutkar/order-processing-platform/services/inventory-service/internal/consumer"
	"github.com/skalgutkar/order-processing-platform/services/inventory-service/internal/repository"
)

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
