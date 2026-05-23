<!--
  DatapointPanel - 数据点/变量面板
  管理数据点与变量（树形展示），支持快速添加、导入导出、右键菜单
-->
<script setup lang="ts">
import dayjs from 'dayjs'
import { ElMessage, ElMessageBox } from 'element-plus'
import { storeToRefs } from 'pinia'
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import * as XLSX from 'xlsx'
import IconEpEditPen from '~icons/ep/edit-pen'
import IconEpFolder from '~icons/ep/folder'
import IconEpLink from '~icons/ep/link'
import IconEpView from '~icons/ep/view'
import { TIME_FORMAT } from '@/constants'
import { datacenterApi } from '@/services'
import {
  buildProjectVariableFromDataPoint,
  findMappedProjectVariableName,
  normalizeProjectVariableName,
} from '@/services/data-variable-mapping'
import { dataServiceApi } from '@/services/dataServiceApi'
import { useEditorStore } from '@/stores/editor-store'
import { unwrapApiData } from '@/types/api'
import VariableGroupFormDialog from '@/ui/shared/tool-panels/VariableGroupFormDialog.vue'
import { requireConnectionsPayload } from '@/utils/datapoint-payload'
import DatapointPanelContextMenu from './DatapointPanelContextMenu.vue'
import DatapointPanelToolbar from './DatapointPanelToolbar.vue'
import DatapointQuickAddDialog from './DatapointQuickAddDialog.vue'
import DatapointVariableEditDialog from './DatapointVariableEditDialog.vue'

type VariableType =
  | 'string'
  | 'number'
  | 'boolean'
  | 'array'
  | 'object'
  | 'set'
  | 'map'
  | 'date'
  | 'regexp'
  | 'function'

interface VariableSourceLike {
  type?: string
  path?: string
  sourceType?: string
  sourceId?: string
  datapointId?: string
}

interface VariableDetailLike {
  type?: string
  default?: unknown
  value?: unknown
  description?: string
  groupId?: string | null
  mapped?: boolean
  source?: VariableSourceLike
}

interface VariableGroupLike {
  id: string
  name: string
  parentId?: string | null
  sortOrder?: number
}

interface TreeNodeMetaLike extends VariableDetailLike {
  mapped?: boolean
}

interface TreeNodeLike {
  id: string
  label: string
  type: 'group' | 'variable' | 'blank'
  name?: string
  meta?: TreeNodeMetaLike
  children?: TreeNodeLike[]
}

interface TreeControllerLike {
  setCurrentKey?: (key: string) => void
}

interface ContextMenuPositionLike {
  x: number
  y: number
}

interface ClipboardItemLike {
  name: string
  detail: VariableDetailLike
}

interface ClipboardLike {
  items: ClipboardItemLike[]
}

interface DatapointFieldLike {
  id: string
  name: string
  path: string
  sourceType: string
  sourceId: string
  sourceLabel: string
  type: string
  typeLabel: string
  description: string
  status: string
  statusLabel: string
  statusType: 'success' | 'info' | 'warning'
  mappingLabel: string
  mappedName: string
  updatedAtLabel: string
}

type VariableMapLike = Record<string, VariableDetailLike>
type ImportRowLike = Record<string, unknown>
type NullableString = string | null
interface TreeDropNodeLike {
  data: TreeNodeLike
  parent?: { data?: TreeNodeLike | null } | null
}

interface ClickLike extends MouseEvent {}

const showSuccess = (message: string): void => {
  ;(ElMessage as any).success(message)
}

const showWarning = (message: string): void => {
  ;(ElMessage as any).warning(message)
}

const showError = (message: string): void => {
  ;(ElMessage as any).error(message)
}

const editorStore = useEditorStore()
const { projectId, projectVariables, projectVariableGroups } = storeToRefs(editorStore)
const maxGroupDepth = 5
const { t } = useI18n()

const types: VariableType[] = [
  'string',
  'number',
  'boolean',
  'array',
  'object',
  'set',
  'map',
  'date',
  'regexp',
  'function',
]

const treeRef = ref<TreeControllerLike | null>(null)
const treeWrapRef = ref<HTMLElement | null>(null)
const selectedNode = ref<TreeNodeLike | null>(null)
const selectedNodes = ref<TreeNodeLike[]>([])
const varClipboard = ref<ClipboardLike | null>(null)
const contextMenuVisible = ref(false)
const contextMenuPosition = ref<ContextMenuPositionLike>({ x: 0, y: 0 })
const contextMenuNode = ref<TreeNodeLike | { id: string; type: 'blank'; label?: string } | null>(
  null,
)
const showMoveToMenu = ref(false)

const editVisible = ref(false)
const editMode = ref(false)
const originalName = ref('')
const editName = ref('')
const editType = ref('string')
const editValue = ref<string | number | boolean | null>('')
const editDescription = ref('')
const ROOT_GROUP_ID = '__root__'
const editGroupId = ref(ROOT_GROUP_ID)
const mapped = ref(false)
const mappedField = ref('')
const dataSources = ref<unknown[]>([])
const mappedSourceLabel = ref('')
const importInputRef = ref<HTMLInputElement | null>(null)
const importType = ref<'json' | 'csv' | 'xlsx'>('json')
const editValueHasErrors = ref(false)
const variableSearchKey = ref('')
const variableKindFilter = ref<'' | 'project' | 'mapped'>('')
const variableDataTypeFilter = ref('')
const variableTableWidth = ref(0)
const variableDetailVisible = ref(false)
const variableDetailNode = ref<TreeNodeLike | null>(null)

const groupVisible = ref(false)
const groupEditMode = ref(false)
const groupId = ref('')
const groupName = ref('')
const groupParentId = ref<NullableString>(null)

const quickVisible = ref(false)
const fields = ref<DatapointFieldLike[]>([])
const selectedFields = ref<DatapointFieldLike[]>([])
const searchKey = ref('')
const prefix = ref('')
const suffix = ref('')
const replaceFrom = ref('')
const replaceTo = ref('')
const quickLoading = ref(false)
const quickPage = ref(1)
const quickPageSize = ref(200)
const quickTotal = ref(0)
const quickStatusFilter = ref('')
const quickTypeFilter = ref('')
const quickSourceIdFilter = ref('')
const quickActiveField = ref<DatapointFieldLike | null>(null)
const quickSingleName = ref('')
const quickSingleType = ref('string')
const quickSingleGroupId = ref(ROOT_GROUP_ID)
const quickSingleDefaultValue = ref('')
const quickSingleDescription = ref('')

const isEditorType = computed(() =>
  ['function', 'array', 'object', 'set', 'map'].includes(editType.value),
)
const isStructuredType = computed(() => ['array', 'object', 'set', 'map'].includes(editType.value))
const editorLanguage = computed(() => (isStructuredType.value ? 'json' : 'javascript'))
const isTextType = computed(() => ['string', 'regexp'].includes(editType.value))

const groupOptions = computed<VariableGroupLike[]>(
  () => (projectVariableGroups.value || []) as VariableGroupLike[],
)

const groupParentOptions = computed(() => {
  if (!groupEditMode.value) return groupOptions.value
  return groupOptions.value.filter(
    (group) => group.id !== groupId.value && !isDescendantGroup(group.id, groupId.value),
  )
})

const variableDataTypeOptions = computed(() => {
  const typeSet = new Set<string>()
  Object.values((projectVariables.value || {}) as VariableMapLike).forEach((detail) => {
    const type = String(detail?.type || 'string')
    if (type) typeSet.add(type)
  })
  return Array.from(typeSet).sort()
})

const selectedVariable = computed(() => {
  if (selectedNode.value?.type !== 'variable') return null
  const name = selectedNode.value?.name
  const detail = name ? (projectVariables.value?.[name] as VariableDetailLike | undefined) : null
  if (!detail) return null
  return { name, detail }
})

const selectedGroup = computed(() => {
  if (selectedNode.value?.type !== 'group') return null
  const nodeId = selectedNode.value.id
  return (
    (projectVariableGroups.value as VariableGroupLike[] | undefined)?.find(
      (group) => group.id === nodeId,
    ) || null
  )
})

const selectedGroupId = computed(() => {
  const groupNode = selectedNodes.value.find((node) => node.type === 'group')
  if (groupNode?.id) return groupNode.id
  if (selectedGroup.value?.id) return selectedGroup.value.id
  if (selectedVariable.value?.detail?.groupId) return selectedVariable.value.detail.groupId ?? null
  return null
})

const quickSelectedCount = computed(() => selectedFields.value.length)
const quickSelectedMappableCount = computed(
  () => selectedFields.value.filter((field) => isQuickFieldSelectable(field)).length,
)
const quickSelectedKeys = computed(() =>
  selectedFields.value.map((field) => resolveQuickFieldKey(field)),
)
const quickSourceOptions = computed(() => {
  const sourceMap = new Map<string, string>()
  fields.value.forEach((field) => {
    if (!field.sourceId) return
    sourceMap.set(field.sourceId, field.sourceLabel || field.sourceId)
  })
  return Array.from(sourceMap, ([id, label]) => ({ id, label }))
})

const variableTree = computed(() =>
  buildTree(
    (projectVariableGroups.value || []) as any[] as VariableGroupLike[],
    (projectVariables.value || {}) as any as VariableMapLike,
  ),
)

const filteredVariableTree = computed(() => filterVariableTree(variableTree.value))
const filteredVariableNodes = computed(() => collectVariableNodes(filteredVariableTree.value))
const filteredVariableCount = computed(() => filteredVariableNodes.value.length)
const showVariableActionColumn = computed(() => variableTableWidth.value >= 190)
const showVariableKindColumn = computed(() => variableTableWidth.value >= 230)
const showVariableDataTypeColumn = computed(() => variableTableWidth.value >= 286)
const variableDetailTitle = computed(() => {
  const node = variableDetailNode.value
  if (!node) return t('datapointPanel.variableDetailTitle')
  return `${t('datapointPanel.variableDetailTitle')}：${node.label}`
})

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

const availableGroups = computed<any[]>(() => {
  const current = contextMenuNode.value
  const groups = (projectVariableGroups.value || []) as VariableGroupLike[]
  if (!current || current.type !== 'group') return groups
  return (groups as VariableGroupLike[]).filter(
    (group) => group.id !== current.id && !isDescendantGroup(group.id, current.id),
  )
})

const canEditSelection = computed(
  () => selectedNodes.value.length === 0 || selectedNodes.value.length === 1,
)

const canDeleteSelection = computed(() => {
  if (selectedNodes.value.length <= 1) return true
  const types = new Set(selectedNodes.value.map((node) => node.type))
  return types.size <= 1
})

// ---- 工具栏状态 ----
const toolbarHasSelection = computed(() => selectedNodes.value.length > 0)
const toolbarSelectionType = computed<'variable' | 'group' | 'mixed' | 'none'>(() => {
  if (!selectedNodes.value.length) return 'none'
  const types = new Set(selectedNodes.value.map((node) => node.type))
  if (types.size > 1) return 'mixed'
  return types.has('group') ? 'group' : 'variable'
})
const toolbarCanEdit = computed(() => {
  if (!selectedNodes.value.length) return false
  return canEditSelection.value
})
const toolbarCanDelete = computed(() => {
  if (!selectedNodes.value.length) return false
  return canDeleteSelection.value
})
const toolbarSelectedCount = computed(() => selectedNodes.value.length)

const importAccept = computed(() => {
  if (importType.value === 'csv') return '.csv'
  if (importType.value === 'xlsx') return '.xlsx,.xls'
  return '.json'
})

function buildTree(groups: VariableGroupLike[], variables: VariableMapLike): TreeNodeLike[] {
  const groupMap = new Map<string, TreeNodeLike>()
  const roots: TreeNodeLike[] = []

  groups.forEach((group: VariableGroupLike) => {
    groupMap.set(group.id, {
      id: group.id,
      label: group.name,
      type: 'group',
      children: [],
    })
  })

  groupMap.forEach((node, id) => {
    const group = groups.find((item: VariableGroupLike) => item.id === id)
    if (group?.parentId && groupMap.has(group.parentId)) {
      groupMap.get(group.parentId)?.children?.push(node)
    } else {
      roots.push(node)
    }
  })

  Object.entries(variables).forEach(([name, detail]) => {
    const normalizedDetail = (detail || {}) as VariableDetailLike
    const node = {
      id: `var:${name}`,
      label: name,
      type: 'variable',
      name,
      meta: {
        ...normalizedDetail,
        mapped: normalizedDetail.source?.type === 'dataCenter' || normalizedDetail.mapped === true,
      },
    } satisfies TreeNodeLike
    const groupIdValue = normalizedDetail.groupId
    if (groupIdValue && groupMap.has(groupIdValue)) {
      groupMap.get(groupIdValue)?.children?.push(node)
    } else {
      roots.push(node)
    }
  })

  return roots
}

function isMappedVariableNode(node: TreeNodeLike): boolean {
  return node.type === 'variable' && Boolean(node.meta?.mapped)
}

function getVariableKindLabel(node: TreeNodeLike): string {
  if (node.type === 'group') return t('datapointPanel.variableKindGroup')
  return isMappedVariableNode(node)
    ? t('datapointPanel.variableKindMapped')
    : t('datapointPanel.variableKindProject')
}

function getVariableMappingPath(node: TreeNodeLike): string {
  if (node.type !== 'variable') return ''
  return String(node.meta?.source?.path || '')
}

function openVariableDetail(row: TreeNodeLike): void {
  variableDetailNode.value = row
  variableDetailVisible.value = true
}

function resolveVariableTableRowClass({ row }: { row: TreeNodeLike }): string {
  const classes = [`node-${row.type}`]
  if (isNodeSelected(row)) classes.push('is-selected')
  return classes.join(' ')
}

function matchesVariableNode(node: TreeNodeLike, ignoreSearch = false): boolean {
  if (node.type !== 'variable') return false
  const search = variableSearchKey.value.trim().toLowerCase()
  const type = String(node.meta?.type || 'string')
  const kindMatched =
    !variableKindFilter.value ||
    (variableKindFilter.value === 'mapped' && isMappedVariableNode(node)) ||
    (variableKindFilter.value === 'project' && !isMappedVariableNode(node))
  const typeMatched = !variableDataTypeFilter.value || type === variableDataTypeFilter.value
  const searchMatched =
    ignoreSearch ||
    !search ||
    node.label.toLowerCase().includes(search) ||
    String(node.meta?.description || '')
      .toLowerCase()
      .includes(search) ||
    String(node.meta?.source?.path || '')
      .toLowerCase()
      .includes(search)
  return kindMatched && typeMatched && searchMatched
}

function filterVariableTree(nodes: TreeNodeLike[], ancestorSearchMatched = false): TreeNodeLike[] {
  const search = variableSearchKey.value.trim().toLowerCase()
  return nodes
    .map((node) => {
      if (node.type === 'variable') {
        return matchesVariableNode(node, ancestorSearchMatched) ? node : null
      }
      const groupSearchMatched = Boolean(search && node.label.toLowerCase().includes(search))
      const children = filterVariableTree(
        node.children || [],
        ancestorSearchMatched || groupSearchMatched,
      )
      if (!children.length) return null
      return {
        ...node,
        children,
      }
    })
    .filter(Boolean) as TreeNodeLike[]
}

function collectVariableNodes(nodes: TreeNodeLike[]): TreeNodeLike[] {
  return nodes.flatMap((node) => {
    if (node.type === 'variable') return [node]
    return collectVariableNodes(node.children || [])
  })
}

function isNodeSelected(data: TreeNodeLike): boolean {
  return selectedNodes.value.some((node) => node.id === data.id)
}

function handleNodeClick(data: TreeNodeLike, event: ClickLike): void {
  const isCtrl = Boolean(event?.ctrlKey || event?.metaKey)
  if (isCtrl) {
    if (isNodeSelected(data)) {
      selectedNodes.value = selectedNodes.value.filter((node) => node.id !== data.id)
    } else {
      selectedNodes.value = [...selectedNodes.value, data]
    }
  } else {
    selectedNodes.value = [data]
  }
  selectedNode.value = data
  treeRef.value?.setCurrentKey?.(data.id)
  if (contextMenuVisible.value) closeContextMenu()
}

function getMenuItemCount(nodeType: string): number {
  if (nodeType === 'blank') return 4
  if (nodeType === 'variable') return 4
  if (nodeType === 'group') return 4
  return 4
}

function setContextMenuPosition(event: MouseEvent, nodeType: string): void {
  const width = 180
  const itemHeight = 38
  const height = getMenuItemCount(nodeType) * itemHeight + 12
  const maxX = window.innerWidth - width - 8
  const maxY = window.innerHeight - height - 8
  const x = Math.max(8, Math.min(event.clientX, maxX))
  const y = Math.max(8, Math.min(event.clientY, maxY))
  contextMenuPosition.value = { x, y }
}

function handleContextMenu(event: MouseEvent, data: TreeNodeLike): void {
  event.preventDefault()
  event.stopPropagation()
  if (!isNodeSelected(data)) {
    selectedNodes.value = [data]
  }
  selectedNode.value = data
  contextMenuNode.value = data
  setContextMenuPosition(event, data.type)
  contextMenuVisible.value = true
  showMoveToMenu.value = false
}

function handleBlankContextMenu(event: MouseEvent): void {
  event.preventDefault()
  selectedNode.value = null
  selectedNodes.value = []
  contextMenuNode.value = { id: '__blank__', type: 'blank', label: '' }
  setContextMenuPosition(event, 'blank')
  contextMenuVisible.value = true
  showMoveToMenu.value = false
}

function closeContextMenu() {
  contextMenuVisible.value = false
  contextMenuNode.value = null
  showMoveToMenu.value = false
}

function openCreateFromMenu() {
  closeContextMenu()
  openCreate()
}

function openQuickAddFromMenu() {
  closeContextMenu()
  openQuickAdd()
}

function pasteVarFromMenu() {
  if (!varClipboard.value) return
  closeContextMenu()
  pasteVar()
}

function openEditFromMenu() {
  if (!canEditSelection.value) return
  closeContextMenu()
  openEdit()
}

function openGroupEditFromMenu() {
  if (!canEditSelection.value) return
  closeContextMenu()
  openGroupEdit()
}

function openGroupCreateFromMenu() {
  closeContextMenu()
  openGroupCreate()
}

/** 工具栏编辑按钮：根据选中类型分发到变量编辑或分组编辑 */
function handleToolbarEdit(): void {
  if (!canEditSelection.value) return
  if (selectedVariable.value) {
    openEdit()
  } else if (selectedGroup.value) {
    openGroupEdit()
  }
}

/** 双击节点：直接打开编辑弹窗 */
function handleNodeDblClick(_event: MouseEvent, data: TreeNodeLike): void {
  if (contextMenuVisible.value) closeContextMenu()
  // 确保选中状态一致
  selectedNodes.value = [data]
  selectedNode.value = data
  treeRef.value?.setCurrentKey?.(data.id)
  if (data.type === 'variable') {
    openEdit()
  } else if (data.type === 'group') {
    openGroupEdit()
  }
}

/** checkbox 切换：只作用于变量节点，勾选加入多选，取消则移除 */
function handleCheckboxChange(data: TreeNodeLike, checked: boolean): void {
  if (data.type !== 'variable') return
  if (checked) {
    if (!isNodeSelected(data)) {
      selectedNodes.value = [...selectedNodes.value, data]
    }
  } else {
    selectedNodes.value = selectedNodes.value.filter((node) => node.id !== data.id)
  }
  selectedNode.value = checked
    ? data
    : (selectedNodes.value[selectedNodes.value.length - 1] ?? null)
}

function resolveGroupVariableNodes(data: TreeNodeLike): TreeNodeLike[] {
  if (data.type !== 'group') return []
  return collectVariableNodes(data.children || [])
}

function getGroupSelectionState(data: TreeNodeLike): { checked: boolean; indeterminate: boolean } {
  const variables = resolveGroupVariableNodes(data)
  if (!variables.length) return { checked: false, indeterminate: false }
  const selectedCount = variables.filter((node) => isNodeSelected(node)).length
  return {
    checked: selectedCount === variables.length,
    indeterminate: selectedCount > 0 && selectedCount < variables.length,
  }
}

function mergeVariableSelection(nodes: TreeNodeLike[], checked: boolean): void {
  const targetIds = new Set(nodes.map((node) => node.id))
  const retained = selectedNodes.value.filter(
    (node) => node.type !== 'variable' || !targetIds.has(node.id),
  )
  selectedNodes.value = checked ? [...retained, ...nodes] : retained
  selectedNode.value = selectedNodes.value[selectedNodes.value.length - 1] ?? null
}

function handleGroupCheckboxChange(data: TreeNodeLike, checked: boolean): void {
  const variables = resolveGroupVariableNodes(data)
  mergeVariableSelection(variables, checked)
  if (contextMenuVisible.value) closeContextMenu()
}

function selectFilteredVariables(): void {
  selectedNodes.value = filteredVariableNodes.value
  selectedNode.value = selectedNodes.value[selectedNodes.value.length - 1] ?? null
  if (selectedNode.value) treeRef.value?.setCurrentKey?.(selectedNode.value.id)
  if (contextMenuVisible.value) closeContextMenu()
}

function clearSelectedNodes(): void {
  selectedNodes.value = []
  selectedNode.value = null
  treeRef.value?.setCurrentKey?.('')
  if (contextMenuVisible.value) closeContextMenu()
}

function handleMoveTo(groupIdValue: string | null): void {
  if (!contextMenuNode.value) return
  const node = contextMenuNode.value
  const nodesToMove = selectedNodes.value.length ? selectedNodes.value : [node]
  closeContextMenu()
  const nextVariables: Record<string, any> = {
    ...((projectVariables.value || {}) as Record<string, any>),
  }
  let nextGroups = ((projectVariableGroups.value as VariableGroupLike[] | undefined) || []).slice()
  let blocked = false

  nodesToMove.forEach((item) => {
    if (item.type === 'variable' && item.name) {
      nextVariables[item.name] = {
        ...nextVariables[item.name],
        groupId: groupIdValue,
      }
    } else if (item.type === 'group') {
      if (groupIdValue && isDescendantGroup(groupIdValue, item.id)) {
        blocked = true
        return
      }
      nextGroups = nextGroups.map((group: VariableGroupLike) =>
        group.id === item.id ? { ...group, parentId: groupIdValue } : group,
      )
    }
  })

  if (blocked) {
    showWarning(t('datapointPanel.moveToChildGroupBlocked'))
  }

  projectVariables.value = nextVariables
  projectVariableGroups.value = nextGroups
  persistProjectGlobals()
}

function allowDrag() {
  return true
}

function allowDrop(
  draggingNode: TreeDropNodeLike,
  dropNode: TreeDropNodeLike,
  type: string,
): boolean {
  const dragData = draggingNode.data
  const dropData = dropNode.data

  if (dragData.type === 'variable') {
    if (type === 'inner' && dropData.type !== 'group') return false
    return true
  }

  if (dragData.type === 'group') {
    if (type === 'inner' && dropData.type !== 'group') return false
    if (dropData.type === 'group' && isDescendantGroup(dropData.id, dragData.id)) {
      return false
    }
    if (type === 'inner') {
      const depth = getGroupDepth(dropData.id) + getGroupSubtreeDepth(dragData.id)
      return depth <= maxGroupDepth
    }
    return true
  }

  return false
}

function handleNodeDrop(
  draggingNode: TreeDropNodeLike,
  dropNode: TreeDropNodeLike,
  dropType: string,
): void {
  const dragData = draggingNode.data
  const targetGroupId = resolveTargetGroupId(dropNode, dropType)

  if (dragData.type === 'variable') {
    if (!dragData.name) return
    const nextVariables: Record<string, any> = {
      ...((projectVariables.value || {}) as Record<string, any>),
    }
    nextVariables[dragData.name] = {
      ...nextVariables[dragData.name],
      groupId: targetGroupId,
    }
    projectVariables.value = nextVariables
    persistProjectGlobals()
    return
  }

  if (dragData.type === 'group') {
    const groups = ((projectVariableGroups.value as VariableGroupLike[] | undefined) || []).slice()
    projectVariableGroups.value = groups.map((group: VariableGroupLike) =>
      group.id === dragData.id ? { ...group, parentId: targetGroupId } : group,
    )
    persistProjectGlobals()
  }
}

function resolveTargetGroupId(dropNode: TreeDropNodeLike, dropType: string): string | null {
  const dropData = dropNode.data
  if (dropType === 'inner') {
    return dropData.type === 'group' ? dropData.id : null
  }
  const parent = dropNode.parent?.data
  return parent?.type === 'group' ? parent.id : null
}

function getGroupDepth(groupIdValue: string | null | undefined): number {
  if (!groupIdValue) return 0
  let depth = 1
  let currentId = groupIdValue
  const groups = ((projectVariableGroups.value as VariableGroupLike[] | undefined) || []).slice()
  const map = new Map(groups.map((group: VariableGroupLike) => [group.id, group] as const))
  while (map.get(currentId)?.parentId) {
    depth += 1
    currentId = map.get(currentId)?.parentId || ''
  }
  return depth
}

function getGroupSubtreeDepth(groupIdValue: string): number {
  const groups = ((projectVariableGroups.value as VariableGroupLike[] | undefined) || []).slice()
  const children = groups.filter((group: VariableGroupLike) => group.parentId === groupIdValue)
  if (!children.length) return 1
  const depths = children.map((child: VariableGroupLike) => getGroupSubtreeDepth(child.id))
  return 1 + Math.max(...depths)
}

function isDescendantGroup(targetId: string, parentId: string): boolean {
  const groups = ((projectVariableGroups.value as VariableGroupLike[] | undefined) || []).slice()
  let current = groups.find((group: VariableGroupLike) => group.id === targetId)
  if (!current) return false
  while (current?.parentId) {
    if (current.parentId === parentId) return true
    current = groups.find((group) => group.id === (current?.parentId || ''))
  }
  return false
}

function defaultEditValue(type: string): string | number | boolean | null {
  switch (type) {
    case 'string':
      return ''
    case 'number':
      return 0
    case 'boolean':
      return false
    case 'array':
      return '[]'
    case 'object':
      return '{}'
    case 'set':
      return '[]'
    case 'map':
      return '[]'
    case 'date':
      return null
    case 'regexp':
      return '/pattern/g'
    case 'function':
      return 'function(){}'
    default:
      return ''
  }
}

function parseEditValue(type: string, value: unknown): unknown {
  if (type === 'number') {
    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : 0
  }
  if (type === 'boolean') {
    return Boolean(value)
  }
  if (type === 'date') {
    return value || null
  }
  if (['array', 'object', 'set', 'map'].includes(type)) {
    if (value instanceof Set) return Array.from(value)
    if (value instanceof Map) return Array.from(value.entries())
    if (Array.isArray(value)) return value
    if (value && typeof value === 'object') {
      if (type === 'map') return Object.entries(value)
      if (type === 'set') return Object.values(value)
      if (type === 'object') return value
    }
    if (value && typeof value === 'string') {
      try {
        const parsed = JSON.parse(value)
        if (type === 'array') return Array.isArray(parsed) ? parsed : []
        if (type === 'set') {
          if (Array.isArray(parsed)) return parsed
          if (parsed && typeof parsed === 'object') return Object.values(parsed)
          return []
        }
        if (type === 'map') {
          if (Array.isArray(parsed)) return parsed
          if (parsed && typeof parsed === 'object') return Object.entries(parsed)
          return []
        }
        if (parsed && typeof parsed === 'object') return parsed
      } catch {
        if (type === 'array' || type === 'set' || type === 'map') return []
        return {}
      }
    }
    if (type === 'array' || type === 'set' || type === 'map') return []
    return {}
  }
  return value ?? ''
}

function resetEditValue() {
  editValue.value = defaultEditValue(editType.value)
  editValueHasErrors.value = false
}

function parseStructuredJson(
  value: unknown,
  type: string,
): { ok: true; parsed: unknown } | { ok: false; error: string } {
  if (!isStructuredType.value) return { ok: true, parsed: value }
  if (typeof value !== 'string') return { ok: true, parsed: value }
  try {
    const parsed = JSON.parse(value)
    if (type === 'array' && !Array.isArray(parsed)) {
      return { ok: false, error: t('datapointPanel.invalidArrayJson') }
    }
    if (type === 'object') {
      if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
        return { ok: false, error: t('datapointPanel.invalidObjectJson') }
      }
    }
    if (type === 'set' || type === 'map') {
      if (!Array.isArray(parsed) && (!parsed || typeof parsed !== 'object')) {
        return { ok: false, error: t('datapointPanel.invalidSetMapJson') }
      }
    }
    return { ok: true, parsed }
  } catch {
    return { ok: false, error: t('datapointPanel.invalidJson') }
  }
}

function validateStructuredValue() {
  if (!isStructuredType.value) return true
  const result = parseStructuredJson(editValue.value, editType.value)
  if (!result.ok) {
    showError(result.error || t('datapointPanel.validationFailed'))
    return false
  }
  return true
}

function formatValue(value: unknown): string {
  if (value === null || value === undefined) return ''
  if (value instanceof Set) {
    return JSON.stringify(Array.from(value))
  }
  if (value instanceof Map) {
    return JSON.stringify(Array.from(value.entries()))
  }
  if (typeof value === 'object') {
    try {
      return JSON.stringify(value)
    } catch {
      return ''
    }
  }
  return String(value)
}

async function loadDataSourcesForMapping() {
  if (!projectId.value) return
  try {
    const result = await datacenterApi.getConnections(projectId.value, {
      page: 1,
      limit: 200,
    })
    const body = unwrapApiData(result)
    dataSources.value = requireConnectionsPayload(body)
  } catch {
    dataSources.value = []
    showError(t('datapointPanel.invalidConnectionsPayload'))
  }
}

async function persistProjectGlobals() {
  if (!projectId.value) {
    showError(t('datapointPanel.missingProject'))
    return
  }
  const result = await editorStore.saveProjectSettings()
  if (!result.ok) {
    showError(result.error?.message || t('datapointPanel.saveFailed'))
  } else {
    showSuccess(t('datapointPanel.saved'))
  }
}

async function openCreate() {
  editMode.value = false
  originalName.value = ''
  editName.value = ''
  editType.value = 'string'
  resetEditValue()
  editDescription.value = ''
  editGroupId.value = selectedGroupId.value || ROOT_GROUP_ID
  mapped.value = false
  mappedField.value = ''
  mappedSourceLabel.value = ''
  await loadDataSourcesForMapping()
  editVisible.value = true
}

async function openEdit() {
  if (selectedNodes.value.length > 1) {
    showWarning(t('datapointPanel.multiEditBlocked'))
    return
  }
  if (!selectedVariable.value) return
  editMode.value = true
  originalName.value = String(selectedVariable.value.name || '')
  editName.value = String(selectedVariable.value.name || '')
  editType.value = selectedVariable.value.detail?.type || 'string'
  editValue.value = String(formatValue(selectedVariable.value.detail?.default) || '')
  editDescription.value = selectedVariable.value.detail?.description || ''
  editGroupId.value = selectedVariable.value.detail?.groupId || ROOT_GROUP_ID
  mapped.value = selectedVariable.value.detail?.source?.type === 'dataCenter'
  await loadDataSourcesForMapping()

  if (mapped.value) {
    const path = selectedVariable.value.detail?.source?.path || ''
    mappedField.value = path
    mappedSourceLabel.value = String(
      getDatapointSourceLabel(selectedVariable.value.detail?.source?.sourceType || '') || '',
    )
  } else {
    mappedField.value = ''
    mappedSourceLabel.value = ''
  }
  editVisible.value = true
}

async function saveEdit() {
  const name = editName.value.trim()
  if (!name) return showWarning(t('datapointPanel.variableNameRequired'))

  const current = (projectVariables.value || {}) as VariableMapLike
  if ((!editMode.value || name !== originalName.value) && current[name]) {
    return showWarning(t('datapointPanel.variableNameDuplicated'))
  }

  if (mapped.value && !mappedField.value.trim()) {
    return showWarning(t('datapointPanel.mappedFieldRequired'))
  }
  if (mapped.value && selectedVariable.value?.detail?.type) {
    editType.value = selectedVariable.value.detail.type
  }
  if (mapped.value && selectedVariable.value?.detail?.type) {
    editType.value = selectedVariable.value.detail.type
  }
  if (editValueHasErrors.value) {
    return showError(t('datapointPanel.initialValueSyntaxError'))
  }
  if (isStructuredType.value && !validateStructuredValue()) {
    return
  }

  const nextVariables: VariableMapLike = { ...((projectVariables.value || {}) as VariableMapLike) }
  if (editMode.value && name !== originalName.value) {
    delete nextVariables[originalName.value]
  }

  const value = parseEditValue(editType.value, editValue.value)
  const groupIdValue = editGroupId.value === ROOT_GROUP_ID ? null : editGroupId.value
  const next: VariableDetailLike = {
    type: editType.value,
    default: value,
    groupId: groupIdValue,
  }

  if (editDescription.value) {
    next.description = editDescription.value
  }

  if (mapped.value && mappedField.value) {
    const sourcePath = mappedField.value
    next.mapped = true
    next.source = {
      type: 'dataCenter',
      path: sourcePath,
      sourceType: selectedVariable.value?.detail?.source?.sourceType || '',
      sourceId: selectedVariable.value?.detail?.source?.sourceId || '',
      datapointId: selectedVariable.value?.detail?.source?.datapointId || '',
    }
  }

  nextVariables[name] = next
  projectVariables.value = nextVariables
  editVisible.value = false
  await persistProjectGlobals()
  nextTick(() => {
    treeRef.value?.setCurrentKey?.(`var:${name}`)
  })
}

async function removeVar() {
  if (contextMenuVisible.value) closeContextMenu()
  if (!canDeleteSelection.value) {
    showWarning(t('datapointPanel.mixedDeleteBlocked'))
    return
  }
  if (!selectedNodes.value.length && !selectedVariable.value) return
  const variablesToRemove = selectedNodes.value
    .filter((node) => node.type === 'variable')
    .map((node) => node.name)
  const groupsToRemove = selectedNodes.value
    .filter((node) => node.type === 'group')
    .map((node) => node.id)
  try {
    const count = variablesToRemove.length + groupsToRemove.length
    let message = t('datapointPanel.deleteVariableNamed', {
      name: selectedVariable.value?.name || '',
    })
    if (count > 1) {
      message = t('datapointPanel.deleteSelectedCount', { count })
    } else if (groupsToRemove.length === 1 && variablesToRemove.length === 0) {
      const name = selectedNodes.value.find((node) => node.type === 'group')?.label || ''
      message = t('datapointPanel.deleteGroupNamed', { name })
    }
    await ElMessageBox.confirm(message, t('datapointPanel.deleteTitle'), {
      type: 'warning',
      lockScroll: false,
    })
  } catch {
    return
  }
  const nextVariables: VariableMapLike = { ...((projectVariables.value || {}) as VariableMapLike) }
  variablesToRemove.forEach((name: string | undefined) => {
    if (!name) return
    delete nextVariables[name]
  })

  if (groupsToRemove.length) {
    const parentMap = new Map()
    ;((projectVariableGroups.value as VariableGroupLike[] | undefined) || []).forEach((group) => {
      if (groupsToRemove.includes(group.id)) {
        parentMap.set(group.id, group.parentId || null)
      }
    })

    const nextGroups = ((projectVariableGroups.value as VariableGroupLike[] | undefined) || [])
      .filter((group) => !groupsToRemove.includes(group.id))
      .map((group) => {
        const parentId = group.parentId || ''
        return groupsToRemove.includes(parentId)
          ? { ...group, parentId: parentMap.get(parentId) || null }
          : group
      }) as VariableGroupLike[]

    Object.entries(nextVariables).forEach(([name, detail]) => {
      if (groupsToRemove.includes(detail?.groupId || '')) {
        nextVariables[name] = {
          ...detail,
          groupId: parentMap.get(detail.groupId || '') || null,
        }
      }
    })

    projectVariableGroups.value = nextGroups
  }

  projectVariables.value = nextVariables
  selectedNodes.value = []
  selectedNode.value = null
  await persistProjectGlobals()
}

function copyVar() {
  if (contextMenuVisible.value) closeContextMenu()
  const selectedVars = selectedNodes.value
    .filter((node) => node.type === 'variable')
    .map((node) => ({
      name: node.name,
      detail: node.name
        ? (projectVariables.value as Record<string, any> | undefined)?.[node.name]
        : {},
    }))
  if (!selectedVars.length && selectedVariable.value) {
    selectedVars.push({
      name: selectedVariable.value.name,
      detail: selectedVariable.value.detail,
    })
  }
  if (!selectedVars.length) {
    showWarning(t('datapointPanel.copySelectFirst'))
    return
  }
  varClipboard.value = {
    items: selectedVars.map((item) => ({
      name: item.name || t('datapointPanel.variableFallback'),
      detail: JSON.parse(JSON.stringify(item.detail || {})),
    })),
  }
  showSuccess(t('datapointPanel.copiedCount', { count: varClipboard.value.items.length }))
}

async function pasteVar() {
  if (contextMenuVisible.value) closeContextMenu()
  if (!varClipboard.value?.items?.length) return
  const targetGroupId = selectedGroupId.value || null
  const nextVariables = { ...(projectVariables.value || {}) }
  let lastName = ''

  varClipboard.value.items.forEach((item) => {
    const baseName = item.name || t('datapointPanel.variableFallback')
    let name = baseName
    let index = 1
    while (nextVariables[name]) {
      name = `${baseName}_copy${index}`
      index += 1
    }
    nextVariables[name] = {
      ...item.detail,
      groupId: targetGroupId ?? item.detail?.groupId ?? null,
    }
    lastName = name
  })

  projectVariables.value = nextVariables
  await persistProjectGlobals()
  if (lastName) {
    nextTick(() => treeRef.value?.setCurrentKey?.(`var:${lastName}`))
  }
}

function openGroupCreate() {
  groupEditMode.value = false
  groupId.value = ''
  groupName.value = ''
  groupParentId.value = selectedGroup.value?.id || null
  if (groupParentId.value && getGroupDepth(groupParentId.value) >= maxGroupDepth) {
    showWarning(`分组最多支持 ${maxGroupDepth} 层`)
    groupParentId.value = null
  }
  groupVisible.value = true
}

function openGroupEdit() {
  if (selectedNodes.value.filter((node) => node.type === 'group').length > 1) {
    showWarning(t('datapointPanel.multiEditBlocked'))
    return
  }
  if (!selectedGroup.value) return
  groupEditMode.value = true
  groupId.value = selectedGroup.value.id
  groupName.value = selectedGroup.value.name
  groupParentId.value = selectedGroup.value.parentId || null
  groupVisible.value = true
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
  const depth = parentId ? getGroupDepth(parentId) + 1 : 1
  if (depth > maxGroupDepth) {
    return showWarning(`分组最多支持 ${maxGroupDepth} 层`)
  }

  const groups = ((projectVariableGroups.value as VariableGroupLike[] | undefined) || []).slice()
  if (groupEditMode.value) {
    projectVariableGroups.value = groups.map((group: VariableGroupLike) =>
      group.id === groupId.value ? { ...group, name, parentId } : group,
    )
  } else {
    projectVariableGroups.value = [...groups, { id: createId(), name, parentId, sortOrder: 0 }]
  }
  groupVisible.value = false
  await persistProjectGlobals()
}

async function removeGroup() {
  await removeVar()
}

async function openQuickAdd() {
  quickVisible.value = true
  fields.value = []
  selectedFields.value = []
  quickActiveField.value = null
  quickPage.value = 1
  await loadDatapoints()
}

function getDatapointSourceLabel(sourceType: string): string {
  if (!sourceType) return t('datapointPanel.sourceUnknown')
  if (sourceType.includes('query')) return t('datapointPanel.sourceQuery')
  if (sourceType.includes('subscription')) return t('datapointPanel.sourceSubscription')
  if (sourceType.includes('tag')) return t('datapointPanel.sourceSubscription')
  return t('datapointPanel.sourceDatapoint')
}

function normalizeDatapointType(type: string): string {
  const normalized = String(type || '').toLowerCase()
  if (
    normalized.includes('int') ||
    normalized.includes('float') ||
    normalized.includes('double') ||
    normalized.includes('decimal') ||
    normalized.includes('number')
  ) {
    return 'number'
  }
  if (normalized.includes('bool')) return 'boolean'
  if (normalized.includes('array')) return 'array'
  if (normalized.includes('map')) return 'map'
  if (normalized.includes('set')) return 'set'
  if (normalized.includes('date') || normalized.includes('time')) return 'date'
  if (normalized.includes('object') || normalized.includes('json')) return 'object'
  return 'string'
}

function normalizeDatapointStatus(status: unknown): string {
  return String(status || 'active').toLowerCase()
}

function getQuickStatusLabel(status: string): string {
  const normalized = normalizeDatapointStatus(status)
  if (normalized === 'active' || normalized === 'online') {
    return t('datapointPanel.quickAddDialog.statusActive')
  }
  if (normalized === 'invalid' || normalized === 'inactive' || normalized === 'deleted') {
    return t('datapointPanel.quickAddDialog.statusInvalid')
  }
  if (normalized === 'disabled') {
    return t('datapointPanel.quickAddDialog.statusDisabled')
  }
  return status || t('datapointPanel.sourceUnknown')
}

function isQuickDatapointInvalid(field: Pick<DatapointFieldLike, 'status'>): boolean {
  return ['invalid', 'disabled', 'inactive', 'deleted'].includes(
    normalizeDatapointStatus(field.status),
  )
}

function resolveQuickMappedName(
  field: Pick<DatapointFieldLike, 'id' | 'path' | 'name'>,
  variables: VariableMapLike = (projectVariables.value || {}) as VariableMapLike,
): string {
  return findMappedProjectVariableName(field, variables) || ''
}

function buildQuickMappingLabel(field: Pick<DatapointFieldLike, 'status' | 'mappedName'>): string {
  if (field.mappedName) {
    return t('datapointPanel.quickAddDialog.mappedTo', { name: field.mappedName })
  }
  if (isQuickDatapointInvalid(field)) {
    return t('datapointPanel.quickAddDialog.invalid')
  }
  return t('datapointPanel.quickAddDialog.unmapped')
}

function resolveQuickStatusType(
  field: Pick<DatapointFieldLike, 'status' | 'mappedName'>,
): 'success' | 'info' | 'warning' {
  if (field.mappedName) return 'success'
  if (isQuickDatapointInvalid(field)) return 'warning'
  return 'info'
}

function formatDatapointTime(value: unknown): string {
  if (!value) return ''
  const date = dayjs(value as any)
  if (!date.isValid()) return String(value)
  return date.format(TIME_FORMAT)
}

async function loadDatapoints() {
  if (!projectId.value) {
    fields.value = []
    return
  }
  quickLoading.value = true
  try {
    const { datapoints, pagination } = await dataServiceApi.listDataPoints(projectId.value, {
      page: quickPage.value,
      pageSize: quickPageSize.value,
      search: searchKey.value.trim(),
      status: quickStatusFilter.value,
      type: quickTypeFilter.value.trim(),
      sourceId: quickSourceIdFilter.value.trim(),
    })
    quickTotal.value = Number(pagination.total || datapoints.length || 0)
    fields.value = (datapoints as Array<Record<string, unknown>>).map((item) => {
      const status = normalizeDatapointStatus(item.status)
      const fieldBase = {
        id: String(item.id || item.datapointId || item.path || ''),
        name: String(item.name || item.path || item.id || ''),
        path: String(item.path || item.name || item.id || ''),
        sourceType: String(item.sourceType || ''),
        sourceId: String(item.sourceId || ''),
        sourceLabel: getDatapointSourceLabel(String(item.sourceType || '')),
        type: normalizeDatapointType(String(item.dataType || item.type || '')),
        typeLabel: String(item.dataType || item.type || 'string'),
        description: String(item.description || ''),
        status,
        statusLabel: getQuickStatusLabel(status),
        mappedName: '',
        mappingLabel: '',
        statusType: 'info' as const,
        updatedAtLabel: formatDatapointTime(item.updated_at || item.updatedAt),
      } satisfies DatapointFieldLike
      const mappedName = resolveQuickMappedName(fieldBase)
      const field = { ...fieldBase, mappedName }
      return {
        ...field,
        mappingLabel: buildQuickMappingLabel(field),
        statusType: resolveQuickStatusType(field),
      }
    })
  } catch {
    fields.value = []
    quickTotal.value = 0
    showError(t('datapointPanel.invalidDatapointsPayload'))
  } finally {
    quickLoading.value = false
  }
}

const filteredFields = computed(() => fields.value)

function handleQuickPageChange(page: number): void {
  quickPage.value = page
  loadDatapoints()
}

function handleQuickSizeChange(size: number): void {
  quickPageSize.value = size
  quickPage.value = 1
  loadDatapoints()
}

function onSelectFields(rows: any): void {
  const selectedInCurrentPage = new Map<string, DatapointFieldLike>(
    ((rows || []) as DatapointFieldLike[]).map((field) => [resolveQuickFieldKey(field), field]),
  )
  const currentPageKeys = new Set(fields.value.map((field) => resolveQuickFieldKey(field)))
  const selectedMap = new Map<string, DatapointFieldLike>(
    selectedFields.value.map((field) => [resolveQuickFieldKey(field), field]),
  )

  currentPageKeys.forEach((key) => {
    if (!selectedInCurrentPage.has(key)) {
      selectedMap.delete(key)
    }
  })
  selectedInCurrentPage.forEach((field, key) => {
    selectedMap.set(key, field)
  })
  selectedFields.value = Array.from(selectedMap.values())
}

function buildRawVarName(field: string): string {
  let name = field
  if (replaceFrom.value) name = name.replace(replaceFrom.value, replaceTo.value)
  return `${prefix.value}${name}${suffix.value}`
}

function buildVarName(field: string): string {
  return normalizeProjectVariableName(
    buildRawVarName(field),
    Object.keys((projectVariables.value || {}) as VariableMapLike),
  )
}

function isQuickFieldSelectable(field: {
  name: string
  id?: unknown
  path?: unknown
  status?: unknown
  [key: string]: unknown
}): boolean {
  const normalizedField = {
    id: String(field.id || ''),
    name: String(field.name || ''),
    path: String(field.path || ''),
    status: normalizeDatapointStatus(field.status),
  }
  return !isQuickDatapointInvalid(normalizedField) && !resolveQuickMappedName(normalizedField)
}

function resolveQuickFieldKey(field: Pick<DatapointFieldLike, 'id' | 'path' | 'name'>): string {
  return String(field.id || field.path || field.name || '')
}

function resetQuickSingleDefaultValue(type: string): void {
  quickSingleDefaultValue.value = String(defaultEditValue(type) ?? '')
}

function parseQuickDefaultValue(type: string, value: string): unknown {
  if (type === 'boolean') {
    return value === 'true'
  }
  if (['array', 'object', 'set', 'map'].includes(type)) {
    try {
      return JSON.parse(value || (type === 'object' ? '{}' : '[]'))
    } catch {
      return type === 'object' ? {} : []
    }
  }
  return parseEditValue(type, value)
}

function openQuickSingleMapping(field: {
  id?: unknown
  name: string
  path?: unknown
  sourceType?: unknown
  sourceId?: unknown
  sourceLabel?: unknown
  type?: unknown
  typeLabel?: unknown
  description?: unknown
  status?: unknown
  statusLabel?: unknown
  statusType?: unknown
  mappingLabel?: unknown
  mappedName?: unknown
  updatedAtLabel?: unknown
  [key: string]: unknown
}): void {
  const normalizedField: DatapointFieldLike = {
    id: String(field.id || ''),
    name: String(field.name || ''),
    path: String(field.path || field.name || ''),
    sourceType: String(field.sourceType || ''),
    sourceId: String(field.sourceId || ''),
    sourceLabel: String(field.sourceLabel || ''),
    type: String(field.type || 'string'),
    typeLabel: String(field.typeLabel || field.type || 'string'),
    description: String(field.description || ''),
    status: normalizeDatapointStatus(field.status),
    statusLabel: String(field.statusLabel || getQuickStatusLabel(String(field.status || 'active'))),
    statusType: (field.statusType as DatapointFieldLike['statusType']) || 'info',
    mappingLabel: String(field.mappingLabel || ''),
    mappedName: String(field.mappedName || ''),
    updatedAtLabel: String(field.updatedAtLabel || ''),
  }
  const mappedName = resolveQuickMappedName(normalizedField)
  if (mappedName) {
    showSuccess(t('datapointPanel.quickAddDialog.mappedTo', { name: mappedName }))
    return
  }
  if (isQuickDatapointInvalid(normalizedField)) {
    showWarning(t('datapointPanel.quickAddDialog.invalid'))
    return
  }
  const type = normalizedField.type || 'string'
  quickActiveField.value = normalizedField
  quickSingleName.value = buildVarName(normalizedField.name)
  quickSingleType.value = type
  quickSingleGroupId.value = selectedGroup.value?.id || ROOT_GROUP_ID
  resetQuickSingleDefaultValue(type)
  quickSingleDescription.value = normalizedField.description || ''
}

function closeQuickSingleMapping(): void {
  quickActiveField.value = null
}

function handleQuickSingleTypeChange(type: string): void {
  quickSingleType.value = type
  resetQuickSingleDefaultValue(type)
}

function markQuickFieldMapped(field: DatapointFieldLike, mappedNameValue: string): void {
  const key = resolveQuickFieldKey(field)
  fields.value = fields.value.map((item) => {
    if (resolveQuickFieldKey(item) !== key) return item
    const mappedField = { ...item, mappedName: mappedNameValue }
    return {
      ...mappedField,
      mappingLabel: buildQuickMappingLabel(mappedField),
      statusType: resolveQuickStatusType(mappedField),
    }
  })
  selectedFields.value = selectedFields.value.filter((item) => resolveQuickFieldKey(item) !== key)
}

async function confirmQuickSingleMapping(): Promise<void> {
  const field = quickActiveField.value
  if (!field) return
  if (!quickSingleName.value.trim()) {
    showWarning(t('datapointPanel.variableNameRequired'))
    return
  }
  const nextVariables: VariableMapLike = { ...((projectVariables.value || {}) as VariableMapLike) }
  if (findMappedProjectVariableName(field, nextVariables)) {
    markQuickFieldMapped(field, resolveQuickMappedName(field, nextVariables))
    quickActiveField.value = null
    showSuccess(t('datapointPanel.quickAddDialog.allSelectedMapped'))
    return
  }
  const built = buildProjectVariableFromDataPoint(field, {
    existingNames: Object.keys(nextVariables),
    name: quickSingleName.value,
    type: quickSingleType.value || field.type || 'string',
    groupId: quickSingleGroupId.value === ROOT_GROUP_ID ? null : quickSingleGroupId.value,
    defaultValue: parseQuickDefaultValue(quickSingleType.value, quickSingleDefaultValue.value),
    description: quickSingleDescription.value,
  })
  nextVariables[built.name] = built.definition
  projectVariables.value = nextVariables
  await persistProjectGlobals()
  markQuickFieldMapped(field, built.name)
  quickActiveField.value = null
  showSuccess(t('datapointPanel.quickAddDialog.singleSaved', { name: built.name }))
}

async function confirmQuickAdd() {
  if (!quickSelectedMappableCount.value) {
    showWarning(t('datapointPanel.quickAddDialog.selectMappableFirst'))
    return
  }

  const targetGroupId = selectedGroup.value?.id || null
  const nextVariables: VariableMapLike = { ...((projectVariables.value || {}) as VariableMapLike) }
  let addedCount = 0

  selectedFields.value.forEach((field) => {
    if (!isQuickFieldSelectable(field)) return
    if (findMappedProjectVariableName(field, nextVariables)) return

    const built = buildProjectVariableFromDataPoint(field, {
      existingNames: Object.keys(nextVariables),
      name: buildRawVarName(field.name),
      type: field.type || 'object',
      groupId: targetGroupId,
      defaultValue: parseEditValue(
        field.type || 'object',
        defaultEditValue(field.type || 'object'),
      ),
      description: field.description,
    })
    nextVariables[built.name] = built.definition
    addedCount += 1
  })

  if (!addedCount) {
    quickVisible.value = false
    showSuccess(t('datapointPanel.quickAddDialog.allSelectedMapped'))
    return
  }

  projectVariables.value = nextVariables
  quickVisible.value = false
  await persistProjectGlobals()
}

function handleClickOutside() {
  if (contextMenuVisible.value) closeContextMenu()
}

function downloadBlob(content: BlobPart, name: string, type: string): void {
  const blob = new Blob([content], { type })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = name
  link.click()
  URL.revokeObjectURL(url)
}

function buildGroupPathMap(groups: any[]): Map<string, string> {
  const map = new Map<string, string>()
  const groupMap = new Map<string, Record<string, any>>(
    (groups || []).map((group: Record<string, any>) => [String(group.id || ''), group]),
  )

  const buildPath = (groupIdValue: string | null | undefined): string => {
    if (!groupIdValue || !groupMap.has(groupIdValue)) return ''
    if (map.has(groupIdValue)) return map.get(groupIdValue) || ''
    const group = groupMap.get(groupIdValue) as Record<string, any> | undefined
    if (!group) return ''
    const parentPath = buildPath((group.parentId as string | null | undefined) || null)
    const path = parentPath
      ? `${parentPath} / ${String(group.name || '')}`
      : String(group.name || '')
    map.set(groupIdValue, path)
    return path
  }

  ;(groups || []).forEach((group: Record<string, any>) => buildPath(String(group.id || '')))
  return map
}

function ensureGroupPath(groups: any[], path: string): string | null {
  if (!path) return null
  const segments = String(path)
    .split('/')
    .map((segment) => segment.trim())
    .filter(Boolean)
  if (!segments.length) return null
  let parentId: string | null = null
  segments.forEach((segment) => {
    let match = groups.find(
      (group) => group.name === segment && (group.parentId || null) === parentId,
    ) as Record<string, any> | undefined
    if (!match) {
      match = { id: createId(), name: segment, parentId, sortOrder: 0 }
      groups.push(match)
    }
    parentId = String(match.id || '')
  })
  return parentId
}

function buildExportRows(): Array<Record<string, string>> {
  const groupPathMap = buildGroupPathMap((projectVariableGroups.value || []) as VariableGroupLike[])
  return Object.entries((projectVariables.value || {}) as Record<string, any>).map(
    ([name, detail]) => ({
      name,
      type: String(detail?.type || 'string'),
      default: formatValue(detail?.default ?? detail?.value),
      description: String(detail?.description || ''),
      groupPath: detail?.groupId ? groupPathMap.get(String(detail.groupId)) || '' : '',
      mappedPath: String(detail?.source?.path || ''),
    }),
  )
}

function normalizeRowKey(row: ImportRowLike, key: string): unknown {
  const lowerKey = key.toLowerCase()
  const hit = Object.keys(row).find((k) => k.toLowerCase() === lowerKey)
  return hit ? row[hit] : ''
}

async function mergeImportedRows(rows: Array<Record<string, any>>): Promise<void> {
  const nextGroups = [...((projectVariableGroups.value as VariableGroupLike[] | undefined) || [])]
  const nextVariables: VariableMapLike = { ...((projectVariables.value || {}) as VariableMapLike) }
  let added = 0
  let skipped = 0

  rows.forEach((row) => {
    const name = String(normalizeRowKey(row, 'name') || '').trim()
    if (!name) return
    if (nextVariables[name]) {
      skipped += 1
      return
    }
    const type = String(normalizeRowKey(row, 'type') || 'string').trim()
    const defaultRaw = normalizeRowKey(row, 'default')
    const description = String(normalizeRowKey(row, 'description') || '')
    const groupPath = String(normalizeRowKey(row, 'groupPath') || '')
    const mappedPath = String(normalizeRowKey(row, 'mappedPath') || '')
    const groupIdValue = ensureGroupPath(nextGroups, groupPath)
    const parsedDefault = parseEditValue(type, defaultRaw)
    nextVariables[name] = {
      type,
      default: parsedDefault,
      description,
      groupId: groupIdValue,
    }
    if (mappedPath) {
      nextVariables[name].mapped = true
      nextVariables[name].source = { type: 'dataCenter', path: mappedPath }
    }
    added += 1
  })

  projectVariableGroups.value = nextGroups
  projectVariables.value = nextVariables
  await persistProjectGlobals()
  showSuccess(t('datapointPanel.importFinished', { added, skipped }))
}

async function mergeImportedDefinitions(
  definitions: Record<string, any>,
  groups: any[],
): Promise<void> {
  const nextGroups = [...((projectVariableGroups.value as VariableGroupLike[] | undefined) || [])]
  const nextVariables: VariableMapLike = { ...((projectVariables.value || {}) as VariableMapLike) }
  const importedGroups = Array.isArray(groups) ? groups : []
  const importedPathMap = buildGroupPathMap(importedGroups)
  const idToNew = new Map()

  importedGroups.forEach((group) => {
    const path = importedPathMap.get(group.id) || group.name
    const newId = ensureGroupPath(nextGroups, path)
    idToNew.set(group.id, newId)
  })

  Object.entries(definitions || {}).forEach(([name, detail]) => {
    if (nextVariables[name]) return
    const normalizedDetail = (detail || {}) as Record<string, any>
    const groupPath = normalizedDetail.groupId
      ? importedPathMap.get(String(normalizedDetail.groupId)) || ''
      : ''
    const groupIdValue = ensureGroupPath(nextGroups, groupPath)
    nextVariables[name] = {
      ...normalizedDetail,
      groupId: groupIdValue,
    }
  })

  projectVariableGroups.value = nextGroups
  projectVariables.value = nextVariables
  await persistProjectGlobals()
  showSuccess(t('datapointPanel.importFinishedSimple'))
}

async function handleExport(format: 'json' | 'csv' | 'xlsx'): Promise<void> {
  const rows = buildExportRows()
  if (format === 'json') {
    const payload = {
      definitions: projectVariables.value || {},
      groups: projectVariableGroups.value || [],
    }
    downloadBlob(JSON.stringify(payload, null, 2), 'project-variables.json', 'application/json')
    return
  }

  const worksheet = XLSX.utils.json_to_sheet(rows)
  const workbook = XLSX.utils.book_new()
  XLSX.utils.book_append_sheet(workbook, worksheet, 'variables')
  if (format === 'csv') {
    const csv = XLSX.utils.sheet_to_csv(worksheet)
    downloadBlob(csv, 'project-variables.csv', 'text/csv')
    return
  }
  const buffer = XLSX.write(workbook, { bookType: 'xlsx', type: 'array' })
  downloadBlob(
    buffer,
    'project-variables.xlsx',
    'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
  )
}

function handleImport(format: 'json' | 'csv' | 'xlsx'): void {
  importType.value = format
  nextTick(() => {
    if (importInputRef.value) {
      importInputRef.value.value = ''
      importInputRef.value.click()
    }
  })
}

function decodeTextBuffer(raw: ArrayBuffer): string {
  try {
    const bytes = raw instanceof ArrayBuffer ? new Uint8Array(raw) : new Uint8Array(raw)
    const hasUtf8Bom =
      bytes.length >= 3 && bytes[0] === 0xef && bytes[1] === 0xbb && bytes[2] === 0xbf
    const hasUtf16LeBom = bytes.length >= 2 && bytes[0] === 0xff && bytes[1] === 0xfe
    const hasUtf16BeBom = bytes.length >= 2 && bytes[0] === 0xfe && bytes[1] === 0xff
    const tryDecode = (encoding: string): string => {
      try {
        return new TextDecoder(encoding, { fatal: false }).decode(raw)
      } catch {
        return ''
      }
    }

    if (hasUtf8Bom) return tryDecode('utf-8')
    if (hasUtf16LeBom) return tryDecode('utf-16le')
    if (hasUtf16BeBom) return tryDecode('utf-16be')

    const utf8Text = tryDecode('utf-8')
    if (utf8Text && !utf8Text.includes('\uFFFD')) return utf8Text

    const sampleLen = Math.min(bytes.length, 2000)
    let zeroCount = 0
    let oddCount = 0
    for (let i = 1; i < sampleLen; i += 2) {
      oddCount += 1
      if (bytes[i] === 0) zeroCount += 1
    }
    if (oddCount > 0 && zeroCount / oddCount > 0.2) {
      const utf16Text = tryDecode('utf-16le')
      if (utf16Text) return utf16Text
    }

    const gbkText = tryDecode('gbk')
    if (gbkText) return gbkText

    return utf8Text || ''
  } catch {
    return ''
  }
}

async function handleFileChange(event: Event): Promise<void> {
  const target = event.target as HTMLInputElement | null
  const file = target?.files?.[0]
  if (!file) return
  if (importType.value === 'json') {
    const text = await file.text()
    try {
      const data = JSON.parse(text)
      if (data && typeof data === 'object' && (data.definitions || data.groups)) {
        await mergeImportedDefinitions(data.definitions || {}, (data.groups || []) as any[])
        return
      }
      if (Array.isArray(data)) {
        await mergeImportedRows(data as Array<Record<string, any>>)
        return
      }
      showError(t('datapointPanel.unsupportedJson'))
    } catch {
      try {
        const buffer = await file.arrayBuffer()
        const fallbackText = decodeTextBuffer(buffer)
        const data = JSON.parse(fallbackText)
        if (data && typeof data === 'object' && (data.definitions || data.groups)) {
          await mergeImportedDefinitions(data.definitions || {}, (data.groups || []) as any[])
          return
        }
        if (Array.isArray(data)) {
          await mergeImportedRows(data as Array<Record<string, any>>)
          return
        }
        showError(t('datapointPanel.unsupportedJson'))
      } catch {
        showError(t('datapointPanel.jsonParseFailed'))
      }
    }
    return
  }

  const buffer = await file.arrayBuffer()
  const workbook =
    importType.value === 'csv'
      ? XLSX.read(decodeTextBuffer(buffer), { type: 'string' })
      : XLSX.read(buffer, { type: 'array' })
  const sheetName = workbook.SheetNames[0]
  if (!sheetName) {
    showError(t('datapointPanel.noWorksheet'))
    return
  }
  const worksheet = workbook.Sheets[sheetName]
  if (!worksheet) {
    showError(t('datapointPanel.noWorksheet'))
    return
  }
  const rows = XLSX.utils.sheet_to_json(worksheet as any, {
    defval: '',
  }) as Array<Record<string, any>>
  await mergeImportedRows(rows)
}

function handleEditValueMarkers(markers: unknown): void {
  if (!isEditorType.value) {
    editValueHasErrors.value = false
    return
  }
  editValueHasErrors.value = ((markers || []) as Array<Record<string, any>>).some(
    (marker) => marker.severity === 8,
  )
}

let searchDebounceTimer: ReturnType<typeof setTimeout> | null = null
let variableTableResizeObserver: ResizeObserver | null = null
watch([searchKey, quickStatusFilter, quickTypeFilter, quickSourceIdFilter], () => {
  if (!quickVisible.value) return
  if (searchDebounceTimer) clearTimeout(searchDebounceTimer)
  searchDebounceTimer = setTimeout(() => {
    quickPage.value = 1
    selectedFields.value = []
    quickActiveField.value = null
    void loadDatapoints()
  }, 300)
})

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
  if (treeWrapRef.value) {
    variableTableWidth.value = treeWrapRef.value.clientWidth
    variableTableResizeObserver = new ResizeObserver(([entry]) => {
      variableTableWidth.value = Math.floor(entry?.contentRect.width || 0)
    })
    variableTableResizeObserver.observe(treeWrapRef.value)
  }
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
  variableTableResizeObserver?.disconnect()
  variableTableResizeObserver = null
})
</script>

<template>
  <div class="global-vars">
    <DatapointPanelToolbar
      :has-selection="toolbarHasSelection"
      :selection-type="toolbarSelectionType"
      :selected-count="toolbarSelectedCount"
      :can-edit="toolbarCanEdit"
      :can-delete="toolbarCanDelete"
      @create-var="openCreate"
      @create-group="openGroupCreate"
      @quick-add="openQuickAdd"
      @edit-selected="handleToolbarEdit"
      @delete-selected="removeVar"
      @clear-selection="clearSelectedNodes"
      @export-vars="handleExport"
      @import-vars="handleImport"
    />
    <input
      ref="importInputRef"
      class="hidden-file-input"
      type="file"
      :accept="importAccept"
      @change="handleFileChange"
    />
    <div class="variable-filter">
      <el-input
        v-model="variableSearchKey"
        class="variable-filter__search"
        size="small"
        clearable
        :placeholder="t('datapointPanel.variableSearchPlaceholder')"
      />
      <div class="variable-filter__row">
        <el-select
          v-model="variableKindFilter"
          class="variable-filter__select"
          size="small"
          :placeholder="t('datapointPanel.variableKindAll')"
        >
          <el-option :label="t('datapointPanel.variableKindAll')" value="" />
          <el-option :label="t('datapointPanel.variableKindProject')" value="project" />
          <el-option :label="t('datapointPanel.variableKindMapped')" value="mapped" />
        </el-select>
        <el-select
          v-model="variableDataTypeFilter"
          class="variable-filter__select"
          size="small"
          clearable
          :placeholder="t('datapointPanel.variableDataTypeAll')"
        >
          <el-option
            v-for="type in variableDataTypeOptions"
            :key="type"
            :label="type"
            :value="type"
          />
        </el-select>
      </div>
      <div class="variable-filter__summary">
        <span>{{ t('datapointPanel.variableResultCount', { count: filteredVariableCount }) }}</span>
        <el-button
          size="small"
          text
          :disabled="!filteredVariableCount"
          @click="selectFilteredVariables"
        >
          {{ t('datapointPanel.selectCurrentResults') }}
        </el-button>
      </div>
    </div>
    <div ref="treeWrapRef" class="tree-wrap" @contextmenu="handleBlankContextMenu">
      <el-table
        ref="treeRef"
        :data="filteredVariableTree"
        row-key="id"
        class="variable-table"
        height="100%"
        size="small"
        :row-class-name="resolveVariableTableRowClass"
        :default-expand-all="true"
        :tree-props="{ children: 'children' }"
        @row-click="
          (row: TreeNodeLike, _column: unknown, event: MouseEvent) => handleNodeClick(row, event)
        "
        @row-contextmenu="
          (row: TreeNodeLike, _column: unknown, event: MouseEvent) => handleContextMenu(event, row)
        "
        @row-dblclick="
          (row: TreeNodeLike, _column: unknown, event: MouseEvent) => handleNodeDblClick(event, row)
        "
      >
        <el-table-column :label="t('datapointPanel.variableColumnName')" min-width="96">
          <template #default="{ row }">
            <div class="variable-name-cell">
              <el-checkbox
                class="node-checkbox"
                :model-value="
                  row.type === 'group' ? getGroupSelectionState(row).checked : isNodeSelected(row)
                "
                :indeterminate="row.type === 'group' && getGroupSelectionState(row).indeterminate"
                @click.stop
                @change="
                  (val: boolean) =>
                    row.type === 'group'
                      ? handleGroupCheckboxChange(row, val)
                      : handleCheckboxChange(row, val)
                "
              />
              <el-icon
                class="node-icon"
                :class="{
                  'is-mapped': row.type === 'variable' && row.meta?.mapped,
                  'is-unmapped': row.type === 'variable' && !row.meta?.mapped,
                }"
              >
                <IconEpFolder v-if="row.type === 'group'" />
                <IconEpLink v-else-if="row.meta?.mapped" />
                <IconEpEditPen v-else />
              </el-icon>
              <span
                class="node-label"
                :class="{ 'is-group': row.type === 'group' }"
                :title="row.label"
              >
                {{ row.label }}
              </span>
            </div>
          </template>
        </el-table-column>
        <el-table-column
          v-if="showVariableKindColumn"
          :label="t('datapointPanel.variableColumnKind')"
          width="64"
        >
          <template #default="{ row }">
            <span class="variable-kind" :title="getVariableKindLabel(row)">
              {{ getVariableKindLabel(row) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column
          v-if="showVariableDataTypeColumn"
          :label="t('datapointPanel.variableColumnType')"
          width="72"
        >
          <template #default="{ row }">
            <span
              v-if="row.type === 'variable'"
              class="node-meta"
              :title="row.meta?.type || 'string'"
            >
              {{ row.meta?.type || 'string' }}
            </span>
            <span v-else class="node-meta" title="-">-</span>
          </template>
        </el-table-column>
        <el-table-column
          v-if="showVariableActionColumn"
          :label="t('datapointPanel.variableColumnAction')"
          width="40"
          align="center"
        >
          <template #default="{ row }">
            <el-button
              class="variable-detail-button"
              :aria-label="t('datapointPanel.variableDetailAction')"
              text
              size="small"
              @click.stop="openVariableDetail(row)"
            >
              <el-icon><IconEpView /></el-icon>
            </el-button>
          </template>
        </el-table-column>
      </el-table>
      <div v-if="!filteredVariableTree.length" class="tree-empty">
        <el-empty
          :description="
            variableTree.length ? t('datapointPanel.noVariableResults') : t('datapointPanel.empty')
          "
          :image-size="60"
        />
      </div>
    </div>
    <DatapointPanelContextMenu
      :context-menu-visible="contextMenuVisible"
      :context-menu-style="contextMenuStyle"
      :context-menu-node="contextMenuNode"
      :show-move-to-menu="showMoveToMenu"
      :submenu-style="submenuStyle"
      :var-clipboard="varClipboard"
      :can-edit-selection="canEditSelection"
      :can-delete-selection="canDeleteSelection"
      :available-groups="availableGroups"
      @open-group-create="openGroupCreateFromMenu"
      @open-create-var="openCreateFromMenu"
      @open-quick-add="openQuickAddFromMenu"
      @paste="pasteVarFromMenu"
      @open-edit="openEditFromMenu"
      @copy="copyVar"
      @toggle-move-to="showMoveToMenu = !showMoveToMenu"
      @remove-var="removeVar"
      @open-group-edit="openGroupEditFromMenu"
      @remove-group="removeGroup"
      @move-to="handleMoveTo"
    />

    <el-dialog
      v-model="variableDetailVisible"
      class="variable-detail-dialog"
      :title="variableDetailTitle"
      width="520px"
      append-to-body
    >
      <div v-if="variableDetailNode" class="variable-detail">
        <div class="variable-detail__hero">
          <div class="variable-detail__icon">
            <el-icon>
              <IconEpFolder v-if="variableDetailNode.type === 'group'" />
              <IconEpLink v-else-if="variableDetailNode.meta?.mapped" />
              <IconEpEditPen v-else />
            </el-icon>
          </div>
          <div class="variable-detail__heading">
            <strong>{{ variableDetailNode.label }}</strong>
            <span>{{ getVariableKindLabel(variableDetailNode) }}</span>
          </div>
        </div>

        <div class="variable-detail__meta">
          <div class="variable-detail__meta-item">
            <span>{{ t('datapointPanel.variableColumnType') }}</span>
            <strong v-if="variableDetailNode.type === 'variable'">
              {{ variableDetailNode.meta?.type || 'string' }}
            </strong>
            <strong v-else>-</strong>
          </div>
          <div class="variable-detail__meta-item">
            <span>{{ t('datapointPanel.variableColumnMapping') }}</span>
            <strong>
              {{
                getVariableMappingPath(variableDetailNode) ||
                t('datapointPanel.variableMappingNone')
              }}
            </strong>
          </div>
        </div>

        <div class="variable-detail__section">
          <div class="variable-detail__section-title">
            {{ t('datapointPanel.variableDetailDescription') }}
          </div>
          <div class="variable-detail__text">
            {{ variableDetailNode.meta?.description || '-' }}
          </div>
        </div>

        <div class="variable-detail__section">
          <div class="variable-detail__section-title">
            {{ t('datapointPanel.variableDetailDefault') }}
          </div>
          <pre>{{
            formatValue(variableDetailNode.meta?.default ?? variableDetailNode.meta?.value) || '-'
          }}</pre>
        </div>
      </div>
    </el-dialog>

    <DatapointVariableEditDialog
      v-model="editVisible"
      v-model:edit-name="editName"
      v-model:edit-group-id="editGroupId"
      v-model:edit-type="editType"
      v-model:edit-value="editValue"
      v-model:edit-description="editDescription"
      v-model:mapped="mapped"
      v-model:mapped-field="mappedField"
      v-model:mapped-source-label="mappedSourceLabel"
      :edit-mode="editMode"
      :types="types"
      :group-options="groupOptions"
      :root-group-id="ROOT_GROUP_ID"
      :is-editor-type="isEditorType"
      :is-text-type="isTextType"
      :editor-language="editorLanguage"
      @confirm="saveEdit"
      @type-change="resetEditValue"
      @edit-value-markers="handleEditValueMarkers"
    />
    <VariableGroupFormDialog
      v-model="groupVisible"
      v-model:name="groupName"
      v-model:parent-id="groupParentId"
      :edit-mode="groupEditMode"
      :parent-options="groupParentOptions"
      @confirm="saveGroup"
    />

    <DatapointQuickAddDialog
      v-model="quickVisible"
      v-model:search-key="searchKey"
      v-model:status-filter="quickStatusFilter"
      v-model:type-filter="quickTypeFilter"
      v-model:source-id-filter="quickSourceIdFilter"
      v-model:prefix="prefix"
      v-model:suffix="suffix"
      v-model:replace-from="replaceFrom"
      v-model:replace-to="replaceTo"
      v-model:single-name="quickSingleName"
      v-model:single-type="quickSingleType"
      v-model:single-group-id="quickSingleGroupId"
      v-model:single-default-value="quickSingleDefaultValue"
      v-model:single-description="quickSingleDescription"
      :quick-loading="quickLoading"
      :filtered-fields="filteredFields"
      :quick-page-size="quickPageSize"
      :quick-total="quickTotal"
      :quick-page="quickPage"
      :selected-count="quickSelectedCount"
      :selected-mappable-count="quickSelectedMappableCount"
      :selected-keys="quickSelectedKeys"
      :active-field="quickActiveField"
      :types="types"
      :source-options="quickSourceOptions"
      :group-options="groupOptions"
      :root-group-id="ROOT_GROUP_ID"
      :build-var-name="buildVarName"
      :is-field-selectable="isQuickFieldSelectable"
      @refresh="loadDatapoints"
      @selection-change="onSelectFields"
      @page-change="handleQuickPageChange"
      @size-change="handleQuickSizeChange"
      @status-click="openQuickSingleMapping"
      @single-close="closeQuickSingleMapping"
      @single-type-change="handleQuickSingleTypeChange"
      @single-confirm="confirmQuickSingleMapping"
      @confirm="confirmQuickAdd"
    />
  </div>
</template>

<style scoped>
.global-vars {
  padding: 12px;
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-height: 0;
  overflow: hidden;
}

.hidden-file-input {
  display: none;
}

.variable-filter {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 8px;
  border: 1px solid var(--designer-border-color);
  border-radius: 10px;
  background: var(--designer-shell-surface);
}

.variable-filter__search,
.variable-filter__select {
  width: 100%;
}

.variable-filter__row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 6px;
}

.variable-filter__summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-height: 24px;
  font-size: 12px;
  color: var(--designer-text-muted);
}

.variable-filter__summary :deep(.el-button) {
  height: 24px;
  padding: 0 4px;
}

.tree-wrap {
  flex: 1;
  overflow: hidden;
  border: 1px solid var(--designer-border-color);
  border-radius: 10px;
  position: relative;
  background: var(--designer-shell-surface);
  box-shadow: var(--designer-shadow-panel);
  min-height: 0;
}

.tree-empty {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--designer-shell-surface);
}

.variable-table {
  height: 100%;
  border-radius: 10px;
}

.variable-table :deep(.el-table__header th) {
  height: 34px;
  background: var(--designer-shell-muted);
  color: var(--designer-text-muted);
  font-size: 12px;
  font-weight: 600;
}

.variable-table :deep(.el-table__row) {
  cursor: default;
}

.variable-table :deep(.el-table__row.is-selected > td) {
  background: var(--designer-primary-soft) !important;
}

.variable-table :deep(.el-table__row.node-group > td) {
  background: rgba(148, 163, 184, 0.08);
}

.variable-table :deep(.el-table__cell) {
  padding: 5px 0;
}

.variable-name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  width: 100%;
}

.node-icon {
  color: var(--designer-text-muted);
  flex-shrink: 0;
}

.node-group .node-icon {
  color: var(--designer-primary-text);
}

.node-variable .node-icon {
  color: var(--designer-success-text);
}

.node-icon.is-mapped {
  color: var(--designer-primary-text);
}

.node-icon.is-unmapped {
  color: var(--designer-text-muted);
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

.node-meta {
  font-size: 12px;
  color: var(--designer-text-muted);
}

.variable-kind {
  font-size: 12px;
  color: var(--designer-text-muted);
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.variable-detail-button {
  width: 24px;
  height: 24px;
  padding: 0;
}

.variable-detail {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.variable-detail__hero {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border: 1px solid var(--designer-border-color);
  border-radius: 8px;
  background: var(--designer-shell-muted);
}

.variable-detail__icon {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  color: var(--designer-primary-text);
  background: var(--designer-primary-soft);
  flex: 0 0 auto;
}

.variable-detail__heading {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.variable-detail__heading strong {
  min-width: 0;
  color: var(--designer-text-primary);
  font-size: 15px;
  font-weight: 600;
  line-height: 1.3;
  overflow-wrap: anywhere;
}

.variable-detail__heading span {
  color: var(--designer-text-muted);
  font-size: 12px;
}

.variable-detail__meta {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.variable-detail__meta-item {
  min-width: 0;
  padding: 10px;
  border: 1px solid var(--designer-border-color);
  border-radius: 8px;
  background: var(--designer-shell-surface);
}

.variable-detail__meta-item span,
.variable-detail__section-title {
  display: block;
  color: var(--designer-text-muted);
  font-size: 12px;
  line-height: 1.4;
}

.variable-detail__meta-item strong {
  display: block;
  min-width: 0;
  margin-top: 5px;
  color: var(--designer-text-primary);
  font-size: 13px;
  font-weight: 500;
  overflow-wrap: anywhere;
}

.variable-detail__section {
  padding: 10px;
  border: 1px solid var(--designer-border-color);
  border-radius: 8px;
  background: var(--designer-shell-surface);
}

.variable-detail__text {
  margin-top: 6px;
  color: var(--designer-text-primary);
  font-size: 13px;
  line-height: 1.6;
  overflow-wrap: anywhere;
}

.variable-detail__section pre {
  max-height: 180px;
  margin: 6px 0 0;
  padding: 8px;
  overflow: auto;
  border: 1px solid var(--designer-border-color);
  border-radius: 6px;
  background: var(--designer-shell-muted);
  color: var(--designer-text-primary);
  font-size: 12px;
  line-height: 1.5;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.variable-table :deep(.el-table__inner-wrapper::before) {
  display: none;
}

.variable-table :deep(.el-table__body),
.variable-table :deep(.el-table__header) {
  width: 100% !important;
}

.variable-table :deep(.el-table__body-wrapper) {
  overflow-x: hidden;
}

.node-checkbox {
  flex-shrink: 0;
  margin-right: 2px;
}

:deep(.node-checkbox .el-checkbox__label) {
  display: none;
}

:deep(.node-checkbox .el-checkbox__inner) {
  width: 14px;
  height: 14px;
  border-radius: 3px;
}

.edit-value-block {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.edit-value-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
