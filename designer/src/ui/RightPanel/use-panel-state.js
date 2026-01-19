/**
 * 面板状态计算 Composable
 * 根据 SelectionModel 状态计算当前应显示的面板类型
 */

import { computed, onBeforeUnmount, ref, watch } from "vue";
import { storeToRefs } from "pinia";
import { useEditorStore } from "@/stores/editor-store";

/**
 * 面板状态类型
 * @typedef {'page' | 'element' | 'multi'} PanelState
 */

/**
 * 计算面板状态
 * @param {number} count - 选中元素数量
 * @returns {PanelState} 面板状态
 */
export function computePanelState(count) {
  if (count === 0) return "page";
  if (count === 1) return "element";
  return "multi";
}

/**
 * 面板状态 Composable
 * @returns {{
 *   panelState: import('vue').Ref<PanelState>,
 *   selectedCount: import('vue').Ref<number>,
 *   selectedElements: import('vue').Ref<Array<{ id: string, kind: string }>>,
 *   primaryElement: import('vue').Ref<{ id: string, kind: string } | null>,
 *   selectedNode: import('vue').Ref<any>,
 *   selectedGraphic: import('vue').Ref<any>
 * }}
 */
export function usePanelState() {
  const editorStore = useEditorStore();
  const { doc, selection, docVersion } = storeToRefs(editorStore);

  /** @type {import('vue').Ref<number>} */
  const selectedCount = ref(0);

  /** @type {import('vue').Ref<Array<{ id: string, kind: string }>>} */
  const selectedElements = ref([]);

  /** @type {import('vue').Ref<{ id: string, kind: string } | null>} */
  const primaryElement = ref(null);

  /** @type {import('vue').Ref<any>} */
  const selectedNode = ref(null);

  /** @type {import('vue').Ref<any>} */
  const selectedGraphic = ref(null);

  let unsubscribeSelection = null;

  /**
   * 同步选中状态
   * @param {{ elements?: Array<{ id: string, kind: string }>, primary?: { id: string, kind: string } }} [payload]
   */
  const syncSelection = (payload) => {
    // ✅ 如果没有 selection 模型，显示页面属性
    if (!selection.value) {
      selectedCount.value = 0;
      selectedElements.value = [];
      primaryElement.value = null;
      selectedNode.value = null;
      selectedGraphic.value = null;
      return;
    }

    const elements = selection.value.getSelectedElements?.() || [];
    selectedCount.value = elements.length;
    selectedElements.value = elements;

    const primary = payload?.primary || selection.value.getPrimaryElement?.();
    primaryElement.value = primary || null;

    // 同步选中的节点或图形
    if (primary && doc.value) {
      if (primary.kind === "node") {
        const node = doc.value.getNode?.(primary.id) || null;
        selectedNode.value = node
          ? {
              ...node,
              props: { ...(node.props || {}) },
              style: { ...(node.style || {}) },
              children: Array.isArray(node.children)
                ? [...node.children]
                : node.children,
            }
          : null;
        selectedGraphic.value = null;
      } else if (primary.kind === "graphic") {
        const graphic = doc.value.getGraphic?.(primary.id) || null;
        selectedNode.value = null;
        selectedGraphic.value = graphic
          ? {
              ...graphic,
              props: { ...(graphic.props || {}) },
              style: { ...(graphic.style || {}) },
            }
          : null;
      } else {
        selectedNode.value = null;
        selectedGraphic.value = null;
      }
    } else {
      selectedNode.value = null;
      selectedGraphic.value = null;
    }
  };

  // ✅ 初始同步一次
  syncSelection();

  /**
   * 订阅选中变化
   * @param {import('@/editor-core').SelectionModel | null} model
   */
  const subscribeSelection = (model) => {
    if (!model) return;
    unsubscribeSelection = model.on?.("change", syncSelection);
    syncSelection();
  };

  // 监听 selection 变化
  watch(
    () => selection.value,
    (model) => {
      if (unsubscribeSelection) {
        unsubscribeSelection();
        unsubscribeSelection = null;
      }
      if (model) {
        subscribeSelection(model);
      } else {
        syncSelection();
      }
    },
    { immediate: true }
  );

  watch(
    () => docVersion.value,
    () => {
      syncSelection();
    }
  );

  // 组件卸载时取消订阅
  onBeforeUnmount(() => {
    if (unsubscribeSelection) {
      unsubscribeSelection();
      unsubscribeSelection = null;
    }
  });

  /**
   * 面板状态
   * @type {import('vue').ComputedRef<PanelState>}
   */
  const panelState = computed(() => {
    // ✅ 选中根容器时，显示页面属性面板
    if (selectedCount.value === 1 && selectedNode.value) {
      const currentPage = doc.value?.getCurrentPage?.();
      if (currentPage && selectedNode.value.id === currentPage.rootNodeId) {
        return "page";
      }
    }
    return computePanelState(selectedCount.value);
  });

  return {
    panelState,
    selectedCount,
    selectedElements,
    primaryElement,
    selectedNode,
    selectedGraphic,
  };
}

export default usePanelState;
