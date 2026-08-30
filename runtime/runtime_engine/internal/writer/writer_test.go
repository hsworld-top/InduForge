package writer

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/indu-forge/runtime-engine/internal/ingress"
	"github.com/indu-forge/runtime-engine/internal/model"
	"github.com/indu-forge/runtime-engine/internal/store/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	writerDeployment = "writer-it"
	writerAccount    = "writer-account"
	writerOwner      = "writer-a"
	writerPoint      = "11111111-1111-4111-8111-111111111111"
)

func TestNormalizeRejectsNonUTCTimeAndMismatchedSubject(t *testing.T) {
	message := writerMessage(strings.Repeat("a", 64), 1, time.Date(2026, 8, 30, 1, 2, 3, 4, time.UTC), `1e3`)
	write, err := normalize(message)
	if err != nil {
		t.Fatal(err)
	}
	if string(write.history.Value) != `1e3` || !write.history.ReceivedAt.Equal(time.Date(2026, 8, 30, 1, 2, 5, 4, time.UTC)) {
		t.Fatalf("JSON/time 被意外改变: %#v", write.history)
	}
	message.Event.SourceTimestamp = "2026-08-30T09:02:03+08:00"
	if _, err := normalize(message); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("non-UTC=%v", err)
	}
	message = writerMessage(strings.Repeat("a", 64), 1, time.Now().UTC(), `1`)
	message.Event.Subject = "data.raw.22222222-2222-4222-8222-222222222222"
	if _, err := normalize(message); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("subject mismatch=%v", err)
	}
}

func TestPostgresWriterV1(t *testing.T) {
	dsn := os.Getenv("RUNTIME_ENGINE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("RUNTIME_ENGINE_TEST_DATABASE_URL 未设置")
	}
	ctx := context.Background()
	store, query, cleanup := setupWriterStore(t, ctx, dsn)
	defer cleanup()
	token := postgres.ConsumerRoleToken{OwnerID: writerOwner, Epoch: 1}
	if _, err := store.ActivateRole(ctx, writerDeployment, "writer", token, 0); err != nil {
		t.Fatal(err)
	}
	producer := postgres.ProducerToken{OwnerID: "collector-writer", Epoch: 1}
	if _, err := store.ActivateProducer(ctx, writerDeployment, "collector-writer", producer, 0); err != nil {
		t.Fatal(err)
	}
	handler := NewPostgresHandler()
	adapter := ingress.NewPostgresAdapter(store, model.EngineConfig{}, handler)
	base := time.Date(2026, 8, 30, 1, 2, 3, 0, time.UTC)

	process := func(message ingress.ValidatedMessage) ingress.ProcessResult {
		t.Helper()
		result, err := adapter.Process(ctx, message)
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	first := writerMessage(strings.Repeat("1", 64), 1, base, `1e3`)
	if result := process(first); result != ingress.Processed {
		t.Fatalf("first=%v", result)
	}
	if result := process(first); result != ingress.Duplicate {
		t.Fatalf("duplicate=%v", result)
	}
	assertPointCounts(t, ctx, query, 1, 1)

	newer := writerMessage(strings.Repeat("2", 64), 2, base.Add(time.Second), `2`)
	if result := process(newer); result != ingress.Processed {
		t.Fatalf("newer=%v", result)
	}
	older := writerMessage(strings.Repeat("3", 64), 3, base, `3`)
	if result := process(older); result != ingress.Processed {
		t.Fatalf("older=%v", result)
	}
	assertPointCounts(t, ctx, query, 3, 1)
	assertCurrent(t, ctx, query, strings.Repeat("2", 64), 1, 2)

	// 同 epoch/time/sequence 时 eventId 是最终稳定 tie-breaker。
	tieLower := writerMessage(strings.Repeat("0", 64), 2, base.Add(time.Second), `4`)
	if result := process(tieLower); result != ingress.Processed {
		t.Fatalf("tie lower=%v", result)
	}
	tieHigher := writerMessage(strings.Repeat("f", 64), 2, base.Add(time.Second), `5`)
	if result := process(tieHigher); result != ingress.Processed {
		t.Fatalf("tie higher=%v", result)
	}
	assertPointCounts(t, ctx, query, 5, 1)
	assertCurrent(t, ctx, query, strings.Repeat("f", 64), 1, 3)

	rollback := writerMessage(strings.Repeat("e", 64), 1, base.Add(2*time.Second), `6`)
	failing := ingress.NewPostgresAdapter(store, model.EngineConfig{}, func(ctx context.Context, tx *postgres.BusinessTx, message ingress.ValidatedMessage) error {
		if err := HandlePostgres(ctx, tx, message); err != nil {
			return err
		}
		return errors.New("injected handler failure")
	})
	if _, err := failing.Process(ctx, rollback); err == nil {
		t.Fatal("handler 中途失败必须回滚")
	}
	assertPointCounts(t, ctx, query, 5, 1)
	var processed int
	if err := query.QueryRow(ctx, `SELECT count(*) FROM runtime_engine.processed_event WHERE event_id=$1`, rollback.Event.EventID).Scan(&processed); err != nil || processed != 0 {
		t.Fatalf("processed rollback count=%d err=%v", processed, err)
	}
}

func writerMessage(eventID string, sequence int64, source time.Time, value string) ingress.ValidatedMessage {
	server := source.Add(time.Second)
	received := source.Add(2 * time.Second)
	event := ingress.Event{
		SchemaVersion: "data.raw.v1", Subject: "data.raw." + writerPoint, EventID: eventID,
		DeploymentID: writerDeployment, AccountID: writerAccount, PointID: writerPoint,
		OwnerID: "collector-a", Epoch: 1, Sequence: sequence, Value: json.RawMessage(value), Quality: "good",
		SourceTimestamp: source.UTC().Format(time.RFC3339Nano), ServerTimestamp: server.UTC().Format(time.RFC3339Nano), ReceivedAt: received.UTC().Format(time.RFC3339Nano),
	}
	body, _ := json.Marshal(event)
	return ingress.ValidatedMessage{Event: event, RawBody: body, Consumer: model.Consumer{Role: "writer", ConsumerKey: "writer.raw"}, Token: ingress.ConsumerToken{OwnerID: writerOwner, Epoch: 1}, ProducerKey: "collector-writer", ProducerToken: ingress.ProducerToken{OwnerID: "collector-writer", Epoch: 1}, StreamPosition: sequence + 1, DeliveryCount: 1, OccurredAt: received}
}

func assertPointCounts(t *testing.T, ctx context.Context, pool *pgxpool.Pool, history, current int) {
	t.Helper()
	var gotHistory, gotCurrent int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM runtime_engine.point_history`).Scan(&gotHistory); err != nil || gotHistory != history {
		t.Fatalf("history=%d err=%v", gotHistory, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM runtime_engine.point_current`).Scan(&gotCurrent); err != nil || gotCurrent != current {
		t.Fatalf("current=%d err=%v", gotCurrent, err)
	}
}

func assertCurrent(t *testing.T, ctx context.Context, pool *pgxpool.Pool, eventID string, epoch, version int64) {
	t.Helper()
	var gotEvent string
	var gotEpoch, gotVersion int64
	if err := pool.QueryRow(ctx, `SELECT event_id,epoch,version FROM runtime_engine.point_current WHERE deployment_id=$1 AND point_id=$2::uuid`, writerDeployment, writerPoint).Scan(&gotEvent, &gotEpoch, &gotVersion); err != nil || gotEvent != eventID || gotEpoch != epoch || gotVersion != version {
		t.Fatalf("current event=%s epoch=%d version=%d err=%v", gotEvent, gotEpoch, gotVersion, err)
	}
}

func setupWriterStore(t *testing.T, ctx context.Context, dsn string) (*postgres.Store, *pgxpool.Pool, func()) {
	t.Helper()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	name := "runtime_writer_" + strings.ReplaceAll(strings.ToLower(t.Name()), "/", "_") + "_" + time.Now().UTC().Format("150405000000")
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.Database = name
	bootstrap, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bootstrap.Exec(ctx, postgres.SchemaSQL); err != nil {
		bootstrap.Close()
		t.Fatal(err)
	}
	bootstrap.Close()
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	parsed.Path = "/" + name
	store, err := postgres.Open(ctx, parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	query, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	return store, query, func() {
		query.Close()
		store.Close()
		_, _ = admin.Exec(ctx, "DROP DATABASE "+pgx.Identifier{name}.Sanitize())
		admin.Close()
	}
}
