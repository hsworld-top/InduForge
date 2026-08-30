package compute

import (
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/indu-forge/runtime-engine/internal/ingress"
	"github.com/indu-forge/runtime-engine/internal/model"
	"github.com/indu-forge/runtime-engine/internal/transportlimits"
)

func TestComputedPayloadBoundaryIncludesEnvelopeAndLargeString(t *testing.T) {
	var artifact model.ProjectArtifact
	var config model.EngineConfig
	computeFixture(t, "runtime-project-artifact.valid.json", &artifact)
	computeFixture(t, "runtime-engine-config.valid.json", &config)
	unit := artifact.ComputeUnits[2]
	unit.Outputs[0].DataType = "string"
	h := &Handler{config: config}
	producer := model.ProducerAssignment{Ownership: model.Ownership{OwnerID: "runtime-engine-compute-0", Epoch: 7}}
	message := ingress.ValidatedMessage{Event: ingress.Event{SourceTimestamp: "2026-08-30T08:30:00Z"}}
	ids := []string{strings.Repeat("a", 64)}
	now := time.Date(2026, 8, 30, 8, 30, 0, 0, time.UTC)
	base, err := h.computedPayload(unit, unit.Outputs[0], producer, message, ids, math.MaxInt64, strings.Repeat("f", 64), json.RawMessage(`""`), now)
	if err != nil {
		t.Fatal(err)
	}
	value := json.RawMessage(`"` + strings.Repeat("x", transportlimits.MaxBodyBytes-len(base)) + `"`)
	payload, err := h.computedPayload(unit, unit.Outputs[0], producer, message, ids, math.MaxInt64, strings.Repeat("f", 64), value, now)
	if err != nil || len(payload) != transportlimits.MaxBodyBytes {
		t.Fatalf("exact envelope boundary len=%d err=%v", len(payload), err)
	}
	value = json.RawMessage(`"` + strings.Repeat("x", transportlimits.MaxBodyBytes-len(base)+1) + `"`)
	if _, err := h.computedPayload(unit, unit.Outputs[0], producer, message, ids, math.MaxInt64, strings.Repeat("f", 64), value, now); !errors.Is(err, transportlimits.ErrOutboundPayloadTooLarge) {
		t.Fatalf("oversized envelope error=%v", err)
	}
}

func TestComputedPayloadBoundaryIncludesLargeJSONObject(t *testing.T) {
	var artifact model.ProjectArtifact
	var config model.EngineConfig
	computeFixture(t, "runtime-project-artifact.valid.json", &artifact)
	computeFixture(t, "runtime-engine-config.valid.json", &config)
	unit := artifact.ComputeUnits[2]
	unit.Outputs[0].DataType = "json"
	h := &Handler{config: config}
	producer := model.ProducerAssignment{Ownership: model.Ownership{OwnerID: "runtime-engine-compute-0", Epoch: 7}}
	message := ingress.ValidatedMessage{Event: ingress.Event{SourceTimestamp: "2026-08-30T08:30:00Z"}}
	now := time.Date(2026, 8, 30, 8, 30, 0, 0, time.UTC)
	base, err := h.computedPayload(unit, unit.Outputs[0], producer, message, []string{strings.Repeat("a", 64)}, math.MaxInt64, strings.Repeat("f", 64), json.RawMessage(`{"v":""}`), now)
	if err != nil {
		t.Fatal(err)
	}
	value := json.RawMessage(`{"v":"` + strings.Repeat("x", transportlimits.MaxBodyBytes-len(base)) + `"}`)
	payload, err := h.computedPayload(unit, unit.Outputs[0], producer, message, []string{strings.Repeat("a", 64)}, math.MaxInt64, strings.Repeat("f", 64), value, now)
	if err != nil || len(payload) != transportlimits.MaxBodyBytes {
		t.Fatalf("exact json envelope boundary len=%d err=%v", len(payload), err)
	}
	value = json.RawMessage(`{"v":"` + strings.Repeat("x", transportlimits.MaxBodyBytes-len(base)+1) + `"}`)
	if _, err := h.computedPayload(unit, unit.Outputs[0], producer, message, []string{strings.Repeat("a", 64)}, math.MaxInt64, strings.Repeat("f", 64), value, now); !errors.Is(err, transportlimits.ErrOutboundPayloadTooLarge) {
		t.Fatalf("oversized json envelope error=%v", err)
	}
}
