// Package alarm 提供 RuntimeEngine V1 的纯报警领域状态机。它不访问网络或数据库；
// 调用者必须把 State 作为 owner/epoch CAS 的一部分持久化后才发布 Event。
package alarm

import (
	"encoding/json"
	"time"

	"github.com/indu-forge/runtime-engine/internal/model"
)

// Identity 是由 lifecycle 已取得 alarm role fence 后传入的发布身份。
type Identity struct {
	DeploymentID string
	AccountID    string
	OwnerID      string
	Epoch        int64
}

// PointInput 是经过 ingress schema/fence 校验后的原始或计算点位事实。
// 所有时间均保存为 UTC，EventID 用于同一输入的幂等与顺序决策。
type PointInput struct {
	SchemaVersion   string
	EventID         string
	PointID         string
	Epoch           int64
	Sequence        int64
	Value           json.RawMessage
	Quality         string
	Offline         bool
	SourceTimestamp time.Time
	ServerTimestamp time.Time
	ReceivedAt      time.Time
}

// Event 是 alarm.event.v1 的强类型输出。ACK 刻意没有生成路径，确认由 Runtime API 负责。
type Event struct {
	SchemaVersion   string      `json:"schemaVersion"`
	Subject         string      `json:"subject"`
	EventID         string      `json:"eventId"`
	DeploymentID    string      `json:"deploymentId"`
	AccountID       string      `json:"accountId"`
	Kind            string      `json:"kind"`
	Operation       string      `json:"operation"`
	AlarmID         string      `json:"alarmId"`
	AlarmItemID     string      `json:"alarmItemId"`
	PointIDs        []string    `json:"pointIds"`
	OwnerID         string      `json:"ownerId"`
	Epoch           int64       `json:"epoch"`
	SourceTimestamp string      `json:"sourceTimestamp"`
	ServerTimestamp string      `json:"serverTimestamp"`
	ReceivedAt      string      `json:"receivedAt"`
	Transition      *Transition `json:"transition"`
}

type Transition struct {
	AlarmState         string          `json:"alarmState"`
	Severity           string          `json:"severity"`
	PreviousSeverity   *string         `json:"previousSeverity"`
	ConditionID        string          `json:"conditionId"`
	Value              json.RawMessage `json:"value"`
	Quality            string          `json:"quality"`
	Message            string          `json:"message"`
	OpenedAt           string          `json:"openedAt"`
	UpdatedAt          string          `json:"updatedAt"`
	ClearedAt          *string         `json:"clearedAt"`
	AckedAt            *string         `json:"ackedAt"`
	AcknowledgedBy     *string         `json:"acknowledgedBy"`
	StateVersion       int64           `json:"stateVersion"`
	TransitionSequence int64           `json:"transitionSequence"`
}

// SampleState、ItemState 全为稳定 JSON 形状，供后续存储层 round-trip。
type SampleState struct {
	EventID         string          `json:"eventId"`
	Epoch           int64           `json:"epoch"`
	Sequence        int64           `json:"sequence"`
	Value           json.RawMessage `json:"value"`
	Quality         string          `json:"quality"`
	Offline         bool            `json:"offline"`
	SourceTimestamp time.Time       `json:"sourceTimestamp"`
	ServerTimestamp time.Time       `json:"serverTimestamp"`
	ReceivedAt      time.Time       `json:"receivedAt"`
}

type RateSample struct {
	At time.Time `json:"at"`
	// Value 保存有理数字符串，避免 JSON float64 抹掉 2^53 以上的相邻整数。
	Value string `json:"value"`
}

type ItemState struct {
	Inputs               map[string]SampleState `json:"inputs"`
	ActiveConditionID    string                 `json:"activeConditionId"`
	CandidateConditionID string                 `json:"candidateConditionId"`
	CandidateSince       *time.Time             `json:"candidateSince"`
	ClearSince           *time.Time             `json:"clearSince"`
	AlarmID              string                 `json:"alarmId"`
	OpenedAt             *time.Time             `json:"openedAt"`
	StateVersion         int64                  `json:"stateVersion"`
	TransitionSequence   int64                  `json:"transitionSequence"`
	LastValue            json.RawMessage        `json:"lastValue"`
	LastQuality          string                 `json:"lastQuality"`
	LastSourceAt         *time.Time             `json:"lastSourceAt"`
	LastEvaluatedAt      *time.Time             `json:"lastEvaluatedAt"`
	RateHistory          []RateSample           `json:"rateHistory"`
}

type State struct {
	Items map[string]ItemState `json:"items"`
}

type Runtime struct {
	identity Identity
	items    map[string]model.AlarmItem
	itemIDs  []string
	state    State
}
