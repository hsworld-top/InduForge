<!--
  AssetManagerDialog - 通用资源管理选择弹窗
  用于属性面板等轻量场景中上传、浏览并选择工程资源。
-->
<script setup lang="ts">
import type { AssetFolder, AssetItem } from '@/types/api'
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import IconEpUpload from '~icons/ep/upload'
import assetApi from '@/services/assetApi'
import { unwrapApiData } from '@/types/api'
import { normalizeAssetExt, resolveAssetTypeLabel } from '@/ui/shared/helpers/asset-meta'

interface ResourceFolderNode extends AssetFolder {
  label: string
  type: 'folder' | 'root'
  children: ResourceFolderNode[]
}

interface AssetView extends AssetItem {
  displayName: string
  ext: string
  sizeValue: number
}

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    projectId: string
    title?: string
    accept?: string
    imageOnly?: boolean
    selectedUrl?: string
  }>(),
  {
    title: '',
    accept: '',
    imageOnly: false,
    selectedUrl: '',
  },
)

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'select', asset: AssetItem): void
}>()

const { t } = useI18n()

const folderSearch = ref('')
const assetSearch = ref('')
const folders = ref<AssetFolder[]>([])
const assets = ref<AssetItem[]>([])
const selectedFolderId = ref('root')
const selectedAssetId = ref('')
const loading = ref(false)
const uploading = ref(false)
const fileInputRef = ref<HTMLInputElement | null>(null)

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

function showSuccessMessage(message: string): void {
  ElMessage.success(message as never)
}

function decodeAssetName(value: string): string {
  if (!value) return ''
  try {
    return decodeURIComponent(value)
  } catch {
    try {
      return decodeURIComponent(escape(value))
    } catch {
      return value
    }
  }
}

function getAssetExt(asset: Partial<AssetItem> & { displayName?: string } = {}): string {
  const name = asset.displayName || asset.name || asset.originalName || ''
  const index = name.lastIndexOf('.')
  if (index > -1 && index < name.length - 1) {
    return normalizeAssetExt(name.slice(index + 1).toLowerCase())
  }
  const mime = String(asset.mimeType || '').toLowerCase()
  if (mime.includes('/')) return normalizeAssetExt(mime.split('/').pop() || '')
  return normalizeAssetExt(asset.type || '')
}

function isImageAsset(asset: Partial<AssetItem> | null | undefined): boolean {
  const type = String(asset?.type || '').toLowerCase()
  const mime = String(asset?.mimeType || '').toLowerCase()
  const ext = getAssetExt(asset || {})
  if (type === 'image' || type === 'svg') return true
  if (mime.startsWith('image/')) return true
  return ['png', 'jpg', 'jpeg', 'gif', 'webp', 'svg', 'bmp', 'ico'].includes(ext)
}

function resolveAssetUrl(asset: Partial<AssetItem> | null | undefined): string {
  return String(asset?.url || asset?.src || asset?.path || '')
}

function formatSize(size: number | string | null | undefined): string {
  if (size === null || size === undefined || size === '') return '-'
  const value = Number(size)
  if (Number.isNaN(value)) return '-'
  if (value === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  let idx = 0
  let num = value
  while (num >= 1024 && idx < units.length - 1) {
    num /= 1024
    idx += 1
  }
  return `${num.toFixed(num >= 10 ? 0 : 1)} ${units[idx]}`
}

function buildFolderTree(items: AssetFolder[] | null | undefined): ResourceFolderNode[] {
  const list = Array.isArray(items) ? items : []
  const nodes: ResourceFolderNode[] = list.map((item) => ({
    ...item,
    label: decodeAssetName(item.name || t('resourcePanel.untitledFolder')),
    type: 'folder' as const,
    children: [],
  }))
  const map = new Map<string, ResourceFolderNode>(nodes.map((item) => [item.id, item]))
  const root: ResourceFolderNode = {
    id: 'root',
    name: t('resourcePanel.rootLabel'),
    label: t('resourcePanel.rootLabel'),
    type: 'root',
    children: [],
  }
  nodes.forEach((node) => {
    const parent = node.parentId ? map.get(node.parentId) : undefined
    if (parent) parent.children.push(node)
    else root.children.push(node)
  })
  return [root]
}

const folderTree = computed<ResourceFolderNode[]>(() => buildFolderTree(folders.value))

function filterFolderTree(nodes: ResourceFolderNode[], keyword: string): ResourceFolderNode[] {
  const normalized = keyword.trim().toLowerCase()
  if (!normalized) return nodes
  return nodes.flatMap((node) => {
    const children = filterFolderTree(node.children || [], normalized)
    if (node.label.toLowerCase().includes(normalized) || children.length) {
      return [{ ...node, children }]
    }
    return []
  })
}

const filteredFolderTree = computed<ResourceFolderNode[]>(() =>
  filterFolderTree(folderTree.value, folderSearch.value),
)

const normalizedAssets = computed<AssetView[]>(() =>
  (assets.value || []).map((asset) => {
    const displayName = decodeAssetName(asset.name || asset.originalName || '')
    const sizeValue = Number(
      asset.size ??
        asset.fileSize ??
        asset.file_size ??
        asset.length ??
        asset.bytes ??
        asset.metadata?.size ??
        asset.metadata?.fileSize ??
        asset.metadata?.length ??
        0,
    )
    return {
      ...asset,
      displayName,
      ext: getAssetExt({ ...asset, displayName }),
      sizeValue: Number.isFinite(sizeValue) ? sizeValue : 0,
    }
  }),
)

const filteredAssets = computed<AssetView[]>(() => {
  let list = normalizedAssets.value
  if (props.imageOnly) list = list.filter((item) => isImageAsset(item))
  if (selectedFolderId.value !== 'root') {
    list = list.filter((item) => item.folderId === selectedFolderId.value)
  }
  const keyword = assetSearch.value.trim().toLowerCase()
  if (keyword) {
    list = list.filter((item) => item.displayName.toLowerCase().includes(keyword))
  }
  return list
})

const selectedAsset = computed<AssetView | null>(
  () => normalizedAssets.value.find((item) => item.id === selectedAssetId.value) || null,
)

function handleFolderClick(data: ResourceFolderNode | null | undefined): void {
  if (!data) return
  selectedFolderId.value = data.id
}

function handleAssetSelect(asset: AssetView): void {
  selectedAssetId.value = asset.id
}

function confirmSelect(): void {
  if (!selectedAsset.value) return
  emit('select', selectedAsset.value)
  visible.value = false
}

function triggerFileSelect(): void {
  fileInputRef.value?.click()
}

async function handleFileInputChange(event: Event): Promise<void> {
  const target = event.target as HTMLInputElement | null
  const files = Array.from(target?.files || [])
  if (target) target.value = ''
  if (!files.length) return
  await uploadFiles(files)
}

async function handleUploadDrop(event: DragEvent): Promise<void> {
  const files = Array.from(event.dataTransfer?.files || [])
  if (!files.length) return
  await uploadFiles(files)
}

async function uploadFiles(files: File[]): Promise<void> {
  if (!props.projectId || !files.length) return
  uploading.value = true
  const folderId = selectedFolderId.value === 'root' ? null : selectedFolderId.value
  const duplicated = files.some((file) =>
    normalizedAssets.value.some(
      (asset) => asset.displayName === file.name && (asset.folderId || null) === folderId,
    ),
  )
  let conflictStrategy = 'rename'
  if (duplicated) {
    const result = await ElMessageBox.confirm(
      t('resourcePanel.uploadConflictMessage'),
      t('resourcePanel.uploadConflictTitle'),
      {
        confirmButtonText: t('resourcePanel.replace'),
        cancelButtonText: t('resourcePanel.rename'),
        type: 'warning',
      },
    ).catch(() => null)
    conflictStrategy = result ? 'replace' : 'rename'
  }
  try {
    const response = await assetApi.uploadAssets(props.projectId, files, folderId || undefined, {
      conflictStrategy,
    })
    const data = unwrapApiData<{ assets?: AssetItem[] }>(response)
    showSuccessMessage(t('resourcePanel.uploadSuccess'))
    await loadAssets()
    const uploaded = Array.isArray(data?.assets) ? data.assets[0] : null
    if (uploaded?.id) selectedAssetId.value = uploaded.id
  } finally {
    uploading.value = false
  }
}

async function loadFolders(): Promise<void> {
  if (!props.projectId) return
  const response = await assetApi.getFolders(props.projectId)
  const data = unwrapApiData<{ folders?: AssetFolder[] }>(response)
  folders.value = Array.isArray(data?.folders) ? data.folders : []
}

async function loadAssets(): Promise<void> {
  if (!props.projectId) return
  const response = await assetApi.getAssets(props.projectId)
  const data = unwrapApiData<{ assets?: AssetItem[] }>(response)
  assets.value = Array.isArray(data?.assets) ? data.assets : []
  if (!selectedAssetId.value && props.selectedUrl) {
    const matched = assets.value.find((asset) => resolveAssetUrl(asset) === props.selectedUrl)
    if (matched?.id) selectedAssetId.value = matched.id
  }
}

async function reloadResources(): Promise<void> {
  if (!visible.value || !props.projectId) return
  loading.value = true
  try {
    await Promise.all([loadFolders(), loadAssets()])
  } finally {
    loading.value = false
  }
}

watch(
  () => props.modelValue,
  (value) => {
    if (value) {
      selectedAssetId.value = ''
      void reloadResources()
    }
  },
)

watch(
  () => props.projectId,
  () => {
    if (visible.value) void reloadResources()
  },
)
</script>

<template>
  <el-dialog
    v-model="visible"
    :title="title || t('resourceManager.title')"
    width="760px"
    top="8vh"
    append-to-body
    class="asset-manager-dialog"
  >
    <div class="asset-manager" v-loading="loading">
      <aside class="asset-manager__folders">
        <div class="asset-manager__toolbar">
          <el-input
            v-model="folderSearch"
            size="small"
            :placeholder="t('resourcePanel.searchGroups')"
            clearable
          />
        </div>
        <el-scrollbar class="asset-manager__folder-scroll">
          <el-tree
            class="asset-manager__tree"
            node-key="id"
            :data="filteredFolderTree"
            :expand-on-click-node="false"
            highlight-current
            default-expand-all
            @node-click="handleFolderClick"
          >
            <template #default="{ data }">
              <div class="asset-manager__folder-node">
                <span class="asset-manager__folder-icon">📁</span>
                <span class="asset-manager__folder-label" :title="data.label">{{
                  data.label
                }}</span>
              </div>
            </template>
          </el-tree>
        </el-scrollbar>
      </aside>

      <section class="asset-manager__content">
        <div class="asset-manager__actions">
          <el-input
            v-model="assetSearch"
            size="small"
            :placeholder="t('resourcePanel.searchAssets')"
            clearable
          />
          <el-button size="small" type="primary" :loading="uploading" @click="triggerFileSelect">
            <el-icon><IconEpUpload /></el-icon>
            <span>{{ t('resourceManager.upload') }}</span>
          </el-button>
        </div>

        <div
          class="asset-manager__dropzone"
          @click="triggerFileSelect"
          @dragover.prevent
          @drop.prevent="handleUploadDrop"
        >
          {{ t('resourcePanel.uploadHint') }}
        </div>

        <el-scrollbar class="asset-manager__asset-scroll">
          <div v-if="filteredAssets.length" class="asset-manager__grid">
            <button
              v-for="asset in filteredAssets"
              :key="asset.id"
              type="button"
              class="asset-manager__card"
              :class="{ 'is-selected': selectedAssetId === asset.id }"
              @click="handleAssetSelect(asset)"
              @dblclick="confirmSelect"
            >
              <span class="asset-manager__thumb">
                <img
                  v-if="isImageAsset(asset)"
                  :src="asset.thumbnailUrl || resolveAssetUrl(asset)"
                  :alt="asset.displayName"
                />
                <span v-else class="asset-manager__thumb-placeholder">
                  {{ resolveAssetTypeLabel(asset) }}
                </span>
              </span>
              <span class="asset-manager__name" :title="asset.displayName">
                {{ asset.displayName }}
              </span>
              <span class="asset-manager__meta">
                {{ resolveAssetTypeLabel(asset) }} · {{ formatSize(asset.sizeValue) }}
              </span>
            </button>
          </div>
          <el-empty v-else :description="t('resourceManager.empty')" />
        </el-scrollbar>
      </section>
    </div>

    <input
      ref="fileInputRef"
      class="asset-manager__hidden-input"
      type="file"
      multiple
      :accept="accept"
      @change="handleFileInputChange"
    />

    <template #footer>
      <el-button @click="visible = false">{{ t('resourcePanel.cancel') }}</el-button>
      <el-button type="primary" :disabled="!selectedAsset" @click="confirmSelect">
        {{ t('resourceManager.select') }}
      </el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.asset-manager {
  display: grid;
  grid-template-columns: 210px minmax(0, 1fr);
  height: min(620px, 70vh);
  min-height: 420px;
  overflow: hidden;
  border: 1px solid var(--designer-border-soft);
  border-radius: var(--designer-radius-md);
  background: var(--designer-shell-surface);
}

.asset-manager__folders {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 8px;
  padding: 10px;
  border-right: 1px solid var(--designer-border-soft);
  background: var(--designer-group-surface);
}

.asset-manager__toolbar {
  flex: 0 0 auto;
}

.asset-manager__folder-scroll,
.asset-manager__asset-scroll {
  flex: 1;
  min-height: 0;
}

.asset-manager__folder-node {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 6px;
  padding: 3px 0;
}

.asset-manager__folder-label {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.asset-manager__content {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  gap: 10px;
  padding: 10px;
}

.asset-manager__actions {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 8px;
  align-items: center;
}

.asset-manager__dropzone {
  flex: 0 0 auto;
  padding: 10px 12px;
  border: 1px dashed var(--designer-border-strong);
  border-radius: var(--designer-radius-md);
  color: var(--designer-text-muted);
  font-size: var(--designer-font-caption);
  text-align: center;
  cursor: pointer;
}

.asset-manager__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(116px, 1fr));
  gap: 10px;
  padding: 2px 2px 12px;
}

.asset-manager__card {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 6px;
  padding: 8px;
  border: 1px solid var(--designer-border-soft);
  border-radius: var(--designer-radius-md);
  background: var(--designer-shell-surface);
  color: var(--designer-text-primary);
  cursor: pointer;
  text-align: left;
}

.asset-manager__card:hover,
.asset-manager__card.is-selected {
  border-color: var(--designer-primary-border);
  background: var(--designer-primary-soft);
}

.asset-manager__thumb {
  display: flex;
  height: 92px;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border-radius: var(--designer-radius-sm);
  background: var(--designer-group-surface);
}

.asset-manager__thumb img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.asset-manager__thumb-placeholder {
  color: var(--designer-text-muted);
  font-size: var(--designer-font-caption);
}

.asset-manager__name,
.asset-manager__meta {
  display: block;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.asset-manager__name {
  font-size: var(--designer-font-label);
}

.asset-manager__meta {
  color: var(--designer-text-muted);
  font-size: var(--designer-font-caption);
}

.asset-manager__hidden-input {
  display: none;
}
</style>
