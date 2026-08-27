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
	ProjectID     string
	Script        string
	Input         map[string]any
	SDKContext    SDKContext
	Dependencies  []RuntimeDependency
	Timeout       time.Duration
	CallbackURL   string
	CallbackToken string
}

// RuntimeDependency 是已在工程沙箱环境安装并允许当前脚本加载的依赖。
type RuntimeDependency struct {
	Language    string `json:"language"`
	PackageName string `json:"packageName"`
	ImportName  string `json:"importName"`
	Version     string `json:"version"`
}

type DependencyInstallRequest struct {
	ProjectID string `json:"projectId"`
	RuntimeDependency
}

type DependencyImportRequest struct {
	ProjectID string
	Language  string
	Filename  string
	Content   []byte
}

type DependencyManager interface {
	InstallDependency(ctx context.Context, request DependencyInstallRequest) (RuntimeDependency, error)
	ImportDependency(ctx context.Context, request DependencyImportRequest) (RuntimeDependency, error)
	UninstallDependency(ctx context.Context, request DependencyInstallRequest) error
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

// SandboxCapabilities 是独立计算沙箱声明的真实能力，data_service 不再维护硬编码能力表。
type SandboxCapabilities struct {
	Available      bool                   `json:"available"`
	Reason         string                 `json:"reason"`
	ServiceVersion string                 `json:"serviceVersion"`
	Languages      []LanguageCapability   `json:"languages"`
	SDK            []string               `json:"sdk"`
	Dependencies   []DependencyCapability `json:"dependencies"`
	Triggers       []string               `json:"triggers"`
	Limits         SandboxLimits          `json:"limits"`
}

type LanguageCapability struct {
	Language string `json:"language"`
	Version  string `json:"version"`
}

type DependencyCapability struct {
	Language string `json:"language"`
	Name     string `json:"name"`
	Version  string `json:"version"`
}

type SandboxLimits struct {
	MaxExecutionTimeMS int `json:"maxExecutionTimeMs"`
	MaxInputBytes      int `json:"maxInputBytes"`
	MaxOutputBytes     int `json:"maxOutputBytes"`
	MaxLogBytes        int `json:"maxLogBytes"`
}

// SDKContext 是由服务端预取后注入脚本运行时的受控上下文。
type SDKContext struct {
	Datapoints    map[string]SDKDataPointValue `json:"datapoints"`
	PointBindings map[string]string            `json:"pointBindings"`
	Variables     map[string]any               `json:"variables"`
	SQL           map[string]any               `json:"sql"`
	Metadata      map[string]any               `json:"metadata"`
}

// SDKDataPointValue 是计算开发态预取的数据点契约与当前值快照。
type SDKDataPointValue struct {
	ID              string                   `json:"id"`
	Path            string                   `json:"path"`
	Name            string                   `json:"name"`
	DisplayName     string                   `json:"displayName"`
	DataType        string                   `json:"dataType"`
	SourceType      string                   `json:"sourceType"`
	SourceID        *string                  `json:"sourceId"`
	Value           any                      `json:"value"`
	DefaultValue    *string                  `json:"defaultValue"`
	Quality         string                   `json:"quality"`
	Timestamp       string                   `json:"timestamp"`
	ObservedAt      *string                  `json:"observedAt"`
	SourceTimestamp *string                  `json:"sourceTimestamp"`
	Status          string                   `json:"status"`
	Unit            *string                  `json:"unit"`
	Precision       *int                     `json:"precision"`
	Min             *float64                 `json:"min"`
	Max             *float64                 `json:"max"`
	Tags            []any                    `json:"tags"`
	Attributes      map[string]string        `json:"attributes"`
	Capabilities    SDKDataPointCapabilities `json:"capabilities"`
}

// SDKDataPointCapabilities 保持稳定方法集合；false 表示方法存在但调用会返回不支持。
type SDKDataPointCapabilities struct {
	Get       bool `json:"get"`
	Read      bool `json:"read"`
	Peek      bool `json:"peek"`
	Set       bool `json:"set"`
	Subscribe bool `json:"subscribe"`
	History   bool `json:"history"`
	Refresh   bool `json:"refresh"`
	Run       bool `json:"run"`
	Execute   bool `json:"execute"`
	Publish   bool `json:"publish"`
}

// Runner 抽象不同脚本语言的执行器。
type Runner interface {
	Run(ctx context.Context, request ExecuteRequest) (ExecuteResult, error)
}

// SyntaxChecker 抽象不同脚本语言的语法检查器。
type SyntaxChecker interface {
	CheckSyntax(ctx context.Context, request SyntaxCheckRequest) (SyntaxCheckResult, error)
}

// CapabilityProvider 返回执行器后端的实时能力和健康状态。
type CapabilityProvider interface {
	Capabilities(ctx context.Context) (SandboxCapabilities, error)
}
