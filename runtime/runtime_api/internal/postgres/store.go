package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/indu-forge/runtime-api/internal/runtimeview"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct{ pool *pgxpool.Pool }

func Open(ctx context.Context, dsn string) (*Store, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, errors.New("PostgreSQL DSN 不能为空")
	}
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("解析 PostgreSQL DSN: %w", err)
	}
	config.MaxConns = 8
	config.MinConns = 1
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("连接 PostgreSQL: %w", err)
	}
	store := &Store{pool: pool}
	if err := store.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() {
	if s != nil && s.pool != nil {
		s.pool.Close()
	}
}

func (s *Store) Ping(ctx context.Context) error {
	if s == nil || s.pool == nil {
		return errors.New("PostgreSQL store 未初始化")
	}
	var version string
	if err := s.pool.QueryRow(ctx, `SELECT version FROM runtime_engine.schema_meta WHERE version='runtime_engine.v1'`).Scan(&version); err != nil {
		return fmt.Errorf("RuntimeEngine PostgreSQL schema 不可用: %w", err)
	}
	return nil
}

// ReserveManualSequence 在同一数据库事务中复核 deployment 级 Runtime API producer fence 并分配单调 sequence。
func (s *Store) ReserveManualSequence(ctx context.Context, deploymentID string, epoch int64) (int64, error) {
	if s == nil || s.pool == nil || strings.TrimSpace(deploymentID) == "" || epoch < 1 {
		return 0, errors.New("manual sequence 参数非法")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	var owner string
	var fencedEpoch int64
	if err = tx.QueryRow(ctx, `SELECT owner_id,epoch FROM runtime_engine.producer_fence WHERE deployment_id=$1 AND producer_key='runtime-api' FOR UPDATE`, deploymentID).Scan(&owner, &fencedEpoch); err != nil {
		return 0, fmt.Errorf("读取 manual producer fence: %w", err)
	}
	if owner != "runtime-api" || fencedEpoch != epoch {
		return 0, errors.New("manual producer fence 已失效")
	}
	var sequence int64
	err = tx.QueryRow(ctx, `INSERT INTO runtime_engine.producer_sequence (deployment_id,producer_key,owner_id,epoch,last_sequence) VALUES ($1,'runtime-api','runtime-api',$2,1) ON CONFLICT (deployment_id,producer_key) DO UPDATE SET last_sequence=runtime_engine.producer_sequence.last_sequence+1,owner_id=EXCLUDED.owner_id,epoch=EXCLUDED.epoch,updated_at=now() WHERE runtime_engine.producer_sequence.owner_id='runtime-api' AND runtime_engine.producer_sequence.epoch=EXCLUDED.epoch RETURNING last_sequence`, deploymentID, epoch).Scan(&sequence)
	if err != nil {
		return 0, fmt.Errorf("分配 manual sequence: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, err
	}
	return sequence, nil
}

func (s *Store) Current(ctx context.Context, deploymentID, pointID string) (runtimeview.PointCurrent, error) {
	var item runtimeview.PointCurrent
	var value []byte
	err := s.pool.QueryRow(ctx, `
		SELECT point_id::text,value,quality,source_timestamp,server_timestamp,sequence,event_id,updated_at
		FROM runtime_engine.point_current
		WHERE deployment_id=$1 AND point_id=$2::uuid`, deploymentID, pointID).Scan(
		&item.PointID, &value, &item.Quality, &item.SourceTimestamp, &item.ServerTimestamp,
		&item.Sequence, &item.EventID, &item.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return runtimeview.PointCurrent{}, runtimeview.ErrNotFound
	}
	if err != nil {
		return runtimeview.PointCurrent{}, fmt.Errorf("查询当前值: %w", err)
	}
	item.Value = normalizeJSON(value)
	return item, nil
}

func (s *Store) History(ctx context.Context, deploymentID, pointID string, query runtimeview.HistoryQuery) ([]runtimeview.PointSample, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	rows, err := s.pool.Query(ctx, `
		SELECT point_id::text,value,quality,source_timestamp,server_timestamp,sequence,event_id,received_at
		FROM runtime_engine.point_history
		WHERE deployment_id=$1 AND point_id=$2::uuid
		  AND ($3::timestamptz IS NULL OR source_timestamp >= $3)
		  AND ($4::timestamptz IS NULL OR source_timestamp <= $4)
		ORDER BY epoch DESC,source_timestamp DESC,sequence DESC
		LIMIT $5`, deploymentID, pointID, query.From, query.To, limit)
	if err != nil {
		return nil, fmt.Errorf("查询历史值: %w", err)
	}
	defer rows.Close()
	result := make([]runtimeview.PointSample, 0, limit)
	for rows.Next() {
		var item runtimeview.PointSample
		var value []byte
		if err := rows.Scan(&item.PointID, &value, &item.Quality, &item.SourceTimestamp, &item.ServerTimestamp, &item.Sequence, &item.EventID, &item.ReceivedAt); err != nil {
			return nil, fmt.Errorf("读取历史值: %w", err)
		}
		item.Value = normalizeJSON(value)
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历历史值: %w", err)
	}
	return result, nil
}

func (s *Store) AlarmStates(ctx context.Context, deploymentID string, limit int) ([]runtimeview.AlarmState, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	rows, err := s.pool.Query(ctx, `
		SELECT alarm_item_id::text,alarm_revision,state,version,updated_at
		FROM runtime_engine.alarm_item_state
		WHERE deployment_id=$1
		ORDER BY updated_at DESC,alarm_item_id
		LIMIT $2`, deploymentID, limit)
	if err != nil {
		return nil, fmt.Errorf("查询报警状态: %w", err)
	}
	defer rows.Close()
	result := make([]runtimeview.AlarmState, 0, limit)
	for rows.Next() {
		var item runtimeview.AlarmState
		var state []byte
		if err := rows.Scan(&item.AlarmItemID, &item.Revision, &state, &item.Version, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("读取报警状态: %w", err)
		}
		item.State = normalizeJSON(state)
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历报警状态: %w", err)
	}
	return result, nil
}

func normalizeJSON(value []byte) json.RawMessage {
	if len(value) == 0 || string(value) == "null" {
		return json.RawMessage("null")
	}
	return append(json.RawMessage(nil), value...)
}
