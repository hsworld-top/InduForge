package ops

import "testing"

func TestResolveRuntimeSupportResourcesRejectsMissingReferences(t *testing.T) {
	refs := map[string]map[string]string{"nats_jetstream": {"service": "nats", "resourceRef": "site-resource://env/nats", "credentialSecretRef": "secret://env/nats"}, "if_history": {"dsnSecretRef": "secret://env/postgres", "schema": "runtime_engine"}}
	if _, err := ResolveRuntimeSupportResources(refs, false); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveRuntimeSupportResources(refs, true); err == nil {
		t.Fatal("缺少 sandbox 必须预检失败")
	}
}
