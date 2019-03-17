package ant

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/md5"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	bot "github.com/MixinNetwork/bot-api-go-client"
	number "github.com/MixinNetwork/go-number"
	jwt "github.com/dgrijalva/jwt-go"
	uuid "github.com/satori/go.uuid"
	"github.com/ugorji/go/codec"
)

func Register(ctx context.Context) (string, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", err
	}
	oceanKey, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return "", err
	}

	pub, err := x509.MarshalPKIXPublicKey(priv.Public())
	if err != nil {
		return "", err
	}
	sig := make([]byte, 140)
	handle := new(codec.MsgpackHandle)
	encoder := codec.NewEncoderBytes(&sig, handle)
	action := map[string][]byte{"U": pub}
	err = encoder.Encode(action)
	if err != nil {
		return "", err
	}

	input := &bot.TransferInput{
		AssetId:     KU16,
		RecipientId: RandomBrokerId(),
		Amount:      number.FromString("0.00000001"),
		TraceId:     getSettlementId(ClientId, "USER|SIG|REGISTER"),
		Memo:        base64.StdEncoding.EncodeToString(sig),
	}
	err = bot.CreateTransfer(ctx, input, ClientId, SessionId, PrivateKey, PinCode, PinToken)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(oceanKey), nil
}

func getSettlementId(id, modifier string) string {
	h := md5.New()
	io.WriteString(h, id)
	io.WriteString(h, modifier)
	sum := h.Sum(nil)
	sum[6] = (sum[6] & 0x0f) | 0x30
	sum[8] = (sum[8] & 0x3f) | 0x80
	return uuid.FromBytesOrNil(sum).String()
}

func OceanToken(userID, key string) (string, error) {
	oceanKey, err := hex.DecodeString(key)
	if err != nil {
		return "", err
	}
	privateKey, err := x509.ParseECPrivateKey(oceanKey)
	if err != nil {
		return "", err
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"uid": userID,
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func ListOrders(state string) ([]string, error) {
	token, err := OceanToken(ClientId, OceanKey)
	if err != nil {
		return nil, err
	}

	params := map[string]interface{}{
		"state":  state,
		"order":  "DESC",
		"limit":  500,
		"offset": time.Now().UTC().Format(time.RFC3339Nano),
	}
	query := "?"
	for k, v := range params {
		query += fmt.Sprintf("%v=%v&", k, v)
	}

	req, err := http.NewRequest("GET", "https://events.ocean.one/orders"+query, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bt, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var Resp struct {
		Data []struct {
			OrderID string `json:"order_id"`
		} `json:"data"`
	}

	if err := json.Unmarshal(bt, &Resp); err != nil {
		return nil, err
	}

	orders := make([]string, 0)
	for _, data := range Resp.Data {
		orders = append(orders, data.OrderID)
	}
	return orders, nil
}

func (ant *Ant) CancelOrders(orders []string) error {
	for _, order := range orders {
		if err := ant.OceanCancel(order); err != nil {
			return err
		}
	}
	return nil
}
