package service

import (
	"context"
	"testing"
	"testing/fstest"
	"time"

	"github.com/indu-forge/data_service/internal/collectorprotocol"
	"github.com/indu-forge/data_service/internal/repository"
	collectorsecurity "github.com/indu-forge/data_service/internal/security"
)

func TestCollectorServiceCreateSeparatesAndEncryptsSecrets(t *testing.T) {
	store := &fakeCollectorConnectionStore{}
	service := newCollectorServiceWithSecretSchema(t, store)
	created, err := service.CreateConnection(context.Background(), "550e8400-e29b-41d4-a716-446655440000", "550e8400-e29b-41d4-a716-446655440001", CreateCollectorConnectionInput{
		Name: "测试连接", DriverID: "test.driver",
		Config: map[string]any{"host": "127.0.0.1"}, Secrets: map[string]string{"password": "hidden"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := store.created.Config["password"]; exists {
		t.Fatal("secret leaked into config")
	}
	if len(store.created.Secrets) != 1 || string(store.created.Secrets[0].Value) == "hidden" {
		t.Fatalf("secret was not encrypted: %#v", store.created.Secrets)
	}
	if !created.SecretStatus["password"] {
		t.Fatal("secret status missing")
	}
	if created.ConfigurationState != "ready" || created.Config == nil || created.Metadata == nil || created.DefaultAcquisition == nil {
		t.Fatalf("collector response defaults are incomplete: %#v", created)
	}
}

func TestToCollectorConnectionNormalizesNilMaps(t *testing.T) {
	value := toCollectorConnection(repository.CollectorConnectionRecord{CreatedAt: time.Now(), UpdatedAt: time.Now()})
	if value.Config == nil || value.Metadata == nil || value.DefaultAcquisition == nil || value.SecretStatus == nil {
		t.Fatalf("collector response contains nil maps: %#v", value)
	}
}

func TestCollectorServiceCreateRejectsInvalidConfig(t *testing.T) {
	service := newCollectorServiceWithSecretSchema(t, &fakeCollectorConnectionStore{})
	_, err := service.CreateConnection(context.Background(), "550e8400-e29b-41d4-a716-446655440000", "550e8400-e29b-41d4-a716-446655440001", CreateCollectorConnectionInput{
		Name: "测试连接", DriverID: "test.driver", Config: map[string]any{"host": 1}, Secrets: map[string]string{"password": "hidden"},
	})
	if err == nil {
		t.Fatal("expected schema validation error")
	}
}

func newCollectorServiceWithSecretSchema(t *testing.T, store CollectorConnectionStore) *CollectorService {
	t.Helper()
	manifest := `{"protocolFamily":"test","driverId":"test.driver","driverVersion":"1.0.0","schemaVersion":1,"displayName":"Test","category":"plc","transports":["tcp"],"operations":[],"dataTypes":["bool"],"acquisitionModes":["polling"],"platforms":{"devAgent":[],"runtime":[]}}`
	connectionSchema := `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","additionalProperties":false,"required":["host","password"],"properties":{"host":{"type":"string"},"password":{"type":"string","minLength":1,"x-induforge-secret":true}}}`
	catalog, err := collectorprotocol.LoadCatalog(fstest.MapFS{
		"test.driver/manifest.json": {Data: []byte(manifest)}, "test.driver/connection.schema.json": {Data: []byte(connectionSchema)},
		"test.driver/address.schema.json": {Data: []byte(`{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","properties":{}}`)},
		"test.driver/ui.schema.json":      {Data: []byte(`{}`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	cipher, err := collectorsecurity.NewCollectorSecretCipher(make([]byte, 32), "v1")
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewCollectorService(store, catalog, cipher)
	if err != nil {
		t.Fatal(err)
	}
	return service
}

type fakeCollectorConnectionStore struct {
	created repository.CreateCollectorConnectionParams
}

func (f *fakeCollectorConnectionStore) ListConnections(context.Context, string, repository.CollectorConnectionListFilter) ([]repository.CollectorConnectionRecord, int, error) {
	return nil, 0, nil
}
func (f *fakeCollectorConnectionStore) GetConnection(context.Context, string, string) (*repository.CollectorConnectionRecord, error) {
	return nil, nil
}
func (f *fakeCollectorConnectionStore) CreateConnection(_ context.Context, params repository.CreateCollectorConnectionParams) (*repository.CollectorConnectionRecord, error) {
	f.created = params
	status := map[string]bool{}
	for _, secret := range params.Secrets {
		status[secret.Key] = true
	}
	return &repository.CollectorConnectionRecord{ID: params.ID, ProjectID: params.ProjectID, Name: params.Name, Code: params.Code, ProtocolFamily: params.ProtocolFamily, DriverID: params.DriverID, DriverVersion: params.DriverVersion, SchemaVersion: params.SchemaVersion, Config: params.Config, Metadata: params.Metadata, SecretStatus: status, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
}
func (f *fakeCollectorConnectionStore) UpdateConnection(context.Context, repository.UpdateCollectorConnectionParams) (*repository.CollectorConnectionRecord, error) {
	return nil, nil
}
func (f *fakeCollectorConnectionStore) DeleteConnection(context.Context, string, string) error {
	return nil
}
