package main

import (
	"context"
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
			Usage: "complete a task on the list",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "snapshot,s"},
			},
			Action: func(c *cli.Context) error {
				snapshot := c.String("snapshot")
				trace, err := ReadSnapshot(context.TODO(), snapshot)
				if err != nil {
					return err
				}
				return OceanCancel(trace)
			},
		},
	}

	app.Commands = []cli.Command{
		{
			Name:  "run",
			Usage: "complete a task on the list",
			// Flags: []cli.Flag{
			// 	cli.StringFlag{Name: "snapshot,s"},
			// },
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
