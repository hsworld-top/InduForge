package integration_test

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAuthoringFenceSerializesProjectWritesAndReleasesIdempotently(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	fixture := setupTestDatabase(t, ctx)
	if err := setupSchemaInitializer(t, fixture.pool).Ensure(ctx); err != nil {
		t.Fatalf("schema initialization failed: %v", err)
	}

	projectID, tenantID, ownerID := uuid.NewString(), "tenant-fence", uuid.NewString()
	if _, err := fixture.pool.Exec(ctx, `INSERT INTO data_project_tenant_bindings(project_id,tenant_id,authoring_epoch) VALUES($1,$2,1)`, projectID, tenantID); err != nil {
		t.Fatal(err)
	}
	repo := repository.NewAuthoringFenceRepository(fixture.pool, newAuthoringLockPool(t, ctx, fixture, 2))
	record, err := repo.Acquire(ctx, projectID, tenantID, ownerID, "capture", "epoch-1", time.Minute)
	if err != nil {
		t.Fatalf("acquire fence: %v", err)
	}
	if record.AuthoringEpoch != "epoch-1" {
		t.Fatalf("acquire did not echo current authoring epoch: %#v", record)
	}
	assertConflict := func(label string, err error) {
		t.Helper()
		var epochErr *repository.AuthoringEpochConflict
		if errors.As(err, &epochErr) {
			return
		}
		var appErr *apperrors.AppError
		if !errors.As(err, &appErr) || appErr.StatusCode != http.StatusConflict {
			t.Fatalf("%s must return 409, got %v", label, err)
		}
	}
	_, err = repo.GateNormalWrite(ctx, projectID, tenantID, "epoch-1")
	assertConflict("normal write while fenced", err)
	_, err = repo.Acquire(ctx, projectID, tenantID, uuid.NewString(), "capture", "epoch-1", time.Minute)
	assertConflict("second acquire", err)
	err = repo.Release(ctx, projectID, tenantID, ownerID, "wrong-token")
	assertConflict("wrong token release", err)
	if err = repo.Release(ctx, projectID, tenantID, ownerID, record.Token); err != nil {
		t.Fatalf("release fence: %v", err)
	}
	if err = repo.Release(ctx, projectID, tenantID, ownerID, record.Token); err != nil {
		t.Fatalf("repeated release must be idempotent: %v", err)
	}
	for _, stale := range []string{"", "invalid", "epoch-2"} {
		if _, staleErr := repo.GateNormalWrite(ctx, projectID, tenantID, stale); staleErr == nil {
			t.Fatalf("stale authoring epoch %q was accepted", stale)
		}
	}

	// Acquire 必须等待已经进入 handler 的普通写完成，避免检查 fence 与实际写入之间的竞态。
	releaseWrite, err := repo.GateNormalWrite(ctx, projectID, tenantID, "epoch-1")
	if err != nil {
		t.Fatalf("gate normal write: %v", err)
	}
	type acquireResult struct {
		record *repository.AuthoringFenceRecord
		err    error
	}
	result := make(chan acquireResult, 1)
	secondOwner := uuid.NewString()
	go func() {
		record, acquireErr := repo.Acquire(ctx, projectID, tenantID, secondOwner, "capture", "epoch-1", time.Minute)
		result <- acquireResult{record: record, err: acquireErr}
	}()
	select {
	case early := <-result:
		t.Fatalf("acquire passed an in-flight normal write: %#v", early)
	case <-time.After(150 * time.Millisecond):
	}
	releaseWrite()
	var acquired acquireResult
	select {
	case acquired = <-result:
	case <-time.After(5 * time.Second):
		t.Fatal("acquire did not continue after normal write released")
	}
	if acquired.err != nil || acquired.record == nil {
		t.Fatalf("acquire after normal write: record=%#v err=%v", acquired.record, acquired.err)
	}
	if err = repo.Release(ctx, projectID, tenantID, secondOwner, acquired.record.Token); err != nil {
		t.Fatalf("release second fence: %v", err)
	}
}

func TestAuthoringFencePoolOneDoesNotSelfDeadlock(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	fixture := setupTestDatabase(t, ctx)
	if err := setupSchemaInitializer(t, fixture.pool).Ensure(ctx); err != nil {
		t.Fatalf("schema initialization failed: %v", err)
	}

	config, err := pgxpool.ParseConfig(fixture.databaseURL)
	if err != nil {
		t.Fatalf("parse database URL: %v", err)
	}
	config.MaxConns = 1
	config.ConnConfig.RuntimeParams["search_path"] = fixture.schemaName
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("create one-connection pool: %v", err)
	}
	defer pool.Close()

	repo := repository.NewAuthoringFenceRepository(pool, newAuthoringLockPool(t, ctx, fixture, 2))
	projectID, tenantID, ownerID := uuid.NewString(), "tenant-pool-one", uuid.NewString()
	if _, err := pool.Exec(ctx, `INSERT INTO data_project_tenant_bindings(project_id,tenant_id,authoring_epoch) VALUES($1,$2,1)`, projectID, tenantID); err != nil {
		t.Fatal(err)
	}
	operationCtx, operationCancel := context.WithTimeout(ctx, 3*time.Second)
	defer operationCancel()
	record, err := repo.Acquire(operationCtx, projectID, tenantID, ownerID, "capture", "epoch-1", time.Minute)
	if err != nil {
		t.Fatalf("pool=1 acquire self-deadlocked: %v", err)
	}
	if err = repo.Release(operationCtx, projectID, tenantID, ownerID, record.Token); err != nil {
		t.Fatalf("pool=1 release self-deadlocked: %v", err)
	}
	releaseWrite, err := repo.GateNormalWrite(operationCtx, projectID, tenantID, "epoch-1")
	if err != nil {
		t.Fatalf("pool=1 gate normal write: %v", err)
	}
	if _, err = pool.Exec(operationCtx, `UPDATE data_project_tenant_bindings SET updated_at=now() WHERE project_id=$1`, projectID); err != nil {
		releaseWrite()
		t.Fatalf("pool=1 gated actual write self-deadlocked: %v", err)
	}
	releaseWrite()

	var wait sync.WaitGroup
	errorsCh := make(chan error, 8)
	for range 8 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			release, gateErr := repo.GateNormalWrite(operationCtx, projectID, tenantID, "epoch-1")
			if gateErr != nil {
				errorsCh <- gateErr
				return
			}
			defer release()
			_, writeErr := pool.Exec(operationCtx, `UPDATE data_project_tenant_bindings SET updated_at=now() WHERE project_id=$1`, projectID)
			if writeErr != nil {
				errorsCh <- writeErr
			}
		}()
	}
	wait.Wait()
	close(errorsCh)
	for concurrentErr := range errorsCh {
		t.Fatalf("concurrent gated write starved business pool: %v", concurrentErr)
	}
}

func TestAuthoringFenceRestoreCommitsSnapshotAndEpochInOneExclusiveTransaction(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	fixture := setupTestDatabase(t, ctx)
	if err := setupSchemaInitializer(t, fixture.pool).Ensure(ctx); err != nil {
		t.Fatalf("schema initialization failed: %v", err)
	}
	projectID, tenantID, ownerID, actorID := uuid.NewString(), "tenant-restore-epoch", uuid.NewString(), uuid.NewString()
	if _, err := fixture.pool.Exec(ctx, `INSERT INTO data_project_tenant_bindings(project_id,tenant_id,authoring_epoch) VALUES($1,$2,1)`, projectID, tenantID); err != nil {
		t.Fatal(err)
	}
	fences := repository.NewAuthoringFenceRepository(fixture.pool, newAuthoringLockPool(t, ctx, fixture, 2))
	snapshots := repository.NewProjectSnapshotRepository(fixture.pool)
	record, err := fences.Acquire(ctx, projectID, tenantID, ownerID, "restore", "epoch-1", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := repository.ProjectSnapshot{DataPoints: []repository.DataPointRecord{{
		ID: uuid.NewString(), ProjectID: projectID, Path: "metrics.restored", Name: "restored",
		SourceType: "manual", SourceConfig: map[string]any{}, DataType: "float64", Tags: []any{},
		AttributeDefaults: map[string]string{}, RuntimePermissions: repository.DefaultDataPointRuntimePermissions(), RefreshMode: "manual", Status: "active",
	}}}
	if err = fences.RestoreSnapshot(ctx, snapshots, projectID, tenantID, actorID, ownerID, record.Token, "epoch-1", "epoch-2", "forward", snapshot); err != nil {
		t.Fatalf("forward restore: %v", err)
	}
	var epoch, points int64
	if err = fixture.pool.QueryRow(ctx, `SELECT authoring_epoch FROM data_project_tenant_bindings WHERE project_id=$1`, projectID).Scan(&epoch); err != nil || epoch != 2 {
		t.Fatalf("forward epoch not committed: epoch=%d err=%v", epoch, err)
	}
	if err = fixture.pool.QueryRow(ctx, `SELECT count(*) FROM data_points WHERE project_id=$1 AND path='metrics.restored'`, projectID).Scan(&points); err != nil || points != 1 {
		t.Fatalf("snapshot data not committed with epoch: points=%d err=%v", points, err)
	}
	// 模拟 core 在 forward 已提交后崩溃且租期过期：restore marker 仍必须阻止普通写，
	// 不同任务不能接管；同一 task owner 可以旋转 token 后继续补偿。
	if _, err = fixture.pool.Exec(ctx, `UPDATE data_authoring_fences SET expires_at=now()-interval '1 second' WHERE project_id=$1`, projectID); err != nil {
		t.Fatal(err)
	}
	if _, err = fences.GateNormalWrite(ctx, projectID, tenantID, "epoch-2"); err == nil {
		t.Fatal("expired restore marker allowed a normal write")
	}
	if _, err = fences.Acquire(ctx, projectID, tenantID, uuid.NewString(), "restore", "epoch-2", time.Minute); err == nil {
		t.Fatal("different owner took over an expired restore marker")
	}
	// 控制面可能仍停留在 task.expected；同 owner recovery 必须接受旧代次，
	// 但响应必须回显 data 已经提交的实际 task.target。
	recovered, err := fences.Acquire(ctx, projectID, tenantID, ownerID, "restore", "epoch-1", time.Minute)
	if err != nil {
		t.Fatalf("same owner failed to recover restore marker from task expected epoch: %v", err)
	}
	if recovered.Token == record.Token || recovered.AuthoringEpoch != "epoch-2" {
		t.Fatalf("restore recovery did not rotate token/echo current epoch: %#v", recovered)
	}
	if _, err = fixture.pool.Exec(ctx, `UPDATE data_authoring_fences SET expires_at=now()-interval '1 second' WHERE project_id=$1`, projectID); err != nil {
		t.Fatal(err)
	}
	recoveredFromTarget, err := fences.Acquire(ctx, projectID, tenantID, ownerID, "restore", "epoch-2", time.Minute)
	if err != nil {
		t.Fatalf("same owner failed to recover restore marker from task target epoch: %v", err)
	}
	if recoveredFromTarget.Token == recovered.Token || recoveredFromTarget.AuthoringEpoch != "epoch-2" {
		t.Fatalf("target-epoch recovery did not rotate token/echo current epoch: %#v", recoveredFromTarget)
	}
	record = recoveredFromTarget

	// 非相邻 target 在覆盖开始前失败，既不改变数据也不推进 epoch。
	if err = fences.RestoreSnapshot(ctx, snapshots, projectID, tenantID, actorID, ownerID, record.Token, "epoch-2", "epoch-4", "forward", repository.ProjectSnapshot{}); err == nil {
		t.Fatal("non-adjacent forward epoch must fail")
	}
	if err = fixture.pool.QueryRow(ctx, `SELECT authoring_epoch FROM data_project_tenant_bindings WHERE project_id=$1`, projectID).Scan(&epoch); err != nil || epoch != 2 {
		t.Fatalf("failed restore changed epoch: epoch=%d err=%v", epoch, err)
	}
	if err = fixture.pool.QueryRow(ctx, `SELECT count(*) FROM data_points WHERE project_id=$1`, projectID).Scan(&points); err != nil || points != 1 {
		t.Fatalf("failed restore changed snapshot: points=%d err=%v", points, err)
	}

	if err = fences.RestoreSnapshot(ctx, snapshots, projectID, tenantID, actorID, ownerID, record.Token, "epoch-2", "epoch-1", "compensate", repository.ProjectSnapshot{}); err != nil {
		t.Fatalf("compensating restore: %v", err)
	}
	if err = fixture.pool.QueryRow(ctx, `SELECT authoring_epoch FROM data_project_tenant_bindings WHERE project_id=$1`, projectID).Scan(&epoch); err != nil || epoch != 1 {
		t.Fatalf("compensating epoch not committed: epoch=%d err=%v", epoch, err)
	}
	if err = fixture.pool.QueryRow(ctx, `SELECT count(*) FROM data_points WHERE project_id=$1`, projectID).Scan(&points); err != nil || points != 0 {
		t.Fatalf("compensating snapshot not committed: points=%d err=%v", points, err)
	}
}

func newAuthoringLockPool(t *testing.T, ctx context.Context, fixture *testDatabase, maxConns int32) *pgxpool.Pool {
	t.Helper()
	config, err := pgxpool.ParseConfig(fixture.databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	config.MaxConns = maxConns
	config.MinConns = 0
	config.ConnConfig.RuntimeParams["search_path"] = fixture.schemaName
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}
