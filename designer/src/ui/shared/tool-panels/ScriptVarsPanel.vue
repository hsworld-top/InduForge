<!--
  ScriptVarsPanel - 脚本与变量面板
  管理系统脚本（启动/关闭）、定时器、变量变更、自定义脚本
-->
<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus'
import { storeToRefs } from 'pinia'
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import IconEpFolder from '~icons/ep/folder'
import IconEpList from '~icons/ep/list'
import {
  resolvePageVariableSnippet,
  resolveProjectVariableSnippet,
  resolveProjectVariableSourceLabel,
} from '@/services/data-variable-mapping'
import { useEditorStore } from '@/stores/editor-store'
import { buildComponentMethodCompletions } from '@/ui/shared/helpers/component-methods'
import ScriptEditorDialog from './ScriptEditorDialog.vue'
import ScriptVarsCustomSection from './ScriptVarsCustomSection.vue'
import ScriptVarsMetaFormDialog from './ScriptVarsMetaFormDialog.vue'
import ScriptVarsSystemScriptDialog from './ScriptVarsSystemScriptDialog.vue'
import ScriptVarsSystemSection from './ScriptVarsSystemSection.vue'
import ScriptVarsTimersSection from './ScriptVarsTimersSection.vue'
import ScriptVarsVariableChangesSection from './ScriptVarsVariableChangesSection.vue'
import VariableGroupFormDialog from './VariableGroupFormDialog.vue'

type SelectedModule = 'timers' | 'variableChanges' | 'custom'

const editorStore = useEditorStore()
const {
  projectId,
  globalScripts,
  projectVariables,
  projectVariableGroups,
  pages,
  doc,
  docVersion,
  currentPage,
} = storeToRefs(editorStore)
const maxGroupDepth = 5
const { t } = useI18n()

const activeSections = ref('system')
const selectedSystemKey = ref<string>('startup')

const selectedTimerId = ref('')
const selectedVariableId = ref('')
const selectedCustomId = ref('')
const selectedNodes = ref<Record<string, Array<any>>>({
  timers: [],
  variableChanges: [],
  custom: [],
})

const systemCode = ref('')
const editorCode = ref('')
const systemOriginalCode = ref('')
const editorOriginalCode = ref('')
const editorOriginalInterval = ref(1000)
const editorInterval = ref(1000)
const editorParams = ref('')
const systemEditorTab = ref('startup')
const scriptEditorTab = ref('')
const systemDrafts = ref<Record<string, string>>({})
const scriptDrafts = ref<Record<string, string>>({})

const systemEditorRef = ref<any>(null)
const activeEditorRef = ref<any>(null)

const scriptClipboard = ref<Record<string, any> | null>(null)

const groupDialogVisible = ref(false)
const groupEditMode = ref(false)
const groupDialogModule = ref<SelectedModule>('custom')
const groupId = ref('')
const groupName = ref('')
const groupParentId = ref<any>(null)

const metaDialogVisible = ref(false)
const metaDialogMode = ref<'create' | 'edit'>('create')
const metaDialogModule = ref<SelectedModule>('custom')
const metaForm = ref<Record<string, any>>({
  id: '',
  name: '',
  interval: 1000,
  description: '',
  variable: '',
  params: '',
  groupId: null,
})

const systemEditorVisible = ref(false)
const scriptEditorVisible = ref(false)
const scriptEditorModule = ref<SelectedModule>('custom')

const contextMenuVisible = ref(false)
const contextMenuPosition = ref({ x: 0, y: 0 })
const contextMenuNode = ref<any>(null)
const contextMenuModule = ref<SelectedModule>('custom')
const showMoveToMenu = ref(false)

const projectVariableNames = computed<Array<string>>(() =>
  Object.keys((projectVariables.value || {}) as Record<string, any>).sort(),
)
const projectVariablesList = computed<Array<any>>(() =>
  (
    Object.entries((projectVariables.value || {}) as Record<string, any>) as Array<[string, any]>
  ).map(([name, detail]) => ({
    name,
    groupId: detail?.groupId || null,
    meta: {
      ...(detail || {}),
      mapped: detail?.source?.type === 'dataCenter' || detail?.mapped === true,
    },
  })),
)
const customScripts = computed<Array<any>>(
  () => (globalScripts.value?.custom?.items || []) as Array<any>,
)
const customScriptGroups = computed<Array<any>>(
  () => (globalScripts.value?.custom?.groups || []) as Array<any>,
)
const variableGroups = computed<Array<any>>(() => (projectVariableGroups.value || []) as Array<any>)

const scriptSearch = ref('')
const pageSearch = ref('')
const pageVariableSearch = ref('')
const variableEnumVisible = ref(false)
const enumVariableSearch = ref('')
const enumGroupTreeRef = ref<any>(null)
const enumSelectedGroupId = ref<any>(null)
const enumSelectedVar = ref<any>(null)

/**
 * 统一消息调用形态，兼容当前项目的 Element Plus 类型约束。
 */
function showSuccess(message: string) {
  ;(ElMessage as any).success(message)
}

/**
 * 统一消息调用形态，兼容当前项目的 Element Plus 类型约束。
 */
function showWarning(message: string) {
  ;(ElMessage as any).warning(message)
}

/**
 * 统一消息调用形态，兼容当前项目的 Element Plus 类型约束。
 */
function showError(message: string) {
  ;(ElMessage as any).error(message)
}

const selectedSystemLabel = computed(() =>
  selectedSystemKey.value === 'startup'
    ? t('scriptPanel.sections.startup')
    : t('scriptPanel.sections.shutdown'),
)
const editorMetaTitle = computed(
  () => getSelectedItem(scriptEditorModule.value)?.name || t('scriptPanel.editor.untitledScript'),
)
const editorMetaDescription = computed(
  () =>
    getSelectedItem(scriptEditorModule.value)?.description || t('scriptPanel.editor.noDescription'),
)
const scriptEditorKindLabel = computed(() => {
  if (scriptEditorModule.value === 'timers') return t('scriptPanel.editor.timerScript')
  if (scriptEditorModule.value === 'variableChanges')
    return t('scriptPanel.editor.variableChangeScript')
  return t('scriptPanel.editor.customScript')
})
const systemEditorTabs = computed(() => [
  { key: 'startup', label: t('scriptPanel.sections.startup') },
  { key: 'shutdown', label: t('scriptPanel.sections.shutdown') },
])
const activeEditorItems = computed(() => getItemsByModule(scriptEditorModule.value))
const scriptEditorTabs = computed(() =>
  activeEditorItems.value.map((item) => ({
    key: item.id,
    label: item.name || item.variable || t('scriptPanel.editor.untitledScript'),
  })),
)

const customScriptSidebarTree = computed(() =>
  buildScriptTree(customScriptGroups.value, customScripts.value),
)
const pageSidebarTree = computed(() => buildPageTree(pages.value || []))
const pageVars = computed<Record<string, any>>(() => {
  void docVersion.value
  const pageId = currentPage.value?.id
  if (!pageId || !doc.value) return {}
  const vars = (doc.value as any).vars?.pages?.[pageId]
  return vars && typeof vars === 'object' ? vars : {}
})
const pageVariableRows = computed<Array<any>>(() => {
  const keyword = String(pageVariableSearch.value || '').toLowerCase()
  return Object.entries(pageVars.value)
    .map(([name, detail]: [string, any]) => ({
      name,
      type: detail?.type || 'string',
      description: detail?.description || '',
    }))
    .filter((item) => {
      if (!keyword) return true
      return String(item.name || '')
        .toLowerCase()
        .includes(keyword)
    })
})
const enumGroupTree = computed(() => [
  {
    id: 'all',
    label: t('scriptPanel.variableEnum.all'),
    type: 'group',
    children: buildGroupTree(variableGroups.value),
  },
])
const enumVariableRows = computed<Array<any>>(() => {
  const keyword = String(enumVariableSearch.value || '').toLowerCase()
  return projectVariablesList.value
    .filter((item) => {
      if (enumSelectedGroupId.value) {
        return item.groupId === enumSelectedGroupId.value
      }
      return true
    })
    .filter((item) => {
      if (!keyword) return true
      return String(item.name || '')
        .toLowerCase()
        .includes(keyword)
    })
    .map((item) => ({
      name: item.name,
      type: item.meta?.type || 'string',
      description: item.meta?.description || '',
      mapped: !!item.meta?.mapped,
      sourceLabel: resolveProjectVariableSourceLabel(item.meta)
        ? `来自数据点：${resolveProjectVariableSourceLabel(item.meta)}`
        : '',
    }))
})

const pageComponentTree = computed<Array<any>>(() => {
  void docVersion.value
  const rootId = currentPage.value?.rootNodeId
  if (!rootId || !doc.value) return []
  const buildNode = (nodeId: any): any => {
    const node = (doc.value as any)?.getNode(nodeId)
    if (!node) return null
    const children = ((node.children || []) as Array<any>)
      .map((childId: any) => buildNode(childId))
      .filter(Boolean)
    const label = node.label || node.type || t('propertyPanel.bindingDialog.targetFallback')
    return {
      id: node.id,
      label,
      type: children.length ? 'group' : 'component',
      componentName: node.label || '',
      componentType: node.type || '',
      children,
    }
  }
  const root = buildNode(rootId)
  if (!root) return []
  return root.children?.length ? root.children : [root]
})

const jsCompletions = computed<Array<any>>(() => {
  const items: Array<any> = [
    {
      label: 'console.log',
      insertText: 'console.log()',
      kind: 'Function',
      detail: 'Log output',
    },
    {
      label: 'if',
      insertText: 'if () {\n  \n}',
      kind: 'Snippet',
      detail: 'if statement',
    },
    {
      label: 'for',
      insertText: 'for (let i = 0; i < ; i++) {\n  \n}',
      kind: 'Snippet',
      detail: 'for loop',
    },
    {
      label: 'function',
      insertText: 'function name() {\n  \n}',
      kind: 'Snippet',
      detail: 'function declaration',
    },
    { label: 'const', insertText: 'const ', kind: 'Keyword' },
    { label: 'let', insertText: 'let ', kind: 'Keyword' },
    { label: 'return', insertText: 'return ', kind: 'Keyword' },
  ]

  projectVariableNames.value.forEach((name) => {
    items.push({
      label: name,
      insertText: name,
      kind: 'Variable',
      detail: t('scriptPanel.messages.variableDetail'),
      prefix: '$global.',
    })
  })

  customScripts.value.forEach((script) => {
    if (!script?.name) return
    const params =
      typeof script.params === 'string' && script.params.trim()
        ? script.params.trim()
        : typeof script.args === 'string'
          ? script.args.trim()
          : ''
    const call = params ? `${script.name}(${params})` : `${script.name}()`
    items.push({
      label: script.name,
      insertText: call,
      kind: 'Function',
      detail: t('scriptPanel.messages.customScriptDetail'),
      prefix: 'customScripts.',
    })
  })

  items.push(...buildComponentMethodCompletions(pageComponentTree.value))

  return items
})

const timerGroups = computed<Array<any>>(
  () => (globalScripts.value?.timers?.groups || []) as Array<any>,
)
const variableChangeGroups = computed<Array<any>>(
  () => (globalScripts.value?.variableChanges?.groups || []) as Array<any>,
)
const customGroups = computed<Array<any>>(
  () => (globalScripts.value?.custom?.groups || []) as Array<any>,
)

const timerItems = computed<Array<any>>(
  () => (globalScripts.value?.timers?.items || []) as Array<any>,
)
const variableChangeItems = computed<Array<any>>(
  () => (globalScripts.value?.variableChanges?.items || []) as Array<any>,
)
const customItems = computed<Array<any>>(
  () => (globalScripts.value?.custom?.items || []) as Array<any>,
)

const selectedTimer = computed(() =>
  timerItems.value.find((item) => item.id === selectedTimerId.value),
)
const selectedVariableChange = computed(() =>
  variableChangeItems.value.find((item) => item.id === selectedVariableId.value),
)
const selectedCustom = computed(() =>
  customItems.value.find((item) => item.id === selectedCustomId.value),
)

const selectedTimerGroup = computed(() =>
  timerGroups.value.find((group) => group.id === selectedTimerId.value),
)
const selectedVariableGroup = computed(() =>
  variableChangeGroups.value.find((group) => group.id === selectedVariableId.value),
)
const selectedCustomGroup = computed(() =>
  customGroups.value.find((group) => group.id === selectedCustomId.value),
)

const timerTree = computed(() => buildScriptTree(timerGroups.value, timerItems.value))
const variableChangeTree = computed(() =>
  buildScriptTree(variableChangeGroups.value, variableChangeItems.value),
)
const customTree = computed(() => buildScriptTree(customGroups.value, customItems.value))

const contextMenuStyle = computed(() => ({
  left: `${contextMenuPosition.value.x}px`,
  top: `${contextMenuPosition.value.y}px`,
}))

const submenuStyle = computed(() => {
  const subWidth = 180
  const leftCandidate = contextMenuPosition.value.x + 180
  const left =
    leftCandidate + subWidth > window.innerWidth
      ? contextMenuPosition.value.x - subWidth
      : leftCandidate
  return {
    left: `${Math.max(8, left)}px`,
    top: `${contextMenuPosition.value.y}px`,
  }
})

const availableGroups = computed(() => {
  const module = contextMenuModule.value
  const current = contextMenuNode.value
  const groups = getGroupsByModule(module)
  if (!current || current.type !== 'group') return groups
  return groups.filter(
    (group) => group.id !== current.id && !isDescendantGroup(group.id, current.id, module),
  )
})

const groupParentOptions = computed(() => {
  const groups = getGroupsByModule(groupDialogModule.value)
  if (!groupEditMode.value) return groups
  return groups.filter(
    (group) =>
      group.id !== groupId.value &&
      !isDescendantGroup(group.id, groupId.value, groupDialogModule.value),
  )
})

const scriptEditorTitle = computed(() => {
  const module = scriptEditorModule.value || contextMenuModule.value || 'system'
  const selected = getSelectedItem(module)
  if (selected?.name) return `${selected.name}${t('scriptPanel.editor.genericScript')}`
  if (module === 'timers') return t('scriptPanel.editor.timerScript')
  if (module === 'variableChanges') return t('scriptPanel.editor.variableChangeScript')
  if (module === 'custom') return t('scriptPanel.editor.customScript')
  return t('scriptPanel.editor.genericScript')
})

const metaDialogTitle = computed(() => {
  if (metaDialogModule.value === 'timers') {
    return metaDialogMode.value === 'edit'
      ? t('scriptPanel.messages.editTimer')
      : t('scriptPanel.messages.createTimer')
  }
  if (metaDialogModule.value === 'variableChanges') {
    return metaDialogMode.value === 'edit'
      ? t('scriptPanel.messages.editVariableChange')
      : t('scriptPanel.messages.createVariableChange')
  }
  if (metaDialogModule.value === 'custom') {
    return metaDialogMode.value === 'edit'
      ? t('scriptPanel.messages.editCustom')
      : t('scriptPanel.messages.createCustom')
  }
  return metaDialogMode.value === 'edit'
    ? t('scriptPanel.messages.editScript')
    : t('scriptPanel.messages.createScript')
})

function syncSystemCode() {
  if (systemEditorVisible.value) return
  const system = globalScripts.value?.system || {}
  systemCode.value =
    selectedSystemKey.value === 'startup' ? system.startup?.code || '' : system.shutdown?.code || ''
}

function getSystemCodeByKey(key: string) {
  const system = globalScripts.value?.system || {}
  return key === 'startup' ? system.startup?.code || '' : system.shutdown?.code || ''
}

function getItemDraftKey(module: string, id: string) {
  return `${module}:${id}`
}

function getItemCodeById(module: string, id: string) {
  const item = getItemsByModule(module).find((row) => row.id === id)
  return item?.code || ''
}

watch(selectedSystemKey, syncSystemCode, { immediate: true })
watch(() => globalScripts.value?.system, syncSystemCode, { deep: true })

watch(systemEditorTab, (nextKey, prevKey) => {
  if (!systemEditorVisible.value) return
  if (prevKey) systemDrafts.value = { ...systemDrafts.value, [prevKey]: systemCode.value || '' }
  selectedSystemKey.value = nextKey
  systemCode.value = systemDrafts.value[nextKey] ?? getSystemCodeByKey(nextKey)
})

watch(scriptEditorTab, (nextId, prevId) => {
  if (!scriptEditorVisible.value) return
  const module = scriptEditorModule.value
  if (prevId) {
    scriptDrafts.value = {
      ...scriptDrafts.value,
      [getItemDraftKey(module, prevId)]: editorCode.value || '',
    }
  }
  if (!nextId) return
  const selected = getItemsByModule(module).find((item) => item.id === nextId)
  if (!selected) return
  if (module === 'timers') selectedTimerId.value = nextId
  if (module === 'variableChanges') selectedVariableId.value = nextId
  if (module === 'custom') selectedCustomId.value = nextId
  editorCode.value =
    scriptDrafts.value[getItemDraftKey(module, nextId)] ?? getItemCodeById(module, nextId)
  const interval = Number(selected.interval ?? selected.time ?? selected.schedule ?? 1000)
  editorInterval.value = Number.isFinite(interval) ? interval : 1000
  editorParams.value = selected.params || selected.args || ''
})

watch(selectedTimer, (item) => {
  if (!item) return
  editorCode.value = item.code || ''
})

watch(selectedVariableChange, (item) => {
  if (!item) return
  editorCode.value = item.code || ''
})

watch(selectedCustom, (item) => {
  if (!item) return
  editorCode.value = item.code || ''
  editorParams.value = item.params || item.args || ''
})

function buildScriptTree(groups: Array<any>, items: Array<any>): Array<any> {
  const groupMap = new Map<any, any>()
  const roots: Array<any> = []

  groups.forEach((group) => {
    groupMap.set(group.id, {
      id: group.id,
      label: group.name,
      type: 'group',
      children: [],
    })
  })

  groupMap.forEach((node, id) => {
    const group = groups.find((item) => item.id === id)
    if (group?.parentId && groupMap.has(group.parentId)) {
      groupMap.get(group.parentId).children.push(node)
    } else {
      roots.push(node)
    }
  })

  items.forEach((item) => {
    const label = item.name || item.variable || t('scriptPanel.editor.untitledScript')
    const node = { id: item.id, label, type: 'item', itemId: item.id }
    if (item.groupId && groupMap.has(item.groupId)) {
      groupMap.get(item.groupId).children.push(node)
    } else {
      roots.push(node)
    }
  })

  return roots
}

function buildGroupTree(groups: Array<any>): Array<any> {
  const groupMap = new Map<any, any>()
  const roots: Array<any> = []
  const normalized = Array.isArray(groups) ? groups : []
  normalized.forEach((group) => {
    groupMap.set(group.id, {
      id: group.id,
      label: group.name,
      type: 'group',
      children: [],
    })
  })
  groupMap.forEach((node, id) => {
    const group = normalized.find((item) => item.id === id)
    if (group?.parentId && groupMap.has(group.parentId)) {
      groupMap.get(group.parentId).children.push(node)
    } else {
      roots.push(node)
    }
  })
  return roots
}

function buildPageTree(pageList: Array<any>): Array<any> {
  const nodeMap = new Map<any, any>()
  const roots: Array<any> = []
  const normalized = Array.isArray(pageList) ? pageList : []
  normalized
    .filter((page) => page.type !== 'dialog')
    .forEach((page) => {
      nodeMap.set(page.id, {
        id: page.id,
        label: page.name || page.title || t('shell.untitledPage'),
        name: page.name || page.title || t('shell.untitledPage'),
        type: page.type || 'page',
        parentId: page.parentId || null,
        children: [],
      })
    })

  nodeMap.forEach((node) => {
    if (node.parentId && nodeMap.has(node.parentId)) {
      nodeMap.get(node.parentId).children.push(node)
    } else {
      roots.push(node)
    }
  })
  return roots
}

function selectSystem(key: string) {
  selectedSystemKey.value = key
}

const getSelectedNodes = (module: any): Array<any> => selectedNodes.value?.[module] || []

function setSelectedNodes(module: any, nodes: Array<any>) {
  selectedNodes.value = {
    ...selectedNodes.value,
    [module]: nodes,
  }
}

function isScriptSelected(module: any, data: any) {
  return getSelectedNodes(module).some((node) => node.id === data.id)
}

function getMenuItemCount(nodeType: any) {
  if (nodeType === 'blank') return 3
  if (nodeType === 'item') return 5
  if (nodeType === 'group') return 4
  return 4
}

function setContextMenuPosition(event: any, nodeType: any) {
  const width = 180
  const itemHeight = 38
  const height = getMenuItemCount(nodeType) * itemHeight + 12
  const maxX = window.innerWidth - width - 8
  const maxY = window.innerHeight - height - 8
  const x = Math.max(8, Math.min(event.clientX, maxX))
  const y = Math.max(8, Math.min(event.clientY, maxY))
  contextMenuPosition.value = { x, y }
}

const isMixedSelection = computed(() => {
  const module = contextMenuModule.value
  const nodes = getSelectedNodes(module)
  if (nodes.length <= 1) return false
  const types = new Set(nodes.map((node) => node.type))
  return types.size > 1
})

const canOpenEditScript = computed(() => {
  const module = contextMenuModule.value
  const nodes = getSelectedNodes(module)
  if (nodes.length !== 1) return false
  return nodes[0]?.type === 'item'
})

const canEditGroup = computed(() => {
  const module = contextMenuModule.value
  const nodes = getSelectedNodes(module)
  if (nodes.length !== 1) return false
  return nodes[0]?.type === 'group'
})

const canDeleteSelection = computed(() => !isMixedSelection.value)
function insertText(text: string) {
  if (systemEditorVisible.value) {
    systemEditorRef.value?.insertText?.(text)
    return
  }
  if (scriptEditorVisible.value) {
    activeEditorRef.value?.insertText?.(text)
  }
}

function handleCustomScriptInsert(data: any) {
  if (data?.type !== 'item') return
  const script = customScripts.value.find((item) => item.id === data.itemId)
  if (!script?.name) return
  const params =
    typeof script.params === 'string' && script.params.trim()
      ? script.params.trim()
      : typeof script.args === 'string'
        ? script.args.trim()
        : ''
  const call = params ? `${script.name}(${params})` : `${script.name}()`
  insertText(`customScripts.${call}`)
}

function handlePageInsert(data: any) {
  if (!data || data.type !== 'page') return
  const name = data.name || data.label
  if (!name) return
  insertText(`components.pages["${name}"]`)
}

function handlePageVariableInsert(row: any) {
  if (!row?.name) return
  insertText(resolvePageVariableSnippet(row.name))
}

function handleEnumGroupSelect(data: any) {
  if (!data) {
    enumSelectedGroupId.value = null
    return
  }
  enumSelectedGroupId.value = data.id === 'all' ? null : data.id
}

function handleEnumRowClick(row: any) {
  enumSelectedVar.value = row || null
}

function handleEnumRowDblClick(row: any) {
  enumSelectedVar.value = row || null
  confirmEnumInsert()
}

function enumRowClass({ row }: any) {
  if (enumSelectedVar.value?.name === row.name) return 'is-selected'
  return ''
}

function filterSidebarNode(value: string, data: any) {
  if (!value) return true
  const keyword = String(value).toLowerCase()
  return String(data?.label || '')
    .toLowerCase()
    .includes(keyword)
}

function openVariableEnum() {
  enumVariableSearch.value = ''
  enumSelectedVar.value = null
  enumSelectedGroupId.value = null
  variableEnumVisible.value = true
}

function confirmEnumInsert() {
  if (!enumSelectedVar.value?.name) return
  insertText(resolveProjectVariableSnippet(enumSelectedVar.value.name))
  variableEnumVisible.value = false
}

function formatSystemCode() {
  systemEditorRef.value?.format?.()
}

function formatActiveCode() {
  activeEditorRef.value?.format?.()
}

function openSystemEditor(key?: string) {
  if (contextMenuVisible.value) closeContextMenu()
  if (key) selectedSystemKey.value = key
  const activeKey = key || selectedSystemKey.value
  systemEditorTab.value = activeKey
  systemDrafts.value = {
    startup: getSystemCodeByKey('startup'),
    shutdown: getSystemCodeByKey('shutdown'),
  }
  systemCode.value = systemDrafts.value[activeKey] || ''
  systemOriginalCode.value = JSON.stringify(systemDrafts.value)
  systemEditorVisible.value = true
}

function openScriptEditor(module: any, data?: any) {
  if (contextMenuVisible.value) closeContextMenu()
  if (!canOpenEditScript.value && !data) return
  if (data?.type === 'item') {
    handleScriptNodeClick(module, data)
  }
  const selected = getSelectedItem(module)
  if (!selected) return
  editorOriginalCode.value = selected.code || ''
  editorCode.value = selected.code || ''
  const interval = Number(selected.interval ?? selected.time ?? selected.schedule ?? 1000)
  editorInterval.value = Number.isFinite(interval) ? interval : 1000
  editorOriginalInterval.value = editorInterval.value
  editorParams.value = selected.params || selected.args || ''
  scriptEditorModule.value = module
  scriptDrafts.value = Object.fromEntries(
    getItemsByModule(module).map((item) => [getItemDraftKey(module, item.id), item.code || '']),
  )
  scriptEditorTab.value = selected.id
  scriptEditorVisible.value = true
}

async function persistGlobals() {
  if (!projectId.value) {
    showError(t('datapointPanel.missingProject'))
    return
  }
  const result = await editorStore.saveProjectSettings()
  if (!result.ok) {
    showError(result.error?.message || t('scriptPanel.messages.saveFailed'))
  }
}

async function saveSystemScript() {
  const activeKey = systemEditorTab.value || selectedSystemKey.value
  systemDrafts.value = { ...systemDrafts.value, [activeKey]: systemCode.value || '' }
  const scripts = (globalScripts.value || {}) as Record<string, any>
  const system = (scripts.system || {
    startup: { code: '' },
    shutdown: { code: '' },
  }) as Record<string, any>
  const next = {
    ...system,
    startup: {
      ...system.startup,
      code: systemDrafts.value.startup || '',
    },
    shutdown: {
      ...system.shutdown,
      code: systemDrafts.value.shutdown || '',
    },
  }
  ;(globalScripts.value as any) = { ...scripts, system: next }
  systemOriginalCode.value = JSON.stringify(systemDrafts.value)
  selectedSystemKey.value = activeKey
  await persistGlobals()
  showSuccess(t('scriptPanel.messages.savedScript'))
}

function handleScriptNodeClick(module: any, data: any, event?: any) {
  const isCtrl = Boolean(event?.ctrlKey || event?.metaKey)
  const current = getSelectedNodes(module)
  if (isCtrl) {
    if (current.some((node) => node.id === data.id)) {
      setSelectedNodes(
        module,
        current.filter((node) => node.id !== data.id),
      )
    } else {
      setSelectedNodes(module, [...current, data])
    }
  } else {
    setSelectedNodes(module, [data])
  }

  if (data.type === 'item') {
    if (module === 'timers') selectedTimerId.value = data.itemId
    if (module === 'variableChanges') selectedVariableId.value = data.itemId
    if (module === 'custom') selectedCustomId.value = data.itemId
  } else if (data.type === 'group') {
    if (module === 'timers') selectedTimerId.value = data.id
    if (module === 'variableChanges') selectedVariableId.value = data.id
    if (module === 'custom') selectedCustomId.value = data.id
  }
  if (contextMenuVisible.value) closeContextMenu()
}

function handleTreeContextMenu(module: any, event: any, data: any) {
  event.preventDefault()
  event.stopPropagation()
  if (!isScriptSelected(module, data)) {
    setSelectedNodes(module, [data])
  }
  contextMenuModule.value = module
  contextMenuNode.value = data
  setContextMenuPosition(event, data.type)
  contextMenuVisible.value = true
  showMoveToMenu.value = false
  if (data.type === 'item') {
    if (module === 'timers') selectedTimerId.value = data.itemId
    if (module === 'variableChanges') selectedVariableId.value = data.itemId
    if (module === 'custom') selectedCustomId.value = data.itemId
  } else if (data.type === 'group') {
    if (module === 'timers') selectedTimerId.value = data.id
    if (module === 'variableChanges') selectedVariableId.value = data.id
    if (module === 'custom') selectedCustomId.value = data.id
  }
}

function handleBlankContextMenu(module: any, event: any) {
  event.preventDefault()
  event.stopPropagation()
  setSelectedNodes(module, [])
  contextMenuModule.value = module
  contextMenuNode.value = { type: 'blank' }
  setContextMenuPosition(event, 'blank')
  contextMenuVisible.value = true
  showMoveToMenu.value = false
}

function closeContextMenu() {
  contextMenuVisible.value = false
  contextMenuNode.value = null
  showMoveToMenu.value = false
}

function openGroupEditFromMenu() {
  if (!canEditGroup.value) return
  const module = contextMenuModule.value
  closeContextMenu()
  openGroupEdit(module)
}

function openGroupCreateFromMenu() {
  const module = contextMenuModule.value
  const groupIdValue = contextMenuNode.value?.type === 'group' ? contextMenuNode.value?.id : null
  closeContextMenu()
  groupDialogModule.value = module
  groupEditMode.value = false
  groupId.value = ''
  groupName.value = ''
  groupParentId.value = groupIdValue
  groupDialogVisible.value = true
}

function handleMoveTo(groupIdValue: any) {
  const module = contextMenuModule.value
  const node = contextMenuNode.value
  closeContextMenu()
  if (!module || !node) return
  const nodesToMove = getSelectedNodes(module).length ? getSelectedNodes(module) : [node]
  let nextItems = getItemsByModule(module)
  let nextGroups = getGroupsByModule(module)
  let blocked = false

  nodesToMove.forEach((item) => {
    if (item.type === 'item') {
      nextItems = nextItems.map((row) =>
        row.id === item.itemId ? { ...row, groupId: groupIdValue } : row,
      )
    } else if (item.type === 'group') {
      if (groupIdValue && isDescendantGroup(groupIdValue, item.id, module)) {
        blocked = true
        return
      }
      nextGroups = nextGroups.map((row) =>
        row.id === item.id ? { ...row, parentId: groupIdValue } : row,
      )
    }
  })

  if (blocked) {
    showWarning(t('scriptPanel.messages.moveToChildGroupBlocked'))
  }

  updateModuleGroups(module, nextGroups)
  updateModuleItems(module, nextItems)
}

function allowScriptDrag() {
  return true
}

function allowScriptDrop(module: any, draggingNode: any, dropNode: any, type: any) {
  const dragData = draggingNode.data
  const dropData = dropNode.data

  if (dragData.type === 'item') {
    if (type === 'inner' && dropData.type !== 'group') return false
    return true
  }

  if (dragData.type === 'group') {
    if (type === 'inner' && dropData.type !== 'group') return false
    if (dropData.type === 'group' && isDescendantGroup(dropData.id, dragData.id, module)) {
      return false
    }
    if (type === 'inner' && dropData.type === 'group') {
      const depth = getGroupDepth(dropData.id, module) + getGroupSubtreeDepth(dragData.id, module)
      if (depth > maxGroupDepth) return false
    }
    return true
  }

  return false
}

function handleScriptDrop(module: any, draggingNode: any, dropNode: any, dropType: any) {
  const dragData = draggingNode.data
  const targetGroupId = resolveTargetGroupId(dropNode, dropType)

  if (dragData.type === 'item') {
    const items = getItemsByModule(module).map((item) =>
      item.id === dragData.itemId ? { ...item, groupId: targetGroupId } : item,
    )
    updateModuleItems(module, items)
    return
  }

  if (dragData.type === 'group') {
    const groups = getGroupsByModule(module).map((group) =>
      group.id === dragData.id ? { ...group, parentId: targetGroupId } : group,
    )
    updateModuleGroups(module, groups)
  }
}

function resolveTargetGroupId(dropNode: any, dropType: any): any {
  const dropData = dropNode.data
  if (dropType === 'inner') {
    return dropData.type === 'group' ? dropData.id : null
  }
  const parent = dropNode.parent?.data
  return parent?.type === 'group' ? parent.id : null
}

async function removeScript(module: any) {
  if (contextMenuVisible.value) closeContextMenu()
  if (!canDeleteSelection.value) {
    showWarning(t('scriptPanel.messages.mixedDeleteBlocked'))
    return
  }
  const selected = getSelectedItem(module)
  const nodes = getSelectedNodes(module)
  const itemsToRemove = nodes.filter((node) => node.type === 'item').map((node) => node.itemId)
  const groupsToRemove = nodes.filter((node) => node.type === 'group').map((node) => node.id)
  if (!itemsToRemove.length && !groupsToRemove.length && !selected) return
  try {
    const count = itemsToRemove.length + groupsToRemove.length || 1
    let message = t('scriptPanel.messages.deleteScriptNamed', {
      name: selected?.name || t('scriptPanel.editor.untitledScript'),
    })
    if (count > 1) {
      message = t('scriptPanel.messages.deleteSelectedCount', { count })
    } else if (groupsToRemove.length === 1 && itemsToRemove.length === 0) {
      const name = getSelectedNodes(module).find((node) => node.type === 'group')?.label || ''
      message = t('scriptPanel.messages.deleteGroupNamed', { name })
    }
    await ElMessageBox.confirm(message, t('scriptPanel.messages.deleteTitle'), {
      type: 'warning',
      lockScroll: false,
    })
  } catch {
    return
  }
  const items = getItemsByModule(module)
  const groups = getGroupsByModule(module)
  const nextItems = items.filter((item) => !itemsToRemove.includes(item.id))

  if (groupsToRemove.length) {
    const parentMap = new Map()
    groups.forEach((group) => {
      if (groupsToRemove.includes(group.id)) {
        parentMap.set(group.id, group.parentId || null)
      }
    })

    const nextGroups = groups
      .filter((group) => !groupsToRemove.includes(group.id))
      .map((group) =>
        groupsToRemove.includes(group.parentId)
          ? { ...group, parentId: parentMap.get(group.parentId) || null }
          : group,
      )

    const reboundItems = nextItems.map((item) =>
      groupsToRemove.includes(item.groupId)
        ? { ...item, groupId: parentMap.get(item.groupId) || null }
        : item,
    )

    updateModuleGroups(module, nextGroups)
    updateModuleItems(module, reboundItems)
  } else {
    updateModuleItems(module, nextItems)
  }
}

function copyScript(module: any) {
  if (contextMenuVisible.value) closeContextMenu()
  const nodes = getSelectedNodes(module)
  const items = nodes
    .filter((node) => node.type === 'item')
    .map((node) => getItemsByModule(module).find((item) => item.id === node.itemId))
    .filter(Boolean)
  if (!items.length) {
    const selected = getSelectedItem(module)
    if (!selected) return
    items.push(selected)
  }
  scriptClipboard.value = {
    module,
    items: items.map((item) => JSON.parse(JSON.stringify(item))),
  }
  showSuccess(`已复制 ${scriptClipboard.value.items.length} 个脚本`)
}

async function pasteScript(module: any) {
  if (contextMenuVisible.value) closeContextMenu()
  if (!scriptClipboard.value?.items?.length) return
  const items = getItemsByModule(module)
  const targetGroupId = getSelectedGroupId(module)
  const nextItems = [...items]

  scriptClipboard.value.items.forEach((source: any) => {
    const nameBase = source.name || t('scriptPanel.editor.genericScript')
    let name = nameBase
    let index = 1
    while (nextItems.some((item) => item.name === name)) {
      name = `${nameBase}_copy${index}`
      index += 1
    }
    nextItems.push({
      ...source,
      id: createId(),
      name,
      groupId: targetGroupId ?? source.groupId ?? null,
    })
  })

  updateModuleItems(module, nextItems)
}

function getSelectedItem(module: any): any {
  if (module === 'timers') return selectedTimer.value
  if (module === 'variableChanges') return selectedVariableChange.value
  if (module === 'custom') return selectedCustom.value
  return null
}

function getGroupsByModule(module: any): Array<any> {
  if (module === 'timers') return timerGroups.value as Array<any>
  if (module === 'variableChanges') return variableChangeGroups.value as Array<any>
  if (module === 'custom') return customGroups.value as Array<any>
  return []
}

function getItemsByModule(module: any): Array<any> {
  if (module === 'timers') return timerItems.value as Array<any>
  if (module === 'variableChanges') return variableChangeItems.value as Array<any>
  if (module === 'custom') return customItems.value as Array<any>
  return []
}

function getSelectedGroupId(module: any): any {
  if (module === 'timers') {
    if (selectedTimerGroup.value) return selectedTimerGroup.value.id
    if (selectedTimer.value?.groupId) return selectedTimer.value.groupId
  }
  if (module === 'variableChanges') {
    if (selectedVariableGroup.value) return selectedVariableGroup.value.id
    if (selectedVariableChange.value?.groupId) return selectedVariableChange.value.groupId
  }
  if (module === 'custom') {
    if (selectedCustomGroup.value) return selectedCustomGroup.value.id
    if (selectedCustom.value?.groupId) return selectedCustom.value.groupId
  }
  return null
}

function updateModuleGroups(module: any, groups: Array<any>) {
  const nextScripts = (globalScripts.value || {}) as Record<string, any>
  ;(globalScripts.value as any) = {
    ...nextScripts,
    [module]: {
      ...(nextScripts[module] || {}),
      groups,
    },
  }
  persistGlobals()
}

function updateModuleItems(module: any, items: Array<any>) {
  const nextScripts = (globalScripts.value || {}) as Record<string, any>
  ;(globalScripts.value as any) = {
    ...nextScripts,
    [module]: {
      ...(nextScripts[module] || {}),
      items,
    },
  }
  persistGlobals()
}

function openGroupEdit(module: any) {
  const groups = getGroupsByModule(module)
  const selectedId =
    module === 'timers'
      ? selectedTimerId.value
      : module === 'variableChanges'
        ? selectedVariableId.value
        : selectedCustomId.value
  const group = groups.find((item) => item.id === selectedId)
  if (!group) return
  groupEditMode.value = true
  groupDialogModule.value = module
  groupId.value = group.id
  groupName.value = group.name
  groupParentId.value = group.parentId || null
  groupDialogVisible.value = true
}

function createId() {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) {
    return crypto.randomUUID()
  }
  return `id_${Date.now()}_${Math.floor(Math.random() * 1000)}`
}

async function saveGroup() {
  const name = groupName.value.trim()
  if (!name) return showWarning(t('datapointPanel.groupNameRequired'))

  const parentId = groupParentId.value || null
  const depth = parentId ? getGroupDepth(parentId, groupDialogModule.value) + 1 : 1
  if (depth > maxGroupDepth) {
    return showWarning(`分组最多支持 ${maxGroupDepth} 层`)
  }

  const groups = getGroupsByModule(groupDialogModule.value)
  if (groupEditMode.value) {
    updateModuleGroups(
      groupDialogModule.value,
      groups.map((group) => (group.id === groupId.value ? { ...group, name, parentId } : group)),
    )
  } else {
    updateModuleGroups(groupDialogModule.value, [
      ...groups,
      { id: createId(), name, parentId, sortOrder: 0 },
    ])
  }
  groupDialogVisible.value = false
}

async function removeGroup(module: any) {
  if (!canDeleteSelection.value) {
    showWarning(t('scriptPanel.messages.mixedDeleteBlocked'))
    return
  }
  const groups = getGroupsByModule(module)
  const selectedId =
    module === 'timers'
      ? selectedTimerId.value
      : module === 'variableChanges'
        ? selectedVariableId.value
        : selectedCustomId.value
  const group = groups.find((item) => item.id === selectedId)
  if (!group) return

  try {
    await ElMessageBox.confirm(
      t('scriptPanel.messages.deleteGroupNamed', { name: group.name }),
      t('scriptPanel.messages.deleteTitle'),
      {
        type: 'warning',
        lockScroll: false,
      },
    )
  } catch {
    return
  }

  const parentId = group.parentId || null
  const nextGroups = groups
    .filter((item) => item.id !== group.id)
    .map((item) => (item.parentId === group.id ? { ...item, parentId } : item))

  const items = getItemsByModule(module).map((item) =>
    item.groupId === group.id ? { ...item, groupId: parentId } : item,
  )

  updateModuleGroups(module, nextGroups)
  updateModuleItems(module, items)
}

function getGroupDepth(groupIdValue: any, module: any): number {
  if (!groupIdValue) return 0
  let depth = 1
  let currentId = groupIdValue
  const groups = getGroupsByModule(module)
  const map = new Map(groups.map((group) => [group.id, group]))
  while (map.get(currentId)?.parentId) {
    depth += 1
    currentId = map.get(currentId).parentId
  }
  return depth
}

function getGroupSubtreeDepth(groupIdValue: any, module: any): number {
  const groups = getGroupsByModule(module)
  const children = groups.filter((group) => group.parentId === groupIdValue)
  if (!children.length) return 1
  const depths = children.map((child) => getGroupSubtreeDepth(child.id, module))
  return 1 + Math.max(...depths)
}

function isDescendantGroup(targetId: any, parentId: any, module: any): boolean {
  const groups = getGroupsByModule(module)
  let current = groups.find((group) => group.id === targetId)
  while (current?.parentId) {
    if (current.parentId === parentId) return true
    current = groups.find((group) => group.id === current.parentId)
  }
  return false
}

function saveScriptCode(module: any) {
  const selected = getSelectedItem(module)
  if (!selected) return
  if (scriptEditorTab.value) {
    scriptDrafts.value = {
      ...scriptDrafts.value,
      [getItemDraftKey(module, scriptEditorTab.value)]: editorCode.value || '',
    }
  }
  const items = getItemsByModule(module).map((item) =>
    scriptDrafts.value[getItemDraftKey(module, item.id)] !== undefined
      ? {
          ...item,
          code: scriptDrafts.value[getItemDraftKey(module, item.id)] || '',
          interval: module === 'timers' ? Number(editorInterval.value) || 1000 : item.interval,
        }
      : item,
  )
  updateModuleItems(module, items)
  scriptDrafts.value = Object.fromEntries(
    items.map((item) => [getItemDraftKey(module, item.id), item.code || '']),
  )
  editorOriginalCode.value = editorCode.value || ''
  editorOriginalInterval.value = editorInterval.value
  showSuccess(t('scriptPanel.messages.savedScript'))
}

function saveActiveScript() {
  if (scriptEditorModule.value === 'timers') {
    saveScriptCode('timers')
  } else if (scriptEditorModule.value === 'variableChanges') {
    saveScriptCode('variableChanges')
  } else if (scriptEditorModule.value === 'custom') {
    saveScriptCode('custom')
  }
}
async function handleSystemBeforeClose(done: any) {
  if (systemEditorTab.value) {
    systemDrafts.value = {
      ...systemDrafts.value,
      [systemEditorTab.value]: systemCode.value || '',
    }
  }
  const isDirty = JSON.stringify(systemDrafts.value) !== (systemOriginalCode.value || '')
  if (!isDirty) {
    done()
    return
  }
  try {
    await ElMessageBox.confirm(
      t('scriptPanel.messages.saveModifiedPrompt'),
      t('scriptPanel.messages.promptTitle'),
      {
        confirmButtonText: t('propertyPanel.bindingDialog.save'),
        cancelButtonText: t('pageTree.switchWithoutSave'),
        type: 'warning',
        lockScroll: false,
      },
    )
    await saveSystemScript()
    done()
  } catch {
    done()
  }
}

async function handleScriptBeforeClose(done: any) {
  if (scriptEditorTab.value) {
    scriptDrafts.value = {
      ...scriptDrafts.value,
      [getItemDraftKey(scriptEditorModule.value, scriptEditorTab.value)]: editorCode.value || '',
    }
  }
  const selected = getSelectedItem(scriptEditorModule.value)
  const codeDirty = getItemsByModule(scriptEditorModule.value).some((item) => {
    const key = getItemDraftKey(scriptEditorModule.value, item.id)
    return (scriptDrafts.value[key] ?? item.code ?? '') !== (item.code || '')
  })
  const intervalDirty =
    scriptEditorModule.value === 'timers' &&
    !!selected &&
    editorInterval.value !== editorOriginalInterval.value
  const isDirty = codeDirty || intervalDirty
  if (!isDirty) {
    done()
    return
  }
  try {
    await ElMessageBox.confirm(
      t('scriptPanel.messages.saveModifiedPrompt'),
      t('scriptPanel.messages.promptTitle'),
      {
        confirmButtonText: t('propertyPanel.bindingDialog.save'),
        cancelButtonText: t('pageTree.switchWithoutSave'),
        type: 'warning',
        lockScroll: false,
      },
    )
    saveActiveScript()
    done()
  } catch {
    done()
  }
}

function handleEditorShortcut(event: any) {
  if (!systemEditorVisible.value && !scriptEditorVisible.value) return
  if (!(event.ctrlKey || event.metaKey)) return
  const key = event.key.toLowerCase()
  if (key === 's') {
    event.preventDefault()
    if (systemEditorVisible.value) saveSystemScript()
    if (scriptEditorVisible.value) saveActiveScript()
  }
  if (key === 'f' && event.shiftKey && event.altKey) {
    event.preventDefault()
    if (systemEditorVisible.value) formatSystemCode()
    if (scriptEditorVisible.value) formatActiveCode()
  }
}

function openMetaDialog(module: any, mode: any) {
  if (contextMenuVisible.value) closeContextMenu()
  if (mode === 'edit' && getSelectedNodes(module).length > 1) {
    showWarning(t('scriptPanel.messages.multiEditBlocked'))
    return
  }
  metaDialogMode.value = mode
  metaDialogModule.value = module
  const selected = getSelectedItem(module)
  if (mode === 'edit' && selected) {
    metaForm.value = {
      id: selected.id,
      name: selected.name || selected.variable || '',
      interval: Number(selected.interval ?? selected.time ?? selected.schedule ?? 1000),
      description: selected.description || '',
      variable: selected.variable || '',
      params: selected.params || selected.args || '',
      groupId: selected.groupId || null,
    }
  } else {
    const groupIdValue =
      contextMenuNode.value?.type === 'group'
        ? contextMenuNode.value?.id
        : getSelectedGroupId(module)
    metaForm.value = {
      id: '',
      name: '',
      interval: 1000,
      description: '',
      variable: '',
      params: '',
      groupId: groupIdValue || null,
    }
  }
  metaDialogVisible.value = true
}

function saveMetaDialog() {
  const module = metaDialogModule.value
  if (!module) return
  const items = getItemsByModule(module)

  if (module === 'timers') {
    const name = metaForm.value.name.trim()
    if (!name) return showWarning(t('scriptPanel.messages.timerNameRequired'))
    const exists = items.some((item) => item.name === name && item.id !== metaForm.value.id)
    if (exists) return showWarning(t('scriptPanel.messages.functionNameExists'))
    if (!Number.isFinite(Number(metaForm.value.interval))) {
      return showWarning(t('scriptPanel.messages.invalidInterval'))
    }
  }

  if (module === 'variableChanges') {
    if (!metaForm.value.variable) return showWarning(t('scriptPanel.metaDialog.selectVariable'))
  }

  if (module === 'custom') {
    const name = metaForm.value.name.trim()
    if (!name) return showWarning(t('scriptPanel.messages.functionNameRequired'))
    const exists = items.some((item) => item.name === name && item.id !== metaForm.value.id)
    if (exists) return showWarning(t('scriptPanel.messages.functionNameExists'))
  }

  if (metaDialogMode.value === 'create') {
    const groupIdValue = metaForm.value.groupId || null
    const newItem = {
      id: createId(),
      name: module === 'variableChanges' ? metaForm.value.variable : metaForm.value.name.trim(),
      description: metaForm.value.description || '',
      variable: metaForm.value.variable || '',
      params: metaForm.value.params || '',
      interval: Number(metaForm.value.interval) || 1000,
      groupId: groupIdValue,
      code: '',
    }
    updateModuleItems(module, [...items, newItem])
  } else {
    const next = items.map((item) => {
      if (item.id !== metaForm.value.id) return item
      return {
        ...item,
        name: module === 'variableChanges' ? metaForm.value.variable : metaForm.value.name.trim(),
        description: metaForm.value.description || '',
        variable: metaForm.value.variable || item.variable || '',
        params: metaForm.value.params || '',
        interval: Number(metaForm.value.interval) || item.interval || 1000,
      }
    })
    updateModuleItems(module, next)
  }

  metaDialogVisible.value = false
}

function handleClickOutside() {
  if (contextMenuVisible.value) closeContextMenu()
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
  window.addEventListener('keydown', handleEditorShortcut)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
  window.removeEventListener('keydown', handleEditorShortcut)
})
</script>

<template>
  <div class="global-scripts">
    <el-collapse v-model="activeSections" class="scripts-collapse" :accordion="true">
      <ScriptVarsSystemSection
        :selected-system-key="selectedSystemKey"
        @select-system="selectSystem"
        @open-system-editor="openSystemEditor"
      />

      <ScriptVarsTimersSection
        :tree="timerTree"
        :allow-drop="
          (draggingNode, dropNode, dropType) =>
            allowScriptDrop('timers', draggingNode, dropNode, dropType)
        "
        :allow-drag="allowScriptDrag"
        :is-selected="(data) => isScriptSelected('timers', data)"
        @blank-contextmenu="(event) => handleBlankContextMenu('timers', event)"
        @node-dblclick="(data) => openScriptEditor('timers', data)"
        @node-contextmenu="(event, data) => handleTreeContextMenu('timers', event, data)"
        @node-drop="
          (draggingNode, dropNode, dropType) =>
            handleScriptDrop('timers', draggingNode, dropNode, dropType)
        "
        @node-click="(data, event) => handleScriptNodeClick('timers', data, event)"
        @create-script="openMetaDialog('timers', 'create')"
      />
      <ScriptVarsVariableChangesSection
        :tree="variableChangeTree"
        :allow-drop="
          (draggingNode, dropNode, dropType) =>
            allowScriptDrop('variableChanges', draggingNode, dropNode, dropType)
        "
        :allow-drag="allowScriptDrag"
        :is-selected="(data) => isScriptSelected('variableChanges', data)"
        @blank-contextmenu="(event) => handleBlankContextMenu('variableChanges', event)"
        @node-dblclick="(data) => openScriptEditor('variableChanges', data)"
        @node-contextmenu="(event, data) => handleTreeContextMenu('variableChanges', event, data)"
        @node-drop="
          (draggingNode, dropNode, dropType) =>
            handleScriptDrop('variableChanges', draggingNode, dropNode, dropType)
        "
        @node-click="(data, event) => handleScriptNodeClick('variableChanges', data, event)"
        @create-script="openMetaDialog('variableChanges', 'create')"
      />
      <ScriptVarsCustomSection
        :tree="customTree"
        :allow-drop="
          (draggingNode, dropNode, dropType) =>
            allowScriptDrop('custom', draggingNode, dropNode, dropType)
        "
        :allow-drag="allowScriptDrag"
        :is-selected="(data) => isScriptSelected('custom', data)"
        @blank-contextmenu="(event) => handleBlankContextMenu('custom', event)"
        @node-dblclick="(data) => openScriptEditor('custom', data)"
        @node-contextmenu="(event, data) => handleTreeContextMenu('custom', event, data)"
        @node-drop="
          (draggingNode, dropNode, dropType) =>
            handleScriptDrop('custom', draggingNode, dropNode, dropType)
        "
        @node-click="(data, event) => handleScriptNodeClick('custom', data, event)"
        @create-script="openMetaDialog('custom', 'create')"
      />
    </el-collapse>
    <VariableGroupFormDialog
      v-model="groupDialogVisible"
      v-model:name="groupName"
      v-model:parent-id="groupParentId"
      :edit-mode="groupEditMode"
      :parent-options="groupParentOptions"
      @confirm="saveGroup"
    />

    <ScriptVarsMetaFormDialog
      v-model="metaDialogVisible"
      v-model:meta="metaForm"
      :title="metaDialogTitle"
      :module="metaDialogModule"
      :project-variable-names="projectVariableNames"
      @confirm="saveMetaDialog"
    />

    <ScriptVarsSystemScriptDialog
      ref="systemEditorRef"
      v-model="systemEditorVisible"
      v-model:system-code="systemCode"
      v-model:script-search="scriptSearch"
      v-model:page-search="pageSearch"
      v-model:page-variable-search="pageVariableSearch"
      v-model:active-tab="systemEditorTab"
      :dialog-title="t('scriptPanel.editor.systemScript')"
      :meta-title="selectedSystemLabel"
      :tabs="systemEditorTabs"
      :custom-script-sidebar-tree="customScriptSidebarTree"
      :page-sidebar-tree="pageSidebarTree"
      :page-variable-rows="pageVariableRows"
      :js-completions="jsCompletions"
      :filter-sidebar-node="filterSidebarNode"
      :before-close="handleSystemBeforeClose"
      @open-variable-enum="openVariableEnum"
      @save="saveSystemScript"
      @custom-insert="handleCustomScriptInsert"
      @page-insert="handlePageInsert"
      @page-variable-insert="handlePageVariableInsert"
    />
    <ScriptEditorDialog
      ref="activeEditorRef"
      v-model="scriptEditorVisible"
      v-model:code="editorCode"
      v-model:script-search="scriptSearch"
      v-model:page-search="pageSearch"
      v-model:page-variable-search="pageVariableSearch"
      v-model:active-tab="scriptEditorTab"
      scope="global"
      :dialog-title="scriptEditorTitle"
      :meta-title="editorMetaTitle"
      :meta-description="scriptEditorKindLabel"
      :tabs="scriptEditorTabs"
      :custom-script-sidebar-tree="customScriptSidebarTree"
      :page-sidebar-tree="pageSidebarTree"
      :page-variable-rows="pageVariableRows"
      :js-completions="jsCompletions"
      :filter-sidebar-node="filterSidebarNode"
      :before-close="handleScriptBeforeClose"
      @open-variable-enum="openVariableEnum"
      @save="saveActiveScript"
      @custom-insert="handleCustomScriptInsert"
      @page-insert="handlePageInsert"
      @page-variable-insert="handlePageVariableInsert"
    >
      <template #meta-extra>
        <div v-if="scriptEditorModule === 'timers'" class="meta-inline">
          <span class="meta-label">{{ t('scriptPanel.editor.timeMs') }}</span>
          <el-input-number v-model="editorInterval" :min="100" :step="100" size="small" />
        </div>
        <div v-if="scriptEditorModule === 'custom'" class="meta-inline">
          <span class="meta-label">{{ t('scriptPanel.editor.params') }}</span>
          <span class="meta-value">{{ editorParams || t('scriptPanel.editor.none') }}</span>
        </div>
      </template>
    </ScriptEditorDialog>

    <el-dialog
      v-model="variableEnumVisible"
      :title="t('scriptPanel.variableEnum.title')"
      width="760px"
      :close-on-click-modal="false"
      :lock-scroll="false"
    >
      <div class="enum-layout">
        <div class="enum-left">
          <div class="sidebar-title">{{ t('scriptPanel.variableEnum.groups') }}</div>
          <el-tree
            ref="enumGroupTreeRef"
            :data="enumGroupTree"
            node-key="id"
            :default-expand-all="true"
            :expand-on-click-node="false"
            :filter-node-method="filterSidebarNode"
            @node-click="handleEnumGroupSelect"
          >
            <template #default="{ data }">
              <div class="tree-node" :class="`node-${data.type}`">
                <el-icon class="node-icon icon-variable">
                  <IconEpFolder />
                </el-icon>
                <span class="node-label is-group">{{ data.label }}</span>
              </div>
            </template>
          </el-tree>
        </div>
        <div class="enum-right">
          <el-input
            v-model="enumVariableSearch"
            size="small"
            :placeholder="t('scriptPanel.variableEnum.search')"
            clearable
          />
          <el-table
            :data="enumVariableRows"
            size="small"
            height="360"
            highlight-current-row
            :row-class-name="enumRowClass"
            @row-click="handleEnumRowClick"
            @row-dblclick="handleEnumRowDblClick"
          >
            <el-table-column
              prop="name"
              :label="t('scriptPanel.variableEnum.name')"
              min-width="160"
            />
            <el-table-column prop="type" :label="t('scriptPanel.variableEnum.type')" width="90" />
            <el-table-column
              prop="description"
              :label="t('scriptPanel.variableEnum.description')"
              min-width="160"
            />
            <el-table-column
              prop="sourceLabel"
              :label="t('scriptPanel.variableEnum.source')"
              min-width="150"
            />
          </el-table>
        </div>
      </div>
      <template #footer>
        <el-button @click="variableEnumVisible = false">{{
          t('scriptPanel.variableEnum.cancel')
        }}</el-button>
        <el-button type="primary" :disabled="!enumSelectedVar" @click="confirmEnumInsert">
          {{ t('scriptPanel.variableEnum.insert') }}
        </el-button>
      </template>
    </el-dialog>

    <div
      v-if="contextMenuVisible"
      class="context-menu"
      :style="contextMenuStyle"
      @click.stop
      @mousedown.stop
    >
      <template v-if="contextMenuNode?.type === 'blank'">
        <div class="context-menu-item" @click="openGroupCreateFromMenu">
          {{ t('scriptPanel.contextMenu.newGroup') }}
        </div>
        <div class="context-menu-item" @click="openMetaDialog(contextMenuModule, 'create')">
          {{ t('scriptPanel.contextMenu.newScript') }}
        </div>
        <div
          class="context-menu-item"
          :class="{ 'is-disabled': !scriptClipboard }"
          @click="pasteScript(contextMenuModule)"
        >
          {{ t('scriptPanel.contextMenu.paste') }}
        </div>
      </template>
      <template v-else-if="contextMenuNode?.type === 'item'">
        <div
          class="context-menu-item"
          :class="{ 'is-disabled': !canOpenEditScript }"
          @click="openScriptEditor(contextMenuModule)"
        >
          {{ t('scriptPanel.contextMenu.open') }}
        </div>
        <div
          class="context-menu-item"
          :class="{ 'is-disabled': !canOpenEditScript }"
          @click="openMetaDialog(contextMenuModule, 'edit')"
        >
          {{ t('scriptPanel.contextMenu.edit') }}
        </div>
        <div class="context-menu-item" @click="copyScript(contextMenuModule)">
          {{ t('scriptPanel.contextMenu.copy') }}
        </div>
        <div class="context-menu-item" @click="showMoveToMenu = !showMoveToMenu">
          {{ t('scriptPanel.contextMenu.moveTo') }}
        </div>
        <div
          class="context-menu-item context-menu-item--danger"
          :class="{ 'is-disabled': !canDeleteSelection }"
          @click="removeScript(contextMenuModule)"
        >
          {{ t('scriptPanel.contextMenu.delete') }}
        </div>
      </template>
      <template v-else-if="contextMenuNode?.type === 'group'">
        <div
          class="context-menu-item"
          :class="{ 'is-disabled': !canEditGroup }"
          @click="openGroupEditFromMenu"
        >
          {{ t('scriptPanel.contextMenu.editGroup') }}
        </div>
        <div class="context-menu-item" @click="openGroupCreateFromMenu">
          {{ t('scriptPanel.contextMenu.newChildGroup') }}
        </div>
        <div class="context-menu-item" @click="showMoveToMenu = !showMoveToMenu">
          {{ t('scriptPanel.contextMenu.moveTo') }}
        </div>
        <div
          class="context-menu-item context-menu-item--danger"
          :class="{ 'is-disabled': !canDeleteSelection }"
          @click="removeGroup(contextMenuModule)"
        >
          {{ t('scriptPanel.contextMenu.deleteGroup') }}
        </div>
      </template>
    </div>

    <div
      v-if="contextMenuVisible && showMoveToMenu"
      class="context-menu context-submenu"
      :style="submenuStyle"
      @click.stop
      @mousedown.stop
    >
      <div class="context-menu-item" @click="handleMoveTo(null)">
        {{ t('scriptPanel.contextMenu.root') }}
      </div>
      <div
        v-for="group in availableGroups"
        :key="group.id"
        class="context-menu-item"
        @click="handleMoveTo(group.id)"
      >
        {{ group.name }}
      </div>
    </div>
  </div>
</template>

<style scoped>
.global-scripts {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.scripts-collapse {
  --el-collapse-header-height: 40px;
  flex: 1;
  overflow: auto;
  padding-bottom: 8px;
}

:deep(.scripts-collapse .el-collapse-item__header) {
  font-weight: 600;
  height: 40px;
  line-height: 40px;
  padding: 0 4px;
}

:deep(.scripts-collapse .el-collapse-item) {
  border-bottom: 1px solid color-mix(in srgb, var(--designer-border-color) 70%, transparent);
}

:deep(.scripts-collapse .el-collapse-item__wrap) {
  border-bottom: 0;
}

:deep(.scripts-collapse .el-collapse-item__content) {
  padding: 0;
}

.scripts-layout {
  display: block;
  padding: 6px 4px;
  box-sizing: border-box;
}

.scripts-list {
  width: 100%;
  border: 0;
  border-radius: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: transparent;
  min-height: 48px;
}

.scripts-list.is-full {
  flex: 1;
}

.list-menu {
  border-right: 0;
}

.meta-desc {
  font-size: 12px;
  color: var(--designer-text-secondary);
}

.meta-inline {
  display: flex;
  align-items: center;
  gap: 8px;
}

.meta-label {
  font-size: 12px;
  color: var(--designer-text-muted);
}

.meta-value {
  font-size: 12px;
  color: var(--designer-text-primary);
}

.sidebar-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--designer-text-secondary);
}

.tree-node {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  width: 100%;
  padding: 6px 8px;
  border-radius: 6px;
  transition: background-color 0.2s;
}

.node-icon {
  color: var(--designer-text-muted);
  flex-shrink: 0;
}

.node-group .node-icon {
  color: var(--designer-primary-text);
}

.node-item .node-icon {
  color: var(--designer-success-text);
}

.node-item .node-icon.icon-timer {
  color: var(--designer-warning-text);
}

.node-item .node-icon.icon-change {
  color: #8b5cf6;
}

.node-item .node-icon.icon-custom {
  color: var(--designer-success-text);
}

.node-variable .node-icon.icon-variable {
  color: #0ea5e9;
}

.node-page .node-icon.icon-page {
  color: #6366f1;
}

.list-menu .node-icon.icon-system {
  color: #6366f1;
  margin-right: 6px;
}

.tree-node:hover {
  background: var(--designer-hover-surface);
}

.tree-node.is-selected {
  background: var(--designer-primary-soft);
  color: var(--designer-text-primary);
}

.node-label {
  font-size: 13px;
  color: var(--designer-text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.node-label.is-group {
  font-weight: 600;
}

.enum-body {
  margin-top: 8px;
  max-height: 420px;
  overflow: auto;
}

.enum-layout {
  display: flex;
  gap: 12px;
}

.enum-left {
  width: 200px;
  border-right: 1px solid var(--designer-border-color);
  padding-right: 8px;
  max-height: 420px;
  overflow: auto;
}

.enum-right {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.enum-right :deep(.el-table__row.is-selected) {
  background: var(--designer-primary-soft);
}

:deep(.scripts-list .el-tree) {
  flex: 1;
  overflow: auto;
  padding: 10px 8px;
}

:deep(.scripts-list .el-tree-node__content) {
  height: 38px;
}

.context-menu {
  position: fixed;
  background: var(--designer-shell-surface);
  border: 1px solid var(--designer-border-color);
  border-radius: 8px;
  box-shadow: var(--designer-shadow-popover);
  z-index: 4000;
  min-width: 160px;
  padding: 6px 0;
}

.context-menu-item {
  padding: 10px 18px;
  cursor: pointer;
  font-size: 14px;
  color: var(--designer-text-secondary);
  white-space: nowrap;
}

.context-menu-item:hover {
  background-color: var(--designer-hover-surface);
}

.context-menu-item--danger {
  color: var(--designer-danger-text);
}

.context-menu-item--danger:hover {
  background-color: var(--designer-danger-surface);
}

.context-submenu {
  max-height: 300px;
  overflow-y: auto;
}

.context-menu-item.is-disabled {
  color: var(--designer-text-muted);
  pointer-events: none;
}
</style>
