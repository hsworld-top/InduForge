package handler

import (
	"fmt"
	"net"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/indu-forge/node_agent/internal/ops"
)

func (h *APIHandler) ListManagedProcesses(w http.ResponseWriter, r *http.Request) {
	if !requireLoopback(w, r) {
		return
	}
	if h.supervisor == nil {
		h.errorResponse(w, http.StatusServiceUnavailable, fmt.Errorf("本机进程托管器未启用"))
		return
	}
	h.jsonResponse(w, http.StatusOK, h.supervisor.List())
}

func (h *APIHandler) ManageProcess(w http.ResponseWriter, r *http.Request) {
	if !requireLoopback(w, r) {
		return
	}
	// 回环地址不足以防止浏览器借助 localhost 发起跨站副作用请求。所有变更操作
	// 额外要求非简单请求头，使普通表单、图片等跨站请求无法触发进程启停。
	action := mux.Vars(r)["action"]
	if action != "status" && r.Header.Get("X-InduForge-Local-Operation") != "1" {
		http.Error(w, "missing local operation confirmation", http.StatusForbidden)
		return
	}
	if h.supervisor == nil {
		h.errorResponse(w, http.StatusServiceUnavailable, fmt.Errorf("本机进程托管器未启用"))
		return
	}
	workloadID := mux.Vars(r)["workloadId"]
	role := ops.ServiceGroup(r.URL.Query().Get("role"))
	generation, _ := strconv.ParseInt(r.URL.Query().Get("generation"), 10, 64)
	var (
		status ops.ProcessStatus
		err    error
	)
	switch action {
	case "start":
		status, err = h.supervisor.Start(workloadID, role, generation)
	case "stop":
		status, err = h.supervisor.Stop(workloadID, generation)
	case "restart":
		status, err = h.supervisor.Restart(workloadID, role, generation)
	case "status":
		status, err = h.supervisor.Status(workloadID)
	default:
		err = fmt.Errorf("不支持的进程操作: %s", action)
	}
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, err)
		return
	}
	h.jsonResponse(w, http.StatusOK, status)
}

func (h *APIHandler) ManagedProcessLogs(w http.ResponseWriter, r *http.Request) {
	if !requireLoopback(w, r) {
		return
	}
	if h.supervisor == nil {
		h.errorResponse(w, http.StatusServiceUnavailable, fmt.Errorf("本机进程托管器未启用"))
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	logs, err := h.supervisor.Logs(mux.Vars(r)["workloadId"], limit)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, err)
		return
	}
	h.jsonResponse(w, http.StatusOK, map[string]any{"lines": logs})
}

// requireLoopback 防止本地诊断/进程控制接口在误配置为非回环监听时暴露给局域网。
func requireLoopback(w http.ResponseWriter, r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	if ip != nil && ip.IsLoopback() {
		return true
	}
	http.Error(w, "local operations API only accepts loopback requests", http.StatusForbidden)
	return false
}
