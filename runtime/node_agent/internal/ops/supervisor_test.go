package ops

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestSupervisorHelperProcess(t *testing.T) {
	if !strings.Contains(strings.Join(os.Args, " "), "demo-workload") {
		return
	}
	fmt.Println("demo workload started")
	for {
		time.Sleep(time.Second)
	}
}

func testSupervisor(t *testing.T) *Supervisor {
	s := NewSupervisor(os.Args[0], t.TempDir(), t.TempDir())
	s.command = func(role WorkloadRole, id string) *exec.Cmd {
		return exec.Command(os.Args[0], "-test.run=TestSupervisorHelperProcess", "--", "demo-workload", string(role), id)
	}
	return s
}

func TestSupervisorStartStopAndReap(t *testing.T) {
	s := testSupervisor(t)
	started, err := s.Start("workload-a", WorkloadCollector, 2)
	if err != nil {
		t.Fatal(err)
	}
	if started.State != "running" || started.PID <= 0 || started.Generation != 2 {
		t.Fatalf("unexpected: %+v", started)
	}
	deadline := time.Now().Add(time.Second)
	for {
		logs, _ := s.Logs("workload-a", 10)
		if len(logs) > 0 || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	stopped, err := s.Stop("workload-a", 3)
	if err != nil {
		t.Fatal(err)
	}
	if stopped.State != "stopped" || stopped.Generation != 3 {
		t.Fatalf("unexpected: %+v", stopped)
	}
	logs, err := s.Logs("workload-a", 10)
	if err != nil || len(logs) == 0 {
		t.Fatalf("logs=%v err=%v", logs, err)
	}
}

func TestSupervisorUsesWorkloadIDInsteadOfRole(t *testing.T) {
	s := testSupervisor(t)
	if _, err := s.Start("a", WorkloadCollector, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Start("b", WorkloadCollector, 1); err != nil {
		t.Fatal(err)
	}
	if len(s.List()) != 2 {
		t.Fatalf("expected separate same-role workloads: %+v", s.List())
	}
	s.Shutdown()
}

func TestSupervisorRejectsUnknownRole(t *testing.T) {
	if _, err := testSupervisor(t).Start("a", "alarm", 1); err == nil {
		t.Fatal("expected role error")
	}
}

func TestSupervisorReportsObservedReplicasFromProcessState(t *testing.T) {
	s := testSupervisor(t)
	started, err := s.Start("replica-test", WorkloadCollector, 1)
	if err != nil {
		t.Fatal(err)
	}
	if started.ReplicasObserved != 1 {
		t.Fatalf("running replicasObserved=%d", started.ReplicasObserved)
	}
	stopped, err := s.Stop("replica-test", 2)
	if err != nil {
		t.Fatal(err)
	}
	if stopped.ReplicasObserved != 0 {
		t.Fatalf("stopped replicasObserved=%d", stopped.ReplicasObserved)
	}
	failed := ProcessStatus{State: "failed"}
	if withObservedReplicas(failed).ReplicasObserved != 0 {
		t.Fatal("failed workload must report zero replicas")
	}
}
