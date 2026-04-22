package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"grid_trade/internal/config"
	"grid_trade/internal/gateway"
)

func TestBuildAppWiresCoreServices(t *testing.T) {
	app, err := BuildApp(config.Config{})
	require.NoError(t, err)
	require.NotNil(t, app.gateway)
	require.NotNil(t, app.runtime)
}

func TestBuildAppUsesEnvHTTPAddrOverride(t *testing.T) {
	t.Setenv("TRADER_HTTP_ADDR", ":18080")

	app, err := BuildApp(config.Config{})
	require.NoError(t, err)
	require.Equal(t, ":18080", app.httpAddr)
}

func TestBuildAppMountsDebugDashboard(t *testing.T) {
	app, err := BuildApp(config.Config{})
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/debug/dashboard", nil)
	app.handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestBuildAppShowsConfiguredMarketsAsDegradedWhenCredentialsMissing(t *testing.T) {
	t.Setenv("MISSING_BINANCE_API_KEY", "")
	t.Setenv("MISSING_BINANCE_SECRET_KEY", "")

	app, err := BuildApp(config.Config{
		Accounts: []config.AccountConfig{
			{
				Name:         "primary",
				MarketTypes:  []string{"spot", "futures_um"},
				APIKeyEnv:    "MISSING_BINANCE_API_KEY",
				SecretKeyEnv: "MISSING_BINANCE_SECRET_KEY",
			},
		},
	})
	require.NoError(t, err)

	snapshot := app.gateway.Snapshot()

	require.Len(t, snapshot.Accounts, 2)
	require.Equal(t, gateway.StateDegraded, snapshot.Accounts[0].SessionState)
	require.Contains(t, snapshot.Accounts[0].LastError, "missing")
}
