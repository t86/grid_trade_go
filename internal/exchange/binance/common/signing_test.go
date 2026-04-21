package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"grid_trade/internal/domain"
)

func TestBuildSignedOrderPlaceRequest(t *testing.T) {
	req, err := BuildOrderPlaceRequest(
		"req-1",
		domain.OrderIntent{
			Symbol:        "BTCUSDT",
			Side:          "BUY",
			OrderType:     "LIMIT",
			Quantity:      "0.001",
			Price:         "100000",
			TimeInForce:   "GTC",
			PositionSide:  "LONG",
			ReduceOnly:    true,
			ClientOrderID: "cid-1",
		},
		"api-key",
		"secret",
		time.UnixMilli(1710000000000),
	)
	require.NoError(t, err)
	require.Equal(t, "req-1", req.ID)
	require.Equal(t, "order.place", req.Method)
	require.Equal(t, "api-key", req.Params["apiKey"])
	require.Equal(t, "BTCUSDT", req.Params["symbol"])
	require.Equal(t, "LONG", req.Params["positionSide"])
	require.Equal(t, "true", req.Params["reduceOnly"])
	require.Equal(t, "cid-1", req.Params["newClientOrderId"])
	require.NotEmpty(t, req.Params["signature"])
	require.Equal(t, int64(1710000000000), req.Params["timestamp"])
}
