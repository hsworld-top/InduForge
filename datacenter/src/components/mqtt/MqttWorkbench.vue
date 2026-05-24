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
              断开
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
            title="新建分组"
            aria-label="新建分组"
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
            @open-tags="openTagManager"
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
            @click="openTagManager(subscription)"
            @dblclick="openTagManager(subscription)"
            @keydown.enter="openTagManager(subscription)"
            @contextmenu.prevent.stop="openSubscriptionMenu($event, subscription)"
          >
            <IconTablerRss class="mqtt-workbench__tree-item-icon" />
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
            {{ filterText ? `没有匹配的${treeItemNoun}` : `暂无${treeItemNoun}` }}
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
          @click="activeTabId = tab.id"
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
              :ref="(el) => setMessageViewerRef(tab.id, el)"
              :subscription="tab.subscription"
              :project-id="projectIdText"
              :connection-id="connection.id"
              :source="workbenchMode"
            />

            <div
              v-else-if="tab.type === 'tags'"
              class="mqtt-workbench__tag-panel"
              :class="{ 'is-config-only': !connectionStarted }"
            >
              <div class="mqtt-workbench__tag-list">
                <MqttTagList
                  :project-id="projectIdText"
                  :subscription-id="tab.subscription.id"
                  :preview-session-id="connectionStarted ? previewSessionId : ''"
                  @open-monitor="openTagMonitor(tab.subscription)"
                  @open-publish="openPublishDialog(tab.subscription)"
                />
              </div>
            </div>
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
      @success="handleSubscriptionSaved"
    />

    <MqttSubscriptionGroupDialog
      v-model="groupDialogVisible"
      :mode="groupDialogMode"
      :groups="subscriptionGroups"
      :group="contextGroup"
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
      title="变量预览/监控"
      width="960px"
      class="mqtt-workbench__monitor-dialog"
      @close="closeMonitorDialog"
    >
      <div class="mqtt-workbench__monitor-heading">
        <strong>{{ monitorSubscription?.name || monitorSubscription?.topic }}</strong>
        <span>{{ monitorSubscription?.topic }}</span>
      </div>
      <MqttTagMonitor
        v-if="monitorDialogVisible && monitorSubscription"
        :project-id="projectIdText"
        :subscription-id="monitorSubscription.id"
        :preview-session-id="previewSessionId"
      />
    </DcDialog>

    <DcDialog v-model="publishDialogVisible" title="发布测试" width="640px">
      <div class="mqtt-workbench__dialog-form">
        <label>
          <span>Topic</span>
          <el-input v-model="publishTopic" placeholder="device/demo/1" />
        </label>
        <label>
          <span>QoS</span>
          <el-input-number v-model="publishQos" :min="0" :max="2" />
        </label>
        <label>
          <span>Payload JSON</span>
          <el-input v-model="payloadText" type="textarea" :rows="10" spellcheck="false" />
        </label>
        <el-alert
          v-if="publishErrorMessage"
          type="error"
          :closable="false"
          :title="publishErrorMessage"
          show-icon
        />
      </div>
      <template #footer>
        <el-button @click="publishDialogVisible = false">取消</el-button>
        <el-button
          type="primary"
          :loading="publishing"
          :disabled="!publishTopic.trim()"
          @click="publishMessage"
        >
          发布
        </el-button>
      </template>
    </DcDialog>

    <DcDialog
      v-model="detailDialogVisible"
      title="订阅详情"
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
          <div class="mqtt-workbench__panel-title">订阅信息</div>
          <dl class="mqtt-workbench__facts">
            <div>
              <dt>名称</dt>
              <dd>{{ detailSubscription.name || '-' }}</dd>
            </div>
            <div>
              <dt>Topic</dt>
              <dd>{{ detailSubscription.topic || '-' }}</dd>
            </div>
            <div>
              <dt>路径</dt>
              <dd>{{ subscriptionPath(detailSubscription) }}</dd>
            </div>
            <div>
              <dt>QoS</dt>
              <dd>{{ detailSubscription.qos ?? 0 }}</dd>
            </div>
            <div>
              <dt>消息保留数</dt>
              <dd>{{ detailSubscription.messageRetention ?? 1000 }}</dd>
            </div>
            <div>
              <dt>描述</dt>
              <dd>{{ detailSubscription.description || '-' }}</dd>
            </div>
            <div>
              <dt>{{ isBuiltinMessageMode ? 'Topic ID' : '订阅 ID' }}</dt>
              <dd>{{ detailSubscription.id }}</dd>
            </div>
          </dl>
        </section>

        <section class="mqtt-workbench__detail-section">
          <div class="mqtt-workbench__panel-title">
            {{ isBuiltinMessageMode ? '当前消息库' : '当前 Broker' }}
          </div>
          <dl class="mqtt-workbench__facts">
            <div>
              <dt>{{ isBuiltinMessageMode ? '标识' : '地址' }}</dt>
              <dd>{{ endpointText }}</dd>
            </div>
            <div>
              <dt>状态</dt>
              <dd>{{ connectionStarted ? '已连接' : '未连接' }}</dd>
            </div>
            <div>
              <dt>{{ treeItemNoun }}</dt>
              <dd>{{ subscriptions.length }}</dd>
            </div>
          </dl>
        </section>

        <div class="mqtt-workbench__detail-footer">
          <button type="button" @click="openDetailTagManager">
            <IconTablerTags />
            <span>变量管理</span>
          </button>
        </div>
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
            <span>详情</span>
          </button>
          <button
            v-if="contextMenu.type === 'subscription'"
            type="button"
            @click="emitContextAction('messages')"
          >
            <IconTablerMessages class="mqtt-workbench__menu-icon" />
            <span>查看消息</span>
          </button>
          <button
            v-if="contextMenu.type === 'subscription'"
            type="button"
            @click="emitContextAction('rename')"
          >
            <IconTablerPencil class="mqtt-workbench__menu-icon" />
            <span>重命名</span>
          </button>
          <button
            v-if="supportsSubscriptionGroups"
            type="button"
            @click="emitContextAction('move')"
          >
            <IconTablerFolderSymlink class="mqtt-workbench__menu-icon" />
            <span>移动到分组</span>
          </button>
          <button
            v-if="contextMenu.type === 'subscription'"
            type="button"
            class="is-danger"
            @click="emitContextAction('delete')"
          >
            <IconTablerTrash class="mqtt-workbench__menu-icon" />
            <span>删除</span>
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
import IconTablerLoader2 from '~icons/tabler/loader-2'
import IconTablerMessages from '~icons/tabler/messages'
import IconTablerPencil from '~icons/tabler/pencil'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerRss from '~icons/tabler/rss'
import IconTablerTags from '~icons/tabler/tags'
import IconTablerTrash from '~icons/tabler/trash'
import IconTablerX from '~icons/tabler/x'
import dataAPI from '@/api/data.api'
import WorkbenchSourceHeader from '@/components/workbench/WorkbenchSourceHeader.vue'
import { usePreviewSession } from '@/composables/usePreviewSession'
import { useMqttSocket } from '@/composables/useMqttSocket'
import { getApiErrorMessage } from '@/utils/request'
import DcDialog from '@/components/shared/DcDialog.vue'
import MqttMessageViewer from './MqttMessageViewer.vue'
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
    return String(config.value.runtimeKey || '未生成')
  }
  const host = config.value.brokerUrl || config.value.host || '未配置 Broker'
  const port = config.value.port ? `:${config.value.port}` : ''
  return `${host}${port}`
})
const fallbackTitle = computed(() =>
  isBuiltinMessageMode.value ? 'IF消息库' : '未命名 MQTT 接入源',
)
const treeItemNoun = computed(() => (isBuiltinMessageMode.value ? 'Topic' : '订阅'))
const filterPlaceholder = computed(() =>
  isBuiltinMessageMode.value ? '筛选 Topic' : '筛选订阅或 Topic',
)
const createSubscriptionLabel = computed(() =>
  isBuiltinMessageMode.value ? '新建 Topic' : '新建订阅',
)
const refreshLabel = computed(() => (isBuiltinMessageMode.value ? '刷新 Topic' : '刷新订阅'))
const loadingLabel = computed(() => (isBuiltinMessageMode.value ? '加载 Topic...' : '加载订阅...'))
const placeholderTitle = computed(() =>
  isBuiltinMessageMode.value ? '选择 Topic 开始工作' : '选择订阅开始工作',
)
const placeholderHint = computed(() =>
  isBuiltinMessageMode.value
    ? '左键管理变量，右键查看消息或打开实时监控。'
    : '从左侧订阅树进入变量管理；连接后可查看实时消息。',
)
const supportsSubscriptionGroups = computed(() => true)
const sourceMetaRows = computed(() =>
  isBuiltinMessageMode.value
    ? [
        { label: '类型', value: 'IF消息库' },
        { label: '标识', value: endpointText.value },
      ]
    : [
        { label: '类型', value: 'MQTT Broker' },
        { label: '地址', value: endpointText.value },
      ],
)

const { sessionId: previewSessionId, ensureSession } = usePreviewSession(projectIdRef, {
  autoStart: false,
})
const {
  connected: socketConnected,
  subscribeMessages,
  subscribeBuiltinMessage,
} = useMqttSocket(projectIdRef, previewSessionId)

const loading = ref(false)
const subscriptions = ref<MqttSubscription[]>([])
const subscriptionGroups = ref<MqttSubscriptionGroup[]>([])
const selectedSubscription = ref<MqttSubscription | null>(null)
const filterText = ref('')
const connectionStarted = ref(false)
const connectButtonLabel = computed(() => (connectionStarted.value ? '已连接' : '连接'))
const connectButtonTitle = computed(() => (connectionStarted.value ? '断开' : '连接'))
const tabs = ref<any[]>([])
const activeTabId = ref('')
const messageViewerRefs = shallowRef(new Map<string, any>())
const messageCleanups = new Map<string, () => void>()
const subscriptionDialogVisible = ref(false)
const subscriptionDialogMode = ref<'create' | 'edit'>('create')
const subscriptionDialogGroupId = ref<string | null>(null)
const editingSubscription = ref<MqttSubscription | null>(null)
const groupDialogVisible = ref(false)
const groupDialogMode = ref<'create' | 'rename'>('create')
const groupSaving = ref(false)
const moveDialogVisible = ref(false)
const moveTargetType = ref<'subscription' | 'group'>('subscription')
const moveSaving = ref(false)
const monitorDialogVisible = ref(false)
const monitorSubscription = ref<MqttSubscription | null>(null)
const detailDialogVisible = ref(false)
const detailSubscription = ref<MqttSubscription | null>(null)
const contextSubscription = ref<MqttSubscription | null>(null)
const contextGroup = ref<MqttSubscriptionGroupNode | null>(null)
const publishTopic = ref('')
const publishQos = ref(0)
const payloadText = ref(JSON.stringify({ value: 1 }, null, 2))
const publishDialogVisible = ref(false)
const publishing = ref(false)
const publishErrorMessage = ref('')
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

const activeTab = computed(() => tabs.value.find((tab) => tab.id === activeTabId.value) || null)
const hasActiveTab = computed(() =>
  Boolean(activeTabId.value && tabs.value.some((tab) => tab.id === activeTabId.value)),
)
const workbenchContentKey = computed(() => activeTab.value?.id || 'empty')
const activeContentTabs = computed(() =>
  hasActiveTab.value && activeTab.value ? [activeTab.value] : [],
)

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

const normalizeBuiltinMessage = (subscription: MqttSubscription, message: any) => ({
  id: String(message?.id || `${Date.now()}-${Math.random()}`),
  topic: String(message?.topic || subscription.topic || ''),
  payload: message?.payload,
  qos: Number(message?.qos ?? 0),
  timestamp: message?.timestamp || Date.now(),
})

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
        isBuiltinMessageMode.value ? '加载 IF消息库 Topic 失败' : '加载 MQTT 订阅树失败',
      ),
    )
  } finally {
    loading.value = false
  }
}

const openPublishDialog = (subscription: MqttSubscription) => {
  if (!connectionStarted.value) {
    ElMessage.warning('请先连接后再发布测试消息')
    return
  }
  publishTopic.value = subscription.topic || ''
  publishErrorMessage.value = ''
  publishDialogVisible.value = true
}

const publishMessage = async () => {
  let payload: unknown
  try {
    payload = JSON.parse(payloadText.value)
  } catch {
    publishErrorMessage.value = 'Payload 必须是合法 JSON'
    return
  }

  publishing.value = true
  publishErrorMessage.value = ''
  try {
    const input = {
      topic: publishTopic.value.trim(),
      qos: publishQos.value,
      payload,
    }
    if (!connectionStarted.value) {
      publishErrorMessage.value = '请先连接后再发布测试消息'
      return
    }
    if (isBuiltinMessageMode.value) {
      await dataAPI.publishBuiltinMessage(projectIdText.value, props.connection.id, input)
    } else {
      await dataAPI.publishMqttMessage(projectIdText.value, props.connection.id, input)
    }
    ElMessage.success('消息已发布')
    publishDialogVisible.value = false
  } catch (error) {
    publishErrorMessage.value = getApiErrorMessage(
      error,
      isBuiltinMessageMode.value ? '发布 IF消息库消息失败' : '发布 MQTT 消息失败',
    )
  } finally {
    publishing.value = false
  }
}

const ensureMqttPreview = async () => {
  const session = await ensureSession()
  if (!session) {
    ElMessage.warning('预览会话不可用')
    return false
  }
  if (connectionStarted.value) return true

  if (isBuiltinMessageMode.value) {
    connectionStarted.value = true
    ElMessage.success('IF消息库已连接')
    return true
  }

  try {
    await dataAPI.startMqttConnection(projectIdText.value, props.connection.id)
    connectionStarted.value = true
    ElMessage.success('MQTT 已连接')
    return true
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, 'MQTT 连接失败'))
    return false
  }
}

const stopMqttPreview = async () => {
  if (!connectionStarted.value) return true

  try {
    cleanupMessageSubscriptions()
    messageViewerRefs.value.clear()
    monitorDialogVisible.value = false
    monitorSubscription.value = null
    tabs.value = tabs.value.filter((tab) => tab.type !== 'messages')
    if (!tabs.value.some((tab) => tab.id === activeTabId.value)) {
      activeTabId.value = tabs.value[0]?.id || ''
    }
    if (!isBuiltinMessageMode.value) {
      await dataAPI.stopMqttConnection(projectIdText.value, props.connection.id)
    }
    connectionStarted.value = false
    ElMessage.success(isBuiltinMessageMode.value ? 'IF消息库已断开' : 'MQTT 已断开')
    return true
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, 'MQTT 断开失败'))
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
    activeTabId.value = existing.id
    return existing
  }
  tabs.value.push(tab)
  activeTabId.value = tab.id
  return tab
}

const selectSubscription = (subscription: MqttSubscription) => {
  void openTagManager(subscription)
}

const openMessages = async (subscription: MqttSubscription) => {
  selectedSubscription.value = subscription
  if (!connectionStarted.value) {
    ElMessage.warning('请先连接后再查看实时消息')
    return
  }

  const tab = addTab({
    id: `mqtt-messages-${subscription.id}`,
    type: 'messages',
    title: `${subscription.name || subscription.topic} / 消息`,
    icon: IconTablerMessages,
    subscription,
  })

  if (!messageCleanups.has(tab.id)) {
    const cleanup = isBuiltinMessageMode.value
      ? subscribeBuiltinMessage(subscription.id, props.connection.id, (data) => {
          const viewer = messageViewerRefs.value.get(tab.id)
          viewer?.addMessage?.(normalizeBuiltinMessage(subscription, data?.message || data || {}))
          viewer?.setConnected?.(true)
        })
      : subscribeMessages(subscription.id, (data) => {
          const viewer = messageViewerRefs.value.get(tab.id)
          viewer?.addMessage?.(data?.message || data)
          viewer?.setConnected?.(true)
        })
    messageCleanups.set(tab.id, cleanup)
  }

  await nextTick()
  messageViewerRefs.value.get(tab.id)?.setConnected?.(socketConnected.value)
}

const openTagManager = async (subscription: MqttSubscription) => {
  selectedSubscription.value = subscription
  if (isBuiltinMessageMode.value) {
    publishTopic.value = subscription.topic || ''
  }

  addTab({
    id: `mqtt-tags-${subscription.id}`,
    type: 'tags',
    title: `${subscription.name || subscription.topic} / 变量`,
    icon: IconTablerTags,
    subscription,
  })
}

const openTagMonitor = (subscription: MqttSubscription) => {
  selectedSubscription.value = subscription
  if (!connectionStarted.value) {
    ElMessage.warning('请先连接后再打开变量监控')
    return
  }
  monitorSubscription.value = subscription
  monitorDialogVisible.value = true
}

const closeMonitorDialog = () => {
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
          title: `${subscription.name || subscription.topic} / ${
            tab.type === 'messages' ? '消息' : '变量'
          }`,
          subscription,
        }
      : tab,
  )
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
  messageRetention: patch.messageRetention ?? subscription.messageRetention ?? 1000,
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

function openDetailTagManager() {
  const subscription = detailSubscription.value
  if (!subscription) return
  detailDialogVisible.value = false
  void openTagManager(subscription)
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

  return ['根目录', ...names].join(' / ')
}

async function handleSubscriptionSaved(data?: any) {
  const saved = data?.subscription || data
  if (saved?.id) refreshSelectedAndTabs(saved)
  await loadWorkbenchTree()
}

function openCreateGroupDialog() {
  if (!supportsSubscriptionGroups.value) {
    return
  }
  contextGroup.value = null
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
      ElMessage.success('分组已创建')
    } else if (contextGroup.value) {
      await dataAPI.updateMqttSubscriptionGroup(projectIdText.value, contextGroup.value.id, {
        name: data.name,
      })
      ElMessage.success('分组已重命名')
    }
    groupDialogVisible.value = false
    await loadWorkbenchTree()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '保存分组失败'))
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
      ElMessage.success('订阅已移动')
    } else if (moveTargetType.value === 'group' && contextGroup.value) {
      await dataAPI.updateMqttSubscriptionGroup(projectIdText.value, contextGroup.value.id, {
        parentId: groupId,
        hasParentId: true,
      })
      ElMessage.success('分组已移动')
    }
    moveDialogVisible.value = false
    await loadWorkbenchTree()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '移动失败'))
  } finally {
    moveSaving.value = false
  }
}

async function deleteSubscription(subscription: MqttSubscription) {
  const ok = await ElMessageBox.confirm(
    `确认删除订阅「${subscription.name || subscription.topic || subscription.id}」？此操作不可恢复。`,
    '删除订阅',
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
    await dataAPI.deleteMqttSubscription(projectIdText.value, subscription.id)
    tabs.value = tabs.value.filter((tab) => tab.subscription?.id !== subscription.id)
    if (!tabs.value.some((tab) => tab.id === activeTabId.value)) {
      activeTabId.value = tabs.value[0]?.id || ''
    }
    if (selectedSubscription.value?.id === subscription.id) {
      selectedSubscription.value = null
    }
    ElMessage.success('订阅已删除')
    await loadWorkbenchTree()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '删除订阅失败'))
  }
}

const countGroupSubscriptions = (group: MqttSubscriptionGroupNode): number =>
  group.subscriptions.length +
  group.children.reduce((sum, child) => sum + countGroupSubscriptions(child), 0)

async function deleteGroup(group: MqttSubscriptionGroupNode) {
  const subscriptionCount = countGroupSubscriptions(group)
  const ok = await ElMessageBox.confirm(
    `确认删除分组「${group.name}」？组内 ${subscriptionCount} 个订阅会回到根目录。`,
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
    await dataAPI.deleteMqttSubscriptionGroup(projectIdText.value, group.id)
    ElMessage.success('分组已删除')
    await loadWorkbenchTree()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '删除分组失败'))
  }
}

function openSubscriptionMenu(event: MouseEvent, subscription: MqttSubscription) {
  contextMenu.value = {
    visible: true,
    type: 'subscription',
    x: Math.min(event.clientX, window.innerWidth - 180),
    y: Math.min(event.clientY, window.innerHeight - 156),
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

function emitContextAction(action: 'rename' | 'move' | 'delete' | 'messages' | 'detail') {
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
    if (action === 'detail') {
      openSubscriptionDetail(subscription)
      return
    }
    void deleteSubscription(subscription)
    return
  }
  if (type === 'group' && group) {
    if (!supportsSubscriptionGroups.value) {
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
    return
  }
  messageViewerRefs.value.delete(tabId)
}

const cleanupMessageSubscriptions = () => {
  Array.from(messageCleanups.values()).forEach((cleanup) => cleanup?.())
  messageCleanups.clear()
}

onMounted(async () => {
  await loadWorkbenchTree()
})

watch(
  () => props.connection.id,
  async () => {
    cleanupMessageSubscriptions()
    messageViewerRefs.value.clear()
    monitorDialogVisible.value = false
    monitorSubscription.value = null
    detailDialogVisible.value = false
    detailSubscription.value = null
    tabs.value = []
    activeTabId.value = ''
    selectedSubscription.value = null
    connectionStarted.value = false
    publishTopic.value = ''
    publishErrorMessage.value = ''
    await loadWorkbenchTree()
  },
)

watch(
  socketConnected,
  (connected) => {
    messageViewerRefs.value.forEach((viewer) => {
      viewer?.setConnected?.(connected)
    })
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  cleanupMessageSubscriptions()
  messageViewerRefs.value.clear()
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
  min-height: 0;
  flex: 1;
  overflow: hidden;
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

.mqtt-workbench__monitor-heading {
  display: grid;
  gap: 4px;
  margin-bottom: 10px;
  padding: 0 2px;
}

.mqtt-workbench__monitor-heading strong,
.mqtt-workbench__monitor-heading span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mqtt-workbench__monitor-heading strong {
  color: var(--dc-text);
  font-size: 14px;
}

.mqtt-workbench__monitor-heading span {
  color: var(--dc-text-muted);
  font-family: var(--dc-font-mono, monospace);
  font-size: 12px;
}

.mqtt-workbench__detail-footer {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

.mqtt-workbench__detail-footer button {
  height: 32px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 0 12px;
  border: 1px solid var(--dc-primary);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-primary);
  color: var(--dc-surface-raised);
  font-size: 12px;
  font-weight: 700;
}

.mqtt-workbench__detail-footer svg {
  width: 15px;
  height: 15px;
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
