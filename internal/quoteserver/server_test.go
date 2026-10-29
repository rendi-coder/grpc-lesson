package quoteserver_test

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	pb "libratrack/gen/quote"
	"libratrack/internal/quote"
	"libratrack/internal/quoteserver"
)

// Справжній gRPC-сервер у пам'яті (bufconn — без мережі й портів); обидва RPC.
func TestQuoteService_OverBufconn(t *testing.T) {
	lis := bufconn.Listen(1 << 20)
	srv := grpc.NewServer()
	pb.RegisterQuoteServiceServer(srv, quoteserver.New(quote.New(quote.NewMemoryRepository()), 10*time.Millisecond))
	go func() { _ = srv.Serve(lis) }()
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close(); srv.Stop() })
	c := pb.NewQuoteServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Серверний стрімінг: кілька котирувань, потім скасування клієнтом.
	streamCtx, cancelStream := context.WithCancel(ctx)
	stream, err := c.StreamQuotes(streamCtx, &pb.StreamQuotesRequest{Symbol: "IGLA"})
	if err != nil {
		t.Fatal(err)
	}
	for range 3 {
		q, err := stream.Recv()
		if err != nil {
			t.Fatal(err)
		}
		if q.GetSymbol() != "IGLA" || q.GetPrice() < 0.01 {
			t.Fatalf("неочікувана котирувка: %v", q)
		}
	}
	cancelStream()
	if _, err := stream.Recv(); status.Code(err) != codes.Canceled {
		t.Fatalf("після скасування: %v, хотіли Canceled", err)
	}

	// Unary: успіх і NotFound (а не Unknown).
	e, err := c.ExecuteQuote(ctx, &pb.ExecuteQuoteRequest{Symbol: "IGLA", Quantity: 10})
	if err != nil {
		t.Fatal(err)
	}
	if e.GetId() == "" || e.GetQuantity() != 10 || e.GetStatus() != quote.StatusFilled {
		t.Fatalf("неочікуване виконання: %v", e)
	}
	_, err = c.ExecuteQuote(ctx, &pb.ExecuteQuoteRequest{Symbol: "DOGE", Quantity: 1})
	if got := status.Code(err); got != codes.NotFound {
		t.Fatalf("code = %v, хотіли NotFound", got)
	}
}
