package deployment

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"

	"github.com/indu-forge/dev_core/internal/auth"
	"github.com/indu-forge/dev_core/internal/authoringsnapshot"
	"github.com/indu-forge/dev_core/internal/sceneasset"
	"github.com/jackc/pgx/v5"
)

type restoreRepoFake struct {
	task                                               RestoreTask
	project                                            Project
	version                                            Version
	finalized, compensated, completed, cleaned, failed bool
}

func (f *restoreRepoFake) GetRestoreTask(context.Context, string, string) (RestoreTask, error) {
	return f.task, nil
}
func (f *restoreRepoFake) ListRecoverableRestoreTasks(context.Context) ([]RestoreTask, error) {
	return nil, nil
}
func (f *restoreRepoFake) SetRestoreStage(_ context.Context, _ string, state, stage string) error {
	f.task.State, f.task.Stage = state, stage
	return nil
}
func (f *restoreRepoFake) SetRestoreBackup(context.Context, string, AuthoringSnapshotMetadata) error {
	return nil
}
func (f *restoreRepoFake) SetRestoreWorkspaceWasRunning(_ context.Context, _ string, running bool) error {
	f.task.WorkspaceWasRunning = &running
	return nil
}
func (f *restoreRepoFake) FailRestoreTask(_ context.Context, _ string, _ string, rolled bool) error {
	f.failed = true
	f.task.State = "failed"
	f.task.RolledBack = rolled
	return nil
}
func (f *restoreRepoFake) FinalizeRestoreCore(_ context.Context, _ RestoreTask, _ applyRestoreCore) error {
	f.finalized = true
	f.task.State = "finalizing"
	f.project.AuthoringEpoch = f.task.TargetEpoch
	return nil
}
func (f *restoreRepoFake) CompensateRestoreCore(_ context.Context, _ RestoreTask, _ applyRestoreCore) error {
	f.compensated = true
	f.project.AuthoringEpoch = f.task.ExpectedEpoch
	return nil
}
func (f *restoreRepoFake) CompleteRestore(context.Context, string) error {
	f.completed = true
	f.task.State = "succeeded"
	return nil
}
func (f *restoreRepoFake) MarkRestoreCleanupComplete(context.Context, string) error {
	f.cleaned = true
	return nil
}
func (f *restoreRepoFake) GetProject(context.Context, string, string) (Project, error) {
	return f.project, nil
}
func (f *restoreRepoFake) GetVersion(context.Context, string, string) (Version, error) {
	return f.version, nil
}

type restoreWorkspaceFake struct{ staged, activated, rolled, finalized int }

func (f *restoreWorkspaceFake) StageRestoreWithModes(string, string, map[string]string, map[string]uint32) error {
	f.staged++
	return nil
}
func (f *restoreWorkspaceFake) ActivateRestore(string, string) error { f.activated++; return nil }
func (f *restoreWorkspaceFake) RollbackRestore(string, string) error { f.rolled++; return nil }
func (f *restoreWorkspaceFake) FinalizeRestore(string, string) error { f.finalized++; return nil }

type restoreScenesFake struct{ prepared int }

func (f *restoreScenesFake) PrepareAuthoringRestore(context.Context, auth.User, string, sceneasset.AuthoringSnapshotOptions, sceneasset.AuthoringSnapshot) (*sceneasset.PreparedAuthoringRestore, error) {
	f.prepared++
	return &sceneasset.PreparedAuthoringRestore{}, nil
}
func (f *restoreScenesFake) ApplyPreparedAuthoringRestore(context.Context, pgx.Tx, auth.User, string, string, *sceneasset.PreparedAuthoringRestore) error {
	return nil
}

type restoreDataFake struct {
	epoch        string
	replacements []string
}

func (f *restoreDataFake) ReplaceAuthoringSnapshotFenced(_ context.Context, _, _, _, _, _, _, _, direction string, _ json.RawMessage) error {
	f.replacements = append(f.replacements, direction)
	if direction == "forward" {
		f.epoch = "epoch-2"
	} else {
		f.epoch = "epoch-1"
	}
	return nil
}
func (f *restoreDataFake) GetAuthoringEpoch(context.Context, string, string) (string, error) {
	return f.epoch, nil
}

type restoreAuthoringFake struct {
	target, backup authoringsnapshot.Snapshot
	targetErr      error
}

func (f *restoreAuthoringFake) Capture(context.Context, auth.User, Project) (CapturedAuthoring, error) {
	return CapturedAuthoring{}, errors.New("unexpected capture")
}
func (f *restoreAuthoringFake) PersistBackup(context.Context, Project, string, CapturedAuthoring) (AuthoringSnapshotMetadata, error) {
	return AuthoringSnapshotMetadata{}, errors.New("unexpected backup")
}
func (f *restoreAuthoringFake) OpenStored(_ context.Context, _ Project, m AuthoringSnapshotMetadata) (authoringsnapshot.Snapshot, error) {
	if m.Key == "backup" {
		return f.backup, nil
	}
	return f.target, f.targetErr
}
func (f *restoreAuthoringFake) DeleteSnapshot(context.Context, string) error { return nil }

type restoreFencesFake struct {
	epoch        string
	calls        int
	cleanupCalls int
	lastErr      error
}

func (f *restoreFencesFake) CleanupRestoreFences(context.Context, Project, string) error {
	f.cleanupCalls++
	return nil
}

func (f *restoreFencesFake) WithRestoreFences(ctx context.Context, _ auth.User, _ Project, _ string, _ *bool, persist func(bool) error, fn func(context.Context, RestoreFenceTokens) error) error {
	f.calls++
	_ = persist(true)
	f.lastErr = fn(ctx, RestoreFenceTokens{Core: "core", Data: "data", DataEpoch: f.epoch, WorkspaceWasRunning: true, ResumeWorkspace: func(context.Context) error { return nil }})
	return f.lastErr
}

func restoreFixture(state string) (*RestoreExecutor, *restoreRepoFake, *restoreWorkspaceFake, *restoreDataFake, *restoreFencesFake) {
	task := RestoreTask{ID: "00000000-0000-0000-0000-000000000010", TenantID: "00000000-0000-0000-0000-000000000001", ProjectID: "00000000-0000-0000-0000-000000000002", VersionID: "00000000-0000-0000-0000-000000000003", RequestedBy: "00000000-0000-0000-0000-000000000004", State: state, ExpectedEpoch: 1, TargetEpoch: 2, BackupBucket: "ifp", BackupKey: "backup", BackupHash: "h", BackupProjectRevision: "r"}
	repo := &restoreRepoFake{task: task, project: Project{ID: task.ProjectID, TenantID: task.TenantID, AuthoringEpoch: 1}, version: Version{ID: task.VersionID, ProjectID: task.ProjectID, AuthoringSnapshotBucket: "ifp", AuthoringSnapshotKey: "target", AuthoringProjectRevision: "r"}}
	workspace := &restoreWorkspaceFake{}
	data := &restoreDataFake{epoch: "epoch-2"}
	fences := &restoreFencesFake{epoch: "epoch-2"}
	snapshot := authoringsnapshot.Snapshot{Workspace: map[string]string{}, WorkspaceModes: map[string]uint32{}, Scenes: json.RawMessage(`{}`), Data: json.RawMessage(`{}`)}
	executor := NewRestoreExecutor(repo, &restoreAuthoringFake{target: snapshot, backup: snapshot}, workspace, &restoreScenesFake{}, data, fences, slog.Default())
	return executor, repo, workspace, data, fences
}

func TestRestoreExecutorResumesAfterDataForwardWithoutRepeatingForward(t *testing.T) {
	executor, repo, workspace, data, _ := restoreFixture("restoring_data")
	if err := executor.process(context.Background(), repo.task); err != nil {
		t.Fatal(err)
	}
	if len(data.replacements) != 0 {
		t.Fatalf("data forward repeated: %v", data.replacements)
	}
	if !repo.finalized || !repo.completed || !repo.cleaned {
		t.Fatalf("core/task not converged: %+v", repo)
	}
	if workspace.activated != 1 || workspace.finalized != 1 {
		t.Fatalf("workspace not finalized: %+v", workspace)
	}
}

func TestRestoreExecutorResumesCompensationAndReturnsTerminal(t *testing.T) {
	executor, repo, workspace, data, fences := restoreFixture("compensating")
	data.epoch = "epoch-1"
	fences.epoch = "epoch-1"
	err := executor.process(context.Background(), repo.task)
	var terminal *restoreTerminalError
	if !errors.As(err, &terminal) {
		t.Fatalf("expected terminal compensated error, got %v", err)
	}
	if !repo.compensated || !repo.failed || !repo.task.RolledBack {
		t.Fatalf("compensation not durable: %+v", repo)
	}
	if workspace.rolled != 1 || len(data.replacements) != 0 {
		t.Fatalf("unexpected compensation effects: workspace=%+v data=%v", workspace, data.replacements)
	}
}

func TestRestoreExecutorCleansSucceededTaskAfterRestart(t *testing.T) {
	executor, repo, workspace, _, fences := restoreFixture("succeeded")
	if err := executor.process(context.Background(), repo.task); err != nil {
		t.Fatal(err)
	}
	if fences.calls != 0 || fences.cleanupCalls != 1 || workspace.finalized != 1 || !repo.cleaned {
		t.Fatalf("cleanup did not converge without freezing workspace: calls=%d cleanup=%d workspace=%+v repo=%+v", fences.calls, fences.cleanupCalls, workspace, repo)
	}
}

func TestRestoreExecutorRollsBackWhenTargetIsUnreadableAfterBackup(t *testing.T) {
	executor, repo, workspace, data, fences := restoreFixture("staging")
	executor.authoring.(*restoreAuthoringFake).targetErr = errors.New("target corrupted")
	data.epoch = "epoch-1"
	fences.epoch = "epoch-1"
	err := executor.process(context.Background(), repo.task)
	var terminal *restoreTerminalError
	if !errors.As(err, &terminal) || !repo.failed || !repo.task.RolledBack {
		t.Fatalf("unreadable target did not converge through compensation: err=%v repo=%+v", err, repo)
	}
	if workspace.rolled != 1 {
		t.Fatalf("workspace rollback not attempted: %+v", workspace)
	}
}
