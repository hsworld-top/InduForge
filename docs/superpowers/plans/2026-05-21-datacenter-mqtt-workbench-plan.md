# 数据中心 MQTT 接入源工作台实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 将 MQTT 查看消息与变量管理收敛到新版接入源工作台，必要时同步校准 `data_service` 后端契约，并删除旧 MQTT tab 入口。

**架构：** `MqttWorkbench.vue` 作为唯一 MQTT 工作台容器，负责 preview session、MQTT 启动、页签和 socket 订阅清理。通用消息流 UI 抽到 `components/workbench/`，MQTT 面板只保留协议语义。后端 MQTT 列表类响应统一输出 `data.list` 与 `data.pagination`，旧前端 MQTT 入口不做兼容。

**技术栈：** Vue 3、Element Plus、Pinia、Socket.IO、Go、pgx、Paho MQTT、Vitest、Go test。

---

## 文件结构

- 修改：`data_service/internal/http/handler/mqtt_handler.go`  
  将订阅消息列表响应从数组调整为 `{ list, pagination }`。
- 修改：`data_service/internal/http/handler/mqtt_management_handler.go`  
  将 MQTT 订阅、变量组、变量列表响应统一为 `{ list, pagination }`；单个创建/更新响应保持实体。
- 修改：`data_service/tests/integration/mqtt_api_test.go`  
  更新消息列表响应断言，覆盖 `data.list` 与分页字段。
- 修改：`datacenter/src/api/data.api.ts`  
  增加 MQTT 列表响应归一化工具，暴露统一 `list/pagination` 数据。
- 创建：`datacenter/src/components/workbench/WorkbenchStatusPill.vue`  
  通用紧凑状态标签。
- 创建：`datacenter/src/components/workbench/WorkbenchStreamToolbar.vue`  
  通用消息流工具栏：连接状态、搜索、显示条数、格式化、清空、刷新。
- 创建：`datacenter/src/components/workbench/WorkbenchStreamMessageList.vue`  
  通用消息流列表：加载、空态、复制、格式化 payload。
- 修改：`datacenter/src/components/mqtt/MqttMessageViewer.vue`  
  改成新版工作台风格，复用公共消息流组件。
- 修改：`datacenter/src/components/mqtt/MqttTagList.vue`  
  改成新版工作台风格，并读取统一 `data.list`。
- 修改：`datacenter/src/components/mqtt/MqttTagMonitor.vue`  
  改成新版工作台风格，读取统一 `data.list`，保留 tag 实时订阅清理。
- 修改：`datacenter/src/components/mqtt/TagItem.vue`  
  与新版变量列表视觉统一。
- 修改：`datacenter/src/components/mqtt/MqttSubscriptionList.vue`  
  读取统一 `data.list`，样式使用 `--dc-*`。
- 修改：`datacenter/src/components/mqtt/MqttWorkbench.vue`  
  集中处理 preview session、启动状态、页签、消息订阅 cleanup、订阅树刷新。
- 修改：`datacenter/src/views/DataCenterNew.vue`  
  删除旧 MQTT tab 模板、import、refs、打开函数和旧事件绑定。
- 修改：`datacenter/src/components/connection/ConnectionList.vue`  
  删除或收敛只服务旧 MQTT tab 的事件出口；保留连接树展示时不再触发旧消息/变量页签。
- 修改：`datacenter/tests/mqtt-socket-shared.test.ts`  
  如 socket 引用计数行为被调整，同步覆盖订阅释放。

---

### 任务 1：校准后端 MQTT 列表响应契约

**文件：**
- 修改：`data_service/internal/http/handler/mqtt_handler.go`
- 修改：`data_service/internal/http/handler/mqtt_management_handler.go`
- 修改：`data_service/tests/integration/mqtt_api_test.go`

- [ ] **步骤 1：更新后端集成测试期望**

在 `data_service/tests/integration/mqtt_api_test.go` 中新增列表 envelope 类型，并修改 `mustListMqttMessages`。

```go
type mqttListResponse[T any] struct {
	List       []T                `json:"list"`
	Pagination mqttPaginationData `json:"pagination"`
}

type mqttPaginationData struct {
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

func mustListMqttMessages(t *testing.T, baseURL, token, projectID, subscriptionID string, limit int) []mqttMessagePayload {
	t.Helper()

	url := baseURL + "/api/v1/data/projects/" + projectID + "/mqtt/subscriptions/" + subscriptionID + "/messages?limit=" + strconv.Itoa(limit)

	responseEnvelope := doJSONRequest(t, http.MethodGet, url, token, nil)
	var result mqttListResponse[mqttMessagePayload]
	if err := json.Unmarshal(responseEnvelope.Data, &result); err != nil {
		t.Fatalf("decode mqtt messages response failed: %v", err)
	}
	if result.Pagination.Page != 1 {
		t.Fatalf("expected message page 1, got %d", result.Pagination.Page)
	}
	if result.Pagination.PageSize != limit {
		t.Fatalf("expected message pageSize %d, got %d", limit, result.Pagination.PageSize)
	}
	return result.List
}
```

- [ ] **步骤 2：运行测试确认失败**

运行：

```powershell
go test ./tests/integration -run TestMqttConnectionLifecycle -count=1
```

工作目录：`data_service`

预期：失败，原因是 `messages` 当前响应仍是数组，不能解码成 `{ list, pagination }`。

- [ ] **步骤 3：实现消息列表响应**

在 `data_service/internal/http/handler/mqtt_handler.go` 的 `ListMessages` 中，把 `WriteSuccess` 数据改成统一列表结构。

```go
messages, err := h.service.ListMessages(r.Context(), r.PathValue("projectId"), r.PathValue("subscriptionId"), limit)
if err != nil {
	return normalizeRepresentativeHandlerError(err)
}

response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{
	"list": messages,
	"pagination": map[string]int{
		"page":       1,
		"pageSize":   limit,
		"total":      len(messages),
		"totalPages": 1,
	},
})
return nil
```

说明：消息缓存接口目前只支持最近 N 条，不新增 offset 分页。`total` 表示本次返回数量，避免前端误以为是全库总数。

- [ ] **步骤 4：统一 MQTT 管理列表响应字段**

在 `data_service/internal/http/handler/mqtt_management_handler.go` 中修改以下响应：

`ListSubscriptions`：

```go
response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{
	"list": subscriptions,
	"pagination": map[string]int{
		"page":       page,
		"pageSize":   pageSize,
		"total":      total,
		"totalPages": totalPages,
	},
})
```

`ListTagGroups`：

```go
response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{
	"list": groups,
	"pagination": map[string]int{
		"page":       1,
		"pageSize":   len(groups),
		"total":      len(groups),
		"totalPages": 1,
	},
})
```

`ListTagsBySubscription` 和 `ListTagsByProject`：

```go
response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{
	"list": tags,
	"pagination": map[string]int{
		"page":       page,
		"pageSize":   pageSize,
		"total":      total,
		"totalPages": totalPages,
	},
})
```

保留 `CreateTagsBatch` 的 `{ "tags": result }`，因为它不是列表查询接口。

- [ ] **步骤 5：验证后端 MQTT 测试**

运行：

```powershell
go test ./tests/integration -run TestMqttConnectionLifecycle -count=1
```

工作目录：`data_service`

预期：PASS。

- [ ] **步骤 6：后端格式化与提交**

运行：

```powershell
gofmt -w internal/http/handler/mqtt_handler.go internal/http/handler/mqtt_management_handler.go tests/integration/mqtt_api_test.go
go test ./tests/integration -run TestMqttConnectionLifecycle -count=1
git add internal/http/handler/mqtt_handler.go internal/http/handler/mqtt_management_handler.go tests/integration/mqtt_api_test.go
git commit -m "feat(data_service): 统一MQTT列表响应结构"
```

工作目录：`data_service`

预期：格式化完成、测试通过、提交成功。

---

### 任务 2：统一前端 MQTT API 读取结构

**文件：**
- 修改：`datacenter/src/api/data.api.ts`
- 测试：后续由组件构建验证覆盖

- [ ] **步骤 1：增加 MQTT 列表归一化工具**

在 `datacenter/src/api/data.api.ts` 的 MQTT API 区域前加入：

```ts
const normalizeMqttListPayload = (payload, legacyKey = "") => {
  const data = payload?.data ?? payload ?? {};
  const list = Array.isArray(data.list)
    ? data.list
    : legacyKey && Array.isArray(data[legacyKey])
      ? data[legacyKey]
      : Array.isArray(data)
        ? data
        : [];
  const pagination = data.pagination || {
    page: 1,
    pageSize: list.length,
    total: list.length,
    totalPages: list.length > 0 ? 1 : 0,
  };
  return { list, pagination };
};
```

说明：虽然本轮不保留旧入口兼容，工具中保留 `legacyKey` 是为了让同一个函数能在后端迁移过程中稳定处理本地 mock 或测试返回，不增加 UI 分支。

- [ ] **步骤 2：包装 MQTT 列表 API 返回值**

把以下函数的 `request(...)` 改成 `.then(...)` 归一化：

```ts
export const getMqttSubscriptions = (projectId, connectionId) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/connections/${connectionId}/subscriptions`,
    method: "get",
  }).then((response) => ({
    ...response,
    data: normalizeMqttListPayload(response, "subscriptions"),
  }));
};

export const getMqttSubscriptionMessages = (projectId, subscriptionId, params = {}) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/subscriptions/${subscriptionId}/messages`,
    method: "get",
    params,
  }).then((response) => ({
    ...response,
    data: normalizeMqttListPayload(response),
  }));
};

export const getMqttTagGroups = (projectId, subscriptionId, params = {}) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/subscriptions/${subscriptionId}/tag-groups`,
    method: "get",
    params,
  }).then((response) => ({
    ...response,
    data: normalizeMqttListPayload(response, "groups"),
  }));
};

export const getMqttTags = (projectId, subscriptionId, params = {}) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/subscriptions/${subscriptionId}/tags`,
    method: "get",
    params,
  }).then((response) => ({
    ...response,
    data: normalizeMqttListPayload(response, "tags"),
  }));
};

export const getProjectMqttTags = (projectId, params = {}) => {
  return request({
    url: `/data/projects/${projectId}/mqtt/tags`,
    method: "get",
    params,
  }).then((response) => ({
    ...response,
    data: normalizeMqttListPayload(response, "tags"),
  }));
};
```

- [ ] **步骤 3：临时构建验证**

运行：

```powershell
pnpm --dir datacenter build
```

预期：如果组件仍读取 `response.data?.subscriptions` 或数组，构建可能仍通过；后续任务会改调用点。

---

### 任务 3：创建通用工作台消息流组件

**文件：**
- 创建：`datacenter/src/components/workbench/WorkbenchStatusPill.vue`
- 创建：`datacenter/src/components/workbench/WorkbenchStreamToolbar.vue`
- 创建：`datacenter/src/components/workbench/WorkbenchStreamMessageList.vue`

- [ ] **步骤 1：创建状态标签组件**

新增 `datacenter/src/components/workbench/WorkbenchStatusPill.vue`：

```vue
<template>
  <span class="workbench-status-pill" :class="`is-${tone}`">
    <slot>{{ label }}</slot>
  </span>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    tone?: "info" | "success" | "warning" | "danger";
    label?: string;
  }>(),
  {
    tone: "info",
    label: "",
  },
);
</script>

<style scoped>
.workbench-status-pill {
  min-height: 22px;
  display: inline-flex;
  align-items: center;
  padding: 0 7px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-xs);
  background: var(--dc-surface-muted);
  color: var(--dc-text-secondary);
  font-size: 11px;
  font-weight: 700;
  line-height: 1;
  white-space: nowrap;
}

.workbench-status-pill.is-success {
  border-color: color-mix(in oklch, var(--dc-success) 28%, var(--dc-border));
  background: var(--dc-success-soft);
  color: var(--dc-success);
}

.workbench-status-pill.is-warning {
  border-color: color-mix(in oklch, var(--dc-warning) 28%, var(--dc-border));
  background: var(--dc-warning-soft);
  color: var(--dc-warning);
}

.workbench-status-pill.is-danger {
  border-color: color-mix(in oklch, var(--dc-danger) 28%, var(--dc-border));
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
}
</style>
```

- [ ] **步骤 2：创建消息流工具栏组件**

新增 `datacenter/src/components/workbench/WorkbenchStreamToolbar.vue`：

```vue
<template>
  <header class="workbench-stream-toolbar">
    <div class="workbench-stream-toolbar__title">
      <slot name="title" />
      <WorkbenchStatusPill :tone="connected ? 'success' : 'info'">
        {{ connected ? "已连接" : "未连接" }}
      </WorkbenchStatusPill>
    </div>

    <div class="workbench-stream-toolbar__actions">
      <el-input
        :model-value="search"
        class="workbench-stream-toolbar__search"
        size="small"
        clearable
        placeholder="搜索消息"
        @update:model-value="$emit('update:search', String($event || ''))"
      />
      <el-input-number
        :model-value="limit"
        :min="10"
        :max="1000"
        :step="10"
        size="small"
        controls-position="right"
        @update:model-value="$emit('update:limit', Number($event || 100))"
      />
      <button type="button" class="workbench-stream-toolbar__icon" title="刷新" @click="$emit('refresh')">
        <IconTablerRefresh />
      </button>
      <button type="button" class="workbench-stream-toolbar__icon" title="清空" @click="$emit('clear')">
        <IconTablerTrash />
      </button>
      <button
        type="button"
        class="workbench-stream-toolbar__toggle"
        :class="{ 'is-active': formatJson }"
        @click="$emit('update:formatJson', !formatJson)"
      >
        JSON
      </button>
    </div>
  </header>
</template>

<script setup lang="ts">
import IconTablerRefresh from "~icons/tabler/refresh";
import IconTablerTrash from "~icons/tabler/trash";
import WorkbenchStatusPill from "./WorkbenchStatusPill.vue";

defineProps<{
  connected: boolean;
  search: string;
  limit: number;
  formatJson: boolean;
}>();

defineEmits<{
  (event: "update:search", value: string): void;
  (event: "update:limit", value: number): void;
  (event: "update:formatJson", value: boolean): void;
  (event: "refresh"): void;
  (event: "clear"): void;
}>();
</script>

<style scoped>
.workbench-stream-toolbar {
  min-height: 46px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
}

.workbench-stream-toolbar__title,
.workbench-stream-toolbar__actions {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 8px;
}

.workbench-stream-toolbar__title {
  color: var(--dc-text);
  font-size: 12px;
  font-weight: 700;
}

.workbench-stream-toolbar__actions {
  flex-shrink: 0;
}

.workbench-stream-toolbar__search {
  width: 210px;
}

.workbench-stream-toolbar__icon,
.workbench-stream-toolbar__toggle {
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
  font-size: 11px;
  font-weight: 700;
}

.workbench-stream-toolbar__icon {
  width: 28px;
}

.workbench-stream-toolbar__icon svg {
  width: 15px;
  height: 15px;
}

.workbench-stream-toolbar__toggle {
  min-width: 42px;
  padding: 0 8px;
}

.workbench-stream-toolbar__icon:hover,
.workbench-stream-toolbar__toggle:hover,
.workbench-stream-toolbar__toggle.is-active {
  border-color: color-mix(in oklch, var(--dc-primary) 28%, var(--dc-border));
  color: var(--dc-primary);
}

@media (max-width: 900px) {
  .workbench-stream-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .workbench-stream-toolbar__actions {
    flex-wrap: wrap;
  }

  .workbench-stream-toolbar__search {
    width: 100%;
  }
}
</style>
```

- [ ] **步骤 3：创建消息列表组件**

新增 `datacenter/src/components/workbench/WorkbenchStreamMessageList.vue`：

```vue
<template>
  <section class="workbench-stream-list">
    <div v-if="loading" class="workbench-stream-list__state">
      <IconTablerLoader2 />
      <span>加载消息...</span>
    </div>
    <div v-else-if="messages.length === 0" class="workbench-stream-list__state">
      <IconTablerInbox />
      <strong>暂无消息</strong>
      <span>等待订阅接收数据。</span>
    </div>
    <button
      v-for="message in messages"
      v-else
      :key="message.id"
      type="button"
      class="workbench-stream-list__item"
      @click="$emit('select', message)"
    >
      <div class="workbench-stream-list__meta">
        <WorkbenchStatusPill tone="info">QoS {{ message.qos ?? 0 }}</WorkbenchStatusPill>
        <span>{{ message.topic || "-" }}</span>
        <time>{{ message.timeText || "-" }}</time>
        <button type="button" title="复制 Payload" @click.stop="$emit('copy', message)">
          <IconTablerCopy />
        </button>
      </div>
      <pre>{{ message.payloadText }}</pre>
    </button>
  </section>
</template>

<script setup lang="ts">
import IconTablerCopy from "~icons/tabler/copy";
import IconTablerInbox from "~icons/tabler/inbox";
import IconTablerLoader2 from "~icons/tabler/loader-2";
import WorkbenchStatusPill from "./WorkbenchStatusPill.vue";

defineProps<{
  loading: boolean;
  messages: Array<{
    id: string | number;
    topic?: string;
    qos?: number;
    timeText?: string;
    payloadText: string;
  }>;
}>();

defineEmits<{
  (event: "select", message: any): void;
  (event: "copy", message: any): void;
}>();
</script>

<style scoped>
.workbench-stream-list {
  min-height: 0;
  flex: 1;
  overflow-y: auto;
  background: var(--dc-surface-raised);
}

.workbench-stream-list__state {
  height: 100%;
  min-height: 260px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--dc-text-muted);
  font-size: 12px;
}

.workbench-stream-list__state svg {
  width: 34px;
  height: 34px;
}

.workbench-stream-list__item {
  width: 100%;
  display: grid;
  gap: 8px;
  padding: 10px 12px;
  border: 0;
  border-bottom: 1px solid var(--dc-border);
  background: transparent;
  color: var(--dc-text);
  text-align: left;
}

.workbench-stream-list__item:hover {
  background: var(--dc-surface-subtle);
}

.workbench-stream-list__meta {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  color: var(--dc-text-muted);
  font-size: 11px;
}

.workbench-stream-list__meta span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.workbench-stream-list__meta time {
  margin-left: auto;
  white-space: nowrap;
}

.workbench-stream-list__meta button {
  width: 26px;
  height: 26px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}

.workbench-stream-list__meta button:hover {
  color: var(--dc-primary);
}

.workbench-stream-list__meta button svg {
  width: 14px;
  height: 14px;
}

.workbench-stream-list__item pre {
  max-height: 300px;
  margin: 0;
  overflow: auto;
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-text);
  font-family: var(--dc-font-mono);
  font-size: 12px;
  line-height: 1.55;
  white-space: pre-wrap;
  word-break: break-word;
}
</style>
```

- [ ] **步骤 4：构建验证公共组件**

运行：

```powershell
pnpm --dir datacenter build
```

预期：PASS 或只暴露后续任务中旧组件调用结构问题。

---

### 任务 4：改造 MQTT 消息查看器

**文件：**
- 修改：`datacenter/src/components/mqtt/MqttMessageViewer.vue`

- [ ] **步骤 1：替换模板为新版工作台结构**

将 `MqttMessageViewer.vue` 模板改为：

```vue
<template>
  <div class="mqtt-message-viewer">
    <WorkbenchStreamToolbar
      v-model:search="searchText"
      v-model:limit="displayLimit"
      v-model:format-json="formatJson"
      :connected="isConnected"
      @refresh="loadMessages"
      @clear="handleClear"
    >
      <template #title>
        <IconTablerRss />
        <span>{{ subscription?.name || "消息查看器" }}</span>
        <WorkbenchStatusPill tone="info">{{ subscription?.topic || "-" }}</WorkbenchStatusPill>
      </template>
    </WorkbenchStreamToolbar>

    <WorkbenchStreamMessageList
      :loading="loading"
      :messages="displayMessages"
      @select="handleSelectMessage"
      @copy="handleCopyMessage"
    />

    <footer class="mqtt-message-viewer__status">
      <span>共 {{ messages.length }} 条消息</span>
      <span v-if="searchText">筛选后 {{ filteredMessages.length }} 条</span>
      <span v-if="lastMessageTime">最后消息 {{ formatTimestamp(lastMessageTime) }}</span>
    </footer>
  </div>
</template>
```

- [ ] **步骤 2：整理脚本逻辑**

保留 `defineExpose({ addMessage, setConnected, loadMessages })`，把历史消息读取改成统一 `data.list`：

```ts
const loadMessages = async () => {
  if (!props.subscription) return;

  loading.value = true;
  try {
    const response = await dataAPI.getMqttSubscriptionMessages(
      props.projectId,
      props.subscription.id,
      { limit: displayLimit.value },
    );
    messages.value = response.data?.list || [];
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, "加载消息失败"));
  } finally {
    loading.value = false;
  }
};
```

新增展示消息 computed：

```ts
const displayMessages = computed(() =>
  filteredMessages.value.slice(0, displayLimit.value).map((message) => ({
    id: message.id || `${message.topic}-${message.receivedAt || message.timestamp}`,
    topic: message.topic,
    qos: message.qos || message.QOS || 0,
    timeText: formatTimestamp(message.receivedAt || message.timestamp),
    payloadText: formatPayload(message.payload),
    raw: message,
  })),
);
```

把 `addMessage` 的输入兼容 socket snapshot：

```ts
const addMessage = async (message) => {
  if (!message) return;

  messages.value.unshift({
    id: message.id || `${Date.now()}-${Math.random()}`,
    subscriptionId: message.subscriptionId || props.subscription?.id,
    topic: message.topic,
    payload: message.payload ?? message.value ?? "",
    qos: message.qos || 0,
    timestamp: message.timestamp || message.receivedAt || Date.now(),
    receivedAt: message.receivedAt || message.timestamp || Date.now(),
  });

  if (messages.value.length > 1000) {
    messages.value = messages.value.slice(0, 1000);
  }
};
```

说明：不再用 `subscriptionEnabled` 阻止实时消息写入。查看消息是 preview 行为，订阅是否启用由后端运行时决定；前端只展示当前 socket 已收到的数据。

- [ ] **步骤 3：替换样式为 `--dc-*`**

删除 Tailwind 依赖类对应的旧样式，保留：

```css
.mqtt-message-viewer {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--dc-surface-raised);
}

.mqtt-message-viewer__status {
  min-height: 34px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 7px 12px;
  border-top: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
  color: var(--dc-text-muted);
  font-size: 11px;
}

.mqtt-message-viewer :deep(svg) {
  width: 15px;
  height: 15px;
}
```

- [ ] **步骤 4：构建验证**

运行：

```powershell
pnpm --dir datacenter build
```

预期：PASS；如果出现隐式 any 或事件类型问题，按组件现有 `@ts-nocheck` 使用情况最小修正。

---

### 任务 5：改造 MQTT 变量配置与监控面板

**文件：**
- 修改：`datacenter/src/components/mqtt/MqttTagList.vue`
- 修改：`datacenter/src/components/mqtt/MqttTagMonitor.vue`
- 修改：`datacenter/src/components/mqtt/TagItem.vue`

- [ ] **步骤 1：更新变量列表读取结构**

在 `MqttTagList.vue` 中修改加载函数：

```ts
const loadGroups = async () => {
  try {
    const response = await getMqttTagGroups(
      props.projectId,
      props.subscriptionId,
    );
    groups.value = response.data?.list || [];
    activeGroups.value = [
      "ungrouped",
      ...groups.value.map((group) => group.id),
    ];
  } catch (error) {
    console.error("Failed to load tag groups:", error);
    ElMessage.error(getApiErrorMessage(error, "加载变量组失败"));
  }
};

const loadTags = async () => {
  try {
    loading.value = true;
    const response = await getMqttTags(props.projectId, props.subscriptionId, {
      pageSize: 200,
    });
    tags.value = response.data?.list || [];
    await loadTagDatapoints(tags.value);
  } catch (error) {
    console.error("Failed to load tags:", error);
    ElMessage.error(getApiErrorMessage(error, "加载变量失败"));
  } finally {
    loading.value = false;
  }
};
```

- [ ] **步骤 2：更新监控面板读取结构**

在 `MqttTagMonitor.vue` 中修改 `fetchTags`：

```ts
const fetchTags = async () => {
  try {
    const res = await getMqttTags(props.projectId, props.subscriptionId, {
      pageSize: 200,
    });
    tags.value = res.data?.list || [];
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, "获取变量列表失败"));
  }
};
```

同时引入：

```ts
import { getApiErrorMessage } from "@/utils/request";
```

- [ ] **步骤 3：改造变量列表视觉**

把 `MqttTagList.vue` 根布局样式替换为：

```css
.mqtt-tag-list {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding: 0;
  background: var(--dc-surface-raised);
}

.mqtt-tag-list__toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
}

.tag-tree {
  margin: 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
  overflow: hidden;
}

.group-header {
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
}
```

模板中把顶部两个 `div` 合并为 `class="mqtt-tag-list__toolbar"`，搜索与按钮保留原功能。

- [ ] **步骤 4：改造监控视觉并保留订阅清理**

在 `MqttTagMonitor.vue` 保留 `onBeforeUnmount` 中的：

```ts
stopSocketWatch?.();
unsubscribeMessage?.();
unsubscribeTagSync(handleTagSyncEvent);
Array.from(tagSubscriptionCleanups.values()).forEach((cleanup) => {
  cleanup?.();
});
tagSubscriptionCleanups.clear();
disconnect();
```

将根样式调整为：

```css
.mqtt-tag-monitor {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--dc-surface-raised);
}

.mqtt-tag-monitor__head {
  min-height: 46px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
}

.tag-row {
  border-color: var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}

.tag-row-header {
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
}
```

模板顶部容器改为 `class="mqtt-tag-monitor__head"`。

- [ ] **步骤 5：改造 TagItem 视觉**

在 `TagItem.vue` 中保留事件和字段展示，样式使用：

```css
.tag-item {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 8px;
  padding: 9px 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}

.tag-item:hover {
  border-color: color-mix(in oklch, var(--dc-primary) 28%, var(--dc-border));
  background: var(--dc-primary-soft);
}
```

- [ ] **步骤 6：构建验证**

运行：

```powershell
pnpm --dir datacenter build
```

预期：PASS。

---

### 任务 6：收敛 MQTT 工作台容器

**文件：**
- 修改：`datacenter/src/components/mqtt/MqttSubscriptionList.vue`
- 修改：`datacenter/src/components/mqtt/MqttWorkbench.vue`

- [ ] **步骤 1：更新订阅列表读取结构**

在 `MqttSubscriptionList.vue` 中修改 `loadSubscriptions`：

```ts
const loadSubscriptions = async () => {
  loading.value = true;
  try {
    const response = await dataAPI.getMqttSubscriptions(
      props.projectId,
      props.connectionId,
    );
    subscriptions.value = response.data?.list || [];
  } catch (error) {
    ElMessage.error(
      t("subscription.loadFailed", {
        message: getApiErrorMessage(error, "加载订阅失败"),
      }),
    );
  } finally {
    loading.value = false;
  }
};
```

- [ ] **步骤 2：更新工作台订阅树读取结构**

在 `MqttWorkbench.vue` 中修改 `loadSubscriptions`：

```ts
const loadSubscriptions = async () => {
  loading.value = true;
  try {
    const response = await dataAPI.getMqttSubscriptions(
      projectIdText.value,
      props.connection.id,
    );
    subscriptions.value = response.data?.list || [];
    if (
      subscriptions.value.length > 0 &&
      !subscriptions.value.some(
        (subscription) => subscription.id === selectedSubscription.value?.id,
      )
    ) {
      selectedSubscription.value = subscriptions.value[0];
    }
    await subscriptionListRef.value?.loadSubscriptions?.();
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, "加载 MQTT 订阅失败"));
  } finally {
    loading.value = false;
  }
};
```

- [ ] **步骤 3：修正消息实时 handler**

在 `openMessages` 中将：

```ts
viewer?.addMessage?.(data.message);
```

改为：

```ts
viewer?.addMessage?.(data?.message || data);
```

说明：`data_service` socket 当前发出的 `mqtt:message` 是订阅快照本身，不包 `message` 字段。

- [ ] **步骤 4：确保切换连接时清理状态**

在 `MqttWorkbench.vue` 中增加对连接 id 的 watch：

```ts
watch(
  () => props.connection.id,
  async () => {
    Array.from(messageCleanups.values()).forEach((cleanup) => cleanup?.());
    messageCleanups.clear();
    messageViewerRefs.value.clear();
    tabs.value = [];
    activeTabId.value = "";
    selectedSubscription.value = null;
    connectionStarted.value = false;
    openSubscriptionList();
    await loadSubscriptions();
  },
);
```

并从 `vue` import 增加 `watch`。

- [ ] **步骤 5：订阅管理成功后刷新左树**

在模板的 `MqttSubscriptionList` 上增加：

```vue
@subscription-select="selectedSubscription = $event"
@view-messages="openMessages"
@manage-tags="openTagManager"
@subscription-deleted="handleSubscriptionDeleted"
```

保留现有事件，同时在 `MqttSubscriptionList.vue` 的 `handleDialogSuccess` 中 emit 刷新事件：

```ts
const emit = defineEmits([
  "view-messages",
  "subscription-select",
  "manage-tags",
  "subscription-deleted",
  "subscriptions-changed",
]);

const handleDialogSuccess = async () => {
  await loadSubscriptions();
  emit("subscriptions-changed");
};
```

在 `MqttWorkbench.vue` 增加：

```vue
@subscriptions-changed="loadSubscriptions"
```

- [ ] **步骤 6：构建验证**

运行：

```powershell
pnpm --dir datacenter build
```

预期：PASS。

---

### 任务 7：删除旧 DataCenterNew MQTT tab 入口

**文件：**
- 修改：`datacenter/src/views/DataCenterNew.vue`
- 修改：`datacenter/src/components/connection/ConnectionList.vue`

- [ ] **步骤 1：移除旧 MQTT tab 模板分支**

在 `DataCenterNew.vue` 中删除：

```vue
<div v-else-if="tab.type === 'mqtt-subscriptions'" ...>
  <MqttSubscriptionList ... />
</div>

<div v-else-if="tab.type === 'mqtt-messages'" ...>
  <MqttMessageViewer ... />
</div>

<div v-else-if="tab.type === 'mqtt-tags'" ...>
  <MqttTagList ... />
  <MqttTagMonitor ... />
</div>
```

- [ ] **步骤 2：移除旧 MQTT imports 与 refs**

删除 `DataCenterNew.vue` 中：

```ts
import MqttSubscriptionList from "@/components/mqtt/MqttSubscriptionList.vue";
import MqttMessageViewer from "@/components/mqtt/MqttMessageViewer.vue";
import MqttTagList from "@/components/mqtt/MqttTagList.vue";
import MqttTagMonitor from "@/components/mqtt/MqttTagMonitor.vue";
import { useMqttSocket } from "@/composables/useMqttSocket";
```

删除：

```ts
const { subscribeMessages } = useMqttSocket(projectId, previewSessionId);
const mqttMessageViewerRefs = ref(new Map());
const mqttSubscriptionListRefs = ref(new Map());
const setMessageViewerRef = (tabId, el) => { ... };
const setSubscriptionListRef = (tabId, el) => { ... };
```

- [ ] **步骤 3：移除旧 MQTT tab 函数**

删除 `DataCenterNew.vue` 中：

```ts
const openMqttSubscriptionList = (connection) => { ... };
const openMqttMessageViewer = async (connection, subscription) => { ... };
const openMqttTagManager = async (connection, subscription) => { ... };
const handleMqttSubscriptionDblClick = async (connection, subscription) => { ... };
const handleMqttSubscriptionView = async (connection, subscription) => { ... };
const handleMqttSubscriptionManage = async (connection, subscription) => { ... };
const handleMqttSubscriptionEdit = async (connection, subscription) => { ... };
const handleMqttSubscriptionDelete = async (connection, subscription) => { ... };
const handleMqttSubscriptionDeleted = (connection, subscription) => { ... };
```

保留打开接入源工作台的逻辑。若有连接树 MQTT 订阅右键事件，统一改为打开当前 MQTT 接入源工作台。

- [ ] **步骤 4：更新模板事件绑定**

在 `DataCenterNew.vue` 的 `ConnectionList` 使用处删除：

```vue
@mqtt-subscription-dblclick="handleMqttSubscriptionDblClick"
@mqtt-subscription-view="handleMqttSubscriptionView"
@mqtt-subscription-manage="handleMqttSubscriptionManage"
@mqtt-subscription-edit="handleMqttSubscriptionEdit"
@mqtt-subscription-delete="handleMqttSubscriptionDelete"
```

如果 `ConnectionList` 仍展示订阅右键菜单，事件不再由父组件消费。

- [ ] **步骤 5：收敛 ConnectionList 事件出口**

在 `ConnectionList.vue` 删除 `defineEmits` 里的：

```ts
"mqtt-subscription-dblclick",
"mqtt-subscription-view",
"mqtt-subscription-manage",
"mqtt-subscription-edit",
"mqtt-subscription-delete",
```

把订阅行双击和右键菜单改成提示打开新版工作台，或只保留展示不触发旧 tab：

```ts
const handleMqttSubscriptionDblClick = () => {
  ElMessage.info("请在 MQTT 接入源工作台中查看消息和变量");
};
```

若文件未引入 `ElMessage`，添加：

```ts
import { ElMessage } from "element-plus";
```

- [ ] **步骤 6：全局搜索旧引用**

运行：

```powershell
git grep -n "mqtt-subscription-view\\|openMqttMessageViewer\\|MqttMessageViewer\\|MqttTagMonitor" -- datacenter/src
```

预期：只剩 `components/mqtt/MqttWorkbench.vue` 和 MQTT 自身组件引用；`DataCenterNew.vue` 不再出现旧 MQTT tab 逻辑。

- [ ] **步骤 7：构建验证**

运行：

```powershell
pnpm --dir datacenter build
```

预期：PASS。

---

### 任务 8：补充 socket 引用计数测试与全量验证

**文件：**
- 修改：`datacenter/tests/mqtt-socket-shared.test.ts`
- 可选修改：`data_service/internal/http/socket/preview_socket_server_test.go`

- [ ] **步骤 1：确认前端共享 socket 测试覆盖释放**

在 `datacenter/tests/mqtt-socket-shared.test.ts` 中确认或补充测试：同一订阅两个 handle 订阅时只发送一次 `mqtt:subscribe`，释放最后一个 handle 时发送一次 `mqtt:unsubscribe`。

测试主体示例：

```ts
test("同一订阅引用计数归零后才发送 unsubscribe", () => {
  const emitted: Array<{ event: string; payload: any }> = [];
  const registry = createMqttSocketSharedRegistry({
    ioFactory: () => ({
      connected: true,
      on: () => {},
      onAny: () => {},
      emit: (event: string, payload: any) => emitted.push({ event, payload }),
      disconnect: () => {},
    }),
    getToken: () => "token",
    getApiUrl: () => "http://localhost:19602",
    logger: {},
  });

  const acquired = registry.acquire({
    projectId: "project-1",
    previewSessionId: "session-1",
  });
  expect(acquired).toBeTruthy();
  registry.subscribeSubscription(acquired!.key, "sub-1");
  registry.subscribeSubscription(acquired!.key, "sub-1");
  registry.unsubscribeSubscription(acquired!.key, "sub-1");
  registry.unsubscribeSubscription(acquired!.key, "sub-1");

  expect(emitted.filter((item) => item.event === "mqtt:subscribe")).toHaveLength(1);
  expect(emitted.filter((item) => item.event === "mqtt:unsubscribe")).toHaveLength(1);
});
```

- [ ] **步骤 2：运行前端相关测试**

运行：

```powershell
pnpm --dir datacenter test -- mqtt-socket-shared
```

预期：PASS。

- [ ] **步骤 3：运行后端相关测试**

运行：

```powershell
go test ./internal/http/socket ./tests/integration -run "TestPreviewSocketServer|TestMqtt" -count=1
```

工作目录：`data_service`

预期：PASS。

- [ ] **步骤 4：运行前端构建**

运行：

```powershell
pnpm --dir datacenter build
```

预期：PASS。

- [ ] **步骤 5：运行后端构建**

运行：

```powershell
go build ./...
```

工作目录：`data_service`

预期：PASS。

- [ ] **步骤 6：最终提交**

运行：

```powershell
git add `
  data_service/internal/http/handler/mqtt_handler.go `
  data_service/internal/http/handler/mqtt_management_handler.go `
  data_service/tests/integration/mqtt_api_test.go `
  datacenter/src/api/data.api.ts `
  datacenter/src/components/workbench/WorkbenchStatusPill.vue `
  datacenter/src/components/workbench/WorkbenchStreamToolbar.vue `
  datacenter/src/components/workbench/WorkbenchStreamMessageList.vue `
  datacenter/src/components/mqtt/MqttMessageViewer.vue `
  datacenter/src/components/mqtt/MqttTagList.vue `
  datacenter/src/components/mqtt/MqttTagMonitor.vue `
  datacenter/src/components/mqtt/TagItem.vue `
  datacenter/src/components/mqtt/MqttSubscriptionList.vue `
  datacenter/src/components/mqtt/MqttWorkbench.vue `
  datacenter/src/views/DataCenterNew.vue `
  datacenter/src/components/connection/ConnectionList.vue `
  datacenter/tests/mqtt-socket-shared.test.ts
git commit -m "feat(datacenter): 实现MQTT接入源工作台"
```

工作目录：仓库根目录。

预期：只提交本计划相关改动，不包含 `.agents/`、截图、未关联的用户改动。

---

## 手动验收

- 打开新版接入源列表，进入 MQTT 接入源工作台，默认看到“订阅管理”页签。
- 新建或编辑订阅后，订阅管理表格与左侧订阅树同步刷新。
- 点击某订阅“查看消息”，历史消息按最新在上展示，实时消息进入当前页签。
- 关闭消息页签后，继续推送消息不会写入已关闭页签。
- 点击某订阅“变量管理”，左侧能创建变量/分组，右侧能看到启用变量实时值。
- 停用或删除变量后，右侧监控取消对应订阅。
- 删除订阅后，该订阅的消息页签和变量页签关闭。
- 返回接入源列表再重新进入 MQTT 工作台，旧 preview session 不残留。
