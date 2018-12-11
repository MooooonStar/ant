package main

import (
	"context"
	"fmt"
	"testing"

	prettyjson "github.com/hokaccha/go-prettyjson"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestStrategyLow(t *testing.T) {
	price := 4500.0
	amount := 1.5 / price
	base, quote := BTC, USDT
	if _, err := ExinTrade(fmt.Sprint(amount*price), quote, base); err == nil {
		trace, err := OceanSell(fmt.Sprint(price), fmt.Sprint(amount), "L", base, quote)
		fmt.Println(trace, err)
	}
}

func TestStrategyHigh(t *testing.T) {
	price := 4500.0
	amount := 1.5 / price
	base, quote := BTC, USDT
	if trace, err := OceanBuy(fmt.Sprint(price), fmt.Sprint(amount*price), "L", base, quote); err == nil {
		fmt.Println(trace)
		ExinTrade(fmt.Sprint(amount), base, quote)
	}
}

func TestGetExinDepth(t *testing.T) {
	ctx := context.Background()
	data, _ := GetExinDepth(ctx, XIN, BTC)
	v, _ := prettyjson.Marshal(&data)
	fmt.Println(string(v))

	// a := data.Bids[0].Max.Exponent()
	// b := data.Bids[0].Min.Exponent()
	// c := data.Asks[0].Min.Exponent()
	// fmt.Println(a, b, c)

	// d := decimal.NewFromFloat(1.2456)
	// e := decimal.NewFromFloat(0.01)
	// fmt.Println(d.Round(-e.Exponent()))
}

func TestExinOrder(t *testing.T) {
	order, err := GetExinOrder(context.TODO(), XIN, BTC)
	assert.Nil(t, err)
	fmt.Println("order", order)
}

func TestOceanDepth(t *testing.T) {
	ctx := context.Background()
	data, _ := GetOceanDepth(ctx, BTC, USDT)
	v, _ := prettyjson.Marshal(&data)
	fmt.Println(string(v))
}

func TestExinTrade(t *testing.T) {
	//price, amount := 3936.6133, 0.0003
	trace, err := ExinTrade("0.0018", BTC, USDT)
	fmt.Println(trace, err)
}

func TestOceanTrade(t *testing.T) {
	//OceanCore = F1exCore
	price, amount := "3711", "0.001"
	sellTrace, err := OceanSell(price, amount, "L", BTC, USDT)
	assert.Nil(t, err)
	fmt.Println("sellTrace: ", sellTrace)

	// buyTrace, err := OceanBuy(price, amount*price, "L", XIN, BTC)
	// assert.Nil(t, err)
	// fmt.Println("buyTrace: ", buyTrace)
}

//201602a3-a0d2-439e-8e63-e1b6dec86b76
func TestOceanCancel(t *testing.T) {
	//OceanCore = F1exCore
	err := OceanCancel("201602a3-a0d2-439e-8e63-e1b6dec86b76")
	fmt.Println(err)
}

func TestUUIDl(t *testing.T) {
	//OceanCore = F1exCore
	e := ProfitEvent{
		Category: "L",
		Base:     EOS,
		Quote:    BTC,
		Price:    decimal.NewFromFloat(0.00087878),
		Amount:   decimal.NewFromFloat(0.1629),
	}
	id := UuidWithString(Who(e.Base) + Who(e.Quote) + e.Price.String() + e.Amount.String() + "L")
	fmt.Println(id)

	id2 := UuidWithString(id + OceanCore)
	fmt.Println(id2)

	// e = Event{
	// 	Category: "L",
	// 	Base:     XIN,
	// 	Quote:    USDT,
	// 	Price:    decimal.NewFromFloat(550.0),
	// 	Amount:   decimal.NewFromFloat(0.011),
	// }
	// id = UuidWithString(Who(e.Base) + Who(e.Quote) + e.Price.String() + e.Amount.String() + "L")
	// fmt.Println(id)

	// id1 := UuidWithString(id + ExinCore)
	// fmt.Println(id1)

	// id2 := UuidWithString(id + OceanCore)
	// fmt.Println(id2)
}

func TestOrderMemo(t *testing.T) {
	match := "hKFBsL8MIKGyNUwSvcuSnP4e9v+hQrDF0+nSQzU94J/kBLvI8OCToU+wAAAAAAAAAAAAAAAAAAAAAKFTpU1BVENI"
	var action TransferAction
	action.Unpack(match)
	fmt.Println(action)

	// var order OceanOrderAction
	// order.Unpack("hKFToUGhUKQzNzExoVShTKFBsIFbCxonZDc2j6pC1pT6Ygo=")
	// fmt.Println("order", order)
}

func TestReadAssets(t *testing.T) {
	data, _ := ReadAssets(context.TODO())
	v, _ := prettyjson.Marshal(data)
	fmt.Println(string(v))
}

func TestGetTrades(t *testing.T) {
	trades, err := GetOceanTrades(context.TODO(), BTC, USDT)
	assert.Nil(t, err)
	v, _ := prettyjson.Marshal(trades)
	fmt.Println(string(v))
}
