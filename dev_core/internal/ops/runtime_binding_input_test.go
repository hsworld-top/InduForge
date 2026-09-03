package ops

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildRuntimeBindingInputIsDeterministicAndRoleScoped(t *testing.T) {
	context := validRuntimeContext()
	// 首次调和时两条服务均可能还是 pending；拓扑必须按 desired services
	// 完整下发，不能等待 observed running。
	context.RuntimeEngines = []string{ServiceAlarm, ServiceCompute}
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
		if binding["role"] != role || binding["fencingEpoch"] != float64(3) || !strings.HasPrefix(binding["accountId"].(string), "if-") || strings.Contains(string(first), "producerAssignments") || strings.Contains(string(first), "computeProducers") {
			t.Fatalf("%s input leaks derived fields: %s", role, first)
		}
		_, sandbox := binding["computeSandbox"]
		if sandbox != (role == ServiceCompute) {
			t.Fatalf("%s sandbox role rule invalid: %s", role, first)
		}
		endpoint, hasEndpoint := binding["computeSandboxEndpoint"]
		if role == ServiceCompute {
			if !hasEndpoint || endpoint != "http://127.0.0.1:18103" || strings.Contains(endpoint.(string), "compute-sandbox") {
				t.Fatalf("compute sandbox must use same-Pod loopback endpoint: %s", first)
			}
		} else if hasEndpoint {
			t.Fatalf("%s must not receive compute sandbox endpoint: %s", role, first)
		}
		jetStream := binding["jetStream"].(map[string]any)
		if got := len(jetStream["topologyConsumers"].([]any)); got != 5 {
			t.Fatalf("%s 首次双角色部署完整 topology consumers=%d", role, got)
		}
		wantActive := 2
		if role == ServiceCompute {
			wantActive = 3
		}
		if got := len(jetStream["consumers"].([]any)); got != wantActive {
			t.Fatalf("%s active consumers=%d", role, got)
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

func TestBuildRuntimeBindingInputIncludesCollectorProducerArtifact(t *testing.T) {
	context := validRuntimeContext()
	context.RuntimeEngines = []string{ServiceCompute, ServiceAlarm}
	context.CollectorSourceSnapshot = json.RawMessage(`{"schemaVersion":"collector-runtime-artifact.v1"}`)
	context.Release.Manifest, _ = json.Marshal(map[string]any{
		"schemaVersion": "2.0",
		"artifacts": map[string]any{
			"runtime":   map[string]any{"file": runtimeArtifactFile, "checksum": "sha256:" + strings.Repeat("a", 64)},
			"collector": map[string]any{"file": collectorArtifactFile, "checksum": "sha256:" + strings.Repeat("b", 64)},
		},
	})
	workload := ProjectWorkload{EnvironmentID: testEnvironmentID, DeploymentID: context.DeploymentID, ServiceID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", NodeID: testNodeID, Engine: ServiceCompute, ReleaseID: testVersionID, Generation: 3}
	raw, err := BuildRuntimeBindingInput(workload, context)
	if err != nil {
		t.Fatal(err)
	}
	var input map[string]any
	if json.Unmarshal(raw, &input) != nil || input["collectorArtifactPath"] != "/opt/induforge/release/"+collectorArtifactFile || input["collectorArtifactSha256"] != "sha256:"+strings.Repeat("b", 64) {
		t.Fatalf("采集 producer 制品未下发: %s", raw)
	}
}

func TestBuildRuntimeBindingInputMapsBaseToWriter(t *testing.T) {
	context := validRuntimeContext()
	context.RuntimeEngines = []string{ServiceBase, ServiceCompute, ServiceAlarm}
	workload := ProjectWorkload{EnvironmentID: testEnvironmentID, DeploymentID: context.DeploymentID, ServiceID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", NodeID: testNodeID, Engine: ServiceBase, ReleaseID: testVersionID, Generation: 3}
	raw, err := BuildRuntimeBindingInput(workload, context)
	if err != nil {
		t.Fatal(err)
	}
	var input map[string]any
	if json.Unmarshal(raw, &input) != nil {
		t.Fatal("运行绑定不是 JSON")
	}
	binding := input["binding"].(map[string]any)
	if binding["role"] != "writer" {
		t.Fatalf("基础引擎内部角色=%v", binding["role"])
	}
	consumers := binding["jetStream"].(map[string]any)["consumers"].([]any)
	if len(consumers) != 2 || consumers[0].(map[string]any)["consumerKey"] != "writer-raw-v1" || consumers[1].(map[string]any)["consumerKey"] != "writer-derived-v1" {
		t.Fatalf("writer consumer 不完整: %s", raw)
	}
}

func TestBuildRuntimeBindingInputFreezesServiceGenerationAsFencingEpoch(t *testing.T) {
	context := validRuntimeContext()
	workload := ProjectWorkload{EnvironmentID: testEnvironmentID, DeploymentID: context.DeploymentID, ServiceID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", NodeID: testNodeID, Engine: ServiceAlarm, ReleaseID: testVersionID, Generation: 3}
	first, err := BuildRuntimeBindingInput(workload, context)
	if err != nil {
		t.Fatal(err)
	}
	workload.Generation = 4
	next, err := BuildRuntimeBindingInput(workload, context)
	if err != nil {
		t.Fatal(err)
	}
	var firstDoc, nextDoc map[string]any
	if json.Unmarshal(first, &firstDoc) != nil || json.Unmarshal(next, &nextDoc) != nil {
		t.Fatal("运行绑定输入不是 JSON")
	}
	firstBinding, nextBinding := firstDoc["binding"].(map[string]any), nextDoc["binding"].(map[string]any)
	if firstBinding["fencingEpoch"] != float64(3) || nextBinding["fencingEpoch"] != float64(4) {
		t.Fatalf("fencingEpoch=%v/%v", firstBinding["fencingEpoch"], nextBinding["fencingEpoch"])
	}
	if firstBinding["instanceId"] == nextBinding["instanceId"] {
		t.Fatal("instanceId 必须保留 generation 诊断区分")
	}
}

func TestRuntimeAccountAndStreamsStayDeploymentStableAcrossRolesAndUpdates(t *testing.T) {
	context := validRuntimeContext()
	context.RuntimeEngines = []string{ServiceCompute, ServiceAlarm}
	wantAccount := runtimeAccountID(context.ProjectID, context.DeploymentID)
	var wantStreams map[string]any
	for _, role := range []string{ServiceBase, ServiceCollector, ServiceCompute, ServiceAlarm} {
		if role == ServiceCollector {
			// collector receives the same value through its bundle request; its
			// source of truth is asserted by the reconciler test below.
			continue
		}
		for _, generation := range []int64{1, 24} {
			workload := ProjectWorkload{EnvironmentID: context.EnvironmentID, DeploymentID: context.DeploymentID, ServiceID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", NodeID: testNodeID, Engine: role, ReleaseID: context.Release.ID, Generation: generation}
			for _, mode := range []string{"development", "production"} {
				context.Mode = mode
				raw, err := BuildRuntimeBindingInput(workload, context)
				if err != nil {
					t.Fatalf("%s generation=%d mode=%s: %v", role, generation, mode, err)
				}
				var doc map[string]any
				if err := json.Unmarshal(raw, &doc); err != nil {
					t.Fatal(err)
				}
				binding := doc["binding"].(map[string]any)
				if binding["accountId"] != wantAccount {
					t.Fatalf("%s generation=%d mode=%s account=%v want=%s", role, generation, mode, binding["accountId"], wantAccount)
				}
				streams := binding["jetStream"].(map[string]any)
				if wantStreams == nil {
					wantStreams = map[string]any{"dataRawStream": streams["dataRawStream"], "dataDerivedStream": streams["dataDerivedStream"], "eventStream": streams["eventStream"], "commandStream": streams["commandStream"], "deadLetterStream": streams["deadLetterStream"]}
				} else {
					for name, want := range wantStreams {
						if streams[name] != want {
							t.Fatalf("%s generation=%d mode=%s %s=%v want=%v", role, generation, mode, name, streams[name], want)
						}
					}
				}
			}
		}
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
