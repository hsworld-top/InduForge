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
            @group-contextmenu="openGroupMenu"
            @mapping-contextmenu="openMappingMenu"
          />

          <button
            v-for="mapping in filteredTree.rootMappings"
            :key="String(mapping.id)"
            type="button"
            class="kafka-workbench__tree-item"
            :class="{ 'is-active': selectedMapping?.id === mapping.id }"
            @click="openVariables(mapping)"
            @dblclick="openVariables(mapping)"
            @contextmenu.prevent.stop="openMappingMenu($event, mapping)"
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
          :connected="connected"
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
      :mode="groupDialogMode"
      :groups="groups"
      :group="editingGroup"
      :initial-parent-id="pendingParentGroupId"
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
    <KafkaTopicMoveDialog
      v-model="moveDialogVisible"
      :target-type="moveTargetType"
      :groups="groups"
      :mapping="movingMapping"
      :group="movingGroup"
      :loading="moving"
      @submit="handleMoveSubmit"
    />

    <Teleport to="body">
      <div
        v-if="contextMenu.visible"
        class="kafka-workbench__menu-mask"
        @click="closeContextMenu"
        @contextmenu.prevent="closeContextMenu"
      >
        <div
          class="kafka-workbench__context-menu"
          :style="{ left: `${contextMenu.x}px`, top: `${contextMenu.y}px` }"
          @click.stop
        >
          <button v-if="contextMenu.type === 'mapping'" type="button" @click="emitContextAction('preview')">
            <IconTablerMessages class="kafka-workbench__menu-icon" />
            <span>消息预览</span>
          </button>
          <button v-if="contextMenu.type === 'mapping'" type="button" @click="emitContextAction('fields')">
            <IconTablerSchema class="kafka-workbench__menu-icon" />
            <span>变量管理</span>
          </button>
          <button v-if="contextMenu.type === 'mapping'" type="button" @click="emitContextAction('edit')">
            <IconTablerPencil class="kafka-workbench__menu-icon" />
            <span>编辑订阅</span>
          </button>
          <button v-if="contextMenu.type === 'mapping'" type="button" @click="emitContextAction('copyTopic')">
            <IconTablerCopy class="kafka-workbench__menu-icon" />
            <span>复制 Topic</span>
          </button>
          <button type="button" @click="emitContextAction('move')">
            <IconTablerFolderSymlink class="kafka-workbench__menu-icon" />
            <span>移动到分组</span>
          </button>
          <button v-if="contextMenu.type === 'group'" type="button" @click="emitContextAction('createChildGroup')">
            <IconTablerFolderPlus class="kafka-workbench__menu-icon" />
            <span>新建子分组</span>
          </button>
          <button v-if="contextMenu.type === 'group'" type="button" @click="emitContextAction('renameGroup')">
            <IconTablerPencil class="kafka-workbench__menu-icon" />
            <span>编辑分组</span>
          </button>
          <button v-if="contextMenu.type === 'group'" type="button" @click="emitContextAction('move')">
            <IconTablerFolderSymlink class="kafka-workbench__menu-icon" />
            <span>移动分组</span>
          </button>
          <button v-if="contextMenu.type === 'group'" type="button" class="is-danger" @click="emitContextAction('deleteGroup')">
            <IconTablerTrash class="kafka-workbench__menu-icon" />
            <span>删除分组</span>
          </button>
          <button v-if="contextMenu.type === 'mapping'" type="button" class="is-danger" @click="emitContextAction('delete')">
            <IconTablerTrash class="kafka-workbench__menu-icon" />
            <span>删除</span>
          </button>
        </div>
      </div>
    </Teleport>
  </section>
</template>

<script setup lang="ts">
import { computed, markRaw, onMounted, ref, shallowRef, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import IconTablerCopy from '~icons/tabler/copy'
import IconTablerFolderPlus from '~icons/tabler/folder-plus'
import IconTablerFolderSymlink from '~icons/tabler/folder-symlink'
import IconTablerLoader2 from '~icons/tabler/loader-2'
import IconTablerMessages from '~icons/tabler/messages'
import IconTablerPencil from '~icons/tabler/pencil'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerSchema from '~icons/tabler/schema'
import IconTablerTrash from '~icons/tabler/trash'
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
import KafkaTopicMoveDialog from './KafkaTopicMoveDialog.vue'
import KafkaTopicTreeBranch from './KafkaTopicTreeBranch.vue'
import type {
  KafkaPreview,
  KafkaPreviewSample,
  KafkaTopicGroup,
  KafkaTopicGroupNode,
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
const groupDialogMode = ref<'create' | 'edit'>('create')
const editingGroup = ref<KafkaTopicGroupNode | null>(null)
const pendingParentGroupId = ref<string | null>(null)
const groupSaving = ref(false)
const mappingDialogVisible = ref(false)
const mappingDialogMode = ref<'create' | 'edit'>('create')
const mappingSaving = ref(false)
const editingMapping = ref<KafkaTopicMapping | null>(null)
const moveDialogVisible = ref(false)
const moveTargetType = ref<'mapping' | 'group'>('mapping')
const movingMapping = ref<KafkaTopicMapping | null>(null)
const movingGroup = ref<KafkaTopicGroupNode | null>(null)
const moving = ref(false)
const contextMenu = ref<{
  visible: boolean
  type: 'mapping' | 'group' | null
  x: number
  y: number
  mapping: KafkaTopicMapping | null
  group: KafkaTopicGroupNode | null
}>({
  visible: false,
  type: null,
  x: 0,
  y: 0,
  mapping: null,
  group: null,
})

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
    activePreview.value = null
    previewSamplesByMapping.value = new Map()
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
    ElMessage.success('Kafka 已连接到 Broker')
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
  groupDialogMode.value = 'create'
  editingGroup.value = null
  pendingParentGroupId.value = null
  groupDialogVisible.value = true
}

const saveGroup = async (payload: { name: string; parentId: string | null }) => {
  groupSaving.value = true
  try {
    if (groupDialogMode.value === 'edit' && editingGroup.value) {
      await dataAPI.updateKafkaTopicGroup(props.projectId, editingGroup.value.id, {
        name: payload.name,
        parentId: payload.parentId,
      })
      ElMessage.success('Topic 分组已更新')
    } else {
      await dataAPI.createKafkaTopicGroup(props.projectId, props.connection.id, {
        name: payload.name,
        parentId: pendingParentGroupId.value || payload.parentId,
      })
      ElMessage.success('Topic 分组已创建')
    }
    groupDialogVisible.value = false
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
  if (!connected.value) {
    ElMessage.warning('请先点击左上角连接，再执行 Kafka 消息预览')
    return
  }
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

const openMappingMenu = (event: MouseEvent, mapping: KafkaTopicMapping) => {
  contextMenu.value = {
    visible: true,
    type: 'mapping',
    x: Math.min(event.clientX, window.innerWidth - 180),
    y: Math.min(event.clientY, window.innerHeight - 240),
    mapping,
    group: null,
  }
}

const openGroupMenu = (event: MouseEvent, group: KafkaTopicGroupNode) => {
  contextMenu.value = {
    visible: true,
    type: 'group',
    x: Math.min(event.clientX, window.innerWidth - 180),
    y: Math.min(event.clientY, window.innerHeight - 160),
    mapping: null,
    group,
  }
}

const closeContextMenu = () => {
  contextMenu.value.visible = false
}

const emitContextAction = async (
  action:
    | 'preview'
    | 'fields'
    | 'edit'
    | 'copyTopic'
    | 'move'
    | 'delete'
    | 'createChildGroup'
    | 'renameGroup'
    | 'deleteGroup',
) => {
  const { type, mapping, group } = contextMenu.value
  closeContextMenu()
  if (type === 'mapping' && mapping) {
    if (action === 'preview') {
      openPreview(mapping)
      return
    }
    if (action === 'fields') {
      openVariables(mapping)
      return
    }
    if (action === 'edit') {
      mappingDialogMode.value = 'edit'
      editingMapping.value = mapping
      mappingDialogVisible.value = true
      return
    }
    if (action === 'copyTopic') {
      await navigator.clipboard.writeText(mapping.topic)
      ElMessage.success('Topic 已复制')
      return
    }
    if (action === 'move') {
      openMoveMappingDialog(mapping)
      return
    }
    if (action === 'delete') {
      await deleteMapping(mapping)
      return
    }
  }
  if (type === 'group' && group) {
    if (action === 'createChildGroup') {
      openCreateChildGroup(group)
      return
    }
    if (action === 'renameGroup') {
      openRenameGroup(group)
      return
    }
    if (action === 'move') {
      openMoveGroupDialog(group)
      return
    }
    if (action === 'deleteGroup') {
      await deleteGroup(group)
    }
  }
}

const buildMappingPayload = (mapping: KafkaTopicMapping) => ({
  name: mapping.name,
  topic: mapping.topic,
  groupId: mapping.groupId || null,
  consumerGroup: mapping.consumerGroup || '',
  partitionMode: mapping.partitionMode,
  partition: mapping.partition ?? null,
  startPosition: mapping.startPosition,
  startOffset: mapping.startOffset ?? null,
  decode: mapping.decode,
  sampleLimit: mapping.sampleLimit,
  timeoutMs: mapping.timeoutMs,
  description: mapping.description || '',
  sortOrder: mapping.sortOrder || 0,
})

const openMoveMappingDialog = (mapping: KafkaTopicMapping) => {
  moveTargetType.value = 'mapping'
  movingMapping.value = mapping
  movingGroup.value = null
  moveDialogVisible.value = true
}

const openCreateChildGroup = (group: KafkaTopicGroupNode) => {
  groupDialogMode.value = 'create'
  editingGroup.value = null
  pendingParentGroupId.value = String(group.id)
  groupDialogVisible.value = true
}

const openRenameGroup = (group: KafkaTopicGroupNode) => {
  groupDialogMode.value = 'edit'
  editingGroup.value = group
  pendingParentGroupId.value = null
  groupDialogVisible.value = true
}

const openMoveGroupDialog = (group: KafkaTopicGroupNode) => {
  moveTargetType.value = 'group'
  movingMapping.value = null
  movingGroup.value = group
  moveDialogVisible.value = true
}

const handleMoveSubmit = async (groupId: string | null) => {
  moving.value = true
  try {
    if (moveTargetType.value === 'mapping' && movingMapping.value) {
      await dataAPI.updateKafkaTopicMapping(props.projectId, movingMapping.value.id, {
        ...buildMappingPayload(movingMapping.value),
        groupId,
      })
      ElMessage.success('Topic 订阅已移动')
    } else if (moveTargetType.value === 'group' && movingGroup.value) {
      await dataAPI.updateKafkaTopicGroup(props.projectId, movingGroup.value.id, {
        name: movingGroup.value.name,
        parentId: groupId,
      })
      ElMessage.success('Topic 分组已移动')
    }
    moveDialogVisible.value = false
    await loadWorkbench()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '移动失败'))
  } finally {
    moving.value = false
  }
}

const deleteMapping = async (mapping: KafkaTopicMapping) => {
  const ok = await ElMessageBox.confirm(
    `删除 Topic 订阅“${mapping.name || mapping.topic}”？相关变量配置会一并删除。`,
    '删除 Topic 订阅',
    {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
    },
  )
    .then(() => true)
    .catch(() => false)
  if (!ok) return

  try {
    await dataAPI.deleteKafkaTopicMapping(props.projectId, mapping.id)
    tabs.value = tabs.value.filter((tab) => tab.mapping.id !== mapping.id)
    if (!tabs.value.some((tab) => tab.id === activeTabId.value)) {
      activeTabId.value = tabs.value[0]?.id || ''
    }
    if (selectedMapping.value?.id === mapping.id) {
      selectedMapping.value = null
    }
    const nextSamples = new Map(previewSamplesByMapping.value)
    nextSamples.delete(String(mapping.id))
    previewSamplesByMapping.value = nextSamples
    ElMessage.success('Topic 订阅已删除')
    await loadWorkbench()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '删除 Topic 订阅失败'))
  }
}

const deleteGroup = async (group: KafkaTopicGroupNode) => {
  const ok = await ElMessageBox.confirm(
    `确认删除分组「${group.name}」？组内 Topic 订阅会回到根目录。`,
    '删除分组',
    {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
    },
  )
    .then(() => true)
    .catch(() => false)
  if (!ok) return

  try {
    await dataAPI.deleteKafkaTopicGroup(props.projectId, group.id)
    ElMessage.success('Topic 分组已删除')
    await loadWorkbench()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '删除 Topic 分组失败'))
  }
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
    connected.value = false
    connecting.value = false
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

.kafka-workbench__menu-mask {
  position: fixed;
  inset: 0;
  z-index: 2100;
}

.kafka-workbench__context-menu {
  position: fixed;
  min-width: 148px;
  padding: 4px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
}

.kafka-workbench__context-menu button {
  width: 100%;
  height: 30px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 8px;
  border: 0;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  font-size: 13px;
  text-align: left;
}

.kafka-workbench__context-menu button:hover {
  background: var(--dc-surface-muted);
  color: var(--dc-primary);
}

.kafka-workbench__context-menu button.is-danger:hover {
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
}

.kafka-workbench__menu-icon {
  width: 15px;
  height: 15px;
  flex: 0 0 auto;
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
