package db

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"strings"
	"testing"
)

type shellTx struct {
	pgx.Tx
	sql   string
	args  []any
	fail  bool
	calls int
}

func (t *shellTx) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	t.sql = sql
	t.args = args
	t.calls++
	if t.fail {
		return pgconn.CommandTag{}, errors.New("write failed")
	}
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}
func TestOrganizationTemplateShell(t *testing.T) {
	for _, fail := range []bool{false, true} {
		tx := &shellTx{fail: fail}
		err := CreateDemoShell(context.Background(), tx, "organization-a", "admin-a", "project-a", "/workspace/project-a/workspace")
		if (err != nil) != fail {
			t.Fatal(err)
		}
		if tx.calls != 1 || !strings.Contains(tx.sql, "INSERT INTO projects") {
			t.Fatal("must only create project metadata")
		}
		if tx.args[0] != "project-a" || tx.args[1] != "organization-a" || tx.args[2] != "工程模板" || tx.args[4] != "admin-a" {
			t.Fatalf("incorrect ownership: %v", tx.args)
		}
	}
}
