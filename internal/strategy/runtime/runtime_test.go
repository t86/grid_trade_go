package runtime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"grid_trade/internal/domain"
)

func TestRuntimeProcessesStrategyEventsSequentially(t *testing.T) {
	rt := New()
	strategy := &fakeStrategy{}

	require.NoError(t, rt.Register(strategy))

	rt.DispatchMarketEvent(context.Background(), domain.MarketEvent{Symbol: "BTCUSDT"})
	rt.DispatchMarketEvent(context.Background(), domain.MarketEvent{Symbol: "BTCUSDT"})

	require.Equal(t, []string{"market", "market"}, strategy.calls)
}

type fakeStrategy struct {
	calls []string
}

func (f *fakeStrategy) ID() string { return "fake" }

func (f *fakeStrategy) OnMarketEvent(context.Context, domain.MarketEvent) []domain.OrderIntent {
	f.calls = append(f.calls, "market")
	return nil
}

func (f *fakeStrategy) OnOrderEvent(context.Context, domain.OrderEvent) []domain.OrderIntent {
	return nil
}

func (f *fakeStrategy) Snapshot() any { return nil }
