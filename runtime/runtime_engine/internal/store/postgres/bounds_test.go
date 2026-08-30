package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/indu-forge/runtime-engine/internal/eventid"
	"github.com/indu-forge/runtime-engine/internal/transportlimits"
)

func TestIngressBoundsRejectBeforeDatabase(t *testing.T) {
	base := Message{DeploymentID: "dep", AccountID: "account", ConsumerKey: "consumer", Role: "writer", Token: ConsumerRoleToken{OwnerID: "owner", Epoch: 1}, ProducerKey: "collector", ProducerToken: ProducerToken{OwnerID: "collector", Epoch: 1}, EventID: strings.Repeat("a", 64), RawBody: []byte("x"), Subject: "data.raw.p", CheckpointPosition: 0, DeliveryCount: 1, OccurredAt: time.Now().UTC()}
	if err := validMessage(base); err != nil {
		t.Fatal(err)
	}
	tooLarge := base
	tooLarge.RawBody = make([]byte, (1<<20)+1)
	if err := validMessage(tooLarge); err == nil {
		t.Fatal("raw body >1MiB 必须拒绝")
	}
	tooLarge = base
	tooLarge.DeliveryMetadata = make([]byte, (16<<10)+1)
	if err := validMessage(tooLarge); err == nil {
		t.Fatal("metadata >16KiB 必须拒绝")
	}
	tooLarge = base
	tooLarge.Subject = strings.Repeat("s", 4096)
	if err := validMessage(tooLarge); err != nil {
		t.Fatal("subject=4096 必须允许")
	}
	tooLarge.Subject = strings.Repeat("s", 4097)
	if err := validMessage(tooLarge); err == nil {
		t.Fatal("subject>4096 必须拒绝")
	}
	body := make([]byte, (1<<20)+1)
	digest := eventid.BodySHA256(body)
	dlq, err := runtimeDLQID("dep", "consumer", "data.raw.p", "", FailurePermanentValidation, digest)
	if err != nil {
		t.Fatal(err)
	}
	failure := PermanentFailure{DeploymentID: "dep", AccountID: "account", ConsumerKey: "consumer", Role: "writer", Token: base.Token, DLQID: dlq, Subject: "data.raw.p", RawBody: body, BodySHA256: digest, ReasonCode: FailurePermanentValidation, DeliveryCount: 1, OccurredAt: time.Now().UTC()}
	if err := validPermanentFailure(failure); err == nil {
		t.Fatal("failure raw body >1MiB 必须拒绝")
	}
	if _, err := objectJSON(make([]byte, (16<<10)+1)); err == nil {
		t.Fatal("outbox headers >16KiB 必须拒绝")
	}
}

func TestPermanentFailureProducerFenceIsAllOrNothingAndReasonBound(t *testing.T) {
	body := []byte(`{"value":"retry"}`)
	digest := eventid.BodySHA256(body)
	eventID := strings.Repeat("a", 64)
	dlq, err := runtimeDLQID("dep", "consumer", "data.raw.p", eventID, FailureHandler, digest)
	if err != nil {
		t.Fatal(err)
	}
	base := PermanentFailure{DeploymentID: "dep", AccountID: "account", ConsumerKey: "consumer", Role: "writer", Token: ConsumerRoleToken{OwnerID: "writer", Epoch: 1}, RequireProducerFence: true, ProducerKey: ProducerFenceKey("collector"), ProducerToken: ProducerToken{OwnerID: "collector", Epoch: 1}, DLQID: dlq, EventID: &eventID, Subject: "data.raw.p", RawBody: body, BodySHA256: digest, ReasonCode: FailureHandler, DeliveryCount: 2, OccurredAt: time.Now().UTC()}
	if err := validPermanentFailure(base); err != nil {
		t.Fatalf("已验证 handler 最终失败必须允许: %v", err)
	}
	missing := base
	missing.ProducerKey = ""
	if err := validPermanentFailure(missing); err == nil {
		t.Fatal("RequireProducerFence 不得接受空 producer key")
	}
	wrongReason := base
	wrongReason.ReasonCode = FailurePermanentValidation
	if err := validPermanentFailure(wrongReason); err == nil {
		t.Fatal("生产者 fence 只能随已验证业务最终失败写入")
	}
	unrequired := base
	unrequired.RequireProducerFence = false
	if err := validPermanentFailure(unrequired); err == nil {
		t.Fatal("未要求 producer fence 时不得携带 key/token")
	}
}

func TestClaimOutboxRejectsUnboundedOrUnstableInputs(t *testing.T) {
	store := &Store{}
	for _, test := range []struct {
		deployment, owner string
		limit             int
		lease             time.Duration
	}{{"dep", "owner", 501, time.Second}, {"dep", "owner", 0, time.Second}, {"bad space", "owner", 1, time.Second}, {"dep", strings.Repeat("o", 129), 1, time.Second}, {"dep", "owner", 1, 0}, {"dep", "owner", 1, 16 * time.Minute}} {
		if _, err := store.ClaimOutbox(context.Background(), test.deployment, test.owner, test.limit, test.lease); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("%+v: %v", test, err)
		}
	}
}

func TestFiniteJSONRejectsTrailingValuesAndGarbage(t *testing.T) {
	for _, raw := range []string{"1 2", `{"value":1} trailing`} {
		if validFiniteJSON([]byte(raw)) {
			t.Fatalf("trailing JSON accepted: %q", raw)
		}
	}
}

func TestOutboxPayloadBoundsAreCheckedBeforeTransactionUse(t *testing.T) {
	message := OutboxMessage{DeploymentID: "dep", DedupeKey: "key", Subject: "data.computed.11111111-1111-4111-8111-111111111111", Payload: make([]byte, transportlimits.MaxBodyBytes+1)}
	if _, _, err := (&Store{}).enqueueTx(context.Background(), nil, message); !errors.Is(err, transportlimits.ErrOutboundPayloadTooLarge) {
		t.Fatalf("data payload error=%v", err)
	}
	message.Subject = "dlq.writer"
	message.Payload = make([]byte, transportlimits.MaxDLQPayloadBytes)
	// Exact DLQ boundary gets past local validation and only then observes the
	// intentionally nil unit-test transaction.
	if _, _, err := (&Store{}).enqueueTx(context.Background(), nil, message); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("exact DLQ boundary must not be rejected by payload guard: %v", err)
	}
	message.Payload = make([]byte, transportlimits.MaxDLQPayloadBytes+1)
	if _, _, err := (&Store{}).enqueueTx(context.Background(), nil, message); !errors.Is(err, transportlimits.ErrOutboundPayloadTooLarge) {
		t.Fatalf("DLQ payload error=%v", err)
	}
}

func TestFiniteJSONRetainsExactLargeAndNestedDecimals(t *testing.T) {
	for _, raw := range []string{`1e1000`, `9007199254740993`, `1.0000000000000000000000000001`, `{"nested":[1e1000,9007199254740993,{"decimal":1.0000000000000000000000000001}]}`} {
		if !validFiniteJSON([]byte(raw)) {
			t.Fatalf("exact finite JSON rejected: %s", raw)
		}
	}
	for _, raw := range []string{`1e10001`, `1 trailing`, `{"x":1} {}`} {
		if validFiniteJSON([]byte(raw)) {
			t.Fatalf("invalid JSON accepted: %s", raw)
		}
	}
}
