package compute

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	defaultNodeBinary  = "node"
	defaultRuntimeDir  = "."
	defaultRunTimeout  = 3 * time.Second
	maxErrorLogPreview = 500
)

const nodeBootstrapScript = `
const input = JSON.parse(process.env.COMPUTE_INPUT || "{}");
const userScript = process.env.COMPUTE_SCRIPT || "";
(async () => {
  const runtime = new Function(
    "input",
    "\"use strict\";\nlet result = null;\n" + userScript + "\nreturn typeof result === 'undefined' ? null : result;"
  );
  const resolved = await runtime(input);
  process.stdout.write("__DATA_SERVICE_RESULT__:" + JSON.stringify({ result: resolved }));
})().catch((error) => {
  const text = error && error.stack ? error.stack : String(error);
  process.stderr.write(text);
  process.exit(1);
});
`

// NodeRunner 负责执行 JavaScript 计算脚本。
type NodeRunner struct {
	binaryPath string
	runtimeDir string
}

// NewNodeRunner 创建 Node.js 执行器。
func NewNodeRunner(binaryPath, runtimeDir string) *NodeRunner {
	binaryPath = strings.TrimSpace(binaryPath)
	if binaryPath == "" {
		binaryPath = defaultNodeBinary
	}

	runtimeDir = strings.TrimSpace(runtimeDir)
	if runtimeDir == "" {
		runtimeDir = defaultRuntimeDir
	}

	return &NodeRunner{
		binaryPath: binaryPath,
		runtimeDir: runtimeDir,
	}
}

// Run 在受限超时上下文中执行 JS 脚本。
func (r *NodeRunner) Run(ctx context.Context, request ExecuteRequest) (ExecuteResult, error) {
	if r == nil {
		return ExecuteResult{}, fmt.Errorf("node runner 未初始化")
	}

	script := strings.TrimSpace(request.Script)
	if script == "" {
		return ExecuteResult{}, fmt.Errorf("script 不能为空")
	}

	timeout := request.Timeout
	if timeout <= 0 {
		timeout = defaultRunTimeout
	}

	inputPayload, err := json.Marshal(request.Input)
	if err != nil {
		return ExecuteResult{}, fmt.Errorf("序列化输入参数失败: %w", err)
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	startedAt := time.Now()
	cmd := exec.CommandContext(execCtx, r.binaryPath, "-e", nodeBootstrapScript)
	cmd.Dir = r.runtimeDir
	cmd.Env = append(
		os.Environ(),
		"NODE_ENV=production",
		"COMPUTE_INPUT="+string(inputPayload),
		"COMPUTE_SCRIPT="+request.Script,
	)

	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer
	cmd.Stdout = &stdoutBuffer
	cmd.Stderr = &stderrBuffer

	err = cmd.Run()
	result := ExecuteResult{
		Stdout:   stdoutBuffer.String(),
		Stderr:   stderrBuffer.String(),
		Duration: time.Since(startedAt),
	}

	if err != nil {
		if errors.Is(execCtx.Err(), context.DeadlineExceeded) {
			return result, ErrTimeout
		}

		errorMessage := strings.TrimSpace(stderrBuffer.String())
		if len(errorMessage) > maxErrorLogPreview {
			errorMessage = errorMessage[:maxErrorLogPreview] + "..."
		}
		if errorMessage == "" {
			errorMessage = err.Error()
		}
		return result, fmt.Errorf("node 执行失败: %s", errorMessage)
	}

	parsedOutput, parseErr := parseScriptResult(result.Stdout)
	if parseErr != nil {
		return result, parseErr
	}
	result.Output = parsedOutput
	return result, nil
}
