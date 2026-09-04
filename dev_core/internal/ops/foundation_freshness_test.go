package ops

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"strconv"
	"strings"
	"testing"
)

type freshnessTestReader struct {
	calls  int
	err    error
	active bool
}

func (r *freshnessTestReader) Query(_ context.Context, _ string, args ...any) (pgx.Rows, error) {
	r.calls++
	if r.err != nil {
		return nil, r.err
	}
	if r.active {
		return nil, errors.New("rows未关闭")
	}
	r.active = true
	start, _ := strconv.Atoi(args[0].(string))
	count := 1200 - start
	if count > 200 {
		count = 200
	}
	return &freshnessTestRows{reader: r, start: start, count: count}, nil
}

type freshnessTestRows struct {
	pgx.Rows
	reader              *freshnessTestReader
	start, count, index int
}

func (r *freshnessTestRows) Next() bool { r.index++; return r.index <= r.count }
func (r *freshnessTestRows) Close()     { r.reader.active = false }
func (r *freshnessTestRows) Err() error { return nil }
func (r *freshnessTestRows) Scan(dest ...any) error {
	*dest[0].(*string) = fmt.Sprintf("%06d", r.start+r.index)
	*dest[1].(*string) = "tenant"
	*dest[2].(*int64) = 7
	*dest[3].(*int64) = 8
	return nil
}

func TestFoundationFreshnessRotatesBeyondThousandAndRetriesErrors(t *testing.T) {
	r := &freshnessTestReader{}
	observer := &changeObserver{}
	seen := map[string]bool{}
	notify := func(tenant, id string, healthy, total int64) {
		if tenant != "tenant" || healthy != 7 || total != 8 {
			t.Fatal("错误的健康投影")
		}
		seen[id] = true
	}
	if err := reconcileFoundationFreshness(context.Background(), r, observer, notify); err != nil || r.calls != 5 || len(seen) != 1000 || observer.freshnessCursor != "001000" {
		t.Fatalf("首轮没有保持1000上限/游标: calls=%d cursor=%s err=%v", r.calls, observer.freshnessCursor, err)
	}
	r.err = errors.New("temporary DB failure")
	if err := reconcileFoundationFreshness(context.Background(), r, observer, notify); err == nil || observer.freshnessCursor != "001000" || len(seen) != 1000 {
		t.Fatal("查询失败推进了游标或指纹")
	}
	r.err = nil
	if err := reconcileFoundationFreshness(context.Background(), r, observer, notify); err != nil || len(seen) != 1200 || observer.freshnessCursor != "" {
		t.Fatal("轮转未覆盖1000以后的环境")
	}
	if !strings.Contains(foundationFreshnessSQL, "s.observed_at>now()-interval '45 seconds'") {
		t.Fatal("时效应与数据库汇总阈值一致")
	}
}

func TestFoundationPayloadUsesDatabaseStalenessRatherThanBrowserClock(t *testing.T) {
	for _, stale := range []bool{false, true} {
		payload := runtimeEnvironmentServicePayload(RuntimeEnvironmentService{ObservedStatus: "running", ObservedStale: stale})
		if payload["observedStale"] != stale {
			t.Fatal("HTTP未透传数据库权威时效")
		}
	}
}
