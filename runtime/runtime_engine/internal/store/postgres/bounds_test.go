package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/indu-forge/runtime-engine/internal/eventid"
)

func TestIngressBoundsRejectBeforeDatabase(t *testing.T) {
	base := Message{DeploymentID: "dep", AccountID: "account", ConsumerKey: "consumer", Role: "writer", Token: ConsumerRoleToken{OwnerID: "owner", Epoch: 1}, EventID: strings.Repeat("a", 64), RawBody: []byte("x"), Subject: "data.raw.p", CheckpointPosition: 0, DeliveryCount: 1, OccurredAt: time.Now().UTC()}
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
