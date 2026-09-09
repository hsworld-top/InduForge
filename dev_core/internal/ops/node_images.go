package ops

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/indu-forge/dev_core/internal/imagecatalog"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type ImagePlan struct {
	Artifacts []imagecatalog.Artifact `json:"artifacts"`
}
type ImageState struct {
	Status    string   `json:"status"`
	Artifacts []string `json:"artifacts"`
	Message   string   `json:"message"`
}
type imagePreparationPending struct{ message string }

func (e *imagePreparationPending) Error() string                        { return e.message }
func (r *PostgreSQLRepository) SetImageCatalog(c *imagecatalog.Catalog) { r.imageCatalog = c }
func (s *Service) SetImageCatalog(c *imagecatalog.Catalog)              { s.imageCatalog = c }
func projectImageReferences(engine string) []string {
	switch engine {
	case ServiceCollector:
		return nil
	case ServiceBase:
		return []string{runtimeEngineImage, projectGatewayImage, runtimeAPIImage}
	case ServiceCompute:
		return []string{runtimeEngineImage, computeSandboxImage}
	default:
		return []string{runtimeEngineImage}
	}
}
func foundationImageReferences(service string) []string {
	switch foundationWorkloadForService(service) {
	case "postgres":
		return []string{"timescale/timescaledb:2.26.4-pg16"}
	case "redis":
		return []string{"redis:7.2-alpine"}
	case "emqx":
		return []string{"emqx/emqx:5.6.1", "nginx:1.28-alpine"}
	case "nats":
		return []string{"nats:2.12.8-alpine"}
	case "object":
		return []string{"chrislusf/seaweedfs:3.85"}
	case "nginx":
		return []string{"nginx:1.28-alpine"}
	}
	return nil
}
func (r *PostgreSQLRepository) imagePlan(ctx context.Context, n Node) (*ImagePlan, error) {
	if r.imageCatalog == nil {
		return nil, fmt.Errorf("中心镜像目录未配置")
	}
	rows, e := r.pool.Query(ctx, `SELECT s.service_type,false FROM deployment_services s JOIN project_deployments d ON d.id=s.project_deployment_id WHERE s.node_id=$1 AND s.desired_status='running' AND d.deleted_at IS NULL UNION SELECT s.service_type,true FROM runtime_environment_services s JOIN runtime_environments e ON e.id=s.environment_id WHERE s.node_id=$1 AND e.deleted_at IS NULL AND e.desired_status<>'deleting'`, n.ID)
	if e != nil {
		return nil, e
	}
	var refs []string
	for rows.Next() {
		var service string
		var foundation bool
		if e = rows.Scan(&service, &foundation); e != nil {
			rows.Close()
			return nil, e
		}
		if foundation {
			refs = append(refs, foundationImageReferences(service)...)
		} else {
			refs = append(refs, projectImageReferences(service)...)
		}
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	if len(refs) == 0 {
		return nil, nil
	}
	artifacts, e := r.imageCatalog.Resolve(ctx, n.Architecture, refs)
	if e != nil {
		return nil, e
	}
	for i := range artifacts {
		artifacts[i].DownloadPath = "/api/v1/ops/agent/nodes/" + n.ID + "/images/" + artifacts[i].SHA256 + "/download"
	}
	return &ImagePlan{Artifacts: artifacts}, nil
}
func (s *Service) AgentImagePlan(ctx context.Context, id, token string) (*ImagePlan, error) {
	r, ok := s.repository.(*PostgreSQLRepository)
	if !ok || s.imageCatalog == nil {
		return nil, nil
	}
	n, e := r.GetNodeByToken(ctx, id, hashToken(token))
	if e != nil || n.DesiredStatus != "active" || n.ApprovedAt == nil {
		return nil, ErrAgentUnauthorized
	}
	return r.imagePlan(ctx, n)
}

// EnsureNodeImages 在修改任何工作负载之前校验期望归档就绪，下载失败不会替换原服务。
func (r *PostgreSQLRepository) EnsureNodeImages(ctx context.Context, id string, refs []string) error {
	if len(refs) == 0 {
		return nil
	}
	if r.imageCatalog == nil {
		return fmt.Errorf("中心镜像目录未配置")
	}
	var n Node
	var raw []byte
	e := r.pool.QueryRow(ctx, `SELECT id::text,tenant_id::text,node_source,architecture,resource_summary FROM host_nodes WHERE id=$1 AND desired_status='active'`, id).Scan(&n.ID, &n.TenantID, &n.NodeSource, &n.Architecture, &raw)
	if e != nil {
		return e
	}
	var summary map[string]json.RawMessage
	_ = json.Unmarshal(raw, &summary)
	var state ImageState
	_ = json.Unmarshal(summary["imageState"], &state)
	artifacts, e := r.imageCatalog.Resolve(ctx, n.Architecture, refs)
	if e != nil {
		return e
	}
	if n.NodeSource == "built_in" {
		return r.ensureCenterImagesAsync(n, artifacts)
	}
	if state.Status == "failed" {
		return fmt.Errorf("运行资源准备失败: %s", state.Message)
	}
	if state.Status != "ready" {
		return &imagePreparationPending{message: "正在准备运行资源：" + state.Message}
	}
	ready := map[string]bool{}
	for _, a := range state.Artifacts {
		ready[a] = true
	}
	for _, a := range artifacts {
		if !ready[a.SHA256] {
			return &imagePreparationPending{message: "等待节点下载并导入运行资源"}
		}
	}
	return nil
}
func (h *Handler) agentImageDownload(w http.ResponseWriter, r *http.Request) {
	plan, e := h.service.AgentImagePlan(r.Context(), chi.URLParam(r, "id"), agentToken(r))
	if e != nil {
		h.err(w, r, e)
		return
	}
	hash := chi.URLParam(r, "sha256")
	var selected *imagecatalog.Artifact
	if plan != nil {
		for i := range plan.Artifacts {
			if plan.Artifacts[i].SHA256 == hash {
				selected = &plan.Artifacts[i]
				break
			}
		}
	}
	if selected == nil {
		http.Error(w, "镜像不在节点当前资源计划中", http.StatusForbidden)
		return
	}
	o, e := h.service.imageCatalog.Store.Open(r.Context(), imagecatalog.Key(hash))
	if e != nil {
		http.Error(w, "中心镜像资源不可用", http.StatusServiceUnavailable)
		return
	}
	defer o.Reader.Close()
	if o.Size != selected.Size {
		http.Error(w, "中心镜像大小校验失败", http.StatusServiceUnavailable)
		return
	}
	offset := int64(0)
	if v := r.Header.Get("Range"); v != "" {
		if !strings.HasPrefix(v, "bytes=") || !strings.HasSuffix(v, "-") {
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return
		}
		offset, e = strconv.ParseInt(strings.TrimSuffix(strings.TrimPrefix(v, "bytes="), "-"), 10, 64)
		if e != nil || offset < 0 || offset >= o.Size {
			w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", o.Size))
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return
		}
		// MinIO 对象支持 Seek，以 Range 读取剩余字节，避免断点恢复重读整个前缀。
		if seeker, ok := o.Reader.(io.Seeker); ok {
			_, e = seeker.Seek(offset, io.SeekStart)
		} else {
			_, e = io.CopyN(io.Discard, o.Reader, offset)
		}
		if e != nil {
			http.Error(w, "读取镜像失败", 503)
			return
		}
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("ETag", `"`+hash+`"`)
	w.Header().Set("Content-Length", strconv.FormatInt(o.Size-offset, 10))
	if r.Header.Get("Range") != "" {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", offset, o.Size-1, o.Size))
		w.WriteHeader(http.StatusPartialContent)
	}
	_, _ = io.CopyN(w, o.Reader, o.Size-offset)
}

// 同一工程的所有目标节点均准备完成后，才允许任何服务切换，避免跨节点半升级。
func (r *PostgreSQLRepository) EnsureDeploymentImages(ctx context.Context, deploymentID string) error {
	rows, e := r.pool.Query(ctx, `SELECT node_id::text,service_type FROM deployment_services WHERE project_deployment_id=$1 AND desired_status='running' ORDER BY node_id,service_type`, deploymentID)
	if e != nil {
		return e
	}
	refs := map[string][]string{}
	var nodes []string
	for rows.Next() {
		var id, service string
		if e = rows.Scan(&id, &service); e != nil {
			rows.Close()
			return e
		}
		if _, ok := refs[id]; !ok {
			nodes = append(nodes, id)
		}
		refs[id] = append(refs[id], projectImageReferences(service)...)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, id := range nodes {
		if e = r.EnsureNodeImages(ctx, id, refs[id]); e != nil {
			return e
		}
	}
	return nil
}
