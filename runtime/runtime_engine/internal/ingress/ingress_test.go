package ingress

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/indu-forge/runtime-engine/internal/loader"
	"github.com/indu-forge/runtime-engine/internal/model"
)

type fakeStore struct {
	processed    int
	permanent    []PermanentFailure
	result       ProcessResult
	processErr   error
	permanentErr error
	delay        time.Duration
}
type fakeHealth struct{ codes []string }

func (h *fakeHealth) ReportIngressFailure(code string) { h.codes = append(h.codes, code) }

func (s *fakeStore) Process(ctx context.Context, message ValidatedMessage) (ProcessResult, error) {
	s.processed++
	if s.delay > 0 {
		time.Sleep(s.delay)
	}
	if s.processErr != nil {
		return 0, s.processErr
	}
	return s.result, nil
}

func (s *fakeStore) ProcessPermanentFailure(_ context.Context, failure PermanentFailure) (PermanentResult, error) {
	s.permanent = append(s.permanent, failure)
	return PermanentStored, s.permanentErr
}

type fakeMessage struct {
	subject            string
	body               []byte
	position           int64
	deliveries         int
	occurred           time.Time
	ack, nak, progress int
}

func (m *fakeMessage) Subject() string                                   { return m.subject }
func (m *fakeMessage) Body() []byte                                      { return m.body }
func (m *fakeMessage) StreamPosition() int64                             { return m.position }
func (m *fakeMessage) DeliveryCount() int                                { return m.deliveries }
func (m *fakeMessage) OccurredAt() time.Time                             { return m.occurred }
func (m *fakeMessage) Ack(context.Context) error                         { m.ack++; return nil }
func (m *fakeMessage) InProgress(context.Context) error                  { m.progress++; return nil }
func (m *fakeMessage) NakWithDelay(context.Context, time.Duration) error { m.nak++; return nil }

func TestRunnerValidatesRawAndAcksAfterStore(t *testing.T) {
	runner, body := testRunner(t)
	message := &fakeMessage{subject: "data.raw.22222222-2222-4222-8222-222222222222", body: body, position: 8, deliveries: 1, occurred: time.Date(2026, 8, 30, 1, 2, 3, 0, time.UTC)}
	if err := runner.Handle(context.Background(), message); err != nil {
		t.Fatal(err)
	}
	if message.ack != 1 || message.nak != 0 {
		t.Fatalf("ack/nak=%d/%d", message.ack, message.nak)
	}
}
func TestRunnerDLQsPermanentFailuresAndMaxDeliver(t *testing.T) {
	runner, _ := testRunner(t)
	store := runner.processor.(*fakeStore)
	message := &fakeMessage{subject: "data.raw.x", body: []byte(`{"schemaVersion":"wat"}`), position: 9, deliveries: 1, occurred: time.Date(2026, 8, 30, 1, 2, 3, 0, time.UTC)}
	if err := runner.Handle(context.Background(), message); err != nil {
		t.Fatal(err)
	}
	if message.ack != 1 || len(store.permanent) != 1 || store.permanent[0].EventID != nil {
		t.Fatalf("permanent=%+v ack=%d", store.permanent, message.ack)
	}
	if len(store.permanent[0].BodySHA256) != 71 || store.permanent[0].BodySHA256[:7] != "sha256:" {
		t.Fatal("DLQ body digest must use contract sha256: prefix")
	}
	if err := loader.ValidateDLQEvent(store.permanent[0].DLQPayload); err != nil {
		t.Fatalf("DLQ schema: %v", err)
	}
	runner, _ = testRunner(t)
	store = runner.processor.(*fakeStore)
	store.processErr = os.ErrDeadlineExceeded
	body := validBody(t)
	message = &fakeMessage{subject: "data.raw.22222222-2222-4222-8222-222222222222", body: body, position: 10, deliveries: 5, occurred: time.Now()}
	if err := runner.Handle(context.Background(), message); err != nil {
		t.Fatal(err)
	}
	if len(store.permanent) != 1 || store.permanent[0].ReasonCode != "max-deliver" || message.ack != 1 {
		t.Fatal("max deliver must durable-DLQ then ack")
	}
	runner, _ = testRunner(t)
	message = &fakeMessage{subject: "data.raw.22222222-2222-4222-8222-222222222222", body: body, position: 11, deliveries: 5, occurred: time.Now()}
	if err := runner.Handle(context.Background(), message); err != nil || message.ack != 1 || len(runner.processor.(*fakeStore).permanent) != 0 {
		t.Fatal("final delivery must still attempt successful business transaction")
	}
}
func TestRunnerBusinessThresholdAndDLQRecovery(t *testing.T) {
	runner, body := testRunner(t)
	runner.consumer.MaxDeliver = 2
	store := runner.processor.(*fakeStore)
	store.processErr = os.ErrDeadlineExceeded
	// The threshold delivery performs the final business attempt, then writes DLQ.
	last := &fakeMessage{subject: "data.raw.22222222-2222-4222-8222-222222222222", body: body, position: 2, deliveries: 2, occurred: time.Now()}
	if err := runner.Handle(context.Background(), last); err != nil || store.processed != 1 || len(store.permanent) != 1 || last.ack != 1 {
		t.Fatalf("threshold disposition processed=%d permanent=%d ack=%d err=%v", store.processed, len(store.permanent), last.ack, err)
	}
	// A retry after threshold must not execute the business handler again.
	store.permanentErr = os.ErrDeadlineExceeded
	retry := &fakeMessage{subject: last.subject, body: body, position: 3, deliveries: 3, occurred: time.Now()}
	if err := runner.Handle(context.Background(), retry); err != nil || store.processed != 1 || retry.nak != 1 {
		t.Fatalf("post-threshold must retry DLQ only: processed=%d nak=%d err=%v", store.processed, retry.nak, err)
	}
	store.permanentErr = nil
	if err := runner.Handle(context.Background(), retry); err != nil || store.processed != 1 || retry.ack != 1 {
		t.Fatalf("DLQ recovery must Ack without handler: processed=%d ack=%d err=%v", store.processed, retry.ack, err)
	}
}
func TestRunnerPoisonSubjectStopsAndReportsHealth(t *testing.T) {
	runner, body := testRunner(t)
	health := &fakeHealth{}
	runner.health = health
	message := &fakeMessage{subject: string(make([]byte, MaxTransportSubjectBytes+1)), body: body, deliveries: 1, occurred: time.Now()}
	if err := runner.Handle(context.Background(), message); !errors.Is(err, ErrFatalPoison) || message.ack != 0 || message.nak != 0 || len(health.codes) != 1 {
		t.Fatalf("poison must stop without disposition: err=%v ack=%d nak=%d health=%v", err, message.ack, message.nak, health.codes)
	}
}

func TestMalformed513ByteSubjectIsPersistedToDLQAndAcked(t *testing.T) {
	runner, _ := testRunner(t)
	store := runner.processor.(*fakeStore)
	message := &fakeMessage{subject: strings.Repeat("x", 513), body: []byte(`{"schemaVersion":"bad"}`), deliveries: 1, occurred: time.Now()}
	if err := runner.Handle(context.Background(), message); err != nil || message.ack != 1 || message.nak != 0 || len(store.permanent) != 1 {
		t.Fatalf("513 byte malformed event must DLQ/Ack: err=%v ack=%d nak=%d permanent=%d", err, message.ack, message.nak, len(store.permanent))
	}
	if len(store.permanent[0].OriginalSubject) != 513 {
		t.Fatal("DLQ original subject truncated")
	}
}

func TestDLQPayloadFrozenUpperBound(t *testing.T) {
	ids := string(make([]byte, 128))
	for i := range ids {
		ids = ids[:i] + "a" + ids[i+1:]
	}
	payload, err := encodeDLQPayload(strings.Repeat("d", 64), ids, ids, ids, strings.Repeat("&", MaxTransportSubjectBytes), ptr(strings.Repeat("e", 64)), "permanent-validation", math.MaxInt, "sha256:"+strings.Repeat("f", 64), make([]byte, MaxBodyBytes), time.Date(9999, 12, 31, 23, 59, 59, 999999999, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(payload) != 1423548 {
		t.Fatalf("frozen maximal DLQ bytes=%d", len(payload))
	}
	if len(payload) > MaxDLQPayloadBytes {
		t.Fatalf("maximal payload exceeds frozen limit: %d", len(payload))
	}
}
func ptr(v string) *string { return &v }
func TestRunnerRejectsProducerFenceAndNaksFailureStore(t *testing.T) {
	runner, body := testRunner(t)
	var value map[string]any
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatal(err)
	}
	value["ownerId"] = "stale"
	body, _ = json.Marshal(value)
	message := &fakeMessage{subject: "data.raw.22222222-2222-4222-8222-222222222222", body: body, position: 1, deliveries: 1, occurred: time.Now()}
	if err := runner.Handle(context.Background(), message); err != nil {
		t.Fatal(err)
	}
	if message.ack != 1 {
		t.Fatal("permanent producer fence must ack only after fake durable failure")
	}
	runner, body = testRunner(t)
	runner.processor.(*fakeStore).processErr = os.ErrDeadlineExceeded
	message = &fakeMessage{subject: "data.raw.22222222-2222-4222-8222-222222222222", body: body, position: 1, deliveries: 1, occurred: time.Now()}
	if err := runner.Handle(context.Background(), message); err != nil {
		t.Fatal(err)
	}
	if message.nak != 1 || message.ack != 0 {
		t.Fatal("temporary store failure must Nak")
	}
}

func TestRunnerExtendsAckForLongTransaction(t *testing.T) {
	runner, body := testRunner(t)
	runner.consumer.AckWaitMS = 2
	runner.processor.(*fakeStore).delay = 8 * time.Millisecond
	message := &fakeMessage{subject: "data.raw.22222222-2222-4222-8222-222222222222", body: body, position: 1, deliveries: 1, occurred: time.Now()}
	if err := runner.Handle(context.Background(), message); err != nil {
		t.Fatal(err)
	}
	if message.progress == 0 {
		t.Fatal("long transaction must periodically send InProgress")
	}
}

func testRunner(t *testing.T) (*Runner, []byte) {
	t.Helper()
	loaded := testLoaded(t)
	consumer := loaded.Config.JetStream.Consumers[0]
	store := &fakeStore{result: Processed}
	runner, err := NewRunner(loaded, consumer, ConsumerToken{OwnerID: "runtime-engine-writer-0", Epoch: 7}, store)
	if err != nil {
		t.Fatal(err)
	}
	return runner, validBody(t)
}
func testLoaded(t *testing.T) *loader.Loaded {
	t.Helper()
	var config model.EngineConfig
	var artifact model.ProjectArtifact
	var collector model.CollectorArtifact
	readFixture(t, "runtime-engine-config.valid.json", &config)
	readFixture(t, "runtime-project-artifact.valid.json", &artifact)
	readFixture(t, "collector-runtime-artifact.valid.json", &collector)
	return &loader.Loaded{Config: config, Artifact: artifact, CollectorArtifacts: map[string]model.CollectorArtifact{"collector-line1-a": collector}}
}
func validBody(t *testing.T) []byte {
	t.Helper()
	return readFixtureBytes(t, "point-event.valid.raw.json")
}
func readFixture(t *testing.T, name string, target any) {
	t.Helper()
	if err := json.Unmarshal(readFixtureBytes(t, name), target); err != nil {
		t.Fatal(err)
	}
}
func readFixtureBytes(t *testing.T, name string) []byte {
	t.Helper()
	value, err := os.ReadFile(filepath.Join("../../../../contracts/runtime/fixtures", name))
	if err != nil {
		t.Fatal(err)
	}
	return value
}
