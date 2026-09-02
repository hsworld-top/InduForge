// Package command 消费隔离的人工计算命令流；命令不经过 data ingress。
package command

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/indu-forge/runtime-engine/internal/compute"
	"github.com/indu-forge/runtime-engine/internal/loader"
	"github.com/indu-forge/runtime-engine/internal/model"
	"github.com/indu-forge/runtime-engine/internal/store/postgres"
	"github.com/indu-forge/runtime-engine/internal/transport/jetstream"
)

type Envelope struct {
	SchemaVersion  string `json:"schemaVersion"`
	Subject        string `json:"subject"`
	DeploymentID   string `json:"deploymentId"`
	AccountID      string `json:"accountId"`
	CommandID      string `json:"commandId"`
	ComputeID      string `json:"computeId"`
	RequestedBy    string `json:"requestedBy"`
	RequestedAt    string `json:"requestedAt"`
	BindingEpoch   int64  `json:"bindingEpoch"`
	IdempotencyKey string `json:"idempotencyKey"`
}

type Consumer struct {
	config   model.EngineConfig
	consumer model.Consumer
	token    postgres.ConsumerRoleToken
	store    *postgres.Store
	handler  *compute.Handler
}

func New(config model.EngineConfig, consumer model.Consumer, token postgres.ConsumerRoleToken, store *postgres.Store, handler *compute.Handler) (*Consumer, error) {
	if store == nil || handler == nil || consumer.Role != "compute" || consumer.Stream != config.JetStream.CommandStream || consumer.FilterSubject != "compute.command.>" || token.OwnerID == "" || token.Epoch < 1 {
		return nil, errors.New("command consumer 配置非法")
	}
	return &Consumer{config: config, consumer: consumer, token: token, store: store, handler: handler}, nil
}

func (c *Consumer) Run(intake, work context.Context, client *jetstream.Client) error {
	if c == nil || client == nil {
		return errors.New("command consumer 依赖非法")
	}
	for {
		if intake.Err() != nil {
			return nil
		}
		messages, err := client.Fetch(intake, c.consumer.Stream, c.consumer.DurableName, 32, 5*time.Second)
		if err != nil {
			if intake.Err() != nil {
				return nil
			}
			return err
		}
		for _, message := range messages {
			if err := c.handle(work, message); err != nil {
				return err
			}
			if err := message.Ack(work); err != nil {
				return err
			}
		}
	}
}

func (c *Consumer) handle(ctx context.Context, message jetstream.DeliveredMessage) error {
	var command Envelope
	if err := loader.ValidateComputeCommand(message.Body); err != nil {
		return fmt.Errorf("command schema 无效: %w", err)
	}
	if err := json.Unmarshal(message.Body, &command); err != nil {
		return errors.New("command JSON 无效")
	}
	if command.Subject != message.Subject || command.Subject != "compute.command."+c.config.DeploymentID || command.DeploymentID != c.config.DeploymentID || command.AccountID != c.config.AccountID {
		return errors.New("command 部署边界不匹配")
	}
	manual, ok := manualFence(c.config)
	if !ok || command.BindingEpoch != manual.Epoch {
		return errors.New("command binding epoch 过期")
	}
	requestedAt, err := time.Parse(time.RFC3339Nano, command.RequestedAt)
	if err != nil || requestedAt.Location() != time.UTC {
		return errors.New("command requestedAt 非法")
	}
	audit, err := c.store.ComputeCommandStatus(ctx, command.DeploymentID, command.CommandID)
	if err != nil {
		return err
	}
	if audit.Status == "succeeded" || audit.Status == "failed" {
		return nil
	}
	if err := c.store.SetComputeCommandStatus(ctx, command.DeploymentID, command.CommandID, "running", ""); err != nil {
		return err
	}
	sum := sha256.Sum256([]byte(command.CommandID))
	eventID := hex.EncodeToString(sum[:])
	_, err = c.store.ProcessMessage(ctx, postgres.Message{DeploymentID: c.config.DeploymentID, AccountID: c.config.AccountID, ConsumerKey: c.consumer.ConsumerKey, Role: "compute", Token: c.token, ProducerKey: "runtime-api", ProducerToken: postgres.ProducerToken{OwnerID: manual.OwnerID, Epoch: manual.Epoch}, EventID: eventID, RawBody: message.Body, Subject: message.Subject, CheckpointPosition: int64(message.StreamSequence), DeliveryCount: int(message.DeliveryCount), OccurredAt: message.OccurredAt}, func(ctx context.Context, tx *postgres.BusinessTx) error {
		return c.handler.ExecuteManual(ctx, tx, command.ComputeID, eventID, requestedAt)
	}, postgres.ProcessOptions{})
	if err != nil {
		_ = c.store.SetComputeCommandStatus(context.Background(), command.DeploymentID, command.CommandID, "failed", "execution-failed")
		return err
	}
	return c.store.SetComputeCommandStatus(ctx, command.DeploymentID, command.CommandID, "succeeded", "")
}

func manualFence(config model.EngineConfig) (model.Ownership, bool) {
	for _, producer := range config.ProducerAssignments {
		if producer.ProducerType == "manual" && producer.ManualID == "runtime-api" {
			return producer.Ownership, true
		}
	}
	return model.Ownership{}, false
}
