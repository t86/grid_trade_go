package common

import (
	"fmt"

	"grid_trade/internal/domain"
)

type Endpoints struct {
	SpotRESTBaseURL    string
	FuturesRESTBaseURL string
	SpotWSAPIURL       string
	FuturesWSAPIURL    string
	SpotStreamURL      string
	FuturesStreamURL   string
}

func DefaultEndpoints() Endpoints {
	return Endpoints{
		SpotRESTBaseURL:    "https://api.binance.com",
		FuturesRESTBaseURL: "https://fapi.binance.com",
		SpotWSAPIURL:       "wss://ws-api.binance.com:443/ws-api/v3",
		FuturesWSAPIURL:    "wss://ws-fapi.binance.com/ws-fapi/v1",
		SpotStreamURL:      "wss://stream.binance.com:9443/ws",
		FuturesStreamURL:   "wss://fstream.binance.com/ws",
	}
}

func (e Endpoints) UserStreamURL(market domain.MarketType, listenKey string) (string, error) {
	switch market {
	case domain.MarketSpot:
		return fmt.Sprintf("%s/%s", e.SpotStreamURL, listenKey), nil
	case domain.MarketFuturesUM:
		return fmt.Sprintf("%s/%s", e.FuturesStreamURL, listenKey), nil
	default:
		return "", fmt.Errorf("unsupported market type %q", market)
	}
}
