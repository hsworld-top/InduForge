package auditlog

import (
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/indu-forge/dev_core/internal/auth"
	platformapi "github.com/indu-forge/dev_core/internal/platform/api"
)

const timeFormat = "2006-01-02 15:04:05"

type Handler struct {
	service     *Service
	authService *auth.Service
}

func NewHandler(service *Service, authService *auth.Service) *Handler {
	return &Handler{service: service, authService: authService}
}

func (h *Handler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.requireCapability(w, r, auth.CapabilityAuditLogRead)
	if !ok {
		return
	}
	filter, err := filterFromRequest(r)
	if err != nil {
		h.invalid(w, r, err)
		return
	}
	items, total, err := h.service.List(r.Context(), actor, filter)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	list := make([]map[string]any, 0, len(items))
	for _, item := range items {
		list = append(list, logPayload(item))
	}
	totalPages := 0
	if filter.Limit > 0 {
		totalPages = int((total + int64(filter.Limit) - 1) / int64(filter.Limit))
	}
	platformapi.WriteSuccess(w, r, map[string]any{
		"list":       map[string]any{"logs": list},
		"pagination": map[string]any{"total": total, "page": filter.Page, "limit": filter.Limit, "totalPages": totalPages},
	})
}

func (h *Handler) GetAuditLog(w http.ResponseWriter, r *http.Request, logID string) {
	actor, ok := h.requireCapability(w, r, auth.CapabilityAuditLogRead)
	if !ok {
		return
	}
	item, err := h.service.Get(r.Context(), actor, logID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, logPayload(item))
}

func (h *Handler) DeleteAuditLogs(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.requireCapability(w, r, auth.CapabilityAuditLogDelete)
	if !ok {
		return
	}
	before, err := parseTime(r.URL.Query().Get("beforeDate"))
	if err != nil || before == nil {
		h.invalid(w, r, fmt.Errorf("删除截止时间不能为空或格式无效"))
		return
	}
	count, err := h.service.DeleteBefore(r.Context(), actor, *before)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]any{"deleted": count})
}

func (h *Handler) ExportAuditLogs(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.requireCapability(w, r, auth.CapabilityAuditLogRead)
	if !ok {
		return
	}
	filter, err := filterFromRequest(r)
	if err != nil {
		h.invalid(w, r, err)
		return
	}
	items, err := h.service.Export(r.Context(), actor, filter)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="system-logs.csv"`)
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})
	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"时间", "级别", "用户", "操作", "资源", "消息", "结果", "请求ID", "请求路径", "IP"})
	for _, item := range items {
		_ = writer.Write([]string{item.CreatedAt.Format(timeFormat), item.Level, item.Username, item.Action, item.Resource, item.Message, item.Result, item.RequestID, item.Path, item.IP})
	}
	writer.Flush()
}

func (h *Handler) GetAuditLogStats(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.requireCapability(w, r, auth.CapabilityAuditLogRead)
	if !ok {
		return
	}
	start, err := parseTime(r.URL.Query().Get("startDate"))
	if err != nil {
		h.invalid(w, r, err)
		return
	}
	end, err := parseTime(r.URL.Query().Get("endDate"))
	if err != nil {
		h.invalid(w, r, err)
		return
	}
	stats, err := h.service.Stats(r.Context(), actor, start, end)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]any{"levelStats": stats.LevelStats, "trendStats": stats.TrendStats, "actionStats": stats.ActionStats})
}

func (h *Handler) ListRecentActivities(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	limit := positiveInt(r.URL.Query().Get("limit"), 5)
	items, err := h.service.Recent(r.Context(), actor, limit)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	list := make([]map[string]any, 0, len(items))
	for _, item := range items {
		list = append(list, activityPayload(item))
	}
	platformapi.WriteSuccess(w, r, map[string]any{"activities": list})
}

func activityPayload(item Log) map[string]any {
	return map[string]any{
		"id": item.ID, "action": item.Action, "resource": item.Resource, "path": item.Path,
		"createdAt": item.CreatedAt.Format(timeFormat),
		"user":      map[string]any{"id": item.UserID, "username": item.Username, "fullName": item.FullName},
	}
}

func (h *Handler) requireCapability(w http.ResponseWriter, r *http.Request, capability auth.Capability) (auth.User, bool) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return auth.User{}, false
	}
	if err := auth.RequireCapability(actor, capability); err != nil {
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodePermissionDenied, "无权访问系统日志")
		return auth.User{}, false
	}
	return actor, true
}

func (h *Handler) requireUser(w http.ResponseWriter, r *http.Request) (auth.User, bool) {
	if actor, ok := auth.UserFromContext(r.Context()); ok {
		return actor, true
	}
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		platformapi.WriteError(w, r, http.StatusUnauthorized, platformapi.ErrorCodeTokenRequired, "缺少访问令牌")
		return auth.User{}, false
	}
	actor, err := h.authService.Authenticate(r.Context(), parts[1])
	if err != nil {
		platformapi.WriteError(w, r, http.StatusUnauthorized, platformapi.ErrorCodeTokenInvalid, "访问令牌无效或已过期")
		return auth.User{}, false
	}
	return actor, true
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, ErrNotFound) {
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeNotFound, err.Error())
		return
	}
	platformapi.WriteError(w, r, http.StatusInternalServerError, platformapi.ErrorCodeInternal, "系统内部错误")
}

func (h *Handler) invalid(w http.ResponseWriter, r *http.Request, err error) {
	platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeInvalidRequest, err.Error())
}

func filterFromRequest(r *http.Request) (Filter, error) {
	query := r.URL.Query()
	start, err := parseTime(query.Get("startDate"))
	if err != nil {
		return Filter{}, err
	}
	end, err := parseTime(query.Get("endDate"))
	if err != nil {
		return Filter{}, err
	}
	return Filter{
		Page: positiveInt(query.Get("page"), 1), Limit: positiveInt(query.Get("limit"), positiveInt(query.Get("size"), 20)),
		Level: query.Get("level"), Action: query.Get("action"), Resource: query.Get("resource"), UserID: query.Get("userId"),
		Keyword: query.Get("keyword"), StartTime: start, EndTime: end,
	}, nil
}

func parseTime(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		location = time.Local
	}
	for _, layout := range []string{timeFormat, time.RFC3339, "2006-01-02"} {
		value, parseErr := time.ParseInLocation(layout, raw, location)
		if parseErr == nil {
			return &value, nil
		}
	}
	return nil, fmt.Errorf("时间格式无效")
}

func logPayload(item Log) map[string]any {
	return map[string]any{
		"id": item.ID, "tenantId": item.TenantID, "userId": item.UserID, "level": item.Level,
		"action": item.Action, "resource": item.Resource, "resourceId": item.ResourceID, "message": item.Message,
		"requestId": item.RequestID, "method": item.Method, "path": item.Path, "result": item.Result,
		"ip": item.IP, "userAgent": item.UserAgent, "metadata": item.Metadata, "createdAt": item.CreatedAt.Format(timeFormat),
		"user": map[string]any{"id": item.UserID, "username": item.Username, "fullName": item.FullName},
	}
}

func positiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
