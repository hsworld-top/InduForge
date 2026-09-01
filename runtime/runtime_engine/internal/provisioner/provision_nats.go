package provisioner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/indu-forge/runtime-engine/internal/binding"
	"github.com/indu-forge/runtime-engine/internal/model"
	"github.com/indu-forge/runtime-engine/internal/transportlimits"
	"github.com/nats-io/nats.go"
)

const (
	provisionTimeout = 15 * time.Second
	streamMaxAge     = 7 * 24 * time.Hour
	streamMaxBytes   = int64(1 << 30)
)

// NATSCredentials 是 bootstrap 挂载中的最小 NATS 凭据。它从不实现 String，且
// 调用方不得将其写入日志、事件或错误中。
type NATSCredentials struct {
	SchemaVersion string `json:"schemaVersion"`
	AuthType      string `json:"authType"`
	Token         string `json:"token"`
}

// JetStreamAdmin 将网络 API 收敛在可测试边界；资源描述不包含认证数据。
type JetStreamAdmin interface {
	Stream(ctx context.Context, name string) (*nats.StreamInfo, bool, error)
	CreateStream(ctx context.Context, config *nats.StreamConfig) error
	UpdateStream(ctx context.Context, config *nats.StreamConfig) error
	Consumer(ctx context.Context, stream, durable string) (*nats.ConsumerInfo, bool, error)
	CreateConsumer(ctx context.Context, stream string, config *nats.ConsumerConfig) error
	UpdateConsumer(ctx context.Context, stream string, config *nats.ConsumerConfig) error
	Close()
}

type natsAdmin struct {
	nc *nats.Conn
	js nats.JetStreamContext
}

func (a *natsAdmin) Stream(ctx context.Context, name string) (*nats.StreamInfo, bool, error) {
	info, err := a.js.StreamInfo(name, nats.Context(ctx))
	if errors.Is(err, nats.ErrStreamNotFound) {
		return nil, false, nil
	}
	return info, err == nil, err
}
func (a *natsAdmin) CreateStream(ctx context.Context, c *nats.StreamConfig) error {
	_, err := a.js.AddStream(c, nats.Context(ctx))
	return err
}
func (a *natsAdmin) UpdateStream(ctx context.Context, c *nats.StreamConfig) error {
	_, err := a.js.UpdateStream(c, nats.Context(ctx))
	return err
}
func (a *natsAdmin) Consumer(ctx context.Context, stream, durable string) (*nats.ConsumerInfo, bool, error) {
	info, err := a.js.ConsumerInfo(stream, durable, nats.Context(ctx))
	if errors.Is(err, nats.ErrConsumerNotFound) {
		return nil, false, nil
	}
	return info, err == nil, err
}
func (a *natsAdmin) CreateConsumer(ctx context.Context, stream string, c *nats.ConsumerConfig) error {
	_, err := a.js.AddConsumer(stream, c, nats.Context(ctx))
	return err
}
func (a *natsAdmin) UpdateConsumer(ctx context.Context, stream string, c *nats.ConsumerConfig) error {
	_, err := a.js.UpdateConsumer(stream, c, nats.Context(ctx))
	return err
}
func (a *natsAdmin) Close() {
	if a != nil && a.nc != nil {
		a.nc.Close()
	}
}

func decodeNATSCredentials(raw []byte) (NATSCredentials, error) {
	var credentials NATSCredentials
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&credentials); err != nil || d.Decode(&struct{}{}) != io.EOF {
		return NATSCredentials{}, errors.New("NATS 凭据格式非法")
	}
	if credentials.SchemaVersion != "nats-credential.v1" || credentials.AuthType != "token" || strings.TrimSpace(credentials.Token) == "" {
		return NATSCredentials{}, errors.New("NATS 凭据不完整")
	}
	return credentials, nil
}

func openNATSAdmin(ctx context.Context, endpoint, account string, credentials NATSCredentials) (JetStreamAdmin, error) {
	// account 是输入中的受控逻辑身份；broker account 由 token 绑定，绝不回显任一值。
	if err := validateNATSEndpoint(endpoint); err != nil || strings.TrimSpace(account) == "" {
		return nil, errors.New("NATS 连接参数非法")
	}
	nc, err := nats.Connect(endpoint, nats.Token(credentials.Token), nats.Timeout(provisionTimeout), nats.Name("if-runtime-provisioner"))
	if err != nil {
		return nil, errors.New("NATS 连接失败")
	}
	if err := nc.FlushWithContext(ctx); err != nil {
		nc.Close()
		return nil, errors.New("NATS 连接校验失败")
	}
	js, err := nc.JetStream(nats.Context(ctx))
	if err != nil {
		nc.Close()
		return nil, errors.New("JetStream 初始化失败")
	}
	return &natsAdmin{nc: nc, js: js}, nil
}

func validateNATSEndpoint(endpoint string) error {
	u, err := url.Parse(endpoint)
	if err != nil || (u.Scheme != "nats" && u.Scheme != "tls") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return errors.New("endpoint 非法")
	}
	return nil
}

// ProvisionNATS 建立本 deployment 所需的 JetStream 拓扑。任一阶段失败即返回，
// 不会把 token、endpoint 或 broker 错误文本带到 Pod 日志中。
func ProvisionNATS(ctx context.Context, input Input, credentialsRaw []byte) error {
	credentials, err := decodeNATSCredentials(credentialsRaw)
	if err != nil {
		return err
	}
	timeout, cancel := context.WithTimeout(ctx, provisionTimeout)
	defer cancel()
	admin, err := openNATSAdmin(timeout, input.Binding.JetStream.Endpoint, input.Binding.AccountID, credentials)
	if err != nil {
		return err
	}
	defer admin.Close()
	return ProvisionNATSWithAdmin(timeout, input, admin)
}

// ProvisionNATSWithAdmin 供单测和受控实现复用；错误只带阶段与资源名称。
func ProvisionNATSWithAdmin(ctx context.Context, input Input, admin JetStreamAdmin) error {
	if admin == nil || validateNATSInput(input) != nil {
		return errors.New("NATS provision 输入非法")
	}
	streams, consumers := natsTopology(input.Binding.JetStream)
	for _, expected := range streams {
		actual, exists, err := admin.Stream(ctx, expected.Name)
		if err != nil {
			return provisionError("读取 stream", expected.Name)
		}
		if !exists {
			if err := admin.CreateStream(ctx, expected); err != nil {
				return provisionError("创建 stream", expected.Name)
			}
			continue
		}
		if !sameSubjects(actual.Config.Subjects, expected.Subjects) {
			return provisionError("stream subject 冲突", expected.Name)
		}
		if !sameStreamPolicy(actual.Config, *expected) {
			if err := admin.UpdateStream(ctx, expected); err != nil {
				return provisionError("更新 stream", expected.Name)
			}
		}
	}
	for _, expected := range consumers {
		actual, exists, err := admin.Consumer(ctx, expected.stream, expected.config.Durable)
		if err != nil {
			return provisionError("读取 consumer", expected.config.Durable)
		}
		if !exists {
			if err := admin.CreateConsumer(ctx, expected.stream, expected.config); err != nil {
				return provisionError("创建 consumer", expected.config.Durable)
			}
			continue
		}
		if actual.Config.FilterSubject != expected.config.FilterSubject || actual.Config.Durable != expected.config.Durable {
			return provisionError("consumer subject 冲突", expected.config.Durable)
		}
		if !sameConsumerPolicy(actual.Config, *expected.config) {
			if err := admin.UpdateConsumer(ctx, expected.stream, expected.config); err != nil {
				return provisionError("更新 consumer", expected.config.Durable)
			}
		}
	}
	return nil
}

func provisionError(stage, name string) error { return fmt.Errorf("NATS %s失败: %s", stage, name) }

func validateNATSInput(in Input) error {
	j := in.Binding.JetStream
	if strings.TrimSpace(in.Binding.AccountID) == "" || validateNATSEndpoint(j.Endpoint) != nil || j.DataRawStream == "" || j.DataDerivedStream == "" || j.EventStream == "" || j.DeadLetterStream == "" || len(j.Consumers) == 0 {
		return errors.New("input")
	}
	for _, c := range j.Consumers {
		if c.Stream == "" || c.DurableName == "" || c.FilterSubject == "" || c.AckWaitMS <= 0 || c.MaxAckPending <= 0 || c.MaxWaiting <= 0 || c.MaxRequestBatch <= 0 || c.MaxRequestExpiresMS <= 0 || c.MaxRequestMaxBytes <= 0 || c.DeadLetterSubject == "" {
			return errors.New("consumer")
		}
		// 输入虽来自受控 ConfigMap，仍只接受 runtime-engine 已冻结的两条
		// 项目数据 subject，避免 provisioner 因错误输入取得其它项目 subject。
		if (c.Stream != j.DataRawStream || c.FilterSubject != "data.raw.>") && (c.Stream != j.DataDerivedStream || c.FilterSubject != "data.computed.>") {
			return errors.New("consumer subject")
		}
	}
	return nil
}

type expectedConsumer struct {
	stream string
	config *nats.ConsumerConfig
}

func natsTopology(in binding.JetStreamInput) ([]*nats.StreamConfig, []expectedConsumer) {
	streams := []*nats.StreamConfig{
		streamConfig(in.DataRawStream, []string{"data.raw.>"}, int32(transportlimits.MaxBodyBytes)),
		streamConfig(in.DataDerivedStream, []string{"data.computed.>"}, int32(transportlimits.MaxBodyBytes)),
		streamConfig(in.EventStream, []string{"alarm.event"}, int32(transportlimits.MaxBodyBytes)),
	}
	dlqSubjects := make([]string, 0, len(in.Consumers))
	seen := make(map[string]struct{}, len(in.Consumers))
	consumers := make([]expectedConsumer, 0, len(in.Consumers))
	for _, consumer := range in.Consumers {
		if _, ok := seen[consumer.DeadLetterSubject]; !ok {
			seen[consumer.DeadLetterSubject] = struct{}{}
			dlqSubjects = append(dlqSubjects, consumer.DeadLetterSubject)
		}
		consumers = append(consumers, expectedConsumer{stream: consumer.Stream, config: consumerConfig(consumer)})
	}
	streams = append(streams, streamConfig(in.DeadLetterStream, dlqSubjects, int32(transportlimits.MaxDLQPayloadBytes)))
	return streams, consumers
}

func streamConfig(name string, subjects []string, maxMsgSize int32) *nats.StreamConfig {
	return &nats.StreamConfig{Name: name, Subjects: append([]string(nil), subjects...), Retention: nats.LimitsPolicy, Storage: nats.FileStorage, Discard: nats.DiscardOld, MaxAge: streamMaxAge, MaxBytes: streamMaxBytes, MaxMsgSize: maxMsgSize}
}

func consumerConfig(in model.Consumer) *nats.ConsumerConfig {
	backoff := make([]time.Duration, len(in.BackoffMS))
	for i, value := range in.BackoffMS {
		backoff[i] = time.Duration(value) * time.Millisecond
	}
	return &nats.ConsumerConfig{Durable: in.DurableName, FilterSubject: in.FilterSubject, DeliverPolicy: nats.DeliverAllPolicy, ReplayPolicy: nats.ReplayInstantPolicy, AckPolicy: nats.AckExplicitPolicy, AckWait: time.Duration(in.AckWaitMS) * time.Millisecond, MaxDeliver: -1, BackOff: backoff, MaxAckPending: in.MaxAckPending, MaxWaiting: in.MaxWaiting, MaxRequestBatch: in.MaxRequestBatch, MaxRequestExpires: time.Duration(in.MaxRequestExpiresMS) * time.Millisecond, MaxRequestMaxBytes: in.MaxRequestMaxBytes}
}

func sameSubjects(actual, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}
	set := make(map[string]struct{}, len(actual))
	for _, value := range actual {
		if value == "" {
			return false
		}
		set[value] = struct{}{}
	}
	for _, value := range expected {
		if _, ok := set[value]; !ok {
			return false
		}
	}
	return true
}

func sameStreamPolicy(actual, expected nats.StreamConfig) bool {
	return actual.Retention == expected.Retention && actual.Storage == expected.Storage && actual.Discard == expected.Discard && actual.MaxAge == expected.MaxAge && actual.MaxBytes == expected.MaxBytes && actual.MaxMsgSize == expected.MaxMsgSize
}

func sameConsumerPolicy(actual, expected nats.ConsumerConfig) bool {
	if actual.AckPolicy != expected.AckPolicy || actual.AckWait != expected.AckWait || actual.MaxDeliver != expected.MaxDeliver || actual.DeliverPolicy != expected.DeliverPolicy || actual.ReplayPolicy != expected.ReplayPolicy || actual.DeliverSubject != "" || actual.DeliverGroup != "" || actual.FlowControl || actual.MaxAckPending != expected.MaxAckPending || actual.MaxWaiting != expected.MaxWaiting || actual.MaxRequestBatch != expected.MaxRequestBatch || actual.MaxRequestExpires != expected.MaxRequestExpires || actual.MaxRequestMaxBytes != expected.MaxRequestMaxBytes {
		return false
	}
	if len(actual.BackOff) != len(expected.BackOff) {
		return false
	}
	for i := range actual.BackOff {
		if actual.BackOff[i] != expected.BackOff[i] {
			return false
		}
	}
	return true
}
