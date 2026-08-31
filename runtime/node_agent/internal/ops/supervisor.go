// Package ops owns the NodeAgent's local, allowlisted process supervisor.
package ops

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// ServiceGroup is a fixed production allowlist. The centre cannot add a group
// or provide executable paths through a desired workload command.
type ServiceGroup string

const (
	ServiceProjectEntry ServiceGroup = "project_entry"
	ServiceDataRuntime  ServiceGroup = "data_runtime"
	ServiceCollector    ServiceGroup = "collector"
)

func validServiceGroup(group ServiceGroup) bool {
	return group == ServiceProjectEntry || group == ServiceDataRuntime || group == ServiceCollector
}

// ServiceConfig is operator-owned local configuration. It deliberately holds
// command line, environment and release identity instead of DesiredWorkload.
type ServiceConfig struct {
	Group         ServiceGroup      `yaml:"group"`
	Component     string            `yaml:"component"`
	Installed     bool              `yaml:"installed"`
	Enabled       bool              `yaml:"enabled"`
	ReleaseRoot   string            `yaml:"releaseRoot"`
	Current       string            `yaml:"current"`
	ReleaseDigest string            `yaml:"releaseDigest"`
	Executable    string            `yaml:"executable"`
	Arguments     []string          `yaml:"arguments"`
	Environment   map[string]string `yaml:"environment"`
	WorkingDir    string            `yaml:"workingDir"`
	HealthURL     string            `yaml:"healthUrl"`
	PublicURL     string            `yaml:"publicUrl"`
	HealthTimeout time.Duration     `yaml:"healthTimeout"`
	DrainTimeout  time.Duration     `yaml:"drainTimeout"`
}

type SupervisorConfig struct {
	StateDir string          `yaml:"stateDir"`
	LogDir   string          `yaml:"logDir"`
	Services []ServiceConfig `yaml:"services"`
}

type ProcessStatus struct {
	WorkloadID       string         `json:"workloadId"`
	Role             ServiceGroup   `json:"role"`
	State            string         `json:"observedStatus"`
	Generation       int64          `json:"observedGeneration"`
	ReplicasObserved int            `json:"replicasObserved"`
	PID              int            `json:"pid,omitempty"`
	ComponentPIDs    map[string]int `json:"componentPids,omitempty"`
	StartedAt        *time.Time     `json:"startedAt,omitempty"`
	StoppedAt        *time.Time     `json:"stoppedAt,omitempty"`
	LastError        string         `json:"message,omitempty"`
	LogPath          string         `json:"logPath"`
}

type managedProcess struct {
	cmd      *exec.Cmd
	process  *os.Process
	log      *os.File
	status   ProcessStatus
	children []*managedProcess
}

// Supervisor persists PID metadata beneath the node's private state root. A
// restart can inspect/stop an already-running configured child, but never
// adopts a command received from the centre.
type Supervisor struct {
	mu        sync.Mutex
	stateDir  string
	logDir    string
	services  map[ServiceGroup][]ServiceConfig
	installed map[ServiceGroup][]ServiceConfig
	processes map[string]*managedProcess
	client    *http.Client
}

// NewSupervisor is kept for loopback diagnostics callers. It has no enabled
// services and therefore cannot start a production child.
func NewSupervisor(_ string, workDir, logDir string) *Supervisor {
	s, _ := NewSupervisorWithConfig(SupervisorConfig{StateDir: workDir, LogDir: logDir})
	return s
}

func NewSupervisorWithConfig(config SupervisorConfig) (*Supervisor, error) {
	if strings.TrimSpace(config.StateDir) == "" || strings.TrimSpace(config.LogDir) == "" {
		return nil, fmt.Errorf("本机进程 stateDir 和 logDir 不能为空")
	}
	stateDir, err := filepath.Abs(config.StateDir)
	if err != nil {
		return nil, err
	}
	logDir, err := filepath.Abs(config.LogDir)
	if err != nil {
		return nil, err
	}
	s := &Supervisor{stateDir: stateDir, logDir: logDir, services: make(map[ServiceGroup][]ServiceConfig), installed: make(map[ServiceGroup][]ServiceConfig), processes: make(map[string]*managedProcess), client: &http.Client{Timeout: 2 * time.Second}}
	for _, service := range config.Services {
		service.Group = ServiceGroup(strings.TrimSpace(string(service.Group)))
		if err := validateServiceTemplate(service); err != nil {
			return nil, err
		}
		service.Installed = service.Installed || service.Enabled
		if service.Installed {
			for _, existing := range s.installed[service.Group] {
				if existing.Component == service.Component {
					return nil, fmt.Errorf("服务组 %s component 重复: %s", service.Group, service.Component)
				}
			}
			s.installed[service.Group] = append(s.installed[service.Group], service)
		}
		if !service.Enabled {
			continue
		}
		if err := validateServiceConfig(service); err != nil {
			return nil, err
		}
		if service.Current == "" {
			service.Current = "current"
		}
		if service.HealthTimeout <= 0 {
			service.HealthTimeout = 10 * time.Second
		}
		if service.DrainTimeout <= 0 {
			service.DrainTimeout = 20 * time.Second
		}
		s.services[service.Group] = append(s.services[service.Group], service)
	}
	if err := s.recover(); err != nil {
		return nil, err
	}
	return s, nil
}

func validateServiceTemplate(service ServiceConfig) error {
	if !validServiceGroup(service.Group) {
		return fmt.Errorf("不支持的本地服务组: %s", service.Group)
	}
	if strings.TrimSpace(service.Component) == "" {
		return fmt.Errorf("服务组 %s 缺少 component", service.Group)
	}
	return validatePublicURL(service)
}

func validateServiceConfig(service ServiceConfig) error {
	if err := validateServiceTemplate(service); err != nil {
		return err
	}
	if strings.TrimSpace(service.ReleaseRoot) == "" || strings.TrimSpace(service.Executable) == "" || strings.TrimSpace(service.HealthURL) == "" {
		return fmt.Errorf("服务组 %s 缺少 releaseRoot、executable 或 healthUrl", service.Group)
	}
	if !strings.HasPrefix(service.ReleaseDigest, "sha256:") || len(service.ReleaseDigest) != 71 {
		return fmt.Errorf("服务组 %s releaseDigest 必须是 sha256", service.Group)
	}
	if !safeRelativePath(service.Current) || !safeRelativePath(service.Executable) || (service.WorkingDir != "" && !safeRelativePath(service.WorkingDir)) {
		return fmt.Errorf("服务组 %s 的 release 路径非法", service.Group)
	}
	request, err := http.NewRequest(http.MethodGet, service.HealthURL, nil)
	if err != nil || request.URL.Scheme != "http" || !isLoopbackHost(request.URL.Hostname()) {
		return fmt.Errorf("服务组 %s healthUrl 非法", service.Group)
	}
	if err := validatePublicURL(service); err != nil {
		return err
	}
	return nil
}

// validatePublicURL 只允许 project-gateway 把本机运维配置的正式入口上报给中心。
// 中心不会下发 URL，也不会根据节点 IP 和端口推断入口。
func validatePublicURL(service ServiceConfig) error {
	if service.PublicURL == "" {
		return nil
	}
	if service.Group != ServiceProjectEntry || service.Component != "project-gateway" {
		return fmt.Errorf("只有 project_entry/project-gateway 可以配置 publicUrl")
	}
	parsed, err := url.Parse(service.PublicURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
		return fmt.Errorf("project-gateway publicUrl 必须为不含 userinfo 或 fragment 的 http/https URL")
	}
	return nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func safeRelativePath(value string) bool {
	if value == "" || filepath.IsAbs(value) || strings.ContainsRune(value, '\x00') {
		return false
	}
	clean := filepath.Clean(value)
	return clean != "." && clean != ".." && !strings.HasPrefix(clean, ".."+string(filepath.Separator))
}

func (s *Supervisor) Start(workloadID string, group ServiceGroup, generation int64) (ProcessStatus, error) {
	if err := validateWorkload(workloadID, group); err != nil {
		return ProcessStatus{}, err
	}
	services, enabled := s.services[group]
	if !enabled {
		return ProcessStatus{}, fmt.Errorf("本节点未显式分配服务组: %s", group)
	}
	s.mu.Lock()
	if existing := s.processes[workloadID]; existing != nil && existing.process != nil && processAlive(existing.process) {
		if generation > existing.status.Generation {
			existing.status.Generation = generation
			_ = s.saveLocked()
		}
		status := withObservedReplicas(existing.status)
		s.mu.Unlock()
		return status, nil
	}
	s.mu.Unlock()
	now := time.Now().UTC()
	process := &managedProcess{status: ProcessStatus{WorkloadID: workloadID, Role: group, State: "starting", Generation: generation, StartedAt: &now, LogPath: s.logPath(workloadID), ComponentPIDs: map[string]int{}}}
	for _, service := range services {
		child, err := s.startComponent(workloadID, service)
		if err != nil {
			for _, started := range process.children {
				_ = terminateChild(started.process)
			}
			return ProcessStatus{}, err
		}
		process.children = append(process.children, child)
		process.status.ComponentPIDs[service.Component] = child.process.Pid
		if process.process == nil {
			process.cmd, process.process, process.log, process.status.PID = child.cmd, child.process, child.log, child.process.Pid
		}
	}
	s.mu.Lock()
	s.processes[workloadID] = process
	_ = s.saveLocked()
	s.mu.Unlock()
	for _, child := range process.children {
		go s.reapComponent(workloadID, process, child)
	}
	for index, child := range process.children {
		if err := s.waitHealthy(child, services[index]); err != nil {
			_, _ = s.Stop(workloadID, generation)
			s.RecordFailure(workloadID, group, generation, err)
			return ProcessStatus{}, err
		}
	}
	s.mu.Lock()
	if s.processes[workloadID] == process {
		process.status.State, process.status.LastError = "running", ""
		_ = s.saveLocked()
	}
	status := withObservedReplicas(process.status)
	s.mu.Unlock()
	return status, nil
}

func (s *Supervisor) startComponent(workloadID string, service ServiceConfig) (*managedProcess, error) {
	executable, workingDir, err := resolveReleaseService(service)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(s.logDir, 0755); err != nil {
		return nil, err
	}
	logPath := s.componentLogPath(workloadID, service.Component)
	if err := rotateLog(logPath, 5<<20); err != nil {
		return nil, err
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(executable, service.Arguments...)
	cmd.Dir, cmd.Env, cmd.Stdout, cmd.Stderr = workingDir, configuredEnvironment(service.Environment), logFile, logFile
	prepareChild(cmd)
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return nil, fmt.Errorf("启动 %s/%s 失败: %w", workloadID, service.Component, err)
	}
	return &managedProcess{cmd: cmd, process: cmd.Process, log: logFile}, nil
}

func configuredEnvironment(values map[string]string) []string {
	env := os.Environ()
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if key != "" && !strings.Contains(key, "=") && !strings.ContainsRune(key, '\x00') && !strings.ContainsRune(values[key], '\x00') {
			env = append(env, key+"="+values[key])
		}
	}
	return env
}

func (s *Supervisor) waitHealthy(process *managedProcess, service ServiceConfig) error {
	deadline := time.NewTimer(service.HealthTimeout)
	defer deadline.Stop()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		if process.process == nil || !processAlive(process.process) {
			return fmt.Errorf("服务组 %s 在健康检查前退出", service.Group)
		}
		if healthy, _ := s.probe(service.HealthURL); healthy {
			return nil
		}
		select {
		case <-deadline.C:
			return fmt.Errorf("服务组 %s 健康检查超时", service.Group)
		case <-ticker.C:
		}
	}
}

func (s *Supervisor) probe(url string) (bool, error) {
	response, err := s.client.Get(url)
	if err != nil {
		return false, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return false, nil
	}
	var envelope struct {
		Code int `json:"code"`
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		return false, err
	}
	return envelope.Code == 0 && envelope.Data.Status == "UP", nil
}

func (s *Supervisor) reapComponent(workloadID string, parent, child *managedProcess) {
	if child.cmd == nil {
		return
	}
	err := child.cmd.Wait()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.processes[workloadID] != parent {
		if child.log != nil {
			_ = child.log.Close()
		}
		return
	}
	if child.log != nil {
		_ = child.log.Close()
		child.log = nil
	}
	if parent.status.State != "stopping" && parent.status.State != "failed" {
		parent.status.State = "failed"
		if err != nil {
			parent.status.LastError = err.Error()
		} else {
			parent.status.LastError = "组件提前退出"
		}
	}
	allStopped := true
	for _, item := range parent.children {
		if item.process != nil && processAlive(item.process) {
			allStopped = false
			break
		}
	}
	if allStopped {
		now := time.Now().UTC()
		parent.status.PID, parent.status.StoppedAt = 0, &now
		if parent.status.State == "stopping" {
			parent.status.State, parent.status.LastError = "stopped", ""
		}
	}
	_ = s.saveLocked()
}

func (s *Supervisor) Stop(workloadID string, generation int64) (ProcessStatus, error) {
	s.mu.Lock()
	process := s.processes[workloadID]
	if process == nil || process.process == nil || !processAlive(process.process) {
		now := time.Now().UTC()
		if process == nil {
			process = &managedProcess{status: ProcessStatus{WorkloadID: workloadID, State: "stopped", Generation: generation, StoppedAt: &now, LogPath: s.logPath(workloadID)}}
			s.processes[workloadID] = process
		} else {
			if generation > process.status.Generation {
				process.status.Generation = generation
			}
			process.status.State, process.status.PID, process.status.LastError, process.status.StoppedAt = "stopped", 0, "", &now
		}
		_ = s.saveLocked()
		status := withObservedReplicas(process.status)
		s.mu.Unlock()
		return status, nil
	}
	services, enabled := s.services[process.status.Role]
	if !enabled {
		s.mu.Unlock()
		return ProcessStatus{}, fmt.Errorf("服务组未配置: %s", process.status.Role)
	}
	process.status.State = "stopping"
	if generation > process.status.Generation {
		process.status.Generation = generation
	}
	children := process.children
	if len(children) == 0 {
		children = []*managedProcess{process}
	}
	_ = s.saveLocked()
	s.mu.Unlock()
	for _, child := range children {
		if err := terminateChild(child.process); err != nil && !errors.Is(err, os.ErrProcessDone) {
			return ProcessStatus{}, err
		}
	}
	drainTimeout := 20 * time.Second
	for _, service := range services {
		if service.DrainTimeout > drainTimeout {
			drainTimeout = service.DrainTimeout
		}
	}
	deadline := time.NewTimer(drainTimeout)
	defer deadline.Stop()
	for {
		s.mu.Lock()
		status := s.statusLocked(workloadID)
		if live := anyChildAlive(process); !live && (status.State == "running" || status.State == "starting" || status.State == "stopping") {
			now := time.Now().UTC()
			process.status.State, process.status.PID, process.status.StoppedAt, process.status.LastError = "stopped", 0, &now, ""
			_ = s.saveLocked()
			status = withObservedReplicas(process.status)
		}
		stopped := status.State != "running" && status.State != "starting" && status.State != "stopping"
		s.mu.Unlock()
		if stopped {
			return status, nil
		}
		select {
		case <-deadline.C:
			for _, child := range children {
				_ = child.process.Kill()
			}
		case <-time.After(25 * time.Millisecond):
		}
	}
}

func anyChildAlive(process *managedProcess) bool {
	children := process.children
	if len(children) == 0 {
		children = []*managedProcess{process}
	}
	for _, child := range children {
		if child.process != nil && processAlive(child.process) {
			return true
		}
	}
	return false
}

func (s *Supervisor) Restart(workloadID string, group ServiceGroup, generation int64) (ProcessStatus, error) {
	if _, err := s.Stop(workloadID, generation); err != nil {
		return ProcessStatus{}, err
	}
	return s.Start(workloadID, group, generation)
}
func (s *Supervisor) Status(workloadID string) (ProcessStatus, error) {
	if workloadID == "" {
		return ProcessStatus{}, fmt.Errorf("workloadId 不能为空")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.statusLocked(workloadID), nil
}

func (s *Supervisor) HasService(group ServiceGroup) bool { return len(s.services[group]) > 0 }

// Capabilities 从本机已安装的 ServiceConfig 派生；未配置 release 的模板可以接入，
// 但不能启动，避免把未部署工程伪报为 running。
func (s *Supervisor) Capabilities() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	capabilities := make([]string, 0, len(s.installed))
	for _, group := range []ServiceGroup{ServiceProjectEntry, ServiceDataRuntime, ServiceCollector} {
		if len(s.installed[group]) > 0 {
			capabilities = append(capabilities, string(group))
		}
	}
	return capabilities
}

// PublicURL 返回 project-gateway 的本机配置地址。其他服务永远没有对外入口。
func (s *Supervisor) PublicURL(group ServiceGroup) string {
	if group != ServiceProjectEntry {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, service := range s.services[group] {
		if service.Component == "project-gateway" {
			return service.PublicURL
		}
	}
	return ""
}
func (s *Supervisor) List() []ProcessStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]ProcessStatus, 0, len(s.processes))
	for id := range s.processes {
		result = append(result, s.statusLocked(id))
	}
	return result
}
func (s *Supervisor) statusLocked(workloadID string) ProcessStatus {
	if process := s.processes[workloadID]; process != nil {
		if process.status.PID > 0 && process.process != nil && !processAlive(process.process) && (process.status.State == "running" || process.status.State == "starting") {
			now := time.Now().UTC()
			process.status.State, process.status.PID, process.status.StoppedAt = "failed", 0, &now
			_ = s.saveLocked()
		}
		return withObservedReplicas(process.status)
	}
	return withObservedReplicas(ProcessStatus{WorkloadID: workloadID, State: "stopped", LogPath: s.logPath(workloadID)})
}
func withObservedReplicas(status ProcessStatus) ProcessStatus {
	if status.State == "running" || status.State == "starting" || status.State == "stopping" {
		status.ReplicasObserved = 1
	} else {
		status.ReplicasObserved = 0
	}
	return status
}

func (s *Supervisor) Logs(workloadID string, maxLines int) ([]string, error) {
	if workloadID == "" {
		return nil, fmt.Errorf("workloadId 不能为空")
	}
	if maxLines <= 0 || maxLines > 500 {
		maxLines = 200
	}
	file, err := os.Open(s.logPath(workloadID))
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	lines := make([]string, 0, maxLines)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > maxLines {
			lines = lines[1:]
		}
	}
	return lines, scanner.Err()
}
func (s *Supervisor) Shutdown() {
	s.mu.Lock()
	ids := make([]string, 0, len(s.processes))
	for id := range s.processes {
		ids = append(ids, id)
	}
	s.mu.Unlock()
	for _, id := range ids {
		_, _ = s.Stop(id, 0)
	}
}
func (s *Supervisor) RecordFailure(workloadID string, group ServiceGroup, generation int64, err error) {
	if workloadID == "" || !validServiceGroup(group) || err == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	process := s.processes[workloadID]
	if process == nil {
		process = &managedProcess{status: ProcessStatus{WorkloadID: workloadID, Role: group, LogPath: s.logPath(workloadID)}}
		s.processes[workloadID] = process
	}
	if generation >= process.status.Generation {
		process.status.Generation, process.status.State, process.status.LastError, process.status.PID = generation, "failed", err.Error(), 0
		_ = s.saveLocked()
	}
}
func (s *Supervisor) logPath(workloadID string) string {
	sum := sha256.Sum256([]byte(workloadID))
	return filepath.Join(s.logDir, fmt.Sprintf("workload-%x.log", sum[:8]))
}
func (s *Supervisor) componentLogPath(workloadID, component string) string {
	sum := sha256.Sum256([]byte(workloadID + "\x00" + component))
	return filepath.Join(s.logDir, fmt.Sprintf("component-%x.log", sum[:8]))
}
func validateWorkload(workloadID string, group ServiceGroup) error {
	if strings.TrimSpace(workloadID) == "" {
		return fmt.Errorf("workloadId 不能为空")
	}
	if !validServiceGroup(group) {
		return fmt.Errorf("不支持的服务组: %s", group)
	}
	return nil
}
func rotateLog(path string, maxBytes int64) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil || info.Size() < maxBytes {
		return err
	}
	return os.Rename(path, path+".1")
}

// resolveReleaseService enforces immutable release/current selection. The
// configured digest is SHA-256(release-manifest.json); a later release
// downloader must populate this file after signature verification.
func resolveReleaseService(service ServiceConfig) (string, string, error) {
	root, err := filepath.Abs(service.ReleaseRoot)
	if err != nil {
		return "", "", err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", "", fmt.Errorf("服务组 %s releaseRoot 不可解析: %w", service.Group, err)
	}
	release, err := filepath.EvalSymlinks(filepath.Join(root, service.Current))
	if err != nil {
		return "", "", fmt.Errorf("服务组 %s current 不可解析: %w", service.Group, err)
	}
	if !within(root, release) {
		return "", "", fmt.Errorf("服务组 %s current 越出 releaseRoot", service.Group)
	}
	info, err := os.Stat(release)
	if err != nil || !info.IsDir() {
		return "", "", fmt.Errorf("服务组 %s current 不是 release 目录", service.Group)
	}
	manifest, err := os.ReadFile(filepath.Join(release, "release-manifest.json"))
	if err != nil {
		return "", "", fmt.Errorf("服务组 %s 缺少 release-manifest.json: %w", service.Group, err)
	}
	digest := sha256.Sum256(manifest)
	if !strings.EqualFold("sha256:"+hex.EncodeToString(digest[:]), service.ReleaseDigest) {
		return "", "", fmt.Errorf("服务组 %s releaseDigest 不匹配", service.Group)
	}
	executable := filepath.Join(release, service.Executable)
	if !within(release, executable) {
		return "", "", fmt.Errorf("服务组 %s executable 越界", service.Group)
	}
	if info, err := os.Stat(executable); err != nil || info.IsDir() {
		return "", "", fmt.Errorf("服务组 %s executable 不存在", service.Group)
	}
	workingDir := release
	if service.WorkingDir != "" {
		workingDir = filepath.Join(release, service.WorkingDir)
	}
	if !within(release, workingDir) {
		return "", "", fmt.Errorf("服务组 %s workingDir 越界", service.Group)
	}
	if info, err := os.Stat(workingDir); err != nil || !info.IsDir() {
		return "", "", fmt.Errorf("服务组 %s workingDir 不存在", service.Group)
	}
	return executable, workingDir, nil
}
func within(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}
func (s *Supervisor) statePath() string { return filepath.Join(s.stateDir, "managed-processes.json") }
func (s *Supervisor) recover() error {
	data, err := os.ReadFile(s.statePath())
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var statuses []ProcessStatus
	if err := json.Unmarshal(data, &statuses); err != nil {
		return fmt.Errorf("读取进程状态失败: %w", err)
	}
	for _, status := range statuses {
		if !validServiceGroup(status.Role) {
			continue
		}
		if services := s.services[status.Role]; len(services) == 0 {
			continue
		}
		if status.PID > 0 && (status.State == "running" || status.State == "starting" || status.State == "stopping") {
			parent := &managedProcess{status: status}
			pids := status.ComponentPIDs
			if len(pids) == 0 {
				pids = map[string]int{"legacy": status.PID}
			}
			allAlive := true
			for _, pid := range pids {
				process, findErr := os.FindProcess(pid)
				if findErr != nil || !processAlive(process) {
					allAlive = false
					break
				}
				child := &managedProcess{process: process}
				parent.children = append(parent.children, child)
				if parent.process == nil {
					parent.process = process
				}
			}
			if allAlive && len(parent.children) > 0 {
				s.processes[status.WorkloadID] = parent
				continue
			}
		}
		now := time.Now().UTC()
		status.State, status.PID, status.StoppedAt = "failed", 0, &now
		s.processes[status.WorkloadID] = &managedProcess{status: status}
	}
	return s.saveLocked()
}
func (s *Supervisor) saveLocked() error {
	if err := os.MkdirAll(s.stateDir, 0700); err != nil {
		return err
	}
	statuses := make([]ProcessStatus, 0, len(s.processes))
	for _, process := range s.processes {
		statuses = append(statuses, process.status)
	}
	sort.Slice(statuses, func(i, j int) bool { return statuses[i].WorkloadID < statuses[j].WorkloadID })
	data, err := json.Marshal(statuses)
	if err != nil {
		return err
	}
	temporary := s.statePath() + ".tmp"
	if err := os.WriteFile(temporary, data, 0600); err != nil {
		return err
	}
	return os.Rename(temporary, s.statePath())
}
