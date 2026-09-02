package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/indu-forge/runtime-api/internal/artifact"
	runtimeauth "github.com/indu-forge/runtime-api/internal/auth"
	"github.com/indu-forge/runtime-api/internal/realtime"
	"github.com/indu-forge/runtime-api/internal/runtimeview"
)

type Config struct {
	DeploymentID  string
	ProjectID     string
	AccountID     string
	SiteID        string
	NodeID        string
	Version       string
	ExecutionForm string
	SecureCookies bool
	Catalog       *artifact.Catalog
	Store         runtimeview.Store
	Authorizer    *runtimeauth.Authorizer
	Realtime      *realtime.Hub
	Now           func() time.Time
	ManualEpoch   int64
	ManualWriter  interface {
		ReserveManualSequence(context.Context, string, int64) (int64, error)
	}
	Publisher interface {
		PublishRaw(context.Context, string, []byte) error
	}
	CommandStore     runtimeview.ComputeCommandStore
	CommandPublisher interface {
		PublishCommand(context.Context, string, []byte) error
	}
}

type Server struct {
	config    Config
	startedAt time.Time
	sessions  *sessionManager
	handler   http.Handler
}

type envelope struct {
	Code  int    `json:"code"`
	Msg   string `json:"msg"`
	Data  any    `json:"data"`
	ReqID string `json:"reqId"`
}

type principalContextKey struct{}

var canonicalUUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func New(config Config) (*Server, error) {
	if strings.TrimSpace(config.DeploymentID) == "" || strings.TrimSpace(config.ProjectID) == "" || strings.TrimSpace(config.AccountID) == "" {
		return nil, errors.New("deploymentId、projectId、accountId 不能为空")
	}
	if strings.TrimSpace(config.SiteID) == "" || strings.TrimSpace(config.NodeID) == "" || strings.TrimSpace(config.Version) == "" {
		return nil, errors.New("siteId、nodeId、version 不能为空")
	}
	if config.ExecutionForm != "native-linux" && config.ExecutionForm != "native-windows" && config.ExecutionForm != "k3s-workload" {
		return nil, errors.New("executionForm 不受支持")
	}
	if config.Catalog == nil || config.Store == nil || config.Authorizer == nil || config.Realtime == nil || config.ManualWriter == nil || config.Publisher == nil || config.CommandStore == nil || config.CommandPublisher == nil || config.ManualEpoch < 1 {
		return nil, errors.New("catalog、store、authorizer、realtime 不能为空")
	}
	if config.Catalog.ProjectID != config.ProjectID {
		return nil, errors.New("catalog projectId 与 Runtime API 配置不一致")
	}
	if config.Now == nil {
		config.Now = func() time.Time { return time.Now().UTC() }
	}
	server := &Server{config: config, startedAt: config.Now(), sessions: newSessionManager()}
	server.handler = server.routes()
	return server, nil
}

func (s *Server) Handler() http.Handler { return s.handler }

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /api/v1/status", s.status)
	mux.HandleFunc("POST /api/v1/runtime/session", s.createSession)
	mux.Handle("GET /api/v1/runtime/session", s.requirePrincipal(http.HandlerFunc(s.getSession)))
	mux.Handle("DELETE /api/v1/runtime/session", s.requirePrincipal(http.HandlerFunc(s.deleteSession)))
	mux.Handle("GET /api/v1/runtime/catalog", s.requirePrincipal(http.HandlerFunc(s.catalog)))
	mux.Handle("GET /api/v1/runtime/points/{path}", s.requirePrincipal(http.HandlerFunc(s.currentPoint)))
	mux.Handle("GET /api/v1/runtime/points/{path}/history", s.requirePrincipal(http.HandlerFunc(s.pointHistory)))
	mux.Handle("POST /api/v1/runtime/points/{path}/write", s.requirePrincipal(http.HandlerFunc(s.writePoint)))
	mux.Handle("GET /api/v1/runtime/alarms/current", s.requirePrincipal(http.HandlerFunc(s.currentAlarms)))
	mux.Handle("POST /api/v1/runtime/alarms/{id}/acknowledge", s.requirePrincipal(http.HandlerFunc(s.acknowledgeAlarm)))
	mux.Handle("GET /api/v1/runtime/computes", s.requirePrincipal(http.HandlerFunc(s.computes)))
	mux.Handle("POST /api/v1/runtime/computes/{id}/run", s.requirePrincipal(http.HandlerFunc(s.runCompute)))
	mux.Handle("GET /api/v1/runtime/compute-commands/{id}", s.requirePrincipal(http.HandlerFunc(s.computeCommandStatus)))
	mux.Handle("GET /ws/v1/points", s.requirePrincipal(http.HandlerFunc(s.pointSocket)))
	mux.Handle("GET /ws/v1/alarms", s.requirePrincipal(http.HandlerFunc(s.alarmSocket)))
	return s.withCommonHeaders(mux)
}

func (s *Server) withCommonHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.Header().Set("Cache-Control", "no-store")
		writer.Header().Set("Referrer-Policy", "same-origin")
		next.ServeHTTP(writer, request)
	})
}

func (s *Server) requirePrincipal(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("X-InduForge-Deployment-Id") != s.config.DeploymentID || request.Header.Get("X-InduForge-Project-Id") != s.config.ProjectID {
			writeError(writer, request, http.StatusForbidden, 40301, "工程入口身份不匹配")
			return
		}
		principal, err := s.authenticateRequest(request)
		if err != nil {
			writeError(writer, request, http.StatusUnauthorized, 40101, "运行会话无效或已过期")
			return
		}
		ctx := context.WithValue(request.Context(), principalContextKey{}, principal)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

func (s *Server) authenticateRequest(request *http.Request) (runtimeauth.Principal, error) {
	if strings.TrimSpace(request.Header.Get("Authorization")) != "" {
		return s.config.Authorizer.Authenticate(request, s.config.Now())
	}
	cookie, err := request.Cookie(sessionCookieName)
	if err != nil {
		return runtimeauth.Principal{}, err
	}
	return s.sessions.authenticate(cookie.Value, s.config.Now())
}

func (s *Server) createSession(writer http.ResponseWriter, request *http.Request) {
	if request.Header.Get("X-InduForge-Deployment-Id") != s.config.DeploymentID || request.Header.Get("X-InduForge-Project-Id") != s.config.ProjectID {
		writeError(writer, request, http.StatusForbidden, 40301, "工程入口身份不匹配")
		return
	}
	principal, err := s.config.Authorizer.Authenticate(request, s.config.Now())
	if err != nil {
		writeError(writer, request, http.StatusUnauthorized, 40101, "访问令牌无效或已过期")
		return
	}
	id, expiresAt, err := s.sessions.issue(principal, s.config.Now())
	if err != nil {
		writeError(writer, request, http.StatusInternalServerError, 50001, "无法创建运行会话")
		return
	}
	http.SetCookie(writer, sessionCookie(id, s.config.Now(), expiresAt, s.config.SecureCookies))
	writeOK(writer, request, map[string]any{"subjectId": principal.SubjectID, "roles": principal.Roles, "expiresAt": expiresAt})
}

func (s *Server) getSession(writer http.ResponseWriter, request *http.Request) {
	principal := principalFrom(request.Context())
	writeOK(writer, request, map[string]any{"subjectId": principal.SubjectID, "roles": principal.Roles})
}

func (s *Server) deleteSession(writer http.ResponseWriter, request *http.Request) {
	if cookie, err := request.Cookie(sessionCookieName); err == nil {
		s.sessions.revoke(cookie.Value)
	}
	http.SetCookie(writer, expiredSessionCookie(s.config.SecureCookies))
	writeOK(writer, request, map[string]any{"revoked": true})
}

func (s *Server) health(writer http.ResponseWriter, request *http.Request) {
	ctx, cancel := context.WithTimeout(request.Context(), time.Second)
	defer cancel()
	if err := s.config.Store.Ping(ctx); err != nil {
		writeJSON(writer, http.StatusServiceUnavailable, envelope{Code: 50031, Msg: "运行依赖不可用：PostgreSQL", Data: map[string]any{
			"status": "DOWN", "observedAt": s.config.Now(), "reasonCode": "postgres-unavailable",
		}, ReqID: requestID(request)})
		return
	}
	writeJSON(writer, http.StatusOK, envelope{Code: 0, Msg: "ok", Data: map[string]any{
		"status": "UP", "observedAt": s.config.Now(),
	}, ReqID: requestID(request)})
}

func (s *Server) status(writer http.ResponseWriter, request *http.Request) {
	now := s.config.Now()
	ctx, cancel := context.WithTimeout(request.Context(), time.Second)
	defer cancel()
	storeErr := s.config.Store.Ping(ctx)
	healthState := "HEALTHY"
	reasonCode := any(nil)
	if storeErr != nil {
		healthState, reasonCode = "UNAVAILABLE", "postgres-unavailable"
	}
	if !s.config.Realtime.Connected() {
		if storeErr == nil {
			healthState, reasonCode = "DEGRADED", "nats-unavailable"
		}
	}
	freshness := map[string]any{"state": "UNKNOWN", "lastBusinessEventAt": nil, "evaluatedAt": now}
	if lastEvent := s.config.Realtime.LastEventAt(); lastEvent != nil {
		state := "FRESH"
		if now.Sub(*lastEvent) > 2*time.Minute {
			state = "STALE"
		}
		freshness = map[string]any{"state": state, "lastBusinessEventAt": lastEvent, "evaluatedAt": now}
	}
	writeOK(writer, request, map[string]any{
		"schemaVersion":     "runtime-health-status.v1",
		"lifecycleState":    "RUNNING",
		"healthState":       healthState,
		"componentRole":     "runtime-api",
		"deploymentId":      s.config.DeploymentID,
		"projectId":         s.config.ProjectID,
		"accountId":         s.config.AccountID,
		"version":           s.config.Version,
		"executionForm":     s.config.ExecutionForm,
		"siteId":            s.config.SiteID,
		"nodeId":            s.config.NodeID,
		"processId":         os.Getpid(),
		"startedAt":         s.startedAt,
		"uptimeSeconds":     int64(now.Sub(s.startedAt).Seconds()),
		"businessFreshness": freshness,
		"reasonCode":        reasonCode,
		"lastError":         reasonCode,
		"observedAt":        now,
	})
}

func (s *Server) catalog(writer http.ResponseWriter, request *http.Request) {
	points := make([]map[string]any, 0)
	for _, point := range s.config.Catalog.Points() {
		points = append(points, pointContract(point))
	}
	writeOK(writer, request, map[string]any{
		"schemaVersion":  s.config.Catalog.SchemaVersion,
		"artifactDigest": s.config.Catalog.Digest,
		"points":         points,
		"computes":       s.config.Catalog.Computes(),
		"alarms":         s.config.Catalog.Alarms(),
	})
}

func pointContract(point artifact.Point) map[string]any {
	return map[string]any{
		"id": point.ID, "path": point.Path, "name": point.Name, "displayName": point.Name,
		"dataType": point.DataType, "sourceType": point.SourceType, "sourceId": point.SourceID,
		"status": point.Status, "unit": point.Unit, "precision": point.Precision,
		"defaultValue": point.DefaultValue, "tags": point.Tags, "attributes": point.Attributes,
		"capabilities": map[string]bool{"get": true, "read": true, "history": true, "subscribe": true, "set": point.SourceType == "manual.input"},
	}
}

func (s *Server) currentPoint(writer http.ResponseWriter, request *http.Request) {
	point, ok := s.config.Catalog.PointByPath(request.PathValue("path"))
	if !ok {
		writeError(writer, request, http.StatusNotFound, 40401, "数据点不存在")
		return
	}
	item, err := s.config.Store.Current(request.Context(), s.config.DeploymentID, point.ID)
	if errors.Is(err, runtimeview.ErrNotFound) {
		writeError(writer, request, http.StatusNotFound, 40401, "数据点尚无运行值")
		return
	}
	if err != nil {
		writeError(writer, request, http.StatusServiceUnavailable, 50031, "运行数据查询失败")
		return
	}
	writeOK(writer, request, map[string]any{
		"path": point.Path, "pointId": item.PointID, "value": item.Value, "quality": item.Quality,
		"sourceTimestamp": item.SourceTimestamp, "serverTimestamp": item.ServerTimestamp,
		"sequence": item.Sequence, "eventId": item.EventID, "updatedAt": item.UpdatedAt,
	})
}

func (s *Server) pointHistory(writer http.ResponseWriter, request *http.Request) {
	point, ok := s.config.Catalog.PointByPath(request.PathValue("path"))
	if !ok {
		writeError(writer, request, http.StatusNotFound, 40401, "数据点不存在")
		return
	}
	query, err := parseHistoryQuery(request)
	if err != nil {
		writeError(writer, request, http.StatusBadRequest, 40001, err.Error())
		return
	}
	items, err := s.config.Store.History(request.Context(), s.config.DeploymentID, point.ID, query)
	if err != nil {
		writeError(writer, request, http.StatusServiceUnavailable, 50031, "运行历史查询失败")
		return
	}
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, map[string]any{
			"path": point.Path, "pointId": item.PointID, "value": item.Value, "quality": item.Quality,
			"sourceTimestamp": item.SourceTimestamp, "serverTimestamp": item.ServerTimestamp,
			"sequence": item.Sequence, "eventId": item.EventID, "receivedAt": item.ReceivedAt,
		})
	}
	writeOK(writer, request, map[string]any{"items": result, "total": len(result)})
}

func parseHistoryQuery(request *http.Request) (runtimeview.HistoryQuery, error) {
	query := runtimeview.HistoryQuery{Limit: 100}
	if value := request.URL.Query().Get("limit"); value != "" {
		limit, err := strconv.Atoi(value)
		if err != nil || limit < 1 || limit > 1000 {
			return query, errors.New("limit 必须在 1 到 1000 之间")
		}
		query.Limit = limit
	}
	for key, target := range map[string]**time.Time{"from": &query.From, "to": &query.To} {
		if value := request.URL.Query().Get(key); value != "" {
			parsed, err := time.Parse(time.RFC3339, value)
			if err != nil {
				return query, fmt.Errorf("%s 必须是 RFC3339 时间", key)
			}
			parsed = parsed.UTC()
			*target = &parsed
		}
	}
	if query.From != nil && query.To != nil && !query.From.Before(*query.To) {
		return query, errors.New("from 必须早于 to")
	}
	return query, nil
}

func (s *Server) currentAlarms(writer http.ResponseWriter, request *http.Request) {
	limit := 100
	if value := request.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 1000 {
			writeError(writer, request, http.StatusBadRequest, 40001, "limit 必须在 1 到 1000 之间")
			return
		}
		limit = parsed
	}
	items, err := s.config.Store.AlarmStates(request.Context(), s.config.DeploymentID, limit)
	if err != nil {
		writeError(writer, request, http.StatusServiceUnavailable, 50031, "报警状态查询失败")
		return
	}
	writeOK(writer, request, map[string]any{"items": items, "total": len(items)})
}

// acknowledgeAlarm 只允许操作员确认当前 deployment 的活动报警，并将确认和审计写入同一事务。
func (s *Server) acknowledgeAlarm(writer http.ResponseWriter, request *http.Request) {
	principal := principalFrom(request.Context())
	if !hasAlarmAcknowledgeRole(principal.Roles) {
		writeError(writer, request, http.StatusForbidden, 40301, "当前身份无权确认报警")
		return
	}
	var body struct {
		ExpectedVersion int64  `json:"expectedVersion"`
		Comment         string `json:"comment"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(writer, request.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil || decoder.Decode(&struct{}{}) != io.EOF || !canonicalUUID.MatchString(request.PathValue("id")) || body.ExpectedVersion < 1 || utf8.RuneCountInString(body.Comment) > 2048 {
		writeError(writer, request, http.StatusBadRequest, 40001, "expectedVersion 或 comment 非法")
		return
	}
	acknowledgement, err := s.config.Store.AcknowledgeAlarm(request.Context(), s.config.DeploymentID, request.PathValue("id"), body.ExpectedVersion, principal.SubjectID, body.Comment, s.config.Now())
	switch {
	case errors.Is(err, runtimeview.ErrNotFound):
		writeError(writer, request, http.StatusNotFound, 40401, "报警不存在")
	case errors.Is(err, runtimeview.ErrAlarmVersionConflict):
		writeError(writer, request, http.StatusConflict, 40001, "报警状态已变更，请刷新后重试")
	case errors.Is(err, runtimeview.ErrAlarmNotActive):
		writeError(writer, request, http.StatusConflict, 40001, "报警不存在活动实例")
	case err != nil:
		writeError(writer, request, http.StatusServiceUnavailable, 50031, "报警确认失败")
	default:
		writeOK(writer, request, acknowledgement)
	}
}

func hasAlarmAcknowledgeRole(roles []string) bool {
	for _, role := range roles {
		if role == "admin" || role == "operator" {
			return true
		}
	}
	return false
}

func (s *Server) computes(writer http.ResponseWriter, request *http.Request) {
	writeOK(writer, request, map[string]any{"items": s.config.Catalog.Computes(), "total": len(s.config.Catalog.Computes())})
}

func (s *Server) runCompute(writer http.ResponseWriter, request *http.Request) {
	compute, ok := s.config.Catalog.ComputeByID(request.PathValue("id"))
	if !ok {
		writeError(writer, request, http.StatusNotFound, 40401, "计算单元不存在")
		return
	}
	if !compute.Enabled || !manualComputeTrigger(compute.Trigger) || !hasAlarmAcknowledgeRole(principalFrom(request.Context()).Roles) {
		writeError(writer, request, http.StatusForbidden, 40301, "当前身份无权手工运行该计算单元")
		return
	}
	var body struct {
		IdempotencyKey string `json:"idempotencyKey"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(writer, request.Body, 16<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil || decoder.Decode(&struct{}{}) != io.EOF || len(body.IdempotencyKey) < 16 || len(body.IdempotencyKey) > 128 {
		writeError(writer, request, http.StatusBadRequest, 40001, "idempotencyKey 非法")
		return
	}
	now := s.config.Now().UTC()
	commandID, err := newCommandUUID()
	if err != nil {
		writeError(writer, request, http.StatusInternalServerError, 50001, "无法创建计算命令")
		return
	}
	command, inserted, err := s.config.CommandStore.QueueComputeCommand(request.Context(), s.config.DeploymentID, runtimeview.ComputeCommand{CommandID: commandID, ComputeID: compute.ID, RequestedBy: principalFrom(request.Context()).SubjectID, RequestedAt: now, BindingEpoch: s.config.ManualEpoch, IdempotencyKey: body.IdempotencyKey, Status: "queued"})
	if err != nil {
		writeError(writer, request, http.StatusServiceUnavailable, 50031, "计算命令审计不可用")
		return
	}
	if !inserted {
		writeOK(writer, request, map[string]any{"accepted": true, "commandId": command.CommandID, "status": command.Status, "idempotent": true})
		return
	}
	subject := "compute.command." + s.config.DeploymentID
	payload, _ := json.Marshal(map[string]any{"schemaVersion": "compute-command.v1", "subject": subject, "deploymentId": s.config.DeploymentID, "accountId": s.config.AccountID, "commandId": command.CommandID, "computeId": compute.ID, "requestedBy": command.RequestedBy, "requestedAt": now.Format(time.RFC3339Nano), "bindingEpoch": s.config.ManualEpoch, "idempotencyKey": command.IdempotencyKey})
	if err := s.config.CommandPublisher.PublishCommand(request.Context(), subject, payload); err != nil {
		_ = s.config.CommandStore.SetComputeCommandStatus(context.Background(), s.config.DeploymentID, command.CommandID, "failed", "publish-failed")
		writeError(writer, request, http.StatusServiceUnavailable, 50031, "计算命令发布失败")
		return
	}
	writeOK(writer, request, map[string]any{"accepted": true, "commandId": command.CommandID, "status": "queued"})
}

func (s *Server) computeCommandStatus(writer http.ResponseWriter, request *http.Request) {
	command, err := s.config.CommandStore.ComputeCommandStatus(request.Context(), s.config.DeploymentID, request.PathValue("id"))
	if errors.Is(err, runtimeview.ErrNotFound) {
		writeError(writer, request, http.StatusNotFound, 40401, "计算命令不存在")
		return
	}
	if err != nil {
		writeError(writer, request, http.StatusServiceUnavailable, 50031, "计算命令审计不可用")
		return
	}
	writeOK(writer, request, command)
}

func manualComputeTrigger(raw json.RawMessage) bool {
	var trigger struct {
		Kind string `json:"kind"`
	}
	return json.Unmarshal(raw, &trigger) == nil && trigger.Kind == "manual"
}

func newCommandUUID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	hex := hex.EncodeToString(raw[:])
	return hex[:8] + "-" + hex[8:12] + "-" + hex[12:16] + "-" + hex[16:20] + "-" + hex[20:], nil
}

func (s *Server) writePoint(writer http.ResponseWriter, request *http.Request) {
	point, ok := s.config.Catalog.PointByPath(request.PathValue("path"))
	if !ok {
		writeError(writer, request, http.StatusNotFound, 40401, "数据点不存在")
		return
	}
	if point.SourceType != "manual.input" || !writeAllowed(point.RuntimePermissions.Write, principalFrom(request.Context()).Roles) {
		writeError(writer, request, http.StatusForbidden, 40301, "当前身份无权写入该数据点")
		return
	}
	var body struct {
		Value json.RawMessage `json:"value"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(writer, request.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil || len(body.Value) == 0 || !validManualValue(point.DataType, body.Value) {
		writeError(writer, request, http.StatusBadRequest, 40001, "value 与数据点类型不匹配")
		return
	}
	sequence, err := s.config.ManualWriter.ReserveManualSequence(request.Context(), s.config.DeploymentID, s.config.ManualEpoch)
	if err != nil {
		writeError(writer, request, http.StatusServiceUnavailable, 50031, "人工写入 producer fence 不可用")
		return
	}
	now := s.config.Now().UTC()
	id := manualEventID(s.config.DeploymentID, point.ID, s.config.ManualEpoch, sequence)
	event := map[string]any{"schemaVersion": "data.raw.v1", "subject": "data.raw." + point.ID, "eventId": id, "deploymentId": s.config.DeploymentID, "accountId": s.config.AccountID, "pointId": point.ID, "ownerId": "runtime-api", "epoch": s.config.ManualEpoch, "sequence": sequence, "value": json.RawMessage(body.Value), "quality": "good", "sourceTimestamp": now.Format(time.RFC3339Nano), "serverTimestamp": now.Format(time.RFC3339Nano), "receivedAt": now.Format(time.RFC3339Nano)}
	payload, _ := json.Marshal(event)
	if err = s.config.Publisher.PublishRaw(request.Context(), "data.raw."+point.ID, payload); err != nil {
		writeError(writer, request, http.StatusServiceUnavailable, 50031, "人工写入事件发布失败")
		return
	}
	writeOK(writer, request, map[string]any{"accepted": true, "path": point.Path, "pointId": point.ID, "eventId": id, "sequence": sequence})
}

func writeAllowed(grant artifact.WritePermission, roles []string) bool {
	for _, role := range roles {
		for _, denied := range grant.DenyRoles {
			if role == denied {
				return false
			}
		}
		for _, allowed := range grant.AllowRoles {
			if role == allowed {
				return true
			}
		}
	}
	return grant.Inherit
}
func validManualValue(dataType string, raw json.RawMessage) bool {
	var v any
	d := json.NewDecoder(strings.NewReader(string(raw)))
	d.UseNumber()
	if d.Decode(&v) != nil || d.Decode(&struct{}{}) != io.EOF {
		return false
	}
	switch dataType {
	case "bool":
		_, ok := v.(bool)
		return ok
	case "string":
		_, ok := v.(string)
		return ok
	case "int8", "int16", "int32", "int64", "uint8", "uint16", "uint32", "uint64", "float32", "float64":
		_, ok := v.(json.Number)
		return ok
	default:
		return false
	}
}
func manualEventID(deploymentID, pointID string, epoch, sequence int64) string {
	value := "data.raw.v1\x1f" + deploymentID + "\x1f" + pointID + "\x1fruntime-api\x1f" + strconv.FormatInt(epoch, 10) + "\x1f" + strconv.FormatInt(sequence, 10)
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func (s *Server) unsupportedAction(writer http.ResponseWriter, request *http.Request) {
	writeError(writer, request, http.StatusNotImplemented, 50031, "当前 Runtime V1 未启用受控写入/人工计算命令链路")
}

func principalFrom(ctx context.Context) runtimeauth.Principal {
	principal, _ := ctx.Value(principalContextKey{}).(runtimeauth.Principal)
	return principal
}

func writeOK(writer http.ResponseWriter, request *http.Request, data any) {
	writeJSON(writer, http.StatusOK, envelope{Code: 0, Msg: "ok", Data: data, ReqID: requestID(request)})
}

func writeError(writer http.ResponseWriter, request *http.Request, status, code int, message string) {
	writeJSON(writer, status, envelope{Code: code, Msg: message, Data: nil, ReqID: requestID(request)})
}

func writeJSON(writer http.ResponseWriter, status int, value envelope) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func requestID(request *http.Request) string {
	if existing := strings.TrimSpace(request.Header.Get("X-Request-Id")); existing != "" && len(existing) <= 128 {
		return existing
	}
	var value [12]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "req_unavailable"
	}
	return "req_" + hex.EncodeToString(value[:])
}
