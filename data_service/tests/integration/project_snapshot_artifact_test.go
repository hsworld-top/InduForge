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
	"github.com/indu-forge/data_service/internal/repository"
	"github.com/indu-forge/data_service/internal/service"
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

	_ = mustCreateQuery(t, server.URL, token, projectID, map[string]any{
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
		SET data_type='float64',default_value='12',runtime_permissions = $3::jsonb
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
	_ = insertTestMqttTag(t, ctx, fixture, projectID, subscriptionID, userID)

	_ = mustCreateKafkaConfig(t, server.URL, token, projectID, map[string]any{
		"name":          "kafka-main",
		"brokers":       "127.0.0.1:9092",
		"topic":         "factory.events",
		"consumerGroup": "artifact-group",
	})
	computeUnit := mustCreateComputeUnit(t, server.URL, token, projectID, map[string]any{
		"name": "weekly-report", "language": "js", "scriptCode": "return null;",
		"triggerType":   "schedule",
		"triggerConfig": map[string]any{"kind": "weekly", "weekdays": []int{1, 3, 5}, "time": "08:30:00", "timezone": "Asia/Shanghai"},
		"outputs":       []map[string]any{{"key": "result", "name": "result", "path": "calc.weekly_report.result", "dataType": "float64", "nullPolicy": "default", "defaultValue": "7"}},
	})
	alarmID := uuid.NewString()
	if _, err := fixture.pool.Exec(ctx, `INSERT INTO data_alarm_items(id,project_id,datapoint_id,display_name,name_key,mode,alarm_type,evaluation_mode,trigger_fingerprint,created_by,updated_by) VALUES($1,$2,$3,'温度过高','temperature-high','point','threshold','single',$4,$5,$5)`, alarmID, projectID, dataPointID, strings.Repeat("a", 64), userID); err != nil {
		t.Fatalf("insert artifact alarm: %v", err)
	}
	if _, err := fixture.pool.Exec(ctx, `INSERT INTO data_alarm_item_inputs(alarm_item_id,datapoint_id,input_key,sort_order) VALUES($1,$2,'value',0)`, alarmID, dataPointID); err != nil {
		t.Fatalf("insert artifact alarm input: %v", err)
	}
	if _, err := fixture.pool.Exec(ctx, `INSERT INTO data_alarm_item_conditions(alarm_item_id,kind,operator,severity,params,sort_order) VALUES($1,'threshold','gt','warning','{"threshold":1}'::jsonb,0)`, alarmID); err != nil {
		t.Fatalf("insert artifact alarm condition: %v", err)
	}
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
	var strict map[string]any
	if err := json.Unmarshal(responseEnvelope.Data, &strict); err != nil {
		t.Fatalf("decode strict runtime artifact: %v", err)
	}
	if strict["schemaVersion"] != "runtime-project-artifact.v1" || strict["projectArtifactVersion"] != "1.0" || strict["projectId"] != projectID {
		t.Fatalf("unexpected strict artifact identity: %#v", strict)
	}
	compute, ok := strict["computeUnits"].([]any)
	if !ok || len(compute) != 1 {
		t.Fatalf("expected one runtime compute unit: %#v", strict["computeUnits"])
	}
	unit := compute[0].(map[string]any)
	if unit["id"] != computeUnit.ID || unit["revision"].(float64) < 1 || unit["trigger"].(map[string]any)["kind"] != "schedule" {
		t.Fatalf("invalid runtime compute projection: %#v", unit)
	}
	outputs := unit["outputs"].([]any)
	if len(outputs) != 1 || outputs[0].(map[string]any)["nullPolicy"] != "default" || outputs[0].(map[string]any)["defaultValue"].(float64) != 7 {
		t.Fatalf("runtime compute output missing: %#v", outputs)
	}
	points := strict["dataPoints"].([]any)
	if len(points) < 2 {
		t.Fatalf("expected manual and compute output datapoints: %#v", points)
	}
	for _, raw := range points {
		source := raw.(map[string]any)["sourceConfig"].(map[string]any)
		encoded, _ := json.Marshal(source)
		if strings.Contains(strings.ToLower(string(encoded)), "outputkey") || strings.Contains(strings.ToLower(string(encoded)), "secret") {
			t.Fatalf("unsafe runtime source config: %s", encoded)
		}
	}
	alarms := strict["alarmItems"].([]any)
	if len(alarms) != 1 || alarms[0].(map[string]any)["id"] != alarmID || alarms[0].(map[string]any)["revision"].(float64) < 1 {
		t.Fatalf("runtime alarm projection missing: %#v", alarms)
	}

	// 快照替换是绕过 ComputeService 的另一条可写路径；它也必须拒绝
	// RuntimeEngine 无法装载的计算输出和保留字别名，且预检失败不能改变
	// 已发布的 Revision 或 Artifact。
	baseline := mustGetProjectSnapshot(t, server.URL, token, projectID)
	outputDatapointID := ""
	baselineRevision := int64(0)
	for _, unit := range baseline.ComputeUnits {
		if unit.ID == computeUnit.ID && len(unit.Outputs) == 1 {
			outputDatapointID = unit.Outputs[0].DatapointID
			baselineRevision = unit.Revision
		}
	}
	if outputDatapointID == "" || baselineRevision < 1 {
		t.Fatal("expected compute output datapoint in snapshot")
	}
	snapshotService := service.NewProjectSnapshotService(repository.NewProjectSnapshotRepository(fixture.pool))
	mutateOutput := func(snapshot *repository.ProjectSnapshot, mutate func(*repository.DataPointRecord)) {
		t.Helper()
		for index := range snapshot.DataPoints {
			if snapshot.DataPoints[index].ID == outputDatapointID {
				mutate(&snapshot.DataPoints[index])
				return
			}
		}
		t.Fatalf("compute output %s is not represented by snapshot datapoints", outputDatapointID)
	}
	assertRejectedSnapshot := func(name string, mutate func(*repository.ProjectSnapshot)) {
		t.Helper()
		candidate := mustGetProjectSnapshot(t, server.URL, token, projectID)
		mutate(&candidate)
		if err := snapshotService.Replace(ctx, projectID, userID, candidate); err == nil {
			t.Fatalf("%s snapshot must be rejected", name)
		}
		current := mustGetProjectSnapshot(t, server.URL, token, projectID)
		for _, unit := range current.ComputeUnits {
			if unit.ID == computeUnit.ID && unit.Revision != baselineRevision {
				t.Fatalf("%s must not change compute revision: got %d want %d", name, unit.Revision, baselineRevision)
			}
		}
		published := doJSONRequest(t, http.MethodGet, server.URL+"/api/v1/data/projects/"+projectID+"/artifact", token, nil)
		if len(published.Data) == 0 {
			t.Fatalf("%s must leave an artifact publishable", name)
		}
	}
	assertRejectedSnapshot("inactive output", func(snapshot *repository.ProjectSnapshot) {
		mutateOutput(snapshot, func(point *repository.DataPointRecord) { point.Status = "inactive" })
	})
	assertRejectedSnapshot("manual output source", func(snapshot *repository.ProjectSnapshot) {
		mutateOutput(snapshot, func(point *repository.DataPointRecord) {
			point.SourceType, point.SourceID, point.SourceConfig = "manual", nil, map[string]any{}
		})
	})
	assertRejectedSnapshot("collector output source", func(snapshot *repository.ProjectSnapshot) {
		wrongID := uuid.NewString()
		mutateOutput(snapshot, func(point *repository.DataPointRecord) {
			point.SourceType, point.SourceID, point.SourceConfig = "collector.point", &wrongID, map[string]any{"connectionId": uuid.NewString()}
		})
	})
	assertRejectedSnapshot("wrong output sourceId", func(snapshot *repository.ProjectSnapshot) {
		wrongID := uuid.NewString()
		mutateOutput(snapshot, func(point *repository.DataPointRecord) { point.SourceID = &wrongID })
	})
	assertRejectedSnapshot("wrong output sourceConfig", func(snapshot *repository.ProjectSnapshot) {
		mutateOutput(snapshot, func(point *repository.DataPointRecord) {
			point.SourceConfig = map[string]any{"computeUnitId": computeUnit.ID, "outputKey": "wrong"}
		})
	})
	assertRejectedSnapshot("extra output sourceConfig", func(snapshot *repository.ProjectSnapshot) {
		mutateOutput(snapshot, func(point *repository.DataPointRecord) {
			point.SourceConfig = map[string]any{"computeUnitId": computeUnit.ID, "outputKey": "result", "extra": true}
		})
	})
	assertRejectedSnapshot("output path mismatch", func(snapshot *repository.ProjectSnapshot) {
		mutateOutput(snapshot, func(point *repository.DataPointRecord) { point.Path = "calc.weekly_report.other" })
	})
	assertRejectedSnapshot("output type mismatch", func(snapshot *repository.ProjectSnapshot) {
		mutateOutput(snapshot, func(point *repository.DataPointRecord) { point.DataType = "int32" })
	})
	assertRejectedSnapshot("reserved compute alias", func(snapshot *repository.ProjectSnapshot) {
		for index := range snapshot.ComputeUnits {
			if snapshot.ComputeUnits[index].ID == computeUnit.ID {
				snapshot.ComputeUnits[index].InputBindings = map[string]any{
					"datapointVariables": []any{map[string]any{"datapointId": dataPointID, "alias": "true"}},
				}
			}
		}
	})
	if err := repository.NewProjectSnapshotRepository(fixture.pool).ReplaceProjectData(ctx, projectID, userID, mustGetProjectSnapshot(t, server.URL, token, projectID)); err != nil {
		t.Fatalf("unchanged legal snapshot must remain replaceable: %v", err)
	}
	if published := doJSONRequest(t, http.MethodGet, server.URL+"/api/v1/data/projects/"+projectID+"/artifact", token, nil); len(published.Data) == 0 {
		t.Fatal("legal snapshot replace must remain artifact publishable")
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
