package ant

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"sync"
	"time"

	bot "github.com/MixinNetwork/bot-api-go-client"
	number "github.com/MixinNetwork/go-number"
	uuid "github.com/satori/go.uuid"
	"github.com/shopspring/decimal"
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

const (
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

type OceanReply struct {
	S string    // source
	O uuid.UUID // cancelled order
	A uuid.UUID // matched ask order
	B uuid.UUID // matched bid order
}

func (reply *OceanReply) Pack() string {
	memo := make([]byte, 140)
	handle := new(codec.MsgpackHandle)
	encoder := codec.NewEncoderBytes(&memo, handle)
	if err := encoder.Encode(reply); err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(memo)
}

func (reply *OceanReply) Unpack(memo string) error {
	byt, err := base64.StdEncoding.DecodeString(memo)
	if err != nil {
		return err
	}

	handle := new(codec.MsgpackHandle)
	decoder := codec.NewDecoderBytes(byt, handle)
	return decoder.Decode(reply)
}

//TODO
func (ant *Ant) OceanTrade(side, price, amount, category, base, quote string, trace ...string) (string, error) {
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
	err := ant.CreateTransfer(context.TODO(), &bot.TransferInput{
		AssetId:     send,
		RecipientId: RandomBrokerId(),
		Amount:      number.FromString(amount).Round(AmountPrecision),
		TraceId:     traceId,
		Memo:        order.Pack(),
	}, ClientId, SessionId, PrivateKey, PinCode, PinToken)
	return traceId, err
}

func (ant *Ant) CreateTransfer(ctx context.Context, in *bot.TransferInput, uid, sid, sessionKey, pin, pinToken string) error {
	mutex := ant.mutexes.fetch(in.RecipientId, in.AssetId)
	mutex.Lock()
	defer mutex.Unlock()
	return bot.CreateTransfer(ctx, in, uid, sid, sessionKey, pin, pinToken)
}

//TODO
func (ant *Ant) OceanCancel(trace string) error {
	log.Printf("-----------Cancel : %v", trace)
	order := OceanOrder{
		O: uuid.Must(uuid.FromString(trace)),
	}
	cancelTrace := uuid.Must(uuid.NewV4()).String()
	return ant.CreateTransfer(context.TODO(), &bot.TransferInput{
		AssetId:     KU16,
		RecipientId: RandomBrokerId(),
		Amount:      number.FromFloat(0.00000001),
		TraceId:     cancelTrace,
		Memo:        order.Pack(),
	}, ClientId, SessionId, PrivateKey, PinCode, PinToken)
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

type tmap struct {
	sync.Map
}

func newTmap() *tmap {
	return &tmap{
		Map: sync.Map{},
	}
}

func (m *tmap) fetch(user, asset string) *sync.Mutex {
	uu, err := uuid.FromString(user)
	if err != nil {
		panic(user)
	}
	u := new(big.Int).SetBytes(uu.Bytes())
	au, err := uuid.FromString(asset)
	if err != nil {
		panic(asset)
	}
	a := new(big.Int).SetBytes(au.Bytes())
	s := new(big.Int).Add(u, a)
	key := new(big.Int).Mod(s, big.NewInt(100)).String()
	if _, found := m.Load(key); !found {
		m.Store(key, new(sync.Mutex))
	}
	val, _ := m.Load(key)
	return val.(*sync.Mutex)
}

func RandomBrokerId() string {
	rand.Seed(time.Now().UnixNano())
	return brokers[rand.Intn(len(brokers))]
}

var brokers = []string{
	"023a4a5a-d5e6-3182-b02e-70ce4b1c2bd5",
	"070f41c2-6941-31b8-9182-50e06291a5f7",
	"0d2b4bad-ef0a-3748-8ce8-67bb0aed660a",
	"0fdf3e21-428e-3fb2-a357-0f0a8886ec5c",
	"1297fb1a-011e-3907-8fa6-41f78972dbbe",
	"134a0b7e-4d0e-3347-9bad-6fa5f60d7aea",
	"137a8d5c-960f-3436-9c01-b725ccad58f4",
	"170a0b0e-d622-3c16-8448-aed9e2df5a49",
	"1e82e9ad-d7b3-3c5b-9172-d615fe17456c",
	"1f710784-e418-3d8e-a339-1a86ad8f3b0c",
	"1fa36c43-98c6-3869-83af-096e70f8ba32",
	"21b10070-0fdc-381c-be52-2d55e916b019",
	"2246f544-3020-3e74-83c9-4ed8096d7ce9",
	"25c523fd-474c-3ac2-9fdc-a53ba30310bb",
	"25d75ba6-ab01-390c-a125-314a0fe17c5c",
	"278b8e88-bd34-359a-a726-72c59b270508",
	"29123e17-0536-37ae-a5f6-8a07f547e29a",
	"2df2fd0a-15d2-378f-bb61-c86bf04b3fd4",
	"2e030fb0-39f6-35d2-ac43-7d7d4a35e906",
	"2e78fd86-cd25-304f-9272-aa3ad4b578e3",
	"3233335c-744b-38b2-a344-1f34a27fb91a",
	"366884a4-9bd3-3026-95b8-cdf80cded2d9",
	"375f696c-6762-3882-ab42-3be6c18d3d47",
	"3d8ce6fc-d92d-328a-ad4f-a827ce272aa5",
	"3f55f661-dae9-3350-bb14-67376ec1d02c",
	"402bda3e-be24-3959-bb91-04f2ea05eeba",
	"46d8e7a8-f08f-365a-bece-bff9426b13c2",
	"47886a65-c619-3040-87b4-75790a98d20b",
	"48f35238-3b5a-399d-abce-385ffda956b0",
	"4a167673-26cb-393a-a618-646602858d1f",
	"4ade331b-4ec8-397d-9b9b-e1df46e3c1e8",
	"4d584963-4c46-3731-9cdd-11b6145ae4e8",
	"533c97bc-1b5d-3095-b7f4-91cebc350b95",
	"55a73865-861e-33bc-ab14-7f5d9026e2b5",
	"578ffbc6-f590-364f-ac35-9774af15fc54",
	"58760cb5-f78c-3a3d-95cd-4c42ae7609da",
	"58de0b64-6971-30d3-971f-96bc2ae9f9ed",
	"5a46f2f2-0387-3543-adc8-dc04d96ed464",
	"605b24a3-31c9-3347-b234-3ee98b3bae9d",
	"63aa5788-4f01-34fd-b1ab-2ea8ddd97921",
	"64634ef2-4096-3a4f-b5dc-752a96ad4aa8",
	"67c0b792-70bb-38d2-bd48-1fd81ab1582e",
	"69db93aa-9176-360e-a3fe-7c35c9de0273",
	"7011b774-8988-3898-b89d-50ceb671b501",
	"721cb5b8-ad1e-31b5-89d1-dfd52739c7b9",
	"764e7369-51dd-3914-9cf0-3c740f6caf54",
	"783157cd-972b-36db-bd8c-53a78c63999f",
	"7ab66d97-8f68-375d-b679-fe73de7fa8d3",
	"7eb4a3fe-05a5-3721-8aea-6a0d1eb33c90",
	"7f6926f2-660f-3264-b24e-8c01efc07a12",
	"7fc5f0e6-8116-3a98-9472-f8a388c9901a",
	"811760a0-a036-3c9c-9f11-2229a343aa41",
	"81e843a1-d49a-3bbf-b1cc-426e42d51a52",
	"89c5b524-b7a8-39d3-ac72-de04bf4ac24d",
	"8aa264d6-e007-3a25-a8fe-e1cf8881f980",
	"8c92cc1b-7ef7-370f-bdef-8aed488a3140",
	"9162e9ef-efae-3628-8d49-f0985a9bbd71",
	"92765fbf-61b4-3d9b-9466-9046f4f181dd",
	"98d4098a-41f2-3113-a992-73b1769e5f00",
	"993a3a1b-04d7-38e3-a3e0-1fa979283f8f",
	"9c65b3a9-4d01-3be7-aa84-ab801a8423bd",
	"9d2951b2-8d08-3270-aa2c-b2581c6ef720",
	"a137bfbe-5889-3c53-ab91-0232fcd77629",
	"a5ad6a65-42cb-3e15-b0d6-4768463b392e",
	"a5dcc5d5-98f2-3100-a838-348981c94fbe",
	"a75386b4-0f80-32a7-8245-38a18345a95b",
	"a8364445-8162-3a88-a0de-f7d5574c092b",
	"a99d2eb4-09e8-310f-8a81-97b5eeac908d",
	"af73d6d3-0bdc-370c-b696-ccf50a32f4e7",
	"b3fcb4d0-4be1-374a-b8de-3816ad0e51af",
	"b631f734-09ac-3d49-8426-6719f36e1903",
	"b6a4c48f-5af5-30c4-8161-fa161e480c25",
	"b80d66fb-2662-3374-8a19-abe7ee9b9967",
	"ba2bc81a-c3f1-34c9-ae48-0e49ad235148",
	"bac6e17c-7ce0-38cd-83e9-2be3001ea842",
	"bb7827d1-5920-3f35-934c-8f80d4fdf7c0",
	"bc1d56e3-5456-3589-a05b-6362a9e5e128",
	"bcc705a2-f78d-33a7-a7de-37707cb14a0d",
	"c0485982-3fbc-3177-9244-5404578c5b39",
	"c3571b86-8679-3379-97e2-3a90a8f291c7",
	"ca484b80-0137-3bb6-a6a2-924dd6b82e90",
	"cb56578d-093e-3571-9d42-b0067b88cba2",
	"d26c8eaf-0108-3945-8ae2-9597fdb8d51d",
	"d4042c95-79f0-38f4-893a-fde1512a2737",
	"d97be2a0-a4c3-31c2-87c5-b03b85b41440",
	"deb386bf-deca-3858-b127-68734da6a3ac",
	"e14094ce-6cdf-3870-af22-be7fcab6148b",
	"e93e433e-4cbf-37db-b69e-12935bed35b2",
	"eca9c38f-4a79-38a1-b090-816d5a0cae04",
	"ecdcbdbd-c41a-35ee-9d9f-2ea9baec48fa",
	"ee8e4f2e-3cb9-3dfa-9cad-da3394352b06",
	"ef0ae2e3-d953-3e89-ae76-6da512e47363",
	"ef8c1327-b71d-3132-91ed-9f89552ce3b9",
	"f03bc7de-5cc7-34ea-9445-3cb98a19ba76",
	"f5bdc281-64d6-3b35-b6e3-719f0d1fa2ab",
	"f7917361-8ae4-3815-bced-8e7bf07d72a3",
	"f80db910-954f-321a-b4ac-81d4b938fd1c",
	"fd24ba77-d6af-363d-98a2-6308f9c34bd1",
	"fd6e11e7-c89d-3165-b29c-0b70e3d4c06e",
	"ffdcda5b-c0e6-3894-a411-a1dced74ba8d",
}
