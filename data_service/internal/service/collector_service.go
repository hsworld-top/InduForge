package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/indu-forge/data_service/internal/collectorprotocol"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
	collectorsecurity "github.com/indu-forge/data_service/internal/security"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

type CollectorConnectionStore interface {
	ListConnections(context.Context, string, repository.CollectorConnectionListFilter) ([]repository.CollectorConnectionRecord, int, error)
	GetConnection(context.Context, string, string) (*repository.CollectorConnectionRecord, error)
	CreateConnection(context.Context, repository.CreateCollectorConnectionParams) (*repository.CollectorConnectionRecord, error)
	UpdateConnection(context.Context, repository.UpdateCollectorConnectionParams) (*repository.CollectorConnectionRecord, error)
	DeleteConnection(context.Context, string, string) error
}

type CreateCollectorConnectionInput struct {
	Name               string            `json:"name"`
	DriverID           string            `json:"driverId"`
	Config             map[string]any    `json:"config"`
	Secrets            map[string]string `json:"secrets"`
	Metadata           map[string]any    `json:"metadata"`
	IsEnabled          *bool             `json:"isEnabled"`
	DefaultAcquisition map[string]any    `json:"defaultAcquisition"`
}

type UpdateCollectorConnectionInput struct {
	Name                  *string
	Config                map[string]any
	HasConfig             bool
	Metadata              map[string]any
	HasMetadata           bool
	Secrets               map[string]*string
	IsEnabled             *bool
	DefaultAcquisition    map[string]any
	HasDefaultAcquisition bool
}

type CollectorConnection struct {
	ID                 string          `json:"id"`
	ProjectID          string          `json:"projectId"`
	Name               string          `json:"name"`
	Code               string          `json:"code"`
	IsEnabled          bool            `json:"isEnabled"`
	ConfigurationState string          `json:"configurationState"`
	DefaultAcquisition map[string]any  `json:"defaultAcquisition"`
	DisplayOrder       int             `json:"displayOrder"`
	ProtocolFamily     string          `json:"protocolFamily"`
	DriverID           string          `json:"driverId"`
	DriverVersion      string          `json:"driverVersion"`
	SchemaVersion      int             `json:"schemaVersion"`
	Config             map[string]any  `json:"config"`
	Metadata           map[string]any  `json:"metadata"`
	SecretStatus       map[string]bool `json:"secretStatus"`
	LastTestStatus     *string         `json:"lastTestStatus"`
	LastTestedAt       *string         `json:"lastTestedAt"`
	CreatedAt          string          `json:"createdAt"`
	UpdatedAt          string          `json:"updatedAt"`
}

type CollectorConnectionPage struct {
	List       []CollectorConnection     `json:"list"`
	Pagination CollectorDriverPagination `json:"pagination"`
}

type CollectorConnectionDiagnostic struct {
	Agent                 map[string]any                              `json:"agent"`
	LastTest              map[string]any                              `json:"lastTest"`
	PointCount            int                                         `json:"pointCount"`
	AttemptedPointCount   int                                         `json:"attemptedPointCount"`
	SucceededPointCount   int                                         `json:"succeededPointCount"`
	FailedPointCount      int                                         `json:"failedPointCount"`
	RecentReadSuccessRate *float64                                    `json:"recentReadSuccessRate"`
	FailedPoints          []repository.CollectorFailedPointDiagnostic `json:"failedPoints"`
}

type CollectorService struct {
	store             CollectorConnectionStore
	catalog           *collectorprotocol.Catalog
	cipher            *collectorsecurity.CollectorSecretCipher
	connectionSchemas map[string]*jsonschema.Schema
	secretFields      map[string]map[string]struct{}
}

func NewCollectorService(store CollectorConnectionStore, catalog *collectorprotocol.Catalog, secretCipher *collectorsecurity.CollectorSecretCipher) (*CollectorService, error) {
	if store == nil || catalog == nil || secretCipher == nil {
		return nil, fmt.Errorf("工业采集连接服务依赖不完整")
	}
	result := &CollectorService{store: store, catalog: catalog, cipher: secretCipher, connectionSchemas: map[string]*jsonschema.Schema{}, secretFields: map[string]map[string]struct{}{}}
	for _, driver := range catalog.Drivers() {
		var document any
		if err := json.Unmarshal(driver.ConnectionSchema, &document); err != nil {
			return nil, fmt.Errorf("解析驱动 %s Connection Schema 失败: %w", driver.Manifest.DriverID, err)
		}
		compiler := jsonschema.NewCompiler()
		compiler.DefaultDraft(jsonschema.Draft2020)
		resource := "urn:induforge:collector:" + driver.Manifest.DriverID + ":connection"
		if err := compiler.AddResource(resource, document); err != nil {
			return nil, fmt.Errorf("注册驱动 %s Connection Schema 失败: %w", driver.Manifest.DriverID, err)
		}
		schema, err := compiler.Compile(resource)
		if err != nil {
			return nil, fmt.Errorf("编译驱动 %s Connection Schema 失败: %w", driver.Manifest.DriverID, err)
		}
		result.connectionSchemas[driver.Manifest.DriverID] = schema
		result.secretFields[driver.Manifest.DriverID] = collectSecretFields(document)
	}
	return result, nil
}

func (s *CollectorService) ListConnections(ctx context.Context, projectID string, filter repository.CollectorConnectionListFilter) (CollectorConnectionPage, error) {
	if err := validateProjectID(projectID); err != nil {
		return CollectorConnectionPage{}, err
	}
	if filter.Page < 1 || filter.PageSize < 1 || filter.PageSize > 100 {
		return CollectorConnectionPage{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "分页参数无效")
	}
	records, total, err := s.store.ListConnections(ctx, projectID, filter)
	if err != nil {
		return CollectorConnectionPage{}, err
	}
	list := make([]CollectorConnection, 0, len(records))
	for _, record := range records {
		list = append(list, toCollectorConnection(record))
	}
	totalPages := 0
	if total > 0 {
		totalPages = (total + filter.PageSize - 1) / filter.PageSize
	}
	return CollectorConnectionPage{List: list, Pagination: CollectorDriverPagination{Page: filter.Page, PageSize: filter.PageSize, Total: total, TotalPages: totalPages}}, nil
}

func (s *CollectorService) GetConnection(ctx context.Context, projectID, connectionID string) (*CollectorConnection, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateConnectionID(connectionID); err != nil {
		return nil, err
	}
	record, err := s.store.GetConnection(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	value := toCollectorConnection(*record)
	return &value, nil
}

func (s *CollectorService) CreateConnection(ctx context.Context, projectID, userID string, input CreateCollectorConnectionInput) (*CollectorConnection, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	name, err := normalizeCollectorConnectionName(input.Name)
	if err != nil {
		return nil, err
	}
	driver, ok := s.catalog.Driver(strings.TrimSpace(input.DriverID))
	if !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "工业采集驱动不存在")
	}
	config := cloneCollectorMap(input.Config)
	metadata := cloneCollectorMap(input.Metadata)
	defaultAcquisition, err := normalizeCollectorDefaultAcquisition(input.DefaultAcquisition)
	if err != nil {
		return nil, err
	}
	isEnabled := true
	if input.IsEnabled != nil {
		isEnabled = *input.IsEnabled
	}
	if err := s.validateConnectionConfig(driver.Manifest.DriverID, config, input.Secrets, nil); err != nil {
		return nil, err
	}
	secrets, err := s.encryptCreateSecrets(driver.Manifest.DriverID, input.Secrets)
	if err != nil {
		return nil, err
	}
	connectionID := uuid.NewString()
	record, err := s.store.CreateConnection(ctx, repository.CreateCollectorConnectionParams{ID: connectionID, ProjectID: projectID, UserID: userID, Name: name, Code: collectorCodeFromName(name, "connection"), ProtocolFamily: driver.Manifest.ProtocolFamily, DriverID: driver.Manifest.DriverID, DriverVersion: driver.Manifest.DriverVersion, SchemaVersion: driver.Manifest.SchemaVersion, Config: config, Metadata: metadata, DefaultAcquisition: defaultAcquisition, IsEnabled: isEnabled, Secrets: secrets})
	if err != nil {
		return nil, err
	}
	value := toCollectorConnection(*record)
	return &value, nil
}

func (s *CollectorService) UpdateConnection(ctx context.Context, projectID, connectionID, userID string, input UpdateCollectorConnectionInput) (*CollectorConnection, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateConnectionID(connectionID); err != nil {
		return nil, err
	}
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	current, err := s.store.GetConnection(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	name := current.Name
	if input.Name != nil {
		name, err = normalizeCollectorConnectionName(*input.Name)
		if err != nil {
			return nil, err
		}
	}

	config := cloneCollectorMap(current.Config)
	if input.HasConfig {
		config = cloneCollectorMap(input.Config)
	}
	metadata := cloneCollectorMap(current.Metadata)
	if input.HasMetadata {
		metadata = cloneCollectorMap(input.Metadata)
	}
	defaultAcquisition := cloneCollectorMap(current.DefaultAcquisition)
	if input.HasDefaultAcquisition {
		defaultAcquisition, err = normalizeCollectorDefaultAcquisition(input.DefaultAcquisition)
		if err != nil {
			return nil, err
		}
	}
	isEnabled := current.IsEnabled
	if input.IsEnabled != nil {
		isEnabled = *input.IsEnabled
	}
	if err := s.validateConnectionConfig(current.DriverID, config, nil, mergeCollectorSecretStatus(current.SecretStatus, input.Secrets)); err != nil {
		return nil, err
	}
	upserts, deletes, err := s.encryptUpdatedSecrets(current.DriverID, input.Secrets)
	if err != nil {
		return nil, err
	}
	record, err := s.store.UpdateConnection(ctx, repository.UpdateCollectorConnectionParams{ID: connectionID, ProjectID: projectID, UserID: userID, Name: name, Config: config, Metadata: metadata, DefaultAcquisition: defaultAcquisition, IsEnabled: isEnabled, Secrets: upserts, DeleteSecretKeys: deletes})
	if err != nil {
		return nil, err
	}
	value := toCollectorConnection(*record)
	return &value, nil
}

func (s *CollectorService) DeleteConnection(ctx context.Context, projectID, connectionID string) error {
	if err := validateProjectID(projectID); err != nil {
		return err
	}
	if err := validateConnectionID(connectionID); err != nil {
		return err
	}
	if repositoryStore, ok := s.store.(*repository.CollectorRepository); ok {
		return repositoryStore.DeleteWithImpact(ctx, projectID, connectionID, "")
	}
	return s.store.DeleteConnection(ctx, projectID, connectionID)
}

func (s *CollectorService) GetConnectionDeleteImpact(ctx context.Context, projectID, connectionID string) (*repository.SourceDeleteImpactRecord, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateConnectionID(connectionID); err != nil {
		return nil, err
	}
	repositoryStore, ok := s.store.(*repository.CollectorRepository)
	if !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "工业采集仓储不支持删除影响检查")
	}
	return repositoryStore.GetDeleteImpact(ctx, projectID, connectionID)
}

func (s *CollectorService) GetConnectionDiagnostic(ctx context.Context, projectID, connectionID string) (*CollectorConnectionDiagnostic, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateConnectionID(connectionID); err != nil {
		return nil, err
	}
	repositoryStore, ok := s.store.(*repository.CollectorRepository)
	if !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "工业采集仓储不支持诊断摘要")
	}
	record, err := repositoryStore.GetConnectionDiagnostic(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	var successRate *float64
	if record.AttemptedPointCount > 0 {
		value := float64(record.SucceededPointCount) / float64(record.AttemptedPointCount)
		successRate = &value
	}
	agent := map[string]any{"ready": record.AgentReady, "reason": record.AgentReason}
	if record.AgentID != "" {
		agent["id"] = record.AgentID
		agent["name"] = record.AgentName
		agent["version"] = record.AgentVersion
		agent["lastSeenAt"] = record.AgentLastSeenAt
	}
	lastTest := map[string]any{"status": record.LastTestStatus, "durationMs": record.LastTestDurationMS}
	if record.LastTestAt != nil {
		lastTest["testedAt"] = record.LastTestAt
	}
	if record.LastErrorCode != "" {
		lastTest["errorCode"] = record.LastErrorCode
	}
	if record.LastErrorMessage != "" {
		lastTest["message"] = record.LastErrorMessage
	}
	return &CollectorConnectionDiagnostic{Agent: agent, LastTest: lastTest, PointCount: record.PointCount,
		AttemptedPointCount: record.AttemptedPointCount, SucceededPointCount: record.SucceededPointCount,
		FailedPointCount: len(record.FailedPoints), RecentReadSuccessRate: successRate, FailedPoints: record.FailedPoints}, nil
}

func (s *CollectorService) validateConnectionConfig(driverID string, config map[string]any, createSecrets map[string]string, secretStatus map[string]bool) error {
	secretFields := s.secretFields[driverID]
	document := cloneCollectorMap(config)
	for key := range config {
		if _, secret := secretFields[key]; secret {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "敏感字段必须通过 secrets 提交")
		}
	}
	for key, value := range createSecrets {
		if _, secret := secretFields[key]; !secret {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "提交了驱动未声明的敏感字段")
		}
		document[key] = value
	}
	for key, configured := range secretStatus {
		if configured {
			document[key] = "configured"
		}
	}
	if err := s.connectionSchemas[driverID].Validate(document); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "连接配置不符合驱动 Schema", err)
	}
	return nil
}

func (s *CollectorService) encryptCreateSecrets(driverID string, values map[string]string) ([]repository.EncryptedCollectorSecret, error) {
	result := make([]repository.EncryptedCollectorSecret, 0, len(values))
	for key, value := range values {
		if _, ok := s.secretFields[driverID][key]; !ok {
			continue
		}
		encrypted, err := s.cipher.Encrypt([]byte(value))
		if err != nil {
			return nil, err
		}
		result = append(result, repository.EncryptedCollectorSecret{Key: key, Value: encrypted, KeyVersion: s.cipher.KeyVersion()})
	}
	return result, nil
}

func (s *CollectorService) encryptUpdatedSecrets(driverID string, values map[string]*string) ([]repository.EncryptedCollectorSecret, []string, error) {
	upserts := make([]repository.EncryptedCollectorSecret, 0)
	deletes := make([]string, 0)
	for key, value := range values {
		if _, ok := s.secretFields[driverID][key]; !ok {
			return nil, nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "提交了驱动未声明的敏感字段")
		}
		if value == nil {
			deletes = append(deletes, key)
			continue
		}
		encrypted, err := s.cipher.Encrypt([]byte(*value))
		if err != nil {
			return nil, nil, err
		}
		upserts = append(upserts, repository.EncryptedCollectorSecret{Key: key, Value: encrypted, KeyVersion: s.cipher.KeyVersion()})
	}
	return upserts, deletes, nil
}

func collectSecretFields(document any) map[string]struct{} {
	result := map[string]struct{}{}
	root, _ := document.(map[string]any)
	properties, _ := root["properties"].(map[string]any)
	for key, value := range properties {
		property, _ := value.(map[string]any)
		if secret, _ := property["x-induforge-secret"].(bool); secret {
			result[key] = struct{}{}
		}
	}
	return result
}

func mergeCollectorSecretStatus(current map[string]bool, updates map[string]*string) map[string]bool {
	result := map[string]bool{}
	for key, value := range current {
		result[key] = value
	}
	for key, value := range updates {
		result[key] = value != nil
	}
	return result
}

func normalizeCollectorConnectionName(value string) (string, error) {
	name, err := normalizeCollectorDisplayName(value, 50)
	if err != nil {
		return "", apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "工业采集连接名称无效", err)
	}
	return name, nil
}

func cloneCollectorMap(value map[string]any) map[string]any {
	result := map[string]any{}
	for key, item := range value {
		result[key] = item
	}
	return result
}

func cloneCollectorSecretStatus(value map[string]bool) map[string]bool {
	result := map[string]bool{}
	for key, item := range value {
		result[key] = item
	}
	return result
}

func toCollectorConnection(record repository.CollectorConnectionRecord) CollectorConnection {
	var lastTestedAt *string
	if record.LastTestedAt != nil {
		value := record.LastTestedAt.Format("2006-01-02 15:04:05")
		lastTestedAt = &value
	}
	configurationState := "ready"
	if len(record.Config) == 0 {
		configurationState = "incomplete"
	}
	return CollectorConnection{ID: record.ID, ProjectID: record.ProjectID, Name: record.Name, Code: record.Code, IsEnabled: record.IsEnabled, ConfigurationState: configurationState, DefaultAcquisition: cloneCollectorMap(record.DefaultAcquisition), DisplayOrder: record.DisplayOrder, ProtocolFamily: record.ProtocolFamily, DriverID: record.DriverID, DriverVersion: record.DriverVersion, SchemaVersion: record.SchemaVersion, Config: cloneCollectorMap(record.Config), Metadata: cloneCollectorMap(record.Metadata), SecretStatus: cloneCollectorSecretStatus(record.SecretStatus), LastTestStatus: record.LastTestStatus, LastTestedAt: lastTestedAt, CreatedAt: record.CreatedAt.Format("2006-01-02 15:04:05"), UpdatedAt: record.UpdatedAt.Format("2006-01-02 15:04:05")}
}

func normalizeCollectorDefaultAcquisition(input map[string]any) (map[string]any, error) {
	result := map[string]any{"intervalMs": 1000, "timeoutMs": 3000, "retryCount": 0, "deadband": 0, "changeOnly": false, "priority": 0}
	for key, value := range input {
		result[key] = value
	}
	for _, key := range []string{"intervalMs", "timeoutMs"} {
		value, ok := result[key].(float64)
		if ok && value <= 0 {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, key+" 必须为正数")
		}
	}
	return result, nil
}
