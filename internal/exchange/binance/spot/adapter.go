package spot

import (
	"encoding/json"

	"grid_trade/internal/domain"
	"grid_trade/internal/exchange/binance/common"
)

type Adapter struct{}

func (a Adapter) NormalizeOrderAck(raw []byte) (domain.OrderEvent, error) {
	var ack common.WSOrderAck
	if err := json.Unmarshal(raw, &ack); err != nil {
		return domain.OrderEvent{}, err
	}

	return domain.OrderEvent{
		ClientOrderID: ack.Result.ClientOrderID,
		Symbol:        ack.Result.Symbol,
		Status:        "ACK",
	}, nil
}
