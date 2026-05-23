<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import MonacoEditor from '@/ui/shared/widgets/base/monaco-editor-async'
import assetApi from '@/services/assetApi'
import IconEpFolder from '~icons/ep/folder'
import IconEpPictureFilled from '~icons/ep/picture-filled'
import {
  buildConfigAssetTree,
  filterConfigAssetNode,
  resolveConfigAssetUrl,
  unwrapApiData,
} from '../property-panel-config-assets'

interface StyleConfigPreset {
  id: string
  label: string
  content: string
}

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    content: string
    title: string
    presets: StyleConfigPreset[]
    projectId?: string | null
    width?: string
    selectorLabel: string
    selectorTokens: string[]
    selectorHelp?: string
    templateLabel: string
    templatePlaceholder: string
    filterLabel: string
    searchTemplate: string
    assetLibrary: string
    searchAssets: string
    clearText: string
    cancelText: string
    saveText: string
  }>(),
  {
    projectId: null,
    width: '980px',
    selectorHelp: '',
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  save: [content: string]
}>()

const draft = ref('')
const selectedPresetId = ref('')
const presetSearch = ref('')
const assetFolders = ref<any[]>([])
const assets = ref<any[]>([])
const assetSearch = ref('')
const assetTreeRef = ref<any>(null)
const editorRef = ref<{ focus?: () => void; insertText?: (text: string) => void } | null>(null)

const dialogVisible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const normalizedPresets = computed<StyleConfigPreset[]>(() => {
  const raw = props.presets as unknown
  const source =
    raw && typeof raw === 'object' && 'value' in raw ? (raw as { value?: unknown }).value : raw
  if (!Array.isArray(source)) return []
  return source
    .map((item) => ({
      id: String((item as StyleConfigPreset).id || ''),
      label: String((item as StyleConfigPreset).label || ''),
      content: String((item as StyleConfigPreset).content || ''),
    }))
    .filter((item) => item.id && item.label)
})

const filteredPresets = computed(() => {
  const keyword = presetSearch.value.trim().toLowerCase()
  if (!keyword) return normalizedPresets.value
  return normalizedPresets.value.filter((item) => item.label.toLowerCase().includes(keyword))
})

const assetTree = computed(() => buildConfigAssetTree(assetFolders.value, assets.value))

watch(
  () => props.content,
  (value) => {
    if (!dialogVisible.value) {
      draft.value = String(value || '')
    }
  },
  { immediate: true },
)

watch(dialogVisible, (visible) => {
  if (!visible) return
  draft.value = String(props.content || '')
  selectedPresetId.value = ''
  presetSearch.value = ''
  assetSearch.value = ''
  loadAssets()
  nextTick(() => {
    window.setTimeout(() => editorRef.value?.focus?.(), 0)
  })
})

watch(assetSearch, () => {
  assetTreeRef.value?.filter?.(assetSearch.value)
})

function clearDraft(): void {
  draft.value = ''
}

function saveDraft(): void {
  emit('save', draft.value)
  dialogVisible.value = false
}

function applyPreset(id: string): void {
  const target = normalizedPresets.value.find((item) => item.id === id)
  if (!target) return
  const current = draft.value.trim()
  const remark = target.label ? `/* ${target.label} */\n` : ''
  const nextContent = `${remark}${target.content}`
  draft.value = current ? `${current}\n\n${nextContent}` : nextContent
  nextTick(() => editorRef.value?.focus?.())
}

function handleAssetNodeDblClick(data: any): void {
  if (!data || data.type !== 'asset') return
  const url = resolveConfigAssetUrl(data.raw)
  if (!url) return
  editorRef.value?.insertText?.(`url("${url}")`)
}

async function loadAssets(): Promise<void> {
  if (!props.projectId) {
    assetFolders.value = []
    assets.value = []
    return
  }
  try {
    const [folderResponse, assetResponse] = await Promise.all([
      assetApi.getFolders(props.projectId),
      assetApi.getAssets(props.projectId),
    ])
    const folderData = unwrapApiData(folderResponse) as { folders?: unknown } | null
    const assetData = unwrapApiData(assetResponse) as { assets?: unknown } | null
    assetFolders.value = Array.isArray(folderData?.folders) ? folderData.folders : []
    assets.value = Array.isArray(assetData?.assets) ? assetData.assets : []
  } catch {
    assetFolders.value = []
    assets.value = []
  }
}
</script>

<template>
  <el-dialog
    v-model="dialogVisible"
    :title="title"
    :width="width"
    top="4vh"
    :close-on-click-modal="false"
    :lock-scroll="false"
    append-to-body
    :modal-append-to-body="true"
    :z-index="3000"
  >
    <div class="style-config-dialog">
      <div class="style-config-toolbar">
        <div class="style-config-toolbar__item">
          <span class="style-config-label">{{ templateLabel }}</span>
          <el-select
            v-model="selectedPresetId"
            size="small"
            class="style-config-select"
            :placeholder="templatePlaceholder"
            :teleported="false"
            @change="applyPreset"
          >
            <el-option
              v-for="item in filteredPresets"
              :key="item.id"
              :label="item.label"
              :value="item.id"
            />
          </el-select>
        </div>
        <div class="style-config-toolbar__item">
          <span class="style-config-label">{{ filterLabel }}</span>
          <el-input
            v-model="presetSearch"
            size="small"
            class="style-config-select"
            :placeholder="searchTemplate"
            clearable
          />
        </div>
      </div>

      <div class="style-config-selectors">
        <span>{{ selectorLabel }}</span>
        <code v-for="token in selectorTokens" :key="token">{{ token }}</code>
        <span v-if="selectorHelp" class="style-config-selector-help">{{ selectorHelp }}</span>
      </div>

      <div class="style-config-body">
        <div class="style-config-editor">
          <MonacoEditor
            ref="editorRef"
            v-model="draft"
            language="css"
            height="520px"
            :options="{ tabSize: 2, wordWrap: 'on', readOnly: false }"
          />
        </div>
        <div class="style-config-assets">
          <div class="style-config-assets__header">{{ assetLibrary }}</div>
          <el-input v-model="assetSearch" size="small" :placeholder="searchAssets" clearable />
          <div class="style-config-assets__body">
            <el-scrollbar>
              <el-tree
                ref="assetTreeRef"
                :data="assetTree"
                node-key="id"
                default-expand-all
                :expand-on-click-node="false"
                :filter-node-method="filterConfigAssetNode"
              >
                <template #default="{ data }">
                  <div
                    class="style-config-asset-node"
                    :class="`node-${data.type}`"
                    @dblclick.stop="handleAssetNodeDblClick(data)"
                  >
                    <el-icon class="style-config-asset-node__icon">
                      <IconEpFolder v-if="data.type === 'folder'" />
                      <IconEpPictureFilled v-else />
                    </el-icon>
                    <span class="style-config-asset-node__label">{{ data.label }}</span>
                  </div>
                </template>
              </el-tree>
            </el-scrollbar>
          </div>
        </div>
      </div>
    </div>
    <template #footer>
      <el-button @click="clearDraft">{{ clearText }}</el-button>
      <el-button @click="dialogVisible = false">{{ cancelText }}</el-button>
      <el-button type="primary" @click="saveDraft">{{ saveText }}</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.style-config-dialog {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.style-config-toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  padding: 8px 10px;
  border: 1px solid var(--designer-border-color);
  border-radius: var(--designer-radius-md);
  background: var(--designer-group-surface);
}

.style-config-toolbar__item {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 6px;
}

.style-config-label {
  flex: 0 0 auto;
  color: var(--designer-text-secondary);
  font-size: var(--designer-font-xs);
}

.style-config-select {
  width: 190px;
}

.style-config-selectors {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  color: var(--designer-text-secondary);
  font-size: var(--designer-font-xs);
  line-height: 1.5;
}

.style-config-selectors code {
  padding: 2px 6px;
  border: 1px solid var(--designer-border-color);
  border-radius: var(--designer-radius-sm);
  background: var(--designer-group-surface);
  color: var(--designer-text-primary);
  font-family: var(--designer-font-mono, Consolas, monospace);
}

.style-config-selector-help {
  min-width: 180px;
  flex: 1;
}

.style-config-body {
  display: flex;
  gap: 12px;
  align-items: stretch;
}

.style-config-editor {
  min-height: 520px;
  flex: 1;
  min-width: 0;
}

.style-config-assets {
  display: flex;
  width: 240px;
  flex-direction: column;
  gap: 8px;
  padding-left: 12px;
  border-left: 1px solid var(--designer-border-color);
}

.style-config-assets__header {
  color: var(--designer-text-secondary);
  font-size: var(--designer-font-label);
  font-weight: 600;
}

.style-config-assets__body {
  min-height: 0;
  flex: 1;
  padding: 6px;
  border: 1px solid var(--designer-border-color);
  border-radius: var(--designer-radius-md);
  background: var(--designer-shell-surface);
}

.style-config-asset-node {
  display: flex;
  width: 100%;
  min-width: 0;
  align-items: center;
  gap: 6px;
  font-size: var(--designer-font-label);
  line-height: 28px;
}

.style-config-asset-node__icon {
  flex: 0 0 auto;
  color: var(--designer-text-secondary);
}

.style-config-asset-node__label {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
