package projectfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/indu-forge/dev_core/internal/auth"
	platformapi "github.com/indu-forge/dev_core/internal/platform/api"
	"github.com/jackc/pgx/v5"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) MountRoutes(router chi.Router) {
	router.Route("/projects/{projectId}/files", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.upload)
		r.Get("/{fileId}/content", h.content)
		r.Put("/{fileId}", h.rename)
		r.Post("/{fileId}/replace", h.replace)
		r.Delete("/{fileId}", h.remove)
	})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	actor, ok := actor(w, r)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, total, err := h.service.List(r.Context(), actor, chi.URLParam(r, "projectId"), ListFilter{Path: r.URL.Query().Get("path"), Keyword: r.URL.Query().Get("keyword"), Page: page, Limit: limit})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	platformapi.WriteSuccess(w, r, map[string]any{"items": items, "total": total, "page": page, "limit": limit})
}

func (h *Handler) upload(w http.ResponseWriter, r *http.Request) {
	actor, ok := actor(w, r)
	if !ok {
		return
	}
	if r.ContentLength <= 0 {
		h.writeError(w, r, fmt.Errorf("上传文件必须提供大小"))
		return
	}
	if r.ContentLength > MaxFileSize {
		h.writeError(w, r, ErrTooLarge)
		return
	}
	item, err := h.service.Upload(r.Context(), actor, chi.URLParam(r, "projectId"), r.URL.Query().Get("path"), r.URL.Query().Get("name"), r.Header.Get("Content-Type"), r.ContentLength, io.LimitReader(r.Body, MaxFileSize+1))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	platformapi.WriteSuccess(w, r, item)
}

func (h *Handler) content(w http.ResponseWriter, r *http.Request) {
	actor, ok := actor(w, r)
	if !ok {
		return
	}
	item, reader, err := h.service.Get(r.Context(), actor, chi.URLParam(r, "projectId"), chi.URLParam(r, "fileId"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	defer reader.Reader.Close()
	w.Header().Set("Content-Type", item.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(item.Size, 10))
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Header().Set("Content-Disposition", `inline; filename="`+strings.ReplaceAll(item.Name, `"`, "")+`"`)
	_, _ = io.Copy(w, reader.Reader)
}

type patchInput struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

func (h *Handler) rename(w http.ResponseWriter, r *http.Request) {
	actor, ok := actor(w, r)
	if !ok {
		return
	}
	var input patchInput
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&input); err != nil {
		h.writeError(w, r, err)
		return
	}
	item, err := h.service.Rename(r.Context(), actor, chi.URLParam(r, "projectId"), chi.URLParam(r, "fileId"), input.Path, input.Name)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, item)
}

func (h *Handler) replace(w http.ResponseWriter, r *http.Request) {
	actor, ok := actor(w, r)
	if !ok {
		return
	}
	if r.ContentLength <= 0 {
		h.writeError(w, r, fmt.Errorf("替换文件必须提供大小"))
		return
	}
	if r.ContentLength > MaxFileSize {
		h.writeError(w, r, ErrTooLarge)
		return
	}
	item, err := h.service.Replace(r.Context(), actor, chi.URLParam(r, "projectId"), chi.URLParam(r, "fileId"), r.URL.Query().Get("path"), r.URL.Query().Get("name"), r.Header.Get("Content-Type"), r.ContentLength, io.LimitReader(r.Body, MaxFileSize+1))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, item)
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	actor, ok := actor(w, r)
	if !ok {
		return
	}
	if err := h.service.Delete(r.Context(), actor, chi.URLParam(r, "projectId"), chi.URLParam(r, "fileId")); err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]bool{"deleted": true})
}

func actor(w http.ResponseWriter, r *http.Request) (auth.User, bool) {
	item, ok := auth.UserFromContext(r.Context())
	if !ok {
		platformapi.WriteError(w, r, http.StatusUnauthorized, platformapi.ErrorCodePermissionDenied, "未登录")
		return auth.User{}, false
	}
	return item, true
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	status, code := http.StatusInternalServerError, platformapi.ErrorCodeInternal
	switch {
	case errors.Is(err, auth.ErrPermissionDenied):
		status, code = http.StatusForbidden, platformapi.ErrorCodePermissionDenied
	case errors.Is(err, ErrNotFound), errors.Is(err, pgx.ErrNoRows):
		status, code = http.StatusNotFound, platformapi.ErrorCodeNotFound
	case errors.Is(err, ErrTooLarge), errors.Is(err, ErrInvalidName), errors.Is(err, ErrInvalidPath), errors.Is(err, ErrConflict):
		status, code = http.StatusBadRequest, platformapi.ErrorCodeInvalidRequest
	}
	platformapi.WriteError(w, r, status, code, err.Error())
}
