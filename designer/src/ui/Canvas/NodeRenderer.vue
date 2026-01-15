<template>
  <div
    v-if="node && !node.hidden"
    :class="nodeClass"
    :style="wrapperStyle"
    :data-node-id="node.id"
    :data-node-type="node.type"
    ref="nodeRef"
    @click.stop="handleClick"
    @pointerdown.capture="handlePointerDown"
    @dragover.prevent="handleDragOver"
    @dragstart.prevent
    @dragleave="handleDragLeave"
    @drop.prevent="handleDrop"
    @contextmenu.prevent="handleContextMenu"
  >
    <component
      :is="renderTag"
      :style="contentStyle"
      v-bind="resolvedProps"
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
      <template v-if="node?.type === 'ImageCarousel' || node?.type === 'CarouselComponent'">
        <el-carousel-item
          v-for="item in carouselItems"
          :key="item.label"
        >
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
        <div class="empty-container-hint">
          <span v-if="isDragOver">释放以添加组件</span>
          <span v-else>拖拽组件到此处</span>
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
  </div>
</template>

<script setup>
import { computed, ref, inject, onBeforeUnmount } from "vue";
import { storeToRefs } from "pinia";
import { useEditorStore } from "@/stores/editor-store";
import { datacenterApi } from "@/services";
import {
  componentRegistry,
  createSelectableElement,
  UpdateNodeCommand,
} from "@/editor-core";
import { createDragDropManager } from "./DragDropManager";
import { useDragState, endDrag } from "./use-drag-state";

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
} = storeToRefs(editorStore);
const canvasZoom = inject("canvasZoom", ref(1));
const dragState = useDragState();
const nodeRef = ref(null);

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
const fallbackCarouselItems = [
  { label: "轮播一" },
  { label: "轮播二" },
];

/**
 * 规范化选项列表
 * @param {Array} source - 原始列表
 * @param {Array} fallback - 默认列表
 * @returns {Array}
 */
const normalizeOptions = (source, fallback) => {
  if (Array.isArray(source) && source.length > 0) return source;
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
const showInsertLine = ref(false);
const insertLineStyle = ref(null);
const dragDropManager = createDragDropManager();
const resolvedProps = computed(() => {
  if (!node.value) return {};
  return filterRenderProps(node.value.type, node.value.props || {});
});

const selectOptions = computed(() =>
  normalizeOptions(node.value?.props?.options, fallbackSelectOptions)
);
const radioOptions = computed(() =>
  normalizeOptions(node.value?.props?.options, fallbackRadioOptions)
);
const checkboxOptions = computed(() =>
  normalizeOptions(node.value?.props?.options, fallbackCheckboxOptions)
);
const dropdownItems = computed(() =>
  normalizeOptions(node.value?.props?.items, fallbackDropdownItems)
);
const menuItems = computed(() =>
  normalizeOptions(node.value?.props?.items, fallbackMenuItems)
);
const tableColumns = computed(() =>
  normalizeOptions(node.value?.props?.columns, fallbackTableColumns)
);
const bigTableColumns = computed(() =>
  normalizeOptions(node.value?.props?.columns, fallbackBigTableColumns)
);
const timelineItems = computed(() =>
  normalizeOptions(node.value?.props?.items, fallbackTimelineItems)
);
const tabsList = computed(() =>
  normalizeOptions(node.value?.props?.tabs, fallbackTabs)
);
const stepsItems = computed(() =>
  normalizeOptions(node.value?.props?.items, fallbackStepsItems)
);
const collapseItems = computed(() =>
  normalizeOptions(node.value?.props?.items, fallbackCollapseItems)
);
const carouselItems = computed(() =>
  normalizeOptions(node.value?.props?.items, fallbackCarouselItems)
);
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

const hasChildren = computed(() => {
  return (node.value?.children || []).length > 0;
});

/**
 * 判断节点是否可自由拖动
 * @returns {boolean} 是否允许拖动
 * @throws {Error} 无
 */
const isMovable = computed(() => {
  if (!node.value || props.isRoot || node.value.locked) return false;
  return true;
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
  if (isDragOver.value && !props.readonly) classes.push("drag-over");
  if (!props.readonly && selection.value?.isSelected(node.value.id)) {
    classes.push("is-selected");
  }
  return classes.join(" ");
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
    case "Text":
      return "div";
    default:
      return "div";
  }
});

const displayContent = computed(() => {
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
  const style = {
    ...containerStyle,
    ...customStyle,
  };
  if (props.isRoot || layoutStyle.value.position === "absolute") {
    if (!style.width) style.width = "100%";
    if (!style.height) style.height = "100%";
  }
  return style;
});

const wrapperStyle = computed(() => {
  if (!node.value) return {};
  return layoutStyle.value;
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
      try {
        // Treat as function body or full function text
        if (raw.trim().startsWith("function")) {
          return new Function(`return (${raw});`)();
        }
        return new Function(raw);
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
  return queries.find((item) => item.name === queryName || item.id === queryName) || null;
};

const resolveMappedGlobalValue = async (name, detail) => {
  const source = detail?.source;
  if (!source || source.type !== "dataCenter" || !source.path) {
    return normalizeGlobalValue(detail);
  }
  if (!projectId.value) return normalizeGlobalValue(detail);
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
        const detail = vars[prop];
        if (!detail) return undefined;
        if (detail?.mapped && detail?.source?.type === "dataCenter") {
          if (!mappedValueCache.has(prop)) {
            const promise = resolveMappedGlobalValue(prop, detail).catch(() => null);
            mappedValueCache.set(prop, promise);
          }
          return mappedValueCache.get(prop);
        }
        return normalizeGlobalValue(detail);
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
      `"use strict";\nreturn (async () => {\n${code}\n})();`
    );
    return await runner(...values);
  } catch (error) {
    console.error("[Preview] Script error:", error);
  }
};

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

const handleClick = (event) => {
  if (props.readonly) {
    void runPreviewScript("click", event);
    return;
  }
  handleSelect(event);
};

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
  if (node.value?.type === "FlexContainer") {
    const direction = node.value.props?.direction || "column";
    const currentElement = event.currentTarget;
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
    if (!canAcceptChild(node.value, resolvedType)) return;
    let insertIndex = (node.value?.children || []).length;

    if (node.value?.type === "FlexContainer") {
      const direction = node.value.props?.direction || "column";
      const currentElement = event.currentTarget;
      const insertInfo = dragDropManager.calculateFlexInsertPosition(
        currentElement,
        event,
        direction
      );
      insertIndex = insertInfo.index;
    }
    const dropPosition =
      node.value?.type === "FreeContainer"
        ? resolveDropOffset(event, event.currentTarget)
        : null;

    // 插入新节点
    editorStore.insertNode(resolvedType, node.value?.id, insertIndex, {
      dropPosition: dropPosition || undefined,
    });
    endDrag();
  } catch (err) {
    const type = payload || fallbackType;
    if (!type) return;
    if (!node.value) return;
    if (!canAcceptChild(node.value, type)) return;

    // 计算插入位置
    let insertIndex = (node.value?.children || []).length;

    if (node.value?.type === "FlexContainer") {
      const direction = node.value.props?.direction || "column";
      const currentElement = event.currentTarget;
      const insertInfo = dragDropManager.calculateFlexInsertPosition(
        currentElement,
        event,
        direction
      );
      insertIndex = insertInfo.index;
    }

    const dropPosition =
      node.value?.type === "FreeContainer"
        ? resolveDropOffset(event, event.currentTarget)
        : null;

    // 插入新节点
    editorStore.insertNode(type, node.value?.id, insertIndex, {
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

  if (currentNode.type === "FlexContainer" || currentNode.type === "ResponsiveLayout") {
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
    : rect?.width ?? 120;
  const height = Number.isFinite(absolutePos.h)
    ? absolutePos.h
    : rect?.height ?? 40;

  let x = Number.isFinite(absolutePos.x) ? absolutePos.x : 0;
  let y = Number.isFinite(absolutePos.y) ? absolutePos.y : 0;

  if (!Number.isFinite(absolutePos.x) || !Number.isFinite(absolutePos.y)) {
    const parentElement = nodeRef.value?.parentElement?.closest?.("[data-node-id]");
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
const handlePointerDown = (event) => {
  if (props.readonly) return;
  if (activeDragHandlers) return;
  if (!node.value || !isMovable.value) return;
  if (event.pointerType === "mouse" && event.button !== 0) return;
  if (isInteractiveTarget(event.target)) return;

  event.preventDefault();
  event.stopPropagation();

  if (selection.value) {
    const element = createSelectableElement("node", node.value.id);
    selection.value.select(element);
  }

  const zoomValue = Number(canvasZoom?.value) || 1;
  const baseLayout = resolveAbsoluteLayout(node.value);
  const startClientX = event.clientX;
  const startClientY = event.clientY;

  cleanupDragHandlers();
  const originalUserSelect = document.body.style.userSelect;
  document.body.style.userSelect = "none";

  if (history.value && !history.value.isInTransaction?.()) {
    history.value.beginTransaction();
  }

  const usePointer = event.type === "pointerdown";
  const pointerTarget =
    event.target instanceof Element ? event.target : nodeRef.value?.$el;
  if (usePointer && pointerTarget?.setPointerCapture && event.pointerId !== undefined) {
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
      return;
    }
    if (history.value?.execute) {
      history.value.execute(new UpdateNodeCommand(node.value.id, patch));
      return;
    }
    if (doc.value?._updateNode) {
      doc.value._updateNode(node.value.id, patch);
    }
  };

  const up = () => {
    cleanupDragHandlers();
    if (history.value?.isInTransaction?.()) {
      history.value.commitTransaction("移动组件");
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
</style>
