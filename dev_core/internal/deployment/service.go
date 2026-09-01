package deployment

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

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
	Version     string
	Name        string
	Description string
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
	MarkVersionReady(context.Context, string, string, CreateVersionInput) (Version, error)
	MarkVersionFailed(context.Context, string, string, string) (Version, error)
	DeleteVersion(context.Context, string, string) error
	ListProjectDeployments(context.Context, string, string) ([]Deployment, error)
	Deploy(context.Context, string, DeployInput) ([]Deployment, error)
	Operate(context.Context, string, string, string) (Deployment, error)
}

type ServiceConfig struct{ ArtifactBucket string }

// ReleaseSource 是正式构建器唯一接受的已构建输入；服务层不执行任意工程命令。
type ReleaseSource struct {
	Client, Runtime, Collector                               []byte
	SBOM, ResourceRecommendation, HealthContract, SchemaPlan []byte
	ProjectDocument                                          map[string]any
	SourceRevision, BuilderID                                string
}
type ReleaseSourceBuilder interface {
	BuildReleaseSource(context.Context, Project) (ReleaseSource, error)
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
	input.Version = strings.TrimPrefix(strings.TrimSpace(input.Version), "v")
	if input.Version == "" {
		return Version{}, fmt.Errorf("版本号不能为空")
	}
	project, err := s.repository.GetProject(ctx, actor.TenantID, projectID)
	if err != nil {
		return Version{}, err
	}
	if err := requireProject(actor, project, auth.CapabilityReleasePublish); err != nil {
		return Version{}, err
	}
	// 旧实现会把 workspace 源码 ZIP 直接标记为 ready，与正式 Release 的构建、
	// 摘要清单和签名契约完全不同。保留读取接口用于识别历史记录，但禁止再制造新旁路。
	return Version{}, ErrLegacyReleaseDisabled
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
