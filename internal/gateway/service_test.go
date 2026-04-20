package gateway

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGatewayEntersDegradedModeOnUserStreamFailure(t *testing.T) {
	gw := NewService(nil, nil)

	gw.MarkUserStreamDown("primary")

	require.Equal(t, StateDegraded, gw.State("primary"))
	health := gw.Health()
	require.True(t, health.ReduceOnly)
	require.False(t, health.TradingEnabled)
}
