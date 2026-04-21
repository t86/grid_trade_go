package common

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"grid_trade/internal/domain"
)

type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

type RESTClient struct {
	apiKey    string
	client    Doer
	endpoints Endpoints
}

type listenKeyResponse struct {
	ListenKey string `json:"listenKey"`
}

func NewRESTClient(apiKey string, client Doer, endpoints Endpoints) *RESTClient {
	if client == nil {
		client = http.DefaultClient
	}
	return &RESTClient{
		apiKey:    apiKey,
		client:    client,
		endpoints: endpoints,
	}
}

func (c *RESTClient) CreateListenKey(ctx context.Context, market domain.MarketType) (string, error) {
	url, err := c.listenKeyURL(market)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-MBX-APIKEY", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("listen key request failed with status %d", resp.StatusCode)
	}

	var payload listenKeyResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if payload.ListenKey == "" {
		return "", fmt.Errorf("empty listen key")
	}
	return payload.ListenKey, nil
}

func (c *RESTClient) listenKeyURL(market domain.MarketType) (string, error) {
	switch market {
	case domain.MarketSpot:
		return c.endpoints.SpotRESTBaseURL + "/api/v3/userDataStream", nil
	case domain.MarketFuturesUM:
		return c.endpoints.FuturesRESTBaseURL + "/fapi/v1/listenKey", nil
	default:
		return "", fmt.Errorf("unsupported market type %q", market)
	}
}
