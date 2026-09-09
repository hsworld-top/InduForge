package codeworkspace

import (
	"context"
	"errors"
	"testing"
	"time"
)

type idleEngine struct {
	fakeEngine
	busy     bool
	probeErr error
	probe    func()
}

func (e *idleEngine) RunningProjects(context.Context) ([]string, error) {
	return []string{testProjectID}, nil
}
func (e *idleEngine) WorkspaceBusy(context.Context, string) (bool, error) {
	if e.probe != nil {
		e.probe()
	}
	return e.busy, e.probeErr
}
func TestReclaimIdleRequiresFullWindowAndNoConnections(t *testing.T) {
	now := time.Now()
	engine := &idleEngine{}
	s := &Service{engine: engine, now: func() time.Time { return now }}
	ctx := context.Background()
	s.ReclaimIdle(ctx)
	now = now.Add(29 * time.Minute)
	s.ReclaimIdle(ctx)
	if len(engine.stopped) != 0 {
		t.Fatal("reclaimed before idle window")
	}
	done := s.beginActivity(testProjectID)
	now = now.Add(time.Hour)
	s.ReclaimIdle(ctx)
	if len(engine.stopped) != 0 {
		t.Fatal("reclaimed active connection")
	}
	done()
	now = now.Add(31 * time.Minute)
	s.ReclaimIdle(ctx)
	if len(engine.stopped) != 1 {
		t.Fatal("idle workspace not stopped")
	}
}
func TestReclaimIdleProtectsTasksProbeFailureAndReconnect(t *testing.T) {
	for _, kind := range []string{"busy", "failure", "reconnect"} {
		t.Run(kind, func(t *testing.T) {
			now := time.Now()
			e := &idleEngine{}
			s := &Service{engine: e, now: func() time.Time { return now }}
			s.ReclaimIdle(context.Background())
			now = now.Add(time.Hour)
			switch kind {
			case "busy":
				e.busy = true
			case "failure":
				e.probeErr = errors.New("unavailable")
			case "reconnect":
				e.probe = func() { s.beginActivity(testProjectID)() }
			}
			s.ReclaimIdle(context.Background())
			if len(e.stopped) != 0 {
				t.Fatal("unsafe reclaim")
			}
		})
	}
}
