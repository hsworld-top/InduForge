/**
 * NodeRenderer 中与组件类型相关的派生数据（computed）与菜单 DSL 解析。
 * 将 setup 内大量 .type === 分支收敛到 registry + 本 composable。
 *
 * @module ui/Canvas/composables/use-node-renderer-derivations
 */

import { computed } from "vue";
import IconEpLocation from "~icons/ep/location";
import IconEpDocument from "~icons/ep/document";
import IconEpMenu from "~icons/ep/menu";
import IconEpSetting from "~icons/ep/setting";
import {
  captureMenuDslConfig,
  normalizeMenuItems,
} from "./use-node-content.js";
import {
  getDesignerNodeLayoutClasses,
  getRegionDesignerHint,
  getRenderKey,
  usesComponentWrapper,
} from "@/components/descriptors/registry.js";

const fallbackMenuItems = [
  { index: "1", label: "菜单一" },
  { index: "2", label: "菜单二" },
  { index: "3", label: "菜单三" },
];
const fallbackTableColumns = [
  { label: "姓名", prop: "name" },
  { label: "年龄", prop: "age" },
  { label: "地址", prop: "address" },
];
const fallbackBigTableColumns = [
  { label: "名称", prop: "name" },
  { label: "数值", prop: "value" },
];
const fallbackTimelineItems = [
  { label: "步骤一", timestamp: "2024-01-01" },
  { label: "步骤二", timestamp: "2024-01-02" },
  { label: "步骤三", timestamp: "2024-01-03" },
];
const fallbackStepsItems = [
  { title: "步骤一" },
  { title: "步骤二" },
  { title: "步骤三" },
];
const fallbackCollapseItems = [
  { name: "1", title: "面板一", content: "内容一" },
  { name: "2", title: "面板二", content: "内容二" },
];
const fallbackCarouselItems = [{ label: "轮播一" }, { label: "轮播二" }];

const normalizeOptions = (source, fallback) => {
  if (Array.isArray(source)) return source;
  return fallback;
};

/**
 * @param {string} content
 * @returns {string}
 */
export function sanitizeDslContent(content) {
  return String(content || "")
    .replace(/[，]/g, ",")
    .replace(/[；]/g, ";")
    .replace(/[：]/g, ":");
}

/**
 * @param {string} content
 * @returns {Object|null}
 */
export function resolveMenuConfigFromContent(content) {
  const text = sanitizeDslContent(content).trim();
  if (!text) return null;
  const captured = captureMenuDslConfig(text);
  if (captured) return captured;
  if (text.startsWith("{") && text.endsWith("}")) {
    try {
      return new Function(`return (${text});`)();
    } catch {
      return null;
    }
  }
  try {
    return new Function(`return ({${text}});`)();
  } catch {
    return null;
  }
}

const resolveMenuIconComponent = (icon) => {
  if (!icon) return null;
  if (typeof icon === "object" || typeof icon === "function") return icon;
  const key = String(icon).trim().toLowerCase();
  if (!key) return null;
  const map = {
    location: IconEpLocation,
    document: IconEpDocument,
    menu: IconEpMenu,
    setting: IconEpSetting,
  };
  return map[key] || null;
};

/**
 * @param {Object} deps
 * @param {import('vue').ComputedRef<Object|null>} deps.node
 * @param {import('vue').ComputedRef<string>} deps.detailConfigText
 * @param {import('vue').Ref<number>} deps.docVersion
 * @param {import('vue').Ref<number>} deps.tableRenderVersion
 * @param {import('vue').ComputedRef<Object>} deps.resolvedNodeProps
 * @param {import('vue').Ref<number>} deps.selectionVersion
 * @param {import('vue').ShallowRef} deps.selection
 * @param {import('vue').ShallowRef} deps.doc
 * @param {import('vue').ComputedRef<boolean>} deps.isContainer
 * @param {import('vue').ComputedRef<boolean>} deps.isMovable
 * @param {import('vue').ComputedRef<boolean>} deps.isDropActive
 * @param {import('vue').Ref<string>} deps.activeTabName
 * @param {import('vue').ComputedRef<Array>} deps.tabsList
 * @param {Object} deps.props - defineProps 结果（isRoot, readonly）
 * @returns {Object}
 */
export function useNodeRendererDerivations({
  node,
  detailConfigText,
  docVersion,
  tableRenderVersion,
  resolvedNodeProps,
  selectionVersion,
  selection,
  doc,
  isContainer,
  isMovable,
  isDropActive,
  activeTabName,
  tabsList,
  props,
}) {
  const menuItems = computed(() => {
    const detailConfig = detailConfigText.value;
    const detailMenuConfig =
      node.value?.type === "Menu" && detailConfig
        ? resolveMenuConfigFromContent(detailConfig)
        : null;
    const detailItems = detailMenuConfig?.props?.items || detailMenuConfig?.items;
    const items = normalizeOptions(
      Array.isArray(detailItems) ? detailItems : node.value?.props?.items,
      fallbackMenuItems,
    );
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
        const iconComponent = resolveMenuIconComponent(item.icon);
        return { ...item, label, index, iconComponent };
      })
      .filter(Boolean);
  });

  const tableColumns = computed(() => {
    tableRenderVersion.value;
    return normalizeOptions(node.value?.props?.columns, fallbackTableColumns);
  });

  const bigTableColumns = computed(() => {
    tableRenderVersion.value;
    return normalizeOptions(
      node.value?.props?.columns,
      fallbackBigTableColumns,
    );
  });

  const timelineItems = computed(() => {
    return normalizeOptions(node.value?.props?.items, fallbackTimelineItems);
  });

  const stepsItems = computed(() => {
    return normalizeOptions(node.value?.props?.items, fallbackStepsItems);
  });

  const collapseItems = computed(() => {
    return normalizeOptions(node.value?.props?.items, fallbackCollapseItems);
  });

  const carouselItems = computed(() => {
    return normalizeOptions(node.value?.props?.items, fallbackCarouselItems);
  });

  const dropdownLabel = computed(() => {
    if (!node.value) return "下拉菜单";
    return node.value.props?.label || node.value.label || "下拉菜单";
  });

  const activeTabChildIds = computed(() => {
    docVersion.value;
    if (!node.value || node.value.type !== "Tabs" || !doc.value) return [];
    const current = activeTabName.value;
    return (node.value.children || []).filter((childId) => {
      const childNode = doc.value?.getNode?.(childId);
      if (!childNode) return false;
      const tabKey = childNode.props?.tabKey;
      if (!tabKey) {
        return Boolean(current);
      }
      return String(tabKey) === String(current);
    });
  });

  const regionHintText = computed(() => {
    if (!node.value) return "";
    return getRegionDesignerHint(node.value.type);
  });

  const useComponentWrapper = computed(() => {
    return usesComponentWrapper(node.value?.type);
  });

  const renderKey = computed(() => {
    if (!node.value) return "";
    const nodeType = node.value.type;
    return getRenderKey(nodeType, node.value, {
      tableRenderVersion: tableRenderVersion.value,
      resolvedProps: resolvedNodeProps.value || {},
    });
  });

  const nodeClass = computed(() => {
    selectionVersion.value;
    if (!node.value) return "";
    const classes = ["designer-node"];
    if (props.isRoot) classes.push("is-root");
    if (props.readonly) classes.push("is-preview");
    if (isContainer.value) classes.push("is-container");
    if (isMovable.value && !props.readonly) classes.push("is-draggable");
    if (node.value.locked) classes.push("is-locked");

    const tabPosition =
      resolvedNodeProps.value?.tabPosition ||
      node.value.props?.tabPosition ||
      "top";
    const parentNode = doc.value?.getParent?.(node.value.id);
    const gutter = Number(parentNode?.props?.gutter) || 0;

    classes.push(
      ...getDesignerNodeLayoutClasses(node.value.type, {
        tabPosition,
        parentGutter: gutter,
      }),
    );

    if (isDropActive.value && !props.readonly) classes.push("drag-over");
    if (!props.readonly && selection.value?.isSelected(node.value.id)) {
      classes.push("is-selected");
    }
    return classes.join(" ");
  });

  return {
    menuItems,
    tableColumns,
    bigTableColumns,
    timelineItems,
    stepsItems,
    collapseItems,
    carouselItems,
    dropdownLabel,
    activeTabChildIds,
    regionHintText,
    useComponentWrapper,
    renderKey,
    nodeClass,
  };
}

/**
 * @param {Object} ctx
 * @param {import('vue').ComputedRef<Object|null>} ctx.node
 * @param {import('vue').Ref<boolean>} ctx.readonly
 * @param {Function} ctx.applyPreviewPatch
 * @param {Object} ctx.editorStore
 * @param {Function} ctx.normalizeMenuItems - 与 useNodeContent 一致
 * @returns {Function}
 */
export function createApplyMenuDslConfig(ctx) {
  const { node, readonly, applyPreviewPatch, editorStore, normalizeMenuItems: normMenu } =
    ctx;
  return (config) => {
    if (!config || typeof config !== "object") return;
    if (node.value?.type !== "Menu") return;
    const propsPatch = { ...(node.value?.props || {}) };
    const rawProps =
      config.props && typeof config.props === "object" ? { ...config.props } : {};
    if (Array.isArray(config.items)) {
      rawProps.items = config.items;
    }
    if (Array.isArray(rawProps.items)) {
      rawProps.items = normMenu(rawProps.items);
    }
    Object.assign(propsPatch, rawProps);
    if (config.className && String(config.className).trim()) {
      propsPatch.class = String(config.className).trim();
    }
    const nextPatch = { props: propsPatch };
    if (config.style && typeof config.style === "object") {
      nextPatch.style = { ...(node.value?.style || {}), ...config.style };
    }
    if (readonly.value) {
      applyPreviewPatch(nextPatch);
      return;
    }
    const ok = editorStore.updateNode(node.value.id, nextPatch);
    if (!ok) {
      applyPreviewPatch(nextPatch);
    }
  };
}

export default { useNodeRendererDerivations, createApplyMenuDslConfig };
