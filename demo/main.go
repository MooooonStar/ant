package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	bot "github.com/MixinNetwork/bot-api-go-client"
	number "github.com/MixinNetwork/go-number"
	"github.com/MooooonStar/ant"
	"github.com/go-redis/redis"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	uuid "github.com/satori/go.uuid"
	"github.com/urfave/cli"
)

type Pair struct {
	Base  string
	Quote string
}

var watchingList = []Pair{
	Pair{ant.EOS, ant.USDT},
	Pair{ant.XIN, ant.USDT},
	Pair{ant.ETH, ant.USDT},
	Pair{ant.BTC, ant.USDT},
	Pair{ant.EOS, ant.BTC},
	Pair{ant.XIN, ant.BTC},
	Pair{ant.ETH, ant.BTC},
	Pair{ant.EOS, ant.XIN},
	Pair{ant.ETH, ant.XIN},
}

func main() {
	app := cli.NewApp()
	app.Commands = []cli.Command{
		{
			Name:  "balance",
			Usage: "show balance",
			Action: func(c *cli.Context) error {
				assets, _, err := ant.ReadAssets(context.TODO())
				if err != nil {
					return err
				}
				balance := make(map[string]string, 0)
				for symbol, amount := range assets {
					balance[symbol] = amount
				}
				log.Println(balance)
				return nil
			},
		},
		{
			Name:  "list",
			Usage: "list orders",
			Flags: []cli.Flag{cli.StringFlag{Name: "state"}},
			Action: func(c *cli.Context) error {
				orders, err := ant.ListOrders(strings.ToUpper(c.String("state")))
				log.Println(orders)
				return err
			},
		},
		{
			Name:  "cancel",
			Usage: "cancel orders",
			Action: func(c *cli.Context) error {
				orders, err := ant.ListOrders("PENDING")
				if err != nil {
					return err
				}
				log.Println(orders)
				return ant.NewAnt().CancelOrders(orders)
			},
		},
		{
			Name:  "clear",
			Usage: "clear all assets",
			Action: func(c *cli.Context) error {
				assets, _, err := ant.ReadAssets(context.TODO())
				if err != nil {
					return err
				}
				for symbol, balance := range assets {
					if symbol == "KU16" {
						continue
					}
					in := bot.TransferInput{
						AssetId:     ant.GetAssetId(symbol),
						RecipientId: ant.MasterID,
						Amount:      number.FromString(balance),
						TraceId:     uuid.Must(uuid.NewV4()).String(),
						Memo:        "long live the bitcoin",
					}
					err := bot.CreateTransfer(context.Background(), &in, ant.ClientId, ant.SessionId, ant.PrivateKey, ant.PinCode, ant.PinToken)
					if err != nil {
						log.Println("clear money error ", err)
					}
				}
				return nil
			},
		},
		{
			Name:  "run",
			Usage: "find profits between different exchanges",
			Flags: []cli.Flag{
				cli.BoolFlag{Name: "ocean"},
				cli.BoolFlag{Name: "exin"},
			},
			Action: func(c *cli.Context) error {
				conf := fmt.Sprintf("%s:%s@%s(%s)/%s?parseTime=True&charset=utf8mb4",
					DBUsername, DBPassword, "tcp", DBHost, DBName)
				db, err := gorm.Open("mysql", conf)
				if err != nil {
					panic(err)
				}
				db.AutoMigrate(&ant.Snapshot{})
				db.AutoMigrate(&ant.ProfitEvent{})

				redisClient := redis.NewClient(&redis.Options{
					DB:           RedisDB,
					Addr:         RedisAddress,
					ReadTimeout:  3 * time.Second,
					WriteTimeout: 3 * time.Second,
					PoolTimeout:  4 * time.Second,
					IdleTimeout:  60 * time.Second,
					PoolSize:     1024,
				})

				ctx := context.Background()
				ctx = ant.SetDB(ctx, db)
				ctx = ant.SetupRedis(ctx, redisClient)

				bot := ant.NewAnt()
				go bot.PollMixinNetwork(ctx)
				go bot.PollMixinMessage(ctx)
				go bot.UpdateBalance(ctx)

				for _, pair := range watchingList {
					base, quote := pair.Base, pair.Quote
					client := ant.NewClient(ctx, base, quote, bot.OnOrderMessage(base, quote))
					go client.PollOceanMessage(ctx)

					go bot.Watching(ctx, base, quote)
					go bot.Fishing(ctx, base, quote)
				}
				go bot.CleanUpTheMess(ctx)
				bot.Trade(ctx)
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
