package ops

import "testing"

func TestResolveRuntimeSupportResourcesRejectsMissingReferences(t *testing.T) {
	refs := map[string]map[string]string{"nats_jetstream": {"service": "nats", "resourceRef": "site-resource://env/nats", "credentialSecretRef": "secret://env/nats"}, "if_history": {"dsnSecretRef": "secret://env/postgres", "schema": "runtime_engine"}}
	if _, err := ResolveRuntimeSupportResources(refs); err != nil {
		t.Fatal(err)
	}
	delete(refs["if_history"], "schema")
	if _, err := ResolveRuntimeSupportResources(refs); err == nil {
		t.Fatal("缺少 state store 必须预检失败")
	}
}
