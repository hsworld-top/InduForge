/**
 * 节点内容 Composable
 *
 * 从 NodeRenderer 抽取的 selectOptions、radioOptions、checkboxOptions、dropdownItems、
 * normalizeMenuItems、captureMenuDslConfig、activeTabName、tableRenderVersion 等组件特化数据。
 * 供递归渲染器复用。
 *
 * @module ui/Canvas/composables/use-node-content
 */

import { computed, ref, watch } from "vue";

/**
 * 规范化选项列表
 * @param {Array} source - 原始列表
 * @param {Array} fallback - 默认列表
 * @returns {Array}
 */
const normalizeOptions = (source, fallback) => {
  if (Array.isArray(source)) return source;
  return fallback || [];
};

// Fallback 数据常量
const fallbackSelectOptions = [
  { label: "选项一", value: "option1" },
  { label: "选项二", value: "option2" },
];
const fallbackRadioOptions = [
  { label: "选项一", value: "option1" },
  { label: "选项二", value: "option2" },
];
const fallbackCheckboxOptions = [
  { label: "选项一", value: "option1" },
  { label: "选项二", value: "option2" },
];
const fallbackDropdownItems = [
  { label: "操作一", value: "action1" },
  { label: "操作二", value: "action2" },
];
const fallbackTabs = [
  { name: "tab1", label: "标签一", content: "" },
  { name: "tab2", label: "标签二", content: "" },
  { name: "tab3", label: "标签三", content: "" },
];

/**
 * 规范化菜单项列表
 * @param {Array} items - 原始菜单项列表
 * @returns {Array} 规范化后的菜单项列表
 */
const normalizeMenuItems = (items) => {
  if (!Array.isArray(items)) return [];
  return items
    .map((item) => {
      if (!item) return null;
      if (typeof item === "string") {
        return { label: item, index: item };
      }
      if (typeof item !== "object") return null;
      const label = item.label ?? item.title ?? item.name ?? "";
      const index =
        item.index ??
        item.command ??
        item.key ??
        (label ? String(label) : undefined);
      return { ...item, label, index };
    })
    .filter(Boolean);
};

/**
 * 从内容中捕获菜单 DSL 配置
 * @param {string} content - 内容文本
 * @returns {Object|null} 菜单配置对象，失败返回 null
 */
const captureMenuDslConfig = (content) => {
  const text = String(content || "");
  if (!text.trim()) return null;
  let captured = null;
  const replaced = text.replace(/this\s*\.\s*menu\s*\(/g, "__menu__(");
  try {
    const runner = new Function(
      "__menu__",
      `"use strict";\n${replaced}\nreturn null;`,
    );
    runner((config) => {
      captured = config;
    });
  } catch (error) {
    return null;
  }
  return captured;
};

/**
 * 创建节点内容数据
 * @param {Object} deps - 依赖对象
 * @param {import('vue').ComputedRef<Object|null>} deps.node - 节点 computed
 * @param {import('vue').ComputedRef<Object>} deps.resolvedNodeProps - 解析后的节点 props computed
 * @param {import('vue').Ref<number>} deps.docVersion - 文档版本 ref（用于触发响应式更新）
 * @returns {Object} 返回 selectOptions、radioOptions、checkboxOptions、dropdownItems、normalizeMenuItems、captureMenuDslConfig、activeTabName、tableRenderVersion
 */
/**
 * 应用 Tabs modelValue 处理到 resolvedProps
 * @param {Object} resolvedProps - 原始 resolvedProps
 * @param {import('vue').Ref<string>} activeTabName - Tabs 激活名称
 * @param {import('vue').ComputedRef<Object>} node - 节点 computed
 * @returns {Object} 处理后的 resolvedProps
 */
function applyTabsModelValue(resolvedProps, activeTabName, node) {
  if (!node.value || node.value.type !== "Tabs") return resolvedProps;
  const nextProps = { ...resolvedProps };
  const hasModelValue =
    Object.prototype.hasOwnProperty.call(nextProps, "modelValue") &&
    nextProps.modelValue !== undefined &&
    nextProps.modelValue !== null &&
    String(nextProps.modelValue).trim();
  const hasActiveName =
    Object.prototype.hasOwnProperty.call(nextProps, "activeName") &&
    nextProps.activeName !== undefined &&
    nextProps.activeName !== null &&
    String(nextProps.activeName).trim();
  if (!hasModelValue && hasActiveName) {
    nextProps.modelValue = nextProps.activeName;
  }
  if (!hasModelValue && !hasActiveName && activeTabName.value) {
    nextProps.modelValue = activeTabName.value;
  }
  return nextProps;
}

export function useNodeContent({ node, resolvedNodeProps, docVersion }) {
  // 响应式 refs
  const tableRenderVersion = ref(0);
  const activeTabName = ref("");

  // Computed options
  const selectOptions = computed(() => {
    docVersion.value; // 依赖 docVersion 触发更新
    return normalizeOptions(node.value?.props?.options, fallbackSelectOptions);
  });

  const radioOptions = computed(() => {
    docVersion.value;
    return normalizeOptions(node.value?.props?.options, fallbackRadioOptions);
  });

  const checkboxOptions = computed(() => {
    docVersion.value;
    return normalizeOptions(node.value?.props?.options, fallbackCheckboxOptions);
  });

  const dropdownItems = computed(() => {
    docVersion.value;
    return normalizeOptions(node.value?.props?.items, fallbackDropdownItems);
  });

  const tabsList = computed(() => {
    docVersion.value;
    return normalizeOptions(node.value?.props?.tabs, fallbackTabs);
  });

  /**
   * 同步 Tabs 激活项（编辑态跟随点击）
   */
  const syncActiveTabName = () => {
    if (!node.value || node.value.type !== "Tabs") return;
    const resolvedPropsValue = resolvedNodeProps.value || {};
    const rawActiveName =
      resolvedPropsValue.modelValue ??
      resolvedPropsValue.activeName ??
      node.value?.props?.activeName;
    if (rawActiveName !== undefined && rawActiveName !== null) {
      const normalized = String(rawActiveName).trim();
      if (normalized) {
        activeTabName.value = normalized;
        return;
      }
    }
    const firstTab = tabsList.value?.[0];
    if (!firstTab) return;
    const fallbackName = firstTab.name ?? firstTab.label;
    if (fallbackName) activeTabName.value = String(fallbackName);
  };

  // 监听 Tabs 组件的 activeName 变化，更新 activeTabName
  watch(
    () => [
      node.value?.id,
      node.value?.type,
      resolvedNodeProps.value?.modelValue,
      resolvedNodeProps.value?.activeName,
      tabsList.value?.length,
    ],
    () => {
      syncActiveTabName();
    },
    { immediate: true },
  );

  /**
   * 应用 Tabs modelValue 处理到 resolvedProps 的辅助函数
   */
  const applyTabsModelValueToProps = (resolvedProps) => {
    return applyTabsModelValue(resolvedProps, activeTabName, node);
  };

  return {
    selectOptions,
    radioOptions,
    checkboxOptions,
    dropdownItems,
    tabsList,
    normalizeMenuItems,
    captureMenuDslConfig,
    activeTabName,
    tableRenderVersion,
    syncActiveTabName,
    applyTabsModelValueToProps,
  };
}

export default { useNodeContent };
