package ant

import (
	bot "MoooonStar/bot-api-go-client"
	number "MoooonStar/go-number"
	"context"
	"encoding/base64"
	"fmt"
	"log"

	uuid "github.com/satori/go.uuid"
	"github.com/shopspring/decimal"
	"github.com/ugorji/go/codec"
)

var (
	OceanCore = "aaff5bef-42fb-4c9f-90e0-29f69176b7d4"
)

const (
	OrderTypeLimit = "L"
)

type OceanTransfer struct {
	S string    // source
	O uuid.UUID // cancelled order
	A uuid.UUID // matched ask order
	B uuid.UUID // matched bid order
}

func (action *OceanTransfer) Pack() string {
	memo := make([]byte, 140)
	handle := new(codec.MsgpackHandle)
	encoder := codec.NewEncoderBytes(&memo, handle)
	if err := encoder.Encode(action); err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(memo)
}

func (action *OceanTransfer) Unpack(memo string) error {
	byt, err := base64.StdEncoding.DecodeString(memo)
	if err != nil {
		return err
	}

	handle := new(codec.MsgpackHandle)
	decoder := codec.NewDecoderBytes(byt, handle)
	return decoder.Decode(action)
}

//TODO
func OceanTrade(side, price, amount, category, base, quote string, trace ...string) (string, error) {
	send, get, s := base, quote, "A"
	if side == PageSideBid {
		send, get, s = quote, base, "B"
	}
	p, _ := decimal.NewFromString(price)

	order := OceanOrder{
		S: s,
		A: uuid.Must(uuid.FromString(get)),
		P: p.Round(PricePrecision).String(),
		T: category,
	}

	if err := OrderCheck(order, fmt.Sprint(amount), quote); err != nil {
		return "", err
	}

	traceId := uuid.Must(uuid.NewV4()).String()
	if len(trace) == 1 {
		traceId = trace[0]
	}
	log.Printf("++++++%s %s at price %12.8s, amount %12.8s, type: %s, trace: %s ", side, Who(base), price, amount, category, traceId)
	err := bot.CreateTransfer(context.TODO(), &bot.TransferInput{
		AssetId:     send,
		RecipientId: OceanCore,
		Amount:      number.FromString(amount).Round(AmountPrecision),
		TraceId:     traceId,
		Memo:        order.Pack(),
	}, ClientId, SessionId, PrivateKey, PinCode, PinToken)
	return traceId, err
}

//TODO
func OceanCancel(trace string) error {
	log.Printf("-----------Cancel : %v", trace)
	order := OceanOrder{
		O: uuid.Must(uuid.FromString(trace)),
	}
	cancelTrace := uuid.Must(uuid.NewV4()).String()
	return bot.CreateTransfer(context.TODO(), &bot.TransferInput{
		AssetId:     CNB,
		RecipientId: OceanCore,
		Amount:      number.FromFloat(0.001010),
		TraceId:     cancelTrace,
		Memo:        order.Pack(),
	}, ClientId, SessionId, PrivateKey, PinCode, PinToken)
}
