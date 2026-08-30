package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5"
)

type ProducerFence struct {
	DeploymentID string
	ProducerKey  string
	OwnerID      string
	Epoch        int64
	Version      int64
}

type ProducerHandler func(context.Context, *BusinessTx) error

// VerifyProducerInTx 仅校验 producer fence；它与 consumer role fence 保持两个类型与
// 命名空间，供 alarm 等不分配顶层 sequence 的 producer 使用。
func (b *BusinessTx) VerifyProducerInTx(ctx context.Context, producerKey string, token ProducerToken) error {
	if b == nil || b.tx == nil || validToken(b.deploymentID, producerKey, token.OwnerID, token.Epoch) != nil {
		return ErrInvalidInput
	}
	if err := b.ensureOpen(); err != nil {
		return err
	}
	var owner string
	var epoch int64
	err := b.tx.QueryRow(ctx, `SELECT owner_id,epoch FROM runtime_engine.producer_fence WHERE deployment_id=$1 AND producer_key=$2 FOR UPDATE`, b.deploymentID, producerKey).Scan(&owner, &epoch)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrFenceStale
	}
	if err != nil {
		return err
	}
	if owner != token.OwnerID || epoch != token.Epoch {
		return ErrFenceStale
	}
	return nil
}

// RunProducerTransaction 为合法 producer 提供同一 fence 事务内的受控状态/outbox 写入口。
func (s *Store) RunProducerTransaction(ctx context.Context, deploymentID, producerKey string, token ProducerToken, handler ProducerHandler) error {
	if err := validToken(deploymentID, producerKey, token.OwnerID, token.Epoch); err != nil {
		return err
	}
	if err := s.ensureOpen(); err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var owner string
	var epoch int64
	err = tx.QueryRow(ctx, `SELECT owner_id,epoch FROM runtime_engine.producer_fence WHERE deployment_id=$1 AND producer_key=$2 FOR UPDATE`, deploymentID, producerKey).Scan(&owner, &epoch)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrFenceStale
	}
	if err != nil {
		return err
	}
	if owner != token.OwnerID || epoch != token.Epoch {
		return ErrFenceStale
	}
	if handler != nil {
		if err = handler(ctx, &BusinessTx{store: s, tx: tx, deploymentID: deploymentID}); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// ActivateProducer 是 producer ownership 的显式 CAS；它从不读取或写入 consumer role_fence。
func (s *Store) ActivateProducer(ctx context.Context, deploymentID, producerKey string, token ProducerToken, expectedVersion int64) (ProducerFence, error) {
	if err := validToken(deploymentID, producerKey, token.OwnerID, token.Epoch); err != nil || expectedVersion < 0 {
		return ProducerFence{}, fmt.Errorf("%w: producer assignment", ErrInvalidInput)
	}
	if err := s.ensureOpen(); err != nil {
		return ProducerFence{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ProducerFence{}, err
	}
	defer tx.Rollback(ctx)
	var current ProducerFence
	err = tx.QueryRow(ctx, `SELECT deployment_id,producer_key,owner_id,epoch,version FROM runtime_engine.producer_fence WHERE deployment_id=$1 AND producer_key=$2 FOR UPDATE`, deploymentID, producerKey).Scan(&current.DeploymentID, &current.ProducerKey, &current.OwnerID, &current.Epoch, &current.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		if expectedVersion != 0 {
			return ProducerFence{}, fmt.Errorf("%w: 初始 producer assignment 的 expectedVersion 必须为 0", ErrFenceRejected)
		}
		current = ProducerFence{DeploymentID: deploymentID, ProducerKey: producerKey, OwnerID: token.OwnerID, Epoch: token.Epoch, Version: 1}
		_, err = tx.Exec(ctx, `INSERT INTO runtime_engine.producer_fence (deployment_id,producer_key,owner_id,epoch,version) VALUES ($1,$2,$3,$4,$5)`, deploymentID, producerKey, token.OwnerID, token.Epoch, current.Version)
	} else if err != nil {
		return ProducerFence{}, err
	} else if current.OwnerID == token.OwnerID && current.Epoch == token.Epoch {
		// 同一 producer token 的重复 assignment 提交幂等。
	} else if token.Epoch <= current.Epoch {
		return ProducerFence{}, fmt.Errorf("%w: producer epoch 只能严格提升", ErrFenceRejected)
	} else if current.Version != expectedVersion {
		return ProducerFence{}, fmt.Errorf("%w: producer expectedVersion 不匹配", ErrFenceRejected)
	} else {
		current.OwnerID, current.Epoch, current.Version = token.OwnerID, token.Epoch, current.Version+1
		_, err = tx.Exec(ctx, `UPDATE runtime_engine.producer_fence SET owner_id=$3,epoch=$4,version=$5,activated_at=now() WHERE deployment_id=$1 AND producer_key=$2`, deploymentID, producerKey, current.OwnerID, current.Epoch, current.Version)
	}
	if err != nil {
		return ProducerFence{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return ProducerFence{}, err
	}
	return current, nil
}

// VerifyProducer 仅在 producer fence 命名空间内校验已提交 ownership，供激活前检查和健康诊断使用。
func (s *Store) VerifyProducer(ctx context.Context, deploymentID, producerKey string, token ProducerToken) error {
	if err := validToken(deploymentID, producerKey, token.OwnerID, token.Epoch); err != nil {
		return err
	}
	if err := s.ensureOpen(); err != nil {
		return err
	}
	var owner string
	var epoch int64
	err := s.pool.QueryRow(ctx, `SELECT owner_id,epoch FROM runtime_engine.producer_fence WHERE deployment_id=$1 AND producer_key=$2`, deploymentID, producerKey).Scan(&owner, &epoch)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrFenceStale
	}
	if err != nil {
		return err
	}
	if owner != token.OwnerID || epoch != token.Epoch {
		return ErrFenceStale
	}
	return err
}

// NextProducerSequence 在 producer fence 内原子分配 sequence。新 epoch 的首值为 0，旧 token 一律拒绝。
func (s *Store) NextProducerSequence(ctx context.Context, deploymentID, producerKey string, token ProducerToken) (int64, error) {
	if err := validToken(deploymentID, producerKey, token.OwnerID, token.Epoch); err != nil {
		return 0, err
	}
	if err := s.ensureOpen(); err != nil {
		return 0, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	var owner string
	var epoch int64
	err = tx.QueryRow(ctx, `SELECT owner_id,epoch FROM runtime_engine.producer_fence WHERE deployment_id=$1 AND producer_key=$2 FOR UPDATE`, deploymentID, producerKey).Scan(&owner, &epoch)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrFenceStale
	}
	if err != nil {
		return 0, err
	}
	if owner != token.OwnerID || epoch != token.Epoch {
		return 0, ErrFenceStale
	}
	var storedOwner string
	var storedEpoch, last int64
	err = tx.QueryRow(ctx, `SELECT owner_id,epoch,last_sequence FROM runtime_engine.producer_sequence WHERE deployment_id=$1 AND producer_key=$2 FOR UPDATE`, deploymentID, producerKey).Scan(&storedOwner, &storedEpoch, &last)
	var next int64
	if errors.Is(err, pgx.ErrNoRows) {
		next = 0
		_, err = tx.Exec(ctx, `INSERT INTO runtime_engine.producer_sequence (deployment_id,producer_key,owner_id,epoch,last_sequence) VALUES ($1,$2,$3,$4,$5)`, deploymentID, producerKey, token.OwnerID, token.Epoch, next)
	} else if err != nil {
		return 0, err
	} else if storedOwner == token.OwnerID && storedEpoch == token.Epoch {
		if last == math.MaxInt64 {
			return 0, ErrSequenceExhausted
		}
		next = last + 1
		_, err = tx.Exec(ctx, `UPDATE runtime_engine.producer_sequence SET last_sequence=$3,updated_at=now() WHERE deployment_id=$1 AND producer_key=$2 AND owner_id=$4 AND epoch=$5`, deploymentID, producerKey, next, token.OwnerID, token.Epoch)
	} else if storedEpoch < token.Epoch {
		next = 0
		_, err = tx.Exec(ctx, `UPDATE runtime_engine.producer_sequence SET owner_id=$3,epoch=$4,last_sequence=$5,updated_at=now() WHERE deployment_id=$1 AND producer_key=$2`, deploymentID, producerKey, token.OwnerID, token.Epoch, next)
	} else {
		return 0, ErrFenceStale
	}
	if err != nil {
		return 0, err
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, err
	}
	return next, nil
}
