package compute

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/indu-forge/runtime-engine/internal/eventid"
	"github.com/indu-forge/runtime-engine/internal/ingress"
	"github.com/indu-forge/runtime-engine/internal/loader"
	"github.com/indu-forge/runtime-engine/internal/model"
	"github.com/indu-forge/runtime-engine/internal/store/postgres"
	"github.com/indu-forge/runtime-engine/internal/transportlimits"
)

type Executor interface {
	Execute(context.Context, ExecutionRequest) (ExecutionResult, error)
}

// ErrUnitBusiness marks sandbox execution and output-contract failures that
// are isolated to one scheduled compute unit. Fence, storage and context
// failures deliberately never use this marker.
var ErrUnitBusiness = ingress.ErrRetryableBusiness

// ErrSandboxTimeout 表示 SandboxClient 为单次执行创建的内部 deadline 已到。
// 它不包装 context.DeadlineExceeded，调用方可将其作为单元业务失败继续调度其余单元。
var ErrSandboxTimeout = errors.New("sandbox 调用超时")

type UnitBusinessError struct {
	ComputeID string
	Cause     error
}

func (e *UnitBusinessError) Error() string        { return ErrUnitBusiness.Error() + ": " + e.ComputeID }
func (e *UnitBusinessError) Unwrap() error        { return e.Cause }
func (e *UnitBusinessError) Is(target error) bool { return target == ErrUnitBusiness }

// UnitBusinessFailures lets a host expose DEGRADED without losing the fact
// that independent due units were still processed in this tick.
type UnitBusinessFailures struct{ Failures []*UnitBusinessError }

func (e *UnitBusinessFailures) Error() string { return "compute 存在单元业务失败" }
func (e *UnitBusinessFailures) Unwrap() []error {
	out := make([]error, len(e.Failures))
	for i, failure := range e.Failures {
		out[i] = failure
	}
	return out
}

// Handler owns no in-memory point state.  Every input, edge and debounce value
// it consults is read/updated through the current BusinessTx.
type Handler struct {
	artifact  model.ProjectArtifact
	config    model.EngineConfig
	executor  Executor
	units     map[string]model.ComputeUnit
	producers map[string]model.ProducerAssignment
}

func NewPostgresHandler(artifact model.ProjectArtifact, config model.EngineConfig, executor Executor) (*Handler, error) {
	if executor == nil {
		return nil, errors.New("compute executor 不能为空")
	}
	h := &Handler{artifact: artifact, config: config, executor: executor, units: map[string]model.ComputeUnit{}, producers: map[string]model.ProducerAssignment{}}
	for _, unit := range artifact.ComputeUnits {
		for _, input := range unit.Inputs {
			if input.Alias == "true" || input.Alias == "false" {
				return nil, errors.New("compute input alias 不得覆盖 bool literal")
			}
		}
		h.units[unit.ID] = unit
	}
	for _, producer := range config.ProducerAssignments {
		if producer.ProducerType == "compute" {
			h.producers[producer.ComputeID] = producer
		}
	}
	return h, nil
}
func (h *Handler) PostgresHandler() ingress.PostgresHandler { return h.HandlePostgres }

func (h *Handler) HandlePostgres(ctx context.Context, tx *postgres.BusinessTx, message ingress.ValidatedMessage) error {
	if h == nil || tx == nil || message.Consumer.Role != "compute" {
		return errors.New("compute role fence 不匹配")
	}
	for _, unit := range h.units {
		producer, assigned := h.producers[unit.ID]
		if !assigned || !unit.Enabled || producer.Role != "compute" {
			continue
		}
		input, isInput := findInput(unit.Inputs, message.Event.PointID)
		if !isInput {
			continue
		}
		if !matchesDataType(message.Event.Value, input.DataType) {
			return errors.New("compute 输入类型不匹配")
		}
		sourceAt, err := strictUTCTime(message.Event.SourceTimestamp)
		if err != nil {
			return err
		}
		serverAt, err := strictUTCTime(message.Event.ServerTimestamp)
		if err != nil {
			return err
		}
		receivedAt, err := strictUTCTime(message.Event.ReceivedAt)
		if err != nil {
			return err
		}
		result, err := tx.UpsertComputeInput(ctx, postgres.ComputeInputSnapshot{DeploymentID: message.Event.DeploymentID, ComputeID: unit.ID, DatapointID: input.DatapointID, EventID: message.Event.EventID, OwnerID: message.Event.OwnerID, Epoch: message.Event.Epoch, Sequence: message.Event.Sequence, SourceTimestamp: sourceAt, ServerTimestamp: serverAt, ReceivedAt: receivedAt, Value: message.Event.Value, Quality: message.Event.Quality})
		if err != nil {
			return err
		}
		if !result.Applied {
			continue
		}
		fire, err := h.evaluateTrigger(ctx, tx, unit, message)
		if err != nil {
			return err
		}
		if !fire {
			continue
		}
		if err := h.executeAndQueue(ctx, tx, unit, producer, message); err != nil {
			return err
		}
	}
	return nil
}

// RunDue schedules only units owned by this compute role. It is safe to call
// repeatedly and after restart: schedule state is locked and persisted in the
// same transaction as output sequence/outbox writes.
func (h *Handler) RunDue(ctx context.Context, store *postgres.Store, token postgres.ConsumerRoleToken, now time.Time) error {
	if h == nil || store == nil {
		return errors.New("compute scheduler 依赖非法")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	var failures []*UnitBusinessError
	for _, unit := range h.sortedUnits(func(unit model.ComputeUnit) bool {
		return unit.Trigger.Kind == "schedule" && unit.Trigger.Schedule != nil
	}) {
		if err := ctx.Err(); err != nil {
			return err
		}
		producer, assigned := h.producers[unit.ID]
		if !assigned || !unit.Enabled || unit.Trigger.Kind != "schedule" || unit.Trigger.Schedule == nil {
			continue
		}
		unit, producer := unit, producer
		err := store.ProcessScheduled(ctx, postgres.ScheduledMessage{DeploymentID: h.config.DeploymentID, Role: "compute", ComputeID: unit.ID, Token: token, ProducerKey: unit.ID, ProducerToken: postgres.ProducerToken{OwnerID: producer.Ownership.OwnerID, Epoch: producer.Ownership.Epoch}, OccurredAt: now.UTC()}, func(ctx context.Context, tx *postgres.BusinessTx) error {
			state, err := tx.ComputeScheduleState(ctx, unit.ID, unit.Revision)
			if err != nil {
				return err
			}
			if state.NextRunAt == nil {
				next, key, err := InitialScheduleOccurrence(*unit.Trigger.Schedule, now)
				if err != nil {
					return err
				}
				if next.IsZero() {
					return tx.SaveComputeScheduleState(ctx, unit.ID, state)
				}
				if next.IsZero() {
					state.NextRunAt = nil
				} else {
					state.NextRunAt = &next
				}
				_ = key
			}
			if state.NextRunAt == nil || state.NextRunAt.After(now.UTC()) {
				return tx.SaveComputeScheduleState(ctx, unit.ID, state)
			}
			if unit.Trigger.Schedule.MaxRuns != nil && state.RunCount >= int64(*unit.Trigger.Schedule.MaxRuns) {
				state.NextRunAt = nil
				return tx.SaveComputeScheduleState(ctx, unit.ID, state)
			}
			occurrence := state.NextRunAt.UTC()
			_, key, err := NextSchedule(*unit.Trigger.Schedule, occurrence.Add(-time.Nanosecond))
			if err != nil && unit.Trigger.Schedule.Kind == "interval" && unit.Trigger.Schedule.StartAt == nil {
				key = occurrence.Format(time.RFC3339Nano)
			} else if err != nil {
				return err
			}
			nextOccurrence := func() (time.Time, error) {
				if unit.Trigger.Schedule.Kind == "interval" && unit.Trigger.Schedule.StartAt == nil {
					return occurrence.Add(intervalDuration(*unit.Trigger.Schedule.Every, *unit.Trigger.Schedule.Unit)), nil
				}
				next, _, err := NextSchedule(*unit.Trigger.Schedule, occurrence)
				return next, err
			}
			if key == state.LastLocalKey {
				next, err := nextOccurrence()
				if err != nil {
					return err
				}
				state.NextRunAt = &next
				return tx.SaveComputeScheduleState(ctx, unit.ID, state)
			}
			runID, err := eventid.HashFields("compute.schedule.v1", h.config.DeploymentID, unit.ID, key)
			if err != nil {
				return err
			}
			synthetic := ingress.ValidatedMessage{Event: ingress.Event{EventID: runID, SourceTimestamp: occurrence.Format(time.RFC3339Nano)}, OccurredAt: occurrence}
			if err = h.executeAndQueue(ctx, tx, unit, producer, synthetic); err != nil {
				return err
			}
			state.RunCount++
			state.LastRunAt = &occurrence
			state.LastLocalKey = key
			next, err := nextOccurrence()
			if err != nil {
				return err
			}
			if next.IsZero() {
				state.NextRunAt = nil
			} else {
				state.NextRunAt = &next
			}
			return tx.SaveComputeScheduleState(ctx, unit.ID, state)
		})
		if err != nil {
			// 外层取消和执行器直接返回的 context 错误均不可被降级为单元业务失败。
			if outerErr := ctx.Err(); outerErr != nil {
				return outerErr
			}
			if isContextError(err) {
				return err
			}
			var failure *UnitBusinessError
			if errors.As(err, &failure) {
				failures = append(failures, failure)
				continue
			}
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	if len(failures) > 0 {
		return &UnitBusinessFailures{Failures: failures}
	}
	return nil
}

// SweepDebounces releases a persisted datapoint-change debounce even when the
// source remains quiet. Hosts call it on their bounded scheduler tick; it uses
// the same role/producer fenced transaction as ingress and never needs an
// in-memory timer to survive restart.
func (h *Handler) SweepDebounces(ctx context.Context, store *postgres.Store, token postgres.ConsumerRoleToken, now time.Time) error {
	if h == nil || store == nil {
		return errors.New("compute debounce sweeper 依赖非法")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	var failures []*UnitBusinessError
	for _, unit := range h.sortedUnits(func(unit model.ComputeUnit) bool { return unit.Trigger.Kind == "datapoint_change" }) {
		if err := ctx.Err(); err != nil {
			return err
		}
		producer, ok := h.producers[unit.ID]
		if !ok || !unit.Enabled || unit.Trigger.Kind != "datapoint_change" {
			continue
		}
		unit, producer := unit, producer
		err := store.ProcessScheduled(ctx, postgres.ScheduledMessage{DeploymentID: h.config.DeploymentID, Role: "compute", ComputeID: unit.ID, Token: token, ProducerKey: unit.ID, ProducerToken: postgres.ProducerToken{OwnerID: producer.Ownership.OwnerID, Epoch: producer.Ownership.Epoch}, OccurredAt: now.UTC()}, func(ctx context.Context, tx *postgres.BusinessTx) error {
			state, err := tx.ComputeTriggerState(ctx, unit.ID, unit.Revision)
			if err != nil {
				return err
			}
			delay := time.Duration(0)
			if unit.Trigger.DebounceMS != nil {
				delay = time.Duration(*unit.Trigger.DebounceMS) * time.Millisecond
			}
			if state.PendingPhase != "entered" || state.PendingSince == nil || now.UTC().Sub(state.PendingSince.UTC()) < delay {
				return nil
			}
			pendingAt := state.PendingSince.UTC()
			state.PendingPhase = ""
			state.PendingSince = nil
			if err = tx.SaveComputeTriggerState(ctx, unit.ID, state); err != nil {
				return err
			}
			runID, err := debounceRunID(h.config.DeploymentID, unit, pendingAt)
			if err != nil {
				return err
			}
			return h.executeAndQueue(ctx, tx, unit, producer, ingress.ValidatedMessage{Event: ingress.Event{EventID: runID, SourceTimestamp: pendingAt.Format(time.RFC3339Nano)}, OccurredAt: pendingAt})
		})
		if err != nil {
			// 与 RunDue 保持相同的终止语义，避免 canceled/deadline 被错误聚合。
			if outerErr := ctx.Err(); outerErr != nil {
				return outerErr
			}
			if isContextError(err) {
				return err
			}
			var failure *UnitBusinessError
			if errors.As(err, &failure) {
				failures = append(failures, failure)
				continue
			}
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	if len(failures) > 0 {
		return &UnitBusinessFailures{Failures: failures}
	}
	return nil
}

func (h *Handler) sortedUnits(include func(model.ComputeUnit) bool) []model.ComputeUnit {
	ids := make([]string, 0, len(h.units))
	for id, unit := range h.units {
		if include(unit) {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	result := make([]model.ComputeUnit, 0, len(ids))
	for _, id := range ids {
		result = append(result, h.units[id])
	}
	return result
}

// debounceRunID is derived solely from persisted trigger facts.  A failed
// transaction or process restart can retry at a different wall time while
// retaining the sandbox execution identity for the same pending transition.
func debounceRunID(deploymentID string, unit model.ComputeUnit, pendingSince time.Time) (string, error) {
	return eventid.HashFields("compute.debounce.v1", deploymentID, unit.ID, fmt.Sprintf("%d", unit.Revision), pendingSince.UTC().Format(time.RFC3339Nano))
}

func findInput(inputs []model.Input, point string) (model.Input, bool) {
	for _, input := range inputs {
		if input.DatapointID == point {
			return input, true
		}
	}
	return model.Input{}, false
}
func strictUTCTime(raw string) (time.Time, error) {
	if !strings.HasSuffix(raw, "Z") {
		return time.Time{}, errors.New("compute 输入时间非法")
	}
	value, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}, errors.New("compute 输入时间非法")
	}
	return value.UTC(), nil
}

func (h *Handler) evaluateTrigger(ctx context.Context, tx *postgres.BusinessTx, unit model.ComputeUnit, message ingress.ValidatedMessage) (bool, error) {
	state, err := tx.ComputeTriggerState(ctx, unit.ID, unit.Revision)
	if err != nil {
		return false, err
	}
	now := message.OccurredAt.UTC()
	trigger := unit.Trigger
	switch trigger.Kind {
	case "manual", "schedule":
		return false, nil
	case "datapoint_change":
		if message.Event.PointID != trigger.DatapointID {
			return false, nil
		}
		band := trigger.Deadband
		changed := ChangeTriggered(trigger.Mode, band, ChangeState{Value: state.PreviousValue, Quality: state.PreviousQuality, Seen: state.PreviousSeen}, message.Event.Value, message.Event.Quality)
		state.PreviousValue = append(state.PreviousValue[:0], message.Event.Value...)
		state.PreviousQuality = message.Event.Quality
		state.PreviousSeen = true
		fire := false
		if changed {
			delay := time.Duration(0)
			if trigger.DebounceMS != nil {
				delay = time.Duration(*trigger.DebounceMS) * time.Millisecond
			}
			if delay == 0 {
				fire = true
				state.PendingPhase = ""
				state.PendingSince = nil
			} else if state.PendingPhase != "entered" || state.PendingSince == nil {
				state.PendingPhase = "entered"
				at := now
				state.PendingSince = &at
			}
		} else if state.PendingPhase == "entered" && state.PendingSince != nil {
			delay := time.Duration(0)
			if trigger.DebounceMS != nil {
				delay = time.Duration(*trigger.DebounceMS) * time.Millisecond
			}
			if now.Sub(state.PendingSince.UTC()) >= delay {
				fire = true
				state.PendingPhase = ""
				state.PendingSince = nil
			}
		}
		return fire, tx.SaveComputeTriggerState(ctx, unit.ID, state)
	case "condition":
		values, complete, err := h.inputValues(ctx, tx, unit)
		if err != nil {
			return false, err
		}
		if !complete {
			return false, nil
		}
		active, err := EvaluateCondition(trigger.Expression, values)
		if err != nil {
			return false, err
		}
		delay := time.Duration(0)
		if trigger.DebounceMS != nil {
			delay = time.Duration(*trigger.DebounceMS) * time.Millisecond
		}
		phase, pending, since := DebouncedPhase(active, state.ConditionActive, state.PendingPhase, state.PendingSince, now, delay)
		state.PendingPhase, state.PendingSince = pending, since
		if phase == "entered" {
			state.ConditionActive = true
		}
		if phase == "exited" {
			state.ConditionActive = false
		}
		if err := tx.SaveComputeTriggerState(ctx, unit.ID, state); err != nil {
			return false, err
		}
		return phase != "" && contains(trigger.Phases, phase), nil
	default:
		return false, errors.New("未知 compute trigger")
	}
}

func (h *Handler) inputValues(ctx context.Context, tx *postgres.BusinessTx, unit model.ComputeUnit) (map[string]json.RawMessage, bool, error) {
	ids := make([]string, len(unit.Inputs))
	for i, input := range unit.Inputs {
		ids[i] = input.DatapointID
	}
	snapshots, err := tx.ComputeInputs(ctx, unit.ID, ids)
	if err != nil {
		return nil, false, err
	}
	values := make(map[string]json.RawMessage, len(unit.Inputs))
	for _, input := range unit.Inputs {
		snapshot, ok := snapshots[input.DatapointID]
		if !ok {
			return nil, false, nil
		}
		values[input.Alias] = append([]byte(nil), snapshot.Value...)
	}
	return values, true, nil
}
func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func isContextError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func (h *Handler) executeAndQueue(ctx context.Context, tx *postgres.BusinessTx, unit model.ComputeUnit, producer model.ProducerAssignment, message ingress.ValidatedMessage) error {
	if err := tx.VerifyComputeProducer(ctx, unit.ID, postgres.ProducerToken{OwnerID: producer.Ownership.OwnerID, Epoch: producer.Ownership.Epoch}); err != nil {
		return err
	}
	values, complete, err := h.inputValues(ctx, tx, unit)
	if err != nil {
		return err
	}
	if !complete {
		return nil
	}
	result, err := h.executor.Execute(ctx, ExecutionRequest{ExecutionID: ExecutionUUID(message.Event.EventID), DeploymentID: h.config.DeploymentID, ProjectID: h.config.ProjectID, ComputeUnitID: unit.ID, ArtifactDigest: h.config.ProjectArtifact.ArtifactDigest, ComputeRevision: unit.Revision, Input: values, Timeout: time.Duration(unit.TimeoutMS) * time.Millisecond})
	// Execute 返回成功后仍须检查外层生命周期，避免在取消窗口继续校验并落库。
	if outerErr := ctx.Err(); outerErr != nil {
		return outerErr
	}
	if err != nil {
		if isContextError(err) {
			return err
		}
		return &UnitBusinessError{ComputeID: unit.ID, Cause: err}
	}
	outputs, err := validateOutputs(unit, result.Output)
	if outerErr := ctx.Err(); outerErr != nil {
		return outerErr
	}
	if err != nil {
		return &UnitBusinessError{ComputeID: unit.ID, Cause: err}
	}
	inputIDs := make([]string, 0, len(unit.Inputs))
	snapshots, err := tx.ComputeInputs(ctx, unit.ID, inputDatapointIDs(unit.Inputs))
	if err != nil {
		return err
	}
	for _, input := range unit.Inputs {
		snapshot, ok := snapshots[input.DatapointID]
		if !ok {
			return errors.New("compute 输入快照不完整")
		}
		inputIDs = append(inputIDs, snapshot.EventID)
	}
	sort.Strings(inputIDs)
	now := time.Now().UTC()
	// The sandbox controls value bytes.  Build every envelope at the largest
	// possible sequence width before consuming a sequence, so a too-large
	// result rolls back the whole business transaction without an output gap.
	for _, output := range unit.Outputs {
		value, emit := outputs[output.OutputKey]
		if !emit {
			continue
		}
		if _, err := h.computedPayload(unit, output, producer, message, inputIDs, math.MaxInt64, strings.Repeat("f", 64), value, now); err != nil {
			return &UnitBusinessError{ComputeID: unit.ID, Cause: err}
		}
	}
	for _, output := range unit.Outputs {
		value, emit := outputs[output.OutputKey]
		if !emit {
			continue
		}
		sequence, err := tx.NextComputeSequence(ctx, unit.ID, postgres.ProducerToken{OwnerID: producer.Ownership.OwnerID, Epoch: producer.Ownership.Epoch})
		if err != nil {
			return err
		}
		id, err := eventid.Computed("data.computed.v1", h.config.DeploymentID, output.DatapointID, producer.Ownership.OwnerID, producer.Ownership.Epoch, sequence, unit.ID, unit.Revision, inputIDs)
		if err != nil {
			return err
		}
		payload, err := h.computedPayload(unit, output, producer, message, inputIDs, sequence, id, value, now)
		if err != nil {
			// The preflight above proves this cannot be a size failure for this
			// sequence. Keep the defensive check nevertheless for future fields.
			return &UnitBusinessError{ComputeID: unit.ID, Cause: err}
		}
		if _, _, err = tx.Enqueue(ctx, postgres.OutboxMessage{DeploymentID: h.config.DeploymentID, DedupeKey: id, Subject: "data.computed." + output.DatapointID, Payload: payload}); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) computedPayload(unit model.ComputeUnit, output model.Output, producer model.ProducerAssignment, message ingress.ValidatedMessage, inputIDs []string, sequence int64, eventID string, value json.RawMessage, now time.Time) ([]byte, error) {
	subject := "data.computed." + output.DatapointID
	payload, err := json.Marshal(ingress.Event{SchemaVersion: "data.computed.v1", Subject: subject, EventID: eventID, DeploymentID: h.config.DeploymentID, AccountID: h.config.AccountID, PointID: output.DatapointID, OwnerID: producer.Ownership.OwnerID, Epoch: producer.Ownership.Epoch, Sequence: sequence, Value: value, Quality: "good", SourceTimestamp: message.Event.SourceTimestamp, ServerTimestamp: now.Format(time.RFC3339Nano), ReceivedAt: now.Format(time.RFC3339Nano), Computation: &ingress.Computation{ComputeID: unit.ID, ComputeRevision: unit.Revision, InputEventIDs: inputIDs}})
	if err != nil {
		return nil, errors.New("computed payload 非法")
	}
	if err := transportlimits.ValidateOutboundPayload(subject, payload); err != nil {
		return nil, err
	}
	if loader.ValidatePointEvent(payload) != nil {
		return nil, errors.New("computed payload 非法")
	}
	return payload, nil
}

func inputDatapointIDs(inputs []model.Input) []string {
	ids := make([]string, len(inputs))
	for i, input := range inputs {
		ids[i] = input.DatapointID
	}
	return ids
}

// validateOutputs rejects missing, unknown and wrongly typed values before any
// sequence allocation.  A sandbox's scalar response is accepted only for one
// declared output; multi-output units must use exact outputKey mapping.
func validateOutputs(unit model.ComputeUnit, raw json.RawMessage) (map[string]json.RawMessage, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		if len(unit.Outputs) != 1 {
			return nil, errors.New("compute 输出必须为完整映射")
		}
		object = map[string]json.RawMessage{unit.Outputs[0].OutputKey: raw}
	}
	if len(object) != len(unit.Outputs) {
		return nil, errors.New("compute 输出数量不匹配")
	}
	result := map[string]json.RawMessage{}
	for _, output := range unit.Outputs {
		value, ok := object[output.OutputKey]
		if !ok {
			return nil, errors.New("compute 输出映射不匹配")
		}
		if string(value) == "null" {
			switch output.NullPolicy {
			case "skip":
				continue
			case "default":
				if len(output.DefaultValue) == 0 || string(output.DefaultValue) == "null" {
					return nil, errors.New("compute default nullPolicy 缺少非空 defaultValue")
				}
				value = output.DefaultValue
			case "propagate":
			default:
				return nil, errors.New("compute null policy 非法")
			}
		}
		if !matchesDataType(value, output.DataType) {
			return nil, fmt.Errorf("compute 输出 %s 类型不匹配", output.OutputKey)
		}
		result[output.OutputKey] = append([]byte(nil), value...)
	}
	return result, nil
}
func matchesDataType(raw json.RawMessage, typ string) bool {
	return model.ValidateDataPointValue(raw, typ, true) == nil
}
