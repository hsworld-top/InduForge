package event

import (
	"encoding/json"
	"github.com/indu-forge/collector-engine/internal/loader"
	"testing"
	"time"
)

func TestNewUsesSafePointSubjectAndQuality(t *testing.T) {
	b := loader.Binding{DeploymentID: "deployment-a", AccountID: "account-a", CollectorID: "collector-a", Ownership: loader.Ownership{OwnerID: "owner-a", Epoch: 1}}
	m := loader.Mapping{DatapointID: "22222222-2222-4222-8222-222222222222", ConnectionID: "33333333-3333-4333-8333-333333333333", VariableID: "44444444-4444-4444-8444-444444444444"}
	e, err := New(b, m, 0, json.RawMessage(`1`), "good", time.Now(), time.Now())
	if err != nil || e.Subject != "data.raw."+m.DatapointID || len(e.EventID) != 64 {
		t.Fatalf("raw envelope invalid: %#v %v", e, err)
	}
	if _, err := New(b, m, 0, json.RawMessage(`1`), "device-error", time.Now(), time.Now()); err == nil {
		t.Fatal("must reject unknown quality")
	}
}
