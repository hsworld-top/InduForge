package deployment_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/indu-forge/dev_core/internal/app"
	"github.com/indu-forge/dev_core/internal/auth"
	"github.com/indu-forge/dev_core/internal/controlplane"
	"github.com/indu-forge/dev_core/internal/deployment"
	"github.com/indu-forge/dev_core/internal/objectstore"
	platformapi "github.com/indu-forge/dev_core/internal/platform/api"
	"github.com/indu-forge/dev_core/internal/testsupport"
)

const (
	testProjectID    = "66666666-6666-4666-8666-666666666666"
	testVersionID    = "77777777-7777-4777-8777-777777777777"
	testNodeID       = "88888888-8888-4888-8888-888888888888"
	testDeploymentID = "99999999-9999-4999-8999-999999999999"
)

func TestDeploymentHTTPFlow(t *testing.T) {
	handler, token, store := newDeploymentServer(t, "PROJECT_ADMIN")

	assertOK(t, call(t, handler, http.MethodPost, "/api/v1/publish/"+testProjectID, map[string]any{"version": "1.0.0", "name": "首个版本"}, token))
	if len(store.content) == 0 {
		t.Fatal("发布后未写入工件")
	}
	if _, err := zip.NewReader(bytes.NewReader(store.content), int64(len(store.content))); err != nil {
		t.Fatalf("IFP 不是有效 ZIP: %v", err)
	}
	assertOK(t, call(t, handler, http.MethodGet, "/api/v1/publish/"+testProjectID+"/versions?page=1&pageSize=20", nil, token))
	assertOK(t, call(t, handler, http.MethodGet, "/api/v1/deployments/project/"+testProjectID+"/nodes", nil, token))
	assertOK(t, call(t, handler, http.MethodPost, "/api/v1/deployments/"+testVersionID+"/deploy", map[string]any{"nodeIds": []string{testNodeID}, "runtimeConfig": map[string]any{}}, token))
	if store.presignCalls == 0 {
		t.Fatal("创建部署时未生成短期签名 URL")
	}
	assertOK(t, call(t, handler, http.MethodPost, "/api/v1/deployments/node-deployment/"+testDeploymentID+"/start", nil, token))
	assertOK(t, call(t, handler, http.MethodPost, "/api/v1/deployments/node-deployment/"+testDeploymentID+"/stop", nil, token))
	assertOK(t, call(t, handler, http.MethodPost, "/api/v1/deployments/node-deployment/"+testDeploymentID+"/restart", nil, token))
	assertOK(t, call(t, handler, http.MethodDelete, "/api/v1/deployments/node-deployment/"+testDeploymentID, nil, token))
	assertOK(t, call(t, handler, http.MethodPost, "/api/v1/deployments/project/"+testProjectID+"/deploy-dev", map[string]any{"nodeIds": []string{testNodeID}}, token))
	assertOK(t, call(t, handler, http.MethodPost, "/api/v1/deployments/"+testVersionID+"/rollback", map[string]any{"nodeId": testNodeID}, token))
	assertOK(t, call(t, handler, http.MethodDelete, "/api/v1/publish/deployment/"+testVersionID, nil, token))
}

func TestDeploymentPermissionBoundaries(t *testing.T) {
	developerHandler, developerToken, _ := newDeploymentServer(t, "DEVELOPER")
	assertOK(t, call(t, developerHandler, http.MethodPost, "/api/v1/publish/"+testProjectID, map[string]any{"version": "1.0.0"}, developerToken))
	developerDeploy := call(t, developerHandler, http.MethodPost, "/api/v1/deployments/"+testVersionID+"/deploy", map[string]any{"nodeIds": []string{testNodeID}}, developerToken)
	if developerDeploy.Code != platformapi.ErrorCodePermissionDenied {
		t.Fatalf("DEVELOPER 不应执行部署，实际 code=%d msg=%s", developerDeploy.Code, developerDeploy.Msg)
	}

	operatorHandler, operatorToken, _ := newDeploymentServer(t, "OPERATOR")
	operatorDeploy := call(t, operatorHandler, http.MethodPost, "/api/v1/deployments/"+testVersionID+"/deploy", map[string]any{"nodeIds": []string{testNodeID}}, operatorToken)
	if operatorDeploy.Code != platformapi.ErrorCodePermissionDenied {
		t.Fatalf("OPERATOR 不应创建部署，实际 code=%d msg=%s", operatorDeploy.Code, operatorDeploy.Msg)
	}
	assertOK(t, call(t, operatorHandler, http.MethodPost, "/api/v1/deployments/node-deployment/"+testDeploymentID+"/start", nil, operatorToken))

	opsHandler, opsToken, _ := newDeploymentServer(t, "OPS_ADMIN")
	assertOK(t, call(t, opsHandler, http.MethodPost, "/api/v1/deployments/"+testVersionID+"/deploy", map[string]any{"nodeIds": []string{testNodeID}}, opsToken))
	assertOK(t, call(t, opsHandler, http.MethodPost, "/api/v1/deployments/node-deployment/"+testDeploymentID+"/restart", nil, opsToken))

	userAdminHandler, userAdminToken, _ := newDeploymentServer(t, "USER_ADMIN")
	listed := call(t, userAdminHandler, http.MethodGet, "/api/v1/publish/"+testProjectID+"/versions", nil, userAdminToken)
	if listed.Code != platformapi.ErrorCodePermissionDenied {
		t.Fatalf("USER_ADMIN 不应读取发布版本，实际 code=%d msg=%s", listed.Code, listed.Msg)
	}
}

func newDeploymentServer(t *testing.T, role string) (http.Handler, string, *fakeStore) {
	t.Helper()
	authService, token, _ := testsupport.NewAuth(t, role)
	now := time.Now()
	repository := &fakeRepository{
		versions: map[string]deployment.Version{
			testVersionID: {ID: testVersionID, TenantID: testsupport.TenantID, ProjectID: testProjectID, Version: "1.0.0", Status: "ready", ArtifactBucket: "artifacts", ArtifactKey: "versions/test.ifp", ArtifactHash: "hash", Manifest: map[string]any{"artifactBucket": "artifacts", "artifactKey": "versions/test.ifp", "artifactHash": "hash", "artifactSize": int64(128)}, CreatedAt: now, UpdatedAt: now},
		},
		deployments: map[string]deployment.Deployment{
			testDeploymentID: {ID: testDeploymentID, TenantID: testsupport.TenantID, ProjectID: testProjectID, NodeID: testNodeID, Status: "stopped", CreatedAt: now, UpdatedAt: now},
		},
	}
	store := &fakeStore{}
	service := deployment.NewService(repository, fakeWorkspace{}, store, deployment.ServiceConfig{ArtifactBucket: "artifacts"})
	root := controlplane.NewHandler(auth.NewHandler(authService))
	root.SetDeploymentHandler(deployment.NewHandler(service, authService))
	application := app.New(app.Options{RequestID: func() string { return "deployment-test" }, Mount: func(router chi.Router) {
		platformapi.HandlerFromMuxWithBaseURL(root, router, "/api/v1")
	}})
	return application.Handler(), token, store
}

type envelope struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func call(t *testing.T, handler http.Handler, method, path string, body any, token string) envelope {
	t.Helper()
	var raw []byte
	if body != nil {
		raw, _ = json.Marshal(body)
	}
	request := httptest.NewRequest(method, path, bytes.NewReader(raw))
	request.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("%s %s status=%d body=%s", method, path, response.Code, response.Body.String())
	}
	var result envelope
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func assertOK(t *testing.T, response envelope) {
	t.Helper()
	if response.Code != 0 {
		t.Fatalf("接口失败: %d %s", response.Code, response.Msg)
	}
}

type fakeWorkspace struct{}

func (fakeWorkspace) Export(string) (map[string]string, error) {
	return map[string]string{"package.json": base64.StdEncoding.EncodeToString([]byte(`{"scripts":{"start":"vite"}}`))}, nil
}

type fakeStore struct {
	content      []byte
	presignCalls int
}

func (s *fakeStore) Put(_ context.Context, key string, reader io.Reader, size int64, contentType string) (objectstore.ObjectRef, error) {
	s.content, _ = io.ReadAll(reader)
	return objectstore.ObjectRef{Bucket: "artifacts", Key: key, Size: size, ContentType: contentType}, nil
}

func (s *fakeStore) PresignGet(_ context.Context, key string, ttl time.Duration) (string, error) {
	s.presignCalls++
	return "http://objects.local/signed/" + key + "?expires=" + ttl.String(), nil
}

type fakeRepository struct {
	versions    map[string]deployment.Version
	deployments map[string]deployment.Deployment
}

func (r *fakeRepository) GetProject(_ context.Context, tenantID, projectID string) (deployment.Project, error) {
	if tenantID != testsupport.TenantID || projectID != testProjectID {
		return deployment.Project{}, deployment.ErrNotFound
	}
	return deployment.Project{ID: testProjectID, TenantID: tenantID, Name: "测试工程", Code: "TEST", WorkspacePath: "workspace", CreatedBy: testsupport.UserID, Visibility: "private"}, nil
}

func (r *fakeRepository) ListVersions(_ context.Context, tenantID, projectID string, _, _ int) ([]deployment.Version, int64, error) {
	items := make([]deployment.Version, 0, len(r.versions))
	for _, item := range r.versions {
		if item.TenantID == tenantID && item.ProjectID == projectID {
			items = append(items, item)
		}
	}
	return items, int64(len(items)), nil
}

func (r *fakeRepository) GetVersion(_ context.Context, tenantID, id string) (deployment.Version, error) {
	item, ok := r.versions[id]
	if !ok || item.TenantID != tenantID {
		return deployment.Version{}, deployment.ErrNotFound
	}
	return item, nil
}

func (r *fakeRepository) GetDeployment(_ context.Context, tenantID, id string) (deployment.Deployment, error) {
	item, ok := r.deployments[id]
	if !ok || item.TenantID != tenantID {
		return deployment.Deployment{}, deployment.ErrNotFound
	}
	return item, nil
}

func (r *fakeRepository) CreateVersion(_ context.Context, input deployment.CreateVersionInput) (deployment.Version, error) {
	now := time.Now()
	item := deployment.Version{ID: testVersionID, TenantID: input.Project.TenantID, ProjectID: input.Project.ID, Version: input.Version, Name: input.Name, Description: input.Description, Status: "ready", SourceHash: input.SourceHash, ArtifactKey: input.ArtifactKey, ArtifactHash: input.ArtifactHash, ArtifactSize: input.ArtifactSize, Manifest: input.Manifest, CompletedAt: &now, CreatedAt: now, UpdatedAt: now}
	r.versions[item.ID] = item
	return item, nil
}

func (r *fakeRepository) DeleteVersion(_ context.Context, tenantID, id string) error {
	item, ok := r.versions[id]
	if !ok || item.TenantID != tenantID {
		return deployment.ErrNotFound
	}
	delete(r.versions, id)
	return nil
}

func (r *fakeRepository) ListProjectDeployments(_ context.Context, tenantID, projectID string) ([]deployment.Deployment, error) {
	items := []deployment.Deployment{}
	for _, item := range r.deployments {
		if item.TenantID == tenantID && item.ProjectID == projectID {
			items = append(items, item)
		}
	}
	return items, nil
}

func (r *fakeRepository) Deploy(_ context.Context, tenantID string, input deployment.DeployInput) ([]deployment.Deployment, error) {
	if len(input.NodeIDs) == 0 {
		return nil, deployment.ErrNotFound
	}
	now := time.Now()
	version := "DEV"
	versionID := ""
	if input.Version != nil {
		version = input.Version.Version
		versionID = input.Version.ID
	}
	item := deployment.Deployment{ID: testDeploymentID, TenantID: tenantID, NodeID: input.NodeIDs[0], ProjectID: input.Project.ID, ApplicationVersionID: versionID, Version: version, Mode: input.Mode, Status: "pending", RuntimeConfig: input.RuntimeConfig, CreatedAt: now, UpdatedAt: now}
	r.deployments[item.ID] = item
	return []deployment.Deployment{item}, nil
}

func (r *fakeRepository) Operate(_ context.Context, tenantID, id, operation string) (deployment.Deployment, error) {
	item, ok := r.deployments[id]
	if !ok || item.TenantID != tenantID {
		return deployment.Deployment{}, deployment.ErrNotFound
	}
	item.Status = map[string]string{"start": "deploying", "stop": "stopped", "restart": "deploying", "remove": "removed"}[operation]
	item.UpdatedAt = time.Now()
	r.deployments[id] = item
	return item, nil
}
