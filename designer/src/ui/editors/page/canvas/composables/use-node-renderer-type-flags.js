/**
 * NodeRenderer 模板中的节点类型布尔标志，减少对 node?.type 的散落比较。
 *
 * @module ui/editors/page/canvas/composables/use-node-renderer-type-flags
 */

import { computed } from "vue";

const EL_LAYOUT_FAMILY = new Set(["ElLayout", "ElLayoutRow", "ElCol"]);

/**
 * @param {import('vue').ComputedRef<Object|null>} node
 * @param {Object} opts
 * @param {import('vue').ComputedRef<boolean>} opts.readonly
 * @param {import('vue').ComputedRef<boolean>} opts.isRegionContainer
 */
export function useNodeRendererTypeFlags(node, { readonly, isRegionContainer }) {
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
    () =>
      type.value === "ImageCarousel" || type.value === "CarouselComponent",
  );
  const isDropdownType = computed(() => type.value === "Dropdown");

  const isElLayoutFamily = computed(() => EL_LAYOUT_FAMILY.has(type.value));

  /** 只读预览下，区域容器或 El 布局族不显示「空容器」占位提示 */
  const suppressReadonlyEmptyHint = computed(
    () =>
      readonly.value &&
      (Boolean(isRegionContainer.value) || isElLayoutFamily.value),
  );

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
