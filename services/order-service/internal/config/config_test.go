package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/skalgutkar/order-processing-platform/services/order-service/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	cfg := config.Load()

	require.Equal(t, "8080", cfg.HTTPPort)
	require.Equal(t, "postgres://postgres:postgres@localhost:5432/orders?sslmode=disable", cfg.DatabaseURL)
	require.Equal(t, "", cfg.SNSTopicARN)
	require.Equal(t, "", cfg.AWSEndpoint)
	require.Equal(t, "us-east-1", cfg.AWSRegion)
}

func TestLoad_EnvOverrides(t *testing.T) {
	t.Setenv("HTTP_PORT", "18080")
	t.Setenv("DATABASE_URL", "postgres://postgres:postgres@postgres:5432/orders?sslmode=disable")
	t.Setenv("SNS_TOPIC_ARN", "arn:aws:sns:us-east-1:000000000000:order-events")
	t.Setenv("AWS_ENDPOINT_URL", "http://localstack:4566")
	t.Setenv("AWS_REGION", "us-west-2")

	cfg := config.Load()

	require.Equal(t, "18080", cfg.HTTPPort)
	require.Equal(t, "postgres://postgres:postgres@postgres:5432/orders?sslmode=disable", cfg.DatabaseURL)
	require.Equal(t, "arn:aws:sns:us-east-1:000000000000:order-events", cfg.SNSTopicARN)
	require.Equal(t, "http://localstack:4566", cfg.AWSEndpoint)
	require.Equal(t, "us-west-2", cfg.AWSRegion)
}
