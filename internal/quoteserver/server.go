// Package quoteserver — тонкий gRPC-адаптер над quote.Service: лише перетворює
// proto <-> типи сервісу і помилки сервісу <-> gRPC-коди.
package quoteserver

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "libratrack/gen/quote"
	"libratrack/internal/quote"
)

type Server struct {
	pb.UnimplementedQuoteServiceServer

	svc   *quote.Service
	every time.Duration // інтервал між котируваннями у StreamQuotes
}

func New(svc *quote.Service, every time.Duration) *Server {
	return &Server{svc: svc, every: every}
}

func (s *Server) ExecuteQuote(ctx context.Context, req *pb.ExecuteQuoteRequest) (*pb.Execution, error) {
	e, err := s.svc.Execute(ctx, req.GetSymbol(), req.GetQuantity())
	if err != nil {
		return nil, toStatus(err, req.GetSymbol())
	}
	return &pb.Execution{
		Id:       e.ID,
		Symbol:   e.Symbol,
		Price:    e.Price,
		Quantity: e.Quantity,
		Total:    e.Total,
		Status:   e.Status,
	}, nil
}

func (s *Server) StreamQuotes(req *pb.StreamQuotesRequest, stream grpc.ServerStreamingServer[pb.Quote]) error {
	ctx := stream.Context() // НЕ context.Background(): так ми бачимо скасування клієнтом
	t := time.NewTicker(s.every)
	defer t.Stop()
	for {
		q, err := s.svc.Next(req.GetSymbol())
		if err != nil {
			return toStatus(err, req.GetSymbol())
		}
		if err := stream.Send(&pb.Quote{Symbol: q.Symbol, Price: q.Price, Timestamp: timestamppb.New(q.At)}); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return status.FromContextError(ctx.Err()).Err()
		case <-t.C:
		}
	}
}

func toStatus(err error, symbol string) error {
	switch {
	case errors.Is(err, quote.ErrUnknownSymbol):
		return status.Errorf(codes.NotFound, "symbol %q not found", symbol)
	case errors.Is(err, quote.ErrInvalidQuantity):
		return status.Errorf(codes.InvalidArgument, "quantity must be positive")
	default:
		return status.Errorf(codes.Internal, "internal error")
	}
}
