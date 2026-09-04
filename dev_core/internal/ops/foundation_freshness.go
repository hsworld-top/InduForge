package ops

import (
	"context"
	"github.com/jackc/pgx/v5"
)

type freshnessReader interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

const foundationFreshnessSQL = `SELECT e.id::text,e.tenant_id::text,count(s.id) FILTER (WHERE s.observed_status='running' AND s.observed_at>now()-interval '45 seconds'),count(s.id) FROM runtime_environments e JOIN runtime_environment_services s ON s.environment_id=e.id AND s.tenant_id=e.tenant_id WHERE e.deleted_at IS NULL AND e.id>COALESCE(NULLIF($1,'')::uuid,'00000000-0000-0000-0000-000000000000'::uuid) GROUP BY e.id,e.tenant_id ORDER BY e.id LIMIT 200`

// ReconcileFoundationFreshness 补齐仅由时间流逝引起的健康变化，复用全局10秒liveness循环。
// 每次最多5页/1000环境，游标跨tick推进；没有按客户端ticker或每连接数据查询。
func (r *PostgreSQLRepository) ReconcileFoundationFreshness(ctx context.Context) error {
	if r.events == nil {
		return nil
	}
	return reconcileFoundationFreshness(ctx, r.pool, &r.observer, func(tenant, id string, healthy, total int64) {
		r.observedChange(tenant, []string{"environments"}, []string{id}, false, []int64{healthy, total})
	})
}

func reconcileFoundationFreshness(ctx context.Context, reader freshnessReader, observer *changeObserver, notify func(string, string, int64, int64)) error {
	for page := 0; page < 5; page++ {
		observer.mu.Lock()
		cursor := observer.freshnessCursor
		observer.mu.Unlock()
		rows, err := reader.Query(ctx, foundationFreshnessSQL, cursor)
		if err != nil {
			return err
		}
		type health struct {
			id, tenant     string
			healthy, total int64
		}
		items := []health{}
		for rows.Next() {
			var item health
			if err = rows.Scan(&item.id, &item.tenant, &item.healthy, &item.total); err != nil {
				rows.Close()
				return err
			}
			items = append(items, item)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		// 读取整页成功后才推进游标/指纹，查询失败不得把未知状态标成健康或丢过期事件。
		next := ""
		if len(items) == 200 {
			next = items[len(items)-1].id
		}
		observer.mu.Lock()
		observer.freshnessCursor = next
		observer.mu.Unlock()
		for _, item := range items {
			notify(item.tenant, item.id, item.healthy, item.total)
		}
		if next == "" {
			break
		}
	}
	return nil
}
