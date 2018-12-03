package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	bot "github.com/MixinNetwork/bot-api-go-client"
	prettyjson "github.com/hokaccha/go-prettyjson"
)

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
