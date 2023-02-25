package common

type Symbol string

const (
	BTCUSDT Symbol = "BTCUSDT"
)

type Order interface {
	GetSymbol() Symbol
	GetPrice() float64
	GetQuantity() float64
	GetSum() float64
}

type OrderBook interface {
	GetAsks() []Order
	GetBids() []Order
}
