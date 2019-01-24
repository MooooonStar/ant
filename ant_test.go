package ant

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	prettyjson "github.com/hokaccha/go-prettyjson"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestGetExinDepth(t *testing.T) {
	ctx := context.Background()
	data, _ := GetExinDepth(ctx, XIN, EOS)
	v, _ := prettyjson.Marshal(&data)
	fmt.Println("0-", string(v))
}

func TestExinOrder(t *testing.T) {
	order, err := GetExinOrder(context.TODO(), EOS, USDT)
	assert.Nil(t, err)
	fmt.Println("order", order)
}

func TestOceanTrade(t *testing.T) {
	order, err := OceanTrade(PageSideAsk, "0", "1.2", "M", EOS, BTC)
	assert.Nil(t, err)
	fmt.Println("order", order)
}

func TestOceanDepth(t *testing.T) {
	ctx := context.Background()
	data, _ := GetOceanDepth(ctx, BTC, USDT)
	v, _ := prettyjson.Marshal(&data)
	fmt.Println(string(v))
}

func TestExinTrade(t *testing.T) {
	//price, amount := 3936.6133, 0.0003
	trace, err := ExinTrade(PageSideAsk, "0.6", EOS, BTC)
	fmt.Println(trace, err)
}

//201602a3-a0d2-439e-8e63-e1b6dec86b76
func TestOceanCancel(t *testing.T) {
	//OceanCore = F1exCore
	err := OceanCancel("7eae417b-e534-361d-a861-a541d27a9680")
	fmt.Println(err, "")
}

func TestUUIDl(t *testing.T) {
	//OceanCore = F1exCore
	e := ProfitEvent{
		Category: "L",
		Base:     EOS,
		Quote:    BTC,
		Price:    decimal.NewFromFloat(0.00087878),
		Amount:   decimal.NewFromFloat(0.1629),
	}
	id := UuidWithString(Who(e.Base) + Who(e.Quote) + e.Price.String() + e.Amount.String() + "L")
	fmt.Println(id)

	id2 := UuidWithString(id + OceanCore)
	fmt.Println(id2)

	// e = Event{
	// 	Category: "L",
	// 	Base:     XIN,
	// 	Quote:    USDT,
	// 	Price:    decimal.NewFromFloat(550.0),
	// 	Amount:   decimal.NewFromFloat(0.011),
	// }
	// id = UuidWithString(Who(e.Base) + Who(e.Quote) + e.Price.String() + e.Amount.String() + "L")
	// fmt.Println(id)

	// id1 := UuidWithString(id + ExinCore)
	// fmt.Println(id1)

	// id2 := UuidWithString(id + OceanCore)
	// fmt.Println(id2)
}

func TestOrderMemo(t *testing.T) {
	match := "hKFBsO26uFtbBklypInhmBpIicyhQrC2wgFWRNg78apXjOjoP4ZuoU+wAAAAAAAAAAAAAAAAAAAAAKFTpU1BVENI"
	var reply OceanReply
	reply.Unpack(match)
	fmt.Println(reply.A, reply.B, reply.O, reply.S)

	var r ExinReply
	r.Unpack("g6FDzQPsoVShRqFPxBA+z+cTxh4+lJdmI5/wKyEu")
	v, _ := prettyjson.Marshal(r)
	fmt.Println("exin==", string(v))
}

func TestExinMemo(t *testing.T) {
	ExinTradeMessager("buy", "2", EOS, USDT)
}

func TestExinReply(t *testing.T) {
	var reply ExinReply
	reply.Unpack("hqFDzQPooVCmMTEzLjAzoUaoMC4wMDI2NDWiRkHEEIFbCxonZDc2j6pC1pT6YgqhVKFGoU/EEF8NwUzCCjhOhEgAMv3/veg=")
	v, _ := prettyjson.Marshal(reply)
	log.Println(string(v))
}

func TestReadAssets(t *testing.T) {
	data, _, _ := ReadAssets(context.TODO())
	v, _ := prettyjson.Marshal(data)
	fmt.Println(string(v))
}

// func TestSumAssets(t *testing.T) {
// 	prices, err := GetExinPrices(context.Background(), BTC)
// 	if err != nil {
// 		panic(err)
// 	}

// 	log.Println("prices", prices)

// 	sum := 0.0
// 	for asset, amount := range Wallet {
// 		price, _ := strconv.ParseFloat(prices[asset], 64)
// 		sum += price * amount
// 	}

// 	log.Println("sum", sum)
// }

func TestReply(t *testing.T) {
	log.Println(Reply("不好笑"))
	log.Println(time.Now())
}
