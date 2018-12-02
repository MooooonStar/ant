package main

import (
	"context"
	"fmt"
	"testing"

	prettyjson "github.com/hokaccha/go-prettyjson"
	"github.com/stretchr/testify/assert"
)

func TestStrategyLow(t *testing.T) {
	ant := Ant{
		e: make(chan Event, 0),
	}
	price := 4000.0
	amount := 1.2 / price
	base, quote := BTC, USDT
	trace, err := ant.StrategyLow(price, amount, base, quote)
	assert.Nil(t, err)
	fmt.Println(trace)
}

func TestStrategyHigh(t *testing.T) {
	ant := Ant{
		e: make(chan Event, 0),
	}
	price := 4000.0
	amount := 1.2 / price
	base, quote := BTC, USDT
	trace, err := ant.StrategyHigh(price, amount, base, quote)
	assert.Nil(t, err)
	fmt.Println(trace)
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

func TestWatching(t *testing.T) {
	// log.SetLevel(log.DebugLevel)
	// ant := Ant{
	// 	e: make(chan Event, 0),
	// }
	// ctx := context.Background()
	// base, quote := BTC, USDT
	// depth, err := GetOceanDepth(ctx, base, quote)
	// assert.Nil(t, err)
	// if err == nil && depth != nil {
	// 	// if err := ant.Low(ctx, *depth, base, quote); err != nil {
	// 	// 	log.Println(err)
	// 	// }

	// 	// if err := ant.High(ctx, *depth, base, quote); err != nil {
	// 	// 	log.Println(err)
	// 	// }
	// }
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

//usdt
//911018ad-c761-4778-96dd-9dbd9047e3a4
func TestOceanCancel(t *testing.T) {
	//OceanCore = F1exCore
	err := OceanCancel("911018ad-c761-4778-96dd-9dbd9047e3a4")
	fmt.Println(err)
}
