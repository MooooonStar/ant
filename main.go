package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"time"

	"github.com/hokaccha/go-prettyjson"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func main() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	app := cli.NewApp()
	app.Commands = []cli.Command{
		{
			Name:  "info",
			Usage: "show assets",
			Action: func(c *cli.Context) error {
				assets, err := ReadAssets(context.TODO())
				if err != nil {
					return err
				}
				balance := make(map[string]string, 0)
				for asset, amount := range assets {
					balance[Who(asset)] = amount
				}
				v, _ := prettyjson.Marshal(balance)
				fmt.Println(string(v))
				return nil
			},
		},
		{
			Name:  "cancel",
			Usage: "cancel order in ocean.one by snapshot or trace",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "snapshot,s"},
				cli.StringFlag{Name: "trace,t"},
			},
			Action: func(c *cli.Context) error {
				snapshot := c.String("snapshot")
				trace, err := ReadSnapshot(context.TODO(), snapshot)
				if err != nil {
					return err
				}
				if traceId := c.String("trace"); len(traceId) > 0 {
					trace = traceId
				}
				if len(trace) == 0 {
					return fmt.Errorf("snapshot or trace id is required.")
				}
				return OceanCancel(trace)
			},
		},
		{
			Name:  "run",
			Usage: "find profits between different exchanges",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "pair"},
			},
			Action: func(c *cli.Context) error {
				log.SetLevel(log.DebugLevel)

				pair := c.String("pair")
				symbols := strings.Split(pair, "/")
				var baseSymbol, quoteSymbol string
				if len(symbols) == 2 {
					baseSymbol, quoteSymbol = symbols[0], symbols[1]
				}
				baseSymbols := []string{"BTC", "EOS", "XIN"}
				quoteSymbols := []string{"USDT", "BTC"}

				if len(baseSymbol) > 0 && len(quoteSymbol) > 0 {
					baseSymbols = []string{baseSymbol}
					quoteSymbols = []string{quoteSymbol}
				}

				ctx := context.Background()
				subctx, cancel := context.WithCancel(ctx)
				ant := NewAnt()
				go ant.PollMixinNetwork(subctx)
				for _, baseSymbol := range baseSymbols {
					for _, quoteSymbol := range quoteSymbols {
						base := GetAssetId(strings.ToUpper(baseSymbol))
						quote := GetAssetId(strings.ToUpper(quoteSymbol))
						go ant.Watching(subctx, base, quote)
					}
				}
				go ant.Trade(subctx)
				select {
				case <-sig:
					fmt.Println("cancel orders in 3 seconds.")
					cancel()
					time.Sleep(3 * time.Second)
					return nil
				}
			},
		},
	}

	sort.Sort(cli.FlagsByName(app.Flags))
	sort.Sort(cli.CommandsByName(app.Commands))

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
