package quote

import (
	"context"
	"fmt"
	"sync"
)

const StatusFilled = "FILLED"

// Execution — записане виконання котирувки.
type Execution struct {
	ID       string
	Symbol   string
	Price    float64
	Quantity float64
	Total    float64
	Status   string
}

// Repository зберігає виконання. Сервіс знає лише інтерфейс, тож map у пам'яті
// пізніше можна замінити справжньою БД, не чіпаючи логіку.
type Repository interface {
	Save(ctx context.Context, e Execution) (Execution, error)
}

// MemoryRepository — реалізація Repository у пам'яті.
type MemoryRepository struct {
	mu     sync.Mutex
	execs  map[string]Execution
	nextID int
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{execs: make(map[string]Execution), nextID: 1}
}

// Save призначає ID (exec-1, exec-2, ...) і зберігає виконання.
func (r *MemoryRepository) Save(_ context.Context, e Execution) (Execution, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e.ID = fmt.Sprintf("exec-%d", r.nextID)
	r.nextID++
	r.execs[e.ID] = e
	return e, nil
}
