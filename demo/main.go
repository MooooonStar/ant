package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sort"
	"strings"
	"time"

	"github.com/MixinNetwork/bot-api-go-client"
	"github.com/MixinNetwork/go-number"
	"github.com/MooooonStar/ant"
	"github.com/go-redis/redis"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	uuid "github.com/satori/go.uuid"
	"github.com/urfave/cli"
)

var baseSymbols = []string{"BTC", "EOS", "ETH", "XIN"}
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
				assets, err := ant.ReadAssets(context.TODO())
				if err != nil {
					return err
				}
				balance := make(map[string]string, 0)
				for asset, amount := range assets {
					balance[ant.Who(asset)] = amount
				}
				log.Println(balance)
				return nil
			},
		},
		{
			Name:  "clear",
			Usage: "clear all assets",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "symbol,s"},
			},
			Action: func(c *cli.Context) error {
				symbol := strings.ToUpper(c.String("symbol"))
				assets, err := ant.ReadAssets(context.TODO())
				if err != nil {
					return err
				}
				if symbol == "ALL" {
					for asset, balance := range assets {
						if asset == ant.CNB {
							continue
						}
						in := bot.TransferInput{
							AssetId:     asset,
							RecipientId: "7b3f0a95-3ee9-4c1b-8ae9-170e3877d909",
							Amount:      number.FromString(balance),
							TraceId:     uuid.Must(uuid.NewV4()).String(),
							Memo:        "master, give you the money",
						}
						bot.CreateTransfer(context.Background(), &in, ant.ClientId, ant.SessionId, ant.PrivateKey, ant.PinCode, ant.PinToken)
					}
					return nil
				} else {
					asset := ant.GetAssetId(symbol)
					in := bot.TransferInput{
						AssetId:     asset,
						RecipientId: "7b3f0a95-3ee9-4c1b-8ae9-170e3877d909",
						Amount:      number.FromString(assets[asset]),
						TraceId:     uuid.Must(uuid.NewV4()).String(),
						Memo:        "master, give you the money",
					}
					return bot.CreateTransfer(context.Background(), &in, ant.ClientId, ant.SessionId, ant.PrivateKey, ant.PinCode, ant.PinToken)
				}
			},
		},
		{
			Name:  "run",
			Usage: "find profits between different exchanges",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "pair"},
				cli.BoolFlag{Name: "ocean"},
				cli.BoolFlag{Name: "exin"},
			},
			Action: func(c *cli.Context) error {
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

				db, err := gorm.Open("mysql", "root:@/test?parseTime=true")
				if err != nil {
					panic(err)
				}
				db.AutoMigrate(&ant.Snapshot{})
				db.AutoMigrate(&ant.ProfitEvent{})

				redisClient := redis.NewClient(&redis.Options{
					DB:           1,
					Addr:         "127.0.0.1:6379",
					ReadTimeout:  3 * time.Second,
					WriteTimeout: 3 * time.Second,
					PoolTimeout:  4 * time.Second,
					IdleTimeout:  60 * time.Second,
					PoolSize:     1024,
				})

				ctx, cancel := context.WithCancel(context.Background())
				ctx = ant.SetDB(ctx, db)
				ctx = ant.SetupRedis(ctx, redisClient)

				bot := ant.NewAnt(ocean, exin)
				go bot.PollMixinNetwork(ctx)
				go bot.PollMixinMessage(ctx)
				go bot.UpdateBalance(ctx)
				for _, baseSymbol := range baseSymbols {
					for _, quoteSymbol := range quoteSymbols {
						base := ant.GetAssetId(strings.ToUpper(baseSymbol))
						quote := ant.GetAssetId(strings.ToUpper(quoteSymbol))
						if base == quote {
							continue
						}

						client := ant.NewClient(ctx, base, quote, bot.OnOrderMessage(base, quote))
						go client.PollOceanMessage(ctx)

						go bot.Watching(ctx, base, quote)
						go bot.Fishing(ctx, base, quote)
					}
				}
				go bot.Trade(ctx)

				//ctrl-c 退出时先取消订单
				select {
				case <-sig:
					cancel()
					bot.Clean()
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
