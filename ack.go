package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	bot "github.com/MixinNetwork/bot-api-go-client"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
)

const (
	OceanWebsite = "https://mixcoin.one"
	ExinWebsite  = "https://exinone.com/#/exchange/flash/flashTakeOrder?uuid=17"
)

func (ant *Ant) OnMessage(ctx context.Context, msgView bot.MessageView, userId string) error {
	if msgView.Category == bot.MessageCategoryPlainText && msgView.ConversationId == bot.UniqueConversationId(ClientId, msgView.UserId) {
		data, err := base64.StdEncoding.DecodeString(msgView.Data)
		if err != nil {
			return err
		}
		switch string(data) {
		case "whoisyourdaddy":
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
		case "sub":
			pay := Payment{Recipient: ClientId, Asset: CNB, Amount: "666", Memo: "I am in"}
			if err := ant.client.SendAppButton(ctx, msgView.ConversationId, msgView.UserId, string(data), pay.Url(), "#ba55d3"); err != nil {
				log.Println("Sub error", err)
			}
		case "trade":
			ocean := bot.Button{Label: "ocean", Action: OceanWebsite, Color: "#2e8b57"}
			exin := bot.Button{Label: "exin", Action: ExinWebsite, Color: "#bc8f8f"}
			if err := ant.client.SendAppButtons(ctx, msgView.ConversationId, msgView.UserId, ocean, exin); err != nil {
				log.Println("Trade error", err)
			}
		}

	}
	return nil
}

func (ant *Ant) Notice(ctx context.Context, event ProfitEvent, id ...int) {
	users := make([]string, 0)
	for _, number := range id {
		if mixinId, err := SearchUser(ctx, fmt.Sprint(number)); err == nil {
			users = append(users, mixinId)
		}
	}

	template := `Action:           %8s,Pair:          %8s,Price:       %10.8s,Amount:      %8s,Profit:           %8s%%`
	ocean := bot.Button{Label: "ocean", Action: OceanWebsite, Color: "#2e8b57"}
	exin := bot.Button{Label: "exin", Action: ExinWebsite, Color: "#bc8f8f"}
	msg := fmt.Sprintf(template, event.Category, Who(event.Base)+"/"+Who(event.Quote), event.Price.String(),
		event.Amount.String(), event.Profit.Mul(decimal.NewFromFloat(100.0)).Round(2).String())

	for _, user := range users {
		msgView := bot.MessageView{
			ConversationId: bot.UniqueConversationId(ClientId, user),
			UserId:         user,
		}

		if err := ant.client.SendPlainText(ctx, msgView, msg); err != nil {
			log.Println("Send message error", err)
		}

		if err := ant.client.SendAppButtons(ctx, msgView.ConversationId, msgView.UserId, ocean, exin); err != nil {
			log.Println("Trade error", err)
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
