package grid

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"grid_trade/internal/domain"
)

func TestGridMovesFromBootstrappingToSeedingGrid(t *testing.T) {
	s := New(Config{Symbol: "BTCUSDT"})

	intents := s.OnMarketEvent(context.Background(), domain.MarketEvent{
		Symbol:  "BTCUSDT",
		BestBid: "100",
	})

	require.Equal(t, StateSeedingGrid, s.state)
	require.NotEmpty(t, intents)
}
