package service

import (
	"strings"
	"testing"
)

func TestNormalizeWritableRealtimeKey(t *testing.T) {
	valid := []string{"device:line1:status", "产线一:温度_1", "device-1"}
	for _, key := range valid {
		if got, err := normalizeWritableRealtimeKey(key); err != nil || got != key {
			t.Fatalf("合法 Key %q 不应被拒绝: got=%q err=%v", key, got, err)
		}
	}

	invalid := []string{
		" device:status",
		"device:status ",
		"device::status",
		":device",
		"device:",
		"device.status",
		"device/status",
		"device status",
		"device\nstatus",
		strings.Repeat("中", 86), // UTF-8 共 258 字节。
	}
	for _, key := range invalid {
		if _, err := normalizeWritableRealtimeKey(key); err == nil {
			t.Fatalf("非法 Key %q 应被拒绝", key)
		}
	}

	if got, err := normalizeRealtimeKey("legacy.key/with space"); err != nil || got != "legacy.key/with space" {
		t.Fatalf("外部 Redis 既有特殊 Key 仍应允许读取: got=%q err=%v", got, err)
	}
}

func TestSQLAndWorkbenchNameValidation(t *testing.T) {
	if _, err := normalizeQueryName("查询_1"); err != nil {
		t.Fatalf("正常查询名称不应被拒绝: %v", err)
	}
	for _, name := range []string{" 查询_1", "查询_1\n", strings.Repeat("a", 101)} {
		if _, err := normalizeQueryName(name); err == nil {
			t.Fatalf("非法查询名称 %q 应被拒绝", name)
		}
	}
	if _, err := normalizeWorkbenchGroupName("SQL 模型"); err != nil {
		t.Fatalf("正常分组名称不应被拒绝: %v", err)
	}
	if _, err := normalizeWorkbenchGroupName("SQL\t模型"); err == nil {
		t.Fatal("包含控制字符的分组名称应被拒绝")
	}
}

func TestNormalizeSQLTextKeepsSQLSyntaxAndLimitsSize(t *testing.T) {
	sqlText := "CREATE FUNCTION f() RETURNS text AS $$ BEGIN RETURN 'a;b'; END $$ LANGUAGE plpgsql;"
	if got, err := normalizeSQLText(sqlText); err != nil || got != sqlText {
		t.Fatalf("SQL 特殊语法不应被过滤: got=%q err=%v", got, err)
	}
	if _, err := normalizeSQLText(strings.Repeat("x", maxSavedSQLBytes+1)); err == nil {
		t.Fatal("超过 1 MiB 的 SQL 应被拒绝")
	}
}

func TestNormalizeTableDesignIdentifierHasPortableLengthLimit(t *testing.T) {
	if _, err := normalizeTableDesignIdentifier(strings.Repeat("a", 63), "表名"); err != nil {
		t.Fatalf("63 字符标识符应允许: %v", err)
	}
	if _, err := normalizeTableDesignIdentifier(strings.Repeat("a", 64), "表名"); err == nil {
		t.Fatal("超过 63 字符的表名应被拒绝")
	}
}
