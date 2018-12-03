package main

import (
	"context"
	"fmt"
	"os"
	"sort"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()

	app.Commands = []cli.Command{
		{
			Name:  "cancel",
			Usage: "cancel order in ocean.one by snapshot or trace",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "snapshot,s"},
				cli.StringFlag{Name: "trace,t"},
			},
			Action: func(c *cli.Context) error {
				log.Println("cancel order info:")
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
			Name:  "mining",
			Usage: "find profits between different exchanges",
			Action: func(c *cli.Context) error {
				log.SetLevel(log.DebugLevel)
				ant := NewAnt()
				ctx := context.Background()
				go ant.PollMixinNetwork(ctx)

				for _, baseSymbol := range []string{"BTC", "EOS", "XIN", "ETH"} {
					for _, quoteSymbol := range []string{"USDT", "BTC", "ETH"} {
						base := GetAssetId(baseSymbol)
						quote := GetAssetId(quoteSymbol)
						go ant.Watching(ctx, base, quote)
					}
				}
				ant.Trade()
				return nil
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
