// Package ops 提供 NodeAgent 的运维控制面适配和本机原生进程托管能力。
package ops

import (
	"bufio"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

type WorkloadRole string

const (
	WorkloadCompute   WorkloadRole = "compute"
	WorkloadAlert     WorkloadRole = "alert"
	WorkloadCollector WorkloadRole = "collector"
)

func validRole(role WorkloadRole) bool {
	return role == WorkloadCompute || role == WorkloadAlert || role == WorkloadCollector
}

// ProcessStatus 是每个 workload 实例的观测事实。role 不是 map key，避免同角色多工程互相覆盖。
type ProcessStatus struct {
	WorkloadID       string       `json:"workloadId"`
	Role             WorkloadRole `json:"role"`
	State            string       `json:"observedStatus"`
	Generation       int64        `json:"observedGeneration"`
	ReplicasObserved int          `json:"replicasObserved"`
	PID              int          `json:"pid,omitempty"`
	StartedAt        *time.Time   `json:"startedAt,omitempty"`
	StoppedAt        *time.Time   `json:"stoppedAt,omitempty"`
	LastError        string       `json:"message,omitempty"`
	LogPath          string       `json:"logPath"`
}

type managedProcess struct {
	cmd    *exec.Cmd
	log    *os.File
	status ProcessStatus
}

// Supervisor 管理本机原生进程。启动不绑定 HTTP request context；每个子进程单独 Wait 回收 PID。
type Supervisor struct {
	mu                   sync.Mutex
	bin, workDir, logDir string
	processes            map[string]*managedProcess
	command              func(WorkloadRole, string) *exec.Cmd
}

func NewSupervisor(bin, workDir, logDir string) *Supervisor {
	s := &Supervisor{bin: bin, workDir: workDir, logDir: logDir, processes: make(map[string]*managedProcess)}
	s.command = func(role WorkloadRole, workloadID string) *exec.Cmd {
		return exec.Command(s.bin, "demo-workload", string(role), workloadID)
	}
	return s
}

func (s *Supervisor) Start(workloadID string, role WorkloadRole, generation int64) (ProcessStatus, error) {
	if err := validateWorkload(workloadID, role); err != nil {
		return ProcessStatus{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing := s.processes[workloadID]; existing != nil && existing.cmd != nil && existing.cmd.Process != nil && processAlive(existing.cmd.Process) {
		if generation > existing.status.Generation {
			existing.status.Generation = generation
		}
		return withObservedReplicas(existing.status), nil
	}
	if err := os.MkdirAll(s.workDir, 0755); err != nil {
		return ProcessStatus{}, err
	}
	if err := os.MkdirAll(s.logDir, 0755); err != nil {
		return ProcessStatus{}, err
	}
	logPath := s.logPath(workloadID)
	if err := rotateLog(logPath, 5<<20); err != nil {
		return ProcessStatus{}, err
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return ProcessStatus{}, err
	}
	cmd := s.command(role, workloadID)
	cmd.Dir = s.workDir
	cmd.Stdout, cmd.Stderr = logFile, logFile
	prepareChild(cmd)
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return ProcessStatus{}, fmt.Errorf("启动 %s 失败: %w", workloadID, err)
	}
	now := time.Now().UTC()
	process := &managedProcess{cmd: cmd, log: logFile, status: ProcessStatus{WorkloadID: workloadID, Role: role, State: "running", Generation: generation, PID: cmd.Process.Pid, StartedAt: &now, LogPath: logPath}}
	s.processes[workloadID] = process
	go s.reap(workloadID, process)
	return withObservedReplicas(process.status), nil
}

func (s *Supervisor) reap(workloadID string, process *managedProcess) {
	err := process.cmd.Wait()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.processes[workloadID] != process {
		if process.log != nil {
			_ = process.log.Close()
		}
		return
	}
	now := time.Now().UTC()
	process.status.PID, process.status.StoppedAt = 0, &now
	if process.status.State == "stopping" {
		process.status.State, process.status.LastError = "stopped", ""
	} else if err != nil {
		process.status.State, process.status.LastError = "failed", err.Error()
	} else {
		process.status.State = "stopped"
	}
	if process.log != nil {
		_ = process.log.Close()
		process.log = nil
	}
}

func (s *Supervisor) Stop(workloadID string, generation int64) (ProcessStatus, error) {
	s.mu.Lock()
	process := s.processes[workloadID]
	if process == nil || process.cmd == nil || process.cmd.Process == nil || !processAlive(process.cmd.Process) {
		now := time.Now().UTC()
		if process == nil {
			process = &managedProcess{status: ProcessStatus{WorkloadID: workloadID, State: "stopped", Generation: generation, StoppedAt: &now, LogPath: s.logPath(workloadID)}}
			s.processes[workloadID] = process
		} else {
			if generation > process.status.Generation {
				process.status.Generation = generation
			}
			if process.status.State != "stopped" || process.status.StoppedAt == nil {
				process.status.StoppedAt = &now
			}
			process.status.State = "stopped"
			process.status.PID = 0
			process.status.LastError = ""
		}
		status := withObservedReplicas(process.status)
		s.mu.Unlock()
		return status, nil
	}
	process.status.State = "stopping"
	if generation > process.status.Generation {
		process.status.Generation = generation
	}
	proc := process.cmd.Process
	s.mu.Unlock()
	if err := terminateChild(proc); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return ProcessStatus{}, err
	}
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	for {
		s.mu.Lock()
		status := s.statusLocked(workloadID)
		stopped := status.State != "running" && status.State != "stopping"
		s.mu.Unlock()
		if stopped {
			return status, nil
		}
		select {
		case <-deadline.C:
			_ = proc.Kill()
		case <-time.After(25 * time.Millisecond):
		}
	}
}

func (s *Supervisor) Restart(workloadID string, role WorkloadRole, generation int64) (ProcessStatus, error) {
	if _, err := s.Stop(workloadID, generation); err != nil {
		return ProcessStatus{}, err
	}
	return s.Start(workloadID, role, generation)
}
func (s *Supervisor) Status(workloadID string) (ProcessStatus, error) {
	if workloadID == "" {
		return ProcessStatus{}, fmt.Errorf("workloadId 不能为空")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.statusLocked(workloadID), nil
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
		return withObservedReplicas(process.status)
	}
	return withObservedReplicas(ProcessStatus{WorkloadID: workloadID, State: "stopped", LogPath: s.logPath(workloadID)})
}

// withObservedReplicas 将本机单进程 supervisor 的事实转换为控制面副本数。
// STOPPING 进程仍存活，故应继续报告一个观测副本。
func withObservedReplicas(status ProcessStatus) ProcessStatus {
	if status.State == "running" || status.State == "stopping" {
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

// RecordFailure 将调和失败作为 observed state 留存，下一次更高 generation 仍可安全重试。
func (s *Supervisor) RecordFailure(workloadID string, role WorkloadRole, generation int64, err error) {
	if workloadID == "" || !validRole(role) || err == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	process := s.processes[workloadID]
	if process == nil {
		process = &managedProcess{status: ProcessStatus{WorkloadID: workloadID, Role: role, LogPath: s.logPath(workloadID)}}
		s.processes[workloadID] = process
	}
	if generation >= process.status.Generation {
		process.status.Generation = generation
		process.status.State = "failed"
		process.status.LastError = err.Error()
		process.status.PID = 0
	}
}
func (s *Supervisor) logPath(workloadID string) string {
	sum := sha256.Sum256([]byte(workloadID))
	return filepath.Join(s.logDir, fmt.Sprintf("workload-%x.log", sum[:8]))
}
func validateWorkload(workloadID string, role WorkloadRole) error {
	if workloadID == "" {
		return fmt.Errorf("workloadId 不能为空")
	}
	if !validRole(role) {
		return fmt.Errorf("不支持的工作负载角色: %s", role)
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
