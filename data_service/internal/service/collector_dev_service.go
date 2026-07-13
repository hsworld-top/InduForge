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
	"time"

	"github.com/google/uuid"
	"github.com/indu-forge/data_service/internal/auth"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

const collectorOnlineWindow = 45 * time.Second

type CollectorDevService struct {
	repository *repository.CollectorDevRepository
}

func NewCollectorDevService(repository *repository.CollectorDevRepository) *CollectorDevService {
	return &CollectorDevService{repository: repository}
}

type CollectorProtocolCapability struct {
	ProtocolType      string   `json:"protocolType"`
	CapabilityVersion string   `json:"capabilityVersion"`
	Operations        []string `json:"operations"`
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
	Online       bool                          `json:"online"`
	Capabilities []CollectorProtocolCapability `json:"capabilities"`
	LastSeenAt   *string                       `json:"lastSeenAt"`
	CreatedAt    string                        `json:"createdAt"`
}
type CollectorRegisterInput struct {
	RegistrationCode string                        `json:"registrationCode"`
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
	Operation      string         `json:"operation"`
	Connection     map[string]any `json:"connection"`
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

func (s *CollectorDevService) Register(ctx context.Context, input CollectorRegisterInput) (*CollectorRegistration, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.RegistrationCode = strings.TrimSpace(input.RegistrationCode)
	if input.Name == "" || input.RegistrationCode == "" {
		return nil, badCollectorRequest("注册码和 Agent 名称不能为空")
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
	record, err := s.repository.RegisterAgent(ctx, repository.CreateCollectorAgentParams{ID: uuid.NewString(), CodeHash: hashSecret(input.RegistrationCode), CredentialHash: hashSecret(token), Name: input.Name, OS: input.OS, Arch: input.Arch, Version: input.Version, Capabilities: input.Capabilities})
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
func (s *CollectorDevService) ListAgents(ctx context.Context, claims *auth.Claims) ([]CollectorAgent, error) {
	if claims == nil || strings.TrimSpace(claims.TenantID) == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeAuthTokenInvalid, http.StatusUnauthorized, "JWT 缺少租户信息")
	}
	records, err := s.repository.ListAgents(ctx, claims.TenantID)
	if err != nil {
		return nil, err
	}
	result := make([]CollectorAgent, 0, len(records))
	now := time.Now()
	for _, record := range records {
		result = append(result, toCollectorAgent(record, now))
	}
	return result, nil
}

func (s *CollectorDevService) DeleteAgent(ctx context.Context, claims *auth.Claims, agentID string) error {
	if !isCollectorAdmin(claims) {
		return apperrors.NewAppError(apperrors.ErrorCodePermissionInsufficient, http.StatusForbidden, "仅系统管理员可以删除 Agent")
	}
	return s.repository.DeleteAgent(ctx, claims.TenantID, agentID)
}
func (s *CollectorDevService) Heartbeat(ctx context.Context, identity *auth.CollectorAgentIdentity, capabilities []CollectorProtocolCapability) (*CollectorAgent, error) {
	if err := validateCapabilities(capabilities); err != nil {
		return nil, err
	}
	record, err := s.repository.Heartbeat(ctx, identity.AgentID, capabilities)
	if err != nil {
		return nil, err
	}
	result := toCollectorAgent(*record, time.Now())
	return &result, nil
}
func (s *CollectorDevService) ClaimTask(ctx context.Context, identity *auth.CollectorAgentIdentity) (*CollectorTask, error) {
	record, err := s.repository.ClaimTask(ctx, identity.AgentID)
	if err != nil || record == nil {
		return nil, err
	}
	result := toCollectorTask(*record)
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
	request := map[string]any{"connection": input.Connection, "input": input.Input}
	record, err := s.repository.CreateTask(ctx, repository.CreateCollectorTaskParams{ID: uuid.NewString(), TenantID: claims.TenantID, ProjectID: projectID, AgentID: input.AgentID, Operation: input.Operation, RequestPayload: request, DeadlineAt: time.Now().Add(time.Duration(input.TimeoutSeconds) * time.Second), CreatedBy: claims.UserID})
	if err != nil {
		return nil, err
	}
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
	allowed := map[string]bool{"connection.test": true, "opcua.browse": true, "opcua.read": true}
	if !allowed[input.Operation] {
		return badCollectorRequest("不支持的调试任务操作")
	}
	if input.TimeoutSeconds < 0 || input.TimeoutSeconds > 120 {
		return badCollectorRequest("timeoutSeconds 必须在 1 到 120 秒之间")
	}
	if input.Connection["protocolType"] != "opcua" {
		return badCollectorRequest("仅支持 OPC UA 调试任务")
	}
	authentication, _ := input.Connection["authentication"].(map[string]any)
	if authentication != nil && authentication["type"] != "anonymous" {
		return badCollectorRequest("当前仅支持 OPC UA 匿名认证")
	}
	if input.Operation == "opcua.browse" {
		if depth, ok := input.Input["maxDepth"].(float64); ok && depth != 1 {
			return badCollectorRequest("Browse 仅支持 maxDepth = 1")
		}
	}
	if input.Operation == "opcua.read" {
		if ids, ok := input.Input["nodeIds"].([]any); !ok || len(ids) == 0 || len(ids) > 500 {
			return badCollectorRequest("Read 节点数必须在 1 到 500 之间")
		}
	}
	return nil
}
func validateCapabilities(capabilities []CollectorProtocolCapability) error {
	for _, capability := range capabilities {
		if capability.ProtocolType != "opcua" {
			return badCollectorRequest("Agent 仅允许声明平台支持的协议能力")
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
	return CollectorAgent{ID: record.ID, Name: record.Name, OS: record.OS, Arch: record.Arch, Version: record.Version, Online: record.LastSeenAt != nil && now.Sub(*record.LastSeenAt) <= collectorOnlineWindow, Capabilities: capabilities, LastSeenAt: optionalCollectorTime(record.LastSeenAt), CreatedAt: formatCollectorTime(record.CreatedAt)}
}
func toCollectorTask(record repository.CollectorDevTaskRecord) CollectorTask {
	return CollectorTask{ID: record.ID, ProjectID: record.ProjectID, AgentID: record.AgentID, Operation: record.Operation, Status: record.Status, Request: record.RequestPayload, Result: record.ResultPayload, ErrorCode: record.ErrorCode, ErrorMessage: record.ErrorMessage, DeadlineAt: formatCollectorTime(record.DeadlineAt), ClaimedAt: optionalCollectorTime(record.ClaimedAt), FinishedAt: optionalCollectorTime(record.FinishedAt), CreatedAt: formatCollectorTime(record.CreatedAt)}
}
