package driver

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gopcua/opcua/ua"
	"github.com/indu-forge/collector-engine/internal/loader"
	"github.com/indu-forge/collector-engine/internal/resolver"
	"testing"
	"time"
)

type fakeUA struct {
	reads    int
	response *ua.ReadResponse
	err      error
	connects int
	closed   int
}

func (f *fakeUA) Connect(context.Context) error { f.connects++; return nil }
func (f *fakeUA) Close(context.Context) error   { f.closed++; return nil }
func (f *fakeUA) Read(_ context.Context, r *ua.ReadRequest) (*ua.ReadResponse, error) {
	f.reads += len(r.NodesToRead)
	return f.response, f.err
}
func TestOPCUABatchMapsStatusAndSourceTime(t *testing.T) {
	resource, _ := json.Marshal(map[string]string{"url": "opc.tcp://127.0.0.1:4840"})
	r, _ := resolver.New(&loader.Loaded{Index: loader.Index{Resources: map[string]json.RawMessage{"site-resource://site/ua": resource}, Secrets: map[string]string{}}, IndexDir: t.TempDir()})
	f := &fakeUA{response: &ua.ReadResponse{Results: []*ua.DataValue{{Value: ua.MustVariant(int32(7)), Status: ua.StatusOK, SourceTimestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}, {Status: ua.StatusBadTimeout}}}}
	d := NewOPCUA(r)
	d.newClient = func(string) (uaClient, error) { return f, nil }
	ms := []loader.Mapping{{DatapointID: "a", Address: json.RawMessage(`{"nodeId":"i=1"}`)}, {DatapointID: "b", Address: json.RawMessage(`{"nodeId":"i=2"}`)}}
	out, e := d.ReadBatch(context.Background(), loader.Connection{ConnectionID: "c"}, loader.BindingConnection{ResourceRef: "site-resource://site/ua"}, ms)
	if e != nil || f.reads != 2 || string(out["a"].Value) != "7" || out["b"].Quality != "bad" || !out["a"].SourceTimestamp.Equal(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected %#v %v", out, e)
	}
}
func TestOPCUAReadFailureInvalidatesWithoutDetail(t *testing.T) {
	resource, _ := json.Marshal(map[string]string{"url": "opc.tcp://127.0.0.1:4840"})
	r, _ := resolver.New(&loader.Loaded{Index: loader.Index{Resources: map[string]json.RawMessage{"site-resource://site/ua": resource}, Secrets: map[string]string{}}, IndexDir: t.TempDir()})
	f := &fakeUA{err: errors.New("secret-value")}
	d := NewOPCUA(r)
	d.newClient = func(string) (uaClient, error) { return f, nil }
	_, e := d.ReadBatch(context.Background(), loader.Connection{ConnectionID: "c"}, loader.BindingConnection{ResourceRef: "site-resource://site/ua"}, []loader.Mapping{{DatapointID: "a", Address: json.RawMessage(`{"nodeId":"i=1"}`)}})
	if e == nil || f.closed != 1 || len(d.clients) != 0 || e.Error() == "secret-value" {
		t.Fatal("must invalidate and redact adapter error")
	}
}
