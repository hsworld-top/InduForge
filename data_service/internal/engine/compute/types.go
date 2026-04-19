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
	Script  string
	Input   map[string]any
	Timeout time.Duration
}

// ExecuteResult 描述脚本执行结果。
type ExecuteResult struct {
	Output   any
	Stdout   string
	Stderr   string
	Duration time.Duration
}

// Runner 抽象不同脚本语言的执行器。
type Runner interface {
	Run(ctx context.Context, request ExecuteRequest) (ExecuteResult, error)
}
