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
import urllib.request
import urllib.error

input_data = json.loads(os.environ.get("COMPUTE_INPUT", "{}"))
sdk_context = json.loads(os.environ.get("COMPUTE_CONTEXT", "{}"))
user_script = os.environ.get("COMPUTE_SCRIPT", "")
callback_url = os.environ.get("COMPUTE_CALLBACK_URL", "")
callback_token = os.environ.get("COMPUTE_CALLBACK_TOKEN", "")
side_effects = []

def call_compute_callback(path, payload):
    if not callback_url:
        raise RuntimeError("compute SDK callback is not configured")
    data = json.dumps(payload or {}).encode("utf-8")
    request = urllib.request.Request(
        callback_url + path,
        data=data,
        method="POST",
        headers={
            "Content-Type": "application/json",
            "Authorization": "Bearer " + callback_token,
        },
    )
    try:
        with urllib.request.urlopen(request, timeout=10) as response:
            body = response.read().decode("utf-8")
            return json.loads(body or "{}")
    except urllib.error.HTTPError as error:
        body = error.read().decode("utf-8")
        try:
            parsed = json.loads(body or "{}")
            message = parsed.get("error") or str(error)
        except Exception:
            message = str(error)
        raise RuntimeError(message)

class DatapointSDK:
    def get(self, path):
        key = str(path or "")
        item = sdk_context.get("datapoints", {}).get(key)
        if item is None:
            raise RuntimeError("ctx.datapoint.get is not declared or prefetched: " + key)
        return item.get("value")

    def meta(self, path):
        key = str(path or "")
        item = sdk_context.get("datapoints", {}).get(key)
        if item is None:
            raise RuntimeError("ctx.datapoint.meta is not declared or prefetched: " + key)
        return item

class SQLSDK:
    def query(self, key, params=None):
        name = str(key or "")
        item = sdk_context.get("sql", {}).get(name)
        if item is not None:
            return item
        if not callback_url:
            raise RuntimeError("ctx.sql.query is not declared or prefetched: " + name)
        result = call_compute_callback("/sql/query", {"key": name, "parameters": params or {}})
        return result.get("result")

class MqttSDK:
    def publish(self, source, topic, payload):
        if callback_url:
            effect = call_compute_callback("/mqtt/publish", {"source": source, "topic": topic, "payload": payload})
        else:
            effect = {"type": "mqtt.publish", "source": source, "topic": topic, "payload": payload, "accepted": False, "published": False, "reason": "compute SDK callback is not configured"}
        side_effects.append(effect)
        return effect

class ComputeSDK:
    def __init__(self):
        self.datapoint = DatapointSDK()
        self.sql = SQLSDK()
        self.mqtt = MqttSDK()
        self.sideEffects = side_effects

ctx = ComputeSDK()
scope = {
    "input": input_data,
    "ctx": ctx,
    "result": None,
}

try:
    exec(user_script, {}, scope)
    payload = {"result": scope.get("result"), "sideEffects": side_effects}
    sys.stdout.write("__DATA_SERVICE_RESULT__:" + json.dumps(payload))
except Exception:
    traceback.print_exc(file=sys.stderr)
    sys.exit(1)
`

// PythonRunner executes Python compute scripts through a short-lived Python process.
type PythonRunner struct {
	binaryPath string
	runtimeDir string
}

// NewPythonRunner creates a Python compute runner.
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

// Run executes a Python compute script under the requested timeout.
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

	contextPayload, err := json.Marshal(request.SDKContext)
	if err != nil {
		return ExecuteResult{}, fmt.Errorf("序列化 SDK 上下文失败: %w", err)
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	startedAt := time.Now()
	cmd := exec.CommandContext(execCtx, binaryPath, "-c", pythonBootstrapScript)
	cmd.Dir = r.runtimeDir
	cmd.Env = append(
		os.Environ(),
		"COMPUTE_INPUT="+string(inputPayload),
		"COMPUTE_CONTEXT="+string(contextPayload),
		"COMPUTE_SCRIPT="+request.Script,
		"COMPUTE_CALLBACK_URL="+strings.TrimSpace(request.CallbackURL),
		"COMPUTE_CALLBACK_TOKEN="+strings.TrimSpace(request.CallbackToken),
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

	parsedOutput, sideEffects, parseErr := parseScriptEnvelope(result.Stdout)
	if parseErr != nil {
		return result, parseErr
	}
	result.Output = parsedOutput
	result.SideEffects = sideEffects
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
