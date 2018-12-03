package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/shopspring/decimal"

	prettyjson "github.com/hokaccha/go-prettyjson"
	"github.com/stretchr/testify/assert"
)

func TestStrategyLow(t *testing.T) {
	price := 4500.0
	amount := 1.5 / price
	base, quote := BTC, USDT
	if _, err := ExinTrade(amount*price, quote, base); err == nil {
		trace, err := OceanSell(price, amount, "L", base, quote)
		fmt.Println(trace, err)
	}
}

func TestStrategyHigh(t *testing.T) {
	price := 4500.0
	amount := 1.5 / price
	base, quote := BTC, USDT
	if trace, err := OceanBuy(price, amount*price, "L", base, quote); err == nil {
		fmt.Println(trace)
		ExinTrade(amount, base, quote)
	}
}

func TestGetExinDepth(t *testing.T) {
	ctx := context.Background()
	data, _ := GetExinDepth(ctx, BTC, USDT)
	v, _ := prettyjson.Marshal(&data)
	fmt.Println(string(v))

	a := data.Bids[0].Max.Exponent()
	b := data.Bids[0].Min.Exponent()
	c := data.Asks[0].Min.Exponent()
	fmt.Println(a, b, c)

	d := decimal.NewFromFloat(1.2456)
	e := decimal.NewFromFloat(0.01)
	fmt.Println(d.Round(-e.Exponent()))
}

func TestOceanDepth(t *testing.T) {
	ctx := context.Background()
	data, _ := GetOceanDepth(ctx, EOS, USDT)
	v, _ := prettyjson.Marshal(&data)
	fmt.Println(string(v))
}

func TestExinTrade(t *testing.T) {
	amount := 1.0
	trace, err := ExinTrade(amount, USDT, EOS)
	fmt.Println(trace, err)
}

func TestOceanTrade(t *testing.T) {
	//OceanCore = F1exCore
	price, amount := 2.8998, 0.3582
	sellTrace, err := OceanSell(price, amount, "L", EOS, USDT)
	assert.Nil(t, err)
	fmt.Println("sellTrace: ", sellTrace)

	// buyTrace, err := OceanBuy(price, amount*price, "L", XIN, BTC)
	// assert.Nil(t, err)
	// fmt.Println("buyTrace: ", buyTrace)
}

//51f73f1f-212e-48cd-b990-b5819716f8f7
func TestOceanCancel(t *testing.T) {
	//OceanCore = F1exCore
	err := OceanCancel("51f73f1f-212e-48cd-b990-b5819716f8f7")
	fmt.Println(err)
}

func TestUUIDl(t *testing.T) {
	//OceanCore = F1exCore
	e := Event{
		Category: "L",
		Base:     XIN,
		Quote:    USDT,
		Price:    decimal.NewFromFloat(550.0),
		Amount:   decimal.NewFromFloat(0.01),
	}
	id := UuidWithString(Who(e.Base) + Who(e.Quote) + e.Price.String() + e.Amount.String() + "L")
	fmt.Println(id)

	e = Event{
		Category: "L",
		Base:     XIN,
		Quote:    USDT,
		Price:    decimal.NewFromFloat(550.0),
		Amount:   decimal.NewFromFloat(0.011),
	}
	id = UuidWithString(Who(e.Base) + Who(e.Quote) + e.Price.String() + e.Amount.String() + "L")
	fmt.Println(id)

	id1 := UuidWithString(id + ExinCore)
	fmt.Println(id1)

	id2 := UuidWithString(id + OceanCore)
	fmt.Println(id2)
}

func TestOrderMemo(t *testing.T) {
	match := "hKFBsGIJVgm6i0IUtYnNDcHGzQihQrCDA1ErlVY6HJhhgmB0qGapoU+wAAAAAAAAAAAAAAAAAAAAAKFTpU1BVENI"
	var action TransferAction
	action.Unpack(match)
	fmt.Println(action.B)
	fmt.Println(action.A)

	var order OceanOrderAction
	order.Unpack("hKFQqTM5OTMuOTk4NqFUoUyhQbDG0McoJiRCm44N2dGbZZL6oVOhQg==")
	fmt.Println("order", order)
}

func TestReadAssets(t *testing.T) {
	data, _ := ReadAssets(context.TODO())
	v, _ := prettyjson.Marshal(data)
	fmt.Println(string(v))
}
