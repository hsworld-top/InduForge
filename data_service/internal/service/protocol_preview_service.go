package service

import (
	"context"
	"net/http"
	"strings"
	"time"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

const (
	defaultProtocolPreviewLimit     = 10
	maxProtocolPreviewLimit         = 100
	defaultProtocolPreviewTimeoutMS = 5000
	maxProtocolPreviewTimeoutMS     = 30000
)

// ProtocolPreviewInput 表示统一短时抓样请求。
type ProtocolPreviewInput struct {
	Limit     int            `json:"limit"`
	TimeoutMS int            `json:"timeoutMs"`
	Options   map[string]any `json:"options"`
}

// ProtocolPreviewResult 是所有 Phase 1 协议统一返回给 datacenter 的抓样结果。
type ProtocolPreviewResult struct {
	Protocol      string         `json:"protocol"`
	ConnectionID  string         `json:"connectionId"`
	Status        string         `json:"status"`
	Schema        map[string]any `json:"schema"`
	Samples       []any          `json:"samples"`
	RawPayload    any            `json:"rawPayload,omitempty"`
	Diagnostics   map[string]any `json:"diagnostics"`
	DurationMS    int64          `json:"durationMs"`
	Truncated     bool           `json:"truncated"`
	EffectiveTime time.Time      `json:"-"`
}

// ProtocolPreviewAdapterInput 是协议 adapter 的归一化输入。
type ProtocolPreviewAdapterInput struct {
	Connection repository.ProtocolPreviewConnectionRecord
	Limit      int
	Timeout    time.Duration
	Options    map[string]any
}

// ProtocolPreviewAdapter 定义一次性短时抓样协议适配器。
type ProtocolPreviewAdapter interface {
	Preview(ctx context.Context, input ProtocolPreviewAdapterInput) (*ProtocolPreviewResult, error)
}

type protocolPreviewRepository interface {
	GetPreviewConnection(ctx context.Context, projectID, connectionID string) (*repository.ProtocolPreviewConnectionRecord, error)
	CreateAccessSourceRecord(ctx context.Context, params repository.CreateAccessSourceRecordParams) error
}

// ProtocolPreviewService 只负责开发态短时抓样，不创建长期运行任务。
type ProtocolPreviewService struct {
	repository protocolPreviewRepository
	adapters   map[string]ProtocolPreviewAdapter
}

// NewProtocolPreviewService 创建统一协议抓样服务。
func NewProtocolPreviewService(repo protocolPreviewRepository, adapters map[string]ProtocolPreviewAdapter) *ProtocolPreviewService {
	normalized := make(map[string]ProtocolPreviewAdapter, len(adapters))
	for protocol, adapter := range adapters {
		if adapter == nil {
			continue
		}
		normalized[strings.ToLower(strings.TrimSpace(protocol))] = adapter
	}
	return &ProtocolPreviewService{repository: repo, adapters: normalized}
}

// Preview 执行一次短时真实抓样，并写入不含原始样本的接入源记录摘要。
func (s *ProtocolPreviewService) Preview(ctx context.Context, projectID, connectionID string, input ProtocolPreviewInput) (*ProtocolPreviewResult, error) {
	if s == nil || s.repository == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "协议预览服务未初始化")
	}
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateConnectionID(connectionID); err != nil {
		return nil, err
	}

	connection, err := s.repository.GetPreviewConnection(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	protocol := strings.ToLower(strings.TrimSpace(connection.Type))
	adapter := s.adapters[protocol]
	if adapter == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "该协议暂不支持短时预览: "+protocol)
	}

	limit, timeout := normalizeProtocolPreviewLimits(input)
	previewCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	startedAt := time.Now()
	result, err := adapter.Preview(previewCtx, ProtocolPreviewAdapterInput{
		Connection: *connection,
		Limit:      limit,
		Timeout:    timeout,
		Options:    cloneMap(input.Options),
	})
	if err != nil {
		result = failedProtocolPreviewResult(protocol, connectionID, startedAt, err)
		_ = s.recordPreview(ctx, connection, result, err)
		return result, err
	}
	if result == nil {
		result = &ProtocolPreviewResult{}
	}
	fillProtocolPreviewDefaults(result, protocol, connectionID, startedAt)
	if recordErr := s.recordPreview(ctx, connection, result, nil); recordErr != nil {
		return nil, recordErr
	}
	return result, nil
}

func normalizeProtocolPreviewLimits(input ProtocolPreviewInput) (int, time.Duration) {
	limit := input.Limit
	if limit <= 0 {
		limit = defaultProtocolPreviewLimit
	}
	if limit > maxProtocolPreviewLimit {
		limit = maxProtocolPreviewLimit
	}
	timeoutMS := input.TimeoutMS
	if timeoutMS <= 0 {
		timeoutMS = defaultProtocolPreviewTimeoutMS
	}
	if timeoutMS > maxProtocolPreviewTimeoutMS {
		timeoutMS = maxProtocolPreviewTimeoutMS
	}
	return limit, time.Duration(timeoutMS) * time.Millisecond
}

func fillProtocolPreviewDefaults(result *ProtocolPreviewResult, protocol, connectionID string, startedAt time.Time) {
	if result.Protocol == "" {
		result.Protocol = protocol
	}
	if result.ConnectionID == "" {
		result.ConnectionID = connectionID
	}
	if result.Status == "" {
		result.Status = "ok"
	}
	if result.Schema == nil {
		result.Schema = map[string]any{}
	}
	if result.Samples == nil {
		result.Samples = []any{}
	}
	if result.Diagnostics == nil {
		result.Diagnostics = map[string]any{}
	}
	if result.DurationMS <= 0 {
		result.DurationMS = time.Since(startedAt).Milliseconds()
	}
	if result.EffectiveTime.IsZero() {
		result.EffectiveTime = time.Now().UTC()
	}
}

func failedProtocolPreviewResult(protocol, connectionID string, startedAt time.Time, err error) *ProtocolPreviewResult {
	return &ProtocolPreviewResult{
		Protocol:     protocol,
		ConnectionID: connectionID,
		Status:       "failed",
		Schema:       map[string]any{},
		Samples:      []any{},
		Diagnostics: map[string]any{
			"error": err.Error(),
		},
		DurationMS:    time.Since(startedAt).Milliseconds(),
		Truncated:     false,
		EffectiveTime: time.Now().UTC(),
	}
}

func (s *ProtocolPreviewService) recordPreview(ctx context.Context, connection *repository.ProtocolPreviewConnectionRecord, result *ProtocolPreviewResult, previewErr error) error {
	if connection == nil || result == nil {
		return nil
	}
	detail := map[string]any{
		"durationMs":  result.DurationMS,
		"sampleCount": len(result.Samples),
		"truncated":   result.Truncated,
	}
	if statusCode, ok := result.Diagnostics["statusCode"]; ok {
		detail["statusCode"] = statusCode
	}
	errorSummary := ""
	if previewErr != nil {
		errorSummary = previewErr.Error()
		detail["error"] = errorSummary
	} else if raw, ok := result.Diagnostics["error"]; ok {
		errorSummary = strings.TrimSpace(toString(raw))
	}
	return s.repository.CreateAccessSourceRecord(ctx, repository.CreateAccessSourceRecordParams{
		ProjectID:    connection.ProjectID,
		ConnectionID: connection.ID,
		RecordType:   "protocol.preview",
		Title:        "执行协议短时预览",
		Status:       result.Status,
		Protocol:     result.Protocol,
		DurationMS:   result.DurationMS,
		SampleCount:  len(result.Samples),
		Truncated:    result.Truncated,
		ErrorSummary: errorSummary,
		Detail:       detail,
	})
}
