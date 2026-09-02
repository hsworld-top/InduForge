package ops

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/indu-forge/dev_core/internal/auth"
	platformapi "github.com/indu-forge/dev_core/internal/platform/api"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Handler struct {
	service *Service
	auth    *auth.Service
}

func NewHandler(s *Service, a *auth.Service) *Handler { return &Handler{service: s, auth: a} }
func (h *Handler) MountRoutes(r chi.Router) {
	r.Route("/api/v1/ops", func(r chi.Router) {
		r.Get("/node-enrollments", h.listEnrollments)
		r.Post("/node-enrollments", h.createEnrollment)
		r.Get("/node-enrollments/{id}", h.getEnrollment)
		r.Post("/node-enrollments/{id}/approve", h.approve)
		r.Post("/node-enrollments/{id}/reject", h.reject)
		r.Get("/nodes", h.listNodes)
		r.Get("/nodes/{id}", h.getNode)
		r.Delete("/nodes/{id}", h.removeNode)
		r.Get("/runtime-environments", h.listRuntimeEnvironments)
		r.Post("/runtime-environments", h.createRuntimeEnvironment)
		r.Get("/runtime-environments/{id}", h.getRuntimeEnvironment)
		r.Patch("/runtime-environments/{id}", h.updateRuntimeEnvironment)
		r.Delete("/runtime-environments/{id}", h.deleteRuntimeEnvironment)
		r.Get("/runtime-environments/{id}/nodes", h.listRuntimeEnvironmentNodes)
		r.Post("/runtime-environments/{id}/nodes", h.addRuntimeEnvironmentNodes)
		r.Delete("/runtime-environments/{id}/nodes/{nodeId}", h.removeRuntimeEnvironmentNode)
		r.Get("/runtime-environments/{id}/events", h.listRuntimeEnvironmentEvents)
		r.Get("/runtime-environments/{id}/foundation-services", h.listRuntimeEnvironmentServices)
		r.Post("/runtime-environments/{id}/foundation-services/deploy", h.deployRuntimeEnvironmentFoundation)
		r.Post("/runtime-environments/{id}/foundation-services/migrate", h.migrateRuntimeEnvironmentFoundation)
		r.Get("/node-packages", h.listPackages)
		r.Get("/node-packages/{id}/download", h.download)
		r.Get("/project-deployments", h.listDeployments)
		r.Get("/project-deployments/development-requirements", h.developmentEngineRequirements)
		r.Post("/project-deployments", h.createDeployment)
		r.Get("/project-deployments/{id}", h.getDeployment)
		r.Post("/project-deployments/{id}/start", h.operateDeployment("start"))
		r.Post("/project-deployments/{id}/stop", h.operateDeployment("stop"))
		r.Post("/project-deployments/{id}/restart", h.operateDeployment("restart"))
		r.Post("/project-deployments/{id}/services/{service}/start", h.operate("start"))
		r.Post("/project-deployments/{id}/services/{service}/stop", h.operate("stop"))
		r.Post("/project-deployments/{id}/services/{service}/restart", h.operate("restart"))
		r.Get("/deployment-runs/{id}", h.getRun)
		r.Get("/deployment-runs/{id}/events", h.events)
		r.Post("/agent/enrollments/claim", h.claim)
		r.Post("/agent/nodes/{id}/heartbeat", h.heartbeat)
		r.Get("/agent/nodes/{id}/commands", h.commands)
		r.Get("/agent/nodes/{id}/deployments/{deployment}/binding", h.agentBinding)
		r.Get("/agent/nodes/{id}/deployments/{deployment}/release", h.agentRelease)
	})
}
func (h *Handler) actor(w http.ResponseWriter, r *http.Request) (auth.User, bool) {
	if u, ok := auth.UserFromContext(r.Context()); ok {
		return u, true
	}
	token := auth.AccessTokenFromRequest(r)
	if token == "" {
		platformapi.WriteError(w, r, 401, platformapi.ErrorCodeTokenRequired, "缺少访问令牌")
		return auth.User{}, false
	}
	u, e := h.auth.Authenticate(r.Context(), token)
	if e != nil {
		platformapi.WriteError(w, r, 401, platformapi.ErrorCodeTokenInvalid, "访问令牌无效或已过期")
		return auth.User{}, false
	}
	return u, true
}
func (h *Handler) user(w http.ResponseWriter, r *http.Request, fn func(auth.User)) {
	if u, ok := h.actor(w, r); ok {
		fn(u)
	}
}
func (h *Handler) listEnrollments(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		x, n, e := h.service.ListEnrollments(r.Context(), u, page(r))
		if e != nil {
			h.err(w, r, e)
			return
		}
		out := []any{}
		for _, v := range x {
			out = append(out, enrollmentPayload(v))
		}
		platformapi.WriteSuccess(w, r, pageData(out, n, page(r)))
	})
}
func (h *Handler) createEnrollment(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		var x CreateEnrollmentInput
		var body struct {
			Platform, DisplayName string
			Capabilities          []string
			TTLMinutes            int `json:"ttlMinutes"`
		}
		if !decode(r, &body) {
			h.invalid(w, r)
			return
		}
		x = CreateEnrollmentInput{Platform: body.Platform, DisplayName: body.DisplayName, Capabilities: body.Capabilities, TTL: time.Duration(body.TTLMinutes) * time.Minute}
		e, code, err := h.service.CreateEnrollment(r.Context(), u, x)
		if err != nil {
			h.err(w, r, err)
			return
		}
		platformapi.WriteSuccess(w, r, map[string]any{"enrollment": enrollmentPayload(e), "code": code})
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
func (h *Handler) approve(w http.ResponseWriter, r *http.Request) { h.change(w, r, true) }
func (h *Handler) reject(w http.ResponseWriter, r *http.Request)  { h.change(w, r, false) }
func (h *Handler) change(w http.ResponseWriter, r *http.Request, ok bool) {
	h.user(w, r, func(u auth.User) {
		x, e := h.service.ApproveEnrollment(r.Context(), u, chi.URLParam(r, "id"), ok)
		if e != nil {
			h.err(w, r, e)
			return
		}
		platformapi.WriteSuccess(w, r, enrollmentPayload(x))
	})
}
func (h *Handler) listNodes(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		x, n, e := h.service.ListNodes(r.Context(), u, page(r))
		if e != nil {
			h.err(w, r, e)
			return
		}
		out := []any{}
		for _, v := range x {
			out = append(out, nodePayload(v))
		}
		platformapi.WriteSuccess(w, r, pageData(out, n, page(r)))
	})
}
func (h *Handler) getNode(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		x, e := h.service.GetNode(r.Context(), u, chi.URLParam(r, "id"))
		if e != nil {
			h.err(w, r, e)
			return
		}
		platformapi.WriteSuccess(w, r, nodePayload(x))
	})
}
func (h *Handler) removeNode(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		if e := h.service.RemoveNode(r.Context(), u, chi.URLParam(r, "id")); e != nil {
			h.err(w, r, e)
			return
		}
		platformapi.WriteSuccess(w, r, map[string]any{"status": "removing"})
	})
}
func (h *Handler) listDeployments(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		x, n, e := h.service.ListDeployments(r.Context(), u, page(r))
		if e != nil {
			h.err(w, r, e)
			return
		}
		out := []any{}
		for _, v := range x {
			out = append(out, deploymentPayload(v))
		}
		platformapi.WriteSuccess(w, r, pageData(out, n, page(r)))
	})
}
func (h *Handler) createDeployment(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		var x CreateDeploymentInput
		if !decode(r, &x) {
			h.invalid(w, r)
			return
		}
		x.Authorization = auth.ForwardAuthorization(r)
		d, run, e := h.service.CreateDeployment(r.Context(), u, x)
		if e != nil {
			h.err(w, r, e)
			return
		}
		platformapi.WriteSuccess(w, r, map[string]any{"deployment": deploymentPayload(d), "run": runPayload(run)})
	})
}
func (h *Handler) developmentEngineRequirements(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		requirements, err := h.service.DevelopmentEngineRequirements(r.Context(), u, r.URL.Query().Get("projectId"), auth.ForwardAuthorization(r))
		if err != nil {
			h.err(w, r, err)
			return
		}
		platformapi.WriteSuccess(w, r, map[string]any{"engines": requirements})
	})
}
func (h *Handler) getDeployment(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		d, e := h.service.GetDeployment(r.Context(), u, chi.URLParam(r, "id"))
		if e != nil {
			h.err(w, r, e)
			return
		}
		platformapi.WriteSuccess(w, r, deploymentPayload(d))
	})
}
func (h *Handler) operate(op string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.user(w, r, func(u auth.User) {
			d, run, e := h.service.OperateService(r.Context(), u, chi.URLParam(r, "id"), chi.URLParam(r, "service"), op)
			if e != nil {
				h.err(w, r, e)
				return
			}
			platformapi.WriteSuccess(w, r, map[string]any{"deployment": deploymentPayload(d), "run": runPayload(run)})
		})
	}
}
func (h *Handler) operateDeployment(op string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.user(w, r, func(u auth.User) {
			d, run, e := h.service.OperateDeployment(r.Context(), u, chi.URLParam(r, "id"), op)
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
		platformapi.WriteSuccess(w, r, runPayload(x))
	})
}
func (h *Handler) events(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		x, e := h.service.ListRunEvents(r.Context(), u, chi.URLParam(r, "id"))
		if e != nil {
			h.err(w, r, e)
			return
		}
		platformapi.WriteSuccess(w, r, map[string]any{"items": x})
	})
}
func (h *Handler) listPackages(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		if e := auth.RequireCapability(u, auth.CapabilityNodeRead); e != nil {
			h.err(w, r, e)
			return
		}
		platformapi.WriteSuccess(w, r, map[string]any{"items": h.service.ListPackages()})
	})
}
func (h *Handler) download(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		if e := auth.RequireCapability(u, auth.CapabilityNodeRead); e != nil {
			h.err(w, r, e)
			return
		}
		p, path, e := h.service.OpenPackage(chi.URLParam(r, "id"))
		if e != nil {
			h.err(w, r, e)
			return
		}
		w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote(p.FileName))
		http.ServeFile(w, r, path)
	})
}
func (h *Handler) claim(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Code               string   `json:"code"`
		DisplayName        string   `json:"displayName"`
		Hostname           string   `json:"hostname"`
		Platform           string   `json:"platform"`
		Architecture       string   `json:"architecture"`
		AgentVersion       string   `json:"agentVersion"`
		MachineFingerprint string   `json:"machineFingerprint"`
		IPAddress          string   `json:"ipAddress"`
		Capabilities       []string `json:"capabilities"`
	}
	if !decode(r, &body) {
		h.invalid(w, r)
		return
	}
	x := ClaimEnrollmentInput{
		Code: body.Code, DisplayName: body.DisplayName, Hostname: body.Hostname,
		Platform: body.Platform, Architecture: body.Architecture, AgentVersion: body.AgentVersion,
		MachineFingerprint: body.MachineFingerprint, IPAddress: remoteIPAddress(r), Capabilities: body.Capabilities,
	}
	e, n, t, err := h.service.ClaimEnrollment(r.Context(), x)
	if err != nil {
		h.err(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]any{"enrollment": enrollmentPayload(e), "node": nodePayload(n), "agentToken": t, "pendingApproval": true})
}

func remoteIPAddress(r *http.Request) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		host = strings.TrimSpace(r.RemoteAddr)
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.String()
	}
	return ""
}
func (h *Handler) heartbeat(w http.ResponseWriter, r *http.Request) {
	var x HeartbeatInput
	if !decode(r, &x) {
		h.invalid(w, r)
		return
	}
	n, s, e := h.service.Heartbeat(r.Context(), chi.URLParam(r, "id"), agentToken(r), x)
	if e != nil {
		h.err(w, r, e)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]any{"node": nodePayload(n), "services": s})
}
func (h *Handler) commands(w http.ResponseWriter, r *http.Request) {
	nodeID, token := chi.URLParam(r, "id"), agentToken(r)
	x, e := h.service.AgentCommands(r.Context(), nodeID, token)
	if e != nil {
		h.err(w, r, e)
		return
	}
	plan, e := h.service.AgentClusterPlan(r.Context(), nodeID, token)
	if e != nil {
		h.err(w, r, e)
		return
	}
	uninstall, e := h.service.AgentClusterUninstall(r.Context(), nodeID, token)
	if e != nil {
		h.err(w, r, e)
		return
	}
	foundationPlan, e := h.service.AgentFoundationPlan(r.Context(), nodeID, token)
	if e != nil {
		h.err(w, r, e)
		return
	}
	foundationDelete, e := h.service.AgentFoundationDelete(r.Context(), nodeID, token)
	if e != nil {
		h.err(w, r, e)
		return
	}
	timeSyncPlan, e := h.service.AgentTimeSyncPlan(r.Context(), nodeID, token)
	if e != nil {
		h.err(w, r, e)
		return
	}
	// 删除优先于部署，避免同一轮命令先重建再删除环境隔离空间。
	if foundationDelete != nil {
		foundationPlan = nil
	}
	platformapi.WriteSuccess(w, r, map[string]any{"commands": x, "clusterPlan": plan, "clusterUninstall": uninstall, "foundationPlan": foundationPlan, "foundationDelete": foundationDelete, "timeSyncPlan": timeSyncPlan})
}
func (h *Handler) agentBinding(w http.ResponseWriter, r *http.Request) {
	binding, err := h.service.AgentDeploymentBinding(r.Context(), chi.URLParam(r, "id"), agentToken(r), chi.URLParam(r, "deployment"), r.URL.Query().Get("serviceId"))
	if err != nil {
		h.err(w, r, err)
		return
	}
	var payload any
	if err := json.Unmarshal(binding.Content, &payload); err != nil {
		h.err(w, r, fmt.Errorf("%w: DeploymentBinding 内容无效", ErrReleaseNotDeployable))
		return
	}
	platformapi.WriteSuccess(w, r, payload)
}
func (h *Handler) agentRelease(w http.ResponseWriter, r *http.Request) {
	_, content, err := h.service.AgentRelease(r.Context(), chi.URLParam(r, "id"), agentToken(r), chi.URLParam(r, "deployment"), r.URL.Query().Get("serviceId"))
	if err != nil {
		h.err(w, r, err)
		return
	}
	defer content.Reader.Close()
	// Release 下载端点只交付固定的 tar.zst 契约，不能透传对象存储中可变的
	// Content-Type 元数据，避免错误或被污染的元数据影响 Agent 的下载边界。
	w.Header().Set("Content-Type", "application/zstd")
	w.Header().Set("Content-Length", strconv.FormatInt(content.Size, 10))
	w.Header().Set("Content-Disposition", `attachment; filename="release.tar.zst"`)
	if _, err := io.Copy(w, content.Reader); err != nil {
		return
	}
}
func agentToken(r *http.Request) string {
	p := strings.Fields(r.Header.Get("Authorization"))
	if len(p) == 2 && strings.EqualFold(p[0], "bearer") {
		return p[1]
	}
	return ""
}
func page(r *http.Request) PageFilter {
	p, _ := strconv.Atoi(r.URL.Query().Get("page"))
	n, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	return PageFilter{Page: p, PageSize: n, Search: r.URL.Query().Get("search"), ProjectID: r.URL.Query().Get("projectId"), Status: r.URL.Query().Get("status")}
}
func pageData(x any, n int64, f PageFilter) map[string]any {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 {
		f.PageSize = 20
	}
	return map[string]any{"items": x, "total": n, "page": f.Page, "pageSize": f.PageSize}
}
func decode(r *http.Request, x any) bool {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(x); err != nil {
		return false
	}
	return decoder.Decode(&struct{}{}) == io.EOF
}
func (h *Handler) invalid(w http.ResponseWriter, r *http.Request) {
	platformapi.WriteError(w, r, 200, platformapi.ErrorCodeInvalidRequest, "请求参数无效")
}
func (h *Handler) err(w http.ResponseWriter, r *http.Request, e error) {
	status, code := http.StatusInternalServerError, platformapi.ErrorCodeInternal
	if errors.Is(e, auth.ErrPermissionDenied) {
		status, code = 200, platformapi.ErrorCodePermissionDenied
	} else if errors.Is(e, ErrNotFound) || errors.Is(e, ErrEnrollmentUnavailable) {
		status, code = 200, platformapi.ErrorCodeNotFound
	} else if errors.Is(e, ErrAgentUnauthorized) {
		status, code = 401, platformapi.ErrorCodeTokenInvalid
	} else if errors.Is(e, ErrDeploymentExists) || errors.Is(e, ErrNodePortConflict) || errors.Is(e, ErrEnvironmentExists) || errors.Is(e, ErrNodeEnvironmentConflict) || errors.Is(e, ErrNodeEnvironmentInUse) || errors.Is(e, ErrNodeDeploymentInUse) || errors.Is(e, ErrEnvironmentNodeServiceInUse) || errors.Is(e, ErrEnvironmentNodeDeploymentInUse) || errors.Is(e, ErrCoordinatorInUse) || errors.Is(e, ErrCenterNodeProtected) || errors.Is(e, ErrDefaultEnvironmentProtected) || errors.Is(e, ErrEnvironmentHasDeployment) || errors.Is(e, ErrEnvironmentDeleting) || errors.Is(e, ErrFoundationExists) {
		status, code = 409, platformapi.ErrorCodeAlreadyExists
	} else if errors.Is(e, ErrReleaseNotDeployable) || errors.Is(e, ErrServiceLifecycleDisabled) || errors.Is(e, ErrNodeNotEligible) || errors.Is(e, ErrNodeOfflineForRemoval) || errors.Is(e, ErrFoundationNotReady) || errors.Is(e, ErrFoundationMoveUnsupported) {
		status, code = 200, platformapi.ErrorCodeInvalidRequest
	} else if errors.Is(e, ErrDeploymentBusy) || strings.Contains(e.Error(), "不支持") || strings.Contains(e.Error(), "无效") || strings.Contains(e.Error(), "不能为空") || strings.Contains(e.Error(), "不能") || strings.Contains(e.Error(), "能力") {
		status, code = 200, platformapi.ErrorCodeInvalidRequest
	}
	platformapi.WriteError(w, r, status, code, e.Error())
}
func enrollmentPayload(x Enrollment) map[string]any {
	var node any
	if x.Node != nil {
		node = nodePayload(*x.Node)
	}
	return map[string]any{
		"id":                 x.ID,
		"platform":           x.Platform,
		"capabilities":       x.Capabilities,
		"displayName":        x.DisplayName,
		"status":             x.Status,
		"expiresAt":          x.ExpiresAt,
		"claimedAt":          x.ClaimedAt,
		"claimedByNodeId":    x.ClaimedByNodeID,
		"reportedHostName":   x.ReportedHostName,
		"machineFingerprint": x.MachineFingerprint,
		"ipAddress":          x.IPAddress,
		"approvedAt":         x.ApprovedAt,
		"rejectedAt":         x.RejectedAt,
		"node":               node,
		"createdAt":          x.CreatedAt,
		"updatedAt":          x.UpdatedAt,
	}
}
func nodePayload(x Node) map[string]any {
	status := x.ObservedStatus
	if status == "online" && x.LastHeartbeatAt != nil && time.Since(*x.LastHeartbeatAt) > 45*time.Second {
		status = "offline"
	}
	return map[string]any{
		"id":                        x.ID,
		"enrollmentId":              x.EnrollmentID,
		"displayName":               x.DisplayName,
		"name":                      x.DisplayName,
		"hostname":                  x.Hostname,
		"platform":                  x.Platform,
		"architecture":              x.Architecture,
		"capabilities":              x.Capabilities,
		"agentVersion":              x.AgentVersion,
		"machineFingerprint":        x.MachineFingerprint,
		"ipAddress":                 x.IPAddress,
		"desiredStatus":             x.DesiredStatus,
		"observedStatus":            status,
		"resourceSummary":           x.ResourceSummary,
		"lastHeartbeatAt":           x.LastHeartbeatAt,
		"approvedAt":                x.ApprovedAt,
		"assignedDeploymentId":      x.AssignedDeploymentID,
		"assignedProjectId":         x.AssignedProjectID,
		"assignedProjectName":       x.AssignedProjectName,
		"environmentId":             x.EnvironmentID,
		"environmentName":           x.EnvironmentName,
		"environmentNames":          x.EnvironmentNames,
		"environmentCount":          x.EnvironmentCount,
		"nodeKind":                  x.NodeKind,
		"clusterId":                 x.ClusterID,
		"clusterRole":               x.ClusterRole,
		"clusterStatus":             x.ClusterStatus,
		"clusterMessage":            x.ClusterMessage,
		"clusterDesiredAction":      x.ClusterDesiredAction,
		"clusterDesiredGeneration":  x.ClusterDesiredGeneration,
		"clusterObservedGeneration": x.ClusterObservedGeneration,
		"clusterObservedAt":         x.ClusterObservedAt,
		"createdAt":                 x.CreatedAt,
		"updatedAt":                 x.UpdatedAt,
	}
}
func deploymentPayload(x ProjectDeployment) map[string]any {
	entryStatus := "pending"
	for _, service := range x.Services {
		if service.ServiceType == ServiceBase {
			entryStatus = service.ObservedStatus
			break
		}
	}
	accessURL := ""
	accessAvailable := false
	updating := x.LastReadyAt == nil
	nodeNames := make([]string, 0, len(x.Services))
	seenNodes := make(map[string]struct{}, len(x.Services))
	for _, service := range x.Services {
		if name := strings.TrimSpace(service.NodeName); name != "" {
			if _, seen := seenNodes[name]; !seen {
				seenNodes[name] = struct{}{}
				nodeNames = append(nodeNames, name)
			}
		}
		if service.ServiceType == ServiceBase {
			accessURL = service.Endpoint
			accessAvailable = service.ObservedStatus == "running" && service.DesiredStatus == "running"
			break
		}
		if service.DesiredStatus == "running" && service.DesiredGeneration != x.LastReadyGeneration {
			updating = true
		}
	}
	return map[string]any{
		"id":                   x.ID,
		"projectId":            x.ProjectID,
		"projectName":          x.ProjectName,
		"environmentId":        x.EnvironmentID,
		"environmentName":      x.EnvironmentName,
		"mode":                 x.Mode,
		"applicationVersionId": x.ApplicationVersionID,
		"version":              x.Version,
		"lastReadyVersion":     x.LastReadyVersion,
		"lastReadyAt":          x.LastReadyAt,
		"updating":             updating,
		"latestRunId":          x.LatestRunID,
		"accessPort":           x.AccessPort,
		"desiredStatus":        x.DesiredStatus,
		"observedStatus":       x.ObservedStatus,
		"health":               x.Health,
		"progress":             x.Progress,
		"services":             x.Services,
		"nodeNames":            nodeNames,
		"entryStatus":          entryStatus,
		"accessUrl":            accessURL,
		"accessAvailable":      accessAvailable,
		"createdAt":            x.CreatedAt,
		"updatedAt":            x.UpdatedAt,
	}
}
func runPayload(x DeploymentRun) map[string]any {
	return map[string]any{"id": x.ID, "deploymentId": x.ProjectDeploymentID, "operation": x.Operation, "desiredStatus": x.DesiredStatus, "observedStatus": x.ObservedStatus, "progress": x.Progress, "message": x.Message, "startedAt": x.StartedAt, "completedAt": x.CompletedAt}
}
func normalizeHealth(v string) string {
	if v == "running" {
		return "healthy"
	}
	if v == "failed" {
		return "unavailable"
	}
	if v == "degraded" {
		return "degraded"
	}
	return "unknown"
}
