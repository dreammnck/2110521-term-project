package main

import (
	"log/slog"
	"os"

	"condo/internal/notification"
	"condo/pkg/config"
	"condo/pkg/events"
	"condo/pkg/mongodb"
	"condo/pkg/server"
)

func main() {
	ctx, stop := server.Context()
	defer stop()
	client, err := mongodb.Connect(ctx, config.Get("MONGO_URL", "mongodb://localhost:27018"))
	if err != nil {
		slog.Error("mongodb", "err", err)
		os.Exit(1)
	}
	defer client.Disconnect(ctx) //nolint:errcheck
	svc := notification.New(client.Database(config.Get("MONGO_DB", "notification_db")).Collection("notifications"), config.Get("IDENTITY_URL", "http://localhost:8081"))
	if err := svc.EnsureIndexes(ctx); err != nil {
		slog.Error("indexes", "err", err)
		os.Exit(1)
	}
	go events.Consume(ctx, config.Get("AMQP_URL", "amqp://guest:guest@localhost:5672/"), "notification-adapter", notification.EventKeys, svc.Handle)
	if err := server.Run(ctx, "notification-adapter", ":"+config.Get("PORT", "8084"), svc.Routes()); err != nil {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}
