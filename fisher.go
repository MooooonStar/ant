package main

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
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

func (ant *Ant) Fishing(ctx context.Context, base, quote string) {
	orders := make(map[string]bool, 0)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			precent := decimal.NewFromFloat(LowerPercent)
			if otc, err := GetExinDepth(ctx, base, quote); err == nil {
				trade := ant.GetOceanTrade(ctx, base, quote)
				if _, ok := orders[trade.CreateAt]; ok {
					continue
				}

				ts, err := time.Parse(time.RFC3339Nano, trade.CreateAt)
				if err != nil || ts.Add(5*time.Minute).Before(time.Now()) {
					continue
				}
				price, _ := decimal.NewFromString(trade.Price)
				precision := price.Exponent()
				amount, _ := decimal.NewFromString(trade.Amount)
				amount = amount.Mul(decimal.NewFromFloat(2.0))
				if len(otc.Asks) > 0 {
					if price.GreaterThan(otc.Asks[0].Price) {
						bidFishing := price.Sub(price.Sub(otc.Asks[0].Price).Mul(precent))
						exchange := Order{
							Price:  bidFishing.Truncate(-precision + 1),
							Amount: amount,
						}
						ant.Inspect(ctx, exchange, otc.Asks[0], base, quote, PageSideBid, 10*OrderExpireTime)
					}
				}

				if len(otc.Bids) > 0 {
					if price.LessThan(otc.Bids[0].Price) {
						askFishing := price.Sub(price.Sub(otc.Bids[0].Price).Mul(precent))
						exchange := Order{
							Price:  askFishing.Truncate(-precision + 1),
							Amount: amount,
						}
						ant.Inspect(ctx, exchange, otc.Bids[0], base, quote, PageSideAsk, 10*OrderExpireTime)
					}
				}
				orders[trade.CreateAt] = true
			}
		}
	}
}

func (ant *Ant) GetOceanTrade(ctx context.Context, base, quote string) Trade {
	pair := base + "-" + quote
	return ant.books[pair].trade
}
