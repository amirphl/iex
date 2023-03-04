package account

type FeeRate interface {
	Symbol() string
	MakerFeeRate() float64
	TakerFeeRate() float64
}
