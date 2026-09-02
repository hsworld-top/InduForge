// Package jetstream 为 RuntimeEngine 提供只读 JetStream 拓扑校验和可靠消息收发。
package jetstream

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/indu-forge/runtime-engine/internal/model"
	"github.com/indu-forge/runtime-engine/internal/transportlimits"
	"github.com/nats-io/nats.go"
	js "github.com/nats-io/nats.go/jetstream"
)

const msgIDHeader = "Nats-Msg-Id"
const (
	MaxFetchBatch   = 32
	MaxFetchWait    = 5 * time.Second
	natsOpenTimeout = 15 * time.Second
)

var (
	// These sentinels let the outbox distinguish a deterministically invalid
	// local record from a broker outage, so it never retries it forever.
	ErrPayloadTooLarge = transportlimits.ErrOutboundPayloadTooLarge
	ErrInvalidSubject  = transportlimits.ErrOutboundSubject
	errNATSConnect     = errors.New("nats connect failed")
	errNATSFlush       = errors.New("nats flush failed")
	errJetStreamInit   = errors.New("jetstream init failed")
)

// DiagnosticCode 只将连接阶段归类为稳定码，禁止把 endpoint、token 或底层错误文本
// 写入运行日志。调用方据此区分网络、认证确认与 JetStream 初始化失败。
func DiagnosticCode(err error) string {
	switch {
	case errors.Is(err, errNATSConnect):
		return "CONNECT"
	case errors.Is(err, errNATSFlush):
		return "FLUSH"
	case errors.Is(err, errJetStreamInit):
		return "JETSTREAM"
	default:
		return "UNKNOWN"
	}
}

// AccountIdentity 是 resolver 已验证的 NATS Account 身份。它刻意不实现可读 String/Marshal。
type AccountIdentity struct{ value string }

func NewAccountIdentity(value string) (AccountIdentity, error) {
	if value == "" {
		return AccountIdentity{}, errors.New("NATS account identity 为空")
	}
	return AccountIdentity{value: value}, nil
}
func (a AccountIdentity) Equal(expected string) bool { return a.value == expected }
func (a AccountIdentity) valid() bool                { return a.value != "" }

// ConnectionOptions 仅由受信 resource/secret resolver 构造；不得写入日志、status 或 JSON。
type ConnectionOptions struct {
	serverURL string
	account   AccountIdentity
	options   []nats.Option
}

func NewConnectionOptions(serverURL string, account AccountIdentity, options ...nats.Option) (ConnectionOptions, error) {
	if serverURL == "" || !account.valid() {
		return ConnectionOptions{}, errors.New("NATS connection options 非法")
	}
	return ConnectionOptions{serverURL: serverURL, account: account, options: append([]nats.Option(nil), options...)}, nil
}

type Client struct {
	nc        *nats.Conn
	js        js.JetStream
	closeOnce sync.Once
}

func Open(ctx context.Context, options ConnectionOptions, expectedAccount string) (*Client, error) {
	if !options.account.Equal(expectedAccount) {
		return nil, errors.New("NATS credential account 与 deployment account 不一致")
	}
	// RuntimeEngine 的启动根 context 通常没有 deadline，而 nats.go 的
	// FlushWithContext 要求 deadline。连接与确认仅使用局部有界上下文，不能把
	// 运行期取消语义或无限等待带入启动预检。
	openCtx, cancel := newOpenContext(ctx)
	defer cancel()
	connectOptions := append(append([]nats.Option(nil), options.options...), nats.Timeout(natsOpenTimeout))
	nc, err := nats.Connect(options.serverURL, connectOptions...)
	if err != nil {
		return nil, errNATSConnect
	}
	if err := nc.FlushWithContext(openCtx); err != nil {
		nc.Close()
		return nil, errNATSFlush
	}
	client, err := js.New(nc)
	if err != nil {
		nc.Close()
		return nil, errJetStreamInit
	}
	return &Client{nc: nc, js: client}, nil
}

func newOpenContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, natsOpenTimeout)
}

func (c *Client) Close() {
	if c != nil {
		c.closeOnce.Do(func() {
			if c.nc != nil {
				// Timeout shutdown must sever the broker immediately. Normal worker
				// drain happened before this boundary; nats.Conn.Close is safe here.
				c.nc.Close()
			}
		})
	}
}

// ValidateStreamsAndConsumers 只读取并比对既有设施，生产代码绝不创建或更新 stream/consumer。
func (c *Client) ValidateStreamsAndConsumers(ctx context.Context, config model.EngineConfig) error {
	for _, expected := range []struct{ name, subject string }{
		{config.JetStream.DataRawStream, "data.raw.>"},
		{config.JetStream.DataDerivedStream, "data.computed.>"},
		{config.JetStream.EventStream, "alarm.event"},
		{config.JetStream.CommandStream, "compute.command.>"},
	} {
		stream, err := c.js.Stream(ctx, expected.name)
		if err != nil {
			return fmt.Errorf("JetStream stream 不可用: %w", err)
		}
		info, err := stream.Info(ctx)
		if err != nil {
			return errors.New("JetStream stream 信息读取失败")
		}
		if !exactSubjects(info.Config.Subjects, []string{expected.subject}) {
			return errors.New("JetStream stream subject 拓扑不匹配")
		}
		if (expected.name == config.JetStream.DataRawStream || expected.name == config.JetStream.DataDerivedStream || expected.name == config.JetStream.CommandStream) && (info.Config.MaxMsgSize <= 0 || info.Config.MaxMsgSize > transportlimits.MaxBodyBytes) {
			return errors.New("DATA/COMMAND stream MaxMsgSize 必须显式且不大于 1MiB")
		}
	}
	dlq, err := c.js.Stream(ctx, config.JetStream.DeadLetterStream)
	if err != nil {
		return errors.New("JetStream DLQ stream 不可用")
	}
	dlqInfo, err := dlq.Info(ctx)
	if err != nil {
		return errors.New("JetStream DLQ stream 信息读取失败")
	}
	if dlqInfo.Config.MaxMsgSize < transportlimits.MaxDLQPayloadBytes || c.nc.MaxPayload() < int64(transportlimits.MaxDLQPayloadBytes) {
		return errors.New("DLQ stream 或 NATS server MaxPayload 小于最大合法 DLQ")
	}
	expectedDLQ := make([]string, 0, len(config.JetStream.Consumers))
	seenDLQ := map[string]struct{}{}
	for _, consumer := range config.JetStream.Consumers {
		if _, ok := seenDLQ[consumer.DeadLetterSubject]; !ok {
			seenDLQ[consumer.DeadLetterSubject] = struct{}{}
			expectedDLQ = append(expectedDLQ, consumer.DeadLetterSubject)
		}
	}
	if !exactSubjects(dlqInfo.Config.Subjects, expectedDLQ) {
		return errors.New("DLQ stream subjects 必须精确等于配置 deadLetterSubject 集合")
	}
	// A durable on either data stream is an active processing authority.  Do
	// not merely verify that configured consumers exist: enumerate every
	// server-side durable and reject additions (including stale query/coord
	// consumers) before any worker or assignment is activated.
	for _, streamName := range []string{config.JetStream.DataRawStream, config.JetStream.DataDerivedStream, config.JetStream.CommandStream} {
		stream, err := c.js.Stream(ctx, streamName)
		if err != nil {
			return errors.New("JetStream consumer stream 不可用")
		}
		expected, err := expectedDurableNames(config, streamName)
		if err != nil {
			return err
		}
		if err := validateDurableNames(ctx, stream, expected); err != nil {
			return err
		}
	}
	for _, expected := range config.JetStream.Consumers {
		stream, err := c.js.Stream(ctx, expected.Stream)
		if err != nil {
			return errors.New("JetStream consumer stream 不可用")
		}
		consumer, err := stream.Consumer(ctx, expected.DurableName)
		if err != nil {
			return errors.New("JetStream durable consumer 不存在")
		}
		info, err := consumer.Info(ctx)
		if err != nil {
			return errors.New("JetStream consumer 信息读取失败")
		}
		if err := compareConsumer(info.Config, expected); err != nil {
			return err
		}
	}
	return nil
}

func expectedDurableNames(config model.EngineConfig, streamName string) ([]string, error) {
	if streamName != config.JetStream.DataRawStream && streamName != config.JetStream.DataDerivedStream && streamName != config.JetStream.CommandStream {
		return nil, errors.New("仅允许校验 data/command stream durable")
	}
	seen := map[string]struct{}{}
	expected := make([]string, 0, len(config.JetStream.Consumers))
	for _, consumer := range config.JetStream.Consumers {
		if consumer.Stream != streamName {
			continue
		}
		if consumer.DurableName == "" {
			return nil, errors.New("JetStream durable consumer 配置非法")
		}
		if _, duplicate := seen[consumer.DurableName]; duplicate {
			return nil, errors.New("JetStream durable consumer 配置重复")
		}
		seen[consumer.DurableName] = struct{}{}
		expected = append(expected, consumer.DurableName)
	}
	return expected, nil
}

func validateDurableNames(ctx context.Context, stream js.Stream, expected []string) error {
	if stream == nil {
		return errors.New("JetStream consumer stream 不可用")
	}
	lister := stream.ConsumerNames(ctx)
	actual := make([]string, 0, len(expected))
	for name := range lister.Name() {
		if name == "" {
			return errors.New("JetStream durable consumer 名称非法")
		}
		actual = append(actual, name)
	}
	if err := lister.Err(); err != nil {
		return errors.New("JetStream durable consumer 枚举失败")
	}
	if !exactDurableNames(actual, expected) {
		return errors.New("JetStream data stream durable 集合与配置不匹配")
	}
	return nil
}

func exactDurableNames(actual, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}
	seen := make(map[string]struct{}, len(actual))
	for _, name := range actual {
		if name == "" {
			return false
		}
		if _, duplicate := seen[name]; duplicate {
			return false
		}
		seen[name] = struct{}{}
	}
	expectedSeen := make(map[string]struct{}, len(expected))
	for _, name := range expected {
		if name == "" {
			return false
		}
		if _, duplicate := expectedSeen[name]; duplicate {
			return false
		}
		expectedSeen[name] = struct{}{}
		if _, present := seen[name]; !present {
			return false
		}
	}
	return true
}

func exactSubjects(actual, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}
	found := map[string]bool{}
	for _, value := range actual {
		if found[value] {
			return false
		}
		found[value] = true
	}
	for _, value := range expected {
		if !found[value] {
			return false
		}
	}
	return true
}

func compareConsumer(actual js.ConsumerConfig, expected model.Consumer) error {
	if actual.Durable != expected.DurableName || actual.FilterSubject != expected.FilterSubject || len(actual.FilterSubjects) != 0 || actual.AckPolicy != js.AckExplicitPolicy || actual.AckWait != time.Duration(model.EffectiveAckWaitMS(expected))*time.Millisecond || actual.MaxDeliver != expected.MaxDeliver || actual.DeliverPolicy != js.DeliverAllPolicy || actual.ReplayPolicy != js.ReplayInstantPolicy || actual.DeliverSubject != "" || actual.DeliverGroup != "" || actual.FlowControl || actual.IdleHeartbeat != 0 || actual.HeadersOnly || actual.OptStartSeq != 0 || actual.OptStartTime != nil || actual.PauseUntil != nil || actual.InactiveThreshold != 0 || actual.MaxRequestBatch != expected.MaxRequestBatch || actual.MaxRequestExpires != time.Duration(expected.MaxRequestExpiresMS)*time.Millisecond || actual.MaxRequestMaxBytes != expected.MaxRequestMaxBytes || actual.MaxAckPending != expected.MaxAckPending || actual.MaxWaiting != expected.MaxWaiting || actual.RateLimit != 0 || actual.SampleFrequency != "" || actual.MemoryStorage || actual.Replicas != 0 || !serverOnlyMetadata(actual.Metadata) || actual.PriorityPolicy != js.PriorityPolicyNone || actual.PinnedTTL != 0 || len(actual.PriorityGroups) != 0 {
		return errors.New("JetStream durable consumer 配置不匹配")
	}
	if len(actual.BackOff) != len(expected.BackoffMS) {
		return errors.New("JetStream consumer backoff 不匹配")
	}
	for index, backoff := range actual.BackOff {
		if backoff != time.Duration(expected.BackoffMS[index])*time.Millisecond {
			return errors.New("JetStream consumer backoff 不匹配")
		}
	}
	return nil
}
func serverOnlyMetadata(metadata map[string]string) bool {
	for key := range metadata {
		if !strings.HasPrefix(key, "_nats.") {
			return false
		}
	}
	return true
}

func coveredByAny(patterns []string, subject string) bool {
	for _, p := range patterns {
		if subjectCovers(p, subject) {
			return true
		}
	}
	return false
}
func subjectCovers(pattern, subject string) bool {
	p, s := strings.Split(pattern, "."), strings.Split(subject, ".")
	for i := 0; i < len(p); i++ {
		if p[i] == ">" {
			return i == len(p)-1 && i < len(s)
		}
		if i >= len(s) || (p[i] != "*" && p[i] != s[i]) {
			return false
		}
	}
	return len(p) == len(s)
}

type DeliveredMessage struct {
	Subject        string
	Body           []byte
	StreamSequence uint64
	DeliveryCount  uint64
	OccurredAt     time.Time
	ack            func(context.Context) error
	inProgress     func(context.Context) error
	nak            func(context.Context, time.Duration) error
}

func (m DeliveredMessage) Ack(ctx context.Context) error        { return m.ack(ctx) }
func (m DeliveredMessage) InProgress(ctx context.Context) error { return m.inProgress(ctx) }
func (m DeliveredMessage) NakWithDelay(ctx context.Context, delay time.Duration) error {
	return m.nak(ctx, delay)
}

// Fetch 获取至多 batch 条消息；超时返回空切片，不会创建 consumer。
func (c *Client) Fetch(ctx context.Context, streamName, durable string, batch int, maxWait time.Duration) ([]DeliveredMessage, error) {
	if batch < 1 || batch > MaxFetchBatch || maxWait <= 0 || maxWait > MaxFetchWait {
		return nil, errors.New("JetStream fetch 参数非法")
	}
	stream, err := c.js.Stream(ctx, streamName)
	if err != nil {
		return nil, errors.New("JetStream stream 不可用")
	}
	consumer, err := stream.Consumer(ctx, durable)
	if err != nil {
		return nil, errors.New("JetStream consumer 不可用")
	}
	messages, err := consumer.Fetch(batch, js.FetchContext(ctx), js.FetchMaxWait(maxWait))
	if err != nil {
		return nil, errors.New("JetStream fetch 失败")
	}
	result := make([]DeliveredMessage, 0, batch)
	for message := range messages.Messages() {
		metadata, err := message.Metadata()
		if err != nil {
			return nil, errors.New("JetStream metadata 不可用")
		}
		body := append([]byte(nil), message.Data()...)
		m := message
		result = append(result, DeliveredMessage{Subject: m.Subject(), Body: body, StreamSequence: metadata.Sequence.Stream, DeliveryCount: metadata.NumDelivered, OccurredAt: metadata.Timestamp.UTC(), ack: func(ctx context.Context) error { return m.Ack() }, inProgress: func(ctx context.Context) error { return m.InProgress() }, nak: func(ctx context.Context, d time.Duration) error { return m.NakWithDelay(d) }})
	}
	if err := messages.Error(); err != nil && !errors.Is(err, js.ErrNoMessages) {
		return nil, errors.New("JetStream fetch 消息失败")
	}
	return result, nil
}

type PublishMessage struct {
	Subject   string
	Headers   json.RawMessage
	Payload   []byte
	DedupeKey string
}

func (c *Client) Publish(ctx context.Context, message PublishMessage) error {
	if message.Subject == "" || message.DedupeKey == "" {
		return errors.New("JetStream publish 参数非法")
	}
	if err := transportlimits.ValidateOutboundPayload(message.Subject, message.Payload); err != nil {
		return err
	}
	var headers map[string]string
	if len(message.Headers) > 0 && json.Unmarshal(message.Headers, &headers) != nil {
		return errors.New("outbox headers 非法")
	}
	msg := nats.NewMsg(message.Subject)
	for key, value := range headers {
		if key != msgIDHeader {
			msg.Header.Set(key, value)
		}
	}
	msg.Header.Set(msgIDHeader, message.DedupeKey)
	msg.Data = append([]byte(nil), message.Payload...)
	ack, err := c.js.PublishMsg(ctx, msg)
	if err != nil || ack == nil {
		return errors.New("JetStream PubAck 失败")
	}
	return nil // Duplicate PubAck 同样表示 broker 已接受该稳定 Msg-Id。
}

func PayloadSHA256(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
