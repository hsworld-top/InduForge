<template>
  <div class="flex flex-col gap-3 style-panel">
    <template v-if="hasSelection">
      <el-collapse v-model="activeNames">
        <!-- 基础样式 -->
        <el-collapse-item title="基础样式" name="basic">
          <template v-if="showSizeEditor">
            <SizeEditor
              :model-value="currentStyle"
              :min-width="containerMinSize?.width"
              :min-height="containerMinSize?.height"
              @update:model-value="handleStyleChange"
            />
            <el-divider style="margin: 12px 0" />
          </template>
          <BackgroundEditor
            :model-value="currentStyle"
            @update:model-value="handleStyleChange"
          />
          <el-divider style="margin: 12px 0" />
          <BorderEditor
            :model-value="currentStyle"
            @update:model-value="handleStyleChange"
          />
        </el-collapse-item>

        <!-- 布局样式 -->
        <el-collapse-item title="布局样式" name="layout">
          <PositionEditor
            :model-value="currentStyle"
            @update:model-value="handleStyleChange"
          />
          <el-divider style="margin: 12px 0" />
          <SpacingEditor
            title="内边距"
            prefix="padding"
            :model-value="currentStyle"
            @update:model-value="handleStyleChange"
          />
          <el-divider style="margin: 12px 0" />
          <SpacingEditor
            title="外边距"
            prefix="margin"
            :model-value="currentStyle"
            @update:model-value="handleStyleChange"
          />
        </el-collapse-item>
      </el-collapse>
    </template>
    <div v-else class="text-sm text-gray-400 text-center py-6">请选择组件</div>
  </div>
</template>

<script setup>
/**
 * 样式面板
 * 提供基础样式和布局样式的编辑功能
 */
import { computed, ref, watch } from "vue";
import { storeToRefs } from "pinia";
import { useEditorStore } from "@/stores/editor-store";
import SizeEditor from "./StylePanel/SizeEditor.vue";
import SpacingEditor from "./StylePanel/SpacingEditor.vue";
import BackgroundEditor from "./StylePanel/BackgroundEditor.vue";
import BorderEditor from "./StylePanel/BorderEditor.vue";
import PositionEditor from "./StylePanel/PositionEditor.vue";

const editorStore = useEditorStore();
const { doc, selection } = storeToRefs(editorStore);

const activeNames = ref(["basic", "layout"]);

/**
 * 获取当前选中的节点
 */
const selectedNode = computed(() => {
  const primary = selection.value?.getPrimaryElement();
  if (!primary || primary.kind !== "node") return null;
  return doc.value?.getNode(primary.id) || null;
});

/**
 * 是否有选中元素
 */
const hasSelection = computed(() => selectedNode.value !== null);

/**
 * 解析尺寸为像素值
 * @param {string | number | undefined} value - 尺寸值
 * @returns {number | undefined}
 */
const parseSizeToNumber = (value) => {
  if (value === undefined || value === null) return undefined;
  const text = String(value).trim();
  if (!text || text === "auto") return undefined;
  if (!text.endsWith("px")) return undefined;
  const num = Number.parseFloat(text.slice(0, -2));
  return Number.isFinite(num) ? num : undefined;
};

/**
 * 计算容器最小尺寸，避免小于内部区域
 * @param {import('@/editor-core').ComponentNode | null} containerNode - 容器节点
 * @returns {{ width: number, height: number } | null}
 */
const resolveElContainerMinSize = (containerNode) => {
  if (!containerNode || containerNode.type !== "ElContainer") return null;
  const children = containerNode.children || [];
  let hasHeader = false;
  let hasFooter = false;
  let hasAside = false;
  let hasMain = false;
  for (const childId of children) {
    const childNode = doc.value?.getNode?.(childId);
    if (!childNode) continue;
    if (childNode.type === "ElHeader") hasHeader = true;
    if (childNode.type === "ElFooter") hasFooter = true;
    if (childNode.type === "ElAside") hasAside = true;
    if (childNode.type === "ElMain") hasMain = true;
  }

  const props = containerNode.props || {};
  if (typeof props.showHeader === "boolean") hasHeader = props.showHeader;
  if (typeof props.showFooter === "boolean") hasFooter = props.showFooter;
  if (typeof props.showAside === "boolean") hasAside = props.showAside;
  if (typeof props.showMain === "boolean") hasMain = props.showMain;

  const headerHeight = parseSizeToNumber(props.headerHeight) ?? 60;
  const footerHeight = parseSizeToNumber(props.footerHeight) ?? 60;
  const asideWidth = parseSizeToNumber(props.asideWidth) ?? 200;
  const minBodySize = 40;

  const hasBody = hasAside || hasMain;
  let minWidth = 0;
  if (hasAside && hasMain) {
    minWidth = asideWidth + minBodySize;
  } else if (hasAside) {
    minWidth = asideWidth;
  } else if (hasMain) {
    minWidth = minBodySize;
  }

  let minHeight = 0;
  if (hasHeader) minHeight += headerHeight;
  if (hasFooter) minHeight += footerHeight;
  if (hasBody) minHeight += minBodySize;

  if (minWidth <= 0 && minHeight <= 0) return null;
  return { width: minWidth, height: minHeight };
};

/**
 * 容器最小尺寸
 */
const containerMinSize = computed(() =>
  resolveElContainerMinSize(selectedNode.value),
);

/**
 * 是否展示尺寸编辑器
 */
const showSizeEditor = computed(() => {
  const node = selectedNode.value;
  if (!node) return true;
  const regionTypes = ["ElHeader", "ElAside", "ElMain", "ElFooter"];
  if (!regionTypes.includes(node.type)) return true;
  const parentNode = doc.value?.getParent?.(node.id);
  return parentNode?.type !== "ElContainer";
});

/**
 * 当前样式
 */
const currentStyle = computed(() => {
  if (!selectedNode.value) return {};
  return selectedNode.value.style || {};
});

/**
 * 处理样式变更
 * @param {Object} newStyle - 新样式对象
 */
const handleStyleChange = (newStyle) => {
  if (!selectedNode.value) return;
  editorStore.updateNode(selectedNode.value.id, { style: newStyle });
};
</script>

<style scoped>
.style-panel {
  padding: 8px;
}

.style-panel :deep(.el-collapse-item__header) {
  font-size: 13px;
  font-weight: 500;
}

.style-panel :deep(.el-collapse-item__content) {
  padding: 12px 8px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
</style>
