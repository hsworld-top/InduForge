<!--
  Collapse 专用属性编辑器。
  面板内容仍在画布中维护，这里只处理面板项结构、默认展开和基础行为。
-->
<script setup lang="ts">
import { ElMessage } from "element-plus";
import { computed, ref, watch } from "vue";
import {
  buildCollapseChildKeyPatches,
  canRemoveCollapseItem,
  createCollapseItem,
  normalizeCollapseItems,
  normalizeCollapseModelValue,
  renameCollapseItem,
  type CollapseChildKeyPatch,
  type CollapseChildLike,
  type CollapsePanelItem,
} from "./collapse-panel-utils";
import ListItemsEditor from "./ListItemsEditor.vue";

const props = defineProps<{
  nodeProps: Record<string, unknown>;
  childNodes: CollapseChildLike[];
}>();

const emit = defineEmits<{
  (event: "propsChange", patch: Record<string, unknown>): void;
  (event: "childPatches", patches: CollapseChildKeyPatch[]): void;
}>();

const selectedKey = ref("");
const itemsDialogVisible = ref(false);

const items = computed<CollapsePanelItem[]>(() => normalizeCollapseItems(props.nodeProps?.items));
const accordion = computed<boolean>(() => Boolean(props.nodeProps?.accordion));
const normalizedModelValue = computed(() =>
  normalizeCollapseModelValue(props.nodeProps?.modelValue, items.value, accordion.value),
);
const selectedItem = computed<CollapsePanelItem | null>(
  () => items.value.find((item) => item.name === selectedKey.value) || items.value[0] || null,
);
const activeSingleValue = computed<string>(() => {
  const value = normalizedModelValue.value;
  return Array.isArray(value) ? String(value[0] || "") : String(value || "");
});
const activeMultiValue = computed<string[]>(() => {
  const value = normalizedModelValue.value;
  return Array.isArray(value) ? value : value ? [String(value)] : [];
});
const itemOptions = computed(() =>
  items.value.map((item) => ({
    label: item.title || item.name,
    value: item.name,
  })),
);

watch(
  items,
  (nextItems) => {
    if (nextItems.length === 0) {
      selectedKey.value = "";
      return;
    }
    if (!nextItems.some((item) => item.name === selectedKey.value)) {
      selectedKey.value = nextItems[0]!.name;
    }
  },
  { immediate: true },
);

function emitProps(nextProps: Record<string, unknown>) {
  emit("propsChange", nextProps);
}

function emitItems(nextItems: CollapsePanelItem[], extraProps: Record<string, unknown> = {}) {
  emitProps({
    ...extraProps,
    items: normalizeCollapseItems(nextItems),
  });
}

function handleAccordionChange(value: boolean) {
  emitProps({
    accordion: value,
    modelValue: normalizeCollapseModelValue(props.nodeProps?.modelValue, items.value, value),
  });
}

function handleSingleActiveChange(value: string) {
  emitProps({ modelValue: normalizeCollapseModelValue(value, items.value, true) });
}

function handleMultiActiveChange(value: string[]) {
  emitProps({ modelValue: normalizeCollapseModelValue(value, items.value, false) });
}

function handleAddItem() {
  const nextItem = createCollapseItem(items.value);
  selectedKey.value = nextItem.name;
  emitItems([...items.value, nextItem], {
    modelValue: normalizeCollapseModelValue(props.nodeProps?.modelValue, [...items.value, nextItem], accordion.value),
  });
}

function handleDuplicateItem(key: string) {
  const source = items.value.find((item) => item.name === key);
  if (!source) return;
  const baseItem = createCollapseItem(items.value, `${source.title || source.name} 副本`);
  const nextItem = {
    ...source,
    name: baseItem.name,
    title: baseItem.title,
  };
  selectedKey.value = nextItem.name;
  emitItems([...items.value, nextItem]);
}

function handleRemoveItem(key: string) {
  const removeCheck = canRemoveCollapseItem(key, props.childNodes || []);
  if (!removeCheck.ok) {
    ElMessage.warning({ message: "该面板项中已有画布内容，请先在画布中移动或删除内容" } as never);
    return;
  }
  if (items.value.length <= 1) {
    ElMessage.warning({ message: "至少保留一个面板项" } as never);
    return;
  }
  const nextItems = items.value.filter((item) => item.name !== key);
  const nextModelValue = normalizeCollapseModelValue(
    props.nodeProps?.modelValue,
    nextItems,
    accordion.value,
  );
  selectedKey.value = nextItems[0]?.name || "";
  emitItems(nextItems, { modelValue: nextModelValue });
}

function handleMoveItem(key: string, direction: -1 | 1) {
  const index = items.value.findIndex((item) => item.name === key);
  const nextIndex = index + direction;
  if (index < 0 || nextIndex < 0 || nextIndex >= items.value.length) return;
  const nextItems = [...items.value];
  const [item] = nextItems.splice(index, 1);
  if (!item) return;
  nextItems.splice(nextIndex, 0, item);
  emitItems(nextItems);
}

function updateSelectedItem(patch: Partial<CollapsePanelItem>) {
  const current = selectedItem.value;
  if (!current) return;
  const nextItems = items.value.map((item) =>
    item.name === current.name ? { ...item, ...patch } : item,
  );
  emitItems(nextItems);
}

function handleNameChange(value: string) {
  const current = selectedItem.value;
  if (!current) return;
  const result = renameCollapseItem(items.value, current.name, value);
  if (!result.ok) {
    const messageMap: Record<string, string> = {
      emptyName: "唯一标识不能为空",
      duplicateName: "唯一标识不能重复",
      missingItem: "当前面板项不存在",
    };
    ElMessage.warning({ message: messageMap[result.reason] || "标识更新失败" } as never);
    return;
  }
  const nextModelValue = normalizeCollapseModelValue(
    activeMultiValue.value.map((item) => (item === result.oldName ? result.newName : item)),
    result.items,
    accordion.value,
  );
  const patches = buildCollapseChildKeyPatches(props.childNodes || [], result.oldName, result.newName);
  selectedKey.value = result.newName;
  emit("childPatches", patches);
  emitItems(result.items, { modelValue: nextModelValue });
}

function openItemsDialog() {
  if (!selectedKey.value && items.value[0]) {
    selectedKey.value = items.value[0].name;
  }
  itemsDialogVisible.value = true;
}
</script>

<template>
  <div class="collapse-items-editor">
    <div class="prop-section">
      <div class="prop-section-header is-static">
        <span class="prop-section-title">展开行为</span>
      </div>
      <div class="prop-section-body">
        <div class="prop-item">
          <div class="prop-label">手风琴</div>
          <el-switch
            :model-value="accordion"
            size="small"
            @update:model-value="(val: any) => handleAccordionChange(Boolean(val))"
          />
        </div>
        <div class="prop-item">
          <div class="prop-label">默认展开</div>
          <el-select
            v-if="accordion"
            :model-value="activeSingleValue"
            size="small"
            clearable
            placeholder="不默认展开"
            @update:model-value="(val: any) => handleSingleActiveChange(String(val || ''))"
          >
            <el-option
              v-for="item in itemOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
          <el-select
            v-else
            :model-value="activeMultiValue"
            size="small"
            multiple
            collapse-tags
            collapse-tags-tooltip
            placeholder="不默认展开"
            @update:model-value="(val: any) => handleMultiActiveChange(Array.isArray(val) ? val : [])"
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
        <span class="prop-section-title">面板项</span>
      </div>
      <div class="prop-section-body">
        <el-tooltip content="标题、标识、禁用和排序在弹窗中配置" placement="top">
          <button class="collapse-items-entry" type="button" @click="openItemsDialog">
            配置面板项
          </button>
        </el-tooltip>
      </div>
    </div>

    <el-dialog
      v-model="itemsDialogVisible"
      title="配置折叠面板项"
      width="720px"
      top="8vh"
      append-to-body
      :close-on-click-modal="false"
      :lock-scroll="false"
    >
      <div class="collapse-items-dialog">
        <ListItemsEditor
          :items="items"
          :selected-key="selectedKey"
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
                :model-value="selectedItem.title"
                size="small"
                placeholder="请输入面板标题"
                @update:model-value="(val: any) => updateSelectedItem({ title: String(val || '') })"
              />
            </div>
            <div class="prop-item">
              <div class="prop-label">唯一标识</div>
              <el-input
                :model-value="selectedItem.name"
                size="small"
                placeholder="如 panel1"
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
                placeholder="面板没有画布内容时显示"
                @update:model-value="(val: any) => updateSelectedItem({ content: String(val || '') })"
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
.collapse-items-editor {
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

.collapse-items-entry {
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

.collapse-items-entry:hover {
  border-color: var(--designer-primary-border);
  background: var(--designer-primary-soft);
  color: var(--designer-primary-text);
}

.collapse-items-dialog {
  max-height: 62vh;
  overflow: auto;
}
</style>
