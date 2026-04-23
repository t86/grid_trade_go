package gateway

import (
	"context"
	"io"
	"testing"
	"time"

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

func TestGatewaySnapshotIncludesAccountMarketState(t *testing.T) {
	gw := NewService(fakeListenKeyProvider{}, fakeConnector{})

	err := gw.BootstrapAccount(context.Background(), "primary", []domain.MarketType{domain.MarketSpot})
	require.NoError(t, err)

	snapshot := gw.Snapshot()

	require.Len(t, snapshot.Accounts, 1)
	require.Equal(t, "primary", snapshot.Accounts[0].Account)
	require.Equal(t, domain.MarketSpot, snapshot.Accounts[0].Market)
	require.Equal(t, StateActive, snapshot.Accounts[0].SessionState)
	require.False(t, snapshot.Accounts[0].ReduceOnly)
	require.Equal(t, "connected", snapshot.Accounts[0].UserStreamState)
	require.Equal(t, "healthy", snapshot.Accounts[0].ListenKeyState)
	require.NotNil(t, snapshot.Accounts[0].LastReconnectAt)
	require.Len(t, snapshot.Events, 1)
}

func TestGatewayRuntimeSignalsAppearInSnapshot(t *testing.T) {
	gw := NewService(fakeListenKeyProvider{}, fakeConnector{})
	err := gw.BootstrapAccount(context.Background(), "primary", []domain.MarketType{domain.MarketFuturesUM})
	require.NoError(t, err)

	heartbeatAt := time.Now().Add(-3 * time.Second)
	expiresAt := time.Now().Add(25 * time.Minute)

	gw.RecordHeartbeat("primary", domain.MarketFuturesUM, heartbeatAt)
	gw.MarkListenKeyState("primary", domain.MarketFuturesUM, "expiring", &expiresAt, "refresh delayed")
	gw.MarkBackoff("primary", domain.MarketFuturesUM, true, "rate limited")

	snapshot := gw.Snapshot()

	require.Len(t, snapshot.Accounts, 1)
	require.Equal(t, int64(3000), snapshot.Accounts[0].HeartbeatLagMs)
	require.Equal(t, "expiring", snapshot.Accounts[0].ListenKeyState)
	require.Equal(t, "rate limited", snapshot.Accounts[0].LastError)
	require.True(t, snapshot.Accounts[0].ActiveBackoff)
	require.GreaterOrEqual(t, len(snapshot.Events), 3)
}

func TestGatewayCanRegisterDegradedConfiguredMarket(t *testing.T) {
	gw := NewService(nil, nil)

	gw.MarkMarketDegraded("primary", domain.MarketSpot, "missing BINANCE_API_KEY")

	snapshot := gw.Snapshot()

	require.Len(t, snapshot.Accounts, 1)
	require.Equal(t, "primary", snapshot.Accounts[0].Account)
	require.Equal(t, domain.MarketSpot, snapshot.Accounts[0].Market)
	require.Equal(t, StateDegraded, snapshot.Accounts[0].SessionState)
	require.Equal(t, "missing BINANCE_API_KEY", snapshot.Accounts[0].LastError)
}

func TestGatewaySnapshotIncludesSecretState(t *testing.T) {
	gw := NewService(nil, nil)

	gw.MarkSecretState("primary", domain.MarketFuturesUM, "primary", "load_failed", "secret file missing")

	snapshot := gw.Snapshot()

	require.Len(t, snapshot.Accounts, 1)
	require.Equal(t, "primary", snapshot.Accounts[0].SecretRef)
	require.Equal(t, "load_failed", snapshot.Accounts[0].SecretStatus)
	require.Equal(t, "secret file missing", snapshot.Accounts[0].LastError)
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
