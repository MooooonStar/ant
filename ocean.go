package ant

import (
	"errors"

	uuid "github.com/satori/go.uuid"
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
)

type OceanOrder struct {
	S string    // side
	A uuid.UUID // asset
	P string    // price
	T string    // type
	O uuid.UUID // order
}

func (action *OceanOrder) Pack() string {
	return ""
}

func (action *OceanOrder) Unpack(memo string) error {
	return errors.New("TODO")
}

type OceanReply struct {
	S string    // source
	O uuid.UUID // cancelled order
	A uuid.UUID // matched ask order
	B uuid.UUID // matched bid order
}

func (reply *OceanReply) Pack() string {
	return ""
}

func (reply *OceanReply) Unpack(memo string) error {
	return errors.New("TODO")
}

func OceanTrade(side, price, amount, category, base, quote string, trace ...string) (string, error) {
	return "", errors.New("TODO")
}

//TODO
func OceanCancel(trace string) error {
	return errors.New("TODO")
}
