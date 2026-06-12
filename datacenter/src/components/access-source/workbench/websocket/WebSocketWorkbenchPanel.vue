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
            :class="{ 'is-active': activeTabId === session.id }"
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
            <span v-if="activeTab.draft.dataPointPath" class="ws-workbench__datapoint-path">
              数据点：{{ activeTab.draft.dataPointPath }}
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
          <el-tooltip
            :content="isActiveConnected ? '断开连接' : '连接并监听消息'"
            placement="top"
            :show-after="400"
          >
            <el-button
              type="primary"
              :loading="activeTab.streamStatus === 'connecting'"
              class="ws-workbench__icon-btn"
              :aria-label="isActiveConnected ? '断开连接' : '连接'"
              @click="connectActive"
            >
              <IconTablerPlugConnected />
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

        <section class="ws-workbench__response">
          <header>
            <div>
              <strong>消息流</strong>
              <span>{{ streamStatusText(activeTab) }}</span>
            </div>
          </header>
          <div v-if="activeTab.streamMessages.length === 0" class="ws-workbench__empty-response">
            连接后持续显示 WebSocket 消息
          </div>
          <div v-else class="ws-workbench__messages">
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
        </section>
      </div>

      <div v-else class="ws-workbench__blank">
        <IconTablerWebhook />
        <strong>选择或新建一个 WebSocket 会话</strong>
        <span>WebSocket 工作台以连接会话为中心，一个会话同步一个 object 数据点。</span>
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
          <button type="button" class="is-danger" @click="emitContextAction('delete')">
            <IconTablerTrash class="ws-workbench__menu-icon" />
            <span>{{ contextMenu.type === 'group' ? '删除分组' : '删除会话' }}</span>
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
  onBeforeUnmount,
  onMounted,
  reactive,
  ref,
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
import IconTablerChevronRight from '~icons/tabler/chevron-right'
import IconTablerDeviceFloppy from '~icons/tabler/device-floppy'
import IconTablerFolder from '~icons/tabler/folder'
import IconTablerFolderPlus from '~icons/tabler/folder-plus'
import IconTablerLoader2 from '~icons/tabler/loader-2'
import IconTablerPencil from '~icons/tabler/pencil'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerPlugConnected from '~icons/tabler/plug-connected'
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerTrash from '~icons/tabler/trash'
import IconTablerWebhook from '~icons/tabler/webhook'
import IconTablerX from '~icons/tabler/x'
import WorkbenchSourceHeader from '@/components/workbench/WorkbenchSourceHeader.vue'
import WorkbenchGroupDialog from '@/components/workbench/WorkbenchGroupDialog.vue'
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
import { Storage } from '@/utils/storage'
import { getApiErrorMessage } from '@/utils/request'

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

const loading = ref(false)
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
const expandedGroups = reactive(new Set<string>(['__ungrouped']))
const pagination = reactive({ page: 1, pageSize: 1000, total: 0, totalPages: 0 })
const tabs = ref<SessionTab[]>([])
const activeTabId = ref('')
const groupDialog = reactive({ visible: false, id: '', name: '', parentId: '' })
const groupDialogRef = ref<InstanceType<typeof WorkbenchGroupDialog> | null>(null)
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

const sourceMetaRows = computed(() => [
  { label: '类型', value: 'WebSocket' },
  { label: '连接数', value: `${pagination.total || sessions.value.length} 个` },
])
const activeTab = computed(() => tabs.value.find((tab) => tab.id === activeTabId.value))
const isActiveConnected = computed(() => activeTab.value?.streamStatus === 'connected')
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
const groupOptions = computed(() =>
  flattenWebSocketGroups(sessionTree.value.groups).map((group) => ({
    id: String(group.id),
    label: group.label,
  })),
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
})

onBeforeUnmount(() => {
  tabs.value.forEach(closeStream)
})

async function reloadWorkbench() {
  await Promise.all([loadGroups(), loadSessions(1)])
}

async function loadGroups() {
  const result = await dataAPI.getWebSocketSessionGroups(props.projectId, props.connection.id)
  groups.value = result.list
  groups.value.forEach((group) => expandedGroups.add(String(group.id)))
}

async function loadSessions(page = pagination.page) {
  loading.value = true
  try {
    const params: Record<string, any> = {
      page,
      pageSize: pagination.pageSize,
      q: search.value || undefined,
    }
    const result = await dataAPI.getWebSocketSessions(props.projectId, props.connection.id, params)
    sessions.value = result.list
    Object.assign(pagination, result.pagination)
    pagination.pageSize = result.pagination?.pageSize || 1000
  } finally {
    loading.value = false
  }
}

function toggleGroup(id: string) {
  if (expandedGroups.has(id)) expandedGroups.delete(id)
  else expandedGroups.add(id)
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
    name: '新建会话',
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

async function closeTab(id: string) {
  const tab = tabs.value.find((item) => item.id === id)
  if (!tab) return
  if (tab.dirty) {
    await ElMessageBox.confirm('当前会话有未保存改动，确认关闭？', '关闭会话', { type: 'warning' })
  }
  closeStream(tab)
  tabs.value = tabs.value.filter((item) => item.id !== id)
  if (activeTabId.value === id) activeTabId.value = tabs.value[0]?.id || ''
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
  const token = Storage.getToken()
  if (!token) {
    ElMessage.error('请先登录后再连接 WebSocket')
    return
  }

  closeStream(tab)
  tab.streamMessages = []
  tab.streamError = ''
  tab.streamStatus = 'connecting'
  const socket = new WebSocket(buildStreamURL(props.projectId, sessionId, token))
  tab.socket = socket
  socket.onopen = () => {
    tab.streamStatus = 'connected'
  }
  socket.onmessage = (event) => handleStreamMessage(tab, event.data)
  socket.onerror = () => {
    tab.streamStatus = 'error'
    tab.streamError = 'WebSocket 工作台连接异常'
  }
  socket.onclose = () => {
    if (tab.socket === socket) {
      tab.socket = null
      if (tab.streamStatus !== 'error') tab.streamStatus = 'idle'
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
    if (groupDialog.id)
      await dataAPI.updateWebSocketSessionGroup(props.projectId, groupDialog.id, payload)
    else await dataAPI.createWebSocketSessionGroup(props.projectId, props.connection.id, payload)
    groupDialogRef.value?.closeSilently()
    groupDialog.visible = false
    await loadGroups()
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
    streamMessages: [],
    streamStatus: 'idle',
    streamError: '',
    messageText: firstMessagePayload(draft.messages),
    messageMode: inferMessageMode(firstMessagePayload(draft.messages)),
  }
}

function closeStream(tab: SessionTab) {
  if (tab.socket) {
    tab.socket.close()
    tab.socket = null
  }
  if (tab.streamStatus === 'connected' || tab.streamStatus === 'connecting') {
    tab.streamStatus = 'idle'
  }
}

function buildStreamURL(projectId: string, sessionId: string, token: string) {
  const base = `/api/v1/data/projects/${projectId}/websocket/sessions/${sessionId}/stream`
  const url = new URL(base, window.location.origin)
  url.protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  url.searchParams.set('token', token)
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
      tab.streamStatus = 'error'
      tab.streamError = envelope.message || 'WebSocket 监听失败'
      return
    }
    if (envelope.type === 'status' && envelope.status === 'connected') {
      tab.streamStatus = 'connected'
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
    tab.streamStatus = 'error'
    tab.streamError = error instanceof Error ? error.message : 'WebSocket 发送消息失败'
  }
}

function streamStatusText(tab: SessionTab) {
  if (tab.streamStatus === 'connecting') return '连接中'
  if (tab.streamStatus === 'connected') return `监听中 · ${tab.streamMessages.length} 条`
  if (tab.streamStatus === 'error') return tab.streamError || '连接异常'
  return '未连接'
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

function emitContextAction(action: 'open' | 'delete' | 'create-child' | 'edit') {
  const { type, session, group } = contextMenu.value
  closeContextMenu()
  if (type === 'session' && session) {
    if (action === 'open') {
      openSession(session)
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
    if (action === 'delete') {
      void deleteGroup(group)
    }
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

  return { groups: roots, rootSessions }
}

function flattenWebSocketGroups(
  nodes: WebSocketSessionGroupNode[],
  depth = 0,
): Array<{ id: string; label: string }> {
  return nodes.flatMap((node) => [
    { id: node.id, label: `${'　'.repeat(depth)}${node.name}` },
    ...flattenWebSocketGroups(node.children, depth + 1),
  ])
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
                      { 'is-active': componentProps.activeTabId === session.id },
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

.ws-workbench__group-head,
.ws-workbench__session-node,
.ws-workbench__tab {
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
}

.ws-workbench__group-head {
  width: 100%;
  height: 32px;
  display: grid;
  grid-template-columns: 16px 18px 1fr auto;
  align-items: center;
  gap: 6px;
  padding: 0 8px;
  border-radius: 6px;
  cursor: pointer;
  text-align: left;
}

.ws-workbench__group-head:hover,
.ws-workbench__session-node:hover {
  background: #eef5ff;
}

.ws-workbench__group-head svg {
  width: 16px;
  height: 16px;
}

.ws-workbench__group-head svg:first-child {
  color: #94a3b8;
  transition: transform 0.15s ease;
}

.ws-workbench__group-head svg:first-child.is-open {
  transform: rotate(90deg);
}

.ws-workbench__group-head svg:nth-child(2) {
  color: #64748b;
}

.ws-workbench__group-head small {
  min-width: 22px;
  height: 20px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 10px;
  background: #f1f5f9;
  color: #64748b;
  font-size: 12px;
}

.ws-workbench__group-list {
  padding: 2px 0 8px 14px;
}

.ws-workbench__session-node {
  width: 100%;
  min-height: 34px;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
  padding: 0 8px;
  border-radius: 6px;
  cursor: pointer;
  text-align: left;
}

.ws-workbench__session-node.is-active {
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
  min-height: 220px;
  background: #fff;
  padding: 0 14px;
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
  height: 320px;
  min-height: 240px;
  border-top: 1px solid #dfe3ea;
  background: #fbfcfe;
  display: flex;
  flex-direction: column;
}

.ws-workbench__response header {
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

.ws-workbench__messages {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 12px 14px;
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

.ws-workbench__empty-response {
  flex: 1;
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

@keyframes ws-spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
