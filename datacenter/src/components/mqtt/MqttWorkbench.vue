<template>
  <section class="mqtt-workbench">
    <aside class="mqtt-workbench__explorer">
      <WorkbenchSourceHeader
        :title="connection.name || fallbackTitle"
        :fallback-title="fallbackTitle"
        :meta="sourceMetaRows"
        @back="$emit('back')"
      >
        <template #status>
          <button
            type="button"
            class="mqtt-workbench__connect-action"
            :class="{ 'is-connected': connectionStarted }"
            :title="connectButtonTitle"
            @click="toggleMqttPreview"
          >
            <span class="mqtt-workbench__connect-label is-default">{{ connectButtonLabel }}</span>
            <span v-if="connectionStarted" class="mqtt-workbench__connect-label is-hover">
              {{ ui('断开', 'Disconnect') }}
            </span>
          </button>
        </template>
        <template #actions>
          <el-input
            v-model="filterText"
            class="mqtt-workbench__search"
            size="small"
            :placeholder="filterPlaceholder"
            clearable
          />
          <button
            type="button"
            class="workbench-source-header__icon-action is-primary"
            :title="createSubscriptionLabel"
            :aria-label="createSubscriptionLabel"
            @click="openCreateSubscriptionDialog(null)"
          >
            <IconTablerPlus />
          </button>
          <button
            v-if="supportsSubscriptionGroups"
            type="button"
            class="workbench-source-header__icon-action"
            :title="ui('新建分组', 'New Group')"
            :aria-label="ui('新建分组', 'New Group')"
            @click="openCreateGroupDialog"
          >
            <IconTablerFolderPlus />
          </button>
          <button
            type="button"
            class="workbench-source-header__icon-action"
            :title="refreshLabel"
            :aria-label="refreshLabel"
            @click="loadWorkbenchTree"
          >
            <IconTablerRefresh />
          </button>
        </template>
      </WorkbenchSourceHeader>

      <div class="mqtt-workbench__tree">
        <div v-if="loading" class="mqtt-workbench__loading">
          <IconTablerLoader2 />
          <span>{{ loadingLabel }}</span>
        </div>
        <template v-else>
          <MqttSubscriptionTreeBranch
            v-for="group in filteredGroups"
            :key="group.id"
            :node="group"
            :selected-subscription-id="selectedSubscription?.id"
            @select-subscription="selectSubscription"
            @group-contextmenu="openGroupMenu"
            @subscription-contextmenu="openSubscriptionMenu"
          />

          <button
            v-for="subscription in filteredRootSubscriptions"
            :key="String(subscription.id)"
            type="button"
            class="mqtt-workbench__tree-item"
            :class="{
              'is-active': selectedSubscription?.id === subscription.id,
            }"
            @click="selectSubscription(subscription)"
            @dblclick="selectSubscription(subscription)"
            @keydown.enter="selectSubscription(subscription)"
            @contextmenu.prevent.stop="openSubscriptionMenu($event, subscription)"
          >
            <component
              :is="resolveSubscriptionIcon(subscription)"
              class="mqtt-workbench__tree-item-icon"
            />
            <el-tooltip
              :content="subscription.name || subscription.topic || subscription.id"
              placement="top"
              :show-after="400"
            >
              <span class="mqtt-workbench__tree-item-name">
                {{ subscription.name || subscription.topic }}
              </span>
            </el-tooltip>
          </button>

          <div
            v-if="filteredGroups.length === 0 && filteredRootSubscriptions.length === 0"
            class="mqtt-workbench__empty"
          >
            {{ filterText ? ui(`没有匹配的${treeItemNoun}`, `No matching ${treeItemNoun.toLowerCase()}`) : ui(`暂无${treeItemNoun}`, `No ${treeItemNoun.toLowerCase()}`) }}
          </div>
        </template>
      </div>
    </aside>

    <main class="mqtt-workbench__main">
      <div class="mqtt-workbench__tabbar">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          type="button"
          class="mqtt-workbench__tab"
          :class="{ 'is-active': activeTabId === tab.id }"
          @click="activateTab(tab.id)"
        >
          <component :is="tab.icon" />
          <span>{{ tab.title }}</span>
          <IconTablerX class="mqtt-workbench__tab-close" @click.stop="closeTab(tab.id)" />
        </button>
      </div>

      <div :key="workbenchContentKey" class="mqtt-workbench__content">
        <template v-if="activeContentTabs.length > 0">
          <template v-for="tab in activeContentTabs" :key="tab.id">
            <MqttMessageViewer
              v-if="tab.type === 'messages'"
              class="mqtt-workbench__content-panel"
              :class="{ 'is-active': activeTabId === tab.id }"
              :ref="(el) => setMessageViewerRef(tab.id, el)"
              :subscription="tab.subscription"
              :project-id="projectIdText"
              :connection-id="connection.id"
              :source="workbenchMode"
              @subscribe="subscribeMessageTab"
              @unsubscribe="unsubscribeMessageTab"
            />

            <div
              v-else-if="tab.type === 'tags'"
              class="mqtt-workbench__tag-panel"
              :class="{
                'is-active': activeTabId === tab.id,
                'is-config-only': !connectionStarted,
              }"
            >
              <div class="mqtt-workbench__tag-list">
                <MqttTagList
                  :ref="(el) => setTagListRef(tab.id, el)"
                  :project-id="projectIdText"
                  :subscription-id="tab.subscription.id"
                  :preview-session-id="connectionStarted ? previewSessionId : ''"
                  @open-monitor="openTagMonitor(tab.subscription)"
                />
              </div>
            </div>

            <div
              v-else-if="tab.type === 'batch'"
              class="mqtt-workbench__tag-panel"
              :class="{
                'is-active': activeTabId === tab.id,
                'is-config-only': !connectionStarted,
              }"
            >
              <MqttBatchMappingPanel
                :ref="(el) => setTagListRef(tab.id, el)"
                :project-id="projectIdText"
                :subscription="tab.subscription"
                :preview-session-id="connectionStarted ? previewSessionId : ''"
                @open-monitor="openTagMonitor(tab.subscription)"
                @subscription-updated="refreshSelectedAndTabs"
              />
            </div>

            <MqttPublishTester
              v-else-if="tab.type === 'publish'"
              class="mqtt-workbench__content-panel"
              :class="{ 'is-active': activeTabId === tab.id }"
              :project-id="projectIdText"
              :connection-id="connection.id"
              :subscription="tab.subscription"
              :source="workbenchMode"
              :connected="connectionStarted"
            />
          </template>
        </template>
        <div v-else class="mqtt-workbench__placeholder">
          <IconTablerRss />
          <strong>{{ placeholderTitle }}</strong>
          <span>{{ placeholderHint }}</span>
        </div>
      </div>
    </main>

    <MqttSubscriptionDialog
      v-model="subscriptionDialogVisible"
      :project-id="projectIdText"
      :connection-id="connection.id"
      :mode="subscriptionDialogMode"
      :subscription="editingSubscription"
      :group-id="subscriptionDialogGroupId"
      :entity-type="isBuiltinMessageMode ? 'topic' : 'subscription'"
      @success="handleSubscriptionSaved"
    />

    <MqttSubscriptionGroupDialog
      ref="groupDialogRef"
      v-model="groupDialogVisible"
      :mode="groupDialogMode"
      :groups="subscriptionGroups"
      :group="contextGroup"
      :initial-parent-id="groupDialogMode === 'create' ? contextGroup?.id || null : null"
      :loading="groupSaving"
      @submit="handleGroupDialogSubmit"
    />

    <MqttSubscriptionMoveDialog
      v-model="moveDialogVisible"
      :target-type="moveTargetType"
      :subscription="contextSubscription"
      :group="contextGroup"
      :groups="subscriptionGroups"
      :loading="moveSaving"
      @submit="handleMoveSubmit"
    />

    <DcDialog
      v-model="monitorDialogVisible"
      width="960px"
      body-max-height="calc(100vh - 180px)"
      class="mqtt-workbench__monitor-dialog"
      @close="closeMonitorDialog"
    >
      <template #header>
        <div class="mqtt-workbench__monitor-header">
          <span class="dc-dialog__title">{{ ui('变量预览/监控', 'Variable Preview / Monitor') }}</span>
          <div class="mqtt-workbench__monitor-actions">
            <el-tooltip :content="ui('列表显示', 'List View')" placement="top">
              <button
                type="button"
                class="mqtt-workbench__monitor-action"
                :class="{ 'is-active': monitorViewMode === 'list' }"
                @click="setMonitorViewMode('list')"
              >
                <IconTablerList />
              </button>
            </el-tooltip>
            <el-tooltip :content="ui('卡片显示', 'Card View')" placement="top">
              <button
                type="button"
                class="mqtt-workbench__monitor-action"
                :class="{ 'is-active': monitorViewMode === 'card' }"
                @click="setMonitorViewMode('card')"
              >
                <IconTablerLayoutGrid />
              </button>
            </el-tooltip>
            <el-tooltip :content="ui('重新加载变量配置', 'Reload Variable Configuration')" placement="top">
              <button
                type="button"
                class="mqtt-workbench__monitor-action"
                @click="refreshMonitorTags"
              >
                <IconTablerRefresh />
              </button>
            </el-tooltip>
          </div>
        </div>
      </template>
      <MqttTagMonitor
        ref="monitorRef"
        v-if="monitorDialogVisible && monitorSubscription"
        :project-id="projectIdText"
        :subscription-id="monitorSubscription.id"
        :preview-session-id="previewSessionId"
        :snapshot="monitorSnapshot"
        @tag-value="handleMonitorTagValue"
        @latest-values="applyMonitorTagValues"
      />
    </DcDialog>

    <DcDialog
      v-model="detailDialogVisible"
      :title="ui('订阅详情', 'Subscription Details')"
      width="680px"
      class="mqtt-workbench__detail-dialog"
      @close="detailSubscription = null"
    >
      <template v-if="detailSubscription">
        <div class="mqtt-workbench__detail-heading">
          <IconTablerRss />
          <div>
            <strong>{{ detailSubscription.name || detailSubscription.topic }}</strong>
            <span>{{ detailSubscription.topic || '-' }}</span>
          </div>
        </div>

        <section class="mqtt-workbench__detail-section">
          <div class="mqtt-workbench__panel-title">{{ ui('订阅信息', 'Subscription Information') }}</div>
          <dl class="mqtt-workbench__facts">
            <div>
              <dt>{{ ui('名称', 'Name') }}</dt>
              <dd>{{ detailSubscription.name || '-' }}</dd>
            </div>
            <div>
              <dt>Topic</dt>
              <dd>{{ detailSubscription.topic || '-' }}</dd>
            </div>
            <div>
              <dt>{{ ui('路径', 'Path') }}</dt>
              <dd>{{ subscriptionPath(detailSubscription) }}</dd>
            </div>
            <div>
              <dt>QoS</dt>
              <dd>{{ detailSubscription.qos ?? 0 }}</dd>
            </div>
            <div>
              <dt>{{ ui('消息保留数', 'Message Retention') }}</dt>
              <dd>{{ detailSubscription.messageRetention ?? 5000 }}</dd>
            </div>
            <div>
              <dt>{{ ui('描述', 'Description') }}</dt>
              <dd>{{ detailSubscription.description || '-' }}</dd>
            </div>
            <div>
              <dt>{{ isBuiltinMessageMode ? 'Topic ID' : ui('订阅 ID', 'Subscription ID') }}</dt>
              <dd>{{ detailSubscription.id }}</dd>
            </div>
          </dl>
        </section>

        <section class="mqtt-workbench__detail-section">
          <div class="mqtt-workbench__panel-title">
            {{ isBuiltinMessageMode ? ui('当前消息库', 'Current Message Store') : ui('当前 Broker', 'Current Broker') }}
          </div>
          <dl class="mqtt-workbench__facts">
            <div>
              <dt>{{ isBuiltinMessageMode ? ui('标识', 'Identifier') : ui('地址', 'Address') }}</dt>
              <dd>{{ endpointText }}</dd>
            </div>
            <div>
              <dt>{{ ui('状态', 'Status') }}</dt>
              <dd>{{ connectionStarted ? ui('已连接', 'Connected') : ui('未连接', 'Disconnected') }}</dd>
            </div>
            <div>
              <dt>{{ treeItemNoun }}</dt>
              <dd>{{ subscriptions.length }}</dd>
            </div>
          </dl>
        </section>
      </template>
    </DcDialog>

    <Teleport to="body">
      <div
        v-if="contextMenu.visible"
        class="mqtt-workbench__menu-mask"
        @click="closeContextMenu"
        @contextmenu.prevent="closeContextMenu"
      >
        <div
          class="mqtt-workbench__context-menu"
          :style="{ left: `${contextMenu.x}px`, top: `${contextMenu.y}px` }"
          @click.stop
        >
          <button
            v-if="contextMenu.type === 'subscription'"
            type="button"
            @click="emitContextAction('detail')"
          >
            <IconTablerInfoCircle class="mqtt-workbench__menu-icon" />
            <span>{{ ui('详情', 'Details') }}</span>
          </button>
          <button
            v-if="contextMenu.type === 'subscription'"
            type="button"
            @click="emitContextAction(contextPrimaryAction)"
          >
            <component :is="contextPrimaryIcon" class="mqtt-workbench__menu-icon" />
            <span>{{ contextPrimaryLabel }}</span>
          </button>
          <button
            v-if="
              contextMenu.type === 'subscription' &&
              contextMenu.subscription &&
              resolveSubscriptionUsageMode(contextMenu.subscription) !== 'raw_datapoint'
            "
            type="button"
            @click="emitContextAction('messages')"
          >
            <IconTablerMessages class="mqtt-workbench__menu-icon" />
            <span>{{ ui('查看消息', 'View Messages') }}</span>
          </button>
          <button
            v-if="contextMenu.type === 'subscription'"
            type="button"
            @click="emitContextAction('rename')"
          >
            <IconTablerPencil class="mqtt-workbench__menu-icon" />
            <span>{{ ui('编辑订阅', 'Edit Subscription') }}</span>
          </button>
          <button
            v-if="contextMenu.type === 'subscription'"
            type="button"
            @click="emitContextAction('publish')"
          >
            <IconTablerSend class="mqtt-workbench__menu-icon" />
            <span>{{ ui('发布测试', 'Publish Test') }}</span>
          </button>
          <button
            v-if="contextMenu.type === 'subscription' && supportsSubscriptionGroups"
            type="button"
            @click="emitContextAction('move')"
          >
            <IconTablerFolderSymlink class="mqtt-workbench__menu-icon" />
            <span>{{ ui('移动到分组', 'Move to Group') }}</span>
          </button>
          <button
            v-if="contextMenu.type === 'group' && supportsSubscriptionGroups"
            type="button"
            @click="emitContextAction('createChildGroup')"
          >
            <IconTablerFolderPlus class="mqtt-workbench__menu-icon" />
            <span>{{ ui('新建子分组', 'New Child Group') }}</span>
          </button>
          <button
            v-if="contextMenu.type === 'group' && supportsSubscriptionGroups"
            type="button"
            @click="emitContextAction('rename')"
          >
            <IconTablerPencil class="mqtt-workbench__menu-icon" />
            <span>{{ ui('编辑分组', 'Edit Group') }}</span>
          </button>
          <button
            v-if="contextMenu.type === 'group' && supportsSubscriptionGroups"
            type="button"
            @click="emitContextAction('move')"
          >
            <IconTablerFolderSymlink class="mqtt-workbench__menu-icon" />
            <span>{{ ui('移动分组', 'Move Group') }}</span>
          </button>
          <button
            v-if="contextMenu.type === 'group' && supportsSubscriptionGroups"
            type="button"
            class="is-danger"
            @click="emitContextAction('delete')"
          >
            <IconTablerTrash class="mqtt-workbench__menu-icon" />
            <span>{{ ui('删除分组', 'Delete Group') }}</span>
          </button>
          <button
            v-if="contextMenu.type === 'subscription'"
            type="button"
            class="is-danger"
            @click="emitContextAction('delete')"
          >
            <IconTablerTrash class="mqtt-workbench__menu-icon" />
            <span>{{ ui('删除', 'Delete') }}</span>
          </button>
        </div>
      </div>
    </Teleport>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import IconTablerFolderPlus from '~icons/tabler/folder-plus'
import IconTablerFolderSymlink from '~icons/tabler/folder-symlink'
import IconTablerInfoCircle from '~icons/tabler/info-circle'
import IconTablerDatabase from '~icons/tabler/database'
import IconTablerLayoutGrid from '~icons/tabler/layout-grid'
import IconTablerList from '~icons/tabler/list'
import IconTablerListTree from '~icons/tabler/list-tree'
import IconTablerLoader2 from '~icons/tabler/loader-2'
import IconTablerMessages from '~icons/tabler/messages'
import IconTablerPencil from '~icons/tabler/pencil'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerRss from '~icons/tabler/rss'
import IconTablerSend from '~icons/tabler/send'
import IconTablerTags from '~icons/tabler/tags'
import IconTablerTrash from '~icons/tabler/trash'
import IconTablerX from '~icons/tabler/x'
import dataAPI from '@/api/data.api'
import WorkbenchSourceHeader from '@/components/workbench/WorkbenchSourceHeader.vue'
import { usePreviewSession } from '@/composables/usePreviewSession'
import { useMqttSocket } from '@/composables/useMqttSocket'
import { getApiErrorMessage } from '@/utils/request'
import { datacenterLocale } from '@/i18n/runtime'
import DcDialog from '@/components/shared/DcDialog.vue'
import MqttMessageViewer from './MqttMessageViewer.vue'
import MqttPublishTester from './MqttPublishTester.vue'
import MqttBatchMappingPanel from './MqttBatchMappingPanel.vue'
import MqttSubscriptionDialog from './MqttSubscriptionDialog.vue'
import MqttSubscriptionGroupDialog from './MqttSubscriptionGroupDialog.vue'
import MqttSubscriptionMoveDialog from './MqttSubscriptionMoveDialog.vue'
import MqttSubscriptionTreeBranch from './MqttSubscriptionTreeBranch.vue'
import MqttTagList from './MqttTagList.vue'
import MqttTagMonitor from './MqttTagMonitor.vue'
import {
  buildMqttSubscriptionTree,
  filterMqttSubscriptionTree,
  type MqttSubscription,
  type MqttSubscriptionGroup,
  type MqttSubscriptionGroupNode,
} from './mqttSubscriptionTreeModel'

type MqttConnection = {
  id: string
  name?: string
  mqttConfig?: Record<string, any>
  config?: Record<string, any>
  type?: string
}

type WorkbenchMode = 'mqtt' | 'builtin-message'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)

const props = defineProps<{
  projectId: string | number
  connection: MqttConnection
  mode?: WorkbenchMode
}>()

defineEmits<{
  (event: 'back'): void
}>()

const projectIdRef = computed(() => String(props.projectId || ''))
const projectIdText = computed(() => projectIdRef.value)
const workbenchMode = computed<WorkbenchMode>(() =>
  props.mode === 'builtin-message' || props.connection.type === 'builtin.message'
    ? 'builtin-message'
    : 'mqtt',
)
const isBuiltinMessageMode = computed(() => workbenchMode.value === 'builtin-message')
const config = computed(() => props.connection.mqttConfig || props.connection.config || {})
const endpointText = computed(() => {
  if (isBuiltinMessageMode.value) {
    return String(config.value.runtimeKey || ui('未生成', 'Not generated'))
  }
  const host = config.value.brokerUrl || config.value.host || ui('未配置 Broker', 'Broker not configured')
  const port = config.value.port ? `:${config.value.port}` : ''
  return `${host}${port}`
})
const fallbackTitle = computed(() =>
  isBuiltinMessageMode.value ? ui('IF消息库', 'IF Message Store') : ui('未命名 MQTT 接入源', 'Unnamed MQTT Source'),
)
const treeItemNoun = computed(() => (isBuiltinMessageMode.value ? 'Topic' : ui('订阅', 'Subscription')))
const filterPlaceholder = computed(() =>
  isBuiltinMessageMode.value ? ui('筛选 Topic', 'Filter topics') : ui('筛选订阅或 Topic', 'Filter subscriptions or topics'),
)
const createSubscriptionLabel = computed(() =>
  isBuiltinMessageMode.value ? ui('新建 Topic', 'New Topic') : ui('新建订阅', 'New Subscription'),
)
const refreshLabel = computed(() => (isBuiltinMessageMode.value ? ui('刷新 Topic', 'Refresh Topics') : ui('刷新订阅', 'Refresh Subscriptions')))
const loadingLabel = computed(() => (isBuiltinMessageMode.value ? ui('加载 Topic...', 'Loading topics...') : ui('加载订阅...', 'Loading subscriptions...')))
const placeholderTitle = computed(() =>
  isBuiltinMessageMode.value ? ui('选择 Topic 开始工作', 'Select a Topic to Begin') : ui('选择订阅开始工作', 'Select a Subscription to Begin'),
)
const placeholderHint = computed(() =>
  isBuiltinMessageMode.value
    ? ui('左键打开对应 Topic，右键查看详情、消息或发布测试。', 'Click a topic to open it; right-click for details, messages, or publish testing.')
    : ui('从左侧订阅树打开订阅；连接后可查看实时消息。', 'Open a subscription from the tree; connect to view live messages.'),
)
const supportsSubscriptionGroups = computed(() => true)
const sourceMetaRows = computed(() =>
  isBuiltinMessageMode.value
    ? [
        { label: ui('类型', 'Type'), value: ui('IF消息库', 'IF Message Store') },
        { label: ui('标识', 'Identifier'), value: endpointText.value },
      ]
    : [
        { label: ui('类型', 'Type'), value: 'MQTT Broker' },
        { label: ui('地址', 'Address'), value: endpointText.value },
      ],
)

const { sessionId: previewSessionId, ensureSession } = usePreviewSession(projectIdRef, {
  autoStart: false,
})
const { connected: socketConnected, subscribeMessages } = useMqttSocket(
  projectIdRef,
  previewSessionId,
)

const loading = ref(false)
const subscriptions = ref<MqttSubscription[]>([])
const subscriptionGroups = ref<MqttSubscriptionGroup[]>([])
const selectedSubscription = ref<MqttSubscription | null>(null)
const filterText = ref('')
const connectionStarted = ref(false)
const connectButtonLabel = computed(() => (connectionStarted.value ? ui('已连接', 'Connected') : ui('连接', 'Connect')))
const connectButtonTitle = computed(() => (connectionStarted.value ? ui('断开', 'Disconnect') : ui('连接', 'Connect')))
const tabs = ref<any[]>([])
const activeTabId = ref('')
const messageViewerRefs = shallowRef(new Map<string, any>())
const tagListRefs = shallowRef(new Map<string, any>())
const messageCleanups = new Map<string, () => void>()
const subscriptionDialogVisible = ref(false)
const subscriptionDialogMode = ref<'create' | 'edit'>('create')
const subscriptionDialogGroupId = ref<string | null>(null)
const editingSubscription = ref<MqttSubscription | null>(null)
const groupDialogVisible = ref(false)
const groupDialogMode = ref<'create' | 'rename'>('create')
const groupDialogRef = ref<InstanceType<typeof MqttSubscriptionGroupDialog> | null>(null)
const groupSaving = ref(false)
const moveDialogVisible = ref(false)
const moveTargetType = ref<'subscription' | 'group'>('subscription')
const moveSaving = ref(false)
const monitorDialogVisible = ref(false)
const monitorSubscription = ref<MqttSubscription | null>(null)
const monitorRef = ref<any>(null)
const monitorSnapshot = ref<any>(null)
const monitorViewMode = ref<'list' | 'card'>('list')
const detailDialogVisible = ref(false)
const detailSubscription = ref<MqttSubscription | null>(null)
const contextSubscription = ref<MqttSubscription | null>(null)
const contextGroup = ref<MqttSubscriptionGroupNode | null>(null)
const contextMenu = ref<{
  visible: boolean
  type: 'subscription' | 'group' | null
  x: number
  y: number
  subscription: MqttSubscription | null
  group: MqttSubscriptionGroupNode | null
}>({
  visible: false,
  type: null,
  x: 0,
  y: 0,
  subscription: null,
  group: null,
})
const contextSubscriptionMode = computed(() =>
  contextMenu.value.subscription
    ? resolveSubscriptionUsageMode(contextMenu.value.subscription)
    : 'single_variable',
)
const contextPrimaryAction = computed<'messages' | 'variables'>(() =>
  contextSubscriptionMode.value === 'raw_datapoint' ? 'messages' : 'variables',
)
const contextPrimaryLabel = computed(() => {
  if (contextSubscriptionMode.value === 'raw_datapoint') return ui('查看消息', 'View Messages')
  if (contextSubscriptionMode.value === 'batch_variable') return ui('批量映射配置', 'Batch Mapping Configuration')
  return ui('单变量配置', 'Single Variable Configuration')
})
const contextPrimaryIcon = computed(() => {
  if (contextSubscriptionMode.value === 'raw_datapoint') return IconTablerMessages
  if (contextSubscriptionMode.value === 'batch_variable') return IconTablerListTree
  return IconTablerTags
})

const hasActiveTab = computed(() =>
  Boolean(activeTabId.value && tabs.value.some((tab) => tab.id === activeTabId.value)),
)
const workbenchContentKey = computed(() => props.connection.id || 'empty')
const activeContentTabs = computed(() => (hasActiveTab.value ? tabs.value : []))

const subscriptionTree = computed(() =>
  buildMqttSubscriptionTree(subscriptionGroups.value, subscriptions.value),
)
const filteredTree = computed(() =>
  filterMqttSubscriptionTree(
    subscriptionTree.value.rootGroups,
    subscriptionTree.value.rootSubscriptions,
    filterText.value,
  ),
)
const filteredGroups = computed(() => filteredTree.value.groups)
const filteredRootSubscriptions = computed(() => filteredTree.value.subscriptions)

const groupNameMap = computed(() => {
  const map = new Map<string, { name: string; parentId: string | null }>()
  const visit = (groups: MqttSubscriptionGroup[], parentId: string | null) => {
    groups.forEach((group) => {
      const id = String(group.id)
      const groupParentId =
        group.parentId === null || group.parentId === undefined || group.parentId === ''
          ? parentId
          : String(group.parentId)
      map.set(id, { name: group.name, parentId: groupParentId })
      visit(group.children || [], id)
    })
  }
  visit(subscriptionGroups.value, null)
  return map
})

const loadWorkbenchTree = async () => {
  loading.value = true
  try {
    const [subscriptionsResponse, groupsResponse] = await Promise.all([
      dataAPI.getMqttSubscriptions(projectIdText.value, props.connection.id),
      dataAPI.getMqttSubscriptionGroups(projectIdText.value, props.connection.id),
    ])
    subscriptions.value = subscriptionsResponse.data?.list || []
    subscriptionGroups.value = groupsResponse.data?.list || []
    if (
      subscriptions.value.length > 0 &&
      !subscriptions.value.some(
        (subscription) => subscription.id === selectedSubscription.value?.id,
      )
    ) {
      selectedSubscription.value = subscriptions.value[0]
    }
  } catch (error) {
    ElMessage.error(
      getApiErrorMessage(
        error,
        isBuiltinMessageMode.value ? ui('加载 IF消息库 Topic 失败', 'Failed to load IF Message Store topics') : ui('加载 MQTT 订阅树失败', 'Failed to load MQTT subscription tree'),
      ),
    )
  } finally {
    loading.value = false
  }
}

const ensureMqttPreview = async () => {
  const session = await ensureSession()
  if (!session) {
    ElMessage.warning(ui('预览会话不可用', 'Preview session unavailable'))
    return false
  }
  if (connectionStarted.value) return true

  try {
    await dataAPI.startMqttConnection(projectIdText.value, props.connection.id)
    connectionStarted.value = true
    ElMessage.success(isBuiltinMessageMode.value ? ui('IF消息库已连接', 'IF Message Store connected') : ui('MQTT 已连接', 'MQTT connected'))
    return true
  } catch (error) {
    ElMessage.error(
      getApiErrorMessage(error, isBuiltinMessageMode.value ? ui('IF消息库连接失败', 'Failed to connect IF Message Store') : ui('MQTT 连接失败', 'Failed to connect MQTT')),
    )
    return false
  }
}

const stopMqttPreview = async () => {
  if (!connectionStarted.value) return true

  try {
    cleanupMessageSubscriptions()
    monitorDialogVisible.value = false
    monitorSubscription.value = null
    await dataAPI.stopMqttConnection(projectIdText.value, props.connection.id)
    connectionStarted.value = false
    ElMessage.success(isBuiltinMessageMode.value ? ui('IF消息库已断开', 'IF Message Store disconnected') : ui('MQTT 已断开', 'MQTT disconnected'))
    return true
  } catch (error) {
    ElMessage.error(
      getApiErrorMessage(error, isBuiltinMessageMode.value ? ui('IF消息库断开失败', 'Failed to disconnect IF Message Store') : ui('MQTT 断开失败', 'Failed to disconnect MQTT')),
    )
    return false
  }
}

const toggleMqttPreview = async () => {
  if (connectionStarted.value) {
    await stopMqttPreview()
    return
  }
  await ensureMqttPreview()
}

const addTab = (tab: any) => {
  const existing = tabs.value.find((item) => item.id === tab.id)
  if (existing) {
    activateTab(existing.id)
    return existing
  }
  tabs.value.push(tab)
  activateTab(tab.id, { refresh: false })
  return tab
}

const activateTab = (tabId: string, options: { refresh?: boolean } = {}) => {
  activeTabId.value = tabId
  if (options.refresh === false) return

  nextTick(() => {
    const tab = tabs.value.find((item) => item.id === tabId)
    if (tab?.type === 'tags' || tab?.type === 'batch') {
      tagListRefs.value.get(tabId)?.refreshQuietly?.()
    }
  })
}

const resolveSubscriptionUsageMode = (subscription: MqttSubscription) =>
  subscription.usageMode || 'single_variable'

const resolveSubscriptionIcon = (subscription: MqttSubscription) => {
  if (resolveSubscriptionUsageMode(subscription) === 'raw_datapoint') return IconTablerMessages
  if (resolveSubscriptionUsageMode(subscription) === 'batch_variable') return IconTablerListTree
  return IconTablerDatabase
}

const selectSubscription = (subscription: MqttSubscription) => {
  if (resolveSubscriptionUsageMode(subscription) === 'raw_datapoint') {
    void openMessages(subscription)
    return
  }
  void openTagManager(subscription)
}

const openMessages = async (subscription: MqttSubscription) => {
  selectedSubscription.value = subscription

  addTab({
    id: `mqtt-messages-${subscription.id}`,
    type: 'messages',
    title: `${subscription.name || subscription.topic} / ${ui('消息', 'Messages')}`,
    icon: IconTablerMessages,
    subscription,
  })

  await nextTick()
}

const subscribeMessageTab = async (subscription: MqttSubscription) => {
  if (!connectionStarted.value) {
    ElMessage.warning(ui('请先连接后再订阅消息', 'Connect before subscribing to messages'))
    return
  }
  const tabId = `mqtt-messages-${subscription.id}`
  const tab = tabs.value.find((item) => item.id === tabId)
  if (!tab) return

  if (!messageCleanups.has(tab.id)) {
    const cleanup = subscribeMessages(subscription.id, (data) => {
      const viewer = messageViewerRefs.value.get(tab.id)
      viewer?.addMessage?.(data?.message || data)
      viewer?.setSubscribed?.(true)
    })
    messageCleanups.set(tab.id, cleanup)
  }

  await nextTick()
  messageViewerRefs.value.get(tab.id)?.setSubscribed?.(true)
}

const unsubscribeMessageTab = (subscription: MqttSubscription) => {
  const tabId = `mqtt-messages-${subscription.id}`
  messageCleanups.get(tabId)?.()
  messageCleanups.delete(tabId)
  messageViewerRefs.value.get(tabId)?.setSubscribed?.(false)
}

const openTagManager = async (subscription: MqttSubscription) => {
  selectedSubscription.value = subscription

  if (resolveSubscriptionUsageMode(subscription) === 'batch_variable') {
    addTab({
      id: `mqtt-batch-${subscription.id}`,
      type: 'batch',
      title: `${subscription.name || subscription.topic} / ${ui('映射', 'Mapping')}`,
      icon: IconTablerListTree,
      subscription,
    })
    return
  }

  addTab({
    id: `mqtt-tags-${subscription.id}`,
    type: 'tags',
    title: `${subscription.name || subscription.topic} / ${ui('变量', 'Variables')}`,
    icon: IconTablerTags,
    subscription,
  })
}

const openTagMonitor = (subscription: MqttSubscription) => {
  selectedSubscription.value = subscription
  if (!connectionStarted.value) {
    ElMessage.warning(ui('请先连接后再打开变量监控', 'Connect before opening variable monitoring'))
    return
  }
  const activeRef = tagListRefs.value.get(activeTabId.value)
  monitorSnapshot.value = activeRef?.getMonitorSnapshot?.() || null
  monitorSubscription.value = subscription
  monitorDialogVisible.value = true
  void nextTick(() => {
    monitorRef.value?.setViewMode?.(monitorViewMode.value)
  })
}

const setMonitorViewMode = (mode: 'list' | 'card') => {
  monitorViewMode.value = mode
  monitorRef.value?.setViewMode?.(mode)
}

const refreshMonitorTags = () => {
  void monitorRef.value?.refresh?.()
}

const openPublishTester = async (subscription: MqttSubscription) => {
  selectedSubscription.value = subscription

  addTab({
    id: `mqtt-publish-${subscription.id}`,
    type: 'publish',
    title: `${subscription.name || subscription.topic} / ${ui('发布', 'Publish')}`,
    icon: IconTablerSend,
    subscription,
  })

  await nextTick()
}

const closeMonitorDialog = () => {
  const values = monitorRef.value?.getLatestValues?.() || []
  applyMonitorTagValues(values)
  monitorRef.value = null
  monitorSnapshot.value = null
  monitorSubscription.value = null
}

const refreshSelectedAndTabs = (subscription: MqttSubscription) => {
  if (selectedSubscription.value?.id === subscription.id) {
    selectedSubscription.value = subscription
  }
  tabs.value = tabs.value.map((tab) =>
    tab.subscription?.id === subscription.id
      ? {
          ...tab,
          title: `${subscription.name || subscription.topic} / ${tabTypeLabel(tab.type)}`,
          subscription,
        }
      : tab,
  )
}

const tabTypeLabel = (type: string) => {
  if (type === 'messages') return ui('消息', 'Messages')
  if (type === 'publish') return ui('发布', 'Publish')
  if (type === 'batch') return ui('映射', 'Mapping')
  return ui('变量', 'Variables')
}

const subscriptionUpdatePayload = (
  subscription: MqttSubscription,
  patch: Partial<MqttSubscription> & { hasGroupId?: boolean },
) => ({
  name: patch.name ?? subscription.name ?? '',
  topic: patch.topic ?? subscription.topic ?? '',
  qos: patch.qos ?? subscription.qos ?? 0,
  description:
    patch.description !== undefined ? patch.description || null : subscription.description || null,
  messageRetention: patch.messageRetention ?? subscription.messageRetention ?? 5000,
  order: patch.order ?? subscription.order ?? 0,
  groupId: patch.groupId !== undefined ? patch.groupId || null : subscription.groupId || null,
  hasGroupId: patch.hasGroupId ?? true,
})

function openCreateSubscriptionDialog(groupId: string | null) {
  editingSubscription.value = null
  subscriptionDialogMode.value = 'create'
  subscriptionDialogGroupId.value = groupId
  subscriptionDialogVisible.value = true
}

function openRenameSubscriptionDialog(subscription: MqttSubscription) {
  editingSubscription.value = subscription
  subscriptionDialogMode.value = 'edit'
  subscriptionDialogGroupId.value = subscription.groupId || null
  subscriptionDialogVisible.value = true
}

function openSubscriptionDetail(subscription: MqttSubscription) {
  selectedSubscription.value = subscription
  detailSubscription.value = subscription
  detailDialogVisible.value = true
}

function subscriptionPath(subscription: MqttSubscription) {
  const names = [subscription.name || subscription.topic || subscription.id]
  let groupId = subscription.groupId ? String(subscription.groupId) : ''
  const visited = new Set<string>()

  while (groupId && !visited.has(groupId)) {
    visited.add(groupId)
    const group = groupNameMap.value.get(groupId)
    if (!group) break
    names.unshift(group.name)
    groupId = group.parentId || ''
  }

  return [ui('根目录', 'Root'), ...names].join(' / ')
}

async function handleSubscriptionSaved(data?: any) {
  const saved = data?.subscription || data
  if (saved?.id) refreshSelectedAndTabs(saved)
  await loadWorkbenchTree()
}

function openCreateGroupDialog(parentGroup?: MqttSubscriptionGroupNode | null) {
  if (!supportsSubscriptionGroups.value) {
    return
  }
  contextGroup.value = parentGroup || null
  groupDialogMode.value = 'create'
  groupDialogVisible.value = true
}

function openRenameGroupDialog(group: MqttSubscriptionGroupNode) {
  contextGroup.value = group
  groupDialogMode.value = 'rename'
  groupDialogVisible.value = true
}

async function handleGroupDialogSubmit(data: { name: string; parentId: string | null }) {
  groupSaving.value = true
  try {
    if (groupDialogMode.value === 'create') {
      await dataAPI.createMqttSubscriptionGroup(projectIdText.value, props.connection.id, data)
      ElMessage.success(ui('分组已创建', 'Group created'))
    } else if (contextGroup.value) {
      await dataAPI.updateMqttSubscriptionGroup(projectIdText.value, contextGroup.value.id, {
        name: data.name,
        parentId: data.parentId,
        hasParentId: true,
      })
      ElMessage.success(ui('分组已重命名', 'Group renamed'))
    }
    groupSaving.value = false
    groupDialogVisible.value = false
    groupDialogRef.value?.closeSilently()
    await loadWorkbenchTree()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, ui('保存分组失败', 'Failed to save group')))
  } finally {
    groupSaving.value = false
  }
}

function openMoveSubscriptionDialog(subscription: MqttSubscription) {
  contextSubscription.value = subscription
  contextGroup.value = null
  moveTargetType.value = 'subscription'
  moveDialogVisible.value = true
}

function openMoveGroupDialog(group: MqttSubscriptionGroupNode) {
  contextSubscription.value = null
  contextGroup.value = group
  moveTargetType.value = 'group'
  moveDialogVisible.value = true
}

async function handleMoveSubmit(groupId: string | null) {
  if (!supportsSubscriptionGroups.value) {
    moveDialogVisible.value = false
    return
  }
  moveSaving.value = true
  try {
    if (moveTargetType.value === 'subscription' && contextSubscription.value) {
      const response = await dataAPI.updateMqttSubscription(
        projectIdText.value,
        contextSubscription.value.id,
        subscriptionUpdatePayload(contextSubscription.value, {
          groupId,
          hasGroupId: true,
        }),
      )
      const saved = response.data?.subscription || response.data
      if (saved?.id) refreshSelectedAndTabs(saved)
      ElMessage.success(ui('订阅已移动', 'Subscription moved'))
    } else if (moveTargetType.value === 'group' && contextGroup.value) {
      await dataAPI.updateMqttSubscriptionGroup(projectIdText.value, contextGroup.value.id, {
        parentId: groupId,
        hasParentId: true,
      })
      ElMessage.success(ui('分组已移动', 'Group moved'))
    }
    moveSaving.value = false
    moveDialogVisible.value = false
    await loadWorkbenchTree()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, ui('移动失败', 'Move failed')))
  } finally {
    moveSaving.value = false
  }
}

async function deleteSubscription(subscription: MqttSubscription) {
  const ok = await ElMessageBox.confirm(
    ui(`确认删除订阅「${subscription.name || subscription.topic || subscription.id}」？此操作不可恢复。`, `Delete subscription “${subscription.name || subscription.topic || subscription.id}”? This cannot be undone.`),
    ui('删除订阅', 'Delete Subscription'),
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
    await dataAPI.deleteMqttSubscription(projectIdText.value, subscription.id)
    tabs.value = tabs.value.filter((tab) => tab.subscription?.id !== subscription.id)
    if (!tabs.value.some((tab) => tab.id === activeTabId.value)) {
      activeTabId.value = tabs.value[0]?.id || ''
    }
    if (selectedSubscription.value?.id === subscription.id) {
      selectedSubscription.value = null
    }
    ElMessage.success(ui('订阅已删除', 'Subscription deleted'))
    await loadWorkbenchTree()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, ui('删除订阅失败', 'Failed to delete subscription')))
  }
}

const countGroupSubscriptions = (group: MqttSubscriptionGroupNode): number =>
  group.subscriptions.length +
  group.children.reduce((sum, child) => sum + countGroupSubscriptions(child), 0)

async function deleteGroup(group: MqttSubscriptionGroupNode) {
  const subscriptionCount = countGroupSubscriptions(group)
  const ok = await ElMessageBox.confirm(
    ui(`确认删除分组「${group.name}」？组内 ${subscriptionCount} 个订阅会回到根目录。`, `Delete group “${group.name}”? Its ${subscriptionCount} subscriptions will return to Root.`),
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
    await dataAPI.deleteMqttSubscriptionGroup(projectIdText.value, group.id)
    ElMessage.success(ui('分组已删除', 'Group deleted'))
    await loadWorkbenchTree()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, ui('删除分组失败', 'Failed to delete group')))
  }
}

function openSubscriptionMenu(event: MouseEvent, subscription: MqttSubscription) {
  contextMenu.value = {
    visible: true,
    type: 'subscription',
    x: Math.min(event.clientX, window.innerWidth - 180),
    y: Math.min(event.clientY, window.innerHeight - 216),
    subscription,
    group: null,
  }
}

function openGroupMenu(event: MouseEvent, group: MqttSubscriptionGroupNode) {
  contextMenu.value = {
    visible: true,
    type: 'group',
    x: Math.min(event.clientX, window.innerWidth - 180),
    y: Math.min(event.clientY, window.innerHeight - 126),
    subscription: null,
    group,
  }
}

function closeContextMenu() {
  contextMenu.value.visible = false
}

function emitContextAction(
  action:
    | 'rename'
    | 'move'
    | 'delete'
    | 'messages'
    | 'variables'
    | 'detail'
    | 'publish'
    | 'createChildGroup',
) {
  const { type, subscription, group } = contextMenu.value
  closeContextMenu()
  if (type === 'subscription' && subscription) {
    if (action === 'rename') {
      openRenameSubscriptionDialog(subscription)
      return
    }
    if (action === 'move') {
      openMoveSubscriptionDialog(subscription)
      return
    }
    if (action === 'messages') {
      void openMessages(subscription)
      return
    }
    if (action === 'variables') {
      void openTagManager(subscription)
      return
    }
    if (action === 'detail') {
      openSubscriptionDetail(subscription)
      return
    }
    if (action === 'publish') {
      void openPublishTester(subscription)
      return
    }
    void deleteSubscription(subscription)
    return
  }
  if (type === 'group' && group) {
    if (!supportsSubscriptionGroups.value) {
      return
    }
    if (action === 'createChildGroup') {
      openCreateGroupDialog(group)
      return
    }
    if (action === 'rename') {
      openRenameGroupDialog(group)
      return
    }
    if (action === 'move') {
      openMoveGroupDialog(group)
      return
    }
    void deleteGroup(group)
  }
}

const closeTab = (tabId: string) => {
  messageCleanups.get(tabId)?.()
  messageCleanups.delete(tabId)
  messageViewerRefs.value.delete(tabId)
  tagListRefs.value.delete(tabId)

  const index = tabs.value.findIndex((tab) => tab.id === tabId)
  if (index < 0) return
  const nextTabs = tabs.value.filter((tab) => tab.id !== tabId)
  const nextActiveTabId =
    activeTabId.value === tabId || !nextTabs.some((tab) => tab.id === activeTabId.value)
      ? nextTabs[index - 1]?.id || nextTabs[index]?.id || ''
      : activeTabId.value
  tabs.value = nextTabs
  activeTabId.value = nextActiveTabId
}

const setMessageViewerRef = (tabId: string, el: any) => {
  if (el) {
    messageViewerRefs.value.set(tabId, el)
    el?.setSubscribed?.(messageCleanups.has(tabId))
    return
  }
  messageViewerRefs.value.delete(tabId)
}

const setTagListRef = (tabId: string, el: any) => {
  if (el) {
    tagListRefs.value.set(tabId, el)
    return
  }
  tagListRefs.value.delete(tabId)
}

const handleMonitorTagValue = (value: any) => {
  applyMonitorTagValues([value])
}

const applyMonitorTagValues = (values: any[]) => {
  if (!Array.isArray(values) || values.length === 0) {
    return
  }
  const subscriptionId =
    values.find((value) => value?.subscriptionId)?.subscriptionId || monitorSubscription.value?.id
  if (!subscriptionId) {
    return
  }
  const singleRef = tagListRefs.value.get(`mqtt-tags-${subscriptionId}`)
  const batchRef = tagListRefs.value.get(`mqtt-batch-${subscriptionId}`)
  values.forEach((value) => singleRef?.applyTagValueUpdate?.(value))
  batchRef?.applyTagValueUpdates?.(values)
}

const cleanupMessageSubscriptions = () => {
  Array.from(messageCleanups.values()).forEach((cleanup) => cleanup?.())
  messageCleanups.clear()
  messageViewerRefs.value.forEach((viewer) => {
    viewer?.setSubscribed?.(false)
  })
}

onMounted(async () => {
  await loadWorkbenchTree()
})

watch(
  () => props.connection.id,
  async () => {
    cleanupMessageSubscriptions()
    messageViewerRefs.value.clear()
    tagListRefs.value.clear()
    monitorDialogVisible.value = false
    monitorSubscription.value = null
    detailDialogVisible.value = false
    detailSubscription.value = null
    tabs.value = []
    activeTabId.value = ''
    selectedSubscription.value = null
    connectionStarted.value = false
    await loadWorkbenchTree()
  },
)

watch(
  socketConnected,
  (connected) => {
    if (!connected) {
      messageViewerRefs.value.forEach((viewer) => {
        viewer?.setSubscribed?.(false)
      })
    }
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  cleanupMessageSubscriptions()
  messageViewerRefs.value.clear()
  tagListRefs.value.clear()
})
</script>

<style scoped>
.mqtt-workbench {
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

.mqtt-workbench__explorer {
  min-height: 0;
  overflow: hidden;
  background: var(--dc-surface-muted);
}

.mqtt-workbench__explorer {
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--dc-border);
}

.mqtt-workbench__connect-action {
  min-width: 58px;
  height: 26px;
  position: relative;
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

.mqtt-workbench__connect-label.is-hover {
  display: none;
}

.mqtt-workbench__connect-action.is-connected:hover .mqtt-workbench__connect-label.is-default {
  display: none;
}

.mqtt-workbench__connect-action.is-connected:hover .mqtt-workbench__connect-label.is-hover {
  display: inline;
}

.mqtt-workbench__search {
  min-width: 0;
}

.mqtt-workbench__connect-action::before {
  width: 6px;
  height: 6px;
  border-radius: 999px;
  background: currentColor;
  content: '';
}

.mqtt-workbench__connect-action:hover {
  border-color: color-mix(in oklch, var(--dc-primary) 48%, var(--dc-border));
}

.mqtt-workbench__connect-action.is-connected {
  border-color: color-mix(in oklch, var(--dc-success) 32%, var(--dc-border));
  background: color-mix(in oklch, var(--dc-success) 12%, var(--dc-surface-raised));
  color: var(--dc-success);
}

.mqtt-workbench__connect-action.is-connected:hover {
  border-color: rgba(220, 38, 38, 0.22);
  background: rgba(220, 38, 38, 0.08);
  color: #b91c1c;
}

.mqtt-workbench__tree {
  min-height: 0;
  flex: 1;
  display: grid;
  align-content: start;
  gap: 2px;
  overflow-y: auto;
  padding: 8px 8px 12px;
}

.mqtt-workbench__tree-item {
  width: 100%;
  display: grid;
  align-items: center;
  border: 0;
  background: transparent;
  color: var(--dc-text-secondary);
  text-align: left;
}

.mqtt-workbench__tree-item-icon {
  width: 15px;
  height: 15px;
  color: var(--dc-primary);
}

.mqtt-workbench__tree-item {
  grid-template-columns: 18px minmax(0, 1fr) auto;
  gap: 6px;
  min-height: 30px;
  padding: 3px 6px;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  font-size: 12px;
}

.mqtt-workbench__tree-item:hover {
  border-color: var(--dc-border);
}

.mqtt-workbench__tree-item-name {
  display: block;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mqtt-workbench__tree-item-name {
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 700;
}

.mqtt-workbench__tree-item:hover,
.mqtt-workbench__tree-item.is-active {
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.mqtt-workbench__main {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.mqtt-workbench__tabbar {
  display: flex;
  min-height: 38px;
  overflow-x: auto;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
}

.mqtt-workbench__tabbar:empty {
  display: none;
}

.mqtt-workbench__tab {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  min-width: 130px;
  max-width: 240px;
  padding: 0 10px;
  border: 0;
  border-right: 1px solid var(--dc-border);
  background: transparent;
  color: var(--dc-text-secondary);
  font-size: 12px;
}

.mqtt-workbench__tab.is-active {
  background: var(--dc-surface-raised);
  color: var(--dc-primary);
  font-weight: 700;
}

.mqtt-workbench__tab span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mqtt-workbench__tab svg {
  width: 15px;
  height: 15px;
  flex-shrink: 0;
}

.mqtt-workbench__tab-close {
  margin-left: auto;
  color: var(--dc-text-muted);
}

.mqtt-workbench__content {
  position: relative;
  min-height: 0;
  flex: 1;
  overflow: hidden;
}

.mqtt-workbench__content-panel,
.mqtt-workbench__tag-panel {
  position: absolute;
  inset: 0;
  visibility: hidden;
  pointer-events: none;
  opacity: 0;
}

.mqtt-workbench__content-panel.is-active,
.mqtt-workbench__tag-panel.is-active {
  visibility: visible;
  pointer-events: auto;
  opacity: 1;
}

.mqtt-workbench__placeholder {
  min-height: 0;
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  gap: 8px;
  padding: 24px;
  color: var(--dc-text-muted);
  text-align: center;
}

.mqtt-workbench__placeholder svg {
  width: 34px;
  height: 34px;
  color: var(--dc-primary);
  opacity: 0.78;
}

.mqtt-workbench__placeholder strong {
  color: var(--dc-text);
  font-size: 14px;
}

.mqtt-workbench__placeholder span {
  max-width: 280px;
  font-size: 12px;
  line-height: 1.6;
}

.mqtt-workbench__tag-panel {
  height: 100%;
  min-height: 0;
  display: block;
}

.mqtt-workbench__tag-panel.is-config-only {
  display: block;
}

.mqtt-workbench__tag-list {
  height: 100%;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
}

.mqtt-workbench__dialog-form {
  display: grid;
  gap: 12px;
}

.mqtt-workbench__dialog-form label {
  display: grid;
  gap: 6px;
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.mqtt-workbench__panel-title {
  margin-bottom: 8px;
  color: var(--dc-text);
  font-size: 12px;
  font-weight: 700;
}

.mqtt-workbench__facts {
  margin: 0;
}

.mqtt-workbench__facts div {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 0;
  border-bottom: 1px solid var(--dc-border);
}

.mqtt-workbench__facts dt {
  color: var(--dc-text-muted);
  font-size: 11px;
}

.mqtt-workbench__facts dd {
  min-width: 0;
  margin: 0;
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
  text-align: right;
  word-break: break-all;
}

.mqtt-workbench__detail-heading {
  display: grid;
  grid-template-columns: 34px minmax(0, 1fr);
  align-items: center;
  gap: 10px;
  margin-bottom: 14px;
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-subtle);
}

.mqtt-workbench__detail-heading svg {
  width: 18px;
  height: 18px;
  justify-self: center;
  color: var(--dc-primary);
}

.mqtt-workbench__detail-heading div {
  min-width: 0;
  display: grid;
  gap: 3px;
}

.mqtt-workbench__detail-heading strong,
.mqtt-workbench__detail-heading span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mqtt-workbench__detail-heading strong {
  color: var(--dc-text);
  font-size: 14px;
}

.mqtt-workbench__detail-heading span {
  color: var(--dc-text-muted);
  font-family: var(--dc-font-mono, monospace);
  font-size: 12px;
}

.mqtt-workbench__detail-section {
  margin-top: 14px;
}

:global(.mqtt-workbench__monitor-dialog .el-dialog__body) {
  height: calc(100vh - 180px);
  max-height: calc(100vh - 180px);
  display: flex;
  min-height: 0;
  flex-direction: column;
  overflow: hidden;
}

:deep(.mqtt-workbench__monitor-dialog .el-dialog__header) {
  margin-right: 0;
}

.mqtt-workbench__monitor-header {
  min-height: 48px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding-right: 34px;
}

.mqtt-workbench__monitor-actions {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.mqtt-workbench__monitor-action {
  width: 30px;
  height: 30px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}

.mqtt-workbench__monitor-action:hover,
.mqtt-workbench__monitor-action.is-active {
  border-color: var(--dc-primary);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.mqtt-workbench__monitor-action svg {
  width: 16px;
  height: 16px;
}

:global(.mqtt-workbench__monitor-dialog .dc-dialog__body) {
  height: 100%;
  display: flex;
  min-height: 0;
  flex-direction: column;
  flex: 1;
  overflow: hidden;
}

:global(.mqtt-workbench__monitor-dialog .mqtt-tag-monitor) {
  flex: 1;
  height: 100%;
  min-height: 0;
}

.mqtt-workbench__empty,
.mqtt-workbench__loading {
  color: var(--dc-text-muted);
  font-size: 11px;
}

.mqtt-workbench__loading {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px;
}

.mqtt-workbench__loading svg {
  width: 14px;
  height: 14px;
  animation: mqtt-workbench-spin 0.9s linear infinite;
}

.mqtt-workbench__menu-mask {
  position: fixed;
  inset: 0;
  z-index: 2100;
}

.mqtt-workbench__context-menu {
  position: fixed;
  min-width: 148px;
  padding: 4px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
}

.mqtt-workbench__context-menu button {
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

.mqtt-workbench__context-menu button:hover {
  background: var(--dc-surface-muted);
  color: var(--dc-primary);
}

.mqtt-workbench__context-menu button.is-danger:hover {
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
}

.mqtt-workbench__menu-icon {
  width: 15px;
  height: 15px;
  flex: 0 0 auto;
}

:deep(.mqtt-subscription-list),
:deep(.mqtt-message-viewer),
:deep(.mqtt-tag-list),
:deep(.mqtt-tag-monitor) {
  background: var(--dc-surface-raised);
}

:deep(.toolbar),
:deep(.mqtt-tag-list > div:first-child),
:deep(.mqtt-tag-monitor > div:first-child) {
  background: var(--dc-surface-subtle);
  border-color: var(--dc-border);
}

@keyframes mqtt-workbench-spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 1100px) {
  .mqtt-workbench {
    grid-template-columns: 240px minmax(0, 1fr);
  }
}

@media (max-width: 760px) {
  .mqtt-workbench {
    grid-template-columns: 1fr;
  }

  .mqtt-workbench__explorer {
    max-height: 320px;
    border-right: 0;
    border-bottom: 1px solid var(--dc-border);
  }

  .mqtt-workbench__tag-panel {
    grid-template-columns: 1fr;
  }
}
</style>
