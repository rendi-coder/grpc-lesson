// grpc-server — QuoteService для мок-монети IGLA.
//
//	go run ./cmd/grpc-server                     # слухає :50051
//	GRPC_ADDR=:50061 go run ./cmd/grpc-server    # інша адреса
package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "libratrack/gen/quote"
	"libratrack/internal/middleware"
	"libratrack/internal/quote"
	"libratrack/internal/quoteserver"
)

func main() {
	addr := envOr("GRPC_ADDR", ":50051")
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	svc := quote.New(quote.NewMemoryRepository())
	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(middleware.UnaryRecovery(log), middleware.UnaryLogging(log)),
		grpc.ChainStreamInterceptor(middleware.StreamRecovery(log), middleware.StreamLogging(log)),
	)
	pb.RegisterQuoteServiceServer(srv, quoteserver.New(svc, time.Second))
	reflection.Register(srv) // дозволяє grpcurl працювати без .proto-файлів (у проді — вимкнути)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Error("listen", "addr", addr, "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		srv.Stop() // відкриті стріми не завершаться самі, тож без GracefulStop
	}()

	log.Info("quote server started", "addr", lis.Addr().String())
	log.Warn("TLS вимкнено — відкритий текст лише для локальної розробки")
	if err := srv.Serve(lis); err != nil {
		log.Error("serve", "err", err)
		os.Exit(1)
	}
	log.Info("bye")
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
