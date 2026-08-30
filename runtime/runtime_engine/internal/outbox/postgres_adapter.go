package outbox

import (
	"context"
	"time"

	"github.com/indu-forge/runtime-engine/internal/store/postgres"
)

// PostgresAdapter 仅转换传输层 DTO 和白名单 RetryCode；不暴露 pgx 或连接池。
type PostgresAdapter struct{ store *postgres.Store }

func NewPostgresAdapter(store *postgres.Store) *PostgresAdapter {
	return &PostgresAdapter{store: store}
}
func (a *PostgresAdapter) ClaimOutbox(ctx context.Context, deployment, owner string, limit int, lease time.Duration) ([]Record, error) {
	records, err := a.store.ClaimOutbox(ctx, deployment, owner, limit, lease)
	if err != nil {
		return nil, err
	}
	out := make([]Record, len(records))
	for i, r := range records {
		out[i] = Record{ID: r.ID, DeploymentID: r.DeploymentID, DedupeKey: r.DedupeKey, Subject: r.Subject, Headers: r.Headers, Payload: r.Payload, PayloadSHA256: r.PayloadSHA256, LeaseToken: r.LeaseToken, AttemptCount: r.AttemptCount}
	}
	return out, nil
}
func (a *PostgresAdapter) MarkPublished(ctx context.Context, id int64, lease string) error {
	return a.store.MarkPublished(ctx, id, lease)
}
func (a *PostgresAdapter) RetryOutbox(ctx context.Context, id int64, lease string, next time.Time, code RetryCode) error {
	return a.store.RetryOutbox(ctx, id, lease, next, postgres.RetryCode(code))
}
