package sandbox

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	maxRequestBytes                 = 1 << 20
	maxOutputBytes                  = 2 << 20
	maxLogBytes                     = 64 << 10
	maxTimeoutMS                    = 120000
	defaultMaxConcurrentExecutions  = 4
	defaultMaxPIDs                  = 64
	maxConfiguredPIDs               = 8192
	developmentRuntimeProfile       = "development"
	runtimeRuntimeProfile           = "runtime"
	invalidRuntimeProfile           = "invalid"
	defaultDevelopmentSiteID        = "local-development"
	invalidRuntimeSiteID            = "invalid-runtime-profile"
	defaultRuntimeExecutionForm     = "k3s-workload"
	defaultDevelopmentExecutionForm = "native-linux"
	setprivPath                     = "/usr/bin/setpriv"
)

var (
	runtimeUUIDPattern    = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	artifactDigestPattern = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
	stableIDPattern       = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
)

type Config struct {
	Addr                    string
	Token                   string
	Bubblewrap              string
	Prlimit                 string
	RuntimeDir              string
	NodeBinary              string
	PythonBinary            string
	DependenciesDir         string
	RuntimeProfile          string
	MaxConcurrentExecutions int
	SiteID                  string
	NodeID                  string
	ExecutionForm           string
	MaxPIDs                 int
	DeploymentID            string
	ProjectID               string
	ArtifactDigest          string
	ArtifactRoot            string
	ArtifactFile            string
	configurationInvalid    bool
	artifactUnavailable     bool
	probeUnavailable        bool
}

func LoadConfig() (Config, error) {
	config := Config{
		Addr:                    firstNonEmpty(os.Getenv("COMPUTE_SANDBOX_ADDR"), ":18103"),
		Token:                   strings.TrimSpace(os.Getenv("COMPUTE_SANDBOX_TOKEN")),
		Bubblewrap:              firstNonEmpty(os.Getenv("COMPUTE_SANDBOX_BWRAP"), "/usr/bin/bwrap"),
		Prlimit:                 firstNonEmpty(os.Getenv("COMPUTE_SANDBOX_PRLIMIT"), "/usr/bin/prlimit"),
		RuntimeDir:              firstNonEmpty(os.Getenv("COMPUTE_SANDBOX_RUNTIME_DIR"), "/opt/induforge/runtime"),
		NodeBinary:              firstNonEmpty(os.Getenv("COMPUTE_SANDBOX_NODE"), "/usr/local/bin/node"),
		PythonBinary:            firstNonEmpty(os.Getenv("COMPUTE_SANDBOX_PYTHON"), "/usr/bin/python3"),
		DependenciesDir:         firstNonEmpty(os.Getenv("COMPUTE_SANDBOX_DEPENDENCIES_DIR"), "/dependencies"),
		RuntimeProfile:          firstNonEmpty(os.Getenv("COMPUTE_SANDBOX_RUNTIME_PROFILE"), developmentRuntimeProfile),
		MaxConcurrentExecutions: defaultMaxConcurrentExecutions,
		SiteID:                  strings.TrimSpace(os.Getenv("COMPUTE_SANDBOX_SITE_ID")),
		NodeID:                  strings.TrimSpace(os.Getenv("COMPUTE_SANDBOX_NODE_ID")),
		ExecutionForm:           strings.TrimSpace(os.Getenv("COMPUTE_SANDBOX_EXECUTION_FORM")),
		MaxPIDs:                 defaultMaxPIDs,
		DeploymentID:            strings.TrimSpace(os.Getenv("COMPUTE_SANDBOX_DEPLOYMENT_ID")),
		ProjectID:               strings.TrimSpace(os.Getenv("COMPUTE_SANDBOX_PROJECT_ID")),
		ArtifactDigest:          strings.TrimSpace(os.Getenv("COMPUTE_SANDBOX_ARTIFACT_DIGEST")),
		ArtifactRoot:            strings.TrimSpace(os.Getenv("COMPUTE_SANDBOX_ARTIFACT_ROOT")),
		ArtifactFile:            strings.TrimSpace(os.Getenv("COMPUTE_SANDBOX_ARTIFACT_FILE")),
	}
	if len(config.Token) < 24 {
		return Config{}, fmt.Errorf("COMPUTE_SANDBOX_TOKEN 至少需要 24 个字符")
	}
	profile, err := normalizeRuntimeProfile(config.RuntimeProfile)
	if err != nil {
		return Config{}, err
	}
	config.RuntimeProfile = profile
	if config.RuntimeProfile == runtimeRuntimeProfile && config.ArtifactDigest == "" {
		path, pathErr := secureArtifactFilePath(config.ArtifactRoot, config.ArtifactFile)
		raw, readErr := os.ReadFile(path)
		if pathErr != nil || readErr != nil {
			return Config{}, fmt.Errorf("COMPUTE_SANDBOX_ARTIFACT_FILE 不可用")
		}
		digest := sha256.Sum256(raw)
		config.ArtifactDigest = fmt.Sprintf("sha256:%x", digest[:])
	}
	if err := normalizeRuntimeIdentity(&config); err != nil {
		return Config{}, err
	}
	if configured := strings.TrimSpace(os.Getenv("COMPUTE_SANDBOX_MAX_PIDS")); configured != "" {
		limit, err := strconv.Atoi(configured)
		if err != nil || limit < 1 || limit > maxConfiguredPIDs {
			return Config{}, fmt.Errorf("COMPUTE_SANDBOX_MAX_PIDS 必须介于 1 到 %d", maxConfiguredPIDs)
		}
		config.MaxPIDs = limit
	}
	if configured := strings.TrimSpace(os.Getenv("COMPUTE_SANDBOX_MAX_CONCURRENT_EXECUTIONS")); configured != "" {
		limit, err := strconv.Atoi(configured)
		if err != nil || limit < 1 || limit > 128 {
			return Config{}, fmt.Errorf("COMPUTE_SANDBOX_MAX_CONCURRENT_EXECUTIONS 必须介于 1 到 128")
		}
		config.MaxConcurrentExecutions = limit
	}
	return config, nil
}

func NewHandler(config Config) http.Handler {
	config = normalizeHandlerConfig(config)
	var artifact *verifiedArtifact
	if isStrictRuntimeProfile(config) {
		var err error
		artifact, err = loadVerifiedArtifact(config)
		config.artifactUnavailable = err != nil || !runtimeMountsReadOnly(config)
		if !config.artifactUnavailable && !probeRuntimeArtifact(config, artifact) {
			config.probeUnavailable = true
		}
	}
	gate := newExecutionGate(config.MaxConcurrentExecutions)
	status := newRuntimeStatus(config)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler(status))
	mux.HandleFunc("GET /api/v1/status", statusHandler(status))
	mux.HandleFunc("GET /v1/capabilities", authorize(config, capabilitiesHandler(config)))
	mux.HandleFunc("POST /v1/syntax-check", authorize(config, syntaxHandler(config)))
	mux.HandleFunc("POST /v1/execute", authorize(config, executeHandler(config, gate, artifact)))
	mux.HandleFunc("POST /v1/dependencies/install", authorize(config, developmentOnlyHandler(config, installDependencyHandler(config))))
	mux.HandleFunc("POST /v1/dependencies/import", authorize(config, developmentOnlyHandler(config, importDependencyHandler(config))))
	mux.HandleFunc("POST /v1/dependencies/uninstall", authorize(config, developmentOnlyHandler(config, uninstallDependencyHandler(config))))
	return mux
}

func normalizeHandlerConfig(config Config) Config {
	profile, err := normalizeRuntimeProfile(config.RuntimeProfile)
	if err != nil {
		// NewHandler 不能返回配置错误，未知 profile 必须收敛到最严格的拒绝态，不能静默降级为开发态。
		profile = invalidRuntimeProfile
	}
	config.RuntimeProfile = profile
	if err := normalizeRuntimeIdentity(&config); err != nil && isStrictRuntimeProfile(config) {
		config.configurationInvalid = true
	}
	// 直接构造 Config 的调用同样不能绕过资源上限；开发态安全回落，运行态同时进入拒绝态。
	if config.MaxConcurrentExecutions < 1 || config.MaxConcurrentExecutions > 128 {
		if isStrictRuntimeProfile(config) {
			config.configurationInvalid = true
		}
		config.MaxConcurrentExecutions = defaultMaxConcurrentExecutions
	}
	if config.MaxPIDs < 1 || config.MaxPIDs > maxConfiguredPIDs {
		if isStrictRuntimeProfile(config) {
			config.configurationInvalid = true
		}
		config.MaxPIDs = defaultMaxPIDs
	}
	return config
}

func normalizeRuntimeProfile(value string) (string, error) {
	profile := strings.ToLower(strings.TrimSpace(value))
	if profile == "" {
		return developmentRuntimeProfile, nil
	}
	if profile != developmentRuntimeProfile && profile != runtimeRuntimeProfile {
		return "", fmt.Errorf("COMPUTE_SANDBOX_RUNTIME_PROFILE 仅支持 development/runtime")
	}
	return profile, nil
}

func isRuntimeProfile(config Config) bool { return config.RuntimeProfile == runtimeRuntimeProfile }

func isStrictRuntimeProfile(config Config) bool {
	return config.RuntimeProfile == runtimeRuntimeProfile || config.RuntimeProfile == invalidRuntimeProfile
}

func normalizeRuntimeIdentity(config *Config) error {
	if config == nil {
		return fmt.Errorf("运行配置不能为空")
	}
	config.SiteID = strings.TrimSpace(config.SiteID)
	config.NodeID = strings.TrimSpace(config.NodeID)
	config.ExecutionForm = strings.TrimSpace(config.ExecutionForm)
	config.DeploymentID = strings.TrimSpace(config.DeploymentID)
	config.ProjectID = strings.TrimSpace(config.ProjectID)
	config.ArtifactDigest = strings.TrimSpace(config.ArtifactDigest)
	if config.RuntimeProfile == developmentRuntimeProfile && config.SiteID == "" {
		// 开发态同样要满足正式状态契约的 siteId 必填约束，但不能影响 runtime 的显式身份校验。
		config.SiteID = defaultDevelopmentSiteID
	}
	if config.ExecutionForm == "" {
		if isStrictRuntimeProfile(*config) {
			config.ExecutionForm = defaultRuntimeExecutionForm
		} else {
			config.ExecutionForm = defaultDevelopmentExecutionForm
		}
	}
	if !validExecutionForm(config.ExecutionForm) {
		return fmt.Errorf("COMPUTE_SANDBOX_EXECUTION_FORM 仅支持 k3s-workload/native-linux/native-windows")
	}
	if config.SiteID != "" && !validStableID(config.SiteID) {
		return fmt.Errorf("COMPUTE_SANDBOX_SITE_ID 格式无效")
	}
	if config.NodeID != "" && !validStableID(config.NodeID) {
		return fmt.Errorf("COMPUTE_SANDBOX_NODE_ID 格式无效")
	}
	if config.RuntimeProfile == runtimeRuntimeProfile {
		if config.SiteID == "" {
			return fmt.Errorf("COMPUTE_SANDBOX_SITE_ID 在 runtime profile 下不能为空")
		}
		if !validRuntimeExecutionForm(config.ExecutionForm) {
			return fmt.Errorf("COMPUTE_SANDBOX_EXECUTION_FORM 在 runtime profile 下仅支持 k3s-workload/native-linux")
		}
		if !validStableID(config.DeploymentID) {
			return fmt.Errorf("COMPUTE_SANDBOX_DEPLOYMENT_ID 格式无效")
		}
		if !canonicalUUID(config.ProjectID) {
			return fmt.Errorf("COMPUTE_SANDBOX_PROJECT_ID 必须为 UUID")
		}
		if !artifactDigestPattern.MatchString(config.ArtifactDigest) {
			return fmt.Errorf("COMPUTE_SANDBOX_ARTIFACT_DIGEST 必须为小写 sha256 摘要")
		}
	}
	return nil
}

func validExecutionForm(value string) bool {
	return value == "k3s-workload" || value == "native-linux" || value == "native-windows"
}

func validRuntimeExecutionForm(value string) bool {
	return value == "k3s-workload" || value == "native-linux"
}

func validStableID(value string) bool { return stableIDPattern.MatchString(value) }

func strictRuntimeConfigurationValid(config Config) bool {
	return config.RuntimeProfile == runtimeRuntimeProfile && !config.configurationInvalid &&
		validStableID(config.SiteID) &&
		(config.NodeID == "" || validStableID(config.NodeID)) &&
		validRuntimeExecutionForm(config.ExecutionForm) &&
		validStableID(config.DeploymentID) &&
		canonicalUUID(config.ProjectID) &&
		artifactDigestPattern.MatchString(config.ArtifactDigest) &&
		config.MaxConcurrentExecutions >= 1 && config.MaxConcurrentExecutions <= 128 &&
		config.MaxPIDs >= 1 && config.MaxPIDs <= maxConfiguredPIDs
}

type languageCapability struct {
	Language string `json:"language"`
	Version  string `json:"version"`
}

func capabilitiesHandler(config Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if isStrictRuntimeProfile(config) {
			available, reasons := runtimeDependenciesAvailable(config)
			if !available {
				// runtime 配置或依赖不可用时禁止探测或创建隔离进程，保持与健康、执行端点一致。
				writeRuntimeCapabilities(w, false, runtimeCapabilitiesReason(reasons))
				return
			}
			// runtime 不运行 capability probe，执行只通过带并发上限的 /v1/execute 入口创建隔离进程。
			writeRuntimeCapabilities(w, true, "")
			return
		}
		languages := []languageCapability{}
		failures := []string{}
		for _, item := range []struct{ language, binary string }{{"js", config.NodeBinary}, {"python", config.PythonBinary}} {
			output, err := exec.CommandContext(r.Context(), item.binary, "--version").CombinedOutput()
			if err == nil {
				languages = append(languages, languageCapability{Language: item.language, Version: strings.TrimSpace(string(output))})
			} else {
				failures = append(failures, fmt.Sprintf("%s 运行时不可用", item.language))
			}
		}
		_, bubblewrapErr := os.Stat(config.Bubblewrap)
		_, prlimitErr := os.Stat(config.Prlimit)
		if runtime.GOOS != "linux" {
			failures = append(failures, "仅支持 Linux 隔离环境")
		}
		if bubblewrapErr != nil {
			failures = append(failures, "bubblewrap 不可用")
		}
		if prlimitErr != nil {
			failures = append(failures, "prlimit 不可用")
		}
		if len(failures) == 0 {
			for _, probe := range []struct {
				language string
				script   string
			}{{"js", `return "ok";`}, {"python", "def main(argv, dp, ctx):\n    return 'ok'"}} {
				if err := probeIsolatedRuntime(r.Context(), config, probe.language, probe.script); err != nil {
					failures = append(failures, fmt.Sprintf("%s 隔离执行失败: %v", probe.language, err))
				}
			}
		}
		available := len(failures) == 0 && len(languages) == 2
		sdk := []string{
			"point.get", "point.read", "point.peek", "point.set", "point.refresh",
			"point.run", "point.execute", "point.publish", "ctx.points", "ctx.sql.query",
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"available": available, "serviceVersion": "1.0.0", "languages": languages,
			"reason":       strings.Join(failures, "；"),
			"sdk":          sdk,
			"dependencies": []any{},
			"triggers":     []string{"manual", "schedule", "datapoint_change", "condition"},
			"limits":       map[string]int{"maxExecutionTimeMs": maxTimeoutMS, "maxInputBytes": maxRequestBytes, "maxOutputBytes": maxOutputBytes, "maxLogBytes": maxLogBytes},
		})
	}
}

func writeRuntimeCapabilities(w http.ResponseWriter, available bool, reason string) {
	writeJSON(w, http.StatusOK, map[string]any{
		"available": available, "serviceVersion": "1.0.0", "languages": []languageCapability{},
		"reason":       reason,
		"sdk":          []string{"point.get", "point.read", "point.peek", "ctx.points", "ctx.sql.query"},
		"dependencies": []any{},
		"triggers":     []string{},
		"limits":       map[string]int{"maxExecutionTimeMs": maxTimeoutMS, "maxInputBytes": maxRequestBytes, "maxOutputBytes": maxOutputBytes, "maxLogBytes": maxLogBytes},
	})
}

func runtimeCapabilitiesReason(reasons []string) string {
	for _, reason := range reasons {
		switch reason {
		case "RUNTIME_PROFILE_INVALID", "RUNTIME_CONFIGURATION_INVALID", "SITE_ID_REQUIRED", "SITE_ID_INVALID", "NODE_ID_INVALID", "EXECUTION_FORM_INVALID", "DEPLOYMENT_ID_INVALID", "PROJECT_ID_INVALID", "ARTIFACT_DIGEST_INVALID", "PIDS_LIMIT_UNCONFIRMED", "PIDS_LIMIT_EXCEEDED":
			return "运行配置无效"
		}
	}
	return "运行依赖不可用"
}

// probeIsolatedRuntime 真实启动一次隔离执行，避免仅检测二进制存在却误报沙箱可用。
func probeIsolatedRuntime(ctx context.Context, config Config, language, script string) error {
	runtimeScript := "node_execute.js"
	if language == "python" {
		runtimeScript = "python_execute.py"
	}
	result, err := runIsolated(ctx, config, language, runtimeScript, map[string]any{
		"script": script, "input": map[string]any{}, "sdkContext": map[string]any{},
	}, 3*time.Second, "")
	if err != nil {
		return err
	}
	var payload map[string]any
	if err := json.Unmarshal(result.Output, &payload); err != nil {
		return fmt.Errorf("运行结果无效")
	}
	return nil
}

// probeRuntimeArtifact 在 handler 创建时仅执行固定探针并缓存结果，健康检查不再创建额外子进程。
func probeRuntimeArtifact(config Config, artifact *verifiedArtifact) bool {
	languages := map[string]struct{}{}
	for _, unit := range artifact.units {
		if unit.Enabled {
			languages[unit.Language] = struct{}{}
		}
	}
	for language := range languages {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		script := `return "probe";`
		if language == "python" {
			script = "def main(argv, dp, ctx):\n    return 'probe'"
		}
		err := probeIsolatedRuntime(ctx, config, language, script)
		cancel()
		if err != nil {
			return false
		}
	}
	return true
}

type syntaxRequest struct {
	Language  string `json:"language"`
	Script    string `json:"script"`
	TimeoutMS int64  `json:"timeoutMs"`
}

type executeRequest struct {
	ExecutionID           string           `json:"executionId"`
	DeploymentID          string           `json:"deploymentId"`
	Language              string           `json:"language"`
	Script                string           `json:"script"`
	Input                 map[string]any   `json:"input"`
	SDKContext            map[string]any   `json:"sdkContext"`
	TimeoutMS             int64            `json:"timeoutMs"`
	ProjectID             string           `json:"projectId"`
	ComputeUnitID         string           `json:"computeUnitId"`
	ArtifactDigest        string           `json:"artifactDigest"`
	Dependencies          []dependencySpec `json:"dependencies"`
	forbiddenSourceFields bool
}

func (request *executeRequest) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	allowed := map[string]bool{"executionId": true, "deploymentId": true, "language": true, "script": true, "input": true, "sdkContext": true, "timeoutMs": true, "projectId": true, "computeUnitId": true, "artifactDigest": true, "dependencies": true}
	for key := range fields {
		if !allowed[key] {
			return fmt.Errorf("unknown field")
		}
	}
	type plain executeRequest
	var decoded plain
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*request = executeRequest(decoded)
	for _, key := range []string{"language", "script", "dependencies", "timeoutMs"} {
		if _, exists := fields[key]; exists {
			request.forbiddenSourceFields = true
		}
	}
	return nil
}

func syntaxHandler(config Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if isStrictRuntimeProfile(config) {
			writeStructuredError(w, http.StatusForbidden, "RUNTIME_PROFILE_RESTRICTED", "运行态不提供语法检查")
			return
		}
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
		result, err := runIsolated(r.Context(), config, request.Language, script, map[string]any{"script": request.Script}, normalizedTimeout(request.TimeoutMS, 3000), "")
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

func executeHandler(config Config, gate *executionGate, artifacts ...*verifiedArtifact) http.HandlerFunc {
	var artifact *verifiedArtifact
	if len(artifacts) > 0 {
		artifact = artifacts[0]
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var request executeRequest
		if err := decodeRequest(w, r, &request); err != nil {
			return
		}
		if err := validateRuntimeExecution(config, request); err != nil {
			writeStructuredError(w, http.StatusBadRequest, "RUNTIME_REQUEST_INVALID", err.Error())
			return
		}
		if isStrictRuntimeProfile(config) {
			if request.forbiddenSourceFields {
				writeStructuredError(w, http.StatusBadRequest, "RUNTIME_REQUEST_INVALID", "运行态执行参数不得携带源码配置")
				return
			}
			if artifact == nil {
				writeStructuredError(w, http.StatusServiceUnavailable, "RUNTIME_UNAVAILABLE", "运行依赖不可用")
				return
			}
			unit, exists := artifact.units[request.ComputeUnitID]
			if !exists || !unit.Enabled {
				writeStructuredError(w, http.StatusBadRequest, "RUNTIME_REQUEST_INVALID", "computeUnitId 不可执行")
				return
			}
			request.Language, request.Script, request.Dependencies, request.TimeoutMS = unit.Language, unit.ScriptCode, unit.Dependencies, unit.TimeoutMS
		}
		if err := validateLanguageAndScript(request.Language, request.Script); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := validateExecuteDependencies(request.Language, request.Dependencies); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if isStrictRuntimeProfile(config) {
			if available, _ := runtimeDependenciesAvailable(config); !available {
				writeStructuredError(w, http.StatusServiceUnavailable, "RUNTIME_UNAVAILABLE", "运行依赖不可用")
				return
			}
		}
		if !gate.tryAcquire() {
			writeStructuredError(w, http.StatusTooManyRequests, "SANDBOX_OVERLOADED", "沙箱并发执行已达上限，请稍后重试")
			return
		}
		defer gate.release()
		script := "node_execute.js"
		if request.Language == "python" {
			script = "python_execute.py"
		}
		started := time.Now()
		result, err := runIsolated(r.Context(), config, request.Language, script, map[string]any{
			"script": request.Script, "input": request.Input, "sdkContext": request.SDKContext, "dependencies": request.Dependencies,
		}, normalizedTimeout(request.TimeoutMS, 3000), request.ProjectID)
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
		// 运行态只产出纯计算结果；副作用必须交给具备 Lease/epoch 的运行引擎单独执行。
		if err := validateExecutionResult(config, payload.SideEffects); err != nil {
			writeStructuredError(w, http.StatusConflict, "RUNTIME_SIDE_EFFECT_REJECTED", err.Error())
			return
		}
		stdout := strings.Join(payload.Logs, "\n")
		response := map[string]any{
			"output": payload.Output, "sideEffects": payload.SideEffects, "stdout": stdout,
			"stderr": result.Stderr, "durationMs": time.Since(started).Milliseconds(),
		}
		if isStrictRuntimeProfile(config) {
			response["executionId"] = request.ExecutionID
			response["artifactDigest"] = request.ArtifactDigest
			response["computeUnitId"] = request.ComputeUnitID
			response["revision"] = artifact.units[request.ComputeUnitID].Revision
		}
		writeJSON(w, http.StatusOK, response)
	}
}

// validateExecuteDependencies 在执行边界再次校验声明依赖，避免调用方绕过保存时的依赖解析约束。
func validateExecuteDependencies(language string, dependencies []dependencySpec) error {
	seenImports := make(map[string]struct{}, len(dependencies))
	for _, item := range dependencies {
		validated := dependencyMutationRequest{
			ProjectID: "00000000-0000-4000-8000-000000000000",
			dependencySpec: dependencySpec{
				Language:    item.Language,
				PackageName: item.PackageName,
				ImportName:  item.ImportName,
				Version:     item.Version,
			},
		}
		if err := validateDependencyMutation(&validated, true); err != nil {
			return fmt.Errorf("dependencies 包含无效依赖: %w", err)
		}
		if validated.Language != language {
			return fmt.Errorf("dependencies 的 language 必须与执行语言一致")
		}
		if unsafeDependencyPath(validated.PackageName) || unsafeDependencyPath(validated.ImportName) {
			return fmt.Errorf("dependencies 不允许路径穿越或绝对路径")
		}
		if _, exists := seenImports[validated.ImportName]; exists {
			return fmt.Errorf("dependencies 不允许重复 importName")
		}
		seenImports[validated.ImportName] = struct{}{}
	}
	return nil
}

func unsafeDependencyPath(value string) bool {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "/") || strings.HasPrefix(value, "\\") || strings.Contains(value, "\\") {
		return true
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == "." || segment == ".." {
			return true
		}
	}
	return false
}

func validateRuntimeExecution(config Config, request executeRequest) error {
	if !isStrictRuntimeProfile(config) {
		return nil
	}
	if !strictRuntimeConfigurationValid(config) {
		return fmt.Errorf("运行配置不可用")
	}
	if !canonicalUUID(strings.TrimSpace(request.ExecutionID)) {
		return fmt.Errorf("executionId 必须为 UUID")
	}
	if !validStableID(strings.TrimSpace(request.DeploymentID)) || request.DeploymentID != config.DeploymentID {
		return fmt.Errorf("deploymentId 与当前运行身份不匹配")
	}
	if !canonicalUUID(strings.TrimSpace(request.ProjectID)) || request.ProjectID != config.ProjectID {
		return fmt.Errorf("projectId 与当前运行身份不匹配")
	}
	if !validDependencyProjectID(request.ProjectID) {
		return fmt.Errorf("projectId 必须为 UUID")
	}
	if !canonicalUUID(strings.TrimSpace(request.ComputeUnitID)) {
		return fmt.Errorf("computeUnitId 必须为 UUID")
	}
	if !artifactDigestPattern.MatchString(strings.TrimSpace(request.ArtifactDigest)) || request.ArtifactDigest != config.ArtifactDigest {
		return fmt.Errorf("artifactDigest 与当前发布产物不匹配")
	}
	return nil
}

func validateExecutionResult(config Config, sideEffects []any) error {
	if isStrictRuntimeProfile(config) && len(sideEffects) > 0 {
		return fmt.Errorf("运行态计算不允许产生副作用")
	}
	return nil
}

// executionGate 将进程创建数量限制在固定上限；运行态不排队，避免高压时耗尽宿主资源。
type executionGate struct{ slots chan struct{} }

func newExecutionGate(limit int) *executionGate {
	if limit <= 0 {
		limit = defaultMaxConcurrentExecutions
	}
	return &executionGate{slots: make(chan struct{}, limit)}
}

func (g *executionGate) tryAcquire() bool {
	select {
	case g.slots <- struct{}{}:
		return true
	default:
		return false
	}
}

func (g *executionGate) release() { <-g.slots }

type runtimeStatus struct {
	config    Config
	startedAt time.Time
}

func newRuntimeStatus(config Config) runtimeStatus {
	return runtimeStatus{config: config, startedAt: time.Now().UTC()}
}

func healthHandler(status runtimeStatus) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		available, _ := runtimeDependenciesAvailable(status.config)
		observedAt := time.Now().UTC()
		if !available && isStrictRuntimeProfile(status.config) {
			writeEnvelope(w, http.StatusServiceUnavailable, 30003, "运行依赖不可用", map[string]any{
				"status": "DOWN", "observedAt": observedAt,
			})
			return
		}
		writeEnvelope(w, http.StatusOK, 0, "ok", map[string]any{
			"status": "UP", "observedAt": observedAt,
		})
	}
}

func statusHandler(status runtimeStatus) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		available, reasons := runtimeDependenciesAvailable(status.config)
		observedAt := time.Now().UTC()
		healthState, reasonCode := "HEALTHY", any(nil)
		if !available {
			healthState = "DEGRADED"
			reasonCode = runtimeUnavailableReasonCode(reasons)
			if isStrictRuntimeProfile(status.config) {
				healthState = "UNAVAILABLE"
			}
		}
		data := map[string]any{
			"schemaVersion":  "runtime-health-status.v1",
			"lifecycleState": "RUNNING",
			"healthState":    healthState,
			"componentRole":  "compute-sandbox",
			"version":        "1.0.0",
			"executionForm":  statusExecutionForm(status.config),
			"siteId":         statusSiteID(status.config),
			"startedAt":      status.startedAt,
			"uptimeSeconds":  int64(observedAt.Sub(status.startedAt).Seconds()),
			"reasonCode":     reasonCode,
			"observedAt":     observedAt,
		}
		if status.config.RuntimeProfile != invalidRuntimeProfile && validStableID(status.config.DeploymentID) {
			data["deploymentId"] = status.config.DeploymentID
		}
		if (status.config.RuntimeProfile == developmentRuntimeProfile && validStableID(status.config.ProjectID)) ||
			(status.config.RuntimeProfile == runtimeRuntimeProfile && runtimeUUIDPattern.MatchString(status.config.ProjectID)) {
			data["projectId"] = status.config.ProjectID
		}
		if validStableID(status.config.NodeID) {
			data["nodeId"] = status.config.NodeID
		}
		statusCode, code, message := http.StatusOK, 0, "ok"
		if isStrictRuntimeProfile(status.config) && !available {
			statusCode, code, message = http.StatusServiceUnavailable, 30003, "运行依赖不可用"
		}
		writeEnvelope(w, statusCode, code, message, data)
	}
}

// statusSiteID 确保所有 profile 的状态输出符合契约；runtime 不会借用开发态默认值。
func statusSiteID(config Config) string {
	if validStableID(config.SiteID) {
		return config.SiteID
	}
	if config.RuntimeProfile == developmentRuntimeProfile {
		return defaultDevelopmentSiteID
	}
	return invalidRuntimeSiteID
}

func runtimeUnavailableReasonCode(reasons []string) string {
	for _, item := range reasons {
		switch item {
		case "RUNTIME_PROFILE_INVALID":
			return "RUNTIME_PROFILE_INVALID"
		case "RUNTIME_CONFIGURATION_INVALID", "SITE_ID_REQUIRED", "SITE_ID_INVALID", "NODE_ID_INVALID", "EXECUTION_FORM_INVALID", "DEPLOYMENT_ID_INVALID", "PROJECT_ID_INVALID", "ARTIFACT_DIGEST_INVALID":
			return "RUNTIME_IDENTITY_INVALID"
		case "PIDS_LIMIT_UNCONFIRMED", "PIDS_LIMIT_EXCEEDED":
			return "RUNTIME_PIDS_LIMIT_INVALID"
		}
	}
	return "RUNTIME_DEPENDENCY_UNAVAILABLE"
}

// runtimeDependenciesAvailable 只返回固定原因码，避免健康接口泄露二进制路径、令牌或业务数据。
func runtimeDependenciesAvailable(config Config) (bool, []string) {
	reasons := make([]string, 0)
	if isStrictRuntimeProfile(config) {
		if config.artifactUnavailable {
			reasons = append(reasons, "ARTIFACT_UNAVAILABLE")
		}
		if config.probeUnavailable {
			reasons = append(reasons, "RUNTIME_PROBE_UNAVAILABLE")
		}
		reasons = append(reasons, runtimeConfigurationReasons(config)...)
	}
	if runtime.GOOS != "linux" {
		reasons = append(reasons, "LINUX_REQUIRED")
	}
	for _, item := range []struct {
		path string
		code string
	}{
		{config.Bubblewrap, "BUBBLEWRAP_UNAVAILABLE"},
		{config.Prlimit, "PRLIMIT_UNAVAILABLE"},
		{"/usr/bin/unshare", "UNSHARE_UNAVAILABLE"},
		{config.NodeBinary, "NODE_RUNTIME_UNAVAILABLE"},
		{config.PythonBinary, "PYTHON_RUNTIME_UNAVAILABLE"},
		{setprivPath, "SETPRIV_UNAVAILABLE"},
	} {
		if !isExecutableRegularFile(item.path) {
			reasons = append(reasons, item.code)
		}
	}
	for _, item := range []struct {
		path string
		code string
	}{
		{filepath.Join(config.RuntimeDir, "node_execute.js"), "NODE_EXECUTOR_UNAVAILABLE"},
		{filepath.Join(config.RuntimeDir, "python_execute.py"), "PYTHON_EXECUTOR_UNAVAILABLE"},
	} {
		if !isReadableRegularFile(item.path) {
			reasons = append(reasons, item.code)
		}
	}
	if !isDirectory(config.DependenciesDir) {
		reasons = append(reasons, "DEPENDENCIES_DIR_UNAVAILABLE")
	}
	if isStrictRuntimeProfile(config) {
		if limit, ok := currentCgroupPIDsLimit(os.ReadFile); !ok {
			reasons = append(reasons, "PIDS_LIMIT_UNCONFIRMED")
		} else if !pidsLimitWithinConfiguredMax(limit, config.MaxPIDs) {
			reasons = append(reasons, "PIDS_LIMIT_EXCEEDED")
		}
	}
	return len(reasons) == 0, reasons
}

func runtimeConfigurationReasons(config Config) []string {
	reasons := make([]string, 0)
	if config.RuntimeProfile == invalidRuntimeProfile {
		reasons = append(reasons, "RUNTIME_PROFILE_INVALID")
	}
	if config.configurationInvalid {
		reasons = append(reasons, "RUNTIME_CONFIGURATION_INVALID")
	}
	if strings.TrimSpace(config.SiteID) == "" {
		reasons = append(reasons, "SITE_ID_REQUIRED")
	} else if !validStableID(config.SiteID) {
		reasons = append(reasons, "SITE_ID_INVALID")
	}
	if config.NodeID != "" && !validStableID(config.NodeID) {
		reasons = append(reasons, "NODE_ID_INVALID")
	}
	if !validRuntimeExecutionForm(config.ExecutionForm) {
		reasons = append(reasons, "EXECUTION_FORM_INVALID")
	}
	if !validStableID(config.DeploymentID) {
		reasons = append(reasons, "DEPLOYMENT_ID_INVALID")
	}
	if !canonicalUUID(config.ProjectID) {
		reasons = append(reasons, "PROJECT_ID_INVALID")
	}
	if !artifactDigestPattern.MatchString(config.ArtifactDigest) {
		reasons = append(reasons, "ARTIFACT_DIGEST_INVALID")
	}
	if config.MaxPIDs < 1 || config.MaxPIDs > maxConfiguredPIDs {
		reasons = append(reasons, "PIDS_LIMIT_UNCONFIRMED")
	}
	return reasons
}

func isExecutableRegularFile(path string) bool {
	info, err := os.Stat(strings.TrimSpace(path))
	return err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0o111 != 0
}

func isReadableRegularFile(path string) bool {
	info, err := os.Stat(strings.TrimSpace(path))
	return err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0o444 != 0
}

func isDirectory(path string) bool {
	info, err := os.Stat(strings.TrimSpace(path))
	return err == nil && info.IsDir()
}

func currentCgroupPIDsLimit(readFile func(string) ([]byte, error)) (int, bool) {
	if readFile == nil {
		return 0, false
	}
	cgroup, err := readFile("/proc/self/cgroup")
	if err != nil {
		return 0, false
	}
	paths := cgroupPIDsMaxPaths(string(cgroup))
	if len(paths) == 0 {
		return 0, false
	}
	minimum := 0
	for _, path := range paths {
		value, err := readFile(path)
		if err != nil {
			return 0, false
		}
		limit, finite, valid := parsePIDsMax(string(value))
		if !valid {
			return 0, false
		}
		if finite && (minimum == 0 || limit < minimum) {
			minimum = limit
		}
	}
	return minimum, minimum > 0
}

func cgroupPIDsMaxPaths(cgroup string) []string {
	for _, line := range strings.Split(cgroup, "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), ":", 3)
		if len(parts) != 3 {
			continue
		}
		if parts[1] == "" {
			return cgroupAncestorPIDsMaxPaths("/sys/fs/cgroup", parts[2])
		}
		for _, controller := range strings.Split(parts[1], ",") {
			if controller == "pids" {
				return cgroupAncestorPIDsMaxPaths("/sys/fs/cgroup/pids", parts[2])
			}
		}
	}
	return nil
}

func cgroupAncestorPIDsMaxPaths(root, cgroupPath string) []string {
	segments := strings.FieldsFunc(strings.Trim(cgroupPath, "/"), func(value rune) bool { return value == '/' })
	current := root
	paths := []string{filepath.Join(current, "pids.max")}
	for _, segment := range segments {
		if segment == "." || segment == ".." || strings.Contains(segment, "\\") {
			return nil
		}
		current = filepath.Join(current, segment)
		paths = append(paths, filepath.Join(current, "pids.max"))
	}
	return paths
}

func parsePIDsMax(value string) (limit int, finite, valid bool) {
	if strings.TrimSpace(value) == "max" {
		return 0, false, true
	}
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || parsed < 1 || parsed > int64(^uint(0)>>1) {
		return 0, false, false
	}
	return int(parsed), true, true
}

func parseFinitePIDsMax(value string) (int, bool) {
	limit, finite, valid := parsePIDsMax(value)
	return limit, finite && valid
}

func pidsLimitWithinConfiguredMax(limit, configuredMax int) bool {
	return limit >= 1 && configuredMax >= 1 && limit <= configuredMax
}

// statusExecutionForm 只报告当前服务真正可运行的形态，避免开发配置中的 native-windows 暗示可在 Windows 执行。
func statusExecutionForm(config Config) string {
	if isStrictRuntimeProfile(config) {
		if validRuntimeExecutionForm(config.ExecutionForm) {
			return config.ExecutionForm
		}
		return defaultRuntimeExecutionForm
	}
	if config.ExecutionForm == "k3s-workload" || config.ExecutionForm == "native-linux" {
		return config.ExecutionForm
	}
	return defaultDevelopmentExecutionForm
}

var errExecutionTimeout = errors.New("execution timeout")

type isolatedResult struct {
	Output []byte
	Stderr string
}

func runIsolated(ctx context.Context, config Config, language, runtimeScript string, input any, timeout time.Duration, projectID string) (isolatedResult, error) {
	payload, err := json.Marshal(input)
	if err != nil {
		return isolatedResult{}, fmt.Errorf("编码沙箱输入失败: %w", err)
	}
	if len(payload) > maxRequestBytes {
		return isolatedResult{}, fmt.Errorf("沙箱输入超过 %d 字节", maxRequestBytes)
	}
	args, err := buildSandboxArgs(config, language, runtimeScript, projectID)
	if err != nil {
		return isolatedResult{}, err
	}
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.Command(args[0], args[1:]...)
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
		if errors.Is(execCtx.Err(), context.DeadlineExceeded) {
			return isolatedResult{}, errExecutionTimeout
		}
		return isolatedResult{}, execCtx.Err()
	}
	if stdout.overflow {
		return isolatedResult{}, fmt.Errorf("脚本输出超过 %d 字节", maxOutputBytes)
	}
	return isolatedResult{Output: stdout.Bytes(), Stderr: stderr.String()}, nil
}

func buildSandboxArgs(config Config, language, runtimeScript, projectID string) ([]string, error) {
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("计算沙箱仅支持 Linux bubblewrap/prlimit 运行环境")
	}
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
	args := []string{
		"/usr/bin/unshare", "--net", "--", config.Bubblewrap, "--die-with-parent", "--unshare-ipc", "--unshare-pid", "--unshare-uts", "--unshare-cgroup-try", "--clearenv",
		"--ro-bind", "/usr", "/usr", "--ro-bind-try", "/bin", "/bin", "--ro-bind-try", "/lib", "/lib",
		"--ro-bind-try", "/lib64", "/lib64", "--ro-bind-try", "/usr/local", "/usr/local",
		// 容器已按非 root 身份运行；嵌套 user namespace 无权挂载新的 procfs，也无法映射另一个 UID。
		// 脚本运行不依赖 /proc，因此不把外层 procfs 暴露给隔离进程，并保持调用者 UID。
		"--ro-bind", config.RuntimeDir, "/runtime", "--dev", "/dev", "--tmpfs", "/tmp",
	}
	if isStrictRuntimeProfile(config) {
		// 运行态依赖根由部署层按 Release/Artifact 精确只读挂载，不从开发缓存按请求 projectId 选择目录。
		args = append(args, "--ro-bind", config.DependenciesDir, "/dependencies")
	} else if validDependencyProjectID(projectID) {
		projectDir := filepath.Join(config.DependenciesDir, projectID)
		args = append(args, "--ro-bind-try", projectDir, "/dependencies")
	}
	args = append(args,
		"--chdir", "/tmp", "--setenv", "PATH", "/usr/local/bin:/usr/bin:/bin", "--setenv", "LANG", "C.UTF-8",
		"--cap-drop", "ALL", "--cap-add", "CAP_SETUID", "--cap-add", "CAP_SETGID", "--cap-add", "CAP_SETPCAP",
		"/usr/local/bin/compute-sandbox", "internal-exec", config.Prlimit,
		"--as=1073741824", "--nofile=64", "--fsize=4194304", "--cpu=120", "--", binary, filepath.Join("/runtime", runtimeScript),
	)
	return args, nil
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

func developmentOnlyHandler(config Config, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if isStrictRuntimeProfile(config) {
			writeStructuredError(w, http.StatusForbidden, "RUNTIME_PROFILE_RESTRICTED", "运行态禁止修改或安装沙箱依赖")
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

func writeStructuredError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{"error": message, "errorCode": code})
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeEnvelope(w http.ResponseWriter, status, code int, message string, data any) {
	writeJSON(w, status, map[string]any{
		"code": code, "msg": message, "data": data, "reqId": requestID(),
	})
}

func requestID() string {
	return fmt.Sprintf("req_%d", time.Now().UTC().UnixNano())
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
