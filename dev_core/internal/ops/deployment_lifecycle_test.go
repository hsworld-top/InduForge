package ops

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type lifecycleRow struct {
	values []any
	err    error
}

func (r lifecycleRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i, value := range r.values {
		reflect.ValueOf(dest[i]).Elem().Set(reflect.ValueOf(value))
	}
	return nil
}

type lifecycleStatement struct {
	sql  string
	args []any
}

type lifecycleTx struct {
	pgx.Tx
	t                   *testing.T
	mode, versionID     string
	descriptor          []byte
	busy, deleting      bool
	missing, incomplete bool
	queries, writes     []lifecycleStatement
}

func (tx *lifecycleTx) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	tx.queries = append(tx.queries, lifecycleStatement{query, args})
	switch {
	case strings.Contains(query, "FOR UPDATE"):
		if args[0] != "deployment" || args[1] != "tenant" || !strings.Contains(query, "deleted_at IS NULL") {
			tx.t.Fatalf("缺少租户/软删除约束或目标不正确: %s %v", query, args)
		}
		if tx.missing {
			return lifecycleRow{err: pgx.ErrNoRows}
		}
		return lifecycleRow{values: []any{testProjectID, tx.mode, tx.versionID, tx.descriptor, tx.deleting}}
	case strings.Contains(query, "FROM deployment_runs"):
		if tx.busy {
			return lifecycleRow{values: []any{"pending-run"}}
		}
		return lifecycleRow{err: pgx.ErrNoRows}
	case strings.Contains(query, "FROM application_versions"):
		if !reflect.DeepEqual(args, []any{testVersionID, testProjectID, "tenant"}) {
			tx.t.Fatalf("必须读取当前租户工程固定版本: %v", args)
		}
		return lifecycleRow{values: []any{"ready", "2026.08.31-001", "releases/tenant/project/release.tar.zst", strings.Repeat("a", 64), strings.Repeat("b", 64), strings.Repeat("c", 64), "induforge-release-2026-01", int64(1), []byte(validReleaseManifest(testProjectID))}}
	case strings.Contains(query, "SELECT count(*) FROM deployment_services"):
		count := len(args[2].([]string))
		if tx.incomplete {
			count--
		}
		return lifecycleRow{values: []any{count}}
	case strings.Contains(query, "INSERT INTO deployment_runs"):
		tx.writes = append(tx.writes, lifecycleStatement{query, args})
		return lifecycleRow{values: []any{"run", args[0], args[1], args[2], args[3], "pending", 0, args[4], time.Now(), (*time.Time)(nil)}}
	default:
		tx.t.Fatalf("意外查询: %s", query)
		return lifecycleRow{}
	}
}

func (tx *lifecycleTx) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	tx.writes = append(tx.writes, lifecycleStatement{query, args})
	return pgconn.NewCommandTag("UPDATE 2"), nil
}

func newLifecycleTx(t *testing.T, mode string) *lifecycleTx {
	t.Helper()
	artifact := DevelopmentArtifact{ReleaseID: testVersionID, Version: "__DEV__", ArtifactKey: "development/tenant/project/dev.tar.zst", ArtifactHash: strings.Repeat("a", 64), ArtifactSize: 1, ManifestHash: strings.Repeat("b", 64), ChecksumsHash: strings.Repeat("c", 64), SigningKeyID: "induforge-release-2026-01", Manifest: []byte(validReleaseManifest(testProjectID))}
	descriptor, err := json.Marshal(artifact)
	if err != nil {
		t.Fatal(err)
	}
	versionID := ""
	if mode == "release" {
		versionID, descriptor = testVersionID, []byte(`{}`)
	}
	return &lifecycleTx{t: t, mode: mode, versionID: versionID, descriptor: descriptor}
}

func TestQueueRedeployUsesCurrentArtifactAndExistingDeployCommand(t *testing.T) {
	for _, mode := range []string{"release", "development"} {
		t.Run(mode, func(t *testing.T) {
			tx := newLifecycleTx(t, mode)
			descriptor := string(tx.descriptor)
			run, err := queueDeploymentOperation(context.Background(), tx, "tenant", "deployment", "redeploy", "operator")
			if err != nil || run.Operation != "deploy" || run.ObservedStatus != "pending" || !strings.Contains(run.Message, "重新部署") {
				t.Fatalf("run=%+v err=%v", run, err)
			}
			if tx.mode != mode || string(tx.descriptor) != descriptor || len(tx.writes) != 4 {
				t.Fatalf("重部署不得改制品快照/模式，且必须原子写服务、部署、run、事件: %+v", tx.writes)
			}
			if tx.writes[0].sql != operateDeploymentServicesSQL || !reflect.DeepEqual(tx.writes[0].args, []any{"running", "deploy", "deployment", "tenant", []string{ServiceBase}}) {
				t.Fatalf("必须重下发当前manifest需要的引擎: %+v", tx.writes[0])
			}
			for _, statement := range tx.writes {
				for _, forbidden := range []string{"artifact_descriptor=", "mode=", "application_version_id=", "node_id=", "public_port=", "access_port=", "DELETE FROM", "INSERT INTO deployment_bindings"} {
					if strings.Contains(statement.sql, forbidden) {
						t.Fatalf("生命周期不允许变更制品/放置/数据: %s", statement.sql)
					}
				}
			}
			if !reflect.DeepEqual(tx.writes[2].args, []any{"tenant", "deployment", "deploy", "running", run.Message, "operator"}) || !reflect.DeepEqual(tx.writes[3].args, []any{"run", run.Message}) {
				t.Fatal("run和queued事件应保留操作者、租户及重新部署审计")
			}
			if mode == "development" {
				for _, query := range tx.queries {
					if strings.Contains(query.sql, "application_versions") {
						t.Fatal("开发重部署不能改读生产版本")
					}
				}
			}
		})
	}
}

func TestQueueDeploymentOperationRejectsBusyDeletedOrInvalidArtifactBeforeWrites(t *testing.T) {
	for _, scenario := range []string{"busy", "deleting", "missing", "invalid artifact", "missing engine"} {
		t.Run(scenario, func(t *testing.T) {
			tx := newLifecycleTx(t, "development")
			want := ErrDeploymentBusy
			switch scenario {
			case "busy":
				tx.busy = true
			case "deleting":
				tx.deleting = true
			case "missing":
				tx.missing, want = true, ErrNotFound
			case "invalid artifact":
				tx.descriptor, want = []byte(`{}`), ErrReleaseNotDeployable
			case "missing engine":
				tx.incomplete, want = true, ErrReleaseNotDeployable
			}
			_, err := queueDeploymentOperation(context.Background(), tx, "tenant", "deployment", "redeploy", "operator")
			if !errors.Is(err, want) || len(tx.writes) != 0 {
				t.Fatalf("err=%v writes=%v", err, tx.writes)
			}
			if !strings.Contains(tx.queries[0].sql, "FOR UPDATE") {
				t.Fatal("必须先锁部署再检查互斥")
			}
		})
	}
}

func TestLifecycleStopsHistoricalEnginesAndCanStopInvalidArtifact(t *testing.T) {
	for _, op := range []string{"start", "restart", "stop"} {
		tx := newLifecycleTx(t, "development")
		if op == "stop" {
			tx.descriptor = []byte(`{}`)
		}
		if _, err := queueDeploymentOperation(context.Background(), tx, "tenant", "deployment", op, "operator"); err != nil {
			t.Fatalf("%s: %v", op, err)
		}
		if !strings.Contains(tx.writes[0].sql, "AND ($1='stopped' OR service_type=ANY($5::text[]))") {
			t.Fatal("运行相关动作不能变更历史非required引擎状态或代次；stop必须覆盖全部引擎")
		}
	}
}

func TestLastReadyRequiresRunningGenerationAndHistoricalStoppedConvergence(t *testing.T) {
	for _, predicate := range []string{"s.observed_status<>s.desired_status", "s.observed_generation<>s.desired_generation", "(s.desired_status='running' AND s.desired_generation<>$2)"} {
		if !strings.Contains(deploymentLastReadyPendingServiceFilter, predicate) {
			t.Fatalf("last-ready缺少收敛约束: %s", predicate)
		}
	}
	if strings.Contains(deploymentLastReadyPendingServiceFilter, "s.desired_status<>'running'") || strings.Contains(deploymentLastReadyPendingServiceFilter, "s.observed_generation<>$2") {
		t.Fatal("已停止历史引擎不应阻止当前运行代次成为last-ready")
	}
}
