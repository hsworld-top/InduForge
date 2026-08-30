package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/indu-forge/runtime-engine/internal/eventid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestApplyPointCurrentConcurrentOrder(t *testing.T) {
	dsn := os.Getenv("RUNTIME_ENGINE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("RUNTIME_ENGINE_TEST_DATABASE_URL 未设置")
	}
	ctx := context.Background()
	store, _, cleanup := setupIntegrationStore(t, ctx, dsn)
	defer cleanup()
	if err := store.ApplySchema(ctx); err != nil {
		t.Fatal(err)
	}
	token := ConsumerRoleToken{OwnerID: "writer-c", Epoch: 1}
	if _, err := store.ActivateRole(ctx, "dep-c", "writer", token, 0); err != nil {
		t.Fatal(err)
	}
	const writes = 16
	base := time.Date(2026, 1, 4, 0, 0, 0, 0, time.UTC)
	var applied atomic.Int64
	var wg sync.WaitGroup
	errs := make(chan error, writes)
	for i := 0; i < writes; i++ {
		wg.Add(1)
		go func(sequence int) {
			defer wg.Done()
			eventID := fmt.Sprintf("%064x", sequence+1)
			message := Message{DeploymentID: "dep-c", AccountID: "account-c", ConsumerKey: "writer.concurrent", Role: "writer", Token: token, EventID: eventID, RawBody: []byte(eventID), Subject: "data.raw.p", CheckpointPosition: int64(sequence + 1), DeliveryCount: 1, OccurredAt: base}
			var result PointCurrentResult
			_, err := store.ProcessMessage(ctx, message, func(ctx context.Context, tx *BusinessTx) error {
				var applyErr error
				result, applyErr = tx.ApplyPointCurrent(ctx, PointCurrentWrite{DeploymentID: "dep-c", PointID: "11111111-1111-4111-8111-111111111111", OwnerID: token.OwnerID, EventID: eventID, Quality: "good", Epoch: 1, Sequence: int64(sequence), SourceTimestamp: base, ServerTimestamp: base, Value: []byte(fmt.Sprintf("%d", sequence))})
				return applyErr
			}, ProcessOptions{})
			if err != nil {
				errs <- err
				return
			}
			if result.Disposition == PointCurrentApplied {
				applied.Add(1)
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	var sequence, version int64
	if err := store.pool.QueryRow(ctx, `SELECT sequence,version FROM runtime_engine.point_current WHERE deployment_id='dep-c' AND point_id='11111111-1111-4111-8111-111111111111'::uuid`).Scan(&sequence, &version); err != nil {
		t.Fatal(err)
	}
	if sequence != writes-1 || version != applied.Load() {
		t.Fatalf("current sequence/version=%d/%d, applied=%d", sequence, version, applied.Load())
	}
}

func TestAlarmItemStateRevisionPolicy(t *testing.T) {
	dsn := os.Getenv("RUNTIME_ENGINE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("RUNTIME_ENGINE_TEST_DATABASE_URL 未设置")
	}
	ctx := context.Background()
	store, _, cleanup := setupIntegrationStore(t, ctx, dsn)
	defer cleanup()
	if err := store.ApplySchema(ctx); err != nil {
		t.Fatal(err)
	}
	token := ProducerToken{OwnerID: "alarm-owner", Epoch: 1}
	if _, err := store.ActivateProducer(ctx, "dep-alarm", "alarm", token, 0); err != nil {
		t.Fatal(err)
	}
	const alarmID = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	if err := store.RunProducerTransaction(ctx, "dep-alarm", "alarm", token, func(ctx context.Context, tx *BusinessTx) error {
		state, err := tx.AlarmItemState(ctx, alarmID, 1)
		if err != nil {
			return err
		}
		state.State = json.RawMessage(`{"activeConditionId":"bbbbbbb1-bbbb-4bbb-8bbb-bbbbbbbbbbbb"}`)
		return tx.SaveAlarmItemState(ctx, alarmID, state)
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.RunProducerTransaction(ctx, "dep-alarm", "alarm", token, func(ctx context.Context, tx *BusinessTx) error {
		_, err := tx.AlarmItemState(ctx, alarmID, 2)
		return err
	}); !errors.Is(err, ErrActiveAlarmRevision) {
		t.Fatalf("active revision change=%v", err)
	}
	if err := store.RunProducerTransaction(ctx, "dep-alarm", "alarm", token, func(ctx context.Context, tx *BusinessTx) error {
		state, err := tx.AlarmItemState(ctx, alarmID, 1)
		if err != nil {
			return err
		}
		state.State = json.RawMessage(`{"stateVersion":2,"transitionSequence":2,"inputs":{"11111111-1111-4111-8111-111111111111":{}}}`)
		return tx.SaveAlarmItemState(ctx, alarmID, state)
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.RunProducerTransaction(ctx, "dep-alarm", "alarm", token, func(ctx context.Context, tx *BusinessTx) error {
		state, err := tx.AlarmItemState(ctx, alarmID, 2)
		if err != nil || !state.RevisionReset || state.AlarmRevision != 2 || string(state.State) != `{"stateVersion":2,"transitionSequence":2}` {
			t.Fatalf("inactive revision reset=%#v %v", state, err)
		}
		return tx.SaveAlarmItemState(ctx, alarmID, state)
	}); err != nil {
		t.Fatal(err)
	}
}

func TestComputeScheduleStateRoundTripsNullableLastLocalKey(t *testing.T) {
	dsn := os.Getenv("RUNTIME_ENGINE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("RUNTIME_ENGINE_TEST_DATABASE_URL 未设置")
	}
	ctx := context.Background()
	store, _, cleanup := setupIntegrationStore(t, ctx, dsn)
	defer cleanup()
	if err := store.ApplySchema(ctx); err != nil {
		t.Fatal(err)
	}
	const computeID = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	token := ProducerToken{OwnerID: "compute-owner", Epoch: 1}
	if _, err := store.ActivateProducer(ctx, "dep-schedule", computeID, token, 0); err != nil {
		t.Fatal(err)
	}
	next := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	if err := store.RunProducerTransaction(ctx, "dep-schedule", computeID, token, func(ctx context.Context, tx *BusinessTx) error {
		state, err := tx.ComputeScheduleState(ctx, computeID, 1)
		if err != nil {
			return err
		}
		state.NextRunAt = &next
		return tx.SaveComputeScheduleState(ctx, computeID, state)
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.RunProducerTransaction(ctx, "dep-schedule", computeID, token, func(ctx context.Context, tx *BusinessTx) error {
		state, err := tx.ComputeScheduleState(ctx, computeID, 1)
		if err != nil {
			return err
		}
		if state.NextRunAt == nil || !state.NextRunAt.Equal(next) || state.LastRunAt != nil || state.LastLocalKey != "" {
			t.Fatalf("restart schedule state=%#v", state)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresV1Foundation(t *testing.T) {
	dsn := os.Getenv("RUNTIME_ENGINE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("RUNTIME_ENGINE_TEST_DATABASE_URL 未设置")
	}
	ctx := context.Background()
	store, testDSN, cleanup := setupIntegrationStore(t, ctx, dsn)
	defer cleanup()

	t.Run("baseline exact version", func(t *testing.T) {
		if err := store.VerifySchema(ctx); err == nil {
			t.Fatal("未初始化数据库不应通过版本校验")
		}
		if opened, err := Open(ctx, testDSN); err == nil {
			opened.Close()
			t.Fatal("Open 不得在缺少基线时继续运行")
		}
		if err := store.ApplySchema(ctx); err != nil {
			t.Fatal(err)
		}
		if err := store.VerifySchema(ctx); err != nil {
			t.Fatal(err)
		}
		opened, err := Open(ctx, testDSN)
		if err != nil {
			t.Fatal(err)
		}
		opened.Close()
		if err := store.ApplySchema(ctx); err == nil {
			t.Fatal("基线第二次执行必须失败")
		}
	})

	writer := ConsumerRoleToken{OwnerID: "writer-a", Epoch: 1}
	t.Run("fence idempotent elevate and reject stale", func(t *testing.T) {
		first, err := store.ActivateRole(ctx, "dep-a", "writer", writer, 0)
		if err != nil {
			t.Fatal(err)
		}
		retry, err := store.ActivateRole(ctx, "dep-a", "writer", writer, 999)
		if err != nil {
			t.Fatal(err)
		}
		if retry.Version != first.Version {
			t.Fatal("同 token 重试不应改变 version")
		}
		if _, err = store.ActivateRole(ctx, "dep-a", "writer", ConsumerRoleToken{OwnerID: "writer-b", Epoch: 1}, first.Version); !errors.Is(err, ErrFenceRejected) {
			t.Fatalf("同 epoch 换 owner 必须拒绝: %v", err)
		}
		if _, err = store.ActivateRole(ctx, "dep-a", "writer", ConsumerRoleToken{OwnerID: "writer-b", Epoch: 2}, 0); !errors.Is(err, ErrFenceRejected) {
			t.Fatalf("错误 expected version 必须拒绝: %v", err)
		}
		fence, err := store.ActivateRole(ctx, "dep-a", "writer", ConsumerRoleToken{OwnerID: "writer-b", Epoch: 2}, first.Version)
		if err != nil {
			t.Fatal(err)
		}
		if fence.Version != first.Version+1 {
			t.Fatal("提升 epoch 必须递增 version")
		}
		stale := Message{DeploymentID: "dep-a", AccountID: "acc-a", ConsumerKey: "writer.stale", Role: "writer", Token: ConsumerRoleToken{OwnerID: "writer-a", Epoch: 1}, EventID: strings.Repeat("e", 64), RawBody: []byte("old"), Subject: "data.raw.p", CheckpointPosition: 1, DeliveryCount: 1, OccurredAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
		if _, err := store.ProcessMessage(ctx, stale, func(context.Context, *BusinessTx) error { t.Fatal("旧 role epoch 不得进入 handler"); return nil }, ProcessOptions{}); !errors.Is(err, ErrFenceStale) {
			t.Fatalf("旧 role epoch 必须拒绝: %v", err)
		}
		writer = ConsumerRoleToken{OwnerID: "writer-b", Epoch: 2}
	})

	t.Run("different consumers do not swallow each other and duplicate collision", func(t *testing.T) {
		for _, role := range []string{"compute", "alarm"} {
			if _, err := store.ActivateRole(ctx, "dep-a", role, writer, 0); err != nil {
				t.Fatal(err)
			}
		}
		body := []byte(`{"value":1}`)
		event := strings.Repeat("a", 64)
		var calls atomic.Int32
		msg := Message{DeploymentID: "dep-a", AccountID: "acc-a", ConsumerKey: "writer.raw", Role: "writer", Token: writer, EventID: event, RawBody: body, Subject: "data.raw.point", CheckpointPosition: 10, DeliveryCount: 1, OccurredAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
		if got, err := store.ProcessMessage(ctx, msg, func(context.Context, *BusinessTx) error { calls.Add(1); return nil }, ProcessOptions{}); err != nil || got != Processed {
			t.Fatalf("first: %s %v", got, err)
		}
		msg.CheckpointPosition = 11
		if got, err := store.ProcessMessage(ctx, msg, func(context.Context, *BusinessTx) error { calls.Add(1); return nil }, ProcessOptions{}); err != nil || got != Duplicate {
			t.Fatalf("duplicate: %s %v", got, err)
		}
		var duplicateCheckpoint int64
		if err := store.pool.QueryRow(ctx, `SELECT position FROM runtime_engine.consumer_checkpoint WHERE deployment_id='dep-a' AND consumer_key='writer.raw'`).Scan(&duplicateCheckpoint); err != nil || duplicateCheckpoint != 11 {
			t.Fatalf("duplicate checkpoint: %d %v", duplicateCheckpoint, err)
		}
		other := msg
		other.ConsumerKey, other.Role = "compute.raw", "compute"
		if got, err := store.ProcessMessage(ctx, other, func(context.Context, *BusinessTx) error { calls.Add(1); return nil }, ProcessOptions{}); err != nil || got != Processed {
			t.Fatalf("other consumer: %s %v", got, err)
		}
		collision := msg
		collision.RawBody = []byte(`{"value":2}`)
		if _, err := store.ProcessMessage(ctx, collision, nil, ProcessOptions{}); err == nil {
			t.Fatal("collision 缺少 DLQ callback 必须回滚")
		}
		got, err := store.ProcessMessage(ctx, collision, func(context.Context, *BusinessTx) error { t.Fatal("collision 不可执行 handler"); return nil }, ProcessOptions{OnCollision: func(ctx context.Context, tx *BusinessTx, c CollisionInfo) error {
			_, _, err := tx.Enqueue(ctx, OutboxMessage{DeploymentID: c.Message.DeploymentID, DedupeKey: c.DLQID, Subject: "dlq.writer", Payload: c.Message.RawBody})
			return err
		}})
		if err != nil || got != Collision {
			t.Fatalf("collision: %s %v", got, err)
		}
		if calls.Load() != 2 {
			t.Fatalf("handler 次数=%d", calls.Load())
		}
		var failures int
		if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM runtime_engine.processing_failure`).Scan(&failures); err != nil || failures != 1 {
			t.Fatalf("collision quarantine: %d %v", failures, err)
		}
		failedCallback := collision
		failedCallback.RawBody = []byte(`{"value":3}`)
		if _, err := store.ProcessMessage(ctx, failedCallback, nil, ProcessOptions{OnCollision: func(context.Context, *BusinessTx, CollisionInfo) error { return errors.New("dlq unavailable") }}); err == nil {
			t.Fatal("DLQ callback 失败必须 rollback")
		}
		if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM runtime_engine.processing_failure`).Scan(&failures); err != nil || failures != 1 {
			t.Fatalf("collision callback rollback: %d %v", failures, err)
		}
	})

	t.Run("handler failure fully rolls back", func(t *testing.T) {
		id := strings.Repeat("b", 64)
		msg := Message{DeploymentID: "dep-a", AccountID: "acc-a", ConsumerKey: "writer.failure", Role: "writer", Token: writer, EventID: id, RawBody: []byte("bad"), Subject: "data.raw.p", CheckpointPosition: 1, DeliveryCount: 1, OccurredAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
		_, err := store.ProcessMessage(ctx, msg, func(ctx context.Context, tx *BusinessTx) error {
			_, _, enqueueErr := tx.Enqueue(ctx, OutboxMessage{DeploymentID: "dep-a", DedupeKey: "rollback", Subject: "x", Payload: []byte("x")})
			if enqueueErr != nil {
				return enqueueErr
			}
			return errors.New("handler failed")
		}, ProcessOptions{})
		if err == nil {
			t.Fatal("handler failure 必须返回错误")
		}
		var count int
		if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM runtime_engine.processed_event WHERE consumer_key='writer.failure'`).Scan(&count); err != nil || count != 0 {
			t.Fatalf("processed rollback: %d %v", count, err)
		}
		if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM runtime_engine.transactional_outbox WHERE dedupe_key='rollback'`).Scan(&count); err != nil || count != 0 {
			t.Fatalf("outbox rollback: %d %v", count, err)
		}
	})

	t.Run("concurrent same event invokes handler once", func(t *testing.T) {
		msg := Message{DeploymentID: "dep-a", AccountID: "acc-a", ConsumerKey: "writer.concurrent", Role: "writer", Token: writer, EventID: strings.Repeat("c", 64), RawBody: []byte("same"), Subject: "data.raw.p", CheckpointPosition: 2, DeliveryCount: 1, OccurredAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
		var handlers atomic.Int32
		var processed atomic.Int32
		var wg sync.WaitGroup
		errorsCh := make(chan error, 12)
		for range 12 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				d, err := store.ProcessMessage(ctx, msg, func(context.Context, *BusinessTx) error { handlers.Add(1); return nil }, ProcessOptions{})
				if err != nil {
					errorsCh <- err
					return
				}
				if d == Processed {
					processed.Add(1)
				}
			}()
		}
		wg.Wait()
		close(errorsCh)
		for err := range errorsCh {
			t.Error(err)
		}
		if handlers.Load() != 1 || processed.Load() != 1 {
			t.Fatalf("handlers=%d processed=%d", handlers.Load(), processed.Load())
		}
	})

	t.Run("outbox claim lease retry and puback crash replay", func(t *testing.T) {
		enqueueForTest(t, store, ctx, OutboxMessage{DeploymentID: "dep-a", DedupeKey: "one", Subject: "out.one", Headers: []byte(`{"x":"1"}`), Payload: []byte("payload-one")})
		enqueueForTest(t, store, ctx, OutboxMessage{DeploymentID: "dep-a", DedupeKey: "two", Subject: "out.two", Payload: []byte("payload-two")})
		var left, right []OutboxRecord
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); left, _ = store.ClaimOutbox(ctx, "dep-a", "publisher-a", 1, time.Second) }()
		go func() { defer wg.Done(); right, _ = store.ClaimOutbox(ctx, "dep-a", "publisher-b", 1, time.Second) }()
		wg.Wait()
		if len(left) != 1 || len(right) != 1 || left[0].ID == right[0].ID {
			t.Fatalf("并发 claim 重复或缺失: %#v %#v", left, right)
		}
		if err := store.MarkPublished(ctx, left[0].ID, "stale-token"); !errors.Is(err, ErrLeaseLost) {
			t.Fatalf("stale token: %v", err)
		}
		if err := store.MarkPublished(ctx, left[0].ID, left[0].LeaseToken); err != nil {
			t.Fatal(err)
		}
		if err := store.RetryOutbox(ctx, right[0].ID, right[0].LeaseToken, time.Now().Add(-time.Second), RetryPublishTimeout); err != nil {
			t.Fatal(err)
		}
		reclaimed, err := store.ClaimOutbox(ctx, "dep-a", "publisher-c", 1, 20*time.Millisecond)
		if err != nil || len(reclaimed) != 1 || reclaimed[0].ID != right[0].ID {
			t.Fatalf("retry claim: %#v %v", reclaimed, err)
		}
		if err := store.MarkPublished(ctx, reclaimed[0].ID, reclaimed[0].LeaseToken); err != nil {
			t.Fatal(err)
		}
		// 模拟获得 PubAck 后进程崩溃：不 MarkPublished，等待租约过期并比较稳定 bytes/msg id。
		crash, err := store.ClaimOutbox(ctx, "dep-a", "publisher-crash", 1, 15*time.Millisecond)
		if err != nil || len(crash) != 1 {
			t.Fatalf("crash claim: %#v %v", crash, err)
		}
		time.Sleep(25 * time.Millisecond)
		afterCrash, err := store.ClaimOutbox(ctx, "dep-a", "publisher-recover", 1, time.Second)
		if err != nil || len(afterCrash) != 1 {
			t.Fatalf("recover claim: %#v %v", afterCrash, err)
		}
		if crash[0].DedupeKey != afterCrash[0].DedupeKey || string(crash[0].Payload) != string(afterCrash[0].Payload) {
			t.Fatal("重领必须使用完全相同 Msg-Id 与 payload")
		}
	})

	t.Run("producer sequence unique and old epoch rejected", func(t *testing.T) {
		old := ProducerToken{OwnerID: "producer-a", Epoch: 1}
		// 同值 token 落在不同表中；更新 consumer fence 不得影响 producer sequence。
		if _, err := store.ActivateRole(ctx, "dep-a", "shared", ConsumerRoleToken{OwnerID: "producer-a", Epoch: 1}, 0); err != nil {
			t.Fatal(err)
		}
		if _, err := store.ActivateProducer(ctx, "dep-a", "shared", old, 0); err != nil {
			t.Fatal(err)
		}
		if _, err := store.ActivateRole(ctx, "dep-a", "shared", ConsumerRoleToken{OwnerID: "consumer-b", Epoch: 2}, 1); err != nil {
			t.Fatal(err)
		}
		if err := store.VerifyProducer(ctx, "dep-a", "shared", old); err != nil {
			t.Fatalf("consumer 变更不得影响同值 producer token: %v", err)
		}
		if _, err := store.ActivateProducer(ctx, "dep-a", "producer", old, 0); err != nil {
			t.Fatal(err)
		}
		values := make(chan int64, 20)
		errorsCh := make(chan error, 20)
		var wg sync.WaitGroup
		for range 20 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				n, err := store.NextProducerSequence(ctx, "dep-a", "producer", old)
				if err != nil {
					errorsCh <- err
				} else {
					values <- n
				}
			}()
		}
		wg.Wait()
		close(values)
		close(errorsCh)
		for err := range errorsCh {
			t.Error(err)
		}
		seen := []int{}
		for n := range values {
			seen = append(seen, int(n))
		}
		sort.Ints(seen)
		if len(seen) != 20 {
			t.Fatalf("sequence count=%d", len(seen))
		}
		for i, n := range seen {
			if i != n {
				t.Fatalf("sequence=%v", seen)
			}
		}
		newToken := ProducerToken{OwnerID: "producer-b", Epoch: 2}
		if _, err := store.ActivateProducer(ctx, "dep-a", "producer", newToken, 1); err != nil {
			t.Fatal(err)
		}
		if _, err := store.NextProducerSequence(ctx, "dep-a", "producer", old); !errors.Is(err, ErrFenceStale) {
			t.Fatalf("old producer epoch: %v", err)
		}
		if n, err := store.NextProducerSequence(ctx, "dep-a", "producer", newToken); err != nil || n != 0 {
			t.Fatalf("new epoch first seq: %d %v", n, err)
		}
		lockConn, err := store.pool.Acquire(ctx)
		if err != nil {
			t.Fatal(err)
		}
		lockTx, err := lockConn.Begin(ctx)
		if err != nil {
			lockConn.Release()
			t.Fatal(err)
		}
		if _, err = lockTx.Exec(ctx, `LOCK TABLE runtime_engine.producer_fence IN ACCESS EXCLUSIVE MODE`); err != nil {
			lockTx.Rollback(ctx)
			lockConn.Release()
			t.Fatal(err)
		}
		deadline, cancel := context.WithTimeout(ctx, 30*time.Millisecond)
		if err = store.VerifyProducer(deadline, "dep-a", "producer", newToken); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("锁/截止错误必须原样保留: %v", err)
		}
		cancel()
		deadline, cancel = context.WithTimeout(ctx, 30*time.Millisecond)
		if _, err = store.NextProducerSequence(deadline, "dep-a", "producer", newToken); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("sequence 锁/截止错误必须原样保留: %v", err)
		}
		cancel()
		_ = lockTx.Rollback(ctx)
		lockConn.Release()
		// 不同值 consumer token 仍只在 consumer fence 中生效，绝不与 producer token 比较。
		consumer := ConsumerRoleToken{OwnerID: "consumer-b", Epoch: 2}
		message := Message{DeploymentID: "dep-a", AccountID: "acc-a", ConsumerKey: "shared.consumer", Role: "shared", Token: consumer, EventID: strings.Repeat("d", 64), RawBody: []byte("independent"), Subject: "data.raw.p", CheckpointPosition: 1, DeliveryCount: 1, OccurredAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
		if disposition, err := store.ProcessMessage(ctx, message, nil, ProcessOptions{}); err != nil || disposition != Processed {
			t.Fatalf("different consumer token must not depend on producer fence: %s %v", disposition, err)
		}
	})

	t.Run("permanent failure is evidence plus stable dlq outbox", func(t *testing.T) {
		body := []byte(`{"bad":true}`)
		digest := eventid.BodySHA256(body)
		dlqID, err := runtimeDLQID("dep-a", "writer.dlq", "data.raw.p", "", FailurePermanentValidation, digest)
		if err != nil {
			t.Fatal(err)
		}
		failure := PermanentFailure{DeploymentID: "dep-a", AccountID: "acc-a", ConsumerKey: "writer.dlq", Role: "writer", Token: writer, DLQID: dlqID, Subject: "data.raw.p", RawBody: body, BodySHA256: digest, ReasonCode: FailurePermanentValidation, DeliveryCount: 1, CheckpointPosition: 20, OccurredAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)}
		if err := store.ProcessPermanentFailure(ctx, failure, nil); err == nil {
			t.Fatal("永久失败缺少 DLQ callback 必须拒绝")
		}
		if err := store.ProcessPermanentFailure(ctx, failure, func(context.Context, *BusinessTx, PermanentFailure) error { return errors.New("outbox failure") }); err == nil {
			t.Fatal("DLQ callback 失败必须 rollback")
		}
		calls := 0
		callback := func(ctx context.Context, tx *BusinessTx, f PermanentFailure) error {
			calls++
			_, _, err := tx.Enqueue(ctx, OutboxMessage{DeploymentID: f.DeploymentID, DedupeKey: f.DLQID, Subject: "dlq.writer", Payload: f.RawBody})
			return err
		}
		if err := store.ProcessPermanentFailure(ctx, failure, callback); err != nil {
			t.Fatal(err)
		}
		redelivery := failure
		redelivery.DeliveryCount = 2
		redelivery.CheckpointPosition = 21
		redelivery.OccurredAt = redelivery.OccurredAt.Add(time.Minute)
		if err := store.ProcessPermanentFailure(ctx, redelivery, callback); err != nil {
			t.Fatal(err)
		}
		if calls != 1 {
			t.Fatalf("commit-before-Ack 重投不得重建 outbox: %d", calls)
		}
		var checkpoint int64
		if err := store.pool.QueryRow(ctx, `SELECT position FROM runtime_engine.consumer_checkpoint WHERE deployment_id='dep-a' AND consumer_key='writer.dlq'`).Scan(&checkpoint); err != nil || checkpoint != 21 {
			t.Fatalf("failure checkpoint: %d %v", checkpoint, err)
		}
	})

	t.Run("apply point current orders atomically and versions internally", func(t *testing.T) {
		pointID := "11111111-1111-4111-8111-111111111111"
		base := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)
		apply := func(event string, epoch, sequence int64, source time.Time, position int64, fail bool) (PointCurrentResult, error) {
			message := Message{DeploymentID: "dep-a", AccountID: "acc-a", ConsumerKey: "writer.current", Role: "writer", Token: writer, EventID: event, RawBody: []byte(event), Subject: "data.raw.p", CheckpointPosition: position, DeliveryCount: 1, OccurredAt: base}
			var result PointCurrentResult
			_, err := store.ProcessMessage(ctx, message, func(ctx context.Context, tx *BusinessTx) error {
				var applyErr error
				result, applyErr = tx.ApplyPointCurrent(ctx, PointCurrentWrite{DeploymentID: "dep-a", PointID: pointID, OwnerID: "writer-b", EventID: event, Quality: "good", Epoch: epoch, Sequence: sequence, SourceTimestamp: source, ServerTimestamp: base, Value: []byte(`1`)})
				if applyErr != nil {
					return applyErr
				}
				if fail {
					return errors.New("rollback")
				}
				return nil
			}, ProcessOptions{})
			return result, err
		}
		if r, err := apply(strings.Repeat("1", 64), 1, 1, base, 30, false); err != nil || r.Disposition != PointCurrentApplied || r.Version != 1 {
			t.Fatalf("first %#v %v", r, err)
		}
		if r, err := apply(strings.Repeat("2", 64), 1, 2, base.Add(time.Second), 31, false); err != nil || r.Disposition != PointCurrentApplied || r.Version != 2 {
			t.Fatalf("update %#v %v", r, err)
		}
		if r, err := apply(strings.Repeat("3", 64), 1, 3, base, 32, false); err != nil || r.Disposition != PointCurrentIgnored || r.Version != 2 {
			t.Fatalf("out of order %#v %v", r, err)
		}
		if r, err := apply(strings.Repeat("4", 64), 2, 0, base, 33, false); err != nil || r.Disposition != PointCurrentApplied || r.Version != 3 {
			t.Fatalf("epoch %#v %v", r, err)
		}
		if r, err := apply(strings.Repeat("0", 64), 2, 0, base, 34, false); err != nil || r.Disposition != PointCurrentIgnored || r.Version != 3 {
			t.Fatalf("tie lower %#v %v", r, err)
		}
		if r, err := apply(strings.Repeat("f", 64), 2, 0, base, 35, false); err != nil || r.Disposition != PointCurrentApplied || r.Version != 4 {
			t.Fatalf("tie higher %#v %v", r, err)
		}
		if _, err := apply(strings.Repeat("e", 64), 3, 0, base, 36, true); err == nil {
			t.Fatal("business transaction rollback required")
		}
		var version int64
		if err := store.pool.QueryRow(ctx, `SELECT version FROM runtime_engine.point_current WHERE deployment_id='dep-a' AND point_id=$1::uuid`, pointID).Scan(&version); err != nil || version != 4 {
			t.Fatalf("rollback version %d %v", version, err)
		}
	})

	t.Run("schema catalog verification rejects forged version", func(t *testing.T) {
		if _, err := store.pool.Exec(ctx, `ALTER TABLE runtime_engine.point_current ADD COLUMN forged text`); err != nil {
			t.Fatal(err)
		}
		if err := store.VerifySchema(ctx); err == nil {
			t.Fatal("只保留 schema_meta 版本但篡改列布局必须被拒绝")
		}
		if _, err := store.pool.Exec(ctx, `ALTER TABLE runtime_engine.processed_event DROP CONSTRAINT processed_event_event_id_check; ALTER TABLE runtime_engine.processed_event ADD CONSTRAINT processed_event_event_id_check CHECK (length(event_id)=64)`); err != nil {
			t.Fatal(err)
		}
		if err := store.VerifySchema(ctx); err == nil {
			t.Fatal("弱化 eventId CHECK 必须被拒绝")
		}
		if _, err := store.pool.Exec(ctx, `ALTER TABLE runtime_engine.processed_event DROP CONSTRAINT processed_event_event_id_check; ALTER TABLE runtime_engine.processed_event ADD CONSTRAINT processed_event_event_id_check CHECK (event_id ~ '^[0-9a-f]{64}$' OR true)`); err != nil {
			t.Fatal(err)
		}
		if err := store.VerifySchema(ctx); err == nil {
			t.Fatal("eventId CHECK OR true 必须被拒绝")
		}
		if _, err := store.pool.Exec(ctx, `ALTER TABLE runtime_engine.processed_event ALTER COLUMN processed_at DROP DEFAULT`); err != nil {
			t.Fatal(err)
		}
		if err := store.VerifySchema(ctx); err == nil {
			t.Fatal("删除 processed_at default 必须被拒绝")
		}
		if _, err := store.pool.Exec(ctx, `DROP INDEX runtime_engine.transactional_outbox_claim_idx; CREATE INDEX transactional_outbox_claim_idx ON runtime_engine.transactional_outbox (deployment_id,state,next_attempt_at,lease_until,id) WHERE false`); err != nil {
			t.Fatal(err)
		}
		if err := store.VerifySchema(ctx); err == nil {
			t.Fatal("outbox partial WHERE false 必须被拒绝")
		}
		if _, err := store.pool.Exec(ctx, `ALTER TABLE runtime_engine.transactional_outbox ALTER COLUMN id DROP IDENTITY`); err != nil {
			t.Fatal(err)
		}
		if err := store.VerifySchema(ctx); err == nil {
			t.Fatal("identity 删除必须被拒绝")
		}
		if _, err := store.pool.Exec(ctx, `ALTER TABLE runtime_engine.point_history SET UNLOGGED`); err != nil {
			t.Fatal(err)
		}
		if err := store.VerifySchema(ctx); err == nil {
			t.Fatal("UNLOGGED 基线表必须被拒绝")
		}
	})
}

func setupIntegrationStore(t *testing.T, ctx context.Context, dsn string) (*Store, string, func()) {
	t.Helper()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal("连接测试 PostgreSQL 失败")
	}
	name := "runtime_engine_it_" + strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_") + "_" + strings.ReplaceAll(time.Now().UTC().Format("150405.000000"), ".", "")
	name = strings.ReplaceAll(name, "-", "_")
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
		admin.Close()
		t.Fatal("创建隔离测试数据库失败")
	}
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.Database = name
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		admin.Close()
		t.Fatal(err)
	}
	store := &Store{pool: pool}
	parsed, err := url.Parse(dsn)
	if err != nil {
		pool.Close()
		admin.Close()
		t.Fatal(err)
	}
	parsed.Path = "/" + name
	return store, parsed.String(), func() {
		pool.Close()
		_, _ = admin.Exec(ctx, "DROP DATABASE "+pgx.Identifier{name}.Sanitize())
		admin.Close()
	}
}

func enqueueForTest(t *testing.T, store *Store, ctx context.Context, message OutboxMessage) {
	t.Helper()
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, _, err := store.enqueueTx(ctx, tx, message); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}
