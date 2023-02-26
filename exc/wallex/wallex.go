package wallex

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"

	"github.com/amirphl/iex/common"
)

var client http.Client

const (
	GLOBAL_CURRENCIES_STATS_URL = "https://api.wallex.ir/v1/currencies/stats"
	ORDER_BOOK_URL              = "https://api.wallex.ir/v1/depth?symbol=%s"
)

type order struct {
	price    float64
	quantity float64
	sum      float64
}

type orderBook struct {
	symbol common.Symbol
	asks   []common.Order
	bids   []common.Order
}

func (o *order) GetPrice() float64 {
	return o.price
}

func (o *order) GetQuantity() float64 {
	return o.quantity
}

func (o *order) GetSum() float64 {
	return o.sum
}

func (o *orderBook) GetSymbol() common.Symbol {
	return o.symbol
}

func (o *orderBook) GetAsks() []common.Order {
	return o.asks
}

func (o *orderBook) GetBids() []common.Order {
	return o.bids
}

func sendHTTPRequest(method, url string, body io.Reader, apiKey string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)

	if err != nil {
		return &http.Response{}, err
	}

	req.Header.Set("X-API-Key", apiKey)

	return client.Do(req)
}

func parseRawOrder(rawOrder reflect.Value) *order {
	rawPrice := rawOrder.MapIndex(reflect.ValueOf("price")).Elem()
	rawQuantity := rawOrder.MapIndex(reflect.ValueOf("quantity")).Elem()
	rawSum := rawOrder.MapIndex(reflect.ValueOf("sum")).Elem()

	price, _ := strconv.ParseFloat(rawPrice.String(), 64)
	quantity := rawQuantity.Float()
	sum, _ := strconv.ParseFloat(rawSum.String(), 64)

	return &order{
		price:    price,
		quantity: quantity,
		sum:      sum,
	}
}

func parseRawOrderBook(raw interface{}) *orderBook {
	rawOrderBook := reflect.ValueOf(raw)
	rawResult := rawOrderBook.MapIndex(reflect.ValueOf("result")).Elem()

	rawAsks := rawResult.MapIndex(reflect.ValueOf("ask")).Elem()
	rawBids := rawResult.MapIndex(reflect.ValueOf("bid")).Elem()

	asks := make([]common.Order, rawAsks.Len())
	bids := make([]common.Order, rawBids.Len())

	for i := 0; i < rawAsks.Len(); i++ {
		v := rawAsks.Index(i).Elem()
		asks[i] = parseRawOrder(v)
	}

	for i := 0; i < rawBids.Len(); i++ {
		v := rawBids.Index(i).Elem()
		bids[i] = parseRawOrder(v)
	}

	return &orderBook{
		asks: asks,
		bids: bids,
	}
}

func GetOrderBook(symbol common.Symbol, apiKey string) (common.OrderBook, error) {
	url := fmt.Sprintf(ORDER_BOOK_URL, symbol)
	resp, err := sendHTTPRequest("GET", url, nil, apiKey)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get orderbook: %d", resp.StatusCode)
	}

	var rawOrderBook interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawOrderBook); err != nil {
		return nil, err
	}

	success := reflect.ValueOf(rawOrderBook).
		MapIndex(reflect.ValueOf("success")).
		Elem().
		Bool()

	if !success {
		message := reflect.ValueOf(rawOrderBook).
			MapIndex(reflect.ValueOf("message")).
			Elem().
			String()
		return nil, fmt.Errorf("failed to get orderbook: %s", message)
	}

	res := parseRawOrderBook(rawOrderBook)
	res.symbol = symbol

	return res, nil
}
