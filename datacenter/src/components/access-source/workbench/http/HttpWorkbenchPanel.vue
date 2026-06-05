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

      <div class="http-workbench__tree">
        <div v-if="loading" class="http-workbench__loading">
          <IconTablerLoader2 />
          <span>加载接口...</span>
        </div>
        <template v-else>
          <HttpTreeNode
            v-for="group in requestTree.groups"
            :key="group.id"
            :node="group"
            :active-tab-id="activeTabId"
            :expanded-groups="expandedGroups"
            @toggle="toggleGroup"
            @open-request="openRequest"
            @request-contextmenu="openRequestMenu"
            @group-contextmenu="openGroupMenu"
          />
          <button
            v-for="request in requestTree.rootRequests"
            :key="String(request.id)"
            type="button"
            class="http-workbench__request-node"
            :class="{ 'is-active': activeTabId === String(request.id) }"
            @click="openRequest(request)"
            @contextmenu.prevent.stop="openRequestMenu($event, request)"
          >
            <span class="http-workbench__method" :class="`is-${request.method.toLowerCase()}`">
              {{ request.method }}
            </span>
            <span class="http-workbench__request-name">{{ request.name }}</span>
          </button>
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
            <el-option label="根目录" :value="null" />
            <el-option
              v-for="group in allGroupOptions"
              :key="group.id"
              :label="group.label"
              :value="group.id"
            />
          </el-select>
        </div>

        <div class="http-workbench__request-line">
          <el-select v-model="activeTab.draft.method" class="http-workbench__method-select" @change="markDirty">
            <el-option v-for="method in methods" :key="method" :label="method" :value="method" />
          </el-select>
          <el-input v-model="activeTab.draft.url" placeholder="https://api.example.com/device/status" @input="markDirty" />
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

    <WorkbenchGroupDialog
      ref="groupDialogRef"
      v-model="groupDialogVisible"
      :mode="groupForm.id ? 'edit' : 'create'"
      :title="groupForm.id ? '编辑请求分组' : '新建请求分组'"
      :group="editingGroup"
      :group-options="groupOptions"
      :initial-parent-id="groupForm.parentId"
      :loading="groupSaving"
      @submit="saveGroup"
    />

    <DcDialog
      v-model="moveDialogVisible"
      :title="moveTargetType === 'group' ? '移动分组' : '移动接口'"
      width="420px"
      :close-disabled="moveSaving"
    >
      <el-form label-position="top" class="http-workbench__move-form" @submit.prevent>
        <el-form-item label="目标分组">
          <el-select
            v-model="moveTargetGroupId"
            class="http-workbench__move-select"
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
        <div class="http-workbench__move-footer">
          <el-button @click="moveDialogVisible = false">取消</el-button>
          <el-button type="primary" :loading="moveSaving" :disabled="!canMoveTarget" @click="moveTarget">
            移动
          </el-button>
        </div>
      </template>
    </DcDialog>

    <Teleport to="body">
      <div
        v-if="contextMenu.visible"
        class="http-workbench__menu-mask"
        @click="closeContextMenu"
        @contextmenu.prevent="closeContextMenu"
      >
        <div
          class="http-workbench__context-menu"
          :style="{ left: `${contextMenu.x}px`, top: `${contextMenu.y}px` }"
          @click.stop
        >
          <button
            v-if="contextMenu.type === 'request'"
            type="button"
            @click="emitContextAction('open')"
          >
            <IconTablerWorldWww class="http-workbench__menu-icon" />
            <span>打开接口</span>
          </button>
          <button
            v-if="contextMenu.type === 'request'"
            type="button"
            @click="emitContextAction('duplicate')"
          >
            <IconTablerCopy class="http-workbench__menu-icon" />
            <span>复制接口</span>
          </button>
          <button
            v-if="contextMenu.type === 'request'"
            type="button"
            @click="emitContextAction('move')"
          >
            <IconTablerFolderSymlink class="http-workbench__menu-icon" />
            <span>移动到分组</span>
          </button>
          <button
            v-if="contextMenu.type === 'group'"
            type="button"
            @click="emitContextAction('create-child')"
          >
            <IconTablerFolderPlus class="http-workbench__menu-icon" />
            <span>新建子分组</span>
          </button>
          <button
            v-if="contextMenu.type === 'group'"
            type="button"
            @click="emitContextAction('edit')"
          >
            <IconTablerPencil class="http-workbench__menu-icon" />
            <span>编辑分组</span>
          </button>
          <button
            v-if="contextMenu.type === 'group'"
            type="button"
            @click="emitContextAction('move')"
          >
            <IconTablerFolderSymlink class="http-workbench__menu-icon" />
            <span>移动分组</span>
          </button>
          <button
            v-if="contextMenu.type === 'group'"
            type="button"
            class="is-danger"
            @click="emitContextAction('delete')"
          >
            <IconTablerTrash class="http-workbench__menu-icon" />
            <span>删除分组</span>
          </button>
          <button
            v-if="contextMenu.type === 'request'"
            type="button"
            class="is-danger"
            @click="emitContextAction('delete')"
          >
            <IconTablerTrash class="http-workbench__menu-icon" />
            <span>删除</span>
          </button>
        </div>
      </div>
    </Teleport>
  </section>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import IconTablerChevronRight from '~icons/tabler/chevron-right'
import IconTablerCopy from '~icons/tabler/copy'
import IconTablerDots from '~icons/tabler/dots'
import IconTablerFolder from '~icons/tabler/folder'
import IconTablerFolderOpen from '~icons/tabler/folder-open'
import IconTablerFolderPlus from '~icons/tabler/folder-plus'
import IconTablerFolderSymlink from '~icons/tabler/folder-symlink'
import IconTablerLoader2 from '~icons/tabler/loader-2'
import IconTablerPencil from '~icons/tabler/pencil'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerSend from '~icons/tabler/send'
import IconTablerTrash from '~icons/tabler/trash'
import IconTablerWorldWww from '~icons/tabler/world-www'
import IconTablerX from '~icons/tabler/x'
import dataAPI from '@/api/data.api'
import DcDialog from '@/components/shared/DcDialog.vue'
import type {
  HttpKeyValueRow,
  HttpRequest,
  HttpRequestGroup,
  HttpSendResponse,
} from '@/api/schemas/http-workbench.schema'
import WorkbenchGroupDialog from '@/components/workbench/WorkbenchGroupDialog.vue'
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

type HttpRequestGroupNode = HttpRequestGroup & {
  children: HttpRequestGroupNode[]
  requests: HttpRequest[]
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
const search = ref('')
const loading = ref(false)
const saving = ref(false)
const sending = ref(false)
const expandedGroups = ref(new Set<string>())
const pagination = reactive({ page: 1, pageSize: 20, total: 0 })
const groupDialogVisible = ref(false)
const groupSaving = ref(false)
const groupForm = reactive({ id: '', name: '', parentId: null as string | null })
const groupDialogRef = ref<InstanceType<typeof WorkbenchGroupDialog> | null>(null)
const moveDialogVisible = ref(false)
const moveSaving = ref(false)
const moveTargetType = ref<'request' | 'group'>('request')
const movingRequest = ref<HttpRequest | null>(null)
const movingGroup = ref<HttpRequestGroupNode | null>(null)
const moveTargetGroupId = ref<string | null>(null)
const contextMenu = ref<{
  visible: boolean
  type: 'request' | 'group' | null
  x: number
  y: number
  request: HttpRequest | null
  group: HttpRequestGroupNode | null
}>({
  visible: false,
  type: null,
  x: 0,
  y: 0,
  request: null,
  group: null,
})
let searchTimer: number | undefined

const activeTab = computed(() => tabs.value.find((tab) => tab.id === activeTabId.value) || null)
const config = computed(() => props.connection.config || {})
const sourceMetaRows = computed(() => [
  { label: '类型', value: 'HTTP' },
  { label: '接口', value: `${pagination.total || 0} 个` },
])

const requestTree = computed(() => buildHttpRequestTree(groups.value, requests.value))
const allGroupOptions = computed(() =>
  flattenHttpGroups(
    groups.value,
    moveTargetType.value === 'group' ? collectHttpGroupIds(movingGroup.value) : new Set(),
  ),
)
const groupOptions = computed(() =>
  flattenHttpGroups(
    groups.value,
    groupForm.id ? collectHttpGroupIds(findHttpGroupNode(groupForm.id)) : new Set(),
  ),
)
const editingGroup = computed(() =>
  groupForm.id ? groups.value.find((group) => String(group.id) === groupForm.id) || null : null,
)
const movingRequestGroupId = computed(() =>
  movingRequest.value?.groupId ? String(movingRequest.value.groupId) : null,
)
const movingGroupParentId = computed(() =>
  movingGroup.value?.parentId ? String(movingGroup.value.parentId) : null,
)
const canMoveTarget = computed(
  () =>
    !moveSaving.value &&
    (moveTargetType.value === 'request'
      ? Boolean(movingRequest.value) && moveTargetGroupId.value !== movingRequestGroupId.value
      : Boolean(movingGroup.value) && moveTargetGroupId.value !== movingGroupParentId.value),
)

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
  }
  const res = await dataAPI.getHttpRequests(props.projectId, props.connection.id, params)
  requests.value = res.list || []
  pagination.page = res.pagination?.page || page
  pagination.pageSize = res.pagination?.pageSize || 20
  pagination.total = res.pagination?.total || 0
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
  groupForm.parentId = group?.parentId ? String(group.parentId) : null
  groupDialogVisible.value = true
}

const openChildGroupDialog = (group: HttpRequestGroup) => {
  groupForm.id = ''
  groupForm.name = ''
  groupForm.parentId = String(group.id)
  groupDialogVisible.value = true
}

const saveGroup = async (payload: { name: string; parentId: string | null }) => {
  if (!payload.name.trim()) {
    ElMessage.warning('分组名称不能为空')
    return
  }
  groupSaving.value = true
  try {
    let savedGroup: HttpRequestGroup | null = null
    if (groupForm.id) {
      savedGroup = await dataAPI.updateHttpRequestGroup(props.projectId, groupForm.id, {
        name: payload.name,
        parentId: payload.parentId || null,
      })
    } else {
      savedGroup = await dataAPI.createHttpRequestGroup(props.projectId, props.connection.id, {
        name: payload.name,
        parentId: payload.parentId || null,
      })
    }
    groupDialogRef.value?.closeSilently()
    const res = await dataAPI.getHttpRequestGroups(props.projectId, props.connection.id)
    groups.value = res.list || []
    if (savedGroup?.id) {
      const next = new Set(expandedGroups.value)
      next.add(String(savedGroup.id))
      if (savedGroup.parentId) next.add(String(savedGroup.parentId))
      expandedGroups.value = next
    }
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '保存分组失败'))
  } finally {
    groupSaving.value = false
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

const openRequestMenu = (event: MouseEvent, request: HttpRequest) => {
  contextMenu.value = {
    visible: true,
    type: 'request',
    x: Math.min(event.clientX, window.innerWidth - 180),
    y: Math.min(event.clientY, window.innerHeight - 186),
    request,
    group: null,
  }
}

const openGroupMenu = (event: MouseEvent, group: HttpRequestGroupNode) => {
  contextMenu.value = {
    visible: true,
    type: 'group',
    x: Math.min(event.clientX, window.innerWidth - 180),
    y: Math.min(event.clientY, window.innerHeight - 168),
    request: null,
    group,
  }
}

const closeContextMenu = () => {
  contextMenu.value.visible = false
}

const emitContextAction = (action: 'open' | 'duplicate' | 'move' | 'delete' | 'create-child' | 'edit') => {
  const { type, request, group } = contextMenu.value
  closeContextMenu()
  if (type === 'request' && request) {
    if (action === 'open') {
      void openRequest(request)
      return
    }
    if (action === 'duplicate') {
      duplicateRequest(request)
      return
    }
    if (action === 'move') {
      openMoveRequestDialog(request)
      return
    }
    if (action === 'delete') {
      void deleteRequest(String(request.id))
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

const openMoveRequestDialog = (request: HttpRequest) => {
  moveTargetType.value = 'request'
  movingRequest.value = request
  movingGroup.value = null
  moveTargetGroupId.value = request.groupId ? String(request.groupId) : null
  moveDialogVisible.value = true
}

const openMoveGroupDialog = (group: HttpRequestGroupNode) => {
  moveTargetType.value = 'group'
  movingRequest.value = null
  movingGroup.value = group
  moveTargetGroupId.value = group.parentId ? String(group.parentId) : null
  moveDialogVisible.value = true
}

const moveTarget = async () => {
  if (moveTargetType.value === 'group') {
    await moveGroup()
    return
  }
  await moveRequest()
}

const moveRequest = async () => {
  const request = movingRequest.value
  if (!request || !canMoveTarget.value) return
  const requestId = String(request.id)
  const openTab = tabs.value.find((tab) => tab.draft.id === requestId)
  if (openTab?.dirty) {
    openTab.draft.groupId = moveTargetGroupId.value || null
    activeTabId.value = openTab.id
    moveDialogVisible.value = false
    ElMessage.info('已在未保存的接口中调整分组，保存接口后生效')
    return
  }

  moveSaving.value = true
  try {
    const draft = toDraft(request)
    draft.groupId = moveTargetGroupId.value || null
    const saved = await dataAPI.updateHttpRequest(props.projectId, requestId, toPayload(draft))
    if (openTab) {
      openTab.draft = toDraft(saved)
      openTab.dirty = false
    }
    moveSaving.value = false
    moveDialogVisible.value = false
    movingRequest.value = null
    await loadRequests()
    ElMessage.success('接口已移动')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '移动接口失败'))
  } finally {
    moveSaving.value = false
  }
}

const moveGroup = async () => {
  const group = movingGroup.value
  if (!group || !canMoveTarget.value) return
  moveSaving.value = true
  try {
    await dataAPI.updateHttpRequestGroup(props.projectId, String(group.id), {
      name: group.name,
      parentId: moveTargetGroupId.value || null,
    })
    moveSaving.value = false
    moveDialogVisible.value = false
    movingGroup.value = null
    const res = await dataAPI.getHttpRequestGroups(props.projectId, props.connection.id)
    groups.value = res.list || []
    ElMessage.success('分组已移动')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '移动分组失败'))
  } finally {
    moveSaving.value = false
  }
}

const deleteGroup = async (group: HttpRequestGroupNode) => {
  try {
    await ElMessageBox.confirm(
      `确认删除分组「${group.name}」？组内接口会回到根目录。`,
      '删除分组',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning',
      },
    )
    await dataAPI.deleteHttpRequestGroup(props.projectId, String(group.id))
    const [groupRes, requestRes] = await Promise.all([
      dataAPI.getHttpRequestGroups(props.projectId, props.connection.id),
      dataAPI.getHttpRequests(props.projectId, props.connection.id, {
        page: pagination.page,
        pageSize: pagination.pageSize,
        q: search.value.trim() || undefined,
      }),
    ])
    groups.value = groupRes.list || []
    requests.value = requestRes.list || []
    pagination.total = requestRes.pagination?.total || requests.value.length
    ElMessage.success('分组已删除')
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(getApiErrorMessage(error, '删除分组失败'))
    }
  }
}

const groupName = (groupId?: string | null) => {
  if (!groupId) return '根目录'
  return groups.value.find((group) => String(group.id) === String(groupId))?.name || '根目录'
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
  url: 'https://',
  params: [],
  headers: [],
  auth: { type: 'none' },
  bodyType: 'none',
  body: {},
  settings: { timeoutMs: 5000, followRedirects: true, tlsVerify: true },
  enabled: true,
  sortOrder: 0,
})

function buildHttpRequestTree(
  groupList: HttpRequestGroup[],
  requestList: HttpRequest[],
): { groups: HttpRequestGroupNode[]; rootRequests: HttpRequest[] } {
  const nodes = new Map<string, HttpRequestGroupNode>()
  groupList.forEach((group) => {
    nodes.set(String(group.id), { ...group, children: [], requests: [] })
  })

  const roots: HttpRequestGroupNode[] = []
  nodes.forEach((node) => {
    const parentId = node.parentId ? String(node.parentId) : ''
    const parent = parentId ? nodes.get(parentId) : null
    if (parent) parent.children.push(node)
    else roots.push(node)
  })

  const rootRequests: HttpRequest[] = []
  requestList.forEach((request) => {
    const groupId = request.groupId ? String(request.groupId) : ''
    const group = groupId ? nodes.get(groupId) : null
    if (group) group.requests.push(request)
    else rootRequests.push(request)
  })

  const sortRequests = (items: HttpRequest[]) =>
    items.sort((left, right) => (left.sortOrder || 0) - (right.sortOrder || 0) || left.name.localeCompare(right.name))
  const sortGroups = (items: HttpRequestGroupNode[]) => {
    items.sort((left, right) => (left.sortOrder || 0) - (right.sortOrder || 0) || left.name.localeCompare(right.name))
    items.forEach((item) => {
      sortGroups(item.children)
      sortRequests(item.requests)
    })
  }

  sortGroups(roots)
  sortRequests(rootRequests)
  return { groups: roots, rootRequests }
}

function flattenHttpGroups(
  groupList: HttpRequestGroup[],
  blockedIds: Set<string> = new Set(),
): Array<{ id: string; label: string }> {
  const roots = buildHttpRequestTree(groupList, []).groups
  const visit = (group: HttpRequestGroupNode, depth: number): Array<{ id: string; label: string }> => {
    const id = String(group.id)
    const children = group.children.flatMap((child) => visit(child, depth + 1))
    if (blockedIds.has(id)) return children
    return [{ id, label: `${'　'.repeat(depth)}${group.name}` }, ...children]
  }
  return roots.flatMap((group) => visit(group, 0))
}

function collectHttpGroupIds(group?: HttpRequestGroupNode | null) {
  const result = new Set<string>()
  const visit = (node?: HttpRequestGroupNode | null) => {
    if (!node) return
    result.add(String(node.id))
    node.children.forEach(visit)
  }
  visit(group)
  return result
}

function findHttpGroupNode(groupId?: string | null) {
  if (!groupId) return null
  const stack = [...requestTree.value.groups]
  while (stack.length) {
    const group = stack.shift()
    if (!group) continue
    if (String(group.id) === String(groupId)) return group
    stack.push(...group.children)
  }
  return null
}

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

const HttpTreeNode = defineComponent({
  name: 'HttpTreeNode',
  props: {
    node: { type: Object, required: true },
    activeTabId: { type: String, default: '' },
    expandedGroups: { type: Object, required: true },
  },
  emits: ['toggle', 'open-request', 'request-contextmenu', 'group-contextmenu'],
  setup(props, { emit }) {
    const renderRequest = (request: HttpRequest) =>
      h(
        'button',
        {
          type: 'button',
          class: [
            'http-workbench__request-node',
            { 'is-active': props.activeTabId === String(request.id) },
          ],
          onClick: () => emit('open-request', request),
          onContextmenu: (event: MouseEvent) => {
            event.preventDefault()
            event.stopPropagation()
            emit('request-contextmenu', event, request)
          },
        },
        [
          h(
            'span',
            { class: ['http-workbench__method', `is-${request.method.toLowerCase()}`] },
            request.method,
          ),
          h('span', { class: 'http-workbench__request-name' }, request.name),
        ],
      )

    const renderGroup = (node: HttpRequestGroupNode) => {
      const id = String(node.id)
      const expanded = (props.expandedGroups as Set<string>).has(id)
      return h('section', { class: 'http-workbench__group' }, [
        h(
          'button',
          {
            type: 'button',
            class: 'http-workbench__group-head',
            onClick: () => emit('toggle', id),
            onContextmenu: (event: MouseEvent) => {
              event.preventDefault()
              event.stopPropagation()
              emit('group-contextmenu', event, node)
            },
          },
          [
            h(IconTablerChevronRight, { class: { 'is-open': expanded } }),
            h(expanded ? IconTablerFolderOpen : IconTablerFolder),
            h('span', { class: 'http-workbench__group-name' }, node.name),
            h('small', countHttpGroupRequests(node)),
          ],
        ),
        expanded
          ? h('div', { class: 'http-workbench__group-list' }, [
              ...node.children.map((child) => renderGroup(child)),
              ...node.requests.map(renderRequest),
            ])
          : null,
      ])
    }

    return () => renderGroup(props.node as HttpRequestGroupNode)
  },
})

function countHttpGroupRequests(node: HttpRequestGroupNode): number {
  return node.requests.length + node.children.reduce((sum, child) => sum + countHttpGroupRequests(child), 0)
}

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
  grid-template-columns: 260px minmax(0, 1fr);
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
  color: var(--dc-text);
  overflow: hidden;
}

.http-workbench__sidebar {
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border-right: 1px solid var(--dc-border);
  background: var(--dc-surface-muted);
}

.http-workbench__search {
  width: 150px;
}

.http-workbench__tree {
  flex: 1;
  min-height: 0;
  display: grid;
  align-content: start;
  gap: 2px;
  overflow-y: auto;
  padding: 8px 8px 12px;
}

.http-workbench__group-head,
.http-workbench__request-node {
  width: 100%;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  text-align: left;
}

.http-workbench__group-head {
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

.http-workbench__group-head:hover,
.http-workbench__request-node:hover {
  border-color: var(--dc-border);
  background: var(--dc-surface-muted);
  color: var(--dc-text);
}

.http-workbench__request-node.is-active {
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.http-workbench__group-head svg {
  width: 16px;
  height: 16px;
}

.http-workbench__group-head svg:first-child {
  width: 14px;
  height: 14px;
  transition: transform 0.16s ease;
}

.http-workbench__group-head svg:first-child.is-open {
  transform: rotate(90deg);
}

.http-workbench__group-head svg:nth-child(2) {
  color: var(--dc-primary);
}

.http-workbench__group-head small {
  min-width: 20px;
  padding: 2px 6px;
  border-radius: 999px;
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
  font-size: 11px;
  text-align: center;
}

.http-workbench__group-list {
  display: grid;
  gap: 2px;
  margin-left: 10px;
  padding-left: 6px;
}

.http-workbench__group {
  min-width: 0;
  display: grid;
  gap: 2px;
}

.http-workbench__request-node {
  display: grid;
  grid-template-columns: 46px minmax(0, 1fr);
  align-items: center;
  gap: 6px;
  min-height: 30px;
  padding: 3px 6px;
  font-size: 12px;
  cursor: pointer;
}

.http-workbench__group-name,
.http-workbench__request-name {
  min-width: 0;
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 700;
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

.http-workbench__move-form {
  display: grid;
  gap: 2px;
}

.http-workbench__move-select {
  width: 100%;
}

.http-workbench__move-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.http-workbench__menu-mask {
  position: fixed;
  inset: 0;
  z-index: 2100;
}

.http-workbench__context-menu {
  position: fixed;
  min-width: 148px;
  padding: 4px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
}

.http-workbench__context-menu button {
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

.http-workbench__context-menu button:hover {
  background: var(--dc-surface-muted);
  color: var(--dc-primary);
}

.http-workbench__context-menu button.is-danger:hover {
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
}

.http-workbench__menu-icon {
  width: 15px;
  height: 15px;
  flex: 0 0 auto;
}

.http-workbench__tree :deep(.http-workbench__group) {
  min-width: 0;
  display: grid;
  gap: 2px;
}

.http-workbench__tree :deep(.http-workbench__group-list) {
  display: grid;
  gap: 2px;
  margin-left: 10px;
  padding-left: 6px;
}

.http-workbench__tree :deep(.http-workbench__group-head),
.http-workbench__tree :deep(.http-workbench__request-node) {
  width: 100%;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  text-align: left;
}

.http-workbench__tree :deep(.http-workbench__group-head) {
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

.http-workbench__tree :deep(.http-workbench__request-node) {
  display: grid;
  grid-template-columns: 46px minmax(0, 1fr);
  align-items: center;
  gap: 6px;
  min-height: 30px;
  padding: 3px 6px;
  font-size: 12px;
  cursor: pointer;
}

.http-workbench__tree :deep(.http-workbench__group-head:hover),
.http-workbench__tree :deep(.http-workbench__request-node:hover) {
  border-color: var(--dc-border);
  background: var(--dc-surface-muted);
  color: var(--dc-text);
}

.http-workbench__tree :deep(.http-workbench__request-node.is-active) {
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.http-workbench__tree :deep(.http-workbench__group-head svg) {
  width: 16px;
  height: 16px;
}

.http-workbench__tree :deep(.http-workbench__group-head svg:first-child) {
  width: 14px;
  height: 14px;
  transition: transform 0.16s ease;
}

.http-workbench__tree :deep(.http-workbench__group-head svg:first-child.is-open) {
  transform: rotate(90deg);
}

.http-workbench__tree :deep(.http-workbench__group-head svg:nth-child(2)) {
  color: var(--dc-primary);
}

.http-workbench__tree :deep(.http-workbench__group-head small) {
  min-width: 20px;
  padding: 2px 6px;
  border-radius: 999px;
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
  font-size: 11px;
  text-align: center;
}

.http-workbench__tree :deep(.http-workbench__group-name),
.http-workbench__tree :deep(.http-workbench__request-name) {
  min-width: 0;
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 700;
}

.http-workbench__tree :deep(.http-workbench__method) {
  width: 46px;
  font-size: 11px;
  font-weight: 700;
  text-align: left;
  color: #64748b;
}

.http-workbench__tree :deep(.http-workbench__method.is-get) {
  color: #059669;
}

.http-workbench__tree :deep(.http-workbench__method.is-post) {
  color: #2563eb;
}

.http-workbench__tree :deep(.http-workbench__method.is-put),
.http-workbench__tree :deep(.http-workbench__method.is-patch) {
  color: #b45309;
}

.http-workbench__tree :deep(.http-workbench__method.is-delete) {
  color: #dc2626;
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
