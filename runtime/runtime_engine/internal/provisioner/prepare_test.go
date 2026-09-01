package provisioner

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/indu-forge/runtime-engine/internal/binding"
	"github.com/indu-forge/runtime-engine/internal/model"
)

func TestDecodeRejectsUnknownFields(t *testing.T) {
	raw := []byte(`{"schemaVersion":"runtime-binding.input.v1","unknown":true}`)
	if _, err := Decode(raw); err == nil {
		t.Fatal("unknown field accepted")
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
		TenantID: "tenant-a", SiteID: "site-a", NodeID: "node-a", ProjectID: "11111111-1111-4111-8111-111111111111", DeploymentID: "deployment-a", AccountID: "account-a", Role: "alarm",
		ProjectArtifact: model.ArtifactRef{ArtifactID: "artifact-a", ArtifactRevision: 1, ArtifactDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, ArtifactMountPath: "/work/artifact", ArtifactFile: "runtime-project-artifact.json", RoleOwnership: model.Ownership{OwnerID: "alarm-a", Epoch: 1}, AlarmOwnership: model.Ownership{OwnerID: "alarm-a", Epoch: 1},
		JetStream:  binding.JetStreamInput{Endpoint: "nats://nats:4222", ServerResourceRef: "site-resource://site-a/nats", CredentialSecretRef: "secret://site-a/nats", CredentialSecretFile: "secrets/nats.json", DataRawStream: "DATA_RAW", DataDerivedStream: "DATA_DERIVED", EventStream: "EVENT", DeadLetterStream: "DLQ", Consumers: []model.Consumer{{Role: "alarm", ConsumerKey: "alarm-raw-v1", Stream: "DATA_RAW", DurableName: "alarm-raw-v1", FilterSubject: "data.raw.>", AckPolicy: "explicit", AckWaitMS: 1000, MaxDeliver: 1, BackoffMS: []int64{1000}, MaxAckPending: 32, MaxWaiting: 32, MaxRequestBatch: 32, MaxRequestExpiresMS: 5000, MaxRequestMaxBytes: 1 << 20, DeadLetterSubject: "dlq.alarm-raw"}, {Role: "alarm", ConsumerKey: "alarm-derived-v1", Stream: "DATA_DERIVED", DurableName: "alarm-derived-v1", FilterSubject: "data.computed.>", AckPolicy: "explicit", AckWaitMS: 1000, MaxDeliver: 1, BackoffMS: []int64{1000}, MaxAckPending: 32, MaxWaiting: 32, MaxRequestBatch: 32, MaxRequestExpiresMS: 5000, MaxRequestMaxBytes: 1 << 20, DeadLetterSubject: "dlq.alarm-derived"}}},
		StateStore: binding.StateStoreInput{ResourceRef: "site-resource://site-a/postgres", CredentialSecretRef: "secret://site-a/pg", CredentialSecretFile: "secrets/pg.json", Schema: "runtime"},
	}}
}
