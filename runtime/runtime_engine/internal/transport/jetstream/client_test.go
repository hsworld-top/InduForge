package jetstream

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/indu-forge/runtime-engine/internal/model"
	"github.com/indu-forge/runtime-engine/internal/transportlimits"
	js "github.com/nats-io/nats.go/jetstream"
)

func TestSevenConfiguredConsumersRequireExactDurablePullSettings(t *testing.T) {
	var config model.EngineConfig
	bytes, err := os.ReadFile(filepath.Join("../../../../../contracts/runtime/fixtures", "runtime-engine-config.valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(bytes, &config); err != nil {
		t.Fatal(err)
	}
	if len(config.JetStream.Consumers) != 7 {
		t.Fatalf("fixture consumer count=%d", len(config.JetStream.Consumers))
	}
	for _, consumer := range config.JetStream.Consumers {
		actual := js.ConsumerConfig{Durable: consumer.DurableName, FilterSubject: consumer.FilterSubject, DeliverPolicy: js.DeliverAllPolicy, ReplayPolicy: js.ReplayInstantPolicy, AckPolicy: js.AckExplicitPolicy, AckWait: time.Duration(consumer.AckWaitMS) * time.Millisecond, MaxDeliver: -1, BackOff: toDurations(consumer.BackoffMS), MaxAckPending: consumer.MaxAckPending, MaxWaiting: consumer.MaxWaiting, MaxRequestBatch: consumer.MaxRequestBatch, MaxRequestExpires: time.Duration(consumer.MaxRequestExpiresMS) * time.Millisecond, MaxRequestMaxBytes: consumer.MaxRequestMaxBytes}
		if err := compareConsumer(actual, consumer); err != nil {
			t.Fatalf("%s: %v", consumer.ConsumerKey, err)
		}
		actual.AckPolicy = js.AckNonePolicy
		if err := compareConsumer(actual, consumer); err == nil {
			t.Fatalf("%s accepted non-explicit ack", consumer.ConsumerKey)
		}
	}
}
func toDurations(values []int64) []time.Duration {
	out := make([]time.Duration, len(values))
	for i, v := range values {
		out[i] = time.Duration(v) * time.Millisecond
	}
	return out
}
func TestConnectionOptionsAccountIdentityIsChecked(t *testing.T) {
	account, err := NewAccountIdentity("account-a")
	if err != nil {
		t.Fatal(err)
	}
	options, err := NewConnectionOptions("nats://example.invalid", account)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Open(context.Background(), options, "account-b"); err == nil {
		t.Fatal("mismatched account must fail before dialing")
	}
}

func TestDiagnosticCodeDoesNotExposeConnectionDetails(t *testing.T) {
	for err, want := range map[error]string{
		errNATSConnect:   "CONNECT",
		errNATSFlush:     "FLUSH",
		errJetStreamInit: "JETSTREAM",
		errors.New("nats://token@internal.example:4222"): "UNKNOWN",
	} {
		if got := DiagnosticCode(err); got != want {
			t.Fatalf("diagnostic code=%q want=%q", got, want)
		}
	}
}

func TestOpenCreatesDeadlineForBackgroundContext(t *testing.T) {
	ctx, cancel := newOpenContext(context.Background())
	defer cancel()
	deadline, ok := ctx.Deadline()
	if !ok || time.Until(deadline) <= 0 || time.Until(deadline) > natsOpenTimeout {
		t.Fatal("NATS startup preflight must use a bounded deadline")
	}
}

func TestClientCloseIsNilSafeAndConcurrentIdempotent(t *testing.T) {
	var client Client
	var group sync.WaitGroup
	for range 16 {
		group.Add(1)
		go func() { defer group.Done(); client.Close() }()
	}
	group.Wait()
}

func TestPublishRejectsPayloadBeforeBrokerAndPreservesDLQLimit(t *testing.T) {
	var client Client
	pointID := "11111111-1111-4111-8111-111111111111"
	for _, test := range []struct {
		subject string
		limit   int
	}{
		{"data.computed." + pointID, transportlimits.MaxBodyBytes},
		{"dlq.writer-derived", transportlimits.MaxDLQPayloadBytes},
	} {
		if err := client.Publish(t.Context(), PublishMessage{Subject: test.subject, DedupeKey: "stable", Payload: make([]byte, test.limit+1)}); !errors.Is(err, ErrPayloadTooLarge) {
			t.Fatalf("%s over boundary error=%v", test.subject, err)
		}
	}
	if err := client.Publish(t.Context(), PublishMessage{Subject: "dlq.writer?x", DedupeKey: "stable"}); !errors.Is(err, ErrInvalidSubject) {
		t.Fatalf("masquerading subject error=%v", err)
	}
}

func TestExactDurableNamesRejectsExtrasAndDuplicates(t *testing.T) {
	expected := []string{"writer-raw-v1", "alarm-raw-v1", "compute-raw-v1"}
	if !exactDurableNames([]string{"compute-raw-v1", "writer-raw-v1", "alarm-raw-v1"}, expected) {
		t.Fatal("same durable set in another order must be accepted")
	}
	for _, actual := range [][]string{
		{"writer-raw-v1", "alarm-raw-v1"},
		{"writer-raw-v1", "alarm-raw-v1", "compute-raw-v1", "query-raw-v1"},
		{"writer-raw-v1", "alarm-raw-v1", "writer-raw-v1"},
	} {
		if exactDurableNames(actual, expected) {
			t.Fatalf("unexpected/duplicate durable set accepted: %#v", actual)
		}
	}
}

func TestExpectedDurableNamesAreScopedToDataStream(t *testing.T) {
	config := model.EngineConfig{JetStream: model.JetStream{DataRawStream: "DATA_RAW", DataDerivedStream: "DATA_DERIVED", Consumers: []model.Consumer{
		{Stream: "DATA_RAW", DurableName: "writer-raw-v1"},
		{Stream: "DATA_DERIVED", DurableName: "writer-derived-v1"},
	}}}
	raw, err := expectedDurableNames(config, "DATA_RAW")
	if err != nil || !exactDurableNames(raw, []string{"writer-raw-v1"}) {
		t.Fatalf("raw expected durable names=%#v err=%v", raw, err)
	}
	if _, err := expectedDurableNames(config, "EVENT"); err == nil {
		t.Fatal("non-data stream must not get an expected durable set")
	}
	config.JetStream.Consumers = append(config.JetStream.Consumers, model.Consumer{Stream: "DATA_RAW", DurableName: "writer-raw-v1"})
	if _, err := expectedDurableNames(config, "DATA_RAW"); err == nil {
		t.Fatal("duplicate configured durable must fail before server comparison")
	}
}

func TestConsumerTopologyRejectsPushAndFrozenAttributeDrift(t *testing.T) {
	expected := model.Consumer{DurableName: "d", FilterSubject: "data.raw.>", AckWaitMS: 1000, BackoffMS: []int64{1000}, MaxAckPending: 32, MaxWaiting: 32, MaxRequestBatch: 32, MaxRequestExpiresMS: 5000, MaxRequestMaxBytes: 1 << 20}
	good := js.ConsumerConfig{Durable: "d", FilterSubject: expected.FilterSubject, DeliverPolicy: js.DeliverAllPolicy, ReplayPolicy: js.ReplayInstantPolicy, AckPolicy: js.AckExplicitPolicy, AckWait: time.Second, MaxDeliver: -1, BackOff: []time.Duration{time.Second}, MaxAckPending: 32, MaxWaiting: 32, MaxRequestBatch: 32, MaxRequestExpires: 5 * time.Second, MaxRequestMaxBytes: 1 << 20}
	if err := compareConsumer(good, expected); err != nil {
		t.Fatal(err)
	}
	good.DeliverSubject = "push.d"
	if err := compareConsumer(good, expected); err == nil {
		t.Fatal("push consumer must be rejected")
	}
	good.DeliverSubject = ""
	good.HeadersOnly = true
	if err := compareConsumer(good, expected); err == nil {
		t.Fatal("headers-only consumer must be rejected")
	}
	good.HeadersOnly = false
	good.FilterSubjects = []string{"data.raw.a"}
	if err := compareConsumer(good, expected); err == nil {
		t.Fatal("plural filter must be rejected")
	}
	good.FilterSubjects = nil
	good.MaxWaiting++
	if err := compareConsumer(good, expected); err == nil {
		t.Fatal("configured pull limit drift must be rejected")
	}
	good.MaxWaiting--
	good.Metadata = map[string]string{"operator": "override"}
	if err := compareConsumer(good, expected); err == nil {
		t.Fatal("operator metadata must be rejected")
	}
}

func TestDLQSubjectCoverageUsesNATSWildcards(t *testing.T) {
	if !coveredByAny([]string{"dlq.>"}, "dlq.writer") || !coveredByAny([]string{"dlq.*"}, "dlq.writer") {
		t.Fatal("valid wildcard coverage rejected")
	}
	if coveredByAny([]string{"dlq.writer"}, "dlq.compute") || coveredByAny([]string{"other.>"}, "dlq.writer") {
		t.Fatal("non-covering stream subject accepted")
	}
}

func TestStreamSubjectSetsAreExactAndNoExtraSubjectsPass(t *testing.T) {
	if !exactSubjects([]string{"data.raw.>"}, []string{"data.raw.>"}) {
		t.Fatal("exact stream subject rejected")
	}
	if exactSubjects([]string{"data.raw.>", "other.>"}, []string{"data.raw.>"}) || exactSubjects([]string{"data.raw.*"}, []string{"data.raw.>"}) {
		t.Fatal("extra or weaker stream subjects accepted")
	}
	if exactSubjects([]string{"dlq.writer", "dlq.writer"}, []string{"dlq.writer"}) {
		t.Fatal("duplicate configured subjects accepted")
	}
}
