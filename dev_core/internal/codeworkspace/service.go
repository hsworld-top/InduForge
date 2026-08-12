package codeworkspace

import (
	"context"
	"errors"
	"fmt"
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
	Image      string
	BindHost   string
	VolumeName string
}

type Status struct {
	ContainerName string       `json:"containerName"`
	Status        string       `json:"status"`
	HostPort      string       `json:"hostPort,omitempty"`
	OnlineUsers   []OnlineUser `json:"onlineUsers,omitempty"`
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
	projects ProjectRepository
	engine   Engine
	config   Config
	presence PresenceStore
	now      func() time.Time
}

func NewService(projects ProjectRepository, engine Engine, config Config) (*Service, error) {
	if projects == nil || engine == nil {
		return nil, fmt.Errorf("代码工作区依赖不能为空")
	}
	config.Image = strings.TrimSpace(config.Image)
	config.BindHost = strings.TrimSpace(config.BindHost)
	config.VolumeName = strings.TrimSpace(config.VolumeName)
	if config.Image == "" {
		return nil, fmt.Errorf("code-server 镜像不能为空")
	}
	if config.BindHost == "" {
		config.BindHost = "127.0.0.1"
	}
	if config.VolumeName == "" {
		return nil, fmt.Errorf("代码工作区共享卷名称不能为空")
	}
	return &Service{projects: projects, engine: engine, config: config, now: time.Now}, nil
}

func (s *Service) SetPresence(store PresenceStore) {
	s.presence = store
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

// Start 首次进入时创建工程唯一容器；已有容器时只启动，不覆盖共享配置和扩展目录。
func (s *Service) Start(ctx context.Context, actor auth.User, projectID string) (Status, error) {
	item, err := s.authorize(ctx, actor, projectID)
	if err != nil {
		return Status{}, err
	}
	name := containerName(projectID)
	state, err := s.engine.Inspect(ctx, name)
	if err == nil {
		if err := s.validateOwnership(state, projectID); err != nil {
			return Status{}, err
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
	if state.Running {
		switch state.Health {
		case "starting":
			status = "starting"
		case "unhealthy":
			status = "error"
		default:
			status = "running"
		}
	}
	return Status{ContainerName: containerName(projectID), Status: status, HostPort: state.HostPort}, nil
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
	return ContainerSpec{
		Name: containerName(item.ID), Image: s.config.Image,
		Command:       []string{"--bind-addr", "0.0.0.0:3000", "--auth", "none", "--disable-telemetry", "--idle-timeout-seconds", "1800", workspacePath},
		User:          "1000:1000",
		WorkingDir:    workspacePath,
		Environment:   []string{"PNPM_HOME=/cache/pnpm", "npm_config_store_dir=/cache/pnpm-store", "XDG_CACHE_HOME=/cache"},
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
