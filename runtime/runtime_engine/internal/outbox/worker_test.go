package outbox

import (
	"context"
	"errors"
	"testing"
	"time"

	transport "github.com/indu-forge/runtime-engine/internal/transport/jetstream"
)

type fakeStore struct {
	records   []Record
	published []int64
	retries   []string
	claimErr  error
	markErr   error
	retryErr  error
}

func (s *fakeStore) ClaimOutbox(context.Context, string, string, int, time.Duration) ([]Record, error) {
	if s.claimErr != nil {
		return nil, s.claimErr
	}
	r := s.records
	s.records = nil
	return r, nil
}
func (s *fakeStore) MarkPublished(_ context.Context, id int64, _ string) error {
	s.published = append(s.published, id)
	return s.markErr
}
func (s *fakeStore) RetryOutbox(_ context.Context, _ int64, _ string, _ time.Time, code RetryCode) error {
	s.retries = append(s.retries, string(code))
	return s.retryErr
}

func TestFlushOnceClassifiesAdapterFailuresWithoutChangingErrorsIs(t *testing.T) {
	payload := []byte("x")
	tests := []struct {
		name, want string
		store      *fakeStore
		publisher  *fakePublisher
		is         error
	}{
		{name: "claim", want: "OUTBOX_CLAIM", store: &fakeStore{claimErr: errors.New("dsn=secret")}, publisher: &fakePublisher{}, is: ErrClaim},
		{name: "mark", want: "OUTBOX_MARK", store: &fakeStore{records: []Record{{ID: 1, DedupeKey: "a", Subject: "x", Payload: payload, PayloadSHA256: transport.PayloadSHA256(payload), LeaseToken: "l"}}, markErr: errors.New("password=secret")}, publisher: &fakePublisher{}, is: ErrMark},
		{name: "lease", want: "OUTBOX_LEASE", store: &fakeStore{records: []Record{{ID: 1, DedupeKey: "a", Subject: "x", Payload: []byte("bad"), PayloadSHA256: "bad", LeaseToken: "l"}}, retryErr: errors.New("token=secret")}, publisher: &fakePublisher{}, is: ErrLease},
		{name: "publish", want: "OUTBOX_PUBLISH", store: &fakeStore{records: []Record{{ID: 1, DedupeKey: "a", Subject: "x", Payload: payload, PayloadSHA256: transport.PayloadSHA256(payload), LeaseToken: "l"}}}, publisher: &fakePublisher{err: transport.ErrPayloadTooLarge}, is: ErrPublish},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := newWorker(t, test.store, test.publisher).FlushOnce(context.Background())
			if !errors.Is(err, test.is) || DiagnosticCode(err) != test.want {
				t.Fatalf("stage=%q err=%v", DiagnosticCode(err), err)
			}
		})
	}
}

type fakePublisher struct {
	messages []transport.PublishMessage
	err      error
}

func (p *fakePublisher) Publish(_ context.Context, m transport.PublishMessage) error {
	p.messages = append(p.messages, m)
	return p.err
}

func TestFlushOncePublishesOriginalBytesAndStableMsgID(t *testing.T) {
	payload := []byte(`{"x":1}`)
	store := &fakeStore{records: []Record{{ID: 1, DeploymentID: "d", DedupeKey: "stable", Subject: "x.y", Payload: payload, PayloadSHA256: transport.PayloadSHA256(payload), LeaseToken: "lease", AttemptCount: 1}}}
	publisher := &fakePublisher{}
	worker := newWorker(t, store, publisher)
	if err := worker.FlushOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(publisher.messages) != 1 || string(publisher.messages[0].Payload) != string(payload) || publisher.messages[0].DedupeKey != "stable" || len(store.published) != 1 {
		t.Fatal("must publish immutable payload with stable Msg-Id then CAS mark")
	}
}
func TestFlushOnceRetriesDigestAndPubAckFailures(t *testing.T) {
	store := &fakeStore{records: []Record{{ID: 1, DeploymentID: "d", DedupeKey: "a", Subject: "x", Payload: []byte("x"), PayloadSHA256: "bad", LeaseToken: "l", AttemptCount: 1}}}
	worker := newWorker(t, store, &fakePublisher{})
	if err := worker.FlushOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(store.retries) != 1 || store.retries[0] != "publish-error" {
		t.Fatal("digest mismatch must never publish")
	}
	payload := []byte("x")
	store = &fakeStore{records: []Record{{ID: 2, DeploymentID: "d", DedupeKey: "b", Subject: "x", Payload: payload, PayloadSHA256: transport.PayloadSHA256(payload), LeaseToken: "l", AttemptCount: 1}}}
	worker = newWorker(t, store, &fakePublisher{err: errors.New("no puback")})
	if err := worker.FlushOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(store.retries) != 1 || store.retries[0] != "publish-error" {
		t.Fatal("puback failure must retry with safe code")
	}
}
func TestPublishBudgetExhaustionIsFatal(t *testing.T) {
	payload := []byte("x")
	store := &fakeStore{records: []Record{{ID: 1, DeploymentID: "d", DedupeKey: "x", Subject: "x", Payload: payload, PayloadSHA256: transport.PayloadSHA256(payload), LeaseToken: "l", AttemptCount: 99}}}
	worker, err := NewWorker(store, &fakePublisher{err: errors.New("down")}, Options{DeploymentID: "d", LeaseOwner: "worker", BatchSize: 1, LeaseFor: time.Second, PollInterval: time.Second, BaseRetry: time.Second, MaxRetry: time.Second, DrainTimeout: time.Second, MaxConsecutiveFailures: 2})
	if err != nil {
		t.Fatal(err)
	}
	if err := worker.FlushOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	store.records = []Record{{ID: 1, DeploymentID: "d", DedupeKey: "x", Subject: "x", Payload: payload, PayloadSHA256: transport.PayloadSHA256(payload), LeaseToken: "l"}}
	if err := worker.FlushOnce(context.Background()); !errors.Is(err, ErrRetryBudgetExhausted) {
		t.Fatalf("retry budget=%v", err)
	}
}

func TestFlushOnceDoesNotRetryDeterministicallyUnpublishableRecord(t *testing.T) {
	payload := []byte("x")
	store := &fakeStore{records: []Record{{ID: 9, DeploymentID: "d", DedupeKey: "bad", Subject: "data.raw.11111111-1111-4111-8111-111111111111", Payload: payload, PayloadSHA256: transport.PayloadSHA256(payload), LeaseToken: "lease"}}}
	degraded := 0
	worker, err := NewWorker(store, &fakePublisher{err: transport.ErrPayloadTooLarge}, Options{DeploymentID: "d", LeaseOwner: "worker", BatchSize: 1, LeaseFor: time.Second, PollInterval: time.Second, BaseRetry: time.Second, MaxRetry: time.Second, DrainTimeout: time.Second, OnIntegrityFault: func() { degraded++ }})
	if err != nil {
		t.Fatal(err)
	}
	if err := worker.FlushOnce(context.Background()); !errors.Is(err, ErrUnpublishableRecord) {
		t.Fatalf("FlushOnce error=%v", err)
	}
	if len(store.retries) != 0 || degraded != 1 {
		t.Fatalf("unpublishable record must not be retried: retries=%v degraded=%d", store.retries, degraded)
	}
}
func newWorker(t *testing.T, s Store, p Publisher) *Worker {
	t.Helper()
	w, err := NewWorker(s, p, Options{DeploymentID: "d", LeaseOwner: "worker", BatchSize: 8, LeaseFor: time.Second, PollInterval: time.Millisecond, BaseRetry: time.Millisecond, MaxRetry: time.Second, DrainTimeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	return w
}
