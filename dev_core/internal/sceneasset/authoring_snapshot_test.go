package sceneasset

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/indu-forge/dev_core/internal/authoringsnapshot"
	"github.com/jackc/pgx/v5"
)

func TestValidateAuthoringSnapshotSupportsDraftOnlyAsset(t *testing.T) {
	content := []byte(`{"type":"component"}`)
	snapshot := AuthoringSnapshot{ProjectID: "project-1", Assets: []AuthoringAssetSnapshot{{
		Asset:      AuthoringAsset{ID: "asset-1", ProjectID: "project-1", Provider: "ht", CurrentGeneration: 0, DraftVersion: 1},
		DraftFiles: []AuthoringFileSnapshot{{Path: "asset.json", ContentType: "application/json", Hash: hashBytes(content), Content: content}},
	}}}
	if err := validateAuthoringSnapshot(snapshot); err != nil {
		t.Fatalf("validateAuthoringSnapshot() error = %v", err)
	}
}

func TestAuthoringSnapshotCodecRoundTripPreservesRestoreFields(t *testing.T) {
	content := []byte(`{"nodes":[]}`)
	original := AuthoringSnapshot{ProjectID: "11111111-1111-4111-8111-111111111111", Fence: "epoch-7", Scenes: []AuthoringSceneSnapshot{{Scene: AuthoringScene{ID: "scene-row", ProjectID: "11111111-1111-4111-8111-111111111111", SceneID: "scene-code", Kind: "2d", Name: "主场景", Provider: "ht", EntryPath: "scene.json", PublicContract: map[string]any{}, ManagedContract: map[string]any{}, DatapointRefs: []string{}, CurrentRevision: 3, DraftVersion: 5, CommittedDraftVersion: 4}, Revision: &AuthoringRevision{ID: "revision-row", Revision: 3, DraftVersion: 4, Provider: "ht", ProviderVersion: "1", EntryPath: "scene.json", RootHash: hashBytes(content), PublicContract: map[string]any{}, DatapointRefs: []string{}}, DraftFiles: []AuthoringFileSnapshot{{Path: "scene.json", ContentType: "application/json", Hash: hashBytes(content), Content: content}}, RevisionFiles: []AuthoringFileSnapshot{{Path: "scene.json", ContentType: "application/json", Hash: hashBytes(content), Content: content}}}}}
	raw, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	envelope := authoringsnapshot.Snapshot{SchemaVersion: authoringsnapshot.SchemaVersion, TenantID: "22222222-2222-4222-8222-222222222222", ProjectID: original.ProjectID, ProjectRevision: "sha256:revision-9", CapturedAt: time.Now().UTC(), Workspace: map[string]string{}, WorkspaceModes: map[string]uint32{}, Scenes: raw, Data: json.RawMessage(`{}`)}
	key := bytes.Repeat([]byte{3}, 32)
	sealed, err := authoringsnapshot.Seal(envelope, key, bytes.NewReader(bytes.Repeat([]byte{4}, 32)))
	if err != nil {
		t.Fatal(err)
	}
	opened, err := authoringsnapshot.Open(sealed.Bytes, key, envelope.TenantID, envelope.ProjectID, "sha256:revision-9")
	if err != nil {
		t.Fatal(err)
	}
	var restored AuthoringSnapshot
	if err = json.Unmarshal(opened.Scenes, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.Scenes[0].Scene.Provider != "ht" || restored.Scenes[0].Scene.EntryPath != "scene.json" || restored.Scenes[0].Revision.ID != "revision-row" {
		t.Fatalf("restore fields lost: %#v", restored.Scenes[0])
	}
	if err = validateAuthoringSnapshot(restored); err != nil {
		t.Fatalf("round-trip snapshot invalid: %v", err)
	}
}

type authoringPoolStub struct {
	sceneAssetPool
	tx      *authoringTxStub
	options pgx.TxOptions
}

func (s *authoringPoolStub) BeginTx(_ context.Context, options pgx.TxOptions) (pgx.Tx, error) {
	s.options = options
	return s.tx, nil
}

type authoringTxStub struct {
	pgx.Tx
	queries    int
	fail       bool
	committed  bool
	rolledBack bool
}

func (s *authoringTxStub) Query(context.Context, string, ...any) (pgx.Rows, error) {
	s.queries++
	if s.fail {
		return nil, errors.New("query failed")
	}
	return authoringRowsStub{}, nil
}
func (s *authoringTxStub) Commit(context.Context) error   { s.committed = true; return nil }
func (s *authoringTxStub) Rollback(context.Context) error { s.rolledBack = true; return nil }

type authoringRowsStub struct{ pgx.Rows }

func (authoringRowsStub) Close()     {}
func (authoringRowsStub) Next() bool { return false }
func (authoringRowsStub) Err() error { return nil }

func TestExportAuthoringSnapshotUsesOneReadOnlyRepeatableReadTransaction(t *testing.T) {
	tx := &authoringTxStub{}
	pool := &authoringPoolStub{tx: tx}
	repo := &PostgreSQLRepository{pool: pool}
	if _, _, err := repo.exportAuthoringSnapshot(context.Background(), "tenant-1", "project-1"); err != nil {
		t.Fatal(err)
	}
	if pool.options.IsoLevel != pgx.RepeatableRead || pool.options.AccessMode != pgx.ReadOnly {
		t.Fatalf("options=%#v", pool.options)
	}
	if tx.queries != 3 {
		t.Fatalf("queries=%d, want 3 through tx", tx.queries)
	}
	if !tx.committed || !tx.rolledBack {
		t.Fatalf("commit=%v rollback=%v", tx.committed, tx.rolledBack)
	}
}

func TestExportAuthoringSnapshotRollsBackOnQueryFailure(t *testing.T) {
	tx := &authoringTxStub{fail: true}
	repo := &PostgreSQLRepository{pool: &authoringPoolStub{tx: tx}}
	if _, _, err := repo.exportAuthoringSnapshot(context.Background(), "tenant-1", "project-1"); err == nil {
		t.Fatal("error=nil")
	}
	if tx.committed || !tx.rolledBack {
		t.Fatalf("commit=%v rollback=%v", tx.committed, tx.rolledBack)
	}
}

func TestValidateAuthoringSnapshotRejectsInvalidNestedReferences(t *testing.T) {
	snapshot := AuthoringSnapshot{ProjectID: "project-1", Bindings: []AuthoringBindingSnapshot{{SceneDocumentID: "missing", AssetID: "missing", Generation: 1}}}
	if err := validateAuthoringSnapshot(snapshot); !errors.Is(err, ErrInvalidArchive) {
		t.Fatalf("error = %v, want ErrInvalidArchive", err)
	}
}

func TestValidateAuthoringSnapshotRejectsChangedFileContent(t *testing.T) {
	snapshot := AuthoringSnapshot{ProjectID: "project-1", Scenes: []AuthoringSceneSnapshot{{
		Scene:      AuthoringScene{ID: "scene-1", ProjectID: "project-1", Provider: "ht"},
		DraftFiles: []AuthoringFileSnapshot{{Path: "scene.json", Hash: hashBytes([]byte("original")), Content: []byte("changed")}},
	}}}
	if err := validateAuthoringSnapshot(snapshot); !errors.Is(err, ErrInvalidArchive) {
		t.Fatalf("error = %v, want ErrInvalidArchive", err)
	}
}
