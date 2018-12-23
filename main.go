package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"

	"github.com/hokaccha/go-prettyjson"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var baseSymbols = []string{"BTC", "EOS", "ETH"}
var quoteSymbols = []string{"BTC", "USDT"}

func main() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	app := cli.NewApp()
	app.Commands = []cli.Command{
		{
			Name:  "balance",
			Usage: "show balance",
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
			Name:  "run",
			Usage: "find profits between different exchanges",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "pair"},
				cli.BoolFlag{Name: "ocean"},
				cli.BoolFlag{Name: "exin"},
				cli.BoolFlag{Name: "debug"},
				cli.BoolFlag{Name: "file"},
			},
			Action: func(c *cli.Context) error {
				if debug := c.Bool("debug"); debug {
					log.SetLevel(log.DebugLevel)
				}
				if tofile := c.Bool("file"); tofile {
					file, err := os.OpenFile("/tmp/ant.log", os.O_WRONLY|os.O_APPEND, 0666)
					if err != nil {
						panic(err)
					}
					log.SetOutput(file)
				}

				pair := c.String("pair")
				ocean := c.Bool("ocean")
				exin := c.Bool("exin")
				symbols := strings.Split(pair, "/")
				var baseSymbol, quoteSymbol string
				if len(symbols) == 2 {
					baseSymbol, quoteSymbol = symbols[0], symbols[1]
				}

				if len(baseSymbol) > 0 && len(quoteSymbol) > 0 {
					baseSymbols = []string{baseSymbol}
					quoteSymbols = []string{quoteSymbol}
				}

				ctx, cancel := context.WithCancel(context.Background())
				ant := NewAnt(ocean, exin)
				go ant.PollMixinNetwork(ctx)
				go ant.PollMixinMessage(ctx)
				go ant.UpdateBalance(ctx)
				for _, baseSymbol := range baseSymbols {
					for _, quoteSymbol := range quoteSymbols {
						base := GetAssetId(strings.ToUpper(baseSymbol))
						quote := GetAssetId(strings.ToUpper(quoteSymbol))
						if base == quote {
							continue
						}

						client := NewClient(ctx, base, quote, ant.OnBlaseMessage(base, quote))
						go client.PollOceanMessage(ctx)

						go ant.Watching(ctx, base, quote)
						go ant.Fishing(ctx, base, quote)
					}
				}
				go ant.Trade(ctx)

				//ctrl-c 退出时先取消订单
				select {
				case <-sig:
					cancel()
					ant.Clean()
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
