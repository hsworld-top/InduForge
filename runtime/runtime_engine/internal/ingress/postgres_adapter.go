package ingress

import (
	"context"
	"strings"

	"github.com/indu-forge/runtime-engine/internal/eventid"
	"github.com/indu-forge/runtime-engine/internal/model"
	"github.com/indu-forge/runtime-engine/internal/store/postgres"
)

// PostgresHandler 是后续 writer/compute/alarm 算法接入的受事务保护扩展点；本轮不实现业务算法。
type PostgresHandler func(context.Context, *postgres.BusinessTx, ValidatedMessage) error

// PostgresAdapter 将 ingress 窄接口映射到已收口的 Store API。业务 handler 仅在 ProcessMessage 的 pgx tx 内执行。
type PostgresAdapter struct {
	store   *postgres.Store
	config  model.EngineConfig
	handler PostgresHandler
}

func NewPostgresAdapter(store *postgres.Store, config model.EngineConfig, handler PostgresHandler) *PostgresAdapter {
	return &PostgresAdapter{store: store, config: config, handler: handler}
}
func (a *PostgresAdapter) Process(ctx context.Context, message ValidatedMessage) (ProcessResult, error) {
	result, err := a.store.ProcessMessage(ctx, postgres.Message{DeploymentID: message.Event.DeploymentID, AccountID: message.Event.AccountID, ConsumerKey: message.Consumer.ConsumerKey, Role: message.Consumer.Role, Token: postgres.ConsumerRoleToken{OwnerID: message.Token.OwnerID, Epoch: message.Token.Epoch}, EventID: message.Event.EventID, RawBody: message.RawBody, Subject: message.Event.Subject, CheckpointPosition: message.StreamPosition, DeliveryCount: message.DeliveryCount, OccurredAt: message.OccurredAt}, func(ctx context.Context, tx *postgres.BusinessTx) error {
		if a.handler == nil {
			return nil
		}
		return a.handler(ctx, tx, message)
	}, postgres.ProcessOptions{OnCollision: func(ctx context.Context, tx *postgres.BusinessTx, collision postgres.CollisionInfo) error {
		failure, err := buildPermanentFailureForCollision(a.config, message, collision.DLQID)
		if err != nil {
			return err
		}
		_, _, err = tx.Enqueue(ctx, postgres.OutboxMessage{DeploymentID: failure.DeploymentID, DedupeKey: collision.DLQID, Subject: failure.DeadLetterSubject, Payload: failure.DLQPayload})
		return err
	}})
	if err != nil {
		return 0, err
	}
	if result == postgres.Duplicate {
		return Duplicate, nil
	}
	if result == postgres.Collision {
		return Collision, nil
	}
	return Processed, nil
}
func (a *PostgresAdapter) ProcessPermanentFailure(ctx context.Context, failure PermanentFailure) (PermanentResult, error) {
	err := a.store.ProcessPermanentFailure(ctx, postgres.PermanentFailure{DeploymentID: failure.DeploymentID, AccountID: failure.AccountID, ConsumerKey: failure.ConsumerKey, Role: failure.Role, Token: postgres.ConsumerRoleToken{OwnerID: failure.ConsumerToken.OwnerID, Epoch: failure.ConsumerToken.Epoch}, DLQID: failure.DLQID, EventID: failure.EventID, Subject: failure.OriginalSubject, RawBody: failure.RawBody, BodySHA256: strings.TrimPrefix(failure.BodySHA256, "sha256:"), ReasonCode: postgres.FailureReasonCode(failure.ReasonCode), DeliveryCount: failure.DeliveryCount, CheckpointPosition: failure.StreamPosition, OccurredAt: failure.OccurredAt}, func(ctx context.Context, tx *postgres.BusinessTx, _ postgres.PermanentFailure) error {
		_, _, err := tx.Enqueue(ctx, postgres.OutboxMessage{DeploymentID: failure.DeploymentID, DedupeKey: failure.DLQID, Subject: failure.DeadLetterSubject, Payload: failure.DLQPayload})
		return err
	})
	return PermanentStored, err
}
func buildPermanentFailureForCollision(config model.EngineConfig, message ValidatedMessage, dlqID string) (PermanentFailure, error) {
	digest := "sha256:" + eventid.BodySHA256(message.RawBody)
	eventID := message.Event.EventID
	payload, err := encodeDLQPayload(dlqID, config.DeploymentID, config.AccountID, message.Consumer.ConsumerKey, message.Event.Subject, &eventID, "event-id-collision", message.DeliveryCount, digest, message.RawBody, message.OccurredAt)
	if err != nil {
		return PermanentFailure{}, err
	}
	return PermanentFailure{DeploymentID: message.Event.DeploymentID, AccountID: message.Event.AccountID, ConsumerKey: message.Consumer.ConsumerKey, Role: message.Consumer.Role, ConsumerToken: message.Token, OriginalSubject: message.Event.Subject, EventID: &eventID, ReasonCode: "event-id-collision", DeliveryCount: message.DeliveryCount, BodySHA256: digest, RawBody: message.RawBody, OccurredAt: message.OccurredAt, DeadLetterSubject: message.Consumer.DeadLetterSubject, DLQID: dlqID, DLQPayload: payload, StreamPosition: message.StreamPosition}, nil
}
