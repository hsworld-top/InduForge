/**
 * NodeRenderer 模板中的节点类型布尔标志
 */

import type { ComputedRef } from "vue";
import type { ComponentNode } from "@/editor-core/document/types";
import { computed } from "vue";

const EL_LAYOUT_FAMILY = new Set(["ElLayout", "ElLayoutRow", "ElCol"]);

export interface UseNodeRendererTypeFlagsOpts {
  readonly: ComputedRef<boolean>;
  isRegionContainer: ComputedRef<boolean>;
}

export function useNodeRendererTypeFlags(
  node: ComputedRef<ComponentNode | null | undefined>,
  { readonly, isRegionContainer }: UseNodeRendererTypeFlagsOpts,
) {
  const type = computed(() => node.value?.type ?? "");

  const isSelectType = computed(() => type.value === "Select");
  const isRadioType = computed(() => type.value === "Radio");
  const isCheckboxType = computed(() => type.value === "Checkbox");
  const isTableType = computed(() => type.value === "Table");
  const isBigDataTableType = computed(() => type.value === "BigDataTable");
  const isMenuType = computed(() => type.value === "Menu");
  const isTimelineType = computed(() => type.value === "Timeline");
  const isTabsType = computed(() => type.value === "Tabs");
  const isCollapseType = computed(() => type.value === "Collapse");
  const isStepsType = computed(() => type.value === "Steps");
  const isCarouselSlotType = computed(
    () => type.value === "ImageCarousel" || type.value === "CarouselComponent",
  );
  const isDropdownType = computed(() => type.value === "Dropdown");

  const isElLayoutFamily = computed(() => EL_LAYOUT_FAMILY.has(type.value));

  const suppressReadonlyEmptyHint = computed(() => readonly.value);

  return {
    type,
    isSelectType,
    isRadioType,
    isCheckboxType,
    isTableType,
    isBigDataTableType,
    isMenuType,
    isTimelineType,
    isTabsType,
    isCollapseType,
    isStepsType,
    isCarouselSlotType,
    isDropdownType,
    suppressReadonlyEmptyHint,
  };
}
