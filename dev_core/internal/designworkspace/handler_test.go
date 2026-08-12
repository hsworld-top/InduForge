package designworkspace

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/indu-forge/dev_core/internal/app"
	"github.com/indu-forge/dev_core/internal/auth"
	platformapi "github.com/indu-forge/dev_core/internal/platform/api"
	"github.com/indu-forge/dev_core/internal/project"
	"github.com/indu-forge/dev_core/internal/testsupport"
)

type testAPIHandler struct {
	platformapi.Unimplemented
	design *Handler
}

func (h testAPIHandler) ExecuteDesignFileAction(w http.ResponseWriter, r *http.Request, projectID string) {
	h.design.ExecuteDesignFileAction(w, r, projectID)
}

func (h testAPIHandler) ExportDesignFiles(w http.ResponseWriter, r *http.Request, projectID string) {
	h.design.ExportDesignFiles(w, r, projectID)
}

func (h testAPIHandler) GetDesignFileContent(w http.ResponseWriter, r *http.Request, projectID string, params platformapi.GetDesignFileContentParams) {
	h.design.GetDesignFileContent(w, r, projectID, params)
}

func TestExecuteDesignFileActionHTTP(t *testing.T) {
	workspace := t.TempDir()
	if err := os.Mkdir(filepath.Join(workspace, "displays"), 0o755); err != nil {
		t.Fatal(err)
	}
	authService, token, actor := testsupport.NewAuth(t, "DEVELOPER")
	projects := fakeProjects{item: project.Project{
		ID: testProjectID, TenantID: actor.TenantID, CreatedBy: actor.ID,
		Visibility: "private", WorkspacePath: workspace,
	}}
	handler := testAPIHandler{design: NewHandler(NewService(projects), authService)}
	application := app.New(app.Options{
		RequestID:   func() string { return "design-workspace-test" },
		Middlewares: []func(http.Handler) http.Handler{auth.ResolveUser(authService)},
		Mount: func(router chi.Router) {
			platformapi.HandlerFromMuxWithBaseURL(handler, router, "/api/v1")
		},
	})

	body, err := json.Marshal(map[string]any{
		"command": "upload",
		"data":    map[string]any{"path": "displays/main.json", "content": `{"name":"main"}`},
	})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+testProjectID+"/design-files/actions", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	application.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("HTTP 状态错误: %d %s", response.Code, response.Body.String())
	}
	var envelope platformapi.Envelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Code != 0 {
		t.Fatalf("业务响应失败: %+v", envelope)
	}
}

func TestGetDesignFileContentHTTP(t *testing.T) {
	workspace := t.TempDir()
	assetPath := filepath.Join(workspace, "assets", "pump.png")
	if err := os.MkdirAll(filepath.Dir(assetPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(assetPath, []byte{1, 2, 3}, 0o644); err != nil {
		t.Fatal(err)
	}
	authService, token, actor := testsupport.NewAuth(t, "DEVELOPER")
	projects := fakeProjects{item: project.Project{
		ID: testProjectID, TenantID: actor.TenantID, CreatedBy: actor.ID,
		Visibility: "private", WorkspacePath: workspace,
	}}
	handler := testAPIHandler{design: NewHandler(NewService(projects), authService)}
	application := app.New(app.Options{
		RequestID:   func() string { return "design-content-test" },
		Middlewares: []func(http.Handler) http.Handler{auth.ResolveUser(authService)},
		Mount: func(router chi.Router) {
			platformapi.HandlerFromMuxWithBaseURL(handler, router, "/api/v1")
		},
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+testProjectID+"/design-files/content?path=assets%2Fpump.png", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	application.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK || string(response.Body.Bytes()) != string([]byte{1, 2, 3}) {
		t.Fatalf("资源响应错误: status=%d body=%v", response.Code, response.Body.Bytes())
	}
	if response.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("资源类型错误: %s", response.Header().Get("Content-Type"))
	}
}

func TestExportDesignFilesHTTP(t *testing.T) {
	workspace := t.TempDir()
	file := filepath.Join(workspace, "scenes", "main.json")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(`{"name":"main"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	authService, token, actor := testsupport.NewAuth(t, "DEVELOPER")
	projects := fakeProjects{item: project.Project{
		ID: testProjectID, TenantID: actor.TenantID, CreatedBy: actor.ID,
		Visibility: "private", WorkspacePath: workspace,
	}}
	handler := testAPIHandler{design: NewHandler(NewService(projects), authService)}
	application := app.New(app.Options{
		RequestID:   func() string { return "design-export-test" },
		Middlewares: []func(http.Handler) http.Handler{auth.ResolveUser(authService)},
		Mount: func(router chi.Router) {
			platformapi.HandlerFromMuxWithBaseURL(handler, router, "/api/v1")
		},
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+testProjectID+"/design-files/export", bytes.NewBufferString(`{"paths":["scenes/main.json"]}`))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	application.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "application/zip" || response.Header().Get("Content-Disposition") == "" {
		t.Fatalf("导出响应错误: status=%d headers=%v body=%s", response.Code, response.Header(), response.Body.String())
	}
	reader, err := zip.NewReader(bytes.NewReader(response.Body.Bytes()), int64(response.Body.Len()))
	if err != nil || len(reader.File) != 1 || reader.File[0].Name != "scenes/main.json" {
		t.Fatalf("导出 ZIP 内容错误: files=%v err=%v", reader.File, err)
	}
}
