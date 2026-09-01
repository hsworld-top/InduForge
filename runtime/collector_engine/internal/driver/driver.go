package driver

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/indu-forge/collector-engine/internal/loader"
	"time"
)

var ErrUnsupported = errors.New("采集驱动未注册")

type Result struct {
	Value           json.RawMessage
	Quality         string
	SourceTimestamp time.Time
}
type Driver interface {
	ID() string
	Ready(context.Context, loader.Connection, loader.BindingConnection) error
	Read(context.Context, loader.Connection, loader.Mapping) (Result, error)
}
type Registry struct{ drivers map[string]Driver }

func NewRegistry(items ...Driver) *Registry {
	r := &Registry{drivers: map[string]Driver{}}
	for _, d := range items {
		if d != nil {
			r.drivers[d.ID()] = d
		}
	}
	return r
}
func (r *Registry) Get(id string) (Driver, error) {
	if r == nil || r.drivers[id] == nil {
		return nil, ErrUnsupported
	}
	return r.drivers[id], nil
}
