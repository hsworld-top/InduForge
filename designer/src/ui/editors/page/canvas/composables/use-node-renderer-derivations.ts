/**
 * NodeRenderer 中与组件类型相关的派生数据（computed）与菜单 DSL 解析。
 * 将 setup 内大量 .type === 分支收敛到 registry + 本 composable。
 *
 * @module ui/Canvas/composables/use-node-renderer-derivations
 */

import type { Component } from "vue";
import type { MenuDslApplyContext, UseNodeRendererDerivationsDeps } from "./types";
import { computed } from "vue";
import IconEpDocument from "~icons/ep/document";
import IconEpLocation from "~icons/ep/location";
import IconEpMenu from "~icons/ep/menu";
import IconEpSetting from "~icons/ep/setting";
import {
  getDesignerNodeLayoutClasses,
  getRegionDesignerHint,
  getRenderKey,
  usesComponentWrapper,
} from "@/editor-core/descriptors/registry";
import { captureMenuDslConfig } from "./use-node-content";

type MenuItem = Record<string, unknown>;
type MenuConfig = Record<string, unknown> & {
  items?: unknown[];
  props?: Record<string, unknown> & {
    items?: unknown[];
  };
  style?: Record<string, unknown>;
  className?: string;
};

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
const fallbackStepsItems = [{ title: "步骤一" }, { title: "步骤二" }, { title: "步骤三" }];
const fallbackCollapseItems = [
  { name: "1", title: "面板1", content: "内容1" },
  { name: "2", title: "面板2", content: "内容2" },
];
const fallbackCarouselItems = [{ label: "轮播一" }, { label: "轮播二" }];
const fullWidthCommaPattern = /，/g;
const fullWidthSemicolonPattern = /；/g;
const fullWidthColonPattern = /：/g;

function normalizeOptions<T>(
  source: unknown,
  fallback: T[],
  options: { fallbackWhenEmpty?: boolean } = {},
): T[] {
  if (Array.isArray(source)) {
    if (source.length > 0 || !options.fallbackWhenEmpty) return source;
    return fallback;
  }
  return fallback;
}

/**
 * @param {string} content
 * @returns {string}
 */
export function sanitizeDslContent(content: string): string {
  return String(content || "")
    .replace(fullWidthCommaPattern, ",")
    .replace(fullWidthSemicolonPattern, ";")
    .replace(fullWidthColonPattern, ":");
}

/**
 * @param {string} content
 * @returns {object | null}
 */
export function resolveMenuConfigFromContent(content: string): MenuConfig | null {
  const text = sanitizeDslContent(content).trim();
  if (!text) return null;
  const captured = captureMenuDslConfig(text) as MenuConfig | null;
  if (captured) return captured;
  if (text.startsWith("{") && text.endsWith("}")) {
    try {
      // eslint-disable-next-line no-new-func
      return new Function(`return (${text});`)() as MenuConfig;
    } catch {
      return null;
    }
  }
  try {
    // eslint-disable-next-line no-new-func
    return new Function(`return ({${text}});`)() as MenuConfig;
  } catch {
    return null;
  }
}

function resolveMenuIconComponent(icon: unknown): Component | null {
  if (!icon) return null;
  if (typeof icon === "object" || typeof icon === "function") return icon;
  const key = String(icon).trim().toLowerCase();
  if (!key) return null;
  const map: Record<string, Component> = {
    location: IconEpLocation,
    document: IconEpDocument,
    menu: IconEpMenu,
    setting: IconEpSetting,
  };
  return map[key] || null;
}

/**
 * @param {object} deps
 * @param {import('vue').ComputedRef<object | null>} deps.node
 * @param {import('vue').ComputedRef<string>} deps.detailConfigText
 * @param {import('vue').Ref<number>} deps.docVersion
 * @param {import('vue').Ref<number>} deps.tableRenderVersion
 * @param {import('vue').ComputedRef<object>} deps.resolvedNodeProps
 * @param {import('vue').Ref<number>} deps.selectionVersion
 * @param {import('vue').ShallowRef} deps.selection
 * @param {import('vue').ShallowRef} deps.doc
 * @param {import('vue').ComputedRef<boolean>} deps.isContainer
 * @param {import('vue').ComputedRef<boolean>} deps.isMovable
 * @param {import('vue').ComputedRef<boolean>} deps.isDropActive
 * @param {import('vue').Ref<string>} deps.activeTabName
 * @param {import('vue').ComputedRef<Array>} deps.tabsList
 * @param {import('vue').Ref<string>} deps.activeCollapseName
 * @param {import('vue').ComputedRef<Array>} deps.collapseItems
 * @param {object} deps.props - defineProps 结果（isRoot, readonly）
 * @returns {object}
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
  tabsList: _tabsList,
  activeCollapseName,
  collapseItems: _collapseItems,
  props,
}: UseNodeRendererDerivationsDeps) {
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
      .map((item): MenuItem | null => {
        if (!item) return null;
        if (typeof item === "string") {
          return { label: item, index: item, iconComponent: null };
        }
        if (typeof item !== "object") return null;
        const menuItem = item as MenuItem;
        const label = menuItem.label ?? menuItem.title ?? menuItem.name ?? "";
        const index =
          menuItem.index ?? menuItem.command ?? menuItem.key ?? (label ? String(label) : undefined);
        const iconComponent = resolveMenuIconComponent(menuItem.icon);
        return { ...menuItem, label, index, iconComponent };
      })
      .filter(Boolean);
  });

  const tableColumns = computed(() => {
    void tableRenderVersion.value;
    return normalizeOptions(node.value?.props?.columns, fallbackTableColumns);
  });

  const bigTableColumns = computed(() => {
    void tableRenderVersion.value;
    return normalizeOptions(node.value?.props?.columns, fallbackBigTableColumns);
  });

  const timelineItems = computed(() => {
    return normalizeOptions(node.value?.props?.items, fallbackTimelineItems);
  });

  const stepsItems = computed(() => {
    return normalizeOptions(node.value?.props?.items, fallbackStepsItems);
  });

  const collapseItems = computed(() => {
    return normalizeOptions(_collapseItems.value, fallbackCollapseItems, {
      fallbackWhenEmpty: true,
    });
  });

  const carouselItems = computed(() => {
    return normalizeOptions(node.value?.props?.items, fallbackCarouselItems);
  });

  const dropdownLabel = computed(() => {
    if (!node.value) return "下拉菜单";
    return node.value.props?.label || node.value.label || "下拉菜单";
  });

  const resolveScopedChildIds = (
    keyProp: "tabKey" | "collapseKey",
    currentKey: string,
    firstKey: string,
  ): string[] => {
    void docVersion.value;
    if (!node.value || !doc.value) return [];
    return (node.value.children || []).filter((childId) => {
      const childNode = doc.value?.getNode?.(childId);
      if (!childNode) return false;
      const slotKey = childNode.props?.[keyProp];
      if (!slotKey) {
        return Boolean(currentKey) && currentKey === firstKey;
      }
      return String(slotKey) === currentKey;
    });
  };

  const resolveTabKey = (tab: Record<string, unknown> | null | undefined): string => {
    const raw = tab?.name ?? tab?.label ?? "";
    return String(raw || "").trim();
  };

  const resolveCollapseKey = (item: Record<string, unknown> | null | undefined): string => {
    const raw = item?.name ?? item?.title ?? item?.label ?? "";
    return String(raw || "").trim();
  };

  const getTabChildIds = (tab: Record<string, unknown> | null | undefined): string[] => {
    if (!node.value || node.value.type !== "Tabs") return [];
    const current = resolveTabKey(tab);
    const first = resolveTabKey(_tabsList.value?.[0]);
    return resolveScopedChildIds("tabKey", current, first);
  };

  const getCollapseChildIds = (item: Record<string, unknown> | null | undefined): string[] => {
    if (!node.value || node.value.type !== "Collapse") return [];
    const current = resolveCollapseKey(item);
    const first = resolveCollapseKey(_collapseItems.value?.[0]);
    return resolveScopedChildIds("collapseKey", current, first);
  };

  const activeTabChildIds = computed(() => {
    return getTabChildIds({ name: activeTabName.value });
  });

  const activeCollapseChildIds = computed(() => {
    return getCollapseChildIds({ name: activeCollapseName.value });
  });

  const regionHintText = computed(() => {
    if (!node.value) return "";
    return getRegionDesignerHint(node.value.type || "");
  });

  const useComponentWrapper = computed(() => {
    return usesComponentWrapper(node.value?.type || "");
  });

  const renderKey = computed(() => {
    if (!node.value) return "";
    const nodeType = node.value.type;
    return getRenderKey(nodeType, node.value as any, {
      tableRenderVersion: tableRenderVersion.value,
      resolvedProps: resolvedNodeProps.value || {},
    });
  });

  const nodeClass = computed(() => {
    void selectionVersion.value;
    void docVersion.value;
    if (!node.value) return "";
    const classes: string[] = ["designer-node"];
    if (props.isRoot) classes.push("is-root");
    if (props.readonly) classes.push("is-preview");
    if (isContainer.value) classes.push("is-container");
    if (isMovable.value && !props.readonly) classes.push("is-draggable");
    if (node.value.locked) classes.push("is-locked");

    const tabPosition = String(
      resolvedNodeProps.value?.tabPosition ?? node.value.props?.tabPosition ?? "top",
    );
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
    activeCollapseChildIds,
    getTabChildIds,
    getCollapseChildIds,
    regionHintText,
    useComponentWrapper,
    renderKey,
    nodeClass,
  };
}

/**
 * @param {object} ctx
 * @param {import('vue').ComputedRef<object | null>} ctx.node
 * @param {import('vue').Ref<boolean>} ctx.readonly
 * @param {Function} ctx.applyPreviewPatch
 * @param {object} ctx.editorStore
 * @param {Function} ctx.normalizeMenuItems - 与 useNodeContent 一致
 * @returns {Function}
 */
export function createApplyMenuDslConfig(ctx: MenuDslApplyContext) {
  const { node, readonly, applyPreviewPatch, editorStore, normalizeMenuItems: normMenu } = ctx;
  return (config: MenuConfig | null | undefined): void => {
    if (!config || typeof config !== "object") return;
    if (node.value?.type !== "Menu") return;
    const propsPatch = { ...(node.value?.props || {}) };
    const rawProps = config.props && typeof config.props === "object" ? { ...config.props } : {};
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
    const nextPatch: { props: Record<string, unknown>; style?: Record<string, unknown> } = {
      props: propsPatch,
    };
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
