<!--
  Tabs 专用属性编辑器。
  标签内容仍在画布中维护，这里只处理标签页结构和默认激活项。
-->
<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, ref, watch } from 'vue'
import {
  buildTabsChildKeyPatches,
  canRemoveTabsItem,
  createTabsItem,
  normalizeTabsItems,
  normalizeTabsModelValue,
  renameTabsItem,
  type TabsChildKeyPatch,
  type TabsChildLike,
  type TabsPanelItem,
} from './tabs-panel-utils'
import ListItemsEditor from './ListItemsEditor.vue'

const props = defineProps<{
  nodeProps: Record<string, unknown>
  childNodes: TabsChildLike[]
}>()

const emit = defineEmits<{
  (event: 'propsChange', patch: Record<string, unknown>): void
  (event: 'childPatches', patches: TabsChildKeyPatch[]): void
}>()

const selectedKey = ref('')
const itemsDialogVisible = ref(false)

const items = computed<TabsPanelItem[]>(() => normalizeTabsItems(props.nodeProps?.tabs))
const activeValue = computed<string>(() =>
  normalizeTabsModelValue(props.nodeProps?.modelValue ?? props.nodeProps?.activeName, items.value),
)
const selectedItem = computed<TabsPanelItem | null>(
  () => items.value.find((item) => item.name === selectedKey.value) || items.value[0] || null,
)
const itemOptions = computed(() =>
  items.value.map((item) => ({
    label: item.label || item.name,
    value: item.name,
  })),
)

watch(
  items,
  (nextItems) => {
    if (nextItems.length === 0) {
      selectedKey.value = ''
      return
    }
    if (!nextItems.some((item) => item.name === selectedKey.value)) {
      selectedKey.value = nextItems[0]!.name
    }
  },
  { immediate: true },
)

function emitProps(nextProps: Record<string, unknown>) {
  emit('propsChange', nextProps)
}

function emitItems(nextItems: TabsPanelItem[], extraProps: Record<string, unknown> = {}) {
  const normalizedItems = normalizeTabsItems(nextItems)
  const nextActive = normalizeTabsModelValue(
    extraProps.modelValue ??
      extraProps.activeName ??
      props.nodeProps?.modelValue ??
      props.nodeProps?.activeName,
    normalizedItems,
  )
  emitProps({
    ...extraProps,
    tabs: normalizedItems,
    activeName: nextActive,
    modelValue: nextActive,
  })
}

function handleActiveChange(value: string) {
  const nextActive = normalizeTabsModelValue(value, items.value)
  emitProps({ activeName: nextActive, modelValue: nextActive })
}

function handleAddItem() {
  const nextItem = createTabsItem(items.value)
  selectedKey.value = nextItem.name
  emitItems([...items.value, nextItem], { modelValue: activeValue.value || nextItem.name })
}

function handleDuplicateItem(key: string) {
  const source = items.value.find((item) => item.name === key)
  if (!source) return
  const baseItem = createTabsItem(items.value, `${source.label || source.name} 副本`)
  const nextItem = {
    ...source,
    name: baseItem.name,
    label: baseItem.label,
  }
  selectedKey.value = nextItem.name
  emitItems([...items.value, nextItem])
}

function handleRemoveItem(key: string) {
  const removeCheck = canRemoveTabsItem(key, props.childNodes || [])
  if (!removeCheck.ok) {
    ElMessage.warning({ message: '该标签页中已有画布内容，请先在画布中移动或删除内容' } as never)
    return
  }
  if (items.value.length <= 1) {
    ElMessage.warning({ message: '至少保留一个标签页' } as never)
    return
  }
  const nextItems = items.value.filter((item) => item.name !== key)
  selectedKey.value = nextItems[0]?.name || ''
  emitItems(nextItems, {
    modelValue: normalizeTabsModelValue(
      props.nodeProps?.modelValue ?? props.nodeProps?.activeName,
      nextItems,
    ),
  })
}

function handleMoveItem(key: string, direction: -1 | 1) {
  const index = items.value.findIndex((item) => item.name === key)
  const nextIndex = index + direction
  if (index < 0 || nextIndex < 0 || nextIndex >= items.value.length) return
  const nextItems = [...items.value]
  const [item] = nextItems.splice(index, 1)
  if (!item) return
  nextItems.splice(nextIndex, 0, item)
  emitItems(nextItems)
}

function updateSelectedItem(patch: Partial<TabsPanelItem>) {
  const current = selectedItem.value
  if (!current) return
  const nextItems = items.value.map((item) =>
    item.name === current.name ? { ...item, ...patch } : item,
  )
  emitItems(nextItems)
}

function handleNameChange(value: string) {
  const current = selectedItem.value
  if (!current) return
  const result = renameTabsItem(items.value, current.name, value)
  if (!result.ok) {
    const messageMap: Record<string, string> = {
      emptyName: '唯一标识不能为空',
      duplicateName: '唯一标识不能重复',
      missingItem: '当前标签页不存在',
    }
    ElMessage.warning({ message: messageMap[result.reason] || '标识更新失败' } as never)
    return
  }
  const patches = buildTabsChildKeyPatches(props.childNodes || [], result.oldName, result.newName)
  const nextActive = activeValue.value === result.oldName ? result.newName : activeValue.value
  selectedKey.value = result.newName
  emit('childPatches', patches)
  emitItems(result.items, { modelValue: nextActive })
}

function openItemsDialog() {
  if (!selectedKey.value && items.value[0]) {
    selectedKey.value = items.value[0].name
  }
  itemsDialogVisible.value = true
}
</script>

<template>
  <div class="tabs-items-editor">
    <div class="prop-section">
      <div class="prop-section-header is-static">
        <span class="prop-section-title">激活状态</span>
      </div>
      <div class="prop-section-body">
        <div class="prop-item">
          <div class="prop-label">默认激活</div>
          <el-select
            :model-value="activeValue"
            size="small"
            placeholder="请选择默认激活标签"
            @update:model-value="(val: any) => handleActiveChange(String(val || ''))"
          >
            <el-option
              v-for="item in itemOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </div>
      </div>
    </div>

    <div class="prop-section">
      <div class="prop-section-header is-static">
        <span class="prop-section-title">标签页</span>
      </div>
      <div class="prop-section-body">
        <el-tooltip content="标题、标识、禁用和排序在弹窗中配置" placement="top">
          <button class="tabs-items-entry" type="button" @click="openItemsDialog">
            配置标签页
          </button>
        </el-tooltip>
      </div>
    </div>

    <el-dialog
      v-model="itemsDialogVisible"
      title="配置标签页"
      width="720px"
      top="8vh"
      append-to-body
      :close-on-click-modal="false"
      :lock-scroll="false"
    >
      <div class="tabs-items-dialog">
        <ListItemsEditor
          :items="items"
          :selected-key="selectedKey"
          title="标签页"
          @select="(key) => (selectedKey = key)"
          @add="handleAddItem"
          @duplicate="handleDuplicateItem"
          @remove="handleRemoveItem"
          @move="handleMoveItem"
        >
          <template v-if="selectedItem">
            <div class="prop-item">
              <div class="prop-label">标题</div>
              <el-input
                :model-value="selectedItem.label"
                size="small"
                placeholder="请输入标签标题"
                @update:model-value="(val: any) => updateSelectedItem({ label: String(val || '') })"
              />
            </div>
            <div class="prop-item">
              <div class="prop-label">唯一标识</div>
              <el-input
                :model-value="selectedItem.name"
                size="small"
                placeholder="如 tab1"
                @change="(val: any) => handleNameChange(String(val || ''))"
              />
            </div>
            <div class="prop-item">
              <div class="prop-label">禁用</div>
              <el-switch
                :model-value="selectedItem.disabled"
                size="small"
                @update:model-value="(val: any) => updateSelectedItem({ disabled: Boolean(val) })"
              />
            </div>
            <div class="prop-item prop-item--textarea">
              <div class="prop-label">空内容提示</div>
              <el-input
                :model-value="selectedItem.content"
                type="textarea"
                :rows="2"
                size="small"
                placeholder="标签页没有画布内容时显示"
                @update:model-value="
                  (val: any) => updateSelectedItem({ content: String(val || '') })
                "
              />
            </div>
          </template>
        </ListItemsEditor>
      </div>
      <template #footer>
        <el-button type="primary" @click="itemsDialogVisible = false">完成</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.tabs-items-editor {
  display: flex;
  flex-direction: column;
  gap: var(--designer-gap-xs);
}

.prop-section {
  overflow: hidden;
  border: 1px solid var(--designer-border-color);
  border-radius: var(--designer-radius-md);
  background: var(--designer-shell-surface);
}

.prop-section-header {
  display: flex;
  align-items: center;
  min-height: 30px;
  padding: 0 10px;
  border-bottom: 1px solid var(--designer-border-soft);
  background: var(--designer-group-surface);
}

.prop-section-title {
  color: var(--designer-text-secondary);
  font-size: var(--designer-font-label);
  font-weight: 600;
}

.prop-section-body {
  display: flex;
  flex-direction: column;
  gap: var(--designer-gap-xs);
  padding: 4px 6px;
}

.prop-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
  min-height: 28px;
  padding: 2px 4px;
  border-radius: var(--designer-radius-sm);
}

.prop-item:hover {
  background: var(--designer-hover-surface);
}

.prop-item > :last-child:not(.prop-label) {
  flex: 1;
  min-width: 0;
}

.prop-item :deep(.el-input),
.prop-item :deep(.el-select),
.prop-item :deep(.el-textarea) {
  width: 100%;
}

.prop-item :deep(.el-switch) {
  margin-left: auto;
}

.prop-item--textarea {
  align-items: flex-start;
}

.prop-label {
  display: flex;
  align-items: center;
  flex-shrink: 0;
  width: 88px;
  min-width: 72px;
  max-width: 88px;
  font-size: var(--designer-font-label);
  color: var(--designer-text-regular);
}

.tabs-items-entry {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--designer-gap-xs);
  width: 100%;
  height: 28px;
  padding: 0 10px;
  box-sizing: border-box;
  border: 1px solid var(--designer-border-color);
  border-radius: var(--designer-radius-md);
  background: var(--designer-group-surface);
  color: var(--designer-text-secondary);
  font-size: var(--designer-font-label);
  cursor: pointer;
  transition:
    border-color 0.15s ease,
    background-color 0.15s ease,
    color 0.15s ease;
}

.tabs-items-entry:hover {
  border-color: var(--designer-primary-border);
  background: var(--designer-primary-soft);
  color: var(--designer-primary-text);
}

.tabs-items-dialog {
  max-height: 62vh;
  overflow: auto;
}
</style>
