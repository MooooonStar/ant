package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"

	"github.com/hokaccha/go-prettyjson"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var baseSymbols = []string{"EOS", "BTC", "XIN", "ETH"}
var quoteSymbols = []string{"USDT", "BTC"}

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
			Name:  "trade",
			Usage: "trade in exin",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "amount"},
				cli.StringFlag{Name: "send"},
				cli.StringFlag{Name: "get"},
			},
			Action: func(c *cli.Context) error {
				amount := c.String("amount")
				send := strings.ToUpper(c.String("send"))
				get := strings.ToUpper(c.String("get"))
				if len(amount) == 0 || len(send) == 0 || len(get) == 0 {
					return fmt.Errorf("invalid params")
				}
				_, err := ExinTrade(amount, GetAssetId(send), GetAssetId(get))
				return err
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
				if trace := c.String("trace"); len(trace) > 0 {
					return OceanCancel(trace)
				}
				snapshot := c.String("snapshot")
				trace, err := ReadSnapshot(context.TODO(), snapshot)
				if err != nil {
					return err
				}
				return OceanCancel(trace)
			},
		},
		{
			Name:  "run",
			Usage: "find profits between different exchanges",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "pair"},
				cli.BoolFlag{Name: "enable"},
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
				enable := c.Bool("enable")
				symbols := strings.Split(pair, "/")
				var baseSymbol, quoteSymbol string
				if len(symbols) == 2 {
					baseSymbol, quoteSymbol = symbols[0], symbols[1]
				}

				if len(baseSymbol) > 0 && len(quoteSymbol) > 0 {
					baseSymbols = []string{baseSymbol}
					quoteSymbols = []string{quoteSymbol}
				}

				ctx := context.Background()
				db, err := gorm.Open("mysql", "root:@/snow")
				if err != nil {
					panic(err)
				}
				ctx = SetDB(ctx, db)
				db.AutoMigrate(&Snapshot{})
				db.AutoMigrate(&Wallet{})
				SaveProperty(ctx, db)

				ant := NewAnt(enable)
				go ant.PollMixinNetwork(ctx)
				go ant.UpdateBalance(ctx)

				subctx, cancel := context.WithCancel(ctx)
				for _, baseSymbol := range baseSymbols {
					for _, quoteSymbol := range quoteSymbols {
						base := GetAssetId(strings.ToUpper(baseSymbol))
						quote := GetAssetId(strings.ToUpper(quoteSymbol))

						client := NewClient(subctx, base, quote, ant.OnMessage(base, quote))
						go client.Receive(subctx)

						go ant.Watching(subctx, base, quote)
						go ant.Fishing(subctx, base, quote)
					}
				}
				go ant.Trade(ctx)
				//ctrl-c 退出时先取消订单
				select {
				case <-sig:
					cancel()
					ant.Clean()
					SaveProperty(ctx, db)
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
