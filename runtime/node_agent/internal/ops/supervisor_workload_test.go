package ops

import (
	"sync"
	"testing"
)

func TestSupervisorConcurrentStartHasOneProcess(t *testing.T) {
	s := configuredSupervisor(t, ServiceCollector)
	t.Cleanup(s.Shutdown)
	var wg sync.WaitGroup
	statuses := make(chan ProcessStatus, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			status, err := s.Start("same-collector", ServiceCollector, 1)
			if err != nil {
				t.Error(err)
				return
			}
			statuses <- status
		}()
	}
	wg.Wait()
	close(statuses)
	pid := 0
	for status := range statuses {
		if pid != 0 && status.PID != pid {
			t.Errorf("并发启动产生多个进程: %d, %d", pid, status.PID)
		}
		pid = status.PID
	}
}

func TestSupervisorIsolatesDeploymentConfigurationAndRecovers(t *testing.T) {
	s := configuredSupervisor(t, ServiceCollector)
	t.Cleanup(s.Shutdown)
	first := s.services[ServiceCollector][0]
	second := first
	second.Environment = map[string]string{"COLLECTOR_DEPLOYMENT": "second"}
	a, err := s.ActivateWorkload("deployment-a", ServiceCollector, 1, []ServiceConfig{first}, false)
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.ActivateWorkload("deployment-b", ServiceCollector, 1, []ServiceConfig{second}, false)
	if err != nil {
		t.Fatal(err)
	}
	if a.PID == b.PID {
		t.Fatal("不同部署共享了进程")
	}
	recovered, err := NewSupervisorWithConfig(SupervisorConfig{StateDir: s.stateDir, LogDir: s.logDir, Services: []ServiceConfig{{Group: ServiceCollector, Component: first.Component, Installed: true}}})
	if err != nil {
		t.Fatal(err)
	}
	if recovered.workloads["deployment-b"][0].Environment["COLLECTOR_DEPLOYMENT"] != "second" {
		t.Fatal("重启丢失部署配置")
	}
	status, err := recovered.Status("deployment-b")
	if err != nil || status.PID != b.PID {
		t.Fatalf("恢复进程失败: %+v %v", status, err)
	}
	if _, err := s.Stop("deployment-a", 2); err != nil {
		t.Fatal(err)
	}
	status, _ = s.Status("deployment-b")
	if status.State != "running" || status.PID != b.PID {
		t.Fatal("停止一个部署影响另一部署")
	}
}

func TestSupervisorCandidateFailureRestoresPreviousProcess(t *testing.T) {
	s := configuredSupervisor(t, ServiceCollector)
	t.Cleanup(s.Shutdown)
	service := s.services[ServiceCollector][0]
	old, err := s.ActivateWorkload("update", ServiceCollector, 1, []ServiceConfig{service}, false)
	if err != nil {
		t.Fatal(err)
	}
	candidate := service
	candidate.Arguments = []string{"-test.run=^$"}
	candidate.HealthURL = "http://127.0.0.1:1/health"
	candidate.HealthTimeout = 100000000
	restored, err := s.ActivateWorkload("update", ServiceCollector, 2, []ServiceConfig{candidate}, true)
	if err == nil || restored.State != "running" || restored.Generation != 1 || restored.PID == old.PID {
		t.Fatalf("失败恢复错误: %+v %v", restored, err)
	}
	s.RecordFailure("update", ServiceCollector, 2, err)
	status, _ := s.Status("update")
	if status.PID != restored.PID || status.Generation != 1 || status.State != "running" {
		t.Fatalf("失败上报抹掉仍在运行的旧进程: %+v", status)
	}
}

func TestSupervisorActivationIsIdempotentAndRejectsUninstalledComponent(t *testing.T) {
	s := configuredSupervisor(t, ServiceCollector)
	t.Cleanup(s.Shutdown)
	service := s.services[ServiceCollector][0]
	service.Environment = map[string]string{"DEPLOYMENT": "original"}
	first, err := s.ActivateWorkload("idempotent", ServiceCollector, 2, []ServiceConfig{service}, true)
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := s.ActivateWorkload("idempotent", ServiceCollector, 2, []ServiceConfig{service}, true)
	if err != nil || repeated.PID != first.PID {
		t.Fatalf("同一代次重复重启: %+v %v", repeated, err)
	}
	service.Environment["DEPLOYMENT"] = "mutated"
	if s.workloads["idempotent"][0].Environment["DEPLOYMENT"] != "original" {
		t.Fatal("调用者修改了已保存配置")
	}
	service.Component = "not-installed"
	if _, err := s.ActivateWorkload("idempotent", ServiceCollector, 3, []ServiceConfig{service}, true); err == nil {
		t.Fatal("接受未安装组件")
	}
	status, _ := s.Status("idempotent")
	if status.PID != first.PID {
		t.Fatal("预检失败停止了现有进程")
	}
}

func TestSupervisorStaleStopCannotStopNewGeneration(t *testing.T) {
	s := configuredSupervisor(t, ServiceCollector)
	t.Cleanup(s.Shutdown)
	started, err := s.Start("generation", ServiceCollector, 3)
	if err != nil {
		t.Fatal(err)
	}
	status, err := s.Stop("generation", 2)
	if err != nil || status.State != "running" || status.Generation != 3 || status.PID != started.PID {
		t.Fatalf("过期停止命令影响新版本: %+v %v", status, err)
	}
}
