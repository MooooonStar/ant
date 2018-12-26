package main

import (
	"encoding/base64"

	uuid "github.com/satori/go.uuid"
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
	return "", nil
}

//TODO
func OceanCancel(trace string) error {
	return nil
}
