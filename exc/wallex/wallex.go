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
	ORDER_BOOK_URL      = "https://api.wallex.ir/v1/depth?symbol=%s"
	ALL_ORDER_BOOKS_URL = "https://api.wallex.ir/v2/depth/all"
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

func parseHTTPRespBody(body io.Reader) (interface{}, error) {
	var rawData interface{}
	err := json.NewDecoder(body).Decode(&rawData)

	return rawData, err
}

func extractSuccess(rawData interface{}) bool {
	success := reflect.ValueOf(rawData).
		MapIndex(reflect.ValueOf("success")).
		Elem().
		Bool()

	return success
}

func extractMessage(rawData interface{}) string {
	message := reflect.ValueOf(rawData).
		MapIndex(reflect.ValueOf("message")).
		Elem().
		String()

	return message
}

func extractRawResult(rawData interface{}) reflect.Value {
	rawOrderBook := reflect.ValueOf(rawData)
	rawResult := rawOrderBook.MapIndex(reflect.ValueOf("result")).Elem()

	return rawResult
}

func parseRawOrder(rawOrder reflect.Value) common.Order {
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

func parseRawOrderBook(rawResult reflect.Value, symbol common.Symbol) common.OrderBook {
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
		symbol: symbol,
		asks:   asks,
		bids:   bids,
	}
}

func parseRawOrderBooks(rawData interface{}) []common.OrderBook {
	rawOrderBooks := reflect.ValueOf(rawData)
	rawResult := rawOrderBooks.MapIndex(reflect.ValueOf("result")).Elem()

	res := make([]common.OrderBook, rawResult.Len())

	iter := rawResult.MapRange()
	i := 0

	for iter.Next() {
		rawSymbol := iter.Key()
		symbol := common.Symbol(rawSymbol.String())
		rawOrderBook := iter.Value().Elem()
		orderBook := parseRawOrderBook(rawOrderBook, symbol)

		res[i] = orderBook
		i++
	}

	return res
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

	rawData, err := parseHTTPRespBody(resp.Body)

	if err != nil {
		return nil, err
	}

	if success := extractSuccess(rawData); !success {
		message := extractMessage(rawData)
		return nil, fmt.Errorf("failed to get orderbook: %s", message)
	}

	rawResult := extractRawResult(rawData)
	res := parseRawOrderBook(rawResult, symbol)

	return res, nil
}

func GetAllOrderBooks(apiKey string) ([]common.OrderBook, error) {
	url := ALL_ORDER_BOOKS_URL
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

	if success := extractSuccess(rawData); !success {
		message := extractMessage(rawData)
		return nil, fmt.Errorf("failed to get all orderbooks: %s", message)
	}

	res := parseRawOrderBooks(rawData)

	return res, nil
}
