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

func TestBuildSandboxArgsRequiresIsolationAndNoNetwork(t *testing.T) {
	tempDir := t.TempDir()
	for _, name := range []string{"bwrap", "prlimit", "node_execute.js"} {
		if err := os.WriteFile(filepath.Join(tempDir, name), []byte("test"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	config := Config{Bubblewrap: filepath.Join(tempDir, "bwrap"), Prlimit: filepath.Join(tempDir, "prlimit"), RuntimeDir: tempDir, NodeBinary: "/usr/local/bin/node"}
	args, err := buildSandboxArgs(config, "js", "node_execute.js")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	for _, required := range []string{"--unshare-net", "--unshare-pid", "--clearenv", "--tmpfs /tmp", "--ro-bind " + tempDir + " /runtime", "--nproc=32", "--as=1073741824"} {
		if !strings.Contains(joined, required) {
			t.Fatalf("sandbox args missing %q: %s", required, joined)
		}
	}
	if strings.Contains(joined, "--share-net") {
		t.Fatalf("sandbox must not share network: %s", joined)
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
