package risk

import (
	"fmt"

	"github.com/shopspring/decimal"

	"grid_trade/internal/domain"
)

type Service struct {
	MaxStrategyNotional decimal.Decimal
}

func (s Service) Check(intent domain.OrderIntent) error {
	qty, err := decimal.NewFromString(intent.Quantity)
	if err != nil {
		return err
	}
	price, err := decimal.NewFromString(intent.Price)
	if err != nil {
		return err
	}
	if qty.Mul(price).GreaterThan(s.MaxStrategyNotional) {
		return fmt.Errorf("strategy exposure exceeds limit")
	}
	return nil
}
