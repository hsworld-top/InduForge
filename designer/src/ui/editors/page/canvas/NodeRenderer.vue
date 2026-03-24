<!--
  NodeRenderer - 节点渲染器
  递归渲染组件树，支持 DOM 组件与 Canvas 图形，处理绑定、事件、拖拽
-->
<template>
  <component v-if="node && isNodeVisible" :is="outerTag" :class="nodeClass" :style="outerStyle" :data-node-id="node.id"
    :data-node-type="node.type" :id="useComponentWrapper ? nodeDomId : null" :ref="setNodeRef"
    v-bind="useComponentWrapper ? filteredProps : {}" v-on="useComponentWrapper ? mergedEventListeners : {}"
    @click.stop="handleClick" @dblclick.stop="handleDoubleClick" @pointerdown.capture="handlePointerDown"
    @dragover.prevent="handleDragOver" @dragstart.prevent @dragleave="handleDragLeave" @drop.prevent="handleDrop"
    @contextmenu.prevent="handleContextMenu">
    <component v-if="!useComponentWrapper" :is="renderTag" :key="renderKey"
      :id="!useComponentWrapper ? nodeDomId : null" :style="contentStyleWithConfig" v-bind="filteredProps"
      v-on="mergedEventListeners" ref="contentRef">
      <template v-if="displayContent !== null">{{ displayContent }}</template>
      <component v-else-if="customRendererComponent" :is="customRendererComponent" :node="node"
        :resolved-props="resolvedNodeProps" />
      <template v-if="isSelectType">
        <el-option v-for="option in selectOptions" :key="option.value ?? option.label" :label="option.label"
          :value="option.value" />
      </template>
      <template v-if="isRadioType">
        <el-radio v-for="option in radioOptions" :key="option.value ?? option.label" :label="option.value"
          :disabled="Boolean(option.disabled)" v-show="option.visible !== false">
          {{ option.label }}
        </el-radio>
      </template>
      <template v-if="isCheckboxType">
        <el-checkbox v-for="option in checkboxOptions" :key="option.value ?? option.label" :label="option.value"
          :disabled="Boolean(option.disabled)" v-show="option.visible !== false">
          {{ option.label }}
        </el-checkbox>
      </template>
      <template v-if="isTableType">
        <el-table-column v-for="column in tableColumns" :key="column.prop ?? column.label" v-bind="column" />
      </template>
      <template v-if="isBigDataTableType">
        <el-table-column v-for="column in bigTableColumns" :key="column.prop ?? column.label" v-bind="column" />
      </template>
      <template v-if="isMenuType">
        <el-menu-item v-for="item in menuItems" :key="item.index ?? item.label" :index="item.index ?? item.label"
          :disabled="Boolean(item.disabled)">
          <el-icon v-if="item.iconComponent" class="menu-item-icon">
            <component :is="item.iconComponent" />
          </el-icon>
          {{ item.label }}
        </el-menu-item>
      </template>
      <template v-if="isTimelineType">
        <el-timeline-item v-for="item in timelineItems" :key="item.timestamp ?? item.label" :timestamp="item.timestamp">
          {{ item.label }}
        </el-timeline-item>
      </template>
      <template v-if="isTabsType">
        <el-tab-pane v-for="tab in tabsList" :key="tab.name ?? tab.label" :label="tab.label" :name="tab.name">
          <div class="tabs-pane-body" :class="{ 'is-drop-active': isDropActive }" :data-node-id="node.id"
            :data-node-type="node.type" @dragover.prevent="handleDragOver" @dragleave="handleDragLeave"
            @drop.prevent="handleDrop">
            <template v-if="isActiveTab(tab)">
              <template v-if="
                isContainer && activeTabChildIds.length === 0 && !props.isRoot
              ">
                <div class="empty-container-hint">
                  <span v-if="isDropActive">释放以添加组件</span>
                  <span v-else>拖拽组件到此处</span>
                </div>
              </template>
              <span v-if="activeTabChildIds.length === 0 && tab.content" class="tabs-pane-placeholder">
                {{ tab.content }}
              </span>
              <NodeRenderer v-for="childId in activeTabChildIds" :key="childId" :node-id="childId"
                :readonly="props.readonly" />
            </template>
            <template v-else>
              <span class="tabs-pane-placeholder">
                {{ tab.content }}
              </span>
            </template>
          </div>
        </el-tab-pane>
      </template>
      <template v-if="isCollapseType">
        <el-collapse-item v-for="item in collapseItems" :key="item.name ?? item.title" :name="item.name"
          :title="item.title">
          {{ item.content }}
        </el-collapse-item>
      </template>
      <template v-if="isStepsType">
        <el-step v-for="item in stepsItems" :key="item.title" :title="item.title" :description="item.description" />
      </template>
      <template v-if="isCarouselSlotType">
        <el-carousel-item v-for="item in carouselItems" :key="item.label">
          <div class="carousel-item-placeholder">{{ item.label }}</div>
        </el-carousel-item>
      </template>
      <template v-if="isDropdownType" #default>
        <el-button size="small" type="primary">{{ dropdownLabel }}</el-button>
      </template>
      <template v-if="isDropdownType" #dropdown>
        <el-dropdown-menu>
          <el-dropdown-item v-for="item in dropdownItems" :key="item.value ?? item.label" :command="item.value"
            :disabled="Boolean(item.disabled)" :divided="Boolean(item.divided ?? item.diveded)" :icon="item.icon">
            {{ item.label }}
          </el-dropdown-item>
        </el-dropdown-menu>
      </template>
      <template v-if="
        isContainer &&
        !hasChildren &&
        !props.isRoot &&
        !isTabsType &&
        !suppressReadonlyEmptyHint
      ">
        <div class="empty-container-hint" :class="{ 'is-region-hint': isRegionContainer }">
          <span v-if="isDropActive">释放以添加组件</span>
          <span v-else>{{
            isRegionContainer ? regionHintText : "拖拽组件到此处"
          }}</span>
        </div>
      </template>
      <!-- 插入线指示器 -->
      <teleport v-if="showInsertLine && insertLineStyle && insertLineBox" to="body">
        <div class="insert-line" :class="insertLineStyle.orientation" :style="{
          left:
            insertLineStyle.orientation === 'vertical'
              ? insertLineBox?.left + insertLineStyle.offset + 'px'
              : insertLineBox?.left + 'px',
          top:
            insertLineStyle.orientation === 'horizontal'
              ? insertLineBox?.top + insertLineStyle.offset + 'px'
              : insertLineBox?.top + 'px',
          width:
            insertLineStyle.orientation === 'vertical'
              ? '2px'
              : insertLineBox?.width + 'px',
          height:
            insertLineStyle.orientation === 'horizontal'
              ? '2px'
              : insertLineBox?.height + 'px',
        }" />
      </teleport>
      <div v-else-if="showInsertLine && insertLineStyle" class="insert-line" :class="insertLineStyle.orientation"
        :style="{
          [insertLineStyle.orientation === 'horizontal' ? 'top' : 'left']:
            insertLineStyle.offset + 'px',
        }" />
      <NodeRenderer v-for="childId in node.children || []" v-if="!isTabsType" :key="childId"
        :node-id="childId" :readonly="props.readonly" />
    </component>
    <template v-if="useComponentWrapper">
      <div v-if="
        isContainer &&
        !hasChildren &&
        !props.isRoot &&
        !suppressReadonlyEmptyHint
      " class="empty-container-hint" :class="{ 'is-region-hint': isRegionContainer }">
        <span v-if="isDropActive">释放以添加组件</span>
        <span v-else>{{
          isRegionContainer ? regionHintText : "拖拽组件到此处"
        }}</span>
      </div>
      <!-- 插入线指示器 -->
      <teleport v-if="showInsertLine && insertLineStyle && insertLineBox" to="body">
        <div class="insert-line" :class="insertLineStyle.orientation" :style="{
          left:
            insertLineStyle.orientation === 'vertical'
              ? insertLineBox?.left + insertLineStyle.offset + 'px'
              : insertLineBox?.left + 'px',
          top:
            insertLineStyle.orientation === 'horizontal'
              ? insertLineBox?.top + insertLineStyle.offset + 'px'
              : insertLineBox?.top + 'px',
          width:
            insertLineStyle.orientation === 'vertical'
              ? '2px'
              : insertLineBox?.width + 'px',
          height:
            insertLineStyle.orientation === 'horizontal'
              ? '2px'
              : insertLineBox?.height + 'px',
        }" />
      </teleport>
      <div v-else-if="showInsertLine && insertLineStyle" class="insert-line" :class="insertLineStyle.orientation"
        :style="{
          [insertLineStyle.orientation === 'horizontal' ? 'top' : 'left']:
            insertLineStyle.offset + 'px',
        }" />
      <NodeRenderer v-for="childId in node.children || []" :key="childId" :node-id="childId"
        :readonly="props.readonly" />
    </template>
    <div v-if="showResizeHandles" class="resize-handles">
      <span v-for="handle in visibleResizeHandles" :key="handle.key" class="resize-handle" :class="handle.key"
        :style="{ cursor: handle.cursor }" @pointerdown.stop="(event) => handleResizePointerDown(event, handle)" />
    </div>
  </component>
</template>

<script setup>
import {
  computed,
  ref,
  inject,
  nextTick,
  onBeforeUnmount,
  onMounted,
  watch,
} from "vue";
import { storeToRefs } from "pinia";
import { ElMessage } from "element-plus";
import { useEditorStore } from "@/stores/editor-store";
import { datacenterApi } from "@/services";
import { getPreviewRuntime } from "@/ui/editors/page/preview/previewRuntime";
import {
  componentRegistry,
  createSelectableElement,
} from "@/editor-core";
import { normalizeEventDefinitions } from "@/editor-core/registry/component-events";
import {
  getCustomRenderer,
  getDisplayContent,
  isFlexContainer,
  isChildResizable as isChildResizableByDescriptor,
  getFlexDirection,
  getRenderTag,
  isNodeDesignerMovable,
  isTableLikeType,
  usesLegacyFlexDirectionProps,
} from "@/components/descriptors/registry.ts";
import { createDragDropManager } from "./services/DragDropManager";
import { createNodeStyleHelpers } from "./composables/use-node-style.js";
import { useNodeContent } from "./composables/use-node-content.js";
import { useNodeProps, buildExpressionContext, resolveExpressionValue } from "./composables/use-node-props.js";
import { useNodeInteraction } from "./composables/use-node-interaction.js";
import { useNodeDrop } from "./composables/use-node-drop.js";
import { usePreview } from "./composables/use-preview.js";
import { useNodeResize } from "./composables/use-node-resize.js";
import { useNodePointer } from "./composables/use-node-pointer.js";
import { useBuildRefInfo } from "./composables/use-build-ref-info.js";
import {
  useNodeRendererDerivations,
  createApplyMenuDslConfig,
  resolveMenuConfigFromContent,
} from "./composables/use-node-renderer-derivations.js";
import { useNodeRendererTypeFlags } from "./composables/use-node-renderer-type-flags.js";
import EChart from "./components/EChart.vue";
import {
  useDragState,
  startDrag,
  endDrag,
  updateDropTarget,
  clearDropTarget,
} from "./composables/use-drag-state";

const props = defineProps({
  nodeId: {
    type: String,
    required: true,
  },
  isRoot: {
    type: Boolean,
    default: false,
  },
  readonly: {
    type: Boolean,
    default: false,
  },
});

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
const showContextMenu = inject("showContextMenu", null);
const nodeStyleHelpers = createNodeStyleHelpers({ doc, currentPage, props });
const resolveLayoutStyle = nodeStyleHelpers.resolveLayoutStyle;
const isRootCanvasContainer = nodeStyleHelpers.isRootCanvasContainer;

const node = computed(() => {
  docVersion.value;
  return doc.value?.getNode(props.nodeId) || null;
});

const detailConfigText = computed(() => {
  docVersion.value;
  return String(node.value?.detailConfig || "").trim();
});

let applyMenuDslConfig = () => {};

// nodeRef 和 contentRef 需要先声明，因为会被 useNodeInteraction 等 composable 使用
const nodeRef = ref(null);
const contentRef = ref(null);
const setNodeRef = (el) => {
  nodeRef.value = el?.$el || el;
};

/**
 * 处理节点选中（Shift/Meta/Ctrl 多选）
 * @param {MouseEvent} event - 鼠标事件
 */
function handleSelect(event) {
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
let buildRefInfo = null;
const previewRuntime = usePreview({
  node,
  doc,
  currentPage,
  projectVariables,
  projectId,
  docVersion,
  globalScripts,
  datacenterApi,
  buildRefInfo: () => buildRefInfo?.(),
  readonly: computed(() => props.readonly),
  detailConfigText,
  resolveMenuConfigFromContent,
  applyMenuDslConfig: (config) => applyMenuDslConfig(config),
});
const {
  runPreviewScript,
  isRunningDetailConfig: isRunningDetailConfigFn,
} = previewRuntime;

// 使用 useNodeInteraction composable
const nodeInteraction = useNodeInteraction({
  node,
  doc,
  selection,
  selectionVersion,
  readonly: computed(() => props.readonly),
  isRoot: computed(() => props.isRoot),
  isRootCanvasContainer,
  isChildResizableByDescriptor,
  nodeRef,
  createSelectableElement,
  runPreviewScript,
  handleSelect,
  showContextMenu,
});
const {
  handleClick,
  handleDoubleClick,
  handleContextMenu,
  showResizeHandlesBase,
  visibleResizeHandles: visibleResizeHandlesFromComposable,
  resizeHandles: resizeHandlesFromComposable,
  getRegionResizeConfig: getRegionResizeConfigFromComposable,
  isElColInRow: isElColInRowFromComposable,
  isChildInElCol: isChildInElColFromComposable,
  isContainer: isContainerFromComposable,
  isRegionContainer: isRegionContainerFromComposable,
} = nodeInteraction;
const canvasZoom = inject("canvasZoom", ref(1));
const dragState = useDragState();
const previewPageId = computed(
  () => currentPage.value?.name || currentPage.value?.id || "",
);
let registerTimer = null;
let registerAttempts = 0;
const maxRegisterAttempts = 10;
const notifyInsertFailure = (fallbackMessage) => {
  const message = error.value || fallbackMessage || "插入失败：当前不可编辑";
  ElMessage.warning(message);
};

// 使用 useNodeProps composable（不传 activeTabName，避免循环依赖）
const nodeProps = useNodeProps({
  node,
  doc,
  currentPage,
  projectVariables,
  docVersion,
  readonly: computed(() => props.readonly),
});
const { resolvedNodeProps, resolvedProps: resolvedPropsBase, filteredProps: filteredPropsFromComposable } = nodeProps;

// 使用 useNodeContent composable（传入 resolvedNodeProps）
const nodeContent = useNodeContent({ node, resolvedNodeProps, docVersion });
const {
  selectOptions,
  radioOptions,
  checkboxOptions,
  dropdownItems,
  tabsList,
  normalizeMenuItems,
  activeTabName,
  tableRenderVersion,
  syncActiveTabName,
  applyTabsModelValueToProps,
} = nodeContent;

const { buildRefInfo: buildRefInfoImpl, applyPreviewPatch } = useBuildRefInfo({
  node,
  nodeRef,
  contentRef,
  editorStore,
  readonly: computed(() => props.readonly),
  tableRenderVersion,
  docVersion,
  isRunningDetailConfigFn,
});
buildRefInfo = buildRefInfoImpl;

applyMenuDslConfig = createApplyMenuDslConfig({
  node,
  readonly: computed(() => props.readonly),
  applyPreviewPatch,
  editorStore,
  normalizeMenuItems,
});

// 应用 Tabs modelValue 处理到 resolvedProps（通过 useNodeContent 的辅助函数）
const resolvedProps = computed(() => {
  const base = resolvedPropsBase.value;
  if (isTableLikeType(node.value?.type)) {
    tableRenderVersion.value;
  }
  return applyTabsModelValueToProps(base);
});

const dragDropManager = createDragDropManager();

function resolveFlexDirection(type, element) {
  const descriptorDirection = getFlexDirection(type);
  if (descriptorDirection) {
    return descriptorDirection;
  }
  if (usesLegacyFlexDirectionProps(type)) {
    return node.value?.props?.direction || "column";
  }
  return dragDropManager.getContainerDirection(element);
}

const isMovable = computed(() => {
  if (!node.value || props.isRoot || node.value.locked) return false;
  if (isRootCanvasContainer(node.value)) return false;
  return isNodeDesignerMovable(node.value.type);
});

const nodeDrop = useNodeDrop({
  node,
  doc,
  dragState,
  dragDropManager,
  isContainer: isContainerFromComposable,
  resolveFlexDirection,
  isFlexContainer,
  readonly: computed(() => props.readonly),
  editorStore,
  canvasZoom,
  endDrag,
  notifyInsertFailure,
  activeTabName,
  tabsList,
});
const {
  handleDragOver,
  handleDrop,
  showInsertLine,
  insertLineStyle,
  rowInsertInfo,
  layoutInsertInfo,
  insertLineBox,
  isDragOver,
  canAcceptChild,
  isDroppableContainer,
  resolveDropContainer,
} = nodeDrop;
const nodeResize = useNodeResize({
  node,
  doc,
  nodeRef,
  readonly: computed(() => props.readonly),
  isMovable,
  isElColInRow: isElColInRowFromComposable,
  isChildInElCol: isChildInElColFromComposable,
  selection,
  canvasZoom,
  history,
  editorStore,
  isChildResizableByDescriptor,
  getRegionResizeConfig: getRegionResizeConfigFromComposable,
});
const { handleResizePointerDown } = nodeResize;
const isDropActive = computed(() => {
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
  activeTabChildIds,
  regionHintText,
  useComponentWrapper,
  renderKey,
  nodeClass,
} = useNodeRendererDerivations({
  node,
  detailConfigText,
  docVersion,
  tableRenderVersion,
  resolvedNodeProps,
  selectionVersion,
  selection,
  doc,
  isContainer: isContainerFromComposable,
  isMovable,
  isDropActive,
  activeTabName,
  tabsList,
  props,
});

const filteredProps = filteredPropsFromComposable;

const tabHeaderWidth = ref(0);

/**
 * 同步 Tabs 头部宽度（用于左右布局）
 */
const syncTabsHeaderWidth = () => {
  if (!node.value || !isTabsType.value) return;
  const tabPosition =
    resolvedNodeProps.value?.tabPosition ||
    node.value?.props?.tabPosition ||
    "top";
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
};

watch(
  () => [
    node.value?.type,
    resolvedNodeProps.value?.tabPosition,
    tabsList.value?.length,
  ],
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
const handleTabsClick = (pane) => {
  if (!pane) return;
  const name =
    pane?.props?.name ?? pane?.name ?? pane?.paneName ?? pane?.label ?? "";
  if (name) {
    activeTabName.value = String(name);
  }
};

/**
 * 处理 Tabs 切换事件
 * @param {string} name - 激活名称
 */
const handleTabsChange = (name) => {
  if (!name) return;
  activeTabName.value = String(name);
};

/**
 * 处理 Tabs 删除事件
 * @param {string} name - Tab 名称
 */
const resolveTabNameValue = (input) => {
  if (input && typeof input === "object") {
    return (
      input?.props?.name ?? input?.name ?? input?.paneName ?? input?.label ?? ""
    );
  }
  return input ?? "";
};

const handleTabsRemove = (name) => {
  if (!isTabsType.value) return;
  const resolvedName = resolveTabNameValue(name);
  if (!resolvedName) return;
  const tabs = Array.isArray(node.value?.props?.tabs)
    ? [...node.value.props.tabs]
    : [];
  if (!tabs.length) return;
  const normalizedName = String(resolvedName);
  const updateTabsProps = (patch) => {
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

  const childIds = Array.isArray(node.value?.children)
    ? [...node.value.children]
    : [];
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
};

/**
 * 处理 Tabs 编辑事件
 * @param {string} name - Tab 名称
 * @param {string} action - edit 动作
 */
const handleTabsEdit = (name, action) => {
  if (action !== "remove") return;
  handleTabsRemove(name);
};
const isActiveTab = (tab) => {
  if (!tab) return false;
  const name = tab.name ?? tab.label ?? "";
  return String(name) === activeTabName.value;
};

const isContainer = isContainerFromComposable;

const isNodeVisible = computed(() => {
  docVersion.value;
  if (!node.value) return false;
  if (node.value.hidden) return false;
  const visibleConfig = node.value.conditions?.visible;
  if (typeof visibleConfig === "boolean") return visibleConfig;
  if (typeof visibleConfig !== "string" || !visibleConfig.trim()) return true;
  const context = buildExpressionContext(resolvedNodeProps.value || {}, { doc, currentPage, projectVariables });
  const value = resolveExpressionValue(visibleConfig, context, true);
  return Boolean(value);
});

const hasChildren = computed(() => {
  docVersion.value;
  if (!node.value || !doc.value) return false;
  if (typeof doc.value.getChildren === "function") {
    return doc.value.getChildren(node.value.id).length > 0;
  }
  return (node.value.children || []).length > 0;
});

const nodePointer = useNodePointer({
  node,
  doc,
  nodeRef,
  readonly: computed(() => props.readonly),
  isMovable,
  isContainer,
  selection,
  canvasZoom,
  history,
  editorStore,
  currentPage,
  startDrag,
  endDrag,
  updateDropTarget,
  clearDropTarget,
  dragDropManager,
  canAcceptChild,
  showInsertLine,
  insertLineStyle,
  rowInsertInfo,
  layoutInsertInfo,
  activeTabName,
  tabsList,
  resolveFlexDirection,
});
const { handlePointerDown } = nodePointer;

const visibleResizeHandles = visibleResizeHandlesFromComposable;
const resizeHandles = resizeHandlesFromComposable;
const getRegionResizeConfig = getRegionResizeConfigFromComposable;
const isElColInRow = isElColInRowFromComposable;
const isChildInElCol = isChildInElColFromComposable;

// showResizeHandles 需要额外检查 selection，所以保留一个包装 computed
const showResizeHandles = computed(() => {
  const base = showResizeHandlesBase.value;
  if (!base) return false;
  // 额外检查：只有选中时才显示
  return Boolean(selection.value?.isSelected?.(node.value?.id));
});


const isRegionContainer = isRegionContainerFromComposable;

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
} = useNodeRendererTypeFlags(node, {
  readonly: computed(() => props.readonly),
  isRegionContainer,
});

const renderTag = computed(() => {
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
const customRendererComponent = computed(() => {
  if (!node.value) return null;
  return getCustomRenderer(node.value.type) ?? null;
});

const outerTag = computed(() => {
  return useComponentWrapper.value ? renderTag.value : "div";
});

const displayContent = computed(() => {
  docVersion.value;
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

const layoutStyle = computed(() => {
  docVersion.value;
  if (!node.value) return {};
  return resolveLayoutStyle(node.value, props.isRoot);
});

// 使用 use-node-style composable 创建 contentStyle
const contentStyle = nodeStyleHelpers.createContentStyle(
  node,
  doc,
  docVersion,
  resolvedNodeProps,
  computed(() => isContainer.value),
  isMovable,
  layoutStyle,
  props,
);

// 使用 use-node-style composable 创建 wrapperStyle
const wrapperStyle = nodeStyleHelpers.createWrapperStyle(
  node,
  doc,
  layoutStyle,
  isMovable,
  computed(() => isContainer.value),
);

// 使用 use-node-style composable 创建 wrapperComponentStyle
const wrapperComponentStyle = nodeStyleHelpers.createWrapperComponentStyle(
  node,
  doc,
  useComponentWrapper,
  wrapperStyle,
  contentStyle,
  resolvedProps,
  props,
);

const styleConfigText = computed(() => {
  docVersion.value;
  return String(node.value?.styleConfig || "").trim();
});
const hasStyleConfigSelector = computed(() =>
  styleConfigText.value.includes("{"),
);
const hasDomIdSelector = computed(() =>
  styleConfigText.value.includes("#domId"),
);
const nodeDomId = computed(() => {
  if (!node.value) return "";
  const customId =
    typeof node.value.props?.id === "string" ? node.value.props.id.trim() : "";
  if (customId) return customId;
  if (!node.value.id) return "";
  return `dom-${node.value.id}`;
});
const styleScopeSelector = computed(() => {
  if (!node.value?.id) return "";
  if (nodeDomId.value) return `#${nodeDomId.value}`;
  return `[data-node-id="${node.value.id}"]`;
});
const normalizedStyleConfigText = computed(() => {
  if (!styleConfigText.value) return "";
  if (!styleScopeSelector.value) return styleConfigText.value;
  return styleConfigText.value.replaceAll("#domId", styleScopeSelector.value);
});
const inlineStyleConfig = computed(() => {
  if (hasStyleConfigSelector.value) return "";
  return styleConfigText.value ? styleConfigText.value : "";
});
const inlineStyleConfigObject = computed(() => {
  const raw = inlineStyleConfig.value;
  if (!raw) return {};
  const stripped = raw.replace(/\/\*[\s\S]*?\*\//g, "");
  const entries = stripped
    .split(";")
    .map((item) => item.trim())
    .filter(Boolean);
  const result = {};
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
const domIdStyleConfigObject = computed(() => {
  const raw = styleConfigText.value;
  if (!raw || !raw.includes("#domId")) return {};
  const stripped = raw.replace(/\/\*[\s\S]*?\*\//g, "");
  const selectorIndex = stripped.indexOf("#domId");
  if (selectorIndex < 0) return {};
  const openIndex = stripped.indexOf("{", selectorIndex);
  if (openIndex < 0) return {};
  const closeIndex = findMatchingBrace(stripped, openIndex);
  if (closeIndex < 0) return {};
  const body = stripped.slice(openIndex + 1, closeIndex);
  const entries = body
    .split(";")
    .map((item) => item.trim())
    .filter(Boolean);
  const result = {};
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
const resolvedInlineStyleConfigObject = computed(() => {
  return {
    ...inlineStyleConfigObject.value,
    ...domIdStyleConfigObject.value,
  };
});

/**
 * 拼接选择器前缀
 * @param {string} selectorText - 选择器文本
 * @param {string} scope - 作用域选择器
 * @returns {string}
 */
const prefixSelectors = (selectorText, scope) => {
  return selectorText
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean)
    .map((item) => (item.startsWith(scope) ? item : `${scope} ${item}`))
    .join(", ");
};

/**
 * 查找匹配的右花括号
 * @param {string} text - 源文本
 * @param {number} startIndex - 起始索引
 * @returns {number}
 */
const findMatchingBrace = (text, startIndex) => {
  let depth = 0;
  for (let i = startIndex; i < text.length; i += 1) {
    const ch = text[i];
    if (ch === "{") depth += 1;
    if (ch === "}") {
      depth -= 1;
      if (depth === 0) return i;
    }
  }
  return -1;
};

/**
 * 构建带作用域的样式规则
 * @param {string} cssText - 原始 CSS
 * @param {string} scope - 作用域选择器
 * @returns {string}
 */
const buildScopedCss = (cssText, scope) => {
  if (!cssText) return "";
  if (!cssText.includes("{")) {
    return `${scope} { ${cssText} }`;
  }

  let result = "";
  let index = 0;
  const text = cssText;

  while (index < text.length) {
    const nextOpen = text.indexOf("{", index);
    if (nextOpen === -1) break;
    const selector = text.slice(index, nextOpen).trim();
    const closeIndex = findMatchingBrace(text, nextOpen);
    if (closeIndex === -1) break;
    const body = text.slice(nextOpen + 1, closeIndex);

    if (selector.startsWith("@")) {
      const nested = buildScopedCss(body, scope);
      result += `${selector}{${nested}}`;
    } else if (selector) {
      result += `${prefixSelectors(selector, scope)}{${body}}`;
    }

    index = closeIndex + 1;
  }

  return result;
};

const styleConfigCss = computed(() => {
  if (!node.value?.id || !hasStyleConfigSelector.value) return "";
  if (hasDomIdSelector.value) {
    return normalizedStyleConfigText.value;
  }
  const scope = styleScopeSelector.value || `[data-node-id="${node.value.id}"]`;
  return buildScopedCss(normalizedStyleConfigText.value, scope);
});

// 使用 use-node-style composable 创建样式注入生命周期
const { styleElementRef } = nodeStyleHelpers.createStyleElementSync(
  node,
  styleConfigCss,
);

/**
 * 外层样式（处理组件包装模式）
 */
const outerStyle = computed(() => {
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
const contentStyleWithConfig = computed(() => {
  if (useComponentWrapper.value) return contentStyle.value;
  const inlineStyle = resolvedInlineStyleConfigObject.value;
  if (!inlineStyle || Object.keys(inlineStyle).length === 0) {
    return contentStyle.value;
  }
  return [contentStyle.value, inlineStyle];
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
const supportsModelValue = computed(() =>
  modelValueTypes.has(node.value?.type),
);

/**
 * 更新组件的 modelValue
 * @param {any} value - 新值
 */
const handleModelValueUpdate = (value) => {
  if (!node.value) return;
  if (props.readonly) {
    applyPreviewPatch({ props: { modelValue: value } });
    return;
  }
  editorStore.updateNode(node.value.id, {
    props: { ...(node.value.props || {}), modelValue: value },
  });
};

const componentEventListeners = computed(() => {
  if (!node.value) return {};
  const listeners = {};
  if (supportsModelValue.value) {
    listeners["update:modelValue"] = handleModelValueUpdate;
  }
  if (!props.readonly) return listeners;
  const manifest = componentRegistry.get(node.value.type);
  const definitions = normalizeEventDefinitions(manifest?.events || []);
  definitions.forEach((eventItem) => {
    if (!eventItem?.name || eventItem.name === "click") return;
    listeners[eventItem.name] = (...args) => {
      const payload = args.length > 1 ? args : args[0];
      void runPreviewScript(eventItem.name, payload);
    };
  });
  return listeners;
});

/**
 * 编辑态组件事件监听
 */
const designEventListeners = computed(() => {
  if (props.readonly || !node.value) return {};
  if (!isTabsType.value) return {};
  return {
    "tab-click": handleTabsClick,
    "tab-change": handleTabsChange,
    "update:modelValue": handleTabsChange,
    "tab-remove": handleTabsRemove,
    edit: handleTabsEdit,
  };
});

/**
 * 合并事件监听
 */
const mergedEventListeners = computed(() => {
  return { ...componentEventListeners.value, ...designEventListeners.value };
});

const tryRegisterPreviewRef = (pageIdValue) => {
  if (!props.readonly || !node.value?.label) return false;
  const runtime = getPreviewRuntime();
  if (!runtime?.registerComponentRef) return false;
  if (!pageIdValue) return false;
  const refInfo = buildRefInfo();
  if (!refInfo) return false;
  runtime.registerComponentRef(pageIdValue, node.value.label, refInfo);
  const pageId = currentPage.value?.id;
  const pageName = currentPage.value?.name;
  if (pageId && pageId !== pageIdValue) {
    runtime.registerComponentRef(pageId, node.value.label, refInfo);
  }
  if (pageName && pageName !== pageIdValue) {
    runtime.registerComponentRef(pageName, node.value.label, refInfo);
  }
  return true;
};

const scheduleRegisterPreviewRef = (pageIdValue) => {
  if (tryRegisterPreviewRef(pageIdValue)) return;
  if (registerAttempts >= maxRegisterAttempts) return;
  registerAttempts += 1;
  if (registerTimer) clearTimeout(registerTimer);
  registerTimer = setTimeout(() => {
    scheduleRegisterPreviewRef(pageIdValue);
  }, 120);
};

const unregisterPreviewRef = (label, pageIdValue = previewPageId.value) => {
  if (!props.readonly || !label) return;
  const runtime = getPreviewRuntime();
  if (!runtime?.unregisterComponentRef) return;
  if (!pageIdValue) return;
  const refInfo = buildRefInfo();
  runtime.unregisterComponentRef(pageIdValue, label, refInfo);
  const pageId = currentPage.value?.id;
  const pageName = currentPage.value?.name;
  if (pageId && pageId !== pageIdValue) {
    runtime.unregisterComponentRef(pageId, label, refInfo);
  }
  if (pageName && pageName !== pageIdValue) {
    runtime.unregisterComponentRef(pageName, label, refInfo);
  }
};

watch(
  () => node.value?.label,
  (next, prev) => {
    if (!props.readonly) return;
    if (prev && prev !== next) {
      unregisterPreviewRef(prev);
    }
    if (next) {
      registerAttempts = 0;
      scheduleRegisterPreviewRef(previewPageId.value);
    }
  },
);

watch(
  () => previewPageId.value,
  (next, prev) => {
    if (!props.readonly) return;
    if (prev && node.value?.label) {
      unregisterPreviewRef(node.value.label, prev);
    }
    if (next && node.value?.label) {
      registerAttempts = 0;
      scheduleRegisterPreviewRef(next);
    }
  },
);

onMounted(() => {
  registerAttempts = 0;
  scheduleRegisterPreviewRef(previewPageId.value);
});

onBeforeUnmount(() => {
  if (registerTimer) {
    clearTimeout(registerTimer);
    registerTimer = null;
  }
  if (node.value?.label) unregisterPreviewRef(node.value.label);
});


/**
 * 处理拖拽离开
 */
const handleDragLeave = () => {
  if (props.readonly) return;
  isDragOver.value = false;
  showInsertLine.value = false;
  insertLineStyle.value = null;
  rowInsertInfo.value = null;
  layoutInsertInfo.value = null;
};

</script>

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
  outline: none;
  background-color: transparent;
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

.designer-node.layout-container-visible:not(.is-preview)>div {
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  min-height: 60px;
}

.designer-node.layout-container-visible:not(.is-preview):hover>div {
  border-color: #a0a8b4;
}

.designer-node.layout-container-visible.is-selected>div {
  border-color: #409eff;
}

.insert-line {
  position: absolute;
  background: #ef4444;
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
  background: #ffffff;
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
