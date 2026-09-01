package binding

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/indu-forge/runtime-engine/internal/model"
)

func TestBuildEngineConfigComputeBuildsV2WithoutSecretValues(t *testing.T) {
	input := validInput(roleCompute)
	config, err := BuildEngineConfig(input)
	if err != nil {
		t.Fatalf("build compute config: %v", err)
	}
	if err := model.ValidateEngineConfig(config); err != nil {
		t.Fatalf("built compute config failed formal validation: %v", err)
	}
	if config.SchemaVersion != "runtime-engine.config.v2" || config.ExecutionForm != "native-linux" || config.ArtifactMount.Source != "native-release" || !config.ArtifactMount.ReadOnly {
		t.Fatalf("v2 native binding was not frozen: %+v", config)
	}
	if got := config.Roles; len(got) != 1 || got[0] != roleCompute {
		t.Fatalf("expected exactly compute role, got %#v", got)
	}
	if config.ComputeSandbox == nil || config.ComputeSandbox.ServerResourceRef != input.ComputeSandbox.ServerResourceRef || config.ComputeSandbox.CredentialSecretRef != input.ComputeSandbox.CredentialSecretRef {
		t.Fatalf("compute sandbox reference mismatch: %+v", config.ComputeSandbox)
	}
	if len(config.ProducerAssignments) != 1 || config.ProducerAssignments[0].ProducerType != "compute" || config.ProducerAssignments[0].ComputeID != input.ComputeProducers[0].ComputeID {
		t.Fatalf("compute producer mismatch: %+v", config.ProducerAssignments)
	}

	input.JetStream.Consumers[0].BackoffMS[0] = 999999
	if config.JetStream.Consumers[0].BackoffMS[0] != 1000 {
		t.Fatal("config consumers must not alias input")
	}
	raw, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"tenantId", "endpoint", "resourceRef", "token", "password"} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("runtime config leaked deployment-only or secret field %q: %s", forbidden, raw)
		}
	}
}

func TestBuildEngineConfigAlarmBuildsSingleAlarmProducer(t *testing.T) {
	input := validInput(roleAlarm)
	config, err := BuildEngineConfig(input)
	if err != nil {
		t.Fatalf("build alarm config: %v", err)
	}
	if err := model.ValidateEngineConfig(config); err != nil {
		t.Fatalf("built alarm config failed formal validation: %v", err)
	}
	if config.ComputeSandbox != nil || len(config.ProducerAssignments) != 1 || config.ProducerAssignments[0].ProducerType != "alarm" || config.ProducerAssignments[0].Role != roleAlarm {
		t.Fatalf("alarm binding did not remain isolated: %+v", config)
	}
}

func TestBuildEngineConfigRejectsIncompleteOrInvalidTopology(t *testing.T) {
	t.Run("deployment context missing", func(t *testing.T) {
		input := validInput(roleCompute)
		input.StateStore.Schema = ""
		if _, err := BuildEngineConfig(input); err == nil || !strings.Contains(err.Error(), "stateStore.schema") {
			t.Fatalf("expected state-store context rejection, got %v", err)
		}
	})
	t.Run("role boundary", func(t *testing.T) {
		input := validInput(roleAlarm)
		input.ComputeSandbox = &model.ComputeSandbox{ServerResourceRef: "site-resource://site-a/sandbox", CredentialSecretRef: "secret://site-a/sandbox"}
		if _, err := BuildEngineConfig(input); err == nil || !strings.Contains(err.Error(), "不得声明 compute") {
			t.Fatalf("expected mixed-role rejection, got %v", err)
		}
	})
	t.Run("formal validator", func(t *testing.T) {
		input := validInput(roleCompute)
		input.JetStream.Consumers[0].BackoffMS = []int64{5000, 1000}
		if _, err := BuildEngineConfig(input); err == nil || !strings.Contains(err.Error(), "非递减") {
			t.Fatalf("expected formal validator error, got %v", err)
		}
	})
}

func validInput(role string) Input {
	input := Input{
		TenantID:          "tenant-a",
		SiteID:            "site-a",
		NodeID:            "node-a",
		ProjectID:         "11111111-1111-4111-8111-111111111111",
		DeploymentID:      "deployment-a",
		AccountID:         "account-a",
		Role:              role,
		ProjectArtifact:   model.ArtifactRef{ArtifactID: "runtime-artifact-a", ArtifactRevision: 1, ArtifactDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		ArtifactMountPath: "/opt/induforge/release/current",
		ArtifactFile:      "runtime-project-artifact.json",
		RoleOwnership:     model.Ownership{OwnerID: "runtime-engine-" + role + "-0", Epoch: 1},
		JetStream: JetStreamInput{
			Endpoint:             "nats://nats:4222",
			ServerResourceRef:    "site-resource://site-a/nats",
			CredentialSecretRef:  "secret://site-a/deployment-a/nats-runtime-engine",
			CredentialSecretFile: "secrets/nats-runtime-engine.json",
			DataRawStream:        "DATA_RAW",
			DataDerivedStream:    "DATA_DERIVED",
			EventStream:          "EVENT",
			DeadLetterStream:     "RUNTIME_DLQ",
			Consumers:            consumers(role),
		},
		StateStore: StateStoreInput{ResourceRef: "site-resource://site-a/postgres", CredentialSecretRef: "secret://site-a/deployment-a/runtime-postgres", CredentialSecretFile: "secrets/runtime-postgres.json", Schema: "runtime"},
	}
	if role == roleCompute {
		input.ComputeSandbox = &model.ComputeSandbox{ServerResourceRef: "site-resource://site-a/compute-sandbox", CredentialSecretRef: "secret://site-a/deployment-a/compute-sandbox"}
		input.ComputeSandboxEndpoint = "http://compute-sandbox:18103"
		input.ComputeSandboxSecretFile = "secrets/compute-sandbox.json"
		input.ComputeProducers = []ComputeProducer{{ComputeID: "44444444-4444-4444-8444-444444444444", Ownership: model.Ownership{OwnerID: "runtime-engine-compute-0", Epoch: 1}}}
	} else {
		input.AlarmOwnership = model.Ownership{OwnerID: "runtime-engine-alarm-0", Epoch: 1}
	}
	return input
}

func consumers(role string) []model.Consumer {
	return []model.Consumer{
		consumer(role, "raw", "DATA_RAW", "data.raw.>"),
		consumer(role, "derived", "DATA_DERIVED", "data.computed.>"),
	}
}

func consumer(role, kind, stream, subject string) model.Consumer {
	return model.Consumer{
		Role: role, ConsumerKey: role + "-" + kind + "-v1", DurableName: role + "-" + kind + "-v1", Stream: stream, FilterSubject: subject,
		AckPolicy: "explicit", AckWaitMS: 30000, MaxDeliver: 5, BackoffMS: []int64{1000, 5000, 30000},
		MaxAckPending: 32, MaxWaiting: 32, MaxRequestBatch: 32, MaxRequestExpiresMS: 5000, MaxRequestMaxBytes: 1 << 20,
		DeadLetterSubject: "dlq." + role + "-" + kind,
	}
}
