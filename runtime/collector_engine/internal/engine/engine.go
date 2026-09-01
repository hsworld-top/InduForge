// Package engine 按连接和采样周期调度读取；暂不含工业协议实现。
package engine

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/indu-forge/collector-engine/internal/driver"
	"github.com/indu-forge/collector-engine/internal/event"
	"github.com/indu-forge/collector-engine/internal/health"
	"github.com/indu-forge/collector-engine/internal/loader"
	"github.com/indu-forge/collector-engine/internal/publisher"
	"sync"
	"time"
)

type Engine struct {
	loaded    *loader.Loaded
	drivers   *driver.Registry
	publisher publisher.Publisher
	health    *health.State
	mu        sync.Mutex
	sequence  int64
}

func New(loaded *loader.Loaded, drivers *driver.Registry, p publisher.Publisher, h *health.State) (*Engine, error) {
	if loaded == nil || drivers == nil || p == nil || h == nil {
		return nil, errors.New("collector 依赖非法")
	}
	return &Engine{loaded: loaded, drivers: drivers, publisher: p, health: h}, nil
}
func (e *Engine) CheckReady(ctx context.Context) error {
	if err := e.publisher.Ready(ctx); err != nil {
		e.health.SetReady(false)
		return err
	}
	bindings := map[string]loader.BindingConnection{}
	for _, b := range e.loaded.Binding.Connections {
		bindings[b.ConnectionID] = b
	}
	for _, c := range e.loaded.Artifact.Connections {
		if !c.Enabled {
			continue
		}
		d, err := e.drivers.Get(c.DriverID)
		if err != nil {
			e.health.SetReady(false)
			return err
		}
		if err = d.Ready(ctx, c, bindings[c.ConnectionID]); err != nil {
			e.health.SetReady(false)
			return errors.New("采集驱动未就绪")
		}
	}
	e.health.SetReady(true)
	return nil
}
func (e *Engine) Run(ctx context.Context) error {
	if err := e.CheckReady(ctx); err != nil {
		return err
	}
	groups := map[int64][]loader.Mapping{}
	for _, m := range e.loaded.Artifact.PointMappings {
		if m.Enabled {
			groups[m.EffectiveAcquisition.IntervalMS] = append(groups[m.EffectiveAcquisition.IntervalMS], m)
		}
	}
	var wg sync.WaitGroup
	for interval, mappings := range groups {
		interval, mappings := interval, mappings
		wg.Add(1)
		go func() { defer wg.Done(); e.readLoop(ctx, time.Duration(interval)*time.Millisecond, mappings) }()
	}
	<-ctx.Done()
	wg.Wait()
	e.health.SetReady(false)
	return nil
}
func (e *Engine) readLoop(ctx context.Context, interval time.Duration, mappings []loader.Mapping) {
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			e.readBatch(ctx, mappings)
			timer.Reset(interval)
		}
	}
}
func (e *Engine) readBatch(ctx context.Context, mappings []loader.Mapping) {
	conns := map[string]loader.Connection{}
	for _, c := range e.loaded.Artifact.Connections {
		conns[c.ConnectionID] = c
	}
	for _, m := range mappings {
		d, err := e.drivers.Get(conns[m.ConnectionID].DriverID)
		if err != nil {
			e.health.SetReady(false)
			return
		}
		r, err := d.Read(ctx, conns[m.ConnectionID], m)
		if err != nil {
			r.Quality = "bad"
			r.Value = []byte("null")
			r.SourceTimestamp = time.Now().UTC()
		}
		if r.Quality == "" {
			r.Quality = "unknown"
		}
		e.mu.Lock()
		seq := e.sequence
		e.sequence++
		e.mu.Unlock()
		body, err := event.New(e.loaded.Binding, m, seq, r.Value, r.Quality, r.SourceTimestamp, time.Now())
		if err != nil {
			e.health.SetReady(false)
			continue
		}
		raw, err := jsonMarshal(body)
		if err != nil {
			e.health.SetReady(false)
			continue
		}
		if err = e.publisher.Publish(ctx, body.Subject, raw); err != nil {
			if errors.Is(err, publisher.ErrBackpressure) {
				e.health.SetReady(false)
			}
			continue
		}
	}
}

var jsonMarshal = func(v any) ([]byte, error) { return json.Marshal(v) }
