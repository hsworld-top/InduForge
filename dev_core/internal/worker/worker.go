package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	dbsqlc "github.com/indu-forge/dev_core/internal/platform/db/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	ListStaleCommands(context.Context) ([]StaleCommand, error)
	RecoverStaleCommand(context.Context, string, []byte) (bool, error)
}

type ArtifactSigner interface {
	PresignGet(context.Context, string, time.Duration) (string, error)
}

type StaleCommand struct {
	ID          string
	Payload     []byte
	Attempts    int32
	MaxAttempts int32
}

type PostgreSQLRepository struct{ queries *dbsqlc.Queries }

func NewPostgreSQLRepository(pool *pgxpool.Pool) *PostgreSQLRepository {
	return &PostgreSQLRepository{queries: dbsqlc.New(pool)}
}

func (r *PostgreSQLRepository) ListStaleCommands(ctx context.Context) ([]StaleCommand, error) {
	rows, err := r.queries.ListStaleNodeCommands(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]StaleCommand, 0, len(rows))
	for _, row := range rows {
		items = append(items, StaleCommand{ID: uuid.UUID(row.ID.Bytes).String(), Payload: row.Payload, Attempts: row.Attempts, MaxAttempts: row.MaxAttempts})
	}
	return items, nil
}

func (r *PostgreSQLRepository) RecoverStaleCommand(ctx context.Context, commandID string, payload []byte) (bool, error) {
	id, err := uuid.Parse(commandID)
	if err != nil {
		return false, err
	}
	rows, err := r.queries.RecoverStaleNodeCommand(ctx, dbsqlc.RecoverStaleNodeCommandParams{Payload: payload, CommandID: pgtype.UUID{Bytes: id, Valid: true}})
	return rows > 0, err
}

type Worker struct {
	repository Repository
	signer     ArtifactSigner
	logger     *slog.Logger
	interval   time.Duration
}

func New(repository Repository, signer ArtifactSigner, logger *slog.Logger, interval time.Duration) *Worker {
	if logger == nil {
		logger = slog.Default()
	}
	if interval <= 0 {
		interval = 10 * time.Second
	}
	return &Worker{repository: repository, signer: signer, logger: logger, interval: interval}
}

// Run 在启动时和固定周期恢复超时命令；所有状态都持久化在 PostgreSQL，进程重启不会丢任务。
func (w *Worker) Run(ctx context.Context) {
	w.recover(ctx)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.recover(ctx)
		}
	}
}

func (w *Worker) recover(ctx context.Context) {
	commands, err := w.repository.ListStaleCommands(ctx)
	if err != nil {
		w.logger.Error("恢复超时节点命令失败", "error", err)
		return
	}
	var count int64
	for _, command := range commands {
		payload := command.Payload
		if command.Attempts < command.MaxAttempts {
			payload, err = w.refreshArtifactURL(ctx, payload)
			if err != nil {
				w.logger.Error("刷新部署工件签名失败", "commandId", command.ID, "error", err)
				continue
			}
		}
		recovered, recoverErr := w.repository.RecoverStaleCommand(ctx, command.ID, payload)
		if recoverErr != nil {
			w.logger.Error("恢复超时节点命令失败", "commandId", command.ID, "error", recoverErr)
			continue
		}
		if recovered {
			count++
		}
	}
	if count > 0 {
		w.logger.Info("已恢复超时节点命令", "count", count)
	}
}

func (w *Worker) refreshArtifactURL(ctx context.Context, raw []byte) ([]byte, error) {
	if w.signer == nil {
		return raw, nil
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	key, _ := payload["artifactKey"].(string)
	if key == "" {
		return raw, nil
	}
	url, err := w.signer.PresignGet(ctx, key, 30*time.Minute)
	if err != nil {
		return nil, err
	}
	payload["artifactUrl"] = url
	return json.Marshal(payload)
}
