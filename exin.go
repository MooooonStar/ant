package main

import (
	"context"
	"encoding/base64"

	"github.com/MixinNetwork/bot-api-go-client"
	"github.com/MixinNetwork/go-number"
	"github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"github.com/vmihailenco/msgpack"
)

const (
	ExinCore = "61103d28-3ac2-44a2-ae34-bd956070dab1"
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

func ExinTrade(amount, send, get string, trace ...string) (string, error) {
	traceId := uuid.Must(uuid.NewV4()).String()
	if len(trace) == 1 {
		traceId = trace[0]
	}
	order := ExinOrder{
		A: uuid.Must(uuid.FromString(get)),
	}

	precision := ExinAssetPrecision(send, get)
	a := number.FromString(amount).Round(precision)

	log.Infof("trade in exin, %s, send %s, get %s", a, Who(send), Who(get))
	transfer := bot.TransferInput{
		AssetId:     send,
		RecipientId: ExinCore,
		Amount:      a,
		TraceId:     traceId,
		Memo:        order.Pack(),
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
