package order

type Order interface {
	Price() float64
	Quantity() float64
}

type OrderBook interface {
	Symbol() string
	Asks() []Order
	Bids() []Order
}
