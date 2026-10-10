package main

import (
	"log/slog"
	"net"
	"os"

	pb "condo/gen/announcement/v1"
	"condo/internal/announcement"
	"condo/pkg/config"
	"condo/pkg/db"
	"condo/pkg/events"
	"condo/pkg/server"

	"google.golang.org/grpc"
)

func main() {
	ctx, stop := server.Context()
	defer stop()
	pool, err := db.Connect(ctx, config.Get("DATABASE_URL", "postgres://condo:condo@localhost:5432/announcement_db"))
	if err != nil {
		slog.Error("db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool, announcement.Schema); err != nil {
		slog.Error("migrate", "err", err)
		os.Exit(1)
	}
	lis, err := net.Listen("tcp", ":"+config.Get("PORT", "9090"))
	if err != nil {
		slog.Error("listen", "err", err)
		os.Exit(1)
	}
	gs := grpc.NewServer()
	pb.RegisterAnnouncementServiceServer(gs, announcement.New(pool, events.NewPublisher(config.Get("AMQP_URL", "amqp://guest:guest@localhost:5672/"))))
	go func() { <-ctx.Done(); gs.GracefulStop() }()
	slog.Info("listening", "service", "announcement", "addr", lis.Addr().String())
	if err := gs.Serve(lis); err != nil {
		slog.Error("serve", "err", err)
		os.Exit(1)
	}
}
