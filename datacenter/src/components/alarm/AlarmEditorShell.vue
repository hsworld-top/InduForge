<template>
  <section class="alarm-editor-shell">
    <template v-if="fileTabs.length">
      <header class="alarm-editor-shell__file-tabs">
        <el-tabs
          class="alarm-editor-shell__tabs"
          type="card"
          :model-value="activeId || ''"
          @tab-change="(name) => emit('activateTab', String(name))"
          @tab-remove="(name) => emit('closeTab', String(name))"
        >
          <el-tab-pane v-for="tab in fileTabs" :key="tab.id" :name="tab.id" :closable="true">
            <template #label>
              <span class="alarm-editor-shell__tab-label">
                <span>{{ tab.name }}</span>
                <em v-if="tab.dirty">*</em>
              </span>
            </template>
          </el-tab-pane>
        </el-tabs>
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
        :coverages="coverages"
        :coverage-loading="coverageLoading"
        @update="emit('update', $event)"
      />
    </div>
    <div v-else class="alarm-editor-shell__state">
      <strong>暂无选中策略</strong>
      <span>从左侧选择一条策略开始编辑</span>
    </div>

    <footer class="alarm-editor-shell__bottom" :class="{ 'is-collapsed': bottomPanelCollapsed }">
      <nav class="alarm-editor-shell__panel-tabs">
        <button
          v-for="tab in panelTabs"
          :key="tab.value"
          type="button"
          :class="{ 'is-active': visibleBottomTab === tab.value }"
          @click="openBottomPanel(tab.value)"
        >
          {{ tab.label }}
        </button>
        <span class="alarm-editor-shell__panel-summary">{{ bottomSummary }}</span>
        <button
          type="button"
          class="alarm-editor-shell__panel-toggle"
          :title="bottomPanelCollapsed ? '展开底部面板' : '收起底部面板'"
          :aria-label="bottomPanelCollapsed ? '展开底部面板' : '收起底部面板'"
          @click="toggleBottomPanel"
        >
          <IconTablerChevronUp
            class="alarm-editor-shell__panel-toggle-icon"
            :class="{ 'is-collapsed': bottomPanelCollapsed }"
          />
        </button>
      </nav>

      <template v-if="!bottomPanelCollapsed">
        <div v-if="!draft" class="alarm-editor-shell__panel">
          <span>未选择策略</span>
        </div>
        <div v-else-if="visibleBottomTab === 'config'" class="alarm-editor-shell__config">
          <AlarmSuppressionPanel :draft="draft" @update="emit('update', $event)" />
          <AlarmMessageTemplatePanel :draft="draft" @update="emit('update', $event)" />
        </div>
        <div v-else-if="visibleBottomTab === 'test'" class="alarm-editor-shell__panel">
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
      </template>
    </footer>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type {
  AlarmPolicyContract,
  AlarmPolicyCoverage,
  AlarmPolicyTrialPayload,
  AlarmPolicyTrialResult,
} from '@/api/schemas/alarm.schema'
import type { AlarmPolicyDraft } from '@/components/alarm/alarmPolicyModel'
import IconTablerChevronUp from '~icons/tabler/chevron-up'
import AlarmContractPanel from './AlarmContractPanel.vue'
import AlarmEditorHeader from './AlarmEditorHeader.vue'
import AlarmMessageTemplatePanel from './AlarmMessageTemplatePanel.vue'
import AlarmPolicyForm from './AlarmPolicyForm.vue'
import AlarmSuppressionPanel from './AlarmSuppressionPanel.vue'
import AlarmTestPanel from './AlarmTestPanel.vue'

export type AlarmEditorTab = 'config' | 'test' | 'contract'
export type AlarmEditorFileTab = {
  id: string
  name: string
  dirty: boolean
}

const props = defineProps<{
  projectId: string
  fileTabs: AlarmEditorFileTab[]
  activeId: string | null
  draft: AlarmPolicyDraft | null
  activeTab: AlarmEditorTab
  loading: boolean
  error: string
  saving: boolean
  deleting: boolean
  trialResult: AlarmPolicyTrialResult | null
  trialRunning: boolean
  trialError: string
  contract: AlarmPolicyContract | null
  contractLoading: boolean
  contractError: string
  coverages?: Record<string, AlarmPolicyCoverage>
  coverageLoading?: boolean
}>()

const emit = defineEmits<{
  update: [patch: Partial<AlarmPolicyDraft>]
  save: []
  toggle: []
  delete: []
  activateTab: [id: string]
  closeTab: [id: string]
  selectTab: [tab: AlarmEditorTab]
  runTrial: [payload: AlarmPolicyTrialPayload]
  refreshContract: []
  checkCurrent: []
  selectTarget: []
}>()

const panelTabs: Array<{ value: AlarmEditorTab; label: string }> = [
  { value: 'config', label: '配置' },
  { value: 'test', label: '试算' },
  { value: 'contract', label: '契约' },
]

const bottomPanelCollapsed = ref(true)
const selectedBottomTab = ref<AlarmEditorTab | null>(null)
const visibleBottomTab = computed(() => selectedBottomTab.value ?? props.activeTab)

watch(
  () => props.activeTab,
  (tab) => {
    selectedBottomTab.value = tab
  },
)

const bottomSummary = computed(() => {
  if (!props.draft) {
    return visibleBottomTab.value === 'config'
      ? '抑制策略 / 消息模板'
      : visibleBottomTab.value === 'test'
        ? '试算样本 / 结果'
        : '策略契约'
  }
  if (visibleBottomTab.value === 'config') {
    return '抑制策略 / 消息模板'
  }
  if (visibleBottomTab.value === 'test') {
    return trialRunning ? '试算中' : trialResult ? '已有试算结果' : '未试算'
  }
  return contractLoading ? '加载契约中' : contract ? '已加载契约' : '未加载契约'
})

const openBottomPanel = (tab: AlarmEditorTab) => {
  selectedBottomTab.value = tab
  emit('selectTab', tab)
  bottomPanelCollapsed.value = false
}

const toggleBottomPanel = () => {
  bottomPanelCollapsed.value = !bottomPanelCollapsed.value
}
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
  height: 40px;
  min-height: 40px;
  display: flex;
  align-items: stretch;
  padding: 0;
  background: var(--dc-surface-raised);
  overflow-x: auto;
}

.alarm-editor-shell__tabs {
  min-width: 0;
  flex: 1;
}

.alarm-editor-shell__tabs :deep(.el-tabs__header) {
  margin: 0;
  border-bottom-color: var(--dc-border);
}

.alarm-editor-shell__tabs :deep(.el-tabs__item) {
  min-width: 106px;
  max-width: 220px;
  padding-right: 34px;
  position: relative;
  border-radius: 0;
}

.alarm-editor-shell__tabs :deep(.el-tabs__nav) {
  border-radius: 0;
}

.alarm-editor-shell__tabs :deep(.el-tabs__item .is-icon-close) {
  position: absolute;
  top: 50%;
  right: 10px;
  width: 14px;
  height: 14px;
  margin-left: 0;
  transform: translateY(-50%);
}

.alarm-editor-shell__tabs :deep(.el-tabs__content) {
  display: none;
}

.alarm-editor-shell__tab-label {
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  overflow: hidden;
}

.alarm-editor-shell__tab-label span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.alarm-editor-shell__tab-label em {
  color: var(--dc-warning);
  font-style: normal;
}

.alarm-editor-shell__body {
  min-height: 0;
  flex: 1 1 auto;
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  align-content: start;
  gap: 0;
  overflow: hidden;
}

.alarm-editor-shell__body > :not(.alarm-editor-header) {
  margin: 6px;
  min-height: 0;
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
  display: flex;
  flex-direction: column;
  border-top: 1px solid var(--dc-border);
  background: var(--dc-surface);
}

.alarm-editor-shell__bottom.is-collapsed {
  flex-basis: 40px;
}

.alarm-editor-shell__panel-tabs {
  min-height: 40px;
  flex: 0 0 40px;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  background: var(--dc-surface-raised);
}

.alarm-editor-shell__panel-tabs button {
  height: 30px;
  padding: 0 10px;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  cursor: pointer;
  font-family: inherit;
  font-size: 12px;
  font-weight: 700;
}

.alarm-editor-shell__panel-tabs button.is-active {
  border-color: var(--dc-border);
  background: var(--dc-surface);
  color: var(--dc-primary);
}

.alarm-editor-shell__panel-summary {
  min-width: 0;
  margin-left: auto;
  overflow: hidden;
  color: var(--dc-text-muted);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.alarm-editor-shell__panel-toggle {
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
}

.alarm-editor-shell__panel-toggle:hover {
  background: var(--dc-surface);
  color: var(--dc-primary);
}

.alarm-editor-shell__panel-toggle-icon {
  width: 16px;
  height: 16px;
  transition: transform 0.16s ease;
}

.alarm-editor-shell__panel-toggle-icon.is-collapsed {
  transform: rotate(180deg);
}

.alarm-editor-shell__config {
  min-height: 0;
  flex: 1 1 auto;
  display: grid;
  grid-template-columns: minmax(300px, 0.9fr) minmax(360px, 1.1fr);
  gap: 12px;
  padding: 12px;
  overflow: auto;
}

.alarm-editor-shell__panel {
  min-height: 0;
  flex: 1 1 auto;
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
