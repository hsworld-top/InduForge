package service

import (
	"context"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/indu-forge/data_service/internal/repository"
	"github.com/redis/go-redis/v9"
)

func TestRealtimeStoreGroupsUseSearchResultBeforeGroupFilter(t *testing.T) {
	items := []RealtimeStoreKeySummary{
		{Key: "line1:temp"},
		{Key: "line2:temp"},
		{Key: "plain"},
	}
	searched := filterRealtimeKeys(items, "temp", "")
	groups := buildRealtimeGroups(searched)
	filtered := filterRealtimeKeys(searched, "", "line1")

	if len(filtered) != 1 || filtered[0].Key != "line1:temp" {
		t.Fatalf("unexpected filtered keys: %#v", filtered)
	}
	names := make([]string, 0, len(groups))
	for _, group := range groups {
		names = append(names, group.Name)
	}
	if strings.Join(names, ",") != "line1,line2" {
		t.Fatalf("groups should keep searched full set, got %#v", groups)
	}
}

func TestWriteRealtimeRedisValueRejectsEmptyCollectionBeforeDeletingOldKey(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()

	ctx := context.Background()
	if err := client.HSet(ctx, "device:hash", map[string]string{"old": "value"}).Err(); err != nil {
		t.Fatalf("seed hash failed: %v", err)
	}
	if err := validateRealtimeRedisValue("hash", []any{}); err == nil {
		t.Fatal("expected empty hash payload to fail validation")
	}
	if got, err := client.HGet(ctx, "device:hash", "old").Result(); err != nil || got != "value" {
		t.Fatalf("old hash should remain untouched, got %q err=%v", got, err)
	}
}

func TestSaveRedisKeyReplacesCollectionInTransaction(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()

	ctx := context.Background()
	if err := client.HSet(ctx, "device:hash", map[string]string{"old": "value"}).Err(); err != nil {
		t.Fatalf("seed hash failed: %v", err)
	}
	svc := &RealtimeStoreService{}
	connection := &repository.ConnectionRecord{Config: map[string]any{"address": server.Addr()}}
	value := []any{map[string]any{"key": "next", "value": "updated"}}
	if err := svc.saveRedisKey(ctx, connection, "device:hash", "hash", value, 30); err != nil {
		t.Fatalf("save redis key failed: %v", err)
	}
	if exists, err := client.HExists(ctx, "device:hash", "old").Result(); err != nil || exists {
		t.Fatalf("old hash field should be removed, exists=%v err=%v", exists, err)
	}
	if got, err := client.HGet(ctx, "device:hash", "next").Result(); err != nil || got != "updated" {
		t.Fatalf("new hash field mismatch, got=%q err=%v", got, err)
	}
	if ttl, err := client.TTL(ctx, "device:hash").Result(); err != nil || ttl <= 0 {
		t.Fatalf("expected ttl to be applied, ttl=%v err=%v", ttl, err)
	}
}

func TestRealtimeRenameNXDoesNotOverwriteTarget(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()

	ctx := context.Background()
	if err := client.Set(ctx, "source", "a", 0).Err(); err != nil {
		t.Fatalf("seed source failed: %v", err)
	}
	if err := client.Set(ctx, "target", "b", 0).Err(); err != nil {
		t.Fatalf("seed target failed: %v", err)
	}
	renamed, err := client.RenameNX(ctx, "source", "target").Result()
	if err != nil {
		t.Fatalf("renamenx failed: %v", err)
	}
	if renamed {
		t.Fatal("expected renamenx to refuse existing target")
	}
	if got, _ := client.Get(ctx, "source").Result(); got != "a" {
		t.Fatalf("source should remain, got %q", got)
	}
	if got, _ := client.Get(ctx, "target").Result(); got != "b" {
		t.Fatalf("target should remain, got %q", got)
	}
}

func TestRealtimeRuntimeKeyFallsBackToConnectionID(t *testing.T) {
	svc := &RealtimeStoreService{}
	connection := &repository.ConnectionRecord{ID: "conn-1", Config: map[string]any{}}
	if got := svc.runtimeKey(connection); got != "conn-1" {
		t.Fatalf("expected fallback runtime key, got %q", got)
	}
}
