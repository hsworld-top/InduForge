package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/indu-forge/data_service/internal/app"
	"github.com/indu-forge/data_service/internal/auth"
	"github.com/indu-forge/data_service/internal/config"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

func TestProjectSnapshotGetAndReplaceRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupSchemaInitializer(t, fixture.pool)
	if err := migrator.Ensure(ctx); err != nil {
		t.Fatalf("schema initialization failed: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "snapshot-roundtrip-secret-01"
	if _, err := fixture.pool.Exec(ctx, `INSERT INTO data_project_tenant_bindings(project_id,tenant_id) VALUES($1,'tenant-snapshot')`, projectID); err != nil {
		t.Fatal(err)
	}

	srv, err := app.NewServer(config.Config{DataServiceInternalToken: "integration-test-internal-token",
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
		TenantID:     "tenant-snapshot",
		ProjectIDs:   []string{projectID},
		Capabilities: []string{"project:read", "project:write"},
	})

	relational := mustCreateConnection(t, server.URL, token, projectID, map[string]any{
		"name":    "pg-main",
		"type":    "relational",
		"enabled": true,
		"config": map[string]any{
			"dbType":   "postgresql",
			"host":     "127.0.0.1",
			"port":     5432,
			"database": "factory",
			"username": "postgres",
			"password": "postgres",
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
	insertOneDataPoint(t, ctx, fixture, projectID, userID, "metrics.main.temp", "active")

	mqttConnection := mustCreateMqttConnection(t, server.URL, token, projectID, map[string]any{
		"name":      "mqtt-main",
		"brokerUrl": "tcp://localhost:1883",
		"protocol":  "mqtt",
		"port":      1883,
	})
	subscriptionID := insertTestMqttSubscription(t, ctx, fixture, projectID, mqttConnection.ID, userID)
	tagID := insertTestMqttTag(t, ctx, fixture, projectID, subscriptionID, userID)

	computeFolderID, computeDependencyID := uuid.NewString(), uuid.NewString()
	workbenchGroupID, mqttGroupID := uuid.NewString(), uuid.NewString()
	if _, err := fixture.pool.Exec(ctx, `INSERT INTO data_compute_folders(id,project_id,name,created_by,updated_by) VALUES($1,$2,'计算目录',$3,$3);
		INSERT INTO data_compute_dependencies(id,project_id,language,package_name,import_name,version,created_by) VALUES($4,$2,'js','lodash','lodash','4.17.21',$3);
		INSERT INTO data_workbench_object_groups(id,project_id,connection_id,scope,name,sort_order,created_by,updated_by) VALUES($5,$2,$6,'query','查询目录',3,$3,$3);
		UPDATE data_queries SET group_id=$5 WHERE id=$7;
		INSERT INTO data_table_group_members(project_id,connection_id,table_name,group_id,updated_by) VALUES($2,$6,'public.metrics',$5,$3);
		INSERT INTO data_mqtt_subscription_groups(id,project_id,connection_id,name,created_by,updated_by) VALUES($8,$2,$9,'订阅目录',$3,$3);
		UPDATE data_mqtt_subscriptions SET group_id=$8,display_order=7 WHERE id=$10`, computeFolderID, projectID, userID, computeDependencyID, workbenchGroupID, relational.ID, query.ID, mqttGroupID, mqttConnection.ID, subscriptionID); err != nil {
		t.Fatalf("seed complete authoring snapshot domains: %v", err)
	}
	kafka := mustCreateKafkaConfig(t, server.URL, token, projectID, map[string]any{"name": "kafka-main", "brokers": "127.0.0.1:9092", "topic": "factory.events", "consumerGroup": "snapshot-group", "startPosition": "earliest"})
	httpConnection := mustCreateHTTPConfig(t, server.URL, token, projectID, map[string]any{"name": "http-main", "description": "workbench-owned"})
	websocketConnection := mustCreateWebSocketConfig(t, server.URL, token, projectID, map[string]any{"name": "ws-main", "description": "workbench-owned"})
	redis := mustCreateRedisConfig(t, server.URL, token, projectID, map[string]any{"name": "redis-main", "address": "127.0.0.1:6380", "keyPattern": "factory:*", "mode": "standalone"})
	tdEnvelope := doJSONRequest(t, http.MethodPost, server.URL+"/api/v1/data/projects/"+projectID+"/tdengine/configs", token, map[string]any{"name": "td-main", "protocol": "wss", "host": "td.example.local", "port": 6041, "username": "root", "databaseName": "factory", "timezone": "Asia/Shanghai", "tlsSkipVerify": true, "secrets": map[string]string{"password": "snapshot-secret"}})
	var td protocolConnectionConnectionPayload
	if err := json.Unmarshal(tdEnvelope.Data, &td); err != nil || td.ID == "" {
		t.Fatalf("create tdengine for snapshot: %v", err)
	}
	kafkaTopicGroupID, kafkaMappingID, kafkaFieldGroupID, kafkaFieldID := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	if _, err := fixture.pool.Exec(ctx, `INSERT INTO data_kafka_topic_groups(id,project_id,connection_id,name,sort_order,created_by,updated_by) VALUES($1,$2,$3,'Topic目录',2,$4,$4);
		INSERT INTO data_kafka_topic_mappings(id,project_id,connection_id,group_id,name,topic,description,consumer_group,output_mode,raw_output_scope,partition_mode,start_position,decode,sample_limit,timeout_ms,sort_order,created_by,updated_by) VALUES($5,$2,$3,$1,'events','factory.events','desc','cg','field_mapping','value','all','earliest','json',50,4000,4,$4,$4);
		INSERT INTO data_kafka_field_groups(id,project_id,connection_id,topic_mapping_id,name,description,sort_order,created_by,updated_by) VALUES($6,$2,$3,$5,'字段目录','desc',1,$4,$4);
		INSERT INTO data_kafka_fields(id,project_id,connection_id,topic_mapping_id,group_id,name,value_path,key_path,data_type,enabled,description,sort_order,created_by,updated_by) VALUES($7,$2,$3,$5,$6,'temperature','["payload","temp"]','[]','float64',true,'温度',5,$4,$4)`, kafkaTopicGroupID, projectID, kafka.ID, userID, kafkaMappingID, kafkaFieldGroupID, kafkaFieldID); err != nil {
		t.Fatalf("seed kafka snapshot domains: %v", err)
	}

	currentSnapshot := mustGetProjectSnapshot(t, server.URL, token, projectID)
	if len(currentSnapshot.Connections) != 7 {
		t.Fatalf("expected 7 connections in snapshot, got %d", len(currentSnapshot.Connections))
	}
	if len(currentSnapshot.RelationalConfigs) != 1 {
		t.Fatalf("expected 1 relational config in snapshot, got %d", len(currentSnapshot.RelationalConfigs))
	}
	if currentSnapshot.RelationalConfigs[0].ConnectionID != relational.ID {
		t.Fatalf("expected relational config to point to %q, got %q", relational.ID, currentSnapshot.RelationalConfigs[0].ConnectionID)
	}
	if len(currentSnapshot.Queries) != 1 || currentSnapshot.Queries[0].ID != query.ID {
		t.Fatalf("expected snapshot queries include %q", query.ID)
	}
	if !snapshotHasDataPointPath(currentSnapshot.DataPoints, "metrics.main.temp") {
		t.Fatal("expected snapshot datapoints include metrics.main.temp")
	}
	if len(currentSnapshot.MqttConfigs) != 1 || currentSnapshot.MqttConfigs[0].ConnectionID != mqttConnection.ID {
		t.Fatalf("expected mqtt config to point to %q", mqttConnection.ID)
	}
	if len(currentSnapshot.MqttSubscriptions) != 1 || currentSnapshot.MqttSubscriptions[0].ID != subscriptionID {
		t.Fatalf("expected mqtt subscription %q in snapshot", subscriptionID)
	}
	if len(currentSnapshot.MqttTags) != 1 || currentSnapshot.MqttTags[0].ID != tagID {
		t.Fatalf("expected mqtt tag %q in snapshot", tagID)
	}
	if len(currentSnapshot.ComputeFolders) != 1 || currentSnapshot.ComputeFolders[0].ID != computeFolderID || len(currentSnapshot.ComputeDependencies) != 1 || currentSnapshot.ComputeDependencies[0].ID != computeDependencyID {
		t.Fatal("compute folder/dependency missing from snapshot")
	}
	if len(currentSnapshot.WorkbenchObjectGroups) != 1 || currentSnapshot.Queries[0].GroupID == nil || *currentSnapshot.Queries[0].GroupID != workbenchGroupID || len(currentSnapshot.TableGroupMembers) != 1 {
		t.Fatal("workbench grouping missing from snapshot")
	}
	if len(currentSnapshot.MqttSubscriptionGroups) != 1 || currentSnapshot.MqttSubscriptions[0].GroupID == nil || *currentSnapshot.MqttSubscriptions[0].GroupID != mqttGroupID || currentSnapshot.MqttSubscriptions[0].DisplayOrder != 7 {
		t.Fatal("mqtt grouping/order missing from snapshot")
	}
	if len(currentSnapshot.KafkaConfigs) != 1 || currentSnapshot.KafkaConfigs[0].ConsumerGroup == nil || *currentSnapshot.KafkaConfigs[0].ConsumerGroup != "snapshot-group" || len(currentSnapshot.KafkaTopicGroups) != 1 || len(currentSnapshot.KafkaTopicMappings) != 1 || len(currentSnapshot.KafkaFieldGroups) != 1 || len(currentSnapshot.KafkaFields) != 1 {
		t.Fatal("kafka authoring configuration missing from snapshot")
	}
	if !snapshotHasConnection(currentSnapshot.Connections, redis.ID, "redis-main") || !snapshotHasConnection(currentSnapshot.Connections, td.ID, "td-main") {
		t.Fatal("redis/tdengine structured configuration missing from snapshot")
	}

	// 对同一完整快照执行覆盖后再次读取，验证所有目录、映射和专用配置表可无损往返。
	doJSONRequest(t, http.MethodPut, server.URL+"/api/v1/data/projects/"+projectID+"/snapshot", token, currentSnapshot)
	roundTripped := mustGetProjectSnapshot(t, server.URL, token, projectID)
	if len(roundTripped.ComputeFolders) != 1 || len(roundTripped.KafkaFields) != 1 || len(roundTripped.MqttSubscriptionGroups) != 1 || len(roundTripped.TableGroupMembers) != 1 || roundTripped.Queries[0].GroupID == nil || !snapshotHasConnection(roundTripped.Connections, redis.ID, "redis-main") || !snapshotHasConnection(roundTripped.Connections, td.ID, "td-main") {
		t.Fatal("complete authoring snapshot round trip lost configuration")
	}
	if !reflect.DeepEqual(currentSnapshot.RelationalConfigs, roundTripped.RelationalConfigs) ||
		!reflect.DeepEqual(currentSnapshot.KafkaConfigs, roundTripped.KafkaConfigs) ||
		!reflect.DeepEqual(currentSnapshot.KafkaTopicMappings, roundTripped.KafkaTopicMappings) ||
		!reflect.DeepEqual(currentSnapshot.KafkaFields, roundTripped.KafkaFields) {
		t.Fatalf("specialized relational/kafka configuration did not round trip losslessly:\nbefore relation=%#v\nafter relation=%#v\nbefore kafka=%#v\nafter kafka=%#v\nbefore mappings=%#v\nafter mappings=%#v\nbefore fields=%#v\nafter fields=%#v", currentSnapshot.RelationalConfigs, roundTripped.RelationalConfigs, currentSnapshot.KafkaConfigs, roundTripped.KafkaConfigs, currentSnapshot.KafkaTopicMappings, roundTripped.KafkaTopicMappings, currentSnapshot.KafkaFields, roundTripped.KafkaFields)
	}
	for _, connectionID := range []string{httpConnection.ID, websocketConnection.ID, redis.ID, td.ID} {
		before, after := snapshotConnectionConfig(currentSnapshot.Connections, connectionID), snapshotConnectionConfig(roundTripped.Connections, connectionID)
		if !reflect.DeepEqual(before, after) {
			t.Fatalf("connection %s configuration changed during round trip: before=%#v after=%#v", connectionID, before, after)
		}
	}

	replacementRelationalID := uuid.NewString()
	replacementMqttID := uuid.NewString()
	replacementQueryID := uuid.NewString()
	replacementSubscriptionID := uuid.NewString()
	replacementTagID := uuid.NewString()
	replacementSourceID := replacementQueryID
	refreshInterval := 5000

	replacement := repository.ProjectSnapshot{
		Connections: []repository.ConnectionRecord{
			{
				ID:        replacementRelationalID,
				ProjectID: projectID,
				Name:      "pg-replaced",
				Type:      "relational",
				IsEnabled: true,
				Config: map[string]any{
					"dbType":   "postgresql",
					"host":     "10.0.0.9",
					"port":     5432,
					"database": "factory_v2",
					"username": "svc_user",
				},
			},
			{
				ID:        replacementMqttID,
				ProjectID: projectID,
				Name:      "mqtt-replaced",
				Type:      "mqtt",
				IsEnabled: true,
				Config: map[string]any{
					"brokerUrl": "tcp://127.0.0.1:1884",
				},
			},
		},
		Queries: []repository.QueryRecord{
			{
				ID:              replacementQueryID,
				ProjectID:       projectID,
				ConnectionID:    replacementRelationalID,
				Name:            "query-replaced",
				QueryType:       "sql",
				Config:          map[string]any{"sql": "SELECT 2 AS value"},
				IsEnabled:       true,
				TimeoutMS:       15000,
				CacheEnabled:    true,
				CacheTtlSeconds: 30,
			},
		},
		MqttConfigs: []repository.SnapshotMqttConfigRecord{
			{
				ConnectionID:      replacementMqttID,
				BrokerURL:         "tcp://127.0.0.1:1884",
				Protocol:          "mqtt",
				Port:              1884,
				Keepalive:         60,
				CleanSession:      true,
				QOS:               1,
				ReconnectPeriodMS: 2000,
				ConnectTimeoutMS:  30000,
				Will:              map[string]any{},
				SSLConfig:         map[string]any{},
			},
		},
		MqttSubscriptions: []repository.SnapshotMqttSubscriptionRecord{
			{
				ID:               replacementSubscriptionID,
				ProjectID:        projectID,
				ConnectionID:     replacementMqttID,
				Name:             "sub-replaced",
				Topic:            "factory/line1/temp",
				QOS:              1,
				MessageRetention: 50,
			},
		},
		MqttTags: []repository.SnapshotMqttTagRecord{
			{
				ID:             replacementTagID,
				ProjectID:      projectID,
				SubscriptionID: replacementSubscriptionID,
				Name:           "tag-replaced",
				Code:           "tag_replaced",
				DataType:       "float64",
				ParseType:      "jsonpath",
				ParseRule:      "$.value",
				Validation:     map[string]any{},
				Order:          1,
			},
		},
		DataPoints: []repository.DataPointRecord{
			{
				ID:                 uuid.NewString(),
				ProjectID:          projectID,
				Path:               "metrics.replaced.temp",
				Name:               "metrics.replaced.temp",
				SourceType:         "query",
				SourceID:           &replacementSourceID,
				SourceConfig:       map[string]any{"column": "value"},
				DataType:           "float64",
				Tags:               []any{"temperature"},
				RuntimePermissions: repository.DefaultDataPointRuntimePermissions(),
				RefreshMode:        "auto",
				RefreshIntervalMS:  &refreshInterval,
				Status:             "active",
			},
		},
	}

	doJSONRequest(t, http.MethodPut, server.URL+"/api/v1/data/projects/"+projectID+"/snapshot", token, replacement)

	replacedSnapshot := mustGetProjectSnapshot(t, server.URL, token, projectID)
	if len(replacedSnapshot.Connections) != 2 {
		t.Fatalf("expected 2 connections after replace, got %d", len(replacedSnapshot.Connections))
	}
	if !snapshotHasConnection(replacedSnapshot.Connections, replacementRelationalID, "pg-replaced") {
		t.Fatalf("expected replaced relational connection %q in snapshot", replacementRelationalID)
	}
	if !snapshotHasConnection(replacedSnapshot.Connections, replacementMqttID, "mqtt-replaced") {
		t.Fatalf("expected replaced mqtt connection %q in snapshot", replacementMqttID)
	}
	if snapshotHasConnection(replacedSnapshot.Connections, relational.ID, relational.Name) {
		t.Fatalf("did not expect old connection %q after replace", relational.ID)
	}
	if len(replacedSnapshot.Queries) != 1 || replacedSnapshot.Queries[0].ID != replacementQueryID {
		t.Fatalf("expected replaced query %q in snapshot", replacementQueryID)
	}
	if len(replacedSnapshot.MqttConfigs) != 1 || replacedSnapshot.MqttConfigs[0].ConnectionID != replacementMqttID {
		t.Fatalf("expected replaced mqtt config to point to %q", replacementMqttID)
	}
	if len(replacedSnapshot.MqttSubscriptions) != 1 || replacedSnapshot.MqttSubscriptions[0].ID != replacementSubscriptionID {
		t.Fatalf("expected replaced mqtt subscription %q in snapshot", replacementSubscriptionID)
	}
	if len(replacedSnapshot.MqttTags) != 1 || replacedSnapshot.MqttTags[0].ID != replacementTagID {
		t.Fatalf("expected replaced mqtt tag %q in snapshot", replacementTagID)
	}
	if len(replacedSnapshot.DataPoints) != 1 || replacedSnapshot.DataPoints[0].Path != "metrics.replaced.temp" {
		t.Fatal("expected replaced datapoint metrics.replaced.temp in snapshot")
	}
}

func snapshotConnectionConfig(connections []repository.ConnectionRecord, connectionID string) map[string]any {
	for _, connection := range connections {
		if connection.ID == connectionID {
			return connection.Config
		}
	}
	return nil
}

func TestProjectSnapshotReplaceRejectsIncompleteConnections(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupSchemaInitializer(t, fixture.pool)
	if err := migrator.Ensure(ctx); err != nil {
		t.Fatalf("schema initialization failed: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "snapshot-roundtrip-secret-02"
	if _, err := fixture.pool.Exec(ctx, `INSERT INTO data_project_tenant_bindings(project_id,tenant_id) VALUES($1,'tenant-snapshot')`, projectID); err != nil {
		t.Fatal(err)
	}

	srv, err := app.NewServer(config.Config{DataServiceInternalToken: "integration-test-internal-token",
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
		TenantID:     "tenant-snapshot",
		ProjectIDs:   []string{projectID},
		Capabilities: []string{"project:read", "project:write"},
	})

	original := mustCreateConnection(t, server.URL, token, projectID, map[string]any{
		"name":    "pg-stable",
		"type":    "relational",
		"enabled": true,
		"config": map[string]any{
			"dbType": "postgresql",
			"host":   "127.0.0.1",
			"port":   5432,
		},
	})

	invalidEnvelope := doJSONRequestWithStatus(t, http.MethodPut, server.URL+"/api/v1/data/projects/"+projectID+"/snapshot", token, repository.ProjectSnapshot{
		Connections: []repository.ConnectionRecord{
			{
				ID:        uuid.NewString(),
				ProjectID: projectID,
				Type:      "relational",
				IsEnabled: true,
				Config:    map[string]any{},
			},
		},
	}, http.StatusOK)
	if invalidEnvelope.Code == 0 {
		t.Fatal("expected replace snapshot response to fail for incomplete connection")
	}

	currentSnapshot := mustGetProjectSnapshot(t, server.URL, token, projectID)
	if len(currentSnapshot.Connections) != 1 {
		t.Fatalf("expected original snapshot to keep 1 connection, got %d", len(currentSnapshot.Connections))
	}
	if currentSnapshot.Connections[0].ID != original.ID {
		t.Fatalf("expected original connection %q to remain after rejected replace, got %q", original.ID, currentSnapshot.Connections[0].ID)
	}
}

func TestProjectSnapshotReplaceRejectsUnsupportedConnectionType(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupSchemaInitializer(t, fixture.pool)
	if err := migrator.Ensure(ctx); err != nil {
		t.Fatalf("schema initialization failed: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "snapshot-roundtrip-secret-03"
	if _, err := fixture.pool.Exec(ctx, `INSERT INTO data_project_tenant_bindings(project_id,tenant_id) VALUES($1,'tenant-snapshot')`, projectID); err != nil {
		t.Fatal(err)
	}

	srv, err := app.NewServer(config.Config{DataServiceInternalToken: "integration-test-internal-token",
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
		TenantID:     "tenant-snapshot",
		ProjectIDs:   []string{projectID},
		Capabilities: []string{"project:read", "project:write"},
	})

	original := mustCreateConnection(t, server.URL, token, projectID, map[string]any{
		"name":    "pg-stable",
		"type":    "relational",
		"enabled": true,
		"config": map[string]any{
			"dbType": "postgresql",
			"host":   "127.0.0.1",
			"port":   5432,
		},
	})

	rejected := doJSONRequestWithStatus(t, http.MethodPut, server.URL+"/api/v1/data/projects/"+projectID+"/snapshot", token, repository.ProjectSnapshot{
		Connections: []repository.ConnectionRecord{
			{
				ID:        uuid.NewString(),
				ProjectID: projectID,
				Name:      "opcua-rejected",
				Type:      "opcua",
				IsEnabled: true,
				Config: map[string]any{
					"endpoint": "opc.tcp://127.0.0.1:4840",
				},
			},
		},
	}, http.StatusOK)
	if rejected.Code == apperrors.SuccessCode || rejected.Code != apperrors.PublicCodeBadRequest {
		t.Fatalf("expected unsupported OPC UA connection to be rejected, got %#v", rejected)
	}

	currentSnapshot := mustGetProjectSnapshot(t, server.URL, token, projectID)
	if len(currentSnapshot.Connections) != 1 {
		t.Fatalf("expected original snapshot to keep 1 connection, got %d", len(currentSnapshot.Connections))
	}
	if currentSnapshot.Connections[0].ID != original.ID {
		t.Fatalf("expected original connection %q to remain after rejected replace, got %q", original.ID, currentSnapshot.Connections[0].ID)
	}
}

func TestProjectSnapshotReplacePreservesDataPointRuntimePermissions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupSchemaInitializer(t, fixture.pool)
	if err := migrator.Ensure(ctx); err != nil {
		t.Fatalf("schema initialization failed: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "snapshot-roundtrip-secret-04"
	if _, err := fixture.pool.Exec(ctx, `INSERT INTO data_project_tenant_bindings(project_id,tenant_id) VALUES($1,'tenant-snapshot')`, projectID); err != nil {
		t.Fatal(err)
	}

	srv, err := app.NewServer(config.Config{DataServiceInternalToken: "integration-test-internal-token",
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
		TenantID:     "tenant-snapshot",
		ProjectIDs:   []string{projectID},
		Capabilities: []string{"project:read", "project:write"},
	})

	doJSONRequest(t, http.MethodPut, server.URL+"/api/v1/data/projects/"+projectID+"/snapshot", token, map[string]any{
		"datapoints": []map[string]any{
			{
				"id":           uuid.NewString(),
				"projectId":    projectID,
				"path":         "metrics.snapshot.permission",
				"name":         "metrics.snapshot.permission",
				"sourceType":   "manual",
				"sourceConfig": map[string]any{},
				"dataType":     "string",
				"tags":         []any{"snapshot"},
				"refreshMode":  "auto",
				"status":       "active",
				"runtimePermissions": map[string]any{
					"write": map[string]any{
						"allowRoles": []string{"ops"},
						"denyRoles":  []string{"guest"},
						"inherit":    false,
					},
				},
			},
		},
	})

	responseEnvelope := doJSONRequest(t, http.MethodGet, server.URL+"/api/v1/data/projects/"+projectID+"/snapshot", token, nil)
	var snapshot struct {
		DataPoints []struct {
			Path               string                   `json:"path"`
			RuntimePermissions dataPointPermissionGroup `json:"runtimePermissions"`
		} `json:"datapoints"`
	}
	if err := json.Unmarshal(responseEnvelope.Data, &snapshot); err != nil {
		t.Fatalf("decode snapshot payload failed: %v", err)
	}

	if len(snapshot.DataPoints) != 1 || snapshot.DataPoints[0].Path != "metrics.snapshot.permission" {
		t.Fatal("expected snapshot datapoints include metrics.snapshot.permission")
	}
	assertRuntimeGrant(t, snapshot.DataPoints[0].RuntimePermissions.Write, []string{"ops"}, []string{"guest"}, false)
}

func mustGetProjectSnapshot(t *testing.T, baseURL, token, projectID string) repository.ProjectSnapshot {
	t.Helper()

	responseEnvelope := doJSONRequest(t, http.MethodGet, baseURL+"/api/v1/data/projects/"+projectID+"/snapshot", token, nil)

	var snapshot repository.ProjectSnapshot
	if err := json.Unmarshal(responseEnvelope.Data, &snapshot); err != nil {
		t.Fatalf("decode snapshot response failed: %v", err)
	}
	return snapshot
}

func snapshotHasConnection(connections []repository.ConnectionRecord, id, name string) bool {
	for _, connection := range connections {
		if connection.ID == id && connection.Name == name {
			return true
		}
	}
	return false
}

func snapshotHasDataPointPath(datapoints []repository.DataPointRecord, path string) bool {
	for _, datapoint := range datapoints {
		if datapoint.Path == path {
			return true
		}
	}
	return false
}
