package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skalgutkar/order-processing-platform/services/notification-service/internal/config"
)

// newSQSClient builds an AWS SDK client from a Config. It never dials the
// network itself (that happens lazily on the first API call), so it's
// exercised here as a plain unit test.
func TestNewSQSClient_UsesConfiguredRegion(t *testing.T) {
	cfg := config.Config{AWSRegion: "eu-west-1"}

	client, err := newSQSClient(context.Background(), cfg)

	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, "eu-west-1", client.Options().Region)
}

// AWSEndpoint is how docker-compose points the service at LocalStack
// instead of real AWS; confirm the override actually takes effect.
func TestNewSQSClient_HonorsCustomEndpoint(t *testing.T) {
	cfg := config.Config{AWSRegion: "us-east-1", AWSEndpoint: "http://localhost:4566"}

	client, err := newSQSClient(context.Background(), cfg)

	require.NoError(t, err)
	require.NotNil(t, client.Options().BaseEndpoint)
	assert.Equal(t, "http://localhost:4566", *client.Options().BaseEndpoint)
}

func TestNewSQSClient_NoEndpointOverride(t *testing.T) {
	cfg := config.Config{AWSRegion: "us-east-1"}

	client, err := newSQSClient(context.Background(), cfg)

	require.NoError(t, err)
	assert.Nil(t, client.Options().BaseEndpoint)
}
