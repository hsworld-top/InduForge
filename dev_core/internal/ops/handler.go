package ops

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/indu-forge/dev_core/internal/auth"
	platformapi "github.com/indu-forge/dev_core/internal/platform/api"
)

type Handler struct {
	service *Service
	auth    *auth.Service
}

func NewHandler(service *Service, authService *auth.Service) *Handler {
	return &Handler{service: service, auth: authService}
}
func (h *Handler) MountRoutes(r chi.Router) {
	r.Route("/api/v1/ops", func(r chi.Router) {
		r.Get("/runtime-clusters", h.listClusters)
		r.Post("/runtime-clusters", h.createCluster)
		r.Get("/runtime-clusters/{id}", h.getCluster)
		r.Get("/node-enrollments", h.listEnrollments)
		r.Post("/node-enrollments", h.createEnrollment)
		r.Get("/node-enrollments/{id}", h.getEnrollment)
		r.Post("/node-enrollments/{id}/approve", h.approveEnrollment)
		r.Post("/node-enrollments/{id}/reject", h.rejectEnrollment)
		r.Get("/host-nodes", h.listNodes)
		r.Get("/host-nodes/{id}", h.getNode)
		r.Get("/project-deployments", h.listDeployments)
		r.Post("/project-deployments", h.createDeployment)
		r.Get("/project-deployments/{id}", h.getDeployment)
		r.Post("/project-deployments/{id}/workloads/{role}/start", h.operate("start"))
		r.Post("/project-deployments/{id}/workloads/{role}/stop", h.operate("stop"))
		r.Post("/project-deployments/{id}/workloads/{role}/restart", h.operate("restart"))
		r.Get("/deployment-runs/{id}", h.getRun)
		r.Get("/deployment-runs/{id}/events", h.listRunEvents)
		r.Get("/node-packages", h.listPackages)
		r.Get("/node-packages/{id}/download", h.downloadPackage)
		r.Post("/agent/enrollments/claim", h.claim)
		r.Post("/agent/host-nodes/{id}/heartbeat", h.heartbeat)
		r.Get("/agent/host-nodes/{id}/commands", h.commands)
	})
}
func (h *Handler) actor(w http.ResponseWriter, r *http.Request) (auth.User, bool) {
	if u, ok := auth.UserFromContext(r.Context()); ok {
		return u, true
	}
	token := auth.AccessTokenFromRequest(r)
	if token == "" {
		platformapi.WriteError(w, r, http.StatusUnauthorized, platformapi.ErrorCodeTokenRequired, "缺少访问令牌")
		return auth.User{}, false
	}
	u, e := h.auth.Authenticate(r.Context(), token)
	if e != nil {
		platformapi.WriteError(w, r, http.StatusUnauthorized, platformapi.ErrorCodeTokenInvalid, "访问令牌无效或已过期")
		return auth.User{}, false
	}
	return u, true
}
func (h *Handler) user(w http.ResponseWriter, r *http.Request, fn func(auth.User)) {
	if u, ok := h.actor(w, r); ok {
		fn(u)
	}
}
func (h *Handler) listClusters(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		items, total, e := h.service.ListClusters(r.Context(), u, page(r))
		if e != nil {
			h.err(w, r, e)
			return
		}
		out := make([]any, 0, len(items))
		for _, x := range items {
			out = append(out, clusterPayload(x))
		}
		platformapi.WriteSuccess(w, r, pageData(out, total, page(r)))
	})
}
func (h *Handler) createCluster(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		var in CreateClusterInput
		if !decode(r, &in) {
			h.invalid(w, r)
			return
		}
		x, e := h.service.CreateCluster(r.Context(), u, in)
		if e != nil {
			h.err(w, r, e)
			return
		}
		platformapi.WriteSuccess(w, r, clusterPayload(x))
	})
}
func (h *Handler) getCluster(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		x, e := h.service.GetCluster(r.Context(), u, chi.URLParam(r, "id"))
		if e != nil {
			h.err(w, r, e)
			return
		}
		platformapi.WriteSuccess(w, r, clusterPayload(x))
	})
}
func (h *Handler) listEnrollments(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		items, total, e := h.service.ListEnrollments(r.Context(), u, page(r))
		if e != nil {
			h.err(w, r, e)
			return
		}
		out := make([]any, 0, len(items))
		for _, x := range items {
			out = append(out, enrollmentPayload(x))
		}
		platformapi.WriteSuccess(w, r, pageData(out, total, page(r)))
	})
}
func (h *Handler) createEnrollment(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		var body struct {
			RuntimeClusterID, Role, DisplayName string
			TTLMinutes                          int `json:"ttlMinutes"`
		}
		if !decode(r, &body) {
			h.invalid(w, r)
			return
		}
		x, code, e := h.service.CreateEnrollment(r.Context(), u, CreateEnrollmentInput{RuntimeClusterID: body.RuntimeClusterID, Role: body.Role, DisplayName: body.DisplayName, TTL: time.Duration(body.TTLMinutes) * time.Minute})
		if e != nil {
			h.err(w, r, e)
			return
		}
		platformapi.WriteSuccess(w, r, map[string]any{"enrollment": enrollmentPayload(x), "code": code})
	})
}
func (h *Handler) getEnrollment(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		x, e := h.service.GetEnrollment(r.Context(), u, chi.URLParam(r, "id"))
		if e != nil {
			h.err(w, r, e)
			return
		}
		platformapi.WriteSuccess(w, r, enrollmentPayload(x))
	})
}
func (h *Handler) approveEnrollment(w http.ResponseWriter, r *http.Request) {
	h.changeEnrollment(w, r, true)
}
func (h *Handler) rejectEnrollment(w http.ResponseWriter, r *http.Request) {
	h.changeEnrollment(w, r, false)
}
func (h *Handler) changeEnrollment(w http.ResponseWriter, r *http.Request, approve bool) {
	h.user(w, r, func(u auth.User) {
		x, e := h.service.ApproveEnrollment(r.Context(), u, chi.URLParam(r, "id"), approve)
		if e != nil {
			h.err(w, r, e)
			return
		}
		platformapi.WriteSuccess(w, r, enrollmentPayload(x))
	})
}
func (h *Handler) listNodes(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		items, total, e := h.service.ListHostNodes(r.Context(), u, page(r))
		if e != nil {
			h.err(w, r, e)
			return
		}
		out := make([]any, 0, len(items))
		for _, x := range items {
			out = append(out, nodePayload(x))
		}
		platformapi.WriteSuccess(w, r, pageData(out, total, page(r)))
	})
}
func (h *Handler) getNode(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		x, e := h.service.GetHostNode(r.Context(), u, chi.URLParam(r, "id"))
		if e != nil {
			h.err(w, r, e)
			return
		}
		platformapi.WriteSuccess(w, r, nodePayload(x))
	})
}
func (h *Handler) listDeployments(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		items, total, e := h.service.ListDeployments(r.Context(), u, page(r))
		if e != nil {
			h.err(w, r, e)
			return
		}
		out := make([]any, 0, len(items))
		for _, x := range items {
			out = append(out, deploymentPayload(x))
		}
		platformapi.WriteSuccess(w, r, pageData(out, total, page(r)))
	})
}
func (h *Handler) createDeployment(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		var in CreateDeploymentInput
		if !decode(r, &in) {
			h.invalid(w, r)
			return
		}
		d, run, e := h.service.CreateDeployment(r.Context(), u, in)
		if e != nil {
			h.err(w, r, e)
			return
		}
		platformapi.WriteSuccess(w, r, map[string]any{"deployment": deploymentPayload(d), "run": runPayload(run)})
	})
}
func (h *Handler) getDeployment(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		x, e := h.service.GetDeployment(r.Context(), u, chi.URLParam(r, "id"))
		if e != nil {
			h.err(w, r, e)
			return
		}
		platformapi.WriteSuccess(w, r, deploymentPayload(x))
	})
}
func (h *Handler) operate(operation string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.user(w, r, func(u auth.User) {
			d, run, e := h.service.OperateWorkload(r.Context(), u, chi.URLParam(r, "id"), chi.URLParam(r, "role"), operation)
			if e != nil {
				h.err(w, r, e)
				return
			}
			platformapi.WriteSuccess(w, r, map[string]any{"deployment": deploymentPayload(d), "run": runPayload(run)})
		})
	}
}
func (h *Handler) getRun(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		x, e := h.service.GetRun(r.Context(), u, chi.URLParam(r, "id"))
		if e != nil {
			h.err(w, r, e)
			return
		}
		events, e := h.service.ListRunEvents(r.Context(), u, chi.URLParam(r, "id"))
		if e != nil {
			h.err(w, r, e)
			return
		}
		payload := runPayload(x)
		payload["events"] = events
		platformapi.WriteSuccess(w, r, payload)
	})
}
func (h *Handler) listRunEvents(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		items, e := h.service.ListRunEvents(r.Context(), u, chi.URLParam(r, "id"))
		if e != nil {
			h.err(w, r, e)
			return
		}
		platformapi.WriteSuccess(w, r, map[string]any{"items": items})
	})
}
func (h *Handler) listPackages(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		if err := auth.RequireCapability(u, auth.CapabilityNodeRead); err != nil {
			h.err(w, r, err)
			return
		}
		platformapi.WriteSuccess(w, r, map[string]any{"items": h.service.ListPackages()})
	})
}
func (h *Handler) downloadPackage(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		if err := auth.RequireCapability(u, auth.CapabilityNodeRead); err != nil {
			h.err(w, r, err)
			return
		}
		p, path, e := h.service.OpenPackage(chi.URLParam(r, "id"))
		if e != nil {
			platformapi.WriteError(w, r, http.StatusNotFound, platformapi.ErrorCodeNotFound, e.Error())
			return
		}
		w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote(p.FileName))
		http.ServeFile(w, r, path)
	})
}
func (h *Handler) claim(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Code  string                                                                                  `json:"code"`
		Host  struct{ DisplayName, Hostname, OS, Architecture, MachineFingerprint, IPAddress string } `json:"host"`
		Agent struct {
			Version      string
			Capabilities map[string]any
		} `json:"agent"`
	}
	if !decode(r, &body) {
		h.invalid(w, r)
		return
	}
	e, n, token, err := h.service.ClaimEnrollment(r.Context(), ClaimEnrollmentInput{Code: body.Code, DisplayName: body.Host.DisplayName, Hostname: body.Host.Hostname, OS: body.Host.OS, Architecture: body.Host.Architecture, MachineFingerprint: body.Host.MachineFingerprint, IPAddress: body.Host.IPAddress, AgentVersion: body.Agent.Version, Capabilities: body.Agent.Capabilities})
	if err != nil {
		h.err(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]any{"enrollment": enrollmentPayload(e), "hostNode": nodePayload(n), "reportedHostName": n.Hostname, "machineFingerprint": n.MachineFingerprint, "ipAddress": n.IPAddress, "agentToken": token, "pendingApproval": true})
}
func (h *Handler) heartbeat(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ResourceSummary map[string]any `json:"resourceSummary"`
		AgentVersion    string         `json:"agentVersion"`
		ObservedState   struct {
			Workloads []WorkloadObservation `json:"workloads"`
		} `json:"observedState"`
	}
	if !decode(r, &body) {
		h.invalid(w, r)
		return
	}
	n, commands, e := h.service.Heartbeat(r.Context(), chi.URLParam(r, "id"), agentToken(r), HeartbeatInput{ResourceSummary: body.ResourceSummary, AgentVersion: body.AgentVersion, Workloads: body.ObservedState.Workloads})
	if e != nil {
		h.err(w, r, e)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]any{"hostNode": nodePayload(n), "commands": commands})
}
func (h *Handler) commands(w http.ResponseWriter, r *http.Request) {
	items, e := h.service.AgentCommands(r.Context(), chi.URLParam(r, "id"), agentToken(r))
	if e != nil {
		h.err(w, r, e)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]any{"commands": items})
}
func agentToken(r *http.Request) string {
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
		return parts[1]
	}
	return ""
}
func page(r *http.Request) PageFilter {
	q := r.URL.Query()
	p, _ := strconv.Atoi(q.Get("page"))
	n, _ := strconv.Atoi(q.Get("pageSize"))
	search := q.Get("search")
	if search == "" {
		search = q.Get("keyword")
	}
	return PageFilter{Page: p, PageSize: n, Search: search}
}
func pageData(items any, total int64, f PageFilter) map[string]any {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 {
		f.PageSize = 20
	}
	return map[string]any{"items": items, "total": total, "page": f.Page, "pageSize": f.PageSize}
}
func decode(r *http.Request, target any) bool {
	return json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(target) == nil
}
func (h *Handler) invalid(w http.ResponseWriter, r *http.Request) {
	platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeInvalidRequest, "请求参数无效")
}
func (h *Handler) err(w http.ResponseWriter, r *http.Request, e error) {
	if errors.Is(e, auth.ErrPermissionDenied) {
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodePermissionDenied, e.Error())
		return
	}
	if errors.Is(e, ErrNotFound) || errors.Is(e, ErrEnrollmentUnavailable) {
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeNotFound, e.Error())
		return
	}
	if errors.Is(e, ErrDeploymentBusy) || errors.Is(e, ErrRuntimeClusterSelectionRequired) {
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeInvalidRequest, e.Error())
		return
	}
	if errors.Is(e, ErrDeploymentExists) {
		platformapi.WriteError(w, r, http.StatusConflict, platformapi.ErrorCodeAlreadyExists, e.Error())
		return
	}
	if errors.Is(e, ErrAgentUnauthorized) {
		platformapi.WriteError(w, r, http.StatusUnauthorized, platformapi.ErrorCodeTokenInvalid, e.Error())
		return
	}
	if strings.Contains(e.Error(), "不能为空") || strings.Contains(e.Error(), "不支持") || strings.Contains(e.Error(), "仅支持") || strings.Contains(e.Error(), "必须") || strings.Contains(e.Error(), "不能") || strings.Contains(e.Error(), "超过") || strings.Contains(e.Error(), "不可调度") || strings.Contains(e.Error(), "格式无效") {
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeInvalidRequest, e.Error())
		return
	}
	platformapi.WriteError(w, r, http.StatusInternalServerError, platformapi.ErrorCodeInternal, "系统内部错误")
}
func clusterPayload(x RuntimeCluster) map[string]any {
	return map[string]any{"id": x.ID, "tenantId": x.TenantID, "name": x.Name, "code": x.Code, "description": x.Description, "topology": x.Topology, "desiredStatus": x.DesiredStatus, "observedStatus": x.ObservedStatus, "controllerStatus": x.ControllerStatus, "nodeCount": x.NodeCount, "onlineNodeCount": x.OnlineNodeCount, "health": normalizeHealth(x.Health), "metadata": x.Metadata, "createdAt": x.CreatedAt, "updatedAt": x.UpdatedAt}
}
func enrollmentPayload(x Enrollment) map[string]any {
	payload := map[string]any{"id": x.ID, "runtimeClusterId": x.RuntimeClusterID, "role": x.Role, "displayName": x.DisplayName, "status": x.Status, "expiresAt": x.ExpiresAt, "claimedAt": x.ClaimedAt, "claimedByNodeId": x.ClaimedByNodeID, "reportedHostName": x.ReportedHostName, "machineFingerprint": x.MachineFingerprint, "ipAddress": x.IPAddress, "approvedAt": x.ApprovedAt, "rejectedAt": x.RejectedAt}
	if x.Node != nil {
		payload["node"] = nodePayload(*x.Node)
	}
	return payload
}
func nodePayload(x HostNode) map[string]any {
	status := x.ObservedStatus
	if status == "online" && x.LastHeartbeatAt != nil && time.Since(*x.LastHeartbeatAt) > 45*time.Second {
		status = "offline"
	}
	metrics := map[string]any{
		"cpuPercent":    resourceMetric(x.ResourceSummary, "cpu"),
		"memoryPercent": resourceMetric(x.ResourceSummary, "memory"),
		"diskPercent":   resourceMetric(x.ResourceSummary, "disk"),
	}
	return map[string]any{"id": x.ID, "runtimeClusterId": x.RuntimeClusterID, "runtimeClusterName": x.RuntimeClusterName, "enrollmentId": x.EnrollmentID, "role": x.Role, "displayName": x.DisplayName, "name": x.DisplayName, "hostname": x.Hostname, "os": x.OS, "architecture": x.Architecture, "machineFingerprint": x.MachineFingerprint, "ipAddress": x.IPAddress, "agentVersion": x.AgentVersion, "desiredStatus": x.DesiredStatus, "observedStatus": status, "health": normalizeNodeHealth(status), "resourceSummary": x.ResourceSummary, "metrics": metrics, "capabilities": x.Capabilities, "lastHeartbeatAt": x.LastHeartbeatAt, "approvedAt": x.ApprovedAt}
}

func resourceMetric(summary map[string]any, section string) any {
	values, ok := summary[section].(map[string]any)
	if !ok {
		return nil
	}
	return values["usedPercent"]
}

func normalizeNodeHealth(status string) string {
	switch status {
	case "online":
		return "healthy"
	case "degraded":
		return "degraded"
	case "offline", "revoked":
		return "unavailable"
	default:
		return "unknown"
	}
}
func deploymentPayload(x ProjectDeployment) map[string]any {
	return map[string]any{"id": x.ID, "projectId": x.ProjectID, "projectName": x.ProjectName, "runtimeClusterId": x.RuntimeClusterID, "runtimeClusterName": x.RuntimeClusterName, "latestRunId": x.LatestRunID, "version": x.Version, "deploymentMode": x.DeploymentMode, "desiredStatus": x.DesiredStatus, "observedStatus": x.ObservedStatus, "status": x.ObservedStatus, "health": normalizeHealth(x.Health), "progress": x.Progress, "workloads": x.Workloads, "createdAt": x.CreatedAt, "updatedAt": x.UpdatedAt}
}
func normalizeHealth(value string) string {
	switch value {
	case "healthy", "degraded", "unavailable", "unknown", "maintenance":
		return value
	case "offline", "failed":
		return "unavailable"
	default:
		return "unknown"
	}
}
func runPayload(x DeploymentRun) map[string]any {
	return map[string]any{"id": x.ID, "deploymentId": x.ProjectDeploymentID, "operation": x.Operation, "desiredStatus": x.DesiredStatus, "observedStatus": x.ObservedStatus, "status": x.ObservedStatus, "progress": x.Progress, "message": x.Message, "startedAt": x.StartedAt, "completedAt": x.CompletedAt}
}
