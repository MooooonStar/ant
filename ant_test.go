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
}

func TestOceanDepth(t *testing.T) {
	ctx := context.Background()
	data, _ := GetOceanDepth(ctx, XIN, BTC)
	v, _ := prettyjson.Marshal(&data)
	fmt.Println(string(v))
}

func TestExinTrade(t *testing.T) {
	ExinTrade(1, USDT, BTC)
	ExinTrade(0.0001, BTC, USDT)
}

func TestOceanTrade(t *testing.T) {
	//OceanCore = F1exCore
	price, amount := 0.5, 0.001
	sellTrace, err := OceanSell(price, amount, "L", XIN, BTC)
	assert.Nil(t, err)
	fmt.Println("sellTrace: ", sellTrace)

	buyTrace, err := OceanBuy(price, amount*price, "L", XIN, BTC)
	assert.Nil(t, err)
	fmt.Println("buyTrace: ", buyTrace)
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
	match := "hKFBsHG/D/cnZkcEpnVRBA+6lcKhQrDgnuKAFu5M8IdN7SkU6dWaoU+wAAAAAAAAAAAAAAAAAAAAAKFTpU1BVENI"
	var action TransferAction
	action.Unpack(match)
	assert.Equal(t, "MATCH", action.S)

	cancel := "hKFBsAAAAAAAAAAAAAAAAAAAAAChQrAAAAAAAAAAAAAAAAAAAAAAoU+wHP59AZJKQVGdKFg/A6sB2KFTpkNBTkNFTA=="
	var transfer TransferAction
	transfer.Unpack(cancel)
	assert.Equal(t, "CANCEL", transfer.S)

	fmt.Println(action, transfer)
}

func TestReadAssets(t *testing.T) {
	data, _ := ReadAssets(context.TODO())
	v, _ := prettyjson.Marshal(data)
	fmt.Println(string(v))
}
