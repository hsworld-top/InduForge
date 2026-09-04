package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

type snapshotTxStarterStub struct {
	tx      *snapshotTxStub
	options pgx.TxOptions
	err     error
}

func (s *snapshotTxStarterStub) BeginTx(_ context.Context, options pgx.TxOptions) (pgx.Tx, error) {
	s.options = options
	if s.err != nil {
		return nil, s.err
	}
	return s.tx, nil
}

type snapshotTxStub struct {
	pgx.Tx
	queries     int
	queryRows   int
	failQueryAt int
	committed   bool
	rolledBack  bool
}

func (s *snapshotTxStub) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
	s.queries++
	if s.failQueryAt == s.queries {
		return nil, errors.New("query failed")
	}
	return snapshotRowsStub{}, nil
}

func (s *snapshotTxStub) QueryRow(_ context.Context, _ string, _ ...any) pgx.Row {
	s.queryRows++
	return snapshotRowStub{binding: s.queryRows == 1}
}

func (s *snapshotTxStub) Commit(context.Context) error   { s.committed = true; return nil }
func (s *snapshotTxStub) Rollback(context.Context) error { s.rolledBack = true; return nil }

type snapshotRowsStub struct{ pgx.Rows }

func (snapshotRowsStub) Close()     {}
func (snapshotRowsStub) Next() bool { return false }
func (snapshotRowsStub) Err() error { return nil }

type snapshotRowStub struct{ binding bool }

func (s snapshotRowStub) Scan(dest ...any) error {
	if s.binding {
		*dest[0].(*bool) = true
		return nil
	}
	return pgx.ErrNoRows
}

func TestProjectSnapshotGetUsesRepeatableReadReadOnlyTransaction(t *testing.T) {
	tx := &snapshotTxStub{}
	starter := &snapshotTxStarterStub{tx: tx}
	repo := &ProjectSnapshotRepository{pool: starter}

	if _, err := repo.GetByProject(context.Background(), "project-1", "tenant-1"); err != nil {
		t.Fatalf("GetByProject() error = %v", err)
	}
	if starter.options.IsoLevel != pgx.RepeatableRead || starter.options.AccessMode != pgx.ReadOnly {
		t.Fatalf("BeginTx options = %#v", starter.options)
	}
	if tx.queries == 0 || tx.queryRows < 3 {
		t.Fatalf("expected snapshot queries through tx, queries=%d queryRows=%d", tx.queries, tx.queryRows)
	}
	if !tx.committed {
		t.Fatal("successful snapshot read did not commit")
	}
	if !tx.rolledBack {
		t.Fatal("deferred rollback was not attempted after commit")
	}
}

func TestProjectSnapshotGetRollsBackWhenTransactionQueryFails(t *testing.T) {
	tx := &snapshotTxStub{failQueryAt: 1}
	repo := &ProjectSnapshotRepository{pool: &snapshotTxStarterStub{tx: tx}}

	if _, err := repo.GetByProject(context.Background(), "project-1", "tenant-1"); err == nil {
		t.Fatal("GetByProject() error = nil")
	}
	if tx.committed {
		t.Fatal("failed snapshot read committed")
	}
	if !tx.rolledBack {
		t.Fatal("failed snapshot read did not roll back")
	}
	if tx.queryRows != 1 {
		t.Fatalf("tenant binding queryRows = %d, want 1", tx.queryRows)
	}
}
