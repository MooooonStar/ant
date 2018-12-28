package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	bot "github.com/MixinNetwork/bot-api-go-client"
	prettyjson "github.com/hokaccha/go-prettyjson"
	"github.com/shopspring/decimal"
)

var Wallet = map[string]float64{
	BTC:  0.010,
	EOS:  40,
	ETH:  1,
	USDT: 100,
}

func ReadAssets(ctx context.Context) (map[string]string, error) {
	uri := "/assets"
	token, err := bot.SignAuthenticationToken(ClientId, SessionId, PrivateKey, "GET", uri, "")
	if err != nil {
		return nil, err
	}
	body, err := bot.Request(ctx, "GET", uri, nil, token)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data []struct {
			AssetId string `json:"asset_id"`
			Balance string `json:"balance"`
		} `json:"data"`
		Error string `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Error != "" {
		return nil, errors.New(resp.Error)
	}

	assets := make(map[string]string, 0)
	for _, item := range resp.Data {
		assets[item.AssetId] = item.Balance
	}
	return assets, nil
}

func GetExinPrices(ctx context.Context, quote string) (map[string]string, error) {
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

	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}
	prices := make(map[string]string, 0)
	for _, v := range response.Data {
		prices[v.Base] = v.Price
	}
	return prices, nil
}

func ReadSnapshot(ctx context.Context, id string) (string, error) {
	uri := "/network/snapshots/" + id
	token, err := bot.SignAuthenticationToken(ClientId, SessionId, PrivateKey, "GET", uri, "")
	if err != nil {
		return "", err
	}
	body, err := bot.Request(ctx, "GET", uri, nil, token)
	if err != nil {
		return "", err
	}
	var resp struct {
		Data struct {
			SnapshotId string `json:"snapshot_id"`
			TraceId    string `json:"trace_id"`
			OpponentId string `json:"opponent_id"`
			Data       string `json:"data"`
			Amount     string `json:"amount"`
			Asset      struct {
				Symbol string `json:"symbol"`
			} `json:"asset"`
		} `json:"data"`
		Error string `json:"error"`
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return "", err
	}
	if resp.Error != "" {
		return "", errors.New(resp.Error)
	}

	v, _ := prettyjson.Marshal(resp.Data)
	fmt.Println("snapshot info:", string(v))

	return resp.Data.TraceId, nil
}

func SumAssetsNow(ctx context.Context) (float64, error) {
	prices, err := GetExinPrices(ctx, USDT)
	if err != nil {
		return 0, err
	}
	assets, err := ReadAssets(ctx)
	if err != nil {
		return 0, err
	}

	sum, _ := decimal.NewFromString(assets[USDT])
	for asset, balance := range assets {
		price, _ := decimal.NewFromString(prices[asset])
		amount, _ := decimal.NewFromString(balance)
		sum = sum.Add(price.Mul(amount))
	}
	s, _ := sum.Float64()
	return s, nil
}

func SumAssetsInit(ctx context.Context) (float64, error) {
	prices, err := GetExinPrices(ctx, USDT)
	if err != nil {
		return 0, err
	}

	sum := Wallet[USDT]
	for asset, amount := range Wallet {
		price, _ := strconv.ParseFloat(prices[asset], 64)
		sum += price * amount
	}

	return sum, nil
}
