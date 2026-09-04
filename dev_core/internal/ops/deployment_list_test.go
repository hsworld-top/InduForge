package ops

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5"
)

type pageRows struct {
	pgx.Rows
	owner        *pageReader
	index, count int
	services     bool
}

func (r *pageRows) Next() bool { r.index++; return r.index <= r.count }
func (r *pageRows) Close()     { r.owner.active = false }
func (r *pageRows) Err() error { return nil }
func (r *pageRows) Scan(dest ...any) error {
	for _, value := range dest {
		reflect.ValueOf(value).Elem().SetZero()
	}
	*dest[0].(*string) = fmt.Sprintf("deployment-%d", r.index)
	*dest[1].(*string) = "tenant"
	if r.services {
		*dest[2].(*string) = fmt.Sprintf("deployment-%d", r.index)
		*dest[5].(*string) = ServiceBase
	}
	return nil
}

type pageReader struct {
	t           *testing.T
	size, calls int
	active      bool
	arguments   [][]any
}

func (r *pageReader) Query(_ context.Context, query string, args ...any) (pgx.Rows, error) {
	if r.active {
		r.t.Fatal("持有上个rows时进行了嵌套查询")
	}
	r.calls++
	r.arguments = append(r.arguments, args)
	r.active = true
	if r.calls == 2 && query != deploymentPageServicesSQL {
		r.t.Fatal("服务查询必须使用租户及当前页ID批量查询")
	}
	return &pageRows{owner: r, count: r.size, services: r.calls == 2}, nil
}
func (r *pageReader) QueryRow(_ context.Context, _ string, args ...any) pgx.Row {
	if r.active {
		r.t.Fatal("count前未关闭rows")
	}
	r.calls++
	r.arguments = append(r.arguments, args)
	return lifecycleRow{values: []any{int64(r.size)}}
}
func TestDeploymentPageUsesThreeQueriesForAnyPageSize(t *testing.T) {
	for _, size := range []int{0, 20, 200} {
		r := &pageReader{t: t, size: size}
		items, total, err := loadDeploymentPage(context.Background(), r, "tenant", PageFilter{Page: 2, PageSize: size, Search: "demo", ProjectID: "project", EnvironmentID: "environment"})
		if err != nil || r.calls != 3 || len(items) != size || total != int64(size) {
			t.Fatalf("size=%d calls=%d items=%d total=%d err=%v", size, r.calls, len(items), total, err)
		}
		if !reflect.DeepEqual(r.arguments[0], []any{"tenant", "demo", size, size, "project", "environment"}) || !reflect.DeepEqual(r.arguments[2], []any{"tenant", "demo", "project", "environment"}) {
			t.Fatalf("分页/count过滤不一致: %v", r.arguments)
		}
		ids := r.arguments[1][1].([]string)
		if len(ids) != size || r.arguments[1][0] != "tenant" {
			t.Fatal("批量服务查询越过当前租户或页")
		}
		for _, item := range items {
			if len(item.Services) != 1 || item.Services[0].ProjectDeploymentID != item.ID {
				t.Fatalf("服务关联错误: %+v", item)
			}
		}
	}
}
