package runtimeview

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrNotFound             = errors.New("runtime value not found")
	ErrAlarmNotActive       = errors.New("alarm has no active instance")
	ErrAlarmVersionConflict = errors.New("alarm state version conflict")
)

type PointCurrent struct {
	PointID         string          `json:"pointId"`
	Value           json.RawMessage `json:"value"`
	Quality         string          `json:"quality"`
	SourceTimestamp time.Time       `json:"sourceTimestamp"`
	ServerTimestamp time.Time       `json:"serverTimestamp"`
	Sequence        int64           `json:"sequence"`
	EventID         string          `json:"eventId"`
	UpdatedAt       time.Time       `json:"updatedAt"`
}

type PointSample struct {
	PointID         string          `json:"pointId"`
	Value           json.RawMessage `json:"value"`
	Quality         string          `json:"quality"`
	SourceTimestamp time.Time       `json:"sourceTimestamp"`
	ServerTimestamp time.Time       `json:"serverTimestamp"`
	Sequence        int64           `json:"sequence"`
	EventID         string          `json:"eventId"`
	ReceivedAt      time.Time       `json:"receivedAt"`
}

type AlarmState struct {
	AlarmItemID string          `json:"alarmItemId"`
	Revision    int64           `json:"revision"`
	State       json.RawMessage `json:"state"`
	Version     int64           `json:"version"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}

type AlarmAcknowledgement struct {
	Version        int64     `json:"version"`
	AcknowledgedAt time.Time `json:"acknowledgedAt"`
}

type HistoryQuery struct {
	From  *time.Time
	To    *time.Time
	Limit int
}

type Store interface {
	Ping(context.Context) error
	Current(context.Context, string, string) (PointCurrent, error)
	History(context.Context, string, string, HistoryQuery) ([]PointSample, error)
	AlarmStates(context.Context, string, int) ([]AlarmState, error)
	AcknowledgeAlarm(context.Context, string, string, int64, string, string, time.Time) (AlarmAcknowledgement, error)
}

type ComputeCommand struct {
	CommandID      string          `json:"commandId"`
	ComputeID      string          `json:"computeId"`
	RequestedBy    string          `json:"requestedBy"`
	RequestedAt    time.Time       `json:"requestedAt"`
	BindingEpoch   int64           `json:"bindingEpoch"`
	IdempotencyKey string          `json:"idempotencyKey"`
	Status         string          `json:"status"`
	FailureCode    string          `json:"failureCode,omitempty"`
	ResultRefs     json.RawMessage `json:"resultRefs,omitempty"`
	ResultVersion  int64           `json:"resultVersion,omitempty"`
}

type ComputeCommandStore interface {
	QueueComputeCommand(context.Context, string, ComputeCommand) (ComputeCommand, bool, error)
	ComputeCommandStatus(context.Context, string, string) (ComputeCommand, error)
	SetComputeCommandStatus(context.Context, string, string, string, string) error
}
