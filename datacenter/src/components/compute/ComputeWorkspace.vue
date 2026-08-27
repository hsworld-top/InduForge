<template>
  <div class="compute-workspace">
    <ComputeTree
      :units="computeStore.list"
      :folders="computeStore.folders"
      :selected-unit-id="selectedComputeFolderId ? null : selectedUnitId"
      :dirty-unit-ids="dirtyUnitIds"
      :loading="computeStore.loading || computeStore.foldersLoading"
      :list-error="computeStore.listError"
      :folders-error="computeStore.foldersError"
      :loading-folder-ids="loadingFolderIds"
      :folder-has-more-ids="folderHasMoreIds"
      :root-folders-has-more="computeStore.hasMoreFolders(null)"
      :root-folders-loading="computeStore.isFolderLoading(null)"
      :selected-folder-id="selectedComputeFolderId"
      @select-unit="selectUnit"
      @create-unit="showCreateUnitDialog = true"
      @create-folder="showCreateFolderDialog = true"
      @manage-dependencies="openDependencyManager"
      @refresh="loadWorkspace"
      @search="handleTreeSearch"
      @load-folder="loadFolderChildren"
      @load-more-folder="loadMoreFolderChildren"
      @select-folder="selectComputeFolder"
      @rename-unit="openRenameUnitDialog"
      @move-unit="openMoveUnitDialog"
      @delete-unit="handleDeleteUnitFromTree"
      @rename-folder="openRenameFolderDialog"
      @move-folder="openMoveFolderDialog"
      @delete-folder="handleDeleteFolder"
    />

    <ComputeEditorShell
      :project-id="String(projectId)"
      :tabs="tabs"
      :active-id="activeTabId"
      :active-draft="activeDraft"
      :loading="computeStore.detailLoading"
      :saving="computeStore.saving"
      :deleting="computeStore.deleting"
      :error="computeStore.detailError"
      :dependencies="computeStore.dependencies"
      :dependencies-loading="computeStore.dependenciesLoading"
      :dependencies-error="computeStore.dependenciesError"
      :capabilities="computeCapabilities"
      :capabilities-loading="capabilitiesLoading"
      @activate-tab="activateTab"
      @close-tab="closeTab"
      @save="saveTab"
      @toggle-enabled="toggleEnabled"
      @delete-unit="deleteUnit"
      @mark-dirty="markDirty"
      @refresh-dependencies="loadDependencies"
      @manage-dependencies="openDependencyManager"
      @retry-capabilities="retryComputeCapabilities"
    />

    <ComputeDependencyManager
      v-model="showDependencyManagerDialog"
      :project-id="String(projectId)"
      @changed="loadDependencies"
    />

    <CreateComputeUnitDialog
      ref="createUnitDialogRef"
      v-model="showCreateUnitDialog"
      :folders="computeStore.folders"
      :initial-folder-id="selectedComputeFolderId"
      :loading="computeStore.creating"
      :error="computeStore.createError"
      @submit="handleCreateUnit"
    />

    <CreateComputeFolderDialog
      ref="createFolderDialogRef"
      v-model="showCreateFolderDialog"
      :folders="computeStore.folders"
      :initial-parent-id="selectedComputeFolderId"
      :loading="computeStore.creating"
      :error="computeStore.createError"
      @submit="handleCreateFolder"
    />

    <RenameComputeUnitDialog
      v-model="showRenameUnitDialog"
      :unit="contextUnit"
      :loading="computeStore.saving"
      @submit="handleRenameUnit"
    />

    <MoveComputeUnitDialog
      v-model="showMoveUnitDialog"
      :unit="contextUnit"
      :folders="computeStore.folders"
      :loading="computeStore.saving"
      @submit="handleMoveUnit"
    />

    <RenameComputeFolderDialog
      v-model="showRenameFolderDialog"
      :folder="contextFolder"
      :loading="computeStore.saving"
      @submit="handleRenameFolder"
    />

    <MoveComputeFolderDialog
      v-model="showMoveFolderDialog"
      :folder="contextFolder"
      :folders="computeStore.folders"
      :loading="computeStore.saving"
      @submit="handleMoveFolder"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, inject, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import type {
  ComputeUnit,
  ComputeFolderSave,
  ComputeUnitSave,
  ComputeCapabilities,
} from '@/api/schemas/compute.schema'
import { getComputeCapabilities, getComputeFolders, getComputeUnits } from '@/api/compute.api'
import { useComputeStore } from '@/stores/compute.store'
import { getApiErrorMessage } from '@/utils/request'
import ComputeEditorShell from './ComputeEditorShell.vue'
import ComputeDependencyManager from './ComputeDependencyManager.vue'
import ComputeTree from './ComputeTree.vue'
import CreateComputeFolderDialog from './CreateComputeFolderDialog.vue'
import CreateComputeUnitDialog from './CreateComputeUnitDialog.vue'
import MoveComputeUnitDialog from './MoveComputeUnitDialog.vue'
import MoveComputeFolderDialog from './MoveComputeFolderDialog.vue'
import RenameComputeUnitDialog from './RenameComputeUnitDialog.vue'
import RenameComputeFolderDialog from './RenameComputeFolderDialog.vue'
import {
  draftToSavePayload,
  toComputeDraft,
  type ComputeDraft,
  type ComputeEditorTab,
} from './computeEditorModel'
import type { ComputeFolderTreeNode } from './computeTreeModel'

const props = defineProps<{
  projectId: string
  selectedUnitId?: string | null
}>()

const route = useRoute()
const router = useRouter()
const computeStore = useComputeStore()
type DraftGuard = {
  isDirty: () => boolean
  save?: () => Promise<boolean>
  discard?: () => void
}
const registerDraftChecker =
  inject<(guard: DraftGuard | (() => boolean)) => () => void>('registerDraftChecker')
let unregisterDraftChecker: (() => void) | undefined

const showCreateUnitDialog = ref(false)
const showCreateFolderDialog = ref(false)
const createUnitDialogRef = ref<InstanceType<typeof CreateComputeUnitDialog> | null>(null)
const createFolderDialogRef = ref<InstanceType<typeof CreateComputeFolderDialog> | null>(null)
const showRenameUnitDialog = ref(false)
const showMoveUnitDialog = ref(false)
const showRenameFolderDialog = ref(false)
const showMoveFolderDialog = ref(false)
const showDependencyManagerDialog = ref(false)
const contextUnit = ref<ComputeUnit | null>(null)
const contextFolder = ref<ComputeFolderTreeNode | null>(null)
const drafts = ref<Record<string, ComputeDraft>>({})
const activeTabId = ref<string | null>(null)
const computeCapabilities = ref<ComputeCapabilities | null>(null)
const capabilitiesLoading = ref(false)
const capabilityRetryDelays = [3_000, 10_000, 30_000]
let capabilityRetryAttempt = 0
let capabilityRetryTimer: ReturnType<typeof window.setTimeout> | null = null
let capabilityLoadVersion = 0

const selectedUnitId = computed(() => props.selectedUnitId || null)

const computeBasePath = computed(() => {
  const prefix = route.path.startsWith('/debug/') ? '/debug' : ''
  return `${prefix}/compute`
})

const tabs = computed<ComputeEditorTab[]>(() =>
  Object.values(drafts.value).map((draft) => ({
    id: draft.id,
    name: draft.name,
    dirty: draft.dirty,
  })),
)

const activeDraft = computed(() => {
  if (!activeTabId.value) return null
  return drafts.value[activeTabId.value] || null
})

const hasDirtyTabs = computed(() => Object.values(drafts.value).some((draft) => draft.dirty))
const dirtyUnitIds = computed(() =>
  Object.values(drafts.value)
    .filter((draft) => draft.dirty)
    .map((draft) => draft.id),
)
const computeSearch = ref('')
const selectedComputeFolderId = ref<string | null>(null)
const loadingFolderIds = computed(() =>
  computeStore.folderLoadingKeys.filter((key) => key !== 'root' && !key.startsWith('search:')),
)
const folderHasMoreIds = computed(() =>
  Object.entries(computeStore.folderPagination)
    .filter(
      ([key, value]) =>
        key !== 'root' && !key.startsWith('search:') && value.page < value.totalPages,
    )
    .map(([key]) => key),
)

const computeListParams = (page = 1) => ({
  page,
  pageSize: 100,
  search: computeSearch.value || undefined,
})
let computeListLoadVersion = 0

// 树区域不暴露分页：接口仍按页读取，前端顺序合并全部计算单元，避免大列表单次请求失控。
async function loadAllComputeUnits() {
  const version = ++computeListLoadVersion
  const projectId = String(props.projectId)
  const search = computeSearch.value || undefined
  const params = (page: number) => ({ page, pageSize: 100, search })
  await computeStore.fetchList(projectId, params(1))
  if (version !== computeListLoadVersion) return
  const totalPages = Math.ceil(computeStore.total / 100)
  for (let page = 2; page <= totalPages; page += 1) {
    await computeStore.fetchList(projectId, params(page), {
      append: true,
      silent: true,
    })
    if (version !== computeListLoadVersion) return
  }
}

async function loadWorkspace() {
  const projectId = String(props.projectId)
  const results = await Promise.allSettled([
    loadAllComputeUnits(),
    computeStore.fetchFolders(projectId),
    loadComputeCapabilities(),
  ])
  const failed = results.find((result) => result.status === 'rejected')
  if (failed) {
    ElMessage.warning('计算单元部分能力未启用')
  }
}

function clearCapabilityRetry() {
  if (capabilityRetryTimer) {
    window.clearTimeout(capabilityRetryTimer)
    capabilityRetryTimer = null
  }
}

function scheduleCapabilityRetry() {
  clearCapabilityRetry()
  const delay =
    capabilityRetryDelays[Math.min(capabilityRetryAttempt, capabilityRetryDelays.length - 1)]
  capabilityRetryAttempt += 1
  capabilityRetryTimer = window.setTimeout(() => {
    capabilityRetryTimer = null
    void loadComputeCapabilities().catch(() => undefined)
  }, delay)
}

// 沙箱可能晚于数据中心启动；不可用时退避重试，恢复后立即停止探测，避免页面永久保留旧状态。
async function loadComputeCapabilities() {
  if (capabilitiesLoading.value) return
  clearCapabilityRetry()
  const version = ++capabilityLoadVersion
  const projectId = String(props.projectId)
  let shouldRetry = false
  capabilitiesLoading.value = true
  try {
    const value = await getComputeCapabilities(projectId)
    if (version !== capabilityLoadVersion || projectId !== String(props.projectId)) return
    computeCapabilities.value = value
    if (value.sandboxStatus === 'available') {
      capabilityRetryAttempt = 0
    } else {
      shouldRetry = true
    }
  } catch (error) {
    if (version === capabilityLoadVersion && projectId === String(props.projectId)) {
      shouldRetry = true
    }
    throw error
  } finally {
    if (version === capabilityLoadVersion) {
      capabilitiesLoading.value = false
      if (shouldRetry) scheduleCapabilityRetry()
    }
  }
}

function retryComputeCapabilities() {
  capabilityRetryAttempt = 0
  void loadComputeCapabilities().catch((error) => {
    ElMessage.error(getApiErrorMessage(error, '重新检测计算沙箱失败'))
  })
}

function retryCapabilitiesWhenVisible() {
  if (
    document.visibilityState === 'visible' &&
    computeCapabilities.value?.sandboxStatus !== 'available'
  ) {
    retryComputeCapabilities()
  }
}

function handleTreeSearch(keyword: string) {
  computeSearch.value = keyword
  void Promise.allSettled([
    loadAllComputeUnits(),
    computeStore.fetchFolders(String(props.projectId), keyword ? { search: keyword } : {}),
  ])
}

function selectComputeFolder(folderId: string | null) {
  selectedComputeFolderId.value = folderId
}

function loadFolderChildren(folderId: string) {
  void computeStore
    .fetchFolders(String(props.projectId), { parentId: folderId, page: 1 })
    .catch(() => undefined)
}

function loadMoreFolderChildren(folderId: string | null) {
  const key = folderId || 'root'
  const pageInfo = computeStore.folderPagination[key]
  if (!pageInfo || pageInfo.page >= pageInfo.totalPages) return
  void computeStore
    .fetchFolders(String(props.projectId), {
      parentId: folderId,
      page: pageInfo.page + 1,
      append: true,
    })
    .catch(() => undefined)
}

function loadDependencies() {
  computeStore.fetchDependencies(String(props.projectId)).catch(() => undefined)
}

function selectUnit(id: string) {
  selectedComputeFolderId.value = null
  void router.push({
    path: `${computeBasePath.value}/${id}`,
    query: route.query,
  })
}

async function openSelectedUnit(id: string | null) {
  if (!id) {
    activeTabId.value = null
    computeStore.closeEdit()
    return
  }
  if (drafts.value[id]) {
    activeTabId.value = id
    return
  }
  try {
    const unit = await computeStore.openForEdit(String(props.projectId), id)
    if (selectedUnitId.value !== id) {
      return
    }
    drafts.value = {
      ...drafts.value,
      [id]: toComputeDraft(unit),
    }
    activeTabId.value = id
  } catch {
    // 错误文案已写入 store，由编辑区展示。
  }
}

function activateTab(id: string) {
  activeTabId.value = id
  loadDependencies()
  selectUnit(id)
}

function openDependencyManager() {
  showDependencyManagerDialog.value = true
}

async function closeTab(id: string) {
  const draft = drafts.value[id]
  if (!draft) return
  if (draft.dirty) {
    const action = await confirmDirtyClose(draft.name)
    if (action === 'cancel') return
    if (action === 'save') {
      const saved = await saveTab(id)
      if (!saved) return
    }
  }
  const nextDrafts = { ...drafts.value }
  delete nextDrafts[id]
  drafts.value = nextDrafts
  if (activeTabId.value === id) {
    const next = Object.keys(nextDrafts)[0] || null
    activeTabId.value = next
    if (next) {
      selectUnit(next)
    } else {
      void router.push({ path: computeBasePath.value, query: route.query })
    }
  }
}

function markDirty(id: string) {
  const draft = drafts.value[id]
  if (!draft) return
  draft.dirty = true
}

async function saveTab(id: string): Promise<boolean> {
  const draft = drafts.value[id]
  if (!draft) return false
  try {
    const unit = await computeStore.saveUnit(String(props.projectId), id, draftToSavePayload(draft))
    drafts.value = {
      ...drafts.value,
      [id]: toComputeDraft(unit),
    }
    ElMessage.success('计算单元已保存')
    await refreshComputeTree()
    return true
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '保存计算单元失败'))
    return false
  }
}

function refreshComputeTree() {
  const projectId = String(props.projectId)
  return Promise.allSettled([loadAllComputeUnits(), computeStore.fetchFolders(projectId)])
}

async function toggleEnabled(id: string, enabled: boolean) {
  const draft = drafts.value[id]
  let discardDraft = false
  if (draft?.dirty) {
    const action = await confirmDirtyToggle(draft.name, enabled)
    if (action === 'cancel') return
    if (action === 'save') {
      const saved = await saveTab(id)
      if (!saved) return
    } else {
      discardDraft = true
    }
  }
  try {
    const unit = await computeStore.setUnitEnabled(String(props.projectId), id, enabled)
    const currentDraft = drafts.value[id]
    drafts.value = {
      ...drafts.value,
      [id]:
        discardDraft || !currentDraft
          ? toComputeDraft(unit)
          : {
              ...currentDraft,
              isEnabled: unit.isEnabled !== false,
              status: String(unit.status || (unit.isEnabled === false ? 'disabled' : 'enabled')),
            },
    }
    ElMessage.success(enabled ? '计算单元已启用' : '计算单元已停用')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '更新计算单元状态失败'))
  }
}

async function confirmDirtyToggle(name: string, enabled: boolean) {
  try {
    await ElMessageBox.confirm(
      `计算单元「${name}」有未保存修改，${enabled ? '启用' : '停用'}前是否先保存？`,
      '切换计算单元状态',
      {
        confirmButtonText: '保存后切换',
        cancelButtonText: '放弃修改并切换',
        distinguishCancelAndClose: true,
        closeOnClickModal: false,
        type: 'warning',
      },
    )
    return 'save' as const
  } catch (action) {
    if (action === 'cancel') return 'discard' as const
    return 'cancel' as const
  }
}

async function deleteUnit(id: string) {
  const draft = drafts.value[id]
  await confirmAndDeleteUnit(id, draft?.name)
}

async function handleDeleteUnitFromTree(unit: ComputeUnit) {
  await confirmAndDeleteUnit(String(unit.id), unit.name)
}

async function confirmAndDeleteUnit(id: string, name?: string) {
  const ok = await ElMessageBox.confirm(
    `确认删除计算单元「${name || id}」？此操作不可恢复。`,
    '删除计算单元',
    {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
    },
  )
    .then(() => true)
    .catch(() => false)
  if (!ok) return
  try {
    await computeStore.removeUnit(String(props.projectId), id)
    const nextDrafts = { ...drafts.value }
    delete nextDrafts[id]
    drafts.value = nextDrafts
    const next = Object.keys(nextDrafts)[0] || null
    activeTabId.value = next
    if (next) {
      selectUnit(next)
    } else {
      void router.push({ path: computeBasePath.value, query: route.query })
    }
    ElMessage.success('计算单元已删除')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '删除计算单元失败'))
  }
}

async function handleDeleteFolder(folder: ComputeFolderTreeNode) {
  let impact: { childFolderCount: number; unitIds: string[] }
  try {
    impact = await loadComputeFolderDeleteImpact(folder.id)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '读取分组删除影响失败'))
    return
  }
  const childFolderCount = impact.childFolderCount
  const unitCount = impact.unitIds.length
  const detail =
    childFolderCount > 0 || unitCount > 0
      ? `该分组包含 ${childFolderCount} 个子分组、${unitCount} 个计算单元。确认后会一起删除。`
      : '该分组为空。确认后会删除该分组。'
  const ok = await ElMessageBox.confirm(
    `确认删除分组「${folder.name}」？${detail}此操作不可恢复。`,
    '删除分组',
    {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
    },
  )
    .then(() => true)
    .catch(() => false)
  if (!ok) return
  try {
    const removedIds = impact.unitIds
    await computeStore.removeFolder(String(props.projectId), folder.id, computeListParams(1))
    if (selectedComputeFolderId.value === folder.id) selectedComputeFolderId.value = null
    await loadAllComputeUnits()
    if (removedIds.length) {
      const removed = new Set(removedIds)
      const nextDrafts = { ...drafts.value }
      for (const id of removed) {
        delete nextDrafts[id]
      }
      drafts.value = nextDrafts
      if (activeTabId.value && removed.has(activeTabId.value)) {
        const next = Object.keys(nextDrafts)[0] || null
        activeTabId.value = next
        if (next) {
          selectUnit(next)
        } else {
          void router.push({ path: computeBasePath.value, query: route.query })
        }
      }
    }
    ElMessage.success('分组已删除')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '删除分组失败'))
  }
}

// 删除确认前按接口分页展开完整子树，避免懒加载目录导致影响数量被低估。
async function loadComputeFolderDeleteImpact(rootFolderId: string) {
  const pending = [rootFolderId]
  const unitIds: string[] = []
  let childFolderCount = 0
  while (pending.length) {
    const folderId = pending.shift()!
    let loadedUnitCount = 0
    for (let page = 1; ; page += 1) {
      const result = await getComputeUnits(String(props.projectId), {
        folderId,
        page,
        pageSize: 100,
      })
      unitIds.push(...result.list.map((unit) => String(unit.id)))
      loadedUnitCount += result.list.length
      const total = result.pagination?.total
      if (result.list.length < 100 || (typeof total === 'number' && loadedUnitCount >= total)) break
    }
    for (let page = 1; ; page += 1) {
      const result = await getComputeFolders(String(props.projectId), {
        parentId: folderId,
        page,
        pageSize: 100,
      })
      childFolderCount += result.list.length
      pending.push(...result.list.map((item) => String(item.id)))
      if (page >= result.pagination.totalPages) break
    }
  }
  return { childFolderCount, unitIds }
}

async function confirmDirtyClose(name: string) {
  try {
    await ElMessageBox.confirm(`计算单元「${name}」有未保存修改。`, '关闭标签', {
      confirmButtonText: '保存',
      cancelButtonText: '丢弃',
      distinguishCancelAndClose: true,
      type: 'warning',
      closeOnClickModal: false,
    })
    return 'save' as const
  } catch (action) {
    if (action === 'cancel') return 'discard' as const
    return 'cancel' as const
  }
}

function handleBeforeUnload(event: BeforeUnloadEvent) {
  if (!hasDirtyTabs.value) return
  event.preventDefault()
  event.returnValue = ''
}

async function saveDirtyTabs() {
  for (const draft of Object.values(drafts.value)) {
    if (draft.dirty && !(await saveTab(draft.id))) return false
  }
  return true
}

function discardDirtyTabs() {
  drafts.value = Object.fromEntries(
    Object.entries(drafts.value).map(([id, draft]) => [id, { ...draft, dirty: false }]),
  )
}

async function handleCreateUnit(data: ComputeUnitSave) {
  try {
    const unit = await computeStore.createUnit(String(props.projectId), data)
    const draft = toComputeDraft(unit)
    const savedUnit = await computeStore.saveUnit(
      String(props.projectId),
      String(unit.id),
      draftToSavePayload(draft),
    )
    createUnitDialogRef.value?.closeSilently()
    drafts.value = {
      ...drafts.value,
      [String(savedUnit.id)]: toComputeDraft(savedUnit),
    }
    await refreshComputeTree()
    selectUnit(String(savedUnit.id))
    ElMessage.success('计算单元已创建')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '新建计算单元失败'))
  }
}

function openRenameUnitDialog(unit: ComputeUnit) {
  contextUnit.value = unit
  showRenameUnitDialog.value = true
}

function openMoveUnitDialog(unit: ComputeUnit) {
  contextUnit.value = unit
  showMoveUnitDialog.value = true
}

function openRenameFolderDialog(folder: ComputeFolderTreeNode) {
  contextFolder.value = folder
  showRenameFolderDialog.value = true
}

function openMoveFolderDialog(folder: ComputeFolderTreeNode) {
  contextFolder.value = folder
  showMoveFolderDialog.value = true
}

function patchDraftFromSavedUnit(unit: ComputeUnit) {
  const id = String(unit.id)
  const draft = drafts.value[id]
  if (!draft) return
  draft.name = unit.name
  draft.folderId = unit.folderId ? String(unit.folderId) : null
}

async function handleRenameUnit(name: string) {
  const unit = contextUnit.value
  if (!unit) return
  try {
    const saved = await computeStore.saveUnit(String(props.projectId), String(unit.id), {
      name,
    })
    patchDraftFromSavedUnit(saved)
    showRenameUnitDialog.value = false
    contextUnit.value = saved
    await refreshComputeTree()
    ElMessage.success('计算单元已重命名')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '重命名计算单元失败'))
  }
}

async function handleMoveUnit(folderId: string | null) {
  const unit = contextUnit.value
  if (!unit) return
  try {
    const saved = await computeStore.saveUnit(String(props.projectId), String(unit.id), {
      folderId,
    })
    patchDraftFromSavedUnit(saved)
    showMoveUnitDialog.value = false
    contextUnit.value = saved
    await refreshComputeTree()
    ElMessage.success('计算单元已移动')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '移动计算单元失败'))
  }
}

async function handleRenameFolder(name: string) {
  const folder = contextFolder.value
  if (!folder) return
  try {
    const saved = await computeStore.saveFolder(String(props.projectId), folder.id, {
      name,
    })
    showRenameFolderDialog.value = false
    contextFolder.value = {
      ...folder,
      name: saved.name,
      parentId: saved.parentId ? String(saved.parentId) : null,
    }
    await refreshComputeTree()
    ElMessage.success('分组已重命名')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '重命名分组失败'))
  }
}

async function handleMoveFolder(parentId: string | null) {
  const folder = contextFolder.value
  if (!folder) return
  try {
    const saved = await computeStore.saveFolder(String(props.projectId), folder.id, {
      parentId,
    })
    showMoveFolderDialog.value = false
    contextFolder.value = {
      ...folder,
      parentId: saved.parentId ? String(saved.parentId) : null,
    }
    await refreshComputeTree()
    ElMessage.success('分组已移动')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '移动分组失败'))
  }
}

async function handleCreateFolder(data: ComputeFolderSave) {
  try {
    await computeStore.createFolder(String(props.projectId), data)
    createFolderDialogRef.value?.closeSilently()
    ElMessage.success('文件夹已创建')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '新建文件夹失败'))
  }
}

watch(
  () => props.projectId,
  () => {
    clearCapabilityRetry()
    capabilityLoadVersion += 1
    capabilityRetryAttempt = 0
    computeCapabilities.value = null
    drafts.value = {}
    activeTabId.value = null
    computeSearch.value = ''
    void loadWorkspace()
    loadDependencies()
  },
)

watch(
  selectedUnitId,
  (id) => {
    void openSelectedUnit(id)
  },
  { immediate: true },
)

onMounted(() => {
  void loadWorkspace()
  loadDependencies()
  unregisterDraftChecker = registerDraftChecker?.({
    isDirty: () => hasDirtyTabs.value,
    save: saveDirtyTabs,
    discard: discardDirtyTabs,
  })
  window.addEventListener('beforeunload', handleBeforeUnload)
  window.addEventListener('focus', retryCapabilitiesWhenVisible)
  document.addEventListener('visibilitychange', retryCapabilitiesWhenVisible)
})

onBeforeUnmount(() => {
  clearCapabilityRetry()
  capabilityLoadVersion += 1
  unregisterDraftChecker?.()
  window.removeEventListener('beforeunload', handleBeforeUnload)
  window.removeEventListener('focus', retryCapabilitiesWhenVisible)
  document.removeEventListener('visibilitychange', retryCapabilitiesWhenVisible)
})
</script>

<style scoped>
.compute-workspace {
  height: 100%;
  min-height: 0;
  display: grid;
  grid-template-columns: 260px minmax(0, 1fr);
  overflow: hidden;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
}

@media (max-width: 900px) {
  .compute-workspace {
    grid-template-columns: 1fr;
    grid-template-rows: minmax(260px, 42vh) minmax(0, 1fr);
  }
}
</style>
