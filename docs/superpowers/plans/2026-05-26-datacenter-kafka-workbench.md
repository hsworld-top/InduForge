# 数据中心 Kafka 工作台实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 将 Kafka 接入源从通用协议预览升级为独立工作台，支持一个 Kafka 接入源管理多个 Topic 映射、短时抓样、字段映射和 `kafka.field` 数据点建模。

**架构：** `data_service` 新增 Kafka Topic 分组、Topic 映射和字段映射领域能力，服务层复用现有 kafka-go 短时 reader 并负责字段推断、预览摘要记录和数据点同步。`datacenter` 新增 Kafka 专属工作台，使用与 MQTT 工作台一致的左侧接入源与 Topic 树、中间页签、右侧诊断结构。

**技术栈：** Go、pgx、PostgreSQL migration、kafka-go、Vue 3、Element Plus、TypeScript、Zod、Vitest、pnpm。

---

## 文件结构

后端新增和修改：

- 创建：`data_service/internal/db/migrations/0030_kafka_workbench.sql`  
  新增 `data_kafka_topic_groups`、`data_kafka_topic_mappings`、`data_kafka_fields`。
- 创建：`data_service/internal/db/migrations/0030_kafka_workbench_down.sql`  
  回滚 Kafka 工作台表和索引。
- 创建：`data_service/internal/repository/kafka_workbench_repository.go`  
  参数化实现 Topic 分组、Topic 映射、字段映射、数据点同步和预览记录所需 SQL。
- 创建：`data_service/internal/service/kafka_workbench_service.go`  
  封装 Kafka 工作台校验、Topic 元数据读取、短时预览、schema 推断、字段映射和数据点同步。
- 修改：`data_service/internal/service/protocol_preview_adapters.go`  
  让 Kafka 短时 reader 支持单 partition、指定 offset、headers 和 decode 选项。
- 创建：`data_service/internal/http/handler/kafka_workbench_handler.go`  
  暴露 Topic 元数据、分组、映射、预览、字段映射接口。
- 修改：`data_service/internal/http/router/router.go`  
  挂载 `/api/v1/data/projects/{projectId}/kafka/{connectionId}` 与 `/kafka/topic-mappings/{mappingId}` 路由。
- 修改：`data_service/internal/app/server.go`  
  初始化 Kafka 工作台 repository、service、handler。
- 创建：`data_service/tests/integration/kafka_workbench_api_test.go`  
  覆盖 Kafka Topic 映射、字段映射、数据点同步、删除清理和预览参数契约。

前端新增和修改：

- 修改：`datacenter/src/api/data.api.ts`  
  增加 Kafka 工作台 API。
- 创建：`datacenter/src/api/schemas/kafka-workbench.schema.ts`  
  定义 Kafka 工作台响应 schema。
- 创建：`datacenter/src/components/kafka/types.ts`  
  Kafka 工作台共享类型。
- 创建：`datacenter/src/components/kafka/kafkaTopicTreeModel.ts`  
  构建和过滤 Topic 分组树。
- 创建：`datacenter/src/components/kafka/KafkaWorkbench.vue`  
  Kafka 工作台业务容器。
- 创建：`datacenter/src/components/kafka/KafkaTopicTreeBranch.vue`  
  左侧 Topic 分组树递归节点。
- 创建：`datacenter/src/components/kafka/KafkaTopicMappingDialog.vue`  
  创建和编辑 Topic 映射。
- 创建：`datacenter/src/components/kafka/KafkaTopicGroupDialog.vue`  
  创建和编辑 Topic 分组。
- 创建：`datacenter/src/components/kafka/KafkaPreviewPanel.vue`  
  短时抓样消息流面板。
- 创建：`datacenter/src/components/kafka/KafkaFieldMappingPanel.vue`  
  字段映射和数据点建模面板。
- 创建：`datacenter/src/components/kafka/KafkaInspectorPanel.vue`  
  右侧接入源、Topic、schema 和错误诊断。
- 创建：`datacenter/src/components/access-source/workbench/KafkaWorkbenchPanel.vue`  
  Kafka 工作台入口壳。
- 修改：`datacenter/src/components/access-source/AccessSourceWorkbench.vue`  
  `type === 'kafka'` 路由到 Kafka 专属工作台。
- 创建：`datacenter/tests/kafka-topic-tree-model.test.ts`  
  覆盖 Topic 分组树和搜索过滤。

验证命令：

- 后端：`cd data_service; go test ./...`
- 前端类型：`pnpm --filter datacenter typecheck`
- 前端构建：`pnpm --filter datacenter build`
- 前端单测：`pnpm --dir datacenter test -- kafka-topic-tree-model`

---

## 实现边界

- Kafka 工作台只做开发态配置和前期预览，不启动长期 consumer。
- Topic 映射是项目内配置，不创建、删除或修改真实 Kafka Topic。
- 短时预览可以读取真实 Kafka 消息，但只保存预览摘要，不保存原始消息。
- 字段映射会生成 `data_points(source_type='kafka.field')`，表示建模配置，不表示已经有后台持续采集。
- Avro、Protobuf、Schema Registry、Kafka 写入和生产测试不在本计划内。
- 不改 MQTT 工作台代码，除非共享组件已有类型错误阻塞 Kafka 构建。

---

### 任务 1：编写 Kafka 工作台后端契约测试

**文件：**

- 创建：`data_service/tests/integration/kafka_workbench_api_test.go`

- [ ] **步骤 1：创建失败的 API 生命周期测试**

新增 `data_service/tests/integration/kafka_workbench_api_test.go`，测试先按最终契约编写，当前应因路由不存在而失败。

```go
package integration

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestKafkaWorkbenchLifecycle(t *testing.T) {
	fixture := newIntegrationFixture(t)
	server := newTestServer(t, fixture)
	token := fixture.Token
	projectID := fixture.ProjectID

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

	mappings := mustListKafkaTopicMappings(t, server.URL, token, projectID, kafka.ID)
	if len(mappings) != 1 || mappings[0].Topic != "device.telemetry" {
		t.Fatalf("expected created topic mapping, got %#v", mappings)
	}

	field := mustCreateKafkaField(t, server.URL, token, projectID, mapping.ID, map[string]any{
		"name":      "temperature",
		"valuePath": "temperature",
		"dataType":  "number",
		"enabled":   true,
	})
	if field.SourceType != "kafka.field" {
		t.Fatalf("expected kafka.field datapoint sync, got %q", field.SourceType)
	}

	datapoints := mustListDataPoints(t, server.URL, token, projectID, "kafka.field", kafka.ID)
	if len(datapoints) != 1 {
		t.Fatalf("expected one kafka.field datapoint, got %d", len(datapoints))
	}

	mustDeleteKafkaTopicMapping(t, server.URL, token, projectID, mapping.ID)
	mappings = mustListKafkaTopicMappings(t, server.URL, token, projectID, kafka.ID)
	if len(mappings) != 0 {
		t.Fatalf("expected mapping deleted, got %d", len(mappings))
	}
}
```

- [ ] **步骤 2：补充测试 DTO 和 helper**

在同一文件加入 helper，保持响应只依赖统一 envelope 的 `data`。

```go
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
	ID             string `json:"id"`
	ConnectionID   string `json:"connectionId"`
	TopicMappingID string `json:"topicMappingId"`
	Name           string `json:"name"`
	ValuePath      string `json:"valuePath"`
	DataType       string `json:"dataType"`
	Enabled        bool   `json:"enabled"`
	SourceType     string `json:"sourceType"`
	DataPointID    string `json:"dataPointId"`
}

type kafkaListPayload[T any] struct {
	List []T `json:"list"`
}

func mustCreateKafkaTopicGroup(t *testing.T, baseURL, token, projectID, connectionID string, payload map[string]any) kafkaTopicGroupPayload {
	t.Helper()
	responseEnvelope := doJSONRequest(t, http.MethodPost, baseURL+"/api/v1/data/projects/"+projectID+"/kafka/"+connectionID+"/topic-groups", token, payload)
	var result kafkaTopicGroupPayload
	if err := json.Unmarshal(responseEnvelope.Data, &result); err != nil {
		t.Fatalf("decode kafka topic group response failed: %v", err)
	}
	return result
}

func mustCreateKafkaTopicMapping(t *testing.T, baseURL, token, projectID, connectionID string, payload map[string]any) kafkaTopicMappingPayload {
	t.Helper()
	responseEnvelope := doJSONRequest(t, http.MethodPost, baseURL+"/api/v1/data/projects/"+projectID+"/kafka/"+connectionID+"/topic-mappings", token, payload)
	var result kafkaTopicMappingPayload
	if err := json.Unmarshal(responseEnvelope.Data, &result); err != nil {
		t.Fatalf("decode kafka topic mapping response failed: %v", err)
	}
	return result
}

func mustListKafkaTopicMappings(t *testing.T, baseURL, token, projectID, connectionID string) []kafkaTopicMappingPayload {
	t.Helper()
	responseEnvelope := doJSONRequest(t, http.MethodGet, baseURL+"/api/v1/data/projects/"+projectID+"/kafka/"+connectionID+"/topic-mappings", token, nil)
	var result kafkaListPayload[kafkaTopicMappingPayload]
	if err := json.Unmarshal(responseEnvelope.Data, &result); err != nil {
		t.Fatalf("decode kafka topic mappings response failed: %v", err)
	}
	return result.List
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

func mustDeleteKafkaTopicMapping(t *testing.T, baseURL, token, projectID, mappingID string) {
	t.Helper()
	_ = doJSONRequest(t, http.MethodDelete, baseURL+"/api/v1/data/projects/"+projectID+"/kafka/topic-mappings/"+mappingID, token, nil)
}
```

- [ ] **步骤 3：运行测试验证失败**

运行：

```powershell
cd D:\SVNCode\indu-forge\data_service
go test ./tests/integration -run TestKafkaWorkbenchLifecycle -count=1
```

预期：FAIL，错误为 Kafka 工作台路由返回 404 或 handler 未注册。

- [ ] **步骤 4：提交失败测试**

运行：

```powershell
git add data_service/tests/integration/kafka_workbench_api_test.go
git commit -m "test(data_service): 覆盖 Kafka 工作台契约"
```

预期：只提交测试文件。

---

### 任务 2：新增 Kafka 工作台数据库表

**文件：**

- 创建：`data_service/internal/db/migrations/0030_kafka_workbench.sql`
- 创建：`data_service/internal/db/migrations/0030_kafka_workbench_down.sql`

- [ ] **步骤 1：创建正向迁移**

新增 `data_service/internal/db/migrations/0030_kafka_workbench.sql`：

```sql
CREATE TABLE IF NOT EXISTS data_kafka_topic_groups (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    parent_id uuid,
    name text NOT NULL CHECK (char_length(name) <= 100),
    sort_order integer NOT NULL DEFAULT 0,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_kafka_topic_groups_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE,
    CONSTRAINT data_kafka_topic_groups_identity_key
        UNIQUE (id, project_id, connection_id),
    CONSTRAINT data_kafka_topic_groups_parent_fkey
        FOREIGN KEY (parent_id, project_id, connection_id)
        REFERENCES data_kafka_topic_groups (id, project_id, connection_id) ON DELETE SET NULL,
    CONSTRAINT data_kafka_topic_groups_name_key
        UNIQUE (project_id, connection_id, parent_id, name)
);

CREATE TABLE IF NOT EXISTS data_kafka_topic_mappings (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    group_id uuid,
    name text NOT NULL CHECK (char_length(name) <= 100),
    topic text NOT NULL CHECK (char_length(topic) <= 500),
    description text NOT NULL DEFAULT '',
    partition_mode text NOT NULL DEFAULT 'all' CHECK (partition_mode IN ('all', 'single')),
    partition integer CHECK (partition IS NULL OR partition >= 0),
    start_position text NOT NULL DEFAULT 'latest' CHECK (start_position IN ('latest', 'earliest', 'offset')),
    decode text NOT NULL DEFAULT 'json' CHECK (decode IN ('json', 'string', 'binary')),
    sample_limit integer NOT NULL DEFAULT 100 CHECK (sample_limit >= 1 AND sample_limit <= 1000),
    timeout_ms integer NOT NULL DEFAULT 5000 CHECK (timeout_ms >= 1000 AND timeout_ms <= 30000),
    sort_order integer NOT NULL DEFAULT 0,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_kafka_topic_mappings_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE,
    CONSTRAINT data_kafka_topic_mappings_group_fkey
        FOREIGN KEY (group_id, project_id, connection_id)
        REFERENCES data_kafka_topic_groups (id, project_id, connection_id) ON DELETE SET NULL,
    CONSTRAINT data_kafka_topic_mappings_partition_check
        CHECK (partition_mode <> 'single' OR partition IS NOT NULL),
    CONSTRAINT data_kafka_topic_mappings_topic_key
        UNIQUE (project_id, connection_id, topic)
);

CREATE TABLE IF NOT EXISTS data_kafka_fields (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    topic_mapping_id uuid NOT NULL,
    name text NOT NULL CHECK (char_length(name) <= 100),
    value_path text NOT NULL CHECK (char_length(value_path) <= 500),
    key_path text NOT NULL DEFAULT '' CHECK (char_length(key_path) <= 500),
    data_type text NOT NULL CHECK (char_length(data_type) <= 50),
    enabled boolean NOT NULL DEFAULT true,
    description text NOT NULL DEFAULT '',
    sort_order integer NOT NULL DEFAULT 0,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_kafka_fields_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE,
    CONSTRAINT data_kafka_fields_mapping_fkey
        FOREIGN KEY (topic_mapping_id) REFERENCES data_kafka_topic_mappings (id) ON DELETE CASCADE,
    CONSTRAINT data_kafka_fields_value_path_key
        UNIQUE (project_id, topic_mapping_id, value_path)
);

CREATE INDEX IF NOT EXISTS data_kafka_topic_groups_tree_idx
    ON data_kafka_topic_groups (project_id, connection_id, parent_id, sort_order, created_at);

CREATE INDEX IF NOT EXISTS data_kafka_topic_mappings_group_idx
    ON data_kafka_topic_mappings (project_id, connection_id, group_id, sort_order, created_at);

CREATE INDEX IF NOT EXISTS data_kafka_fields_connection_idx
    ON data_kafka_fields (project_id, connection_id);
```

- [ ] **步骤 2：创建回滚迁移**

新增 `data_service/internal/db/migrations/0030_kafka_workbench_down.sql`：

```sql
DROP INDEX IF EXISTS data_kafka_fields_connection_idx;
DROP TABLE IF EXISTS data_kafka_fields;
DROP INDEX IF EXISTS data_kafka_topic_mappings_group_idx;
DROP TABLE IF EXISTS data_kafka_topic_mappings;
DROP INDEX IF EXISTS data_kafka_topic_groups_tree_idx;
DROP TABLE IF EXISTS data_kafka_topic_groups;
```

- [ ] **步骤 3：运行迁移编译测试**

运行：

```powershell
cd D:\SVNCode\indu-forge\data_service
go test ./internal/db/... -count=1
```

预期：PASS，embed migrations 编译通过。

- [ ] **步骤 4：提交迁移**

运行：

```powershell
git add data_service/internal/db/migrations/0030_kafka_workbench.sql data_service/internal/db/migrations/0030_kafka_workbench_down.sql
git commit -m "feat(data_service): 新增 Kafka 工作台表"
```

---

### 任务 3：实现 Kafka 工作台 repository

**文件：**

- 创建：`data_service/internal/repository/kafka_workbench_repository.go`

- [ ] **步骤 1：定义 record 和参数类型**

创建 `data_service/internal/repository/kafka_workbench_repository.go`：

```go
package repository

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/indu-forge/data_service/internal/errors"
)

type KafkaTopicGroupRecord struct {
	ID           string
	ProjectID    string
	ConnectionID string
	ParentID     *string
	Name         string
	SortOrder    int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type KafkaTopicMappingRecord struct {
	ID            string
	ProjectID     string
	ConnectionID  string
	GroupID       *string
	Name          string
	Topic         string
	Description   string
	PartitionMode string
	Partition     *int
	StartPosition string
	Decode        string
	SampleLimit   int
	TimeoutMS     int
	SortOrder     int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type KafkaFieldRecord struct {
	ID             string
	ProjectID      string
	ConnectionID   string
	TopicMappingID string
	Name           string
	ValuePath      string
	KeyPath         string
	DataType       string
	Enabled        bool
	Description    string
	SortOrder      int
	DataPointID    *string
	DataPointPath  *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type KafkaWorkbenchRepository struct {
	pool *pgxpool.Pool
}

func NewKafkaWorkbenchRepository(pool *pgxpool.Pool) *KafkaWorkbenchRepository {
	return &KafkaWorkbenchRepository{pool: pool}
}
```

- [ ] **步骤 2：实现 Topic 分组 SQL**

在同一文件实现：

```go
func (r *KafkaWorkbenchRepository) ListTopicGroups(ctx context.Context, projectID, connectionID string) ([]KafkaTopicGroupRecord, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, project_id, connection_id, parent_id, name, sort_order, created_at, updated_at
		FROM data_kafka_topic_groups
		WHERE project_id = $1 AND connection_id = $2
		ORDER BY sort_order ASC, created_at ASC
	`, projectID, connectionID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询 Kafka Topic 分组失败", err)
	}
	defer rows.Close()

	result := []KafkaTopicGroupRecord{}
	for rows.Next() {
		var record KafkaTopicGroupRecord
		if err := rows.Scan(&record.ID, &record.ProjectID, &record.ConnectionID, &record.ParentID, &record.Name, &record.SortOrder, &record.CreatedAt, &record.UpdatedAt); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取 Kafka Topic 分组失败", err)
		}
		result = append(result, record)
	}
	return result, rows.Err()
}

func (r *KafkaWorkbenchRepository) CreateTopicGroup(ctx context.Context, params CreateKafkaTopicGroupParams) (*KafkaTopicGroupRecord, error) {
	record := KafkaTopicGroupRecord{}
	err := r.pool.QueryRow(ctx, `
		INSERT INTO data_kafka_topic_groups (project_id, connection_id, parent_id, name, sort_order, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $6)
		RETURNING id, project_id, connection_id, parent_id, name, sort_order, created_at, updated_at
	`, params.ProjectID, params.ConnectionID, params.ParentID, params.Name, params.SortOrder, params.UserID).Scan(
		&record.ID, &record.ProjectID, &record.ConnectionID, &record.ParentID, &record.Name, &record.SortOrder, &record.CreatedAt, &record.UpdatedAt,
	)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "创建 Kafka Topic 分组失败", err)
	}
	return &record, nil
}
```

同时定义：

```go
type CreateKafkaTopicGroupParams struct {
	ProjectID    string
	ConnectionID string
	ParentID     *string
	Name         string
	SortOrder    int
	UserID       string
}

type UpdateKafkaTopicGroupParams struct {
	ProjectID string
	GroupID   string
	ParentID  *string
	HasParent bool
	Name      string
	SortOrder int
	UserID    string
}
```

实现 `UpdateTopicGroup` 和 `DeleteTopicGroup`。`DeleteTopicGroup` 在事务内先执行：

```sql
UPDATE data_kafka_topic_mappings SET group_id = NULL WHERE project_id = $1 AND group_id = $2;
UPDATE data_kafka_topic_groups SET parent_id = NULL WHERE project_id = $1 AND parent_id = $2;
DELETE FROM data_kafka_topic_groups WHERE project_id = $1 AND id = $2;
```

- [ ] **步骤 3：实现 Topic 映射 SQL**

实现：

```go
func (r *KafkaWorkbenchRepository) ListTopicMappings(ctx context.Context, projectID, connectionID string) ([]KafkaTopicMappingRecord, error)
func (r *KafkaWorkbenchRepository) GetTopicMapping(ctx context.Context, projectID, mappingID string) (*KafkaTopicMappingRecord, error)
func (r *KafkaWorkbenchRepository) CreateTopicMapping(ctx context.Context, params CreateKafkaTopicMappingParams) (*KafkaTopicMappingRecord, error)
func (r *KafkaWorkbenchRepository) UpdateTopicMapping(ctx context.Context, params UpdateKafkaTopicMappingParams) (*KafkaTopicMappingRecord, error)
func (r *KafkaWorkbenchRepository) DeleteTopicMapping(ctx context.Context, projectID, mappingID string) error
```

`CreateTopicMapping` 使用：

```sql
INSERT INTO data_kafka_topic_mappings (
    project_id, connection_id, group_id, name, topic, description,
    partition_mode, partition, start_position, decode, sample_limit,
    timeout_ms, sort_order, created_by, updated_by
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $14)
RETURNING id, project_id, connection_id, group_id, name, topic, description,
          partition_mode, partition, start_position, decode, sample_limit,
          timeout_ms, sort_order, created_at, updated_at
```

- [ ] **步骤 4：实现字段映射与数据点同步 SQL**

实现：

```go
func (r *KafkaWorkbenchRepository) ListFields(ctx context.Context, projectID, mappingID string) ([]KafkaFieldRecord, error)
func (r *KafkaWorkbenchRepository) CreateFieldWithDataPoint(ctx context.Context, params CreateKafkaFieldParams) (*KafkaFieldRecord, error)
func (r *KafkaWorkbenchRepository) UpdateFieldWithDataPoint(ctx context.Context, params UpdateKafkaFieldParams) (*KafkaFieldRecord, error)
func (r *KafkaWorkbenchRepository) DeleteFieldWithDataPoint(ctx context.Context, projectID, fieldID string) error
```

字段创建必须在事务中写 `data_kafka_fields`，随后 upsert `data_points`：

```sql
INSERT INTO data_points (
    project_id, path, name, source_type, source_id, source_config,
    data_type, status, created_by, updated_by
) VALUES ($1, $2, $3, 'kafka.field', $4, $5::jsonb, $6, $7, $8, $8)
ON CONFLICT (project_id, path)
DO UPDATE SET
    name = EXCLUDED.name,
    source_type = EXCLUDED.source_type,
    source_id = EXCLUDED.source_id,
    source_config = EXCLUDED.source_config,
    data_type = EXCLUDED.data_type,
    status = EXCLUDED.status,
    updated_by = EXCLUDED.updated_by,
    updated_at = now()
RETURNING id, path
```

数据点 path 由 service 传入，格式固定为：

```text
kafka.{topicMappingName}.{fieldName}
```

service 需要先用现有 datapoint path 工具或等价规则清理为合法 path 片段。

- [ ] **步骤 5：实现预览记录摘要写入**

实现：

```go
func (r *KafkaWorkbenchRepository) CreatePreviewRecord(ctx context.Context, params CreateAccessSourceRecordParams) error {
	return r.CreateAccessSourceRecord(ctx, params)
}
```

如果 `CreateAccessSourceRecord` 当前只在 `ProtocolPreviewRepository` 上，改为在 `KafkaWorkbenchRepository` 中复制同样的参数化 INSERT，不抽公共抽象。

- [ ] **步骤 6：运行 repository 编译测试**

运行：

```powershell
cd D:\SVNCode\indu-forge\data_service
go test ./internal/repository -count=1
```

预期：PASS。

- [ ] **步骤 7：提交 repository**

运行：

```powershell
gofmt -w data_service/internal/repository/kafka_workbench_repository.go
git add data_service/internal/repository/kafka_workbench_repository.go
git commit -m "feat(data_service): 增加 Kafka 工作台仓储"
```

---

### 任务 4：实现 Kafka 工作台 service

**文件：**

- 创建：`data_service/internal/service/kafka_workbench_service.go`
- 修改：`data_service/internal/service/protocol_preview_adapters.go`

- [ ] **步骤 1：定义 service 输出模型**

创建 `data_service/internal/service/kafka_workbench_service.go`：

```go
package service

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

type KafkaTopicGroup struct {
	ID           string  `json:"id"`
	ProjectID    string  `json:"projectId"`
	ConnectionID string  `json:"connectionId"`
	ParentID     *string `json:"parentId"`
	Name         string  `json:"name"`
	SortOrder    int     `json:"sortOrder"`
	CreatedAt    string  `json:"createdAt"`
	UpdatedAt    string  `json:"updatedAt"`
}

type KafkaTopicMapping struct {
	ID            string  `json:"id"`
	ProjectID     string  `json:"projectId"`
	ConnectionID  string  `json:"connectionId"`
	GroupID       *string `json:"groupId"`
	Name          string  `json:"name"`
	Topic         string  `json:"topic"`
	Description   string  `json:"description"`
	PartitionMode string  `json:"partitionMode"`
	Partition     *int    `json:"partition"`
	StartPosition string  `json:"startPosition"`
	Decode        string  `json:"decode"`
	SampleLimit   int     `json:"sampleLimit"`
	TimeoutMS     int     `json:"timeoutMs"`
	SortOrder     int     `json:"sortOrder"`
	CreatedAt     string  `json:"createdAt"`
	UpdatedAt     string  `json:"updatedAt"`
}

type KafkaField struct {
	ID             string `json:"id"`
	ProjectID      string `json:"projectId"`
	ConnectionID   string `json:"connectionId"`
	TopicMappingID string `json:"topicMappingId"`
	Name           string `json:"name"`
	ValuePath      string `json:"valuePath"`
	KeyPath         string `json:"keyPath"`
	DataType       string `json:"dataType"`
	Enabled        bool   `json:"enabled"`
	Description    string `json:"description"`
	SortOrder      int    `json:"sortOrder"`
	SourceType     string `json:"sourceType"`
	DataPointID    string `json:"dataPointId,omitempty"`
	DataPointPath  string `json:"dataPointPath,omitempty"`
}

type KafkaWorkbenchService struct {
	repository *repository.KafkaWorkbenchRepository
	preview    KafkaPreviewAdapter
}

func NewKafkaWorkbenchService(repo *repository.KafkaWorkbenchRepository) *KafkaWorkbenchService {
	return &KafkaWorkbenchService{repository: repo, preview: KafkaPreviewAdapter{}}
}
```

- [ ] **步骤 2：实现输入校验和映射转换**

添加输入类型：

```go
type CreateKafkaTopicMappingInput struct {
	Name          string
	GroupID       *string
	Topic         string
	Description   string
	PartitionMode string
	Partition     *int
	StartPosition string
	Decode        string
	SampleLimit   int
	TimeoutMS     int
	SortOrder     int
}

type CreateKafkaFieldInput struct {
	Name        string
	ValuePath   string
	KeyPath     string
	DataType    string
	Enabled     *bool
	Description string
	SortOrder   int
}
```

实现规范化函数：

```go
func normalizeKafkaTopicMappingInput(input CreateKafkaTopicMappingInput) (CreateKafkaTopicMappingInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Topic = strings.TrimSpace(input.Topic)
	input.Description = strings.TrimSpace(input.Description)
	if input.Name == "" {
		input.Name = input.Topic
	}
	if input.Topic == "" {
		return input, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "topic 不能为空")
	}
	input.PartitionMode = strings.ToLower(strings.TrimSpace(input.PartitionMode))
	if input.PartitionMode == "" {
		input.PartitionMode = "all"
	}
	if input.PartitionMode != "all" && input.PartitionMode != "single" {
		return input, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "partitionMode 仅支持 all/single")
	}
	if input.PartitionMode == "single" && input.Partition == nil {
		return input, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "指定单分区时 partition 不能为空")
	}
	input.StartPosition = strings.ToLower(strings.TrimSpace(input.StartPosition))
	if input.StartPosition == "" {
		input.StartPosition = "latest"
	}
	if _, ok := allowedKafkaStartPositions[input.StartPosition]; !ok && input.StartPosition != "offset" {
		return input, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "startPosition 仅支持 latest/earliest/offset")
	}
	input.Decode = strings.ToLower(strings.TrimSpace(input.Decode))
	if input.Decode == "" {
		input.Decode = "json"
	}
	if input.Decode != "json" && input.Decode != "string" && input.Decode != "binary" {
		return input, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "decode 仅支持 json/string/binary")
	}
	if input.SampleLimit <= 0 {
		input.SampleLimit = 100
	}
	if input.SampleLimit > 1000 {
		input.SampleLimit = 1000
	}
	if input.TimeoutMS <= 0 {
		input.TimeoutMS = 5000
	}
	if input.TimeoutMS > 30000 {
		input.TimeoutMS = 30000
	}
	return input, nil
}
```

- [ ] **步骤 3：实现 Topic 分组和映射 service 方法**

实现：

```go
func (s *KafkaWorkbenchService) ListTopicGroups(ctx context.Context, projectID, connectionID string) ([]KafkaTopicGroup, error)
func (s *KafkaWorkbenchService) CreateTopicGroup(ctx context.Context, projectID, connectionID, userID string, input CreateKafkaTopicGroupInput) (*KafkaTopicGroup, error)
func (s *KafkaWorkbenchService) UpdateTopicGroup(ctx context.Context, projectID, groupID, userID string, input UpdateKafkaTopicGroupInput) (*KafkaTopicGroup, error)
func (s *KafkaWorkbenchService) DeleteTopicGroup(ctx context.Context, projectID, groupID string) error
func (s *KafkaWorkbenchService) ListTopicMappings(ctx context.Context, projectID, connectionID string) ([]KafkaTopicMapping, error)
func (s *KafkaWorkbenchService) CreateTopicMapping(ctx context.Context, projectID, connectionID, userID string, input CreateKafkaTopicMappingInput) (*KafkaTopicMapping, error)
func (s *KafkaWorkbenchService) UpdateTopicMapping(ctx context.Context, projectID, mappingID, userID string, input CreateKafkaTopicMappingInput) (*KafkaTopicMapping, error)
func (s *KafkaWorkbenchService) DeleteTopicMapping(ctx context.Context, projectID, mappingID string) error
```

每个入口先调用：

```go
if err := validateProjectID(projectID); err != nil {
	return nil, err
}
if err := validateConnectionID(connectionID); err != nil {
	return nil, err
}
```

`DeleteTopicMapping` 直接调用 repository 的级联删除，依赖 `data_kafka_fields` 外键级联，并在 repository 内清理对应数据点。

- [ ] **步骤 4：扩展 KafkaPreviewAdapter**

修改 `data_service/internal/service/protocol_preview_adapters.go` 的 `KafkaPreviewAdapter.Preview`：

```go
partitionMode := strings.ToLower(strings.TrimSpace(toString(input.Options["partitionMode"])))
partition := intFromAny(input.Options["partition"], -1)
offset := intFromAny(input.Options["offset"], -1)
decode := strings.ToLower(strings.TrimSpace(toString(input.Options["decode"])))
if decode == "" {
	decode = "json"
}

readerConfig := kafka.ReaderConfig{
	Brokers:  brokers,
	Topic:    topic,
	MinBytes: 1,
	MaxBytes: maxProtocolPreviewPayloadBytes,
	MaxWait:  input.Timeout,
}
if partitionMode == "single" && partition >= 0 {
	readerConfig.Partition = partition
	readerConfig.StartOffset = startOffset
	if offset >= 0 {
		readerConfig.StartOffset = int64(offset)
	}
} else {
	readerConfig.GroupID = fmt.Sprintf("data-service-preview-%s-%d", input.Connection.ID, time.Now().UnixNano())
	readerConfig.StartOffset = startOffset
}
```

更新 `kafkaMessagePreviewSample`：

```go
func kafkaMessagePreviewSample(message kafka.Message) map[string]any {
	value, _, rawPayload := decodePreviewPayload(message.Value)
	headers := map[string]string{}
	for _, header := range message.Headers {
		if isSensitiveConfigKey(header.Key) {
			headers[header.Key] = "******"
			continue
		}
		headers[header.Key] = string(header.Value)
	}
	return map[string]any{
		"topic":      message.Topic,
		"partition":  message.Partition,
		"offset":     message.Offset,
		"key":        string(message.Key),
		"headers":    headers,
		"value":      value,
		"rawPayload": rawPayload,
		"timestamp":  message.Time,
	}
}
```

说明：`decode` 当前只影响前端和 schema 语义。底层仍优先尝试 JSON，失败返回 string；binary 消息以 raw 字符串摘要展示，不做长期保存。

- [ ] **步骤 5：实现工作台预览 service**

在 `kafka_workbench_service.go` 实现：

```go
type KafkaPreviewInput struct {
	Topic         string
	PartitionMode string
	Partition     *int
	StartPosition string
	Offset        *int
	Limit         int
	TimeoutMS     int
	Decode        string
}

func (s *KafkaWorkbenchService) PreviewTopicMapping(ctx context.Context, projectID, mappingID string, input KafkaPreviewInput) (*ProtocolPreviewResult, error) {
	mapping, err := s.repository.GetTopicMapping(ctx, projectID, mappingID)
	if err != nil {
		return nil, err
	}
	if input.Topic == "" {
		input.Topic = mapping.Topic
	}
	if input.PartitionMode == "" {
		input.PartitionMode = mapping.PartitionMode
	}
	if input.Partition == nil {
		input.Partition = mapping.Partition
	}
	if input.StartPosition == "" {
		input.StartPosition = mapping.StartPosition
	}
	if input.Decode == "" {
		input.Decode = mapping.Decode
	}
	if input.Limit <= 0 {
		input.Limit = mapping.SampleLimit
	}
	if input.TimeoutMS <= 0 {
		input.TimeoutMS = mapping.TimeoutMS
	}
	return s.previewConnection(ctx, projectID, mapping.ConnectionID, input)
}
```

`previewConnection` 查询连接配置后构造 `ProtocolPreviewAdapterInput`，把 topic 覆盖到 `Connection.Config`：

```go
connection, err := s.repository.GetPreviewConnection(ctx, projectID, connectionID)
if err != nil {
	return nil, err
}
config := cloneMap(connection.Config)
config["topic"] = input.Topic
connection.Config = config

result, err := s.preview.Preview(previewCtx, ProtocolPreviewAdapterInput{
	Connection: *connection,
	Limit:      limit,
	Timeout:    timeout,
	Options: map[string]any{
		"partitionMode": input.PartitionMode,
		"partition":     input.Partition,
		"offset":        input.Offset,
		"decode":        input.Decode,
	},
})
```

预览成功或失败后都写 `RecordType = "kafka.preview"` 的摘要记录，detail 只保存 `durationMs`、`sampleCount`、`truncated`、`topic`、`partitionMode`。

- [ ] **步骤 6：实现字段映射和 schema 推断**

实现字段路径推断：

```go
type KafkaSchemaField struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

func inferKafkaSchemaFields(samples []any) []KafkaSchemaField {
	fields := []KafkaSchemaField{}
	seen := map[string]struct{}{}
	for _, sample := range samples {
		mapped, ok := sample.(map[string]any)
		if !ok {
			continue
		}
		value, ok := mapped["value"]
		if !ok {
			continue
		}
		collectKafkaSchemaFields("", value, seen, &fields)
	}
	return fields
}

func collectKafkaSchemaFields(prefix string, value any, seen map[string]struct{}, fields *[]KafkaSchemaField) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			next := key
			if prefix != "" {
				next = prefix + "." + key
			}
			collectKafkaSchemaFields(next, child, seen, fields)
		}
	default:
		if prefix == "" {
			return
		}
		if _, ok := seen[prefix]; ok {
			return
		}
		seen[prefix] = struct{}{}
		*fields = append(*fields, KafkaSchemaField{Path: prefix, Type: kafkaFieldType(value)})
	}
}
```

字段创建时生成数据点 path：

```go
var pathSegmentPattern = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

func kafkaDataPointPath(mapping repository.KafkaTopicMappingRecord, fieldName string) string {
	topicPart := pathSegmentPattern.ReplaceAllString(strings.TrimSpace(mapping.Name), "_")
	fieldPart := pathSegmentPattern.ReplaceAllString(strings.TrimSpace(fieldName), "_")
	topicPart = strings.Trim(topicPart, "_")
	fieldPart = strings.Trim(fieldPart, "_")
	if topicPart == "" {
		topicPart = "topic"
	}
	if fieldPart == "" {
		fieldPart = "field"
	}
	return fmt.Sprintf("kafka.%s.%s", strings.ToLower(topicPart), strings.ToLower(fieldPart))
}
```

- [ ] **步骤 7：运行 service 测试**

运行：

```powershell
cd D:\SVNCode\indu-forge\data_service
go test ./internal/service -run "TestKafka|TestProtocolPreview" -count=1
```

预期：PASS。

- [ ] **步骤 8：提交 service**

运行：

```powershell
gofmt -w data_service/internal/service/kafka_workbench_service.go data_service/internal/service/protocol_preview_adapters.go
git add data_service/internal/service/kafka_workbench_service.go data_service/internal/service/protocol_preview_adapters.go
git commit -m "feat(data_service): 实现 Kafka 工作台服务"
```

---

### 任务 5：挂载 Kafka 工作台 HTTP 接口

**文件：**

- 创建：`data_service/internal/http/handler/kafka_workbench_handler.go`
- 修改：`data_service/internal/http/router/router.go`
- 修改：`data_service/internal/app/server.go`

- [ ] **步骤 1：创建 handler**

新增 `data_service/internal/http/handler/kafka_workbench_handler.go`：

```go
package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/indu-forge/data_service/internal/auth"
	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/service"
)

type KafkaWorkbenchHandler struct {
	service *service.KafkaWorkbenchService
}

func NewKafkaWorkbenchHandler(kafkaService *service.KafkaWorkbenchService) *KafkaWorkbenchHandler {
	return &KafkaWorkbenchHandler{service: kafkaService}
}

func (h *KafkaWorkbenchHandler) ListTopicGroups(w http.ResponseWriter, r *http.Request) error {
	groups, err := h.service.ListTopicGroups(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"))
	if err != nil {
		return err
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"list": groups})
	return nil
}

func (h *KafkaWorkbenchHandler) CreateTopicGroup(w http.ResponseWriter, r *http.Request) error {
	claims, _ := auth.ClaimsFromContext(r.Context())
	var input service.CreateKafkaTopicGroupInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		return badRequest("请求体格式无效", err)
	}
	group, err := h.service.CreateTopicGroup(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, input)
	if err != nil {
		return err
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), group)
	return nil
}
```

同文件继续实现：

```go
func (h *KafkaWorkbenchHandler) UpdateTopicGroup(w http.ResponseWriter, r *http.Request) error
func (h *KafkaWorkbenchHandler) DeleteTopicGroup(w http.ResponseWriter, r *http.Request) error
func (h *KafkaWorkbenchHandler) ListTopicMappings(w http.ResponseWriter, r *http.Request) error
func (h *KafkaWorkbenchHandler) CreateTopicMapping(w http.ResponseWriter, r *http.Request) error
func (h *KafkaWorkbenchHandler) GetTopicMapping(w http.ResponseWriter, r *http.Request) error
func (h *KafkaWorkbenchHandler) UpdateTopicMapping(w http.ResponseWriter, r *http.Request) error
func (h *KafkaWorkbenchHandler) DeleteTopicMapping(w http.ResponseWriter, r *http.Request) error
func (h *KafkaWorkbenchHandler) PreviewConnection(w http.ResponseWriter, r *http.Request) error
func (h *KafkaWorkbenchHandler) PreviewTopicMapping(w http.ResponseWriter, r *http.Request) error
func (h *KafkaWorkbenchHandler) ListFields(w http.ResponseWriter, r *http.Request) error
func (h *KafkaWorkbenchHandler) CreateField(w http.ResponseWriter, r *http.Request) error
func (h *KafkaWorkbenchHandler) CreateFieldsBatch(w http.ResponseWriter, r *http.Request) error
func (h *KafkaWorkbenchHandler) UpdateField(w http.ResponseWriter, r *http.Request) error
func (h *KafkaWorkbenchHandler) DeleteField(w http.ResponseWriter, r *http.Request) error
func (h *KafkaWorkbenchHandler) ToggleField(w http.ResponseWriter, r *http.Request) error
```

`CreateFieldsBatch` 请求体：

```go
var input struct {
	Fields []service.CreateKafkaFieldInput `json:"fields"`
}
```

返回：

```go
response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"list": fields})
```

- [ ] **步骤 2：在 router options 中增加 handler**

修改 `data_service/internal/http/router/router.go` 的 `options`：

```go
kafkaWorkbenchHandler *handler.KafkaWorkbenchHandler
```

增加 Option：

```go
func WithKafkaWorkbenchRoutes(kafkaHandler *handler.KafkaWorkbenchHandler, jwtValidator *auth.JWTValidator) Option {
	return func(opts *options) {
		opts.kafkaWorkbenchHandler = kafkaHandler
		opts.jwtValidator = jwtValidator
	}
}
```

- [ ] **步骤 3：注册 Kafka 工作台路由**

在 router 中新增 `registerKafkaWorkbenchRoutes`，路径如下：

```go
mux.Handle("GET /api/v1/data/projects/{projectId}/kafka/{connectionId}/topic-groups", chain(middleware.Authenticate(opts.jwtValidator), middleware.ErrorHandler(opts.kafkaWorkbenchHandler.ListTopicGroups)))
mux.Handle("POST /api/v1/data/projects/{projectId}/kafka/{connectionId}/topic-groups", chain(middleware.Authenticate(opts.jwtValidator), middleware.ErrorHandler(opts.kafkaWorkbenchHandler.CreateTopicGroup)))
mux.Handle("PUT /api/v1/data/projects/{projectId}/kafka/topic-groups/{groupId}", chain(middleware.Authenticate(opts.jwtValidator), middleware.ErrorHandler(opts.kafkaWorkbenchHandler.UpdateTopicGroup)))
mux.Handle("DELETE /api/v1/data/projects/{projectId}/kafka/topic-groups/{groupId}", chain(middleware.Authenticate(opts.jwtValidator), middleware.ErrorHandler(opts.kafkaWorkbenchHandler.DeleteTopicGroup)))

mux.Handle("GET /api/v1/data/projects/{projectId}/kafka/{connectionId}/topic-mappings", chain(middleware.Authenticate(opts.jwtValidator), middleware.ErrorHandler(opts.kafkaWorkbenchHandler.ListTopicMappings)))
mux.Handle("POST /api/v1/data/projects/{projectId}/kafka/{connectionId}/topic-mappings", chain(middleware.Authenticate(opts.jwtValidator), middleware.ErrorHandler(opts.kafkaWorkbenchHandler.CreateTopicMapping)))
mux.Handle("GET /api/v1/data/projects/{projectId}/kafka/topic-mappings/{mappingId}", chain(middleware.Authenticate(opts.jwtValidator), middleware.ErrorHandler(opts.kafkaWorkbenchHandler.GetTopicMapping)))
mux.Handle("PUT /api/v1/data/projects/{projectId}/kafka/topic-mappings/{mappingId}", chain(middleware.Authenticate(opts.jwtValidator), middleware.ErrorHandler(opts.kafkaWorkbenchHandler.UpdateTopicMapping)))
mux.Handle("DELETE /api/v1/data/projects/{projectId}/kafka/topic-mappings/{mappingId}", chain(middleware.Authenticate(opts.jwtValidator), middleware.ErrorHandler(opts.kafkaWorkbenchHandler.DeleteTopicMapping)))

mux.Handle("POST /api/v1/data/projects/{projectId}/kafka/{connectionId}/preview", chain(middleware.Authenticate(opts.jwtValidator), middleware.ErrorHandler(opts.kafkaWorkbenchHandler.PreviewConnection)))
mux.Handle("POST /api/v1/data/projects/{projectId}/kafka/topic-mappings/{mappingId}/preview", chain(middleware.Authenticate(opts.jwtValidator), middleware.ErrorHandler(opts.kafkaWorkbenchHandler.PreviewTopicMapping)))

mux.Handle("GET /api/v1/data/projects/{projectId}/kafka/topic-mappings/{mappingId}/fields", chain(middleware.Authenticate(opts.jwtValidator), middleware.ErrorHandler(opts.kafkaWorkbenchHandler.ListFields)))
mux.Handle("POST /api/v1/data/projects/{projectId}/kafka/topic-mappings/{mappingId}/fields", chain(middleware.Authenticate(opts.jwtValidator), middleware.ErrorHandler(opts.kafkaWorkbenchHandler.CreateField)))
mux.Handle("POST /api/v1/data/projects/{projectId}/kafka/topic-mappings/{mappingId}/fields/batch", chain(middleware.Authenticate(opts.jwtValidator), middleware.ErrorHandler(opts.kafkaWorkbenchHandler.CreateFieldsBatch)))
mux.Handle("PUT /api/v1/data/projects/{projectId}/kafka/fields/{fieldId}", chain(middleware.Authenticate(opts.jwtValidator), middleware.ErrorHandler(opts.kafkaWorkbenchHandler.UpdateField)))
mux.Handle("DELETE /api/v1/data/projects/{projectId}/kafka/fields/{fieldId}", chain(middleware.Authenticate(opts.jwtValidator), middleware.ErrorHandler(opts.kafkaWorkbenchHandler.DeleteField)))
mux.Handle("PATCH /api/v1/data/projects/{projectId}/kafka/fields/{fieldId}/toggle", chain(middleware.Authenticate(opts.jwtValidator), middleware.ErrorHandler(opts.kafkaWorkbenchHandler.ToggleField)))
```

在主注册流程调用 `registerKafkaWorkbenchRoutes`。

- [ ] **步骤 4：在 server 初始化 Kafka 工作台**

修改 `data_service/internal/app/server.go`，创建：

```go
kafkaWorkbenchRepository := repository.NewKafkaWorkbenchRepository(dbPool)
kafkaWorkbenchService := service.NewKafkaWorkbenchService(kafkaWorkbenchRepository)
kafkaWorkbenchHandler := handler.NewKafkaWorkbenchHandler(kafkaWorkbenchService)
```

传入 router options：

```go
router.WithKafkaWorkbenchRoutes(kafkaWorkbenchHandler, jwtValidator)
```

- [ ] **步骤 5：运行后端集成测试**

运行：

```powershell
cd D:\SVNCode\indu-forge\data_service
go test ./tests/integration -run TestKafkaWorkbenchLifecycle -count=1
```

预期：PASS。此测试不依赖真实 Kafka，只覆盖配置、映射、字段和数据点同步。真实 Kafka 预览由已有 `protocol_preview_adapters_test.go` 和后续手动联调覆盖。

- [ ] **步骤 6：提交 HTTP 接口**

运行：

```powershell
gofmt -w data_service/internal/http/handler/kafka_workbench_handler.go data_service/internal/http/router/router.go data_service/internal/app/server.go
git add data_service/internal/http/handler/kafka_workbench_handler.go data_service/internal/http/router/router.go data_service/internal/app/server.go data_service/tests/integration/kafka_workbench_api_test.go
git commit -m "feat(data_service): 暴露 Kafka 工作台接口"
```

---

### 任务 6：实现前端 Kafka API、schema 和类型

**文件：**

- 修改：`datacenter/src/api/data.api.ts`
- 创建：`datacenter/src/api/schemas/kafka-workbench.schema.ts`
- 创建：`datacenter/src/components/kafka/types.ts`

- [ ] **步骤 1：创建 Zod schema**

新增 `datacenter/src/api/schemas/kafka-workbench.schema.ts`：

```ts
import { z } from 'zod'
import { listResponseSchema } from './common.schema'

export const KafkaTopicGroupSchema = z.object({
  id: z.string(),
  projectId: z.string().optional(),
  connectionId: z.string(),
  parentId: z.string().nullable().optional(),
  name: z.string(),
  sortOrder: z.number().default(0),
})

export const KafkaTopicMappingSchema = z.object({
  id: z.string(),
  projectId: z.string().optional(),
  connectionId: z.string(),
  groupId: z.string().nullable().optional(),
  name: z.string(),
  topic: z.string(),
  description: z.string().default(''),
  partitionMode: z.enum(['all', 'single']).default('all'),
  partition: z.number().nullable().optional(),
  startPosition: z.enum(['latest', 'earliest', 'offset']).default('latest'),
  decode: z.enum(['json', 'string', 'binary']).default('json'),
  sampleLimit: z.number().default(100),
  timeoutMs: z.number().default(5000),
  sortOrder: z.number().default(0),
})

export const KafkaFieldSchema = z.object({
  id: z.string(),
  projectId: z.string().optional(),
  connectionId: z.string(),
  topicMappingId: z.string(),
  name: z.string(),
  valuePath: z.string(),
  keyPath: z.string().default(''),
  dataType: z.string(),
  enabled: z.boolean().default(true),
  description: z.string().default(''),
  sortOrder: z.number().default(0),
  sourceType: z.string().optional(),
  dataPointId: z.string().optional(),
  dataPointPath: z.string().optional(),
})

export const KafkaPreviewSampleSchema = z.object({
  topic: z.string().optional(),
  partition: z.number().optional(),
  offset: z.number().optional(),
  key: z.string().optional(),
  headers: z.record(z.string(), z.string()).optional(),
  value: z.unknown(),
  rawPayload: z.string().optional(),
  timestamp: z.string().optional(),
})

export const KafkaPreviewSchema = z.object({
  status: z.string(),
  topic: z.string().optional(),
  samples: z.array(KafkaPreviewSampleSchema).default([]),
  schema: z.record(z.string(), z.unknown()).default({}),
  diagnostics: z.record(z.string(), z.unknown()).default({}),
})

export const KafkaTopicGroupListSchema = listResponseSchema(KafkaTopicGroupSchema)
export const KafkaTopicMappingListSchema = listResponseSchema(KafkaTopicMappingSchema)
export const KafkaFieldListSchema = listResponseSchema(KafkaFieldSchema)

export type KafkaTopicGroup = z.infer<typeof KafkaTopicGroupSchema>
export type KafkaTopicMapping = z.infer<typeof KafkaTopicMappingSchema>
export type KafkaField = z.infer<typeof KafkaFieldSchema>
export type KafkaPreviewSample = z.infer<typeof KafkaPreviewSampleSchema>
export type KafkaPreview = z.infer<typeof KafkaPreviewSchema>
```

- [ ] **步骤 2：创建组件共享类型**

新增 `datacenter/src/components/kafka/types.ts`：

```ts
import type {
  KafkaField,
  KafkaPreview,
  KafkaPreviewSample,
  KafkaTopicGroup,
  KafkaTopicMapping,
} from '@/api/schemas/kafka-workbench.schema'

export type {
  KafkaField,
  KafkaPreview,
  KafkaPreviewSample,
  KafkaTopicGroup,
  KafkaTopicMapping,
}

export type KafkaTopicGroupNode = KafkaTopicGroup & {
  children: KafkaTopicGroupNode[]
  mappings: KafkaTopicMapping[]
}

export type KafkaWorkbenchConnection = {
  id: string
  name?: string
  type?: string
  status?: string
  config?: Record<string, unknown>
}
```

- [ ] **步骤 3：新增 API 方法**

在 `datacenter/src/api/data.api.ts` 导入 schema：

```ts
import {
  KafkaFieldListSchema,
  KafkaFieldSchema,
  KafkaPreviewSchema,
  KafkaTopicGroupListSchema,
  KafkaTopicGroupSchema,
  KafkaTopicMappingListSchema,
  KafkaTopicMappingSchema,
} from './schemas/kafka-workbench.schema'
```

新增方法：

```ts
export const getKafkaTopicGroups = async (projectId, connectionId) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/${connectionId}/topic-groups`,
    method: 'get',
  })
  return KafkaTopicGroupListSchema.parse(res)
}

export const createKafkaTopicGroup = async (projectId, connectionId, data) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/${connectionId}/topic-groups`,
    method: 'post',
    data,
  })
  return KafkaTopicGroupSchema.parse(res)
}

export const updateKafkaTopicGroup = async (projectId, groupId, data) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/topic-groups/${groupId}`,
    method: 'put',
    data,
  })
  return KafkaTopicGroupSchema.parse(res)
}

export const deleteKafkaTopicGroup = (projectId, groupId) =>
  request({
    url: `/data/projects/${projectId}/kafka/topic-groups/${groupId}`,
    method: 'delete',
  })
```

新增映射、预览、字段 API：

```ts
export const getKafkaTopicMappings = async (projectId, connectionId) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/${connectionId}/topic-mappings`,
    method: 'get',
  })
  return KafkaTopicMappingListSchema.parse(res)
}

export const createKafkaTopicMapping = async (projectId, connectionId, data) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/${connectionId}/topic-mappings`,
    method: 'post',
    data,
  })
  return KafkaTopicMappingSchema.parse(res)
}

export const updateKafkaTopicMapping = async (projectId, mappingId, data) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/topic-mappings/${mappingId}`,
    method: 'put',
    data,
  })
  return KafkaTopicMappingSchema.parse(res)
}

export const deleteKafkaTopicMapping = (projectId, mappingId) =>
  request({
    url: `/data/projects/${projectId}/kafka/topic-mappings/${mappingId}`,
    method: 'delete',
  })

export const previewKafkaTopicMapping = async (projectId, mappingId, data = {}) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/topic-mappings/${mappingId}/preview`,
    method: 'post',
    data,
  })
  return KafkaPreviewSchema.parse(res)
}

export const getKafkaFields = async (projectId, mappingId) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/topic-mappings/${mappingId}/fields`,
    method: 'get',
  })
  return KafkaFieldListSchema.parse(res)
}

export const createKafkaField = async (projectId, mappingId, data) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/topic-mappings/${mappingId}/fields`,
    method: 'post',
    data,
  })
  return KafkaFieldSchema.parse(res)
}

export const createKafkaFieldsBatch = async (projectId, mappingId, fields) => {
  const res = await request({
    url: `/data/projects/${projectId}/kafka/topic-mappings/${mappingId}/fields/batch`,
    method: 'post',
    data: { fields },
  })
  return KafkaFieldListSchema.parse(res)
}
```

把这些方法加入 `export default`。

- [ ] **步骤 4：运行类型检查**

运行：

```powershell
cd D:\SVNCode\indu-forge
pnpm --filter datacenter typecheck
```

预期：PASS。

- [ ] **步骤 5：提交前端 API**

运行：

```powershell
git add datacenter/src/api/data.api.ts datacenter/src/api/schemas/kafka-workbench.schema.ts datacenter/src/components/kafka/types.ts
git commit -m "feat(datacenter): 增加 Kafka 工作台 API"
```

---

### 任务 7：实现 Topic 树模型和单测

**文件：**

- 创建：`datacenter/src/components/kafka/kafkaTopicTreeModel.ts`
- 创建：`datacenter/tests/kafka-topic-tree-model.test.ts`

- [ ] **步骤 1：编写失败的树模型测试**

新增 `datacenter/tests/kafka-topic-tree-model.test.ts`：

```ts
import { describe, expect, test } from 'vitest'
import { buildKafkaTopicTree, filterKafkaTopicTree } from '@/components/kafka/kafkaTopicTreeModel'
import type { KafkaTopicGroup, KafkaTopicMapping } from '@/components/kafka/types'

describe('kafkaTopicTreeModel', () => {
  test('按 parentId 组装 Topic 分组树并挂载映射', () => {
    const groups: KafkaTopicGroup[] = [
      { id: 'g1', connectionId: 'c1', parentId: null, name: '产线 A', sortOrder: 0 },
      { id: 'g2', connectionId: 'c1', parentId: 'g1', name: '炉区', sortOrder: 0 },
    ]
    const mappings: KafkaTopicMapping[] = [
      {
        id: 'm1',
        connectionId: 'c1',
        groupId: 'g2',
        name: '温度',
        topic: 'device.temperature',
        description: '',
        partitionMode: 'all',
        startPosition: 'latest',
        decode: 'json',
        sampleLimit: 100,
        timeoutMs: 5000,
        sortOrder: 0,
      },
    ]

    const tree = buildKafkaTopicTree(groups, mappings)
    expect(tree).toHaveLength(1)
    expect(tree[0].children[0].mappings[0].topic).toBe('device.temperature')
  })

  test('按名称和 topic 过滤树', () => {
    const tree = buildKafkaTopicTree([], [
      {
        id: 'm1',
        connectionId: 'c1',
        name: '报警事件',
        topic: 'alarm.events',
        description: '',
        partitionMode: 'all',
        startPosition: 'latest',
        decode: 'json',
        sampleLimit: 100,
        timeoutMs: 5000,
        sortOrder: 0,
      },
    ])
    const result = filterKafkaTopicTree(tree.groups, tree.rootMappings, 'alarm')
    expect(result.rootMappings).toHaveLength(1)
  })
})
```

- [ ] **步骤 2：运行测试确认失败**

运行：

```powershell
pnpm --dir datacenter test -- kafka-topic-tree-model
```

预期：FAIL，模块不存在。

- [ ] **步骤 3：实现树模型**

新增 `datacenter/src/components/kafka/kafkaTopicTreeModel.ts`：

```ts
import type { KafkaTopicGroup, KafkaTopicGroupNode, KafkaTopicMapping } from './types'

export function buildKafkaTopicTree(
  groups: KafkaTopicGroup[],
  mappings: KafkaTopicMapping[],
): { groups: KafkaTopicGroupNode[]; rootMappings: KafkaTopicMapping[] } {
  const nodes = new Map<string, KafkaTopicGroupNode>()
  groups.forEach((group) => {
    nodes.set(group.id, { ...group, children: [], mappings: [] })
  })

  const roots: KafkaTopicGroupNode[] = []
  nodes.forEach((node) => {
    const parentId = node.parentId || ''
    const parent = parentId ? nodes.get(parentId) : null
    if (parent) {
      parent.children.push(node)
    } else {
      roots.push(node)
    }
  })

  const rootMappings: KafkaTopicMapping[] = []
  mappings.forEach((mapping) => {
    const groupId = mapping.groupId || ''
    const group = groupId ? nodes.get(groupId) : null
    if (group) {
      group.mappings.push(mapping)
    } else {
      rootMappings.push(mapping)
    }
  })

  const sortMappings = (items: KafkaTopicMapping[]) =>
    items.sort((left, right) => (left.sortOrder || 0) - (right.sortOrder || 0) || left.name.localeCompare(right.name))

  const sortNodes = (items: KafkaTopicGroupNode[]) => {
    items.sort((left, right) => (left.sortOrder || 0) - (right.sortOrder || 0) || left.name.localeCompare(right.name))
    items.forEach((item) => {
      sortNodes(item.children)
      sortMappings(item.mappings)
    })
  }

  sortNodes(roots)
  sortMappings(rootMappings)
  return { groups: roots, rootMappings }
}

export function filterKafkaTopicTree(
  groups: KafkaTopicGroupNode[],
  rootMappings: KafkaTopicMapping[],
  keyword: string,
): { groups: KafkaTopicGroupNode[]; rootMappings: KafkaTopicMapping[] } {
  const normalized = keyword.trim().toLowerCase()
  if (!normalized) return { groups, rootMappings }

  const mappingMatches = (mapping: KafkaTopicMapping) =>
    `${mapping.name} ${mapping.topic}`.toLowerCase().includes(normalized)

  const filterNode = (node: KafkaTopicGroupNode): KafkaTopicGroupNode | null => {
    const children = node.children.map(filterNode).filter(Boolean) as KafkaTopicGroupNode[]
    const mappings = node.mappings.filter(mappingMatches)
    const selfMatches = node.name.toLowerCase().includes(normalized)
    if (selfMatches || children.length > 0 || mappings.length > 0) {
      return { ...node, children, mappings: selfMatches ? node.mappings : mappings }
    }
    return null
  }

  return {
    groups: groups.map(filterNode).filter(Boolean) as KafkaTopicGroupNode[],
    rootMappings: rootMappings.filter(mappingMatches),
  }
}
```

- [ ] **步骤 4：运行测试验证通过**

运行：

```powershell
pnpm --dir datacenter test -- kafka-topic-tree-model
```

预期：PASS。

- [ ] **步骤 5：提交树模型**

运行：

```powershell
git add datacenter/src/components/kafka/kafkaTopicTreeModel.ts datacenter/tests/kafka-topic-tree-model.test.ts
git commit -m "feat(datacenter): 增加 Kafka Topic 树模型"
```

---

### 任务 8：实现 Kafka 工作台入口和容器

**文件：**

- 创建：`datacenter/src/components/access-source/workbench/KafkaWorkbenchPanel.vue`
- 创建：`datacenter/src/components/kafka/KafkaWorkbench.vue`
- 创建：`datacenter/src/components/kafka/KafkaTopicTreeBranch.vue`
- 创建：`datacenter/src/components/kafka/KafkaInspectorPanel.vue`
- 修改：`datacenter/src/components/access-source/AccessSourceWorkbench.vue`

- [ ] **步骤 1：创建工作台入口壳**

新增 `datacenter/src/components/access-source/workbench/KafkaWorkbenchPanel.vue`：

```vue
<template>
  <div class="aws-kafka">
    <KafkaWorkbench :project-id="projectId" :connection="connection" @back="$emit('back')" />
  </div>
</template>

<script setup lang="ts">
import KafkaWorkbench from '@/components/kafka/KafkaWorkbench.vue'

defineProps<{
  connection: Record<string, any>
  projectId: string
}>()

defineEmits<{
  (event: 'back'): void
}>()
</script>

<style scoped>
.aws-kafka {
  height: 100%;
  min-height: 0;
  overflow: hidden;
}
</style>
```

- [ ] **步骤 2：创建 KafkaWorkbench 骨架**

新增 `datacenter/src/components/kafka/KafkaWorkbench.vue`，先实现布局和加载：

```vue
<template>
  <section class="kafka-workbench">
    <aside class="kafka-workbench__explorer">
      <WorkbenchSourceHeader
        :title="connection.name || '未命名 Kafka 接入源'"
        fallback-title="未命名 Kafka 接入源"
        :meta="sourceMetaRows"
        @back="$emit('back')"
      >
        <template #status>
          <button
            type="button"
            class="kafka-workbench__connect-action"
            :class="{ 'is-connected': connected }"
            @click="toggleConnection"
          >
            <span>{{ connected ? '已连接' : '连接' }}</span>
          </button>
        </template>
        <template #actions>
          <el-input v-model="filterText" size="small" clearable placeholder="筛选 Topic" />
          <button type="button" class="workbench-source-header__icon-action is-primary" title="新建 Topic 映射" @click="openCreateMapping">
            <IconTablerPlus />
          </button>
          <button type="button" class="workbench-source-header__icon-action" title="新建分组" @click="openCreateGroup">
            <IconTablerFolderPlus />
          </button>
          <button type="button" class="workbench-source-header__icon-action" title="刷新" @click="loadWorkbench">
            <IconTablerRefresh />
          </button>
        </template>
      </WorkbenchSourceHeader>

      <div class="kafka-workbench__tree">
        <KafkaTopicTreeBranch
          v-for="group in filteredTree.groups"
          :key="group.id"
          :node="group"
          :selected-mapping-id="selectedMapping?.id"
          @select-mapping="openPreview"
          @open-fields="openFields"
        />
        <button
          v-for="mapping in filteredTree.rootMappings"
          :key="mapping.id"
          type="button"
          class="kafka-workbench__tree-item"
          :class="{ 'is-active': selectedMapping?.id === mapping.id }"
          @click="openPreview(mapping)"
        >
          <IconTablerMessages />
          <span>{{ mapping.name || mapping.topic }}</span>
        </button>
      </div>
    </aside>

    <main class="kafka-workbench__main">
      <div class="kafka-workbench__tabbar">
        <button v-for="tab in tabs" :key="tab.id" type="button" class="kafka-workbench__tab" :class="{ 'is-active': tab.id === activeTabId }" @click="activeTabId = tab.id">
          <component :is="tab.icon" />
          <span>{{ tab.title }}</span>
          <IconTablerX v-if="tab.closable" @click.stop="closeTab(tab.id)" />
        </button>
      </div>
      <div class="kafka-workbench__content">
        <KafkaPreviewPanel
          v-if="activeTab?.type === 'preview'"
          :project-id="projectId"
          :mapping="activeTab.mapping"
          @samples="handlePreviewSamples"
        />
        <KafkaFieldMappingPanel
          v-else-if="activeTab?.type === 'fields'"
          :project-id="projectId"
          :mapping="activeTab.mapping"
          :samples="previewSamplesByMapping.get(activeTab.mapping.id) || []"
        />
        <div v-else class="kafka-workbench__placeholder">
          <IconTablerMessages />
          <strong>选择 Topic 开始预览</strong>
          <span>从左侧 Topic 映射进入消息预览或字段映射。</span>
        </div>
      </div>
    </main>

    <KafkaInspectorPanel :connection="connection" :mapping="selectedMapping" :preview="activePreview" />
  </section>
</template>
```

脚本核心：

```ts
import { computed, markRaw, onMounted, ref, shallowRef, watch } from 'vue'
import { ElMessage } from 'element-plus'
import IconTablerFolderPlus from '~icons/tabler/folder-plus'
import IconTablerMessages from '~icons/tabler/messages'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerX from '~icons/tabler/x'
import dataAPI from '@/api/data.api'
import WorkbenchSourceHeader from '@/components/workbench/WorkbenchSourceHeader.vue'
import { getApiErrorMessage } from '@/utils/request'
import { buildKafkaTopicTree, filterKafkaTopicTree } from './kafkaTopicTreeModel'
import KafkaFieldMappingPanel from './KafkaFieldMappingPanel.vue'
import KafkaInspectorPanel from './KafkaInspectorPanel.vue'
import KafkaPreviewPanel from './KafkaPreviewPanel.vue'
import KafkaTopicTreeBranch from './KafkaTopicTreeBranch.vue'
import type { KafkaPreview, KafkaPreviewSample, KafkaTopicGroup, KafkaTopicMapping, KafkaWorkbenchConnection } from './types'

const props = defineProps<{
  projectId: string
  connection: KafkaWorkbenchConnection
}>()

defineEmits<{ (event: 'back'): void }>()

const groups = ref<KafkaTopicGroup[]>([])
const mappings = ref<KafkaTopicMapping[]>([])
const filterText = ref('')
const connected = ref(false)
const selectedMapping = ref<KafkaTopicMapping | null>(null)
const activeTabId = ref('')
const tabs = ref<any[]>([])
const previewSamplesByMapping = shallowRef(new Map<string, KafkaPreviewSample[]>())
const activePreview = ref<KafkaPreview | null>(null)

const config = computed(() => props.connection.config || {})
const sourceMetaRows = computed(() => [
  { label: '类型', value: 'Kafka' },
  { label: 'Brokers', value: String(config.value.brokers || '未配置') },
])
const tree = computed(() => buildKafkaTopicTree(groups.value, mappings.value))
const filteredTree = computed(() => filterKafkaTopicTree(tree.value.groups, tree.value.rootMappings, filterText.value))
const activeTab = computed(() => tabs.value.find((tab) => tab.id === activeTabId.value) || null)

const loadWorkbench = async () => {
  try {
    const [groupRes, mappingRes] = await Promise.all([
      dataAPI.getKafkaTopicGroups(props.projectId, props.connection.id),
      dataAPI.getKafkaTopicMappings(props.projectId, props.connection.id),
    ])
    groups.value = groupRes.list || []
    mappings.value = mappingRes.list || []
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载 Kafka 工作台失败'))
  }
}

const toggleConnection = async () => {
  connected.value = !connected.value
}

const openPreview = (mapping: KafkaTopicMapping) => {
  selectedMapping.value = mapping
  const id = `kafka-preview-${mapping.id}`
  if (!tabs.value.some((tab) => tab.id === id)) {
    tabs.value.push({ id, type: 'preview', title: `${mapping.name} / 消息预览`, icon: markRaw(IconTablerMessages), mapping, closable: true })
  }
  activeTabId.value = id
}

const openFields = (mapping: KafkaTopicMapping) => {
  selectedMapping.value = mapping
  const id = `kafka-fields-${mapping.id}`
  if (!tabs.value.some((tab) => tab.id === id)) {
    tabs.value.push({ id, type: 'fields', title: `${mapping.name} / 字段映射`, icon: markRaw(IconTablerMessages), mapping, closable: true })
  }
  activeTabId.value = id
}

const closeTab = (tabId: string) => {
  tabs.value = tabs.value.filter((tab) => tab.id !== tabId)
  if (activeTabId.value === tabId) {
    activeTabId.value = tabs.value[0]?.id || ''
  }
}

const handlePreviewSamples = (payload: { mappingId: string; samples: KafkaPreviewSample[]; preview: KafkaPreview }) => {
  const next = new Map(previewSamplesByMapping.value)
  next.set(payload.mappingId, payload.samples)
  previewSamplesByMapping.value = next
  activePreview.value = payload.preview
}

onMounted(loadWorkbench)
watch(() => props.connection.id, loadWorkbench)
```

- [ ] **步骤 3：创建 Topic 树节点组件**

新增 `datacenter/src/components/kafka/KafkaTopicTreeBranch.vue`：

```vue
<template>
  <section class="kafka-topic-branch">
    <button type="button" class="kafka-topic-branch__group">
      <IconTablerFolder />
      <span>{{ node.name }}</span>
      <small>{{ node.mappings.length }}</small>
    </button>
    <div class="kafka-topic-branch__children">
      <KafkaTopicTreeBranch
        v-for="child in node.children"
        :key="child.id"
        :node="child"
        :selected-mapping-id="selectedMappingId"
        @select-mapping="$emit('select-mapping', $event)"
        @open-fields="$emit('open-fields', $event)"
      />
      <button
        v-for="mapping in node.mappings"
        :key="mapping.id"
        type="button"
        class="kafka-topic-branch__mapping"
        :class="{ 'is-active': selectedMappingId === mapping.id }"
        @click="$emit('select-mapping', mapping)"
        @dblclick="$emit('open-fields', mapping)"
      >
        <IconTablerMessages />
        <span>{{ mapping.name || mapping.topic }}</span>
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import IconTablerFolder from '~icons/tabler/folder'
import IconTablerMessages from '~icons/tabler/messages'
import type { KafkaTopicGroupNode, KafkaTopicMapping } from './types'

defineProps<{
  node: KafkaTopicGroupNode
  selectedMappingId?: string
}>()

defineEmits<{
  (event: 'select-mapping', mapping: KafkaTopicMapping): void
  (event: 'open-fields', mapping: KafkaTopicMapping): void
}>()
</script>
```

- [ ] **步骤 4：创建右侧 Inspector**

新增 `datacenter/src/components/kafka/KafkaInspectorPanel.vue`：

```vue
<template>
  <aside class="kafka-inspector">
    <section>
      <h3>接入源摘要</h3>
      <dl>
        <dt>Brokers</dt>
        <dd>{{ brokers }}</dd>
        <dt>安全协议</dt>
        <dd>{{ securityProtocol }}</dd>
      </dl>
    </section>
    <section v-if="mapping">
      <h3>当前 Topic</h3>
      <dl>
        <dt>名称</dt>
        <dd>{{ mapping.name }}</dd>
        <dt>Topic</dt>
        <dd>{{ mapping.topic }}</dd>
        <dt>Offset</dt>
        <dd>{{ mapping.startPosition }}</dd>
      </dl>
    </section>
    <section v-if="preview">
      <h3>最近预览</h3>
      <dl>
        <dt>状态</dt>
        <dd>{{ preview.status }}</dd>
        <dt>样本数</dt>
        <dd>{{ preview.samples.length }}</dd>
      </dl>
    </section>
  </aside>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { KafkaPreview, KafkaTopicMapping, KafkaWorkbenchConnection } from './types'

const props = defineProps<{
  connection: KafkaWorkbenchConnection
  mapping: KafkaTopicMapping | null
  preview: KafkaPreview | null
}>()

const brokers = computed(() => String(props.connection.config?.brokers || '未配置'))
const securityProtocol = computed(() => String(props.connection.config?.securityProtocol || 'PLAINTEXT'))
</script>
```

- [ ] **步骤 5：接入 AccessSourceWorkbench**

修改 `datacenter/src/components/access-source/AccessSourceWorkbench.vue`：

```ts
import KafkaWorkbenchPanel from './workbench/KafkaWorkbenchPanel.vue'
```

把 Kafka 分支从通用协议中移出：

```ts
if (type === 'kafka') {
  return KafkaWorkbenchPanel
}
if (['http', 'websocket'].includes(type)) {
  return ProtocolWorkbenchPanel
}
```

- [ ] **步骤 6：补齐工作台 CSS**

在 `KafkaWorkbench.vue` 使用与 MQTT 工作台一致的结构样式：

```css
.kafka-workbench {
  height: 100%;
  min-height: 0;
  display: grid;
  grid-template-columns: 260px minmax(0, 1fr) 280px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
  overflow: hidden;
}

.kafka-workbench__explorer {
  min-height: 0;
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--dc-border);
  background: var(--dc-surface-muted);
}

.kafka-workbench__main {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.kafka-workbench__tabbar {
  display: flex;
  min-height: 38px;
  overflow-x: auto;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
}

.kafka-workbench__content {
  min-height: 0;
  flex: 1;
  overflow: hidden;
}
```

- [ ] **步骤 7：运行前端类型检查**

运行：

```powershell
pnpm --filter datacenter typecheck
```

预期：PASS。

- [ ] **步骤 8：提交工作台容器**

运行：

```powershell
git add datacenter/src/components/access-source/workbench/KafkaWorkbenchPanel.vue datacenter/src/components/access-source/AccessSourceWorkbench.vue datacenter/src/components/kafka/KafkaWorkbench.vue datacenter/src/components/kafka/KafkaTopicTreeBranch.vue datacenter/src/components/kafka/KafkaInspectorPanel.vue
git commit -m "feat(datacenter): 增加 Kafka 工作台容器"
```

---

### 任务 9：实现 Topic 分组和 Topic 映射弹窗

**文件：**

- 创建：`datacenter/src/components/kafka/KafkaTopicMappingDialog.vue`
- 创建：`datacenter/src/components/kafka/KafkaTopicGroupDialog.vue`
- 修改：`datacenter/src/components/kafka/KafkaWorkbench.vue`

- [ ] **步骤 1：创建分组弹窗**

新增 `KafkaTopicGroupDialog.vue`：

```vue
<template>
  <DcDialog v-model="visible" :title="mode === 'create' ? '新建 Topic 分组' : '编辑 Topic 分组'" width="420px">
    <el-form label-position="top">
      <el-form-item label="分组名称" required>
        <el-input v-model="form.name" maxlength="100" show-word-limit />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" :loading="loading" :disabled="!form.name.trim()" @click="submit">保存</el-button>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import DcDialog from '@/components/shared/DcDialog.vue'
import type { KafkaTopicGroup } from './types'

const props = defineProps<{
  modelValue: boolean
  mode: 'create' | 'edit'
  group?: KafkaTopicGroup | null
  loading?: boolean
}>()

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'submit', value: { name: string }): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})
const form = reactive({ name: '' })

watch(
  () => [props.modelValue, props.group] as const,
  () => {
    form.name = props.group?.name || ''
  },
  { immediate: true },
)

const submit = () => emit('submit', { name: form.name.trim() })
</script>
```

- [ ] **步骤 2：创建 Topic 映射弹窗**

新增 `KafkaTopicMappingDialog.vue`：

```vue
<template>
  <DcDialog v-model="visible" :title="mode === 'create' ? '新建 Topic 映射' : '编辑 Topic 映射'" width="640px">
    <el-form label-position="top" class="kafka-topic-dialog">
      <el-form-item label="显示名称" required>
        <el-input v-model="form.name" maxlength="100" show-word-limit />
      </el-form-item>
      <el-form-item label="Topic" required>
        <el-input v-model="form.topic" placeholder="device.telemetry" />
      </el-form-item>
      <el-form-item label="分区策略">
        <el-segmented v-model="form.partitionMode" :options="partitionModeOptions" />
      </el-form-item>
      <el-form-item v-if="form.partitionMode === 'single'" label="Partition" required>
        <el-input-number v-model="form.partition" :min="0" />
      </el-form-item>
      <el-form-item label="Offset 模式">
        <el-select v-model="form.startPosition">
          <el-option label="latest" value="latest" />
          <el-option label="earliest" value="earliest" />
          <el-option label="指定 offset" value="offset" />
        </el-select>
      </el-form-item>
      <el-form-item label="解码方式">
        <el-select v-model="form.decode">
          <el-option label="JSON" value="json" />
          <el-option label="String" value="string" />
          <el-option label="Binary" value="binary" />
        </el-select>
      </el-form-item>
      <el-form-item label="样本数">
        <el-input-number v-model="form.sampleLimit" :min="1" :max="1000" />
      </el-form-item>
      <el-form-item label="超时时间 ms">
        <el-input-number v-model="form.timeoutMs" :min="1000" :max="30000" :step="1000" />
      </el-form-item>
      <el-form-item label="描述">
        <el-input v-model="form.description" type="textarea" :rows="3" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" :loading="loading" :disabled="!canSubmit" @click="submit">保存</el-button>
    </template>
  </DcDialog>
</template>
```

脚本：

```ts
import { computed, reactive, watch } from 'vue'
import DcDialog from '@/components/shared/DcDialog.vue'
import type { KafkaTopicMapping } from './types'

const props = defineProps<{
  modelValue: boolean
  mode: 'create' | 'edit'
  mapping?: KafkaTopicMapping | null
  loading?: boolean
}>()

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'submit', value: Record<string, unknown>): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})
const partitionModeOptions = [
  { label: '全部', value: 'all' },
  { label: '单分区', value: 'single' },
]
const form = reactive({
  name: '',
  topic: '',
  partitionMode: 'all',
  partition: 0,
  startPosition: 'latest',
  decode: 'json',
  sampleLimit: 100,
  timeoutMs: 5000,
  description: '',
})

const canSubmit = computed(() => {
  if (!form.name.trim() || !form.topic.trim()) return false
  if (form.partitionMode === 'single' && form.partition < 0) return false
  return true
})

watch(
  () => [props.modelValue, props.mapping] as const,
  () => {
    form.name = props.mapping?.name || ''
    form.topic = props.mapping?.topic || ''
    form.partitionMode = props.mapping?.partitionMode || 'all'
    form.partition = props.mapping?.partition ?? 0
    form.startPosition = props.mapping?.startPosition || 'latest'
    form.decode = props.mapping?.decode || 'json'
    form.sampleLimit = props.mapping?.sampleLimit || 100
    form.timeoutMs = props.mapping?.timeoutMs || 5000
    form.description = props.mapping?.description || ''
  },
  { immediate: true },
)

const submit = () => {
  emit('submit', {
    ...form,
    name: form.name.trim(),
    topic: form.topic.trim(),
    partition: form.partitionMode === 'single' ? form.partition : null,
  })
}
```

- [ ] **步骤 3：串联弹窗到 KafkaWorkbench**

在 `KafkaWorkbench.vue` 引入弹窗并添加状态：

```ts
import KafkaTopicGroupDialog from './KafkaTopicGroupDialog.vue'
import KafkaTopicMappingDialog from './KafkaTopicMappingDialog.vue'

const groupDialogVisible = ref(false)
const groupSaving = ref(false)
const mappingDialogVisible = ref(false)
const mappingSaving = ref(false)
const mappingDialogMode = ref<'create' | 'edit'>('create')
const editingMapping = ref<KafkaTopicMapping | null>(null)
```

模板底部添加：

```vue
<KafkaTopicGroupDialog
  v-model="groupDialogVisible"
  mode="create"
  :loading="groupSaving"
  @submit="saveGroup"
/>
<KafkaTopicMappingDialog
  v-model="mappingDialogVisible"
  :mode="mappingDialogMode"
  :mapping="editingMapping"
  :loading="mappingSaving"
  @submit="saveMapping"
/>
```

实现保存：

```ts
const openCreateGroup = () => {
  groupDialogVisible.value = true
}

const saveGroup = async (payload: { name: string }) => {
  groupSaving.value = true
  try {
    await dataAPI.createKafkaTopicGroup(props.projectId, props.connection.id, payload)
    groupDialogVisible.value = false
    await loadWorkbench()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '保存 Topic 分组失败'))
  } finally {
    groupSaving.value = false
  }
}

const openCreateMapping = () => {
  mappingDialogMode.value = 'create'
  editingMapping.value = null
  mappingDialogVisible.value = true
}

const saveMapping = async (payload: Record<string, unknown>) => {
  mappingSaving.value = true
  try {
    if (mappingDialogMode.value === 'edit' && editingMapping.value) {
      await dataAPI.updateKafkaTopicMapping(props.projectId, editingMapping.value.id, payload)
    } else {
      await dataAPI.createKafkaTopicMapping(props.projectId, props.connection.id, payload)
    }
    mappingDialogVisible.value = false
    await loadWorkbench()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '保存 Topic 映射失败'))
  } finally {
    mappingSaving.value = false
  }
}
```

- [ ] **步骤 4：运行类型和构建**

运行：

```powershell
pnpm --filter datacenter typecheck
pnpm --filter datacenter build
```

预期：PASS。

- [ ] **步骤 5：提交弹窗**

运行：

```powershell
git add datacenter/src/components/kafka/KafkaTopicGroupDialog.vue datacenter/src/components/kafka/KafkaTopicMappingDialog.vue datacenter/src/components/kafka/KafkaWorkbench.vue
git commit -m "feat(datacenter): 实现 Kafka Topic 管理弹窗"
```

---

### 任务 10：实现 Kafka 消息预览面板

**文件：**

- 创建：`datacenter/src/components/kafka/KafkaPreviewPanel.vue`
- 修改：`datacenter/src/components/kafka/KafkaWorkbench.vue`

- [ ] **步骤 1：创建预览面板**

新增 `KafkaPreviewPanel.vue`：

```vue
<template>
  <section class="kafka-preview-panel">
    <WorkbenchStreamToolbar
      :title="`${mapping.name} 消息预览`"
      :subtitle="mapping.topic"
      :loading="loading"
      :status-label="statusLabel"
      :status-tone="statusTone"
      v-model:search="search"
      v-model:limit="displayLimit"
      v-model:format-json="formatJson"
      @refresh="runPreview"
      @clear="clearSamples"
    />
    <WorkbenchStreamMessageList
      :messages="displayMessages"
      :loading="loading"
      empty-text="暂无 Kafka 样本"
      empty-hint="点击获取样本执行一次短时预览"
      @copy="copyMessage"
    />
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import dataAPI from '@/api/data.api'
import WorkbenchStreamMessageList from '@/components/workbench/WorkbenchStreamMessageList.vue'
import WorkbenchStreamToolbar from '@/components/workbench/WorkbenchStreamToolbar.vue'
import { getApiErrorMessage } from '@/utils/request'
import type { KafkaPreview, KafkaPreviewSample, KafkaTopicMapping } from './types'

const props = defineProps<{
  projectId: string
  mapping: KafkaTopicMapping
}>()

const emit = defineEmits<{
  (event: 'samples', payload: { mappingId: string; samples: KafkaPreviewSample[]; preview: KafkaPreview }): void
}>()

const loading = ref(false)
const search = ref('')
const displayLimit = ref(100)
const formatJson = ref(true)
const samples = ref<KafkaPreviewSample[]>([])
const lastPreview = ref<KafkaPreview | null>(null)
const lastError = ref('')

const statusLabel = computed(() => {
  if (loading.value) return '预览中'
  if (lastError.value) return '预览失败'
  if (samples.value.length > 0) return `样本 ${samples.value.length}`
  return '待预览'
})
const statusTone = computed(() => {
  if (lastError.value) return 'danger'
  if (samples.value.length > 0) return 'success'
  return 'info'
})
const displayMessages = computed(() => {
  const keyword = search.value.trim().toLowerCase()
  return samples.value
    .filter((sample) => {
      if (!keyword) return true
      return `${sample.topic || ''} ${sample.key || ''} ${formatPayload(sample.value)}`.toLowerCase().includes(keyword)
    })
    .slice(0, displayLimit.value)
    .map((sample) => ({
      id: `${sample.topic}-${sample.partition}-${sample.offset}`,
      topic: `${sample.topic || props.mapping.topic} / p${sample.partition ?? '-'}`,
      payload: sample.value,
      payloadText: formatPayload(sample.value),
      timeText: sample.timestamp || '',
      qos: 0,
      raw: sample,
    }))
})

const runPreview = async () => {
  loading.value = true
  lastError.value = ''
  try {
    const preview = await dataAPI.previewKafkaTopicMapping(props.projectId, props.mapping.id, {
      limit: displayLimit.value,
      timeoutMs: props.mapping.timeoutMs,
      decode: props.mapping.decode,
    })
    lastPreview.value = preview
    samples.value = preview.samples || []
    emit('samples', { mappingId: props.mapping.id, samples: samples.value, preview })
  } catch (error) {
    lastError.value = getApiErrorMessage(error, 'Kafka 预览失败')
    ElMessage.error(lastError.value)
  } finally {
    loading.value = false
  }
}

const clearSamples = () => {
  samples.value = []
  lastPreview.value = null
  lastError.value = ''
}

const copyMessage = async (message: any) => {
  await navigator.clipboard.writeText(message.payloadText || '')
  ElMessage.success('Payload 已复制')
}

const formatPayload = (value: unknown) => {
  if (typeof value === 'string') return value
  try {
    return formatJson.value ? JSON.stringify(value, null, 2) : JSON.stringify(value)
  } catch {
    return String(value)
  }
}

onMounted(runPreview)
watch(() => props.mapping.id, runPreview)
</script>
```

- [ ] **步骤 2：补齐样式**

在同文件添加：

```css
<style scoped>
.kafka-preview-panel {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--dc-surface-raised);
}
</style>
```

- [ ] **步骤 3：确认工具栏 prop 对齐**

如果当前 `WorkbenchStreamToolbar.vue` 的 prop 名与本计划不同，按已有组件实际接口最小调整 `KafkaPreviewPanel.vue`，保持这些能力不变：

```text
标题、副标题、状态、搜索、显示数量、JSON 格式化、刷新、清空、loading
```

- [ ] **步骤 4：运行构建**

运行：

```powershell
pnpm --filter datacenter build
```

预期：PASS。

- [ ] **步骤 5：提交预览面板**

运行：

```powershell
git add datacenter/src/components/kafka/KafkaPreviewPanel.vue datacenter/src/components/kafka/KafkaWorkbench.vue
git commit -m "feat(datacenter): 实现 Kafka 消息预览面板"
```

---

### 任务 11：实现 Kafka 字段映射面板

**文件：**

- 创建：`datacenter/src/components/kafka/KafkaFieldMappingPanel.vue`

- [ ] **步骤 1：创建字段候选推断函数**

在 `KafkaFieldMappingPanel.vue` script 中定义：

```ts
type CandidateField = {
  path: string
  dataType: string
  selected: boolean
  exists: boolean
}

const inferCandidateFields = (samples: KafkaPreviewSample[], existing: KafkaField[]) => {
  const seen = new Map<string, CandidateField>()
  const existingPaths = new Set(existing.map((field) => field.valuePath))
  const visit = (prefix: string, value: unknown) => {
    if (value && typeof value === 'object' && !Array.isArray(value)) {
      Object.entries(value as Record<string, unknown>).forEach(([key, child]) => {
        visit(prefix ? `${prefix}.${key}` : key, child)
      })
      return
    }
    if (!prefix || seen.has(prefix)) return
    seen.set(prefix, {
      path: prefix,
      dataType: resolveDataType(value),
      selected: false,
      exists: existingPaths.has(prefix),
    })
  }
  samples.forEach((sample) => visit('', sample.value))
  return Array.from(seen.values())
}

const resolveDataType = (value: unknown) => {
  if (typeof value === 'number') return 'number'
  if (typeof value === 'boolean') return 'boolean'
  if (typeof value === 'string') return 'string'
  return 'json'
}
```

- [ ] **步骤 2：创建字段映射面板模板**

新增 `KafkaFieldMappingPanel.vue`：

```vue
<template>
  <section class="kafka-field-panel">
    <header class="kafka-field-panel__toolbar">
      <div>
        <strong>{{ mapping.name }} 字段映射</strong>
        <span>{{ mapping.topic }}</span>
      </div>
      <el-button type="primary" size="small" :disabled="selectedCandidates.length === 0" :loading="saving" @click="createSelectedFields">
        创建数据点
      </el-button>
    </header>

    <div class="kafka-field-panel__body">
      <section class="kafka-field-panel__section">
        <h3>样本字段</h3>
        <el-table :data="candidates" height="100%">
          <el-table-column width="46">
            <template #default="{ row }">
              <el-checkbox v-model="row.selected" :disabled="row.exists" />
            </template>
          </el-table-column>
          <el-table-column prop="path" label="字段路径" min-width="180" />
          <el-table-column prop="dataType" label="类型" width="100" />
          <el-table-column label="状态" width="110">
            <template #default="{ row }">
              <WorkbenchStatusPill :tone="row.exists ? 'success' : 'info'">
                {{ row.exists ? '已映射' : '候选' }}
              </WorkbenchStatusPill>
            </template>
          </el-table-column>
        </el-table>
      </section>

      <section class="kafka-field-panel__section">
        <h3>已建字段</h3>
        <el-table :data="fields" height="100%">
          <el-table-column prop="name" label="名称" />
          <el-table-column prop="valuePath" label="字段路径" />
          <el-table-column prop="dataType" label="类型" width="90" />
          <el-table-column prop="dataPointPath" label="数据点" min-width="180" />
        </el-table>
      </section>
    </div>
  </section>
</template>
```

- [ ] **步骤 3：实现加载和批量创建**

脚本：

```ts
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import dataAPI from '@/api/data.api'
import WorkbenchStatusPill from '@/components/workbench/WorkbenchStatusPill.vue'
import { getApiErrorMessage } from '@/utils/request'
import type { KafkaField, KafkaPreviewSample, KafkaTopicMapping } from './types'

const props = defineProps<{
  projectId: string
  mapping: KafkaTopicMapping
  samples: KafkaPreviewSample[]
}>()

const fields = ref<KafkaField[]>([])
const candidates = ref<CandidateField[]>([])
const saving = ref(false)
const selectedCandidates = computed(() => candidates.value.filter((candidate) => candidate.selected && !candidate.exists))

const loadFields = async () => {
  try {
    const res = await dataAPI.getKafkaFields(props.projectId, props.mapping.id)
    fields.value = res.list || []
    candidates.value = inferCandidateFields(props.samples, fields.value)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载 Kafka 字段失败'))
  }
}

const createSelectedFields = async () => {
  saving.value = true
  try {
    await dataAPI.createKafkaFieldsBatch(
      props.projectId,
      props.mapping.id,
      selectedCandidates.value.map((candidate) => ({
        name: candidate.path.split('.').pop() || candidate.path,
        valuePath: candidate.path,
        dataType: candidate.dataType,
        enabled: true,
      })),
    )
    ElMessage.success('字段映射已创建')
    await loadFields()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '创建字段映射失败'))
  } finally {
    saving.value = false
  }
}

onMounted(loadFields)
watch(() => [props.mapping.id, props.samples.length] as const, loadFields)
```

- [ ] **步骤 4：补齐样式**

添加：

```css
<style scoped>
.kafka-field-panel {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--dc-surface-raised);
}

.kafka-field-panel__toolbar {
  min-height: 48px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
}

.kafka-field-panel__toolbar div {
  min-width: 0;
  display: grid;
  gap: 3px;
}

.kafka-field-panel__toolbar strong,
.kafka-field-panel__toolbar span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.kafka-field-panel__body {
  min-height: 0;
  flex: 1;
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 0;
}

.kafka-field-panel__section {
  min-height: 0;
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--dc-border);
}

.kafka-field-panel__section h3 {
  margin: 0;
  padding: 10px 12px;
  color: var(--dc-text);
  font-size: 12px;
}
</style>
```

- [ ] **步骤 5：运行构建**

运行：

```powershell
pnpm --filter datacenter build
```

预期：PASS。

- [ ] **步骤 6：提交字段映射面板**

运行：

```powershell
git add datacenter/src/components/kafka/KafkaFieldMappingPanel.vue
git commit -m "feat(datacenter): 实现 Kafka 字段映射面板"
```

---

### 任务 12：全量验证和文档同步

**文件：**

- 修改：`docs/统一REST接口规范与清单.md`
- 涉及前面所有 Kafka 文件

- [ ] **步骤 1：更新 REST 接口清单**

在 `docs/统一REST接口规范与清单.md` 的 Kafka 小节补充：

```markdown
| `GET`    | `/api/v1/data/projects/{projectId}/kafka/{connectionId}/topic-groups`                    | 获取 Kafka Topic 分组 |
| `POST`   | `/api/v1/data/projects/{projectId}/kafka/{connectionId}/topic-groups`                    | 创建 Kafka Topic 分组 |
| `PUT`    | `/api/v1/data/projects/{projectId}/kafka/topic-groups/{groupId}`                         | 更新 Kafka Topic 分组 |
| `DELETE` | `/api/v1/data/projects/{projectId}/kafka/topic-groups/{groupId}`                         | 删除 Kafka Topic 分组 |
| `GET`    | `/api/v1/data/projects/{projectId}/kafka/{connectionId}/topic-mappings`                  | 获取 Kafka Topic 映射 |
| `POST`   | `/api/v1/data/projects/{projectId}/kafka/{connectionId}/topic-mappings`                  | 创建 Kafka Topic 映射 |
| `PUT`    | `/api/v1/data/projects/{projectId}/kafka/topic-mappings/{mappingId}`                     | 更新 Kafka Topic 映射 |
| `DELETE` | `/api/v1/data/projects/{projectId}/kafka/topic-mappings/{mappingId}`                     | 删除 Kafka Topic 映射 |
| `POST`   | `/api/v1/data/projects/{projectId}/kafka/topic-mappings/{mappingId}/preview`             | 短时预览 Kafka Topic |
| `GET`    | `/api/v1/data/projects/{projectId}/kafka/topic-mappings/{mappingId}/fields`              | 获取 Kafka 字段映射 |
| `POST`   | `/api/v1/data/projects/{projectId}/kafka/topic-mappings/{mappingId}/fields`              | 创建 Kafka 字段映射 |
| `POST`   | `/api/v1/data/projects/{projectId}/kafka/topic-mappings/{mappingId}/fields/batch`        | 批量创建 Kafka 字段映射 |
| `PUT`    | `/api/v1/data/projects/{projectId}/kafka/fields/{fieldId}`                               | 更新 Kafka 字段映射 |
| `DELETE` | `/api/v1/data/projects/{projectId}/kafka/fields/{fieldId}`                               | 删除 Kafka 字段映射 |
| `PATCH`  | `/api/v1/data/projects/{projectId}/kafka/fields/{fieldId}/toggle`                        | 启停 Kafka 字段映射 |
```

- [ ] **步骤 2：运行后端全量测试**

运行：

```powershell
cd D:\SVNCode\indu-forge\data_service
go test ./...
```

预期：PASS。

- [ ] **步骤 3：运行前端单测、类型和构建**

运行：

```powershell
cd D:\SVNCode\indu-forge
pnpm --dir datacenter test -- kafka-topic-tree-model
pnpm --filter datacenter typecheck
pnpm --filter datacenter build
```

预期：PASS。若构建只出现 chunk size 警告，记录但不作为失败。

- [ ] **步骤 4：运行工作区基础校验**

运行：

```powershell
pnpm lint
pnpm typecheck
```

预期：PASS。若仓库既有无关 lint 错误阻塞，记录具体文件和错误，不修改无关文件。

- [ ] **步骤 5：手动验收路径**

验收：

```text
创建 Kafka 接入源后进入 Kafka 工作台。
左侧 Header 显示 Kafka 名称、Brokers 和连接按钮。
点击新建分组后左侧 Topic 树出现分组。
点击新建 Topic 映射，保存 device.telemetry 后左侧树出现 Topic。
打开 Topic 消息预览，后端无真实 Kafka 时展示明确错误诊断，不启动长期 consumer。
接入可用 Kafka 后，消息流展示 key、value、headers、partition、offset、timestamp。
打开字段映射页，JSON 样本字段被推断为候选字段。
勾选 temperature 创建字段映射后，数据点列表可按 sourceType=kafka.field 查到数据点。
删除 Topic 映射后，对应页签关闭，字段映射和数据点按策略清理。
返回接入源列表再进入 Kafka 工作台，不保留上一次预览任务。
```

- [ ] **步骤 6：提交最终集成**

运行：

```powershell
git add docs/统一REST接口规范与清单.md
git commit -m "docs(datacenter): 更新 Kafka 工作台接口清单"
```

如果验证阶段产生代码修正，单独提交：

```powershell
git add <修正文件>
git commit -m "fix(datacenter): 修正 Kafka 工作台联调问题"
```

没有修正时不创建空提交。

---

## 计划自检

- 规格覆盖：Topic 分组、Topic 映射、短时预览、字段推断、字段映射、数据点同步、专属工作台、错误处理、验证均有任务覆盖。
- 禁止表达扫描：已检查常见计划缺陷，未发现需要修正的问题。
- 类型一致性：后端统一使用 `KafkaTopicGroup / KafkaTopicMapping / KafkaField`；前端 schema 和组件类型同名。
- 修改边界：计划新增 Kafka 专属文件，修改接入源工作台分发、API 和后端路由；不要求重构 MQTT、HTTP、WebSocket 或 Redis 工作台。
- 验证闭环：后端有集成测试和 `go test ./...`，前端有模型单测、typecheck 和 build，最后有手动验收路径。
