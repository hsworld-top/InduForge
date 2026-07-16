package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gorilla/websocket"
)

func TestConnectionProtocolTest_HTTP(t *testing.T) {
	if _, err := testHTTPConnection(context.Background(), map[string]any{}); err == nil {
		t.Fatal("expected HTTP source-level connection test to be disabled")
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
