package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/indu-forge/data_service/internal/app"
	"github.com/indu-forge/data_service/internal/auth"
	"github.com/indu-forge/data_service/internal/config"
)

func TestProjectArtifactV1Contract(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupSchemaInitializer(t, fixture.pool)
	if err := migrator.Ensure(ctx); err != nil {
		t.Fatalf("schema initialization failed: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "snapshot-artifact-secret-01"

	srv, err := app.NewServer(config.Config{
		Addr:                       ":0",
		DatabaseURL:                fixture.databaseURL,
		DatabaseSearchPath:         fixture.schemaName,
		JWTSecret:                  secret,
		ConnectionSecretKey:        []byte("0123456789abcdef0123456789abcdef"),
		ConnectionSecretKeyVersion: "v1",
	})
	if err != nil {
		t.Fatalf("create server failed: %v", err)
	}
	t.Cleanup(srv.Close)

	server := httptest.NewServer(srv.Handler())
	t.Cleanup(server.Close)

	token := mustSignIntegrationJWT(t, secret, &auth.Claims{
		UserID:       userID,
		TenantID:     "tenant-artifact",
		ProjectIDs:   []string{projectID},
		Capabilities: []string{"project:read", "project:write"},
	})

	relational := mustCreateConnection(t, server.URL, token, projectID, map[string]any{
		"name":    "pg-main",
		"type":    "relational",
		"enabled": true,
		"config": map[string]any{
			"host":     "127.0.0.1",
			"port":     5432,
			"database": "factory",
		},
	})

	query := mustCreateQuery(t, server.URL, token, projectID, map[string]any{
		"name":         "query-main",
		"connectionId": relational.ID,
		"queryType":    "sql",
		"config": map[string]any{
			"sql": "SELECT 1 AS value",
		},
	})

	dataPointID := insertOneDataPoint(t, ctx, fixture, projectID, userID, "metrics.main.temp", "active")
	_, err = fixture.pool.Exec(ctx, `
        UPDATE data_points
        SET runtime_permissions = $3::jsonb
        WHERE project_id = $1 AND id = $2
    `, projectID, dataPointID, `{"write":{"allowRoles":["operator"],"denyRoles":["guest"],"inherit":false}}`)
	if err != nil {
		t.Fatalf("update datapoint runtime permissions failed: %v", err)
	}

	mqttConnection := mustCreateMqttConnection(t, server.URL, token, projectID, map[string]any{
		"name":      "mqtt-main",
		"brokerUrl": "tcp://localhost:1883",
		"protocol":  "mqtt",
		"port":      1883,
	})
	subscriptionID := insertTestMqttSubscription(t, ctx, fixture, projectID, mqttConnection.ID, userID)
	tagID := insertTestMqttTag(t, ctx, fixture, projectID, subscriptionID, userID)

	kafkaConnection := mustCreateKafkaConfig(t, server.URL, token, projectID, map[string]any{
		"name":          "kafka-main",
		"brokers":       "127.0.0.1:9092",
		"topic":         "factory.events",
		"consumerGroup": "artifact-group",
	})
	computeUnit := mustCreateComputeUnit(t, server.URL, token, projectID, map[string]any{
		"name": "weekly-report", "language": "js", "scriptCode": "return null;",
		"triggerType":   "schedule",
		"triggerConfig": map[string]any{"kind": "weekly", "weekdays": []int{1, 3, 5}, "time": "08:30:00", "timezone": "Asia/Shanghai"},
	})
	mustCreateConnection(t, server.URL, token, projectID, map[string]any{
		"name": "IF关系库",
		"type": "builtin.relation",
		"config": map[string]any{
			"host": "127.0.0.1",
		},
	})
	mustCreateConnection(t, server.URL, token, projectID, map[string]any{
		"name": "IF时序库",
		"type": "builtin.timeseries",
		"config": map[string]any{
			"retentionDays": 30,
		},
	})
	mustCreateConnection(t, server.URL, token, projectID, map[string]any{
		"name": "IF实时库",
		"type": "builtin.realtime",
		"config": map[string]any{
			"defaultTtlSeconds": 300,
		},
	})
	mustCreateConnection(t, server.URL, token, projectID, map[string]any{
		"name": "IF消息库",
		"type": "builtin.message",
	})

	responseEnvelope := doJSONRequest(t, http.MethodGet, server.URL+"/api/v1/data/projects/"+projectID+"/artifact", token, nil)
	var artifact projectArtifactPayload
	if err := json.Unmarshal(responseEnvelope.Data, &artifact); err != nil {
		t.Fatalf("decode artifact payload failed: %v", err)
	}

	if artifact.Version != "1.0" {
		t.Fatalf("expected version=1.0, got %q", artifact.Version)
	}
	if artifact.ProjectID != projectID {
		t.Fatalf("expected projectId=%q, got %q", projectID, artifact.ProjectID)
	}
	if artifact.GeneratedAt.IsZero() {
		t.Fatal("expected generatedAt to be non-zero")
	}

	if !artifactHasConnection(artifact.Connections, relational.ID, "relational") {
		t.Fatalf("expected relational connection %q in artifact", relational.ID)
	}
	if !artifactHasConnection(artifact.Connections, mqttConnection.ID, "mqtt") {
		t.Fatalf("expected mqtt connection %q in artifact", mqttConnection.ID)
	}
	if !artifactHasQuery(artifact.Queries, query.ID) {
		t.Fatalf("expected query %q in artifact", query.ID)
	}
	if !artifactHasDataPointPath(artifact.DataPoints, "metrics.main.temp") {
		t.Fatal("expected datapoint metrics.main.temp in artifact")
	}
	artifactDataPoint := findArtifactDataPointByPath(artifact.DataPoints, "metrics.main.temp")
	if artifactDataPoint == nil {
		t.Fatal("expected datapoint metrics.main.temp in artifact")
	}
	assertRuntimeGrant(t, artifactDataPoint.RuntimePermissions.Write, []string{"operator"}, []string{"guest"}, false)
	if !artifactHasMqttConnection(artifact.Mqtt.Connections, mqttConnection.ID) {
		t.Fatalf("expected mqtt.connections include %q", mqttConnection.ID)
	}
	if !artifactHasMqttID(artifact.Mqtt.Subscriptions, subscriptionID) {
		t.Fatalf("expected mqtt.subscriptions include %q", subscriptionID)
	}
	if !artifactHasMqttID(artifact.Mqtt.Tags, tagID) {
		t.Fatalf("expected mqtt.tags include %q", tagID)
	}
	if !artifactHasProtocol(artifact.Protocols.Kafka, kafkaConnection.ID) {
		t.Fatalf("expected protocols.kafka include %q", kafkaConnection.ID)
	}
	if len(artifact.Compute) != 1 || artifact.Compute[0].ID != computeUnit.ID || artifact.Compute[0].TriggerType != "schedule" || artifact.Compute[0].TriggerConfig["kind"] != "weekly" {
		t.Fatalf("expected weekly compute trigger in artifact, got %#v", artifact.Compute)
	}
	if len(artifact.BuiltinStores.Relations) != 1 || artifact.BuiltinStores.Relations[0].RuntimeKey == "" {
		t.Fatalf("expected relation builtin store contract, got %#v", artifact.BuiltinStores.Relations)
	}
	if artifact.BuiltinStores.Relations[0].Schema != artifact.BuiltinStores.Relations[0].RuntimeKey {
		t.Fatalf("expected relation runtime schema to use runtimeKey, got %#v", artifact.BuiltinStores.Relations[0])
	}
	if len(artifact.BuiltinStores.Timeseries) != 1 || artifact.BuiltinStores.Timeseries[0].RuntimeKey == "" {
		t.Fatalf("expected timeseries builtin store contract, got %#v", artifact.BuiltinStores.Timeseries)
	}
	if len(artifact.BuiltinStores.RealtimeSpaces) != 1 || artifact.BuiltinStores.RealtimeSpaces[0].Namespace != artifact.BuiltinStores.RealtimeSpaces[0].RuntimeKey {
		t.Fatalf("expected realtime builtin store contract, got %#v", artifact.BuiltinStores.RealtimeSpaces)
	}
	rawArtifact, err := json.Marshal(artifact)
	if err != nil {
		t.Fatalf("marshal artifact failed: %v", err)
	}
	for _, forbidden := range []string{"if_dev_data", "18379", "18883", "password", "broker"} {
		if strings.Contains(strings.ToLower(string(rawArtifact)), forbidden) {
			t.Fatalf("artifact leaked internal value %q: %s", forbidden, string(rawArtifact))
		}
	}
}

type projectArtifactPayload struct {
	Version       string                         `json:"version"`
	ProjectID     string                         `json:"projectId"`
	GeneratedAt   time.Time                      `json:"generatedAt"`
	Connections   []projectArtifactConnection    `json:"connections"`
	Queries       []projectArtifactQuery         `json:"queries"`
	DataPoints    []projectArtifactDataPoint     `json:"datapoints"`
	Mqtt          projectArtifactMqttPayload     `json:"mqtt"`
	Protocols     projectArtifactProtocolPayload `json:"protocols"`
	BuiltinStores projectArtifactBuiltinStores   `json:"builtinStores"`
	Compute       []projectArtifactCompute       `json:"compute"`
}

type projectArtifactCompute struct {
	ID            string         `json:"id"`
	TriggerType   string         `json:"triggerType"`
	TriggerConfig map[string]any `json:"triggerConfig"`
}

type projectArtifactConnection struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

type projectArtifactQuery struct {
	ID string `json:"id"`
}

type projectArtifactDataPoint struct {
	ID                 string                   `json:"id"`
	Path               string                   `json:"path"`
	RuntimePermissions dataPointPermissionGroup `json:"runtimePermissions"`
}

type projectArtifactMqttPayload struct {
	Connections   []projectArtifactMqttConnection `json:"connections"`
	Subscriptions []projectArtifactNamedID        `json:"subscriptions"`
	Tags          []projectArtifactNamedID        `json:"tags"`
}

type projectArtifactMqttConnection struct {
	ID string `json:"id"`
}

type projectArtifactNamedID struct {
	ID string `json:"id"`
}

type projectArtifactProtocolPayload struct {
	Kafka     []projectArtifactNamedID `json:"kafka"`
	HTTP      []projectArtifactNamedID `json:"http"`
	Websocket []projectArtifactNamedID `json:"websocket"`
	Redis     []projectArtifactNamedID `json:"redis"`
}

type projectArtifactBuiltinStores struct {
	Relations      []projectArtifactBuiltinRelationStore   `json:"relations"`
	Timeseries     []projectArtifactBuiltinTimeseriesStore `json:"timeseries"`
	RealtimeSpaces []projectArtifactBuiltinRealtimeStore   `json:"realtimeSpaces"`
}

type projectArtifactBuiltinRelationStore struct {
	RuntimeKey string `json:"runtimeKey"`
	Schema     string `json:"schema"`
}

type projectArtifactBuiltinTimeseriesStore struct {
	RuntimeKey string `json:"runtimeKey"`
	Schema     string `json:"schema"`
}

type projectArtifactBuiltinRealtimeStore struct {
	RuntimeKey string `json:"runtimeKey"`
	Namespace  string `json:"namespace"`
}

func insertTestMqttTag(t *testing.T, ctx context.Context, fixture *testDatabase, projectID, subscriptionID, userID string) string {
	t.Helper()

	tagID := uuid.NewString()
	_, err := fixture.pool.Exec(ctx, `
		INSERT INTO data_mqtt_tags (
			id,
			project_id,
			subscription_id,
			name,
			code,
			data_type,
			parse_type,
			parse_rule,
			validation,
			display_order,
			created_by,
			updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, '{}'::jsonb, $9, $10, $10)
	`, tagID, projectID, subscriptionID, "tag-main", "tag_main", "float64", "jsonpath", "$.value", 1, userID)
	if err != nil {
		t.Fatalf("insert mqtt tag failed: %v", err)
	}
	return tagID
}

func artifactHasConnection(connections []projectArtifactConnection, id, connectionType string) bool {
	for _, connection := range connections {
		if connection.ID == id && connection.Type == connectionType {
			return true
		}
	}
	return false
}

func artifactHasQuery(queries []projectArtifactQuery, id string) bool {
	for _, query := range queries {
		if query.ID == id {
			return true
		}
	}
	return false
}

func artifactHasDataPointPath(dataPoints []projectArtifactDataPoint, path string) bool {
	for _, dataPoint := range dataPoints {
		if dataPoint.Path == path {
			return true
		}
	}
	return false
}

func findArtifactDataPointByPath(dataPoints []projectArtifactDataPoint, path string) *projectArtifactDataPoint {
	for index := range dataPoints {
		if dataPoints[index].Path == path {
			return &dataPoints[index]
		}
	}
	return nil
}

func artifactHasMqttConnection(connections []projectArtifactMqttConnection, id string) bool {
	for _, connection := range connections {
		if connection.ID == id {
			return true
		}
	}
	return false
}

func artifactHasMqttID(items []projectArtifactNamedID, id string) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}

func artifactHasProtocol(protocols []projectArtifactNamedID, id string) bool {
	for _, protocol := range protocols {
		if protocol.ID == id {
			return true
		}
	}
	return false
}
