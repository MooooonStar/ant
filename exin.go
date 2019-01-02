package ant

import (
	"errors"

	"github.com/satori/go.uuid"
)

const (
	ExinCore = "61103d28-3ac2-44a2-ae34-bd956070dab1"
)

type ExinOrder struct {
	A uuid.UUID // asset uuid
}

func (order *ExinOrder) Pack() string {
	return ""
}

func (order *ExinOrder) Unpack(memo string) error {
	return errors.New("TODO")
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
	return ""
}

func (order *ExinReply) Unpack(memo string) error {
	return errors.New("TODO")
}

func ExinTrade(side, amount, base, quote string, trace ...string) (string, error) {
	return "", errors.New("TODO")
}
