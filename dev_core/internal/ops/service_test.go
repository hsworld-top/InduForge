package ops

import (
	"context"
	"strings"
	"testing"

	"github.com/indu-forge/dev_core/internal/auth"
)

const (
	testProjectID = "11111111-1111-4111-8111-111111111111"
	testNodeID    = "33333333-3333-4333-8333-333333333333"
	testVersionID = "44444444-4444-4444-8444-444444444444"
)

type deploymentRepository struct {
	Repository
	validated bool
	input     CreateDeploymentInput
}

func (r *deploymentRepository) ValidateDeploymentTargets(_ context.Context, _ string, in CreateDeploymentInput) error {
	r.validated = true
	r.input = in
	return nil
}
func (r *deploymentRepository) CreateDeployment(_ context.Context, _, _ string, in CreateDeploymentInput) (ProjectDeployment, DeploymentRun, error) {
	return ProjectDeployment{ProjectID: in.ProjectID, NodeID: in.NodeID, ApplicationVersionID: in.ApplicationVersionID}, DeploymentRun{}, nil
}
func (r *deploymentRepository) OperateService(context.Context, string, string, string, string, string) (ProjectDeployment, DeploymentRun, error) {
	return ProjectDeployment{}, DeploymentRun{}, nil
}

func TestCreateDeploymentUsesOneNodeAndDerivedCollectorFlag(t *testing.T) {
	r := &deploymentRepository{}
	_, _, e := NewService(r, nil).CreateDeployment(context.Background(), auth.User{TenantID: "tenant", Role: "OPS_ADMIN"}, CreateDeploymentInput{ProjectID: testProjectID, NodeID: testNodeID, ApplicationVersionID: testVersionID, EnableCollector: true})
	if e != nil || !r.validated || r.input.NodeID != testNodeID || !r.input.EnableCollector {
		t.Fatalf("single-node deployment not validated: %#v %v", r.input, e)
	}
}
func TestCreateDeploymentRejectsMalformedNodeOrVersion(t *testing.T) {
	for _, in := range []CreateDeploymentInput{{ProjectID: testProjectID, NodeID: "bad", ApplicationVersionID: testVersionID}, {ProjectID: testProjectID, NodeID: testNodeID, ApplicationVersionID: "bad"}} {
		r := &deploymentRepository{}
		_, _, e := NewService(r, nil).CreateDeployment(context.Background(), auth.User{Role: "OPS_ADMIN"}, in)
		if e == nil || !strings.Contains(e.Error(), "格式无效") || r.validated {
			t.Fatalf("invalid input reached repository: %#v %v", in, e)
		}
	}
}
func TestCapabilityValidationRejectsUnsupportedAndDuplicates(t *testing.T) {
	for _, v := range [][]string{{}, {"unknown"}, {CapabilityCollector, CapabilityCollector}} {
		if validateCapabilities(v) == nil {
			t.Fatalf("must reject %#v", v)
		}
	}
	if validateCapabilities([]string{CapabilityProjectEntry, CapabilityDataRuntime, CapabilityCollector}) != nil {
		t.Fatal("valid capability set rejected")
	}
}

func TestEnrollmentCapabilitiesMustMatchExactly(t *testing.T) {
	requested := []string{CapabilityProjectEntry, CapabilityDataRuntime}
	if !sameStrings([]string{CapabilityDataRuntime, CapabilityProjectEntry}, requested) {
		t.Fatal("capability order must not affect enrollment matching")
	}
	if sameStrings([]string{CapabilityProjectEntry, CapabilityDataRuntime, CapabilityCollector}, requested) {
		t.Fatal("NodeAgent must not escalate beyond the approved capability set")
	}
}

func TestClaimRequiresAuditableAgentIdentity(t *testing.T) {
	service := NewService(&deploymentRepository{}, nil)
	base := ClaimEnrollmentInput{
		Code: "code", Hostname: "edge-01", Platform: PlatformLinux, Architecture: "amd64",
		AgentVersion: "1.0.0", MachineFingerprint: "sha256:fingerprint",
		Capabilities: []string{CapabilityProjectEntry, CapabilityDataRuntime},
	}
	base.MachineFingerprint = ""
	if _, _, _, err := service.ClaimEnrollment(context.Background(), base); err == nil || !strings.Contains(err.Error(), "机器指纹") {
		t.Fatalf("missing fingerprint must be rejected: %v", err)
	}
	base.MachineFingerprint = "sha256:fingerprint"
	base.AgentVersion = ""
	if _, _, _, err := service.ClaimEnrollment(context.Background(), base); err == nil || !strings.Contains(err.Error(), "Agent 版本") {
		t.Fatalf("missing agent version must be rejected: %v", err)
	}
}

func TestHeartbeatRejectsMalformedServiceObservation(t *testing.T) {
	service := NewService(&deploymentRepository{}, nil)
	for _, observation := range []ServiceObservation{
		{ServiceID: "not-a-uuid", ObservedStatus: "running", ObservedGeneration: 1, ReplicasObserved: 1},
		{ServiceID: testVersionID, ObservedStatus: "starting", ObservedGeneration: 1, ReplicasObserved: 1},
		{ServiceID: testVersionID, ObservedStatus: "running", ObservedGeneration: 0, ReplicasObserved: 1},
		{ServiceID: testVersionID, ObservedStatus: "running", ObservedGeneration: 1, ReplicasObserved: 2},
	} {
		if _, _, err := service.Heartbeat(context.Background(), testNodeID, "agent-token", HeartbeatInput{Services: []ServiceObservation{observation}}); err == nil {
			t.Fatalf("malformed service observation accepted: %#v", observation)
		}
	}
}
func TestDefaultServicesAreExplicit(t *testing.T) {
	if !validServiceType(ServiceProjectEntry) || !validServiceType(ServiceDataRuntime) || !validServiceType(ServiceCollector) || validServiceType("compute") {
		t.Fatal("unexpected service vocabulary")
	}
}

func TestReportedEndpointRejectsUnsafeValues(t *testing.T) {
	if err := validateReportedEndpoint("https://gateway.example.com/engineering"); err != nil {
		t.Fatalf("valid endpoint rejected: %v", err)
	}
	for _, endpoint := range []string{
		"ftp://gateway.example.com",
		"https://user:pass@gateway.example.com",
		"https://gateway.example.com/#fragment",
	} {
		if err := validateReportedEndpoint(endpoint); err == nil {
			t.Fatalf("unsafe endpoint accepted: %s", endpoint)
		}
	}
}
