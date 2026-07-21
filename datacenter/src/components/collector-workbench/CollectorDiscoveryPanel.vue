<template>
  <div class="collector-discovery">
    <div v-if="isOpcUa" class="collector-discovery__head">
      <div class="collector-discovery__title">
        <strong>OPC UA 设备节点</strong>
        <span v-if="isCascading">正在读取下级节点</span>
        <span v-else-if="selectedCount > 0">已选择 {{ selectedCount }} 个变量</span>
      </div>
      <div class="collector-discovery__toolbar">
        <el-input
          v-model="nodeIdFilter"
          clearable
          placeholder="筛选已加载节点的 NodeId"
          aria-label="NodeId 筛选"
        />
        <el-button
          type="primary"
          :loading="batchLoading"
          :disabled="selectedCount === 0 || isCascading || !enabled"
          @click="emitSelectedPoints"
        >
          批量新增 {{ selectedCount || '' }}
        </el-button>
        <el-button :loading="refreshing" :disabled="!enabled || isCascading" @click="refreshTree">
          <IconTablerRefresh />刷新
        </el-button>
      </div>
    </div>

    <div v-else class="collector-discovery__toolbar is-generic">
      <el-input v-model="parentNodeId" placeholder="父节点 NodeId" />
      <el-button
        data-test="browse-device"
        type="primary"
        :disabled="!enabled"
        :loading="loading"
        @click="browseGeneric"
      >
        浏览设备
      </el-button>
      <el-button :loading="batchLoading" :disabled="!selected.length" @click="emitPoints">
        保存为变量
      </el-button>
    </div>

    <el-alert
      v-if="!enabled"
      title="当前未选择兼容 Agent，仍可离线编辑连接和变量"
      type="info"
      :closable="false"
      show-icon
    />
    <el-alert
      v-else-if="browseDiagnostic"
      :title="browseDiagnostic"
      type="warning"
      :closable="false"
      show-icon
    />

    <div v-if="isOpcUa" class="collector-discovery__tree-shell">
      <el-tree
        v-if="enabled"
        :key="treeVersion"
        ref="treeRef"
        lazy
        show-checkbox
        check-strictly
        node-key="nodeId"
        highlight-current
        :load="loadTreeNode"
        :props="treeProps"
        :filter-node-method="filterTreeNode"
        @check="handleTreeCheck"
      >
        <template #default="{ data }">
          <div
            class="collector-discovery__tree-node"
            :class="[`is-${data.nodeClass}`, { 'is-modeled': data.modeled }]"
          >
            <span class="collector-discovery__node-icon">
              <IconTablerVariable v-if="data.nodeClass === 'variable'" />
              <IconTablerFunction v-else-if="data.nodeClass === 'method'" />
              <IconTablerFolder v-else />
            </span>
            <span class="collector-discovery__node-main">
              <strong>{{ data.displayName || data.browseName || data.nodeId }}</strong>
              <small>{{ data.nodeId }}</small>
            </span>
            <span class="collector-discovery__node-meta">
              <IconTablerLoader2
                v-if="cascadingNodeIds.has(data.nodeId)"
                class="collector-discovery__loading-icon"
              />
              <em v-if="data.dataType">{{ data.dataType }}</em>
              <el-tag v-if="data.modeled" size="small" type="info">已添加</el-tag>
              <el-tag
                v-else
                size="small"
                :type="data.nodeClass === 'variable' ? 'success' : 'info'"
              >
                {{ nodeClassLabel(data.nodeClass) }}
              </el-tag>
              <el-button
                v-if="data.nodeClass === 'variable' && !data.modeled"
                class="collector-discovery__node-action"
                text
                circle
                title="配置后新增变量"
                aria-label="配置后新增变量"
                @click.stop="openPointDrawer(data)"
              >
                <IconTablerPlus />
              </el-button>
            </span>
          </div>
        </template>
      </el-tree>
      <el-empty v-else description="选择在线且支持 OPC UA 浏览的调试代理" />
    </div>

    <el-table v-else :data="nodes" height="100%" @selection-change="selected = $event">
      <el-table-column type="selection" width="44" :selectable="isGenericNodeSelectable" />
      <el-table-column prop="displayName" label="显示名" min-width="180" />
      <el-table-column prop="nodeId" label="NodeId" min-width="260" />
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag v-if="row.modeled" size="small" type="info">已添加</el-tag>
          <span v-else>{{ nodeClassLabel(row.nodeClass) }}</span>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { ElMessage, type TreeInstance } from 'element-plus'
import {
  checkCollectorPointAddresses,
  createCollectorTask,
  getCollectorTask,
} from '@/api/collector.api'
import { CollectorBrowseCache, type CollectorBrowsePriority } from './collector-browse-cache'
import {
  buildCollectorPointCreateDefaults,
  collectCollectorBrowseVariables,
  filterCollectorBrowseNode,
  type CollectorBrowseNode,
  type CollectorPointCreateDefaults,
} from './collector-workbench-model'
import IconTablerFolder from '~icons/tabler/folder'
import IconTablerFunction from '~icons/tabler/function'
import IconTablerLoader2 from '~icons/tabler/loader-2'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerVariable from '~icons/tabler/variable'

const opcuaObjectsNodeId = 'ns=0;i=85'
const browseBatchSize = 32
const addressCheckBatchSize = 500
const cascadeConcurrency = 3
const props = defineProps<{
  projectId: string
  connectionId: string
  protocolFamily: string
  agentId?: string
  workspaceSessionId: string
  enabled: boolean
  batchLoading?: boolean
}>()
const emit = defineEmits<{
  points: [points: Record<string, unknown>[]]
  createPoint: [defaults: CollectorPointCreateDefaults]
  sessionError: [message: string]
}>()
const isOpcUa = computed(() => props.protocolFamily.trim().toLowerCase() === 'opcua')
const browseContextKey = computed(
  () => `${props.projectId}:${props.connectionId}:${props.agentId || 'no-agent'}`,
)
const browseCache = new CollectorBrowseCache<CollectorBrowseNode>(3)
const treeRef = ref<TreeInstance>()
const treeVersion = ref(0)
const nodeIdFilter = ref('')
const refreshing = ref(false)
const agentCacheRefreshPending = ref(false)
const browseDiagnostic = ref('')
const parentNodeId = ref(opcuaObjectsNodeId)
const nodes = ref<CollectorBrowseNode[]>([])
const selected = ref<CollectorBrowseNode[]>([])
const loading = ref(false)
const existingNodeIds = ref(new Set<string>())
const selectedVariableIds = ref(new Set<string>())
const selectedBranchIds = ref(new Set<string>())
const cascadingNodeIds = ref(new Set<string>())
const knownNodes = new Map<string, CollectorBrowseNode>()
const childIdsByParent = new Map<string, Set<string>>()
const parentIdsByChild = new Map<string, Set<string>>()
const selectedCount = computed(() => selectedVariableIds.value.size)
const isCascading = computed(() => cascadingNodeIds.value.size > 0)
const treeProps = {
  label: 'displayName',
  isLeaf: (data: CollectorBrowseNode) => !data.hasChildren,
  disabled: (data: CollectorBrowseNode) =>
    Boolean(data.modeled) ||
    data.nodeClass === 'method' ||
    (!data.hasChildren && data.nodeClass !== 'variable'),
}

watch(nodeIdFilter, (keyword) => treeRef.value?.filter(keyword))
watch(
  () => [props.connectionId, props.agentId, props.enabled],
  () => {
    resetBrowseState()
    treeVersion.value += 1
  },
)
watch(
  () => props.enabled,
  (enabled, previousEnabled) => {
    if (previousEnabled && !enabled) browseCache.refresh(browseContextKey.value)
  },
)

async function waitTask(projectId: string, taskId: string) {
  for (let index = 0; index < 150; index++) {
    const task = await getCollectorTask(projectId, taskId)
    if (task.status === 'succeeded') return task
    if (['failed', 'cancelled', 'expired'].includes(task.status)) {
      throw new Error(task.errorMessage || '设备浏览失败')
    }
    await new Promise((resolve) => setTimeout(resolve, 200))
  }
  throw new Error('设备浏览超时')
}

function normalizeBrowseNodes(nodesToNormalize: CollectorBrowseNode[] = []) {
  return nodesToNormalize.map((node) => ({
    nodeId: String(node.nodeId || ''),
    browseName: String(node.browseName || ''),
    displayName: String(node.displayName || ''),
    nodeClass: String(node.nodeClass || 'other').toLowerCase(),
    dataType: node.dataType ? String(node.dataType) : null,
    hasChildren: Boolean(node.hasChildren),
    modeled: false,
  }))
}

// 只检查当前浏览批次中的地址，避免为了置灰状态一次性拉取全部变量。
async function findExistingNodeIds(nodesToCheck: CollectorBrowseNode[]) {
  const variables = [
    ...new Map(
      nodesToCheck
        .filter((node) => node.nodeClass === 'variable')
        .map((node) => [node.nodeId, node]),
    ).values(),
  ]
  const result = new Set<string>()
  for (let offset = 0; offset < variables.length; offset += addressCheckBatchSize) {
    const batch = variables.slice(offset, offset + addressCheckBatchSize)
    const indexes = await checkCollectorPointAddresses(
      props.projectId,
      props.connectionId,
      batch.map((node) => buildCollectorPointCreateDefaults(node).address),
    )
    for (const index of indexes) {
      const node = batch[index]
      if (node) result.add(node.nodeId)
    }
  }
  return result
}

async function decorateExistingNodes(nodesToDecorate: CollectorBrowseNode[]) {
  const found = await findExistingNodeIds(nodesToDecorate)
  const merged = new Set(existingNodeIds.value)
  found.forEach((nodeId) => merged.add(nodeId))
  existingNodeIds.value = merged
  for (const node of nodesToDecorate) {
    node.modeled = merged.has(node.nodeId)
    knownNodes.set(node.nodeId, node)
  }
  dropExistingSelections()
  return nodesToDecorate
}

async function refreshExisting() {
  const variables = [...knownNodes.values()].filter((node) => node.nodeClass === 'variable')
  const found = await findExistingNodeIds(variables)
  existingNodeIds.value = found
  for (const node of variables) node.modeled = found.has(node.nodeId)
  selectedBranchIds.value = new Set()
  selected.value = selected.value.filter((node) => !found.has(node.nodeId))
  dropExistingSelections()
  await nextTick()
  syncTreeChecks()
}

function selectedBatchVariables() {
  return selected.value.filter((node) => node.nodeClass === 'variable' && !node.modeled)
}

function markSelectedCreated(createdIndexes?: number[]) {
  const candidates = selectedBatchVariables()
  const createdNodes = createdIndexes
    ? createdIndexes.map((index) => candidates[index]).filter(Boolean)
    : candidates
  const createdIds = new Set(createdNodes.map((node) => node.nodeId))
  const existing = new Set(existingNodeIds.value)
  createdIds.forEach((nodeId) => {
    existing.add(nodeId)
    const node = knownNodes.get(nodeId)
    if (node) node.modeled = true
  })
  existingNodeIds.value = existing
  selected.value = selected.value.filter((node) => !createdIds.has(node.nodeId))
  selectedVariableIds.value = new Set(
    [...selectedVariableIds.value].filter((nodeId) => !createdIds.has(nodeId)),
  )
  selectedBranchIds.value = new Set()
  syncTreeChecks()
}

function rememberChildren(parentId: string, children: CollectorBrowseNode[]) {
  childIdsByParent.set(parentId, new Set(children.map((node) => node.nodeId)))
  for (const child of children) {
    knownNodes.set(child.nodeId, child)
    const parents = new Set(parentIdsByChild.get(child.nodeId) || [])
    parents.add(parentId)
    parentIdsByChild.set(child.nodeId, parents)
  }
}

async function fetchBrowseChildren(
  projectId: string,
  connectionId: string,
  agentId: string,
  parentNodeIdValue: string,
  silent: boolean,
  refreshCache = false,
) {
  const task = await createCollectorTask(projectId, {
    agentId,
    connectionId,
    operation: 'device.browse',
    input: {
      workspaceSessionId: props.workspaceSessionId,
      parentNodeId: parentNodeIdValue,
      maxDepth: 1,
      refreshCache,
    },
  })
  const completed = await waitTask(projectId, task.taskId)
  const result = completed.result as {
    nodes?: CollectorBrowseNode[]
    diagnostics?: Array<{ message?: string } | string>
  } | null
  if (!silent) {
    const diagnostic = result?.diagnostics?.[0]
    browseDiagnostic.value =
      typeof diagnostic === 'string' ? diagnostic : diagnostic?.message?.trim() || ''
  }
  const children = await decorateExistingNodes(normalizeBrowseNodes(result?.nodes))
  rememberChildren(parentNodeIdValue, children)
  return children
}

async function fetchBrowseChildrenBatch(
  projectId: string,
  connectionId: string,
  agentId: string,
  parentNodeIds: string[],
) {
  const task = await createCollectorTask(projectId, {
    agentId,
    connectionId,
    operation: 'device.browse',
    input: {
      workspaceSessionId: props.workspaceSessionId,
      parentNodeIds,
      maxDepth: 1,
    },
  })
  const completed = await waitTask(projectId, task.taskId)
  const result = completed.result as {
    branches?: Array<{ parentNodeId?: string; nodes?: CollectorBrowseNode[] }>
  } | null
  const branches = (result?.branches || []).map((branch) => ({
    parentNodeId: String(branch.parentNodeId || ''),
    nodes: normalizeBrowseNodes(branch.nodes),
  }))
  await decorateExistingNodes(branches.flatMap((branch) => branch.nodes))
  for (const branch of branches) rememberChildren(branch.parentNodeId, branch.nodes)
  return branches
}

function browseChildren(
  parentNodeIdValue: string,
  priority: CollectorBrowsePriority,
  refreshCache = false,
) {
  const agentId = props.agentId
  if (!agentId) return Promise.resolve([])
  const projectId = props.projectId
  const connectionId = props.connectionId
  return browseCache.get(browseContextKey.value, parentNodeIdValue, priority, () =>
    fetchBrowseChildren(
      projectId,
      connectionId,
      agentId,
      parentNodeIdValue,
      priority === 'prefetch',
      refreshCache,
    ),
  )
}

function prefetchNextLevel(nodesToPrefetch: CollectorBrowseNode[]) {
  const agentId = props.agentId
  if (!agentId) return
  const parentNodeIds = [
    ...new Set(nodesToPrefetch.filter((node) => node.hasChildren).map((node) => node.nodeId)),
  ]
  if (parentNodeIds.length === 0) return

  const projectId = props.projectId
  const connectionId = props.connectionId
  const cacheEpoch = browseCache.epoch(browseContextKey.value)
  for (let offset = 0; offset < parentNodeIds.length; offset += browseBatchSize) {
    const batch = parentNodeIds.slice(offset, offset + browseBatchSize)
    void fetchBrowseChildrenBatch(projectId, connectionId, agentId, batch)
      .then((branches) => {
        for (const branch of branches) {
          if (branch.parentNodeId) {
            browseCache.set(browseContextKey.value, branch.parentNodeId, branch.nodes, cacheEpoch)
          }
        }
        void nextTick(syncTreeChecks)
      })
      .catch(() => undefined)
  }
}

async function loadTreeNode(
  node: { level: number; data?: CollectorBrowseNode },
  resolve: (nodes: CollectorBrowseNode[]) => void,
  reject: () => void,
) {
  if (!props.enabled) {
    resolve([])
    return
  }
  const targetNodeId = node.level === 0 ? opcuaObjectsNodeId : node.data?.nodeId
  if (!targetNodeId) {
    resolve([])
    return
  }
  const cachedChildren = browseCache.peek(browseContextKey.value, targetNodeId)
  if (cachedChildren) {
    rememberChildren(targetNodeId, cachedChildren)
    resolve(cachedChildren)
    prefetchNextLevel(cachedChildren)
    refreshing.value = false
    await nextTick()
    syncTreeChecks()
    return
  }
  try {
    const refreshCache = node.level === 0 && agentCacheRefreshPending.value
    const children = await browseChildren(targetNodeId, 'user', refreshCache)
    if (refreshCache) agentCacheRefreshPending.value = false
    resolve(children)
    prefetchNextLevel(children)
    await nextTick()
    syncTreeChecks()
  } catch (error) {
    reject()
    handleBrowseError(error)
  } finally {
    refreshing.value = false
  }
}

function filterTreeNode(keyword: string, node: CollectorBrowseNode) {
  return filterCollectorBrowseNode(keyword, node)
}

async function handleTreeCheck(
  node: CollectorBrowseNode,
  checked: { checkedKeys: Array<string | number> },
) {
  const isChecked = checked.checkedKeys.map(String).includes(node.nodeId)
  if (node.modeled) {
    syncTreeChecks()
    return
  }
  if (node.nodeClass === 'variable') {
    const next = new Set(selectedVariableIds.value)
    if (isChecked) next.add(node.nodeId)
    else {
      next.delete(node.nodeId)
      clearSelectedAncestors(node.nodeId)
    }
    selectedVariableIds.value = next
    syncTreeChecks()
    return
  }
  if (!node.hasChildren) {
    syncTreeChecks()
    return
  }
  if (isChecked) await selectBranch(node)
  else removeBranchSelection(node.nodeId)
}

async function selectBranch(node: CollectorBrowseNode) {
  if (cascadingNodeIds.value.has(node.nodeId)) return
  cascadingNodeIds.value = new Set(cascadingNodeIds.value).add(node.nodeId)
  try {
    const result = await collectCollectorBrowseVariables(
      node,
      (nodeId) => browseChildren(nodeId, 'user'),
      cascadeConcurrency,
    )
    for (const branch of result.branches) rememberChildren(branch.parentNodeId, branch.nodes)
    const variableIds = new Set(selectedVariableIds.value)
    result.variables.forEach((variable) => variableIds.add(variable.nodeId))
    selectedVariableIds.value = variableIds
    const branches = new Set(selectedBranchIds.value)
    if (result.variables.length > 0) branches.add(node.nodeId)
    else ElMessage.info('该节点下没有可新增的变量')
    selectedBranchIds.value = branches
  } catch (error) {
    handleBrowseError(error)
  } finally {
    const next = new Set(cascadingNodeIds.value)
    next.delete(node.nodeId)
    cascadingNodeIds.value = next
    syncTreeChecks()
  }
}

function removeBranchSelection(nodeId: string) {
  const variableIds = new Set(selectedVariableIds.value)
  const branchIds = new Set(selectedBranchIds.value)
  const pending = [nodeId]
  const visited = new Set<string>()
  while (pending.length > 0) {
    const current = pending.pop()
    if (!current || visited.has(current)) continue
    visited.add(current)
    variableIds.delete(current)
    branchIds.delete(current)
    for (const childId of childIdsByParent.get(current) || []) pending.push(childId)
  }
  selectedVariableIds.value = variableIds
  selectedBranchIds.value = branchIds
  clearSelectedAncestors(nodeId)
  syncTreeChecks()
}

function clearSelectedAncestors(nodeId: string) {
  const branches = new Set(selectedBranchIds.value)
  const pending = [...(parentIdsByChild.get(nodeId) || [])]
  const visited = new Set<string>()
  while (pending.length > 0) {
    const parentId = pending.pop()
    if (!parentId || visited.has(parentId)) continue
    visited.add(parentId)
    branches.delete(parentId)
    pending.push(...(parentIdsByChild.get(parentId) || []))
  }
  selectedBranchIds.value = branches
}

function dropExistingSelections() {
  if (existingNodeIds.value.size === 0) return
  const selectedIds = new Set(selectedVariableIds.value)
  existingNodeIds.value.forEach((nodeId) => selectedIds.delete(nodeId))
  selectedVariableIds.value = selectedIds
}

function syncTreeChecks() {
  treeRef.value?.setCheckedKeys(
    [...existingNodeIds.value, ...selectedVariableIds.value, ...selectedBranchIds.value],
    false,
  )
}

function emitSelectedPoints() {
  const variables = [...selectedVariableIds.value]
    .map((nodeId) => knownNodes.get(nodeId))
    .filter((node): node is CollectorBrowseNode => Boolean(node) && !node.modeled)
  emit('points', buildBatchPoints(variables))
}

function openPointDrawer(node: CollectorBrowseNode) {
  if (node.modeled) return
  emit('createPoint', buildCollectorPointCreateDefaults(node))
}

function refreshTree() {
  refreshing.value = true
  resetBrowseState()
  browseCache.refresh(browseContextKey.value)
  agentCacheRefreshPending.value = true
  treeVersion.value += 1
  nextTick(() => {
    if (!props.enabled) refreshing.value = false
  })
}

function resetBrowseState() {
  nodeIdFilter.value = ''
  browseDiagnostic.value = ''
  agentCacheRefreshPending.value = false
  existingNodeIds.value = new Set()
  selectedVariableIds.value = new Set()
  selectedBranchIds.value = new Set()
  cascadingNodeIds.value = new Set()
  selected.value = []
  knownNodes.clear()
  childIdsByParent.clear()
  parentIdsByChild.clear()
}

function nodeClassLabel(nodeClass: string) {
  if (nodeClass === 'variable') return '变量'
  if (nodeClass === 'method') return '方法'
  if (nodeClass === 'view') return '视图'
  if (nodeClass === 'object') return '对象'
  return '节点'
}

async function browseGeneric() {
  loading.value = true
  try {
    nodes.value = await browseChildren(parentNodeId.value.trim() || opcuaObjectsNodeId, 'user')
  } catch (error) {
    handleBrowseError(error)
  } finally {
    loading.value = false
  }
}

function isGenericNodeSelectable(node: CollectorBrowseNode) {
  return node.nodeClass === 'variable' && !node.modeled
}

function emitPoints() {
  emit('points', buildBatchPoints(selectedBatchVariables()))
}

function buildBatchPoints(variables: CollectorBrowseNode[]) {
  return variables.map((node, index) => ({
    groupId: null,
    ...buildCollectorPointCreateDefaults(node),
    description: null,
    readOptions: {},
    acquisition: { mode: 'polling', intervalMs: 1000 },
    sortOrder: index,
    metadata: {},
  }))
}

function handleBrowseError(error: unknown) {
  const message = error instanceof Error ? error.message : '设备浏览失败'
  if (/会话.*(断开|建立)|连接.*断开/.test(message)) emit('sessionError', message)
  ElMessage.error(message)
}

defineExpose({ markSelectedCreated, refreshExisting })
</script>

<style scoped>
.collector-discovery {
  display: flex;
  height: 100%;
  min-height: 0;
  flex-direction: column;
  gap: 12px;
}
.collector-discovery__head,
.collector-discovery__toolbar,
.collector-discovery__tree-node,
.collector-discovery__node-meta,
.collector-discovery__title {
  display: flex;
  align-items: center;
}
.collector-discovery__head {
  min-height: 40px;
  justify-content: space-between;
  gap: 16px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--dc-border);
}
.collector-discovery__title {
  flex: 0 0 auto;
  gap: 10px;
}
.collector-discovery__title strong {
  color: var(--dc-text);
  font-size: 14px;
}
.collector-discovery__title span {
  color: var(--dc-text-muted);
  font-size: 12px;
}
.collector-discovery__toolbar {
  width: min(720px, 68%);
  justify-content: flex-end;
  gap: 10px;
}
.collector-discovery__toolbar .el-input {
  min-width: 220px;
  flex: 1;
}
.collector-discovery__toolbar .el-button {
  flex: 0 0 auto;
}
.collector-discovery__toolbar.is-generic {
  width: 100%;
  display: grid;
  grid-template-columns: 1fr auto auto;
}
.collector-discovery__tree-shell {
  min-height: 0;
  flex: 1;
  overflow: auto;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-lg);
  background: var(--dc-surface-raised);
}
.collector-discovery__tree-shell :deep(.el-tree) {
  min-width: 760px;
  padding: 8px 10px 18px;
  background: transparent;
}
.collector-discovery__tree-shell :deep(.el-tree-node__content) {
  min-height: 46px;
  border-radius: 8px;
}
.collector-discovery__tree-shell :deep(.el-tree-node__content:hover),
.collector-discovery__tree-shell :deep(.el-tree-node.is-current > .el-tree-node__content) {
  background: var(--dc-primary-soft);
}
.collector-discovery__tree-node {
  min-width: 0;
  flex: 1;
  gap: 10px;
  padding-right: 8px;
}
.collector-discovery__tree-node.is-modeled .collector-discovery__node-main {
  opacity: 0.58;
}
.collector-discovery__node-icon {
  display: grid;
  width: 28px;
  height: 28px;
  flex: 0 0 28px;
  place-items: center;
  border: 1px solid var(--dc-border);
  border-radius: 7px;
  color: var(--dc-text-secondary);
  background: var(--dc-surface);
}
.collector-discovery__tree-node.is-variable:not(.is-modeled) .collector-discovery__node-icon {
  border-color: color-mix(in srgb, var(--el-color-success) 28%, var(--dc-border));
  color: var(--el-color-success);
  background: color-mix(in srgb, var(--el-color-success) 8%, var(--dc-surface));
}
.collector-discovery__node-main {
  display: grid;
  min-width: 0;
  flex: 1;
  gap: 2px;
}
.collector-discovery__node-main strong,
.collector-discovery__node-main small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.collector-discovery__node-main strong {
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 600;
}
.collector-discovery__node-main small {
  color: var(--dc-text-muted);
  font-family: Consolas, 'Courier New', monospace;
  font-size: 11px;
}
.collector-discovery__node-meta {
  flex: 0 0 auto;
  gap: 8px;
}
.collector-discovery__node-meta em {
  color: var(--dc-text-secondary);
  font-size: 11px;
  font-style: normal;
}
.collector-discovery__node-action {
  width: 28px;
  height: 28px;
  color: var(--dc-primary);
}
.collector-discovery__loading-icon {
  color: var(--dc-primary);
  animation: collector-discovery-spin 0.8s linear infinite;
}
.collector-discovery :deep(.el-table) {
  flex: 1;
}
@keyframes collector-discovery-spin {
  to {
    transform: rotate(360deg);
  }
}
@media (max-width: 900px) {
  .collector-discovery__head {
    align-items: stretch;
    flex-direction: column;
  }
  .collector-discovery__toolbar {
    width: 100%;
  }
}
</style>
