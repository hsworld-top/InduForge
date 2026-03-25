/**
 * 节点内容 Composable
 *
 * 从 NodeRenderer 抽取的 selectOptions、radioOptions、checkboxOptions、dropdownItems、
 * normalizeMenuItems、captureMenuDslConfig、activeTabName、tableRenderVersion 等组件特化数据。
 */

import type { ComputedRef, Ref } from "vue";
import type { ComponentNode } from "@/editor-core/document/types";
import { computed, ref, watch } from "vue";

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
  const replaced = text.replace(/this\s*\.\s*menu\s*\(/g, "__menu__(");
  try {
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

export interface UseNodeContentDeps {
  node: ComputedRef<ComponentNode | null | undefined>;
  resolvedNodeProps: ComputedRef<Record<string, unknown>>;
  docVersion: Ref<number>;
}

export function useNodeContent({ node, resolvedNodeProps, docVersion }: UseNodeContentDeps) {
  const tableRenderVersion = ref(0);
  const activeTabName = ref("");

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

  const applyTabsModelValueToProps = (resolvedProps: Record<string, unknown>) => {
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
