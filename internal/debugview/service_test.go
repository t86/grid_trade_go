package debugview

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"grid_trade/internal/domain"
	"grid_trade/internal/gateway"
)

func TestServiceBuildsDashboardFromGatewaySnapshot(t *testing.T) {
	reconnectAt := time.Now().Add(-2 * time.Minute)
	svc := NewService(fakeGatewaySnapshot{
		accounts: []gateway.AccountSnapshot{
			{Account: "primary", Market: domain.MarketFuturesUM, SessionState: gateway.StateDegraded, ReduceOnly: true, LastError: "listen key stale", HeartbeatLagMs: 43000, ListenKeyState: "stale", LastReconnectAt: &reconnectAt},
			{Account: "primary", Market: domain.MarketSpot, SessionState: gateway.StateActive, ListenKeyState: "healthy", HeartbeatLagMs: 1800},
		},
	})

	dashboard := svc.Dashboard()

	require.Len(t, dashboard.Alerts, 2)
	require.Equal(t, domain.MarketSpot, dashboard.Markets[0].Market)
	require.Equal(t, domain.MarketFuturesUM, dashboard.Markets[1].Market)
	require.Equal(t, HealthDegraded, dashboard.Markets[1].Health)
	require.Equal(t, 1, dashboard.Markets[1].ReduceOnlyAccounts)
	require.Equal(t, int64(43000), dashboard.Markets[1].AvgHeartbeatLagMs)
	require.Equal(t, 1, dashboard.Markets[0].ListenKeyHealthyAccounts)
}

func TestAccountsSortsProblemRowsFirst(t *testing.T) {
	svc := NewService(fakeGatewaySnapshot{accounts: []gateway.AccountSnapshot{
		{Account: "ok", Market: domain.MarketSpot, SessionState: gateway.StateActive},
		{Account: "bad", Market: domain.MarketSpot, SessionState: gateway.StateDegraded},
	}})

	rows := svc.Accounts(domain.MarketSpot)

	require.Equal(t, "bad", rows[0].Account)
	require.Equal(t, "ok", rows[1].Account)
}

func TestEventsReturnsRecentGatewayEvents(t *testing.T) {
	now := time.Now()
	svc := NewService(fakeGatewaySnapshot{
		events: []gateway.EventSnapshot{
			{Account: "primary", Market: domain.MarketFuturesUM, Category: "connection", Message: "heartbeat timeout", Timestamp: now},
		},
	})

	events := svc.Events(domain.MarketFuturesUM, "primary")

	require.Len(t, events, 1)
	require.Equal(t, "heartbeat timeout", events[0].Message)
}

type fakeGatewaySnapshot struct {
	accounts []gateway.AccountSnapshot
	events   []gateway.EventSnapshot
}

func (f fakeGatewaySnapshot) Snapshot() gateway.Snapshot {
	return gateway.Snapshot{Accounts: f.accounts, Events: f.events}
}
