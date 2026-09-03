// Package ingress 实现 RuntimeEngine V1 的严格入站校验和 Ack/Nak/DLQ 编排。
package ingress

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"time"
	"unicode/utf8"

	"github.com/indu-forge/runtime-engine/internal/eventid"
	"github.com/indu-forge/runtime-engine/internal/loader"
	"github.com/indu-forge/runtime-engine/internal/model"
	"github.com/indu-forge/runtime-engine/internal/store/postgres"
	transport "github.com/indu-forge/runtime-engine/internal/transport/jetstream"
	"github.com/indu-forge/runtime-engine/internal/transportlimits"
)

const (
	MaxBodyBytes             = transportlimits.MaxBodyBytes
	MaxTransportSubjectBytes = transportlimits.MaxTransportSubjectBytes
	MaxDLQPayloadBytes       = transportlimits.MaxDLQPayloadBytes
)
const maxFetchRetries = 3

var ErrPermanent = errors.New("永久入站拒绝")
var ErrFatalPoison = errors.New("不可持久化的 transport poison")

// 以下错误只作为宿主的安全诊断边界：不携带 transport、消息或数据库错误文本，
// 也不改变已有的 Fetch、处理或 Ack/Nak 重试语义。
var ErrFetchExhausted = errors.New("ingress fetch retries exhausted")
var ErrProcess = errors.New("ingress process failed")
var ErrAck = errors.New("ingress acknowledgement failed")

// DiagnosticCode 将内部错误归一为可安全写入 RuntimeEngine 日志和状态的阶段码。
func DiagnosticCode(err error) string {
	switch {
	case errors.Is(err, ErrFetchExhausted):
		return "INGRESS_FETCH_EXHAUSTED"
	case errors.Is(err, ErrAck):
		return "INGRESS_ACK"
	default:
		return "INGRESS_PROCESS"
	}
}

// ErrRetryableBusiness is the narrow marker for a handler-declared isolated
// business failure. Store, fence and context failures must never use it.
var ErrRetryableBusiness = errors.New("可重试业务失败")
var eventIDPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// ConsumerToken 与 producer fence 有意分为不同类型，避免调用者错误比较或替换两种 token。
type ConsumerToken struct {
	OwnerID string
	Epoch   int64
}
type ProducerToken struct {
	OwnerID string
	Epoch   int64
}

// ProducerKey 只用于永久失败的已验证生产者 fence，避免把未经校验的字符串误当作生产者身份。
type ProducerKey string

type Event struct {
	SchemaVersion   string          `json:"schemaVersion"`
	Subject         string          `json:"subject"`
	EventID         string          `json:"eventId"`
	DeploymentID    string          `json:"deploymentId"`
	AccountID       string          `json:"accountId"`
	PointID         string          `json:"pointId"`
	OwnerID         string          `json:"ownerId"`
	Epoch           int64           `json:"epoch"`
	Sequence        int64           `json:"sequence"`
	Value           json.RawMessage `json:"value"`
	Quality         string          `json:"quality"`
	SourceTimestamp string          `json:"sourceTimestamp"`
	ServerTimestamp string          `json:"serverTimestamp"`
	ReceivedAt      string          `json:"receivedAt"`
	Source          *Source         `json:"source,omitempty"`
	Computation     *Computation    `json:"computation,omitempty"`
}
type Source struct {
	CollectorID  string `json:"collectorId"`
	ConnectionID string `json:"connectionId"`
	VariableID   string `json:"variableId"`
}
type Computation struct {
	ComputeID       string   `json:"computeId"`
	ComputeRevision int64    `json:"computeRevision"`
	InputEventIDs   []string `json:"inputEventIds"`
}

type ValidatedMessage struct {
	Event          Event
	RawBody        []byte
	Consumer       model.Consumer
	Token          ConsumerToken
	ProducerKey    string
	ProducerToken  ProducerToken
	StreamPosition int64
	DeliveryCount  int
	OccurredAt     time.Time
}
type ProcessResult uint8

const (
	Processed ProcessResult = iota
	Duplicate
	Collision
)

type PermanentResult uint8

const (
	PermanentStored PermanentResult = iota
	PermanentDuplicate
)

// Store 是故意窄化的事务边界。postgres red-team 会实现此接口；ingress 不直接依赖半成品表结构。
type Processor interface {
	Process(context.Context, ValidatedMessage) (ProcessResult, error)
	// 对相同 dlqId 的重投，Store 必须复用首次已提交的 outbox payload；不能以新的
	// deliveryCount/occurredAt 触发 dedupe content conflict。
	ProcessPermanentFailure(context.Context, PermanentFailure) (PermanentResult, error)
}

type PermanentFailure struct {
	DeploymentID  string
	AccountID     string
	ConsumerKey   string
	Role          string
	ConsumerToken ConsumerToken
	// RequireProducerFence 仅在完整校验成功、业务处理进入最终隔离时为 true。
	// 解析失败的消息不能伪造生产者身份，必须保持 false 和零值 key/token。
	RequireProducerFence bool
	ProducerKey          ProducerKey
	ProducerToken        ProducerToken
	OriginalSubject      string
	EventID              *string
	ReasonCode           string
	DeliveryCount        int
	BodySHA256           string
	RawBody              []byte
	OccurredAt           time.Time
	DeadLetterSubject    string
	DLQID                string
	DLQPayload           []byte
	StreamPosition       int64
}

type Message interface {
	Subject() string
	Body() []byte
	StreamPosition() int64
	DeliveryCount() int
	OccurredAt() time.Time
	Ack(context.Context) error
	InProgress(context.Context) error
	NakWithDelay(context.Context, time.Duration) error
}
type transportMessage struct{ transport.DeliveredMessage }

func (m transportMessage) Subject() string       { return m.DeliveredMessage.Subject }
func (m transportMessage) Body() []byte          { return m.DeliveredMessage.Body }
func (m transportMessage) StreamPosition() int64 { return int64(m.DeliveredMessage.StreamSequence) }
func (m transportMessage) DeliveryCount() int    { return int(m.DeliveredMessage.DeliveryCount) }
func (m transportMessage) OccurredAt() time.Time { return m.DeliveredMessage.OccurredAt }

type Runner struct {
	loaded    *loader.Loaded
	consumer  model.Consumer
	token     ConsumerToken
	processor Processor
	health    HealthSignal
}

// HealthSignal 只接收稳定原因码，供宿主汇总健康状态，永不传递 body/subject/Secret。
type HealthSignal interface{ ReportIngressFailure(string) }
type businessSuccessSignal interface{ RecordBusinessSuccess(time.Time) }

// PullClient 是 transport 的最小消费接口，便于 lifecycle 按 config 为每个 consumer 启动一个 Runner。
type PullClient interface {
	Fetch(context.Context, string, string, int, time.Duration) ([]transport.DeliveredMessage, error)
}

func NewRunner(loaded *loader.Loaded, consumer model.Consumer, token ConsumerToken, processor Processor) (*Runner, error) {
	return NewRunnerWithHealth(loaded, consumer, token, processor, nil)
}
func NewRunnerWithHealth(loaded *loader.Loaded, consumer model.Consumer, token ConsumerToken, processor Processor, health HealthSignal) (*Runner, error) {
	if loaded == nil || processor == nil || token.OwnerID == "" || token.Epoch < 1 {
		return nil, errors.New("ingress runner 依赖非法")
	}
	if consumer.Stream == loaded.Config.JetStream.EventStream || (consumer.FilterSubject != "data.raw.>" && consumer.FilterSubject != "data.computed.>") {
		return nil, errors.New("EVENT 不允许作为 Engine ingress consumer")
	}
	return &Runner{loaded: loaded, consumer: consumer, token: token, processor: processor, health: health}, nil
}

// Handle 在 DB 成功提交之后才 Ack；所有永久失败也必须由 Store 原子写 failure+DLQ outbox 后才 Ack。
func (r *Runner) Handle(ctx context.Context, message Message) error {
	if len(message.Subject()) > MaxTransportSubjectBytes {
		r.report("transport-subject-too-long")
		return fmt.Errorf("%w: transport-subject-too-long", ErrFatalPoison)
	}
	validated, err := r.validate(message)
	if err != nil {
		return r.permanent(ctx, message, eventIDFromBody(message.Body()), reasonCode(err), nil, true)
	}
	// maxDeliver 是业务 handler 的尝试阈值，不是 broker 配额。最后一次仍可成功；其后只恢复 DLQ 持久化。
	if message.DeliveryCount() > r.consumer.MaxDeliver {
		return r.permanent(ctx, message, &validated.Event.EventID, "max-deliver", &validated, true)
	}
	result, err := r.processWithProgress(ctx, message, validated)
	if err != nil {
		// A Store transaction error, stale fence or outer cancellation is not a
		// delivery failure.  It must leave the broker disposition untouched so
		// this owner stops instead of extending an obsolete lease forever.
		if !retryableBusinessError(err) {
			return err
		}
		// 业务失败第一次出现即进入宿主健康视图；不能等到最终 DLQ 后才暴露退化。
		r.report("handler-failure")
		if message.DeliveryCount() == r.consumer.MaxDeliver {
			return r.permanent(ctx, message, &validated.Event.EventID, "max-deliver", &validated, false)
		}
		return message.NakWithDelay(ctx, r.backoff(message.DeliveryCount()))
	}
	// Collision 的 Store 实现必须在同一事务内写 processing_failure、DLQ outbox 和 checkpoint；
	// 此处不得再按新的 deliveryCount 重建 DLQ payload。
	if result == Collision {
		return acknowledge(ctx, message)
	}
	if result == Processed {
		if health, ok := r.health.(businessSuccessSignal); ok {
			health.RecordBusinessSuccess(time.Now().UTC())
		}
	}
	return acknowledge(ctx, message)
}

func (r *Runner) processWithProgress(ctx context.Context, message Message, validated ValidatedMessage) (ProcessResult, error) {
	interval := time.Duration(r.consumer.AckWaitMS) * time.Millisecond / 2
	if interval <= 0 {
		interval = time.Second
	}
	done := make(chan struct{})
	stopped := make(chan struct{})
	defer func() { close(done); <-stopped }()
	go func() {
		defer close(stopped)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = message.InProgress(ctx)
			}
		}
	}()
	return r.processor.Process(ctx, validated)
}

// Run 循环拉取并串行处理一个 durable；取消 ctx 会停止新 Fetch，已取到消息先完成 disposition。
func (r *Runner) Run(ctx context.Context, client PullClient, batch int, maxWait time.Duration) error {
	return r.RunWithDrain(ctx, ctx, client, batch, maxWait)
}

// RunWithDrain separates intake cancellation from the context used to finish
// an already fetched batch.  Shutdown first cancels intake, then gives work a
// bounded grace period before force-cancelling it.
func (r *Runner) RunWithDrain(intake, work context.Context, client PullClient, batch int, maxWait time.Duration) error {
	failures := 0
	for {
		if err := intake.Err(); err != nil {
			return nil
		}
		messages, err := client.Fetch(intake, r.consumer.Stream, r.consumer.DurableName, batch, maxWait)
		if err != nil {
			if intake.Err() != nil {
				return nil
			}
			failures++
			if failures > maxFetchRetries {
				return fmt.Errorf("%w: %w", ErrFetchExhausted, err)
			}
			if !waitIntake(intake, time.Duration(failures)*100*time.Millisecond) {
				return nil
			}
			continue
		}
		failures = 0
		for _, delivered := range messages {
			if delivered.StreamSequence > uint64(math.MaxInt64) || delivered.DeliveryCount > uint64(math.MaxInt) {
				return fmt.Errorf("%w: JetStream metadata 超出本机整数范围", ErrProcess)
			}
			if err := r.Handle(work, transportMessage{delivered}); err != nil {
				if work.Err() != nil {
					return nil
				}
				if errors.Is(err, ErrAck) {
					return err
				}
				return fmt.Errorf("%w: %w", ErrProcess, err)
			}
		}
	}
}
func waitIntake(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (r *Runner) permanent(ctx context.Context, message Message, eventID *string, code string, validated *ValidatedMessage, report bool) error {
	failure, err := buildPermanentFailure(r.loaded.Config, r.consumer, r.token, message, eventID, code, validated)
	if err != nil {
		return message.NakWithDelay(ctx, r.backoff(message.DeliveryCount()))
	}
	if _, err := r.processor.ProcessPermanentFailure(ctx, failure); err != nil {
		if retryableBusinessError(err) {
			return message.NakWithDelay(ctx, r.backoff(message.DeliveryCount()))
		}
		return err
	}
	if report {
		r.report(code)
	}
	return acknowledge(ctx, message)
}

func acknowledge(ctx context.Context, message Message) error {
	if err := message.Ack(ctx); err != nil {
		return fmt.Errorf("%w: %w", ErrAck, err)
	}
	return nil
}
func retryableBusinessError(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, postgres.ErrFenceStale) || errors.Is(err, postgres.ErrFenceRejected) || errors.Is(err, postgres.ErrInvalidInput) {
		return false
	}
	return errors.Is(err, ErrRetryableBusiness)
}
func (r *Runner) report(code string) {
	if r.health != nil {
		r.health.ReportIngressFailure(code)
	}
}
func (r *Runner) backoff(delivery int) time.Duration {
	if delivery < 1 {
		delivery = 1
	}
	values := r.consumer.BackoffMS
	index := delivery - 1
	if index >= len(values) {
		index = len(values) - 1
	}
	return time.Duration(values[index]) * time.Millisecond
}

func (r *Runner) validate(message Message) (ValidatedMessage, error) {
	body := message.Body()
	if len(body) == 0 || len(body) > MaxBodyBytes || !utf8.Valid(body) {
		return ValidatedMessage{}, fmt.Errorf("%w: invalid_body", ErrPermanent)
	}
	if err := loader.ValidatePointEvent(body); err != nil {
		return ValidatedMessage{}, fmt.Errorf("%w: schema", ErrPermanent)
	}
	var event Event
	if err := json.Unmarshal(body, &event); err != nil {
		return ValidatedMessage{}, fmt.Errorf("%w: json", ErrPermanent)
	}
	if event.Subject != message.Subject() || !subjectMatches(event.SchemaVersion, event.Subject, event.PointID) {
		return ValidatedMessage{}, fmt.Errorf("%w: subject_identity", ErrPermanent)
	}
	if event.AccountID != r.loaded.Config.AccountID {
		return ValidatedMessage{}, fmt.Errorf("%w: account_fence", ErrPermanent)
	}
	if event.DeploymentID != r.loaded.Config.DeploymentID {
		return ValidatedMessage{}, fmt.Errorf("%w: deployment_fence", ErrPermanent)
	}
	if err := verifyEventID(event); err != nil {
		return ValidatedMessage{}, fmt.Errorf("%w: event_id", ErrPermanent)
	}
	producerKey, producerToken, err := r.verifyPointAndProducer(event)
	if err != nil {
		return ValidatedMessage{}, err
	}
	return ValidatedMessage{Event: event, RawBody: append([]byte(nil), body...), Consumer: r.consumer, Token: r.token, ProducerKey: producerKey, ProducerToken: producerToken, StreamPosition: message.StreamPosition(), DeliveryCount: message.DeliveryCount(), OccurredAt: message.OccurredAt().UTC()}, nil
}

func subjectMatches(version, subject, pointID string) bool {
	if pointID == "" {
		return false
	}
	if version == "data.raw.v1" {
		return subject == "data.raw."+pointID
	}
	return version == "data.computed.v1" && subject == "data.computed."+pointID
}
func verifyEventID(event Event) error {
	var expected string
	var err error
	if event.SchemaVersion == "data.raw.v1" {
		expected, err = eventid.Raw(event.SchemaVersion, event.DeploymentID, event.PointID, event.OwnerID, event.Epoch, event.Sequence)
	} else if event.Computation != nil {
		expected, err = eventid.Computed(event.SchemaVersion, event.DeploymentID, event.PointID, event.OwnerID, event.Epoch, event.Sequence, event.Computation.ComputeID, event.Computation.ComputeRevision, event.Computation.InputEventIDs)
	} else {
		return errors.New("computation 缺失")
	}
	if err != nil || expected != event.EventID {
		return errors.New("eventId 不匹配")
	}
	return nil
}

func (r *Runner) verifyPointAndProducer(event Event) (string, ProducerToken, error) {
	point := findPoint(r.loaded.Artifact, event.PointID)
	if point == nil || point.Status != "active" {
		return "", ProducerToken{}, fmt.Errorf("%w: point_status", ErrPermanent)
	}
	if event.SchemaVersion == "data.raw.v1" {
		if point.SourceType == "manual.input" {
			if event.Source != nil || model.ValidateManualProducerFence(r.loaded.Config, model.Ownership{OwnerID: event.OwnerID, Epoch: event.Epoch}) != nil {
				return "", ProducerToken{}, fmt.Errorf("%w: manual_producer_fence", ErrPermanent)
			}
			return "runtime-api", ProducerToken{OwnerID: event.OwnerID, Epoch: event.Epoch}, nil
		}
		if point.SourceType != "collector.point" || event.Source == nil {
			return "", ProducerToken{}, fmt.Errorf("%w: raw_point", ErrPermanent)
		}
		assignment := findCollector(r.loaded.Config, event.Source.CollectorID)
		artifact, exists := r.loaded.CollectorArtifacts[event.Source.CollectorID]
		if assignment == nil || !exists || assignment.Ownership.OwnerID != event.OwnerID || assignment.Ownership.Epoch != event.Epoch {
			return "", ProducerToken{}, fmt.Errorf("%w: raw_producer_fence", ErrPermanent)
		}
		for _, mapping := range artifact.PointMappings {
			if mapping.Enabled && mapping.DatapointID == event.PointID && mapping.ConnectionID == event.Source.ConnectionID && mapping.VariableID == event.Source.VariableID && mapping.DataType == point.DataType {
				return assignment.CollectorID, ProducerToken{OwnerID: assignment.Ownership.OwnerID, Epoch: assignment.Ownership.Epoch}, nil
			}
		}
		return "", ProducerToken{}, fmt.Errorf("%w: raw_mapping", ErrPermanent)
	}
	if point.SourceType != "calc.output" || point.SourceID == nil || event.Computation == nil || *point.SourceID != event.Computation.ComputeID {
		return "", ProducerToken{}, fmt.Errorf("%w: computed_point", ErrPermanent)
	}
	unit := findCompute(r.loaded.Artifact, event.Computation.ComputeID)
	assignment := findComputeProducer(r.loaded.Config, event.Computation.ComputeID)
	if unit == nil || !unit.Enabled || unit.Revision != event.Computation.ComputeRevision || assignment == nil || assignment.Ownership.OwnerID != event.OwnerID || assignment.Ownership.Epoch != event.Epoch {
		return "", ProducerToken{}, fmt.Errorf("%w: computed_producer_fence", ErrPermanent)
	}
	for _, output := range unit.Outputs {
		if output.DatapointID == event.PointID && output.DataType == point.DataType {
			return assignment.ComputeID, ProducerToken{OwnerID: assignment.Ownership.OwnerID, Epoch: assignment.Ownership.Epoch}, nil
		}
	}
	return "", ProducerToken{}, fmt.Errorf("%w: computed_output", ErrPermanent)
}
func findPoint(a model.ProjectArtifact, id string) *model.DataPoint {
	for i := range a.DataPoints {
		if a.DataPoints[i].ID == id {
			return &a.DataPoints[i]
		}
	}
	return nil
}
func findCompute(a model.ProjectArtifact, id string) *model.ComputeUnit {
	for i := range a.ComputeUnits {
		if a.ComputeUnits[i].ID == id {
			return &a.ComputeUnits[i]
		}
	}
	return nil
}
func findCollector(c model.EngineConfig, id string) *model.ProducerAssignment {
	for i := range c.ProducerAssignments {
		p := &c.ProducerAssignments[i]
		if p.ProducerType == "collector" && p.CollectorID == id {
			return p
		}
	}
	return nil
}
func findComputeProducer(c model.EngineConfig, id string) *model.ProducerAssignment {
	for i := range c.ProducerAssignments {
		p := &c.ProducerAssignments[i]
		if p.ProducerType == "compute" && p.ComputeID == id {
			return p
		}
	}
	return nil
}

func buildPermanentFailure(config model.EngineConfig, consumer model.Consumer, token ConsumerToken, message Message, eventID *string, code string, validated *ValidatedMessage) (PermanentFailure, error) {
	body := append([]byte(nil), message.Body()...)
	digest := sha256.Sum256(body)
	bodySHA := "sha256:" + hex.EncodeToString(digest[:])
	id := ""
	if eventID != nil {
		id = *eventID
	}
	dlqID, _ := eventid.HashFields("runtime.dlq.event.v1", config.DeploymentID, consumer.ConsumerKey, message.Subject(), id, code, bodySHA)
	payload, err := encodeDLQPayload(dlqID, config.DeploymentID, config.AccountID, consumer.ConsumerKey, message.Subject(), eventID, code, message.DeliveryCount(), bodySHA, body, message.OccurredAt())
	if err != nil {
		return PermanentFailure{}, err
	}
	failure := PermanentFailure{DeploymentID: config.DeploymentID, AccountID: config.AccountID, ConsumerKey: consumer.ConsumerKey, Role: consumer.Role, ConsumerToken: token, OriginalSubject: message.Subject(), EventID: eventID, ReasonCode: code, DeliveryCount: message.DeliveryCount(), BodySHA256: bodySHA, RawBody: body, OccurredAt: message.OccurredAt().UTC(), DeadLetterSubject: consumer.DeadLetterSubject, DLQID: dlqID, DLQPayload: payload, StreamPosition: message.StreamPosition()}
	if validated != nil {
		failure.RequireProducerFence = true
		failure.ProducerKey = ProducerKey(validated.ProducerKey)
		failure.ProducerToken = validated.ProducerToken
	}
	return failure, nil
}
func encodeDLQPayload(dlqID, deploymentID, accountID, consumerKey, subject string, eventID *string, code string, deliveryCount int, bodySHA string, body []byte, occurredAt time.Time) ([]byte, error) {
	payload, _ := json.Marshal(struct {
		SchemaVersion   string  `json:"schemaVersion"`
		DLQID           string  `json:"dlqId"`
		DeploymentID    string  `json:"deploymentId"`
		AccountID       string  `json:"accountId"`
		ConsumerKey     string  `json:"consumerKey"`
		OriginalSubject string  `json:"originalSubject"`
		EventID         *string `json:"eventId"`
		ReasonCode      string  `json:"reasonCode"`
		DeliveryCount   int     `json:"deliveryCount"`
		BodySHA256      string  `json:"bodySha256"`
		BodyBase64      string  `json:"bodyBase64"`
		OccurredAt      string  `json:"occurredAt"`
	}{"runtime.dlq.event.v1", dlqID, deploymentID, accountID, consumerKey, subject, eventID, code, deliveryCount, bodySHA, base64.StdEncoding.EncodeToString(body), occurredAt.UTC().Format(time.RFC3339Nano)})
	if err := loader.ValidateDLQEvent(payload); err != nil {
		return nil, err
	}
	return payload, nil
}
func eventIDFromBody(body []byte) *string {
	// 仅完整 Schema 成功的原 body 才能作为可审计 eventId；重复键、尾随 JSON 或未知字段一律为空。
	if loader.ValidatePointEvent(body) != nil {
		return nil
	}
	var raw struct {
		EventID string `json:"eventId"`
	}
	if json.Unmarshal(body, &raw) == nil && eventIDPattern.MatchString(raw.EventID) {
		return &raw.EventID
	}
	return nil
}
func reasonCode(err error) string {
	_ = err // 详细原因只可进入受限内部指标；DLQ 契约冻结稳定枚举，禁止透传错误文本。
	return "permanent-validation"
}
