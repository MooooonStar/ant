package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	bot "github.com/MixinNetwork/bot-api-go-client"
	log "github.com/sirupsen/logrus"
)

func (ant *Ant) OnMessage(ctx context.Context, msgView bot.MessageView, userId string) error {
	if msgView.Category == bot.MessageCategoryPlainText && msgView.ConversationId == bot.UniqueConversationId(ClientId, msgView.UserId) {
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
			if amount, _ := strconv.ParseFloat(balance, 64); amount > 0.0 {
				out[Who(asset)] = balance
			}
		}
		bt, err := json.Marshal(out)
		if err != nil {
			return err
		}
		ant.client.SendPlainText(ctx, msgView, string(bt))
	}
	return nil
}

func (ant *Ant) Notice(ctx context.Context, content string, id ...int) {
	users := make([]string, 0)
	for _, number := range id {
		if mixinId, err := SearchUser(ctx, fmt.Sprint(number)); err == nil {
			users = append(users, mixinId)
		}
	}

	for _, user := range users {
		view := bot.MessageView{
			ConversationId: bot.UniqueConversationId(ClientId, user),
			UserId:         user,
		}

		if err := ant.client.SendPlainText(ctx, view, content); err != nil {
			log.Println(err)
		}
	}
}

func (ant *Ant) PollMixinMessage(ctx context.Context) {
	for {
		if err := ant.client.Loop(ctx, ant); err != nil {
			log.Println(err)
			time.Sleep(1 * time.Second)
		}
	}
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
