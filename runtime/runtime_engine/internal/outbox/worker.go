// Package outbox 提供可组合的事务 outbox publisher；它不管理 NATS 或数据库连接生命周期。
package outbox

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"time"

	transport "github.com/indu-forge/runtime-engine/internal/transport/jetstream"
)

type Record struct {
	ID                               int64
	DeploymentID, DedupeKey, Subject string
	Headers, Payload                 []byte
	PayloadSHA256, LeaseToken        string
	AttemptCount                     int
}
type RetryCode string

const (
	RetryPublishError   RetryCode = "publish-error"
	RetryPublishTimeout RetryCode = "publish-timeout"
	RetryShutdown       RetryCode = "shutdown"
)

// Store 为窄接口，postgres 适配器负责将稳定 RetryCode 映射到状态库白名单。
type Store interface {
	ClaimOutbox(context.Context, string, string, int, time.Duration) ([]Record, error)
	MarkPublished(context.Context, int64, string) error
	RetryOutbox(context.Context, int64, string, time.Time, RetryCode) error
}
type Publisher interface {
	Publish(context.Context, transport.PublishMessage) error
}
type Options struct {
	DeploymentID, LeaseOwner                                  string
	BatchSize                                                 int
	LeaseFor, PollInterval, BaseRetry, MaxRetry, DrainTimeout time.Duration
	OnIntegrityFault                                          func()
	MaxConsecutiveFailures                                    int
}
type Worker struct {
	store     Store
	publisher Publisher
	options   Options
	failures  map[int64]int
}

func NewWorker(store Store, publisher Publisher, options Options) (*Worker, error) {
	if options.MaxConsecutiveFailures == 0 {
		options.MaxConsecutiveFailures = 5
	}
	if store == nil || publisher == nil || options.DeploymentID == "" || options.LeaseOwner == "" || options.BatchSize < 1 || options.LeaseFor <= 0 || options.PollInterval <= 0 || options.BaseRetry <= 0 || options.MaxRetry < options.BaseRetry || options.DrainTimeout <= 0 || options.MaxConsecutiveFailures < 1 {
		return nil, errors.New("outbox worker 配置非法")
	}
	return &Worker{store: store, publisher: publisher, options: options, failures: map[int64]int{}}, nil
}

// Run 在外层取消后不再开始新 Claim；已经开始的批次使用有界 detached context 排空。
func (w *Worker) Run(ctx context.Context) error {
	return w.RunWithDrain(ctx, ctx)
}

// RunWithDrain stops new claims when intake is cancelled while allowing the
// active claim/publish batch to finish under the bounded work context.
func (w *Worker) RunWithDrain(intake, work context.Context) error {
	ticker := time.NewTicker(w.options.PollInterval)
	defer ticker.Stop()
	for {
		if intake.Err() != nil {
			return nil
		}
		drain, cancel := context.WithTimeout(work, w.options.DrainTimeout)
		err := w.FlushOnce(drain)
		cancel()
		if err != nil && work.Err() == nil {
			return err
		}
		select {
		case <-intake.Done():
			return nil
		case <-ticker.C:
		}
	}
}
func (w *Worker) FlushOnce(ctx context.Context) error {
	records, err := w.store.ClaimOutbox(ctx, w.options.DeploymentID, w.options.LeaseOwner, w.options.BatchSize, w.options.LeaseFor)
	if err != nil {
		return err
	}
	for _, record := range records {
		if err := w.publishOne(ctx, record); err != nil {
			if errors.Is(err, ErrDeferred) {
				w.failures[record.ID]++
				if w.failures[record.ID] >= w.options.MaxConsecutiveFailures {
					return ErrRetryBudgetExhausted
				}
				continue
			}
			return err
		}
		delete(w.failures, record.ID)
	}
	return nil
}
func (w *Worker) publishOne(ctx context.Context, record Record) error {
	if transport.PayloadSHA256(record.Payload) != record.PayloadSHA256 {
		if w.options.OnIntegrityFault != nil {
			w.options.OnIntegrityFault()
		}
		if err := w.retry(ctx, record, RetryPublishError); err != nil {
			return err
		}
		return ErrDeferred
	}
	if err := w.publisher.Publish(ctx, transport.PublishMessage{Subject: record.Subject, Headers: record.Headers, Payload: record.Payload, DedupeKey: record.DedupeKey}); err != nil {
		if errors.Is(err, transport.ErrPayloadTooLarge) || errors.Is(err, transport.ErrInvalidSubject) {
			if w.options.OnIntegrityFault != nil {
				w.options.OnIntegrityFault()
			}
			return fmt.Errorf("%w: %w", ErrUnpublishableRecord, err)
		}
		if errors.Is(err, context.DeadlineExceeded) {
			if retryErr := w.retry(ctx, record, RetryPublishTimeout); retryErr != nil {
				return retryErr
			}
			return ErrDeferred
		}
		if retryErr := w.retry(ctx, record, RetryPublishError); retryErr != nil {
			return retryErr
		}
		return ErrDeferred
	}
	return w.store.MarkPublished(ctx, record.ID, record.LeaseToken)
}

var ErrDeferred = errors.New("outbox publish deferred")
var ErrRetryBudgetExhausted = errors.New("outbox publish retry budget exhausted")

// ErrUnpublishableRecord is terminal for this worker invocation. The record
// cannot be repaired by retrying the same immutable bytes against JetStream.
var ErrUnpublishableRecord = errors.New("outbox record cannot be published")

func (w *Worker) retry(ctx context.Context, record Record, code RetryCode) error {
	return w.store.RetryOutbox(ctx, record.ID, record.LeaseToken, time.Now().UTC().Add(w.retryDelay(record)), code)
}
func (w *Worker) retryDelay(record Record) time.Duration {
	attempt := record.AttemptCount
	if attempt < 1 {
		attempt = 1
	}
	d := w.options.BaseRetry
	for i := 1; i < attempt && d < w.options.MaxRetry; i++ {
		d *= 2
		if d > w.options.MaxRetry {
			d = w.options.MaxRetry
		}
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(record.DedupeKey))
	seed := uint64(h.Sum32()) ^ uint64(record.ID)
	jitter := time.Duration((seed+uint64(attempt)*37)%101) * d / 1000
	if d+jitter > w.options.MaxRetry {
		return w.options.MaxRetry
	}
	return d + jitter
}
