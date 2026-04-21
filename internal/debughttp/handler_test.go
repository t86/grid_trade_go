package debughttp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"grid_trade/internal/debugview"
	"grid_trade/internal/domain"
)

func TestDashboardEndpointReturnsJSON(t *testing.T) {
	handler := NewHandler(fakeView{
		dashboard: debugview.Dashboard{
			Markets: []debugview.MarketHealthCard{{Market: domain.MarketSpot}},
		},
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/debug/dashboard", nil)
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Header().Get("Content-Type"), "application/json")
	require.Contains(t, rr.Body.String(), `"markets"`)
}

func TestAccountsEndpointRequiresMarket(t *testing.T) {
	handler := NewHandler(fakeView{})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/debug/accounts", nil)

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestEventsEndpointReturnsJSON(t *testing.T) {
	handler := NewHandler(fakeView{
		events: []debugview.Event{{Message: "heartbeat timeout"}},
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/debug/events?market=futures_um&account=primary", nil)

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), "heartbeat timeout")
}

type fakeView struct {
	dashboard debugview.Dashboard
	accounts  []debugview.AccountConnectionRow
	events    []debugview.Event
}

func (f fakeView) Dashboard() debugview.Dashboard {
	return f.dashboard
}

func (f fakeView) Accounts(domain.MarketType) []debugview.AccountConnectionRow {
	return f.accounts
}

func (f fakeView) Events(domain.MarketType, string) []debugview.Event {
	return f.events
}
