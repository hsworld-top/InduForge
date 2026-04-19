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

const defaultPythonBinary = "python"
const fallbackPythonBinary = "python3"
const probePythonTimeout = 2 * time.Second

const pythonBootstrapScript = `
import json
import os
import sys
import traceback

input_data = json.loads(os.environ.get("COMPUTE_INPUT", "{}"))
user_script = os.environ.get("COMPUTE_SCRIPT", "")
scope = {
    "input": input_data,
    "result": None,
}

try:
    exec(user_script, {}, scope)
    payload = {"result": scope.get("result")}
    sys.stdout.write("__DATA_SERVICE_RESULT__:" + json.dumps(payload))
except Exception:
    traceback.print_exc(file=sys.stderr)
    sys.exit(1)
`

// PythonRunner 负责执行 Python 计算脚本。
type PythonRunner struct {
	binaryPath string
	runtimeDir string
}

// NewPythonRunner 创建 Python 执行器。
func NewPythonRunner(binaryPath, runtimeDir string) *PythonRunner {
	binaryPath = strings.TrimSpace(binaryPath)
	if binaryPath == "" {
		if preferred, err := exec.LookPath(fallbackPythonBinary); err == nil {
			binaryPath = preferred
		} else {
			binaryPath = defaultPythonBinary
		}
	}

	runtimeDir = strings.TrimSpace(runtimeDir)
	if runtimeDir == "" {
		runtimeDir = defaultRuntimeDir
	}

	return &PythonRunner{
		binaryPath: binaryPath,
		runtimeDir: runtimeDir,
	}
}

// Run 在受限超时上下文中执行 Python 脚本。
func (r *PythonRunner) Run(ctx context.Context, request ExecuteRequest) (ExecuteResult, error) {
	if r == nil {
		return ExecuteResult{}, fmt.Errorf("python runner 未初始化")
	}

	script := strings.TrimSpace(request.Script)
	if script == "" {
		return ExecuteResult{}, fmt.Errorf("script 不能为空")
	}

	timeout := request.Timeout
	if timeout <= 0 {
		timeout = defaultRunTimeout
	}

	probeCtx, probeCancel := context.WithTimeout(ctx, probePythonTimeout)
	defer probeCancel()

	binaryPath, resolveErr := resolvePythonBinary(probeCtx, r.binaryPath)
	if resolveErr != nil {
		return ExecuteResult{}, resolveErr
	}

	inputPayload, err := json.Marshal(request.Input)
	if err != nil {
		return ExecuteResult{}, fmt.Errorf("序列化输入参数失败: %w", err)
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	startedAt := time.Now()
	cmd := exec.CommandContext(execCtx, binaryPath, "-c", pythonBootstrapScript)
	cmd.Dir = r.runtimeDir
	cmd.Env = append(
		os.Environ(),
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
		return result, fmt.Errorf("python 执行失败: %s", errorMessage)
	}

	parsedOutput, parseErr := parseScriptResult(result.Stdout)
	if parseErr != nil {
		return result, parseErr
	}
	result.Output = parsedOutput
	return result, nil
}

func resolvePythonBinary(ctx context.Context, preferred string) (string, error) {
	candidates := []string{strings.TrimSpace(preferred)}
	if strings.TrimSpace(preferred) != fallbackPythonBinary {
		candidates = append(candidates, fallbackPythonBinary)
	}
	if strings.TrimSpace(preferred) != defaultPythonBinary {
		candidates = append(candidates, defaultPythonBinary)
	}

	visited := make(map[string]struct{})
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if _, exists := visited[candidate]; exists {
			continue
		}
		visited[candidate] = struct{}{}

		path, err := exec.LookPath(candidate)
		if err != nil {
			continue
		}
		if probeErr := probePythonBinary(ctx, path); probeErr != nil {
			continue
		}
		return path, nil
	}

	return "", fmt.Errorf("python 可执行文件不可用")
}

func probePythonBinary(ctx context.Context, binaryPath string) error {
	cmd := exec.CommandContext(ctx, binaryPath, "--version")
	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer
	cmd.Stdout = &stdoutBuffer
	cmd.Stderr = &stderrBuffer
	return cmd.Run()
}
