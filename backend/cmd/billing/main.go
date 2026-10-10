package main

import (
	"log/slog"
	"os"
	"strconv"

	"condo/internal/billing"
	"condo/pkg/config"
	"condo/pkg/db"
	"condo/pkg/events"
	"condo/pkg/server"
)

func rate(key string, fallback float64) float64 {
	if v, err := strconv.ParseFloat(config.Get(key, ""), 64); err == nil {
		return v
	}
	return fallback
}

func main() {
	ctx, stop := server.Context()
	defer stop()
	pool, err := db.Connect(ctx, config.Get("DATABASE_URL", "postgres://condo:condo@localhost:5432/billing_db"))
	if err != nil {
		slog.Error("db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool, billing.Schema); err != nil {
		slog.Error("migrate", "err", err)
		os.Exit(1)
	}
	svc := billing.New(pool, events.NewPublisher(config.Get("AMQP_URL", "amqp://guest:guest@localhost:5672/")), billing.Config{
		WaterRate:         rate("WATER_RATE", 18),
		ElectricRate:      rate("ELECTRIC_RATE", 7),
		PaymentAdapterURL: config.Get("PAYMENT_ADAPTER_URL", "http://localhost:8085"),
		ReturnURL:         config.Get("RESIDENT_RETURN_URL", "http://localhost:3001/bills/return"),
	})
	if err := server.Run(ctx, "billing", ":"+config.Get("PORT", "8083"), svc.Routes()); err != nil {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}
