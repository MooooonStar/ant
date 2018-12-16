package main

import (
	"context"
	"encoding/base64"
	"fmt"

	bot "github.com/MixinNetwork/bot-api-go-client"
	"github.com/MixinNetwork/go-number"
	uuid "github.com/satori/go.uuid"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"
)

const (
	OrderSideAsk    = "A"
	OrderSideBid    = "B"
	OrderTypeLimit  = "L"
	OrderTypeMarket = "M"

	PricePrecision  = 8
	AmountPrecision = 4
	MaxPrice        = 1000000000
	MaxAmount       = 5000000000
	MaxFunds        = MaxPrice * MaxAmount
)

var (
	OceanCore = "aaff5bef-42fb-4c9f-90e0-29f69176b7d4"
	F1exCore  = "32cc0fda-5deb-448a-be70-a81dac4a3eed"
)

type OceanOrder struct {
	S string    // side
	A uuid.UUID // asset
	P string    // price
	T string    // type
	O uuid.UUID // order
}

func (action *OceanOrder) Pack() string {
	order := make(map[string]interface{}, 0)
	if action.O != uuid.Nil {
		order["O"] = action.O
	} else {
		order["S"] = action.S
		order["P"] = action.P
		order["T"] = action.T
		order["A"] = action.A
	}
	memo := make([]byte, 140)
	handle := new(codec.MsgpackHandle)
	encoder := codec.NewEncoderBytes(&memo, handle)
	if err := encoder.Encode(order); err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(memo)
}

func (action *OceanOrder) Unpack(memo string) error {
	byt, err := base64.StdEncoding.DecodeString(memo)
	if err != nil {
		return err
	}

	handle := new(codec.MsgpackHandle)
	decoder := codec.NewDecoderBytes(byt, handle)
	return decoder.Decode(action)
}

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

func QuotePrecision(assetId string) uint8 {
	switch assetId {
	case XIN:
		return 8
	case BTC:
		return 8
	case USDT:
		return 4
	default:
		log.Panicln("QuotePrecision", assetId)
	}
	return 0
}

func QuoteMinimum(assetId string) number.Decimal {
	switch assetId {
	case XIN:
		return number.FromString("0.0001")
	case BTC:
		return number.FromString("0.0001")
	case USDT:
		return number.FromString("1")
	default:
		log.Panicln("QuoteMinimum", assetId)
	}
	return number.Zero()
}

func OrderCheck(action OceanOrder, desireAmount, quote string) error {
	if action.T != OrderTypeLimit && action.T != OrderTypeMarket {
		return fmt.Errorf("the price type should be ether limit or market")
	}

	if (quote != XIN) && (quote != USDT) && (quote != BTC) {
		return fmt.Errorf("the quote should be XIN, USDT or BTC")
	}

	priceDecimal := number.FromString(action.P)
	maxPrice := number.NewDecimal(MaxPrice, int32(QuotePrecision(quote)))
	if priceDecimal.Cmp(maxPrice) > 0 {
		return fmt.Errorf("the price should less than %s", maxPrice)
	}
	price := priceDecimal.Integer(QuotePrecision(quote))
	if action.T == OrderTypeLimit {
		if price.IsZero() {
			return fmt.Errorf("the price can`t be zero in limit price")
		}
	} else if !price.IsZero() {
		return fmt.Errorf("the price should be zero in market price")
	}

	fundsPrecision := AmountPrecision + QuotePrecision(quote)
	funds := number.NewInteger(0, fundsPrecision)
	amount := number.NewInteger(0, AmountPrecision)

	assetDecimal := number.FromString(desireAmount)
	if action.S == OrderSideBid {
		maxFunds := number.NewDecimal(MaxFunds, int32(fundsPrecision))
		if assetDecimal.Cmp(maxFunds) > 0 {
			return fmt.Errorf("the funds should be less than %v", maxFunds)
		}
		funds = assetDecimal.Integer(fundsPrecision)
		if funds.Decimal().Cmp(QuoteMinimum(quote)) < 0 {
			return fmt.Errorf("the funds should be greater than %v", funds.Persist())
		}
	} else {
		maxAmount := number.NewDecimal(MaxAmount, AmountPrecision)
		if assetDecimal.Cmp(maxAmount) > 0 {
			return fmt.Errorf("the amount should be less than %v", maxAmount)
		}
		amount = assetDecimal.Integer(AmountPrecision)
		if action.T == OrderTypeLimit && price.Mul(amount).Decimal().Cmp(QuoteMinimum(quote)) < 0 {
			return fmt.Errorf("the amount should be greater than %v %s", QuoteMinimum(quote), quote)
		}
	}
	return nil
}

func OceanTrade(side, price, amount, category, base, quote string, trace ...string) (string, error) {
	log.Infof("++++++%s %s at price %12.8s, amount %12.8s, type: %s ", side, Who(base), price, amount, category)
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
	log.Infof("-----order check passed, trace ----%s", traceId)

	err := bot.CreateTransfer(context.TODO(), &bot.TransferInput{
		AssetId:     send,
		RecipientId: OceanCore,
		Amount:      number.FromString(amount).Round(AmountPrecision),
		TraceId:     traceId,
		Memo:        order.Pack(),
	}, ClientId, SessionId, PrivateKey, PinCode, PinToken)
	return traceId, err
}

func OceanBuy(price, amount, category, base, quote string, trace ...string) (string, error) {
	log.Infof("++++++Buy %s at price %12.8s, amount %12.8s, type: %s ", Who(base), price, amount, category)
	order := OceanOrder{
		S: "B",
		A: uuid.Must(uuid.FromString(base)),
		P: number.FromString(price).Round(PricePrecision).String(),
		T: category,
	}

	if err := OrderCheck(order, fmt.Sprint(amount), quote); err != nil {
		return "", err
	}

	traceId := uuid.Must(uuid.NewV4()).String()
	if len(trace) == 1 {
		traceId = trace[0]
	}
	log.Infof("-----buy trace ----%s", traceId)

	err := bot.CreateTransfer(context.TODO(), &bot.TransferInput{
		AssetId:     quote,
		RecipientId: OceanCore,
		Amount:      number.FromString(amount).Round(AmountPrecision),
		TraceId:     traceId,
		Memo:        order.Pack(),
	}, ClientId, SessionId, PrivateKey, PinCode, PinToken)
	return traceId, err
}

func OceanSell(price, amount, category, base, quote string, trace ...string) (string, error) {
	log.Infof("-----Sell %s at price %12.8s, amount %12.8s, type: %s", Who(base), price, amount, category)
	order := OceanOrder{
		S: "A",
		A: uuid.Must(uuid.FromString(quote)),
		P: number.FromString(price).Round(PricePrecision).String(),
		T: category,
	}

	if err := OrderCheck(order, fmt.Sprint(amount), quote); err != nil {
		return "", err
	}

	traceId := uuid.Must(uuid.NewV4()).String()
	if len(trace) == 1 {
		traceId = trace[0]
	}

	log.Infof("-----Sell trace ----%s", traceId)
	err := bot.CreateTransfer(context.TODO(), &bot.TransferInput{
		AssetId:     base,
		RecipientId: OceanCore,
		Amount:      number.FromString(amount).Round(AmountPrecision),
		TraceId:     traceId,
		Memo:        order.Pack(),
	}, ClientId, SessionId, PrivateKey, PinCode, PinToken)
	return traceId, err
}

func OceanCancel(trace string) error {
	log.Infof("*****Cancel : %v", trace)
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
