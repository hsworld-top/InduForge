package compute

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/indu-forge/runtime-engine/internal/model"
	"github.com/indu-forge/runtime-engine/internal/store/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRunDueContinuesAfterUnitBusinessFailure(t *testing.T) {
	dsn := os.Getenv("RUNTIME_ENGINE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("RUNTIME_ENGINE_TEST_DATABASE_URL 未设置")
	}
	ctx := context.Background()
	store, audit, cleanup := computeIntegrationStore(t, ctx, dsn)
	defer cleanup()
	var artifact model.ProjectArtifact
	var config model.EngineConfig
	computeFixture(t, "runtime-project-artifact.valid.json", &artifact)
	computeFixture(t, "runtime-engine-config.valid.json", &config)
	// Reuse a second owned output unit as an input-free schedule unit; both
	// transactions are independently observable without changing contracts.
	every, intervalUnit := int64(1), "minutes"
	artifact.ComputeUnits[2].Trigger = model.Trigger{Kind: "schedule", Schedule: &model.Schedule{Kind: "interval", Every: &every, Unit: &intervalUnit}}
	artifact.ComputeUnits[3].Trigger = artifact.ComputeUnits[2].Trigger
	executor := &selectiveExecutor{badID: artifact.ComputeUnits[2].ID}
	handler, err := NewPostgresHandler(artifact, config, executor)
	if err != nil {
		t.Fatal(err)
	}
	consumer := postgres.ConsumerRoleToken{OwnerID: "runtime-engine-compute-0", Epoch: 7}
	producer := postgres.ProducerToken{OwnerID: "runtime-engine-compute-0", Epoch: 7}
	if _, err = store.ActivateRole(ctx, config.DeploymentID, "compute", consumer, 0); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{artifact.ComputeUnits[2].ID, artifact.ComputeUnits[3].ID} {
		if _, err = store.ActivateProducer(ctx, config.DeploymentID, id, producer, 0); err != nil {
			t.Fatal(err)
		}
	}
	inputAt := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
	goodUnit := artifact.ComputeUnits[3]
	if err = store.ProcessScheduled(ctx, postgres.ScheduledMessage{DeploymentID: config.DeploymentID, Role: "compute", ComputeID: goodUnit.ID, Token: consumer, ProducerKey: goodUnit.ID, ProducerToken: producer, OccurredAt: inputAt}, func(ctx context.Context, tx *postgres.BusinessTx) error {
		_, err := tx.UpsertComputeInput(ctx, postgres.ComputeInputSnapshot{DeploymentID: config.DeploymentID, ComputeID: goodUnit.ID, DatapointID: goodUnit.Inputs[0].DatapointID, EventID: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", OwnerID: "collector-owner", Epoch: 1, Sequence: 1, SourceTimestamp: inputAt, ServerTimestamp: inputAt, ReceivedAt: inputAt, Value: json.RawMessage(`81`), Quality: "good"})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 31, 0, 31, 0, 0, time.UTC)
	err = handler.RunDue(ctx, store, consumer, now)
	if !errors.Is(err, ErrUnitBusiness) {
		t.Fatalf("RunDue error=%v, want classified unit failure", err)
	}
	if executor.calls[artifact.ComputeUnits[2].ID] != 1 || executor.calls[artifact.ComputeUnits[3].ID] != 1 {
		t.Fatalf("first due calls=%v", executor.calls)
	}
	var count int
	if err = audit.QueryRow(ctx, `SELECT count(*) FROM runtime_engine.transactional_outbox WHERE deployment_id=$1`, config.DeploymentID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("outbox=%d err=%v", count, err)
	}
	err = handler.RunDue(ctx, store, consumer, now.Add(time.Second))
	if !errors.Is(err, ErrUnitBusiness) {
		t.Fatalf("retry error=%v", err)
	}
	if executor.calls[artifact.ComputeUnits[2].ID] != 2 || executor.calls[artifact.ComputeUnits[3].ID] != 1 {
		t.Fatalf("retry calls=%v", executor.calls)
	}
	if err = audit.QueryRow(ctx, `SELECT count(*) FROM runtime_engine.transactional_outbox WHERE deployment_id=$1`, config.DeploymentID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("retry outbox=%d err=%v", count, err)
	}
}

func TestSweepDebouncesContinuesAfterUnitBusinessFailure(t *testing.T) {
	dsn := os.Getenv("RUNTIME_ENGINE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("RUNTIME_ENGINE_TEST_DATABASE_URL 未设置")
	}
	ctx := context.Background()
	store, audit, cleanup := computeIntegrationStore(t, ctx, dsn)
	defer cleanup()
	var artifact model.ProjectArtifact
	var config model.EngineConfig
	computeFixture(t, "runtime-project-artifact.valid.json", &artifact)
	computeFixture(t, "runtime-engine-config.valid.json", &config)
	artifact.ComputeUnits[3].Trigger = artifact.ComputeUnits[1].Trigger
	executor := &selectiveExecutor{badID: artifact.ComputeUnits[1].ID}
	handler, err := NewPostgresHandler(artifact, config, executor)
	if err != nil {
		t.Fatal(err)
	}
	consumer := postgres.ConsumerRoleToken{OwnerID: "runtime-engine-compute-0", Epoch: 7}
	producer := postgres.ProducerToken{OwnerID: "runtime-engine-compute-0", Epoch: 7}
	if _, err = store.ActivateRole(ctx, config.DeploymentID, "compute", consumer, 0); err != nil {
		t.Fatal(err)
	}
	units := []model.ComputeUnit{artifact.ComputeUnits[1], artifact.ComputeUnits[3]}
	for _, unit := range units {
		if _, err = store.ActivateProducer(ctx, config.DeploymentID, unit.ID, producer, 0); err != nil {
			t.Fatal(err)
		}
	}
	pendingAt := time.Date(2026, 8, 31, 1, 0, 0, 0, time.UTC)
	for _, unit := range units {
		unit := unit
		if err = store.ProcessScheduled(ctx, postgres.ScheduledMessage{DeploymentID: config.DeploymentID, Role: "compute", ComputeID: unit.ID, Token: consumer, ProducerKey: unit.ID, ProducerToken: producer, OccurredAt: pendingAt}, func(ctx context.Context, tx *postgres.BusinessTx) error {
			if _, err := tx.UpsertComputeInput(ctx, postgres.ComputeInputSnapshot{DeploymentID: config.DeploymentID, ComputeID: unit.ID, DatapointID: unit.Inputs[0].DatapointID, EventID: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", OwnerID: "collector-owner", Epoch: 1, Sequence: 1, SourceTimestamp: pendingAt, ServerTimestamp: pendingAt, ReceivedAt: pendingAt, Value: json.RawMessage(`81`), Quality: "good"}); err != nil {
				return err
			}
			state, err := tx.ComputeTriggerState(ctx, unit.ID, unit.Revision)
			if err != nil {
				return err
			}
			state.PendingPhase, state.PendingSince = "entered", &pendingAt
			return tx.SaveComputeTriggerState(ctx, unit.ID, state)
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err = handler.SweepDebounces(ctx, store, consumer, pendingAt.Add(time.Second)); !errors.Is(err, ErrUnitBusiness) {
		t.Fatalf("sweep error=%v", err)
	}
	if executor.calls[units[0].ID] != 1 || executor.calls[units[1].ID] != 1 {
		t.Fatalf("calls=%v", executor.calls)
	}
	var count int
	if err = audit.QueryRow(ctx, `SELECT count(*) FROM runtime_engine.transactional_outbox WHERE deployment_id=$1`, config.DeploymentID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("outbox=%d err=%v", count, err)
	}
	if err = handler.SweepDebounces(ctx, store, consumer, pendingAt.Add(2*time.Second)); !errors.Is(err, ErrUnitBusiness) {
		t.Fatalf("retry error=%v", err)
	}
	if executor.calls[units[0].ID] != 2 || executor.calls[units[1].ID] != 1 {
		t.Fatalf("retry calls=%v", executor.calls)
	}
}

type selectiveExecutor struct {
	badID string
	calls map[string]int
}

func (e *selectiveExecutor) Execute(_ context.Context, request ExecutionRequest) (ExecutionResult, error) {
	if e.calls == nil {
		e.calls = map[string]int{}
	}
	e.calls[request.ComputeUnitID]++
	if request.ComputeUnitID == e.badID {
		return ExecutionResult{Output: json.RawMessage(`{"unexpected":1}`)}, nil
	}
	return ExecutionResult{Output: json.RawMessage(`1`)}, nil
}

// TestSweepDebouncesRetryUsesStableExecutionID verifies both sides of the
// crash boundary: a failed business transaction retains the pending state and
// retries the identical sandbox execution, while a committed retry creates one
// outbox record even if a later scheduler tick uses a different wall time.
func TestSweepDebouncesRetryUsesStableExecutionID(t *testing.T) {
	dsn := os.Getenv("RUNTIME_ENGINE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("RUNTIME_ENGINE_TEST_DATABASE_URL 未设置")
	}
	ctx := context.Background()
	store, audit, cleanup := computeIntegrationStore(t, ctx, dsn)
	defer cleanup()
	var artifact model.ProjectArtifact
	var config model.EngineConfig
	computeFixture(t, "runtime-project-artifact.valid.json", &artifact)
	computeFixture(t, "runtime-engine-config.valid.json", &config)
	executor := &retryingDebounceExecutor{}
	handler, err := NewPostgresHandler(artifact, config, executor)
	if err != nil {
		t.Fatal(err)
	}
	unit := artifact.ComputeUnits[1]
	consumer := postgres.ConsumerRoleToken{OwnerID: "runtime-engine-compute-0", Epoch: 7}
	producer := postgres.ProducerToken{OwnerID: "runtime-engine-compute-0", Epoch: 7}
	if _, err = store.ActivateRole(ctx, config.DeploymentID, "compute", consumer, 0); err != nil {
		t.Fatal(err)
	}
	if _, err = store.ActivateProducer(ctx, config.DeploymentID, unit.ID, producer, 0); err != nil {
		t.Fatal(err)
	}
	pendingAt := time.Date(2026, 8, 30, 1, 2, 3, 456000000, time.UTC)
	if err = store.ProcessScheduled(ctx, postgres.ScheduledMessage{DeploymentID: config.DeploymentID, Role: "compute", ComputeID: unit.ID, Token: consumer, ProducerKey: unit.ID, ProducerToken: producer, OccurredAt: pendingAt}, func(ctx context.Context, tx *postgres.BusinessTx) error {
		if _, err := tx.UpsertComputeInput(ctx, postgres.ComputeInputSnapshot{
			DeploymentID: config.DeploymentID, ComputeID: unit.ID, DatapointID: unit.Inputs[0].DatapointID,
			EventID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", OwnerID: "collector-owner", Epoch: 1, Sequence: 1,
			SourceTimestamp: pendingAt, ServerTimestamp: pendingAt, ReceivedAt: pendingAt, Value: json.RawMessage(`81`), Quality: "good",
		}); err != nil {
			return err
		}
		state, err := tx.ComputeTriggerState(ctx, unit.ID, unit.Revision)
		if err != nil {
			return err
		}
		state.PendingPhase, state.PendingSince = "entered", &pendingAt
		return tx.SaveComputeTriggerState(ctx, unit.ID, state)
	}); err != nil {
		t.Fatal(err)
	}

	if err := handler.SweepDebounces(ctx, store, consumer, pendingAt.Add(time.Second)); err == nil {
		t.Fatal("invalid first sandbox output must roll back the pending transition")
	}
	if err := handler.SweepDebounces(ctx, store, consumer, pendingAt.Add(3*time.Second)); err != nil {
		t.Fatalf("retry sweep failed: %v", err)
	}
	if err := handler.SweepDebounces(ctx, store, consumer, pendingAt.Add(9*time.Second)); err != nil {
		t.Fatalf("post-commit sweep failed: %v", err)
	}
	if len(executor.requests) != 2 || executor.requests[0].ExecutionID != executor.requests[1].ExecutionID {
		t.Fatalf("retry execution IDs=%v, want identical two-call retry", executor.executionIDs())
	}
	var outboxCount int
	if err := audit.QueryRow(ctx, `SELECT count(*) FROM runtime_engine.transactional_outbox WHERE deployment_id=$1`, config.DeploymentID).Scan(&outboxCount); err != nil {
		t.Fatal(err)
	}
	if outboxCount != 1 {
		t.Fatalf("outbox count=%d, want 1", outboxCount)
	}
}

type retryingDebounceExecutor struct{ requests []ExecutionRequest }

func (e *retryingDebounceExecutor) Execute(_ context.Context, request ExecutionRequest) (ExecutionResult, error) {
	e.requests = append(e.requests, request)
	if len(e.requests) == 1 {
		return ExecutionResult{Output: json.RawMessage(`{"unexpected":1}`)}, nil
	}
	return ExecutionResult{Output: json.RawMessage(`1`)}, nil
}

func (e *retryingDebounceExecutor) executionIDs() []string {
	ids := make([]string, len(e.requests))
	for i, request := range e.requests {
		ids[i] = request.ExecutionID
	}
	return ids
}

func TestRunDueContextFailuresStopRemainingUnits(t *testing.T) {
	dsn := os.Getenv("RUNTIME_ENGINE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("RUNTIME_ENGINE_TEST_DATABASE_URL 未设置")
	}
	for _, test := range []struct {
		name string
		run  func(context.Context, func()) (error, error)
		want error
	}{
		{
			name: "outer canceled", want: context.Canceled,
			run: func(ctx context.Context, cancel func()) (error, error) {
				return errors.New("sandbox failed"), cancelThenReturn(cancel)
			},
		},
		{
			name: "outer deadline", want: context.DeadlineExceeded,
			run: func(ctx context.Context, _ func()) (error, error) {
				<-ctx.Done()
				return errors.New("sandbox failed"), nil
			},
		},
		{
			name: "executor raw canceled", want: context.Canceled,
			run: func(context.Context, func()) (error, error) { return context.Canceled, nil },
		},
		{
			name: "executor raw deadline", want: context.DeadlineExceeded,
			run: func(context.Context, func()) (error, error) { return context.DeadlineExceeded, nil },
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			store, _, cleanup := computeIntegrationStore(t, context.Background(), dsn)
			defer cleanup()
			executor := &contextExecutor{failID: "66666666-6666-4666-8666-666666666666", run: test.run}
			handler, units, consumer := scheduledContextHandler(t, store, executor)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			executor.cancel = cancel
			if test.want == context.DeadlineExceeded && test.name == "outer deadline" {
				var deadlineCancel context.CancelFunc
				ctx, deadlineCancel = context.WithTimeout(context.Background(), 20*time.Millisecond)
				defer deadlineCancel()
			}
			err := handler.RunDue(ctx, store, consumer, time.Date(2026, 8, 31, 0, 31, 0, 0, time.UTC))
			if !errors.Is(err, test.want) || errors.Is(err, ErrUnitBusiness) {
				t.Fatalf("RunDue error=%v, want fatal %v", err, test.want)
			}
			if executor.calls[units[0].ID] != 1 || executor.calls[units[1].ID] != 0 {
				t.Fatalf("calls=%v, want A stopped and B uncalled", executor.calls)
			}
		})
	}
}

func TestSweepDebouncesRawContextFailureStopsRemainingUnits(t *testing.T) {
	dsn := os.Getenv("RUNTIME_ENGINE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("RUNTIME_ENGINE_TEST_DATABASE_URL 未设置")
	}
	for _, want := range []error{context.Canceled, context.DeadlineExceeded} {
		t.Run(want.Error(), func(t *testing.T) {
			ctx := context.Background()
			store, _, cleanup := computeIntegrationStore(t, ctx, dsn)
			defer cleanup()
			executor := &contextExecutor{failID: "55555555-5555-4555-8555-555555555555", run: func(context.Context, func()) (error, error) { return want, nil }}
			handler, units, consumer, pendingAt := debounceContextHandler(t, store, executor)
			err := handler.SweepDebounces(ctx, store, consumer, pendingAt.Add(time.Second))
			if !errors.Is(err, want) || errors.Is(err, ErrUnitBusiness) {
				t.Fatalf("SweepDebounces error=%v, want fatal %v", err, want)
			}
			if executor.calls[units[0].ID] != 1 || executor.calls[units[1].ID] != 0 {
				t.Fatalf("calls=%v, want A stopped and B uncalled", executor.calls)
			}
		})
	}
}

func TestRunDueSandboxTimeoutIsUnitBusinessAndContinues(t *testing.T) {
	dsn := os.Getenv("RUNTIME_ENGINE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("RUNTIME_ENGINE_TEST_DATABASE_URL 未设置")
	}
	ctx := context.Background()
	store, audit, cleanup := computeIntegrationStore(t, ctx, dsn)
	defer cleanup()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(100 * time.Millisecond):
		}
	}))
	defer server.Close()
	endpoint, err := NewSandboxEndpoint(server.URL, "012345678901234567890123")
	if err != nil {
		t.Fatal(err)
	}
	sandbox, err := NewSandboxClient(staticSandboxResolver{endpoint: endpoint}, "resource", "secret")
	if err != nil {
		t.Fatal(err)
	}
	executor := sandboxThenSuccessExecutor{sandbox: sandbox, failID: "66666666-6666-4666-8666-666666666666"}
	handler, units, consumer := scheduledContextHandler(t, store, &executor)
	handler.units[units[0].ID] = withTimeout(units[0], 20)
	err = handler.RunDue(ctx, store, consumer, time.Date(2026, 8, 31, 0, 31, 0, 0, time.UTC))
	if !errors.Is(err, ErrUnitBusiness) || !errors.Is(err, ErrSandboxTimeout) || errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("RunDue error=%v, want UnitBusiness wrapping ErrSandboxTimeout only", err)
	}
	if executor.calls[units[0].ID] != 1 || executor.calls[units[1].ID] != 1 {
		t.Fatalf("calls=%v, want timeout A and committed B", executor.calls)
	}
	var count int
	if err = audit.QueryRow(ctx, `SELECT count(*) FROM runtime_engine.transactional_outbox WHERE deployment_id=$1`, handler.config.DeploymentID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("outbox=%d err=%v, want B committed", count, err)
	}
}

func TestRunDueCancellationAfterSuccessfulExecuteStopsBeforeValidation(t *testing.T) {
	dsn := os.Getenv("RUNTIME_ENGINE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("RUNTIME_ENGINE_TEST_DATABASE_URL 未设置")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store, audit, cleanup := computeIntegrationStore(t, context.Background(), dsn)
	defer cleanup()
	executor := &contextExecutor{failID: "66666666-6666-4666-8666-666666666666", cancel: cancel, run: func(context.Context, func()) (error, error) { return nil, cancelThenReturn(cancel) }}
	handler, units, consumer := scheduledContextHandler(t, store, executor)
	err := handler.RunDue(ctx, store, consumer, time.Date(2026, 8, 31, 0, 31, 0, 0, time.UTC))
	if !errors.Is(err, context.Canceled) || errors.Is(err, ErrUnitBusiness) {
		t.Fatalf("RunDue error=%v, want fatal context.Canceled", err)
	}
	if executor.calls[units[0].ID] != 1 || executor.calls[units[1].ID] != 0 {
		t.Fatalf("calls=%v, want no validation/commit path and B uncalled", executor.calls)
	}
	var count int
	if err = audit.QueryRow(context.Background(), `SELECT count(*) FROM runtime_engine.transactional_outbox WHERE deployment_id=$1`, handler.config.DeploymentID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("outbox=%d err=%v, canceled transaction must stay atomic", count, err)
	}
}

func cancelThenReturn(cancel func()) error {
	cancel()
	return nil
}

type contextExecutor struct {
	failID string
	run    func(context.Context, func()) (error, error)
	cancel func()
	calls  map[string]int
}

func (e *contextExecutor) Execute(ctx context.Context, request ExecutionRequest) (ExecutionResult, error) {
	if e.calls == nil {
		e.calls = map[string]int{}
	}
	e.calls[request.ComputeUnitID]++
	if request.ComputeUnitID == e.failID {
		resultErr, controlErr := e.run(ctx, e.cancel)
		if controlErr != nil {
			return ExecutionResult{}, controlErr
		}
		return ExecutionResult{Output: json.RawMessage(`1`)}, resultErr
	}
	return ExecutionResult{Output: json.RawMessage(`1`)}, nil
}

type sandboxThenSuccessExecutor struct {
	sandbox *SandboxClient
	failID  string
	calls   map[string]int
}

func (e *sandboxThenSuccessExecutor) Execute(ctx context.Context, request ExecutionRequest) (ExecutionResult, error) {
	if e.calls == nil {
		e.calls = map[string]int{}
	}
	e.calls[request.ComputeUnitID]++
	if request.ComputeUnitID == e.failID {
		return e.sandbox.Execute(ctx, request)
	}
	return ExecutionResult{Output: json.RawMessage(`1`)}, nil
}

func scheduledContextHandler(t *testing.T, store *postgres.Store, executor Executor) (*Handler, []model.ComputeUnit, postgres.ConsumerRoleToken) {
	t.Helper()
	ctx := context.Background()
	var artifact model.ProjectArtifact
	var config model.EngineConfig
	computeFixture(t, "runtime-project-artifact.valid.json", &artifact)
	computeFixture(t, "runtime-engine-config.valid.json", &config)
	every, intervalUnit := int64(1), "minutes"
	units := []model.ComputeUnit{artifact.ComputeUnits[2], artifact.ComputeUnits[3]}
	for i := range units {
		units[i].Trigger = model.Trigger{Kind: "schedule", Schedule: &model.Schedule{Kind: "interval", Every: &every, Unit: &intervalUnit}}
		artifact.ComputeUnits[2+i] = units[i]
	}
	handler, err := NewPostgresHandler(artifact, config, executor)
	if err != nil {
		t.Fatal(err)
	}
	consumer := postgres.ConsumerRoleToken{OwnerID: "runtime-engine-compute-0", Epoch: 7}
	producer := postgres.ProducerToken{OwnerID: "runtime-engine-compute-0", Epoch: 7}
	if _, err = store.ActivateRole(ctx, config.DeploymentID, "compute", consumer, 0); err != nil {
		t.Fatal(err)
	}
	for _, unit := range units {
		if _, err = store.ActivateProducer(ctx, config.DeploymentID, unit.ID, producer, 0); err != nil {
			t.Fatal(err)
		}
	}
	// 第二个 fixture 单元原本带一个输入，保留它以覆盖成功单元真正的提交路径。
	inputAt := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
	if err = store.ProcessScheduled(ctx, postgres.ScheduledMessage{DeploymentID: config.DeploymentID, Role: "compute", ComputeID: units[1].ID, Token: consumer, ProducerKey: units[1].ID, ProducerToken: producer, OccurredAt: inputAt}, func(ctx context.Context, tx *postgres.BusinessTx) error {
		_, err := tx.UpsertComputeInput(ctx, postgres.ComputeInputSnapshot{DeploymentID: config.DeploymentID, ComputeID: units[1].ID, DatapointID: units[1].Inputs[0].DatapointID, EventID: "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", OwnerID: "collector-owner", Epoch: 1, Sequence: 1, SourceTimestamp: inputAt, ServerTimestamp: inputAt, ReceivedAt: inputAt, Value: json.RawMessage(`81`), Quality: "good"})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return handler, units, consumer
}

func debounceContextHandler(t *testing.T, store *postgres.Store, executor Executor) (*Handler, []model.ComputeUnit, postgres.ConsumerRoleToken, time.Time) {
	t.Helper()
	ctx := context.Background()
	var artifact model.ProjectArtifact
	var config model.EngineConfig
	computeFixture(t, "runtime-project-artifact.valid.json", &artifact)
	computeFixture(t, "runtime-engine-config.valid.json", &config)
	artifact.ComputeUnits[3].Trigger = artifact.ComputeUnits[1].Trigger
	handler, err := NewPostgresHandler(artifact, config, executor)
	if err != nil {
		t.Fatal(err)
	}
	units := []model.ComputeUnit{artifact.ComputeUnits[1], artifact.ComputeUnits[3]}
	consumer := postgres.ConsumerRoleToken{OwnerID: "runtime-engine-compute-0", Epoch: 7}
	producer := postgres.ProducerToken{OwnerID: "runtime-engine-compute-0", Epoch: 7}
	if _, err = store.ActivateRole(ctx, config.DeploymentID, "compute", consumer, 0); err != nil {
		t.Fatal(err)
	}
	pendingAt := time.Date(2026, 8, 31, 1, 0, 0, 0, time.UTC)
	for _, unit := range units {
		if _, err = store.ActivateProducer(ctx, config.DeploymentID, unit.ID, producer, 0); err != nil {
			t.Fatal(err)
		}
		unit := unit
		if err = store.ProcessScheduled(ctx, postgres.ScheduledMessage{DeploymentID: config.DeploymentID, Role: "compute", ComputeID: unit.ID, Token: consumer, ProducerKey: unit.ID, ProducerToken: producer, OccurredAt: pendingAt}, func(ctx context.Context, tx *postgres.BusinessTx) error {
			if _, err := tx.UpsertComputeInput(ctx, postgres.ComputeInputSnapshot{DeploymentID: config.DeploymentID, ComputeID: unit.ID, DatapointID: unit.Inputs[0].DatapointID, EventID: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd", OwnerID: "collector-owner", Epoch: 1, Sequence: 1, SourceTimestamp: pendingAt, ServerTimestamp: pendingAt, ReceivedAt: pendingAt, Value: json.RawMessage(`81`), Quality: "good"}); err != nil {
				return err
			}
			state, err := tx.ComputeTriggerState(ctx, unit.ID, unit.Revision)
			if err != nil {
				return err
			}
			state.PendingPhase, state.PendingSince = "entered", &pendingAt
			return tx.SaveComputeTriggerState(ctx, unit.ID, state)
		}); err != nil {
			t.Fatal(err)
		}
	}
	return handler, units, consumer, pendingAt
}

func withTimeout(unit model.ComputeUnit, timeoutMS int64) model.ComputeUnit {
	unit.TimeoutMS = timeoutMS
	return unit
}

func computeFixture(t *testing.T, name string, target any) {
	t.Helper()
	raw, err := os.ReadFile("../../../../contracts/runtime/fixtures/" + name)
	if err != nil || json.Unmarshal(raw, target) != nil {
		t.Fatalf("读取 fixture %s 失败", name)
	}
}

func computeIntegrationStore(t *testing.T, ctx context.Context, dsn string) (*postgres.Store, *pgxpool.Pool, func()) {
	t.Helper()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	name := "runtime_engine_compute_" + strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_") + "_" + strings.ReplaceAll(time.Now().UTC().Format("150405.000000"), ".", "")
	if _, err = admin.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	parsed, err := url.Parse(dsn)
	if err != nil {
		admin.Close()
		t.Fatal(err)
	}
	parsed.Path = "/" + name
	testDSN := parsed.String()
	bootstrap, err := pgxpool.New(ctx, testDSN)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = bootstrap.Exec(ctx, postgres.SchemaSQL); err != nil {
		bootstrap.Close()
		t.Fatal(err)
	}
	bootstrap.Close()
	store, err := postgres.Open(ctx, testDSN)
	if err != nil {
		t.Fatal(err)
	}
	audit, err := pgxpool.New(ctx, testDSN)
	if err != nil {
		store.Close()
		t.Fatal(err)
	}
	return store, audit, func() {
		audit.Close()
		store.Close()
		_, _ = admin.Exec(ctx, "DROP DATABASE "+pgx.Identifier{name}.Sanitize())
		admin.Close()
	}
}
