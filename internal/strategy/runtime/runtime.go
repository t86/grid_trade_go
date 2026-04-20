package runtime

import (
	"context"
	"fmt"
	"sync"

	"grid_trade/internal/domain"
)

type Strategy interface {
	ID() string
	OnMarketEvent(context.Context, domain.MarketEvent) []domain.OrderIntent
	OnOrderEvent(context.Context, domain.OrderEvent) []domain.OrderIntent
	Snapshot() any
}

type Runtime struct {
	mu         sync.RWMutex
	strategies map[string]Strategy
}

func New() *Runtime {
	return &Runtime{
		strategies: make(map[string]Strategy),
	}
}

func (r *Runtime) Register(strategy Strategy) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.strategies[strategy.ID()]; exists {
		return fmt.Errorf("strategy %s already registered", strategy.ID())
	}
	r.strategies[strategy.ID()] = strategy
	return nil
}

func (r *Runtime) DispatchMarketEvent(ctx context.Context, event domain.MarketEvent) []domain.OrderIntent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var intents []domain.OrderIntent
	for _, strategy := range r.strategies {
		intents = append(intents, strategy.OnMarketEvent(ctx, event)...)
	}
	return intents
}
