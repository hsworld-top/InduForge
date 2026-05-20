<template>
  <section class="alarm-contract-panel">
    <div class="alarm-contract-panel__bar">
      <div>
        <strong>策略契约</strong>
        <span>Runtime Contract JSON</span>
      </div>
      <div class="alarm-contract-panel__actions">
        <button type="button" :disabled="loading" @click="emit('refresh')">
          {{ loading ? "刷新中" : "刷新" }}
        </button>
        <button type="button" :disabled="!contractText" @click="copyContract">
          复制
        </button>
      </div>
    </div>

    <p v-if="error" class="alarm-contract-panel__error">{{ error }}</p>
    <pre v-else>{{ contractText || "暂无契约数据" }}</pre>
  </section>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { ElMessage } from "element-plus";
import type { AlarmPolicyContract } from "@/api/schemas/alarm.schema";

const props = defineProps<{
  contract: AlarmPolicyContract | null;
  loading: boolean;
  error: string;
}>();

const emit = defineEmits<{
  refresh: [];
}>();

const contractText = computed(() =>
  props.contract ? JSON.stringify(props.contract, null, 2) : "",
);

const copyContract = async () => {
  if (!contractText.value) {
    return;
  }
  try {
    await navigator.clipboard.writeText(contractText.value);
    ElMessage.success("契约 JSON 已复制");
  } catch {
    ElMessage.error("复制失败，请手动复制");
  }
};
</script>

<style scoped>
.alarm-contract-panel {
  height: 100%;
  min-height: 0;
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  gap: 10px;
}

.alarm-contract-panel__bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.alarm-contract-panel__bar strong,
.alarm-contract-panel__bar span {
  display: block;
}

.alarm-contract-panel__bar strong {
  color: var(--dc-text);
  font-size: 13px;
}

.alarm-contract-panel__bar span {
  color: var(--dc-text-muted);
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
}

.alarm-contract-panel__actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.alarm-contract-panel__actions button {
  height: 30px;
  padding: 0 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-text);
  cursor: pointer;
  font-family: inherit;
  font-size: 12px;
  font-weight: 700;
}

.alarm-contract-panel__actions button:hover:not(:disabled) {
  border-color: var(--dc-primary);
  color: var(--dc-primary);
}

.alarm-contract-panel__actions button:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.alarm-contract-panel pre {
  min-height: 0;
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

.alarm-contract-panel__error {
  margin: 0;
  color: var(--dc-danger, #b91c1c);
  font-size: 12px;
}
</style>
