package compute

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestNodeRunnerCheckSyntax(t *testing.T) {
	runner := NewNodeRunner("", ".")

	t.Run("valid async script", func(t *testing.T) {
		result, err := runner.CheckSyntax(context.Background(), SyntaxCheckRequest{
			Script:  "const value = await Promise.resolve(1);\nreturn value;",
			Timeout: 3 * time.Second,
		})
		if err != nil {
			t.Fatalf("CheckSyntax() error = %v", err)
		}
		if len(result.Diagnostics) != 0 {
			t.Fatalf("diagnostics = %+v, want empty", result.Diagnostics)
		}
	})

	t.Run("invalid script returns diagnostic", func(t *testing.T) {
		result, err := runner.CheckSyntax(context.Background(), SyntaxCheckRequest{
			Script:  "const value = ;",
			Timeout: 3 * time.Second,
		})
		if err != nil {
			t.Fatalf("CheckSyntax() error = %v", err)
		}
		if len(result.Diagnostics) != 1 {
			t.Fatalf("diagnostics len = %d, want 1", len(result.Diagnostics))
		}
		got := result.Diagnostics[0]
		if got.Line < 1 || got.Column < 1 {
			t.Fatalf("diagnostic position = %d:%d, want positive position", got.Line, got.Column)
		}
		if got.Severity != "error" || got.Source != "javascript" {
			t.Fatalf("diagnostic = %+v, want javascript error", got)
		}
	})
}

func TestPythonRunnerCheckSyntax(t *testing.T) {
	runner := NewPythonRunner("", ".")

	t.Run("valid script", func(t *testing.T) {
		result, err := runner.CheckSyntax(context.Background(), SyntaxCheckRequest{
			Script:  "def main(argv, dp, ctx):\n    return 1",
			Timeout: 3 * time.Second,
		})
		if err != nil {
			t.Fatalf("CheckSyntax() error = %v", err)
		}
		if len(result.Diagnostics) != 0 {
			t.Fatalf("diagnostics = %+v, want empty", result.Diagnostics)
		}
	})

	t.Run("invalid script returns diagnostic", func(t *testing.T) {
		result, err := runner.CheckSyntax(context.Background(), SyntaxCheckRequest{
			Script:  "def main(argv, dp, ctx):\n    value = ",
			Timeout: 3 * time.Second,
		})
		if err != nil {
			if strings.Contains(err.Error(), "python 可执行文件不可用") {
				t.Skip(err)
			}
			t.Fatalf("CheckSyntax() error = %v", err)
		}
		if len(result.Diagnostics) != 1 {
			t.Fatalf("diagnostics len = %d, want 1", len(result.Diagnostics))
		}
		got := result.Diagnostics[0]
		if got.Line < 1 || got.Column < 1 {
			t.Fatalf("diagnostic position = %d:%d, want positive position", got.Line, got.Column)
		}
		if got.Severity != "error" || got.Source != "python" {
			t.Fatalf("diagnostic = %+v, want python error", got)
		}
	})
}

func TestNodeRunnerRunReturnsScriptReturnValue(t *testing.T) {
	runner := NewNodeRunner("", ".")
	result, err := runner.Run(context.Background(), ExecuteRequest{
		Script:  "return argv[0] + tag1;",
		Input:   map[string]any{"argv": []any{2}},
		Timeout: 3 * time.Second,
		SDKContext: SDKContext{
			Variables: map[string]any{"tag1": float64(3)},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Output != float64(5) {
		t.Fatalf("Output = %v, want 5", result.Output)
	}
}

func TestNodeRunnerRunReturnsScriptErrorMessage(t *testing.T) {
	runner := NewNodeRunner("", ".")
	result, err := runner.Run(context.Background(), ExecuteRequest{
		Script:  "throw new Error('aaa');",
		Input:   map[string]any{"argv": []any{}},
		Timeout: 3 * time.Second,
	})
	if err == nil {
		t.Fatal("Run() error = nil, want script error")
	}
	if !strings.Contains(err.Error(), "aaa") {
		t.Fatalf("Run() error = %v, want contain aaa", err)
	}
	if !strings.Contains(result.Stderr, "aaa") {
		t.Fatalf("Stderr = %q, want contain aaa", result.Stderr)
	}
}

func TestPythonRunnerRunReturnsMainValue(t *testing.T) {
	runner := NewPythonRunner("", ".")
	result, err := runner.Run(context.Background(), ExecuteRequest{
		Script:  "def main(argv, dp, ctx):\n    return argv[0] + tag1",
		Input:   map[string]any{"argv": []any{2}},
		Timeout: 3 * time.Second,
		SDKContext: SDKContext{
			Variables: map[string]any{"tag1": 3},
		},
	})
	if err != nil {
		if strings.Contains(err.Error(), "python 可执行文件不可用") {
			t.Skip(err)
		}
		t.Fatalf("Run() error = %v", err)
	}
	if result.Output != float64(5) {
		t.Fatalf("Output = %v, want 5", result.Output)
	}
}

func TestPythonRunnerRunReturnsScriptErrorMessage(t *testing.T) {
	runner := NewPythonRunner("", ".")
	result, err := runner.Run(context.Background(), ExecuteRequest{
		Script:  "def main(argv, dp, ctx):\n    raise RuntimeError('aaa')",
		Input:   map[string]any{"argv": []any{}},
		Timeout: 3 * time.Second,
	})
	if err != nil && strings.Contains(err.Error(), "python 可执行文件不可用") {
		t.Skip(err)
	}
	if err == nil {
		t.Fatal("Run() error = nil, want script error")
	}
	if !strings.Contains(err.Error(), "aaa") {
		t.Fatalf("Run() error = %v, want contain aaa", err)
	}
	if !strings.Contains(result.Stderr, "aaa") {
		t.Fatalf("Stderr = %q, want contain aaa", result.Stderr)
	}
}
