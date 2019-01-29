package ant

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
)

type Order struct {
	Price  decimal.Decimal `json:"price"`
	Amount decimal.Decimal `json:"amount"`
	Min    decimal.Decimal `json:"min"`
	Max    decimal.Decimal `json:"max"`
}

type Depth struct {
	Asks []Order `json:"asks"`
	Bids []Order `json:"bids"`
}

type Ticker struct {
	Base  string `json:"exchange_asset"`
	Quote string `json:"base_asset"`
	Price string `json:"price"`
	Min   string `json:"minimum_amount"`
	Max   string `json:"maximum_amount"`
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

func FetchExinDepth(ctx context.Context, base, quote string) (*Depth, error) {
	url := "https://api.exinone.com/instant/" + fmt.Sprint(PairIndex[Who(base)+"/"+Who(quote)])

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bt, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var Resp struct {
		Data struct {
			Depth struct {
				Asks []Order `json:"sell"`
				Bids []Order `json:"buy"`
			} `json:"depth"`
			BuyMax  decimal.Decimal `json:"buyMax"`
			SellMax decimal.Decimal `json:"sellMax"`
			BuyMin  decimal.Decimal `json:"buyMin"`
			SellMin decimal.Decimal `json:"SellMin"`
			Price   decimal.Decimal `json:"price"`
		} `json:"data"`
	}

	err = json.Unmarshal(bt, &Resp)
	if err != nil {
		return nil, err
	}

	for idx, _ := range Resp.Data.Depth.Asks {
		Resp.Data.Depth.Asks[idx].Min = Resp.Data.SellMin
		Resp.Data.Depth.Asks[idx].Max = Resp.Data.SellMax
	}

	for idx, order := range Resp.Data.Depth.Bids {
		Resp.Data.Depth.Bids[idx].Min = Resp.Data.BuyMin.Div(order.Price)
		Resp.Data.Depth.Bids[idx].Max = Resp.Data.BuyMax.Div(order.Price)
	}

	d := Depth{
		// Asks: Resp.Data.Depth.Asks,
		// Bids: Resp.Data.Depth.Bids,
		Bids: Resp.Data.Depth.Asks,
		Asks: Resp.Data.Depth.Bids,
	}
	return &d, nil
}

func GetExinDepth(ctx context.Context, base, quote string) (*Depth, error) {
	var depth Depth
	if order, err := GetExinOrder(ctx, base, quote); err != nil {
		return nil, err
	} else {
		order.Max = order.Max.Div(order.Price)
		order.Min = order.Min.Div(order.Price)
		depth.Asks = []Order{*order}
	}

	if order, err := GetExinOrder(ctx, quote, base); err != nil {
		return nil, err
	} else {
		order.Price = decimal.NewFromFloat(1.0).Div(order.Price)
		depth.Bids = []Order{*order}
	}
	return &depth, nil
}

func GetExinOrder(ctx context.Context, base, quote string) (*Order, error) {
	url := "https://exinone.com/exincore/markets" + fmt.Sprintf("?&base_asset=%s&exchange_asset=%s", quote, base)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)
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

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)
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
