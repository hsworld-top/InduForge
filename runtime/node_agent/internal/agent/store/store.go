package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/indu-forge/node_agent/internal/pkg/logger"
	"github.com/indu-forge/node_agent/internal/pkg/types"
	"github.com/indu-forge/node_agent/internal/pkg/utils"
)

// LocalStore 本地存储实现（SQLite 元数据 + 文件版本目录）。
type LocalStore struct {
	dataDir string
	db      *sql.DB
	logger  *logger.SimpleLogger
}

// NewLocalStore 创建本地存储。
func NewLocalStore(dataDir string) *LocalStore {
	store := &LocalStore{
		dataDir: dataDir,
		logger:  logger.GlobalLogger,
	}
	if err := store.initSQLite(); err != nil {
		panic("初始化 SQLite 存储失败: " + err.Error())
	}
	return store
}

func (s *LocalStore) ensureDir(path string) error {
	return utils.EnsureDir(path)
}

// DataDir 返回数据目录路径。
func (s *LocalStore) DataDir() string {
	return s.dataDir
}

func (s *LocalStore) initSQLite() error {
	if err := s.ensureDir(s.dataDir); err != nil {
		return err
	}
	dbPath := filepath.Join(s.dataDir, "node_agent.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	if _, err := db.Exec(`PRAGMA journal_mode=WAL;`); err != nil {
		_ = db.Close()
		return err
	}

	schema := []string{
		`CREATE TABLE IF NOT EXISTS projects (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			source TEXT NOT NULL,
			current_version TEXT NOT NULL,
			status TEXT NOT NULL,
			connection_profile TEXT,
			deployed_at TEXT,
			last_started_at TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS connection_profiles (
			project_id TEXT PRIMARY KEY,
			profile_json TEXT NOT NULL
		);`,
	}
	for _, stmt := range schema {
		if _, err := db.Exec(stmt); err != nil {
			_ = db.Close()
			return err
		}
	}
	s.db = db
	return nil
}

// Close 关闭 SQLite 连接。
func (s *LocalStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// SaveProject 保存项目信息。
func (s *LocalStore) SaveProject(project *types.ProjectInfo) error {
	if project == nil {
		return errors.New("project 不能为空")
	}
	if strings.TrimSpace(project.Source) == "" {
		project.Source = types.ProjectSourceLocal
	}
	profileJSON, err := json.Marshal(project.ConnectionProfile)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`INSERT INTO projects (id, name, source, current_version, status, connection_profile, deployed_at, last_started_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   name=excluded.name,
		   source=excluded.source,
		   current_version=excluded.current_version,
		   status=excluded.status,
		   connection_profile=excluded.connection_profile,
		   deployed_at=excluded.deployed_at,
		   last_started_at=excluded.last_started_at`,
		project.ID,
		project.Name,
		project.Source,
		project.CurrentVersion,
		project.Status,
		string(profileJSON),
		timeToNullableRFC3339(project.DeployedAt),
		timeToNullableRFC3339(project.LastStartedAt),
	)
	return err
}

// GetProject 获取项目信息。
func (s *LocalStore) GetProject(projectID string) (*types.ProjectInfo, error) {
	row := s.db.QueryRow(
		`SELECT id, name, source, current_version, status, connection_profile, deployed_at, last_started_at
		   FROM projects WHERE id = ?`,
		projectID,
	)
	return scanProject(row)
}

// ListProjects 列出所有项目。
func (s *LocalStore) ListProjects() ([]*types.ProjectInfo, error) {
	rows, err := s.db.Query(
		`SELECT id, name, source, current_version, status, connection_profile, deployed_at, last_started_at
		   FROM projects ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*types.ProjectInfo
	for rows.Next() {
		project, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return projects, nil
}

// DeleteProject 删除项目。
func (s *LocalStore) DeleteProject(projectID string) error {
	if _, err := s.db.Exec(`DELETE FROM projects WHERE id = ?`, projectID); err != nil {
		return err
	}
	return os.RemoveAll(filepath.Join(s.dataDir, "projects", projectID))
}

// SaveConnectionProfile 保存连接配置。
func (s *LocalStore) SaveConnectionProfile(projectID string, profile types.ConnectionProfile) error {
	profile.Secrets = nil
	data, err := json.Marshal(profile)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`INSERT INTO connection_profiles (project_id, profile_json)
		 VALUES (?, ?)
		 ON CONFLICT(project_id) DO UPDATE SET profile_json=excluded.profile_json`,
		projectID,
		string(data),
	)
	return err
}

// GetConnectionProfile 获取连接配置。
func (s *LocalStore) GetConnectionProfile(projectID string) (*types.ConnectionProfile, error) {
	row := s.db.QueryRow(`SELECT profile_json FROM connection_profiles WHERE project_id = ?`, projectID)
	var profileJSON string
	if err := row.Scan(&profileJSON); err != nil {
		return nil, err
	}
	var profile types.ConnectionProfile
	if err := json.Unmarshal([]byte(profileJSON), &profile); err != nil {
		return nil, err
	}
	return &profile, nil
}

// SaveVersion 保存版本文件。
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

// GetVersionDir 获取版本目录。
func (s *LocalStore) GetVersionDir(projectID string, version string) (string, error) {
	dir := filepath.Join(s.dataDir, "projects", projectID, "versions", version)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return "", err
	}
	return dir, nil
}

// ListVersions 列出版本。
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

// SwitchVersion 切换当前版本（软链接）。
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

// GetCurrentVersion 获取当前版本。
func (s *LocalStore) GetCurrentVersion(projectID string) (string, error) {
	projectDir := filepath.Join(s.dataDir, "projects", projectID)
	currentLink := filepath.Join(projectDir, "current")

	link, err := os.Readlink(currentLink)
	if err != nil {
		return "", err
	}
	return filepath.Base(link), nil
}

func scanProject(scanner interface {
	Scan(dest ...interface{}) error
}) (*types.ProjectInfo, error) {
	var (
		project       types.ProjectInfo
		profileJSON   string
		deployedAt    sql.NullString
		lastStartedAt sql.NullString
	)

	if err := scanner.Scan(
		&project.ID,
		&project.Name,
		&project.Source,
		&project.CurrentVersion,
		&project.Status,
		&profileJSON,
		&deployedAt,
		&lastStartedAt,
	); err != nil {
		return nil, err
	}

	if profileJSON != "" {
		if err := json.Unmarshal([]byte(profileJSON), &project.ConnectionProfile); err != nil {
			return nil, err
		}
	}
	if deployedAt.Valid {
		if parsed, err := time.Parse(time.RFC3339Nano, deployedAt.String); err == nil {
			project.DeployedAt = &parsed
		}
	}
	if lastStartedAt.Valid {
		if parsed, err := time.Parse(time.RFC3339Nano, lastStartedAt.String); err == nil {
			project.LastStartedAt = &parsed
		}
	}
	if project.Source == "" {
		project.Source = types.ProjectSourceLocal
	}

	return &project, nil
}

func timeToNullableRFC3339(t *time.Time) interface{} {
	if t == nil || t.IsZero() {
		return nil
	}
	return t.UTC().Format(time.RFC3339Nano)
}
