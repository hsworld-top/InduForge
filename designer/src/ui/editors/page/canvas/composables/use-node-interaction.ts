// @ts-nocheck — 从 JS 收口为 .ts；registry 与事件参数待补类型。
/**
 * 节点交互 Composable
 *
 * 从 NodeRenderer 抽取的 handleClick、右键菜单、resize 手柄与布局辅助等逻辑。
 * handlePointerDown / handleResizePointerDown 分别在 use-node-pointer、use-node-resize。
 * 供递归渲染器复用。
 *
 * @module ui/Canvas/composables/use-node-interaction
 */

import { computed } from "vue";
import { isContainerType, isLayoutType, isRegionType } from "@/components/descriptors/registry";

/**
 * 获取区域 resize 配置
 * @param {string} type - 组件类型
 * @returns {object | null} resize 配置
 */
function getRegionResizeConfig(type) {
  const configMap = {
    ElHeader: { axis: "y", prop: "height", handles: ["s"] },
    ElAside: { axis: "x", prop: "width", handles: ["e"] },
  };
  if (type === "ElFooter") {
    return {
      axis: "y",
      prop: "height",
      handles: ["n"],
      invert: true,
    };
  }
  return configMap[type] || null;
}

/**
 * resize handles 定义
 */
const resizeHandles = [
  { key: "nw", x: -1, y: -1, cursor: "nwse-resize" },
  { key: "n", x: 0, y: -1, cursor: "ns-resize" },
  { key: "ne", x: 1, y: -1, cursor: "nesw-resize" },
  { key: "e", x: 1, y: 0, cursor: "ew-resize" },
  { key: "se", x: 1, y: 1, cursor: "nwse-resize" },
  { key: "s", x: 0, y: 1, cursor: "ns-resize" },
  { key: "sw", x: -1, y: 1, cursor: "nesw-resize" },
  { key: "w", x: -1, y: 0, cursor: "ew-resize" },
];

/**
 * 创建节点交互逻辑
 * @param {object} deps - 依赖
 * @param {import('vue').ComputedRef<object>} deps.node - 节点 computed
 * @param {import('vue').ComputedRef<object>} deps.doc - 文档 computed
 * @param {import('vue').ComputedRef<object>} deps.selection - 选择状态 computed
 * @param {import('vue').ComputedRef<number>} deps.selectionVersion - 选择版本 computed
 * @param {import('vue').ComputedRef<boolean>} deps.readonly - 只读模式 computed
 * @param {import('vue').ComputedRef<boolean>} deps.isRoot - 是否为根节点 computed
 * @param {Function} deps.isRootCanvasContainer - 判断是否为根画布容器函数
 * @param {Function} deps.isChildResizableByDescriptor - 判断子节点是否可 resize 函数
 * @param {import('vue').Ref<HTMLElement>} deps.nodeRef - 节点 DOM 引用
 * @param {Function} deps.createSelectableElement - 创建可选元素函数
 * @param {Function} deps.runPreviewScript - 运行预览脚本函数
 * @param {Function} deps.handleSelect - 处理选择函数
 * @param {Function} [deps.showContextMenu] - 显示右键菜单函数（可选，通过 inject 获取）
 * @returns {object} 返回 handleClick、showResizeHandles、visibleResizeHandles 等
 */
export function useNodeInteraction(deps) {
  const {
    node,
    doc,
    selection,
    selectionVersion,
    readonly,
    isRoot,
    isRootCanvasContainer,
    isChildResizableByDescriptor,
    nodeRef,
    createSelectableElement,
    runPreviewScript,
    handleSelect,
    showContextMenu = null,
  } = deps;

  /**
   * 判断节点是否在 ElLayoutRow 中
   */
  const isElColInRow = computed(() => {
    if (!node.value || node.value.type !== "ElCol") return false;
    const parentNode = doc.value?.getParent?.(node.value.id);
    return parentNode?.type === "ElLayoutRow";
  });

  /**
   * 判断节点是否被放入 ElCol 中
   */
  const isChildInElCol = computed(() => {
    if (!node.value) return false;
    const parentNode = doc.value?.getParent?.(node.value.id);
    return parentNode?.type === "ElCol";
  });

  /**
   * 可见的 resize handles
   */
  const visibleResizeHandles = computed(() => {
    if (isChildInElCol.value) return [];
    if (isElColInRow.value) {
      return resizeHandles.filter((handle) => ["e", "w"].includes(handle.key));
    }
    if (node.value?.type === "ElLayoutRow") {
      return resizeHandles.filter((handle) => ["n", "s"].includes(handle.key));
    }
    const config = getRegionResizeConfig(node.value?.type);
    if (!config) return resizeHandles;
    return resizeHandles.filter((handle) => config.handles.includes(handle.key));
  });

  /**
   * 是否显示 resize handles（基础检查，不含 selection 检查）
   */
  const showResizeHandlesBase = computed(() => {
    selectionVersion.value;
    if (readonly.value || isRoot.value) return false;
    if (!node.value || node.value.locked) return false;
    if (isRootCanvasContainer(node.value)) return false;
    if (node.value.type === "ElMain") return false;
    if (isChildInElCol.value) return false;
    const parentNode = doc.value?.getParent?.(node.value.id);
    if (
      parentNode?.type === "ElHeader" ||
      parentNode?.type === "ElAside" ||
      parentNode?.type === "ElMain" ||
      parentNode?.type === "ElFooter"
    ) {
      return false;
    }
    const config = getRegionResizeConfig(node.value?.type);
    if (config && config.handles.length === 0) return false;
    const parent = doc.value?.getParent?.(node.value.id);
    if (parent && !isChildResizableByDescriptor(parent.type)) {
      return false;
    }
    return true;
  });

  /**
   * 判断是否为容器
   */
  const isContainer = computed(() => {
    if (!node.value) return false;
    return isContainerType(node.value.type);
  });

  /**
   * 判断是否为区域容器
   */
  const isRegionContainer = computed(() => {
    if (!node.value) return false;
    return isRegionType(node.value.type);
  });

  /**
   * 解析点击时的选中目标
   * @param {MouseEvent} event - 鼠标事件
   * @returns {import('@/editor-core').ComponentNode | null}
   */
  const resolveClickSelectionTarget = (event) => {
    if (!node.value) return null;
    if (!isRegionContainer.value) return node.value;
    if (event?.altKey) return node.value;
    const parentNode = doc.value?.getParent?.(node.value.id);
    if (!parentNode || parentNode.type !== "ElContainer") return node.value;
    if (parentNode.locked) return node.value;
    return parentNode;
  };

  /**
   * 获取布局节点的最外层 ElLayout
   * @param {import('@/editor-core').ComponentNode | null} currentNode - 当前节点
   * @returns {import('@/editor-core').ComponentNode | null}
   */
  const resolveLayoutRootNode = (currentNode) => {
    if (!currentNode) return null;
    if (
      currentNode.type !== "ElLayout" &&
      currentNode.type !== "ElLayoutRow" &&
      currentNode.type !== "ElCol"
    ) {
      return null;
    }
    if (currentNode.type === "ElLayout") return currentNode;
    let parentNode = doc.value?.getParent?.(currentNode.id);
    while (parentNode) {
      if (parentNode.type === "ElLayout") return parentNode;
      parentNode = doc.value?.getParent?.(parentNode.id);
    }
    return null;
  };

  /**
   * 获取布局节点所在的 ElLayoutRow
   * @param {import('@/editor-core').ComponentNode | null} currentNode - 当前节点
   * @returns {import('@/editor-core').ComponentNode | null}
   */
  const resolveAncestorLayoutRow = (currentNode) => {
    if (!currentNode || !doc.value) return null;
    if (currentNode.type === "ElLayoutRow") return currentNode;
    let parentNode = doc.value.getParent?.(currentNode.id);
    while (parentNode) {
      if (parentNode.type === "ElLayoutRow") return parentNode;
      parentNode = doc.value.getParent?.(parentNode.id);
    }
    return null;
  };

  /**
   * 判断是否点击在容器边框区域
   * @param {MouseEvent} event - 鼠标事件
   * @returns {boolean}
   */
  const isClickOnContainerBorder = (event) => {
    if (!node.value || !isContainer.value) return false;
    const element = nodeRef.value;
    if (!element || !event || typeof event.clientX !== "number") return false;
    const rect = element.getBoundingClientRect?.();
    if (!rect) return false;
    const x = event.clientX;
    const y = event.clientY;
    if (x < rect.left || x > rect.right || y < rect.top || y > rect.bottom) {
      return false;
    }
    const isLayoutNode = node.value.type === "ElLayoutRow" || node.value.type === "ElCol";
    const edge = node.value.type === "ElCol" ? 16 : isLayoutNode ? 10 : 6;
    const nearEdge =
      x - rect.left <= edge ||
      rect.right - x <= edge ||
      y - rect.top <= edge ||
      rect.bottom - y <= edge;
    if (!nearEdge) return false;
    if (isLayoutNode) return true;
    const hitNodeEl = event.target?.closest?.("[data-node-id]");
    if (!hitNodeEl) return true;
    return hitNodeEl.getAttribute("data-node-id") === node.value.id;
  };

  /**
   * 处理点击事件
   * @param {MouseEvent} event - 鼠标事件
   */
  const handleClick = (event) => {
    if (readonly.value) {
      void runPreviewScript("click", event);
      return;
    }
    if (!node.value || !selection.value) return;
    if (node.value?.type === "Tabs") {
      const hitNodeEl = event.target?.closest?.("[data-node-id]");
      const hitNodeId = hitNodeEl?.getAttribute?.("data-node-id");
      if (hitNodeId && hitNodeId !== node.value.id) {
        return;
      }
    }
    const isLayoutContainerNode = node.value.type === "ElLayoutRow" || node.value.type === "ElCol";
    const hasSelectionModifier = event.shiftKey || event.altKey || event.metaKey || event.ctrlKey;
    const isBorderClick = isClickOnContainerBorder(event);
    if (node.value.type === "ElCol" && !hasSelectionModifier) {
      const rowNode = doc.value?.getParent?.(node.value.id);
      const isColSelected = selection.value.isSelected?.(node.value.id);
      if (rowNode?.type === "ElLayoutRow" && !rowNode.locked && isColSelected) {
        const rowElement = createSelectableElement("node", rowNode.id);
        selection.value.select(rowElement);
        return;
      }
    }
    if (isBorderClick) {
      handleSelect(event);
      return;
    }
    if (event.metaKey || event.ctrlKey) {
      let parentNode = doc.value?.getParent?.(node.value?.id);
      while (parentNode) {
        if (parentNode.type === "ElLayout") {
          // Ctrl/Meta 多选优先针对当前节点切换，不再强制退化为 ElLayout 根单选
          const element = createSelectableElement("node", node.value.id);
          selection.value.toggleSelect(element);
          return;
        }
        parentNode = doc.value?.getParent?.(parentNode.id);
      }
    }
    let targetNode = resolveClickSelectionTarget(event);
    let forceRowSelection = false;
    const layoutRoot = resolveLayoutRootNode(node.value);
    if (!isLayoutContainerNode) {
      if (event.metaKey || event.ctrlKey) {
        if (layoutRoot && layoutRoot.id !== node.value?.id) {
          targetNode = layoutRoot;
          forceRowSelection = true;
        }
      } else if (layoutRoot && layoutRoot.id !== node.value?.id) {
        targetNode = layoutRoot;
      }
    }
    if (targetNode === node.value && !isRegionContainer.value && !event?.altKey) {
      const parentNode = doc.value?.getParent?.(node.value.id);
      if (
        parentNode &&
        (parentNode.type === "ElHeader" ||
          parentNode.type === "ElAside" ||
          parentNode.type === "ElMain" ||
          parentNode.type === "ElFooter")
      ) {
        if (!isLayoutType(node.value.type)) {
          const containerNode = doc.value?.getParent?.(parentNode.id);
          if (containerNode?.type === "ElContainer" && !containerNode.locked) {
            targetNode = containerNode;
          }
        }
      }
    }
    if (!targetNode) return;
    const element = createSelectableElement("node", targetNode.id);
    if (event.shiftKey) {
      selection.value.selectRange(element);
      return;
    }
    if (forceRowSelection && (event.metaKey || event.ctrlKey)) {
      selection.value.select(element);
      return;
    }
    if (!forceRowSelection && (event.metaKey || event.ctrlKey)) {
      selection.value.toggleSelect(element);
      return;
    }
    selection.value.select(element);
  };

  /**
   * 处理双击事件
   * @param {MouseEvent} event - 鼠标事件
   */
  const handleDoubleClick = (event) => {
    if (readonly.value) return;
    if (!node.value || !selection.value) return;
    if (event && (event.ctrlKey || event.metaKey)) {
      const rowNode = resolveAncestorLayoutRow(node.value);
      if (rowNode && !rowNode.locked) {
        const element = createSelectableElement("node", rowNode.id);
        selection.value.select(element);
        return;
      }
    }
    if (node.value.type !== "ElCol") {
      const parentNode = doc.value?.getParent?.(node.value.id);
      if (parentNode?.type === "ElCol") {
        if (parentNode.locked) return;
        const element = createSelectableElement("node", parentNode.id);
        selection.value.select(element);
        return;
      }
    }
    if (node.value.type === "ElCol") {
      const rowNode = doc.value?.getParent?.(node.value.id);
      if (rowNode?.type === "ElLayoutRow" && selection.value.isSelected?.(rowNode.id)) {
        return;
      }
      const layoutRoot = resolveLayoutRootNode(node.value);
      if (layoutRoot) {
        const element = createSelectableElement("node", node.value.id);
        selection.value.select(element);
        return;
      }
    }
    if (!isRegionContainer.value) return;
    const parentNode = doc.value?.getParent?.(node.value.id);
    if (!parentNode || parentNode.type !== "ElContainer") return;
    if (parentNode.locked) return;
    const element = createSelectableElement("node", parentNode.id);
    selection.value.select(element);
  };

  /**
   * 处理右键菜单事件
   * @param {MouseEvent} event - 鼠标事件
   */
  const handleContextMenu = (event) => {
    if (readonly.value) return;
    // 阻止浏览器默认右键菜单
    event.preventDefault();
    event.stopPropagation();

    // 如果当前节点未被选中，先选中它
    if (!node.value || !selection.value) return;
    if (node.value.locked) return;

    const selectedElements = selection.value.getSelectedElements?.() || [];
    const isSelected = selectedElements.some((el) => el.id === node.value.id && el.kind === "node");

    if (!isSelected) {
      // 若已选中外层容器且当前节点是其子孙，保持外层选中
      const primary = selection.value.getPrimaryElement?.();
      let keepSelection = false;
      if (primary?.kind === "node" && doc.value) {
        let parent = doc.value.getParent?.(node.value.id);
        while (parent) {
          if (parent.id === primary.id) {
            keepSelection = true;
            break;
          }
          parent = doc.value.getParent?.(parent.id);
        }
      }
      if (!keepSelection) {
        const element = createSelectableElement("node", node.value.id);
        selection.value.select(element);
      }
    }

    // 显示右键菜单
    if (showContextMenu) {
      showContextMenu(event);
    }
  };

  return {
    handleClick,
    handleDoubleClick,
    handleContextMenu,
    showResizeHandlesBase,
    visibleResizeHandles,
    resizeHandles,
    getRegionResizeConfig,
    isElColInRow,
    isChildInElCol,
    isContainer,
    isRegionContainer,
    resolveClickSelectionTarget,
    resolveLayoutRootNode,
    resolveAncestorLayoutRow,
    isClickOnContainerBorder,
  };
}

export default { useNodeInteraction };
