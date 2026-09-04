package ops

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

type nodeClusterTestReader struct {
	calls int
	query string
	args  []any
	rows  *nodeClusterTestRows
	err   error
}

func (r *nodeClusterTestReader) Query(_ context.Context, query string, args ...any) (pgx.Rows, error) {
	r.calls++
	r.query = query
	r.args = args
	return r.rows, r.err
}

type nodeClusterTestRows struct {
	pgx.Rows
	items                 []Node
	index                 int
	closed                bool
	scanError, errorAfter error
}

func (r *nodeClusterTestRows) Close() { r.closed = true }
func (r *nodeClusterTestRows) Next() bool {
	if r.index >= len(r.items) {
		return false
	}
	r.index++
	return true
}
func (r *nodeClusterTestRows) Err() error { return r.errorAfter }
func (r *nodeClusterTestRows) Scan(dest ...any) error {
	if r.scanError != nil {
		return r.scanError
	}
	n := r.items[r.index-1]
	values := []any{n.ID, n.NodeKind, n.ClusterID, n.ClusterRole, n.ClusterStatus, n.ClusterMessage, n.ClusterDesiredAction, n.ClusterDesiredGeneration, n.ClusterObservedGeneration, n.ClusterObservedAt, n.EnvironmentNames}
	for i, value := range values {
		reflect.ValueOf(dest[i]).Elem().Set(reflect.ValueOf(value))
	}
	return nil
}

func TestAttachNodePageClustersUsesOneTenantScopedQuery(t *testing.T) {
	for _, size := range []int{1, 50, 200} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			now := time.Now()
			nodes := make([]Node, size)
			ids := make([]string, size)
			for i := range nodes {
				ids[i] = fmt.Sprintf("00000000-0000-0000-0000-%012d", i)
				nodes[i] = Node{ID: ids[i], DisplayName: fmt.Sprint(i)}
			}
			rows := &nodeClusterTestRows{items: []Node{{ID: ids[0], NodeKind: "worker", ClusterID: "cluster", ClusterRole: "agent", ClusterStatus: "ready", ClusterDesiredGeneration: 3, ClusterObservedGeneration: 3, ClusterObservedAt: &now, EnvironmentNames: []string{"factory-a", "factory-b"}}}}
			reader := &nodeClusterTestReader{rows: rows}
			if err := attachNodePageClusters(context.Background(), reader, "tenant", nodes); err != nil {
				t.Fatal(err)
			}
			if reader.calls != 1 || !reflect.DeepEqual(reader.args, []any{"tenant", ids}) {
				t.Fatalf("calls=%d args=%v", reader.calls, reader.args)
			}
			if !strings.Contains(reader.query, "n.tenant_id=$1") || !strings.Contains(reader.query, "ANY($2::uuid[])") || !strings.Contains(reader.query, "e.tenant_id=n.tenant_id") {
				t.Fatal("batch query lacks tenant/id binding")
			}
			if !rows.closed || nodes[0].EnvironmentCount != 2 || nodes[0].ClusterObservedGeneration != 3 || nodes[0].DisplayName != "0" {
				t.Fatalf("cluster attachment failed: %+v", nodes[0])
			}
			if size > 1 && (nodes[1].EnvironmentNames == nil || nodes[1].EnvironmentCount != 0) {
				t.Fatal("unattached node must have empty environments")
			}
		})
	}
}

func TestAttachNodePageClustersEmptyPageAndErrors(t *testing.T) {
	empty := &nodeClusterTestReader{}
	if err := attachNodePageClusters(context.Background(), empty, "tenant", nil); err != nil || empty.calls != 0 {
		t.Fatal("empty page queried database")
	}
	marker := errors.New("database read failure")
	for _, kind := range []string{"query", "scan", "iteration"} {
		t.Run(kind, func(t *testing.T) {
			rows := &nodeClusterTestRows{items: []Node{{ID: "node"}}}
			reader := &nodeClusterTestReader{rows: rows}
			switch kind {
			case "query":
				reader.err = marker
			case "scan":
				rows.scanError = marker
			case "iteration":
				rows.errorAfter = marker
			}
			if err := attachNodePageClusters(context.Background(), reader, "tenant", []Node{{ID: "node"}}); !errors.Is(err, marker) {
				t.Fatalf("error=%v", err)
			}
			if kind != "query" && !rows.closed {
				t.Fatal("rows leaked after failed read")
			}
		})
	}
}

func TestNodeListsReleaseRowsBeforeBatchAndUseStableOrdering(t *testing.T) {
	for _, spec := range []struct{ file, method, next, order string }{
		{"repository.go", "func (r *PostgreSQLRepository) ListNodes(", "// LoadHostNodeAddresses", "ORDER BY n.created_at DESC,n.id DESC"},
		{"environment_repository.go", "func (r *PostgreSQLRepository) ListRuntimeEnvironmentNodes(", "func (r *PostgreSQLRepository) attachNodeCluster(", "ORDER BY en.created_at DESC,n.id DESC"},
	} {
		t.Run(spec.file, func(t *testing.T) {
			raw, err := os.ReadFile(spec.file)
			if err != nil {
				t.Fatal(err)
			}
			body := strings.SplitN(strings.SplitN(string(raw), spec.method, 2)[1], spec.next, 2)[0]
			if strings.Contains(body, "attachNodeCluster(ctx,") || !strings.Contains(body, spec.order) || strings.Contains(body, "ORDER BY n.updated_at") {
				t.Fatal("node list regressed to N+1 or heartbeat-dependent sort")
			}
			batch := strings.Index(body, "r.attachNodeClusters(ctx,")
			if batch < 0 {
				t.Fatal("node list missing batch attachment")
			}
			closeRows := strings.LastIndex(body[:batch], "rows.Close()")
			if batch < 0 || closeRows < 0 || !strings.Contains(body[:batch], "rows.Err()") {
				t.Fatal("pagination rows not checked and closed before batch")
			}
		})
	}
}
