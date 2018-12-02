package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	WatchingMode    = true
	ProfitThreshold = 0.1 / (1 - OceanFee) / (1 - ExinFee)
	OceanFee        = 0.002
	ExinFee         = 0.001
)

type Event struct {
	Category   string  `json:"category"`
	OceanPrice float64 `json:"ocean_price"`
	ExinPrice  float64 `json:"exin_price"`
	Profit     float64 `json:"profit"`
	Amount     float64 `json:"amount"`
	Base       string  `json:"base"`
	Quote      string  `json:"quote"`
}

type Ant struct {
	e chan Event
}

func (ant *Ant) Run() {
	for {
		select {
		case e := <-ant.e:
			if WatchingMode {
				continue
			}
			switch e.Category {
			case "H":
				ant.StrategyHigh(e.OceanPrice, e.Amount, e.Base, e.Quote)
			case "L":
				ant.StrategyLow(e.OceanPrice, e.Amount, e.Base, e.Quote)
			}
		}
	}
}

func (ant *Ant) StrategyLow(price, amount float64, base, quote string) (string, error) {
	trace, err := OceanSell(price, amount, "L", base, quote)
	if err == nil {
		_, err = ExinTrade(amount*price, quote, base)
	}
	return trace, err
}

func (ant *Ant) StrategyHigh(price, amount float64, base, quote string) (string, error) {
	trace, err := OceanBuy(price, amount/price, "L", base, quote)
	if err == nil {
		_, err = ExinTrade(amount, base, quote)
	}
	return trace, err
}

func (ant *Ant) Watching(ctx context.Context, base, quote string) {
	for {
		depth, err := GetOceanOrder(ctx, base, quote)
		if err == nil && depth != nil {
			ant.Low(ctx, *depth, base, quote)
			ant.High(ctx, *depth, base, quote)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (ant *Ant) Low(ctx context.Context, depth Depth, base, quote string) error {
	if depth.Bids == nil {
		return fmt.Errorf("no bids in ocean.one")
	}

	tickers, err := GetExinPrice(ctx, base, quote)
	if err != nil {
		return err
	}

	price, minAmount, maxAmount := 0.0, 0.0, 0.0
	for _, v := range tickers {
		if v.Base == base && v.Quote == quote {
			if price, _ = strconv.ParseFloat(v.Price, 64); err == nil {
				minAmount, _ = strconv.ParseFloat(v.Min, 64)
				maxAmount, _ = strconv.ParseFloat(v.Max, 64)
				break
			}
		}
	}
	if price <= 0.0 {
		return fmt.Errorf("not found in exin")
	}

	if depth.Bids == nil {
		return fmt.Errorf("no order in ocean.one")
	}

	bidProfit := 0.0
	bidPrice, _ := strconv.ParseFloat(depth.Bids[0].Price, 64)
	bidProfit = (bidPrice - price) / price
	log.Debugf("ocean bid price: %10.8v, exin price: %10.8v, profit: %5.2v, %v/%v", bidPrice, price, bidProfit, Who(base), Who(quote))

	if bidProfit > ProfitThreshold {
		bidAmount, _ := strconv.ParseFloat(depth.Bids[0].Amount, 64)
		if bidAmount < minAmount {
			return fmt.Errorf("amount is too small, %10.8v < %10.8v", bidAmount, minAmount)
		}
		if bidAmount > maxAmount {
			bidAmount = maxAmount
		}
		log.Infof("ocean bid price: %10.8v, exin price: %10.8v, profit: %5.2v, %v/%v", bidPrice, price, bidProfit, Who(base), Who(quote))
		ant.e <- Event{
			Category:   "L",
			OceanPrice: bidPrice,
			ExinPrice:  price,
			Profit:     bidProfit,
			Amount:     bidAmount,
			Base:       base,
			Quote:      quote,
		}
	}
	return nil
}

func (ant *Ant) High(ctx context.Context, depth Depth, base, quote string) error {
	if depth.Asks == nil {
		return fmt.Errorf("no asks in ocean.one")
	}

	tickers, err := GetExinPrice(ctx, quote, base)
	if err != nil {
		return err
	}

	price, minAmount, maxAmount := 0.0, 0.0, 0.0
	for _, v := range tickers {
		if v.Base == quote && v.Quote == base {
			if price, _ = strconv.ParseFloat(v.Price, 64); err == nil {
				minAmount, _ = strconv.ParseFloat(v.Min, 64)
				maxAmount, _ = strconv.ParseFloat(v.Max, 64)
				break
			}
		}
	}
	if price <= 0.0 {
		return fmt.Errorf("no order in exin")
	}
	price = 1 / price

	askProfit := 0.0
	if depth.Asks != nil {
		askPrice, _ := strconv.ParseFloat(depth.Asks[0].Price, 64)
		askProfit = (price - askPrice) / price
		log.Debugf("ocean ask price: %10.8v, exin price: %10.8v, profit: %5.2v, %v/%v", askPrice, price, askProfit, Who(base), Who(quote))

		if askProfit > ProfitThreshold {
			askAmount, _ := strconv.ParseFloat(depth.Asks[0].Amount, 64)
			if askAmount < minAmount {
				return fmt.Errorf("amount is too small, %10.8v < %10.8v", askAmount, minAmount)
			}
			if askAmount > maxAmount {
				askAmount = maxAmount
			}
			log.Infof("ocean ask price: %10.8v, exin price: %10.8v, profit: %5.2v, %v/%v", askPrice, price, askProfit, Who(base), Who(quote))
			ant.e <- Event{
				Category:   "H",
				OceanPrice: askPrice,
				ExinPrice:  price,
				Profit:     askProfit,
				Amount:     askAmount,
				Base:       base,
				Quote:      quote,
			}
		}
	}
	return nil
}
