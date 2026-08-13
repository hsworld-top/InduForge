package designworkspace

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/indu-forge/dev_core/internal/auth"
	platformapi "github.com/indu-forge/dev_core/internal/platform/api"
	"github.com/indu-forge/dev_core/internal/project"
)

type Handler struct {
	service     *Service
	authService *auth.Service
	contextSync ContextSynchronizer
}

type ContextSynchronizer interface {
	Sync(context.Context, auth.User, string, string) (map[string]any, error)
}

func NewHandler(service *Service, authService *auth.Service) *Handler {
	return &Handler{service: service, authService: authService}
}

func (h *Handler) SetContextSynchronizer(sync ContextSynchronizer) { h.contextSync = sync }

func (h *Handler) ExecuteDesignFileAction(w http.ResponseWriter, r *http.Request, projectID string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	var body platformapi.DesignFileActionRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 48<<20)).Decode(&body); err != nil {
		h.writeInvalid(w, r, err)
		return
	}
	raw, err := json.Marshal(body.Data)
	if err != nil {
		h.writeInvalid(w, r, err)
		return
	}
	result, err := h.service.Execute(r.Context(), actor, projectID, Action{Command: string(body.Command), Data: raw})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	payload := map[string]any{"result": result}
	if shouldSyncContext(body.Command) {
		payload["contextSync"] = h.syncContext(r, actor, projectID)
	}
	platformapi.WriteSuccess(w, r, payload)
}

// HT 保存成功后再刷新上下文，避免未落盘的画布状态进入代码工作区。
// 同步失败不影响保存结果，调用方根据 stale 状态提示用户手动重试。
func (h *Handler) syncContext(r *http.Request, actor auth.User, projectID string) map[string]any {
	if h.contextSync == nil {
		return map[string]any{"status": "not-configured"}
	}
	result, err := h.contextSync.Sync(r.Context(), actor, projectID, auth.ForwardAuthorization(r))
	if err != nil {
		return map[string]any{"status": "stale", "message": err.Error()}
	}
	return map[string]any{"status": "updated", "result": result}
}

func shouldSyncContext(command platformapi.DesignFileActionRequestCommand) bool {
	switch command {
	case platformapi.Upload, platformapi.Import, platformapi.Remove, platformapi.Rename, platformapi.Paste:
		return true
	default:
		return false
	}
}

func (h *Handler) ExportDesignFiles(w http.ResponseWriter, r *http.Request, projectID string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	var body platformapi.DesignFileExportRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		h.writeInvalid(w, r, err)
		return
	}
	archive, err := h.service.ExportArchive(r.Context(), actor, projectID, body.Paths)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+archive.Filename+`"`)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(archive.Content)
}

func (h *Handler) GetDesignFileContent(w http.ResponseWriter, r *http.Request, projectID string, params platformapi.GetDesignFileContentParams) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	content, err := h.service.OpenContent(r.Context(), actor, projectID, params.Path)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	defer content.File.Close()

	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(content.Info.Name())))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, content.Info.Name(), content.Info.ModTime(), content.File)
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
	switch {
	case errors.Is(err, project.ErrNotFound):
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeProjectNotFound, err.Error())
	case errors.Is(err, auth.ErrPermissionDenied):
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodePermissionDenied, err.Error())
	case errors.Is(err, os.ErrNotExist):
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeNotFound, "设计文件不存在")
	case errors.Is(err, ErrInvalidAction), errors.Is(err, ErrInvalidPath), errors.Is(err, ErrProtectedPath), errors.Is(err, ErrFileTooLarge), errors.Is(err, ErrUnsupported), errors.Is(err, ErrSymbolicLink), errors.Is(err, ErrDestinationBusy):
		h.writeInvalid(w, r, err)
	default:
		platformapi.WriteError(w, r, http.StatusInternalServerError, platformapi.ErrorCodeInternal, "系统内部错误")
	}
}

func (h *Handler) writeInvalid(w http.ResponseWriter, r *http.Request, err error) {
	platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeInvalidRequest, err.Error())
}
