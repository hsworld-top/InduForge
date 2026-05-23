<template>
  <section class="redis-manager">
    <header class="redis-manager__topbar">
      <button type="button" class="redis-manager__back" @click="$emit('back')">
        <IconTablerArrowLeft />
      </button>
      <div class="redis-manager__identity">
        <span>Redis</span>
        <strong>{{ connection.name || "未命名 Redis" }}</strong>
      </div>
      <WorkbenchStatusPill :label="statusLabel" :tone="statusTone" />
      <div class="redis-manager__topbar-spacer"></div>
      <el-input
        v-model="pattern"
        class="redis-manager__pattern"
        size="small"
        placeholder="Key pattern"
        @keyup.enter="loadKeys"
      />
      <el-button size="small" :loading="testing" @click="testConnection">测试连接</el-button>
      <el-button type="primary" size="small" :loading="loadingKeys" @click="loadKeys">
        刷新 Key
      </el-button>
    </header>

    <div class="redis-manager__layout">
      <aside class="redis-manager__sidebar">
        <section class="redis-manager__server">
          <div class="redis-manager__server-icon">
            <IconTablerDatabase />
          </div>
          <div>
            <strong>{{ endpoint }}</strong>
            <span>{{ modeLabel }} · DB {{ config.db ?? 0 }}</span>
          </div>
        </section>

        <nav class="redis-manager__tree">
          <button type="button" class="redis-manager__tree-node is-active">
            <IconTablerServer />
            <span>连接</span>
          </button>
          <button type="button" class="redis-manager__tree-node is-child is-active">
            <IconTablerDatabase />
            <span>db{{ config.db ?? 0 }}</span>
            <em>{{ keys.length }}</em>
          </button>
          <button
            v-for="group in keyGroups"
            :key="group.name"
            type="button"
            class="redis-manager__tree-node is-child"
            @click="pattern = `${group.name}:*`; loadKeys()"
          >
            <IconTablerFolder />
            <span>{{ group.name }}</span>
            <em>{{ group.count }}</em>
          </button>
        </nav>
      </aside>

      <main class="redis-manager__main">
        <section class="redis-manager__key-pane">
          <div class="redis-manager__pane-head">
            <div>
              <strong>Key 浏览器</strong>
              <span>{{ loadingKeys ? "正在扫描..." : `共 ${filteredKeys.length} 个 key` }}</span>
            </div>
            <el-input
              v-model="keyword"
              class="redis-manager__search"
              size="small"
              clearable
              placeholder="过滤 key/type"
            />
          </div>

          <div class="redis-manager__table">
            <div class="redis-manager__table-head">
              <span>Key</span>
              <span>Type</span>
              <span>TTL</span>
              <span>Size</span>
            </div>
            <button
              v-for="item in filteredKeys"
              :key="item.key"
              type="button"
              class="redis-manager__row"
              :class="{ 'is-selected': selectedKey === item.key }"
              @click="selectKey(item.key)"
            >
              <span :title="item.key">{{ item.key }}</span>
              <em>{{ item.type }}</em>
              <em>{{ formatTTL(item.ttl) }}</em>
              <em>{{ item.size }}</em>
            </button>
            <div v-if="!loadingKeys && filteredKeys.length === 0" class="redis-manager__empty">
              未找到匹配 key
            </div>
          </div>
        </section>

        <aside class="redis-manager__detail">
          <section class="redis-manager__value">
            <div class="redis-manager__pane-head">
              <div>
                <strong>Value Viewer</strong>
                <span>{{ selectedValue?.type || "未选择 key" }}</span>
              </div>
              <el-button size="small" :disabled="!selectedKey" :loading="loadingValue" @click="reloadValue">
                读取
              </el-button>
            </div>
            <dl v-if="selectedValue" class="redis-manager__value-meta">
              <dt>Key</dt>
              <dd :title="selectedValue.key">{{ selectedValue.key }}</dd>
              <dt>TTL</dt>
              <dd>{{ formatTTL(selectedValue.ttl) }}</dd>
            </dl>
            <pre class="redis-manager__value-body">{{ formattedValue }}</pre>
          </section>

          <section class="redis-manager__console">
            <div class="redis-manager__pane-head">
              <div>
                <strong>Console</strong>
                <span>只读命令</span>
              </div>
              <el-button size="small" :loading="runningCommand" @click="runCommand">
                执行
              </el-button>
            </div>
            <el-input
              v-model="commandText"
              class="redis-manager__command"
              size="small"
              placeholder="PING / GET key / HGETALL key"
              @keyup.enter="runCommand"
            />
            <pre class="redis-manager__console-output">{{ commandOutput }}</pre>
          </section>
        </aside>
      </main>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { ElMessage } from "element-plus";
import dataAPI from "@/api/data.api";
import { getApiErrorMessage } from "@/utils/request";
import WorkbenchStatusPill from "@/components/workbench/WorkbenchStatusPill.vue";
import IconTablerArrowLeft from "~icons/tabler/arrow-left";
import IconTablerDatabase from "~icons/tabler/database";
import IconTablerFolder from "~icons/tabler/folder";
import IconTablerServer from "~icons/tabler/server";

type AccessSourceConnection = {
  id: string;
  name?: string;
  type?: string;
  status?: string;
  config?: Record<string, any>;
};

type RedisKey = {
  key: string;
  type: string;
  ttl: number;
  size: number;
};

type RedisValue = {
  key: string;
  type: string;
  ttl: number;
  value: unknown;
};

const props = defineProps<{
  connection: AccessSourceConnection;
  projectId: string;
}>();

defineEmits<{
  (event: "back"): void;
}>();

const config = computed(() => props.connection.config || {});
const pattern = ref(String(config.value.keyPattern || "*"));
const keyword = ref("");
const keys = ref<RedisKey[]>([]);
const selectedKey = ref("");
const selectedValue = ref<RedisValue | null>(null);
const loadingKeys = ref(false);
const loadingValue = ref(false);
const testing = ref(false);
const runningCommand = ref(false);
const commandText = ref("PING");
const commandOutput = ref("等待执行命令");

const endpoint = computed(() => String(config.value.address || "未配置地址"));
const modeLabel = computed(() => String(config.value.mode || "standalone"));

const statusLabel = computed(() => {
  const labels: Record<string, string> = {
    connected: "在线",
    disconnected: "离线",
    error: "异常",
    unknown: "未知",
  };
  return labels[props.connection.status || "unknown"] || "未知";
});

const statusTone = computed(() => {
  if (props.connection.status === "connected") return "success";
  if (props.connection.status === "error") return "danger";
  if (props.connection.status === "disconnected") return "warning";
  return "neutral";
});

const filteredKeys = computed(() => {
  const text = keyword.value.trim().toLowerCase();
  if (!text) return keys.value;
  return keys.value.filter((item) =>
    `${item.key} ${item.type}`.toLowerCase().includes(text),
  );
});

const keyGroups = computed(() => {
  const counts = new Map<string, number>();
  for (const item of keys.value) {
    const group = item.key.includes(":") ? item.key.split(":")[0] : "(root)";
    counts.set(group, (counts.get(group) || 0) + 1);
  }
  return Array.from(counts.entries())
    .map(([name, count]) => ({ name, count }))
    .sort((a, b) => b.count - a.count)
    .slice(0, 12);
});

const formattedValue = computed(() => {
  if (!selectedValue.value) return "从左侧选择一个 key 查看内容";
  return formatPayload(selectedValue.value.value);
});

const loadKeys = async () => {
  loadingKeys.value = true;
  try {
    const response = await dataAPI.getRedisKeys(props.projectId, props.connection.id, {
      pattern: pattern.value || "*",
      limit: 500,
    });
    const payload = response?.data || response || {};
    keys.value = payload.list || [];
    if (keys.value.length > 0 && !selectedKey.value) {
      await selectKey(keys.value[0].key);
    }
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, "加载 Redis key 失败"));
  } finally {
    loadingKeys.value = false;
  }
};

const selectKey = async (key: string) => {
  selectedKey.value = key;
  await reloadValue();
};

const reloadValue = async () => {
  if (!selectedKey.value) return;
  loadingValue.value = true;
  try {
    const response = await dataAPI.getRedisValue(
      props.projectId,
      props.connection.id,
      selectedKey.value,
    );
    selectedValue.value = (response?.data || response) as RedisValue;
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, "读取 Redis key 失败"));
  } finally {
    loadingValue.value = false;
  }
};

const testConnection = async () => {
  testing.value = true;
  try {
    const response = await dataAPI.testConnection(props.projectId, {
      type: "redis",
      config: config.value,
    });
    const result = response?.data || response || {};
    ElMessage.success(result.message || "Redis 连接测试通过");
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, "Redis 连接测试失败"));
  } finally {
    testing.value = false;
  }
};

const runCommand = async () => {
  const parts = commandText.value.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return;
  runningCommand.value = true;
  try {
    const response = await dataAPI.executeRedisCommand(props.projectId, props.connection.id, {
      command: parts[0],
      args: parts.slice(1),
    });
    const payload = response?.data || response || {};
    commandOutput.value = formatPayload(payload.result);
  } catch (error) {
    commandOutput.value = getApiErrorMessage(error, "Redis 命令执行失败");
  } finally {
    runningCommand.value = false;
  }
};

const formatTTL = (ttl: number) => {
  if (ttl === -1) return "永久";
  if (ttl === -2) return "不存在";
  if (ttl < 0) return "-";
  return `${ttl}s`;
};

const formatPayload = (value: unknown) => {
  if (typeof value === "string") {
    try {
      return JSON.stringify(JSON.parse(value), null, 2);
    } catch {
      return value;
    }
  }
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
};

onMounted(() => {
  void loadKeys();
});
</script>

<style scoped>
.redis-manager {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: #f6f7f9;
}

.redis-manager__topbar {
  min-height: 50px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border-bottom: 1px solid #d7dce2;
  background: linear-gradient(180deg, #ffffff 0%, #edf1f5 100%);
}

.redis-manager__back {
  width: 30px;
  height: 30px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid #cbd5df;
  border-radius: 4px;
  background: #ffffff;
  color: #334155;
}

.redis-manager__back svg,
.redis-manager__server-icon svg,
.redis-manager__tree-node svg {
  width: 16px;
  height: 16px;
}

.redis-manager__identity {
  min-width: 0;
  display: grid;
  gap: 2px;
}

.redis-manager__identity span {
  color: #64748b;
  font-size: 11px;
  font-weight: 700;
}

.redis-manager__identity strong {
  overflow: hidden;
  color: #0f172a;
  font-size: 14px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.redis-manager__topbar-spacer {
  flex: 1;
  min-width: 12px;
}

.redis-manager__pattern {
  width: 240px;
}

.redis-manager__layout {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: 260px minmax(0, 1fr);
}

.redis-manager__sidebar {
  min-height: 0;
  display: flex;
  flex-direction: column;
  border-right: 1px solid #cfd6df;
  background: #eef2f6;
}

.redis-manager__server {
  display: grid;
  grid-template-columns: 34px minmax(0, 1fr);
  gap: 10px;
  padding: 12px;
  border-bottom: 1px solid #d8dee6;
}

.redis-manager__server-icon {
  width: 34px;
  height: 34px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  background: #dbeafe;
  color: #1d4ed8;
}

.redis-manager__server strong,
.redis-manager__server span {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.redis-manager__server strong {
  color: #0f172a;
  font-size: 13px;
}

.redis-manager__server span {
  margin-top: 3px;
  color: #64748b;
  font-size: 12px;
}

.redis-manager__tree {
  min-height: 0;
  overflow: auto;
  padding: 8px;
}

.redis-manager__tree-node {
  width: 100%;
  height: 30px;
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr) auto;
  align-items: center;
  gap: 7px;
  padding: 0 8px;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: #334155;
  font-size: 12px;
  text-align: left;
}

.redis-manager__tree-node.is-child {
  padding-left: 22px;
}

.redis-manager__tree-node.is-active,
.redis-manager__tree-node:hover {
  background: #dbeafe;
  color: #1d4ed8;
}

.redis-manager__tree-node span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.redis-manager__tree-node em {
  color: #64748b;
  font-style: normal;
  font-size: 11px;
}

.redis-manager__main {
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-columns: minmax(460px, 1fr) 420px;
  background: #ffffff;
}

.redis-manager__key-pane,
.redis-manager__detail {
  min-width: 0;
  min-height: 0;
}

.redis-manager__key-pane {
  display: flex;
  flex-direction: column;
  border-right: 1px solid #d7dce2;
}

.redis-manager__detail {
  display: grid;
  grid-template-rows: minmax(0, 1fr) 210px;
}

.redis-manager__pane-head {
  min-height: 44px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 8px 10px;
  border-bottom: 1px solid #d7dce2;
  background: #f8fafc;
}

.redis-manager__pane-head strong,
.redis-manager__pane-head span {
  display: block;
}

.redis-manager__pane-head strong {
  color: #0f172a;
  font-size: 13px;
}

.redis-manager__pane-head span {
  margin-top: 2px;
  color: #64748b;
  font-size: 11px;
}

.redis-manager__search {
  width: 220px;
}

.redis-manager__table {
  min-height: 0;
  overflow: auto;
}

.redis-manager__table-head,
.redis-manager__row {
  display: grid;
  grid-template-columns: minmax(220px, 1fr) 82px 74px 64px;
  align-items: center;
}

.redis-manager__table-head {
  position: sticky;
  top: 0;
  z-index: 1;
  height: 30px;
  padding: 0 10px;
  border-bottom: 1px solid #d7dce2;
  background: #eef2f6;
  color: #475569;
  font-size: 11px;
  font-weight: 700;
}

.redis-manager__row {
  width: 100%;
  min-height: 31px;
  padding: 0 10px;
  border: 0;
  border-bottom: 1px solid #eef2f6;
  background: #ffffff;
  color: #0f172a;
  font-size: 12px;
  text-align: left;
}

.redis-manager__row:hover,
.redis-manager__row.is-selected {
  background: #eaf2ff;
}

.redis-manager__row span {
  overflow: hidden;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.redis-manager__row em {
  color: #64748b;
  font-style: normal;
}

.redis-manager__empty {
  padding: 28px;
  color: #64748b;
  font-size: 13px;
  text-align: center;
}

.redis-manager__value,
.redis-manager__console {
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.redis-manager__value {
  border-bottom: 1px solid #d7dce2;
}

.redis-manager__value-meta {
  display: grid;
  grid-template-columns: 48px minmax(0, 1fr);
  gap: 6px 8px;
  margin: 0;
  padding: 10px;
  border-bottom: 1px solid #eef2f6;
  font-size: 12px;
}

.redis-manager__value-meta dt {
  color: #64748b;
}

.redis-manager__value-meta dd {
  min-width: 0;
  margin: 0;
  overflow: hidden;
  color: #0f172a;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.redis-manager__value-body,
.redis-manager__console-output {
  flex: 1;
  min-height: 0;
  margin: 0;
  padding: 10px;
  overflow: auto;
  background: #0f172a;
  color: #e2e8f0;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
  line-height: 1.55;
  white-space: pre-wrap;
  word-break: break-word;
}

.redis-manager__command {
  padding: 10px;
  border-bottom: 1px solid #eef2f6;
}

@media (max-width: 1180px) {
  .redis-manager__layout {
    grid-template-columns: 220px minmax(0, 1fr);
  }

  .redis-manager__main {
    grid-template-columns: 1fr;
    grid-template-rows: minmax(320px, 1fr) 380px;
  }

  .redis-manager__detail {
    border-top: 1px solid #d7dce2;
    grid-template-columns: 1fr 1fr;
    grid-template-rows: 1fr;
  }
}
</style>
