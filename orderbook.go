package ant

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/emirpasic/gods/trees/redblacktree"
	"github.com/shopspring/decimal"
)

const (
	PageSideAsk          = "ASK"
	PageSideBid          = "BID"
	EventTypeOrderOpen   = "ORDER-OPEN"
	EventTypeOrderMatch  = "ORDER-MATCH"
	EventTypeOrderCancel = "ORDER-CANCEL"
	EventTypeBookT0      = "BOOK-T0"
	WrongSequenceError   = "WRONGSEQUENCE"
)

type OrderEvent struct {
	Market    string                 `json:"market"`
	Type      string                 `json:"event"`
	Sequence  string                 `json:"sequence"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

type Entry struct {
	Side   string          `json:"side"`
	Price  decimal.Decimal `json:"price"`
	Amount decimal.Decimal `json:"amount"`
	Funds  decimal.Decimal `json:"funds"`
}

type OrderBook struct {
	bids      *redblacktree.Tree
	asks      *redblacktree.Tree
	sequences map[string]bool
	previous  int
	pair      string
	trade     Trade
}

func NewBook(base, quote string) *OrderBook {
	return &OrderBook{
		bids:      redblacktree.NewWith(NewComparer(PageSideBid)),
		asks:      redblacktree.NewWith(NewComparer(PageSideAsk)),
		sequences: make(map[string]bool, 0),
		pair:      base + "-" + quote,
	}
}

func (book *OrderBook) GetDepth(limit int) *Depth {
	var depth Depth
	handler := func(tree *redblacktree.Tree, limit int) []Order {
		count := 0
		orders := make([]Order, 0, limit)
		for it := tree.Iterator(); it.Next(); {
			value := it.Value().(*Entry)
			order := Order{
				Price:  value.Price,
				Amount: value.Amount,
			}
			orders = append(orders, order)
			count += 1
			if count >= limit {
				it.End()
			}
		}
		return orders
	}

	depth.Bids = handler(book.bids, limit)
	depth.Asks = handler(book.asks, limit)
	return &depth
}

func NewComparer(side string) func(a, b interface{}) int {
	return func(a, b interface{}) int {
		entry := a.(decimal.Decimal)
		opponent := b.(decimal.Decimal)
		if entry == opponent {
			return 0
		}
		switch side {
		case PageSideAsk:
			return entry.Cmp(opponent)
		case PageSideBid:
			return opponent.Cmp(entry)
		default:
			log.Println("side", side, entry, opponent)
			return 0
		}
	}
}

func (book *OrderBook) OnOrderMessage(msg *BlazeMessage) error {
	bt, err := json.Marshal(msg.Data)
	if err != nil {
		return err
	}

	var e OrderEvent
	err = json.Unmarshal(bt, &e)
	if err != nil {
		return err
	}

	if _, ok := book.sequences[e.Sequence]; ok {
		return nil
	}

	now, err := strconv.Atoi(e.Sequence)
	if err != nil {
		return err
	}

	if book.previous > 0 {
		if now <= book.previous {
			return nil
		}
		if now != book.previous+1 {
			book.asks.Clear()
			book.bids.Clear()
			book.previous = 0
			book.sequences = make(map[string]bool, 0)
			return fmt.Errorf("%s, previous %v, but now %v", WrongSequenceError, book.previous, now)
		}
	}

	book.previous = now

	switch e.Type {
	case EventTypeOrderOpen, EventTypeOrderCancel:
		bt, err := json.Marshal(e.Data)
		if err != nil {
			return err
		}
		var entry Entry
		err = json.Unmarshal(bt, &entry)
		if err != nil {
			return err
		}

		update := func(tree *redblacktree.Tree, entry Entry) {
			if value, ok := tree.Get(entry.Price); ok {
				newEntry := value.(*Entry)
				if e.Type == EventTypeOrderCancel {
					entry.Amount = entry.Amount.Mul(decimal.NewFromFloat(-1.0))
				}
				newEntry.Amount = newEntry.Amount.Add(entry.Amount)
				newEntry.Funds = newEntry.Amount.Add(entry.Funds)
				if !newEntry.Amount.IsPositive() || !newEntry.Funds.IsPositive() {
					tree.Remove(entry.Price)
				} else {
					tree.Put(entry.Price, newEntry)
				}
			} else {
				if e.Type == EventTypeOrderOpen {
					tree.Put(entry.Price, &entry)
				}
			}
		}
		switch entry.Side {
		case PageSideAsk:
			update(book.asks, entry)
		case PageSideBid:
			update(book.bids, entry)
		default:
			return fmt.Errorf("wrong side. %v", entry.Side)
		}
	case EventTypeOrderMatch:
		bt, err := json.Marshal(e.Data)
		if err != nil {
			return err
		}
		var entry Entry
		err = json.Unmarshal(bt, &entry)
		if err != nil {
			return err
		}

		delete := func(tree *redblacktree.Tree, entry Entry) {
			if value, ok := tree.Get(entry.Price); ok {
				newEntry := value.(*Entry)
				newEntry.Amount = newEntry.Amount.Sub(entry.Amount)
				newEntry.Funds = newEntry.Amount.Sub(entry.Funds)
				if !newEntry.Amount.IsPositive() || !newEntry.Funds.IsPositive() {
					tree.Remove(entry.Price)
				} else {
					tree.Put(entry.Price, newEntry)
				}
			}
		}

		delete(book.asks, entry)
		delete(book.bids, entry)
		book.trade = Trade{
			Price:    entry.Price.String(),
			Amount:   entry.Amount.String(),
			Side:     entry.Side,
			CreateAt: time.Now().Format(time.RFC3339Nano),
		}
	case EventTypeBookT0:
		book.asks.Clear()
		book.bids.Clear()
		bt, err := json.Marshal(e.Data)
		if err != nil {
			return err
		}

		type Order struct {
			Side   string `json:"side"`
			Price  string `json:"price"`
			Amount string `json:"amount"`
			Funds  string `json:"funds"`
		}

		var depth struct {
			Asks []Order `json:"asks"`
			Bids []Order `json:"bids"`
		}
		err = json.Unmarshal(bt, &depth)
		if err != nil {
			return err
		}

		Add := func(tree *redblacktree.Tree, orders []Order, side string) {
			for _, order := range orders {
				price, _ := decimal.NewFromString(order.Price)
				amount, _ := decimal.NewFromString(order.Amount)
				funds, _ := decimal.NewFromString(order.Funds)
				entry := Entry{
					Side:   side,
					Price:  price,
					Amount: amount,
					Funds:  funds,
				}
				tree.Put(price, &entry)
			}
		}

		Add(book.bids, depth.Bids, PageSideBid)
		Add(book.asks, depth.Asks, PageSideAsk)
	}
	return nil
}
