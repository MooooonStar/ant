package main

import (
	"context"
	"fmt"
	"testing"

	prettyjson "github.com/hokaccha/go-prettyjson"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestGetExinTicker(t *testing.T) {
	ctx := context.Background()
	data, _ := GetExinPrice(ctx, BTC, XIN)
	v, _ := prettyjson.Marshal(&data)
	fmt.Println(string(v))
}

func TestOceanOrder(t *testing.T) {
	ctx := context.Background()
	data, _ := GetOceanOrder(ctx, XIN, BTC)
	v, _ := prettyjson.Marshal(&data)
	fmt.Println(string(v))
}

func TestWatching(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	ant := Ant{
		e: make(chan Event, 0),
	}
	ctx := context.Background()
	base, quote := XIN, BTC
	depth, err := GetOceanOrder(ctx, base, quote)
	assert.Nil(t, err)
	if err == nil && depth != nil {
		if err := ant.Low(ctx, *depth, base, quote); err != nil {
			log.Println(err)
		}

		if err := ant.High(ctx, *depth, base, quote); err != nil {
			log.Println(err)
		}
	}
}
