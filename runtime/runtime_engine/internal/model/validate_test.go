package model

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func modelFixture(t *testing.T, name string, target any) {
	t.Helper()
	bytes, err := os.ReadFile(filepath.Join("../../../../contracts/runtime/fixtures", name))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(bytes, target); err != nil {
		t.Fatal(err)
	}
}

func TestValidateEngineConfigCrossFieldFailures(t *testing.T) {
	var config EngineConfig
	modelFixture(t, "runtime-engine-config.valid.json", &config)
	config.RoleAssignments[1].Role = "writer"
	if err := ValidateEngineConfig(config); err == nil || !strings.Contains(err.Error(), "重复角色") {
		t.Fatalf("expected duplicate role error, got %v", err)
	}

	modelFixture(t, "runtime-engine-config.valid.json", &config)
	config.ProducerAssignments[1].Ownership = Ownership{OwnerID: "independent-compute-producer", Epoch: 99}
	config.ProducerAssignments[2].Ownership = Ownership{OwnerID: "independent-alarm-producer", Epoch: 99}
	if err := ValidateEngineConfig(config); err != nil {
		t.Fatalf("producer and consumer fencing tokens are independent: %v", err)
	}

	modelFixture(t, "runtime-engine-config.valid.json", &config)
	config.JetStream.Consumers[0].BackoffMS = []int64{5000, 1000}
	if err := ValidateEngineConfig(config); err == nil || !strings.Contains(err.Error(), "非递减") {
		t.Fatalf("expected backoff error, got %v", err)
	}

	modelFixture(t, "runtime-engine-config.valid.json", &config)
	config.JetStream.Consumers = config.JetStream.Consumers[1:]
	if err := ValidateEngineConfig(config); err == nil || !strings.Contains(err.Error(), "writer role") {
		t.Fatalf("expected consumer coverage error, got %v", err)
	}
}

func TestValidateProjectArtifactRejectsAmbiguousOutput(t *testing.T) {
	var artifact ProjectArtifact
	var config EngineConfig
	modelFixture(t, "runtime-project-artifact.valid.json", &artifact)
	modelFixture(t, "runtime-engine-config.valid.json", &config)
	artifact.ComputeUnits[1].Outputs[0].DatapointID = artifact.ComputeUnits[0].Outputs[0].DatapointID
	artifact.ComputeUnits[1].Outputs[0].Path = artifact.ComputeUnits[0].Outputs[0].Path
	artifact.ComputeUnits[1].Outputs[0].DataType = artifact.ComputeUnits[0].Outputs[0].DataType
	if err := ValidateProjectArtifact(artifact, config); err == nil {
		t.Fatalf("expected output ownership error, got %v", err)
	}
}

func TestValidateProjectArtifactAlarmItemsBound(t *testing.T) {
	var artifact ProjectArtifact
	var config EngineConfig
	modelFixture(t, "runtime-project-artifact.valid.json", &artifact)
	modelFixture(t, "runtime-engine-config.valid.json", &config)
	if err := ValidateProjectArtifact(artifact, config); err != nil {
		t.Fatalf("valid fixture rejected: %v", err)
	}

	template := artifact.AlarmItems[0]
	artifact.AlarmItems = make([]AlarmItem, 0, MaxAlarmItems+1)
	for index := range MaxAlarmItems {
		item := template
		item.ID = alarmBoundUUID(0x70000000 + index)
		item.Conditions = append([]AlarmCondition(nil), template.Conditions...)
		item.Conditions[0].ID = alarmBoundUUID(0x80000000 + index)
		artifact.AlarmItems = append(artifact.AlarmItems, item)
	}
	if err := ValidateProjectArtifact(artifact, config); err != nil {
		t.Fatalf("boundary %d rejected: %v", MaxAlarmItems, err)
	}
	extra := artifact.AlarmItems[0]
	extra.ID = alarmBoundUUID(0x90000000)
	extra.Conditions = append([]AlarmCondition(nil), extra.Conditions...)
	extra.Conditions[0].ID = alarmBoundUUID(0xa0000000)
	artifact.AlarmItems = append(artifact.AlarmItems, extra)
	if err := ValidateProjectArtifact(artifact, config); err == nil || !strings.Contains(err.Error(), "超过 V1 上限") {
		t.Fatalf("max+1 accepted: %v", err)
	}
}

func alarmBoundUUID(value int) string {
	return fmt.Sprintf("%08x-0000-4000-8000-%012x", value, value)
}

func TestValidateProjectArtifactCrossFieldFailures(t *testing.T) {
	var artifact ProjectArtifact
	var config EngineConfig
	modelFixture(t, "runtime-project-artifact.valid.json", &artifact)
	modelFixture(t, "runtime-engine-config.valid.json", &config)
	artifact.ComputeUnits[0].Inputs[0].Path = "wrong.path"
	if err := ValidateProjectArtifact(artifact, config); err == nil {
		t.Fatal("expected reference metadata validation failure")
	}

	modelFixture(t, "runtime-project-artifact.valid.json", &artifact)
	artifact.ComputeUnits = artifact.ComputeUnits[:1]
	for index := range artifact.DataPoints {
		if artifact.DataPoints[index].SourceType == "calc.output" {
			source := artifact.ComputeUnits[0].ID
			artifact.DataPoints[index].SourceID = &source
			artifact.DataPoints[index].SourceConfig = json.RawMessage(fmt.Sprintf(`{"computeId":%q}`, source))
		}
	}
	config.ProducerAssignments = config.ProducerAssignments[:1]
	config.ProducerAssignments = append(config.ProducerAssignments, ProducerAssignment{ProducerType: "compute", ComputeID: artifact.ComputeUnits[0].ID, Role: "compute", Ownership: Ownership{OwnerID: "different-producer-token", Epoch: 9}}, ProducerAssignment{ProducerType: "alarm", Role: "alarm", Ownership: Ownership{OwnerID: "different-alarm-token", Epoch: 9}})
	artifact.AlarmItems[0].Conditions[0].Kind = "range"
	artifact.AlarmItems[0].Conditions[0].Params = json.RawMessage(`{"lower": 100, "upper": 10}`)
	if err := ValidateProjectArtifact(artifact, config); err == nil || !strings.Contains(err.Error(), "lower < upper") {
		t.Fatalf("expected range error, got %v", err)
	}
}

func TestValidateProjectArtifactSourceDiscriminator(t *testing.T) {
	load := func(t *testing.T) (ProjectArtifact, EngineConfig) {
		t.Helper()
		var artifact ProjectArtifact
		var config EngineConfig
		modelFixture(t, "runtime-project-artifact.valid.json", &artifact)
		modelFixture(t, "runtime-engine-config.valid.json", &config)
		return artifact, config
	}
	find := func(t *testing.T, artifact *ProjectArtifact, sourceType string) *DataPoint {
		t.Helper()
		for index := range artifact.DataPoints {
			if artifact.DataPoints[index].SourceType == sourceType {
				return &artifact.DataPoints[index]
			}
		}
		t.Fatalf("fixture missing %s datapoint", sourceType)
		return nil
	}
	assertRejected := func(t *testing.T, name string, mutate func(*ProjectArtifact)) {
		t.Helper()
		artifact, config := load(t)
		mutate(&artifact)
		if err := ValidateProjectArtifact(artifact, config); err == nil {
			t.Fatalf("%s was accepted", name)
		}
	}
	assertRejected(t, "unknown sourceType", func(artifact *ProjectArtifact) {
		find(t, artifact, "manual.input").SourceType = "manual"
	})
	assertRejected(t, "manual sourceId", func(artifact *ProjectArtifact) {
		point := find(t, artifact, "manual.input")
		id := "aaaaaaaa-1111-4111-8111-111111111111"
		point.SourceID = &id
	})
	assertRejected(t, "manual non-empty config", func(artifact *ProjectArtifact) {
		find(t, artifact, "manual.input").SourceConfig = []byte(`{"unexpected":true}`)
	})
	for _, invalid := range []string{
		`{"connectionId":"aaaaaaaa-1111-4111-8111-111111111111","endpoint":"tcp://unsafe"}`,
		`{"connectionId":"aaaaaaaa-1111-4111-8111-111111111111","secret":"unsafe"}`,
		`{"connectionId":"aaaaaaaa-1111-4111-8111-111111111111","connectionId":"aaaaaaaa-1111-4111-8111-111111111111"}`,
	} {
		invalid := invalid
		assertRejected(t, "collector strict sourceConfig", func(artifact *ProjectArtifact) {
			find(t, artifact, "collector.point").SourceConfig = []byte(invalid)
		})
	}
	for _, invalid := range []string{
		`{"computeId":"44444444-4444-4444-8444-444444444444","outputKey":"result"}`,
		`{"computeId":"44444444-4444-4444-8444-444444444444","computeId":"44444444-4444-4444-8444-444444444444"}`,
	} {
		invalid := invalid
		assertRejected(t, "calc strict sourceConfig", func(artifact *ProjectArtifact) {
			find(t, artifact, "calc.output").SourceConfig = []byte(invalid)
		})
	}
}

func TestValidateComputeOutputOwnershipAndConditionMappings(t *testing.T) {
	load := func(t *testing.T) (ProjectArtifact, EngineConfig) {
		var a ProjectArtifact
		var c EngineConfig
		modelFixture(t, "runtime-project-artifact.valid.json", &a)
		modelFixture(t, "runtime-engine-config.valid.json", &c)
		return a, c
	}
	t.Run("collector point output", func(t *testing.T) {
		a, c := load(t)
		point := &a.DataPoints[2]
		point.SourceType = "collector.point"
		if err := ValidateProjectArtifact(a, c); err == nil {
			t.Fatal("collector point accepted as output")
		}
	})
	t.Run("inactive output", func(t *testing.T) {
		a, c := load(t)
		a.DataPoints[2].Status = "inactive"
		if err := ValidateProjectArtifact(a, c); err == nil {
			t.Fatal("inactive output accepted")
		}
	})
	t.Run("wrong source config", func(t *testing.T) {
		a, c := load(t)
		a.DataPoints[2].SourceConfig = []byte(`{"computeId":"66666666-6666-4666-8666-666666666666"}`)
		if err := ValidateProjectArtifact(a, c); err == nil {
			t.Fatal("wrong compute source accepted")
		}
	})
	t.Run("duplicate source config key", func(t *testing.T) {
		a, c := load(t)
		a.DataPoints[2].SourceConfig = []byte(`{"computeId":"55555555-5555-4555-8555-555555555555","computeId":"55555555-5555-4555-8555-555555555555"}`)
		if err := ValidateProjectArtifact(a, c); err == nil {
			t.Fatal("duplicate sourceConfig computeId accepted")
		}
	})
	t.Run("other unit point", func(t *testing.T) {
		a, c := load(t)
		other := a.DataPoints[1]
		a.ComputeUnits[1].Outputs[0].DatapointID = other.ID
		a.ComputeUnits[1].Outputs[0].Path = other.Path
		a.ComputeUnits[1].Outputs[0].DataType = other.DataType
		if err := ValidateProjectArtifact(a, c); err == nil {
			t.Fatal("other unit output point accepted")
		}
	})
	t.Run("condition point path type", func(t *testing.T) {
		a, c := load(t)
		a.ComputeUnits[3].Trigger.Variables[0].DatapointID = a.DataPoints[1].ID
		if err := ValidateProjectArtifact(a, c); err == nil {
			t.Fatal("condition datapoint mismatch accepted")
		}
		a, c = load(t)
		a.ComputeUnits[3].Trigger.Variables[0].Path = "wrong"
		if err := ValidateProjectArtifact(a, c); err == nil {
			t.Fatal("condition path mismatch accepted")
		}
		a, c = load(t)
		a.ComputeUnits[3].Trigger.Variables[0].DataType = "int64"
		if err := ValidateProjectArtifact(a, c); err == nil {
			t.Fatal("condition type mismatch accepted")
		}
	})
	t.Run("condition missing variable", func(t *testing.T) {
		a, c := load(t)
		a.ComputeUnits[3].Trigger.Variables = nil
		if err := ValidateProjectArtifact(a, c); err == nil {
			t.Fatal("condition missing mapping accepted")
		}
	})
	t.Run("condition extra variable", func(t *testing.T) {
		a, c := load(t)
		a.ComputeUnits[3].Inputs = append(a.ComputeUnits[3].Inputs, Input{
			DatapointID: a.DataPoints[1].ID,
			Alias:       "other",
			Path:        a.DataPoints[1].Path,
			DataType:    a.DataPoints[1].DataType,
		})
		if err := ValidateProjectArtifact(a, c); err == nil {
			t.Fatal("condition extra input without variable accepted")
		}
	})
	t.Run("condition literal alias", func(t *testing.T) {
		a, c := load(t)
		a.ComputeUnits[3].Inputs[0].Alias = "true"
		a.ComputeUnits[3].Trigger.Variables[0].Alias = "true"
		if err := ValidateProjectArtifact(a, c); err == nil {
			t.Fatal("condition true alias accepted")
		}
	})
}

func TestValidateComputeDefaultValueAtLoad(t *testing.T) {
	load := func(t *testing.T) (ProjectArtifact, EngineConfig) {
		var a ProjectArtifact
		var c EngineConfig
		modelFixture(t, "runtime-project-artifact.valid.json", &a)
		modelFixture(t, "runtime-engine-config.valid.json", &c)
		return a, c
	}
	for _, value := range []json.RawMessage{nil, []byte(`null`), []byte(`"wrong"`), []byte(`1 trailing`)} {
		a, c := load(t)
		a.ComputeUnits[1].Outputs[0].DefaultValue = value
		if err := ValidateProjectArtifact(a, c); err == nil {
			t.Fatalf("bad default accepted: %s", value)
		}
	}
	a, c := load(t)
	a.ComputeUnits[1].Outputs[0].DataType = "int64"
	a.DataPoints[2].DataType = "int64"
	a.ComputeUnits[1].Outputs[0].DefaultValue = []byte(`9223372036854775808`)
	if err := ValidateProjectArtifact(a, c); err == nil {
		t.Fatal("int64 overflow default accepted")
	}
	a, c = load(t)
	a.ComputeUnits[1].Outputs[0].DefaultValue = []byte(`18446744073709551615`)
	a.ComputeUnits[1].Outputs[0].DataType = "uint64"
	a.DataPoints[2].DataType = "uint64"
	if err := ValidateProjectArtifact(a, c); err != nil {
		t.Fatalf("uint64 max default rejected: %v", err)
	}
	a, c = load(t)
	a.ComputeUnits[0].Outputs[0].DefaultValue = []byte(`0`)
	if err := ValidateProjectArtifact(a, c); err == nil {
		t.Fatal("propagate output defaultValue accepted")
	}
}

func TestValidateDataPointValueStrictBoundaries(t *testing.T) {
	for _, test := range []struct {
		name string
		raw  json.RawMessage
		typ  string
		ok   bool
	}{
		{"uint64 max", []byte(`18446744073709551615`), "uint64", true},
		{"uint64 overflow", []byte(`18446744073709551616`), "uint64", false},
		{"finite float64", []byte(`1.7976931348623157e308`), "float64", true},
		{"float64 overflow", []byte(`1e309`), "float64", false},
		{"base64", []byte(`"AQI="`), "bytes", true},
		{"bad base64", []byte(`"AQI"`), "bytes", false},
		{"base64 newline", []byte(`"AQI=\n"`), "bytes", false},
		{"UTC datetime", []byte(`"2026-01-02T03:04:05Z"`), "datetime", true},
		{"offset datetime", []byte(`"2026-01-02T11:04:05+08:00"`), "datetime", false},
		{"trailing", []byte(`1 2`), "int64", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := ValidateDataPointValue(test.raw, test.typ, false) == nil; got != test.ok {
				t.Fatalf("ValidateDataPointValue(%s, %s) ok=%v, want %v", test.raw, test.typ, got, test.ok)
			}
		})
	}
}

func TestValidateDeadbandRetainsExactJSON(t *testing.T) {
	var a ProjectArtifact
	var c EngineConfig
	modelFixture(t, "runtime-project-artifact.valid.json", &a)
	modelFixture(t, "runtime-engine-config.valid.json", &c)
	a.ComputeUnits[1].Trigger.Deadband = []byte(`9007199254740993.0000000000000000000001`)
	if err := ValidateProjectArtifact(a, c); err != nil {
		t.Fatalf("exact deadband rejected: %v", err)
	}
	a.ComputeUnits[1].Trigger.Deadband = []byte(`1 trailing`)
	if err := ValidateProjectArtifact(a, c); err == nil {
		t.Fatal("trailing deadband accepted")
	}
}

func TestValidateAlarmConditionExactNumbers(t *testing.T) {
	load := func(t *testing.T) (ProjectArtifact, EngineConfig) {
		var a ProjectArtifact
		var c EngineConfig
		modelFixture(t, "runtime-project-artifact.valid.json", &a)
		modelFixture(t, "runtime-engine-config.valid.json", &c)
		return a, c
	}
	// Adjacent values above 2^53 and high-precision decimals are valid ordered
	// range endpoints; a float64 parser would merge each pair and reject them.
	for _, params := range []json.RawMessage{
		[]byte(`{"lower":9007199254740992,"upper":9007199254740993}`),
		[]byte(`{"lower":1.0000000000000000000000000001,"upper":1.0000000000000000000000000002}`),
	} {
		a, c := load(t)
		a.AlarmItems[1].Conditions[0].Params = params
		if err := ValidateProjectArtifact(a, c); err != nil {
			t.Fatalf("exact range rejected %s: %v", params, err)
		}
	}
	a, c := load(t)
	a.AlarmItems[3].Conditions[0].Params = []byte(`{"from":9007199254740992,"to":9007199254740993}`)
	if err := ValidateProjectArtifact(a, c); err != nil {
		t.Fatalf("adjacent transition integers rejected: %v", err)
	}
	a, c = load(t)
	a.AlarmItems[3].Conditions[0].Params = []byte(`{"from":1.0000000000000000000000000001,"to":1.0000000000000000000000000002}`)
	if err := ValidateProjectArtifact(a, c); err != nil {
		t.Fatalf("adjacent transition decimals rejected: %v", err)
	}
	a, c = load(t)
	a.AlarmItems[3].Conditions[0].Params = []byte(`{"from":1,"to":1.0}`)
	if err := ValidateProjectArtifact(a, c); err == nil {
		t.Fatal("numerically equal transition endpoints accepted")
	}
	a, c = load(t)
	a.AlarmItems[1].Conditions[0].Params = []byte(`{"lower":1,"upper":2} trailing`)
	if err := ValidateProjectArtifact(a, c); err == nil {
		t.Fatal("trailing range params accepted")
	}
}

func TestValidatePointAlarmConditionDataTypeCompatibility(t *testing.T) {
	condition := func(kind, operator, params string) AlarmCondition {
		return AlarmCondition{Kind: kind, Operator: operator, Params: json.RawMessage(params), Deadband: json.RawMessage(`0`)}
	}
	for _, test := range []struct {
		name string
		typ  string
		cond AlarmCondition
		ok   bool
	}{
		{"float64 typed adjacent endpoints", "float64", condition("transition", "from_to", `{"from":9007199254740992,"to":9007199254740993}`), true},
		{"string threshold", "string", condition("threshold", "gt", `{"threshold":1}`), false},
		{"numeric text", "int64", condition("text_match", "contains", `{"expected":"x"}`), false},
		{"numeric rising", "int64", condition("transition", "rising", `{}`), false},
		{"state type", "bool", condition("state", "eq", `{"expected":1}`), false},
		{"from_to type", "bool", condition("transition", "from_to", `{"from":false,"to":1}`), false},
		{"from_to int overflow", "int64", condition("transition", "from_to", `{"from":1,"to":9223372036854775808}`), false},
		{"string text positive", "string", condition("text_match", "contains", `{"expected":"error"}`), true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := validatePointAlarmConditionType(test.typ, test.cond) == nil; got != test.ok {
				t.Fatalf("compatibility ok=%v, want %v", got, test.ok)
			}
		})
	}
}

func TestValidateWALCapacity(t *testing.T) {
	if err := ValidateWALCapacity(WALCapacity{MaxBytes: 1024, HighWatermarkBytes: 768, DiagnosticReserveBytes: 64, DataGapPolicy: "emit-alarm-event"}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateWALCapacity(WALCapacity{MaxBytes: 1024, HighWatermarkBytes: 1024, DiagnosticReserveBytes: 64, DataGapPolicy: "emit-alarm-event"}); err == nil {
		t.Fatal("expected high-watermark failure")
	}
}

func TestValidateCollectorIngressMappings(t *testing.T) {
	var artifact ProjectArtifact
	var config EngineConfig
	var collector CollectorArtifact
	modelFixture(t, "runtime-project-artifact.valid.json", &artifact)
	modelFixture(t, "runtime-engine-config.valid.json", &config)
	modelFixture(t, "collector-runtime-artifact.valid.json", &collector)
	artifacts := map[string]CollectorArtifact{"collector-line1-a": collector}
	if err := ValidateCollectorIngressMappings(config, artifact, artifacts); err != nil {
		t.Fatalf("expected valid collector ingress mapping: %v", err)
	}
	mapping := collector.PointMappings[0]
	if err := ValidateRawSourceMapping(artifacts, "collector-line1-a", mapping.DatapointID, mapping.ConnectionID, mapping.VariableID); err != nil {
		t.Fatalf("expected trusted raw source mapping: %v", err)
	}
	if err := ValidateRawSourceMapping(artifacts, "collector-line1-a", mapping.DatapointID, mapping.ConnectionID, "cccccccc-1111-4111-8111-111111111111"); err == nil {
		t.Fatal("expected raw variable mapping rejection")
	}
	collector.PointMappings[0].VariableID = "cccccccc-1111-4111-8111-111111111111"
	if err := ValidateCollectorIngressMappings(config, artifact, map[string]CollectorArtifact{"collector-line1-a": collector}); err == nil || !strings.Contains(err.Error(), "sourceId") {
		t.Fatalf("expected sourceId mismatch, got %v", err)
	}
}
