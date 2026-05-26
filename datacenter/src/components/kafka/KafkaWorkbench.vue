<template>
  <section class="kafka-workbench">
    <aside class="kafka-workbench__explorer">
      <WorkbenchSourceHeader
        :title="connection.name || '未命名 Kafka 接入源'"
        fallback-title="未命名 Kafka 接入源"
        :meta="sourceMetaRows"
        @back="$emit('back')"
      >
        <template #status>
          <button
            type="button"
            class="kafka-workbench__connect-action"
            :class="{ 'is-connected': connected }"
            :title="connected ? '断开' : '连接'"
            @click="toggleConnection"
          >
            <IconTablerLoader2 v-if="connecting" />
            <span>{{ connecting ? '连接中' : connected ? '已连接' : '连接' }}</span>
          </button>
        </template>
        <template #actions>
          <el-input
            v-model="filterText"
            class="kafka-workbench__search"
            size="small"
            clearable
            placeholder="筛选 Topic"
          />
          <button
            type="button"
            class="workbench-source-header__icon-action is-primary"
            title="新建 Topic 映射"
            aria-label="新建 Topic 映射"
            @click="openCreateMapping"
          >
            <IconTablerPlus />
          </button>
          <button
            type="button"
            class="workbench-source-header__icon-action"
            title="新建分组"
            aria-label="新建分组"
            @click="openCreateGroup"
          >
            <IconTablerFolderPlus />
          </button>
          <button
            type="button"
            class="workbench-source-header__icon-action"
            title="刷新"
            aria-label="刷新"
            @click="loadWorkbench"
          >
            <IconTablerRefresh />
          </button>
        </template>
      </WorkbenchSourceHeader>

      <div class="kafka-workbench__tree">
        <div v-if="loading" class="kafka-workbench__loading">
          <IconTablerLoader2 />
          <span>加载 Topic...</span>
        </div>
        <template v-else>
          <KafkaTopicTreeBranch
            v-for="group in filteredTree.groups"
            :key="String(group.id)"
            :node="group"
            :selected-mapping-id="selectedMapping ? String(selectedMapping.id) : ''"
            @select-mapping="openVariables"
            @open-fields="openVariables"
          />

          <button
            v-for="mapping in filteredTree.rootMappings"
            :key="String(mapping.id)"
            type="button"
            class="kafka-workbench__tree-item"
            :class="{ 'is-active': selectedMapping?.id === mapping.id }"
            @click="openVariables(mapping)"
            @dblclick="openVariables(mapping)"
            @keydown.enter="openVariables(mapping)"
          >
            <IconTablerMessages class="kafka-workbench__tree-item-icon" />
            <el-tooltip :content="mapping.topic" placement="top" :show-after="400">
              <span class="kafka-workbench__tree-item-name">{{
                mapping.name || mapping.topic
              }}</span>
            </el-tooltip>
          </button>

          <div
            v-if="filteredTree.groups.length === 0 && filteredTree.rootMappings.length === 0"
            class="kafka-workbench__empty"
          >
            {{ filterText ? '没有匹配的 Topic' : '暂无 Topic 映射' }}
          </div>
        </template>
      </div>
    </aside>

    <main class="kafka-workbench__main">
      <div class="kafka-workbench__tabbar">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          type="button"
          class="kafka-workbench__tab"
          :class="{ 'is-active': tab.id === activeTabId }"
          @click="activateTab(tab)"
        >
          <component :is="tab.icon" />
          <span>{{ tab.title }}</span>
          <IconTablerX class="kafka-workbench__tab-close" @click.stop="closeTab(tab.id)" />
        </button>
      </div>

      <div class="kafka-workbench__content">
        <KafkaPreviewPanel
          v-if="activeTab?.type === 'preview'"
          :project-id="projectId"
          :mapping="activeTab.mapping"
          @samples="handlePreviewSamples"
        />
        <KafkaFieldMappingPanel
          v-else-if="activeTab?.type === 'fields'"
          :project-id="projectId"
          :mapping="activeTab.mapping"
          :samples="getPreviewSamples(activeTab.mapping.id)"
          :connected="connected"
          @open-preview="openPreview"
          @samples="handlePreviewSamples"
        />
        <div v-else class="kafka-workbench__placeholder">
          <IconTablerMessages />
          <strong>选择 Topic 管理变量</strong>
          <span>从左侧 Topic 映射进入变量管理。</span>
        </div>
      </div>
    </main>

    <KafkaInspectorPanel
      :connection="inspectorConnection"
      :mapping="selectedMapping"
      :preview="activePreview"
    />

    <KafkaTopicGroupDialog
      v-model="groupDialogVisible"
      mode="create"
      :loading="groupSaving"
      @submit="saveGroup"
    />
    <KafkaTopicMappingDialog
      v-model="mappingDialogVisible"
      :mode="mappingDialogMode"
      :groups="groups"
      :mapping="editingMapping"
      :loading="mappingSaving"
      @submit="saveMapping"
    />
  </section>
</template>

<script setup lang="ts">
import { computed, markRaw, onMounted, ref, shallowRef, watch } from 'vue'
import { ElMessage } from 'element-plus'
import IconTablerFolderPlus from '~icons/tabler/folder-plus'
import IconTablerLoader2 from '~icons/tabler/loader-2'
import IconTablerMessages from '~icons/tabler/messages'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerSchema from '~icons/tabler/schema'
import IconTablerX from '~icons/tabler/x'
import dataAPI from '@/api/data.api'
import WorkbenchSourceHeader from '@/components/workbench/WorkbenchSourceHeader.vue'
import { getApiErrorMessage } from '@/utils/request'
import { buildKafkaTopicTree, filterKafkaTopicTree } from './kafkaTopicTreeModel'
import KafkaFieldMappingPanel from './KafkaFieldMappingPanel.vue'
import KafkaInspectorPanel from './KafkaInspectorPanel.vue'
import KafkaPreviewPanel from './KafkaPreviewPanel.vue'
import KafkaTopicGroupDialog from './KafkaTopicGroupDialog.vue'
import KafkaTopicMappingDialog from './KafkaTopicMappingDialog.vue'
import KafkaTopicTreeBranch from './KafkaTopicTreeBranch.vue'
import type {
  KafkaPreview,
  KafkaPreviewSample,
  KafkaTopicGroup,
  KafkaTopicMapping,
  KafkaWorkbenchConnection,
} from './types'

const props = defineProps<{
  projectId: string
  connection: KafkaWorkbenchConnection
}>()

defineEmits<{
  (event: 'back'): void
}>()

type KafkaWorkbenchTab = {
  id: string
  type: 'preview' | 'fields'
  title: string
  icon: any
  mapping: KafkaTopicMapping
}

const groups = ref<KafkaTopicGroup[]>([])
const mappings = ref<KafkaTopicMapping[]>([])
const loading = ref(false)
const filterText = ref('')
const connected = ref(false)
const connecting = ref(false)
const selectedMapping = ref<KafkaTopicMapping | null>(null)
const activeTabId = ref('')
const tabs = ref<KafkaWorkbenchTab[]>([])
const previewSamplesByMapping = shallowRef(new Map<string, KafkaPreviewSample[]>())
const activePreview = ref<KafkaPreview | null>(null)
const groupDialogVisible = ref(false)
const groupSaving = ref(false)
const mappingDialogVisible = ref(false)
const mappingDialogMode = ref<'create' | 'edit'>('create')
const mappingSaving = ref(false)
const editingMapping = ref<KafkaTopicMapping | null>(null)

const config = computed(() => props.connection.config || {})
const sourceMetaRows = computed(() => [
  { label: '类型', value: 'Kafka' },
  { label: 'Brokers', value: String(config.value.brokers || '未配置') },
])
const tree = computed(() => buildKafkaTopicTree(groups.value, mappings.value))
const filteredTree = computed(() =>
  filterKafkaTopicTree(tree.value.groups, tree.value.rootMappings, filterText.value),
)
const activeTab = computed(() => tabs.value.find((tab) => tab.id === activeTabId.value) || null)
const inspectorConnection = computed(() => ({
  ...props.connection,
  status: connected.value ? 'connected' : 'disconnected',
}))

const loadWorkbench = async () => {
  loading.value = true
  try {
    const [groupRes, mappingRes] = await Promise.all([
      dataAPI.getKafkaTopicGroups(props.projectId, props.connection.id),
      dataAPI.getKafkaTopicMappings(props.projectId, props.connection.id),
    ])
    groups.value = groupRes.list || []
    mappings.value = mappingRes.list || []
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载 Kafka 工作台失败'))
  } finally {
    loading.value = false
  }
}

const toggleConnection = async () => {
  if (connecting.value) return
  if (connected.value) {
    connected.value = false
    tabs.value = tabs.value.filter((tab) => tab.type !== 'preview')
    if (!tabs.value.some((tab) => tab.id === activeTabId.value)) {
      activeTabId.value = tabs.value[0]?.id || ''
    }
    ElMessage.success('Kafka 已断开')
    return
  }

  connecting.value = true
  try {
    await dataAPI.previewKafkaConnection(props.projectId, props.connection.id, {
      limit: 1,
      timeoutMs: 1000,
      probe: true,
    })
    connected.value = true
    ElMessage.success('Kafka 已连接')
  } catch (error) {
    connected.value = false
    ElMessage.error(getApiErrorMessage(error, 'Kafka 连接失败'))
  } finally {
    connecting.value = false
  }
}

const openCreateMapping = () => {
  mappingDialogMode.value = 'create'
  editingMapping.value = null
  mappingDialogVisible.value = true
}

const openCreateGroup = () => {
  groupDialogVisible.value = true
}

const saveGroup = async (payload: { name: string }) => {
  groupSaving.value = true
  try {
    await dataAPI.createKafkaTopicGroup(props.projectId, props.connection.id, payload)
    groupSaving.value = false
    groupDialogVisible.value = false
    ElMessage.success('Topic 分组已创建')
    await loadWorkbench()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '保存 Topic 分组失败'))
  } finally {
    groupSaving.value = false
  }
}

const saveMapping = async (payload: Record<string, unknown>) => {
  mappingSaving.value = true
  try {
    const saved =
      mappingDialogMode.value === 'edit' && editingMapping.value
        ? await dataAPI.updateKafkaTopicMapping(props.projectId, editingMapping.value.id, payload)
        : await dataAPI.createKafkaTopicMapping(props.projectId, props.connection.id, payload)
    mappingSaving.value = false
    mappingDialogVisible.value = false
    ElMessage.success('Topic 映射已保存')
    await loadWorkbench()
    if (saved?.id) {
      openVariables(saved)
    }
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '保存 Topic 映射失败'))
  } finally {
    mappingSaving.value = false
  }
}

const openPreview = (mapping: KafkaTopicMapping) => {
  selectedMapping.value = mapping
  const id = `kafka-preview-${mapping.id}`
  if (!tabs.value.some((tab) => tab.id === id)) {
    tabs.value.push({
      id,
      type: 'preview',
      title: `${mapping.name || mapping.topic} / 消息预览`,
      icon: markRaw(IconTablerMessages),
      mapping,
    })
  }
  activeTabId.value = id
}

const openVariables = (mapping: KafkaTopicMapping) => {
  selectedMapping.value = mapping
  const id = `kafka-fields-${mapping.id}`
  if (!tabs.value.some((tab) => tab.id === id)) {
    tabs.value.push({
      id,
      type: 'fields',
      title: `${mapping.name || mapping.topic} / 变量`,
      icon: markRaw(IconTablerSchema),
      mapping,
    })
  }
  activeTabId.value = id
}

const activateTab = (tab: KafkaWorkbenchTab) => {
  activeTabId.value = tab.id
  selectedMapping.value = tab.mapping
}

const closeTab = (tabId: string) => {
  const nextTabs = tabs.value.filter((tab) => tab.id !== tabId)
  tabs.value = nextTabs
  if (activeTabId.value === tabId) {
    activeTabId.value = nextTabs[0]?.id || ''
  }
}

const handlePreviewSamples = (payload: {
  mappingId: string
  samples: KafkaPreviewSample[]
  preview: KafkaPreview
}) => {
  const next = new Map(previewSamplesByMapping.value)
  next.set(payload.mappingId, payload.samples)
  previewSamplesByMapping.value = next
  activePreview.value = payload.preview
}

const getPreviewSamples = (mappingId: string | number) => {
  return previewSamplesByMapping.value.get(String(mappingId)) || []
}

onMounted(loadWorkbench)

watch(
  () => props.connection.id,
  async () => {
    tabs.value = []
    activeTabId.value = ''
    selectedMapping.value = null
    activePreview.value = null
    previewSamplesByMapping.value = new Map()
    await loadWorkbench()
  },
)

defineExpose({ handlePreviewSamples })
</script>

<style scoped>
.kafka-workbench {
  height: 100%;
  min-height: 0;
  display: grid;
  grid-template-columns: 260px minmax(0, 1fr) 280px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
  overflow: hidden;
}

.kafka-workbench__explorer {
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border-right: 1px solid var(--dc-border);
  background: var(--dc-surface-muted);
}

.kafka-workbench__connect-action {
  min-width: 58px;
  height: 26px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 0 9px;
  border: 1px solid color-mix(in oklch, var(--dc-primary) 32%, var(--dc-border));
  border-radius: var(--dc-radius-sm);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
  font-size: 11px;
  font-weight: 700;
  white-space: nowrap;
}

.kafka-workbench__connect-action::before {
  width: 6px;
  height: 6px;
  border-radius: 999px;
  background: currentColor;
  content: '';
}

.kafka-workbench__connect-action.is-connected {
  border-color: color-mix(in oklch, var(--dc-success) 32%, var(--dc-border));
  background: color-mix(in oklch, var(--dc-success) 12%, var(--dc-surface-raised));
  color: var(--dc-success);
}

.kafka-workbench__search {
  min-width: 0;
}

.kafka-workbench__tree {
  min-height: 0;
  flex: 1;
  display: grid;
  align-content: start;
  gap: 2px;
  overflow-y: auto;
  padding: 8px 8px 12px;
}

.kafka-workbench__tree-item {
  width: 100%;
  min-height: 30px;
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr);
  align-items: center;
  gap: 6px;
  padding: 3px 6px;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  font-size: 12px;
  text-align: left;
}

.kafka-workbench__tree-item:hover {
  border-color: var(--dc-border);
}

.kafka-workbench__tree-item:hover,
.kafka-workbench__tree-item.is-active {
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.kafka-workbench__tree-item-icon {
  width: 15px;
  height: 15px;
  color: var(--dc-primary);
}

.kafka-workbench__tree-item-name {
  min-width: 0;
  display: block;
  overflow: hidden;
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.kafka-workbench__main {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.kafka-workbench__tabbar {
  display: flex;
  min-height: 38px;
  overflow-x: auto;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
}

.kafka-workbench__tabbar:empty {
  display: none;
}

.kafka-workbench__tab {
  min-width: 130px;
  max-width: 260px;
  display: inline-flex;
  align-items: center;
  gap: 7px;
  padding: 0 10px;
  border: 0;
  border-right: 1px solid var(--dc-border);
  background: transparent;
  color: var(--dc-text-secondary);
  font-size: 12px;
}

.kafka-workbench__tab.is-active {
  background: var(--dc-surface-raised);
  color: var(--dc-primary);
  font-weight: 700;
}

.kafka-workbench__tab span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.kafka-workbench__tab svg {
  width: 15px;
  height: 15px;
  flex-shrink: 0;
}

.kafka-workbench__tab-close {
  margin-left: auto;
  color: var(--dc-text-muted);
}

.kafka-workbench__content {
  min-height: 0;
  flex: 1;
  overflow: hidden;
}

.kafka-workbench__placeholder {
  height: 100%;
  min-height: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  gap: 8px;
  padding: 24px;
  color: var(--dc-text-muted);
  text-align: center;
}

.kafka-workbench__placeholder svg {
  width: 34px;
  height: 34px;
  color: var(--dc-primary);
  opacity: 0.78;
}

.kafka-workbench__placeholder strong {
  color: var(--dc-text);
  font-size: 14px;
}

.kafka-workbench__placeholder span {
  max-width: 280px;
  font-size: 12px;
  line-height: 1.6;
}

.kafka-workbench__empty,
.kafka-workbench__loading {
  color: var(--dc-text-muted);
  font-size: 11px;
}

.kafka-workbench__loading {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px;
}

.kafka-workbench__loading svg {
  width: 14px;
  height: 14px;
  animation: kafka-workbench-spin 0.9s linear infinite;
}

@keyframes kafka-workbench-spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 1180px) {
  .kafka-workbench {
    grid-template-columns: 240px minmax(0, 1fr);
  }

  .kafka-workbench :deep(.kafka-inspector) {
    display: none;
  }
}

@media (max-width: 760px) {
  .kafka-workbench {
    grid-template-columns: 1fr;
  }

  .kafka-workbench__explorer {
    max-height: 320px;
    border-right: 0;
    border-bottom: 1px solid var(--dc-border);
  }
}
</style>
