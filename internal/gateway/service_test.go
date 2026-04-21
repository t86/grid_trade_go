package gateway

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"grid_trade/internal/domain"
)

func TestGatewayEntersDegradedModeOnUserStreamFailure(t *testing.T) {
	gw := NewService(nil, nil)

	gw.MarkUserStreamDown("primary")

	require.Equal(t, StateDegraded, gw.State("primary"))
	health := gw.Health()
	require.True(t, health.ReduceOnly)
	require.False(t, health.TradingEnabled)
}

func TestGatewayBootstrapsUserStreamsAndMarksSessionActive(t *testing.T) {
	gw := NewService(fakeListenKeyProvider{}, fakeConnector{})

	err := gw.BootstrapAccount(context.Background(), "primary", []domain.MarketType{domain.MarketSpot, domain.MarketFuturesUM})
	require.NoError(t, err)
	require.Equal(t, StateActive, gw.State("primary"))
	health := gw.Health()
	require.False(t, health.ReduceOnly)
	require.True(t, health.TradingEnabled)
}

type fakeListenKeyProvider struct{}

func (fakeListenKeyProvider) CreateListenKey(context.Context, domain.MarketType) (string, error) {
	return "listen-key", nil
}

type fakeConnector struct{}

func (fakeConnector) Connect(context.Context, domain.MarketType, string) (io.Closer, error) {
	return nopCloser{}, nil
}

type nopCloser struct{}

func (nopCloser) Close() error { return nil }
