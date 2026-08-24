package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
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
	"schedule":         {},
	"datapoint_change": {},
}

var computeScheduleTimePattern = regexp.MustCompile(`^(?:[01]\d|2[0-3]):[0-5]\d:[0-5]\d$`)

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

// ComputeCapabilities 返回独立沙箱声明的真实能力；Available=false 时前端必须禁用调试和语法检查。
type ComputeCapabilities struct {
	SandboxStatus    string                               `json:"sandboxStatus"`
	LanguageVersions []enginecompute.LanguageCapability   `json:"languages"`
	SDK              []string                             `json:"sdk"`
	Dependencies     []enginecompute.DependencyCapability `json:"dependencies"`
	TriggerTypes     []string                             `json:"triggerTypes"`
	Limits           enginecompute.SandboxLimits          `json:"limits"`
}

// ComputeSchedulePreview 描述调度配置的规范化结果和未来执行时间；仅用于开发态校验与预览。
type ComputeSchedulePreview struct {
	TriggerType   string                      `json:"triggerType"`
	TriggerConfig map[string]any              `json:"triggerConfig"`
	Summary       string                      `json:"summary"`
	NextRuns      []time.Time                 `json:"nextRuns"`
	Errors        []ComputeScheduleFieldError `json:"errors"`
}

type ComputeScheduleFieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
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
	nodeRunner   enginecompute.Runner
	pythonRunner enginecompute.Runner
}

// NewComputeService 创建 compute 服务。
func NewComputeService(
	repo *repository.ComputeRepository,
	nodeRunner enginecompute.Runner,
	pythonRunner enginecompute.Runner,
	deps ...any,
) *ComputeService {
	var datapoints *repository.DataPointRepository
	var queries *QueryService
	for _, dep := range deps {
		switch typed := dep.(type) {
		case *repository.DataPointRepository:
			datapoints = typed
		case *QueryService:
			queries = typed
		}
	}
	return &ComputeService{
		repository:   repo,
		datapoints:   datapoints,
		queries:      queries,
		nodeRunner:   nodeRunner,
		pythonRunner: pythonRunner,
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

// ListComputeDependencies 返回沙箱真实声明的依赖，不再展示宿主机或前端臆测的能力。
func (s *ComputeService) ListComputeDependencies(ctx context.Context, claims *auth.Claims, projectID string) (*ComputeDependencyListResult, error) {
	if err := s.validateReadAccess(claims, projectID); err != nil {
		return nil, err
	}
	capabilities, err := s.GetComputeCapabilities(ctx, claims, projectID)
	if err != nil {
		return nil, err
	}
	result := make([]ComputeDependency, 0, len(capabilities.Dependencies))
	for _, dependency := range capabilities.Dependencies {
		runtimeName := strings.TrimSpace(strings.ToLower(dependency.Language))
		if runtimeName == "js" {
			runtimeName = "javascript"
		}
		result = append(result, ComputeDependency{
			ID: dependency.Language + ":" + dependency.Name, Name: dependency.Name,
			Runtime: runtimeName, Version: dependency.Version, Description: "由独立计算沙箱提供。",
			Status: "enabled", ImportName: dependency.Name,
		})
	}
	return &ComputeDependencyListResult{List: result}, nil
}

// GetComputeCapabilities 读取沙箱实时能力；不可用时返回 unavailable，不回退到宿主执行器。
func (s *ComputeService) GetComputeCapabilities(ctx context.Context, claims *auth.Claims, projectID string) (*ComputeCapabilities, error) {
	if err := s.validateReadAccess(claims, projectID); err != nil {
		return nil, err
	}
	provider, ok := s.nodeRunner.(enginecompute.CapabilityProvider)
	if !ok || provider == nil {
		return &ComputeCapabilities{SandboxStatus: "unavailable", LanguageVersions: []enginecompute.LanguageCapability{}, SDK: []string{}, Dependencies: []enginecompute.DependencyCapability{}, TriggerTypes: []string{}}, nil
	}
	capabilities, err := provider.Capabilities(ctx)
	if err != nil || !capabilities.Available {
		return &ComputeCapabilities{SandboxStatus: "unavailable", LanguageVersions: []enginecompute.LanguageCapability{}, SDK: []string{}, Dependencies: []enginecompute.DependencyCapability{}, TriggerTypes: []string{}, Limits: capabilities.Limits}, nil
	}
	return &ComputeCapabilities{
		SandboxStatus: "available", LanguageVersions: capabilities.Languages, SDK: capabilities.SDK,
		Dependencies: capabilities.Dependencies, TriggerTypes: capabilities.Triggers, Limits: capabilities.Limits,
	}, nil
}

// PreviewComputeSchedule 在不保存、不执行的前提下校验并展示未来五次计划时间。
func (s *ComputeService) PreviewComputeSchedule(ctx context.Context, claims *auth.Claims, projectID, triggerType string, triggerConfig map[string]any) (*ComputeSchedulePreview, error) {
	if err := s.validateReadAccess(claims, projectID); err != nil {
		return nil, err
	}
	preview := &ComputeSchedulePreview{TriggerType: strings.TrimSpace(strings.ToLower(triggerType)), TriggerConfig: cloneMap(triggerConfig), NextRuns: []time.Time{}, Errors: []ComputeScheduleFieldError{}}
	normalizedType, err := normalizeComputeTriggerType(triggerType)
	if err != nil {
		preview.Errors = append(preview.Errors, ComputeScheduleFieldError{Field: "triggerType", Message: "触发类型不受支持"})
		return preview, nil
	}
	draft := normalizedComputeUnitInput{TriggerType: normalizedType, TriggerConfig: cloneMap(triggerConfig)}
	if err := s.validateAndNormalizeTrigger(ctx, projectID, &draft); err != nil {
		preview.TriggerType = normalizedType
		preview.Errors = append(preview.Errors, computeScheduleValidationError(normalizedType, triggerConfig, err))
		return preview, nil
	}
	preview.TriggerType, preview.TriggerConfig = draft.TriggerType, draft.TriggerConfig
	if draft.TriggerType != "schedule" {
		preview.Summary = "该触发方式不按时间计划自动执行"
		return preview, nil
	}
	preview.Summary, preview.NextRuns = computeScheduleOccurrences(draft.TriggerConfig, time.Now().UTC(), 5)
	return preview, nil
}

func computeScheduleValidationError(triggerType string, config map[string]any, validationErr error) ComputeScheduleFieldError {
	field := "triggerConfig"
	if triggerType == "datapoint_change" {
		field = "triggerConfig.datapointId"
	} else if triggerType == "schedule" {
		switch strings.ToLower(strings.TrimSpace(toString(config["kind"]))) {
		case "interval":
			if _, ok := positiveInteger(config["every"]); !ok {
				field = "triggerConfig.every"
			} else {
				field = "triggerConfig.unit"
			}
		case "daily", "weekly":
			if !computeScheduleTimePattern.MatchString(strings.TrimSpace(toString(config["time"]))) {
				field = "triggerConfig.time"
			} else if timezone := strings.TrimSpace(toString(config["timezone"])); timezone == "" {
				field = "triggerConfig.timezone"
			} else if _, err := time.LoadLocation(timezone); err != nil {
				field = "triggerConfig.timezone"
			} else {
				field = "triggerConfig.weekdays"
			}
		default:
			field = "triggerConfig.kind"
		}
	}
	return ComputeScheduleFieldError{Field: field, Message: validationErr.Error()}
}

func computeScheduleOccurrences(config map[string]any, now time.Time, count int) (string, []time.Time) {
	kind := toString(config["kind"])
	if kind == "interval" {
		every, _ := positiveInteger(config["every"])
		unit := toString(config["unit"])
		step := time.Duration(every) * time.Second
		if unit == "minutes" {
			step = time.Duration(every) * time.Minute
		}
		if unit == "hours" {
			step = time.Duration(every) * time.Hour
		}
		result := make([]time.Time, 0, count)
		for index := 1; index <= count; index++ {
			result = append(result, now.Add(time.Duration(index)*step))
		}
		return fmt.Sprintf("每 %d %s执行一次", every, map[string]string{"seconds": "秒", "minutes": "分钟", "hours": "小时"}[unit]), result
	}
	location, _ := time.LoadLocation(toString(config["timezone"]))
	if location == nil {
		location = time.UTC
	}
	hour, minute, second := parseScheduleClock(toString(config["time"]))
	localNow := now.In(location)
	result := make([]time.Time, 0, count)
	for offset := 0; len(result) < count && offset < 370; offset++ {
		day := localNow.AddDate(0, 0, offset)
		if kind == "weekly" && !scheduleContainsWeekday(config["weekdays"], day.Weekday()) {
			continue
		}
		candidate := time.Date(day.Year(), day.Month(), day.Day(), hour, minute, second, 0, location)
		// 夏令时跳跃导致本地时刻不存在时，time.Date 会自动归一化；契约要求该次跳过而不是补跑。
		candidateLocal := candidate.In(location)
		if candidateLocal.Year() != day.Year() || candidateLocal.Month() != day.Month() || candidateLocal.Day() != day.Day() || candidateLocal.Hour() != hour || candidateLocal.Minute() != minute || candidateLocal.Second() != second {
			continue
		}
		if !candidate.After(localNow) {
			continue
		}
		result = append(result, candidate.UTC())
	}
	if kind == "weekly" {
		return "每周指定日期定时执行", result
	}
	return "每天定时执行", result
}

func parseScheduleClock(value string) (int, int, int) {
	parts := strings.Split(value, ":")
	if len(parts) != 3 {
		return 0, 0, 0
	}
	hour, _ := strconv.Atoi(parts[0])
	minute, _ := strconv.Atoi(parts[1])
	second, _ := strconv.Atoi(parts[2])
	return hour, minute, second
}

func scheduleContainsWeekday(value any, weekday time.Weekday) bool {
	for _, day := range normalizedWeekdays(value) {
		if day == int(weekday) || (weekday == time.Sunday && day == 7) {
			return true
		}
	}
	return false
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
	if err := s.validateAndNormalizeTrigger(ctx, projectID, &normalized); err != nil {
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
	if err := s.validateAndNormalizeTrigger(ctx, projectID, &normalized); err != nil {
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

	startedAt := time.Now().UTC()
	request := enginecompute.ExecuteRequest{
		Script:     unit.ScriptCode,
		Input:      cloneMap(input.Input),
		SDKContext: sdkContext,
		Timeout:    time.Duration(unit.TimeoutMS) * time.Millisecond,
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
	if s == nil || s.repository == nil || s.nodeRunner == nil || s.pythonRunner == nil {
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

// validateAndNormalizeTrigger 统一校验未来节点会消费的触发契约；data_service 不负责调度执行。
func (s *ComputeService) validateAndNormalizeTrigger(ctx context.Context, projectID string, input *normalizedComputeUnitInput) error {
	if input == nil {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "trigger 配置不能为空")
	}
	config := cloneMap(input.TriggerConfig)
	switch input.TriggerType {
	case "manual":
		input.TriggerConfig = map[string]any{}
		return nil
	case "datapoint_change":
		id := strings.TrimSpace(toString(config["datapointId"]))
		if id == "" {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "数据点变化触发必须选择数据点")
		}
		if _, err := uuid.Parse(id); err != nil {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "datapointId 格式无效")
		}
		if s.datapoints == nil {
			return apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "数据点仓储未初始化")
		}
		point, err := s.datapoints.GetByProjectAndID(ctx, projectID, id)
		if err != nil {
			return err
		}
		if point == nil || !strings.EqualFold(point.Status, "active") {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "变化触发数据点必须是有效数据点")
		}
		input.TriggerConfig = map[string]any{"datapointId": point.ID, "path": point.Path}
		return nil
	case "schedule":
		kind := strings.TrimSpace(strings.ToLower(toString(config["kind"])))
		switch kind {
		case "interval":
			every, ok := positiveInteger(config["every"])
			if !ok {
				return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "周期执行 every 必须为正整数")
			}
			unit := strings.TrimSpace(strings.ToLower(toString(config["unit"])))
			if unit != "seconds" && unit != "minutes" && unit != "hours" {
				return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "周期执行 unit 仅支持 seconds/minutes/hours")
			}
			input.TriggerConfig = map[string]any{"kind": kind, "every": every, "unit": unit}
			return nil
		case "daily", "weekly":
			timeOfDay := strings.TrimSpace(toString(config["time"]))
			if !computeScheduleTimePattern.MatchString(timeOfDay) {
				return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "定时执行 time 必须为 HH:mm:ss")
			}
			timezone := strings.TrimSpace(toString(config["timezone"]))
			if timezone == "" {
				return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "定时执行 timezone 不能为空")
			}
			if _, err := time.LoadLocation(timezone); err != nil {
				return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "timezone 必须为有效 IANA 时区")
			}
			normalized := map[string]any{"kind": kind, "time": timeOfDay, "timezone": timezone}
			if kind == "weekly" {
				if !validWeekdaysInput(config["weekdays"]) {
					return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "每周定时 weekdays 只能包含 ISO 1-7")
				}
				weekdays := normalizedWeekdays(config["weekdays"])
				if len(weekdays) == 0 {
					return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "每周定时至少选择一个星期")
				}
				normalized["weekdays"] = weekdays
			}
			input.TriggerConfig = normalized
			return nil
		default:
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "schedule.kind 仅支持 interval/daily/weekly")
		}
	default:
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "triggerType 不受支持")
	}
}

func positiveInteger(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, typed > 0
	case int64:
		return int(typed), typed > 0 && typed <= int64(^uint(0)>>1)
	case float64:
		return int(typed), typed > 0 && typed == float64(int(typed))
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		return parsed, err == nil && parsed > 0
	default:
		return 0, false
	}
}

func normalizedWeekdays(value any) []int {
	items, ok := value.([]any)
	if !ok {
		if ints, castOK := value.([]int); castOK {
			items = make([]any, len(ints))
			for index, item := range ints {
				items[index] = item
			}
		}
	}
	seen := map[int]struct{}{}
	for _, item := range items {
		day, ok := positiveInteger(item)
		if ok && day >= 1 && day <= 7 {
			seen[day] = struct{}{}
		}
	}
	result := make([]int, 0, len(seen))
	for day := range seen {
		result = append(result, day)
	}
	sort.Ints(result)
	return result
}

func validWeekdaysInput(value any) bool {
	items, ok := value.([]any)
	if !ok {
		if ints, castOK := value.([]int); castOK {
			items = make([]any, len(ints))
			for index, item := range ints {
				items[index] = item
			}
		} else {
			return false
		}
	}
	if len(items) == 0 {
		return false
	}
	for _, item := range items {
		day, valid := positiveInteger(item)
		if !valid || day < 1 || day > 7 {
			return false
		}
	}
	return true
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
