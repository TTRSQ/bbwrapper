package bbwrapper

import (
	"github.com/TTRSQ/bbwrapper/interface/exchange"
	"github.com/TTRSQ/bbwrapper/src/bybit"
)

// ExchangeKey ..
type ExchangeKey = exchange.Key

// ByBit .. no SpecificParam.
func New(key exchange.Key) (exchange.Exchange, error) {
	return bybit.New(key)
}
