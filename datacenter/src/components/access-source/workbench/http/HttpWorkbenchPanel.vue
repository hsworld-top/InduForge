<template>
  <section class="http-workbench">
    <aside class="http-workbench__sidebar">
      <WorkbenchSourceHeader
        :title="connection.name || ui('未命名 HTTP 接入源', 'Unnamed HTTP Source')"
        :fallback-title="ui('未命名 HTTP 接入源', 'Unnamed HTTP Source')"
        :meta="sourceMetaRows"
        @back="$emit('back')"
      >
        <template #actions>
          <el-input
            v-model="search"
            class="http-workbench__search"
            size="small"
            clearable
            :placeholder="ui('搜索接口', 'Search endpoints')"
            @keyup.enter="loadRequests(1)"
          />
          <button
            type="button"
            class="workbench-source-header__icon-action is-primary"
            :title="ui('新建接口', 'New Endpoint')"
            :aria-label="ui('新建接口', 'New Endpoint')"
            @click="createDraftRequest"
          >
            <IconTablerPlus />
          </button>
          <button
            type="button"
            class="workbench-source-header__icon-action"
            :title="ui('新建分组', 'New Group')"
            :aria-label="ui('新建分组', 'New Group')"
            @click="openGroupDialog()"
          >
            <IconTablerFolderPlus />
          </button>
          <button
            type="button"
            class="workbench-source-header__icon-action"
            :title="ui('刷新', 'Refresh')"
            :aria-label="ui('刷新', 'Refresh')"
            @click="reloadWorkbench"
          >
            <IconTablerRefresh />
          </button>
        </template>
      </WorkbenchSourceHeader>

      <div class="http-workbench__tree">
        <div v-if="loading" class="http-workbench__loading">
          <IconTablerLoader2 />
          <span>{{ ui('加载接口...', 'Loading endpoints...') }}</span>
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
            {{ search ? ui('没有匹配的接口', 'No matching endpoints') : ui('暂无接口', 'No endpoints') }}
          </div>
          <button
            v-if="requests.length < pagination.total"
            type="button"
            class="http-workbench__load-more"
            :disabled="loadingMore"
            @click="loadMoreRequests"
          >
            {{ loadingMore ? ui('加载中...', 'Loading...') : ui(`加载更多（${requests.length}/${pagination.total}）`, `Load more (${requests.length}/${pagination.total})`) }}
          </button>
        </template>
      </div>
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
          <el-tooltip
            :content="tab.draft.name || ui('未命名接口', 'Unnamed Endpoint')"
            placement="top"
            :show-after="400"
            :disabled="(tab.draft.name || '').length <= 12"
          >
            <span class="http-workbench__tab-name">{{ tab.draft.name || ui('未命名接口', 'Unnamed Endpoint') }}</span>
          </el-tooltip>
          <i v-if="tab.dirty" />
          <IconTablerX @click.stop="closeTab(tab.id)" />
        </button>
      </div>

      <div v-if="activeTab" class="http-workbench__editor">
        <div class="http-workbench__crumb-row">
          <div class="http-workbench__crumb">
            <span>{{ ui('HTTP 接入源', 'HTTP Source') }}</span>
            <IconTablerChevronRight />
            <span>{{ groupName(activeTab.draft.groupId) }}</span>
            <IconTablerChevronRight />
            <el-input
              v-model="activeTab.draft.name"
              class="http-workbench__name-input"
              size="small"
              :placeholder="ui('未命名接口', 'Unnamed Endpoint')"
              maxlength="64"
              @input="markDirty"
            />
          </div>
          <div class="http-workbench__actions">
            <span v-if="activeTab.draft.outputs.length" class="http-workbench__datapoint-path">
              {{ outputCountLabel(activeTab.draft.outputs.length) }}
            </span>
            <el-tooltip :content="ui('保存当前接口', 'Save endpoint')" placement="top" :show-after="400">
              <el-button
                size="small"
                type="primary"
                :loading="saving"
                class="http-workbench__icon-btn"
                :aria-label="ui('保存', 'Save')"
                @click="saveActive"
              >
                <IconTablerDeviceFloppy />
              </el-button>
            </el-tooltip>
          </div>
        </div>

        <div class="http-workbench__request-line">
          <el-select
            v-model="activeTab.draft.method"
            class="http-workbench__method-select"
            @change="markDirty"
          >
            <el-option v-for="method in methods" :key="method" :label="method" :value="method" />
          </el-select>
          <el-select
            v-model="activeHttpScheme"
            class="http-workbench__protocol-select"
            @change="markDirty"
          >
            <el-option label="http://" value="http" />
            <el-option label="https://" value="https" />
          </el-select>
          <el-input
            v-model="activeTab.draft.url"
            placeholder="https://api.example.com/device/status"
            @input="markDirty"
          />
          <el-tooltip :content="ui('发送请求', 'Send request')" placement="top" :show-after="400">
            <el-button
              size="small"
              type="primary"
              :loading="sending"
              class="http-workbench__icon-btn"
              :aria-label="ui('发送', 'Send')"
              @click="sendActive"
            >
              <IconTablerSend />
            </el-button>
          </el-tooltip>
        </div>

        <el-tabs v-model="activeConfigTab" class="http-workbench__config-tabs">
          <el-tab-pane :label="ui('参数', 'Parameters')" name="params">
            <HttpKeyValueEditor v-model="activeTab.draft.params" @change="markDirty" />
          </el-tab-pane>
          <el-tab-pane :label="ui('认证', 'Authorization')" name="auth">
            <el-form class="http-workbench__form" label-position="top" size="small" @submit.prevent>
              <el-form-item :label="ui('认证方式', 'Authorization Type')">
                <el-select
                  v-model="activeTab.draft.auth.type"
                  class="http-workbench__form-control"
                  @change="markDirty"
                >
                  <el-option :label="ui('无认证', 'No Authorization')" value="none" />
                  <el-option label="Bearer Token" value="bearer" />
                  <el-option label="Basic Auth" value="basic" />
                </el-select>
              </el-form-item>
              <el-form-item v-if="activeTab.draft.auth.type === 'bearer'" label="Token">
                <el-input
                  v-model="activeTab.draft.auth.token"
                  class="http-workbench__form-control"
                  :placeholder="ui('请输入 Token', 'Enter a token')"
                  @input="markDirty"
                />
              </el-form-item>
              <template v-if="activeTab.draft.auth.type === 'basic'">
                <el-form-item :label="ui('用户名', 'Username')">
                  <el-input
                    v-model="activeTab.draft.auth.username"
                    class="http-workbench__form-control"
                    :placeholder="ui('请输入用户名', 'Enter a username')"
                    @input="markDirty"
                  />
                </el-form-item>
                <el-form-item :label="ui('密码', 'Password')">
                  <el-input
                    v-model="activeTab.draft.auth.password"
                    class="http-workbench__form-control"
                    :placeholder="ui('请输入密码', 'Enter a password')"
                    show-password
                    @input="markDirty"
                  />
                </el-form-item>
              </template>
            </el-form>
          </el-tab-pane>
          <el-tab-pane :label="ui('请求头', 'Headers')" name="headers">
            <HttpKeyValueEditor v-model="activeTab.draft.headers" @change="markDirty" />
          </el-tab-pane>
          <el-tab-pane :label="ui('请求体', 'Body')" name="body">
            <div class="http-workbench__body-mode">
              <el-radio-group v-model="activeTab.draft.bodyType" size="small" @change="markDirty">
                <el-radio-button label="none">{{ ui('无', 'None') }}</el-radio-button>
                <el-radio-button label="json"> <IconTablerBraces />json </el-radio-button>
                <el-radio-button label="raw"> <IconTablerCode />{{ ui('原始', 'Raw') }} </el-radio-button>
                <el-radio-button label="form-data"> <IconTablerForms />{{ ui('表单', 'Form Data') }} </el-radio-button>
                <el-radio-button label="x-www-form-urlencoded">
                  <IconTablerBraces />{{ ui('表单编码', 'URL Encoded') }}
                </el-radio-button>
              </el-radio-group>
            </div>
            <div v-if="activeTab.draft.bodyType === 'json'" class="http-workbench__monaco">
              <MonacoEditor
                v-model="jsonBodyText"
                language="json"
                :height="'240px'"
                :theme="isDark ? 'vs-dark' : 'vs'"
                :options="bodyEditorOptions"
                @change="handleJsonBodyInput"
              />
            </div>
            <div v-else-if="activeTab.draft.bodyType === 'raw'" class="http-workbench__monaco">
              <MonacoEditor
                v-model="activeTab.draft.body.raw"
                language="plaintext"
                :height="'240px'"
                :theme="isDark ? 'vs-dark' : 'vs'"
                :options="bodyEditorOptions"
                @change="markDirty"
              />
            </div>
            <HttpKeyValueEditor
              v-else-if="activeTab.draft.bodyType !== 'none'"
              v-model="formBodyRows"
              @change="handleFormBodyChange"
            />
            <el-empty v-else :description="ui('该请求不发送 Body', 'This request does not send a body')" />
          </el-tab-pane>
          <el-tab-pane :label="ui('输出映射', 'Output Mapping')" name="outputs">
            <SourceOutputEditor
              v-model="activeTab.draft.outputs"
              :title="ui('响应输出', 'Response Output')"
              :sample="activeTab.response?.body"
              whole-data-type="object"
              @change="markDirty"
            />
          </el-tab-pane>
          <el-tab-pane :label="ui('设置', 'Settings')" name="settings">
            <el-form class="http-workbench__form" label-position="top" size="small" @submit.prevent>
              <el-form-item :label="ui('超时（毫秒）', 'Timeout (ms)')">
                <el-input-number
                  v-model="activeTab.draft.settings.timeoutMs"
                  :min="1000"
                  :max="30000"
                  :step="500"
                  controls-position="right"
                  @change="markDirty"
                />
              </el-form-item>
              <el-form-item :label="ui('行为', 'Behavior')">
                <div class="http-workbench__settings-checks">
                  <el-checkbox
                    v-model="activeTab.draft.settings.followRedirects"
                    @change="markDirty"
                  >
                    {{ ui('跟随重定向', 'Follow Redirects') }}
                  </el-checkbox>
                  <el-checkbox v-model="activeTab.draft.settings.tlsVerify" @change="markDirty">
                    {{ ui('TLS 校验', 'Verify TLS') }}
                  </el-checkbox>
                </div>
              </el-form-item>
            </el-form>
          </el-tab-pane>
        </el-tabs>

        <section class="http-workbench__response" :style="responsePanelStyle">
          <div
            class="http-workbench__response-resizer"
            role="separator"
            aria-orientation="horizontal"
            :title="ui('拖拽调整响应区高度，双击恢复默认', 'Drag to resize the response panel; double-click to reset')"
            @mousedown.prevent="startResponseResize"
            @dblclick="resetResponseHeight"
          />
          <div class="http-workbench__response-head">
            <strong>{{ ui('响应', 'Response') }}</strong>
            <div v-if="activeTab.response" class="http-workbench__response-meta">
              <WorkbenchStatusPill :label="responseStatusLabel" :tone="responseStatusTone" />
              <span>{{ activeTab.response.durationMs }} ms</span>
              <span>{{ activeTab.response.sizeBytes }} B</span>
              <span>{{ formatTime(activeTab.response.receivedAt) }}</span>
            </div>
          </div>
          <el-tabs v-if="activeTab.response" v-model="activeResponseTab">
            <el-tab-pane :label="ui('响应体', 'Body')" name="body">
              <pre>{{ formattedResponseBody }}</pre>
            </el-tab-pane>
            <el-tab-pane :label="ui('响应头', 'Headers')" name="headers">
              <el-table :data="responseHeaders" size="small" class="http-workbench__response-table">
                <el-table-column prop="key" label="Header" min-width="160" />
                <el-table-column prop="value" label="Value" min-width="240" />
              </el-table>
            </el-tab-pane>
          </el-tabs>
          <div v-else class="http-workbench__response-empty">
            <IconTablerSend />
            <span>{{ ui('点击发送按钮发起请求', 'Click Send to make a request') }}</span>
          </div>
        </section>
      </div>

      <div v-else class="http-workbench__placeholder">
        <IconTablerApiApp />
        <strong>{{ ui('选择或新建接口', 'Select or Create an Endpoint') }}</strong>
        <span>{{ ui('每个接口可以把完整响应或样本字段映射为一个或多个强类型数据点。', 'Map a complete response or sample fields to one or more strongly typed data points.') }}</span>
      </div>
    </main>

    <WorkbenchGroupDialog
      ref="groupDialogRef"
      v-model="groupDialogVisible"
      :mode="groupForm.id ? 'edit' : 'create'"
      :title="groupForm.id ? ui('编辑请求分组', 'Edit Request Group') : ui('新建请求分组', 'New Request Group')"
      :group="editingGroup"
      :group-options="groupOptions"
      :initial-parent-id="groupForm.parentId"
      :loading="groupSaving"
      @submit="saveGroup"
    />

    <DcDialog
      v-model="moveDialogVisible"
      :title="moveTargetType === 'group' ? ui('移动分组', 'Move Group') : ui('移动接口', 'Move Endpoint')"
      width="420px"
      :close-disabled="moveSaving"
    >
      <el-form label-position="top" class="http-workbench__move-form" @submit.prevent>
        <el-form-item :label="ui('目标分组', 'Destination Group')">
          <el-select
            v-model="moveTargetGroupId"
            class="http-workbench__move-select"
            clearable
            :placeholder="ui('根目录', 'Root')"
          >
            <el-option :label="ui('根目录', 'Root')" :value="null" />
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
          <el-button @click="moveDialogVisible = false">{{ ui('取消', 'Cancel') }}</el-button>
          <el-button
            type="primary"
            :loading="moveSaving"
            :disabled="!canMoveTarget"
            @click="moveTarget"
          >
            {{ ui('移动', 'Move') }}
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
            <IconTablerApi class="http-workbench__menu-icon" />
            <span>{{ ui('打开接口', 'Open Endpoint') }}</span>
          </button>
          <button
            v-if="contextMenu.type === 'request'"
            type="button"
            @click="emitContextAction('duplicate')"
          >
            <IconTablerCopy class="http-workbench__menu-icon" />
            <span>{{ ui('复制接口', 'Duplicate Endpoint') }}</span>
          </button>
          <button
            v-if="contextMenu.type === 'request'"
            type="button"
            @click="emitContextAction('move')"
          >
            <IconTablerFolderSymlink class="http-workbench__menu-icon" />
            <span>{{ ui('移动到分组', 'Move to Group') }}</span>
          </button>
          <button
            v-if="contextMenu.type === 'group'"
            type="button"
            @click="emitContextAction('create-child')"
          >
            <IconTablerFolderPlus class="http-workbench__menu-icon" />
            <span>{{ ui('新建子分组', 'New Child Group') }}</span>
          </button>
          <button
            v-if="contextMenu.type === 'group'"
            type="button"
            @click="emitContextAction('edit')"
          >
            <IconTablerPencil class="http-workbench__menu-icon" />
            <span>{{ ui('编辑分组', 'Edit Group') }}</span>
          </button>
          <button
            v-if="contextMenu.type === 'group'"
            type="button"
            @click="emitContextAction('move')"
          >
            <IconTablerFolderSymlink class="http-workbench__menu-icon" />
            <span>{{ ui('移动分组', 'Move Group') }}</span>
          </button>
          <button
            v-if="contextMenu.type === 'group'"
            type="button"
            class="is-danger"
            @click="emitContextAction('delete')"
          >
            <IconTablerTrash class="http-workbench__menu-icon" />
            <span>{{ ui('删除分组', 'Delete Group') }}</span>
          </button>
          <button
            v-if="contextMenu.type === 'request'"
            type="button"
            class="is-danger"
            @click="emitContextAction('delete')"
          >
            <IconTablerTrash class="http-workbench__menu-icon" />
            <span>{{ ui('删除', 'Delete') }}</span>
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
} from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import IconTablerApi from '~icons/tabler/api'
import IconTablerApiApp from '~icons/tabler/api-app'
import IconTablerBraces from '~icons/tabler/braces'
import IconTablerChevronRight from '~icons/tabler/chevron-right'
import IconTablerCode from '~icons/tabler/code'
import IconTablerCopy from '~icons/tabler/copy'
import IconTablerDeviceFloppy from '~icons/tabler/device-floppy'
import IconTablerFolder from '~icons/tabler/folder'
import IconTablerFolderOpen from '~icons/tabler/folder-open'
import IconTablerFolderPlus from '~icons/tabler/folder-plus'
import IconTablerFolderSymlink from '~icons/tabler/folder-symlink'
import IconTablerForms from '~icons/tabler/forms'
import IconTablerLoader2 from '~icons/tabler/loader-2'
import IconTablerPencil from '~icons/tabler/pencil'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerSend from '~icons/tabler/send'
import IconTablerTrash from '~icons/tabler/trash'
import IconTablerX from '~icons/tabler/x'
import dataAPI from '@/api/data.api'
import DcDialog from '@/components/shared/DcDialog.vue'
import type {
  HttpKeyValueRow,
  HttpRequest,
  HttpRequestGroup,
  HttpSendResponse,
} from '@/api/schemas/http-workbench.schema'
import type { SourceOutputInput } from '@/api/schemas/source-output.schema'
import { createWholeSourceOutput } from '@/api/schemas/source-output.schema'
import SourceOutputEditor from '@/components/shared/SourceOutputEditor.vue'
import HttpKeyValueEditor from './HttpKeyValueEditor.vue'
import MonacoEditor from '@/components/MonacoEditor.vue'
import WorkbenchGroupDialog from '@/components/workbench/WorkbenchGroupDialog.vue'
import WorkbenchSourceHeader from '@/components/workbench/WorkbenchSourceHeader.vue'
import WorkbenchStatusPill from '@/components/workbench/WorkbenchStatusPill.vue'
import { getApiErrorMessage } from '@/utils/request'
import { useWorkbenchBottomPanelResize } from '@/composables/useWorkbenchBottomPanelResize'
import { datacenterLocale } from '@/i18n/runtime'
import { datacenterTheme } from '@/theme/runtime'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)
const isDark = computed(() => datacenterTheme.value === 'dark')
const outputCountLabel = (count: number) =>
  ui(`${count} 个输出数据点`, `${count} output data point${count === 1 ? '' : 's'}`)

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
  dataPointPath: string
  outputs: SourceOutputInput[]
}

type HttpTab = {
  id: string
  draft: RequestDraft
  dirty: boolean
  response: HttpSendResponse | null
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

const {
  panelStyle: responsePanelStyle,
  startResize: startResponseResize,
  resetHeight: resetResponseHeight,
} = useWorkbenchBottomPanelResize({
  defaultHeight: 320,
  bodyClass: 'http-workbench--resizing-panel',
})

const methods = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE'] as const
// 请求体编辑器的 Monaco 通用配置：关 minimap、关行号装饰，给工作台更紧凑的视觉
const bodyEditorOptions = {
  minimap: { enabled: false },
  lineNumbers: 'on',
  fontSize: 12,
  scrollBeyondLastLine: false,
  automaticLayout: true,
  wordWrap: 'on',
  tabSize: 2,
}
const groups = ref<HttpRequestGroup[]>([])
const requests = ref<HttpRequest[]>([])
const tabs = ref<HttpTab[]>([])
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
const activeConfigTab = ref('params')
const activeResponseTab = ref('body')
const search = ref('')
const loading = ref(false)
const loadingMore = ref(false)
const saving = ref(false)
const sending = ref(false)
const expandedGroups = ref(new Set<string>())
const pagination = reactive({ page: 1, pageSize: 50, total: 0 })
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
let requestListSequence = 0

const activeTab = computed(() => tabs.value.find((tab) => tab.id === activeTabId.value) || null)
const sourceMetaRows = computed(() => [
  { label: ui('类型', 'Type'), value: 'HTTP' },
  { label: ui('接口', 'Endpoints'), value: ui(`${pagination.total || 0} 个`, String(pagination.total || 0)) },
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
const activeHttpScheme = computed({
  get() {
    const url = activeTab.value?.draft.url || ''
    return url.toLowerCase().startsWith('http://') ? 'http' : 'https'
  },
  set(value: 'http' | 'https') {
    const tab = activeTab.value
    if (!tab) return
    tab.draft.url = replaceURLScheme(tab.draft.url, value)
  },
})

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
  searchTimer = window.setTimeout(() => loadRequests(1), 250)
})

onBeforeUnmount(() => {
  unregisterDraftChecker?.()
  window.clearTimeout(searchTimer)
})

const reloadWorkbench = async () => {
  loading.value = true
  try {
    const groupRes = await dataAPI.getHttpRequestGroups(props.projectId, props.connection.id)
    groups.value = groupRes.list || []
    await loadRequests(1)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, ui('加载 HTTP 工作台失败', 'Failed to load HTTP workbench')))
  } finally {
    loading.value = false
  }
}

const loadRequests = async (page = 1, append = false) => {
  const requestSequence = ++requestListSequence
  const projectId = props.projectId
  const connectionId = props.connection.id
  if (append) loadingMore.value = true
  else loadingMore.value = false
  const params: Record<string, any> = {
    page,
    pageSize: pagination.pageSize,
    q: search.value || undefined,
  }
  try {
    const res = await dataAPI.getHttpRequests(projectId, connectionId, params)
    if (
      requestSequence !== requestListSequence ||
      projectId !== props.projectId ||
      connectionId !== props.connection.id
    ) {
      return
    }
    const next = res.list || []
    if (append) {
      const byId = new Map(requests.value.map((item) => [String(item.id), item]))
      next.forEach((item) => byId.set(String(item.id), item))
      requests.value = Array.from(byId.values())
    } else {
      requests.value = next
    }
    pagination.page = res.pagination?.page || page
    pagination.pageSize = res.pagination?.pageSize || pagination.pageSize
    pagination.total = res.pagination?.total ?? requests.value.length
  } finally {
    if (append && requestSequence === requestListSequence) loadingMore.value = false
  }
}

const loadMoreRequests = () => {
  if (loadingMore.value || requests.value.length >= pagination.total) return
  void loadRequests(pagination.page + 1, true)
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
    response: null,
  })
  activeTabId.value = id
}

const createDraftRequest = () => {
  const id = `new-${Date.now()}`
  // 每次新建接口默认命名 "接口 N"，N 取所有已存在接口中 "接口 N" 形式的最大编号 + 1，避免重名冲突
  const draft = createEmptyDraft(generateUniqueDraftName())
  tabs.value.push({ id, draft, dirty: true, response: null })
  activeTabId.value = id
}

const activateTab = async (id: string) => {
  activeTabId.value = id
}

// 新建但未保存的 draft：没有 id、名字是默认「接口 N」、method/url/auth/bodyType/params/headers/Body 等都还是初始值。
// 这种"全新空白"草稿关闭时不应弹"未保存"提示，避免点 × 体验割裂。
const isPristineNewDraft = (draft: RequestDraft): boolean => {
  if (draft.id) return false
  // 名字被用户改过就不再算"全新空白"，关闭时仍需提示
  if (draft.name && !/^(?:接口\s+|Endpoint_)\d+$/.test(draft.name)) return false
  if (draft.url && draft.url !== 'https://') return false
  if (draft.params.length > 0) return false
  if (draft.headers.length > 0) return false
  if (draft.bodyType !== 'none') return false
  if (draft.auth.type !== 'none') return false
  const formBody = (draft.body as { form?: unknown[] }).form
  if (Array.isArray(formBody) && formBody.length > 0) return false
  if (draft.settings.timeoutMs !== 5000) return false
  if (draft.settings.followRedirects !== true) return false
  if (draft.settings.tlsVerify !== true) return false
  return true
}

const closeTab = async (id: string) => {
  const index = tabs.value.findIndex((tab) => tab.id === id)
  if (index < 0) return
  const tab = tabs.value[index]
  // 已保存过的接口有 dirty 改动：必须确认放弃；新建且未输入任何内容的空白草稿直接关闭
  if (tab.dirty && !isPristineNewDraft(tab.draft)) {
    try {
      await ElMessageBox.confirm(ui('当前接口有未保存改动，关闭后会丢失。', 'This endpoint has unsaved changes that will be lost.'), ui('关闭接口', 'Close Endpoint'), {
        confirmButtonText: ui('放弃', 'Discard'),
        cancelButtonText: ui('取消', 'Cancel'),
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
    ElMessage.success(ui('接口已保存', 'Endpoint saved'))
    return saved
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, ui('保存接口失败', 'Failed to save endpoint')))
    return null
  } finally {
    saving.value = false
  }
}

const saveDirtyTabs = async (): Promise<boolean> => {
  const dirtyIds = tabs.value
    .filter((tab) => tab.dirty && !isPristineNewDraft(tab.draft))
    .map((tab) => tab.id)
  for (const id of dirtyIds) {
    activeTabId.value = id
    if (!(await saveActive())) return false
  }
  return true
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
    activeResponseTab.value = 'body'
    await loadRequests()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, ui('发送 HTTP 请求失败', 'Failed to send HTTP request')))
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
    ElMessage.warning(ui('分组名称不能为空', 'Group name is required'))
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
    ElMessage.error(getApiErrorMessage(error, ui('保存分组失败', 'Failed to save group')))
  } finally {
    groupSaving.value = false
  }
}

const duplicateRequest = (request: HttpRequest) => {
  const draft = toDraft(request)
  draft.id = undefined
  draft.name = `${draft.name} Copy`
  tabs.value.push({ id: `copy-${Date.now()}`, draft, dirty: true, response: null })
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

const emitContextAction = (
  action: 'open' | 'duplicate' | 'move' | 'delete' | 'create-child' | 'edit',
) => {
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
    await ElMessageBox.confirm(ui('删除接口后，对应数据点会标记为失效。', 'Deleting the endpoint marks its data points as invalid.'), ui('删除接口', 'Delete Endpoint'), {
      confirmButtonText: ui('删除', 'Delete'),
      cancelButtonText: ui('取消', 'Cancel'),
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
      ElMessage.error(getApiErrorMessage(error, ui('删除接口失败', 'Failed to delete endpoint')))
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
    ElMessage.info(ui('已在未保存的接口中调整分组，保存接口后生效', 'Group changed in the draft; save the endpoint to apply'))
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
    ElMessage.success(ui('接口已移动', 'Endpoint moved'))
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, ui('移动接口失败', 'Failed to move endpoint')))
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
    ElMessage.success(ui('分组已移动', 'Group moved'))
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, ui('移动分组失败', 'Failed to move group')))
  } finally {
    moveSaving.value = false
  }
}

const deleteGroup = async (group: HttpRequestGroupNode) => {
  try {
    await ElMessageBox.confirm(
      ui(`确认删除分组「${group.name}」？组内接口会回到根目录。`, `Delete group “${group.name}”? Its endpoints will return to Root.`),
      ui('删除分组', 'Delete Group'),
      {
        confirmButtonText: ui('删除', 'Delete'),
        cancelButtonText: ui('取消', 'Cancel'),
        type: 'warning',
      },
    )
    await dataAPI.deleteHttpRequestGroup(props.projectId, String(group.id))
    const groupRes = await dataAPI.getHttpRequestGroups(props.projectId, props.connection.id)
    groups.value = groupRes.list || []
    await loadRequests(1)
    ElMessage.success(ui('分组已删除', 'Group deleted'))
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(getApiErrorMessage(error, ui('删除分组失败', 'Failed to delete group')))
    }
  }
}

const groupName = (groupId?: string | null) => {
  if (!groupId) return ui('根目录', 'Root')
  return groups.value.find((group) => String(group.id) === String(groupId))?.name || ui('根目录', 'Root')
}

// 把 HTTP 响应码映射为家族化 Pill 的 tone：
// 2xx → success、3xx → info、4xx/5xx → danger，1xx/网络异常等其它情况 → warning
const responseStatusTone = computed<'success' | 'info' | 'warning' | 'danger'>(() => {
  const status = activeTab.value?.response?.status
  if (status === undefined || status === null) return 'warning'
  if (status >= 200 && status < 300) return 'success'
  if (status >= 300 && status < 400) return 'info'
  if (status >= 400) return 'danger'
  return 'warning'
})

const responseStatusLabel = computed(() => {
  const response = activeTab.value?.response
  if (!response) return ''
  return `${response.status} ${response.statusText || ''}`.trim()
})

const formatTime = (value?: string | null) => {
  if (!value) return '-'
  return String(value).replace('T', ' ').slice(0, 19)
}

// 通用请求头默认值：仅在新建时预填，提示用户常见协商项，但不实际发送直到用户确认 value
const DEFAULT_REQUEST_HEADERS: HttpKeyValueRow[] = [
  { enabled: true, key: 'Accept', value: 'application/json', description: ui('期望的响应内容类型', 'Expected response content type') },
  { enabled: true, key: 'User-Agent', value: '', description: ui('客户端标识，可留空使用默认值', 'Client identifier; leave empty to use the default') },
  { enabled: true, key: 'Cache-Control', value: 'no-cache', description: ui('缓存策略', 'Cache policy') },
]

const replaceURLScheme = (rawUrl: string, scheme: string) => {
  const url = rawUrl.trim()
  if (/^[a-z][a-z\d+\-.]*:\/\//i.test(url)) {
    return url.replace(/^[a-z][a-z\d+\-.]*:\/\//i, `${scheme}://`)
  }
  return `${scheme}://${url}`
}

const createEmptyDraft = (name = ui('新建接口', 'New Endpoint')): RequestDraft => ({
  name,
  method: 'GET',
  url: 'https://',
  params: [],
  headers: DEFAULT_REQUEST_HEADERS.map((row) => ({ ...row })),
  auth: { type: 'none' },
  bodyType: 'none',
  body: {},
  settings: { timeoutMs: 5000, followRedirects: true, tlsVerify: true },
  enabled: true,
  sortOrder: 0,
  dataPointPath: '',
  outputs: [createWholeSourceOutput('body', ui('完整响应', 'Complete Response'), 'object')],
})

// 生成新建接口的默认名称 "接口 N"：N 取所有 "接口 N" 形式名称中最大的编号 + 1，
// 避免连续新建产生同名接口，也保证编号尽量紧凑（不出现"接口 99"但中间有大量空号的情况）。
const generateUniqueDraftName = (): string => {
  const taken = new Set<string>()
  for (const request of requests.value) {
    if (request.name) taken.add(request.name)
  }
  for (const tab of tabs.value) {
    if (tab.draft.name) taken.add(tab.draft.name)
  }
  let maxIndex = 0
  for (const name of taken) {
    const match = /^(?:接口\s+|Endpoint_)(\d+)$/.exec(name)
    if (match) {
      const n = Number(match[1])
      if (Number.isFinite(n) && n > maxIndex) maxIndex = n
    }
  }
  let next = maxIndex + 1
  const prefix = datacenterLocale.value === 'en' ? 'Endpoint_' : '接口 '
  while (taken.has(`${prefix}${next}`)) next += 1
  return `${prefix}${next}`
}

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
    items.sort(
      (left, right) =>
        (left.sortOrder || 0) - (right.sortOrder || 0) || left.name.localeCompare(right.name),
    )
  const sortGroups = (items: HttpRequestGroupNode[]) => {
    items.sort(
      (left, right) =>
        (left.sortOrder || 0) - (right.sortOrder || 0) || left.name.localeCompare(right.name),
    )
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
  const visit = (
    group: HttpRequestGroupNode,
    depth: number,
  ): Array<{ id: string; label: string }> => {
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
  settings: {
    timeoutMs: 5000,
    followRedirects: true,
    tlsVerify: true,
    ...(request.settings || {}),
  },
  enabled: request.enabled,
  sortOrder: request.sortOrder || 0,
  dataPointPath: request.dataPointPath || '',
  outputs: request.outputs.map((output) => ({ ...output, selector: { ...output.selector } })),
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
  outputs: draft.outputs,
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
  return (
    node.requests.length +
    node.children.reduce((sum, child) => sum + countHttpGroupRequests(child), 0)
  )
}
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

.http-workbench__load-more {
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

.http-workbench__load-more:hover:not(:disabled) {
  border-color: var(--dc-primary);
  background: var(--dc-primary-soft);
}

.http-workbench__load-more:disabled {
  cursor: wait;
  opacity: 0.65;
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
  color: var(--dc-text-muted);
}

.http-workbench__method.is-get {
  color: var(--dc-success);
}

.http-workbench__method.is-post {
  color: var(--dc-primary);
}

.http-workbench__method.is-put,
.http-workbench__method.is-patch {
  color: var(--dc-warning);
}

.http-workbench__method.is-delete {
  color: var(--dc-danger);
}

.http-workbench__pager {
  padding: 8px;
  border-top: 1px solid var(--dc-border);
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
  color: var(--dc-text-muted);
}

.http-workbench__tree :deep(.http-workbench__method.is-get) {
  color: var(--dc-success);
}

.http-workbench__tree :deep(.http-workbench__method.is-post) {
  color: var(--dc-primary);
}

.http-workbench__tree :deep(.http-workbench__method.is-put),
.http-workbench__tree :deep(.http-workbench__method.is-patch) {
  color: var(--dc-warning);
}

.http-workbench__tree :deep(.http-workbench__method.is-delete) {
  color: var(--dc-danger);
}

.http-workbench__main {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--dc-surface-raised);
}

.http-workbench__tabs {
  height: 36px;
  display: flex;
  align-items: end;
  gap: 2px;
  padding: 0 10px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-muted);
  overflow-x: auto;
}

.http-workbench__tab {
  height: 32px;
  max-width: 180px;
  min-width: 110px;
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 0 8px;
  border: 1px solid transparent;
  border-bottom: 0;
  background: transparent;
  border-radius: 6px 6px 0 0;
  color: var(--dc-text-secondary);
}

.http-workbench__tab.is-active {
  background: var(--dc-surface-raised);
  border-color: var(--dc-border);
  color: var(--dc-text);
}

.http-workbench__tab-name {
  min-width: 0;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
}

.http-workbench__tab i {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--dc-warning);
  flex-shrink: 0;
}

.http-workbench__tab svg {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
}

.http-workbench__editor {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--dc-surface-raised);
}

.http-workbench__crumb-row,
.http-workbench__name-row,
.http-workbench__request-line {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  border-bottom: 1px solid var(--dc-border);
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
  color: var(--dc-text-muted);
  font-size: 13px;
}

.http-workbench__crumb svg {
  width: 13px;
  height: 13px;
}

.http-workbench__crumb strong {
  color: var(--dc-text);
}

.http-workbench__name-input {
  width: 220px;
}

.http-workbench__name-input :deep(.el-input__wrapper) {
  padding: 0 8px;
  background: transparent;
  box-shadow: none;
  font-weight: 700;
  color: var(--dc-text);
}

.http-workbench__name-input :deep(.el-input__wrapper:hover),
.http-workbench__name-input :deep(.el-input__wrapper.is-focus) {
  background: var(--dc-surface-muted);
  box-shadow: 0 0 0 1px var(--dc-border-strong) inset;
}

.http-workbench__actions {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 8px;
}

.http-workbench__datapoint-path {
  max-width: min(360px, 34vw);
  color: var(--dc-text-muted);
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.http-workbench__icon-btn {
  padding: 6px 8px;
}

.http-workbench__icon-btn :deep(.el-icon),
.http-workbench__icon-btn :deep(svg) {
  width: 14px;
  height: 14px;
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

.http-workbench__protocol-select {
  width: 92px;
  flex: 0 0 92px;
}

.http-workbench__config-tabs {
  flex: 1;
  min-height: 0;
  padding: 0 14px;
}

/* 配置区与响应区 tab 标签字号对齐 method(11px) + 紧凑字距 */
.http-workbench__config-tabs :deep(.el-tabs__item),
.http-workbench__response :deep(.el-tabs__item) {
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.02em;
  padding: 0 12px !important;
  height: 32px;
  line-height: 32px;
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

/* 配置区表单样式：el-form 紧凑布局 */
.http-workbench__form {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 8px 0 4px;
  max-width: 520px;
}

.http-workbench__form :deep(.el-form-item) {
  margin-bottom: 14px;
}

.http-workbench__form :deep(.el-form-item__label) {
  font-weight: 700;
  color: var(--dc-text-secondary);
  font-size: 12px;
  padding-bottom: 4px;
  line-height: 1.4;
}

.http-workbench__form-control {
  width: 100%;
}

.http-workbench__settings-checks {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
}

/* Monaco 编辑器容器：加边框对齐家族化卡片风格 */
.http-workbench__monaco {
  height: 240px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  overflow: hidden;
}

.http-workbench__body-mode {
  padding: 8px 0;
}

.http-workbench__response {
  position: relative;
  flex: 0 0 auto;
  min-height: 160px;
  display: flex;
  flex-direction: column;
  border-top: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
}

.http-workbench__response-resizer {
  position: absolute;
  top: -4px;
  left: 0;
  z-index: 3;
  width: 100%;
  height: 8px;
  cursor: row-resize;
}

.http-workbench__response-resizer::before {
  content: '';
  position: absolute;
  top: 3px;
  left: 50%;
  width: 52px;
  height: 2px;
  border-radius: 999px;
  background: color-mix(in oklch, var(--dc-text-muted) 42%, transparent);
  transform: translateX(-50%);
}

.http-workbench__response-resizer:hover::before {
  background: var(--dc-primary);
}

:global(body.http-workbench--resizing-panel) {
  cursor: row-resize;
  user-select: none;
}

.http-workbench__response-head {
  flex: 0 0 auto;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 14px;
  border-bottom: 1px solid var(--dc-border);
}

.http-workbench__response-meta {
  display: flex;
  align-items: center;
  gap: 14px;
  font-size: 12px;
  color: var(--dc-text-muted);
}

.http-workbench__response-meta .is-success {
  color: var(--dc-success);
}

.http-workbench__response-meta .is-warning {
  color: var(--dc-warning);
}

.http-workbench__response-meta .is-danger {
  color: var(--dc-danger);
}

.http-workbench__response :deep(.el-tabs) {
  min-height: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 0 14px;
}

.http-workbench__response :deep(.el-tabs__content) {
  min-height: 0;
  flex: 1;
}

.http-workbench__response :deep(.el-tab-pane) {
  height: 100%;
  overflow: auto;
}

.http-workbench__response pre {
  min-height: 120px;
  height: 100%;
  margin: 0;
  padding: 12px;
  overflow: auto;
  border: 1px solid var(--dc-border);
  border-radius: 6px;
  background: #0f172a;
  color: #dbeafe;
  font-size: 12px;
  line-height: 1.55;
  box-sizing: border-box;
}

.http-workbench__response-table {
  min-height: 120px;
  height: 100%;
}

.http-workbench__response-empty,
.http-workbench__placeholder,
.http-workbench__empty,
.http-workbench__loading {
  min-height: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--dc-text-muted);
}

.http-workbench__placeholder {
  flex: 1;
  background: var(--dc-surface-raised);
}

.http-workbench__placeholder svg,
.http-workbench__response-empty svg {
  width: 28px;
  height: 28px;
}

.http-workbench__placeholder strong {
  color: var(--dc-text);
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
  border: 1px solid var(--dc-border);
  border-radius: 4px;
  background: var(--dc-surface-raised);
}

.http-kv-editor table {
  width: 100%;
  border-collapse: collapse;
}

.http-kv-editor td {
  height: 34px;
  border: 1px solid var(--dc-border);
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
