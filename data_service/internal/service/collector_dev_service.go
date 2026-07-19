package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/indu-forge/data_service/internal/auth"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
	collectorsecurity "github.com/indu-forge/data_service/internal/security"
)

const collectorOnlineWindow = 45 * time.Second

type CollectorDevService struct {
	repository          *repository.CollectorDevRepository
	collectorRepository *repository.CollectorRepository
	secretCipher        *collectorsecurity.CollectorSecretCipher
	taskSignals         sync.Map
}

func (s *CollectorDevService) ConfigureTaskEnvelope(collectorRepository *repository.CollectorRepository, secretCipher *collectorsecurity.CollectorSecretCipher) {
	s.collectorRepository = collectorRepository
	s.secretCipher = secretCipher
}

func NewCollectorDevService(repository *repository.CollectorDevRepository) *CollectorDevService {
	return &CollectorDevService{repository: repository}
}

type CollectorProtocolCapability struct {
	DriverID       string   `json:"driverId"`
	DriverVersion  string   `json:"driverVersion"`
	SchemaVersions []int    `json:"schemaVersions"`
	Operations     []string `json:"operations"`
}
type CollectorRegistrationCode struct {
	Code      string `json:"code"`
	ExpiresAt string `json:"expiresAt"`
}
type CollectorAgent struct {
	ID           string                        `json:"id"`
	Name         string                        `json:"name"`
	OS           string                        `json:"os"`
	Arch         string                        `json:"arch"`
	Version      string                        `json:"version"`
	IPAddress    string                        `json:"ipAddress"`
	Status       string                        `json:"status"`
	Capabilities []CollectorProtocolCapability `json:"capabilities"`
	LastSeenAt   *string                       `json:"lastSeenAt"`
	CreatedAt    string                        `json:"createdAt"`
}
type CollectorRegisterInput struct {
	RegistrationCode string                        `json:"registrationCode"`
	MachineID        string                        `json:"machineId"`
	Name             string                        `json:"name"`
	OS               string                        `json:"os"`
	Arch             string                        `json:"arch"`
	Version          string                        `json:"version"`
	Capabilities     []CollectorProtocolCapability `json:"capabilities"`
}
type CollectorRegistration struct {
	AgentID    string `json:"agentId"`
	AgentToken string `json:"agentToken"`
	TenantID   string `json:"tenantId"`
}
type CollectorTaskInput struct {
	AgentID        string         `json:"agentId"`
	ConnectionID   string         `json:"connectionId"`
	Operation      string         `json:"operation"`
	Input          map[string]any `json:"input"`
	TimeoutSeconds int            `json:"timeoutSeconds"`
}
type CollectorTaskCompletion struct {
	Status string              `json:"status"`
	Result any                 `json:"result"`
	Error  *CollectorTaskError `json:"error"`
}
type CollectorTaskError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}
type CollectorTask struct {
	ID           string          `json:"taskId"`
	ProjectID    string          `json:"projectId"`
	ConnectionID string          `json:"connectionId"`
	AgentID      string          `json:"agentId"`
	Operation    string          `json:"operation"`
	Status       string          `json:"status"`
	Request      json.RawMessage `json:"request"`
	Result       json.RawMessage `json:"result"`
	ErrorCode    *string         `json:"errorCode"`
	ErrorMessage *string         `json:"errorMessage"`
	DeadlineAt   string          `json:"deadlineAt"`
	ClaimedAt    *string         `json:"claimedAt"`
	FinishedAt   *string         `json:"finishedAt"`
	CreatedAt    string          `json:"createdAt"`
}

func (s *CollectorDevService) CreateRegistrationCode(ctx context.Context, claims *auth.Claims) (*CollectorRegistrationCode, error) {
	if !isCollectorAdmin(claims) {
		return nil, apperrors.NewAppError(apperrors.ErrorCodePermissionInsufficient, http.StatusForbidden, "仅系统管理员可以生成注册码")
	}
	code, err := randomSecret(24)
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().Add(10 * time.Minute)
	if _, err = s.repository.CreateRegistrationCode(ctx, uuid.NewString(), claims.TenantID, hashSecret(code), claims.UserID, expiresAt); err != nil {
		return nil, err
	}
	return &CollectorRegistrationCode{Code: code, ExpiresAt: formatCollectorTime(expiresAt)}, nil
}

func (s *CollectorDevService) Register(ctx context.Context, input CollectorRegisterInput, clientIP string) (*CollectorRegistration, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.RegistrationCode = strings.TrimSpace(input.RegistrationCode)
	input.MachineID = strings.ToLower(strings.TrimSpace(input.MachineID))
	if input.Name == "" || input.RegistrationCode == "" || input.MachineID == "" {
		return nil, badCollectorRequest("注册码、机器标识和 Agent 名称不能为空")
	}
	if len(input.MachineID) != sha256.Size*2 {
		return nil, badCollectorRequest("机器标识格式无效")
	}
	if _, err := hex.DecodeString(input.MachineID); err != nil {
		return nil, badCollectorRequest("机器标识格式无效")
	}
	if input.OS == "" {
		input.OS = runtime.GOOS
	}
	if input.Arch == "" {
		input.Arch = runtime.GOARCH
	}
	if err := validateCapabilities(input.Capabilities); err != nil {
		return nil, err
	}
	token, err := randomSecret(32)
	if err != nil {
		return nil, err
	}
	record, err := s.repository.RegisterAgent(ctx, repository.CreateCollectorAgentParams{ID: uuid.NewString(), CodeHash: hashSecret(input.RegistrationCode), CredentialHash: hashSecret(token), MachineID: input.MachineID, Name: input.Name, OS: input.OS, Arch: input.Arch, Version: input.Version, LastIP: clientIP, Capabilities: input.Capabilities})
	if err != nil {
		return nil, err
	}
	return &CollectorRegistration{AgentID: record.ID, AgentToken: token, TenantID: record.TenantID}, nil
}

func (s *CollectorDevService) AuthenticateAgent(ctx context.Context, token string) (*auth.CollectorAgentIdentity, error) {
	record, err := s.repository.AuthenticateAgent(ctx, HashCollectorAgentToken(token))
	if err != nil {
		return nil, err
	}
	return &auth.CollectorAgentIdentity{AgentID: record.ID, TenantID: record.TenantID}, nil
}

// ListAgentsPage 查询当前租户下的分页采集调试代理。
func (s *CollectorDevService) ListAgentsPage(ctx context.Context, claims *auth.Claims, page, pageSize int) ([]CollectorAgent, int, error) {
	if claims == nil || strings.TrimSpace(claims.TenantID) == "" {
		return nil, 0, apperrors.NewAppError(apperrors.ErrorCodeAuthTokenInvalid, http.StatusUnauthorized, "JWT 缺少租户信息")
	}
	records, total, err := s.repository.ListAgentsPage(ctx, claims.TenantID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	result := make([]CollectorAgent, 0, len(records))
	now := time.Now()
	for _, record := range records {
		result = append(result, toCollectorAgent(record, now))
	}
	return result, total, nil
}

func (s *CollectorDevService) DeleteAgent(ctx context.Context, claims *auth.Claims, agentID string) error {
	if !isCollectorAdmin(claims) {
		return apperrors.NewAppError(apperrors.ErrorCodePermissionInsufficient, http.StatusForbidden, "仅系统管理员可以删除 Agent")
	}
	return s.repository.DeleteAgent(ctx, claims.TenantID, agentID)
}

// Disconnect 标记代理主动离线，凭据仍可用于后续重新连接。
func (s *CollectorDevService) Disconnect(ctx context.Context, identity *auth.CollectorAgentIdentity) error {
	return s.repository.DisconnectAgent(ctx, identity.AgentID)
}

// Revoke 撤销代理当前凭据，节点记录保留等待新注册码激活。
func (s *CollectorDevService) Revoke(ctx context.Context, identity *auth.CollectorAgentIdentity) error {
	return s.repository.RevokeAgent(ctx, identity.AgentID)
}

func (s *CollectorDevService) Heartbeat(ctx context.Context, identity *auth.CollectorAgentIdentity, capabilities []CollectorProtocolCapability, clientIP string) (*CollectorAgent, error) {
	if err := validateCapabilities(capabilities); err != nil {
		return nil, err
	}
	record, err := s.repository.Heartbeat(ctx, identity.AgentID, clientIP, capabilities)
	if err != nil {
		return nil, err
	}
	result := toCollectorAgent(*record, time.Now())
	return &result, nil
}

const collectorTaskClaimFallbackInterval = time.Second

func waitForCollectorTask(
	ctx context.Context,
	wait time.Duration,
	signal <-chan struct{},
	claim func() (*repository.CollectorDevTaskRecord, error),
) (*repository.CollectorDevTaskRecord, error) {
	deadline := time.Now().Add(wait)
	for {
		record, err := claim()
		if err != nil || record != nil || wait <= 0 {
			return record, err
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil, nil
		}
		delay := collectorTaskClaimFallbackInterval
		if remaining < delay {
			delay = remaining
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-signal:
			if !timer.Stop() {
				<-timer.C
			}
		case <-timer.C:
		}
	}
}

func (s *CollectorDevService) taskSignal(agentID string) chan struct{} {
	value, _ := s.taskSignals.LoadOrStore(agentID, make(chan struct{}, 1))
	return value.(chan struct{})
}

func (s *CollectorDevService) notifyTaskCreated(agentID string) {
	select {
	case s.taskSignal(agentID) <- struct{}{}:
	default:
	}
}

func (s *CollectorDevService) ClaimTask(ctx context.Context, identity *auth.CollectorAgentIdentity, wait time.Duration) (*CollectorTask, error) {
	record, err := waitForCollectorTask(ctx, wait, s.taskSignal(identity.AgentID), func() (*repository.CollectorDevTaskRecord, error) {
		return s.repository.ClaimTask(ctx, identity.AgentID)
	})
	if err != nil || record == nil {
		return nil, err
	}
	result := toCollectorTask(*record)
	envelope, err := s.buildTaskEnvelope(ctx, record)
	if err != nil {
		return nil, err
	}
	result.Request = envelope
	return &result, nil
}

func (s *CollectorDevService) CreateTask(ctx context.Context, claims *auth.Claims, projectID string, input CollectorTaskInput) (*CollectorTask, error) {
	if claims == nil || !claims.HasProjectAccess(projectID) || !claims.HasCapability("project:write") {
		return nil, apperrors.NewAppError(apperrors.ErrorCodePermissionInsufficient, http.StatusForbidden, "项目写权限不足")
	}
	if err := validateCollectorTaskInput(input); err != nil {
		return nil, err
	}
	if input.TimeoutSeconds == 0 {
		input.TimeoutSeconds = 30
	}
	if s.collectorRepository == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "工业采集任务服务未初始化")
	}
	connection, err := s.collectorRepository.GetConnection(ctx, projectID, input.ConnectionID)
	if err != nil {
		return nil, err
	}
	agent, err := s.repository.GetAgent(ctx, claims.TenantID, input.AgentID)
	if err != nil {
		return nil, err
	}
	capabilities := []CollectorProtocolCapability{}
	if err := json.Unmarshal(agent.Capabilities, &capabilities); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解析 Agent 能力失败", err)
	}
	if !collectorAgentSupports(capabilities, connection.DriverID, connection.DriverVersion, connection.SchemaVersion, input.Operation) {
		return nil, badCollectorRequest("所选 Agent 不支持该驱动、版本或操作")
	}
	record, err := s.repository.CreateTask(ctx, repository.CreateCollectorTaskParams{ID: uuid.NewString(), TenantID: claims.TenantID, ProjectID: projectID, ConnectionID: input.ConnectionID, AgentID: input.AgentID, Operation: input.Operation, RequestPayload: input.Input, DeadlineAt: time.Now().Add(time.Duration(input.TimeoutSeconds) * time.Second), CreatedBy: claims.UserID})
	if err != nil {
		return nil, err
	}
	s.notifyTaskCreated(input.AgentID)
	result := toCollectorTask(*record)
	return &result, nil
}
func (s *CollectorDevService) GetTask(ctx context.Context, claims *auth.Claims, projectID, taskID string) (*CollectorTask, error) {
	if claims == nil || !claims.HasProjectAccess(projectID) {
		return nil, apperrors.NewAppError(apperrors.ErrorCodePermissionProjectMismatch, http.StatusForbidden, "项目范围不足")
	}
	record, err := s.repository.GetTask(ctx, claims.TenantID, projectID, taskID)
	if err != nil {
		return nil, err
	}
	result := toCollectorTask(*record)
	return &result, nil
}
func (s *CollectorDevService) CancelTask(ctx context.Context, claims *auth.Claims, projectID, taskID string) (*CollectorTask, error) {
	if claims == nil || !claims.HasProjectAccess(projectID) || !claims.HasCapability("project:write") {
		return nil, apperrors.NewAppError(apperrors.ErrorCodePermissionInsufficient, http.StatusForbidden, "项目写权限不足")
	}
	record, err := s.repository.CancelTask(ctx, claims.TenantID, projectID, taskID)
	if err != nil {
		return nil, err
	}
	result := toCollectorTask(*record)
	return &result, nil
}
func (s *CollectorDevService) CompleteTask(ctx context.Context, identity *auth.CollectorAgentIdentity, taskID string, input CollectorTaskCompletion) (*CollectorTask, error) {
	if input.Status != "succeeded" && input.Status != "failed" {
		return nil, badCollectorRequest("任务完成状态只能是 succeeded 或 failed")
	}
	params := repository.CompleteCollectorTaskParams{AgentID: identity.AgentID, TaskID: taskID, Status: input.Status, Result: input.Result}
	if input.Error != nil {
		params.ErrorCode = input.Error.Code
		params.ErrorMessage = input.Error.Message
	}
	record, err := s.repository.CompleteTask(ctx, params)
	if err != nil {
		return nil, err
	}
	result := toCollectorTask(*record)
	return &result, nil
}

func validateCollectorTaskInput(input CollectorTaskInput) error {
	if strings.TrimSpace(input.AgentID) == "" {
		return badCollectorRequest("agentId 不能为空")
	}
	if _, err := uuid.Parse(strings.TrimSpace(input.ConnectionID)); err != nil {
		return badCollectorRequest("connectionId 格式无效")
	}
	allowed := map[string]bool{"connection.test": true, "connection.open": true, "connection.close": true, "device.browse": true, "point.read": true, "point.write": true, "point.subscribe.preview": true}
	if !allowed[input.Operation] {
		return badCollectorRequest("不支持的调试任务操作")
	}
	if input.TimeoutSeconds < 0 || input.TimeoutSeconds > 120 {
		return badCollectorRequest("timeoutSeconds 必须在 1 到 120 秒之间")
	}
	if input.Operation == "device.browse" {
		if depth, ok := input.Input["maxDepth"].(float64); ok && depth != 1 {
			return badCollectorRequest("Browse 仅支持 maxDepth = 1")
		}
		if parentNodeIDs, exists := input.Input["parentNodeIds"]; exists {
			values, ok := parentNodeIDs.([]any)
			if !ok || len(values) == 0 || len(values) > 100 {
				return badCollectorRequest("parentNodeIds 数量必须在 1 到 100 之间")
			}
			for _, value := range values {
				if nodeID, ok := value.(string); !ok || strings.TrimSpace(nodeID) == "" {
					return badCollectorRequest("parentNodeIds 只能包含非空字符串")
				}
			}
		}
	}
	if input.Operation == "point.read" || input.Operation == "point.write" || input.Operation == "point.subscribe.preview" {
		if ids, ok := input.Input["pointIds"].([]any); !ok || len(ids) == 0 || len(ids) > 500 {
			return badCollectorRequest("pointIds 数量必须在 1 到 500 之间")
		}
	}
	return nil
}
func validateCapabilities(capabilities []CollectorProtocolCapability) error {
	for _, capability := range capabilities {
		if strings.TrimSpace(capability.DriverID) == "" || strings.TrimSpace(capability.DriverVersion) == "" || len(capability.SchemaVersions) == 0 {
			return badCollectorRequest("Agent capability 缺少 driverId、driverVersion 或 schemaVersions")
		}
		for _, operation := range capability.Operations {
			if !map[string]bool{"connection.test": true, "connection.open": true, "connection.close": true, "device.browse": true, "point.read": true, "point.write": true, "point.subscribe.preview": true}[operation] {
				return badCollectorRequest("Agent capability 包含不支持的操作")
			}
		}
	}
	return nil
}
func isCollectorAdmin(claims *auth.Claims) bool {
	return claims != nil && (claims.Role == "SYSTEM_ADMIN" || claims.Role == "SUPER_ADMIN")
}
func randomSecret(size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "生成安全随机凭据失败", err)
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}
func hashSecret(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
func HashCollectorAgentToken(value string) string { return hashSecret(strings.TrimSpace(value)) }
func badCollectorRequest(message string) error {
	return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, message)
}
func formatCollectorTime(value time.Time) string { return value.Local().Format("2006-01-02 15:04:05") }
func optionalCollectorTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := formatCollectorTime(*value)
	return &formatted
}
func toCollectorAgent(record repository.CollectorDevAgentRecord, now time.Time) CollectorAgent {
	capabilities := []CollectorProtocolCapability{}
	_ = json.Unmarshal(record.Capabilities, &capabilities)
	// 历史代理能力可能缺少数组字段，统一输出空数组，避免 JSON null 破坏前端契约。
	for index := range capabilities {
		if capabilities[index].SchemaVersions == nil {
			capabilities[index].SchemaVersions = []int{}
		}
		if capabilities[index].Operations == nil {
			capabilities[index].Operations = []string{}
		}
	}
	return CollectorAgent{ID: record.ID, Name: record.Name, OS: record.OS, Arch: record.Arch, Version: record.Version, IPAddress: record.LastIP, Status: collectorAgentStatus(record, now), Capabilities: capabilities, LastSeenAt: optionalCollectorTime(record.LastSeenAt), CreatedAt: formatCollectorTime(record.CreatedAt)}
}
func collectorAgentStatus(record repository.CollectorDevAgentRecord, now time.Time) string {
	if record.RevokedAt != nil {
		return "invalid"
	}
	if record.LastSeenAt == nil {
		return "offline"
	}
	if record.DisconnectedAt != nil && !record.DisconnectedAt.Before(*record.LastSeenAt) {
		return "offline"
	}
	if now.Sub(*record.LastSeenAt) > collectorOnlineWindow {
		return "offline"
	}
	return "online"
}

func toCollectorTask(record repository.CollectorDevTaskRecord) CollectorTask {
	return CollectorTask{ID: record.ID, ProjectID: record.ProjectID, ConnectionID: record.ConnectionID, AgentID: record.AgentID, Operation: record.Operation, Status: record.Status, Request: record.RequestPayload, Result: record.ResultPayload, ErrorCode: record.ErrorCode, ErrorMessage: record.ErrorMessage, DeadlineAt: formatCollectorTime(record.DeadlineAt), ClaimedAt: optionalCollectorTime(record.ClaimedAt), FinishedAt: optionalCollectorTime(record.FinishedAt), CreatedAt: formatCollectorTime(record.CreatedAt)}
}

func collectorAgentSupports(capabilities []CollectorProtocolCapability, driverID, driverVersion string, schemaVersion int, operation string) bool {
	for _, capability := range capabilities {
		if capability.DriverID != driverID || capability.DriverVersion != driverVersion || !containsInt(capability.SchemaVersions, schemaVersion) || !containsFold(capability.Operations, operation) {
			continue
		}
		return true
	}
	return false
}

func containsInt(values []int, expected int) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func (s *CollectorDevService) buildTaskEnvelope(ctx context.Context, record *repository.CollectorDevTaskRecord) (json.RawMessage, error) {
	if s.collectorRepository == nil || s.secretCipher == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "工业采集任务信封服务未初始化")
	}
	connection, err := s.collectorRepository.GetConnection(ctx, record.ProjectID, record.ConnectionID)
	if err != nil {
		return nil, err
	}
	secretRecords, err := s.collectorRepository.GetConnectionSecrets(ctx, record.ConnectionID)
	if err != nil {
		return nil, err
	}
	secrets := map[string]string{}
	for _, secret := range secretRecords {
		if secret.KeyVersion != s.secretCipher.KeyVersion() {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "连接密钥版本当前不可解密")
		}
		plaintext, err := s.secretCipher.Decrypt(secret.Value)
		if err != nil {
			return nil, err
		}
		secrets[secret.Key] = string(plaintext)
	}
	input := map[string]any{}
	if len(record.RequestPayload) > 0 {
		if err := json.Unmarshal(record.RequestPayload, &input); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解析调试任务输入失败", err)
		}
	}
	if record.Operation == "point.read" || record.Operation == "point.write" || record.Operation == "point.subscribe.preview" {
		pointIDs := collectorStringSliceFromAny(input["pointIds"])
		points, err := s.collectorRepository.GetPointsByIDs(ctx, record.ProjectID, record.ConnectionID, pointIDs)
		if err != nil {
			return nil, err
		}
		if len(points) != len(pointIDs) {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "部分调试点位不存在")
		}
		resolved := make([]map[string]any, 0, len(points))
		for _, point := range points {
			resolved = append(resolved, map[string]any{"pointId": point.ID, "address": point.Address, "dataType": point.DataType, "elementCount": point.ElementCount, "readOptions": point.ReadOptions})
		}
		input["points"] = resolved
		delete(input, "pointIds")
	}
	payload, err := json.Marshal(map[string]any{"connectionId": connection.ID, "driverId": connection.DriverID, "driverVersion": connection.DriverVersion, "schemaVersion": connection.SchemaVersion, "connection": map[string]any{"protocolFamily": connection.ProtocolFamily, "config": connection.Config, "secrets": secrets}, "input": input})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "生成调试任务执行信封失败", err)
	}
	return payload, nil
}

func collectorStringSliceFromAny(value any) []string {
	raw, _ := value.([]any)
	result := make([]string, 0, len(raw))
	for _, item := range raw {
		if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
			result = append(result, text)
		}
	}
	return result
}
