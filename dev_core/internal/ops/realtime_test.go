package ops

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/indu-forge/dev_core/internal/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type recordedChange struct {
	tenant      string
	topics, ids []string
	terminal    bool
}

type reconciliationTx struct {
	pgx.Tx
	t              *testing.T
	stale          bool
	locked         bool
	statements     []string
	aggregateErr   error
	stopDesired    string
	stopGeneration int64
	stopFailed     bool
}

func (tx *reconciliationTx) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	tx.statements = append(tx.statements, query)
	switch {
	case strings.HasPrefix(query, "SELECT d.desired_status"):
		tx.locked = true
		return lifecycleRow{values: []any{tx.stopDesired, tx.stopGeneration}}
	case strings.HasPrefix(query, "SELECT id::text FROM project_deployments"):
		tx.locked = true
		return lifecycleRow{values: []any{"deployment"}}
	case strings.HasPrefix(query, "SELECT tenant_id::text"):
		if tx.stopDesired != "" {
			return lifecycleRow{values: []any{"tenant", tx.stopDesired, "pending"}}
		}
		return lifecycleRow{values: []any{"tenant", "running", "pending"}}
	case strings.HasPrefix(query, "SELECT count(*) FILTER"):
		if tx.aggregateErr != nil {
			return lifecycleRow{err: tx.aggregateErr}
		}
		if tx.stopDesired != "" {
			failed := 0
			if tx.stopFailed {
				failed = 1
			}
			return lifecycleRow{values: []any{0, failed, int64(0)}}
		}
		pending := 0
		if tx.stale {
			pending = 1
		}
		return lifecycleRow{values: []any{pending, 0, int64(7)}}
	case strings.HasPrefix(query, "WITH latest AS"):
		return lifecycleRow{values: []any{"original-run"}}
	default:
		tx.t.Fatalf("unexpected transaction query %s", query)
		return lifecycleRow{}
	}
}

func (tx *reconciliationTx) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	tx.statements = append(tx.statements, query)
	if !tx.locked {
		tx.t.Fatal("观察及事件必须先持有部署行锁，避免旧事件进入新run")
	}
	if strings.HasPrefix(query, "INSERT INTO deployment_run_events") {
		// 使用正式基线约束检查事务真正送入数据库的阶段，避免 mock 接受非法终态。
		schema, err := os.ReadFile("../../db/schema/core-schema.sql")
		if err != nil {
			tx.t.Fatal(err)
		}
		constraint := regexp.MustCompile(`(?s)CREATE TABLE deployment_run_events \(.*?stage text NOT NULL CHECK \(stage IN \(([^)]+)\)\)`).FindSubmatch(schema)
		if len(constraint) != 2 {
			tx.t.Fatal("缺少正式阶段约束")
		}
		stageIndex := 0
		if strings.Contains(query, "VALUES($1,$2,$3)") {
			stageIndex = 1
		}
		stage, ok := args[stageIndex].(string)
		if !ok || !strings.Contains(string(constraint[1]), "'"+stage+"'") || strings.Contains(query, "'completed'") {
			tx.t.Fatalf("非法事件阶段 query=%s args=%v", query, args)
		}
	}
	if query == recordWorkloadObservationSQL && tx.stale {
		return pgconn.NewCommandTag("UPDATE 0"), nil
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func TestWorkloadObservationAndRunCompletionShareLockedTransaction(t *testing.T) {
	for _, stale := range []bool{false, true} {
		tx := &reconciliationTx{t: t, stale: stale}
		state, err := recordAndReconcileWorkload(context.Background(), tx, ProjectWorkload{DeploymentID: "deployment", ServiceID: "service", Generation: 7, Engine: ServiceBase}, ProjectWorkloadStatus{Ready: true})
		if err != nil {
			t.Fatal(err)
		}
		var hasStage, hasCompletion bool
		for _, query := range tx.statements {
			hasStage = hasStage || query == insertRunEventOnceSQL
			hasCompletion = hasCompletion || strings.HasPrefix(query, "WITH latest AS")
		}
		if stale && (hasStage || hasCompletion || state.changed) {
			t.Fatalf("旧代次不应写新run阶段或终态: %+v", tx.statements)
		}
		if !stale && (!hasStage || !hasCompletion || state.observed != "running" || !state.changed) {
			t.Fatalf("服务、阶段、终态没有共同进入事务: %+v", tx.statements)
		}
	}
	tx := &reconciliationTx{t: t, aggregateErr: errors.New("aggregate failed")}
	if _, err := recordAndReconcileWorkload(context.Background(), tx, ProjectWorkload{DeploymentID: "deployment", Generation: 7}, ProjectWorkloadStatus{Ready: true}); err == nil {
		t.Fatal("聚合失败必须让调用方回滚整个观测事务")
	}
}

func TestStoppedObservationCASAndRunCompletionAreAtomic(t *testing.T) {
	for _, test := range []struct {
		desired          string
		generation       int64
		failed, accepted bool
	}{{"stopped", 7, false, true}, {"stopped", 7, true, true}, {"stopped", 8, false, false}, {"running", 7, true, false}} {
		tx := &reconciliationTx{t: t, stopDesired: test.desired, stopGeneration: test.generation, stopFailed: test.failed}
		state, err := recordStoppedTransaction(context.Background(), tx, "deployment", 7, test.failed, "stopped result")
		if err != nil {
			t.Fatal(err)
		}
		if !test.accepted {
			if len(tx.statements) != 1 || state.changed {
				t.Fatal("旧stop代次/旧失败污染了新操作")
			}
			continue
		}
		if !state.changed || state.tenant != "tenant" {
			t.Fatal("已接受的停止结果未聚合")
		}
		if test.failed && state.observed != "failed" || !test.failed && state.observed != "stopped" {
			t.Fatal("停止终态错误")
		}
		found := false
		for _, q := range tx.statements {
			found = found || strings.HasPrefix(q, "WITH latest AS")
		}
		if !found {
			t.Fatal("停止观察与run完成不在同一事务")
		}
	}
}

type changeRecorder struct {
	mu     sync.Mutex
	values []recordedChange
}

func (r *changeRecorder) PublishChange(tenant string, topics, ids []string, terminal bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.values = append(r.values, recordedChange{tenant, topics, ids, terminal})
}

func TestOpsMutationsPublishOnlyAfterSuccessWithinAuthenticatedTenant(t *testing.T) {
	for _, failure := range []error{nil, ErrDeploymentBusy} {
		repo := &deploymentRepository{deploymentErr: failure}
		events := &changeRecorder{}
		h := NewHandler(NewService(repo, nil), nil)
		h.SetEvents(events)
		router := chi.NewRouter()
		h.MountRoutes(router)
		request := httptest.NewRequest(http.MethodPost, "/api/v1/ops/project-deployments/deployment/redeploy?tenantId=other", nil)
		request = request.WithContext(auth.WithUser(request.Context(), auth.User{TenantID: "tenant", Role: "OPERATOR"}))
		router.ServeHTTP(httptest.NewRecorder(), request)
		if failure != nil {
			if len(events.values) != 0 {
				t.Fatal("失败请求不得发布成功失效")
			}
			continue
		}
		if len(events.values) != 1 || events.values[0].tenant != "tenant" || !reflect.DeepEqual(events.values[0].topics, []string{"deployments", "environments", "events"}) || len(events.values[0].ids) != 0 {
			t.Fatalf("事件越权或未覆盖结构依赖: %+v", events.values)
		}
	}
}

func TestOpsFingerprintDeduplicatesBoundsCacheAndIsConcurrentSafe(t *testing.T) {
	events := &changeRecorder{}
	r := &PostgreSQLRepository{}
	r.SetEvents(events)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.observedChange("tenant", []string{"nodes"}, []string{"node"}, false, map[string]any{"cpu": 12})
		}()
	}
	wg.Wait()
	if len(events.values) != 1 {
		t.Fatalf("重复状态发出了%d次通知", len(events.values))
	}
	r.observedChange("tenant", []string{"nodes"}, []string{"node"}, false, map[string]any{"cpu": 13})
	if len(events.values) != 2 {
		t.Fatal("真实指标变化未通知")
	}
	for i := 0; i < 5000; i++ {
		r.observedChange("tenant", []string{"nodes"}, []string{string(rune(i)) + "node"}, false, i)
	}
	if len(r.observer.values) > 4096 {
		t.Fatal("历史实体指纹缓存无界")
	}
}

type observationExecutor struct {
	calls   []lifecycleStatement
	changed bool
	err     error
}

func (e *observationExecutor) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	e.calls = append(e.calls, lifecycleStatement{query, args})
	if e.err != nil {
		return pgconn.CommandTag{}, e.err
	}
	if !e.changed {
		return pgconn.NewCommandTag("UPDATE 0"), nil
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}
func TestWorkloadObservationCASAndDedup(t *testing.T) {
	for _, changed := range []bool{false, true} {
		executor := &observationExecutor{changed: changed}
		got, err := recordProjectWorkloadObservation(context.Background(), executor, ProjectWorkload{ServiceID: "service", DeploymentID: "deployment", Generation: 7, Engine: ServiceBase}, ProjectWorkloadStatus{Ready: true, ReplicasObserved: 1, Message: "ready"})
		if err != nil || got != changed {
			t.Fatalf("got=%v err=%v", got, err)
		}
		if !reflect.DeepEqual(executor.calls[0].args, []any{"running", "ready", int64(7), 1, "service"}) {
			t.Fatal("未使用捕获的generation")
		}
		if !strings.Contains(executor.calls[0].sql, "desired_generation=$3") || !strings.Contains(executor.calls[0].sql, "IS DISTINCT FROM") {
			t.Fatal("旧代次/重复结果不得写回")
		}
		if !changed && len(executor.calls) != 1 {
			t.Fatal("CAS失败/状态重复不得记录或推送事件")
		}
		if changed && (len(executor.calls) != 2 || !strings.Contains(executor.calls[1].sql, "NOT EXISTS") || executor.calls[1].args[1] != "base：工程引擎已就绪") {
			t.Fatal("阶段事件未按run/stage/message去重")
		}
	}
	executor := &observationExecutor{err: errors.New("write failed")}
	if changed, err := recordProjectWorkloadObservation(context.Background(), executor, ProjectWorkload{}, ProjectWorkloadStatus{}); err == nil || changed || len(executor.calls) != 1 {
		t.Fatal("观测失败不得继续事件写入")
	}
}
