package project

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"net/http"
	"strconv"
	"strings"
)

type authoringEpochContextKey struct{}
type AuthoringEpochConflict struct{ ProjectID, Current string }

func (e *AuthoringEpochConflict) Error() string {
	return "工程开发态已变化，请刷新后重试"
}
func AuthoringEpochConflictData(err error) (map[string]any, bool) {
	var conflict *AuthoringEpochConflict
	if !errors.As(err, &conflict) {
		return nil, false
	}
	return map[string]any{"projectId": conflict.ProjectID, "currentAuthoringEpoch": conflict.Current, "action": "reload"}, true
}

func ResolveAuthoringEpoch(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := strings.TrimSpace(r.Header.Get("X-InduForge-Authoring-Epoch"))
		if raw != "" {
			r = r.WithContext(context.WithValue(r.Context(), authoringEpochContextKey{}, raw))
		}
		next.ServeHTTP(w, r)
	})
}
func AuthoringEpochFromContext(ctx context.Context) string {
	value, _ := ctx.Value(authoringEpochContextKey{}).(string)
	return value
}
func FormatAuthoringEpoch(value int64) string { return "epoch-" + strconv.FormatInt(value, 10) }
func ParseAuthoringEpoch(value string) (int64, error) {
	if !strings.HasPrefix(value, "epoch-") {
		return 0, fmt.Errorf("工程编辑代次无效")
	}
	n, err := strconv.ParseInt(strings.TrimPrefix(value, "epoch-"), 10, 64)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("工程编辑代次无效")
	}
	return n, nil
}

func (r *PostgreSQLRepository) RequireAuthoringEpoch(ctx context.Context, tenantID, projectID, expected string) error {
	epoch, parseErr := ParseAuthoringEpoch(expected)
	tenant, err := parseUUID(tenantID)
	if err != nil {
		return ErrNotFound
	}
	project, err := parseUUID(projectID)
	if err != nil {
		return ErrNotFound
	}
	var current int64
	var busy bool
	err = r.pool.QueryRow(ctx, `SELECT p.authoring_epoch,EXISTS(SELECT 1 FROM authoring_project_fences f WHERE f.project_id=p.id AND f.expires_at>now()) OR EXISTS(SELECT 1 FROM authoring_restore_tasks t WHERE t.project_id=p.id AND t.state IN ('queued','staging','restoring_workspace','restoring_scenes','restoring_data','finalizing','compensating')) FROM projects p WHERE p.id=$1 AND p.tenant_id=$2 AND p.status<>'deleted'`, project, tenant).Scan(&current, &busy)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if parseErr != nil || busy || current != epoch {
		return &AuthoringEpochConflict{ProjectID: projectID, Current: FormatAuthoringEpoch(current)}
	}
	return nil
}
