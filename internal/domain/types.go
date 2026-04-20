package domain

type MarketType string

const (
	MarketSpot      MarketType = "spot"
	MarketFuturesUM MarketType = "futures_um"
)

type PositionKey struct {
	Market       MarketType
	Symbol       string
	PositionSide string
}

type OrderIntent struct {
	StrategyID    string
	Market        MarketType
	Symbol        string
	PositionSide  string
	Side          string
	OrderType     string
	Price         string
	Quantity      string
	TimeInForce   string
	ReduceOnly    bool
	ClientOrderID string
	Reason        string
}

type OrderRecord struct {
	ClientOrderID string
	Symbol        string
	Status        string
}

type OrderEvent struct {
	ClientOrderID string
	Symbol        string
	Status        string
}

type PositionSnapshot struct {
	Symbol       string
	PositionSide string
	Quantity     string
}

type Balance struct {
	Asset  string
	Free   string
	Locked string
}
