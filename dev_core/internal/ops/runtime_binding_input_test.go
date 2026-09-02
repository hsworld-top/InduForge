package ops

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildRuntimeBindingInputIsDeterministicAndRoleScoped(t *testing.T) {
	context := validRuntimeContext()
	for _, role := range []string{ServiceCompute, ServiceAlarm} {
		workload := ProjectWorkload{EnvironmentID: testEnvironmentID, DeploymentID: context.DeploymentID, ServiceID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", NodeID: testNodeID, Engine: role, ReleaseID: testVersionID, Generation: 3}
		first, err := BuildRuntimeBindingInput(workload, context)
		if err != nil {
			t.Fatalf("%s input: %v", role, err)
		}
		second, _ := BuildRuntimeBindingInput(workload, context)
		if string(first) != string(second) {
			t.Fatalf("%s input not deterministic", role)
		}
		var value map[string]any
		if json.Unmarshal(first, &value) != nil || value["schemaVersion"] != runtimeBindingInputVersion || value["runtimeArtifactSha256"] == "" {
			t.Fatalf("%s input contract incomplete: %s", role, first)
		}
		if value["runtimeArtifactPath"] != "/opt/induforge/release/"+runtimeArtifactFile {
			t.Fatalf("%s runtime artifact path = %v", role, value["runtimeArtifactPath"])
		}
		binding := value["binding"].(map[string]any)
		if binding["role"] != role || !strings.HasPrefix(binding["accountId"].(string), "if-") || strings.Contains(string(first), "producerAssignments") || strings.Contains(string(first), "computeProducers") {
			t.Fatalf("%s input leaks derived fields: %s", role, first)
		}
		_, sandbox := binding["computeSandbox"]
		if sandbox != (role == ServiceCompute) {
			t.Fatalf("%s sandbox role rule invalid: %s", role, first)
		}
	}
}

func TestBuildRuntimeBindingInputRejectsMissingArtifactOrSupport(t *testing.T) {
	context := validRuntimeContext()
	workload := ProjectWorkload{EnvironmentID: testEnvironmentID, DeploymentID: context.DeploymentID, ServiceID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", NodeID: testNodeID, Engine: ServiceCompute, ReleaseID: testVersionID, Generation: 1}
	context.Release.Manifest = []byte(`{"artifacts":{}}`)
	if _, err := BuildRuntimeBindingInput(workload, context); err == nil {
		t.Fatal("missing runtime artifact accepted")
	}
	context = validRuntimeContext()
	context.Support.NATSResourceRef = ""
	if _, err := BuildRuntimeBindingInput(workload, context); err == nil {
		t.Fatal("missing support reference accepted")
	}
}

func TestRuntimeStateSlotUsesFixedSchemaAndStableDatabaseName(t *testing.T) {
	context := validRuntimeContext()
	workload := ProjectWorkload{EnvironmentID: testEnvironmentID, DeploymentID: context.DeploymentID, ServiceID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", NodeID: testNodeID, Engine: ServiceCompute, ReleaseID: testVersionID, Generation: 1}
	raw, err := BuildRuntimeBindingInput(workload, context)
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err = json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	state := value["binding"].(map[string]any)["stateStore"].(map[string]any)
	if state["schema"] != runtimeStateSchema {
		t.Fatalf("schema = %v", state["schema"])
	}
	first := runtimeStateDatabaseName(context.ProjectID, context.EnvironmentID)
	if !strings.HasPrefix(first, "ifrt_") || len(first) > 63 || first != runtimeStateDatabaseName(context.ProjectID, context.EnvironmentID) {
		t.Fatalf("database name invalid: %s", first)
	}
	dev := context
	dev.Mode = "development"
	if first != runtimeStateDatabaseName(dev.ProjectID, dev.EnvironmentID) {
		t.Fatal("mode changed runtime state slot")
	}
	if first == runtimeStateDatabaseName("another-project", context.EnvironmentID) {
		t.Fatal("projects share runtime state slot")
	}
}
