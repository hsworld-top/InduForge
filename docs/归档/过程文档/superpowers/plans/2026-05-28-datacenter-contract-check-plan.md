# 数据中心契约检查实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 将现有数据契约检查从基础 dry-run 和静态弹窗升级为数据中心可独立运行、可供运维发布流程调用的契约检查中心。

**架构：** 后端复用现有 `data_contract_check_runs`、`ContractCheckService` 和接口入口，升级结果模型为 `progress/summary/issues/blocking`，并用规则流水线覆盖数据点、接入源、查询、协议对象、计算、报警和 `DataDomainArtifact` dry-run。前端复用左下角入口，替换静态弹窗为不强遮罩的悬浮检查中心，统一展示进度、汇总、筛选、问题详情和跳转修复。当前开发边界只包含 `data_service` 和 `datacenter`；`dev_core` 代码调用点只作为接口契约和后续集成说明，不在本计划中实现。

**技术栈：** Go、pgx、PostgreSQL jsonb、Vue 3、Pinia、Zod、Element Plus、pnpm、Go test。

---

## 文件结构

- 修改：`data_service/internal/service/contract_check_service.go`
  - 定义新结果模型、检查输入、上下文加载、规则流水线、汇总和过滤逻辑。
- 修改：`data_service/internal/service/contract_check_service_test.go`
  - 覆盖级别汇总、阻断判断、过滤、规则输出和历史记录转换。
- 修改：`data_service/internal/http/handler/contract_check_handler.go`
  - 解析 `scope/mode/module/objectType/objectId`，返回新版结果。
- 修改：`data_service/internal/repository/contract_check_repository.go`
  - 保存和读取新版 `summary/result`，补齐 `mode/module/blocking` 字段的读写。
- 修改：`data_service/internal/db/migrations/0011_contract_check_runs.sql`
  - 当前开发阶段可重建库，直接压实表结构，加入 `mode`、`module`、`blocking`。
- 修改：`data_service/internal/db/migrations/0011_contract_check_runs_down.sql`
  - 与 up 文件保持配对删除。
- 修改：`data_service/internal/app/server.go`
  - 为 `ContractCheckService` 注入连接、MQTT、Kafka、HTTP、WebSocket、实时库、OPC UA、Modbus、S7、快照仓储或服务。
- 修改：`datacenter/src/api/schemas/contract-check.schema.ts`
  - 定义新版 Zod schema 与类型。
- 修改：`datacenter/src/api/contract-check.api.ts`
  - 支持新版 run 请求参数和 `/runs` 历史接口。
- 修改：`datacenter/src/stores/contract-check.store.ts`
  - 管理最近结果、运行状态、筛选条件、选中问题和入口图标状态。
- 删除：`datacenter/src/components/contract/dataContractCheck.ts`
  - 静态本地检查清单不再作为数据源。
- 重写：`datacenter/src/components/contract/DataContractCheckDialog.vue`
  - 改造成悬浮检查中心。
- 修改：`datacenter/src/views/DataCenterNew.vue`
  - 左下角入口接入 store 状态色，打开悬浮检查中心。
- 修改：`docs/统一REST接口规范与清单.md`
  - 记录运维发布流程调用数据中心契约检查的请求体和结果门禁字段。
- 修改：`docs/superpowers/specs/2026-05-28-datacenter-contract-check-design.md`
  - 同步 `DataDomainArtifact` 与发布部署边界。

---

## 任务 1：升级后端契约检查结果模型

**文件：**

- 修改：`data_service/internal/service/contract_check_service.go`
- 修改：`data_service/internal/service/contract_check_service_test.go`

- [ ] **步骤 1：编写失败的模型汇总测试**

在 `data_service/internal/service/contract_check_service_test.go` 中增加：

```go
func TestBuildContractCheckRunResultBlockingSummary(t *testing.T) {
	result := buildContractCheckRunResult("project-1", ContractCheckInput{
		Scope: "project",
		Mode:  "contract",
	}, []ContractCheckIssue{
		{
			Status:        "warning",
			Blocking:      false,
			Category:      "diagnostic",
			RuntimeImpact: "data_quality_risk",
			Module:        "http",
			ObjectType:    "httpRequest",
			Title:         "HTTP 请求从未成功发送",
		},
		{
			Status:        "failed",
			Blocking:      true,
			Category:      "reference",
			RuntimeImpact: "contract_blocker",
			Module:        "datapoint",
			ObjectType:    "datapoint",
			ObjectID:      "dp-1",
			Title:         "数据点来源不存在",
		},
	})

	if result.Status != "failed" {
		t.Fatalf("status = %s, want failed", result.Status)
	}
	if !result.Blocking {
		t.Fatalf("blocking = false, want true")
	}
	if result.Summary.Failed != 1 || result.Summary.Warning != 1 || result.Summary.Blocking != 1 {
		t.Fatalf("summary mismatch: %#v", result.Summary)
	}
	if result.Progress.CheckedCount != 2 || result.Progress.TotalCount != 2 {
		t.Fatalf("progress mismatch: %#v", result.Progress)
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：

```powershell
go test ./internal/service -run TestBuildContractCheckRunResultBlockingSummary -count=1
```

预期：FAIL，报 `undefined: ContractCheckIssue` 或 `undefined: buildContractCheckRunResult`。

- [ ] **步骤 3：实现新版类型和汇总函数**

在 `data_service/internal/service/contract_check_service.go` 中替换旧结果结构，保留 JSON 字段向前端输出：

```go
type ContractCheckInput struct {
	Scope      string
	Mode       string
	Module     string
	ObjectType string
	ObjectID   string
	Trigger    string
}

type ContractCheckRunResult struct {
	ID         string                  `json:"id"`
	ProjectID  string                  `json:"projectId"`
	Scope      string                  `json:"scope"`
	Mode       string                  `json:"mode"`
	Trigger    string                  `json:"trigger,omitempty"`
	Status     string                  `json:"status"`
	Blocking   bool                    `json:"blocking"`
	StartedAt  time.Time               `json:"startedAt"`
	FinishedAt *time.Time              `json:"finishedAt,omitempty"`
	Progress   ContractCheckProgress   `json:"progress"`
	Summary    ContractCheckSummary    `json:"summary"`
	Issues     []ContractCheckIssue    `json:"issues"`
}

type ContractCheckProgress struct {
	CurrentStage      string `json:"currentStage"`
	CurrentModule     string `json:"currentModule,omitempty"`
	CurrentObjectName string `json:"currentObjectName,omitempty"`
	CheckedCount      int    `json:"checkedCount"`
	TotalCount        int    `json:"totalCount"`
}

type ContractCheckSummary struct {
	Failed   int `json:"failed"`
	Pending  int `json:"pending"`
	Warning  int `json:"warning"`
	Passed   int `json:"passed"`
	Blocking int `json:"blocking"`
}

type ContractCheckIssue struct {
	ID            string                   `json:"id"`
	Status        string                   `json:"status"`
	Blocking      bool                     `json:"blocking"`
	Category      string                   `json:"category"`
	RuntimeImpact string                   `json:"runtimeImpact"`
	Module        string                   `json:"module"`
	ObjectType    string                   `json:"objectType"`
	ObjectID      string                   `json:"objectId,omitempty"`
	ObjectName    string                   `json:"objectName,omitempty"`
	Title         string                   `json:"title"`
	Detail        string                   `json:"detail,omitempty"`
	Suggestion    string                   `json:"suggestion,omitempty"`
	JumpTarget    *ContractCheckJumpTarget `json:"jumpTarget,omitempty"`
}

type ContractCheckJumpTarget struct {
	RouteName string            `json:"routeName"`
	Params    map[string]string `json:"params"`
	Query     map[string]string `json:"query,omitempty"`
}

func buildContractCheckRunResult(projectID string, input ContractCheckInput, issues []ContractCheckIssue) *ContractCheckRunResult {
	now := time.Now().UTC()
	scope := normalizeContractCheckScope(input.Scope)
	mode := normalizeContractCheckMode(input.Mode)
	summary := ContractCheckSummary{}
	status := "passed"
	blocking := false
	for index := range issues {
		issue := &issues[index]
		issue.Status = normalizeContractIssueStatus(issue.Status)
		if strings.TrimSpace(issue.ID) == "" {
			issue.ID = contractIssueID(issue.Module, issue.ObjectType, issue.ObjectID, issue.Title, index)
		}
		switch issue.Status {
		case "failed":
			summary.Failed++
			status = "failed"
		case "pending":
			summary.Pending++
			if status != "failed" {
				status = "pending"
			}
		case "warning":
			summary.Warning++
			if status == "passed" {
				status = "warning"
			}
		default:
			summary.Passed++
		}
		if issue.Blocking {
			summary.Blocking++
			blocking = true
		}
	}
	if len(issues) == 0 {
		summary.Passed = 1
	}
	return &ContractCheckRunResult{
		ID:        contractIssueID(projectID, scope, mode, now.Format(time.RFC3339Nano), 0),
		ProjectID: projectID,
		Scope:     scope,
		Mode:      mode,
		Trigger:   strings.TrimSpace(input.Trigger),
		Status:    status,
		Blocking:  blocking,
		StartedAt: now,
		FinishedAt: &now,
		Progress: ContractCheckProgress{
			CurrentStage: "汇总检查结果",
			CheckedCount: len(issues),
			TotalCount:   len(issues),
		},
		Summary: summary,
		Issues:  issues,
	}
}
```

- [ ] **步骤 4：实现归一化辅助函数**

继续在同文件增加：

```go
func normalizeContractCheckScope(scope string) string {
	switch strings.TrimSpace(scope) {
	case "module", "object":
		return strings.TrimSpace(scope)
	default:
		return "project"
	}
}

func normalizeContractCheckMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case "contract_with_diagnostics":
		return "contract_with_diagnostics"
	default:
		return "contract"
	}
}

func normalizeContractIssueStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "failed", "pending", "warning":
		return strings.TrimSpace(status)
	default:
		return "passed"
	}
}

func contractIssueID(parts ...any) string {
	raw := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(toString(part))
		if value != "" {
			raw = append(raw, value)
		}
	}
	if len(raw) == 0 {
		return "contract-check"
	}
	return strings.NewReplacer(" ", "-", "/", "-", "\\", "-", ":", "-").Replace(strings.Join(raw, "-"))
}
```

- [ ] **步骤 5：运行测试验证通过**

运行：

```powershell
go test ./internal/service -run TestBuildContractCheckRunResultBlockingSummary -count=1
```

预期：PASS。

- [ ] **步骤 6：Commit**

```powershell
git add data_service/internal/service/contract_check_service.go data_service/internal/service/contract_check_service_test.go
git commit -m "feat(data): 升级契约检查结果模型"
```

---

## 任务 2：压实契约检查运行记录表和仓储

**文件：**

- 修改：`data_service/internal/db/migrations/0011_contract_check_runs.sql`
- 修改：`data_service/internal/db/migrations/0011_contract_check_runs_down.sql`
- 修改：`data_service/internal/repository/contract_check_repository.go`
- 修改：`data_service/internal/service/contract_check_service.go`
- 修改：`data_service/internal/service/contract_check_service_test.go`

- [ ] **步骤 1：编写历史记录转换失败测试**

在 `contract_check_service_test.go` 增加：

```go
func TestToContractCheckRunIncludesModeAndBlocking(t *testing.T) {
	record := repository.ContractCheckRunRecord{
		ID:        11,
		ProjectID: "project-1",
		Scope:     "module",
		Mode:      "contract_with_diagnostics",
		Module:    stringPtr("http"),
		Status:    "warning",
		Blocking:  false,
		Summary:   json.RawMessage(`{"failed":0,"pending":0,"warning":1,"passed":2,"blocking":0}`),
		Result:    json.RawMessage(`{"status":"warning","blocking":false,"issues":[]}`),
		CreatedAt: time.Date(2026, 5, 28, 10, 0, 0, 0, time.UTC),
	}

	run, err := toContractCheckRun(record)
	if err != nil {
		t.Fatalf("toContractCheckRun() error = %v", err)
	}
	if run.Mode != "contract_with_diagnostics" || run.Module == nil || *run.Module != "http" {
		t.Fatalf("run scope fields mismatch: %#v", run)
	}
	if run.Blocking {
		t.Fatalf("blocking = true, want false")
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

```powershell
go test ./internal/service -run TestToContractCheckRunIncludesModeAndBlocking -count=1
```

预期：FAIL，仓储记录或服务运行记录缺少 `Mode/Module/Blocking`。

- [ ] **步骤 3：压实迁移表结构**

修改 `data_service/internal/db/migrations/0011_contract_check_runs.sql`：

```sql
CREATE TABLE IF NOT EXISTS data_contract_check_runs (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    project_id uuid NOT NULL,
    scope text NOT NULL DEFAULT 'project' CHECK (scope IN ('project', 'module', 'object')),
    mode text NOT NULL DEFAULT 'contract' CHECK (mode IN ('contract', 'contract_with_diagnostics')),
    module text,
    object_type text,
    object_id text,
    status text NOT NULL CHECK (status IN ('passed', 'warning', 'pending', 'failed')),
    blocking boolean NOT NULL DEFAULT false,
    summary jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(summary) = 'object'),
    result jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(result) = 'object'),
    created_by uuid,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS data_contract_check_runs_project_created_idx
    ON data_contract_check_runs (project_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS data_contract_check_runs_project_status_idx
    ON data_contract_check_runs (project_id, status, blocking, created_at DESC);
```

`0011_contract_check_runs_down.sql` 保持：

```sql
DROP TABLE IF EXISTS data_contract_check_runs;
```

- [ ] **步骤 4：升级仓储记录和保存参数**

在 `contract_check_repository.go` 中加入字段：

```go
type ContractCheckRunRecord struct {
	ID         int64
	ProjectID  string
	Scope      string
	Mode       string
	Module     *string
	ObjectType *string
	ObjectID   *string
	Status     string
	Blocking   bool
	Summary    json.RawMessage
	Result     json.RawMessage
	CreatedBy  *string
	CreatedAt  time.Time
}

type SaveContractCheckRunParams struct {
	ProjectID  string
	Scope      string
	Mode       string
	Module     *string
	ObjectType *string
	ObjectID   *string
	Status     string
	Blocking   bool
	Summary    any
	Result     any
	CreatedBy  *string
}
```

更新 `INSERT/SELECT/Scan` 字段顺序，SQL 使用：

```sql
INSERT INTO data_contract_check_runs (
    project_id, scope, mode, module, object_type, object_id,
    status, blocking, summary, result, created_by
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10::jsonb, $11)
RETURNING id, project_id, scope, mode, module, object_type, object_id,
          status, blocking, summary, result, created_by, created_at
```

- [ ] **步骤 5：升级服务运行记录类型**

将 `ContractCheckRun` 改为：

```go
type ContractCheckRun struct {
	ID         int64           `json:"id"`
	ProjectID  string          `json:"projectId"`
	Scope      string          `json:"scope"`
	Mode       string          `json:"mode"`
	Module     *string         `json:"module,omitempty"`
	ObjectType *string         `json:"objectType,omitempty"`
	ObjectID   *string         `json:"objectId,omitempty"`
	Status     string          `json:"status"`
	Blocking   bool            `json:"blocking"`
	Summary    map[string]int  `json:"summary"`
	Result     json.RawMessage `json:"result"`
	CreatedBy  *string         `json:"createdBy,omitempty"`
	CreatedAt  time.Time       `json:"createdAt"`
}
```

保存记录时传入：

```go
Mode:     result.Mode,
Module:   optionalTrimmedString(input.Module),
Blocking: result.Blocking,
```

- [ ] **步骤 6：运行测试验证通过**

```powershell
go test ./internal/service -run 'TestToContractCheckRunIncludesModeAndBlocking|TestBuildContractCheckRunResultBlockingSummary' -count=1
```

预期：PASS。

- [ ] **步骤 7：Commit**

```powershell
git add data_service/internal/db/migrations/0011_contract_check_runs.sql data_service/internal/db/migrations/0011_contract_check_runs_down.sql data_service/internal/repository/contract_check_repository.go data_service/internal/service/contract_check_service.go data_service/internal/service/contract_check_service_test.go
git commit -m "feat(data): 扩展契约检查运行记录"
```

---

## 任务 3：实现规则流水线和范围过滤

**文件：**

- 修改：`data_service/internal/service/contract_check_service.go`
- 修改：`data_service/internal/service/contract_check_service_test.go`
- 修改：`data_service/internal/http/handler/contract_check_handler.go`

- [ ] **步骤 1：编写范围过滤测试**

在 `contract_check_service_test.go` 增加：

```go
func TestFilterContractCheckIssuesByModuleAndObject(t *testing.T) {
	issues := []ContractCheckIssue{
		{Module: "http", ObjectType: "httpRequest", ObjectID: "r1", Status: "failed"},
		{Module: "http", ObjectType: "httpRequest", ObjectID: "r2", Status: "warning"},
		{Module: "datapoint", ObjectType: "datapoint", ObjectID: "dp1", Status: "failed"},
	}

	moduleOnly := filterContractCheckIssues(issues, ContractCheckInput{Scope: "module", Module: "http"})
	if len(moduleOnly) != 2 {
		t.Fatalf("moduleOnly len = %d, want 2", len(moduleOnly))
	}

	objectOnly := filterContractCheckIssues(issues, ContractCheckInput{Scope: "object", Module: "http", ObjectType: "httpRequest", ObjectID: "r2"})
	if len(objectOnly) != 1 || objectOnly[0].ObjectID != "r2" {
		t.Fatalf("objectOnly mismatch: %#v", objectOnly)
	}
}
```

- [ ] **步骤 2：编写流水线阶段测试**

增加：

```go
func TestContractCheckPipelineAddsPassedIssueForCleanProject(t *testing.T) {
	ctx := contractCheckContext{
		ProjectID: "project-1",
		Datapoints: []repository.DataPointRecord{
			{ID: "dp-1", Path: "manual.temperature", Name: "温度", DataType: "number", Status: "active"},
		},
	}
	issues := runContractCheckPipeline(ctx, ContractCheckInput{Scope: "project"})
	if len(issues) == 0 {
		t.Fatalf("expected at least one passed issue")
	}
	if issues[0].Status != "passed" {
		t.Fatalf("first issue status = %s, want passed", issues[0].Status)
	}
}
```

- [ ] **步骤 3：运行测试验证失败**

```powershell
go test ./internal/service -run 'TestFilterContractCheckIssuesByModuleAndObject|TestContractCheckPipelineAddsPassedIssueForCleanProject' -count=1
```

预期：FAIL，缺少 `filterContractCheckIssues`、`contractCheckContext` 或 `runContractCheckPipeline`。

- [ ] **步骤 4：实现上下文和流水线**

在 `contract_check_service.go` 增加：

```go
type contractCheckContext struct {
	ProjectID      string
	Connections    []repository.ConnectionRecord
	Datapoints     []repository.DataPointRecord
	Queries        []repository.QueryRecord
	ComputeUnits   []repository.ComputeUnitRecord
	AlarmRules     []repository.AlarmRuleRecord
	DatapointByID  map[string]repository.DataPointRecord
	DatapointByPath map[string]repository.DataPointRecord
	ConnectionByID map[string]repository.ConnectionRecord
	QueryByID      map[string]repository.QueryRecord
}

func runContractCheckPipeline(ctx contractCheckContext, input ContractCheckInput) []ContractCheckIssue {
	issues := make([]ContractCheckIssue, 0)
	issues = append(issues, checkDatapointStaticContracts(ctx)...)
	issues = append(issues, checkAccessSourceStaticContracts(ctx, input)...)
	issues = append(issues, checkQueryStaticContracts(ctx)...)
	issues = append(issues, checkComputeContractIssues(ctx)...)
	issues = append(issues, checkAlarmContractIssues(ctx)...)
	issues = append(issues, checkArtifactDryRunIssues(ctx)...)
	if len(issues) == 0 {
		issues = append(issues, ContractCheckIssue{
			Status:        "passed",
			Blocking:      false,
			Category:      "artifact",
			RuntimeImpact: "advisory",
			Module:        "artifact",
			ObjectType:    "project",
			ObjectID:      ctx.ProjectID,
			Title:         "数据中心契约检查通过",
			Detail:        "未发现阻断发布的数据域契约问题",
			Suggestion:    "可以继续发布前 Designer 契约检查或工程发布流程",
		})
	}
	return filterContractCheckIssues(issues, input)
}
```

- [ ] **步骤 5：实现范围过滤**

```go
func filterContractCheckIssues(issues []ContractCheckIssue, input ContractCheckInput) []ContractCheckIssue {
	scope := normalizeContractCheckScope(input.Scope)
	module := strings.TrimSpace(input.Module)
	objectType := strings.TrimSpace(input.ObjectType)
	objectID := strings.TrimSpace(input.ObjectID)
	filtered := make([]ContractCheckIssue, 0, len(issues))
	for _, issue := range issues {
		if scope == "module" && module != "" && issue.Module != module {
			continue
		}
		if scope == "object" {
			if module != "" && issue.Module != module {
				continue
			}
			if objectType != "" && issue.ObjectType != objectType {
				continue
			}
			if objectID != "" && issue.ObjectID != objectID {
				continue
			}
		}
		filtered = append(filtered, issue)
	}
	return filtered
}
```

- [ ] **步骤 6：升级 handler 请求解析**

在 `contract_check_handler.go` 的 Run 请求体中加入：

```go
var request struct {
	Scope      string `json:"scope"`
	Mode       string `json:"mode"`
	Module     string `json:"module"`
	ObjectType string `json:"objectType"`
	ObjectID   string `json:"objectId"`
	Trigger    string `json:"trigger"`
}
```

调用服务时传入 `Mode` 和 `Module`。
同时传入 `Trigger`，用于区分 datacenter UI 自检与 dev_core 发布门禁。

- [ ] **步骤 7：运行测试验证通过**

```powershell
go test ./internal/service -run 'ContractCheck.*Pipeline|FilterContractCheckIssues|BuildContractCheckRunResult' -count=1
```

预期：PASS。

- [ ] **步骤 8：Commit**

```powershell
git add data_service/internal/service/contract_check_service.go data_service/internal/service/contract_check_service_test.go data_service/internal/http/handler/contract_check_handler.go
git commit -m "feat(data): 契约检查支持规则流水线"
```

---

## 任务 4：加载接入源、查询、协议对象和 artifact 上下文

**文件：**

- 修改：`data_service/internal/app/server.go`
- 修改：`data_service/internal/service/contract_check_service.go`
- 修改：`data_service/internal/service/contract_check_service_test.go`

- [ ] **步骤 1：编写上下文索引测试**

在 `contract_check_service_test.go` 增加：

```go
func TestBuildContractCheckIndexes(t *testing.T) {
	ctx := contractCheckContext{
		Connections: []repository.ConnectionRecord{{ID: "conn-1", Type: "http", Name: "HTTP"}},
		Datapoints: []repository.DataPointRecord{{ID: "dp-1", Path: "http.api.get", Status: "active"}},
		Queries:    []repository.QueryRecord{{ID: "q-1", Name: "查询"}},
	}
	buildContractCheckIndexes(&ctx)
	if _, ok := ctx.ConnectionByID["conn-1"]; !ok {
		t.Fatalf("missing connection index")
	}
	if _, ok := ctx.DatapointByPath["http.api.get"]; !ok {
		t.Fatalf("missing datapoint path index")
	}
	if _, ok := ctx.QueryByID["q-1"]; !ok {
		t.Fatalf("missing query index")
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

```powershell
go test ./internal/service -run TestBuildContractCheckIndexes -count=1
```

预期：FAIL，缺少索引构建函数或上下文字段。

- [ ] **步骤 3：扩展服务依赖**

将 `ContractCheckService` 增加依赖：

```go
connections     *repository.ConnectionRepository
mqtt            *repository.MqttRepository
kafkaWorkbench  *repository.KafkaWorkbenchRepository
httpWorkbench   *repository.HTTPWorkbenchRepository
websocketWb     *repository.WebSocketWorkbenchRepository
realtimeStore   *repository.RealtimeStoreRepository
opcuaModeling   *repository.OpcuaModelingRepository
modbusModeling  *repository.ModbusModelingRepository
s7Modeling      *repository.S7ModelingRepository
snapshotService *ProjectSnapshotService
```

保留现有 `deps ...any` 注入方式，在 switch 中识别这些类型，减少构造函数大面积改动。

- [ ] **步骤 4：在 server 注入依赖**

修改 `data_service/internal/app/server.go` 的 `NewContractCheckService` 调用，传入：

```go
contractCheckService := service.NewContractCheckService(
	dataPointRepository,
	computeRepository,
	alarmRuleRepository,
	queryRepository,
	contractCheckRepository,
	connectionRepository,
	mqttRepository,
	kafkaWorkbenchRepository,
	httpWorkbenchRepository,
	websocketWorkbenchRepository,
	realtimeStoreRepository,
	opcuaModelingRepository,
	modbusModelingRepository,
	s7ModelingRepository,
	projectSnapshotService,
)
```

- [ ] **步骤 5：实现上下文加载和索引**

在 `contract_check_service.go` 中新增：

```go
func (s *ContractCheckService) loadContractCheckContext(ctx context.Context, projectID string) (contractCheckContext, error) {
	result := contractCheckContext{ProjectID: projectID}
	var err error
	result.Datapoints, err = s.loadProjectDatapoints(ctx, projectID)
	if err != nil {
		return result, err
	}
	if s.connections != nil {
		result.Connections, err = s.connections.ListByProject(ctx, projectID)
		if err != nil {
			return result, err
		}
	}
	queryIndex, err := s.loadProjectQueryIndex(ctx, projectID)
	if err != nil {
		return result, err
	}
	result.Queries = make([]repository.QueryRecord, 0, len(queryIndex))
	for _, query := range queryIndex {
		result.Queries = append(result.Queries, query)
	}
	result.ComputeUnits, err = s.loadProjectComputeUnits(ctx, projectID)
	if err != nil {
		return result, err
	}
	result.AlarmRules, err = s.loadProjectAlarmRules(ctx, projectID)
	if err != nil {
		return result, err
	}
	buildContractCheckIndexes(&result)
	return result, nil
}

func buildContractCheckIndexes(ctx *contractCheckContext) {
	ctx.DatapointByID = map[string]repository.DataPointRecord{}
	ctx.DatapointByPath = map[string]repository.DataPointRecord{}
	for _, point := range ctx.Datapoints {
		ctx.DatapointByID[point.ID] = point
		ctx.DatapointByPath[point.Path] = point
	}
	ctx.ConnectionByID = map[string]repository.ConnectionRecord{}
	for _, connection := range ctx.Connections {
		ctx.ConnectionByID[connection.ID] = connection
	}
	ctx.QueryByID = map[string]repository.QueryRecord{}
	for _, query := range ctx.Queries {
		ctx.QueryByID[query.ID] = query
	}
}
```

- [ ] **步骤 6：Run 使用新上下文**

将 `Run` 中散落的加载逻辑替换为：

```go
checkCtx, err := s.loadContractCheckContext(ctx, projectID)
if err != nil {
	return nil, err
}
issues := runContractCheckPipeline(checkCtx, input)
result := buildContractCheckRunResult(projectID, input, issues)
```

- [ ] **步骤 7：运行测试验证通过**

```powershell
go test ./internal/service -run 'ContractCheck|BuildContractCheckIndexes' -count=1
```

预期：PASS。

- [ ] **步骤 8：Commit**

```powershell
git add data_service/internal/app/server.go data_service/internal/service/contract_check_service.go data_service/internal/service/contract_check_service_test.go
git commit -m "feat(data): 契约检查加载完整上下文"
```

---

## 任务 5：实现数据点、接入源和查询规则

**文件：**

- 修改：`data_service/internal/service/contract_check_service.go`
- 修改：`data_service/internal/service/contract_check_service_test.go`

- [ ] **步骤 1：编写规则测试**

在 `contract_check_service_test.go` 增加：

```go
func TestContractRulesForDatapointAccessSourceAndQuery(t *testing.T) {
	ctx := contractCheckContext{
		Connections: []repository.ConnectionRecord{
			{ID: "http-1", Type: "http", Name: "HTTP", Config: map[string]any{}},
			{ID: "rt-1", Type: "builtin.realtime", Name: "实时库", Config: map[string]any{}},
		},
		Datapoints: []repository.DataPointRecord{
			{ID: "dp-1", Path: "", Name: "空路径", SourceType: "manual", Status: "active"},
			{ID: "dp-2", Path: "same.path", Name: "A", SourceType: "manual", Status: "active"},
			{ID: "dp-3", Path: "same.path", Name: "B", SourceType: "manual", Status: "active"},
			{ID: "dp-4", Path: "http.api", Name: "HTTP", SourceType: "http.request", Status: "invalid"},
		},
		Queries: []repository.QueryRecord{
			{ID: "q-1", Name: "空 SQL", ConnectionID: "http-1", QueryType: "sql", Config: map[string]any{"sql": ""}, IsEnabled: true},
		},
	}
	buildContractCheckIndexes(&ctx)

	issues := append(checkDatapointStaticContracts(ctx), checkAccessSourceStaticContracts(ctx, ContractCheckInput{} )...)
	issues = append(issues, checkQueryStaticContracts(ctx)...)
	if !hasContractIssue(issues, "datapoint", "数据点 path 为空") {
		t.Fatalf("expected empty path issue: %#v", issues)
	}
	if !hasContractIssue(issues, "datapoint", "数据点 path 重复") {
		t.Fatalf("expected duplicate path issue: %#v", issues)
	}
	if !hasContractIssue(issues, "access_source", "IF 实时库缺少 runtimeKey") {
		t.Fatalf("expected runtimeKey issue: %#v", issues)
	}
	if !hasContractIssue(issues, "query", "查询 SQL 为空") {
		t.Fatalf("expected empty sql issue: %#v", issues)
	}
}
```

同时加入辅助断言：

```go
func hasContractIssue(issues []ContractCheckIssue, module, title string) bool {
	for _, issue := range issues {
		if issue.Module == module && issue.Title == title {
			return true
		}
	}
	return false
}
```

- [ ] **步骤 2：运行测试验证失败**

```powershell
go test ./internal/service -run TestContractRulesForDatapointAccessSourceAndQuery -count=1
```

预期：FAIL，规则函数尚未输出这些问题。

- [ ] **步骤 3：实现数据点规则**

在 `contract_check_service.go` 中实现：

```go
func checkDatapointStaticContracts(ctx contractCheckContext) []ContractCheckIssue {
	issues := make([]ContractCheckIssue, 0)
	pathOwners := map[string][]repository.DataPointRecord{}
	for _, point := range ctx.Datapoints {
		path := strings.TrimSpace(point.Path)
		if path == "" {
			issues = append(issues, blockingIssue("datapoint", "datapoint", point.ID, point.Name, "数据点 path 为空", "数据点必须有稳定 path，运行态按 path 解析数据。", "填写唯一 path 或删除无效数据点"))
			continue
		}
		pathOwners[path] = append(pathOwners[path], point)
		if strings.TrimSpace(point.SourceType) == "" {
			issues = append(issues, blockingIssue("datapoint", "datapoint", point.ID, point.Name, "数据点 sourceType 为空", path, "补齐 sourceType"))
		}
		if point.Status == "invalid" {
			issues = append(issues, blockingIssue("datapoint", "datapoint", point.ID, point.Name, "数据点已失效", path, "恢复来源对象或移除数据点引用"))
		}
	}
	for path, owners := range pathOwners {
		if len(owners) <= 1 {
			continue
		}
		issues = append(issues, blockingIssue("datapoint", "datapoint", owners[0].ID, owners[0].Name, "数据点 path 重复", path, "调整重复数据点 path，确保项目内唯一"))
	}
	return issues
}
```

- [ ] **步骤 4：实现接入源规则**

```go
func checkAccessSourceStaticContracts(ctx contractCheckContext, input ContractCheckInput) []ContractCheckIssue {
	issues := make([]ContractCheckIssue, 0)
	for _, connection := range ctx.Connections {
		switch connection.Type {
		case "builtin.realtime":
			if strings.TrimSpace(toString(connection.Config["runtimeKey"])) == "" {
				issues = append(issues, blockingIssue("access_source", "connection", connection.ID, connection.Name, "IF 实时库缺少 runtimeKey", "运行态需要 runtimeKey 隔离实时库 key。", "在接入源配置中补齐 runtimeKey"))
			}
		case "kafka":
			if len(stringSliceFromAny(connection.Config["brokers"])) == 0 && strings.TrimSpace(toString(connection.Config["brokers"])) == "" {
				issues = append(issues, blockingIssue("access_source", "connection", connection.ID, connection.Name, "Kafka 接入源缺少 brokers", "运行态无法构造 Kafka 客户端配置。", "补齐 brokers 配置"))
			}
		case "http":
			// HTTP 允许每个请求使用完整 URL；baseUrl 缺失不单独阻断。
		case "redis":
			if strings.TrimSpace(toString(connection.Config["address"])) == "" && strings.TrimSpace(toString(connection.Config["host"])) == "" {
				issues = append(issues, blockingIssue("access_source", "connection", connection.ID, connection.Name, "Redis 接入源缺少地址", "运行态无法定位 Redis 服务。", "补齐 Redis address 或 host/port"))
			}
		}
	}
	return issues
}
```

- [ ] **步骤 5：实现查询规则**

```go
func checkQueryStaticContracts(ctx contractCheckContext) []ContractCheckIssue {
	issues := make([]ContractCheckIssue, 0)
	for _, query := range ctx.Queries {
		sqlText := ""
		if query.QueryType == "sql" {
			extracted, err := extractQuerySQL(query.Config)
			if err != nil {
				issue := blockingIssue("query", "query", query.ID, query.Name, "查询 SQL 为空", "sql 配置缺失或为空", "填写只读 SQL 或停用该查询")
				if !query.IsEnabled {
					issue.Blocking = false
					issue.Status = "warning"
					issue.RuntimeImpact = "advisory"
				}
				issues = append(issues, issue)
			} else {
				sqlText = extracted
			}
		}
		if strings.TrimSpace(query.ConnectionID) == "" {
			issue := blockingIssue("query", "query", query.ID, query.Name, "查询所属接入源为空", "启用查询必须绑定可生成运行契约的接入源。", "选择关系库接入源或停用该查询")
			if !query.IsEnabled {
				issue.Blocking = false
				issue.Status = "warning"
			}
			issues = append(issues, issue)
		}
		if _, ok := ctx.ConnectionByID[query.ConnectionID]; strings.TrimSpace(query.ConnectionID) != "" && !ok {
			issues = append(issues, blockingIssue("query", "query", query.ID, query.Name, "查询所属接入源不存在", query.ConnectionID, "重新选择接入源或删除查询"))
		}
		if sqlText != "" && ensureReadOnlySQL(sqlText) != nil {
			issues = append(issues, blockingIssue("query", "query", query.ID, query.Name, "查询 SQL 不满足只读约束", sqlText, "改为 SELECT/WITH 查询"))
		}
	}
	return issues
}
```

- [ ] **步骤 6：实现 issue 构造辅助函数**

```go
func blockingIssue(module, objectType, objectID, objectName, title, detail, suggestion string) ContractCheckIssue {
	return ContractCheckIssue{
		Status:        "failed",
		Blocking:      true,
		Category:      "static_config",
		RuntimeImpact: "contract_blocker",
		Module:        module,
		ObjectType:    objectType,
		ObjectID:      objectID,
		ObjectName:    objectName,
		Title:         title,
		Detail:        detail,
		Suggestion:    suggestion,
	}
}
```

- [ ] **步骤 7：运行测试验证通过**

```powershell
go test ./internal/service -run TestContractRulesForDatapointAccessSourceAndQuery -count=1
```

预期：PASS。

- [ ] **步骤 8：Commit**

```powershell
git add data_service/internal/service/contract_check_service.go data_service/internal/service/contract_check_service_test.go
git commit -m "feat(data): 增加数据点接入源查询契约规则"
```

---

## 任务 6：实现计算、报警和 DataDomainArtifact dry-run 规则

**文件：**

- 修改：`data_service/internal/service/contract_check_service.go`
- 修改：`data_service/internal/service/contract_check_service_test.go`

- [ ] **步骤 1：编写计算和报警阻断测试**

在 `contract_check_service_test.go` 增加：

```go
func TestComputeAndAlarmContractIssuesAreBlockingForEnabledObjects(t *testing.T) {
	ctx := contractCheckContext{
		DatapointByPath: map[string]repository.DataPointRecord{
			"good.point": {ID: "dp-1", Path: "good.point", Status: "active", DataType: "number"},
		},
		QueryByID: map[string]repository.QueryRecord{},
		ComputeUnits: []repository.ComputeUnitRecord{
			{
				ID: "calc-1", Name: "计算", IsEnabled: true,
				InputBindings: map[string]any{
					"datapointVariables": []any{map[string]any{"path": "missing.point"}},
				},
			},
		},
		AlarmRules: []repository.AlarmRuleRecord{
			{ID: "alarm-1", Name: "报警", TargetPath: "missing.point", IsEnabled: true},
		},
	}

	issues := append(checkComputeContractIssues(ctx), checkAlarmContractIssues(ctx)...)
	if !hasContractIssue(issues, "compute", "计算输入数据点不存在") {
		t.Fatalf("expected compute input issue: %#v", issues)
	}
	if !hasContractIssue(issues, "alarm", "报警目标数据点不存在") {
		t.Fatalf("expected alarm target issue: %#v", issues)
	}
	for _, issue := range issues {
		if !issue.Blocking {
			t.Fatalf("enabled object issue should block: %#v", issue)
		}
	}
}
```

- [ ] **步骤 2：编写 DataDomainArtifact 基础 dry-run 测试**

```go
func TestArtifactDryRunReportsProjectWithoutDatapoints(t *testing.T) {
	ctx := contractCheckContext{ProjectID: "project-1"}
	issues := checkArtifactDryRunIssues(ctx)
	if !hasContractIssue(issues, "artifact", "artifact 缺少数据点契约") {
		t.Fatalf("expected artifact issue: %#v", issues)
	}
}
```

- [ ] **步骤 3：编写 DataDomainArtifact 完整性 dry-run 测试**

```go
func TestArtifactDryRunValidatesDataDomainArtifactSections(t *testing.T) {
	ctx := contractCheckContext{
		ProjectID: "project-1",
		Connections: []repository.ConnectionRecord{
			{ID: "builtin-rt", Name: "IF实时库", Type: "builtin.realtime", Status: "active", Config: map[string]any{"runtimeKey": "rt_1"}},
			{ID: "s7-1", Name: "S7", Type: "s7", Status: "active", Config: map[string]any{"host": "192.168.1.10"}},
		},
		Datapoints: []repository.DataPointRecord{
			{ID: "dp-1", Path: "line.temp", Name: "温度", SourceType: "s7.variable", DataType: "number", Status: "active"},
		},
		ComputeUnits: []repository.ComputeUnitRecord{
			{ID: "calc-1", Name: "计算", IsEnabled: true},
		},
		AlarmRules: []repository.AlarmRuleRecord{
			{ID: "alarm-1", Name: "报警", TargetPath: "line.temp", Severity: "high", IsEnabled: true},
		},
	}
	buildContractCheckIndexes(&ctx)

	issues := checkArtifactDryRunIssues(ctx)
	if len(issues) != 0 {
		t.Fatalf("expected clean artifact dry-run, got %#v", issues)
	}
}

func TestArtifactDryRunRejectsInternalValueLeak(t *testing.T) {
	ctx := contractCheckContext{
		ProjectID: "project-1",
		Connections: []repository.ConnectionRecord{
			{ID: "rel-1", Name: "关系库", Type: "relational", Status: "active", Config: map[string]any{
				"database": "if_dev_data",
				"password": "plain-password",
			}},
		},
		Datapoints: []repository.DataPointRecord{
			{ID: "dp-1", Path: "db.value", Name: "值", SourceType: "query", DataType: "number", Status: "active"},
		},
	}

	issues := checkArtifactDryRunIssues(ctx)
	if !hasContractIssue(issues, "artifact", "DataDomainArtifact 包含开发态敏感信息") {
		t.Fatalf("expected sensitive leak issue: %#v", issues)
	}
}
```

- [ ] **步骤 4：运行测试验证失败**

```powershell
go test ./internal/service -run 'TestComputeAndAlarmContractIssuesAreBlockingForEnabledObjects|TestArtifactDryRun' -count=1
```

预期：FAIL，规则函数行为未升级。

- [ ] **步骤 5：升级计算规则**

将旧 `checkComputeContracts` 改为新版 `checkComputeContractIssues`：

```go
func checkComputeContractIssues(ctx contractCheckContext) []ContractCheckIssue {
	issues := make([]ContractCheckIssue, 0)
	for _, unit := range ctx.ComputeUnits {
		enabled := unit.IsEnabled
		for _, path := range collectBindingPaths(unit.InputBindings) {
			if _, ok := ctx.DatapointByPath[path]; !ok {
				issue := blockingIssue("compute", "computeUnit", unit.ID, unit.Name, "计算输入数据点不存在", path, "修正输入绑定或补齐数据点")
				if !enabled {
					issue.Status = "warning"
					issue.Blocking = false
					issue.RuntimeImpact = "advisory"
				}
				issues = append(issues, issue)
			}
		}
		for key, binding := range extractComputeSQLBindings(unit.InputBindings) {
			if _, ok := ctx.QueryByID[binding.QueryID]; !ok {
				issue := blockingIssue("compute", "computeUnit", unit.ID, unit.Name, "计算 SQL 绑定查询不存在", key+":"+binding.QueryID, "修正 SQL 绑定或补齐查询")
				if !enabled {
					issue.Status = "warning"
					issue.Blocking = false
				}
				issues = append(issues, issue)
			}
		}
	}
	return issues
}
```

- [ ] **步骤 6：升级报警规则**

```go
func checkAlarmContractIssues(ctx contractCheckContext) []ContractCheckIssue {
	issues := make([]ContractCheckIssue, 0)
	for _, rule := range ctx.AlarmRules {
		point, ok := ctx.DatapointByPath[rule.TargetPath]
		if !ok {
			issue := blockingIssue("alarm", "alarmRule", rule.ID, rule.Name, "报警目标数据点不存在", rule.TargetPath, "修正报警目标或停用该规则")
			if !rule.IsEnabled {
				issue.Status = "warning"
				issue.Blocking = false
			}
			issues = append(issues, issue)
			continue
		}
		if rule.IsEnabled && point.Status != "active" {
			issues = append(issues, blockingIssue("alarm", "alarmRule", rule.ID, rule.Name, "报警目标数据点不可用", rule.TargetPath, "恢复目标数据点"))
		}
		if rule.IsEnabled && strings.TrimSpace(rule.Severity) == "" {
			issue := blockingIssue("alarm", "alarmRule", rule.ID, rule.Name, "报警等级缺失", rule.TargetPath, "补齐报警等级")
			issue.Status = "pending"
			issues = append(issues, issue)
		}
	}
	return issues
}
```

- [ ] **步骤 7：实现 DataDomainArtifact dry-run 规则**

```go
func checkArtifactDryRunIssues(ctx contractCheckContext) []ContractCheckIssue {
	issues := make([]ContractCheckIssue, 0)
	if len(ctx.Datapoints) == 0 {
		issues = append(issues, ContractCheckIssue{
			Status:        "failed",
			Blocking:      true,
			Category:      "artifact",
			RuntimeImpact: "contract_blocker",
			Module:        "artifact",
			ObjectType:    "project",
			ObjectID:      ctx.ProjectID,
			Title:         "artifact 缺少数据点契约",
			Detail:        "运行态数据契约至少需要包含数据点定义。",
			Suggestion:    "创建数据点或保存接入源变量以生成数据点",
		})
	}

	artifact := repository.BuildProjectArtifactV1(ctx.ProjectID, &repository.ProjectSnapshot{
		Connections:  ctx.Connections,
		Queries:      ctx.Queries,
		DataPoints:   ctx.Datapoints,
		ComputeUnits: ctx.ComputeUnits,
		AlarmRules:   ctx.AlarmRules,
	}, time.Now().UTC())
	artifactBytes, err := json.Marshal(artifact)
	if err != nil {
		issues = append(issues, blockingIssue("artifact", "project", ctx.ProjectID, "", "DataDomainArtifact 无法序列化", err.Error(), "检查数据域配置中的 JSON 字段"))
		return issues
	}
	if artifact == nil || artifact.Version == "" || artifact.ProjectID == "" {
		issues = append(issues, blockingIssue("artifact", "project", ctx.ProjectID, "", "DataDomainArtifact 基础字段缺失", "version/projectId 不能为空", "检查 artifact 生成逻辑"))
	}
	if artifact.BuiltinStores.RealtimeSpaces == nil {
		issues = append(issues, blockingIssue("artifact", "project", ctx.ProjectID, "", "DataDomainArtifact 缺少 builtinStores 区块", "IF 内置库契约必须进入发布态数据产物", "检查内置库契约生成逻辑"))
	}

	lower := strings.ToLower(string(artifactBytes))
	for _, forbidden := range []string{"if_dev_data", "plain-password", "\"password\"", "18379", "18883"} {
		if strings.Contains(lower, strings.ToLower(forbidden)) {
			issues = append(issues, blockingIssue("artifact", "project", ctx.ProjectID, "", "DataDomainArtifact 包含开发态敏感信息", forbidden, "过滤开发态库名、端口和明文密钥，只发布运行态契约"))
			break
		}
	}
	return issues
}
```

说明：本步骤只检查 `data_service` 生成的 `DataDomainArtifact`。最终 `.ifp` 的 `datacenter.json`、`runtime-capabilities.json` 和部署 profile 由后续 `dev_core` 发布部署计划覆盖，不在本计划中修改代码。

- [ ] **步骤 8：运行测试验证通过**

```powershell
go test ./internal/service -run 'ComputeAndAlarmContract|ArtifactDryRun' -count=1
```

预期：PASS。

- [ ] **步骤 9：Commit**

```powershell
git add data_service/internal/service/contract_check_service.go data_service/internal/service/contract_check_service_test.go
git commit -m "feat(data): 契约检查覆盖计算报警和artifact"
```

---

## 任务 7：接入协议工作台静态规则

**文件：**

- 修改：`data_service/internal/service/contract_check_service.go`
- 修改：`data_service/internal/service/contract_check_service_test.go`

- [ ] **步骤 1：编写协议 sourceConfig 规则测试**

在 `contract_check_service_test.go` 增加：

```go
func TestGeneratedDatapointSourceConfigRules(t *testing.T) {
	ctx := contractCheckContext{
		Datapoints: []repository.DataPointRecord{
			{ID: "kafka-dp", Path: "kafka.topic.value", Name: "Kafka", SourceType: "kafka.field", SourceConfig: map[string]any{}, Status: "active"},
			{ID: "http-dp", Path: "http.api.get", Name: "HTTP", SourceType: "http.request", SourceConfig: map[string]any{"requestId": "req-1"}, Status: "active"},
			{ID: "rt-dp", Path: "realtime.device", Name: "Key", SourceType: "realtime.key", SourceConfig: map[string]any{"key": "device:1"}, Status: "active"},
		},
	}
	issues := checkGeneratedDatapointReferenceContracts(ctx)
	if !hasContractIssue(issues, "kafka", "Kafka 变量数据点缺少 fieldId") {
		t.Fatalf("expected kafka fieldId issue: %#v", issues)
	}
	if !hasContractIssue(issues, "realtime_store", "实时库数据点缺少 keyId") {
		t.Fatalf("expected realtime keyId issue: %#v", issues)
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

```powershell
go test ./internal/service -run TestGeneratedDatapointSourceConfigRules -count=1
```

预期：FAIL，缺少生成型数据点引用规则。

- [ ] **步骤 3：实现生成型数据点 sourceConfig 规则**

在 `contract_check_service.go` 增加：

```go
func checkGeneratedDatapointReferenceContracts(ctx contractCheckContext) []ContractCheckIssue {
	issues := make([]ContractCheckIssue, 0)
	for _, point := range ctx.Datapoints {
		switch point.SourceType {
		case "kafka.field":
			if strings.TrimSpace(firstString(point.SourceConfig, "fieldId")) == "" {
				issues = append(issues, blockingIssue("kafka", "datapoint", point.ID, point.Name, "Kafka 变量数据点缺少 fieldId", point.Path, "重新保存 Kafka 变量以同步数据点 sourceConfig"))
			}
		case "http.request":
			if strings.TrimSpace(firstString(point.SourceConfig, "requestId")) == "" {
				issues = append(issues, blockingIssue("http", "datapoint", point.ID, point.Name, "HTTP 请求数据点缺少 requestId", point.Path, "重新保存 HTTP 请求"))
			}
		case "websocket.session":
			if strings.TrimSpace(firstString(point.SourceConfig, "sessionId")) == "" {
				issues = append(issues, blockingIssue("websocket", "datapoint", point.ID, point.Name, "WebSocket 会话数据点缺少 sessionId", point.Path, "重新保存 WebSocket 会话"))
			}
		case "realtime.key":
			if strings.TrimSpace(firstString(point.SourceConfig, "keyId")) == "" {
				issues = append(issues, blockingIssue("realtime_store", "datapoint", point.ID, point.Name, "实时库数据点缺少 keyId", point.Path, "重新创建实时库 key 数据点"))
			}
		}
	}
	return issues
}
```

- [ ] **步骤 4：加入流水线**

在 `runContractCheckPipeline` 中追加：

```go
issues = append(issues, checkGeneratedDatapointReferenceContracts(ctx)...)
```

- [ ] **步骤 5：运行测试验证通过**

```powershell
go test ./internal/service -run TestGeneratedDatapointSourceConfigRules -count=1
```

预期：PASS。

- [ ] **步骤 6：Commit**

```powershell
git add data_service/internal/service/contract_check_service.go data_service/internal/service/contract_check_service_test.go
git commit -m "feat(data): 契约检查覆盖协议数据点引用"
```

---

## 任务 8：升级前端 API、schema 和 store

**文件：**

- 修改：`datacenter/src/api/schemas/contract-check.schema.ts`
- 修改：`datacenter/src/api/contract-check.api.ts`
- 修改：`datacenter/src/stores/contract-check.store.ts`

- [ ] **步骤 1：编写 schema 类型解析样例**

在 `datacenter/src/api/schemas/contract-check.schema.ts` 底部加入导出的样例解析函数，供类型检查约束使用：

```ts
export function parseContractCheckRunForTest(input: unknown): ContractCheckRun {
  return ContractCheckRunSchema.parse(input)
}
```

临时在同文件中写出新 schema 后会通过 typecheck 验证。不要提交单独的测试工具文件。

- [ ] **步骤 2：替换 schema**

将文件内容改为：

```ts
import { z } from 'zod'

export const ContractCheckStatusSchema = z.enum(['passed', 'warning', 'pending', 'failed'])
export const ContractCheckScopeSchema = z.enum(['project', 'module', 'object'])
export const ContractCheckModeSchema = z.enum(['contract', 'contract_with_diagnostics'])
export const ContractCheckCategorySchema = z.enum([
  'static_config',
  'reference',
  'artifact',
  'diagnostic',
])
export const ContractCheckRuntimeImpactSchema = z.enum([
  'startup_blocker',
  'contract_blocker',
  'feature_unavailable',
  'data_quality_risk',
  'advisory',
])
export const ContractCheckModuleSchema = z.enum([
  'access_source',
  'datapoint',
  'query',
  'mqtt',
  'kafka',
  'http',
  'websocket',
  'realtime_store',
  'opcua',
  'modbus',
  's7',
  'compute',
  'alarm',
  'artifact',
])

export const ContractCheckJumpTargetSchema = z.object({
  routeName: z.string(),
  params: z.record(z.string(), z.string()).default({}),
  query: z.record(z.string(), z.string()).optional(),
})

export const ContractCheckIssueSchema = z.object({
  id: z.string(),
  status: ContractCheckStatusSchema,
  blocking: z.boolean(),
  category: ContractCheckCategorySchema,
  runtimeImpact: ContractCheckRuntimeImpactSchema,
  module: ContractCheckModuleSchema,
  objectType: z.string(),
  objectId: z.string().optional().default(''),
  objectName: z.string().optional().default(''),
  title: z.string(),
  detail: z.string().optional().default(''),
  suggestion: z.string().optional().default(''),
  jumpTarget: ContractCheckJumpTargetSchema.optional(),
})

export const ContractCheckProgressSchema = z.object({
  currentStage: z.string().default(''),
  currentModule: z.string().optional().default(''),
  currentObjectName: z.string().optional().default(''),
  checkedCount: z.number().default(0),
  totalCount: z.number().default(0),
})

export const ContractCheckSummarySchema = z.object({
  failed: z.number().default(0),
  pending: z.number().default(0),
  warning: z.number().default(0),
  passed: z.number().default(0),
  blocking: z.number().default(0),
})

export const ContractCheckRunSchema = z.object({
  id: z.string().optional().default(''),
  projectId: z.string(),
  scope: ContractCheckScopeSchema,
  mode: ContractCheckModeSchema,
  status: ContractCheckStatusSchema,
  blocking: z.boolean(),
  startedAt: z.string(),
  finishedAt: z.string().optional(),
  progress: ContractCheckProgressSchema,
  summary: ContractCheckSummarySchema,
  issues: z.array(ContractCheckIssueSchema).default([]),
})

export const ContractCheckRunListSchema = z.object({
  list: z.array(ContractCheckRunSchema).default([]),
  pagination: z
    .object({
      page: z.number().default(1),
      pageSize: z.number().default(20),
      total: z.number().default(0),
      totalPages: z.number().default(0),
    })
    .default({}),
})

export type ContractCheckStatus = z.infer<typeof ContractCheckStatusSchema>
export type ContractCheckModule = z.infer<typeof ContractCheckModuleSchema>
export type ContractCheckIssue = z.infer<typeof ContractCheckIssueSchema>
export type ContractCheckRun = z.infer<typeof ContractCheckRunSchema>
export type ContractCheckRunList = z.infer<typeof ContractCheckRunListSchema>

export interface RunContractCheckPayload {
  scope: 'project' | 'module' | 'object'
  mode?: 'contract' | 'contract_with_diagnostics'
  module?: ContractCheckModule
  objectType?: string
  objectId?: string
  trigger?: 'datacenter.ui' | 'dev_core.publish'
}
```

- [ ] **步骤 3：升级 API**

修改 `contract-check.api.ts`：

```ts
import request from '@/utils/request'
import {
  ContractCheckRunListSchema,
  ContractCheckRunSchema,
  type ContractCheckRun,
  type ContractCheckRunList,
  type RunContractCheckPayload,
} from './schemas/contract-check.schema'

export async function runContractCheck(
  projectId: string,
  payload: RunContractCheckPayload = { scope: 'project' },
): Promise<ContractCheckRun> {
  const res = await request({
    url: `/data/projects/${projectId}/contract-checks/run`,
    method: 'post',
    data: payload,
  })
  return ContractCheckRunSchema.parse(res)
}

export async function getLatestContractCheck(projectId: string): Promise<ContractCheckRun> {
  const res = await request({
    url: `/data/projects/${projectId}/contract-checks/latest`,
    method: 'get',
  })
  return ContractCheckRunSchema.parse(res)
}

export async function getContractCheckRuns(
  projectId: string,
  params: Record<string, unknown> = {},
): Promise<ContractCheckRunList> {
  const res = await request({
    url: `/data/projects/${projectId}/contract-checks/runs`,
    method: 'get',
    params,
  })
  return ContractCheckRunListSchema.parse(res)
}
```

- [ ] **步骤 4：升级 store**

修改 `contract-check.store.ts`：

```ts
import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { getLatestContractCheck, runContractCheck } from '@/api/contract-check.api'
import type {
  ContractCheckIssue,
  ContractCheckModule,
  ContractCheckRun,
  RunContractCheckPayload,
} from '@/api/schemas/contract-check.schema'

export const useContractCheckStore = defineStore('contractCheck', () => {
  const latestResult = ref<ContractCheckRun | null>(null)
  const running = ref(false)
  const selectedIssueId = ref('')
  const levelFilter = ref<string[]>(['failed', 'pending', 'warning'])
  const moduleFilter = ref<ContractCheckModule | ''>('')
  const keyword = ref('')
  const showPassed = ref(false)

  const highestStatus = computed(() => {
    if (running.value) return 'running'
    const summary = latestResult.value?.summary
    if (!summary) return 'idle'
    if (summary.failed > 0) return 'failed'
    if (summary.pending > 0) return 'pending'
    if (summary.warning > 0) return 'warning'
    return 'passed'
  })

  const filteredIssues = computed<ContractCheckIssue[]>(() => {
    const issues = latestResult.value?.issues ?? []
    const text = keyword.value.trim().toLowerCase()
    return issues.filter((issue) => {
      if (!showPassed.value && issue.status === 'passed') return false
      if (levelFilter.value.length && !levelFilter.value.includes(issue.status)) return false
      if (moduleFilter.value && issue.module !== moduleFilter.value) return false
      if (!text) return true
      return [issue.title, issue.detail, issue.objectName, issue.suggestion]
        .join(' ')
        .toLowerCase()
        .includes(text)
    })
  })

  const selectedIssue = computed(
    () =>
      filteredIssues.value.find((issue) => issue.id === selectedIssueId.value) ??
      filteredIssues.value[0] ??
      null,
  )

  async function fetchLatest(projectId: string) {
    try {
      latestResult.value = await getLatestContractCheck(projectId)
    } catch {
      latestResult.value = null
    }
  }

  async function run(projectId: string, payload: RunContractCheckPayload = { scope: 'project' }) {
    running.value = true
    try {
      latestResult.value = await runContractCheck(projectId, payload)
      selectedIssueId.value = filteredIssues.value[0]?.id ?? ''
    } finally {
      running.value = false
    }
  }

  function clear() {
    latestResult.value = null
    selectedIssueId.value = ''
  }

  return {
    latestResult,
    running,
    selectedIssueId,
    levelFilter,
    moduleFilter,
    keyword,
    showPassed,
    highestStatus,
    filteredIssues,
    selectedIssue,
    fetchLatest,
    run,
    clear,
  }
})
```

- [ ] **步骤 5：运行类型检查**

```powershell
pnpm --filter datacenter typecheck
```

预期：PASS。

- [ ] **步骤 6：Commit**

```powershell
git add datacenter/src/api/schemas/contract-check.schema.ts datacenter/src/api/contract-check.api.ts datacenter/src/stores/contract-check.store.ts
git commit -m "feat(datacenter): 升级契约检查前端数据模型"
```

---

## 任务 9：实现悬浮检查中心 UI

**文件：**

- 重写：`datacenter/src/components/contract/DataContractCheckDialog.vue`
- 删除：`datacenter/src/components/contract/dataContractCheck.ts`
- 修改：`datacenter/src/views/DataCenterNew.vue`

- [ ] **步骤 1：改造入口状态色**

在 `DataCenterNew.vue` 中引入 store：

```ts
import { useContractCheckStore } from '@/stores/contract-check.store'

const contractCheckStore = useContractCheckStore()
```

将入口 button 增加状态 class：

```vue
<button
  type="button"
  class="datacenter-side-action"
  :class="`is-contract-${contractCheckStore.highestStatus}`"
  title="数据契约检查"
  aria-label="数据契约检查"
  @click="showDataContractCheckDialog = true"
>
  <IconTablerShieldCheck class="h-5 w-5" />
  <span>数据契约检查</span>
</button>
```

在样式中加入：

```css
.datacenter-side-action.is-contract-idle svg {
  color: var(--dc-text-muted);
}

.datacenter-side-action.is-contract-running svg {
  color: var(--dc-primary);
}

.datacenter-side-action.is-contract-passed svg {
  color: #16a34a;
}

.datacenter-side-action.is-contract-warning svg {
  color: #d97706;
}

.datacenter-side-action.is-contract-pending svg {
  color: #ea580c;
}

.datacenter-side-action.is-contract-failed svg {
  color: #dc2626;
}
```

- [ ] **步骤 2：重写悬浮面板模板**

将 `DataContractCheckDialog.vue` 的模板替换为非强遮罩浮层：

```vue
<template>
  <Teleport to="body">
    <Transition name="contract-check-panel">
      <section v-if="visible" class="contract-check-panel" role="dialog" aria-label="契约检查">
        <header class="contract-check-panel__header">
          <div>
            <h2>契约检查</h2>
            <p>{{ statusText }}</p>
          </div>
          <el-button text :icon="IconTablerX" @click="visible = false" />
        </header>

        <div class="contract-check-panel__actions">
          <el-button type="primary" :loading="store.running" @click="runProject"
            >全量检查</el-button
          >
          <el-button :disabled="store.running" @click="runModule">当前模块</el-button>
          <el-button :disabled="store.running" @click="runObject">当前对象</el-button>
          <el-switch v-model="withDiagnostics" active-text="包含连接诊断" />
        </div>

        <section class="contract-check-panel__progress">
          <span>{{ progressLabel }}</span>
          <el-progress :percentage="progressPercent" :stroke-width="8" />
        </section>

        <section class="contract-check-panel__summary">
          <button type="button" class="is-failed">failed {{ summary.failed }}</button>
          <button type="button" class="is-pending">pending {{ summary.pending }}</button>
          <button type="button" class="is-warning">warning {{ summary.warning }}</button>
          <button type="button" class="is-passed">passed {{ summary.passed }}</button>
          <button type="button">阻断 {{ summary.blocking }}</button>
        </section>

        <section class="contract-check-panel__filters">
          <el-select v-model="store.moduleFilter" clearable placeholder="模块" size="small">
            <el-option
              v-for="item in moduleOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
          <el-input v-model="store.keyword" clearable size="small" placeholder="搜索问题" />
          <el-checkbox v-model="store.showPassed">显示通过项</el-checkbox>
        </section>

        <main class="contract-check-panel__body">
          <el-table
            :data="store.filteredIssues"
            height="100%"
            highlight-current-row
            @current-change="selectIssue"
          >
            <el-table-column prop="status" label="级别" width="92" />
            <el-table-column prop="module" label="模块" width="120" />
            <el-table-column prop="objectName" label="对象" min-width="140" />
            <el-table-column prop="title" label="问题" min-width="220" />
            <el-table-column prop="runtimeImpact" label="影响" width="150" />
          </el-table>

          <aside class="contract-check-panel__detail">
            <template v-if="store.selectedIssue">
              <h3>{{ store.selectedIssue.title }}</h3>
              <el-tag :type="store.selectedIssue.blocking ? 'danger' : 'warning'" effect="plain">
                {{ store.selectedIssue.blocking ? '阻断发布' : '不阻断' }}
              </el-tag>
              <p>{{ store.selectedIssue.detail || '无额外详情' }}</p>
              <strong>修复建议</strong>
              <p>{{ store.selectedIssue.suggestion || '根据对象配置补齐契约信息' }}</p>
              <el-button v-if="store.selectedIssue.jumpTarget" type="primary" @click="jumpToIssue"
                >跳转修复</el-button
              >
            </template>
            <div v-else class="contract-check-panel__empty">暂无检查结果</div>
          </aside>
        </main>
      </section>
    </Transition>
  </Teleport>
</template>
```

- [ ] **步骤 3：实现脚本逻辑**

`DataContractCheckDialog.vue` 脚本使用：

```ts
<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { IconTablerX } from '@tabler/icons-vue'
import { useContractCheckStore } from '@/stores/contract-check.store'
import type { ContractCheckIssue, ContractCheckModule } from '@/api/schemas/contract-check.schema'

const props = defineProps<{
  modelValue: boolean
  projectId?: string | number
  currentModule?: ContractCheckModule | ''
  currentObjectType?: string
  currentObjectId?: string
}>()

const emit = defineEmits<{ (event: 'update:modelValue', value: boolean): void }>()
const router = useRouter()
const store = useContractCheckStore()
const withDiagnostics = ref(false)

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const mode = computed(() => (withDiagnostics.value ? 'contract_with_diagnostics' : 'contract'))
const summary = computed(() => store.latestResult?.summary ?? { failed: 0, pending: 0, warning: 0, passed: 0, blocking: 0 })
const progress = computed(() => store.latestResult?.progress ?? { currentStage: '未运行检查', checkedCount: 0, totalCount: 0 })
const progressPercent = computed(() => {
  if (!progress.value.totalCount) return store.running ? 10 : 0
  return Math.min(100, Math.round((progress.value.checkedCount / progress.value.totalCount) * 100))
})
const progressLabel = computed(() => `${progress.value.currentStage} ${progress.value.checkedCount}/${progress.value.totalCount}`)
const statusText = computed(() => {
  if (store.running) return '正在检查数据中心契约'
  if (!store.latestResult) return '尚未运行检查'
  if (store.latestResult.blocking) return `${store.latestResult.summary.blocking} 个阻断项需要处理`
  return '没有阻断发布的问题'
})

const moduleOptions: Array<{ label: string; value: ContractCheckModule }> = [
  { label: '接入源', value: 'access_source' },
  { label: '数据点', value: 'datapoint' },
  { label: '查询', value: 'query' },
  { label: '计算', value: 'compute' },
  { label: '报警', value: 'alarm' },
  { label: 'Artifact', value: 'artifact' },
]

watch(
  () => [visible.value, props.projectId] as const,
  ([open, projectId]) => {
    if (open && projectId) void store.fetchLatest(String(projectId))
  },
)

async function runProject() {
  if (!props.projectId) return
  await store.run(String(props.projectId), { scope: 'project', mode: mode.value })
}

async function runModule() {
  if (!props.projectId || !props.currentModule) {
    ElMessage.warning('当前模块不可检查')
    return
  }
  await store.run(String(props.projectId), { scope: 'module', mode: mode.value, module: props.currentModule })
}

async function runObject() {
  if (!props.projectId || !props.currentModule || !props.currentObjectType || !props.currentObjectId) {
    ElMessage.warning('当前对象不可检查')
    return
  }
  await store.run(String(props.projectId), {
    scope: 'object',
    mode: mode.value,
    module: props.currentModule,
    objectType: props.currentObjectType,
    objectId: props.currentObjectId,
  })
}

function selectIssue(issue?: ContractCheckIssue) {
  store.selectedIssueId = issue?.id ?? ''
}

async function jumpToIssue() {
  const target = store.selectedIssue?.jumpTarget
  if (!target) return
  visible.value = false
  await router.push({ name: target.routeName, params: target.params, query: target.query })
}
</script>
```

- [ ] **步骤 4：实现浮层样式**

在同文件 scoped style 中加入：

```css
.contract-check-panel {
  position: fixed;
  left: 96px;
  bottom: 24px;
  z-index: 2100;
  display: flex;
  flex-direction: column;
  width: min(920px, calc(100vw - 128px));
  height: min(720px, calc(100vh - 96px));
  border: 1px solid var(--dc-border);
  border-radius: 8px;
  background: var(--dc-surface);
  box-shadow: 0 22px 70px rgba(15, 23, 42, 0.22);
}

.contract-check-panel__header,
.contract-check-panel__actions,
.contract-check-panel__progress,
.contract-check-panel__filters {
  padding: 12px 14px;
  border-bottom: 1px solid var(--dc-border);
}

.contract-check-panel__header,
.contract-check-panel__actions,
.contract-check-panel__filters,
.contract-check-panel__summary {
  display: flex;
  align-items: center;
  gap: 10px;
}

.contract-check-panel__header {
  justify-content: space-between;
}

.contract-check-panel__header h2 {
  margin: 0;
  font-size: 18px;
  letter-spacing: 0;
}

.contract-check-panel__header p {
  margin: 4px 0 0;
  color: var(--dc-text-secondary);
  font-size: 12px;
}

.contract-check-panel__summary {
  padding: 10px 14px;
  border-bottom: 1px solid var(--dc-border);
}

.contract-check-panel__summary button {
  height: 28px;
  border: 1px solid var(--dc-border);
  border-radius: 6px;
  background: var(--dc-surface-subtle);
  color: var(--dc-text-secondary);
  font-size: 12px;
}

.contract-check-panel__body {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 280px;
  min-height: 0;
  flex: 1;
}

.contract-check-panel__detail {
  min-width: 0;
  padding: 14px;
  border-left: 1px solid var(--dc-border);
  overflow: auto;
}

.contract-check-panel__empty {
  color: var(--dc-text-muted);
  font-size: 13px;
}

.contract-check-panel-enter-active,
.contract-check-panel-leave-active {
  transition:
    opacity 0.16s ease,
    transform 0.16s ease;
}

.contract-check-panel-enter-from,
.contract-check-panel-leave-to {
  opacity: 0;
  transform: translateY(8px);
}
```

- [ ] **步骤 5：删除静态数据文件引用**

删除 `datacenter/src/components/contract/dataContractCheck.ts`，确认没有 import 残留：

```powershell
rg -n "dataContractCheck|dataContractCheckItems|dataContractCheckSummary" datacenter/src -S
```

预期：无输出。

- [ ] **步骤 6：运行类型检查**

```powershell
pnpm --filter datacenter typecheck
```

预期：PASS。

- [ ] **步骤 7：Commit**

```powershell
git add datacenter/src/components/contract/DataContractCheckDialog.vue datacenter/src/components/contract/dataContractCheck.ts datacenter/src/views/DataCenterNew.vue
git commit -m "feat(datacenter): 实现契约检查悬浮面板"
```

---

## 任务 10：后端接口、前端构建和验收检查

**文件：**

- 修改：`docs/统一REST接口规范与清单.md`
- 修改：`docs/superpowers/specs/2026-05-28-datacenter-contract-check-design.md`，仅在实际实现与设计存在字段名差异时同步。

- [ ] **步骤 1：更新 REST 文档**

在 `docs/统一REST接口规范与清单.md` 的契约检查段落中明确：

```markdown
契约检查 `run` 请求体支持 `scope`、`mode`、`module`、`objectType`、`objectId`、`trigger`。
响应包含 `status`、`blocking`、`progress`、`summary`、`issues`。
发布门禁以 `blocking=true` 为阻断依据，外部接入源连接诊断默认不阻断。
```

- [ ] **步骤 2：运行后端测试**

```powershell
go test ./... -count=1
```

工作目录：`D:\SVNCode\indu-forge\data_service`

预期：PASS。

- [ ] **步骤 3：运行前端类型检查**

```powershell
pnpm --filter datacenter typecheck
```

工作目录：`D:\SVNCode\indu-forge`

预期：PASS。

- [ ] **步骤 4：运行前端构建**

```powershell
pnpm --filter datacenter build
```

预期：PASS。允许既有 VueUse PURE 注释 warning 和 chunk size warning。

- [ ] **步骤 5：静态扫描旧入口**

```powershell
rg -n "前端本地模型|dataContractCheckItems|ContractCheckStatusSchema = z.enum\\(\\['pending', 'running'" datacenter/src data_service/internal -S
```

预期：无输出。

- [ ] **步骤 6：浏览器验收清单**

启动服务后人工检查：

- 左下角契约检查入口仍固定可见。
- 图标颜色能反映未运行、运行中、通过、warning、pending、failed。
- 点击入口打开悬浮面板，不压缩主工作区，不强遮罩。
- 全量检查能显示进度、汇总、问题表和详情。
- 默认隐藏 passed 明细，打开开关后显示。
- 当前模块检查会传 `scope=module` 和当前模块。
- 当前对象缺少上下文时给出提示，不发送错误请求。
- 外部连接诊断关闭时不主动测试外部网络。
- 检查结果存在阻断项时 `blocking=true`。

- [ ] **步骤 7：Commit**

```powershell
git add docs/统一REST接口规范与清单.md docs/superpowers/specs/2026-05-28-datacenter-contract-check-design.md
git commit -m "docs(data): 更新契约检查接口说明"
```

---

## 任务 11：记录运维发布调用契约

**文件：**

- 修改：`docs/统一REST接口规范与清单.md`
- 修改：`docs/superpowers/specs/2026-05-28-datacenter-contract-check-design.md`
- 修改：`docs/superpowers/specs/2026-05-28-datacenter-contract-check-ops-integration-temp-design.md`

- [ ] **步骤 1：在 REST 文档补充运维调用示例**

在 `docs/统一REST接口规范与清单.md` 的契约检查接口段落中加入：

````markdown
发布流程调用数据中心契约检查时，请求体固定使用：

```json
{
  "scope": "project",
  "mode": "contract",
  "trigger": "dev_core.publish"
}
```
````

调用方只按响应中的 `blocking` 字段做发布门禁判断。`blocking=true` 时中止发布；`blocking=false` 时可继续读取 `GET /api/v1/data/projects/{projectId}/artifact` 获取 `DataDomainArtifact`。

````

- [ ] **步骤 2：在设计文档补充当前开发边界**

确认 `docs/superpowers/specs/2026-05-28-datacenter-contract-check-design.md` 中包含以下边界：

```markdown
当前开发边界限定在 `data_service` 和 `datacenter`：本设计只要求数据中心提供可被运维发布流程调用的检查接口和稳定 `DataDomainArtifact`；`dev_core` 的发布制品装配、部署命令和节点适配检查作为后续运维实现范围。
```
````

- [ ] **步骤 3：运行文档占位符扫描**

```powershell
$pattern = @('T'+'ODO', '待'+'定', 'T'+'BD', 'x'+'xx', 'X'+'XX', '后续'+'实现', '补充'+'细节', '必要'+'时', '类似'+'任务', '添加'+'适当', '添加'+'验证', '处理'+'边界') -join '|'
rg -n $pattern docs/superpowers/specs/2026-05-28-datacenter-contract-check-design.md docs/superpowers/specs/2026-05-28-datacenter-contract-check-ops-integration-temp-design.md docs/统一REST接口规范与清单.md -S
```

预期：无输出。

- [ ] **步骤 4：Commit**

```powershell
git add docs/统一REST接口规范与清单.md docs/superpowers/specs/2026-05-28-datacenter-contract-check-design.md docs/superpowers/specs/2026-05-28-datacenter-contract-check-ops-integration-temp-design.md
git commit -m "docs(data): 明确契约检查运维调用边界"
```

---

## 自检

- 设计文档中的对象范围已映射到任务 5、6、7。
- 设计文档中的 UI 形态已映射到任务 8、9。
- 设计文档中的发布前调用与 API 结构已映射到任务 2、3、10、11。
- 设计文档中的“外部连接诊断不阻断”通过 `mode`、`category=diagnostic` 和 `blocking` 字段表达。
- 计划不要求开发态真实连通用户现场外部接入源。
- 计划保持现有接口入口，不引入新的发布编排职责。
- `dev_core` 只负责发布编排和阻断展示，不复制 data_service 的数据域检查规则。

```

```
