package ant

import (
	"context"
	"crypto/md5"
	"io"
	"log"
	"sync"
	"time"

	bot "github.com/MixinNetwork/bot-api-go-client"
	"github.com/emirpasic/gods/lists/arraylist"
	uuid "github.com/satori/go.uuid"
	"github.com/shopspring/decimal"
)

const (
	ProfitThreshold = 0.010 / (1 - OceanFee) / (1 - ExinFee)
	OceanFee        = 0.001
	ExinFee         = 0.003
	OrderExpireTime = int64(10 * time.Second)

	//Ocean One上订单未成交
	StatusPending = "Pending"
	//OceanOne上订单成交但Exin上未成交，最有可能是受Exin最小数量限制
	StatusFailed = "Failed"
	//订单未成交，全部退款或者搬砖成功
	StatusSuccess = "Success"
	//状态为Failed的订单，会累计到一定数量后集中去exin上处理
	StatusDone = "Done"
)

type ProfitEvent struct {
	ID            string          `json:"-"                gorm:"type:varchar(36);primary_key"`
	CreatedAt     time.Time       `json:"created_at"       gorm:"precision:6"`
	Category      string          `json:"category"         gorm:"type:varchar(10)"`
	Price         decimal.Decimal `json:"price"            gorm:"type:varchar(36)"`
	Profit        decimal.Decimal `json:"profit"           gorm:"type:varchar(36)"`
	Amount        decimal.Decimal `json:"amount"           gorm:"type:varchar(36)"`
	Min           decimal.Decimal `json:"min"              gorm:"type:varchar(36)"`
	Max           decimal.Decimal `json:"max"              gorm:"type:varchar(36)"`
	Base          string          `json:"base"             gorm:"type:varchar(36);INDEX"`
	Quote         string          `json:"quote"            gorm:"type:varchar(36);INDEX"`
	Expire        int64           `json:"expire"           gorm:"type:bigint(36)"`
	BaseAmount    decimal.Decimal `json:"base_amount"      gorm:"type:varchar(36)"`
	QuoteAmount   decimal.Decimal `json:"quote_amount"     gorm:"type:varchar(36)"`
	ExchangeOrder string          `json:"exchange_order"   gorm:"type:varchar(36);"`
	OtcOrder      string          `json:"otc_order"        gorm:"type:varchar(36);"`
	Status        string          `json:"status"           gorm:"INDEX;type:varchar(10);default:'Pending'"`
}

func (ProfitEvent) TableName() string {
	return "ant_profit_events"
}

type Ant struct {
	//是否开启交易
	enableExin bool
	//发现套利机会
	event chan *ProfitEvent
	//所有交易的snapshot_id
	snapshots map[string]bool
	//机器人向ocean.one交易的trace_id
	orders map[string]bool
	//买单和卖单的红黑树，生成深度用
	books      map[string]*OrderBook
	OrderQueue *arraylist.List
	assetsLock sync.Mutex
	assets     map[string]decimal.Decimal
	client     *bot.BlazeClient
	mutexes    *tmap
}

func NewAnt() *Ant {
	return &Ant{
		enableExin: false,
		event:      make(chan *ProfitEvent, 20),
		snapshots:  make(map[string]bool, 0),
		orders:     make(map[string]bool, 0),
		books:      make(map[string]*OrderBook, 0),
		assets:     make(map[string]decimal.Decimal, 0),
		OrderQueue: arraylist.New(),
		client:     bot.NewBlazeClient(ClientId, SessionId, PrivateKey),
		mutexes:    newTmap(),
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

func (ant *Ant) OnOrderMessage(base, quote string) *OrderBook {
	pair := base + "-" + quote
	ant.books[pair] = NewBook(base, quote)
	return ant.books[pair]
}

func (ant *Ant) Clean() {
	for trace, ok := range ant.orders {
		if !ok {
			ant.OceanCancel(trace)
		}
	}
	//TODO, event中baseAmount和quoteAmout的数量和预期不一致
	log.Println("+++exit because ctrl-c++++")
}

func (ant *Ant) trade(ctx context.Context, e *ProfitEvent) error {
	exchangeOrder := UuidWithString(e.ID + OceanCore)
	if _, ok := ant.orders[exchangeOrder]; ok {
		return nil
	}

	defer func() {
		time.AfterFunc(time.Duration(OrderExpireTime), func() {
			if done, ok := ant.orders[exchangeOrder]; ok && !done {
				if err := ant.OceanCancel(exchangeOrder); err == nil {
					ant.orders[exchangeOrder] = true
				}
			}
		})

		go ant.Notice(ctx, *e)
	}()

	//多付款，保证扣完手续费后能清空挂单
	amount := e.Amount.Mul(decimal.NewFromFloat(1.1))

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

	ant.orders[exchangeOrder] = false
	_, err := ant.OceanTrade(e.Category, e.Price.String(), amount.String(), OrderTypeLimit, e.Base, e.Quote, exchangeOrder)
	if err != nil {
		return err
	}

	e.ExchangeOrder = exchangeOrder
	ant.OrderQueue.Add(e)

	if err := Database(ctx).FirstOrCreate(e).Error; err != nil {
		return err
	}
	return nil
}

func LimitAmount(amount, balance, min, max decimal.Decimal) decimal.Decimal {
	if amount.LessThanOrEqual(min) {
		log.Printf("amount too small, %v < min : %v", amount, min)
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
			removed := make([]*ProfitEvent, 0)
			for it := ant.OrderQueue.Iterator(); it.Next(); {
				event := it.Value().(*ProfitEvent)
				//获利了结或者未成交全退款的订单
				if !event.BaseAmount.IsNegative() && !event.QuoteAmount.IsNegative() {
					event.Status = StatusSuccess
					removed = append(removed, event)
				}
				//OceanOne上未成交也未收到退款的订单和成交数额太小，exin上无法卖出的订单
				if event.CreatedAt.Add(time.Duration(event.Expire)).Add(2 * time.Minute).Before(time.Now()) {
					//本不需要这里再取消一次的，但实际情况不是，可能是并发转账时出错了
					if done, ok := ant.orders[event.ExchangeOrder]; ok && !done {
						if err := ant.OceanCancel(event.ExchangeOrder); err == nil {
							ant.orders[event.ExchangeOrder] = true
						}
					}

					//只将成交数额小的订单标记为Failed
					//if event.BaseAmount.Mul(event.QuoteAmount).IsNegative() {
					if !event.BaseAmount.IsZero() && !event.QuoteAmount.IsZero() {
						event.Status = StatusFailed
						removed = append(removed, event)
					} else {
						//退款有可能3min后才收到
						//Nothing need to do
					}
				}
				//每笔订单都会取消，这里留3s接收取消订单请求发出后仍成交的钱款。
				if event.CreatedAt.Add(time.Duration(event.Expire)).Add(3 * time.Second).Before(time.Now()) {
					amount := event.BaseAmount
					send, side := event.Base, PageSideAsk
					if !amount.IsPositive() {
						amount = event.QuoteAmount
						send, side = event.Quote, PageSideBid
						if !amount.IsPositive() {
							continue
						}
					}

					ant.assetsLock.Lock()
					balance := ant.assets[send]
					ant.assetsLock.Unlock()

					var limited decimal.Decimal
					if send == event.Base {
						limited = LimitAmount(amount, balance, event.Min, event.Max)
					} else if send == event.Quote {
						limited = LimitAmount(amount, balance, event.Min.Mul(event.Price), event.Max.Mul(event.Price))
					}

					if !limited.IsPositive() || !ant.enableExin {
						log.Printf("%s, balance: %v, min: %v, send: %v,amount: %v, limited: %v, enable: %v", Who(send), balance, event.Min, send, amount, limited, ant.enableExin)
					} else {
						otcOrder := UuidWithString(event.ID + ExinCore)
						if _, err := ant.ExinTrade(side, limited.String(), event.Base, event.Quote, otcOrder); err != nil {
							log.Println(err)
							continue
						}
						event.OtcOrder = otcOrder
					}
					ant.orders[event.ExchangeOrder] = true
				}
			}
			if len(removed) > 0 {
				for _, event := range removed {
					index := ant.OrderQueue.IndexOf(event)
					ant.OrderQueue.Remove(index)
					updates := map[string]interface{}{
						"base_amount":  event.BaseAmount,
						"quote_amount": event.QuoteAmount,
						"otc_order":    event.OtcOrder,
						"status":       event.Status,
					}
					if err := Database(ctx).Model(event).Where("id=?", event.ID).Updates(updates).Error; err != nil {
						log.Println("update event error", err)
					}
				}
			}
		}
	}
}

//处理受exin上xin最小数量限制无法成交的订单
func (ant *Ant) CleanUpTheMess(ctx context.Context) error {
	var mess []struct {
		Base        string
		Quote       string
		BaseAmount  decimal.Decimal
		QuoteAmount decimal.Decimal
	}
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
			//价格可能波动，只处理最近10min的订单
			to, from := time.Now(), time.Now().Add(-10*time.Minute)
			if err := Database(ctx).Model(&ProfitEvent{}).Where("created_at > ? AND created_at <  ? AND status = ?", from, to, StatusFailed).
				Select("base, quote, SUM(base_amount) AS base_amount, SUM(quote_amount) AS quote_amount").
				Group("base, quote").Scan(&mess).Error; err != nil {
				continue
			}
			for _, m := range mess {
				if m.BaseAmount.IsPositive() && m.QuoteAmount.IsPositive() {
					continue
				}
				side, amount := PageSideAsk, m.BaseAmount
				if m.QuoteAmount.IsPositive() {
					side, amount = PageSideBid, m.QuoteAmount
				}

				var event ProfitEvent
				if err := Database(ctx).Where("base = ? AND quote = ?", m.Base, m.Quote).Order("created_at DESC").First(&event).Error; err != nil {
					continue
				}
				trace := UuidWithString(side + amount.String() + m.Base + m.Quote)
				var limited decimal.Decimal
				balance := decimal.NewFromFloat(1000000)
				if side == PageSideAsk {
					limited = LimitAmount(amount, balance, event.Min, event.Max)
				} else if side == PageSideBid {
					limited = LimitAmount(amount, balance, event.Min.Mul(event.Price), event.Max.Mul(event.Price))
				}

				if limited.IsPositive() && ant.enableExin {
					if _, err := ant.ExinTrade(side, limited.String(), m.Base, m.Quote, trace); err == nil {
						Database(ctx).Model(&ProfitEvent{}).Where("created_at > ? AND created_at < ?", from, to).
							Where("status = ? AND base = ? AND quote = ?", StatusFailed, m.Base, m.Quote).
							Update(ProfitEvent{Status: StatusDone, OtcOrder: trace})
					}
				}
			}
		}
	}
}

func (ant *Ant) HandleSnapshot(ctx context.Context, s *Snapshot) error {
	if len(s.Data) == 0 || s.OpponentId == MasterID {
		return nil
	}
	amount, _ := decimal.NewFromString(s.Amount)
	matched := &ProfitEvent{}
	for it := ant.OrderQueue.Iterator(); it.Next(); {
		event := it.Value().(*ProfitEvent)
		if s.OpponentId == ExinCore {
			var reply ExinReply
			if err := reply.Unpack(s.Data); err != nil {
				return err
			}
			if event.OtcOrder == s.TraceId || event.OtcOrder == reply.O.String() {
				matched = event
				it.End()
			}
		} else {
			var reply OceanReply
			if err := reply.Unpack(s.Data); err != nil {
				return err
			}
			if event.ExchangeOrder == s.TraceId || event.ExchangeOrder == reply.A.String() || event.ExchangeOrder == reply.B.String() || event.ExchangeOrder == reply.O.String() {
				matched = event
				it.End()
			}
		}
	}

	if s.AssetId == matched.Base {
		matched.BaseAmount = matched.BaseAmount.Add(amount)
	} else if s.AssetId == matched.Quote {
		matched.QuoteAmount = matched.QuoteAmount.Add(amount)
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
			if err := ant.trade(ctx, e); err != nil {
				log.Println(err)
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
			//if otc, err := GetExinDepth(ctx, base, quote); err == nil {
			if otc, err := FetchExinDepth(ctx, base, quote); err == nil {
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
	}

	profit := exchange.Price.Sub(otc.Price).Div(otc.Price)
	if side == PageSideAsk {
		profit = profit.Mul(decimal.NewFromFloat(-1.0))
	}

	if profit.LessThan(decimal.NewFromFloat(ProfitThreshold)) {
		return
	}

	id := UuidWithString(ClientId + exchange.Price.String() + exchange.Amount.String() + category + Who(base) + Who(quote))
	amount := exchange.Amount
	event := ProfitEvent{
		ID:          id,
		Category:    category,
		Price:       exchange.Price,
		Amount:      amount,
		Min:         otc.Min,
		Max:         otc.Max,
		Profit:      profit,
		Base:        base,
		Quote:       quote,
		Expire:      expire,
		CreatedAt:   time.Now(),
		BaseAmount:  decimal.Zero,
		QuoteAmount: decimal.Zero,
	}
	select {
	case ant.event <- &event:
	case <-time.After(5 * time.Second):
	}
	return
}

func (ant *Ant) UpdateBalance(ctx context.Context) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	update := func() {
		assets, _, err := ReadAssets(ctx)
		if err != nil {
			return
		}
		for symbol, balance := range assets {
			asset := GetAssetId(symbol)
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
