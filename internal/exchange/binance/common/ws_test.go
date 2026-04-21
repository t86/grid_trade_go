package common

import (
	"context"
	"net/http"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"

	"grid_trade/internal/domain"
)

func TestUserStreamConnectorUsesMarketSpecificURL(t *testing.T) {
	dialer := &fakeDialer{}
	connector := NewUserStreamConnector(Endpoints{
		SpotStreamURL:    "wss://spot.example/ws",
		FuturesStreamURL: "wss://futures.example/ws",
	}, dialer)

	_, err := connector.Connect(context.Background(), domain.MarketSpot, "spot-key")
	require.NoError(t, err)
	require.Equal(t, "wss://spot.example/ws/spot-key", dialer.lastURL)

	_, err = connector.Connect(context.Background(), domain.MarketFuturesUM, "futures-key")
	require.NoError(t, err)
	require.Equal(t, "wss://futures.example/ws/futures-key", dialer.lastURL)
}

type fakeDialer struct {
	lastURL string
}

func (f *fakeDialer) DialContext(_ context.Context, urlStr string, _ http.Header) (*websocket.Conn, *http.Response, error) {
	f.lastURL = urlStr
	return &websocket.Conn{}, nil, nil
}
