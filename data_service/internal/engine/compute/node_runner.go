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
const sdkContext = JSON.parse(process.env.COMPUTE_CONTEXT || "{}");
const userScript = process.env.COMPUTE_SCRIPT || "";
const callbackURL = process.env.COMPUTE_CALLBACK_URL || "";
const callbackToken = process.env.COMPUTE_CALLBACK_TOKEN || "";

(async () => {
  const sideEffects = [];

  async function callComputeCallback(path, payload) {
    if (!callbackURL) {
      throw new Error("compute SDK callback is not configured");
    }
    const response = await fetch(callbackURL + path, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "Authorization": "Bearer " + callbackToken
      },
      body: JSON.stringify(payload || {})
    });
    let body = {};
    try {
      body = await response.json();
    } catch (_) {}
    if (!response.ok) {
      throw new Error((body && body.error) || ("compute SDK callback failed: " + response.status));
    }
    return body;
  }

  const ctx = {
    datapoint: {
      get(path) {
        const key = String(path || "");
        const item = sdkContext.datapoints && sdkContext.datapoints[key];
        if (!item) throw new Error("ctx.datapoint.get is not declared or prefetched: " + key);
        return item.value;
      },
      meta(path) {
        const key = String(path || "");
        const item = sdkContext.datapoints && sdkContext.datapoints[key];
        if (!item) throw new Error("ctx.datapoint.meta is not declared or prefetched: " + key);
        return item;
      }
    },
    sql: {
      async query(key, params) {
        const name = String(key || "");
        const result = sdkContext.sql && sdkContext.sql[name];
        if (result) return result;
        if (!callbackURL) throw new Error("ctx.sql.query is not declared or prefetched: " + name);
        const dynamicResult = await callComputeCallback("/sql/query", { key: name, parameters: params || {} });
        return dynamicResult.result;
      }
    },
    mqtt: {
      async publish(source, topic, payload) {
        if (callbackURL) {
          const effect = await callComputeCallback("/mqtt/publish", { source, topic, payload });
          sideEffects.push(effect);
          return effect;
        }
        const effect = { type: "mqtt.publish", source, topic, payload, accepted: false, published: false, reason: "compute SDK callback is not configured" };
        sideEffects.push(effect);
        return effect;
      }
    },
    sideEffects
  };

  const runtime = new Function(
    "input",
    "ctx",
    "\"use strict\";\nlet result = null;\n" + userScript + "\nreturn typeof result === 'undefined' ? null : result;"
  );
  const resolved = await runtime(input, ctx);
  process.stdout.write("__DATA_SERVICE_RESULT__:" + JSON.stringify({ result: resolved, sideEffects }));
})().catch((error) => {
  const text = error && error.stack ? error.stack : String(error);
  process.stderr.write(text);
  process.exit(1);
});
`

// NodeRunner executes JavaScript compute scripts through a short-lived Node process.
type NodeRunner struct {
	binaryPath string
	runtimeDir string
}

// NewNodeRunner creates a JavaScript compute runner.
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

// Run executes a JavaScript compute script under the requested timeout.
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

	contextPayload, err := json.Marshal(request.SDKContext)
	if err != nil {
		return ExecuteResult{}, fmt.Errorf("序列化 SDK 上下文失败: %w", err)
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
		return result, fmt.Errorf("node 执行失败: %s", errorMessage)
	}

	parsedOutput, sideEffects, parseErr := parseScriptEnvelope(result.Stdout)
	if parseErr != nil {
		return result, parseErr
	}
	result.Output = parsedOutput
	result.SideEffects = sideEffects
	return result, nil
}
