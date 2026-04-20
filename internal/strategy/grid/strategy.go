package grid

import (
	"context"

	"grid_trade/internal/domain"
)

type State string

const (
	StateBootstrapping State = "Bootstrapping"
	StateSeedingGrid   State = "SeedingGrid"
	StateActive        State = "Active"
	StateReduceOnly    State = "ReduceOnly"
)

type Config struct {
	Symbol string
}

type Strategy struct {
	cfg   Config
	state State
}

func New(cfg Config) *Strategy {
	return &Strategy{
		cfg:   cfg,
		state: StateBootstrapping,
	}
}

func (s *Strategy) OnMarketEvent(_ context.Context, event domain.MarketEvent) []domain.OrderIntent {
	if s.state == StateBootstrapping && event.Symbol == s.cfg.Symbol {
		s.state = StateSeedingGrid
		return s.seedOrders()
	}
	return nil
}

func (s *Strategy) seedOrders() []domain.OrderIntent {
	return []domain.OrderIntent{
		{
			StrategyID:    "grid",
			Symbol:        s.cfg.Symbol,
			ClientOrderID: "seed-1",
			Reason:        "initial grid seed",
		},
	}
}
