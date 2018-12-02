package main

import (
	"context"
	"fmt"
	"testing"

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
	data, _ := GetExinDepth(ctx, EOS, BTC)
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
