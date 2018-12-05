package main

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/MixinNetwork/bot-api-go-client"
	"github.com/MixinNetwork/go-number"
	"github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"github.com/vmihailenco/msgpack"
)

const (
	ExinCore = "61103d28-3ac2-44a2-ae34-bd956070dab1"
)

type ExinOrderAction struct {
	A uuid.UUID // asset uuid
}

func (order *ExinOrderAction) Pack() string {
	pack, err := msgpack.Marshal(order)
	if err != nil {
		return ""
	}

	return base64.StdEncoding.EncodeToString(pack)
}

func (order *ExinOrderAction) Unpack(memo string) error {
	parsedpack, err := base64.StdEncoding.DecodeString(memo)
	if err != nil {
		return err
	}
	return msgpack.Unmarshal(parsedpack, order)
}

func ExinTrade(amount, send, get string, trace ...string) (string, error) {
	traceId := uuid.Must(uuid.NewV4()).String()
	if len(trace) == 1 {
		traceId = trace[0]
	}
	order := ExinOrderAction{
		A: uuid.Must(uuid.FromString(get)),
	}

	precision := ExinAssetPrecision(send)
	a := number.FromString(amount).Round(precision)

	log.Infof("trade in exin, %s, send %s, get %s", amount, Who(send), Who(get))
	transfer := bot.TransferInput{
		AssetId:     send,
		RecipientId: ExinCore,
		Amount:      a,
		TraceId:     traceId,
		Memo:        order.Pack(),
	}
	return traceId, bot.CreateTransfer(context.TODO(), &transfer, ClientId, SessionId, PrivateKey, PinCode, PinToken)
}

func ExinAssetPrecision(assetId string) int32 {
	switch assetId {
	case XIN:
		return 4
	case ETH:
		return 4
	case BTC:
		return 4
	case USDT:
		return 2
	case EOS:
		return 2
	default:
		log.Panicln("AssetPrecision", assetId)
	}
	return 0
}

func ExinTradeMessager(side string, amount float64, base, quote string) (string, error) {
	memo := fmt.Sprintf("ExinOne %s/%s %s", Who(base), Who(quote), side)
	trace := uuid.Must(uuid.NewV4()).String()
	var asset string
	if side == "buy" {
		asset = quote
	} else if side == "sell" {
		asset = base
	} else {
		panic("invlid type")
	}
	transfer := bot.TransferInput{
		AssetId:     asset,
		RecipientId: ExinCore,
		Amount:      number.FromFloat(amount),
		TraceId:     trace,
		Memo:        memo,
	}
	return trace, bot.CreateTransfer(context.TODO(), &transfer, ClientId, SessionId, PrivateKey, PinCode, PinToken)
}
