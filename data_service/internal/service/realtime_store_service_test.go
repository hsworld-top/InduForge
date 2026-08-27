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

func TestApplyRealtimeValueMetadataRestoresLogicalType(t *testing.T) {
	value := &RealtimeStoreValue{Type: "string"}
	record := &repository.RealtimeKeyRecord{RedisType: "hash", ValueType: "object"}

	applyRealtimeValueMetadata(value, record)

	if value.Type != "hash" {
		t.Fatalf("expected logical type hash, got %q", value.Type)
	}
}

func TestNormalizeBuiltinRealtimeValueConvertsHashRowsToObject(t *testing.T) {
	value, err := normalizeBuiltinRealtimeValue("hash", []any{
		map[string]any{"key": "status", "value": "ok"},
	})
	if err != nil {
		t.Fatalf("normalize builtin hash failed: %v", err)
	}
	object, ok := value.(map[string]string)
	if !ok {
		t.Fatalf("expected hash object, got %#v", value)
	}
	if object["status"] != "ok" {
		t.Fatalf("unexpected hash value: %#v", object)
	}
}

func TestBuildRealtimeDataPointPathIncludesConnectionAndKeyNamespace(t *testing.T) {
	tests := []struct {
		name           string
		provider       string
		connectionName string
		key            string
		want           string
	}{
		{name: "redis", provider: "redis", connectionName: "生产 Redis", key: "device:line1:status", want: "redis.生产_Redis.device.line1.status"},
		{name: "builtin", provider: "builtin", connectionName: "IF实时库", key: "device/line1/temperature", want: "realtime.IF实时库.device.line1.temperature"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := buildRealtimeDataPointPath(test.provider, test.connectionName, test.key); got != test.want {
				t.Fatalf("unexpected realtime datapoint path: got %q want %q", got, test.want)
			}
		})
	}
}

func TestRealtimeKeyDisplayNameUsesLastNamespaceSegment(t *testing.T) {
	if got := realtimeKeyDisplayName("device:line1:status"); got != "status" {
		t.Fatalf("unexpected realtime datapoint display name: %q", got)
	}
}

func TestInferRealtimeDataPointType(t *testing.T) {
	tests := []struct {
		name      string
		redisType string
		valueType string
		value     any
		want      string
	}{
		{name: "plain string", redisType: "string", valueType: "text", value: "online", want: "string"},
		{name: "json object string", redisType: "string", valueType: "json", value: `{"online":true}`, want: "object"},
		{name: "json array string", redisType: "string", valueType: "json", value: `[1,2]`, want: "array"},
		{name: "hash", redisType: "hash", value: map[string]string{"status": "ok"}, want: "object"},
		{name: "list", redisType: "list", value: []string{"a"}, want: "array"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := inferRealtimeDataPointType("", test.redisType, test.valueType, test.value); got != test.want {
				t.Fatalf("unexpected inferred type: got %q want %q", got, test.want)
			}
		})
	}
}

func TestNormalizeRealtimeStoredValueTypeKeepsStringEditorMode(t *testing.T) {
	if got := normalizeRealtimeStoredValueType("string", "json", `{}`); got != "json" {
		t.Fatalf("expected json editor mode, got %q", got)
	}
	if got := normalizeRealtimeStoredValueType("string", "text", "online"); got != "text" {
		t.Fatalf("expected text editor mode, got %q", got)
	}
	if got := normalizeRealtimeStoredValueType("hash", "json", map[string]any{"status": "ok"}); got != "object" {
		t.Fatalf("expected collection metadata type object, got %q", got)
	}
}

func TestBuildRedisKeyValueDecodesJSONStringForObjectDataPoint(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()
	ctx := context.Background()
	if err := client.Set(ctx, "device:status", `{"online":true}`, 0).Err(); err != nil {
		t.Fatalf("seed redis json string failed: %v", err)
	}

	svc := &DataPointService{}
	value, err := svc.buildRedisKeyValue(
		ctx,
		&repository.ConnectionRecord{Config: map[string]any{"address": server.Addr()}},
		"device:status",
		map[string]any{"valueType": "json"},
		DataPointValue{},
	)
	if err != nil {
		t.Fatalf("read redis json string failed: %v", err)
	}
	object, ok := value.Value.(map[string]any)
	if !ok || object["online"] != true {
		t.Fatalf("expected decoded json object, got %#v", value.Value)
	}
}
