<!--
  ComponentPanel - 组件物料面板
  当前版本仅保留：6 个布局容器 + Button
-->
<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { computed, h, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { componentRegistry } from '@/editor-core'
import { useEditorStore } from '@/stores/editor-store'
import { endDrag, startDrag } from '@/ui/editors/page/canvas/composables/use-drag-state'

interface ComponentItemLike {
  type: string
  name: string
}

type PreviewComponentMapLike = Record<string, () => ReturnType<typeof h>>

const keyword = ref('')
const activeSections = ref(['layout', 'ui', 'system'])
const { t, locale } = useI18n()
const editorStore = useEditorStore()
const { projectI18n } = storeToRefs(editorStore)

const layoutTypeOrder = [
  'HorizontalLayout',
  'VerticalLayout',
  'Collapse',
  'Tabs',
  'FormLayout',
  'ElContainer',
]

const allowedTypesByCategory: Record<string, string[]> = {
  layout: layoutTypeOrder,
  uiPc: [
    'Button',
    'Input',
    'InputNumber',
    'Select',
    'Radio',
    'Checkbox',
    'Switch',
    'Table',
    'Pagination',
  ],
}

function filterItemsByCategory(category: string): ComponentItemLike[] {
  void locale.value
  const keywordValue = keyword.value.trim().toLowerCase()
  const allowedTypes = allowedTypesByCategory[category] ?? []
  const items = componentRegistry.getByCategory(category) as ComponentItemLike[]
  const filtered = items.filter((item) => {
    if (!allowedTypes.includes(item.type)) return false
    if (!keywordValue) return true
    return (
      item.type.toLowerCase().includes(keywordValue) ||
      item.name.toLowerCase().includes(keywordValue)
    )
  })

  if (category !== 'layout') return filtered
  return filtered.sort((a, b) => layoutTypeOrder.indexOf(a.type) - layoutTypeOrder.indexOf(b.type))
}

function filterSystemItems(): ComponentItemLike[] {
  void locale.value
  const keywordValue = keyword.value.trim().toLowerCase()
  const types = ['UserAvatarMenu', ...(projectI18n.value.enabled ? ['LanguageSwitcher'] : [])]
  return types
    .map((type) => componentRegistry.get(type) as ComponentItemLike | undefined)
    .filter((item): item is ComponentItemLike => Boolean(item))
    .filter((item) => {
      if (!keywordValue) return true
      return (
        item.type.toLowerCase().includes(keywordValue) ||
        item.name.toLowerCase().includes(keywordValue)
      )
    })
}

const layoutItems = computed<ComponentItemLike[]>(() => filterItemsByCategory('layout'))
const uiItems = computed<ComponentItemLike[]>(() => filterItemsByCategory('uiPc'))
const systemItems = computed<ComponentItemLike[]>(filterSystemItems)
const showSystemSection = computed(() => systemItems.value.length > 0 || !keyword.value.trim())

function getPreviewComponent(type: string): { render: () => ReturnType<typeof h> } {
  const previewMap: PreviewComponentMapLike = {
    Button: () =>
      h('div', { class: 'preview-button' }, [h('span', { class: 'preview-button-line' })]),
    Input: () => h('div', { class: 'preview-input' }, [h('span', { class: 'preview-input-line' })]),
    InputNumber: () =>
      h('div', { class: 'preview-input-number' }, [
        h('span', { class: 'preview-input-number-line' }),
        h('span', { class: 'preview-input-number-step' }),
      ]),
    Select: () =>
      h('div', { class: 'preview-select' }, [
        h('span', { class: 'preview-select-line' }),
        h('span', { class: 'preview-select-arrow' }),
      ]),
    Radio: () =>
      h('div', { class: 'preview-choice' }, [
        h('span', { class: 'preview-radio-dot' }),
        h('span', { class: 'preview-choice-line' }),
      ]),
    Checkbox: () =>
      h('div', { class: 'preview-choice' }, [
        h('span', { class: 'preview-checkbox-box' }),
        h('span', { class: 'preview-choice-line' }),
      ]),
    Switch: () =>
      h('div', { class: 'preview-switch' }, [h('span', { class: 'preview-switch-dot' })]),
    Table: () =>
      h('div', { class: 'preview-table' }, [
        h('span', { class: 'preview-table-row is-head' }),
        h('span', { class: 'preview-table-row' }),
        h('span', { class: 'preview-table-row' }),
      ]),
    Pagination: () =>
      h('div', { class: 'preview-pagination' }, [
        h('span'),
        h('span', { class: 'is-active' }),
        h('span'),
      ]),
    LanguageSwitcher: () =>
      h('div', { class: 'preview-language-switcher' }, [
        h('span', { class: 'preview-language-content' }, [
          h('span', { class: 'preview-language-line' }),
          h('span', { class: 'preview-language-arrow' }),
        ]),
      ]),
    UserAvatarMenu: () =>
      h('div', { class: 'preview-user-avatar' }, [
        h('span', { class: 'preview-user-avatar-circle' }),
        h('span', { class: 'preview-user-avatar-lines' }, [
          h('span', { class: 'preview-user-avatar-line' }),
          h('span', { class: 'preview-user-avatar-subline' }),
        ]),
        h('span', { class: 'preview-user-avatar-arrow' }),
      ]),
    HorizontalLayout: () =>
      h('div', { class: 'preview-flex' }, [
        h('div', { class: 'preview-block' }),
        h('div', { class: 'preview-block' }),
        h('div', { class: 'preview-block' }),
      ]),
    VerticalLayout: () =>
      h('div', { class: 'preview-flex preview-flex-column' }, [
        h('div', { class: 'preview-block' }),
        h('div', { class: 'preview-block' }),
        h('div', { class: 'preview-block' }),
      ]),
    FormLayout: () =>
      h('div', { class: 'preview-form' }, [
        h('div', { class: 'preview-form-item' }),
        h('div', { class: 'preview-form-item' }),
        h('div', { class: 'preview-form-item' }),
      ]),
    ElContainer: () =>
      h('div', { class: 'preview-el-container' }, [
        h('div', { class: 'preview-el-header' }),
        h('div', { class: 'preview-el-body' }, [
          h('div', { class: 'preview-el-aside' }),
          h('div', { class: 'preview-el-main' }),
        ]),
        h('div', { class: 'preview-el-footer' }),
      ]),
    Tabs: () =>
      h('div', { class: 'preview-tabs' }, [
        h('div', { class: 'preview-tabs-header' }),
        h('div', { class: 'preview-tabs-body' }),
      ]),
    Collapse: () =>
      h('div', { class: 'preview-collapse' }, [
        h('div', { class: 'preview-collapse-header' }),
        h('div', { class: 'preview-collapse-body' }),
      ]),
  }

  const render = previewMap[type] ?? (() => h('div', { class: 'preview-unknown' }, type))
  return { render }
}

function getPreviewIcon(type: string): string {
  const iconMap: Record<string, string> = {
    Button:
      '<div style="width:60px;height:40px;border:1px solid #3b6cff;border-radius:4px;background:#fff;box-sizing:border-box;display:flex;align-items:center;justify-content:center;"><span style="width:36px;height:10px;background:#3b6cff;border-radius:2px;"></span></div>',
    Input:
      '<div style="width:60px;height:40px;border:1px solid #3b6cff;border-radius:4px;background:#fff;box-sizing:border-box;display:flex;align-items:center;justify-content:center;"><span style="width:42px;height:14px;border:1px solid #3b6cff;border-radius:3px;display:flex;align-items:center;padding-left:5px;box-sizing:border-box;"><span style="width:22px;height:3px;background:#3b6cff;border-radius:2px;"></span></span></div>',
    InputNumber:
      '<div style="width:60px;height:40px;border:1px solid #3b6cff;border-radius:4px;background:#fff;box-sizing:border-box;display:flex;align-items:center;justify-content:center;"><span style="width:42px;height:14px;border:1px solid #3b6cff;border-radius:3px;display:flex;align-items:center;justify-content:space-between;padding-left:5px;box-sizing:border-box;"><span style="width:16px;height:3px;background:#3b6cff;border-radius:2px;"></span><span style="width:10px;height:12px;background:#3b6cff;"></span></span></div>',
    Select:
      '<div style="width:60px;height:40px;border:1px solid #3b6cff;border-radius:4px;background:#fff;box-sizing:border-box;display:flex;align-items:center;justify-content:center;"><span style="width:42px;height:14px;border:1px solid #3b6cff;border-radius:3px;display:flex;align-items:center;gap:7px;padding:0 5px;box-sizing:border-box;"><span style="width:18px;height:3px;background:#3b6cff;border-radius:2px;"></span><span style="width:0;height:0;border-left:3px solid transparent;border-right:3px solid transparent;border-top:4px solid #3b6cff;"></span></span></div>',
    Radio:
      '<div style="width:60px;height:40px;border:1px solid #3b6cff;border-radius:4px;background:#fff;box-sizing:border-box;display:flex;align-items:center;justify-content:center;gap:5px;"><span style="width:10px;height:10px;border:2px solid #3b6cff;border-radius:50%;box-sizing:border-box;"></span><span style="width:24px;height:4px;background:#3b6cff;border-radius:2px;"></span></div>',
    Checkbox:
      '<div style="width:60px;height:40px;border:1px solid #3b6cff;border-radius:4px;background:#fff;box-sizing:border-box;display:flex;align-items:center;justify-content:center;gap:5px;"><span style="width:10px;height:10px;border:2px solid #3b6cff;border-radius:2px;box-sizing:border-box;"></span><span style="width:24px;height:4px;background:#3b6cff;border-radius:2px;"></span></div>',
    Switch:
      '<div style="width:60px;height:40px;border:1px solid #3b6cff;border-radius:4px;background:#fff;box-sizing:border-box;display:flex;align-items:center;justify-content:center;"><span style="width:34px;height:16px;background:#3b6cff;border-radius:12px;display:flex;align-items:center;justify-content:flex-end;padding:2px;box-sizing:border-box;"><span style="width:12px;height:12px;background:#fff;border-radius:50%;"></span></span></div>',
    Table:
      '<div style="width:60px;height:40px;border:1px solid #3b6cff;border-radius:4px;background:#fff;box-sizing:border-box;display:flex;align-items:center;justify-content:center;"><span style="width:42px;height:24px;border:1px solid #3b6cff;border-radius:2px;display:grid;grid-template-rows:repeat(3,1fr);box-sizing:border-box;overflow:hidden;"><span style="background:#3b6cff;"></span><span style="border-top:1px solid #3b6cff;"></span><span style="border-top:1px solid #3b6cff;"></span></span></div>',
    Pagination:
      '<div style="width:60px;height:40px;border:1px solid #3b6cff;border-radius:4px;background:#fff;box-sizing:border-box;display:flex;align-items:center;justify-content:center;gap:3px;"><span style="width:8px;height:8px;border:1px solid #3b6cff;border-radius:2px;"></span><span style="width:10px;height:10px;background:#3b6cff;border-radius:2px;"></span><span style="width:8px;height:8px;border:1px solid #3b6cff;border-radius:2px;"></span></div>',
    LanguageSwitcher:
      '<div style="width:60px;height:40px;border:1px solid #3b6cff;border-radius:4px;display:flex;align-items:center;justify-content:center;background:#fff;color:#3b6cff;box-sizing:border-box;"><span style="width:42px;height:16px;background:#3b6cff;border-radius:2px;display:flex;align-items:center;justify-content:center;gap:5px;"><span style="width:22px;height:4px;background:#fff;border-radius:2px;"></span><span style="width:0;height:0;border-left:4px solid transparent;border-right:4px solid transparent;border-top:5px solid #fff;"></span></span></div>',
    UserAvatarMenu:
      '<div style="width:60px;height:40px;border:1px solid #3b6cff;border-radius:4px;display:flex;align-items:center;justify-content:center;background:#fff;box-sizing:border-box;"><span style="width:44px;height:18px;border:1px solid #3b6cff;border-radius:12px;display:flex;align-items:center;gap:4px;padding:2px 4px;box-sizing:border-box;"><span style="width:10px;height:10px;border-radius:50%;background:#3b6cff;"></span><span style="width:16px;height:4px;background:#3b6cff;border-radius:2px;"></span><span style="width:0;height:0;border-left:3px solid transparent;border-right:3px solid transparent;border-top:4px solid #3b6cff;"></span></span></div>',
    HorizontalLayout:
      '<div style="width:60px;height:40px;border:1px solid #3b6cff;border-radius:4px;display:flex;gap:4px;padding:4px;box-sizing:border-box;"><div style="flex:1;background:#3b6cff;border-radius:2px;"></div><div style="flex:1;background:#3b6cff;border-radius:2px;"></div><div style="flex:1;background:#3b6cff;border-radius:2px;"></div></div>',
    VerticalLayout:
      '<div style="width:60px;height:40px;border:1px solid #3b6cff;border-radius:4px;display:flex;flex-direction:column;gap:4px;padding:4px;box-sizing:border-box;"><div style="height:6px;background:#3b6cff;border-radius:2px;"></div><div style="height:6px;background:#3b6cff;border-radius:2px;"></div><div style="height:6px;background:#3b6cff;border-radius:2px;"></div></div>',
    FormLayout:
      '<div style="width:60px;height:40px;border:1px solid #3b6cff;border-radius:4px;display:flex;flex-direction:column;gap:4px;padding:4px;box-sizing:border-box;"><div style="height:6px;border:1px solid #3b6cff;border-radius:2px;"></div><div style="height:6px;border:1px solid #3b6cff;border-radius:2px;"></div><div style="height:6px;border:1px solid #3b6cff;border-radius:2px;"></div></div>',
    ElContainer:
      '<div style="width:60px;height:40px;border:1px solid #3b6cff;border-radius:4px;display:flex;flex-direction:column;gap:3px;padding:4px;box-sizing:border-box;"><div style="height:5px;background:#3b6cff;border-radius:2px;"></div><div style="flex:1;display:flex;gap:3px;"><div style="width:10px;background:#3b6cff;border-radius:2px;"></div><div style="flex:1;background:#e4e7ed;border-radius:2px;"></div></div><div style="height:5px;background:#3b6cff;border-radius:2px;"></div></div>',
    Tabs: '<div style="width:60px;height:40px;border:1px solid #3b6cff;border-radius:4px;box-sizing:border-box;"><div style="height:9px;background:#3b6cff;"></div><div style="height:18px;margin:5px;border:1px solid #3b6cff;border-radius:2px;"></div></div>',
    Collapse:
      '<div style="width:60px;height:40px;border:1px solid #3b6cff;border-radius:4px;box-sizing:border-box;"><div style="height:9px;background:#3b6cff;"></div><div style="height:16px;margin:5px;border:1px solid #3b6cff;border-radius:2px;"></div></div>',
  }
  return (
    iconMap[type] ||
    '<div style="width:60px;height:40px;border:1px dashed #c0c4cc;border-radius:4px;box-sizing:border-box;"></div>'
  )
}

function handleDragStart(item: ComponentItemLike, event: DragEvent): void {
  startDrag(item.type)
  if (!event.dataTransfer) return

  const payload = JSON.stringify({ type: item.type })
  event.dataTransfer.effectAllowed = 'copy'
  event.dataTransfer.setData('application/x-designer-component', payload)
  event.dataTransfer.setData('text/plain', item.type)

  const dragPreview = document.createElement('div')
  dragPreview.innerHTML = `
    <div style="display:flex;flex-direction:column;align-items:center;gap:8px;padding:10px;background:#fff;border:1px solid #409eff;border-radius:8px;box-shadow:0 4px 12px rgba(0,0,0,.15);">
      <div>${getPreviewIcon(item.type)}</div>
      <div style="font-size:12px;color:#303133;">${item.name}</div>
    </div>
  `
  dragPreview.style.position = 'absolute'
  dragPreview.style.top = '-1000px'
  dragPreview.style.left = '-1000px'
  dragPreview.style.pointerEvents = 'none'
  document.body.appendChild(dragPreview)
  event.dataTransfer.setDragImage(dragPreview, 60, 30)
  setTimeout(() => {
    if (dragPreview.parentNode) dragPreview.parentNode.removeChild(dragPreview)
  }, 0)
}

function handlePointerStart(item: ComponentItemLike, event: MouseEvent): void {
  if (event.button !== 0) return
  startDrag(item.type)
}

function handleDragEnd(): void {
  endDrag()
}
</script>

<template>
  <div class="component-panel">
    <el-input
      v-model="keyword"
      size="small"
      :placeholder="t('componentPanel.searchPlaceholder')"
      clearable
    />
    <div class="component-list">
      <el-collapse v-model="activeSections" class="component-collapse">
        <el-collapse-item name="layout">
          <template #title>
            <span class="component-section-title">{{ t('componentPanel.layout') }}</span>
          </template>
          <div v-if="layoutItems.length" class="component-grid">
            <div
              v-for="item in layoutItems"
              :key="item.type"
              class="component-card"
              draggable="true"
              @mousedown="handlePointerStart(item, $event)"
              @dragstart="handleDragStart(item, $event)"
              @dragend="handleDragEnd"
            >
              <div class="card-preview">
                <component :is="getPreviewComponent(item.type)" />
              </div>
              <div class="card-name">{{ item.name }}</div>
            </div>
          </div>
          <div v-else class="empty-tip">{{ t('componentPanel.emptyLayout') }}</div>
        </el-collapse-item>

        <el-collapse-item v-if="showSystemSection" name="system">
          <template #title>
            <span class="component-section-title">{{ t('componentPanel.system') }}</span>
          </template>
          <div v-if="systemItems.length" class="component-grid">
            <div
              v-for="item in systemItems"
              :key="item.type"
              class="component-card"
              draggable="true"
              @mousedown="handlePointerStart(item, $event)"
              @dragstart="handleDragStart(item, $event)"
              @dragend="handleDragEnd"
            >
              <div class="card-preview">
                <component :is="getPreviewComponent(item.type)" />
              </div>
              <div class="card-name">{{ item.name }}</div>
            </div>
          </div>
          <div v-else class="empty-tip">{{ t('componentPanel.emptySystem') }}</div>
        </el-collapse-item>

        <el-collapse-item name="ui">
          <template #title>
            <span class="component-section-title">{{ t('componentPanel.ui') }}</span>
          </template>
          <div v-if="uiItems.length" class="component-grid">
            <div
              v-for="item in uiItems"
              :key="item.type"
              class="component-card"
              draggable="true"
              @mousedown="handlePointerStart(item, $event)"
              @dragstart="handleDragStart(item, $event)"
              @dragend="handleDragEnd"
            >
              <div class="card-preview">
                <component :is="getPreviewComponent(item.type)" />
              </div>
              <div class="card-name">{{ item.name }}</div>
            </div>
          </div>
          <div v-else class="empty-tip">{{ t('componentPanel.emptyUi') }}</div>
        </el-collapse-item>
      </el-collapse>
    </div>
  </div>
</template>

<style>
.component-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 0;
  overflow: hidden;
  background: transparent;
}

.component-list {
  flex: 1;
  overflow: auto;
  padding: 0 8px 10px;
}

.component-panel > .el-input {
  padding: 8px;
  background: var(--designer-shell-surface);
  border-bottom: 1px solid var(--designer-border-color);
}

.component-panel > .el-input .el-input__wrapper {
  border-radius: 999px;
  background: var(--designer-group-surface);
  box-shadow: none;
}

.component-collapse {
  border: none;
  background: transparent;
}

.component-collapse .el-collapse-item__header {
  background: transparent;
  border: none;
  border-bottom: 1px solid var(--designer-border-color);
  border-radius: 0;
  padding: 0;
  margin: 0;
  height: 38px;
  font-size: 13px;
  color: var(--designer-text-primary);
}

.component-collapse .el-collapse-item__wrap {
  border: none;
  background: transparent;
}

.component-collapse .el-collapse-item__content {
  padding: 8px 0 14px;
}

.component-section-title {
  font-weight: 600;
}

.component-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, 76px);
  gap: 6px;
}

.component-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  min-width: 0;
  min-height: 74px;
  border: 1px solid transparent;
  border-radius: 6px;
  overflow: hidden;
  cursor: grab;
  transition:
    border-color 0.16s ease,
    background-color 0.16s ease,
    color 0.16s ease;
  background: transparent;
}

.component-card:hover {
  border-color: var(--designer-border-color);
  background: var(--designer-hover-surface);
}

.component-card:active {
  cursor: grabbing;
}

.card-preview {
  height: 44px;
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  padding: 6px 4px 2px;
}

.card-name {
  width: 100%;
  padding: 2px 4px 7px;
  text-align: center;
  font-size: 12px;
  font-weight: 500;
  line-height: 1.25;
  color: var(--designer-text-regular);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.empty-tip {
  color: var(--designer-text-muted);
  font-size: 12px;
  text-align: center;
  padding: 16px 0;
}

.preview-button {
  width: 44px;
  height: 32px;
  border: 1.5px solid var(--designer-material-icon-color, #888d92);
  border-radius: 4px;
  box-sizing: border-box;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  color: var(--designer-material-icon-color, #888d92);
  background: transparent;
}

.preview-button-line {
  width: 28px;
  height: 8px;
  background: var(--designer-material-icon-color, #888d92);
  border-radius: 2px;
}

.preview-input,
.preview-input-number,
.preview-select {
  width: 44px;
  height: 32px;
  border: 1.5px solid var(--designer-material-icon-color, #888d92);
  border-radius: 4px;
  box-sizing: border-box;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  background: transparent;
}

.preview-input-line,
.preview-select-line {
  width: 27px;
  height: 4px;
  background: var(--designer-material-icon-color, #888d92);
  border-radius: 2px;
}

.preview-input-number-line {
  width: 20px;
  height: 4px;
  background: var(--designer-material-icon-color, #888d92);
  border-radius: 2px;
}

.preview-input-number-step {
  width: 8px;
  height: 16px;
  border-radius: 2px;
  background: var(--designer-material-icon-color, #888d92);
}

.preview-select-arrow {
  width: 0;
  height: 0;
  border-left: 4px solid transparent;
  border-right: 4px solid transparent;
  border-top: 5px solid var(--designer-material-icon-color, #888d92);
}

.preview-choice {
  width: 44px;
  height: 32px;
  border: 1.5px solid var(--designer-material-icon-color, #888d92);
  border-radius: 4px;
  box-sizing: border-box;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  background: transparent;
}

.preview-radio-dot {
  width: 12px;
  height: 12px;
  border: 2px solid var(--designer-material-icon-color, #888d92);
  border-radius: 50%;
  box-sizing: border-box;
}

.preview-checkbox-box {
  width: 12px;
  height: 12px;
  border: 2px solid var(--designer-material-icon-color, #888d92);
  border-radius: 2px;
  box-sizing: border-box;
}

.preview-choice-line {
  width: 20px;
  height: 4px;
  background: var(--designer-material-icon-color, #888d92);
  border-radius: 2px;
}

.preview-switch {
  width: 44px;
  height: 32px;
  border: 1.5px solid var(--designer-material-icon-color, #888d92);
  border-radius: 4px;
  box-sizing: border-box;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
}

.preview-switch::before {
  content: '';
  width: 32px;
  height: 16px;
  border-radius: 12px;
  background: var(--designer-material-icon-color, #888d92);
}

.preview-switch-dot {
  position: absolute;
  width: 12px;
  height: 12px;
  margin-left: 14px;
  border-radius: 50%;
  background: var(--designer-shell-surface);
}

.preview-table {
  width: 44px;
  height: 32px;
  border: 1.5px solid var(--designer-material-icon-color, #888d92);
  border-radius: 4px;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  gap: 3px;
  padding: 5px;
  background: transparent;
}

.preview-table-row {
  flex: 1;
  border: 1px solid var(--designer-material-icon-color, #888d92);
  border-radius: 2px;
}

.preview-table-row.is-head {
  background: var(--designer-material-icon-color, #888d92);
}

.preview-pagination {
  width: 44px;
  height: 32px;
  border: 1.5px solid var(--designer-material-icon-color, #888d92);
  border-radius: 4px;
  box-sizing: border-box;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 3px;
  background: transparent;
}

.preview-pagination span {
  width: 8px;
  height: 8px;
  border: 1px solid var(--designer-material-icon-color, #888d92);
  border-radius: 2px;
}

.preview-pagination .is-active {
  background: var(--designer-material-icon-color, #888d92);
}

.preview-language-switcher {
  width: 44px;
  height: 32px;
  border: 1.5px solid var(--designer-material-icon-color, #888d92);
  border-radius: 4px;
  box-sizing: border-box;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--designer-material-icon-color, #888d92);
  background: transparent;
}

.preview-language-content {
  width: 32px;
  height: 13px;
  background: var(--designer-material-icon-color, #888d92);
  border-radius: 2px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
}

.preview-language-line {
  width: 17px;
  height: 3px;
  background: var(--designer-shell-surface);
  border-radius: 2px;
}

.preview-language-arrow {
  width: 0;
  height: 0;
  border-left: 4px solid transparent;
  border-right: 4px solid transparent;
  border-top: 5px solid var(--designer-shell-surface);
  flex: 0 0 auto;
}

.preview-user-avatar {
  width: 44px;
  height: 32px;
  border: 1.5px solid var(--designer-material-icon-color, #888d92);
  border-radius: 4px;
  box-sizing: border-box;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 3px;
  background: transparent;
}

.preview-user-avatar-circle {
  width: 12px;
  height: 12px;
  flex: 0 0 auto;
  border-radius: 50%;
  background: var(--designer-material-icon-color, #888d92);
}

.preview-user-avatar-lines {
  display: flex;
  width: 16px;
  flex: 0 0 auto;
  flex-direction: column;
  gap: 3px;
}

.preview-user-avatar-line,
.preview-user-avatar-subline {
  height: 3px;
  border-radius: 2px;
  background: var(--designer-material-icon-color, #888d92);
}

.preview-user-avatar-subline {
  width: 11px;
  opacity: 0.62;
}

.preview-user-avatar-arrow {
  width: 0;
  height: 0;
  flex: 0 0 auto;
  border-top: 4px solid var(--designer-material-icon-color, #888d92);
  border-right: 3px solid transparent;
  border-left: 3px solid transparent;
}

.preview-flex {
  width: 44px;
  height: 32px;
  border: 1.5px solid var(--designer-material-icon-color, #888d92);
  border-radius: 4px;
  display: flex;
  gap: 3px;
  padding: 3px;
  background: transparent;
}

.preview-flex-column {
  flex-direction: column;
}

.preview-block {
  flex: 1;
  background: var(--designer-material-icon-color, #888d92);
  border-radius: 2px;
}

.preview-form {
  width: 44px;
  height: 32px;
  border: 1.5px solid var(--designer-material-icon-color, #888d92);
  border-radius: 4px;
  display: flex;
  flex-direction: column;
  gap: 3px;
  padding: 3px;
  background: transparent;
}

.preview-form-item {
  height: 5px;
  border: 1px solid var(--designer-material-icon-color, #888d92);
  border-radius: 2px;
}

.preview-el-container {
  width: 44px;
  height: 32px;
  border: 1.5px solid var(--designer-material-icon-color, #888d92);
  border-radius: 4px;
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 3px;
  background: transparent;
}

.preview-el-header,
.preview-el-footer {
  height: 4px;
  background: var(--designer-material-icon-color, #888d92);
  border-radius: 2px;
}

.preview-el-body {
  flex: 1;
  display: flex;
  gap: 2px;
}

.preview-el-aside {
  width: 8px;
  background: var(--designer-material-icon-color, #888d92);
  border-radius: 2px;
}

.preview-el-main {
  flex: 1;
  background: var(--designer-border-color);
  border-radius: 2px;
}

.preview-tabs {
  width: 44px;
  height: 32px;
  border: 1.5px solid var(--designer-material-icon-color, #888d92);
  border-radius: 4px;
  padding: 3px;
  background: transparent;
}

.preview-tabs-header {
  height: 6px;
  background: var(--designer-material-icon-color, #888d92);
  border-radius: 2px;
}

.preview-tabs-body {
  margin-top: 3px;
  height: 17px;
  border: 1px solid var(--designer-material-icon-color, #888d92);
  border-radius: 2px;
}

.preview-collapse {
  width: 44px;
  height: 32px;
  border: 1.5px solid var(--designer-material-icon-color, #888d92);
  border-radius: 4px;
  background: transparent;
}

.preview-collapse-header {
  height: 6px;
  background: var(--designer-material-icon-color, #888d92);
  border-radius: 4px 4px 0 0;
}

.preview-collapse-body {
  height: 12px;
  margin: 4px;
  border: 1px solid var(--designer-material-icon-color, #888d92);
  border-radius: 2px;
}

.preview-unknown {
  font-size: 10px;
  color: var(--designer-text-muted);
}
</style>
