package ant

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	bot "github.com/MixinNetwork/bot-api-go-client"
	"github.com/shopspring/decimal"
)

const (
	SubcribedUser = "ant_subcriberd_user"
	OceanWebsite  = "https://mixcoin.one"
	ExinWebsite   = "https://exinone.com/#/exchange/flash/flashTakeOrder?uuid=%d"
)

var PairIndex = map[string]int{
	"BTC/USDT": 15,
	"ETH/USDT": 17,
	"BCH/USDT": 16,
	"EOS/USDT": 18,
	"ETH/BTC":  19,
	"BCH/BTC":  21,
	"EOS/BTC":  20,
	"XIN/BTC":  4,
	"EOS/ETH":  22,
	"EOS/XIN":  23,
	"ETH/XIN":  24,
}

func (ant *Ant) OnMessage(ctx context.Context, msgView bot.MessageView, userId string) error {
	if msgView.Category == bot.MessageCategoryPlainText {
		data, err := base64.StdEncoding.DecodeString(msgView.Data)
		if err != nil {
			return err
		}
		log.Println("I got a message, it said: ", string(data))
		switch strings.ToLower(string(data)) {
		case "whoisyourdaddy":
			assets, _, err := ReadAssets(ctx)
			if err != nil {
				return err
			}
			out := make(map[string]string, 0)
			for symbol, balance := range assets {
				if amount, _ := strconv.ParseFloat(balance, 64); amount > 0.0 {
					out[symbol] = balance
				}
			}
			bt, err := json.Marshal(out)
			if err != nil {
				return err
			}
			return ant.client.SendPlainText(ctx, msgView, string(bt))
		case "sub":
			if _, err := Redis(ctx).SAdd(SubcribedUser, msgView.UserId).Result(); err != nil {
				log.Println("Add user err", err)
			}
			ant.client.SendPlainText(ctx, msgView, "Thanks for your attention.\n You may get a notification if you can benefit from the price differences below.")
			ocean := bot.Button{Label: "Mixcoin", Action: OceanWebsite, Color: "#2e8b57"}
			exin := bot.Button{Label: "ExinOne", Action: fmt.Sprintf(ExinWebsite, 15), Color: "#bc8f8f"}
			return ant.client.SendAppButtons(ctx, msgView.ConversationId, msgView.UserId, ocean, exin)
		case "unsub":
			if _, err := Redis(ctx).SRem(SubcribedUser, msgView.UserId).Result(); err != nil {
				return err
			}
			return ant.client.SendPlainText(ctx, msgView, "Goodbye! But I am sure you will come back soon.")
		case "help", "帮助":
			return ant.client.SendPlainText(ctx, msgView, "Too young too simple. No help message.")
		case "profit":
			pre, err := ReadAssetsInit(ctx)
			if err != nil {
				return err
			}
			now, prices, err := ReadAssets(ctx)
			if err != nil {
				return err
			}

			sum := decimal.Zero
			s := fmt.Sprintf("%5s:%8v%8v%8v\n", "Symbol", "Before", "Now", "Delta")
			for symbol, amount := range now {
				if symbol == "CNB" {
					continue
				}
				a, _ := decimal.NewFromString(amount)
				b, _ := decimal.NewFromString(pre[symbol])
				c, _ := decimal.NewFromString(prices[symbol])
				sum = sum.Add(a.Sub(b).Mul(c))
				s += fmt.Sprintf("%5s:%8v%8v%8v\n", symbol, b.Round(4), a.Round(4), a.Sub(b).Round(4))
			}
			return ant.client.SendPlainText(ctx, msgView, fmt.Sprintf("%s  Total:  %8v USD", s, sum.Round(4)))
		default:
			reply, err := Reply(string(data))
			if err != nil {
				return ant.client.SendPlainText(ctx, msgView, "I am busy!!! Stop disturbing me.")
			}
			return ant.client.SendPlainText(ctx, msgView, reply)
		}
	}
	return nil
}

func (ant *Ant) Notice(ctx context.Context, event ProfitEvent) error {
	users, err := Redis(ctx).SMembers(SubcribedUser).Result()
	if err != nil {
		return err
	}
	actions := map[string]string{
		PageSideBid: " Buy in Mixcoin",
		PageSideAsk: "Sell in Mixcoin",
	}

	template := "Go Go Go!\nAction:  %-10s\nPair:         %-10s\nPrice:       %-10.8s\nAmount:    %-10s\nProfit:   %8s%%"
	pair := Who(event.Base) + "/" + Who(event.Quote)
	ocean := bot.Button{Label: "Mixcoin", Action: OceanWebsite, Color: "#2e8b57"}
	exin := bot.Button{Label: "ExinOne", Action: fmt.Sprintf(ExinWebsite, PairIndex[pair]), Color: "#bc8f8f"}
	msg := fmt.Sprintf(template, actions[event.Category], pair, event.Price.String(),
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
	return nil
}

func (ant *Ant) PollMixinMessage(ctx context.Context) {
	for {
		ant.client = bot.NewBlazeClient(ClientId, SessionId, PrivateKey)
		if err := ant.client.Loop(ctx, ant); err != nil {
			log.Println(err)
		}
		time.Sleep(1 * time.Second)
	}
}
