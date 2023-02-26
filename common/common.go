package common

type Symbol string

const (
	BTCUSDT Symbol = "BTCUSDT"
)

type Order interface {
	GetPrice() float64
	GetQuantity() float64
	GetSum() float64
}

type OrderBook interface {
	GetSymbol() Symbol
	GetAsks() []Order
	GetBids() []Order
}
