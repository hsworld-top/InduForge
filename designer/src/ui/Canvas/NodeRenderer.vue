<template>
  <component
    v-if="node && isNodeVisible"
    :is="outerTag"
    :class="nodeClass"
    :style="outerStyle"
    :data-node-id="node.id"
    :data-node-type="node.type"
    :id="useComponentWrapper ? nodeDomId : null"
    :ref="setNodeRef"
    v-bind="useComponentWrapper ? resolvedProps : {}"
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
      v-if="!useComponentWrapper"
      :is="renderTag"
      :key="renderKey"
      :id="!useComponentWrapper ? nodeDomId : null"
      :style="contentStyleWithConfig"
      v-bind="resolvedProps"
      v-on="mergedEventListeners"
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
          :disabled="Boolean(option.disabled)"
          v-show="option.visible !== false"
        >
          {{ option.label }}
        </el-radio>
      </template>
      <template v-if="node?.type === 'Checkbox'">
        <el-checkbox
          v-for="option in checkboxOptions"
          :key="option.value ?? option.label"
          :label="option.value"
          :disabled="Boolean(option.disabled)"
          v-show="option.visible !== false"
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
          <div
            class="tabs-pane-body"
            :class="{ 'is-drop-active': isDropActive }"
            :data-node-id="node.id"
            :data-node-type="node.type"
            @dragover.prevent="handleDragOver"
            @dragleave="handleDragLeave"
            @drop.prevent="handleDrop"
          >
            <template v-if="isActiveTab(tab)">
              <template
                v-if="isContainer && activeTabChildIds.length === 0 && !props.isRoot"
              >
                <div class="empty-container-hint">
                  <span v-if="isDropActive">释放以添加组件</span>
                  <span v-else>拖拽组件到此处</span>
                </div>
              </template>
              <span
                v-if="activeTabChildIds.length === 0 && tab.content"
                class="tabs-pane-placeholder"
              >
                {{ tab.content }}
              </span>
              <NodeRenderer
                v-for="childId in activeTabChildIds"
                :key="childId"
                :node-id="childId"
                :readonly="props.readonly"
              />
            </template>
            <template v-else>
              <span class="tabs-pane-placeholder">
                {{ tab.content }}
              </span>
            </template>
          </div>
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
          node?.type !== 'Tabs' &&
          !(
            props.readonly &&
            (isRegionContainer ||
              node?.type === 'ElLayout' ||
              node?.type === 'ElLayoutRow' ||
              node?.type === 'ElCol')
          )
        "
      >
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
      <teleport
        v-if="showInsertLine && insertLineStyle && insertLineBox"
        to="body"
      >
        <div
          class="insert-line"
          :class="insertLineStyle.orientation"
          :style="{
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
          }"
        />
      </teleport>
      <div
        v-else-if="showInsertLine && insertLineStyle"
        class="insert-line"
        :class="insertLineStyle.orientation"
        :style="{
          [insertLineStyle.orientation === 'horizontal' ? 'top' : 'left']:
            insertLineStyle.offset + 'px',
        }"
      />
      <NodeRenderer
        v-for="childId in node.children || []"
        v-if="node?.type !== 'Tabs'"
        :key="childId"
        :node-id="childId"
        :readonly="props.readonly"
      />
    </component>
    <template v-if="useComponentWrapper">
      <div
        v-if="
          isContainer &&
          !hasChildren &&
          !props.isRoot &&
          !(
            props.readonly &&
            (isRegionContainer ||
              node?.type === 'ElLayout' ||
              node?.type === 'ElLayoutRow' ||
              node?.type === 'ElCol')
          )
        "
        class="empty-container-hint"
        :class="{ 'is-region-hint': isRegionContainer }"
      >
        <span v-if="isDropActive">释放以添加组件</span>
        <span v-else>{{
          isRegionContainer ? regionHintText : "拖拽组件到此处"
        }}</span>
      </div>
      <!-- 插入线指示器 -->
      <teleport
        v-if="showInsertLine && insertLineStyle && insertLineBox"
        to="body"
      >
        <div
          class="insert-line"
          :class="insertLineStyle.orientation"
          :style="{
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
          }"
        />
      </teleport>
      <div
        v-else-if="showInsertLine && insertLineStyle"
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

<script setup>
import {
  computed,
  ref,
  inject,
  nextTick,
  onBeforeUnmount,
  onMounted,
  watch,
  watchEffect,
} from "vue";
import { storeToRefs } from "pinia";
import { ElMessage } from "element-plus";
import { useEditorStore } from "@/stores/editor-store";
import { datacenterApi } from "@/services";
import { evaluate, evaluateTemplate } from "@/data";
import { getPreviewRuntime } from "@/ui/Preview/previewRuntime";
import {
  componentRegistry,
  createSelectableElement,
  UpdateNodeCommand,
  MoveNodeCommand,
} from "@/editor-core";
import { normalizeEventDefinitions } from "@/editor-core/registry/componentEvents.js";
import { createDragDropManager } from "./DragDropManager";
import EChart from "./components/EChart.vue";
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
  error,
} = storeToRefs(editorStore);
const canvasZoom = inject("canvasZoom", ref(1));
const dragState = useDragState();
const nodeRef = ref(null);
const setNodeRef = (el) => {
  nodeRef.value = el?.$el || el;
};
const contentRef = ref(null);
const previewPageId = computed(
  () => currentPage.value?.name || currentPage.value?.id || ""
);
let registerTimer = null;
let registerAttempts = 0;
const maxRegisterAttempts = 10;
const notifyInsertFailure = (fallbackMessage) => {
  const message = error.value || fallbackMessage || "插入失败：当前不可编辑";
  ElMessage.warning(message);
};

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
  { name: "tab1", label: "标签一", content: "" },
  { name: "tab2", label: "标签二", content: "" },
  { name: "tab3", label: "标签三", content: "" },
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
    ElLayout: ["columns", "rows"],
    ElLayoutRow: ["columns"],
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
const rowInsertInfo = ref(null);
const layoutInsertInfo = ref(null);
const rowInsertEdgeThreshold = 8;
const colInsertEdgeThreshold = 8;
const insertLineBox = computed(
  () => rowInsertInfo.value?.lineBox || layoutInsertInfo.value?.lineBox || null
);
watch(
  () => dragState.dragType,
  (value) => {
    if (!value) {
      showInsertLine.value = false;
      insertLineStyle.value = null;
      rowInsertInfo.value = null;
      layoutInsertInfo.value = null;
    }
  }
);
const dragDropManager = createDragDropManager();
const tableRenderVersion = ref(0);
const resolvedNodeProps = computed(() => {
  docVersion.value;
  if (!node.value) return {};
  const baseProps = node.value.props || {};
  const bindingValues = resolveExprBindings(node.value.bindings, baseProps);
  if (Object.keys(bindingValues).length === 0) return baseProps;
  return { ...baseProps, ...bindingValues };
});
const resolvedProps = computed(() => {
  docVersion.value;
  if (!node.value) return {};
  if (node.value.type === "Table" || node.value.type === "BigDataTable") {
    tableRenderVersion.value;
  }
  const nextProps = filterRenderProps(
    node.value.type,
    resolvedNodeProps.value || {}
  );
  if (node.value.type === "Tabs") {
    const hasModelValue =
      Object.prototype.hasOwnProperty.call(nextProps, "modelValue") &&
      nextProps.modelValue !== undefined &&
      nextProps.modelValue !== null &&
      String(nextProps.modelValue).trim();
    const hasActiveName =
      Object.prototype.hasOwnProperty.call(nextProps, "activeName") &&
      nextProps.activeName !== undefined &&
      nextProps.activeName !== null &&
      String(nextProps.activeName).trim();
    if (!hasModelValue && hasActiveName) {
      nextProps.modelValue = nextProps.activeName;
    }
    if (!hasModelValue && !hasActiveName && activeTabName.value) {
      nextProps.modelValue = activeTabName.value;
    }
  }
  if (node.value.type === "ElCol") {
    const parentNode = doc.value?.getParent?.(node.value.id);
    const rawColumns = Number(parentNode?.props?.columns);
    if (
      parentNode?.type === "ElLayoutRow" &&
      Number.isFinite(rawColumns) &&
      !Object.prototype.hasOwnProperty.call(nextProps, "span")
    ) {
      const columns = Math.max(1, Math.min(24, rawColumns));
      const colIds = (parentNode.children || []).filter((childId) => {
        const childNode = doc.value?.getNode?.(childId);
        return childNode?.type === "ElCol";
      });
      const colIndex = Math.max(0, colIds.indexOf(node.value.id));
      const offsets = colIds.map((colId) => {
        const colNode = doc.value?.getNode?.(colId);
        const offset = Number(colNode?.props?.offset) || 0;
        return Math.max(0, Math.min(24, offset));
      });
      // 按剩余格数等分列宽，避免只压缩右侧区域
      const totalOffset = offsets.reduce((sum, value) => sum + value, 0);
      const remainingUnits = Math.max(columns, 24 - totalOffset);
      const base = Math.floor(remainingUnits / columns);
      const rem = remainingUnits - base * columns;
      const span = base + (colIndex < rem ? 1 : 0);
      nextProps.span = Math.max(1, span);
    }
  }
  return nextProps;
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
  docVersion.value;
  return normalizeOptions(node.value?.props?.tabs, fallbackTabs);
});
const activeTabName = ref("");
const tabHeaderWidth = ref(0);

/**
 * 同步 Tabs 激活项（编辑态跟随点击）
 */
const syncActiveTabName = () => {
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
  const firstTab = tabsList.value?.[0];
  if (!firstTab) return;
  const fallbackName = firstTab.name ?? firstTab.label;
  if (fallbackName) activeTabName.value = String(fallbackName);
};

watch(
  () => [
    node.value?.id,
    resolvedNodeProps.value?.modelValue,
    resolvedNodeProps.value?.activeName,
    tabsList.value?.length,
  ],
  () => {
    syncActiveTabName();
  },
  { immediate: true }
);

/**
 * 同步 Tabs 头部宽度（用于左右布局）
 */
const syncTabsHeaderWidth = () => {
  if (!node.value || node.value.type !== "Tabs") return;
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
  { immediate: true }
);

/**
 * 处理 Tabs 点击事件
 * @param {Object} pane - Tab 面板
 */
const handleTabsClick = (pane) => {
  if (!pane) return;
  const name =
    pane?.props?.name ??
    pane?.name ??
    pane?.paneName ??
    pane?.label ??
    "";
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
      input?.props?.name ??
      input?.name ??
      input?.paneName ??
      input?.label ??
      ""
    );
  }
  return input ?? "";
};

const handleTabsRemove = (name) => {
  if (node.value?.type !== "Tabs") return;
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

  const nextTab =
    tabs[targetIndex] ||
    tabs[targetIndex - 1] ||
    tabs[0] ||
    null;
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

/**
 * Tabs 激活内容区子节点
 * @returns {string[]} 子节点ID列表
 */
const activeTabChildIds = computed(() => {
  docVersion.value;
  if (!node.value || node.value.type !== "Tabs" || !doc.value) return [];
  const current = activeTabName.value;
  return (node.value.children || []).filter((childId) => {
    const childNode = doc.value?.getNode?.(childId);
    if (!childNode) return false;
    const tabKey = childNode.props?.tabKey;
    if (!tabKey) {
      // 兼容历史数据：未标记的默认归属当前激活页
      return Boolean(current);
    }
    return String(tabKey) === String(current);
  });
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

const isNodeVisible = computed(() => {
  docVersion.value;
  if (!node.value) return false;
  if (node.value.hidden) return false;
  const visibleConfig = node.value.conditions?.visible;
  if (typeof visibleConfig === "boolean") return visibleConfig;
  if (typeof visibleConfig !== "string" || !visibleConfig.trim()) return true;
  const context = buildExpressionContext(resolvedNodeProps.value || {});
  const value = resolveExpressionValue(visibleConfig, context, true);
  return Boolean(value);
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
    "ElLayoutRow",
    "ElHeader",
    "ElAside",
    "ElMain",
    "ElFooter",
    "ElCol",
  ].includes(type);
};

/**
 * 判断是否为布局类节点
 * @param {string} type - 组件类型
 * @returns {boolean}
 */
const isLayoutNodeType = (type) => {
  return [
    "ElLayout",
    "ElLayoutRow",
    "ElCol",
    "FlexContainer",
    "GridContainer",
    "FreeContainer",
    "ResponsiveLayout",
    "ColumnLayout1",
    "ColumnLayout2",
    "ColumnLayout4",
  ].includes(type);
};

/**
 * 获取节点所在的 ElLayoutRow 祖先
 * @param {import('@/editor-core').ComponentNode | null} currentNode - 当前节点
 * @returns {import('@/editor-core').ComponentNode | null}
 */
const resolveAncestorLayoutRow = (currentNode) => {
  if (!currentNode || !doc.value) return null;
  if (currentNode.type === "ElLayoutRow") return currentNode;
  let parentNode = doc.value.getParent?.(currentNode.id);
  while (parentNode) {
    if (parentNode.type === "ElLayoutRow") return parentNode;
    parentNode = doc.value.getParent?.(parentNode.id);
  }
  return null;
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
 * 判断是否为根画布下的 ElContainer
 * @param {import('@/editor-core').ComponentNode | null} currentNode - 当前节点
 * @returns {boolean}
 */
const isRootCanvasContainer = (currentNode) => {
  if (!currentNode || currentNode.type !== "ElContainer") return false;
  const rootId = currentPage.value?.rootNodeId;
  if (!rootId) return false;
  const parentNode = doc.value?.getParent?.(currentNode.id);
  return parentNode?.id === rootId;
};

/**
 * 判断节点是否可自由拖动
 * @returns {boolean} 是否允许拖动
 * @throws {Error} 无
 */
const isMovable = computed(() => {
  if (!node.value || props.isRoot || node.value.locked) return false;
  if (isRootCanvasContainer(node.value)) return false;
  if (
    node.value.type === "ElHeader" ||
    node.value.type === "ElAside" ||
    node.value.type === "ElMain" ||
    node.value.type === "ElFooter" ||
    node.value.type === "ElCol" ||
    node.value.type === "ElLayoutRow"
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

const isElColInRow = computed(() => {
  if (!node.value || node.value.type !== "ElCol") return false;
  const parentNode = doc.value?.getParent?.(node.value.id);
  return parentNode?.type === "ElLayoutRow";
});

/**
 * 判断节点是否被放入 ElCol 中
 * @returns {boolean} 是否为 ElCol 子节点
 */
const isChildInElCol = computed(() => {
  if (!node.value) return false;
  const parentNode = doc.value?.getParent?.(node.value.id);
  return parentNode?.type === "ElCol";
});

const visibleResizeHandles = computed(() => {
  if (isChildInElCol.value) return [];
  if (isElColInRow.value) {
    return resizeHandles.filter((handle) =>
      ["e", "w"].includes(handle.key)
    );
  }
  if (node.value?.type === "ElLayoutRow") {
    return resizeHandles.filter((handle) =>
      ["n", "s"].includes(handle.key)
    );
  }
  const config = getRegionResizeConfig(node.value?.type);
  if (!config) return resizeHandles;
  return resizeHandles.filter((handle) => config.handles.includes(handle.key));
});

const showResizeHandles = computed(() => {
  selectionVersion.value;
  if (props.readonly || props.isRoot) return false;
  if (!node.value || node.value.locked) return false;
  if (isRootCanvasContainer(node.value)) return false;
  if (node.value.type === "ElMain") return false;
  if (isChildInElCol.value) return false;
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
  if (parentNode.type === "ElLayout") {
    return childType === "ElLayoutRow";
  }
  if (parentNode.type === "ElLayoutRow") {
    return childType === "ElCol";
  }
  if (parentNode.type === "ElCol") {
    return (parentNode.children || []).length === 0;
  }
  if (!Array.isArray(allowed) || allowed.length === 0) return true;
  return allowed.includes(childType);
};

/**
 * 判断节点是否为可放置容器
 * @param {import('@/editor-core').ComponentNode | null} targetNode - 目标节点
 * @returns {boolean}
 */
const isDroppableContainer = (targetNode) => {
  if (!targetNode) return false;
  if (
    targetNode.type === "ElLayout" ||
    targetNode.type === "ElLayoutRow" ||
    targetNode.type === "ElCol" ||
    targetNode.type === "ElContainer" ||
    targetNode.type === "ElHeader" ||
    targetNode.type === "ElAside" ||
    targetNode.type === "ElMain" ||
    targetNode.type === "ElFooter"
  ) {
    return true;
  }
  const manifest = componentRegistry.get(targetNode.type);
  return Boolean(manifest?.isContainer);
};

/**
 * 从鼠标位置解析可放置容器
 * @param {DragEvent} event - 拖拽事件
 * @param {string} childType - 子组件类型
 * @returns {{ node: import('@/editor-core').ComponentNode, element: HTMLElement | null } | null}
 */
const resolveDropContainer = (event, childType) => {
  if (!doc.value || !event) return null;
  const hitList = document.elementsFromPoint(event.clientX, event.clientY);
  let containerNode = null;
  const isRegionType = (type) =>
    type === "ElHeader" ||
    type === "ElAside" ||
    type === "ElMain" ||
    type === "ElFooter";
  const resolveContainerMain = (container) => {
    if (!container || container.type !== "ElContainer") return null;
    const mainChildId = (container.children || []).find((childId) => {
      const childNode = doc.value?.getNode?.(childId);
      return childNode?.type === "ElMain";
    });
    return mainChildId ? doc.value?.getNode?.(mainChildId) || null : null;
  };

  const canAcceptByAutoInsert = (targetNode, nextChildType) => {
    if (!targetNode || !nextChildType) return false;
    if (targetNode.type === "ElLayout") {
      return nextChildType !== "ElLayoutRow";
    }
    if (targetNode.type === "ElLayoutRow") {
      return nextChildType !== "ElCol";
    }
    if (targetNode.type === "ElCol") {
      return nextChildType !== "ElCol";
    }
    return false;
  };

  for (const hit of hitList) {
    if (!(hit instanceof Element)) continue;
    const nodeElement = hit.closest?.("[data-node-id][data-node-type]");
    if (!nodeElement) continue;
    const nodeId = nodeElement.getAttribute("data-node-id");
    const targetNode = nodeId ? doc.value?.getNode?.(nodeId) : null;
    if (!targetNode) continue;
    if (isRegionType(targetNode.type)) {
      if (childType && !canAcceptChild(targetNode, childType)) continue;
      return { node: targetNode, element: nodeElement };
    }
    if (targetNode.type === "ElContainer") {
      if (!containerNode) {
        containerNode = resolveContainerMain(targetNode) || targetNode;
      }
      continue;
    }
    if (isDroppableContainer(targetNode)) {
      if (childType && !canAcceptChild(targetNode, childType)) {
        if (!canAcceptByAutoInsert(targetNode, childType)) continue;
      }
      return { node: targetNode, element: nodeElement };
    }
  }
  if (containerNode && (!childType || canAcceptChild(containerNode, childType))) {
    const containerElement = document.querySelector(
      `[data-node-id="${containerNode.id}"]`
    );
    return { node: containerNode, element: containerElement };
  }
  return null;
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

/**
 * 获取组件默认尺寸
 * @param {string} type - 组件类型
 * @returns {{ width: number, height: number }}
 */
const resolveDefaultSize = (type) => {
  const manifest = componentRegistry.get(type);
  if (manifest?.defaultSize) {
    return {
      width: manifest.defaultSize.width || 120,
      height: manifest.defaultSize.height || 32,
    };
  }
  const sizeMap = {
    FlexContainer: { width: 360, height: 200 },
    FreeContainer: { width: 360, height: 200 },
    GridContainer: { width: 360, height: 200 },
    ElContainer: { width: 360, height: 240 },
    ElLayout: { width: 360, height: 200 },
    Text: { width: 120, height: 32 },
    Button: { width: 120, height: 36 },
  };
  return sizeMap[type] || { width: 160, height: 80 };
};

/**
 * 限制拖拽放置坐标不超出容器可见范围
 * @param {{ x: number, y: number }} position - 原始落点
 * @param {HTMLElement} element - 目标容器
 * @param {{ width: number, height: number }} size - 组件尺寸
 * @returns {{ x: number, y: number }}
 */
const clampDropPosition = (position, element, size) => {
  const rect = element.getBoundingClientRect();
  const zoomValue = Number(canvasZoom?.value) || 1;
  const maxX = Math.max(0, Math.round(rect.width / zoomValue - size.width));
  const maxY = Math.max(0, Math.round(rect.height / zoomValue - size.height));
  return {
    x: Math.min(Math.max(0, position.x), maxX),
    y: Math.min(Math.max(0, position.y), maxY),
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
  if (node.value.type === "ElLayout") {
    classes.push("el-layout");
  }
  if (node.value.type === "ElLayoutRow") {
    classes.push("el-layout-row");
  }
  if (node.value.type === "Tabs") {
    classes.push("tabs-container");
    const position =
      resolvedNodeProps.value?.tabPosition ||
      node.value.props?.tabPosition ||
      "top";
    classes.push(`tabs-pos-${position}`);
  }
  if (node.value.type === "ElCol") {
    classes.push("el-col");
    const parentNode = doc.value?.getParent?.(node.value.id);
    const gutter = Number(parentNode?.props?.gutter) || 0;
    if (gutter > 0) classes.push("is-guttered");
  }
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
      return "div";
    case "ElLayoutRow":
      return "el-row";
    case "ElCol":
      return "el-col";
    case "EChart":
      return EChart;
    case "Text":
      return resolvedNodeProps.value?.tag || "div";
    default:
      return "div";
  }
});

const useComponentWrapper = computed(() => {
  const type = node.value?.type;
  return type === "ElLayoutRow" || type === "ElCol";
});

const outerTag = computed(() => {
  return useComponentWrapper.value ? renderTag.value : "div";
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
  if (node.value.type === "Tabs") {
    const tabs = Array.isArray(node.value.props?.tabs)
      ? node.value.props.tabs
      : [];
    const tabKey = tabs
      .map((item) => item?.name ?? item?.label ?? "")
      .join("|");
    const activeName =
      node.value.props?.activeName ??
      resolvedNodeProps.value?.activeName ??
      "";
    return `${node.value.id}-${tabKey}-${String(activeName)}`;
  }
  return node.value.id || "";
});

const displayContent = computed(() => {
  docVersion.value;
  if (!node.value) return null;
  const resolvedPropsValue = resolvedNodeProps.value || {};
  if (node.value.type === "Text") {
    return resolvedPropsValue.text ?? node.value.label ?? "";
  }
  if (node.value.type === "Button") {
    return resolvedPropsValue.text ?? node.value.label ?? "按钮";
  }
  if (node.value.type === "Tag") {
    return resolvedPropsValue.text ?? node.value.label ?? "标签";
  }
  if (node.value.type === "Card") {
    return resolvedPropsValue.content ?? node.value.label ?? "卡片";
  }
  if (node.value.type === "BusinessCard") {
    return resolvedPropsValue.content ?? node.value.label ?? "业务卡片";
  }
  if (node.value.type === "WebContainer") {
    return resolvedPropsValue.url
      ? `网页容器: ${resolvedPropsValue.url}`
      : "网页容器";
  }
  if (node.value.type === "Barcode") {
    return resolvedPropsValue.value ?? "1234567890";
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
  const textStyle =
    node.value.type === "Text"
      ? normalizeStyleObject(resolveTextPropStyle(resolvedNodeProps.value || {}))
      : {};
  const parentNode = doc.value?.getParent?.(node.value.id);
  const style = {
    ...containerStyle,
    ...textStyle,
    ...customStyle,
  };
  if (!style.overflow && !isContainer.value) {
    style.overflow = "hidden";
  }
  if (node.value.type === "ElLayoutRow") {
    style.overflow = "visible";
  }
  if (!props.readonly) {
    if (node.value.type === "ElLayout") {
      style.overflow = "hidden";
    } else if (
      node.value.type === "ElLayoutRow" ||
      node.value.type === "ElCol"
    ) {
      style.overflow = "visible";
    }
  }
  if (node.value.type === "ElLayout") {
    const rawPadding = node.value.props?.padding;
    const paddingValue =
      typeof rawPadding === "number" && Number.isFinite(rawPadding)
        ? `${rawPadding}px`
        : rawPadding !== undefined
          ? String(rawPadding)
          : "0px";
    style.padding = paddingValue;
    delete style.minHeight;
  }
  if (
    parentNode?.type === "ElHeader" ||
    parentNode?.type === "ElAside" ||
    parentNode?.type === "ElMain" ||
    parentNode?.type === "ElFooter"
  ) {
    style.overflow = "hidden";
  }
  if (node.value.type === "ElCol" && parentNode?.type === "ElLayoutRow") {
    style.height = "100%";
  }
  if (parentNode?.type === "ElCol") {
    style.width = "100%";
    style.height = "100%";
    style.minHeight = "100%";
    style.alignSelf = "stretch";
    if (!style.display) {
      style.display = "block";
    }
  }
  if (parentNode?.type === "Tabs") {
    style.width = "100%";
    style.height = "100%";
    style.minHeight = "100%";
    style.alignSelf = "stretch";
    style.flex = "1 1 auto";
    if (!style.display) {
      style.display = "block";
    }
  }
  if (parentNode?.type === "ElCol" && isMovable.value) {
    style.width = "100%";
    if (isContainer.value) {
      style.height = "100%";
    } else if (!style.height) {
      style.height = "auto";
    }
    const rowParent = doc.value?.getParent?.(parentNode.id);
    const rowHeight =
      rowParent?.type === "ElLayoutRow"
        ? rowParent.style?.height
        : undefined;
    const normalizedRowHeight =
      rowHeight === undefined || rowHeight === null
        ? ""
        : String(rowHeight).trim();
    const hasFixedRowHeight =
      normalizedRowHeight !== "" && normalizedRowHeight !== "auto";
    if (hasFixedRowHeight && (!style.height || style.height === "auto")) {
      style.height = "100%";
    }
  }
  if (parentNode?.type === "ElCol") {
    const rowParent = doc.value?.getParent?.(parentNode.id);
    const rowHeight =
      rowParent?.type === "ElLayoutRow"
        ? rowParent.style?.height
        : undefined;
    const normalizedRowHeight =
      rowHeight === undefined || rowHeight === null
        ? ""
        : String(rowHeight).trim();
    const hasFixedRowHeight =
      normalizedRowHeight !== "" && normalizedRowHeight !== "auto";
    if (hasFixedRowHeight) {
      style.height = "100%";
    }
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
  if (
    node.value.type === "ElHeader" ||
    node.value.type === "ElAside" ||
    node.value.type === "ElMain" ||
    node.value.type === "ElFooter"
  ) {
    style.overflow = "hidden";
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
  const parentNode = doc.value?.getParent?.(node.value.id);
  const hasCustomWidth = Boolean(customStyle.width);
  const hasCustomHeight = Boolean(customStyle.height);
  if (
    node.value.type === "ElCol" &&
    parentNode?.type === "ElLayoutRow" &&
    !style.height
  ) {
    style.height = "auto";
  }
  if (parentNode?.type === "ElCol" && isMovable.value) {
    style.width = "100%";
    style.height = "100%";
    if (style.position === "absolute") {
      style.left = "0";
      style.top = "0";
    }
  }
  if (parentNode?.type === "Tabs") {
    style.width = "100%";
    style.height = "100%";
    if (style.position === "absolute") {
      style.left = "0";
      style.top = "0";
    }
  }
  if (isContainer.value && isMovable.value) {
    if (!hasCustomWidth && !style.width) style.width = "100%";
    if (!hasCustomHeight && !style.height) style.height = "100%";
  }
  return style;
});

const wrapperComponentStyle = computed(() => {
  if (!useComponentWrapper.value) return {};
  const style = { ...wrapperStyle.value, ...contentStyle.value };
  const type = node.value?.type;
  const parentNode = doc.value?.getParent?.(node.value?.id);
  const customStyle = normalizeStyleObject(node.value?.style || {});
  const hasCustomHeight = Boolean(customStyle.height);
  if (type === "ElLayoutRow" || type === "ElCol") {
    delete style.display;
  }
  if (type === "ElLayoutRow" && parentNode?.type === "ElLayout") {
    style.flex = hasCustomHeight ? "0 0 auto" : "1 1 0";
    style.width = "100%";
    style.minHeight = "0";
    style.maxHeight = hasCustomHeight ? "none" : "100%";
    style.alignItems = "stretch";
    style.overflow = "visible";
    style.columnGap = "0";
    style.gap = "0";
    style.padding = "0";
    style.paddingTop = "0";
    style.paddingRight = "0";
    style.paddingBottom = "0";
    style.paddingLeft = "0";
  }
  if (type === "ElLayoutRow") {
    const gutter = Number(node.value?.props?.gutter) || 0;
    const columns = Math.max(
      1,
      Math.min(24, Number(node.value?.props?.columns) || 1)
    );
    style["--row-gutter"] = `${Math.max(0, gutter)}px`;
    style["--row-columns"] = String(columns);
    style.paddingLeft = "0";
    style.paddingRight = "0";
    if (!gutter) {
      style["--el-row-gutter"] = "0px";
      style.paddingTop = "0";
      style.paddingBottom = "0";
      style.marginLeft = "0";
      style.marginRight = "0";
      style.columnGap = "0px";
      style.gap = "0px";
      style.margin = "0";
      style.padding = "0";
    }
    if (gutter > 0) {
      style.marginLeft = "0";
      style.marginRight = "0";
    }
    style.boxSizing = "border-box";
    style.flexWrap = "nowrap";
    style.overflow = "visible";
  }
  if (type === "ElCol" && parentNode?.type === "ElLayoutRow") {
    delete style.flexGrow;
    delete style.flexShrink;
    delete style.flexBasis;
    style.alignSelf = "stretch";
    style.boxSizing = "border-box";
    style.maxHeight = "100%";
    style.overflow = "visible";
    style.minHeight = "0";
    style.padding = "0";
    style.height = "100%";
    const rowHeight =
      parentNode?.style?.height !== undefined
        ? parentNode.style.height
        : undefined;
    const normalizedRowHeight =
      rowHeight === undefined || rowHeight === null
        ? ""
        : String(rowHeight).trim();
    const hasFixedRowHeight =
      normalizedRowHeight !== "" && normalizedRowHeight !== "auto";
    if (hasFixedRowHeight) {
      style.height = "100%";
    }
    const colProps = resolvedProps.value || node.value?.props || {};
    const gutter = Number(parentNode.props?.gutter) || 0;
    if (!gutter) {
      style.paddingLeft = "0";
      style.paddingRight = "0";
      style["--col-gutter-x"] = "0px";
      style.paddingTop = "0";
      style.paddingBottom = "0";
      style.marginLeft = "0";
      style.marginRight = "0";
      style.margin = "0";
      style.padding = "0";
    } else {
      const halfGutter = "calc(var(--row-gutter) / 2)";
      const halfCol = "calc(100% / var(--row-columns) / 2)";
      const clampedHalf = `min(${halfGutter}, ${halfCol})`;
      style.paddingLeft = clampedHalf;
      style.paddingRight = clampedHalf;
      style["--col-gutter-x"] = clampedHalf;
    }
    const offset = Math.max(0, Math.min(24, Number(colProps?.offset) || 0));
    if (offset > 0) {
      style.marginLeft = `${(offset / 24) * 100}%`;
    }
    const span = Math.max(1, Math.min(24, Number(colProps?.span) || 24));
    const cols = (parentNode?.children || []).filter((childId) => {
      const childNode = doc.value?.getNode?.(childId);
      return childNode?.type === "ElCol";
    });
    const colCount = Math.max(1, cols.length);
    const spans = cols.map((colId) => {
      const colNode = doc.value?.getNode?.(colId);
      const colSpan = Number(colNode?.props?.span);
      if (Number.isFinite(colSpan) && colSpan > 0) {
        return Math.max(1, Math.min(24, colSpan));
      }
      return Math.max(1, Math.floor(24 / colCount));
    });
    const totalSpan = Math.max(
      1,
      spans.reduce((sum, value) => sum + value, 0)
    );
    const percent = `${(span / totalSpan) * 100}%`;
    style.flex = `0 0 ${percent}`;
    style.maxWidth = percent;
    style.width = percent;
    const push = Math.max(0, Math.min(24, Number(colProps?.push) || 0));
    const pull = Math.max(0, Math.min(24, Number(colProps?.pull) || 0));
    const shift = push - pull;
    if (shift !== 0) {
      style.position = "relative";
      style.left = `${(shift / 24) * 100}%`;
    }
  }
  return style;
});

const styleConfigText = computed(() => {
  docVersion.value;
  return String(node.value?.styleConfig || "").trim();
});
const hasStyleConfigSelector = computed(() =>
  styleConfigText.value.includes("{")
);
const hasDomIdSelector = computed(() => styleConfigText.value.includes("#domId"));
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

const styleElementRef = ref(null);

/**
 * 注入/更新样式节点，确保样式生效
 */
const syncStyleElement = () => {
  if (typeof document === "undefined") return;
  const nodeId = node.value?.id;
  const cssText = styleConfigCss.value;

  if (!nodeId || !cssText) {
    if (styleElementRef.value?.parentNode) {
      styleElementRef.value.parentNode.removeChild(styleElementRef.value);
    }
    styleElementRef.value = null;
    return;
  }

  let styleEl = styleElementRef.value;
  if (!styleEl || styleEl.getAttribute("data-style-node") !== nodeId) {
    if (styleEl?.parentNode) {
      styleEl.parentNode.removeChild(styleEl);
    }
    styleEl = document.querySelector(`style[data-style-node="${nodeId}"]`);
    if (!styleEl) {
      styleEl = document.createElement("style");
      styleEl.setAttribute("data-style-node", nodeId);
      document.head.appendChild(styleEl);
    }
    styleElementRef.value = styleEl;
  }

  if (styleEl.textContent !== cssText) {
    styleEl.textContent = cssText;
  }
};

watch(
  [() => node.value?.id, styleConfigCss],
  () => {
    syncStyleElement();
  },
  { immediate: true }
);

watchEffect(() => {
  styleConfigCss.value;
  syncStyleElement();
});

onMounted(() => {
  syncStyleElement();
});

onBeforeUnmount(() => {
  if (styleElementRef.value?.parentNode) {
    styleElementRef.value.parentNode.removeChild(styleElementRef.value);
  }
  styleElementRef.value = null;
});

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

/**
 * 构建变量默认值映射
 * @param {Record<string, { default?: any }>} definitions - 变量定义
 * @returns {Record<string, any>}
 */
function buildVarValuesFromDefinitions(definitions) {
  const result = {};
  if (!definitions || typeof definitions !== "object") return result;
  Object.entries(definitions).forEach(([name, detail]) => {
    result[name] = normalizeGlobalValue(detail);
  });
  return result;
}

/**
 * 构建表达式上下文
 * @param {Record<string, any>} propsValue - 当前组件属性
 * @returns {import("@/data").ExpressionContext}
 */
function buildExpressionContext(propsValue = {}) {
  const pageId = currentPage.value?.id;
  const pageDefs = doc.value?.vars?.pages?.[pageId] || {};
  const globalDefs = projectVariables.value || {};
  const runtimeGlobals = getPreviewRuntime()?.globals;
  const globalValues =
    runtimeGlobals && typeof runtimeGlobals === "object"
      ? runtimeGlobals
      : buildVarValuesFromDefinitions(globalDefs);
  return {
    $dp: {},
    $vars: buildVarValuesFromDefinitions(pageDefs),
    $global: globalValues,
    $props: propsValue,
    state: globalValues,
  };
}

/**
 * 解析表达式值
 * @param {string} expr - 表达式
 * @param {import("@/data").ExpressionContext} context - 上下文
 * @param {*} fallback - 兜底值
 * @returns {*}
 */
function resolveExpressionValue(expr, context, fallback) {
  if (typeof expr !== "string" || !expr.trim()) return fallback;
  const text = expr.trim();
  const value = text.includes("{{")
    ? evaluateTemplate(text, context)
    : evaluate(text, context);
  return value === undefined ? fallback : value;
}

/**
 * 解析表达式绑定
 * @param {Record<string, any>} bindings - 绑定配置
 * @param {Record<string, any>} propsValue - 当前组件属性
 * @returns {Record<string, any>}
 */
function resolveExprBindings(bindings, propsValue) {
  if (!bindings || typeof bindings !== "object") return {};
  const context = buildExpressionContext(propsValue);
  const resolved = {};
  Object.entries(bindings).forEach(([key, binding]) => {
    if (!binding || typeof binding !== "object") return;
    if (binding.kind !== "expr" || typeof binding.expr !== "string") return;
    const value = resolveExpressionValue(
      binding.expr,
      context,
      binding.fallback
    );
    if (value !== undefined) {
      resolved[key] = value;
    }
  });
  return resolved;
}

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
  modelValueTypes.has(node.value?.type)
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
  if (node.value.type !== "Tabs") return {};
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

/**
 * 获取布局节点的最外层 ElLayout
 * @param {import('@/editor-core').ComponentNode | null} currentNode - 当前节点
 * @returns {import('@/editor-core').ComponentNode | null}
 */
const resolveLayoutRootNode = (currentNode) => {
  if (!currentNode) return null;
  if (
    currentNode.type !== "ElLayout" &&
    currentNode.type !== "ElLayoutRow" &&
    currentNode.type !== "ElCol"
  ) {
    return null;
  }
  if (currentNode.type === "ElLayout") return currentNode;
  let parentNode = doc.value?.getParent?.(currentNode.id);
  while (parentNode) {
    if (parentNode.type === "ElLayout") return parentNode;
    parentNode = doc.value?.getParent?.(parentNode.id);
  }
  return null;
};

/**
 * 获取布局节点所在的 ElLayoutRow
 * @param {import('@/editor-core').ComponentNode | null} currentNode - 当前节点
 * @returns {import('@/editor-core').ComponentNode | null}
 */
const resolveLayoutRowNode = (currentNode) => {
  if (!currentNode) return null;
  if (currentNode.type === "ElLayoutRow") return currentNode;
  if (currentNode.type === "ElCol") {
    const parentNode = doc.value?.getParent?.(currentNode.id);
    if (parentNode?.type === "ElLayoutRow") return parentNode;
  }
  return null;
};

/**
 * 判断是否点击在容器边框区域
 * @param {MouseEvent} event - 鼠标事件
 * @returns {boolean}
 */
const isClickOnContainerBorder = (event) => {
  if (!node.value || !isContainer.value) return false;
  const element = nodeRef.value;
  if (!element || !event || typeof event.clientX !== "number") return false;
  const rect = element.getBoundingClientRect?.();
  if (!rect) return false;
  const x = event.clientX;
  const y = event.clientY;
  if (
    x < rect.left ||
    x > rect.right ||
    y < rect.top ||
    y > rect.bottom
  ) {
    return false;
  }
  const edge = 6;
  const nearEdge =
    x - rect.left <= edge ||
    rect.right - x <= edge ||
    y - rect.top <= edge ||
    rect.bottom - y <= edge;
  if (!nearEdge) return false;
  const hitNodeEl = event.target?.closest?.("[data-node-id]");
  if (!hitNodeEl) return true;
  return hitNodeEl.getAttribute("data-node-id") === node.value.id;
};

const handleClick = (event) => {
  if (props.readonly) {
    void runPreviewScript("click", event);
    return;
  }
  if (node.value?.type === "Tabs") {
    const hitNodeEl = event.target?.closest?.("[data-node-id]");
    const hitNodeId = hitNodeEl?.getAttribute?.("data-node-id");
    if (hitNodeId && hitNodeId !== node.value.id) {
      return;
    }
  }
  if (isClickOnContainerBorder(event)) {
    handleSelect(event);
    return;
  }
  if (event.metaKey || event.ctrlKey) {
    let parentNode = doc.value?.getParent?.(node.value?.id);
    while (parentNode) {
      if (parentNode.type === "ElLayout") {
        const element = createSelectableElement("node", parentNode.id);
        selection.value.select(element);
        return;
      }
      parentNode = doc.value?.getParent?.(parentNode.id);
    }
  }
  let targetNode = resolveClickSelectionTarget(event);
  let forceRowSelection = false;
  const layoutRoot = resolveLayoutRootNode(node.value);
  if (event.metaKey || event.ctrlKey) {
    if (layoutRoot && layoutRoot.id !== node.value?.id) {
      targetNode = layoutRoot;
      forceRowSelection = true;
    }
  } else if (layoutRoot && layoutRoot.id !== node.value?.id) {
    targetNode = layoutRoot;
  }
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
      if (!isLayoutNodeType(node.value.type)) {
        const containerNode = doc.value?.getParent?.(parentNode.id);
        if (containerNode?.type === "ElContainer" && !containerNode.locked) {
          targetNode = containerNode;
        }
      }
    }
  }
  if (!targetNode || !selection.value) return;
  const element = createSelectableElement("node", targetNode.id);
  if (event.shiftKey) {
    selection.value.selectRange(element);
    return;
  }
  if (forceRowSelection && (event.metaKey || event.ctrlKey)) {
    selection.value.select(element);
    return;
  }
  if (!forceRowSelection && (event.metaKey || event.ctrlKey)) {
    selection.value.toggleSelect(element);
    return;
  }
  selection.value.select(element);
};

const handleDoubleClick = (event) => {
  if (props.readonly) return;
  if (!node.value || !selection.value) return;
  if (event && (event.ctrlKey || event.metaKey)) {
    const rowNode = resolveAncestorLayoutRow(node.value);
    if (rowNode && !rowNode.locked) {
      const element = createSelectableElement("node", rowNode.id);
      selection.value.select(element);
      return;
    }
  }
  if (node.value.type !== "ElCol") {
    const parentNode = doc.value?.getParent?.(node.value.id);
    if (parentNode?.type === "ElCol") {
      if (parentNode.locked) return;
      const element = createSelectableElement("node", parentNode.id);
      selection.value.select(element);
      return;
    }
  }
  if (node.value.type === "ElCol") {
    const layoutRoot = resolveLayoutRootNode(node.value);
    if (layoutRoot) {
      const element = createSelectableElement("node", node.value.id);
      selection.value.select(element);
      return;
    }
  }
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
  if (patch.bindings) {
    node.value.bindings = { ...(node.value.bindings || {}), ...patch.bindings };
  }
  if (patch.events) {
    node.value.events = { ...(node.value.events || {}), ...patch.events };
  }
  if (patch.conditions) {
    node.value.conditions = {
      ...(node.value.conditions || {}),
      ...patch.conditions,
    };
  }
  if (patch.permissions) {
    node.value.permissions = {
      ...(node.value.permissions || {}),
      ...patch.permissions,
    };
  }
  if (typeof patch.hidden === "boolean") {
    node.value.hidden = patch.hidden;
  }
  if (patch.label !== undefined) {
    node.value.label = patch.label;
  }
  docVersion.value += 1;
};

const detailConfigText = computed(() => {
  docVersion.value;
  return String(node.value?.detailConfig || "").trim();
});

/**
 * @typedef {Object} ButtonDslOnClick
 * @property {string} action - 点击动作脚本
 * @property {string} [confirm] - 二次确认提示
 */

/**
 * @typedef {Object} ButtonDslConfig
 * @property {string} id - DOM 唯一 ID
 * @property {string} [text] - 静态文字
 * @property {string} [textExpr] - 初始化表达式
 * @property {boolean | string} [visible] - 显隐（支持表达式）
 * @property {string} [permission] - 权限码
 * @property {'primary' | 'success' | 'warning' | 'danger' | 'info' | 'text'} [type] - 类型
 * @property {'large' | 'default' | 'small'} [size] - 尺寸
 * @property {boolean} [plain] - 朴素按钮
 * @property {boolean} [round] - 圆角按钮
 * @property {boolean} [circle] - 圆形按钮
 * @property {boolean} [disabled] - 禁用
 * @property {boolean} [loading] - 加载中
 * @property {string} [icon] - 图标
 * @property {Record<string, string>} [style] - 样式
 * @property {string} [className] - 自定义类名
 * @property {string | ButtonDslOnClick} [onClick] - 点击事件
 * @property {*} [plugin] - 插件扩展
 */

/**
 * 生成按钮 DSL 的点击脚本
 * @param {string | ButtonDslOnClick} onClick - 点击配置
 * @returns {string}
 */
const buildButtonClickScript = (onClick) => {
  if (typeof onClick === "string") return onClick;
  if (!onClick || typeof onClick !== "object") return "";
  const action = typeof onClick.action === "string" ? onClick.action.trim() : "";
  if (!action) return "";
  if (typeof onClick.confirm === "string" && onClick.confirm.trim()) {
    const confirmText = JSON.stringify(onClick.confirm.trim());
    return `if (confirm(${confirmText})) {\n  ${action}\n}`;
  }
  return action;
};

/**
 * 生成按钮 DSL 更新补丁
 * @param {ButtonDslConfig} config - 按钮 DSL 配置
 * @returns {{
 *   propsPatch: Record<string, *>,
 *   stylePatch: Record<string, *>,
 *   bindingsPatch: Record<string, *>,
 *   eventsPatch: Record<string, *>,
 *   conditionsPatch: Record<string, *>,
 *   hidden: boolean | undefined,
 * }}
 */
const buildButtonDslPatch = (config) => {
  const propsPatch = {};
  const stylePatch = {};
  const bindingsPatch = {};
  const eventsPatch = {};
  const conditionsPatch = {};
  let hidden;

  if (!config || typeof config !== "object") {
    return {
      propsPatch,
      stylePatch,
      bindingsPatch,
      eventsPatch,
      conditionsPatch,
      hidden,
    };
  }

  if (Object.prototype.hasOwnProperty.call(config, "id")) {
    const value = String(config.id || "");
    if (value) propsPatch.id = value;
  }
  if (Object.prototype.hasOwnProperty.call(config, "text")) {
    propsPatch.text = String(config.text ?? "");
  }
  if (typeof config.textExpr === "string" && config.textExpr.trim()) {
    bindingsPatch.text = {
      kind: "expr",
      expr: config.textExpr.trim(),
      fallback:
        Object.prototype.hasOwnProperty.call(config, "text") &&
        config.text !== undefined
          ? String(config.text ?? "")
          : undefined,
    };
  }
  if (typeof config.visible === "boolean") {
    hidden = !config.visible;
  } else if (typeof config.visible === "string" && config.visible.trim()) {
    conditionsPatch.visible = config.visible.trim();
  }
  if (typeof config.permission === "string" && config.permission.trim()) {
    propsPatch.permission = config.permission.trim();
  }
  if (typeof config.type === "string" && config.type.trim()) {
    propsPatch.type = config.type.trim();
  }
  if (typeof config.size === "string" && config.size.trim()) {
    propsPatch.size = config.size.trim();
  }
  if (typeof config.plain === "boolean") propsPatch.plain = config.plain;
  if (typeof config.round === "boolean") propsPatch.round = config.round;
  if (typeof config.circle === "boolean") propsPatch.circle = config.circle;
  if (typeof config.disabled === "boolean")
    propsPatch.disabled = config.disabled;
  if (typeof config.loading === "boolean") propsPatch.loading = config.loading;
  if (typeof config.icon === "string" && config.icon.trim()) {
    propsPatch.icon = config.icon.trim();
  }
  if (config.style && typeof config.style === "object") {
    Object.entries(config.style).forEach(([key, value]) => {
      if (typeof value === "string") {
        stylePatch[key] = value;
      }
    });
  }
  if (typeof config.className === "string" && config.className.trim()) {
    propsPatch.class = config.className.trim();
  }
  if (config.onClick) {
    const code = buildButtonClickScript(config.onClick);
    if (code) {
      eventsPatch.click = [{ type: "script", code, enabled: true }];
    }
  }
  if (Object.prototype.hasOwnProperty.call(config, "plugin")) {
    propsPatch.plugin = config.plugin;
  }

  return {
    propsPatch,
    stylePatch,
    bindingsPatch,
    eventsPatch,
    conditionsPatch,
    hidden,
  };
};

const buildRefInfo = () => {
  if (!node.value) return null;
  const dslMethodMap = {
    Input: "input",
    InputNumber: "inputNumber",
    Select: "select",
    Cascader: "cascader",
    Radio: "radio",
    Checkbox: "checkbox",
    Switch: "switch",
    Table: "table",
    BigDataTable: "bigDataTable",
    Tree: "tree",
    Dropdown: "dropdown",
    Menu: "menu",
    Tabs: "tabs",
    Transfer: "transfer",
    Tag: "tag",
    Timeline: "timeline",
    Steps: "steps",
    ImageCarousel: "imageCarousel",
    CarouselComponent: "carouselComponent",
    Card: "card",
    Pagination: "pagination",
    Collapse: "collapse",
    Slider: "slider",
    Calendar: "calendar",
    WebContainer: "webContainer",
    Text: "text",
  };
  const applyCommonDslConfig = (config) => {
    if (!config || typeof config !== "object") return;
    const reservedKeys = new Set([
      "id",
      "label",
      "type",
      "props",
      "state",
      "style",
      "className",
      "events",
      "visible",
      "disabled",
      "loading",
      "permission",
      "onClick",
    ]);
    const rawProps =
      config.props && typeof config.props === "object" ? { ...config.props } : {};
    Object.entries(config).forEach(([key, value]) => {
      if (reservedKeys.has(key)) return;
      if (rawProps[key] === undefined) {
        rawProps[key] = value;
      }
    });
    if (
      Object.prototype.hasOwnProperty.call(rawProps, "value") &&
      !Object.prototype.hasOwnProperty.call(rawProps, "modelValue")
    ) {
      rawProps.modelValue = rawProps.value;
      delete rawProps.value;
    }
    const nextPatch = {};
    if (Object.keys(rawProps).length > 0) {
      nextPatch.props = { ...(node.value.props || {}), ...rawProps };
    }
    if (config.style && typeof config.style === "object") {
      nextPatch.style = { ...(node.value.style || {}), ...config.style };
    }
    if (typeof config.className === "string" && config.className.trim()) {
      const className = config.className.trim();
      nextPatch.props = {
        ...(nextPatch.props || node.value.props || {}),
        class: className,
      };
    }
    if (typeof config.label === "string" && config.label.trim()) {
      nextPatch.label = config.label.trim();
    }
    if (typeof config.visible === "boolean") {
      nextPatch.hidden = !config.visible;
    }
    if (typeof config.disabled === "boolean") {
      nextPatch.props = {
        ...(nextPatch.props || node.value.props || {}),
        disabled: config.disabled,
      };
    }
    if (typeof config.loading === "boolean") {
      nextPatch.props = {
        ...(nextPatch.props || node.value.props || {}),
        loading: config.loading,
      };
    }
    if (Object.keys(nextPatch).length === 0) return;
    if (props.readonly) {
      applyPreviewPatch(nextPatch);
      return;
    }
    editorStore.updateNode(node.value.id, nextPatch);
  };
  /**
   * 规范化下拉菜单项数据
   * @param {Record<string, any>} input - 菜单项数据
   * @returns {Record<string, any> | null} 规范化后的菜单项
   */
  const normalizeDropdownItem = (input) => {
    if (!input || typeof input !== "object") return null;
    const text = input.text ?? input.label ?? "";
    const command = input.command ?? input.value ?? input.key ?? "";
    const disabled = Boolean(input.disabled);
    const divided = Boolean(
      Object.prototype.hasOwnProperty.call(input, "divided")
        ? input.divided
        : input.diveded
    );
    const icon = typeof input.icon === "string" ? input.icon : "";
    return {
      label: text || String(command ?? ""),
      value: command || String(text ?? ""),
      disabled,
      divided,
      icon,
    };
  };
  /**
   * 获取当前下拉菜单项列表
   * @returns {Array} 菜单项列表
   */
  const getDropdownItems = () => {
    const items = node.value?.props?.items;
    return Array.isArray(items) ? [...items] : [];
  };
  /**
   * 更新下拉菜单项列表
   * @param {Array} nextItems - 新的菜单项列表
   * @returns {void}
   */
  const updateDropdownItems = (nextItems) => {
    if (props.readonly) {
      applyPreviewPatch({ props: { items: nextItems } });
      return;
    }
    editorStore.updateNode(node.value.id, {
      props: { ...(node.value.props || {}), items: nextItems },
    });
  };
  /**
   * 更新组件属性
   * @param {Record<string, any>} patch - 属性补丁
   * @returns {void}
   */
  const updateNodeProps = (patch) => {
    if (!patch || typeof patch !== "object") return;
    if (props.readonly) {
      applyPreviewPatch({ props: patch });
      return;
    }
    editorStore.updateNode(node.value.id, {
      props: { ...(node.value.props || {}), ...patch },
    });
  };
  /**
   * 更新组件样式
   * @param {Record<string, any>} patch - 样式补丁
   * @returns {void}
   */
  const updateNodeStyle = (patch) => {
    if (!patch || typeof patch !== "object") return;
    if (props.readonly) {
      applyPreviewPatch({ style: patch });
      return;
    }
    editorStore.updateNode(node.value.id, {
      style: { ...(node.value.style || {}), ...patch },
    });
  };
  /**
   * 更新组件显隐
   * @param {boolean} visible - 是否显示
   * @returns {void}
   */
  const updateNodeVisibility = (visible) => {
    const hidden = !Boolean(visible);
    if (props.readonly) {
      applyPreviewPatch({ hidden });
      return;
    }
    editorStore.updateNode(node.value.id, { hidden });
  };
  /**
   * 获取样式数值
   * @param {string} key - 样式字段
   * @returns {number}
   */
  const getStyleNumber = (key) => {
    const raw = node.value?.style?.[key];
    if (typeof raw === "number") return raw;
    if (typeof raw === "string") {
      const parsed = Number.parseFloat(raw);
      return Number.isFinite(parsed) ? parsed : 0;
    }
    return 0;
  };
  /**
   * 规范化单选/多选项数据
   * @param {any} input - 选项数据
   * @returns {Record<string, any> | null} 规范化后的选项
   */
  const normalizeChoiceOption = (input) => {
    if (input == null) return null;
    if (typeof input === "object") {
      const label = input.label ?? input.text ?? "";
      const value = input.value ?? input.label ?? input.text ?? "";
      return {
        ...input,
        label,
        value,
      };
    }
    const text = String(input);
    return { label: text, value: text };
  };
  /**
   * 获取单选/多选项列表
   * @returns {Array} 选项列表
   */
  const getChoiceOptions = () => {
    const options = node.value?.props?.options;
    if (!Array.isArray(options)) return [];
    return options
      .map((item) => normalizeChoiceOption(item))
      .filter(Boolean);
  };
  /**
   * 更新单选/多选项列表
   * @param {Array} nextOptions - 新的选项列表
   * @returns {void}
   */
  const updateChoiceOptions = (nextOptions) => {
    if (props.readonly) {
      applyPreviewPatch({ props: { options: nextOptions } });
      return;
    }
    editorStore.updateNode(node.value.id, {
      props: { ...(node.value.props || {}), options: nextOptions },
    });
  };
  /**
   * 获取表格数据
   * @returns {Array} 表格数据
   */
  const getTableData = () => {
    const data = node.value?.props?.data;
    return Array.isArray(data) ? [...data] : [];
  };
  /**
   * 更新表格数据
   * @param {Array} nextData - 表格数据
   * @returns {void}
   */
  const updateTableData = (nextData) => {
    if (props.readonly) {
      applyPreviewPatch({ props: { data: nextData } });
      tableRenderVersion.value += 1;
      return;
    }
    editorStore.updateNode(node.value.id, {
      props: { ...(node.value.props || {}), data: nextData },
    });
    tableRenderVersion.value += 1;
  };
  /**
   * 更新树组件数据
   * @param {Array} nextData - 树数据
   * @returns {void}
   */
  const updateTreeData = (nextData) => {
    updateNodeProps({ data: Array.isArray(nextData) ? nextData : [] });
  };
  /**
   * 更新级联选择器数据
   * @param {Array} nextOptions - 级联数据
   * @returns {void}
   */
  const updateCascaderOptions = (nextOptions) => {
    updateNodeProps({ options: Array.isArray(nextOptions) ? nextOptions : [] });
  };
  /**
   * 更新选择器数据
   * @param {Array} nextOptions - 选择器选项
   * @returns {void}
   */
  const updateSelectOptions = (nextOptions) => {
    updateNodeProps({ options: Array.isArray(nextOptions) ? nextOptions : [] });
  };
  /**
   * 更新输入值
   * @param {any} value - 输入值
   * @returns {void}
   */
  const updateInputValue = (value) => {
    updateNodeProps({ modelValue: value });
  };
  /**
   * 更新开关值
   * @param {boolean} value - 开关状态
   * @returns {void}
   */
  const updateSwitchValue = (value) => {
    updateNodeProps({ modelValue: Boolean(value) });
  };
  /**
   * 更新穿梭框数据
   * @param {Array} leftData - 左侧数据
   * @param {Array} rightData - 右侧数据
   * @returns {void}
   */
  const updateTransferData = (leftData, rightData) => {
    updateNodeProps({
      data: Array.isArray(leftData) ? leftData : [],
      modelValue: Array.isArray(rightData) ? rightData : [],
    });
  };
  /**
   * 获取表格行键字段
   * @returns {string}
   */
  const getTableRowKeyProp = () => {
    const propsValue = node.value?.props || {};
    return (
      propsValue.rowKey ||
      propsValue["row-key"] ||
      propsValue.keyField ||
      "id"
    );
  };
  /**
   * 获取表格行键值
   * @param {Record<string, any>} row - 行数据
   * @returns {string|number|undefined}
   */
  const getTableRowKeyValue = (row) => {
    if (!row || typeof row !== "object") return undefined;
    const keyProp = getTableRowKeyProp();
    return row[keyProp];
  };
  /**
   * 获取树节点键字段
   * @returns {string}
   */
  const getTreeNodeKeyProp = () => {
    const propsValue = node.value?.props || {};
    return propsValue.nodeKey || propsValue["node-key"] || "id";
  };
  /**
   * 收集树数据中的节点 key
   * @param {Array} list - 树节点列表
   * @returns {Array<string|number>}
   */
  const collectTreeKeys = (list) => {
    const keys = [];
    const keyProp = getTreeNodeKeyProp();
    const walk = (items) => {
      if (!Array.isArray(items)) return;
      items.forEach((item) => {
        if (!item || typeof item !== "object") return;
        if (Object.prototype.hasOwnProperty.call(item, keyProp)) {
          keys.push(item[keyProp]);
        }
        walk(item.children);
      });
    };
    walk(list);
    return keys;
  };
  const refInfo = {
    get Name() {
      return node.value?.label || "";
    },
    get Comment() {
      const type = node.value?.type || "";
      const manifest = type ? componentRegistry.get(type) : null;
      return manifest?.name || type || "";
    },
    get Location() {
      return {
        get X() {
          return getStyleNumber("left");
        },
        set X(value) {
          const next = Number(value);
          if (!Number.isFinite(next)) return;
          updateNodeStyle({ left: `${next}px` });
        },
        get Y() {
          return getStyleNumber("top");
        },
        set Y(value) {
          const next = Number(value);
          if (!Number.isFinite(next)) return;
          updateNodeStyle({ top: `${next}px` });
        },
      };
    },
    get Size() {
      return {
        get Width() {
          return getStyleNumber("width");
        },
        set Width(value) {
          const next = Number(value);
          if (!Number.isFinite(next)) return;
          updateNodeStyle({ width: `${next}px` });
        },
        get Height() {
          return getStyleNumber("height");
        },
        set Height(value) {
          const next = Number(value);
          if (!Number.isFinite(next)) return;
          updateNodeStyle({ height: `${next}px` });
        },
      };
    },
    get Visible() {
      return !node.value?.hidden;
    },
    set Visible(value) {
      updateNodeVisibility(Boolean(value));
    },
    get Enable() {
      return !Boolean(node.value?.props?.disabled);
    },
    set Enable(value) {
      updateNodeProps({ disabled: !Boolean(value) });
    },
    get Caption() {
      if (node.value?.type === "Button" || node.value?.type === "Tag") {
        return node.value?.props?.text ?? "";
      }
      return undefined;
    },
    set Caption(value) {
      if (node.value?.type === "Button" || node.value?.type === "Tag") {
        updateNodeProps({ text: String(value ?? "") });
      }
    },
    get Image() {
      if (node.value?.type !== "Image") return undefined;
      return node.value?.props?.src ?? "";
    },
    set Image(value) {
      if (node.value?.type !== "Image") return;
      updateNodeProps({ src: String(value ?? "") });
    },
    name: node.value.label,
    id: node.value.id,
    el: nodeRef.value || null,
    component: contentRef.value || null,
    elContainer: () => nodeRef.value || null,
    elMain: () => nodeRef.value || null,
    elLayout: () => nodeRef.value || null,
    elLayoutRow: () => nodeRef.value || null,
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
        echarts: (method, ...args) => {
      if (node.value?.type !== "EChart") return;
      const chartApi = contentRef.value;
      if (chartApi?.callECharts) {
        return chartApi.callECharts(method, ...args);
      }
    },setStyle: (patch) => {
      if (!patch || typeof patch !== "object") return;
      if (props.readonly) {
        applyPreviewPatch({ style: patch });
        return;
      }
      editorStore.updateNode(node.value.id, {
        style: { ...(node.value.style || {}), ...patch },
      });
    },
        setOption: (option, notMergeOrOpts, lazyUpdate = false, silent = false, replaceMerge) => {
      if (node.value?.type !== "EChart") return;
      if (option && typeof option.then === "function") {
        option.then((resolved) => {
          refInfo.setOption?.(resolved, notMergeOrOpts, lazyUpdate, silent, replaceMerge);
        });
        return;
      }
      const chartApi = contentRef.value;
      if (chartApi?.setOption) {
        chartApi.setOption(option, notMergeOrOpts, lazyUpdate, silent, replaceMerge);
      }
      if (props.readonly) {
        applyPreviewPatch({ props: { option } });
        return;
      }
      editorStore.updateNode(node.value.id, {
        props: { ...(node.value.props || {}), option },
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
    SetText: (text) => {
      if (
        node.value?.type !== "Text" &&
        node.value?.type !== "Tag" &&
        node.value?.type !== "Button"
      ) {
        return;
      }
      const value = String(text ?? "");
      updateNodeProps({ text: value });
    },
    GetText: () => {
      if (
        node.value?.type !== "Text" &&
        node.value?.type !== "Tag" &&
        node.value?.type !== "Button"
      ) {
        return undefined;
      }
      return node.value?.props?.text ?? "";
    },
    SetType: (type) => {
      if (
        node.value?.type !== "Button" &&
        node.value?.type !== "Tag" &&
        node.value?.type !== "Text"
      ) {
        return;
      }
      updateNodeProps({ type: String(type ?? "") });
    },
    SetEllipsis: (value) => {
      if (node.value?.type !== "Text") return;
      updateNodeProps({ truncate: Boolean(value) });
    },
    SetTooltip: (value) => {
      if (node.value?.type !== "Text") return;
      updateNodeProps({ showTooltip: Boolean(value) });
    },
    SetLoading: (value) => {
      if (
        node.value?.type !== "Button" &&
        node.value?.type !== "Card" &&
        node.value?.type !== "BusinessCard"
      ) {
        return;
      }
      updateNodeProps({ loading: Boolean(value) });
    },
    SetDisabled: (value) => {
      if (node.value?.type !== "Button") return;
      updateNodeProps({ disabled: Boolean(value) });
    },
    Click: () => {
      if (node.value?.type !== "Button") return;
      const el = contentRef.value?.$el || contentRef.value || nodeRef.value;
      if (el?.click) {
        el.click();
        return;
      }
      contentRef.value?.$emit?.("click");
    },
    SetSrc: (src) => {
      if (node.value?.type !== "Image") return;
      updateNodeProps({ src: String(src ?? "") });
    },
    GetSrc: () => {
      if (node.value?.type !== "Image") return undefined;
      return node.value?.props?.src ?? "";
    },
    Preview: (urls, startIndex = 0) => {
      if (node.value?.type !== "Image") return;
      if (Array.isArray(urls) && urls.length > 0) {
        updateNodeProps({
          previewSrcList: urls,
          initialIndex: Number(startIndex) || 0,
        });
      }
      contentRef.value?.showPreview?.();
    },
    Reload: () => {
      if (node.value?.type === "Image") {
        const src = String(node.value?.props?.src || "");
        if (!src) return;
        const url = new URL(src, window.location.href);
        url.searchParams.set("_t", String(Date.now()));
        updateNodeProps({ src: url.toString() });
        return;
      }
      if (node.value?.type === "WebContainer") {
        const src = String(node.value?.props?.url || "");
        if (!src) return;
        const url = new URL(src, window.location.href);
        url.searchParams.set("_t", String(Date.now()));
        updateNodeProps({ url: url.toString() });
      }
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
    /**
     * 应用按钮 DSL 配置
     * @param {ButtonDslConfig} config - 按钮 DSL 配置
     * @returns {void}
     */
    button: (config) => {
      if (node.value?.type !== "Button") return;
      const patch = buildButtonDslPatch(config);
      const nextPatch = {};
      if (Object.keys(patch.propsPatch).length > 0) {
        nextPatch.props = {
          ...(node.value.props || {}),
          ...patch.propsPatch,
        };
      }
      if (Object.keys(patch.stylePatch).length > 0) {
        nextPatch.style = {
          ...(node.value.style || {}),
          ...patch.stylePatch,
        };
      }
      if (Object.keys(patch.bindingsPatch).length > 0) {
        nextPatch.bindings = {
          ...(node.value.bindings || {}),
          ...patch.bindingsPatch,
        };
      }
      if (Object.keys(patch.eventsPatch).length > 0) {
        nextPatch.events = {
          ...(node.value.events || {}),
          ...patch.eventsPatch,
        };
      }
      if (Object.keys(patch.conditionsPatch).length > 0) {
        nextPatch.conditions = {
          ...(node.value.conditions || {}),
          ...patch.conditionsPatch,
        };
      }
      if (typeof patch.hidden === "boolean") {
        nextPatch.hidden = patch.hidden;
      }
      if (Object.keys(nextPatch).length === 0) return;
      if (props.readonly) {
        applyPreviewPatch(nextPatch);
        return;
      }
      editorStore.updateNode(node.value.id, nextPatch);
    },
    /**
     * 应用 Tabs DSL 配置
     * @param {Record<string, any>} config - Tabs DSL 配置
     * @returns {void}
     */
    tabs: (config) => {
      if (node.value?.type !== "Tabs" || !config || typeof config !== "object") {
        return;
      }
      if (config.type && String(config.type) !== "Tabs") return;
      const nextPatch = {};
      if (Object.prototype.hasOwnProperty.call(config, "label")) {
        nextPatch.label = String(config.label ?? "");
      }
      const propsPatch = { ...(node.value.props || {}) };
      if (Object.prototype.hasOwnProperty.call(config, "id")) {
        const value = String(config.id ?? "");
        if (value) propsPatch.id = value;
      }
      if (config.className && String(config.className).trim()) {
        propsPatch.class = String(config.className).trim();
      }
      if (config.props && typeof config.props === "object") {
        const rawProps = { ...config.props };
        if (Array.isArray(rawProps.items) && !Array.isArray(rawProps.tabs)) {
          rawProps.tabs = rawProps.items;
          delete rawProps.items;
        }
        Object.assign(propsPatch, rawProps);
      }
      nextPatch.props = propsPatch;
      if (config.style && typeof config.style === "object") {
        nextPatch.style = {
          ...(node.value.style || {}),
          ...config.style,
        };
      }
      if (Object.keys(nextPatch).length === 0) return;
      if (props.readonly) {
        applyPreviewPatch(nextPatch);
        return;
      }
      editorStore.updateNode(node.value.id, nextPatch);
    },
    InsertItem: (item) => {
      if (node.value?.type !== "Dropdown") return;
      const normalized = normalizeDropdownItem(item);
      if (!normalized) return;
      const items = getDropdownItems();
      items.push(normalized);
      updateDropdownItems(items);
    },
    GetCommandItem: (menuItem) => {
      if (node.value?.type !== "Dropdown") return undefined;
      const items = getDropdownItems();
      if (menuItem && typeof menuItem === "object") {
        return menuItem.value ?? menuItem.command ?? menuItem.key;
      }
      const found = items.find(
        (item) =>
          item?.label === menuItem ||
          item?.text === menuItem ||
          item?.value === menuItem
      );
      return found?.value;
    },
    GetMenuItem: (command) => {
      if (node.value?.type !== "Dropdown") return undefined;
      const items = getDropdownItems();
      return items.find((item) => item?.value === command);
    },
    DeleteItem: (menuItem) => {
      if (node.value?.type !== "Dropdown") return;
      const items = getDropdownItems();
      const index = items.findIndex(
        (item) =>
          item?.label === menuItem ||
          item?.text === menuItem ||
          item?.value === menuItem
      );
      if (index === -1) return;
      items.splice(index, 1);
      updateDropdownItems(items);
    },
    ClearAll: () => {
      if (node.value?.type === "Dropdown") {
        updateDropdownItems([]);
        return;
      }
      if (node.value?.type === "Cascader") {
        updateCascaderOptions([]);
        updateInputValue([]);
        return;
      }
      if (node.value?.type === "Select") {
        const multiple = Boolean(node.value?.props?.multiple);
        updateInputValue(multiple ? [] : "");
        return;
      }
    },
    UpdateKeyChildren: (key, data) => {
      if (node.value?.type !== "Tree") return;
      contentRef.value?.updateKeyChildren?.(key, data);
    },
    GetCheckedNodes: (leafOnly, includeHalfChecked) => {
      if (node.value?.type === "Tree") {
        return contentRef.value?.getCheckedNodes?.(leafOnly, includeHalfChecked);
      }
      if (node.value?.type === "Cascader") {
        return contentRef.value?.getCheckedNodes?.(leafOnly);
      }
      if (node.value?.type === "Select") {
        const options = getChoiceOptions();
        const value = node.value?.props?.modelValue;
        if (Array.isArray(value)) {
          return options.filter((item) => value.includes(item.value));
        }
        const hit = options.find((item) => item.value === value);
        return hit ? [hit] : [];
      }
      return undefined;
    },
    SetCheckedNodes: (nodes) => {
      if (node.value?.type === "Tree") {
        contentRef.value?.setCheckedNodes?.(nodes);
        return;
      }
      if (node.value?.type === "Select") {
        const options = getChoiceOptions();
        const target = options.find((item) => item.label === nodes);
        updateNodeProps({ modelValue: target?.value ?? "" });
      }
    },
    GetCheckedKeys: (leafOnly) => {
      if (node.value?.type !== "Tree") return undefined;
      return contentRef.value?.getCheckedKeys?.(leafOnly);
    },
    SetCheckedKeys: (keys, leafOnly) => {
      if (node.value?.type !== "Tree") return;
      contentRef.value?.setCheckedKeys?.(keys, leafOnly);
    },
    SetChecked: (keyOrData, checked, deep) => {
      if (node.value?.type !== "Tree") return;
      contentRef.value?.setChecked?.(keyOrData, checked, deep);
    },
    GetHalfCheckedNodes: () => {
      if (node.value?.type !== "Tree") return undefined;
      return contentRef.value?.getHalfCheckedNodes?.();
    },
    GetHalfCheckedKeys: () => {
      if (node.value?.type !== "Tree") return undefined;
      return contentRef.value?.getHalfCheckedKeys?.();
    },
    GetCurrentKey: () => {
      if (node.value?.type !== "Tree") return undefined;
      return contentRef.value?.getCurrentKey?.();
    },
    GetCurrentNode: () => {
      if (node.value?.type !== "Tree") return undefined;
      return contentRef.value?.getCurrentNode?.();
    },
    SetCurrentKey: (key) => {
      if (node.value?.type !== "Tree") return;
      contentRef.value?.setCurrentKey?.(key);
    },
    SetCurrentNode: (nodeData) => {
      if (node.value?.type !== "Tree") return;
      contentRef.value?.setCurrentNode?.(nodeData);
    },
    GetNode: (dataOrKey) => {
      if (node.value?.type !== "Tree") return undefined;
      return contentRef.value?.getNode?.(dataOrKey);
    },
    Remove: (dataOrNode) => {
      if (node.value?.type !== "Tree") return;
      contentRef.value?.remove?.(dataOrNode);
    },
    Append: (data, parentNode) => {
      if (node.value?.type !== "Tree") return;
      contentRef.value?.append?.(data, parentNode);
    },
    InsertBefore: (data, refNode) => {
      if (node.value?.type !== "Tree") return;
      contentRef.value?.insertBefore?.(data, refNode);
    },
    InsertAfter: (data, refNode) => {
      if (node.value?.type !== "Tree") return;
      contentRef.value?.insertAfter?.(data, refNode);
    },
    ExpandAll: () => {
      if (node.value?.type !== "Tree") return;
      const data = node.value?.props?.data || [];
      const keys = collectTreeKeys(data);
      contentRef.value?.setExpandedKeys?.(keys);
    },
    CollapseAll: () => {
      if (node.value?.type !== "Tree") return;
      contentRef.value?.setExpandedKeys?.([]);
    },
    SetExpandedKeys: (keys) => {
      if (node.value?.type !== "Tree") return;
      contentRef.value?.setExpandedKeys?.(Array.isArray(keys) ? keys : []);
    },
    GetExpandedKeys: () => {
      if (node.value?.type !== "Tree") return [];
      return contentRef.value?.getExpandedKeys?.() || [];
    },
    Filter: (keyword) => {
      if (node.value?.type !== "Tree") return;
      contentRef.value?.filter?.(keyword ?? "");
    },
    Open: (index) => {
      if (node.value?.type === "Menu") {
        contentRef.value?.open?.(index);
        return;
      }
      if (node.value?.type === "Dropdown") {
        contentRef.value?.handleOpen?.();
        updateNodeProps({ visible: true });
        return;
      }
      if (node.value?.type === "Select" || node.value?.type === "Cascader") {
        contentRef.value?.toggleMenu?.(true);
        contentRef.value?.togglePopperVisible?.(true);
        contentRef.value?.focus?.();
        return;
      }
      if (node.value?.type === "Collapse") {
        const names = Array.isArray(index) ? index : [index];
        updateNodeProps({ modelValue: names });
      }
    },
    Close: (index) => {
      if (node.value?.type === "Menu") {
        contentRef.value?.close?.(index);
        return;
      }
      if (node.value?.type === "Dropdown") {
        contentRef.value?.handleClose?.();
        updateNodeProps({ visible: false });
        return;
      }
      if (node.value?.type === "Select" || node.value?.type === "Cascader") {
        contentRef.value?.toggleMenu?.(false);
        contentRef.value?.togglePopperVisible?.(false);
        contentRef.value?.blur?.();
        return;
      }
      if (node.value?.type === "Tag") {
        updateNodeVisibility(false);
        return;
      }
      if (node.value?.type === "Collapse") {
        const current = Array.isArray(node.value?.props?.modelValue)
          ? node.value.props.modelValue
          : [];
        const remove = new Set(Array.isArray(index) ? index : [index]);
        updateNodeProps({
          modelValue: current.filter((name) => !remove.has(name)),
        });
      }
    },
    Toggle: () => {
      if (node.value?.type === "Dropdown") {
        const visible = Boolean(node.value?.props?.visible);
        if (visible) {
          refInfo.Close();
          return;
        }
        refInfo.Open();
        return;
      }
      if (node.value?.type === "Cascader" || node.value?.type === "Select") {
        contentRef.value?.toggleMenu?.();
        contentRef.value?.togglePopperVisible?.();
      }
      if (node.value?.type === "Switch") {
        const current = Boolean(node.value?.props?.modelValue);
        updateSwitchValue(!current);
      }
      if (node.value?.type === "Collapse") {
        const name = node.value?.props?.modelValue?.[0];
        if (name !== undefined) {
          refInfo.Close([name]);
        }
      }
    },
    GetVisible: () => {
      if (node.value?.type !== "Dropdown") return undefined;
      return Boolean(node.value?.props?.visible);
    },
    Focus: () => {
      if (
        node.value?.type !== "Input" &&
        node.value?.type !== "Select" &&
        node.value?.type !== "Cascader" &&
        node.value?.type !== "InputNumber" &&
        node.value?.type !== "Button"
      ) {
        return;
      }
      const el = contentRef.value?.$el || contentRef.value || nodeRef.value;
      el?.focus?.();
      contentRef.value?.focus?.();
    },
    Blur: () => {
      if (
        node.value?.type !== "Input" &&
        node.value?.type !== "Select" &&
        node.value?.type !== "Cascader" &&
        node.value?.type !== "InputNumber" &&
        node.value?.type !== "Button"
      ) {
        return;
      }
      const el = contentRef.value?.$el || contentRef.value || nodeRef.value;
      el?.blur?.();
      contentRef.value?.blur?.();
    },
    Select: () => {
      if (node.value?.type !== "Input" && node.value?.type !== "InputNumber") {
        return;
      }
      contentRef.value?.select?.();
    },
    GetInputValue: () => {
      if (node.value?.type !== "Input" && node.value?.type !== "InputNumber") {
        return undefined;
      }
      return node.value?.props?.modelValue ?? "";
    },
    SetInputValue: (value) => {
      if (node.value?.type === "Input") {
        updateInputValue(String(value ?? ""));
        return;
      }
      if (node.value?.type === "InputNumber") {
        const next = Number(value);
        if (!Number.isFinite(next)) return;
        updateInputValue(next);
      }
    },
    ClearQuery: (area) => {
      if (node.value?.type !== "Transfer") return;
      contentRef.value?.clearQuery?.(area);
    },
    SetValue: (value) => {
      if (node.value?.type === "Switch") {
        updateSwitchValue(value);
        return;
      }
      if (node.value?.type === "Input") {
        updateInputValue(String(value ?? ""));
        return;
      }
      if (node.value?.type === "InputNumber") {
        const next = Number(value);
        if (!Number.isFinite(next)) return;
        updateInputValue(next);
        return;
      }
      if (node.value?.type === "Select" || node.value?.type === "Cascader") {
        updateInputValue(value);
        return;
      }
      if (node.value?.type === "Radio") {
        updateNodeProps({ modelValue: value });
        return;
      }
      if (node.value?.type === "Checkbox") {
        updateNodeProps({ modelValue: Array.isArray(value) ? value : [] });
        return;
      }
      if (node.value?.type === "Slider") {
        const next = Number(value);
        if (!Number.isFinite(next)) return;
        updateNodeProps({ modelValue: next });
        return;
      }
      if (node.value?.type === "Transfer") {
        updateNodeProps({ modelValue: Array.isArray(value) ? value : [] });
        return;
      }
      if (node.value?.type === "Barcode") {
        updateNodeProps({ value: String(value ?? "") });
      }
    },
    GetValue: () => {
      if (node.value?.type === "Switch") {
        return Boolean(node.value?.props?.modelValue);
      }
      if (
        node.value?.type === "Input" ||
        node.value?.type === "InputNumber" ||
        node.value?.type === "Select" ||
        node.value?.type === "Cascader" ||
        node.value?.type === "Radio" ||
        node.value?.type === "Slider"
      ) {
        return node.value?.props?.modelValue ?? "";
      }
      if (node.value?.type === "Checkbox" || node.value?.type === "Transfer") {
        return Array.isArray(node.value?.props?.modelValue)
          ? node.value.props.modelValue
          : [];
      }
      if (node.value?.type === "Barcode") {
        return node.value?.props?.value ?? "";
      }
      return undefined;
    },
    Clear: () => {
      if (
        node.value?.type === "Input" ||
        node.value?.type === "InputNumber" ||
        node.value?.type === "Select" ||
        node.value?.type === "Cascader"
      ) {
        updateInputValue(
          node.value?.type === "Select" && node.value?.props?.multiple
            ? []
            : ""
        );
        return;
      }
      if (node.value?.type === "Radio") {
        updateNodeProps({ modelValue: "" });
        return;
      }
      if (node.value?.type === "Checkbox" || node.value?.type === "Transfer") {
        updateNodeProps({ modelValue: [] });
        return;
      }
      if (node.value?.type === "Timeline") {
        updateNodeProps({ items: [] });
        return;
      }
      if (node.value?.type === "Signature") {
        updateInputValue("");
      }
    },
    ClearSelection: () => {
      if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
        return;
      }
      contentRef.value?.clearSelection?.();
    },
    AppendRow: (row) => {
      if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
        return;
      }
      const data = getTableData();
      data.push(row);
      updateTableData(data);
    },
    ToggleRowSelection: (row, selected) => {
      if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
        return;
      }
      if (contentRef.value?.toggleRowSelection) {
        contentRef.value.toggleRowSelection(row, selected);
      }
    },
    ToggleAllSelection: () => {
      if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
        return;
      }
      contentRef.value?.toggleAllSelection?.();
    },
    ToggleRowExpansion: (row, expanded) => {
      if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
        return;
      }
      if (contentRef.value?.toggleRowExpansion) {
        contentRef.value.toggleRowExpansion(row, expanded);
      }
    },
    SetCurrentRow: (row) => {
      if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
        return;
      }
      contentRef.value?.setCurrentRow?.(row);
    },
    ClearSort: () => {
      if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
        return;
      }
      contentRef.value?.clearSort?.();
    },
    ClearFilter: (columnKeys) => {
      if (node.value?.type === "Table" || node.value?.type === "BigDataTable") {
        if (typeof columnKeys === "undefined") {
          contentRef.value?.clearFilter?.();
          return;
        }
        contentRef.value?.clearFilter?.(columnKeys);
        return;
      }
      if (node.value?.type === "Tree") {
        contentRef.value?.filter?.("");
      }
    },
    Dolayout: () => {
      if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
        return;
      }
      contentRef.value?.doLayout?.();
    },
    Sort: (prop, order) => {
      if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
        return;
      }
      contentRef.value?.sort?.(prop, order);
    },
    GetSelection: () => {
      if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
        return [];
      }
      return contentRef.value?.getSelectionRows?.() || [];
    },
    GetSelectionKeys: () => {
      if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
        return [];
      }
      const selected = contentRef.value?.getSelectionRows?.() || [];
      return selected
        .map((row) => getTableRowKeyValue(row))
        .filter((value) => value !== undefined);
    },
    SetPage: (page) => {
      const next = Number(page);
      if (!Number.isFinite(next)) return;
      if (node.value?.type === "Pagination") {
        updateNodeProps({ currentPage: next });
        return;
      }
      if (node.value?.type === "Table" || node.value?.type === "BigDataTable") {
        updateNodeProps({ currentPage: next, page: next });
      }
    },
    SetPageSize: (size) => {
      const next = Number(size);
      if (!Number.isFinite(next)) return;
      if (node.value?.type === "Pagination") {
        updateNodeProps({ pageSize: next });
        return;
      }
      if (node.value?.type === "Table" || node.value?.type === "BigDataTable") {
        updateNodeProps({ pageSize: next });
      }
    },
    GetPageData: () => {
      if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
        return [];
      }
      const data = getTableData();
      const page =
        Number(node.value?.props?.currentPage ?? node.value?.props?.page) || 1;
      const size = Number(node.value?.props?.pageSize) || data.length || 1;
      const start = Math.max(0, (page - 1) * size);
      return data.slice(start, start + size);
    },
    UpdateRowByKey: (key, patch) => {
      if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
        return;
      }
      const data = getTableData();
      const index = data.findIndex((row) => getTableRowKeyValue(row) === key);
      if (index === -1) return;
      data[index] = { ...data[index], ...(patch || {}) };
      updateTableData(data);
    },
    RemoveRowByKey: (key) => {
      if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
        return;
      }
      const data = getTableData().filter(
        (row) => getTableRowKeyValue(row) !== key
      );
      updateTableData(data);
    },
    UpsertRowByKey: (key, row) => {
      if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
        return;
      }
      const data = getTableData();
      const index = data.findIndex((item) => getTableRowKeyValue(item) === key);
      if (index === -1) {
        data.push(row);
      } else {
        data[index] = { ...data[index], ...(row || {}) };
      }
      updateTableData(data);
    },
    ScrollToTop: () => {
      if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
        return;
      }
      contentRef.value?.setScrollTop?.(0);
      const wrapper = contentRef.value?.$el?.querySelector?.(
        ".el-scrollbar__wrap"
      );
      if (wrapper) wrapper.scrollTop = 0;
    },
    ScrollToRow: (keyOrRow) => {
      if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
        return;
      }
      if (contentRef.value?.scrollTo) {
        contentRef.value.scrollTo(keyOrRow);
        return;
      }
      const data = getTableData();
      const key =
        typeof keyOrRow === "object"
          ? getTableRowKeyValue(keyOrRow)
          : keyOrRow;
      const index = data.findIndex((row) => getTableRowKeyValue(row) === key);
      if (index < 0) return;
      const wrapper = contentRef.value?.$el?.querySelector?.(
        ".el-scrollbar__wrap"
      );
      if (wrapper) {
        wrapper.scrollTop = index * 32;
      }
    },
    DoLayoutSafe: () => {
      if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
        return;
      }
      nextTick(() => {
        contentRef.value?.doLayout?.();
      });
    },
    SetData: (data, rightData) => {
      if (node.value?.type === "Table" || node.value?.type === "BigDataTable") {
        updateTableData(Array.isArray(data) ? data : []);
        return;
      }
      if (node.value?.type === "Cascader") {
        updateCascaderOptions(data);
        return;
      }
      if (node.value?.type === "Select") {
        updateSelectOptions(data);
        return;
      }
      if (node.value?.type === "Transfer") {
        updateTransferData(data, rightData);
        return;
      }
      if (node.value?.type === "Tree") {
        updateTreeData(data);
        return;
      }
      if (node.value?.type === "BusinessCard") {
        updateNodeProps({ data });
      }
    },
    GetData: () => {
      if (node.value?.type === "Table" || node.value?.type === "BigDataTable") {
        return getTableData();
      }
      if (node.value?.type === "Cascader") {
        return node.value?.props?.options ?? [];
      }
      if (node.value?.type === "Select") {
        return node.value?.props?.options ?? [];
      }
      if (node.value?.type === "Transfer") {
        return {
          leftData: node.value?.props?.data ?? [],
          rightData: node.value?.props?.modelValue ?? [],
        };
      }
      if (node.value?.type === "Tree") {
        return node.value?.props?.data ?? [];
      }
      if (node.value?.type === "BusinessCard") {
        return node.value?.props?.data;
      }
      return undefined;
    },
    GetRadioChecked: () => {
      if (node.value?.type !== "Radio") return undefined;
      const options = getChoiceOptions();
      const value = node.value?.props?.modelValue;
      return options.findIndex((item) => item?.value === value);
    },
    GetRadioValue: (labelIndex) => {
      if (node.value?.type !== "Radio") return undefined;
      const options = getChoiceOptions();
      return options?.[Number(labelIndex)]?.value;
    },
    GetRadioLabel: (radioValue) => {
      if (node.value?.type !== "Radio") return undefined;
      const options = getChoiceOptions();
      return options.find((item) => item?.value === radioValue)?.label;
    },
    SetRadioEnable: (labelIndex, enable) => {
      if (node.value?.type !== "Radio") return;
      const options = getChoiceOptions();
      const index = Number(labelIndex);
      if (!options[index]) return;
      options[index] = { ...options[index], disabled: !Boolean(enable) };
      updateChoiceOptions(options);
    },
    GetRadioEnable: (labelIndex) => {
      if (node.value?.type !== "Radio") return undefined;
      const options = getChoiceOptions();
      const index = Number(labelIndex);
      if (!options[index]) return undefined;
      return !options[index].disabled;
    },
    SetRadioVisible: (labelIndex, visible) => {
      if (node.value?.type !== "Radio") return;
      const options = getChoiceOptions();
      const index = Number(labelIndex);
      if (!options[index]) return;
      options[index] = { ...options[index], visible: Boolean(visible) };
      updateChoiceOptions(options);
    },
    GetRadioVisible: (labelIndex) => {
      if (node.value?.type !== "Radio") return undefined;
      const options = getChoiceOptions();
      const index = Number(labelIndex);
      if (!options[index]) return undefined;
      return options[index].visible !== false;
    },
    GetCheckState: (labelIndex) => {
      if (node.value?.type !== "Checkbox") return undefined;
      const options = getChoiceOptions();
      const index = Number(labelIndex);
      const option = options[index];
      const values = Array.isArray(node.value?.props?.modelValue)
        ? node.value.props.modelValue
        : [];
      if (!option) return undefined;
      return values.includes(option.value);
    },
    SetCheckState: (labelIndex, state) => {
      if (node.value?.type !== "Checkbox") return;
      const options = getChoiceOptions();
      const index = Number(labelIndex);
      const option = options[index];
      if (!option) return;
      const values = Array.isArray(node.value?.props?.modelValue)
        ? [...node.value.props.modelValue]
        : [];
      const exists = values.includes(option.value);
      if (Boolean(state) && !exists) values.push(option.value);
      if (!Boolean(state) && exists) {
        const nextValues = values.filter((value) => value !== option.value);
        if (props.readonly) {
          applyPreviewPatch({ props: { modelValue: nextValues } });
          return;
        }
        editorStore.updateNode(node.value.id, {
          props: { ...(node.value.props || {}), modelValue: nextValues },
        });
        return;
      }
      if (props.readonly) {
        applyPreviewPatch({ props: { modelValue: values } });
        return;
      }
      editorStore.updateNode(node.value.id, {
        props: { ...(node.value.props || {}), modelValue: values },
      });
    },
    SetCheckEnable: (labelIndex, enable) => {
      if (node.value?.type !== "Checkbox") return;
      const options = getChoiceOptions();
      const index = Number(labelIndex);
      if (!options[index]) return;
      options[index] = { ...options[index], disabled: !Boolean(enable) };
      updateChoiceOptions(options);
    },
    GetCheckEnable: (labelIndex) => {
      if (node.value?.type !== "Checkbox") return undefined;
      const options = getChoiceOptions();
      const index = Number(labelIndex);
      if (!options[index]) return undefined;
      return !options[index].disabled;
    },
    SetCheckVisible: (labelIndex, visible) => {
      if (node.value?.type !== "Checkbox") return;
      const options = getChoiceOptions();
      const index = Number(labelIndex);
      if (!options[index]) return;
      options[index] = { ...options[index], visible: Boolean(visible) };
      updateChoiceOptions(options);
    },
    GetCheckVisible: (labelIndex) => {
      if (node.value?.type !== "Checkbox") return undefined;
      const options = getChoiceOptions();
      const index = Number(labelIndex);
      if (!options[index]) return undefined;
      return options[index].visible !== false;
    },
    CheckAll: (value) => {
      if (node.value?.type !== "Checkbox") return;
      const options = getChoiceOptions();
      const next = Boolean(value) ? options.map((item) => item.value) : [];
      updateNodeProps({ modelValue: next });
    },
    SetActive: (name) => {
      if (node.value?.type === "Menu") {
        updateNodeProps({ defaultActive: String(name ?? "") });
        return;
      }
      if (node.value?.type === "Tabs") {
        updateNodeProps({ activeName: String(name ?? "") });
        return;
      }
      if (node.value?.type === "Steps") {
        const next = Number(name);
        if (!Number.isFinite(next)) return;
        updateNodeProps({ active: next });
      }
    },
    GetActive: () => {
      if (node.value?.type === "Menu") {
        return node.value?.props?.defaultActive ?? "";
      }
      if (node.value?.type === "Tabs") {
        return node.value?.props?.activeName ?? "";
      }
      return undefined;
    },
    Collapse: (value) => {
      if (node.value?.type === "Menu") {
        updateNodeProps({ collapse: Boolean(value) });
        return;
      }
      if (node.value?.type === "Card") {
        updateNodeProps({ collapsed: Boolean(value) });
      }
    },
    Next: () => {
      if (node.value?.type === "Tabs") {
        const tabs = Array.isArray(node.value?.props?.tabs)
          ? node.value.props.tabs
          : [];
        const current = node.value?.props?.activeName;
        const index = tabs.findIndex((item) => item.name === current);
        const next = tabs[index + 1] || tabs[0];
        if (next?.name) updateNodeProps({ activeName: next.name });
        return;
      }
      if (node.value?.type === "Steps") {
        const active = Number(node.value?.props?.active) || 0;
        updateNodeProps({ active: active + 1 });
        return;
      }
      if (node.value?.type === "ImageCarousel" || node.value?.type === "CarouselComponent") {
        contentRef.value?.next?.();
      }
    },
    Prev: () => {
      if (node.value?.type === "Tabs") {
        const tabs = Array.isArray(node.value?.props?.tabs)
          ? node.value.props.tabs
          : [];
        const current = node.value?.props?.activeName;
        const index = tabs.findIndex((item) => item.name === current);
        const prev = tabs[index - 1] || tabs[tabs.length - 1];
        if (prev?.name) updateNodeProps({ activeName: prev.name });
        return;
      }
      if (node.value?.type === "Steps") {
        const active = Number(node.value?.props?.active) || 0;
        updateNodeProps({ active: Math.max(0, active - 1) });
        return;
      }
      if (node.value?.type === "ImageCarousel" || node.value?.type === "CarouselComponent") {
        contentRef.value?.prev?.();
      }
    },
    AddTab: (tab) => {
      if (node.value?.type !== "Tabs") return;
      const tabs = Array.isArray(node.value?.props?.tabs)
        ? [...node.value.props.tabs]
        : [];
      tabs.push(tab);
      updateNodeProps({ tabs });
    },
    RemoveTab: (name) => {
      if (node.value?.type !== "Tabs") return;
      const tabs = Array.isArray(node.value?.props?.tabs)
        ? node.value.props.tabs.filter((item) => item.name !== name)
        : [];
      updateNodeProps({ tabs });
    },
    SetSelectionByKeys: (keys) => {
      if (node.value?.type !== "Table" && node.value?.type !== "BigDataTable") {
        return;
      }
      const data = getTableData();
      const keySet = new Set(Array.isArray(keys) ? keys : []);
      contentRef.value?.clearSelection?.();
      data.forEach((row) => {
        const rowKey = getTableRowKeyValue(row);
        if (keySet.has(rowKey)) {
          contentRef.value?.toggleRowSelection?.(row, true);
        }
      });
    },
    MoveToRight: (keys) => {
      if (node.value?.type !== "Transfer") return;
      const current = Array.isArray(node.value?.props?.modelValue)
        ? [...node.value.props.modelValue]
        : [];
      const nextKeys = Array.isArray(keys) ? keys : [];
      nextKeys.forEach((key) => {
        if (!current.includes(key)) current.push(key);
      });
      updateNodeProps({ modelValue: current });
    },
    MoveToLeft: (keys) => {
      if (node.value?.type !== "Transfer") return;
      const current = Array.isArray(node.value?.props?.modelValue)
        ? [...node.value.props.modelValue]
        : [];
      const remove = new Set(Array.isArray(keys) ? keys : []);
      const next = current.filter((key) => !remove.has(key));
      updateNodeProps({ modelValue: next });
    },
    Increase: (step) => {
      if (node.value?.type !== "InputNumber") return;
      const current = Number(node.value?.props?.modelValue) || 0;
      const delta = Number(step ?? node.value?.props?.step ?? 1);
      updateInputValue(current + (Number.isFinite(delta) ? delta : 1));
    },
    Decrease: (step) => {
      if (node.value?.type !== "InputNumber") return;
      const current = Number(node.value?.props?.modelValue) || 0;
      const delta = Number(step ?? node.value?.props?.step ?? 1);
      updateInputValue(current - (Number.isFinite(delta) ? delta : 1));
    },
    SetItems: (items) => {
      if (
        node.value?.type !== "Timeline" &&
        node.value?.type !== "ImageCarousel" &&
        node.value?.type !== "CarouselComponent" &&
        node.value?.type !== "Steps"
      ) {
        return;
      }
      updateNodeProps({ items: Array.isArray(items) ? items : [] });
    },
    AppendItem: (item) => {
      if (node.value?.type !== "Timeline") return;
      const items = Array.isArray(node.value?.props?.items)
        ? [...node.value.props.items]
        : [];
      items.push(item);
      updateNodeProps({ items });
    },
    Play: () => {
      if (
        node.value?.type !== "ImageCarousel" &&
        node.value?.type !== "CarouselComponent"
      ) {
        return;
      }
      updateNodeProps({ autoplay: true });
    },
    Pause: () => {
      if (
        node.value?.type !== "ImageCarousel" &&
        node.value?.type !== "CarouselComponent"
      ) {
        return;
      }
      updateNodeProps({ autoplay: false });
    },
    SetActiveItem: (nameOrIndex) => {
      if (
        node.value?.type !== "ImageCarousel" &&
        node.value?.type !== "CarouselComponent"
      ) {
        return;
      }
      contentRef.value?.setActiveItem?.(nameOrIndex);
    },
    Load: (url) => {
      if (node.value?.type !== "WebContainer") return;
      updateNodeProps({ url: String(url ?? "") });
    },
    PostMessage: (_data) => {
      if (node.value?.type !== "WebContainer") return;
    },
    GetUrl: () => {
      if (node.value?.type !== "WebContainer") return undefined;
      return node.value?.props?.url ?? "";
    },
    Back: () => {
      if (node.value?.type !== "WebContainer") return;
    },
    Forward: () => {
      if (node.value?.type !== "WebContainer") return;
    },
    Reset: () => {
      if (node.value?.type === "Steps") {
        updateNodeProps({ active: 1 });
        return;
      }
      if (node.value?.type === "Pagination") {
        updateNodeProps({ currentPage: 1 });
        return;
      }
      if (node.value?.type === "Slider") {
        const min = Number(node.value?.props?.min);
        updateNodeProps({ modelValue: Number.isFinite(min) ? min : 0 });
      }
    },
    SetTitle: (title) => {
      if (node.value?.type !== "Card" && node.value?.type !== "BusinessCard") {
        return;
      }
      updateNodeProps({ title: String(title ?? "") });
    },
    GetPage: () => {
      if (node.value?.type !== "Pagination") return undefined;
      return Number(node.value?.props?.currentPage) || 1;
    },
    GetPageSize: () => {
      if (node.value?.type !== "Pagination") return undefined;
      return Number(node.value?.props?.pageSize) || 0;
    },
    SetTotal: (total) => {
      if (node.value?.type !== "Pagination") return;
      const next = Number(total);
      if (!Number.isFinite(next)) return;
      updateNodeProps({ total: next });
    },
    GetTotal: () => {
      if (node.value?.type !== "Pagination") return undefined;
      return Number(node.value?.props?.total) || 0;
    },
    GetActiveNames: () => {
      if (node.value?.type !== "Collapse") return [];
      return Array.isArray(node.value?.props?.modelValue)
        ? node.value.props.modelValue
        : [];
    },
    SetActiveNames: (names) => {
      if (node.value?.type !== "Collapse") return;
      updateNodeProps({ modelValue: Array.isArray(names) ? names : [] });
    },
    Refresh: () => {
      if (node.value?.type !== "BusinessCard") return;
      updateNodeProps({ refreshAt: Date.now() });
    },
    OpenDetail: (_id) => {
      if (node.value?.type !== "BusinessCard") return;
    },
    Render: () => {
      if (node.value?.type !== "Barcode") return;
    },
    Download: (_format) => {
      if (node.value?.type !== "Barcode") return undefined;
      return undefined;
    },
    Disable: (value) => {
      if (
        node.value?.type !== "Switch" &&
        node.value?.type !== "Slider"
      ) {
        return;
      }
      updateNodeProps({ disabled: Boolean(value) });
    },
    SetDate: (date) => {
      if (node.value?.type !== "Calendar") return;
      updateNodeProps({ date });
    },
    GetDate: () => {
      if (node.value?.type !== "Calendar") return undefined;
      return node.value?.props?.date ?? null;
    },
    Today: () => {
      if (node.value?.type !== "Calendar") return;
      updateNodeProps({ date: new Date() });
    },
    ClearSignature: () => {
      if (node.value?.type !== "Signature") return;
      updateInputValue("");
    },
    GetImage: () => {
      if (node.value?.type !== "Signature") return undefined;
      return node.value?.props?.modelValue ?? "";
    },
    SetImage: (value) => {
      if (node.value?.type !== "Signature") return;
      updateInputValue(String(value ?? ""));
    },
    IsEmpty: () => {
      if (node.value?.type !== "Signature") return true;
      const value = node.value?.props?.modelValue;
      return !value;
    },
    SetPen: (_color, _width) => {
      if (node.value?.type !== "Signature") return;
    },
  };
  const currentType = node.value.type;
  const dslMethodName = dslMethodMap[currentType];
  if (
    dslMethodName &&
    !Object.prototype.hasOwnProperty.call(refInfo, dslMethodName)
  ) {
    refInfo[dslMethodName] = (config) => {
      if (node.value?.type !== currentType) return;
      applyCommonDslConfig(config);
    };
  }
  return refInfo;
};

let detailConfigTimer = null;
let lastDetailConfigKey = "";

const runDetailConfigScript = async (code) => {
  if (!code || !code.trim()) return;
  const runtime = getPreviewRuntime();
  const pageId = currentPage.value?.name || currentPage.value?.id;
  const instance = buildRefInfo();
  if (!instance) return;
  if (runtime?.runCode) {
    await runtime.runCode(code, { type: "detail" }, instance, pageId);
    return;
  }
  const globals = buildPreviewGlobals();
  const customScripts = buildPreviewCustomScripts(globals);
  const context = {
    $event: { type: "detail" },
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
    await runner.call(instance || null, ...values);
  } catch (error) {
    console.error("[Preview] Detail script error:", error);
  }
};

const scheduleDetailConfig = (force = false) => {
  if (!props.readonly) return;
  const code = detailConfigText.value;
  if (!code) return;
  const key = `${node.value?.id || ""}::${code}`;
  if (!force && key === lastDetailConfigKey) return;
  lastDetailConfigKey = key;
  if (detailConfigTimer) {
    clearTimeout(detailConfigTimer);
  }
  detailConfigTimer = setTimeout(() => {
    runDetailConfigScript(code);
  }, 120);
};

watch(
  () => [props.readonly, node.value?.id, detailConfigText.value],
  () => {
    scheduleDetailConfig(true);
  },
  { immediate: true }
);

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
  if (detailConfigTimer) {
    clearTimeout(detailConfigTimer);
    detailConfigTimer = null;
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
    // 若已选中外层容器且当前节点是其子孙，保持外层选中
    const primary = selection.value.getPrimaryElement?.();
    let keepSelection = false;
    if (primary?.kind === "node" && doc.value) {
      let parent = doc.value.getParent?.(node.value.id);
      while (parent) {
        if (parent.id === primary.id) {
          keepSelection = true;
          break;
        }
        parent = doc.value.getParent?.(parent.id);
      }
    }
    if (!keepSelection) {
      const element = createSelectableElement("node", node.value.id);
      selection.value.select(element);
    }
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

  const resolveLayoutInsertFromPoint = () => {
    const hitList = document.elementsFromPoint(
      event.clientX,
      event.clientY
    );
    for (const hit of hitList) {
      const layoutElement = hit?.closest?.(
        '[data-node-type="ElLayout"][data-node-id]'
      );
      if (!layoutElement) continue;
      const layoutId = layoutElement.getAttribute("data-node-id");
      const layoutNode = layoutId ? doc.value?.getNode?.(layoutId) : null;
      if (!layoutNode || layoutNode.type !== "ElLayout") continue;

      const rowIds = (layoutNode.children || []).filter((childId) => {
        const childNode = doc.value?.getNode?.(childId);
        return childNode?.type === "ElLayoutRow";
      });
      if (rowIds.length === 0) return null;

      const pointY = event.clientY;
      const threshold = rowInsertEdgeThreshold;
      for (let i = 0; i < rowIds.length; i += 1) {
        const rowId = rowIds[i];
        const rowElement = layoutElement.querySelector(
          `[data-node-id="${rowId}"]`
        );
        if (!rowElement) continue;
        const rect = rowElement.getBoundingClientRect?.();
        if (!rect) continue;
        if (Math.abs(pointY - rect.top) <= threshold) {
          return { layoutNode, layoutElement, index: i, lineY: rect.top };
        }
        if (Math.abs(pointY - rect.bottom) <= threshold) {
          return { layoutNode, layoutElement, index: i + 1, lineY: rect.bottom };
        }
      }
    }
    return null;
  };

  const resolveRowInsertFromPath = () => {
    const path = event.composedPath?.() || [];
    for (const item of path) {
      if (!(item instanceof Element)) continue;
      const nodeElement = item.closest?.("[data-node-id][data-node-type]");
      if (!nodeElement) continue;
      const nodeType = nodeElement.getAttribute("data-node-type");
      if (nodeType !== "ElCol") continue;
      const nodeId = nodeElement.getAttribute("data-node-id");
      const colNode = nodeId ? doc.value?.getNode?.(nodeId) : null;
      if (!colNode) continue;
      const parentNode = doc.value?.getParent?.(colNode.id);
      if (parentNode?.type !== "ElLayoutRow") continue;
      const rowSelector = `[data-node-id="${parentNode.id}"]`;
      const rowElement =
        document.querySelector(rowSelector) ||
        nodeElement.closest?.(rowSelector);
      if (!rowElement) continue;
      const colRect = nodeElement.getBoundingClientRect?.();
      const edgeThreshold = colInsertEdgeThreshold;
      const nearEdge = colRect
        ? event.clientX - colRect.left <= edgeThreshold ||
          colRect.right - event.clientX <= edgeThreshold
        : false;
      return { rowElement, parentNode, nearEdge, colRect };
    }
    return null;
  };
  const resolveLayoutInsertFromPath = () => {
    const path = event.composedPath?.() || [];
    for (const item of path) {
      if (!(item instanceof Element)) continue;
      const nodeElement = item.closest?.("[data-node-id][data-node-type]");
      if (!nodeElement) continue;
      const nodeType = nodeElement.getAttribute("data-node-type");
      if (nodeType !== "ElLayoutRow") continue;
      const nodeId = nodeElement.getAttribute("data-node-id");
      const rowNode = nodeId ? doc.value?.getNode?.(nodeId) : null;
      if (!rowNode) continue;
      const parentNode = doc.value?.getParent?.(rowNode.id);
      if (parentNode?.type !== "ElLayout") continue;
      const layoutSelector = `[data-node-id="${parentNode.id}"]`;
      const layoutElement =
        document.querySelector(layoutSelector) ||
        nodeElement.closest?.(layoutSelector);
      if (!layoutElement) continue;
      const rowRect = nodeElement.getBoundingClientRect?.();
      const edgeThreshold = rowInsertEdgeThreshold;
      const nearEdge = rowRect
        ? event.clientY - rowRect.top <= edgeThreshold ||
          rowRect.bottom - event.clientY <= edgeThreshold
        : false;
      return { layoutElement, parentNode, rowElement: nodeElement, nearEdge };
    }
    return null;
  };

  if (!isContainer.value) return;
  // ✅ 阻止事件冒泡
  event.stopPropagation();
  const hasComponent =
    event.dataTransfer?.types?.includes("application/x-designer-component") ||
    event.dataTransfer?.types?.includes("application/x-designer-node") ||
    event.dataTransfer?.types?.includes("text/plain") ||
    Boolean(dragState.dragType);
  if (!hasComponent) return;

  if (event.dataTransfer) {
    event.dataTransfer.dropEffect = "copy";
  }
  isDragOver.value = true;

  // ElLayoutRow 内拖入组件时，优先提示左右插入
  const payload =
    event.dataTransfer?.getData("application/x-designer-component") ||
    event.dataTransfer?.getData("application/x-designer-node") ||
    event.dataTransfer?.getData("text/plain");
  const fallbackType = dragState.dragType || "";
  let dragType = "";
  if (payload) {
    try {
      const parsed = JSON.parse(payload);
      dragType = parsed?.type || "";
    } catch (error) {
      dragType = payload;
    }
  }
  dragType = dragType || fallbackType;

  if (dragType && dragType !== "ElLayoutRow") {
    const resolvedLayout = resolveLayoutInsertFromPoint();
    if (resolvedLayout) {
      const { layoutElement, layoutNode, index, lineY } = resolvedLayout;
      const layoutRect = layoutElement.getBoundingClientRect?.();
      if (layoutRect) {
        showInsertLine.value = true;
        insertLineStyle.value = {
          orientation: "horizontal",
          offset: Math.max(0, lineY - layoutRect.top),
        };
        rowInsertInfo.value = null;
        layoutInsertInfo.value = {
          layoutId: layoutNode.id,
          index,
          lineBox: {
            left: layoutRect.left,
            top: layoutRect.top,
            width: layoutRect.width,
            height: layoutRect.height,
          },
        };
        return;
      }
    } else {
      layoutInsertInfo.value = null;
    }
  }

  if (dragType && dragType !== "ElLayoutRow") {
    const resolvedLayout = resolveLayoutInsertFromPath();
    if (resolvedLayout) {
      const { layoutElement, parentNode, nearEdge } = resolvedLayout;
      if (!nearEdge) {
        showInsertLine.value = false;
        insertLineStyle.value = null;
        layoutInsertInfo.value = null;
      } else {
        const layoutRect = layoutElement.getBoundingClientRect?.();
        if (layoutRect) {
          const direction = resolveFlexDirection("ElLayout", layoutElement);
          const insertInfo = dragDropManager.calculateFlexInsertPosition(
            layoutElement,
            event,
            direction
          );
          const insertLine =
            insertInfo.insertLine || {
              orientation: "horizontal",
              offset: layoutRect.bottom - layoutRect.top,
            };
          const adjustedLine = {
            ...insertLine,
            offset: Math.max(0, insertLine.offset),
          };
          showInsertLine.value = true;
          insertLineStyle.value = adjustedLine;
          rowInsertInfo.value = null;
          layoutInsertInfo.value = {
            layoutId: parentNode.id,
            index: insertInfo.index,
            lineBox: {
              left: layoutRect.left,
              top: layoutRect.top,
              width: layoutRect.width,
              height: layoutRect.height,
            },
          };
          return;
        }
      }
    } else {
      layoutInsertInfo.value = null;
    }
  }

  if (dragType && dragType !== "ElCol") {
    const resolvedRow = resolveRowInsertFromPath();
    if (resolvedRow) {
      const { rowElement, parentNode, nearEdge } = resolvedRow;
      if (!nearEdge) {
        showInsertLine.value = false;
        insertLineStyle.value = null;
        rowInsertInfo.value = null;
        layoutInsertInfo.value = null;
        return;
      }
      const rowRect = rowElement.getBoundingClientRect?.();
      if (rowRect) {
        const direction = resolveFlexDirection("ElLayoutRow", rowElement);
        const insertInfo = dragDropManager.calculateFlexInsertPosition(
          rowElement,
          event,
          direction
        );
        const rowRectSnapshot = rowRect;
        const insertLine =
          insertInfo.insertLine || {
            orientation: "vertical",
            offset: rowRectSnapshot.right - rowRectSnapshot.left,
          };
        const hostRect = rowElement.getBoundingClientRect?.();
        if (hostRect && insertLine) {
          const rawOffset =
            insertLine.orientation === "vertical"
              ? insertLine.offset
              : insertLine.offset;
          const adjustedLine = {
            ...insertLine,
            offset: Math.max(0, rawOffset),
          };
          showInsertLine.value = true;
          insertLineStyle.value = adjustedLine;
          rowInsertInfo.value = {
            rowId: parentNode.id,
            index: insertInfo.index,
            lineBox: {
              left: rowRectSnapshot.left,
              top: rowRectSnapshot.top,
              width: rowRectSnapshot.width,
              height: rowRectSnapshot.height,
            },
          };
          layoutInsertInfo.value = null;
          return;
        }
      }
    }
  }

  // ElCol 侧边插入提示由上面的 resolveRowInsertFromPath 处理

  // 计算插入位置
  if (node.value?.type && isFlexDropContainer(node.value.type)) {
    rowInsertInfo.value = null;
    layoutInsertInfo.value = null;
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
  rowInsertInfo.value = null;
  layoutInsertInfo.value = null;
};

/**
 * 处理拖拽放置
 * @param {DragEvent} event - 拖拽事件
 */
const handleDrop = (event) => {
  if (props.readonly) return;
  // ✅ 阻止事件冒泡，避免重复插入
  event.stopPropagation();

  const rowInsertSnapshot = rowInsertInfo.value;
  const layoutInsertSnapshot = layoutInsertInfo.value;
  isDragOver.value = false;
  showInsertLine.value = false;
  insertLineStyle.value = null;
  rowInsertInfo.value = null;
  layoutInsertInfo.value = null;

  const resolveLayoutInsertFromPoint = () => {
    const hit = event.target instanceof Element ? event.target : null;
    const layoutElement = hit?.closest?.('[data-node-type="ElLayout"][data-node-id]');
    if (!layoutElement) return null;
    const layoutId = layoutElement.getAttribute("data-node-id");
    const layoutNode = layoutId ? doc.value?.getNode?.(layoutId) : null;
    if (!layoutNode || layoutNode.type !== "ElLayout") return null;

    const rowIds = (layoutNode.children || []).filter((childId) => {
      const childNode = doc.value?.getNode?.(childId);
      return childNode?.type === "ElLayoutRow";
    });
    if (rowIds.length === 0) return null;

    const pointY = event.clientY;
    const threshold = rowInsertEdgeThreshold;
    for (let i = 0; i < rowIds.length; i += 1) {
      const rowId = rowIds[i];
      const rowElement = layoutElement.querySelector(`[data-node-id="${rowId}"]`);
      if (!rowElement) continue;
      const rect = rowElement.getBoundingClientRect?.();
      if (!rect) continue;
      if (Math.abs(pointY - rect.top) <= threshold) {
        return { layoutNode, layoutElement, index: i };
      }
      if (Math.abs(pointY - rect.bottom) <= threshold) {
        return { layoutNode, layoutElement, index: i + 1 };
      }
    }
    return null;
  };
  /**
   * 在 ElLayout 中按行插入组件
   * @param {import('@/editor-core').ComponentNode} layoutNode - 布局节点
   * @param {number} insertIndex - 行插入位置
   * @param {string} componentType - 组件类型
   */
  const insertIntoLayoutByRow = (layoutNode, insertIndex, componentType) => {
    if (!layoutNode || layoutNode.type !== "ElLayout") return false;
    const rowNode = editorStore.insertNode("ElLayoutRow", layoutNode.id, insertIndex);
    if (!rowNode) return false;
    const latestLayout = doc.value?.getNode?.(layoutNode.id);
    const rowCount = (latestLayout?.children || []).filter((childId) => {
      const childNode = doc.value?.getNode?.(childId);
      return childNode?.type === "ElLayoutRow";
    }).length;
    editorStore.updateNode(layoutNode.id, {
      props: { ...(latestLayout?.props || layoutNode.props || {}), rows: Math.max(1, rowCount) },
    });
    editorStore.updateNode(rowNode.id, {
      props: { ...(rowNode.props || {}), columns: 1 },
    });
    const latestRow = doc.value?.getNode?.(rowNode.id);
    const colIds = (latestRow?.children || []).filter((childId) => {
      const childNode = doc.value?.getNode?.(childId);
      return childNode?.type === "ElCol";
    });
    let colId = colIds[0];
    if (!colId) {
      const colNode = editorStore.insertNode("ElCol", rowNode.id, 0);
      if (!colNode) return false;
      colId = colNode.id;
    }
    return Boolean(editorStore.insertNode(componentType, colId));
  };

  const resolveRowInsertTarget = () => {
    const path = event.composedPath?.() || [];
    for (const item of path) {
      if (!(item instanceof Element)) continue;
      const nodeElement = item.closest?.("[data-node-id][data-node-type]");
      if (!nodeElement) continue;
      const nodeType = nodeElement.getAttribute("data-node-type");
      if (nodeType !== "ElCol") continue;
      const nodeId = nodeElement.getAttribute("data-node-id");
      const colNode = nodeId ? doc.value?.getNode?.(nodeId) : null;
      if (!colNode) continue;
      const parentNode = doc.value?.getParent?.(colNode.id);
      if (parentNode?.type !== "ElLayoutRow") continue;
      const rowSelector = `[data-node-id="${parentNode.id}"]`;
      const rowElement =
        document.querySelector(rowSelector) ||
        nodeElement.closest?.(rowSelector);
      const colRect = nodeElement.getBoundingClientRect?.();
      const edgeThreshold = colInsertEdgeThreshold;
      const nearEdge = colRect
        ? event.clientX - colRect.left <= edgeThreshold ||
          colRect.right - event.clientX <= edgeThreshold
        : false;
      let index = null;
      if (colRect && nearEdge) {
        const colIds = (parentNode.children || []).filter((childId) => {
          const childNode = doc.value?.getNode?.(childId);
          return childNode?.type === "ElCol";
        });
        const currentIndex = Math.max(0, colIds.indexOf(colNode.id));
        const nearLeft = event.clientX - colRect.left <= edgeThreshold;
        index = nearLeft ? currentIndex : currentIndex + 1;
      }
      return { rowNode: parentNode, rowElement, nearEdge, index };
    }
    return null;
  };
  const resolveLayoutInsertTarget = () => {
    const path = event.composedPath?.() || [];
    for (const item of path) {
      if (!(item instanceof Element)) continue;
      const nodeElement = item.closest?.("[data-node-id][data-node-type]");
      if (!nodeElement) continue;
      const nodeType = nodeElement.getAttribute("data-node-type");
      if (nodeType !== "ElLayoutRow") continue;
      const nodeId = nodeElement.getAttribute("data-node-id");
      const rowNode = nodeId ? doc.value?.getNode?.(nodeId) : null;
      if (!rowNode) continue;
      const parentNode = doc.value?.getParent?.(rowNode.id);
      if (parentNode?.type !== "ElLayout") continue;
      const layoutSelector = `[data-node-id="${parentNode.id}"]`;
      const layoutElement =
        document.querySelector(layoutSelector) ||
        nodeElement.closest?.(layoutSelector);
      const rowRect = nodeElement.getBoundingClientRect?.();
      const edgeThreshold = rowInsertEdgeThreshold;
      const nearEdge = rowRect
        ? event.clientY - rowRect.top <= edgeThreshold ||
          rowRect.bottom - event.clientY <= edgeThreshold
        : false;
      return { layoutNode: parentNode, layoutElement, nearEdge };
    }
    return null;
  };

    const resolveElContainerTarget = () => {
      const currentType = node.value?.type;
      const isElContainerScope =
        currentType === "ElContainer" ||
        currentType === "ElHeader" ||
        currentType === "ElAside" ||
        currentType === "ElMain" ||
        currentType === "ElFooter";
      if (!isElContainerScope) return null;
      const resolveLayoutTarget = (element) => {
        if (!element) return null;
        const nodeElement = element.closest?.("[data-node-id][data-node-type]");
        if (!nodeElement) return null;
        const nodeType = nodeElement.getAttribute("data-node-type");
        if (nodeType !== "ElLayout" && nodeType !== "ElLayoutRow") return null;
        const nodeId = nodeElement.getAttribute("data-node-id");
        const targetNode = nodeId ? doc.value?.getNode?.(nodeId) : null;
        if (!targetNode) return null;
        return { node: targetNode, element: nodeElement };
      };
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
        const layoutTarget = resolveLayoutTarget(element);
        if (layoutTarget) return layoutTarget;
        const nodeElement = element.closest?.("[data-node-id]");
        if (!nodeElement) return null;
        const nodeId = nodeElement.getAttribute("data-node-id");
        const resolved = resolveNodeFromId(nodeId);
        if (!resolved) return null;
        return { node: resolved.node, element: nodeElement };
      };

      const hitList = document.elementsFromPoint(
        event.clientX,
        event.clientY
      );
      for (const item of hitList) {
        if (!(item instanceof Element)) continue;
        const resolved = resolveFromElement(item);
        if (resolved) return resolved;
      }

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
    event.dataTransfer?.getData("application/x-designer-node") ||
    event.dataTransfer?.getData("text/plain");
  const fallbackType = dragState.dragType || "";

  try {
    const parsed = JSON.parse(payload);
    const type = parsed?.type || "";
    if (!type && !fallbackType) return;
    const resolvedType = type || fallbackType;
    if (!node.value) return;
    if (!layoutInsertSnapshot) {
      const resolvedLayoutByPoint = resolveLayoutInsertFromPoint();
      if (resolvedLayoutByPoint && resolvedType !== "ElLayoutRow") {
        const inserted = insertIntoLayoutByRow(
          resolvedLayoutByPoint.layoutNode,
          resolvedLayoutByPoint.index,
          resolvedType
        );
        if (!inserted) {
          notifyInsertFailure();
        }
        endDrag();
        return;
      }
    }
    if (
      layoutInsertSnapshot?.layoutId &&
      resolvedType !== "ElLayoutRow" &&
      doc.value?.getNode?.(layoutInsertSnapshot.layoutId)?.type === "ElLayout"
    ) {
      const layoutNode = doc.value.getNode(layoutInsertSnapshot.layoutId);
      const inserted = insertIntoLayoutByRow(
        layoutNode,
        layoutInsertSnapshot.index,
        resolvedType
      );
      if (!inserted) notifyInsertFailure();
      endDrag();
      return;
    }
    const resolvedTarget = resolveElContainerTarget();
    let targetNode = (resolvedTarget && resolvedTarget.node) || node.value;
    let targetElement =
      (resolvedTarget && resolvedTarget.element) || event.currentTarget;
    if (!isDroppableContainer(targetNode)) {
      const resolvedContainer = resolveDropContainer(event, resolvedType);
      if (resolvedContainer) {
        targetNode = resolvedContainer.node;
        targetElement = resolvedContainer.element || targetElement;
      } else {
        return;
      }
    }
    const rowResolved = resolveRowInsertTarget();
    const layoutResolved = resolveLayoutInsertTarget();
    const layoutResolvedByPoint = resolveLayoutInsertFromPoint();
    if (
      rowResolved &&
      resolvedType !== "ElCol" &&
      rowResolved.nearEdge &&
      rowInsertSnapshot?.rowId === rowResolved.rowNode.id
    ) {
      targetNode = rowResolved.rowNode;
      targetElement = rowResolved.rowElement || targetElement;
    }
    if (
      targetNode?.type === "ElCol" &&
      resolvedType !== "ElCol" &&
      doc.value?.getParent?.(targetNode.id)?.type === "ElLayoutRow" &&
      rowInsertSnapshot?.rowId === doc.value.getParent(targetNode.id)?.id
    ) {
      const parentNode = doc.value.getParent(targetNode.id);
      if (parentNode) {
        targetNode = parentNode;
        const rowSelector = `[data-node-id="${parentNode.id}"]`;
        targetElement =
          document.querySelector(rowSelector) ||
          event.currentTarget.closest?.(rowSelector) ||
          targetElement;
      }
    }
    if (
      targetNode?.type === "ElCol" &&
      resolvedType !== "ElCol" &&
      doc.value?.getParent?.(targetNode.id)?.type === "ElLayoutRow"
    ) {
      const rowNode = doc.value.getParent(targetNode.id);
      const layoutNode = rowNode ? doc.value.getParent?.(rowNode.id) : null;
      if (
        layoutInsertSnapshot?.layoutId &&
        layoutNode?.type === "ElLayout" &&
        layoutInsertSnapshot.layoutId === layoutNode.id &&
        resolvedType !== "ElLayoutRow"
      ) {
        const rowNodeInserted = editorStore.insertNode(
          "ElLayoutRow",
          layoutNode.id,
          layoutInsertSnapshot.index
        );
        if (rowNodeInserted) {
          const latestLayout = doc.value?.getNode?.(layoutNode.id);
          const rowCount = (latestLayout?.children || []).filter((childId) => {
            const childNode = doc.value?.getNode?.(childId);
            return childNode?.type === "ElLayoutRow";
          }).length;
          editorStore.updateNode(layoutNode.id, {
            props: {
              ...(latestLayout?.props || layoutNode.props || {}),
              rows: Math.max(1, rowCount),
            },
          });
          editorStore.updateNode(rowNodeInserted.id, {
            props: { ...(rowNodeInserted.props || {}), columns: 1 },
          });
          const latestRow = doc.value?.getNode?.(rowNodeInserted.id);
          const colIds = (latestRow?.children || []).filter((childId) => {
            const childNode = doc.value?.getNode?.(childId);
            return childNode?.type === "ElCol";
          });
          let colId = colIds[0];
          if (!colId) {
            const colNode = editorStore.insertNode(
              "ElCol",
              rowNodeInserted.id,
              0
            );
            if (!colNode) {
              notifyInsertFailure();
              endDrag();
              return;
            }
            colId = colNode.id;
          }
          const inserted = editorStore.insertNode(resolvedType, colId);
          if (!inserted) {
            notifyInsertFailure();
          }
        } else {
          notifyInsertFailure();
        }
        endDrag();
        return;
      }
      const colHasChild = (targetNode.children || []).length > 0;
      if (colHasChild) {
        const rowResolved = resolveRowInsertTarget();
        const colIds = (rowNode?.children || []).filter((childId) => {
          const childNode = doc.value?.getNode?.(childId);
          return childNode?.type === "ElCol";
        });
        const currentIndex = Math.max(0, colIds.indexOf(targetNode.id));
        let insertIndex = currentIndex + 1;
        if (rowInsertSnapshot?.rowId === rowNode.id) {
          insertIndex = rowInsertSnapshot.index;
        } else if (Number.isInteger(rowResolved?.index)) {
          insertIndex = rowResolved.index;
        }
        const colNode = editorStore.insertNode(
          "ElCol",
          rowNode.id,
          insertIndex
        );
        if (colNode) {
          const latestRow = doc.value?.getNode?.(rowNode.id);
          const colCount = (latestRow?.children || []).filter((childId) => {
            const childNode = doc.value?.getNode?.(childId);
            return childNode?.type === "ElCol";
          }).length;
          editorStore.updateNode(rowNode.id, {
            props: { ...(latestRow?.props || rowNode.props || {}), columns: colCount },
          });
          const inserted = editorStore.insertNode(resolvedType, colNode.id);
          if (!inserted) {
            notifyInsertFailure();
          }
        } else {
          notifyInsertFailure();
        }
        endDrag();
        return;
      }
    }
    if (
      layoutResolved &&
      resolvedType !== "ElLayoutRow" &&
      layoutResolved.nearEdge &&
      layoutInsertSnapshot?.layoutId === layoutResolved.layoutNode.id
    ) {
      targetNode = layoutResolved.layoutNode;
      targetElement = layoutResolved.layoutElement || targetElement;
    }
    if (
      layoutResolvedByPoint &&
      resolvedType !== "ElLayoutRow" &&
      !layoutInsertSnapshot
    ) {
      targetNode = layoutResolvedByPoint.layoutNode;
      targetElement = layoutResolvedByPoint.layoutElement || targetElement;
    }
    const allowRowAutoInsert =
      targetNode?.type === "ElLayoutRow" && resolvedType !== "ElCol";
    const allowLayoutAutoInsert =
      targetNode?.type === "ElLayout" && resolvedType !== "ElLayoutRow";
    if (
      !allowRowAutoInsert &&
      !allowLayoutAutoInsert &&
      !canAcceptChild(targetNode, resolvedType)
    )
      return;
    let insertIndex = (targetNode?.children || []).length;
    if (
      rowInsertSnapshot &&
      targetNode?.type === "ElLayoutRow" &&
      rowInsertSnapshot.rowId === targetNode.id
    ) {
      insertIndex = rowInsertSnapshot.index;
    }
    if (
      layoutInsertSnapshot &&
      targetNode?.type === "ElLayout" &&
      layoutInsertSnapshot.layoutId === targetNode.id
    ) {
      insertIndex = layoutInsertSnapshot.index;
    }
    if (
      layoutResolvedByPoint &&
      targetNode?.type === "ElLayout" &&
      layoutResolvedByPoint.layoutNode.id === targetNode.id &&
      layoutInsertSnapshot?.layoutId !== targetNode.id
    ) {
      insertIndex = layoutResolvedByPoint.index;
    }

    const skipFlexInsertForLayout =
      targetNode?.type === "ElLayout" &&
      (layoutInsertSnapshot || layoutResolvedByPoint);
    if (
      targetNode?.type &&
      isFlexDropContainer(targetNode.type) &&
      !skipFlexInsertForLayout
    ) {
      const currentElement = targetElement;
      const direction = resolveFlexDirection(targetNode.type, currentElement);
      const insertInfo = dragDropManager.calculateFlexInsertPosition(
        currentElement,
        event,
        direction
      );
      insertIndex = insertInfo.index;
    }
    let dropPosition = null;
    if (targetNode?.type === "FreeContainer" && targetElement) {
      const rawPosition = resolveDropOffset(event, targetElement);
      const defaultSize = resolveDefaultSize(resolvedType);
      dropPosition = clampDropPosition(rawPosition, targetElement, defaultSize);
    }

    // ElLayout 内拖入组件：自动新增一行并将组件放入该行的列
    if (allowLayoutAutoInsert) {
      const rowNode = editorStore.insertNode(
        "ElLayoutRow",
        targetNode.id,
        insertIndex
      );
      if (rowNode) {
        const latestLayout = doc.value?.getNode?.(targetNode.id);
        const rowCount = (latestLayout?.children || []).filter((childId) => {
          const childNode = doc.value?.getNode?.(childId);
          return childNode?.type === "ElLayoutRow";
        }).length;
        const nextRows = Math.max(1, rowCount);
        if ((latestLayout?.props?.rows || 0) !== nextRows) {
          editorStore.updateNode(targetNode.id, {
            props: { ...(latestLayout?.props || targetNode.props || {}), rows: nextRows },
          });
        }
        editorStore.updateNode(rowNode.id, {
          props: { ...(rowNode.props || {}), columns: 1 },
        });
        const latestRow = doc.value?.getNode?.(rowNode.id);
        const colIds = (latestRow?.children || []).filter((childId) => {
          const childNode = doc.value?.getNode?.(childId);
          return childNode?.type === "ElCol";
        });
        let colId = colIds[0];
        if (!colId) {
          const colNode = editorStore.insertNode("ElCol", rowNode.id, 0);
          if (!colNode) {
            notifyInsertFailure();
            endDrag();
            return;
          }
          colId = colNode.id;
        }
        const inserted = editorStore.insertNode(resolvedType, colId);
        if (!inserted) {
          notifyInsertFailure();
        }
      } else {
        notifyInsertFailure();
      }
      endDrag();
      return;
    }

    // ElLayoutRow 内拖入组件：自动新增一列并将组件放入该列
    if (allowRowAutoInsert) {
      const colNode = editorStore.insertNode(
        "ElCol",
        targetNode.id,
        insertIndex
      );
      if (colNode) {
        const latestRow = doc.value?.getNode?.(targetNode.id);
        const colCount = (latestRow?.children || []).filter((childId) => {
          const childNode = doc.value?.getNode?.(childId);
          return childNode?.type === "ElCol";
        }).length;
        const nextColumns = Math.max(1, colCount);
        editorStore.updateNode(targetNode.id, {
          props: { ...(latestRow?.props || targetNode.props || {}), columns: nextColumns },
        });
        const inserted = editorStore.insertNode(resolvedType, colNode.id);
        if (!inserted) {
          notifyInsertFailure();
        }
      } else {
        notifyInsertFailure();
      }
      endDrag();
      return;
    }

    // 插入新节点
    const inserted = editorStore.insertNode(resolvedType, targetNode?.id, insertIndex, {
      dropPosition: dropPosition || undefined,
    });
    if (!inserted) {
      notifyInsertFailure();
    }
    if (inserted && targetNode?.type === "Tabs") {
      const tabKey =
        activeTabName.value ||
        tabsList.value?.[0]?.name ||
        tabsList.value?.[0]?.label ||
        "";
      if (tabKey) {
        editorStore.updateNode(inserted.id, {
          props: { ...(inserted.props || {}), tabKey: String(tabKey) },
        });
      }
    }
    endDrag();
  } catch (err) {
    const type = payload || fallbackType;
    if (!type) return;
    if (!node.value) return;
    if (!layoutInsertSnapshot) {
      const resolvedLayoutByPoint = resolveLayoutInsertFromPoint();
      if (resolvedLayoutByPoint && type !== "ElLayoutRow") {
        const inserted = insertIntoLayoutByRow(
          resolvedLayoutByPoint.layoutNode,
          resolvedLayoutByPoint.index,
          type
        );
        if (!inserted) {
          notifyInsertFailure();
        }
        endDrag();
        return;
      }
    }
    if (
      layoutInsertSnapshot?.layoutId &&
      type !== "ElLayoutRow" &&
      doc.value?.getNode?.(layoutInsertSnapshot.layoutId)?.type === "ElLayout"
    ) {
      const layoutNode = doc.value.getNode(layoutInsertSnapshot.layoutId);
      const inserted = insertIntoLayoutByRow(
        layoutNode,
        layoutInsertSnapshot.index,
        type
      );
      if (!inserted) notifyInsertFailure();
      endDrag();
      return;
    }
    const resolvedTarget = resolveElContainerTarget();
    let targetNode = resolvedTarget?.node || node.value;
    let targetElement = resolvedTarget?.element || event.currentTarget;
    if (!isDroppableContainer(targetNode)) {
      const resolvedContainer = resolveDropContainer(event, type);
      if (resolvedContainer) {
        targetNode = resolvedContainer.node;
        targetElement = resolvedContainer.element || targetElement;
      } else {
        return;
      }
    }
    const rowResolved = resolveRowInsertTarget();
    const layoutResolved = resolveLayoutInsertTarget();
    const layoutResolvedByPoint = resolveLayoutInsertFromPoint();
    if (
      rowResolved &&
      type !== "ElCol" &&
      rowResolved.nearEdge &&
      rowInsertSnapshot?.rowId === rowResolved.rowNode.id
    ) {
      targetNode = rowResolved.rowNode;
      targetElement = rowResolved.rowElement || targetElement;
    }
    if (
      targetNode?.type === "ElCol" &&
      type !== "ElCol" &&
      doc.value?.getParent?.(targetNode.id)?.type === "ElLayoutRow" &&
      rowInsertSnapshot?.rowId === doc.value.getParent(targetNode.id)?.id
    ) {
      const parentNode = doc.value.getParent(targetNode.id);
      if (parentNode) {
        targetNode = parentNode;
        const rowSelector = `[data-node-id="${parentNode.id}"]`;
        targetElement =
          document.querySelector(rowSelector) ||
          event.currentTarget.closest?.(rowSelector) ||
          targetElement;
      }
    }
    if (
      targetNode?.type === "ElCol" &&
      type !== "ElCol" &&
      doc.value?.getParent?.(targetNode.id)?.type === "ElLayoutRow"
    ) {
      const rowNode = doc.value.getParent(targetNode.id);
      const layoutNode = rowNode ? doc.value.getParent?.(rowNode.id) : null;
      if (
        layoutInsertSnapshot?.layoutId &&
        layoutNode?.type === "ElLayout" &&
        layoutInsertSnapshot.layoutId === layoutNode.id &&
        type !== "ElLayoutRow"
      ) {
        const rowNodeInserted = editorStore.insertNode(
          "ElLayoutRow",
          layoutNode.id,
          layoutInsertSnapshot.index
        );
        if (rowNodeInserted) {
          const latestLayout = doc.value?.getNode?.(layoutNode.id);
          const rowCount = (latestLayout?.children || []).filter((childId) => {
            const childNode = doc.value?.getNode?.(childId);
            return childNode?.type === "ElLayoutRow";
          }).length;
          editorStore.updateNode(layoutNode.id, {
            props: {
              ...(latestLayout?.props || layoutNode.props || {}),
              rows: Math.max(1, rowCount),
            },
          });
          editorStore.updateNode(rowNodeInserted.id, {
            props: { ...(rowNodeInserted.props || {}), columns: 1 },
          });
          const latestRow = doc.value?.getNode?.(rowNodeInserted.id);
          const colIds = (latestRow?.children || []).filter((childId) => {
            const childNode = doc.value?.getNode?.(childId);
            return childNode?.type === "ElCol";
          });
          let colId = colIds[0];
          if (!colId) {
            const colNode = editorStore.insertNode(
              "ElCol",
              rowNodeInserted.id,
              0
            );
            if (!colNode) {
              notifyInsertFailure();
              endDrag();
              return;
            }
            colId = colNode.id;
          }
          const inserted = editorStore.insertNode(type, colId);
          if (!inserted) {
            notifyInsertFailure();
          }
        } else {
          notifyInsertFailure();
        }
        endDrag();
        return;
      }
      const colHasChild = (targetNode.children || []).length > 0;
      if (colHasChild) {
        const rowResolved = resolveRowInsertTarget();
        const colIds = (rowNode?.children || []).filter((childId) => {
          const childNode = doc.value?.getNode?.(childId);
          return childNode?.type === "ElCol";
        });
        const currentIndex = Math.max(0, colIds.indexOf(targetNode.id));
        let insertIndex = currentIndex + 1;
        if (rowInsertSnapshot?.rowId === rowNode.id) {
          insertIndex = rowInsertSnapshot.index;
        } else if (Number.isInteger(rowResolved?.index)) {
          insertIndex = rowResolved.index;
        }
        const colNode = editorStore.insertNode("ElCol", rowNode.id, insertIndex);
        if (colNode) {
          const latestRow = doc.value?.getNode?.(rowNode.id);
          const colCount = (latestRow?.children || []).filter((childId) => {
            const childNode = doc.value?.getNode?.(childId);
            return childNode?.type === "ElCol";
          }).length;
          editorStore.updateNode(rowNode.id, {
            props: { ...(latestRow?.props || rowNode.props || {}), columns: colCount },
          });
          const inserted = editorStore.insertNode(type, colNode.id);
          if (!inserted) {
            notifyInsertFailure();
          }
        } else {
          notifyInsertFailure();
        }
        endDrag();
        return;
      }
    }
    if (
      layoutResolved &&
      type !== "ElLayoutRow" &&
      layoutResolved.nearEdge &&
      layoutInsertSnapshot?.layoutId === layoutResolved.layoutNode.id
    ) {
      targetNode = layoutResolved.layoutNode;
      targetElement = layoutResolved.layoutElement || targetElement;
    }
    if (
      layoutResolvedByPoint &&
      type !== "ElLayoutRow" &&
      !layoutInsertSnapshot
    ) {
      targetNode = layoutResolvedByPoint.layoutNode;
      targetElement = layoutResolvedByPoint.layoutElement || targetElement;
    }
    const allowRowAutoInsert =
      targetNode?.type === "ElLayoutRow" && type !== "ElCol";
    const allowLayoutAutoInsert =
      targetNode?.type === "ElLayout" && type !== "ElLayoutRow";
    if (
      !allowRowAutoInsert &&
      !allowLayoutAutoInsert &&
      !canAcceptChild(targetNode, type)
    )
      return;

    // 计算插入位置
    let insertIndex = (targetNode?.children || []).length;
    if (
      rowInsertSnapshot &&
      targetNode?.type === "ElLayoutRow" &&
      rowInsertSnapshot.rowId === targetNode.id
    ) {
      insertIndex = rowInsertSnapshot.index;
    }
    if (
      layoutInsertSnapshot &&
      targetNode?.type === "ElLayout" &&
      layoutInsertSnapshot.layoutId === targetNode.id
    ) {
      insertIndex = layoutInsertSnapshot.index;
    }
    if (
      layoutResolvedByPoint &&
      targetNode?.type === "ElLayout" &&
      layoutResolvedByPoint.layoutNode.id === targetNode.id &&
      layoutInsertSnapshot?.layoutId !== targetNode.id
    ) {
      insertIndex = layoutResolvedByPoint.index;
    }

    const skipFlexInsertForLayout =
      targetNode?.type === "ElLayout" &&
      (layoutInsertSnapshot || layoutResolvedByPoint);
    if (
      targetNode?.type &&
      isFlexDropContainer(targetNode.type) &&
      !skipFlexInsertForLayout
    ) {
      const currentElement = targetElement;
      const direction = resolveFlexDirection(targetNode.type, currentElement);
      const insertInfo = dragDropManager.calculateFlexInsertPosition(
        currentElement,
        event,
        direction
      );
      insertIndex = insertInfo.index;
    }

    let dropPosition = null;
    if (targetNode?.type === "FreeContainer" && targetElement) {
      const rawPosition = resolveDropOffset(event, targetElement);
      const defaultSize = resolveDefaultSize(type);
      dropPosition = clampDropPosition(rawPosition, targetElement, defaultSize);
    }

    // ElLayout 内拖入组件：自动新增一行并将组件放入该行的列
    if (allowLayoutAutoInsert) {
      const rowNode = editorStore.insertNode(
        "ElLayoutRow",
        targetNode.id,
        insertIndex
      );
      if (rowNode) {
        const latestLayout = doc.value?.getNode?.(targetNode.id);
        const rowCount = (latestLayout?.children || []).filter((childId) => {
          const childNode = doc.value?.getNode?.(childId);
          return childNode?.type === "ElLayoutRow";
        }).length;
        const nextRows = Math.max(1, rowCount);
        if ((latestLayout?.props?.rows || 0) !== nextRows) {
          editorStore.updateNode(targetNode.id, {
            props: { ...(latestLayout?.props || targetNode.props || {}), rows: nextRows },
          });
        }
        editorStore.updateNode(rowNode.id, {
          props: { ...(rowNode.props || {}), columns: 1 },
        });
        const latestRow = doc.value?.getNode?.(rowNode.id);
        const colIds = (latestRow?.children || []).filter((childId) => {
          const childNode = doc.value?.getNode?.(childId);
          return childNode?.type === "ElCol";
        });
        let colId = colIds[0];
        if (!colId) {
          const colNode = editorStore.insertNode("ElCol", rowNode.id, 0);
          if (!colNode) {
            notifyInsertFailure();
            endDrag();
            return;
          }
          colId = colNode.id;
        }
        const inserted = editorStore.insertNode(type, colId);
        if (!inserted) {
          notifyInsertFailure();
        }
      } else {
        notifyInsertFailure();
      }
      endDrag();
      return;
    }

    // ElLayoutRow 内拖入组件：自动新增一列并将组件放入该列
    if (allowRowAutoInsert) {
      const colNode = editorStore.insertNode("ElCol", targetNode.id, insertIndex);
      if (colNode) {
        const latestRow = doc.value?.getNode?.(targetNode.id);
        const colCount = (latestRow?.children || []).filter((childId) => {
          const childNode = doc.value?.getNode?.(childId);
          return childNode?.type === "ElCol";
        }).length;
        const nextColumns = Math.max(1, colCount);
        editorStore.updateNode(targetNode.id, {
          props: { ...(latestRow?.props || targetNode.props || {}), columns: nextColumns },
        });
        const inserted = editorStore.insertNode(type, colNode.id);
        if (!inserted) {
          notifyInsertFailure();
        }
      } else {
        notifyInsertFailure();
      }
      endDrag();
      return;
    }

    // 插入新节点
    const inserted = editorStore.insertNode(type, targetNode?.id, insertIndex, {
      dropPosition: dropPosition || undefined,
    });
    if (!inserted) {
      notifyInsertFailure();
    }
    if (inserted && targetNode?.type === "Tabs") {
      const tabKey =
        activeTabName.value ||
        tabsList.value?.[0]?.name ||
        tabsList.value?.[0]?.label ||
        "";
      if (tabKey) {
        editorStore.updateNode(inserted.id, {
          props: { ...(inserted.props || {}), tabKey: String(tabKey) },
        });
      }
    }
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
  const parentNode = doc.value?.getParent?.(currentNode.id);
  if (
    parentNode?.type === "ElHeader" ||
    parentNode?.type === "ElAside" ||
    parentNode?.type === "ElMain" ||
    parentNode?.type === "ElFooter"
  ) {
    return {
      position: "relative",
      width: "100%",
      height: "100%",
      flexGrow: 1,
      flexShrink: 1,
      alignSelf: "stretch",
      justifySelf: "stretch",
    };
  }

  if (isRootCanvasContainer(currentNode)) {
    const zIndex =
      currentNode.absolutePos?.z ?? currentNode.layoutItem?.free?.abs?.z;
    return {
      position: "absolute",
      left: 0,
      top: 0,
      width: "100%",
      height: "100%",
      boxSizing: "border-box",
      ...(zIndex !== undefined ? { zIndex } : {}),
    };
  }

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
    if (parentNode?.type === "ElCol") {
      style.width = "100%";
      style.alignSelf = "stretch";
    }
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
  } else if (currentNode.type === "Tabs") {
    style.display = "flex";
    style.flexDirection = "column";
    style.position = "relative";
    style.overflow = "hidden";
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
  } else if (currentNode.type === "ElLayoutRow") {
    const gutter = Math.max(0, Number(currentNode.props?.gutter) || 0);
    const halfGutter = gutter ? gutter / 2 : 0;
    const parentNode = doc.value?.getParent?.(currentNode.id);
    const rawHeight = currentNode.style?.height;
    const normalizedHeight =
      rawHeight === undefined || rawHeight === null
        ? ""
        : String(rawHeight).trim();
    const hasFixedHeight =
      normalizedHeight !== "" && normalizedHeight !== "auto";
    style.display = "flex";
    style.flexWrap = "wrap";
    style.alignItems = "stretch";
    if (parentNode?.type === "ElLayout") {
      if (hasFixedHeight) {
        style.flexGrow = 0;
        style.flexShrink = 0;
      } else {
        style.flexGrow = 1;
        style.flexShrink = 1;
        style.flexBasis = "0%";
      }
    }
    if (halfGutter) {
      style.marginLeft = `${-halfGutter}px`;
      style.marginRight = `${-halfGutter}px`;
    }
    style.width = "100%";
    style.position = "relative";
    style.boxSizing = "border-box";
  } else if (currentNode.type === "ElCol") {
    const parentNode = doc.value?.getParent?.(currentNode.id);
    const gutter = Math.max(0, Number(parentNode?.props?.gutter) || 0);
    const halfGutter = gutter ? gutter / 2 : 0;
    if (halfGutter) {
      style.paddingLeft = `${halfGutter}px`;
      style.paddingRight = `${halfGutter}px`;
    }
    style.display = "flex";
    style.flexDirection = "column";
    style.alignItems = "stretch";
    style.width = "100%";
    style.position = "relative";
    style.boxSizing = "border-box";
  } else if (currentNode.type === "ElLayout") {
    const rowGap = Math.max(0, Number(currentNode.props?.gutter) || 0);
    const rawPadding = currentNode.props?.padding;
    const layoutPadding =
      typeof rawPadding === "number" && Number.isFinite(rawPadding)
        ? `${rawPadding}px`
        : rawPadding !== undefined
          ? String(rawPadding)
          : "0px";
    style.display = "flex";
    style.flexDirection = "column";
    style.alignItems = "stretch";
    style.rowGap = `${rowGap}px`;
    style.padding = layoutPadding;
    style.width = "100%";
    style.height = "100%";
    style.overflow = "hidden";
    style.position = "relative";
  } else if (isContainer && !baseStyle.position) {
    style.position = "relative";
  }

  if (isContainer && currentNode.positioning === "absolute") {
    if (style.width === undefined) {
      style.width = "100%";
    }
    if (style.height === undefined) {
      style.height = "100%";
    }
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
  if (typeof value === "string" && needsPxUnit(key)) {
    const trimmed = value.trim();
    if (trimmed && /^-?\d+(\.\d+)?$/.test(trimmed)) {
      return `${trimmed}px`;
    }
  }
  if (typeof value === "number" && needsPxUnit(key)) {
    return `${value}px`;
  }
  return value;
};

/**
 * 解析文本组件的样式属性
 * @param {Record<string, any>} props - 文本属性
 * @returns {Record<string, any>}
 */
const resolveTextPropStyle = (props) => {
  if (!props || typeof props !== "object") return {};
  const style = {};
  if (props.fontSize !== undefined) style.fontSize = props.fontSize;
  if (props.fontWeight !== undefined) style.fontWeight = props.fontWeight;
  if (props.fontFamily) style.fontFamily = props.fontFamily;
  if (props.color) style.color = props.color;
  if (props.textAlign) style.textAlign = props.textAlign;
  if (props.textAlign === "justify") {
    style.textAlignLast = "justify";
  }
  if (props.lineHeight !== undefined) style.lineHeight = props.lineHeight;
  if (props.letterSpacing !== undefined) style.letterSpacing = props.letterSpacing;
  if (props.wordBreak) style.wordBreak = props.wordBreak;

  const lineClamp = Number(props.lineClamp);
  if (Number.isFinite(lineClamp) && lineClamp > 0) {
    style.display = "-webkit-box";
    style.overflow = "hidden";
    style.WebkitLineClamp = lineClamp;
    style.WebkitBoxOrient = "vertical";
  } else if (props.truncate) {
    style.whiteSpace = "nowrap";
    style.overflow = "hidden";
    style.textOverflow = "ellipsis";
  }

  return style;
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
    "letterSpacing",
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
  if (target.closest("[contenteditable='true']")) return true;
  if (target.closest("input,textarea,select,button")) return true;
  if (
    target.closest(
      ".el-input,.el-input__inner,.el-textarea__inner,.el-select,.el-select__input,.el-select__wrapper,.el-radio,.el-checkbox,.el-switch,.el-slider,.el-cascader"
    )
  ) {
    return true;
  }
  return false;
};

/**
 * 解析节点自由布局信息
 * @param {import('@/editor-core').ComponentNode} currentNode - 当前节点
 * @returns {{ x: number, y: number, w: number, h: number, z: number }}
 * @throws {Error} 无
 */
const resolveAbsoluteLayout = (currentNode) => {
  const fallbackAbs = currentNode.layoutItem?.free?.abs || {};
  const useAbsolute =
    currentNode.positioning === "absolute" ||
    currentNode.layoutItem?.free?.mode === "abs";
  const absolutePos = useAbsolute ? currentNode.absolutePos || fallbackAbs : {};
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
 * 生成流式布局下的尺寸样式（清理绝对布局尺寸）
 * @param {Record<string, any> | undefined} currentStyle - 当前样式
 * @returns {Record<string, any>}
 * @throws {Error} 无
 */
const buildFlowResetStyle = (currentStyle) => {
  const nextStyle = { ...(currentStyle || {}) };
  delete nextStyle.width;
  delete nextStyle.height;
  return nextStyle;
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
 * 计算 Layout 布局最小高度（当前不强制最小高度）
 * @param {import('@/editor-core').ComponentNode | null} layoutNode - Layout 节点
 * @returns {number}
 * @throws {Error} 无
 */
const resolveElLayoutMinHeight = (layoutNode) => {
  if (!layoutNode || layoutNode.type !== "ElLayout") return 0;
  return 0;
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

/**
 * 根据容器高度缩放 Header/Footer 尺寸
 * @param {import('@/editor-core').ComponentNode} containerNode - 容器节点
 * @param {number} baseWidth - 原始宽度
 * @param {number} nextWidth - 目标宽度
 * @param {number} baseHeight - 原始高度
 * @param {number} nextHeight - 目标高度
 * @param {{ headerHeight?: number, footerHeight?: number, asideWidth?: number }} baseSectionSizes - 基础区域尺寸
 * @returns {Record<string, string> | null}
 */
const buildContainerSectionSizePatch = (
  containerNode,
  baseWidth,
  nextWidth,
  baseHeight,
  nextHeight,
  baseSectionSizes
) => {
  if (!containerNode || containerNode.type !== "ElContainer") return null;
  const scaleX =
    Number.isFinite(baseWidth) && baseWidth > 0 && Number.isFinite(nextWidth)
      ? nextWidth / baseWidth
      : 1;
  const scaleY =
    Number.isFinite(baseHeight) && baseHeight > 0 && Number.isFinite(nextHeight)
      ? nextHeight / baseHeight
      : 1;
  const hasScaleX = Number.isFinite(scaleX) && Math.abs(scaleX - 1) >= 0.001;
  const hasScaleY = Number.isFinite(scaleY) && Math.abs(scaleY - 1) >= 0.001;
  if (!hasScaleX && !hasScaleY) return null;
  const props = containerNode.props || {};
  const minSectionSize = 40;
  const patch = {};

  if (hasScaleY && props.showHeader !== false) {
    const headerHeight = baseSectionSizes?.headerHeight;
    if (Number.isFinite(headerHeight)) {
      patch.headerHeight = `${Math.max(
        minSectionSize,
        Math.round(headerHeight * scaleY)
      )}px`;
    }
  }
  if (hasScaleY && props.showFooter !== false) {
    const footerHeight = baseSectionSizes?.footerHeight;
    if (Number.isFinite(footerHeight)) {
      patch.footerHeight = `${Math.max(
        minSectionSize,
        Math.round(footerHeight * scaleY)
      )}px`;
    }
  }
  if (hasScaleX && props.showAside !== false) {
    const asideWidth = baseSectionSizes?.asideWidth;
    if (Number.isFinite(asideWidth)) {
      patch.asideWidth = `${Math.max(
        minSectionSize,
        Math.round(asideWidth * scaleX)
      )}px`;
    }
  }

  return Object.keys(patch).length > 0 ? patch : null;
};

/**
 * 按容器尺寸限制区域大小，避免超出容器
 * @param {import('@/editor-core').ComponentNode} containerNode - 容器节点
 * @param {number} width - 容器宽度
 * @param {number} height - 容器高度
 * @returns {Record<string, string> | null}
 */
const clampElContainerPropsBySize = (containerNode, width, height) => {
  if (!containerNode || containerNode.type !== "ElContainer") return null;
  if (!Number.isFinite(width) || !Number.isFinite(height)) return null;
  if (width <= 0 || height <= 0) return null;
  const props = containerNode.props || {};
  const hasHeader = props.showHeader !== false;
  const hasFooter = props.showFooter !== false;
  const hasAside = props.showAside !== false;
  const hasMain = props.showMain !== false;
  const hasBody = hasAside || hasMain;
  const minBodySize = 40;
  const minSectionSize = 40;
  let headerHeight = parseSizeToNumber(props.headerHeight) ?? 60;
  let footerHeight = parseSizeToNumber(props.footerHeight) ?? 60;
  let asideWidth = parseSizeToNumber(props.asideWidth) ?? 200;
  const patch = {};

  if (hasAside) {
    const maxAside = Math.max(0, width - (hasMain ? minBodySize : 0));
    if (Number.isFinite(maxAside)) {
      asideWidth = Math.min(asideWidth, maxAside);
      asideWidth = Math.max(minSectionSize, asideWidth);
      const nextAside = `${Math.round(asideWidth)}px`;
      if (nextAside !== props.asideWidth) {
        patch.asideWidth = nextAside;
      }
    }
  }

  if (hasHeader || hasFooter) {
    const available = Math.max(0, height - (hasBody ? minBodySize : 0));
    let nextHeader = hasHeader ? headerHeight : 0;
    let nextFooter = hasFooter ? footerHeight : 0;
    const total = nextHeader + nextFooter;
    if (total > available && total > 0) {
      const scale = available / total;
      nextHeader = Math.max(
        minSectionSize,
        Math.round(nextHeader * scale)
      );
      nextFooter = Math.max(
        minSectionSize,
        Math.round(nextFooter * scale)
      );
    } else {
      if (hasHeader) nextHeader = Math.min(nextHeader, available);
      if (hasFooter) nextFooter = Math.min(nextFooter, available);
    }
    if (hasHeader) {
      const nextHeaderText = `${nextHeader}px`;
      if (nextHeaderText !== props.headerHeight) {
        patch.headerHeight = nextHeaderText;
      }
    }
    if (hasFooter) {
      const nextFooterText = `${nextFooter}px`;
      if (nextFooterText !== props.footerHeight) {
        patch.footerHeight = nextFooterText;
      }
    }
  }

  return Object.keys(patch).length > 0 ? patch : null;
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
  if (
    !node.value ||
    isChildInElCol.value ||
    (!isMovable.value &&
      !isElColInRow.value &&
      node.value.type !== "ElLayoutRow")
  )
    return;
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

  if (isElColInRow.value) {
    if (handle.x === 0) return;
    const parentNode = doc.value?.getParent?.(node.value.id);
    if (!parentNode) return;
    const rowChildren = Array.isArray(parentNode.children)
      ? parentNode.children
      : [];
    const colIds = rowChildren.filter((childId) => {
      const childNode = doc.value?.getNode?.(childId);
      return childNode?.type === "ElCol";
    });
    const currentIndex = colIds.indexOf(node.value.id);
    const leftColId = currentIndex > 0 ? colIds[currentIndex - 1] : "";
    const leftColNode = leftColId ? doc.value?.getNode?.(leftColId) : null;
    const baseLeftSpan = Math.max(
      1,
      Math.min(24, Number(leftColNode?.props?.span) || 1)
    );
    const baseSpan = Math.max(
      1,
      Math.min(24, Number(node.value.props?.span) || 1)
    );
    const totalSpan = baseLeftSpan + baseSpan;
    const rowSelector = `[data-node-id="${parentNode.id}"]`;
    const rowEl =
      document.querySelector(rowSelector) ||
      nodeRef.value?.closest?.(rowSelector);
    const rowRect = rowEl?.getBoundingClientRect?.();
    if (!rowRect || rowRect.width <= 0) return;
    const zoomValue = Number(canvasZoom?.value) || 1;
    const rowWidth = rowRect.width / zoomValue;
    const startClientX = event.clientX;
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
        // 忽略捕获失败
      }
    }

    const move = (moveEvent) => {
      if (!node.value) return;
      const deltaX = (moveEvent.clientX - startClientX) / zoomValue;
      const rawDelta = (deltaX / rowWidth) * 24;
      let deltaSpan =
        rawDelta > 0 ? Math.floor(rawDelta) : Math.ceil(rawDelta);
      if (handle.x === -1) {
        deltaSpan = -deltaSpan;
      }
      if (handle.x === -1 && leftColNode) {
        let nextSpan = Math.max(1, Math.min(totalSpan - 1, baseSpan + deltaSpan));
        let nextLeft = Math.max(1, totalSpan - nextSpan);
        if (nextSpan === baseSpan && nextLeft === baseLeftSpan) return;
        if (history.value?.isInTransaction?.()) {
          history.value.executeInTransaction(
            new UpdateNodeCommand(leftColNode.id, {
              props: { ...(leftColNode.props || {}), span: nextLeft },
            })
          );
          history.value.executeInTransaction(
            new UpdateNodeCommand(node.value.id, {
              props: { ...(node.value.props || {}), span: nextSpan },
            })
          );
        } else if (history.value?.execute) {
          history.value.execute(
            new UpdateNodeCommand(leftColNode.id, {
              props: { ...(leftColNode.props || {}), span: nextLeft },
            })
          );
          history.value.execute(
            new UpdateNodeCommand(node.value.id, {
              props: { ...(node.value.props || {}), span: nextSpan },
            })
          );
        } else if (doc.value?._updateNode) {
          doc.value._updateNode(leftColNode.id, {
            props: { ...(leftColNode.props || {}), span: nextLeft },
          });
          doc.value._updateNode(node.value.id, {
            props: { ...(node.value.props || {}), span: nextSpan },
          });
        }
        return;
      }
      let nextSpan = Math.max(1, Math.min(24, baseSpan + deltaSpan));
      if (nextSpan === Number(node.value.props?.span || baseSpan)) return;
      editorStore.updateNode(node.value.id, {
        props: { ...(node.value.props || {}), span: nextSpan },
      });
    };

    const up = () => {
      if (history.value?.isInTransaction?.()) {
        history.value.commitTransaction("调整栅格");
      }
      cleanupDragHandlers();
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
      document.addEventListener("pointerup", up, { once: true });
      document.addEventListener("pointercancel", up, { once: true });
    } else {
      document.addEventListener("mousemove", move);
      document.addEventListener("mouseup", up, { once: true });
    }
    return;
  }

  if (node.value.type === "ElLayoutRow" && handle.x === 0 && handle.y !== 0) {
    const parentNode = doc.value?.getParent?.(node.value.id);
    if (parentNode?.type === "ElLayout") {
      const rowChildren = Array.isArray(parentNode.children)
        ? parentNode.children
        : [];
      const rowIds = rowChildren.filter((childId) => {
        const childNode = doc.value?.getNode?.(childId);
        return childNode?.type === "ElLayoutRow";
      });
      const currentIndex = rowIds.indexOf(node.value.id);
      const targetIndex = handle.y === -1 ? currentIndex - 1 : currentIndex + 1;
      const targetRowId = targetIndex >= 0 ? rowIds[targetIndex] : "";
      const targetRowNode = targetRowId
        ? doc.value?.getNode?.(targetRowId)
        : null;
      const currentEl = nodeRef.value;
      const targetEl = targetRowId
        ? document.querySelector(`[data-node-id="${targetRowId}"]`)
        : null;
      const zoomValue = Number(canvasZoom?.value) || 1;
      const currentRect = currentEl?.getBoundingClientRect?.();
      const targetRect = targetEl?.getBoundingClientRect?.();
      if (
        targetRowNode &&
        currentRect &&
        targetRect &&
        currentRect.height > 0 &&
        targetRect.height > 0
      ) {
        const baseCurrentHeight = currentRect.height / zoomValue;
        const baseTargetHeight = targetRect.height / zoomValue;
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

        const minSize = 1;
        const move = (moveEvent) => {
          if (!node.value || !targetRowNode) return;
          const deltaY = (moveEvent.clientY - startClientY) / zoomValue;
          let nextCurrentHeight =
            handle.y === -1
              ? baseCurrentHeight - deltaY
              : baseCurrentHeight + deltaY;
          let nextTargetHeight =
            handle.y === -1
              ? baseTargetHeight + deltaY
              : baseTargetHeight - deltaY;

          if (nextTargetHeight < minSize) {
            nextTargetHeight = minSize;
            nextCurrentHeight = Math.max(
              minSize,
              baseCurrentHeight + (baseTargetHeight - nextTargetHeight)
            );
          }
          if (nextCurrentHeight < minSize) {
            nextCurrentHeight = minSize;
            nextTargetHeight = Math.max(
              minSize,
              baseTargetHeight + (baseCurrentHeight - nextCurrentHeight)
            );
          }

          const currentPatch = {
            style: {
              ...(node.value.style || {}),
              height: `${Math.round(nextCurrentHeight)}px`,
            },
          };
          const targetPatch = {
            style: {
              ...(targetRowNode.style || {}),
              height: `${Math.round(nextTargetHeight)}px`,
            },
          };

          if (history.value?.isInTransaction?.()) {
            history.value.executeInTransaction(
              new UpdateNodeCommand(node.value.id, currentPatch)
            );
            history.value.executeInTransaction(
              new UpdateNodeCommand(targetRowNode.id, targetPatch)
            );
          } else if (history.value?.execute) {
            history.value.execute(
              new UpdateNodeCommand(node.value.id, currentPatch)
            );
            history.value.execute(
              new UpdateNodeCommand(targetRowNode.id, targetPatch)
            );
          } else if (doc.value?._updateNode) {
            doc.value._updateNode(node.value.id, currentPatch);
            doc.value._updateNode(targetRowNode.id, targetPatch);
          }
        };

        const up = () => {
          if (history.value?.isInTransaction?.()) {
            history.value.commitTransaction("调整布局行高度");
          }
          cleanupDragHandlers();
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
          document.addEventListener("pointerup", up, { once: true });
          document.addEventListener("pointercancel", up, { once: true });
        } else {
          document.addEventListener("mousemove", move);
          document.addEventListener("mouseup", up, { once: true });
        }
        return;
      }
    }
  }

  const zoomValue = Number(canvasZoom?.value) || 1;
  const baseLayout = resolveAbsoluteLayout(node.value);
  const rect = nodeRef.value?.getBoundingClientRect?.();
  const rectWidth = rect ? rect.width / zoomValue : undefined;
  const rectHeight = rect ? rect.height / zoomValue : undefined;
  const baseWidth = baseLayout.w || rectWidth || 120;
  const baseHeight = baseLayout.h || rectHeight || 40;
  const baseSectionSizes =
    node.value.type === "ElContainer"
      ? {
          headerHeight:
            parseSizeToNumber(node.value.props?.headerHeight) ?? 60,
          footerHeight:
            parseSizeToNumber(node.value.props?.footerHeight) ?? 60,
          asideWidth: parseSizeToNumber(node.value.props?.asideWidth) ?? 200,
        }
      : null;
  const startClientX = event.clientX;
  const startClientY = event.clientY;
  const minSize = ["ElLayout", "ElLayoutRow", "ElCol"].includes(node.value?.type)
    ? 1
    : 40;
  const containerMinSize = resolveElContainerMinSize(node.value);
  const childMinSize = (() => {
    if (
      !nodeRef.value ||
      !node.value?.children?.length ||
      ["ElLayout", "ElLayoutRow", "ElCol"].includes(node.value?.type)
    ) {
      return null;
    }
    const parentRect = nodeRef.value.getBoundingClientRect?.();
    if (!parentRect) return null;
    let minLeft = Number.POSITIVE_INFINITY;
    let minTop = Number.POSITIVE_INFINITY;
    let maxRight = Number.NEGATIVE_INFINITY;
    let maxBottom = Number.NEGATIVE_INFINITY;
    for (const childId of node.value.children) {
      const childEl = nodeRef.value.querySelector?.(
        `[data-node-id="${childId}"]`
      );
      if (!childEl) continue;
      const childRect = childEl.getBoundingClientRect?.();
      if (!childRect) continue;
      minLeft = Math.min(minLeft, childRect.left);
      minTop = Math.min(minTop, childRect.top);
      maxRight = Math.max(maxRight, childRect.right);
      maxBottom = Math.max(maxBottom, childRect.bottom);
    }
    if (
      minLeft === Number.POSITIVE_INFINITY ||
      minTop === Number.POSITIVE_INFINITY
    ) {
      return null;
    }
    return {
      width: Math.max(0, Math.round((maxRight - minLeft) / zoomValue)),
      height: Math.max(0, Math.round((maxBottom - minTop) / zoomValue)),
    };
  })();

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
    if (childMinSize) {
      if (handle.x === -1 && nextWidth < childMinSize.width) {
        nextX = baseLayout.x + (baseWidth - childMinSize.width);
        nextWidth = childMinSize.width;
      } else if (handle.x === 1 && nextWidth < childMinSize.width) {
        nextWidth = childMinSize.width;
      }
      if (handle.y === -1 && nextHeight < childMinSize.height) {
        nextY = baseLayout.y + (baseHeight - childMinSize.height);
        nextHeight = childMinSize.height;
      } else if (handle.y === 1 && nextHeight < childMinSize.height) {
        nextHeight = childMinSize.height;
      }
    }
    if (node.value.type === "ElLayout") {
      const layoutMinHeight = resolveElLayoutMinHeight(node.value);
      if (layoutMinHeight > 0 && nextHeight < layoutMinHeight) {
        if (handle.y === -1) {
          nextY = baseLayout.y + (baseHeight - layoutMinHeight);
        }
        nextHeight = layoutMinHeight;
      }
    }

    nextWidth = Math.round(nextWidth);
    nextHeight = Math.round(nextHeight);
    nextX = Math.max(0, Math.round(nextX));
    nextY = Math.max(0, Math.round(nextY));

    const sectionPatch =
      node.value.type === "ElContainer"
        ? buildContainerSectionSizePatch(
            node.value,
            baseWidth,
            nextWidth,
            baseHeight,
            nextHeight,
            baseSectionSizes
          )
        : null;

    const nextStyle = { ...(node.value.style || {}) };
    if (node.value.type === "ElLayoutRow") {
      nextStyle.height = `${nextHeight}px`;
      if (handle.x !== 0) {
        nextStyle.width = `${nextWidth}px`;
      }
    } else {
      nextStyle.width = `${nextWidth}px`;
      nextStyle.height = `${nextHeight}px`;
    }

    let patch = sectionPatch
      ? {
          style: nextStyle,
          props: { ...(node.value.props || {}), ...sectionPatch },
        }
      : { style: nextStyle };

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
      if (node.value.type !== "ElLayoutRow" || handle.x !== 0) {
        nodeRef.value.style.width = `${nextWidth}px`;
      }
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

/**
 * 处理节点指针按下事件
 * @param {PointerEvent} event - 指针事件
 */
const handlePointerDown = (event) => {
  if (props.readonly) return;
  if (activeDragHandlers) return;
  if (!node.value || !isMovable.value) return;
  if (event.pointerType === "mouse" && event.button !== 0) return;
  if (event.target?.closest?.(".resize-handle")) return;
  if (isInteractiveTarget(event.target)) return;
  const targetNodeEl = event.target?.closest?.("[data-node-id]");
  const targetNodeId = targetNodeEl?.getAttribute?.("data-node-id");
  if (
    node.value.type === "Tabs" &&
    targetNodeId &&
    targetNodeId !== node.value.id
  ) {
    return;
  }
  if (targetNodeId && targetNodeId !== node.value.id) {
    if (event.altKey) return;
    if (!isContainer.value) return;
  }

  event.preventDefault();
  event.stopPropagation();

  if (selection.value) {
    const element = createSelectableElement("node", node.value.id);
    selection.value.select(element);
  }

  const resetFlowStyle = () => {
    if (!node.value) return;
    const parentNode = doc.value?.getParent?.(node.value.id);
    const isFlow =
      parentNode?.type &&
      parentNode.type !== "FreeContainer" &&
      node.value.positioning !== "absolute";
    if (!isFlow) return;
    const patch = { ...(node.value.style || {}) };
    delete patch.width;
    delete patch.height;
    if (Object.keys(patch).length === 0) {
      return;
    }
    if (history.value?.execute) {
      history.value.execute(new UpdateNodeCommand(node.value.id, { style: patch }));
      return;
    }
    if (doc.value?._updateNode) {
      doc.value._updateNode(node.value.id, { style: patch });
    }
  };

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
  const isInsideElContainer = (targetNode) => {
    if (!targetNode || !doc.value) return false;
    if (targetNode.type === "ElContainer") return true;
    let current = doc.value.getParent?.(targetNode.id);
    while (current) {
      if (current.type === "ElContainer") return true;
      current = doc.value.getParent?.(current.id);
    }
    return false;
  };
  /**
   * 获取最近的 ElContainer 祖先
   * @param {string} nodeId - 节点 ID
   * @returns {import('@/editor-core').ComponentNode | null}
   */
  const resolveAncestorContainer = (nodeId) => {
    if (!nodeId || !doc.value) return null;
    let current = doc.value.getParent?.(nodeId);
    while (current) {
      if (current.type === "ElContainer") return current;
      current = doc.value.getParent?.(current.id);
    }
    return null;
  };
  const dragContainer = resolveAncestorContainer(node.value?.id);
  const restrictToContainer =
    Boolean(dragContainer) && node.value?.type !== "ElContainer";

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
    const childType = node.value?.type;
    let containerNode = null;
    /**
     * 判断是否为区域容器类型
     * @param {string} type - 组件类型
     * @returns {boolean}
     */
    const isRegionType = (type) =>
      type === "ElHeader" ||
      type === "ElAside" ||
      type === "ElMain" ||
      type === "ElFooter";
    /**
     * 获取容器的 Main 区域节点
     * @param {import('@/editor-core').ComponentNode} container - 容器节点
     * @returns {import('@/editor-core').ComponentNode | null}
     */
    const resolveContainerMain = (container) => {
      if (!container || container.type !== "ElContainer") return null;
      const mainChildId = (container.children || []).find((childId) => {
        const childNode = doc.value?.getNode?.(childId);
        return childNode?.type === "ElMain";
      });
      return mainChildId ? doc.value?.getNode?.(mainChildId) || null : null;
    };

    for (const hit of hitList) {
      const nodeElement = hit.closest?.("[data-node-id]");
      const nodeId = nodeElement?.getAttribute?.("data-node-id");
      if (!nodeId) continue;
      if (isSelfOrDescendant(nodeId)) continue;
      const targetNode = doc.value?.getNode?.(nodeId);
      if (!targetNode) continue;
      if (restrictToContainer && !isInsideElContainer(targetNode)) {
        continue;
      }
      if (isRegionType(targetNode.type)) {
        if (childType && !canAcceptChild(targetNode, childType)) continue;
        return targetNode;
      }
      if (targetNode.type === "ElContainer") {
        if (!containerNode) {
          containerNode = resolveContainerMain(targetNode) || targetNode;
        }
        continue;
      }
      const manifest = componentRegistry.get(targetNode.type);
      if (manifest?.isContainer) {
        if (childType && !canAcceptChild(targetNode, childType)) continue;
        return targetNode;
      }
    }
    if (containerNode && (!childType || canAcceptChild(containerNode, childType))) {
      return containerNode;
    }
    return null;
  };
  const isSelfOrDescendant = (targetId) => {
    if (!targetId || !node.value?.id || !doc.value) return false;
    if (targetId === node.value.id) return true;
    let current = doc.value.getParent?.(targetId);
    while (current) {
      if (current.id === node.value.id) return true;
      current = doc.value.getParent?.(current.id);
    }
    return false;
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

  const resolveRowInsertFromPoint = (pointEvent) => {
    const hitList = document.elementsFromPoint(
      pointEvent.clientX,
      pointEvent.clientY
    );
    for (const hit of hitList) {
      const nodeElement = hit.closest?.("[data-node-id][data-node-type]");
      if (!nodeElement) continue;
      const nodeType = nodeElement.getAttribute("data-node-type");
      if (nodeType !== "ElCol") continue;
      const nodeId = nodeElement.getAttribute("data-node-id");
      const colNode = nodeId ? doc.value?.getNode?.(nodeId) : null;
      if (!colNode) continue;
      const parentNode = doc.value?.getParent?.(colNode.id);
      if (parentNode?.type !== "ElLayoutRow") continue;
      const colRect = nodeElement.getBoundingClientRect?.();
      const edgeThreshold = colInsertEdgeThreshold;
      const nearEdge = colRect
        ? pointEvent.clientX - colRect.left <= edgeThreshold ||
          colRect.right - pointEvent.clientX <= edgeThreshold
        : false;
      const rowSelector = `[data-node-id="${parentNode.id}"]`;
      const rowElement =
        document.querySelector(rowSelector) ||
        nodeElement.closest?.(rowSelector);
      if (!rowElement) continue;
      return { rowElement, parentNode, nearEdge };
    }
    return null;
  };

  const resolveLayoutInsertFromPoint = (pointEvent) => {
    const hitList = document.elementsFromPoint(
      pointEvent.clientX,
      pointEvent.clientY
    );
    for (const hit of hitList) {
      const layoutElement = hit.closest?.(
        '[data-node-type="ElLayout"][data-node-id]'
      );
      if (!layoutElement) continue;
      const layoutId = layoutElement.getAttribute("data-node-id");
      const layoutNode = layoutId ? doc.value?.getNode?.(layoutId) : null;
      if (!layoutNode || layoutNode.type !== "ElLayout") continue;
      const rowIds = (layoutNode.children || []).filter((childId) => {
        const childNode = doc.value?.getNode?.(childId);
        return childNode?.type === "ElLayoutRow";
      });
      if (rowIds.length === 0) return null;
      const pointY = pointEvent.clientY;
      const threshold = rowInsertEdgeThreshold;
      for (let i = 0; i < rowIds.length; i += 1) {
        const rowId = rowIds[i];
        const rowElement = layoutElement.querySelector(
          `[data-node-id="${rowId}"]`
        );
        if (!rowElement) continue;
        const rect = rowElement.getBoundingClientRect?.();
        if (!rect) continue;
        if (Math.abs(pointY - rect.top) <= threshold) {
          return { layoutNode, layoutElement, index: i, lineY: rect.top };
        }
        if (Math.abs(pointY - rect.bottom) <= threshold) {
          return { layoutNode, layoutElement, index: i + 1, lineY: rect.bottom };
        }
      }
    }
    return null;
  };

  const updateRowInsertLineFromPoint = (pointEvent) => {
    if (!node.value || node.value.type === "ElCol") return false;
    const resolvedRow = resolveRowInsertFromPoint(pointEvent);
    if (!resolvedRow || !resolvedRow.nearEdge) {
      showInsertLine.value = false;
      insertLineStyle.value = null;
      rowInsertInfo.value = null;
      return false;
    }
    const { rowElement, parentNode } = resolvedRow;
    const rowRect = rowElement.getBoundingClientRect?.();
    if (!rowRect) return false;
    const direction = resolveFlexDirection("ElLayoutRow", rowElement);
    const insertInfo = dragDropManager.calculateFlexInsertPosition(
      rowElement,
      pointEvent,
      direction
    );
    const insertLine =
      insertInfo.insertLine || {
        orientation: "vertical",
        offset: rowRect.right - rowRect.left,
      };
    const adjustedLine = {
      ...insertLine,
      offset: Math.max(0, insertLine.offset),
    };
    showInsertLine.value = true;
    insertLineStyle.value = adjustedLine;
    rowInsertInfo.value = {
      rowId: parentNode.id,
      index: insertInfo.index,
      lineBox: {
        left: rowRect.left,
        top: rowRect.top,
        width: rowRect.width,
        height: rowRect.height,
      },
    };
    return true;
  };

  const updateLayoutInsertLineFromPoint = (pointEvent) => {
    if (!node.value || node.value.type === "ElLayoutRow") return false;
    const resolvedLayout = resolveLayoutInsertFromPoint(pointEvent);
    if (!resolvedLayout) {
      showInsertLine.value = false;
      insertLineStyle.value = null;
      layoutInsertInfo.value = null;
      return false;
    }
    const { layoutElement, layoutNode, index, lineY } = resolvedLayout;
    const layoutRect = layoutElement.getBoundingClientRect?.();
    if (!layoutRect) return false;
    showInsertLine.value = true;
    insertLineStyle.value = {
      orientation: "horizontal",
      offset: Math.max(0, lineY - layoutRect.top),
    };
    rowInsertInfo.value = null;
    layoutInsertInfo.value = {
      layoutId: layoutNode.id,
      index,
      lineBox: {
        left: layoutRect.left,
        top: layoutRect.top,
        width: layoutRect.width,
        height: layoutRect.height,
      },
    };
    return true;
  };

  const move = (moveEvent) => {
    if (!node.value) return;
    const deltaX = (moveEvent.clientX - startClientX) / zoomValue;
    const deltaY = (moveEvent.clientY - startClientY) / zoomValue;
    if (Math.abs(deltaX) > 1 || Math.abs(deltaY) > 1) {
      hasMoved = true;
      if (!startedDragFromMove) {
        startDrag(node.value.type);
        startedDragFromMove = true;
        resetFlowStyle();
      }
      const dropRegion = resolveDropRegion(moveEvent);
      if (dropRegion?.id && !isSelfOrDescendant(dropRegion.id)) {
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
      if (!updateLayoutInsertLineFromPoint(moveEvent)) {
        updateRowInsertLineFromPoint(moveEvent);
      }
      if (restrictToContainer && dragContainer) {
        const containerEl = document.querySelector(
          `[data-node-id="${dragContainer.id}"]`
        );
        const containerRect = containerEl?.getBoundingClientRect?.();
        const isInsideContainer = containerRect
          ? moveEvent.clientX >= containerRect.left &&
            moveEvent.clientX <= containerRect.right &&
            moveEvent.clientY >= containerRect.top &&
            moveEvent.clientY <= containerRect.bottom
          : true;
        if (isInsideContainer) {
          return;
        }
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
    if (node.value.type === "ElLayout") {
      const defaultSize = resolveDefaultSize("ElLayout");
      if (defaultSize?.width) {
        nextAbs.w = Math.max(nextAbs.w, defaultSize.width);
      }
    }

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
    const regionClampPatch =
      node.value.type === "ElContainer"
        ? clampElContainerPropsBySize(node.value, nextAbs.w, nextAbs.h)
        : null;
    const patch = {
      positioning: "absolute",
      absolutePos: nextAbs,
      layoutItem: nextLayoutItem,
      ...(regionClampPatch
        ? { props: { ...(node.value.props || {}), ...regionClampPatch } }
        : {}),
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
    const rowInsertSnapshot = rowInsertInfo.value;
    const layoutInsertSnapshot = layoutInsertInfo.value;
    showInsertLine.value = false;
    insertLineStyle.value = null;
    rowInsertInfo.value = null;
    layoutInsertInfo.value = null;
    if (hasMoved && !isRegionNode && upEvent) {
      if (
        layoutInsertSnapshot?.layoutId &&
        node.value?.type !== "ElLayoutRow" &&
        node.value?.type !== "ElCol"
      ) {
        const layoutNode = doc.value?.getNode?.(layoutInsertSnapshot.layoutId);
        if (layoutNode) {
          const rowNode = editorStore.insertNode(
            "ElLayoutRow",
            layoutNode.id,
            layoutInsertSnapshot.index
          );
          if (rowNode) {
            const latestLayout = doc.value?.getNode?.(layoutNode.id);
            const rowCount = (latestLayout?.children || []).filter((childId) => {
              const childNode = doc.value?.getNode?.(childId);
              return childNode?.type === "ElLayoutRow";
            }).length;
            editorStore.updateNode(layoutNode.id, {
              props: {
                ...(latestLayout?.props || layoutNode.props || {}),
                rows: Math.max(1, rowCount),
              },
            });
            editorStore.updateNode(rowNode.id, {
              props: { ...(rowNode.props || {}), columns: 1 },
            });
            const latestRow = doc.value?.getNode?.(rowNode.id);
            const colIds = (latestRow?.children || []).filter((childId) => {
              const childNode = doc.value?.getNode?.(childId);
              return childNode?.type === "ElCol";
            });
            let colId = colIds[0];
            if (!colId) {
              const colNode = editorStore.insertNode("ElCol", rowNode.id, 0);
              if (!colNode) return;
              colId = colNode.id;
            }
            const moveCommand = new MoveNodeCommand(
              node.value.id,
              colId,
              (latestRow?.children || []).length
            );
            const updatePatch = {
              positioning: "flow",
              absolutePos: undefined,
              flowLayout: undefined,
              layoutItem: undefined,
              style: {
                ...(buildFlowResetStyle(node.value.style) || {}),
                width: "100%",
                height: isContainer.value ? "100%" : "auto",
              },
            };
            const updateCommand = new UpdateNodeCommand(
              node.value.id,
              updatePatch
            );
            if (history.value?.isInTransaction?.()) {
              history.value.executeInTransaction(moveCommand);
              history.value.executeInTransaction(updateCommand);
            } else if (history.value?.execute) {
              history.value.execute(moveCommand);
              history.value.execute(updateCommand);
            } else if (doc.value?._moveNode && doc.value?._updateNode) {
              doc.value._moveNode(
                node.value.id,
                colId,
                (latestRow?.children || []).length
              );
              doc.value._updateNode(node.value.id, updatePatch);
            }
          }
        }
        return;
      }
      if (rowInsertSnapshot?.rowId && node.value?.type !== "ElCol") {
        const rowNode = doc.value?.getNode?.(rowInsertSnapshot.rowId);
        if (rowNode) {
          const colNode = editorStore.insertNode(
            "ElCol",
            rowNode.id,
            rowInsertSnapshot.index
          );
          if (colNode) {
            const latestRow = doc.value?.getNode?.(rowNode.id);
            const colCount = (latestRow?.children || []).filter((childId) => {
              const childNode = doc.value?.getNode?.(childId);
              return childNode?.type === "ElCol";
            }).length;
            editorStore.updateNode(rowNode.id, {
              props: { ...(latestRow?.props || rowNode.props || {}), columns: colCount },
            });
            const moveCommand = new MoveNodeCommand(
              node.value.id,
              colNode.id,
              (colNode.children || []).length
            );
            const shouldResetSize =
              node.value.positioning === "absolute" || node.value.layoutItem?.free;
            const updatePatch = {
              positioning: "flow",
              absolutePos: undefined,
              flowLayout: undefined,
              layoutItem: undefined,
            };
            // 放入 ElCol 时默认铺满容器
            updatePatch.style = {
              ...(buildFlowResetStyle(node.value.style) || {}),
              width: "100%",
              height: isContainer.value ? "100%" : "auto",
            };
            if (shouldResetSize) {
              updatePatch.style = {
                ...(buildFlowResetStyle(node.value.style) || {}),
                width: "100%",
                height: isContainer.value ? "100%" : "auto",
              };
            }
            const updateCommand = new UpdateNodeCommand(
              node.value.id,
              updatePatch
            );
            if (history.value?.isInTransaction?.()) {
              history.value.executeInTransaction(moveCommand);
              history.value.executeInTransaction(updateCommand);
            } else if (history.value?.execute) {
              history.value.execute(moveCommand);
              history.value.execute(updateCommand);
            } else if (doc.value?._moveNode && doc.value?._updateNode) {
              doc.value._moveNode(
                node.value.id,
                colNode.id,
                (colNode.children || []).length
              );
              doc.value._updateNode(node.value.id, updatePatch);
            }
          }
        }
        return;
      }
      const dropRegion = resolveDropRegion(upEvent);
      if (
        restrictToContainer &&
        dropRegion &&
        isInsideElContainer(dropRegion)
      ) {
        clearDropTarget();
        return;
      }
      if (
        dropRegion &&
        dropRegion.id &&
        dropRegion.id !== originParent?.id &&
        canAcceptChild(dropRegion, node.value.type) &&
        !isSelfOrDescendant(dropRegion.id)
      ) {
        const insertIndex = (dropRegion.children || []).length;
        const moveCommand = new MoveNodeCommand(
          node.value.id,
          dropRegion.id,
          insertIndex
        );
        const shouldResetSize =
          node.value.positioning === "absolute" || node.value.layoutItem?.free;
        if (dropRegion?.type === "FreeContainer") {
          const containerEl = document.querySelector(
            `[data-node-id="${dropRegion.id}"]`
          );
          const containerRect = containerEl?.getBoundingClientRect?.();
          const nextX = containerRect
            ? (upEvent.clientX - containerRect.left) / zoomValue
            : 0;
          const nextY = containerRect
            ? (upEvent.clientY - containerRect.top) / zoomValue
            : 0;
          const defaultSize = resolveDefaultSize(node.value.type);
          const nextAbs = {
            x: Math.max(0, Math.round(nextX)),
            y: Math.max(0, Math.round(nextY)),
            w: defaultSize.width,
            h: defaultSize.height,
            z: baseLayout.z,
          };
          const nextLayoutItem = {
            ...(node.value.layoutItem || {}),
            free: { mode: "abs", abs: { ...nextAbs } },
          };
          const nextProps = { ...(node.value.props || {}) };
          if ("tabKey" in nextProps) delete nextProps.tabKey;
          const updateCommand = new UpdateNodeCommand(node.value.id, {
            positioning: "absolute",
            absolutePos: nextAbs,
            flowLayout: undefined,
            layoutItem: nextLayoutItem,
            props: nextProps,
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
              positioning: "absolute",
              absolutePos: nextAbs,
              flowLayout: undefined,
              layoutItem: nextLayoutItem,
              props: nextProps,
            });
          }
          return;
        }
        const updatePatch = {
          positioning: "flow",
          absolutePos: undefined,
          flowLayout: undefined,
          layoutItem: undefined,
        };
        if (dropRegion?.type === "ElCol") {
          updatePatch.style = {
            ...(buildFlowResetStyle(node.value.style) || {}),
            width: "100%",
            height: isContainer.value ? "100%" : "auto",
          };
        } else if (dropRegion?.type === "Tabs") {
          updatePatch.style = {
            ...(buildFlowResetStyle(node.value.style) || {}),
            width: "100%",
            height: "100%",
          };
          const tabKey =
            activeTabName.value ||
            tabsList.value?.[0]?.name ||
            tabsList.value?.[0]?.label ||
            "";
          if (tabKey) {
            updatePatch.props = {
              ...(node.value.props || {}),
              tabKey: String(tabKey),
            };
          }
        } else if (shouldResetSize) {
          updatePatch.style = buildFlowResetStyle(node.value.style);
        }
        const updateCommand = new UpdateNodeCommand(node.value.id, updatePatch);
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
      if (restrictToContainer && dragContainer && rootNodeId) {
        const containerEl = document.querySelector(
          `[data-node-id="${dragContainer.id}"]`
        );
        const containerRect = containerEl?.getBoundingClientRect?.();
        const isInsideContainer = containerRect
          ? upEvent.clientX >= containerRect.left &&
            upEvent.clientX <= containerRect.right &&
            upEvent.clientY >= containerRect.top &&
            upEvent.clientY <= containerRect.bottom
          : false;
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
          if (node.value.type === "ElLayout") {
            const defaultSize = resolveDefaultSize("ElLayout");
            if (defaultSize?.width) {
              nextAbs.w = Math.max(nextAbs.w, defaultSize.width);
            }
          }
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
    }
    if (
      hasMoved &&
      (isRegionParent || allowRegionMoveOut) &&
      originContainer?.type === "ElContainer" &&
      rootNodeId &&
      upEvent &&
      !restrictToContainer
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
        if (node.value.type === "ElLayout") {
          const defaultSize = resolveDefaultSize("ElLayout");
          if (defaultSize?.width) {
            nextAbs.w = Math.max(nextAbs.w, defaultSize.width);
          }
        }
        if (node.value.type === "ElContainer") {
          const minSize = resolveElContainerMinSize(node.value);
          if (minSize) {
            if (Number.isFinite(minSize.width) && nextAbs.w < minSize.width) {
              nextAbs.w = minSize.width;
            }
            if (Number.isFinite(minSize.height) && nextAbs.h < minSize.height) {
              nextAbs.h = minSize.height;
            }
          }
        }
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
  z-index: 2;
}

.designer-node.is-draggable {
  cursor: move;
}

.designer-node.is-container {
  min-height: 40px;
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
}

.designer-node.el-col {
  min-height: 0;
  outline-offset: 0;
}

.designer-node.el-layout-row.is-selected,
.designer-node.el-layout-row:hover {
  outline: none;
}

.designer-node.el-col.is-selected,
.designer-node.el-col:hover {
  outline: none;
}

.designer-node.el-layout-row.is-selected::after,
.designer-node.el-layout-row:hover::after {
  content: "";
  position: absolute;
  top: 0;
  bottom: 0;
  left: calc(-1 * var(--row-gutter, 0));
  right: calc(-1 * var(--row-gutter, 0));
  border: 2px solid #3b82f6;
  border-radius: 2px;
  pointer-events: none;
  box-sizing: border-box;
}

.designer-node.el-layout-row:hover::after {
  border-width: 1px;
  border-color: rgba(59, 130, 246, 0.4);
}

.designer-node.el-col.is-selected::after,
.designer-node.el-col:hover::after {
  content: "";
  position: absolute;
  top: 0;
  bottom: 0;
  left: calc(-1 * var(--col-gutter-x, 0));
  right: calc(-1 * var(--col-gutter-x, 0));
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
  min-height: 40px;
  color: #9ca3af;
  font-size: 12px;
  pointer-events: none;
  border: 1px dashed #d1d5db;
  border-radius: 4px;
  transition: all 0.2s ease;
}

.designer-node.el-layout .empty-container-hint,
.designer-node.el-layout-row .empty-container-hint,
.designer-node.el-col .empty-container-hint {
  min-height: 0;
}

.designer-node.el-layout-row .empty-container-hint,
.designer-node.el-col .empty-container-hint {
  left: calc(-1 * var(--col-gutter-x, 0));
  right: calc(-1 * var(--col-gutter-x, 0));
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
