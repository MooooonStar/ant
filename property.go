package ant

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	bot "github.com/MixinNetwork/bot-api-go-client"
	"github.com/shopspring/decimal"
)

var (
	in, _     = time.LoadLocation("Asia/Chongqing")
	StartTime = time.Date(2019, 1, 24, 22, 10, 0, 0, in)
)

func ReadAssetsInit(ctx context.Context) (map[string]string, error) {
	var wallets []struct {
		Asset  string
		Amount string
	}

	var s Snapshot
	if err := Database(ctx).Model(&Snapshot{}).Where("opponent_id = ? AND amount <  0", MasterID).Order("created_at DESC").First(&s).Error; err != nil {
		return nil, err
	}

	db := Database(ctx).Model(&Snapshot{}).Where("opponent_id = ? AND created_at > ?", MasterID, s.CreatedAt).
		Select("asset_id AS asset,sum(amount) AS amount").Group("asset").Scan(&wallets)
	if db.Error != nil {
		return nil, db.Error
	}

	start := make(map[string]string, 0)
	for _, wallet := range wallets {
		symbol := Who(wallet.Asset)
		start[symbol] = wallet.Amount
	}
	return start, nil
}

func ReadAssets(ctx context.Context) (map[string]string, map[string]string, error) {
	uri := "/assets"
	token, err := bot.SignAuthenticationToken(ClientId, SessionId, PrivateKey, "GET", uri, "")
	if err != nil {
		return nil, nil, err
	}
	body, err := bot.Request(ctx, "GET", uri, nil, token)
	if err != nil {
		return nil, nil, err
	}
	var resp struct {
		Data []struct {
			Symbol   string `json:"symbol"`
			Balance  string `json:"balance"`
			PriceUsd string `json:"price_usd"`
		} `json:"data"`
		Error string `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, nil, err
	}
	if resp.Error != "" {
		return nil, nil, errors.New(resp.Error)
	}

	prices := make(map[string]string, 0)
	assets := make(map[string]string, 0)
	for _, item := range resp.Data {
		balance, _ := decimal.NewFromString(item.Balance)
		if balance.IsZero() {
			continue
		}
		assets[item.Symbol] = item.Balance
		prices[item.Symbol] = item.PriceUsd
	}
	return assets, prices, nil
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

	return resp.Data.TraceId, nil
}

// func ReadAssets(ctx context.Context) (map[string]string, error) {
// 	uri := "/assets"
// 	token, err := bot.SignAuthenticationToken(ClientId, SessionId, PrivateKey, "GET", uri, "")
// 	if err != nil {
// 		return nil, err
// 	}
// 	body, err := bot.Request(ctx, "GET", uri, nil, token)
// 	if err != nil {
// 		return nil, err
// 	}
// 	var resp struct {
// 		Data []struct {
// 			AssetId string `json:"asset_id"`
// 			Balance string `json:"balance"`
// 		} `json:"data"`
// 		Error string `json:"error"`
// 	}
// 	err = json.Unmarshal(body, &resp)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if resp.Error != "" {
// 		return nil, errors.New(resp.Error)
// 	}

// 	assets := make(map[string]string, 0)
// 	for _, item := range resp.Data {
// 		assets[item.AssetId] = item.Balance
// 	}
// 	return assets, nil
// }
