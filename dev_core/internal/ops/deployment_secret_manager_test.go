package ops

import (
	"context"
	"strings"
	"testing"
)

type memorySecretClient struct {
	values  map[string]map[string]string
	applies int
}

func (m *memorySecretClient) GetSecret(_ context.Context, ns, name string) (map[string]string, bool, error) {
	value, ok := m.values[ns+"/"+name]
	copy := map[string]string{}
	for k, v := range value {
		copy[k] = v
	}
	return copy, ok, nil
}
func (m *memorySecretClient) ApplySecret(_ context.Context, ns, name string, data map[string]string) error {
	copy := map[string]string{}
	for k, v := range data {
		copy[k] = v
	}
	m.values[ns+"/"+name] = copy
	m.applies++
	return nil
}
func (m *memorySecretClient) DeleteSecret(_ context.Context, ns, name string) error {
	delete(m.values, ns+"/"+name)
	return nil
}

func TestDeploymentSecretManagerBuildsResolverFilesAndRefreshesSources(t *testing.T) {
	client := &memorySecretClient{values: map[string]map[string]string{"runtime/nats": {"token": "source-token"}, "runtime/postgres": {"password": "source-password"}}}
	manager := NewDeploymentSecretManager(client)
	support := RuntimeSupportResources{StateStoreEndpoint: "postgres.runtime.svc:5432", StateStoreDatabase: "induforge_runtime", StateStoreSchema: "runtime", StateStoreAdminUser: "postgres", NATSCredentialSource: KubernetesSecretSource{Namespace: "runtime", Name: "nats", Keys: map[string]string{"credential": "token"}}, StateStoreCredentialSource: KubernetesSecretSource{Namespace: "runtime", Name: "postgres", Keys: map[string]string{"password": "password"}}}
	name, err := manager.Ensure(context.Background(), "project", "99999999-9999-4999-8999-999999999999", "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", ServiceCompute, support)
	if err != nil {
		t.Fatal(err)
	}
	data := client.values["project/"+name]
	for _, key := range []string{resolverNATSFile, resolverPostgresFile, bootstrapPostgresFile, resolverSandboxFile, "runtime-db-username", "runtime-db-password", "sandbox-token"} {
		if data[key] == "" {
			t.Fatalf("missing target key %q", key)
		}
	}
	if !strings.Contains(data[resolverNATSFile], "source-token") || !strings.Contains(data[resolverPostgresFile], data["runtime-db-username"]) || strings.Contains(data[resolverPostgresFile], "source-password") || !strings.Contains(data[bootstrapPostgresFile], "source-password") || !strings.Contains(data[bootstrapPostgresFile], `"maintenanceDsn"`) || !strings.Contains(data[bootstrapPostgresFile], "/postgres?") || !strings.Contains(data[bootstrapPostgresFile], `"schema":"runtime_engine"`) || !strings.Contains(data[bootstrapPostgresFile], runtimeStateDatabaseName("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb")) {
		t.Fatal("resolver files missing source values")
	}
	password, applies := data["runtime-db-password"], client.applies
	if _, err = manager.Ensure(context.Background(), "project", "99999999-9999-4999-8999-999999999999", "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", ServiceAlarm, support); err != nil || client.applies != applies {
		t.Fatalf("stable alarm reuse failed: %v applies=%d", err, client.applies)
	}
	if client.values["project/"+name]["runtime-db-password"] != password {
		t.Fatal("stable generated password changed")
	}
	client.values["runtime/nats"]["token"] = "rotated-token"
	if _, err = manager.Ensure(context.Background(), "project", "99999999-9999-4999-8999-999999999999", "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", ServiceAlarm, support); err != nil || client.applies != applies+1 || !strings.Contains(client.values["project/"+name][resolverNATSFile], "rotated-token") {
		t.Fatalf("derived source refresh failed: %v", err)
	}
}

func TestDeploymentSecretManagerRejectsMissingSourceKeyWithoutLeakingValue(t *testing.T) {
	client := &memorySecretClient{values: map[string]map[string]string{"runtime/nats": {}, "runtime/postgres": {"password": "secret-password"}}}
	manager := NewDeploymentSecretManager(client)
	_, err := manager.Ensure(context.Background(), "project", "99999999-9999-4999-8999-999999999999", "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", ServiceAlarm, RuntimeSupportResources{StateStoreEndpoint: "postgres:5432", StateStoreAdminUser: "postgres", NATSCredentialSource: KubernetesSecretSource{Namespace: "runtime", Name: "nats", Keys: map[string]string{"credential": "token"}}, StateStoreCredentialSource: KubernetesSecretSource{Namespace: "runtime", Name: "postgres", Keys: map[string]string{"password": "password"}}})
	if err == nil || strings.Contains(err.Error(), "secret-password") || client.applies != 0 {
		t.Fatalf("missing source was not safely rejected: %v", err)
	}
}

func TestBuildResolverSecretFilesKeepsResolverShapeInMemory(t *testing.T) {
	files, err := BuildResolverSecretFiles("environment-token", "postgres://runtime")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(files[resolverNATSFile], `"authType":"token"`) || !strings.Contains(files[resolverPostgresFile], `"schemaVersion":"postgres-dsn.v1"`) {
		t.Fatalf("resolver file shapes invalid")
	}
}

func TestDeploymentSecretManagerEnsuresCollectorFilesWithoutOverwritingOtherKeys(t *testing.T) {
	deploymentID := "99999999-9999-4999-8999-999999999999"
	name, err := collectorBundleSecretName(deploymentID)
	if err != nil {
		t.Fatal(err)
	}
	client := &memorySecretClient{values: sourceSecrets()}
	client.values["project/"+name] = map[string]string{"runtime-db-password": "preserve", "connection-bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb.json": "obsolete"}
	manager := NewDeploymentSecretManager(client)
	support := RuntimeSupportResources{NATSCredentialSource: KubernetesSecretSource{Namespace: "runtime", Name: "nats", Keys: map[string]string{"credential": "token"}}}
	_, err = manager.EnsureCollector(context.Background(), "project", deploymentID, support, map[string][]byte{"connection-aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa.json": []byte(`{"host":"modbus"}`)})
	if err != nil {
		t.Fatal(err)
	}
	files := client.values["project/"+name]
	if files["runtime-db-password"] != "preserve" || !strings.Contains(files[resolverNATSFile], "test-token") || files["connection-aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa.json"] == "" || files["connection-bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb.json"] != "" {
		t.Fatalf("collector secret was not controlled per key: %#v", files)
	}
}

func TestDeploymentSecretManagerBuildsBaseRuntimeAPIFilesWithoutBootstrapLeak(t *testing.T) {
	client := &memorySecretClient{values: map[string]map[string]string{"runtime/nats": {"token": "environment-token"}, "runtime/postgres": {"password": "admin-password"}}}
	manager := NewDeploymentSecretManager(client)
	support := RuntimeSupportResources{StateStoreEndpoint: "postgres.runtime.svc:5432", StateStoreAdminUser: "postgres", NATSCredentialSource: KubernetesSecretSource{Namespace: "runtime", Name: "nats", Keys: map[string]string{"credential": "token"}}, StateStoreCredentialSource: KubernetesSecretSource{Namespace: "runtime", Name: "postgres", Keys: map[string]string{"password": "password"}}}
	name, err := manager.Ensure(context.Background(), "project", "99999999-9999-4999-8999-999999999999", "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", ServiceBase, support)
	if err != nil {
		t.Fatal(err)
	}
	data := client.values["project/"+name]
	for _, key := range []string{runtimeAPINATSFile, runtimeAPIPostgresFile, runtimeAPITokensFile} {
		if data[key] == "" {
			t.Fatalf("base Runtime API missing %s", key)
		}
	}
	if strings.Contains(data[runtimeAPIPostgresFile], "admin-password") || strings.Contains(data[runtimeAPITokensFile], data["runtime-api-token"]) || strings.Contains(data[runtimeAPITokensFile], "runtime-db-password") || data["sandbox-token"] != "" {
		t.Fatalf("base API Secret leaked admin/sandbox value: %#v", data)
	}
}

func TestRuntimePostgresDSNSeparatesUsersAndEscapesCredentials(t *testing.T) {
	support := RuntimeSupportResources{StateStoreEndpoint: "postgres.runtime.svc:5432", StateStoreDatabase: "induforge_runtime", StateStoreAdminUser: "admin"}
	admin, err := runtimePostgresDSN(support, "admin", "p@ss:/?")
	if err != nil || !strings.Contains(admin, "admin:p%40ss%3A%2F%3F@") {
		t.Fatalf("admin DSN escaping failed: %v %s", err, admin)
	}
	runtime, err := runtimePostgresDSN(support, "runtime_abc", "runtime-password")
	if err != nil || !strings.Contains(runtime, "runtime_abc:runtime-password@") || strings.Contains(runtime, "admin") {
		t.Fatalf("runtime DSN isolation failed: %v %s", err, runtime)
	}
}
