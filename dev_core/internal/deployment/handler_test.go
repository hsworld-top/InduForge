package deployment_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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
	testBuildingID   = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	testNodeID       = "88888888-8888-4888-8888-888888888888"
	testDeploymentID = "99999999-9999-4999-8999-999999999999"
)

func TestDeploymentHTTPFlow(t *testing.T) {
	handler, token, store := newDeploymentServer(t, "PROJECT_ADMIN")

	published := call(t, handler, http.MethodPost, "/api/v1/publish/"+testProjectID, map[string]any{"version": "9.9.9", "name": "首个版本"}, token)
	assertOK(t, published)
	var release map[string]any
	if err := json.Unmarshal(published.Data, &release); err != nil {
		t.Fatal(err)
	}
	if release["version"] != "1.0.0" || release["status"] != "ready" || release["restorable"] != true {
		t.Fatalf("完整恢复执行器接线后版本必须可恢复: %#v", release)
	}
	if len(store.content) == 0 || store.contentType != "application/zstd" || !strings.Contains(store.key, release["artifactHash"].(string)) {
		t.Fatalf("正式发布未上传受签名制品: key=%s type=%s size=%d", store.key, store.contentType, len(store.content))
	}
	assertOK(t, call(t, handler, http.MethodGet, "/api/v1/publish/"+testProjectID+"/versions?page=1&pageSize=20", nil, token))
	assertOK(t, call(t, handler, http.MethodGet, "/api/v1/deployments/project/"+testProjectID+"/nodes", nil, token))
	assertLegacyReleaseDisabled(t, call(t, handler, http.MethodPost, "/api/v1/deployments/"+testVersionID+"/deploy", map[string]any{"nodeIds": []string{testNodeID}, "runtimeConfig": map[string]any{}}, token))
	if store.presignCalls != 0 {
		t.Fatal("停用的旧部署链路仍生成了对象存储签名 URL")
	}
	assertLegacyReleaseDisabled(t, call(t, handler, http.MethodPost, "/api/v1/deployments/node-deployment/"+testDeploymentID+"/start", nil, token))
	assertOK(t, call(t, handler, http.MethodPost, "/api/v1/deployments/node-deployment/"+testDeploymentID+"/stop", nil, token))
	assertLegacyReleaseDisabled(t, call(t, handler, http.MethodPost, "/api/v1/deployments/node-deployment/"+testDeploymentID+"/restart", nil, token))
	assertOK(t, call(t, handler, http.MethodDelete, "/api/v1/deployments/node-deployment/"+testDeploymentID, nil, token))
	assertLegacyReleaseDisabled(t, call(t, handler, http.MethodPost, "/api/v1/deployments/project/"+testProjectID+"/deploy-dev", map[string]any{"nodeIds": []string{testNodeID}}, token))
	assertLegacyReleaseDisabled(t, call(t, handler, http.MethodPost, "/api/v1/deployments/"+testVersionID+"/rollback", map[string]any{"nodeId": testNodeID}, token))
	restored := call(t, handler, http.MethodPost, "/api/v1/publish/versions/"+testVersionID+"/restore-development", map[string]any{"confirmation": "RESTORE"}, token)
	assertOK(t, restored)
	if restored.HTTPStatus != http.StatusAccepted {
		t.Fatalf("restore status=%d", restored.HTTPStatus)
	}
	assertOK(t, call(t, handler, http.MethodGet, "/api/v1/publish/restore-tasks/"+testDeploymentID, nil, token))
	assertOK(t, call(t, handler, http.MethodDelete, "/api/v1/publish/deployment/"+testVersionID, nil, token))
}

func TestDeploymentPermissionBoundaries(t *testing.T) {
	developerHandler, developerToken, _ := newDeploymentServer(t, "DEVELOPER")
	assertOK(t, call(t, developerHandler, http.MethodPost, "/api/v1/publish/"+testProjectID, map[string]any{}, developerToken))
	developerDeploy := call(t, developerHandler, http.MethodPost, "/api/v1/deployments/"+testVersionID+"/deploy", map[string]any{"nodeIds": []string{testNodeID}}, developerToken)
	if developerDeploy.Code != platformapi.ErrorCodePermissionDenied {
		t.Fatalf("DEVELOPER 不应执行部署，实际 code=%d msg=%s", developerDeploy.Code, developerDeploy.Msg)
	}

	operatorHandler, operatorToken, _ := newDeploymentServer(t, "OPERATOR")
	operatorDeploy := call(t, operatorHandler, http.MethodPost, "/api/v1/deployments/"+testVersionID+"/deploy", map[string]any{"nodeIds": []string{testNodeID}}, operatorToken)
	if operatorDeploy.Code != platformapi.ErrorCodePermissionDenied {
		t.Fatalf("OPERATOR 不应创建部署，实际 code=%d msg=%s", operatorDeploy.Code, operatorDeploy.Msg)
	}
	assertOK(t, call(t, operatorHandler, http.MethodPost, "/api/v1/deployments/node-deployment/"+testDeploymentID+"/stop", nil, operatorToken))

	opsHandler, opsToken, _ := newDeploymentServer(t, "OPS_ADMIN")
	assertLegacyReleaseDisabled(t, call(t, opsHandler, http.MethodPost, "/api/v1/deployments/"+testVersionID+"/deploy", map[string]any{"nodeIds": []string{testNodeID}}, opsToken))
	assertLegacyReleaseDisabled(t, call(t, opsHandler, http.MethodPost, "/api/v1/deployments/node-deployment/"+testDeploymentID+"/restart", nil, opsToken))

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
			testVersionID: {ID: testVersionID, TenantID: testsupport.TenantID, ProjectID: testProjectID, Version: "1.0.0", Status: "ready", Restorable: true, ArtifactBucket: "artifacts", ArtifactKey: "versions/test.ifp", ArtifactHash: "hash", Manifest: map[string]any{"artifactBucket": "artifacts", "artifactKey": "versions/test.ifp", "artifactHash": "hash", "artifactSize": int64(128)}, CreatedAt: now, UpdatedAt: now},
		},
		deployments: map[string]deployment.Deployment{
			testDeploymentID: {ID: testDeploymentID, TenantID: testsupport.TenantID, ProjectID: testProjectID, NodeID: testNodeID, Status: "stopped", CreatedAt: now, UpdatedAt: now},
		},
	}
	store := &fakeStore{}
	service := deployment.NewService(repository, fakeWorkspace{}, store, deployment.ServiceConfig{ArtifactBucket: "artifacts", MinNodeAgentVersion: "1.0.0", MinRuntimeVersion: "1.0.0"})
	service.SetReleaseSourceBuilder(fakeReleaseSourceBuilder{source: validReleaseSource()})
	service.SetFormalAuthoringBuilder(fakeReleaseSourceBuilder{source: validReleaseSource()})
	service.SetAuthoringCaptureCoordinator(fakeCaptureCoordinator{})
	service.SetRestoreScheduler(fakeRestoreScheduler{})
	service.SetSigningConfig(deployment.SigningConfig{Key: ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x42}, ed25519.SeedSize)), KeyID: "test-release-key"})
	root := controlplane.NewHandler(auth.NewHandler(authService))
	root.SetDeploymentHandler(deployment.NewHandler(service, authService))
	application := app.New(app.Options{RequestID: func() string { return "deployment-test" }, Mount: func(router chi.Router) {
		platformapi.HandlerFromMuxWithBaseURL(root, router, "/api/v1")
	}})
	return application.Handler(), token, store
}

type envelope struct {
	Code       int             `json:"code"`
	Msg        string          `json:"msg"`
	Data       json.RawMessage `json:"data"`
	HTTPStatus int             `json:"-"`
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
	if response.Code != http.StatusOK && response.Code != http.StatusAccepted && response.Code != http.StatusInternalServerError {
		t.Fatalf("%s %s status=%d body=%s", method, path, response.Code, response.Body.String())
	}
	var result envelope
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	result.HTTPStatus = response.Code
	return result
}

func assertOK(t *testing.T, response envelope) {
	t.Helper()
	if response.Code != 0 {
		t.Fatalf("接口失败: %d %s", response.Code, response.Msg)
	}
}

func assertLegacyReleaseDisabled(t *testing.T, response envelope) {
	t.Helper()
	if response.Code != platformapi.ErrorCodeInvalidRequest || !strings.Contains(response.Msg, "正式版本构建与交付尚未开放") {
		t.Fatalf("旧发布旁路未失败关闭: code=%d msg=%s", response.Code, response.Msg)
	}
}

type fakeWorkspace struct{}

func (fakeWorkspace) Export(string) (map[string]string, error) {
	return map[string]string{"package.json": base64.StdEncoding.EncodeToString([]byte(`{"scripts":{"start":"vite"}}`))}, nil
}

type fakeStore struct {
	content      []byte
	presignCalls int
	key          string
	contentType  string
	putErr       error
}

func (s *fakeStore) Put(_ context.Context, key string, reader io.Reader, size int64, contentType string) (objectstore.ObjectRef, error) {
	if s.putErr != nil {
		return objectstore.ObjectRef{}, s.putErr
	}
	s.content, _ = io.ReadAll(reader)
	s.key, s.contentType = key, contentType
	return objectstore.ObjectRef{Bucket: "artifacts", Key: key, Size: size, ContentType: contentType}, nil
}

func (s *fakeStore) PresignGet(_ context.Context, key string, ttl time.Duration) (string, error) {
	s.presignCalls++
	return "http://objects.local/signed/" + key + "?expires=" + ttl.String(), nil
}

type fakeRepository struct {
	versions             map[string]deployment.Version
	deployments          map[string]deployment.Deployment
	markReadyErr         error
	markFailedTenant     string
	markFailedVersion    string
	markFailedBackground bool
}
type fakeRestoreScheduler struct{}

func (fakeRestoreScheduler) Enqueue(deployment.RestoreTask) {}

func (r *fakeRepository) AcquireAuthoringFence(context.Context, string, string, string, string, time.Duration) (string, error) {
	return "test-fence", nil
}
func (r *fakeRepository) ReleaseAuthoringFence(context.Context, string, string, string, string) error {
	return nil
}
func (r *fakeRepository) RenewAuthoringFence(context.Context, string, string, string, string, time.Duration) error {
	return nil
}
func (r *fakeRepository) GetRestoreTask(context.Context, string, string) (deployment.RestoreTask, error) {
	return deployment.RestoreTask{ID: testDeploymentID, TenantID: testsupport.TenantID, ProjectID: testProjectID, VersionID: testVersionID, State: "queued", Stage: "queued", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
}
func (r *fakeRepository) CreateRestoreTask(context.Context, string, string, string) (deployment.RestoreTask, error) {
	return deployment.RestoreTask{ID: testDeploymentID, TenantID: testsupport.TenantID, ProjectID: testProjectID, VersionID: testVersionID, State: "queued", Stage: "queued", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
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
func (r *fakeRepository) BeginProductionBuild(_ context.Context, project deployment.Project, userID, name, description string) (deployment.Version, error) {
	now := time.Now()
	item := deployment.Version{ID: testBuildingID, TenantID: project.TenantID, ProjectID: project.ID, Version: "1.0.0", Status: "building", Name: name, Description: description, CreatedAt: now, UpdatedAt: now}
	r.versions[item.ID] = item
	return item, nil
}
func (r *fakeRepository) MarkVersionReady(_ context.Context, tenantID, id string, input deployment.VersionReadyInput) (deployment.Version, error) {
	if r.markReadyErr != nil {
		return deployment.Version{}, r.markReadyErr
	}
	item, ok := r.versions[id]
	if !ok || item.TenantID != tenantID || item.Status != "building" {
		return deployment.Version{}, deployment.ErrNotFound
	}
	item.Status = "ready"
	item.ArtifactBucket, item.ArtifactKey, item.ArtifactHash, item.ArtifactSize = input.Bucket, input.ArtifactKey, input.ArtifactHash, input.ArtifactSize
	item.Manifest, item.ManifestHash, item.ChecksumsHash, item.SigningKeyID = input.Manifest, input.ManifestHash, input.ChecksumsHash, input.SigningKeyID
	item.Restorable, item.AuthoringProjectRevision = input.Restorable, input.AuthoringProjectRevision
	now := time.Now()
	item.CompletedAt, item.UpdatedAt = &now, now
	r.versions[id] = item
	return item, nil
}

type fakeReleaseSourceBuilder struct {
	source deployment.ReleaseSource
	err    error
}

func (b fakeReleaseSourceBuilder) BuildReleaseSource(context.Context, deployment.Project, deployment.Version, string) (deployment.ReleaseSource, error) {
	if b.err != nil {
		return deployment.ReleaseSource{}, b.err
	}
	return b.source, nil
}

func (b fakeReleaseSourceBuilder) Build(ctx context.Context, _ auth.User, project deployment.Project, version deployment.Version, authorization string) (deployment.AuthoringReleaseResult, error) {
	source, err := b.BuildReleaseSource(ctx, project, version, authorization)
	return deployment.AuthoringReleaseResult{Source: source, Bucket: "authoring", Key: "authoring/test", ContentHash: strings.Repeat("a", 64), CipherHash: strings.Repeat("b", 64), Size: 1, KeyID: "test-key", ProjectRevision: "sha256:test"}, err
}
func (b fakeReleaseSourceBuilder) Capture(context.Context, auth.User, deployment.Project) (deployment.CapturedAuthoring, error) {
	return deployment.CapturedAuthoring{}, b.err
}
func (b fakeReleaseSourceBuilder) BuildCaptured(ctx context.Context, project deployment.Project, version deployment.Version, _ deployment.CapturedAuthoring) (deployment.AuthoringReleaseResult, error) {
	return b.Build(ctx, auth.User{}, project, version, "")
}
func (b fakeReleaseSourceBuilder) BuildDevelopmentCaptured(ctx context.Context, project deployment.Project, version deployment.Version, captured deployment.CapturedAuthoring) (deployment.AuthoringReleaseResult, error) {
	return b.BuildCaptured(ctx, project, version, captured)
}
func (b fakeReleaseSourceBuilder) DeleteSnapshot(context.Context, string) error { return nil }

type fakeCaptureCoordinator struct{}

func (fakeCaptureCoordinator) Capture(ctx context.Context, _ auth.User, _ deployment.Project, _ string, fn func(context.Context) (deployment.CapturedAuthoring, error)) (deployment.CapturedAuthoring, error) {
	return fn(ctx)
}

func validReleaseSource() deployment.ReleaseSource {
	snapshot := deploymentCollectorSnapshot()
	return deployment.ReleaseSource{
		Client: []byte("client"), Runtime: []byte("runtime"), Collector: []byte("collector"), CollectorSourceSnapshot: snapshot,
		SBOM: []byte(`{"bomFormat":"CycloneDX"}`), ResourceRecommendation: []byte(`{"cpu":"1"}`),
		HealthContract: []byte(`{"health":"ok"}`), SchemaPlan: []byte(`{"changes":[]}`),
		ProjectDocument: map[string]any{"computeUnits": []any{map[string]any{"id": "sum"}}, "alarmItems": []any{map[string]any{"id": "high"}}, "dataPoints": []any{map[string]any{"sourceType": "collector.modbus"}}},
		SourceRevision:  "git:" + strings.Repeat("a", 40), BuilderID: "test-release-builder",
	}
}

func deploymentCollectorSnapshot() []byte {
	artifact := []byte(`{"schemaVersion":"collector-runtime-artifact.v1","artifactId":"collector-a","artifactRevision":1,"projectId":"` + testProjectID + `"}`)
	sum := sha256.Sum256(artifact)
	raw, err := json.Marshal(map[string]any{"schemaVersion": "collector-runtime-artifact.v1", "projectId": testProjectID, "artifactRevision": 1, "sha256": "sha256:" + hex.EncodeToString(sum[:]), "size": len(artifact), "artifact": json.RawMessage(artifact)})
	if err != nil {
		panic(err)
	}
	return raw
}

func TestPublishMarksBuildFailedOnPipelineErrors(t *testing.T) {
	tests := []struct {
		name      string
		builder   fakeReleaseSourceBuilder
		store     *fakeStore
		config    deployment.ServiceConfig
		markReady error
	}{
		{name: "source", builder: fakeReleaseSourceBuilder{err: errors.New("source unavailable")}, store: &fakeStore{}, config: deployment.ServiceConfig{MinNodeAgentVersion: "1.0.0", MinRuntimeVersion: "1.0.0"}},
		{name: "source with docker multiplex NUL", builder: fakeReleaseSourceBuilder{err: errors.New("source log\x00contains NUL")}, store: &fakeStore{}, config: deployment.ServiceConfig{MinNodeAgentVersion: "1.0.0", MinRuntimeVersion: "1.0.0"}},
		{name: "build", builder: fakeReleaseSourceBuilder{source: deployment.ReleaseSource{}}, store: &fakeStore{}, config: deployment.ServiceConfig{MinNodeAgentVersion: "1.0.0", MinRuntimeVersion: "1.0.0"}},
		{name: "invalid compatibility config", builder: fakeReleaseSourceBuilder{source: validReleaseSource()}, store: &fakeStore{}, config: deployment.ServiceConfig{MinNodeAgentVersion: "invalid", MinRuntimeVersion: "1.0.0"}},
		{name: "upload", builder: fakeReleaseSourceBuilder{source: validReleaseSource()}, store: &fakeStore{putErr: errors.New("object store unavailable")}, config: deployment.ServiceConfig{MinNodeAgentVersion: "1.0.0", MinRuntimeVersion: "1.0.0"}},
		{name: "mark ready", builder: fakeReleaseSourceBuilder{source: validReleaseSource()}, store: &fakeStore{}, config: deployment.ServiceConfig{MinNodeAgentVersion: "1.0.0", MinRuntimeVersion: "1.0.0"}, markReady: errors.New("state conflict")},
	}
	actor := auth.User{ID: testsupport.UserID, TenantID: testsupport.TenantID, Role: "PROJECT_ADMIN"}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &fakeRepository{versions: map[string]deployment.Version{}, deployments: map[string]deployment.Deployment{}, markReadyErr: test.markReady}
			service := deployment.NewService(repository, fakeWorkspace{}, test.store, test.config)
			service.SetReleaseSourceBuilder(test.builder)
			service.SetFormalAuthoringBuilder(test.builder)
			service.SetAuthoringCaptureCoordinator(fakeCaptureCoordinator{})
			service.SetSigningConfig(deployment.SigningConfig{Key: ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x42}, ed25519.SeedSize)), KeyID: "test-release-key"})
			if _, err := service.Publish(context.Background(), actor, testProjectID, deployment.PublishInput{}); err == nil {
				t.Fatal("失败的正式发布不应返回成功")
			}
			if got := repository.versions[testBuildingID]; got.Status != "failed" || got.ErrorMessage == "" {
				t.Fatalf("失败必须回写构建记录: %#v", got)
			}
			if repository.markFailedTenant != testsupport.TenantID || repository.markFailedVersion != testBuildingID || !repository.markFailedBackground {
				t.Fatalf("失败回写必须使用后台上下文和当前租户版本: tenant=%s version=%s background=%v", repository.markFailedTenant, repository.markFailedVersion, repository.markFailedBackground)
			}
			if strings.IndexByte(repository.versions[testBuildingID].ErrorMessage, 0) >= 0 {
				t.Fatal("失败回写文本不得含 PostgreSQL 不接受的 NUL")
			}
		})
	}
}

func TestPublishBuildsSignedImmutableRelease(t *testing.T) {
	repository := &fakeRepository{versions: map[string]deployment.Version{}, deployments: map[string]deployment.Deployment{}}
	store := &fakeStore{}
	service := deployment.NewService(repository, fakeWorkspace{}, store, deployment.ServiceConfig{ArtifactBucket: "artifacts", MinNodeAgentVersion: "1.0.0", MinRuntimeVersion: "1.0.0"})
	service.SetReleaseSourceBuilder(fakeReleaseSourceBuilder{source: validReleaseSource()})
	service.SetFormalAuthoringBuilder(fakeReleaseSourceBuilder{source: validReleaseSource()})
	service.SetAuthoringCaptureCoordinator(fakeCaptureCoordinator{})
	service.SetSigningConfig(deployment.SigningConfig{Key: ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x42}, ed25519.SeedSize)), KeyID: "test-release-key"})
	actor := auth.User{ID: testsupport.UserID, TenantID: testsupport.TenantID, Role: "PROJECT_ADMIN"}

	item, err := service.Publish(context.Background(), actor, testProjectID, deployment.PublishInput{Name: "正式版本"})
	if err != nil {
		t.Fatal(err)
	}
	if item.Status != "ready" || item.Version != "1.0.0" || item.ArtifactSize != int64(len(store.content)) || item.SigningKeyID != "test-release-key" {
		t.Fatalf("正式版本就绪元数据不完整: %#v", item)
	}
	if store.contentType != "application/zstd" || !strings.HasPrefix(item.ArtifactKey, "releases/"+testsupport.TenantID+"/"+testProjectID+"/1.0.0/") || !strings.HasSuffix(item.ArtifactKey, item.ArtifactHash+".tar.zst") {
		t.Fatalf("不可变制品对象不符合契约: key=%s type=%s hash=%s", item.ArtifactKey, store.contentType, item.ArtifactHash)
	}
	if item.ManifestHash == "" || item.ChecksumsHash == "" || item.Manifest == nil {
		t.Fatalf("Release 摘要未持久化: %#v", item)
	}
	capabilities, _ := item.Manifest["capabilities"].([]any)
	got := make(map[string]bool, len(capabilities))
	for _, capability := range capabilities {
		if value, ok := capability.(string); ok {
			got[value] = true
		}
	}
	for _, engine := range []string{"base", "compute", "alarm", "collector"} {
		if !got[engine] {
			t.Fatalf("正式 manifest 缺少自动识别引擎 %s: %#v", engine, item.Manifest)
		}
	}
}
func (r *fakeRepository) MarkVersionFailed(ctx context.Context, tenantID, id, message string) (deployment.Version, error) {
	r.markFailedTenant, r.markFailedVersion = tenantID, id
	r.markFailedBackground = ctx.Err() == nil
	item, ok := r.versions[id]
	if !ok || item.TenantID != tenantID {
		return deployment.Version{}, deployment.ErrNotFound
	}
	item.Status = "failed"
	item.ErrorMessage = message
	r.versions[id] = item
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
