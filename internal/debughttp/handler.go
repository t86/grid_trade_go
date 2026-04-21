package debughttp

import (
	"encoding/json"
	"net/http"
	"strings"

	"grid_trade/internal/debugview"
	"grid_trade/internal/domain"
)

type View interface {
	Dashboard() debugview.Dashboard
	Accounts(domain.MarketType) []debugview.AccountConnectionRow
	Events(domain.MarketType, string) []debugview.Event
}

type Handler struct {
	view View
	mux  *http.ServeMux
}

type AccountsResponse struct {
	Market   domain.MarketType                `json:"market"`
	Accounts []debugview.AccountConnectionRow `json:"accounts"`
}

type EventsResponse struct {
	Events []debugview.Event `json:"events"`
}

func NewHandler(view View) *Handler {
	mux := http.NewServeMux()
	h := &Handler{
		view: view,
		mux:  mux,
	}
	mux.HandleFunc("/debug/dashboard", h.dashboard)
	mux.HandleFunc("/debug/accounts", h.accounts)
	mux.HandleFunc("/debug/events", h.events)
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) dashboard(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.view.Dashboard())
}

func (h *Handler) accounts(w http.ResponseWriter, r *http.Request) {
	market := domain.MarketType(strings.TrimSpace(r.URL.Query().Get("market")))
	if market == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "market is required"})
		return
	}

	writeJSON(w, http.StatusOK, AccountsResponse{
		Market:   market,
		Accounts: h.view.Accounts(market),
	})
}

func (h *Handler) events(w http.ResponseWriter, r *http.Request) {
	market := domain.MarketType(strings.TrimSpace(r.URL.Query().Get("market")))
	account := strings.TrimSpace(r.URL.Query().Get("account"))
	writeJSON(w, http.StatusOK, EventsResponse{
		Events: h.view.Events(market, account),
	})
}

func writeJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}
