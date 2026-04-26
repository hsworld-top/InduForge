package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/indu-forge/data_service/internal/auth"
	enginecompute "github.com/indu-forge/data_service/internal/engine/compute"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

const (
	defaultComputeTimeoutMS = 3000
	maxComputeTimeoutMS     = 120000
)

var allowedComputeLanguages = map[string]struct{}{
	"js":     {},
	"python": {},
}

var allowedComputeTriggerTypes = map[string]struct{}{
	"manual":           {},
	"timer":            {},
	"datapoint_change": {},
}

// ComputeUnit 表示返回给 HTTP 层的计算单元定义。
type ComputeUnit struct {
	ID            string         `json:"id"`
	ProjectID     string         `json:"projectId"`
	Name          string         `json:"name"`
	Language      string         `json:"language"`
	ScriptCode    string         `json:"scriptCode"`
	TriggerType   string         `json:"triggerType"`
	TriggerConfig map[string]any `json:"triggerConfig"`
	InputBindings map[string]any `json:"inputBindings"`
	OutputBinding map[string]any `json:"outputBindings"`
	TimeoutMS     int            `json:"timeoutMs"`
	IsEnabled     bool           `json:"isEnabled"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
}

// ComputeUnitPagination 表示计算单元分页信息。
type ComputeUnitPagination struct {
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

// ComputeUnitListResult 表示计算单元列表结果。
type ComputeUnitListResult struct {
	Units      []ComputeUnit         `json:"list"`
	Pagination ComputeUnitPagination `json:"pagination"`
}

// ComputeRun 表示一次计算运行记录。
type ComputeRun struct {
	ID            int64     `json:"id"`
	ProjectID     string    `json:"projectId"`
	ComputeUnitID string    `json:"computeUnitId"`
	TriggerMode   string    `json:"triggerMode"`
	Status        string    `json:"status"`
	DurationMS    int       `json:"durationMs"`
	Output        any       `json:"output"`
	ErrorMessage  *string   `json:"errorMessage,omitempty"`
	StartedAt     time.Time `json:"startedAt"`
	FinishedAt    time.Time `json:"finishedAt"`
	CreatedAt     time.Time `json:"createdAt"`
}

// ComputeRunListResult 表示计算运行记录列表结果。
type ComputeRunListResult struct {
	Runs       []ComputeRun          `json:"list"`
	Pagination ComputeUnitPagination `json:"pagination"`
}

// ComputeRunResult 表示一次计算执行结果。
type ComputeRunResult struct {
	Status       string    `json:"status"`
	DurationMS   int       `json:"durationMs"`
	Output       any       `json:"output"`
	SideEffects  []any     `json:"sideEffects,omitempty"`
	ErrorMessage string    `json:"errorMessage,omitempty"`
	StartedAt    time.Time `json:"startedAt"`
	FinishedAt   time.Time `json:"finishedAt"`
}

// CreateComputeUnitInput 描述创建计算单元的输入参数。
type CreateComputeUnitInput struct {
	Name          string
	Language      string
	ScriptCode    string
	TriggerType   string
	TriggerConfig map[string]any
	InputBindings map[string]any
	OutputBinding map[string]any
	TimeoutMS     *int
}

// UpdateComputeUnitInput 描述更新计算单元的输入参数。
type UpdateComputeUnitInput struct {
	Name             *string
	Language         *string
	ScriptCode       *string
	TriggerType      *string
	TriggerConfig    map[string]any
	HasTriggerConfig bool
	InputBindings    map[string]any
	HasInputBindings bool
	OutputBinding    map[string]any
	HasOutputBinding bool
	TimeoutMS        *int
	IsEnabled        *bool
}

// ComputeUnitListFilter 表示 service 层计算单元列表过滤条件。
type ComputeUnitListFilter struct {
	Language string
	Enabled  *bool
	Search   string
	Page     int
	PageSize int
}

// ComputeRunListFilter 表示 service 层计算运行记录过滤条件。
type ComputeRunListFilter struct {
	UnitID      string
	Status      string
	TriggerMode string
	Page        int
	PageSize    int
}

// RunComputeUnitInput 描述运行/调试计算单元时的输入参数。
type RunComputeUnitInput struct {
	Input map[string]any
}

// ComputeService 承载 compute 领域业务逻辑。
type ComputeService struct {
	repository   *repository.ComputeRepository
	datapoints   *repository.DataPointRepository
	queries      *QueryService
	mqtt         *repository.MqttRepository
	nodeRunner   enginecompute.Runner
	pythonRunner enginecompute.Runner
	scheduler    *enginecompute.Scheduler
}

// NewComputeService 创建 compute 服务。
func NewComputeService(
	repo *repository.ComputeRepository,
	nodeRunner enginecompute.Runner,
	pythonRunner enginecompute.Runner,
	scheduler *enginecompute.Scheduler,
	deps ...any,
) *ComputeService {
	var datapoints *repository.DataPointRepository
	var queries *QueryService
	var mqtt *repository.MqttRepository
	for _, dep := range deps {
		switch typed := dep.(type) {
		case *repository.DataPointRepository:
			datapoints = typed
		case *QueryService:
			queries = typed
		case *repository.MqttRepository:
			mqtt = typed
		}
	}
	return &ComputeService{
		repository:   repo,
		datapoints:   datapoints,
		queries:      queries,
		mqtt:         mqtt,
		nodeRunner:   nodeRunner,
		pythonRunner: pythonRunner,
		scheduler:    scheduler,
	}
}

// ListComputeUnits 查询项目下的计算单元列表。
func (s *ComputeService) ListComputeUnits(ctx context.Context, claims *auth.Claims, projectID string, filter ComputeUnitListFilter) (*ComputeUnitListResult, error) {
	if err := s.validateReadAccess(claims, projectID); err != nil {
		return nil, err
	}
	language := strings.TrimSpace(strings.ToLower(filter.Language))
	if language != "" {
		if _, ok := allowedComputeLanguages[language]; !ok {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "language 仅支持 js/python")
		}
	}

	records, total, err := s.repository.ListUnits(ctx, projectID, repository.ComputeUnitListFilter{
		Language: language,
		Enabled:  filter.Enabled,
		Search:   strings.TrimSpace(filter.Search),
		Page:     filter.Page,
		PageSize: filter.PageSize,
	})
	if err != nil {
		return nil, err
	}
	units := make([]ComputeUnit, 0, len(records))
	for _, record := range records {
		units = append(units, toComputeUnit(record))
	}
	page, pageSize := normalizePageAndSize(filter.Page, filter.PageSize, 20, 100)
	return &ComputeUnitListResult{
		Units: units,
		Pagination: ComputeUnitPagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages(total, pageSize),
		},
	}, nil
}

// GetComputeUnit 返回单个计算单元详情。
func (s *ComputeService) GetComputeUnit(ctx context.Context, claims *auth.Claims, projectID, unitID string) (*ComputeUnit, error) {
	if err := s.validateReadAccess(claims, projectID); err != nil {
		return nil, err
	}
	if err := validateComputeUnitID(unitID); err != nil {
		return nil, err
	}
	record, err := s.repository.GetUnitByProjectAndID(ctx, projectID, unitID)
	if err != nil {
		return nil, err
	}
	unit := toComputeUnit(*record)
	return &unit, nil
}

// CreateComputeUnit 创建计算单元定义。
func (s *ComputeService) CreateComputeUnit(ctx context.Context, claims *auth.Claims, projectID string, input CreateComputeUnitInput) (*ComputeUnit, error) {
	if err := s.validateWriteAccess(claims, projectID); err != nil {
		return nil, err
	}
	normalized, err := normalizeCreateComputeInput(input)
	if err != nil {
		return nil, err
	}

	record, err := s.repository.CreateUnit(ctx, repository.CreateComputeUnitParams{
		ProjectID:     projectID,
		UserID:        claims.UserID,
		Name:          normalized.Name,
		Language:      normalized.Language,
		ScriptCode:    normalized.ScriptCode,
		TriggerType:   normalized.TriggerType,
		TriggerConfig: normalized.TriggerConfig,
		InputBindings: normalized.InputBindings,
		OutputBinding: normalized.OutputBinding,
		TimeoutMS:     normalized.TimeoutMS,
	})
	if err != nil {
		return nil, err
	}
	if err := s.syncComputeOutputDataPoints(ctx, *record, claims.UserID); err != nil {
		return nil, err
	}

	unit := toComputeUnit(*record)
	return &unit, nil
}

// UpdateComputeUnit 更新计算单元定义。
func (s *ComputeService) UpdateComputeUnit(ctx context.Context, claims *auth.Claims, projectID, unitID string, input UpdateComputeUnitInput) (*ComputeUnit, error) {
	if err := s.validateWriteAccess(claims, projectID); err != nil {
		return nil, err
	}
	if err := validateComputeUnitID(unitID); err != nil {
		return nil, err
	}
	current, err := s.repository.GetUnitByProjectAndID(ctx, projectID, unitID)
	if err != nil {
		return nil, err
	}
	normalized, err := mergeComputeUpdateInput(*current, input)
	if err != nil {
		return nil, err
	}

	record, err := s.repository.UpdateUnit(ctx, repository.UpdateComputeUnitParams{
		ID:            current.ID,
		ProjectID:     projectID,
		UserID:        claims.UserID,
		Name:          normalized.Name,
		Language:      normalized.Language,
		ScriptCode:    normalized.ScriptCode,
		TriggerType:   normalized.TriggerType,
		TriggerConfig: normalized.TriggerConfig,
		InputBindings: normalized.InputBindings,
		OutputBinding: normalized.OutputBinding,
		TimeoutMS:     normalized.TimeoutMS,
		IsEnabled:     normalized.IsEnabled,
	})
	if err != nil {
		return nil, err
	}
	if err := s.syncComputeOutputDataPoints(ctx, *record, claims.UserID); err != nil {
		return nil, err
	}
	unit := toComputeUnit(*record)
	return &unit, nil
}

// DeleteComputeUnit 删除计算单元并将输出数据点标记失效。
func (s *ComputeService) DeleteComputeUnit(ctx context.Context, claims *auth.Claims, projectID, unitID string) error {
	if err := s.validateWriteAccess(claims, projectID); err != nil {
		return err
	}
	if err := validateComputeUnitID(unitID); err != nil {
		return err
	}
	if err := s.repository.DeleteUnit(ctx, projectID, unitID); err != nil {
		return err
	}
	if s.datapoints != nil {
		_, _ = s.datapoints.MarkInvalidBySource(ctx, projectID, "calc.output", unitID, &claims.UserID)
	}
	return nil
}

// ToggleComputeUnit 切换计算单元启用状态。
func (s *ComputeService) ToggleComputeUnit(ctx context.Context, claims *auth.Claims, projectID, unitID string, enabled bool) (*ComputeUnit, error) {
	if err := s.validateWriteAccess(claims, projectID); err != nil {
		return nil, err
	}
	if err := validateComputeUnitID(unitID); err != nil {
		return nil, err
	}
	record, err := s.repository.UpdateUnitEnabled(ctx, projectID, unitID, claims.UserID, enabled)
	if err != nil {
		return nil, err
	}
	unit := toComputeUnit(*record)
	return &unit, nil
}

// ListComputeRuns 查询项目下计算运行记录。
func (s *ComputeService) ListComputeRuns(ctx context.Context, claims *auth.Claims, projectID string, filter ComputeRunListFilter) (*ComputeRunListResult, error) {
	if err := s.validateReadAccess(claims, projectID); err != nil {
		return nil, err
	}
	status := strings.TrimSpace(strings.ToLower(filter.Status))
	if status != "" && status != "success" && status != "timeout" && status != "failed" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "status 不受支持")
	}
	triggerMode := strings.TrimSpace(strings.ToLower(filter.TriggerMode))
	if triggerMode != "" && triggerMode != "run" && triggerMode != "debug" && triggerMode != "schedule" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "triggerMode 不受支持")
	}
	if strings.TrimSpace(filter.UnitID) != "" {
		if err := validateComputeUnitID(filter.UnitID); err != nil {
			return nil, err
		}
	}

	records, total, err := s.repository.ListRuns(ctx, projectID, repository.ComputeRunListFilter{
		UnitID:      strings.TrimSpace(filter.UnitID),
		Status:      status,
		TriggerMode: triggerMode,
		Page:        filter.Page,
		PageSize:    filter.PageSize,
	})
	if err != nil {
		return nil, err
	}
	runs := make([]ComputeRun, 0, len(records))
	for _, record := range records {
		runs = append(runs, toComputeRun(record))
	}
	page, pageSize := normalizePageAndSize(filter.Page, filter.PageSize, 20, 100)
	return &ComputeRunListResult{
		Runs: runs,
		Pagination: ComputeUnitPagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages(total, pageSize),
		},
	}, nil
}

// RunComputeUnit 触发一次正式运行。
func (s *ComputeService) RunComputeUnit(ctx context.Context, claims *auth.Claims, projectID, unitID string, input RunComputeUnitInput) (*ComputeRunResult, error) {
	return s.executeComputeUnit(ctx, claims, projectID, unitID, "run", input)
}

// DebugComputeUnit 触发一次调试运行。
func (s *ComputeService) DebugComputeUnit(ctx context.Context, claims *auth.Claims, projectID, unitID string, input RunComputeUnitInput) (*ComputeRunResult, error) {
	return s.executeComputeUnit(ctx, claims, projectID, unitID, "debug", input)
}

func (s *ComputeService) executeComputeUnit(ctx context.Context, claims *auth.Claims, projectID, unitID, triggerMode string, input RunComputeUnitInput) (*ComputeRunResult, error) {
	if err := s.validateWriteAccess(claims, projectID); err != nil {
		return nil, err
	}
	if err := validateComputeUnitID(unitID); err != nil {
		return nil, err
	}

	unit, err := s.repository.GetUnitByProjectAndID(ctx, projectID, unitID)
	if err != nil {
		return nil, err
	}
	if !unit.IsEnabled {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "计算单元已禁用")
	}

	runner, err := s.pickRunner(unit.Language)
	if err != nil {
		return nil, err
	}
	sdkContext, err := s.prepareComputeSDKContext(ctx, *unit, input.Input)
	if err != nil {
		return nil, err
	}

	callback, err := s.startComputeSDKCallback(*unit)
	if err != nil {
		return nil, err
	}
	if callback != nil {
		defer callback.Close()
	}

	startedAt := time.Now().UTC()
	request := enginecompute.ExecuteRequest{
		Script:     unit.ScriptCode,
		Input:      cloneMap(input.Input),
		SDKContext: sdkContext,
		Timeout:    time.Duration(unit.TimeoutMS) * time.Millisecond,
	}
	if callback != nil {
		request.CallbackURL = callback.URL
		request.CallbackToken = callback.Token
	}
	executeResult, executeErr := runner.Run(ctx, request)
	finishedAt := time.Now().UTC()

	status := "success"
	var errorMessage *string
	if executeErr != nil {
		if errors.Is(executeErr, enginecompute.ErrTimeout) {
			status = "timeout"
			errorMessage = stringPointer("计算执行超时")
		} else {
			status = "failed"
			errorMessage = stringPointer(executeErr.Error())
		}
	}

	saveErr := s.repository.SaveRun(ctx, repository.SaveComputeRunParams{
		ProjectID:     projectID,
		ComputeUnitID: unit.ID,
		TriggerMode:   triggerMode,
		Status:        status,
		DurationMS:    int(executeResult.Duration.Milliseconds()),
		Output:        computeRunOutputPayload(executeResult),
		ErrorMessage:  errorMessage,
		StartedAt:     startedAt,
		FinishedAt:    finishedAt,
	})
	if saveErr != nil {
		return nil, saveErr
	}

	if executeErr != nil {
		if errors.Is(executeErr, enginecompute.ErrTimeout) {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "计算执行超时", executeErr)
		}
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "计算脚本执行失败", executeErr)
	}
	if status == "success" {
		_ = s.writeComputeOutputValues(ctx, *unit, executeResult.Output, claims.UserID)
	}

	return &ComputeRunResult{
		Status:      status,
		DurationMS:  int(executeResult.Duration.Milliseconds()),
		Output:      executeResult.Output,
		SideEffects: cloneJSONArray(executeResult.SideEffects),
		StartedAt:   startedAt,
		FinishedAt:  finishedAt,
	}, nil
}

func (s *ComputeService) validateDependencies() error {
	if s == nil || s.repository == nil || s.nodeRunner == nil || s.pythonRunner == nil || s.scheduler == nil {
		return apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "compute 服务依赖未初始化")
	}
	return nil
}

func computeRunOutputPayload(result enginecompute.ExecuteResult) any {
	if len(result.SideEffects) == 0 {
		return result.Output
	}
	return map[string]any{
		"result":      result.Output,
		"sideEffects": cloneJSONArray(result.SideEffects),
	}
}

func (s *ComputeService) validateReadAccess(claims *auth.Claims, projectID string) error {
	if err := s.validateDependencies(); err != nil {
		return err
	}
	if claims == nil {
		return apperrors.NewAppError(apperrors.ErrorCodeAuthTokenRequired, http.StatusUnauthorized, "请先完成认证")
	}
	if err := validateProjectID(projectID); err != nil {
		return err
	}
	if !claims.HasProjectAccess(projectID) {
		return apperrors.NewAppError(apperrors.ErrorCodePermissionProjectMismatch, http.StatusForbidden, "项目范围不足")
	}
	return nil
}

func (s *ComputeService) validateWriteAccess(claims *auth.Claims, projectID string) error {
	if err := s.validateReadAccess(claims, projectID); err != nil {
		return err
	}
	return validateUserID(claims.UserID)
}

func (s *ComputeService) pickRunner(language string) (enginecompute.Runner, error) {
	switch strings.TrimSpace(strings.ToLower(language)) {
	case "js":
		return s.nodeRunner, nil
	case "python":
		return s.pythonRunner, nil
	default:
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "language 不受支持")
	}
}

func toComputeUnit(record repository.ComputeUnitRecord) ComputeUnit {
	return ComputeUnit{
		ID:            record.ID,
		ProjectID:     record.ProjectID,
		Name:          record.Name,
		Language:      record.Language,
		ScriptCode:    record.ScriptCode,
		TriggerType:   record.TriggerType,
		TriggerConfig: cloneMap(record.TriggerConfig),
		InputBindings: cloneMap(record.InputBindings),
		OutputBinding: cloneMap(record.OutputBinding),
		TimeoutMS:     record.TimeoutMS,
		IsEnabled:     record.IsEnabled,
		CreatedAt:     record.CreatedAt,
		UpdatedAt:     record.UpdatedAt,
	}
}

func toComputeRun(record repository.ComputeRunRecord) ComputeRun {
	return ComputeRun{
		ID:            record.ID,
		ProjectID:     record.ProjectID,
		ComputeUnitID: record.ComputeUnitID,
		TriggerMode:   record.TriggerMode,
		Status:        record.Status,
		DurationMS:    record.DurationMS,
		Output:        record.Output,
		ErrorMessage:  cloneOptionalString(record.ErrorMessage),
		StartedAt:     record.StartedAt,
		FinishedAt:    record.FinishedAt,
		CreatedAt:     record.CreatedAt,
	}
}

type normalizedComputeUnitInput struct {
	Name          string
	Language      string
	ScriptCode    string
	TriggerType   string
	TriggerConfig map[string]any
	InputBindings map[string]any
	OutputBinding map[string]any
	TimeoutMS     int
	IsEnabled     bool
}

func normalizeCreateComputeInput(input CreateComputeUnitInput) (normalizedComputeUnitInput, error) {
	name, err := normalizeComputeName(input.Name)
	if err != nil {
		return normalizedComputeUnitInput{}, err
	}
	language, err := normalizeComputeLanguage(input.Language)
	if err != nil {
		return normalizedComputeUnitInput{}, err
	}
	scriptCode, err := normalizeComputeScript(input.ScriptCode)
	if err != nil {
		return normalizedComputeUnitInput{}, err
	}
	triggerType, err := normalizeComputeTriggerType(input.TriggerType)
	if err != nil {
		return normalizedComputeUnitInput{}, err
	}
	timeoutMS, err := normalizeComputeTimeout(input.TimeoutMS, defaultComputeTimeoutMS)
	if err != nil {
		return normalizedComputeUnitInput{}, err
	}
	return normalizedComputeUnitInput{
		Name:          name,
		Language:      language,
		ScriptCode:    scriptCode,
		TriggerType:   triggerType,
		TriggerConfig: cloneMap(input.TriggerConfig),
		InputBindings: cloneMap(input.InputBindings),
		OutputBinding: cloneMap(input.OutputBinding),
		TimeoutMS:     timeoutMS,
		IsEnabled:     true,
	}, nil
}

func mergeComputeUpdateInput(current repository.ComputeUnitRecord, input UpdateComputeUnitInput) (normalizedComputeUnitInput, error) {
	if input.Name == nil && input.Language == nil && input.ScriptCode == nil && input.TriggerType == nil &&
		!input.HasTriggerConfig && !input.HasInputBindings && !input.HasOutputBinding && input.TimeoutMS == nil && input.IsEnabled == nil {
		return normalizedComputeUnitInput{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "至少需要提供一个待更新字段")
	}

	result := normalizedComputeUnitInput{
		Name:          current.Name,
		Language:      current.Language,
		ScriptCode:    current.ScriptCode,
		TriggerType:   current.TriggerType,
		TriggerConfig: cloneMap(current.TriggerConfig),
		InputBindings: cloneMap(current.InputBindings),
		OutputBinding: cloneMap(current.OutputBinding),
		TimeoutMS:     current.TimeoutMS,
		IsEnabled:     current.IsEnabled,
	}
	var err error
	if input.Name != nil {
		result.Name, err = normalizeComputeName(*input.Name)
		if err != nil {
			return normalizedComputeUnitInput{}, err
		}
	}
	if input.Language != nil {
		result.Language, err = normalizeComputeLanguage(*input.Language)
		if err != nil {
			return normalizedComputeUnitInput{}, err
		}
	}
	if input.ScriptCode != nil {
		result.ScriptCode, err = normalizeComputeScript(*input.ScriptCode)
		if err != nil {
			return normalizedComputeUnitInput{}, err
		}
	}
	if input.TriggerType != nil {
		result.TriggerType, err = normalizeComputeTriggerType(*input.TriggerType)
		if err != nil {
			return normalizedComputeUnitInput{}, err
		}
	}
	if input.HasTriggerConfig {
		result.TriggerConfig = cloneMap(input.TriggerConfig)
	}
	if input.HasInputBindings {
		result.InputBindings = cloneMap(input.InputBindings)
	}
	if input.HasOutputBinding {
		result.OutputBinding = cloneMap(input.OutputBinding)
	}
	if input.TimeoutMS != nil {
		result.TimeoutMS, err = normalizeComputeTimeout(input.TimeoutMS, current.TimeoutMS)
		if err != nil {
			return normalizedComputeUnitInput{}, err
		}
	}
	if input.IsEnabled != nil {
		result.IsEnabled = *input.IsEnabled
	}
	return result, nil
}

func normalizeComputeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "name 不能为空")
	}
	if len([]rune(name)) > 100 {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "name 长度不能超过 100")
	}
	return name, nil
}

func normalizeComputeLanguage(language string) (string, error) {
	language = strings.TrimSpace(strings.ToLower(language))
	if _, ok := allowedComputeLanguages[language]; !ok {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "language 仅支持 js/python")
	}
	return language, nil
}

func normalizeComputeScript(scriptCode string) (string, error) {
	scriptCode = strings.TrimSpace(scriptCode)
	if scriptCode == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "scriptCode 不能为空")
	}
	return scriptCode, nil
}

func normalizeComputeTriggerType(triggerType string) (string, error) {
	triggerType = strings.TrimSpace(strings.ToLower(triggerType))
	if triggerType == "" {
		triggerType = "manual"
	}
	if _, ok := allowedComputeTriggerTypes[triggerType]; !ok {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "triggerType 不受支持")
	}
	return triggerType, nil
}

func normalizeComputeTimeout(timeout *int, defaultValue int) (int, error) {
	timeoutMS := defaultValue
	if timeout != nil {
		timeoutMS = *timeout
	}
	if timeoutMS <= 0 || timeoutMS > maxComputeTimeoutMS {
		return 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "timeoutMs 超出允许范围")
	}
	return timeoutMS, nil
}

func validateComputeUnitID(unitID string) error {
	if _, err := uuid.Parse(strings.TrimSpace(unitID)); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "id 格式无效", err)
	}
	return nil
}

type computeOutputBinding struct {
	Name        string
	Path        string
	DataType    string
	Unit        *string
	Description *string
}

func (s *ComputeService) syncComputeOutputDataPoints(ctx context.Context, unit repository.ComputeUnitRecord, userID string) error {
	if s == nil || s.datapoints == nil {
		return nil
	}
	_, _ = s.datapoints.MarkInvalidBySource(ctx, unit.ProjectID, "calc.output", unit.ID, &userID)
	outputs := extractComputeOutputBindings(unit)
	for _, output := range outputs {
		path := output.Path
		if existing, err := s.datapoints.GetByProjectAndPath(ctx, unit.ProjectID, path); err == nil && existing != nil &&
			(existing.SourceID == nil || *existing.SourceID != unit.ID || existing.SourceType != "calc.output") {
			path = path + "_" + unit.ID[:8]
		}
		_, err := s.datapoints.UpsertByPath(ctx, repository.CreateDataPointParams{
			ProjectID:    unit.ProjectID,
			UserID:       &userID,
			Path:         path,
			Name:         output.Name,
			Description:  cloneOptionalString(output.Description),
			SourceType:   "calc.output",
			SourceID:     &unit.ID,
			SourceConfig: map[string]any{"computeUnitId": unit.ID, "outputName": output.Name},
			DataType:     output.DataType,
			Unit:         cloneOptionalString(output.Unit),
			Tags:         []any{},
			RefreshMode:  "manual",
			Status:       "active",
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *ComputeService) writeComputeOutputValues(ctx context.Context, unit repository.ComputeUnitRecord, output any, userID string) error {
	if s == nil || s.datapoints == nil {
		return nil
	}
	outputs := extractComputeOutputBindings(unit)
	if len(outputs) == 0 {
		return nil
	}
	outputMap, _ := output.(map[string]any)
	for _, binding := range outputs {
		value := output
		if outputMap != nil {
			if namedValue, ok := outputMap[binding.Name]; ok {
				value = namedValue
			}
		}
		defaultValue := stringifyDataPointValue(value)
		path := binding.Path
		if existing, err := s.datapoints.GetByProjectAndPath(ctx, unit.ProjectID, path); err == nil && existing != nil {
			_, _ = s.datapoints.UpsertByPath(ctx, repository.CreateDataPointParams{
				ProjectID:    unit.ProjectID,
				UserID:       &userID,
				Path:         existing.Path,
				Name:         existing.Name,
				Description:  cloneOptionalString(existing.Description),
				SourceType:   existing.SourceType,
				SourceID:     cloneOptionalString(existing.SourceID),
				SourceConfig: cloneMap(existing.SourceConfig),
				DataType:     existing.DataType,
				Unit:         cloneOptionalString(existing.Unit),
				PrecisionNum: cloneOptionalInt(existing.PrecisionNum),
				DefaultValue: &defaultValue,
				MinValue:     cloneOptionalFloat64(existing.MinValue),
				MaxValue:     cloneOptionalFloat64(existing.MaxValue),
				AlarmLow:     cloneOptionalFloat64(existing.AlarmLow),
				AlarmHigh:    cloneOptionalFloat64(existing.AlarmHigh),
				Tags:         cloneJSONArray(existing.Tags),
				RefreshMode:  existing.RefreshMode,
				Status:       existing.Status,
			})
		}
	}
	return nil
}

func extractComputeOutputBindings(unit repository.ComputeUnitRecord) []computeOutputBinding {
	outputs := make([]computeOutputBinding, 0)
	rawOutputs, ok := unit.OutputBinding["outputs"]
	if ok {
		if list, ok := rawOutputs.([]any); ok {
			for _, item := range list {
				if mapped, ok := item.(map[string]any); ok {
					if binding := outputBindingFromMap(unit, "", mapped); binding.Name != "" {
						outputs = append(outputs, binding)
					}
				}
			}
			return outputs
		}
	}

	for key, raw := range unit.OutputBinding {
		key = strings.TrimSpace(key)
		if key == "" || key == "mode" || key == "schema" || key == "description" {
			continue
		}
		switch typed := raw.(type) {
		case string:
			outputs = append(outputs, computeOutputBinding{
				Name:     key,
				Path:     normalizeComputeOutputPath(unit, key, typed),
				DataType: "object",
			})
		case map[string]any:
			if binding := outputBindingFromMap(unit, key, typed); binding.Name != "" {
				outputs = append(outputs, binding)
			}
		}
	}
	return outputs
}

func outputBindingFromMap(unit repository.ComputeUnitRecord, fallbackName string, input map[string]any) computeOutputBinding {
	name := strings.TrimSpace(firstString(input, "name", "key", "outputName"))
	if name == "" {
		name = strings.TrimSpace(fallbackName)
	}
	if name == "" {
		return computeOutputBinding{}
	}
	path := normalizeComputeOutputPath(unit, name, firstString(input, "path"))
	dataType := strings.TrimSpace(firstString(input, "dataType", "type"))
	if dataType == "" {
		dataType = "object"
	}
	var unitText *string
	if rawUnit := strings.TrimSpace(firstString(input, "unit")); rawUnit != "" {
		unitText = &rawUnit
	}
	var description *string
	if rawDescription := strings.TrimSpace(firstString(input, "description")); rawDescription != "" {
		description = &rawDescription
	}
	return computeOutputBinding{Name: name, Path: path, DataType: dataType, Unit: unitText, Description: description}
}

func normalizeComputeOutputPath(unit repository.ComputeUnitRecord, outputName, configuredPath string) string {
	path := strings.TrimSpace(configuredPath)
	if path != "" {
		return path
	}
	return "calc." + normalizeDatapointSegment(unit.Name) + "." + normalizeDatapointSegment(outputName)
}

func firstString(input map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := input[key]; ok && value != nil {
			return strings.TrimSpace(fmt.Sprintf("%v", value))
		}
	}
	return ""
}

func totalPages(total, pageSize int) int {
	if total <= 0 || pageSize <= 0 {
		return 0
	}
	return (total + pageSize - 1) / pageSize
}

func stringPointer(value string) *string {
	v := strings.TrimSpace(value)
	return &v
}
