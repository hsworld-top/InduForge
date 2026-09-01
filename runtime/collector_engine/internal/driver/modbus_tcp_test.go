package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/indu-forge/collector-engine/internal/loader"
	"github.com/indu-forge/collector-engine/internal/resolver"
	"net"
	"testing"
)

func TestModbusTCPReadsHoldingRegisterFromLocalServer(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		c, e := ln.Accept()
		if e != nil {
			return
		}
		defer c.Close()
		req := make([]byte, 12)
		if _, e = c.Read(req); e != nil {
			return
		}
		if req[7] != 3 || req[6] != 7 {
			t.Errorf("unexpected unit/function %d/%d", req[6], req[7])
			return
		}
		response := []byte{req[0], req[1], 0, 0, 0, 5, req[6], 3, 2, 0, 42}
		_, _ = c.Write(response)
	}()
	host, port, _ := net.SplitHostPort(ln.Addr().String())
	raw, _ := json.Marshal(map[string]any{"host": host, "port": mustPort(t, port)})
	r, _ := resolver.New(&loader.Loaded{Index: loader.Index{Resources: map[string]json.RawMessage{"site-resource://site/modbus": raw}, Secrets: map[string]string{}}, IndexDir: t.TempDir()})
	d := NewModbusTCP(r)
	m := loader.Mapping{DatapointID: "22222222-2222-4222-8222-222222222222", DataType: "uint16", ElementCount: 1, Address: json.RawMessage(`{"station":7,"area":"holdingRegister","address":0}`)}
	result, err := d.ReadWithBinding(context.Background(), loader.Connection{ConnectionID: "c"}, loader.BindingConnection{ResourceRef: "site-resource://site/modbus"}, m)
	if err != nil || string(result.Value) != "42" {
		t.Fatalf("result=%s err=%v", result.Value, err)
	}
}
func mustPort(t *testing.T, s string) int {
	t.Helper()
	var p uint16
	if _, e := fmt.Sscan(s, &p); e != nil {
		t.Fatal(e)
	}
	return int(p)
}
