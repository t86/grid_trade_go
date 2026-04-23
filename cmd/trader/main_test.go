package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func TestBuildAppMarksAccountDegradedWhenSecretFileMissing(t *testing.T) {
	app, err := BuildApp(config.Config{
		System: config.SystemConfig{SecretDir: t.TempDir()},
		Accounts: []config.AccountConfig{
			{
				Name:        "primary",
				Enabled:     true,
				MarketTypes: []string{"spot", "futures_um"},
				SecretRef:   "primary",
			},
		},
	})
	require.NoError(t, err)

	snapshot := app.gateway.Snapshot()

	require.Len(t, snapshot.Accounts, 2)
	require.Equal(t, gateway.StateDegraded, snapshot.Accounts[0].SessionState)
	require.Equal(t, "load_failed", snapshot.Accounts[0].SecretStatus)
	require.Contains(t, snapshot.Accounts[0].LastError, "secret load failed")
}

func TestBuildAppSkipsDisabledAccounts(t *testing.T) {
	secretDir := t.TempDir()
	payload, err := json.Marshal(map[string]string{
		"api_key":    "api",
		"secret_key": "secret",
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(secretDir, "disabled.json"), payload, 0o600))

	app, err := BuildApp(config.Config{
		System: config.SystemConfig{SecretDir: secretDir},
		Accounts: []config.AccountConfig{
			{
				Name:        "disabled",
				Enabled:     false,
				MarketTypes: []string{"spot"},
				SecretRef:   "disabled",
			},
		},
	})
	require.NoError(t, err)

	require.Empty(t, app.gateway.Snapshot().Accounts)
}
