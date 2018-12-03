package main

import (
	"context"

	log "github.com/sirupsen/logrus"
)

func main() {
	log.SetLevel(log.DebugLevel)
	ant := Ant{
		event:     make(chan Event, 0),
		snapshots: make(map[string]bool, 0),
		orders:    make(map[string]bool, 0),
	}
	ctx := context.Background()
	go ant.PollMixinNetwork(ctx)

	for _, baseSymbol := range []string{"BTC", "EOS", "XIN", "ETH"} {
		for _, quoteSymbol := range []string{"USDT", "BTC", "ETH"} {
			base := GetAssetId(baseSymbol)
			quote := GetAssetId(quoteSymbol)
			go ant.Watching(ctx, base, quote)
		}
	}
	ant.Run()
}
