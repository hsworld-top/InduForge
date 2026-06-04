package service

import (
	"context"
	"testing"

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
