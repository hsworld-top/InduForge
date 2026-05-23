package service

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gorilla/websocket"
)

func TestConnectionProtocolTest_HTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("X-Test"); got != "ok" {
			t.Fatalf("expected header X-Test=ok, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	result, err := testHTTPConnection(context.Background(), map[string]any{
		"baseUrl":      server.URL,
		"method":       "post",
		"headers":      map[string]any{"X-Test": "ok"},
		"bodyTemplate": map[string]any{"probe": true},
		"timeoutMs":    1000,
	})
	if err != nil {
		t.Fatalf("HTTP test failed: %v", err)
	}
	if !result.Connected || result.Type != "http" {
		t.Fatalf("unexpected HTTP result: %#v", result)
	}
}

func TestConnectionProtocolTest_WebSocket(t *testing.T) {
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("upgrade failed: %v", err)
		}
		_ = conn.Close()
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[len("http"):]
	result, err := testWebSocketConnection(context.Background(), map[string]any{
		"url": wsURL,
	})
	if err != nil {
		t.Fatalf("WebSocket test failed: %v", err)
	}
	if !result.Connected || result.Type != "websocket" {
		t.Fatalf("unexpected WebSocket result: %#v", result)
	}
}

func TestConnectionProtocolTest_Redis(t *testing.T) {
	redisServer := miniredis.RunT(t)
	redisServer.RequireAuth("secret")

	result, err := testRedisConnection(context.Background(), map[string]any{
		"address":  redisServer.Addr(),
		"password": "secret",
		"db":       0,
		"mode":     "standalone",
	})
	if err != nil {
		t.Fatalf("Redis test failed: %v", err)
	}
	if !result.Connected || result.Type != "redis" {
		t.Fatalf("unexpected Redis result: %#v", result)
	}
}

func TestConnectionProtocolTest_HTTPRejectsBadStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}))
	defer server.Close()

	if _, err := testHTTPConnection(context.Background(), map[string]any{
		"baseUrl": server.URL,
		"method":  "GET",
	}); err == nil {
		t.Fatal("expected HTTP bad status to fail")
	}
}

func TestConnectionProtocolTest_OPCUATcpProbe(t *testing.T) {
	listener := mustTCPListener(t)
	defer listener.Close()
	go acceptAndClose(listener)

	result, err := testOPCUAConnection(context.Background(), map[string]any{
		"endpoint": "opc.tcp://" + listener.Addr().String(),
	})
	if err != nil {
		t.Fatalf("OPC UA test failed: %v", err)
	}
	if !result.Connected || result.Type != "opcua" {
		t.Fatalf("unexpected OPC UA result: %#v", result)
	}
}

func TestConnectionProtocolTest_ModbusTCPProbe(t *testing.T) {
	listener := mustTCPListener(t)
	defer listener.Close()
	go acceptAndClose(listener)
	host, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("split listener address: %v", err)
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil {
		t.Fatalf("parse listener port: %v", err)
	}

	result, err := testModbusConnection(context.Background(), map[string]any{
		"mode": "tcp",
		"host": host,
		"port": portNumber,
	})
	if err != nil {
		t.Fatalf("Modbus TCP test failed: %v", err)
	}
	if !result.Connected || result.Type != "modbus" {
		t.Fatalf("unexpected Modbus result: %#v", result)
	}
}

func TestConnectionProtocolTest_ModbusRTUConfig(t *testing.T) {
	result, err := testModbusConnection(context.Background(), map[string]any{
		"mode": "rtu",
		"serialConfig": map[string]any{
			"port": "COM3",
		},
	})
	if err != nil {
		t.Fatalf("Modbus RTU config test failed: %v", err)
	}
	if !result.Connected || result.Type != "modbus" {
		t.Fatalf("unexpected Modbus RTU result: %#v", result)
	}
}

func mustTCPListener(t *testing.T) net.Listener {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen tcp: %v", err)
	}
	return listener
}

func acceptAndClose(listener net.Listener) {
	conn, err := listener.Accept()
	if err == nil {
		_ = conn.Close()
	}
}
