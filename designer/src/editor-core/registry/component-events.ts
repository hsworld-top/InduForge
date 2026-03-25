/**
 * 组件事件定义：事件面板与预览脚本
 */

export interface EventDefinition {
  name: string;
  label: string;
  description?: string;
}

export const defaultEventDefinitions: EventDefinition[] = [
  { name: "click", label: "点击", description: "点击时触发" },
  { name: "dblclick", label: "双击", description: "双击时触发" },
  { name: "mouseenter", label: "鼠标移入", description: "鼠标移入时触发" },
  { name: "mouseleave", label: "鼠标移出", description: "鼠标移出时触发" },
];

type ComponentEventEntry = string | EventDefinition;

const componentEventMap: Record<string, ComponentEventEntry[]> = {
  Button: ["click"],
  Input: [
    "update:modelValue",
    "input",
    "change",
    "focus",
    "blur",
    "clear",
    "mouseleave",
    "mouseenter",
    "keydown",
    "compositionstart",
    "compositionupdate",
    "compositionend",
  ],
  InputNumber: ["update:modelValue", "input", "change", "focus", "blur"],
  Select: [
    "update:modelValue",
    "change",
    "visible-change",
    "remove-tag",
    "clear",
    "focus",
    "blur",
    "popup-scroll",
  ],
  Switch: ["update:modelValue", "change", "input"],
  Table: [
    "select",
    "select-all",
    "selection-change",
    "cell-mouse-enter",
    "cell-mouse-leave",
    "cell-contextmenu",
    "cell-click",
    "cell-dblclick",
    "row-click",
    "row-contextmenu",
    "row-dblclick",
    "header-click",
    "header-contextmenu",
    "sort-change",
    "filter-change",
    "current-change",
    "header-dragend",
    "expand-change",
    "scroll",
  ],
  BigDataTable: [
    "select",
    "select-all",
    "selection-change",
    "cell-mouse-enter",
    "cell-mouse-leave",
    "cell-contextmenu",
    "cell-click",
    "cell-dblclick",
    "row-click",
    "row-contextmenu",
    "row-dblclick",
    "header-click",
    "header-contextmenu",
    "sort-change",
    "filter-change",
    "current-change",
    "header-dragend",
    "expand-change",
    "scroll",
  ],
  Tree: [
    "check",
    "check-change",
    "current-change",
    "node-click",
    "node-contextmenu",
    "node-collapse",
    "node-expand",
    "node-drag-start",
    "node-drag-end",
    "node-drop",
    "node-drag-leave",
    "node-drag-enter",
    "node-drag-over",
  ],
  Transfer: ["update:modelValue", "change", "left-check-change", "right-check-change"],
  Tag: ["close", "click"],
  Dropdown: ["visible-change", "click", "command"],
  Menu: ["select", "open", "close"],
  Radio: ["update:modelValue", "change"],
  Checkbox: ["update:modelValue", "change"],
  Cascader: [
    "update:modelValue",
    "change",
    "focus",
    "blur",
    "clear",
    "visible-change",
    "expand-change",
    "remove-tag",
  ],
  Tabs: ["update:modelValue", "tab-click", "tab-change", "tab-remove", "tab-add", "edit"],
  Timeline: [],
  Image: ["load", "error", "switch", "close", "show"],
  ImageCarousel: ["change"],
  CarouselComponent: ["change"],
  WebContainer: [],
  Steps: ["change"],
  Card: [],
  BusinessCard: [],
  Barcode: [],
  Pagination: [
    "update:current-page",
    "update:page-size",
    "size-change",
    "change",
    "current-change",
    "prev-click",
    "next-click",
  ],
  Collapse: ["update:modelValue", "change"],
  Slider: ["update:modelValue", "input", "change"],
  Calendar: ["update:modelValue", "input"],
  Signature: [
    "update:modelValue",
    "input",
    "change",
    "focus",
    "blur",
    "clear",
    "mouseleave",
    "mouseenter",
    "keydown",
    "compositionstart",
    "compositionupdate",
    "compositionend",
  ],
};

function buildEventLabelMap(): Map<string, EventDefinition> {
  const entries: [string, EventDefinition][] = [];
  for (const def of defaultEventDefinitions) {
    entries.push([def.name, def]);
  }
  for (const items of Object.values(componentEventMap)) {
    for (const item of items) {
      if (typeof item === "string") {
        entries.push([item, { name: item, label: item, description: "" }]);
      } else if (item?.name) {
        entries.push([item.name, item]);
      }
    }
  }
  return new Map(entries);
}

const eventLabelMap = buildEventLabelMap();

export function getComponentEventDefinitions(type: string | undefined): ComponentEventEntry[] {
  if (!type) return defaultEventDefinitions;
  if (Object.hasOwn(componentEventMap, type)) {
    return componentEventMap[type] ?? [];
  }
  return defaultEventDefinitions;
}

export function normalizeEventDefinitions(
  events: Array<string | EventDefinition> | undefined,
): EventDefinition[] {
  if (!Array.isArray(events) || events.length === 0) return [];
  const normalized: EventDefinition[] = [];
  for (const item of events) {
    if (!item) continue;
    if (typeof item === "string") {
      const fallback = eventLabelMap.get(item);
      normalized.push({
        name: item,
        label: fallback?.label ?? item,
        description: fallback?.description ?? "",
      });
      continue;
    }
    if (!item.name) continue;
    normalized.push({
      name: item.name,
      label: item.label || item.name,
      description: item.description || "",
    });
  }
  return normalized;
}

export default {
  defaultEventDefinitions,
  getComponentEventDefinitions,
  normalizeEventDefinitions,
};
