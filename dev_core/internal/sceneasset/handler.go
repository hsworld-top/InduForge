package sceneasset

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/indu-forge/dev_core/internal/auth"
	platformapi "github.com/indu-forge/dev_core/internal/platform/api"
	"github.com/indu-forge/dev_core/internal/project"
)

type Handler struct {
	service        *Service
	dataServiceURL string
	httpClient     *http.Client
}

func NewHandler(service *Service, dataServiceURL string) *Handler {
	return &Handler{service: service, dataServiceURL: strings.TrimRight(dataServiceURL, "/"), httpClient: http.DefaultClient}
}

func (h *Handler) MountRoutes(router chi.Router) {
	router.Get("/scene-provider", h.getProvider)
	router.Put("/scene-provider", h.putProvider)
	router.Route("/projects/{projectId}/scenes", func(router chi.Router) {
		router.Get("/", h.listScenes)
		router.Post("/", h.createScene)
		router.Route("/{sceneId}", func(router chi.Router) {
			router.Get("/", h.getScene)
			router.Put("/", h.updateScene)
			router.Delete("/", h.deleteScene)
			router.Post("/editor-session", h.createEditorSession)
			router.Post("/commit", h.commit)
		})
	})
	router.Route("/projects/{projectId}/scene-assets", func(router chi.Router) {
		router.Get("/", h.listAssets)
		router.Post("/import", h.importAsset)
		router.Route("/{assetId}", func(router chi.Router) {
			router.Get("/", h.getAsset)
			router.Put("/", h.updateAsset)
			router.Delete("/", h.archiveAsset)
			router.Post("/replace", h.replaceAsset)
			router.Get("/thumbnail", h.getAssetThumbnail)
			router.Post("/editor-session", h.createAssetEditorSession)
		})
	})
	router.Route("/scene-editor-sessions/{sessionId}", func(router chi.Router) {
		router.Get("/files", h.listFiles)
		router.Get("/files/content", h.getFileContent)
		router.Put("/files/content", h.putFileContent)
		router.Post("/import", h.importFiles)
		router.Post("/export", h.exportFiles)
		router.Post("/commit", h.commitSession)
		router.Get("/datapoints", h.listDatapoints)
		router.Get("/assets", h.listSessionAssets)
		router.Post("/assets/import", h.importSessionAsset)
		router.Post("/assets/actions", h.assetAction)
		router.Post("/assets/{assetId}/replace", h.replaceSessionAsset)
		router.Delete("/assets/{assetId}", h.archiveSessionAsset)
		router.Post("/assets/{assetId}/editor-session", h.createSessionAssetEditor)
		router.Get("/dependencies", h.listDependencies)
	})
	router.Route("/scene-asset-editor-sessions/{sessionId}", func(router chi.Router) {
		router.Get("/files", h.listAssetDraftFiles)
		router.Get("/files/content", h.getAssetDraftFile)
		router.Put("/files/content", h.putAssetDraftFile)
		router.Post("/commit", h.commitAssetDraft)
	})
}

func (h *Handler) listAssets(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireActor(w, r)
	if !ok {
		return
	}
	page, limit := pagination(r)
	items, total, err := h.service.ListAssets(r.Context(), actor, chi.URLParam(r, "projectId"), AssetListFilter{
		Type: AssetType(r.URL.Query().Get("type")), Kind: r.URL.Query().Get("kind"), Keyword: r.URL.Query().Get("keyword"),
		Page: page, Limit: limit, Sort: r.URL.Query().Get("sort"), Order: r.URL.Query().Get("order"), Archived: r.URL.Query().Get("archived") == "true",
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]any{"items": items, "total": total, "page": page, "limit": limit})
}

func (h *Handler) getAsset(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireActor(w, r)
	if !ok {
		return
	}
	item, err := h.service.GetAsset(r.Context(), actor, chi.URLParam(r, "projectId"), chi.URLParam(r, "assetId"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, item)
}

func (h *Handler) importAsset(w http.ResponseWriter, r *http.Request) {
	h.writeImportedAsset(w, r, false)
}

func (h *Handler) replaceAsset(w http.ResponseWriter, r *http.Request) {
	h.writeImportedAsset(w, r, true)
}

func (h *Handler) writeImportedAsset(w http.ResponseWriter, r *http.Request, replace bool) {
	actor, ok := requireActor(w, r)
	if !ok {
		return
	}
	content, err := io.ReadAll(io.LimitReader(r.Body, maxAssetArchiveSize+1))
	if err != nil || int64(len(content)) > maxAssetArchiveSize {
		h.writeError(w, r, ErrFileTooLarge)
		return
	}
	input := AssetImportInput{Name: r.URL.Query().Get("name"), Type: AssetType(r.URL.Query().Get("type")),
		Filename: r.URL.Query().Get("filename"), ContentType: r.Header.Get("Content-Type"), Content: content}
	var item SceneAsset
	if replace {
		item, err = h.service.ReplaceAsset(r.Context(), actor, chi.URLParam(r, "projectId"), chi.URLParam(r, "assetId"), input)
	} else {
		item, err = h.service.ImportAsset(r.Context(), actor, chi.URLParam(r, "projectId"), input)
	}
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, item)
}

func (h *Handler) updateAsset(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireActor(w, r)
	if !ok {
		return
	}
	var input AssetPatch
	if err := decodeJSON(r, &input, 1<<20); err != nil {
		h.writeError(w, r, err)
		return
	}
	item, err := h.service.UpdateAsset(r.Context(), actor, chi.URLParam(r, "projectId"), chi.URLParam(r, "assetId"), input)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, item)
}

func (h *Handler) archiveAsset(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireActor(w, r)
	if !ok {
		return
	}
	if err := h.service.ArchiveAsset(r.Context(), actor, chi.URLParam(r, "projectId"), chi.URLParam(r, "assetId")); err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]bool{"archived": true})
}

func (h *Handler) getAssetThumbnail(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireActor(w, r)
	if !ok {
		return
	}
	content, err := h.service.OpenAssetThumbnail(r.Context(), actor, chi.URLParam(r, "projectId"), chi.URLParam(r, "assetId"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	writeFileContent(w, content, "private, max-age=300")
}

func (h *Handler) createAssetEditorSession(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireActor(w, r)
	if !ok {
		return
	}
	result, err := h.service.CreateAssetEditorSession(r.Context(), actor, chi.URLParam(r, "projectId"), chi.URLParam(r, "assetId"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, result)
}

func (h *Handler) listSessionAssets(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireActor(w, r)
	if !ok {
		return
	}
	page, limit := pagination(r)
	items, bindings, total, scene, err := h.service.ListSessionAssets(r.Context(), actor, chi.URLParam(r, "sessionId"), AssetListFilter{
		Type: AssetType(r.URL.Query().Get("type")), Keyword: r.URL.Query().Get("keyword"), Page: page, Limit: limit,
		Sort: r.URL.Query().Get("sort"), Order: r.URL.Query().Get("order"),
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]any{"items": items, "bindings": bindings, "total": total,
		"page": page, "limit": limit, "draftVersion": scene.DraftVersion})
}

func (h *Handler) assetAction(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireActor(w, r)
	if !ok {
		return
	}
	var action AssetAction
	if err := decodeJSON(r, &action, 1<<20); err != nil {
		h.writeError(w, r, err)
		return
	}
	next, err := h.service.ApplyAssetAction(r.Context(), actor, chi.URLParam(r, "sessionId"), action)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]int64{"draftVersion": next})
}

func (h *Handler) importSessionAsset(w http.ResponseWriter, r *http.Request) {
	h.writeSessionImportedAsset(w, r, false)
}

func (h *Handler) replaceSessionAsset(w http.ResponseWriter, r *http.Request) {
	h.writeSessionImportedAsset(w, r, true)
}

func (h *Handler) writeSessionImportedAsset(w http.ResponseWriter, r *http.Request, replace bool) {
	actor, ok := requireActor(w, r)
	if !ok {
		return
	}
	content, err := io.ReadAll(io.LimitReader(r.Body, maxAssetArchiveSize+1))
	if err != nil || int64(len(content)) > maxAssetArchiveSize {
		h.writeError(w, r, ErrFileTooLarge)
		return
	}
	input := AssetImportInput{Name: r.URL.Query().Get("name"), Type: AssetType(r.URL.Query().Get("type")),
		Filename: r.URL.Query().Get("filename"), ContentType: r.Header.Get("Content-Type"), Content: content}
	var item SceneAsset
	if replace {
		item, err = h.service.ReplaceSessionAsset(r.Context(), actor, chi.URLParam(r, "sessionId"), chi.URLParam(r, "assetId"), input)
	} else {
		item, err = h.service.ImportSessionAsset(r.Context(), actor, chi.URLParam(r, "sessionId"), input)
	}
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, item)
}

func (h *Handler) archiveSessionAsset(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireActor(w, r)
	if !ok {
		return
	}
	if err := h.service.ArchiveSessionAsset(r.Context(), actor, chi.URLParam(r, "sessionId"), chi.URLParam(r, "assetId")); err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]bool{"archived": true})
}

func (h *Handler) createSessionAssetEditor(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireActor(w, r)
	if !ok {
		return
	}
	result, err := h.service.CreateSessionAssetEditor(r.Context(), actor, chi.URLParam(r, "sessionId"), chi.URLParam(r, "assetId"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, result)
}

func (h *Handler) listDependencies(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireActor(w, r)
	if !ok {
		return
	}
	page, limit := pagination(r)
	items, total, scene, err := h.service.ListDependencies(r.Context(), actor, chi.URLParam(r, "sessionId"), FileListFilter{
		Directory: r.URL.Query().Get("directory"), Keyword: r.URL.Query().Get("keyword"), Page: page, Limit: limit,
		Sort: r.URL.Query().Get("sort"), Order: r.URL.Query().Get("order"),
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]any{"items": items, "total": total, "page": page, "limit": limit, "draftVersion": scene.DraftVersion})
}

func (h *Handler) listAssetDraftFiles(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireActor(w, r)
	if !ok {
		return
	}
	page, limit := pagination(r)
	items, total, asset, err := h.service.ListAssetDraftFiles(r.Context(), actor, chi.URLParam(r, "sessionId"), FileListFilter{
		Directory: r.URL.Query().Get("directory"), Keyword: r.URL.Query().Get("keyword"), Page: page, Limit: limit,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]any{"items": items, "total": total, "page": page, "limit": limit,
		"draftVersion": asset.DraftVersion, "assetType": asset.Type, "entryFile": asset.EntryPath})
}

func (h *Handler) getAssetDraftFile(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireActor(w, r)
	if !ok {
		return
	}
	content, err := h.service.OpenAssetDraftFile(r.Context(), actor, chi.URLParam(r, "sessionId"), r.URL.Query().Get("path"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	writeFileContent(w, content, "no-store")
}

func (h *Handler) putAssetDraftFile(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireActor(w, r)
	if !ok {
		return
	}
	content, err := io.ReadAll(io.LimitReader(r.Body, MaxFileSize+1))
	if err != nil || int64(len(content)) > MaxFileSize {
		h.writeError(w, r, ErrFileTooLarge)
		return
	}
	base, _ := strconv.ParseInt(r.URL.Query().Get("baseDraftVersion"), 10, 64)
	next, err := h.service.PutAssetDraftFile(r.Context(), actor, chi.URLParam(r, "sessionId"), FileWrite{
		Path: r.URL.Query().Get("path"), ContentType: r.Header.Get("Content-Type"), Content: content, BaseDraftVersion: base,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]int64{"draftVersion": next})
}

func (h *Handler) commitAssetDraft(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireActor(w, r)
	if !ok {
		return
	}
	var input CommitInput
	if err := decodeJSON(r, &input, 1<<20); err != nil {
		h.writeError(w, r, err)
		return
	}
	item, err := h.service.CommitAssetDraft(r.Context(), actor, chi.URLParam(r, "sessionId"), input.BaseDraftVersion)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, item)
}

func (h *Handler) commitSession(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireActor(w, r)
	if !ok {
		return
	}
	var input CommitInput
	if err := decodeJSON(r, &input, 1<<20); err != nil {
		h.writeError(w, r, err)
		return
	}
	session, _, err := h.service.ResolveSession(r.Context(), actor, chi.URLParam(r, "sessionId"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	result, err := h.service.Commit(r.Context(), actor, session.ProjectID, session.Kind, session.SceneID, input.BaseDraftVersion)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, result)
}

func (h *Handler) getProvider(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireActor(w, r)
	if !ok {
		return
	}
	state, capabilities, err := h.service.GetProvider(r.Context(), actor)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]any{"provider": PublicProvider, "providerVersion": PublicVersion, "updatedAt": state.UpdatedAt, "capabilities": capabilities})
}

func (h *Handler) putProvider(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireActor(w, r)
	if !ok {
		return
	}
	var input struct {
		Provider string `json:"provider"`
	}
	if err := decodeJSON(r, &input, 1<<20); err != nil {
		h.writeError(w, r, err)
		return
	}
	provider := input.Provider
	if provider == PublicProvider {
		provider = ProviderHT
	}
	state, err := h.service.SetProvider(r.Context(), actor, provider)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]any{"provider": PublicProvider, "providerVersion": PublicVersion, "updatedAt": state.UpdatedAt})
}

func (h *Handler) createScene(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireActor(w, r)
	if !ok {
		return
	}
	var input SceneInput
	if err := decodeJSON(r, &input, 2<<20); err != nil {
		h.writeError(w, r, err)
		return
	}
	item, err := h.service.CreateScene(r.Context(), actor, chi.URLParam(r, "projectId"), input)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, item)
}

func (h *Handler) listScenes(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireActor(w, r)
	if !ok {
		return
	}
	page, limit := pagination(r)
	items, total, err := h.service.ListScenes(r.Context(), actor, chi.URLParam(r, "projectId"), ListFilter{
		Kind: r.URL.Query().Get("kind"), Keyword: r.URL.Query().Get("keyword"), Page: page, Limit: limit,
		Sort: r.URL.Query().Get("sort"), Order: r.URL.Query().Get("order"),
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]any{"items": items, "total": total, "page": page, "limit": limit})
}

func (h *Handler) getScene(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireActor(w, r)
	if !ok {
		return
	}
	item, err := h.service.GetScene(r.Context(), actor, chi.URLParam(r, "projectId"), sceneKind(r), chi.URLParam(r, "sceneId"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, item)
}

func (h *Handler) updateScene(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireActor(w, r)
	if !ok {
		return
	}
	var input ScenePatch
	if err := decodeJSON(r, &input, 2<<20); err != nil {
		h.writeError(w, r, err)
		return
	}
	item, err := h.service.UpdateScene(r.Context(), actor, chi.URLParam(r, "projectId"), sceneKind(r), chi.URLParam(r, "sceneId"), input)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, item)
}

func (h *Handler) deleteScene(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireActor(w, r)
	if !ok {
		return
	}
	err := h.service.DeleteScene(r.Context(), actor, chi.URLParam(r, "projectId"), sceneKind(r), chi.URLParam(r, "sceneId"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]bool{"deleted": true})
}

func (h *Handler) createEditorSession(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireActor(w, r)
	if !ok {
		return
	}
	result, err := h.service.CreateEditorSession(r.Context(), actor, chi.URLParam(r, "projectId"), sceneKind(r), chi.URLParam(r, "sceneId"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, result)
}

func (h *Handler) commit(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireActor(w, r)
	if !ok {
		return
	}
	var input CommitInput
	if err := decodeJSON(r, &input, 1<<20); err != nil {
		h.writeError(w, r, err)
		return
	}
	result, err := h.service.Commit(r.Context(), actor, chi.URLParam(r, "projectId"), sceneKind(r), chi.URLParam(r, "sceneId"), input.BaseDraftVersion)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, result)
}

func (h *Handler) listFiles(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireActor(w, r)
	if !ok {
		return
	}
	page, limit := pagination(r)
	items, total, scene, err := h.service.ListFiles(r.Context(), actor, chi.URLParam(r, "sessionId"), FileListFilter{
		Directory: r.URL.Query().Get("directory"), Keyword: r.URL.Query().Get("keyword"), Page: page, Limit: limit,
		Sort: r.URL.Query().Get("sort"), Order: r.URL.Query().Get("order"),
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]any{"items": items, "total": total, "page": page, "limit": limit, "draftVersion": scene.DraftVersion})
}

func (h *Handler) getFileContent(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireActor(w, r)
	if !ok {
		return
	}
	content, err := h.service.OpenFile(r.Context(), actor, chi.URLParam(r, "sessionId"), r.URL.Query().Get("path"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	writeFileContent(w, content, "no-store")
}

func writeFileContent(w http.ResponseWriter, content FileContent, cacheControl string) {
	if content.Reader != nil {
		defer content.Reader.Close()
	}
	w.Header().Set("Content-Type", content.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(content.Size, 10))
	w.Header().Set("ETag", `"`+content.Hash+`"`)
	w.Header().Set("Cache-Control", cacheControl)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if content.Reader != nil {
		_, _ = io.Copy(w, content.Reader)
	} else {
		_, _ = w.Write(content.Bytes)
	}
}

func (h *Handler) putFileContent(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireActor(w, r)
	if !ok {
		return
	}
	content, err := io.ReadAll(io.LimitReader(r.Body, MaxFileSize+1))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	if int64(len(content)) > MaxFileSize {
		h.writeError(w, r, ErrFileTooLarge)
		return
	}
	version, _ := strconv.ParseInt(r.URL.Query().Get("baseDraftVersion"), 10, 64)
	next, err := h.service.PutFile(r.Context(), actor, chi.URLParam(r, "sessionId"), FileWrite{Path: r.URL.Query().Get("path"), ContentType: r.Header.Get("Content-Type"), Content: content, BaseDraftVersion: version})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]int64{"draftVersion": next})
}

func (h *Handler) importFiles(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireActor(w, r)
	if !ok {
		return
	}
	content, err := io.ReadAll(io.LimitReader(r.Body, (512<<20)+1))
	if err != nil || len(content) > 512<<20 {
		h.writeError(w, r, ErrFileTooLarge)
		return
	}
	base, _ := strconv.ParseInt(r.URL.Query().Get("baseDraftVersion"), 10, 64)
	next, err := h.service.Import(r.Context(), actor, chi.URLParam(r, "sessionId"), content, base)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]int64{"draftVersion": next})
}

func (h *Handler) exportFiles(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireActor(w, r)
	if !ok {
		return
	}
	content, err := h.service.Export(r.Context(), actor, chi.URLParam(r, "sessionId"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="scene-assets.zip"`)
	w.Header().Set("Cache-Control", "no-store")
	_, _ = io.Copy(w, bytes.NewReader(content))
}

func (h *Handler) listDatapoints(w http.ResponseWriter, r *http.Request) {
	actor, ok := requireActor(w, r)
	if !ok {
		return
	}
	session, _, err := h.service.ResolveSession(r.Context(), actor, chi.URLParam(r, "sessionId"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	if h.dataServiceURL == "" {
		h.writeError(w, r, errors.New("数据服务地址未配置"))
		return
	}
	target, _ := url.Parse(h.dataServiceURL + "/api/v1/data/projects/" + url.PathEscape(session.ProjectID) + "/datapoints")
	target.RawQuery = r.URL.RawQuery
	request, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target.String(), nil)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	if authorization := auth.ForwardAuthorization(r); authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	response, err := h.httpClient.Do(request)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	defer response.Body.Close()
	w.Header().Set("Content-Type", response.Header.Get("Content-Type"))
	w.WriteHeader(response.StatusCode)
	_, _ = io.Copy(w, response.Body)
}

func requireActor(w http.ResponseWriter, r *http.Request) (auth.User, bool) {
	actor, ok := auth.UserFromContext(r.Context())
	if !ok {
		platformapi.WriteError(w, r, http.StatusUnauthorized, platformapi.ErrorCodeTokenRequired, "缺少或无效的访问令牌")
	}
	return actor, ok
}

func decodeJSON(r *http.Request, target any, limit int64) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, limit))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("请求体格式无效: %w", err)
	}
	return nil
}

func pagination(r *http.Request) (int, int) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	return normalizePage(page, limit)
}

func sceneKind(r *http.Request) string {
	kind := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("kind")))
	if kind == "" {
		kind = "2d"
	}
	return kind
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, auth.ErrPermissionDenied):
		platformapi.WriteError(w, r, http.StatusForbidden, platformapi.ErrorCodePermissionDenied, err.Error())
	case errors.Is(err, project.ErrNotFound), errors.Is(err, ErrNotFound), errors.Is(err, ErrEntryMissing), errors.Is(err, ErrDependencyMissing):
		platformapi.WriteError(w, r, http.StatusNotFound, platformapi.ErrorCodeNotFound, err.Error())
	case errors.Is(err, ErrAssetNotFound):
		platformapi.WriteError(w, r, http.StatusNotFound, platformapi.ErrorCodeNotFound, err.Error())
	case errors.Is(err, ErrDraftConflict), errors.Is(err, ErrAssetConflict), errors.Is(err, ErrAssetInUse):
		platformapi.WriteError(w, r, http.StatusConflict, platformapi.ErrorCodeAlreadyExists, err.Error())
	case errors.Is(err, ErrSessionExpired):
		platformapi.WriteError(w, r, http.StatusUnauthorized, platformapi.ErrorCodeTokenInvalid, err.Error())
	case errors.Is(err, ErrSessionForbidden):
		platformapi.WriteError(w, r, http.StatusForbidden, platformapi.ErrorCodePermissionDenied, err.Error())
	case errors.Is(err, ErrProviderBusy), errors.Is(err, ErrInvalidProvider), errors.Is(err, ErrInvalidKind), errors.Is(err, ErrInvalidSceneID),
		errors.Is(err, ErrInvalidPath), errors.Is(err, ErrProtectedPath), errors.Is(err, ErrFileTooLarge), errors.Is(err, ErrInvalidJSON),
		errors.Is(err, ErrInvalidAssetType), errors.Is(err, ErrInvalidAssetName), errors.Is(err, ErrAssetArchived),
		errors.Is(err, ErrAssetNotEditable), errors.Is(err, ErrDuplicateFile), errors.Is(err, ErrInvalidArchive):
		platformapi.WriteError(w, r, http.StatusBadRequest, platformapi.ErrorCodeInvalidRequest, err.Error())
	default:
		platformapi.WriteError(w, r, http.StatusInternalServerError, platformapi.ErrorCodeInternal, "系统内部错误")
	}
}
