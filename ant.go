package main

import (
	"context"
	"crypto/md5"
	"io"
	"sync"
	"time"

	"github.com/emirpasic/gods/lists/arraylist"
	uuid "github.com/satori/go.uuid"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
)

const (
	ProfitThreshold = 0.001 / (1 - OceanFee) / (1 - ExinFee) / (1 - HuobiFee)
	OceanFee        = 0.001
	ExinFee         = 0.001
	HuobiFee        = 0.001
	OrderExpireTime = int64(5 * time.Second)
)

type ProfitEvent struct {
	ID            string          `json:"-"`
	Category      string          `json:"category"`
	Price         decimal.Decimal `json:"price"`
	Profit        decimal.Decimal `json:"profit"`
	Amount        decimal.Decimal `json:"amount"`
	Min           decimal.Decimal `json:"min"`
	Max           decimal.Decimal `json:"max"`
	Base          string          `json:"base"`
	Quote         string          `json:"quote"`
	CreatedAt     time.Time       `json:"created_at"`
	Expire        int64           `json:"-"`
	BaseAmount    decimal.Decimal `json:"-"`
	QuoteAmount   decimal.Decimal `json:"-"`
	ExchangeOrder string          `json:"-"`
}

type Ant struct {
	//是否开启交易
	Enable bool
	//发现套利机会
	event chan *ProfitEvent
	//所有交易的snapshot_id
	snapshots map[string]bool
	orders    map[string]bool
	//买单和卖单的红黑树，生成深度用
	books      map[string]*OrderBook
	orderQueue *arraylist.List
	assetsLock sync.Mutex
	assets     map[string]decimal.Decimal
}

func NewAnt(enable bool) *Ant {
	return &Ant{
		Enable:     enable,
		event:      make(chan *ProfitEvent, 10),
		snapshots:  make(map[string]bool, 0),
		orders:     make(map[string]bool, 0),
		books:      make(map[string]*OrderBook, 0),
		assets:     make(map[string]decimal.Decimal, 0),
		orderQueue: arraylist.New(),
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

func (ant *Ant) Clean() {
	log.Info("++++++++++Cancel orders before exit.+++++++++++", ant.orders)
	for trace, ok := range ant.orders {
		if !ok {
			OceanCancel(trace)
		}
	}
}

func (ant *Ant) trade(e *ProfitEvent) error {
	exchangeOrder := UuidWithString(e.ID + OceanCore)
	if _, ok := ant.orders[exchangeOrder]; ok {
		return nil
	}

	if !ant.Enable {
		ant.orders[exchangeOrder] = true
		return nil
	}

	defer func() {
		go func(trace string) {
			select {
			case <-time.After(time.Duration(OrderExpireTime)):
				OceanCancel(trace)
				ant.orders[exchangeOrder] = true
			}
		}(exchangeOrder)
	}()

	ant.orders[exchangeOrder] = false

	amount := e.Amount
	ant.assetsLock.Lock()
	baseBalance := ant.assets[e.Base]
	quoteBalance := ant.assets[e.Quote]
	ant.assetsLock.Unlock()
	if amount.GreaterThan(baseBalance) {
		amount = baseBalance
	}
	if e.Category == PageSideBid {
		amount = e.Amount.Mul(e.Price)
		if amount.GreaterThan(quoteBalance) {
			amount = quoteBalance
		}
	}
	_, err := OceanTrade(e.Category, e.Price.String(), amount.String(), OrderTypeLimit, e.Base, e.Quote, exchangeOrder)
	if err != nil {
		return err
	}

	amount = amount.Mul(decimal.NewFromFloat(-1.0))
	if e.Category == PageSideBid {
		e.QuoteAmount = amount
	} else {
		e.BaseAmount = amount
	}
	e.ExchangeOrder = exchangeOrder
	ant.orderQueue.Add(e)
	return nil
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

func (ant *Ant) OnExpire(ctx context.Context) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			for it := ant.orderQueue.Iterator(); it.Next(); {
				event := it.Value().(*ProfitEvent)
				if event.CreatedAt.Add(time.Duration(2 * event.Expire)).Before(time.Now()) {
					log.Info("+++++++++Expired+++++++++++++")
					amount := event.BaseAmount
					send, get := event.Base, event.Quote
					if amount.IsNegative() {
						amount = event.QuoteAmount
						send, get = event.Quote, event.Base
						if amount.IsNegative() {
							panic(amount)
						}
					}

					ant.assetsLock.Lock()
					balance := ant.assets[send]
					ant.assetsLock.Unlock()
					limited := LimitAmount(amount, balance, event.Min, event.Max)

					log.Info("EXIN--------", limited, Who(send), Who(get))

					if !limited.IsPositive() {
						log.Errorf("%s, balance: %v, min: %v, send: %v", Who(send), balance, event.Min, send)
					} else {
						if _, err := ExinTrade(limited.String(), send, get); err != nil {
							log.Error(err)
							continue
						}
					}
					ant.orderQueue.Remove(it.Index())
					ant.orders[event.ExchangeOrder] = true
				}
			}
		}
	}
}

func (ant *Ant) HandleSnapshot(ctx context.Context, s *Snapshot) error {
	if s.SnapshotId == ExinCore {
		return nil
	}
	amount, _ := decimal.NewFromString(s.Amount)
	if amount.IsNegative() {
		return nil
	}

	for it := ant.orderQueue.Iterator(); it.Next(); {
		event := it.Value().(*ProfitEvent)
		var order OceanTransfer
		if err := order.Unpack(s.Data); err != nil {
			return err
		}

		if event.ExchangeOrder != order.A.String() &&
			event.ExchangeOrder != order.B.String() &&
			event.ExchangeOrder != order.O.String() {
			continue
		}

		log.Info("++++++++++Order Matched++++++++++")

		if s.AssetId == event.Base {
			event.BaseAmount.Add(amount)
		} else if s.AssetId == event.Quote {
			event.QuoteAmount.Add(amount)
		} else {
			panic(s.AssetId)
		}
		it.End()
	}
	return nil
}

func (ant *Ant) Trade(ctx context.Context) error {
	go ant.OnExpire(ctx)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case e := <-ant.event:
			if err := ant.trade(e); err != nil {
				log.Error(err)
				return err
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
			if otc, err := GetExinDepth(ctx, base, quote); err == nil {
				pair := base + "-" + quote
				if exchange := ant.books[pair].GetDepth(3); exchange != nil {
					if len(exchange.Bids) > 0 && len(otc.Asks) > 0 {
						ant.Inspect(ctx, exchange.Bids[0], otc.Asks[0], base, quote, PageSideBid, OrderExpireTime)
					}

					if len(exchange.Asks) > 0 && len(otc.Bids) > 0 {
						ant.Inspect(ctx, exchange.Asks[0], otc.Bids[0], base, quote, PageSideAsk, OrderExpireTime)
					}
				}
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (ant *Ant) Inspect(ctx context.Context, exchange, otc Order, base, quote string, side string, expire int64) {
	var category string
	if side == PageSideBid {
		category = PageSideAsk
	} else if side == PageSideAsk {
		category = PageSideBid
	} else {
		panic(category)
	}

	profit := exchange.Price.Sub(otc.Price).Div(otc.Price)
	if side == PageSideAsk {
		profit = profit.Mul(decimal.NewFromFloat(-1.0))
	}
	log.Debugf("%s --amount:%10.8v, ocean price: %10.8v, exin price: %10.8v, profit: %10.8v, %5v/%5v", side, exchange.Amount.Round(8), exchange.Price, otc.Price, profit, Who(base), Who(quote))
	if profit.LessThan(decimal.NewFromFloat(ProfitThreshold)) {
		return
	}
	id := UuidWithString(exchange.Price.String() + exchange.Amount.String() + category + Who(base) + Who(quote))
	log.Infof("%s --amount:%10.8v, ocean price: %10.8v, exin price: %10.8v, profit: %10.8v, %5v/%5v", side, exchange.Amount.Round(8), exchange.Price, otc.Price, profit, Who(base), Who(quote))
	log.Info("id+++++++", id)
	ant.event <- &ProfitEvent{
		ID:        id,
		Category:  category,
		Price:     exchange.Price,
		Amount:    exchange.Amount,
		Min:       otc.Min,
		Max:       otc.Max,
		Profit:    profit,
		Base:      base,
		Quote:     quote,
		Expire:    expire,
		CreatedAt: time.Now(),
	}
	return
}

func (ant *Ant) UpdateBalance(ctx context.Context) error {
	ticker := time.NewTicker(5 * time.Second)
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
