package main

import (
	"log/slog"
	"os"

	"condo/internal/paymentadapter"
	"condo/pkg/config"
	"condo/pkg/server"
)

func main() {
	ctx, stop := server.Context()
	defer stop()
	svc := paymentadapter.New(paymentadapter.Config{
		GatewayURL:    config.Get("GATEWAY_URL", "http://localhost:8090"),
		WebhookSecret: config.Get("WEBHOOK_SECRET", "dev-webhook-secret"),
		WebhookURL:    config.Get("WEBHOOK_URL", "http://localhost:8085/webhooks/gateway"),
		BillingURL:    config.Get("BILLING_URL", "http://localhost:8083"),
	})
	if err := server.Run(ctx, "payment-adapter", ":"+config.Get("PORT", "8085"), svc.Routes()); err != nil {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}
