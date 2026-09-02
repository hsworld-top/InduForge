// Package runtimeengine owns RuntimeEngine's startup and shutdown boundary.
package runtimeengine

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/indu-forge/runtime-engine/internal/alarm"
	"github.com/indu-forge/runtime-engine/internal/command"
	"github.com/indu-forge/runtime-engine/internal/compute"
	"github.com/indu-forge/runtime-engine/internal/httpapi"
	"github.com/indu-forge/runtime-engine/internal/ingress"
	"github.com/indu-forge/runtime-engine/internal/loader"
	"github.com/indu-forge/runtime-engine/internal/model"
	"github.com/indu-forge/runtime-engine/internal/outbox"
	"github.com/indu-forge/runtime-engine/internal/resolver"
	"github.com/indu-forge/runtime-engine/internal/store/postgres"
	"github.com/indu-forge/runtime-engine/internal/transport/jetstream"
	"github.com/indu-forge/runtime-engine/internal/writer"
)

// Options deliberately accepts paths rather than dependency values: secrets
// are resolved inside the trusted resolver and never cross the host API.
type Options struct {
	ConfigPath, ConfigRoot, IndexPath string
	Production                        bool
	Version                           string
}
type Host struct {
	options        Options
	State          *httpapi.EngineState
	mu             sync.Mutex
	store          *postgres.Store
	nats           *jetstream.Client
	cancel         context.CancelFunc
	intakeCancel   context.CancelFunc
	fatal          chan struct{}
	fatalOnce      sync.Once
	done           chan struct{}
	wg             sync.WaitGroup
	startAttempted bool
	stopping       bool
	shutdownDone   chan struct{}
	shutdownErr    error
}

func New(options Options) *Host {
	return &Host{options: options, State: httpapi.NewEngineState(options.Version, nil, nil), done: make(chan struct{}), fatal: make(chan struct{})}
}
func (h *Host) Fatal() <-chan struct{} { return h.fatal }

// Start performs every mutating step only after all read-only preflight gates
// pass.  Activation is a single Store transaction, so a failed start leaves no
// partially live role or producer fence.
func (h *Host) Start(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.startAttempted || h.stopping {
		return errors.New("RuntimeEngine 已启动")
	}
	h.startAttempted = true
	readOnly := h.options.Production
	loaded, err := loader.Load(loader.Options{ConfigPath: h.options.ConfigPath, ConfigRoot: h.options.ConfigRoot, RequireReadOnlyMount: &readOnly})
	if err != nil {
		// 仅记录固定阶段码，避免将制品路径、摘要或 Secret 泄露至容器日志。
		log.Printf("RuntimeEngine artifact validation stage=%s", loader.DiagnosticCode(err))
		return h.fail("ARTIFACT_VALIDATION_FAILED")
	}
	h.State.SetLoaded(loaded)
	index, err := resolver.Open(h.options.IndexPath, h.options.Production)
	if err != nil {
		return h.fail("SITE_INDEX_INVALID")
	}
	dsn, err := index.ResolvePostgres(ctx, loaded.Config.StateStore.DSNSecretRef)
	if err != nil {
		return h.fail("POSTGRES_SECRET_INVALID")
	}
	store, err := postgres.Open(ctx, dsn)
	if err != nil {
		log.Printf("RuntimeEngine PostgreSQL preflight stage=%s", postgres.DiagnosticCode(err))
		return h.fail("POSTGRES_PREFLIGHT_FAILED")
	}
	natsOptions, err := index.ResolveNATS(ctx, loaded.Config.JetStream.ServerResourceRef, loaded.Config.JetStream.CredentialSecretRef, loaded.Config.AccountID)
	if err != nil {
		store.Close()
		return h.fail("NATS_CREDENTIAL_INVALID")
	}
	natsClient, err := jetstream.Open(ctx, natsOptions, loaded.Config.AccountID)
	if err != nil {
		store.Close()
		return h.fail("NATS_PREFLIGHT_FAILED")
	}
	if err = natsClient.ValidateStreamsAndConsumers(ctx, loaded.Config); err != nil {
		natsClient.Close()
		store.Close()
		return h.fail("NATS_TOPOLOGY_INVALID")
	}
	// Construct all role handlers and perform their read-only preflight before
	// any fence can become active.
	handlers, alarmHandler, computeHandler, err := buildHandlers(ctx, loaded, index, store)
	if err != nil {
		natsClient.Close()
		store.Close()
		return h.fail("HANDLER_PREFLIGHT_FAILED")
	}
	plan, err := buildWorkerPlan(loaded, store, natsClient, handlers, alarmHandler, computeHandler, h.State)
	if err != nil {
		natsClient.Close()
		store.Close()
		return h.fail("WORKER_CONFIGURATION_FAILED")
	}
	if err = store.ActivateAssignments(ctx, loaded.Config); err != nil {
		natsClient.Close()
		store.Close()
		return h.fail("ASSIGNMENT_ACTIVATION_FAILED")
	}
	intakeCtx, intakeCancel := context.WithCancel(context.Background())
	runCtx, cancel := context.WithCancel(context.Background())
	h.store = store
	h.nats = natsClient
	h.cancel = cancel
	h.intakeCancel = intakeCancel
	// Never let a worker failure race a later RUNNING write.
	h.State.SetState(httpapi.Running, httpapi.Healthy, "")
	h.startWorkers(intakeCtx, runCtx, loaded, plan)
	close(h.done)
	return nil
}
func (h *Host) fail(code string) error {
	h.State.SetState(httpapi.Failed, httpapi.Unavailable, code)
	return errors.New(code)
}
func (h *Host) Shutdown(ctx context.Context) error {
	h.mu.Lock()
	if h.shutdownDone != nil {
		done := h.shutdownDone
		h.mu.Unlock()
		select {
		case <-done:
			h.mu.Lock()
			err := h.shutdownErr
			h.mu.Unlock()
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	h.shutdownDone = make(chan struct{})
	done := h.shutdownDone
	defer func() { h.mu.Lock(); close(done); h.mu.Unlock() }()
	h.stopping = true
	if h.cancel == nil {
		h.State.SetState(httpapi.Stopped, httpapi.Unavailable, "")
		h.shutdownErr = nil
		h.mu.Unlock()
		return nil
	}
	h.State.SetState(httpapi.Stopping, httpapi.Unavailable, "")
	cancel := h.cancel
	intakeCancel := h.intakeCancel
	natsClient, store := h.nats, h.store
	h.cancel = nil
	h.mu.Unlock()
	if intakeCancel != nil {
		intakeCancel()
	}
	select {
	case <-ctx.Done():
		cancel()
		if natsClient != nil {
			natsClient.Close()
		}
		if store != nil {
			store.Abort()
		}
		h.State.SetState(httpapi.Failed, httpapi.Unavailable, "SHUTDOWN_TIMEOUT")
		h.mu.Lock()
		h.store = nil
		h.nats = nil
		h.shutdownErr = ctx.Err()
		h.mu.Unlock()
		return ctx.Err()
	case <-waitGroup(&h.wg):
	}
	cancel()
	if natsClient != nil {
		natsClient.Close()
	}
	if store != nil {
		store.Close()
	}
	h.State.SetState(httpapi.Stopped, httpapi.Unavailable, "")
	h.mu.Lock()
	h.store = nil
	h.nats = nil
	h.shutdownErr = nil
	h.mu.Unlock()
	return nil
}

func waitGroup(group *sync.WaitGroup) <-chan struct{} {
	done := make(chan struct{})
	go func() { group.Wait(); close(done) }()
	return done
}

func buildHandlers(ctx context.Context, loaded *loader.Loaded, index *resolver.Index, store *postgres.Store) (map[string]ingress.PostgresHandler, *alarm.Handler, *compute.Handler, error) {
	handlers := map[string]ingress.PostgresHandler{}
	if roleEnabled(loaded.Config, "writer") {
		handlers["writer"] = writer.NewPostgresHandler()
	}
	var alarmHandler *alarm.Handler
	if roleEnabled(loaded.Config, "alarm") {
		var err error
		alarmHandler, err = alarm.NewPostgresHandler(loaded.Artifact, loaded.Config)
		if err != nil {
			return nil, nil, nil, err
		}
		if err = alarmHandler.Preflight(ctx, store); err != nil {
			return nil, nil, nil, err
		}
		handlers["alarm"] = alarmHandler.PostgresHandler()
	}
	var computeHandler *compute.Handler
	if roleEnabled(loaded.Config, "compute") {
		sandbox, err := compute.NewSandboxClient(index, loaded.Config.ComputeSandbox.ServerResourceRef, loaded.Config.ComputeSandbox.CredentialSecretRef)
		if err != nil {
			return nil, nil, nil, err
		}
		if err = sandbox.Preflight(ctx, compute.SandboxIdentity{SiteID: loaded.Config.SiteID, DeploymentID: loaded.Config.DeploymentID, ProjectID: loaded.Config.ProjectID}); err != nil {
			return nil, nil, nil, err
		}
		computeHandler, err = compute.NewPostgresHandler(loaded.Artifact, loaded.Config, sandbox)
		if err != nil {
			return nil, nil, nil, err
		}
		handlers["compute"] = computeHandler.PostgresHandler()
	}
	return handlers, alarmHandler, computeHandler, nil
}
func roleEnabled(c model.EngineConfig, role string) bool {
	for _, v := range c.Roles {
		if v == role {
			return true
		}
	}
	return false
}
func roleToken(c model.EngineConfig, role string) (ingress.ConsumerToken, bool) {
	for _, a := range c.RoleAssignments {
		if a.Role == role {
			return ingress.ConsumerToken{OwnerID: a.Ownership.OwnerID, Epoch: a.Ownership.Epoch}, true
		}
	}
	return ingress.ConsumerToken{}, false
}

type workerPlan struct {
	store   *postgres.Store
	nats    *jetstream.Client
	runners []*ingress.Runner
	outbox  *outbox.Worker
	alarm   *alarm.Handler
	compute *compute.Handler
	command *command.Consumer
}

func buildWorkerPlan(loaded *loader.Loaded, store *postgres.Store, natsClient *jetstream.Client, handlers map[string]ingress.PostgresHandler, alarmHandler *alarm.Handler, computeHandler *compute.Handler, health *httpapi.EngineState) (*workerPlan, error) {
	plan := &workerPlan{store: store, nats: natsClient, alarm: alarmHandler, compute: computeHandler}
	for _, consumer := range loaded.Config.JetStream.Consumers {
		if consumer.FilterSubject == "compute.command.>" {
			token, ok := roleToken(loaded.Config, "compute")
			if !ok || computeHandler == nil {
				return nil, errors.New("command worker invalid")
			}
			commandConsumer, err := command.New(loaded.Config, consumer, postgres.ConsumerRoleToken{OwnerID: token.OwnerID, Epoch: token.Epoch}, store, computeHandler)
			if err != nil {
				return nil, err
			}
			plan.command = commandConsumer
			continue
		}
		handler := handlers[consumer.Role]
		token, ok := roleToken(loaded.Config, consumer.Role)
		if !ok || handler == nil {
			return nil, errors.New("worker plan invalid")
		}
		runner, err := ingress.NewRunnerWithHealth(loaded, consumer, token, ingress.NewPostgresAdapter(store, loaded.Config, handler), health)
		if err != nil {
			return nil, err
		}
		plan.runners = append(plan.runners, runner)
	}
	worker, err := outbox.NewWorker(outbox.NewPostgresAdapter(store), natsClient, outbox.Options{DeploymentID: loaded.Config.DeploymentID, LeaseOwner: "runtime-engine-outbox", BatchSize: 32, LeaseFor: 15 * time.Second, PollInterval: time.Second, BaseRetry: time.Second, MaxRetry: 30 * time.Second, DrainTimeout: 20 * time.Second, OnIntegrityFault: func() { health.ReportDegraded("OUTBOX_INTEGRITY_FAULT") }})
	if err != nil {
		return nil, err
	}
	plan.outbox = worker
	return plan, nil
}
func (h *Host) startWorkers(intake, work context.Context, loaded *loader.Loaded, plan *workerPlan) {
	for _, runner := range plan.runners {
		h.launch(work, func() {
			if err := runner.RunWithDrain(intake, work, plan.nats, 32, 5*time.Second); err != nil {
				h.workerFatal("INGRESS_FATAL")
			}
		})
	}
	if plan.command != nil {
		h.launch(work, func() {
			if err := plan.command.Run(intake, work, plan.nats); err != nil && intake.Err() == nil {
				h.workerFatal("COMMAND_FATAL")
			}
		})
	}
	if plan.outbox != nil {
		h.launch(work, func() {
			if err := plan.outbox.RunWithDrain(intake, work); err != nil {
				h.workerFatal("OUTBOX_FATAL")
			}
		})
	}
	if plan.alarm != nil {
		token, _ := roleToken(loaded.Config, "alarm")
		h.launch(work, func() {
			if err := plan.alarm.RunSweepRunnerWithDrain(intake, work, plan.store, postgres.ConsumerRoleToken{OwnerID: token.OwnerID, Epoch: token.Epoch}, time.Second, 64, func() { h.State.ReportDegraded("ALARM_SWEEP_POISON") }); err != nil && intake.Err() == nil {
				if !errors.Is(err, postgres.ErrAlarmSweepItemPoison) {
					h.workerFatal("ALARM_SWEEP_FATAL")
				}
			}
		})
	}
	if plan.compute != nil {
		token, _ := roleToken(loaded.Config, "compute")
		h.launch(work, func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-intake.Done():
					return
				case now := <-ticker.C:
					if err := plan.compute.RunDue(work, plan.store, postgres.ConsumerRoleToken{OwnerID: token.OwnerID, Epoch: token.Epoch}, now.UTC()); err != nil {
						if errors.Is(err, compute.ErrUnitBusiness) {
							h.State.ReportDegraded("COMPUTE_UNIT_BUSINESS")
						} else {
							h.workerFatal("COMPUTE_SCHEDULE_FATAL")
							return
						}
					}
					if err := plan.compute.SweepDebounces(work, plan.store, postgres.ConsumerRoleToken{OwnerID: token.OwnerID, Epoch: token.Epoch}, now.UTC()); err != nil {
						if errors.Is(err, compute.ErrUnitBusiness) {
							h.State.ReportDegraded("COMPUTE_UNIT_BUSINESS")
						} else {
							h.workerFatal("COMPUTE_DEBOUNCE_FATAL")
							return
						}
					}
				}
			}
		})
	}
}
func (h *Host) launch(ctx context.Context, fn func()) {
	h.wg.Add(1)
	go func() { defer h.wg.Done(); fn() }()
}
func (h *Host) workerFatal(code string) {
	h.State.SetState(httpapi.Failed, httpapi.Unavailable, code)
	h.mu.Lock()
	cancel := h.cancel
	intakeCancel := h.intakeCancel
	h.mu.Unlock()
	h.fatalOnce.Do(func() { close(h.fatal) })
	if intakeCancel != nil {
		intakeCancel()
	}
	if cancel != nil {
		cancel()
	}
}
