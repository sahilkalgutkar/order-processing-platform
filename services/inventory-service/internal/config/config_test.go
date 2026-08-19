package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/skalgutkar/order-processing-platform/services/inventory-service/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	cfg := config.Load()

	require.Equal(t, "8081", cfg.HTTPPort)
	require.Equal(t, "mongodb://localhost:27017", cfg.MongoURI)
	require.Equal(t, "inventory", cfg.MongoDBName)
	require.Equal(t, "", cfg.SQSQueueURL)
	require.Equal(t, "", cfg.AWSEndpoint)
	require.Equal(t, "us-east-1", cfg.AWSRegion)
}

func TestLoad_EnvOverrides(t *testing.T) {
	t.Setenv("HTTP_PORT", "18081")
	t.Setenv("MONGO_URI", "mongodb://mongo:27017")
	t.Setenv("MONGO_DB_NAME", "test_inventory")
	t.Setenv("SQS_QUEUE_URL", "http://localstack:4566/000000000000/inventory-queue")
	t.Setenv("AWS_ENDPOINT_URL", "http://localstack:4566")
	t.Setenv("AWS_REGION", "us-west-2")

	cfg := config.Load()

	require.Equal(t, "18081", cfg.HTTPPort)
	require.Equal(t, "mongodb://mongo:27017", cfg.MongoURI)
	require.Equal(t, "test_inventory", cfg.MongoDBName)
	require.Equal(t, "http://localstack:4566/000000000000/inventory-queue", cfg.SQSQueueURL)
	require.Equal(t, "http://localstack:4566", cfg.AWSEndpoint)
	require.Equal(t, "us-west-2", cfg.AWSRegion)
}
