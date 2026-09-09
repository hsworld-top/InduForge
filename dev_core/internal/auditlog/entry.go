package auditlog

import (
	"encoding/json"
	"fmt"
	"github.com/indu-forge/dev_core/internal/auth"
	api "github.com/indu-forge/dev_core/internal/platform/api"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
)

// 入口由 Studio 显式上报，工程名称从本组织数据库读取，避免伪造显示名称。
func EntryHandler(pool *pgxpool.Pool, writer Writer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := auth.UserFromContext(r.Context())
		if !ok {
			api.WriteError(w, r, 401, api.ErrorCodeTokenRequired, "请先登录")
			return
		}
		var input struct {
			ProjectID string `json:"projectId"`
			Target    string `json:"target"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&input) != nil {
			api.WriteError(w, r, 400, api.ErrorCodeInvalidRequest, "入口参数无效")
			return
		}
		target := map[string]string{"workspace": "开发工作区", "datacenter": "数据中心"}[input.Target]
		if target == "" {
			api.WriteError(w, r, 400, api.ErrorCodeInvalidRequest, "入口无效")
			return
		}
		var name string
		if err := pool.QueryRow(r.Context(), `SELECT name FROM projects WHERE id::text=$1 AND tenant_id::text=$2 AND status <> 'deleted'`, input.ProjectID, actor.TenantID).Scan(&name); err != nil {
			api.WriteError(w, r, 404, api.ErrorCodeNotFound, "工程不存在")
			return
		}
		err := writer.Create(r.Context(), Log{TenantID: actor.TenantID, UserID: actor.ID, Level: "INFO", Action: "enter", Resource: "projects", ResourceID: input.ProjectID, Message: fmt.Sprintf("进入“%s”的%s", name, target), Result: "success", RequestID: api.RequestIDFromContext(r.Context()), IP: requestIP(r), Metadata: map[string]any{"entry": input.Target}})
		if err != nil {
			api.WriteError(w, r, 500, api.ErrorCodeInternal, "记录操作失败")
			return
		}
		api.WriteSuccess(w, r, map[string]bool{"recorded": true})
	}
}
