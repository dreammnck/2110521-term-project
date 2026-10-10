package main

import (
	"log/slog"
	"os"

	"condo/internal/identity"
	"condo/pkg/config"
	"condo/pkg/db"
	"condo/pkg/server"
)

func main() {
	ctx, stop := server.Context()
	defer stop()
	pool, err := db.Connect(ctx, config.Get("DATABASE_URL", "postgres://condo:condo@localhost:5432/identity_db"))
	if err != nil {
		slog.Error("db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool, identity.Schema); err != nil {
		slog.Error("migrate", "err", err)
		os.Exit(1)
	}
	svc := identity.New(pool, config.Get("JWT_SECRET", "dev-secret"))
	if err := svc.Seed(ctx); err != nil {
		slog.Error("seed", "err", err)
		os.Exit(1)
	}
	if err := server.Run(ctx, "identity", ":"+config.Get("PORT", "8081"), svc.Routes()); err != nil {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}
