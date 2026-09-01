// Package event 生成与 RuntimeEngine point-event.schema.json 精确兼容的 raw envelope。
package event

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/indu-forge/collector-engine/internal/loader"
	"strings"
	"time"
)

type Envelope struct {
	SchemaVersion   string          `json:"schemaVersion"`
	Subject         string          `json:"subject"`
	EventID         string          `json:"eventId"`
	DeploymentID    string          `json:"deploymentId"`
	AccountID       string          `json:"accountId"`
	PointID         string          `json:"pointId"`
	OwnerID         string          `json:"ownerId"`
	Epoch           int64           `json:"epoch"`
	Sequence        int64           `json:"sequence"`
	Value           json.RawMessage `json:"value"`
	Quality         string          `json:"quality"`
	SourceTimestamp string          `json:"sourceTimestamp"`
	ServerTimestamp string          `json:"serverTimestamp"`
	ReceivedAt      string          `json:"receivedAt"`
	Source          Source          `json:"source"`
}
type Source struct {
	CollectorID  string `json:"collectorId"`
	ConnectionID string `json:"connectionId"`
	VariableID   string `json:"variableId"`
}

func New(binding loader.Binding, m loader.Mapping, sequence int64, value json.RawMessage, quality string, sourceAt, now time.Time) (Envelope, error) {
	if sequence < 0 || !json.Valid(value) || (quality != "good" && quality != "bad" && quality != "unknown") || sourceAt.IsZero() {
		return Envelope{}, errors.New("采集结果非法")
	}
	if strings.ContainsAny(m.DatapointID, "/ >*\\") {
		return Envelope{}, errors.New("pointId 不安全")
	}
	sourceAt = sourceAt.UTC()
	now = now.UTC()
	e := Envelope{SchemaVersion: "data.raw.v1", Subject: "data.raw." + m.DatapointID, DeploymentID: binding.DeploymentID, AccountID: binding.AccountID, PointID: m.DatapointID, OwnerID: binding.Ownership.OwnerID, Epoch: binding.Ownership.Epoch, Sequence: sequence, Value: append(json.RawMessage(nil), value...), Quality: quality, SourceTimestamp: sourceAt.Format(time.RFC3339Nano), ServerTimestamp: now.Format(time.RFC3339Nano), ReceivedAt: now.Format(time.RFC3339Nano), Source: Source{CollectorID: binding.CollectorID, ConnectionID: m.ConnectionID, VariableID: m.VariableID}}
	e.EventID = eventID(e)
	return e, nil
}
func eventID(e Envelope) string {
	fields := []string{e.SchemaVersion, e.DeploymentID, e.PointID, e.OwnerID, fmt.Sprint(e.Epoch), fmt.Sprint(e.Sequence)}
	sum := sha256.Sum256([]byte(strings.Join(fields, "\x1f")))
	return hex.EncodeToString(sum[:])
}
