package main

import (
	"log/slog"
	"os"

	"condo/internal/mockgateway"
	"condo/pkg/config"
	"condo/pkg/server"
)

func main() {
	ctx, stop := server.Context()
	defer stop()
	srv := mockgateway.New(config.Get("WEBHOOK_SECRET", "dev-webhook-secret"), config.Get("PUBLIC_URL", "http://localhost:8090"))
	if err := server.Run(ctx, "mock-gateway", ":"+config.Get("PORT", "8090"), srv.Routes()); err != nil {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}
