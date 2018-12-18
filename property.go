package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/shopspring/decimal"

	bot "github.com/MixinNetwork/bot-api-go-client"
	prettyjson "github.com/hokaccha/go-prettyjson"
	"github.com/jinzhu/gorm"
)

type Wallet struct {
	ID        uint      `gorm:"primary_key"       json:"-"`
	CreatedAt time.Time `gorm:"created_at"        json:"time"`
	BTC       string    `gorm:"type:varchar(20);" json:"BTC"`
	ETH       string    `gorm:"type:varchar(20);" json:"ETH"`
	EOS       string    `gorm:"type:varchar(20);" json:"EOS"`
	XIN       string    `gorm:"type:varchar(20);" json:"XIN"`
	USDT      string    `gorm:"type:varchar(20);" json:"USDT"`
	Total     string    `gorm:"type:varchar(20);" json:"total"`
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

func SaveProperty(ctx context.Context, db *gorm.DB) error {
	assets, err := ReadAssets(context.TODO())
	if err != nil {
		return err
	}

	var total decimal.Decimal
	for asset, balance := range assets {
		amount, _ := decimal.NewFromString(balance)
		if !amount.IsPositive() {
			continue
		}

		depth, err := GetExinDepth(ctx, asset, BTC)
		if err != nil {
			continue
		}
		if len(depth.Bids) == 0 {
			continue
		}
		total = total.Add(amount.Mul(depth.Bids[0].Price))
	}

	amount, _ := decimal.NewFromString(assets[BTC])
	total = total.Add(amount)

	balance := make(map[string]string, 0)
	for asset, amount := range assets {
		balance[Who(asset)] = amount
	}
	balance["total"] = total.Round(5).String()

	bt, err := json.Marshal(balance)
	if err != nil {
		return err
	}
	var wallet Wallet
	err = json.Unmarshal(bt, &wallet)
	if err != nil {
		return err
	}
	return db.Create(&wallet).Error
}
