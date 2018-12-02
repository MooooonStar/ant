package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type Ticker struct {
	Base        string `json:"echange_asset"`
	BaseSymbol  string `json:"echange_asset_symbol"`
	Quote       string `json:"base_asset"`
	QuoteSymbol string `json:"base_asset_symbol"`
	Price       string `json:"price"`
	Min         string `json:"minimum_amount"`
	Max         string `json:"maximum_amount"`
}

func GetExinPrice(ctx context.Context, base, quote string) (map[string]Ticker, error) {
	url := "https://exinone.com/exincore/markets" + fmt.Sprintf("?echange_asset=%s&base_asset=%s", base, quote)
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

	err = json.Unmarshal(body, &response)
	return response.Data, err
}

type Order struct {
	Price  string `json:"price"`
	Amount string `json:"amount"`
}

type Depth struct {
	Asks []Order `json:"asks"`
	Bids []Order `json:"bids"`
}

func GetOceanOrder(ctx context.Context, base, quote string) (*Depth, error) {
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
