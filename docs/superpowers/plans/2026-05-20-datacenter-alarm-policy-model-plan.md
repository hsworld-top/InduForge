# 数据中心报警策略模型实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 将报警工作区从单目标单条件规则升级为支持分组、策略、条件集、批量设置和多点计算报警的最终模型。

**架构：** 后端新增 `data_alarm_policy_groups` 和 `data_alarm_policies` 作为事实来源，`AlarmPolicyService` 负责分组、策略、条件集、试算、契约和批量操作。前端切到 `/alarm-policies` 接口，左侧使用策略管理面板，右侧按单策略、分组概览和批量设置切换视图。

**技术栈：** Go、PostgreSQL、Vue 3、Pinia、Vite、Element Plus、Zod、dayjs、Vitest。

---

## 文件结构

### 后端

- 创建：`data_service/internal/db/migrations/0018_alarm_policies.sql`  
  创建报警分组表和报警策略表。
- 创建：`data_service/internal/db/migrations/0018_alarm_policies_down.sql`  
  回滚报警策略表和分组表。
- 修改：`data_service/internal/db/migrations/0010_alarm_rules.sql`  
  新库初始化时直接包含策略表，避免全新环境只得到旧规则表。
- 修改：`data_service/internal/db/migrations/0010_alarm_rules_down.sql`  
  同步删除策略表和索引。
- 创建：`data_service/internal/repository/alarm_policy_repository.go`  
  封装分组、策略、树查询和批量更新 SQL。
- 创建：`data_service/internal/service/alarm_policy_service.go`  
  封装策略模型校验、条件评估、试算、契约生成和批量操作。
- 创建：`data_service/internal/service/alarm_policy_service_test.go`  
  覆盖策略模型、条件集、派生计算、契约和批量选择。
- 创建：`data_service/internal/http/handler/alarm_policy_handler.go`  
  承接分组、策略、试算、契约、草稿校验和批量 HTTP 请求。
- 修改：`data_service/internal/http/router/router.go`  
  新增 `/api/v1/data/projects/{projectId}/alarm-policy-groups` 和 `/alarm-policies` 路由。
- 修改：`data_service/internal/app/server.go`  
  初始化 `AlarmPolicyRepository`、`AlarmPolicyService`、`AlarmPolicyHandler` 并接入路由。
- 修改：`data_service/internal/service/contract_check_service.go`  
  若契约检查读取报警数据，改为读取策略契约摘要。
- 修改：`data_service/internal/service/contract_check_service_test.go`  
  同步契约检查中的报警断言。
- 修改：`data_service/tests/integration/migration_test.go`  
  增加策略表迁移断言。

### 前端

- 修改：`datacenter/src/api/schemas/alarm.schema.ts`  
  改为报警分组、策略、条件集、树查询、批量选择和试算 schema。
- 修改：`datacenter/src/api/alarm.api.ts`  
  切到 `/alarm-policy-groups` 和 `/alarm-policies` 接口。
- 修改：`datacenter/src/stores/alarm.store.ts`  
  管理分组、策略树、勾选、详情、批量操作、试算和契约状态。
- 创建：`datacenter/src/components/alarm/alarmPolicyModel.ts`  
  定义策略草稿、默认值、序列化、反序列化和批量选择工具。
- 删除或停止引用：`datacenter/src/components/alarm/alarmRuleModel.ts`  
  旧单规则模型不再作为工作区入口使用。
- 修改：`datacenter/src/components/alarm/AlarmWorkspace.vue`  
  切换为左侧管理面板 + 右侧策略工作区。
- 创建：`datacenter/src/components/alarm/AlarmManagementPanel.vue`  
  左侧管理区，承载筛选、全选、树和批量工具。
- 创建：`datacenter/src/components/alarm/AlarmPolicyTree.vue`  
  分组和策略混合树，支持根级未分组策略、勾选和半选。
- 创建：`datacenter/src/components/alarm/AlarmPolicyFilters.vue`  
  名称、分组、启停、级别、条件类型、目标点、判断模式筛选。
- 创建：`datacenter/src/components/alarm/AlarmBulkToolbar.vue`  
  展示选中数量和批量操作入口。
- 创建：`datacenter/src/components/alarm/AlarmPolicyEditor.vue`  
  单策略编辑器。
- 创建：`datacenter/src/components/alarm/AlarmConditionMatrix.vue`  
  条件集矩阵。
- 创建：`datacenter/src/components/alarm/AlarmTargetSelector.vue`  
  多目标点选择。
- 创建：`datacenter/src/components/alarm/AlarmInputExpressionPanel.vue`  
  输入点变量和派生表达式。
- 创建：`datacenter/src/components/alarm/AlarmGroupOverview.vue`  
  分组概览和分组级操作入口。
- 创建：`datacenter/src/components/alarm/AlarmBulkEditor.vue`  
  多选后的批量设置视图。
- 修改：`datacenter/src/components/alarm/AlarmTestPanel.vue`  
  适配策略试算结果。
- 修改：`datacenter/src/components/alarm/AlarmContractPanel.vue`  
  适配 `alarm.policy.v1` 契约。
- 修改：`datacenter/src/components/alarm/CreateAlarmRuleDialog.vue`  
  改为创建报警策略入口，或替换为 `CreateAlarmPolicyDialog.vue` 并更新引用。
- 修改：`datacenter/tests/alarm-schema.test.ts`
- 修改：`datacenter/tests/alarm-store.test.ts`
- 修改：`datacenter/tests/alarm-rule-model.test.ts`  
  改名或重写为策略模型测试。

## 任务 1：数据库迁移改为策略模型

**文件：**
- 创建：`data_service/internal/db/migrations/0018_alarm_policies.sql`
- 创建：`data_service/internal/db/migrations/0018_alarm_policies_down.sql`
- 修改：`data_service/internal/db/migrations/0010_alarm_rules.sql`
- 修改：`data_service/internal/db/migrations/0010_alarm_rules_down.sql`
- 修改：`data_service/tests/integration/migration_test.go`

- [ ] **步骤 1：编写失败的迁移测试**

在 `data_service/tests/integration/migration_test.go` 增加测试：

```go
func TestAlarmPolicyTablesMigration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupMigrator(t, fixture.pool)
	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("migrate up failed: %v", err)
	}

	assertColumnExists(t, ctx, fixture.pool, fixture.schemaName, "data_alarm_policy_groups", "is_enabled", "boolean")
	assertColumnExists(t, ctx, fixture.pool, fixture.schemaName, "data_alarm_policies", "group_id", "uuid")
	assertColumnExists(t, ctx, fixture.pool, fixture.schemaName, "data_alarm_policies", "targets", "jsonb")
	assertColumnExists(t, ctx, fixture.pool, fixture.schemaName, "data_alarm_policies", "conditions", "jsonb")
	assertColumnExists(t, ctx, fixture.pool, fixture.schemaName, "data_alarm_policies", "contract", "jsonb")
}
```

如测试文件没有 `assertColumnExists`，在同文件增加：

```go
func assertColumnExists(t *testing.T, ctx context.Context, pool *pgxpool.Pool, schemaName, tableName, columnName, wantType string) {
	t.Helper()
	var dataType string
	err := pool.QueryRow(ctx, `
        SELECT data_type
        FROM information_schema.columns
        WHERE table_schema = $1 AND table_name = $2 AND column_name = $3
    `, schemaName, tableName, columnName).Scan(&dataType)
	if err != nil {
		t.Fatalf("column %s.%s missing: %v", tableName, columnName, err)
	}
	if dataType != wantType {
		t.Fatalf("column %s.%s type = %q, want %q", tableName, columnName, dataType, wantType)
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：

```powershell
Set-Location data_service
go test ./tests/integration -run TestAlarmPolicyTablesMigration -count=1
```

预期：FAIL，提示 `data_alarm_policy_groups` 或 `data_alarm_policies` 不存在。

- [ ] **步骤 3：创建 `0018_alarm_policies.sql`**

写入：

```sql
CREATE TABLE IF NOT EXISTS data_alarm_policy_groups (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL REFERENCES data_projects(id) ON DELETE CASCADE,
    name varchar(100) NOT NULL,
    description text,
    is_enabled boolean NOT NULL DEFAULT true,
    sort_order integer NOT NULL DEFAULT 0,
    created_by uuid,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_alarm_policy_groups_project_name_unique UNIQUE (project_id, name)
);

CREATE TABLE IF NOT EXISTS data_alarm_policies (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL REFERENCES data_projects(id) ON DELETE CASCADE,
    group_id uuid REFERENCES data_alarm_policy_groups(id) ON DELETE SET NULL,
    name varchar(100) NOT NULL,
    description text,
    mode varchar(20) NOT NULL DEFAULT 'per_target',
    targets jsonb NOT NULL DEFAULT '[]'::jsonb,
    inputs jsonb NOT NULL DEFAULT '[]'::jsonb,
    derived_expression text NOT NULL DEFAULT '',
    conditions jsonb NOT NULL DEFAULT '[]'::jsonb,
    suppression jsonb NOT NULL DEFAULT '{}'::jsonb,
    message_template text NOT NULL DEFAULT '',
    is_enabled boolean NOT NULL DEFAULT true,
    contract jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_by uuid,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_alarm_policies_project_name_unique UNIQUE (project_id, name),
    CONSTRAINT data_alarm_policies_mode_check CHECK (mode IN ('per_target', 'derived')),
    CONSTRAINT data_alarm_policies_targets_array_check CHECK (jsonb_typeof(targets) = 'array'),
    CONSTRAINT data_alarm_policies_inputs_array_check CHECK (jsonb_typeof(inputs) = 'array'),
    CONSTRAINT data_alarm_policies_conditions_array_check CHECK (jsonb_typeof(conditions) = 'array')
);

CREATE INDEX IF NOT EXISTS idx_alarm_policy_groups_project_sort
    ON data_alarm_policy_groups(project_id, sort_order, created_at);

CREATE INDEX IF NOT EXISTS idx_alarm_policies_project_group
    ON data_alarm_policies(project_id, group_id);

CREATE INDEX IF NOT EXISTS idx_alarm_policies_project_updated
    ON data_alarm_policies(project_id, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_alarm_policies_project_enabled
    ON data_alarm_policies(project_id, is_enabled);
```

- [ ] **步骤 4：创建 down 迁移**

写入 `0018_alarm_policies_down.sql`：

```sql
DROP INDEX IF EXISTS idx_alarm_policies_project_enabled;
DROP INDEX IF EXISTS idx_alarm_policies_project_updated;
DROP INDEX IF EXISTS idx_alarm_policies_project_group;
DROP INDEX IF EXISTS idx_alarm_policy_groups_project_sort;
DROP TABLE IF EXISTS data_alarm_policies;
DROP TABLE IF EXISTS data_alarm_policy_groups;
```

- [ ] **步骤 5：同步 0010 初始化脚本**

在 `0010_alarm_rules.sql` 末尾追加同样的两个表和索引。  
在 `0010_alarm_rules_down.sql` 开头先删除策略表和索引，再删除旧规则表。

- [ ] **步骤 6：运行迁移测试验证通过**

运行：

```powershell
Set-Location data_service
go test ./tests/integration -run TestAlarmPolicyTablesMigration -count=1
```

预期：PASS。

- [ ] **步骤 7：Commit**

```powershell
git add data_service/internal/db/migrations/0018_alarm_policies.sql data_service/internal/db/migrations/0018_alarm_policies_down.sql data_service/internal/db/migrations/0010_alarm_rules.sql data_service/internal/db/migrations/0010_alarm_rules_down.sql data_service/tests/integration/migration_test.go
git commit -m "feat(data_service): 新增报警策略数据表"
```

## 任务 2：后端仓储实现分组、策略和树查询

**文件：**
- 创建：`data_service/internal/repository/alarm_policy_repository.go`

- [ ] **步骤 1：创建仓储类型和记录结构**

写入核心结构：

```go
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/indu-forge/data_service/internal/errors"
)

type AlarmPolicyGroupRecord struct {
	ID          string
	ProjectID   string
	Name        string
	Description *string
	IsEnabled   bool
	SortOrder   int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type AlarmPolicyRecord struct {
	ID                string
	ProjectID         string
	GroupID           *string
	GroupName         *string
	GroupEnabled      *bool
	Name              string
	Description       *string
	Mode              string
	Targets           []map[string]any
	Inputs            []map[string]any
	DerivedExpression string
	Conditions        []map[string]any
	Suppression        map[string]any
	MessageTemplate   string
	IsEnabled         bool
	Contract          map[string]any
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type AlarmPolicyListFilter struct {
	Search        string
	GroupID       *string
	Enabled       *bool
	Severity      string
	ConditionType string
	TargetPath    string
	Mode          string
	Page          int
	PageSize      int
}

type AlarmPolicyRepository struct {
	pool *pgxpool.Pool
}

func NewAlarmPolicyRepository(pool *pgxpool.Pool) *AlarmPolicyRepository {
	return &AlarmPolicyRepository{pool: pool}
}
```

- [ ] **步骤 2：实现 JSON 工具**

在同文件添加：

```go
func marshalJSONArray(value []map[string]any) ([]byte, error) {
	if value == nil {
		value = []map[string]any{}
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "报警策略数组 JSON 无效", err)
	}
	return payload, nil
}

func unmarshalJSONArray(payload []byte) ([]map[string]any, error) {
	if len(payload) == 0 {
		return []map[string]any{}, nil
	}
	var result []map[string]any
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解析报警策略数组失败", err)
	}
	if result == nil {
		return []map[string]any{}, nil
	}
	return result, nil
}
```

- [ ] **步骤 3：实现分组 CRUD**

添加方法：

```go
type CreateAlarmPolicyGroupParams struct {
	ProjectID   string
	UserID      string
	Name        string
	Description *string
	IsEnabled   bool
	SortOrder   int
}

type UpdateAlarmPolicyGroupParams struct {
	ID          string
	ProjectID   string
	UserID      string
	Name        string
	Description *string
	IsEnabled   bool
	SortOrder   int
}

func (r *AlarmPolicyRepository) ListGroups(ctx context.Context, projectID string) ([]AlarmPolicyGroupRecord, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT id, project_id, name, description, is_enabled, sort_order, created_at, updated_at
        FROM data_alarm_policy_groups
        WHERE project_id = $1
        ORDER BY sort_order ASC, created_at ASC
    `, projectID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询报警分组失败", err)
	}
	defer rows.Close()

	records := make([]AlarmPolicyGroupRecord, 0)
	for rows.Next() {
		record, scanErr := scanAlarmPolicyGroup(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, record)
	}
	return records, rows.Err()
}
```

同一步内实现这些方法：

```go
func (r *AlarmPolicyRepository) GetGroupByProjectAndID(ctx context.Context, projectID, id string) (*AlarmPolicyGroupRecord, error)
func (r *AlarmPolicyRepository) CreateGroup(ctx context.Context, params CreateAlarmPolicyGroupParams) (*AlarmPolicyGroupRecord, error)
func (r *AlarmPolicyRepository) UpdateGroup(ctx context.Context, params UpdateAlarmPolicyGroupParams) (*AlarmPolicyGroupRecord, error)
func (r *AlarmPolicyRepository) DeleteGroup(ctx context.Context, projectID, id string) error
```

写入要求：

- 所有 SQL 使用参数化查询。
- `GetGroupByProjectAndID` 查不到时返回 `ErrorCodeNotFound`，消息为 `报警分组不存在`。
- `CreateGroup` 和 `UpdateGroup` 遇到唯一约束冲突时返回 `报警分组名称已存在`。
- `DeleteGroup` 只删除分组记录，数据库外键自动把策略 `group_id` 置空。

- [ ] **步骤 4：实现策略 CRUD 和树查询**

添加方法签名：

```go
func (r *AlarmPolicyRepository) ListPolicies(ctx context.Context, projectID string, filter AlarmPolicyListFilter) ([]AlarmPolicyRecord, int, error)
func (r *AlarmPolicyRepository) ListPoliciesForTree(ctx context.Context, projectID string, filter AlarmPolicyListFilter) ([]AlarmPolicyRecord, error)
func (r *AlarmPolicyRepository) GetPolicyByProjectAndID(ctx context.Context, projectID, id string) (*AlarmPolicyRecord, error)
func (r *AlarmPolicyRepository) CreatePolicy(ctx context.Context, params CreateAlarmPolicyParams) (*AlarmPolicyRecord, error)
func (r *AlarmPolicyRepository) UpdatePolicy(ctx context.Context, params UpdateAlarmPolicyParams) (*AlarmPolicyRecord, error)
func (r *AlarmPolicyRepository) DeletePolicy(ctx context.Context, projectID, id string) error
```

`ListPoliciesForTree` 不分页，按 `group_id NULLS FIRST, name ASC` 排序，只返回筛选命中的策略。

- [ ] **步骤 5：实现批量更新**

添加：

```go
func (r *AlarmPolicyRepository) SetPoliciesEnabled(ctx context.Context, projectID, userID string, ids []string, enabled bool) error
func (r *AlarmPolicyRepository) MovePolicies(ctx context.Context, projectID, userID string, ids []string, groupID *string) error
func (r *AlarmPolicyRepository) ListPolicyIDsByFilter(ctx context.Context, projectID string, filter AlarmPolicyListFilter, excludeIDs []string) ([]string, error)
```

所有 SQL 使用 `$1`、`$2` 参数。`ids` 使用 `ANY($n::uuid[])`。

- [ ] **步骤 6：运行后端编译测试**

```powershell
Set-Location data_service
go test ./internal/repository
```

预期：PASS 或提示没有测试文件但编译通过。

- [ ] **步骤 7：Commit**

```powershell
git add data_service/internal/repository/alarm_policy_repository.go
git commit -m "feat(data_service): 实现报警策略仓储"
```

## 任务 3：后端服务实现策略校验、条件集和契约

**文件：**
- 创建：`data_service/internal/service/alarm_policy_service.go`
- 创建：`data_service/internal/service/alarm_policy_service_test.go`

- [ ] **步骤 1：编写条件集和契约测试**

在 `alarm_policy_service_test.go` 写入：

```go
package service

import "testing"

func TestEvaluateAlarmPolicyConditions(t *testing.T) {
	policy := AlarmPolicy{
		Mode: "per_target",
		Conditions: []AlarmCondition{
			{ID: "c-h", Type: "H", Name: "高限", IsEnabled: true, Severity: "major", Params: map[string]any{"limit": 80.0}},
			{ID: "c-l", Type: "L", Name: "低限", IsEnabled: true, Severity: "warning", Params: map[string]any{"limit": 20.0}},
		},
	}
	result, err := evaluateAlarmPolicy(&policy, 82.0, nil)
	if err != nil {
		t.Fatalf("evaluateAlarmPolicy() error = %v", err)
	}
	if !result.Triggered || len(result.TriggeredConditions) != 1 || result.TriggeredConditions[0].Type != "H" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestBuildAlarmPolicyContract(t *testing.T) {
	policy := normalizedAlarmPolicyInput{
		Name: "温度策略",
		Mode: "per_target",
		Targets: []AlarmTargetRef{{DatapointID: "dp-1", Path: "metrics.temperature", DataType: "number"}},
		Conditions: []AlarmCondition{{ID: "c-h", Type: "H", Name: "高限", IsEnabled: true, Severity: "major", Params: map[string]any{"limit": 80.0}}},
		IsEnabled: true,
	}
	contract := buildAlarmPolicyContract(policy, "policy-1", "project-1", nil)
	if contract["schemaVersion"] != "alarm.policy.v1" {
		t.Fatalf("schemaVersion = %v", contract["schemaVersion"])
	}
	if contract["effectiveEnabled"] != true {
		t.Fatalf("effectiveEnabled = %v", contract["effectiveEnabled"])
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

```powershell
Set-Location data_service
go test ./internal/service -run 'TestEvaluateAlarmPolicyConditions|TestBuildAlarmPolicyContract' -count=1
```

预期：FAIL，提示类型或函数未定义。

- [ ] **步骤 3：实现服务类型和 DTO**

在 `alarm_policy_service.go` 定义：

```go
type AlarmPolicyGroup struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"projectId"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	IsEnabled   bool      `json:"isEnabled"`
	SortOrder   int       `json:"sortOrder"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type AlarmTargetRef struct {
	DatapointID string `json:"datapointId"`
	Path        string `json:"path"`
	Name        string `json:"name,omitempty"`
	DataType    string `json:"dataType"`
}

type AlarmInputRef struct {
	Key         string `json:"key"`
	DatapointID string `json:"datapointId"`
	Path        string `json:"path"`
	Name        string `json:"name,omitempty"`
	DataType    string `json:"dataType"`
}

type AlarmCondition struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Name      string         `json:"name"`
	IsEnabled bool           `json:"isEnabled"`
	Severity  string         `json:"severity"`
	Params    map[string]any `json:"params"`
}

type AlarmPolicy struct {
	ID                string           `json:"id"`
	ProjectID         string           `json:"projectId"`
	GroupID           *string          `json:"groupId,omitempty"`
	GroupName         *string          `json:"groupName,omitempty"`
	GroupEnabled      *bool            `json:"groupEnabled,omitempty"`
	Name              string           `json:"name"`
	Description       *string          `json:"description,omitempty"`
	Mode              string           `json:"mode"`
	Targets           []AlarmTargetRef `json:"targets"`
	Inputs            []AlarmInputRef  `json:"inputs"`
	DerivedExpression string           `json:"derivedExpression"`
	Conditions        []AlarmCondition `json:"conditions"`
	Suppression        map[string]any   `json:"suppression"`
	MessageTemplate   string           `json:"messageTemplate"`
	IsEnabled         bool             `json:"isEnabled"`
	EffectiveEnabled  bool             `json:"effectiveEnabled"`
	Contract          map[string]any   `json:"contract"`
	CreatedAt         time.Time        `json:"createdAt"`
	UpdatedAt         time.Time        `json:"updatedAt"`
}
```

- [ ] **步骤 4：实现校验**

实现：

```go
func validateAlarmPolicyMode(mode string) (string, error)
func validateAlarmConditions(conditions []AlarmCondition, numericOnly bool) error
func validateAlarmConditionParams(condition AlarmCondition) error
func validateAlarmInputs(inputs []AlarmInputRef) error
func validateAlarmTargets(targets []AlarmTargetRef) error
```

规则：

- `mode` 只允许 `per_target` 和 `derived`。
- `per_target` 必须至少一个 target。
- `derived` 必须至少一个 input 和非空 `derivedExpression`。
- 条件集至少一个启用条件。
- 条件类型只允许 `HH/H/L/LL/deviation_high/deviation_low/rate_of_change/cel`。
- 非数值目标禁止非 `cel` 条件。
- `condition.params.limit` 对阈值、偏差、变化率必填且为数字。
- `rate_of_change` 必须有 `windowMs`。
- `cel` 必须有 `expression`。

- [ ] **步骤 5：实现条件评估和派生表达式**

复用 `alarm_rule_service.go` 中的 `evaluateUpperLimit`、`evaluateLowerLimit`、`evaluateDeviation`、`evaluateRateOfChange`、`evaluateCEL`、`anyToFloat64`。  
如函数是私有但同包可用，直接调用。

实现：

```go
type AlarmPolicyEvaluationResult struct {
	Triggered           bool                      `json:"triggered"`
	State               string                    `json:"state"`
	TriggeredConditions []AlarmCondition          `json:"triggeredConditions"`
	Diagnostics         map[string]any            `json:"diagnostics"`
	ConditionResults    []map[string]any          `json:"conditionResults"`
}

func evaluateAlarmPolicy(policy *AlarmPolicy, value any, context map[string]any) (*AlarmPolicyEvaluationResult, error)
func evaluateDerivedValue(expression string, values map[string]any) (float64, error)
```

`evaluateDerivedValue` 的表达式范围固定为变量名、数字、`+ - * /` 和括号。不要实现函数调用。

- [ ] **步骤 6：实现契约生成**

实现：

```go
func buildAlarmPolicyContract(input normalizedAlarmPolicyInput, policyID string, projectID string, groupEnabled *bool) map[string]any {
	effectiveEnabled := input.IsEnabled
	if groupEnabled != nil {
		effectiveEnabled = effectiveEnabled && *groupEnabled
	}
	return map[string]any{
		"schemaVersion": "alarm.policy.v1",
		"policyId": policyID,
		"projectId": projectID,
		"groupId": input.GroupID,
		"mode": input.Mode,
		"targets": input.Targets,
		"inputs": input.Inputs,
		"derivedExpression": input.DerivedExpression,
		"conditions": input.Conditions,
		"suppression": input.Suppression,
		"messageTemplate": input.MessageTemplate,
		"enabled": input.IsEnabled,
		"effectiveEnabled": effectiveEnabled,
		"updatedAt": "",
	}
}
```

- [ ] **步骤 7：实现服务方法**

实现：

```go
func (s *AlarmPolicyService) ListGroups(ctx context.Context, claims *auth.Claims, projectID string) ([]AlarmPolicyGroup, error)
func (s *AlarmPolicyService) CreateGroup(ctx context.Context, claims *auth.Claims, projectID string, input CreateAlarmPolicyGroupInput) (*AlarmPolicyGroup, error)
func (s *AlarmPolicyService) UpdateGroup(ctx context.Context, claims *auth.Claims, projectID, groupID string, input UpdateAlarmPolicyGroupInput) (*AlarmPolicyGroup, error)
func (s *AlarmPolicyService) DeleteGroup(ctx context.Context, claims *auth.Claims, projectID, groupID string) error
func (s *AlarmPolicyService) List(ctx context.Context, claims *auth.Claims, projectID string, filter AlarmPolicyListFilter) (*AlarmPolicyListResult, error)
func (s *AlarmPolicyService) Tree(ctx context.Context, claims *auth.Claims, projectID string, filter AlarmPolicyListFilter) (*AlarmPolicyTreeResult, error)
func (s *AlarmPolicyService) Get(ctx context.Context, claims *auth.Claims, projectID, policyID string) (*AlarmPolicy, error)
func (s *AlarmPolicyService) Create(ctx context.Context, claims *auth.Claims, projectID string, input CreateAlarmPolicyInput) (*AlarmPolicy, error)
func (s *AlarmPolicyService) Update(ctx context.Context, claims *auth.Claims, projectID, policyID string, input UpdateAlarmPolicyInput) (*AlarmPolicy, error)
func (s *AlarmPolicyService) Delete(ctx context.Context, claims *auth.Claims, projectID, policyID string) error
func (s *AlarmPolicyService) Test(ctx context.Context, claims *auth.Claims, projectID, policyID string, input AlarmPolicyTestInput) (*AlarmPolicyEvaluationResult, error)
func (s *AlarmPolicyService) Contract(ctx context.Context, claims *auth.Claims, projectID, policyID string) (map[string]any, error)
func (s *AlarmPolicyService) ValidateDraft(ctx context.Context, claims *auth.Claims, projectID string, input CreateAlarmPolicyInput) (map[string]any, error)
```

- [ ] **步骤 8：运行服务测试**

```powershell
Set-Location data_service
go test ./internal/service -run AlarmPolicy -count=1
```

预期：PASS。

- [ ] **步骤 9：Commit**

```powershell
git add data_service/internal/service/alarm_policy_service.go data_service/internal/service/alarm_policy_service_test.go
git commit -m "feat(data_service): 实现报警策略服务"
```

## 任务 4：HTTP handler、路由和应用装配

**文件：**
- 创建：`data_service/internal/http/handler/alarm_policy_handler.go`
- 修改：`data_service/internal/http/router/router.go`
- 修改：`data_service/internal/app/server.go`

- [ ] **步骤 1：创建 handler**

在 `alarm_policy_handler.go` 定义：

```go
type AlarmPolicyHandler struct {
	service *service.AlarmPolicyService
}

func NewAlarmPolicyHandler(alarmPolicyService *service.AlarmPolicyService) *AlarmPolicyHandler {
	return &AlarmPolicyHandler{service: alarmPolicyService}
}
```

实现以下方法：

```go
func (h *AlarmPolicyHandler) ListGroups(w http.ResponseWriter, r *http.Request) error
func (h *AlarmPolicyHandler) CreateGroup(w http.ResponseWriter, r *http.Request) error
func (h *AlarmPolicyHandler) UpdateGroup(w http.ResponseWriter, r *http.Request) error
func (h *AlarmPolicyHandler) ToggleGroupEnabled(w http.ResponseWriter, r *http.Request) error
func (h *AlarmPolicyHandler) DeleteGroup(w http.ResponseWriter, r *http.Request) error
func (h *AlarmPolicyHandler) List(w http.ResponseWriter, r *http.Request) error
func (h *AlarmPolicyHandler) Tree(w http.ResponseWriter, r *http.Request) error
func (h *AlarmPolicyHandler) Create(w http.ResponseWriter, r *http.Request) error
func (h *AlarmPolicyHandler) Get(w http.ResponseWriter, r *http.Request) error
func (h *AlarmPolicyHandler) Update(w http.ResponseWriter, r *http.Request) error
func (h *AlarmPolicyHandler) ToggleEnabled(w http.ResponseWriter, r *http.Request) error
func (h *AlarmPolicyHandler) Delete(w http.ResponseWriter, r *http.Request) error
func (h *AlarmPolicyHandler) ValidateDraft(w http.ResponseWriter, r *http.Request) error
func (h *AlarmPolicyHandler) Test(w http.ResponseWriter, r *http.Request) error
func (h *AlarmPolicyHandler) Contract(w http.ResponseWriter, r *http.Request) error
func (h *AlarmPolicyHandler) BatchEnable(w http.ResponseWriter, r *http.Request) error
func (h *AlarmPolicyHandler) BatchDisable(w http.ResponseWriter, r *http.Request) error
func (h *AlarmPolicyHandler) BatchMove(w http.ResponseWriter, r *http.Request) error
func (h *AlarmPolicyHandler) BatchApplyConditions(w http.ResponseWriter, r *http.Request) error
```

所有成功响应使用：

```go
response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
```

- [ ] **步骤 2：解析筛选参数**

添加：

```go
func parseAlarmPolicyListFilter(r *http.Request) (service.AlarmPolicyListFilter, error) {
	query := r.URL.Query()
	page, err := parseOptionalInt(query.Get("page"), 1, "page")
	if err != nil {
		return service.AlarmPolicyListFilter{}, err
	}
	pageSize, err := parseOptionalInt(firstNonEmpty(query.Get("pageSize"), query.Get("limit")), 20, "pageSize")
	if err != nil {
		return service.AlarmPolicyListFilter{}, err
	}
	enabled, err := parseOptionalBool(query.Get("enabled"))
	if err != nil {
		return service.AlarmPolicyListFilter{}, err
	}
	return service.AlarmPolicyListFilter{
		Search:        query.Get("search"),
		GroupID:       nullableQueryString(query.Get("groupId")),
		Enabled:       enabled,
		Severity:      query.Get("severity"),
		ConditionType: query.Get("conditionType"),
		TargetPath:    query.Get("targetPath"),
		Mode:          query.Get("mode"),
		Page:          page,
		PageSize:      pageSize,
	}, nil
}
```

- [ ] **步骤 3：注册路由**

在 `router.go` 增加 `alarmPolicyHandler` 字段、`WithAlarmPolicyRoutes` 和 `mountAlarmPolicyRoutes`。  
注册路径：

```go
"GET /api/v1/data/projects/{projectId}/alarm-policy-groups"
"POST /api/v1/data/projects/{projectId}/alarm-policy-groups"
"PUT /api/v1/data/projects/{projectId}/alarm-policy-groups/{id}"
"PATCH /api/v1/data/projects/{projectId}/alarm-policy-groups/{id}/enabled"
"DELETE /api/v1/data/projects/{projectId}/alarm-policy-groups/{id}"
"GET /api/v1/data/projects/{projectId}/alarm-policies"
"GET /api/v1/data/projects/{projectId}/alarm-policies/tree"
"POST /api/v1/data/projects/{projectId}/alarm-policies"
"GET /api/v1/data/projects/{projectId}/alarm-policies/{id}"
"PUT /api/v1/data/projects/{projectId}/alarm-policies/{id}"
"PATCH /api/v1/data/projects/{projectId}/alarm-policies/{id}/enabled"
"DELETE /api/v1/data/projects/{projectId}/alarm-policies/{id}"
"POST /api/v1/data/projects/{projectId}/alarm-policies/validate-draft"
"POST /api/v1/data/projects/{projectId}/alarm-policies/{id}/test"
"GET /api/v1/data/projects/{projectId}/alarm-policies/{id}/contract"
"POST /api/v1/data/projects/{projectId}/alarm-policies/batch-enable"
"POST /api/v1/data/projects/{projectId}/alarm-policies/batch-disable"
"POST /api/v1/data/projects/{projectId}/alarm-policies/batch-move"
"POST /api/v1/data/projects/{projectId}/alarm-policies/batch-apply-conditions"
```

- [ ] **步骤 4：装配 app server**

在 `server.go` 初始化：

```go
alarmPolicyRepository := repository.NewAlarmPolicyRepository(dbPool)
alarmPolicyService := service.NewAlarmPolicyService(alarmPolicyRepository, dataPointRepository)
alarmPolicyHandler := handler.NewAlarmPolicyHandler(alarmPolicyService)
```

在 router options 中加入：

```go
router.WithAlarmPolicyRoutes(alarmPolicyHandler, jwtValidator)
```

- [ ] **步骤 5：运行后端全量测试**

```powershell
Set-Location data_service
go test ./...
```

预期：PASS。

- [ ] **步骤 6：Commit**

```powershell
git add data_service/internal/http/handler/alarm_policy_handler.go data_service/internal/http/router/router.go data_service/internal/app/server.go
git commit -m "feat(data_service): 接入报警策略接口"
```

## 任务 5：前端 schema 和 API 切到策略模型

**文件：**
- 修改：`datacenter/src/api/schemas/alarm.schema.ts`
- 修改：`datacenter/src/api/alarm.api.ts`
- 修改：`datacenter/tests/alarm-schema.test.ts`

- [ ] **步骤 1：编写 schema 测试**

在 `alarm-schema.test.ts` 写入：

```ts
import { describe, expect, it } from "vitest";
import {
  AlarmPolicySchema,
  AlarmPolicySaveSchema,
  AlarmPolicyTreeSchema,
  AlarmBulkSelectionSchema,
} from "@/api/schemas/alarm.schema";

describe("alarm policy schema", () => {
  it("parses policy with multiple conditions", () => {
    const parsed = AlarmPolicySchema.parse({
      id: "policy-1",
      projectId: "project-1",
      groupId: null,
      name: "温度策略",
      mode: "per_target",
      targets: [{ datapointId: "dp-1", path: "metrics.temperature", dataType: "number" }],
      inputs: [],
      derivedExpression: "",
      conditions: [
        { id: "c-h", type: "H", name: "高限", isEnabled: true, severity: "major", params: { limit: 80 } },
        { id: "c-l", type: "L", name: "低限", isEnabled: true, severity: "warning", params: { limit: 20 } },
      ],
      suppression: {},
      messageTemplate: "",
      isEnabled: true,
      effectiveEnabled: true,
      contract: {},
      createdAt: "2026-05-20T10:00:00Z",
      updatedAt: "2026-05-20T10:00:00Z",
    });

    expect(parsed.conditions).toHaveLength(2);
  });

  it("rejects unknown save fields", () => {
    expect(() =>
      AlarmPolicySaveSchema.parse({
        name: "策略",
        mode: "per_target",
        targets: [],
        inputs: [],
        derivedExpression: "",
        conditions: [],
        suppression: {},
        messageTemplate: "",
        isEnabled: true,
        targetPath: "old.field",
      }),
    ).toThrow();
  });

  it("parses filtered selection", () => {
    const parsed = AlarmBulkSelectionSchema.parse({
      mode: "filtered",
      filters: { search: "温度", enabled: true },
      excludePolicyIds: ["policy-2"],
    });
    expect(parsed.mode).toBe("filtered");
  });
});
```

- [ ] **步骤 2：运行测试验证失败**

```powershell
pnpm --dir datacenter test -- alarm-schema
```

预期：FAIL，提示 schema 未导出。

- [ ] **步骤 3：实现 schema**

在 `alarm.schema.ts` 定义并导出：

```ts
export const AlarmConditionTypeSchema = z.enum([
  "HH",
  "H",
  "L",
  "LL",
  "deviation_high",
  "deviation_low",
  "rate_of_change",
  "cel",
]);

export const AlarmPolicyModeSchema = z.enum(["per_target", "derived"]);
export const AlarmSeveritySchema = z.enum(["info", "warning", "major", "critical"]);

export const AlarmTargetRefSchema = z.object({
  datapointId: z.string(),
  path: z.string(),
  name: z.string().optional(),
  dataType: z.string(),
});

export const AlarmInputRefSchema = AlarmTargetRefSchema.extend({
  key: z.string(),
});

export const AlarmConditionSchema = z.object({
  id: z.string(),
  type: AlarmConditionTypeSchema,
  name: z.string(),
  isEnabled: z.boolean(),
  severity: AlarmSeveritySchema,
  params: z.record(z.string(), z.unknown()).default({}),
});

export const AlarmPolicyGroupSchema = z.object({
  id: z.string(),
  projectId: z.string(),
  name: z.string(),
  description: z.string().nullish(),
  isEnabled: z.boolean(),
  sortOrder: z.number(),
  createdAt: z.string(),
  updatedAt: z.string(),
});

export const AlarmPolicySchema = z.object({
  id: z.string(),
  projectId: z.string(),
  groupId: z.string().nullish(),
  groupName: z.string().nullish(),
  groupEnabled: z.boolean().nullish(),
  name: z.string(),
  description: z.string().nullish(),
  mode: AlarmPolicyModeSchema,
  targets: z.array(AlarmTargetRefSchema),
  inputs: z.array(AlarmInputRefSchema),
  derivedExpression: z.string(),
  conditions: z.array(AlarmConditionSchema),
  suppression: z.record(z.string(), z.unknown()).default({}),
  messageTemplate: z.string(),
  isEnabled: z.boolean(),
  effectiveEnabled: z.boolean(),
  contract: z.record(z.string(), z.unknown()).default({}),
  createdAt: z.string(),
  updatedAt: z.string(),
});

export const AlarmPolicySaveSchema = z.object({
  groupId: z.string().nullable().optional(),
  name: z.string().min(1),
  description: z.string().nullable().optional(),
  mode: AlarmPolicyModeSchema,
  targets: z.array(AlarmTargetRefSchema),
  inputs: z.array(AlarmInputRefSchema),
  derivedExpression: z.string(),
  conditions: z.array(AlarmConditionSchema),
  suppression: z.record(z.string(), z.unknown()).default({}),
  messageTemplate: z.string().default(""),
  isEnabled: z.boolean().default(true),
}).strict();
```

同一步内导出：

```ts
export const AlarmPolicyUpdateSchema = AlarmPolicySaveSchema.partial().strict();

export const AlarmPolicyTreeSchema = z.object({
  groups: z.array(AlarmPolicyGroupSchema),
  rootPolicies: z.array(AlarmPolicySchema),
  matchedPolicyCount: z.number(),
  totalPolicyCount: z.number(),
});

export const AlarmBulkSelectionSchema = z.discriminatedUnion("mode", [
  z.object({ mode: z.literal("ids"), policyIds: z.array(z.string()) }),
  z.object({
    mode: z.literal("filtered"),
    filters: z.record(z.string(), z.unknown()),
    excludePolicyIds: z.array(z.string()).default([]),
  }),
]);

export const AlarmPolicyTrialPayloadSchema = z.object({
  value: z.unknown().optional(),
  values: z.record(z.string(), z.unknown()).optional(),
  timestamp: z.string().optional(),
  context: z.record(z.string(), z.unknown()).default({}),
}).default({});

export const AlarmPolicyTrialResultSchema = z.object({
  triggered: z.boolean(),
  state: z.enum(["triggered", "not_triggered", "insufficient_input"]),
  triggeredConditions: z.array(AlarmConditionSchema).default([]),
  diagnostics: z.record(z.string(), z.unknown()).default({}),
  conditionResults: z.array(z.record(z.string(), z.unknown())).default([]),
});

export const AlarmPolicyContractSchema = z.record(z.string(), z.unknown());
```

导出对应 type：

```ts
export type AlarmPolicy = z.infer<typeof AlarmPolicySchema>;
export type AlarmPolicySave = z.infer<typeof AlarmPolicySaveSchema>;
export type AlarmPolicyUpdate = z.infer<typeof AlarmPolicyUpdateSchema>;
export type AlarmPolicyGroup = z.infer<typeof AlarmPolicyGroupSchema>;
export type AlarmPolicyTree = z.infer<typeof AlarmPolicyTreeSchema>;
export type AlarmBulkSelection = z.infer<typeof AlarmBulkSelectionSchema>;
export type AlarmPolicyTrialPayload = z.infer<typeof AlarmPolicyTrialPayloadSchema>;
export type AlarmPolicyTrialResult = z.infer<typeof AlarmPolicyTrialResultSchema>;
export type AlarmPolicyContract = z.infer<typeof AlarmPolicyContractSchema>;
```

- [ ] **步骤 4：更新 API**

在 `alarm.api.ts` 替换规则 API 为策略 API：

```ts
export async function getAlarmPolicyTree(projectId: string, params: Record<string, unknown> = {}) {
  const res = await request({
    url: `/data/projects/${projectId}/alarm-policies/tree`,
    method: "get",
    params,
  });
  return AlarmPolicyTreeSchema.parse(unwrapData(res));
}

export async function createAlarmPolicy(projectId: string, data: AlarmPolicySave): Promise<AlarmPolicy> {
  const body = AlarmPolicySaveSchema.parse(data);
  const res = await request({
    url: `/data/projects/${projectId}/alarm-policies`,
    method: "post",
    data: body,
  });
  return AlarmPolicySchema.parse(unwrapData(res));
}
```

同文件新增这些 API 函数：

```ts
export async function getAlarmPolicyGroups(projectId: string): Promise<AlarmPolicyGroup[]>
export async function createAlarmPolicyGroup(projectId: string, data: AlarmPolicyGroupSave): Promise<AlarmPolicyGroup>
export async function updateAlarmPolicyGroup(projectId: string, groupId: string, data: AlarmPolicyGroupUpdate): Promise<AlarmPolicyGroup>
export async function toggleAlarmPolicyGroup(projectId: string, groupId: string, isEnabled: boolean): Promise<AlarmPolicyGroup>
export async function deleteAlarmPolicyGroup(projectId: string, groupId: string): Promise<void>
export async function getAlarmPolicies(projectId: string, params?: Record<string, unknown>): Promise<AlarmPolicyListResp>
export async function getAlarmPolicy(projectId: string, policyId: string): Promise<AlarmPolicy>
export async function updateAlarmPolicy(projectId: string, policyId: string, data: AlarmPolicyUpdate): Promise<AlarmPolicy>
export async function toggleAlarmPolicy(projectId: string, policyId: string, isEnabled: boolean): Promise<AlarmPolicy>
export async function deleteAlarmPolicy(projectId: string, policyId: string): Promise<void>
export async function testAlarmPolicy(projectId: string, policyId: string, payload?: AlarmPolicyTrialPayload): Promise<AlarmPolicyTrialResult>
export async function getAlarmPolicyContract(projectId: string, policyId: string): Promise<AlarmPolicyContract>
export async function validateAlarmPolicyDraft(projectId: string, data: AlarmPolicySave): Promise<AlarmDraftValidation>
export async function batchEnableAlarmPolicies(projectId: string, selection: AlarmBulkSelection): Promise<void>
export async function batchDisableAlarmPolicies(projectId: string, selection: AlarmBulkSelection): Promise<void>
export async function batchMoveAlarmPolicies(projectId: string, selection: AlarmBulkSelection, groupId: string | null): Promise<void>
export async function batchApplyAlarmConditions(projectId: string, selection: AlarmBulkSelection, conditions: AlarmCondition[]): Promise<void>
```

- [ ] **步骤 5：运行 schema 测试**

```powershell
pnpm --dir datacenter test -- alarm-schema
```

预期：PASS。

- [ ] **步骤 6：Commit**

```powershell
git add datacenter/src/api/schemas/alarm.schema.ts datacenter/src/api/alarm.api.ts datacenter/tests/alarm-schema.test.ts
git commit -m "feat(datacenter): 定义报警策略前端接口"
```

## 任务 6：前端策略草稿模型和选择模型

**文件：**
- 创建：`datacenter/src/components/alarm/alarmPolicyModel.ts`
- 修改：`datacenter/tests/alarm-rule-model.test.ts`

- [ ] **步骤 1：重写模型测试**

将 `alarm-rule-model.test.ts` 改为：

```ts
import { describe, expect, it } from "vitest";
import {
  createDefaultAlarmPolicyDraft,
  draftToAlarmPolicySavePayload,
  makeFilteredSelection,
  togglePolicyInSelection,
  toAlarmPolicyDraft,
} from "@/components/alarm/alarmPolicyModel";
import type { AlarmPolicy } from "@/api/schemas/alarm.schema";

describe("alarmPolicyModel", () => {
  it("creates per target draft with empty conditions", () => {
    const draft = createDefaultAlarmPolicyDraft();
    expect(draft.mode).toBe("per_target");
    expect(draft.targets).toEqual([]);
    expect(draft.conditions).toEqual([]);
  });

  it("serializes multiple conditions", () => {
    const payload = draftToAlarmPolicySavePayload({
      ...createDefaultAlarmPolicyDraft(),
      name: "温度策略",
      conditions: [
        { id: "c-h", type: "H", name: "高限", isEnabled: true, severity: "major", params: { limit: 80 } },
        { id: "c-l", type: "L", name: "低限", isEnabled: true, severity: "warning", params: { limit: 20 } },
      ],
    });
    expect(payload.conditions).toHaveLength(2);
  });

  it("keeps root policy group id null", () => {
    const policy = makePolicy({ groupId: null });
    const draft = toAlarmPolicyDraft(policy);
    expect(draft.groupId).toBeNull();
  });

  it("tracks filtered select all with exclusions", () => {
    const selection = makeFilteredSelection({ search: "温度", enabled: true });
    const next = togglePolicyInSelection(selection, "policy-2", false);
    expect(next.mode).toBe("filtered");
    expect(next.excludePolicyIds).toContain("policy-2");
  });
});

function makePolicy(patch: Partial<AlarmPolicy> = {}): AlarmPolicy {
  return {
    id: "policy-1",
    projectId: "project-1",
    groupId: null,
    name: "温度策略",
    mode: "per_target",
    targets: [],
    inputs: [],
    derivedExpression: "",
    conditions: [],
    suppression: {},
    messageTemplate: "",
    isEnabled: true,
    effectiveEnabled: true,
    contract: {},
    createdAt: "2026-05-20T10:00:00Z",
    updatedAt: "2026-05-20T10:00:00Z",
    ...patch,
  };
}
```

- [ ] **步骤 2：运行测试验证失败**

```powershell
pnpm --dir datacenter test -- alarm-rule-model
```

预期：FAIL，提示 `alarmPolicyModel` 不存在。

- [ ] **步骤 3：实现草稿模型**

创建 `alarmPolicyModel.ts`：

```ts
import type {
  AlarmBulkSelection,
  AlarmCondition,
  AlarmPolicy,
  AlarmPolicyMode,
  AlarmPolicySave,
  AlarmTargetRef,
  AlarmInputRef,
} from "@/api/schemas/alarm.schema";

export type AlarmPolicyDraft = {
  id?: string;
  groupId: string | null;
  name: string;
  description: string;
  mode: AlarmPolicyMode;
  targets: AlarmTargetRef[];
  inputs: AlarmInputRef[];
  derivedExpression: string;
  conditions: AlarmCondition[];
  suppression: Record<string, unknown>;
  messageTemplate: string;
  isEnabled: boolean;
  dirty: boolean;
};

export function createDefaultAlarmPolicyDraft(): AlarmPolicyDraft {
  return {
    groupId: null,
    name: "",
    description: "",
    mode: "per_target",
    targets: [],
    inputs: [],
    derivedExpression: "",
    conditions: [],
    suppression: { enabled: false },
    messageTemplate: "",
    isEnabled: true,
    dirty: false,
  };
}
```

同文件实现：

```ts
export function toAlarmPolicyDraft(policy: AlarmPolicy): AlarmPolicyDraft {
  return {
    id: policy.id,
    groupId: policy.groupId ?? null,
    name: policy.name,
    description: policy.description ?? "",
    mode: policy.mode,
    targets: [...policy.targets],
    inputs: [...policy.inputs],
    derivedExpression: policy.derivedExpression,
    conditions: policy.conditions.map((condition) => ({
      ...condition,
      params: { ...condition.params },
    })),
    suppression: { ...policy.suppression },
    messageTemplate: policy.messageTemplate,
    isEnabled: policy.isEnabled,
    dirty: false,
  };
}

export function draftToAlarmPolicySavePayload(draft: AlarmPolicyDraft): AlarmPolicySave {
  return {
    groupId: draft.groupId,
    name: draft.name.trim(),
    description: draft.description.trim() || null,
    mode: draft.mode,
    targets: draft.targets,
    inputs: draft.inputs,
    derivedExpression: draft.derivedExpression.trim(),
    conditions: draft.conditions,
    suppression: draft.suppression,
    messageTemplate: draft.messageTemplate.trim(),
    isEnabled: draft.isEnabled,
  };
}

export function makeIdSelection(policyIds: string[] = []): AlarmBulkSelection {
  return { mode: "ids", policyIds };
}

export function makeFilteredSelection(filters: Record<string, unknown>): AlarmBulkSelection {
  return { mode: "filtered", filters, excludePolicyIds: [] };
}

export function togglePolicyInSelection(selection: AlarmBulkSelection, policyId: string, selected: boolean): AlarmBulkSelection {
  if (selection.mode === "filtered") {
    const excluded = new Set(selection.excludePolicyIds);
    if (selected) {
      excluded.delete(policyId);
    } else {
      excluded.add(policyId);
    }
    return { ...selection, excludePolicyIds: [...excluded] };
  }
  const ids = new Set(selection.policyIds);
  if (selected) {
    ids.add(policyId);
  } else {
    ids.delete(policyId);
  }
  return { mode: "ids", policyIds: [...ids] };
}
```

- [ ] **步骤 4：运行模型测试**

```powershell
pnpm --dir datacenter test -- alarm-rule-model
```

预期：PASS。

- [ ] **步骤 5：Commit**

```powershell
git add datacenter/src/components/alarm/alarmPolicyModel.ts datacenter/tests/alarm-rule-model.test.ts
git commit -m "feat(datacenter): 实现报警策略草稿模型"
```

## 任务 7：前端 store 管理策略树和批量操作

**文件：**
- 修改：`datacenter/src/stores/alarm.store.ts`
- 修改：`datacenter/tests/alarm-store.test.ts`

- [ ] **步骤 1：编写 store 测试**

在 `alarm-store.test.ts` 增加：

```ts
import { createPinia, setActivePinia } from "pinia";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useAlarmStore } from "@/stores/alarm.store";

vi.mock("@/api/alarm.api", () => ({
  getAlarmPolicyGroups: vi.fn(async () => []),
  getAlarmPolicyTree: vi.fn(async () => ({
    groups: [{ id: "group-1", projectId: "project-1", name: "A", isEnabled: true, sortOrder: 1, createdAt: "", updatedAt: "" }],
    rootPolicies: [{ id: "policy-root", projectId: "project-1", groupId: null, name: "根策略", mode: "per_target", targets: [], inputs: [], derivedExpression: "", conditions: [], suppression: {}, messageTemplate: "", isEnabled: true, effectiveEnabled: true, contract: {}, createdAt: "", updatedAt: "" }],
    matchedPolicyCount: 1,
    totalPolicyCount: 1,
  })),
}));

describe("alarm store policy tree", () => {
  beforeEach(() => setActivePinia(createPinia()));

  it("loads root policies without virtual ungrouped group", async () => {
    const store = useAlarmStore();
    await store.fetchTree("project-1", {});
    expect(store.tree.rootPolicies.map((item) => item.name)).toEqual(["根策略"]);
  });

  it("selects filtered result with snapshot", () => {
    const store = useAlarmStore();
    store.selectFiltered({ search: "温度" });
    expect(store.selection.mode).toBe("filtered");
  });
});
```

- [ ] **步骤 2：运行测试验证失败**

```powershell
pnpm --dir datacenter test -- alarm-store
```

预期：FAIL，提示 `fetchTree` 或 `selection` 不存在。

- [ ] **步骤 3：重构 store 状态**

在 `alarm.store.ts` 使用策略命名：

```ts
const groups = ref<AlarmPolicyGroup[]>([]);
const tree = ref<AlarmPolicyTree>({
  groups: [],
  rootPolicies: [],
  matchedPolicyCount: 0,
  totalPolicyCount: 0,
});
const selection = ref<AlarmBulkSelection>({ mode: "ids", policyIds: [] });
const editing = ref<AlarmPolicy | null>(null);
```

实现：

```ts
async function fetchGroups(projectId: string) {}
async function fetchTree(projectId: string, filters: Record<string, unknown> = {}) {}
function selectPolicy(id: string, selected: boolean) {}
function selectGroup(policyIds: string[], selected: boolean) {}
function selectFiltered(filters: Record<string, unknown>) {}
function clearSelection() {}
async function batchEnable(projectId: string) {}
async function batchDisable(projectId: string) {}
async function batchMove(projectId: string, groupId: string | null) {}
async function batchApplyConditions(projectId: string, conditions: AlarmCondition[]) {}
```

- [ ] **步骤 4：更新详情和试算方法**

把 `openForEdit`、`createRule`、`saveRule`、`removeRule`、`setRuleEnabled` 改为策略语义：

```ts
async function openPolicy(projectId: string, id: string) {}
async function createPolicy(projectId: string, data: AlarmPolicySave) {}
async function savePolicy(projectId: string, id: string, data: AlarmPolicyUpdate) {}
async function removePolicy(projectId: string, id: string) {}
async function setPolicyEnabled(projectId: string, id: string, isEnabled: boolean) {}
async function runTrial(projectId: string, id: string, payload: AlarmPolicyTrialPayload = {}) {}
async function fetchContract(projectId: string, id: string) {}
```

- [ ] **步骤 5：运行 store 测试**

```powershell
pnpm --dir datacenter test -- alarm-store
```

预期：PASS。

- [ ] **步骤 6：Commit**

```powershell
git add datacenter/src/stores/alarm.store.ts datacenter/tests/alarm-store.test.ts
git commit -m "feat(datacenter): 接入报警策略状态管理"
```

## 任务 8：左侧管理面板 UI

**文件：**
- 创建：`datacenter/src/components/alarm/AlarmManagementPanel.vue`
- 创建：`datacenter/src/components/alarm/AlarmPolicyTree.vue`
- 创建：`datacenter/src/components/alarm/AlarmPolicyFilters.vue`
- 创建：`datacenter/src/components/alarm/AlarmBulkToolbar.vue`

- [ ] **步骤 1：实现筛选组件**

`AlarmPolicyFilters.vue` props 和 emits：

```ts
const props = defineProps<{
  groups: AlarmPolicyGroup[];
  modelValue: Record<string, string | boolean | null>;
}>();

const emit = defineEmits<{
  "update:modelValue": [value: Record<string, string | boolean | null>];
  search: [value: Record<string, string | boolean | null>];
}>();
```

界面包含名称、分组、启停、级别、条件类型、模式筛选。文案短。

- [ ] **步骤 2：实现混合树**

`AlarmPolicyTree.vue` props：

```ts
const props = defineProps<{
  groups: AlarmPolicyGroup[];
  rootPolicies: AlarmPolicy[];
  policiesByGroup: Record<string, AlarmPolicy[]>;
  selectedPolicyId: string;
  selection: AlarmBulkSelection;
  loading: boolean;
}>();
```

规则：

- 根目录直接渲染分组节点和 `rootPolicies`。
- 不渲染“未分组”标题。
- 分组 checkbox 支持 checked 和 indeterminate。
- 全选状态由外层传入。

- [ ] **步骤 3：实现批量工具栏**

`AlarmBulkToolbar.vue` 展示：

- 已选数量。
- 全选筛选结果按钮。
- 清除选择按钮。
- 启用。
- 停用。
- 移动分组。
- 套用条件。

按钮使用 Element Plus 图标，icon-only 按钮加 `aria-label`。

- [ ] **步骤 4：组装管理面板**

`AlarmManagementPanel.vue`：

```vue
<template>
  <aside class="alarm-management-panel">
    <AlarmPolicyFilters ... />
    <AlarmBulkToolbar ... />
    <AlarmPolicyTree ... />
  </aside>
</template>
```

样式：

```css
.alarm-management-panel {
  min-width: 360px;
  width: 400px;
  max-width: 460px;
  border-right: 1px solid var(--dc-border);
  background: var(--dc-surface);
  display: grid;
  grid-template-rows: auto auto minmax(0, 1fr);
}
```

- [ ] **步骤 5：运行前端测试**

```powershell
pnpm --dir datacenter test -- alarm-store alarm-schema alarm-rule-model
```

预期：PASS。

- [ ] **步骤 6：Commit**

```powershell
git add datacenter/src/components/alarm/AlarmManagementPanel.vue datacenter/src/components/alarm/AlarmPolicyTree.vue datacenter/src/components/alarm/AlarmPolicyFilters.vue datacenter/src/components/alarm/AlarmBulkToolbar.vue
git commit -m "feat(datacenter): 实现报警策略管理面板"
```

## 任务 9：右侧策略编辑器和条件矩阵

**文件：**
- 创建：`datacenter/src/components/alarm/AlarmPolicyEditor.vue`
- 创建：`datacenter/src/components/alarm/AlarmConditionMatrix.vue`
- 创建：`datacenter/src/components/alarm/AlarmTargetSelector.vue`
- 创建：`datacenter/src/components/alarm/AlarmInputExpressionPanel.vue`
- 修改：`datacenter/src/components/alarm/AlarmTestPanel.vue`
- 修改：`datacenter/src/components/alarm/AlarmContractPanel.vue`

- [ ] **步骤 1：实现条件矩阵**

`AlarmConditionMatrix.vue` props：

```ts
const props = defineProps<{
  modelValue: AlarmCondition[];
  disabled?: boolean;
}>();
```

emits：

```ts
const emit = defineEmits<{
  "update:modelValue": [value: AlarmCondition[]];
}>();
```

表格列：启用、类型、名称、参数摘要、持续时间、回差、级别、操作。  
新增条件默认：

```ts
{
  id: crypto.randomUUID(),
  type: "H",
  name: "高限",
  isEnabled: true,
  severity: "warning",
  params: { limit: 0, hysteresis: 0, durationMs: 0 },
}
```

- [ ] **步骤 2：实现目标选择器**

`AlarmTargetSelector.vue` 使用 `DataPointPicker.vue` 选择 active 数据点，并在本组件内维护已选列表。  
不要改 `DataPointPicker.vue` 的既有单选行为。每次选择一个点后追加到 `modelValue`，已存在的 `datapointId` 不重复追加。

- [ ] **步骤 3：实现输入表达式面板**

`AlarmInputExpressionPanel.vue` 管理：

- 输入点列表。
- 每个输入点变量名 `key`。
- 派生表达式。

变量名只允许 `/^[A-Za-z_][A-Za-z0-9_]*$/`。

- [ ] **步骤 4：实现策略编辑器**

`AlarmPolicyEditor.vue` 区域：

- 头部：名称、分组、启用、保存、删除。
- 基本信息：描述、模式。
- `per_target`：展示 `AlarmTargetSelector`。
- `derived`：展示 `AlarmInputExpressionPanel`。
- 条件区：展示 `AlarmConditionMatrix`。
- 抑制和消息模板：复用旧面板。
- 底部：试算和契约。

- [ ] **步骤 5：适配试算和契约面板**

`AlarmTestPanel.vue` 展示：

- `triggered`
- `state`
- `triggeredConditions`
- `conditionResults`
- `diagnostics`

`AlarmContractPanel.vue` 展示 `schemaVersion = alarm.policy.v1`。

- [ ] **步骤 6：运行前端测试**

```powershell
pnpm --dir datacenter test -- alarm-schema alarm-rule-model
```

预期：PASS。

- [ ] **步骤 7：Commit**

```powershell
git add datacenter/src/components/alarm/AlarmPolicyEditor.vue datacenter/src/components/alarm/AlarmConditionMatrix.vue datacenter/src/components/alarm/AlarmTargetSelector.vue datacenter/src/components/alarm/AlarmInputExpressionPanel.vue datacenter/src/components/alarm/AlarmTestPanel.vue datacenter/src/components/alarm/AlarmContractPanel.vue
git commit -m "feat(datacenter): 实现报警策略编辑器"
```

## 任务 10：工作区编排、分组概览和批量设置视图

**文件：**
- 修改：`datacenter/src/components/alarm/AlarmWorkspace.vue`
- 创建：`datacenter/src/components/alarm/AlarmGroupOverview.vue`
- 创建：`datacenter/src/components/alarm/AlarmBulkEditor.vue`
- 修改或替换：`datacenter/src/components/alarm/CreateAlarmRuleDialog.vue`

- [ ] **步骤 1：实现分组概览**

`AlarmGroupOverview.vue` 展示：

- 分组名称。
- 启用状态。
- 组内策略数量。
- 各级别数量。
- 条件类型分布。
- 分组级启停和移动入口。

- [ ] **步骤 2：实现批量设置视图**

`AlarmBulkEditor.vue` 支持：

- 批量启用。
- 批量停用。
- 批量移动分组。
- 批量套用条件集。
- 批量设置高限。
- 批量设置低限。
- 批量设置报警级别。

批量套用条件集使用确认弹窗。文案明确“同类型条件会被覆盖，不存在则新增”。

- [ ] **步骤 3：改造创建入口**

将 `CreateAlarmRuleDialog.vue` 改成策略创建。  
提交 payload 使用 `draftToAlarmPolicySavePayload`。  
界面支持：

- 名称。
- 分组。
- 模式。
- 多目标点或输入表达式。
- 初始条件集。

- [ ] **步骤 4：改造工作区**

`AlarmWorkspace.vue`：

- 左侧使用 `AlarmManagementPanel`。
- 右侧根据状态切换：
  - 单策略：`AlarmPolicyEditor`
  - 分组：`AlarmGroupOverview`
  - 多选：`AlarmBulkEditor`
  - 空态：统计和创建入口
- 路由仍使用 `/debug/alarm/:objectId/:tab?`。
- `objectId` 指策略 ID。分组选中不写入该路由，保存在本地状态。

样式：

```css
.alarm-workspace {
  height: 100%;
  min-height: 0;
  display: grid;
  grid-template-columns: minmax(360px, 400px) minmax(0, 1fr);
  background: var(--dc-surface-subtle);
}
```

- [ ] **步骤 5：运行前端测试**

```powershell
pnpm --dir datacenter test -- alarm-store alarm-schema alarm-rule-model
```

预期：PASS。

- [ ] **步骤 6：Commit**

```powershell
git add datacenter/src/components/alarm/AlarmWorkspace.vue datacenter/src/components/alarm/AlarmGroupOverview.vue datacenter/src/components/alarm/AlarmBulkEditor.vue datacenter/src/components/alarm/CreateAlarmRuleDialog.vue
git commit -m "feat(datacenter): 完成报警策略工作区编排"
```

## 任务 11：契约检查联动和旧规则入口收口

**文件：**
- 修改：`data_service/internal/service/contract_check_service.go`
- 修改：`data_service/internal/service/contract_check_service_test.go`
- 检查：`datacenter/src` 中所有 `AlarmRule`、`alarm-rules`、`alarmRuleModel` 引用

- [ ] **步骤 1：搜索旧引用**

运行：

```powershell
Get-ChildItem -Path datacenter\src,data_service\internal -Recurse -File |
  Select-String -Pattern 'AlarmRule|alarm-rules|alarmRuleModel|data_alarm_rules'
```

预期：只剩旧表迁移、旧服务文件或已明确停用的引用。工作区、API、store 不再引用旧规则模型。

- [ ] **步骤 2：更新契约检查**

如果 `contract_check_service.go` 读取 `data_alarm_rules`，改为读取 `data_alarm_policies.contract`。  
契约检查摘要使用：

```go
type alarmContractSummary struct {
	ID        string
	Name      string
	Contract  map[string]any
	Enabled   bool
	UpdatedAt time.Time
}
```

输出 schemaVersion 期望为 `alarm.policy.v1`。

- [ ] **步骤 3：更新测试**

在 `contract_check_service_test.go` 把报警样例从单规则改为策略：

```go
alarms: []alarmContractSummary{
  {
    ID: "policy-1",
    Name: "温度策略",
    Enabled: true,
    Contract: map[string]any{"schemaVersion": "alarm.policy.v1"},
  },
}
```

- [ ] **步骤 4：运行后端服务测试**

```powershell
Set-Location data_service
go test ./internal/service -run 'ContractCheck|AlarmPolicy' -count=1
```

预期：PASS。

- [ ] **步骤 5：Commit**

```powershell
git add data_service/internal/service/contract_check_service.go data_service/internal/service/contract_check_service_test.go
git commit -m "feat(data_service): 契约检查接入报警策略"
```

## 任务 12：最终验证和构建

**文件：**
- 无固定修改。只允许修复本计划引入的问题。

- [ ] **步骤 1：后端全量测试**

```powershell
Set-Location data_service
go test ./...
```

预期：PASS。

- [ ] **步骤 2：前端相关测试**

```powershell
pnpm --dir datacenter test -- alarm-store alarm-rule-model alarm-schema
```

预期：PASS。

- [ ] **步骤 3：前端全量测试**

```powershell
pnpm --dir datacenter test
```

预期：PASS。

- [ ] **步骤 4：前端构建**

```powershell
pnpm --dir datacenter build
```

预期：PASS。允许已有 chunk size warning。

- [ ] **步骤 5：检查 typecheck 状态**

运行：

```powershell
pnpm --dir datacenter typecheck
```

预期：当前仓库可能仍因既有 `src/stores/data-catalog.store.ts` AxiosResponse 类型断言失败。  
如果失败内容只包含既有 `data-catalog.store.ts`，记录为未验证项，不在本计划内修复。  
如果出现本计划新增或修改文件的类型错误，必须修复。

- [ ] **步骤 6：检查工作区**

```powershell
git status --short
```

预期：只剩用户原有未跟踪文件，或本计划执行者明确生成的待提交文件。

- [ ] **步骤 7：Commit 验证修复**

如步骤 1-5 修了本计划引入的问题：

```powershell
git add <修复文件>
git commit -m "fix(datacenter): 修复报警策略验证问题"
```

如果没有修复文件，不提交。

## 自检清单

- 规格中的分组、策略、目标范围、条件集、批量设置、试算、契约均有任务覆盖。
- 未分组策略直接在根目录展示，由任务 8 和任务 12 验证。
- 筛选后全选使用筛选快照和排除项，由任务 6、任务 7、任务 8 覆盖。
- 左侧管理区默认 400px，由任务 8 和任务 10 覆盖。
- 后端接口全部使用 `/api/v1` 下的新 `/alarm-policy-groups` 和 `/alarm-policies`。
- 成功响应仍由 `response.WriteSuccess` 包裹为 `code/msg/data/reqId`。
- 计划不要求兼容旧 `/alarm-rules` 前端入口。
