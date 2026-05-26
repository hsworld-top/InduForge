package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/indu-forge/data_service/internal/app"
	"github.com/indu-forge/data_service/internal/auth"
	"github.com/indu-forge/data_service/internal/config"
)

func TestKafkaWorkbenchLifecycle(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupMigrator(t, fixture.pool)
	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("migrate up failed: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "kafka-workbench-secret-01"

	srv, err := app.NewServer(config.Config{
		Addr:               ":0",
		DatabaseURL:        fixture.databaseURL,
		DatabaseSearchPath: fixture.schemaName,
		JWTSecret:          secret,
	})
	if err != nil {
		t.Fatalf("create server failed: %v", err)
	}
	t.Cleanup(srv.Close)

	server := httptest.NewServer(srv.Handler())
	t.Cleanup(server.Close)

	token := mustSignIntegrationJWT(t, secret, &auth.Claims{
		UserID:       userID,
		TenantID:     "tenant-kafka-workbench",
		ProjectIDs:   []string{projectID},
		Capabilities: []string{"project:read", "project:write"},
	})

	kafka := mustCreateKafkaConfig(t, server.URL, token, projectID, map[string]any{
		"name":          "kafka-workbench-main",
		"brokers":       "127.0.0.1:9092",
		"topic":         "device.default",
		"consumerGroup": "if-preview-default",
		"startPosition": "latest",
	})

	group := mustCreateKafkaTopicGroup(t, server.URL, token, projectID, kafka.ID, map[string]any{
		"name": "产线 A",
	})
	mapping := mustCreateKafkaTopicMapping(t, server.URL, token, projectID, kafka.ID, map[string]any{
		"name":          "设备遥测",
		"topic":         "device.telemetry",
		"groupId":       group.ID,
		"partitionMode": "all",
		"startPosition": "latest",
		"decode":        "json",
		"sampleLimit":   100,
		"timeoutMs":     5000,
	})
	fieldGroup := mustCreateKafkaFieldGroup(t, server.URL, token, projectID, mapping.ID, map[string]any{
		"name": "实时遥测",
	})

	mappings := mustListKafkaTopicMappings(t, server.URL, token, projectID, kafka.ID)
	if len(mappings) != 1 || mappings[0].Topic != "device.telemetry" {
		t.Fatalf("expected created topic mapping, got %#v", mappings)
	}

	field := mustCreateKafkaField(t, server.URL, token, projectID, mapping.ID, map[string]any{
		"name":      "temperature",
		"valuePath": "temperature",
		"dataType":  "number",
		"enabled":   true,
		"groupId":   fieldGroup.ID,
	})
	mustCreateKafkaField(t, server.URL, token, projectID, mapping.ID, map[string]any{
		"name":      "humidity",
		"valuePath": "humidity",
		"dataType":  "number",
		"enabled":   true,
	})
	groupedFields := mustListKafkaFields(t, server.URL, token, projectID, mapping.ID, "groupId="+url.QueryEscape(fieldGroup.ID)+"&page=1&pageSize=1")
	if groupedFields.Pagination.Total != 1 || len(groupedFields.List) != 1 || groupedFields.List[0].GroupID == nil || *groupedFields.List[0].GroupID != fieldGroup.ID {
		t.Fatalf("expected one paged kafka field in group, got %#v", groupedFields)
	}
	ungroupedFields := mustListKafkaFields(t, server.URL, token, projectID, mapping.ID, "groupId=__ungrouped&page=1&pageSize=20")
	if ungroupedFields.Pagination.Total != 1 || len(ungroupedFields.List) != 1 || ungroupedFields.List[0].Name != "humidity" {
		t.Fatalf("expected one ungrouped kafka field, got %#v", ungroupedFields)
	}
	if field.SourceType != "kafka.field" {
		t.Fatalf("expected kafka.field datapoint sync, got %q", field.SourceType)
	}
	if field.Quality != "unknown" {
		t.Fatalf("expected initial kafka field quality unknown, got %q", field.Quality)
	}
	field = mustUpdateKafkaField(t, server.URL, token, projectID, field.ID, map[string]any{
		"name":      "temperature2",
		"valuePath": "payload.temperature2",
		"dataType":  "string",
		"enabled":   true,
	})
	if field.DataPointPath != "kafka.设备遥测.temperature2" {
		t.Fatalf("expected updated datapoint path, got %q", field.DataPointPath)
	}

	query := url.Values{}
	query.Set("type", "kafka.field")
	query.Set("sourceId", kafka.ID)
	query.Set("search", "temperature2")
	datapoints := mustListDataPoints(t, server.URL, token, projectID, query.Encode())
	if len(datapoints.DataPoints) != 1 {
		t.Fatalf("expected one kafka.field datapoint after search filter, got %d", len(datapoints.DataPoints))
	}

	mustDeleteKafkaFieldGroup(t, server.URL, token, projectID, fieldGroup.ID)
	ungroupedFields = mustListKafkaFields(t, server.URL, token, projectID, mapping.ID, "groupId=__ungrouped&page=1&pageSize=20")
	if ungroupedFields.Pagination.Total != 2 {
		t.Fatalf("expected deleted kafka field group to move fields to ungrouped, got total %d", ungroupedFields.Pagination.Total)
	}

	mustDeleteKafkaTopicMapping(t, server.URL, token, projectID, mapping.ID)
	mappings = mustListKafkaTopicMappings(t, server.URL, token, projectID, kafka.ID)
	if len(mappings) != 0 {
		t.Fatalf("expected mapping deleted, got %d", len(mappings))
	}
}

type kafkaTopicGroupPayload struct {
	ID           string  `json:"id"`
	ConnectionID string  `json:"connectionId"`
	ParentID     *string `json:"parentId"`
	Name         string  `json:"name"`
	SortOrder    int     `json:"sortOrder"`
}

type kafkaTopicMappingPayload struct {
	ID            string  `json:"id"`
	ConnectionID  string  `json:"connectionId"`
	GroupID       *string `json:"groupId"`
	Name          string  `json:"name"`
	Topic         string  `json:"topic"`
	PartitionMode string  `json:"partitionMode"`
	Partition     *int    `json:"partition"`
	StartPosition string  `json:"startPosition"`
	Decode        string  `json:"decode"`
	SampleLimit   int     `json:"sampleLimit"`
	TimeoutMS     int     `json:"timeoutMs"`
}

type kafkaFieldPayload struct {
	ID             string  `json:"id"`
	ConnectionID   string  `json:"connectionId"`
	TopicMappingID string  `json:"topicMappingId"`
	GroupID        *string `json:"groupId"`
	Name           string  `json:"name"`
	ValuePath      string  `json:"valuePath"`
	DataType       string  `json:"dataType"`
	Enabled        bool    `json:"enabled"`
	SourceType     string  `json:"sourceType"`
	DataPointID    string  `json:"dataPointId"`
	DataPointPath  string  `json:"dataPointPath"`
	Quality        string  `json:"quality"`
}

type kafkaFieldGroupPayload struct {
	ID             string  `json:"id"`
	TopicMappingID string  `json:"topicMappingId"`
	ParentID       *string `json:"parentId"`
	Name           string  `json:"name"`
	SortOrder      int     `json:"sortOrder"`
}

type kafkaListPayload[T any] struct {
	List       []T                `json:"list"`
	Pagination mqttPaginationData `json:"pagination"`
}

func mustCreateKafkaTopicGroup(t *testing.T, baseURL, token, projectID, connectionID string, payload map[string]any) kafkaTopicGroupPayload {
	t.Helper()

	responseEnvelope := doJSONRequest(t, http.MethodPost, baseURL+"/api/v1/data/projects/"+projectID+"/kafka/sources/"+connectionID+"/topic-groups", token, payload)
	var result kafkaTopicGroupPayload
	if err := json.Unmarshal(responseEnvelope.Data, &result); err != nil {
		t.Fatalf("decode kafka topic group response failed: %v", err)
	}
	return result
}

func mustCreateKafkaTopicMapping(t *testing.T, baseURL, token, projectID, connectionID string, payload map[string]any) kafkaTopicMappingPayload {
	t.Helper()

	responseEnvelope := doJSONRequest(t, http.MethodPost, baseURL+"/api/v1/data/projects/"+projectID+"/kafka/sources/"+connectionID+"/topic-mappings", token, payload)
	var result kafkaTopicMappingPayload
	if err := json.Unmarshal(responseEnvelope.Data, &result); err != nil {
		t.Fatalf("decode kafka topic mapping response failed: %v", err)
	}
	return result
}

func mustListKafkaTopicMappings(t *testing.T, baseURL, token, projectID, connectionID string) []kafkaTopicMappingPayload {
	t.Helper()

	responseEnvelope := doJSONRequest(t, http.MethodGet, baseURL+"/api/v1/data/projects/"+projectID+"/kafka/sources/"+connectionID+"/topic-mappings", token, nil)
	var result kafkaListPayload[kafkaTopicMappingPayload]
	if err := json.Unmarshal(responseEnvelope.Data, &result); err != nil {
		t.Fatalf("decode kafka topic mappings response failed: %v", err)
	}
	return result.List
}

func mustCreateKafkaFieldGroup(t *testing.T, baseURL, token, projectID, mappingID string, payload map[string]any) kafkaFieldGroupPayload {
	t.Helper()

	responseEnvelope := doJSONRequest(t, http.MethodPost, baseURL+"/api/v1/data/projects/"+projectID+"/kafka/topic-mappings/"+mappingID+"/field-groups", token, payload)
	var result kafkaFieldGroupPayload
	if err := json.Unmarshal(responseEnvelope.Data, &result); err != nil {
		t.Fatalf("decode kafka field group response failed: %v", err)
	}
	return result
}

func mustCreateKafkaField(t *testing.T, baseURL, token, projectID, mappingID string, payload map[string]any) kafkaFieldPayload {
	t.Helper()

	responseEnvelope := doJSONRequest(t, http.MethodPost, baseURL+"/api/v1/data/projects/"+projectID+"/kafka/topic-mappings/"+mappingID+"/fields", token, payload)
	var result kafkaFieldPayload
	if err := json.Unmarshal(responseEnvelope.Data, &result); err != nil {
		t.Fatalf("decode kafka field response failed: %v", err)
	}
	return result
}

func mustListKafkaFields(t *testing.T, baseURL, token, projectID, mappingID, rawQuery string) kafkaListPayload[kafkaFieldPayload] {
	t.Helper()

	endpoint := baseURL + "/api/v1/data/projects/" + projectID + "/kafka/topic-mappings/" + mappingID + "/fields"
	if rawQuery != "" {
		endpoint += "?" + rawQuery
	}
	responseEnvelope := doJSONRequest(t, http.MethodGet, endpoint, token, nil)
	var result kafkaListPayload[kafkaFieldPayload]
	if err := json.Unmarshal(responseEnvelope.Data, &result); err != nil {
		t.Fatalf("decode kafka fields response failed: %v", err)
	}
	return result
}

func mustUpdateKafkaField(t *testing.T, baseURL, token, projectID, fieldID string, payload map[string]any) kafkaFieldPayload {
	t.Helper()

	responseEnvelope := doJSONRequest(t, http.MethodPut, baseURL+"/api/v1/data/projects/"+projectID+"/kafka/fields/"+fieldID, token, payload)
	var result kafkaFieldPayload
	if err := json.Unmarshal(responseEnvelope.Data, &result); err != nil {
		t.Fatalf("decode kafka field update response failed: %v", err)
	}
	return result
}

func mustDeleteKafkaFieldGroup(t *testing.T, baseURL, token, projectID, groupID string) {
	t.Helper()

	_ = doJSONRequest(t, http.MethodDelete, baseURL+"/api/v1/data/projects/"+projectID+"/kafka/field-groups/"+groupID, token, nil)
}

func mustDeleteKafkaTopicMapping(t *testing.T, baseURL, token, projectID, mappingID string) {
	t.Helper()

	_ = doJSONRequest(t, http.MethodDelete, baseURL+"/api/v1/data/projects/"+projectID+"/kafka/topic-mappings/"+mappingID, token, nil)
}
