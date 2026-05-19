package compute

import (
	"context"
	"errors"
	"time"
)

// ErrTimeout 表示脚本执行超过配置的超时时间。
var ErrTimeout = errors.New("compute execution timeout")

// ExecuteRequest 描述一次脚本执行请求。
type ExecuteRequest struct {
	Script        string
	Input         map[string]any
	SDKContext    SDKContext
	Timeout       time.Duration
	CallbackURL   string
	CallbackToken string
}

// ExecuteResult 描述脚本执行结果。
type ExecuteResult struct {
	Output      any
	SideEffects []any
	Stdout      string
	Stderr      string
	Duration    time.Duration
}

// SyntaxCheckRequest 描述一次脚本语法检查请求。
type SyntaxCheckRequest struct {
	Script  string
	Timeout time.Duration
}

// SyntaxDiagnostic 描述脚本语法诊断。
type SyntaxDiagnostic struct {
	Severity  string `json:"severity"`
	Message   string `json:"message"`
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	EndLine   int    `json:"endLine"`
	EndColumn int    `json:"endColumn"`
	Source    string `json:"source"`
}

// SyntaxCheckResult 描述脚本语法检查结果。
type SyntaxCheckResult struct {
	Diagnostics []SyntaxDiagnostic `json:"diagnostics"`
}

// SDKContext 是由服务端预取后注入脚本运行时的受控上下文。
type SDKContext struct {
	Datapoints map[string]SDKDataPointValue `json:"datapoints"`
	Variables  map[string]any               `json:"variables"`
	SQL        map[string]any               `json:"sql"`
	Metadata   map[string]any               `json:"metadata"`
}

// SDKDataPointValue 是 ctx.datapoint.get 可读取的数据点快照。
type SDKDataPointValue struct {
	Path      string `json:"path"`
	Value     any    `json:"value"`
	Quality   string `json:"quality"`
	Timestamp string `json:"timestamp"`
	Status    string `json:"status"`
}

// Runner 抽象不同脚本语言的执行器。
type Runner interface {
	Run(ctx context.Context, request ExecuteRequest) (ExecuteResult, error)
}

// SyntaxChecker 抽象不同脚本语言的语法检查器。
type SyntaxChecker interface {
	CheckSyntax(ctx context.Context, request SyntaxCheckRequest) (SyntaxCheckResult, error)
}
