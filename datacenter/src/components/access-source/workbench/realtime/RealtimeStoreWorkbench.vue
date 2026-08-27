<template>
  <section class="realtime-store">
    <aside class="realtime-store__sidebar">
      <WorkbenchSourceHeader
        :title="connection.name || titleFallback"
        :fallback-title="titleFallback"
        :meta="sourceMetaRows"
        @back="$emit('back')"
      >
        <template #actions>
          <el-input
            v-model="keyword"
            class="realtime-store__search"
            size="small"
            clearable
            placeholder="搜索 Key"
            @keyup.enter="loadKeys"
          />
          <button
            v-if="showBatchCreateFilteredButton"
            type="button"
            class="workbench-source-header__icon-action"
            title="为筛选结果创建数据点"
            aria-label="为筛选结果创建数据点"
            @click="batchCreateFilteredDatapoints"
          >
            <IconTablerDatabasePlus />
          </button>
          <button
            type="button"
            class="workbench-source-header__icon-action is-primary"
            title="新建 Key"
            aria-label="新建 Key"
            @click="createDraftKey"
          >
            <IconTablerPlus />
          </button>
          <button
            type="button"
            class="workbench-source-header__icon-action"
            title="刷新"
            aria-label="刷新"
            @click="loadKeys"
          >
            <IconTablerRefresh />
          </button>
        </template>
      </WorkbenchSourceHeader>

      <div ref="keyListRef" class="realtime-store__tree" v-loading="loadingKeys">
        <button
          v-for="node in visibleTreeNodes"
          :key="node.id"
          type="button"
          class="realtime-store__tree-node"
          :class="{
            'is-folder': node.kind === 'folder',
            'is-key': node.kind === 'key',
            'is-active': node.fullKey && activeTabKey === node.fullKey,
          }"
          :style="{ '--tree-depth': node.depth }"
          @click="handleTreeNodeClick(node)"
          @contextmenu.stop.prevent="handleTreeNodeContextMenu(node, $event)"
        >
          <span class="realtime-store__tree-toggle">
            <IconTablerChevronDown v-if="node.kind === 'folder' && isTreeNodeExpanded(node.id)" />
            <IconTablerChevronRight v-else-if="node.kind === 'folder'" />
          </span>
          <IconTablerFolder v-if="node.kind === 'folder'" class="realtime-store__tree-icon" />
          <IconTablerFile v-else class="realtime-store__tree-icon" />
          <span class="realtime-store__tree-label" :title="node.fullKey || node.label">
            {{ node.label }}
          </span>
          <IconTablerCircleCheck
            v-if="node.kind === 'key' && node.dataPointPath"
            class="realtime-store__datapoint-badge"
            :title="node.dataPointPath"
          />
          <em v-if="node.kind === 'folder'">({{ node.count }})</em>
          <small v-else>{{ node.type }}</small>
        </button>
        <el-empty
          v-if="!loadingKeys && visibleTreeNodes.length === 0"
          description="暂无 Key"
          :image-size="72"
        />
      </div>
      <button
        v-if="hasMoreKeys"
        type="button"
        class="realtime-store__load-more"
        :disabled="loadingKeys || keyLimit >= 10000"
        @click="loadMoreKeys"
      >
        {{ keyLimit >= 10000 ? '已显示前 10000 个 Key' : '加载更多 Key' }}
      </button>
    </aside>

    <main class="realtime-store__main">
      <header class="realtime-store__tab-strip">
        <button
          type="button"
          class="realtime-store__tab-arrow"
          title="向左滚动"
          :disabled="tabs.length <= 1"
          @click="scrollTabs(-1)"
        >
          <IconTablerChevronLeft />
        </button>
        <div ref="tabListRef" class="realtime-store__tabs">
          <button
            v-for="tab in tabs"
            :key="tab.key"
            :ref="(element) => setTabElementRef(tab.key, element)"
            type="button"
            class="realtime-store__tab"
            :class="{ 'is-active': tab.key === activeTabKey }"
            :title="formatTabTitle(tab)"
            @click="activeTabKey = tab.key"
          >
            <IconTablerKey class="realtime-store__tab-icon" />
            <span>{{ formatTabTitle(tab) }}</span>
            <i v-if="tab.dirty" />
            <IconTablerX class="realtime-store__tab-close" @click.stop="closeTab(tab.key)" />
          </button>
        </div>
        <button
          type="button"
          class="realtime-store__tab-arrow"
          title="向右滚动"
          :disabled="tabs.length <= 1"
          @click="scrollTabs(1)"
        >
          <IconTablerChevronRight />
        </button>
      </header>

      <section v-if="activeTab" class="realtime-store__editor">
        <div class="realtime-store__toolbar">
          <div class="realtime-store__identity">
            <span class="realtime-store__type-badge">{{ activeTab.draft.type }}</span>
            <el-input
              v-model="activeTab.draft.key"
              class="realtime-store__key-input"
              :readonly="Boolean(activeTab.draft.originalKey)"
              placeholder="device:line1:status"
              @input="markDirty"
            />
            <el-button
              v-if="activeTab.draft.originalKey"
              class="realtime-store__rename"
              title="重命名 Key"
              aria-label="重命名 Key"
              @click="openActiveRenameDialog"
            >
              <IconTablerEdit />
            </el-button>
            <el-select
              v-model="activeTab.draft.type"
              class="realtime-store__type"
              aria-label="Key 类型"
              :disabled="activeTab.draft.type === 'stream'"
              @change="handleTypeChange"
            >
              <el-option v-for="type in editableTypes" :key="type" :label="type" :value="type" />
              <el-option v-if="activeTab.draft.type === 'stream'" label="stream" value="stream" />
            </el-select>
            <label class="realtime-store__ttl-field">
              <span>TTL（秒）</span>
              <el-input-number
                v-model="activeTab.draft.ttlSeconds"
                class="realtime-store__ttl"
                aria-label="TTL 秒数，0 表示不过期"
                :min="0"
                controls-position="right"
                @change="handleTtlChange"
              />
            </label>
          </div>
          <div class="realtime-store__actions">
            <el-button title="刷新" :loading="loadingValue" @click="reloadActive">
              <IconTablerRefresh />
            </el-button>
            <el-button
              type="primary"
              title="保存"
              :disabled="activeTab.draft.type === 'stream'"
              :loading="saving"
              @click="saveActive"
            >
              <IconTablerDeviceFloppy />
            </el-button>
          </div>
        </div>

        <div class="realtime-store__meta">
          <span>{{ providerLabel }}</span>
          <span>TTL {{ formatTTL(activeTab.draft.ttlSeconds) }}</span>
          <span v-if="activeTab.draft.dataPointPath"
            >数据点 {{ activeTab.draft.dataPointPath }}</span
          >
          <span v-else>未创建数据点</span>
        </div>

        <div class="realtime-store__content">
          <section
            v-if="activeTab.draft.type === 'string'"
            class="realtime-store__value-panel realtime-store__value-panel--editor"
          >
            <header class="realtime-store__panel-header">
              <div>
                <strong>Value</strong>
                <el-select
                  v-model="activeTab.draft.valueType"
                  class="realtime-store__view-mode"
                  size="small"
                  @change="handleStringViewModeChange"
                >
                  <el-option
                    v-for="option in stringViewModeOptions"
                    :key="option.value"
                    :label="option.label"
                    :value="option.value"
                  />
                </el-select>
                <span>{{ stringSizeLabel }}</span>
              </div>
              <div class="realtime-store__panel-actions">
                <el-button
                  size="small"
                  :disabled="!activeStringCanFormat"
                  @click="formatActiveString"
                >
                  格式化
                </el-button>
                <el-button size="small" title="复制" @click="copyText(activeTab.draft.stringValue)">
                  <IconTablerCopy />
                </el-button>
              </div>
            </header>
            <MonacoEditor
              v-model="activeTab.draft.stringValue"
              class="realtime-store__monaco"
              :language="activeStringLanguage"
              theme="vs"
              height="100%"
              :options="monacoOptions"
              @change="markDirty"
              @save="saveActive"
            />
          </section>

          <section v-else-if="activeTab.draft.type === 'hash'" class="realtime-store__value-panel">
            <header class="realtime-store__panel-header">
              <div>
                <strong>Hash Fields</strong>
                <span>{{ activeTab.draft.rows.length }} 项</span>
              </div>
              <div class="realtime-store__panel-actions">
                <el-input
                  v-model="rowKeyword"
                  class="realtime-store__row-search"
                  size="small"
                  clearable
                  placeholder="搜索 field / value"
                >
                  <template #prefix>
                    <IconTablerSearch />
                  </template>
                </el-input>
                <el-button size="small" type="primary" @click="addRow">
                  <IconTablerPlus />
                  新增行
                </el-button>
              </div>
            </header>
            <el-table :data="visibleEditorRows" height="calc(100% - 42px)" border size="small">
              <el-table-column label="Field" min-width="180">
                <template #default="{ row }">
                  <el-input v-model="row.key" size="small" @input="markDirty" />
                </template>
              </el-table-column>
              <el-table-column label="Value" min-width="260">
                <template #default="{ row }">
                  <el-input v-model="row.value" size="small" @input="markDirty" />
                </template>
              </el-table-column>
              <el-table-column label="操作" width="112" align="center">
                <template #default="{ row }">
                  <el-button link type="primary" title="源码查看" @click="openRowSource(row)">
                    <IconTablerCode />
                  </el-button>
                  <el-button link type="danger" title="删除" @click="removeRow(row)">
                    <IconTablerTrash />
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
          </section>

          <section
            v-else-if="['list', 'set'].includes(activeTab.draft.type)"
            class="realtime-store__value-panel"
          >
            <header class="realtime-store__panel-header">
              <div>
                <strong>{{
                  activeTab.draft.type === 'list' ? 'List Items' : 'Set Members'
                }}</strong>
                <span>{{ activeTab.draft.rows.length }} 项</span>
              </div>
              <div class="realtime-store__panel-actions">
                <el-input
                  v-model="rowKeyword"
                  class="realtime-store__row-search"
                  size="small"
                  clearable
                  placeholder="搜索 value"
                >
                  <template #prefix>
                    <IconTablerSearch />
                  </template>
                </el-input>
                <el-button size="small" type="primary" @click="addRow">
                  <IconTablerPlus />
                  新增行
                </el-button>
              </div>
            </header>
            <el-table :data="visibleEditorRows" height="calc(100% - 42px)" border size="small">
              <el-table-column label="#" width="72">
                <template #default="{ row }">{{ getRowIndex(row) }}</template>
              </el-table-column>
              <el-table-column label="Value" min-width="300">
                <template #default="{ row }">
                  <el-input v-model="row.value" size="small" @input="markDirty" />
                </template>
              </el-table-column>
              <el-table-column label="操作" width="112" align="center">
                <template #default="{ row }">
                  <el-button link type="primary" title="源码查看" @click="openRowSource(row)">
                    <IconTablerCode />
                  </el-button>
                  <el-button link type="danger" title="删除" @click="removeRow(row)">
                    <IconTablerTrash />
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
          </section>

          <section v-else-if="activeTab.draft.type === 'zset'" class="realtime-store__value-panel">
            <header class="realtime-store__panel-header">
              <div>
                <strong>ZSet Members</strong>
                <span>{{ activeTab.draft.rows.length }} 项</span>
              </div>
              <div class="realtime-store__panel-actions">
                <el-input
                  v-model="rowKeyword"
                  class="realtime-store__row-search"
                  size="small"
                  clearable
                  placeholder="搜索 member / score"
                >
                  <template #prefix>
                    <IconTablerSearch />
                  </template>
                </el-input>
                <el-button size="small" type="primary" @click="addRow">
                  <IconTablerPlus />
                  新增行
                </el-button>
              </div>
            </header>
            <el-table :data="visibleEditorRows" height="calc(100% - 42px)" border size="small">
              <el-table-column label="Member" min-width="240">
                <template #default="{ row }">
                  <el-input v-model="row.member" size="small" @input="markDirty" />
                </template>
              </el-table-column>
              <el-table-column label="Score" width="180">
                <template #default="{ row }">
                  <el-input-number
                    v-model="row.score"
                    size="small"
                    controls-position="right"
                    @change="markDirty"
                  />
                </template>
              </el-table-column>
              <el-table-column label="操作" width="112" align="center">
                <template #default="{ row }">
                  <el-button link type="primary" title="源码查看" @click="openRowSource(row)">
                    <IconTablerCode />
                  </el-button>
                  <el-button link type="danger" title="删除" @click="removeRow(row)">
                    <IconTablerTrash />
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
          </section>

          <pre v-else class="realtime-store__readonly">{{ formattedReadonlyValue }}</pre>
        </div>
      </section>

      <el-empty v-else class="realtime-store__empty-main" description="选择或新建一个 Key" />
    </main>

    <el-dialog v-model="renameDialog.visible" title="重命名 Key" width="420px">
      <el-input v-model="renameDialog.newKey" placeholder="新的 Key 名称" />
      <template #footer>
        <el-button @click="renameDialog.visible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="confirmRename">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="datapointDialog.visible" title="创建数据点" width="420px">
      <p class="realtime-store__dialog-tip">
        数据点类型会根据 Key 类型和值自动推断；String 的 JSON 内容会识别为对象或数组。
      </p>
      <template #footer>
        <el-button @click="datapointDialog.visible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="confirmCreateDatapoint">
          创建
        </el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="sourceDialog.visible" :title="sourceDialog.title" width="720px">
      <div class="realtime-store__source-viewer">
        <MonacoEditor
          v-model="sourceDialog.content"
          :language="sourceDialog.language"
          theme="vs"
          height="360px"
          :read-only="sourceDialog.readOnly"
          :options="sourceMonacoOptions"
        />
      </div>
      <template #footer>
        <el-button @click="copyText(sourceDialog.content)">复制</el-button>
        <el-button type="primary" @click="sourceDialog.visible = false">关闭</el-button>
      </template>
    </el-dialog>

    <Teleport to="body">
      <div
        v-if="keyContextMenu.visible"
        class="realtime-store__menu-mask"
        @click="closeKeyContextMenu"
        @contextmenu.prevent="closeKeyContextMenu"
      >
        <div
          class="realtime-store__context-menu"
          :style="{ left: keyContextMenu.x + 'px', top: keyContextMenu.y + 'px' }"
          @click.stop
        >
          <button
            v-if="keyContextMenu.kind === 'key'"
            type="button"
            @click="openContextKeyInNewTab"
          >
            <IconTablerKey />
            <span>打开 Key</span>
          </button>
          <button
            v-if="keyContextMenu.kind === 'key'"
            type="button"
            @click="createContextKeyDatapoint"
          >
            <IconTablerDatabasePlus />
            <span>创建数据点</span>
          </button>
          <button v-if="keyContextMenu.kind === 'key'" type="button" @click="renameContextKey">
            <IconTablerEdit />
            <span>重命名 Key</span>
          </button>
          <button v-if="keyContextMenu.kind === 'key'" type="button" @click="deleteContextKey">
            <IconTablerTrash />
            <span>删除 Key</span>
          </button>
          <button
            v-if="keyContextMenu.kind === 'folder'"
            type="button"
            @click="batchCreateContextGroupDatapoints"
          >
            <IconTablerFolder />
            <span>为该分组创建数据点</span>
          </button>
        </div>
      </div>
    </Teleport>
  </section>
</template>

<script setup lang="ts">
import { computed, inject, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import type { ComponentPublicInstance } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import dataAPI from '@/api/data.api'
import { getApiErrorMessage } from '@/utils/request'
import MonacoEditor from '@/components/MonacoEditor.vue'
import WorkbenchSourceHeader from '@/components/workbench/WorkbenchSourceHeader.vue'
import IconTablerChevronDown from '~icons/tabler/chevron-down'
import IconTablerChevronLeft from '~icons/tabler/chevron-left'
import IconTablerChevronRight from '~icons/tabler/chevron-right'
import IconTablerCode from '~icons/tabler/code'
import IconTablerCircleCheck from '~icons/tabler/circle-check'
import IconTablerCopy from '~icons/tabler/copy'
import IconTablerDatabasePlus from '~icons/tabler/database-plus'
import IconTablerDeviceFloppy from '~icons/tabler/device-floppy'
import IconTablerEdit from '~icons/tabler/edit'
import IconTablerFile from '~icons/tabler/file'
import IconTablerFolder from '~icons/tabler/folder'
import IconTablerKey from '~icons/tabler/key'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerSearch from '~icons/tabler/search'
import IconTablerTrash from '~icons/tabler/trash'
import IconTablerX from '~icons/tabler/x'

type AccessSourceConnection = {
  id: string
  name?: string
  type?: string
  config?: Record<string, any>
}

type KeySummary = {
  id?: string
  key: string
  type: string
  ttl: number
  size: number
  provider: string
  valueType?: string
  dataPointId?: string
  dataPointPath?: string
}

type KeyGroup = {
  name: string
  count: number
}

type KeyTreeNode = {
  id: string
  label: string
  kind: 'folder' | 'key'
  depth: number
  count: number
  fullKey?: string
  type?: string
  dataPointPath?: string
  children: KeyTreeNode[]
}

type EditorRow = {
  key?: string
  value?: string
  member?: string
  score?: number
}

type KeyDraft = {
  originalKey: string
  key: string
  type: string
  ttlSeconds: number
  valueType: string
  stringValue: string
  rows: EditorRow[]
  readonlyValue: unknown
  dataPointPath: string
}

type KeyTab = {
  key: string
  dirty: boolean
  draft: KeyDraft
}

const props = defineProps<{
  connection: AccessSourceConnection
  projectId: string
}>()

defineEmits<{
  (event: 'back'): void
}>()

const editableTypes = ['string', 'hash', 'list', 'set', 'zset']
const writableKeyPattern = /^[\p{L}\p{N}_:-]+$/u
const utf8Length = (value: string) => new TextEncoder().encode(value).length
const validateWritableKey = (value: string): string => {
  if (!value) return 'Key 不能为空'
  if (value !== value.trim()) return 'Key 不能包含首尾空格'
  if (utf8Length(value) > 256) return 'Key 长度不能超过 256 字节'
  if (value.startsWith(':') || value.endsWith(':') || value.includes('::')) {
    return 'Key 的层级分隔符 : 之间不能为空'
  }
  if (!writableKeyPattern.test(value)) {
    return 'Key 只能包含中文、字母、数字、冒号、下划线和短横线'
  }
  return ''
}
const stringViewModeOptions = [
  { label: 'Text', value: 'text' },
  { label: 'JSON', value: 'json' },
]
const keyword = ref('')
const keys = ref<KeySummary[]>([])
const groups = ref<KeyGroup[]>([])
const keyLimit = ref(500)
const hasMoreKeys = ref(false)
const tabs = ref<KeyTab[]>([])
const registerDraftChecker =
  inject<
    (guard: {
      isDirty: () => boolean
      save: () => Promise<boolean>
      discard: () => void
    }) => () => void
  >('registerDraftChecker')
let unregisterDraftChecker: (() => void) | undefined
const activeTabKey = ref('')
const loadingKeys = ref(false)
const loadingValue = ref(false)
const saving = ref(false)
const keyListRef = ref<HTMLElement | null>(null)
const tabListRef = ref<HTMLElement | null>(null)
const tabElementRefs = new Map<string, HTMLElement>()
const expandedTreeNodeIds = ref<Set<string>>(new Set())
const rowKeyword = ref('')
let searchTimer: ReturnType<typeof setTimeout> | null = null

const renameDialog = reactive({ visible: false, key: '', newKey: '' })
const datapointDialog = reactive({ visible: false })
const keyContextMenu = reactive({
  visible: false,
  kind: 'key' as 'key' | 'folder',
  key: '',
  nodeId: '',
  x: 0,
  y: 0,
})
const sourceDialog = reactive({
  visible: false,
  title: '源码查看',
  content: '',
  language: 'plaintext',
  readOnly: false,
})
const monacoOptions = {
  minimap: { enabled: false },
  fontSize: 13,
  lineNumbers: 'on',
  wordWrap: 'on',
  folding: true,
  renderLineHighlight: 'line',
}
const sourceMonacoOptions = {
  ...monacoOptions,
  readOnly: true,
}

const config = computed(() => props.connection.config || {})
const isBuiltin = computed(() => props.connection.type === 'builtin.realtime')
const titleFallback = computed(() => (isBuiltin.value ? 'IF实时库' : 'Redis'))
const providerLabel = computed(() => (isBuiltin.value ? 'IF 实时库' : 'Redis'))
const sourceMetaRows = computed(() => {
  if (isBuiltin.value) {
    return [
      { label: '类型', value: 'IF实时库' },
      { label: '命名空间', value: String(config.value.runtimeKey || props.connection.id) },
    ]
  }
  return [
    { label: '类型', value: 'Redis' },
    { label: '地址', value: String(config.value.address || '-') },
    { label: 'DB', value: String(config.value.db ?? config.value.database ?? 0) },
  ]
})
const filteredKeys = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  return keys.value.filter((item) => {
    const keywordMatched = !q || item.key.toLowerCase().includes(q) || item.type.includes(q)
    return keywordMatched
  })
})
const keyTree = computed(() => buildKeyTree(filteredKeys.value))
const visibleTreeNodes = computed(() => flattenKeyTree(keyTree.value, expandedTreeNodeIds.value))
const showBatchCreateFilteredButton = computed(
  () => keyword.value.trim() !== '' && filteredKeys.value.length > 0,
)

const activeTab = computed(() => tabs.value.find((tab) => tab.key === activeTabKey.value) || null)
const formattedReadonlyValue = computed(() =>
  JSON.stringify(activeTab.value?.draft.readonlyValue ?? null, null, 2),
)
const activeStringIsJson = computed(() => isJsonText(activeTab.value?.draft.stringValue || ''))
const activeStringViewMode = computed(() =>
  activeTab.value?.draft.valueType === 'json' ? 'json' : 'text',
)
const activeStringLanguage = computed(() =>
  activeStringViewMode.value === 'json' ? 'json' : 'plaintext',
)
const activeStringCanFormat = computed(
  () => activeStringViewMode.value === 'json' && activeStringIsJson.value,
)
const stringSizeLabel = computed(() => formatByteSize(activeTab.value?.draft.stringValue || ''))
const visibleEditorRows = computed(() => {
  const rows = activeTab.value?.draft.rows || []
  const keyword = rowKeyword.value.trim().toLowerCase()
  if (!keyword) return rows
  return rows.filter((row) =>
    [row.key, row.value, row.member, row.score]
      .map((value) => String(value ?? '').toLowerCase())
      .some((value) => value.includes(keyword)),
  )
})

const loadKeys = async () => {
  loadingKeys.value = true
  try {
    const response = await dataAPI.getRealtimeStoreKeys(props.projectId, props.connection.id, {
      q: keyword.value || undefined,
      limit: keyLimit.value,
    })
    keys.value = response?.data?.list || []
    groups.value = response?.data?.groups || []
    hasMoreKeys.value = Boolean(response?.data?.hasMore)
    syncExpandedTreeNodeIds()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载实时库 Key 失败'))
  } finally {
    loadingKeys.value = false
  }
}

const scheduleLoadKeys = () => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    keyLimit.value = 500
    loadKeys()
  }, 300)
}

const loadMoreKeys = async () => {
  keyLimit.value = Math.min(10000, keyLimit.value + 500)
  await loadKeys()
}

const handleTreeNodeClick = async (node: KeyTreeNode) => {
  if (node.kind === 'folder') {
    const next = new Set(expandedTreeNodeIds.value)
    if (next.has(node.id)) next.delete(node.id)
    else next.add(node.id)
    expandedTreeNodeIds.value = next
    await nextTick()
    return
  }
  if (node.fullKey) await openKey(node.fullKey)
}

const handleTreeNodeContextMenu = (node: KeyTreeNode, event: MouseEvent) => {
  if (node.kind === 'key' && !node.fullKey) return
  keyContextMenu.visible = true
  keyContextMenu.kind = node.kind
  keyContextMenu.key = node.fullKey || ''
  keyContextMenu.nodeId = node.id
  keyContextMenu.x = event.clientX
  keyContextMenu.y = event.clientY
}

const openContextKeyInNewTab = async () => {
  const key = keyContextMenu.key
  keyContextMenu.visible = false
  if (!key) return
  await openKey(key)
}

const closeKeyContextMenu = () => {
  keyContextMenu.visible = false
}

const createContextKeyDatapoint = async () => {
  const key = keyContextMenu.key
  keyContextMenu.visible = false
  if (!key) return
  await createSingleDatapoint(key)
}

const renameContextKey = () => {
  const key = keyContextMenu.key
  keyContextMenu.visible = false
  if (!key) return
  renameDialog.key = key
  renameDialog.newKey = key
  renameDialog.visible = true
}

const deleteContextKey = async () => {
  const key = keyContextMenu.key
  keyContextMenu.visible = false
  if (!key) return
  const tab = tabs.value.find((item) => item.draft.originalKey === key || item.draft.key === key)
  await deleteKey(key, tab?.key)
}

const batchCreateContextGroupDatapoints = async () => {
  const nodeId = keyContextMenu.nodeId
  keyContextMenu.visible = false
  const node = findTreeNode(keyTree.value, nodeId)
  if (!node) return
  await batchCreateDatapoints(collectLeafKeys(node.children), '该分组')
}

const batchCreateFilteredDatapoints = async () => {
  await batchCreateDatapoints(
    filteredKeys.value.map((item) => item.key),
    '筛选结果',
  )
}

const isTreeNodeExpanded = (id: string) => expandedTreeNodeIds.value.has(id)

const syncExpandedTreeNodeIds = () => {
  const folderIds = collectKeyTreeFolderIds(buildKeyTree(keys.value))
  if (keyword.value.trim()) {
    expandedTreeNodeIds.value = folderIds
    return
  }
  const next = new Set<string>()
  for (const id of expandedTreeNodeIds.value) {
    if (folderIds.has(id)) next.add(id)
  }
  expandedTreeNodeIds.value = next
}

const createDraftKey = async () => {
  const key = `__draft__${Date.now()}`
  const tab = {
    key,
    dirty: true,
    draft: {
      originalKey: '',
      key: '',
      type: 'string',
      ttlSeconds: 0,
      valueType: 'json',
      stringValue: '{}',
      rows: [],
      readonlyValue: null,
      dataPointPath: '',
    },
  }
  tabs.value.push(tab)
  activeTabKey.value = key
  scrollActiveTabIntoView()
}

const openKey = async (key: string) => {
  const existing = tabs.value.find((tab) => tab.draft.originalKey === key || tab.draft.key === key)
  if (existing) {
    moveTabToEnd(existing)
    activeTabKey.value = existing.key
    scrollActiveTabIntoView()
    return
  }
  loadingValue.value = true
  try {
    const response = await dataAPI.getRealtimeStoreKey(props.projectId, props.connection.id, key)
    const draft = createDraftFromResponse(response?.data || {})
    tabs.value.push({ key, dirty: false, draft })
    activeTabKey.value = key
    scrollActiveTabIntoView()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '读取实时库 Key 失败'))
  } finally {
    loadingValue.value = false
  }
}

const reloadActive = async () => {
  if (!activeTab.value?.draft.key) return
  const key = activeTab.value.draft.originalKey || activeTab.value.draft.key
  await openOrReplaceKey(key)
}

const openOrReplaceKey = async (key: string) => {
  loadingValue.value = true
  try {
    const response = await dataAPI.getRealtimeStoreKey(props.projectId, props.connection.id, key)
    const draft = createDraftFromResponse(response?.data || {})
    const tab = activeTab.value
    if (tab) {
      tab.draft = draft
      tab.key = draft.key
      tab.dirty = false
      activeTabKey.value = draft.key
      scrollActiveTabIntoView()
    }
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '刷新实时库 Key 失败'))
  } finally {
    loadingValue.value = false
  }
}

const saveActive = async () => {
  const tab = activeTab.value
  if (!tab) return false
  const keyError = validateWritableKey(tab.draft.key)
  if (keyError) {
    ElMessage.warning(keyError)
    return false
  }
  saving.value = true
  try {
    if (tab.draft.originalKey && tab.draft.key !== tab.draft.originalKey) {
      ElMessage.warning('已存在 Key 请使用重命名操作')
      return false
    }
    const payload = {
      key: tab.draft.key,
      type: tab.draft.type,
      ttlSeconds: tab.draft.ttlSeconds,
      valueType: tab.draft.valueType || 'object',
      value: buildSaveValue(tab.draft),
    }
    const response = await dataAPI.saveRealtimeStoreKey(
      props.projectId,
      props.connection.id,
      payload,
    )
    const draft = createDraftFromResponse(response?.data || {})
    let autoDatapointError = ''
    if (!draft.dataPointPath) {
      try {
        const datapointResponse = await dataAPI.createRealtimeStoreKeyDatapoint(
          props.projectId,
          props.connection.id,
          draft.key,
          {},
        )
        draft.dataPointPath = datapointResponse?.data?.dataPointPath || ''
      } catch (error) {
        autoDatapointError = getApiErrorMessage(error, '自动生成数据点失败')
      }
    }
    tab.draft = draft
    tab.key = draft.key
    tab.dirty = false
    activeTabKey.value = draft.key
    scrollActiveTabIntoView()
    await loadKeys()
    if (autoDatapointError) {
      ElMessage.warning(`Key 已保存，但${autoDatapointError}`)
    } else {
      ElMessage.success(draft.dataPointPath ? 'Key 已保存，数据点已生成' : 'Key 已保存')
    }
    return true
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '保存实时库 Key 失败'))
    return false
  } finally {
    saving.value = false
  }
}

const saveDirtyTabs = async (): Promise<boolean> => {
  const dirtyKeys = tabs.value.filter((tab) => tab.dirty).map((tab) => tab.key)
  for (const key of dirtyKeys) {
    activeTabKey.value = key
    if (!(await saveActive())) return false
  }
  return true
}

const closeTab = async (key: string) => {
  const tab = tabs.value.find((item) => item.key === key)
  if (tab?.dirty) {
    await ElMessageBox.confirm('当前 Key 有未保存改动，关闭后会丢失。', '关闭 Key', {
      type: 'warning',
    })
  }
  const index = tabs.value.findIndex((item) => item.key === key)
  if (index >= 0) tabs.value.splice(index, 1)
  if (activeTabKey.value === key) {
    activeTabKey.value = tabs.value[Math.max(0, index - 1)]?.key || ''
    scrollActiveTabIntoView()
  }
}

const moveTabToEnd = (tab: KeyTab) => {
  const index = tabs.value.indexOf(tab)
  if (index < 0 || index === tabs.value.length - 1) return
  tabs.value.splice(index, 1)
  tabs.value.push(tab)
}

const confirmRename = async () => {
  const key = renameDialog.key || activeTab.value?.draft.originalKey || activeTab.value?.draft.key
  if (!key) return
  const keyError = validateWritableKey(renameDialog.newKey)
  if (keyError) {
    ElMessage.warning(keyError)
    return
  }
  saving.value = true
  try {
    const response = await dataAPI.renameRealtimeStoreKey(
      props.projectId,
      props.connection.id,
      key,
      {
        newKey: renameDialog.newKey,
      },
    )
    const draft = createDraftFromResponse(response?.data || {})
    const tab = tabs.value.find((item) => item.draft.originalKey === key || item.draft.key === key)
    if (tab) {
      tab.draft = draft
      tab.key = draft.key
      tab.dirty = false
    }
    activeTabKey.value = draft.key
    renameDialog.visible = false
    scrollActiveTabIntoView()
    await loadKeys()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '重命名 Key 失败'))
  } finally {
    saving.value = false
  }
}

const openActiveRenameDialog = () => {
  const key = activeTab.value?.draft.originalKey || activeTab.value?.draft.key
  if (!key) return
  renameDialog.key = key
  renameDialog.newKey = key
  renameDialog.visible = true
}

const confirmCreateDatapoint = async () => {
  const tab = activeTab.value
  if (!tab?.draft.key) return
  await createSingleDatapoint(tab.draft.originalKey || tab.draft.key)
}

const createSingleDatapoint = async (key: string, options: { silent?: boolean } = {}) => {
  saving.value = true
  try {
    const response = await dataAPI.createRealtimeStoreKeyDatapoint(
      props.projectId,
      props.connection.id,
      key,
      {},
    )
    const tab = activeTab.value
    if (tab && (tab.draft.originalKey === key || tab.draft.key === key)) {
      tab.draft.dataPointPath = response?.data?.dataPointPath || ''
    }
    datapointDialog.visible = false
    await loadKeys()
    if (!options.silent) ElMessage.success('数据点已创建')
  } catch (error) {
    if (!options.silent) ElMessage.error(getApiErrorMessage(error, '创建数据点失败'))
  } finally {
    saving.value = false
  }
}

const batchCreateDatapoints = async (targetKeys: string[], label: string) => {
  const dedupedKeys = [...new Set(targetKeys)].filter((key) => key.trim() !== '')
  if (dedupedKeys.length === 0) {
    ElMessage.warning('没有可创建数据点的 Key')
    return
  }
  saving.value = true
  try {
    const response = await dataAPI.batchCreateRealtimeStoreKeyDatapoints(
      props.projectId,
      props.connection.id,
      {
        keys: dedupedKeys,
      },
    )
    const data = response?.data || {}
    await loadKeys()
    ElMessage.success(
      `${label}数据点创建完成：新建 ${data.created || 0} 个，已存在 ${data.exists || 0} 个，失败 ${data.failed || 0} 个`,
    )
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '批量创建数据点失败'))
  } finally {
    saving.value = false
  }
}

const deleteKey = async (key: string, tabKey?: string) => {
  await ElMessageBox.confirm('删除 Key 后，对应数据点会标记为失效。', '删除 Key', {
    type: 'warning',
  })
  saving.value = true
  try {
    await dataAPI.deleteRealtimeStoreKey(props.projectId, props.connection.id, key)
    tabs.value = tabs.value.filter((item) => item.key !== (tabKey || key) && item.draft.key !== key)
    if (activeTabKey.value === (tabKey || key)) activeTabKey.value = tabs.value.at(-1)?.key || ''
    await loadKeys()
    ElMessage.success('Key 已删除')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '删除 Key 失败'))
  } finally {
    saving.value = false
  }
}

const markDirty = () => {
  if (activeTab.value) activeTab.value.dirty = true
}

const setTabElementRef = (key: string, element: Element | ComponentPublicInstance | null) => {
  if (element instanceof HTMLElement) {
    tabElementRefs.set(key, element)
    return
  }
  tabElementRefs.delete(key)
}

const scrollActiveTabIntoView = () => {
  nextTick(() => {
    tabElementRefs.get(activeTabKey.value)?.scrollIntoView({
      block: 'nearest',
      inline: 'nearest',
    })
  })
}

const scrollTabs = (direction: number) => {
  const element = tabListRef.value
  if (!element) return
  element.scrollBy({
    left: direction * Math.max(240, element.clientWidth * 0.7),
    behavior: 'smooth',
  })
}

const formatTabTitle = (tab: KeyTab) => {
  return getKeyDisplayName(tab.draft.key || '新建 Key')
}

const getKeyDisplayName = (key: string) => {
  const segments = key.split(':').filter((segment) => segment.trim() !== '')
  return segments.at(-1) || key
}

const handleTypeChange = () => {
  const draft = activeTab.value?.draft
  if (!draft) return
  draft.rows = []
  draft.valueType = draft.type === 'string' ? 'json' : 'object'
  draft.stringValue = draft.type === 'string' ? '{}' : ''
  rowKeyword.value = ''
  markDirty()
}

const handleTtlChange = () => {
  markDirty()
}

const handleStringViewModeChange = (value: string | number | boolean) => {
  const draft = activeTab.value?.draft
  if (!draft) return
  draft.valueType = value === 'json' ? 'json' : 'text'
  if (draft.valueType === 'json') {
    draft.stringValue = formatJsonTextIfValid(draft.stringValue)
  }
  markDirty()
}

const addRow = () => {
  const draft = activeTab.value?.draft
  if (!draft) return
  if (draft.type === 'zset') draft.rows.push({ member: '', score: 0 })
  else if (draft.type === 'hash') draft.rows.push({ key: '', value: '' })
  else draft.rows.push({ value: '' })
  markDirty()
}

const removeRow = (row: EditorRow) => {
  const rows = activeTab.value?.draft.rows
  if (!rows) return
  const index = rows.indexOf(row)
  if (index < 0) return
  rows.splice(index, 1)
  markDirty()
}

const getRowIndex = (row: EditorRow) => {
  const rows = activeTab.value?.draft.rows || []
  const index = rows.indexOf(row)
  return index >= 0 ? index : '-'
}

const openRowSource = (row: EditorRow) => {
  const source = row.value ?? row.member ?? ''
  sourceDialog.title = row.key ? `Field: ${row.key}` : '源码查看'
  sourceDialog.content = formatSourceText(source)
  sourceDialog.language = isJsonText(source) ? 'json' : 'plaintext'
  sourceDialog.readOnly = false
  sourceDialog.visible = true
  nextTick(() => {
    sourceDialog.readOnly = true
  })
}

const formatActiveString = () => {
  const draft = activeTab.value?.draft
  if (!draft || !isJsonText(draft.stringValue)) return
  draft.stringValue = JSON.stringify(JSON.parse(draft.stringValue), null, 2)
  markDirty()
}

const copyText = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text || '')
    ElMessage.success('已复制')
  } catch {
    ElMessage.error('复制失败')
  }
}

const createDraftFromResponse = (data: any): KeyDraft => {
  const type = data.type || 'string'
  const stringValue = type === 'string' ? stringifyEditorValue(data.value) : ''
  const valueType = normalizeStringViewMode(type, data.valueType, stringValue)
  return {
    originalKey: data.key || '',
    key: data.key || '',
    type,
    ttlSeconds: normalizeRealtimeTTL(data.ttl),
    valueType,
    stringValue: valueType === 'json' ? formatJsonTextIfValid(stringValue) : stringValue,
    rows: rowsFromValue(type, data.value),
    readonlyValue: data.value,
    dataPointPath: data.dataPointPath || '',
  }
}

const rowsFromValue = (type: string, value: any): EditorRow[] => {
  if (type === 'hash' && value && typeof value === 'object' && !Array.isArray(value)) {
    return Object.entries(value).map(([key, rowValue]) => ({
      key,
      value: stringifyEditorValue(rowValue),
    }))
  }
  if (['list', 'set'].includes(type) && Array.isArray(value)) {
    return value.map((item) => ({ value: stringifyEditorValue(item) }))
  }
  if (type === 'zset' && Array.isArray(value)) {
    return value.map((item) => ({
      member: stringifyEditorValue(item?.member),
      score: Number(item?.score || 0),
    }))
  }
  return []
}

const buildSaveValue = (draft: KeyDraft) => {
  if (draft.type === 'string') return draft.stringValue
  if (draft.type === 'hash') {
    return draft.rows
      .filter((row) => row.key)
      .map((row) => ({ key: row.key, value: row.value || '' }))
  }
  if (['list', 'set'].includes(draft.type)) return draft.rows.map((row) => row.value || '')
  if (draft.type === 'zset') {
    return draft.rows
      .filter((row) => row.member)
      .map((row) => ({ member: row.member, score: Number(row.score || 0) }))
  }
  return draft.readonlyValue
}

const stringifyEditorValue = (value: unknown) => {
  if (typeof value === 'string') return value
  if (value == null) return ''
  return JSON.stringify(value)
}

const isJsonText = (value: unknown) => {
  if (typeof value !== 'string') return false
  const text = value.trim()
  if (!text || !['{', '['].includes(text[0])) return false
  try {
    JSON.parse(text)
    return true
  } catch {
    return false
  }
}

const normalizeStringViewMode = (type: string, valueType: unknown, stringValue: string) => {
  if (type !== 'string') return String(valueType || 'object')
  if (valueType === 'json' || valueType === 'text') return valueType
  return isJsonText(stringValue) ? 'json' : 'text'
}

const formatSourceText = (value: unknown) => {
  const text = stringifyEditorValue(value)
  return formatJsonTextIfValid(text)
}

const formatJsonTextIfValid = (value: string) => {
  if (!isJsonText(value)) return value
  return JSON.stringify(JSON.parse(value), null, 2)
}

const formatByteSize = (value: string) => {
  const bytes = new Blob([value || '']).size
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

const formatTTL = (ttl: number) => {
  if (ttl <= 0) return '不过期'
  return `${ttl}s`
}

const normalizeRealtimeTTL = (ttl: unknown) => {
  const value = Number(ttl ?? -1)
  return value <= 0 ? 0 : value
}

const buildKeyTree = (items: KeySummary[]) => {
  const roots: KeyTreeNode[] = []
  const folderById = new Map<string, KeyTreeNode>()
  const sortedItems = [...items].sort((left, right) => left.key.localeCompare(right.key))

  for (const item of sortedItems) {
    const segments = item.key.split(':').filter((segment) => segment.trim() !== '')
    if (segments.length <= 1) {
      roots.push(createKeyTreeLeaf(item, item.key, 'key:' + item.key, 0))
      continue
    }

    let currentChildren = roots
    let path = ''
    for (let index = 0; index < segments.length - 1; index++) {
      const segment = segments[index]
      path = path ? `${path}:${segment}` : segment
      let folder = folderById.get('folder:' + path)
      if (!folder) {
        folder = {
          id: 'folder:' + path,
          label: segment,
          kind: 'folder',
          depth: index,
          count: 0,
          children: [],
        }
        folderById.set(folder.id, folder)
        currentChildren.push(folder)
      }
      folder.count += 1
      currentChildren = folder.children
    }

    currentChildren.push(
      createKeyTreeLeaf(
        item,
        segments[segments.length - 1],
        'key:' + item.key,
        segments.length - 1,
      ),
    )
  }

  return sortKeyTreeNodes(roots)
}

const createKeyTreeLeaf = (
  item: KeySummary,
  label: string,
  id: string,
  depth: number,
): KeyTreeNode => ({
  id,
  label,
  kind: 'key',
  depth,
  count: 0,
  fullKey: item.key,
  type: item.type,
  dataPointPath: item.dataPointPath,
  children: [],
})

const sortKeyTreeNodes = (nodes: KeyTreeNode[]) =>
  nodes
    .sort((left, right) => {
      if (left.kind !== right.kind) return left.kind === 'folder' ? -1 : 1
      return left.label.localeCompare(right.label)
    })
    .map((node) => {
      if (node.kind === 'folder') node.children = sortKeyTreeNodes(node.children)
      return node
    })

const flattenKeyTree = (nodes: KeyTreeNode[], expandedIds: Set<string>) => {
  const result: KeyTreeNode[] = []
  const walk = (items: KeyTreeNode[]) => {
    for (const node of items) {
      result.push(node)
      if (node.kind === 'folder' && expandedIds.has(node.id)) {
        walk(node.children)
      }
    }
  }
  walk(nodes)
  return result
}

const collectKeyTreeFolderIds = (nodes: KeyTreeNode[]) => {
  const result = new Set<string>()
  const walk = (items: KeyTreeNode[]) => {
    for (const node of items) {
      if (node.kind !== 'folder') continue
      result.add(node.id)
      walk(node.children)
    }
  }
  walk(nodes)
  return result
}

const findTreeNode = (nodes: KeyTreeNode[], id: string): KeyTreeNode | null => {
  for (const node of nodes) {
    if (node.id === id) return node
    const child = findTreeNode(node.children, id)
    if (child) return child
  }
  return null
}

const collectLeafKeys = (nodes: KeyTreeNode[]) => {
  const result: string[] = []
  const walk = (items: KeyTreeNode[]) => {
    for (const node of items) {
      if (node.kind === 'key' && node.fullKey) result.push(node.fullKey)
      else walk(node.children)
    }
  }
  walk(nodes)
  return result
}

onMounted(() => {
  loadKeys()
  unregisterDraftChecker = registerDraftChecker?.({
    isDirty: () => tabs.value.some((tab) => tab.dirty),
    save: saveDirtyTabs,
    discard: () => tabs.value.forEach((tab) => (tab.dirty = false)),
  })
  window.addEventListener('click', closeKeyContextMenu)
  window.addEventListener('scroll', closeKeyContextMenu, true)
})

watch(keyword, () => {
  scheduleLoadKeys()
})

onBeforeUnmount(() => {
  unregisterDraftChecker?.()
  if (searchTimer) clearTimeout(searchTimer)
  window.removeEventListener('click', closeKeyContextMenu)
  window.removeEventListener('scroll', closeKeyContextMenu, true)
})
</script>

<style scoped>
.realtime-store {
  height: 100%;
  min-height: 0;
  display: grid;
  grid-template-columns: 300px minmax(0, 1fr);
  overflow: hidden;
  background: #f6f8fb;
  color: #172033;
}

.realtime-store__sidebar {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border-right: 1px solid #dfe5ee;
  background: #fff;
}

.realtime-store__search {
  width: 132px;
}

.realtime-store__tree-node,
.realtime-store__tab,
.realtime-store__tab-arrow {
  border: 0;
  background: transparent;
  font: inherit;
  color: inherit;
  cursor: pointer;
}

.realtime-store__tree {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 8px 6px;
}

.realtime-store__load-more {
  height: 36px;
  flex: 0 0 36px;
  border: 0;
  border-top: 1px solid #e5ebf3;
  background: #fff;
  color: #2563eb;
  cursor: pointer;
}

.realtime-store__load-more:hover:not(:disabled) {
  background: #f7faff;
}

.realtime-store__load-more:disabled {
  cursor: default;
  opacity: 0.55;
}

.realtime-store__tree-node {
  width: 100%;
  height: 24px;
  display: grid;
  grid-template-columns: 14px 16px minmax(0, 1fr) 14px auto;
  align-items: center;
  gap: 4px;
  padding: 0 6px 0 calc(4px + var(--tree-depth, 0) * 18px);
  border-radius: 4px;
  color: #4c5b70;
  text-align: left;
}

.realtime-store__actions svg,
.realtime-store__tab svg {
  width: 16px;
  height: 16px;
}

.realtime-store__tree-toggle {
  width: 14px;
  height: 14px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: #6b7688;
}

.realtime-store__tree-toggle svg {
  width: 13px;
  height: 13px;
}

.realtime-store__tree-icon {
  width: 15px;
  height: 15px;
  color: #7a8594;
}

.realtime-store__tree-label,
.realtime-store__tab span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.realtime-store__tree-node em {
  font-style: normal;
  color: #7f8ca3;
  font-size: 12px;
}

.realtime-store__tree-node small {
  justify-self: end;
  color: #8a95a5;
  font-size: 11px;
}

.realtime-store__datapoint-badge {
  width: 13px;
  height: 13px;
  color: #16a34a;
}

.realtime-store__tree-node:hover,
.realtime-store__tree-node.is-active {
  background: #eef2f7;
}

.realtime-store__tree-node.is-active {
  color: #1d4ed8;
}

.realtime-store__tree-node.is-active .realtime-store__tree-icon {
  color: #1d4ed8;
}

.realtime-store__main {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.realtime-store__tab-strip {
  height: 36px;
  min-width: 0;
  display: flex;
  align-items: stretch;
  border-bottom: 1px solid #dfe5ee;
  background: #fff;
}

.realtime-store__tabs {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: stretch;
  overflow-x: auto;
  overflow-y: hidden;
  scrollbar-width: none;
}

.realtime-store__tabs::-webkit-scrollbar {
  display: none;
}

.realtime-store__tab-arrow {
  width: 28px;
  flex: 0 0 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-right: 1px solid #e5ebf3;
  color: #9aa6b5;
}

.realtime-store__tab-arrow:last-child {
  border-right: 0;
  border-left: 1px solid #e5ebf3;
}

.realtime-store__tab-arrow:hover:not(:disabled) {
  color: #1d4ed8;
  background: #f7faff;
}

.realtime-store__tab-arrow:disabled {
  cursor: default;
  opacity: 0.45;
}

.realtime-store__tab {
  height: 35px;
  width: clamp(150px, 16vw, 230px);
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
  justify-content: space-between;
  gap: 7px;
  padding: 0 9px;
  border-right: 1px solid #dfe5ee;
  color: #344054;
}

.realtime-store__tab.is-active {
  background: #f7faff;
  color: #2684ff;
  font-weight: 600;
}

.realtime-store__tab-icon,
.realtime-store__tab-close {
  flex: 0 0 auto;
}

.realtime-store__tab-icon {
  color: currentColor;
}

.realtime-store__tab-close {
  color: #97a3b4;
}

.realtime-store__tab:hover .realtime-store__tab-close {
  color: #64748b;
}

.realtime-store__tab i {
  flex: 0 0 auto;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #f59e0b;
}

.realtime-store__editor {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding: 12px;
}

.realtime-store__toolbar {
  min-height: 48px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 10px;
  border: 1px solid #dfe5ee;
  border-radius: 6px 6px 0 0;
  background: #fff;
}

.realtime-store__identity {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
}

.realtime-store__key-input {
  min-width: 160px;
  max-width: 520px;
  flex: 1 1 280px;
}

.realtime-store__rename {
  flex: 0 0 32px;
  width: 32px;
  min-width: 32px;
  padding: 0;
}

.realtime-store__type-badge {
  height: 28px;
  min-width: 58px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0 10px;
  border: 1px solid #cfd8e6;
  border-radius: 4px;
  background: #f8fafc;
  color: #1f3b57;
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
}

.realtime-store__type {
  width: 120px;
  flex: 0 0 120px;
}

.realtime-store__ttl-field {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 6px;
  color: #697589;
  font-size: 12px;
  white-space: nowrap;
}

.realtime-store__ttl {
  width: 128px;
}

.realtime-store__actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.realtime-store__actions :deep(.el-button) {
  min-width: 32px;
}

.realtime-store__meta {
  height: 30px;
  display: flex;
  align-items: center;
  gap: 16px;
  color: #697589;
  font-size: 12px;
}

.realtime-store__content {
  flex: 1;
  min-height: 0;
}

.realtime-store__value-panel {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  border: 1px solid #dfe5ee;
  border-radius: 0 0 6px 6px;
  background: #fff;
  overflow: hidden;
}

.realtime-store__value-panel--editor {
  background: #fbfcfe;
}

.realtime-store__panel-header {
  min-height: 42px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 6px 10px;
  border-bottom: 1px solid #e4e9f1;
  background: #f8fafc;
}

.realtime-store__panel-header > div:first-child {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
}

.realtime-store__panel-header strong {
  color: #172033;
  font-size: 13px;
}

.realtime-store__panel-header span {
  color: #728097;
  font-size: 12px;
}

.realtime-store__view-mode {
  width: 92px;
}

.realtime-store__panel-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.realtime-store__panel-actions svg {
  width: 15px;
  height: 15px;
}

.realtime-store__row-search {
  width: 220px;
}

.realtime-store__value-panel :deep(.el-table) {
  flex: 1;
  min-height: 0;
}

.realtime-store__value-panel :deep(.el-table__cell) {
  padding: 6px 0;
}

.realtime-store__value-panel :deep(.el-input__wrapper),
.realtime-store__toolbar :deep(.el-input__wrapper),
.realtime-store__toolbar :deep(.el-select__wrapper) {
  box-shadow: 0 0 0 1px #d8e0eb inset;
}

.realtime-store__monaco {
  flex: 1;
  min-height: 0;
}

.realtime-store__monaco :deep(.monaco-editor-container),
.realtime-store__source-viewer :deep(.monaco-editor-container) {
  min-height: 0;
}

.realtime-store__source-viewer {
  height: 360px;
  border: 1px solid #dfe5ee;
  border-radius: 6px;
  overflow: hidden;
}

.realtime-store__menu-mask {
  position: fixed;
  inset: 0;
  z-index: 2100;
}

.realtime-store__context-menu {
  position: fixed;
  min-width: 148px;
  padding: 4px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
}

.realtime-store__context-menu button {
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
  cursor: pointer;
}

.realtime-store__context-menu button:hover {
  background: var(--dc-surface-muted);
  color: var(--dc-primary);
}

.realtime-store__context-menu svg {
  width: 15px;
  height: 15px;
}

.realtime-store__dialog-tip {
  margin: 0;
  color: #4b5565;
  font-size: 13px;
  line-height: 1.7;
}

.realtime-store__readonly {
  height: 100%;
  overflow: auto;
  margin: 0;
  padding: 12px;
  border: 1px solid #dfe5ee;
  border-radius: 6px;
  background: #fff;
  color: #263244;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 13px;
}

.realtime-store__empty-main {
  flex: 1;
  min-height: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

@media (max-width: 960px) {
  .realtime-store {
    grid-template-columns: 260px minmax(0, 1fr);
  }

  .realtime-store__toolbar,
  .realtime-store__identity {
    align-items: stretch;
    flex-direction: column;
  }

  .realtime-store__actions {
    align-self: flex-start;
  }

  .realtime-store__key-input,
  .realtime-store__type,
  .realtime-store__ttl-field {
    width: 100%;
    max-width: none;
    flex-basis: auto;
  }

  .realtime-store__ttl {
    flex: 1;
    width: auto;
  }
}
</style>
