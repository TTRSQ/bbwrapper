# bbwrapper
Crypto Currency Exchange Wrapper for Go.

# Supporting exchanges
- bitbank
- bitflyer
- bybit
- coincheck
- ftx
- gmo
- liquid
- dummy
  - for back test

# Usage
```
import (
	"fmt"
	"time"

	"github.com/TTRSQ/bbwrapper"
)

func main() {
	bfClient, _ := bbwrapper.Bitflyer(bbwrapper.ExchangeKey{
		APIKey:    "your_api_key",
		APISecKey: "your_api_sec_key",
	})

	// create order
	orderID, _ := bfClient.CreateOrder(
		950000, 0.01, true,
		bfClient.Symbols().FxBtcJpy,
		bfClient.OrderTypes().Limit,
	)
	fmt.Printf("%+v\n", orderID)

	// wait for server processing.
	time.Sleep(time.Second * 2)

	// get my order
	orders, _ := bfClient.ActiveOrders(bfClient.Symbols().FxBtcJpy)
	fmt.Printf("%+v\n", orders)

	// cancel order
	_ = bfClient.CancelOrder(
		bfClient.Symbols().FxBtcJpy,
		orderID.LocalID,
	)
}
```
