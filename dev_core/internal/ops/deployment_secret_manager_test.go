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
	client := &memorySecretClient{values: map[string]map[string]string{"runtime/nats": {"token": "source-token"}, "runtime/postgres": {"dsn": "postgres://source"}}}
	manager := NewDeploymentSecretManager(client)
	support := RuntimeSupportResources{NATSCredentialSource: KubernetesSecretSource{Namespace: "runtime", Name: "nats", Keys: map[string]string{"credential": "token"}}, StateStoreCredentialSource: KubernetesSecretSource{Namespace: "runtime", Name: "postgres", Keys: map[string]string{"dsn": "dsn"}}}
	name, err := manager.Ensure(context.Background(), "project", "99999999-9999-4999-8999-999999999999", ServiceCompute, support)
	if err != nil {
		t.Fatal(err)
	}
	data := client.values["project/"+name]
	for _, key := range []string{resolverNATSFile, resolverPostgresFile, resolverSandboxFile, "runtime-db-username", "runtime-db-password", "sandbox-token"} {
		if data[key] == "" {
			t.Fatalf("missing target key %q", key)
		}
	}
	if !strings.Contains(data[resolverNATSFile], "source-token") || !strings.Contains(data[resolverPostgresFile], "postgres://source") {
		t.Fatal("resolver files missing source values")
	}
	password, applies := data["runtime-db-password"], client.applies
	if _, err = manager.Ensure(context.Background(), "project", "99999999-9999-4999-8999-999999999999", ServiceAlarm, support); err != nil || client.applies != applies {
		t.Fatalf("stable alarm reuse failed: %v applies=%d", err, client.applies)
	}
	if client.values["project/"+name]["runtime-db-password"] != password {
		t.Fatal("stable generated password changed")
	}
	client.values["runtime/nats"]["token"] = "rotated-token"
	if _, err = manager.Ensure(context.Background(), "project", "99999999-9999-4999-8999-999999999999", ServiceAlarm, support); err != nil || client.applies != applies+1 || !strings.Contains(client.values["project/"+name][resolverNATSFile], "rotated-token") {
		t.Fatalf("derived source refresh failed: %v", err)
	}
}

func TestDeploymentSecretManagerRejectsMissingSourceKeyWithoutLeakingValue(t *testing.T) {
	client := &memorySecretClient{values: map[string]map[string]string{"runtime/nats": {}, "runtime/postgres": {"dsn": "secret-dsn"}}}
	manager := NewDeploymentSecretManager(client)
	_, err := manager.Ensure(context.Background(), "project", "99999999-9999-4999-8999-999999999999", ServiceAlarm, RuntimeSupportResources{NATSCredentialSource: KubernetesSecretSource{Namespace: "runtime", Name: "nats", Keys: map[string]string{"credential": "token"}}, StateStoreCredentialSource: KubernetesSecretSource{Namespace: "runtime", Name: "postgres", Keys: map[string]string{"dsn": "dsn"}}})
	if err == nil || strings.Contains(err.Error(), "secret-dsn") || client.applies != 0 {
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
