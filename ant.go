package main

import (
	"context"
	"time"

	"github.com/hokaccha/go-prettyjson"

	"github.com/shopspring/decimal"
	"github.com/smallnest/rpcx/log"
)

const (
	WatchingMode    = true
	ProfitThreshold = 0.1 / (1 - OceanFee) / (1 - ExinFee)
	OceanFee        = 0.002
	ExinFee         = 0.001
)

type Event struct {
	Category string          `json:"category"`
	Price    decimal.Decimal `json:"ocean_price"`
	Profit   decimal.Decimal `json:"profit"`
	Amount   decimal.Decimal `json:"amount"`
	Base     string          `json:"base"`
	Quote    string          `json:"quote"`
}

type Ant struct {
	e chan Event
}

func (ant *Ant) Run() {
	for {
		select {
		case e := <-ant.e:
			v, _ := prettyjson.Marshal(e)
			log.Info(string(v))
			if WatchingMode {
				continue
			}
			amount, _ := e.Amount.Float64()
			price, _ := e.Price.Float64()
			switch e.Category {
			case "L":
				if _, err := ExinTrade(amount*price, e.Quote, e.Base); err == nil {
					OceanSell(price, amount, "L", e.Base, e.Quote)
				}
			case "H":
				if _, err := OceanBuy(price, amount*price, "L", e.Base, e.Quote); err == nil {
					ExinTrade(amount, e.Base, e.Quote)
				}
			}
		}
	}
}

func (ant *Ant) Watching(ctx context.Context, base, quote string) {
	for {
		if exchange, err := GetOceanDepth(ctx, base, quote); err == nil {
			if otc, err := GetExinDepth(ctx, base, quote); err == nil {
				if len(exchange.Bids) > 0 && len(otc.Bids) > 0 {
					ant.Low(ctx, exchange.Bids[0], otc.Bids[0], base, quote)
				}

				if len(exchange.Asks) > 0 && len(otc.Asks) > 0 {
					ant.High(ctx, exchange.Asks[0], otc.Asks[0], base, quote)
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (ant *Ant) Low(ctx context.Context, exchange, otc Order, base, quote string) {
	bidPrice, price := exchange.Price, otc.Price
	bidProfit := bidPrice.Sub(price).Div(price)
	log.Debugf("bid -- ocean price: %10.8v, exin price: %10.8v, profit: %10.8v, %5v/%5v", exchange.Price, otc.Price, bidProfit, Who(base), Who(quote))
	if bidProfit.GreaterThan(decimal.NewFromFloat(ProfitThreshold)) {
		if exchange.Amount.LessThanOrEqual(otc.Amount) {
			log.Errorf("amount is too small, %v <= %v", exchange.Amount, otc.Amount)
			return
		}
		amount := exchange.Amount
		if amount.GreaterThanOrEqual(otc.Amount) {
			amount = otc.Amount
		}
		ant.e <- Event{
			Category: "L",
			Price:    bidPrice,
			Amount:   amount,
			Profit:   bidProfit,
			Base:     base,
			Quote:    quote,
		}
	}
	return
}

func (ant *Ant) High(ctx context.Context, exchange, otc Order, base, quote string) {
	askPrice, price := exchange.Price, otc.Price
	askProfit := price.Sub(askPrice).Div(price)
	log.Debugf("ask -- ocean price: %10.8v, exin price: %10.8v, profit: %10.8v, %5v/%5v", exchange.Price, otc.Price, askProfit, Who(base), Who(quote))
	if askProfit.GreaterThan(decimal.NewFromFloat(ProfitThreshold)) {
		if exchange.Amount.LessThanOrEqual(otc.Amount) {
			log.Errorf("amount is too small, %v <= %v", exchange.Amount, otc.Amount)
			return
		}
		amount := exchange.Amount
		if amount.GreaterThanOrEqual(otc.Amount) {
			amount = otc.Amount
		}
		ant.e <- Event{
			Category: "H",
			Price:    askPrice,
			Amount:   amount,
			Profit:   askProfit,
			Base:     base,
			Quote:    quote,
		}
	}
	return
}
