package provisioner

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/indu-forge/runtime-engine/internal/binding"
	"github.com/indu-forge/runtime-engine/internal/model"
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
func (f *fakeAdmin) Close() {}

func TestProvisionNATSCreateAndIdempotent(t *testing.T) {
	in := provisionInput()
	fake := &fakeAdmin{streams: map[string]*nats.StreamInfo{}, consumers: map[string]*nats.ConsumerInfo{}}
	if err := ProvisionNATSWithAdmin(context.Background(), in, fake); err != nil {
		t.Fatal(err)
	}
	if fake.created != 6 {
		t.Fatalf("created = %d, want 6", fake.created)
	}
	if err := ProvisionNATSWithAdmin(context.Background(), in, fake); err != nil {
		t.Fatal(err)
	}
	if fake.created != 6 || fake.updated != 0 {
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
	if len(streams) != 4 || total != 1<<30 {
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
	return Input{Binding: binding.Input{AccountID: "if-project", JetStream: binding.JetStreamInput{Endpoint: "nats://nats.default.svc:4222", DataRawStream: "RAW", DataDerivedStream: "DERIVED", EventStream: "EVENT", DeadLetterStream: "DLQ", Consumers: []model.Consumer{{Stream: "RAW", DurableName: "COMPUTE", FilterSubject: "data.raw.>", AckWaitMS: 30000, MaxAckPending: 10, MaxWaiting: 10, MaxRequestBatch: 10, MaxRequestExpiresMS: 5000, MaxRequestMaxBytes: 1048576, DeadLetterSubject: "dlq.compute"}, {Stream: "DERIVED", DurableName: "ALARM", FilterSubject: "data.computed.>", AckWaitMS: 30000, MaxAckPending: 10, MaxWaiting: 10, MaxRequestBatch: 10, MaxRequestExpiresMS: 5000, MaxRequestMaxBytes: 1048576, DeadLetterSubject: "dlq.alarm"}}}}}
}
