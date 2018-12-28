package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"testing"

	prettyjson "github.com/hokaccha/go-prettyjson"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestGetExinDepth(t *testing.T) {
	ctx := context.Background()
	data, _ := GetExinDepth(ctx, XIN, BTC)
	v, _ := prettyjson.Marshal(&data)
	fmt.Println(string(v))
}

func TestExinOrder(t *testing.T) {
	order, err := GetExinOrder(context.TODO(), EOS, USDT)
	assert.Nil(t, err)
	fmt.Println("order", order)
}

func TestOceanTrade(t *testing.T) {
	order, err := OceanTrade(PageSideAsk, "0", "1.2", "M", EOS, BTC)
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
	trace, err := ExinTrade(PageSideAsk, "0.6", EOS, BTC)
	fmt.Println(trace, err)
}

//201602a3-a0d2-439e-8e63-e1b6dec86b76
func TestOceanCancel(t *testing.T) {
	//OceanCore = F1exCore
	err := OceanCancel("1a282e67-0ad8-3f86-a94c-2d1fdf871b01")
	fmt.Println(err, "")
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
	match := "hKFBsO8Id5AaGjcxppQ2d6tcyzahQrBE7ELXHndB37BYRgNXcaMjoU+wAAAAAAAAAAAAAAAAAAAAAKFTpU1BVENI"
	var action OceanTransfer
	action.Unpack(match)
	fmt.Println(action.A, action.B, action.O, action.S)
}

func TestReadAssets(t *testing.T) {
	data, _ := ReadAssets(context.TODO())
	v, _ := prettyjson.Marshal(data)
	fmt.Println(string(v))
}

func TestSumAssets(t *testing.T) {
	prices, err := GetExinPrices(context.Background(), BTC)
	if err != nil {
		panic(err)
	}

	log.Println("prices", prices)

	sum := 0.0
	for asset, amount := range Wallet {
		price, _ := strconv.ParseFloat(prices[asset], 64)
		sum += price * amount
	}

	log.Println("sum", sum)
}
