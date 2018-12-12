package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	bot "github.com/MixinNetwork/bot-api-go-client"
	"github.com/hokaccha/go-prettyjson"
	uuid "github.com/satori/go.uuid"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"
)

const (
	PollInterval                    = 100 * time.Millisecond
	CheckpointMixinNetworkSnapshots = "exchange-checkpoint-mixin-network-snapshots"
)

type TransferAction struct {
	S string    // source
	O uuid.UUID // cancelled order
	A uuid.UUID // matched ask order
	B uuid.UUID // matched bid order
}

func (action *TransferAction) Pack() string {
	memo := make([]byte, 140)
	handle := new(codec.MsgpackHandle)
	encoder := codec.NewEncoderBytes(&memo, handle)
	if err := encoder.Encode(action); err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(memo)
}

func (action *TransferAction) Unpack(memo string) error {
	byt, err := base64.StdEncoding.DecodeString(memo)
	if err != nil {
		return err
	}

	handle := new(codec.MsgpackHandle)
	decoder := codec.NewDecoderBytes(byt, handle)
	return decoder.Decode(action)
}

type Asset struct {
	AssetId string `json:"asset_id"               gorm:"type:varchar(36)"`
}

type Snapshot struct {
	SnapshotId string `json:"snapshot_id"      gorm:"primary_key;type:varchar(36)"`
	Amount     string `json:"amount"           gorm:"type:varchar(10)"`
	TraceId    string `json:"trace_id"         gorm:"type:varchar(36)"`
	UserId     string `json:"user_id"          gorm:"type:varchar(36)"`
	OpponentId string `json:"opponent_id"      gorm:"type:varchar(36)"`
	Data       string `json:"data"             gorm:"type:varchar(255)"`
	Asset      `json:"asset"            gorm:"type:varchar(36)"`
	CreatedAt  time.Time `json:"created_at"`
}

func (ex *Ant) requestMixinNetwork(ctx context.Context, checkpoint time.Time, limit int) ([]*Snapshot, error) {
	uri := fmt.Sprintf("/network/snapshots?offset=%s&order=ASC&limit=%d", checkpoint.Format(time.RFC3339Nano), limit)
	token, err := bot.SignAuthenticationToken(ClientId, SessionId, PrivateKey, "GET", uri, "")
	if err != nil {
		return nil, err
	}
	body, err := bot.Request(ctx, "GET", uri, nil, token)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data  []*Snapshot `json:"data"`
		Error string      `json:"error"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Error != "" {
		return nil, errors.New(resp.Error)
	}
	return resp.Data, nil
}

func (ex *Ant) PollMixinNetwork(ctx context.Context) {
	const limit = 500
	checkpoint := time.Now().UTC()
	for {
		snapshots, err := ex.requestMixinNetwork(ctx, checkpoint, limit)
		if err != nil {
			log.Println("PollMixinNetwork ERROR", err)
			time.Sleep(PollInterval)
			continue
		}
		for _, s := range snapshots {
			if ex.snapshots[s.SnapshotId] {
				continue
			}
			ex.ensureProcessSnapshot(ctx, s)
			checkpoint = s.CreatedAt
			ex.snapshots[s.SnapshotId] = true
		}
		if len(snapshots) < limit {
			time.Sleep(PollInterval)
		}
	}
}

func (ex *Ant) ensureProcessSnapshot(ctx context.Context, s *Snapshot) {
	for {
		err := ex.processSnapshot(ctx, s)
		if err == nil {
			break
		}
		log.Println("ensureProcessSnapshot", err)
		time.Sleep(100 * time.Millisecond)
	}
}

func (ex *Ant) processSnapshot(ctx context.Context, s *Snapshot) error {
	if len(s.OpponentId) == 0 || len(s.Data) == 0 {
		return nil
	}

	if s.Asset.AssetId == CNB {
		return nil
	}

	v, _ := prettyjson.Marshal(s)
	log.Info("find snapshot:", string(v))

	if err := Database(ctx).FirstOrCreate(s).Error; err != nil {
		return err
	}

	var order TransferAction
	if err := order.Unpack(s.Data); err != nil {
		return nil
	}

	if order.S != "MATCH" {
		return nil
	}

	amount, _ := decimal.NewFromString(s.Amount)
	ex.orderLock.Lock()
	defer ex.orderLock.Unlock()
	//一个订单可能对应多笔成交，只正常处理第一笔
	if bidFinished, bidOK := ex.orders[order.B.String()]; bidOK {
		if !bidFinished {
			log.Info("order matched,", order)
			ex.matchedAmount <- amount
		} else {
			ExinTrade(s.Amount, s.AssetId, USDT)
		}
	} else if askFinished, askOK := ex.orders[order.A.String()]; askOK {
		if !askFinished {
			log.Info("order matched,", order)
			ex.matchedAmount <- amount
		} else {
			ExinTrade(s.Amount, s.AssetId, USDT)
		}
	}

	return nil
}
