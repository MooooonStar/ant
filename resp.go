package main

import (
	"context"
	"encoding/base64"
	"encoding/json"

	bot "github.com/MixinNetwork/bot-api-go-client"
	"github.com/shopspring/decimal"
)

type Handler struct {
	client *bot.BlazeClient
}

func (r *Handler) OnMessage(ctx context.Context, msgView bot.MessageView, botID string) error {
	if msgView.Category == bot.MessageCategoryPlainText { //&& msgView.ConversationId == bot.UniqueConversationId(ClientId, msgView.UserId) {
		if data, err := base64.StdEncoding.DecodeString(msgView.Data); err != nil {
			return err
		} else if string(data) != "show" {
			return nil
		}
		assets, err := ReadAssets(ctx)
		if err != nil {
			return err
		}
		out := make(map[string]string, 0)
		for asset, balance := range assets {
			if amount, _ := decimal.NewFromString(balance); amount.IsPositive() {
				out[Who(asset)] = balance
			}
		}
		bt, err := json.Marshal(out)
		if err != nil {
			return err
		}
		r.client.SendPlainText(ctx, msgView, string(bt))
	}
	return nil
}

func SearchUser(ctx context.Context, id string) (string, error) {
	method, uri := "GET", "/search/"+id
	token, err := bot.SignAuthenticationToken(ClientId, SessionId, PrivateKey, "GET", uri, "")
	if err != nil {
		return "", err
	}
	bt, err := bot.Request(ctx, method, uri, nil, token)
	if err != nil {
		return "", err
	}

	var resp struct {
		Data struct {
			UserId string `json:"user_id"`
		} `json:"data"`
	}

	err = json.Unmarshal(bt, &resp)
	return resp.Data.UserId, err
}
