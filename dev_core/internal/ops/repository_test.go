package ops

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

type scanRowFunc func(...any) error

func (f scanRowFunc) Scan(destinations ...any) error { return f(destinations...) }

type scriptedRowQuerier struct {
	t       *testing.T
	rows    []pgx.Row
	queries []string
	args    [][]any
}

func (q *scriptedRowQuerier) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	q.t.Helper()
	q.queries = append(q.queries, query)
	q.args = append(q.args, args)
	if len(q.rows) == 0 {
		q.t.Fatalf("未预期的 QueryRow: %s", query)
	}
	row := q.rows[0]
	q.rows = q.rows[1:]
	return row
}

func stringRow(value string) pgx.Row {
	return scanRowFunc(func(destinations ...any) error {
		if len(destinations) != 1 {
			return fmt.Errorf("期望 1 个扫描目标，实际 %d", len(destinations))
		}
		*destinations[0].(*string) = value
		return nil
	})
}

func boolRow(value bool) pgx.Row {
	return scanRowFunc(func(destinations ...any) error {
		if len(destinations) != 1 {
			return fmt.Errorf("期望 1 个扫描目标，实际 %d", len(destinations))
		}
		*destinations[0].(*bool) = value
		return nil
	})
}

func enrollmentRow(clusterID string) pgx.Row {
	now := time.Now()
	return scanRowFunc(func(destinations ...any) error {
		if len(destinations) != 13 {
			return fmt.Errorf("期望 13 个接入任务扫描目标，实际 %d", len(destinations))
		}
		*destinations[0].(*string) = "enrollment-1"
		*destinations[1].(*string) = "tenant-1"
		*destinations[2].(*string) = clusterID
		*destinations[3].(*string) = RoleRuntimeLinux
		*destinations[4].(*string) = "运行节点-1"
		*destinations[5].(*string) = "created"
		*destinations[6].(*time.Time) = now.Add(time.Hour)
		*destinations[8].(*string) = ""
		*destinations[11].(*time.Time) = now
		*destinations[12].(*time.Time) = now
		return nil
	})
}

func TestCreateEnrollmentRecordCreatesAndBindsDefaultRuntimeCluster(t *testing.T) {
	query := &scriptedRowQuerier{t: t, rows: []pgx.Row{
		stringRow("tenant-1"),
		boolRow(false),
		stringRow(testClusterID),
		enrollmentRow(testClusterID),
	}}
	enrollment, err := createEnrollmentRecord(context.Background(), query, "tenant-1", "user-1", CreateEnrollmentInput{
		Role: RoleRuntimeLinux, DisplayName: "运行节点-1", TTL: time.Hour,
	}, "code-hash")
	if err != nil {
		t.Fatal(err)
	}
	if enrollment.RuntimeClusterID != testClusterID {
		t.Fatalf("接入任务未绑定自动创建的集群: %#v", enrollment)
	}
	if len(query.queries) != 4 || !strings.Contains(query.queries[0], "FOR UPDATE") || !strings.Contains(query.queries[2], "'single_node'") {
		t.Fatalf("自动创建应先锁定租户再写入单节点集群: %#v", query.queries)
	}
	clusterArgs := query.args[2]
	if clusterArgs[1] != "默认运行资源池" || clusterArgs[2] != "default-runtime" || clusterArgs[3] != "由首台 Linux 运行节点接入任务自动创建" {
		t.Fatalf("默认资源池标识不稳定: %#v", clusterArgs)
	}
	var metadata map[string]any
	if err = json.Unmarshal(clusterArgs[4].([]byte), &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata["systemManaged"] != true || metadata["isDefault"] != true || metadata["createdFrom"] != "first_runtime_enrollment" {
		t.Fatalf("默认集群缺少系统创建标记: %#v", metadata)
	}
}

func TestCreateEnrollmentRecordRequiresSelectionWhenClusterExists(t *testing.T) {
	query := &scriptedRowQuerier{t: t, rows: []pgx.Row{stringRow("tenant-1"), boolRow(true)}}
	_, err := createEnrollmentRecord(context.Background(), query, "tenant-1", "user-1", CreateEnrollmentInput{
		Role: RoleRuntimeLinux, TTL: time.Hour,
	}, "code-hash")
	if !errors.Is(err, ErrRuntimeClusterSelectionRequired) {
		t.Fatalf("已有集群时应要求明确选择，实际 %v", err)
	}
	if len(query.queries) != 2 {
		t.Fatalf("已有集群时不应写入默认集群或接入任务: %#v", query.queries)
	}
}

func TestCreateEnrollmentRecordKeepsExplicitRuntimeCluster(t *testing.T) {
	query := &scriptedRowQuerier{t: t, rows: []pgx.Row{enrollmentRow(testClusterID)}}
	enrollment, err := createEnrollmentRecord(context.Background(), query, "tenant-1", "user-1", CreateEnrollmentInput{
		Role: RoleRuntimeLinux, RuntimeClusterID: testClusterID, TTL: time.Hour,
	}, "code-hash")
	if err != nil {
		t.Fatal(err)
	}
	if enrollment.RuntimeClusterID != testClusterID || len(query.queries) != 1 || strings.Contains(query.queries[0], "runtime_clusters") {
		t.Fatalf("显式集群不应触发自动创建: enrollment=%#v queries=%#v", enrollment, query.queries)
	}
}

func TestScanManagementHostNodeIncludesRuntimeClusterName(t *testing.T) {
	row := scanRowFunc(func(destinations ...any) error {
		if len(destinations) != 21 {
			return fmt.Errorf("期望 21 个管理端节点扫描目标，实际 %d", len(destinations))
		}
		*destinations[0].(*string) = "node-1"
		*destinations[20].(*string) = "默认运行资源池"
		return nil
	})
	node, err := scanManagementHostNode(row)
	if err != nil {
		t.Fatal(err)
	}
	if node.ID != "node-1" || node.RuntimeClusterName != "默认运行资源池" {
		t.Fatalf("管理端节点未扫描所属运行集群名称: %#v", node)
	}
}
