package base

// Norm norm of something (e.g. Order, Position, Stock)
type Norm struct {
	Price float64
	Size  float64
}

// Balance of Currency
type Balance struct {
	CurrencyCode string
	Size         float64
}

type OpenInterest struct {
	OpenInterest int
	Timestamp    int
}
