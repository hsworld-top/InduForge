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
	"math/rand/v2"
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
	retries   map[string]retryState
	now       func() time.Time
	jitter    func(time.Duration) time.Duration
}
type retryState struct {
	failures int
	until    time.Time
}

func New(loaded *loader.Loaded, drivers *driver.Registry, p publisher.Publisher, h *health.State) (*Engine, error) {
	if loaded == nil || drivers == nil || p == nil || h == nil {
		return nil, errors.New("collector 依赖非法")
	}
	return &Engine{loaded: loaded, drivers: drivers, publisher: p, health: h, retries: map[string]retryState{}, now: time.Now, jitter: defaultJitter}, nil
}
func defaultJitter(base time.Duration) time.Duration {
	return base + time.Duration(rand.Int64N(int64(base/5+1)))
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func (e *Engine) retrying(id string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.retries[id].until.After(e.now())
}
func (e *Engine) failed(id string) {
	e.mu.Lock()
	s := e.retries[id]
	s.failures++
	base := time.Second * time.Duration(1<<min(s.failures-1, 5))
	if base > 30*time.Second {
		base = 30 * time.Second
	}
	s.until = e.now().Add(e.jitter(base))
	e.retries[id] = s
	e.mu.Unlock()
	e.health.SetReady(false)
}
func (e *Engine) succeeded(id string) {
	e.mu.Lock()
	delete(e.retries, id)
	empty := len(e.retries) == 0
	e.mu.Unlock()
	if empty {
		e.health.SetReady(true)
	}
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
	bindings := map[string]loader.BindingConnection{}
	for _, c := range e.loaded.Artifact.Connections {
		conns[c.ConnectionID] = c
	}
	for _, binding := range e.loaded.Binding.Connections {
		bindings[binding.ConnectionID] = binding
	}
	byConnection := map[string][]loader.Mapping{}
	for _, m := range mappings {
		byConnection[m.ConnectionID] = append(byConnection[m.ConnectionID], m)
	}
	for connectionID, group := range byConnection {
		if e.retrying(connectionID) {
			e.health.SetReady(false)
			continue
		}
		d, err := e.drivers.Get(conns[connectionID].DriverID)
		if err != nil {
			e.health.SetReady(false)
			continue
		}
		if batch, ok := d.(driver.BatchReader); ok {
			results, readErr := batch.ReadBatch(ctx, conns[connectionID], bindings[connectionID], group)
			if readErr != nil {
				e.failed(connectionID)
				continue
			}
			published := true
			for _, m := range group {
				if !e.publishResult(ctx, m, results[m.DatapointID]) {
					published = false
				}
			}
			if published {
				e.succeeded(connectionID)
			} else {
				e.failed(connectionID)
			}
			continue
		}
		for _, m := range group {
			d, err := e.drivers.Get(conns[m.ConnectionID].DriverID)
			if err != nil {
				e.failed(connectionID)
				return
			}
			var r driver.Result
			if bound, ok := d.(interface {
				ReadWithBinding(context.Context, loader.Connection, loader.BindingConnection, loader.Mapping) (driver.Result, error)
			}); ok {
				r, err = bound.ReadWithBinding(ctx, conns[m.ConnectionID], bindings[m.ConnectionID], m)
			} else {
				r, err = d.Read(ctx, conns[m.ConnectionID], m)
			}
			if err != nil {
				// 断线、超时或协议异常没有可验证的样本，不能伪造 null 或旧值发布。
				e.health.SetReady(false)
				continue
			}
			if r.Quality == "" {
				r.Quality = "unknown"
			}
			if e.publishResult(ctx, m, r) {
				e.succeeded(connectionID)
			} else {
				e.failed(connectionID)
			}
		}
	}
}
func (e *Engine) publishResult(ctx context.Context, m loader.Mapping, r driver.Result) bool {
	e.mu.Lock()
	seq := e.sequence
	e.sequence++
	e.mu.Unlock()
	body, err := event.New(e.loaded.Binding, m, seq, r.Value, r.Quality, r.SourceTimestamp, time.Now())
	if err != nil {
		e.health.SetReady(false)
		return false
	}
	raw, err := jsonMarshal(body)
	if err != nil {
		e.health.SetReady(false)
		return false
	}
	if err = e.publisher.Publish(ctx, body.Subject, raw); err != nil {
		e.health.SetReady(false)
		return false
	}
	return true
}

var jsonMarshal = func(v any) ([]byte, error) { return json.Marshal(v) }
