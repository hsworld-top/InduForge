package sandbox

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
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

func TestLoadConfigSelectsValidatedRuntimeProfileAndBoundedConcurrency(t *testing.T) {
	t.Setenv("COMPUTE_SANDBOX_TOKEN", strings.Repeat("a", 24))
	t.Setenv("COMPUTE_SANDBOX_RUNTIME_PROFILE", runtimeRuntimeProfile)
	t.Setenv("COMPUTE_SANDBOX_SITE_ID", "site-a")
	t.Setenv("COMPUTE_SANDBOX_NODE_ID", "node-01")
	t.Setenv("COMPUTE_SANDBOX_EXECUTION_FORM", "")
	t.Setenv("COMPUTE_SANDBOX_MAX_CONCURRENT_EXECUTIONS", "2")
	t.Setenv("COMPUTE_SANDBOX_MAX_PIDS", "32")
	t.Setenv("COMPUTE_SANDBOX_DEPLOYMENT_ID", "deployment-a")
	t.Setenv("COMPUTE_SANDBOX_PROJECT_ID", "7797bf3a-df92-4594-9cff-9c918817bc9e")
	t.Setenv("COMPUTE_SANDBOX_ARTIFACT_DIGEST", "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	config, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.RuntimeProfile != runtimeRuntimeProfile || config.MaxConcurrentExecutions != 2 || config.MaxPIDs != 32 || config.SiteID != "site-a" || config.DeploymentID != "deployment-a" || config.ExecutionForm != defaultRuntimeExecutionForm {
		t.Fatalf("runtime config = %+v, want selected runtime profile and concurrency 2", config)
	}
	t.Setenv("COMPUTE_SANDBOX_MAX_CONCURRENT_EXECUTIONS", "0")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("unbounded concurrency configuration must be rejected")
	}
}

func TestLoadConfigRejectsInvalidRuntimeIdentity(t *testing.T) {
	t.Setenv("COMPUTE_SANDBOX_TOKEN", strings.Repeat("a", 24))
	t.Setenv("COMPUTE_SANDBOX_RUNTIME_PROFILE", runtimeRuntimeProfile)
	t.Setenv("COMPUTE_SANDBOX_SITE_ID", "")
	t.Setenv("COMPUTE_SANDBOX_NODE_ID", "")
	t.Setenv("COMPUTE_SANDBOX_DEPLOYMENT_ID", "deployment-a")
	t.Setenv("COMPUTE_SANDBOX_PROJECT_ID", "7797bf3a-df92-4594-9cff-9c918817bc9e")
	t.Setenv("COMPUTE_SANDBOX_ARTIFACT_DIGEST", "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	t.Setenv("COMPUTE_SANDBOX_MAX_PIDS", "64")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("runtime profile without siteId must be rejected")
	}
	t.Setenv("COMPUTE_SANDBOX_SITE_ID", "site-a")
	t.Setenv("COMPUTE_SANDBOX_EXECUTION_FORM", "bare-metal")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("invalid executionForm must be rejected")
	}
	t.Setenv("COMPUTE_SANDBOX_EXECUTION_FORM", "native-windows")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("runtime profile must reject native-windows executionForm")
	}
	t.Setenv("COMPUTE_SANDBOX_EXECUTION_FORM", "native-linux")
	t.Setenv("COMPUTE_SANDBOX_SITE_ID", "site with space")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("invalid siteId must be rejected")
	}
	t.Setenv("COMPUTE_SANDBOX_SITE_ID", "site-a")
	t.Setenv("COMPUTE_SANDBOX_NODE_ID", "node with space")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("invalid nodeId must be rejected")
	}
	t.Setenv("COMPUTE_SANDBOX_NODE_ID", "node-a")
	t.Setenv("COMPUTE_SANDBOX_DEPLOYMENT_ID", "")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("runtime profile without deploymentId must be rejected")
	}
	t.Setenv("COMPUTE_SANDBOX_DEPLOYMENT_ID", "deployment-a")
	t.Setenv("COMPUTE_SANDBOX_PROJECT_ID", "project-a")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("runtime profile with invalid projectId must be rejected")
	}
	t.Setenv("COMPUTE_SANDBOX_PROJECT_ID", "7797bf3a-df92-4594-9cff-9c918817bc9e")
	t.Setenv("COMPUTE_SANDBOX_ARTIFACT_DIGEST", "sha256:ABC")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("runtime profile with invalid artifactDigest must be rejected")
	}
	t.Setenv("COMPUTE_SANDBOX_ARTIFACT_DIGEST", "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	t.Setenv("COMPUTE_SANDBOX_MAX_PIDS", "0")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("invalid max pids must be rejected")
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
const businessValue = temperature.get();
console.log(temperature);
const write = manualPoint.set(1200);
return {
  sameObject: temperature === dp.temperature && temperature === ctx.points.temperature,
  sample,
  businessValue,
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

func TestNodeRuntimeExecutesTemperatureGetContract(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node unavailable")
	}
	context := runtimeSDKContextFixture()
	context["pointBindings"] = map[string]string{"temperature": "factory.temperature"}
	payload, _ := json.Marshal(map[string]any{
		"script": `const reading = await temperature.get();
if (reading.code !== 0) return reading;
const celsius = Number(reading.data.value);
return { result: { celsius, fahrenheit: celsius * 9 / 5 + 32 } };`,
		"input": map[string]any{}, "sdkContext": context,
	})
	cmd := exec.Command(node, filepath.Join("..", "..", "runtime", "node_execute.js"))
	cmd.Stdin = bytes.NewReader(payload)
	output, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Output struct {
			Result struct{ Celsius, Fahrenheit float64 } `json:"result"`
		} `json:"output"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatal(err)
	}
	if result.Output.Result.Celsius != 26.5 || result.Output.Result.Fahrenheit != 79.7 {
		t.Fatalf("unexpected output: %s", output)
	}
}

func TestPythonRuntimeInjectsDataPointObjects(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python unavailable")
	}
	payload, _ := json.Marshal(map[string]any{
		"script": `def main(argv, dp, ctx):
    sample = temperature.read()
    business_value = temperature.get()
    print(temperature)
    write = manualPoint.set(1200)
    return {
        "sameObject": temperature is dp["temperature"] and temperature is ctx.points.temperature,
        "sample": sample,
        "businessValue": business_value,
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
			SameObject    bool           `json:"sameObject"`
			Sample        map[string]any `json:"sample"`
			BusinessValue map[string]any `json:"businessValue"`
			Write         map[string]any `json:"write"`
			Unsupported   map[string]any `json:"unsupported"`
			DataType      string         `json:"dataType"`
		} `json:"output"`
		SideEffects []map[string]any `json:"sideEffects"`
		Logs        []string         `json:"logs"`
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
	businessData, ok := payload.Output.BusinessValue["data"].(map[string]any)
	if !ok || businessData["value"] != 26.5 {
		t.Fatalf("get result must return a sample envelope: %+v", payload.Output.BusinessValue)
	}
	if payload.Output.Write["code"] != float64(0) || len(payload.SideEffects) != 1 || payload.SideEffects[0]["operation"] != "set" {
		t.Fatalf("unexpected side effects: write=%+v sideEffects=%+v", payload.Output.Write, payload.SideEffects)
	}
	joinedLogs := strings.Join(payload.Logs, "\n")
	if strings.Contains(joinedLogs, "[object Object]") || !strings.Contains(joinedLogs, "factory.temperature") || !strings.Contains(joinedLogs, "capabilities") {
		t.Fatalf("datapoint log must be structured: %q", joinedLogs)
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
	if runtime.GOOS != "linux" {
		if err == nil {
			t.Fatal("non-Linux host must not build sandbox arguments")
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	for _, required := range []string{"/usr/bin/unshare --net --", "--unshare-pid", "--clearenv", "--tmpfs /tmp", "--ro-bind " + tempDir + " /runtime", "--as=1073741824"} {
		if !strings.Contains(joined, required) {
			t.Fatalf("sandbox args missing %q: %s", required, joined)
		}
	}
	if strings.Contains(joined, "--share-net") {
		t.Fatalf("sandbox must not share network: %s", joined)
	}
	if !strings.Contains(joined, "/usr/bin/unshare --net -- "+config.Bubblewrap+" ") {
		t.Fatalf("sandbox execution must invoke bubblewrap without host fallback: %s", joined)
	}
	if strings.Contains(joined, "--nproc") {
		t.Fatalf("sandbox must rely on the container PID limit instead of host UID process counting: %s", joined)
	}
	for _, forbidden := range []string{"--proc /proc", "--ro-bind /proc /proc"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("non-root sandbox must not require nested namespace privilege %q: %s", forbidden, joined)
		}
	}
	for _, required := range []string{"--cap-drop ALL --cap-add CAP_SETUID --cap-add CAP_SETGID --cap-add CAP_SETPCAP", "/usr/local/bin/compute-sandbox internal-exec"} {
		if !strings.Contains(joined, required) {
			t.Fatalf("sandbox must enforce child privilege boundary %q: %s", required, joined)
		}
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
	if runtime.GOOS != "linux" {
		if err == nil {
			t.Fatal("non-Linux host must not build sandbox arguments")
		}
		return
	}
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

func TestBuildSandboxArgsMountsRuntimeReleaseDependenciesOnly(t *testing.T) {
	tempDir := t.TempDir()
	for _, name := range []string{"bwrap", "prlimit", "node_execute.js"} {
		if err := os.WriteFile(filepath.Join(tempDir, name), []byte("test"), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	dependenciesDir := filepath.Join(tempDir, "release-dependencies")
	if err := os.Mkdir(dependenciesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	config := Config{RuntimeProfile: runtimeRuntimeProfile, Bubblewrap: filepath.Join(tempDir, "bwrap"), Prlimit: filepath.Join(tempDir, "prlimit"), RuntimeDir: tempDir, NodeBinary: "/usr/local/bin/node", DependenciesDir: dependenciesDir}
	projectID := "7797bf3a-df92-4594-9cff-9c918817bc9e"
	args, err := buildSandboxArgs(config, "js", "node_execute.js", projectID)
	if runtime.GOOS != "linux" {
		if err == nil {
			t.Fatal("non-Linux host must not build sandbox arguments")
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--ro-bind "+dependenciesDir+" /dependencies") {
		t.Fatalf("runtime must mount exact release dependency root: %s", joined)
	}
	if strings.Contains(joined, filepath.Join(dependenciesDir, projectID)) {
		t.Fatalf("runtime must not derive dependency path from request projectId: %s", joined)
	}
}

func TestCgroupPIDsLimitParsing(t *testing.T) {
	for _, test := range []struct {
		value string
		want  bool
	}{
		{"64\n", true},
		{"max", false},
		{"0", false},
		{"invalid", false},
	} {
		_, got := parseFinitePIDsMax(test.value)
		if got != test.want {
			t.Fatalf("parseFinitePIDsMax(%q) = %t, want %t", test.value, got, test.want)
		}
	}
	for _, test := range []struct {
		name  string
		files map[string]string
		want  int
		ok    bool
	}{
		{
			name: "v2 root 100 leaf 32", want: 32, ok: true,
			files: map[string]string{"/proc/self/cgroup": "0::/workload\n", "/sys/fs/cgroup/pids.max": "100\n", "/sys/fs/cgroup/workload/pids.max": "32\n"},
		},
		{
			name: "v2 root 32 leaf max", want: 32, ok: true,
			files: map[string]string{"/proc/self/cgroup": "0::/workload\n", "/sys/fs/cgroup/pids.max": "32\n", "/sys/fs/cgroup/workload/pids.max": "max\n"},
		},
		{
			name: "v2 all max", ok: false,
			files: map[string]string{"/proc/self/cgroup": "0::/workload\n", "/sys/fs/cgroup/pids.max": "max\n", "/sys/fs/cgroup/workload/pids.max": "max\n"},
		},
		{
			name: "v1 root 100 leaf 32", want: 32, ok: true,
			files: map[string]string{"/proc/self/cgroup": "7:pids:/workload\n", "/sys/fs/cgroup/pids/pids.max": "100\n", "/sys/fs/cgroup/pids/workload/pids.max": "32\n"},
		},
		{
			name: "v1 root 32 leaf max", want: 32, ok: true,
			files: map[string]string{"/proc/self/cgroup": "7:pids:/workload\n", "/sys/fs/cgroup/pids/pids.max": "32\n", "/sys/fs/cgroup/pids/workload/pids.max": "max\n"},
		},
		{
			name: "v1 all max", ok: false,
			files: map[string]string{"/proc/self/cgroup": "7:pids:/workload\n", "/sys/fs/cgroup/pids/pids.max": "max\n", "/sys/fs/cgroup/pids/workload/pids.max": "max\n"},
		},
	} {
		limit, ok := currentCgroupPIDsLimit(func(path string) ([]byte, error) {
			value, exists := test.files[path]
			if !exists {
				return nil, os.ErrNotExist
			}
			return []byte(value), nil
		})
		if ok != test.ok || limit != test.want {
			t.Fatalf("%s cgroup limit = %d, %t; want %d, %t", test.name, limit, ok, test.want, test.ok)
		}
	}
	if !pidsLimitWithinConfiguredMax(16, 16) || pidsLimitWithinConfiguredMax(17, 16) || pidsLimitWithinConfiguredMax(0, 16) {
		t.Fatal("finite cgroup pid limit must be bounded by configured maximum")
	}
}

func TestRuntimeDependenciesValidateFileModesAndDirectory(t *testing.T) {
	tempDir := t.TempDir()
	for _, name := range []string{"bwrap", "prlimit", "node", "python", "node_execute.js", "python_execute.py"} {
		mode := os.FileMode(0o700)
		if strings.HasSuffix(name, ".js") || strings.HasSuffix(name, ".py") {
			mode = 0o400
		}
		if err := os.WriteFile(filepath.Join(tempDir, name), []byte("test"), mode); err != nil {
			t.Fatal(err)
		}
	}
	dependenciesDir := filepath.Join(tempDir, "dependencies")
	if err := os.Mkdir(dependenciesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	config := Config{Bubblewrap: filepath.Join(tempDir, "bwrap"), Prlimit: filepath.Join(tempDir, "prlimit"), NodeBinary: filepath.Join(tempDir, "node"), PythonBinary: filepath.Join(tempDir, "python"), RuntimeDir: tempDir, DependenciesDir: dependenciesDir}
	_, reasons := runtimeDependenciesAvailable(config)
	for _, code := range []string{"BUBBLEWRAP_UNAVAILABLE", "PRLIMIT_UNAVAILABLE", "NODE_RUNTIME_UNAVAILABLE", "PYTHON_RUNTIME_UNAVAILABLE", "NODE_EXECUTOR_UNAVAILABLE", "PYTHON_EXECUTOR_UNAVAILABLE", "DEPENDENCIES_DIR_UNAVAILABLE"} {
		if containsReason(reasons, code) {
			t.Fatalf("valid dependency %s reported unavailable: %v", code, reasons)
		}
	}
	if err := os.Chmod(config.Bubblewrap, 0o600); err != nil {
		t.Fatal(err)
	}
	_, reasons = runtimeDependenciesAvailable(config)
	if !containsReason(reasons, "BUBBLEWRAP_UNAVAILABLE") {
		t.Fatalf("non-executable binary must be rejected: %v", reasons)
	}
	if err := os.Chmod(filepath.Join(tempDir, "node_execute.js"), 0o000); err != nil {
		t.Fatal(err)
	}
	_, reasons = runtimeDependenciesAvailable(config)
	if !containsReason(reasons, "NODE_EXECUTOR_UNAVAILABLE") {
		t.Fatalf("unreadable runtime script must be rejected: %v", reasons)
	}
	if err := os.Remove(dependenciesDir); err != nil {
		t.Fatal(err)
	}
	_, reasons = runtimeDependenciesAvailable(config)
	if !containsReason(reasons, "DEPENDENCIES_DIR_UNAVAILABLE") {
		t.Fatalf("missing dependency directory must be rejected: %v", reasons)
	}
}

func containsReason(reasons []string, wanted string) bool {
	for _, reason := range reasons {
		if reason == wanted {
			return true
		}
	}
	return false
}

func TestRuntimeProfileRequiresExecutionProvenance(t *testing.T) {
	config := Config{
		RuntimeProfile: runtimeRuntimeProfile, SiteID: "site-a", ExecutionForm: "k3s-workload", MaxConcurrentExecutions: 4, MaxPIDs: 64,
		DeploymentID: "deployment-a", ProjectID: "7797bf3a-df92-4594-9cff-9c918817bc9e",
		ArtifactDigest: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}
	valid := executeRequest{
		ExecutionID:    "127b9bda-15c6-4a16-aa02-d031199f37dc",
		DeploymentID:   "deployment-a",
		ProjectID:      "7797bf3a-df92-4594-9cff-9c918817bc9e",
		ComputeUnitID:  "ae26d169-e82b-470c-a2c8-249d89f2d2a4",
		ArtifactDigest: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}
	if err := validateRuntimeExecution(config, valid); err != nil {
		t.Fatalf("valid runtime request rejected: %v", err)
	}
	for _, field := range []string{"executionId", "deploymentId", "projectId", "computeUnitId", "artifactDigest"} {
		invalid := valid
		switch field {
		case "executionId":
			invalid.ExecutionID = ""
		case "deploymentId":
			invalid.DeploymentID = "deployment-b"
		case "projectId":
			invalid.ProjectID = "project-a"
		case "computeUnitId":
			invalid.ComputeUnitID = "compute-a"
		case "artifactDigest":
			invalid.ArtifactDigest = "sha256:not-a-digest"
		}
		if err := validateRuntimeExecution(config, invalid); err == nil {
			t.Fatalf("runtime request without valid %s was accepted", field)
		}
	}
	uppercaseDigest := valid
	uppercaseDigest.ArtifactDigest = "sha256:0123456789ABCDEF0123456789abcdef0123456789abcdef0123456789abcdef"
	if err := validateRuntimeExecution(config, uppercaseDigest); err == nil {
		t.Fatal("artifactDigest with uppercase hexadecimal must be rejected")
	}
	if err := validateRuntimeExecution(Config{RuntimeProfile: developmentRuntimeProfile}, executeRequest{}); err != nil {
		t.Fatalf("development profile must preserve legacy request shape: %v", err)
	}
}

func TestRuntimeProfileRejectsAllSideEffects(t *testing.T) {
	if err := validateExecutionResult(Config{RuntimeProfile: runtimeRuntimeProfile}, []any{map[string]any{"operation": "set"}}); err == nil {
		t.Fatal("runtime profile must reject side effects")
	}
	if err := validateExecutionResult(Config{RuntimeProfile: developmentRuntimeProfile}, []any{map[string]any{"operation": "set"}}); err != nil {
		t.Fatalf("development profile must retain side effect previews: %v", err)
	}
}

func TestExecutionGateRejectsOverload(t *testing.T) {
	gate := newExecutionGate(1)
	if !gate.tryAcquire() {
		t.Fatal("first execution slot must be available")
	}
	if gate.tryAcquire() {
		t.Fatal("bounded execution gate must reject overload")
	}
	gate.release()
	if !gate.tryAcquire() {
		t.Fatal("released execution slot must become available")
	}
	gate.release()
}

func TestRuntimeProfileRestrictsDependencyMutation(t *testing.T) {
	config := Config{Token: strings.Repeat("a", 24), RuntimeProfile: runtimeRuntimeProfile}
	handler := NewHandler(config)
	request := httptest.NewRequest(http.MethodPost, "/v1/dependencies/install", strings.NewReader(`{}`))
	request.Header.Set("Authorization", "Bearer "+config.Token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), "RUNTIME_PROFILE_RESTRICTED") {
		t.Fatalf("runtime dependency mutation = %d %s, want structured forbidden", response.Code, response.Body.String())
	}
}

func TestInvalidRuntimeProfileFailsClosed(t *testing.T) {
	config := Config{Token: strings.Repeat("a", 24), RuntimeProfile: "unexpected-profile"}
	handler := NewHandler(config)
	capabilities := httptest.NewRequest(http.MethodGet, "/v1/capabilities", nil)
	capabilities.Header.Set("Authorization", "Bearer "+config.Token)
	capabilitiesResponse := httptest.NewRecorder()
	handler.ServeHTTP(capabilitiesResponse, capabilities)
	var capabilitiesPayload struct {
		Available bool   `json:"available"`
		Reason    string `json:"reason"`
	}
	if err := json.Unmarshal(capabilitiesResponse.Body.Bytes(), &capabilitiesPayload); err != nil {
		t.Fatal(err)
	}
	if capabilitiesPayload.Available || capabilitiesPayload.Reason != "运行配置无效" {
		t.Fatalf("invalid profile capabilities must fail closed: %+v", capabilitiesPayload)
	}
	for _, path := range []string{"/v1/dependencies/install", "/v1/syntax-check"} {
		request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`))
		request.Header.Set("Authorization", "Bearer "+config.Token)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), "RUNTIME_PROFILE_RESTRICTED") {
			t.Fatalf("invalid profile %s = %d %s, want restricted", path, response.Code, response.Body.String())
		}
	}
	execute := httptest.NewRequest(http.MethodPost, "/v1/execute", strings.NewReader(`{"language":"js","script":"return 1;"}`))
	execute.Header.Set("Authorization", "Bearer "+config.Token)
	executeResponse := httptest.NewRecorder()
	handler.ServeHTTP(executeResponse, execute)
	if executeResponse.Code != http.StatusBadRequest || !strings.Contains(executeResponse.Body.String(), "RUNTIME_REQUEST_INVALID") {
		t.Fatalf("invalid profile execute = %d %s, want strict provenance rejection", executeResponse.Code, executeResponse.Body.String())
	}
	for _, path := range []string{"/health", "/api/v1/status"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusServiceUnavailable {
			t.Fatalf("invalid profile %s = %d %s, want unavailable status", path, response.Code, response.Body.String())
		}
		if path == "/health" && !strings.Contains(response.Body.String(), `"status":"DOWN"`) {
			t.Fatalf("invalid profile health must be down: %s", response.Body.String())
		}
		if path == "/api/v1/status" && !strings.Contains(response.Body.String(), "RUNTIME_PROFILE_INVALID") {
			t.Fatalf("invalid profile status must include a safe reason code: %s", response.Body.String())
		}
	}
}

func TestDevelopmentStatusNeverClaimsNativeWindowsExecution(t *testing.T) {
	if got := statusExecutionForm(Config{RuntimeProfile: developmentRuntimeProfile, ExecutionForm: "native-windows"}); got != "native-linux" {
		t.Fatalf("development status executionForm = %q, want native-linux", got)
	}
}

func TestRuntimeProfileCapabilitiesAreReadOnlyAndSyntaxIsDisabled(t *testing.T) {
	config := Config{Token: strings.Repeat("a", 24), RuntimeProfile: runtimeRuntimeProfile}
	capabilitiesRequest := httptest.NewRequest(http.MethodGet, "/v1/capabilities", nil)
	capabilitiesResponse := httptest.NewRecorder()
	capabilitiesHandler(config).ServeHTTP(capabilitiesResponse, capabilitiesRequest)
	var payload struct {
		Available bool     `json:"available"`
		Reason    string   `json:"reason"`
		SDK       []string `json:"sdk"`
	}
	if err := json.Unmarshal(capabilitiesResponse.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Available || payload.Reason != "运行配置无效" {
		t.Fatalf("runtime identity invalid capabilities must fail closed: %+v", payload)
	}
	for _, forbidden := range []string{"point.set", "point.refresh", "point.run", "point.execute", "point.publish"} {
		for _, capability := range payload.SDK {
			if capability == forbidden {
				t.Fatalf("runtime capabilities must not advertise %s: %v", forbidden, payload.SDK)
			}
		}
	}
	handler := NewHandler(config)
	syntaxRequest := httptest.NewRequest(http.MethodPost, "/v1/syntax-check", strings.NewReader(`{"language":"js","script":"return 1;"}`))
	syntaxRequest.Header.Set("Authorization", "Bearer "+config.Token)
	syntaxResponse := httptest.NewRecorder()
	handler.ServeHTTP(syntaxResponse, syntaxRequest)
	if syntaxResponse.Code != http.StatusForbidden || !strings.Contains(syntaxResponse.Body.String(), "RUNTIME_PROFILE_RESTRICTED") {
		t.Fatalf("runtime syntax check = %d %s, want restricted", syntaxResponse.Code, syntaxResponse.Body.String())
	}
}

func TestRuntimeIdentityInvalidCapabilitiesHealthAndExecuteFailClosed(t *testing.T) {
	config := Config{
		Token: strings.Repeat("a", 24), RuntimeProfile: runtimeRuntimeProfile, ExecutionForm: "k3s-workload", MaxPIDs: 64,
		DeploymentID: "deployment-a", ProjectID: "7797bf3a-df92-4594-9cff-9c918817bc9e", ArtifactDigest: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}
	handler := NewHandler(config)
	capabilities := httptest.NewRequest(http.MethodGet, "/v1/capabilities", nil)
	capabilities.Header.Set("Authorization", "Bearer "+config.Token)
	capabilitiesResponse := httptest.NewRecorder()
	handler.ServeHTTP(capabilitiesResponse, capabilities)
	if !strings.Contains(capabilitiesResponse.Body.String(), `"available":false`) || !strings.Contains(capabilitiesResponse.Body.String(), "运行配置无效") {
		t.Fatalf("invalid runtime identity capabilities must fail closed: %s", capabilitiesResponse.Body.String())
	}
	health := httptest.NewRequest(http.MethodGet, "/health", nil)
	healthResponse := httptest.NewRecorder()
	handler.ServeHTTP(healthResponse, health)
	if healthResponse.Code != http.StatusServiceUnavailable {
		t.Fatalf("invalid runtime identity health = %d, want unavailable", healthResponse.Code)
	}
	execute := httptest.NewRequest(http.MethodPost, "/v1/execute", strings.NewReader(`{"language":"js","script":"return 1;","executionId":"127b9bda-15c6-4a16-aa02-d031199f37dc","deploymentId":"deployment-a","projectId":"7797bf3a-df92-4594-9cff-9c918817bc9e","computeUnitId":"ae26d169-e82b-470c-a2c8-249d89f2d2a4","artifactDigest":"sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}`))
	execute.Header.Set("Authorization", "Bearer "+config.Token)
	executeResponse := httptest.NewRecorder()
	handler.ServeHTTP(executeResponse, execute)
	if executeResponse.Code != http.StatusBadRequest || !strings.Contains(executeResponse.Body.String(), "RUNTIME_REQUEST_INVALID") {
		t.Fatalf("invalid runtime identity execute must fail closed: %d %s", executeResponse.Code, executeResponse.Body.String())
	}
}

func TestExecuteRequestRejectsCallbackField(t *testing.T) {
	config := Config{Token: strings.Repeat("a", 24)}
	handler := NewHandler(config)
	request := httptest.NewRequest(http.MethodPost, "/v1/execute", strings.NewReader(`{"language":"js","script":"return 1;","callbackUrl":"https://example.test/callback"}`))
	request.Header.Set("Authorization", "Bearer "+config.Token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "请求格式无效") {
		t.Fatalf("callback field must be rejected before execution: %d %s", response.Code, response.Body.String())
	}
}

func TestExecuteRequestRejectsInvalidDependenciesBeforeExecution(t *testing.T) {
	config := Config{Token: strings.Repeat("a", 24)}
	handler := NewHandler(config)
	for _, dependencies := range []string{
		`[{"language":"js","packageName":"lodash","importName":"../../runtime/evil","version":"1.2.3"}]`,
		`[{"language":"python","packageName":"requests","importName":"requests","version":"2.32.0"}]`,
		`[{"language":"js","packageName":"lodash","importName":"lodash","version":"1.2.3"},{"language":"js","packageName":"lodash-es","importName":"lodash","version":"4.17.21"}]`,
		`[{}]`,
	} {
		body := `{"language":"js","script":"return 1;","dependencies":` + dependencies + `}`
		request := httptest.NewRequest(http.MethodPost, "/v1/execute", strings.NewReader(body))
		request.Header.Set("Authorization", "Bearer "+config.Token)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "dependencies") {
			t.Fatalf("invalid dependencies %s must be rejected before execution: %d %s", dependencies, response.Code, response.Body.String())
		}
	}
}

func TestFormalHealthAndStatusDoNotExposeSensitiveConfiguration(t *testing.T) {
	config := Config{
		RuntimeProfile: runtimeRuntimeProfile, Token: "very-secret-token-that-must-not-appear", SiteID: "site-a", NodeID: "node-01", ExecutionForm: "k3s-workload", MaxPIDs: 64,
		DeploymentID: "deployment-a", ProjectID: "7797bf3a-df92-4594-9cff-9c918817bc9e", ArtifactDigest: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}
	handler := NewHandler(config)
	for _, path := range []string{"/health", "/api/v1/status"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		var envelope struct {
			Code  int            `json:"code"`
			Msg   string         `json:"msg"`
			Data  map[string]any `json:"data"`
			ReqID string         `json:"reqId"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
			t.Fatalf("%s invalid envelope: %v", path, err)
		}
		if envelope.ReqID == "" || len(envelope.Data) == 0 || strings.Contains(response.Body.String(), config.Token) {
			t.Fatalf("%s must return a safe formal envelope: %s", path, response.Body.String())
		}
		if path == "/api/v1/status" && (envelope.Data["schemaVersion"] != "runtime-health-status.v1" || envelope.Data["siteId"] != "site-a" || envelope.Data["nodeId"] != "node-01" || envelope.Data["executionForm"] != "k3s-workload" || envelope.Data["deploymentId"] != "deployment-a" || envelope.Data["projectId"] != "7797bf3a-df92-4594-9cff-9c918817bc9e") {
			t.Fatalf("status must include runtime identity without sensitive values: %+v", envelope.Data)
		}
		if response.Code != http.StatusServiceUnavailable || envelope.Code == 0 {
			t.Fatalf("%s must report runtime dependency unavailable: http=%d code=%d", path, response.Code, envelope.Code)
		}
	}
}

func TestStatusUsesRuntimeHealthStatusSchemaAcrossProfiles(t *testing.T) {
	allowedFields := map[string]struct{}{
		"schemaVersion": {}, "componentRole": {}, "siteId": {}, "deploymentId": {}, "accountId": {}, "projectId": {}, "projectCode": {}, "deploymentStage": {},
		"executionForm": {}, "nodeId": {}, "processId": {}, "lifecycleState": {}, "healthState": {}, "version": {}, "startedAt": {}, "uptimeSeconds": {},
		"observedAt": {}, "lastError": {}, "reasonCode": {}, "businessFreshness": {}, "collector": {},
	}
	requiredFields := []string{"schemaVersion", "componentRole", "siteId", "executionForm", "lifecycleState", "healthState", "version", "startedAt", "uptimeSeconds", "observedAt"}
	runtimeConfig := Config{
		RuntimeProfile: runtimeRuntimeProfile, SiteID: "site-a", ExecutionForm: "k3s-workload", MaxPIDs: 64,
		DeploymentID: "deployment-a", ProjectID: "7797bf3a-df92-4594-9cff-9c918817bc9e", ArtifactDigest: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}
	for _, test := range []struct {
		name               string
		config             Config
		wantSiteID         string
		wantDeploymentData bool
		wantProjectData    bool
	}{
		{name: "development uses local identity by default", config: Config{}, wantSiteID: defaultDevelopmentSiteID},
		{name: "development uses configured identity", config: Config{SiteID: "dev-site"}, wantSiteID: "dev-site"},
		{name: "runtime keeps configured identity", config: runtimeConfig, wantSiteID: "site-a", wantDeploymentData: true, wantProjectData: true},
		{name: "invalid profile remains schema compliant", config: Config{RuntimeProfile: "unexpected-profile"}, wantSiteID: invalidRuntimeSiteID},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
			response := httptest.NewRecorder()
			NewHandler(test.config).ServeHTTP(response, request)
			var envelope struct {
				Data map[string]any `json:"data"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
				t.Fatal(err)
			}
			for field := range envelope.Data {
				if _, ok := allowedFields[field]; !ok {
					t.Fatalf("status returned field outside schema: %q in %+v", field, envelope.Data)
				}
			}
			for _, field := range requiredFields {
				if _, ok := envelope.Data[field]; !ok {
					t.Fatalf("status missing required field %q: %+v", field, envelope.Data)
				}
			}
			if envelope.Data["schemaVersion"] != "runtime-health-status.v1" || envelope.Data["siteId"] != test.wantSiteID {
				t.Fatalf("status schema identity = %+v", envelope.Data)
			}
			_, hasDeployment := envelope.Data["deploymentId"]
			_, hasProject := envelope.Data["projectId"]
			if hasDeployment != test.wantDeploymentData || hasProject != test.wantProjectData {
				t.Fatalf("optional runtime identity fields = %+v", envelope.Data)
			}
		})
	}
}

func TestNewHandlerNormalizesUnsafeDirectConfig(t *testing.T) {
	runtimeConfig := normalizeHandlerConfig(Config{
		RuntimeProfile: runtimeRuntimeProfile, SiteID: "site-a", ExecutionForm: "invalid-form", MaxConcurrentExecutions: 1000000, MaxPIDs: maxConfiguredPIDs + 1,
		DeploymentID: "deployment-a", ProjectID: "not-a-uuid", ArtifactDigest: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	})
	if !runtimeConfig.configurationInvalid || runtimeConfig.MaxConcurrentExecutions != defaultMaxConcurrentExecutions || runtimeConfig.MaxPIDs != defaultMaxPIDs {
		t.Fatalf("runtime direct config must fail closed with safe resource defaults: %+v", runtimeConfig)
	}
	if !isStrictRuntimeProfile(runtimeConfig) || strictRuntimeConfigurationValid(runtimeConfig) {
		t.Fatal("invalid runtime direct config must not become executable")
	}
	developmentConfig := normalizeHandlerConfig(Config{RuntimeProfile: developmentRuntimeProfile, MaxConcurrentExecutions: 1000000})
	if developmentConfig.MaxConcurrentExecutions != defaultMaxConcurrentExecutions {
		t.Fatalf("development concurrency = %d, want safe default", developmentConfig.MaxConcurrentExecutions)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	response := httptest.NewRecorder()
	NewHandler(runtimeConfig).ServeHTTP(response, request)
	var envelope struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	assertRuntimeStatusData(t, envelope.Data)
	if envelope.Data["executionForm"] != defaultRuntimeExecutionForm {
		t.Fatalf("invalid execution form leaked into status: %+v", envelope.Data)
	}
	if _, exists := envelope.Data["projectId"]; exists {
		t.Fatalf("invalid projectId must not be emitted: %+v", envelope.Data)
	}
}

func TestSetprivReadinessCheck(t *testing.T) {
	if isExecutableRegularFile(filepath.Join(t.TempDir(), "missing-setpriv")) {
		t.Fatal("missing setpriv must be unavailable")
	}
	path := filepath.Join(t.TempDir(), "setpriv")
	if err := os.WriteFile(path, []byte("test"), 0o600); err != nil {
		t.Fatal(err)
	}
	if isExecutableRegularFile(path) {
		t.Fatal("non-executable setpriv must be unavailable")
	}
}

func assertRuntimeStatusData(t *testing.T, data map[string]any) {
	t.Helper()
	for _, field := range []string{"schemaVersion", "componentRole", "siteId", "executionForm", "lifecycleState", "healthState", "version", "startedAt", "uptimeSeconds", "observedAt"} {
		if _, ok := data[field]; !ok {
			t.Fatalf("missing schema field %q: %+v", field, data)
		}
	}
	if data["schemaVersion"] != "runtime-health-status.v1" || data["componentRole"] != "compute-sandbox" || !validStableID(data["siteId"].(string)) {
		t.Fatalf("invalid schema identity: %+v", data)
	}
	form, _ := data["executionForm"].(string)
	if !validExecutionForm(form) || form == "native-windows" {
		t.Fatalf("invalid execution form: %+v", data)
	}
	for _, field := range []string{"startedAt", "observedAt"} {
		if _, err := time.Parse(time.RFC3339, data[field].(string)); err != nil {
			t.Fatalf("%s is not RFC3339: %+v", field, data)
		}
	}
}

func TestArtifactPathAndMountReadOnly(t *testing.T) {
	if _, err := artifactFilePath("/artifact", "../escape.json"); err == nil {
		t.Fatal("artifact traversal must be rejected")
	}
	if path, err := artifactFilePath("/artifact", "release/artifact.json"); err != nil || path != "/artifact/release/artifact.json" {
		t.Fatalf("artifact path = %q, %v", path, err)
	}
	ro := "36 25 0:32 / /artifact ro,relatime - ext4 /dev/vda ro\n37 36 0:33 / /artifact/release ro,relatime - ext4 /dev/vdb ro"
	if !mountPathReadOnly(ro, "/artifact/release/a.json") {
		t.Fatal("most specific ro mount must be accepted")
	}
	rw := "36 25 0:32 / /artifact ro,relatime - ext4 /dev/vda ro\n37 36 0:33 / /artifact/release rw,relatime - ext4 /dev/vdb rw"
	if mountPathReadOnly(rw, "/artifact/release/a.json") {
		t.Fatal("most specific rw mount must be rejected")
	}
	escaped := "36 25 0:32 / /artifact\\040root ro,relatime - ext4 /dev/vda ro"
	if !mountPathReadOnly(escaped, "/artifact root/a.json") {
		t.Fatal("escaped mountpoint must be decoded")
	}
}

func TestLoadVerifiedArtifactStrictlyValidatesFixtureAndFileBoundary(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "..", "..", "contracts", "runtime", "fixtures", "runtime-project-artifact.valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(fixture)
	root := t.TempDir()
	file := filepath.Join(root, "artifact.json")
	if err := os.WriteFile(file, fixture, 0o600); err != nil {
		t.Fatal(err)
	}
	config := Config{ArtifactRoot: root, ArtifactFile: "artifact.json", ArtifactDigest: fmt.Sprintf("sha256:%x", digest[:]), ProjectID: "11111111-1111-4111-8111-111111111111"}
	if _, err := loadVerifiedArtifact(config); err != nil {
		t.Fatalf("valid artifact rejected: %v", err)
	}
	for _, broken := range [][]byte{bytes.Replace(fixture, []byte(`"generatedAt": "2026-08-30T10:00:00Z"`), []byte(`"generatedAt": null`), 1), bytes.Replace(fixture, []byte("2026-08-30T10:00:00Z"), []byte("not-a-time"), 1), append(append([]byte{}, fixture...), []byte(" {}")...)} {
		if err := os.WriteFile(file, broken, 0o600); err != nil {
			t.Fatal(err)
		}
		copy := config
		sum := sha256.Sum256(broken)
		copy.ArtifactDigest = fmt.Sprintf("sha256:%x", sum[:])
		if _, err := loadVerifiedArtifact(copy); err == nil {
			t.Fatal("invalid artifact accepted")
		}
	}
	if err := os.Symlink(file, filepath.Join(root, "linked.json")); err == nil {
		copy := config
		copy.ArtifactFile = "linked.json"
		if _, err := loadVerifiedArtifact(copy); err == nil {
			t.Fatal("artifact symlink accepted")
		}
	}
}

func TestRuntimeUUIDAndArtifactFractionalUTC(t *testing.T) {
	for _, value := range []string{"11111111-1111-4111-8111-111111111111", "aaaaaaaa-aaaa-5aaa-8aaa-aaaaaaaaaaaa"} {
		if !canonicalUUID(value) {
			t.Fatalf("valid UUID rejected: %s", value)
		}
	}
	for _, value := range []string{"11111111-1111-0111-8111-111111111111", "11111111-1111-4111-7111-111111111111", "11111111-1111-4111-8111-111111111111", "11111111-1111-4111-8111-111111111111"} {
		_ = value
	}
	for _, value := range []string{"00000000-0000-0000-0000-000000000000", "11111111-1111-4111-7111-111111111111", "11111111-1111-4111-8111-11111111111A"} {
		if canonicalUUID(value) {
			t.Fatalf("invalid UUID accepted: %s", value)
		}
	}
	for _, value := range []string{"2026-08-30T10:00:00.123456789Z"} {
		if !utcRFC3339Pattern.MatchString(value) {
			t.Fatalf("fractional UTC rejected: %s", value)
		}
	}
	for _, value := range []string{"2026-08-30T10:00:00+00:00", "2026-08-30T10:00:00,1Z", "2026-02-30T10:00:00Z"} {
		if utcRFC3339Pattern.MatchString(value) {
			if _, err := time.Parse(time.RFC3339Nano, value); err == nil {
				t.Fatalf("invalid UTC accepted: %s", value)
			}
		}
	}
}

func TestRuntimeExecuteRejectsSourceFieldsByPresence(t *testing.T) {
	for _, source := range []string{`"language":""`, `"script":null`, `"dependencies":[]`, `"timeoutMs":0`} {
		var request executeRequest
		if err := json.Unmarshal([]byte(`{"executionId":"11111111-1111-4111-8111-111111111111",`+source+`}`), &request); err != nil || !request.forbiddenSourceFields {
			t.Fatalf("source presence was not retained: %s, %v", source, err)
		}
	}
}
