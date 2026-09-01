package engine

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/indu-forge/collector-engine/internal/driver"
	"github.com/indu-forge/collector-engine/internal/health"
	"github.com/indu-forge/collector-engine/internal/loader"
	"github.com/indu-forge/collector-engine/internal/publisher"
	"testing"
	"time"
)

type fakeDriver struct{}

func (fakeDriver) ID() string { return "modbus.tcp" }
func (fakeDriver) Ready(context.Context, loader.Connection, loader.BindingConnection) error {
	return nil
}
func (fakeDriver) Read(context.Context, loader.Connection, loader.Mapping) (driver.Result, error) {
	return driver.Result{Value: json.RawMessage(`2`), Quality: "good", SourceTimestamp: time.Now().UTC()}, nil
}

type fakePublisher struct {
	err   error
	calls int
}

func (p *fakePublisher) Ready(context.Context) error                   { return nil }
func (p *fakePublisher) Publish(context.Context, string, []byte) error { p.calls++; return p.err }
func (*fakePublisher) Close()                                          {}
func fixture() *loader.Loaded {
	return &loader.Loaded{Artifact: loader.Artifact{Connections: []loader.Connection{{ConnectionID: "33333333-3333-4333-8333-333333333333", DriverID: "modbus.tcp", Enabled: true}}, PointMappings: []loader.Mapping{{DatapointID: "22222222-2222-4222-8222-222222222222", ConnectionID: "33333333-3333-4333-8333-333333333333", VariableID: "44444444-4444-4444-8444-444444444444", Enabled: true, EffectiveAcquisition: loader.Acquisition{IntervalMS: 1}}}}, Binding: loader.Binding{DeploymentID: "deployment-a", AccountID: "account-a", CollectorID: "collector-a", Ownership: loader.Ownership{OwnerID: "owner-a", Epoch: 1}}}
}
func TestBackpressureMakesNotReady(t *testing.T) {
	h := &health.State{}
	p := &fakePublisher{err: publisher.ErrBackpressure}
	e, _ := New(fixture(), driver.NewRegistry(fakeDriver{}), p, h)
	e.health.SetReady(true)
	e.readBatch(context.Background(), fixture().Artifact.PointMappings)
	if h.Ready() {
		t.Fatal("backpressure must make collector not ready")
	}
	if p.calls != 1 {
		t.Fatal("expected publish")
	}
}
func TestUnsupportedDriverIsNotReady(t *testing.T) {
	h := &health.State{}
	e, _ := New(fixture(), driver.NewRegistry(), &fakePublisher{}, h)
	if !errors.Is(e.CheckReady(context.Background()), driver.ErrUnsupported) {
		t.Fatal("must expose unsupported driver")
	}
	if h.Ready() {
		t.Fatal("unsupported driver cannot be ready")
	}
}
func TestConnectionBackoffGrowsCapsAndResets(t *testing.T) {
	h := &health.State{}
	e, _ := New(fixture(), driver.NewRegistry(fakeDriver{}), &fakePublisher{}, h)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	e.now = func() time.Time { return now }
	e.jitter = func(v time.Duration) time.Duration { return v }
	e.failed("c")
	if e.retries["c"].until.Sub(now) != time.Second || !e.retrying("c") {
		t.Fatal("first retry must be 1s")
	}
	now = now.Add(time.Second)
	e.failed("c")
	if e.retries["c"].until.Sub(now) != 2*time.Second {
		t.Fatal("second retry must double")
	}
	for i := 0; i < 8; i++ {
		now = e.retries["c"].until
		e.failed("c")
	}
	if got := e.retries["c"].until.Sub(now); got != 30*time.Second {
		t.Fatalf("cap=%s", got)
	}
	e.succeeded("c")
	if len(e.retries) != 0 || !h.Ready() {
		t.Fatal("success must reset retry and ready")
	}
}
