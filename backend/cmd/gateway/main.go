package main

import (
	"log/slog"
	"os"
	"strings"

	pb "condo/gen/announcement/v1"
	"condo/internal/gateway"
	"condo/pkg/config"
	"condo/pkg/server"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	ctx, stop := server.Context()
	defer stop()
	conn, err := grpc.NewClient(config.Get("ANNOUNCEMENT_GRPC_ADDR", "localhost:9090"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("grpc", "err", err)
		os.Exit(1)
	}
	defer conn.Close()
	gw, err := gateway.New(gateway.Config{
		JWTSecret:       config.Get("JWT_SECRET", "dev-secret"),
		IdentityURL:     config.Get("IDENTITY_URL", "http://localhost:8081"),
		ParcelURL:       config.Get("PARCEL_URL", "http://localhost:8082"),
		BillingURL:      config.Get("BILLING_URL", "http://localhost:8083"),
		NotificationURL: config.Get("NOTIFICATION_URL", "http://localhost:8084"),
		AllowedOrigins:  strings.Split(config.Get("ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:3001"), ","),
	}, pb.NewAnnouncementServiceClient(conn))
	if err != nil {
		slog.Error("gateway", "err", err)
		os.Exit(1)
	}
	if err := server.Run(ctx, "gateway", ":"+config.Get("PORT", "8080"), gw.Routes()); err != nil {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}
