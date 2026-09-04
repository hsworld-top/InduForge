package project

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

type summaryReader struct {
	t      *testing.T
	rows   [][]any
	args   []any
	calls  int
	closed bool
	err    error
}

func (r *summaryReader) Query(_ context.Context, query string, args ...any) (pgx.Rows, error) {
	r.calls++
	r.args = args
	if query != deploymentSummarySQL {
		r.t.Fatal("未使用正式批量SQL")
	}
	return &summaryRows{reader: r}, r.err
}

type summaryRows struct {
	pgx.Rows
	reader *summaryReader
	index  int
}

func (r *summaryRows) Next() bool { return r.index < len(r.reader.rows) }
func (r *summaryRows) Scan(dest ...any) error {
	row := r.reader.rows[r.index]
	r.index++
	if len(row) != len(dest) {
		return errors.New("scan arity")
	}
	for i := range dest {
		reflect.ValueOf(dest[i]).Elem().Set(reflect.ValueOf(row[i]))
	}
	return nil
}
func (r *summaryRows) Close()     { r.reader.closed = true }
func (r *summaryRows) Err() error { return nil }
func deploymentRow(project, deployment, environment, service string, pending, deleting bool, desiredGeneration, observedGeneration int64) []any {
	return []any{project, deployment, environment, "环境", false, "production", "version-id", "v1", 18080, "running", "pending", deleting, time.Date(2026, 9, 3, 1, 0, 0, 0, time.UTC), "deploy", pending, service, "node", "节点", "running", "pending", desiredGeneration, observedGeneration}
}

func TestDeploymentSummariesAreBatchedAndPrimaryOnlyWhenUnique(t *testing.T) {
	reader := &summaryReader{t: t, rows: [][]any{deploymentRow("p1", "d1", "e1", "base", false, false, 2, 1), deploymentRow("p1", "d1", "e1", "compute", false, false, 2, 2), deploymentRow("p2", "d2", "e1", "base", false, false, 1, 1), deploymentRow("p2", "d3", "e2", "base", false, true, 1, 1)}}
	items := []Project{{ID: "p1"}, {ID: "p2"}, {ID: "p3"}}
	if err := attachDeploymentSummaries(context.Background(), reader, "tenant", items); err != nil {
		t.Fatal(err)
	}
	if reader.calls != 1 || !reader.closed || !reflect.DeepEqual(reader.args, []any{"tenant", []string{"p1", "p2", "p3"}}) {
		t.Fatalf("必须当前页单次tenant+ANY查询 %+v", reader)
	}
	one := items[0].DeploymentSummary
	if one.DeploymentCount != 1 || one.EnvironmentCount != 1 || one.PrimarySelection != "unique" || one.PrimaryDeployment == nil || len(one.PrimaryDeployment.Services) != 2 || one.PrimaryDeployment.Mode != "production" || !one.OperationInProgress || !one.PrimaryDeployment.Updating {
		t.Fatalf("唯一部署摘要错误 %+v", one)
	}
	many := items[1].DeploymentSummary
	if many.DeploymentCount != 2 || many.EnvironmentCount != 2 || many.PrimarySelection != "multiple" || many.PrimaryDeployment != nil || !many.OperationInProgress {
		t.Fatalf("多部署不得取最近/default冒充primary %+v", many)
	}
	if zero := items[2].DeploymentSummary; zero.DeploymentCount != 0 || zero.PrimarySelection != "none" || zero.PrimaryDeployment != nil {
		t.Fatalf("无部署摘要错误 %+v", zero)
	}
	payload := projectResponse(items[0])["deploymentSummary"]
	raw, _ := json.Marshal(payload)
	text := string(raw)
	for _, part := range []string{`"deploymentCount":1`, `"primaryDeployment"`, `"placements":{"base":"node","compute":"node"}`, `"services"`, `"updating":true`} {
		if !strings.Contains(text, part) {
			t.Fatalf("响应契约缺少%s: %s", part, text)
		}
	}
	untouched := projectResponse(Project{ID: "detail"})
	if _, exists := untouched["deploymentSummary"]; exists {
		t.Fatal("未读取摘要不能伪装0部署")
	}
}

func TestFailedOrReverseGenerationIsNotOperationInProgress(t *testing.T) {
	failed := deploymentRow("failed-project", "failed-deployment", "e1", "base", false, false, 2, 1)
	failed[10], failed[19] = "failed", "failed"
	reverse := deploymentRow("reverse-project", "reverse-deployment", "e1", "base", false, false, 1, 2)
	degraded := deploymentRow("degraded-project", "degraded-deployment", "e1", "base", false, false, 2, 1)
	degraded[10] = "degraded"
	reader := &summaryReader{t: t, rows: [][]any{failed, reverse, degraded}}
	items := []Project{{ID: "failed-project"}, {ID: "reverse-project"}, {ID: "degraded-project"}}
	if err := attachDeploymentSummaries(context.Background(), reader, "tenant", items); err != nil {
		t.Fatal(err)
	}
	for _, item := range items {
		summary := item.DeploymentSummary
		if summary.OperationInProgress || summary.PrimaryDeployment == nil || summary.PrimaryDeployment.OperationInProgress || summary.PrimaryDeployment.Updating {
			t.Fatalf("失败/异常/反向代次不能永久阻止处理: %s %+v", item.ID, summary)
		}
	}
}

func TestDeploymentSummaryEmptyPageDoesNotQuery(t *testing.T) {
	reader := &summaryReader{t: t}
	if err := attachDeploymentSummaries(context.Background(), reader, "tenant", nil); err != nil || reader.calls != 0 {
		t.Fatal("空页不应查询")
	}
}

func TestDeploymentSummarySQLScopesTenantPageAndStableSources(t *testing.T) {
	for _, part := range []string{"d.tenant_id=$1", "d.project_id=ANY($2::uuid[])", "d.deleted_at IS NULL", "e.tenant_id=d.tenant_id", "v.tenant_id=d.tenant_id", "s.tenant_id=d.tenant_id", "n.tenant_id=d.tenant_id", "ORDER BY started_at DESC,id DESC LIMIT 1", "CASE WHEN d.mode='release' THEN 'production'", "d.deletion_requested_at IS NOT NULL", "s.desired_generation"} {
		if !strings.Contains(deploymentSummarySQL, part) {
			t.Fatalf("SQL边界缺失 %s", part)
		}
	}
}

func TestProjectDeploymentDeletionRequestIsInFormalSchema(t *testing.T) {
	raw, err := os.ReadFile("../../db/schema/core-schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	schema := string(raw)
	start := strings.Index(schema, "CREATE TABLE project_deployments (")
	if start < 0 {
		t.Fatal("缺少project_deployments正式表")
	}
	end := strings.Index(schema[start:], ");")
	if end < 0 {
		t.Fatal("project_deployments定义未闭合")
	}
	table := schema[start : start+end]
	if strings.Count(table, "deletion_requested_at timestamptz,") != 1 {
		t.Fatal("删除请求字段必须在project_deployments最终基线且无默认值")
	}
}
