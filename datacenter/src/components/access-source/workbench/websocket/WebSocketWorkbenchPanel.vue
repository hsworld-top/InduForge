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
            size="small"
            clearable
            placeholder="搜索会话"
            @keyup.enter="loadSessions(1)"
          />
          <button
            class="workbench-source-header__icon-action is-primary"
            title="新建会话"
            type="button"
            @click="createDraftSession"
          >
            <IconTablerPlus />
          </button>
          <button
            class="workbench-source-header__icon-action"
            title="新建分组"
            type="button"
            @click="openGroupDialog()"
          >
            <IconTablerFolderPlus />
          </button>
          <button
            class="workbench-source-header__icon-action"
            title="刷新"
            type="button"
            @click="reloadWorkbench"
          >
            <IconTablerRefresh />
          </button>
        </template>
      </WorkbenchSourceHeader>

      <div class="ws-workbench__filter">
        <el-select v-model="activeGroupId" size="small" @change="handleGroupChange">
          <el-option label="全部会话" value="" />
          <el-option label="未分组" value="__ungrouped" />
          <el-option
            v-for="group in groups"
            :key="group.id"
            :label="group.name"
            :value="group.id"
          />
        </el-select>
      </div>

      <div class="ws-workbench__tree">
        <div v-if="loading" class="ws-workbench__empty">加载会话...</div>
        <template v-else>
          <section v-for="group in groupedSessions" :key="group.id" class="ws-workbench__group">
            <button type="button" class="ws-workbench__group-head" @click="toggleGroup(group.id)">
              <IconTablerChevronRight :class="{ 'is-open': expandedGroups.has(group.id) }" />
              <IconTablerFolder />
              <span>{{ group.name }}</span>
              <small>{{ group.sessions.length }}</small>
            </button>
            <div v-show="expandedGroups.has(group.id)" class="ws-workbench__group-list">
              <button
                v-for="session in group.sessions"
                :key="session.id"
                type="button"
                class="ws-workbench__session-node"
                :class="{ 'is-active': activeTabId === session.id }"
                @click="openSession(session)"
              >
                <span class="ws-workbench__badge">WS</span>
                <span>{{ session.name }}</span>
                <el-tag
                  v-if="session.quality !== 'unknown'"
                  size="small"
                  :type="session.quality === 'good' ? 'success' : 'danger'"
                >
                  {{ session.quality }}
                </el-tag>
              </button>
            </div>
          </section>
          <div v-if="sessions.length === 0" class="ws-workbench__empty">
            {{ search ? '没有匹配的会话' : '暂无会话' }}
          </div>
        </template>
      </div>

      <el-pagination
        v-if="pagination.total"
        class="ws-workbench__pager"
        small
        layout="prev, pager, next"
        :current-page="pagination.page"
        :page-size="pagination.pageSize"
        :total="pagination.total"
        @current-change="loadSessions"
      />
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
          <span>{{ tab.draft.name || '未命名会话' }}</span>
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
            <strong>{{ activeTab.draft.name || '未命名会话' }}</strong>
          </div>
          <div class="ws-workbench__actions">
            <el-switch v-model="activeTab.draft.enabled" size="small" @change="markDirty" />
            <el-button size="small" :loading="saving" @click="saveActive">Save</el-button>
            <el-button
              size="small"
              type="danger"
              plain
              :disabled="activeTab.isNew"
              @click="deleteActive"
              >删除</el-button
            >
          </div>
        </div>

        <div class="ws-workbench__name-row">
          <el-input v-model="activeTab.draft.name" placeholder="会话名称" @input="markDirty" />
          <el-select
            v-model="activeTab.draft.groupId"
            placeholder="分组"
            clearable
            @change="markDirty"
          >
            <el-option
              v-for="group in groups"
              :key="group.id"
              :label="group.name"
              :value="group.id"
            />
          </el-select>
        </div>

        <div class="ws-workbench__request-line">
          <span class="ws-workbench__method">CONNECT</span>
          <el-input
            v-model="activeTab.draft.url"
            placeholder="/stream 或 ws://example.com/stream"
            @input="markDirty"
          />
          <el-button type="primary" :loading="connecting" @click="connectActive">Connect</el-button>
        </div>

        <el-tabs v-model="activeConfigTab" class="ws-workbench__config-tabs">
          <el-tab-pane label="Headers" name="headers">
            <WebSocketKeyValueEditor v-model="activeTab.draft.headers" @change="markDirty" />
          </el-tab-pane>
          <el-tab-pane label="Authorization" name="auth">
            <div class="ws-workbench__auth">
              <el-select v-model="activeTab.draft.auth.type" @change="markDirty">
                <el-option label="No Auth" value="none" />
                <el-option label="Bearer Token" value="bearer" />
                <el-option label="Basic Auth" value="basic" />
              </el-select>
              <el-input
                v-if="activeTab.draft.auth.type === 'bearer'"
                v-model="activeTab.draft.auth.token"
                placeholder="Token"
                @input="markDirty"
              />
              <template v-if="activeTab.draft.auth.type === 'basic'">
                <el-input
                  v-model="activeTab.draft.auth.username"
                  placeholder="Username"
                  @input="markDirty"
                />
                <el-input
                  v-model="activeTab.draft.auth.password"
                  placeholder="Password"
                  show-password
                  @input="markDirty"
                />
              </template>
            </div>
          </el-tab-pane>
          <el-tab-pane label="Protocols" name="protocols">
            <WebSocketProtocolEditor v-model="activeTab.draft.protocols" @change="markDirty" />
          </el-tab-pane>
          <el-tab-pane label="Messages" name="messages">
            <WebSocketMessageEditor v-model="activeTab.draft.messages" @change="markDirty" />
          </el-tab-pane>
          <el-tab-pane label="Settings" name="settings">
            <div class="ws-workbench__settings">
              <label>超时 ms</label>
              <el-input-number
                v-model="activeTab.draft.settings.timeoutMs"
                :min="1000"
                :max="30000"
                :step="500"
                @change="markDirty"
              />
              <label>读取条数</label>
              <el-input-number
                v-model="activeTab.draft.settings.messageLimit"
                :min="1"
                :max="50"
                @change="markDirty"
              />
              <el-checkbox v-model="activeTab.draft.settings.tlsVerify" @change="markDirty"
                >校验 TLS 证书</el-checkbox
              >
            </div>
          </el-tab-pane>
        </el-tabs>

        <section class="ws-workbench__response">
          <header>
            <div>
              <strong>消息流</strong>
              <span v-if="activeTab.preview"
                >{{ activeTab.preview.status }} · {{ activeTab.preview.durationMs }} ms</span
              >
            </div>
            <el-tag v-if="activeTab.draft.dataPointPath" size="small">{{
              activeTab.draft.dataPointPath
            }}</el-tag>
          </header>
          <div v-if="!activeTab.preview" class="ws-workbench__empty-response">
            短连接预览后显示收发消息
          </div>
          <div v-else class="ws-workbench__messages">
            <article
              v-for="(message, index) in activeTab.preview.messages"
              :key="index"
              class="ws-workbench__message"
              :class="`is-${message.direction}`"
            >
              <div>
                <el-tag size="small" :type="message.direction === 'in' ? 'success' : 'info'">{{
                  message.direction
                }}</el-tag>
                <span>{{ message.type }}</span>
                <small>{{ message.sizeBytes }} bytes</small>
              </div>
              <pre>{{ formatJSON(message.payload ?? message.rawPayload) }}</pre>
            </article>
            <pre v-if="activeTab.preview.status === 'error'" class="ws-workbench__diagnostic">{{
              formatJSON(activeTab.preview.diagnostics)
            }}</pre>
          </div>
        </section>
      </div>

      <div v-else class="ws-workbench__blank">
        <IconTablerWebhook />
        <strong>选择或新建一个 WebSocket 会话</strong>
      </div>
    </main>

    <el-dialog
      v-model="groupDialog.visible"
      :title="groupDialog.id ? '编辑分组' : '新建分组'"
      width="420px"
    >
      <el-form label-width="80px">
        <el-form-item label="名称">
          <el-input v-model="groupDialog.name" placeholder="分组名称" />
        </el-form-item>
        <el-form-item label="上级">
          <el-select v-model="groupDialog.parentId" clearable placeholder="根集合">
            <el-option
              v-for="group in groups"
              :key="group.id"
              :label="group.name"
              :value="group.id"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="groupDialog.visible = false">取消</el-button>
        <el-button type="primary" @click="saveGroup">保存</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import IconTablerChevronRight from '~icons/tabler/chevron-right'
import IconTablerFolder from '~icons/tabler/folder'
import IconTablerFolderPlus from '~icons/tabler/folder-plus'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerWebhook from '~icons/tabler/webhook'
import IconTablerX from '~icons/tabler/x'
import WorkbenchSourceHeader from '@/components/workbench/WorkbenchSourceHeader.vue'
import * as dataAPI from '@/api/data.api'
import type {
  WebSocketKeyValueRow,
  WebSocketPreviewResponse,
  WebSocketProtocolRow,
  WebSocketSession,
  WebSocketSessionGroup,
  WebSocketSubscribeMessageRow,
} from '@/api/schemas/websocket-workbench.schema'

type AccessSourceConnection = {
  id: string
  name?: string
  type?: string
  config?: Record<string, any>
}

type SessionDraft = Omit<WebSocketSession, 'id' | 'createdAt' | 'updatedAt'> & { id?: string }
type SessionTab = {
  id: string
  isNew: boolean
  dirty: boolean
  draft: SessionDraft
  preview?: WebSocketPreviewResponse
}

const props = defineProps<{ connection: AccessSourceConnection; projectId: string }>()
defineEmits<{ (event: 'back'): void }>()

const loading = ref(false)
const saving = ref(false)
const connecting = ref(false)
const search = ref('')
const activeGroupId = ref('')
const activeConfigTab = ref('headers')
const groups = ref<WebSocketSessionGroup[]>([])
const sessions = ref<WebSocketSession[]>([])
const expandedGroups = reactive(new Set<string>(['__ungrouped']))
const pagination = reactive({ page: 1, pageSize: 20, total: 0, totalPages: 0 })
const tabs = ref<SessionTab[]>([])
const activeTabId = ref('')
const groupDialog = reactive({ visible: false, id: '', name: '', parentId: '' })

const sourceMetaRows = computed(() => [
  { label: '类型', value: 'WebSocket' },
  { label: '地址', value: props.connection.config?.url || '未配置' },
])
const activeTab = computed(() => tabs.value.find((tab) => tab.id === activeTabId.value))
const groupedSessions = computed(() => {
  const buckets = new Map<string, { id: string; name: string; sessions: WebSocketSession[] }>()
  buckets.set('__ungrouped', { id: '__ungrouped', name: '未分组', sessions: [] })
  groups.value.forEach((group) =>
    buckets.set(group.id, { id: group.id, name: group.name, sessions: [] }),
  )
  sessions.value.forEach((session) => {
    const id = session.groupId || '__ungrouped'
    if (!buckets.has(id)) buckets.set(id, { id, name: groupName(id), sessions: [] })
    buckets.get(id)?.sessions.push(session)
  })
  return Array.from(buckets.values()).filter(
    (group) => group.sessions.length > 0 || group.id !== '__ungrouped',
  )
})

onMounted(() => {
  reloadWorkbench()
})

async function reloadWorkbench() {
  await Promise.all([loadGroups(), loadSessions(1)])
}

async function loadGroups() {
  const result = await dataAPI.getWebSocketSessionGroups(props.projectId, props.connection.id)
  groups.value = result.list
  groups.value.forEach((group) => expandedGroups.add(group.id))
}

async function loadSessions(page = pagination.page) {
  loading.value = true
  try {
    const params: Record<string, any> = {
      page,
      pageSize: pagination.pageSize,
      q: search.value || undefined,
    }
    if (activeGroupId.value) params.groupId = activeGroupId.value
    const result = await dataAPI.getWebSocketSessions(props.projectId, props.connection.id, params)
    sessions.value = result.list
    Object.assign(pagination, result.pagination)
  } finally {
    loading.value = false
  }
}

function handleGroupChange() {
  loadSessions(1)
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
    groupId:
      activeGroupId.value && activeGroupId.value !== '__ungrouped' ? activeGroupId.value : null,
    name: '新建会话',
    url: props.connection.config?.url || '',
  })
  tabs.value.push({ id, isNew: true, dirty: true, draft })
  activeTabId.value = id
}

function openSession(session: WebSocketSession) {
  const exists = tabs.value.find((tab) => tab.id === session.id)
  if (exists) {
    activeTabId.value = exists.id
    return
  }
  tabs.value.push({ id: session.id, isNew: false, dirty: false, draft: normalizeDraft(session) })
  activeTabId.value = session.id
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
    tab.id = saved.id
    tab.isNew = false
    tab.dirty = false
    tab.draft = normalizeDraft(saved)
    activeTabId.value = saved.id
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
  connecting.value = true
  try {
    let saved = tab.draft as WebSocketSession
    if (tab.isNew || tab.dirty) {
      const result = await saveActive()
      if (!result) return
      saved = result
    }
    tab.preview = await dataAPI.connectWebSocketPreview(props.projectId, String(saved.id))
    await loadSessions()
    ElMessage.success('预览完成')
  } finally {
    connecting.value = false
  }
}

async function deleteActive() {
  const tab = activeTab.value
  if (!tab || tab.isNew) return
  await ElMessageBox.confirm('删除会话后对应数据点会标记失效，确认删除？', '删除会话', {
    type: 'warning',
  })
  await dataAPI.deleteWebSocketSession(props.projectId, String(tab.draft.id))
  tabs.value = tabs.value.filter((item) => item.id !== tab.id)
  activeTabId.value = tabs.value[0]?.id || ''
  await loadSessions()
  ElMessage.success('会话已删除')
}

function openGroupDialog(group?: WebSocketSessionGroup) {
  groupDialog.id = group?.id || ''
  groupDialog.name = group?.name || ''
  groupDialog.parentId = group?.parentId || ''
  groupDialog.visible = true
}

async function saveGroup() {
  const payload = { name: groupDialog.name, parentId: groupDialog.parentId || null }
  if (groupDialog.id)
    await dataAPI.updateWebSocketSessionGroup(props.projectId, groupDialog.id, payload)
  else await dataAPI.createWebSocketSessionGroup(props.projectId, props.connection.id, payload)
  groupDialog.visible = false
  await loadGroups()
}

function normalizeDraft(input: Partial<WebSocketSession>): SessionDraft {
  return {
    id: input.id,
    projectId: input.projectId,
    connectionId: input.connectionId || props.connection.id,
    groupId: input.groupId || null,
    name: input.name || '未命名会话',
    url: input.url || '',
    headers: [...(input.headers || [])],
    auth: { type: 'none', ...(input.auth || {}) },
    protocols: [...(input.protocols || [])],
    messages: [...(input.messages || [])],
    settings: { timeoutMs: 5000, messageLimit: 5, tlsVerify: true, ...(input.settings || {}) },
    enabled: input.enabled ?? true,
    sortOrder: input.sortOrder || 0,
    sourceType: input.sourceType,
    dataPointId: input.dataPointId || '',
    dataPointPath: input.dataPointPath || '',
    lastMessage: input.lastMessage,
    lastDiagnostic: input.lastDiagnostic || '',
    quality: input.quality || 'unknown',
    lastMessageAt: input.lastMessageAt,
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

const WebSocketKeyValueEditor = defineComponent({
  props: { modelValue: { type: Array, default: () => [] } },
  emits: ['update:modelValue', 'change'],
  setup(componentProps, { emit }) {
    const rows = computed<WebSocketKeyValueRow[]>({
      get: () => componentProps.modelValue as WebSocketKeyValueRow[],
      set: (value) => emit('update:modelValue', value),
    })
    const update = () => emit('change')
    const add = () => {
      rows.value = [...rows.value, { enabled: true, key: '', value: '', description: '' }]
      update()
    }
    const remove = (index: number) => {
      rows.value = rows.value.filter((_, i) => i !== index)
      update()
    }
    return () =>
      h('div', { class: 'ws-workbench__table-editor' }, [
        h(
          'button',
          { type: 'button', class: 'ws-workbench__mini-add', onClick: add },
          '新增 Header',
        ),
        h(
          'div',
          { class: 'ws-workbench__rows' },
          rows.value.map((row, index) =>
            h('div', { class: 'ws-workbench__row' }, [
              h('input', {
                type: 'checkbox',
                checked: row.enabled,
                onChange: (event: Event) => (
                  (row.enabled = (event.target as HTMLInputElement).checked),
                  update()
                ),
              }),
              h('input', {
                value: row.key,
                placeholder: 'Key',
                onInput: (event: Event) => (
                  (row.key = (event.target as HTMLInputElement).value),
                  update()
                ),
              }),
              h('input', {
                value: row.value,
                placeholder: 'Value',
                onInput: (event: Event) => (
                  (row.value = (event.target as HTMLInputElement).value),
                  update()
                ),
              }),
              h('button', { type: 'button', onClick: () => remove(index) }, '删除'),
            ]),
          ),
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
      set: (value) => emit('update:modelValue', value),
    })
    const add = () => {
      rows.value = [...rows.value, { enabled: true, value: '', description: '' }]
      emit('change')
    }
    return () =>
      h('div', { class: 'ws-workbench__table-editor' }, [
        h(
          'button',
          { type: 'button', class: 'ws-workbench__mini-add', onClick: add },
          '新增 Protocol',
        ),
        ...rows.value.map((row, index) =>
          h('div', { class: 'ws-workbench__row' }, [
            h('input', {
              type: 'checkbox',
              checked: row.enabled,
              onChange: (event: Event) => (
                (row.enabled = (event.target as HTMLInputElement).checked),
                emit('change')
              ),
            }),
            h('input', {
              value: row.value,
              placeholder: 'chat.v1',
              onInput: (event: Event) => (
                (row.value = (event.target as HTMLInputElement).value),
                emit('change')
              ),
            }),
            h(
              'button',
              {
                type: 'button',
                onClick: () => (
                  (rows.value = rows.value.filter((_, i) => i !== index)),
                  emit('change')
                ),
              },
              '删除',
            ),
          ]),
        ),
      ])
  },
})

const WebSocketMessageEditor = defineComponent({
  props: { modelValue: { type: Array, default: () => [] } },
  emits: ['update:modelValue', 'change'],
  setup(componentProps, { emit }) {
    const rows = computed<WebSocketSubscribeMessageRow[]>({
      get: () => componentProps.modelValue as WebSocketSubscribeMessageRow[],
      set: (value) => emit('update:modelValue', value),
    })
    const add = () => {
      rows.value = [
        ...rows.value,
        { enabled: true, name: '订阅消息', payload: '{}', description: '' },
      ]
      emit('change')
    }
    return () =>
      h('div', { class: 'ws-workbench__table-editor is-message' }, [
        h(
          'button',
          { type: 'button', class: 'ws-workbench__mini-add', onClick: add },
          '新增订阅消息',
        ),
        ...rows.value.map((row, index) =>
          h('div', { class: 'ws-workbench__message-row' }, [
            h('input', {
              type: 'checkbox',
              checked: row.enabled,
              onChange: (event: Event) => (
                (row.enabled = (event.target as HTMLInputElement).checked),
                emit('change')
              ),
            }),
            h('input', {
              value: row.name,
              placeholder: '名称',
              onInput: (event: Event) => (
                (row.name = (event.target as HTMLInputElement).value),
                emit('change')
              ),
            }),
            h('textarea', {
              value: row.payload,
              placeholder: '消息内容',
              onInput: (event: Event) => (
                (row.payload = (event.target as HTMLTextAreaElement).value),
                emit('change')
              ),
            }),
            h(
              'button',
              {
                type: 'button',
                onClick: () => (
                  (rows.value = rows.value.filter((_, i) => i !== index)),
                  emit('change')
                ),
              },
              '删除',
            ),
          ]),
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

.ws-workbench__filter {
  padding: 10px 12px;
  border-bottom: 1px solid #eef1f5;
}

.ws-workbench__filter :deep(.el-select) {
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
}

.ws-workbench__group-head svg.is-open {
  transform: rotate(90deg);
}

.ws-workbench__group-list {
  padding: 2px 0 8px 14px;
}

.ws-workbench__session-node {
  width: 100%;
  min-height: 34px;
  display: grid;
  grid-template-columns: auto 1fr auto;
  align-items: center;
  gap: 8px;
  padding: 0 8px;
  border-radius: 6px;
  cursor: pointer;
  text-align: left;
}

.ws-workbench__session-node.is-active,
.ws-workbench__session-node:hover,
.ws-workbench__group-head:hover {
  background: #eef5ff;
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
  grid-template-columns: auto 1fr auto auto;
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

.ws-workbench__editor {
  min-height: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
}

.ws-workbench__crumb-row,
.ws-workbench__name-row,
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

.ws-workbench__actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.ws-workbench__name-row :deep(.el-input) {
  flex: 1;
}

.ws-workbench__name-row :deep(.el-select) {
  width: 220px;
}

.ws-workbench__request-line :deep(.el-input) {
  flex: 1;
}

.ws-workbench__config-tabs {
  flex: 1;
  min-height: 220px;
  background: #fff;
  padding: 0 14px;
  overflow: auto;
}

.ws-workbench__auth,
.ws-workbench__settings {
  max-width: 760px;
  display: grid;
  grid-template-columns: 160px 1fr;
  gap: 12px;
  padding: 12px 0;
}

.ws-workbench__table-editor {
  padding: 12px 0;
}

.ws-workbench__mini-add {
  height: 30px;
  padding: 0 10px;
  border: 1px solid #d8dee8;
  border-radius: 6px;
  background: #fff;
  cursor: pointer;
}

.ws-workbench__row,
.ws-workbench__message-row {
  display: grid;
  grid-template-columns: 28px minmax(120px, 220px) minmax(180px, 1fr) 60px;
  gap: 8px;
  margin-top: 8px;
}

.ws-workbench__message-row {
  grid-template-columns: 28px 160px minmax(220px, 1fr) 60px;
}

.ws-workbench__row input,
.ws-workbench__message-row input,
.ws-workbench__message-row textarea {
  min-width: 0;
  border: 1px solid #d8dee8;
  border-radius: 6px;
  padding: 6px 8px;
  font: inherit;
}

.ws-workbench__message-row textarea {
  min-height: 72px;
  resize: vertical;
  font-family: ui-monospace, SFMono-Regular, Consolas, monospace;
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
.ws-workbench__blank {
  color: #64748b;
  display: flex;
  align-items: center;
  justify-content: center;
}

.ws-workbench__empty {
  min-height: 80px;
}

.ws-workbench__empty-response {
  flex: 1;
}

.ws-workbench__blank {
  flex: 1;
  flex-direction: column;
  gap: 10px;
}
</style>
