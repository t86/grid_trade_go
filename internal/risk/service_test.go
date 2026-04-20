package risk

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"grid_trade/internal/domain"
)

func TestRiskRejectsExposureAboveLimit(t *testing.T) {
	svc := Service{MaxStrategyNotional: decimal.RequireFromString("1000")}
	intent := domain.OrderIntent{
		StrategyID: "grid-btc",
		Symbol:     "BTCUSDT",
		Quantity:   "2",
		Price:      "1000",
	}

	err := svc.Check(intent)
	require.ErrorContains(t, err, "strategy exposure")
}
