package wallex

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"

	"github.com/amirphl/iex/order"
)

var client http.Client

const (
	OrderBookURL     = "https://api.wallex.ir/v1/depth?symbol=%s"
	AllOrderBooksURL = "https://api.wallex.ir/v2/depth/all"
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
	rawQuantity := rawOrder.MapIndex(reflect.ValueOf("quantity")).Elem()
	rawSum := rawOrder.MapIndex(reflect.ValueOf("sum")).Elem()

	price, _ := strconv.ParseFloat(rawPrice.String(), 64)
	quantity := rawQuantity.Float()
	sum, _ := strconv.ParseFloat(rawSum.String(), 64)

	return &order_{
		price:    price,
		quantity: quantity,
		sum:      sum,
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

func parseRawOrderBooks(rawOrderBooks reflect.Value) []order.OrderBook {
	res := make([]order.OrderBook, rawOrderBooks.Len())

	iter := rawOrderBooks.MapRange()
	i := 0

	for iter.Next() {
		symbol := iter.Key().String()
		rawOrderBook := iter.Value().Elem()
		orderBook := parseRawOrderBook(rawOrderBook, symbol)

		res[i] = orderBook
		i++
	}

	return res
}

func GetOrderBook(symbol string, apiKey string) (order.OrderBook, error) {
	url := fmt.Sprintf(OrderBookURL, symbol)
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

func GetAllOrderBooks(apiKey string) ([]order.OrderBook, error) {
	url := AllOrderBooksURL
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
