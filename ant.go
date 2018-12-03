package main

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/hokaccha/go-prettyjson"

	uuid "github.com/satori/go.uuid"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
)

const (
	WatchingMode       = true
	ProfitThreshold    = 0.001 / (1 - OceanFee) / (1 - ExinFee)
	OceanFee           = 0.002
	ExinFee            = 0.001
	StrategyLow        = "L"
	StrategyHigh       = "H"
	PendingOrdersLimit = 10
	OrderLife          = 60 * time.Second
	OrderConfirmedTime = 10 * time.Second
)

type Event struct {
	ID       string          `json:"-"`
	Category string          `json:"category"`
	Price    decimal.Decimal `json:"price"`
	Profit   decimal.Decimal `json:"profit"`
	Amount   decimal.Decimal `json:"amount"`
	Base     string          `json:"base"`
	Quote    string          `json:"quote"`
}

type Ant struct {
	event        chan Event
	snapshots    map[string]bool
	exOrders     map[string]bool
	otcOrders    map[string]bool
	lock         sync.Mutex
	orderMatched chan bool
}

func NewAnt() *Ant {
	return &Ant{
		event:        make(chan Event, 0),
		snapshots:    make(map[string]bool, 0),
		exOrders:     make(map[string]bool, 0),
		otcOrders:    make(map[string]bool, 0),
		orderMatched: make(chan bool, 0),
	}
}

func UuidWithString(str string) string {
	h := md5.New()
	io.WriteString(h, str)
	sum := h.Sum(nil)
	sum[6] = (sum[6] & 0x0f) | 0x30
	sum[8] = (sum[8] & 0x3f) | 0x80
	return uuid.FromBytesOrNil(sum).String()
}

func (ant *Ant) Trade(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			for trace, _ := range ant.exOrders {
				for i := 0; i < 3; i++ {
					fmt.Println("cancel order:", trace)
					OceanCancel(trace)
					time.Sleep(100 * time.Millisecond)
				}
			}
			return
		case e := <-ant.event:
			v, _ := prettyjson.Marshal(e)
			fmt.Println("profit found, ", string(v))

			amount, _ := e.Amount.Float64()
			price, _ := e.Price.Float64()
			switch e.Category {
			case StrategyLow:
				exchangeOrder := UuidWithString(e.ID + OceanCore)
				if _, err := OceanSell(price, amount, StrategyLow, e.Base, e.Quote, exchangeOrder); err == nil {
					ant.exOrders[exchangeOrder] = true
					select {
					case <-ant.orderMatched:
						delete(ant.exOrders, exchangeOrder)
						otcOrder := UuidWithString(e.ID + ExinCore)
						if _, err := ExinTrade(amount*price, e.Quote, e.Base, otcOrder); err == nil {
							ant.otcOrders[otcOrder] = true
						}
					case <-time.After(OrderConfirmedTime):
						for i := 0; i < 3; i++ {
							OceanCancel(exchangeOrder)
						}
					}
				}
			case StrategyHigh:
				exchangeOrder := UuidWithString(e.ID + OceanCore)
				if _, err := OceanBuy(price, amount*price, StrategyHigh, e.Base, e.Quote, exchangeOrder); err == nil {
					ant.exOrders[exchangeOrder] = true
					select {
					case <-ant.orderMatched:
						delete(ant.exOrders, exchangeOrder)
						otcOrder := UuidWithString(e.ID + ExinCore)
						if _, err := ExinTrade(amount, e.Base, e.Quote, otcOrder); err == nil {
							ant.otcOrders[otcOrder] = true
						}
					case <-time.After(OrderConfirmedTime):
						for i := 0; i < 3; i++ {
							OceanCancel(exchangeOrder)
						}
					}
				}
			}
		}
	}
}

func (ant *Ant) Watching(ctx context.Context, base, quote string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
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
}

func (ant *Ant) Low(ctx context.Context, exchange, otc Order, base, quote string) {
	bidPrice, price := exchange.Price, otc.Price
	bidProfit := bidPrice.Sub(price).Div(price)
	log.Debugf("bid -- ocean price: %10.8v, exin price: %10.8v, profit: %10.8v, %5v/%5v", exchange.Price, otc.Price, bidProfit, Who(base), Who(quote))
	if bidProfit.GreaterThan(decimal.NewFromFloat(ProfitThreshold)) {
		if exchange.Amount.LessThanOrEqual(otc.Min) {
			log.Errorf("amount is too small, %v <= %v", exchange.Amount, otc.Min)
			return
		}
		amount := exchange.Amount
		if amount.GreaterThanOrEqual(otc.Max) {
			amount = otc.Max
		}
		id := UuidWithString(Who(base) + Who(quote) + bidPrice.String() + amount.String() + StrategyLow)
		ant.event <- Event{
			ID:       id,
			Category: StrategyLow,
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
	fmt.Println("hi, ", exchange, otc)
	askPrice, price := exchange.Price, otc.Price
	askProfit := price.Sub(askPrice).Div(price)
	log.Debugf("ask -- ocean price: %10.8v, exin price: %10.8v, profit: %10.8v, %5v/%5v", exchange.Price, otc.Price, askProfit, Who(base), Who(quote))
	if askProfit.GreaterThan(decimal.NewFromFloat(ProfitThreshold)) {
		if exchange.Amount.LessThanOrEqual(otc.Min) {
			log.Errorf("amount is too small, %v <= %v", exchange.Amount, otc.Min)
			return
		}
		amount := exchange.Amount
		if amount.GreaterThanOrEqual(otc.Max) {
			amount = otc.Max
		}
		id := UuidWithString(Who(base) + Who(quote) + askPrice.String() + amount.String() + StrategyHigh)
		ant.event <- Event{
			ID:       id,
			Category: StrategyHigh,
			Price:    askPrice,
			Amount:   amount,
			Profit:   askProfit,
			Base:     base,
			Quote:    quote,
		}
	}
	return
}
