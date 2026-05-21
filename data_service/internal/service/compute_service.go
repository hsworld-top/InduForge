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
	defaultComputeTimeoutMS   = 3000
	maxComputeTimeoutMS       = 120000
	defaultSyntaxCheckTimeout = 3 * time.Second
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
	Description   *string        `json:"description,omitempty"`
	FolderID      *string        `json:"folderId,omitempty"`
	Language      string         `json:"language"`
	Lang          string         `json:"lang,omitempty"`
	ScriptCode    string         `json:"scriptCode"`
	Code          string         `json:"code,omitempty"`
	TriggerType   string         `json:"triggerType"`
	TriggerConfig map[string]any `json:"triggerConfig"`
	InputBindings map[string]any `json:"inputBindings"`
	OutputBinding map[string]any `json:"outputBindings"`
	Dependencies  []any          `json:"dependencies"`
	TimeoutMS     int            `json:"timeoutMs"`
	IsEnabled     bool           `json:"isEnabled"`
	Status        string         `json:"status"`
	Path          string         `json:"path"`
	OutputPath    string         `json:"outputPath"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
}

// ComputeFolder 表示计算单元文件夹。
type ComputeFolder struct {
	ID        string           `json:"id"`
	ProjectID string           `json:"projectId"`
	Name      string           `json:"name"`
	ParentID  *string          `json:"parentId,omitempty"`
	Children  []*ComputeFolder `json:"children,omitempty"`
	CreatedAt time.Time        `json:"createdAt"`
	UpdatedAt time.Time        `json:"updatedAt"`
}

// ComputeDependency 表示计算单元可用依赖。
type ComputeDependency struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Runtime     string `json:"runtime"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Status      string `json:"status"`
	ImportName  string `json:"importName"`
}

// ComputeDependencyListResult 表示依赖清单响应。
type ComputeDependencyListResult struct {
	List []ComputeDependency `json:"list"`
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
	DryRun       bool      `json:"dryRun"`
	Output       any       `json:"output"`
	SideEffects  []any     `json:"sideEffects,omitempty"`
	Logs         []string  `json:"logs,omitempty"`
	ErrorMessage string    `json:"errorMessage,omitempty"`
	StartedAt    time.Time `json:"startedAt"`
	FinishedAt   time.Time `json:"finishedAt"`
}

// ComputeSyntaxDiagnostic 表示计算脚本语法诊断。
type ComputeSyntaxDiagnostic struct {
	Severity  string `json:"severity"`
	Message   string `json:"message"`
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	EndLine   int    `json:"endLine"`
	EndColumn int    `json:"endColumn"`
	Source    string `json:"source"`
}

// ComputeSyntaxCheckResult 表示计算脚本语法检查结果。
type ComputeSyntaxCheckResult struct {
	Diagnostics []ComputeSyntaxDiagnostic `json:"diagnostics"`
}

// CreateComputeUnitInput 描述创建计算单元的输入参数。
type CreateComputeUnitInput struct {
	Name          string
	Description   *string
	FolderID      *string
	Language      string
	ScriptCode    string
	TriggerType   string
	TriggerConfig map[string]any
	InputBindings map[string]any
	OutputBinding map[string]any
	Dependencies  []any
	TimeoutMS     *int
}

// CreateComputeFolderInput 描述创建计算文件夹的输入参数。
type CreateComputeFolderInput struct {
	Name     string
	ParentID *string
}

// UpdateComputeFolderInput 描述更新计算文件夹的输入参数。
type UpdateComputeFolderInput struct {
	Name        *string
	ParentID    *string
	HasParentID bool
}

// UpdateComputeUnitInput 描述更新计算单元的输入参数。
type UpdateComputeUnitInput struct {
	Name             *string
	Description      *string
	HasDescription   bool
	FolderID         *string
	HasFolderID      bool
	Language         *string
	ScriptCode       *string
	TriggerType      *string
	TriggerConfig    map[string]any
	HasTriggerConfig bool
	InputBindings    map[string]any
	HasInputBindings bool
	OutputBinding    map[string]any
	HasOutputBinding bool
	Dependencies     []any
	HasDependencies  bool
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
	Input  map[string]any
	DryRun bool
}

// ComputeSyntaxCheckInput 描述语法检查输入。
type ComputeSyntaxCheckInput struct {
	Language   string
	ScriptCode string
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

// ListComputeDependencies 返回计算运行时内置依赖清单。
func (s *ComputeService) ListComputeDependencies(ctx context.Context, claims *auth.Claims, projectID string) (*ComputeDependencyListResult, error) {
	if err := s.validateReadAccess(claims, projectID); err != nil {
		return nil, err
	}
	return &ComputeDependencyListResult{List: builtInComputeDependencies()}, nil
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
	if err := s.ensureComputeFolderInProject(ctx, projectID, normalized.FolderID); err != nil {
		return nil, err
	}

	record, err := s.repository.CreateUnit(ctx, repository.CreateComputeUnitParams{
		ProjectID:     projectID,
		UserID:        claims.UserID,
		Name:          normalized.Name,
		Description:   normalized.Description,
		FolderID:      normalized.FolderID,
		Language:      normalized.Language,
		ScriptCode:    normalized.ScriptCode,
		TriggerType:   normalized.TriggerType,
		TriggerConfig: normalized.TriggerConfig,
		InputBindings: normalized.InputBindings,
		OutputBinding: normalized.OutputBinding,
		Dependencies:  normalized.Dependencies,
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

// ListComputeFolders 查询项目下计算文件夹。
func (s *ComputeService) ListComputeFolders(ctx context.Context, claims *auth.Claims, projectID string) ([]*ComputeFolder, error) {
	if err := s.validateReadAccess(claims, projectID); err != nil {
		return nil, err
	}
	records, err := s.repository.ListFolders(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return toComputeFolderTree(records), nil
}

// CreateComputeFolder 创建计算文件夹。
func (s *ComputeService) CreateComputeFolder(ctx context.Context, claims *auth.Claims, projectID string, input CreateComputeFolderInput) (*ComputeFolder, error) {
	if err := s.validateWriteAccess(claims, projectID); err != nil {
		return nil, err
	}
	name, err := normalizeComputeFolderName(input.Name)
	if err != nil {
		return nil, err
	}
	parentID, err := normalizeOptionalComputeFolderID(input.ParentID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureComputeFolderInProject(ctx, projectID, parentID); err != nil {
		return nil, err
	}
	record, err := s.repository.CreateFolder(ctx, repository.CreateComputeFolderParams{
		ProjectID: projectID,
		UserID:    claims.UserID,
		Name:      name,
		ParentID:  parentID,
	})
	if err != nil {
		return nil, err
	}
	folder := toComputeFolder(*record)
	return &folder, nil
}

// UpdateComputeFolder 更新计算文件夹。
func (s *ComputeService) UpdateComputeFolder(ctx context.Context, claims *auth.Claims, projectID, folderID string, input UpdateComputeFolderInput) (*ComputeFolder, error) {
	if err := s.validateWriteAccess(claims, projectID); err != nil {
		return nil, err
	}
	if err := validateComputeFolderID(folderID); err != nil {
		return nil, err
	}
	records, err := s.repository.ListFolders(ctx, projectID)
	if err != nil {
		return nil, err
	}
	current, ok := findComputeFolderRecord(records, folderID)
	if !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "计算文件夹不存在")
	}
	if input.Name == nil && !input.HasParentID {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "至少需要提供一个待更新字段")
	}

	name := current.Name
	if input.Name != nil {
		name, err = normalizeComputeFolderName(*input.Name)
		if err != nil {
			return nil, err
		}
	}
	parentID := cloneOptionalString(current.ParentID)
	if input.HasParentID {
		parentID, err = normalizeOptionalComputeFolderID(input.ParentID)
		if err != nil {
			return nil, err
		}
		if parentID != nil {
			if *parentID == folderID {
				return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "上级文件夹不能指向自身")
			}
			if _, ok := findComputeFolderRecord(records, *parentID); !ok {
				return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "上级计算文件夹不存在")
			}
			if isDescendantComputeFolder(records, folderID, *parentID) {
				return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "上级文件夹不能指向下级文件夹")
			}
		}
	}

	record, err := s.repository.UpdateFolder(ctx, projectID, folderID, claims.UserID, name, parentID)
	if err != nil {
		return nil, err
	}
	folder := toComputeFolder(*record)
	return &folder, nil
}

// DeleteComputeFolder 删除计算文件夹。
func (s *ComputeService) DeleteComputeFolder(ctx context.Context, claims *auth.Claims, projectID, folderID string) error {
	if err := s.validateWriteAccess(claims, projectID); err != nil {
		return err
	}
	if err := validateComputeFolderID(folderID); err != nil {
		return err
	}
	unitIDs, err := s.repository.ListUnitIDsByFolderTree(ctx, projectID, folderID)
	if err != nil {
		return err
	}
	if err := s.repository.DeleteUnits(ctx, projectID, unitIDs); err != nil {
		return err
	}
	if s.datapoints != nil {
		for _, unitID := range unitIDs {
			_, _ = s.datapoints.MarkInvalidBySource(ctx, projectID, "calc.output", unitID, &claims.UserID)
		}
	}
	return s.repository.DeleteFolder(ctx, projectID, folderID)
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
	if err := s.ensureComputeFolderInProject(ctx, projectID, normalized.FolderID); err != nil {
		return nil, err
	}

	record, err := s.repository.UpdateUnit(ctx, repository.UpdateComputeUnitParams{
		ID:            current.ID,
		ProjectID:     projectID,
		UserID:        claims.UserID,
		Name:          normalized.Name,
		Description:   normalized.Description,
		FolderID:      normalized.FolderID,
		Language:      normalized.Language,
		ScriptCode:    normalized.ScriptCode,
		TriggerType:   normalized.TriggerType,
		TriggerConfig: normalized.TriggerConfig,
		InputBindings: normalized.InputBindings,
		OutputBinding: normalized.OutputBinding,
		Dependencies:  normalized.Dependencies,
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
	input.DryRun = false
	return s.executeComputeUnit(ctx, claims, projectID, unitID, "run", input)
}

// DebugComputeUnit 触发一次调试运行。
func (s *ComputeService) DebugComputeUnit(ctx context.Context, claims *auth.Claims, projectID, unitID string, input RunComputeUnitInput) (*ComputeRunResult, error) {
	return s.executeComputeUnit(ctx, claims, projectID, unitID, "debug", input)
}

// CheckComputeSyntax 检查未保存脚本语法。
func (s *ComputeService) CheckComputeSyntax(ctx context.Context, claims *auth.Claims, projectID string, input ComputeSyntaxCheckInput) (*ComputeSyntaxCheckResult, error) {
	if err := s.validateReadAccess(claims, projectID); err != nil {
		return nil, err
	}
	language, err := normalizeComputeLanguage(input.Language)
	if err != nil {
		return nil, err
	}
	checker, err := s.pickSyntaxChecker(language)
	if err != nil {
		return nil, err
	}
	result, err := checker.CheckSyntax(ctx, enginecompute.SyntaxCheckRequest{
		Script:  strings.TrimSpace(input.ScriptCode),
		Timeout: defaultSyntaxCheckTimeout,
	})
	if err != nil {
		if errors.Is(err, enginecompute.ErrTimeout) {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "语法检查超时")
		}
		return nil, err
	}
	diagnostics := make([]ComputeSyntaxDiagnostic, 0, len(result.Diagnostics))
	for _, item := range result.Diagnostics {
		diagnostics = append(diagnostics, ComputeSyntaxDiagnostic{
			Severity:  normalizeDiagnosticSeverity(item.Severity),
			Message:   strings.TrimSpace(item.Message),
			Line:      positiveOrDefault(item.Line, 1),
			Column:    positiveOrDefault(item.Column, 1),
			EndLine:   positiveOrDefault(item.EndLine, positiveOrDefault(item.Line, 1)),
			EndColumn: positiveOrDefault(item.EndColumn, positiveOrDefault(item.Column, 1)+1),
			Source:    strings.TrimSpace(item.Source),
		})
	}
	return &ComputeSyntaxCheckResult{Diagnostics: diagnostics}, nil
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
	dryRun := input.DryRun && triggerMode == "debug"

	runner, err := s.pickRunner(unit.Language)
	if err != nil {
		return nil, err
	}
	sdkContext, err := s.prepareComputeSDKContext(ctx, *unit, input.Input)
	if err != nil {
		return nil, err
	}

	callback, err := s.startComputeSDKCallback(*unit, dryRun)
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

	if executeErr != nil && !dryRun {
		if errors.Is(executeErr, enginecompute.ErrTimeout) {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "计算执行超时", executeErr)
		}
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "计算脚本执行失败", executeErr)
	}
	if status == "success" && !dryRun {
		_ = s.writeComputeOutputValues(ctx, *unit, executeResult.Output, claims.UserID)
	}

	result := &ComputeRunResult{
		Status:      status,
		DurationMS:  int(executeResult.Duration.Milliseconds()),
		DryRun:      dryRun,
		Output:      executeResult.Output,
		SideEffects: cloneJSONArray(executeResult.SideEffects),
		Logs:        computeRunLogs(executeResult),
		StartedAt:   startedAt,
		FinishedAt:  finishedAt,
	}
	if errorMessage != nil {
		result.ErrorMessage = *errorMessage
	}
	return result, nil
}

func (s *ComputeService) validateDependencies() error {
	if s == nil || s.repository == nil || s.nodeRunner == nil || s.pythonRunner == nil || s.scheduler == nil {
		return apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "compute 服务依赖未初始化")
	}
	return nil
}

func computeRunOutputPayload(result enginecompute.ExecuteResult) any {
	logs := computeRunLogs(result)
	if len(result.SideEffects) == 0 && len(logs) == 0 {
		return result.Output
	}
	return map[string]any{
		"result":      result.Output,
		"sideEffects": cloneJSONArray(result.SideEffects),
		"logs":        logs,
	}
}

func computeRunLogs(result enginecompute.ExecuteResult) []string {
	logs := make([]string, 0, 2)
	if stdout := computeVisibleLog(result.Stdout); stdout != "" {
		logs = append(logs, stdout)
	}
	if stderr := strings.TrimSpace(result.Stderr); stderr != "" {
		logs = append(logs, stderr)
	}
	return logs
}

func computeVisibleLog(stdout string) string {
	stdout = strings.TrimSpace(stdout)
	if stdout == "" {
		return ""
	}
	index := strings.LastIndex(stdout, "__DATA_SERVICE_RESULT__:")
	if index < 0 {
		return stdout
	}
	return strings.TrimSpace(stdout[:index])
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

func (s *ComputeService) ensureComputeFolderInProject(ctx context.Context, projectID string, folderID *string) error {
	if folderID == nil {
		return nil
	}
	records, err := s.repository.ListFolders(ctx, projectID)
	if err != nil {
		return err
	}
	if _, ok := findComputeFolderRecord(records, *folderID); !ok {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "计算文件夹不存在")
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

func (s *ComputeService) pickSyntaxChecker(language string) (enginecompute.SyntaxChecker, error) {
	runner, err := s.pickRunner(language)
	if err != nil {
		return nil, err
	}
	checker, ok := runner.(enginecompute.SyntaxChecker)
	if !ok || checker == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "当前语言不支持语法检查")
	}
	return checker, nil
}

func normalizeDiagnosticSeverity(severity string) string {
	switch strings.TrimSpace(strings.ToLower(severity)) {
	case "warning", "warn":
		return "warning"
	case "info", "hint":
		return "info"
	default:
		return "error"
	}
}

func positiveOrDefault(value, fallback int) int {
	if value > 0 {
		return value
	}
	if fallback > 0 {
		return fallback
	}
	return 1
}

func toComputeUnit(record repository.ComputeUnitRecord) ComputeUnit {
	status := "disabled"
	if record.IsEnabled {
		status = "enabled"
	}
	path := "calc." + normalizeDatapointSegment(record.Name)
	return ComputeUnit{
		ID:            record.ID,
		ProjectID:     record.ProjectID,
		Name:          record.Name,
		Description:   cloneOptionalString(record.Description),
		FolderID:      cloneOptionalString(record.FolderID),
		Language:      record.Language,
		Lang:          computeLanguageForFrontend(record.Language),
		ScriptCode:    record.ScriptCode,
		Code:          record.ScriptCode,
		TriggerType:   record.TriggerType,
		TriggerConfig: cloneMap(record.TriggerConfig),
		InputBindings: cloneMap(record.InputBindings),
		OutputBinding: cloneMap(record.OutputBinding),
		Dependencies:  cloneJSONArray(record.Dependencies),
		TimeoutMS:     record.TimeoutMS,
		IsEnabled:     record.IsEnabled,
		Status:        status,
		Path:          path,
		OutputPath:    path,
		CreatedAt:     record.CreatedAt,
		UpdatedAt:     record.UpdatedAt,
	}
}

func computeLanguageForFrontend(language string) string {
	switch strings.TrimSpace(strings.ToLower(language)) {
	case "js":
		return "javascript"
	default:
		return strings.TrimSpace(strings.ToLower(language))
	}
}

func toComputeFolder(record repository.ComputeFolderRecord) ComputeFolder {
	return ComputeFolder{
		ID:        record.ID,
		ProjectID: record.ProjectID,
		Name:      record.Name,
		ParentID:  cloneOptionalString(record.ParentID),
		CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt,
	}
}

func toComputeFolderTree(records []repository.ComputeFolderRecord) []*ComputeFolder {
	nodes := make(map[string]*ComputeFolder, len(records))
	result := make([]*ComputeFolder, 0)
	for _, record := range records {
		folder := toComputeFolder(record)
		nodes[folder.ID] = &folder
	}
	for _, record := range records {
		node := nodes[record.ID]
		if node == nil {
			continue
		}
		if record.ParentID != nil {
			if parent := nodes[*record.ParentID]; parent != nil {
				parent.Children = append(parent.Children, node)
				continue
			}
		}
		result = append(result, node)
	}
	return result
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
	Description   *string
	FolderID      *string
	Language      string
	ScriptCode    string
	TriggerType   string
	TriggerConfig map[string]any
	InputBindings map[string]any
	OutputBinding map[string]any
	Dependencies  []any
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
	scriptCode, err := normalizeComputeScript(input.ScriptCode, language)
	if err != nil {
		return normalizedComputeUnitInput{}, err
	}
	folderID, err := normalizeOptionalComputeFolderID(input.FolderID)
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
		Description:   normalizeOptionalText(input.Description),
		FolderID:      folderID,
		Language:      language,
		ScriptCode:    scriptCode,
		TriggerType:   triggerType,
		TriggerConfig: cloneMap(input.TriggerConfig),
		InputBindings: cloneMap(input.InputBindings),
		OutputBinding: normalizeComputeOutputBinding(input.OutputBinding),
		Dependencies:  cloneJSONArray(input.Dependencies),
		TimeoutMS:     timeoutMS,
		IsEnabled:     true,
	}, nil
}

func mergeComputeUpdateInput(current repository.ComputeUnitRecord, input UpdateComputeUnitInput) (normalizedComputeUnitInput, error) {
	if input.Name == nil && !input.HasDescription && input.Language == nil && input.ScriptCode == nil && input.TriggerType == nil &&
		!input.HasFolderID && !input.HasTriggerConfig && !input.HasInputBindings && !input.HasOutputBinding && !input.HasDependencies && input.TimeoutMS == nil && input.IsEnabled == nil {
		return normalizedComputeUnitInput{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "至少需要提供一个待更新字段")
	}

	result := normalizedComputeUnitInput{
		Name:          current.Name,
		Description:   cloneOptionalString(current.Description),
		FolderID:      cloneOptionalString(current.FolderID),
		Language:      current.Language,
		ScriptCode:    current.ScriptCode,
		TriggerType:   current.TriggerType,
		TriggerConfig: cloneMap(current.TriggerConfig),
		InputBindings: cloneMap(current.InputBindings),
		OutputBinding: cloneMap(current.OutputBinding),
		Dependencies:  cloneJSONArray(current.Dependencies),
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
	if input.HasDescription {
		result.Description = normalizeOptionalText(input.Description)
	}
	if input.HasFolderID {
		result.FolderID, err = normalizeOptionalComputeFolderID(input.FolderID)
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
		result.ScriptCode, err = normalizeComputeScript(*input.ScriptCode, result.Language)
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
		result.OutputBinding = normalizeComputeOutputBinding(input.OutputBinding)
	}
	if input.HasDependencies {
		result.Dependencies = cloneJSONArray(input.Dependencies)
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
	result.OutputBinding = normalizeComputeOutputBinding(result.OutputBinding)
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
	if language == "" {
		language = "js"
	}
	if language == "javascript" {
		language = "js"
	}
	if _, ok := allowedComputeLanguages[language]; !ok {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "language 仅支持 js/python")
	}
	return language, nil
}

func normalizeComputeScript(scriptCode, language string) (string, error) {
	scriptCode = strings.TrimSpace(scriptCode)
	if scriptCode == "" {
		return defaultComputeScript(language), nil
	}
	return scriptCode, nil
}

func defaultComputeScript(language string) string {
	if strings.TrimSpace(strings.ToLower(language)) == "python" {
		return "def main(argv, dp, ctx):\n    return argv[0] if len(argv) > 0 else None"
	}
	return "return argv[0] ?? null;"
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

func validateComputeFolderID(folderID string) error {
	if _, err := uuid.Parse(strings.TrimSpace(folderID)); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "id 格式无效", err)
	}
	return nil
}

func normalizeOptionalComputeFolderID(folderID *string) (*string, error) {
	if folderID == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*folderID)
	if trimmed == "" {
		return nil, nil
	}
	if _, err := uuid.Parse(trimmed); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "folderId 格式无效", err)
	}
	return &trimmed, nil
}

func normalizeComputeFolderName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "name 不能为空")
	}
	if len([]rune(name)) > 100 {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "name 长度不能超过 100")
	}
	return name, nil
}

func builtInComputeDependencies() []ComputeDependency {
	return []ComputeDependency{
		{
			ID:          "js:dayjs",
			Name:        "dayjs",
			Runtime:     "javascript",
			Version:     "1.11.x",
			Description: "时间解析、格式化和计算。",
			Status:      "enabled",
			ImportName:  "dayjs",
		},
		{
			ID:          "js:lodash",
			Name:        "lodash",
			Runtime:     "javascript",
			Version:     "4.17.x",
			Description: "集合、对象和数组处理工具。",
			Status:      "enabled",
			ImportName:  "_",
		},
		{
			ID:          "python:math",
			Name:        "math",
			Runtime:     "python",
			Version:     "stdlib",
			Description: "Python 标准数学函数库。",
			Status:      "enabled",
			ImportName:  "math",
		},
		{
			ID:          "python:statistics",
			Name:        "statistics",
			Runtime:     "python",
			Version:     "stdlib",
			Description: "Python 标准统计函数库。",
			Status:      "enabled",
			ImportName:  "statistics",
		},
	}
}

func findComputeFolderRecord(records []repository.ComputeFolderRecord, folderID string) (repository.ComputeFolderRecord, bool) {
	for _, record := range records {
		if record.ID == folderID {
			return record, true
		}
	}
	return repository.ComputeFolderRecord{}, false
}

func isDescendantComputeFolder(records []repository.ComputeFolderRecord, folderID, possibleDescendantID string) bool {
	currentID := possibleDescendantID
	visited := map[string]struct{}{}
	for currentID != "" {
		if currentID == folderID {
			return true
		}
		if _, exists := visited[currentID]; exists {
			return false
		}
		visited[currentID] = struct{}{}
		current, ok := findComputeFolderRecord(records, currentID)
		if !ok || current.ParentID == nil {
			return false
		}
		currentID = *current.ParentID
	}
	return false
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
	existingOutputs, err := s.datapoints.ListByProjectAndSource(ctx, unit.ProjectID, "calc.output", unit.ID)
	if err != nil {
		return err
	}
	existingByOutputName := make(map[string]repository.DataPointRecord, len(existingOutputs))
	for _, existing := range existingOutputs {
		outputName := strings.TrimSpace(firstString(existing.SourceConfig, "outputName"))
		if outputName == "" {
			outputName = datapointOutputNameFromPath(existing.Path)
		}
		if outputName != "" {
			existingByOutputName[outputName] = existing
		}
	}
	_, _ = s.datapoints.MarkInvalidBySource(ctx, unit.ProjectID, "calc.output", unit.ID, &userID)
	outputs := extractComputeOutputBindings(unit)
	for _, output := range outputs {
		path := output.Path
		existingOutput, hasExistingOutput := existingByOutputName[output.Name]
		if existing, err := s.datapoints.GetByProjectAndPath(ctx, unit.ProjectID, path); err == nil && existing != nil &&
			existing.ID != existingOutput.ID &&
			(existing.SourceID == nil || *existing.SourceID != unit.ID || existing.SourceType != "calc.output") {
			path = path + "_" + unit.ID[:8]
		}
		params := repository.CreateDataPointParams{
			ProjectID:    unit.ProjectID,
			UserID:       &userID,
			Path:         path,
			Name:         unit.Name,
			Description:  cloneOptionalString(output.Description),
			SourceType:   "calc.output",
			SourceID:     &unit.ID,
			SourceConfig: map[string]any{"computeUnitId": unit.ID, "outputName": output.Name},
			DataType:     output.DataType,
			Unit:         cloneOptionalString(output.Unit),
			Tags:         []any{},
			RefreshMode:  "manual",
			Status:       "active",
		}
		if hasExistingOutput {
			_, err = s.datapoints.UpdateGeneratedOutput(ctx, existingOutput.ID, params)
		} else {
			_, err = s.datapoints.UpsertByPath(ctx, params)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func datapointOutputNameFromPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	index := strings.LastIndex(path, ".")
	if index < 0 || index == len(path)-1 {
		return ""
	}
	return strings.TrimSpace(path[index+1:])
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

func normalizeComputeOutputBinding(input map[string]any) map[string]any {
	result := cloneMap(input)
	if len(extractComputeOutputBindings(repository.ComputeUnitRecord{
		Name:          "default",
		OutputBinding: result,
	})) > 0 {
		return result
	}
	result["outputs"] = []any{map[string]any{
		"name":     "result",
		"dataType": "object",
	}}
	return result
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
