<template>
  <section class="alarm-editor-shell">
    <AlarmEditorHeader
      :draft="draft"
      :saving="saving"
      :deleting="deleting"
      @save="emit('save')"
      @toggle="emit('toggle')"
      @delete="emit('delete')"
    />

    <div v-if="error" class="alarm-editor-shell__state is-error">
      <strong>详情不可用</strong>
      <span>{{ error }}</span>
    </div>
    <div v-else-if="loading" class="alarm-editor-shell__state">
      <strong>正在读取</strong>
      <span>加载报警规则详情</span>
    </div>
    <div v-else-if="draft" class="alarm-editor-shell__body">
      <AlarmRuleForm :draft="draft" @update="emit('update', $event)" />
    </div>
    <div v-else class="alarm-editor-shell__state">
      <strong>暂无选中规则</strong>
      <span>从左侧选择一条规则开始编辑</span>
    </div>

    <footer class="alarm-editor-shell__bottom">
      <nav class="alarm-editor-shell__tabs">
        <button
          v-for="tab in tabs"
          :key="tab.value"
          type="button"
          :class="{ 'is-active': activeTab === tab.value }"
          @click="emit('selectTab', tab.value)"
        >
          {{ tab.label }}
        </button>
      </nav>

      <div v-if="!draft" class="alarm-editor-shell__panel">
        <span>未选择规则</span>
      </div>
      <div v-else-if="activeTab === 'config'" class="alarm-editor-shell__config">
        <AlarmSuppressionPanel :draft="draft" @update="emit('update', $event)" />
        <AlarmMessageTemplatePanel :draft="draft" @update="emit('update', $event)" />
      </div>
      <div v-else class="alarm-editor-shell__panel">
        <strong>{{ tabTitle }}</strong>
        <span>任务 8 实现</span>
      </div>
    </footer>
  </section>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type { AlarmRuleDraft } from "@/components/alarm/alarmRuleModel";
import AlarmEditorHeader from "./AlarmEditorHeader.vue";
import AlarmMessageTemplatePanel from "./AlarmMessageTemplatePanel.vue";
import AlarmRuleForm from "./AlarmRuleForm.vue";
import AlarmSuppressionPanel from "./AlarmSuppressionPanel.vue";

export type AlarmEditorTab = "config" | "test" | "contract";

const props = defineProps<{
  draft: AlarmRuleDraft | null;
  activeTab: AlarmEditorTab;
  loading: boolean;
  error: string;
  saving: boolean;
  deleting: boolean;
}>();

const emit = defineEmits<{
  update: [patch: Partial<AlarmRuleDraft>];
  save: [];
  toggle: [];
  delete: [];
  selectTab: [tab: AlarmEditorTab];
}>();

const tabs: Array<{ value: AlarmEditorTab; label: string }> = [
  { value: "config", label: "配置" },
  { value: "test", label: "试算" },
  { value: "contract", label: "契约" },
];

const tabTitle = computed(() =>
  props.activeTab === "test" ? "报警试算" : "规则契约",
);
</script>

<style scoped>
.alarm-editor-shell {
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-rows: auto minmax(0, 1fr) 260px;
  background: var(--dc-surface-subtle);
}

.alarm-editor-shell__body {
  min-height: 0;
  padding: 12px;
  overflow-y: auto;
}

.alarm-editor-shell__state {
  display: grid;
  align-content: start;
  gap: 8px;
  margin: 12px;
  padding: 14px;
  border: 1px dashed var(--dc-border);
  border-radius: var(--dc-radius-sm);
  color: var(--dc-text-secondary);
  font-size: 13px;
}

.alarm-editor-shell__state strong {
  color: var(--dc-text);
  font-size: 13px;
}

.alarm-editor-shell__state.is-error {
  border-color: rgba(220, 38, 38, 0.32);
}

.alarm-editor-shell__state.is-error strong,
.alarm-editor-shell__state.is-error span {
  color: var(--dc-danger, #b91c1c);
}

.alarm-editor-shell__bottom {
  min-height: 0;
  border-top: 1px solid var(--dc-border);
  background: var(--dc-surface);
}

.alarm-editor-shell__tabs {
  display: flex;
  gap: 6px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--dc-border);
}

.alarm-editor-shell__tabs button {
  height: 30px;
  padding: 0 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-text-secondary);
  cursor: pointer;
  font-family: inherit;
  font-size: 12px;
  font-weight: 700;
}

.alarm-editor-shell__tabs button.is-active {
  border-color: var(--dc-primary);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.alarm-editor-shell__config {
  height: calc(100% - 47px);
  min-height: 0;
  display: grid;
  grid-template-columns: minmax(300px, 0.9fr) minmax(360px, 1.1fr);
  gap: 12px;
  padding: 12px;
  overflow: auto;
}

.alarm-editor-shell__panel {
  height: calc(100% - 47px);
  display: grid;
  align-content: start;
  gap: 8px;
  padding: 12px;
  color: var(--dc-text-secondary);
  font-size: 13px;
}

.alarm-editor-shell__panel strong {
  color: var(--dc-text);
  font-size: 13px;
}

@media (max-width: 920px) {
  .alarm-editor-shell {
    grid-template-rows: auto minmax(360px, 1fr) auto;
  }

  .alarm-editor-shell__config {
    height: auto;
    grid-template-columns: 1fr;
  }
}
</style>
