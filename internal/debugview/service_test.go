package debugview

import (
	"testing"

	"github.com/stretchr/testify/require"

	"grid_trade/internal/domain"
	"grid_trade/internal/gateway"
)

func TestServiceBuildsDashboardFromGatewaySnapshot(t *testing.T) {
	svc := NewService(fakeGatewaySnapshot{
		accounts: []gateway.AccountSnapshot{
			{Account: "primary", Market: domain.MarketFuturesUM, SessionState: gateway.StateDegraded, ReduceOnly: true, LastError: "listen key stale"},
			{Account: "primary", Market: domain.MarketSpot, SessionState: gateway.StateActive},
		},
	})

	dashboard := svc.Dashboard()

	require.Len(t, dashboard.Alerts, 2)
	require.Equal(t, domain.MarketSpot, dashboard.Markets[0].Market)
	require.Equal(t, domain.MarketFuturesUM, dashboard.Markets[1].Market)
	require.Equal(t, HealthDegraded, dashboard.Markets[1].Health)
	require.Equal(t, 1, dashboard.Markets[1].ReduceOnlyAccounts)
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

type fakeGatewaySnapshot struct {
	accounts []gateway.AccountSnapshot
}

func (f fakeGatewaySnapshot) Snapshot() gateway.Snapshot {
	return gateway.Snapshot{Accounts: f.accounts}
}
