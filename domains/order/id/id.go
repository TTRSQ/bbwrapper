package id

import "fmt"

// ID GlobalID = "{ExchangeName}::{Symbol}::{LocalID}" = "bitflyer::FX_BTC_JPY::001"
type ID struct {
	ExchangeName string
	Symbol       string
	LocalID      string
}

// NewID .. make id obj.
func NewID(exchange, symbol, localID string) ID {
	return ID{
		ExchangeName: exchange,
		Symbol:       symbol,
		LocalID:      localID,
	}
}

// ToString return globalID with string.
func (i *ID) ToString() string {
	return fmt.Sprintf("%s::%s::%s", i.ExchangeName, i.Symbol, i.LocalID)
}
