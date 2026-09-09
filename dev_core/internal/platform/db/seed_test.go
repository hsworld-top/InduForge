package db

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"strings"
	"testing"
)

type seedTestPool struct{ tx *seedTestTx }

func (p seedTestPool) Begin(context.Context) (pgx.Tx, error) { return p.tx, nil }

type seedTestTx struct {
	pgx.Tx
	count     int64
	queries   []string
	committed bool
	failUser  bool
}

func (t *seedTestTx) Exec(_ context.Context, q string, _ ...any) (pgconn.CommandTag, error) {
	t.queries = append(t.queries, q)
	if t.failUser && strings.Contains(q, "INSERT INTO users") {
		return pgconn.CommandTag{}, errors.New("write failed")
	}
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}
func (t *seedTestTx) QueryRow(_ context.Context, q string, _ ...any) pgx.Row {
	t.queries = append(t.queries, q)
	return seedCountRow(t.count)
}
func (t *seedTestTx) Commit(context.Context) error   { t.committed = true; return nil }
func (t *seedTestTx) Rollback(context.Context) error { return nil }

type seedCountRow int64

func (r seedCountRow) Scan(dest ...any) error { *dest[0].(*int64) = int64(r); return nil }
func TestInitialDataTransaction(t *testing.T) {
	for _, tc := range []struct {
		name       string
		count      int64
		fail       bool
		wantWrites int
		wantError  bool
	}{{"empty", 0, false, 4, false}, {"existing", 1, false, 2, false}, {"rollback", 0, true, 4, true}} {
		t.Run(tc.name, func(t *testing.T) {
			tx := &seedTestTx{count: tc.count, failUser: tc.fail}
			err := EnsureInitialData(context.Background(), seedTestPool{tx}, SeedConfig{TenantID: "tenant", SuperAdminUserID: "platform", SuperAdminPasswordHash: "hash"})
			if (err != nil) != tc.wantError || len(tx.queries) != tc.wantWrites || tx.committed == tc.wantError {
				t.Fatalf("err=%v queries=%v committed=%v", err, tx.queries, tx.committed)
			}
			if !strings.Contains(tx.queries[0], "pg_advisory_xact_lock") || !strings.Contains(tx.queries[1], "count(*)") {
				t.Fatal("empty check must occur under transaction lock")
			}
		})
	}
}
func TestEmptyInstallRequiresConfiguredPassword(t *testing.T) {
	tx := &seedTestTx{}
	if err := EnsureInitialData(context.Background(), seedTestPool{tx}, SeedConfig{}); err == nil || tx.committed || len(tx.queries) != 2 {
		t.Fatal("unconfigured install wrote data")
	}
}
