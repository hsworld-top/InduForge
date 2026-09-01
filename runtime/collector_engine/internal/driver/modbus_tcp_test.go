package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/indu-forge/collector-engine/internal/loader"
	"github.com/indu-forge/collector-engine/internal/resolver"
	"net"
	"testing"
	"time"
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

func TestModbusTCPBatchesContiguousHoldingRegisters(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	count := make(chan int, 1)
	go func() {
		c, _ := ln.Accept()
		defer c.Close()
		req := make([]byte, 12)
		_, _ = c.Read(req)
		if req[7] != 3 || req[10] != 0 || req[11] != 3 {
			t.Errorf("expected one 3-register holding request: %v", req)
			return
		}
		count <- 1
		_, _ = c.Write([]byte{req[0], req[1], 0, 0, 0, 9, req[6], 3, 6, 0, 1, 0, 2, 0, 3})
	}()
	host, port, _ := net.SplitHostPort(ln.Addr().String())
	raw, _ := json.Marshal(map[string]any{"host": host, "port": mustPort(t, port)})
	r, _ := resolver.New(&loader.Loaded{Index: loader.Index{Resources: map[string]json.RawMessage{"site-resource://site/modbus": raw}, Secrets: map[string]string{}}, IndexDir: t.TempDir()})
	d := NewModbusTCP(r)
	c := loader.Connection{ConnectionID: "c"}
	b := loader.BindingConnection{ResourceRef: "site-resource://site/modbus"}
	mappings := []loader.Mapping{point("a", 0), point("b", 1), point("c", 2)}
	results, err := d.ReadBatch(context.Background(), c, b, mappings)
	if err != nil || string(results["a"].Value) != "1" || string(results["c"].Value) != "3" {
		t.Fatalf("batch failed %#v %v", results, err)
	}
	select {
	case <-count:
	case <-time.After(time.Second):
		t.Fatal("expected exactly one request")
	}
}
func point(id string, address int) loader.Mapping {
	return loader.Mapping{DatapointID: id, DataType: "uint16", ElementCount: 1, Address: json.RawMessage(fmt.Sprintf(`{"station":1,"area":"holdingRegister","address":%d}`, address))}
}
func mustPort(t *testing.T, s string) int {
	t.Helper()
	var p uint16
	if _, e := fmt.Sscan(s, &p); e != nil {
		t.Fatal(e)
	}
	return int(p)
}
