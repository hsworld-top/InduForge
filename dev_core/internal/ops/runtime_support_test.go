package ops

import "testing"

func TestResolveRuntimeSupportResourcesRejectsMissingReferences(t *testing.T) {
	refs := map[string]map[string]string{"nats_jetstream": {"service": "nats", "resourceRef": "site-resource://env/nats", "credentialSecretRef": "secret://env/nats", "secretNamespace": "runtime", "secretName": "nats", "secretKey": "token"}, "if_history": {"service": "postgres:5432", "resourceRef": "site-resource://env/postgres", "dsnSecretRef": "secret://env/postgres", "schemaPrefix": "runtime_", "database": "induforge_runtime", "adminUser": "postgres", "secretNamespace": "runtime", "secretName": "postgres", "secretKey": "password"}}
	if _, err := ResolveRuntimeSupportResources(refs); err != nil {
		t.Fatal(err)
	}
	if resolved, err := ResolveRuntimeSupportResources(refs); err != nil || resolved.StateStoreSchema != runtimeStateSchema || resolved.StateStoreDatabase != "" {
		t.Fatalf("state contract not server derived: %#v %v", resolved, err)
	}
	delete(refs["if_history"], "adminUser")
	if _, err := ResolveRuntimeSupportResources(refs); err == nil {
		t.Fatal("缺少 state store 必须预检失败")
	}
}
