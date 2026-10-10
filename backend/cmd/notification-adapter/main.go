package main

import (
	"log/slog"
	"os"

	"condo/internal/notification"
	"condo/pkg/config"
	"condo/pkg/db"
	"condo/pkg/events"
	"condo/pkg/server"
)

func main() {
	ctx, stop := server.Context()
	defer stop()
	pool, err := db.Connect(ctx, config.Get("DATABASE_URL", "postgres://condo:condo@localhost:5432/notification_db"))
	if err != nil {
		slog.Error("db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool, notification.Schema); err != nil {
		slog.Error("migrate", "err", err)
		os.Exit(1)
	}
	svc := notification.New(pool, config.Get("IDENTITY_URL", "http://localhost:8081"))
	go events.Consume(ctx, config.Get("AMQP_URL", "amqp://guest:guest@localhost:5672/"), "notification-adapter", notification.EventKeys, svc.Handle)
	if err := server.Run(ctx, "notification-adapter", ":"+config.Get("PORT", "8084"), svc.Routes()); err != nil {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}
