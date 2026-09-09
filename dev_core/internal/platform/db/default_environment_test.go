package db

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"strings"
	"testing"
)

type defaultEnvironmentTx struct {
	pgx.Tx
	exists    bool
	failEvent bool
	events    int
	queries   []string
}
type defaultEnvironmentRow struct{ err error }

func (r defaultEnvironmentRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	*dest[0].(*string) = "environment-id"
	return nil
}

type builtInRuntimeTx struct {
	pgx.Tx
	queries []string
	execs   []string
}

type builtInRuntimeRow struct{ value string }

func (r builtInRuntimeRow) Scan(dest ...any) error {
	*dest[0].(*string) = r.value
	if len(dest) == 3 {
		*dest[1].(*string) = "v1.36.4+k3s1"
		*dest[2].(*int) = 6443
	}
	return nil
}

func (tx *builtInRuntimeTx) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	tx.queries = append(tx.queries, sql)
	switch {
	case strings.Contains(sql, "runtime_environments"):
		return builtInRuntimeRow{value: "environment-id"}
	case strings.Contains(sql, "host_nodes"):
		return builtInRuntimeRow{value: "node-id"}
	default:
		return builtInRuntimeRow{value: "cluster-id"}
	}
}

func (tx *builtInRuntimeTx) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	tx.execs = append(tx.execs, sql)
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

func TestBuiltInRuntimeInitialization(t *testing.T) {
	tx := &builtInRuntimeTx{}
	environmentID, nodeID, err := EnsureBuiltInRuntime(context.Background(), tx, "organization", "administrator", BuiltInRuntimeConfig{
		NodeName: "center-01", NodeIP: "172.16.125.129", Architecture: "arm64", K3sVersion: "v1.36.4+k3s1", K3sAPIPort: 6443,
	})
	if err != nil || environmentID != "environment-id" || nodeID != "node-id" {
		t.Fatalf("environment=%s node=%s err=%v", environmentID, nodeID, err)
	}
	joined := strings.Join(append(tx.queries, tx.execs...), "\n")
	for _, expected := range []string{
		"'built_in'", "runtime_clusters", "runtime_cluster_nodes", "'center'", "runtime_environment_nodes", "built_in_node_assigned",
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("中心内置节点初始化缺少 %q", expected)
		}
	}
	if strings.Contains(joined, "node_enrollments") || strings.Contains(joined, "agent_token_hash") {
		t.Fatal("中心内置节点不得创建 NodeAgent 接入身份")
	}
}

func TestBuiltInRuntimeRejectsInvalidClusterConfig(t *testing.T) {
	if _, _, err := EnsureBuiltInRuntime(context.Background(), &builtInRuntimeTx{}, "organization", "administrator", BuiltInRuntimeConfig{}); err == nil {
		t.Fatal("无效的 K3s 配置必须拒绝初始化")
	}
}
func (tx *defaultEnvironmentTx) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	tx.queries = append(tx.queries, sql)
	if strings.HasPrefix(sql, "INSERT") && tx.exists {
		return defaultEnvironmentRow{pgx.ErrNoRows}
	}
	return defaultEnvironmentRow{}
}
func (tx *defaultEnvironmentTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	tx.events++
	if tx.failEvent {
		return pgconn.CommandTag{}, errors.New("event failed")
	}
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}
func TestDefaultEnvironmentInitialization(t *testing.T) {
	for _, tc := range []struct {
		name         string
		exists, fail bool
		events       int
	}{
		{"首次初始化", false, false, 1}, {"重复初始化", true, false, 0}, {"事件失败整体失败", false, true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx := &defaultEnvironmentTx{exists: tc.exists, failEvent: tc.fail}
			id, err := EnsureDefaultRuntimeEnvironment(context.Background(), tx, "organization", "administrator")
			if (err != nil) != tc.fail || (!tc.fail && id != "environment-id") || tx.events != tc.events {
				t.Fatalf("id=%s err=%v events=%d", id, err, tx.events)
			}
			for _, sql := range tx.queries {
				if strings.Contains(sql, "runtime_environment_nodes") || strings.Contains(sql, "runtime_environment_services") {
					t.Fatal("初始化不得关联节点或部署服务")
				}
			}
		})
	}
}
