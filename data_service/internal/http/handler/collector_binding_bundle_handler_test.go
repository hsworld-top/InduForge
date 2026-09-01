package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
	collectorsecurity "github.com/indu-forge/data_service/internal/security"
	"github.com/indu-forge/data_service/internal/service"
)

type bundleSnapshotReader struct {
	snapshot *repository.ProjectSnapshot
	err      error
}

func (r bundleSnapshotReader) Get(_ context.Context, _, _ string) (*repository.ProjectSnapshot, error) {
	return r.snapshot, r.err
}

func TestCollectorBindingBundleHandlerBuildsNoStoreBundle(t *testing.T) {
	cipher := handlerBundleCipher(t)
	secret, err := cipher.Encrypt([]byte("do-not-log"))
	if err != nil {
		t.Fatal(err)
	}
	connections := []repository.SnapshotCollectorConnectionRecord{
		{ID: bundleHandlerModbusID, DriverID: "modbus.tcp", Config: map[string]any{"host": "10.0.0.8", "port": 502}, IsEnabled: true},
		{ID: bundleHandlerOPCUAID, DriverID: "opcua.standard", Config: map[string]any{"host": "opc.example", "port": 4840, "endpointPath": "/ua", "securityMode": "None", "securityPolicy": "None", "authenticationType": "anonymous"}, IsEnabled: true},
	}
	snapshot := &repository.ProjectSnapshot{CollectorConnections: connections, CollectorSecrets: []repository.SnapshotCollectorSecretRecord{{ConnectionID: bundleHandlerModbusID, SecretKey: "password", EncryptedValue: secret, EncryptionKeyVersion: "v1"}}}
	handler := newBundleHandler(t, snapshot, nil)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bundleHandlerRequest(t, connections, false)))
	req.SetPathValue("projectId", bundleHandlerProjectID)
	recorder := httptest.NewRecorder()
	if err := handler.Build(recorder, req); err != nil {
		t.Fatal(err)
	}
	if recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("expected no-store, got %q", recorder.Header().Get("Cache-Control"))
	}
	if bytes.Contains(recorder.Body.Bytes(), []byte("do-not-log")) {
		t.Fatal("secret 不应以明文出现在响应")
	}
	var envelope struct {
		Data struct {
			SchemaVersion string            `json:"schemaVersion"`
			Binding       json.RawMessage   `json:"binding"`
			Index         json.RawMessage   `json:"index"`
			SecretFiles   map[string]string `json:"secretFiles"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Data.SchemaVersion != "collector-binding-bundle.v1" || len(envelope.Data.Binding) == 0 || len(envelope.Data.Index) == 0 {
		t.Fatalf("bundle 响应不完整: %s", recorder.Body.String())
	}
	decoded, err := base64.StdEncoding.DecodeString(envelope.Data.SecretFiles["connection-"+bundleHandlerModbusID+".json"])
	if err != nil || string(decoded) != `{"password":"do-not-log"}` {
		t.Fatalf("secret 文件编码错误: %q, %v", decoded, err)
	}
}

func TestCollectorBindingBundleHandlerRejectsUnknownAndNoCollector(t *testing.T) {
	handler := newBundleHandler(t, &repository.ProjectSnapshot{}, nil)
	unknown := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"tenantId":"tenant-a","unexpected":true}`))
	unknown.SetPathValue("projectId", bundleHandlerProjectID)
	if err := handler.Build(httptest.NewRecorder(), unknown); err == nil {
		t.Fatal("未知字段必须被拒绝")
	}
	connections := []repository.SnapshotCollectorConnectionRecord{}
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bundleHandlerRequest(t, connections, false)))
	req.SetPathValue("projectId", bundleHandlerProjectID)
	err := handler.Build(httptest.NewRecorder(), req)
	appErr := mustExtractAppError(t, err)
	if appErr.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", appErr.StatusCode)
	}
}

func TestCollectorBindingBundleHandlerMapsSnapshotConflictAndTenantNotFound(t *testing.T) {
	connections := []repository.SnapshotCollectorConnectionRecord{{ID: bundleHandlerModbusID, DriverID: "modbus.tcp", Config: map[string]any{"host": "10.0.0.8", "port": 502}, IsEnabled: true}}
	handler := newBundleHandler(t, &repository.ProjectSnapshot{CollectorConnections: connections}, nil)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bundleHandlerRequest(t, connections, true)))
	req.SetPathValue("projectId", bundleHandlerProjectID)
	appErr := mustExtractAppError(t, handler.Build(httptest.NewRecorder(), req))
	if appErr.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", appErr.StatusCode)
	}
	notFound := newBundleHandler(t, nil, serviceTestNotFound())
	req = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bundleHandlerRequest(t, connections, false)))
	req.SetPathValue("projectId", bundleHandlerProjectID)
	appErr = mustExtractAppError(t, notFound.Build(httptest.NewRecorder(), req))
	if appErr.StatusCode != http.StatusNotFound {
		t.Fatalf("cross tenant must be 404, got %d", appErr.StatusCode)
	}
}

const (
	bundleHandlerProjectID = "11111111-1111-4111-8111-111111111111"
	bundleHandlerModbusID  = "22222222-2222-4222-8222-222222222222"
	bundleHandlerOPCUAID   = "33333333-3333-4333-8333-333333333333"
)

func handlerBundleCipher(t *testing.T) *collectorsecurity.CollectorSecretCipher {
	t.Helper()
	cipher, err := collectorsecurity.NewCollectorSecretCipher(bytes.Repeat([]byte{4}, 32), "v1")
	if err != nil {
		t.Fatal(err)
	}
	return cipher
}

func newBundleHandler(t *testing.T, snapshot *repository.ProjectSnapshot, snapshotErr error) *CollectorBindingBundleHandler {
	t.Helper()
	builder, err := service.NewCollectorBindingBundleBuilder(handlerBundleCipher(t))
	if err != nil {
		t.Fatal(err)
	}
	return NewCollectorBindingBundleHandler(bundleSnapshotReader{snapshot: snapshot, err: snapshotErr}, builder)
}

func bundleHandlerRequest(t *testing.T, connections []repository.SnapshotCollectorConnectionRecord, mismatch bool) []byte {
	t.Helper()
	artifactConnections := make([]any, 0, len(connections))
	for _, connection := range connections {
		if connection.IsEnabled {
			artifactConnections = append(artifactConnections, map[string]any{"connectionId": connection.ID, "protocolFamily": "modbus", "driverId": connection.DriverID, "driverVersion": "1.0.0", "schemaVersion": 1, "enabled": true, "requiredOperations": []any{"point.read"}, "defaultAcquisition": map[string]any{"intervalMs": 1000, "deadband": 0, "changeOnly": false}})
		}
	}
	if mismatch {
		artifactConnections = []any{}
	}
	artifact, err := json.Marshal(map[string]any{"schemaVersion": "collector-runtime-artifact.v1", "artifactId": "artifact-a", "artifactRevision": 1, "projectId": bundleHandlerProjectID, "collectorVersion": "1.0.0", "connections": artifactConnections, "pointMappings": []any{}, "wal": map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	source, err := json.Marshal(map[string]any{"schemaVersion": "collector-runtime-artifact.v1", "projectId": bundleHandlerProjectID, "artifactRevision": 1, "sha256": handlerSHA(artifact), "size": len(artifact), "artifact": json.RawMessage(artifact)})
	if err != nil {
		t.Fatal(err)
	}
	request, err := json.Marshal(map[string]any{"tenantId": "tenant-a", "deploymentId": "deployment-a", "environmentId": "prod", "nodeId": "node-a", "releaseId": "release-a", "revision": 1, "sourceSnapshot": json.RawMessage(source), "accountId": "account-a", "natsEndpoint": "nats://nats.example:4222", "natsResourceRef": "site-resource://site/nats", "natsCredentialSecretRef": "secret://site/nats"})
	if err != nil {
		t.Fatal(err)
	}
	return request
}

func handlerSHA(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func serviceTestNotFound() error {
	return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "项目快照不存在")
}
