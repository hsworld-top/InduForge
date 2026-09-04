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
	failErr                                            error
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
	if f.failErr != nil {
		return f.failErr
	}
	f.failed = true
	f.task.State = "failed"
	f.task.RolledBack = rolled
	return nil
}

func TestRestoreExecutorRetriesWhenPermanentFailureCannotBePersisted(t *testing.T) {
	executor, repo, _, _, fences := restoreFixture("queued")
	repo.task.BackupKey = ""
	repo.failErr = errors.New("database temporarily unavailable")
	executor.actors.(*restoreActorResolverFake).user.Role = "TENANT_ADMIN"
	err := executor.process(context.Background(), repo.task)
	var terminal *restoreTerminalError
	if err == nil || errors.As(err, &terminal) || repo.failed {
		t.Fatalf("unpersisted failure must remain retryable: err=%v repo=%+v", err, repo)
	}
	if fences.calls != 0 || fences.active {
		t.Fatalf("actor validation failure acquired a fence: %+v", fences)
	}
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
	captureErr     error
}

func (f *restoreAuthoringFake) Capture(context.Context, auth.User, Project) (CapturedAuthoring, error) {
	if f.captureErr != nil {
		return CapturedAuthoring{}, f.captureErr
	}
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
	active       bool
	actor        auth.User
}

func (f *restoreFencesFake) CleanupRestoreFences(context.Context, Project, string) error {
	f.cleanupCalls++
	return nil
}

func (f *restoreFencesFake) WithRestoreFences(ctx context.Context, actor auth.User, _ Project, _ string, _ *bool, persist func(bool) error, fn func(context.Context, RestoreFenceTokens) error) error {
	f.calls++
	f.active = true
	f.actor = actor
	_ = persist(true)
	f.lastErr = fn(ctx, RestoreFenceTokens{Core: "core", Data: "data", DataEpoch: f.epoch, WorkspaceWasRunning: true, ResumeWorkspace: func(context.Context) error { return nil }})
	var terminal *restoreTerminalError
	if f.lastErr == nil || errors.As(f.lastErr, &terminal) {
		f.active = false
	}
	return f.lastErr
}

type restoreActorResolverFake struct {
	user auth.User
	err  error
}

type restoreChangeRecorder struct {
	calls []struct {
		tenant      string
		topics, ids []string
		terminal    bool
	}
}

func (r *restoreChangeRecorder) PublishChange(tenant string, topics, ids []string, terminal bool) {
	r.calls = append(r.calls, struct {
		tenant      string
		topics, ids []string
		terminal    bool
	}{tenant, topics, ids, terminal})
}

func (f *restoreActorResolverFake) GetActiveUser(context.Context, string) (auth.User, error) {
	return f.user, f.err
}

func restoreFixture(state string) (*RestoreExecutor, *restoreRepoFake, *restoreWorkspaceFake, *restoreDataFake, *restoreFencesFake) {
	task := RestoreTask{ID: "00000000-0000-0000-0000-000000000010", TenantID: "00000000-0000-0000-0000-000000000001", ProjectID: "00000000-0000-0000-0000-000000000002", VersionID: "00000000-0000-0000-0000-000000000003", RequestedBy: "00000000-0000-0000-0000-000000000004", State: state, ExpectedEpoch: 1, TargetEpoch: 2, BackupBucket: "ifp", BackupKey: "backup", BackupHash: "h", BackupProjectRevision: "r"}
	repo := &restoreRepoFake{task: task, project: Project{ID: task.ProjectID, TenantID: task.TenantID, AuthoringEpoch: 1}, version: Version{ID: task.VersionID, ProjectID: task.ProjectID, AuthoringSnapshotBucket: "ifp", AuthoringSnapshotKey: "target", AuthoringProjectRevision: "r"}}
	workspace := &restoreWorkspaceFake{}
	data := &restoreDataFake{epoch: "epoch-2"}
	fences := &restoreFencesFake{epoch: "epoch-2"}
	snapshot := authoringsnapshot.Snapshot{Workspace: map[string]string{}, WorkspaceModes: map[string]uint32{}, Scenes: json.RawMessage(`{}`), Data: json.RawMessage(`{}`)}
	actors := &restoreActorResolverFake{user: auth.User{ID: task.RequestedBy, TenantID: task.TenantID, Role: "DEVELOPER"}}
	executor := NewRestoreExecutor(repo, &restoreAuthoringFake{target: snapshot, backup: snapshot}, workspace, &restoreScenesFake{}, data, fences, actors, slog.Default())
	return executor, repo, workspace, data, fences
}

func TestRestoreExecutorPublishesCommittedOperationChanges(t *testing.T) {
	executor, repo, _, _, _ := restoreFixture("queued")
	changes := &restoreChangeRecorder{}
	executor.SetChangePublisher(changes)
	if err := executor.process(context.Background(), repo.task); err != nil {
		t.Fatal(err)
	}
	if len(changes.calls) == 0 {
		t.Fatal("恢复状态已推进但未通知运维记录")
	}
	var running, terminal bool
	for _, call := range changes.calls {
		if call.tenant != repo.task.TenantID || len(call.topics) != 1 || call.topics[0] != "events" || len(call.ids) != 2 || call.ids[0] != repo.task.ProjectID || call.ids[1] != repo.task.ID {
			t.Fatalf("恢复通知范围错误 %+v", call)
		}
		if call.terminal {
			terminal = true
		} else {
			running = true
		}
	}
	if !running || !terminal {
		t.Fatalf("运行中/终态通知不完整 %+v", changes.calls)
	}
}

func TestRestoreExecutorEnqueuePublishesAcceptedAsNonTerminal(t *testing.T) {
	executor, repo, _, _, _ := restoreFixture("queued")
	changes := &restoreChangeRecorder{}
	executor.SetChangePublisher(changes)
	// 预占运行槽，隔离 Enqueue 的受理通知，避免后台执行器并发追加后续阶段通知。
	executor.running[repo.task.ID] = struct{}{}
	executor.Enqueue(repo.task)
	if len(changes.calls) == 0 {
		t.Fatal("已受理恢复任务未通知运维记录")
	}
	accepted := changes.calls[0]
	if accepted.terminal || accepted.tenant != repo.task.TenantID || len(accepted.topics) != 1 || accepted.topics[0] != "events" || len(accepted.ids) != 2 || accepted.ids[0] != repo.task.ProjectID || accepted.ids[1] != repo.task.ID {
		t.Fatalf("已受理通知必须是精确的非终态 events 失效提示 %+v", accepted)
	}
}

func TestRestoreExecutorUsesRequestedUsersRealRole(t *testing.T) {
	executor, repo, _, _, fences := restoreFixture("restoring_data")
	if err := executor.process(context.Background(), repo.task); err != nil {
		t.Fatal(err)
	}
	if fences.actor.ID != repo.task.RequestedBy || fences.actor.TenantID != repo.task.TenantID || fences.actor.Role != "DEVELOPER" {
		t.Fatalf("restore did not use resolved request actor: %+v", fences.actor)
	}
}

func TestRestoreExecutorRejectsCrossTenantActorAndFailsTaskWithoutFence(t *testing.T) {
	executor, repo, _, _, fences := restoreFixture("queued")
	repo.task.BackupKey = ""
	executor.actors.(*restoreActorResolverFake).user.TenantID = "00000000-0000-0000-0000-000000000099"
	err := executor.process(context.Background(), repo.task)
	var terminal *restoreTerminalError
	if !errors.As(err, &terminal) || !repo.failed || repo.task.State != "failed" {
		t.Fatalf("cross-tenant actor did not fail durably: err=%v repo=%+v", err, repo)
	}
	if fences.calls != 0 || fences.active {
		t.Fatalf("cross-tenant actor acquired a fence: %+v", fences)
	}
}

func TestRestoreExecutorRejectsInvalidRoleAndFailsTaskWithoutFence(t *testing.T) {
	executor, repo, _, _, fences := restoreFixture("queued")
	repo.task.BackupKey = ""
	executor.actors.(*restoreActorResolverFake).user.Role = "TENANT_ADMIN"
	err := executor.process(context.Background(), repo.task)
	var terminal *restoreTerminalError
	if !errors.As(err, &terminal) || !repo.failed || repo.task.State != "failed" {
		t.Fatalf("invalid actor role did not fail durably: err=%v repo=%+v", err, repo)
	}
	if fences.calls != 0 || fences.active {
		t.Fatalf("invalid actor role acquired a fence: %+v", fences)
	}
}

func TestRestoreExecutorFailsPermanentCaptureErrorAndReleasesFence(t *testing.T) {
	for _, captureErr := range []error{auth.ErrPermissionDenied, ErrAuthoringSnapshotBuilderUnavailable} {
		executor, repo, _, _, fences := restoreFixture("queued")
		repo.task.BackupKey = ""
		executor.authoring.(*restoreAuthoringFake).captureErr = captureErr
		err := executor.process(context.Background(), repo.task)
		var terminal *restoreTerminalError
		if !errors.As(err, &terminal) || !repo.failed || repo.task.State != "failed" {
			t.Fatalf("permanent capture error did not fail durably: cause=%v err=%v repo=%+v", captureErr, err, repo)
		}
		if fences.calls != 1 || fences.active {
			t.Fatalf("permanent capture error retained a fence: cause=%v fences=%+v", captureErr, fences)
		}
	}
}

func TestRestoreExecutorRetainsFenceForRetryableCaptureError(t *testing.T) {
	executor, repo, _, _, fences := restoreFixture("queued")
	repo.task.BackupKey = ""
	executor.authoring.(*restoreAuthoringFake).captureErr = errors.New("temporary object store failure")
	err := executor.process(context.Background(), repo.task)
	if err == nil || repo.failed || repo.task.State == "failed" {
		t.Fatalf("retryable capture error must remain recoverable: err=%v repo=%+v", err, repo)
	}
	if fences.calls != 1 || !fences.active {
		t.Fatalf("retryable capture error must retain its fence: %+v", fences)
	}
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
