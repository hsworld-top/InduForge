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
	"github.com/indu-forge/runtime-engine/internal/store/postgres"
)

type fakeStore struct {
	processed    int
	positions    []int64
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
	s.positions = append(s.positions, message.StreamPosition)
	if s.delay > 0 {
		time.Sleep(s.delay)
	}
	if s.processErr != nil {
		return 0, s.processErr
	}
	return s.result, nil
}

func TestRunnerTerminatesMaxDeliverAndContinuesWithNextEvent(t *testing.T) {
	runner, body := testRunner(t)
	store := runner.processor.(*fakeStore)
	runner.consumer.MaxDeliver = 2
	store.processErr = ErrRetryableBusiness
	poison := &fakeMessage{subject: "data.raw.22222222-2222-4222-8222-222222222222", body: body, position: 40, deliveries: 2, occurred: time.Now()}
	if err := runner.Handle(context.Background(), poison); err != nil {
		t.Fatal(err)
	}
	if poison.term != 1 || poison.ack != 0 || len(store.permanent) != 1 || store.permanent[0].StreamPosition != 40 {
		t.Fatalf("poison isolation term/ack=%d/%d failure=%+v", poison.term, poison.ack, store.permanent)
	}
	store.processErr = nil
	next := &fakeMessage{subject: poison.subject, body: body, position: 41, deliveries: 1, occurred: time.Now()}
	if err := runner.Handle(context.Background(), next); err != nil {
		t.Fatal(err)
	}
	if next.ack != 1 || next.term != 0 || store.processed != 2 || len(store.positions) != 2 || store.positions[1] != 41 || len(store.permanent) != 1 {
		t.Fatalf("next event did not advance: ack/term=%d/%d processed=%d positions=%v failures=%d", next.ack, next.term, store.processed, store.positions, len(store.permanent))
	}
}

func (s *fakeStore) ProcessPermanentFailure(_ context.Context, failure PermanentFailure) (PermanentResult, error) {
	s.permanent = append(s.permanent, failure)
	return PermanentStored, s.permanentErr
}

type fakeMessage struct {
	subject                  string
	body                     []byte
	position                 int64
	deliveries               int
	occurred                 time.Time
	ack, term, nak, progress int
	ackErr                   error
}

func (m *fakeMessage) Subject() string                                   { return m.subject }
func (m *fakeMessage) Body() []byte                                      { return m.body }
func (m *fakeMessage) StreamPosition() int64                             { return m.position }
func (m *fakeMessage) DeliveryCount() int                                { return m.deliveries }
func (m *fakeMessage) OccurredAt() time.Time                             { return m.occurred }
func (m *fakeMessage) Ack(context.Context) error                         { m.ack++; return m.ackErr }
func (m *fakeMessage) Terminate(context.Context) error                   { m.term++; return m.ackErr }
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

func TestRunnerAcknowledgementFailureHasSafeStage(t *testing.T) {
	runner, body := testRunner(t)
	message := &fakeMessage{subject: "data.raw.22222222-2222-4222-8222-222222222222", body: body, position: 8, deliveries: 1, occurred: time.Now(), ackErr: errors.New("token=secret-value")}
	err := runner.Handle(context.Background(), message)
	if !errors.Is(err, ErrAck) || DiagnosticCode(err) != "INGRESS_ACK" {
		t.Fatalf("ack stage=%q err=%v", DiagnosticCode(err), err)
	}
}
func TestRunnerDLQsPermanentFailuresAndMaxDeliver(t *testing.T) {
	runner, _ := testRunner(t)
	store := runner.processor.(*fakeStore)
	message := &fakeMessage{subject: "data.raw.x", body: []byte(`{"schemaVersion":"wat"}`), position: 9, deliveries: 1, occurred: time.Date(2026, 8, 30, 1, 2, 3, 0, time.UTC)}
	if err := runner.Handle(context.Background(), message); err != nil {
		t.Fatal(err)
	}
	if message.term != 1 || message.ack != 0 || len(store.permanent) != 1 || store.permanent[0].EventID != nil {
		t.Fatalf("permanent=%+v term/ack=%d/%d", store.permanent, message.term, message.ack)
	}
	if len(store.permanent[0].BodySHA256) != 71 || store.permanent[0].BodySHA256[:7] != "sha256:" {
		t.Fatal("DLQ body digest must use contract sha256: prefix")
	}
	if store.permanent[0].RequireProducerFence || store.permanent[0].ProducerKey != "" || store.permanent[0].ProducerToken.OwnerID != "" || store.permanent[0].ProducerToken.Epoch != 0 {
		t.Fatal("未通过校验的坏消息不得伪造 producer fence")
	}
	if err := loader.ValidateDLQEvent(store.permanent[0].DLQPayload); err != nil {
		t.Fatalf("DLQ schema: %v", err)
	}
	runner, _ = testRunner(t)
	store = runner.processor.(*fakeStore)
	store.processErr = ErrRetryableBusiness
	body := validBody(t)
	message = &fakeMessage{subject: "data.raw.22222222-2222-4222-8222-222222222222", body: body, position: 10, deliveries: 5, occurred: time.Now()}
	if err := runner.Handle(context.Background(), message); err != nil {
		t.Fatal(err)
	}
	if len(store.permanent) != 1 || store.permanent[0].ReasonCode != "max-deliver" || message.term != 1 || message.ack != 0 {
		t.Fatal("max deliver must durable-DLQ then terminate")
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
	store.processErr = ErrRetryableBusiness
	// The threshold delivery performs the final business attempt, then writes DLQ.
	last := &fakeMessage{subject: "data.raw.22222222-2222-4222-8222-222222222222", body: body, position: 2, deliveries: 2, occurred: time.Now()}
	if err := runner.Handle(context.Background(), last); err != nil || store.processed != 1 || len(store.permanent) != 1 || last.term != 1 {
		t.Fatalf("threshold disposition processed=%d permanent=%d term=%d err=%v", store.processed, len(store.permanent), last.term, err)
	}
	// A retry after threshold must not execute the business handler again.
	store.permanentErr = ErrRetryableBusiness
	retry := &fakeMessage{subject: last.subject, body: body, position: 3, deliveries: 3, occurred: time.Now()}
	if err := runner.Handle(context.Background(), retry); err != nil || store.processed != 1 || retry.nak != 1 {
		t.Fatalf("post-threshold must retry DLQ only: processed=%d nak=%d err=%v", store.processed, retry.nak, err)
	}
	store.permanentErr = nil
	if err := runner.Handle(context.Background(), retry); err != nil || store.processed != 1 || retry.term != 1 {
		t.Fatalf("DLQ recovery must terminate without handler: processed=%d term=%d err=%v", store.processed, retry.term, err)
	}
}

func TestRunnerBusinessFailureReportsImmediatelyAndCarriesProducerFenceToDLQ(t *testing.T) {
	runner, body := testRunner(t)
	runner.consumer.MaxDeliver = 2
	store := runner.processor.(*fakeStore)
	health := &fakeHealth{}
	runner.health = health
	store.processErr = ErrRetryableBusiness
	first := &fakeMessage{subject: "data.raw.22222222-2222-4222-8222-222222222222", body: body, position: 1, deliveries: 1, occurred: time.Now()}
	if err := runner.Handle(context.Background(), first); err != nil || first.nak != 1 || len(health.codes) != 1 || health.codes[0] != "handler-failure" {
		t.Fatalf("首次业务失败必须立即退化并 NAK: err=%v nak=%d health=%v", err, first.nak, health.codes)
	}
	last := &fakeMessage{subject: first.subject, body: body, position: 2, deliveries: 2, occurred: time.Now()}
	if err := runner.Handle(context.Background(), last); err != nil || last.term != 1 || len(store.permanent) != 1 {
		t.Fatalf("阈值业务失败必须写入 DLQ 后终止确认: err=%v term=%d permanent=%d", err, last.term, len(store.permanent))
	}
	failure := store.permanent[0]
	if failure.ReasonCode != "max-deliver" || !failure.RequireProducerFence || failure.ProducerKey == "" || failure.ProducerToken.OwnerID == "" || failure.ProducerToken.Epoch < 1 {
		t.Fatalf("已验证业务失败必须携带 producer fence: %#v", failure)
	}
	if len(health.codes) != 2 || health.codes[1] != "handler-failure" {
		t.Fatalf("阈值业务失败仍必须保持退化: %v", health.codes)
	}
}

func TestRunnerNonBusinessFailureDoesNotReportOrDispose(t *testing.T) {
	runner, body := testRunner(t)
	health := &fakeHealth{}
	runner.health = health
	runner.processor.(*fakeStore).processErr = postgres.ErrFenceStale
	message := &fakeMessage{subject: "data.raw.22222222-2222-4222-8222-222222222222", body: body, position: 1, deliveries: 1, occurred: time.Now()}
	if err := runner.Handle(context.Background(), message); !errors.Is(err, postgres.ErrFenceStale) || message.ack != 0 || message.nak != 0 || len(health.codes) != 0 {
		t.Fatalf("fence/PG/context 错误不可被标作业务退化或 disposition: err=%v ack=%d nak=%d health=%v", err, message.ack, message.nak, health.codes)
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
	if err := runner.Handle(context.Background(), message); err != nil || message.term != 1 || message.ack != 0 || message.nak != 0 || len(store.permanent) != 1 {
		t.Fatalf("513 byte malformed event must DLQ/terminate: err=%v term=%d nak=%d permanent=%d", err, message.term, message.nak, len(store.permanent))
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
	if message.term != 1 {
		t.Fatal("permanent producer fence must terminate only after fake durable failure")
	}
	runner, body = testRunner(t)
	runner.processor.(*fakeStore).processErr = ErrRetryableBusiness
	message = &fakeMessage{subject: "data.raw.22222222-2222-4222-8222-222222222222", body: body, position: 1, deliveries: 1, occurred: time.Now()}
	if err := runner.Handle(context.Background(), message); err != nil {
		t.Fatal(err)
	}
	if message.nak != 1 || message.ack != 0 {
		t.Fatal("temporary store failure must Nak")
	}
}

func TestRunnerOrdinaryStoreErrorIsFatalWithoutDisposition(t *testing.T) {
	runner, body := testRunner(t)
	runner.processor.(*fakeStore).processErr = errors.New("pg query failed")
	message := &fakeMessage{subject: "data.raw.22222222-2222-4222-8222-222222222222", body: body, position: 1, deliveries: 1, occurred: time.Now()}
	err := runner.Handle(context.Background(), message)
	if err == nil || message.ack != 0 || message.nak != 0 {
		t.Fatalf("ordinary store error must be fatal: err=%v ack=%d nak=%d", err, message.ack, message.nak)
	}
}

func TestRunnerStaleStoreFenceDoesNotDisposeMessage(t *testing.T) {
	runner, body := testRunner(t)
	runner.processor.(*fakeStore).processErr = postgres.ErrFenceStale
	message := &fakeMessage{subject: "data.raw.22222222-2222-4222-8222-222222222222", body: body, position: 1, deliveries: 1, occurred: time.Now()}
	err := runner.Handle(context.Background(), message)
	if !errors.Is(err, postgres.ErrFenceStale) || message.ack != 0 || message.nak != 0 {
		t.Fatalf("stale fence must be fatal without disposition: err=%v ack=%d nak=%d", err, message.ack, message.nak)
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
