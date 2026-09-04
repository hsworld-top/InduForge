package codeworkspace

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/indu-forge/dev_core/internal/auth"
	platformcache "github.com/indu-forge/dev_core/internal/platform/cache"
	"github.com/indu-forge/dev_core/internal/project"
)

const (
	containerPort = "3000/tcp"
	workspacePath = "/workspace"
)

type ProjectRepository interface {
	Get(context.Context, string, string) (project.Project, error)
}

type Config struct {
	Image                         string
	BindHost                      string
	AllowedOrigins                string
	WorkspacePublicOriginTemplate string
	AllowInsecureHTTPDev          bool
	VolumeName                    string
	DefaultTemplateProjectID      string
	DefaultTemplateID             string
}

type Status struct {
	ContainerName string            `json:"containerName"`
	Status        string            `json:"status"`
	HostPort      string            `json:"hostPort,omitempty"`
	ServicePorts  map[string]string `json:"-"`
	OnlineUsers   []OnlineUser      `json:"onlineUsers,omitempty"`
}

type OnlineUser struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IsCurrent bool   `json:"isCurrent"`
}

type PresenceStore interface {
	TouchProjectMember(context.Context, string, string, string, time.Duration) error
	ListProjectMembers(context.Context, string, time.Time) ([]platformcache.ProjectMember, error)
}

type Service struct {
	projects   ProjectRepository
	engine     Engine
	config     Config
	presence   PresenceStore
	now        func() time.Time
	epochGuard interface {
		RequireAuthoringEpoch(context.Context, string, string, string) error
	}
}

func NewService(projects ProjectRepository, engine Engine, config Config) (*Service, error) {
	if projects == nil || engine == nil {
		return nil, fmt.Errorf("代码工作区依赖不能为空")
	}
	config.Image = strings.TrimSpace(config.Image)
	config.BindHost = strings.TrimSpace(config.BindHost)
	config.AllowedOrigins = strings.TrimSpace(config.AllowedOrigins)
	config.WorkspacePublicOriginTemplate = strings.TrimSpace(config.WorkspacePublicOriginTemplate)
	config.VolumeName = strings.TrimSpace(config.VolumeName)
	config.DefaultTemplateProjectID = strings.TrimSpace(config.DefaultTemplateProjectID)
	config.DefaultTemplateID = strings.TrimSpace(config.DefaultTemplateID)
	if config.Image == "" {
		return nil, fmt.Errorf("code-server 镜像不能为空")
	}
	if config.BindHost == "" {
		config.BindHost = "127.0.0.1"
	}
	if config.VolumeName == "" {
		return nil, fmt.Errorf("代码工作区共享卷名称不能为空")
	}
	if (config.DefaultTemplateProjectID == "") != (config.DefaultTemplateID == "") {
		return nil, fmt.Errorf("默认工程模板配置不完整")
	}
	if _, err := workspacePublicHosts(config.WorkspacePublicOriginTemplate, "00000000-0000-0000-0000-000000000000", config.AllowInsecureHTTPDev); err != nil {
		return nil, err
	}
	return &Service{projects: projects, engine: engine, config: config, now: time.Now}, nil
}

func (s *Service) SetPresence(store PresenceStore) {
	s.presence = store
}
func (s *Service) SetAuthoringEpochGuard(guard interface {
	RequireAuthoringEpoch(context.Context, string, string, string) error
}) {
	s.epochGuard = guard
}

func (s *Service) requireEpoch(ctx context.Context, actor auth.User, projectID string) error {
	if s.epochGuard == nil {
		return nil
	}
	return s.epochGuard.RequireAuthoringEpoch(ctx, actor.TenantID, projectID, project.AuthoringEpochFromContext(ctx))
}

// FreezeAuthoring 在开发态捕获/恢复临界区停止工程容器，并返回按原状态恢复的函数。
func (s *Service) FreezeAuthoring(ctx context.Context, actor auth.User, projectID string) (func(context.Context) error, error) {
	return s.freezeAuthoring(ctx, actor, projectID, nil, nil)
}

// FreezeAuthoringRestore 在首次停止前持久化原运行状态；重启后使用 known 恢复同一决定。
func (s *Service) FreezeAuthoringRestore(ctx context.Context, actor auth.User, projectID string, known *bool, persist func(bool) error) (func(context.Context) error, bool, error) {
	observed := false
	remember := func(running bool) error {
		observed = running
		if persist != nil {
			return persist(running)
		}
		return nil
	}
	resume, err := s.freezeAuthoring(ctx, actor, projectID, known, remember)
	wasRunning := known != nil && *known
	if known == nil {
		wasRunning = observed
	}
	return resume, wasRunning, err
}

func (s *Service) freezeAuthoring(ctx context.Context, actor auth.User, projectID string, known *bool, persist func(bool) error) (func(context.Context) error, error) {
	item, err := s.authorize(ctx, actor, projectID)
	if err != nil {
		return nil, err
	}
	name := containerName(projectID)
	state, err := s.engine.Inspect(ctx, name)
	if known != nil {
		if *known {
			if err == nil && state.Running {
				if stopErr := s.engine.Stop(ctx, name); stopErr != nil {
					return nil, stopErr
				}
			}
			return s.resumeAuthoring(item), nil
		}
		if err == nil && state.Running {
			return nil, fmt.Errorf("恢复任务记录的工作区状态不一致")
		}
		return func(context.Context) error { return nil }, nil
	}
	if errors.Is(err, ErrContainerNotFound) {
		if persist != nil {
			if persistErr := persist(false); persistErr != nil {
				return nil, persistErr
			}
		}
		return func(context.Context) error { return nil }, nil
	}
	if err != nil {
		return nil, err
	}
	if !state.Running {
		if persist != nil {
			if persistErr := persist(false); persistErr != nil {
				return nil, persistErr
			}
		}
		return func(context.Context) error { return nil }, nil
	}
	if persist != nil {
		if persistErr := persist(true); persistErr != nil {
			return nil, persistErr
		}
	}
	if err := s.engine.Stop(ctx, name); err != nil {
		return nil, err
	}
	return s.resumeAuthoring(item), nil
}

func (s *Service) resumeAuthoring(item project.Project) func(context.Context) error {
	name := containerName(item.ID)
	return func(resumeCtx context.Context) error {
		if _, inspectErr := s.engine.Inspect(resumeCtx, name); errors.Is(inspectErr, ErrContainerNotFound) {
			spec, specErr := s.containerSpec(item)
			if specErr != nil {
				return specErr
			}
			if createErr := s.engine.Create(resumeCtx, spec); createErr != nil && !errors.Is(createErr, ErrContainerConflict) {
				return createErr
			}
		} else if inspectErr != nil {
			return inspectErr
		}
		return s.engine.Start(resumeCtx, name)
	}
}

func (s *Service) Status(ctx context.Context, actor auth.User, projectID string) (Status, error) {
	if _, err := s.authorize(ctx, actor, projectID); err != nil {
		return Status{}, err
	}
	status, err := s.inspect(ctx, projectID)
	if err != nil {
		return Status{}, err
	}
	s.attachPresence(ctx, actor, projectID, &status)
	return status, nil
}
func (s *Service) AuthoringEpoch(ctx context.Context, actor auth.User, projectID string) (string, error) {
	item, err := s.authorize(ctx, actor, projectID)
	if err != nil {
		return "", err
	}
	return project.FormatAuthoringEpoch(item.AuthoringEpoch), nil
}
func (s *Service) RequireProxyEpoch(ctx context.Context, actor auth.User, projectID, epoch string) error {
	if _, err := s.authorize(ctx, actor, projectID); err != nil {
		return err
	}
	if s.epochGuard == nil {
		return fmt.Errorf("工程编辑代次保护未配置")
	}
	return s.epochGuard.RequireAuthoringEpoch(ctx, actor.TenantID, projectID, epoch)
}

// Start 首次进入时创建工程唯一容器；已有容器时只启动，不覆盖共享配置和扩展目录。
func (s *Service) Start(ctx context.Context, actor auth.User, projectID string) (Status, error) {
	item, err := s.authorize(ctx, actor, projectID)
	if err != nil {
		return Status{}, err
	}
	if err := s.requireEpoch(ctx, actor, projectID); err != nil {
		return Status{}, err
	}
	name := containerName(projectID)
	state, err := s.engine.Inspect(ctx, name)
	if err == nil {
		if err := s.validateOwnership(state, projectID); err != nil {
			return Status{}, err
		}
		// 不对已失败的集群 Pod 调用 Start。Kubernetes 不会启动终止 Pod，
		// 继续调用只会掩盖错误并让前端误判成可以首次初始化的 stopped 状态。
		if state.Health == "unhealthy" {
			return s.inspect(ctx, projectID)
		}
		if !state.Running {
			if err := s.engine.Start(ctx, name); err != nil {
				return Status{}, err
			}
		}
		return s.inspect(ctx, projectID)
	}
	if !errors.Is(err, ErrContainerNotFound) {
		return Status{}, err
	}

	spec, err := s.containerSpec(item)
	if err != nil {
		return Status{}, err
	}
	if err := s.engine.Create(ctx, spec); err != nil {
		if !errors.Is(err, ErrContainerConflict) {
			return Status{}, err
		}
		state, inspectErr := s.engine.Inspect(ctx, name)
		if inspectErr != nil {
			return Status{}, err
		}
		if ownershipErr := s.validateOwnership(state, projectID); ownershipErr != nil {
			return Status{}, ownershipErr
		}
	}
	if err := s.engine.Start(ctx, name); err != nil {
		return Status{}, err
	}
	return s.inspect(ctx, projectID)
}

func (s *Service) Stop(ctx context.Context, actor auth.User, projectID string) (Status, error) {
	if _, err := s.authorize(ctx, actor, projectID); err != nil {
		return Status{}, err
	}
	name := containerName(projectID)
	state, err := s.engine.Inspect(ctx, name)
	if errors.Is(err, ErrContainerNotFound) {
		return Status{ContainerName: name, Status: "missing"}, nil
	}
	if err != nil {
		return Status{}, err
	}
	if err := s.requireEpoch(ctx, actor, projectID); err != nil {
		return Status{}, err
	}
	if err := s.validateOwnership(state, projectID); err != nil {
		return Status{}, err
	}
	if err := s.engine.Stop(ctx, name); err != nil {
		return Status{}, err
	}
	return s.inspect(ctx, projectID)
}

// Rebuild 只重建容器可写层，五个工程持久化目录均保留。
func (s *Service) Rebuild(ctx context.Context, actor auth.User, projectID string) (Status, error) {
	item, err := s.authorize(ctx, actor, projectID)
	if err != nil {
		return Status{}, err
	}
	if err := s.requireEpoch(ctx, actor, projectID); err != nil {
		return Status{}, err
	}
	name := containerName(projectID)
	state, inspectErr := s.engine.Inspect(ctx, name)
	if inspectErr == nil {
		if err := s.validateOwnership(state, projectID); err != nil {
			return Status{}, err
		}
		if err := s.engine.Remove(ctx, name); err != nil {
			return Status{}, err
		}
	} else if !errors.Is(inspectErr, ErrContainerNotFound) {
		return Status{}, inspectErr
	}
	spec, err := s.containerSpec(item)
	if err != nil {
		return Status{}, err
	}
	if err := s.engine.Create(ctx, spec); err != nil {
		return Status{}, err
	}
	if err := s.engine.Start(ctx, spec.Name); err != nil {
		return Status{}, err
	}
	return s.inspect(ctx, projectID)
}

func (s *Service) authorize(ctx context.Context, actor auth.User, projectID string) (project.Project, error) {
	if _, err := uuid.Parse(projectID); err != nil {
		return project.Project{}, project.ErrNotFound
	}
	item, err := s.projects.Get(ctx, actor.TenantID, projectID)
	if err != nil {
		return project.Project{}, err
	}
	if err := project.RequireCapability(actor, item, auth.CapabilityProjectWrite); err != nil {
		return project.Project{}, err
	}
	return item, nil
}

func (s *Service) inspect(ctx context.Context, projectID string) (Status, error) {
	state, err := s.engine.Inspect(ctx, containerName(projectID))
	if errors.Is(err, ErrContainerNotFound) {
		return Status{ContainerName: containerName(projectID), Status: "missing"}, nil
	}
	if err != nil {
		return Status{}, err
	}
	if err := s.validateOwnership(state, projectID); err != nil {
		return Status{}, err
	}
	status := "stopped"
	switch state.Health {
	case "starting":
		status = "starting"
	case "unhealthy":
		status = "error"
	default:
		if state.Running {
			status = "running"
		}
	}
	return Status{ContainerName: containerName(projectID), Status: status, HostPort: state.HostPort, ServicePorts: state.ServicePorts}, nil
}

func (s *Service) containerSpec(item project.Project) (ContainerSpec, error) {
	projectDirectory := filepath.Dir(item.WorkspacePath)
	for _, directory := range []string{
		item.WorkspacePath,
		filepath.Join(projectDirectory, "code-server-data"),
		filepath.Join(projectDirectory, "code-server-config"),
		filepath.Join(projectDirectory, "cache"),
		filepath.Join(projectDirectory, "context-state", "current"),
	} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			return ContainerSpec{}, fmt.Errorf("准备代码工作区目录失败: %w", err)
		}
	}
	environment := []string{"PNPM_HOME=/cache/pnpm", "npm_config_store_dir=/cache/pnpm-store", "XDG_CACHE_HOME=/cache"}
	publicHosts, err := workspacePublicHosts(s.config.WorkspacePublicOriginTemplate, item.ID, s.config.AllowInsecureHTTPDev)
	if err != nil {
		return ContainerSpec{}, err
	}
	if len(publicHosts) > 0 {
		// 仅在中心公开工作区模板启用时给两个开发服务注入自身的精确 Host。
		// 不复用 Origin 白名单，更不能使用通配符，避免 DNS rebinding 扩大源码暴露面。
		environment = append(environment,
			"PI_WEB_ALLOWED_HOSTS="+publicHosts["ai"],
			"__VITE_ADDITIONAL_SERVER_ALLOWED_HOSTS="+publicHosts["preview"],
		)
	}
	if s.config.AllowedOrigins != "" {
		// 工作区四入口由中心页面跨端口访问；来源清单随中心部署配置注入，
		// 不能沿用镜像内仅供本机开发的 localhost 默认值。
		environment = append(environment, "PREVIEW_CONTROL_ALLOWED_ORIGINS="+s.config.AllowedOrigins, "PI_WEB_ALLOWED_PARENT_ORIGINS="+s.config.AllowedOrigins)
	}
	// 内置教程工程的源码同样必须经由代码工作区的官方模板初始化，不能由发布链路
	// 复制或拼装。该环境变量只在工作区为空时生效，初始化器会原子写入共享卷。
	if item.ID == s.config.DefaultTemplateProjectID {
		environment = append(environment, "INDUFORGE_DEFAULT_WORKSPACE_TEMPLATE="+s.config.DefaultTemplateID)
	}
	return ContainerSpec{
		Name: containerName(item.ID), Image: s.config.Image,
		Command:       []string{"--bind-addr", "0.0.0.0:3000", "--auth", "none", "--disable-telemetry", "--idle-timeout-seconds", "1800", workspacePath},
		User:          "1000:1000",
		WorkingDir:    workspacePath,
		Environment:   environment,
		ContainerPort: containerPort, BindHost: s.config.BindHost,
		Labels: map[string]string{"com.induforge.managed": "true", "com.induforge.project-id": item.ID, "com.induforge.role": "code-workspace"},
		Mounts: []Mount{
			{Source: s.config.VolumeName, Subpath: path.Join(item.ID, "workspace"), Target: workspacePath},
			{Source: s.config.VolumeName, Subpath: path.Join(item.ID, "context-state", "current"), Target: "/workspace/.induforge/context", ReadOnly: true},
			{Source: s.config.VolumeName, Subpath: path.Join(item.ID, "code-server-data"), Target: "/home/coder/.local/share/code-server"},
			{Source: s.config.VolumeName, Subpath: path.Join(item.ID, "code-server-config"), Target: "/home/coder/.config/code-server"},
			{Source: s.config.VolumeName, Subpath: path.Join(item.ID, "cache"), Target: "/cache"},
		},
	}, nil
}

// workspacePublicHosts 从受控公开 Origin 模板为一个工程生成服务自身的 Hostname。
// URL 的端口属于网关/浏览器 Origin，不应传给 Pi Web 或 Vite 的 allowedHosts 配置。
func workspacePublicHosts(template, projectID string, allowInsecureHTTPDev bool) (map[string]string, error) {
	template = strings.TrimSpace(template)
	if template == "" {
		return nil, nil
	}
	if !strings.Contains(template, "{projectId}") || !strings.Contains(template, "{service}") {
		return nil, fmt.Errorf("工作区公开 Origin 模板必须包含 {projectId} 和 {service}")
	}
	if _, err := uuid.Parse(projectID); err != nil {
		return nil, fmt.Errorf("工作区公开 Origin 模板工程 ID 无效: %w", err)
	}
	hosts := make(map[string]string, 2)
	for _, serviceName := range []string{"ai", "preview"} {
		origin := strings.NewReplacer("{projectId}", strings.ToLower(projectID), "{service}", serviceName).Replace(template)
		parsed, err := url.Parse(origin)
		expectedScheme := "https"
		if allowInsecureHTTPDev {
			expectedScheme = "http"
		}
		if err != nil || parsed.Scheme != expectedScheme || parsed.Hostname() == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
			return nil, fmt.Errorf("工作区公开 Origin 模板无效")
		}
		host := strings.ToLower(parsed.Hostname())
		if strings.Contains(host, "*") {
			return nil, fmt.Errorf("工作区公开 Origin 模板不得使用通配 Host")
		}
		hosts[serviceName] = host
	}
	return hosts, nil
}

func (s *Service) validateOwnership(state ContainerState, projectID string) error {
	if state.Labels["com.induforge.managed"] != "true" || state.Labels["com.induforge.project-id"] != projectID || state.Labels["com.induforge.role"] != "code-workspace" {
		return ErrContainerConflict
	}
	return nil
}

func (s *Service) attachPresence(ctx context.Context, actor auth.User, projectID string, status *Status) {
	if s.presence == nil {
		return
	}
	name := strings.TrimSpace(actor.FullName)
	if name == "" {
		name = actor.Username
	}
	const ttl = 45 * time.Second
	if err := s.presence.TouchProjectMember(ctx, projectID, actor.ID, name, ttl); err != nil {
		return
	}
	members, err := s.presence.ListProjectMembers(ctx, projectID, s.now())
	if err != nil {
		return
	}
	status.OnlineUsers = make([]OnlineUser, 0, len(members))
	for _, member := range members {
		status.OnlineUsers = append(status.OnlineUsers, OnlineUser{ID: member.ID, Name: member.Name, IsCurrent: member.ID == actor.ID})
	}
}

func containerName(projectID string) string {
	return "induforge-code-" + strings.ToLower(projectID)
}
