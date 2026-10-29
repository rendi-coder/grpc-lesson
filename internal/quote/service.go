// Package quote — логіка мок-монети IGLA: ціна (випадкове блукання) і виконання котирувок.
// Про gRPC тут нічого не відомо: перетворення proto <-> типи сервісу робить internal/quoteserver.
package quote

import (
	"context"
	"errors"
	"math/rand/v2"
	"strings"
	"sync"
	"time"
)

const Symbol = "IGLA"

var (
	ErrUnknownSymbol   = errors.New("unknown symbol")
	ErrInvalidQuantity = errors.New("quantity must be positive")
)

// Quote — ціна символу в момент часу.
type Quote struct {
	Symbol string
	Price  float64
	At     time.Time
}

type Service struct {
	repo Repository

	mu    sync.Mutex
	price float64
}

// New створює сервіс зі стартовою ціною 1.00.
func New(repo Repository) *Service {
	return &Service{repo: repo, price: 1.00}
}

// Next робить один крок випадкового блукання (±1 цент) і повертає нову ціну.
func (s *Service) Next(symbol string) (Quote, error) {
	if !supported(symbol) {
		return Quote{}, ErrUnknownSymbol
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.price += (rand.Float64() - 0.5) * 0.02
	if s.price < 0.01 { // ціна ніколи не падає нижче 0.01
		s.price = 0.01
	}
	return Quote{Symbol: Symbol, Price: s.price, At: time.Now()}, nil
}

// Execute бере поточну ціну, записує виконання в репозиторій і повертає його.
func (s *Service) Execute(ctx context.Context, symbol string, quantity float64) (Execution, error) {
	if !supported(symbol) {
		return Execution{}, ErrUnknownSymbol
	}
	if quantity <= 0 {
		return Execution{}, ErrInvalidQuantity
	}
	s.mu.Lock()
	price := s.price
	s.mu.Unlock()
	return s.repo.Save(ctx, Execution{
		Symbol:   Symbol,
		Price:    price,
		Quantity: quantity,
		Total:    price * quantity,
		Status:   StatusFilled,
	})
}

func supported(symbol string) bool { return strings.EqualFold(symbol, Symbol) }
