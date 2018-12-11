package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/hokaccha/go-prettyjson"

	"github.com/shopspring/decimal"
)

type Order struct {
	Price  decimal.Decimal
	Amount decimal.Decimal
	Min    decimal.Decimal
	Max    decimal.Decimal
}

type Depth struct {
	Asks []Order `json:"asks"`
	Bids []Order `json:"bids"`
}

type Ticker struct {
	Base  string `json:"echange_asset"`
	Quote string `json:"base_asset"`
	Price string `json:"price"`
	Min   string `json:"minimum_amount"`
	Max   string `json:"maximum_amount"`
}

func GetExinDepth(ctx context.Context, base, quote string) (*Depth, error) {
	var depth Depth
	if order, err := GetExinOrder(ctx, base, quote); err != nil {
		return nil, err
	} else {
		price := order.Price
		order.Max = order.Max.Div(price)
		order.Min = order.Min.Div(price)
		depth.Asks = []Order{*order}
	}

	if order, err := GetExinOrder(ctx, quote, base); err != nil {
		return nil, err
	} else {
		price := decimal.NewFromFloat(1.0).Div(order.Price)
		order.Max = order.Max.Mul(price)
		order.Min = order.Min.Mul(price)
		order.Price = price
		depth.Bids = []Order{*order}
	}
	return &depth, nil
}

func GetExinOrder(ctx context.Context, base, quote string) (*Order, error) {
	url := "https://exinone.com/exincore/markets" + fmt.Sprintf("?&base_asset=%s", quote)
	client := http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response struct {
		Data map[string]Ticker `json:"data"`
	}

	v, _ := prettyjson.Marshal(response.Data)
	fmt.Println("data", string(v))

	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	for _, v := range response.Data {
		if v.Base == base {
			price, _ := decimal.NewFromString(v.Price)
			min, _ := decimal.NewFromString(v.Min)
			max, _ := decimal.NewFromString(v.Max)
			return &Order{Price: price, Max: max, Min: min}, nil
		}
	}
	return nil, fmt.Errorf("not found.")
}

func GetOceanDepth(ctx context.Context, base, quote string) (*Depth, error) {
	url := "https://events.ocean.one/markets/" + fmt.Sprintf("%s-%s", base, quote) + "/book"
	client := http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response struct {
		Data struct {
			Depth `json:"data"`
		} `json:"data"`
	}

	err = json.Unmarshal(body, &response)
	return &response.Data.Depth, err
}
