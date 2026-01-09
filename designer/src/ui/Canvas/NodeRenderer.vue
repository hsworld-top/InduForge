<template>
  <component
    v-if="node && !node.hidden"
    :is="renderTag"
    :class="nodeClass"
    :style="resolvedStyle"
    :data-node-id="node.id"
    :data-node-type="node.type"
    @click.stop="handleSelect"
    @dragover.prevent="handleDragOver"
    @dragleave="handleDragLeave"
    @drop.prevent="handleDrop"
    @contextmenu.prevent="handleContextMenu"
  >
    <template v-if="displayContent !== null">{{ displayContent }}</template>
    <template v-if="isContainer && !hasChildren && !props.isRoot">
      <div class="empty-container-hint">
        <span v-if="isDragOver">释放以添加组件</span>
        <span v-else>拖拽组件到此处</span>
      </div>
    </template>
    <!-- 插入线指示器 -->
    <div
      v-if="showInsertLine && insertLineStyle"
      class="insert-line"
      :class="insertLineStyle.orientation"
      :style="{
        [insertLineStyle.orientation === 'horizontal' ? 'top' : 'left']:
          insertLineStyle.offset + 'px',
      }"
    />
    <NodeRenderer
      v-for="childId in node.children || []"
      :key="childId"
      :node-id="childId"
    />
  </component>
</template>

<script setup>
import { computed, ref, inject } from "vue";
import { storeToRefs } from "pinia";
import { useEditorStore } from "@/stores/editor-store";
import { componentRegistry, createSelectableElement } from "@/editor-core";
import { createDragDropManager } from "./DragDropManager";

const props = defineProps({
  nodeId: {
    type: String,
    required: true,
  },
  isRoot: {
    type: Boolean,
    default: false,
  },
});

const editorStore = useEditorStore();
const { doc, selection } = storeToRefs(editorStore);
const canvasZoom = inject("canvasZoom", ref(1));

const node = computed(() => doc.value?.getNode(props.nodeId) || null);

/** 拖拽状态 */
const isDragOver = ref(false);
const showInsertLine = ref(false);
const insertLineStyle = ref(null);
const dragDropManager = createDragDropManager();

// ✅ 注入右键菜单显示函数
const showContextMenu = inject("showContextMenu", null);

const isContainer = computed(() => {
  if (!node.value) return false;
  const manifest = componentRegistry.get(node.value.type);
  return Boolean(manifest?.isContainer);
});

const hasChildren = computed(() => {
  return (node.value?.children || []).length > 0;
});

/**
 * 判断容器是否允许子组件
 * @param {import('@/editor-core').ComponentNode} parentNode - 父节点
 * @param {string} childType - 子组件类型
 * @returns {boolean}
 */
const canAcceptChild = (parentNode, childType) => {
  const manifest = componentRegistry.get(parentNode.type);
  const allowed = manifest?.allowedChildren;
  if (!Array.isArray(allowed) || allowed.length === 0) return true;
  return allowed.includes(childType);
};

/**
 * 计算落点相对坐标（考虑缩放）
 * @param {DragEvent} event - 拖拽事件
 * @param {HTMLElement} element - 目标容器
 * @returns {{ x: number, y: number }}
 */
const resolveDropOffset = (event, element) => {
  const rect = element.getBoundingClientRect();
  const zoomValue = Number(canvasZoom?.value) || 1;
  const offsetX = (event.clientX - rect.left) / zoomValue;
  const offsetY = (event.clientY - rect.top) / zoomValue;
  return {
    x: Math.max(0, Math.round(offsetX)),
    y: Math.max(0, Math.round(offsetY)),
  };
};

const nodeClass = computed(() => {
  if (!node.value) return "";
  const classes = ["designer-node"];
  if (props.isRoot) classes.push("is-root");
  if (isContainer.value) classes.push("is-container");
  if (node.value.locked) classes.push("is-locked");
  if (isDragOver.value) classes.push("drag-over");
  if (selection.value?.isSelected(node.value.id)) {
    classes.push("is-selected");
  }
  return classes.join(" ");
});

const renderTag = computed(() => {
  if (!node.value) return "div";
  switch (node.value.type) {
    case "Button":
      return "button";
    case "Text":
      return "div";
    default:
      return "div";
  }
});

const displayContent = computed(() => {
  if (!node.value) return null;
  if (node.value.type === "Text") {
    return node.value.props?.text ?? node.value.label ?? "";
  }
  if (node.value.type === "Button") {
    return node.value.props?.text ?? node.value.label ?? "按钮";
  }
  return null;
});

const resolvedStyle = computed(() => {
  if (!node.value) return {};
  const baseStyle = resolveLayoutStyle(node.value, props.isRoot);
  const containerStyle = resolveContainerStyle(node.value, baseStyle);
  const customStyle = normalizeStyleObject(node.value.style || {});
  return {
    ...baseStyle,
    ...containerStyle,
    ...customStyle,
  };
});

/**
 * 处理节点选中逻辑
 * @param {MouseEvent} event - 鼠标事件
 */
const handleSelect = (event) => {
  if (!node.value || !selection.value) return;
  // 锁定的节点不能选中
  if (node.value.locked) return;

  const element = createSelectableElement("node", node.value.id);
  if (event.shiftKey) {
    selection.value.selectRange(element);
    return;
  }
  if (event.metaKey || event.ctrlKey) {
    selection.value.toggleSelect(element);
    return;
  }
  selection.value.select(element);
};

/**
 * 处理右键菜单
 * @param {MouseEvent} event - 鼠标事件
 */
const handleContextMenu = (event) => {
  // ✅ 阻止浏览器默认右键菜单
  event.preventDefault();
  event.stopPropagation();

  // 如果当前节点未被选中，先选中它
  if (!node.value || !selection.value) return;
  if (node.value.locked) return;

  const selectedElements = selection.value.getSelectedElements?.() || [];
  const isSelected = selectedElements.some(
    (el) => el.id === node.value.id && el.kind === "node"
  );

  if (!isSelected) {
    const element = createSelectableElement("node", node.value.id);
    selection.value.select(element);
  }

  // ✅ 显示右键菜单
  if (showContextMenu) {
    showContextMenu(event);
  }
};

/**
 * 处理拖拽悬停
 * @param {DragEvent} event - 拖拽事件
 */
const handleDragOver = (event) => {
  // ✅ 阻止事件冒泡
  event.stopPropagation();

  if (!isContainer.value) return;
  if (!event.dataTransfer) return;

  const hasComponent = event.dataTransfer.types.includes(
    "application/x-designer-component"
  );
  if (!hasComponent) return;

  event.dataTransfer.dropEffect = "copy";
  isDragOver.value = true;

  // 计算插入位置
  if (node.value?.type === "FlexContainer") {
    const direction = node.value.props?.direction || "column";
    const currentElement = event.currentTarget;
    const insertInfo = dragDropManager.calculateFlexInsertPosition(
      currentElement,
      event,
      direction
    );

    if (insertInfo.insertLine) {
      showInsertLine.value = true;
      insertLineStyle.value = insertInfo.insertLine;
    }
  }
};

/**
 * 处理拖拽离开
 */
const handleDragLeave = () => {
  isDragOver.value = false;
  showInsertLine.value = false;
  insertLineStyle.value = null;
};

/**
 * 处理拖拽放置
 * @param {DragEvent} event - 拖拽事件
 */
const handleDrop = (event) => {
  // ✅ 阻止事件冒泡，避免重复插入
  event.stopPropagation();

  isDragOver.value = false;
  showInsertLine.value = false;
  insertLineStyle.value = null;

  if (!event.dataTransfer) return;

  const payload = event.dataTransfer.getData(
    "application/x-designer-component"
  );
  if (!payload) return;

  try {
    const { type } = JSON.parse(payload);
    if (!type) return;
    if (!node.value) return;
    if (!canAcceptChild(node.value, type)) return;

    // 计算插入位置
    let insertIndex = (node.value?.children || []).length;

    if (node.value?.type === "FlexContainer") {
      const direction = node.value.props?.direction || "column";
      const currentElement = event.currentTarget;
      const insertInfo = dragDropManager.calculateFlexInsertPosition(
        currentElement,
        event,
        direction
      );
      insertIndex = insertInfo.index;
    }

    const dropPosition =
      node.value?.type === "FreeContainer"
        ? resolveDropOffset(event, event.currentTarget)
        : null;

    // 插入新节点
    editorStore.insertNode(type, node.value?.id, insertIndex, {
      dropPosition: dropPosition || undefined,
    });
  } catch (err) {
    console.error("拖拽数据解析失败:", err);
  }
};

/**
 * 将布局配置转换为样式（支持新旧两种架构）
 * @param {import('@/editor-core').ComponentNode} currentNode - 当前节点
 * @param {boolean} isRoot - 是否根节点
 * @returns {Record<string, any>} 样式对象
 */
const resolveLayoutStyle = (currentNode, isRoot) => {
  if (isRoot) {
    return {
      position: "relative",
      width: "100%",
      height: "100%",
      boxSizing: "border-box",
    };
  }

  const style = {};

  // ✅ 新架构：positioning + absolutePos/flowLayout
  if (currentNode.positioning === "absolute" && currentNode.absolutePos) {
    const pos = currentNode.absolutePos;
    style.position = "absolute";
    style.left = `${pos.x ?? 0}px`;
    style.top = `${pos.y ?? 0}px`;
    if (pos.w !== undefined) style.width = `${pos.w}px`;
    if (pos.h !== undefined) style.height = `${pos.h}px`;
    if (pos.z !== undefined) style.zIndex = pos.z;
    return style;
  }

  if (currentNode.positioning === "flow") {
    // ✅ 流式布局：不使用绝对定位
    if (currentNode.flowLayout) {
      const flow = currentNode.flowLayout;
      // Flex 布局项
      if (
        flow.grow !== undefined ||
        flow.shrink !== undefined ||
        flow.basis !== undefined
      ) {
        style.flexGrow = flow.grow ?? 0;
        style.flexShrink = flow.shrink ?? 1;
        style.flexBasis = flow.basis ?? "auto";
        if (flow.alignSelf) style.alignSelf = flow.alignSelf;
      }
      // Grid 布局项
      if (flow.row !== undefined) {
        const rowSpan = flow.rowSpan || 1;
        style.gridRow = `${flow.row} / span ${rowSpan}`;
      }
      if (flow.col !== undefined) {
        const colSpan = flow.colSpan || 1;
        style.gridColumn = `${flow.col} / span ${colSpan}`;
      }
    }
    // 流式布局默认占据整行
    style.position = "relative";
    style.display = "block";
    return style;
  }

  // ✅ 旧架构：layoutItem（向后兼容）
  if (currentNode.layoutItem?.free) {
    const free = currentNode.layoutItem.free;
    if (free.mode === "abs" && free.abs) {
      const abs = free.abs;
      style.position = "absolute";
      style.left = `${abs.x ?? 0}px`;
      style.top = `${abs.y ?? 0}px`;
      if (abs.w !== undefined) style.width = `${abs.w}px`;
      if (abs.h !== undefined) style.height = `${abs.h}px`;
      if (abs.z !== undefined) style.zIndex = abs.z;
    }
    if (free.mode === "constraints" && free.constraints) {
      const constraints = free.constraints;
      style.position = "absolute";
      if (constraints.left !== undefined) style.left = `${constraints.left}px`;
      if (constraints.right !== undefined)
        style.right = `${constraints.right}px`;
      if (constraints.top !== undefined) style.top = `${constraints.top}px`;
      if (constraints.bottom !== undefined)
        style.bottom = `${constraints.bottom}px`;
      if (constraints.width !== undefined)
        style.width = `${constraints.width}px`;
      if (constraints.height !== undefined)
        style.height = `${constraints.height}px`;
    }
  }

  if (currentNode.layoutItem?.flex) {
    const flex = currentNode.layoutItem.flex;
    style.flexGrow = flex.grow ?? 0;
    style.flexShrink = flex.shrink ?? 1;
    style.flexBasis = flex.basis ?? "auto";
    if (flex.alignSelf) style.alignSelf = flex.alignSelf;
  }

  if (currentNode.layoutItem?.grid) {
    const grid = currentNode.layoutItem.grid;
    if (grid.row !== undefined) {
      const rowSpan = grid.rowSpan || 1;
      style.gridRow = `${grid.row} / span ${rowSpan}`;
    }
    if (grid.col !== undefined) {
      const colSpan = grid.colSpan || 1;
      style.gridColumn = `${grid.col} / span ${colSpan}`;
    }
  }

  return style;
};

/**
 * 解析容器布局样式
 * @param {import('@/editor-core').ComponentNode} currentNode - 当前节点
 * @param {Record<string, any>} baseStyle - 基础样式
 * @returns {Record<string, any>} 容器样式
 */
const resolveContainerStyle = (currentNode, baseStyle) => {
  const style = {};
  const manifest = componentRegistry.get(currentNode.type);
  const isContainer = manifest?.isContainer || false;

  if (currentNode.type === "FlexContainer") {
    style.display = "flex";
    style.flexDirection = currentNode.props?.direction || "row";
    style.flexWrap = currentNode.props?.wrap || "nowrap";
    style.justifyContent = currentNode.props?.justify || "flex-start";
    style.alignItems = currentNode.props?.align || "stretch";
    if (currentNode.props?.gap !== undefined) {
      style.gap = currentNode.props.gap;
    }
  } else if (currentNode.type === "GridContainer") {
    style.display = "grid";
    if (currentNode.props?.columns) {
      style.gridTemplateColumns = formatGridTemplate(currentNode.props.columns);
    }
    if (currentNode.props?.rows) {
      style.gridTemplateRows = formatGridTemplate(currentNode.props.rows);
    }
    if (currentNode.props?.gap !== undefined) {
      style.gap = currentNode.props.gap;
    }
  } else if (currentNode.type === "FreeContainer") {
    // ✅ FreeContainer 使用 flex 布局，支持流式布局子元素
    style.display = "flex";
    style.flexDirection = "column";
    style.flexWrap = "nowrap";
    style.position = "relative";
    if (currentNode.props?.overflow) {
      style.overflow = currentNode.props.overflow;
    }
  } else if (isContainer && !baseStyle.position) {
    style.position = "relative";
  }

  return style;
};

/**
 * 标准化样式对象
 * @param {Record<string, any>} rawStyle - 原始样式
 * @returns {Record<string, any>} 规范化样式
 */
const normalizeStyleObject = (rawStyle) => {
  const style = {};
  for (const [key, value] of Object.entries(rawStyle)) {
    if (value && typeof value === "object" && key === "background") {
      if (value.value !== undefined) {
        style.background = value.value;
      }
      continue;
    }
    style[key] = normalizeStyleValue(key, value);
  }
  return style;
};

/**
 * 标准化样式值（补充单位）
 * @param {string} key - 样式键
 * @param {any} value - 样式值
 * @returns {any} 标准化结果
 */
const normalizeStyleValue = (key, value) => {
  if (value === null || value === undefined) return value;
  if (typeof value === "number" && needsPxUnit(key)) {
    return `${value}px`;
  }
  return value;
};

/**
 * 判断是否需要追加 px 单位
 * @param {string} key - 样式键
 * @returns {boolean}
 */
const needsPxUnit = (key) => {
  return [
    "left",
    "top",
    "right",
    "bottom",
    "width",
    "height",
    "minWidth",
    "minHeight",
    "maxWidth",
    "maxHeight",
    "fontSize",
    "borderRadius",
    "gap",
    "padding",
    "paddingTop",
    "paddingRight",
    "paddingBottom",
    "paddingLeft",
    "margin",
    "marginTop",
    "marginRight",
    "marginBottom",
    "marginLeft",
  ].includes(key);
};

/**
 * 格式化 Grid 模板
 * @param {string | number} value - 模板配置
 * @returns {string}
 */
const formatGridTemplate = (value) => {
  if (typeof value === "number") {
    return `repeat(${value}, minmax(0, 1fr))`;
  }
  return value;
};
</script>

<style scoped>
.designer-node {
  position: relative;
  box-sizing: border-box;
  outline: 1px dashed transparent;
  transition: outline-color 0.15s ease;
}

.designer-node:hover {
  outline-color: rgba(59, 130, 246, 0.4);
}

.designer-node.is-selected {
  outline: 2px solid #3b82f6;
}

.designer-node.is-container {
  min-height: 40px;
}

.designer-node.is-locked {
  opacity: 0.6;
  pointer-events: none;
}

.designer-node.drag-over {
  outline: 2px solid #3b82f6;
  background-color: rgba(59, 130, 246, 0.05);
}

.empty-container-hint {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  min-height: 40px;
  color: #9ca3af;
  font-size: 12px;
  pointer-events: none;
  border: 1px dashed #d1d5db;
  border-radius: 4px;
  transition: all 0.2s ease;
}

.drag-over .empty-container-hint {
  border-color: #3b82f6;
  color: #3b82f6;
  background-color: rgba(59, 130, 246, 0.05);
}

.insert-line {
  position: absolute;
  background: #3b82f6;
  pointer-events: none;
  z-index: 9999;
  transition: all 0.1s ease;
}

.insert-line.horizontal {
  height: 2px;
  left: 0;
  right: 0;
}

.insert-line.vertical {
  width: 2px;
  top: 0;
  bottom: 0;
}

.designer-node.is-root {
  outline: none;
}

.designer-node.is-root:hover {
  outline: none;
}
</style>
