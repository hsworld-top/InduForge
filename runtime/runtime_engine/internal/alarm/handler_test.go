package alarm

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/indu-forge/runtime-engine/internal/ingress"
	"github.com/indu-forge/runtime-engine/internal/model"
	"github.com/indu-forge/runtime-engine/internal/store/postgres"
)

func TestPointInputOfflineAdapterIsOneWay(t *testing.T) {
	base := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)
	for _, test := range []struct {
		quality string
		offline bool
	}{{"good", false}, {"unknown", false}, {"bad", true}} {
		input, err := pointInput(ingress.ValidatedMessage{Event: ingress.Event{SchemaVersion: "data.raw.v1", EventID: eventID("a"), PointID: pointID, Epoch: 1, Sequence: 1, Value: json.RawMessage(`1`), Quality: test.quality, SourceTimestamp: base, ServerTimestamp: base, ReceivedAt: base}})
		if err != nil || input.Offline != test.offline {
			t.Fatalf("quality=%s offline=%v err=%v", test.quality, input.Offline, err)
		}
	}
}

func TestNewPostgresHandlerRequiresOnlyAlarmAssignment(t *testing.T) {
	item := artifact([]model.AlarmCondition{condition("gt", 10, "warning", 0, 0)}).AlarmItems[0]
	config := model.EngineConfig{DeploymentID: "dep", AccountID: "account", ProducerAssignments: []model.ProducerAssignment{{ProducerType: "alarm", Role: "alarm", Ownership: model.Ownership{OwnerID: "alarm-owner", Epoch: 1}}}}
	h, err := NewPostgresHandler(model.ProjectArtifact{AlarmItems: []model.AlarmItem{item}}, config)
	if err != nil || len(h.byPoint[pointID]) != 1 {
		t.Fatalf("handler=%#v err=%v", h, err)
	}
	config.ProducerAssignments = nil
	if _, err := NewPostgresHandler(model.ProjectArtifact{AlarmItems: []model.AlarmItem{item}}, config); err == nil {
		t.Fatal("missing alarm producer assignment accepted")
	}
}

func TestSweepDueNoopsWithoutEnabledItems(t *testing.T) {
	config := model.EngineConfig{DeploymentID: "dep", AccountID: "account", ProducerAssignments: []model.ProducerAssignment{{ProducerType: "alarm", Role: "alarm", Ownership: model.Ownership{OwnerID: "alarm-owner", Epoch: 1}}}}
	disabled := artifact([]model.AlarmCondition{condition("gt", 10, "warning", 0, 0)}).AlarmItems[0]
	disabled.Enabled = false
	for _, test := range []struct {
		name     string
		artifact model.ProjectArtifact
	}{
		{name: "empty artifact", artifact: model.ProjectArtifact{}},
		{name: "all disabled", artifact: model.ProjectArtifact{AlarmItems: []model.AlarmItem{disabled}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler, err := NewPostgresHandler(test.artifact, config)
			if err != nil {
				t.Fatal(err)
			}
			store := &recordingAlarmSweepStore{}
			now := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
			if err := handler.sweepDue(context.Background(), store, postgres.ConsumerRoleToken{OwnerID: "alarm-owner", Epoch: 1}, now, 1); err != nil {
				t.Fatalf("SweepDue err=%v, want nil", err)
			}
			if store.calls != 0 {
				t.Fatalf("store calls=%d, want 0", store.calls)
			}
		})
	}
}

func TestRunSweepRunnerNoEnabledItemsContinuesAcrossTicks(t *testing.T) {
	config := model.EngineConfig{DeploymentID: "dep", AccountID: "account", ProducerAssignments: []model.ProducerAssignment{{ProducerType: "alarm", Role: "alarm", Ownership: model.Ownership{OwnerID: "alarm-owner", Epoch: 1}}}}
	handler, err := NewPostgresHandler(model.ProjectArtifact{}, config)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	initial := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	ticks := make(chan time.Time)
	calls := make(chan time.Time)
	done := make(chan error, 1)
	store := &recordingAlarmSweepStore{}
	go func() {
		done <- runSweepRunner(ctx, initial, ticks, func(now time.Time) error {
			if err := handler.sweepDue(ctx, store, postgres.ConsumerRoleToken{OwnerID: "alarm-owner", Epoch: 1}, now, 1); err != nil {
				return err
			}
			calls <- now
			return nil
		})
	}()

	for _, expected := range []time.Time{initial, initial.Add(time.Second), initial.Add(2 * time.Second)} {
		if !expected.Equal(initial) {
			ticks <- expected
		}
		select {
		case got := <-calls:
			if !got.Equal(expected) {
				t.Fatalf("sweep at %s, want %s", got, expected)
			}
		case <-time.After(time.Second):
			t.Fatal("runner did not process expected sweep")
		}
	}
	select {
	case err := <-done:
		t.Fatalf("runner exited before cancellation: %v", err)
	default:
	}
	if store.calls != 0 {
		t.Fatalf("store calls=%d, want 0", store.calls)
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("runner err=%v, want context.Canceled", err)
	}
}

type recordingAlarmSweepStore struct{ calls int }

func (s *recordingAlarmSweepStore) ProcessAlarmSweep(context.Context, postgres.AlarmSweepMessage, postgres.AlarmSweepHandler) error {
	s.calls++
	return nil
}
