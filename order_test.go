package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

//btc_order_trace:fa3b1472-b34e-4bdf-abd9-e15e28959aa2

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
	amount := 1.0 / price
	base, quote := BTC, USDT
	trace, err := ant.StrategyHigh(price, amount, base, quote)
	assert.Nil(t, err)
	fmt.Println(trace)
}

func TestExinTrade(t *testing.T) {
	ExinTrade(1, USDT, BTC)
	ExinTrade(0.0001, BTC, USDT)
}

func TestOceanTrade(t *testing.T) {
	OceanCore = F1exCore
	price, amount := 0.5, 0.001
	sellTrace, err := OceanSell(price, amount, "L", XIN, BTC)
	assert.Nil(t, err)
	fmt.Println("sellTrace: ", sellTrace)

	buyTrace, err := OceanBuy(price, amount*price, "L", XIN, BTC)
	assert.Nil(t, err)
	fmt.Println("buyTrace: ", buyTrace)
}

func TestOceanCancel(t *testing.T) {
	//OceanCore = F1exCore
	err := OceanCancel("fa3b1472-b34e-4bdf-abd9-e15e28959aa2")
	fmt.Println(err)
}
