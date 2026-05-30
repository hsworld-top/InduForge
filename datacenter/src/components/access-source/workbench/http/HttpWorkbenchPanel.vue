<template>
  <section class="http-workbench">
    <aside class="http-workbench__sidebar">
      <WorkbenchSourceHeader
        :title="connection.name || '未命名 HTTP 接入源'"
        fallback-title="未命名 HTTP 接入源"
        :meta="sourceMetaRows"
        @back="$emit('back')"
      >
        <template #actions>
          <el-input
            v-model="search"
            class="http-workbench__search"
            size="small"
            clearable
            placeholder="搜索接口"
            @keyup.enter="loadRequests(1)"
          />
          <button
            type="button"
            class="workbench-source-header__icon-action is-primary"
            title="新建接口"
            aria-label="新建接口"
            @click="createDraftRequest"
          >
            <IconTablerPlus />
          </button>
          <button
            type="button"
            class="workbench-source-header__icon-action"
            title="新建分组"
            aria-label="新建分组"
            @click="openGroupDialog()"
          >
            <IconTablerFolderPlus />
          </button>
          <button
            type="button"
            class="workbench-source-header__icon-action"
            title="刷新"
            aria-label="刷新"
            @click="reloadWorkbench"
          >
            <IconTablerRefresh />
          </button>
        </template>
      </WorkbenchSourceHeader>

      <div class="http-workbench__group-filter">
        <el-select v-model="activeGroupId" size="small" @change="handleGroupChange">
          <el-option label="全部接口" value="" />
          <el-option label="未分组" value="__ungrouped" />
          <el-option
            v-for="group in groups"
            :key="String(group.id)"
            :label="group.name"
            :value="String(group.id)"
          />
        </el-select>
      </div>

      <div class="http-workbench__tree">
        <div v-if="loading" class="http-workbench__loading">
          <IconTablerLoader2 />
          <span>加载接口...</span>
        </div>
        <template v-else>
          <section
            v-for="group in groupedRequests"
            :key="group.id"
            class="http-workbench__group"
          >
            <button type="button" class="http-workbench__group-head" @click="toggleGroup(group.id)">
              <IconTablerChevronRight :class="{ 'is-open': expandedGroups.has(group.id) }" />
              <IconTablerFolder />
              <span>{{ group.name }}</span>
              <small>{{ group.requests.length }}</small>
            </button>
            <div v-show="expandedGroups.has(group.id)" class="http-workbench__group-list">
              <button
                v-for="request in group.requests"
                :key="String(request.id)"
                type="button"
                class="http-workbench__request-node"
                :class="{ 'is-active': activeTabId === String(request.id) }"
                @click="openRequest(request)"
              >
                <span class="http-workbench__method" :class="`is-${request.method.toLowerCase()}`">
                  {{ request.method }}
                </span>
                <span>{{ request.name }}</span>
                <el-dropdown trigger="click" @command="handleRequestCommand(request, $event)">
                  <button class="http-workbench__node-action" type="button" @click.stop>
                    <IconTablerDots />
                  </button>
                  <template #dropdown>
                    <el-dropdown-menu>
                      <el-dropdown-item command="duplicate">复制</el-dropdown-item>
                      <el-dropdown-item command="move">移动</el-dropdown-item>
                      <el-dropdown-item command="delete" divided>删除</el-dropdown-item>
                    </el-dropdown-menu>
                  </template>
                </el-dropdown>
              </button>
            </div>
          </section>
          <div v-if="requests.length === 0" class="http-workbench__empty">
            {{ search ? '没有匹配的接口' : '暂无接口' }}
          </div>
        </template>
      </div>

      <el-pagination
        v-if="pagination.total"
        class="http-workbench__pager"
        small
        layout="prev, pager, next"
        :current-page="pagination.page"
        :page-size="pagination.pageSize"
        :total="pagination.total"
        @current-change="loadRequests"
      />
    </aside>

    <main class="http-workbench__main">
      <div class="http-workbench__tabs">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          type="button"
          class="http-workbench__tab"
          :class="{ 'is-active': tab.id === activeTabId }"
          @click="activateTab(tab.id)"
        >
          <span class="http-workbench__method" :class="`is-${tab.draft.method.toLowerCase()}`">
            {{ tab.draft.method }}
          </span>
          <span>{{ tab.draft.name || '未命名接口' }}</span>
          <i v-if="tab.dirty" />
          <IconTablerX @click.stop="closeTab(tab.id)" />
        </button>
      </div>

      <div v-if="activeTab" class="http-workbench__editor">
        <div class="http-workbench__crumb-row">
          <div class="http-workbench__crumb">
            <span>HTTP 接入源</span>
            <IconTablerChevronRight />
            <span>{{ groupName(activeTab.draft.groupId) }}</span>
            <IconTablerChevronRight />
            <strong>{{ activeTab.draft.name || '未命名接口' }}</strong>
          </div>
          <div class="http-workbench__actions">
            <el-switch v-model="activeTab.draft.enabled" size="small" @change="markDirty" />
            <el-button size="small" :loading="saving" @click="saveActive">Save</el-button>
            <el-dropdown trigger="click" @command="handleActiveCommand">
              <el-button size="small">
                <IconTablerDots />
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="duplicate">复制接口</el-dropdown-item>
                  <el-dropdown-item command="move">移动接口</el-dropdown-item>
                  <el-dropdown-item command="delete" divided>删除接口</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </div>

        <div class="http-workbench__name-row">
          <el-input v-model="activeTab.draft.name" placeholder="接口名称" @input="markDirty" />
          <el-select
            v-model="activeTab.draft.groupId"
            placeholder="分组"
            clearable
            @change="markDirty"
          >
            <el-option
              v-for="group in groups"
              :key="String(group.id)"
              :label="group.name"
              :value="String(group.id)"
            />
          </el-select>
        </div>

        <div class="http-workbench__request-line">
          <el-select v-model="activeTab.draft.method" class="http-workbench__method-select" @change="markDirty">
            <el-option v-for="method in methods" :key="method" :label="method" :value="method" />
          </el-select>
          <el-input v-model="activeTab.draft.url" placeholder="/api/device/status 或 https://..." @input="markDirty" />
          <el-button type="primary" :loading="sending" @click="sendActive">Send</el-button>
        </div>

        <el-tabs v-model="activeConfigTab" class="http-workbench__config-tabs">
          <el-tab-pane label="Params" name="params">
            <HttpKeyValueEditor v-model="activeTab.draft.params" @change="markDirty" />
          </el-tab-pane>
          <el-tab-pane label="Authorization" name="auth">
            <div class="http-workbench__auth">
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
                <el-input v-model="activeTab.draft.auth.username" placeholder="Username" @input="markDirty" />
                <el-input v-model="activeTab.draft.auth.password" placeholder="Password" show-password @input="markDirty" />
              </template>
            </div>
          </el-tab-pane>
          <el-tab-pane label="Headers" name="headers">
            <HttpKeyValueEditor v-model="activeTab.draft.headers" @change="markDirty" />
          </el-tab-pane>
          <el-tab-pane label="Body" name="body">
            <div class="http-workbench__body-mode">
              <el-radio-group v-model="activeTab.draft.bodyType" size="small" @change="markDirty">
                <el-radio-button label="none">none</el-radio-button>
                <el-radio-button label="json">json</el-radio-button>
                <el-radio-button label="raw">raw</el-radio-button>
                <el-radio-button label="form-data">form-data</el-radio-button>
                <el-radio-button label="x-www-form-urlencoded">urlencoded</el-radio-button>
              </el-radio-group>
            </div>
            <el-input
              v-if="activeTab.draft.bodyType === 'json'"
              v-model="jsonBodyText"
              type="textarea"
              :rows="9"
              spellcheck="false"
              @input="handleJsonBodyInput"
            />
            <el-input
              v-else-if="activeTab.draft.bodyType === 'raw'"
              v-model="activeTab.draft.body.raw"
              type="textarea"
              :rows="9"
              spellcheck="false"
              @input="markDirty"
            />
            <HttpKeyValueEditor
              v-else-if="activeTab.draft.bodyType !== 'none'"
              v-model="formBodyRows"
              @change="handleFormBodyChange"
            />
            <el-empty v-else description="该请求不发送 Body" />
          </el-tab-pane>
          <el-tab-pane label="Settings" name="settings">
            <div class="http-workbench__settings">
              <label>
                <span>超时</span>
                <el-input-number
                  v-model="activeTab.draft.settings.timeoutMs"
                  :min="1000"
                  :max="30000"
                  :step="500"
                  controls-position="right"
                  @change="markDirty"
                />
              </label>
              <el-checkbox v-model="activeTab.draft.settings.followRedirects" @change="markDirty">
                跟随重定向
              </el-checkbox>
              <el-checkbox v-model="activeTab.draft.settings.tlsVerify" @change="markDirty">
                TLS 校验
              </el-checkbox>
            </div>
          </el-tab-pane>
        </el-tabs>

        <section class="http-workbench__response">
          <div class="http-workbench__response-head">
            <strong>Response</strong>
            <div v-if="activeTab.response" class="http-workbench__response-meta">
              <span :class="statusTone(activeTab.response.status)">{{ activeTab.response.statusText }}</span>
              <span>{{ activeTab.response.durationMs }} ms</span>
              <span>{{ activeTab.response.sizeBytes }} B</span>
              <span>{{ formatTime(activeTab.response.receivedAt) }}</span>
            </div>
          </div>
          <el-tabs v-if="activeTab.response" v-model="activeResponseTab">
            <el-tab-pane label="Body" name="body">
              <pre>{{ formattedResponseBody }}</pre>
            </el-tab-pane>
            <el-tab-pane label="Headers" name="headers">
              <el-table :data="responseHeaders" size="small" height="220">
                <el-table-column prop="key" label="Header" min-width="160" />
                <el-table-column prop="value" label="Value" min-width="240" />
              </el-table>
            </el-tab-pane>
            <el-tab-pane label="History" name="history">
              <el-table :data="activeTab.history" size="small" height="220">
                <el-table-column prop="statusText" label="状态" width="140" />
                <el-table-column prop="durationMs" label="耗时" width="100" />
                <el-table-column prop="sizeBytes" label="大小" width="100" />
                <el-table-column prop="receivedAt" label="时间" min-width="180" />
              </el-table>
            </el-tab-pane>
          </el-tabs>
          <div v-else class="http-workbench__response-empty">
            <IconTablerSend />
            <span>Click Send to get a response</span>
          </div>
        </section>
      </div>

      <div v-else class="http-workbench__placeholder">
        <IconTablerWorldWww />
        <strong>选择或新建接口</strong>
        <span>HTTP 工作台以接口请求为中心，一个接口同步一个 object 数据点。</span>
      </div>
    </main>

    <el-dialog v-model="groupDialogVisible" title="请求分组" width="420px">
      <el-form label-position="top" @submit.prevent>
        <el-form-item label="分组名称">
          <el-input v-model="groupForm.name" autofocus />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="groupDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="groupSaving" @click="saveGroup">保存</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import IconTablerChevronRight from '~icons/tabler/chevron-right'
import IconTablerDots from '~icons/tabler/dots'
import IconTablerFolder from '~icons/tabler/folder'
import IconTablerFolderPlus from '~icons/tabler/folder-plus'
import IconTablerLoader2 from '~icons/tabler/loader-2'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerSend from '~icons/tabler/send'
import IconTablerWorldWww from '~icons/tabler/world-www'
import IconTablerX from '~icons/tabler/x'
import dataAPI from '@/api/data.api'
import type {
  HttpKeyValueRow,
  HttpRequest,
  HttpRequestGroup,
  HttpSendResponse,
} from '@/api/schemas/http-workbench.schema'
import WorkbenchSourceHeader from '@/components/workbench/WorkbenchSourceHeader.vue'
import { getApiErrorMessage } from '@/utils/request'

type AccessSourceConnection = {
  id: string
  name?: string
  type?: string
  status?: string
  config?: Record<string, unknown>
}

type RequestDraft = {
  id?: string
  groupId?: string | null
  name: string
  method: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  url: string
  params: HttpKeyValueRow[]
  headers: HttpKeyValueRow[]
  auth: Record<string, any>
  bodyType: 'none' | 'json' | 'raw' | 'form-data' | 'x-www-form-urlencoded'
  body: Record<string, any>
  settings: Record<string, any>
  enabled: boolean
  sortOrder: number
}

type HttpTab = {
  id: string
  draft: RequestDraft
  dirty: boolean
  response: HttpSendResponse | null
  history: HttpSendResponse[]
}

const props = defineProps<{
  connection: AccessSourceConnection
  projectId: string
}>()

defineEmits<{
  (event: 'back'): void
}>()

const methods = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE'] as const
const groups = ref<HttpRequestGroup[]>([])
const requests = ref<HttpRequest[]>([])
const tabs = ref<HttpTab[]>([])
const activeTabId = ref('')
const activeConfigTab = ref('params')
const activeResponseTab = ref('body')
const activeGroupId = ref('')
const search = ref('')
const loading = ref(false)
const saving = ref(false)
const sending = ref(false)
const expandedGroups = ref(new Set<string>(['__ungrouped']))
const pagination = reactive({ page: 1, pageSize: 20, total: 0 })
const groupDialogVisible = ref(false)
const groupSaving = ref(false)
const groupForm = reactive({ id: '', name: '' })
let searchTimer: number | undefined

const activeTab = computed(() => tabs.value.find((tab) => tab.id === activeTabId.value) || null)
const config = computed(() => props.connection.config || {})
const sourceMetaRows = computed(() => [
  { label: '类型', value: 'HTTP' },
  { label: 'Base URL', value: String(config.value.baseUrl || '未配置') },
])

const groupedRequests = computed(() => {
  const buckets = groups.value.map((group) => ({
    id: String(group.id),
    name: group.name,
    requests: requests.value.filter((request) => String(request.groupId || '') === String(group.id)),
  }))
  buckets.unshift({
    id: '__ungrouped',
    name: '未分组',
    requests: requests.value.filter((request) => !request.groupId),
  })
  return buckets.filter((bucket) => bucket.requests.length > 0 || bucket.id === '__ungrouped')
})

const jsonBodyText = computed({
  get() {
    const draft = activeTab.value?.draft
    if (!draft) return ''
    const value = draft.body.json ?? {}
    return typeof value === 'string' ? value : JSON.stringify(value, null, 2)
  },
  set(value: string) {
    if (activeTab.value) {
      activeTab.value.draft.body.json = value
    }
  },
})

const formBodyRows = computed({
  get() {
    return (activeTab.value?.draft.body.form || []) as HttpKeyValueRow[]
  },
  set(value: HttpKeyValueRow[]) {
    if (activeTab.value) {
      activeTab.value.draft.body.form = value
    }
  },
})

const formattedResponseBody = computed(() => {
  const body = activeTab.value?.response?.body
  if (body === undefined || body === null) return ''
  if (typeof body === 'string') return body
  return JSON.stringify(body, null, 2)
})

const responseHeaders = computed(() =>
  Object.entries(activeTab.value?.response?.headers || {}).map(([key, value]) => ({ key, value })),
)

onMounted(() => {
  reloadWorkbench()
})

watch(search, () => {
  window.clearTimeout(searchTimer)
  searchTimer = window.setTimeout(() => loadRequests(1), 250)
})

const reloadWorkbench = async () => {
  loading.value = true
  try {
    const groupRes = await dataAPI.getHttpRequestGroups(props.projectId, props.connection.id)
    groups.value = groupRes.list || []
    await loadRequests(1)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载 HTTP 工作台失败'))
  } finally {
    loading.value = false
  }
}

const loadRequests = async (page = pagination.page) => {
  const params: Record<string, any> = {
    page,
    pageSize: pagination.pageSize,
    q: search.value || undefined,
    groupId: activeGroupId.value || undefined,
  }
  const res = await dataAPI.getHttpRequests(props.projectId, props.connection.id, params)
  requests.value = res.list || []
  pagination.page = res.pagination?.page || page
  pagination.pageSize = res.pagination?.pageSize || 20
  pagination.total = res.pagination?.total || 0
}

const handleGroupChange = () => {
  loadRequests(1).catch((error) => ElMessage.error(getApiErrorMessage(error, '切换分组失败')))
}

const openRequest = async (request: HttpRequest) => {
  const id = String(request.id)
  if (tabs.value.some((tab) => tab.id === id)) {
    activeTabId.value = id
    return
  }
  tabs.value.push({
    id,
    draft: toDraft(request),
    dirty: false,
    response: (request.lastResponse as HttpSendResponse) || null,
    history: [],
  })
  activeTabId.value = id
}

const createDraftRequest = () => {
  const id = `new-${Date.now()}`
  const draft = createEmptyDraft()
  if (activeGroupId.value && activeGroupId.value !== '__ungrouped') {
    draft.groupId = activeGroupId.value
  }
  tabs.value.push({ id, draft, dirty: true, response: null, history: [] })
  activeTabId.value = id
}

const activateTab = async (id: string) => {
  activeTabId.value = id
}

const closeTab = async (id: string) => {
  const index = tabs.value.findIndex((tab) => tab.id === id)
  if (index < 0) return
  const tab = tabs.value[index]
  if (tab.dirty) {
    try {
      await ElMessageBox.confirm('当前接口有未保存改动，关闭后会丢失。', '关闭接口', {
        confirmButtonText: '放弃',
        cancelButtonText: '取消',
        type: 'warning',
      })
    } catch {
      return
    }
  }
  tabs.value.splice(index, 1)
  if (activeTabId.value === id) {
    activeTabId.value = tabs.value[Math.max(0, index - 1)]?.id || ''
  }
}

const saveActive = async () => {
  const tab = activeTab.value
  if (!tab) return null
  saving.value = true
  try {
    const payload = toPayload(tab.draft)
    const saved = tab.draft.id
      ? await dataAPI.updateHttpRequest(props.projectId, tab.draft.id, payload)
      : await dataAPI.createHttpRequest(props.projectId, props.connection.id, payload)
    const oldId = tab.id
    tab.id = String(saved.id)
    tab.draft = toDraft(saved)
    tab.dirty = false
    activeTabId.value = tab.id
    if (oldId !== tab.id) {
      tabs.value = [...tabs.value]
    }
    await loadRequests()
    ElMessage.success('接口已保存')
    return saved
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '保存接口失败'))
    return null
  } finally {
    saving.value = false
  }
}

const sendActive = async () => {
  const tab = activeTab.value
  if (!tab) return
  const saved = tab.dirty || !tab.draft.id ? await saveActive() : tab.draft
  if (!saved || !tab.draft.id) return
  sending.value = true
  try {
    const response = await dataAPI.sendHttpRequest(props.projectId, tab.draft.id)
    tab.response = response
    tab.history.unshift(response)
    tab.history = tab.history.slice(0, 20)
    activeResponseTab.value = 'body'
    await loadRequests()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '发送 HTTP 请求失败'))
  } finally {
    sending.value = false
  }
}

const markDirty = () => {
  if (activeTab.value) activeTab.value.dirty = true
}

const handleJsonBodyInput = () => {
  const tab = activeTab.value
  if (!tab) return
  try {
    tab.draft.body.json = JSON.parse(jsonBodyText.value || '{}')
  } catch {
    tab.draft.body.json = jsonBodyText.value
  }
  markDirty()
}

const handleFormBodyChange = (rows: HttpKeyValueRow[]) => {
  formBodyRows.value = rows
  markDirty()
}

const toggleGroup = (id: string) => {
  const next = new Set(expandedGroups.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  expandedGroups.value = next
}

const openGroupDialog = (group?: HttpRequestGroup) => {
  groupForm.id = group ? String(group.id) : ''
  groupForm.name = group?.name || ''
  groupDialogVisible.value = true
}

const saveGroup = async () => {
  if (!groupForm.name.trim()) {
    ElMessage.warning('分组名称不能为空')
    return
  }
  groupSaving.value = true
  try {
    if (groupForm.id) {
      await dataAPI.updateHttpRequestGroup(props.projectId, groupForm.id, { name: groupForm.name })
    } else {
      await dataAPI.createHttpRequestGroup(props.projectId, props.connection.id, { name: groupForm.name })
    }
    groupDialogVisible.value = false
    const res = await dataAPI.getHttpRequestGroups(props.projectId, props.connection.id)
    groups.value = res.list || []
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '保存分组失败'))
  } finally {
    groupSaving.value = false
  }
}

const handleRequestCommand = async (request: HttpRequest, command: string | number | object) => {
  if (command === 'delete') {
    await deleteRequest(String(request.id))
  } else if (command === 'duplicate') {
    duplicateRequest(request)
  } else if (command === 'move') {
    openRequest(request)
  }
}

const handleActiveCommand = async (command: string | number | object) => {
  const tab = activeTab.value
  if (!tab) return
  if (command === 'delete' && tab.draft.id) await deleteRequest(tab.draft.id)
  if (command === 'duplicate') duplicateDraft(tab.draft)
}

const duplicateRequest = (request: HttpRequest) => {
  const draft = toDraft(request)
  draft.id = undefined
  draft.name = `${draft.name} Copy`
  tabs.value.push({ id: `copy-${Date.now()}`, draft, dirty: true, response: null, history: [] })
  activeTabId.value = tabs.value[tabs.value.length - 1].id
}

const duplicateDraft = (draft: RequestDraft) => {
  const next = cloneDraft(draft)
  next.id = undefined
  next.name = `${next.name} Copy`
  tabs.value.push({ id: `copy-${Date.now()}`, draft: next, dirty: true, response: null, history: [] })
  activeTabId.value = tabs.value[tabs.value.length - 1].id
}

const deleteRequest = async (requestId: string) => {
  try {
    await ElMessageBox.confirm('删除接口后，对应数据点会标记为失效。', '删除接口', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await dataAPI.deleteHttpRequest(props.projectId, requestId)
    tabs.value = tabs.value.filter((tab) => tab.draft.id !== requestId)
    if (!tabs.value.some((tab) => tab.id === activeTabId.value)) {
      activeTabId.value = tabs.value[0]?.id || ''
    }
    await loadRequests()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(getApiErrorMessage(error, '删除接口失败'))
    }
  }
}

const groupName = (groupId?: string | null) => {
  if (!groupId) return '未分组'
  return groups.value.find((group) => String(group.id) === String(groupId))?.name || '未分组'
}

const statusTone = (status: number) => {
  if (status >= 200 && status < 300) return 'is-success'
  if (status >= 400) return 'is-danger'
  return 'is-warning'
}

const formatTime = (value?: string | null) => {
  if (!value) return '-'
  return String(value).replace('T', ' ').slice(0, 19)
}

const createEmptyDraft = (): RequestDraft => ({
  name: 'New Request',
  method: 'GET',
  url: '/',
  params: [],
  headers: [],
  auth: { type: 'none' },
  bodyType: 'none',
  body: {},
  settings: { timeoutMs: 5000, followRedirects: true, tlsVerify: true },
  enabled: true,
  sortOrder: 0,
})

const toDraft = (request: HttpRequest): RequestDraft => ({
  id: String(request.id),
  groupId: request.groupId ? String(request.groupId) : null,
  name: request.name,
  method: request.method as RequestDraft['method'],
  url: request.url,
  params: cloneRows(request.params || []),
  headers: cloneRows(request.headers || []),
  auth: { type: 'none', ...(request.auth || {}) },
  bodyType: request.bodyType as RequestDraft['bodyType'],
  body: { ...(request.body || {}) },
  settings: { timeoutMs: 5000, followRedirects: true, tlsVerify: true, ...(request.settings || {}) },
  enabled: request.enabled,
  sortOrder: request.sortOrder || 0,
})

const cloneDraft = (draft: RequestDraft): RequestDraft => ({
  ...draft,
  params: cloneRows(draft.params),
  headers: cloneRows(draft.headers),
  auth: { ...draft.auth },
  body: JSON.parse(JSON.stringify(draft.body || {})),
  settings: { ...draft.settings },
})

const cloneRows = (rows: HttpKeyValueRow[]) => rows.map((row) => ({ ...row }))

const toPayload = (draft: RequestDraft) => ({
  groupId: draft.groupId || null,
  name: draft.name,
  method: draft.method,
  url: draft.url,
  params: draft.params,
  headers: draft.headers,
  auth: draft.auth,
  bodyType: draft.bodyType,
  body: draft.body,
  settings: draft.settings,
  enabled: draft.enabled,
  sortOrder: draft.sortOrder,
})

const HttpKeyValueEditor = defineComponent({
  name: 'HttpKeyValueEditor',
  props: {
    modelValue: { type: Array, required: true },
  },
  emits: ['update:modelValue', 'change'],
  setup(componentProps, { emit }) {
    const rows = computed({
      get: () => componentProps.modelValue as HttpKeyValueRow[],
      set: (value: HttpKeyValueRow[]) => {
        emit('update:modelValue', value)
        emit('change', value)
      },
    })
    const updateRow = (index: number, key: keyof HttpKeyValueRow, value: any) => {
      const next = rows.value.map((row) => ({ ...row }))
      next[index] = { ...next[index], [key]: value }
      rows.value = next
    }
    const addRow = () => {
      rows.value = [...rows.value, { enabled: true, key: '', value: '', description: '' }]
    }
    const removeRow = (index: number) => {
      rows.value = rows.value.filter((_, rowIndex) => rowIndex !== index)
    }
    return () =>
      h('div', { class: 'http-kv-editor' }, [
        h(
          'div',
          { class: 'http-kv-editor__head' },
          h('button', { type: 'button', onClick: addRow }, 'Add'),
        ),
        h(
          'table',
          rows.value.map((row, index) =>
            h('tr', { key: index }, [
              h('td', h('input', { type: 'checkbox', checked: row.enabled, onChange: (event: Event) => updateRow(index, 'enabled', (event.target as HTMLInputElement).checked) })),
              h('td', h('input', { value: row.key, placeholder: 'Key', onInput: (event: Event) => updateRow(index, 'key', (event.target as HTMLInputElement).value) })),
              h('td', h('input', { value: row.value, placeholder: 'Value', onInput: (event: Event) => updateRow(index, 'value', (event.target as HTMLInputElement).value) })),
              h('td', h('input', { value: row.description, placeholder: 'Description', onInput: (event: Event) => updateRow(index, 'description', (event.target as HTMLInputElement).value) })),
              h('td', h('button', { type: 'button', onClick: () => removeRow(index) }, '×')),
            ]),
          ),
        ),
      ])
  },
})
</script>

<style scoped>
.http-workbench {
  height: 100%;
  display: grid;
  grid-template-columns: 304px minmax(0, 1fr);
  background: #f6f7f9;
  color: #1f2937;
  overflow: hidden;
}

.http-workbench__sidebar {
  min-height: 0;
  display: flex;
  flex-direction: column;
  border-right: 1px solid #dfe3ea;
  background: #fff;
}

.http-workbench__search {
  width: 150px;
}

.http-workbench__group-filter {
  padding: 8px 10px;
  border-bottom: 1px solid #eef1f5;
}

.http-workbench__tree {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 8px 8px 0;
}

.http-workbench__group-head,
.http-workbench__request-node {
  width: 100%;
  height: 32px;
  display: flex;
  align-items: center;
  gap: 6px;
  border: 0;
  background: transparent;
  border-radius: 6px;
  color: #334155;
  cursor: pointer;
}

.http-workbench__group-head:hover,
.http-workbench__request-node:hover,
.http-workbench__request-node.is-active {
  background: #edf4ff;
}

.http-workbench__group-head svg {
  width: 15px;
  height: 15px;
}

.http-workbench__group-head svg:first-child {
  transition: transform 0.15s ease;
}

.http-workbench__group-head svg:first-child.is-open {
  transform: rotate(90deg);
}

.http-workbench__group-head small {
  margin-left: auto;
  color: #94a3b8;
}

.http-workbench__group-list {
  padding-left: 18px;
}

.http-workbench__request-node {
  padding: 0 6px;
}

.http-workbench__request-node span:nth-child(2) {
  min-width: 0;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  text-align: left;
}

.http-workbench__node-action {
  width: 24px;
  height: 24px;
  border: 0;
  background: transparent;
  color: #64748b;
}

.http-workbench__method {
  width: 46px;
  font-size: 11px;
  font-weight: 700;
  text-align: left;
  color: #64748b;
}

.http-workbench__method.is-get {
  color: #059669;
}

.http-workbench__method.is-post {
  color: #2563eb;
}

.http-workbench__method.is-put,
.http-workbench__method.is-patch {
  color: #b45309;
}

.http-workbench__method.is-delete {
  color: #dc2626;
}

.http-workbench__pager {
  padding: 8px;
  border-top: 1px solid #eef1f5;
  justify-content: center;
}

.http-workbench__main {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.http-workbench__tabs {
  height: 36px;
  display: flex;
  align-items: end;
  gap: 2px;
  padding: 0 10px;
  border-bottom: 1px solid #dfe3ea;
  background: #f8fafc;
  overflow-x: auto;
}

.http-workbench__tab {
  height: 32px;
  max-width: 220px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 0 10px;
  border: 1px solid transparent;
  border-bottom: 0;
  background: transparent;
  border-radius: 6px 6px 0 0;
  color: #475569;
}

.http-workbench__tab.is-active {
  background: #fff;
  border-color: #dfe3ea;
  color: #111827;
}

.http-workbench__tab span:nth-child(2) {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.http-workbench__tab i {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #f59e0b;
}

.http-workbench__tab svg {
  width: 14px;
  height: 14px;
}

.http-workbench__editor {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: #fff;
}

.http-workbench__crumb-row,
.http-workbench__name-row,
.http-workbench__request-line {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  border-bottom: 1px solid #eef1f5;
}

.http-workbench__crumb-row {
  height: 40px;
}

.http-workbench__crumb {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 6px;
  color: #64748b;
  font-size: 13px;
}

.http-workbench__crumb svg {
  width: 13px;
  height: 13px;
}

.http-workbench__crumb strong {
  color: #0f172a;
}

.http-workbench__actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.http-workbench__name-row {
  height: 48px;
}

.http-workbench__name-row .el-select {
  width: 180px;
}

.http-workbench__request-line {
  height: 50px;
}

.http-workbench__method-select {
  width: 118px;
}

.http-workbench__config-tabs {
  flex: 1;
  min-height: 0;
  padding: 0 14px;
}

.http-workbench__auth,
.http-workbench__settings {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 0;
}

.http-workbench__auth .el-select {
  width: 180px;
}

.http-workbench__settings label {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.http-workbench__body-mode {
  padding: 8px 0;
}

.http-workbench__response {
  height: 320px;
  display: flex;
  flex-direction: column;
  border-top: 1px solid #dfe3ea;
  background: #fbfcfe;
}

.http-workbench__response-head {
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 14px;
  border-bottom: 1px solid #eef1f5;
}

.http-workbench__response-meta {
  display: flex;
  align-items: center;
  gap: 14px;
  font-size: 12px;
  color: #64748b;
}

.http-workbench__response-meta .is-success {
  color: #059669;
}

.http-workbench__response-meta .is-warning {
  color: #b45309;
}

.http-workbench__response-meta .is-danger {
  color: #dc2626;
}

.http-workbench__response :deep(.el-tabs) {
  min-height: 0;
  flex: 1;
  padding: 0 14px;
}

.http-workbench__response pre {
  height: 238px;
  margin: 0;
  padding: 12px;
  overflow: auto;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  background: #0f172a;
  color: #dbeafe;
  font-size: 12px;
  line-height: 1.55;
}

.http-workbench__response-empty,
.http-workbench__placeholder,
.http-workbench__empty,
.http-workbench__loading {
  min-height: 120px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: #94a3b8;
}

.http-workbench__placeholder {
  flex: 1;
  background: #fff;
}

.http-workbench__placeholder svg,
.http-workbench__response-empty svg {
  width: 28px;
  height: 28px;
}

.http-workbench__placeholder strong {
  color: #334155;
}

.http-kv-editor {
  padding: 8px 0;
}

.http-kv-editor__head {
  height: 30px;
  display: flex;
  justify-content: flex-end;
}

.http-kv-editor__head button,
.http-kv-editor table button {
  height: 24px;
  border: 1px solid #d5dbe5;
  border-radius: 4px;
  background: #fff;
}

.http-kv-editor table {
  width: 100%;
  border-collapse: collapse;
}

.http-kv-editor td {
  height: 34px;
  border: 1px solid #e5e7eb;
}

.http-kv-editor td:first-child {
  width: 38px;
  text-align: center;
}

.http-kv-editor td:last-child {
  width: 40px;
  text-align: center;
}

.http-kv-editor input[type='text'],
.http-kv-editor td input:not([type]) {
  width: 100%;
  height: 32px;
  padding: 0 8px;
  border: 0;
  outline: none;
  background: transparent;
}
</style>
