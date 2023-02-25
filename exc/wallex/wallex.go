package wallex

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/amirphl/iex/common"
)

var client http.Client

const (
	GLOBAL_CURRENCIES_STATS_URL = "https://api.wallex.ir/v1/currencies/stats"
	ORDER_BOOK_URL              = "https://api.wallex.ir/v1/depth?symbol=%s"
)

type order struct {
	Symbol   common.Symbol
	Price    float64 `json:",string"`
	Quantity float64
	Sum      float64 `json:",string"`
}

type orderBookResult struct {
	Asks []order `json:"ask"`
	Bids []order `json:"bid"`
}

type orderBook struct {
	Success bool
	Message string
	Result  orderBookResult
}

func (o *order) GetSymbol() common.Symbol {
	return o.Symbol
}

func (o *order) GetPrice() float64 {
	return o.Price
}

func (o *order) GetQuantity() float64 {
	return o.Quantity
}

func (o *order) GetSum() float64 {
	return o.Sum
}

func (o *orderBook) GetAsks() []common.Order {
	size := len(o.Result.Asks)
	asks := make([]common.Order, size)

	for i := 0; i < size; i++ {
		asks[i] = &(o.Result.Asks[i])
	}

	return asks
}

func (o *orderBook) GetBids() []common.Order {
	size := len(o.Result.Bids)
	bids := make([]common.Order, size)

	for i := 0; i < size; i++ {
		bids[i] = &(o.Result.Bids[i])
	}

	return bids
}

func sendHTTPRequest(method, url string, body io.Reader, apiKey string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)

	if err != nil {
		return &http.Response{}, err
	}

	req.Header.Set("X-API-Key", apiKey)

	return client.Do(req)
}

func GetOrderBook(symbol common.Symbol, apiKey string) (common.OrderBook, error) {
	url := fmt.Sprintf(ORDER_BOOK_URL, symbol)
	resp, err := sendHTTPRequest("GET", url, nil, apiKey)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	res := orderBook{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	return &res, nil
}
