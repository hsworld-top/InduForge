package service

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gorilla/websocket"
	"github.com/segmentio/kafka-go"

	"github.com/indu-forge/data_service/internal/repository"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestHTTPPreviewAdapter_ReturnsJSONSampleAndDiagnostics(t *testing.T) {
	adapter := HTTPPreviewAdapter{Client: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", req.Method)
		}
		if req.Header.Get("X-Token") != "secret" {
			t.Fatalf("expected custom header")
		}
		return &http.Response{
			StatusCode: http.StatusCreated,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`[{"name":"a"},{"name":"b"}]`)),
			Request:    req,
		}, nil
	})}}

	result, err := adapter.Preview(context.Background(), ProtocolPreviewAdapterInput{
		Connection: repository.ProtocolPreviewConnectionRecord{
			ID:   "conn-http",
			Type: "http",
			Config: map[string]any{
				"baseUrl": "http://example.test/data",
				"method":  "POST",
				"headers": map[string]any{
					"X-Token": "secret",
				},
				"bodyTemplate": map[string]any{"q": "ok"},
			},
		},
		Limit:   1,
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("http preview failed: %v", err)
	}
	if result.Protocol != "http" || len(result.Samples) != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.Diagnostics["statusCode"] != http.StatusCreated {
		t.Fatalf("expected statusCode diagnostic, got %#v", result.Diagnostics)
	}
}

func TestRedisPreviewAdapter_ReadsStringAndHashSamples(t *testing.T) {
	server, err := miniredis.Run()
	if err != nil {
		t.Skipf("当前环境无法启动本地 Redis 测试监听: %v", err)
	}
	defer server.Close()
	server.Set("factory:string", "ok")
	server.HSet("factory:hash", "field", "value")

	result, err := RedisPreviewAdapter{}.Preview(context.Background(), ProtocolPreviewAdapterInput{
		Connection: repository.ProtocolPreviewConnectionRecord{
			ID:   "conn-redis",
			Type: "redis",
			Config: map[string]any{
				"address":    server.Addr(),
				"keyPattern": "factory:*",
				"mode":       "standalone",
			},
		},
		Limit:   10,
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("redis preview failed: %v", err)
	}
	if len(result.Samples) != 2 {
		t.Fatalf("expected 2 samples, got %#v", result.Samples)
	}
}

func TestWebSocketPreviewAdapter_SendsSubscribeMessageAndReadsSample(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("当前环境无法启动本地 WebSocket 测试监听: %v", err)
	}
	upgrader := websocket.Upgrader{}
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, upgradeErr := upgrader.Upgrade(w, r, nil)
		if upgradeErr != nil {
			t.Errorf("upgrade failed: %v", upgradeErr)
			return
		}
		defer conn.Close()
		_, message, readErr := conn.ReadMessage()
		if readErr != nil {
			t.Errorf("read subscribe failed: %v", readErr)
			return
		}
		if string(message) != "subscribe" {
			t.Errorf("expected subscribe message, got %q", string(message))
		}
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"value":42}`))
	}))
	server.Listener = listener
	server.Start()
	defer server.Close()

	result, err := WebSocketPreviewAdapter{}.Preview(context.Background(), ProtocolPreviewAdapterInput{
		Connection: repository.ProtocolPreviewConnectionRecord{
			ID:     "conn-ws",
			Type:   "websocket",
			Config: map[string]any{"url": "ws" + strings.TrimPrefix(server.URL, "http")},
		},
		Limit:   1,
		Timeout: time.Second,
		Options: map[string]any{"subscribeMessage": "subscribe"},
	})
	if err != nil {
		t.Fatalf("websocket preview failed: %v", err)
	}
	if len(result.Samples) != 1 {
		t.Fatalf("expected one websocket sample, got %#v", result.Samples)
	}
}

type fakeKafkaPreviewReader struct {
	messages []kafka.Message
	index    int
	closed   bool
}

func (r *fakeKafkaPreviewReader) ReadMessage(ctx context.Context) (kafka.Message, error) {
	if r.index >= len(r.messages) {
		<-ctx.Done()
		return kafka.Message{}, ctx.Err()
	}
	message := r.messages[r.index]
	r.index++
	return message, nil
}

func (r *fakeKafkaPreviewReader) Close() error {
	r.closed = true
	return nil
}

type fakeKafkaProbeConn struct {
	closed bool
}

func (c *fakeKafkaProbeConn) Close() error {
	c.closed = true
	return nil
}

func TestKafkaPreviewAdapter_ProbeDoesNotRequireTopic(t *testing.T) {
	probeConn := &fakeKafkaProbeConn{}
	var capturedNetwork string
	var capturedAddress string
	adapter := KafkaPreviewAdapter{
		ProbeFactory: func(ctx context.Context, network, address string) (kafkaProbeConn, error) {
			capturedNetwork = network
			capturedAddress = address
			return probeConn, nil
		},
	}

	result, err := adapter.Preview(context.Background(), ProtocolPreviewAdapterInput{
		Connection: repository.ProtocolPreviewConnectionRecord{
			ID:   "conn-kafka",
			Type: "kafka",
			Config: map[string]any{
				"brokers": "127.0.0.1:9092",
			},
		},
		Limit:   1,
		Timeout: time.Second,
		Options: map[string]any{
			"probe": true,
		},
	})
	if err != nil {
		t.Fatalf("kafka probe failed: %v", err)
	}
	if capturedNetwork != "tcp" || capturedAddress != "127.0.0.1:9092" {
		t.Fatalf("unexpected probe target: %s %s", capturedNetwork, capturedAddress)
	}
	if !probeConn.closed {
		t.Fatal("expected probe connection to be closed")
	}
	if result.Status != "ok" || result.Diagnostics["probe"] != true {
		t.Fatalf("unexpected probe result: %#v", result)
	}
}

func TestKafkaPreviewAdapter_UsesTemporaryReaderAndReturnsSamples(t *testing.T) {
	reader := &fakeKafkaPreviewReader{
		messages: []kafka.Message{
			{Topic: "factory.events", Partition: 0, Offset: 10, Key: []byte("k1"), Value: []byte(`{"temperature":32}`), Time: time.Unix(100, 0)},
			{Topic: "factory.events", Partition: 0, Offset: 11, Value: []byte("plain-text"), Time: time.Unix(101, 0)},
		},
	}
	var capturedConfig kafka.ReaderConfig
	adapter := KafkaPreviewAdapter{
		ReaderFactory: func(config kafka.ReaderConfig) kafkaPreviewReader {
			capturedConfig = config
			return reader
		},
	}

	result, err := adapter.Preview(context.Background(), ProtocolPreviewAdapterInput{
		Connection: repository.ProtocolPreviewConnectionRecord{
			ID:   "conn-kafka",
			Type: "kafka",
			Config: map[string]any{
				"brokers":       "127.0.0.1:9092,127.0.0.1:9093",
				"topic":         "factory.events",
				"consumerGroup": "runtime-group",
				"startPosition": "latest",
			},
		},
		Limit:   2,
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("kafka preview failed: %v", err)
	}
	if capturedConfig.GroupID == "runtime-group" || !strings.HasPrefix(capturedConfig.GroupID, "data-service-preview-") {
		t.Fatalf("preview reader must use a temporary consumer group, got %q", capturedConfig.GroupID)
	}
	if capturedConfig.StartOffset != kafka.LastOffset {
		t.Fatalf("expected latest start offset, got %d", capturedConfig.StartOffset)
	}
	if len(capturedConfig.Brokers) != 2 || capturedConfig.Topic != "factory.events" {
		t.Fatalf("unexpected kafka reader config: %#v", capturedConfig)
	}
	if !reader.closed {
		t.Fatal("expected kafka reader to be closed after preview")
	}
	if result.Protocol != "kafka" || len(result.Samples) != 2 {
		t.Fatalf("unexpected kafka result: %#v", result)
	}
	if result.Diagnostics["topic"] != "factory.events" {
		t.Fatalf("expected topic diagnostic, got %#v", result.Diagnostics)
	}
}

func TestKafkaPreviewAdapter_TimesOutWithoutSamplesAsEmptyResult(t *testing.T) {
	reader := &fakeKafkaPreviewReader{}
	adapter := KafkaPreviewAdapter{ReaderFactory: func(kafka.ReaderConfig) kafkaPreviewReader { return reader }}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	result, err := adapter.Preview(ctx, ProtocolPreviewAdapterInput{
		Connection: repository.ProtocolPreviewConnectionRecord{ID: "conn-kafka", Type: "kafka", Config: map[string]any{
			"brokers": "127.0.0.1:9092", "topic": "factory.events", "startPosition": "latest",
		}},
		Limit: 1, Timeout: 10 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("empty Kafka preview should not fail on timeout: %v", err)
	}
	if result.Status != "ok" || len(result.Samples) != 0 {
		t.Fatalf("unexpected empty preview result: %#v", result)
	}
}

func TestKafkaPreviewAdapter_SupportsSinglePartitionOffsetAndHeaders(t *testing.T) {
	reader := &fakeKafkaPreviewReader{
		messages: []kafka.Message{
			{
				Topic:     "factory.override",
				Partition: 2,
				Offset:    42,
				Value:     []byte("plain-text"),
				Headers: []kafka.Header{
					{Key: "access-token", Value: []byte("secret")},
					{Key: "x-source", Value: []byte("bench")},
				},
				Time: time.Unix(102, 0),
			},
		},
	}
	var capturedConfig kafka.ReaderConfig
	adapter := KafkaPreviewAdapter{
		ReaderFactory: func(config kafka.ReaderConfig) kafkaPreviewReader {
			capturedConfig = config
			return reader
		},
	}

	result, err := adapter.Preview(context.Background(), ProtocolPreviewAdapterInput{
		Connection: repository.ProtocolPreviewConnectionRecord{
			ID:   "conn-kafka",
			Type: "kafka",
			Config: map[string]any{
				"brokers":       "127.0.0.1:9092",
				"topic":         "factory.default",
				"startPosition": "latest",
			},
		},
		Limit:   1,
		Timeout: time.Second,
		Options: map[string]any{
			"topic":         "factory.override",
			"partitionMode": "single",
			"partition":     2,
			"startPosition": "offset",
			"offset":        int64(42),
			"decode":        "string",
		},
	})
	if err != nil {
		t.Fatalf("kafka preview failed: %v", err)
	}
	if capturedConfig.GroupID != "" || capturedConfig.Partition != 2 || capturedConfig.StartOffset != 42 {
		t.Fatalf("expected single partition reader without group, got %#v", capturedConfig)
	}
	if capturedConfig.Topic != "factory.override" {
		t.Fatalf("expected topic override, got %q", capturedConfig.Topic)
	}
	sample, ok := result.Samples[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected sample: %#v", result.Samples[0])
	}
	if sample["value"] != "plain-text" {
		t.Fatalf("expected string decoded value, got %#v", sample["value"])
	}
	headers := sample["headers"].(map[string]string)
	if headers["access-token"] != "******" || headers["x-source"] != "bench" {
		t.Fatalf("expected sanitized headers, got %#v", headers)
	}
}
