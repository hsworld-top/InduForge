<template>
  <div
    v-if="node && !node.hidden"
    :class="nodeClass"
    :style="wrapperStyle"
    :data-node-id="node.id"
    :data-node-type="node.type"
    ref="nodeRef"
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
      :key="renderKey"
      :style="contentStyle"
      v-bind="resolvedProps"
      v-on="componentEventListeners"
      ref="contentRef"
    >
      <template v-if="displayContent !== null">{{ displayContent }}</template>
      <template v-if="node?.type === 'Select'">
        <el-option
          v-for="option in selectOptions"
          :key="option.value ?? option.label"
          :label="option.label"
          :value="option.value"
        />
      </template>
      <template v-if="node?.type === 'Radio'">
        <el-radio
          v-for="option in radioOptions"
          :key="option.value ?? option.label"
          :label="option.value"
        >
          {{ option.label }}
        </el-radio>
      </template>
      <template v-if="node?.type === 'Checkbox'">
        <el-checkbox
          v-for="option in checkboxOptions"
          :key="option.value ?? option.label"
          :label="option.value"
        >
          {{ option.label }}
        </el-checkbox>
      </template>
      <template v-if="node?.type === 'Table'">
        <el-table-column
          v-for="column in tableColumns"
          :key="column.prop ?? column.label"
          v-bind="column"
        />
      </template>
      <template v-if="node?.type === 'BigDataTable'">
        <el-table-column
          v-for="column in bigTableColumns"
          :key="column.prop ?? column.label"
          v-bind="column"
        />
      </template>
      <template v-if="node?.type === 'Menu'">
        <el-menu-item
          v-for="item in menuItems"
          :key="item.index ?? item.label"
          :index="item.index ?? item.label"
        >
          {{ item.label }}
        </el-menu-item>
      </template>
      <template v-if="node?.type === 'Timeline'">
        <el-timeline-item
          v-for="item in timelineItems"
          :key="item.timestamp ?? item.label"
          :timestamp="item.timestamp"
        >
          {{ item.label }}
        </el-timeline-item>
      </template>
      <template v-if="node?.type === 'Tabs'">
        <el-tab-pane
          v-for="tab in tabsList"
          :key="tab.name ?? tab.label"
          :label="tab.label"
          :name="tab.name"
        >
          {{ tab.content }}
        </el-tab-pane>
      </template>
      <template v-if="node?.type === 'Collapse'">
        <el-collapse-item
          v-for="item in collapseItems"
          :key="item.name ?? item.title"
          :name="item.name"
          :title="item.title"
        >
          {{ item.content }}
        </el-collapse-item>
      </template>
      <template v-if="node?.type === 'Steps'">
        <el-step
          v-for="item in stepsItems"
          :key="item.title"
          :title="item.title"
          :description="item.description"
        />
      </template>
      <template
        v-if="
          node?.type === 'ImageCarousel' || node?.type === 'CarouselComponent'
        "
      >
        <el-carousel-item v-for="item in carouselItems" :key="item.label">
          <div class="carousel-item-placeholder">{{ item.label }}</div>
        </el-carousel-item>
      </template>
      <template v-if="node?.type === 'Dropdown'" #default>
        <el-button size="small" type="primary">{{ dropdownLabel }}</el-button>
      </template>
      <template v-if="node?.type === 'Dropdown'" #dropdown>
        <el-dropdown-menu>
          <el-dropdown-item
            v-for="item in dropdownItems"
            :key="item.value ?? item.label"
            :command="item.value"
          >
            {{ item.label }}
          </el-dropdown-item>
        </el-dropdown-menu>
      </template>
      <template v-if="isContainer && !hasChildren && !props.isRoot">
        <div
          class="empty-container-hint"
          :class="{ 'is-region-hint': isRegionContainer }"
        >
          <span v-if="isDropActive">释放以添加组件</span>
          <span v-else>{{
            isRegionContainer ? regionHintText : "拖拽组件到此处"
          }}</span>
        </div>
      </template>
      <!-- 插入线指示器 -->
      <div
        v-if="showInsertLine && insertLineStyle"
        class="insert-line"
        :class="insertLineStyle.orientation"
        :style="{
          [insertLineStyle.orientation === 'horizontal' ? 'top' : 'left']:
            insertLineStyle.offset + 'px',
        }"
      />
      <NodeRenderer
        v-for="childId in node.children || []"
        :key="childId"
        :node-id="childId"
        :readonly="props.readonly"
      />
    </component>

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
  </div>
</template>

<script setup>
import { computed, ref, inject, onBeforeUnmount, onMounted, watch } from "vue";
import { storeToRefs } from "pinia";
import { useEditorStore } from "@/stores/editor-store";
import { datacenterApi } from "@/services";
import { getPreviewRuntime } from "@/ui/Preview/previewRuntime";
import {
  componentRegistry,
  createSelectableElement,
  UpdateNodeCommand,
  MoveNodeCommand,
} from "@/editor-core";
import { normalizeEventDefinitions } from "@/editor-core/registry/componentEvents.js";
import { createDragDropManager } from "./DragDropManager";
import {
  useDragState,
  startDrag,
  endDrag,
  updateDropTarget,
  clearDropTarget,
} from "./use-drag-state";

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
} = storeToRefs(editorStore);
const canvasZoom = inject("canvasZoom", ref(1));
const dragState = useDragState();
const nodeRef = ref(null);
const contentRef = ref(null);
const previewPageId = computed(
  () => currentPage.value?.name || currentPage.value?.id || ""
);
let registerTimer = null;
let registerAttempts = 0;
const maxRegisterAttempts = 10;

const node = computed(() => {
  docVersion.value;
  return doc.value?.getNode(props.nodeId) || null;
});
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
const fallbackTableData = [
  { name: "张三", age: 28, address: "上海" },
  { name: "李四", age: 32, address: "北京" },
];
const fallbackBigTableColumns = [
  { label: "名称", prop: "name" },
  { label: "数值", prop: "value" },
];
const fallbackBigTableData = [
  { name: "张三", value: 100 },
  { name: "李四", value: 200 },
];
const fallbackTimelineItems = [
  { label: "步骤一", timestamp: "2024-01-01" },
  { label: "步骤二", timestamp: "2024-01-02" },
  { label: "步骤三", timestamp: "2024-01-03" },
];
const fallbackTabs = [
  { name: "tab1", label: "标签一", content: "内容一" },
  { name: "tab2", label: "标签二", content: "内容二" },
  { name: "tab3", label: "标签三", content: "内容三" },
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

/**
 * 规范化选项列表
 * @param {Array} source - 原始列表
 * @param {Array} fallback - 默认列表
 * @returns {Array}
 */
const normalizeOptions = (source, fallback) => {
  if (Array.isArray(source)) return source;
  return fallback;
};

/**
 * 过滤用于渲染的组件属性
 * @param {string} type - 组件类型
 * @param {Record<string, any>} props - 原始属性
 * @returns {Record<string, any>}
 */
const filterRenderProps = (type, props) => {
  const nextProps = { ...props };
  const removeKeysByType = {
    Button: ["text"],
    Text: ["text"],
    Tag: ["text"],
    Dropdown: ["label", "items"],
    Menu: ["items"],
    Tabs: ["tabs"],
    Table: ["columns"],
    BigDataTable: ["columns"],
    Radio: ["options"],
    Checkbox: ["options"],
    Select: ["options"],
    Timeline: ["items"],
    Steps: ["items"],
    Collapse: ["items"],
    ImageCarousel: ["items"],
    CarouselComponent: ["items"],
    Card: ["title", "content"],
    BusinessCard: ["title", "content"],
    ElContainer: ["showHeader", "showAside", "showMain", "showFooter"],
    ElLayout: ["columns"],
  };
  const removeKeys = removeKeysByType[type] || [];
  for (const key of removeKeys) {
    delete nextProps[key];
  }
  if (type === "Table") {
    nextProps.data = normalizeOptions(nextProps.data, fallbackTableData);
  }
  if (type === "BigDataTable") {
    nextProps.data = normalizeOptions(nextProps.data, fallbackBigTableData);
  }
  if (type === "Signature") {
    nextProps.type = "textarea";
    if (!nextProps.rows) {
      nextProps.rows = 3;
    }
  }
  if (type === "ImageCarousel" || type === "CarouselComponent") {
    if (!nextProps.height) {
      nextProps.height = "160px";
    }
  }
  return nextProps;
};

/** 拖拽状态 */
const isDragOver = ref(false);
const isDropActive = computed(() => {
  if (!node.value) return isDragOver.value;
  return isDragOver.value || dragState.targetContainerId === node.value.id;
});
const showInsertLine = ref(false);
const insertLineStyle = ref(null);
const dragDropManager = createDragDropManager();
const tableRenderVersion = ref(0);
const resolvedProps = computed(() => {
  if (!node.value) return {};
  if (node.value.type === "Table" || node.value.type === "BigDataTable") {
    tableRenderVersion.value;
  }
  return filterRenderProps(node.value.type, node.value.props || {});
});

const selectOptions = computed(() => {
  return normalizeOptions(node.value?.props?.options, fallbackSelectOptions);
});
const radioOptions = computed(() => {
  return normalizeOptions(node.value?.props?.options, fallbackRadioOptions);
});
const checkboxOptions = computed(() => {
  return normalizeOptions(node.value?.props?.options, fallbackCheckboxOptions);
});
const dropdownItems = computed(() => {
  return normalizeOptions(node.value?.props?.items, fallbackDropdownItems);
});
const menuItems = computed(() => {
  return normalizeOptions(node.value?.props?.items, fallbackMenuItems);
});
const tableColumns = computed(() => {
  tableRenderVersion.value;
  return normalizeOptions(node.value?.props?.columns, fallbackTableColumns);
});
const bigTableColumns = computed(() => {
  tableRenderVersion.value;
  return normalizeOptions(node.value?.props?.columns, fallbackBigTableColumns);
});
const timelineItems = computed(() => {
  return normalizeOptions(node.value?.props?.items, fallbackTimelineItems);
});
const tabsList = computed(() => {
  return normalizeOptions(node.value?.props?.tabs, fallbackTabs);
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

// ✅ 注入右键菜单显示函数
const showContextMenu = inject("showContextMenu", null);

const isContainer = computed(() => {
  if (!node.value) return false;
  const manifest = componentRegistry.get(node.value.type);
  return Boolean(manifest?.isContainer);
});

/**
 * 判断是否为 Flex 插入容器
 * @param {string} type - 组件类型
 * @returns {boolean}
 */
const isFlexDropContainer = (type) => {
  return [
    "FlexContainer",
    "ResponsiveLayout",
    "ElContainer",
    "ElLayout",
    "ElHeader",
    "ElAside",
    "ElMain",
    "ElFooter",
    "ElCol",
  ].includes(type);
};

/**
 * 获取 Flex 方向
 * @param {string} type - 组件类型
 * @param {HTMLElement} element - 目标元素
 * @returns {string}
 */
const resolveFlexDirection = (type, element) => {
  if (type === "FlexContainer" || type === "ResponsiveLayout") {
    return node.value?.props?.direction || "column";
  }
  return dragDropManager.getContainerDirection(element);
};

const hasChildren = computed(() => {
  docVersion.value;
  if (!node.value || !doc.value) return false;
  if (typeof doc.value.getChildren === "function") {
    return doc.value.getChildren(node.value.id).length > 0;
  }
  return (node.value.children || []).length > 0;
});

/**
 * 判断节点是否可自由拖动
 * @returns {boolean} 是否允许拖动
 * @throws {Error} 无
 */
const isMovable = computed(() => {
  if (!node.value || props.isRoot || node.value.locked) return false;
  if (
    node.value.type === "ElHeader" ||
    node.value.type === "ElAside" ||
    node.value.type === "ElMain" ||
    node.value.type === "ElFooter"
  ) {
    return false;
  }
  return true;
});

const resizeHandles = [
  { key: "nw", x: -1, y: -1, cursor: "nwse-resize" },
  { key: "n", x: 0, y: -1, cursor: "ns-resize" },
  { key: "ne", x: 1, y: -1, cursor: "nesw-resize" },
  { key: "e", x: 1, y: 0, cursor: "ew-resize" },
  { key: "se", x: 1, y: 1, cursor: "nwse-resize" },
  { key: "s", x: 0, y: 1, cursor: "ns-resize" },
  { key: "sw", x: -1, y: 1, cursor: "nesw-resize" },
  { key: "w", x: -1, y: 0, cursor: "ew-resize" },
];

const getRegionResizeConfig = (type) => {
  const configMap = {
    ElHeader: { axis: "y", prop: "height", handles: ["s"] },
    ElAside: { axis: "x", prop: "width", handles: ["e"] },
  };
  if (type === "ElFooter") {
    return {
      axis: "y",
      prop: "height",
      handles: ["n"],
      invert: true,
    };
  }
  return configMap[type] || null;
};

const visibleResizeHandles = computed(() => {
  const config = getRegionResizeConfig(node.value?.type);
  if (!config) return resizeHandles;
  return resizeHandles.filter((handle) => config.handles.includes(handle.key));
});

const showResizeHandles = computed(() => {
  selectionVersion.value;
  if (props.readonly || props.isRoot) return false;
  if (!node.value || node.value.locked) return false;
  if (node.value.type === "ElMain") return false;
  const parentNode = doc.value?.getParent?.(node.value.id);
  if (
    parentNode?.type === "ElHeader" ||
    parentNode?.type === "ElAside" ||
    parentNode?.type === "ElMain" ||
    parentNode?.type === "ElFooter"
  ) {
    return false;
  }
  const config = getRegionResizeConfig(node.value?.type);
  if (config && config.handles.length === 0) return false;
  return Boolean(selection.value?.isSelected?.(node.value.id));
});

/**
 * 判断容器是否允许子组件
 * @param {import('@/editor-core').ComponentNode} parentNode - 父节点
 * @param {string} childType - 子组件类型
 * @returns {boolean}
 */
const canAcceptChild = (parentNode, childType) => {
  const manifest = componentRegistry.get(parentNode.type);
  const allowed = manifest?.allowedChildren;
  if (
    parentNode.type === "ElHeader" ||
    parentNode.type === "ElAside" ||
    parentNode.type === "ElMain" ||
    parentNode.type === "ElFooter"
  ) {
    return (parentNode.children || []).length === 0;
  }
  if (!Array.isArray(allowed) || allowed.length === 0) return true;
  return allowed.includes(childType);
};

/**
 * 计算落点相对坐标（考虑缩放）
 * @param {DragEvent} event - 拖拽事件
 * @param {HTMLElement} element - 目标容器
 * @returns {{ x: number, y: number }}
 */
const resolveDropOffset = (event, element) => {
  const rect = element.getBoundingClientRect();
  const zoomValue = Number(canvasZoom?.value) || 1;
  const offsetX = (event.clientX - rect.left) / zoomValue;
  const offsetY = (event.clientY - rect.top) / zoomValue;
  return {
    x: Math.max(0, Math.round(offsetX)),
    y: Math.max(0, Math.round(offsetY)),
  };
};

const nodeClass = computed(() => {
  selectionVersion.value;
  if (!node.value) return "";
  const classes = ["designer-node"];
  if (props.isRoot) classes.push("is-root");
  if (props.readonly) classes.push("is-preview");
  if (isContainer.value) classes.push("is-container");
  if (isMovable.value && !props.readonly) classes.push("is-draggable");
  if (node.value.locked) classes.push("is-locked");
  if (isDropActive.value && !props.readonly) classes.push("drag-over");
  if (!props.readonly && selection.value?.isSelected(node.value.id)) {
    classes.push("is-selected");
  }
  return classes.join(" ");
});

const isRegionContainer = computed(() => {
  const type = node.value?.type;
  return (
    type === "ElHeader" ||
    type === "ElAside" ||
    type === "ElMain" ||
    type === "ElFooter"
  );
});

const regionHintText = computed(() => {
  if (!node.value) return "";
  const hintMap = {
    ElHeader: "Header区域",
    ElAside: "Aside区域",
    ElMain: "Main区域",
    ElFooter: "Footer区域",
  };
  return hintMap[node.value.type] || "区域";
});

const renderTag = computed(() => {
  if (!node.value) return "div";
  switch (node.value.type) {
    case "Button":
      return "el-button";
    case "Input":
      return "el-input";
    case "Select":
      return "el-select";
    case "InputNumber":
      return "el-input-number";
    case "Switch":
      return "el-switch";
    case "Table":
      return "el-table";
    case "BigDataTable":
      return "el-table";
    case "Tree":
      return "el-tree";
    case "Transfer":
      return "el-transfer";
    case "Tag":
      return "el-tag";
    case "Dropdown":
      return "el-dropdown";
    case "Menu":
      return "el-menu";
    case "Radio":
      return "el-radio-group";
    case "Checkbox":
      return "el-checkbox-group";
    case "Cascader":
      return "el-cascader";
    case "Image":
      return "el-image";
    case "Tabs":
      return "el-tabs";
    case "Timeline":
      return "el-timeline";
    case "ImageCarousel":
      return "el-carousel";
    case "CarouselComponent":
      return "el-carousel";
    case "WebContainer":
      return "el-card";
    case "Steps":
      return "el-steps";
    case "Card":
      return "el-card";
    case "Pagination":
      return "el-pagination";
    case "Collapse":
      return "el-collapse";
    case "BusinessCard":
      return "el-card";
    case "Barcode":
      return "el-card";
    case "Slider":
      return "el-slider";
    case "Calendar":
      return "el-calendar";
    case "Signature":
      return "el-input";
    case "ElContainer":
      return "el-container";
    case "ElHeader":
      return "el-header";
    case "ElAside":
      return "el-aside";
    case "ElMain":
      return "el-main";
    case "ElFooter":
      return "el-footer";
    case "ElLayout":
      return "el-row";
    case "ElCol":
      return "el-col";
    case "Text":
      return "div";
    default:
      return "div";
  }
});

const renderKey = computed(() => {
  if (!node.value) return "";
  if (node.value.type === "Table" || node.value.type === "BigDataTable") {
    const columnsSize = Array.isArray(node.value.props?.columns)
      ? node.value.props.columns.length
      : 0;
    const dataSize = Array.isArray(node.value.props?.data)
      ? node.value.props.data.length
      : 0;
    return `${node.value.id}-${columnsSize}-${dataSize}-${tableRenderVersion.value}`;
  }
  return node.value.id || "";
});

const displayContent = computed(() => {
  docVersion.value;
  if (!node.value) return null;
  if (node.value.type === "Text") {
    return node.value.props?.text ?? node.value.label ?? "";
  }
  if (node.value.type === "Button") {
    return node.value.props?.text ?? node.value.label ?? "按钮";
  }
  if (node.value.type === "Tag") {
    return node.value.props?.text ?? node.value.label ?? "标签";
  }
  if (node.value.type === "Card") {
    return node.value.props?.content ?? node.value.label ?? "卡片";
  }
  if (node.value.type === "BusinessCard") {
    return node.value.props?.content ?? node.value.label ?? "业务卡片";
  }
  if (node.value.type === "WebContainer") {
    return node.value.props?.url
      ? `网页容器: ${node.value.props.url}`
      : "网页容器";
  }
  if (node.value.type === "Barcode") {
    return node.value.props?.value ?? "1234567890";
  }
  return null;
});

const layoutStyle = computed(() => {
  docVersion.value;
  if (!node.value) return {};
  return resolveLayoutStyle(node.value, props.isRoot);
});

const contentStyle = computed(() => {
  docVersion.value;
  if (!node.value) return {};
  const containerStyle = resolveContainerStyle(node.value, {});
  const customStyle = normalizeStyleObject(node.value.style || {});
  const parentNode = doc.value?.getParent?.(node.value.id);
  const style = {
    ...containerStyle,
    ...customStyle,
  };
  if (!style.overflow && !isContainer.value) {
    style.overflow = "hidden";
  }
  if (
    parentNode?.type === "ElHeader" ||
    parentNode?.type === "ElAside" ||
    parentNode?.type === "ElMain" ||
    parentNode?.type === "ElFooter"
  ) {
    style.overflow = "hidden";
  }
  if (
    node.value.type === "ElContainer" ||
    node.value.type === "ElHeader" ||
    node.value.type === "ElAside" ||
    node.value.type === "ElMain" ||
    node.value.type === "ElFooter"
  ) {
    style.padding = "0";
  }
  if (parentNode?.type === "ElContainer") {
    if (node.value.type === "ElHeader") {
      style.width = "100%";
      style.height = parentNode.props?.headerHeight || "60px";
    } else if (node.value.type === "ElFooter") {
      style.width = "100%";
      style.height = parentNode.props?.footerHeight || "60px";
    } else if (node.value.type === "ElAside") {
      style.width = parentNode.props?.asideWidth || "200px";
      style.height = "100%";
    }
  }
  if (
    !style.width &&
    (parentNode?.type === "ElHeader" ||
      parentNode?.type === "ElAside" ||
      parentNode?.type === "ElMain" ||
      parentNode?.type === "ElFooter")
  ) {
    style.width = "100%";
  }
  if (
    !style.height &&
    (parentNode?.type === "ElHeader" ||
      parentNode?.type === "ElAside" ||
      parentNode?.type === "ElMain" ||
      parentNode?.type === "ElFooter")
  ) {
    style.height = "100%";
  }
  if (props.isRoot || layoutStyle.value.position === "absolute") {
    if (!style.width) style.width = "100%";
    if (!style.height) style.height = "100%";
  }
  return style;
});

const wrapperStyle = computed(() => {
  if (!node.value) return {};
  const style = { ...layoutStyle.value };
  const customStyle = normalizeStyleObject(node.value.style || {});
  if (customStyle.width && !style.width) {
    style.width = customStyle.width;
  }
  if (customStyle.height && !style.height) {
    style.height = customStyle.height;
  }
  return style;
});

/**
 * 处理节点选中逻辑
 * @param {MouseEvent} event - 鼠标事件
 */
const normalizeGlobalValue = (detail) => {
  const type = detail?.type;
  const raw = detail?.default;
  if (type === "function") {
    if (typeof raw === "function") return raw;
    if (typeof raw === "string") {
      const text = raw.trim();
      if (!text) return () => undefined;
      try {
        // Treat as function expression or async function expression
        if (
          text.startsWith("function") ||
          text.startsWith("async function") ||
          text.startsWith("(") ||
          text.startsWith("async (") ||
          text.startsWith("async(")
        ) {
          return new Function(`return (${text});`)();
        }
        // Treat as function body
        return new Function(text);
      } catch (error) {
        return () => undefined;
      }
    }
    return () => undefined;
  }
  if (type === "set") {
    if (raw instanceof Set) return raw;
    if (Array.isArray(raw)) return new Set(raw);
    if (typeof raw === "string") {
      try {
        const parsed = JSON.parse(raw);
        return new Set(Array.isArray(parsed) ? parsed : []);
      } catch (error) {
        return new Set();
      }
    }
    return new Set();
  }
  if (type === "map") {
    if (raw instanceof Map) return raw;
    if (Array.isArray(raw)) return new Map(raw);
    if (raw && typeof raw === "object") return new Map(Object.entries(raw));
    if (typeof raw === "string") {
      try {
        const parsed = JSON.parse(raw);
        if (Array.isArray(parsed)) return new Map(parsed);
        if (parsed && typeof parsed === "object") {
          return new Map(Object.entries(parsed));
        }
      } catch (error) {
        return new Map();
      }
    }
    return new Map();
  }
  if (type === "regexp") {
    if (raw instanceof RegExp) return raw;
    if (typeof raw === "string") {
      try {
        const match = raw.match(/^\/(.*)\/([gimsuy]*)$/);
        if (match) return new RegExp(match[1], match[2]);
        return new RegExp(raw);
      } catch (error) {
        return null;
      }
    }
  }
  return raw ?? null;
};

const connectionCache = new Map();
const queryCache = new Map();
const mappedValueCache = new Map();
const previewOverrides = new Map();

const unwrapApiData = (payload) => {
  if (payload && typeof payload === "object" && "data" in payload) {
    return payload.data;
  }
  return payload;
};

const resolveConnection = async (name) => {
  if (connectionCache.has(name)) return connectionCache.get(name);
  if (!projectId.value) return null;
  const result = await datacenterApi.getConnections(projectId.value, {
    page: 1,
    limit: 200,
  });
  const data = unwrapApiData(result) || {};
  const connections = data.connections || data.items || data.list || [];
  const found = connections.find((item) => item.name === name);
  if (found) {
    connectionCache.set(name, found);
  }
  return found || null;
};

const resolveQuery = async (connectionId, queryName) => {
  if (!projectId.value) return null;
  const cacheKey = `${connectionId}`;
  let queries = queryCache.get(cacheKey);
  if (!queries) {
    const result = await datacenterApi.getQueries(projectId.value, {
      connectionId,
      page: 1,
      limit: 200,
    });
    const data = unwrapApiData(result) || {};
    queries = data.queries || data.items || data.list || [];
    queryCache.set(cacheKey, queries);
  }
  return (
    queries.find((item) => item.name === queryName || item.id === queryName) ||
    null
  );
};

const executeQueryByPath = async (path) => {
  const [sourceName, ...rest] = String(path || "").split(".");
  const field = rest.join(".");
  if (!sourceName || !field) return undefined;
  const connection = await resolveConnection(sourceName);
  if (!connection || connection.type !== "relational") return undefined;
  const query = await resolveQuery(connection.id, field);
  if (!query) return undefined;
  const result = await datacenterApi.executeQuery(query.id);
  const payload = unwrapApiData(result) || result;
  return payload?.data ?? payload;
};

const resolveMappedGlobalValue = async (name, detail) => {
  const source = detail?.source;
  if (!source || source.type !== "dataCenter" || !source.path) {
    return normalizeGlobalValue(detail);
  }
  if (!projectId.value) return normalizeGlobalValue(detail);

  if (source.datapointId || source.sourceType || source.sourceId) {
    const sourceType = String(source.sourceType || "");
    if (sourceType.includes("query") && source.sourceId) {
      try {
        const result = await datacenterApi.executeQuery(source.sourceId);
        const payload = unwrapApiData(result) || result;
        return payload?.data ?? payload;
      } catch (error) {
        try {
          const fallbackResult = await executeQueryByPath(source.path);
          if (fallbackResult !== undefined) return fallbackResult;
        } catch (fallbackError) {
          // ignore
        }
        return normalizeGlobalValue(detail);
      }
    }
    if (source.datapointId) {
      const result = await datacenterApi.getDatapointValues(projectId.value, [
        source.datapointId,
      ]);
      const payload = unwrapApiData(result) || result;
      const picked = extractDatapointValue(payload, source.datapointId);
      return picked ?? payload?.data ?? payload;
    }
  }

  const [sourceName, ...rest] = String(source.path).split(".");
  const field = rest.join(".");
  if (!sourceName || !field) return normalizeGlobalValue(detail);

  const connection = await resolveConnection(sourceName);
  if (!connection) return normalizeGlobalValue(detail);

  if (connection.type === "relational") {
    const query = await resolveQuery(connection.id, field);
    if (!query) return normalizeGlobalValue(detail);
    const result = await datacenterApi.executeQuery(query.id);
    const payload = unwrapApiData(result) || result;
    return payload?.data ?? payload;
  }

  return normalizeGlobalValue(detail);
};

const buildPreviewGlobals = () => {
  const vars = projectVariables.value || {};
  return new Proxy(
    {},
    {
      get(_target, prop) {
        if (typeof prop !== "string") return undefined;
        if (previewOverrides.has(prop)) {
          return previewOverrides.get(prop);
        }
        const detail = vars[prop];
        if (!detail) return undefined;
        if (detail?.mapped && detail?.source?.type === "dataCenter") {
          if (!mappedValueCache.has(prop)) {
            const promise = resolveMappedGlobalValue(prop, detail).catch(
              () => null
            );
            mappedValueCache.set(prop, promise);
          }
          return mappedValueCache.get(prop);
        }
        return normalizeGlobalValue(detail);
      },
      set(_target, prop, value) {
        if (typeof prop !== "string") return false;
        previewOverrides.set(prop, value);
        return true;
      },
    }
  );
};

const parseParamNames = (value) => {
  if (!value || typeof value !== "string") return [];
  return value
    .split(",")
    .map((name) => name.trim())
    .filter((name) => /^[A-Za-z_$][\w$]*$/.test(name));
};

const buildPreviewCustomScripts = (globals) => {
  const items = globalScripts.value?.custom?.items || [];
  const handlers = {};
  items.forEach((item) => {
    if (!item?.name) return;
    const paramNames = parseParamNames(item.params || item.args);
    handlers[item.name] = async (...args) => {
      const code = item.code || "";
      if (!code.trim()) return undefined;
      const scope = {
        $global: globals,
        customScripts: handlers,
        console,
        $event: undefined,
      };
      const localKeys = [...paramNames, ...Object.keys(scope)];
      const localValues = [
        ...paramNames.map((_, index) => args[index]),
        ...Object.values(scope),
      ];
      try {
        const runner = new Function(
          ...localKeys,
          `"use strict";\nreturn (async () => {\n${code}\n})();`
        );
        return await runner(...localValues);
      } catch (error) {
        console.error(`[Preview] customScripts.${item.name} error:`, error);
        return undefined;
      }
    };
  });
  return handlers;
};

const runPreviewScript = async (eventName, event) => {
  if (!node.value) return;
  const handlers = node.value?.events?.[eventName];
  if (!Array.isArray(handlers) || handlers.length === 0) return;
  const action = handlers.find((item) => item?.type === "script" || item?.code);
  if (!action) return;
  if (action?.enabled === false) return;
  const code = typeof action === "string" ? action : action?.code || "";
  if (!code.trim()) return;

  const runtime = getPreviewRuntime();
  if (runtime?.runCode) {
    const pageId = currentPage.value?.name || currentPage.value?.id;
    const instance = buildRefInfo();
    return await runtime.runCode(code, event, instance, pageId);
  }
  const instance = buildRefInfo();
  const globals = buildPreviewGlobals();
  const customScripts = buildPreviewCustomScripts(globals);
  const context = {
    $event: event,
    $global: globals,
    customScripts,
    console,
  };

  try {
    const keys = Object.keys(context);
    const values = Object.values(context);
    const runner = new Function(
      ...keys,
      `"use strict";\nreturn (async function() {\n${code}\n}).call(this);`
    );
    return await runner.call(instance || null, ...values);
  } catch (error) {
    console.error("[Preview] Script error:", error);
  }
};

const extractDatapointValue = (payload, datapointId) => {
  if (!payload || !datapointId) return null;
  if (payload.values && typeof payload.values === "object") {
    if (datapointId in payload.values) return payload.values[datapointId];
  }
  if (Array.isArray(payload.values)) {
    const hit = payload.values.find((item) => item?.id === datapointId);
    if (hit)
      return (
        hit.value ??
        hit.currentValue ??
        hit.dataValue ??
        hit.lastValue ??
        hit.rawValue
      );
  }
  if (Array.isArray(payload.datapoints)) {
    const hit = payload.datapoints.find((item) => item?.id === datapointId);
    if (hit)
      return (
        hit.value ??
        hit.currentValue ??
        hit.dataValue ??
        hit.lastValue ??
        hit.rawValue
      );
  }
  if (Array.isArray(payload)) {
    const hit = payload.find((item) => item?.id === datapointId);
    if (hit)
      return (
        hit.value ??
        hit.currentValue ??
        hit.dataValue ??
        hit.lastValue ??
        hit.rawValue
      );
  }
  if (payload && typeof payload === "object" && datapointId in payload) {
    return payload[datapointId];
  }
  return null;
};

const componentEventListeners = computed(() => {
  if (!props.readonly || !node.value) return {};
  const manifest = componentRegistry.get(node.value.type);
  const definitions = normalizeEventDefinitions(manifest?.events || []);
  const listeners = {};
  definitions.forEach((eventItem) => {
    if (!eventItem?.name || eventItem.name === "click") return;
    listeners[eventItem.name] = (...args) => {
      const payload = args.length > 1 ? args : args[0];
      void runPreviewScript(eventItem.name, payload);
    };
  });
  return listeners;
});

const handleSelect = (event) => {
  if (!node.value || !selection.value) return;
  // 锁定的节点不能选中
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
};

/**
 * 解析点击时的选中目标
 * @param {MouseEvent} event - 鼠标事件
 * @returns {import('@/editor-core').ComponentNode | null}
 */
const resolveClickSelectionTarget = (event) => {
  if (!node.value) return null;
  if (!isRegionContainer.value) return node.value;
  if (event?.altKey) return node.value;
  const parentNode = doc.value?.getParent?.(node.value.id);
  if (!parentNode || parentNode.type !== "ElContainer") return node.value;
  if (parentNode.locked) return node.value;
  return parentNode;
};

const handleClick = (event) => {
  if (props.readonly) {
    void runPreviewScript("click", event);
    return;
  }
  let targetNode = resolveClickSelectionTarget(event);
  if (
    targetNode === node.value &&
    !isRegionContainer.value &&
    !event?.altKey
  ) {
    const parentNode = doc.value?.getParent?.(node.value.id);
    if (
      parentNode &&
      (parentNode.type === "ElHeader" ||
        parentNode.type === "ElAside" ||
        parentNode.type === "ElMain" ||
        parentNode.type === "ElFooter")
    ) {
      const containerNode = doc.value?.getParent?.(parentNode.id);
      if (containerNode?.type === "ElContainer" && !containerNode.locked) {
        targetNode = containerNode;
      }
    }
  }
  if (!targetNode || !selection.value) return;
  const element = createSelectableElement("node", targetNode.id);
  if (event.shiftKey) {
    selection.value.selectRange(element);
    return;
  }
  if (event.metaKey || event.ctrlKey) {
    selection.value.toggleSelect(element);
    return;
  }
  selection.value.select(element);
};

const handleDoubleClick = (event) => {
  if (props.readonly) return;
  if (!node.value || !selection.value) return;
  if (!isRegionContainer.value) return;
  const parentNode = doc.value?.getParent?.(node.value.id);
  if (!parentNode || parentNode.type !== "ElContainer") return;
  if (parentNode.locked) return;
  const element = createSelectableElement("node", parentNode.id);
  selection.value.select(element);
};

const applyPreviewPatch = (patch) => {
  if (!node.value || !patch || typeof patch !== "object") return;
  if (patch.props) {
    node.value.props = { ...(node.value.props || {}), ...patch.props };
  }
  if (patch.style) {
    node.value.style = { ...(node.value.style || {}), ...patch.style };
  }
  if (patch.label !== undefined) {
    node.value.label = patch.label;
  }
  docVersion.value += 1;
};

const buildRefInfo = () => {
  if (!node.value) return null;
  return {
    name: node.value.label,
    id: node.value.id,
    el: nodeRef.value || null,
    component: contentRef.value || null,
    node: node.value,
    setProps: (patch) => {
      if (!patch || typeof patch !== "object") return;
      if (props.readonly) {
        applyPreviewPatch({ props: patch });
        return;
      }
      editorStore.updateNode(node.value.id, {
        props: { ...(node.value.props || {}), ...patch },
      });
    },
    setStyle: (patch) => {
      if (!patch || typeof patch !== "object") return;
      if (props.readonly) {
        applyPreviewPatch({ style: patch });
        return;
      }
      editorStore.updateNode(node.value.id, {
        style: { ...(node.value.style || {}), ...patch },
      });
    },
    setText: (text) => {
      const value = String(text ?? "");
      if (props.readonly) {
        const propsPatch = { text: value };
        if (
          node.value?.type === "Card" ||
          node.value?.type === "BusinessCard"
        ) {
          propsPatch.content = value;
        }
        applyPreviewPatch({ props: propsPatch });
        return;
      }
      editorStore.updateNode(node.value.id, {
        props: { ...(node.value.props || {}), text: value },
      });
    },
    setTableHeader: (columns) => {
      if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
        return;
      }
      if (columns && typeof columns.then === "function") {
        columns.then((resolved) => {
          refInfo.setTableHeader(resolved);
        });
        return;
      }
      let input = columns;
      if (input && typeof input === "object" && !Array.isArray(input)) {
        if (Array.isArray(input.columns)) {
          input = input.columns;
        } else if (Array.isArray(input.data)) {
          input = input.data;
        }
      }
      if (!Array.isArray(input)) return;
      const normalized = input
        .map((item) => {
          if (item && typeof item === "object") {
            const label = item.label ?? item.title ?? item.name ?? item.prop;
            const prop =
              item.prop ?? item.field ?? item.key ?? item.name ?? item.label;
            return { ...item, label, prop };
          }
          if (typeof item === "string") {
            return { label: item, prop: item };
          }
          return null;
        })
        .filter(Boolean);
      if (props.readonly) {
        applyPreviewPatch({ props: { columns: normalized } });
        tableRenderVersion.value += 1;
        return;
      }
      editorStore.updateNode(node.value.id, {
        props: { ...(node.value.props || {}), columns: normalized },
      });
      tableRenderVersion.value += 1;
    },
    setTableData: (data, header) => {
      if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
        return;
      }
      if (!data) return;
      if (data && typeof data.then === "function") {
        data.then((resolved) => {
          refInfo.setTableData(resolved, header);
        });
        return;
      }
      let rows = data;
      let columns = header || null;
      if (rows && typeof rows === "object" && !Array.isArray(rows)) {
        if (Array.isArray(rows.rows)) {
          rows = rows.rows;
        } else if (Array.isArray(rows.data)) {
          rows = rows.data;
        }
        if (Array.isArray(rows.columns)) {
          columns = rows.columns;
        } else if (Array.isArray(data.columns)) {
          columns = data.columns;
        }
      }
      if (!Array.isArray(rows)) return;
      const normalizeColumns = (input) => {
        if (!Array.isArray(input)) return [];
        return input
          .map((item) => {
            if (item && typeof item === "object") {
              const label = item.label ?? item.title ?? item.name ?? item.prop;
              const prop =
                item.prop ?? item.field ?? item.key ?? item.name ?? item.label;
              return { ...item, label, prop };
            }
            if (typeof item === "string") {
              return { label: item, prop: item };
            }
            return null;
          })
          .filter(Boolean);
      };
      const normalizedColumns = normalizeColumns(
        columns || node.value?.props?.columns || []
      );
      const normalizedData =
        Array.isArray(rows) && Array.isArray(rows[0])
          ? rows.map((row) => {
              if (!Array.isArray(row)) return row;
              if (normalizedColumns.length === 0) return row;
              const next = {};
              normalizedColumns.forEach((col, index) => {
                const key = col.prop ?? col.label ?? `col${index}`;
                next[key] = row[index];
              });
              return next;
            })
          : rows;
      if (props.readonly) {
        applyPreviewPatch({ props: { data: normalizedData } });
        tableRenderVersion.value += 1;
        return;
      }
      editorStore.updateNode(node.value.id, {
        props: { ...(node.value.props || {}), data: normalizedData },
      });
      tableRenderVersion.value += 1;
    },
  };
};

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
  }
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
  }
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
 * 处理右键菜单
 * @param {MouseEvent} event - 鼠标事件
 */
const handleContextMenu = (event) => {
  if (props.readonly) return;
  // ✅ 阻止浏览器默认右键菜单
  event.preventDefault();
  event.stopPropagation();

  // 如果当前节点未被选中，先选中它
  if (!node.value || !selection.value) return;
  if (node.value.locked) return;

  const selectedElements = selection.value.getSelectedElements?.() || [];
  const isSelected = selectedElements.some(
    (el) => el.id === node.value.id && el.kind === "node"
  );

  if (!isSelected) {
    const element = createSelectableElement("node", node.value.id);
    selection.value.select(element);
  }

  // ✅ 显示右键菜单
  if (showContextMenu) {
    showContextMenu(event);
  }
};

/**
 * 处理拖拽悬停
 * @param {DragEvent} event - 拖拽事件
 */
const handleDragOver = (event) => {
  if (props.readonly) return;
  // ✅ 阻止事件冒泡
  event.stopPropagation();

  if (!isContainer.value) return;
  const hasComponent =
    event.dataTransfer?.types?.includes("application/x-designer-component") ||
    event.dataTransfer?.types?.includes("text/plain") ||
    Boolean(dragState.dragType);
  if (!hasComponent) return;

  if (event.dataTransfer) {
    event.dataTransfer.dropEffect = "copy";
  }
  isDragOver.value = true;

  // 计算插入位置
  if (node.value?.type && isFlexDropContainer(node.value.type)) {
    const currentElement = event.currentTarget;
    const direction = resolveFlexDirection(node.value.type, currentElement);
    const insertInfo = dragDropManager.calculateFlexInsertPosition(
      currentElement,
      event,
      direction
    );

    if (insertInfo.insertLine) {
      showInsertLine.value = true;
      insertLineStyle.value = insertInfo.insertLine;
    }
  }
};

/**
 * 处理拖拽离开
 */
const handleDragLeave = () => {
  if (props.readonly) return;
  isDragOver.value = false;
  showInsertLine.value = false;
  insertLineStyle.value = null;
};

/**
 * 处理拖拽放置
 * @param {DragEvent} event - 拖拽事件
 */
const handleDrop = (event) => {
  if (props.readonly) return;
  // ✅ 阻止事件冒泡，避免重复插入
  event.stopPropagation();

  isDragOver.value = false;
  showInsertLine.value = false;
  insertLineStyle.value = null;

    const resolveElContainerTarget = () => {
      const isRegionType = (type) =>
        type === "ElHeader" ||
        type === "ElAside" ||
        type === "ElMain" ||
        type === "ElFooter";
      const resolveContainerMain = (containerNode) => {
        if (!containerNode || containerNode.type !== "ElContainer") return null;
        const mainChildId = (containerNode.children || []).find((childId) => {
          const childNode = doc.value?.getNode?.(childId);
          return childNode?.type === "ElMain";
        });
        if (!mainChildId) return null;
        return doc.value?.getNode?.(mainChildId) || null;
      };
      const resolveNodeFromId = (nodeId) => {
        if (!nodeId) return null;
        const targetNode = doc.value?.getNode?.(nodeId);
        if (!targetNode) return null;
        if (isRegionType(targetNode.type)) {
          return { node: targetNode, element: null };
        }
        if (targetNode.type === "ElContainer") {
          const mainNode = resolveContainerMain(targetNode) || targetNode;
          return { node: mainNode, element: null };
        }
        return null;
      };
      const resolveFromElement = (element) => {
        if (!element) return null;
        const nodeElement = element.closest?.("[data-node-id]");
        if (!nodeElement) return null;
        const nodeId = nodeElement.getAttribute("data-node-id");
        const resolved = resolveNodeFromId(nodeId);
        if (!resolved) return null;
        return { node: resolved.node, element: nodeElement };
      };

      const path = event.composedPath?.() || [];
      for (const item of path) {
        if (!(item instanceof Element)) continue;
        const resolved = resolveFromElement(item);
        if (resolved) return resolved;
      }

      const hit = document.elementFromPoint(event.clientX, event.clientY);
      const resolvedHit = resolveFromElement(hit);
      if (resolvedHit) return resolvedHit;

      return null;
    };

  const payload =
    event.dataTransfer?.getData("application/x-designer-component") ||
    event.dataTransfer?.getData("text/plain");
  const fallbackType = dragState.dragType || "";

  try {
    const parsed = JSON.parse(payload);
    const type = parsed?.type || "";
    if (!type && !fallbackType) return;
    const resolvedType = type || fallbackType;
    if (!node.value) return;
      const resolvedTarget = resolveElContainerTarget();
      const targetNode = resolvedTarget?.node || node.value;
      const targetElement = resolvedTarget?.element || event.currentTarget;
    if (!canAcceptChild(targetNode, resolvedType)) return;
    let insertIndex = (targetNode?.children || []).length;

    if (targetNode?.type && isFlexDropContainer(targetNode.type)) {
      const currentElement = targetElement;
      const direction = resolveFlexDirection(targetNode.type, currentElement);
      const insertInfo = dragDropManager.calculateFlexInsertPosition(
        currentElement,
        event,
        direction
      );
      insertIndex = insertInfo.index;
    }
    const dropPosition =
      targetNode?.type === "FreeContainer"
        ? resolveDropOffset(event, targetElement)
        : null;

    // 插入新节点
    editorStore.insertNode(resolvedType, targetNode?.id, insertIndex, {
      dropPosition: dropPosition || undefined,
    });
    endDrag();
  } catch (err) {
    const type = payload || fallbackType;
    if (!type) return;
    if (!node.value) return;
    const resolvedTarget = resolveElContainerTarget();
    const targetNode = resolvedTarget?.node || node.value;
    const targetElement = resolvedTarget?.element || event.currentTarget;
    if (!canAcceptChild(targetNode, type)) return;

    // 计算插入位置
    let insertIndex = (targetNode?.children || []).length;

    if (targetNode?.type && isFlexDropContainer(targetNode.type)) {
      const currentElement = targetElement;
      const direction = resolveFlexDirection(targetNode.type, currentElement);
      const insertInfo = dragDropManager.calculateFlexInsertPosition(
        currentElement,
        event,
        direction
      );
      insertIndex = insertInfo.index;
    }

    const dropPosition =
      targetNode?.type === "FreeContainer"
        ? resolveDropOffset(event, targetElement)
        : null;

    // 插入新节点
    editorStore.insertNode(type, targetNode?.id, insertIndex, {
      dropPosition: dropPosition || undefined,
    });
    endDrag();
  }
};

/**
 * 将布局配置转换为样式（支持新旧两种架构）
 * @param {import('@/editor-core').ComponentNode} currentNode - 当前节点
 * @param {boolean} isRoot - 是否根节点
 * @returns {Record<string, any>} 样式对象
 */
const resolveLayoutStyle = (currentNode, isRoot) => {
  if (isRoot) {
    return {
      position: "relative",
      width: "100%",
      height: "100%",
      boxSizing: "border-box",
    };
  }

  const style = {};

  // ✅ 新架构：positioning + absolutePos/flowLayout
  if (currentNode.positioning === "absolute" && currentNode.absolutePos) {
    const pos = currentNode.absolutePos;
    style.position = "absolute";
    style.left = `${pos.x ?? 0}px`;
    style.top = `${pos.y ?? 0}px`;
    if (pos.w !== undefined) style.width = `${pos.w}px`;
    if (pos.h !== undefined) style.height = `${pos.h}px`;
    if (pos.z !== undefined) style.zIndex = pos.z;
    return style;
  }

  if (currentNode.positioning === "flow") {
    // ✅ 流式布局：不使用绝对定位
    if (currentNode.flowLayout) {
      const flow = currentNode.flowLayout;
      // Flex 布局项
      if (
        flow.grow !== undefined ||
        flow.shrink !== undefined ||
        flow.basis !== undefined
      ) {
        style.flexGrow = flow.grow ?? 0;
        style.flexShrink = flow.shrink ?? 1;
        style.flexBasis = flow.basis ?? "auto";
        if (flow.alignSelf) style.alignSelf = flow.alignSelf;
      }
      // Grid 布局项
      if (flow.row !== undefined) {
        const rowSpan = flow.rowSpan || 1;
        style.gridRow = `${flow.row} / span ${rowSpan}`;
      }
      if (flow.col !== undefined) {
        const colSpan = flow.colSpan || 1;
        style.gridColumn = `${flow.col} / span ${colSpan}`;
      }
    }
    // 流式布局默认占据整行
    style.position = "relative";
    style.display = "block";
      const parentNode = doc.value?.getParent?.(currentNode.id);
      if (parentNode?.type === "ElContainer") {
        if (currentNode.type === "ElHeader") {
          style.gridArea = "header";
        } else if (currentNode.type === "ElAside") {
          style.gridArea = "aside";
        } else if (currentNode.type === "ElMain") {
          style.gridArea = "main";
        } else if (currentNode.type === "ElFooter") {
          style.gridArea = "footer";
        }
        style.width = "100%";
        style.height = "100%";
        style.alignSelf = "stretch";
        style.justifySelf = "stretch";
      } else if (
        parentNode?.type === "ElHeader" ||
        parentNode?.type === "ElAside" ||
        parentNode?.type === "ElMain" ||
        parentNode?.type === "ElFooter"
      ) {
        style.width = "100%";
        style.height = "100%";
        style.flexGrow = 1;
        style.flexShrink = 1;
      }
      return style;
    }

  // ✅ 旧架构：layoutItem（向后兼容）
  if (currentNode.layoutItem?.free) {
    const free = currentNode.layoutItem.free;
    if (free.mode === "abs" && free.abs) {
      const abs = free.abs;
      style.position = "absolute";
      style.left = `${abs.x ?? 0}px`;
      style.top = `${abs.y ?? 0}px`;
      if (abs.w !== undefined) style.width = `${abs.w}px`;
      if (abs.h !== undefined) style.height = `${abs.h}px`;
      if (abs.z !== undefined) style.zIndex = abs.z;
    }
    if (free.mode === "constraints" && free.constraints) {
      const constraints = free.constraints;
      style.position = "absolute";
      if (constraints.left !== undefined) style.left = `${constraints.left}px`;
      if (constraints.right !== undefined)
        style.right = `${constraints.right}px`;
      if (constraints.top !== undefined) style.top = `${constraints.top}px`;
      if (constraints.bottom !== undefined)
        style.bottom = `${constraints.bottom}px`;
      if (constraints.width !== undefined)
        style.width = `${constraints.width}px`;
      if (constraints.height !== undefined)
        style.height = `${constraints.height}px`;
    }
  }

  if (currentNode.layoutItem?.flex) {
    const flex = currentNode.layoutItem.flex;
    style.flexGrow = flex.grow ?? 0;
    style.flexShrink = flex.shrink ?? 1;
    style.flexBasis = flex.basis ?? "auto";
    if (flex.alignSelf) style.alignSelf = flex.alignSelf;
  }

  if (currentNode.layoutItem?.grid) {
    const grid = currentNode.layoutItem.grid;
    if (grid.row !== undefined) {
      const rowSpan = grid.rowSpan || 1;
      style.gridRow = `${grid.row} / span ${rowSpan}`;
    }
    if (grid.col !== undefined) {
      const colSpan = grid.colSpan || 1;
      style.gridColumn = `${grid.col} / span ${colSpan}`;
    }
  }

  return style;
};

/**
 * 解析容器布局样式
 * @param {import('@/editor-core').ComponentNode} currentNode - 当前节点
 * @param {Record<string, any>} baseStyle - 基础样式
 * @returns {Record<string, any>} 容器样式
 */
const resolveContainerStyle = (currentNode, baseStyle) => {
  const style = {};
  const manifest = componentRegistry.get(currentNode.type);
  const isContainer = manifest?.isContainer || false;

  if (
    currentNode.type === "FlexContainer" ||
    currentNode.type === "ResponsiveLayout"
  ) {
    style.display = "flex";
    style.flexDirection = currentNode.props?.direction || "row";
    style.flexWrap = currentNode.props?.wrap || "nowrap";
    style.justifyContent = currentNode.props?.justify || "flex-start";
    style.alignItems = currentNode.props?.align || "stretch";
    style.position = "relative";
    if (currentNode.props?.gap !== undefined) {
      style.gap = currentNode.props.gap;
    }
  } else if (
    currentNode.type === "GridContainer" ||
    currentNode.type === "ColumnLayout1" ||
    currentNode.type === "ColumnLayout2" ||
    currentNode.type === "ColumnLayout4"
  ) {
    style.display = "grid";
    style.position = "relative";
    if (currentNode.props?.columns) {
      style.gridTemplateColumns = formatGridTemplate(currentNode.props.columns);
    }
    if (currentNode.props?.rows) {
      style.gridTemplateRows = formatGridTemplate(currentNode.props.rows);
    }
    if (currentNode.props?.gap !== undefined) {
      style.gap = currentNode.props.gap;
    }
  } else if (currentNode.type === "FreeContainer") {
    // ✅ FreeContainer 使用 flex 布局，支持流式布局子元素
    style.display = "flex";
    style.flexDirection = "column";
    style.flexWrap = "nowrap";
    style.position = "relative";
    if (currentNode.props?.overflow) {
      style.overflow = currentNode.props.overflow;
    }
  } else if (currentNode.type === "ElContainer") {
    const children = currentNode.children || [];
    let headerNode = null;
    let footerNode = null;
    let asideNode = null;
    let mainNode = null;
    for (const childId of children) {
      const childNode = doc.value?.getNode?.(childId);
      if (!childNode) continue;
      if (childNode.type === "ElHeader" && !headerNode) headerNode = childNode;
      if (childNode.type === "ElFooter" && !footerNode) footerNode = childNode;
      if (childNode.type === "ElAside" && !asideNode) asideNode = childNode;
      if (childNode.type === "ElMain" && !mainNode) mainNode = childNode;
    }
    const hasHeader = Boolean(headerNode);
    const hasFooter = Boolean(footerNode);
    const hasAside = Boolean(asideNode);
    const hasMain = Boolean(mainNode);
    const hasBody = hasAside || hasMain;
    const hasTwoCols = hasAside && hasMain;
    const containerProps = currentNode.props || {};
    const headerHeight =
      containerProps.headerHeight || headerNode?.props?.height || "60px";
    const footerHeight =
      containerProps.footerHeight || footerNode?.props?.height || "60px";
    const asideWidth =
      containerProps.asideWidth || asideNode?.props?.width || "200px";

    style.display = "grid";
    style.position = "relative";
    style.gridTemplateColumns = hasTwoCols ? `${asideWidth} 1fr` : "1fr";

    const rows = [];
    const areas = [];
    if (hasHeader) {
      rows.push(headerHeight);
      areas.push(hasTwoCols ? '"header header"' : '"header"');
    }
    if (hasBody) {
      rows.push("1fr");
      if (hasTwoCols) {
        areas.push('"aside main"');
      } else if (hasAside) {
        areas.push('"aside"');
      } else {
        areas.push('"main"');
      }
    }
    if (hasFooter) {
      rows.push(footerHeight);
      areas.push(hasTwoCols ? '"footer footer"' : '"footer"');
    }
    if (rows.length === 0) {
      rows.push("1fr");
      areas.push('"main"');
    }

    style.gridTemplateRows = rows.join(" ");
    style.gridTemplateAreas = areas.join(" ");
  } else if (currentNode.type === "ElHeader") {
    const parentNode = doc.value?.getParent?.(currentNode.id);
    const headerHeight =
      parentNode?.type === "ElContainer"
        ? parentNode.props?.headerHeight
        : currentNode.props?.height;
    style.display = "flex";
    style.flexDirection = "column";
    style.alignItems = "stretch";
    style.width = "100%";
    style.height = headerHeight || "60px";
    style.padding = "0";
    style.overflow = "hidden";
    style.position = "relative";
  } else if (currentNode.type === "ElFooter") {
    const parentNode = doc.value?.getParent?.(currentNode.id);
    const footerHeight =
      parentNode?.type === "ElContainer"
        ? parentNode.props?.footerHeight
        : currentNode.props?.height;
    style.display = "flex";
    style.flexDirection = "column";
    style.alignItems = "stretch";
    style.width = "100%";
    style.height = footerHeight || "60px";
    style.padding = "0";
    style.overflow = "hidden";
    style.position = "relative";
  } else if (currentNode.type === "ElAside") {
    const parentNode = doc.value?.getParent?.(currentNode.id);
    const asideWidth =
      parentNode?.type === "ElContainer"
        ? parentNode.props?.asideWidth
        : currentNode.props?.width;
    style.display = "flex";
    style.flexDirection = "column";
    style.alignItems = "stretch";
    style.width = asideWidth || "200px";
    style.height = "100%";
    style.padding = "0";
    style.overflow = "hidden";
    style.position = "relative";
  } else if (currentNode.type === "ElMain") {
    style.display = "flex";
    style.flexDirection = "column";
    style.alignItems = "stretch";
    style.width = "100%";
    style.height = "100%";
    style.padding = "0";
    style.overflow = "hidden";
    style.position = "relative";
  } else if (currentNode.type === "ElLayout") {
    const gutter = Number(currentNode.props?.gutter) || 0;
    const justifyMap = {
      start: "flex-start",
      end: "flex-end",
      center: "center",
      "space-between": "space-between",
      "space-around": "space-around",
      "space-evenly": "space-evenly",
    };
    const alignMap = {
      top: "flex-start",
      middle: "center",
      bottom: "flex-end",
    };
    style.display = "flex";
    style.flexWrap = "wrap";
    style.alignItems = alignMap[currentNode.props?.align] || "stretch";
    style.justifyContent =
      justifyMap[currentNode.props?.justify] || "flex-start";
    style.gap = gutter ? `${gutter}px` : "0px";
    style.width = "100%";
    style.position = "relative";
  } else if (currentNode.type === "ElCol") {
    const span = Number(currentNode.props?.span) || 1;
    const percent = Math.max(1, Math.min(24, span)) / 24;
    style.display = "flex";
    style.flexDirection = "column";
    style.alignItems = "stretch";
    style.flex = `0 0 ${percent * 100}%`;
    style.maxWidth = `${percent * 100}%`;
    style.position = "relative";
  } else if (isContainer && !baseStyle.position) {
    style.position = "relative";
  }

  return style;
};

/**
 * 标准化样式对象
 * @param {Record<string, any>} rawStyle - 原始样式
 * @returns {Record<string, any>} 规范化样式
 */
const normalizeStyleObject = (rawStyle) => {
  const style = {};
  for (const [key, value] of Object.entries(rawStyle)) {
    if (value && typeof value === "object" && key === "background") {
      if (value.value !== undefined) {
        style.background = value.value;
      }
      continue;
    }
    style[key] = normalizeStyleValue(key, value);
  }
  return style;
};

/**
 * 标准化样式值（补充单位）
 * @param {string} key - 样式键
 * @param {any} value - 样式值
 * @returns {any} 标准化结果
 */
const normalizeStyleValue = (key, value) => {
  if (value === null || value === undefined) return value;
  if (typeof value === "number" && needsPxUnit(key)) {
    return `${value}px`;
  }
  return value;
};

/**
 * 判断是否需要追加 px 单位
 * @param {string} key - 样式键
 * @returns {boolean}
 */
const needsPxUnit = (key) => {
  return [
    "left",
    "top",
    "right",
    "bottom",
    "width",
    "height",
    "minWidth",
    "minHeight",
    "maxWidth",
    "maxHeight",
    "fontSize",
    "borderRadius",
    "gap",
    "padding",
    "paddingTop",
    "paddingRight",
    "paddingBottom",
    "paddingLeft",
    "margin",
    "marginTop",
    "marginRight",
    "marginBottom",
    "marginLeft",
  ].includes(key);
};

/**
 * 格式化 Grid 模板
 * @param {string | number} value - 模板配置
 * @returns {string}
 */
const formatGridTemplate = (value) => {
  if (typeof value === "number") {
    return `repeat(${value}, minmax(0, 1fr))`;
  }
  return value;
};

/**
 * 判断是否为可编辑元素（避免拖拽干扰编辑）
 * @param {EventTarget | null} target - 事件目标
 * @returns {boolean} 是否为可交互元素
 * @throws {Error} 无
 */
const isInteractiveTarget = (target) => {
  if (!target || !(target instanceof Element)) return false;
  if (target.isContentEditable) return true;
  return Boolean(target.closest("[contenteditable='true']"));
};

/**
 * 解析节点自由布局信息
 * @param {import('@/editor-core').ComponentNode} currentNode - 当前节点
 * @returns {{ x: number, y: number, w: number, h: number, z: number }}
 * @throws {Error} 无
 */
const resolveAbsoluteLayout = (currentNode) => {
  const fallbackAbs = currentNode.layoutItem?.free?.abs || {};
  const absolutePos = currentNode.absolutePos || fallbackAbs;
  const rect = nodeRef.value?.getBoundingClientRect?.();
  const width = Number.isFinite(absolutePos.w)
    ? absolutePos.w
    : (rect?.width ?? 120);
  const height = Number.isFinite(absolutePos.h)
    ? absolutePos.h
    : (rect?.height ?? 40);

  let x = Number.isFinite(absolutePos.x) ? absolutePos.x : 0;
  let y = Number.isFinite(absolutePos.y) ? absolutePos.y : 0;

  if (!Number.isFinite(absolutePos.x) || !Number.isFinite(absolutePos.y)) {
    const parentElement =
      nodeRef.value?.parentElement?.closest?.("[data-node-id]");
    const parentRect = parentElement?.getBoundingClientRect?.();
    if (rect && parentRect) {
      x = rect.left - parentRect.left;
      y = rect.top - parentRect.top;
    }
  }

  return {
    x: Math.max(0, Math.round(x)),
    y: Math.max(0, Math.round(y)),
    w: Math.max(1, Math.round(width)),
    h: Math.max(1, Math.round(height)),
    z: Number.isFinite(absolutePos.z) ? absolutePos.z : 1,
  };
};

/**
 * 解析尺寸为像素值
 * @param {string | number | undefined | null} value - 尺寸值
 * @returns {number | undefined}
 */
const parseSizeToNumber = (value) => {
  if (value === null || value === undefined) return undefined;
  if (typeof value === "number" && Number.isFinite(value)) return value;
  const text = String(value).trim();
  if (!text || text === "auto") return undefined;
  if (text.endsWith("px")) {
    const num = Number.parseFloat(text.slice(0, -2));
    return Number.isFinite(num) ? num : undefined;
  }
  if (/^[\d.]+$/.test(text)) {
    const num = Number.parseFloat(text);
    return Number.isFinite(num) ? num : undefined;
  }
  return undefined;
};

/**
 * 计算容器最小尺寸，避免小于内部区域
 * @param {import('@/editor-core').ComponentNode | null} containerNode - 容器节点
 * @returns {{ width: number, height: number } | null}
 */
const resolveElContainerMinSize = (containerNode) => {
  if (!containerNode || containerNode.type !== "ElContainer") return null;
  const children = containerNode.children || [];
  let hasHeader = false;
  let hasFooter = false;
  let hasAside = false;
  let hasMain = false;
  for (const childId of children) {
    const childNode = doc.value?.getNode?.(childId);
    if (!childNode) continue;
    if (childNode.type === "ElHeader") hasHeader = true;
    if (childNode.type === "ElFooter") hasFooter = true;
    if (childNode.type === "ElAside") hasAside = true;
    if (childNode.type === "ElMain") hasMain = true;
  }

  const props = containerNode.props || {};
  if (typeof props.showHeader === "boolean") hasHeader = props.showHeader;
  if (typeof props.showFooter === "boolean") hasFooter = props.showFooter;
  if (typeof props.showAside === "boolean") hasAside = props.showAside;
  if (typeof props.showMain === "boolean") hasMain = props.showMain;

  const headerHeight = parseSizeToNumber(props.headerHeight) ?? 60;
  const footerHeight = parseSizeToNumber(props.footerHeight) ?? 60;
  const asideWidth = parseSizeToNumber(props.asideWidth) ?? 200;
  const minBodySize = 40;

  const hasBody = hasAside || hasMain;
  let minWidth = 0;
  if (hasAside && hasMain) {
    minWidth = asideWidth + minBodySize;
  } else if (hasAside) {
    minWidth = asideWidth;
  } else if (hasMain) {
    minWidth = minBodySize;
  }

  let minHeight = 0;
  if (hasHeader) minHeight += headerHeight;
  if (hasFooter) minHeight += footerHeight;
  if (hasBody) minHeight += minBodySize;

  if (minWidth <= 0 && minHeight <= 0) return null;
  return { width: minWidth, height: minHeight };
};

let activeDragHandlers = null;

/**
 * 清理拖拽事件监听
 * @returns {void}
 * @throws {Error} 无
 */
const cleanupDragHandlers = () => {
  if (!activeDragHandlers) return;
  const {
    move,
    up,
    userSelect,
    pointerTarget,
    pointerId,
    usePointer,
    pointerEvents,
    pointerElement,
  } = activeDragHandlers;
  if (usePointer) {
    document.removeEventListener("pointermove", move);
    document.removeEventListener("pointerup", up);
    document.removeEventListener("pointercancel", up);
  } else {
    document.removeEventListener("mousemove", move);
    document.removeEventListener("mouseup", up);
  }
  if (pointerTarget?.releasePointerCapture && pointerId !== undefined) {
    try {
      pointerTarget.releasePointerCapture(pointerId);
    } catch (error) {
      // 忽略释放失败
    }
  }
  if (pointerElement && pointerEvents !== undefined) {
    pointerElement.style.pointerEvents = pointerEvents;
  }
  document.body.style.userSelect = userSelect ?? "";
  activeDragHandlers = null;
};

onBeforeUnmount(() => {
  cleanupDragHandlers();
});

/**
 * 处理节点拖拽移动
 * @param {MouseEvent} event - 鼠标事件
 * @returns {void}
 * @throws {Error} 无
 */
/**
 * ??????
 * @param {PointerEvent} event - ????
 * @param {{ key: string, x: number, y: number }} handle - ????
 * @returns {void}
 */
const handleResizePointerDown = (event, handle) => {
  if (props.readonly) return;
  if (activeDragHandlers) return;
  if (!node.value || !isMovable.value) return;
  if (event.pointerType === "mouse" && event.button !== 0) return;

  event.preventDefault();
  event.stopPropagation();

  if (selection.value) {
    const element = createSelectableElement("node", node.value.id);
    selection.value.select(element);
  }

  const regionConfig = getRegionResizeConfig(node.value.type);
  if (regionConfig && !regionConfig.handles.includes(handle.key)) {
    return;
  }

  const zoomValue = Number(canvasZoom?.value) || 1;
  const baseLayout = resolveAbsoluteLayout(node.value);
  const rect = nodeRef.value?.getBoundingClientRect?.();
  const baseWidth = baseLayout.w || rect?.width || 120;
  const baseHeight = baseLayout.h || rect?.height || 40;
  const startClientX = event.clientX;
  const startClientY = event.clientY;
  const minSize = 40;
  const containerMinSize = resolveElContainerMinSize(node.value);

  const parentNode = doc.value?.getParent?.(node.value.id);
  const shouldUpdateAbsolute =
    parentNode?.type === "FreeContainer" ||
    node.value.positioning === "absolute" ||
    node.value.layoutItem?.free?.mode === "abs" ||
    node.value.absolutePos;

  cleanupDragHandlers();
  const originalUserSelect = document.body.style.userSelect;
  document.body.style.userSelect = "none";

  if (history.value && !history.value.isInTransaction?.()) {
    history.value.beginTransaction();
  }

  const usePointer = event.type === "pointerdown";
  const pointerTarget =
    event.target instanceof Element ? event.target : nodeRef.value?.$el;
  const pointerElement = nodeRef.value;
  const originalPointerEvents = pointerElement?.style.pointerEvents;
  if (pointerElement) {
    pointerElement.style.pointerEvents = "none";
  }
  if (
    usePointer &&
    pointerTarget?.setPointerCapture &&
    event.pointerId !== undefined
  ) {
    try {
      pointerTarget.setPointerCapture(event.pointerId);
    } catch (error) {
      // ??????
    }
  }

  const move = (moveEvent) => {
    if (!node.value) return;
    const deltaX = (moveEvent.clientX - startClientX) / zoomValue;
    const deltaY = (moveEvent.clientY - startClientY) / zoomValue;

    if (regionConfig) {
      const currentSize =
        regionConfig.axis === "x" ? baseWidth : baseHeight;
      const rawDelta = regionConfig.axis === "x" ? deltaX : deltaY;
      const delta = regionConfig.invert ? -rawDelta : rawDelta;
      let nextSize = currentSize + delta;
      const containerNode =
        parentNode?.type === "ElContainer"
          ? parentNode
          : doc.value?.getParent?.(parentNode?.id);
      if (containerNode) {
        const containerEl = document.querySelector(
          `[data-node-id="${containerNode.id}"]`
        );
        const containerRect = containerEl?.getBoundingClientRect?.();
        const minBodySize = minSize;
        if (containerRect) {
          if (node.value.type === "ElAside") {
            const maxWidth = Math.max(
              minBodySize,
              Math.round(containerRect.width - minBodySize)
            );
            nextSize = Math.min(nextSize, maxWidth);
          } else if (node.value.type === "ElHeader") {
            const footerHeight = Number.parseFloat(
              containerNode.props?.footerHeight || "0"
            );
            const maxHeight = Math.max(
              minBodySize,
              Math.round(containerRect.height - footerHeight - minBodySize)
            );
            nextSize = Math.min(nextSize, maxHeight);
          } else if (node.value.type === "ElFooter") {
            const headerHeight = Number.parseFloat(
              containerNode.props?.headerHeight || "0"
            );
            const maxHeight = Math.max(
              minBodySize,
              Math.round(containerRect.height - headerHeight - minBodySize)
            );
            nextSize = Math.min(nextSize, maxHeight);
          }
        }
      }
      if (nextSize < minSize) nextSize = minSize;
      nextSize = Math.round(nextSize);
      const nextProps = {
        ...(node.value.props || {}),
        [regionConfig.prop]: `${nextSize}px`,
      };
      const parentContainer = parentNode?.type === "ElContainer" ? parentNode : null;
      const parentPatch =
        parentContainer && regionConfig.prop === "height"
          ? node.value.type === "ElHeader"
            ? { headerHeight: `${nextSize}px` }
            : node.value.type === "ElFooter"
              ? { footerHeight: `${nextSize}px` }
              : null
          : parentContainer && regionConfig.prop === "width"
            ? { asideWidth: `${nextSize}px` }
            : null;

      if (nodeRef.value) {
        if (regionConfig.axis === "x") {
          nodeRef.value.style.width = `${nextSize}px`;
        } else {
          nodeRef.value.style.height = `${nextSize}px`;
        }
      }

      const patch = { props: nextProps };
      if (history.value?.isInTransaction?.()) {
        history.value.executeInTransaction(
          new UpdateNodeCommand(node.value.id, patch)
        );
        if (parentContainer && parentPatch) {
          history.value.executeInTransaction(
            new UpdateNodeCommand(parentContainer.id, {
              props: { ...(parentContainer.props || {}), ...parentPatch },
            })
          );
        }
        return;
      }
      if (history.value?.execute) {
        history.value.execute(new UpdateNodeCommand(node.value.id, patch));
        if (parentContainer && parentPatch) {
          history.value.execute(
            new UpdateNodeCommand(parentContainer.id, {
              props: { ...(parentContainer.props || {}), ...parentPatch },
            })
          );
        }
        return;
      }
      if (doc.value?._updateNode) {
        doc.value._updateNode(node.value.id, patch);
      }
      return;
    }

    let nextWidth = baseWidth;
    let nextHeight = baseHeight;
    let nextX = baseLayout.x;
    let nextY = baseLayout.y;

    if (handle.x === 1) {
      nextWidth = baseWidth + deltaX;
    } else if (handle.x === -1) {
      nextWidth = baseWidth - deltaX;
      nextX = baseLayout.x + deltaX;
    }

    if (handle.y === 1) {
      nextHeight = baseHeight + deltaY;
    } else if (handle.y === -1) {
      nextHeight = baseHeight - deltaY;
      nextY = baseLayout.y + deltaY;
    }

    if (handle.x === -1 && nextWidth < minSize) {
      nextX = baseLayout.x + (baseWidth - minSize);
      nextWidth = minSize;
    }
    if (handle.x === 1 && nextWidth < minSize) {
      nextWidth = minSize;
    }
    if (handle.y === -1 && nextHeight < minSize) {
      nextY = baseLayout.y + (baseHeight - minSize);
      nextHeight = minSize;
    }
    if (handle.y === 1 && nextHeight < minSize) {
      nextHeight = minSize;
    }

    if (containerMinSize) {
      if (nextWidth < containerMinSize.width) {
        nextWidth = containerMinSize.width;
      }
      if (nextHeight < containerMinSize.height) {
        nextHeight = containerMinSize.height;
      }
    }

    nextWidth = Math.round(nextWidth);
    nextHeight = Math.round(nextHeight);
    nextX = Math.max(0, Math.round(nextX));
    nextY = Math.max(0, Math.round(nextY));

    const nextStyle = {
      ...(node.value.style || {}),
      width: `${nextWidth}px`,
      height: `${nextHeight}px`,
    };

    let patch = { style: nextStyle };

    if (shouldUpdateAbsolute) {
      const nextAbs = {
        x: nextX,
        y: nextY,
        w: nextWidth,
        h: nextHeight,
        z: baseLayout.z,
      };

      if (nodeRef.value) {
        nodeRef.value.style.position = "absolute";
        nodeRef.value.style.left = `${nextAbs.x}px`;
        nodeRef.value.style.top = `${nextAbs.y}px`;
        nodeRef.value.style.width = `${nextAbs.w}px`;
        nodeRef.value.style.height = `${nextAbs.h}px`;
        nodeRef.value.style.zIndex = `${nextAbs.z}`;
      }

      const nextLayoutItem = {
        ...(node.value.layoutItem || {}),
        free: {
          mode: "abs",
          abs: { ...nextAbs },
        },
      };

      patch = {
        ...patch,
        positioning: "absolute",
        absolutePos: nextAbs,
        layoutItem: nextLayoutItem,
      };
    } else if (nodeRef.value) {
      nodeRef.value.style.width = `${nextWidth}px`;
      nodeRef.value.style.height = `${nextHeight}px`;
    }

    if (history.value?.isInTransaction?.()) {
      history.value.executeInTransaction(
        new UpdateNodeCommand(node.value.id, patch)
      );
      if (typeof window !== "undefined") {
        window.dispatchEvent(
          new CustomEvent("designer:node-transform", {
            detail: { x: nextX, y: nextY },
          })
        );
      }
      return;
    }
    if (history.value?.execute) {
      history.value.execute(new UpdateNodeCommand(node.value.id, patch));
      if (typeof window !== "undefined") {
        window.dispatchEvent(
          new CustomEvent("designer:node-transform", {
            detail: { x: nextX, y: nextY },
          })
        );
      }
      return;
    }
    if (doc.value?._updateNode) {
      doc.value._updateNode(node.value.id, patch);
      if (typeof window !== "undefined") {
        window.dispatchEvent(
          new CustomEvent("designer:node-transform", {
            detail: { x: nextX, y: nextY },
          })
        );
      }
    }
  };

  const up = () => {
    cleanupDragHandlers();
    if (history.value?.isInTransaction?.()) {
      history.value.commitTransaction("????");
    }
    if (typeof window !== "undefined") {
      window.dispatchEvent(new CustomEvent("designer:node-transform-end"));
    }
  };

  activeDragHandlers = {
    move,
    up,
    userSelect: originalUserSelect,
    pointerTarget,
    pointerId: event.pointerId,
    usePointer,
    pointerEvents: originalPointerEvents,
    pointerElement,
  };

  if (usePointer) {
    document.addEventListener("pointermove", move);
    document.addEventListener("pointerup", up);
    document.addEventListener("pointercancel", up);
  } else {
    document.addEventListener("mousemove", move);
    document.addEventListener("mouseup", up);
  }
};

const handlePointerDown = (event) => {
  if (props.readonly) return;
  if (activeDragHandlers) return;
  if (!node.value || !isMovable.value) return;
  if (event.pointerType === "mouse" && event.button !== 0) return;
  if (event.target?.closest?.(".resize-handle")) return;
  if (isInteractiveTarget(event.target)) return;
  const targetNodeEl = event.target?.closest?.("[data-node-id]");
  const targetNodeId = targetNodeEl?.getAttribute?.("data-node-id");
  if (targetNodeId && targetNodeId !== node.value.id) {
    return;
  }

  event.preventDefault();
  event.stopPropagation();

  if (selection.value) {
    const element = createSelectableElement("node", node.value.id);
    selection.value.select(element);
  }

  const originParent = doc.value?.getParent?.(node.value.id);
  const isRegionNode =
    node.value.type === "ElHeader" ||
    node.value.type === "ElAside" ||
    node.value.type === "ElMain" ||
    node.value.type === "ElFooter";
  const isRegionParent =
    originParent?.type === "ElHeader" ||
    originParent?.type === "ElAside" ||
    originParent?.type === "ElMain" ||
    originParent?.type === "ElFooter";
  const originContainer = isRegionParent
    ? doc.value?.getParent?.(originParent?.id)
    : isRegionNode
      ? originParent
      : null;
  const allowRegionMoveOut = isRegionNode;
  const rootNodeId = currentPage.value?.rootNodeId || "";

  const zoomValue = Number(canvasZoom?.value) || 1;
  const baseLayout = resolveAbsoluteLayout(node.value);
  const startClientX = event.clientX;
  const startClientY = event.clientY;
  let hasMoved = false;
  let startedDragFromMove = false;

  const resolveDropRegion = (upEvent) => {
    if (!upEvent) return null;
    const hitList = document.elementsFromPoint(
      upEvent.clientX,
      upEvent.clientY
    );
    let containerNode = null;
    for (const hit of hitList) {
      const nodeElement = hit.closest?.("[data-node-id]");
      const nodeId = nodeElement?.getAttribute?.("data-node-id");
      if (!nodeId) continue;
      const targetNode = doc.value?.getNode?.(nodeId);
      if (!targetNode) continue;
      if (
        targetNode.type === "ElHeader" ||
        targetNode.type === "ElAside" ||
        targetNode.type === "ElMain" ||
        targetNode.type === "ElFooter"
      ) {
        return targetNode;
      }
      if (targetNode.type === "ElContainer" && !containerNode) {
        containerNode = targetNode;
      }
    }
    if (containerNode) {
      const mainChildId = (containerNode.children || []).find((childId) => {
        const childNode = doc.value?.getNode?.(childId);
        return childNode?.type === "ElMain";
      });
      if (mainChildId) {
        return doc.value?.getNode?.(mainChildId) || null;
      }
    }
    return null;
  };

  cleanupDragHandlers();
  const originalUserSelect = document.body.style.userSelect;
  document.body.style.userSelect = "none";

  if (history.value && !history.value.isInTransaction?.()) {
    history.value.beginTransaction();
  }

  const usePointer = event.type === "pointerdown";
  const pointerTarget =
    event.target instanceof Element ? event.target : nodeRef.value?.$el;
  if (
    usePointer &&
    pointerTarget?.setPointerCapture &&
    event.pointerId !== undefined
  ) {
    try {
      pointerTarget.setPointerCapture(event.pointerId);
    } catch (error) {
      // 忽略捕获失败
    }
  }

  const move = (moveEvent) => {
    if (!node.value) return;
    const deltaX = (moveEvent.clientX - startClientX) / zoomValue;
    const deltaY = (moveEvent.clientY - startClientY) / zoomValue;
    if (Math.abs(deltaX) > 1 || Math.abs(deltaY) > 1) {
      hasMoved = true;
      if (!startedDragFromMove) {
        startDrag(node.value.type);
        startedDragFromMove = true;
      }
      const dropRegion = resolveDropRegion(moveEvent);
      if (dropRegion?.id) {
        const rect = document
          .querySelector(`[data-node-id="${dropRegion.id}"]`)
          ?.getBoundingClientRect?.();
        updateDropTarget({
          containerId: dropRegion.id,
          insertIndex: (dropRegion.children || []).length,
          position: rect
            ? {
                x: rect.left,
                y: rect.top,
                width: rect.width,
                height: rect.height,
              }
            : { x: 0, y: 0, width: 0, height: 0 },
          layoutType: "flex",
          direction: "column",
        });
      } else {
        clearDropTarget();
      }
    }
    const nextX = Math.max(0, Math.round(baseLayout.x + deltaX));
    const nextY = Math.max(0, Math.round(baseLayout.y + deltaY));

    const nextAbs = {
      x: nextX,
      y: nextY,
      w: baseLayout.w,
      h: baseLayout.h,
      z: baseLayout.z,
    };

    if (nodeRef.value) {
      nodeRef.value.style.position = "absolute";
      nodeRef.value.style.left = `${nextAbs.x}px`;
      nodeRef.value.style.top = `${nextAbs.y}px`;
      nodeRef.value.style.width = `${nextAbs.w}px`;
      nodeRef.value.style.height = `${nextAbs.h}px`;
      nodeRef.value.style.zIndex = `${nextAbs.z}`;
    }

    const nextLayoutItem = {
      ...(node.value.layoutItem || {}),
      free: {
        mode: "abs",
        abs: { ...nextAbs },
      },
    };

    // 同步新旧布局字段，确保自由拖动可见
    const patch = {
      positioning: "absolute",
      absolutePos: nextAbs,
      layoutItem: nextLayoutItem,
    };
    if (history.value?.isInTransaction?.()) {
      history.value.executeInTransaction(
        new UpdateNodeCommand(node.value.id, patch)
      );
      if (typeof window !== "undefined") {
        window.dispatchEvent(
          new CustomEvent("designer:node-transform", {
            detail: { x: nextAbs.x, y: nextAbs.y },
          })
        );
      }
      return;
    }
    if (history.value?.execute) {
      history.value.execute(new UpdateNodeCommand(node.value.id, patch));
      if (typeof window !== "undefined") {
        window.dispatchEvent(
          new CustomEvent("designer:node-transform", {
            detail: { x: nextAbs.x, y: nextAbs.y },
          })
        );
      }
      return;
    }
    if (doc.value?._updateNode) {
      doc.value._updateNode(node.value.id, patch);
      if (typeof window !== "undefined") {
        window.dispatchEvent(
          new CustomEvent("designer:node-transform", {
            detail: { x: nextAbs.x, y: nextAbs.y },
          })
        );
      }
    }
  };


  const up = (upEvent) => {
    if (startedDragFromMove) {
      endDrag();
      clearDropTarget();
    }
    if (hasMoved && !isRegionNode && upEvent) {
      const dropRegion = resolveDropRegion(upEvent);
      if (
        dropRegion &&
        dropRegion.id &&
        dropRegion.id !== originParent?.id &&
        canAcceptChild(dropRegion, node.value.type)
      ) {
        const insertIndex = (dropRegion.children || []).length;
        const moveCommand = new MoveNodeCommand(
          node.value.id,
          dropRegion.id,
          insertIndex
        );
        const updateCommand = new UpdateNodeCommand(node.value.id, {
          positioning: "flow",
          absolutePos: undefined,
          flowLayout: undefined,
          layoutItem: undefined,
        });
        if (history.value?.isInTransaction?.()) {
          history.value.executeInTransaction(moveCommand);
          history.value.executeInTransaction(updateCommand);
        } else if (history.value?.execute) {
          history.value.execute(moveCommand);
          history.value.execute(updateCommand);
        } else if (doc.value?._moveNode && doc.value?._updateNode) {
          doc.value._moveNode(node.value.id, dropRegion.id, insertIndex);
          doc.value._updateNode(node.value.id, {
            positioning: "flow",
            absolutePos: undefined,
            flowLayout: undefined,
            layoutItem: undefined,
          });
        }
      }
    }
    if (
      hasMoved &&
      (isRegionParent || allowRegionMoveOut) &&
      originContainer?.type === "ElContainer" &&
      rootNodeId &&
      upEvent
    ) {
      const containerEl = document.querySelector(
        `[data-node-id="${originContainer.id}"]`
      );
      const containerRect = containerEl?.getBoundingClientRect?.();
      const isInsideContainer = containerRect
        ? upEvent.clientX >= containerRect.left &&
          upEvent.clientX <= containerRect.right &&
          upEvent.clientY >= containerRect.top &&
          upEvent.clientY <= containerRect.bottom
        : (() => {
            const hit = document.elementFromPoint(
              upEvent.clientX,
              upEvent.clientY
            );
            return Boolean(containerEl && hit && containerEl.contains(hit));
          })();
      if (!isInsideContainer) {
        const rootEl = document.querySelector(
          `[data-node-id="${rootNodeId}"]`
        );
        const rootRect = rootEl?.getBoundingClientRect?.();
        const nextX = rootRect
          ? (upEvent.clientX - rootRect.left) / zoomValue
          : 0;
        const nextY = rootRect
          ? (upEvent.clientY - rootRect.top) / zoomValue
          : 0;
        const nextAbs = {
          x: Math.max(0, Math.round(nextX)),
          y: Math.max(0, Math.round(nextY)),
          w: baseLayout.w,
          h: baseLayout.h,
          z: baseLayout.z,
        };
        const nextLayoutItem = {
          ...(node.value.layoutItem || {}),
          free: {
            mode: "abs",
            abs: { ...nextAbs },
          },
        };
        const rootNode = doc.value?.getNode?.(rootNodeId);
        const insertIndex = rootNode?.children?.length ?? 0;
        const moveCommand = new MoveNodeCommand(
          node.value.id,
          rootNodeId,
          insertIndex
        );
        const updateCommand = new UpdateNodeCommand(node.value.id, {
          positioning: "absolute",
          absolutePos: nextAbs,
          layoutItem: nextLayoutItem,
        });
        if (history.value?.isInTransaction?.()) {
          history.value.executeInTransaction(moveCommand);
          history.value.executeInTransaction(updateCommand);
        } else if (history.value?.execute) {
          history.value.execute(moveCommand);
          history.value.execute(updateCommand);
        } else if (doc.value?._moveNode && doc.value?._updateNode) {
          doc.value._moveNode(node.value.id, rootNodeId, insertIndex);
          doc.value._updateNode(node.value.id, {
            positioning: "absolute",
            absolutePos: nextAbs,
            layoutItem: nextLayoutItem,
          });
        }
      }
    }
    cleanupDragHandlers();
    if (history.value?.isInTransaction?.()) {
      history.value.commitTransaction("移动组件");
    }
    if (typeof window !== "undefined") {
      window.dispatchEvent(new CustomEvent("designer:node-transform-end"));
    }
  };

  activeDragHandlers = {
    move,
    up,
    userSelect: originalUserSelect,
    pointerTarget,
    pointerId: event.pointerId,
    usePointer,
  };

  if (usePointer) {
    document.addEventListener("pointermove", move);
    document.addEventListener("pointerup", up);
    document.addEventListener("pointercancel", up);
  } else {
    document.addEventListener("mousemove", move);
    document.addEventListener("mouseup", up);
  }
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
}

.designer-node.is-draggable {
  cursor: move;
}

.designer-node.is-container {
  min-height: 40px;
}

.designer-node.is-locked {
  opacity: 0.6;
  pointer-events: none;
}

.designer-node.drag-over {
  outline: 2px solid #3b82f6;
  background-color: rgba(59, 130, 246, 0.05);
}

.empty-container-hint {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  min-height: 40px;
  color: #9ca3af;
  font-size: 12px;
  pointer-events: none;
  border: 1px dashed #d1d5db;
  border-radius: 4px;
  transition: all 0.2s ease;
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

.insert-line {
  position: absolute;
  background: #3b82f6;
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


