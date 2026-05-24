package service

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRejectDangerousBuiltinSQL(t *testing.T) {
	cases := []string{
		"drop database if_data",
		"create database x",
		"alter system set work_mem='1GB'",
		"copy t to program 'cat /etc/passwd'",
		"select * from pg_catalog.pg_tables",
		"select * from information_schema.tables",
		"select * from p_other_app.orders",
	}
	for _, sqlText := range cases {
		if err := validateBuiltinSQL(sqlText); err == nil {
			t.Fatalf("expected sql to be rejected: %s", sqlText)
		}
	}
}

func TestAllowBuiltinProjectSQL(t *testing.T) {
	cases := []string{
		"create table orders(id int primary key)",
		"insert into orders(id) values ($1)",
		"select * from orders limit 10",
		"alter table orders add column name text",
		"create index idx_orders_name on orders(name)",
	}
	for _, sqlText := range cases {
		if err := validateBuiltinSQL(sqlText); err != nil {
			t.Fatalf("expected sql to be allowed %q: %v", sqlText, err)
		}
	}
}

func TestBuiltinRealtimeUsesProjectPrefix(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis failed: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer client.Close()

	svc := NewBuiltinRuntimeService(BuiltinRuntimeOptions{
		RealtimeClient:    client,
		RealtimeKeyPrefix: "ifdev",
	})
	projectID := "11111111-1111-1111-1111-111111111111"
	if _, err := svc.SetRealtimeKey(context.Background(), projectID, BuiltinRealtimeSetInput{
		Key:        "current/temp",
		Value:      "36.5",
		TtlSeconds: 30,
		RuntimeKey: "rt_test",
	}); err != nil {
		t.Fatalf("set realtime key failed: %v", err)
	}

	fullKey := "ifdev:11111111-1111-1111-1111-111111111111:rt_test:current/temp"
	if !mr.Exists(fullKey) {
		t.Fatalf("expected key %s", fullKey)
	}
	value, err := svc.GetRealtimeKey(context.Background(), projectID, "rt_test", "current/temp")
	if err != nil {
		t.Fatalf("get realtime key failed: %v", err)
	}
	if value.Value != "36.5" {
		t.Fatalf("unexpected value: %#v", value.Value)
	}
}

func TestBuiltinMessageTopicPrefix(t *testing.T) {
	publisher := &fakeBuiltinMessagePublisher{}
	svc := NewBuiltinRuntimeService(BuiltinRuntimeOptions{
		MessagePublisher:   publisher,
		MessageTopicPrefix: "ifdev",
	})
	projectID := "11111111-1111-1111-1111-111111111111"

	_, err := svc.PublishMessage(context.Background(), projectID, BuiltinMessagePublishInput{
		Topic:      "device/dev-1/telemetry",
		Payload:    map[string]any{"value": 1},
		QOS:        0,
		RuntimeKey: "msg_test",
	})
	if err != nil {
		t.Fatalf("publish failed: %v", err)
	}
	if publisher.topic != "ifdev/11111111-1111-1111-1111-111111111111/msg_test/device/dev-1/telemetry" {
		t.Fatalf("unexpected topic %q", publisher.topic)
	}
}

func TestBuiltinMessagePublishNotifiesTopicSubscribers(t *testing.T) {
	publisher := &fakeBuiltinMessagePublisher{}
	svc := NewBuiltinRuntimeService(BuiltinRuntimeOptions{
		MessagePublisher:   publisher,
		MessageTopicPrefix: "ifdev",
	})
	received := make(chan BuiltinMessageEvent, 1)
	cancel := svc.SubscribeMessageTopic("topic-1", func(event BuiltinMessageEvent) {
		received <- event
	})
	defer cancel()

	svc.messageHub.publish("topic-1", BuiltinMessageEvent{
		ConnectionID: "conn-1",
		RuntimeKey:   "msg_test",
		Topic:        "device/1/telemetry",
		Payload:      map[string]any{"value": 1},
		QOS:          0,
		Timestamp:    time.Now().UTC(),
	})

	select {
	case event := <-received:
		if event.Topic != "device/1/telemetry" || event.RuntimeKey != "msg_test" {
			t.Fatalf("unexpected event: %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("expected builtin message event")
	}
}

func TestDeriveBuiltinRealtimeKeyUsesRuntimeKey(t *testing.T) {
	key, err := deriveBuiltinRealtimeKey("ifdev", "project-1", "rt_a1b2c3", "device/line1/status")
	if err != nil {
		t.Fatalf("derive realtime key failed: %v", err)
	}
	expected := "ifdev:project-1:rt_a1b2c3:device/line1/status"
	if key != expected {
		t.Fatalf("expected %q, got %q", expected, key)
	}
}

func TestDeriveBuiltinMessageTopicUsesRuntimeKey(t *testing.T) {
	topic, err := deriveBuiltinMessageTopic("ifdev", "project-1", "msg_a1b2c3", "device/1/telemetry")
	if err != nil {
		t.Fatalf("derive message topic failed: %v", err)
	}
	expected := "ifdev/project-1/msg_a1b2c3/device/1/telemetry"
	if topic != expected {
		t.Fatalf("expected %q, got %q", expected, topic)
	}
}

type fakeBuiltinMessagePublisher struct {
	topic   string
	payload []byte
}

func (p *fakeBuiltinMessagePublisher) Publish(ctx context.Context, topic string, payload []byte, qos byte) error {
	p.topic = topic
	p.payload = append([]byte(nil), payload...)
	return nil
}
