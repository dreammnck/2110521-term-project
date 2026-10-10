package main

import (
	"log/slog"
	"os"

	"condo/internal/parcel"
	"condo/pkg/config"
	"condo/pkg/db"
	"condo/pkg/events"
	"condo/pkg/server"
)

func main() {
	ctx, stop := server.Context()
	defer stop()
	pool, err := db.Connect(ctx, config.Get("DATABASE_URL", "postgres://condo:condo@localhost:5432/parcel_db"))
	if err != nil {
		slog.Error("db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool, parcel.Schema); err != nil {
		slog.Error("migrate", "err", err)
		os.Exit(1)
	}
	pub := events.NewPublisher(config.Get("AMQP_URL", "amqp://guest:guest@localhost:5672/"))
	svc := parcel.New(pool, pub, config.Get("PHOTO_DIR", "./data/photos"))
	if err := server.Run(ctx, "parcel", ":"+config.Get("PORT", "8082"), svc.Routes()); err != nil {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}
