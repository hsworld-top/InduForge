package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/repository"
	"github.com/indu-forge/data_service/internal/service"
)

const collectorBindingRequestMaxBytes int64 = 17 << 20

// collectorSnapshotReader 保证内部接口也经由项目-租户绑定读取权威快照。
type collectorSnapshotReader interface {
	Get(context.Context, string, string) (*repository.ProjectSnapshot, error)
}

// CollectorBindingBundleHandler 只向 control-plane 暴露短生命周期的采集器装配包。
// 响应可能含连接 secretFiles，因此一律 no-store，且不得在任何日志中记录 body。
type CollectorBindingBundleHandler struct {
	snapshots collectorSnapshotReader
	builder   *service.CollectorBindingBundleBuilder
}

func NewCollectorBindingBundleHandler(snapshots collectorSnapshotReader, builder *service.CollectorBindingBundleBuilder) *CollectorBindingBundleHandler {
	return &CollectorBindingBundleHandler{snapshots: snapshots, builder: builder}
}

func (h *CollectorBindingBundleHandler) Build(w http.ResponseWriter, r *http.Request) error {
	// 即使失败响应也不允许代理缓存；请求与成功响应都可能与短期 Secret 装配有关。
	w.Header().Set("Cache-Control", "no-store")
	if h == nil || h.snapshots == nil || h.builder == nil {
		return apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusServiceUnavailable, "采集器运行绑定暂不可用")
	}
	r.Body = http.MaxBytesReader(w, r.Body, collectorBindingRequestMaxBytes)
	var input struct {
		TenantID                string          `json:"tenantId"`
		DeploymentID            string          `json:"deploymentId"`
		EnvironmentID           string          `json:"environmentId"`
		NodeID                  string          `json:"nodeId"`
		ReleaseID               string          `json:"releaseId"`
		Revision                int64           `json:"revision"`
		SourceSnapshot          json.RawMessage `json:"sourceSnapshot"`
		AccountID               string          `json:"accountId"`
		NATSEndpoint            string          `json:"natsEndpoint"`
		NATSResourceRef         string          `json:"natsResourceRef"`
		NATSCredentialSecretRef string          `json:"natsCredentialSecretRef"`
	}
	if err := decodeJSONBody(r, &input); err != nil {
		return err
	}
	projectID := r.PathValue("projectId")
	if strings.TrimSpace(input.TenantID) == "" || input.Revision < 1 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "采集器运行绑定请求无效")
	}
	snapshot, err := h.snapshots.Get(r.Context(), projectID, input.TenantID)
	if err != nil {
		// 内部接口保留 404，从而不向 control-plane 暴露跨租户项目是否存在。
		return err
	}
	bundle, err := h.builder.Build(service.TenantProjectSnapshot{TenantID: input.TenantID, ProjectID: projectID, Snapshot: snapshot}, service.DeploymentBindingInput{
		DeploymentID: input.DeploymentID, EnvironmentID: input.EnvironmentID, NodeID: input.NodeID, ReleaseID: input.ReleaseID,
		AccountID: input.AccountID, BindingRevision: input.Revision, OwnershipEpoch: input.Revision,
		NATSEndpoint: input.NATSEndpoint, NATSServerResourceRef: input.NATSResourceRef, NATSCredentialSecretRef: input.NATSCredentialSecretRef,
		SourceSnapshot: input.SourceSnapshot,
	})
	if err != nil {
		return collectorBindingError(err)
	}
	secretFiles := make(map[string]string, len(bundle.SecretFiles))
	for name, content := range bundle.SecretFiles {
		secretFiles[name] = base64.StdEncoding.EncodeToString(content)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{
		"schemaVersion": "collector-binding-bundle.v1",
		"binding":       bundle.Binding,
		"bindingSha256": bundle.BindingSHA256,
		"bindingSize":   bundle.BindingSize,
		"index":         bundle.Index,
		"indexSha256":   bundle.IndexSHA256,
		"indexSize":     bundle.IndexSize,
		"secretFiles":   secretFiles,
	})
	return nil
}

func collectorBindingError(err error) error {
	if errors.Is(err, service.ErrCollectorNotRequired) {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusUnprocessableEntity, "工程不需要采集器运行绑定")
	}
	if (strings.Contains(err.Error(), "sourceSnapshot") && strings.Contains(err.Error(), "已变化")) || strings.Contains(err.Error(), "不一致") {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "采集器源快照已变化")
	}
	// Builder 的错误均使用固定诊断文本，不包含 secret 明文或密文；此处进一步收口为
	// 公共参数错误，防止未来实现不小心把内部细节回显至控制面。
	return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "采集器运行绑定构建失败")
}
