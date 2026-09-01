package hostd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type timeSyncRunner struct{}

func (timeSyncRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	joined := name + " " + strings.Join(args, " ")
	if strings.Contains(joined, "chronyc -n tracking") {
		return []byte("Reference ID : 172.16.125.129\nStratum : 11\nSystem time : 0.006500000 seconds fast of NTP time\nLast offset : +0.001250 seconds\nLeap status : Normal\n"), nil
	}
	if strings.Contains(joined, "is-active chrony.service") {
		return []byte("active\n"), nil
	}
	return nil, nil
}

type largeOffsetRunner struct{ stepped bool }

func (runner *largeOffsetRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	joined := name + " " + strings.Join(args, " ")
	if strings.Contains(joined, "chronyc makestep") {
		runner.stepped = true
		return []byte("200 OK"), nil
	}
	if strings.Contains(joined, "chronyc -n tracking") {
		offset := "31.000000000"
		if runner.stepped {
			offset = "0.001000000"
		}
		return []byte("System time : " + offset + " seconds fast of NTP time\nLeap status : Normal\n"), nil
	}
	if strings.Contains(joined, "is-active chrony.service") {
		return []byte("active\n"), nil
	}
	return nil, nil
}

func TestTimeSyncPlanRendersOfflineCenterAndClientConfig(t *testing.T) {
	center := TimeSyncPlan{SchemaVersion: TimeSyncSchemaVersion, Generation: 1, NodeID: "11111111-1111-4111-8111-111111111111", Role: "center", CenterIP: "172.16.125.129", AllowedClients: []string{"172.16.125.130"}}
	if err := center.Validate(); err != nil {
		t.Fatal(err)
	}
	centerConfig := center.RenderChronyConfig()
	if !strings.Contains(centerConfig, "local stratum 10") || !strings.Contains(centerConfig, "allow 172.16.125.130") || strings.Contains(centerConfig, "pool ") || strings.Contains(centerConfig, "server ") {
		t.Fatalf("center config must serve only its local system clock:\n%s", centerConfig)
	}
	if !strings.Contains(centerConfig, "driftfile /var/lib/chrony/induforge-center.drift") {
		t.Fatal("center must not reuse a worker clock-frequency estimate")
	}
	if !strings.Contains(center.RenderChronyServiceDropIn(), "chronyd-starter.sh -F 1 -x") {
		t.Fatal("center chrony service must be forbidden from adjusting system time")
	}
	client := center
	client.Role, client.AllowedClients = "client", nil
	clientConfig := client.RenderChronyConfig()
	if !strings.Contains(clientConfig, "server 172.16.125.129 iburst prefer minpoll 2 maxpoll 4") || !strings.Contains(clientConfig, "makestep 1.0 -1") || strings.Contains(clientConfig, "local stratum") {
		t.Fatalf("client must follow only center:\n%s", clientConfig)
	}
	if strings.Contains(client.RenderChronyServiceDropIn(), " -x") {
		t.Fatal("client chrony service must be allowed to adjust system time")
	}
	if !strings.Contains(clientConfig, "driftfile /var/lib/chrony/induforge-client.drift") {
		t.Fatal("worker must keep an independent clock-frequency estimate")
	}
}

func TestTimeSyncStatusReportsCenterOffset(t *testing.T) {
	temporary := t.TempDir()
	chronyc := filepath.Join(temporary, "chronyc")
	if err := os.WriteFile(chronyc, []byte("test"), 0755); err != nil {
		t.Fatal(err)
	}
	manager, err := NewManager(ManagerConfig{AssetsDir: "/assets", BinaryPath: "/k3s", ConfigDir: "/config", SystemdDir: "/systemd", StateDir: temporary, Systemctl: "systemctl", Runner: timeSyncRunner{}, RuntimeArch: "arm64", ChronyConfig: filepath.Join(temporary, "chrony.conf"), ChronyDropIn: filepath.Join(temporary, "chrony.service.d", "induforge.conf"), Chronyc: chronyc})
	if err != nil {
		t.Fatal(err)
	}
	plan := TimeSyncPlan{SchemaVersion: TimeSyncSchemaVersion, Generation: 1, NodeID: "11111111-1111-4111-8111-111111111111", Role: "client", CenterIP: "172.16.125.129"}
	state, err := manager.ApplyTimeSync(context.Background(), plan)
	if err != nil {
		t.Fatal(err)
	}
	if state.Status != "synchronized" || state.OffsetMillis != 6.5 {
		t.Fatalf("unexpected state: %+v", state)
	}
}

func TestTimeSyncImmediatelyCorrectsLargeWorkerOffset(t *testing.T) {
	temporary := t.TempDir()
	chronyc := filepath.Join(temporary, "chronyc")
	if err := os.WriteFile(chronyc, []byte("test"), 0755); err != nil {
		t.Fatal(err)
	}
	runner := &largeOffsetRunner{}
	manager, err := NewManager(ManagerConfig{AssetsDir: "/assets", BinaryPath: "/k3s", ConfigDir: "/config", SystemdDir: "/systemd", StateDir: temporary, Systemctl: "systemctl", Runner: runner, RuntimeArch: "arm64", ChronyConfig: filepath.Join(temporary, "chrony.conf"), ChronyDropIn: filepath.Join(temporary, "chrony.service.d", "induforge.conf"), Chronyc: chronyc})
	if err != nil {
		t.Fatal(err)
	}
	plan := TimeSyncPlan{SchemaVersion: TimeSyncSchemaVersion, Generation: 1, NodeID: "11111111-1111-4111-8111-111111111111", Role: "client", CenterIP: "172.16.125.129"}
	state, err := manager.ApplyTimeSync(context.Background(), plan)
	if err != nil {
		t.Fatal(err)
	}
	if !runner.stepped || state.Status != "synchronized" || state.OffsetMillis != 1 {
		t.Fatalf("large offset was not corrected: %+v, stepped=%v", state, runner.stepped)
	}
}
