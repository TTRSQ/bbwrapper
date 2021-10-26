package bybit

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"time"

	"github.com/TTRSQ/bbwrapper/domains/base"
	"github.com/TTRSQ/bbwrapper/domains/board"
	"github.com/TTRSQ/bbwrapper/domains/order"
	"github.com/TTRSQ/bbwrapper/domains/order/id"
	"github.com/TTRSQ/bbwrapper/domains/stock"
	"github.com/TTRSQ/bbwrapper/interface/exchange"
)

type bybit struct {
	name       string
	host       string
	key        exchange.Key
	httpClient *http.Client
}

// New return exchange obj.
func New(key exchange.Key) (exchange.Exchange, error) {
	bb := bybit{}
	bb.name = "bybit"
	bb.host = "api.bybit.com"

	if key.APIKey == "" || key.APISecKey == "" {
		return nil, errors.New("APIKey and APISecKey Required")
	}
	bb.key = key

	bb.httpClient = new(http.Client)
	if key.SpecificParam["timeoutMS"] != nil {
		bb.httpClient.Timeout = time.Duration(key.SpecificParam["timeoutMS"].(int)) * time.Millisecond
	}

	return &bb, nil
}

func (bb *bybit) ExchangeName() string {
	return bb.name
}

func (bb *bybit) OrderTypes() exchange.OrderTypes {
	return exchange.OrderTypes{
		Limit:  "Limit",
		Market: "Market",
	}
}

func (bb *bybit) CreateOrder(price, size float64, isBuy bool, symbol, orderType string) (*order.Responce, error) {
	type Req struct {
		Side        string  `json:"side"`
		Symbol      string  `json:"symbol"`
		OrderType   string  `json:"order_type"`
		Qty         float64 `json:"qty"`
		Price       float64 `json:"price"`
		TimeInForce string  `json:"time_in_force"`
	}

	res, err := bb.postRequest("/v2/private/order/create", structToMap(&Req{
		Symbol:      symbol,
		OrderType:   orderType,
		Side:        map[bool]string{true: "Buy", false: "Sell"}[isBuy],
		Price:       map[bool]float64{true: price, false: 0}[orderType == bb.OrderTypes().Limit],
		Qty:         size,
		TimeInForce: "GoodTillCancel",
	}))

	if err != nil {
		return nil, err
	}

	// レスポンスの変換
	type Res struct {
		RetCode int    `json:"ret_code"`
		RetMsg  string `json:"ret_msg"`
		ExtCode string `json:"ext_code"`
		ExtInfo string `json:"ext_info"`
		Result  struct {
			UserID        int       `json:"user_id"`
			OrderID       string    `json:"order_id"`
			Symbol        string    `json:"symbol"`
			Side          string    `json:"side"`
			OrderType     string    `json:"order_type"`
			Price         int       `json:"price"`
			Qty           int       `json:"qty"`
			TimeInForce   string    `json:"time_in_force"`
			OrderStatus   string    `json:"order_status"`
			LastExecTime  int       `json:"last_exec_time"`
			LastExecPrice int       `json:"last_exec_price"`
			LeavesQty     int       `json:"leaves_qty"`
			CumExecQty    int       `json:"cum_exec_qty"`
			CumExecValue  int       `json:"cum_exec_value"`
			CumExecFee    int       `json:"cum_exec_fee"`
			RejectReason  string    `json:"reject_reason"`
			OrderLinkID   string    `json:"order_link_id"`
			CreatedAt     time.Time `json:"created_at"`
			UpdatedAt     time.Time `json:"updated_at"`
		} `json:"result"`
		TimeNow          string `json:"time_now"`
		RateLimitStatus  int    `json:"rate_limit_status"`
		RateLimitResetMs int64  `json:"rate_limit_reset_ms"`
		RateLimit        int    `json:"rate_limit"`
	}

	resData := Res{}
	json.Unmarshal(res, &resData)

	return &order.Responce{
		ID:         id.NewID(bb.name, symbol, fmt.Sprint(resData.Result.OrderID)),
		FilledSize: size - float64(resData.Result.LeavesQty),
	}, nil
}

func (bb *bybit) LiquidationOrder(price, size float64, isBuy bool, symbol, orderType string) (*order.Responce, error) {
	return nil, errors.New("LiquidationOrder not supported.")
}

func (bb *bybit) OpenInterest(symbol string, minute, limit int) ([]base.OpenInterest, error) {
	// リクエスト
	type Req struct {
		Symbol string `json:"symbol"`
		Period string `json:"period"`
		Limit  string `json:"limit"`
	}
	res, err := bb.getRequest("/v2/public/open-interest", structToMap(&Req{
		Symbol: symbol,
		Period: fmt.Sprint(minute) + "min",
		Limit:  fmt.Sprint(limit),
	}))
	if err != nil {
		return []base.OpenInterest{}, err
	}

	// レスポンスの変換
	type Res struct {
		RetCode int    `json:"ret_code"`
		RetMsg  string `json:"ret_msg"`
		ExtCode string `json:"ext_code"`
		ExtInfo string `json:"ext_info"`
		Result  []struct {
			OpenInterest int    `json:"open_interest"`
			Timestamp    int    `json:"timestamp"`
			Symbol       string `json:"symbol"`
		} `json:"result"`
		TimeNow string `json:"time_now"`
	}
	resData := Res{}
	json.Unmarshal(res, &resData)
	if resData.RetMsg != "OK" {
		return []base.OpenInterest{}, errors.New(resData.RetMsg + ":" + resData.ExtCode)
	}

	ret := []base.OpenInterest{}
	for i := range resData.Result {
		item := resData.Result[i]
		ret = append(ret, base.OpenInterest{
			OpenInterest: item.OpenInterest,
			Timestamp:    item.Timestamp,
		})
	}

	return ret, nil
}

func (bb *bybit) EditOrder(symbol, localID string, price, size float64) (*order.Order, error) {
	// リクエスト
	type Req struct {
		OrderID string `json:"order_id"`
		Symbol  string `json:"symbol"`
		Qty     string `json:"p_r_qty"`
		Price   string `json:"p_r_price"`
	}
	res, err := bb.postRequest("/v2/private/order/replace", structToMap(&Req{
		OrderID: localID,
		Symbol:  symbol,
		Qty:     fmt.Sprint(size),
		Price:   fmt.Sprint(price),
	}))
	if err != nil {
		return nil, err
	}

	// レスポンスの変換
	type Res struct {
		RetCode int    `json:"ret_code"`
		RetMsg  string `json:"ret_msg"`
		ExtCode string `json:"ext_code"`
		Result  struct {
			OrderID string `json:"order_id"`
		} `json:"result"`
		TimeNow          string `json:"time_now"`
		RateLimitStatus  int    `json:"rate_limit_status"`
		RateLimitResetMs int64  `json:"rate_limit_reset_ms"`
		RateLimit        int    `json:"rate_limit"`
	}
	resData := Res{}
	json.Unmarshal(res, &resData)
	if resData.RetMsg != "OK" {
		return nil, errors.New(resData.RetMsg + ":" + resData.ExtCode)
	}
	t, _ := strconv.ParseFloat(resData.TimeNow, 64)
	return &order.Order{
		ID:            id.NewID(bb.name, symbol, resData.Result.OrderID),
		Request:       order.Request{},
		UpdatedAtUnix: int(t),
	}, nil
}

func (bb *bybit) CancelOrder(symbol, localID string) error {
	type Req struct {
		Symbol  string `json:"symbol"`
		OrderID string `json:"order_id"`
	}

	type Res struct {
		RetCode int    `json:"ret_code"`
		RetMsg  string `json:"ret_msg"`
		ExtCode string `json:"ext_code"`
		ExtInfo string `json:"ext_info"`
		Result  struct {
			UserID        int       `json:"user_id"`
			OrderID       string    `json:"order_id"`
			Symbol        string    `json:"symbol"`
			Side          string    `json:"side"`
			OrderType     string    `json:"order_type"`
			Price         int       `json:"price"`
			Qty           int       `json:"qty"`
			TimeInForce   string    `json:"time_in_force"`
			OrderStatus   string    `json:"order_status"`
			LastExecTime  int       `json:"last_exec_time"`
			LastExecPrice int       `json:"last_exec_price"`
			LeavesQty     int       `json:"leaves_qty"`
			CumExecQty    int       `json:"cum_exec_qty"`
			CumExecValue  int       `json:"cum_exec_value"`
			CumExecFee    int       `json:"cum_exec_fee"`
			RejectReason  string    `json:"reject_reason"`
			OrderLinkID   string    `json:"order_link_id"`
			CreatedAt     time.Time `json:"created_at"`
			UpdatedAt     time.Time `json:"updated_at"`
		} `json:"result"`
		TimeNow          string `json:"time_now"`
		RateLimitStatus  int    `json:"rate_limit_status"`
		RateLimitResetMs int64  `json:"rate_limit_reset_ms"`
		RateLimit        int    `json:"rate_limit"`
	}
	res, err := bb.postRequest("/v2/private/order/cancel", structToMap(&Req{
		Symbol:  symbol,
		OrderID: localID,
	}))
	if err != nil {
		return err
	}

	resData := Res{}
	json.Unmarshal(res, &resData)

	return nil
}

func (bb *bybit) CancelAllOrder(symbol string) error {
	type Req struct {
		Symbol string `json:"symbol"`
	}

	_, err := bb.postRequest("/v2/private/order/cancelAll", structToMap(&Req{
		Symbol: symbol,
	}))

	return err
}

func (bb *bybit) ActiveOrders(symbol string) ([]order.Order, error) {
	type Req struct {
		Symbol      string `json:"symbol"`
		OrderStatus string `json:"order_status"`
	}
	res, err := bb.getRequest("/v2/private/order/list", structToMap(&Req{
		Symbol:      symbol,
		OrderStatus: "Created,New,PartiallyFilled", // 今後の取引に関わるもののみ
	}))
	if err != nil {
		return []order.Order{}, err
	}
	type Res struct {
		RetCode int    `json:"ret_code"`
		RetMsg  string `json:"ret_msg"`
		ExtCode string `json:"ext_code"`
		ExtInfo string `json:"ext_info"`
		Result  struct {
			Data []struct {
				UserID       int       `json:"user_id"`
				OrderStatus  string    `json:"order_status"`
				Symbol       string    `json:"symbol"`
				Side         string    `json:"side"`
				OrderType    string    `json:"order_type"`
				Price        string    `json:"price"`
				Qty          string    `json:"qty"`
				TimeInForce  string    `json:"time_in_force"`
				OrderLinkID  string    `json:"order_link_id"`
				OrderID      string    `json:"order_id"`
				CreatedAt    time.Time `json:"created_at"`
				UpdatedAt    time.Time `json:"updated_at"`
				LeavesQty    string    `json:"leaves_qty"`
				LeavesValue  string    `json:"leaves_value"`
				CumExecQty   string    `json:"cum_exec_qty"`
				CumExecValue string    `json:"cum_exec_value"`
				CumExecFee   string    `json:"cum_exec_fee"`
				RejectReason string    `json:"reject_reason"`
			} `json:"data"`
			Cursor string `json:"cursor"`
		} `json:"result"`
		TimeNow          string `json:"time_now"`
		RateLimitStatus  int    `json:"rate_limit_status"`
		RateLimitResetMs int64  `json:"rate_limit_reset_ms"`
		RateLimit        int    `json:"rate_limit"`
	}
	resData := Res{}
	json.Unmarshal(res, &resData)

	orders := []order.Order{}
	for _, v := range resData.Result.Data {
		price, _ := strconv.ParseFloat(v.Price, 64)
		size, _ := strconv.ParseFloat(v.Side, 64)
		orders = append(orders, order.Order{
			ID: id.NewID(bb.name, symbol, v.OrderID),
			Request: order.Request{
				Norm: base.Norm{
					Price: price,
					Size:  size,
				},
				Symbol:    symbol,
				IsBuy:     v.Side == "Buy",
				OrderType: v.OrderType,
			},
			UpdatedAtUnix: int(v.UpdatedAt.Unix()),
		})
	}

	return orders, nil
}

func (bb *bybit) Stocks(symbol string) (stock.Stock, error) {
	type Req struct {
		Symbol string `json:"symbol"`
	}
	res, err := bb.getRequest("/v2/private/position/list", structToMap(&Req{
		Symbol: symbol,
	}))
	if err != nil {
		return stock.Stock{}, err
	}
	// レスポンスの変換
	type Res struct {
		RetCode int    `json:"ret_code"`
		RetMsg  string `json:"ret_msg"`
		ExtCode string `json:"ext_code"`
		ExtInfo string `json:"ext_info"`
		Result  struct {
			ID                  int       `json:"id"`
			UserID              int       `json:"user_id"`
			RiskID              int       `json:"risk_id"`
			Symbol              string    `json:"symbol"`
			Side                string    `json:"side"`
			Size                float64   `json:"size"`
			PositionValue       string    `json:"position_value"`
			EntryPrice          string    `json:"entry_price"`
			IsIsolated          bool      `json:"is_isolated"`
			AutoAddMargin       int       `json:"auto_add_margin"`
			Leverage            string    `json:"leverage"`
			EffectiveLeverage   string    `json:"effective_leverage"`
			PositionMargin      string    `json:"position_margin"`
			LiqPrice            string    `json:"liq_price"`
			BustPrice           string    `json:"bust_price"`
			OccClosingFee       string    `json:"occ_closing_fee"`
			OccFundingFee       string    `json:"occ_funding_fee"`
			TakeProfit          string    `json:"take_profit"`
			StopLoss            string    `json:"stop_loss"`
			TrailingStop        string    `json:"trailing_stop"`
			PositionStatus      string    `json:"position_status"`
			DeleverageIndicator int       `json:"deleverage_indicator"`
			OcCalcData          string    `json:"oc_calc_data"`
			OrderMargin         string    `json:"order_margin"`
			WalletBalance       string    `json:"wallet_balance"`
			RealisedPnl         string    `json:"realised_pnl"`
			UnrealisedPnl       int       `json:"unrealised_pnl"`
			CumRealisedPnl      string    `json:"cum_realised_pnl"`
			CrossSeq            int       `json:"cross_seq"`
			PositionSeq         int       `json:"position_seq"`
			CreatedAt           time.Time `json:"created_at"`
			UpdatedAt           time.Time `json:"updated_at"`
		} `json:"result"`
		TimeNow          string `json:"time_now"`
		RateLimitStatus  int    `json:"rate_limit_status"`
		RateLimitResetMs int64  `json:"rate_limit_reset_ms"`
		RateLimit        int    `json:"rate_limit"`
	}
	resData := Res{}
	json.Unmarshal(res, &resData)

	size := resData.Result.Size
	sizeAbs := math.Abs(size)
	if resData.Result.Side == "Sell" {
		size *= -1
	}

	stock := stock.Stock{Symbol: symbol, Summary: size}
	if size > 0 {
		stock.LongSize = sizeAbs
	} else {
		stock.ShortSize = sizeAbs
	}

	return stock, nil
}

func (bb *bybit) Balance() ([]base.Balance, error) {
	res, err := bb.getRequest("/v2/private/wallet/balance", map[string]string{})
	if err != nil {
		return []base.Balance{}, err
	}
	type Res struct {
		RetCode int    `json:"ret_code"`
		RetMsg  string `json:"ret_msg"`
		ExtCode string `json:"ext_code"`
		ExtInfo string `json:"ext_info"`
		Result  map[string]struct {
			Equity           int     `json:"equity"`
			AvailableBalance float64 `json:"available_balance"`
			UsedMargin       float64 `json:"used_margin"`
			OrderMargin      float64 `json:"order_margin"`
			PositionMargin   int     `json:"position_margin"`
			OccClosingFee    int     `json:"occ_closing_fee"`
			OccFundingFee    int     `json:"occ_funding_fee"`
			WalletBalance    int     `json:"wallet_balance"`
			RealisedPnl      int     `json:"realised_pnl"`
			UnrealisedPnl    int     `json:"unrealised_pnl"`
			CumRealisedPnl   int     `json:"cum_realised_pnl"`
			GivenCash        int     `json:"given_cash"`
			ServiceCash      int     `json:"service_cash"`
		} `json:"result"`
		TimeNow          string `json:"time_now"`
		RateLimitStatus  int    `json:"rate_limit_status"`
		RateLimitResetMs int64  `json:"rate_limit_reset_ms"`
		RateLimit        int    `json:"rate_limit"`
	}
	resData := Res{}
	json.Unmarshal(res, &resData)

	balances := []base.Balance{}
	for k, v := range resData.Result {
		balances = append(balances, base.Balance{
			CurrencyCode: k,
			Size:         v.AvailableBalance,
		})
	}

	return balances, nil
}

func (bb *bybit) Boards(symbol string) (board.Board, error) {
	type Req struct {
		Symbol string `json:"symbol"`
	}
	res, err := bb.getRequest("/v2/public/orderBook/L2", structToMap(&Req{
		Symbol: symbol,
	}))
	if err != nil {
		return board.Board{}, err
	}
	// レスポンスの変換
	type Res struct {
		RetCode int    `json:"ret_code"`
		RetMsg  string `json:"ret_msg"`
		ExtCode string `json:"ext_code"`
		ExtInfo string `json:"ext_info"`
		Result  []struct {
			Symbol string `json:"symbol"`
			Price  string `json:"price"`
			Size   int    `json:"size"`
			Side   string `json:"side"`
		} `json:"result"`
		TimeNow string `json:"time_now"`
	}
	resData := Res{}
	json.Unmarshal(res, &resData)

	asks := []base.Norm{}
	bids := []base.Norm{}

	for _, v := range resData.Result {
		price, _ := strconv.ParseFloat(v.Price, 64)
		if v.Side == "Buy" {
			bids = append(bids, base.Norm{
				Price: price,
				Size:  float64(v.Size),
			})
		} else {
			asks = append(asks, base.Norm{
				Price: price,
				Size:  float64(v.Size),
			})
		}
	}

	sort.Slice(asks, func(i, j int) bool {
		return asks[i].Price < asks[j].Price
	})
	sort.Slice(bids, func(i, j int) bool {
		return bids[i].Price > bids[j].Price
	})
	midPrice := (bids[0].Price + asks[0].Price) / 2

	return board.Board{
		ExchangeName: bb.name,
		Symbol:       symbol,
		MidPrice:     midPrice,
		Asks:         asks,
		Bids:         bids,
	}, nil
}

func (bb *bybit) InScheduledMaintenance() bool {
	// TODO
	return false
}

func (bb *bybit) postRequest(path string, param map[string]string) ([]byte, error) {
	param["api_key"] = bb.key.APIKey
	param["timestamp"] = fmt.Sprint(time.Now().UnixNano() / 1000000)
	sign := getSignature(param, bb.key.APISecKey)
	param["sign"] = sign

	url := url.URL{Scheme: "https", Host: bb.host, Path: path}
	jsonParam, _ := json.Marshal(param)
	req, _ := http.NewRequest(
		"POST",
		url.String(),
		bytes.NewBuffer(jsonParam),
	)
	req.Header.Add("Content-Type", "application/json")

	return bb.request(req)
}

func (bb *bybit) getRequest(path string, param map[string]string) ([]byte, error) {
	param["api_key"] = bb.key.APIKey
	param["timestamp"] = fmt.Sprint(time.Now().UnixNano() / 1000000)
	sign := getSignature(param, bb.key.APISecKey)
	queryStr := getQuery(param) + "&sign=" + sign

	url := url.URL{Scheme: "https", Host: bb.host, Path: path}
	req, _ := http.NewRequest(
		"GET",
		url.String()+"?"+queryStr,
		nil, //bytes.NewBuffer([]byte(queryStr)),
	)

	return bb.request(req)
}

func getSignature(params map[string]string, key string) string {
	_val := getQuery(params)
	h := hmac.New(sha256.New, []byte(key))
	io.WriteString(h, _val)

	return fmt.Sprintf("%x", h.Sum(nil))
}

func getQuery(params map[string]string) string {
	keys := make([]string, len(params))
	i := 0
	_val := ""
	for k := range params {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	for _, k := range keys {
		_val += k + "=" + params[k] + "&"
	}
	return _val[0 : len(_val)-1]
}

func structToMap(data interface{}) map[string]string {
	result := make(map[string]string)
	elem := reflect.ValueOf(data).Elem()
	size := elem.NumField()

	for i := 0; i < size; i++ {
		field := elem.Type().Field(i).Tag.Get("json")
		value := elem.Field(i).Interface()
		result[field] = fmt.Sprint(value)
	}
	return result
}

func (bb *bybit) request(req *http.Request) ([]byte, error) {
	resp, err := bb.httpClient.Do(req)

	if err != nil {
		errStr := fmt.Sprintf("err ==> %+v\nreq ==> %v\n", err, req)
		return nil, errors.New(errStr)
	}
	if resp.StatusCode/100 != 2 {
		body, _ := ioutil.ReadAll(resp.Body)
		errStr := fmt.Sprintf("code:%d\n", resp.StatusCode)
		errStr += fmt.Sprintf("body ==> %s\n", string(body))
		errStr += fmt.Sprintf("resp ==> %+v\nreq ==> %v\n", resp, req)
		return nil, errors.New(errStr)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	type errCheck struct {
		Fail    int    `json:"ret_code"`
		Message string `json:"ret_msg"`
		Code    string `json:"ext_code"`
		Info    string `json:"ext_info"`
	}
	check := errCheck{}
	err = json.Unmarshal(body, &check)
	if check.Fail != 0 || err != nil {
		msg := fmt.Sprintf("msg: %s\n info: %s\n code: %s\n", check.Message, check.Info, check.Code)
		if err != nil {
			msg = err.Error()
		}
		return nil, errors.New(msg)
	}

	return body, err
}

func (bb *bybit) UpdateLTP(lastTimePrice float64) error {
	return errors.New("not supported.")
}

func (bb *bybit) UpdateBestPrice(bestAsk, bestBid float64) error {
	return errors.New("not supported.")
}
