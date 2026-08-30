package alarm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/indu-forge/runtime-engine/internal/ingress"
	"github.com/indu-forge/runtime-engine/internal/loader"
	"github.com/indu-forge/runtime-engine/internal/model"
	"github.com/indu-forge/runtime-engine/internal/store/postgres"
	"github.com/indu-forge/runtime-engine/internal/transportlimits"
)

const producerKey = "alarm"

// errStoredAlarmState marks a malformed persisted domain snapshot after its
// database read succeeded. Sweep uses this marker to isolate that one item;
// query, transaction and fence errors deliberately remain unmarked/fatal.
var errStoredAlarmState = errors.New("alarm 持久化状态不合法")

// Handler 仅维护 alarm 自己的输入/状态投影；它故意不读取 writer 的 current，
// 因而 raw 与 computed stream 的消费先后不会改变报警结论。
type Handler struct {
	artifact model.ProjectArtifact
	config   model.EngineConfig
	items    map[string]model.AlarmItem
	byPoint  map[string][]string
	producer model.ProducerAssignment
}

type alarmSweepStore interface {
	ProcessAlarmSweep(context.Context, postgres.AlarmSweepMessage, postgres.AlarmSweepHandler) error
}

func NewPostgresHandler(artifact model.ProjectArtifact, config model.EngineConfig) (*Handler, error) {
	if len(artifact.AlarmItems) > model.MaxAlarmItems {
		return nil, errors.New("alarmItems 超过 V1 上限")
	}
	h := &Handler{artifact: artifact, config: config, items: map[string]model.AlarmItem{}, byPoint: map[string][]string{}}
	for _, assignment := range config.ProducerAssignments {
		if assignment.ProducerType != "alarm" {
			continue
		}
		if assignment.Role != "alarm" || h.producer.ProducerType != "" {
			return nil, errors.New("alarm producer assignment 非法")
		}
		h.producer = assignment
	}
	if h.producer.ProducerType == "" {
		return nil, errors.New("缺少唯一 alarm producer assignment")
	}
	for _, item := range artifact.AlarmItems {
		if !item.Enabled {
			continue
		}
		// 让纯领域构造器复核完整 config，handler 不维护第二套条件白名单。
		if _, err := NewRuntime(h.identity(), model.ProjectArtifact{AlarmItems: []model.AlarmItem{item}}, State{}); err != nil {
			return nil, err
		}
		h.items[item.ID] = item
		for _, input := range item.Inputs {
			h.byPoint[input.DatapointID] = append(h.byPoint[input.DatapointID], item.ID)
		}
	}
	for point := range h.byPoint {
		sort.Strings(h.byPoint[point])
	}
	return h, nil
}

func (h *Handler) PostgresHandler() ingress.PostgresHandler { return h.HandlePostgres }

func (h *Handler) identity() Identity {
	return Identity{DeploymentID: h.config.DeploymentID, AccountID: h.config.AccountID, OwnerID: h.producer.Ownership.OwnerID, Epoch: h.producer.Ownership.Epoch}
}

// Preflight is a read-only startup/lifecycle gate.  It detects an active
// instance that no longer has an enabled matching definition before a handler
// starts consuming, rather than silently dropping its only clear path.
func (h *Handler) Preflight(ctx context.Context, store *postgres.Store) error {
	if h == nil || store == nil {
		return errors.New("alarm preflight 依赖非法")
	}
	records, err := store.AlarmItemStates(ctx, h.config.DeploymentID)
	if err != nil {
		return err
	}
	for id, record := range records {
		state, err := strictStoredItemState(record.State)
		if err != nil {
			return errors.New("alarm 持久化状态 JSON 非法")
		}
		item, enabled := h.items[id]
		if !enabled {
			if state.ActiveConditionID != "" {
				return errors.New("disabled 或已删除 alarm 存在 active 实例")
			}
			continue
		}
		if record.AlarmRevision != item.Revision {
			if state.ActiveConditionID != "" {
				return errors.New("alarm revision 不匹配且存在 active 实例")
			}
			continue
		}
		if _, err := NewRuntime(h.identity(), model.ProjectArtifact{AlarmItems: []model.AlarmItem{item}}, State{Items: map[string]ItemState{id: state}}); err != nil {
			return errors.New("alarm 持久化状态不通过当前定义校验")
		}
	}
	return nil
}

func strictStoredItemState(raw json.RawMessage) (ItemState, error) {
	var state ItemState
	trimmed := strings.TrimSpace(string(raw))
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return ItemState{}, errors.New("状态 JSON 必须为对象")
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&state) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return ItemState{}, errors.New("状态 JSON 非法")
	}
	return state, nil
}

// HandlePostgres 在 ingress 已打开的 BusinessTx 内运行；处理成功才会由外层提交
// processed_event、consumer_checkpoint、alarm state 与 outbox，随后 ingress 才 Ack。
func (h *Handler) HandlePostgres(ctx context.Context, tx *postgres.BusinessTx, message ingress.ValidatedMessage) error {
	if h == nil || tx == nil || message.Consumer.Role != "alarm" || message.Event.DeploymentID != h.config.DeploymentID || message.Event.AccountID != h.config.AccountID {
		return errors.New("alarm handler role 或 identity 非法")
	}
	ids := append([]string(nil), h.byPoint[message.Event.PointID]...)
	if len(ids) == 0 {
		return nil
	}
	input, err := pointInput(message)
	if err != nil {
		return err
	}
	if err = tx.VerifyProducerInTx(ctx, producerKey, postgres.ProducerToken{OwnerID: h.producer.Ownership.OwnerID, Epoch: h.producer.Ownership.Epoch}); err != nil {
		return err
	}
	runtime, records, err := h.loadRuntime(ctx, tx, ids)
	if err != nil {
		return err
	}
	events, err := runtime.Apply(input)
	if err != nil {
		return err
	}
	return h.persist(ctx, tx, runtime, records, events, input.ServerTimestamp)
}

func (h *Handler) loadRuntime(ctx context.Context, tx *postgres.BusinessTx, ids []string) (*Runtime, map[string]postgres.AlarmItemState, error) {
	items := make([]model.AlarmItem, 0, len(ids))
	restored := State{Items: map[string]ItemState{}}
	records := make(map[string]postgres.AlarmItemState, len(ids))
	for _, id := range ids {
		item, ok := h.items[id]
		if !ok {
			return nil, nil, errors.New("alarm item 不存在")
		}
		record, err := tx.AlarmItemState(ctx, id, item.Revision)
		if err != nil {
			return nil, nil, err
		}
		state, err := strictStoredItemState(record.State)
		if err != nil {
			return nil, nil, errStoredAlarmState
		}
		restored.Items[id] = state
		records[id] = record
		items = append(items, item)
	}
	runtime, err := NewRuntime(h.identity(), model.ProjectArtifact{AlarmItems: items}, restored)
	if err != nil {
		return nil, nil, errStoredAlarmState
	}
	return runtime, records, nil
}

func (h *Handler) persist(ctx context.Context, tx *postgres.BusinessTx, runtime *Runtime, records map[string]postgres.AlarmItemState, events []Event, now time.Time) error {
	state := runtime.State()
	for _, id := range sortedKeys(records) {
		item := h.items[id]
		raw, err := json.Marshal(state.Items[id])
		if err != nil {
			return err
		}
		record := records[id]
		record.AlarmRevision = item.Revision
		record.State = raw
		record.NextEvaluationAt = NextEvaluationAt(item, state.Items[id], now.UTC())
		if err := tx.SaveAlarmItemState(ctx, id, record); err != nil {
			if errors.Is(err, postgres.ErrAlarmStateTooLarge) {
				return retryableOutputError(err)
			}
			return err
		}
	}
	for _, event := range events {
		payload, err := json.Marshal(event)
		if err != nil || loader.ValidateAlarmEvent(payload) != nil {
			return retryableOutputError(errors.New("alarm outbox payload 非法"))
		}
		if _, _, err = tx.Enqueue(ctx, postgres.OutboxMessage{DeploymentID: h.config.DeploymentID, DedupeKey: event.EventID, Subject: "alarm.event", Payload: payload}); err != nil {
			if errors.Is(err, transportlimits.ErrOutboundPayloadTooLarge) || errors.Is(err, transportlimits.ErrOutboundSubject) {
				return retryableOutputError(err)
			}
			return err
		}
	}
	return nil
}

// retryableOutputError is deliberately limited to data derived entirely from
// one input event. It gives ingress its normal bounded maxDeliver→DLQ path;
// store, fence and context errors must retain their fatal semantics.
func retryableOutputError(err error) error {
	return fmt.Errorf("%w: %w", ingress.ErrRetryableBusiness, err)
}

// SweepDue 使用持久化 next_evaluation_at 进行有界、无忙轮询的恢复性扫描。
func (h *Handler) SweepDue(ctx context.Context, store *postgres.Store, token postgres.ConsumerRoleToken, now time.Time, limit int) error {
	var sweepStore alarmSweepStore
	if store != nil {
		sweepStore = store
	}
	return h.sweepDue(ctx, sweepStore, token, now, limit)
}

func (h *Handler) sweepDue(ctx context.Context, store alarmSweepStore, token postgres.ConsumerRoleToken, now time.Time, limit int) error {
	if h == nil || now.IsZero() || now.Location() != time.UTC || limit < 1 || limit > 256 {
		return errors.New("alarm sweep 参数非法")
	}
	items := h.sweepItems()
	if len(items) == 0 {
		return nil
	}
	if store == nil {
		return errors.New("alarm sweep 参数非法")
	}
	return store.ProcessAlarmSweep(ctx, postgres.AlarmSweepMessage{DeploymentID: h.config.DeploymentID, Role: "alarm", Token: token, ProducerKey: producerKey, ProducerToken: postgres.ProducerToken{OwnerID: h.producer.Ownership.OwnerID, Epoch: h.producer.Ownership.Epoch}, OccurredAt: now, Limit: limit, AlarmItems: items}, func(ctx context.Context, tx *postgres.BusinessTx, id string) error {
		runtime, records, err := h.loadRuntime(ctx, tx, []string{id})
		if err != nil {
			return sweepItemLoadError(err)
		}
		events, err := runtime.Sweep(now)
		if err != nil {
			return postgres.NewAlarmSweepItemPoison(postgres.AlarmSweepPoisonInvalidState)
		}
		if err := h.persist(ctx, tx, runtime, records, events, now); err != nil {
			return sweepPersistError(err)
		}
		return nil
	})
}

func sweepPersistError(err error) error {
	// These markers originate only from persist's deterministic event/envelope
	// construction. All storage, fence and lifecycle errors remain fatal.
	if errors.Is(err, postgres.ErrInvalidInput) || errors.Is(err, postgres.ErrAlarmStateTooLarge) || errors.Is(err, ingress.ErrRetryableBusiness) {
		return postgres.NewAlarmSweepItemPoison(postgres.AlarmSweepPoisonInvalidOutput)
	}
	return err
}

func sweepItemLoadError(err error) error {
	switch {
	case errors.Is(err, errStoredAlarmState), errors.Is(err, postgres.ErrInvalidInput):
		return postgres.NewAlarmSweepItemPoison(postgres.AlarmSweepPoisonInvalidState)
	case errors.Is(err, postgres.ErrActiveAlarmRevision):
		return postgres.NewAlarmSweepItemPoison(postgres.AlarmSweepPoisonActiveRevision)
	default:
		return err
	}
}

func (h *Handler) sweepItems() []postgres.AlarmSweepItem {
	ids := make([]string, 0, len(h.items))
	for id := range h.items {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	items := make([]postgres.AlarmSweepItem, 0, len(ids))
	for _, id := range ids {
		items = append(items, postgres.AlarmSweepItem{AlarmItemID: id, AlarmRevision: h.items[id].Revision})
	}
	return items
}

// RunSweepRunner 是可取消 runner；它不拥有服务生命周期，main 后续只需传入 ctx。
func (h *Handler) RunSweepRunner(ctx context.Context, store *postgres.Store, token postgres.ConsumerRoleToken, interval time.Duration, limit int) error {
	err := h.RunSweepRunnerWithDrain(ctx, ctx, store, token, interval, limit, nil)
	if err == nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}

// RunSweepRunnerWithDrain stops new sweep ticks when intake is cancelled, but
// gives a sweep already selected a separate work context to finish.  Poison is
// isolated by SweepDue and reported through a stable callback without killing
// later alarms.
func (h *Handler) RunSweepRunnerWithDrain(intake, work context.Context, store *postgres.Store, token postgres.ConsumerRoleToken, interval time.Duration, limit int, onPoison func()) error {
	if interval <= 0 {
		return errors.New("alarm runner interval 非法")
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	if intake.Err() != nil {
		return nil
	}
	sweep := func(now time.Time) error { return h.SweepDue(work, store, token, now.UTC(), limit) }
	if err := sweep(time.Now().UTC()); err != nil {
		if errors.Is(err, postgres.ErrAlarmSweepItemPoison) {
			if onPoison != nil {
				onPoison()
			}
		} else if work.Err() == nil {
			return err
		}
	}
	for {
		select {
		case <-intake.Done():
			return nil
		case now := <-ticker.C:
			if intake.Err() != nil {
				return nil
			}
			if err := sweep(now); err != nil {
				if work.Err() != nil {
					return nil
				}
				if errors.Is(err, postgres.ErrAlarmSweepItemPoison) {
					if onPoison != nil {
						onPoison()
					}
					continue
				}
				return err
			}
		}
	}
}

func runSweepRunner(ctx context.Context, initial time.Time, ticks <-chan time.Time, sweep func(time.Time) error) error {
	if err := sweep(initial.UTC()); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, postgres.ErrAlarmSweepItemPoison) {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case now := <-ticks:
			if err := sweep(now.UTC()); err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				if errors.Is(err, postgres.ErrAlarmSweepItemPoison) {
					continue
				}
				return err
			}
		}
	}
}

func pointInput(message ingress.ValidatedMessage) (PointInput, error) {
	event := message.Event
	if event.SchemaVersion != "data.raw.v1" && event.SchemaVersion != "data.computed.v1" {
		return PointInput{}, errors.New("alarm 输入 schema 非法")
	}
	parse := func(value string) (time.Time, error) {
		if !strings.HasSuffix(value, "Z") {
			return time.Time{}, errors.New("alarm 输入时间非法")
		}
		parsed, err := time.Parse(time.RFC3339Nano, value)
		if err != nil || parsed.IsZero() || parsed.Location() != time.UTC {
			return time.Time{}, errors.New("alarm 输入时间非法")
		}
		return parsed.UTC(), nil
	}
	source, err := parse(event.SourceTimestamp)
	if err != nil {
		return PointInput{}, err
	}
	server, err := parse(event.ServerTimestamp)
	if err != nil {
		return PointInput{}, err
	}
	received, err := parse(event.ReceivedAt)
	if err != nil {
		return PointInput{}, err
	}
	// V1 point wire 没有 offline 字段：bad 是 ingress 的唯一单向 offline 事实，
	// unknown 不代表离线；stale 始终交给缺失时间的 Sweep 处理。
	return PointInput{SchemaVersion: event.SchemaVersion, EventID: event.EventID, PointID: event.PointID, Epoch: event.Epoch, Sequence: event.Sequence, Value: append(json.RawMessage(nil), event.Value...), Quality: event.Quality, Offline: event.Quality == "bad", SourceTimestamp: source, ServerTimestamp: server, ReceivedAt: received}, nil
}

func sortedKeys(values map[string]postgres.AlarmItemState) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
