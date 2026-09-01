package hostd

import (
	"strings"
	"testing"
)

const (
	testEnvironmentID = "8f2d5eca-6c78-4e42-9a1b-7f98bda7192c"
	testNodeID        = "5f0c6c9c-d3f1-4c08-a5b2-8731f22136ac"
	testToken         = "f566ad3bbbd9d00c39a2ea05b49032fb7b231c2e5607d279"
)

func validPlan() ClusterPlan {
	return ClusterPlan{
		SchemaVersion: "induforge.cluster-plan.v1",
		Generation:    1,
		ClusterID:     testEnvironmentID,
		NodeID:        testNodeID,
		Operation:     OperationInitServer,
		K3sVersion:    K3sVersion,
		Token:         testToken,
		NodeIP:        "172.16.125.129",
		DataDir:       "/var/lib/induforge/k3s",
		APIPort:       6443,
		VXLANPort:     8472,
		KubeletPort:   10250,
	}
}

func TestClusterPlanValidation(t *testing.T) {
	if err := validPlan().Validate(); err != nil {
		t.Fatal(err)
	}
	join := validPlan()
	join.Operation = OperationJoinAgent
	join.ServerURL = "https://172.16.125.129:6443"
	if err := join.Validate(); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		mutate func(*ClusterPlan)
	}{
		{name: "unknown schema", mutate: func(p *ClusterPlan) { p.SchemaVersion = "v2" }},
		{name: "arbitrary operation", mutate: func(p *ClusterPlan) { p.Operation = "shell" }},
		{name: "different k3s", mutate: func(p *ClusterPlan) { p.K3sVersion = "latest" }},
		{name: "short token", mutate: func(p *ClusterPlan) { p.Token = "short" }},
		{name: "loopback ip", mutate: func(p *ClusterPlan) { p.NodeIP = "127.0.0.1" }},
		{name: "relative data", mutate: func(p *ClusterPlan) { p.DataDir = "data/k3s" }},
		{name: "broad data", mutate: func(p *ClusterPlan) { p.DataDir = "/var" }},
		{name: "server on init", mutate: func(p *ClusterPlan) { p.ServerURL = "https://172.16.125.129:6443" }},
		{name: "http join", mutate: func(p *ClusterPlan) { p.Operation = OperationJoinAgent; p.ServerURL = "http://172.16.125.129:6443" }},
		{name: "wrong join port", mutate: func(p *ClusterPlan) { p.Operation = OperationJoinAgent; p.ServerURL = "https://172.16.125.129:443" }},
		{name: "wrong vxlan port", mutate: func(p *ClusterPlan) { p.VXLANPort = 8473 }},
		{name: "wrong kubelet port", mutate: func(p *ClusterPlan) { p.KubeletPort = 10251 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := validPlan()
			test.mutate(&plan)
			if err := plan.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestClusterPlanSupportsConfiguredAPIPort(t *testing.T) {
	plan := validPlan()
	plan.APIPort = 16443
	if err := plan.Validate(); err != nil {
		t.Fatal(err)
	}
	if config := plan.RenderK3sConfig(); !strings.Contains(config, "https-listen-port: 16443") {
		t.Fatalf("configured API port missing from K3s config: %s", config)
	}
	plan.Operation = OperationJoinAgent
	plan.ServerURL = "https://172.16.125.129:16443"
	if err := plan.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestClusterPlanRendersOnlyFixedK3sConfiguration(t *testing.T) {
	plan := validPlan()
	config := plan.RenderK3sConfig()
	for _, expected := range []string{
		`data-dir: "/var/lib/induforge/k3s"`,
		`node-name: "if-5f0c6c9cd3f1"`,
		`node-ip: "172.16.125.129"`,
		"cluster-init: true",
		"https-listen-port: 6443",
		"secrets-encryption: true",
	} {
		if !strings.Contains(config, expected) {
			t.Fatalf("config missing %q:\n%s", expected, config)
		}
	}
	if strings.Contains(config, "server:") {
		t.Fatalf("init config must not contain server: %s", config)
	}
}
