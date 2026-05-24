# 数据中心报警工作区实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 交付数据中心报警工作区前后端闭环，包括最终规则模型、接口、列表、编辑、试算和契约预览。

**架构：** 后端以 `data_alarm_rules` 为事实来源，`AlarmRuleService` 负责规则校验、试算和契约生成，HTTP 层只做请求解析和响应包裹。前端采用规则 IDE 双栏，`alarm.store` 只调用最终报警 API，`AlarmWorkspace` 负责路由、草稿和组件编排。

**技术栈：** Go、PostgreSQL、Vue 3、Pinia、Vite、Element Plus、Zod、dayjs、Vitest。

---

## 文件结构

### 后端

- 修改：`data_service/internal/db/migrations/0010_alarm_rules.sql`  
  保持建表脚本与最新模型一致，供全新库初始化。
- 创建：`data_service/internal/db/migrations/0017_alarm_rule_final_model.sql`  
  将已有报警规则表升级到最终模型。
- 创建：`data_service/internal/db/migrations/0017_alarm_rule_final_model_down.sql`  
  回滚本次表结构调整。
- 修改：`data_service/internal/repository/alarm_rule_repository.go`  
  增加 `suppression`、`message_template`、目标数据点摘要字段读写。
- 修改：`data_service/internal/service/alarm_rule_service.go`  
  替换旧规则枚举，实现最终校验、试算、契约和草稿校验。
- 修改：`data_service/internal/http/handler/alarm_rule_handler.go`  
  增加启停和草稿校验 handler，调整请求字段。
- 修改：`data_service/internal/http/router/router.go`  
  增加 `PATCH enabled` 和 `POST validate-draft` 路由。
- 创建：`data_service/internal/service/alarm_rule_service_test.go`  
  覆盖规则校验、试算和契约生成。
- 创建：`data_service/tests/integration/alarm_rule_api_test.go`  
  覆盖报警规则 HTTP 闭环。

### 前端

- 修改：`datacenter/src/api/schemas/alarm.schema.ts`  
  定义最终 Zod schema。
- 修改：`datacenter/src/api/alarm.api.ts`  
  对接最终报警规则接口。
- 修改：`datacenter/src/stores/alarm.store.ts`  
  增加列表、详情、创建、保存、启停、删除、试算、契约和草稿校验状态。
- 重写：`datacenter/src/components/alarm/alarmRuleModel.ts`  
  定义前端草稿、默认值、序列化和反序列化。
- 重写：`datacenter/src/components/alarm/AlarmWorkspace.vue`  
  替换旧样例构建器为规则 IDE 双栏容器。
- 创建：`datacenter/src/components/alarm/AlarmRuleList.vue`
- 创建：`datacenter/src/components/alarm/AlarmEditorShell.vue`
- 创建：`datacenter/src/components/alarm/AlarmEditorHeader.vue`
- 创建：`datacenter/src/components/alarm/AlarmRuleForm.vue`
- 创建：`datacenter/src/components/alarm/AlarmSuppressionPanel.vue`
- 创建：`datacenter/src/components/alarm/AlarmMessageTemplatePanel.vue`
- 创建：`datacenter/src/components/alarm/AlarmTestPanel.vue`
- 创建：`datacenter/src/components/alarm/AlarmContractPanel.vue`
- 创建：`datacenter/src/components/alarm/CreateAlarmRuleDialog.vue`
- 创建：`datacenter/src/components/alarm/DataPointPicker.vue`
- 修改：`datacenter/tests/alarm-rule-model.test.ts`  
  改为最终模型测试。
- 创建：`datacenter/tests/alarm-schema.test.ts`
- 创建：`datacenter/tests/alarm-store.test.ts`

## 任务 1：数据库模型升级

**文件：**

- 修改：`data_service/internal/db/migrations/0010_alarm_rules.sql`
- 创建：`data_service/internal/db/migrations/0017_alarm_rule_final_model.sql`
- 创建：`data_service/internal/db/migrations/0017_alarm_rule_final_model_down.sql`
- 测试：`data_service/tests/integration/migration_test.go`

- [ ] **步骤 1：编写迁移断言**

在 `data_service/tests/integration/migration_test.go` 增加测试，执行迁移后检查最终字段和约束。

```go
func TestAlarmRuleFinalModelMigration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupMigrator(t, fixture.pool)
	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("migrate up failed: %v", err)
	}

	var suppressionType string
	var messageTemplateDefault string
	err := fixture.pool.QueryRow(ctx, `
        SELECT data_type, column_default
        FROM information_schema.columns
        WHERE table_schema = $1 AND table_name = 'data_alarm_rules' AND column_name = 'suppression'
    `, fixture.schemaName).Scan(&suppressionType, new(string))
	if err != nil {
		t.Fatalf("expected suppression column: %v", err)
	}
	if suppressionType != "jsonb" {
		t.Fatalf("suppression data_type = %q, want jsonb", suppressionType)
	}

	err = fixture.pool.QueryRow(ctx, `
        SELECT column_default
        FROM information_schema.columns
        WHERE table_schema = $1 AND table_name = 'data_alarm_rules' AND column_name = 'message_template'
    `, fixture.schemaName).Scan(&messageTemplateDefault)
	if err != nil {
		t.Fatalf("expected message_template column: %v", err)
	}
	if !strings.Contains(messageTemplateDefault, "''") {
		t.Fatalf("message_template default = %q, want empty string", messageTemplateDefault)
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./tests/integration -run TestAlarmRuleFinalModelMigration -count=1`  
目录：`data_service`  
预期：FAIL，报 `suppression column` 不存在。

- [ ] **步骤 3：编写迁移**

创建 `0017_alarm_rule_final_model.sql`：

```sql
ALTER TABLE data_alarm_rules
    ADD COLUMN IF NOT EXISTS suppression jsonb NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS message_template text NOT NULL DEFAULT '';

UPDATE data_alarm_rules
SET rule_type = CASE
    WHEN rule_type = 'threshold' THEN 'H'
    WHEN rule_type = 'range' THEN 'H'
    WHEN rule_type = 'expression' THEN 'cel'
    ELSE rule_type
END;

ALTER TABLE data_alarm_rules
    DROP CONSTRAINT IF EXISTS data_alarm_rules_rule_type_check,
    ADD CONSTRAINT data_alarm_rules_rule_type_check
        CHECK (rule_type IN ('H', 'L', 'HH', 'LL', 'deviation_high', 'deviation_low', 'rate_of_change', 'cel'));

ALTER TABLE data_alarm_rules
    DROP CONSTRAINT IF EXISTS data_alarm_rules_severity_check,
    ADD CONSTRAINT data_alarm_rules_severity_check
        CHECK (severity IN ('info', 'warning', 'major', 'critical'));

CREATE INDEX IF NOT EXISTS data_alarm_rules_project_updated_idx
    ON data_alarm_rules (project_id, updated_at DESC);
```

创建 `0017_alarm_rule_final_model_down.sql`：

```sql
DROP INDEX IF EXISTS data_alarm_rules_project_updated_idx;

ALTER TABLE data_alarm_rules
    DROP CONSTRAINT IF EXISTS data_alarm_rules_rule_type_check,
    ADD CONSTRAINT data_alarm_rules_rule_type_check
        CHECK (rule_type IN ('threshold', 'range', 'expression'));

ALTER TABLE data_alarm_rules
    DROP CONSTRAINT IF EXISTS data_alarm_rules_severity_check,
    ADD CONSTRAINT data_alarm_rules_severity_check
        CHECK (severity IN ('info', 'warning', 'critical'));

ALTER TABLE data_alarm_rules
    DROP COLUMN IF EXISTS message_template,
    DROP COLUMN IF EXISTS suppression;
```

同步修改 `0010_alarm_rules.sql` 的 `rule_type`、`severity`、`suppression`、`message_template` 和索引，使全新库与迁移后结构一致。

- [ ] **步骤 4：运行迁移测试**

运行：`go test ./tests/integration -run TestAlarmRuleFinalModelMigration -count=1`  
目录：`data_service`  
预期：PASS。

- [ ] **步骤 5：Commit**

```powershell
git add data_service/internal/db/migrations/0010_alarm_rules.sql data_service/internal/db/migrations/0017_alarm_rule_final_model.sql data_service/internal/db/migrations/0017_alarm_rule_final_model_down.sql data_service/tests/integration/migration_test.go
git commit -m "feat(data_service): 升级报警规则数据模型"
```

## 任务 2：后端规则校验、试算和契约

**文件：**

- 修改：`data_service/internal/service/alarm_rule_service.go`
- 创建：`data_service/internal/service/alarm_rule_service_test.go`

- [ ] **步骤 1：编写服务测试**

创建 `alarm_rule_service_test.go`，覆盖最终枚举、试算和契约。

```go
package service

import "testing"

func TestEvaluateAlarmRuleFinalTypes(t *testing.T) {
	tests := []struct {
		name      string
		ruleType  string
		condition map[string]any
		value     any
		context   map[string]any
		wantState string
	}{
		{"H triggered", "H", map[string]any{"limit": 80.0}, 81.0, nil, "triggered"},
		{"L not triggered", "L", map[string]any{"limit": 10.0}, 11.0, nil, "not_triggered"},
		{"deviation needs baseline", "deviation_high", map[string]any{"limit": 5.0}, 80.0, nil, "insufficient_input"},
		{"rate needs previous", "rate_of_change", map[string]any{"limit": 2.0, "windowMs": 60000.0}, 80.0, nil, "insufficient_input"},
		{"cel triggered", "cel", map[string]any{"expression": "value > 80"}, 81.0, nil, "triggered"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluateAlarmRule(tt.ruleType, tt.condition, tt.value, tt.context)
			if err != nil {
				t.Fatalf("evaluateAlarmRule() error = %v", err)
			}
			if result.State != tt.wantState {
				t.Fatalf("State = %q, want %q", result.State, tt.wantState)
			}
		})
	}
}

func TestBuildAlarmRuleContractFinalShape(t *testing.T) {
	input := normalizedAlarmRuleInput{
		Name:            "温度高报",
		TargetPath:      "metrics.temperature",
		RuleType:        "H",
		Condition:       map[string]any{"limit": 80.0},
		Severity:        "major",
		Suppression:     map[string]any{"enabled": false},
		MessageTemplate: "温度 {{value}}",
		IsEnabled:       true,
	}
	target := fakeAlarmDatapoint("dp-1", "metrics.temperature", "number")

	contract := buildAlarmRuleContract(input, target, "rule-1", "project-1")
	if contract["schemaVersion"] != "alarm.rule.v1" {
		t.Fatalf("schemaVersion = %v", contract["schemaVersion"])
	}
	if contract["ruleType"] != "H" {
		t.Fatalf("ruleType = %v", contract["ruleType"])
	}
	if contract["messageTemplate"] != "温度 {{value}}" {
		t.Fatalf("messageTemplate = %v", contract["messageTemplate"])
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/service -run 'TestEvaluateAlarmRuleFinalTypes|TestBuildAlarmRuleContractFinalShape' -count=1`  
目录：`data_service`  
预期：FAIL，报 `evaluateAlarmRule` 签名或字段不匹配。

- [ ] **步骤 3：实现最终服务模型**

在 `alarm_rule_service.go` 中更新枚举：

```go
var allowedAlarmRuleTypes = map[string]struct{}{
	"H": {},
	"L": {},
	"HH": {},
	"LL": {},
	"deviation_high": {},
	"deviation_low": {},
	"rate_of_change": {},
	"cel": {},
}

var allowedAlarmSeverities = map[string]struct{}{
	"info": {},
	"warning": {},
	"major": {},
	"critical": {},
}
```

增加结果类型：

```go
type AlarmRuleEvaluationResult struct {
	Triggered   bool           `json:"triggered"`
	State       string         `json:"state"`
	Diagnostics map[string]any `json:"diagnostics"`
}
```

实现 `evaluateAlarmRule(ruleType string, condition map[string]any, value any, context map[string]any) (*AlarmRuleEvaluationResult, error)`：

```go
switch ruleType {
case "H", "HH":
	return evaluateUpperLimit(ruleType, condition, value)
case "L", "LL":
	return evaluateLowerLimit(ruleType, condition, value)
case "deviation_high", "deviation_low":
	return evaluateDeviation(ruleType, condition, value, context)
case "rate_of_change":
	return evaluateRateOfChange(condition, value, context)
case "cel":
	return evaluateCEL(condition, value, context)
default:
	return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "ruleType 不受支持")
}
```

同步更新 `validateAlarmCondition`、`Test`、`Contract`、`buildAlarmRuleContract`，保证返回 `state`、`message`、`schemaVersion`、`suppression`、`messageTemplate`。

- [ ] **步骤 4：运行服务测试**

运行：`go test ./internal/service -run 'TestEvaluateAlarmRuleFinalTypes|TestBuildAlarmRuleContractFinalShape' -count=1`  
目录：`data_service`  
预期：PASS。

- [ ] **步骤 5：Commit**

```powershell
git add data_service/internal/service/alarm_rule_service.go data_service/internal/service/alarm_rule_service_test.go
git commit -m "feat(data_service): 完成报警规则校验和试算"
```

## 任务 3：后端仓储、Handler 和路由闭环

**文件：**

- 修改：`data_service/internal/repository/alarm_rule_repository.go`
- 修改：`data_service/internal/http/handler/alarm_rule_handler.go`
- 修改：`data_service/internal/http/router/router.go`
- 创建：`data_service/tests/integration/alarm_rule_api_test.go`

- [ ] **步骤 1：编写 HTTP 集成测试**

创建 `alarm_rule_api_test.go`。测试创建、列表、详情、启停、试算、契约和草稿校验。

```go
func TestAlarmRuleAPIFinalContract(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fixture := setupTestDatabase(t, ctx)
	migrator := setupMigrator(t, fixture.pool)
	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("migrate up failed: %v", err)
	}

	projectID := uuid.NewString()
	userID := uuid.NewString()
	secret := "alarm-rule-secret-01"
	server, token := startIntegrationServer(t, fixture.databaseURL, fixture.schemaName, secret, userID, projectID)

	datapointID := uuid.NewString()
	insertActiveDatapoint(t, ctx, fixture.pool, projectID, datapointID, "metrics.temperature", "number", userID)

	create := doJSONRequest(t, http.MethodPost, server.URL+"/api/v1/data/projects/"+projectID+"/alarm-rules", token, map[string]any{
		"name": "温度高报",
		"targetPath": "metrics.temperature",
		"ruleType": "H",
		"condition": map[string]any{"limit": 80, "hysteresis": 2, "durationMs": 30000},
		"severity": "major",
		"suppression": map[string]any{"enabled": false},
		"messageTemplate": "{{targetPath}} {{value}}",
	})
	var rule alarmRulePayload
	if err := json.Unmarshal(create.Data, &rule); err != nil {
		t.Fatalf("decode create response failed: %v", err)
	}
	if rule.RuleType != "H" || rule.Severity != "major" {
		t.Fatalf("unexpected rule: %#v", rule)
	}

	testEnvelope := doJSONRequest(t, http.MethodPost, server.URL+"/api/v1/data/projects/"+projectID+"/alarm-rules/"+rule.ID+"/test", token, map[string]any{
		"value": 82,
	})
	var trial alarmRuleTrialPayload
	if err := json.Unmarshal(testEnvelope.Data, &trial); err != nil {
		t.Fatalf("decode trial response failed: %v", err)
	}
	if trial.State != "triggered" {
		t.Fatalf("trial state = %q, want triggered", trial.State)
	}
}
```

测试文件内补充 `alarmRulePayload`、`alarmRuleTrialPayload`、`insertActiveDatapoint`、`startIntegrationServer`，写法参考 `compute_test.go`。

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./tests/integration -run TestAlarmRuleAPIFinalContract -count=1`  
目录：`data_service`  
预期：FAIL，报字段、路由或数据库列缺失。

- [ ] **步骤 3：更新仓储读写字段**

在 `AlarmRuleRecord`、`CreateAlarmRuleParams`、`UpdateAlarmRuleParams` 增加：

```go
TargetName       *string
TargetDataType   string
Suppression      map[string]any
MessageTemplate  string
```

SQL 查询通过 `LEFT JOIN data_points dp ON dp.id = target_datapoint_id` 取 `dp.name`、`dp.data_type`。`INSERT` 和 `UPDATE` 写入 `suppression`、`message_template`。

- [ ] **步骤 4：更新 Handler 和路由**

在 `router.go` 增加：

```go
mux.Handle(
	"PATCH /api/v1/data/projects/{projectId}/alarm-rules/{id}/enabled",
	middleware.Authenticate(opts.jwtValidator)(
		middleware.RequireCapability("project:write")(
			middleware.ErrorHandler(opts.alarmRuleHandler.ToggleEnabled),
		),
	),
)
mux.Handle(
	"POST /api/v1/data/projects/{projectId}/alarm-rules/validate-draft",
	middleware.Authenticate(opts.jwtValidator)(
		middleware.RequireCapability("project:read")(
			middleware.ErrorHandler(opts.alarmRuleHandler.ValidateDraft),
		),
	),
)
```

在 `alarm_rule_handler.go` 增加：

```go
func (h *AlarmRuleHandler) ToggleEnabled(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		IsEnabled bool `json:"isEnabled"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.ToggleEnabled(r.Context(), claims, r.PathValue("projectId"), r.PathValue("id"), request.IsEnabled)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}
```

`ValidateDraft` 解析创建字段，调用 `service.ValidateDraft`。

- [ ] **步骤 5：运行 HTTP 测试**

运行：`go test ./tests/integration -run TestAlarmRuleAPIFinalContract -count=1`  
目录：`data_service`  
预期：PASS。

- [ ] **步骤 6：Commit**

```powershell
git add data_service/internal/repository/alarm_rule_repository.go data_service/internal/http/handler/alarm_rule_handler.go data_service/internal/http/router/router.go data_service/tests/integration/alarm_rule_api_test.go
git commit -m "feat(data_service): 补齐报警规则接口闭环"
```

## 任务 4：前端 Schema、API 和模型

**文件：**

- 修改：`datacenter/src/api/schemas/alarm.schema.ts`
- 修改：`datacenter/src/api/alarm.api.ts`
- 重写：`datacenter/src/components/alarm/alarmRuleModel.ts`
- 修改：`datacenter/tests/alarm-rule-model.test.ts`
- 创建：`datacenter/tests/alarm-schema.test.ts`

- [ ] **步骤 1：编写前端模型测试**

将 `alarm-rule-model.test.ts` 改为最终模型测试：

```ts
import { describe, expect, test } from 'vitest'
import {
  alarmRuleTypeOptions,
  createAlarmDraft,
  draftToAlarmSavePayload,
  toAlarmDraft,
} from '../src/components/alarm/alarmRuleModel'

describe('alarm rule model', () => {
  test('规则类型覆盖最终枚举', () => {
    expect(alarmRuleTypeOptions.map((item) => item.value)).toEqual([
      'H',
      'L',
      'HH',
      'LL',
      'deviation_high',
      'deviation_low',
      'rate_of_change',
      'cel',
    ])
  })

  test('新建草稿序列化为最终保存 payload', () => {
    const draft = createAlarmDraft()
    draft.name = '温度高报'
    draft.targetPath = 'metrics.temperature'
    draft.ruleType = 'H'
    draft.condition.limit = 80
    draft.severity = 'major'

    expect(draftToAlarmSavePayload(draft)).toMatchObject({
      name: '温度高报',
      targetPath: 'metrics.temperature',
      ruleType: 'H',
      condition: { limit: 80 },
      severity: 'major',
    })
  })

  test('详情转草稿保留脏状态为 false', () => {
    const draft = toAlarmDraft({
      id: 'rule-1',
      projectId: 'project-1',
      name: '温度高报',
      targetDatapointId: 'dp-1',
      targetPath: 'metrics.temperature',
      targetDataType: 'number',
      ruleType: 'H',
      condition: { limit: 80 },
      severity: 'major',
      isEnabled: true,
      suppression: { enabled: false },
      messageTemplate: '',
      contract: {},
      createdAt: '2026-05-20T00:00:00Z',
      updatedAt: '2026-05-20T00:00:00Z',
    })
    expect(draft.dirty).toBe(false)
  })
})
```

创建 `alarm-schema.test.ts`：

```ts
import { describe, expect, test } from 'vitest'
import { AlarmRuleSchema, AlarmTrialResultSchema } from '../src/api/schemas/alarm.schema'

describe('alarm schema', () => {
  test('解析最终报警规则', () => {
    const rule = AlarmRuleSchema.parse({
      id: 'rule-1',
      projectId: 'project-1',
      name: '温度高报',
      targetDatapointId: 'dp-1',
      targetPath: 'metrics.temperature',
      targetDataType: 'number',
      ruleType: 'H',
      condition: { limit: 80 },
      severity: 'major',
      isEnabled: true,
      suppression: { enabled: false },
      messageTemplate: '',
      contract: {},
      createdAt: '2026-05-20T00:00:00Z',
      updatedAt: '2026-05-20T00:00:00Z',
    })
    expect(rule.ruleType).toBe('H')
  })

  test('解析试算输入不足状态', () => {
    const result = AlarmTrialResultSchema.parse({
      triggered: false,
      state: 'insufficient_input',
      severity: 'warning',
      ruleType: 'deviation_high',
      targetPath: 'metrics.temperature',
      diagnostics: {},
    })
    expect(result.state).toBe('insufficient_input')
  })
})
```

- [ ] **步骤 2：运行前端测试验证失败**

运行：`pnpm --dir datacenter test -- alarm-rule-model alarm-schema`  
预期：FAIL，报导出不存在或 schema 不匹配。

- [ ] **步骤 3：实现 Schema、API 和模型**

`alarm.schema.ts` 定义：

```ts
export const AlarmRuleTypeSchema = z.enum([
  'H',
  'L',
  'HH',
  'LL',
  'deviation_high',
  'deviation_low',
  'rate_of_change',
  'cel',
])

export const AlarmSeveritySchema = z.enum(['info', 'warning', 'major', 'critical'])
export const AlarmTrialStateSchema = z.enum(['triggered', 'not_triggered', 'insufficient_input'])
```

`alarm.api.ts` 增加 `toggleAlarmRule`、`testAlarmRule`、`getAlarmRuleContract`、`validateAlarmRuleDraft`。请求路径使用设计文档中的最终路径。

`alarmRuleModel.ts` 导出：

```ts
export const alarmRuleTypeOptions = [
  { value: 'H', label: 'H 高限' },
  { value: 'L', label: 'L 低限' },
  { value: 'HH', label: 'HH 高高限' },
  { value: 'LL', label: 'LL 低低限' },
  { value: 'deviation_high', label: '大偏差' },
  { value: 'deviation_low', label: '小偏差' },
  { value: 'rate_of_change', label: '变化率' },
  { value: 'cel', label: 'CEL' },
] as const
```

实现 `createAlarmDraft`、`toAlarmDraft`、`draftToAlarmSavePayload`，不导出旧样例数组。

- [ ] **步骤 4：运行前端模型测试**

运行：`pnpm --dir datacenter test -- alarm-rule-model alarm-schema`  
预期：PASS。

- [ ] **步骤 5：Commit**

```powershell
git add datacenter/src/api/schemas/alarm.schema.ts datacenter/src/api/alarm.api.ts datacenter/src/components/alarm/alarmRuleModel.ts datacenter/tests/alarm-rule-model.test.ts datacenter/tests/alarm-schema.test.ts
git commit -m "feat(datacenter): 定义报警规则前端模型"
```

## 任务 5：前端 Store 与工作区路由状态

**文件：**

- 修改：`datacenter/src/stores/alarm.store.ts`
- 重写：`datacenter/src/components/alarm/AlarmWorkspace.vue`
- 创建：`datacenter/tests/alarm-store.test.ts`

- [ ] **步骤 1：编写 Store 测试**

创建 `alarm-store.test.ts`，用 mock API 验证列表、详情和保存。

```ts
import { beforeEach, describe, expect, test, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

vi.mock('../src/api/alarm.api', () => ({
  getAlarmRules: vi.fn(async () => ({
    list: [{ id: 'rule-1', name: '温度高报' }],
    pagination: { total: 1 },
  })),
  getAlarmRule: vi.fn(async () => ({
    id: 'rule-1',
    name: '温度高报',
    ruleType: 'H',
    severity: 'major',
    isEnabled: true,
    condition: {},
    suppression: {},
    contract: {},
    targetPath: 'metrics.temperature',
    targetDatapointId: 'dp-1',
    targetDataType: 'number',
    createdAt: '',
    updatedAt: '',
  })),
  updateAlarmRule: vi.fn(async () => ({
    id: 'rule-1',
    name: '温度高报2',
    ruleType: 'H',
    severity: 'major',
    isEnabled: true,
    condition: {},
    suppression: {},
    contract: {},
    targetPath: 'metrics.temperature',
    targetDatapointId: 'dp-1',
    targetDataType: 'number',
    createdAt: '',
    updatedAt: '',
  })),
}))

describe('alarm store', () => {
  beforeEach(() => setActivePinia(createPinia()))

  test('拉取列表并记录 total', async () => {
    const { useAlarmStore } = await import('../src/stores/alarm.store')
    const store = useAlarmStore()
    await store.fetchList('project-1')
    expect(store.total).toBe(1)
    expect(store.list[0].id).toBe('rule-1')
  })
})
```

- [ ] **步骤 2：运行测试验证失败**

运行：`pnpm --dir datacenter test -- alarm-store`  
预期：FAIL，报 store 方法或字段不完整。

- [ ] **步骤 3：实现 Store**

`alarm.store.ts` 对齐 `compute.store.ts` 风格，暴露：

```ts
fetchList(projectId, params)
openForEdit(projectId, id)
createRule(projectId, data)
saveRule(projectId, id, data)
removeRule(projectId, id)
setRuleEnabled(projectId, id, isEnabled)
runTrial(projectId, id, payload)
fetchContract(projectId, id)
validateDraft(projectId, payload)
```

状态包括 `list`、`total`、`loading`、`listError`、`editing`、`detailLoading`、`detailError`、`creating`、`saving`、`deleting`、`trial`、`contract`。

- [ ] **步骤 4：重写工作区容器**

`AlarmWorkspace.vue` 只做布局、路由和草稿编排。保持事件名清楚：

```vue
<AlarmRuleList
  :rules="alarmStore.list"
  :selected-rule-id="selectedRuleId"
  :loading="alarmStore.loading"
  :error="alarmStore.listError"
  @select-rule="selectRule"
  @create-rule="showCreateDialog = true"
/>
<AlarmEditorShell
  :project-id="projectId"
  :draft="activeDraft"
  :loading="alarmStore.detailLoading"
  :saving="alarmStore.saving"
  @save="saveActiveDraft"
  @delete-rule="deleteRule"
  @toggle-enabled="toggleEnabled"
/>
```

- [ ] **步骤 5：运行测试**

运行：`pnpm --dir datacenter test -- alarm-store`  
预期：PASS。

- [ ] **步骤 6：Commit**

```powershell
git add datacenter/src/stores/alarm.store.ts datacenter/src/components/alarm/AlarmWorkspace.vue datacenter/tests/alarm-store.test.ts
git commit -m "feat(datacenter): 接入报警工作区状态"
```

## 任务 6：报警列表与新建弹窗

**文件：**

- 创建：`datacenter/src/components/alarm/AlarmRuleList.vue`
- 创建：`datacenter/src/components/alarm/CreateAlarmRuleDialog.vue`
- 创建：`datacenter/src/components/alarm/DataPointPicker.vue`

- [ ] **步骤 1：编写基础组件代码**

创建 `AlarmRuleList.vue`，实现搜索、筛选、列表和分页事件：

```vue
<template>
  <aside class="alarm-rule-list">
    <header class="alarm-rule-list__head">
      <input v-model="localSearch" placeholder="搜索规则或点位" @input="emitFilter" />
      <button type="button" title="新建规则" aria-label="新建规则" @click="$emit('create-rule')">
        <IconTablerPlus />
      </button>
    </header>
    <div class="alarm-rule-list__filters">
      <select v-model="enabled" @change="emitFilter">
        <option value="">全部</option>
        <option value="true">启用</option>
        <option value="false">停用</option>
      </select>
      <select v-model="severity" @change="emitFilter">
        <option value="">全部级别</option>
        <option value="info">info</option>
        <option value="warning">warning</option>
        <option value="major">major</option>
        <option value="critical">critical</option>
      </select>
    </div>
    <EmptyState
      v-if="!loading && !rules.length"
      icon-name="bell"
      title="暂无报警规则"
      description="新建规则后会显示在这里。"
    />
    <button
      v-for="rule in rules"
      v-else
      :key="rule.id"
      type="button"
      class="alarm-rule-list__row"
      :class="{ 'is-active': rule.id === selectedRuleId }"
      @click="$emit('select-rule', String(rule.id))"
    >
      <i :class="`is-${rule.severity}`"></i>
      <strong>{{ rule.name }}</strong>
      <code>{{ rule.targetPath }}</code>
      <StatusBadge
        :text="rule.isEnabled ? '启用' : '停用'"
        :tone="rule.isEnabled ? 'success' : 'neutral'"
      />
    </button>
  </aside>
</template>
```

`CreateAlarmRuleDialog.vue` 使用 `DataPointPicker` 选择目标，提交 `draftToAlarmSavePayload(createAlarmDraft())` 结果。

- [ ] **步骤 2：连接工作区**

在 `AlarmWorkspace.vue` 绑定 `@filter-change` 调用 `alarmStore.fetchList(projectId, params)`，绑定 `@submit` 调用 `alarmStore.createRule`。

- [ ] **步骤 3：运行构建**

运行：`pnpm --dir datacenter build`  
预期：PASS。

- [ ] **步骤 4：Commit**

```powershell
git add datacenter/src/components/alarm/AlarmRuleList.vue datacenter/src/components/alarm/CreateAlarmRuleDialog.vue datacenter/src/components/alarm/DataPointPicker.vue datacenter/src/components/alarm/AlarmWorkspace.vue
git commit -m "feat(datacenter): 实现报警规则列表和新建入口"
```

## 任务 7：报警编辑器表单

**文件：**

- 创建：`datacenter/src/components/alarm/AlarmEditorShell.vue`
- 创建：`datacenter/src/components/alarm/AlarmEditorHeader.vue`
- 创建：`datacenter/src/components/alarm/AlarmRuleForm.vue`
- 创建：`datacenter/src/components/alarm/AlarmSuppressionPanel.vue`
- 创建：`datacenter/src/components/alarm/AlarmMessageTemplatePanel.vue`

- [ ] **步骤 1：实现编辑器壳**

`AlarmEditorShell.vue` 使用头条、表单和底部 tab 容器：

```vue
<template>
  <section class="alarm-editor">
    <EmptyState
      v-if="!draft"
      icon-name="bell"
      title="选择报警规则"
      description="从左侧选择规则或新建规则。"
    />
    <template v-else>
      <AlarmEditorHeader
        :draft="draft"
        :saving="saving"
        @save="$emit('save')"
        @delete-rule="$emit('delete-rule', draft.id)"
        @toggle-enabled="$emit('toggle-enabled', draft.id, !draft.isEnabled)"
        @mark-dirty="$emit('mark-dirty')"
      />
      <main class="alarm-editor__main">
        <AlarmRuleForm :draft="draft" @mark-dirty="$emit('mark-dirty')" />
      </main>
      <footer class="alarm-editor__bottom">
        <button
          type="button"
          :class="{ 'is-active': activePanel === 'config' }"
          @click="activePanel = 'config'"
        >
          规则配置
        </button>
        <button
          type="button"
          :class="{ 'is-active': activePanel === 'test' }"
          @click="$emit('switch-panel', 'test')"
        >
          试算
        </button>
        <button
          type="button"
          :class="{ 'is-active': activePanel === 'contract' }"
          @click="$emit('switch-panel', 'contract')"
        >
          契约
        </button>
        <AlarmSuppressionPanel
          v-if="activePanel === 'config'"
          :draft="draft"
          @mark-dirty="$emit('mark-dirty')"
        />
        <AlarmMessageTemplatePanel
          v-if="activePanel === 'config'"
          :draft="draft"
          @mark-dirty="$emit('mark-dirty')"
        />
      </footer>
    </template>
  </section>
</template>
```

- [ ] **步骤 2：实现动态规则字段**

`AlarmRuleForm.vue` 按 `draft.ruleType` 渲染字段：

```vue
<template v-if="['H', 'HH', 'L', 'LL'].includes(draft.ruleType)">
  <label
    ><span>{{ draft.ruleType.includes('H') ? '上限' : '下限' }}</span
    ><input v-model.number="draft.condition.limit" type="number" @input="$emit('mark-dirty')"
  /></label>
  <label
    ><span>回差</span
    ><input
      v-model.number="draft.condition.hysteresis"
      type="number"
      min="0"
      @input="$emit('mark-dirty')"
  /></label>
  <label
    ><span>持续毫秒</span
    ><input
      v-model.number="draft.condition.durationMs"
      type="number"
      min="0"
      @input="$emit('mark-dirty')"
  /></label>
</template>
<template v-else-if="draft.ruleType === 'cel'">
  <MonacoEditor
    v-model="draft.condition.expression"
    language="javascript"
    height="180px"
    @change="$emit('mark-dirty')"
  />
</template>
```

- [ ] **步骤 3：连接保存、启停、删除**

在 `AlarmWorkspace.vue` 中实现：

```ts
async function saveActiveDraft() {
  if (!activeDraft.value) return
  const saved = await alarmStore.saveRule(
    projectId.value,
    activeDraft.value.id,
    draftToAlarmSavePayload(activeDraft.value),
  )
  drafts.value[saved.id] = toAlarmDraft(saved)
}
```

启停调用 `setRuleEnabled`，删除调用 `removeRule` 并切换到下一条规则。

- [ ] **步骤 4：运行构建**

运行：`pnpm --dir datacenter build`  
预期：PASS。

- [ ] **步骤 5：Commit**

```powershell
git add datacenter/src/components/alarm/AlarmEditorShell.vue datacenter/src/components/alarm/AlarmEditorHeader.vue datacenter/src/components/alarm/AlarmRuleForm.vue datacenter/src/components/alarm/AlarmSuppressionPanel.vue datacenter/src/components/alarm/AlarmMessageTemplatePanel.vue datacenter/src/components/alarm/AlarmWorkspace.vue
git commit -m "feat(datacenter): 实现报警规则编辑器"
```

## 任务 8：试算、契约和跨模块入口

**文件：**

- 创建：`datacenter/src/components/alarm/AlarmTestPanel.vue`
- 创建：`datacenter/src/components/alarm/AlarmContractPanel.vue`
- 修改：`datacenter/src/components/alarm/AlarmEditorShell.vue`
- 修改：`datacenter/src/components/alarm/AlarmWorkspace.vue`

- [ ] **步骤 1：实现试算面板**

`AlarmTestPanel.vue`：

```vue
<template>
  <section class="alarm-test-panel">
    <textarea v-model="sampleText" aria-label="试算样本 JSON"></textarea>
    <button type="button" :disabled="running" @click="run">试算</button>
    <div v-if="result" :class="`alarm-test-panel__result is-${result.state}`">
      <strong>{{ stateText }}</strong>
      <span>{{ result.message || '-' }}</span>
      <pre>{{ JSON.stringify(result.diagnostics, null, 2) }}</pre>
    </div>
  </section>
</template>
```

`run` 解析 JSON，调用 `alarmStore.runTrial(projectId, draft.id, payload)`。

- [ ] **步骤 2：实现契约面板**

`AlarmContractPanel.vue`：

```vue
<template>
  <section class="alarm-contract-panel">
    <button type="button" title="复制契约 JSON" aria-label="复制契约 JSON" @click="copyContract">
      <IconTablerCopy />
    </button>
    <pre>{{ formattedContract }}</pre>
  </section>
</template>
```

进入 `contract` tab 时调用 `alarmStore.fetchContract(projectId, draft.id)`。

- [ ] **步骤 3：连接路由 tab**

`AlarmWorkspace.vue` 根据 `route.params.tab` 设置底部 tab。切换 tab 时 push：

```ts
void router.push({
  path: `${alarmBasePath.value}/${activeDraft.value.id}/${tab}`,
  query: route.query,
})
```

- [ ] **步骤 4：连接“打开目标数据点”和“检查当前规则”**

`AlarmEditorHeader.vue` 发出：

```ts
emit('open-target', draft.targetDatapointId)
emit('check-current', draft.id)
```

`AlarmWorkspace.vue` 中：

```ts
function openTargetDatapoint(id: string) {
  void router.push({ path: `${debugPrefix.value}/datapoint/${id}`, query: route.query })
}
```

`check-current` 先触发已有契约检查弹窗入口；若当前弹窗只支持项目级，传递 `{ objectType: "alarmRule", objectId: id }` 给 contract store 后再打开。

- [ ] **步骤 5：运行构建**

运行：`pnpm --dir datacenter build`  
预期：PASS。

- [ ] **步骤 6：Commit**

```powershell
git add datacenter/src/components/alarm/AlarmTestPanel.vue datacenter/src/components/alarm/AlarmContractPanel.vue datacenter/src/components/alarm/AlarmEditorShell.vue datacenter/src/components/alarm/AlarmWorkspace.vue
git commit -m "feat(datacenter): 完成报警试算和契约预览"
```

## 任务 9：全量验证与收尾

**文件：**

- 修改：按验证发现的问题精确修改相关文件。

- [ ] **步骤 1：运行后端全量测试**

运行：`go test ./...`  
目录：`data_service`  
预期：PASS。

- [ ] **步骤 2：运行前端测试和构建**

运行：

```powershell
pnpm --dir datacenter test
pnpm --dir datacenter build
```

预期：两条命令都 PASS。

- [ ] **步骤 3：静态检查旧样例入口**

运行：

```powershell
rg -n "alarmRuleSamples|alarmRuntimeBoundaryNotes|规则分组|Release Contract Gate" datacenter/src/components/alarm datacenter/tests
```

预期：无输出。若有输出，只删除本轮替换后不再引用的旧样例标识。

- [ ] **步骤 4：手工验收**

启动前端和后端后验收：

- `/debug/alarm`
- `/debug/alarm/:objectId`
- `/debug/alarm/:objectId/test`
- `/debug/alarm/:objectId/contract`

确认列表搜索、筛选、分页、新建、编辑、保存、启停、删除、试算、契约 JSON、刷新恢复都可用。

- [ ] **步骤 5：最终 Commit**

```powershell
git add data_service datacenter
git commit -m "feat(datacenter): 完成报警工作区前后端闭环"
```

如果任务 1 到任务 8 已经逐项提交，最终没有新增改动时跳过本步骤。

## 自检记录

- 规格覆盖度：已覆盖设计文档中的规则模型、API、后端校验、契约、数据库、前端 UI、数据流、测试与验收。
- 红旗词扫描：已按计划规范检查模糊表述，正文没有未完成标记。
- 类型一致性：前后端统一使用 `ruleType`、`severity`、`isEnabled`、`suppression`、`messageTemplate`、`targetDatapointId`、`targetDataType`。
