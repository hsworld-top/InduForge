package provisioner

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/indu-forge/runtime-engine/internal/binding"
	"github.com/indu-forge/runtime-engine/internal/model"
	"github.com/indu-forge/runtime-engine/internal/transportlimits"
	"github.com/nats-io/nats.go"
)

type fakeAdmin struct {
	streams   map[string]*nats.StreamInfo
	consumers map[string]*nats.ConsumerInfo
	failAt    string
	created   int
	updated   int
}

func (f *fakeAdmin) Stream(_ context.Context, name string) (*nats.StreamInfo, bool, error) {
	if f.failAt == "stream:"+name {
		return nil, false, errors.New("broker failure")
	}
	v, ok := f.streams[name]
	return v, ok, nil
}
func (f *fakeAdmin) CreateStream(_ context.Context, c *nats.StreamConfig) error {
	if f.failAt == "create-stream:"+c.Name {
		return errors.New("broker failure")
	}
	f.streams[c.Name] = &nats.StreamInfo{Config: *c}
	f.created++
	return nil
}
func (f *fakeAdmin) UpdateStream(_ context.Context, c *nats.StreamConfig) error {
	f.streams[c.Name] = &nats.StreamInfo{Config: *c}
	f.updated++
	return nil
}
func (f *fakeAdmin) Consumer(_ context.Context, stream, durable string) (*nats.ConsumerInfo, bool, error) {
	if f.failAt == "consumer:"+durable {
		return nil, false, errors.New("broker failure")
	}
	v, ok := f.consumers[stream+"/"+durable]
	return v, ok, nil
}
func (f *fakeAdmin) CreateConsumer(_ context.Context, stream string, c *nats.ConsumerConfig) error {
	if f.failAt == "create-consumer:"+c.Durable {
		return errors.New("broker failure")
	}
	f.consumers[stream+"/"+c.Durable] = &nats.ConsumerInfo{Stream: stream, Config: *c}
	f.created++
	return nil
}
func (f *fakeAdmin) UpdateConsumer(_ context.Context, stream string, c *nats.ConsumerConfig) error {
	f.consumers[stream+"/"+c.Durable] = &nats.ConsumerInfo{Stream: stream, Config: *c}
	f.updated++
	return nil
}
func (f *fakeAdmin) DeleteConsumer(_ context.Context, stream, durable string) error {
	delete(f.consumers, stream+"/"+durable)
	f.updated++
	return nil
}
func (f *fakeAdmin) Close() {}

func TestProvisionNATSCreateAndIdempotent(t *testing.T) {
	in := provisionInput()
	fake := &fakeAdmin{streams: map[string]*nats.StreamInfo{}, consumers: map[string]*nats.ConsumerInfo{}}
	if err := ProvisionNATSWithAdmin(context.Background(), in, fake); err != nil {
		t.Fatal(err)
	}
	if fake.created != 8 {
		t.Fatalf("created = %d, want 8", fake.created)
	}
	if err := ProvisionNATSWithAdmin(context.Background(), in, fake); err != nil {
		t.Fatal(err)
	}
	if fake.created != 8 || fake.updated != 0 {
		t.Fatalf("idempotent result created=%d updated=%d", fake.created, fake.updated)
	}
	if got := fake.streams["RAW"].Config; got.Storage != nats.FileStorage || got.Retention != nats.LimitsPolicy {
		t.Fatalf("stream policy = %#v", got)
	}
}

func TestProvisionNATSUsesOneGiBProjectQuotaAndShrinksExistingStreams(t *testing.T) {
	in := provisionInput()
	streams, _ := natsTopology(in.Binding.JetStream)
	var total int64
	for _, stream := range streams {
		total += stream.MaxBytes
	}
	if len(streams) != 5 || total != (4<<28)+(1<<26) {
		t.Fatalf("stream count=%d total quota=%d", len(streams), total)
	}
	fake := &fakeAdmin{streams: map[string]*nats.StreamInfo{
		"RAW":     {Config: nats.StreamConfig{Name: "RAW", Subjects: []string{"data.raw.>"}, Retention: nats.LimitsPolicy, Storage: nats.FileStorage, Discard: nats.DiscardOld, MaxAge: streamMaxAge, MaxBytes: 1 << 30, MaxMsgSize: int32(1 << 20)}},
		"DERIVED": {Config: nats.StreamConfig{Name: "DERIVED", Subjects: []string{"data.computed.>"}, Retention: nats.LimitsPolicy, Storage: nats.FileStorage, Discard: nats.DiscardOld, MaxAge: streamMaxAge, MaxBytes: 1 << 30, MaxMsgSize: int32(1 << 20)}},
	}, consumers: map[string]*nats.ConsumerInfo{}}
	if err := ProvisionNATSWithAdmin(context.Background(), in, fake); err != nil {
		t.Fatal(err)
	}
	if fake.updated != 2 || fake.streams["RAW"].Config.MaxBytes != streamMaxBytes || fake.streams["DERIVED"].Config.MaxBytes != streamMaxBytes {
		t.Fatalf("existing stream quota was not shrunk: updated=%d raw=%d derived=%d", fake.updated, fake.streams["RAW"].Config.MaxBytes, fake.streams["DERIVED"].Config.MaxBytes)
	}
}

func TestProvisionNATSReconcilesDLQSubjectsAfterRoleRemoved(t *testing.T) {
	in := provisionInput()
	// 旧 deployment 曾启用 alarm；当前完整 topology 只剩 compute，必须删除
	// alarm 的 DLQ subject，而不是把已停止角色留下的 subject 合并保留。
	in.Binding.JetStream.Consumers = in.Binding.JetStream.Consumers[:1]
	in.Binding.JetStream.TopologyConsumers = in.Binding.JetStream.Consumers
	streams, _ := natsTopology(in.Binding.JetStream)
	var dlq *nats.StreamConfig
	for _, stream := range streams {
		if stream.Name == in.Binding.JetStream.DeadLetterStream {
			dlq = stream
		}
	}
	if dlq == nil {
		t.Fatal("缺少 DLQ stream")
	}
	fake := &fakeAdmin{streams: map[string]*nats.StreamInfo{dlq.Name: {Config: nats.StreamConfig{Name: dlq.Name, Subjects: []string{"dlq.alarm"}, Retention: nats.LimitsPolicy, Storage: nats.FileStorage, Discard: nats.DiscardOld, MaxAge: streamMaxAge, MaxBytes: streamMaxBytes, MaxMsgSize: int32(transportlimits.MaxDLQPayloadBytes)}}}, consumers: map[string]*nats.ConsumerInfo{}}
	if err := ProvisionNATSWithAdmin(context.Background(), in, fake); err != nil {
		t.Fatalf("角色删减后的 DLQ 调和失败: %v", err)
	}
	if got := fake.streams[dlq.Name].Config.Subjects; !sameSubjects(got, dlq.Subjects) {
		t.Fatalf("DLQ subjects=%v", got)
	}
}

func TestProvisionNATSRejectsForeignSubject(t *testing.T) {
	in := provisionInput()
	fake := &fakeAdmin{streams: map[string]*nats.StreamInfo{"RAW": {Config: nats.StreamConfig{Name: "RAW", Subjects: []string{"other.project.>"}}}}, consumers: map[string]*nats.ConsumerInfo{}}
	err := ProvisionNATSWithAdmin(context.Background(), in, fake)
	if err == nil || !strings.Contains(err.Error(), "RAW") || !strings.Contains(err.Error(), "冲突") {
		t.Fatalf("err = %v", err)
	}
}

func TestProvisionNATSRejectsConsumerSubjectConflict(t *testing.T) {
	in := provisionInput()
	fake := &fakeAdmin{streams: map[string]*nats.StreamInfo{}, consumers: map[string]*nats.ConsumerInfo{}}
	for _, stream := range func() []*nats.StreamConfig { streams, _ := natsTopology(in.Binding.JetStream); return streams }() {
		fake.streams[stream.Name] = &nats.StreamInfo{Config: *stream}
	}
	fake.consumers["RAW/COMPUTE"] = &nats.ConsumerInfo{Stream: "RAW", Config: nats.ConsumerConfig{Durable: "COMPUTE", FilterSubject: "data.computed.>"}}
	err := ProvisionNATSWithAdmin(context.Background(), in, fake)
	if err == nil || !strings.Contains(err.Error(), "COMPUTE") || !strings.Contains(err.Error(), "冲突") {
		t.Fatalf("err = %v", err)
	}
}

func TestRemoveDisabledRoleConsumersKeepsExpectedAndRejectsUnknownAtRuntime(t *testing.T) {
	topology := binding.JetStreamInput{DataRawStream: "RAW", DataDerivedStream: "DERIVED", CommandStream: "COMMAND"}
	fake := &fakeAdmin{consumers: map[string]*nats.ConsumerInfo{
		"RAW/compute-raw-v1":       {Stream: "RAW", Config: nats.ConsumerConfig{Durable: "compute-raw-v1"}},
		"RAW/alarm-raw-v1":         {Stream: "RAW", Config: nats.ConsumerConfig{Durable: "alarm-raw-v1"}},
		"DERIVED/alarm-derived-v1": {Stream: "DERIVED", Config: nats.ConsumerConfig{Durable: "alarm-derived-v1"}},
		"RAW/base-raw-v1":          {Stream: "RAW", Config: nats.ConsumerConfig{Durable: "base-raw-v1"}},
		// 陌生 durable 不在删除白名单内，预检的精确集合会负责拒绝它。
		"RAW/foreign-v1": {Stream: "RAW", Config: nats.ConsumerConfig{Durable: "foreign-v1"}},
	}}
	expected := []expectedConsumer{{stream: "RAW", config: &nats.ConsumerConfig{Durable: "compute-raw-v1"}}}
	if err := removeDisabledRoleConsumers(context.Background(), fake, topology, expected); err != nil {
		t.Fatal(err)
	}
	if _, exists := fake.consumers["RAW/alarm-raw-v1"]; exists {
		t.Fatal("已停用 alarm durable 未删除")
	}
	if _, exists := fake.consumers["DERIVED/alarm-derived-v1"]; exists {
		t.Fatal("已停用 alarm derived durable 未删除")
	}
	if _, exists := fake.consumers["RAW/base-raw-v1"]; exists {
		t.Fatal("历史 base durable 未删除")
	}
	if _, exists := fake.consumers["RAW/compute-raw-v1"]; !exists {
		t.Fatal("当前 compute durable 被误删")
	}
	if _, exists := fake.consumers["RAW/foreign-v1"]; !exists {
		t.Fatal("陌生 durable 不应在 provision 阶段静默删除")
	}
}

func TestProvisionNATSStopsAtPartialFailureWithoutCredentialLeak(t *testing.T) {
	in := provisionInput()
	fake := &fakeAdmin{streams: map[string]*nats.StreamInfo{}, consumers: map[string]*nats.ConsumerInfo{}, failAt: "create-stream:DERIVED"}
	err := ProvisionNATSWithAdmin(context.Background(), in, fake)
	if err == nil || !strings.Contains(err.Error(), "DERIVED") {
		t.Fatalf("err = %v", err)
	}
	if strings.Contains(err.Error(), "super-secret") || strings.Contains(err.Error(), "nats://") {
		t.Fatalf("secret leaked: %v", err)
	}
	if _, ok := fake.streams["RAW"]; !ok {
		t.Fatal("first successful stream should remain")
	}
}

func TestDecodeNATSCredentialsStrict(t *testing.T) {
	if _, err := decodeNATSCredentials([]byte(`{"schemaVersion":"nats-credential.v1","authType":"token","token":"super-secret","extra":true}`)); err == nil {
		t.Fatal("unknown field accepted")
	}
	if _, err := decodeNATSCredentials([]byte(`{"schemaVersion":"nats-credential.v1","authType":"token","token":"super-secret"}`)); err != nil {
		t.Fatal(err)
	}
}

func provisionInput() Input {
	consumers := []model.Consumer{{Stream: "RAW", DurableName: "COMPUTE", FilterSubject: "data.raw.>", AckWaitMS: 30000, MaxAckPending: 10, MaxWaiting: 10, MaxRequestBatch: 10, MaxRequestExpiresMS: 5000, MaxRequestMaxBytes: 1048576, DeadLetterSubject: "dlq.compute"}, {Stream: "DERIVED", DurableName: "ALARM", FilterSubject: "data.computed.>", AckWaitMS: 30000, MaxAckPending: 10, MaxWaiting: 10, MaxRequestBatch: 10, MaxRequestExpiresMS: 5000, MaxRequestMaxBytes: 1048576, DeadLetterSubject: "dlq.alarm"}, {Stream: "COMMAND", DurableName: "COMMAND", FilterSubject: "compute.command.>", AckWaitMS: 30000, MaxAckPending: 10, MaxWaiting: 10, MaxRequestBatch: 10, MaxRequestExpiresMS: 5000, MaxRequestMaxBytes: 1048576, DeadLetterSubject: "dlq.command"}}
	return Input{Binding: binding.Input{AccountID: "if-project", JetStream: binding.JetStreamInput{Endpoint: "nats://nats.default.svc:4222", DataRawStream: "RAW", DataDerivedStream: "DERIVED", EventStream: "EVENT", CommandStream: "COMMAND", DeadLetterStream: "DLQ", Consumers: consumers, TopologyConsumers: consumers}}}
}
