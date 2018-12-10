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
	OrderConfirmedTime = 10 * time.Second
)

type ProfitEvent struct {
	ID       string          `json:"-"`
	Category string          `json:"category"`
	Price    decimal.Decimal `json:"price"`
	Profit   decimal.Decimal `json:"profit"`
	Amount   decimal.Decimal `json:"amount"`
	Base     string          `json:"base"`
	Quote    string          `json:"quote"`
}

type Ant struct {
	//是否开启交易
	Enable bool
	//发现套利机会
	event     chan ProfitEvent
	snapshots map[string]bool
	orders    map[string]bool
	//买单和卖单的红黑树，生成深度用
	books         map[string]*OrderBook
	assets        map[string]decimal.Decimal
	matchedAmount chan decimal.Decimal
	assetsLock    sync.Mutex
	orderLock     sync.Mutex
}

func NewAnt(enable bool) *Ant {
	return &Ant{
		Enable:        enable,
		event:         make(chan ProfitEvent, 0),
		snapshots:     make(map[string]bool, 0),
		orders:        make(map[string]bool, 0),
		books:         make(map[string]*OrderBook, 0),
		assets:        make(map[string]decimal.Decimal, 0),
		matchedAmount: make(chan decimal.Decimal, 0),
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

func (ant *Ant) OnMessage(base, quote string) *OrderBook {
	pair := base + "-" + quote
	ant.books[pair] = NewBook(base, quote)
	return ant.books[pair]
}

func (ant *Ant) Trade(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			//退出时取消未完成的订单
			for trace, ok := range ant.orders {
				if !ok {
					OceanCancel(trace)
				}
			}
			return
		case e := <-ant.event:
			exchangeOrder := UuidWithString(e.ID + OceanCore)
			if _, ok := ant.orders[exchangeOrder]; ok {
				continue
			}

			v, _ := prettyjson.Marshal(e)
			log.Infof("profit found, %s/%s\n  %s", Who(e.Base), Who(e.Quote), string(v))

			if !ant.Enable {
				ant.orders[exchangeOrder] = true
				continue
			}

			ant.orders[exchangeOrder] = false
			switch e.Category {
			case PageSideBid:
				amount := e.Amount.Mul(e.Price)
				if _, err := OceanBuy(e.Price.String(), amount.String(), OrderTypeLimit, e.Base, e.Quote, exchangeOrder); err != nil {
					log.Error(err)
					continue
				}
			case PageSideAsk:
				if _, err := OceanSell(e.Price.String(), e.Amount.String(), OrderTypeLimit, e.Base, e.Quote, exchangeOrder); err != nil {
					log.Error(err)
					continue
				}
			default:
				panic(e)
			}

			select {
			case amount := <-ant.matchedAmount:
				ant.orderLock.Lock()
				ant.orders[exchangeOrder] = true
				ant.orderLock.Unlock()

				otcOrder := UuidWithString(e.ID + ExinCore)
				send, get := e.Base, e.Quote
				if e.Category == PageSideAsk {
					send, get = e.Quote, e.Base
				}
				if _, err := ExinTrade(amount.String(), send, get, otcOrder); err != nil {
					log.Error(err)
				}
			case <-time.After(OrderConfirmedTime):
			}
			ant.orderLock.Lock()
			ant.orders[exchangeOrder] = true
			ant.orderLock.Unlock()

			//无论是否成交，均取消订单
			OceanCancel(exchangeOrder)
		}
	}
}

func (ant *Ant) Watching(ctx context.Context, base, quote string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if otc, err := GetExinDepth(ctx, base, quote); err == nil {
				pair := base + "-" + quote
				if exchange := ant.books[pair].GetDepth(3); exchange != nil {
					if len(exchange.Bids) > 0 && len(otc.Bids) > 0 {
						ant.Strategy(ctx, exchange.Bids[0], otc.Bids[0], base, quote, PageSideBid)
					}

					if len(exchange.Asks) > 0 && len(otc.Asks) > 0 {
						ant.Strategy(ctx, exchange.Asks[0], otc.Asks[0], base, quote, PageSideAsk)
					}
				}
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func LimitAmount(amount, balance, min, max decimal.Decimal) decimal.Decimal {
	if amount.LessThanOrEqual(min) {
		return decimal.Zero
	}

	less := max
	if max.GreaterThan(balance) {
		less = balance
	}
	if amount.GreaterThan(less) {
		return less
	}
	return amount
}

func (ant *Ant) Strategy(ctx context.Context, exchange, otc Order, base, quote string, side string) {
	var category string
	if side == PageSideBid {
		category = PageSideAsk
	} else if side == PageSideAsk {
		category = PageSideBid
	} else {
		return
	}

	profit := exchange.Price.Sub(otc.Price).Div(otc.Price)
	if side == PageSideAsk {
		profit = profit.Mul(decimal.NewFromFloat(-1.0))
	}
	log.Debugf("%s -- ocean price: %10.8v, exin price: %10.8v, profit: %10.8v, %5v/%5v", side, exchange.Price, otc.Price, profit, Who(base), Who(quote))
	if profit.LessThan(decimal.NewFromFloat(ProfitThreshold)) {
		return
	}

	ant.assetsLock.Lock()
	balance := ant.assets[base]
	if category == PageSideBid {
		balance = ant.assets[quote].Div(exchange.Price)
	}
	ant.assetsLock.Unlock()
	amount := LimitAmount(exchange.Amount, balance, otc.Min, otc.Max)
	if !amount.IsPositive() {
		return
	}
	id := UuidWithString(Who(base) + Who(quote) + exchange.Price.String() + exchange.Amount.String() + category)
	ant.event <- ProfitEvent{
		ID:       id,
		Category: category,
		Price:    exchange.Price,
		Amount:   amount,
		Profit:   profit,
		Base:     base,
		Quote:    quote,
	}
	return
}

func (ant *Ant) UpdateBalance(ctx context.Context) error {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	update := func() {
		assets, err := ReadAssets(ctx)
		if err != nil {
			return
		}
		for asset, balance := range assets {
			b, err := decimal.NewFromString(balance)
			if err == nil && !b.Equal(ant.assets[asset]) {
				ant.assetsLock.Lock()
				ant.assets[asset] = b
				ant.assetsLock.Unlock()
			}
		}
	}

	update()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			update()
		}
	}
}
