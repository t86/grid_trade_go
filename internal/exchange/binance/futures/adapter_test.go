package futures

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeOrderAck(t *testing.T) {
	adapter := Adapter{}
	raw := []byte(`{"id":"1","status":200,"result":{"symbol":"BTCUSDT","clientOrderId":"cid-2"}}`)

	event, err := adapter.NormalizeOrderAck(raw)
	require.NoError(t, err)
	require.Equal(t, "cid-2", event.ClientOrderID)
	require.Equal(t, "BTCUSDT", event.Symbol)
	require.Equal(t, "ACK", event.Status)
}
