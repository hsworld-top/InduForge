package deployment

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/indu-forge/dev_core/internal/auth"
	projectaccess "github.com/indu-forge/dev_core/internal/project"
)

var (
	ErrNotFound              = errors.New("发布或部署记录不存在")
	ErrAlreadyExists         = errors.New("版本已存在")
	ErrVersionInUse          = errors.New("版本正在使用，不能删除")
	ErrScenesNotReady        = errors.New("场景草稿未提交或运行工件校验失败")
	ErrLegacyReleaseDisabled = errors.New("正式版本构建与交付尚未开放，当前不能创建正式部署；已保存源码仍可用于开发预览")
)

type Project struct {
	ID            string
	TenantID      string
	Name          string
	Code          string
	WorkspacePath string
	CreatedBy     string
	Visibility    string
}

type Version struct {
	ID             string
	TenantID       string
	ProjectID      string
	Version        string
	Name           string
	Description    string
	Status         string
	SourceHash     string
	ArtifactKey    string
	ArtifactBucket string
	ArtifactHash   string
	ArtifactSize   int64
	Manifest       map[string]any
	ManifestHash   string
	ChecksumsHash  string
	SigningKeyID   string
	ErrorMessage   string
	CompletedAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Deployment struct {
	ID                   string
	TenantID             string
	NodeID               string
	ProjectID            string
	ApplicationVersionID string
	Version              string
	Mode                 string
	Status               string
	RuntimeConfig        map[string]any
	RuntimeMetrics       map[string]any
	ErrorMessage         string
	NodeName             string
	NodeStatus           string
	IPAddress            string
	ProjectName          string
	ProjectCode          string
	ArtifactHash         string
	Manifest             map[string]any
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type PublishInput struct {
	Name        string
	Description string
	// Authorization 仅用于向数据域转发当前请求身份，绝不写入日志或发布制品。
	Authorization string `json:"-"`
	// RequestID 仅用于发布链路的运维诊断，不进入制品、数据库错误信息或下游请求。
	RequestID string `json:"-"`
}

type CreateVersionInput struct {
	Project      Project
	Version      string
	Name         string
	Description  string
	SourceHash   string
	ArtifactKey  string
	ArtifactHash string
	ArtifactSize int64
	Manifest     map[string]any
	CreatedBy    string
	Bucket       string
}
type VersionReadyInput struct {
	Bucket, ArtifactKey, ArtifactHash, ManifestHash, ChecksumsHash, SigningKeyID string
	ArtifactSize                                                                 int64
	Manifest                                                                     map[string]any
}

type DeployInput struct {
	Project       Project
	Version       *Version
	NodeIDs       []string
	Mode          string
	RuntimeConfig map[string]any
	ArtifactURL   string
	ArtifactKey   string
	ArtifactHash  string
	DeployedBy    string
}

type Repository interface {
	GetProject(context.Context, string, string) (Project, error)
	ListVersions(context.Context, string, string, int, int) ([]Version, int64, error)
	GetVersion(context.Context, string, string) (Version, error)
	GetDeployment(context.Context, string, string) (Deployment, error)
	CreateVersion(context.Context, CreateVersionInput) (Version, error)
	BeginProductionBuild(context.Context, Project, string, string, string) (Version, error)
	MarkVersionReady(context.Context, string, string, VersionReadyInput) (Version, error)
	MarkVersionFailed(context.Context, string, string, string) (Version, error)
	DeleteVersion(context.Context, string, string) error
	ListProjectDeployments(context.Context, string, string) ([]Deployment, error)
	Deploy(context.Context, string, DeployInput) ([]Deployment, error)
	Operate(context.Context, string, string, string) (Deployment, error)
}

type ServiceConfig struct{ ArtifactBucket, MinNodeAgentVersion, MinRuntimeVersion string }

// ReleaseSource 是正式构建器唯一接受的已构建输入；服务层不执行任意工程命令。
type ReleaseSource struct {
	Client, Runtime, Collector                               []byte
	CollectorSourceSnapshot                                  []byte
	SBOM, ResourceRecommendation, HealthContract, SchemaPlan []byte
	ProjectDocument                                          map[string]any
	SourceRevision, BuilderID                                string
}
type ReleaseSourceBuilder interface {
	BuildReleaseSource(context.Context, Project, Version, string) (ReleaseSource, error)
}

// DevelopmentRequirementsSource 只读取当前数据域运行工件，用于开发部署在创建
// 临时制品前展示权威引擎需求。它不构建前端、不写版本或对象存储。
type DevelopmentRequirementsSource interface {
	DevelopmentEngineRequirements(context.Context, Project, string) ([]string, error)
}

// DevelopmentArtifact 是内部部署槽使用的临时制品描述；它不写 application_versions，
// 固定展示版本为 __DEV__，每次构建以唯一 releaseId 和摘要区分。
type DevelopmentArtifact struct {
	ReleaseID, Version, Bucket, ArtifactKey, ArtifactHash, ManifestHash, ChecksumsHash, SigningKeyID string
	ArtifactSize                                                                                     int64
	// Manifest 必须保留 Builder 输出的原始字节。若经 map 反序列化再编码，会改变
	// 嵌套 collector sourceSnapshot 的 artifact 字节，从而破坏其 SHA-256。
	Manifest json.RawMessage
}
type SigningConfig struct {
	Key   ed25519.PrivateKey
	KeyID string
}

type Service struct {
	repository    Repository
	workspace     Workspace
	store         ArtifactStore
	config        ServiceConfig
	events        Events
	releases      ReleaseValidator
	sourceBuilder ReleaseSourceBuilder
	signing       SigningConfig
}

type Events interface {
	DeployStatus(string, map[string]any)
}

type ReleaseValidator interface {
	ValidateRelease(context.Context, auth.User, string) (map[string]any, error)
}

func NewService(repository Repository, workspace Workspace, store ArtifactStore, config ServiceConfig) *Service {
	return &Service{repository: repository, workspace: workspace, store: store, config: config}
}

func (s *Service) SetEvents(events Events) { s.events = events }

func (s *Service) SetReleaseValidator(validator ReleaseValidator)       { s.releases = validator }
func (s *Service) SetReleaseSourceBuilder(builder ReleaseSourceBuilder) { s.sourceBuilder = builder }
func (s *Service) SetSigningConfig(config SigningConfig)                { s.signing = config }

// DevelopmentEngineRequirements 返回当前工程数据工件决定的开发部署引擎；开发态
// 不创建 application_versions，也不允许页面从历史版本猜测可选引擎。
func (s *Service) DevelopmentEngineRequirements(ctx context.Context, actor auth.User, projectID, authorization string) ([]string, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityDeploymentExecute); err != nil {
		return nil, err
	}
	project, err := s.repository.GetProject(ctx, actor.TenantID, projectID)
	if err != nil {
		return nil, err
	}
	if err = requireProject(actor, project, auth.CapabilityDeploymentExecute); err != nil {
		return nil, err
	}
	source, ok := s.sourceBuilder.(DevelopmentRequirementsSource)
	if !ok {
		return nil, fmt.Errorf("开发引擎需求预检未配置")
	}
	requirements, err := source.DevelopmentEngineRequirements(ctx, project, authorization)
	if err != nil {
		return nil, fmt.Errorf("读取开发引擎需求失败: %w", err)
	}
	return requirements, nil
}

// BuildDevelopmentArtifact 复用正式受控源码构建、签名和对象存储链路，但不创建用户可见版本记录。
func (s *Service) BuildDevelopmentArtifact(ctx context.Context, actor auth.User, projectID, authorization string) (DevelopmentArtifact, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityDeploymentExecute); err != nil {
		return DevelopmentArtifact{}, err
	}
	project, err := s.repository.GetProject(ctx, actor.TenantID, projectID)
	if err != nil {
		return DevelopmentArtifact{}, err
	}
	if err = requireProject(actor, project, auth.CapabilityDeploymentExecute); err != nil {
		return DevelopmentArtifact{}, err
	}
	if s.sourceBuilder == nil || len(s.signing.Key) != ed25519.PrivateKeySize || strings.TrimSpace(s.signing.KeyID) == "" {
		return DevelopmentArtifact{}, fmt.Errorf("开发制品构建器或签名配置未配置")
	}
	version := Version{ID: uuid.NewString(), ProjectID: project.ID, TenantID: project.TenantID, Version: "__DEV__"}
	source, err := s.sourceBuilder.BuildReleaseSource(ctx, project, version, authorization)
	if err != nil {
		return DevelopmentArtifact{}, fmt.Errorf("构建开发制品输入失败: %w", err)
	}
	result, err := assembleFormalRelease(project, version, source, s.signing, s.config, time.Now().UTC())
	if err != nil {
		return DevelopmentArtifact{}, fmt.Errorf("组装开发制品失败: %w", err)
	}
	key := path.Join("development", project.TenantID, project.ID, version.ID, result.OuterSHA256+".tar.zst")
	ref, err := s.store.Put(ctx, key, bytes.NewReader(result.Bundle), result.Size, "application/zstd")
	if err != nil || ref.Key != key || ref.Size != result.Size {
		if err == nil {
			err = fmt.Errorf("开发制品对象存储确认不匹配")
		}
		return DevelopmentArtifact{}, err
	}
	if !json.Valid(result.Manifest) {
		return DevelopmentArtifact{}, fmt.Errorf("开发制品 Manifest 无效")
	}
	return DevelopmentArtifact{ReleaseID: version.ID, Version: version.Version, Bucket: ref.Bucket, ArtifactKey: key, ArtifactHash: result.OuterSHA256, ArtifactSize: result.Size, Manifest: append(json.RawMessage(nil), result.Manifest...), ManifestHash: result.ManifestSHA256, ChecksumsHash: result.ChecksumsSHA256, SigningKeyID: s.signing.KeyID}, nil
}

func (s *Service) ListVersions(ctx context.Context, actor auth.User, projectID string, page, limit int) ([]Version, int64, error) {
	project, err := s.repository.GetProject(ctx, actor.TenantID, projectID)
	if err != nil {
		return nil, 0, err
	}
	if err := requireProject(actor, project, auth.CapabilityProjectRead); err != nil {
		return nil, 0, err
	}
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	return s.repository.ListVersions(ctx, actor.TenantID, projectID, page, limit)
}

func (s *Service) Publish(ctx context.Context, actor auth.User, projectID string, input PublishInput) (Version, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityReleasePublish); err != nil {
		return Version{}, err
	}
	project, err := s.repository.GetProject(ctx, actor.TenantID, projectID)
	if err != nil {
		return Version{}, err
	}
	if err := requireProject(actor, project, auth.CapabilityReleasePublish); err != nil {
		return Version{}, err
	}
	version, err := s.repository.BeginProductionBuild(ctx, project, actor.ID, input.Name, input.Description)
	if err != nil {
		return Version{}, err
	}
	fail := func(stage string, cause error) (Version, error) {
		// HTTP 客户端取消或上游超时不能让版本永久停在 building。使用短后台上下文
		// 尝试完成终态回写；若数据库暂不可用，下次发布的超时调和仍会兜底。
		markCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		message := releaseDiagnosticError(cause)
		if _, markErr := s.repository.MarkVersionFailed(markCtx, actor.TenantID, version.ID, message); markErr != nil {
			slog.Default().Error("正式 Release 失败状态回写失败", "requestId", input.RequestID, "versionId", version.ID, "stage", stage, "cause", message, "error", releaseDiagnosticError(markErr))
		} else {
			slog.Default().Error("正式 Release 构建失败", "requestId", input.RequestID, "versionId", version.ID, "stage", stage, "error", releaseDiagnosticError(cause))
		}
		return Version{}, cause
	}
	if s.sourceBuilder == nil || len(s.signing.Key) != ed25519.PrivateKeySize || strings.TrimSpace(s.signing.KeyID) == "" {
		return fail("configuration", fmt.Errorf("正式 Release 构建器或签名配置未配置"))
	}
	source, err := s.sourceBuilder.BuildReleaseSource(ctx, project, version, input.Authorization)
	if err != nil {
		return fail("source", fmt.Errorf("读取正式 Release 构建输入失败: %w", err))
	}
	result, err := assembleFormalRelease(project, version, source, s.signing, s.config, time.Now().UTC())
	if err != nil {
		return fail("assemble", fmt.Errorf("组装正式 Release 失败: %w", err))
	}
	ready, err := uploadFormalRelease(ctx, s.store, project, version, result)
	if err != nil {
		return fail("upload", fmt.Errorf("上传正式 Release 失败: %w", err))
	}
	ready.SigningKeyID = s.signing.KeyID
	completed, err := s.repository.MarkVersionReady(ctx, actor.TenantID, version.ID, ready)
	if err != nil {
		return fail("mark-ready", fmt.Errorf("确认正式 Release 失败: %w", err))
	}
	return completed, nil
}

// releaseDiagnosticError 截断可能来自外部依赖的错误，避免意外把认证信息写入日志。
func releaseDiagnosticError(err error) string {
	message := strings.TrimSpace(err.Error())
	// PostgreSQL text 不接受 NUL。Docker multiplex 日志或异常上游响应可能含控制字节，
	// 统一替换后再作为失败状态与运维日志内容，保证错误回写本身不会再次失败。
	message = strings.Map(func(value rune) rune {
		if value == 0 || (value < 0x20 && value != '\n' && value != '\t') {
			return ' '
		}
		return value
	}, message)
	for _, prefix := range []string{"Bearer ", "bearer ", "Basic ", "basic "} {
		if index := strings.Index(message, prefix); index >= 0 {
			end := strings.IndexAny(message[index:], " \t\r\n\"'")
			if end <= len(prefix) {
				end = len(message) - index
			}
			message = message[:index] + prefix + "[REDACTED]" + message[index+end:]
		}
	}
	const maxLength = 1024
	if len(message) > maxLength {
		return message[:maxLength] + "…(已截断)"
	}
	return message
}

func (s *Service) DeleteVersion(ctx context.Context, actor auth.User, versionID string) error {
	if err := auth.RequireCapability(actor, auth.CapabilityReleasePublish); err != nil {
		return err
	}
	version, err := s.repository.GetVersion(ctx, actor.TenantID, versionID)
	if err != nil {
		return err
	}
	project, err := s.repository.GetProject(ctx, actor.TenantID, version.ProjectID)
	if err != nil {
		return err
	}
	if err := requireProject(actor, project, auth.CapabilityReleasePublish); err != nil {
		return err
	}
	return s.repository.DeleteVersion(ctx, actor.TenantID, versionID)
}

func (s *Service) ListProjectDeployments(ctx context.Context, actor auth.User, projectID string) ([]Deployment, error) {
	project, err := s.repository.GetProject(ctx, actor.TenantID, projectID)
	if err != nil {
		return nil, err
	}
	if err := requireProject(actor, project, auth.CapabilityProjectRead); err != nil {
		return nil, err
	}
	return s.repository.ListProjectDeployments(ctx, actor.TenantID, projectID)
}

func (s *Service) DeployVersion(ctx context.Context, actor auth.User, versionID string, nodeIDs []string, runtimeConfig map[string]any) ([]Deployment, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityDeploymentExecute); err != nil {
		return nil, err
	}
	version, err := s.repository.GetVersion(ctx, actor.TenantID, versionID)
	if err != nil {
		return nil, err
	}
	project, err := s.repository.GetProject(ctx, actor.TenantID, version.ProjectID)
	if err != nil {
		return nil, err
	}
	if err := requireProject(actor, project, auth.CapabilityDeploymentExecute); err != nil {
		return nil, err
	}
	return nil, ErrLegacyReleaseDisabled
}

func (s *Service) DeployDevelopment(ctx context.Context, actor auth.User, projectID string, nodeIDs []string, runtimeConfig map[string]any) ([]Deployment, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityDeploymentExecute); err != nil {
		return nil, err
	}
	project, err := s.repository.GetProject(ctx, actor.TenantID, projectID)
	if err != nil {
		return nil, err
	}
	if err := requireProject(actor, project, auth.CapabilityDeploymentExecute); err != nil {
		return nil, err
	}
	return nil, ErrLegacyReleaseDisabled
}

func (s *Service) Operate(ctx context.Context, actor auth.User, nodeDeploymentID, operation string) (Deployment, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityDeploymentOperate); err != nil {
		return Deployment{}, err
	}
	if operation != "start" && operation != "stop" && operation != "restart" && operation != "remove" {
		return Deployment{}, fmt.Errorf("部署操作无效")
	}
	current, err := s.repository.GetDeployment(ctx, actor.TenantID, nodeDeploymentID)
	if err != nil {
		return Deployment{}, err
	}
	project, err := s.repository.GetProject(ctx, actor.TenantID, current.ProjectID)
	if err != nil {
		return Deployment{}, err
	}
	if err := requireProject(actor, project, auth.CapabilityDeploymentOperate); err != nil {
		return Deployment{}, err
	}
	// stop/remove 仍保留用于收口历史节点部署；start/restart 不得重新拉起旧源码包。
	if operation == "start" || operation == "restart" {
		return Deployment{}, ErrLegacyReleaseDisabled
	}
	item, err := s.repository.Operate(ctx, actor.TenantID, nodeDeploymentID, operation)
	if err == nil && s.events != nil {
		s.events.DeployStatus(actor.TenantID, deploymentEvent(item, operation == "remove"))
	}
	return item, err
}

func requireProject(actor auth.User, item Project, capability auth.Capability) error {
	return projectaccess.RequireCapability(actor, projectaccess.Project{
		ID: item.ID, TenantID: item.TenantID, CreatedBy: item.CreatedBy, Visibility: item.Visibility,
	}, capability)
}

func (s *Service) publishDeployments(tenantID string, items []Deployment, err error) {
	if err != nil || s.events == nil {
		return
	}
	for _, item := range items {
		s.events.DeployStatus(tenantID, deploymentEvent(item, false))
	}
}

func deploymentEvent(item Deployment, removed bool) map[string]any {
	return map[string]any{
		"deploymentId": item.ID, "nodeId": item.NodeID, "projectId": item.ProjectID,
		"status": item.Status, "errorMessage": item.ErrorMessage, "removed": removed,
	}
}

func (s *Service) Rollback(ctx context.Context, actor auth.User, versionID, nodeID string) ([]Deployment, error) {
	return s.DeployVersion(ctx, actor, versionID, []string{nodeID}, map[string]any{"rollback": true})
}

type uploadedArtifact struct {
	Key          string
	Bucket       string
	SourceHash   string
	ArtifactHash string
	Size         int64
}

func (s *Service) buildAndUpload(ctx context.Context, project Project, version string, manifest map[string]any) (uploadedArtifact, error) {
	files, err := s.workspace.Export(project.WorkspacePath)
	if err != nil {
		return uploadedArtifact{}, fmt.Errorf("读取工程工作空间失败: %w", err)
	}
	artifact, err := BuildArtifact(files, manifest)
	if err != nil {
		return uploadedArtifact{}, err
	}
	key := path.Join("versions", project.TenantID, project.ID, version, artifact.ArtifactHash+".ifp")
	ref, err := s.store.Put(ctx, key, bytes.NewReader(artifact.Content), artifact.Size, "application/zip")
	if err != nil {
		return uploadedArtifact{}, err
	}
	return uploadedArtifact{Key: ref.Key, Bucket: ref.Bucket, SourceHash: artifact.SourceHash, ArtifactHash: artifact.ArtifactHash, Size: ref.Size}, nil
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
