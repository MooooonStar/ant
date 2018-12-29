package ant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	bot "github.com/MixinNetwork/bot-api-go-client"
	"github.com/hokaccha/go-prettyjson"
)

const (
	PollInterval                    = 100 * time.Millisecond
	CheckpointMixinNetworkSnapshots = "exchange-checkpoint-mixin-network-snapshots"
)

type Asset struct {
	AssetId string `json:"asset_id"               gorm:"type:varchar(36)"`
}

type Snapshot struct {
	SnapshotId string    `json:"snapshot_id"      gorm:"primary_key;type:varchar(36)"`
	Amount     string    `json:"amount"           gorm:"type:varchar(36)"`
	TraceId    string    `json:"trace_id"         gorm:"type:varchar(36)"`
	UserId     string    `json:"user_id"          gorm:"type:varchar(36)"`
	OpponentId string    `json:"opponent_id"      gorm:"type:varchar(36)"`
	Data       string    `json:"data"             gorm:"type:varchar(255)"`
	CreatedAt  time.Time `json:"created_at"       gorm:"type:timestamp"`
	Asset      `json:"asset"            gorm:"type:varchar(36)"`
}

func (Snapshot) TableName() string {
	return "snapshots"
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
	if len(s.OpponentId) == 0 || s.Asset.AssetId == CNB {
		return nil
	}

	v, _ := prettyjson.Marshal(s)
	log.Println("find snapshot", string(v))

	if err := ex.HandleSnapshot(ctx, s); err != nil {
		log.Println(err)
		return err
	}

	if err := Database(ctx).FirstOrCreate(s).Error; err != nil {
		return err
	}

	return nil
}
