// Package writer 将已通过 ingress 校验的点位事件原子写入历史与当前状态。
package writer

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/indu-forge/runtime-engine/internal/ingress"
	"github.com/indu-forge/runtime-engine/internal/store/postgres"
)

// ErrInvalidEvent 是 writer 对已验证消息进行业务边界复核时返回的稳定错误。
// 它绝不携带原始 value、subject 或外部错误文本。
var ErrInvalidEvent = errors.New("writer event invalid")

var (
	pointIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	eventIDPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// NewPostgresHandler 创建 writer 角色的纯数据库处理器。它不产生 outbox 消息；
// ProcessMessage 负责去重、fence、checkpoint 以及整个 BusinessTx 的提交/回滚。
func NewPostgresHandler() ingress.PostgresHandler {
	return HandlePostgres
}

// HandlePostgres 按固定顺序将一个事件写入历史和 current：历史记录永远保留，
// current 的乱序裁决由 Store 在同一事务中完成。
func HandlePostgres(ctx context.Context, tx *postgres.BusinessTx, message ingress.ValidatedMessage) error {
	write, err := normalize(message)
	if err != nil {
		return err
	}
	if err := tx.InsertPointHistory(ctx, write.history); err != nil {
		return err
	}
	_, err = tx.ApplyPointCurrent(ctx, write.current)
	return err
}

type pointWrite struct {
	history postgres.PointHistoryWrite
	current postgres.PointCurrentWrite
}

func normalize(message ingress.ValidatedMessage) (pointWrite, error) {
	event := message.Event
	if event.DeploymentID == "" || event.AccountID == "" || event.PointID == "" || event.EventID == "" || event.OwnerID == "" ||
		event.Epoch < 1 || event.Sequence < 0 || (event.Quality != "good" && event.Quality != "bad" && event.Quality != "unknown") || len(event.Value) == 0 {
		return pointWrite{}, ErrInvalidEvent
	}
	if !pointIDPattern.MatchString(event.PointID) || !eventIDPattern.MatchString(event.EventID) ||
		(event.SchemaVersion == "data.raw.v1" && event.Subject != "data.raw."+event.PointID) ||
		(event.SchemaVersion == "data.computed.v1" && event.Subject != "data.computed."+event.PointID) {
		return pointWrite{}, ErrInvalidEvent
	}
	source, err := parseUTC(event.SourceTimestamp)
	if err != nil {
		return pointWrite{}, ErrInvalidEvent
	}
	server, err := parseUTC(event.ServerTimestamp)
	if err != nil {
		return pointWrite{}, ErrInvalidEvent
	}
	received, err := parseUTC(event.ReceivedAt)
	if err != nil {
		return pointWrite{}, ErrInvalidEvent
	}
	// 不对 value 反序列化再编码，避免把数值精度、对象字段顺序等 JSON 语义变成 Go 值。
	value := append(json.RawMessage(nil), event.Value...)
	return pointWrite{
		history: postgres.PointHistoryWrite{
			DeploymentID: event.DeploymentID, PointID: event.PointID, EventID: event.EventID,
			OwnerID: event.OwnerID, Epoch: event.Epoch, Sequence: event.Sequence,
			SourceTimestamp: source, ServerTimestamp: server, ReceivedAt: received,
			Value: value, Quality: event.Quality,
		},
		current: postgres.PointCurrentWrite{
			DeploymentID: event.DeploymentID, PointID: event.PointID, EventID: event.EventID,
			OwnerID: event.OwnerID, Epoch: event.Epoch, Sequence: event.Sequence,
			SourceTimestamp: source, ServerTimestamp: server, Value: value, Quality: event.Quality,
		},
	}, nil
}

func parseUTC(value string) (time.Time, error) {
	// RFC3339 的 +00:00 不是本契约的 UTC wire 形式；只接受 Z，避免不同表示混入排序事实。
	if !strings.HasSuffix(value, "Z") {
		return time.Time{}, ErrInvalidEvent
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil || parsed.IsZero() || parsed.Location() != time.UTC {
		return time.Time{}, ErrInvalidEvent
	}
	return parsed.UTC(), nil
}
