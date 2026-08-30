package postgres

// Atomic deployment assignment activation.  This is intentionally separate
// from the older single-fence CAS helpers: the host must never make roles live
// while only a subset of producers has been fenced.

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/indu-forge/runtime-engine/internal/model"
	"github.com/jackc/pgx/v5"
)

// ActivateAssignments installs every trusted role and producer assignment in
// one transaction.  It serializes a deployment with a transaction advisory
// lock, keeps the role/producer namespaces separate even when names match,
// and needs no caller-provided expectedVersion.
func (s *Store) ActivateAssignments(ctx context.Context, config model.EngineConfig) error {
	if !validStableID(config.DeploymentID) {
		return fmt.Errorf("%w: assignments", ErrInvalidInput)
	}
	roles, producers, err := normalizeAssignments(config)
	if err != nil {
		return err
	}
	if err := s.ensureOpen(); err != nil {
		return err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, config.DeploymentID); err != nil {
		return err
	}
	for _, a := range roles {
		if err = activateRoleTx(ctx, tx, config.DeploymentID, a.Role, ConsumerRoleToken{OwnerID: a.Ownership.OwnerID, Epoch: a.Ownership.Epoch}); err != nil {
			return err
		}
	}
	for _, a := range producers {
		if err = activateProducerTx(ctx, tx, config.DeploymentID, a.key, ProducerToken{OwnerID: a.Ownership.OwnerID, Epoch: a.Ownership.Epoch}); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

type producerAssignment struct {
	key string
	model.ProducerAssignment
}

func normalizeAssignments(config model.EngineConfig) ([]model.RoleAssignment, []producerAssignment, error) {
	roles := append([]model.RoleAssignment(nil), config.RoleAssignments...)
	producers := make([]producerAssignment, 0, len(config.ProducerAssignments))
	seenRole, seenProducer := map[string]bool{}, map[string]bool{}
	for _, r := range roles {
		if validToken(config.DeploymentID, r.Role, r.Ownership.OwnerID, r.Ownership.Epoch) != nil || seenRole[r.Role] {
			return nil, nil, fmt.Errorf("%w: duplicate or invalid role assignment", ErrInvalidInput)
		}
		seenRole[r.Role] = true
	}
	for _, p := range config.ProducerAssignments {
		key, err := ProducerKey(p)
		if err != nil || validToken(config.DeploymentID, key, p.Ownership.OwnerID, p.Ownership.Epoch) != nil || seenProducer[key] {
			return nil, nil, fmt.Errorf("%w: duplicate or invalid producer assignment", ErrInvalidInput)
		}
		seenProducer[key] = true
		producers = append(producers, producerAssignment{key, p})
	}
	sort.Slice(roles, func(i, j int) bool { return roles[i].Role < roles[j].Role })
	sort.Slice(producers, func(i, j int) bool { return producers[i].key < producers[j].key })
	return roles, producers, nil
}

// ProducerKey is the frozen fence identity used by ingress and activation.
// Collector IDs are external producer identities; the Engine records their
// owner/epoch but never claims ownership on their behalf.
func ProducerKey(p model.ProducerAssignment) (string, error) {
	switch p.ProducerType {
	case "collector":
		if p.CollectorID != "" {
			return p.CollectorID, nil
		}
	case "compute":
		if p.ComputeID != "" {
			return p.ComputeID, nil
		}
	case "alarm":
		if p.Role == "alarm" {
			return "alarm", nil
		}
	}
	return "", errors.New("producer assignment key 非法")
}

func activateRoleTx(ctx context.Context, tx pgx.Tx, deployment, role string, token ConsumerRoleToken) error {
	var owner string
	var epoch, version int64
	err := tx.QueryRow(ctx, `SELECT owner_id,epoch,version FROM runtime_engine.role_fence WHERE deployment_id=$1 AND role=$2 FOR UPDATE`, deployment, role).Scan(&owner, &epoch, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		_, err = tx.Exec(ctx, `INSERT INTO runtime_engine.role_fence(deployment_id,role,owner_id,epoch,version) VALUES($1,$2,$3,$4,1)`, deployment, role, token.OwnerID, token.Epoch)
		return err
	}
	if err != nil {
		return err
	}
	if owner == token.OwnerID && epoch == token.Epoch {
		return nil
	}
	if token.Epoch <= epoch {
		return ErrFenceRejected
	}
	_, err = tx.Exec(ctx, `UPDATE runtime_engine.role_fence SET owner_id=$3,epoch=$4,version=$5,activated_at=now() WHERE deployment_id=$1 AND role=$2`, deployment, role, token.OwnerID, token.Epoch, version+1)
	return err
}
func activateProducerTx(ctx context.Context, tx pgx.Tx, deployment, key string, token ProducerToken) error {
	var owner string
	var epoch, version int64
	err := tx.QueryRow(ctx, `SELECT owner_id,epoch,version FROM runtime_engine.producer_fence WHERE deployment_id=$1 AND producer_key=$2 FOR UPDATE`, deployment, key).Scan(&owner, &epoch, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		_, err = tx.Exec(ctx, `INSERT INTO runtime_engine.producer_fence(deployment_id,producer_key,owner_id,epoch,version) VALUES($1,$2,$3,$4,1)`, deployment, key, token.OwnerID, token.Epoch)
		return err
	}
	if err != nil {
		return err
	}
	if owner == token.OwnerID && epoch == token.Epoch {
		return nil
	}
	if token.Epoch <= epoch {
		return ErrFenceRejected
	}
	_, err = tx.Exec(ctx, `UPDATE runtime_engine.producer_fence SET owner_id=$3,epoch=$4,version=$5,activated_at=now() WHERE deployment_id=$1 AND producer_key=$2`, deployment, key, token.OwnerID, token.Epoch, version+1)
	return err
}
