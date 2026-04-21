package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"grid_trade/internal/config"
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
