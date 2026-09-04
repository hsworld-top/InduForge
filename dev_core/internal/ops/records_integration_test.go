package ops

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// 仅只读 CTE：用正式查询验证语义，不建表、不写历史数据。
const recordsProjectionFixtureSQL = `WITH users(id,tenant_id,full_name,username) AS (VALUES ('user','tenant','操作者','admin')),
 projects(id,tenant_id,name) AS (VALUES ('project','tenant','demo')),
 project_deployments(id,tenant_id,project_id,environment_id,deleted_at) AS (VALUES
 ('deployment','tenant','project','environment',NULL::timestamptz),('deleted','tenant','project','environment',now())),
 deployment_runs(id,tenant_id,project_deployment_id,operation,observed_status,created_by,started_at,completed_at,message) AS (VALUES
 ('redeploy','tenant','deployment','deploy','running','user',now()-interval '1 second',now(),'重新部署任务已创建：重下发当前制品'),
 ('deleted-run','tenant','deleted','delete','stopped','user',now()-interval '1 second',now(),'删除'),
 ('pending','tenant','deployment','start','pending','user',now(),NULL::timestamptz,'启动')),
 runtime_clusters(id,tenant_id,name) AS (VALUES ('cluster','tenant','中心')),
 runtime_environments(id,tenant_id,name) AS (VALUES ('environment','tenant','环境')),
 runtime_cluster_events(id,tenant_id,cluster_id,node_id,event_type,name,target,result,created_by,created_at,message) AS (VALUES
 ('online','tenant','cluster','node','node_online','恢复在线','节点','success',NULL::text,now(),''),
 ('adjusting','tenant','cluster','node','node_time_sync_changed','正在校准','节点','success',NULL::text,now(),''),
 ('join','tenant','cluster','node','worker_join_requested','加入请求','节点','success','user',now(),'')),
 runtime_environment_events(id,tenant_id,environment_id,event_type,name,result,created_by,created_at,message) AS (VALUES
 ('request','tenant','environment','foundation_redeploy_requested','重部署请求','success','user',now(),''),
 ('update','tenant','environment','environment_updated','已编辑','success','user',now(),''),
 ('mirror','tenant','environment','node_online','恢复在线','success',NULL::text,now(),''),
 ('progress','tenant','environment','foundation_state_changed','正在部署','success',NULL::text,now(),''),
 ('failure','tenant','environment','foundation_state_changed','异常','failed',NULL::text,now(),'')) `

func TestRecordsProjectionPostgreSQL(t *testing.T) {
	dsn := os.Getenv("INDUFORGE_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("未配置 PostgreSQL 集成测试 DSN")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	for _, test := range []struct {
		tenant string
		events bool
		want   int
	}{{"tenant", true, 10}, {"tenant", false, 3}, {"other", true, 0}} {
		rows, err := conn.Query(ctx, recordsProjectionFixtureSQL+`SELECT id,record_type,status,run_id,title FROM (`+opsRecordsUnionSQL+`) records ORDER BY id`, test.tenant, true, test.events)
		if err != nil {
			t.Fatal(err)
		}
		count := 0
		for rows.Next() {
			var id, kind, status, runID, title string
			if err = rows.Scan(&id, &kind, &status, &runID, &title); err != nil {
				t.Fatal(err)
			}
			count++
			switch {
			case strings.HasSuffix(id, ":request"), strings.HasSuffix(id, ":join"):
				if kind != "operation" || status != "accepted" {
					t.Fatal("请求被误判执行成功")
				}
			case strings.HasSuffix(id, ":update"):
				if kind != "operation" || status != "success" {
					t.Fatal("同步操作分类/状态不正确")
				}
			case strings.HasSuffix(id, ":adjusting"), strings.HasSuffix(id, ":progress"):
				if kind != "event" || status != "info" {
					t.Fatal("校准/部署中误判恢复/成功")
				}
			case strings.HasSuffix(id, ":failure"):
				if kind != "event" || status != "warning" {
					t.Fatal("系统失败不得变成任务")
				}
			case strings.HasSuffix(id, ":online"):
				if kind != "event" || status != "recovered" {
					t.Fatal("恢复事件错误")
				}
			case strings.HasSuffix(id, ":deleted-run"):
				if kind != "operation" || status != "success" || runID != "" {
					t.Fatal("删除摘要丢失或生成不可访问任务链接")
				}
			case strings.HasSuffix(id, ":redeploy"):
				if kind != "operation" || status != "success" || title != "重新部署工程" {
					t.Fatal("重部署语义错误")
				}
			case strings.HasSuffix(id, ":pending"):
				if status != "running" {
					t.Fatal("pending不应成功")
				}
			default:
				t.Fatalf("非规范镜像或未知记录 %s", id)
			}
		}
		rows.Close()
		if rows.Err() != nil || count != test.want {
			t.Fatalf("tenant=%s events=%v count=%d want=%d err=%v", test.tenant, test.events, count, test.want, rows.Err())
		}
	}
}
