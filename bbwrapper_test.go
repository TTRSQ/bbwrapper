package bbwrapper

import (
	"testing"
)

func TestNew(t *testing.T) {
	name := "bybit"

	bf, err := New(ExchangeKey{
		APIKey:    "hoge",
		APISecKey: "fuga",
	})

	if err != nil {
		t.Errorf(" %s\n", err.Error())
	}

	if bf.ExchangeName() != name {
		t.Error(bf.ExchangeName() + " != " + name)
	}
}
