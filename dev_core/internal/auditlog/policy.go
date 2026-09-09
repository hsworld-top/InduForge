package auditlog

import (
	"net/http"
	"strings"
)

// 只记录 Studio 管理操作；工程内部编辑、后台查询和运维执行事件不进入此日志。
func studioOperation(method, path string) (string, string, bool) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(path, "/api/v1/"), "/"), "/")
	if len(parts) == 0 {
		return "", "", false
	}
	if parts[0] == "auth" && (len(parts) < 2 || parts[1] == "refresh") {
		return "", "", false
	}
	module := map[string]string{"projects": "工程", "users": "用户", "auth": "账户", "ops": "运维", "logs": "操作日志"}[parts[0]]
	if module == "" {
		return "", "", false
	}
	if method == http.MethodGet {
		if !strings.HasSuffix(path, "/export") {
			return "", "", false
		}
		return "export", "导出" + module, true
	}
	if method != http.MethodPost && method != http.MethodPatch && method != http.MethodPut && method != http.MethodDelete {
		return "", "", false
	}
	if parts[0] == "ops" && (len(parts) < 2 || parts[1] == "agent") {
		return "", "", false
	}
	if parts[0] == "projects" && len(parts) > 2 {
		allowed := map[string]bool{"tags": true, "groups": true, "publish": true, "export": true, "visibility": true, "members": true, "runtime-access": true, "group": true, "operations": true, "runtime-users": true, "runtime-roles": true, "releases": true}
		if !allowed[parts[2]] {
			return "", "", false
		}
	}
	action := actionForMethod(method)
	verb := map[string]string{"create": "创建", "update": "修改", "delete": "删除"}[action]
	suffix := parts[len(parts)-1]
	special := map[string]string{"login": "登录", "logout": "退出登录", "change-password": "修改密码", "reset-password": "重置密码", "publish": "发布工程", "migrate": "调整基础服务分布", "repair": "发起基础服务修复", "foundation-services": "部署基础服务", "start": "启动", "stop": "停止", "restart": "重启", "redeploy": "重新部署", "export": "导出", "import": "导入"}
	if label := special[suffix]; label != "" {
		return suffix, label, true
	}
	return action, verb + module, true
}
