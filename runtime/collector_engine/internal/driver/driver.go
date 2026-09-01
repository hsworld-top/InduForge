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

// BatchReader 是可选的同连接批读扩展；调度器仅把同采样周期的 mapping 交给它。
// V1 Modbus schema 的 readOptions 为严格空对象，因此默认按协议网络字节序解码，
// 不支持字节/字交换、scale 或 offset，避免私自接受 schema 外现场配置。
type BatchReader interface {
	ReadBatch(context.Context, loader.Connection, loader.BindingConnection, []loader.Mapping) (map[string]Result, error)
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
