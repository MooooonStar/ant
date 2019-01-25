package ant

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"

	bot "github.com/MixinNetwork/bot-api-go-client"
	number "github.com/MixinNetwork/go-number"
	uuid "github.com/satori/go.uuid"
	"github.com/vmihailenco/msgpack"
)

const (
	ExinCore = "61103d28-3ac2-44a2-ae34-bd956070dab1"
	//ExinCore = MasterID
)

type ExinOrder struct {
	A uuid.UUID // asset uuid
}

func (order *ExinOrder) Pack() string {
	pack, err := msgpack.Marshal(order)
	if err != nil {
		return ""
	}

	return base64.StdEncoding.EncodeToString(pack)
}

func (order *ExinOrder) Unpack(memo string) error {
	parsedpack, err := base64.StdEncoding.DecodeString(memo)
	if err != nil {
		return err
	}
	return msgpack.Unmarshal(parsedpack, order)
}

type ExinReply struct {
	C  int       // code
	P  string    // price, only type is return
	F  string    // ExinCore fee, only type is return
	FA string    // ExinCore fee asset, only type is return
	T  string    // type: refund(F)|return(R)|Error(E)
	O  uuid.UUID // order: trace_id
}

func (order *ExinReply) Pack() string {
	pack, err := msgpack.Marshal(order)
	if err != nil {
		return ""
	}

	return base64.StdEncoding.EncodeToString(pack)
}

func (order *ExinReply) Unpack(memo string) error {
	parsedpack, err := base64.StdEncoding.DecodeString(memo)
	if err != nil {
		return err
	}
	return msgpack.Unmarshal(parsedpack, order)
}

func ExinTrade(side, amount, base, quote string, trace ...string) (string, error) {
	traceId := uuid.Must(uuid.NewV4()).String()
	if len(trace) == 1 {
		traceId = trace[0]
	}
	send, get := base, quote
	if side == PageSideBid {
		send, get = quote, base
	}
	order := ExinOrder{
		A: uuid.Must(uuid.FromString(get)),
	}

	precision := ExinAssetPrecision(send, get)
	//这里不使用round，防止多卖
	a := number.FromString(amount).RoundFloor(precision)

	log.Printf("=============trade in exin, %s, send %s, get %s, trace: %s", a, Who(send), Who(get), traceId)
	transfer := bot.TransferInput{
		AssetId:     send,
		RecipientId: ExinCore,
		Amount:      a,
		TraceId:     traceId,
		Memo:        order.Pack(),
	}
	return traceId, bot.CreateTransfer(context.TODO(), &transfer, ClientId, SessionId, PrivateKey, PinCode, PinToken)
}

func ExinTradeMessager(side, amount, base, quote string, trace ...string) (string, error) {
	memo := fmt.Sprintf("ExinOne %s/%s %s", Who(base), Who(quote), side)
	traceId := uuid.Must(uuid.NewV4()).String()
	if len(trace) == 1 {
		traceId = trace[0]
	}
	send, get := base, quote
	if side == "buy" {
		send, get = quote, base
	}
	precision := ExinAssetPrecision(send, get)
	a := number.FromString(amount).Round(precision)

	transfer := bot.TransferInput{
		AssetId:     send,
		RecipientId: ExinCore,
		Amount:      a,
		TraceId:     traceId,
		Memo:        memo,
	}
	return traceId, bot.CreateTransfer(context.TODO(), &transfer, ClientId, SessionId, PrivateKey, PinCode, PinToken)
}

func ExinAssetPrecision(send, get string) int32 {
	if send == USDT {
		return 2
	}

	if get == USDT {
		return 4
	}

	if send == BTC {
		if get == XIN {
			return 4
		}
		return 6
	}

	if send == ETH || send == XIN {
		return 4
	}

	if send == EOS {
		return 2
	}

	return 0
}
