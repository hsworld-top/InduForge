package alarm

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/indu-forge/runtime-engine/internal/model"
	"github.com/indu-forge/runtime-engine/internal/store/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSweepDuePoisonDoesNotStarveLaterItems(t *testing.T) {
	dsn := os.Getenv("RUNTIME_ENGINE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("RUNTIME_ENGINE_TEST_DATABASE_URL 未设置")
	}
	ctx := context.Background()
	store, audit, cleanup := alarmIntegrationStore(t, ctx, dsn)
	defer cleanup()

	const deploymentID = "alarm-sweep"
	config := model.EngineConfig{DeploymentID: deploymentID, AccountID: "account", ProducerAssignments: []model.ProducerAssignment{{ProducerType: "alarm", Role: "alarm", Ownership: model.Ownership{OwnerID: "alarm-owner", Epoch: 1}}}}
	items := []model.AlarmItem{
		sweepTestItem("11111111-1111-4111-8111-111111111111", "21111111-1111-4111-8111-111111111111", "31111111-1111-4111-8111-111111111111"),
		sweepTestItem("22222222-2222-4222-8222-222222222222", "22222222-2222-4222-8222-222222222222", "32222222-2222-4222-8222-222222222222"),
		sweepTestItem("33333333-3333-4333-8333-333333333333", "23333333-3333-4333-8333-333333333333", "33333333-3333-4333-8333-333333333333"),
		sweepTestItem("44444444-4444-4444-8444-444444444444", "24444444-4444-4444-8444-444444444444", "34444444-4444-4444-8444-444444444444"),
		sweepTestItem("55555555-5555-4555-8555-555555555555", "25555555-5555-4555-8555-555555555555", "35555555-5555-4555-8555-555555555555"),
	}
	handler, err := NewPostgresHandler(model.ProjectArtifact{AlarmItems: items}, config)
	if err != nil {
		t.Fatal(err)
	}
	consumer := postgres.ConsumerRoleToken{OwnerID: "alarm-owner", Epoch: 1}
	producer := postgres.ProducerToken{OwnerID: "alarm-owner", Epoch: 1}
	roleFence, err := store.ActivateRole(ctx, deploymentID, "alarm", consumer, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.ActivateProducer(ctx, deploymentID, "alarm", producer, 0); err != nil {
		t.Fatal(err)
	}

	base := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	now := base.Add(time.Second)
	states := make(map[string]json.RawMessage, len(items))
	for _, item := range items[1:4] {
		state := sweepCandidateState(t, handler.identity(), item, base)
		states[item.ID] = state
	}
	// A is earlier and permanently invalid under the current enabled definition.
	// It remains due and unchanged; Preflight must keep rejecting it on restart.
	states[items[0].ID] = json.RawMessage(`{"activeConditionId":"ffffffff-ffff-4fff-8fff-ffffffffffff"}`)
	if err := store.RunProducerTransaction(ctx, deploymentID, "alarm", producer, func(ctx context.Context, tx *postgres.BusinessTx) error {
		for _, item := range items[:4] {
			state, err := tx.AlarmItemState(ctx, item.ID, item.Revision)
			if err != nil {
				return err
			}
			state.State = states[item.ID]
			state.NextEvaluationAt = &now
			if err := tx.SaveAlarmItemState(ctx, item.ID, state); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := handler.Preflight(ctx, store); err == nil {
		t.Fatal("坏状态必须继续阻止下次启动")
	}

	err = handler.SweepDue(ctx, store, consumer, now, 1)
	assertSweepPoison(t, err, postgres.AlarmSweepPoisonInvalidState)
	assertAlarmStateVersion(t, ctx, audit, deploymentID, items[0].ID, 1)
	assertAlarmStateVersion(t, ctx, audit, deploymentID, items[1].ID, 2)
	assertAlarmOutboxCount(t, ctx, audit, deploymentID, 1)

	// The next direct sweep must skip the same A and commit C rather than
	// treating A as a terminal error. D remains for the concurrent-runner case.
	err = handler.SweepDue(ctx, store, consumer, now, 1)
	assertSweepPoison(t, err, postgres.AlarmSweepPoisonInvalidState)
	assertAlarmStateVersion(t, ctx, audit, deploymentID, items[2].ID, 2)
	assertAlarmOutboxCount(t, ctx, audit, deploymentID, 2)

	var runners sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		runners.Add(1)
		go func() {
			defer runners.Done()
			errs <- handler.SweepDue(ctx, store, consumer, now, 1)
		}()
	}
	runners.Wait()
	close(errs)
	for err := range errs {
		assertSweepPoison(t, err, postgres.AlarmSweepPoisonInvalidState)
	}
	assertAlarmStateVersion(t, ctx, audit, deploymentID, items[3].ID, 2)
	assertAlarmOutboxCount(t, ctx, audit, deploymentID, 3)

	// A poison result is reported to direct callers, but a long-lived runner
	// continues at its bounded interval until its context is cancelled.
	runnerCtx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- handler.RunSweepRunner(runnerCtx, store, consumer, 5*time.Millisecond, 1) }()
	select {
	case err := <-done:
		cancel()
		t.Fatalf("poison must not stop runner: %v", err)
	case <-time.After(25 * time.Millisecond):
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("runner cancel=%v", err)
	}

	// A stale role fence is never classified as poison and the otherwise-due E
	// remains unmodified, proving the transaction did not commit.
	if err := store.RunProducerTransaction(ctx, deploymentID, "alarm", producer, func(ctx context.Context, tx *postgres.BusinessTx) error {
		state, err := tx.AlarmItemState(ctx, items[4].ID, items[4].Revision)
		if err != nil {
			return err
		}
		state.State = sweepCandidateState(t, handler.identity(), items[4], base)
		state.NextEvaluationAt = &now
		return tx.SaveAlarmItemState(ctx, items[4].ID, state)
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ActivateRole(ctx, deploymentID, "alarm", postgres.ConsumerRoleToken{OwnerID: "alarm-owner-2", Epoch: 2}, roleFence.Version); err != nil {
		t.Fatal(err)
	}
	err = handler.SweepDue(ctx, store, consumer, now, 1)
	if !errors.Is(err, postgres.ErrFenceStale) || errors.Is(err, postgres.ErrAlarmSweepItemPoison) {
		t.Fatalf("stale fence must remain fatal: %v", err)
	}
	assertAlarmStateVersion(t, ctx, audit, deploymentID, items[4].ID, 1)
	assertAlarmOutboxCount(t, ctx, audit, deploymentID, 3)
}

func TestSweepDueOnlySelectsCurrentEnabledItemRevisionPairs(t *testing.T) {
	dsn := os.Getenv("RUNTIME_ENGINE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("RUNTIME_ENGINE_TEST_DATABASE_URL 未设置")
	}
	ctx := context.Background()
	store, audit, cleanup := alarmIntegrationStore(t, ctx, dsn)
	defer cleanup()

	const deploymentID = "alarm-sweep-revision"
	config := model.EngineConfig{DeploymentID: deploymentID, AccountID: "account", ProducerAssignments: []model.ProducerAssignment{{ProducerType: "alarm", Role: "alarm", Ownership: model.Ownership{OwnerID: "alarm-owner", Epoch: 1}}}}
	old := sweepTestItem("11111111-1111-4111-8111-111111111111", "21111111-1111-4111-8111-111111111111", "31111111-1111-4111-8111-111111111111")
	old.Revision = 2
	b := sweepTestItem("22222222-2222-4222-8222-222222222222", "22222222-2222-4222-8222-222222222222", "32222222-2222-4222-8222-222222222222")
	c := sweepTestItem("33333333-3333-4333-8333-333333333333", "23333333-3333-4333-8333-333333333333", "33333333-3333-4333-8333-333333333333")
	handler, err := NewPostgresHandler(model.ProjectArtifact{AlarmItems: []model.AlarmItem{old, b, c}}, config)
	if err != nil {
		t.Fatal(err)
	}
	consumer := postgres.ConsumerRoleToken{OwnerID: "alarm-owner", Epoch: 1}
	producer := postgres.ProducerToken{OwnerID: "alarm-owner", Epoch: 1}
	if _, err = store.ActivateRole(ctx, deploymentID, "alarm", consumer, 0); err != nil {
		t.Fatal(err)
	}
	if _, err = store.ActivateProducer(ctx, deploymentID, "alarm", producer, 0); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 30, 10, 0, 1, 0, time.UTC)
	if err = store.RunProducerTransaction(ctx, deploymentID, "alarm", producer, func(ctx context.Context, tx *postgres.BusinessTx) error {
		state, err := tx.AlarmItemState(ctx, old.ID, 1)
		if err != nil {
			return err
		}
		state.NextEvaluationAt = &now
		return tx.SaveAlarmItemState(ctx, old.ID, state)
	}); err != nil {
		t.Fatal(err)
	}
	if err = handler.SweepDue(ctx, store, consumer, now, 1); err != nil {
		t.Fatalf("old revision alone must not be selected: %v", err)
	}
	assertAlarmStateRevisionVersion(t, ctx, audit, deploymentID, old.ID, 1, 1)
	assertAlarmOutboxCount(t, ctx, audit, deploymentID, 0)

	base := now.Add(-time.Second)
	if err = store.RunProducerTransaction(ctx, deploymentID, "alarm", producer, func(ctx context.Context, tx *postgres.BusinessTx) error {
		for _, item := range []model.AlarmItem{b, c} {
			state, err := tx.AlarmItemState(ctx, item.ID, item.Revision)
			if err != nil {
				return err
			}
			state.State = sweepCandidateState(t, handler.identity(), item, base)
			state.NextEvaluationAt = &now
			if err := tx.SaveAlarmItemState(ctx, item.ID, state); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err = handler.SweepDue(ctx, store, consumer, now, 1); err != nil {
		t.Fatalf("old revision before current B must not block B: %v", err)
	}
	assertAlarmStateRevisionVersion(t, ctx, audit, deploymentID, old.ID, 1, 1)
	assertAlarmStateVersion(t, ctx, audit, deploymentID, b.ID, 2)
	assertAlarmOutboxCount(t, ctx, audit, deploymentID, 1)
	if err = handler.SweepDue(ctx, store, consumer, now, 1); err != nil {
		t.Fatalf("current C must remain eligible after B: %v", err)
	}
	assertAlarmStateVersion(t, ctx, audit, deploymentID, c.ID, 2)
	assertAlarmOutboxCount(t, ctx, audit, deploymentID, 2)
}

func sweepTestItem(id, point, condition string) model.AlarmItem {
	return model.AlarmItem{ID: id, Revision: 1, DisplayName: "sweep", Enabled: true, Mode: "point", EvaluationMode: "single", Inputs: []model.Input{{Alias: "v", DatapointID: point}}, Conditions: []model.AlarmCondition{{ID: condition, Kind: "threshold", Operator: "gt", Params: json.RawMessage(`{"threshold":10}`), Severity: "warning", TriggerDelayMS: 1000, Deadband: json.RawMessage(`0`)}}}
}

func sweepCandidateState(t *testing.T, identity Identity, item model.AlarmItem, at time.Time) json.RawMessage {
	t.Helper()
	runtime, err := NewRuntime(identity, model.ProjectArtifact{AlarmItems: []model.AlarmItem{item}}, State{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Apply(PointInput{SchemaVersion: "data.raw.v1", EventID: strings.Repeat("a", 64), PointID: item.Inputs[0].DatapointID, Epoch: 1, Sequence: 1, Value: json.RawMessage(`11`), Quality: "good", SourceTimestamp: at, ServerTimestamp: at, ReceivedAt: at}); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(runtime.State().Items[item.ID])
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func assertSweepPoison(t *testing.T, err error, reason postgres.AlarmSweepPoisonReason) {
	t.Helper()
	var poison *postgres.AlarmSweepPoisonError
	if !errors.Is(err, postgres.ErrAlarmSweepItemPoison) || !errors.As(err, &poison) || len(poison.Reasons) != 1 || poison.Reasons[0] != reason {
		t.Fatalf("want poison %s, got %v", reason, err)
	}
}

func assertAlarmStateVersion(t *testing.T, ctx context.Context, pool *pgxpool.Pool, deploymentID, itemID string, want int64) {
	t.Helper()
	var got int64
	if err := pool.QueryRow(ctx, `SELECT version FROM runtime_engine.alarm_item_state WHERE deployment_id=$1 AND alarm_item_id=$2::uuid`, deploymentID, itemID).Scan(&got); err != nil || got != want {
		t.Fatalf("state %s version=%d err=%v, want %d", itemID, got, err, want)
	}
}

func assertAlarmStateRevisionVersion(t *testing.T, ctx context.Context, pool *pgxpool.Pool, deploymentID, itemID string, wantRevision, wantVersion int64) {
	t.Helper()
	var revision, version int64
	if err := pool.QueryRow(ctx, `SELECT alarm_revision,version FROM runtime_engine.alarm_item_state WHERE deployment_id=$1 AND alarm_item_id=$2::uuid`, deploymentID, itemID).Scan(&revision, &version); err != nil || revision != wantRevision || version != wantVersion {
		t.Fatalf("state %s revision/version=%d/%d err=%v, want %d/%d", itemID, revision, version, err, wantRevision, wantVersion)
	}
}

func assertAlarmOutboxCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, deploymentID string, want int) {
	t.Helper()
	var got int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM runtime_engine.transactional_outbox WHERE deployment_id=$1`, deploymentID).Scan(&got); err != nil || got != want {
		t.Fatalf("outbox=%d err=%v, want %d", got, err, want)
	}
}

func alarmIntegrationStore(t *testing.T, ctx context.Context, dsn string) (*postgres.Store, *pgxpool.Pool, func()) {
	t.Helper()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	name := "runtime_engine_alarm_" + strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_") + "_" + strings.ReplaceAll(time.Now().UTC().Format("150405.000000"), ".", "")
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
		admin.Close()
		t.Fatal(err)
	}
	if _, err = bootstrap.Exec(ctx, postgres.SchemaSQL); err != nil {
		bootstrap.Close()
		admin.Close()
		t.Fatal(err)
	}
	bootstrap.Close()
	store, err := postgres.Open(ctx, testDSN)
	if err != nil {
		admin.Close()
		t.Fatal(err)
	}
	audit, err := pgxpool.New(ctx, testDSN)
	if err != nil {
		store.Close()
		admin.Close()
		t.Fatal(err)
	}
	return store, audit, func() {
		audit.Close()
		store.Close()
		_, _ = admin.Exec(ctx, "DROP DATABASE "+pgx.Identifier{name}.Sanitize())
		admin.Close()
	}
}
