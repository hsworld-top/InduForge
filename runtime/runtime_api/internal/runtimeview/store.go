package runtimeview

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var ErrNotFound = errors.New("runtime value not found")

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
}
