package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
)

// ComputeInputSnapshot is the compute role's durable, independently ordered
// input view.  Its EventID is later included in computed-event identity.
type ComputeInputSnapshot struct {
	DeploymentID, ComputeID, DatapointID, EventID, OwnerID, Quality string
	Epoch, Sequence                                                 int64
	SourceTimestamp, ServerTimestamp, ReceivedAt                    time.Time
	Value                                                           json.RawMessage
}

type ComputeInputResult struct {
	Applied bool
	Version int64
}

type ComputeCommandAudit struct {
	DeploymentID, CommandID, ComputeID, RequestedBy, IdempotencyKey, Status string
	RequestedAt                                                             time.Time
	BindingEpoch, ResultVersion                                             int64
	ResultEventIDs                                                          json.RawMessage
	FailureCode                                                             string
}

// QueueComputeCommand 由 Runtime API 在发布前写入；同 idempotencyKey 的重试返回
// 原记录，调用者据此禁止发布第二条命令。
func (s *Store) QueueComputeCommand(ctx context.Context, audit ComputeCommandAudit) (ComputeCommandAudit, bool, error) {
	if s == nil || s.pool == nil || !validCommandAudit(audit) || audit.Status != "queued" {
		return ComputeCommandAudit{}, false, ErrInvalidInput
	}
	var existing ComputeCommandAudit
	err := s.pool.QueryRow(ctx, `INSERT INTO runtime_engine.compute_command_audit(deployment_id,command_id,compute_id,requested_by,requested_at,binding_epoch,idempotency_key,status)
VALUES ($1,$2::uuid,$3::uuid,$4,$5,$6,$7,'queued')
ON CONFLICT (deployment_id,idempotency_key) DO NOTHING
RETURNING deployment_id,command_id::text,compute_id::text,requested_by,requested_at,binding_epoch,idempotency_key,status,result_event_ids,result_version,COALESCE(failure_code,'')`, audit.DeploymentID, audit.CommandID, audit.ComputeID, audit.RequestedBy, audit.RequestedAt.UTC(), audit.BindingEpoch, audit.IdempotencyKey).Scan(&existing.DeploymentID, &existing.CommandID, &existing.ComputeID, &existing.RequestedBy, &existing.RequestedAt, &existing.BindingEpoch, &existing.IdempotencyKey, &existing.Status, &existing.ResultEventIDs, &existing.ResultVersion, &existing.FailureCode)
	if err == nil {
		return existing, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return ComputeCommandAudit{}, false, err
	}
	err = s.pool.QueryRow(ctx, `SELECT deployment_id,command_id::text,compute_id::text,requested_by,requested_at,binding_epoch,idempotency_key,status,result_event_ids,result_version,COALESCE(failure_code,'') FROM runtime_engine.compute_command_audit WHERE deployment_id=$1 AND idempotency_key=$2`, audit.DeploymentID, audit.IdempotencyKey).Scan(&existing.DeploymentID, &existing.CommandID, &existing.ComputeID, &existing.RequestedBy, &existing.RequestedAt, &existing.BindingEpoch, &existing.IdempotencyKey, &existing.Status, &existing.ResultEventIDs, &existing.ResultVersion, &existing.FailureCode)
	return existing, false, err
}

func (s *Store) SetComputeCommandStatus(ctx context.Context, deploymentID, commandID, status, failureCode string) error {
	if s == nil || s.pool == nil || deploymentID == "" || !canonicalPointUUID.MatchString(commandID) || (status != "running" && status != "succeeded" && status != "failed") || (status == "failed") != (failureCode != "") {
		return ErrInvalidInput
	}
	var previous string
	err := s.pool.QueryRow(ctx, `UPDATE runtime_engine.compute_command_audit SET status=$3,failure_code=NULLIF($4,''),updated_at=now() WHERE deployment_id=$1 AND command_id=$2::uuid AND ((status='queued' AND $3 IN ('running','failed')) OR (status='running' AND $3 IN ('succeeded','failed')) OR status=$3) RETURNING status`, deploymentID, commandID, status, failureCode).Scan(&previous)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrFenceStale
	}
	return err
}

func (s *Store) ComputeCommandStatus(ctx context.Context, deploymentID, commandID string) (ComputeCommandAudit, error) {
	if s == nil || s.pool == nil || deploymentID == "" || !canonicalPointUUID.MatchString(commandID) {
		return ComputeCommandAudit{}, ErrInvalidInput
	}
	var audit ComputeCommandAudit
	err := s.pool.QueryRow(ctx, `SELECT deployment_id,command_id::text,compute_id::text,requested_by,requested_at,binding_epoch,idempotency_key,status,result_event_ids,result_version,COALESCE(failure_code,'') FROM runtime_engine.compute_command_audit WHERE deployment_id=$1 AND command_id=$2::uuid`, deploymentID, commandID).Scan(&audit.DeploymentID, &audit.CommandID, &audit.ComputeID, &audit.RequestedBy, &audit.RequestedAt, &audit.BindingEpoch, &audit.IdempotencyKey, &audit.Status, &audit.ResultEventIDs, &audit.ResultVersion, &audit.FailureCode)
	return audit, err
}

func validCommandAudit(a ComputeCommandAudit) bool {
	return a.DeploymentID != "" && canonicalPointUUID.MatchString(a.CommandID) && canonicalPointUUID.MatchString(a.ComputeID) && a.RequestedBy != "" && !a.RequestedAt.IsZero() && a.BindingEpoch > 0 && len(a.IdempotencyKey) >= 16 && len(a.IdempotencyKey) <= 128
}

// UpsertComputeInput applies the same fact ordering as point_current, but in
// compute's own projection.  A writer consumer can therefore be delayed or
// disabled without affecting calculation correctness.
func (b *BusinessTx) UpsertComputeInput(ctx context.Context, input ComputeInputSnapshot) (ComputeInputResult, error) {
	if b == nil || b.tx == nil || input.DeploymentID != b.deploymentID ||
		!validPointWrite(input.DeploymentID, input.DatapointID, input.EventID, input.OwnerID, input.Quality, input.Epoch, input.Sequence, input.SourceTimestamp, input.ServerTimestamp, input.Value) || input.ReceivedAt.IsZero() || !canonicalPointUUID.MatchString(input.ComputeID) {
		return ComputeInputResult{}, ErrInvalidInput
	}
	if err := b.ensureOpen(); err != nil {
		return ComputeInputResult{}, err
	}
	for attempt := 0; attempt < 2; attempt++ {
		var current ComputeInputSnapshot
		var version int64
		err := b.tx.QueryRow(ctx, `SELECT event_id,owner_id,epoch,sequence,source_timestamp,server_timestamp,received_at,value,quality,version
FROM runtime_engine.compute_input_snapshot WHERE deployment_id=$1 AND compute_id=$2::uuid AND datapoint_id=$3::uuid FOR UPDATE`, input.DeploymentID, input.ComputeID, input.DatapointID).
			Scan(&current.EventID, &current.OwnerID, &current.Epoch, &current.Sequence, &current.SourceTimestamp, &current.ServerTimestamp, &current.ReceivedAt, &current.Value, &current.Quality, &version)
		if errors.Is(err, pgx.ErrNoRows) {
			var inserted int64
			err = b.tx.QueryRow(ctx, `INSERT INTO runtime_engine.compute_input_snapshot
(deployment_id,compute_id,datapoint_id,event_id,owner_id,epoch,sequence,source_timestamp,server_timestamp,received_at,value,quality,version)
VALUES ($1,$2::uuid,$3::uuid,$4,$5,$6,$7,$8,$9,$10,$11::jsonb,$12,1) ON CONFLICT DO NOTHING RETURNING version`,
				input.DeploymentID, input.ComputeID, input.DatapointID, input.EventID, input.OwnerID, input.Epoch, input.Sequence, input.SourceTimestamp.UTC(), input.ServerTimestamp.UTC(), input.ReceivedAt.UTC(), nullableJSON(input.Value), input.Quality).Scan(&inserted)
			if err == nil {
				return ComputeInputResult{Applied: true, Version: inserted}, nil
			}
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return ComputeInputResult{}, err
		}
		if err != nil {
			return ComputeInputResult{}, err
		}
		current.DeploymentID, current.ComputeID, current.DatapointID = input.DeploymentID, input.ComputeID, input.DatapointID
		if compareComputeInput(input, current) <= 0 {
			return ComputeInputResult{Version: version}, nil
		}
		var next int64
		err = b.tx.QueryRow(ctx, `UPDATE runtime_engine.compute_input_snapshot SET event_id=$4,owner_id=$5,epoch=$6,sequence=$7,source_timestamp=$8,server_timestamp=$9,received_at=$10,value=$11::jsonb,quality=$12,version=version+1,updated_at=now()
WHERE deployment_id=$1 AND compute_id=$2::uuid AND datapoint_id=$3::uuid AND version=$13 RETURNING version`, input.DeploymentID, input.ComputeID, input.DatapointID, input.EventID, input.OwnerID, input.Epoch, input.Sequence, input.SourceTimestamp.UTC(), input.ServerTimestamp.UTC(), input.ReceivedAt.UTC(), nullableJSON(input.Value), input.Quality, version).Scan(&next)
		if err == nil {
			return ComputeInputResult{Applied: true, Version: next}, nil
		}
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		return ComputeInputResult{}, err
	}
	return ComputeInputResult{}, ErrFenceStale
}

func compareComputeInput(next, current ComputeInputSnapshot) int {
	return comparePointOrder(PointCurrentWrite{Epoch: next.Epoch, Sequence: next.Sequence, SourceTimestamp: next.SourceTimestamp, EventID: next.EventID}, PointCurrentWrite{Epoch: current.Epoch, Sequence: current.Sequence, SourceTimestamp: current.SourceTimestamp, EventID: current.EventID})
}

// ComputeInputs returns only the requested inputs; callers must reject a
// partial result before invoking a sandbox.
func (b *BusinessTx) ComputeInputs(ctx context.Context, computeID string, datapointIDs []string) (map[string]ComputeInputSnapshot, error) {
	if b == nil || b.tx == nil || !canonicalPointUUID.MatchString(computeID) {
		return nil, ErrInvalidInput
	}
	if err := b.ensureOpen(); err != nil {
		return nil, err
	}
	result := make(map[string]ComputeInputSnapshot, len(datapointIDs))
	for _, id := range datapointIDs {
		if !canonicalPointUUID.MatchString(id) {
			return nil, ErrInvalidInput
		}
		var value ComputeInputSnapshot
		err := b.tx.QueryRow(ctx, `SELECT datapoint_id,event_id,owner_id,epoch,sequence,source_timestamp,server_timestamp,received_at,value,quality
FROM runtime_engine.compute_input_snapshot WHERE deployment_id=$1 AND compute_id=$2::uuid AND datapoint_id=$3::uuid FOR UPDATE`, b.deploymentID, computeID, id).
			Scan(&value.DatapointID, &value.EventID, &value.OwnerID, &value.Epoch, &value.Sequence, &value.SourceTimestamp, &value.ServerTimestamp, &value.ReceivedAt, &value.Value, &value.Quality)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, err
		}
		value.DeploymentID, value.ComputeID = b.deploymentID, computeID
		result[id] = value
	}
	return result, nil
}

// RefreshComputeInputsFromCurrent 为手工计算在同一业务事务内读取 point_current
// 并同步到 compute 输入视图。它拒绝缺失和 bad 质量，避免用陈旧 Artifact 快照运行。
func (b *BusinessTx) RefreshComputeInputsFromCurrent(ctx context.Context, computeID string, datapointIDs []string) error {
	if b == nil || b.tx == nil || !canonicalPointUUID.MatchString(computeID) || len(datapointIDs) == 0 {
		return ErrInvalidInput
	}
	if err := b.ensureOpen(); err != nil {
		return err
	}
	for _, pointID := range datapointIDs {
		if !canonicalPointUUID.MatchString(pointID) {
			return ErrInvalidInput
		}
		var input ComputeInputSnapshot
		err := b.tx.QueryRow(ctx, `SELECT owner_id,epoch,source_timestamp,server_timestamp,sequence,event_id,value,quality
FROM runtime_engine.point_current WHERE deployment_id=$1 AND point_id=$2::uuid FOR SHARE`, b.deploymentID, pointID).
			Scan(&input.OwnerID, &input.Epoch, &input.SourceTimestamp, &input.ServerTimestamp, &input.Sequence, &input.EventID, &input.Value, &input.Quality)
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("manual compute 输入点位不存在")
		}
		if err != nil {
			return err
		}
		if input.Quality == "bad" {
			return fmt.Errorf("manual compute 输入点位质量为 bad")
		}
		input.DeploymentID, input.ComputeID, input.DatapointID, input.ReceivedAt = b.deploymentID, computeID, pointID, input.ServerTimestamp.UTC()
		if _, err = b.UpsertComputeInput(ctx, input); err != nil {
			return err
		}
	}
	return nil
}

type ComputeTriggerState struct {
	ComputeRevision int64
	PreviousValue   json.RawMessage
	PreviousQuality string
	PreviousSeen    bool
	ConditionActive bool
	PendingPhase    string
	PendingSince    *time.Time
}

func (b *BusinessTx) ComputeTriggerState(ctx context.Context, computeID string, revision int64) (ComputeTriggerState, error) {
	if b == nil || b.tx == nil || !canonicalPointUUID.MatchString(computeID) || revision < 1 {
		return ComputeTriggerState{}, ErrInvalidInput
	}
	if err := b.ensureOpen(); err != nil {
		return ComputeTriggerState{}, err
	}
	var state ComputeTriggerState
	var previousQuality *string
	var pendingPhase *string
	err := b.tx.QueryRow(ctx, `SELECT compute_revision,previous_value,previous_quality,previous_seen,condition_active,pending_phase,pending_since FROM runtime_engine.compute_trigger_state WHERE deployment_id=$1 AND compute_id=$2::uuid FOR UPDATE`, b.deploymentID, computeID).
		Scan(&state.ComputeRevision, &state.PreviousValue, &previousQuality, &state.PreviousSeen, &state.ConditionActive, &pendingPhase, &state.PendingSince)
	if errors.Is(err, pgx.ErrNoRows) {
		return ComputeTriggerState{ComputeRevision: revision}, nil
	}
	if err != nil {
		return ComputeTriggerState{}, err
	}
	if state.ComputeRevision != revision {
		return ComputeTriggerState{ComputeRevision: revision}, nil
	}
	if previousQuality != nil {
		state.PreviousQuality = *previousQuality
	}
	if pendingPhase != nil {
		state.PendingPhase = *pendingPhase
	}
	return state, nil
}

func (b *BusinessTx) SaveComputeTriggerState(ctx context.Context, computeID string, state ComputeTriggerState) error {
	if b == nil || b.tx == nil || !canonicalPointUUID.MatchString(computeID) || state.ComputeRevision < 1 || (state.PendingPhase == "") != (state.PendingSince == nil) || (state.PendingPhase != "" && state.PendingPhase != "entered" && state.PendingPhase != "exited") || (state.PreviousQuality != "" && state.PreviousQuality != "good" && state.PreviousQuality != "bad" && state.PreviousQuality != "unknown") || (len(state.PreviousValue) > 0 && !validFiniteJSON(state.PreviousValue)) {
		return ErrInvalidInput
	}
	if err := b.ensureOpen(); err != nil {
		return err
	}
	_, err := b.tx.Exec(ctx, `INSERT INTO runtime_engine.compute_trigger_state (deployment_id,compute_id,compute_revision,previous_value,previous_quality,previous_seen,condition_active,pending_phase,pending_since)
VALUES ($1,$2::uuid,$3,$4::jsonb,$5,$6,$7,NULLIF($8,''),$9)
ON CONFLICT (deployment_id,compute_id) DO UPDATE SET compute_revision=EXCLUDED.compute_revision,previous_value=EXCLUDED.previous_value,previous_quality=EXCLUDED.previous_quality,previous_seen=EXCLUDED.previous_seen,condition_active=EXCLUDED.condition_active,pending_phase=EXCLUDED.pending_phase,pending_since=EXCLUDED.pending_since,updated_at=now()`, b.deploymentID, computeID, state.ComputeRevision, nullableJSON(state.PreviousValue), nullableText(state.PreviousQuality), state.PreviousSeen, state.ConditionActive, state.PendingPhase, state.PendingSince)
	return err
}

// NextComputeSequence is intentionally part of BusinessTx: output sequence,
// processed input, snapshots and outbox are committed or rolled back together.
func (b *BusinessTx) NextComputeSequence(ctx context.Context, producerKey string, token ProducerToken) (int64, error) {
	if b == nil || b.tx == nil || validToken(b.deploymentID, producerKey, token.OwnerID, token.Epoch) != nil {
		return 0, ErrInvalidInput
	}
	if err := b.ensureOpen(); err != nil {
		return 0, err
	}
	var owner string
	var epoch int64
	err := b.tx.QueryRow(ctx, `SELECT owner_id,epoch FROM runtime_engine.producer_fence WHERE deployment_id=$1 AND producer_key=$2 FOR UPDATE`, b.deploymentID, producerKey).Scan(&owner, &epoch)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrFenceStale
	}
	if err != nil {
		return 0, err
	}
	if owner != token.OwnerID || epoch != token.Epoch {
		return 0, ErrFenceStale
	}
	var storedOwner string
	var storedEpoch, last int64
	err = b.tx.QueryRow(ctx, `SELECT owner_id,epoch,last_sequence FROM runtime_engine.producer_sequence WHERE deployment_id=$1 AND producer_key=$2 FOR UPDATE`, b.deploymentID, producerKey).Scan(&storedOwner, &storedEpoch, &last)
	if errors.Is(err, pgx.ErrNoRows) {
		_, err = b.tx.Exec(ctx, `INSERT INTO runtime_engine.producer_sequence (deployment_id,producer_key,owner_id,epoch,last_sequence) VALUES ($1,$2,$3,$4,0)`, b.deploymentID, producerKey, token.OwnerID, token.Epoch)
		return 0, err
	}
	if err != nil {
		return 0, err
	}
	if storedOwner != token.OwnerID || storedEpoch != token.Epoch {
		if storedEpoch < token.Epoch {
			_, err = b.tx.Exec(ctx, `UPDATE runtime_engine.producer_sequence SET owner_id=$3,epoch=$4,last_sequence=0,updated_at=now() WHERE deployment_id=$1 AND producer_key=$2`, b.deploymentID, producerKey, token.OwnerID, token.Epoch)
			return 0, err
		}
		return 0, ErrFenceStale
	}
	if last == math.MaxInt64 {
		return 0, ErrSequenceExhausted
	}
	next := last + 1
	_, err = b.tx.Exec(ctx, `UPDATE runtime_engine.producer_sequence SET last_sequence=$3,updated_at=now() WHERE deployment_id=$1 AND producer_key=$2 AND owner_id=$4 AND epoch=$5`, b.deploymentID, producerKey, next, token.OwnerID, token.Epoch)
	return next, err
}

// VerifyComputeProducer fences execution before the sandbox is contacted;
// otherwise a stale worker could spend resources or observe inputs after its
// producer assignment was revoked even if sequence allocation later failed.
func (b *BusinessTx) VerifyComputeProducer(ctx context.Context, producerKey string, token ProducerToken) error {
	if b == nil || b.tx == nil || validToken(b.deploymentID, producerKey, token.OwnerID, token.Epoch) != nil {
		return ErrInvalidInput
	}
	if err := b.ensureOpen(); err != nil {
		return err
	}
	var owner string
	var epoch int64
	err := b.tx.QueryRow(ctx, `SELECT owner_id,epoch FROM runtime_engine.producer_fence WHERE deployment_id=$1 AND producer_key=$2 FOR UPDATE`, b.deploymentID, producerKey).Scan(&owner, &epoch)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrFenceStale
	}
	if err != nil {
		return err
	}
	if owner != token.OwnerID || epoch != token.Epoch {
		return ErrFenceStale
	}
	return nil
}

func nullableText(value string) any {
	if value == "" {
		return nil
	}
	return value
}

// ComputeScheduleState is revision-scoped durable scheduler state. A changed
// Artifact revision intentionally gets a fresh run counter and occurrence key.
type ComputeScheduleState struct {
	ComputeRevision int64
	RunCount        int64
	NextRunAt       *time.Time
	LastRunAt       *time.Time
	LastLocalKey    string
}

func (b *BusinessTx) ComputeScheduleState(ctx context.Context, computeID string, revision int64) (ComputeScheduleState, error) {
	if b == nil || b.tx == nil || !canonicalPointUUID.MatchString(computeID) || revision < 1 {
		return ComputeScheduleState{}, ErrInvalidInput
	}
	if err := b.ensureOpen(); err != nil {
		return ComputeScheduleState{}, err
	}
	var state ComputeScheduleState
	var lastLocalKey *string
	err := b.tx.QueryRow(ctx, `SELECT compute_revision,run_count,next_run_at,last_run_at,last_local_key FROM runtime_engine.compute_schedule_state WHERE deployment_id=$1 AND compute_id=$2::uuid FOR UPDATE`, b.deploymentID, computeID).Scan(&state.ComputeRevision, &state.RunCount, &state.NextRunAt, &state.LastRunAt, &lastLocalKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return ComputeScheduleState{ComputeRevision: revision}, nil
	}
	if err != nil {
		return ComputeScheduleState{}, err
	}
	if state.ComputeRevision != revision {
		return ComputeScheduleState{ComputeRevision: revision}, nil
	}
	if lastLocalKey != nil {
		state.LastLocalKey = *lastLocalKey
	}
	return state, nil
}
func (b *BusinessTx) SaveComputeScheduleState(ctx context.Context, computeID string, state ComputeScheduleState) error {
	if b == nil || b.tx == nil || !canonicalPointUUID.MatchString(computeID) || state.ComputeRevision < 1 || state.RunCount < 0 || len(state.LastLocalKey) > 256 || ((state.LastRunAt == nil) != (state.LastLocalKey == "")) {
		return ErrInvalidInput
	}
	if err := b.ensureOpen(); err != nil {
		return err
	}
	_, err := b.tx.Exec(ctx, `INSERT INTO runtime_engine.compute_schedule_state (deployment_id,compute_id,compute_revision,run_count,next_run_at,last_run_at,last_local_key) VALUES ($1,$2::uuid,$3,$4,$5,$6,NULLIF($7,'')) ON CONFLICT (deployment_id,compute_id) DO UPDATE SET compute_revision=EXCLUDED.compute_revision,run_count=EXCLUDED.run_count,next_run_at=EXCLUDED.next_run_at,last_run_at=EXCLUDED.last_run_at,last_local_key=EXCLUDED.last_local_key,updated_at=now()`, b.deploymentID, computeID, state.ComputeRevision, state.RunCount, state.NextRunAt, state.LastRunAt, state.LastLocalKey)
	return err
}

type ScheduledMessage struct {
	DeploymentID, Role, ComputeID string
	Token                         ConsumerRoleToken
	ProducerKey                   string
	ProducerToken                 ProducerToken
	OccurredAt                    time.Time
}
type ScheduledHandler func(context.Context, *BusinessTx) error

// ProcessScheduled is the scheduler counterpart of ProcessMessage: role and
// producer fences plus all schedule/output/outbox writes share one transaction.
func (s *Store) ProcessScheduled(ctx context.Context, message ScheduledMessage, handler ScheduledHandler) error {
	if handler == nil || validToken(message.DeploymentID, message.Role, message.Token.OwnerID, message.Token.Epoch) != nil || !canonicalPointUUID.MatchString(message.ComputeID) || message.ProducerKey == "" || validToken(message.DeploymentID, message.ProducerKey, message.ProducerToken.OwnerID, message.ProducerToken.Epoch) != nil || message.OccurredAt.IsZero() {
		return ErrInvalidInput
	}
	if err := s.ensureOpen(); err != nil {
		return err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = assertConsumerFence(ctx, tx, Message{DeploymentID: message.DeploymentID, Role: message.Role, Token: message.Token}); err != nil {
		return err
	}
	// Lock the producer fence before exposing BusinessTx. NextComputeSequence
	// rechecks it, so an ownership change cannot slip between checks.
	var owner string
	var epoch int64
	if err = tx.QueryRow(ctx, `SELECT owner_id,epoch FROM runtime_engine.producer_fence WHERE deployment_id=$1 AND producer_key=$2 FOR UPDATE`, message.DeploymentID, message.ProducerKey).Scan(&owner, &epoch); errors.Is(err, pgx.ErrNoRows) {
		return ErrFenceStale
	} else if err != nil {
		return err
	}
	if owner != message.ProducerToken.OwnerID || epoch != message.ProducerToken.Epoch {
		return ErrFenceStale
	}
	if err = handler(ctx, &BusinessTx{store: s, tx: tx, deploymentID: message.DeploymentID}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
