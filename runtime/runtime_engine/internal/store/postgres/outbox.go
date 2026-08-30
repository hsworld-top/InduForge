package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/indu-forge/runtime-engine/internal/transportlimits"
	"github.com/jackc/pgx/v5"
)

type OutboxMessage struct {
	DeploymentID string
	DedupeKey    string // 同时是发布端稳定的 NATS Msg-Id。
	Subject      string
	Headers      json.RawMessage
	Payload      []byte
}

type OutboxRecord struct {
	ID            int64
	DeploymentID  string
	DedupeKey     string
	Subject       string
	Headers       json.RawMessage
	Payload       []byte
	PayloadSHA256 string
	LeaseToken    string
	LeaseOwner    string
	LeaseUntil    time.Time
	AttemptCount  int
}

// BusinessTx 是已通过 consumer/producer fence 的受控业务事务；不暴露底层 pgx.Tx，避免伪造事务写入 outbox。
type BusinessTx struct {
	store        *Store
	tx           pgx.Tx
	deploymentID string
}

func (b *BusinessTx) ensureOpen() error {
	if b == nil || b.store == nil {
		return ErrStoreClosed
	}
	return b.store.ensureOpen()
}

// PointHistoryWrite/PointCurrentWrite 是算法层唯一可用的点位状态写模型；底层事务不外泄。
type PointHistoryWrite struct {
	DeploymentID, PointID, EventID, OwnerID, Quality string
	Epoch, Sequence                                  int64
	SourceTimestamp, ServerTimestamp, ReceivedAt     time.Time
	Value                                            json.RawMessage
}
type PointCurrentWrite struct {
	DeploymentID, PointID, OwnerID, EventID, Quality string
	Epoch, Sequence                                  int64
	SourceTimestamp, ServerTimestamp                 time.Time
	Value                                            json.RawMessage
}
type PointCurrentDisposition string

const (
	PointCurrentApplied PointCurrentDisposition = "applied"
	PointCurrentIgnored PointCurrentDisposition = "ignored"
)

type PointCurrentResult struct {
	Disposition PointCurrentDisposition
	Version     int64
}

// Enqueue 是业务算法可用的受控 outbox 写入口，消息只能写入当前 fenced deployment。
func (b *BusinessTx) Enqueue(ctx context.Context, message OutboxMessage) (int64, bool, error) {
	if b == nil || b.store == nil || message.DeploymentID != b.deploymentID {
		return 0, false, fmt.Errorf("%w: outbox deployment", ErrInvalidInput)
	}
	if err := b.ensureOpen(); err != nil {
		return 0, false, err
	}
	return b.store.enqueueTx(ctx, b.tx, message)
}

func (b *BusinessTx) InsertPointHistory(ctx context.Context, write PointHistoryWrite) error {
	if b == nil || b.tx == nil || write.DeploymentID != b.deploymentID || !validPointWrite(write.DeploymentID, write.PointID, write.EventID, write.OwnerID, write.Quality, write.Epoch, write.Sequence, write.SourceTimestamp, write.ServerTimestamp, write.Value) || write.ReceivedAt.IsZero() {
		return fmt.Errorf("%w: point history", ErrInvalidInput)
	}
	if err := b.ensureOpen(); err != nil {
		return err
	}
	_, err := b.tx.Exec(ctx, `INSERT INTO runtime_engine.point_history (deployment_id,point_id,event_id,owner_id,epoch,sequence,source_timestamp,server_timestamp,value,quality,received_at) VALUES ($1,$2::uuid,$3,$4,$5,$6,$7,$8,$9::jsonb,$10,$11) ON CONFLICT DO NOTHING`, write.DeploymentID, write.PointID, write.EventID, write.OwnerID, write.Epoch, write.Sequence, write.SourceTimestamp.UTC(), write.ServerTimestamp.UTC(), nullableJSON(write.Value), write.Quality, write.ReceivedAt.UTC())
	return err
}

// ApplyPointCurrent 原子锁定当前行，按 (epoch, sourceTimestamp, sequence, eventId) 决定覆盖；version 仅由数据库递增。
func (b *BusinessTx) ApplyPointCurrent(ctx context.Context, write PointCurrentWrite) (PointCurrentResult, error) {
	if b == nil || b.tx == nil || write.DeploymentID != b.deploymentID || !validPointWrite(write.DeploymentID, write.PointID, write.EventID, write.OwnerID, write.Quality, write.Epoch, write.Sequence, write.SourceTimestamp, write.ServerTimestamp, write.Value) {
		return PointCurrentResult{}, fmt.Errorf("%w: point current", ErrInvalidInput)
	}
	if err := b.ensureOpen(); err != nil {
		return PointCurrentResult{}, err
	}
	for attempts := 0; attempts < 2; attempts++ {
		var current PointCurrentWrite
		var version int64
		err := b.tx.QueryRow(ctx, `SELECT owner_id,epoch,source_timestamp,server_timestamp,sequence,event_id,value,quality,version FROM runtime_engine.point_current WHERE deployment_id=$1 AND point_id=$2::uuid FOR UPDATE`, write.DeploymentID, write.PointID).Scan(&current.OwnerID, &current.Epoch, &current.SourceTimestamp, &current.ServerTimestamp, &current.Sequence, &current.EventID, &current.Value, &current.Quality, &version)
		if err == pgx.ErrNoRows {
			var inserted int64
			err = b.tx.QueryRow(ctx, `INSERT INTO runtime_engine.point_current (deployment_id,point_id,owner_id,epoch,source_timestamp,server_timestamp,sequence,event_id,value,quality,version) VALUES ($1,$2::uuid,$3,$4,$5,$6,$7,$8,$9::jsonb,$10,1) ON CONFLICT DO NOTHING RETURNING version`, write.DeploymentID, write.PointID, write.OwnerID, write.Epoch, write.SourceTimestamp.UTC(), write.ServerTimestamp.UTC(), write.Sequence, write.EventID, nullableJSON(write.Value), write.Quality).Scan(&inserted)
			if err == nil {
				return PointCurrentResult{PointCurrentApplied, inserted}, nil
			}
			if err == pgx.ErrNoRows {
				continue
			}
			return PointCurrentResult{}, err
		}
		if err != nil {
			return PointCurrentResult{}, err
		}
		if comparePointOrder(write, current) <= 0 {
			return PointCurrentResult{PointCurrentIgnored, version}, nil
		}
		var next int64
		err = b.tx.QueryRow(ctx, `UPDATE runtime_engine.point_current SET owner_id=$3,epoch=$4,source_timestamp=$5,server_timestamp=$6,sequence=$7,event_id=$8,value=$9::jsonb,quality=$10,version=version+1,updated_at=now() WHERE deployment_id=$1 AND point_id=$2::uuid AND version=$11 RETURNING version`, write.DeploymentID, write.PointID, write.OwnerID, write.Epoch, write.SourceTimestamp.UTC(), write.ServerTimestamp.UTC(), write.Sequence, write.EventID, nullableJSON(write.Value), write.Quality, version).Scan(&next)
		if err == nil {
			return PointCurrentResult{PointCurrentApplied, next}, nil
		}
		if err == pgx.ErrNoRows {
			continue
		}
		return PointCurrentResult{}, err
	}
	return PointCurrentResult{}, ErrFenceStale
}

func comparePointOrder(next, current PointCurrentWrite) int {
	if next.Epoch != current.Epoch {
		if next.Epoch > current.Epoch {
			return 1
		}
		return -1
	}
	if !next.SourceTimestamp.Equal(current.SourceTimestamp) {
		if next.SourceTimestamp.After(current.SourceTimestamp) {
			return 1
		}
		return -1
	}
	if next.Sequence != current.Sequence {
		if next.Sequence > current.Sequence {
			return 1
		}
		return -1
	}
	return strings.Compare(next.EventID, current.EventID)
}

var canonicalPointUUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
var canonicalEventID = regexp.MustCompile(`^[0-9a-f]{64}$`)

func validPointWrite(deployment, pointID, eventID, owner, quality string, epoch, sequence int64, source, server time.Time, value json.RawMessage) bool {
	return validStableID(deployment) && validStableID(owner) && canonicalPointUUID.MatchString(pointID) && canonicalEventID.MatchString(eventID) && (quality == "good" || quality == "bad" || quality == "unknown") && epoch >= 1 && sequence >= 0 && !source.IsZero() && !server.IsZero() && len(value) <= 1<<20 && validFiniteJSON(value)
}
func validFiniteJSON(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return true
	}
	d := json.NewDecoder(strings.NewReader(string(raw)))
	d.UseNumber()
	var value any
	if d.Decode(&value) != nil {
		return false
	}
	var trailing any
	if err := d.Decode(&trailing); err != io.EOF {
		return false
	}
	return finiteJSON(value)
}
func finiteJSON(value any) bool {
	switch x := value.(type) {
	case json.Number:
		return exactJSONNumber(x.String())
	case []any:
		for _, v := range x {
			if !finiteJSON(v) {
				return false
			}
		}
	case map[string]any:
		for _, v := range x {
			if !finiteJSON(v) {
				return false
			}
		}
	}
	return true
}

// exactJSONNumber accepts every finite decimal JSON number representable by
// the V1 bounded grammar without routing through float64. Generic jsonb state
// may legitimately contain 1e1000 and precise nested decimals; float bounds
// belong only to declared datapoint dataType validation.
func exactJSONNumber(value string) bool {
	if value == "" || strings.ContainsAny(value, "NaInF") {
		return false
	}
	if value[0] == '-' {
		value = value[1:]
	} else if value[0] == '+' {
		return false
	}
	parts := strings.Split(strings.ToLower(value), "e")
	if len(parts) > 2 {
		return false
	}
	exponent := 0
	if len(parts) == 2 {
		var err error
		exponent, err = strconv.Atoi(parts[1])
		if err != nil || exponent < -10000 || exponent > 10000 {
			return false
		}
	}
	dot := strings.Split(parts[0], ".")
	if len(dot) > 2 || dot[0] == "" {
		return false
	}
	whole, fraction := dot[0], ""
	if len(dot) == 2 {
		fraction = dot[1]
		if fraction == "" {
			return false
		}
	}
	if (whole != "0" && (whole[0] == '0' || !decimalDigits(whole))) || !decimalDigits(fraction) {
		return false
	}
	digits := strings.TrimLeft(whole+fraction, "0")
	if digits == "" {
		return true
	}
	numerator := new(big.Int)
	if _, ok := numerator.SetString(digits, 10); !ok {
		return false
	}
	scale := len(fraction) - exponent
	denominator := big.NewInt(1)
	if scale >= 0 {
		denominator.Exp(big.NewInt(10), big.NewInt(int64(scale)), nil)
	} else {
		numerator.Mul(numerator, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-scale)), nil))
	}
	// Construct the exact rational as a final guard that this grammar never
	// routes a decimal through binary floating point.
	_ = new(big.Rat).SetFrac(numerator, denominator)
	return true
}

func decimalDigits(value string) bool {
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func nullableJSON(value json.RawMessage) any {
	if len(value) == 0 {
		return nil
	}
	return value
}

type RetryCode string

const (
	RetryPublishError   RetryCode = "publish-error"
	RetryPublishTimeout RetryCode = "publish-timeout"
	RetryShutdown       RetryCode = "shutdown"
)

// enqueueTx 仅供 Store 内部已验证 fence 的事务调用。重复 key 仅接受完全相同的不可变发布内容。
func (s *Store) enqueueTx(ctx context.Context, tx pgx.Tx, message OutboxMessage) (int64, bool, error) {
	if message.DeploymentID == "" || message.DedupeKey == "" || message.Subject == "" || len(message.DedupeKey) > 512 || len(message.Subject) > 4096 {
		return 0, false, fmt.Errorf("%w: outbox message", ErrInvalidInput)
	}
	if err := transportlimits.ValidateOutboundPayload(message.Subject, message.Payload); err != nil {
		return 0, false, err
	}
	if tx == nil {
		return 0, false, fmt.Errorf("%w: outbox transaction", ErrInvalidInput)
	}
	headers, err := objectJSON(message.Headers)
	if err != nil {
		return 0, false, err
	}
	digest := payloadSHA256(message.Payload)
	var id int64
	err = tx.QueryRow(ctx, `INSERT INTO runtime_engine.transactional_outbox (deployment_id,dedupe_key,subject,headers,payload,payload_sha256)
VALUES ($1,$2,$3,$4::jsonb,$5,$6) ON CONFLICT DO NOTHING RETURNING id`, message.DeploymentID, message.DedupeKey, message.Subject, headers, message.Payload, digest).Scan(&id)
	if err == nil {
		return id, true, nil
	}
	if err != pgx.ErrNoRows {
		return 0, false, err
	}
	var oldSubject, oldDigest string
	var oldHeaders, oldPayload []byte
	err = tx.QueryRow(ctx, `SELECT id,subject,headers,payload,payload_sha256 FROM runtime_engine.transactional_outbox WHERE deployment_id=$1 AND dedupe_key=$2`, message.DeploymentID, message.DedupeKey).
		Scan(&id, &oldSubject, &oldHeaders, &oldPayload, &oldDigest)
	if err != nil {
		return 0, false, err
	}
	if oldSubject != message.Subject || oldDigest != digest || !bytes.Equal(oldPayload, message.Payload) || !jsonEqual(oldHeaders, headers) {
		return 0, false, ErrOutboxConflict
	}
	return id, false, nil
}

// ClaimOutbox 通过 SKIP LOCKED 获取租约；已过期租约可重领且返回保持不变的 payload 和 dedupe key。
func (s *Store) ClaimOutbox(ctx context.Context, deploymentID, leaseOwner string, limit int, leaseFor time.Duration) ([]OutboxRecord, error) {
	if !validStableID(deploymentID) || !validStableID(leaseOwner) || limit < 1 || limit > 500 || leaseFor < time.Millisecond || leaseFor > 15*time.Minute {
		return nil, fmt.Errorf("%w: outbox claim", ErrInvalidInput)
	}
	if err := s.ensureOpen(); err != nil {
		return nil, err
	}
	token, err := randomLeaseToken()
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `WITH candidates AS (
 SELECT id FROM runtime_engine.transactional_outbox
 WHERE deployment_id=$1 AND state <> 'published' AND next_attempt_at <= now()
   AND (state='pending' OR (state='leased' AND lease_until <= now()))
 ORDER BY id FOR UPDATE SKIP LOCKED LIMIT $2
)
UPDATE runtime_engine.transactional_outbox o
SET state='leased', lease_token=$3, lease_owner=$4, lease_until=now()+$5::interval,
    attempt_count=attempt_count+1, updated_at=now()
FROM candidates c WHERE o.id=c.id
RETURNING o.id,o.deployment_id,o.dedupe_key,o.subject,o.headers,o.payload,o.payload_sha256,o.lease_token,o.lease_owner,o.lease_until,o.attempt_count`, deploymentID, limit, token, leaseOwner, durationInterval(leaseFor))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	claimed := make([]OutboxRecord, 0, limit)
	for rows.Next() {
		var r OutboxRecord
		if err := rows.Scan(&r.ID, &r.DeploymentID, &r.DedupeKey, &r.Subject, &r.Headers, &r.Payload, &r.PayloadSHA256, &r.LeaseToken, &r.LeaseOwner, &r.LeaseUntil, &r.AttemptCount); err != nil {
			return nil, err
		}
		claimed = append(claimed, r)
	}
	return claimed, rows.Err()
}

func (s *Store) MarkPublished(ctx context.Context, id int64, leaseToken string) error {
	if id <= 0 || leaseToken == "" {
		return fmt.Errorf("%w: mark published", ErrInvalidInput)
	}
	if err := s.ensureOpen(); err != nil {
		return err
	}
	command, err := s.pool.Exec(ctx, `UPDATE runtime_engine.transactional_outbox
SET state='published', published_at=now(), lease_token=NULL, lease_owner=NULL, lease_until=NULL, updated_at=now()
WHERE id=$1 AND state='leased' AND lease_token=$2`, id, leaseToken)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return ErrLeaseLost
	}
	return nil
}

// RetryOutbox 只接受稳定白名单码，调用方不得把 error.Error() 或外部响应持久化。
func (s *Store) RetryOutbox(ctx context.Context, id int64, leaseToken string, nextAttemptAt time.Time, code RetryCode) error {
	if id <= 0 || leaseToken == "" || nextAttemptAt.IsZero() || !validRetryCode(code) {
		return fmt.Errorf("%w: retry outbox", ErrInvalidInput)
	}
	if err := s.ensureOpen(); err != nil {
		return err
	}
	command, err := s.pool.Exec(ctx, `UPDATE runtime_engine.transactional_outbox
SET state='pending', lease_token=NULL, lease_owner=NULL, lease_until=NULL, next_attempt_at=$3, last_error_code=$4, updated_at=now()
WHERE id=$1 AND state='leased' AND lease_token=$2`, id, leaseToken, nextAttemptAt.UTC(), code)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return ErrLeaseLost
	}
	return nil
}

// ReleaseOutbox 是无错误重试的便捷形式，仍走 lease token CAS。
func (s *Store) ReleaseOutbox(ctx context.Context, id int64, leaseToken string, nextAttemptAt time.Time, code RetryCode) error {
	if id <= 0 || leaseToken == "" || nextAttemptAt.IsZero() || !validRetryCode(code) {
		return fmt.Errorf("%w: retry outbox", ErrInvalidInput)
	}
	if err := s.ensureOpen(); err != nil {
		return err
	}
	return s.RetryOutbox(ctx, id, leaseToken, nextAttemptAt, code)
}

func objectJSON(raw json.RawMessage) ([]byte, error) {
	if len(raw) == 0 {
		return []byte("{}"), nil
	}
	if len(raw) > 16<<10 {
		return nil, fmt.Errorf("%w: headers 过大", ErrInvalidInput)
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return nil, fmt.Errorf("%w: headers 必须为 JSON object", ErrInvalidInput)
	}
	return raw, nil
}

func validRetryCode(code RetryCode) bool {
	return code == RetryPublishError || code == RetryPublishTimeout || code == RetryShutdown
}

func jsonEqual(left, right []byte) bool {
	var a, b any
	return json.Unmarshal(left, &a) == nil && json.Unmarshal(right, &b) == nil && reflect.DeepEqual(a, b)
}

func durationInterval(d time.Duration) string {
	// PostgreSQL interval 参数避免拼接 SQL；微秒精度足以表示租约。
	return fmt.Sprintf("%d microseconds", d.Microseconds())
}
