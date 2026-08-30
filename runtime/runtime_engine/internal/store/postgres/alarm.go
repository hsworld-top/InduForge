package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/indu-forge/runtime-engine/internal/model"
	"github.com/jackc/pgx/v5"
)

// ErrActiveAlarmRevision 表示配置 revision 已改变，但旧状态仍有活动实例；V1 不伪造
// CLEAR，也不允许悄悄抹除该实例，调用方必须把它作为健康失败处理。
var ErrActiveAlarmRevision = errors.New("活动 alarm 不允许跨 revision 重置")

// ErrAlarmStateTooLarge is a deterministic V1 capacity refusal. The database
// schema keeps the final CHECK; this marker lets callers classify it before
// an INSERT/UPDATE can turn an otherwise valid ingress event into a driver
// error.
var ErrAlarmStateTooLarge = errors.New("alarm state exceeds frozen size limit")

const maxAlarmStateBytes = 1 << 20

// ErrAlarmSweepItemPoison 标识已知的单个 alarm 领域/状态坏项。它只用于
// sweep 的隔离语义：该项的事务必须回滚，但其他独立 item 可以继续提交。
// 它不是数据库、提交或 fence 错误的替代品，后者始终会让整个 sweep 失败。
var ErrAlarmSweepItemPoison = errors.New("alarm sweep item poison")

// AlarmSweepPoisonReason 是给直接调用者或健康层使用的稳定分类码，不包含
// 持久化状态内容、SQL 或底层驱动错误。
type AlarmSweepPoisonReason string

const (
	AlarmSweepPoisonInvalidState   AlarmSweepPoisonReason = "invalid-state"
	AlarmSweepPoisonActiveRevision AlarmSweepPoisonReason = "active-revision"
	AlarmSweepPoisonInvalidOutput  AlarmSweepPoisonReason = "invalid-output"
)

// AlarmSweepItemPoisonError 是 handler 对可恢复隔离错误的显式声明。调用方
// 不得把任意 error 包成此类型，否则会掩盖数据库或 fence 的基础故障。
type AlarmSweepItemPoisonError struct{ Reason AlarmSweepPoisonReason }

func (e *AlarmSweepItemPoisonError) Error() string {
	if e == nil || e.Reason == "" {
		return ErrAlarmSweepItemPoison.Error()
	}
	return ErrAlarmSweepItemPoison.Error() + ": " + string(e.Reason)
}

func (e *AlarmSweepItemPoisonError) Is(target error) bool { return target == ErrAlarmSweepItemPoison }

// NewAlarmSweepItemPoison constructs the only error shape that lets a sweep
// continue after a per-item rollback. Unknown reason values are rejected so
// health consumers only need to handle the stable codes above.
func NewAlarmSweepItemPoison(reason AlarmSweepPoisonReason) error {
	switch reason {
	case AlarmSweepPoisonInvalidState, AlarmSweepPoisonActiveRevision, AlarmSweepPoisonInvalidOutput:
		return &AlarmSweepItemPoisonError{Reason: reason}
	default:
		return ErrInvalidInput
	}
}

// AlarmSweepPoisonError aggregates isolated item failures from one sweep.
// It deliberately exposes only stable reason codes, never row IDs, JSON or
// PostgreSQL diagnostics.
type AlarmSweepPoisonError struct{ Reasons []AlarmSweepPoisonReason }

func (e *AlarmSweepPoisonError) Error() string        { return ErrAlarmSweepItemPoison.Error() }
func (e *AlarmSweepPoisonError) Is(target error) bool { return target == ErrAlarmSweepItemPoison }

func (e *AlarmSweepPoisonError) add(reason AlarmSweepPoisonReason) {
	for _, existing := range e.Reasons {
		if existing == reason {
			return
		}
	}
	e.Reasons = append(e.Reasons, reason)
}

// AlarmItemState 是每个 alarm item 的 revision-scoped 持久化领域快照。
// owner/epoch 属于 role/producer fence，刻意不复制到状态行。
type AlarmItemState struct {
	AlarmRevision    int64
	State            json.RawMessage
	Version          int64
	NextEvaluationAt *time.Time
	RevisionReset    bool
}

// AlarmItemStates is a read-only deployment snapshot for lifecycle preflight.
// It never repairs, migrates or filters rows: callers must decide whether an
// orphaned/disabled active instance makes the assignment unsafe to start.
func (s *Store) AlarmItemStates(ctx context.Context, deploymentID string) (map[string]AlarmItemState, error) {
	if s == nil || s.pool == nil || !validStableID(deploymentID) {
		return nil, ErrInvalidInput
	}
	if err := s.ensureOpen(); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT alarm_item_id::text,alarm_revision,state,version,next_evaluation_at FROM runtime_engine.alarm_item_state WHERE deployment_id=$1 ORDER BY alarm_item_id`, deploymentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	states := map[string]AlarmItemState{}
	for rows.Next() {
		var id string
		var state AlarmItemState
		if err := rows.Scan(&id, &state.AlarmRevision, &state.State, &state.Version, &state.NextEvaluationAt); err != nil {
			return nil, err
		}
		states[id] = state
	}
	return states, rows.Err()
}

func (b *BusinessTx) AlarmItemState(ctx context.Context, alarmItemID string, revision int64) (AlarmItemState, error) {
	if b == nil || b.tx == nil || !canonicalPointUUID.MatchString(alarmItemID) || revision < 1 {
		return AlarmItemState{}, ErrInvalidInput
	}
	if err := b.ensureOpen(); err != nil {
		return AlarmItemState{}, err
	}
	var state AlarmItemState
	err := b.tx.QueryRow(ctx, `SELECT alarm_revision,state,version,next_evaluation_at
FROM runtime_engine.alarm_item_state WHERE deployment_id=$1 AND alarm_item_id=$2::uuid FOR UPDATE`, b.deploymentID, alarmItemID).
		Scan(&state.AlarmRevision, &state.State, &state.Version, &state.NextEvaluationAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return AlarmItemState{AlarmRevision: revision, State: json.RawMessage(`{}`)}, nil
	}
	if err != nil {
		return AlarmItemState{}, err
	}
	if state.AlarmRevision == revision {
		return state, nil
	}
	// Revision 变更必须丢弃业务状态和输入快照，但 transition 计数是
	// eventId 防冲突的持久事实，不能回到零。这里不依赖 alarm 包，仍对
	// 存储的最小形状 fail-closed 校验。
	var snapshot struct {
		ActiveConditionID  string `json:"activeConditionId"`
		StateVersion       int64  `json:"stateVersion"`
		TransitionSequence int64  `json:"transitionSequence"`
	}
	if err := json.Unmarshal(state.State, &snapshot); err != nil {
		return AlarmItemState{}, ErrInvalidInput
	}
	if snapshot.ActiveConditionID != "" {
		return AlarmItemState{}, ErrActiveAlarmRevision
	}
	if snapshot.StateVersion < 0 || snapshot.TransitionSequence < 0 || snapshot.StateVersion != snapshot.TransitionSequence {
		return AlarmItemState{}, ErrInvalidInput
	}
	resetState, err := json.Marshal(struct {
		StateVersion       int64 `json:"stateVersion"`
		TransitionSequence int64 `json:"transitionSequence"`
	}{StateVersion: snapshot.StateVersion, TransitionSequence: snapshot.TransitionSequence})
	if err != nil {
		return AlarmItemState{}, ErrInvalidInput
	}
	return AlarmItemState{AlarmRevision: revision, State: resetState, Version: state.Version, RevisionReset: true}, nil
}

// SaveAlarmItemState 以读取时 version 作 CAS。正常路径已持有 FOR UPDATE；CAS 是为了
// 阻止将来错误复用 BusinessTx 时把并发更新静默覆盖。
func (b *BusinessTx) SaveAlarmItemState(ctx context.Context, alarmItemID string, state AlarmItemState) error {
	if b == nil || b.tx == nil || !canonicalPointUUID.MatchString(alarmItemID) || state.AlarmRevision < 1 || !validFiniteJSON(state.State) || len(state.State) == 0 || state.Version < 0 || (state.NextEvaluationAt != nil && state.NextEvaluationAt.Location() != time.UTC) {
		return ErrInvalidInput
	}
	if err := b.ensureOpen(); err != nil {
		return err
	}
	// PostgreSQL jsonb may canonicalize numeric spellings differently from the
	// input bytes. Ask the same engine used by the CHECK constraint so this
	// preflight is conservative with respect to the eventual persisted text.
	var encodedBytes int
	if err := b.tx.QueryRow(ctx, `SELECT octet_length($1::jsonb::text)`, state.State).Scan(&encodedBytes); err != nil {
		return err
	}
	if encodedBytes > maxAlarmStateBytes {
		return ErrAlarmStateTooLarge
	}
	var command pgconnTag
	var err error
	if state.Version == 0 {
		command, err = b.tx.Exec(ctx, `INSERT INTO runtime_engine.alarm_item_state (deployment_id,alarm_item_id,alarm_revision,state,version,next_evaluation_at)
VALUES ($1,$2::uuid,$3,$4::jsonb,1,$5)`, b.deploymentID, alarmItemID, state.AlarmRevision, state.State, state.NextEvaluationAt)
	} else {
		command, err = b.tx.Exec(ctx, `UPDATE runtime_engine.alarm_item_state SET alarm_revision=$3,state=$4::jsonb,version=version+1,next_evaluation_at=$5,updated_at=now()
WHERE deployment_id=$1 AND alarm_item_id=$2::uuid AND version=$6`, b.deploymentID, alarmItemID, state.AlarmRevision, state.State, state.NextEvaluationAt, state.Version)
	}
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return ErrFenceStale
	}
	return nil
}

// AlarmSweepItem 精确标识当前 handler 已启用的一项 alarm 定义。revision 是
// allowed 集合的一部分，不能仅靠 item ID 匹配旧状态行。
type AlarmSweepItem struct {
	AlarmItemID   string
	AlarmRevision int64
}

// AlarmDueItems is used only from the fenced sweep transaction. SKIP LOCKED
// prevents one slow item from blocking unrelated due alarms and gives runners
// crash-safe retry through the persisted next_evaluation_at.
func (b *BusinessTx) AlarmDueItems(ctx context.Context, items []AlarmSweepItem, now time.Time, limit int) ([]AlarmSweepItem, error) {
	if b == nil || b.tx == nil || !validAlarmSweepItems(items) || now.IsZero() || now.Location() != time.UTC || limit < 1 || limit > len(items) {
		return nil, ErrInvalidInput
	}
	if err := b.ensureOpen(); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(items))
	revisions := make([]int64, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.AlarmItemID)
		revisions = append(revisions, item.AlarmRevision)
	}
	rows, err := b.tx.Query(ctx, `SELECT state.alarm_item_id::text,state.alarm_revision FROM runtime_engine.alarm_item_state AS state
JOIN unnest($2::uuid[],$3::bigint[]) AS allowed(alarm_item_id,alarm_revision)
  ON state.alarm_item_id=allowed.alarm_item_id AND state.alarm_revision=allowed.alarm_revision
WHERE state.deployment_id=$1 AND state.next_evaluation_at IS NOT NULL AND state.next_evaluation_at <= $4
ORDER BY state.next_evaluation_at,state.alarm_item_id FOR UPDATE OF state SKIP LOCKED LIMIT $5`, b.deploymentID, ids, revisions, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var due []AlarmSweepItem
	for rows.Next() {
		var item AlarmSweepItem
		if err := rows.Scan(&item.AlarmItemID, &item.AlarmRevision); err != nil {
			return nil, err
		}
		due = append(due, item)
	}
	return due, rows.Err()
}

// pgconnTag 让 SaveAlarmItemState 不暴露 pgconn 依赖到调用方。
type pgconnTag interface{ RowsAffected() int64 }

type AlarmSweepMessage struct {
	DeploymentID, Role, ProducerKey string
	Token                           ConsumerRoleToken
	ProducerToken                   ProducerToken
	OccurredAt                      time.Time
	Limit                           int
	AlarmItems                      []AlarmSweepItem
}

type AlarmSweepHandler func(context.Context, *BusinessTx, string) error

// ProcessAlarmSweep first selects at most the current handler's allowed item
// set, then processes each item in its own fenced transaction. Limit counts
// successful commits, not attempts: a declared poison row rolls back only its
// own transaction and cannot permanently starve later independent alarms.
func (s *Store) ProcessAlarmSweep(ctx context.Context, message AlarmSweepMessage, handler AlarmSweepHandler) error {
	if handler == nil || validToken(message.DeploymentID, message.Role, message.Token.OwnerID, message.Token.Epoch) != nil || validToken(message.DeploymentID, message.ProducerKey, message.ProducerToken.OwnerID, message.ProducerToken.Epoch) != nil || message.OccurredAt.IsZero() || message.OccurredAt.Location() != time.UTC || message.Limit < 1 || message.Limit > 256 {
		return ErrInvalidInput
	}
	if !validAlarmSweepItems(message.AlarmItems) {
		return ErrInvalidInput
	}
	if err := s.ensureOpen(); err != nil {
		return err
	}
	var items []AlarmSweepItem
	if err := s.withAlarmSweepTx(ctx, message, func(business *BusinessTx) error {
		var err error
		items, err = business.AlarmDueItems(ctx, message.AlarmItems, message.OccurredAt, len(message.AlarmItems))
		return err
	}); err != nil {
		return err
	}
	var poisons AlarmSweepPoisonError
	successes := 0
	for _, item := range items {
		if successes == message.Limit {
			break
		}
		item := item
		processed := false
		err := s.withAlarmSweepTx(ctx, message, func(business *BusinessTx) error {
			// The selection transaction has committed. Re-lock/recheck this one
			// row so another runner that already advanced it cannot duplicate work.
			due, err := business.AlarmDueItems(ctx, []AlarmSweepItem{item}, message.OccurredAt, 1)
			if err != nil || len(due) == 0 {
				return err
			}
			if err := handler(ctx, business, item.AlarmItemID); err != nil {
				return err
			}
			processed = true
			return nil
		})
		if err == nil {
			if processed {
				successes++
			}
			continue
		}
		var poison *AlarmSweepItemPoisonError
		if errors.As(err, &poison) {
			poisons.add(poison.Reason)
			continue
		}
		return err
	}
	if len(poisons.Reasons) != 0 {
		return &poisons
	}
	return nil
}

func validAlarmSweepItems(items []AlarmSweepItem) bool {
	if len(items) == 0 || len(items) > model.MaxAlarmItems {
		return false
	}
	seenIDs := make(map[string]struct{}, len(items))
	for _, item := range items {
		if !canonicalPointUUID.MatchString(item.AlarmItemID) || item.AlarmRevision < 1 {
			return false
		}
		if _, exists := seenIDs[item.AlarmItemID]; exists {
			return false
		}
		seenIDs[item.AlarmItemID] = struct{}{}
	}
	return true
}

func (s *Store) withAlarmSweepTx(ctx context.Context, message AlarmSweepMessage, work func(*BusinessTx) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = assertConsumerFence(ctx, tx, Message{DeploymentID: message.DeploymentID, Role: message.Role, Token: message.Token}); err != nil {
		return err
	}
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
	if err = work(&BusinessTx{store: s, tx: tx, deploymentID: message.DeploymentID}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
