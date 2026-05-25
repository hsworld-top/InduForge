<!--
  脚本编辑通用弹窗：只负责编辑体验，不决定脚本保存位置。
-->
<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import IconEpDocument from '~icons/ep/document'
import IconEpEditPen from '~icons/ep/edit-pen'
import IconEpFolder from '~icons/ep/folder'
import IconEpList from '~icons/ep/list'
import MonacoEditor from '@/ui/shared/widgets/base/monaco-editor-async'

interface SidebarNodeLike {
  id: string
  label: string
  type: string
}

interface PageVariableRowLike {
  name: string
  type?: string
}

export interface ScriptEditorTab {
  key: string
  label: string
}

const props = withDefaults(
  defineProps<{
    dialogTitle: string
    metaTitle: string
    metaDescription?: string
    scope?: 'global' | 'page'
    customScriptSidebarTree?: SidebarNodeLike[]
    pageSidebarTree?: SidebarNodeLike[]
    pageVariableRows?: PageVariableRowLike[]
    jsCompletions?: unknown[]
    filterSidebarNode: (value: string, data: SidebarNodeLike) => boolean
    beforeClose?: (...args: unknown[]) => void
    tabs?: ScriptEditorTab[]
  }>(),
  {
    metaDescription: '',
    scope: 'global',
  },
)

const emit = defineEmits([
  'openVariableEnum',
  'save',
  'customInsert',
  'pageInsert',
  'pageVariableInsert',
])
const visible = defineModel<boolean>({ default: false })
const code = defineModel<string>('code', { default: '' })
const scriptSearch = defineModel<string>('scriptSearch', { default: '' })
const pageSearch = defineModel<string>('pageSearch', { default: '' })
const pageVariableSearch = defineModel<string>('pageVariableSearch', { default: '' })
const activeTab = defineModel<string>('activeTab', { default: '' })

interface TreeFilterLike {
  filter?: (value: string) => void
}

interface MonacoExposeLike {
  insertText?: (text: string) => void
  format?: () => void
}

const monacoRef = ref<MonacoExposeLike | null>(null)
const customTreeRef = ref<TreeFilterLike | null>(null)
const pageTreeRef = ref<TreeFilterLike | null>(null)
const sidebarTab = ref('customScripts')
const { t } = useI18n()

const scopeLabel = computed(() =>
  props.scope === 'page' ? t('scriptPanel.page.scope') : t('scriptPanel.global.scope'),
)
const visibleTabs = computed(() => props.tabs || [])

watch(scriptSearch, (value: string) => {
  customTreeRef.value?.filter?.(value)
})

watch(pageSearch, (value: string) => {
  pageTreeRef.value?.filter?.(value)
})

defineExpose({
  insertText: (text: string) => monacoRef.value?.insertText?.(text),
  format: () => monacoRef.value?.format?.(),
})

function handleCustomInsert(data: SidebarNodeLike) {
  emit('customInsert', data)
}

function handlePageInsert(data: SidebarNodeLike) {
  emit('pageInsert', data)
}

function handlePageVariableInsert(row: PageVariableRowLike) {
  emit('pageVariableInsert', row)
}
</script>

<template>
  <el-dialog
    v-model="visible"
    :title="dialogTitle"
    class="script-editor-dialog"
    width="1120px"
    top="3vh"
    :close-on-click-modal="false"
    :lock-scroll="false"
    :before-close="beforeClose"
  >
    <div class="editor-topbar">
      <el-tabs v-if="visibleTabs.length" v-model="activeTab" class="editor-tabs">
        <el-tab-pane v-for="tab in visibleTabs" :key="tab.key" :label="tab.label" :name="tab.key" />
      </el-tabs>
      <div v-else class="meta-title">{{ metaTitle }}</div>
      <slot name="meta-extra" />
      <div class="meta-actions">
        <el-tooltip :content="t('scriptPanel.editor.variableEnum')" placement="top">
          <el-button class="icon-button" size="small" circle @click="emit('openVariableEnum')">
            <IconEpList />
          </el-button>
        </el-tooltip>
      </div>
    </div>
    <div class="editor-body">
      <div class="editor-main">
        <MonacoEditor
          ref="monacoRef"
          v-model="code"
          language="javascript"
          height="520px"
          :completions="jsCompletions"
        />
      </div>
      <div class="editor-sidebar">
        <el-tabs v-model="sidebarTab" class="sidebar-tabs">
          <el-tab-pane :label="t('scriptPanel.editor.customScripts')" name="customScripts">
            <div class="sidebar-section">
              <el-input
                v-model="scriptSearch"
                size="small"
                :placeholder="t('scriptPanel.editor.searchScripts')"
                clearable
              />
              <div class="sidebar-scroll">
                <el-tree
                  ref="customTreeRef"
                  :data="customScriptSidebarTree"
                  node-key="id"
                  :default-expand-all="true"
                  :expand-on-click-node="false"
                  :filter-node-method="filterSidebarNode"
                  @node-click="handleCustomInsert"
                >
                  <template #default="{ data }">
                    <div class="tree-node" :class="`node-${data.type}`">
                      <el-icon class="node-icon icon-custom">
                        <IconEpFolder v-if="data.type === 'group'" />
                        <IconEpEditPen v-else />
                      </el-icon>
                      <span
                        class="node-label"
                        :class="{
                          'is-group': data.type === 'group' || data.type === 'folder',
                        }"
                        >{{ data.label }}</span
                      >
                    </div>
                  </template>
                </el-tree>
              </div>
            </div>
          </el-tab-pane>
          <el-tab-pane :label="t('scriptPanel.editor.pageVariables')" name="pageVariables">
            <div class="sidebar-section">
              <el-input
                v-model="pageVariableSearch"
                size="small"
                :placeholder="t('scriptPanel.editor.searchPageVariables')"
                clearable
              />
              <div class="sidebar-scroll page-var-list">
                <div
                  v-for="row in pageVariableRows"
                  :key="row.name"
                  class="page-var-item"
                  @click="handlePageVariableInsert(row)"
                >
                  <el-icon class="node-icon icon-variable">
                    <IconEpList />
                  </el-icon>
                  <span class="node-label">{{ row.name }}</span>
                  <span class="page-var-type">{{ row.type || 'string' }}</span>
                </div>
              </div>
            </div>
          </el-tab-pane>
          <el-tab-pane :label="t('scriptPanel.editor.pages')" name="pages">
            <div class="sidebar-section">
              <el-input
                v-model="pageSearch"
                size="small"
                :placeholder="t('scriptPanel.editor.searchPages')"
                clearable
              />
              <div class="sidebar-scroll">
                <el-tree
                  ref="pageTreeRef"
                  :data="pageSidebarTree"
                  node-key="id"
                  :default-expand-all="true"
                  :expand-on-click-node="false"
                  :filter-node-method="filterSidebarNode"
                  @node-click="handlePageInsert"
                >
                  <template #default="{ data }">
                    <div class="tree-node" :class="`node-${data.type}`">
                      <el-icon class="node-icon icon-page">
                        <IconEpFolder v-if="data.type === 'folder'" />
                        <IconEpDocument v-else />
                      </el-icon>
                      <span
                        class="node-label"
                        :class="{
                          'is-group': data.type === 'group' || data.type === 'folder',
                        }"
                        >{{ data.label }}</span
                      >
                    </div>
                  </template>
                </el-tree>
              </div>
            </div>
          </el-tab-pane>
        </el-tabs>
      </div>
    </div>
    <template #footer>
      <el-button @click="visible = false">{{ t('scriptPanel.editor.cancel') }}</el-button>
      <el-button type="primary" @click="emit('save')">{{
        t('scriptPanel.editor.saveShortcut')
      }}</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.script-editor-dialog {
  --script-dialog-border: #e6ebf2;
  --script-dialog-soft: #f7f9fc;
  --script-dialog-hover: #f0f5ff;
  --script-dialog-primary: #3b82f6;
  border-radius: 12px;
  overflow: hidden;
}

.script-editor-dialog :deep(.el-dialog__header) {
  margin-right: 0;
  padding: 18px 20px 14px;
  border-bottom: 1px solid var(--script-dialog-border);
}

.script-editor-dialog :deep(.el-dialog__title) {
  color: var(--designer-text-primary);
  font-size: 16px;
  font-weight: 600;
}

.script-editor-dialog :deep(.el-dialog__body) {
  padding: 14px 18px 0;
}

.script-editor-dialog :deep(.el-dialog__footer) {
  padding: 14px 18px 16px;
  border-top: 1px solid var(--script-dialog-border);
  background: #fff;
}

.editor-body {
  display: flex;
  gap: 14px;
  flex: 1;
  align-items: stretch;
  height: 520px;
}

.editor-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 12px 16px;
  padding: 10px 12px;
  border: 1px solid var(--designer-border-color);
  border-radius: 6px;
  background: var(--designer-group-surface);
  margin-bottom: 10px;
  align-items: center;
}

.editor-topbar {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 46px;
  margin-bottom: 12px;
  padding: 0 12px;
  border: 1px solid var(--script-dialog-border);
  border-radius: 8px;
  background: var(--script-dialog-soft);
}

.editor-tabs {
  flex: 1;
  min-width: 0;
}

.editor-tabs :deep(.el-tabs__header) {
  margin: 0;
}

.editor-tabs :deep(.el-tabs__nav-wrap::after) {
  display: none;
}

.editor-tabs :deep(.el-tabs__item) {
  height: 40px;
  padding: 0 14px;
  font-size: 13px;
}

.meta-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--designer-text-primary);
}

.meta-desc {
  font-size: 12px;
  color: var(--designer-text-secondary);
}

.meta-actions {
  margin-left: auto;
  display: flex;
  align-items: center;
}

.icon-button {
  background: #eaf2ff;
  border: none;
  color: var(--script-dialog-primary);
}

.icon-button:hover {
  background: #dbeafe;
}

.editor-main {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  border: 1px solid var(--script-dialog-border);
  border-radius: 8px;
  background: #fff;
}

.editor-sidebar {
  width: 246px;
  height: 520px;
  padding: 8px 10px 10px;
  border: 1px solid var(--script-dialog-border);
  border-radius: 8px;
  background: var(--script-dialog-soft);
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.sidebar-tabs {
  display: flex;
  flex: 1;
  min-height: 0;
  flex-direction: column;
}

.sidebar-tabs :deep(.el-tabs__header) {
  margin: 0 0 10px;
}

.sidebar-tabs :deep(.el-tabs__nav-wrap::after) {
  height: 1px;
  background: var(--designer-border-color);
}

.sidebar-tabs :deep(.el-tabs__item) {
  height: 32px;
  padding: 0 9px;
  font-size: 12px;
}

.sidebar-tabs :deep(.el-tabs__content),
.sidebar-tabs :deep(.el-tab-pane) {
  flex: 1;
  min-height: 0;
}

.sidebar-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
  flex: 1;
  min-height: 0;
}

.sidebar-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--designer-text-secondary);
}

.sidebar-scroll {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 2px 2px 2px 0;
}

.tree-node {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  width: 100%;
  min-height: 32px;
  padding: 6px 8px;
  border-radius: 6px;
  transition: background-color 0.2s;
}

.tree-node:hover {
  background: var(--script-dialog-hover);
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

.node-label {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
}

.node-label.is-group {
  font-weight: 600;
  color: var(--designer-text-primary);
}

.page-var-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.page-var-item {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 32px;
  padding: 5px 8px;
  border-radius: var(--designer-radius-sm);
  cursor: pointer;
}

.page-var-item:hover {
  background: var(--script-dialog-hover);
}

.page-var-type {
  margin-left: auto;
  color: var(--designer-text-muted);
  font-size: 12px;
}
</style>
