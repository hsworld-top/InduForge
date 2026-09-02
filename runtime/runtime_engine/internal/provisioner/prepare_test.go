package provisioner

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/indu-forge/runtime-engine/internal/binding"
	"github.com/indu-forge/runtime-engine/internal/model"
)

func TestDecodeAcceptsDevCoreGoldenFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/dev-core.runtime-binding.input.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Decode(raw); err != nil {
		t.Fatalf("dev_core golden fixture rejected: %v", err)
	}
}

func TestDecodeRejectsUnknownFields(t *testing.T) {
	raw := []byte(`{"schemaVersion":"runtime-binding.input.v1","unknown":true}`)
	if _, err := Decode(raw); err == nil {
		t.Fatal("unknown field accepted")
	}
}

func TestDecodeRejectsForgedProducerFields(t *testing.T) {
	raw, err := json.Marshal(validInput())
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	document["binding"].(map[string]any)["computeProducers"] = []any{map[string]any{"computeId": "forged"}}
	tampered, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decode(tampered); err == nil {
		t.Fatal("forged producer assignment accepted")
	}
}

func TestDeriveBuildInputUsesArtifactProducers(t *testing.T) {
	compute := validInput().Binding
	compute.Role = "compute"
	compute.InstanceID = "compute-instance"
	compute.FencingEpoch = 4
	compute.ComputeSandbox = &model.ComputeSandbox{ServerResourceRef: "site-resource://site-a/sandbox", CredentialSecretRef: "secret://site-a/sandbox"}
	compute.ComputeSandboxEndpoint = "http://sandbox"
	compute.ComputeSandboxSecretFile = "secrets/sandbox.json"
	build, err := deriveBuildInput(compute, []byte(`{"schemaVersion":"runtime-project-artifact.v1","projectArtifactVersion":"1.0","projectId":"11111111-1111-4111-8111-111111111111","computeUnits":[{"id":"22222222-2222-4222-8222-222222222222","revision":9,"enabled":true},{"id":"33333333-3333-4333-8333-333333333333","revision":2,"enabled":false}],"alarmItems":[]}`))
	if err != nil {
		t.Fatalf("derive compute: %v", err)
	}
	if len(build.ComputeProducers) != 1 || build.ComputeProducers[0].ComputeID != "22222222-2222-4222-8222-222222222222" || build.ComputeProducers[0].Ownership != build.RoleOwnership || build.ComputeProducers[0].Ownership.Epoch != 4 || build.ProjectArtifact.ArtifactDigest == "" {
		t.Fatalf("compute derivation wrong: %+v", build)
	}

	alarm := validInput().Binding
	alarm.InstanceID = "alarm-instance"
	alarm.FencingEpoch = 4
	build, err = deriveBuildInput(alarm, []byte(`{"schemaVersion":"runtime-project-artifact.v1","projectArtifactVersion":"1.0","projectId":"11111111-1111-4111-8111-111111111111","computeUnits":[],"alarmItems":[{"id":"44444444-4444-4444-8444-444444444444","revision":3,"enabled":true},{"id":"55555555-5555-4555-8555-555555555555","revision":7,"enabled":true}]}`))
	if err != nil {
		t.Fatalf("derive alarm: %v", err)
	}
	if build.AlarmOwnership != build.RoleOwnership || build.AlarmOwnership.Epoch != 4 || build.AlarmOwnership.OwnerID != stableRoleOwner(alarm.DeploymentID, alarm.Role) {
		t.Fatalf("alarm derivation wrong: %+v", build.AlarmOwnership)
	}
	if build.AlarmOwnership.OwnerID == alarm.InstanceID {
		t.Fatal("诊断 instanceId 不得成为 alarm fence owner")
	}
}

func TestDeriveBuildInputRejectsEmptyRoleProducer(t *testing.T) {
	input := validInput().Binding
	if _, err := deriveBuildInput(input, []byte(`{"projectArtifactVersion":"1.0","projectId":"11111111-1111-4111-8111-111111111111","computeUnits":[],"alarmItems":[]}`)); err == nil || !strings.Contains(err.Error(), "缺少可用 producer") {
		t.Fatalf("expected empty producer rejection, got %v", err)
	}
}

func TestDeriveBuildInputSeparatesStableOwnerFromRolloutEpoch(t *testing.T) {
	artifact := []byte(`{"schemaVersion":"runtime-project-artifact.v1","projectArtifactVersion":"1.0","projectId":"11111111-1111-4111-8111-111111111111","computeUnits":[],"alarmItems":[{"id":"44444444-4444-4444-8444-444444444444","revision":9,"enabled":true}]}`)
	first := validInput().Binding
	first.InstanceID, first.FencingEpoch = "if-alarm-old-instance", 3
	firstBuild, err := deriveBuildInput(first, artifact)
	if err != nil {
		t.Fatal(err)
	}
	retry := first
	retry.InstanceID = "if-alarm-restarted-instance"
	retryBuild, err := deriveBuildInput(retry, artifact)
	if err != nil {
		t.Fatal(err)
	}
	if retryBuild.RoleOwnership != firstBuild.RoleOwnership || retryBuild.AlarmOwnership != firstBuild.AlarmOwnership {
		t.Fatalf("同 generation 重试必须保持 fencing token：first=%+v retry=%+v", firstBuild.RoleOwnership, retryBuild.RoleOwnership)
	}
	next := retry
	next.InstanceID, next.FencingEpoch = "if-alarm-next-instance", 4
	nextBuild, err := deriveBuildInput(next, artifact)
	if err != nil {
		t.Fatal(err)
	}
	if nextBuild.RoleOwnership.OwnerID != firstBuild.RoleOwnership.OwnerID || nextBuild.RoleOwnership.Epoch != firstBuild.RoleOwnership.Epoch+1 || nextBuild.AlarmOwnership != nextBuild.RoleOwnership {
		t.Fatalf("generation 提升必须只提升同一逻辑 owner 的 epoch：first=%+v next=%+v", firstBuild.RoleOwnership, nextBuild.RoleOwnership)
	}
}

func TestPrepareRejectsIdentityAndPathBoundariesBeforeIO(t *testing.T) {
	in := validInput()
	in.ArtifactDir = "/tmp/artifact"
	if err := Prepare(in); err == nil || !strings.Contains(err.Error(), "路径") {
		t.Fatalf("expected work-root rejection, got %v", err)
	}
	in = validInput()
	in.Binding.JetStream.CredentialSecretFile = "../nats.json"
	if err := Prepare(in); err == nil || !strings.Contains(err.Error(), "secret") {
		t.Fatalf("expected secret-path rejection, got %v", err)
	}
	in = validInput()
	in.Binding.ArtifactMountPath = "/work/other"
	if err := Prepare(in); err == nil || !strings.Contains(err.Error(), "目标") {
		t.Fatalf("expected artifact identity rejection, got %v", err)
	}
}

func TestInputRoundTripHasNoSecretValueFields(t *testing.T) {
	raw, err := json.Marshal(validInput())
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"token", "password", "dsn", "postgres://"} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("input contains %q", forbidden)
		}
	}
}

func validInput() Input {
	return Input{SchemaVersion: SchemaVersion, ReleaseID: "release-a", RuntimeArtifactPath: "/opt/induforge/release/runtime-artifact.tar.zst", RuntimeArtifactSHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ArtifactDir: "/work/artifact", BundleDir: "/work/bundle", Binding: binding.Input{
		TenantID: "tenant-a", SiteID: "site-a", NodeID: "node-a", InstanceID: "runtime-engine-alarm-0", FencingEpoch: 1, ProjectID: "11111111-1111-4111-8111-111111111111", DeploymentID: "deployment-a", AccountID: "account-a", Role: "alarm",
		ArtifactMountPath: "/work/artifact", ArtifactFile: "runtime-project-artifact.json",
		JetStream:  binding.JetStreamInput{Endpoint: "nats://nats:4222", ServerResourceRef: "site-resource://site-a/nats", CredentialSecretRef: "secret://site-a/nats", CredentialSecretFile: "secrets/nats.json", DataRawStream: "DATA_RAW", DataDerivedStream: "DATA_DERIVED", EventStream: "EVENT", CommandStream: "COMMAND", DeadLetterStream: "DLQ", Consumers: []model.Consumer{{Role: "alarm", ConsumerKey: "alarm-raw-v1", Stream: "DATA_RAW", DurableName: "alarm-raw-v1", FilterSubject: "data.raw.>", AckPolicy: "explicit", AckWaitMS: 1000, MaxDeliver: 1, BackoffMS: []int64{1000}, MaxAckPending: 32, MaxWaiting: 32, MaxRequestBatch: 32, MaxRequestExpiresMS: 5000, MaxRequestMaxBytes: 1 << 20, DeadLetterSubject: "dlq.alarm-raw"}, {Role: "alarm", ConsumerKey: "alarm-derived-v1", Stream: "DATA_DERIVED", DurableName: "alarm-derived-v1", FilterSubject: "data.computed.>", AckPolicy: "explicit", AckWaitMS: 1000, MaxDeliver: 1, BackoffMS: []int64{1000}, MaxAckPending: 32, MaxWaiting: 32, MaxRequestBatch: 32, MaxRequestExpiresMS: 5000, MaxRequestMaxBytes: 1 << 20, DeadLetterSubject: "dlq.alarm-derived"}}},
		StateStore: binding.StateStoreInput{ResourceRef: "site-resource://site-a/postgres", CredentialSecretRef: "secret://site-a/pg", CredentialSecretFile: "secrets/pg.json", Schema: "runtime"},
	}}
}
