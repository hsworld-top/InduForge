<script setup lang="ts">
/**
 * 属性面板
 * 用于编辑组件属性、样式配置和数据绑定
 * - 详情配置
 * - 样式配置
 * - 绑定配置
 */

import {
  ElAside as _ElAside,
  ElCalendar as _ElCalendar,
  ElCard as _ElCard,
  ElCarousel as _ElCarousel,
  ElCascader as _ElCascader,
  ElCheckboxGroup as _ElCheckboxGroup,
  ElCol as _ElCol,
  ElCollapse as _ElCollapse,
  ElContainer as _ElContainer,
  ElDropdown as _ElDropdown,
  ElFooter as _ElFooter,
  ElHeader as _ElHeader,
  ElImage as _ElImage,
  ElInput as _ElInput,
  ElInputNumber as _ElInputNumber,
  ElMain as _ElMain,
  ElMessage as _ElMessage,
  ElPagination as _ElPagination,
  ElRadioGroup as _ElRadioGroup,
  ElRow as _ElRow,
  ElSelect as _ElSelect,
  ElSlider as _ElSlider,
  ElSteps as _ElSteps,
  ElSwitch as _ElSwitch,
  ElTable as _ElTable,
  ElTabs as _ElTabs,
  ElTimeline as _ElTimeline,
  ElTransfer as _ElTransfer,
} from "element-plus";
import { storeToRefs } from "pinia";
import { computed, nextTick, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import IconEpArrowRight from "~icons/ep/arrow-right";
import IconEpEditPen from "~icons/ep/edit-pen";
import IconEpFolder from "~icons/ep/folder";
import IconEpGrid from "~icons/ep/grid";
import IconEpLink from "~icons/ep/link";
import IconEpList from "~icons/ep/list";
import MonacoEditor from "@/ui/shared/widgets/base/monaco-editor-async";
import { getManifest } from "@/materials/manifests";
import { resolveButtonShapeProps } from "@/editor-core/descriptors/button-props";
import {
  resolvePageVariableSnippet,
  resolveProjectVariableSnippet,
  resolveProjectVariableSourceLabel,
} from "@/services/data-variable-mapping";
import { useEditorStore } from "@/stores/editor-store";
import { getMenuDefaultProps } from "@/stores/editor/normalize-settings";
import { usePanelState } from "../composables/use-panel-state";
import CollapseItemsEditor from "./CollapseItemsEditor.vue";
import MultiInspectorPanel from "./MultiInspectorPanel.vue";
import PageInspectorPanel from "./PageInspectorPanel.vue";
import PropEditor from "./PropEditor.vue";
import StyleConfigEditorDialog from "./style-config/StyleConfigEditorDialog.vue";
import TabsItemsEditor from "./TabsItemsEditor.vue";
import {
  buildMenuDslContent,
  normalizeMenuItems,
  resolveMenuConfigFromContent,
} from "./property-panel-menu-dsl";
import {
  buildRegionSizeState,
  clampRegionSize as clampRegionSizeValue,
  resolveContainerMaxSize as computeContainerMaxSize,
  resolveElContainerMinSize as computeElContainerMinSize,
  getRegionMaxLabel as formatRegionMaxLabel,
  getRegionSizeText,
  parseSize,
  regionSizeDefaults,
} from "./property-panel-region-size";
import { elementTypeName } from "./property-panel-utils";
import PropertyPanelBasicSection from "./PropertyPanelBasicSection.vue";
import PropertyPanelBindingVarEnumDialog from "./PropertyPanelBindingVarEnumDialog.vue";
import PropertyPanelConfigStrip from "./PropertyPanelConfigStrip.vue";
import PropertyPanelLayoutForceSection from "./PropertyPanelLayoutForceSection.vue";
import PropertyPanelRawPropsFallback from "./PropertyPanelRawPropsFallback.vue";
import { usePropertyPanelNormalizeNodeType } from "./use-property-panel-normalize-node-type";

interface AnyRecord extends Record<string, any> {}

interface AnyArray extends Array<any> {}

interface SidebarNodeLike {
  id?: string;
  label?: string;
  type?: string;
  raw?: AnyRecord;
  componentName?: string;
  params?: string;
  children?: SidebarNodeLike[];
}

interface BindingRowLike {
  name?: string;
  groupId?: string | null;
  type?: string;
  description?: string;
  mapped?: boolean;
  [key: string]: any;
}

interface PresetFactoryLike {
  (...args: any[]): AnyArray;
}

interface PresetGroupLike {
  name: string;
  types: string[];
  detail?: PresetFactoryLike;
  style?: PresetFactoryLike;
}

const ElAside: any = _ElAside;
const ElCalendar: any = _ElCalendar;
const ElCard: any = _ElCard;
const ElCarousel: any = _ElCarousel;
const ElCascader: any = _ElCascader;
const ElCheckboxGroup: any = _ElCheckboxGroup;
const ElCol: any = _ElCol;
const ElCollapse: any = _ElCollapse;
const ElContainer: any = _ElContainer;
const ElDropdown: any = _ElDropdown;
const ElFooter: any = _ElFooter;
const ElHeader: any = _ElHeader;
const ElImage: any = _ElImage;
const ElInput: any = _ElInput;
const ElInputNumber: any = _ElInputNumber;
const ElMain: any = _ElMain;
const ElMessage: any = _ElMessage;
const ElPagination: any = _ElPagination;
const ElRadioGroup: any = _ElRadioGroup;
const ElRow: any = _ElRow;
const ElSelect: any = _ElSelect;
const ElSlider: any = _ElSlider;
const ElSteps: any = _ElSteps;
const ElSwitch: any = _ElSwitch;
const ElTable: any = _ElTable;
const ElTabs: any = _ElTabs;
const ElTimeline: any = _ElTimeline;
const ElTransfer: any = _ElTransfer;
const DSL_EL_PREFIX_RE = /^El/;
const DSL_TRAILING_COMMA_RE = /,$/;
const DETAIL_VAR_BINDING_RE = /:\s*(\$vars\.\w+)/g;
const DETAIL_GLOBAL_BINDING_RE = /:\s*(\$global\.\w+)/g;
const DETAIL_VAR_STRING_BINDING_RE = /:\s*["'](\$vars\.\w+)["']/g;
const DETAIL_GLOBAL_STRING_BINDING_RE = /:\s*["'](\$global\.\w+)["']/g;
const TEXT_NUMBER_PREFIX_RE = /^(-?\d+(\.\d+)?)/;
const DOUBLE_QUOTE_RE = /"/g;
const SNIPPET_OPTION_PLACEHOLDER = "$" + "{1:option}";
const SNIPPET_BOOL_PLACEHOLDER = "$" + "{2:false}";
const SNIPPET_METHOD_PLACEHOLDER = "$" + '{1:"setOption"}';
const SNIPPET_OPTION_ARG_PLACEHOLDER = "$" + "{2:option}";
const EL_CONTAINER_MANAGED_PROPS = new Set([
  "regionPreset",
  "showHeader",
  "showAside",
  "showMain",
  "showFooter",
  "headerHeight",
  "asideWidth",
  "footerHeight",
]);
const EL_CONTAINER_PRESET_OPTIONS = [
  {
    value: "aside-between",
    label: "标准区域",
    description: "上中下 + 左侧",
    grid: ["header header", "aside main", "footer footer"],
  },
  {
    value: "top-main",
    label: "上下结构",
    description: "顶部 + 主内容",
    grid: ["header", "main"],
  },
  {
    value: "aside-full-height",
    label: "侧栏通高",
    description: "左侧贯穿全高",
    grid: ["aside header", "aside main", "aside footer"],
  },
];

const editorStore = useEditorStore() as any;
const {
  doc,
  docVersion,
  currentPage,
  currentPageId,
  projectVariables,
  projectVariableGroups,
  globalScripts,
} = storeToRefs(editorStore as any) as any;

const projectId = computed<string>(
  () => editorStore.projectId || editorStore.project?.id || currentPage.value?.projectId || "",
);

/** 属性 */

const { panelState, selectedElements, selectedNode, selectedGraphic } = usePanelState();

/** 属性 */
const currentElement = computed<AnyRecord | null>(
  () => selectedNode.value || selectedGraphic.value,
);

/** 组件 ID */
const elementId = computed<string>(() => currentElement.value?.id || "-");

/** 组件类型 */
const elementType = computed<string>(() => elementTypeName(currentElement.value?.type) || "-");

/** 属性 */
const elementLabel = ref<string>("");
const elementDescription = ref<string>("");
const regionSizeState = ref<any>(buildRegionSizeState(undefined));
const { t, locale } = useI18n();
const propertyPanelLanguage = computed<string>(() =>
  String(locale.value || "")
    .toLowerCase()
    .startsWith("en")
    ? "en"
    : "zh",
);
const propertyPanelPresetLabelMap: Record<string, string> = {
  empty: "Empty Template",
  center: "Center Layout",
  "btn-font": "Font Size / Color / Weight",
  "btn-type-primary": "Primary Variant",
  "btn-type-text": "Text Variant",
  "btn-focus": "Remove Focus Border",
  "btn-radius-none": "Square Corners",
  "btn-radius-pill": "Pill Radius",
  "btn-bg-solid": "Solid Background",
  "btn-bg-gradient": "Gradient Background",
  "btn-bg-image": "Image Button",
  "btn-bg-image-hover": "Hover Background Swap",
  "btn-size-fixed": "Fixed Size Button",
  "btn-size-lineheight": "Auto Height + Centered Line Height",
  "btn-size-block": "Full Width Block Button",
  "btn-icon-size": "Icon Size",
  "btn-icon-gap": "Icon/Text Gap",
  "btn-icon-only": "Icon Only Button",
  "btn-disabled": "Disabled but Visible",
  "btn-disabled-keep": "Disabled Without Dimming",
  "btn-text-hover": "Text Hover Without Background",
  "btn-text-underline": "Text Button Underline",
  "btn-group-radius": "Remove Group Radius",
  "btn-group-divider": "Button Group Divider",
  "btn-hover-scale": "Hover Scale Up",
  "btn-active-scale": "Active Scale Down",
  "text-font": "Font Size / Color / Weight",
  "text-type-primary": "Primary Variant",
  "text-type-text": "Text Variant",
  "text-focus": "Remove Focus Border",
  "text-radius-none": "Square Corners",
  "text-radius-pill": "Pill Radius",
  "text-bg-solid": "Solid Background",
  "text-bg-gradient": "Gradient Background",
  "text-bg-image": "Background Image",
  "text-bg-image-hover": "Hover Background Swap",
  "text-size-fixed": "Fixed Size",
  "text-size-lineheight": "Auto Height + Centered Line Height",
  "text-size-block": "Full Width Block",
  "text-icon-size": "Icon Size",
  "text-icon-gap": "Icon/Text Gap",
  "text-icon-only": "Icon Only",
  "text-disabled": "Disabled but Visible",
  "text-disabled-keep": "Disabled Without Dimming",
  "text-hover-clear": "Hover Without Background",
  "text-underline": "Underline",
  "text-group-radius": "Remove Radius",
  "text-group-divider": "Divider",
  "text-hover-scale": "Hover Scale Up",
  "text-active-scale": "Active Scale Down",
  comment: "Comment Template",
  "echart-line": "Line Chart Template",
  "echart-bar": "Bar Chart Template",
  "echart-pie": "Pie Chart Template",
  "echart-radar": "Radar Chart Template",
  "echart-scatter": "Scatter Chart Template",
  "echart-gauge": "Gauge Template",
  "echart-area": "Area Line Chart",
  "echart-stacked-bar": "Stacked Bar Chart",
  "echart-donut": "Donut Chart",
  "echart-multi-line": "Multi-Line Chart",
  "btn-full": "Full Template",
  "btn-basic": "Basic Button",
  "btn-text": "Text Button",
  "btn-types": "Button Types",
  "btn-plain": "Plain Button",
  "btn-state": "Disabled / Loading",
  "btn-size": "Size Variants",
  "btn-round": "Rounded / Circle",
  "btn-icon": "Icon Button",
  "btn-loading": "Loading Button",
  "btn-confirm": "Confirmation Button",
  "btn-throttle": "Throttle",
  "btn-debounce": "Debounce",
  "btn-perms": "Permission Button",
  "btn-visibility": "Conditional Visibility",
  "text-basic": "Basic Text",
  "text-bind": "Bound Field",
  "text-empty": "Placeholder Text",
  "text-style": "Styled Text",
  "text-wrap": "Multi-line Text",
  "text-ellipsis": "Ellipsis",
  "text-ellipsis-tooltip": "Ellipsis + Tooltip",
  "text-dict": "Status Text Mapping",
  "text-dict-color": "Status Mapping + Color",
  "text-tag": "Tag Text",
  "text-link": "Clickable Text",
  "text-tooltip": "Tooltip Text",
  "text-copy": "Copy Text",
  "text-format-money": "Currency Format",
  "text-format-datetime": "Datetime Format",
  "text-action-edit": "Table Action - Edit",
  "text-action-delete": "Table Action - Delete",
  "text-multi-prefix": "Multi-Text Group Prefix",
  "text-multi-price": "Multi-Text Group Price",
  "text-full-dsl": "Full Config Template",
  "style-border": "Set Border",
  "style-background": "Set Background",
  "style-radius": "Set Border Radius",
  "style-shadow": "Set Shadow",
  "style-opacity": "Set Opacity",
};
const propertyPanelLiteralLabelMap: Record<string, string> = {
  属性: "Properties",
  组件: "Component",
  未命名: "Unnamed",
  工程变量: "Project Variables",
  自定义脚本: "Custom Scripts",
  页面变量: "Page Variables",
  图表方法: "Chart Methods",
  页面组件: "Page Components",
};

function localizePropertyPanelPreset(preset: AnyRecord) {
  if (propertyPanelLanguage.value !== "en") return preset;
  return {
    ...preset,
    label: propertyPanelPresetLabelMap[String(preset.id || "")] || preset.label,
  };
}

function localizePropertyPanelLiteral(value: string) {
  if (propertyPanelLanguage.value !== "en") return value;
  return propertyPanelLiteralLabelMap[value] || value;
}

usePropertyPanelNormalizeNodeType(selectedNode, editorStore.updateNode);

watch(
  () => selectedNode.value?.id,
  () => {
    if (!selectedNode.value) return;
    const type = elementTypeName(selectedNode.value.type);
    const manifest = type ? (getManifest(type) as AnyRecord | null) : null;
    const defaults = (manifest?.defaultProps as AnyRecord) || {};
    if (!defaults || Object.keys(defaults).length === 0) return;
    const props = selectedNode.value.props || {};
    let changed = false;
    const nextProps = { ...props };
    Object.entries(defaults).forEach(([key, value]) => {
      if (nextProps[key] === undefined) {
        if (type === "Button" && key === "shape") {
          nextProps[key] = props.circle ? "circle" : props.round ? "round" : value;
        } else {
          nextProps[key] = value;
        }
        changed = true;
      }
    });
    if (changed) {
      editorStore.updateNode(selectedNode.value.id, { props: nextProps });
    }
  },
  { immediate: true },
);

const configDialogVisible = ref(false);
const configDialogType = ref<"style" | "detail">("style");
const configDraft = ref<string>("");
const configEditorRef = ref<any>(null);
const selectedPresetId = ref<string>("");
const presetSearch = ref<string>("");
let detailDraftTimer: ReturnType<typeof setTimeout> | null = null;
function emitDetailConfig(nodeId: string, code: string) {
  if (!nodeId || !code) return;
  window.dispatchEvent(
    new CustomEvent("designer:detail-config", {
      detail: { nodeId, code },
    }),
  );
}

const bindingDialogVisible = ref(false);
const bindingEditorCode = ref<string>("");
const bindingEditorRef = ref<any>(null);
const bindingProp = ref<{ name: string; label?: string } | null>(null);
const bindingScriptSearch = ref<string>("");
const bindingComponentSearch = ref<string>("");
const bindingPageContextSearch = ref<string>("");
const bindingCustomTreeRef = ref<any>(null);
const bindingComponentTreeRef = ref<any>(null);
const configScriptSearch = ref<string>("");
const configComponentSearch = ref<string>("");
const configPageContextSearch = ref<string>("");
const configCustomTreeRef = ref<any>(null);
const configComponentTreeRef = ref<any>(null);
const enumInsertTarget = ref<"binding" | "config">("binding");
const bindingVariableEnumVisible = ref(false);
const bindingProjectVarSearch = ref<string>("");
const bindingEnumSelectedProjectGroupId = ref<string | null>(null);
const bindingEnumSelectedProjectVar = ref<BindingRowLike | null>(null);

/**
 * 应用本地补丁（用于设计态强制刷新）
 * @param {import('@/editor-core').ComponentNode} node - ?
 * @param {Partial<import('@/editor-core').ComponentNode>} patch - 补丁
 */
function applyLocalNodePatch(node: AnyRecord, patch: AnyRecord) {
  if (!node || !patch || typeof patch !== "object") return;
  if (doc.value?._updateNode && node.id) {
    doc.value._updateNode(node.id, patch);
    docVersion.value += 1;
    return;
  }
  if (patch.props) {
    node.props = { ...(node.props || {}), ...patch.props };
  }
  if (patch.style) {
    node.style = { ...(node.style || {}), ...patch.style };
  }
  if (patch.events) {
    node.events = { ...(node.events || {}), ...patch.events };
  }
  if (patch.bindings) {
    node.bindings = { ...(node.bindings || {}), ...patch.bindings };
  }
  if (patch.conditions) {
    node.conditions = { ...(node.conditions || {}), ...patch.conditions };
  }
  if (patch.detailConfig !== undefined) {
    node.detailConfig = patch.detailConfig;
  }
  if (patch.styleConfig !== undefined) {
    node.styleConfig = patch.styleConfig;
  }
  if (typeof patch.hidden === "boolean") {
    node.hidden = patch.hidden;
  }
  if (patch.label !== undefined) {
    node.label = patch.label;
  }
  if (patch.description !== undefined) {
    node.description = patch.description;
  }
  docVersion.value += 1;
}

/**
 * 强制更新节点（绕过只读限制）
 * @param {string} nodeId - 节点 ID
 * @param {Partial<import('@/editor-core').ComponentNode>} patch - 补丁
 */
function forceUpdateNode(nodeId: string, patch: AnyRecord) {
  if (!nodeId || !patch || typeof patch !== "object") return;
  const target = doc.value?.getNode?.(nodeId);
  if (doc.value?._updateNode) {
    doc.value._updateNode(nodeId, patch);
    docVersion.value += 1;
  } else if (target) {
    applyLocalNodePatch(target, patch);
  }
  if (target && target !== selectedNode.value) {
    applyLocalNodePatch(target, patch);
  }
}

/**
 * 设计态本地执行 Menu 详细配置
 * @param {import('@/editor-core').ComponentNode} node - 节点
 * @param {Record<string, any>} config - 配置
 */
function applyMenuDetailConfig(node: AnyRecord, config: AnyRecord) {
  const normalizedType = elementTypeName(node?.type);
  if (!node || normalizedType !== "Menu" || !config || typeof config !== "object") {
    return;
  }
  const propsPatch: AnyRecord = { ...(node.props || {}) };
  const rawProps = config.props && typeof config.props === "object" ? { ...config.props } : {};
  if (Array.isArray(config.items)) {
    rawProps.items = config.items;
  }
  if (Array.isArray(rawProps.items)) {
    rawProps.items = normalizeMenuItems(rawProps.items);
  }
  Object.assign(propsPatch, rawProps);
  if (config.className && String(config.className).trim()) {
    propsPatch.class = String(config.className).trim();
  }
  const nextPatch: AnyRecord = { props: propsPatch };
  if (config.style && typeof config.style === "object") {
    nextPatch.style = { ...(node.style || {}), ...config.style };
  }
  const ok = editorStore.updateNode(node.id, nextPatch);
  if (!ok) {
    applyLocalNodePatch(node, nextPatch);
  }
}

function resolveDetailConfigProps(config: AnyRecord) {
  const reservedKeys = new Set([
    "id",
    "label",
    "type",
    "props",
    "style",
    "className",
    "events",
    "bindings",
    "conditions",
    "visible",
    "click",
    "onClick",
    "confirm",
    "confirmTitle",
    "confirmType",
    "permission",
    "perms",
    "if",
    "show",
  ]);
  const propsPatch: AnyRecord =
    config.props && typeof config.props === "object" ? { ...(config.props || {}) } : {};
  if (Object.hasOwn(config, "id") && String(config.id || "").trim()) {
    propsPatch.id = String(config.id).trim();
  }
  const permissionCode = config.permission ?? config.perms;
  if (typeof permissionCode === "string" && permissionCode.trim()) {
    propsPatch.permission = permissionCode.trim();
  }
  Object.entries(config).forEach(([key, value]) => {
    if (reservedKeys.has(key)) return;
    if (propsPatch[key] === undefined) propsPatch[key] = value;
  });
  if (config.className && String(config.className).trim()) {
    propsPatch.class = String(config.className).trim();
  }
  return propsPatch;
}

function buildDetailConfigPatch(node: AnyRecord, config: AnyRecord) {
  if (!node || !config || typeof config !== "object") return null;
  const patch: AnyRecord = {};
  const propsPatch = resolveDetailConfigProps(config);
  if (propsPatch.type === node.type) {
    delete propsPatch.type;
  }
  if (Object.keys(propsPatch).length > 0) {
    patch.props = { ...(node.props || {}), ...propsPatch };
  }
  if (config.style && typeof config.style === "object") {
    patch.style = { ...(node.style || {}), ...(config.style || {}) };
  }
  if (config.events && typeof config.events === "object") {
    patch.events = { ...(node.events || {}), ...(config.events || {}) };
  }
  if (config.bindings && typeof config.bindings === "object") {
    patch.bindings = { ...(node.bindings || {}), ...(config.bindings || {}) };
  }
  if (config.conditions && typeof config.conditions === "object") {
    patch.conditions = { ...(node.conditions || {}), ...(config.conditions || {}) };
  }
  let clickCode = String(config.click ?? config.onClick ?? "").trim();
  if (clickCode && typeof config.confirm === "string" && config.confirm.trim()) {
    const confirmText = JSON.stringify(config.confirm.trim());
    clickCode = `if (confirm(${confirmText})) {\n  ${clickCode}\n}`;
  }
  if (clickCode) {
    patch.events = {
      ...(patch.events || node.events || {}),
      click: [{ type: "script", code: clickCode, enabled: true }],
    };
  }
  const visibleExpr = config.if ?? config.show;
  if (typeof visibleExpr === "boolean") {
    patch.hidden = !visibleExpr;
  } else if (typeof visibleExpr === "string" && visibleExpr.trim()) {
    patch.conditions = {
      ...(patch.conditions || node.conditions || {}),
      visible: visibleExpr.trim(),
    };
  }
  if (typeof config.visible === "boolean") {
    patch.hidden = !config.visible;
  }
  if (config.label !== undefined && String(config.label).trim()) {
    patch.label = String(config.label).trim();
  }
  return Object.keys(patch).length > 0 ? patch : null;
}

function applyGenericDetailConfig(node: AnyRecord, config: AnyRecord) {
  const patch = buildDetailConfigPatch(node, config);
  if (!patch) return;
  const ok = editorStore.updateNode(node.id, patch);
  if (!ok) {
    applyLocalNodePatch(node, patch);
  }
}

function createDslFactory(type: string) {
  return (config: AnyRecord = {}) => ({
    type,
    props: resolveDetailConfigProps(config),
    style: config?.style && typeof config.style === "object" ? config.style : undefined,
    events: config?.events && typeof config.events === "object" ? config.events : undefined,
    bindings:
      config?.bindings && typeof config.bindings === "object" ? config.bindings : undefined,
    conditions:
      config?.conditions && typeof config.conditions === "object" ? config.conditions : undefined,
  });
}

/**
 * 设计态执行详细配置脚本
 * @param {import('@/editor-core').ComponentNode} node - 节点
 * @param {string} content - 脚本
 */
function runDetailConfigLocal(node: AnyRecord, content: string) {
  if (!node || !content || !content.trim()) return;
  const normalizedType = elementTypeName(node?.type);
  const methodName = getDslMethodName(normalizedType || "");
  const safeContent =
    normalizedType === "Menu" ? buildMenuDslContent(content, methodName) : content;
  let lastMenuConfig: AnyRecord | null = null;
  // eslint-disable-next-line no-new-func
  const runner = new Function(
    `"use strict";\nreturn (function() {\n${safeContent}\n}).call(this);`,
  );
  const context = {
    menu: (config: AnyRecord) => {
      lastMenuConfig = config;
      applyMenuDetailConfig(node, config);
    },
    button: (config: AnyRecord) => {
      if (normalizedType === "Button") {
        applyGenericDetailConfig(node, config);
        return;
      }
      return createDslFactory("Button")(config);
    },
    text: createDslFactory("Text"),
    card: createDslFactory("Card"),
    horizontalLayout: (config: AnyRecord) => {
      if (normalizedType === "HorizontalLayout") {
        applyGenericDetailConfig(node, config);
        return;
      }
      return createDslFactory("HorizontalLayout")(config);
    },
    verticalLayout: (config: AnyRecord) => {
      if (normalizedType === "VerticalLayout") {
        applyGenericDetailConfig(node, config);
        return;
      }
      return createDslFactory("VerticalLayout")(config);
    },
    collapse: (config: AnyRecord) => {
      if (normalizedType === "Collapse") {
        applyGenericDetailConfig(node, config);
        return;
      }
      return createDslFactory("Collapse")(config);
    },
  } as AnyRecord;
  if (!Object.hasOwn(context, methodName)) {
    context[methodName] = (config: AnyRecord) => applyGenericDetailConfig(node, config);
  }
  try {
    runner.call(context);
    if (lastMenuConfig) {
      applyMenuDetailConfig(node, lastMenuConfig);
    }
  } catch (error) {
    console.error("[Designer] Detail config error:", error);
  }
}

const commonStylePresets: AnyArray = [
  {
    id: "style-border",
    label: "设置边框",
    content: "#domId {\n  border: 1px solid #dcdfe6;\n}",
  },
  {
    id: "style-background",
    label: "设置背景",
    content: "#domId {\n  background: #ffffff;\n}",
  },
  {
    id: "style-radius",
    label: "设置边框圆角",
    content: "#domId {\n  border-radius: 8px;\n  overflow: hidden;\n}",
  },
  {
    id: "style-shadow",
    label: "设置阴影",
    content: "#domId {\n  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.12);\n}",
  },
  {
    id: "style-opacity",
    label: "设置透明",
    content: "#domId {\n  opacity: 0.85;\n}",
  },
];

function withCommonStylePresets(presets: AnyArray): AnyArray {
  const existing = new Set((presets || []).map((item: AnyRecord) => String(item?.id || "")));
  return [
    ...commonStylePresets.filter((item: AnyRecord) => !existing.has(String(item.id || ""))),
    ...(presets || []),
  ];
}

const stylePresets: AnyArray = [
  { id: "empty", label: "空模板", content: "" },
  {
    id: "center",
    label: "居中布局",
    content: "display: flex;\nalign-items: center;\njustify-content: center;",
  },
  ...commonStylePresets,
];
const buttonStylePresets: AnyArray = [
  {
    id: "btn-font",
    label: "字体大小 / 颜色 / 粗细",
    content:
      "#domId .el-button {\n  font-size: 22px;\n  color: red;\n  font-weight: bold;\n  letter-spacing: 1px;\n}",
  },
  {
    id: "btn-type-primary",
    label: "primary 单独控制",
    content: "#domId .el-button--primary {\n  color: #fff;\n  font-size: 18px;\n}",
  },
  {
    id: "btn-type-text",
    label: "text 单独控制",
    content: "#domId .el-button--text {\n  color: #409EFF;\n  font-size: 14px;\n}",
  },
  {
    id: "btn-hover",
    label: "Hover",
    content:
      "#domId .el-button:hover {\n  background-color: red;\n  border-color: red;\n  color: white;\n}",
  },
  {
    id: "btn-active",
    label: "Active",
    content:
      "#domId .el-button:active {\n  background-color: #c00000;\n  border-color: #c00000;\n  color: #fff;\n}",
  },
  {
    id: "btn-focus",
    label: "Focus 去边框",
    content: "#domId .el-button:focus {\n  outline: none;\n  box-shadow: none;\n}",
  },
  {
    id: "btn-radius-none",
    label: "全部直角",
    content: "#domId .el-button { border-radius: 0px; }",
  },
  {
    id: "btn-radius-primary",
    label: "primary",
    content: "#domId .el-button--primary { border-radius: 0px; }",
  },
  {
    id: "btn-radius-pill",
    label: "圆角胶囊",
    content: "#domId .el-button { border-radius: 999px; }",
  },
  {
    id: "btn-bg-solid",
    label: "纯色背景",
    content:
      "#domId .el-button--primary {\n  background-color: #ff4d4f;\n  border-color: #ff4d4f;\n}",
  },
  {
    id: "btn-bg-gradient",
    label: "渐变背景",
    content:
      "#domId .el-button--primary {\n  background: linear-gradient(90deg, #ff7a18, #ffb347);\n  border: none;\n  color: #fff;\n}",
  },
  {
    id: "btn-bg-image",
    label: "背景图片按钮",
    content:
      '#domId .el-button--primary {\n  background: url("/static/images/images/login_run.png") no-repeat;\n  background-size: 100% 100%;\n  background-position: center;\n  border-color: transparent;\n  background-color: transparent !important;\n  width: 100% !important;\n  height: 100% !important;\n}',
  },
  {
    id: "btn-bg-image-hover",
    label: "Hover 切换背景图",
    content:
      '#domId .el-button--primary {\n  background: url("/static/images/btn_normal.png") no-repeat;\n  background-size: 100% 100%;\n}\n\n#domId .el-button--primary:hover {\n  background: url("/static/images/btn_hover.png") no-repeat;\n  background-size: 100% 100%;\n}',
  },
  {
    id: "btn-size-fixed",
    label: "固定尺寸按钮",
    content: "#domId .el-button {\n  width: 200px;\n  height: 56px;\n  padding: 0;\n}",
  },
  {
    id: "btn-size-lineheight",
    label: "自适应高度 + 行高居中",
    content: "#domId .el-button {\n  height: 48px;\n  line-height: 48px;\n}",
  },
  {
    id: "btn-size-block",
    label: "块级按钮 100%",
    content: "#domId .el-button {\n  width: 100%;\n  display: block;\n}",
  },
  {
    id: "btn-icon-size",
    label: "图标大小",
    content: "#domId .el-button i { font-size: 20px; }",
  },
  {
    id: "btn-icon-gap",
    label: "图标与文字间距",
    content: "#domId .el-button i { margin-right: 8px; }",
  },
  {
    id: "btn-icon-only",
    label: "纯图标按钮",
    content: "#domId .el-button.is-circle {\n  width: 48px;\n  height: 48px;\n  padding: 0;\n}",
  },
  {
    id: "btn-disabled",
    label: "禁用但可见",
    content:
      "#domId .el-button.is-disabled {\n  background-color: #ddd;\n  border-color: #ddd;\n  color: #999;\n  cursor: not-allowed;\n}",
  },
  {
    id: "btn-disabled-keep",
    label: "禁用保持主色",
    content:
      "#domId .el-button.is-disabled,\n#domId .el-button.is-disabled:hover {\n  background: #409EFF;\n  border-color: #409EFF;\n  color: #fff;\n  opacity: 0.5;\n}",
  },
  {
    id: "btn-text-hover",
    label: "text 按钮 hover 无背景",
    content: "#domId .el-button--text:hover {\n  background: transparent;\n  color: #66b1ff;\n}",
  },
  {
    id: "btn-text-underline",
    label: "text 按钮下划线",
    content: "#domId .el-button--text { text-decoration: underline; }",
  },
  {
    id: "btn-group-radius",
    label: "按钮组去圆角",
    content: "#domId .el-button-group .el-button {\n  border-radius: 0;\n}",
  },
  {
    id: "btn-group-divider",
    label: "按钮组分割线",
    content: "#domId .el-button-group .el-button + .el-button {\n  border-left: 1px solid #ddd;\n}",
  },
  {
    id: "btn-hover-scale",
    label: "Hover 放大",
    content:
      "#domId .el-button {\n  transition: all 0.2s ease;\n}\n\n#domId .el-button:hover {\n  transform: scale(1.05);\n}",
  },
  {
    id: "btn-active-scale",
    label: "点击缩放",
    content: "#domId .el-button:active {\n  transform: scale(0.97);\n}",
  },
];

function withButtonRootSelectorSupport(preset: AnyRecord): AnyRecord {
  const content = String(preset?.content || "").replace(
    /#domId\s+(\.el-button[^,{]*)/g,
    "#domId$1,\n#domId $1",
  );
  return { ...preset, content };
}

const textStylePresets: AnyArray = [
  {
    id: "text-font",
    label: "字体大小 / 颜色 / 粗细",
    content:
      "#domId {\n  font-size: 22px;\n  color: red;\n  font-weight: bold;\n  letter-spacing: 1px;\n}",
  },
  {
    id: "text-type-primary",
    label: "primary 单独控制",
    content: "#domId {\n  color: #fff;\n  font-size: 18px;\n}",
  },
  {
    id: "text-type-text",
    label: "text 单独控制",
    content: "#domId {\n  color: #409EFF;\n  font-size: 14px;\n}",
  },
  {
    id: "text-hover",
    label: "Hover",
    content: "#domId:hover {\n  background-color: red;\n  border-color: red;\n  color: white;\n}",
  },
  {
    id: "text-active",
    label: "Active",
    content:
      "#domId:active {\n  background-color: #c00000;\n  border-color: #c00000;\n  color: #fff;\n}",
  },
  {
    id: "text-focus",
    label: "Focus 去边框",
    content: "#domId:focus {\n  outline: none;\n  box-shadow: none;\n}",
  },
  {
    id: "text-radius-none",
    label: "全部直角",
    content: "#domId { border-radius: 0px; }",
  },
  {
    id: "text-radius-primary",
    label: "primary",
    content: "#domId { border-radius: 0px; }",
  },
  {
    id: "text-radius-pill",
    label: "圆角胶囊",
    content: "#domId { border-radius: 999px; }",
  },
  {
    id: "text-bg-solid",
    label: "纯色背景",
    content: "#domId {\n  background-color: #ff4d4f;\n  border-color: #ff4d4f;\n}",
  },
  {
    id: "text-bg-gradient",
    label: "渐变背景",
    content:
      "#domId {\n  background: linear-gradient(90deg, #ff7a18, #ffb347);\n  border: none;\n  color: #fff;\n}",
  },
  {
    id: "text-bg-image",
    label: "背景图片",
    content:
      '#domId {\n  background: url("/static/images/images/login_run.png") no-repeat;\n  background-size: 100% 100%;\n  background-position: center;\n  border-color: transparent;\n  background-color: transparent !important;\n  width: 100% !important;\n  height: 100% !important;\n}',
  },
  {
    id: "text-bg-image-hover",
    label: "Hover 切换背景图",
    content:
      '#domId {\n  background: url("/static/images/btn_normal.png") no-repeat;\n  background-size: 100% 100%;\n}\n\n#domId:hover {\n  background: url("/static/images/btn_hover.png") no-repeat;\n  background-size: 100% 100%;\n}',
  },
  {
    id: "text-size-fixed",
    label: "固定尺寸",
    content: "#domId {\n  width: 200px;\n  height: 56px;\n  padding: 0;\n}",
  },
  {
    id: "text-size-lineheight",
    label: "自适应高度 + 行高居中",
    content: "#domId {\n  height: 48px;\n  line-height: 48px;\n}",
  },
  {
    id: "text-size-block",
    label: "块级 100%",
    content: "#domId {\n  width: 100%;\n  display: block;\n}",
  },
  {
    id: "text-icon-size",
    label: "图标大小",
    content: "#domId i { font-size: 20px; }",
  },
  {
    id: "text-icon-gap",
    label: "图标与文字间距",
    content: "#domId i { margin-right: 8px; }",
  },
  {
    id: "text-icon-only",
    label: "纯图标",
    content: "#domId.is-circle {\n  width: 48px;\n  height: 48px;\n  padding: 0;\n}",
  },
  {
    id: "text-disabled",
    label: "禁用但可见",
    content:
      "#domId.is-disabled {\n  background-color: #ddd;\n  border-color: #ddd;\n  color: #999;\n  cursor: not-allowed;\n}",
  },
  {
    id: "text-disabled-keep",
    label: "禁用不变色",
    content:
      "#domId.is-disabled,\n#domId.is-disabled:hover {\n  background: #409EFF;\n  border-color: #409EFF;\n  color: #fff;\n  opacity: 0.5;\n}",
  },
  {
    id: "text-hover-clear",
    label: "Hover 无背景",
    content: "#domId:hover {\n  background: transparent;\n  color: #66b1ff;\n}",
  },
  {
    id: "text-underline",
    label: "下划线",
    content: "#domId { text-decoration: underline; }",
  },
  {
    id: "text-group-radius",
    label: "去圆角",
    content: "#domId { border-radius: 0; }",
  },
  {
    id: "text-group-divider",
    label: "分割线",
    content: "#domId + #domId { border-left: 1px solid #ddd; }",
  },
  {
    id: "text-hover-scale",
    label: "Hover 放大",
    content:
      "#domId {\n  transition: all 0.2s ease;\n}\n\n#domId:hover {\n  transform: scale(1.05);\n}",
  },
  {
    id: "text-active-scale",
    label: "点击缩放",
    content: "#domId:active {\n  transform: scale(0.97);\n}",
  },
];
function buildStylePreset(type: string, id: string, label: string, content: string) {
  return {
    id: `${type}-${id}`,
    label,
    content,
  };
}
const customStylePresetMap: AnyRecord = {
  HorizontalLayout: [
    buildStylePreset(
      "HorizontalLayout",
      "gap",
      "紧凑横排",
      "#domId {\n  gap: 8px;\n  align-items: center;\n}",
    ),
    buildStylePreset(
      "HorizontalLayout",
      "space-between",
      "两端对齐",
      "#domId {\n  justify-content: space-between;\n  align-items: center;\n}",
    ),
    buildStylePreset(
      "HorizontalLayout",
      "toolbar",
      "工具栏容器",
      "#domId {\n  gap: 10px;\n  align-items: center;\n  min-height: 44px;\n  padding: 0 8px;\n}",
    ),
    buildStylePreset(
      "HorizontalLayout",
      "border-default",
      "默认边框",
      "#domId {\n  border: 1px solid #dcdfe6;\n  border-radius: 4px;\n}",
    ),
    buildStylePreset(
      "HorizontalLayout",
      "border-color",
      "边框颜色",
      "#domId {\n  border: 1px solid #409eff;\n}",
    ),
    buildStylePreset(
      "HorizontalLayout",
      "border-radius",
      "圆角容器",
      "#domId {\n  border-radius: 8px;\n  overflow: hidden;\n}",
    ),
    buildStylePreset(
      "HorizontalLayout",
      "card-border",
      "卡片边框",
      "#domId {\n  border: 1px solid #e5e7eb;\n  border-radius: 8px;\n  padding: 12px;\n  background: #ffffff;\n}",
    ),
    buildStylePreset(
      "HorizontalLayout",
      "shadow-soft",
      "柔和阴影",
      "#domId {\n  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.12);\n}",
    ),
    buildStylePreset(
      "HorizontalLayout",
      "shadow-card",
      "卡片阴影",
      "#domId {\n  border: 1px solid #e5e7eb;\n  border-radius: 8px;\n  box-shadow: 0 12px 32px rgba(15, 23, 42, 0.14);\n}",
    ),
    buildStylePreset(
      "HorizontalLayout",
      "shadow-inner",
      "内阴影",
      "#domId {\n  box-shadow: inset 0 1px 3px rgba(15, 23, 42, 0.16);\n}",
    ),
  ],
  VerticalLayout: [
    buildStylePreset(
      "VerticalLayout",
      "gap",
      "紧凑纵排",
      "#domId {\n  gap: 8px;\n  align-items: stretch;\n}",
    ),
    buildStylePreset(
      "VerticalLayout",
      "center",
      "居中纵排",
      "#domId {\n  justify-content: center;\n  align-items: center;\n  gap: 12px;\n}",
    ),
    buildStylePreset(
      "VerticalLayout",
      "panel",
      "面板容器",
      "#domId {\n  gap: 12px;\n  padding: 12px;\n  background: #f7f8fa;\n  border: 1px solid #e5e7eb;\n}",
    ),
    buildStylePreset(
      "VerticalLayout",
      "border-default",
      "默认边框",
      "#domId {\n  border: 1px solid #dcdfe6;\n  border-radius: 4px;\n}",
    ),
    buildStylePreset(
      "VerticalLayout",
      "border-color",
      "边框颜色",
      "#domId {\n  border: 1px solid #409eff;\n}",
    ),
    buildStylePreset(
      "VerticalLayout",
      "border-radius",
      "圆角容器",
      "#domId {\n  border-radius: 8px;\n  overflow: hidden;\n}",
    ),
    buildStylePreset(
      "VerticalLayout",
      "card-border",
      "卡片边框",
      "#domId {\n  border: 1px solid #e5e7eb;\n  border-radius: 8px;\n  padding: 12px;\n  background: #ffffff;\n}",
    ),
    buildStylePreset(
      "VerticalLayout",
      "shadow-soft",
      "柔和阴影",
      "#domId {\n  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.12);\n}",
    ),
    buildStylePreset(
      "VerticalLayout",
      "shadow-card",
      "卡片阴影",
      "#domId {\n  border: 1px solid #e5e7eb;\n  border-radius: 8px;\n  box-shadow: 0 12px 32px rgba(15, 23, 42, 0.14);\n}",
    ),
    buildStylePreset(
      "VerticalLayout",
      "shadow-inner",
      "内阴影",
      "#domId {\n  box-shadow: inset 0 1px 3px rgba(15, 23, 42, 0.16);\n}",
    ),
  ],
  ElContainer: [
    buildStylePreset(
      "ElContainer",
      "region-colors",
      "设置区域颜色",
      "#domId [data-node-type=\"ElHeader\"] {\n  background: #f3f4f6;\n}\n#domId [data-node-type=\"ElAside\"] {\n  background: #eef6ff;\n}\n#domId [data-node-type=\"ElMain\"] {\n  background: #ffffff;\n}\n#domId [data-node-type=\"ElFooter\"] {\n  background: #f9fafb;\n}",
    ),
    buildStylePreset(
      "ElContainer",
      "header-color",
      "设置顶部区域颜色",
      "#domId [data-node-type=\"ElHeader\"] {\n  background: #e8f3ff;\n  color: #1f2937;\n}",
    ),
    buildStylePreset(
      "ElContainer",
      "aside-color",
      "设置左侧区域颜色",
      "#domId [data-node-type=\"ElAside\"] {\n  background: #f0f9ff;\n  color: #1f2937;\n}",
    ),
    buildStylePreset(
      "ElContainer",
      "main-color",
      "设置主内容区颜色",
      "#domId [data-node-type=\"ElMain\"] {\n  background: #ffffff;\n  color: #1f2937;\n}",
    ),
    buildStylePreset(
      "ElContainer",
      "footer-color",
      "设置底部区域颜色",
      "#domId [data-node-type=\"ElFooter\"] {\n  background: #f8fafc;\n  color: #475569;\n}",
    ),
  ],
  FormLayout: [
    buildStylePreset(
      "FormLayout",
      "border-default",
      "默认边框",
      "#domId {\n  border: 1px solid #dcdfe6;\n  border-radius: 4px;\n}",
    ),
    buildStylePreset(
      "FormLayout",
      "border-color",
      "边框颜色",
      "#domId {\n  border: 1px solid #409eff;\n}",
    ),
    buildStylePreset(
      "FormLayout",
      "border-radius",
      "圆角表单",
      "#domId {\n  border-radius: 8px;\n  overflow: hidden;\n}",
    ),
    buildStylePreset(
      "FormLayout",
      "form-card",
      "表单卡片",
      "#domId {\n  border: 1px solid #e5e7eb;\n  border-radius: 8px;\n  padding: 16px;\n  background: #ffffff;\n}",
    ),
    buildStylePreset(
      "FormLayout",
      "shadow-soft",
      "柔和阴影",
      "#domId {\n  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.12);\n}",
    ),
    buildStylePreset(
      "FormLayout",
      "shadow-card",
      "卡片阴影",
      "#domId {\n  border: 1px solid #e5e7eb;\n  border-radius: 8px;\n  box-shadow: 0 12px 32px rgba(15, 23, 42, 0.14);\n}",
    ),
    buildStylePreset(
      "FormLayout",
      "shadow-inner",
      "内阴影",
      "#domId {\n  box-shadow: inset 0 1px 3px rgba(15, 23, 42, 0.16);\n}",
    ),
  ],
  Collapse: [
    buildStylePreset(
      "Collapse",
      "compact",
      "紧凑面板",
      "#domId .el-collapse-item__header {\n  height: 36px;\n  line-height: 36px;\n}\n#domId .el-collapse-item__content {\n  padding-bottom: 0;\n}",
    ),
    buildStylePreset(
      "Collapse",
      "header",
      "标题强调",
      "#domId .el-collapse-item__header {\n  font-weight: 600;\n  color: #1f2937;\n}",
    ),
    buildStylePreset(
      "Collapse",
      "borderless",
      "弱化边框",
      "#domId,\n#domId .el-collapse-item__wrap,\n#domId .el-collapse-item__header {\n  border-color: transparent;\n}",
    ),
  ],
  Image: [
    buildStylePreset(
      "Image",
      "base",
      "容器尺寸：图片/裁切",
      "#domId .el-image { width: 120px; height: 120px; display: inline-block; }\n#domId .el-image__inner { width: 100%; height: 100%; object-fit: cover; }",
    ),
    buildStylePreset(
      "Image",
      "radius",
      "圆角/边框/阴影",
      "#domId .el-image { border-radius: 10px; overflow: hidden; border: 1px solid #ebeef5; }\n#domId .el-image__inner { border-radius: 10px; }",
    ),
    buildStylePreset(
      "Image",
      "hover",
      "Hover（放大/高亮）",
      "#domId .el-image__inner { transition: transform .2s ease, opacity .2s ease; }\n#domId .el-image:hover { cursor: pointer; }\n#domId .el-image:hover .el-image__inner { transform: scale(1.03); opacity: .95; }",
    ),
    buildStylePreset(
      "Image",
      "placeholder",
      "占位/错误",
      "#domId .el-image__placeholder,\n#domId .el-image__error { background:#f5f7fa; color:#909399; font-size:12px; }",
    ),
    buildStylePreset(
      "Image",
      "disabled",
      "禁用（业务 class）",
      "#domId .is-disabled-image { pointer-events: none; opacity: .6; }",
    ),
  ],
  Input: [
    buildStylePreset(
      "Input",
      "base",
      "基础（字体/高度/圆角/placeholder）",
      "#domId .el-input__inner{\n  height:44px; line-height:44px;\n  font-size:16px; color:#303133;\n  border-radius:6px;\n}\n#domId .el-input__inner::placeholder{ color:#c0c4cc; }",
    ),
    buildStylePreset(
      "Input",
      "state",
      "状态（hover/focus）",
      "#domId .el-input__inner:hover{ border-color:#409EFF; }\n#domId .el-input__inner:focus{\n  border-color:#409EFF;\n  box-shadow:0 0 0 2px rgba(64,158,255,.15);\n}",
    ),
    buildStylePreset(
      "Input",
      "children",
      "子元素（前后缀/clear）",
      "#domId .el-input__prefix, #domId .el-input__suffix{ color:#909399; }\n#domId .el-input__clear{ color:#c0c4cc; }\n#domId .el-input__clear:hover{ color:#409EFF; }",
    ),
    buildStylePreset(
      "Input",
      "textarea",
      "Textarea（多行）",
      "#domId .el-textarea__inner{\n  font-size:14px; line-height:1.6;\n  border-radius:8px;\n  padding:10px 12px;\n}\n#domId .el-textarea__inner:focus{\n  border-color:#409EFF;\n  box-shadow:0 0 0 2px rgba(64,158,255,.12);\n}",
    ),
    buildStylePreset(
      "Input",
      "error",
      "错误态（el-form-item.is-error）",
      "#domId .el-form-item.is-error .el-input__inner,\n#domId .el-form-item.is-error .el-textarea__inner{ border-color:#f56c6c; }\n#domId .el-form-item.is-error .el-input__inner:focus{\n  box-shadow:0 0 0 2px rgba(245,108,108,.15);\n}",
    ),
    buildStylePreset(
      "Input",
      "disabled",
      "禁用/只读",
      "#domId .el-input.is-disabled .el-input__inner,\n#domId .el-textarea.is-disabled .el-textarea__inner{\n  background:#f5f7fa; color:#c0c4cc; cursor:not-allowed;\n}",
    ),
    buildStylePreset(
      "Input",
      "autocomplete",
      "Autocomplete 浮层（popper-class=ac-popper）",
      ".ac-popper{ border-radius:12px; overflow:hidden; }\n.ac-popper li{ font-size:14px; line-height:38px; }\n.ac-popper li:hover{ background:#ecf5ff; color:#409EFF; }",
    ),
  ],
  Select: [
    buildStylePreset(
      "Select",
      "base",
      "基础（输入框）",
      "#domId .el-select .el-input__inner{\n  height:44px; line-height:44px;\n  border-radius:6px; font-size:16px;\n}",
    ),
    buildStylePreset(
      "Select",
      "state",
      "状态（hover/focus）",
      "#domId .el-select .el-input__inner:hover{ border-color:#409EFF; }\n#domId .el-select .el-input__inner:focus{\n  border-color:#409EFF;\n  box-shadow:0 0 0 2px rgba(64,158,255,.15);\n}",
    ),
    buildStylePreset(
      "Select",
      "children",
      "子元素（箭头/清空/多选tag）",
      "#domId .el-select .el-input__suffix{ color:#909399; }\n#domId .el-select .el-icon-circle-close{ color:#c0c4cc; }\n#domId .el-select .el-icon-circle-close:hover{ color:#409EFF; }\n\n#domId .el-select .el-tag{\n  border-radius:999px;\n  height:24px; line-height:22px;\n  margin:4px 4px 4px 0;\n}\n#domId .el-select .el-tag__close{ color:#909399; }\n#domId .el-select .el-tag__close:hover{ color:#409EFF; }",
    ),
    buildStylePreset(
      "Select",
      "popper",
      "下拉浮层（popper-class=select-popper）",
      ".select-popper{ border-radius:12px; overflow:hidden; }\n.select-popper .el-select-dropdown__item{ font-size:14px; line-height:38px; }\n.select-popper .el-select-dropdown__item:hover{ background:#ecf5ff; color:#409EFF; }\n.select-popper .el-select-dropdown__item.selected{ color:#409EFF; font-weight:600; }\n.select-popper .el-select-dropdown__empty{ color:#909399; padding:12px 0; }\n.select-popper .el-select-group__title{ color:#909399; font-size:12px; padding:8px 12px; }",
    ),
    buildStylePreset(
      "Select",
      "disabled",
      "禁用",
      "#domId .el-select .el-input.is-disabled .el-input__inner{ background:#f5f7fa; }",
    ),
  ],
  Switch: [
    buildStylePreset(
      "Switch",
      "base",
      "基础（尺寸/圆点）",
      "#domId .el-switch__core{ width:48px!important; height:24px; border-radius:999px; }\n#domId .el-switch__core:after{ width:20px; height:20px; top:1px; }",
    ),
    buildStylePreset(
      "Switch",
      "state",
      "状态（开/关/hover）",
      "#domId .el-switch .el-switch__core{ background:#dcdfe6; border-color:#dcdfe6; }\n#domId .el-switch.is-checked .el-switch__core{ background:#67c23a; border-color:#67c23a; }\n#domId .el-switch:hover{ opacity:.95; }",
    ),
    buildStylePreset(
      "Switch",
      "label",
      "文案（active/inactive）",
      "#domId .el-switch__label{ font-size:14px; }",
    ),
    buildStylePreset(
      "Switch",
      "disabled",
      "禁用",
      "#domId .el-switch.is-disabled{ opacity:.6; cursor:not-allowed; }",
    ),
  ],
  Table: [
    buildStylePreset(
      "Table",
      "base",
      "基础（表头/字体）",
      "#domId .el-table th{ background:#f5f7fa; color:#303133; font-weight:600; }\n#domId .el-table .cell{ font-size:14px; color:#303133; }",
    ),
    buildStylePreset(
      "Table",
      "row-state",
      "行状态（hover/当前行）",
      "#domId .el-table__body tr:hover>td{ background:#ecf5ff!important; }\n#domId .el-table__body tr.current-row>td{ background:#409EFF; color:#fff; }",
    ),
    buildStylePreset(
      "Table",
      "striped",
      "斑马线",
      "#domId .el-table--striped .el-table__body tr.el-table__row--striped td{ background:#fafafa; }",
    ),
    buildStylePreset(
      "Table",
      "ellipsis",
      "省略（show-overflow-tooltip）",
      "#domId .el-table .cell{ white-space:nowrap; overflow:hidden; text-overflow:ellipsis; }",
    ),
    buildStylePreset(
      "Table",
      "fixed-shadow",
      "固定列阴影",
      "#domId .el-table__fixed-right{ box-shadow:-6px 0 10px rgba(0,0,0,.06); }\n#domId .el-table__fixed{ box-shadow:6px 0 10px rgba(0,0,0,.06); }",
    ),
    buildStylePreset(
      "Table",
      "sort-filter",
      "排序/筛选图标",
      "#domId .el-table .caret-wrapper{ height:18px; }\n#domId .el-table .sort-caret{ border-width:5px; }",
    ),
    buildStylePreset(
      "Table",
      "disabled",
      "禁用（业务：整表不可点）",
      "#domId .is-readonly-table{ pointer-events:none; opacity:.75; }",
    ),
  ],
  Tree: [
    buildStylePreset(
      "Tree",
      "base",
      "基础（节点行高/字体）",
      "#domId .el-tree-node__content{ height:36px; font-size:14px; }\n#domId .el-tree-node__label{ color:#303133; }",
    ),
    buildStylePreset(
      "Tree",
      "state",
      "状态（hover/当前）",
      "#domId .el-tree-node__content:hover{ background:#ecf5ff; }\n#domId .el-tree--highlight-current .el-tree-node.is-current>.el-tree-node__content{\n  background:#409EFF; color:#fff;\n}",
    ),
    buildStylePreset(
      "Tree",
      "children",
      "子元素（展开图标/checkbox/缩进）",
      "#domId .el-tree-node__expand-icon{ font-size:14px; color:#909399; }\n#domId .el-tree .el-checkbox{ margin-right:6px; }\n#domId .el-tree-node__content{ padding-right:10px; }",
    ),
    buildStylePreset(
      "Tree",
      "disabled",
      "禁用（业务）",
      "#domId .is-disabled-tree{ pointer-events:none; opacity:.65; }",
    ),
  ],
  Dropdown: [
    buildStylePreset(
      "Dropdown",
      "panel",
      "面板（popper-class=dropdown-popper）",
      ".dropdown-popper{ border-radius:12px; padding:6px 0; overflow:hidden; }",
    ),
    buildStylePreset(
      "Dropdown",
      "item",
      "菜单项（高度/字体）",
      ".dropdown-popper .el-dropdown-menu__item{ font-size:14px; line-height:38px; }",
    ),
    buildStylePreset(
      "Dropdown",
      "state",
      "状态（hover/禁用/分割线）",
      ".dropdown-popper .el-dropdown-menu__item:hover{ background:#ecf5ff; color:#409EFF; }\n.dropdown-popper .el-dropdown-menu__item.is-disabled{ opacity:.5; cursor:not-allowed; }\n.dropdown-popper .el-dropdown-menu__item--divided{ border-top:1px solid #ebeef5; }",
    ),
  ],
  Menu: [
    buildStylePreset(
      "Menu",
      "base",
      "基础弹层样式",
      "#domId .el-menu{ border-right:none; border-radius:12px; overflow:hidden; }",
    ),
    buildStylePreset(
      "Menu",
      "item",
      "菜单项（高度/字体）",
      "#domId .el-menu-item, #domId .el-submenu__title{\n  height:48px; line-height:48px; font-size:14px;\n}",
    ),
    buildStylePreset(
      "Menu",
      "state",
      "状态（hover/激活/禁用）",
      "#domId .el-menu-item:hover, #domId .el-submenu__title:hover{ background:#ecf5ff; }\n#domId .el-menu-item.is-active{ background:#409EFF; color:#fff!important; }\n#domId .el-menu-item.is-disabled{ opacity:.5; cursor:not-allowed; }",
    ),
    buildStylePreset(
      "Menu",
      "submenu",
      "子菜单（缩进/图标）",
      "#domId .el-menu--inline .el-menu-item{ padding-left:48px!important; }\n#domId .el-submenu__icon-arrow{ color:#909399; }",
    ),
  ],
  Radio: [
    buildStylePreset(
      "Radio",
      "base",
      "基础（字体/对齐）",
      "#domId .el-radio__label{ font-size:14px; }\n#domId .el-radio__inner{ width:16px; height:16px; }",
    ),
    buildStylePreset(
      "Radio",
      "checked",
      "选中态",
      "#domId .el-radio__input.is-checked .el-radio__inner{ background:#409EFF; border-color:#409EFF; }\n#domId .el-radio__input.is-checked + .el-radio__label{ color:#409EFF; }",
    ),
    buildStylePreset(
      "Radio",
      "bordered",
      "边框样式 radio",
      "#domId .el-radio.is-bordered{ border-radius:8px; }",
    ),
    buildStylePreset(
      "Radio",
      "disabled",
      "禁用",
      "#domId .el-radio.is-disabled{ opacity:.6; cursor:not-allowed; }",
    ),
  ],
  Checkbox: [
    buildStylePreset(
      "Checkbox",
      "base",
      "基础（字体/大小）",
      "#domId .el-checkbox__label{ font-size:14px; }\n#domId .el-checkbox__inner{ width:16px; height:16px; }",
    ),
    buildStylePreset(
      "Checkbox",
      "checked",
      "选中态",
      "#domId .el-checkbox__input.is-checked .el-checkbox__inner{ background:#409EFF; border-color:#409EFF; }\n#domId .el-checkbox__input.is-checked + .el-checkbox__label{ color:#409EFF; }",
    ),
    buildStylePreset(
      "Checkbox",
      "button",
      "checkbox-button",
      "#domId .el-checkbox-button__inner{ border-radius:8px; }\n#domId .el-checkbox-button.is-checked .el-checkbox-button__inner{\n  background:#409EFF; border-color:#409EFF;\n}",
    ),
    buildStylePreset(
      "Checkbox",
      "disabled",
      "禁用",
      "#domId .el-checkbox.is-disabled{ opacity:.6; cursor:not-allowed; }",
    ),
  ],
  Cascader: [
    buildStylePreset(
      "Cascader",
      "input",
      "输入框",
      "#domId .el-cascader .el-input__inner{\n  height:44px; line-height:44px; border-radius:6px; font-size:16px;\n}",
    ),
    buildStylePreset(
      "Cascader",
      "panel",
      "面板（popper-class=cascader-popper）",
      ".cascader-popper{ border-radius:12px; overflow:hidden; }",
    ),
    buildStylePreset(
      "Cascader",
      "node",
      "节点（hover/active/disabled）",
      ".cascader-popper .el-cascader-node{ font-size:14px; }\n.cascader-popper .el-cascader-node:hover{ background:#ecf5ff; }\n.cascader-popper .el-cascader-node.is-active{ color:#409EFF; font-weight:600; }\n.cascader-popper .el-cascader-node.is-disabled{ opacity:.5; }",
    ),
    buildStylePreset(
      "Cascader",
      "disabled",
      "禁用",
      "#domId .el-cascader.is-disabled{ opacity:.6; }",
    ),
  ],
  Tabs: [
    buildStylePreset(
      "Tabs",
      "base",
      "基础（字体/高度）",
      "#domId .el-tabs__item{ font-size:14px; height:44px; line-height:44px; }",
    ),
    buildStylePreset(
      "Tabs",
      "state",
      "状态（hover/active/disabled）",
      "#domId .el-tabs__item:hover{ color:#66b1ff; }\n#domId .el-tabs__item.is-active{ color:#409EFF; font-weight:600; }\n#domId .el-tabs__item.is-disabled{ opacity:.5; cursor:not-allowed; }",
    ),
    buildStylePreset("Tabs", "bar", "指示条", "#domId .el-tabs__active-bar{ background:#409EFF; }"),
    buildStylePreset(
      "Tabs",
      "card",
      "card 模式",
      "#domId .el-tabs--card > .el-tabs__header .el-tabs__item{ border-radius:10px 10px 0 0; }",
    ),
  ],
  Transfer: [
    buildStylePreset(
      "Transfer",
      "panel",
      "面板/标题",
      "#domId .el-transfer-panel{ border-radius:12px; overflow:hidden; }\n#domId .el-transfer-panel__header{ background:#f5f7fa; font-weight:600; }",
    ),
    buildStylePreset(
      "Transfer",
      "item",
      "列表项（hover/禁用）",
      "#domId .el-transfer-panel__item{ font-size:14px; }\n#domId .el-transfer-panel__item:hover{ background:#ecf5ff; }\n#domId .el-transfer-panel__item.is-disabled{ opacity:.5; }",
    ),
    buildStylePreset(
      "Transfer",
      "search",
      "搜索框（filterable）",
      "#domId .el-transfer-panel__filter .el-input__inner{ height:38px; line-height:38px; border-radius:8px; }",
    ),
    buildStylePreset(
      "Transfer",
      "button",
      "中间按钮",
      "#domId .el-transfer__buttons .el-button{ border-radius:10px; }",
    ),
  ],
  Tag: [
    buildStylePreset(
      "Tag",
      "base",
      "基础（圆角/字号）",
      "#domId .el-tag{ border-radius:999px; font-size:12px; height:24px; line-height:22px; }",
    ),
    buildStylePreset(
      "Tag",
      "type",
      "类型柔和色",
      "#domId .el-tag--success{ background:#f0f9eb; color:#67c23a; border-color:#e1f3d8; }\n#domId .el-tag--warning{ background:#fdf6ec; color:#e6a23c; border-color:#faecd8; }\n#domId .el-tag--danger { background:#fef0f0; color:#f56c6c; border-color:#fde2e2; }",
    ),
    buildStylePreset(
      "Tag",
      "close",
      "关闭按钮 close",
      "#domId .el-tag .el-tag__close{ color:#909399; }\n#domId .el-tag .el-tag__close:hover{ color:#409EFF; background:transparent; }",
    ),
  ],
  InputNumber: [
    buildStylePreset(
      "InputNumber",
      "input",
      "输入框",
      "#domId .el-input-number .el-input__inner{\n  height:44px; line-height:44px; border-radius:6px; font-size:16px;\n}",
    ),
    buildStylePreset(
      "InputNumber",
      "control",
      "加减按钮（hover/禁用）",
      "#domId .el-input-number__increase,\n#domId .el-input-number__decrease{ border-radius:8px; }\n#domId .el-input-number__increase:hover,\n#domId .el-input-number__decrease:hover{ color:#409EFF; }",
    ),
    buildStylePreset(
      "InputNumber",
      "disabled",
      "禁用",
      "#domId .el-input-number.is-disabled{ opacity:.6; }",
    ),
  ],
  Timeline: [
    buildStylePreset(
      "Timeline",
      "timestamp",
      "时间戳",
      "#domId .el-timeline-item__timestamp{ font-size:12px; color:#909399; }",
    ),
    buildStylePreset(
      "Timeline",
      "node",
      "节点",
      "#domId .el-timeline-item__node{ width:14px; height:14px; }",
    ),
    buildStylePreset(
      "Timeline",
      "content",
      "内容卡片化",
      "#domId .el-timeline-item__content{\n  background:#f5f7fa;\n  padding:10px 12px;\n  border-radius:10px;\n}",
    ),
  ],
  ImageCarousel: [
    buildStylePreset(
      "ImageCarousel",
      "container",
      "容器/圆角",
      "#domId .el-carousel__container{ border-radius:12px; overflow:hidden; }",
    ),
    buildStylePreset(
      "ImageCarousel",
      "arrow",
      "箭头",
      "#domId .el-carousel__arrow{ background:rgba(0,0,0,.35); }\n#domId .el-carousel__arrow:hover{ opacity:.9; }",
    ),
    buildStylePreset(
      "ImageCarousel",
      "indicator",
      "指示器",
      "#domId .el-carousel__indicator button{ border-radius:999px; }",
    ),
  ],
  CarouselComponent: [
    buildStylePreset(
      "CarouselComponent",
      "container",
      "容器/圆角",
      "#domId .el-carousel__container{ border-radius:12px; overflow:hidden; }",
    ),
    buildStylePreset(
      "CarouselComponent",
      "arrow",
      "箭头",
      "#domId .el-carousel__arrow{ background:rgba(0,0,0,.35); }",
    ),
  ],
  WebContainer: [
    buildStylePreset(
      "WebContainer",
      "frame",
      "边框/圆角",
      "#domId .web-container{ border:1px solid #ebeef5; border-radius:12px; overflow:hidden; background:#fff; }",
    ),
    buildStylePreset(
      "WebContainer",
      "iframe",
      "iframe",
      "#domId .web-container iframe{ width:100%; height:600px; border:0; display:block; }",
    ),
    buildStylePreset(
      "WebContainer",
      "hover",
      "Hover 阴影",
      "#domId .web-container:hover{ box-shadow:0 8px 24px rgba(0,0,0,.08); }",
    ),
  ],
  Steps: [
    buildStylePreset(
      "Steps",
      "text",
      "标题/描述",
      "#domId .el-step__title{ font-size:14px; }\n#domId .el-step__description{ font-size:12px; color:#909399; }",
    ),
    buildStylePreset(
      "Steps",
      "status",
      "过程/完成色",
      "#domId .el-step__head.is-process,\n#domId .el-step__title.is-process{ color:#409EFF; border-color:#409EFF; }\n\n#domId .el-step__head.is-finish,\n#domId .el-step__title.is-finish{ color:#67c23a; border-color:#67c23a; }",
    ),
  ],
  Card: [
    buildStylePreset(
      "Card",
      "base",
      "外观",
      "#domId .el-card{ border-radius:12px; overflow:hidden; }\n#domId .el-card__header{ background:#fafafa; font-weight:600; }",
    ),
    buildStylePreset(
      "Card",
      "hover",
      "Hover",
      "#domId .el-card:hover{ box-shadow:0 8px 24px rgba(0,0,0,.08); }",
    ),
  ],
  Pagination: [
    buildStylePreset(
      "Pagination",
      "size",
      "按钮/页码",
      "#domId .el-pagination .btn-prev,\n#domId .el-pagination .btn-next,\n#domId .el-pagination .el-pager li{ border-radius:8px; }\n#domId .el-pagination .el-pager li{ min-width:32px; height:32px; line-height:32px; }",
    ),
    buildStylePreset(
      "Pagination",
      "state",
      "Active / Hover",
      "#domId .el-pagination .el-pager li.active{ background:#409EFF; color:#fff; }\n#domId .el-pagination .el-pager li:hover{ color:#409EFF; }",
    ),
  ],
  BigDataTable: [
    buildStylePreset(
      "BigDataTable",
      "wrapper",
      "容器/滚动区域",
      "#domId .big-table{ height:600px; border:1px solid #ebeef5; border-radius:12px; overflow:hidden; background:#fff; }\n#domId .big-table .viewport{ height:100%; overflow:auto; }",
    ),
    buildStylePreset(
      "BigDataTable",
      "row",
      "行/单元格",
      "#domId .big-table .row{ height:44px; display:flex; align-items:center; }\n#domId .big-table .cell{ font-size:14px; color:#303133; padding:0 12px; }\n#domId .big-table .row:hover{ background:#ecf5ff; }",
    ),
    buildStylePreset(
      "BigDataTable",
      "header",
      "表头",
      "#domId .big-table .header{ background:#f5f7fa; font-weight:600; }",
    ),
  ],
  BusinessCard: [
    buildStylePreset(
      "BusinessCard",
      "base",
      "容器/排版",
      "#domId .biz-card{ border:1px solid #ebeef5; border-radius:14px; padding:16px; background:#fff; }\n#domId .biz-card .title{ font-size:14px; color:#909399; }\n#domId .biz-card .value{ font-size:28px; font-weight:700; color:#303133; }",
    ),
    buildStylePreset(
      "BusinessCard",
      "state",
      "Hover / 状态色",
      "#domId .biz-card:hover{ box-shadow:0 8px 24px rgba(0,0,0,.08); }\n#domId .biz-card.success{ border-color:#e1f3d8; }\n#domId .biz-card.danger{ border-color:#fde2e2; }",
    ),
  ],
  Barcode: [
    buildStylePreset(
      "Barcode",
      "frame",
      "边框/画布",
      "#domId .barcode{ padding:10px; background:#fff; border:1px dashed #dcdfe6; border-radius:12px; }\n#domId .barcode svg, #domId .barcode canvas{ display:block; margin:0 auto; }",
    ),
    buildStylePreset(
      "Barcode",
      "text",
      "文本（编号）",
      "#domId .barcode .text{ font-size:12px; color:#303133; text-align:center; margin-top:6px; }",
    ),
  ],
  Slider: [
    buildStylePreset(
      "Slider",
      "track",
      "轨道/进度",
      "#domId .el-slider__runway{ height:6px; border-radius:999px; }\n#domId .el-slider__bar{ background:#409EFF; border-radius:999px; }",
    ),
    buildStylePreset(
      "Slider",
      "thumb",
      "圆点/hover",
      "#domId .el-slider__button{ border:2px solid #409EFF; transition:.15s; }\n#domId .el-slider__button:hover{ transform:scale(1.05); }",
    ),
    buildStylePreset(
      "Slider",
      "tooltip",
      "Tooltip（效果显示）",
      "#domId .el-slider__button-wrapper{ font-size:12px; }",
    ),
  ],
  Calendar: [
    buildStylePreset(
      "Calendar",
      "base",
      "外观",
      "#domId .el-calendar{ border-radius:12px; overflow:hidden; }\n#domId .el-calendar__header{ font-weight:600; }",
    ),
    buildStylePreset(
      "Calendar",
      "state",
      "Hover / 今日",
      "#domId .el-calendar-table td:hover{ background:#ecf5ff; }\n#domId .el-calendar-table td.is-today{ background:#409EFF; color:#fff; }",
    ),
  ],
  Signature: [
    buildStylePreset(
      "Signature",
      "frame",
      "边框/画布",
      "#domId .signature{ border:1px solid #dcdfe6; border-radius:12px; overflow:hidden; background:#fff; }\n#domId .signature canvas{ width:100%; height:240px; display:block; }",
    ),
    buildStylePreset(
      "Signature",
      "actions",
      "操作区（清空/保存）",
      "#domId .signature .actions{ padding:10px; border-top:1px solid #ebeef5; text-align:right; }",
    ),
    buildStylePreset(
      "Signature",
      "disabled",
      "禁用",
      "#domId .signature.is-disabled{ pointer-events:none; opacity:.7; }",
    ),
  ],
};
const detailPresets: AnyArray = [
  {
    id: "empty",
    label: "空模板",
    content: "// 在这里编写详细配置脚本\n",
  },
  {
    id: "comment",
    label: "注释模板",
    content: "/* id为组件的唯一标识 */\n",
  },
];
const echartDetailPresets: AnyArray = [
  {
    id: "echart-line",
    label: "折线图模板",
    content:
      'const option = {\n  title: { text: "折线图" },\n  tooltip: { trigger: "axis" },\n  xAxis: { type: "category", data: ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"] },\n  yAxis: { type: "value" },\n  series: [{ name: "访问量", type: "line", data: [120, 200, 150, 80, 70, 110, 130] }],\n};\nreturn option;',
  },
  {
    id: "echart-bar",
    label: "柱状图模板",
    content:
      'const option = {\n  title: { text: "柱状图" },\n  tooltip: { trigger: "axis" },\n  xAxis: { type: "category", data: ["A", "B", "C", "D", "E"] },\n  yAxis: { type: "value" },\n  series: [{ type: "bar", data: [12, 20, 15, 8, 25] }],\n};\nreturn option;',
  },
  {
    id: "echart-pie",
    label: "饼图模板",
    content:
      'const option = {\n  title: { text: "饼图", left: "center" },\n  tooltip: { trigger: "item" },\n  legend: { bottom: 0 },\n  series: [{ type: "pie", radius: ["30%", "70%"], data: [\n    { value: 1048, name: "A" },\n    { value: 735, name: "B" },\n    { value: 580, name: "C" },\n    { value: 484, name: "D" },\n  ] }],\n};\nreturn option;',
  },
  {
    id: "echart-radar",
    label: "雷达图模板",
    content:
      'const option = {\n  title: { text: "雷达图" },\n  tooltip: {},\n  radar: {\n    indicator: [\n      { name: "指标A", max: 100 },\n      { name: "指标B", max: 100 },\n      { name: "指标C", max: 100 },\n      { name: "指标D", max: 100 },\n    ],\n  },\n  series: [{ type: "radar", data: [{ value: [80, 90, 60, 70], name: "数据" }] }],\n};\nreturn option;',
  },
  {
    id: "echart-scatter",
    label: "散点图模板",
    content:
      'const option = {\n  title: { text: "散点图" },\n  xAxis: {},\n  yAxis: {},\n  series: [{ type: "scatter", data: [[10, 8], [12, 16], [15, 12], [20, 6]] }],\n};\nreturn option;',
  },
  {
    id: "echart-gauge",
    label: "仪表盘模板",
    content:
      'const option = {\n  title: { text: "仪表盘" },\n  series: [{\n    type: "gauge",\n    progress: { show: true },\n    detail: { valueAnimation: true, formatter: "{value}%" },\n    data: [{ value: 50, name: "完成率" }],\n  }],\n};\nreturn option;',
  },
  {
    id: "echart-area",
    label: "面积折线图",
    content:
      'const option = {\n  title: { text: "面积折线" },\n  tooltip: { trigger: "axis" },\n  xAxis: { type: "category", data: ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"] },\n  yAxis: { type: "value" },\n  series: [{ type: "line", areaStyle: {}, data: [220, 182, 191, 234, 290, 330, 310] }],\n};\nreturn option;',
  },
  {
    id: "echart-stacked-bar",
    label: "堆叠柱状图",
    content:
      'const option = {\n  title: { text: "堆叠柱状" },\n  tooltip: { trigger: "axis" },\n  legend: { bottom: 0 },\n  xAxis: { type: "category", data: ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"] },\n  yAxis: { type: "value" },\n  series: [\n    { name: "A", type: "bar", stack: "total", data: [120, 132, 101, 134, 90, 230, 210] },\n    { name: "B", type: "bar", stack: "total", data: [220, 182, 191, 234, 290, 330, 310] },\n  ],\n};\nreturn option;',
  },
  {
    id: "echart-donut",
    label: "环形饼图",
    content:
      'const option = {\n  title: { text: "环形饼图", left: "center" },\n  tooltip: { trigger: "item" },\n  legend: { bottom: 0 },\n  series: [{\n    type: "pie",\n    radius: ["45%", "70%"],\n    data: [\n      { value: 335, name: "A" },\n      { value: 310, name: "B" },\n      { value: 234, name: "C" },\n      { value: 135, name: "D" },\n    ],\n  }],\n};\nreturn option;',
  },
  {
    id: "echart-multi-line",
    label: "多折线图",
    content:
      'const option = {\n  title: { text: "多折线" },\n  tooltip: { trigger: "axis" },\n  legend: { bottom: 0 },\n  xAxis: { type: "category", data: ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"] },\n  yAxis: { type: "value" },\n  series: [\n    { name: "A", type: "line", data: [120, 132, 101, 134, 90, 230, 210] },\n    { name: "B", type: "line", data: [220, 182, 191, 234, 290, 330, 310] },\n  ],\n};\nreturn option;',
  },
];
const buttonDetailPresets: AnyArray = [
  {
    id: "btn-full",
    label: "通用模板（全字段）",
    content:
      "this.button({\n  text: '按钮',\n  type: 'default',\n  size: 'small',\n  plain: false,\n  round: false,\n  circle: false,\n  loading: false,\n  disabled: false,\n  autofocus: false,\n\n  icon: '',\n  'prefix-icon': '',\n  'suffix-icon': '',\n\n  click: 'onClick',\n  confirm: '',\n  confirmType: 'warning',\n  confirmTitle: '提示',\n  throttle: 0,\n  debounce: 0,\n  perms: '',\n  show: true,\n  if: true,\n\n  block: false,\n  width: null,\n});",
  },
  {
    id: "btn-basic",
    label: "按钮",
    content: "this.button({\n  text: '保存',\n  type: 'primary',\n  click: 'onSave',\n});",
  },
  {
    id: "btn-text",
    label: "文字按钮",
    content: "this.button({\n  text: '查看详情',\n  type: 'text',\n  click: 'onView',\n});",
  },
  {
    id: "btn-types",
    label: "不同类型按钮",
    content:
      "this.button({ text:'主要', type:'primary', click:'onPrimary' })\n// 示例: 成功 type='success' click='onSuccess'\n// 示例: 警告 type='warning' click='onWarning'\n// 示例: 危险 type='danger' click='onDanger'\n// 示例: 信息 type='info' click='onInfo'",
  },
  {
    id: "btn-plain",
    label: "朴素按钮",
    content:
      "this.button({\n  text: '取消',\n  type: 'primary',\n  plain: true,\n  click: 'onCancel',\n});",
  },
  {
    id: "btn-state",
    label: "禁用 / 加载中",
    content:
      "this.button({\n  text: '提交',\n  type: 'primary',\n  disabled: true,\n})\n\n// 示例: loading=true text='提交中' type='primary'",
  },
  {
    id: "btn-size",
    label: "不同尺寸",
    content:
      "this.button({ text:'默认', type:'primary' })\n// 示例: size='medium' text='中等'\n// 示例: size='small' text='小'\n// 示例: size='mini' text='迷你'",
  },
  {
    id: "btn-round",
    label: "圆角 / 圆形",
    content:
      "this.button({\n  text: '圆角按钮',\n  type: 'primary',\n  round: true,\n})\n\n// 示例: circle=true icon='el-icon-search' click='onSearch'",
  },
  {
    id: "btn-icon",
    label: "图标按钮",
    content:
      "this.button({\n  text: '新增',\n  type: 'primary',\n  icon: 'el-icon-plus',\n  click: 'onAdd',\n})\n\n// 示例: text='下载' type='success' icon='el-icon-download' click='onDownload'",
  },
  {
    id: "btn-loading",
    label: "加载状态按钮",
    content:
      "this.button({\n  text: '提交中',\n  type: 'primary',\n  loading: true,\n  click: 'onSubmit',\n})\n\n// 示例: text='保存' type='success' loading=true click='onSave'",
  },
  {
    id: "btn-confirm",
    label: "二次确认",
    content:
      "this.button({\n  text: '删除',\n  type: 'danger',\n  click: 'onDelete',\n  confirm: '确定删除该条数据吗？',\n  confirmTitle: '删除确认',\n  confirmType: 'warning',\n})",
  },
  {
    id: "btn-throttle",
    label: "节流",
    content:
      "this.button({\n  text: '发送验证码',\n  type: 'primary',\n  click: 'sendCode',\n  throttle: 1500,\n})",
  },
  {
    id: "btn-debounce",
    label: "防抖",
    content:
      "this.button({\n  text: '搜索',\n  type: 'primary',\n  click: 'onSearch',\n  debounce: 300,\n})",
  },
  {
    id: "btn-perms",
    label: "权限按钮",
    content:
      "this.button({\n  text: '导出',\n  type: 'primary',\n  icon: 'el-icon-download',\n  perms: 'order:export',\n  click: 'exportOrder',\n})",
  },
  {
    id: "btn-visibility",
    label: "条件显示",
    content:
      "this.button({\n  text: '管理员按钮',\n  type: 'primary',\n  if: 'user.isAdmin === true',\n  click: 'adminAction',\n})\n\n// 示例: show='form.status === 1' text='根据状态显示/隐藏' click='statusAction'",
  },
];

const elementPlusTypes = new Set<string>([
  "Input",
  "InputNumber",
  "Select",
  "Switch",
  "Table",
  "BigDataTable",
  "Tree",
  "Dropdown",
  "Menu",
  "Radio",
  "Checkbox",
  "Cascader",
  "Tabs",
  "Transfer",
  "Tag",
  "Timeline",
  "ImageCarousel",
  "CarouselComponent",
  "Steps",
  "Card",
  "Pagination",
  "Collapse",
  "Slider",
  "Calendar",
  "ElContainer",
  "ElHeader",
  "ElAside",
  "ElMain",
  "ElFooter",
  "ElLayout",
  "ElLayoutRow",
  "ElCol",
]);

/**
 * 是否为 Element Plus 组件
 * @param {string | undefined} type - 组件类型
 * @returns {boolean}
 */
function isElementPlusType(type: string | undefined) {
  const normalized = elementTypeName(type);
  if (!normalized) return false;
  if (normalized.startsWith("El")) return true;
  return elementPlusTypes.has(normalized);
}

/**
 * 获取 Element Plus 组件定义
 * @param {string | undefined} type - 组件类型
 * @returns {any}
 */
function resolveElementPlusComponent(type: string | undefined) {
  const normalized = elementTypeName(type);
  if (!normalized) return null;
  const map: AnyRecord = {
    Input: ElInput,
    InputNumber: ElInputNumber,
    Select: ElSelect,
    Switch: ElSwitch,
    Table: ElTable,
    BigDataTable: ElTable,
    Dropdown: ElDropdown,
    Radio: ElRadioGroup,
    Checkbox: ElCheckboxGroup,
    Cascader: ElCascader,
    Image: ElImage,
    Tabs: ElTabs,
    Timeline: ElTimeline,
    ImageCarousel: ElCarousel,
    CarouselComponent: ElCarousel,
    WebContainer: ElCard,
    Card: ElCard,
    BusinessCard: ElCard,
    Steps: ElSteps,
    Pagination: ElPagination,
    Collapse: ElCollapse,
    Slider: ElSlider,
    Calendar: ElCalendar,
    Signature: ElInput,
    ElContainer,
    ElHeader,
    ElAside,
    ElMain,
    ElFooter,
    ElLayoutRow: ElRow,
    ElCol,
    Transfer: ElTransfer,
  };
  return map[normalized] || null;
}

/**
 * 构建 Element Plus 组件的属性定义
 * @param {string | undefined} type - 组件类型
 * @returns {Array<{ name: string, type: string, label: string, group: string, defaultValue?: any, options?: Array<{label: string, value: any}> }>}
 */
const elementPlusPropLabelMap: AnyRecord = {
  Input: {
    modelValue: "输入值",
    placeholder: "占位符",
    type: "类型",
    disabled: "禁用",
    readonly: "只读",
    clearable: "可清空",
    showPassword: "显示密码",
    maxlength: "最大长度",
    minlength: "最小长度",
    rows: "行数",
    autosize: "自适应高度",
  },
  Select: {
    modelValue: "选中值",
    placeholder: "占位符",
    multiple: "多选",
    clearable: "可清空",
    filterable: "可筛选",
    disabled: "禁用",
    collapseTags: "折叠标签",
    collapseTagsTooltip: "折叠提示",
    multipleLimit: "多选上限",
    emptyValues: "空值列表",
    valueKey: "值字段",
  },
  Table: {
    data: "数据",
    columns: "列配置",
    stripe: "斑马线",
    border: "边框",
    size: "尺寸",
    height: "高度",
    maxHeight: "最大高度",
    rowKey: "行键",
    highlightCurrentRow: "高亮当前行",
    showHeader: "显示表头",
    fit: "列宽自适应",
    emptyText: "空提示",
  },
  BigDataTable: {
    data: "数据",
    columns: "列配置",
    stripe: "斑马线",
    border: "边框",
    size: "尺寸",
    height: "高度",
    maxHeight: "最大高度",
    rowKey: "行键",
    highlightCurrentRow: "高亮当前行",
    showHeader: "显示表头",
    fit: "列宽自适应",
    emptyText: "空提示",
  },
  Dropdown: {
    trigger: "触发方式",
    placement: "弹出位置",
    hideOnClick: "点击关闭",
    splitButton: "分裂按钮",
    disabled: "禁用",
  },
  Radio: {
    modelValue: "选中值",
    size: "尺寸",
    disabled: "禁用",
  },
  Checkbox: {
    modelValue: "选中值",
    size: "尺寸",
    disabled: "禁用",
    min: "最少选中",
    max: "最多选中",
  },
  Cascader: {
    modelValue: "选中值",
    options: "级联数据",
    placeholder: "占位符",
    clearable: "可清空",
    filterable: "可筛选",
    disabled: "禁用",
    showAllLevels: "显示全路径",
    collapseTags: "折叠标签",
    separator: "分隔符",
    props: "配置",
  },
  Transfer: {
    modelValue: "选中值",
    data: "数据",
    filterable: "可筛选",
    filterPlaceholder: "搜索占位",
    titles: "标题",
    buttonTexts: "按钮文本",
    props: "配置",
  },
  InputNumber: {
    modelValue: "数值",
    min: "最小值",
    max: "最大值",
    step: "步长",
    precision: "精度",
    controls: "显示控制",
    controlsPosition: "控制位置",
    disabled: "禁用",
  },
  Image: {
    src: "图片地址",
    fit: "填充模式",
    alt: "替代文本",
    lazy: "懒加载",
    previewSrcList: "预览列表",
    initialIndex: "预览索引",
  },
  ImageCarousel: {
    height: "高度",
    autoplay: "自动播放",
    interval: "间隔",
    indicatorPosition: "指示器位置",
    arrow: "箭头显示",
    loop: "循环",
  },
  CarouselComponent: {
    height: "高度",
    autoplay: "自动播放",
    interval: "间隔",
    indicatorPosition: "指示器位置",
    arrow: "箭头显示",
    loop: "循环",
  },
  WebContainer: {
    header: "头部",
    shadow: "阴影",
    bodyStyle: "内容样式",
  },
  Card: {
    header: "头部",
    shadow: "阴影",
    bodyStyle: "内容样式",
  },
  BusinessCard: {
    header: "头部",
    shadow: "阴影",
    bodyStyle: "内容样式",
  },
  Collapse: {
    modelValue: "展开项",
    accordion: "手风琴",
  },
  Slider: {
    modelValue: "数值",
    min: "最小值",
    max: "最大值",
    step: "步长",
    showInput: "显示输入框",
    range: "范围选择",
    vertical: "竖向",
    height: "高度",
    disabled: "禁用",
  },
  Signature: {
    modelValue: "签名数据",
    disabled: "禁用",
    readonly: "只读",
    placeholder: "占位符",
  },
};

function buildElementPlusPropDefs(type: string | undefined): AnyArray {
  const comp = resolveElementPlusComponent(type);
  const rawProps: AnyRecord | AnyArray | null = comp?.props || comp?.__props || null;
  if (!rawProps || typeof rawProps !== "object") return [];
  const labelMap: AnyRecord | null = elementPlusPropLabelMap[elementTypeName(type)] || null;
  if (!labelMap) return [];
  const entries = (
    Array.isArray(rawProps)
      ? rawProps.map((key: string) => [key, {}])
      : Object.entries(rawProps as AnyRecord)
  ) as Array<[string, AnyRecord]>;

  const resolvePropType = (prop: AnyRecord) => {
    const typeDef = prop?.type;
    const typeValue = Array.isArray(typeDef) ? typeDef[0] : typeDef;
    if (typeValue === Boolean) return "boolean";
    if (typeValue === Number) return "number";
    if (typeValue === Array) return "array";
    if (typeValue === Object) return "object";
    return "string";
  };

  return entries
    .filter(([name]: [string, AnyRecord]) => Object.hasOwn(labelMap, name))
    .map(([name, prop]: [string, AnyRecord]) => {
      const typeValue = resolvePropType(prop);
      const def = typeof prop?.default === "function" ? prop.default() : prop?.default;
      const options = Array.isArray(prop?.values)
        ? prop.values.map((value) => ({ label: String(value), value }))
        : undefined;
      return {
        name,
        type: options ? "enum" : typeValue,
        label: labelMap[name] || name,
        group: "官方属性",
        defaultValue: def,
        options,
      };
    });
}

/**
 * 是否显示属性绑定入口
 * @param {{ group?: string }} propDef - 属性定义
 * @returns {boolean}
 */
function shouldShowBindButton(propDef: AnyRecord) {
  if (!propDef) return false;
  return propDef.bindable === true;
}

/**
 * 获取 DSL 方法名
 * @param {string} type - 组件类型
 * @returns {string}
 */
function getDslMethodName(type: string) {
  if (!type) return "component";
  const normalized = type.replace(DSL_EL_PREFIX_RE, "el");
  return normalized.charAt(0).toLowerCase() + normalized.slice(1);
}

/**
 * 生成预设 ID
 * @param {string} type - 组件类型
 * @param {string} key - 预设标识
 * @returns {string}
 */
function buildPresetId(type: string, key: string) {
  return `${String(type || "component").toLowerCase()}-${key}`;
}

/**
 * 生成 DSL 模板
 * @param {string} methodName - 方法名
 * @param {string} body - 模板主体
 * @returns {string}
 */
function buildDslTemplate(methodName: string, body: string) {
  return `this.${methodName}({\n${body}\n});`;
}

function buildCommonDetailPresets(type: string | undefined): AnyArray {
  const normalizedType = elementTypeName(type) || "component";
  const methodName = getDslMethodName(normalizedType);
  const presets: AnyArray = [
    {
      id: buildPresetId(normalizedType, "hover-select"),
      label: "鼠标移入选中",
      content: buildDslTemplate(
        methodName,
        "  hoverSelect: true,",
      ),
    },
  ];

  presets.push(
    {
      id: buildPresetId(normalizedType, "add-menu"),
      label: "添加菜单",
      content:
        "this.menu({\n  mode: 'horizontal',\n  defaultActive: 'home',\n  items: [\n    { label: '首页', index: 'home' },\n    { label: '数据', index: 'data' },\n    { label: '设置', index: 'settings' },\n  ],\n});",
    },
    {
      id: buildPresetId(normalizedType, "header"),
      label: "头部设置",
      content: buildDslTemplate(
        methodName,
        "  height: '64px',\n  style: {\n    backgroundColor: '#ffffff',\n    borderBottom: '1px solid #e5e7eb',\n    display: 'flex',\n    alignItems: 'center',\n    padding: '0 16px',\n  },",
      ),
    },
    {
      id: buildPresetId(normalizedType, "panel-title"),
      label: "修改面板标题",
      content: buildDslTemplate(
        "collapse",
        "  modelValue: 'panel1',\n  items: [\n    { name: 'panel1', title: '基础信息', content: '' },\n    { name: 'panel2', title: '更多设置', content: '' },\n  ],",
      ),
    },
  );

  return presets;
}

function withCommonDetailPresets(type: string | undefined, presets: AnyArray): AnyArray {
  const existing = new Set((presets || []).map((item: AnyRecord) => String(item?.id || "")));
  return [
    ...buildCommonDetailPresets(type).filter(
      (item: AnyRecord) => !existing.has(String(item.id || "")),
    ),
    ...(presets || []),
  ];
}

const textDetailPresets: AnyArray = [
  {
    id: "text-basic",
    label: "基础文本",
    content: buildDslTemplate("text", "  value: '这是一段文本',"),
  },
  {
    id: "text-bind",
    label: "绑定字段",
    content: buildDslTemplate("text", "  text: 'username',"),
  },
  {
    id: "text-empty",
    label: "占位文本",
    content: buildDslTemplate("text", "  text: 'remark',\n  empty: '--',"),
  },
  {
    id: "text-style",
    label: "样式",
    content: buildDslTemplate(
      "text",
      "  value: '重要信息',\n  color: '#f56c6c',\n  size: 14,\n  weight: 'bold',",
    ),
  },
  {
    id: "text-wrap",
    label: "多行文本",
    content: buildDslTemplate("text", "  text: 'description',\n  lineHeight: 1.6,\n  wrap: true,"),
  },
  {
    id: "text-ellipsis",
    label: "超出省略",
    content: buildDslTemplate("text", "  text: 'content',\n  ellipsis: true,\n  maxWidth: 200,"),
  },
  {
    id: "text-ellipsis-tooltip",
    label: "省略 + Tooltip",
    content: buildDslTemplate(
      "text",
      "  text: 'content',\n  ellipsis: true,\n  tooltip: true,\n  maxWidth: 200,",
    ),
  },
  {
    id: "text-dict",
    label: "状态文本映射",
    content: buildDslTemplate(
      "text",
      "  text: 'status',\n  dict: {\n    0: '禁用',\n    1: '启用',\n  },",
    ),
  },
  {
    id: "text-dict-color",
    label: "+ 颜色",
    content: buildDslTemplate(
      "text",
      "  text: 'status',\n  dict: {\n    0: { label: '禁用', color: '#909399' },\n    1: { label: '启用', color: '#67c23a' },\n  },",
    ),
  },
  {
    id: "text-tag",
    label: "Tag 文本",
    content: buildDslTemplate(
      "text",
      "  text: 'status',\n  tag: true,\n  dict: {\n    0: { label: '禁用', type: 'info' },\n    1: { label: '启用', type: 'success' },\n  },",
    ),
  },
  {
    id: "text-link",
    label: "可点击文本",
    content: buildDslTemplate(
      "text",
      "  value: '查看详情',\n  link: true,\n  type: 'primary',\n  click: 'onViewDetail',",
    ),
  },
  {
    id: "text-tooltip",
    label: "Tooltip 文本",
    content: buildDslTemplate("text", "  value: '鼠标移上来',\n  tooltip: '这是提示',"),
  },
  {
    id: "text-copy",
    label: "复制文本",
    content: buildDslTemplate("text", "  text: 'orderNo',\n  copy: true,"),
  },
  {
    id: "text-format-money",
    label: "金额格式化",
    content: buildDslTemplate(
      "text",
      "  text: 'price',\n  format: 'money',\n  precision: 2,\n  unit: '元',",
    ),
  },
  {
    id: "text-format-datetime",
    label: "时间格式化",
    content: buildDslTemplate(
      "text",
      "  text: 'createTime',\n  format: 'datetime',\n  valueFormat: 'YYYY-MM-DD HH:mm:ss',",
    ),
  },
  {
    id: "text-action-edit",
    label: "表格操作-编辑",
    content: buildDslTemplate("text", "  value: '编辑',\n  type: 'primary',\n  click: 'onEdit',"),
  },
  {
    id: "text-action-delete",
    label: "表格操作-删除",
    content: buildDslTemplate(
      "text",
      "  value: '删除',\n  type: 'danger',\n  click: 'onDelete',\n  confirm: '确认删除该条数据吗？',",
    ),
  },
  {
    id: "text-multi-prefix",
    label: "多文本组合前缀",
    content: buildDslTemplate("text", "  value: '￥',\n  color: '#f56c6c',"),
  },
  {
    id: "text-multi-price",
    label: "多文本组合金额",
    content: buildDslTemplate("text", "  text: 'price',\n  color: '#f56c6c',\n  weight: 'bold',"),
  },
  {
    id: "text-full-dsl",
    label: "完整配置模板",
    content: buildDslTemplate(
      "text",
      "  text: '',\n  value: '',\n  empty: '--',\n\n  color: '',\n  size: 14,\n  weight: 'normal',\n  lineHeight: 1.5,\n\n  wrap: false,\n  ellipsis: false,\n  maxWidth: 200,\n\n  tooltip: false,\n\n  dict: null,\n  tag: false,\n\n  link: false,\n  type: 'default',\n  click: '',\n\n  copy: false,\n\n  format: '',\n  precision: 2,\n  unit: '',",
    ),
  },
];

const customDetailPresetMap: AnyRecord = {
  HorizontalLayout: [
    {
      id: buildPresetId("HorizontalLayout", "basic"),
      label: "基础横排",
      content: buildDslTemplate(
        "horizontalLayout",
        "  justify: 'flex-start',\n  align: 'center',\n  gap: 10,\n  showBorder: false,",
      ),
    },
    {
      id: buildPresetId("HorizontalLayout", "space-between"),
      label: "两端对齐",
      content: buildDslTemplate(
        "horizontalLayout",
        "  justify: 'space-between',\n  align: 'center',\n  gap: 12,\n  showBorder: false,",
      ),
    },
    {
      id: buildPresetId("HorizontalLayout", "toolbar"),
      label: "工具栏",
      content: buildDslTemplate(
        "horizontalLayout",
        "  justify: 'flex-start',\n  align: 'center',\n  gap: 8,\n  wrap: false,\n  showBorder: false,",
      ),
    },
  ],
  VerticalLayout: [
    {
      id: buildPresetId("VerticalLayout", "basic"),
      label: "基础纵排",
      content: buildDslTemplate(
        "verticalLayout",
        "  justify: 'flex-start',\n  align: 'stretch',\n  gap: 10,\n  showBorder: false,",
      ),
    },
    {
      id: buildPresetId("VerticalLayout", "center"),
      label: "居中纵排",
      content: buildDslTemplate(
        "verticalLayout",
        "  justify: 'center',\n  align: 'center',\n  gap: 12,\n  showBorder: false,",
      ),
    },
    {
      id: buildPresetId("VerticalLayout", "form-stack"),
      label: "表单纵排",
      content: buildDslTemplate(
        "verticalLayout",
        "  justify: 'flex-start',\n  align: 'stretch',\n  gap: 16,\n  showBorder: true,",
      ),
    },
  ],
  FormLayout: [
    {
      id: buildPresetId("FormLayout", "basic"),
      label: "基础表单",
      content: buildDslTemplate(
        "formLayout",
        "  columns: 2,\n  columnGap: 12,\n  rowGap: 12,\n  showBorder: false,",
      ),
    },
    {
      id: buildPresetId("FormLayout", "border"),
      label: "带边框表单",
      content: buildDslTemplate(
        "formLayout",
        "  columns: 2,\n  columnGap: 16,\n  rowGap: 16,\n  showBorder: true,",
      ),
    },
  ],
};

/**
 * 格式化 DSL 值
 * @param {*} value - 值
 * @param {number} indentSize - 缩进空格数
 * @returns {string}
 */
function formatDslValue(value: any, indentSize = 2): string {
  if (value === undefined) return "undefined";
  const json = JSON.stringify(value, null, 2);
  if (!json) return "null";
  const lines = json.split("\n");
  if (lines.length === 1) return json;
  const pad = " ".repeat(indentSize);
  return [lines[0], ...lines.slice(1).map((line) => `${pad}${line}`)].join("\n");
}

/**
 * 获取 Manifest 默认值
 * @param {string} type - 组件类型
 * @param {string} propName - 属性名
 * @returns {*}
 */
function getManifestDefaultValue(type: string, propName: string): any {
  if (!type || !propName) return undefined;
  const manifest = getManifest(type);
  if (!manifest?.props?.length) return undefined;
  return manifest.props.find((prop) => prop.name === propName)?.defaultValue;
}

function clonePanelValue<T>(value: T): T {
  if (value === undefined || value === null) return value;
  try {
    return JSON.parse(JSON.stringify(value)) as T;
  } catch {
    return value;
  }
}

/**
 * 详细配置清空后恢复组件默认配置。
 * 只回滚详细配置会写入的 props/style/events/bindings/conditions，保留画布位置、尺寸和子节点。
 */
function buildDefaultDetailResetPatch(node: AnyRecord): AnyRecord {
  const type = elementTypeName(node?.type);
  const manifest = type ? getManifest(type) : null;
  const defaultProps: AnyRecord = {};
  Object.keys(node?.props || {}).forEach((key) => {
    defaultProps[key] = undefined;
  });
  (manifest?.props || []).forEach((prop: AnyRecord) => {
    if (!prop?.name) return;
    if (prop.defaultValue !== undefined) {
      defaultProps[prop.name] = clonePanelValue(prop.defaultValue);
    }
  });
  if (type === "Menu") {
    Object.assign(defaultProps, getMenuDefaultProps());
  }
  return {
    detailConfig: "",
    props: defaultProps,
    style: { ...(manifest?.defaultStyle || {}) },
    events: {},
    bindings: {},
    conditions: {},
    hidden: false,
  };
}

function resetNodeDetailToDefault(node: AnyRecord): void {
  if (!node?.id) return;
  const patch = buildDefaultDetailResetPatch(node);
  const ok = editorStore.updateNode(node.id, patch);
  if (!ok) {
    applyLocalNodePatch(node, patch);
  }
  emitDetailConfig(node.id, "");
  editorStore.saveCurrentPageDraft?.();
}

/**
 * 提取 DSL 模板主体
 * @param {string} content - 模板内容
 * @returns {string}
 */
function extractDslBody(content: string): string {
  const text = String(content || "");
  const start = text.indexOf("{");
  const end = text.lastIndexOf("}");
  if (start === -1 || end === -1 || end <= start) return text.trim();
  return text.slice(start + 1, end).trim();
}

const dslEventKeyMap: AnyRecord = {
  onClick: "click",
  onChange: "change",
  onRowClick: "rowClick",
  onCommand: "command",
  onClose: "close",
  onTabClick: "tabClick",
  onNodeClick: "nodeClick",
};

/**
 * 拆分 DSL 顶层字段
 * @param {string} body - 模板主体
 * @returns {string[]}
 */
function splitTopLevelEntries(body: string): AnyArray {
  const entries: AnyArray = [];
  let buffer = "";
  let depth = 0;
  let inString = false;
  let stringChar = "";

  for (let i = 0; i < body.length; i += 1) {
    const ch = body[i];
    const prev = body[i - 1];
    if (inString) {
      buffer += ch;
      if (ch === stringChar && prev !== "\\") {
        inString = false;
      }
      continue;
    }
    if (ch === "'" || ch === '"' || ch === "`") {
      inString = true;
      stringChar = ch;
      buffer += ch;
      continue;
    }
    if (ch === "{" || ch === "[" || ch === "(") depth += 1;
    if (ch === "}" || ch === "]" || ch === ")") depth -= 1;
    if (ch === "," && depth === 0) {
      if (buffer.trim()) entries.push(buffer.trim());
      buffer = "";
      continue;
    }
    buffer += ch;
  }

  if (buffer.trim()) entries.push(buffer.trim());
  return entries;
}

/**
 * 生成标准 DSL 模板
 * @param {string} type - 组件类型
 * @param {{ id: string, label: string, content: string }} preset - 预设项
 * @returns {{ id: string, label: string, content: string }}
 */
function normalizeDslPreset(type: string, preset: AnyRecord): AnyRecord {
  const rawContent = preset?.content || "";
  if (rawContent.includes("props:") || rawContent.includes("events:")) {
    return preset;
  }
  const methodName = getDslMethodName(type);
  const body = extractDslBody(rawContent);
  const entries = splitTopLevelEntries(body);
  const dslId = `${methodName}Id`;
  const label = preset?.label || "";
  const props: AnyArray = [];
  const state: AnyArray = [];
  const events: AnyArray = [];
  let styleBody = "";
  let className = "";
  let idValue = "";
  let labelValue = "";
  let typeValue = "";

  const normalizeValue = (value: string) => value.replace(DSL_TRAILING_COMMA_RE, "").trim();

  entries.forEach((entry: string) => {
    const colonIndex = entry.indexOf(":");
    if (colonIndex === -1) return;
    const key = entry.slice(0, colonIndex).trim();
    const value = normalizeValue(entry.slice(colonIndex + 1));
    if (key === "id") {
      idValue = value;
      return;
    }
    if (key === "label") {
      labelValue = value;
      return;
    }
    if (key === "type") {
      typeValue = value;
      return;
    }
    if (key === "className") {
      className = value;
      return;
    }
    if (key === "style") {
      if (value.startsWith("{") && value.endsWith("}")) {
        styleBody = value.slice(1, -1).trim();
      }
      return;
    }
    if (dslEventKeyMap[key]) {
      events.push(`  ${dslEventKeyMap[key]}: ${value}`);
      return;
    }
    if (["visible", "disabled", "loading"].includes(key)) {
      state.push(`  ${key}: ${value}`);
      return;
    }
    props.push(`  ${key}: ${value}`);
  });

  const toBlock = (items: AnyArray) =>
    items.length ? items.map((item) => item.replace(DSL_TRAILING_COMMA_RE, "")).join(",\n") : "";

  const styleLines = styleBody
    ? styleBody
        .split("\n")
        .map((line) => line.trim())
        .filter(Boolean)
        .map((line) => `  ${line.replace(DSL_TRAILING_COMMA_RE, "")}`)
        .join(",\n")
    : "";

  const blocks = [
    `  id: ${idValue || `"${dslId}"`},`,
    `  label: ${labelValue || (label ? `"${label}"` : '""')},`,
    `  type: ${typeValue || `"${type}"`},`,
    props.length ? `  props: {\n${toBlock(props)}\n  },` : "",
    state.length ? `  state: {\n${toBlock(state)}\n  },` : "",
    styleLines ? `  style: {\n${styleLines}\n  },` : "",
    className ? `  className: ${className},` : "",
    events.length ? `  events: {\n${toBlock(events)}\n  },` : "",
  ]
    .filter(Boolean)
    .join("\n");

  return {
    ...preset,
    tags: preset?.tags || [type, "detail"],
    keywords: preset?.keywords || [preset?.label, type].filter(Boolean).join(" "),
    content: `this.${methodName}({\n${blocks}\n});`,
  };
}

const elementPlusPresetGroups: PresetGroupLike[] = [
  {
    name: "custom",
    types: [
      "Radio",
      "Checkbox",
      "Cascader",
      "Tabs",
      "Transfer",
      "Tag",
      "InputNumber",
      "Timeline",
      "ImageCarousel",
      "CarouselComponent",
      "WebContainer",
      "Steps",
      "Card",
      "Pagination",
      "Collapse",
      "BigDataTable",
      "BusinessCard",
      "Barcode",
      "Slider",
      "Calendar",
      "Signature",
    ],
    detail: (methodName, type) => {
      switch (type) {
        case "Radio":
          return [
            {
              id: buildPresetId(type, "full"),
              label: "通用模板",
              content: buildDslTemplate(
                methodName,
                "  text: 'field',\n  disabled: false,\n  size: 'small',\n  button: false,\n  border: false,\n  options: [],\n  change: 'onChange',",
              ),
            },
            {
              id: buildPresetId(type, "basic"),
              label: "普通单选",
              content: buildDslTemplate(
                methodName,
                "  text:'gender',\n  options:[{label:'男',value:'M'},{label:'女',value:'F'}],\n  change:'onGenderChange',",
              ),
            },
            {
              id: buildPresetId(type, "button"),
              label: "按钮样式",
              content: buildDslTemplate(
                methodName,
                "  text:'level',\n  button:true,\n  options:[{label:'A',value:'A'},{label:'B',value:'B'},{label:'C',value:'C'}],",
              ),
            },
            {
              id: buildPresetId(type, "border"),
              label: "带边框 + 禁用项",
              content: buildDslTemplate(
                methodName,
                "  text:'payType',\n  border:true,\n  options:[{label:'微信',value:'wx'},{label:'\u652F\u4ED8\u5B9D',value:'ali',disabled:true}],",
              ),
            },
          ];
        case "Checkbox":
          return [
            {
              id: buildPresetId(type, "full"),
              label: "通用模板",
              content: buildDslTemplate(
                methodName,
                "  text: 'field',\n  disabled: false,\n  size: 'small',\n  button: false,\n  min: 0,\n  max: null,\n  options: [],\n  change: 'onChange',\n\n  checkAll: false,\n  indeterminate: false,\n  checkAllText: '\u5168\u9009',\n  checkAllChange: 'onCheckAllChange',",
              ),
            },
            {
              id: buildPresetId(type, "basic"),
              label: "普通多选",
              content: buildDslTemplate(
                methodName,
                "  text:'hobbies',\n  options:[{label:'音乐',value:'music'},{label:'运动',value:'sports'}],",
              ),
            },
            {
              id: buildPresetId(type, "limit"),
              label: "min/max 限制",
              content: buildDslTemplate(
                methodName,
                "  text:'features',\n  min:1,\n  max:2,\n  options:'featureOptions',",
              ),
            },
            {
              id: buildPresetId(type, "check-all"),
              label: "全选/半选",
              content: buildDslTemplate(
                methodName,
                "  text:'perms',\n  checkAll:true,\n  indeterminate:true,\n  options:'permOptions',\n  checkAllChange:'onCheckAllChange',\n  change:'onPermsChange',",
              ),
            },
            {
              id: buildPresetId(type, "button"),
              label: "按钮样式",
              content: buildDslTemplate(
                methodName,
                "  text:'days',\n  button:true,\n  options:[{label:'周一',value:1},{label:'周二',value:2}],",
              ),
            },
          ];
        case "Cascader":
          return [
            {
              id: buildPresetId(type, "full"),
              label: "通用模板",
              content: buildDslTemplate(
                methodName,
                "  text: 'field',\n  placeholder: '请选择',\n  disabled: false,\n  clearable: true,\n  filterable: false,\n  collapseTags: false,\n\n  options: [],\n  props: { value:'value', label:'label', children:'children', checkStrictly:false, multiple:false, emitPath:true },\n\n  showAllLevels: true,\n  separator: '/',\n  change: 'onChange',\n\n  lazy: false,\n  lazyLoad: '',",
              ),
            },
            {
              id: buildPresetId(type, "basic"),
              label: "基础",
              content: buildDslTemplate(
                methodName,
                "  text:'region',\n  placeholder:'请选择地区',\n  options:'regionOptions',\n  clearable:true,",
              ),
            },
            {
              id: buildPresetId(type, "strict"),
              label: "任意层级可选",
              content: buildDslTemplate(
                methodName,
                "  text:'dept',\n  options:'deptOptions',\n  props:{ checkStrictly:true },",
              ),
            },
            {
              id: buildPresetId(type, "multiple"),
              label: "多选",
              content: buildDslTemplate(
                methodName,
                "  text:'regions',\n  options:'regionOptions',\n  props:{ multiple:true },\n  collapseTags:true,",
              ),
            },
            {
              id: buildPresetId(type, "emit-path"),
              label: "只取最后一级值",
              content: buildDslTemplate(
                methodName,
                "  text:'category',\n  options:'catOptions',\n  props:{ emitPath:false },",
              ),
            },
            {
              id: buildPresetId(type, "lazy"),
              label: "懒加载",
              content: buildDslTemplate(
                methodName,
                "  text:'area',\n  props:{ lazy:true, lazyLoad:'lazyLoadCascader', value:'value', label:'label', children:'children' },",
              ),
            },
            {
              id: buildPresetId(type, "filter"),
              label: "可搜索",
              content: buildDslTemplate(
                methodName,
                "  text:'region',\n  options:'regionOptions',\n  filterable:true,",
              ),
            },
          ];
        case "Tabs":
          return [
            {
              id: buildPresetId(type, "full"),
              label: "通用模板",
              content: buildDslTemplate(
                methodName,
                "  modelValue: 'tab1',\n  activeName: 'tab1',\n  type: '',\n  tabPosition: 'top',\n  stretch: false,\n  closable: false,\n\n  tabs: [\n    { name: 'tab1', label: '标签1' },\n    { name: 'tab2', label: '标签2' },\n  ],\n\n  tabClick: 'onTabClick',\n  tabRemove: 'onTabRemove',\n  tabAdd: 'onTabAdd',\n  edit: 'onTabEdit',",
              ),
            },
            {
              id: buildPresetId(type, "basic"),
              label: "Tabs",
              content: buildDslTemplate(
                methodName,
                "  modelValue: 'tab1',\n  activeName: 'tab1',\n  tabs: [\n    { name: 'tab1', label: '标签1' },\n    { name: 'tab2', label: '标签2' },\n  ],",
              ),
            },
            {
              id: buildPresetId(type, "card"),
              label: "Card / Border Card",
              content: buildDslTemplate(
                methodName,
                "  modelValue: 'tab1',\n  activeName: 'tab1',\n  type: 'card',\n  tabs: [\n    { name: 'tab1', label: '标签1' },\n    { name: 'tab2', label: '标签2' },\n  ],",
              ),
            },
            {
              id: buildPresetId(type, "border-card"),
              label: "Border Card",
              content: buildDslTemplate(
                methodName,
                "  text:'activeTab',\n  type:'border-card',\n  tabs:'tabList',",
              ),
            },
            {
              id: buildPresetId(type, "editable"),
              label: "可新增关闭",
              content: buildDslTemplate(
                methodName,
                "  text:'activeTab',\n  editable:true,\n  tabs:'tabList',\n  edit:'onTabEdit',",
              ),
            },
            {
              id: buildPresetId(type, "lazy"),
              label: "懒加载",
              content: buildDslTemplate(
                methodName,
                "  text:'activeTab',\n  tabs:[{name:'a',label:'A',lazy:true,content:'renderA'}],",
              ),
            },
            {
              id: buildPresetId(type, "left"),
              label: "左侧竖向",
              content: buildDslTemplate(
                methodName,
                "  text:'activeTab',\n  tabPosition:'left',\n  tabs:'tabList',",
              ),
            },
          ];
        case "Transfer":
          return [
            {
              id: buildPresetId(type, "full"),
              label: "通用模板",
              content: buildDslTemplate(
                methodName,
                "  text: 'selectedKeys',\n  data: [],\n  props: { key:'key', label:'label', disabled:'disabled' },\n  titles: ['左侧','右侧'],\n  buttonTexts: ['到左侧','到右侧'],\n  filterable: false,\n  filterPlaceholder: '请输入搜索关键词',\n  targetOrder: 'original',\n  leftDefaultChecked: [],\n  rightDefaultChecked: [],\n  change: 'onChange',\n  leftCheckChange: 'onLeftCheckChange',\n  rightCheckChange: 'onRightCheckChange',",
              ),
            },
            {
              id: buildPresetId(type, "basic"),
              label: "基础",
              content: buildDslTemplate(
                methodName,
                "  text:'selectedKeys',\n  data:[{key:1,label:'选项1'},{key:2,label:'选项2'}],\n  filterable:true,\n  change:'onTransferChange',",
              ),
            },
            {
              id: buildPresetId(type, "mapping"),
              label: "字段映射",
              content: buildDslTemplate(
                methodName,
                "  text:'selectedIds',\n  data:'userData',\n  props:{ key:'id', label:'name' },",
              ),
            },
            {
              id: buildPresetId(type, "checked"),
              label: "默认勾选",
              content: buildDslTemplate(
                methodName,
                "  text:'selectedKeys',\n  data:'transferData',\n  leftDefaultChecked:[1,2],\n  rightDefaultChecked:[3],",
              ),
            },
            {
              id: buildPresetId(type, "order"),
              label: "目标排序",
              content: buildDslTemplate(
                methodName,
                "  text:'selectedKeys',\n  data:'transferData',\n  targetOrder:'push',",
              ),
            },
          ];
        case "Tag":
          return [
            {
              id: buildPresetId(type, "full"),
              label: "通用模板",
              content: buildDslTemplate(
                methodName,
                "  text: '',\n  value: '',\n  type: 'info',\n  effect: 'light',\n  size: 'small',\n  closable: false,\n  disableTransitions: false,\n  color: '',\n  close: 'onClose',\n  click: 'onClick',\n\n  dict: null,",
              ),
            },
            {
              id: buildPresetId(type, "basic"),
              label: "基础",
              content: buildDslTemplate(
                methodName,
                "  value:'已完成',\n  type:'success',\n  effect:'light',",
              ),
            },
            {
              id: buildPresetId(type, "closable"),
              label: "可关闭",
              content: buildDslTemplate(
                methodName,
                "  text:'tagName',\n  closable:true,\n  close:'onTagClose',",
              ),
            },
            {
              id: buildPresetId(type, "dict"),
              label: "状态映射",
              content: buildDslTemplate(
                methodName,
                "  text:'status',\n  dict:{ 0:{label:'\u7981\u7528',type:'info'}, 1:{label:'\u542F\u7528',type:'success'}, 2:{label:'\u8B66\u544A',type:'warning'} },",
              ),
            },
            {
              id: buildPresetId(type, "color"),
              label: "自定义颜色",
              content: buildDslTemplate(
                methodName,
                "  value:'\u793A\u4F8B',\n  color:'#409EFF',\n  effect:'dark',",
              ),
            },
          ];
        case "InputNumber":
          return [
            {
              id: buildPresetId(type, "full"),
              label: "通用模板",
              content: buildDslTemplate(
                "counter",
                "  text: 'field',\n  disabled: false,\n  min: 0,\n  max: 999,\n  step: 1,\n  precision: 0,\n  size: 'small',\n  controls: true,\n  controlsPosition: 'right',\n\n  change: 'onChange',\n  blur: 'onBlur',\n  focus: 'onFocus',",
              ),
            },
            {
              id: buildPresetId(type, "basic"),
              label: "整数",
              content: buildDslTemplate(
                "counter",
                "  text:'count',\n  min:0,\n  max:999,\n  step:1,\n  precision:0,\n  change:'onCountChange',",
              ),
            },
            {
              id: buildPresetId(type, "precision"),
              label: "小数精度",
              content: buildDslTemplate(
                "counter",
                "  text:'price',\n  min:0,\n  max:99999,\n  step:0.1,\n  precision:2,",
              ),
            },
            {
              id: buildPresetId(type, "disabled"),
              label: "禁用/动态禁用",
              content: buildDslTemplate("counter", "  text:'amount',\n  disabled:true,"),
            },
            {
              id: buildPresetId(type, "exceed"),
              label: "超范围提绀",
              content: buildDslTemplate(
                "counter",
                "  text:'qty',\n  min:1,\n  max:10,\n  exceedTip:'数量范围 1~10',",
              ),
            },
          ];
        case "Timeline":
          return [
            {
              id: buildPresetId(type, "full"),
              label: "通用模板",
              content: buildDslTemplate(
                methodName,
                "  reverse: false,\n  items: [\n    // {timestamp, hideTimestamp, type, color, icon, size, placement, content: string|dsl|renderFn}\n  ],",
              ),
            },
            {
              id: buildPresetId(type, "basic"),
              label: "基础",
              content: buildDslTemplate(
                methodName,
                "  items:[\n    { timestamp:'2026-02-01 10:00', type:'primary', content:'创建' },\n    { timestamp:'2026-02-02 12:00', type:'success', content:'完成' },\n  ],",
              ),
            },
            {
              id: buildPresetId(type, "reverse"),
              label: "反向",
              content: buildDslTemplate(methodName, "  reverse:true,\n  items:'timeItems',"),
            },
            {
              id: buildPresetId(type, "icon"),
              label: "图标 + 大节点",
              content: buildDslTemplate(
                methodName,
                "  items:[{ timestamp:'2026-02-09', icon:'el-icon-check', size:'large', type:'success', content:'已审核' }],",
              ),
            },
            {
              id: buildPresetId(type, "card"),
              label: "为卡鐗",
              content: buildDslTemplate(
                methodName,
                "  items:[{\n    timestamp:'2026-02-09 09:00',\n    type:'warning',\n    content: this.card({ title:'异常', body:[ this.text({value:'接口超时'}) ] })\n  }],",
              ),
            },
          ];
        case "ImageCarousel":
          return [
            {
              id: buildPresetId(type, "full"),
              label: "通用模板",
              content: buildDslTemplate(
                methodName,
                "  height: 320,\n  initialIndex: 0,\n  autoplay: true,\n  interval: 3000,\n  loop: true,\n  direction: 'horizontal',\n  arrow: 'hover',\n  indicatorPosition: 'outside',\n  trigger: 'hover',\n\n  images: [],\n  fit: 'cover',\n  preview: false,\n  click: 'onImageClick',",
              ),
            },
            {
              id: buildPresetId(type, "basic"),
              label: "自动轮播",
              content: buildDslTemplate(
                methodName,
                "  images:[{src:'https://x/1.jpg'},{src:'https://x/2.jpg'}],\n  autoplay:true,\n  preview:true,",
              ),
            },
            {
              id: buildPresetId(type, "manual"),
              label: "手动 + 常显",
              content: buildDslTemplate(
                methodName,
                "  autoplay:false,\n  arrow:'always',\n  images:'bannerList',",
              ),
            },
            {
              id: buildPresetId(type, "vertical"),
              label: "垂直轮播",
              content: buildDslTemplate(
                methodName,
                "  direction:'vertical',\n  height:260,\n  images:'bannerList',",
              ),
            },
            {
              id: buildPresetId(type, "click"),
              label: "点击跳转/埋点",
              content: buildDslTemplate(
                methodName,
                "  images:'bannerList',\n  click:'onBannerClick',",
              ),
            },
          ];
        case "CarouselComponent":
          return [
            {
              id: buildPresetId(type, "full"),
              label: "通用模板",
              content: buildDslTemplate(
                "carousel",
                "  height: 200,\n  initialIndex: 0,\n  autoplay: true,\n  interval: 4000,\n  loop: true,\n  direction: 'horizontal',\n  arrow: 'hover',\n  indicatorPosition: 'outside',\n  trigger: 'hover',\n\n  items: [\n    // {content: string|dsl|renderFn}\n  ],\n  change: 'onCarouselChange',",
              ),
            },
            {
              id: buildPresetId(type, "text"),
              label: "文字轮播",
              content: buildDslTemplate(
                "carousel",
                "  items:[\n    { content: this.text({ value:'公告1' }) },\n    { content: this.text({ value:'公告2' }) },\n  ],",
              ),
            },
            {
              id: buildPresetId(type, "card"),
              label: "卡片轮播",
              content: buildDslTemplate(
                "carousel",
                "  items:[\n    { content: this.card({ title:'A', body:[ this.text({value:'?A'}) ] }) },\n    { content: this.card({ title:'B', body:[ this.text({value:'?B'}) ] }) },\n  ],",
              ),
            },
            {
              id: buildPresetId(type, "simple"),
              label: "无指示器 + 垂直",
              content: buildDslTemplate(
                "carousel",
                "  indicatorPosition:'none',\n  direction:'vertical',\n  height:240,\n  items:'carouselItems',",
              ),
            },
          ];
        case "WebContainer":
          return [
            {
              id: buildPresetId(type, "full"),
              label: "通用模板",
              content: buildDslTemplate(
                methodName,
                "  src: '',\n  width: '100%',\n  height: 600,\n  border: false,\n  loading: true,\n\n  sandbox: '',\n  allow: '',\n  referrerPolicy: '',\n\n  message: 'onIframeMessage',\n  load: 'onIframeLoad',\n  error: 'onIframeError',",
              ),
            },
            {
              id: buildPresetId(type, "basic"),
              label: "基础",
              content: buildDslTemplate(
                methodName,
                "  src:'https://example.com',\n  height:600,\n  border:false,",
              ),
            },
            {
              id: buildPresetId(type, "sandbox"),
              label: "沙箱策略",
              content: buildDslTemplate(
                methodName,
                "  src:'https://example.com',\n  sandbox:'allow-scripts allow-same-origin',\n  height:700,",
              ),
            },
            {
              id: buildPresetId(type, "params"),
              label: "动态参数",
              content: buildDslTemplate(
                methodName,
                "  src:'https://example.com?id={{form.id}}&token={{token}}',\n  height:650,",
              ),
            },
            {
              id: buildPresetId(type, "message"),
              label: "postMessage 通信",
              content: buildDslTemplate(
                methodName,
                "  src:'https://example.com',\n  message:'onIframeMessage',",
              ),
            },
          ];
        case "Steps":
          return [
            {
              id: buildPresetId(type, "full"),
              label: "通用模板",
              content: buildDslTemplate(
                methodName,
                "  active: 0,\n  direction: 'horizontal',\n  alignCenter: true,\n  simple: false,\n  processStatus: 'process',\n  finishStatus: 'success',\n  space: '',\n  steps: [\n    // {title, description, icon, status}\n  ],",
              ),
            },
            {
              id: buildPresetId(type, "desc"),
              label: "横向带描杩",
              content: buildDslTemplate(
                methodName,
                "  active: 1,\n  steps:[\n    { title:'提交', description:'填写信息' },\n    { title:'审核', description:'等待审核' },\n    { title:'完成', description:'结果输出' },\n  ],",
              ),
            },
            {
              id: buildPresetId(type, "vertical"),
              label: "竖向",
              content: buildDslTemplate(
                methodName,
                "  active:2,\n  direction:'vertical',\n  steps:'stepList',",
              ),
            },
            {
              id: buildPresetId(type, "simple"),
              label: "简娲Simple",
              content: buildDslTemplate(
                methodName,
                "  active:1,\n  simple:true,\n  steps:[{title:'Step1'},{title:'Step2'},{title:'Step3'}],",
              ),
            },
            {
              id: buildPresetId(type, "custom"),
              label: "图标",
              content: buildDslTemplate(
                methodName,
                "  active:1,\n  steps:[\n    { title:'提交', icon:'el-icon-edit' },\n    { title:'审核', status:'error', description:'驳回' },\n    { title:'完成' },\n  ],",
              ),
            },
          ];
        case "Card":
          return [
            {
              id: buildPresetId(type, "full"),
              label: "通用模板",
              content: buildDslTemplate(
                methodName,
                "  title: '',\n  shadow: 'hover',\n  bodyStyle: {},\n\n  extra: null,\n  header: null,\n  body: [],\n\n  loading: false,",
              ),
            },
            {
              id: buildPresetId(type, "basic"),
              label: "卡片",
              content: buildDslTemplate(
                methodName,
                "  title:'?',\n  body:[ this.text({value:'?'}) ],",
              ),
            },
            {
              id: buildPresetId(type, "extra"),
              label: "带右侧操作区",
              content: buildDslTemplate(
                methodName,
                "  title:'业务概览',\n  extra: this.button({ text:'刷新', type:'text', click:'refreshCard' }),\n  body:[ this.text({value:'...'}) ],",
              ),
            },
            {
              id: buildPresetId(type, "plain"),
              label: "Header",
              content: buildDslTemplate(
                methodName,
                "  shadow:'never',\n  body:[ this.text({value:'\u793A\u4F8B'}) ],",
              ),
            },
            {
              id: buildPresetId(type, "table"),
              label: "卡片内嵌表格/表单",
              content: buildDslTemplate(
                methodName,
                "  title:'列表',\n  body:[\n    this.table({ data:'tableData', columns:'columns' }),\n    this.pagination({ page:'page.current', size:'page.size', total:'page.total' }),\n  ],",
              ),
            },
          ];
        case "Pagination":
          return [
            {
              id: buildPresetId(type, "full"),
              label: "通用模板",
              content: buildDslTemplate(
                methodName,
                "  page: 'page.current',\n  size: 'page.size',\n  total: 'page.total',\n  small: false,\n  background: true,\n\n  layout: 'total, sizes, prev, pager, next, jumper',\n  pageSizes: [10,20,50,100],\n\n  pagerCount: 7,\n  disabled: false,\n  hideOnSinglePage: false,\n\n  pageChange: 'onPageChange',\n  sizeChange: 'onSizeChange',\n  prevClick: 'onPrevClick',\n  nextClick: 'onNextClick',",
              ),
            },
            {
              id: buildPresetId(type, "basic"),
              label: "标准",
              content: buildDslTemplate(
                methodName,
                "  page:'page.current',\n  size:'page.size',\n  total:'page.total',\n  pageChange:'onPageChange',\n  sizeChange:'onSizeChange',",
              ),
            },
            {
              id: buildPresetId(type, "simple"),
              label: "简娲",
              content: buildDslTemplate(
                methodName,
                "  small:true,\n  layout:'prev, pager, next',\n  page:'page.current',\n  total:'page.total',",
              ),
            },
            {
              id: buildPresetId(type, "jumper"),
              label: "仅总数 + 跳页",
              content: buildDslTemplate(
                methodName,
                "  layout:'total, jumper',\n  page:'page.current',\n  total:'page.total',\n  pageChange:'onPageChange',",
              ),
            },
            {
              id: buildPresetId(type, "hide"),
              label: "隐藏单页",
              content: buildDslTemplate(
                methodName,
                "  hideOnSinglePage:true,\n  page:'page.current',\n  total:'page.total',",
              ),
            },
          ];
        case "Collapse":
          return [
            {
              id: buildPresetId(type, "full"),
              label: "通用模板",
              content: buildDslTemplate(
                methodName,
                "  modelValue: ['1'],\n  accordion: false,\n  items: [\n    { name: '1', title: '面板1' },\n    { name: '2', title: '面板2' },\n  ],\n  change: 'onCollapseChange',",
              ),
            },
            {
              id: buildPresetId(type, "multi"),
              label: "多开",
              content: buildDslTemplate(
                methodName,
                "  modelValue: ['1', '2'],\n  accordion: false,\n  items: [\n    { name: '1', title: '面板1' },\n    { name: '2', title: '面板2' },\n    { name: '3', title: '面板3' },\n  ],",
              ),
            },
            {
              id: buildPresetId(type, "accordion"),
              label: "手风琴",
              content: buildDslTemplate(
                methodName,
                "  modelValue: '1',\n  accordion: true,\n  items: [\n    { name: '1', title: '面板1' },\n    { name: '2', title: '面板2' },\n  ],",
              ),
            },
            {
              id: buildPresetId(type, "disabled"),
              label: "默认展开 + 禁用项",
              content: buildDslTemplate(
                methodName,
                "  modelValue: ['1'],\n  accordion: false,\n  items: [\n    { name: '1', title: '面板1' },\n    { name: '2', title: '面板2', disabled: true },\n  ],",
              ),
            },
          ];
        case "BigDataTable":
          return [
            {
              id: buildPresetId(type, "full"),
              label: "通用模板",
              content: buildDslTemplate(
                "bigTable",
                "  data: 'bigData',\n  rowKey: 'id',\n  height: 600,\n  rowHeight: 44,\n  overscan: 10,\n\n  columns: [],\n\n  selectable: false,\n  selectionChange: 'onSelectionChange',\n\n  remote: false,\n  fetch: '',\n\n  rowClick: 'onRowClick',\n  sortChange: 'onSortChange',",
              ),
            },
            {
              id: buildPresetId(type, "local"),
              label: "本地虚拟",
              content: buildDslTemplate(
                "bigTable",
                "  data:'bigData',\n  height:600,\n  rowKey:'id',\n  columns:[{prop:'id',label:'ID',width:100},{prop:'name',label:'名称',minWidth:180}],",
              ),
            },
            {
              id: buildPresetId(type, "remote"),
              label: "远程加载",
              content: buildDslTemplate(
                "bigTable",
                "  remote:true,\n  fetch:'fetchBigTable',\n  height:620,\n  columns:'bigColumns',",
              ),
            },
            {
              id: buildPresetId(type, "select"),
              label: "选择 + 批量操作",
              content: buildDslTemplate(
                "bigTable",
                "  data:'bigData',\n  selectable:true,\n  selectionChange:'onBigSelectionChange',\n  columns:'bigColumns',",
              ),
            },
            {
              id: buildPresetId(type, "expand"),
              label: "行展开/详情",
              content: buildDslTemplate(
                "bigTable",
                "  data:'bigData',\n  expand:'renderExpand(row)',\n  columns:'bigColumns',",
              ),
            },
          ];
        case "BusinessCard":
          return [
            {
              id: buildPresetId(type, "full"),
              label: "通用模板",
              content: buildDslTemplate(
                "bizCard",
                "  title: '',\n  subTitle: '',\n  value: null,\n  unit: '',\n  trend: null,\n  status: null,\n  icon: '',\n  color: '',\n\n  metrics: null,\n  body: null,\n  footer: null,\n  extra: null,\n\n  click: 'onCardClick',",
              ),
            },
            {
              id: buildPresetId(type, "metric"),
              label: "指标鍗",
              content: buildDslTemplate(
                "bizCard",
                "  title:'今日订单',\n  value:1280,\n  unit:'单',\n  trend:{ type:'up', value:'12%' },\n  extra:[ this.button({ text:'查看', type:'text', click:'goOrders' }) ],\n  footer:[ this.text({ value:'更新时间：2026-02-09' }) ],",
              ),
            },
            {
              id: buildPresetId(type, "metrics"),
              label: "多指标",
              content: buildDslTemplate(
                "bizCard",
                "  title:'转化漏斗',\n  metrics:[{label:'访问',value:10234},{label:'下单',value:823},{label:'支付',value:612}],",
              ),
            },
            {
              id: buildPresetId(type, "status"),
              label: "状态告警卡",
              content: buildDslTemplate(
                "bizCard",
                "  title:'服务健康',\n  status:{ type:'warning', text:'部分异常' },\n  body:[ this.text({ value:'错误率 1.2%' }) ],\n  extra:[ this.button({ text:'查看告警', type:'text', click:'goAlerts' }) ],",
              ),
            },
            {
              id: buildPresetId(type, "click"),
              label: "可点击跳杞",
              content: buildDslTemplate(
                "bizCard",
                "  title:'客户数',\n  value:320,\n  unit:'人',\n  click:'goCustomers',",
              ),
            },
          ];
        case "Barcode":
          return [
            {
              id: buildPresetId(type, "full"),
              label: "通用模板",
              content: buildDslTemplate(
                methodName,
                "  text: '',\n  value: '',\n  format: 'CODE128',\n  width: 2,\n  height: 60,\n  displayValue: true,\n  fontSize: 12,\n  margin: 8,\n  background: '#fff',\n  lineColor: '#000',",
              ),
            },
            {
              id: buildPresetId(type, "bind"),
              label: "绑定字段",
              content: buildDslTemplate(
                methodName,
                "  text:'orderNo',\n  format:'CODE128',\n  height:60,\n  displayValue:true,",
              ),
            },
            {
              id: buildPresetId(type, "static"),
              label: "静态值 + 不显示文字",
              content: buildDslTemplate(
                methodName,
                "  value:'ABC-20260209-001',\n  displayValue:false,\n  height:50,",
              ),
            },
            {
              id: buildPresetId(type, "print"),
              label: "打印适配",
              content: buildDslTemplate(
                methodName,
                "  text:'sn',\n  height:70,\n  margin:12,\n  fontSize:14,\n  displayValue:true,",
              ),
            },
          ];
        case "Slider":
          return [
            {
              id: buildPresetId(type, "full"),
              label: "通用模板",
              content: buildDslTemplate(
                methodName,
                "  text: 'field',\n  disabled: false,\n  min: 0,\n  max: 100,\n  step: 1,\n  showInput: false,\n  showInputControls: true,\n  range: false,\n  vertical: false,\n  height: null,\n  marks: null,\n  showStops: false,\n  showTooltip: true,\n  formatTooltip: 'formatTooltip',\n  change: 'onChange',\n  input: 'onInput',",
              ),
            },
            {
              id: buildPresetId(type, "input"),
              label: "+",
              content: buildDslTemplate(
                methodName,
                "  text:'score',\n  min:0,\n  max:100,\n  step:1,\n  showInput:true,\n  change:'onScoreChange',",
              ),
            },
            {
              id: buildPresetId(type, "range"),
              label: "范围选择",
              content: buildDslTemplate(
                methodName,
                "  text:'range',\n  range:true,\n  min:0,\n  max:100,\n  step:5,",
              ),
            },
            {
              id: buildPresetId(type, "marks"),
              label: "离散鐐Marks",
              content: buildDslTemplate(
                methodName,
                "  text:'level',\n  min:0,\n  max:3,\n  step:1,\n  marks:{0:'低',1:'中',2:'高',3:'极高'},",
              ),
            },
            {
              id: buildPresetId(type, "vertical"),
              label: "垂直",
              content: buildDslTemplate(
                methodName,
                "  text:'volume',\n  vertical:true,\n  height:200,\n  min:0,\n  max:100,",
              ),
            },
            {
              id: buildPresetId(type, "tooltip"),
              label: "Tooltip",
              content: buildDslTemplate(
                methodName,
                "  text:'score',\n  formatTooltip:'fmtScoreTip',\n  showTooltip:true,",
              ),
            },
          ];
        case "Calendar":
          return [
            {
              id: buildPresetId(type, "full"),
              label: "通用模板",
              content: buildDslTemplate(
                methodName,
                "  text: 'calendarDate',\n  range: null,\n  firstDayOfWeek: 1,\n  dateCell: '',\n  monthChange: 'onMonthChange',",
              ),
            },
            {
              id: buildPresetId(type, "basic"),
              label: "基础",
              content: buildDslTemplate(methodName, "  text:'calendarDate',"),
            },
            {
              id: buildPresetId(type, "range"),
              label: "范围",
              content: buildDslTemplate(methodName, "  range:['2026-02-01','2026-02-29'],"),
            },
            {
              id: buildPresetId(type, "cell"),
              label: "日期内瀹",
              content: buildDslTemplate(
                methodName,
                "  text:'calendarDate',\n  dateCell:'renderDateCell',",
              ),
            },
            {
              id: buildPresetId(type, "events"),
              label: "结合数据源",
              content: buildDslTemplate(
                methodName,
                "  text:'calendarDate',\n  events:'calendarEvents',\n  dateCell:'renderDateCellWithEvents',",
              ),
            },
          ];
        case "Signature":
          return [
            {
              id: buildPresetId(type, "full"),
              label: "通用模板",
              content: buildDslTemplate(
                methodName,
                "  text: 'signatureDataUrl',\n  width: 600,\n  height: 240,\n  readonly: false,\n  disabled: false,\n\n  penColor: '#000',\n  backgroundColor: '#fff',\n  minWidth: 0.5,\n  maxWidth: 2.5,\n\n  clearText: '清空',\n  saveText: '保存',\n  required: false,\n  emptyTip: '请先签名',\n\n  change: 'onSignatureChange',\n  clear: 'onSignatureClear',\n  save: 'onSignatureSave',\n  validate: 'validateSignature',",
              ),
            },
            {
              id: buildPresetId(type, "basic"),
              label: "签名",
              content: buildDslTemplate(
                methodName,
                "  text:'signatureDataUrl',\n  width:600,\n  height:240,\n  change:'onSignatureChange',",
              ),
            },
            {
              id: buildPresetId(type, "readonly"),
              label: "回显",
              content: buildDslTemplate(
                methodName,
                "  text:'signatureDataUrl',\n  readonly:true,\n  width:600,\n  height:240,",
              ),
            },
            {
              id: buildPresetId(type, "required"),
              label: "必签校验",
              content: buildDslTemplate(
                methodName,
                "  text:'signatureDataUrl',\n  required:true,\n  emptyTip:'请签名后再提交',\n  validate:'validateSignature',",
              ),
            },
            {
              id: buildPresetId(type, "save"),
              label: "提交前强制保瀛",
              content: buildDslTemplate(
                methodName,
                "  text:'signatureDataUrl',\n  save:'saveSignatureThenSubmit',\n  clear:'onSignatureClear',",
              ),
            },
          ];
        default:
          return [];
      }
    },
  },
  {
    name: "input",
    types: ["Input", "InputNumber"],
    detail: (methodName, type) => [
      ...(type === "InputNumber"
        ? [
            {
              id: buildPresetId(type, "basic"),
              label: "计数器基础配置",
              content: buildDslTemplate(
                methodName,
                '  id: "inputField",\n  value: 0,\n  placeholder: "请输入",\n  controls: true,\n  step: 1,',
              ),
            },
            {
              id: buildPresetId(type, "status"),
              label: "计数器状态配缃",
              content: buildDslTemplate(
                methodName,
                "  disabled: false,\n  min: 0,\n  max: 100,\n  precision: 0,",
              ),
            },
            {
              id: buildPresetId(type, "full-dsl"),
              label: "完整 DSL 模板",
              content: buildDslTemplate(
                methodName,
                '  /** 基础配置 */\n  id: "inputField",\n  value: 0,\n  placeholder: "请输入",\n  controls: true,\n  step: 1,\n\n  /** 基础配置 */\n  disabled: false,\n  min: 0,\n  max: 100,\n  precision: 0,\n\n  /** 样式 */\n  style: {\n    width: "220px",\n  },\n  className: "custom-input-number",\n\n  /** 事件 */\n  onChange: {\n    action: "setVar",\n    target: "$vars.inputNumberValue",\n  },',
              ),
            },
          ]
        : [
            {
              id: buildPresetId(type, "basic"),
              label: "输入",
              content: buildDslTemplate(
                methodName,
                "  text: 'username',\n  placeholder: '请输入用户名',\n  disabled: false,\n  clearable: true,\n  maxlength: 50,\n  showWordLimit: false,\n  size: 'small',",
              ),
            },
            {
              id: buildPresetId(type, "icons"),
              label: "前后图标",
              content: buildDslTemplate(
                methodName,
                "  text: 'account',\n  placeholder: '请输入账号',\n  'prefix-icon': 'el-icon-user',\n  'suffix-icon': 'el-icon-view',",
              ),
            },
            {
              id: buildPresetId(type, "password"),
              label: "密码输入框",
              content: buildDslTemplate(
                methodName,
                "  text: 'password',\n  placeholder: '请输入密码',\n  type: 'password',\n  showPassword: true,\n  autocomplete: 'off',",
              ),
            },
            {
              id: buildPresetId(type, "readonly"),
              label: "输入框",
              content: buildDslTemplate(
                methodName,
                "  text: 'readonlyField',\n  placeholder: '\u8BF7\u8F93\u5165',\n  readonly: true,",
              ),
            },
            {
              id: buildPresetId(type, "disabled"),
              label: "禁用输入框",
              content: buildDslTemplate(
                methodName,
                "  text: 'disabledField',\n  placeholder: '禁用输入',\n  disabled: true,",
              ),
            },
            {
              id: buildPresetId(type, "textarea"),
              label: "多行文本",
              content: buildDslTemplate(
                methodName,
                "  text: 'remark',\n  type: 'textarea',\n  placeholder: '请输入备注',\n  rows: 4,\n  autosize: { minRows: 3, maxRows: 6 },\n  maxlength: 200,\n  showWordLimit: true,",
              ),
            },
            {
              id: buildPresetId(type, "autocomplete"),
              label: "本地输入建议",
              content: buildDslTemplate(
                methodName,
                "  text: 'restaurant',\n  placeholder: '请输入内容',\n  disabled: false,\n  autocomplete: true,\n  restaurants: [\n    { value: '值1' },\n    { value: '值2' },\n    { value: '值3' },\n  ],",
              ),
            },
            {
              id: buildPresetId(type, "remote"),
              label: "远程输入建议",
              content: buildDslTemplate(
                methodName,
                "  text: 'company',\n  placeholder: '请输入公司名称',\n  autocomplete: true,\n  remote: true,\n  fetch: 'fetchCompanyList',\n  debounce: 300,",
              ),
            },
            {
              id: buildPresetId(type, "affix"),
              label: "前后缀",
              content: buildDslTemplate(
                methodName,
                "  text: 'price',\n  placeholder: '请输入金额',\n  prepend: '￥',\n  append: '元',",
              ),
            },
            {
              id: buildPresetId(type, "append-button"),
              label: "尾部按钮",
              content: buildDslTemplate(
                methodName,
                "  text: 'search',\n  placeholder: '请输入关键词',\n  appendButton: {\n    text: '搜索',\n    icon: 'el-icon-search',\n    click: 'onSearch',\n  },",
              ),
            },
            {
              id: buildPresetId(type, "number"),
              label: "数字限制",
              content: buildDslTemplate(
                methodName,
                "  text: 'age',\n  placeholder: '请输入年龄',\n  type: 'number',\n  min: 1,\n  max: 120,",
              ),
            },
            {
              id: buildPresetId(type, "events"),
              label: "校验与事浠",
              content: buildDslTemplate(
                methodName,
                "  text: 'email',\n  placeholder: '请输入邮箱',\n  clearable: true,\n  change: 'onEmailChange',\n  input: 'onEmailInput',\n  blur: 'onEmailBlur',",
              ),
            },
            {
              id: buildPresetId(type, "rules"),
              label: "表单校验规则",
              content: buildDslTemplate(
                methodName,
                "  text: 'phone',\n  placeholder: '请输入手机号',\n  maxlength: 11,\n  rules: [\n    { required: true, message: '手机号不能为空', trigger: 'blur' },\n    { pattern: /^1\\d{10}$/, message: '手机号格式错误', trigger: 'blur' },\n  ],",
              ),
            },
            {
              id: buildPresetId(type, "full-dsl"),
              label: "完整豪华鐗",
              content: buildDslTemplate(
                methodName,
                "  text: 'keyword',\n  placeholder: '请输入关键词',\n  clearable: true,\n  size: 'small',\n  'prefix-icon': 'el-icon-search',\n  autocomplete: true,\n  restaurants: [\n    { value: 'Vue' },\n    { value: 'React' },\n    { value: 'Element UI' },\n  ],\n  debounce: 300,\n  change: 'onKeywordChange',",
              ),
            },
          ]),
    ],
    style: (type) => [
      {
        id: buildPresetId(type, "base"),
        label: "输入框本浣",
        content:
          "#domId .el-input__inner{\n  height: 44px; line-height: 44px;\n  font-size: 16px; color: #303133;\n  border-radius: 6px;\n}\n#domId .el-input__inner::placeholder{ color:#c0c4cc; }",
      },
      {
        id: buildPresetId(type, "hover"),
        label: "Hover / Focus",
        content:
          "#domId .el-input__inner:hover{ border-color:#409EFF; }\n#domId .el-input__inner:focus{\n  border-color:#409EFF;\n  box-shadow:0 0 0 2px rgba(64,158,255,.15);\n}",
      },
      {
        id: buildPresetId(type, "prefix"),
        label: "前缀/后缀区域",
        content:
          "#domId .el-input__prefix,\n#domId .el-input__suffix{ color:#909399; }\n#domId .el-input__prefix i,\n#domId .el-input__suffix i{ font-size:16px; }\n#domId .el-input__suffix:hover{ color:#409EFF; }",
      },
      {
        id: buildPresetId(type, "clear"),
        label: "清空图标",
        content:
          "#domId .el-input__clear{ color:#c0c4cc; }\n#domId .el-input__clear:hover{ color:#409EFF; }",
      },
      {
        id: buildPresetId(type, "textarea"),
        label: "Textarea",
        content:
          "#domId .el-textarea__inner{\n  font-size: 14px;\n  border-radius: 8px;\n  line-height: 1.6;\n  padding: 10px 12px;\n}\n#domId .el-textarea__inner:focus{\n  border-color:#409EFF;\n  box-shadow:0 0 0 2px rgba(64,158,255,.12);\n}",
      },
      {
        id: buildPresetId(type, "disabled"),
        label: "禁用/",
        content:
          "#domId .el-input.is-disabled .el-input__inner,\n#domId .el-textarea.is-disabled .el-textarea__inner{\n  background:#f5f7fa;\n  color:#c0c4cc;\n  cursor:not-allowed;\n}",
      },
      {
        id: buildPresetId(type, "error"),
        label: "输入错误",
        content:
          "#domId .el-form-item.is-error .el-input__inner,\n#domId .el-form-item.is-error .el-textarea__inner{\n  border-color:#f56c6c;\n}\n#domId .el-form-item.is-error .el-input__inner:focus{\n  box-shadow:0 0 0 2px rgba(245,108,108,.15);\n}",
      },
      {
        id: buildPresetId(type, "autocomplete"),
        label: "Autocomplete 面板",
        content:
          '/* 建议：给 el-autocomplete 使用 popper-class="ac-popper" */\n.ac-popper{ border-radius:12px; overflow:hidden; }\n.ac-popper .el-autocomplete-suggestion__list li:hover{ background:#ecf5ff; color:#409EFF; }',
      },
    ],
  },
  {
    name: "switch",
    types: ["Switch"],
    detail: (methodName) => [
      {
        id: buildPresetId("Switch", "full"),
        label: "通用模板",
        content: buildDslTemplate(
          methodName,
          "  text: 'field',\n  disabled: false,\n  size: 'small',\n  activeText: '',\n  inactiveText: '',\n  activeValue: true,\n  inactiveValue: false,\n  activeColor: '',\n  inactiveColor: '',\n  validateEvent: true,\n  beforeChange: '',\n  change: 'onChange',",
        ),
      },
      {
        id: buildPresetId("Switch", "text"),
        label: "文本开鍏",
        content: buildDslTemplate(
          methodName,
          "  text:'enabled',\n  activeText:'启用',\n  inactiveText:'禁用',\n  change:'onEnabledChange',",
        ),
      },
      {
        id: buildPresetId("Switch", "value"),
        label: "自定义值",
        content: buildDslTemplate(
          methodName,
          "  text:'flag',\n  activeValue:1,\n  inactiveValue:0,\n  change:'onFlagChange',",
        ),
      },
      {
        id: buildPresetId("Switch", "async"),
        label: "异步拦截",
        content: buildDslTemplate(
          methodName,
          "  text:'online',\n  beforeChange:'beforeToggleOnline',\n  change:'onOnlineChange',",
        ),
      },
      {
        id: buildPresetId("Switch", "row"),
        label: "行内开鍏",
        content: buildDslTemplate(
          methodName,
          "  text: 'row.enabled',\n  disabled: 'row.locked',\n  change: 'toggleRowEnabled(row)',",
        ),
      },
    ],
    style: (type) => [
      {
        id: buildPresetId(type, "size"),
        label: "结构",
        content:
          "#domId .el-switch__core{\n  width:48px!important;\n  height:24px;\n  border-radius:999px;\n}\n#domId .el-switch__core:after{\n  width:20px; height:20px;\n  top:1px;\n}",
      },
      {
        id: buildPresetId(type, "color"),
        label: "开/关颜鑹",
        content:
          "#domId .el-switch .el-switch__core{ background:#dcdfe6; border-color:#dcdfe6; }\n#domId .el-switch.is-checked .el-switch__core{ background:#67c23a; border-color:#67c23a; }",
      },
    ],
  },
  {
    name: "image",
    types: ["Image"],
    style: (type) => [
      {
        id: buildPresetId(type, "size"),
        label: "容器/图片本体",
        content:
          "#domId .el-image { width: 120px; height: 120px; display: inline-block; }\n#domId .el-image__inner { width: 100%; height: 100%; object-fit: cover; }",
      },
      {
        id: buildPresetId(type, "radius"),
        label: "//阴影",
        content:
          "#domId .el-image { border-radius: 10px; overflow: hidden; border: 1px solid #ebeef5; }\n#domId .el-image__inner { border-radius: 10px; }",
      },
      {
        id: buildPresetId(type, "hover"),
        label: "Hover 放大/遮罩",
        content:
          "#domId .el-image__inner { transition: transform .2s ease, opacity .2s ease; }\n#domId .el-image:hover .el-image__inner { transform: scale(1.03); opacity: .95; cursor: pointer; }",
      },
      {
        id: buildPresetId(type, "placeholder"),
        label: "占位/错误插槽",
        content:
          "#domId .el-image__placeholder,\n#domId .el-image__error {\n  background: #f5f7fa;\n  color: #909399;\n  font-size: 12px;\n}",
      },
      {
        id: buildPresetId(type, "viewer"),
        label: "预览查看器",
        content:
          "/* 如果预览是全局挂载，#domId 可能覆盖不到 */\n.el-image-viewer__close { font-size: 28px; }",
      },
    ],
  },
  {
    name: "select",
    types: ["Select", "Cascader"],
    detail: (methodName, type) => [
      ...(type === "Select"
        ? [
            {
              id: buildPresetId(type, "full"),
              label: "通用模板（全字段",
              content: buildDslTemplate(
                methodName,
                "  text: 'field',\n  placeholder: '请选择',\n  disabled: false,\n  clearable: true,\n  filterable: false,\n  multiple: false,\n  collapseTags: false,\n  valueKey: 'value',\n  options: [],\n  optionLabel: 'label',\n  optionValue: 'value',\n  optionDisabled: 'disabled',\n  loading: false,\n  noDataText: '无数据',\n  popperClass: '',\n  size: 'small',\n  change: 'onChange',\n  visibleChange: 'onVisibleChange',\n  clear: 'onClear',\n  removeTag: 'onRemoveTag',",
              ),
            },
            {
              id: buildPresetId(type, "basic"),
              label: "单选",
              content: buildDslTemplate(
                methodName,
                "  text: 'city',\n  placeholder: '请选择城市',\n  options: [{ label:'北京', value:'bj' }, { label:'上海', value:'sh' }],\n  change: 'onCityChange',",
              ),
            },
            {
              id: buildPresetId(type, "clearable"),
              label: "可清空",
              content: buildDslTemplate(
                methodName,
                "  text: 'dept',\n  placeholder: '请选择',\n  clearable: true,",
              ),
            },
            {
              id: buildPresetId(type, "disabled"),
              label: "禁用",
              content: buildDslTemplate(
                methodName,
                "  text: 'dept',\n  placeholder: '请选择',\n  disabled: true,",
              ),
            },
            {
              id: buildPresetId(type, "filterable"),
              label: "可搜索",
              content: buildDslTemplate(
                methodName,
                "  text: 'userId',\n  placeholder: '搜索用户',\n  filterable: true,\n  options: 'userOptions',",
              ),
            },
            {
              id: buildPresetId(type, "multiple"),
              label: "+ 折叠 Tag",
              content: buildDslTemplate(
                methodName,
                "  text: 'tags',\n  placeholder: '请选择标签',\n  multiple: true,\n  collapseTags: true,\n  clearable: true,\n  options: 'tagOptions',",
              ),
            },
            {
              id: buildPresetId(type, "remote"),
              label: "远程搜索",
              content: buildDslTemplate(
                methodName,
                "  text: 'companyId',\n  placeholder: '输入关键词搜索',\n  filterable: true,\n  remote: true,\n  reserveKeyword: true,\n  remoteMethod: 'remoteSearchCompany',\n  loading: 'companyLoading',\n  options: 'companyOptions',",
              ),
            },
            {
              id: buildPresetId(type, "allow-create"),
              label: "可创寤",
              content: buildDslTemplate(
                methodName,
                "  text: 'customItem',\n  placeholder: '输入创建',\n  filterable: true,\n  allowCreate: true,\n  defaultFirstOption: true,\n  options: 'customOptions',",
              ),
            },
            {
              id: buildPresetId(type, "group"),
              label: "分组（Option Group",
              content: buildDslTemplate(
                methodName,
                "  text: 'food',\n  placeholder: '请选择',\n  group: true,\n  groups: [\n    { label:'水果', options:[{label:'苹果',value:'apple'}] },\n    { label:'主食', options:[{label:'米饭',value:'rice'}] },\n  ],",
              ),
            },
            {
              id: buildPresetId(type, "max"),
              label: "选择上限",
              content: buildDslTemplate(
                methodName,
                "  text: 'members',\n  multiple: true,\n  max: 5,\n  exceed: 'onSelectExceed',\n  options: 'memberOptions',",
              ),
            },
            {
              id: buildPresetId(type, "fill"),
              label: "联动回填",
              content: buildDslTemplate(
                methodName,
                "  text: 'productId',\n  options: 'productOptions',\n  change: 'onProductChange',\n  fill: {\n    from: 'option',\n    map: { productName:'label', price:'price' },\n  },",
              ),
            },
          ]
        : [
            {
              id: buildPresetId(type, "basic"),
              label: "选择配置",
              content: buildDslTemplate(
                methodName,
                '  id: "selectField",\n  value: "",\n  placeholder: "请选择",\n  clearable: true,\n  options: [\n    { label: "选项A", value: "A" },\n    { label: "选项B", value: "B" },\n  ],',
              ),
            },
            {
              id: buildPresetId(type, "multi"),
              label: "多选配缃",
              content: buildDslTemplate(
                methodName,
                "  multiple: true,\n  filterable: true,\n  collapseTags: true,",
              ),
            },
            {
              id: buildPresetId(type, "full-dsl"),
              label: "完整 DSL 模板",
              content: buildDslTemplate(
                methodName,
                '  /** 基础配置 */\n  id: "selectField",\n  value: "",\n  placeholder: "请选择",\n  clearable: true,\n  options: [\n    { label: "选项A", value: "A" },\n    { label: "选项B", value: "B" },\n  ],\n\n  /** 基础配置 */\n  disabled: false,\n\n  /** 样式 */\n  style: {\n    width: "220px",\n  },\n  className: "custom-select",\n\n  /** 事件 */\n  onChange: {\n    action: "setVar",\n    target: "$vars.selectValue",\n  },',
              ),
            },
          ]),
    ],
    style: (type) => [
      {
        id: buildPresetId(type, "input"),
        label: "输入框",
        content:
          "#domId .el-select .el-input__inner{\n  height:44px; line-height:44px;\n  border-radius:6px; font-size:16px;\n}\n#domId .el-select .el-input__inner:hover{ border-color:#409EFF; }",
      },
      {
        id: buildPresetId(type, "icons"),
        label: "箭头/清空图标",
        content:
          "#domId .el-select .el-input__suffix{ color:#909399; }\n#domId .el-select .el-input__suffix:hover{ color:#409EFF; }\n#domId .el-select .el-icon-circle-close{ color:#c0c4cc; }\n#domId .el-select .el-icon-circle-close:hover{ color:#409EFF; }",
      },
      {
        id: buildPresetId(type, "tags"),
        label: "Tag",
        content:
          "#domId .el-select .el-tag{\n  border-radius:999px;\n  height:24px;\n  line-height:22px;\n  margin: 4px 4px 4px 0;\n}\n#domId .el-select .el-tag__close{ color:#909399; }\n#domId .el-select .el-tag__close:hover{ color:#409EFF; }",
      },
      {
        id: buildPresetId(type, "dropdown"),
        label: "下拉面板",
        content:
          '/* 建议：select 使用 popper-class="select-popper" */\n.select-popper{ border-radius:12px; overflow:hidden; }\n.select-popper .el-select-dropdown__item{ font-size:14px; line-height:38px; }\n.select-popper .el-select-dropdown__item:hover{ background:#ecf5ff; color:#409EFF; }\n.select-popper .el-select-dropdown__item.selected{ color:#409EFF; font-weight:600; }\n.select-popper .el-select-dropdown__empty{ color:#909399; padding:12px 0; }',
      },
      {
        id: buildPresetId(type, "group"),
        label: "分组",
        content:
          ".select-popper .el-select-group__title{ color:#909399; font-size:12px; padding:8px 12px; }",
      },
      {
        id: buildPresetId(type, "disabled"),
        label: "禁用",
        content: "#domId .el-select .el-input.is-disabled .el-input__inner{ background:#f5f7fa; }",
      },
    ],
  },
  {
    name: "choice",
    types: ["Switch", "Radio", "Checkbox"],
    detail: (methodName, type) => [
      {
        id: buildPresetId(type, "basic"),
        label: "选择配置",
        content: buildDslTemplate(
          methodName,
          '  id: "choiceField",\n  value: "",\n  options: [\n    { label: "选项A", value: "A" },\n    { label: "选项B", value: "B" },\n  ],',
        ),
      },
      {
        id: buildPresetId(type, "status"),
        label: "选择状态配缃",
        content: buildDslTemplate(methodName, '  disabled: false,\n  size: "small",'),
      },
      {
        id: buildPresetId(type, "full-dsl"),
        label: "完整 DSL 模板",
        content: buildDslTemplate(
          methodName,
          '  /** 基础配置 */\n  id: "choiceField",\n  value: "",\n  options: [\n    { label: "选项A", value: "A" },\n    { label: "选项B", value: "B" },\n  ],\n\n  /** 基础配置 */\n  disabled: false,\n\n  /** 样式 */\n  className: "custom-choice",\n\n  /** 事件 */\n  onChange: {\n    action: "setVar",\n    target: "$vars.choiceValue",\n  },',
        ),
      },
    ],
    style: (type) => [
      {
        id: buildPresetId(type, "active"),
        label: "激活色",
        content: "#domId .is-checked, #domId .is-active { color: #409eff; }",
      },
      {
        id: buildPresetId(type, "border"),
        label: "颜色",
        content: "#domId .el-radio, #domId .el-checkbox { border-color: #dcdfe6; }",
      },
      {
        id: buildPresetId(type, "size"),
        label: "组件间距",
        content: "#domId .el-radio, #domId .el-checkbox { margin-right: 12px; }",
      },
    ],
  },
  {
    name: "table",
    types: ["Table", "BigDataTable"],
    detail: (methodName, type) => {
      if (type === "Table") {
        return [
          {
            id: buildPresetId(type, "full"),
            label: "通用模板（高密度",
            content: buildDslTemplate(
              methodName,
              "  data: 'tableData',\n  rowKey: 'id',\n  border: true,\n  stripe: true,\n  size: 'small',\n  height: null,\n  maxHeight: null,\n  fit: true,\n  highlightCurrentRow: true,\n  showHeader: true,\n  emptyText: '暂无数据',\n\n  selectable: false,\n  reserveSelection: true,\n\n  defaultSort: { prop: 'id', order: 'descending' },\n\n  selectionChange: 'onSelectionChange',\n  rowClick: 'onRowClick',\n  rowDblclick: 'onRowDblclick',\n  sortChange: 'onSortChange',\n  filterChange: 'onFilterChange',\n  currentChange: 'onCurrentChange',\n\n  columns: [\n    // {type:'selection'|'index'|'expand', prop, label, width, minWidth, fixed, align, headerAlign,\n    //  sortable, filters, filterMethod, showOverflowTooltip,\n    //  render: this.text()/this.image()/slotName, actions:[...], editable:{...}}\n  ],",
            ),
          },
          {
            id: buildPresetId(type, "basic"),
            label: "表格",
            content: buildDslTemplate(
              methodName,
              "  data:'tableData',\n  columns:[\n    { prop:'name', label:'姓名', minWidth:120 },\n    { prop:'age', label:'年龄', width:80 },\n  ],",
            ),
          },
          {
            id: buildPresetId(type, "action"),
            label: "选择 + 序号 + 固定列 + 操作列",
            content: buildDslTemplate(
              methodName,
              "  data:'tableData',\n  rowKey:'id',\n  columns:[\n    { type:'selection', width:48 },\n    { type:'index', label:'#', width:60 },\n    { prop:'name', label:'姓名', minWidth:120, fixed:'left' },\n    { prop:'address', label:'地址', minWidth:220, showOverflowTooltip:true },\n    { label:'操作', width:180, fixed:'right',\n      actions:[\n        this.button({ text:'编辑', type:'text', click:'editRow(row)' }),\n        this.button({ text:'删除', type:'text', click:'deleteRow(row)', confirm:'确定删除该行吗？' }),\n      ]\n    }\n  ],",
            ),
          },
          {
            id: buildPresetId(type, "sort"),
            label: "排序/远程排序",
            content: buildDslTemplate(
              methodName,
              "  data:'tableData',\n  columns:[\n    { prop:'age', label:'年龄', width:80, sortable:true },\n    { prop:'score', label:'分数', width:90, sortable:'custom' },\n  ],\n  sortChange:'onSortChange',",
            ),
          },
          {
            id: buildPresetId(type, "filter"),
            label: "Filters",
            content: buildDslTemplate(
              methodName,
              "  data:'tableData',\n  columns:[\n    { prop:'status', label:'状态', width:110,\n      filters:[{text:'启用',value:1},{text:'禁用',value:0}],\n      filterMethod:'filterStatus',\n    }\n  ],\n  filterChange:'onFilterChange',",
            ),
          },
          {
            id: buildPresetId(type, "render-text"),
            label: "渲染：文本省略+Tooltip",
            content: buildDslTemplate(
              methodName,
              "  data:'tableData',\n  columns:[\n    { label:'备注', minWidth:240,\n      render: this.text({ text:'remark', ellipsis:true, tooltip:true, maxWidth:200, empty:'--' })\n    }\n  ],",
            ),
          },
          {
            id: buildPresetId(type, "render-tag"),
            label: "渲染：Tag",
            content: buildDslTemplate(
              methodName,
              "  data:'tableData',\n  columns:[\n    { label:'状态', width:110,\n      render: this.text({\n        text:'status',\n        tag:true,\n        dict:{ 0:{label:'禁用',type:'info'}, 1:{label:'启用',type:'success'} }\n      })\n    }\n  ],",
            ),
          },
          {
            id: buildPresetId(type, "render-image"),
            label: "渲染：图片预览列",
            content: buildDslTemplate(
              methodName,
              "  data:'tableData',\n  columns:[\n    { label:'封面', width:100,\n      render: this.image({ text:'cover', width:64, height:40, fit:'cover', preview:true })\n    }\n  ],",
            ),
          },
          {
            id: buildPresetId(type, "expand"),
            label: "展开行 Expand",
            content: buildDslTemplate(
              methodName,
              "  data:'tableData',\n  columns:[\n    { type:'expand', width:48, expand:'renderExpand(row)' },\n    { prop:'name', label:'名称', minWidth:160 },\n  ],",
            ),
          },
          {
            id: buildPresetId(type, "tree"),
            label: "树形表格",
            content: buildDslTemplate(
              methodName,
              "  data:'treeTableData',\n  rowKey:'id',\n  tree:true,\n  treeProps:{ children:'children', hasChildren:'hasChildren' },\n  columns:[\n    { prop:'name', label:'节点', minWidth:220 },\n    { prop:'code', label:'编码', width:140 },\n  ],",
            ),
          },
          {
            id: buildPresetId(type, "summary"),
            label: "合并单元格 + 表尾合计",
            content: buildDslTemplate(
              methodName,
              "  data:'tableData',\n  spanMethod:'spanMethod',\n  showSummary:true,\n  summaryMethod:'summaryMethod',\n  columns:'columns',",
            ),
          },
          {
            id: buildPresetId(type, "editable"),
            label: "可编辑单元格",
            content: buildDslTemplate(
              methodName,
              "  data:'tableData',\n  columns:[\n    { prop:'name', label:'姓名', minWidth:140,\n      editable: { editor: this.input({ text:'row.name', placeholder:'请输入' }), save:'saveCell(row)' }\n    },\n    { prop:'status', label:'状态', width:120,\n      editable: { editor: this.select({ text:'row.status', options:'statusOptions' }), save:'saveCell(row)' }\n    },\n  ],",
            ),
          },
        ];
      }
      const dataDefault = getManifestDefaultValue(type, "data") || [];
      const columnsDefault = getManifestDefaultValue(type, "columns") || [
        { prop: "name", label: "名称" },
        { prop: "value", label: "值" },
      ];
      return [
        {
          id: buildPresetId(type, "basic"),
          label: "表格配置",
          content: buildDslTemplate(
            methodName,
            `  id: "tableList",\n  data: ${formatDslValue(
              dataDefault,
              2,
            )},\n  columns: ${formatDslValue(columnsDefault, 2)},`,
          ),
        },
        {
          id: buildPresetId(type, "layout"),
          label: "表格布局配置",
          content: buildDslTemplate(methodName, "  stripe: true,\n  border: true,\n  height: 360,"),
        },
        {
          id: buildPresetId(type, "full-dsl"),
          label: "完整 DSL 模板",
          content: buildDslTemplate(
            methodName,
            `  /** 基础配置 */\n  id: "tableList",\n  data: ${formatDslValue(
              dataDefault,
              2,
            )},\n  columns: ${formatDslValue(columnsDefault, 2)},

  /** 表格属性 */
  stripe: true,
  border: true,
  rowKey: "id",
  height: 360,

  /** 样式 */
  className: "custom-table",

  /** 事件 */
  onRowClick: {
    action: "setVar",
    target: "$vars.currentRow",
  },`,
          ),
        },
      ];
    },
    style: (type) => [
      {
        id: buildPresetId(type, "header"),
        label: "表头背景",
        content: "#domId .el-table__header th { background: #f5f7fa; }",
      },
      {
        id: buildPresetId(type, "row"),
        label: "行高",
        content: "#domId .el-table__body td { height: 44px; }",
      },
      {
        id: buildPresetId(type, "border"),
        label: "颜色",
        content: "#domId .el-table { border-color: #dcdfe6; }",
      },
    ],
  },
  {
    name: "tree",
    types: ["Tree"],
    detail: (methodName, type) => [
      {
        id: buildPresetId(type, "full"),
        label: "通用模板",
        content: buildDslTemplate(
          methodName,
          "  data: 'treeData',\n  'node-key': 'id',\n  props: { label:'name', children:'children', isLeaf:'leaf' },\n  showCheckbox: false,\n  checkStrictly: false,\n  defaultExpandAll: false,\n  defaultExpandedKeys: [],\n  defaultCheckedKeys: [],\n  highlightCurrent: true,\n  expandOnClickNode: true,\n  checkOnClickNode: false,\n  accordion: false,\n\n  lazy: false,\n  load: '',\n\n  filterable: false,\n  filterNodeMethod: '',\n\n  nodeClick: 'onNodeClick',\n  check: 'onTreeCheck',\n  currentChange: 'onCurrentChange',",
        ),
      },
      {
        id: buildPresetId(type, "basic"),
        label: "基础",
        content: buildDslTemplate(
          methodName,
          "  data:'treeData',\n  'node-key':'id',\n  props:{label:'name',children:'children'},\n  nodeClick:'onNodeClick',",
        ),
      },
      {
        id: buildPresetId(type, "check"),
        label: "勾选树（父子联动）",
        content: buildDslTemplate(
          methodName,
          "  data:'treeData',\n  'node-key':'id',\n  showCheckbox:true,\n  checkStrictly:false,\n  defaultCheckedKeys:'checkedIds',\n  check:'onTreeCheck',",
        ),
      },
      {
        id: buildPresetId(type, "check-strict"),
        label: "勾选树（父子不联动",
        content: buildDslTemplate(
          methodName,
          "  data:'treeData',\n  'node-key':'id',\n  showCheckbox:true,\n  checkStrictly:true,\n  check:'onTreeCheck',",
        ),
      },
      {
        id: buildPresetId(type, "lazy"),
        label: "懒加载",
        content: buildDslTemplate(
          methodName,
          "  lazy:true,\n  'node-key':'id',\n  load:'loadTreeNode',\n  props:{ label:'name', children:'children', isLeaf:'leaf' },",
        ),
      },
      {
        id: buildPresetId(type, "filter"),
        label: "搜索过滤",
        content: buildDslTemplate(
          methodName,
          "  data:'treeData',\n  filterable:true,\n  filterNodeMethod:'filterTreeNode',",
        ),
      },
      {
        id: buildPresetId(type, "context"),
        label: "右键菜单",
        content: buildDslTemplate(
          methodName,
          "  data:'treeData',\n  contextmenu:true,\n  contextItems:[\n    { command:'add', label:'新增' },\n    { command:'edit', label:'编辑' },\n    { command:'del', label:'删除' },\n  ],\n  command:'onTreeCommand(node,data)',",
        ),
      },
    ],
    style: (type) => [
      {
        id: buildPresetId(type, "node"),
        label: "节点间距",
        content: "#domId .el-tree-node__content { padding: 6px 8px; }",
      },
      {
        id: buildPresetId(type, "hover"),
        label: "悬停背景",
        content: "#domId .el-tree-node__content:hover { background: #f5f7fa; }",
      },
      {
        id: buildPresetId(type, "font"),
        label: "节点字体",
        content: "#domId .el-tree-node__label { font-size: 13px; }",
      },
    ],
  },
  {
    name: "menu",
    types: ["Dropdown", "Menu"],
    detail: (methodName, type) => [
      ...(type === "Dropdown"
        ? [
            {
              id: buildPresetId(type, "full"),
              label: "通用模板",
              content: buildDslTemplate(
                methodName,
                "  text: '操作',\n  trigger: 'hover',\n  type: '',\n  size: 'small',\n  splitButton: false,\n  placement: 'bottom-end',\n  hideOnClick: true,\n\n  click: 'onMainClick',\n  command: 'onCommand',\n\n  items: [\n    // {command,label,icon,disabled,divided/divider:true}\n  ],",
              ),
            },
            {
              id: buildPresetId(type, "basic"),
              label: "普通下拉",
              content: buildDslTemplate(
                methodName,
                "  text:'操作',\n  trigger:'click',\n  items:[\n    { command:'refresh', label:'刷新', icon:'el-icon-refresh' },\n    { command:'export', label:'导出', icon:'el-icon-download' },\n  ],\n  command:'onDropdownCommand',",
              ),
            },
            {
              id: buildPresetId(type, "split"),
              label: "分裂按钮",
              content: buildDslTemplate(
                methodName,
                "  text:'操作',\n  splitButton:true,\n  type:'primary',\n  click:'onMainClick',\n  items:[ {command:'a',label:'动作A'}, {command:'b',label:'动作B'} ],\n  command:'onCommand',",
              ),
            },
            {
              id: buildPresetId(type, "divider"),
              label: "分割线 + 禁用项",
              content: buildDslTemplate(
                methodName,
                "  text:'操作',\n  items:[\n    { command:'detail', label:'详情' },\n    { divider:true },\n    { command:'delete', label:'删除', disabled:true },\n  ],\n  command:'onCommand',",
              ),
            },
            {
              id: buildPresetId(type, "confirm"),
              label: "危险操作确认",
              content: buildDslTemplate(
                methodName,
                "  text:'操作',\n  items:[\n    { command:'reset', label:'重置', confirm:'确认重置？' },\n  ],\n  command:'onCommand',",
              ),
            },
          ]
        : []),
      ...(type === "Menu"
        ? [
            {
              id: buildPresetId(type, "full"),
              label: "通用模板",
              content: buildDslTemplate(
                methodName,
                "  mode: 'vertical',\n  active: 'dashboard',\n  collapse: false,\n  uniqueOpened: true,\n  router: false,\n  backgroundColor: '',\n  textColor: '',\n  activeTextColor: '',\n  defaultOpeneds: [],\n  select: 'onMenuSelect',\n\n  filter: '',\n  items: [\n    // {index,title,icon,disabled,children:[...]}\n  ],",
              ),
            },
            {
              id: buildPresetId(type, "sidebar"),
              label: "侧边栏（多级）",
              content: buildDslTemplate(
                methodName,
                "  active:'dashboard',\n  items:[\n    { index:'dashboard', title:'\u4EEA\u8868\u76D8', icon:'el-icon-data-analysis' },\n    { index:'sys', title:'系统', icon:'el-icon-setting', children:[\n      { index:'user', title:'用户管理' },\n      { index:'role', title:'角色管理' },\n    ] }\n  ],\n  select:'onMenuSelect',",
              ),
            },
            {
              id: buildPresetId(type, "top"),
              label: "顶部导航（Horizontal",
              content: buildDslTemplate(
                methodName,
                "  mode:'horizontal',\n  active:'home',\n  items:[ {index:'home',title:'首页'}, {index:'docs',title:'文档'} ],\n  select:'onTopMenuSelect',",
              ),
            },
            {
              id: buildPresetId(type, "collapse"),
              label: "折叠侧边栏",
              content: buildDslTemplate(
                methodName,
                "  collapse:true,\n  active:'dashboard',\n  items:'menuItems',\n  select:'onMenuSelect',",
              ),
            },
            {
              id: buildPresetId(type, "filter"),
              label: "权限过滤",
              content: buildDslTemplate(
                methodName,
                "  items:'menuItems',\n  filter:'filterMenuByPerm',\n  active:'dashboard',",
              ),
            },
          ]
        : []),
      ...(type !== "Dropdown" && type !== "Menu"
        ? [
            {
              id: buildPresetId(type, "basic"),
              label: "菜单配置",
              content: buildDslTemplate(
                methodName,
                '  id: "menuNav",\n  label: "菜单配置",\n  type: "Menu",\n  props: {\n    defaultActive: "2",\n    items: [\n      { index: "1", label: "导航一", icon: "location" },\n      { index: "2", label: "导航二", icon: "menu" },\n      { index: "3", label: "导航三", icon: "document", disabled: true },\n      { index: "4", label: "导航四", icon: "setting" },\n    ],\n  },',
              ),
            },
            {
              id: buildPresetId(type, "trigger"),
              label: "菜单触发配置",
              content: buildDslTemplate(methodName, '  trigger: "click",\n  splitButton: false,'),
            },
            {
              id: buildPresetId(type, "full-dsl"),
              label: "完整 DSL 模板",
              content: buildDslTemplate(
                methodName,
                '  /** 基础配置 */\n  id: "menuNav",\n  items: [\n    { label: "编辑", command: "edit" },\n    { label: "删除", command: "delete" },\n  ],\n\n  /** 基础配置 */\n  disabled: false,\n\n  /** 样式 */\n  className: "custom-menu",\n\n  /** 事件 */\n  onCommand: {\n    action: "setVar",\n    target: "$vars.menuCommand",\n  },',
              ),
            },
          ]
        : []),
    ],
    style: (type) => [
      {
        id: buildPresetId(type, "active"),
        label: "菜单激活色",
        content: "#domId .is-active { color: #409eff; }",
      },
      {
        id: buildPresetId(type, "bg"),
        label: "菜单背景",
        content: "#domId { background: #ffffff; }",
      },
      {
        id: buildPresetId(type, "hover"),
        label: "菜单悬停",
        content: "#domId .el-menu-item:hover { background: #f5f7fa; }",
      },
    ],
  },
  {
    name: "tabs",
    types: ["Tabs", "Collapse"],
    detail: (methodName, type) => [
      {
        id: buildPresetId(type, "basic"),
        label: "Tabs/折叠配置",
        content: buildDslTemplate(
          methodName,
          '  /* Tabs/\u6298\u53E0\u914D\u7F6E */\n  id: "tabsPanel",\n  label: "Tabs/\u6298\u53E0\u914D\u7F6E",\n  props: {\n    modelValue: "tab1",\n    activeName: "tab1",\n    tabs: [\n      { name: "tab1", label: "\u6807\u7B7E1" },\n      { name: "tab2", label: "\u6807\u7B7E2" },\n      { name: "tab3", label: "\u6807\u7B7E3" },\n    ],\n  },',
        ),
      },
      {
        id: buildPresetId(type, "type"),
        label: "样式类型配置",
        content: buildDslTemplate(methodName, '  type: "card",\n  stretch: true,'),
      },
      {
        id: buildPresetId(type, "full-dsl"),
        label: "完整 DSL 模板",
        content: buildDslTemplate(
          methodName,
          '  /** \u57FA\u7840 */\n  id: "tabsPanel",\n  modelValue: "tab1",\n  activeName: "tab1",\n  tabs: [\n    { name: "tab1", label: "\u6807\u7B7E1" },\n    { name: "tab2", label: "\u6807\u7B7E2" },\n  ],\n\n  /** \u5C5E\u6027 */\n  type: "card",\n  stretch: true,\n\n  /** \u6837\u5F0F */\n  className: "custom-tabs",\n\n  /** \u4E8B\u4EF6 */\n  onTabClick: {\n    action: "setVar",\n    target: "$vars.activeTab",\n  },',
        ),
      },
    ],
    style: (type) => [
      {
        id: buildPresetId(type, "active"),
        label: "激活色",
        content: "#domId .el-tabs__item.is-active { color: #409eff; }",
      },
      {
        id: buildPresetId(type, "bar"),
        label: "底部条",
        content: "#domId .el-tabs__active-bar { background: #409eff; }",
      },
    ],
  },
  {
    name: "pagination",
    types: ["Pagination"],
    detail: (methodName, type) => [
      {
        id: buildPresetId(type, "basic"),
        label: "分页配置",
        content: buildDslTemplate(
          methodName,
          '  id: "pagination",\n  total: 200,\n  pageSize: 20,\n  currentPage: 1,',
        ),
      },
      {
        id: buildPresetId(type, "layout"),
        label: "分页布局配置",
        content: buildDslTemplate(
          methodName,
          '  layout: "total, sizes, prev, pager, next, jumper",\n  pageSizes: [10, 20, 50, 100],',
        ),
      },
      {
        id: buildPresetId(type, "full-dsl"),
        label: "完整 DSL 模板",
        content: buildDslTemplate(
          methodName,
          '  /** \u57FA\u7840 */\n  id: "pagination",\n  total: 200,\n  pageSize: 20,\n  currentPage: 1,\n\n  /** \u5E03\u5C40 */\n  layout: "total, sizes, prev, pager, next, jumper",\n  pageSizes: [10, 20, 50, 100],\n\n  /** \u6837\u5F0F */\n  className: "custom-pagination",\n\n  /** \u4E8B\u4EF6 */\n  onChange: {\n    action: "setVar",\n    target: "$vars.pageIndex",\n  },',
        ),
      },
    ],
    style: (type) => [
      {
        id: buildPresetId(type, "active"),
        label: "分页激活色",
        content: "#domId .el-pagination .is-active { color: #409eff; }",
      },
      {
        id: buildPresetId(type, "size"),
        label: "分页间距",
        content: "#domId .el-pagination { gap: 6px; }",
      },
      {
        id: buildPresetId(type, "button"),
        label: "按钮背景",
        content: "#domId .btn-prev, #domId .btn-next { background: #f5f7fa; }",
      },
    ],
  },
  {
    name: "tag",
    types: ["Tag"],
    detail: (methodName, type) => [
      {
        id: buildPresetId(type, "basic"),
        label: "配置",
        content: buildDslTemplate(
          methodName,
          '  id: "tagStatus",\n  text: "标签",\n  type: "success",',
        ),
      },
      {
        id: buildPresetId(type, "closable"),
        label: "可关闭",
        content: buildDslTemplate(methodName, '  closable: true,\n  effect: "dark",'),
      },
      {
        id: buildPresetId(type, "full-dsl"),
        label: "完整 DSL 模板",
        content: buildDslTemplate(
          methodName,
          '  /** 基础配置 */\n  id: "tagStatus",\n  text: "标签",\n  type: "success",\n  effect: "light",\n\n  /** 基础配置 */\n  closable: true,\n\n  /** 样式 */\n  className: "custom-tag",\n\n  /** 事件 */\n  onClose: {\n    action: "setVar",\n    target: "$vars.tagClosed",\n  },',
        ),
      },
    ],
    style: (type) => [
      {
        id: buildPresetId(type, "bg"),
        label: "背景鑹",
        content: "#domId .el-tag { background: #ecf5ff; }",
      },
      {
        id: buildPresetId(type, "border"),
        label: "边框色",
        content: "#domId .el-tag { border-color: #b3d8ff; }",
      },
      {
        id: buildPresetId(type, "radius"),
        label: "圆角",
        content: "#domId .el-tag { border-radius: 6px; }",
      },
    ],
  },
  {
    name: "slider",
    types: ["Slider"],
    detail: (methodName, type) => [
      {
        id: buildPresetId(type, "basic"),
        label: "滑块配置",
        content: buildDslTemplate(
          methodName,
          '  id: "sliderValue",\n  value: 30,\n  min: 0,\n  max: 100,',
        ),
      },
      {
        id: buildPresetId(type, "range"),
        label: "范围滑块配置",
        content: buildDslTemplate(methodName, "  range: true,\n  showStops: true,\n  step: 10,"),
      },
      {
        id: buildPresetId(type, "full-dsl"),
        label: "完整 DSL 模板",
        content: buildDslTemplate(
          methodName,
          '  /** 基础配置 */\n  id: "sliderValue",\n  value: 30,\n  min: 0,\n  max: 100,\n  step: 1,\n\n  /** 属性 */\n  range: false,\n  showStops: false,\n\n  /** 样式 */\n  className: "custom-slider",\n\n  /** 事件 */\n  onChange: {\n    action: "setVar",\n    target: "$vars.sliderValue",\n  },',
        ),
      },
    ],
    style: (type) => [
      {
        id: buildPresetId(type, "bar"),
        label: "滑块鑹",
        content: "#domId .el-slider__bar { background: #409eff; }",
      },
      {
        id: buildPresetId(type, "button"),
        label: "滑块按钮",
        content: "#domId .el-slider__button { border-color: #409eff; }",
      },
      {
        id: buildPresetId(type, "rail"),
        label: "滑轨颜色",
        content: "#domId .el-slider__runway { background: #e4e7ed; }",
      },
    ],
  },
  {
    name: "card",
    types: ["Card"],
    detail: (methodName, type) => [
      {
        id: buildPresetId(type, "basic"),
        label: "卡片配置",
        content: buildDslTemplate(
          methodName,
          '  id: "cardInfo",\n  header: "卡片标题",\n  shadow: "hover",',
        ),
      },
      {
        id: buildPresetId(type, "simple"),
        label: "简洁卡鐗",
        content: buildDslTemplate(methodName, '  header: "概览",\n  shadow: "never",'),
      },
      {
        id: buildPresetId(type, "full-dsl"),
        label: "完整 DSL 模板",
        content: buildDslTemplate(
          methodName,
          '  /** 基础配置 */\n  id: "cardInfo",\n  header: "卡片标题",\n  shadow: "hover",\n\n  /** 样式 */\n  className: "custom-card",\n  style: {\n    padding: "12px",\n  },',
        ),
      },
    ],
    style: (type) => [
      {
        id: buildPresetId(type, "shadow"),
        label: "卡片阴影",
        content: "#domId .el-card { box-shadow: 0 4px 12px rgba(0,0,0,0.08); }",
      },
      {
        id: buildPresetId(type, "radius"),
        label: "卡片",
        content: "#domId .el-card { border-radius: 8px; }",
      },
      {
        id: buildPresetId(type, "border"),
        label: "卡片",
        content: "#domId .el-card { border-color: #ebeef5; }",
      },
    ],
  },
  {
    name: "transfer",
    types: ["Transfer"],
    detail: (methodName, type) => [
      {
        id: buildPresetId(type, "basic"),
        label: "穿梭配置",
        content: buildDslTemplate(
          methodName,
          '  id: "transferData",\n  data: [\n    { key: "A", label: "选项A" },\n    { key: "B", label: "选项B" },\n  ],\n  value: [],',
        ),
      },
      {
        id: buildPresetId(type, "filter"),
        label: "穿梭筛选配缃",
        content: buildDslTemplate(
          methodName,
          '  filterable: true,\n  filterPlaceholder: "请输入关键字",',
        ),
      },
      {
        id: buildPresetId(type, "full-dsl"),
        label: "完整 DSL 模板",
        content: buildDslTemplate(
          methodName,
          '  /** 基础配置 */\n  id: "transferData",\n  data: [\n    { key: "A", label: "选项A" },\n    { key: "B", label: "选项B" },\n  ],\n  value: [],\n\n  /** 属性 */\n  filterable: true,\n  filterPlaceholder: "请输入关键字",\n\n  /** 样式 */\n  className: "custom-transfer",\n\n  /** 事件 */\n  onChange: {\n    action: "setVar",\n    target: "$vars.transferValue",\n  },',
        ),
      },
    ],
    style: (type) => [
      {
        id: buildPresetId(type, "panel"),
        label: "穿梭面板背景",
        content: "#domId .el-transfer-panel { background: #fafafa; }",
      },
      {
        id: buildPresetId(type, "border"),
        label: "穿梭面板",
        content: "#domId .el-transfer-panel { border-color: #e4e7ed; }",
      },
      {
        id: buildPresetId(type, "title"),
        label: "字体",
        content: "#domId .el-transfer-panel__header { font-weight: 600; }",
      },
    ],
  },
  {
    name: "timeline",
    types: ["Timeline"],
    detail: (methodName, type) => [
      {
        id: buildPresetId(type, "basic"),
        label: "时间线基础配置",
        content: buildDslTemplate(
          methodName,
          '  id: "timeline",\n  items: [\n    { timestamp: "2024-01-01", content: "创建" },\n    { timestamp: "2024-01-02", content: "完成" },\n  ],',
        ),
      },
      {
        id: buildPresetId(type, "align"),
        label: "时间线布局",
        content: buildDslTemplate(methodName, '  placement: "top",\n  reverse: false,'),
      },
      {
        id: buildPresetId(type, "full-dsl"),
        label: "完整 DSL 模板",
        content: buildDslTemplate(
          methodName,
          '  /** 基础配置 */\n  id: "timeline",\n  items: [\n    { timestamp: "2024-01-01", content: "创建" },\n    { timestamp: "2024-01-02", content: "完成" },\n  ],\n\n  /** 属性 */\n  placement: "top",\n  reverse: false,\n\n  /** 样式 */\n  className: "custom-timeline",',
        ),
      },
    ],
    style: (type) => [
      {
        id: buildPresetId(type, "node"),
        label: "节点颜色",
        content: "#domId .el-timeline-item__node { background: #409eff; }",
      },
      {
        id: buildPresetId(type, "tail"),
        label: "连线颜色",
        content: "#domId .el-timeline-item__tail { border-color: #dcdfe6; }",
      },
      {
        id: buildPresetId(type, "title"),
        label: "字体",
        content: "#domId .el-timeline-item__content { font-size: 13px; }",
      },
    ],
  },
  {
    name: "steps",
    types: ["Steps"],
    detail: (methodName, type) => [
      {
        id: buildPresetId(type, "basic"),
        label: "步骤配置",
        content: buildDslTemplate(
          methodName,
          '  id: "steps",\n  active: 1,\n  items: [\n    { title: "开始" },\n    { title: "处理" },\n    { title: "完成" },\n  ],',
        ),
      },
      {
        id: buildPresetId(type, "status"),
        label: "步骤状态配置",
        content: buildDslTemplate(
          methodName,
          '  direction: "horizontal",\n  finishStatus: "success",',
        ),
      },
      {
        id: buildPresetId(type, "full-dsl"),
        label: "完整 DSL 模板",
        content: buildDslTemplate(
          methodName,
          '  /** 基础配置 */\n  id: "steps",\n  active: 1,\n  items: [\n    { title: "开始" },\n    { title: "处理" },\n    { title: "完成" },\n  ],\n\n  /** 属性 */\n  direction: "horizontal",\n  finishStatus: "success",\n\n  /** 样式 */\n  className: "custom-steps",',
        ),
      },
    ],
    style: (type) => [
      {
        id: buildPresetId(type, "active"),
        label: "步骤激活色",
        content: "#domId .is-process .el-step__icon { border-color: #409eff; }",
      },
      {
        id: buildPresetId(type, "finish"),
        label: "完成鑹",
        content: "#domId .is-finish .el-step__icon { border-color: #67c23a; }",
      },
      {
        id: buildPresetId(type, "title"),
        label: "字体",
        content: "#domId .el-step__title { font-size: 13px; }",
      },
    ],
  },
  {
    name: "carousel",
    types: ["ImageCarousel", "CarouselComponent"],
    detail: (methodName, type) => [
      {
        id: buildPresetId(type, "basic"),
        label: "轮播配置",
        content: buildDslTemplate(
          methodName,
          '  id: "carousel",\n  height: "240px",\n  items: [\n    { src: "https://", title: "图片1" },\n    { src: "https://", title: "图片2" },\n  ],',
        ),
      },
      {
        id: buildPresetId(type, "auto"),
        label: "轮播自动播放",
        content: buildDslTemplate(
          methodName,
          '  autoplay: true,\n  interval: 3000,\n  arrow: "hover",',
        ),
      },
      {
        id: buildPresetId(type, "full-dsl"),
        label: "完整 DSL 模板",
        content: buildDslTemplate(
          methodName,
          '  /** 基础配置 */\n  id: "carousel",\n  height: "240px",\n  items: [\n    { src: "https://", title: "图片1" },\n    { src: "https://", title: "图片2" },\n  ],\n\n  /** 属性 */\n  autoplay: true,\n  interval: 3000,\n  arrow: "hover",\n\n  /** 样式 */\n  className: "custom-carousel",',
        ),
      },
    ],
    style: (type) => [
      {
        id: buildPresetId(type, "dot"),
        label: "指示点颜色",
        content: "#domId .el-carousel__indicator.is-active button { background: #409eff; }",
      },
      {
        id: buildPresetId(type, "arrow"),
        label: "颜色",
        content: "#domId .el-carousel__arrow { background: rgba(0,0,0,0.4); }",
      },
      {
        id: buildPresetId(type, "radius"),
        label: "轮播",
        content: "#domId { border-radius: 8px; overflow: hidden; }",
      },
    ],
  },
  {
    name: "calendar",
    types: ["Calendar"],
    detail: (methodName, type) => [
      {
        id: buildPresetId(type, "basic"),
        label: "日历配置",
        content: buildDslTemplate(methodName, '  id: "calendar",\n  value: new Date(),'),
      },
      {
        id: buildPresetId(type, "range"),
        label: "日历范围",
        content: buildDslTemplate(methodName, "  range: [new Date(), new Date()],"),
      },
      {
        id: buildPresetId(type, "full-dsl"),
        label: "完整 DSL 模板",
        content: buildDslTemplate(
          methodName,
          '  /** 基础配置 */\n  id: "calendar",\n  value: new Date(),\n\n  /** 属性 */\n  range: null,\n\n  /** 样式 */\n  className: "custom-calendar",',
        ),
      },
    ],
    style: (type) => [
      {
        id: buildPresetId(type, "cell"),
        label: "日历格子",
        content: "#domId .el-calendar-day { padding: 8px; }",
      },
      {
        id: buildPresetId(type, "today"),
        label: "今天高亮",
        content: "#domId .is-today { background: #ecf5ff; }",
      },
      {
        id: buildPresetId(type, "header"),
        label: "头部字体",
        content: "#domId .el-calendar__title { font-weight: 600; }",
      },
    ],
  },
  {
    name: "layout",
    types: [
      "ElContainer",
      "ElHeader",
      "ElAside",
      "ElMain",
      "ElFooter",
      "ElLayout",
      "ElLayoutRow",
      "ElCol",
    ],
    detail: (methodName, type) => [
      {
        id: buildPresetId(type, "basic"),
        label: "布局配置",
        content: buildDslTemplate(
          methodName,
          '  id: "layout",\n  direction: "horizontal",\n  gutter: 12,',
        ),
      },
      {
        id: buildPresetId(type, "size"),
        label: "布局配置",
        content: buildDslTemplate(
          methodName,
          '  span: 12,\n  offset: 0,\n  height: "100%",\n  width: "100%",',
        ),
      },
      {
        id: buildPresetId(type, "full-dsl"),
        label: "完整 DSL 模板",
        content: buildDslTemplate(
          methodName,
          '  /** 基础配置 */\n  id: "layout",\n  direction: "horizontal",\n  gutter: 12,\n\n  /** 基础配置 */\n  span: 12,\n  offset: 0,\n  height: "100%",\n  width: "100%",\n\n  /** 样式 */\n  className: "custom-layout",',
        ),
      },
    ],
    style: (type) => [
      {
        id: buildPresetId(type, "padding"),
        label: "容器内边距",
        content: "#domId { padding: 12px; }",
      },
      {
        id: buildPresetId(type, "bg"),
        label: "容器背景",
        content: "#domId { background: #f5f7fa; }",
      },
      {
        id: buildPresetId(type, "border"),
        label: "容器",
        content: "#domId { border: 1px solid #e4e7ed; }",
      },
    ],
  },
];

/**
 * 获取 Element Plus 详细配置预设
 * @param {string} type - 组件类型
 * @returns {Array<{ id: string, label: string, content: string }>}
 */
function resolveElementPlusDetailPresets(type: string): AnyArray {
  const group = elementPlusPresetGroups.find((item: AnyRecord) => item.types.includes(type));
  const methodName = getDslMethodName(type);
  if (group?.detail) {
    return withCommonDetailPresets(type, group.detail(methodName, type))
      .map((preset: AnyRecord) => normalizeDslPreset(type, preset))
      .map((preset: AnyRecord) => localizePropertyPanelPreset(preset))
      .map((preset: AnyRecord) => ({
        ...preset,
        tags: preset.tags || [type, "detail", group.name],
        keywords: preset.keywords || [preset.label, type, group.name].filter(Boolean).join(" "),
      }));
  }
  return [
    localizePropertyPanelPreset(
      normalizeDslPreset(type, {
        id: buildPresetId(type, "basic"),
        label: "DSL 模板",
        content: buildDslTemplate(methodName, '  id: "component",\n  props: {},'),
      }),
    ),
  ];
}

/**
 * 获取 Element Plus 样式配置预设
 * @param {string} type - 组件类型
 * @returns {Array<{ id: string, label: string, content: string }>}
 */
function resolveElementPlusStylePresets(type: string): AnyArray {
  const customPresets = customStylePresetMap[type] as AnyArray | undefined;
  if (customPresets) {
    return withCommonStylePresets(customPresets).map((preset: AnyRecord) => ({
      ...localizePropertyPanelPreset(preset),
      tags: preset.tags || [type, "style", "custom"],
      keywords: preset.keywords || [preset.label, type, "custom"].filter(Boolean).join(" "),
    }));
  }
  const group = elementPlusPresetGroups.find((item: AnyRecord) => item.types.includes(type));
  if (group?.style) {
    return withCommonStylePresets(group.style(type)).map((preset: AnyRecord) => ({
      ...localizePropertyPanelPreset(preset),
      tags: preset.tags || [type, "style", group.name],
      keywords: preset.keywords || [preset.label, type, group.name].filter(Boolean).join(" "),
    }));
  }
  return stylePresets.map((preset: AnyRecord) => localizePropertyPanelPreset(preset));
}

/**
 * 获取区域默认尺寸
 * @param {string} propName - 属性名
 * @returns {string | undefined}
 */
/**
 * 判断区域是否启用
 * @param {{ toggleProp: string }} item - 区域配置项
 * @returns {boolean}
 */
function isRegionEnabled(item: AnyRecord) {
  if (!item?.toggleProp) return true;
  return Boolean(getPropValue(item.toggleProp));
}

/**
 * 获取当前容器最大尺寸
 * @returns {{ width?: number, height?: number }}
 */
function resolveContainerMaxSize(): AnyRecord {
  const element = currentElement.value;
  const liveNode = element?.id ? doc.value?.getNode?.(element.id) || element : element;
  let containerRect;
  if (typeof document !== "undefined" && element?.id) {
    const containerEl = document.querySelector(`[data-node-id="${element.id}"]`);
    if (containerEl) {
      const rect = containerEl.getBoundingClientRect();
      containerRect = { width: rect.width, height: rect.height };
    }
  }
  return computeContainerMaxSize(element, liveNode, containerRect);
}

/**
 * 获取区域尺寸上限文案
 * @param {string | null} propName - 属性名
 * @returns {string}
 */
function getRegionMaxLabel(propName: string | null): string {
  return formatRegionMaxLabel(propName, currentElement.value, resolveContainerMaxSize());
}

/**
 * 限制区域尺寸输入值
 * @param {string} propName - 属性名
 * @param {string} value - 数值
 * @param {string} unit - 单位
 * @returns {{ value: string, unit: string }}
 */
function clampRegionSize(propName: string, value: string, unit: string) {
  return clampRegionSizeValue(
    propName,
    value,
    unit,
    currentElement.value,
    resolveContainerMaxSize(),
  );
}

/**
 * ??
 * @param {Record<string, any> | undefined} props - ?
 */
function syncRegionSizeState(props: AnyRecord | undefined) {
  regionSizeState.value = buildRegionSizeState(props);
}

watch(
  currentElement,
  (el) => {
    elementLabel.value = el?.label || "";
    elementDescription.value = el?.description || "";
  },
  { immediate: true },
);

watch(
  () => currentElement.value?.id,
  () => {
    syncRegionSizeState(currentElement.value?.props);
  },
  { immediate: true },
);

/**
 * 直接应用节点补丁（兜底处理编辑器状态异常）
 * @param {string} nodeId - 节点 ID
 * @param {Record<string, any>} patch - 更新补丁
 * @returns {boolean}
 */
function applyNodePatch(nodeId: string, patch: AnyRecord) {
  if (!doc.value || !nodeId || !patch) return false;
  if (typeof doc.value._updateNode !== "function") return false;
  doc.value._updateNode(nodeId, patch);
  docVersion.value += 1;
  return true;
}

/**
 * 获取组件 Manifest
 */
const manifest = computed<AnyRecord | null>(() => {
  void locale.value;
  const type = elementTypeName(currentElement.value?.type);
  if (!type) return null;
  return getManifest(type) ?? null;
});

const effectiveManifest = computed<AnyRecord | null>(() => {
  const type = elementTypeName(currentElement.value?.type);
  const elementPlusProps = isElementPlusType(type) ? buildElementPlusPropDefs(type) : [];
  if (manifest.value) {
    const baseProps = Array.isArray(manifest.value.props) ? manifest.value.props : [];
    if (elementPlusProps.length === 0) {
      return { ...manifest.value, props: baseProps };
    }
    const existing = new Set(baseProps.map((item) => item.name));
    const mergedProps = [
      ...baseProps,
      ...elementPlusProps.filter((item) => !existing.has(item.name)),
    ];
    return { ...manifest.value, props: mergedProps };
  }
  if (elementPlusProps.length > 0) {
    return {
      type: type || "ElementPlus",
      name: type || "ElementPlus",
      category: localizePropertyPanelLiteral("组件"),
      props: elementPlusProps,
    };
  }
  return null;
});

const layoutForceProps = computed<AnyArray>(() => []);

const configDialogTitle = computed<string>(() =>
  configDialogType.value === "detail"
    ? t("propertyPanel.configDialog.detailTitle")
    : t("propertyPanel.configDialog.styleTitle"),
);
const configEditorLanguage = computed<string>(() =>
  configDialogType.value === "detail" ? "javascript" : "css",
);
const currentStyleConfigContent = computed<string>(() =>
  String(currentElement.value?.styleConfig || ""),
);
const presetLabel = computed<string>(() =>
  configDialogType.value === "detail"
    ? t("propertyPanel.configDialog.detailPreset")
    : t("propertyPanel.configDialog.stylePreset"),
);
const currentPresetOptions = computed<AnyArray>(() => {
  const type = currentElement.value?.type;
  if (configDialogType.value === "detail") {
    if (type === "EChart")
      return echartDetailPresets.map((preset: AnyRecord) => localizePropertyPanelPreset(preset));
    if (type === "Button") {
      return withCommonDetailPresets(type, buttonDetailPresets)
        .map((preset: AnyRecord) => normalizeDslPreset(type, preset))
        .map((preset: AnyRecord) => localizePropertyPanelPreset(preset))
        .map((preset: AnyRecord) => ({
          ...preset,
          tags: preset.tags || [type, "detail", "button"],
          keywords: preset.keywords || [preset.label, type, "button"].filter(Boolean).join(" "),
        }));
    }
    if (type === "Text") {
      return withCommonDetailPresets(type, textDetailPresets)
        .map((preset: AnyRecord) => normalizeDslPreset(type, preset))
        .map((preset: AnyRecord) => localizePropertyPanelPreset(preset))
        .map((preset: AnyRecord) => ({
          ...preset,
          tags: preset.tags || [type, "detail", "text"],
          keywords: preset.keywords || [preset.label, type, "text"].filter(Boolean).join(" "),
        }));
    }
    if (customDetailPresetMap[type]) {
      const sourcePresets =
        type === "FormLayout"
          ? (customDetailPresetMap[type] as AnyArray)
          : withCommonDetailPresets(type, customDetailPresetMap[type] as AnyArray);
      return sourcePresets
        .map((preset: AnyRecord) => normalizeDslPreset(type, preset))
        .map((preset: AnyRecord) => localizePropertyPanelPreset(preset))
        .map((preset: AnyRecord) => ({
          ...preset,
          tags: preset.tags || [type, "detail", "custom"],
          keywords: preset.keywords || [preset.label, type, "custom"].filter(Boolean).join(" "),
        }));
    }
    if (isElementPlusType(type)) return resolveElementPlusDetailPresets(type);
    return withCommonDetailPresets(type, detailPresets).map((preset: AnyRecord) =>
      localizePropertyPanelPreset(preset),
    );
  }
  if (type === "Button") {
    return withCommonStylePresets(buttonStylePresets.map(withButtonRootSelectorSupport)).map((preset: AnyRecord) => ({
      ...localizePropertyPanelPreset(preset),
      tags: preset.tags || [type, "style", "button"],
      keywords: preset.keywords || [preset.label, type, "button"].filter(Boolean).join(" "),
    }));
  }
  if (type === "Text") {
    return withCommonStylePresets(textStylePresets).map((preset: AnyRecord) => ({
      ...localizePropertyPanelPreset(preset),
      tags: preset.tags || [type, "style", "text"],
      keywords: preset.keywords || [preset.label, type, "text"].filter(Boolean).join(" "),
    }));
  }
  if (customStylePresetMap[type]) {
    return withCommonStylePresets(customStylePresetMap[type] as AnyArray).map((preset: AnyRecord) => ({
      ...localizePropertyPanelPreset(preset),
      tags: preset.tags || [type, "style", "custom"],
      keywords: preset.keywords || [preset.label, type, "custom"].filter(Boolean).join(" "),
    }));
  }
  if (isElementPlusType(type)) return resolveElementPlusStylePresets(type);
  return stylePresets.map((preset: AnyRecord) => localizePropertyPanelPreset(preset));
});

const filteredPresetOptions = computed<AnyArray>(() => {
  const keyword = String(presetSearch.value || "")
    .trim()
    .toLowerCase();
  if (!keyword) return currentPresetOptions.value;
  return currentPresetOptions.value.filter((item: AnyRecord) => {
    const haystack = `${item.label || ""} ${item.tags?.join(" ") || ""} ${item.keywords || ""}`;
    return haystack.toLowerCase().includes(keyword);
  });
});

const bindingDialogTitle = computed<string>(() => {
  const label = bindingProp.value?.label || bindingProp.value?.name;
  return label
    ? t("propertyPanel.bindingDialog.titleWithLabel", { label })
    : t("propertyPanel.bindingDialog.title");
});

const bindingDialogDescription = computed<string>(() => {
  const elementName =
    elementLabel.value || elementType.value || t("propertyPanel.bindingDialog.targetFallback");
  const propLabel = bindingProp.value?.label || bindingProp.value?.name || "";
  return propLabel
    ? t("propertyPanel.bindingDialog.descriptionWithProp", { elementName, propLabel })
    : t("propertyPanel.bindingDialog.description", { elementName });
});
const runtimePermissionSchemeOptions = computed<AnyArray>(() => {
  const schemes = currentPage.value?.config?.runtimeAccess?.schemes;
  if (!Array.isArray(schemes)) return [];
  return schemes.map((scheme: AnyRecord) => ({
    label: String(scheme.name || scheme.id),
    value: String(scheme.id),
  }));
});

const selectedNodeRuntimeAccess = computed<AnyRecord>(() => {
  const access = selectedNode.value?.permissions?.runtimeAccess;
  return access && typeof access === "object" ? access : {};
});

const visibleSchemeId = computed<string>({
  get: () => String(selectedNodeRuntimeAccess.value.visibleSchemeId || ""),
  set: (value) => updateNodeRuntimeAccess("visibleSchemeId", value),
});

const operableSchemeId = computed<string>({
  get: () => String(selectedNodeRuntimeAccess.value.operableSchemeId || ""),
  set: (value) => updateNodeRuntimeAccess("operableSchemeId", value),
});

const isElContainer = computed<boolean>(() => elementType.value === "ElContainer");
const isEChart = computed<boolean>(() => elementType.value === "EChart");
const isCollapseNode = computed<boolean>(() => elementType.value === "Collapse" && !!selectedNode.value);
const isTabsNode = computed<boolean>(() => elementType.value === "Tabs" && !!selectedNode.value);
const collapseChildNodes = computed<AnyArray>(() => {
  const node = selectedNode.value;
  if (!node?.id || !doc.value?.getChildren) return [];
  return doc.value.getChildren(node.id).map((child: AnyRecord) => ({
    id: child.id,
    props: { ...(child.props || {}) },
  }));
});
const tabsChildNodes = computed<AnyArray>(() => {
  const node = selectedNode.value;
  if (!node?.id || !doc.value?.getChildren) return [];
  return doc.value.getChildren(node.id).map((child: AnyRecord) => ({
    id: child.id,
    props: { ...(child.props || {}) },
  }));
});
const regionPropRows = computed<AnyArray>(() => [
  {
    key: "header",
    label: "顶部区域",
    toggleProp: "showHeader",
    sizeProp: "headerHeight",
    sizeLabel: "高度",
  },
  {
    key: "aside",
    label: "左侧区域",
    toggleProp: "showAside",
    sizeProp: "asideWidth",
    sizeLabel: "宽度",
  },
  {
    key: "main",
    label: "主内容区",
    toggleProp: "showMain",
    sizeProp: null,
    sizeLabel: "",
  },
  {
    key: "footer",
    label: "底部区域",
    toggleProp: "showFooter",
    sizeProp: "footerHeight",
    sizeLabel: "高度",
  },
]);

const visibleRegionSizeRows = computed<AnyArray>(() =>
  regionPropRows.value.filter((item) => item.sizeProp && isRegionEnabled(item)),
);

const activeRegionPreset = computed<string>(() =>
  String(getPropValue("regionPreset") || "aside-between"),
);

function regionPresetGridAreas(option: AnyRecord): string {
  return (option.grid || []).map((row: string) => `"${row}"`).join(" ");
}

/**
 * 判断是否为区域分组
 * @param {{ props?: Array<{ name: string }> }} group - 分组
 * @returns {boolean}
 */
function isRegionGroup(group: AnyRecord) {
  return Boolean(group?.props?.some((prop: AnyRecord) => prop.name === "showHeader"));
}

/**
 * 按 group 对属性分组
 */
const groupedProps = computed<AnyArray>(() => {
  if (!effectiveManifest.value) return [];

  const groups = new Map<string, AnyRecord>();
  for (const prop of effectiveManifest.value.props as AnyRecord[]) {
    const propDef: AnyRecord = prop;
    const groupName = propDef.group || "?";
    if (!groups.has(groupName)) {
      groups.set(groupName, { name: groupName, props: [] });
    }
    groups.get(groupName)!.props.push(propDef);
  }
  return Array.from(groups.values());
});

const displayPropGroups = computed<AnyArray>(() => {
  const groups = groupedProps.value
    .map((group) => ({ ...group, props: getVisibleGroupProps(group) }))
    .filter((group) => group.props.length > 0);
  if (groups.length > 0) return groups;
  if (effectiveManifest.value?.props?.length) {
    const visibleProps = getVisibleGroupProps({
      name: localizePropertyPanelLiteral("属性"),
      props: effectiveManifest.value.props,
    });
    return visibleProps.length
      ? [{ name: localizePropertyPanelLiteral("属性"), props: visibleProps }]
      : [];
  }
  return [];
});

const visibleLayoutForceProps = computed<AnyArray>(() =>
  layoutForceProps.value.filter((propDef) => !shouldHideProp(propDef)),
);

const sectionExpanded = ref<AnyRecord>({});

const getGroupSectionKey = (group: AnyRecord) => `group:${String(group?.name || "属性")}`;

function resolveGroupTitle(group: AnyRecord) {
  const name = String(group?.name || "").trim();
  return !name || name === "?"
    ? localizePropertyPanelLiteral("属性")
    : localizePropertyPanelLiteral(name);
}

function getVisibleGroupProps(group: AnyRecord): AnyArray {
  return (group?.props || []).filter(
    (propDef: AnyRecord) =>
      propDef && (!isEChart.value || propDef.name !== "option") && !shouldHideProp(propDef),
  );
}

const isSectionExpanded = (key: string) => sectionExpanded.value[key] !== false;

function toggleSection(key: string) {
  sectionExpanded.value = {
    ...sectionExpanded.value,
    [key]: !isSectionExpanded(key),
  };
}

watch(
  displayPropGroups,
  (groups: AnyArray) => {
    const nextState = { ...sectionExpanded.value };
    groups.forEach((group: AnyRecord) => {
      const key = getGroupSectionKey(group);
      if (!(key in nextState)) {
        nextState[key] = true;
      }
    });
    sectionExpanded.value = nextState;
  },
  { immediate: true },
);

const pageVars = computed<AnyRecord>(() => {
  void docVersion.value;
  const pageId = currentPageId.value;
  if (!pageId || !doc.value) return {};
  const vars = doc.value.vars?.pages?.[pageId];
  return vars && typeof vars === "object" ? vars : {};
});

/**
 * 收集组件名称
 * @param {string} nodeId - 节点 ID
 * @param {Set<string>} nameSet - 名称集合
 */
function collectComponentNames(nodeId: string, nameSet: Set<string>) {
  if (!doc.value || !nodeId) return;
  const node = doc.value.getNode?.(nodeId);
  if (!node) return;
  const label = node.label || node.type;
  if (label) nameSet.add(label);
  (node.children || []).forEach((childId: string) => collectComponentNames(childId, nameSet));
}

const pageComponentNames = computed<string[]>(() => {
  void docVersion.value;
  const rootId = currentPage.value?.rootNodeId;
  if (!rootId || !doc.value) return [];
  const nameSet = new Set<string>();
  collectComponentNames(rootId, nameSet);
  return Array.from(nameSet);
});

const bindingProjectGroupTree = computed<AnyArray>(() => [
  {
    id: "all",
    label: t("propertyPanel.bindingVarEnum.all"),
    type: "group",
    children: buildBindingGroupTree(projectVariableGroups.value || []),
  },
]);

const bindingProjectVariableRows = computed<AnyArray>(() => {
  const keyword = String(bindingProjectVarSearch.value || "").toLowerCase();
  const items = (Object.entries(projectVariables.value || {}) as Array<[string, AnyRecord]>).map(
    ([name, detail]) => ({
      name,
      groupId: detail?.groupId || null,
      type: detail?.type || "string",
      description: detail?.description || "",
      mapped: detail?.source?.type === "dataCenter" || detail?.mapped === true,
      sourceLabel: resolveProjectVariableSourceLabel(detail)
        ? `来自数据点：${resolveProjectVariableSourceLabel(detail)}`
        : "",
    }),
  );
  return items
    .filter((item) => {
      if (bindingEnumSelectedProjectGroupId.value) {
        return item.groupId === bindingEnumSelectedProjectGroupId.value;
      }
      return true;
    })
    .filter((item) => {
      if (!keyword) return true;
      return String(item.name || "")
        .toLowerCase()
        .includes(keyword);
    });
});

const pageVariableRows = computed<AnyArray>(() =>
  (Object.entries(pageVars.value || {}) as Array<[string, AnyRecord]>).map(([name, detail]) => ({
    name,
    type: detail?.type || "string",
    description: detail?.description || "",
  })),
);

function filterPageVariableRows(keywordValue: string): AnyArray {
  const keyword = String(keywordValue || "").toLowerCase();
  return pageVariableRows.value.filter((item) => {
    if (!keyword) return true;
    return String(item.name || "")
      .toLowerCase()
      .includes(keyword);
  });
}

const bindingPageVariableRows = computed<AnyArray>(() =>
  filterPageVariableRows(bindingPageContextSearch.value),
);

const configPageVariableRows = computed<AnyArray>(() =>
  filterPageVariableRows(configPageContextSearch.value),
);

const bindingPageComponentTree = computed<AnyArray>(() => {
  void docVersion.value;
  const rootId = currentPage.value?.rootNodeId;
  if (!rootId || !doc.value) return [];
  const buildNode = (nodeId: string): AnyRecord | null => {
    const node = doc.value.getNode(nodeId);
    if (!node) return null;
    const children = Array.isArray(node.children)
      ? node.children.map((childId: string) => buildNode(childId)).filter(Boolean)
      : [];
    const label = node.label || node.type || localizePropertyPanelLiteral("组件");
    return {
      id: node.id,
      label,
      type: children.length ? "group" : "component",
      componentName: node.label || "",
      children,
    };
  };
  const root = buildNode(rootId);
  if (!root) return [];
  return root.children?.length ? root.children : [root];
});

const bindingCustomScriptTree = computed<AnyArray>(() => {
  const rawGroups = globalScripts.value?.custom?.groups;
  const groups = Array.isArray(rawGroups) ? (rawGroups as AnyArray) : [];
  const rawItems = globalScripts.value?.custom?.items;
  const items = Array.isArray(rawItems) ? (rawItems as AnyArray) : [];
  const groupMap = new Map<string, AnyRecord>();
  const roots: AnyArray = [];

  groups.forEach((group: AnyRecord) => {
    groupMap.set(group.id, {
      id: group.id,
      label: group.name,
      type: "group",
      children: [],
    });
  });

  groupMap.forEach((node, id) => {
    const group = groups.find((item: AnyRecord) => item.id === id);
    if (group?.parentId && groupMap.has(group.parentId)) {
      groupMap.get(group.parentId)!.children.push(node);
    } else {
      roots.push(node);
    }
  });

  items.forEach((item: AnyRecord) => {
    if (!item?.id) return;
    const node = {
      id: item.id,
      label: item.name || localizePropertyPanelLiteral("未命名"),
      type: "item",
      params: item.params || item.args || "",
    };
    if (item.groupId && groupMap.has(item.groupId)) {
      groupMap.get(item.groupId)!.children.push(node);
    } else {
      roots.push(node);
    }
  });

  return roots;
});

const bindingCompletions = computed<AnyArray>(() => {
  const items: AnyArray = [
    {
      label: "console.log",
      insertText: "console.log()",
      kind: "Function",
      detail: "Log output",
    },
    {
      label: "if",
      insertText: "if () {\\n  \\n}",
      kind: "Snippet",
      detail: "if statement",
    },
    {
      label: "for",
      insertText: "for (let i = 0; i < ; i++) {\\n  \\n}",
      kind: "Snippet",
    },
    {
      label: "function",
      insertText: "function name() {\\n  \\n}",
      kind: "Snippet",
    },
    { label: "const", insertText: "const ", kind: "Keyword" },
    { label: "let", insertText: "let ", kind: "Keyword" },
    { label: "return", insertText: "return ", kind: "Keyword" },
  ];

  Object.keys(projectVariables.value || {}).forEach((name) => {
    items.push({
      label: name,
      insertText: name,
      kind: "Variable",
      detail: localizePropertyPanelLiteral("工程变量"),
      prefix: "$global.",
    });
  });

  (Array.isArray(globalScripts.value?.custom?.items)
    ? globalScripts.value.custom.items
    : []
  ).forEach((script: AnyRecord) => {
    if (!script?.name) return;
    const params =
      typeof script.params === "string" && script.params.trim()
        ? script.params.trim()
        : typeof script.args === "string"
          ? script.args.trim()
          : "";
    const call = params ? `${script.name}(${params})` : `${script.name}()`;
    items.push({
      label: script.name,
      insertText: call,
      kind: "Function",
      detail: localizePropertyPanelLiteral("自定义脚本"),
      prefix: "customScripts.",
    });
  });

  Object.keys(pageVars.value || {}).forEach((name) => {
    items.push({
      label: name,
      insertText: name,
      kind: "Variable",
      detail: localizePropertyPanelLiteral("页面变量"),
      prefix: "$vars.",
    });
  });

  return items;
});

const detailCompletions = computed<AnyArray>(() => {
  const items: AnyArray = [
    {
      label: "console.log",
      insertText: "console.log()",
      kind: "Function",
      detail: "Log output",
    },
    {
      label: "if",
      insertText: "if () {\n  \n}",
      kind: "Snippet",
      detail: "if statement",
    },
    {
      label: "for",
      insertText: "for (let i = 0; i < ; i++) {\n  \n}",
      kind: "Snippet",
      detail: "for loop",
    },
    {
      label: "function",
      insertText: "function name() {\n  \n}",
      kind: "Snippet",
      detail: "function",
    },
    { label: "const", insertText: "const ", kind: "Keyword" },
    { label: "let", insertText: "let ", kind: "Keyword" },
    { label: "return", insertText: "return ", kind: "Keyword" },
  ];

  Object.keys(projectVariables.value || {}).forEach((name) => {
    items.push({
      label: name,
      insertText: name,
      kind: "Variable",
      detail: localizePropertyPanelLiteral("工程变量"),
      prefix: "$global.",
    });
  });

  (Array.isArray(globalScripts.value?.custom?.items)
    ? globalScripts.value.custom.items
    : []
  ).forEach((scriptItem: AnyRecord) => {
    if (!scriptItem?.name) return;
    const params =
      typeof scriptItem.params === "string" && scriptItem.params.trim()
        ? scriptItem.params.trim()
        : typeof scriptItem.args === "string"
          ? scriptItem.args.trim()
          : "";
    const call = params ? `${scriptItem.name}(${params})` : `${scriptItem.name}()`;
    items.push({
      label: scriptItem.name,
      insertText: call,
      kind: "Function",
      detail: localizePropertyPanelLiteral("自定义脚本"),
      prefix: "customScripts.",
    });
  });

  Object.keys(pageVars.value || {}).forEach((name) => {
    items.push({
      label: name,
      insertText: name,
      kind: "Variable",
      detail: localizePropertyPanelLiteral("页面变量"),
      prefix: "$vars.",
    });
  });

  pageComponentNames.value.forEach((name) => {
    const chartMethodCompletions = [
      {
        label: "setOption",
        insertText: [
          "setOption(",
          SNIPPET_OPTION_PLACEHOLDER,
          ", ",
          SNIPPET_BOOL_PLACEHOLDER,
          ")",
        ].join(""),
        kind: "Method",
        detail: localizePropertyPanelLiteral("图表方法"),
        prefix: ".",
      },
      {
        label: "echarts",
        insertText: [
          "echarts(",
          SNIPPET_METHOD_PLACEHOLDER,
          ", ",
          SNIPPET_OPTION_ARG_PLACEHOLDER,
          ")",
        ].join(""),
        kind: "Method",
        detail: localizePropertyPanelLiteral("图表方法"),
        prefix: ".",
      },
      {
        label: "getInstance",
        insertText: "getInstance()",
        kind: "Method",
        detail: localizePropertyPanelLiteral("图表方法"),
        prefix: ".",
      },
    ];
    chartMethodCompletions.forEach((item) => items.push(item));
    items.push({
      label: name,
      insertText: name,
      kind: "Variable",
      detail: localizePropertyPanelLiteral("页面组件"),
      prefix: "components.",
    });
  });

  return items;
});

const configEditorCompletions = computed<AnyArray>(() => {
  return configDialogType.value === "detail" ? detailCompletions.value : [];
});

/**
 * 构建绑定面板分组树
 * @param {Array<{ id: string, name: string, parentId?: string }>} groups - 分组数据
 * @returns {Array<{ id: string, label: string, type: string, children: Array }>}
 */
function buildBindingGroupTree(groups: AnyArray) {
  const groupMap = new Map<string, AnyRecord>();
  const roots: AnyArray = [];
  const normalized = Array.isArray(groups) ? groups : [];
  normalized.forEach((group) => {
    groupMap.set(group.id, {
      id: group.id,
      label: group.name,
      type: "group",
      children: [],
    });
  });
  groupMap.forEach((node, id) => {
    const group = normalized.find((item) => item.id === id);
    if (group?.parentId && groupMap.has(group.parentId)) {
      groupMap.get(group.parentId)!.children.push(node);
    } else {
      roots.push(node);
    }
  });
  return roots;
}

/**
 * 过滤绑定侧边栏节点
 * @param {string} value - 关键词
 * @param {{ label?: string }} data - 节点数据
 * @returns {boolean}
 */
function filterBindingSidebarNode(value: string, data: AnyRecord) {
  if (!value) return true;
  return String(data?.label || "")
    .toLowerCase()
    .includes(value.toLowerCase());
}

watch(bindingScriptSearch, (value) => {
  bindingCustomTreeRef.value?.filter?.(value);
});

watch(bindingComponentSearch, (value) => {
  bindingComponentTreeRef.value?.filter?.(value);
});

watch(configScriptSearch, (value) => {
  configCustomTreeRef.value?.filter?.(value);
});

watch(configComponentSearch, (value) => {
  configComponentTreeRef.value?.filter?.(value);
});

/**
 * 规范化样式配置输出
 * @param {string} content - 样式内容
 * @returns {string}
 */
function formatStyleConfigOutput(content: string): string {
  const text = String(content || "");
  if (configDialogType.value !== "style") return text;
  if (text.includes("{")) return text;
  const trimmed = text.trim();
  if (!trimmed) return "";
  return `#domId {
${trimmed}
}`;
}

/**
 * 自动为选择器补全 `#domId` 作用域
 * @param {string} content - 样式内容
 * @returns {string}
 */
function prefixStyleConfigScope(content: string): string {
  const text = String(content || "").trim();
  if (!text || !text.includes("{")) return text;
  const blocks = text.split("}");
  const rebuilt = blocks
    .map((block) => {
      const [selector = "", body] = block.split("{");
      if (!body) return "";
      const trimmedSelector = selector.trim();
      if (!trimmedSelector) return "";
      if (trimmedSelector.startsWith("@")) {
        return `${trimmedSelector} {${body}`;
      }
      const selectors = trimmedSelector.split(",").map((item) => {
        const sel = item.trim();
        if (!sel) return "";
        if (sel.includes("#domId")) return sel;
        if (sel.startsWith(":")) return `#domId${sel}`;
        return `#domId ${sel}`;
      });
      return `${selectors.filter(Boolean).join(", ")} {${body}`;
    })
    .filter(Boolean)
    .join("}\n");
  return rebuilt ? `${rebuilt}}` : text;
}

/**
 * 轻量校验 DSL 脚本
 * @param {string} content - DSL 内容
 * @returns {{ valid: boolean, message?: string }}
 */
function validateDetailConfig(content: string): { valid: boolean; message?: string } {
  const text = String(content || "").trim();
  if (!text) return { valid: true };
  try {
    // 仅做语法检查，不执行
    // eslint-disable-next-line no-new-func
    const validator = new Function(text);
    void validator;
    return { valid: true };
  } catch (error: any) {
    return {
      valid: false,
      message: error?.message || t("propertyPanel.configDialog.dslSyntaxError"),
    };
  }
}

/**
 * 变量引用结构化
 * @param {string} content - DSL 内容
 * @returns {string}
 */
function normalizeDetailConfigBindings(content: string): string {
  let text = String(content || "");
  text = text.replace(DETAIL_VAR_BINDING_RE, ': { $var: "$1" }');
  text = text.replace(DETAIL_GLOBAL_BINDING_RE, ': { $var: "$1" }');
  text = text.replace(DETAIL_VAR_STRING_BINDING_RE, ': { $var: "$1" }');
  text = text.replace(DETAIL_GLOBAL_STRING_BINDING_RE, ': { $var: "$1" }');
  return text;
}

const textStylePropNames = new Set([
  "fontSize",
  "fontWeight",
  "fontFamily",
  "color",
  "textAlign",
  "lineHeight",
  "letterSpacing",
  "wordBreak",
]);
const textStyleNumberProps = new Set(["fontSize", "lineHeight", "letterSpacing"]);

/**
 * 读取文本样式值（去掉 px 转成数字）
 * @param {string} propName - 属性名
 * @param {any} value - 原始值
 * @returns {any}
 */
function resolveTextStyleValue(propName: string, value: any): any {
  if (value === null || value === undefined) return value;
  if (textStyleNumberProps.has(propName)) {
    if (typeof value === "number") return value;
    const text = String(value).trim();
    if (!text) return undefined;
    const match = text.match(TEXT_NUMBER_PREFIX_RE);
    if (match) {
      const num = Number.parseFloat(match[1] || "0");
      return Number.isFinite(num) ? num : value;
    }
  }
  return value;
}

/**
 * 读取属性值
 * @param {string} propName - 属性名
 */
function getPropValue(propName: string): any {
  const el = currentElement.value;
  if (!el) return undefined;
  const propValue = el.props ? el.props[propName] : undefined;
  if (propValue !== undefined) return propValue;
  if (elementTypeName(el.type) === "Text" && textStylePropNames.has(propName)) {
    return resolveTextStyleValue(propName, el.style?.[propName]);
  }
  return propValue;
}

/**
 * 获取区域尺寸状态
 * @param {string} propName - 属性名
 * @returns {{ value: string, unit: string } | undefined}
 */
const getRegionSizeState = (propName: string) => regionSizeState.value[propName];

/**
 * 设置区域尺寸状态
 * @param {string} propName - 属性名
 * @param {string} value - 值
 * @param {string} unit - 单位
 */
function setRegionSizeState(propName: string, value: string, unit: string) {
  if (!propName) return;
  regionSizeState.value = {
    ...regionSizeState.value,
    [propName]: { value, unit },
  };
}

/**
 * 获取尺寸输入值
 * @param {string} propName - 属性名
 * @returns {string}
 */
function getSizeValue(propName: string): string {
  if (!propName) return "";
  const cached = getRegionSizeState(propName);
  if (cached) return cached.value;
  return "";
}

/**
 * 获取尺寸单位
 * @param {string} propName - 属性名
 * @returns {string}
 */
function getSizeUnit(propName: string): string {
  if (!propName) return "auto";
  const cached = getRegionSizeState(propName);
  if (cached) return cached.unit;
  return "auto";
}

/**
 * 处理尺寸值变更
 * @param {string} propName - 属性名
 * @param {string} value - 输入值
 */
function handleSizeValueChange(propName: string, value: string) {
  if (!propName) return;
  const unit = getSizeUnit(propName);
  const nextUnit = unit === "auto" ? "px" : unit;
  if (!value) {
    setRegionSizeState(propName, "", "auto");
    handlePropChange(propName, "auto");
    return;
  }
  const clamped = clampRegionSize(propName, value, nextUnit);
  setRegionSizeState(propName, clamped.value || "", clamped.unit);
  handlePropChange(propName, `${clamped.value}${clamped.unit}`);
}

/**
 * 处理尺寸单位变更
 * @param {string} propName - 属性名
 * @param {string} unit - 单位
 */
function handleSizeUnitChange(propName: string, unit: string) {
  if (!propName) return;
  if (unit === "auto") {
    setRegionSizeState(propName, "", "auto");
    handlePropChange(propName, "auto");
    return;
  }
  const value = getSizeValue(propName) || "100";
  const clamped = clampRegionSize(propName, value, unit);
  setRegionSizeState(propName, clamped.value, clamped.unit);
  handlePropChange(propName, `${clamped.value}${clamped.unit}`);
}

/**
 * 将 Props 格式化为 JSON 字符串
 */
const formattedProps = computed(() => {
  const target = currentElement.value;
  if (!target) return "{}";
  return JSON.stringify(target.props || {}, null, 2);
});

/**
 * 判断属性是否已绑定
 * @param {string} propName - 属性名
 * @returns {boolean}
 */
function hasPropBinding(propName: string): boolean {
  return Boolean(currentElement.value?.bindings?.[propName]);
}

/**
 * 当前是否有样式选择目标
 */
const hasStyleSelection = computed(() => selectedNode.value !== null);

const hasDetailConfig = computed(() => {
  if (isEChart.value) {
    const value = currentElement.value?.props?.option;
    return Boolean(String(value || "").trim());
  }
  const value = currentElement.value?.detailConfig;
  return Boolean(String(value || "").trim());
});

const hasStyleConfig = computed(() => {
  const value = currentElement.value?.styleConfig;
  return Boolean(String(value || "").trim());
});

/**
 * 判断属性是否需要隐藏
 * @param {any} propDef - 属性定义
 * @returns {boolean}
 */
function shouldHideProp(propDef: AnyRecord): boolean {
  if (!propDef || !propDef.name) return false;
  if (propDef.hidden === true) return true;
  const nodeType = elementTypeName(selectedNode.value?.type);
  if (nodeType === "Tabs" && propDef.name === "tabs") return true;
  if (nodeType === "Menu" && propDef.name === "items") return true;
  if (nodeType === "ElContainer" && EL_CONTAINER_MANAGED_PROPS.has(propDef.name)) return true;

  return false;
}

/**
 * 是否显示尺寸编辑器
 */
const showSizeEditor = computed<boolean>(() => {
  const node = selectedNode.value;
  if (!node) return true;
  const regionTypes = ["ElHeader", "ElAside", "ElMain", "ElFooter"];
  if (!regionTypes.includes(node.type)) return true;
  const parentNode = doc.value?.getParent?.(node.id);
  return parentNode?.type !== "ElContainer";
});

/**
 * 当前样式对象
 */
const containerMinSize = computed<any>(() =>
  computeElContainerMinSize(selectedNode.value, (id) => doc.value?.getNode?.(id)),
);

/**
 * ?
 */
const currentStyle = computed<AnyRecord>(() => {
  if (!selectedNode.value) return {};
  return selectedNode.value.style || {};
});

function resolveCanvasDomPosition(nodeId: string): { left: number; top: number } | null {
  const rootId = currentPage.value?.rootNodeId;
  if (!rootId || typeof document === "undefined") return null;
  const rootEl = document.querySelector<HTMLElement>(`[data-node-id="${rootId}"]`);
  const nodeEl = document.querySelector<HTMLElement>(`[data-node-id="${nodeId}"]`);
  if (!rootEl || !nodeEl) return null;
  const rootRect = rootEl.getBoundingClientRect();
  const nodeRect = nodeEl.getBoundingClientRect();
  const scale = rootEl.offsetWidth > 0 ? rootRect.width / rootEl.offsetWidth : 1;
  const zoomValue = Number.isFinite(scale) && scale > 0 ? scale : 1;
  return {
    left: Math.round((nodeRect.left - rootRect.left) / zoomValue),
    top: Math.round((nodeRect.top - rootRect.top) / zoomValue),
  };
}

/**
 * 基础信息区展示的实际位置。
 * 绝对定位节点读模型坐标；flow 节点读实际 DOM 坐标，和底部状态栏保持一致。
 */
const currentPositionStyle = computed<AnyRecord>(() => {
  void docVersion.value;
  void currentPage.value?.rootNodeId;
  const node = selectedNode.value as AnyRecord | null;
  if (node) {
    const style = node.style || {};
    if (node.positioning === "flow") {
      const domPosition = node.id ? resolveCanvasDomPosition(node.id) : null;
      if (domPosition) {
        return {
          ...style,
          left: domPosition.left,
          top: domPosition.top,
        };
      }
    }

    const absolutePos = node.absolutePos;
    if (
      absolutePos &&
      (Number.isFinite(absolutePos.x) || Number.isFinite(absolutePos.y))
    ) {
      return {
        ...style,
        left: Number.isFinite(absolutePos.x) ? absolutePos.x : style.left,
        top: Number.isFinite(absolutePos.y) ? absolutePos.y : style.top,
      };
    }

    const free = node.layoutItem?.free;
    const abs = free?.abs;
    if (abs) {
      return {
        ...style,
        left: Number.isFinite(abs.x) ? abs.x : style.left,
        top: Number.isFinite(abs.y) ? abs.y : style.top,
      };
    }

    const constraints = free?.constraints;
    if (constraints) {
      return {
        ...style,
        left: constraints.left ?? style.left,
        top: constraints.top ?? style.top,
      };
    }

    const domPosition = node.id ? resolveCanvasDomPosition(node.id) : null;
    return domPosition ? { ...style, left: domPosition.left, top: domPosition.top } : style;
  }

  const graphic = selectedGraphic.value as AnyRecord | null;
  if (graphic) {
    return {
      ...(graphic.style || {}),
      left: graphic.x,
      top: graphic.y,
    };
  }
  return {};
});

/**
 * 处理样式变更
 * @param {Object} newStyle - 新样式
 */
function handleStyleChange(newStyle: AnyRecord) {
  if (!selectedNode.value) return;
  const nodeId = selectedNode.value.id;
  const nextPatch: AnyRecord = { style: newStyle };
  const ok = editorStore.updateNode(nodeId, nextPatch);
  const latest = doc.value?.getNode?.(nodeId);
  const shouldPatch =
    !ok || (latest && JSON.stringify(latest.style || {}) !== JSON.stringify(newStyle || {}));
  if (shouldPatch) {
    applyNodePatch(nodeId, nextPatch);
  }
  docVersion.value += 1;
}

/**
 * 更新折叠面板专用属性，保持运行态字段仍落在 props 中。
 * @param {Record<string, unknown>} propsPatch - 折叠面板 props 补丁
 */
function handleCollapsePropsChange(propsPatch: AnyRecord) {
  const node = selectedNode.value;
  if (!node || elementType.value !== "Collapse") return;
  const nextProps = { ...(node.props || {}), ...(propsPatch || {}) };
  // Collapse 已由可视化表单接管结构配置，旧 detailConfig DSL 会在预览/刷新时重放并覆盖 props。
  const patch = { props: nextProps, detailConfig: "" };
  const ok = editorStore.updateNode(node.id, patch);
  if (!ok) {
    applyNodePatch(node.id, patch);
  }
  docVersion.value += 1;
  editorStore.saveCurrentPageDraft?.();
}

function handleTabsPropsChange(propsPatch: AnyRecord) {
  const node = selectedNode.value;
  if (!node || elementType.value !== "Tabs") return;
  const nextProps = { ...(node.props || {}), ...(propsPatch || {}) };
  if (nextProps.type === "Tabs") {
    delete nextProps.type;
  }
  // Tabs 结构由可视化表单接管；清空旧 detailConfig，避免预览/刷新时重放旧 DSL 覆盖 props。
  const patch = { props: nextProps, detailConfig: "" };
  const ok = editorStore.updateNode(node.id, patch);
  if (!ok) {
    applyNodePatch(node.id, patch);
  }
  docVersion.value += 1;
  editorStore.saveCurrentPageDraft?.();
}

/**
 * 同步折叠项标识变更到直接子组件的 collapseKey。
 * @param {{id: string, props: Record<string, unknown>}[]} patches - 子组件 props 补丁列表
 */
function handleCollapseChildPatches(patches: AnyArray) {
  patches.forEach((item) => {
    if (!item?.id || !item.props) return;
    const patch = { props: item.props };
    const ok = editorStore.updateNode(item.id, patch);
    if (!ok) {
      applyNodePatch(item.id, patch);
    }
  });
  editorStore.saveCurrentPageDraft?.();
}

function handleTabsChildPatches(patches: AnyArray) {
  patches.forEach((item) => {
    if (!item?.id || !item.props) return;
    const patch = { props: item.props };
    const ok = editorStore.updateNode(item.id, patch);
    if (!ok) {
      applyNodePatch(item.id, patch);
    }
  });
  editorStore.saveCurrentPageDraft?.();
}

/**
 * 打开配置弹窗
 * @param {"style" | "detail"} type - 配置类型
 */
function openConfigDialog(type: "style" | "detail") {
  if (!selectedNode.value) return;
  configDialogType.value = type === "detail" ? "detail" : "style";
  if (configDialogType.value === "detail") {
    const node = selectedNode.value;
    if (node?.type === "EChart") {
      configDraft.value = String(node?.props?.option || "");
    } else {
      configDraft.value = String(node.detailConfig || "");
    }
  } else {
    configDraft.value = selectedNode.value.styleConfig || "";
  }
  selectedPresetId.value = "";
  presetSearch.value = "";
  configDialogVisible.value = true;
}

/**
 * 保存配置弹窗内容
 */
function saveConfigDialog(): void {
  const node = selectedNode.value;
  if (!node) return;
  let content = String(configDraft.value || "");
  if (configDialogType.value === "style") {
    const normalized = formatStyleConfigOutput(content);
    content = prefixStyleConfigScope(normalized);
    configDraft.value = content;
  } else if (!isEChart.value) {
    content = normalizeDetailConfigBindings(content);
    configDraft.value = content;
    if (!content.trim()) {
      resetNodeDetailToDefault(node);
      configDialogVisible.value = false;
      return;
    }
    const validation = validateDetailConfig(content);
    if (!validation.valid) {
      ElMessage.warning({
        message: validation.message || t("propertyPanel.configDialog.validationFailed"),
      });
      return;
    }
  } else {
    configDraft.value = content;
  }

  if (configDialogType.value === "detail") {
    if (isEChart.value) {
      editorStore.updateNode(node.id, {
        props: { ...(node.props || {}), option: content },
      });
    } else {
      const ok = editorStore.updateNode(node.id, { detailConfig: content });
      if (!ok) {
        applyLocalNodePatch(node, { detailConfig: content });
      }
      emitDetailConfig(node.id, content);
      if (elementTypeName(node.type) === "Menu") {
        const config = resolveMenuConfigFromContent(content);
        if (!config) {
          ElMessage.error({ message: t("propertyPanel.configDialog.menuDslInvalid") });
          return;
        }
        applyMenuDetailConfig(node, config);
        const propsPatch: AnyRecord = {
          ...(node.props || {}),
          ...(config.props && typeof config.props === "object" ? { ...config.props } : {}),
        };
        if (Array.isArray(config.items)) {
          propsPatch.items = config.items;
        }
        if (Array.isArray(propsPatch.items)) {
          propsPatch.items = normalizeMenuItems(propsPatch.items);
        }
        const nextPatch: AnyRecord = { detailConfig: content, props: propsPatch };
        if (config.style && typeof config.style === "object") {
          nextPatch.style = { ...(node.style || {}), ...config.style };
        }
        forceUpdateNode(node.id, nextPatch);
        if (typeof window !== "undefined") {
          window.dispatchEvent(new CustomEvent("designer:force-refresh"));
        }
      }
      runDetailConfigLocal(node, content);
    }
  } else {
    editorStore.updateNode(node.id, { styleConfig: content });
  }
  configDialogVisible.value = false;
}

function saveStyleConfigDialog(content: string): void {
  const node = selectedNode.value;
  if (!node) return;
  const normalized = formatStyleConfigOutput(String(content || ""));
  const scoped = prefixStyleConfigScope(normalized);
  editorStore.updateNode(node.id, { styleConfig: scoped });
  configDialogVisible.value = false;
}

/**
 * 详细配置编辑时实时应用到组件（草稿态）
 */
watch([configDraft, configDialogVisible, configDialogType, () => selectedNode.value?.id], () => {
  if (!configDialogVisible.value) return;
  if (configDialogType.value !== "detail") return;
  const node = selectedNode.value;
  if (!node) return;
  if (detailDraftTimer) clearTimeout(detailDraftTimer);
  detailDraftTimer = setTimeout(() => {
    const raw = String(configDraft.value || "");
    if (isEChart.value) {
      editorStore.updateNode(node.id, {
        props: { ...(node.props || {}), option: raw },
      });
      return;
    }
    const content = normalizeDetailConfigBindings(raw);
    const validation = validateDetailConfig(content);
    if (!validation.valid) return;
    editorStore.updateNode(node.id, { detailConfig: content });
    emitDetailConfig(node.id, content);
  }, 200);
});

/**
 * 清除配置
 */
function clearConfigDialog() {
  const node = selectedNode.value;
  if (!node) return;
  configDraft.value = "";
  if (configDialogType.value === "detail") {
    if (isEChart.value) {
      editorStore.updateNode(node.id, {
        props: { ...(node.props || {}), option: "" },
      });
    } else {
      resetNodeDetailToDefault(node);
    }
  } else {
    editorStore.updateNode(node.id, { styleConfig: "" });
  }
}

/**
 * 应用预设模板
 * @param {string} id - 预设 ID
 */
function handlePresetChange(id: string): void {
  const target = currentPresetOptions.value.find((item) => item.id === id);
  if (!target) return;
  const nextContent = formatStyleConfigOutput(target.content || "");
  if (!nextContent) return;
  const current = String(configDraft.value || "").trim();
  const remark = target.label ? `/* ${target.label} */` : "";
  const remarkBlock = remark ? `${remark}\n` : "";
  const separator = current ? "\n\n" : "";
  configDraft.value = `${current}${separator}${remarkBlock}${nextContent}`;
  nextTick(() => {
    configDraft.value = String(configDraft.value ?? "");
  });
}

/**
 * 获取绑定表达式
 * @param {string} propName - 属性名
 * @returns {string}
 */
function resolveBindingExpr(propName: string): string {
  if (!propName) return "";
  const target = currentElement.value;
  const binding = target?.bindings?.[propName];
  if (!binding || typeof binding !== "object") return "";
  if (binding.kind === "expr") return binding.expr || "";
  if (binding.kind === "var" && binding.name) {
    return binding.scope === "global"
      ? `$global.${binding.name || ""}`
      : `$vars.${binding.name || ""}`;
  }
  return "";
}

/**
 * 结构化表达式/变量引用
 * @param {string} raw - 输入值
 * @returns {string}
 */
function normalizeBindingReference(raw: string): string {
  const trimmed = String(raw || "").trim();
  if (!trimmed) return "";
  if (trimmed.startsWith("{ $var:") || trimmed.startsWith("{ $expr:")) return trimmed;
  if (trimmed.startsWith("$vars.") || trimmed.startsWith("$global.")) {
    return `{ $var: "${trimmed}" }`;
  }
  return `{ $expr: "${trimmed.replace(DOUBLE_QUOTE_RE, '\\"')}" }`;
}

/**
 * 解析绑定输入
 * @param {string} raw - 输入值
 * @returns {{ kind: "expr" | "var", expr?: string, scope?: "page" | "global", name?: string } | null}
 */
function parseBindingInput(
  raw: string,
): { kind: "expr" | "var"; expr?: string; scope?: "page" | "global"; name?: string } | null {
  const trimmed = String(raw || "").trim();
  if (!trimmed) return null;
  if (trimmed.startsWith("$global.")) {
    const name = trimmed.slice("$global.".length).trim();
    if (!name) return null;
    return { kind: "var", scope: "global", name };
  }
  if (trimmed.startsWith("$vars.")) {
    const name = trimmed.slice("$vars.".length).trim();
    if (!name) return null;
    return { kind: "var", scope: "page", name };
  }
  return { kind: "expr", expr: trimmed };
}

/**
 * 打开绑定编辑器
 * @param {{ name: string, label?: string }} propDef - 属性定义
 */
function handleBindClick(propDef: AnyRecord) {
  if (!propDef?.name || !currentElement.value) return;
  bindingProp.value = {
    name: propDef.name,
    label: propDef.label || propDef.name,
  };
  bindingEditorCode.value = resolveBindingExpr(propDef.name);
  bindingDialogVisible.value = true;
}

/**
 * 插入脚本
 * @param {{ type?: string, label?: string, params?: string }} data - 节点数据
 */
function handleBindingCustomScriptInsert(data: SidebarNodeLike) {
  if (data?.type === "group") return;
  const params = data?.params ? data.params : "";
  const call = params ? `customScripts.${data.label}(${params})` : `customScripts.${data.label}()`;
  bindingEditorRef.value?.insertText?.(call);
}

/**
 * 插入页面组件引用
 * @param {{ componentName?: string }} data - 节点数据
 */
function handleBindingComponentInsert(data: SidebarNodeLike) {
  if (!data?.componentName) return;
  bindingEditorRef.value?.insertText?.(`components.${data.componentName}`);
}

function handleBindingPageVariableInsert(row: BindingRowLike) {
  if (!row?.name) return;
  bindingEditorRef.value?.insertText?.(resolvePageVariableSnippet(row.name));
}

/**
 * 插入自定义脚本到详细配置
 * @param {{ type?: string, label?: string, params?: string }} data - 节点数据
 */
function handleConfigCustomScriptInsert(data: SidebarNodeLike) {
  if (data?.type === "group") return;
  const params = data?.params ? data.params : "";
  const call = params ? `customScripts.${data.label}(${params})` : `customScripts.${data.label}()`;
  configEditorRef.value?.insertText?.(call);
}

/**
 * @param {{ componentName?: string }} data - 节点数据
 * @param {{ componentName?: string }} data - 节点数据
 */
function handleConfigComponentInsert(data: SidebarNodeLike) {
  if (!data?.componentName) return;
  configEditorRef.value?.insertText?.(`components.${data.componentName}`);
}

function handleConfigPageVariableInsert(row: BindingRowLike) {
  if (!row?.name) return;
  configEditorRef.value?.insertText?.(resolvePageVariableSnippet(row.name));
}

/**
 * 打开变量枚举
 */
function openBindingVariableEnum(target: "binding" | "config" = "binding") {
  enumInsertTarget.value = target;
  bindingProjectVarSearch.value = "";
  bindingEnumSelectedProjectGroupId.value = null;
  bindingEnumSelectedProjectVar.value = null;
  bindingVariableEnumVisible.value = true;
}

/**
 * 打开变量枚举（详细配置）
 */
function openConfigVariableEnum() {
  openBindingVariableEnum("config");
}

function handleBindingProjectGroupSelect(data: AnyRecord | null) {
  if (!data) {
    bindingEnumSelectedProjectGroupId.value = null;
    return;
  }
  bindingEnumSelectedProjectGroupId.value = data.id === "all" ? null : data.id;
}

function handleBindingProjectRowClick(row: BindingRowLike | null) {
  bindingEnumSelectedProjectVar.value = row || null;
}

function handleBindingProjectRowDblClick(row: BindingRowLike | null) {
  bindingEnumSelectedProjectVar.value = row || null;
  confirmBindingEnumInsert();
}

function bindingEnumProjectRowClass(...args: unknown[]): string {
  const [{ row }] = args as [{ row: BindingRowLike }];
  if (bindingEnumSelectedProjectVar.value?.name === row.name) return "is-selected";
  return "";
}

function confirmBindingEnumInsert(): void {
  const insertText = (text: string) => {
    if (enumInsertTarget.value === "config") {
      configEditorRef.value?.insertText?.(text);
      return;
    }
    bindingEditorRef.value?.insertText?.(text);
  };
  if (bindingEnumSelectedProjectVar.value?.name) {
    insertText(resolveProjectVariableSnippet(bindingEnumSelectedProjectVar.value.name));
    bindingVariableEnumVisible.value = false;
  }
}

/**
 * 保存绑定配置
 */
function saveBinding(): void {
  const target = currentElement.value;
  const propName = bindingProp.value?.name;
  if (!target || !propName) return;
  const code = String(bindingEditorCode.value || "");
  const nextBindings = { ...(target.bindings || {}) };
  const parsed = parseBindingInput(code);
  if (!parsed) {
    delete nextBindings[propName];
  } else {
    if (parsed.kind === "var") {
      nextBindings[propName] = {
        kind: "var",
        scope: parsed.scope,
        name: parsed.name,
        fallback: target.props?.[propName],
      };
    } else {
      nextBindings[propName] = {
        kind: "expr",
        expr: parsed.expr,
        fallback: target.props?.[propName],
      };
    }
  }
  if (selectedNode.value) {
    editorStore.updateNode(target.id, { bindings: nextBindings });
  } else if (selectedGraphic.value) {
    editorStore.updateGraphic(target.id, { bindings: nextBindings });
  }
  bindingEditorCode.value = normalizeBindingReference(code);
  bindingDialogVisible.value = false;
}

/**
 * 处理组件标签修改
 */
function handleLabelChange(): void {
  const el = currentElement.value;
  if (!el) return;

  const nextLabel = (elementLabel.value || "").trim();
  if (!nextLabel) {
    elementLabel.value = el.label || "";
    return;
  }
  if (!editorStore.isLabelUnique?.(nextLabel, el.id)) {
    ElMessage.warning({ message: t("propertyPanel.duplicateName") });
    elementLabel.value = el.label || "";
    return;
  }

  if (selectedNode.value) {
    editorStore.updateNode(el.id, { label: nextLabel });
  } else if (selectedGraphic.value) {
    editorStore.updateGraphic(el.id, { label: nextLabel });
  }
}

/**
 * 处理组件描述修改
 */
function handleDescriptionChange(): void {
  const el = currentElement.value;
  if (!el) return;

  const nextDescription = String(elementDescription.value || "").trim();
  if (selectedNode.value) {
    editorStore.updateNode(el.id, { description: nextDescription });
    applyLocalNodePatch(el, { description: nextDescription });
  } else if (selectedGraphic.value) {
    editorStore.updateGraphic(el.id, { description: nextDescription });
    applyLocalNodePatch(el, { description: nextDescription });
  }
}

/**
 * 更新按钮 DOM ID
 * @param {string} value - DOM ID
 */
function handlePropChange(propName: string, value: any): void {
  const el = currentElement.value;
  if (!el) return;

  const newProps: AnyRecord = { ...(el.props || {}), [propName]: value };
  const isTextComponent = elementTypeName(el.type) === "Text";
  const isButtonComponent = elementTypeName(el.type) === "Button";
  let nextStyle: AnyRecord = {};
  let shouldPatchStyle = false;
  if (isButtonComponent) {
    if (propName === "shape") {
      Object.assign(newProps, resolveButtonShapeProps(value));
    }
  }
  if (isTextComponent) {
    nextStyle = { ...(el.style || {}) };
    shouldPatchStyle = true;
    const stylePatch: AnyRecord = {};
    if (propName === "fontSize") stylePatch.fontSize = value;
    if (propName === "fontWeight") stylePatch.fontWeight = value;
    if (propName === "fontFamily") stylePatch.fontFamily = value;
    if (propName === "color") stylePatch.color = value;
    if (propName === "textAlign") {
      stylePatch.textAlign = value;
      if (value === "justify") {
        stylePatch.textAlignLast = "justify";
      } else {
        delete nextStyle.textAlignLast;
      }
      if (value && value !== "left") {
        if (!nextStyle.width) {
          nextStyle.width = "100%";
        }
        if (!nextStyle.display) {
          nextStyle.display = "block";
        }
      }
    }
    if (propName === "lineHeight") stylePatch.lineHeight = value;
    if (propName === "letterSpacing") stylePatch.letterSpacing = value;
    if (propName === "wordBreak") stylePatch.wordBreak = value;

    const lineClampValue = propName === "lineClamp" ? Number(value) : Number(newProps.lineClamp);
    const truncateValue = propName === "truncate" ? Boolean(value) : Boolean(newProps.truncate);
    if (Number.isFinite(lineClampValue) && lineClampValue > 0) {
      nextStyle.display = "-webkit-box";
      nextStyle.overflow = "hidden";
      nextStyle.WebkitLineClamp = lineClampValue;
      nextStyle.WebkitBoxOrient = "vertical";
      nextStyle.whiteSpace = "normal";
      nextStyle.textOverflow = "clip";
    } else if (truncateValue) {
      nextStyle.whiteSpace = "nowrap";
      nextStyle.overflow = "hidden";
      nextStyle.textOverflow = "ellipsis";
      delete nextStyle.WebkitLineClamp;
      delete nextStyle.WebkitBoxOrient;
      if (nextStyle.display === "-webkit-box") {
        delete nextStyle.display;
      }
    } else {
      delete nextStyle.WebkitLineClamp;
      delete nextStyle.WebkitBoxOrient;
      if (nextStyle.display === "-webkit-box") {
        delete nextStyle.display;
      }
      delete nextStyle.whiteSpace;
      delete nextStyle.textOverflow;
    }

    if (Object.keys(stylePatch).length > 0) {
      Object.assign(nextStyle, stylePatch);
    }
  }
  const currentBindings = el.bindings || {};
  const hasBinding = Object.hasOwn(currentBindings, propName);
  const nextBindings = hasBinding
    ? Object.fromEntries(Object.entries(currentBindings).filter(([key]) => key !== propName))
    : currentBindings;
  if (regionSizeDefaults[propName]) {
    const parsed = parseSize(value);
    setRegionSizeState(propName, parsed.value, parsed.unit);
  }

  if (selectedNode.value) {
    const patch: AnyRecord = { props: newProps };
    if (shouldPatchStyle) {
      patch.style = nextStyle;
    }
    if (hasBinding) {
      patch.bindings = nextBindings;
    }
    const nodeId = el.id;
    const ok = editorStore.updateNode(nodeId, patch);
    const latest = doc.value?.getNode?.(nodeId);
    const latestValue = latest?.props?.[propName];
    const shouldPatch = !ok || latestValue !== value;
    if (shouldPatch) {
      applyNodePatch(nodeId, patch);
    }
    docVersion.value += 1;
  } else if (selectedGraphic.value) {
    editorStore.updateGraphic(el.id, { props: newProps });
  }
}

/**
 * 更新组件运行态权限方案引用；空值表示该维度始终允许。
 */
function updateNodeRuntimeAccess(key: "visibleSchemeId" | "operableSchemeId", value: string): void {
  const node = selectedNode.value;
  if (!node) return;
  const nextRuntimeAccess: AnyRecord = {
    ...(node.permissions?.runtimeAccess || {}),
    [key]: String(value || "").trim() || undefined,
  };
  Object.keys(nextRuntimeAccess).forEach((itemKey) => {
    if (!nextRuntimeAccess[itemKey]) {
      delete nextRuntimeAccess[itemKey];
    }
  });
  const nextPermissions: AnyRecord = {
    ...(node.permissions || {}),
    runtimeAccess: Object.keys(nextRuntimeAccess).length > 0 ? nextRuntimeAccess : undefined,
  };
  if (!nextPermissions.runtimeAccess) {
    delete nextPermissions.runtimeAccess;
  }
  const patch: AnyRecord = { permissions: nextPermissions };
  const ok = editorStore.updateNode(node.id, patch);
  if (!ok) {
    applyNodePatch(node.id, patch);
  }
  docVersion.value += 1;
}
</script>

<template>
  <div class="property-panel-root">
    <!-- 页面属性面板：未选中任何元素 -->
    <PageInspectorPanel v-if="panelState === 'page'" />

    <!-- 多选面板：选中多个元素 -->
    <MultiInspectorPanel v-else-if="panelState === 'multi'" :elements="selectedElements" />

    <!-- 单选面板：选中单个元素 -->
    <div v-else class="element-inspector">
      <PropertyPanelBasicSection
        v-model:label="elementLabel"
        v-model:description="elementDescription"
        :element-type="elementType"
        :element-id="elementId"
        :current-style="currentPositionStyle"
        @label-commit="handleLabelChange"
        @description-commit="handleDescriptionChange"
      />

      <template v-if="hasStyleSelection">
        <PropertyPanelConfigStrip
          :show-size-editor="showSizeEditor"
          :current-style="currentStyle"
          :container-min-size="containerMinSize"
          :has-detail-config="hasDetailConfig"
          :has-style-config="hasStyleConfig"
          @update:current-style="handleStyleChange"
          @open-config="openConfigDialog"
        />
      </template>

      <div class="prop-section">
        <div class="prop-section-header is-static">
          <span class="prop-section-heading">
            <span class="prop-section-title">权限控制</span>
          </span>
        </div>
        <div class="prop-section-body runtime-access-controls">
          <div class="runtime-access-row">
            <span class="runtime-access-row__label">可见方案</span>
            <el-select v-model="visibleSchemeId" size="small" clearable placeholder="始终可见">
              <el-option
                v-for="item in runtimePermissionSchemeOptions"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              />
            </el-select>
          </div>
          <div class="runtime-access-row">
            <span class="runtime-access-row__label">可操作方案</span>
            <el-select v-model="operableSchemeId" size="small" clearable placeholder="始终可操作">
              <el-option
                v-for="item in runtimePermissionSchemeOptions"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              />
            </el-select>
          </div>
        </div>
      </div>

      <!-- 属性表单：根据 Manifest 生成 -->
      <CollapseItemsEditor
        v-if="isCollapseNode"
        :node-props="selectedNode?.props || {}"
        :child-nodes="collapseChildNodes"
        @props-change="handleCollapsePropsChange"
        @child-patches="handleCollapseChildPatches"
      />
      <TabsItemsEditor
        v-else-if="isTabsNode"
        :node-props="selectedNode?.props || {}"
        :child-nodes="tabsChildNodes"
        @props-change="handleTabsPropsChange"
        @child-patches="handleTabsChildPatches"
      />
      <template v-else-if="layoutForceProps.length > 0">
        <PropertyPanelLayoutForceSection
          :visible-props="visibleLayoutForceProps"
          :should-show-bind-button="shouldShowBindButton"
          :has-prop-binding="hasPropBinding"
          :get-prop-value="getPropValue"
          @bind-click="handleBindClick"
          @prop-change="handlePropChange"
        />
      </template>
      <template v-else-if="effectiveManifest && effectiveManifest.props.length > 0">
        <div class="prop-sections">
          <div v-if="isElContainer" class="prop-section region-config-section">
            <div class="prop-section-header is-static">
              <span class="prop-section-heading">
                <IconEpGrid class="section-icon" />
                <span class="prop-section-title">区域配置</span>
              </span>
            </div>
            <div class="prop-section-body region-config-body">
              <div class="region-config-block">
                <div class="region-config-label">布局样式</div>
                <div class="region-preset-grid">
                  <button
                    v-for="option in EL_CONTAINER_PRESET_OPTIONS"
                    :key="option.value"
                    class="region-preset-card"
                    :class="{ 'is-active': activeRegionPreset === option.value }"
                    type="button"
                    @click="handlePropChange('regionPreset', option.value)"
                  >
                    <span
                      class="region-preset-preview"
                      :style="{ gridTemplateAreas: regionPresetGridAreas(option) }"
                    >
                      <span
                        v-show="String(option.grid).includes('header')"
                        class="region-preview-part region-preview-header"
                      />
                      <span
                        v-show="String(option.grid).includes('aside')"
                        class="region-preview-part region-preview-aside"
                      />
                      <span
                        v-show="String(option.grid).includes('main')"
                        class="region-preview-part region-preview-main"
                      />
                      <span
                        v-show="String(option.grid).includes('footer')"
                        class="region-preview-part region-preview-footer"
                      />
                    </span>
                    <span class="region-preset-copy">
                      <span class="region-preset-title">{{ option.label }}</span>
                      <span class="region-preset-desc">{{ option.description }}</span>
                    </span>
                  </button>
                </div>
              </div>

              <div class="region-config-block">
                <div class="region-config-label">显示区域</div>
                <div class="region-toggle-grid">
                  <button
                    v-for="item in regionPropRows"
                    :key="item.key"
                    class="region-toggle-card"
                    :class="{ 'is-active': isRegionEnabled(item) }"
                    type="button"
                    @click="handlePropChange(item.toggleProp, !isRegionEnabled(item))"
                  >
                    <span>{{ item.label }}</span>
                    <ElSwitch
                      class="region-toggle"
                      :model-value="isRegionEnabled(item)"
                      size="small"
                      @click.stop
                      @update:model-value="
                        (val: any) => handlePropChange(item.toggleProp, Boolean(val))
                      "
                    />
                  </button>
                </div>
              </div>

              <div v-if="visibleRegionSizeRows.length" class="region-config-block">
                <div class="region-config-label">区域尺寸</div>
                <div
                  v-for="item in visibleRegionSizeRows"
                  :key="item.key"
                  class="region-size-row"
                >
                  <div class="region-size-row-label">
                    <span>{{ item.label }}{{ item.sizeLabel ? ` ${item.sizeLabel}` : "" }}</span>
                    <span v-if="getRegionMaxLabel(item.sizeProp)" class="region-size-limit">
                      {{ getRegionMaxLabel(item.sizeProp) }}
                    </span>
                  </div>
                  <div class="region-prop-controls">
                    <ElInput
                      class="region-size-input"
                      :model-value="getSizeValue(item.sizeProp)"
                      size="small"
                      placeholder="auto"
                      @update:model-value="
                        (val: any) => handleSizeValueChange(item.sizeProp, val)
                      "
                    />
                    <ElSelect
                      :model-value="getSizeUnit(item.sizeProp)"
                      size="small"
                      class="region-unit-select"
                      @update:model-value="(val: any) => handleSizeUnitChange(item.sizeProp, val)"
                      @change="(val: any) => handleSizeUnitChange(item.sizeProp, val)"
                    >
                      <el-option label="px" value="px" />
                      <el-option label="%" value="%" />
                      <el-option label="auto" value="auto" />
                    </ElSelect>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <template v-for="group in displayPropGroups" :key="group.name">
            <div
              class="prop-section"
              :class="{
                'is-collapsed': !isSectionExpanded(getGroupSectionKey(group)),
              }"
            >
              <button
                class="prop-section-header"
                type="button"
                @click="toggleSection(getGroupSectionKey(group))"
              >
                <span class="prop-section-heading">
                  <IconEpArrowRight class="section-chevron" />
                  <span class="prop-section-title">{{ resolveGroupTitle(group) }}</span>
                </span>
              </button>
              <div v-show="isSectionExpanded(getGroupSectionKey(group))" class="prop-section-body">
                <template v-if="isElContainer && isRegionGroup(group)">
                  <div
                    v-for="item in regionPropRows"
                    :key="item.key"
                    class="prop-item region-prop-row"
                  >
                    <div class="prop-label">
                      <div class="region-label">
                        <span>{{ item.label }}</span>
                        <span v-if="getRegionSizeText(item)" class="region-size-text">
                          {{ getRegionSizeText(item) }}
                        </span>
                        <span v-if="getRegionMaxLabel(item.sizeProp)" class="region-size-limit">
                          {{ getRegionMaxLabel(item.sizeProp) }}
                        </span>
                      </div>
                      <ElSwitch
                        v-if="!item.sizeProp"
                        class="region-toggle"
                        :model-value="Boolean(getPropValue(item.toggleProp))"
                        size="small"
                        @update:model-value="
                          (val: any) => handlePropChange(item.toggleProp, Boolean(val))
                        "
                      />
                    </div>
                    <div v-if="item.sizeProp" class="region-prop-controls">
                      <ElInput
                        class="region-size-input"
                        :model-value="getSizeValue(item.sizeProp)"
                        size="small"
                        placeholder="auto"
                        :disabled="!isRegionEnabled(item)"
                        @update:model-value="
                          (val: any) => handleSizeValueChange(item.sizeProp, val)
                        "
                      />
                      <ElSelect
                        :model-value="getSizeUnit(item.sizeProp)"
                        size="small"
                        class="region-unit-select"
                        :disabled="!isRegionEnabled(item)"
                        @update:model-value="(val: any) => handleSizeUnitChange(item.sizeProp, val)"
                        @change="(val: any) => handleSizeUnitChange(item.sizeProp, val)"
                      >
                        <el-option label="px" value="px" />
                        <el-option label="%" value="%" />
                        <el-option label="auto" value="auto" />
                      </ElSelect>
                      <ElSwitch
                        class="region-toggle"
                        :model-value="Boolean(getPropValue(item.toggleProp))"
                        size="small"
                        @update:model-value="
                          (val: any) => handlePropChange(item.toggleProp, Boolean(val))
                        "
                      />
                    </div>
                  </div>
                </template>
                <template v-else>
                  <div
                    v-for="(propDef, propIndex) in getVisibleGroupProps(group)"
                    :key="propDef?.name || propIndex"
                    class="prop-item"
                  >
                    <div class="prop-label">
                      <span>{{ propDef.label }}</span>
                      <el-tooltip
                        v-if="shouldShowBindButton(propDef)"
                        :content="t('propertyPanel.bindingDialog.tooltip')"
                        placement="top"
                      >
                        <button
                          class="bind-btn"
                          :class="{ 'is-active': hasPropBinding(propDef.name) }"
                          @click="handleBindClick(propDef)"
                        >
                          <IconEpLink class="bind-icon" />
                        </button>
                      </el-tooltip>
                    </div>
                    <PropEditor
                      :prop="propDef"
                      :model-value="getPropValue(propDef.name)"
                      @update:model-value="(val: any) => handlePropChange(propDef.name, val)"
                    />
                  </div>
                </template>
              </div>
            </div>
          </template>
        </div>
      </template>

      <!-- 无 Manifest 时显示原始 Props -->
      <PropertyPanelRawPropsFallback v-else :text="formattedProps" />

      <div v-if="!hasStyleSelection" class="text-sm text-gray-400 text-center py-6">
        {{ t("propertyPanel.stylePanel.selectComponent") }}
      </div>
    </div>
  </div>

  <StyleConfigEditorDialog
    v-if="configDialogType === 'style'"
    v-model="configDialogVisible"
    :content="currentStyleConfigContent"
    :title="t('propertyPanel.configDialog.styleTitle')"
    :presets="currentPresetOptions"
    :project-id="projectId || ''"
    :selector-label="t('propertyPanel.configDialog.selectorHint')"
    :selector-tokens="['#domId']"
    :selector-help="t('propertyPanel.configDialog.selectorHelp')"
    :template-label="t('propertyPanel.configDialog.stylePreset')"
    :template-placeholder="t('propertyPanel.configDialog.selectPlaceholder')"
    :filter-label="t('propertyPanel.configDialog.filterLabel')"
    :search-template="t('propertyPanel.configDialog.searchPreset')"
    :asset-library="t('propertyPanel.configDialog.assetLibrary')"
    :search-assets="t('propertyPanel.configDialog.searchAssets')"
    :clear-text="t('propertyPanel.configDialog.clear')"
    :cancel-text="t('propertyPanel.configDialog.cancel')"
    :save-text="t('propertyPanel.configDialog.save')"
    @save="saveStyleConfigDialog"
  />

  <el-dialog
    v-if="configDialogType === 'detail'"
    v-model="configDialogVisible"
    :title="configDialogTitle"
    width="980px"
    top="4vh"
    :close-on-click-modal="false"
    :lock-scroll="false"
  >
    <div class="config-toolbar">
      <div class="config-toolbar-item">
        <span class="config-label">{{ presetLabel }}</span>
        <ElSelect
          v-model="selectedPresetId"
          size="small"
          class="config-select preset-select"
          :placeholder="t('propertyPanel.configDialog.selectPlaceholder')"
          @change="handlePresetChange"
        >
          <el-option
            v-for="item in filteredPresetOptions"
            :key="item.id"
            :label="item.label"
            :value="item.id"
          />
        </ElSelect>
      </div>
      <div class="config-toolbar-item">
        <span class="config-label">{{ t("propertyPanel.configDialog.filterLabel") }}</span>
        <ElInput
          v-model="presetSearch"
          size="small"
          class="config-select"
          :placeholder="t('propertyPanel.configDialog.searchPreset')"
          clearable
        />
      </div>
      <div v-if="configDialogType === 'detail'" class="config-toolbar-item config-toolbar-actions">
        <el-tooltip :content="t('propertyPanel.configDialog.variableEnum')" placement="top">
          <el-button class="icon-button" size="small" circle @click.stop="openConfigVariableEnum">
            <IconEpList />
          </el-button>
        </el-tooltip>
      </div>
    </div>
    <div class="config-body">
      <div class="config-editor">
        <MonacoEditor
          ref="configEditorRef"
          v-model="configDraft"
          :language="configEditorLanguage"
          height="520px"
          :completions="configEditorCompletions"
        />
      </div>
      <div v-if="configDialogType === 'detail'" class="config-sidebar">
        <div class="sidebar-section">
          <div class="sidebar-title">{{ t("propertyPanel.configDialog.customScripts") }}</div>
          <ElInput
            v-model="configScriptSearch"
            size="small"
            :placeholder="t('propertyPanel.configDialog.searchScripts')"
            clearable
          />
          <div class="sidebar-scroll">
            <el-tree
              ref="configCustomTreeRef"
              :data="bindingCustomScriptTree"
              node-key="id"
              :default-expand-all="true"
              :expand-on-click-node="false"
              :filter-node-method="filterBindingSidebarNode"
              @node-click="handleConfigCustomScriptInsert"
            >
              <template #default="{ data }">
                <div class="tree-node" :class="`node-${data.type}`">
                  <el-icon class="node-icon icon-custom">
                    <IconEpFolder v-if="data.type === 'group'" />
                    <IconEpEditPen v-else />
                  </el-icon>
                  <span class="node-label" :class="{ 'is-group': data.type === 'group' }">
                    {{ data.label }}
                  </span>
                </div>
              </template>
            </el-tree>
          </div>
        </div>
        <div class="sidebar-section">
          <div class="sidebar-title">{{ t("propertyPanel.bindingVarEnum.pageVars") }}</div>
          <ElInput
            v-model="configPageContextSearch"
            size="small"
            :placeholder="t('propertyPanel.bindingVarEnum.searchPageVars')"
            clearable
          />
          <div class="sidebar-scroll page-var-list">
            <div
              v-for="row in configPageVariableRows"
              :key="row.name"
              class="page-var-item"
              @click="handleConfigPageVariableInsert(row)"
            >
              <el-icon class="node-icon icon-variable">
                <IconEpLink />
              </el-icon>
              <span class="node-label">{{ row.name }}</span>
              <span class="page-var-type">{{ row.type }}</span>
            </div>
          </div>
        </div>
        <div class="sidebar-section">
          <div class="sidebar-title">{{ t("propertyPanel.configDialog.pageComponents") }}</div>
          <ElInput
            v-model="configComponentSearch"
            size="small"
            :placeholder="t('propertyPanel.configDialog.searchComponents')"
            clearable
          />
          <div class="sidebar-scroll">
            <el-tree
              ref="configComponentTreeRef"
              :data="bindingPageComponentTree"
              node-key="id"
              :default-expand-all="true"
              :expand-on-click-node="false"
              :filter-node-method="filterBindingSidebarNode"
              @node-click="handleConfigComponentInsert"
            >
              <template #default="{ data }">
                <div class="tree-node" :class="`node-${data.type}`">
                  <el-icon class="node-icon icon-component">
                    <IconEpFolder v-if="data.type === 'group'" />
                    <IconEpGrid v-else />
                  </el-icon>
                  <span class="node-label" :class="{ 'is-group': data.type === 'group' }">
                    {{ data.label }}
                  </span>
                </div>
              </template>
            </el-tree>
          </div>
        </div>
      </div>
    </div>
    <template #footer>
      <el-button @click="clearConfigDialog">{{ t("propertyPanel.configDialog.clear") }}</el-button>
      <el-button @click="configDialogVisible = false">{{
        t("propertyPanel.configDialog.cancel")
      }}</el-button>
      <el-button type="primary" @click="saveConfigDialog()">{{
        t("propertyPanel.configDialog.save")
      }}</el-button>
    </template>
  </el-dialog>

  <el-dialog
    v-model="bindingDialogVisible"
    :title="bindingDialogTitle"
    width="980px"
    top="3vh"
    :z-index="3000"
    append-to-body
    :modal-append-to-body="true"
    :close-on-click-modal="false"
    :lock-scroll="false"
  >
    <div class="editor-meta">
      <div class="meta-title">{{ bindingDialogTitle }}</div>
      <div class="meta-desc">{{ bindingDialogDescription }}</div>
      <div class="meta-actions">
        <el-tooltip :content="t('propertyPanel.bindingDialog.variableEnum')" placement="top">
          <el-button class="icon-button" size="small" circle @click.stop="openBindingVariableEnum">
            <IconEpList />
          </el-button>
        </el-tooltip>
      </div>
    </div>
    <div class="editor-body">
      <div class="editor-main">
        <MonacoEditor
          ref="bindingEditorRef"
          v-model="bindingEditorCode"
          language="javascript"
          height="520px"
          :completions="bindingCompletions"
        />
      </div>
      <div class="editor-sidebar">
        <div class="sidebar-section">
          <div class="sidebar-title">{{ t("propertyPanel.bindingDialog.customScripts") }}</div>
          <ElInput
            v-model="bindingScriptSearch"
            size="small"
            :placeholder="t('propertyPanel.bindingDialog.searchScripts')"
            clearable
          />
          <div class="sidebar-scroll">
            <el-tree
              ref="bindingCustomTreeRef"
              :data="bindingCustomScriptTree"
              node-key="id"
              :default-expand-all="true"
              :expand-on-click-node="false"
              :filter-node-method="filterBindingSidebarNode"
              @node-click="handleBindingCustomScriptInsert"
            >
              <template #default="{ data }">
                <div class="tree-node" :class="`node-${data.type}`">
                  <el-icon class="node-icon icon-custom">
                    <IconEpFolder v-if="data.type === 'group'" />
                    <IconEpEditPen v-else />
                  </el-icon>
                  <span class="node-label" :class="{ 'is-group': data.type === 'group' }">
                    {{ data.label }}
                  </span>
                </div>
              </template>
            </el-tree>
          </div>
        </div>
        <div class="sidebar-section">
          <div class="sidebar-title">{{ t("propertyPanel.bindingVarEnum.pageVars") }}</div>
          <ElInput
            v-model="bindingPageContextSearch"
            size="small"
            :placeholder="t('propertyPanel.bindingVarEnum.searchPageVars')"
            clearable
          />
          <div class="sidebar-scroll page-var-list">
            <div
              v-for="row in bindingPageVariableRows"
              :key="row.name"
              class="page-var-item"
              @click="handleBindingPageVariableInsert(row)"
            >
              <el-icon class="node-icon icon-variable">
                <IconEpLink />
              </el-icon>
              <span class="node-label">{{ row.name }}</span>
              <span class="page-var-type">{{ row.type }}</span>
            </div>
          </div>
        </div>
        <div class="sidebar-section">
          <div class="sidebar-title">{{ t("propertyPanel.bindingDialog.pageComponents") }}</div>
          <ElInput
            v-model="bindingComponentSearch"
            size="small"
            :placeholder="t('propertyPanel.bindingDialog.searchComponents')"
            clearable
          />
          <div class="sidebar-scroll">
            <el-tree
              ref="bindingComponentTreeRef"
              :data="bindingPageComponentTree"
              node-key="id"
              :default-expand-all="true"
              :expand-on-click-node="false"
              :filter-node-method="filterBindingSidebarNode"
              @node-click="handleBindingComponentInsert"
            >
              <template #default="{ data }">
                <div class="tree-node" :class="`node-${data.type}`">
                  <el-icon class="node-icon icon-component">
                    <IconEpFolder v-if="data.type === 'group'" />
                    <IconEpGrid v-else />
                  </el-icon>
                  <span class="node-label" :class="{ 'is-group': data.type === 'group' }">
                    {{ data.label }}
                  </span>
                </div>
              </template>
            </el-tree>
          </div>
        </div>
      </div>
    </div>
    <template #footer>
      <el-button @click="bindingDialogVisible = false">{{
        t("propertyPanel.bindingDialog.cancel")
      }}</el-button>
      <el-button type="primary" @click="saveBinding">{{
        t("propertyPanel.bindingDialog.save")
      }}</el-button>
    </template>
  </el-dialog>

  <PropertyPanelBindingVarEnumDialog
    v-model="bindingVariableEnumVisible"
    v-model:binding-project-var-search="bindingProjectVarSearch"
    :binding-project-group-tree="bindingProjectGroupTree"
    :binding-project-variable-rows="bindingProjectVariableRows"
    :filter-binding-sidebar-node="filterBindingSidebarNode as any"
    :binding-enum-project-row-class="bindingEnumProjectRowClass as any"
    :can-confirm-insert="Boolean(bindingEnumSelectedProjectVar)"
    @project-group-select="handleBindingProjectGroupSelect as any"
    @project-row-click="handleBindingProjectRowClick as any"
    @project-row-dblclick="handleBindingProjectRowDblClick as any"
    @confirm-insert="confirmBindingEnumInsert as any"
    @cancel="bindingVariableEnumVisible = false"
  />
</template>

<style scoped>
.element-inspector {
  display: flex;
  flex-direction: column;
  gap: var(--designer-gap-md);
  min-width: 0;
}

.property-panel-root {
  display: flex;
  flex-direction: column;
  gap: var(--designer-gap-md);
  min-width: 0;
}

.panel-section,
.prop-section {
  box-sizing: border-box;
  min-width: 0;
  border: 1px solid var(--designer-border-color);
  border-radius: var(--designer-radius-md);
  background: var(--designer-shell-surface);
}

.axis-inline-group {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--designer-gap-xs);
  flex: 1;
  min-width: 0;
}

.axis-inline-item {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
}

.axis-inline-item :deep(.el-input) {
  width: 100%;
}

.axis-inline-tag {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  min-width: 18px;
  height: 18px;
  border-radius: 999px;
  background: var(--designer-group-surface);
  color: var(--designer-text-secondary);
  font-size: 11px;
  font-weight: 600;
}

.panel-section {
  padding: 8px 10px;
}

.panel-section-title {
  font-size: var(--designer-font-label);
  font-weight: 600;
  color: var(--designer-text-secondary);
}

.prop-sections,
.prop-section-body {
  display: flex;
  flex-direction: column;
  gap: var(--designer-gap-xs);
  min-width: 0;
}

.prop-section {
  overflow: hidden;
}

.prop-section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  min-height: 30px;
  padding: 0 10px;
  border: none;
  border-bottom: 1px solid var(--designer-border-soft);
  background: var(--designer-group-surface);
  cursor: pointer;
  transition: background-color 0.15s ease;
}

.prop-section-header:hover {
  background: var(--designer-hover-surface);
}

.prop-section-header.is-static {
  cursor: default;
}

.prop-section-header.is-static:hover {
  background: var(--designer-group-surface);
}

.prop-section-heading {
  display: inline-flex;
  align-items: center;
  gap: var(--designer-gap-xs);
}

.prop-section-title {
  font-size: var(--designer-font-label);
  font-weight: 600;
  color: var(--designer-text-secondary);
  letter-spacing: 0.02em;
}

.section-chevron {
  color: var(--designer-text-muted);
  transition: transform 0.15s ease;
}

.prop-section.is-collapsed .section-chevron {
  transform: rotate(-90deg);
}

.prop-section-body {
  padding: 4px 6px;
}

.prop-item {
  display: flex;
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
  min-height: 28px;
  padding: 2px 4px;
  border-radius: var(--designer-radius-sm);
  transition: background-color 0.15s ease;
}

.prop-item:hover {
  background: var(--designer-hover-surface);
}

.prop-item > :last-child:not(.prop-label) {
  flex: 1;
  min-width: 0;
}

.prop-item .prop-editor {
  flex: 1;
  min-width: 0;
}

.prop-item :deep(.el-input),
.prop-item :deep(.el-input-number),
.prop-item :deep(.el-select),
.prop-item :deep(.el-input-number),
.prop-item :deep(.el-color-picker) {
  width: 100%;
}

.prop-item :deep(.el-switch) {
  margin-left: auto;
}

.runtime-access-controls {
  display: flex;
  flex-direction: column;
  gap: var(--designer-gap-xs);
}

.runtime-access-row {
  display: grid;
  grid-template-columns: 76px minmax(0, 1fr);
  align-items: center;
  gap: var(--designer-gap-sm);
}

.runtime-access-row__label {
  color: var(--designer-text-secondary);
  font-size: var(--designer-font-label);
}

.runtime-access-row :deep(.el-select) {
  width: 100%;
}

.prop-label {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
  width: 88px;
  min-width: 72px;
  max-width: 88px;
  gap: 4px;
  font-size: var(--designer-font-label);
  color: var(--designer-text-regular);
}

.bind-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border: 1px solid var(--designer-primary-border);
  border-radius: var(--designer-radius-sm);
  background: var(--designer-primary-soft);
  cursor: pointer;
  opacity: 1;
  color: var(--designer-primary-text);
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.35);
  transition:
    opacity 0.15s ease,
    border-color 0.15s ease,
    color 0.15s ease,
    background-color 0.15s ease;
  flex-shrink: 0;
}

.bind-btn:hover {
  border-color: var(--designer-primary-border);
  color: var(--designer-primary-text);
  background: var(--designer-active-surface);
}

.bind-btn.is-active {
  opacity: 1;
  border-color: var(--designer-primary-border);
  color: var(--designer-shell-surface);
  background: var(--designer-primary);
  box-shadow: none;
}

.bind-icon {
  width: 13px;
  height: 13px;
}

.region-config-section {
  overflow: hidden;
}

.section-icon {
  width: 14px;
  height: 14px;
  color: var(--designer-text-muted);
}

.region-config-body {
  gap: 8px;
  padding: 8px;
}

.region-config-block {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}

.region-config-label {
  font-size: var(--designer-font-label);
  font-weight: 600;
  color: var(--designer-text-secondary);
}

.region-preset-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 6px;
}

.region-preset-card {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 5px;
  min-height: 86px;
  padding: 5px;
  border: 1px solid var(--designer-border-color);
  border-radius: var(--designer-radius-sm);
  background: var(--designer-shell-surface);
  color: var(--designer-text-regular);
  cursor: pointer;
  text-align: left;
  transition:
    border-color 0.15s ease,
    background-color 0.15s ease,
    box-shadow 0.15s ease;
}

.region-preset-card:hover {
  border-color: var(--designer-primary-border);
  background: var(--designer-hover-surface);
}

.region-preset-card.is-active {
  border-color: var(--designer-primary-border);
  background: var(--designer-primary-soft);
  box-shadow: inset 0 0 0 1px var(--designer-primary-border);
}

.region-preset-preview {
  display: grid;
  grid-template-columns: 18px 1fr;
  grid-template-rows: 12px 20px 12px;
  gap: 2px;
  width: 100%;
  height: 42px;
  padding: 3px;
  border: 1px solid var(--designer-border-soft);
  border-radius: 4px;
  background: var(--designer-group-surface);
  box-sizing: border-box;
}

.region-preview-part {
  display: block;
  min-width: 0;
  min-height: 0;
  border-radius: 2px;
}

.region-preview-header {
  grid-area: header;
  background: #dbeafe;
}

.region-preview-aside {
  grid-area: aside;
  background: #d1fae5;
}

.region-preview-main {
  grid-area: main;
  background: #fef3c7;
}

.region-preview-footer {
  grid-area: footer;
  background: #e5e7eb;
}

.region-preset-copy {
  display: flex;
  flex-direction: column;
  align-items: center;
  min-width: 0;
  gap: 2px;
}

.region-preset-title {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: var(--designer-font-sm);
  font-weight: 600;
  color: var(--designer-text-regular);
}

.region-preset-desc {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: var(--designer-font-label);
  color: var(--designer-text-secondary);
}

.region-toggle-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 6px;
}

.region-toggle-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
  min-width: 0;
  min-height: 30px;
  padding: 4px 6px;
  border: 1px solid var(--designer-border-color);
  border-radius: var(--designer-radius-sm);
  background: var(--designer-shell-surface);
  color: var(--designer-text-regular);
  cursor: pointer;
  font-size: var(--designer-font-label);
  transition:
    border-color 0.15s ease,
    background-color 0.15s ease;
}

.region-toggle-card:hover {
  border-color: var(--designer-primary-border);
  background: var(--designer-hover-surface);
}

.region-toggle-card.is-active {
  border-color: var(--designer-primary-border);
  background: var(--designer-primary-soft);
}

.region-size-row {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 6px;
  border: 1px solid var(--designer-border-soft);
  border-radius: var(--designer-radius-sm);
  background: var(--designer-group-surface);
}

.region-size-row-label {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
  min-width: 0;
  font-size: var(--designer-font-label);
  color: var(--designer-text-regular);
}

.region-prop-row .prop-label {
  margin-bottom: 4px;
  max-width: none;
  width: 100%;
}

.region-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.region-size-text {
  font-size: var(--designer-font-label);
  color: var(--designer-text-secondary);
}

.region-size-limit {
  font-size: var(--designer-font-label);
  color: var(--designer-text-secondary);
}

.region-prop-controls {
  display: flex;
  align-items: center;
  gap: 6px;
}

.region-size-input {
  flex: 1;
}

.region-unit-select {
  width: 64px;
}

.region-input-placeholder {
  flex: 1;
}

.region-toggle {
  margin-left: auto;
}

.config-toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 12px 16px;
  align-items: center;
  margin-bottom: 12px;
  padding: 8px 10px;
  border: 1px solid var(--designer-border-color);
  border-radius: var(--designer-radius-md);
  background: var(--designer-group-surface);
}

.config-toolbar-item {
  display: flex;
  align-items: center;
  gap: 6px;
}

.config-toolbar-actions {
  margin-left: auto;
}

.config-label {
  font-size: var(--designer-font-label);
  color: var(--designer-text-secondary);
}

.config-select {
  width: 180px;
}

.config-body {
  display: flex;
  gap: 12px;
  align-items: stretch;
}

.config-editor {
  min-height: 520px;
  flex: 1;
}

.config-assets {
  width: 240px;
  border-left: 1px solid var(--designer-border-color);
  padding-left: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.config-sidebar {
  width: 240px;
  border-left: 1px solid var(--designer-border-color);
  padding-left: 12px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  height: 520px;
}

.config-assets-header {
  font-size: var(--designer-font-label);
  font-weight: 600;
  color: var(--designer-text-secondary);
}

.config-assets-body {
  flex: 1;
  min-height: 0;
  border: 1px solid var(--designer-border-color);
  border-radius: var(--designer-radius-md);
  padding: 6px;
  background: var(--designer-shell-surface);
}

.editor-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 10px 16px;
  padding: 10px 12px;
  border: 1px solid var(--designer-border-color);
  border-radius: var(--designer-radius-md);
  background: var(--designer-group-surface);
  margin-bottom: 10px;
  align-items: center;
}

.meta-title {
  font-size: var(--designer-font-lg);
  font-weight: 600;
  color: var(--designer-text-primary);
}

.meta-desc {
  font-size: var(--designer-font-label);
  color: var(--designer-text-secondary);
}

.meta-actions {
  margin-left: auto;
  display: flex;
  align-items: center;
}

.icon-button {
  background: var(--designer-primary-soft);
  border: none;
  color: var(--designer-primary-text);
}

.icon-button:hover {
  background: var(--designer-hover-surface);
}

.editor-body {
  display: flex;
  gap: 12px;
  flex: 1;
  align-items: stretch;
  height: 520px;
}

.editor-main {
  flex: 1;
  min-width: 0;
}

.editor-sidebar {
  width: 220px;
  height: 520px;
  border-left: 1px solid var(--designer-border-color);
  padding-left: 12px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.sidebar-section {
  display: flex;
  flex-direction: column;
  gap: 6px;
  flex: 1;
  min-height: 0;
}

.sidebar-title {
  font-size: var(--designer-font-label);
  font-weight: 600;
  color: var(--designer-text-secondary);
}

.sidebar-scroll {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding-right: 4px;
}

.tree-node {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  width: 100%;
  padding: 6px 8px;
  border-radius: var(--designer-radius-md);
  transition: background-color 0.2s;
}

.tree-node:hover {
  background: var(--designer-hover-surface);
}

.page-var-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.page-var-item {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 28px;
  padding: 4px 6px;
  border-radius: var(--designer-radius-sm);
  cursor: pointer;
}

.page-var-item:hover {
  background: var(--designer-hover-surface);
}

.page-var-type {
  margin-left: auto;
  color: var(--designer-text-muted);
  font-size: var(--designer-font-sm);
}

.node-icon {
  color: var(--designer-text-muted);
  flex-shrink: 0;
}

.node-item .node-icon.icon-custom {
  color: #10b981;
}

.node-variable .node-icon.icon-variable {
  color: #0ea5e9;
}

.node-label {
  font-size: var(--designer-font-md);
  color: var(--designer-text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.node-label.is-group {
  font-weight: 600;
}

.node-component .node-icon.icon-component {
  color: #6366f1;
}

.enum-layout {
  display: flex;
  gap: 12px;
  padding-top: 8px;
}

.enum-left {
  width: 200px;
  border-right: 1px solid var(--designer-border-color);
  padding-right: 8px;
  max-height: 360px;
  overflow: auto;
}

.enum-right {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.enum-right :deep(.el-table__row.is-selected) {
  background: var(--designer-primary-soft);
}
</style>
