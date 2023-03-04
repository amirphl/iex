package account

type Balance interface {
	Asset() string
	FAName() string
	Fiat() bool
	Value() float64
	Locked() float64
}
