<template>
  <section class="kafka-workbench">
    <aside class="kafka-workbench__explorer">
      <WorkbenchSourceHeader
        :title="connection.name || ui('未命名 Kafka 接入源', 'Unnamed Kafka Source')"
        :fallback-title="ui('未命名 Kafka 接入源', 'Unnamed Kafka Source')"
        :meta="sourceMetaRows"
        @back="$emit('back')"
      >
        <template #status>
          <button
            type="button"
            class="kafka-workbench__connect-action"
            :title="ui('测试 Broker 连通性', 'Test Broker Connectivity')"
            @click="testBroker"
          >
            <IconTablerLoader2 v-if="testingBroker" />
            <span>{{ testingBroker ? ui('测试中', 'Testing') : ui('测试 Broker', 'Test Broker') }}</span>
          </button>
        </template>
        <template #actions>
          <el-input
            v-model="filterText"
            class="kafka-workbench__search"
            size="small"
            clearable
            :placeholder="ui('筛选消费规则', 'Filter consumer rules')"
          />
          <button
            type="button"
            class="workbench-source-header__icon-action is-primary"
            :title="ui('新建消费规则', 'New Consumer Rule')"
            :aria-label="ui('新建消费规则', 'New consumer rule')"
            @click="openCreateMapping"
          >
            <IconTablerPlus />
          </button>
          <button
            type="button"
            class="workbench-source-header__icon-action"
            :title="ui('新建分组', 'New Group')"
            :aria-label="ui('新建分组', 'New group')"
            @click="openCreateGroup"
          >
            <IconTablerFolderPlus />
          </button>
          <button
            type="button"
            class="workbench-source-header__icon-action"
            :title="ui('刷新', 'Refresh')"
            :aria-label="ui('刷新', 'Refresh')"
            @click="loadWorkbench"
          >
            <IconTablerRefresh />
          </button>
        </template>
      </WorkbenchSourceHeader>

      <div class="kafka-workbench__tree">
        <div v-if="loading" class="kafka-workbench__loading">
          <IconTablerLoader2 />
          <span>{{ ui('加载消费规则...', 'Loading consumer rules...') }}</span>
        </div>
        <template v-else>
          <KafkaTopicTreeBranch
            v-for="group in filteredTree.groups"
            :key="String(group.id)"
            :node="group"
            :selected-mapping-id="selectedMapping ? String(selectedMapping.id) : ''"
            @select-mapping="selectMapping"
            @open-fields="selectMapping"
            @group-contextmenu="openGroupMenu"
            @mapping-contextmenu="openMappingMenu"
          />

          <button
            v-for="mapping in filteredTree.rootMappings"
            :key="String(mapping.id)"
            type="button"
            class="kafka-workbench__tree-item"
            :class="{ 'is-active': selectedMapping?.id === mapping.id }"
            @click="selectMapping(mapping)"
            @dblclick="selectMapping(mapping)"
            @contextmenu.prevent.stop="openMappingMenu($event, mapping)"
            @keydown.enter="selectMapping(mapping)"
          >
            <IconTablerMessages class="kafka-workbench__tree-item-icon" />
            <el-tooltip :content="mappingTooltip(mapping)" placement="top" :show-after="400">
              <span class="kafka-workbench__tree-item-name">{{
                mapping.name || mapping.topic
              }}</span>
            </el-tooltip>
          </button>

          <div
            v-if="filteredTree.groups.length === 0 && filteredTree.rootMappings.length === 0"
            class="kafka-workbench__empty"
          >
            {{ filterText ? ui('没有匹配的消费规则', 'No matching consumer rules') : ui('暂无消费规则', 'No consumer rules') }}
          </div>
        </template>
      </div>
    </aside>

    <main class="kafka-workbench__main">
      <WorkbenchTabBar
        v-if="modeTabs.length > 0"
        v-model:active-id="activeModeTabId"
        :tabs="
          modeTabs.map((tab) => ({
            id: tab.id,
            title: tab.title,
            icon: tab.type === 'raw' ? IconTablerMessages : IconTablerBraces,
          }))
        "
        @close="closeModeTab"
      />
      <div class="kafka-workbench__content">
        <template v-if="modeTabs.length > 0">
          <template v-for="tab in modeTabs" :key="tab.id">
            <KafkaRawOutputPanel
              v-if="tab.type === 'raw' && getMappingById(tab.mappingId)"
              class="kafka-workbench__content-panel"
              :class="{ 'is-active': activeModeTabId === tab.id }"
              :project-id="projectId"
              :connection-id="connection.id"
              :mapping="getMappingById(tab.mappingId)!"
              :pull-request-id="samplePullRequestId"
              @samples="handlePreviewSamples"
            />
            <KafkaFieldMappingPanel
              v-else-if="tab.type === 'fields' && getMappingById(tab.mappingId)"
              class="kafka-workbench__content-panel"
              :class="{ 'is-active': activeModeTabId === tab.id }"
              :project-id="projectId"
              :mapping="getMappingById(tab.mappingId)!"
              :samples="getPreviewSamples(tab.mappingId)"
              @samples="handlePreviewSamples"
            />
          </template>
        </template>
        <div v-else class="kafka-workbench__placeholder">
          <IconTablerMessages />
          <strong>{{ ui('选择消费规则', 'Select a Consumer Rule') }}</strong>
          <span>{{ ui('整包规则可测试拉取样本，字段规则可通过 JSON 样例生成字段映射。', 'Raw-message rules can pull test samples; field rules can generate mappings from JSON samples.') }}</span>
        </div>
      </div>
    </main>

    <KafkaTopicGroupDialog
      ref="groupDialogRef"
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
    <el-dialog
      v-model="detailDialogVisible"
      :title="ui('消费规则详情', 'Consumer Rule Details')"
      width="620px"
      class="kafka-workbench__detail-dialog"
    >
      <div v-if="detailMapping" class="kafka-workbench__detail">
        <section class="kafka-workbench__detail-section">
          <h3>{{ ui('基础信息', 'Basic Information') }}</h3>
          <dl class="kafka-workbench__detail-grid">
            <template v-for="row in basicDetailRows" :key="row.label">
              <dt>{{ row.label }}</dt>
              <dd>{{ row.value }}</dd>
            </template>
          </dl>
        </section>
        <section class="kafka-workbench__detail-section">
          <h3>{{ ui('消费配置', 'Consumer Configuration') }}</h3>
          <dl class="kafka-workbench__detail-grid">
            <template v-for="row in consumeDetailRows" :key="row.label">
              <dt>{{ row.label }}</dt>
              <dd>{{ row.value }}</dd>
            </template>
          </dl>
        </section>
        <section class="kafka-workbench__detail-section">
          <h3>{{ ui('输出配置', 'Output Configuration') }}</h3>
          <dl class="kafka-workbench__detail-grid">
            <template v-for="row in outputDetailRows" :key="row.label">
              <dt>{{ row.label }}</dt>
              <dd>{{ row.value }}</dd>
            </template>
          </dl>
        </section>
      </div>
      <template #footer>
        <el-button @click="detailDialogVisible = false">{{ ui('关闭', 'Close') }}</el-button>
      </template>
    </el-dialog>

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
          <button
            v-if="contextMenu.type === 'mapping'"
            type="button"
            @click="emitContextAction('details')"
          >
            <IconTablerInfoCircle class="kafka-workbench__menu-icon" />
            <span>{{ ui('查看详情', 'View Details') }}</span>
          </button>
          <button
            v-if="contextMenu.type === 'mapping'"
            type="button"
            @click="emitContextAction('edit')"
          >
            <IconTablerPencil class="kafka-workbench__menu-icon" />
            <span>{{ ui('编辑规则', 'Edit Rule') }}</span>
          </button>
          <button
            v-if="contextMenu.type === 'mapping'"
            type="button"
            @click="emitContextAction('copyTopic')"
          >
            <IconTablerCopy class="kafka-workbench__menu-icon" />
            <span>{{ ui('复制 Topic', 'Copy Topic') }}</span>
          </button>
          <button type="button" @click="emitContextAction('move')">
            <IconTablerFolderSymlink class="kafka-workbench__menu-icon" />
            <span>{{ ui('移动到分组', 'Move to Group') }}</span>
          </button>
          <button
            v-if="contextMenu.type === 'group'"
            type="button"
            @click="emitContextAction('createChildGroup')"
          >
            <IconTablerFolderPlus class="kafka-workbench__menu-icon" />
            <span>{{ ui('新建子分组', 'New Child Group') }}</span>
          </button>
          <button
            v-if="contextMenu.type === 'group'"
            type="button"
            @click="emitContextAction('renameGroup')"
          >
            <IconTablerPencil class="kafka-workbench__menu-icon" />
            <span>{{ ui('编辑分组', 'Edit Group') }}</span>
          </button>
          <button
            v-if="contextMenu.type === 'group'"
            type="button"
            @click="emitContextAction('move')"
          >
            <IconTablerFolderSymlink class="kafka-workbench__menu-icon" />
            <span>{{ ui('移动分组', 'Move Group') }}</span>
          </button>
          <button
            v-if="contextMenu.type === 'group'"
            type="button"
            class="is-danger"
            @click="emitContextAction('deleteGroup')"
          >
            <IconTablerTrash class="kafka-workbench__menu-icon" />
            <span>{{ ui('删除分组', 'Delete Group') }}</span>
          </button>
          <button
            v-if="contextMenu.type === 'mapping'"
            type="button"
            class="is-danger"
            @click="emitContextAction('delete')"
          >
            <IconTablerTrash class="kafka-workbench__menu-icon" />
            <span>{{ ui('删除', 'Delete') }}</span>
          </button>
        </div>
      </div>
    </Teleport>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, shallowRef, watch } from 'vue'
import dayjs from 'dayjs'
import { ElMessage, ElMessageBox } from 'element-plus'
import IconTablerBraces from '~icons/tabler/braces'
import IconTablerCopy from '~icons/tabler/copy'
import IconTablerFolderPlus from '~icons/tabler/folder-plus'
import IconTablerFolderSymlink from '~icons/tabler/folder-symlink'
import IconTablerInfoCircle from '~icons/tabler/info-circle'
import IconTablerLoader2 from '~icons/tabler/loader-2'
import IconTablerMessages from '~icons/tabler/messages'
import IconTablerPencil from '~icons/tabler/pencil'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerTrash from '~icons/tabler/trash'
import dataAPI from '@/api/data.api'
import WorkbenchSourceHeader from '@/components/workbench/WorkbenchSourceHeader.vue'
import WorkbenchTabBar from '@/components/workbench/WorkbenchTabBar.vue'
import { TIME_FORMAT } from '@/constants'
import { getApiErrorMessage } from '@/utils/request'
import { datacenterLocale } from '@/i18n/runtime'
import { buildKafkaTopicTree, filterKafkaTopicTree } from './kafkaTopicTreeModel'
import KafkaFieldMappingPanel from './KafkaFieldMappingPanel.vue'
import KafkaRawOutputPanel from './KafkaRawOutputPanel.vue'
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

type KafkaModeTab = {
  id: string
  type: 'raw' | 'fields'
  title: string
  mappingId: string
}

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)

const props = defineProps<{
  projectId: string
  connection: KafkaWorkbenchConnection
}>()

defineEmits<{
  (event: 'back'): void
}>()

const groups = ref<KafkaTopicGroup[]>([])
const mappings = ref<KafkaTopicMapping[]>([])
const loading = ref(false)
const filterText = ref('')
const testingBroker = ref(false)
const selectedMapping = ref<KafkaTopicMapping | null>(null)
const activeMapping = ref<KafkaTopicMapping | null>(null)
const modeTabs = ref<KafkaModeTab[]>([])
const activeModeTabId = ref('')
const samplePullRequestId = ref(0)
const previewSamplesByMapping = shallowRef(new Map<string, KafkaPreviewSample[]>())
const groupDialogVisible = ref(false)
const groupDialogRef = ref<InstanceType<typeof KafkaTopicGroupDialog> | null>(null)
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
const detailDialogVisible = ref(false)
const detailMapping = ref<KafkaTopicMapping | null>(null)
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
  { label: ui('类型', 'Type'), value: 'Kafka' },
  { label: 'Brokers', value: String(config.value.brokers || ui('未配置', 'Not Configured')) },
])
const tree = computed(() => buildKafkaTopicTree(groups.value, mappings.value))
const filteredTree = computed(() =>
  filterKafkaTopicTree(tree.value.groups, tree.value.rootMappings, filterText.value),
)
const basicDetailRows = computed(() => {
  const mapping = detailMapping.value
  if (!mapping) return []
  return [
    { label: ui('规则名称', 'Rule Name'), value: mapping.name || '-' },
    { label: ui('所属分组', 'Group'), value: getGroupName(mapping.groupId) },
    { label: 'Topic', value: mapping.topic || '-' },
    { label: ui('描述', 'Description'), value: mapping.description || '-' },
    { label: ui('创建时间', 'Created At'), value: formatTime(mapping.createdAt) },
    { label: ui('更新时间', 'Updated At'), value: formatTime(mapping.updatedAt) },
  ]
})
const consumeDetailRows = computed(() => {
  const mapping = detailMapping.value
  if (!mapping) return []
  return [
    { label: ui('消费组', 'Consumer Group'), value: mapping.consumerGroup || ui('默认生成', 'Generated by Default') },
    { label: ui('分区', 'Partition'), value: formatPartition(mapping) },
    { label: ui('起始位置', 'Start Position'), value: formatStartPosition(mapping) },
    { label: ui('解码方式', 'Decode'), value: formatDecode(mapping.decode) },
    { label: ui('样本上限', 'Sample Limit'), value: String(mapping.sampleLimit ?? 100) },
    { label: ui('拉取超时', 'Pull Timeout'), value: `${mapping.timeoutMs ?? 5000} ms` },
  ]
})
const outputDetailRows = computed(() => {
  const mapping = detailMapping.value
  if (!mapping) return []
  const rows = [
    { label: ui('输出模式', 'Output Mode'), value: formatOutputMode(mapping.outputMode) },
    { label: ui('整包范围', 'Raw Message Scope'), value: formatRawOutputScope(mapping) },
    { label: ui('数据点路径', 'Data Point Path'), value: mapping.rawDataPointPath || '-' },
  ]
  if (mapping.outputMode === 'field_mapping') {
    return rows.filter((row) => row.label !== ui('整包范围', 'Raw Message Scope'))
  }
  return rows
})

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
    ElMessage.error(getApiErrorMessage(error, ui('加载 Kafka 工作台失败', 'Failed to load Kafka workbench')))
  } finally {
    loading.value = false
  }
}

const testBroker = async () => {
  if (testingBroker.value) return
  testingBroker.value = true
  try {
    await dataAPI.previewKafkaConnection(props.projectId, props.connection.id, {
      limit: 1,
      timeoutMs: 1000,
      probe: true,
    })
    ElMessage.success(ui('Kafka Broker 可连接', 'Kafka Broker is reachable'))
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, ui('Kafka Broker 测试失败', 'Kafka Broker test failed')))
  } finally {
    testingBroker.value = false
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
      ElMessage.success(ui('规则分组已更新', 'Rule group updated'))
    } else {
      await dataAPI.createKafkaTopicGroup(props.projectId, props.connection.id, {
        name: payload.name,
        parentId: pendingParentGroupId.value || payload.parentId,
      })
      ElMessage.success(ui('规则分组已创建', 'Rule group created'))
    }
    groupDialogRef.value?.closeSilently()
    groupDialogVisible.value = false
    await loadWorkbench()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, ui('保存规则分组失败', 'Failed to save rule group')))
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
    ElMessage.success(ui('消费规则已保存', 'Consumer rule saved'))
    await loadWorkbench()
    if (saved?.id) {
      selectMapping(saved)
    }
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, ui('保存消费规则失败', 'Failed to save consumer rule')))
  } finally {
    mappingSaving.value = false
  }
}

const selectMapping = (mapping: KafkaTopicMapping) => {
  openModeTab(mapping)
}

const modeTabId = (mapping: KafkaTopicMapping) =>
  mapping.outputMode === 'raw_message' ? `kafka-raw-${mapping.id}` : `kafka-fields-${mapping.id}`

const modeTabType = (mapping: KafkaTopicMapping): KafkaModeTab['type'] =>
  mapping.outputMode === 'raw_message' ? 'raw' : 'fields'

const openModeTab = (mapping: KafkaTopicMapping) => {
  const id = modeTabId(mapping)
  const existing = modeTabs.value.find((tab) => tab.id === id)
  if (existing) {
    existing.title = mapping.name || mapping.topic
    existing.mappingId = String(mapping.id)
  } else {
    modeTabs.value.push({
      id,
      type: modeTabType(mapping),
      title: mapping.name || mapping.topic,
      mappingId: String(mapping.id),
    })
  }
  selectedMapping.value = mapping
  activeMapping.value = mapping
  activeModeTabId.value = id
}

const closeModeTab = (tabId: string) => {
  const index = modeTabs.value.findIndex((tab) => tab.id === tabId)
  if (index < 0) return
  modeTabs.value = modeTabs.value.filter((tab) => tab.id !== tabId)
  if (activeModeTabId.value === tabId) {
    activeModeTabId.value = modeTabs.value[index - 1]?.id || modeTabs.value[index]?.id || ''
  }
  if (!activeModeTabId.value) {
    selectedMapping.value = null
    activeMapping.value = null
  }
}

const getMappingById = (mappingId: string | number) =>
  mappings.value.find((mapping) => String(mapping.id) === String(mappingId)) ||
  (activeMapping.value && String(activeMapping.value.id) === String(mappingId)
    ? activeMapping.value
    : null)

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
    | 'details'
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
    if (action === 'details') {
      openMappingDetail(mapping)
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
      ElMessage.success(ui('Topic 已复制', 'Topic copied'))
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

const openMappingDetail = (mapping: KafkaTopicMapping) => {
  detailMapping.value = mapping
  detailDialogVisible.value = true
}

const buildMappingPayload = (mapping: KafkaTopicMapping) => ({
  name: mapping.name,
  topic: mapping.topic,
  groupId: mapping.groupId || null,
  consumerGroup: mapping.consumerGroup || '',
  outputMode: mapping.outputMode || 'field_mapping',
  rawOutputScope: mapping.rawOutputScope || 'value',
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
      ElMessage.success(ui('消费规则已移动', 'Consumer rule moved'))
    } else if (moveTargetType.value === 'group' && movingGroup.value) {
      await dataAPI.updateKafkaTopicGroup(props.projectId, movingGroup.value.id, {
        name: movingGroup.value.name,
        parentId: groupId,
      })
      ElMessage.success(ui('规则分组已移动', 'Rule group moved'))
    }
    moving.value = false
    moveDialogVisible.value = false
    await loadWorkbench()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, ui('移动失败', 'Move failed')))
  } finally {
    moving.value = false
  }
}

const deleteMapping = async (mapping: KafkaTopicMapping) => {
  const ok = await ElMessageBox.confirm(
    ui(`删除消费规则“${mapping.name || mapping.topic}”？相关数据点配置会一并标记为失效。`, `Delete consumer rule “${mapping.name || mapping.topic}”? Related data points will be marked invalid.`),
    ui('删除消费规则', 'Delete Consumer Rule'),
    {
      confirmButtonText: ui('删除', 'Delete'),
      cancelButtonText: ui('取消', 'Cancel'),
      type: 'warning',
    },
  )
    .then(() => true)
    .catch(() => false)
  if (!ok) return

  try {
    await dataAPI.deleteKafkaTopicMapping(props.projectId, mapping.id)
    if (selectedMapping.value?.id === mapping.id) {
      selectedMapping.value = null
    }
    if (activeMapping.value?.id === mapping.id) {
      activeMapping.value = null
    }
    modeTabs.value = modeTabs.value.filter((tab) => tab.mappingId !== String(mapping.id))
    if (!modeTabs.value.some((tab) => tab.id === activeModeTabId.value)) {
      activeModeTabId.value = modeTabs.value[0]?.id || ''
    }
    const nextSamples = new Map(previewSamplesByMapping.value)
    nextSamples.delete(String(mapping.id))
    previewSamplesByMapping.value = nextSamples
    ElMessage.success(ui('消费规则已删除', 'Consumer rule deleted'))
    await loadWorkbench()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, ui('删除消费规则失败', 'Failed to delete consumer rule')))
  }
}

const deleteGroup = async (group: KafkaTopicGroupNode) => {
  const ok = await ElMessageBox.confirm(
    ui(`确认删除分组「${group.name}」？组内消费规则会回到根目录。`, `Delete group “${group.name}”? Its consumer rules will move to the root.`),
    ui('删除分组', 'Delete Group'),
    {
      confirmButtonText: ui('删除', 'Delete'),
      cancelButtonText: ui('取消', 'Cancel'),
      type: 'warning',
    },
  )
    .then(() => true)
    .catch(() => false)
  if (!ok) return

  try {
    await dataAPI.deleteKafkaTopicGroup(props.projectId, group.id)
    ElMessage.success(ui('规则分组已删除', 'Rule group deleted'))
    await loadWorkbench()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, ui('删除规则分组失败', 'Failed to delete rule group')))
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
}

const getPreviewSamples = (mappingId: string | number) => {
  return previewSamplesByMapping.value.get(String(mappingId)) || []
}

const getGroupName = (groupId?: string | number | null) => {
  if (!groupId) return ui('根目录', 'Root')
  return groups.value.find((group) => String(group.id) === String(groupId))?.name || ui('未知分组', 'Unknown Group')
}

const formatTime = (value?: string | number | Date | null) => {
  if (!value) return '-'
  const date = dayjs(value)
  return date.isValid() ? date.format(TIME_FORMAT) : '-'
}

const formatOutputMode = (mode?: string) => {
  if (mode === 'raw_message') return ui('整包数据点', 'Raw Message Data Point')
  if (mode === 'field_mapping') return ui('字段数据点', 'Field Data Points')
  return '-'
}

const formatRawOutputScope = (mapping: KafkaTopicMapping) => {
  if (mapping.outputMode !== 'raw_message') return '-'
  return mapping.rawOutputScope === 'full_message' ? ui('完整 Kafka 消息', 'Complete Kafka Message') : ui('仅消息 Value', 'Message Value Only')
}

const formatPartition = (mapping: KafkaTopicMapping) => {
  if (mapping.partitionMode === 'single') return ui(`分区 ${mapping.partition ?? '-'}`, `Partition ${mapping.partition ?? '-'}`)
  return ui('全部分区', 'All Partitions')
}

const formatStartPosition = (mapping: KafkaTopicMapping) => {
  if (mapping.startPosition === 'earliest') return ui('最早消息', 'Earliest')
  if (mapping.startPosition === 'offset') return ui(`指定 Offset ${mapping.startOffset ?? '-'}`, `Offset ${mapping.startOffset ?? '-'}`)
  return ui('最新消息', 'Latest')
}

const formatDecode = (decode?: string) => {
  if (decode === 'json') return 'JSON'
  if (decode === 'string') return ui('字符串', 'String')
  if (decode === 'binary') return ui('二进制', 'Binary')
  return '-'
}

const mappingTooltip = (mapping: KafkaTopicMapping) => {
  const output = mapping.outputMode === 'raw_message' ? ui('整包数据点', 'Raw Message Data Point') : ui('字段数据点', 'Field Data Points')
  return ui(`Topic: ${mapping.topic}\n消费组: ${mapping.consumerGroup || '默认生成'}\n输出: ${output}`, `Topic: ${mapping.topic}\nConsumer Group: ${mapping.consumerGroup || 'Generated by Default'}\nOutput: ${output}`)
}

onMounted(loadWorkbench)

watch(
  () => props.connection.id,
  async () => {
    testingBroker.value = false
    selectedMapping.value = null
    activeMapping.value = null
    modeTabs.value = []
    activeModeTabId.value = ''
    previewSamplesByMapping.value = new Map()
    await loadWorkbench()
  },
)

watch(activeModeTabId, (tabId) => {
  const tab = modeTabs.value.find((item) => item.id === tabId)
  const mapping = tab ? getMappingById(tab.mappingId) : null
  selectedMapping.value = mapping
  activeMapping.value = mapping
})

defineExpose({ handlePreviewSamples })
</script>

<style scoped>
.kafka-workbench {
  height: 100%;
  min-height: 0;
  display: grid;
  grid-template-columns: 260px minmax(0, 1fr);
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

.kafka-workbench__content {
  position: relative;
  min-height: 0;
  flex: 1;
  overflow: hidden;
}

.kafka-workbench__content-panel {
  position: absolute;
  inset: 0;
  visibility: hidden;
  pointer-events: none;
  opacity: 0;
}

.kafka-workbench__content-panel.is-active {
  visibility: visible;
  pointer-events: auto;
  opacity: 1;
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

.kafka-workbench__detail {
  display: grid;
  gap: 14px;
}

.kafka-workbench__detail-section {
  display: grid;
  gap: 8px;
}

.kafka-workbench__detail-section h3 {
  margin: 0;
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 800;
}

.kafka-workbench__detail-grid {
  display: grid;
  grid-template-columns: 100px minmax(0, 1fr);
  gap: 8px 12px;
  margin: 0;
  padding: 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
}

.kafka-workbench__detail-grid dt {
  color: var(--dc-text-muted);
  font-size: 12px;
}

.kafka-workbench__detail-grid dd {
  min-width: 0;
  margin: 0;
  overflow-wrap: anywhere;
  color: var(--dc-text);
  font-size: 12px;
  font-weight: 700;
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
