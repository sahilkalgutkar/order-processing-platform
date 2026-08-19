//go:build integration

package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	tcmongodb "github.com/testcontainers/testcontainers-go/modules/mongodb"
)

// connectMongo retries for up to 10 attempts so `docker compose up` doesn't
// require a manual restart while Mongo is still initializing. That retry
// loop is the actual behavior worth testing here — a real Mongo container
// stands in for the database instead of mocking the driver, matching how
// internal/repository's integration tests work.
func TestConnectMongo_ConnectsToRealMongo(t *testing.T) {
	ctx := context.Background()

	container, err := tcmongodb.Run(ctx, "mongo:7")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(context.Background())) })

	uri, err := container.ConnectionString(ctx)
	require.NoError(t, err)

	client, err := connectMongo(ctx, uri, discardLogger())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Disconnect(context.Background())) })

	require.NoError(t, client.Ping(ctx, nil))
}

func TestConnectMongo_GivesUpAfterRetriesOnBadURI(t *testing.T) {
	if testing.Short() {
		t.Skip("exercises the full 10-attempt retry loop (~20s); skipped with -short")
	}

	// A syntactically valid but unreachable URI makes mongo.Connect succeed
	// immediately while every Ping fails, driving the function through its
	// full retry loop (10 attempts, 2s apart) before it gives up and
	// returns the last error.
	_, err := connectMongo(context.Background(), "mongodb://localhost:1/?connectTimeoutMS=500&serverSelectionTimeoutMS=500", discardLogger())

	require.Error(t, err)
}
