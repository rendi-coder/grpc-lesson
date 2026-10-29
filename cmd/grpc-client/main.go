// grpc-client — демо QuoteService: 5 котирувань зі стріму, виконання IGLA,
// помилка NotFound для невідомого символу, скасування стріму.
//
//	go run ./cmd/grpc-client
//	GRPC_ADDR=localhost:50061 go run ./cmd/grpc-client
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	pb "libratrack/gen/quote"
)

func main() {
	if err := run(); err != nil {
		printRPCError(err)
		os.Exit(1)
	}
}

func run() error {
	// Відкритий текст — ЛИШЕ для локальної розробки.
	conn, err := grpc.NewClient(envOr("GRPC_ADDR", "localhost:50051"),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()
	c := pb.NewQuoteServiceClient(conn)

	streamCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream, err := c.StreamQuotes(streamCtx, &pb.StreamQuotesRequest{Symbol: "IGLA"})
	if err != nil {
		return err
	}
	for range 5 {
		q, err := stream.Recv()
		if err != nil {
			return err
		}
		fmt.Printf("quote  %s  %s  %.4f\n", q.GetTimestamp().AsTime().Local().Format("15:04:05.000"), q.GetSymbol(), q.GetPrice())
	}

	ctx, cancelCall := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelCall()
	e, err := c.ExecuteQuote(ctx, &pb.ExecuteQuoteRequest{Symbol: "IGLA", Quantity: 10})
	if err != nil {
		return err
	}
	fmt.Printf("executed  id=%s symbol=%s price=%.4f quantity=%g total=%.4f status=%s\n",
		e.GetId(), e.GetSymbol(), e.GetPrice(), e.GetQuantity(), e.GetTotal(), e.GetStatus())

	_, err = c.ExecuteQuote(ctx, &pb.ExecuteQuoteRequest{Symbol: "DOGE", Quantity: 1})
	printRPCError(err) // очікуємо code=NotFound

	cancel() // скасовуємо стрім: сервер побачить це через stream.Context()
	_, err = stream.Recv()
	fmt.Printf("stream cancelled: code=%s\n", status.Code(err))
	return nil
}

func printRPCError(err error) {
	if st, ok := status.FromError(err); ok {
		fmt.Fprintf(os.Stderr, "RPC помилка: code=%s message=%q\n", st.Code(), st.Message())
		return
	}
	fmt.Fprintln(os.Stderr, "помилка:", err)
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
