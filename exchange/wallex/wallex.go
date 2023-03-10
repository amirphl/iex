package wallex

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"

	"github.com/amirphl/iex/account"
	"github.com/amirphl/iex/order"
)

var client http.Client

const (
	orderBookURL     = "https://api.wallex.ir/v1/depth?symbol=%s"
	allOrderBooksURL = "https://api.wallex.ir/v2/depth/all"
	feeRateURL       = "https://api.wallex.ir/v1/account/fee"
	balanceURL       = "https://api.wallex.ir/v1/account/balances"
)

type order_ struct {
	price    float64
	quantity float64
	sum      float64
}

type orderBook struct {
	symbol string
	asks   []order.Order
	bids   []order.Order
}

type feeRate struct {
	symbol        string
	makerFeeRate  float64
	takerFeeRate  float64
	recentDaysSum float64
}

type balance struct {
	asset  string
	faName string
	fiat   bool
	value  float64
	locked float64
}

func (o *order_) Price() float64 {
	return o.price
}

func (o *order_) Quantity() float64 {
	return o.quantity
}

func (o *order_) Sum() float64 {
	return o.sum
}

func (o *orderBook) Symbol() string {
	return o.symbol
}

func (o *orderBook) Asks() []order.Order {
	return o.asks
}

func (o *orderBook) Bids() []order.Order {
	return o.bids
}

func (f *feeRate) Symbol() string {
	return f.symbol
}

func (f *feeRate) MakerFeeRate() float64 {
	return f.makerFeeRate
}

func (f *feeRate) TakerFeeRate() float64 {
	return f.takerFeeRate
}

func (f *feeRate) RecentDaysSum() float64 {
	return f.recentDaysSum
}

func (b *balance) Asset() string {
	return b.asset
}

func (b *balance) FAName() string {
	return b.faName
}

func (b *balance) Fiat() bool {
	return b.fiat
}

func (b *balance) Value() float64 {
	return b.value
}

func (b *balance) Locked() float64 {
	return b.locked
}

func sendHTTPRequest(method, url string, body io.Reader, apiKey string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)

	if err != nil {
		return &http.Response{}, err
	}

	req.Header.Set("X-API-Key", apiKey)

	return client.Do(req)
}

func parseHTTPRespBody(body io.Reader) (interface{}, error) {
	var rawData interface{}
	err := json.NewDecoder(body).Decode(&rawData)

	return rawData, err
}

func extractKey(rawData reflect.Value, key string) reflect.Value {
	val := rawData.MapIndex(reflect.ValueOf(key)).Elem()

	return val
}

func parseRawOrder(rawOrder reflect.Value) order.Order {
	rawPrice := rawOrder.MapIndex(reflect.ValueOf("price")).Elem()
	quantity := rawOrder.MapIndex(reflect.ValueOf("quantity")).Elem().Float()
	rawSum := rawOrder.MapIndex(reflect.ValueOf("sum")).Elem()

	price, _ := strconv.ParseFloat(rawPrice.String(), 64)
	sum, _ := strconv.ParseFloat(rawSum.String(), 64)

	return &order_{
		price:    price,
		quantity: quantity,
		sum:      sum,
	}
}

func parseRawFeeRate(rawFeeRate reflect.Value, symbol string) account.FeeRate {
	rawMaker := rawFeeRate.MapIndex(reflect.ValueOf("makerFeeRate")).Elem()
	rawTaker := rawFeeRate.MapIndex(reflect.ValueOf("takerFeeRate")).Elem()
	recentDaysSum := rawFeeRate.MapIndex(reflect.ValueOf("recent_days_sum")).Elem().Float()

	maker, _ := strconv.ParseFloat(rawMaker.String(), 64)
	taker, _ := strconv.ParseFloat(rawTaker.String(), 64)

	return &feeRate{
		symbol:        symbol,
		makerFeeRate:  maker,
		takerFeeRate:  taker,
		recentDaysSum: recentDaysSum,
	}
}

func parseRawBalance(rawBalance reflect.Value) account.Balance {
	asset := rawBalance.MapIndex(reflect.ValueOf("asset")).Elem().String()
	faName := rawBalance.MapIndex(reflect.ValueOf("faName")).Elem().String()
	fiat := rawBalance.MapIndex(reflect.ValueOf("fiat")).Elem().Bool()
	rawValue := rawBalance.MapIndex(reflect.ValueOf("value")).Elem()
	rawLocked := rawBalance.MapIndex(reflect.ValueOf("locked")).Elem()

	value, _ := strconv.ParseFloat(rawValue.String(), 64)
	locked, _ := strconv.ParseFloat(rawLocked.String(), 64)

	return &balance{
		asset:  asset,
		faName: faName,
		fiat:   fiat,
		value:  value,
		locked: locked,
	}
}

func parseRawOrderBook(rawOrderBook reflect.Value, symbol string) order.OrderBook {
	rawAsks := rawOrderBook.MapIndex(reflect.ValueOf("ask")).Elem()
	rawBids := rawOrderBook.MapIndex(reflect.ValueOf("bid")).Elem()

	asks := make([]order.Order, rawAsks.Len())
	bids := make([]order.Order, rawBids.Len())

	for i := 0; i < rawAsks.Len(); i++ {
		v := rawAsks.Index(i).Elem()
		asks[i] = parseRawOrder(v)
	}

	for i := 0; i < rawBids.Len(); i++ {
		v := rawBids.Index(i).Elem()
		bids[i] = parseRawOrder(v)
	}

	return &orderBook{
		symbol: symbol,
		asks:   asks,
		bids:   bids,
	}
}

func parseRawOrderBooks(rawOrderBooks reflect.Value) map[string]order.OrderBook {
	res := make(map[string]order.OrderBook, rawOrderBooks.Len())

	iter := rawOrderBooks.MapRange()

	for iter.Next() {
		symbol := iter.Key().String()
		rawOrderBook := iter.Value().Elem()
		orderBook := parseRawOrderBook(rawOrderBook, symbol)

		res[symbol] = orderBook
	}

	return res
}

func parseRawFeeRates(rawFeeRates reflect.Value) map[string]account.FeeRate {
	res := make(map[string]account.FeeRate, rawFeeRates.Len())

	iter := rawFeeRates.MapRange()

	for iter.Next() {
		symbol := iter.Key().String()

		if symbol == "default" || symbol == "metaData" {
			// TODO Include in FeeRate?
			continue
		}

		rawFeeRate := iter.Value().Elem()
		feeRate := parseRawFeeRate(rawFeeRate, symbol)

		res[symbol] = feeRate
	}

	return res
}

func parseRawBalances(rawBalances reflect.Value) map[string]account.Balance {
	res := make(map[string]account.Balance, rawBalances.Len())

	iter := rawBalances.MapRange()

	for iter.Next() {
		asset := iter.Key().String()
		rawBalance := iter.Value().Elem()
		balance := parseRawBalance(rawBalance)

		res[asset] = balance
	}

	return res
}

func OrderBook(symbol string, apiKey string) (order.OrderBook, error) {
	url := fmt.Sprintf(orderBookURL, symbol)
	resp, err := sendHTTPRequest("GET", url, nil, apiKey)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get orderbook: %d", resp.StatusCode)
	}

	rawData, err := parseHTTPRespBody(resp.Body)

	if err != nil {
		return nil, err
	}

	refData := reflect.ValueOf(rawData)

	if success := extractKey(refData, "success").Bool(); !success {
		message := extractKey(refData, "message").String()
		return nil, fmt.Errorf("failed to get orderbook: %s", message)
	}

	refRes := extractKey(refData, "result")
	book := parseRawOrderBook(refRes, symbol)

	return book, nil
}

func AllOrderBooks(apiKey string) (map[string]order.OrderBook, error) {
	url := allOrderBooksURL
	resp, err := sendHTTPRequest("GET", url, nil, apiKey)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get all orderbooks: %d", resp.StatusCode)
	}

	rawData, err := parseHTTPRespBody(resp.Body)

	if err != nil {
		return nil, err
	}

	refData := reflect.ValueOf(rawData)

	if success := extractKey(refData, "success").Bool(); !success {
		message := extractKey(refData, "message").String()
		return nil, fmt.Errorf("failed to get all orderbooks: %s", message)
	}

	refRes := extractKey(refData, "result")
	books := parseRawOrderBooks(refRes)

	return books, nil
}

// TODO Implement by query param. performance problem
func FeeRate(symbol string, apiKey string) (account.FeeRate, error) {
	feeRates, err := FeeRates(apiKey)

	if err != nil {
		return nil, err
	}

	if val, ok := feeRates[symbol]; ok {
		return val, nil
	}

	return nil, fmt.Errorf("symbol %s not found", symbol)
}

func FeeRates(apiKey string) (map[string]account.FeeRate, error) {
	url := feeRateURL
	resp, err := sendHTTPRequest("GET", url, nil, apiKey)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get fee rates: %d", resp.StatusCode)
	}

	rawData, err := parseHTTPRespBody(resp.Body)

	if err != nil {
		return nil, err
	}

	refData := reflect.ValueOf(rawData)

	if success := extractKey(refData, "success").Bool(); !success {
		message := extractKey(refData, "message").String()
		return nil, fmt.Errorf("failed to get fee rates: %s", message)
	}

	refRes := extractKey(refData, "result")
	feeRates := parseRawFeeRates(refRes)

	return feeRates, nil
}

// TODO Implement by query param. performance problem
func Balance(asset string, apiKey string) (account.Balance, error) {
	balances, err := Balances(apiKey)

	if err != nil {
		return nil, err
	}

	if val, ok := balances[asset]; ok {
		return val, nil
	}

	return nil, fmt.Errorf("asset %s not found", asset)
}

func Balances(apiKey string) (map[string]account.Balance, error) {
	url := balanceURL
	resp, err := sendHTTPRequest("GET", url, nil, apiKey)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get balances: %d", resp.StatusCode)
	}

	rawData, err := parseHTTPRespBody(resp.Body)

	if err != nil {
		return nil, err
	}

	refData := reflect.ValueOf(rawData)

	if success := extractKey(refData, "success").Bool(); !success {
		message := extractKey(refData, "message").String()
		return nil, fmt.Errorf("failed to get balances: %s", message)
	}

	refRes := extractKey(refData, "result")
	refBal := extractKey(refRes, "balances")

	balances := parseRawBalances(refBal)

	return balances, nil
}
