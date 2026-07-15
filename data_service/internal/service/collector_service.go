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
	Name     string            `json:"name"`
	DriverID string            `json:"driverId"`
	Config   map[string]any    `json:"config"`
	Secrets  map[string]string `json:"secrets"`
	Metadata map[string]any    `json:"metadata"`
}

type UpdateCollectorConnectionInput struct {
	Name        *string
	Enabled     *bool
	Config      map[string]any
	HasConfig   bool
	Metadata    map[string]any
	HasMetadata bool
	Secrets     map[string]*string
}

type CollectorConnection struct {
	ID             string          `json:"id"`
	ProjectID      string          `json:"projectId"`
	Name           string          `json:"name"`
	Status         string          `json:"status"`
	Enabled        bool            `json:"enabled"`
	DisplayOrder   int             `json:"displayOrder"`
	ProtocolFamily string          `json:"protocolFamily"`
	DriverID       string          `json:"driverId"`
	DriverVersion  string          `json:"driverVersion"`
	SchemaVersion  int             `json:"schemaVersion"`
	Config         map[string]any  `json:"config"`
	Metadata       map[string]any  `json:"metadata"`
	SecretStatus   map[string]bool `json:"secretStatus"`
	CreatedAt      string          `json:"createdAt"`
	UpdatedAt      string          `json:"updatedAt"`
}

type CollectorConnectionPage struct {
	List       []CollectorConnection     `json:"list"`
	Pagination CollectorDriverPagination `json:"pagination"`
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
	if err := s.validateConnectionConfig(driver.Manifest.DriverID, config, input.Secrets, nil); err != nil {
		return nil, err
	}
	secrets, err := s.encryptCreateSecrets(driver.Manifest.DriverID, input.Secrets)
	if err != nil {
		return nil, err
	}
	record, err := s.store.CreateConnection(ctx, repository.CreateCollectorConnectionParams{ID: uuid.NewString(), ProjectID: projectID, UserID: userID, Name: name, ProtocolFamily: driver.Manifest.ProtocolFamily, DriverID: driver.Manifest.DriverID, DriverVersion: driver.Manifest.DriverVersion, SchemaVersion: driver.Manifest.SchemaVersion, Config: config, Metadata: metadata, Secrets: secrets})
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
	enabled := current.Enabled
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	config := cloneCollectorMap(current.Config)
	if input.HasConfig {
		config = cloneCollectorMap(input.Config)
	}
	metadata := cloneCollectorMap(current.Metadata)
	if input.HasMetadata {
		metadata = cloneCollectorMap(input.Metadata)
	}
	if err := s.validateConnectionConfig(current.DriverID, config, nil, mergeCollectorSecretStatus(current.SecretStatus, input.Secrets)); err != nil {
		return nil, err
	}
	upserts, deletes, err := s.encryptUpdatedSecrets(current.DriverID, input.Secrets)
	if err != nil {
		return nil, err
	}
	record, err := s.store.UpdateConnection(ctx, repository.UpdateCollectorConnectionParams{ID: connectionID, ProjectID: projectID, UserID: userID, Name: name, Enabled: enabled, Config: config, Metadata: metadata, Secrets: upserts, DeleteSecretKeys: deletes})
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
	return s.store.DeleteConnection(ctx, projectID, connectionID)
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
	value = strings.TrimSpace(value)
	if value == "" || len([]rune(value)) > 100 {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "工业采集连接名称长度必须为 1 到 100 个字符")
	}
	return value, nil
}

func cloneCollectorMap(value map[string]any) map[string]any {
	result := map[string]any{}
	for key, item := range value {
		result[key] = item
	}
	return result
}

func toCollectorConnection(record repository.CollectorConnectionRecord) CollectorConnection {
	return CollectorConnection{ID: record.ID, ProjectID: record.ProjectID, Name: record.Name, Status: record.Status, Enabled: record.Enabled, DisplayOrder: record.DisplayOrder, ProtocolFamily: record.ProtocolFamily, DriverID: record.DriverID, DriverVersion: record.DriverVersion, SchemaVersion: record.SchemaVersion, Config: cloneCollectorMap(record.Config), Metadata: cloneCollectorMap(record.Metadata), SecretStatus: record.SecretStatus, CreatedAt: record.CreatedAt.Format("2006-01-02 15:04:05"), UpdatedAt: record.UpdatedAt.Format("2006-01-02 15:04:05")}
}
