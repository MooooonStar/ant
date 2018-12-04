package main

import (
	"context"
	"crypto/md5"
	"io"
	"sync"
	"time"

	"github.com/hokaccha/go-prettyjson"
	uuid "github.com/satori/go.uuid"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
)

const (
	ProfitThreshold    = 0.01 / (1 - OceanFee) / (1 - ExinFee) / (1 - HuobiFee)
	OceanFee           = 0.002
	ExinFee            = 0.001
	HuobiFee           = 0.001
	StrategyLow        = "L"
	StrategyHigh       = "H"
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
	orderMatched chan bool
	Enabled      bool
	lock         sync.Mutex
}

func NewAnt(enabled bool) *Ant {
	return &Ant{
		event:        make(chan Event, 0),
		snapshots:    make(map[string]bool, 0),
		exOrders:     make(map[string]bool, 0),
		otcOrders:    make(map[string]bool, 0),
		orderMatched: make(chan bool, 0),
		Enabled:      enabled,
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
			for trace, ok := range ant.exOrders {
				if !ok {
					log.Println("cancel order:", trace)
					for i := 0; i < 3; i++ {
						//连发三次，尽可能取消订单
						OceanCancel(trace)
						time.Sleep(100 * time.Millisecond)
					}
				}
			}
			return
		case e := <-ant.event:
			if !ant.Enabled {
				continue
			}
			exchangeOrder := UuidWithString(e.ID + OceanCore)
			if _, ok := ant.exOrders[exchangeOrder]; ok {
				continue
			}

			v, _ := prettyjson.Marshal(e)
			log.Printf("profit found, %s/%s\n  %s", Who(e.Base), Who(e.Quote), string(v))

			ant.exOrders[exchangeOrder] = false
			switch e.Category {
			case StrategyLow:
				if _, err := OceanSell(e.Price.String(), e.Amount.String(), OrderTypeLimit, e.Base, e.Quote, exchangeOrder); err != nil {
					log.Error(err)
					continue
				}

				select {
				case <-ant.orderMatched:
					otcOrder := UuidWithString(e.ID + ExinCore)
					equalAmount := e.Price.Mul(e.Amount)
					if _, err := ExinTrade(equalAmount.String(), e.Quote, e.Base, otcOrder); err == nil {
						ant.otcOrders[otcOrder] = true
					}
					ant.exOrders[exchangeOrder] = true
				case <-time.After(OrderConfirmedTime):
					ant.lock.Lock()
					ant.exOrders[exchangeOrder] = true
					ant.lock.Unlock()
					for i := 0; i < 3; i++ {
						OceanCancel(exchangeOrder)
						time.Sleep(100 * time.Millisecond)
					}
				}
			case StrategyHigh:
				equalAmount := e.Amount.Mul(e.Price)
				if _, err := OceanBuy(e.Price.String(), equalAmount.String(), OrderTypeLimit, e.Base, e.Quote, exchangeOrder); err != nil {
					log.Error(err)
					continue
				}

				select {
				case <-ant.orderMatched:
					ant.exOrders[exchangeOrder] = true
					otcOrder := UuidWithString(e.ID + ExinCore)
					if _, err := ExinTrade(e.Amount.String(), e.Base, e.Quote, otcOrder); err == nil {
						ant.otcOrders[otcOrder] = true
					}
				case <-time.After(OrderConfirmedTime):
					ant.lock.Lock()
					ant.exOrders[exchangeOrder] = true
					ant.lock.Unlock()
					for i := 0; i < 3; i++ {
						OceanCancel(exchangeOrder)
						time.Sleep(100 * time.Millisecond)
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
		amount := exchange.Amount.Round(ExinAssetPrecision(quote))
		if amount.LessThanOrEqual(otc.Min) {
			log.Errorf("---amount is too small, %v <= %v , (precision %v:%v)", amount, otc.Min, Who(quote), ExinAssetPrecision(quote))
			return
		}
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
	askPrice, price := exchange.Price, otc.Price
	askProfit := price.Sub(askPrice).Div(price)
	log.Debugf("ask -- ocean price: %10.8v, exin price: %10.8v, profit: %10.8v, %5v/%5v", exchange.Price, otc.Price, askProfit, Who(base), Who(quote))
	if askProfit.GreaterThan(decimal.NewFromFloat(ProfitThreshold)) {
		amount := exchange.Amount.Round(ExinAssetPrecision(base))
		if exchange.Amount.LessThanOrEqual(otc.Min) {
			log.Errorf("+++amount is too small, %v <= %v,  (min amount %v:%v)", exchange.Amount, otc.Min, Who(base), ExinAssetPrecision(base))
			return
		}
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
