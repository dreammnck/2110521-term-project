// Package mongodb opens a MongoDB connection, retrying while the server starts.
package mongodb

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func Connect(ctx context.Context, uri string) (*mongo.Client, error) {
	var lastErr error
	for i := 0; i < 30; i++ {
		client, err := mongo.Connect(options.Client().ApplyURI(uri).SetServerSelectionTimeout(3 * time.Second))
		if err == nil {
			pctx, cancel := context.WithTimeout(ctx, 3*time.Second)
			err = client.Ping(pctx, nil)
			cancel()
			if err == nil {
				return client, nil
			}
			_ = client.Disconnect(ctx)
		}
		lastErr = err
		slog.Info("waiting for mongodb", "attempt", i+1, "err", err)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return nil, fmt.Errorf("mongodb unavailable: %w", lastErr)
}
