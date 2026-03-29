/**
 * 节点内容 Composable
 *
 * 从 NodeRenderer 抽取的 selectOptions、radioOptions、checkboxOptions、dropdownItems、
 * normalizeMenuItems、captureMenuDslConfig、activeTabName、tableRenderVersion 等组件特化数据。
 */

import type { ComputedRef, Ref } from "vue";
import type { ComponentNode } from "@/editor-core/document/types";
import { computed, ref, watch } from "vue";

const MENU_DSL_CALL_RE = /this\s*\.\s*menu\s*\(/g;

function normalizeOptions<T>(source: unknown, fallback: T[] | undefined): T[] {
  if (Array.isArray(source)) return source as T[];
  return fallback || [];
}

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
const fallbackCollapseItems = [
  { name: "1", title: "面板一", content: "内容一" },
  { name: "2", title: "面板二", content: "内容二" },
];

export interface NormalizedMenuItem extends Record<string, unknown> {
  label: string;
  index: string | undefined;
}

export function normalizeMenuItems(items: unknown): NormalizedMenuItem[] {
  if (!Array.isArray(items)) return [];
  return items
    .map((item): NormalizedMenuItem | null => {
      if (!item) return null;
      if (typeof item === "string") {
        return { label: item, index: item };
      }
      if (typeof item !== "object") return null;
      const o = item as Record<string, unknown>;
      const label = String(o.label ?? o.title ?? o.name ?? "");
      const index = (o.index ?? o.command ?? o.key ?? (label ? String(label) : undefined)) as
        | string
        | undefined;
      return { ...o, label, index };
    })
    .filter((x): x is NormalizedMenuItem => Boolean(x));
}

export function captureMenuDslConfig(content: unknown): unknown {
  const text = String(content || "");
  if (!text.trim()) return null;
  let captured: unknown = null;
  const replaced = text.replace(MENU_DSL_CALL_RE, "__menu__(");
  try {
    // eslint-disable-next-line no-new-func
    const runner = new Function("__menu__", `"use strict";\n${replaced}\nreturn null;`) as (
      fn: (config: unknown) => void,
    ) => unknown;
    runner((config: unknown) => {
      captured = config;
    });
  } catch {
    return null;
  }
  return captured;
}

function applyTabsModelValue(
  resolvedProps: Record<string, unknown>,
  activeTabName: Ref<string>,
  node: ComputedRef<ComponentNode | null | undefined>,
): Record<string, unknown> {
  if (!node.value || node.value.type !== "Tabs") return resolvedProps;
  const nextProps = { ...resolvedProps };
  const hasModelValue =
    Object.hasOwn(nextProps, "modelValue") &&
    nextProps.modelValue !== undefined &&
    nextProps.modelValue !== null &&
    String(nextProps.modelValue).trim();
  const hasActiveName =
    Object.hasOwn(nextProps, "activeName") &&
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

function applyCollapseModelValue(
  resolvedProps: Record<string, unknown>,
  activeCollapseName: Ref<string>,
  node: ComputedRef<ComponentNode | null | undefined>,
): Record<string, unknown> {
  if (!node.value || node.value.type !== "Collapse") return resolvedProps;
  const nextProps = { ...resolvedProps };
  const hasModelValue = Object.hasOwn(nextProps, "modelValue");
  if (!hasModelValue && activeCollapseName.value) {
    nextProps.modelValue = [activeCollapseName.value];
  }
  return nextProps;
}

export interface UseNodeContentDeps {
  node: ComputedRef<ComponentNode | null | undefined>;
  resolvedNodeProps: ComputedRef<Record<string, unknown>>;
  docVersion: Ref<number>;
}

export function useNodeContent({ node, resolvedNodeProps, docVersion }: UseNodeContentDeps) {
  const tableRenderVersion = ref(0);
  const activeTabName = ref("");
  const activeCollapseName = ref("");

  const selectOptions = computed(() => {
    void docVersion.value;
    return normalizeOptions(node.value?.props?.options, fallbackSelectOptions);
  });

  const radioOptions = computed(() => {
    void docVersion.value;
    return normalizeOptions(node.value?.props?.options, fallbackRadioOptions);
  });

  const checkboxOptions = computed(() => {
    void docVersion.value;
    return normalizeOptions(node.value?.props?.options, fallbackCheckboxOptions);
  });

  const dropdownItems = computed(() => {
    void docVersion.value;
    return normalizeOptions(node.value?.props?.items, fallbackDropdownItems);
  });

  const tabsList = computed(() => {
    void docVersion.value;
    return normalizeOptions(node.value?.props?.tabs, fallbackTabs);
  });

  const collapseItems = computed(() => {
    void docVersion.value;
    return normalizeOptions(node.value?.props?.items, fallbackCollapseItems);
  });

  const syncActiveTabName = (): void => {
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
    const firstTab = tabsList.value?.[0] as { name?: string; label?: string } | undefined;
    if (!firstTab) return;
    const fallbackName = firstTab.name ?? firstTab.label;
    if (fallbackName) activeTabName.value = String(fallbackName);
  };

  const syncActiveCollapseName = (): void => {
    if (!node.value || node.value.type !== "Collapse") return;
    const resolvedPropsValue = resolvedNodeProps.value || {};
    const rawModelValue = resolvedPropsValue.modelValue ?? node.value?.props?.modelValue;
    if (Array.isArray(rawModelValue) && rawModelValue.length) {
      const firstValue = String(rawModelValue[0] ?? "").trim();
      if (firstValue) {
        activeCollapseName.value = firstValue;
        return;
      }
    }
    if (rawModelValue !== undefined && rawModelValue !== null && !Array.isArray(rawModelValue)) {
      const normalized = String(rawModelValue).trim();
      if (normalized) {
        activeCollapseName.value = normalized;
        return;
      }
    }
    const firstItem = collapseItems.value?.[0] as
      | { name?: string; title?: string; label?: string }
      | undefined;
    if (!firstItem) return;
    const fallbackName = firstItem.name ?? firstItem.title ?? firstItem.label;
    if (fallbackName) {
      activeCollapseName.value = String(fallbackName);
    }
  };

  watch(
    () => [
      node.value?.id,
      node.value?.type,
      resolvedNodeProps.value?.modelValue,
      resolvedNodeProps.value?.activeName,
      tabsList.value?.length,
      collapseItems.value?.length,
    ],
    () => {
      syncActiveTabName();
      syncActiveCollapseName();
    },
    { immediate: true },
  );

  const applyTabsModelValueToProps = (resolvedProps: Record<string, unknown>) => {
    return applyTabsModelValue(resolvedProps, activeTabName, node);
  };

  const applyCollapseModelValueToProps = (resolvedProps: Record<string, unknown>) => {
    return applyCollapseModelValue(resolvedProps, activeCollapseName, node);
  };

  return {
    selectOptions,
    radioOptions,
    checkboxOptions,
    dropdownItems,
    tabsList,
    collapseItems,
    normalizeMenuItems,
    captureMenuDslConfig,
    activeTabName,
    activeCollapseName,
    tableRenderVersion,
    syncActiveTabName,
    syncActiveCollapseName,
    applyTabsModelValueToProps,
    applyCollapseModelValueToProps,
  };
}

export default { useNodeContent };
