package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/skalgutkar/order-processing-platform/services/notification-service/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	cfg := config.Load()

	require.Equal(t, "8082", cfg.HTTPPort)
	require.Equal(t, "", cfg.SQSQueueURL)
	require.Equal(t, "", cfg.AWSEndpoint)
	require.Equal(t, "us-east-1", cfg.AWSRegion)
}

func TestLoad_EnvOverrides(t *testing.T) {
	t.Setenv("HTTP_PORT", "18082")
	t.Setenv("SQS_QUEUE_URL", "http://localstack:4566/000000000000/notification-queue")
	t.Setenv("AWS_ENDPOINT_URL", "http://localstack:4566")
	t.Setenv("AWS_REGION", "us-west-2")

	cfg := config.Load()

	require.Equal(t, "18082", cfg.HTTPPort)
	require.Equal(t, "http://localstack:4566/000000000000/notification-queue", cfg.SQSQueueURL)
	require.Equal(t, "http://localstack:4566", cfg.AWSEndpoint)
	require.Equal(t, "us-west-2", cfg.AWSRegion)
}
