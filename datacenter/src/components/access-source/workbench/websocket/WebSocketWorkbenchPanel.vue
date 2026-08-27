<template>
  <section class="ws-workbench">
    <aside class="ws-workbench__sidebar">
      <WorkbenchSourceHeader
        :title="connection.name || '未命名 WebSocket 接入源'"
        fallback-title="未命名 WebSocket 接入源"
        :meta="sourceMetaRows"
        @back="$emit('back')"
      >
        <template #actions>
          <el-input
            v-model="search"
            class="ws-workbench__search"
            size="small"
            clearable
            placeholder="搜索会话"
            @keyup.enter="loadSessions(1)"
          />
          <button
            class="workbench-source-header__icon-action is-primary"
            title="新建会话"
            aria-label="新建会话"
            type="button"
            @click="createDraftSession"
          >
            <IconTablerPlus />
          </button>
          <button
            class="workbench-source-header__icon-action"
            title="新建分组"
            aria-label="新建分组"
            type="button"
            @click="openGroupDialog()"
          >
            <IconTablerFolderPlus />
          </button>
          <button
            class="workbench-source-header__icon-action"
            title="刷新"
            aria-label="刷新"
            type="button"
            @click="reloadWorkbench"
          >
            <IconTablerRefresh />
          </button>
        </template>
      </WorkbenchSourceHeader>

      <div class="ws-workbench__tree">
        <div v-if="loading" class="ws-workbench__loading">
          <IconTablerLoader2 />
          <span>加载会话...</span>
        </div>
        <template v-else>
          <WebSocketTreeNode
            v-for="group in sessionTree.groups"
            :key="group.id"
            :node="group"
            :active-tab-id="activeTabId"
            :expanded-groups="expandedGroups"
            @toggle="toggleGroup"
            @open-session="openSession"
            @session-contextmenu="openSessionMenu"
            @group-contextmenu="openGroupMenu"
          />
          <button
            v-for="session in sessionTree.rootSessions"
            :key="session.id"
            type="button"
            class="ws-workbench__session-node"
            :class="{ 'is-active': activeTabId === String(session.id) }"
            @click="openSession(session)"
            @contextmenu.prevent.stop="openSessionMenu($event, session)"
          >
            <span class="ws-workbench__badge">WS</span>
            <span class="ws-workbench__session-name">{{ session.name }}</span>
            <el-tag
              v-if="session.quality !== 'unknown'"
              size="small"
              :type="session.quality === 'good' ? 'success' : 'danger'"
            >
              {{ session.quality }}
            </el-tag>
          </button>
          <div v-if="sessions.length === 0" class="ws-workbench__empty">
            {{ search ? '没有匹配的会话' : '暂无会话' }}
          </div>
          <button
            v-if="sessions.length < pagination.total"
            type="button"
            class="ws-workbench__load-more"
            :disabled="loadingMore"
            @click="loadMoreSessions"
          >
            {{ loadingMore ? '加载中...' : `加载更多（${sessions.length}/${pagination.total}）` }}
          </button>
        </template>
      </div>
    </aside>

    <main class="ws-workbench__main">
      <div class="ws-workbench__tabs">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          type="button"
          class="ws-workbench__tab"
          :class="{ 'is-active': tab.id === activeTabId }"
          @click="activateTab(tab.id)"
        >
          <span class="ws-workbench__badge">WS</span>
          <el-tooltip
            :content="tab.draft.name || '未命名会话'"
            placement="top"
            :show-after="400"
            :disabled="(tab.draft.name || '').length <= 12"
          >
            <span class="ws-workbench__tab-name">{{ tab.draft.name || '未命名会话' }}</span>
          </el-tooltip>
          <i v-if="tab.dirty" />
          <IconTablerX @click.stop="closeTab(tab.id)" />
        </button>
      </div>

      <div v-if="activeTab" class="ws-workbench__editor">
        <div class="ws-workbench__crumb-row">
          <div class="ws-workbench__crumb">
            <span>WebSocket 接入源</span>
            <IconTablerChevronRight />
            <span>{{ groupName(activeTab.draft.groupId) }}</span>
            <IconTablerChevronRight />
            <el-input
              v-model="activeTab.draft.name"
              class="ws-workbench__name-input"
              size="small"
              placeholder="未命名会话"
              maxlength="64"
              @input="markDirty"
            />
          </div>
          <div class="ws-workbench__actions">
            <span v-if="activeTab.draft.outputs.length" class="ws-workbench__datapoint-path">
              {{ activeTab.draft.outputs.length }} 个输出数据点
            </span>
            <el-tooltip content="保存当前会话" placement="top" :show-after="400">
              <el-button
                size="small"
                type="primary"
                :loading="saving"
                class="ws-workbench__icon-btn"
                aria-label="保存"
                @click="saveActive"
              >
                <IconTablerDeviceFloppy />
              </el-button>
            </el-tooltip>
          </div>
        </div>

        <div class="ws-workbench__request-line">
          <el-select
            v-model="activeUrlScheme"
            class="ws-workbench__protocol-select"
            @change="markDirty"
          >
            <el-option label="ws://" value="ws" />
            <el-option label="wss://" value="wss" />
          </el-select>
          <el-input
            v-model="activeTab.draft.url"
            placeholder="ws://example.com/stream 或 wss://example.com/stream"
            @input="markDirty"
          />
          <el-tooltip :content="connectButtonTooltip" placement="top" :show-after="400">
            <el-button
              :type="connectButtonType"
              :loading="activeTab.streamStatus === 'connecting'"
              class="ws-workbench__icon-btn ws-workbench__connect-btn"
              :class="`is-${activeTab.streamStatus}`"
              :aria-label="isActiveConnected ? '断开连接' : '连接'"
              @click="connectActive"
            >
              <IconTablerPlugConnected v-if="isActiveConnected" />
              <IconTablerPlugConnectedX v-else-if="activeTab.streamStatus !== 'connecting'" />
            </el-button>
          </el-tooltip>
        </div>

        <el-tabs v-model="activeConfigTab" class="ws-workbench__config-tabs">
          <el-tab-pane label="请求头" name="headers">
            <WebSocketKeyValueEditor v-model="activeTab.draft.headers" @change="markDirty" />
          </el-tab-pane>
          <el-tab-pane label="认证" name="auth">
            <el-form class="ws-workbench__form" label-position="top" size="small" @submit.prevent>
              <el-form-item label="认证方式">
                <el-select
                  v-model="activeTab.draft.auth.type"
                  class="ws-workbench__form-control"
                  @change="markDirty"
                >
                  <el-option label="无认证" value="none" />
                  <el-option label="Bearer Token" value="bearer" />
                  <el-option label="Basic Auth" value="basic" />
                </el-select>
              </el-form-item>
              <el-form-item v-if="activeTab.draft.auth.type === 'bearer'" label="Token">
                <el-input
                  v-model="activeTab.draft.auth.token"
                  class="ws-workbench__form-control"
                  placeholder="请输入 Token"
                  @input="markDirty"
                />
              </el-form-item>
              <template v-if="activeTab.draft.auth.type === 'basic'">
                <el-form-item label="用户名">
                  <el-input
                    v-model="activeTab.draft.auth.username"
                    class="ws-workbench__form-control"
                    placeholder="请输入用户名"
                    @input="markDirty"
                  />
                </el-form-item>
                <el-form-item label="密码">
                  <el-input
                    v-model="activeTab.draft.auth.password"
                    class="ws-workbench__form-control"
                    placeholder="请输入密码"
                    show-password
                    @input="markDirty"
                  />
                </el-form-item>
              </template>
            </el-form>
          </el-tab-pane>
          <el-tab-pane label="子协议" name="protocols">
            <WebSocketProtocolEditor v-model="activeTab.draft.protocols" @change="markDirty" />
          </el-tab-pane>
          <el-tab-pane label="消息" name="messages">
            <div class="ws-workbench__message-composer">
              <div class="ws-workbench__message-toolbar">
                <el-radio-group v-model="activeTab.messageMode" size="small" @change="markDirty">
                  <el-radio-button label="json">json</el-radio-button>
                  <el-radio-button label="text">text</el-radio-button>
                </el-radio-group>
              </div>
              <MonacoEditor
                v-model="activeTab.messageText"
                :language="activeTab.messageMode === 'json' ? 'json' : 'plaintext'"
                height="240px"
                theme="vs-dark"
                :options="messageEditorOptions"
                @change="handleMessageEditorChange"
              />
              <div class="ws-workbench__message-composer-footer">
                <el-button
                  type="primary"
                  size="small"
                  :disabled="
                    activeTab.streamStatus !== 'connected' || !activeTab.messageText.trim()
                  "
                  @click="sendActiveMessage"
                >
                  <IconTablerSend />
                  发送消息
                </el-button>
              </div>
            </div>
          </el-tab-pane>
          <el-tab-pane label="输出映射" name="outputs">
            <SourceOutputEditor
              v-model="activeTab.draft.outputs"
              title="消息输出"
              :sample="latestIncomingPayload"
              whole-data-type="object"
              @change="markDirty"
            />
          </el-tab-pane>
          <el-tab-pane label="设置" name="settings">
            <el-form class="ws-workbench__form" label-position="top" size="small" @submit.prevent>
              <div class="ws-workbench__form-grid">
                <el-form-item label="超时（毫秒）">
                  <el-input-number
                    v-model="activeTab.draft.settings.timeoutMs"
                    :min="1000"
                    :max="30000"
                    :step="500"
                    controls-position="right"
                    @change="markDirty"
                  />
                </el-form-item>
                <el-form-item label="行为">
                  <div class="ws-workbench__settings-checks">
                    <el-checkbox v-model="activeTab.draft.settings.tlsVerify" @change="markDirty">
                      TLS 校验
                    </el-checkbox>
                  </div>
                </el-form-item>
              </div>
            </el-form>
          </el-tab-pane>
        </el-tabs>

        <section class="ws-workbench__response" :style="responsePanelStyle">
          <div
            class="ws-workbench__response-resizer"
            role="separator"
            aria-orientation="horizontal"
            title="拖拽调整消息流高度，双击恢复默认"
            @mousedown.prevent="startResponseResize"
            @dblclick="resetResponseHeight"
          />
          <header>
            <div>
              <strong>消息流</strong>
              <WorkbenchStatusPill :label="streamStatusLabel" :tone="streamStatusTone" />
            </div>
          </header>
          <div class="ws-workbench__response-body">
            <div v-show="activeTab.streamStatus === 'error'" class="ws-workbench__error-response">
              <IconTablerAlertTriangle />
              <strong>连接失败</strong>
              <span>{{ activeTab.streamError || '未知错误' }}</span>
            </div>
            <div
              v-show="activeTab.streamStatus !== 'error' && activeTab.streamMessages.length === 0"
              class="ws-workbench__empty-response"
            >
              {{
                activeTab.streamStatus === 'connected'
                  ? '已连接，等待消息...'
                  : activeTab.streamStatus === 'connecting'
                    ? '连接中...'
                    : '连接后持续显示 WebSocket 消息'
              }}
            </div>
            <div v-show="activeTab.streamMessages.length > 0" class="ws-workbench__messages">
              <article
                v-for="(message, index) in activeTab.streamMessages"
                :key="index"
                class="ws-workbench__message"
                :class="`is-${message.direction}`"
              >
                <div>
                  <el-tag size="small" :type="message.direction === 'in' ? 'success' : 'info'">
                    {{ message.direction === 'in' ? '接收' : '发送' }}
                  </el-tag>
                  <span>{{ message.type }}</span>
                  <small>{{ message.sizeBytes }} bytes</small>
                </div>
                <pre>{{ formatJSON(message.payload ?? message.rawPayload) }}</pre>
              </article>
              <pre v-if="activeTab.streamError" class="ws-workbench__diagnostic">{{
                activeTab.streamError
              }}</pre>
            </div>
          </div>
        </section>
      </div>

      <div v-else class="ws-workbench__blank">
        <IconTablerWebhook />
        <strong>选择或新建一个 WebSocket 会话</strong>
        <span>每个会话可以把完整消息或样本字段映射为一个或多个强类型数据点。</span>
      </div>
    </main>

    <WorkbenchGroupDialog
      ref="groupDialogRef"
      v-model="groupDialog.visible"
      :mode="groupDialog.id ? 'edit' : 'create'"
      :title="groupDialog.id ? '编辑分组' : '新建分组'"
      :group="editingGroup"
      :group-options="groupOptions"
      :initial-parent-id="groupDialog.parentId"
      :loading="groupSaving"
      @submit="saveGroup"
    />

    <DcDialog
      v-model="moveDialogVisible"
      :title="moveTargetType === 'group' ? '移动分组' : '移动会话'"
      width="420px"
      :close-disabled="moveSaving"
    >
      <el-form label-position="top" class="ws-workbench__move-form" @submit.prevent>
        <el-form-item label="目标分组">
          <el-select
            v-model="moveTargetGroupId"
            class="ws-workbench__move-select"
            clearable
            placeholder="根目录"
          >
            <el-option label="根目录" :value="null" />
            <el-option
              v-for="group in allGroupOptions"
              :key="group.id"
              :label="group.label"
              :value="group.id"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="ws-workbench__move-footer">
          <el-button @click="moveDialogVisible = false">取消</el-button>
          <el-button
            type="primary"
            :loading="moveSaving"
            :disabled="!canMoveTarget"
            @click="moveTarget"
          >
            移动
          </el-button>
        </div>
      </template>
    </DcDialog>

    <Teleport to="body">
      <div
        v-if="contextMenu.visible"
        class="ws-workbench__menu-mask"
        @click="closeContextMenu"
        @contextmenu.prevent="closeContextMenu"
      >
        <div
          class="ws-workbench__context-menu"
          :style="{ left: `${contextMenu.x}px`, top: `${contextMenu.y}px` }"
          @click.stop
        >
          <button
            v-if="contextMenu.type === 'session'"
            type="button"
            @click="emitContextAction('open')"
          >
            <IconTablerWebhook class="ws-workbench__menu-icon" />
            <span>打开会话</span>
          </button>
          <button
            v-if="contextMenu.type === 'session'"
            type="button"
            @click="emitContextAction('duplicate')"
          >
            <IconTablerCopy class="ws-workbench__menu-icon" />
            <span>复制会话</span>
          </button>
          <button
            v-if="contextMenu.type === 'session'"
            type="button"
            @click="emitContextAction('move')"
          >
            <IconTablerFolderSymlink class="ws-workbench__menu-icon" />
            <span>移动到分组</span>
          </button>
          <button
            v-if="contextMenu.type === 'group'"
            type="button"
            @click="emitContextAction('create-child')"
          >
            <IconTablerFolderPlus class="ws-workbench__menu-icon" />
            <span>新建子分组</span>
          </button>
          <button
            v-if="contextMenu.type === 'group'"
            type="button"
            @click="emitContextAction('edit')"
          >
            <IconTablerPencil class="ws-workbench__menu-icon" />
            <span>编辑分组</span>
          </button>
          <button
            v-if="contextMenu.type === 'group'"
            type="button"
            @click="emitContextAction('move')"
          >
            <IconTablerFolderSymlink class="ws-workbench__menu-icon" />
            <span>移动分组</span>
          </button>
          <button
            v-if="contextMenu.type === 'group'"
            type="button"
            class="is-danger"
            @click="emitContextAction('delete')"
          >
            <IconTablerTrash class="ws-workbench__menu-icon" />
            <span>删除分组</span>
          </button>
          <button
            v-if="contextMenu.type === 'session'"
            type="button"
            class="is-danger"
            @click="emitContextAction('delete')"
          >
            <IconTablerTrash class="ws-workbench__menu-icon" />
            <span>删除</span>
          </button>
        </div>
      </div>
    </Teleport>
  </section>
</template>

<script setup lang="ts">
import {
  computed,
  defineComponent,
  h,
  inject,
  onBeforeUnmount,
  onMounted,
  reactive,
  ref,
  watch,
  type PropType,
} from 'vue'
import {
  ElButton,
  ElCheckbox,
  ElInput,
  ElMessage,
  ElMessageBox,
  ElTable,
  ElTableColumn,
} from 'element-plus'
import IconTablerAlertTriangle from '~icons/tabler/alert-triangle'
import IconTablerChevronRight from '~icons/tabler/chevron-right'
import IconTablerCopy from '~icons/tabler/copy'
import IconTablerDeviceFloppy from '~icons/tabler/device-floppy'
import IconTablerFolder from '~icons/tabler/folder'
import IconTablerFolderPlus from '~icons/tabler/folder-plus'
import IconTablerFolderSymlink from '~icons/tabler/folder-symlink'
import IconTablerLoader2 from '~icons/tabler/loader-2'
import IconTablerPencil from '~icons/tabler/pencil'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerPlugConnected from '~icons/tabler/plug-connected'
import IconTablerPlugConnectedX from '~icons/tabler/plug-connected-x'
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerTrash from '~icons/tabler/trash'
import IconTablerWebhook from '~icons/tabler/webhook'
import IconTablerX from '~icons/tabler/x'
import WorkbenchSourceHeader from '@/components/workbench/WorkbenchSourceHeader.vue'
import WorkbenchGroupDialog from '@/components/workbench/WorkbenchGroupDialog.vue'
import WorkbenchStatusPill from '@/components/workbench/WorkbenchStatusPill.vue'
import DcDialog from '@/components/shared/DcDialog.vue'
import SourceOutputEditor from '@/components/shared/SourceOutputEditor.vue'
import MonacoEditor from '@/components/MonacoEditor.vue'
import * as dataAPI from '@/api/data.api'
import type {
  WebSocketKeyValueRow,
  WebSocketPreviewMessage,
  WebSocketProtocolRow,
  WebSocketSession,
  WebSocketSessionGroup,
  WebSocketSubscribeMessageRow,
} from '@/api/schemas/websocket-workbench.schema'
import type { SourceOutputInput } from '@/api/schemas/source-output.schema'
import { createWholeSourceOutput } from '@/api/schemas/source-output.schema'
import { getApiErrorMessage } from '@/utils/request'
import { useWorkbenchBottomPanelResize } from '@/composables/useWorkbenchBottomPanelResize'

type AccessSourceConnection = {
  id: string
  name?: string
  type?: string
  config?: Record<string, any>
}

type WebSocketAuthDraft = {
  type: string
  token?: string
  username?: string
  password?: string
  [key: string]: unknown
}

type WebSocketSettingsDraft = {
  timeoutMs: number
  tlsVerify: boolean
  [key: string]: unknown
}

type SessionDraft = {
  id?: string
  projectId?: string
  connectionId: string
  groupId: string | null
  name: string
  url: string
  headers: WebSocketKeyValueRow[]
  auth: WebSocketAuthDraft
  protocols: WebSocketProtocolRow[]
  messages: WebSocketSubscribeMessageRow[]
  settings: WebSocketSettingsDraft
  enabled: boolean
  sortOrder: number
  sourceType?: string
  dataPointId: string
  dataPointPath: string
  outputs: SourceOutputInput[]
  lastMessage?: unknown
  lastDiagnostic: string
  quality: 'good' | 'bad' | 'unknown' | string
  lastMessageAt?: string
}
type SessionTab = {
  id: string
  isNew: boolean
  dirty: boolean
  draft: SessionDraft
  socket: WebSocket | null
  streamEpoch: number
  connectTimer: ReturnType<typeof setTimeout> | null
  streamMessages: WebSocketPreviewMessage[]
  streamStatus: 'idle' | 'connecting' | 'connected' | 'error'
  streamError: string
  messageText: string
  messageMode: 'json' | 'text'
}

type WebSocketSessionGroupNode = WebSocketSessionGroup & {
  id: string
  parentId?: string | null
  children: WebSocketSessionGroupNode[]
  sessions: WebSocketSession[]
}

const props = defineProps<{ connection: AccessSourceConnection; projectId: string }>()
defineEmits<{ (event: 'back'): void }>()

const {
  panelStyle: responsePanelStyle,
  startResize: startResponseResize,
  resetHeight: resetResponseHeight,
} = useWorkbenchBottomPanelResize({
  defaultHeight: 320,
  bodyClass: 'ws-workbench--resizing-panel',
})

const loading = ref(false)
const loadingMore = ref(false)
const saving = ref(false)
const groupSaving = ref(false)
const search = ref('')
const activeConfigTab = ref('headers')
const messageEditorOptions = {
  minimap: { enabled: false },
  fontSize: 12,
  scrollBeyondLastLine: false,
  automaticLayout: true,
  wordWrap: 'on',
  tabSize: 2,
}
const groups = ref<WebSocketSessionGroup[]>([])
const sessions = ref<WebSocketSession[]>([])
const expandedGroups = ref(new Set<string>())
const pagination = reactive({ page: 1, pageSize: 50, total: 0, totalPages: 0 })
const tabs = ref<SessionTab[]>([])
const registerDraftChecker =
  inject<
    (guard: {
      isDirty: () => boolean
      save: () => Promise<boolean>
      discard: () => void
    }) => () => void
  >('registerDraftChecker')
let unregisterDraftChecker: (() => void) | undefined
const activeTabId = ref('')
const groupDialog = reactive({ visible: false, id: '', name: '', parentId: '' })
const groupDialogRef = ref<InstanceType<typeof WorkbenchGroupDialog> | null>(null)
const moveDialogVisible = ref(false)
const moveSaving = ref(false)
const moveTargetType = ref<'session' | 'group'>('session')
const movingSession = ref<WebSocketSession | null>(null)
const movingGroup = ref<WebSocketSessionGroupNode | null>(null)
const moveTargetGroupId = ref<string | null>(null)
const contextMenu = ref<{
  visible: boolean
  type: 'session' | 'group' | null
  x: number
  y: number
  session: WebSocketSession | null
  group: WebSocketSessionGroupNode | null
}>({
  visible: false,
  type: null,
  x: 0,
  y: 0,
  session: null,
  group: null,
})
let searchTimer: number | undefined
let sessionListSequence = 0

const sourceMetaRows = computed(() => [
  { label: '类型', value: 'WebSocket' },
  { label: '连接数', value: `${pagination.total || sessions.value.length} 个` },
])
const activeTab = computed(() => tabs.value.find((tab) => tab.id === activeTabId.value))
const latestIncomingPayload = computed(() => {
  const messages = activeTab.value?.streamMessages || []
  for (let index = messages.length - 1; index >= 0; index -= 1) {
    if (messages[index]?.direction === 'in') return messages[index]?.payload
  }
  return activeTab.value?.draft.lastMessage
})
const isActiveConnected = computed(() => activeTab.value?.streamStatus === 'connected')
const connectButtonType = computed(() => {
  const status = activeTab.value?.streamStatus
  if (status === 'connected') return 'success'
  if (status === 'error') return 'danger'
  return 'primary'
})
const connectButtonTooltip = computed(() => {
  const tab = activeTab.value
  if (!tab) return '连接'
  if (tab.streamStatus === 'connected') return '断开连接'
  if (tab.streamStatus === 'connecting') return '连接中...'
  if (tab.streamStatus === 'error') return tab.streamError || '连接失败，点击重试'
  return '连接'
})
const streamStatusLabel = computed(() => {
  const tab = activeTab.value
  if (!tab) return '未连接'
  return streamStatusText(tab)
})
const streamStatusTone = computed<'neutral' | 'success' | 'warning' | 'danger' | 'info'>(() => {
  const status = activeTab.value?.streamStatus
  if (status === 'connected') return 'success'
  if (status === 'connecting') return 'info'
  if (status === 'error') return 'danger'
  return 'neutral'
})
const activeUrlScheme = computed({
  get() {
    const url = activeTab.value?.draft.url || ''
    return url.toLowerCase().startsWith('wss://') ? 'wss' : 'ws'
  },
  set(value: 'ws' | 'wss') {
    const tab = activeTab.value
    if (!tab) return
    tab.draft.url = replaceURLScheme(tab.draft.url, value)
  },
})
const sessionTree = computed(() => buildWebSocketSessionTree(groups.value, sessions.value))
const allGroupOptions = computed(() =>
  flattenWebSocketGroups(
    groups.value,
    moveTargetType.value === 'group' ? collectWebSocketGroupIds(movingGroup.value) : new Set(),
  ),
)
const groupOptions = computed(() =>
  flattenWebSocketGroups(
    groups.value,
    groupDialog.id ? collectWebSocketGroupIds(findWebSocketGroupNode(groupDialog.id)) : new Set(),
  ),
)
const movingSessionGroupId = computed(() =>
  movingSession.value?.groupId ? String(movingSession.value.groupId) : null,
)
const movingGroupParentId = computed(() =>
  movingGroup.value?.parentId ? String(movingGroup.value.parentId) : null,
)
const canMoveTarget = computed(
  () =>
    !moveSaving.value &&
    (moveTargetType.value === 'session'
      ? Boolean(movingSession.value) && moveTargetGroupId.value !== movingSessionGroupId.value
      : Boolean(movingGroup.value) && moveTargetGroupId.value !== movingGroupParentId.value),
)
const editingGroup = computed(() =>
  groupDialog.id
    ? {
        id: groupDialog.id,
        name: groupDialog.name,
        parentId: groupDialog.parentId || null,
      }
    : null,
)

onMounted(() => {
  reloadWorkbench()
  unregisterDraftChecker = registerDraftChecker?.({
    isDirty: () => tabs.value.some((tab) => tab.dirty && !isPristineNewDraft(tab.draft)),
    save: saveDirtyTabs,
    discard: () => tabs.value.forEach((tab) => (tab.dirty = false)),
  })
})

watch(search, () => {
  window.clearTimeout(searchTimer)
  searchTimer = window.setTimeout(() => loadSessions(1), 250)
})

onBeforeUnmount(() => {
  unregisterDraftChecker?.()
  window.clearTimeout(searchTimer)
  tabs.value.forEach(closeStream)
})

async function reloadWorkbench() {
  await Promise.all([loadGroups(), loadSessions(1)])
}

async function loadGroups() {
  const result = await dataAPI.getWebSocketSessionGroups(props.projectId, props.connection.id)
  groups.value = result.list
  const next = new Set(expandedGroups.value)
  groups.value.forEach((group) => next.add(String(group.id)))
  expandedGroups.value = next
}

async function loadSessions(page = 1, append = false) {
  const requestSequence = ++sessionListSequence
  const projectId = props.projectId
  const connectionId = props.connection.id
  if (append) {
    loadingMore.value = true
  } else {
    loading.value = true
    loadingMore.value = false
  }
  try {
    const params: Record<string, any> = {
      page,
      pageSize: pagination.pageSize,
      q: search.value || undefined,
    }
    const result = await dataAPI.getWebSocketSessions(projectId, connectionId, params)
    if (
      requestSequence !== sessionListSequence ||
      projectId !== props.projectId ||
      connectionId !== props.connection.id
    ) {
      return
    }
    if (append) {
      const byId = new Map(sessions.value.map((item) => [String(item.id), item]))
      result.list.forEach((item) => byId.set(String(item.id), item))
      sessions.value = Array.from(byId.values())
    } else {
      sessions.value = result.list
    }
    Object.assign(pagination, result.pagination)
    pagination.pageSize = result.pagination?.pageSize || pagination.pageSize
  } finally {
    if (requestSequence === sessionListSequence) {
      if (append) loadingMore.value = false
      else loading.value = false
    }
  }
}

function loadMoreSessions() {
  if (loadingMore.value || sessions.value.length >= pagination.total) return
  void loadSessions(pagination.page + 1, true)
}

function toggleGroup(id: string) {
  const next = new Set(expandedGroups.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  expandedGroups.value = next
}

function groupName(id?: string | null) {
  if (!id || id === '__ungrouped') return '未分组'
  return groups.value.find((group) => group.id === id)?.name || '未知分组'
}

function createDraftSession() {
  const id = `draft-${Date.now()}`
  const draft = normalizeDraft({
    id,
    connectionId: props.connection.id,
    groupId: null,
    name: generateUniqueDraftName(),
    url: '',
  })
  tabs.value.push(createSessionTab(id, true, true, draft))
  activeTabId.value = id
}

function openSession(session: WebSocketSession) {
  const sessionId = String(session.id)
  const exists = tabs.value.find((tab) => tab.id === sessionId)
  if (exists) {
    activeTabId.value = exists.id
    return
  }
  tabs.value.push(createSessionTab(sessionId, false, false, normalizeDraft(session)))
  activeTabId.value = sessionId
}

function activateTab(id: string) {
  activeTabId.value = id
}

const isPristineNewDraft = (draft: SessionDraft): boolean => {
  if (draft.id && !String(draft.id).startsWith('draft-')) return false
  if (draft.name && !/^新建会话\s+\d+$/.test(draft.name)) return false
  if (draft.url) return false
  if (draft.headers.length > 0) return false
  if (draft.auth.type !== 'none') return false
  if (draft.protocols.length > 0) return false
  if (draft.messages.some((row) => String(row.payload || '').trim())) return false
  if (draft.settings.timeoutMs !== 5000) return false
  if (draft.settings.tlsVerify !== true) return false
  return true
}

async function closeTab(id: string) {
  const index = tabs.value.findIndex((item) => item.id === id)
  if (index < 0) return
  const tab = tabs.value[index]
  if (tab.dirty && !isPristineNewDraft(tab.draft)) {
    try {
      await ElMessageBox.confirm('当前会话有未保存改动，关闭后会丢失。', '关闭会话', {
        confirmButtonText: '放弃',
        cancelButtonText: '取消',
        type: 'warning',
      })
    } catch {
      return
    }
  }
  closeStream(tab)
  tabs.value.splice(index, 1)
  if (activeTabId.value === id) {
    activeTabId.value = tabs.value[Math.max(0, index - 1)]?.id || ''
  }
}

function markDirty() {
  if (activeTab.value) activeTab.value.dirty = true
}

async function saveActive() {
  const tab = activeTab.value
  if (!tab) return null
  saving.value = true
  try {
    const payload = buildPayload(tab.draft)
    const saved = tab.isNew
      ? await dataAPI.createWebSocketSession(props.projectId, props.connection.id, payload)
      : await dataAPI.updateWebSocketSession(props.projectId, String(tab.draft.id), payload)
    const savedId = String(saved.id)
    tab.id = savedId
    tab.isNew = false
    tab.dirty = false
    tab.draft = normalizeDraft(saved)
    activeTabId.value = savedId
    await loadSessions()
    ElMessage.success('会话已保存')
    return saved
  } finally {
    saving.value = false
  }
}

async function saveDirtyTabs(): Promise<boolean> {
  const dirtyIds = tabs.value
    .filter((tab) => tab.dirty && !isPristineNewDraft(tab.draft))
    .map((tab) => tab.id)
  for (const id of dirtyIds) {
    activeTabId.value = id
    if (!(await saveActive())) return false
  }
  return true
}

async function connectActive() {
  const tab = activeTab.value
  if (!tab) return
  if (tab.streamStatus === 'connected' || tab.streamStatus === 'connecting') {
    closeStream(tab)
    return
  }
  let saved = tab.draft as WebSocketSession
  if (tab.isNew || tab.dirty) {
    const result = await saveActive()
    if (!result) return
    saved = result
  }
  const sessionId = String(saved.id)
  beginConnect(tab)
  const epoch = tab.streamEpoch
  const timeoutMs = Math.max(1000, Math.min(30000, tab.draft.settings.timeoutMs || 5000))
  const socket = new WebSocket(buildStreamURL(props.projectId, sessionId))
  tab.socket = socket
  tab.connectTimer = setTimeout(() => {
    if (tab.streamEpoch !== epoch || tab.socket !== socket || tab.streamStatus !== 'connecting') {
      return
    }
    setStreamFailure(tab, `连接超时（${timeoutMs} ms）`)
    detachStreamSocket(tab, socket)
    socket.close()
  }, timeoutMs)
  socket.onmessage = (event) => {
    if (tab.streamEpoch !== epoch) return
    handleStreamMessage(tab, event.data)
  }
  socket.onerror = () => {
    if (tab.streamEpoch !== epoch || tab.streamStatus !== 'connecting') return
    setStreamFailure(tab, tab.streamError || 'WebSocket 连接异常')
  }
  socket.onclose = (event) => {
    if (tab.streamEpoch !== epoch) return
    clearConnectTimer(tab)
    if (tab.socket !== socket) return
    tab.socket = null
    if (tab.streamStatus === 'connecting') {
      const reason =
        tab.streamError || normalizeCloseReason(event) || `连接失败（code ${event.code || 1006}）`
      setStreamFailure(tab, reason, !tab.streamError)
    } else if (tab.streamStatus !== 'error') {
      tab.streamStatus = 'idle'
    }
  }
}

function openGroupDialog(group?: WebSocketSessionGroup) {
  groupDialog.id = group?.id ? String(group.id) : ''
  groupDialog.name = group?.name || ''
  groupDialog.parentId = group?.parentId ? String(group.parentId) : ''
  groupDialog.visible = true
}

function openChildGroupDialog(group: WebSocketSessionGroup) {
  groupDialog.id = ''
  groupDialog.name = ''
  groupDialog.parentId = String(group.id)
  groupDialog.visible = true
}

async function saveGroup(value: { name: string; parentId: string | null }) {
  groupSaving.value = true
  try {
    const payload = { name: value.name, parentId: value.parentId || null }
    let savedGroup: WebSocketSessionGroup | null = null
    if (groupDialog.id) {
      savedGroup = await dataAPI.updateWebSocketSessionGroup(
        props.projectId,
        groupDialog.id,
        payload,
      )
    } else {
      savedGroup = await dataAPI.createWebSocketSessionGroup(
        props.projectId,
        props.connection.id,
        payload,
      )
    }
    groupDialogRef.value?.closeSilently()
    groupDialog.visible = false
    await loadGroups()
    if (savedGroup?.id) {
      const next = new Set(expandedGroups.value)
      next.add(String(savedGroup.id))
      if (savedGroup.parentId) next.add(String(savedGroup.parentId))
      expandedGroups.value = next
    }
  } finally {
    groupSaving.value = false
  }
}

function createSessionTab(
  id: string,
  isNew: boolean,
  dirty: boolean,
  draft: SessionDraft,
): SessionTab {
  return {
    id,
    isNew,
    dirty,
    draft,
    socket: null,
    streamEpoch: 0,
    connectTimer: null,
    streamMessages: [],
    streamStatus: 'idle',
    streamError: '',
    messageText: firstMessagePayload(draft.messages),
    messageMode: inferMessageMode(firstMessagePayload(draft.messages)),
  }
}

function clearConnectTimer(tab: SessionTab) {
  if (tab.connectTimer) {
    clearTimeout(tab.connectTimer)
    tab.connectTimer = null
  }
}

function detachStreamSocket(tab: SessionTab, socket: WebSocket) {
  if (tab.socket !== socket) return
  tab.socket = null
  socket.onopen = null
  socket.onmessage = null
  socket.onerror = null
  socket.onclose = null
}

function beginConnect(tab: SessionTab) {
  tab.streamEpoch += 1
  clearConnectTimer(tab)
  if (tab.socket) {
    const stale = tab.socket
    detachStreamSocket(tab, stale)
    stale.close()
  }
  tab.streamMessages = []
  tab.streamError = ''
  tab.streamStatus = 'connecting'
}

function closeStream(tab: SessionTab) {
  tab.streamEpoch += 1
  clearConnectTimer(tab)
  if (tab.socket) {
    const stale = tab.socket
    detachStreamSocket(tab, stale)
    stale.close()
  }
  if (tab.streamStatus === 'connected' || tab.streamStatus === 'connecting') {
    tab.streamStatus = 'idle'
  }
}

function buildStreamURL(projectId: string, sessionId: string) {
  const base = `/api/v1/data/projects/${projectId}/websocket/sessions/${sessionId}/stream`
  const url = new URL(base, window.location.origin)
  url.protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return url.toString()
}

function handleStreamMessage(tab: SessionTab, rawData: string) {
  try {
    const envelope = JSON.parse(rawData) as {
      type?: string
      status?: string
      message?: string
      data?: WebSocketPreviewMessage
    }
    if (envelope.type === 'message' && envelope.data) {
      tab.streamMessages.push(envelope.data)
      return
    }
    if (envelope.type === 'error') {
      setStreamFailure(tab, envelope.message || 'WebSocket 连接失败')
      return
    }
    if (envelope.type === 'status' && envelope.status === 'connected') {
      clearConnectTimer(tab)
      tab.streamStatus = 'connected'
      tab.streamError = ''
    }
  } catch {
    tab.streamMessages.push({
      direction: 'in',
      type: 'text',
      payload: rawData,
      rawPayload: rawData,
      sizeBytes: rawData.length,
      timestamp: new Date().toISOString(),
    } as WebSocketPreviewMessage)
  }
}

function handleMessageEditorChange(value: string) {
  const tab = activeTab.value
  if (!tab) return
  tab.messageText = value
  syncDraftMessage(tab)
  markDirty()
}

function sendActiveMessage() {
  const tab = activeTab.value
  if (!tab || tab.streamStatus !== 'connected' || !tab.socket) {
    ElMessage.warning('请先连接 WebSocket')
    return
  }
  const payload = tab.messageText.trim()
  if (!payload) return
  try {
    tab.socket.send(payload)
  } catch (error) {
    setStreamFailure(tab, error instanceof Error ? error.message : 'WebSocket 发送消息失败')
  }
}

function setStreamFailure(tab: SessionTab, message: string, notify = true) {
  clearConnectTimer(tab)
  const reason = message.trim() || 'WebSocket 连接失败'
  const shouldNotify = notify && (tab.streamStatus !== 'error' || tab.streamError !== reason)
  tab.streamStatus = 'error'
  tab.streamError = reason
  if (tab.socket) {
    const stale = tab.socket
    detachStreamSocket(tab, stale)
    stale.close()
  }
  if (shouldNotify) ElMessage.error(reason)
}

function normalizeCloseReason(event: CloseEvent) {
  const reason = event.reason?.trim()
  if (reason) return reason
  if (event.code === 1000) return ''
  if (event.code === 1006) return '连接异常关闭'
  if (event.code === 1008) return '连接被拒绝'
  if (event.code === 1011) return '服务端内部错误'
  return ''
}

function streamStatusText(tab: SessionTab) {
  if (tab.streamStatus === 'connecting') return '连接中'
  if (tab.streamStatus === 'connected') return `监听中 · ${tab.streamMessages.length} 条`
  if (tab.streamStatus === 'error') return tab.streamError || '连接异常'
  return '未连接'
}

function duplicateSession(session: WebSocketSession) {
  const draft = normalizeDraft(session)
  draft.id = undefined
  draft.name = `${draft.name} Copy`
  tabs.value.push(createSessionTab(`copy-${Date.now()}`, true, true, draft))
  activeTabId.value = tabs.value[tabs.value.length - 1].id
}

function openSessionMenu(event: MouseEvent, session: WebSocketSession) {
  contextMenu.value = {
    visible: true,
    type: 'session',
    x: Math.min(event.clientX, window.innerWidth - 180),
    y: Math.min(event.clientY, window.innerHeight - 132),
    session,
    group: null,
  }
}

function openGroupMenu(event: MouseEvent, group: WebSocketSessionGroupNode) {
  contextMenu.value = {
    visible: true,
    type: 'group',
    x: Math.min(event.clientX, window.innerWidth - 180),
    y: Math.min(event.clientY, window.innerHeight - 168),
    session: null,
    group,
  }
}

function closeContextMenu() {
  contextMenu.value.visible = false
}

function emitContextAction(
  action: 'open' | 'duplicate' | 'move' | 'delete' | 'create-child' | 'edit',
) {
  const { type, session, group } = contextMenu.value
  closeContextMenu()
  if (type === 'session' && session) {
    if (action === 'open') {
      openSession(session)
      return
    }
    if (action === 'duplicate') {
      duplicateSession(session)
      return
    }
    if (action === 'move') {
      openMoveSessionDialog(session)
      return
    }
    if (action === 'delete') {
      void deleteSession(String(session.id))
    }
    return
  }
  if (type === 'group' && group) {
    if (action === 'create-child') {
      openChildGroupDialog(group)
      return
    }
    if (action === 'edit') {
      openGroupDialog(group)
      return
    }
    if (action === 'move') {
      openMoveGroupDialog(group)
      return
    }
    if (action === 'delete') {
      void deleteGroup(group)
    }
  }
}

const openMoveSessionDialog = (session: WebSocketSession) => {
  moveTargetType.value = 'session'
  movingSession.value = session
  movingGroup.value = null
  moveTargetGroupId.value = session.groupId ? String(session.groupId) : null
  moveDialogVisible.value = true
}

const openMoveGroupDialog = (group: WebSocketSessionGroupNode) => {
  moveTargetType.value = 'group'
  movingSession.value = null
  movingGroup.value = group
  moveTargetGroupId.value = group.parentId ? String(group.parentId) : null
  moveDialogVisible.value = true
}

const moveTarget = async () => {
  if (moveTargetType.value === 'group') {
    await moveGroup()
    return
  }
  await moveSession()
}

const moveSession = async () => {
  const session = movingSession.value
  if (!session || !canMoveTarget.value) return
  const sessionId = String(session.id)
  const openTab = tabs.value.find((tab) => tab.draft.id === sessionId)
  if (openTab?.dirty) {
    openTab.draft.groupId = moveTargetGroupId.value || null
    activeTabId.value = openTab.id
    moveDialogVisible.value = false
    ElMessage.info('已在未保存的会话中调整分组，保存会话后生效')
    return
  }

  moveSaving.value = true
  try {
    const draft = normalizeDraft(session)
    draft.groupId = moveTargetGroupId.value || null
    const saved = await dataAPI.updateWebSocketSession(
      props.projectId,
      sessionId,
      buildPayload(draft),
    )
    if (openTab) {
      openTab.draft = normalizeDraft(saved)
      openTab.dirty = false
    }
    moveDialogVisible.value = false
    movingSession.value = null
    await loadSessions()
    ElMessage.success('会话已移动')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '移动会话失败'))
  } finally {
    moveSaving.value = false
  }
}

const moveGroup = async () => {
  const group = movingGroup.value
  if (!group || !canMoveTarget.value) return
  moveSaving.value = true
  try {
    await dataAPI.updateWebSocketSessionGroup(props.projectId, String(group.id), {
      name: group.name,
      parentId: moveTargetGroupId.value || null,
    })
    moveDialogVisible.value = false
    movingGroup.value = null
    await loadGroups()
    ElMessage.success('分组已移动')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '移动分组失败'))
  } finally {
    moveSaving.value = false
  }
}

async function deleteSession(sessionId: string) {
  try {
    await ElMessageBox.confirm('删除会话后对应数据点会标记失效，确认删除？', '删除会话', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
    })
    const tab = tabs.value.find((item) => item.draft.id === sessionId)
    if (tab) closeStream(tab)
    await dataAPI.deleteWebSocketSession(props.projectId, sessionId)
    tabs.value = tabs.value.filter((item) => item.draft.id !== sessionId)
    if (!tabs.value.some((item) => item.id === activeTabId.value)) {
      activeTabId.value = tabs.value[0]?.id || ''
    }
    await loadSessions()
    ElMessage.success('会话已删除')
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(getApiErrorMessage(error, '删除会话失败'))
    }
  }
}

async function deleteGroup(group: WebSocketSessionGroupNode) {
  try {
    await ElMessageBox.confirm(
      `确认删除分组「${group.name}」？组内会话会回到根目录。`,
      '删除分组',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning',
      },
    )
    await dataAPI.deleteWebSocketSessionGroup(props.projectId, String(group.id))
    await Promise.all([loadGroups(), loadSessions()])
    ElMessage.success('分组已删除')
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(getApiErrorMessage(error, '删除分组失败'))
    }
  }
}

function normalizeDraft(input: Partial<WebSocketSession>): SessionDraft {
  return {
    id: input.id ? String(input.id) : undefined,
    projectId: input.projectId ? String(input.projectId) : undefined,
    connectionId: input.connectionId ? String(input.connectionId) : props.connection.id,
    groupId: input.groupId ? String(input.groupId) : null,
    name: input.name || '未命名会话',
    url: input.url || '',
    headers: Array.isArray(input.headers) ? [...input.headers] : [],
    auth: normalizeAuthDraft(input.auth),
    protocols: Array.isArray(input.protocols) ? [...input.protocols] : [],
    messages: Array.isArray(input.messages) ? [...input.messages] : [],
    settings: normalizeSettingsDraft(input.settings),
    enabled: true,
    sortOrder: input.sortOrder || 0,
    sourceType: input.sourceType,
    dataPointId: input.dataPointId ? String(input.dataPointId) : '',
    dataPointPath: input.dataPointPath || '',
    outputs: input.outputs?.length
      ? input.outputs.map((output) => ({ ...output, selector: { ...output.selector } }))
      : [createWholeSourceOutput('message', '完整消息', 'object')],
    lastMessage: input.lastMessage,
    lastDiagnostic: input.lastDiagnostic || '',
    quality: input.quality || 'unknown',
    lastMessageAt: input.lastMessageAt,
  }
}

function firstMessagePayload(messages: WebSocketSubscribeMessageRow[]) {
  return messages[0]?.payload || ''
}

function inferMessageMode(payload: string): 'json' | 'text' {
  const text = payload.trim()
  if (!text) return 'json'
  if (!(text.startsWith('{') || text.startsWith('['))) return 'text'
  try {
    JSON.parse(text)
    return 'json'
  } catch {
    return 'text'
  }
}

function syncDraftMessage(tab: SessionTab) {
  const existing = tab.draft.messages[0]
  tab.draft.messages = [
    {
      enabled: true,
      name: existing?.name || '发送消息',
      description: existing?.description || '',
      payload: tab.messageText,
    },
  ]
}

function replaceURLScheme(rawUrl: string, scheme: string) {
  const url = rawUrl.trim()
  if (/^[a-z][a-z\d+\-.]*:\/\//i.test(url)) {
    return url.replace(/^[a-z][a-z\d+\-.]*:\/\//i, `${scheme}://`)
  }
  return `${scheme}://${url}`
}

function normalizeAuthDraft(value: unknown): WebSocketAuthDraft {
  const input = value && typeof value === 'object' ? (value as Record<string, unknown>) : {}
  return {
    ...input,
    type: typeof input.type === 'string' ? input.type : 'none',
    token: typeof input.token === 'string' ? input.token : '',
    username: typeof input.username === 'string' ? input.username : '',
    password: typeof input.password === 'string' ? input.password : '',
  }
}

function normalizeSettingsDraft(value: unknown): WebSocketSettingsDraft {
  const input = value && typeof value === 'object' ? (value as Record<string, unknown>) : {}
  return {
    ...input,
    timeoutMs: Number(input.timeoutMs) || 5000,
    tlsVerify: typeof input.tlsVerify === 'boolean' ? input.tlsVerify : true,
  }
}

function buildPayload(draft: SessionDraft) {
  return {
    groupId: draft.groupId || null,
    name: draft.name,
    url: draft.url,
    headers: draft.headers,
    auth: draft.auth,
    protocols: draft.protocols,
    messages: draft.messages,
    settings: draft.settings,
    enabled: draft.enabled,
    sortOrder: draft.sortOrder,
    outputs: draft.outputs,
  }
}

function formatJSON(value: unknown) {
  if (typeof value === 'string') return value
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return String(value)
  }
}

function generateUniqueDraftName(): string {
  const taken = new Set<string>()
  for (const session of sessions.value) {
    if (session.name) taken.add(session.name)
  }
  for (const tab of tabs.value) {
    if (tab.draft.name) taken.add(tab.draft.name)
  }
  let maxIndex = 0
  for (const name of taken) {
    const match = /^新建会话\s+(\d+)$/.exec(name)
    if (match) {
      const n = Number(match[1])
      if (Number.isFinite(n) && n > maxIndex) maxIndex = n
    }
  }
  let next = maxIndex + 1
  while (taken.has(`新建会话 ${next}`)) next += 1
  return `新建会话 ${next}`
}

function buildWebSocketSessionTree(
  groupList: WebSocketSessionGroup[],
  sessionList: WebSocketSession[],
): { groups: WebSocketSessionGroupNode[]; rootSessions: WebSocketSession[] } {
  const nodes = new Map<string, WebSocketSessionGroupNode>()
  groupList.forEach((group) => {
    const id = String(group.id)
    nodes.set(id, {
      ...group,
      id,
      parentId: group.parentId ? String(group.parentId) : null,
      children: [],
      sessions: [],
    })
  })

  const roots: WebSocketSessionGroupNode[] = []
  nodes.forEach((node) => {
    const parentId = node.parentId ? String(node.parentId) : ''
    const parent = parentId ? nodes.get(parentId) : null
    if (parent) parent.children.push(node)
    else roots.push(node)
  })

  const rootSessions: WebSocketSession[] = []
  sessionList.forEach((session) => {
    const groupId = session.groupId ? String(session.groupId) : ''
    const group = groupId ? nodes.get(groupId) : null
    if (group) group.sessions.push(session)
    else rootSessions.push(session)
  })

  const sortSessions = (items: WebSocketSession[]) =>
    items.sort(
      (left, right) =>
        (left.sortOrder || 0) - (right.sortOrder || 0) || left.name.localeCompare(right.name),
    )
  const sortGroups = (items: WebSocketSessionGroupNode[]) => {
    items.sort(
      (left, right) =>
        (left.sortOrder || 0) - (right.sortOrder || 0) || left.name.localeCompare(right.name),
    )
    items.forEach((item) => {
      sortGroups(item.children)
      sortSessions(item.sessions)
    })
  }

  sortGroups(roots)
  sortSessions(rootSessions)
  return { groups: roots, rootSessions }
}

function flattenWebSocketGroups(
  groupList: WebSocketSessionGroup[],
  blockedIds: Set<string> = new Set(),
): Array<{ id: string; label: string }> {
  const roots = buildWebSocketSessionTree(groupList, []).groups
  const visit = (
    group: WebSocketSessionGroupNode,
    depth: number,
  ): Array<{ id: string; label: string }> => {
    const id = String(group.id)
    const children = group.children.flatMap((child) => visit(child, depth + 1))
    if (blockedIds.has(id)) return children
    return [{ id, label: `${'　'.repeat(depth)}${group.name}` }, ...children]
  }
  return roots.flatMap((group) => visit(group, 0))
}

function collectWebSocketGroupIds(group?: WebSocketSessionGroupNode | null) {
  const result = new Set<string>()
  const visit = (node?: WebSocketSessionGroupNode | null) => {
    if (!node) return
    result.add(String(node.id))
    node.children.forEach(visit)
  }
  visit(group)
  return result
}

function findWebSocketGroupNode(groupId?: string | null) {
  if (!groupId) return null
  const stack = [...sessionTree.value.groups]
  while (stack.length) {
    const group = stack.shift()
    if (!group) continue
    if (String(group.id) === String(groupId)) return group
    stack.push(...group.children)
  }
  return null
}

function countWebSocketGroupSessions(node: WebSocketSessionGroupNode): number {
  return (
    node.sessions.length +
    node.children.reduce((sum, child) => sum + countWebSocketGroupSessions(child), 0)
  )
}

const WebSocketTreeNode = defineComponent({
  name: 'WebSocketTreeNode',
  props: {
    node: { type: Object as PropType<WebSocketSessionGroupNode>, required: true },
    activeTabId: { type: String, default: '' },
    expandedGroups: { type: Object as PropType<Set<string>>, required: true },
  },
  emits: ['toggle', 'open-session', 'session-contextmenu', 'group-contextmenu'],
  setup(componentProps, { emit }) {
    const isExpanded = (id: string) => componentProps.expandedGroups.has(id)
    const renderGroup = (node: WebSocketSessionGroupNode) =>
      h('section', { class: 'ws-workbench__group' }, [
        h(
          'button',
          {
            type: 'button',
            class: 'ws-workbench__group-head',
            onClick: () => emit('toggle', node.id),
            onContextmenu: (event: MouseEvent) => {
              event.preventDefault()
              event.stopPropagation()
              emit('group-contextmenu', event, node)
            },
          },
          [
            h(IconTablerChevronRight, { class: { 'is-open': isExpanded(node.id) } }),
            h(IconTablerFolder),
            h('span', { class: 'ws-workbench__group-name' }, node.name),
            h('small', countWebSocketGroupSessions(node)),
          ],
        ),
        isExpanded(node.id)
          ? h('div', { class: 'ws-workbench__group-list' }, [
              ...node.children.map(renderGroup),
              ...node.sessions.map((session) =>
                h(
                  'button',
                  {
                    type: 'button',
                    class: [
                      'ws-workbench__session-node',
                      { 'is-active': componentProps.activeTabId === String(session.id) },
                    ],
                    onClick: () => emit('open-session', session),
                    onContextmenu: (event: MouseEvent) => {
                      event.preventDefault()
                      event.stopPropagation()
                      emit('session-contextmenu', event, session)
                    },
                  },
                  [
                    h('span', { class: 'ws-workbench__badge' }, 'WS'),
                    h('span', { class: 'ws-workbench__session-name' }, session.name),
                    session.quality !== 'unknown'
                      ? h(
                          'span',
                          { class: `ws-workbench__quality is-${session.quality}` },
                          session.quality,
                        )
                      : null,
                  ],
                ),
              ),
            ])
          : null,
      ])

    return () => renderGroup(componentProps.node as WebSocketSessionGroupNode)
  },
})

const WebSocketKeyValueEditor = defineComponent({
  props: { modelValue: { type: Array, default: () => [] } },
  emits: ['update:modelValue', 'change'],
  setup(componentProps, { emit }) {
    const rows = computed<WebSocketKeyValueRow[]>({
      get: () => componentProps.modelValue as WebSocketKeyValueRow[],
      set: (value) => {
        emit('update:modelValue', value)
        emit('change')
      },
    })
    const updateRow = (index: number, patch: Partial<WebSocketKeyValueRow>) => {
      rows.value = rows.value.map((row, rowIndex) =>
        rowIndex === index ? { ...row, ...patch } : row,
      )
    }
    const add = () => {
      rows.value = [...rows.value, { enabled: true, key: '', value: '', description: '' }]
    }
    const remove = (index: number) => {
      rows.value = rows.value.filter((_, i) => i !== index)
    }
    return () =>
      h('div', { class: 'ws-workbench__table-editor' }, [
        h('div', { class: 'ws-workbench__table-toolbar' }, [
          h(ElButton, { size: 'small', onClick: add }, () => '添加'),
        ]),
        h(
          ElTable,
          { data: rows.value, size: 'small', border: true, class: 'ws-workbench__edit-table' },
          () => [
            h(ElTableColumn, {
              label: '启用',
              width: 70,
              align: 'center',
              formatter: (
                _row: WebSocketKeyValueRow,
                _column: unknown,
                _cell: unknown,
                index: number,
              ) =>
                h(ElCheckbox, {
                  modelValue: rows.value[index]?.enabled ?? true,
                  'onUpdate:modelValue': (value: boolean) => updateRow(index, { enabled: value }),
                }),
            }),
            h(ElTableColumn, {
              label: 'Key',
              minWidth: 180,
              formatter: (
                _row: WebSocketKeyValueRow,
                _column: unknown,
                _cell: unknown,
                index: number,
              ) =>
                h(ElInput, {
                  modelValue: rows.value[index]?.key || '',
                  size: 'small',
                  placeholder: 'Header',
                  'onUpdate:modelValue': (value: string) => updateRow(index, { key: value }),
                }),
            }),
            h(ElTableColumn, {
              label: 'Value',
              minWidth: 260,
              formatter: (
                _row: WebSocketKeyValueRow,
                _column: unknown,
                _cell: unknown,
                index: number,
              ) =>
                h(ElInput, {
                  modelValue: rows.value[index]?.value || '',
                  size: 'small',
                  placeholder: 'Value',
                  'onUpdate:modelValue': (value: string) => updateRow(index, { value }),
                }),
            }),
            h(ElTableColumn, {
              label: '操作',
              width: 90,
              align: 'center',
              formatter: (
                _row: WebSocketKeyValueRow,
                _column: unknown,
                _cell: unknown,
                index: number,
              ) =>
                h(
                  ElButton,
                  { size: 'small', text: true, type: 'danger', onClick: () => remove(index) },
                  () => '删除',
                ),
            }),
          ],
        ),
      ])
  },
})

const WebSocketProtocolEditor = defineComponent({
  props: { modelValue: { type: Array, default: () => [] } },
  emits: ['update:modelValue', 'change'],
  setup(componentProps, { emit }) {
    const rows = computed<WebSocketProtocolRow[]>({
      get: () => componentProps.modelValue as WebSocketProtocolRow[],
      set: (value) => {
        emit('update:modelValue', value)
        emit('change')
      },
    })
    const updateRow = (index: number, patch: Partial<WebSocketProtocolRow>) => {
      rows.value = rows.value.map((row, rowIndex) =>
        rowIndex === index ? { ...row, ...patch } : row,
      )
    }
    const add = () => {
      rows.value = [...rows.value, { enabled: true, value: '', description: '' }]
    }
    const remove = (index: number) => {
      rows.value = rows.value.filter((_, i) => i !== index)
    }
    return () =>
      h('div', { class: 'ws-workbench__table-editor' }, [
        h('div', { class: 'ws-workbench__table-toolbar' }, [
          h(ElButton, { size: 'small', onClick: add }, () => '添加'),
        ]),
        h(
          ElTable,
          { data: rows.value, size: 'small', border: true, class: 'ws-workbench__edit-table' },
          () => [
            h(ElTableColumn, {
              label: '启用',
              width: 70,
              align: 'center',
              formatter: (
                _row: WebSocketProtocolRow,
                _column: unknown,
                _cell: unknown,
                index: number,
              ) =>
                h(ElCheckbox, {
                  modelValue: rows.value[index]?.enabled ?? true,
                  'onUpdate:modelValue': (value: boolean) => updateRow(index, { enabled: value }),
                }),
            }),
            h(ElTableColumn, {
              label: '子协议',
              minWidth: 260,
              formatter: (
                _row: WebSocketProtocolRow,
                _column: unknown,
                _cell: unknown,
                index: number,
              ) =>
                h(ElInput, {
                  modelValue: rows.value[index]?.value || '',
                  size: 'small',
                  placeholder: 'chat.v1',
                  'onUpdate:modelValue': (value: string) => updateRow(index, { value }),
                }),
            }),
            h(ElTableColumn, {
              label: '操作',
              width: 90,
              align: 'center',
              formatter: (
                _row: WebSocketProtocolRow,
                _column: unknown,
                _cell: unknown,
                index: number,
              ) =>
                h(
                  ElButton,
                  { size: 'small', text: true, type: 'danger', onClick: () => remove(index) },
                  () => '删除',
                ),
            }),
          ],
        ),
      ])
  },
})
</script>

<style scoped>
.ws-workbench {
  height: 100%;
  min-height: 0;
  display: grid;
  grid-template-columns: 300px minmax(0, 1fr);
  background: #f6f7f9;
  color: #1f2937;
}

.ws-workbench__sidebar {
  min-width: 0;
  border-right: 1px solid #dfe3ea;
  background: #fff;
  display: flex;
  flex-direction: column;
}

.ws-workbench__search {
  width: 100%;
}

.ws-workbench__tree {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 8px;
}

.ws-workbench__load-more {
  width: 100%;
  min-height: 30px;
  margin-top: 6px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
  color: var(--dc-primary);
  cursor: pointer;
  font: inherit;
  font-size: 12px;
}

.ws-workbench__load-more:hover:not(:disabled) {
  border-color: var(--dc-primary);
  background: var(--dc-primary-soft);
}

.ws-workbench__load-more:disabled {
  cursor: wait;
  opacity: 0.65;
}

.ws-workbench__group-head,
.ws-workbench__session-node,
.ws-workbench__tab {
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
}

.ws-workbench__tree :deep(.ws-workbench__group) {
  min-width: 0;
  display: grid;
  gap: 2px;
}

.ws-workbench__tree :deep(.ws-workbench__group-list) {
  display: grid;
  gap: 2px;
  margin-left: 10px;
  padding-left: 6px;
}

.ws-workbench__tree :deep(.ws-workbench__group-head),
.ws-workbench__tree :deep(.ws-workbench__session-node) {
  width: 100%;
  border: 1px solid transparent;
  border-radius: 6px;
  background: transparent;
  color: inherit;
  text-align: left;
}

.ws-workbench__tree :deep(.ws-workbench__group-head) {
  min-height: 28px;
  display: grid;
  grid-template-columns: 16px 18px minmax(0, 1fr) auto;
  align-items: center;
  gap: 5px;
  padding: 0 6px;
  font-size: 13px;
  font-weight: 700;
  cursor: pointer;
}

.ws-workbench__tree :deep(.ws-workbench__session-node) {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
  min-height: 30px;
  padding: 3px 6px;
  font-size: 12px;
  cursor: pointer;
}

.ws-workbench__tree :deep(.ws-workbench__group-head:hover),
.ws-workbench__tree :deep(.ws-workbench__session-node:hover) {
  border-color: #dfe3ea;
  background: #eef5ff;
  color: #1f2937;
}

.ws-workbench__tree :deep(.ws-workbench__session-node.is-active) {
  background: #e9f5f3;
  color: #035f59;
}

.ws-workbench__tree :deep(.ws-workbench__group-head svg) {
  width: 16px;
  height: 16px;
}

.ws-workbench__tree :deep(.ws-workbench__group-head svg:first-child) {
  width: 14px;
  height: 14px;
  color: #94a3b8;
  transition: transform 0.16s ease;
}

.ws-workbench__tree :deep(.ws-workbench__group-head svg:first-child.is-open) {
  transform: rotate(90deg);
}

.ws-workbench__tree :deep(.ws-workbench__group-head svg:nth-child(2)) {
  color: #04756f;
}

.ws-workbench__tree :deep(.ws-workbench__group-head small) {
  min-width: 20px;
  padding: 2px 6px;
  border-radius: 999px;
  background: #f1f5f9;
  color: #64748b;
  font-size: 11px;
  text-align: center;
}

.ws-workbench__tree :deep(.ws-workbench__group-name),
.ws-workbench__tree :deep(.ws-workbench__session-name) {
  min-width: 0;
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #1f2937;
  font-size: 13px;
  font-weight: 700;
}

.ws-workbench__tree > .ws-workbench__session-node {
  width: 100%;
  border: 1px solid transparent;
  border-radius: 6px;
  background: transparent;
  color: inherit;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
  min-height: 30px;
  padding: 3px 6px;
  font-size: 12px;
  cursor: pointer;
  text-align: left;
}

.ws-workbench__tree > .ws-workbench__session-node:hover {
  border-color: #dfe3ea;
  background: #eef5ff;
  color: #1f2937;
}

.ws-workbench__tree > .ws-workbench__session-node.is-active {
  background: #e9f5f3;
  color: #035f59;
}

.ws-workbench__group-name,
.ws-workbench__session-name,
.ws-workbench__tab-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ws-workbench__badge,
.ws-workbench__method {
  height: 22px;
  min-width: 34px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  background: #e9f5f3;
  color: #04756f;
  font-size: 12px;
  font-weight: 700;
}

.ws-workbench__quality {
  height: 22px;
  padding: 0 7px;
  display: inline-flex;
  align-items: center;
  border-radius: 999px;
  font-size: 12px;
  background: #f1f5f9;
  color: #64748b;
}

.ws-workbench__quality.is-good {
  background: #dcfce7;
  color: #166534;
}

.ws-workbench__quality.is-bad {
  background: #fee2e2;
  color: #991b1b;
}

.ws-workbench__pager {
  padding: 8px;
  border-top: 1px solid #eef1f5;
}

.ws-workbench__main {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.ws-workbench__tabs {
  height: 36px;
  display: flex;
  overflow-x: auto;
  border-bottom: 1px solid #dfe3ea;
  background: #fff;
}

.ws-workbench__tab {
  min-width: 160px;
  max-width: 240px;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto auto;
  align-items: center;
  gap: 8px;
  padding: 0 10px;
  border-right: 1px solid #eef1f5;
  cursor: pointer;
}

.ws-workbench__tab.is-active {
  background: #f8fafc;
  border-top: 2px solid #04756f;
}

.ws-workbench__tab i {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #f59e0b;
}

.ws-workbench__tab svg {
  width: 14px;
  height: 14px;
  color: #94a3b8;
}

.ws-workbench__editor {
  min-height: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.ws-workbench__crumb-row,
.ws-workbench__request-line {
  min-height: 44px;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 14px;
  background: #fff;
  border-bottom: 1px solid #eef1f5;
}

.ws-workbench__crumb {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 6px;
  color: #64748b;
}

.ws-workbench__crumb svg {
  width: 14px;
  height: 14px;
  color: #94a3b8;
}

.ws-workbench__name-input {
  width: min(320px, 42vw);
}

.ws-workbench__name-input :deep(.el-input__wrapper) {
  background: transparent;
  box-shadow: none;
  padding: 0 4px;
}

.ws-workbench__name-input :deep(.el-input__wrapper:hover),
.ws-workbench__name-input :deep(.el-input__wrapper.is-focus) {
  background: #f8fafc;
  box-shadow: 0 0 0 1px #d8dee8 inset;
}

.ws-workbench__actions {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 8px;
}

.ws-workbench__datapoint-path {
  max-width: min(360px, 34vw);
  color: #64748b;
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ws-workbench__icon-btn {
  width: 32px;
  padding: 0;
}

.ws-workbench__icon-btn :deep(.el-icon),
.ws-workbench__icon-btn :deep(svg) {
  width: 16px;
  height: 16px;
}

.ws-workbench__request-line :deep(.el-input) {
  flex: 1;
}

.ws-workbench__protocol-select {
  width: 82px;
  flex: 0 0 82px;
}

.ws-workbench__config-tabs {
  flex: 1;
  min-height: 0;
  background: #fff;
  padding: 0 14px;
  overflow: hidden;
}

.ws-workbench__config-tabs :deep(.el-tabs__content) {
  min-height: 0;
  overflow: auto;
}

.ws-workbench__config-tabs :deep(.el-tabs__item),
.ws-workbench__response :deep(.el-tabs__item) {
  height: 36px;
  color: #64748b;
}

.ws-workbench__config-tabs :deep(.el-tabs__item.is-active) {
  color: #04756f;
}

.ws-workbench__form {
  max-width: 760px;
  padding: 12px 0;
}

.ws-workbench__form-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 14px;
}

.ws-workbench__form :deep(.el-form-item) {
  margin-bottom: 14px;
}

.ws-workbench__form :deep(.el-form-item__label) {
  height: auto;
  margin-bottom: 6px;
  color: #475569;
  font-size: 12px;
  line-height: 1.4;
}

.ws-workbench__form-control {
  width: 100%;
}

.ws-workbench__settings-checks {
  min-height: 32px;
  display: flex;
  align-items: center;
}

.ws-workbench__table-editor {
  padding: 12px 0;
}

.ws-workbench__message-composer {
  padding: 12px 0;
}

.ws-workbench__message-toolbar,
.ws-workbench__message-composer-footer {
  display: flex;
  align-items: center;
}

.ws-workbench__message-toolbar {
  justify-content: flex-start;
  margin-bottom: 8px;
}

.ws-workbench__message-composer-footer {
  justify-content: flex-end;
  margin-top: 8px;
}

.ws-workbench__message-composer-footer :deep(.el-button svg) {
  width: 14px;
  height: 14px;
}

.ws-workbench__response {
  position: relative;
  flex: 0 0 auto;
  min-height: 160px;
  border-top: 1px solid #dfe3ea;
  background: #fbfcfe;
  display: flex;
  flex-direction: column;
}

.ws-workbench__response-resizer {
  position: absolute;
  top: -4px;
  left: 0;
  z-index: 3;
  width: 100%;
  height: 8px;
  cursor: row-resize;
}

.ws-workbench__response-resizer::before {
  content: '';
  position: absolute;
  top: 3px;
  left: 50%;
  width: 52px;
  height: 2px;
  border-radius: 999px;
  background: color-mix(in oklch, #64748b 42%, transparent);
  transform: translateX(-50%);
}

.ws-workbench__response-resizer:hover::before {
  background: #04756f;
}

:global(body.ws-workbench--resizing-panel) {
  cursor: row-resize;
  user-select: none;
}

.ws-workbench__response header {
  flex: 0 0 auto;
  min-height: 42px;
  padding: 8px 14px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid #eef1f5;
}

.ws-workbench__response header > div {
  display: flex;
  align-items: center;
  gap: 10px;
}

.ws-workbench__response header span {
  color: #64748b;
  font-size: 12px;
}

.ws-workbench__response-body {
  position: relative;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.ws-workbench__error-response,
.ws-workbench__empty-response,
.ws-workbench__messages {
  position: absolute;
  inset: 0;
  overflow: auto;
}

.ws-workbench__messages {
  padding: 12px 14px;
  box-sizing: border-box;
}

.ws-workbench__message {
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  background: #fff;
  margin-bottom: 10px;
  overflow: hidden;
}

.ws-workbench__message > div {
  height: 32px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 10px;
  border-bottom: 1px solid #eef1f5;
}

.ws-workbench__message pre,
.ws-workbench__diagnostic {
  margin: 0;
  padding: 10px;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  font-family: ui-monospace, SFMono-Regular, Consolas, monospace;
  font-size: 12px;
}

.ws-workbench__empty,
.ws-workbench__empty-response,
.ws-workbench__blank,
.ws-workbench__loading {
  color: #64748b;
  display: flex;
  align-items: center;
  justify-content: center;
}

.ws-workbench__empty,
.ws-workbench__loading {
  min-height: 80px;
  gap: 8px;
}

.ws-workbench__loading svg {
  width: 16px;
  height: 16px;
  animation: ws-spin 0.9s linear infinite;
}

.ws-workbench__connect-btn.is-connected :deep(svg) {
  color: #fff;
}

.ws-workbench__connect-btn.is-error:not(.is-loading) {
  --el-button-bg-color: #fee2e2;
  --el-button-border-color: #fecaca;
  --el-button-text-color: #b91c1c;
}

.ws-workbench__error-response {
  padding: 24px 16px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: #b91c1c;
  text-align: center;
  box-sizing: border-box;
}

.ws-workbench__error-response svg {
  width: 28px;
  height: 28px;
}

.ws-workbench__error-response strong {
  color: #991b1b;
  font-size: 14px;
}

.ws-workbench__error-response span {
  max-width: min(560px, 90%);
  color: #7f1d1d;
  font-size: 13px;
  line-height: 1.5;
  word-break: break-word;
}

.ws-workbench__empty-response {
  box-sizing: border-box;
}

.ws-workbench__blank {
  flex: 1;
  flex-direction: column;
  gap: 10px;
}

.ws-workbench__blank svg {
  width: 42px;
  height: 42px;
  color: #94a3b8;
}

.ws-workbench__blank strong {
  color: #334155;
}

.ws-workbench__move-form {
  display: grid;
  gap: 2px;
}

.ws-workbench__move-select {
  width: 100%;
}

.ws-workbench__move-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.ws-workbench__menu-mask {
  position: fixed;
  inset: 0;
  z-index: 2100;
}

.ws-workbench__context-menu {
  position: fixed;
  min-width: 148px;
  padding: 4px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
}

.ws-workbench__context-menu button {
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

.ws-workbench__context-menu button:hover {
  background: var(--dc-surface-muted);
  color: var(--dc-primary);
}

.ws-workbench__context-menu button.is-danger:hover {
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
}

.ws-workbench__menu-icon {
  width: 15px;
  height: 15px;
  flex: 0 0 auto;
}

@keyframes ws-spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
