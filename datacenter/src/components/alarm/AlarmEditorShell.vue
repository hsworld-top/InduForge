<template>
  <section class="alarm-editor-shell">
    <template v-if="fileTabs.length">
      <header class="alarm-editor-shell__file-tabs">
        <button
          v-for="tab in fileTabs"
          :key="tab.id"
          type="button"
          class="alarm-editor-shell__file-tab"
          :class="{ 'is-active': tab.id === activeId, 'is-dirty': tab.dirty }"
          @click="emit('activateTab', tab.id)"
        >
          <span>{{ tab.name }}</span>
          <em v-if="tab.dirty">*</em>
          <button
            type="button"
            class="alarm-editor-shell__close"
            aria-label="关闭标签"
            @click.stop="emit('closeTab', tab.id)"
          >
            <IconTablerX />
          </button>
        </button>
      </header>
    </template>

    <div v-if="error && !draft" class="alarm-editor-shell__state is-error">
      <strong>详情不可用</strong>
      <span>{{ error }}</span>
    </div>
    <div v-else-if="loading && !draft" class="alarm-editor-shell__state">
      <strong>正在读取</strong>
      <span>加载报警策略详情</span>
    </div>
    <div v-else-if="draft" class="alarm-editor-shell__body">
      <AlarmEditorHeader
        :draft="draft"
        :saving="saving"
        :deleting="deleting"
        @update="emit('update', $event)"
        @save="emit('save')"
        @toggle="emit('toggle')"
        @delete="emit('delete')"
        @check-current="emit('checkCurrent')"
        @select-target="emit('selectTarget')"
      />

      <AlarmPolicyForm
        :project-id="projectId"
        :draft="draft"
        @update="emit('update', $event)"
      />
    </div>
    <div v-else class="alarm-editor-shell__state">
      <strong>暂无选中策略</strong>
      <span>从左侧选择一条策略开始编辑</span>
    </div>

    <footer class="alarm-editor-shell__bottom">
      <nav class="alarm-editor-shell__panel-tabs">
        <button
          v-for="tab in panelTabs"
          :key="tab.value"
          type="button"
          :class="{ 'is-active': activeTab === tab.value }"
          @click="emit('selectTab', tab.value)"
        >
          {{ tab.label }}
        </button>
      </nav>

      <div v-if="!draft" class="alarm-editor-shell__panel">
        <span>未选择策略</span>
      </div>
      <div v-else-if="activeTab === 'config'" class="alarm-editor-shell__config">
        <AlarmSuppressionPanel :draft="draft" @update="emit('update', $event)" />
        <AlarmMessageTemplatePanel :draft="draft" @update="emit('update', $event)" />
      </div>
      <div v-else-if="activeTab === 'test'" class="alarm-editor-shell__panel">
        <AlarmTestPanel
          :result="trialResult"
          :running="trialRunning"
          :error="trialError"
          @run="emit('runTrial', $event)"
        />
      </div>
      <div v-else class="alarm-editor-shell__panel">
        <AlarmContractPanel
          :contract="contract"
          :loading="contractLoading"
          :error="contractError"
          @refresh="emit('refreshContract')"
        />
      </div>
    </footer>
  </section>
</template>

<script setup lang="ts">
import type {
  AlarmPolicyContract,
  AlarmPolicyTrialPayload,
  AlarmPolicyTrialResult,
} from "@/api/schemas/alarm.schema";
import IconTablerX from "~icons/tabler/x";
import type { AlarmPolicyDraft } from "@/components/alarm/alarmPolicyModel";
import AlarmContractPanel from "./AlarmContractPanel.vue";
import AlarmEditorHeader from "./AlarmEditorHeader.vue";
import AlarmMessageTemplatePanel from "./AlarmMessageTemplatePanel.vue";
import AlarmPolicyForm from "./AlarmPolicyForm.vue";
import AlarmSuppressionPanel from "./AlarmSuppressionPanel.vue";
import AlarmTestPanel from "./AlarmTestPanel.vue";

export type AlarmEditorTab = "config" | "test" | "contract";
export type AlarmEditorFileTab = {
  id: string;
  name: string;
  dirty: boolean;
};

defineProps<{
  projectId: string;
  fileTabs: AlarmEditorFileTab[];
  activeId: string | null;
  draft: AlarmPolicyDraft | null;
  activeTab: AlarmEditorTab;
  loading: boolean;
  error: string;
  saving: boolean;
  deleting: boolean;
  trialResult: AlarmPolicyTrialResult | null;
  trialRunning: boolean;
  trialError: string;
  contract: AlarmPolicyContract | null;
  contractLoading: boolean;
  contractError: string;
}>();

const emit = defineEmits<{
  update: [patch: Partial<AlarmPolicyDraft>];
  save: [];
  toggle: [];
  delete: [];
  activateTab: [id: string];
  closeTab: [id: string];
  selectTab: [tab: AlarmEditorTab];
  runTrial: [payload: AlarmPolicyTrialPayload];
  refreshContract: [];
  checkCurrent: [];
  selectTarget: [];
}>();

const panelTabs: Array<{ value: AlarmEditorTab; label: string }> = [
  { value: "config", label: "配置" },
  { value: "test", label: "试算" },
  { value: "contract", label: "契约" },
];
</script>

<style scoped>
.alarm-editor-shell {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--dc-surface-subtle);
}

.alarm-editor-shell__file-tabs {
  flex: 0 0 auto;
  height: 38px;
  display: flex;
  align-items: flex-end;
  gap: 2px;
  padding: 0 8px;
  background: var(--dc-surface-raised);
  overflow-x: auto;
}

.alarm-editor-shell__file-tab {
  height: 32px;
  max-width: 220px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 0 6px 0 10px;
  border: 1px solid transparent;
  border-bottom: none;
  border-radius: var(--dc-radius-sm) var(--dc-radius-sm) 0 0;
  background: transparent;
  color: var(--dc-text-secondary);
  font-size: 13px;
  font-weight: 700;
}

.alarm-editor-shell__file-tab.is-active {
  border-color: var(--dc-border);
  background: var(--dc-surface);
  color: var(--dc-primary);
}

.alarm-editor-shell__file-tab span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.alarm-editor-shell__file-tab em {
  color: var(--dc-warning);
  font-style: normal;
}

.alarm-editor-shell__close {
  width: 20px;
  height: 20px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: inherit;
}

.alarm-editor-shell__close:hover {
  background: var(--dc-surface-muted);
}

.alarm-editor-shell__body {
  min-height: 0;
  flex: 1 1 auto;
  display: grid;
  align-content: start;
  gap: 0;
  overflow-y: auto;
}

.alarm-editor-shell__body > :not(.alarm-editor-header) {
  margin: 6px;
}

.alarm-editor-shell__state {
  flex: 1 1 auto;
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
  flex: 0 0 260px;
  min-height: 0;
  border-top: 1px solid var(--dc-border);
  background: var(--dc-surface);
}

.alarm-editor-shell__panel-tabs {
  display: flex;
  gap: 6px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--dc-border);
}

.alarm-editor-shell__panel-tabs button {
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

.alarm-editor-shell__panel-tabs button.is-active {
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
    min-height: 0;
  }

  .alarm-editor-shell__config {
    height: auto;
    grid-template-columns: 1fr;
  }

  .alarm-editor-shell__bottom {
    flex-basis: auto;
  }
}
</style>
