package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
)

const (
	LowerPercent = 0.10
)

type Trade struct {
	Price    string `json:"price"`
	Amount   string `json:"amount"`
	Side     string `json:"side"`
	CreateAt string `json:"created_at"`
}

//Exin上价格在变动，导致钓鱼单的价格也会变化，造成ocean.one上一笔成交生成多笔钓鱼单，待优化
func (ant *Ant) Fishing(ctx context.Context, base, quote string) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			precent := decimal.NewFromFloat(LowerPercent)
			if otc, err := GetExinDepth(ctx, base, quote); err == nil {
				if trades, err := GetOceanTrades(ctx, base, quote); err == nil && len(trades) > 0 {
					ts, _ := time.Parse(time.RFC3339Nano, trades[0].CreateAt)
					fmt.Println("time", ts.Add(30*time.Minute).Before(time.Now()))
					fmt.Println("time", ts.Add(8*time.Hour+30*time.Minute).Before(time.Now()))
					if ts.Add(8*time.Hour + 5*time.Minute).Before(time.Now()) {
						continue
					}
					fmt.Println("trade time", ts.Add(8*time.Hour+5*time.Minute))
					fmt.Println("time now", time.Now())
					price, _ := decimal.NewFromString(trades[0].Price)
					precision := price.Exponent()
					amount, _ := decimal.NewFromString(trades[0].Amount)
					if len(otc.Asks) > 0 {
						if price.GreaterThan(otc.Asks[0].Price) {
							log.Debugf("!!!!!--find trade profit, amount %s, price %s, %s/%s, start fishing--!!!!!", amount, price, Who(base), Who(quote))
							bidFishing := price.Sub(price.Sub(otc.Asks[0].Price).Mul(precent))
							exchange := Order{
								Price:  bidFishing.Truncate(-precision + 1),
								Amount: amount,
							}
							ant.Strategy(ctx, exchange, otc.Asks[0], base, quote, PageSideBid)
						}
					}

					if len(otc.Bids) > 0 {
						if price.LessThan(otc.Bids[0].Price) {
							log.Debugf("find trade, amount %s, price %s, %s/%s, start fishing....", amount, price, Who(base), Who(quote))
							askFishing := price.Sub(price.Sub(otc.Bids[0].Price).Mul(precent))
							exchange := Order{
								Price:  askFishing.Truncate(-precision + 1),
								Amount: amount,
							}
							ant.Strategy(ctx, exchange, otc.Bids[0], base, quote, PageSideAsk)
						}
					}
				}
			}
		}
	}
}

func GetOceanTrades(ctx context.Context, base, quote string) ([]Trade, error) {
	url := "https://events.ocean.one/markets/" + base + "-" + quote + "/trades"
	offset := time.Now().Add(-5 * time.Minute).UTC().Format(time.RFC3339Nano)
	query := fmt.Sprintf("?limit=%d&offset=%s&order=DESC", 10, offset)
	client := http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", url+query, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	bt, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}

	var data struct {
		Trades []Trade `json:"data"`
	}
	err = json.Unmarshal(bt, &data)
	return data.Trades, err
}
