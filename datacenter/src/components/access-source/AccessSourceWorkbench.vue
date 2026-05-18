<template>
  <div class="asw-container">
    <!-- 顶部固定头 -->
    <header class="asw-head">
      <button type="button" class="asw-back-btn" @click="$emit('back')">
        <IconTablerArrowLeft class="asw-back-btn__icon" />
        <span>返回接入源</span>
      </button>

      <div class="asw-head__info">
        <span class="asw-head__name">{{ connection.name || "未命名接入源" }}</span>
        <StatusBadge
          :tone="statusTone"
          :text="statusText"
        />
      </div>
    </header>

    <!-- 主体：由协议子壳填充 -->
    <div class="asw-body">
      <component
        :is="resolvedPanel"
        :connection="connection"
        :project-id="projectId"
        @back="$emit('back')"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import IconTablerArrowLeft from "~icons/tabler/arrow-left";
import StatusBadge from "@/components/shared/StatusBadge.vue";
import SqlWorkbench from "./workbench/SqlWorkbench.vue";
import MqttWorkbenchPanel from "./workbench/MqttWorkbenchPanel.vue";
import ProtocolPreviewPanel from "./workbench/ProtocolPreviewPanel.vue";
import ReadOnlyConfigPanel from "./workbench/ReadOnlyConfigPanel.vue";

type AccessSourceConnection = {
  id: string;
  name?: string;
  type?: string;
  status?: string;
  relationalConfig?: Record<string, any>;
  mqttConfig?: Record<string, any>;
  config?: Record<string, any>;
};

const props = defineProps<{
  connection: AccessSourceConnection;
  projectId: string;
}>();

defineEmits<{
  (event: "back"): void;
}>();

/* 按协议类型选对应子壳 */
const resolvedPanel = computed(() => {
  const type = props.connection.type || "";
  if (["relational", "mysql", "postgresql", "sqlserver", "tdengine"].includes(type)) {
    return SqlWorkbench;
  }
  if (type === "mqtt") {
    return MqttWorkbenchPanel;
  }
  if (["kafka", "http", "websocket", "redis"].includes(type)) {
    return ProtocolPreviewPanel;
  }
  if (["opcua", "opcda", "s7", "modbus"].includes(type)) {
    return ReadOnlyConfigPanel;
  }
  return ReadOnlyConfigPanel;
});

/* 状态文本与色调 */
const statusTone = computed(() => {
  const s = props.connection.status || "";
  if (s === "connected") return "success" as const;
  if (s === "disconnected") return "muted" as const;
  if (s === "error" || s === "degraded") return "danger" as const;
  return "info" as const;
});

const statusText = computed(() => {
  const s = props.connection.status || "";
  if (s === "connected") return "在线";
  if (s === "disconnected") return "离线";
  if (s === "error") return "异常";
  if (s === "degraded") return "降级";
  return "未知";
});
</script>

<style scoped>
.asw-container {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* 顶部头：高度 56px，下边框，raised 背景 */
.asw-head {
  flex: 0 0 56px;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 18px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
}

.asw-back-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 0 10px;
  height: 32px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
  color: var(--dc-text-secondary);
  font-size: 13px;
  cursor: pointer;
  transition: color 0.16s ease, border-color 0.16s ease;
}

.asw-back-btn:hover {
  color: var(--dc-primary);
  border-color: color-mix(in oklch, var(--dc-primary) 40%, var(--dc-border));
}

.asw-back-btn__icon {
  width: 15px;
  height: 15px;
  flex: 0 0 auto;
}

.asw-head__info {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.asw-head__name {
  color: var(--dc-text);
  font-size: 15px;
  font-weight: 700;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* 主体区撑满剩余高度 */
.asw-body {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}
</style>
