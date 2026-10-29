package quote_test

import (
	"context"
	"errors"
	"testing"

	"libratrack/internal/quote"
)

func TestExecute_OK(t *testing.T) {
	svc := quote.New(quote.NewMemoryRepository())
	e, err := svc.Execute(context.Background(), "igla", 10) // регістр не важливий
	if err != nil {
		t.Fatal(err)
	}
	if e.ID != "exec-1" || e.Symbol != "IGLA" || e.Status != quote.StatusFilled {
		t.Fatalf("неочікуване виконання: %+v", e)
	}
	if e.Price < 0.01 || e.Total != e.Price*10 {
		t.Fatalf("ціна/сума: %+v", e)
	}
}

func TestExecute_UnknownSymbol(t *testing.T) {
	svc := quote.New(quote.NewMemoryRepository())
	_, err := svc.Execute(context.Background(), "DOGE", 1)
	if !errors.Is(err, quote.ErrUnknownSymbol) {
		t.Fatalf("err = %v, хотіли ErrUnknownSymbol", err)
	}
}

func TestExecute_BadQuantity(t *testing.T) {
	svc := quote.New(quote.NewMemoryRepository())
	for _, q := range []float64{0, -5} {
		if _, err := svc.Execute(context.Background(), "IGLA", q); !errors.Is(err, quote.ErrInvalidQuantity) {
			t.Fatalf("quantity=%v: err = %v, хотіли ErrInvalidQuantity", q, err)
		}
	}
}
