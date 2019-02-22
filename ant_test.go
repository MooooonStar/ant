package ant

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"testing"
	"time"

	prettyjson "github.com/hokaccha/go-prettyjson"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	uuid "github.com/satori/go.uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestGetExinDepth(t *testing.T) {
	ctx := context.Background()
	//data, _ := GetExinDepth(ctx, USDT, XIN)
	data, _ := FetchExinDepth(ctx, ETH, XIN)
	v, _ := prettyjson.Marshal(&data)
	fmt.Println("11111", string(v))
}

func TestExinOrder(t *testing.T) {
	order, err := GetExinOrder(context.TODO(), EOS, XIN)
	assert.Nil(t, err)
	fmt.Println("order+", order)
}

func TestOceanDepth(t *testing.T) {
	ctx := context.Background()
	data, _ := GetOceanDepth(ctx, BTC, USDT)
	v, _ := prettyjson.Marshal(&data)
	fmt.Println(string(v))
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
	match := "hKFBsAAAAAAAAAAAAAAAAAAAAAChQrAAAAAAAAAAAAAAAAAAAAAAoU+wI1JmpPASO+27lBlPgcN5ZaFTpkNBTkNFTA=="
	var reply OceanReply
	reply.Unpack(match)
	fmt.Println(reply.A, reply.B, reply.O, reply.S)

	order := OceanOrder{
		O: uuid.FromStringOrNil("235266a4-f012-3bed-bb94-194f81c37965"),
	}
	fmt.Println(order.Pack())

	// var r ExinReply
	// r.Unpack("g6FDzQPsoVShRqFPxBA+z+cTxh4+lJdmI5/wKyEu")
	// v, _ := prettyjson.Marshal(r)
	// fmt.Println("exin==", string(v))
	//fmt.Println(base64.StdEncoding.EncodeToString([]byte("235266a4-f012-3bed-bb94-194f81c37965")))
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

// a57cc436-0b27-4ddb-82b8-2d6eb02f95f8
func TestCreateOrder(t *testing.T) {
	//NewAnt().OceanCancel("a9b7813f-2d7f-30c5-8128-c1bf96b13b1e")
	trace, err := NewAnt().OceanTrade(PageSideAsk, "1.0", "0.0001", "L", XIN, BTC)
	if err != nil {
		panic(err)
	}
	log.Println(trace)
}

func TestRegister(t *testing.T) {
	key, err := Register(context.TODO())
	if err != nil {
		panic(err)
	}

	log.Println(key)
}
func TestQueryOrder(t *testing.T) {
	token, err := Token(ClientId, OceanKey)
	if err != nil {
		panic(err)
	}

	params := map[string]interface{}{
		"state": "DONE",
		"order": "DESC",
		"limit": 20,
		//"market": XIN + "-" + BTC,
		"offset": time.Now().UTC().Format(time.RFC3339Nano),
	}
	query := "?"
	for k, v := range params {
		query += fmt.Sprintf("%v=%v&", k, v)
	}

	req, err := http.NewRequest("GET", "https://events.ocean.one/orders"+query, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := httpClient.Do(req)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	bt, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	v, _ := prettyjson.Format(bt)
	log.Println(string(v))
}
