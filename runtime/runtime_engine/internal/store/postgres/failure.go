package postgres

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/indu-forge/runtime-engine/internal/eventid"
	"github.com/jackc/pgx/v5"
)

type FailureReasonCode string

const (
	FailurePermanentValidation FailureReasonCode = "permanent-validation"
	FailureEventIDCollision    FailureReasonCode = "event-id-collision"
	FailureMaxDeliver          FailureReasonCode = "max-deliver"
	FailureHandler             FailureReasonCode = "handler-failure"
)

// PermanentFailure 是可安全隔离的最小证据；不承载 endpoint、Secret、DSN 或任意错误文本。
type PermanentFailure struct {
	DeploymentID       string
	ConsumerKey        string
	Role               string
	Token              ConsumerRoleToken
	DLQID              string
	AccountID          string
	EventID            *string // 解析失败的 body 可以没有合法 eventId。
	Subject            string
	RawBody            []byte
	BodySHA256         string
	ReasonCode         FailureReasonCode
	DeliveryCount      int
	CheckpointPosition int64
	OccurredAt         time.Time
}

type PermanentFailureHandler func(context.Context, *BusinessTx, PermanentFailure) error

// ProcessPermanentFailure 以 role fence、失败证据、DLQ outbox 与 checkpoint 的单一事务持久化最终失败。
func (s *Store) ProcessPermanentFailure(ctx context.Context, failure PermanentFailure, enqueueDLQ PermanentFailureHandler) error {
	if err := validPermanentFailure(failure); err != nil {
		return err
	}
	if enqueueDLQ == nil {
		return fmt.Errorf("%w: permanent failure 必须写入 DLQ outbox", ErrInvalidInput)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	msg := Message{DeploymentID: failure.DeploymentID, Role: failure.Role, Token: failure.Token}
	if err := assertConsumerFence(ctx, tx, msg); err != nil {
		return err
	}
	var inserted int
	err = tx.QueryRow(ctx, `INSERT INTO runtime_engine.processing_failure (deployment_id,consumer_key,dlq_id,event_id,reason_code,delivery_count,jetstream_position,subject,body_sha256,raw_body,occurred_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) ON CONFLICT (deployment_id,dlq_id) DO NOTHING RETURNING 1`, failure.DeploymentID, failure.ConsumerKey, failure.DLQID, failure.EventID, failure.ReasonCode, failure.DeliveryCount, failure.CheckpointPosition, failure.Subject, failure.BodySHA256, failure.RawBody, failure.OccurredAt.UTC()).Scan(&inserted)
	if err != nil && err != pgx.ErrNoRows {
		return err
	}
	if inserted == 0 {
		var consumer, reason, existingDigest string
		var eventID *string
		if err = tx.QueryRow(ctx, `SELECT consumer_key,event_id,reason_code,body_sha256 FROM runtime_engine.processing_failure WHERE deployment_id=$1 AND dlq_id=$2`, failure.DeploymentID, failure.DLQID).Scan(&consumer, &eventID, &reason, &existingDigest); err != nil {
			return err
		}
		if consumer != failure.ConsumerKey || !sameOptionalString(eventID, failure.EventID) || reason != string(failure.ReasonCode) || existingDigest != failure.BodySHA256 {
			return fmt.Errorf("%w: dlqId 身份冲突", ErrOutboxConflict)
		}
	} else if err = enqueueDLQ(ctx, &BusinessTx{store: s, tx: tx, deploymentID: failure.DeploymentID}, failure); err != nil {
		return err
	}
	if err = ensureOutboxTx(ctx, tx, failure.DeploymentID, failure.DLQID); err != nil {
		return err
	}
	if err = advanceCheckpointTx(ctx, tx, failure.DeploymentID, failure.ConsumerKey, failure.CheckpointPosition); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func validPermanentFailure(f PermanentFailure) error {
	if err := validToken(f.DeploymentID, f.Role, f.Token.OwnerID, f.Token.Epoch); err != nil || f.AccountID == "" || f.ConsumerKey == "" || f.DLQID == "" || f.Subject == "" || len(f.Subject) > 4096 || len(f.RawBody) > 1<<20 || f.DeliveryCount < 1 || f.CheckpointPosition < 0 || f.OccurredAt.IsZero() || !validFailureReason(f.ReasonCode) {
		return fmt.Errorf("%w: permanent failure", ErrInvalidInput)
	}
	if len(f.BodySHA256) != 64 || f.BodySHA256 != strings.ToLower(f.BodySHA256) {
		return fmt.Errorf("%w: body digest", ErrInvalidInput)
	}
	if _, err := hex.DecodeString(f.BodySHA256); err != nil || eventid.BodySHA256(f.RawBody) != f.BodySHA256 {
		return fmt.Errorf("%w: body digest", ErrInvalidInput)
	}
	if f.EventID != nil {
		if err := validEventID(*f.EventID); err != nil {
			return err
		}
	}
	expected, err := runtimeDLQID(f.DeploymentID, f.ConsumerKey, f.Subject, optionalString(f.EventID), f.ReasonCode, f.BodySHA256)
	if err != nil || f.DLQID != expected {
		return fmt.Errorf("%w: dlqId", ErrInvalidInput)
	}
	return nil
}

func validFailureReason(code FailureReasonCode) bool {
	return code == FailurePermanentValidation || code == FailureEventIDCollision || code == FailureMaxDeliver || code == FailureHandler
}

func sameOptionalString(left, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func optionalString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func ensureOutboxTx(ctx context.Context, tx pgx.Tx, deploymentID, dedupeKey string) error {
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM runtime_engine.transactional_outbox WHERE deployment_id=$1 AND dedupe_key=$2)`, deploymentID, dedupeKey).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%w: failure 缺少 DLQ outbox", ErrOutboxConflict)
	}
	return nil
}

func validEventID(id string) error {
	if len(id) != 64 || id != strings.ToLower(id) {
		return fmt.Errorf("%w: eventId", ErrInvalidInput)
	}
	if _, err := hex.DecodeString(id); err != nil {
		return fmt.Errorf("%w: eventId", ErrInvalidInput)
	}
	return nil
}
