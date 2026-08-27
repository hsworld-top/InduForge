package sandbox

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInternalEndpointsRequireToken(t *testing.T) {
	config := Config{Token: strings.Repeat("a", 24)}
	handler := NewHandler(config)
	request := httptest.NewRequest(http.MethodGet, "/v1/capabilities", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestCapabilitiesReportsIsolationFailureReason(t *testing.T) {
	config := Config{
		NodeBinary: "/missing/node", PythonBinary: "/missing/python",
		Bubblewrap: "/missing/bwrap", Prlimit: "/missing/prlimit",
	}
	request := httptest.NewRequest(http.MethodGet, "/v1/capabilities", nil)
	response := httptest.NewRecorder()
	capabilitiesHandler(config).ServeHTTP(response, request)
	var payload struct {
		Available bool   `json:"available"`
		Reason    string `json:"reason"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Available || strings.TrimSpace(payload.Reason) == "" {
		t.Fatalf("capabilities = %+v, want unavailable with reason", payload)
	}
}

func TestNodeRuntimeBlocksConstructorEscape(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node unavailable")
	}
	payload, _ := json.Marshal(map[string]any{"script": `return ({}).constructor.constructor("return process")().env;`, "input": map[string]any{}, "sdkContext": map[string]any{}})
	cmd := exec.Command(node, filepath.Join("..", "..", "runtime", "node_execute.js"))
	cmd.Stdin = bytes.NewReader(payload)
	if err := cmd.Run(); err == nil {
		t.Fatal("constructor escape must fail")
	}
}

func TestPythonRuntimeBlocksReflectionAndImport(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python unavailable")
	}
	for _, script := range []string{"import os\ndef main(argv, dp, ctx): return 1", "def main(argv, dp, ctx): return ().__class__.__base__.__subclasses__()"} {
		payload, _ := json.Marshal(map[string]any{"script": script, "input": map[string]any{}, "sdkContext": map[string]any{}})
		cmd := exec.Command(python, filepath.Join("..", "..", "runtime", "python_execute.py"))
		cmd.Stdin = bytes.NewReader(payload)
		if err := cmd.Run(); err == nil {
			t.Fatalf("python escape must fail: %s", script)
		}
	}
}

func TestNodeRuntimeInjectsDataPointObjects(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node unavailable")
	}
	payload, _ := json.Marshal(map[string]any{
		"script": `
const sample = temperature.read();
const write = manualPoint.set(1200);
return {
  sameObject: temperature === dp.temperature && temperature === ctx.points.temperature,
  sample,
  write,
  unsupported: temperature.history({ limit: 1 }),
  dataType: temperature.dataType,
};`,
		"input":      map[string]any{"trigger": map[string]any{"type": "manual"}},
		"sdkContext": runtimeSDKContextFixture(),
	})
	cmd := exec.Command(node, filepath.Join("..", "..", "runtime", "node_execute.js"))
	cmd.Stdin = bytes.NewReader(payload)
	output, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	assertRuntimeSDKOutput(t, output)
}

func TestPythonRuntimeInjectsDataPointObjects(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python unavailable")
	}
	payload, _ := json.Marshal(map[string]any{
		"script": `def main(argv, dp, ctx):
    sample = temperature.read()
    write = manualPoint.set(1200)
    return {
        "sameObject": temperature is dp["temperature"] and temperature is ctx.points.temperature,
        "sample": sample,
        "write": write,
        "unsupported": temperature.history({"limit": 1}),
        "dataType": temperature.dataType,
    }`,
		"input":      map[string]any{"trigger": map[string]any{"type": "manual"}},
		"sdkContext": runtimeSDKContextFixture(),
	})
	cmd := exec.Command(python, filepath.Join("..", "..", "runtime", "python_execute.py"))
	cmd.Stdin = bytes.NewReader(payload)
	output, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	assertRuntimeSDKOutput(t, output)
}

func runtimeSDKContextFixture() map[string]any {
	capabilities := map[string]bool{"get": true, "read": true, "peek": true}
	manualCapabilities := map[string]bool{"get": true, "read": true, "peek": true, "set": true}
	return map[string]any{
		"pointBindings": map[string]string{"temperature": "factory.temperature", "manualPoint": "manual.speed"},
		"datapoints": map[string]any{
			"factory.temperature": map[string]any{
				"id": "point-temperature", "path": "factory.temperature", "name": "温度", "dataType": "float64",
				"value": 26.5, "quality": "good", "timestamp": "2026-08-27T12:00:00Z", "status": "active",
				"attributes": map[string]string{"area": "A"}, "capabilities": capabilities,
			},
			"manual.speed": map[string]any{
				"id": "point-speed", "path": "manual.speed", "name": "速度", "dataType": "int64",
				"value": 1000, "quality": "good", "timestamp": "2026-08-27T12:00:00Z", "status": "active",
				"attributes": map[string]string{}, "capabilities": manualCapabilities,
			},
		},
		"metadata": map[string]any{"mode": "development"},
	}
}

func assertRuntimeSDKOutput(t *testing.T, output []byte) {
	t.Helper()
	var payload struct {
		Output struct {
			SameObject  bool           `json:"sameObject"`
			Sample      map[string]any `json:"sample"`
			Write       map[string]any `json:"write"`
			Unsupported map[string]any `json:"unsupported"`
			DataType    string         `json:"dataType"`
		} `json:"output"`
		SideEffects []map[string]any `json:"sideEffects"`
	}
	if err := json.Unmarshal(output, &payload); err != nil {
		t.Fatal(err)
	}
	if !payload.Output.SameObject || payload.Output.DataType != "float64" {
		t.Fatalf("unexpected point object: %+v", payload.Output)
	}
	if payload.Output.Sample["code"] != float64(0) || payload.Output.Unsupported["code"] != float64(40031) {
		t.Fatalf("unexpected SDK results: sample=%+v unsupported=%+v", payload.Output.Sample, payload.Output.Unsupported)
	}
	if payload.Output.Write["code"] != float64(0) || len(payload.SideEffects) != 1 || payload.SideEffects[0]["operation"] != "set" {
		t.Fatalf("unexpected side effects: write=%+v sideEffects=%+v", payload.Output.Write, payload.SideEffects)
	}
}

func TestBuildSandboxArgsRequiresIsolationAndNoNetwork(t *testing.T) {
	tempDir := t.TempDir()
	for _, name := range []string{"bwrap", "prlimit", "node_execute.js"} {
		if err := os.WriteFile(filepath.Join(tempDir, name), []byte("test"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	config := Config{Bubblewrap: filepath.Join(tempDir, "bwrap"), Prlimit: filepath.Join(tempDir, "prlimit"), RuntimeDir: tempDir, NodeBinary: "/usr/local/bin/node"}
	args, err := buildSandboxArgs(config, "js", "node_execute.js", "")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	for _, required := range []string{"--unshare-net", "--unshare-pid", "--clearenv", "--tmpfs /tmp", "--ro-bind " + tempDir + " /runtime", "--as=1073741824"} {
		if !strings.Contains(joined, required) {
			t.Fatalf("sandbox args missing %q: %s", required, joined)
		}
	}
	if strings.Contains(joined, "--share-net") {
		t.Fatalf("sandbox must not share network: %s", joined)
	}
	if strings.Contains(joined, "--nproc") {
		t.Fatalf("sandbox must rely on the container PID limit instead of host UID process counting: %s", joined)
	}
}

func TestValidateLanguageAndScript(t *testing.T) {
	if err := validateLanguageAndScript("js", "return 1"); err != nil {
		t.Fatal(err)
	}
	if err := validateLanguageAndScript("python", "def main(): pass"); err != nil {
		t.Fatal(err)
	}
	if err := validateLanguageAndScript("lua", "return 1"); err == nil {
		t.Fatal("expected unsupported language")
	}
	if err := validateLanguageAndScript("js", " "); err == nil {
		t.Fatal("expected empty script")
	}
}

func TestValidateDependencyMutation(t *testing.T) {
	projectID := "7797bf3a-df92-4594-9cff-9c918817bc9e"
	valid := []dependencyMutationRequest{
		{ProjectID: projectID, dependencySpec: dependencySpec{Language: "js", PackageName: "@scope/toolkit", ImportName: "@scope/toolkit/helpers", Version: "1.2.3"}},
		{ProjectID: projectID, dependencySpec: dependencySpec{Language: "python", PackageName: "python-dateutil", ImportName: "dateutil.parser", Version: "2.9.0.post0"}},
	}
	for index := range valid {
		if err := validateDependencyMutation(&valid[index], true); err != nil {
			t.Fatalf("valid dependency rejected: %v", err)
		}
	}
	latest := dependencyMutationRequest{ProjectID: projectID, dependencySpec: dependencySpec{Language: "js", PackageName: "dayjs"}}
	if err := validateDependencyMutation(&latest, false); err != nil {
		t.Fatalf("latest dependency request rejected: %v", err)
	}
	invalid := []dependencyMutationRequest{
		{ProjectID: "../../tmp", dependencySpec: dependencySpec{Language: "js", PackageName: "lodash", ImportName: "lodash", Version: "4.17.21"}},
		{ProjectID: projectID, dependencySpec: dependencySpec{Language: "js", PackageName: "lodash;touch-x", ImportName: "lodash", Version: "4.17.21"}},
		{ProjectID: projectID, dependencySpec: dependencySpec{Language: "js", PackageName: "lodash", ImportName: "lodash", Version: "latest"}},
		{ProjectID: projectID, dependencySpec: dependencySpec{Language: "python", PackageName: "requests", ImportName: "../../requests", Version: "2.32.5"}},
	}
	for index := range invalid {
		if err := validateDependencyMutation(&invalid[index], true); err == nil {
			t.Fatalf("invalid dependency accepted: %+v", invalid[index])
		}
	}
}

func TestBuildSandboxArgsMountsOnlySelectedProjectDependencies(t *testing.T) {
	tempDir := t.TempDir()
	for _, name := range []string{"bwrap", "prlimit", "node_execute.js"} {
		if err := os.WriteFile(filepath.Join(tempDir, name), []byte("test"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	projectID := "7797bf3a-df92-4594-9cff-9c918817bc9e"
	dependenciesDir := filepath.Join(tempDir, "dependencies")
	config := Config{Bubblewrap: filepath.Join(tempDir, "bwrap"), Prlimit: filepath.Join(tempDir, "prlimit"), RuntimeDir: tempDir, NodeBinary: "/usr/local/bin/node", DependenciesDir: dependenciesDir}
	args, err := buildSandboxArgs(config, "js", "node_execute.js", projectID)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	expected := "--ro-bind-try " + filepath.Join(dependenciesDir, projectID) + " /dependencies"
	if !strings.Contains(joined, expected) {
		t.Fatalf("sandbox args missing project dependency mount %q: %s", expected, joined)
	}
	if strings.Contains(joined, "--bind "+dependenciesDir+" ") {
		t.Fatalf("sandbox must not mount the writable dependency root: %s", joined)
	}
}
