package store

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/indu-forge/node_agent/internal/pkg/types"
	"github.com/indu-forge/node_agent/internal/pkg/utils"
	"github.com/indu-forge/node_agent/internal/pkg/logger"
)

// LocalStore 本地存储实现
type LocalStore struct {
	dataDir string
	logger  *logger.SimpleLogger
}

// NewLocalStore 创建本地存储
func NewLocalStore(dataDir string) *LocalStore {
	return &LocalStore{
		dataDir: dataDir,
		logger:  logger.GlobalLogger,
	}
}

func (s *LocalStore) ensureDir(path string) error {
	return utils.EnsureDir(path)
}

// SaveProject 保存项目信息
func (s *LocalStore) SaveProject(project *types.ProjectInfo) error {
	dir := filepath.Join(s.dataDir, "projects", project.ID)
	if err := s.ensureDir(dir); err != nil {
		return err
	}

	file := filepath.Join(dir, "project.json")
	data, err := json.MarshalIndent(project, "", "  ")
	if err != nil {
		return err
	}

	s.logger.Info("保存项目信息", "project", project.ID, "path", file)
	return os.WriteFile(file, data, 0644)
}

// GetProject 获取项目信息
func (s *LocalStore) GetProject(projectID string) (*types.ProjectInfo, error) {
	file := filepath.Join(s.dataDir, "projects", projectID, "project.json")
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var project types.ProjectInfo
	if err := json.Unmarshal(data, &project); err != nil {
		return nil, err
	}

	return &project, nil
}

// ListProjects 列出所有项目
func (s *LocalStore) ListProjects() ([]*types.ProjectInfo, error) {
	dir := filepath.Join(s.dataDir, "projects")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*types.ProjectInfo{}, nil
		}
		return nil, err
	}

	var projects []*types.ProjectInfo
	for _, entry := range entries {
		if entry.IsDir() {
			project, err := s.GetProject(entry.Name())
			if err != nil {
				s.logger.Warn("读取项目失败", "project", entry.Name(), "error", err)
				continue
			}
			projects = append(projects, project)
		}
	}

	return projects, nil
}

// DeleteProject 删除项目
func (s *LocalStore) DeleteProject(projectID string) error {
	dir := filepath.Join(s.dataDir, "projects", projectID)
	s.logger.Info("删除项目", "project", projectID, "path", dir)
	return os.RemoveAll(dir)
}

// SaveConnectionProfile 保存连接配置
func (s *LocalStore) SaveConnectionProfile(projectID string, profile types.ConnectionProfile) error {
	dir := filepath.Join(s.dataDir, "projects", projectID)
	if err := s.ensureDir(dir); err != nil {
		return err
	}

	// 敏感数据脱敏
	profile.Secrets = nil

	file := filepath.Join(dir, "profile.json")
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(file, data, 0600)
}

// GetConnectionProfile 获取连接配置
func (s *LocalStore) GetConnectionProfile(projectID string) (*types.ConnectionProfile, error) {
	file := filepath.Join(s.dataDir, "projects", projectID, "profile.json")
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var profile types.ConnectionProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, err
	}

	return &profile, nil
}

// SaveVersion 保存版本文件
func (s *LocalStore) SaveVersion(projectID string, version string, files map[string][]byte) error {
	dir := filepath.Join(s.dataDir, "projects", projectID, "versions", version)
	if err := s.ensureDir(dir); err != nil {
		return err
	}

	for name, content := range files {
		file := filepath.Join(dir, name)
		if err := os.WriteFile(file, content, 0755); err != nil {
			return err
		}
	}

	s.logger.Info("保存版本", "project", projectID, "version", version)
	return nil
}

// GetVersionDir 获取版本目录
func (s *LocalStore) GetVersionDir(projectID string, version string) (string, error) {
	dir := filepath.Join(s.dataDir, "projects", projectID, "versions", version)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return "", err
	}
	return dir, nil
}

// ListVersions 列出版本
func (s *LocalStore) ListVersions(projectID string) ([]string, error) {
	dir := filepath.Join(s.dataDir, "projects", projectID, "versions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var versions []string
	for _, entry := range entries {
		if entry.IsDir() {
			versions = append(versions, entry.Name())
		}
	}

	return versions, nil
}

// SwitchVersion 切换当前版本（软链接）
func (s *LocalStore) SwitchVersion(projectID string, version string) error {
	projectDir := filepath.Join(s.dataDir, "projects", projectID)
	versionDir := filepath.Join(projectDir, "versions", version)
	currentLink := filepath.Join(projectDir, "current")

	if err := utils.Symlink(versionDir, currentLink); err != nil {
		return err
	}

	s.logger.Info("切换版本", "project", projectID, "version", version)
	return nil
}

// GetCurrentVersion 获取当前版本
func (s *LocalStore) GetCurrentVersion(projectID string) (string, error) {
	projectDir := filepath.Join(s.dataDir, "projects", projectID)
	currentLink := filepath.Join(projectDir, "current")

	link, err := os.Readlink(currentLink)
	if err != nil {
		return "", err
	}

	return filepath.Base(link), nil
}
