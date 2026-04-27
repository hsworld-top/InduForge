<!--
  NodeRenderer - 节点渲染器
  递归渲染组件树，支持 DOM 组件与 Canvas 图形，处理绑定、事件、拖拽
-->
<script setup lang="ts">
import { ElMessage } from "element-plus";
import { storeToRefs } from "pinia";
import { computed, inject, nextTick, ref, watch } from "vue";
import {
  getCustomRenderer,
  getDisplayContent,
  getFlexDirection,
  getRenderTag,
  isChildResizable as isChildResizableByDescriptor,
  isFlexContainer,
  isNodeDesignerMovable,
  isTableLikeType,
  usesLegacyFlexDirectionProps,
} from "@/editor-core/descriptors/registry";
import { componentRegistry, createSelectableElement } from "@/editor-core";
import { normalizeEventDefinitions } from "@/editor-core/registry/component-events";
import { datacenterApi } from "@/services";
import { useEditorStore } from "@/stores/editor-store";
import EChart from "./renderers/EChart.vue";
import { useBuildRefInfo } from "./composables/use-build-ref-info";
import {
  clearDropTarget,
  endDrag,
  startDrag,
  updateDropTarget,
  useDragState,
} from "./composables/use-drag-state";
import { useNodeContent } from "./composables/use-node-content";
import { useNodeDrop } from "./composables/use-node-drop";
import { useNodeInteraction } from "./composables/use-node-interaction";
import { useNodePointer } from "./composables/use-node-pointer";
import {
  buildExpressionContext,
  resolveExpressionValue,
  useNodeProps,
} from "./composables/use-node-props";
import {
  createApplyMenuDslConfig,
  resolveMenuConfigFromContent,
  useNodeRendererDerivations,
} from "./composables/use-node-renderer-derivations";
import { useNodeRendererPreviewRef } from "./composables/use-node-renderer-preview-ref";
import { useNodeRendererTypeFlags } from "./composables/use-node-renderer-type-flags";
import { useNodeResize } from "./composables/use-node-resize";
import { createNodeStyleHelpers } from "./composables/use-node-style";
import { usePreview } from "./composables/use-preview";
import { canvasSnapEnabledKey, canvasZoomKey, runtimeAccessContextKey } from "./injection-keys";
import { createDragDropManager } from "./interaction/DragDropManager";
import { buildDesignerNodeDomId } from "./style-config-css";

interface NodeRendererProps {
  nodeId: string;
  isRoot?: boolean;
  readonly?: boolean;
}

type NodeLike = Record<string, any>;
type TabsPaneLike = Record<string, any>;
type EventListenerMap = Record<string, any>;
type StyleObjectLike = Record<string, any>;
type ShowContextMenuLike = any;

const props = withDefaults(defineProps<NodeRendererProps>(), {
  isRoot: false,
  readonly: false,
});

const STYLE_COMMENT_RE = /\/\*[\s\S]*?\*\//g;

const editorStore = useEditorStore();
const {
  doc,
  selection,
  docVersion,
  selectionVersion,
  history,
  projectId,
  projectVariables,
  globalScripts,
  currentPage,
  error,
} = storeToRefs(editorStore);
const showContextMenu = inject<ShowContextMenuLike>("showContextMenu", null);
const runtimeAccessContext = inject(runtimeAccessContextKey, null);
const nodeStyleHelpers = createNodeStyleHelpers({ doc, currentPage, props } as any) as any;
const resolveLayoutStyle = nodeStyleHelpers.resolveLayoutStyle;

const node = computed<NodeLike | null>(() => {
  void docVersion.value;
  return doc.value?.getNode(props.nodeId) || null;
});

const detailConfigText = computed(() => {
  void docVersion.value;
  return String(node.value?.detailConfig || "").trim();
});

let applyMenuDslConfig: any = () => {};

// nodeRef 和 contentRef 需要先声明，因为会被 useNodeInteraction 等 composable 使用
const nodeRef = ref<any>(null);
const contentRef = ref<any>(null);
function setNodeRef(el: any): void {
  nodeRef.value = el?.$el || el;
}

/**
 * 处理节点选中（Shift/Meta/Ctrl 多选）
 * @param {MouseEvent} event - 鼠标事件
 */
function handleSelect(event: any): void {
  if (!node.value || !selection.value) return;
  if (node.value.locked) return;
  const element = createSelectableElement("node", node.value.id);
  if (event.shiftKey) {
    selection.value.selectRange(element);
    return;
  }
  if (event.metaKey || event.ctrlKey) {
    selection.value.toggleSelect(element);
    return;
  }
  selection.value.select(element);
}

// 使用 usePreview composable（需要在 useNodeInteraction 之前，因为 runPreviewScript 需要传入）
// buildRefInfo 由下方 useBuildRefInfo 赋值，此处先声明以便 usePreview 闭包引用
let buildRefInfo: any = null;
const previewRuntime: any = usePreview({
  node: node as any,
  doc: doc as any,
  currentPage: currentPage as any,
  projectVariables: projectVariables as any,
  projectId: projectId as any,
  docVersion: docVersion as any,
  globalScripts: globalScripts as any,
  datacenterApi: datacenterApi as any,
  buildRefInfo: () => buildRefInfo?.(),
  readonly: computed(() => props.readonly),
  detailConfigText,
  resolveMenuConfigFromContent: resolveMenuConfigFromContent as any,
  applyMenuDslConfig: (config: any) => applyMenuDslConfig(config),
} as any);
const { runPreviewScript, isRunningDetailConfig: isRunningDetailConfigFn } = previewRuntime;

const isRuntimeOperable = computed(() => {
  void docVersion.value;
  if (!node.value || !runtimeAccessContext) return true;
  return runtimeAccessContext.isNodeOperable(node.value as any);
});

function runOperablePreviewScript(eventName: string, payload: any): unknown {
  if (!isRuntimeOperable.value) return undefined;
  return runPreviewScript(eventName, payload);
}

// 使用 useNodeInteraction composable
const nodeInteraction: any = useNodeInteraction({
  node: node as any,
  doc: doc as any,
  selection: selection as any,
  selectionVersion: selectionVersion as any,
  readonly: computed(() => props.readonly),
  isRoot: computed(() => props.isRoot),
  isChildResizableByDescriptor: isChildResizableByDescriptor as any,
  nodeRef,
  createSelectableElement: createSelectableElement as any,
  runPreviewScript: runOperablePreviewScript as any,
  handleSelect,
  showContextMenu,
} as any);
const {
  handleClick,
  handleDoubleClick,
  handleContextMenu,
  showResizeHandlesBase,
  visibleResizeHandles: visibleResizeHandlesFromComposable,
  isContainer: isContainerFromComposable,
  isRegionContainer: isRegionContainerFromComposable,
  isElColInRow: isElColInRowFromComposable,
  isChildInElCol: isChildInElColFromComposable,
  getRegionResizeConfig: getRegionResizeConfigFromComposable,
} = nodeInteraction;
const canvasZoom = inject<any>(canvasZoomKey, ref(1));
const canvasSnapEnabled = inject<any>(canvasSnapEnabledKey, ref(true));
const dragState = useDragState();
function notifyInsertFailure(fallbackMessage?: string): void {
  const message = error.value || fallbackMessage || "插入失败：当前不可编辑";
  ElMessage.warning({ message } as any);
}

// 使用 useNodeProps composable（不传 activeTabName，避免循环依赖）
const nodeProps: any = useNodeProps({
  node: node as any,
  doc: doc as any,
  currentPage: currentPage as any,
  projectVariables: projectVariables as any,
  docVersion: docVersion as any,
  readonly: computed(() => props.readonly),
} as any);
const {
  resolvedNodeProps,
  resolvedProps: resolvedPropsBase,
  filteredProps: filteredPropsFromComposable,
} = nodeProps;

// 使用 useNodeContent composable（传入 resolvedNodeProps）
const nodeContent: any = useNodeContent({
  node: node as any,
  resolvedNodeProps: resolvedNodeProps as any,
  docVersion: docVersion as any,
} as any);
const {
  selectOptions,
  radioOptions,
  checkboxOptions,
  dropdownItems,
  tabsList,
  collapseItems: collapseItemsFromContent,
  normalizeMenuItems,
  activeTabName,
  activeCollapseName,
  tableRenderVersion,
  applyTabsModelValueToProps,
  applyCollapseModelValueToProps,
} = nodeContent;
const selectOptionsList = computed<any[]>(() => (selectOptions.value || []) as any[]);
const radioOptionsList = computed<any[]>(() => (radioOptions.value || []) as any[]);
const checkboxOptionsList = computed<any[]>(() => (checkboxOptions.value || []) as any[]);
const dropdownItemsList = computed<any[]>(() => (dropdownItems.value || []) as any[]);
const tabsListItems = computed<any[]>(() => (tabsList.value || []) as any[]);
const collapseItemsListFromContent = computed<any[]>(
  () => (collapseItemsFromContent.value || []) as any[],
);
const {
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
} = useNodeRendererTypeFlags(
  node as any,
  {
    readonly: computed(() => props.readonly),
    isRegionContainer: isRegionContainerFromComposable,
  } as any,
);

const { buildRefInfo: buildRefInfoImpl, applyPreviewPatch } = useBuildRefInfo({
  node: node as any,
  nodeRef,
  contentRef,
  editorStore: editorStore as any,
  readonly: computed(() => props.readonly),
  tableRenderVersion,
  docVersion,
  isRunningDetailConfigFn: isRunningDetailConfigFn as any,
} as any);
buildRefInfo = buildRefInfoImpl;

useNodeRendererPreviewRef({
  readonly: computed(() => props.readonly),
  node: node as any,
  currentPage: currentPage as any,
  buildRefInfo: () => buildRefInfo?.(),
} as any);

applyMenuDslConfig = createApplyMenuDslConfig({
  node: node as any,
  readonly: computed(() => props.readonly),
  applyPreviewPatch: applyPreviewPatch as any,
  editorStore: editorStore as any,
  normalizeMenuItems: normalizeMenuItems as any,
} as any);

// 应用 Tabs/Collapse modelValue 处理到 resolvedProps（通过 useNodeContent 的辅助函数）
const resolvedProps = computed<Record<string, any>>(() => {
  const base = resolvedPropsBase.value;
  if (isTableLikeType(node.value?.type)) {
    void tableRenderVersion.value;
  }
  const withTabs = applyTabsModelValueToProps(base);
  return applyCollapseModelValueToProps(withTabs);
});

const dragDropManager: any = createDragDropManager();

function resolveFlexDirection(type: any, element: any): string {
  const descriptorDirection = getFlexDirection(type);
  if (descriptorDirection) {
    return descriptorDirection;
  }
  if (usesLegacyFlexDirectionProps(type)) {
    return node.value?.props?.direction || "column";
  }
  return dragDropManager.getContainerDirection((element || undefined) as any);
}

const isMovable = computed(() => {
  if (!node.value || props.isRoot || node.value.locked) return false;
  return isNodeDesignerMovable(node.value.type);
});

const nodeDrop: any = useNodeDrop({
  node: node as any,
  doc: doc as any,
  dragState,
  dragDropManager: dragDropManager as any,
  isContainer: isContainerFromComposable,
  resolveFlexDirection,
  isFlexContainer: isFlexContainer as any,
  readonly: computed(() => props.readonly),
  editorStore: editorStore as any,
  canvasZoom,
  endDrag: endDrag as any,
  notifyInsertFailure,
  activeTabName,
  tabsList,
  activeCollapseName,
  collapseItems: collapseItemsListFromContent,
} as any);
const {
  handleDragOver,
  handleDrop,
  showInsertLine,
  insertLineStyle,
  genericInsertLineBox,
  rowInsertInfo,
  layoutInsertInfo,
  insertLineBox,
  isDragOver,
  suppressDropByAlt,
  canAcceptChild,
} = nodeDrop;
const nodeResize: any = useNodeResize({
  node: node as any,
  doc: doc as any,
  nodeRef,
  readonly: computed(() => props.readonly),
  isMovable,
  isElColInRow: isElColInRowFromComposable,
  isChildInElCol: isChildInElColFromComposable,
  selection: selection as any,
  canvasZoom,
  history: history as any,
  editorStore: editorStore as any,
  isChildResizableByDescriptor: isChildResizableByDescriptor as any,
  getRegionResizeConfig: getRegionResizeConfigFromComposable,
} as any);
const { handleResizePointerDown } = nodeResize;
const isDropActive = computed<boolean>(() => {
  if (suppressDropByAlt.value) return false;
  if (!node.value) return isDragOver.value;
  return isDragOver.value || dragState.targetContainerId === node.value.id;
});

const {
  menuItems,
  tableColumns,
  bigTableColumns,
  timelineItems,
  stepsItems,
  collapseItems,
  carouselItems,
  dropdownLabel,
  getTabChildIds,
  getCollapseChildIds,
  regionHintText,
  useComponentWrapper,
  renderKey,
  nodeClass,
} = useNodeRendererDerivations({
  node: node as any,
  detailConfigText,
  docVersion: docVersion as any,
  tableRenderVersion,
  resolvedNodeProps: resolvedNodeProps as any,
  selectionVersion: selectionVersion as any,
  selection: selection as any,
  doc: doc as any,
  isContainer: isContainerFromComposable,
  isMovable,
  isDropActive,
  activeTabName,
  tabsList,
  activeCollapseName,
  collapseItems: collapseItemsListFromContent,
  props,
} as any);
const menuItemsList = computed<any[]>(() => (menuItems.value || []) as any[]);
const tableColumnsList = computed<any[]>(() => (tableColumns.value || []) as any[]);
const bigTableColumnsList = computed<any[]>(() => (bigTableColumns.value || []) as any[]);
const timelineItemsList = computed<any[]>(() => (timelineItems.value || []) as any[]);
const collapseItemsList = computed<any[]>(() => (collapseItems.value || []) as any[]);
const stepsItemsList = computed<any[]>(() => (stepsItems.value || []) as any[]);
const carouselItemsList = computed<any[]>(() => (carouselItems.value || []) as any[]);

const filteredProps = computed<Record<string, any>>(() => {
  const base = { ...(filteredPropsFromComposable.value || {}) };
  if (props.readonly && !isRuntimeOperable.value) {
    base.disabled = true;
    base.readonly = true;
  }
  return base;
});

const tabHeaderWidth = ref(0);

/**
 * 同步 Tabs 头部宽度（用于左右布局）
 */
function syncTabsHeaderWidth(): void {
  if (!node.value || !isTabsType.value) return;
  const tabPosition =
    resolvedNodeProps.value?.tabPosition || node.value?.props?.tabPosition || "top";
  if (tabPosition !== "left" && tabPosition !== "right") return;
  const rootEl = nodeRef.value;
  if (!rootEl) return;
  const headerSelector =
    tabPosition === "left"
      ? ".el-tabs__header-vertical.is-left"
      : tabPosition === "right"
        ? ".el-tabs__header-vertical.is-right"
        : ".el-tabs__header";
  const headerEl = rootEl.querySelector?.(headerSelector);
  if (!headerEl) return;
  const rect = headerEl.getBoundingClientRect?.();
  if (!rect) return;
  const nextWidth = Math.max(0, Math.round(rect.width));
  if (!nextWidth) return;
  if (tabHeaderWidth.value !== nextWidth) {
    tabHeaderWidth.value = nextWidth;
  }
  rootEl.style.setProperty("--tabs-vertical-width", `${nextWidth}px`);
}

watch(
  () => [node.value?.type, resolvedNodeProps.value?.tabPosition, tabsList.value?.length],
  async () => {
    await nextTick();
    syncTabsHeaderWidth();
  },
  { immediate: true },
);

/**
 * 处理 Tabs 点击事件
 * @param {Object} pane - Tab 面板
 */
function handleTabsClick(pane: TabsPaneLike): void {
  if (!pane) return;
  const name = pane?.props?.name ?? pane?.name ?? pane?.paneName ?? pane?.label ?? "";
  if (name) {
    activeTabName.value = String(name);
  }
}

/**
 * 处理 Tabs 切换事件
 * @param {string} name - 激活名称
 */
function handleTabsChange(name: string | number): void {
  if (!name) return;
  activeTabName.value = String(name);
}

/**
 * 处理 Tabs 删除事件
 * @param {string} name - Tab 名称
 */
function resolveTabNameValue(input: any): any {
  if (input && typeof input === "object") {
    return input?.props?.name ?? input?.name ?? input?.paneName ?? input?.label ?? "";
  }
  return input ?? "";
}

function handleTabsRemove(name: any): void {
  if (!isTabsType.value) return;
  const resolvedName = resolveTabNameValue(name);
  if (!resolvedName) return;
  const tabs = Array.isArray(node.value?.props?.tabs) ? [...node.value.props.tabs] : [];
  if (!tabs.length) return;
  const normalizedName = String(resolvedName);
  const updateTabsProps = (patch: Record<string, any>): void => {
    if (!node.value || !patch || typeof patch !== "object") return;
    editorStore.updateNode(node.value.id, {
      props: { ...(node.value.props || {}), ...patch },
    });
  };
  const removeIndex = tabs.findIndex((item) => {
    const tabName = item?.name ?? item?.label ?? "";
    return String(tabName) === normalizedName;
  });
  let targetIndex = removeIndex;
  if (targetIndex < 0) {
    const numericIndex = Number(normalizedName);
    if (Number.isFinite(numericIndex)) {
      const candidate = Math.trunc(numericIndex);
      if (candidate >= 0 && candidate < tabs.length) {
        targetIndex = candidate;
      }
    }
  }
  if (targetIndex < 0) return;
  tabs.splice(targetIndex, 1);
  updateTabsProps({ tabs });

  const childIds = Array.isArray(node.value?.children) ? [...node.value.children] : [];
  childIds.forEach((childId) => {
    const childNode = doc.value?.getNode?.(childId);
    if (!childNode) return;
    const tabKey = childNode.props?.tabKey ?? "";
    if (String(tabKey) === normalizedName) {
      editorStore.removeNode(childId);
    }
  });

  const nextTab = tabs[targetIndex] || tabs[targetIndex - 1] || tabs[0] || null;
  const nextName = nextTab?.name ?? nextTab?.label ?? "";
  updateTabsProps({ activeName: nextName ? String(nextName) : "" });
}

/**
 * 处理 Tabs 编辑事件
 * @param {string} name - Tab 名称
 * @param {string} action - edit 动作
 */
function handleTabsEdit(name: any, action: string): void {
  if (action !== "remove") return;
  handleTabsRemove(name);
}
function isActiveTab(tab: any): boolean {
  if (!tab) return false;
  const name = tab.name ?? tab.label ?? "";
  return String(name) === activeTabName.value;
}

function resolveCollapseItemKey(item: any): string {
  if (!item) return "";
  const raw = item.name ?? item.title ?? item.label ?? "";
  return String(raw || "").trim();
}

function isActiveCollapseItem(item: any): boolean {
  const key = resolveCollapseItemKey(item);
  if (!key) return false;
  return key === String(activeCollapseName.value || "").trim();
}

function handleCollapseChange(value: unknown): void {
  if (Array.isArray(value)) {
    const firstKey = String(value[0] ?? "").trim();
    if (firstKey) {
      activeCollapseName.value = firstKey;
      return;
    }
  }
  const normalized = String(value ?? "").trim();
  if (normalized) {
    activeCollapseName.value = normalized;
  }
}

function handleCollapseHeaderClick(item: any): void {
  const key = resolveCollapseItemKey(item);
  if (!key) return;
  activeCollapseName.value = key;
}

const isContainer = isContainerFromComposable;

const isNodeVisible = computed(() => {
  void docVersion.value;
  if (!node.value) return false;
  if (node.value.hidden) return false;
  if (runtimeAccessContext && !runtimeAccessContext.isNodeVisible(node.value as any)) return false;
  const visibleConfig = node.value.conditions?.visible;
  if (typeof visibleConfig === "boolean") return visibleConfig;
  if (typeof visibleConfig !== "string" || !visibleConfig.trim()) return true;
  const context = buildExpressionContext(
    (resolvedNodeProps.value || {}) as any,
    {
      doc: doc as any,
      currentPage: currentPage as any,
      projectVariables: projectVariables as any,
    } as any,
  );
  const value = resolveExpressionValue(visibleConfig, context, true);
  return Boolean(value);
});

const hasChildren = computed(() => {
  void docVersion.value;
  if (!node.value || !doc.value) return false;
  if (typeof (doc.value as any).getChildren === "function") {
    return (doc.value as any).getChildren(node.value.id).length > 0;
  }
  return (node.value.children || []).length > 0;
});

const nodePointer = useNodePointer({
  node: node as any,
  doc: doc as any,
  nodeRef,
  readonly: computed(() => props.readonly),
  isMovable,
  isContainer,
  selection: selection as any,
  canvasZoom,
  enableSnap: canvasSnapEnabled,
  history: history as any,
  editorStore: editorStore as any,
  currentPage: currentPage as any,
  startDrag: startDrag as any,
  endDrag: endDrag as any,
  updateDropTarget: updateDropTarget as any,
  clearDropTarget: clearDropTarget as any,
  dragDropManager: dragDropManager as any,
  canAcceptChild,
  showInsertLine,
  insertLineStyle,
  genericInsertLineBox,
  rowInsertInfo,
  layoutInsertInfo,
  activeTabName,
  tabsList,
  activeCollapseName,
  collapseItems: collapseItemsListFromContent,
  resolveFlexDirection,
} as any);
const { handlePointerDown } = nodePointer;

const visibleResizeHandles = visibleResizeHandlesFromComposable;

// showResizeHandles 需要额外检查 selection，所以保留一个包装 computed
const showResizeHandles = computed(() => {
  // 依赖 selectionVersion，保证选中变化时可重新计算手柄显示
  void selectionVersion.value;
  const base = showResizeHandlesBase.value;
  if (!base) return false;
  const currentNodeId = node.value?.id;
  const currentSelection = selection.value;
  if (!currentNodeId || !currentSelection) return false;
  // 额外检查：只有选中时才显示（与 nodeClass 保持一致的调用方式）
  return Boolean(currentSelection.isSelected(currentNodeId));
});

const isRegionContainer = isRegionContainerFromComposable;

const renderTag = computed<any>(() => {
  if (!node.value) return "div";
  const type = node.value.type;
  // 使用 getRenderTag 统一处理（支持函数类型 renderTag，如 Text 组件）
  const tag = getRenderTag(type, node.value, resolvedNodeProps.value);
  if (tag && tag !== "div") return tag;
  // EChart 特殊处理（需要返回 Vue 组件实例，无法在描述符中直接表示）
  if (type === "EChart") return EChart;
  // 降级：未注册 descriptor 的组件返回 div
  return tag || "div";
});

/** 复杂组件自定义渲染器（由 descriptor.customRenderer 指定） */
const customRendererComponent = computed<any>(() => {
  if (!node.value) return null;
  return getCustomRenderer(node.value.type) ?? null;
});

const outerTag = computed<any>(() => {
  return useComponentWrapper.value ? renderTag.value : "div";
});

const displayContent = computed<any>(() => {
  void docVersion.value;
  if (!node.value) return null;
  const resolvedPropsValue = resolvedNodeProps.value || {};

  // 优先从 descriptor 读取（新架构组件）
  const descriptorContent = getDisplayContent(node.value.type, node.value, resolvedPropsValue);
  if (descriptorContent !== null) {
    return descriptorContent;
  }

  // 向后兼容：未注册 displayContent 的组件返回 null（不再有 fallback 硬编码）
  return null;
});

const layoutStyle = computed<Record<string, any>>(() => {
  void docVersion.value;
  if (!node.value) return {};
  return resolveLayoutStyle(node.value as any, props.isRoot);
});

// 使用 use-node-style composable 创建 contentStyle
const contentStyle = nodeStyleHelpers.createContentStyle(
  node as any,
  doc as any,
  docVersion as any,
  resolvedNodeProps as any,
  computed(() => isContainer.value),
  isMovable,
  layoutStyle,
  props,
);

// 使用 use-node-style composable 创建 wrapperStyle
const wrapperStyle = nodeStyleHelpers.createWrapperStyle(
  node as any,
  doc as any,
  layoutStyle,
  isMovable,
  computed(() => isContainer.value),
);

// 使用 use-node-style composable 创建 wrapperComponentStyle
const wrapperComponentStyle = nodeStyleHelpers.createWrapperComponentStyle(
  node as any,
  doc as any,
  useComponentWrapper,
  wrapperStyle,
  contentStyle,
  resolvedProps,
  props,
);

const styleConfigText = computed<string>(() => {
  void docVersion.value;
  return String(node.value?.styleConfig || "").trim();
});
const hasStyleConfigSelector = computed(() => styleConfigText.value.includes("{"));
const nodeDomId = computed<string>(() => {
  if (!node.value) return "";
  if (!node.value.id) return "";
  return buildDesignerNodeDomId(node.value.id);
});
const selectorStyleConfig = computed<string>(() =>
  hasStyleConfigSelector.value ? styleConfigText.value : "",
);
nodeStyleHelpers.createStyleElementSync(node as any, selectorStyleConfig as any);
const inlineStyleConfig = computed<string>(() => {
  if (hasStyleConfigSelector.value) return "";
  return styleConfigText.value ? styleConfigText.value : "";
});
const inlineStyleConfigObject = computed<StyleObjectLike>(() => {
  const raw = inlineStyleConfig.value;
  if (!raw) return {};
  const stripped = raw.replace(STYLE_COMMENT_RE, "");
  const entries = stripped
    .split(";")
    .map((item) => item.trim())
    .filter(Boolean);
  const result: StyleObjectLike = {};
  entries.forEach((item) => {
    const [key, ...rest] = item.split(":");
    if (!key || rest.length === 0) return;
    const value = rest.join(":").trim();
    const prop = key.trim();
    if (!prop || !value) return;
    result[prop] = value;
  });
  return result;
});
const resolvedInlineStyleConfigObject = computed<StyleObjectLike>(() => {
  return inlineStyleConfigObject.value;
});

/**
 * 外层样式（处理组件包装模式）
 */
const outerStyle = computed<any>(() => {
  if (useComponentWrapper.value) {
    const inlineStyle = resolvedInlineStyleConfigObject.value;
    if (!inlineStyle || Object.keys(inlineStyle).length === 0) {
      return wrapperComponentStyle.value;
    }
    return [wrapperComponentStyle.value, inlineStyle];
  }
  return wrapperStyle.value;
});

/**
 * 内容样式（非包装组件时附加样式配置）
 */
const contentStyleWithConfig = computed<any>(() => {
  if (useComponentWrapper.value) return contentStyle.value;
  const inlineStyle = resolvedInlineStyleConfigObject.value;
  if (!inlineStyle || Object.keys(inlineStyle).length === 0) {
    return contentStyle.value;
  }
  return [contentStyle.value, inlineStyle];
});

const childNodeIds = computed<string[]>(() => {
  void docVersion.value;
  if (!node.value || isTabsType.value || isCollapseType.value) return [];
  const children = Array.isArray(node.value.children) ? node.value.children : [];
  return [...children];
});

const modelValueTypes = new Set([
  "Input",
  "InputNumber",
  "Select",
  "Switch",
  "Radio",
  "Checkbox",
  "Cascader",
  "Transfer",
  "Slider",
  "Rate",
  "ColorPicker",
]);
const supportsModelValue = computed<boolean>(() => modelValueTypes.has(node.value?.type));

/**
 * 更新组件的 modelValue
 * @param {any} value - 新值
 */
function handleModelValueUpdate(value: any): void {
  if (!node.value) return;
  if (props.readonly && !isRuntimeOperable.value) return;
  if (props.readonly) {
    applyPreviewPatch({ props: { modelValue: value } });
    return;
  }
  editorStore.updateNode(node.value.id, {
    props: { ...(node.value.props || {}), modelValue: value },
  });
}

const componentEventListeners = computed<EventListenerMap>(() => {
  if (!node.value) return {};
  const listeners: EventListenerMap = {};
  if (supportsModelValue.value) {
    listeners["update:modelValue"] = handleModelValueUpdate;
  }
  if (!props.readonly) return listeners;
  const manifest = componentRegistry.get(node.value.type);
  const definitions = normalizeEventDefinitions(manifest?.events || []);
  definitions.forEach((eventItem) => {
    if (!eventItem?.name || eventItem.name === "click") return;
    listeners[eventItem.name] = (...args: any[]) => {
      const payload = args.length > 1 ? args : args[0];
      void runOperablePreviewScript(eventItem.name, payload);
    };
  });
  return listeners;
});

/**
 * 编辑态组件事件监听
 */
const designEventListeners = computed<EventListenerMap>(() => {
  if (props.readonly || !node.value) return {};
  if (isTabsType.value) {
    return {
      "tab-click": handleTabsClick,
      "tab-change": handleTabsChange,
      "update:modelValue": handleTabsChange,
      "tab-remove": handleTabsRemove,
      edit: handleTabsEdit,
    };
  }
  if (isCollapseType.value) {
    return {
      change: handleCollapseChange,
      "update:modelValue": handleCollapseChange,
    };
  }
  return {};
});

/**
 * 合并事件监听
 */
const mergedEventListeners = computed<EventListenerMap>(() => {
  return { ...componentEventListeners.value, ...designEventListeners.value };
});

/**
 * 处理拖拽离开
 */
function handleDragLeave(): void {
  if (props.readonly) return;
  isDragOver.value = false;
  showInsertLine.value = false;
  insertLineStyle.value = null;
  genericInsertLineBox.value = null;
  rowInsertInfo.value = null;
  layoutInsertInfo.value = null;
}
</script>

<template>
  <component
    :is="outerTag"
    v-if="node && isNodeVisible"
    v-bind="useComponentWrapper ? filteredProps : {}"
    :id="useComponentWrapper ? nodeDomId : null"
    :ref="setNodeRef"
    :class="nodeClass"
    :style="outerStyle"
    :data-node-id="node.id"
    :data-node-type="node.type"
    :data-node-dom-id="nodeDomId"
    v-on="useComponentWrapper ? mergedEventListeners : {}"
    @click.stop="handleClick"
    @dblclick.stop="handleDoubleClick"
    @pointerdown.capture="handlePointerDown"
    @dragover.prevent="handleDragOver"
    @dragstart.prevent
    @dragleave="handleDragLeave"
    @drop.prevent="handleDrop"
    @contextmenu.prevent="handleContextMenu"
  >
    <component
      :is="renderTag"
      v-if="!useComponentWrapper"
      v-bind="filteredProps"
      :id="!useComponentWrapper ? nodeDomId : null"
      :key="renderKey"
      ref="contentRef"
      :style="contentStyleWithConfig"
      v-on="mergedEventListeners"
    >
      <template v-if="displayContent !== null">{{ displayContent }}</template>
      <component
        :is="customRendererComponent"
        v-else-if="customRendererComponent"
        :node="node"
        :resolved-props="resolvedNodeProps"
      />
      <template v-if="isSelectType">
        <el-option
          v-for="option in selectOptionsList"
          :key="option.value ?? option.label"
          :label="option.label"
          :value="option.value"
        />
      </template>
      <template v-if="isRadioType">
        <el-radio
          v-for="option in radioOptionsList"
          v-show="option.visible !== false"
          :key="option.value ?? option.label"
          :label="option.value"
          :disabled="Boolean(option.disabled)"
        >
          {{ option.label }}
        </el-radio>
      </template>
      <template v-if="isCheckboxType">
        <el-checkbox
          v-for="option in checkboxOptionsList"
          v-show="option.visible !== false"
          :key="option.value ?? option.label"
          :label="option.value"
          :disabled="Boolean(option.disabled)"
        >
          {{ option.label }}
        </el-checkbox>
      </template>
      <template v-if="isTableType">
        <el-table-column
          v-for="column in tableColumnsList"
          :key="column.prop ?? column.label"
          v-bind="column"
        />
      </template>
      <template v-if="isBigDataTableType">
        <el-table-column
          v-for="column in bigTableColumnsList"
          :key="column.prop ?? column.label"
          v-bind="column"
        />
      </template>
      <template v-if="isMenuType">
        <el-menu-item
          v-for="item in menuItemsList"
          :key="item.index ?? item.label"
          :index="item.index ?? item.label"
          :disabled="Boolean(item.disabled)"
        >
          <el-icon v-if="item.iconComponent" class="menu-item-icon">
            <component :is="item.iconComponent" />
          </el-icon>
          {{ item.label }}
        </el-menu-item>
      </template>
      <template v-if="isTimelineType">
        <el-timeline-item
          v-for="item in timelineItemsList"
          :key="item.timestamp ?? item.label"
          :timestamp="item.timestamp"
        >
          {{ item.label }}
        </el-timeline-item>
      </template>
      <template v-if="isTabsType">
        <el-tab-pane
          v-for="tab in tabsListItems"
          :key="tab.name ?? tab.label"
          :label="tab.label"
          :name="tab.name"
        >
          <div
            class="tabs-pane-body"
            :class="{ 'is-drop-active': isDropActive }"
            :data-node-id="node.id"
            :data-node-type="node.type"
            :data-tab-key="String(tab.name ?? tab.label ?? '')"
            @dragover.prevent="handleDragOver"
            @dragleave="handleDragLeave"
            @drop.prevent="handleDrop"
          >
            <template v-if="isContainer && getTabChildIds(tab).length === 0 && !props.isRoot">
              <div class="empty-container-hint">
                <span v-if="isDropActive">释放以添加组件</span>
                <span v-else>拖拽组件到此处</span>
              </div>
            </template>
            <span
              v-if="getTabChildIds(tab).length === 0 && tab.content"
              class="tabs-pane-placeholder"
            >
              {{ tab.content }}
            </span>
            <NodeRenderer
              v-for="childId in getTabChildIds(tab)"
              :key="childId"
              :node-id="childId"
              :readonly="props.readonly"
            />
          </div>
        </el-tab-pane>
      </template>
      <template v-if="isCollapseType">
        <el-collapse-item
          v-for="item in collapseItemsList"
          :key="item.name ?? item.title"
          :name="item.name"
          :title="item.title"
          @click="handleCollapseHeaderClick(item)"
        >
          <div
            class="collapse-pane-body"
            :class="{ 'is-drop-active': isDropActive && isActiveCollapseItem(item) }"
            :data-node-id="node.id"
            :data-node-type="node.type"
            :data-collapse-key="resolveCollapseItemKey(item)"
            @dragover.prevent="handleDragOver"
            @dragleave="handleDragLeave"
            @drop.prevent="handleDrop"
          >
            <template v-if="isContainer && getCollapseChildIds(item).length === 0 && !props.isRoot">
              <div class="empty-container-hint">
                <span v-if="isDropActive">释放以添加组件</span>
                <span v-else>拖拽组件到此处</span>
              </div>
            </template>
            <span
              v-if="getCollapseChildIds(item).length === 0 && item.content"
              class="collapse-pane-placeholder"
            >
              {{ item.content }}
            </span>
            <NodeRenderer
              v-for="childId in getCollapseChildIds(item)"
              :key="childId"
              :node-id="childId"
              :readonly="props.readonly"
            />
          </div>
        </el-collapse-item>
      </template>
      <template v-if="isStepsType">
        <el-step
          v-for="item in stepsItemsList"
          :key="item.title"
          :title="item.title"
          :description="item.description"
        />
      </template>
      <template v-if="isCarouselSlotType">
        <el-carousel-item v-for="item in carouselItemsList" :key="item.label">
          <div class="carousel-item-placeholder">{{ item.label }}</div>
        </el-carousel-item>
      </template>
      <template v-if="isDropdownType" #default>
        <el-button size="small" type="primary">{{ dropdownLabel }}</el-button>
      </template>
      <template v-if="isDropdownType" #dropdown>
        <el-dropdown-menu>
          <el-dropdown-item
            v-for="item in dropdownItemsList"
            :key="item.value ?? item.label"
            :command="item.value"
            :disabled="Boolean(item.disabled)"
            :divided="Boolean(item.divided ?? item.diveded)"
            :icon="item.icon"
          >
            {{ item.label }}
          </el-dropdown-item>
        </el-dropdown-menu>
      </template>
      <template
        v-if="
          isContainer &&
          !hasChildren &&
          !props.isRoot &&
          !isTabsType &&
          !isCollapseType &&
          !suppressReadonlyEmptyHint
        "
      >
        <div class="empty-container-hint" :class="{ 'is-region-hint': isRegionContainer }">
          <span v-if="isDropActive">释放以添加组件</span>
          <span v-else>{{ isRegionContainer ? regionHintText : "拖拽组件到此处" }}</span>
        </div>
      </template>
      <!-- 插入线指示器 -->
      <teleport v-if="showInsertLine && insertLineStyle && insertLineBox" to="body">
        <div
          class="insert-line"
          :class="insertLineStyle.orientation"
          :style="{
            left:
              insertLineStyle.orientation === 'vertical'
                ? `${insertLineBox?.left + insertLineStyle.offset}px`
                : `${insertLineBox?.left}px`,
            top:
              insertLineStyle.orientation === 'horizontal'
                ? `${insertLineBox?.top + insertLineStyle.offset}px`
                : `${insertLineBox?.top}px`,
            width: insertLineStyle.orientation === 'vertical' ? '2px' : `${insertLineBox?.width}px`,
            height:
              insertLineStyle.orientation === 'horizontal' ? '2px' : `${insertLineBox?.height}px`,
          }"
        />
      </teleport>
      <div
        v-else-if="showInsertLine && insertLineStyle"
        class="insert-line"
        :class="insertLineStyle.orientation"
        :style="{
          [insertLineStyle.orientation === 'horizontal' ? 'top' : 'left']:
            `${insertLineStyle.offset}px`,
        }"
      />
      <NodeRenderer
        v-for="childId in childNodeIds"
        :key="childId"
        :node-id="childId"
        :readonly="props.readonly"
      />
    </component>
    <template v-if="useComponentWrapper">
      <div
        v-if="isContainer && !hasChildren && !props.isRoot && !suppressReadonlyEmptyHint"
        class="empty-container-hint"
        :class="{ 'is-region-hint': isRegionContainer }"
      >
        <span v-if="isDropActive">释放以添加组件</span>
        <span v-else>{{ isRegionContainer ? regionHintText : "拖拽组件到此处" }}</span>
      </div>
      <!-- 插入线指示器 -->
      <teleport v-if="showInsertLine && insertLineStyle && insertLineBox" to="body">
        <div
          class="insert-line"
          :class="insertLineStyle.orientation"
          :style="{
            left:
              insertLineStyle.orientation === 'vertical'
                ? `${insertLineBox?.left + insertLineStyle.offset}px`
                : `${insertLineBox?.left}px`,
            top:
              insertLineStyle.orientation === 'horizontal'
                ? `${insertLineBox?.top + insertLineStyle.offset}px`
                : `${insertLineBox?.top}px`,
            width: insertLineStyle.orientation === 'vertical' ? '2px' : `${insertLineBox?.width}px`,
            height:
              insertLineStyle.orientation === 'horizontal' ? '2px' : `${insertLineBox?.height}px`,
          }"
        />
      </teleport>
      <div
        v-else-if="showInsertLine && insertLineStyle"
        class="insert-line"
        :class="insertLineStyle.orientation"
        :style="{
          [insertLineStyle.orientation === 'horizontal' ? 'top' : 'left']:
            `${insertLineStyle.offset}px`,
        }"
      />
      <NodeRenderer
        v-for="childId in childNodeIds"
        :key="childId"
        :node-id="childId"
        :readonly="props.readonly"
      />
    </template>
    <div v-if="showResizeHandles" class="resize-handles">
      <span
        v-for="handle in visibleResizeHandles"
        :key="handle.key"
        class="resize-handle"
        :class="handle.key"
        :style="{ cursor: handle.cursor }"
        @pointerdown.stop="(event) => handleResizePointerDown(event, handle)"
      />
    </div>
  </component>
</template>

<style scoped>
.designer-node {
  position: relative;
  box-sizing: border-box;
  outline: 1px dashed transparent;
  transition: outline-color 0.15s ease;
}

.designer-node:hover {
  outline-color: rgba(59, 130, 246, 0.4);
}

.designer-node.is-selected {
  outline: 2px solid #3b82f6;
  z-index: 2;
}

.designer-node.is-draggable {
  cursor: move;
}

.designer-node.is-container {
  min-height: 40px;
}

.menu-item-icon {
  margin-right: 6px;
}

.designer-node.tabs-container {
  display: flex;
  flex-direction: column;
}

.designer-node.tabs-container :deep(.el-tabs__content) {
  flex: 1;
  min-height: 0;
  display: flex;
  overflow: visible;
}

.designer-node.tabs-container :deep(.el-tab-pane) {
  flex: 1;
  min-height: 0;
  overflow: visible;
}

.designer-node.tabs-container :deep(.el-tabs__header) {
  margin: 0;
}

.designer-node.tabs-container :deep(.el-tabs--border-card > .el-tabs__content) {
  padding: 0;
}

.designer-node.tabs-container :deep(.el-tabs) {
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: stretch;
}

.designer-node.tabs-container.tabs-pos-left :deep(.el-tabs),
.designer-node.tabs-container.tabs-pos-right :deep(.el-tabs) {
  display: flex !important;
  flex-direction: row !important;
  align-items: stretch;
}

.designer-node.tabs-container.tabs-pos-right :deep(.el-tabs) {
  flex-direction: row !important;
}

.designer-node.tabs-container.tabs-pos-left :deep(.el-tabs__header),
.designer-node.tabs-container.tabs-pos-right :deep(.el-tabs__header) {
  width: auto;
  height: 100%;
  margin: 0;
  padding: 0;
  flex: 0 0 auto;
  float: none;
  clear: none;
  align-self: stretch;
}

.designer-node.tabs-container.tabs-pos-right :deep(.el-tabs__header) {
  order: 2;
}

.designer-node.tabs-container.tabs-pos-left :deep(.el-tabs__content),
.designer-node.tabs-container.tabs-pos-right :deep(.el-tabs__content) {
  width: auto;
  height: 100%;
  margin: 0;
  padding: 0;
  overflow: hidden;
  flex: 1 1 auto;
  min-width: 0;
  float: none;
  clear: none;
}

.designer-node.tabs-container.tabs-pos-right :deep(.el-tabs__content) {
  order: 1;
}

.designer-node.tabs-container.tabs-pos-left :deep(.el-tabs__nav-wrap),
.designer-node.tabs-container.tabs-pos-right :deep(.el-tabs__nav-wrap) {
  height: 100%;
  margin: 0;
  padding: 0;
}

.designer-node.tabs-container.tabs-pos-left :deep(.el-tabs__nav-scroll),
.designer-node.tabs-container.tabs-pos-right :deep(.el-tabs__nav-scroll) {
  height: 100%;
}

.designer-node.tabs-container.tabs-pos-left :deep(.el-tabs__nav),
.designer-node.tabs-container.tabs-pos-right :deep(.el-tabs__nav) {
  display: flex;
  flex-direction: column;
  height: auto;
  margin-top: 0;
}

.designer-node.tabs-container.tabs-pos-left :deep(.el-tabs__content),
.designer-node.tabs-container.tabs-pos-right :deep(.el-tabs__content) {
  min-height: 100%;
}

.tabs-pane-body {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: stretch;
  height: 100%;
  min-height: 40px;
  overflow: visible;
}

.tabs-pane-body .designer-node {
  flex: 1 1 auto;
}

.tabs-pane-body.is-drop-active .empty-container-hint {
  border-color: #3b82f6;
  color: #3b82f6;
  background-color: rgba(59, 130, 246, 0.05);
}

.tabs-pane-placeholder {
  color: #9ca3af;
  font-size: 12px;
}

.collapse-pane-body {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: stretch;
  min-height: 40px;
  overflow: visible;
}

.collapse-pane-body .designer-node {
  flex: 1 1 auto;
}

.collapse-pane-body.is-drop-active .empty-container-hint {
  border-color: #3b82f6;
  color: #3b82f6;
  background-color: rgba(59, 130, 246, 0.05);
}

.collapse-pane-placeholder {
  color: #9ca3af;
  font-size: 12px;
}

.designer-node.el-layout {
  min-height: 0;
}

.designer-node.el-layout-row {
  min-height: 0;
  --layout-select-inset: 0px;
}

.designer-node.el-layout-row:not(.is-preview) {
  border: 1px dashed rgba(59, 130, 246, 0.45);
  border-radius: 2px;
  background: transparent;
}

.designer-node.el-col {
  min-height: 0;
  outline-offset: 0;
  --layout-select-inset: 0px;
}

.designer-node.el-col:not(.is-preview) {
  border: 1px solid rgba(59, 130, 246, 0.45);
  border-radius: 4px;
  background: transparent;
  transition: border-color 0.15s ease;
}

.designer-node.el-col:not(.is-preview):hover {
  border-color: rgba(59, 130, 246, 0.85);
  background: transparent;
}

.designer-node.el-layout-row.is-selected,
.designer-node.el-layout-row:hover {
  outline: none;
}

.designer-node.el-col.is-selected,
.designer-node.el-col:hover {
  outline: none;
}

.designer-node.el-layout-row.is-selected:not(.is-preview) {
  border-style: solid;
  border-color: #3b82f6;
  background: transparent;
}

.designer-node.el-col.is-selected::after {
  content: "";
  position: absolute;
  top: 0;
  bottom: 0;
  left: 0;
  right: 0;
  border: 2px solid #3b82f6;
  border-radius: 2px;
  pointer-events: none;
  box-sizing: border-box;
}

.designer-node.is-locked {
  opacity: 0.6;
  pointer-events: none;
}

.designer-node.drag-over {
  outline: 2px dashed #409eff;
  background-color: rgba(64, 158, 255, 0.1);
}

.empty-container-hint {
  position: absolute;
  top: 0;
  bottom: 0;
  left: var(--col-gutter-x, 0);
  right: var(--col-gutter-x, 0);
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 60px;
  min-height: 40px;
  color: #9ca3af;
  font-size: 12px;
  pointer-events: none;
  border: 1px dashed #bcc3ce;
  border-radius: 4px;
  background-color: rgba(148, 163, 184, 0.06);
  transition: all 0.2s ease;
}

.designer-node.el-layout .empty-container-hint,
.designer-node.el-layout-row .empty-container-hint,
.designer-node.el-col .empty-container-hint {
  min-height: 0;
}

.designer-node.el-layout-row .empty-container-hint {
  left: calc(-1 * var(--col-gutter-x, 0));
  right: calc(-1 * var(--col-gutter-x, 0));
}

.designer-node.el-col .empty-container-hint {
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  box-sizing: border-box;
}

.empty-container-hint.is-region-hint {
  color: #6b7280;
  border-color: #cbd5e1;
  background-color: rgba(59, 130, 246, 0.06);
  letter-spacing: 0.5px;
}

.drag-over .empty-container-hint {
  border-color: #3b82f6;
  color: #3b82f6;
  background-color: rgba(59, 130, 246, 0.05);
}

.designer-node.layout-container-visible:not(.is-preview) > div {
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  min-height: 60px;
}

.designer-node.layout-container-visible:not(.is-preview):hover > div {
  border-color: #a0a8b4;
}

.designer-node.layout-container-visible.is-selected > div {
  border-color: #409eff;
}

.insert-line {
  position: absolute;
  background: var(--designer-danger);
  pointer-events: none;
  z-index: 9999;
  transition: all 0.1s ease;
}

.insert-line.horizontal {
  height: 2px;
  left: 0;
  right: 0;
}

.insert-line.vertical {
  width: 2px;
  top: 0;
  bottom: 0;
}

.carousel-item-placeholder {
  width: 100%;
  height: 100%;
  background: #f5f7fa;
  border: 1px dashed #dcdfe6;
  color: #909399;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
}

.designer-node.is-root {
  outline: none;
}

.designer-node.is-root:hover {
  outline: none;
}

.designer-node.is-preview {
  outline: none;
}

.designer-node.is-preview:hover {
  outline: none;
}

.designer-node.is-preview {
  outline: none !important;
  box-shadow: none !important;
}

.designer-node.is-preview::after {
  display: none !important;
}

.resize-handles {
  position: absolute;
  inset: 0;
  pointer-events: none;
}

.resize-handle {
  position: absolute;
  width: 8px;
  height: 8px;
  background: var(--designer-shell-surface);
  border: 1px solid #3b82f6;
  box-sizing: border-box;
  border-radius: 2px;
  pointer-events: auto;
  z-index: 10;
}

.resize-handle.nw {
  left: -4px;
  top: -4px;
}

.resize-handle.n {
  left: 50%;
  top: -4px;
  transform: translateX(-50%);
}

.resize-handle.ne {
  right: -4px;
  top: -4px;
}

.resize-handle.e {
  right: -4px;
  top: 50%;
  transform: translateY(-50%);
}

.resize-handle.se {
  right: -4px;
  bottom: -4px;
}

.resize-handle.s {
  left: 50%;
  bottom: -4px;
  transform: translateX(-50%);
}

.resize-handle.sw {
  left: -4px;
  bottom: -4px;
}

.resize-handle.w {
  left: -4px;
  top: 50%;
  transform: translateY(-50%);
}
</style>
