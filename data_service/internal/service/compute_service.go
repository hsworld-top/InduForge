package service

import (
	"context"
	"errors"
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

// ComputeRunResult 表示一次计算执行结果。
type ComputeRunResult struct {
	Status       string    `json:"status"`
	DurationMS   int       `json:"durationMs"`
	Output       any       `json:"output"`
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

// RunComputeUnitInput 描述运行/调试计算单元时的输入参数。
type RunComputeUnitInput struct {
	Input map[string]any
}

// ComputeService 承载 compute 领域业务逻辑。
type ComputeService struct {
	repository   *repository.ComputeRepository
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
) *ComputeService {
	return &ComputeService{
		repository:   repo,
		nodeRunner:   nodeRunner,
		pythonRunner: pythonRunner,
		scheduler:    scheduler,
	}
}

// CreateComputeUnit 创建计算单元定义。
func (s *ComputeService) CreateComputeUnit(ctx context.Context, claims *auth.Claims, projectID string, input CreateComputeUnitInput) (*ComputeUnit, error) {
	if err := s.validateDependencies(); err != nil {
		return nil, err
	}
	if claims == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeAuthTokenRequired, http.StatusUnauthorized, "请先完成认证")
	}
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateUserID(claims.UserID); err != nil {
		return nil, err
	}
	if !claims.HasProjectAccess(projectID) {
		return nil, apperrors.NewAppError(apperrors.ErrorCodePermissionProjectMismatch, http.StatusForbidden, "项目范围不足")
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "name 不能为空")
	}
	if len([]rune(name)) > 100 {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "name 长度不能超过 100")
	}

	language := strings.TrimSpace(strings.ToLower(input.Language))
	if _, ok := allowedComputeLanguages[language]; !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "language 仅支持 js/python")
	}

	scriptCode := strings.TrimSpace(input.ScriptCode)
	if scriptCode == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "scriptCode 不能为空")
	}

	triggerType := strings.TrimSpace(strings.ToLower(input.TriggerType))
	if triggerType == "" {
		triggerType = "manual"
	}
	if _, ok := allowedComputeTriggerTypes[triggerType]; !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "triggerType 不受支持")
	}

	timeoutMS := defaultComputeTimeoutMS
	if input.TimeoutMS != nil {
		timeoutMS = *input.TimeoutMS
	}
	if timeoutMS <= 0 || timeoutMS > maxComputeTimeoutMS {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "timeoutMs 超出允许范围")
	}

	record, err := s.repository.CreateUnit(ctx, repository.CreateComputeUnitParams{
		ProjectID:     projectID,
		UserID:        claims.UserID,
		Name:          name,
		Language:      language,
		ScriptCode:    scriptCode,
		TriggerType:   triggerType,
		TriggerConfig: cloneMap(input.TriggerConfig),
		InputBindings: cloneMap(input.InputBindings),
		OutputBinding: cloneMap(input.OutputBinding),
		TimeoutMS:     timeoutMS,
	})
	if err != nil {
		return nil, err
	}

	unit := toComputeUnit(*record)
	return &unit, nil
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
	if err := s.validateDependencies(); err != nil {
		return nil, err
	}
	if claims == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeAuthTokenRequired, http.StatusUnauthorized, "请先完成认证")
	}
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if _, err := uuid.Parse(strings.TrimSpace(unitID)); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "id 格式无效", err)
	}
	if !claims.HasProjectAccess(projectID) {
		return nil, apperrors.NewAppError(apperrors.ErrorCodePermissionProjectMismatch, http.StatusForbidden, "项目范围不足")
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

	startedAt := time.Now().UTC()
	executeResult, executeErr := runner.Run(ctx, enginecompute.ExecuteRequest{
		Script:  unit.ScriptCode,
		Input:   cloneMap(input.Input),
		Timeout: time.Duration(unit.TimeoutMS) * time.Millisecond,
	})
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
		Output:        executeResult.Output,
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

	return &ComputeRunResult{
		Status:     status,
		DurationMS: int(executeResult.Duration.Milliseconds()),
		Output:     executeResult.Output,
		StartedAt:  startedAt,
		FinishedAt: finishedAt,
	}, nil
}

func (s *ComputeService) validateDependencies() error {
	if s == nil || s.repository == nil || s.nodeRunner == nil || s.pythonRunner == nil || s.scheduler == nil {
		return apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "compute 服务依赖未初始化")
	}
	return nil
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

func stringPointer(value string) *string {
	v := strings.TrimSpace(value)
	return &v
}
