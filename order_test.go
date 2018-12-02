package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExinTrade(t *testing.T) {
	ExinTrade(1, USDT, BTC)
	ExinTrade(0.0001, BTC, USDT)
}

func TestOceanTrade(t *testing.T) {
	OceanCore = F1exCore
	price, amount := 0.5, 0.0001
	sellTrace, err := OceanSell(price, amount/price, "L", XIN, BTC)
	assert.Nil(t, err)
	fmt.Println("sellTrace: ", sellTrace)

	buyTrace, err := OceanBuy(price, amount, "L", XIN, BTC)
	assert.Nil(t, err)
	fmt.Println("buyTrace: ", buyTrace)
}

func TestOceanCancel(t *testing.T) {
	OceanCore = F1exCore
	err := OceanCancel("e92a900c-5c62-4d3b-9661-3f8625d9ccd9")
	fmt.Println(err)
}
