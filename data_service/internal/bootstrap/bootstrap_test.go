package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/indu-forge/data_service/internal/config"
)

func TestEnsureReady_RunsMigrationsWhenDatabaseExists(t *testing.T) {
	t.Helper()

	targetPool := &fakePool{name: "target"}
	migrator := &fakeMigrator{}

	bootstrapper := NewRuntimeBootstrapper(
		withPoolOpener(func(_ context.Context, databaseURL, searchPath string) (poolHandle, error) {
			if databaseURL != "postgres://demo/exists_db" {
				t.Fatalf("unexpected database url: %s", databaseURL)
			}
			if searchPath != "public" {
				t.Fatalf("unexpected search path: %s", searchPath)
			}
			return targetPool, nil
		}),
		withMigratorFactory(func(pool poolHandle) (migrationRunner, error) {
			if pool != targetPool {
				t.Fatalf("unexpected pool passed to migrator")
			}
			return migrator, nil
		}),
	)

	err := bootstrapper.EnsureReady(context.Background(), config.Config{
		DatabaseURL:        "postgres://demo/exists_db",
		DatabaseSearchPath: "public",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if migrator.upCalls != 1 {
		t.Fatalf("expected migrator.Up to be called once, got %d", migrator.upCalls)
	}
}

func TestEnsureReady_CreatesMissingDatabaseThenRunsMigrations(t *testing.T) {
	t.Helper()

	targetPool := &fakePool{name: "target"}
	adminPool := &fakeAdminPool{}
	migrator := &fakeMigrator{}
	logs := make([]string, 0, 4)
	openCalls := 0
	reset := stubBootstrapLogf(func(format string, args ...any) {
		logs = append(logs, fmt.Sprintf(format, args...))
	})
	t.Cleanup(reset)

	bootstrapper := NewRuntimeBootstrapper(
		withPoolOpener(func(_ context.Context, databaseURL, searchPath string) (poolHandle, error) {
			openCalls++
			if openCalls == 1 {
				return nil, errMissingDatabase
			}
			if databaseURL != "postgres://demo/app_db" {
				t.Fatalf("unexpected database url: %s", databaseURL)
			}
			if searchPath != "app_schema" {
				t.Fatalf("unexpected search path: %s", searchPath)
			}
			return targetPool, nil
		}),
		withAdminPoolOpener(func(_ context.Context, databaseURL string) (adminExecutor, error) {
			if databaseURL != "postgres://demo/postgres" {
				t.Fatalf("unexpected admin database url: %s", databaseURL)
			}
			return adminPool, nil
		}),
		withMigratorFactory(func(pool poolHandle) (migrationRunner, error) {
			if pool != targetPool {
				t.Fatalf("unexpected pool passed to migrator")
			}
			return migrator, nil
		}),
	)

	err := bootstrapper.EnsureReady(context.Background(), config.Config{
		DatabaseURL:        "postgres://demo/app_db",
		DatabaseSearchPath: "app_schema",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if openCalls != 2 {
		t.Fatalf("expected target pool to open twice, got %d", openCalls)
	}
	if len(adminPool.execSQL) != 1 {
		t.Fatalf("expected one create database statement, got %d", len(adminPool.execSQL))
	}
	if !strings.Contains(adminPool.execSQL[0], "CREATE DATABASE") || !strings.Contains(adminPool.execSQL[0], "app_db") {
		t.Fatalf("unexpected create database sql: %s", adminPool.execSQL[0])
	}
	if migrator.upCalls != 1 {
		t.Fatalf("expected migrator.Up to be called once, got %d", migrator.upCalls)
	}
	assertBootstrapLogContains(t, logs, "目标数据库不存在，准备自动创建")
	assertBootstrapLogContains(t, logs, "自动创建数据库成功")
	assertBootstrapLogContains(t, logs, "开始执行数据库迁移")
	assertBootstrapLogContains(t, logs, "数据库迁移完成")
}

func TestEnsureReady_FailsWhenCreateDatabasePermissionDenied(t *testing.T) {
	t.Helper()

	adminPool := &fakeAdminPool{
		execErr: errPermissionDenied,
	}

	bootstrapper := NewRuntimeBootstrapper(
		withPoolOpener(func(_ context.Context, _ string, _ string) (poolHandle, error) {
			return nil, errMissingDatabase
		}),
		withAdminPoolOpener(func(_ context.Context, _ string) (adminExecutor, error) {
			return adminPool, nil
		}),
	)

	err := bootstrapper.EnsureReady(context.Background(), config.Config{
		DatabaseURL: "postgres://demo/forbidden_db",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "CREATE DATABASE 权限") {
		t.Fatalf("expected permission error message, got %v", err)
	}
}

type fakePool struct {
	name   string
	closed bool
}

func (p *fakePool) Close() {
	p.closed = true
}

type fakeAdminPool struct {
	execSQL []string
	execErr error
	closed  bool
}

func (p *fakeAdminPool) Exec(_ context.Context, sql string) error {
	p.execSQL = append(p.execSQL, sql)
	return p.execErr
}

func (p *fakeAdminPool) Close() {
	p.closed = true
}

type fakeMigrator struct {
	upCalls int
}

func (m *fakeMigrator) Up(_ context.Context) error {
	m.upCalls++
	return nil
}

var (
	errMissingDatabase  = errors.New("database does not exist")
	errPermissionDenied = errors.New("permission denied to create database")
)

func assertBootstrapLogContains(t *testing.T, logs []string, expected string) {
	t.Helper()

	for _, logLine := range logs {
		if strings.Contains(logLine, expected) {
			return
		}
	}
	t.Fatalf("expected logs %v to contain %q", logs, expected)
}

func stubBootstrapLogf(logger func(string, ...any)) func() {
	previous := logf
	logf = logger
	return func() {
		logf = previous
	}
}
