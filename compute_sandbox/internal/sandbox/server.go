package sandbox

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

const (
	maxRequestBytes = 1 << 20
	maxOutputBytes  = 2 << 20
	maxLogBytes     = 64 << 10
	maxTimeoutMS    = 120000
)

type Config struct {
	Addr         string
	Token        string
	Bubblewrap   string
	Prlimit      string
	RuntimeDir   string
	NodeBinary   string
	PythonBinary string
}

func LoadConfig() (Config, error) {
	config := Config{
		Addr:         firstNonEmpty(os.Getenv("COMPUTE_SANDBOX_ADDR"), ":18103"),
		Token:        strings.TrimSpace(os.Getenv("COMPUTE_SANDBOX_TOKEN")),
		Bubblewrap:   firstNonEmpty(os.Getenv("COMPUTE_SANDBOX_BWRAP"), "/usr/bin/bwrap"),
		Prlimit:      firstNonEmpty(os.Getenv("COMPUTE_SANDBOX_PRLIMIT"), "/usr/bin/prlimit"),
		RuntimeDir:   firstNonEmpty(os.Getenv("COMPUTE_SANDBOX_RUNTIME_DIR"), "/opt/induforge/runtime"),
		NodeBinary:   firstNonEmpty(os.Getenv("COMPUTE_SANDBOX_NODE"), "/usr/local/bin/node"),
		PythonBinary: firstNonEmpty(os.Getenv("COMPUTE_SANDBOX_PYTHON"), "/usr/bin/python3"),
	}
	if len(config.Token) < 24 {
		return Config{}, fmt.Errorf("COMPUTE_SANDBOX_TOKEN 至少需要 24 个字符")
	}
	return config, nil
}

func NewHandler(config Config) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	})
	mux.HandleFunc("GET /v1/capabilities", authorize(config, capabilitiesHandler(config)))
	mux.HandleFunc("POST /v1/syntax-check", authorize(config, syntaxHandler(config)))
	mux.HandleFunc("POST /v1/execute", authorize(config, executeHandler(config)))
	return mux
}

type languageCapability struct {
	Language string `json:"language"`
	Version  string `json:"version"`
}

func capabilitiesHandler(config Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		languages := []languageCapability{}
		for _, item := range []struct{ language, binary string }{{"js", config.NodeBinary}, {"python", config.PythonBinary}} {
			output, err := exec.CommandContext(r.Context(), item.binary, "--version").CombinedOutput()
			if err == nil {
				languages = append(languages, languageCapability{Language: item.language, Version: strings.TrimSpace(string(output))})
			}
		}
		_, bubblewrapErr := os.Stat(config.Bubblewrap)
		_, prlimitErr := os.Stat(config.Prlimit)
		available := bubblewrapErr == nil && prlimitErr == nil && len(languages) == 2 && runtime.GOOS == "linux"
		writeJSON(w, http.StatusOK, map[string]any{
			"available": available, "serviceVersion": "1.0.0", "languages": languages,
			"sdk":          []string{"ctx.datapoint.get", "ctx.datapoint.meta", "ctx.sql.query"},
			"dependencies": []any{},
			"triggers":     []string{"manual", "schedule", "datapoint_change"},
			"limits":       map[string]int{"maxExecutionTimeMs": maxTimeoutMS, "maxInputBytes": maxRequestBytes, "maxOutputBytes": maxOutputBytes, "maxLogBytes": maxLogBytes},
		})
	}
}

type syntaxRequest struct {
	Language  string `json:"language"`
	Script    string `json:"script"`
	TimeoutMS int64  `json:"timeoutMs"`
}

type executeRequest struct {
	Language   string         `json:"language"`
	Script     string         `json:"script"`
	Input      map[string]any `json:"input"`
	SDKContext map[string]any `json:"sdkContext"`
	TimeoutMS  int64          `json:"timeoutMs"`
}

func syntaxHandler(config Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request syntaxRequest
		if err := decodeRequest(w, r, &request); err != nil {
			return
		}
		if err := validateLanguageAndScript(request.Language, request.Script); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		script := "node_syntax.js"
		if request.Language == "python" {
			script = "python_syntax.py"
		}
		result, err := runIsolated(r.Context(), config, request.Language, script, map[string]any{"script": request.Script}, normalizedTimeout(request.TimeoutMS, 3000))
		if err != nil {
			writeExecutionError(w, err)
			return
		}
		var payload map[string]any
		if err := json.Unmarshal(result.Output, &payload); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("沙箱语法结果无效"))
			return
		}
		writeJSON(w, http.StatusOK, payload)
	}
}

func executeHandler(config Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request executeRequest
		if err := decodeRequest(w, r, &request); err != nil {
			return
		}
		if err := validateLanguageAndScript(request.Language, request.Script); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		script := "node_execute.js"
		if request.Language == "python" {
			script = "python_execute.py"
		}
		started := time.Now()
		result, err := runIsolated(r.Context(), config, request.Language, script, map[string]any{
			"script": request.Script, "input": request.Input, "sdkContext": request.SDKContext,
		}, normalizedTimeout(request.TimeoutMS, 3000))
		if err != nil {
			writeExecutionError(w, err)
			return
		}
		var payload struct {
			Output      any      `json:"output"`
			SideEffects []any    `json:"sideEffects"`
			Logs        []string `json:"logs"`
		}
		if err := json.Unmarshal(result.Output, &payload); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("沙箱执行结果无效"))
			return
		}
		stdout := strings.Join(payload.Logs, "\n")
		writeJSON(w, http.StatusOK, map[string]any{
			"output": payload.Output, "sideEffects": payload.SideEffects, "stdout": stdout,
			"stderr": result.Stderr, "durationMs": time.Since(started).Milliseconds(),
		})
	}
}

var errExecutionTimeout = errors.New("execution timeout")

type isolatedResult struct {
	Output []byte
	Stderr string
}

func runIsolated(ctx context.Context, config Config, language, runtimeScript string, input any, timeout time.Duration) (isolatedResult, error) {
	payload, err := json.Marshal(input)
	if err != nil {
		return isolatedResult{}, fmt.Errorf("编码沙箱输入失败: %w", err)
	}
	if len(payload) > maxRequestBytes {
		return isolatedResult{}, fmt.Errorf("沙箱输入超过 %d 字节", maxRequestBytes)
	}
	args, err := buildSandboxArgs(config, language, runtimeScript)
	if err != nil {
		return isolatedResult{}, err
	}
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.Command(config.Prlimit, args...)
	cmd.Stdin = bytes.NewReader(payload)
	cmd.Env = []string{"PATH=/usr/local/bin:/usr/bin:/bin", "LANG=C.UTF-8"}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stdout := &limitedBuffer{limit: maxOutputBytes}
	stderr := &limitedBuffer{limit: maxLogBytes}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	if err := cmd.Start(); err != nil {
		return isolatedResult{}, fmt.Errorf("启动隔离执行器失败: %w", err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			return isolatedResult{Stderr: stderr.String()}, fmt.Errorf("脚本执行失败: %s", strings.TrimSpace(stderr.String()))
		}
	case <-execCtx.Done():
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		<-done
		return isolatedResult{}, errExecutionTimeout
	}
	if stdout.overflow {
		return isolatedResult{}, fmt.Errorf("脚本输出超过 %d 字节", maxOutputBytes)
	}
	return isolatedResult{Output: stdout.Bytes(), Stderr: stderr.String()}, nil
}

func buildSandboxArgs(config Config, language, runtimeScript string) ([]string, error) {
	if _, err := os.Stat(config.Bubblewrap); err != nil {
		return nil, fmt.Errorf("bubblewrap 不可用")
	}
	if _, err := os.Stat(config.Prlimit); err != nil {
		return nil, fmt.Errorf("prlimit 不可用")
	}
	runtimeScriptPath := filepath.Join(config.RuntimeDir, runtimeScript)
	if _, err := os.Stat(runtimeScriptPath); err != nil {
		return nil, fmt.Errorf("沙箱运行脚本不存在: %s", runtimeScript)
	}
	binary := config.NodeBinary
	if language == "python" {
		binary = config.PythonBinary
	}
	return []string{
		"--as=1073741824", "--nproc=32", "--nofile=64", "--fsize=4194304", "--cpu=120", "--",
		config.Bubblewrap, "--die-with-parent", "--new-session", "--unshare-ipc", "--unshare-pid", "--unshare-net", "--unshare-uts", "--unshare-cgroup-try", "--clearenv",
		"--ro-bind", "/usr", "/usr", "--ro-bind-try", "/bin", "/bin", "--ro-bind-try", "/lib", "/lib",
		"--ro-bind-try", "/lib64", "/lib64", "--ro-bind-try", "/usr/local", "/usr/local",
		"--ro-bind", config.RuntimeDir, "/runtime", "--proc", "/proc", "--dev", "/dev", "--tmpfs", "/tmp",
		"--chdir", "/tmp", "--setenv", "PATH", "/usr/local/bin:/usr/bin:/bin", "--setenv", "LANG", "C.UTF-8",
		"--cap-add", "CAP_SETUID", "--cap-add", "CAP_SETGID",
		"/usr/bin/setpriv", "--reuid=10001", "--regid=10001", "--clear-groups", "--bounding-set=-all", "--no-new-privs", binary, filepath.Join("/runtime", runtimeScript),
	}, nil
}

type limitedBuffer struct {
	buffer   bytes.Buffer
	limit    int
	overflow bool
}

func (b *limitedBuffer) Write(value []byte) (int, error) {
	original := len(value)
	remaining := b.limit - b.buffer.Len()
	if remaining <= 0 {
		b.overflow = true
		return original, nil
	}
	if len(value) > remaining {
		value = value[:remaining]
		b.overflow = true
	}
	_, _ = b.buffer.Write(value)
	return original, nil
}
func (b *limitedBuffer) Bytes() []byte  { return b.buffer.Bytes() }
func (b *limitedBuffer) String() string { return b.buffer.String() }

func authorize(config Config, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		provided := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if subtle.ConstantTimeCompare([]byte(provided), []byte(config.Token)) != 1 {
			writeError(w, http.StatusUnauthorized, fmt.Errorf("unauthorized"))
			return
		}
		next(w, r)
	}
}

func decodeRequest(w http.ResponseWriter, r *http.Request, output any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(output); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("请求格式无效"))
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, fmt.Errorf("请求只能包含一个 JSON 对象"))
		return fmt.Errorf("trailing payload")
	}
	return nil
}

func validateLanguageAndScript(language, script string) error {
	if language != "js" && language != "python" {
		return fmt.Errorf("language 仅支持 js/python")
	}
	if strings.TrimSpace(script) == "" {
		return fmt.Errorf("script 不能为空")
	}
	return nil
}

func normalizedTimeout(value int64, fallback int64) time.Duration {
	if value <= 0 {
		value = fallback
	}
	if value > maxTimeoutMS {
		value = maxTimeoutMS
	}
	return time.Duration(value) * time.Millisecond
}

func writeExecutionError(w http.ResponseWriter, err error) {
	if errors.Is(err, errExecutionTimeout) || errors.Is(err, context.DeadlineExceeded) {
		writeError(w, http.StatusRequestTimeout, errExecutionTimeout)
		return
	}
	writeError(w, http.StatusBadRequest, err)
}
func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
