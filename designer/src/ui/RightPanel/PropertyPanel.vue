<template>
  <div class="flex flex-col gap-3">
    <!-- 页面设置面板：未选中任何元素 -->
    <PageInspectorPanel v-if="panelState === 'page'" />

    <!-- 多选面板：选中多个元素 -->
    <MultiInspectorPanel
      v-else-if="panelState === 'multi'"
      :elements="selectedElements"
    />

    <!-- 单选面板：选中单个元素 -->
    <div v-else class="element-inspector">
      <!-- 基础信息 -->
      <el-descriptions :column="1" size="small" border>
        <el-descriptions-item label="ID">
          {{ elementId }}
        </el-descriptions-item>
        <el-descriptions-item label="类型">
          {{ elementType }}
        </el-descriptions-item>
        <el-descriptions-item label="名称">
          <el-input
            v-model="elementLabel"
            size="small"
            placeholder="未命名"
            @change="handleLabelChange"
          />
        </el-descriptions-item>
      </el-descriptions>

      <el-divider />

      <!-- 属性表单：根据 Manifest 生成 -->
      <template v-if="manifest && manifest.props.length > 0">
        <el-collapse v-model="activeGroupNames">
          <el-collapse-item
            v-for="group in groupedProps"
            :key="group.name"
            :title="group.name"
            :name="group.name"
          >
            <div class="prop-list">
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
                      <span
                        v-if="getRegionMaxLabel(item.sizeProp)"
                        class="region-size-limit"
                      >
                        {{ getRegionMaxLabel(item.sizeProp) }}
                      </span>
                    </div>
                    <el-switch
                      v-if="!item.sizeProp"
                      class="region-toggle"
                      :model-value="Boolean(getPropValue(item.toggleProp))"
                      size="small"
                      @update:model-value="
                        (val) => handlePropChange(item.toggleProp, Boolean(val))
                      "
                    />
                  </div>
                  <div v-if="item.sizeProp" class="region-prop-controls">
                    <el-input
                      class="region-size-input"
                      :model-value="getSizeValue(item.sizeProp)"
                      size="small"
                      placeholder="auto"
                      :disabled="!isRegionEnabled(item)"
                      @update:model-value="
                        (val) => handleSizeValueChange(item.sizeProp, val)
                      "
                    />
                    <el-select
                      :model-value="getSizeUnit(item.sizeProp)"
                      size="small"
                      class="region-unit-select"
                      :disabled="!isRegionEnabled(item)"
                      @update:model-value="
                        (val) => handleSizeUnitChange(item.sizeProp, val)
                      "
                      @change="(val) => handleSizeUnitChange(item.sizeProp, val)"
                    >
                      <el-option label="px" value="px" />
                      <el-option label="%" value="%" />
                      <el-option label="auto" value="auto" />
                    </el-select>
                    <el-switch
                      class="region-toggle"
                      :model-value="Boolean(getPropValue(item.toggleProp))"
                      size="small"
                      @update:model-value="
                        (val) => handlePropChange(item.toggleProp, Boolean(val))
                      "
                    />
                  </div>
                </div>
              </template>
              <template v-else>
                <div
                  v-for="propDef in group.props"
                  :key="propDef.name"
                  class="prop-item"
                >
                  <div class="prop-label">
                    <span>{{ propDef.label }}</span>
                    <el-tooltip content="绑定数据" placement="top">
                      <el-button
                        size="small"
                        text
                        class="bind-btn"
                        @click="handleBindClick(propDef.name)"
                      >
                        <IconEpLink />
                      </el-button>
                    </el-tooltip>
                  </div>
                  <PropEditor
                    :prop="propDef"
                    :model-value="getPropValue(propDef.name)"
                    @update:model-value="
                      (val) => handlePropChange(propDef.name, val)
                    "
                  />
                </div>
              </template>
            </div>
          </el-collapse-item>
        </el-collapse>
      </template>

      <!-- 无 Manifest 时显示原始 Props -->
      <template v-else>
        <div class="text-xs text-gray-500 mb-2">Props</div>
        <pre
          class="text-xs bg-gray-50 dark:bg-gray-900 p-2 rounded overflow-auto max-h-60"
          >{{ formattedProps }}</pre
        >
      </template>

      <el-divider />

      <template v-if="hasStyleSelection">
        <el-collapse v-model="activeStyleNames">
          <el-collapse-item title="基础样式" name="basic">
            <template v-if="showSizeEditor">
              <SizeEditor
                :model-value="currentStyle"
                :min-width="containerMinSize?.width"
                :min-height="containerMinSize?.height"
                @update:model-value="handleStyleChange"
              />
              <el-divider style="margin: 12px 0" />
            </template>
            <BackgroundEditor
              :model-value="currentStyle"
              @update:model-value="handleStyleChange"
            />
            <el-divider style="margin: 12px 0" />
            <BorderEditor
              :model-value="currentStyle"
              @update:model-value="handleStyleChange"
            />
          </el-collapse-item>

          <el-collapse-item title="布局样式" name="layout">
            <PositionEditor
              :model-value="currentStyle"
              @update:model-value="handleStyleChange"
            />
            <el-divider style="margin: 12px 0" />
            <SpacingEditor
              title="内边距"
              prefix="padding"
              :model-value="currentStyle"
              @update:model-value="handleStyleChange"
            />
            <el-divider style="margin: 12px 0" />
            <SpacingEditor
              title="外边距"
              prefix="margin"
              :model-value="currentStyle"
              @update:model-value="handleStyleChange"
            />
          </el-collapse-item>
        </el-collapse>
      </template>
      <div v-else class="text-sm text-gray-400 text-center py-6">
        请选择组件
      </div>


    </div>
  </div>
</template>

<script setup>
/**
 * 属性面板
 * 根据选中状态显示不同的面板：
 * - 未选中：页面设置
 * - 单选：元素属性
 * - 多选：批量编辑
 */

import { computed, ref, watch } from "vue";
import { storeToRefs } from "pinia";
import PageInspectorPanel from "./PageInspectorPanel.vue";
import MultiInspectorPanel from "./MultiInspectorPanel.vue";
import PropEditor from "./PropEditor.vue";
import SizeEditor from "./StylePanel/SizeEditor.vue";
import SpacingEditor from "./StylePanel/SpacingEditor.vue";
import BackgroundEditor from "./StylePanel/BackgroundEditor.vue";
import BorderEditor from "./StylePanel/BorderEditor.vue";
import PositionEditor from "./StylePanel/PositionEditor.vue";
import { usePanelState } from "./use-panel-state";
import { getManifest } from "@/manifests";
import { useEditorStore } from "@/stores/editor-store";
import { ElMessage } from "element-plus";
import IconEpLink from "~icons/ep/link";

const editorStore = useEditorStore();
const { doc } = storeToRefs(editorStore);

/** 当前展开的分组 */
const activeGroupNames = ref([]);
const activeStyleNames = ref(["basic", "layout"]);

const { panelState, selectedElements, selectedNode, selectedGraphic } =
  usePanelState();

/** 当前选中的元素 */
const currentElement = computed(
  () => selectedNode.value || selectedGraphic.value
);

/** 元素 ID */
const elementId = computed(() => currentElement.value?.id || "-");

/** 元素类型 */
const elementType = computed(() => currentElement.value?.type || "-");

/** 元素名称（可编辑） */
const elementLabel = ref("");
const regionSizeState = ref({});

const regionSizeDefaults = {
  headerHeight: "60px",
  asideWidth: "200px",
  footerHeight: "60px",
};

/**
 * 获取区域尺寸默认值
 * @param {string} propName - 属性名
 * @returns {string | undefined}
 */
const getRegionDefaultValue = (propName) => regionSizeDefaults[propName];

/**
 * 解析尺寸值和单位
 * @param {string | number | undefined} value - 尺寸值
 * @returns {{ value: string, unit: string }}
 */
const parseSize = (value) => {
  if (!value || value === "auto") {
    return { value: "", unit: "auto" };
  }
  const str = String(value);
  const match = str.match(/^([\d.]+)(px|%)?$/);
  if (match) {
    return { value: match[1], unit: match[2] || "px" };
  }
  return { value: "", unit: "auto" };
};

/**
 * 解析尺寸为像素值
 * @param {string | number | undefined} value - 尺寸值
 * @returns {number | undefined}
 */
const parseSizeToNumber = (value) => {
  const parsed = parseSize(value);
  if (!parsed.value || parsed.unit !== "px") return undefined;
  const num = Number.parseFloat(parsed.value);
  return Number.isFinite(num) ? num : undefined;
};

/**
 * 获取区域尺寸提示文本
 * @param {{ key: string }} item - 区域配置
 * @returns {string}
 */
const getRegionSizeText = (item) => {
  if (!item) return "";
  if (item.key === "aside") return "宽度";
  if (item.key === "header" || item.key === "footer") return "高度";
  return "";
};

/**
 * 判断区域是否启用
 * @param {{ toggleProp: string }} item - 区域配置
 * @returns {boolean}
 */
const isRegionEnabled = (item) => {
  if (!item?.toggleProp) return true;
  return Boolean(getPropValue(item.toggleProp));
};

/**
 * 获取容器最大尺寸（用于限制区域尺寸）
 * @returns {{ width?: number, height?: number }}
 */
const resolveContainerMaxSize = () => {
  const el = currentElement.value;
  if (!el || el.type !== "ElContainer") return {};
  const liveNode = doc.value?.getNode?.(el.id) || el;
  const styleWidth = parseSizeToNumber(liveNode.style?.width);
  const styleHeight = parseSizeToNumber(liveNode.style?.height);
  const absWidth =
    liveNode.absolutePos && Number.isFinite(liveNode.absolutePos.w)
      ? liveNode.absolutePos.w
      : undefined;
  const absHeight =
    liveNode.absolutePos && Number.isFinite(liveNode.absolutePos.h)
      ? liveNode.absolutePos.h
      : undefined;
  if (typeof document !== "undefined" && el.id) {
    const containerEl = document.querySelector(`[data-node-id="${el.id}"]`);
    if (containerEl) {
      const rect = containerEl.getBoundingClientRect();
      return {
        width: rect.width || styleWidth || absWidth,
        height: rect.height || styleHeight || absHeight,
      };
    }
  }
  return {
    width: styleWidth ?? absWidth,
    height: styleHeight ?? absHeight,
  };
};

/**
 * 限制区域尺寸不超过容器尺寸
 * @param {string} propName - 属性名
 * @param {string} value - 数值
 * @param {string} unit - 单位
 * @returns {{ value: string, unit: string }}
 */
const resolveRegionMaxValue = (propName) => {
  const maxSize = resolveContainerMaxSize();
  const containerProps = currentElement.value?.props || {};
  const hasHeader = containerProps.showHeader !== false;
  const hasFooter = containerProps.showFooter !== false;
  const hasAside = containerProps.showAside !== false;
  const hasMain = containerProps.showMain !== false;
  const minBodySize = 40;

  const headerHeight =
    parseSizeToNumber(containerProps.headerHeight) ??
    parseSizeToNumber(regionSizeDefaults.headerHeight) ??
    60;
  const footerHeight =
    parseSizeToNumber(containerProps.footerHeight) ??
    parseSizeToNumber(regionSizeDefaults.footerHeight) ??
    60;
  const asideWidth =
    parseSizeToNumber(containerProps.asideWidth) ??
    parseSizeToNumber(regionSizeDefaults.asideWidth) ??
    200;

  if (propName === "asideWidth" && maxSize.width) {
    const bodyMin = hasMain ? minBodySize : 0;
    return Math.max(0, maxSize.width - bodyMin);
  }
  if (propName === "headerHeight" && maxSize.height) {
    const footer = hasFooter ? footerHeight : 0;
    const body = hasAside || hasMain ? minBodySize : 0;
    return Math.max(0, maxSize.height - footer - body);
  }
  if (propName === "footerHeight" && maxSize.height) {
    const header = hasHeader ? headerHeight : 0;
    const body = hasAside || hasMain ? minBodySize : 0;
    return Math.max(0, maxSize.height - header - body);
  }
  return undefined;
};

/**
 * 获取区域最大值提示文本
 * @param {string | null} propName - 属性名
 * @returns {string}
 */
const getRegionMaxLabel = (propName) => {
  if (!propName) return "";
  const maxValue = resolveRegionMaxValue(propName);
  if (!maxValue) return "";
  return `<= ${Math.round(maxValue)}px`;
};

/**
 * 限制区域尺寸不超过容器尺寸
 * @param {string} propName - 属性名
 * @param {string} value - 数值
 * @param {string} unit - 单位
 * @returns {{ value: string, unit: string }}
 */
const clampRegionSize = (propName, value, unit) => {
  const maxSize = resolveContainerMaxSize();
  const maxValue = resolveRegionMaxValue(propName);
  if (unit === "%") {
    const num = Number.parseFloat(value || "0");
    if (!Number.isFinite(num)) return { value, unit };
    if (maxValue !== undefined && (maxSize.width || maxSize.height)) {
      const base =
        propName === "asideWidth" ? maxSize.width : maxSize.height;
      if (base) {
        const maxPercent = Math.max(0, (maxValue / base) * 100);
        return {
          value: String(Math.min(num, Math.min(100, maxPercent))),
          unit,
        };
      }
    }
    return { value: String(Math.min(100, Math.max(0, num))), unit };
  }
  if (unit !== "px") return { value, unit };
  const num = Number.parseFloat(value || "0");
  if (!Number.isFinite(num)) return { value, unit };
  if (maxValue !== undefined) {
    return { value: String(Math.min(num, maxValue)), unit };
  }
  return { value, unit };
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

  const headerHeight =
    parseSizeToNumber(props.headerHeight) ??
    parseSizeToNumber(regionSizeDefaults.headerHeight) ??
    60;
  const footerHeight =
    parseSizeToNumber(props.footerHeight) ??
    parseSizeToNumber(regionSizeDefaults.footerHeight) ??
    60;
  const asideWidth =
    parseSizeToNumber(props.asideWidth) ??
    parseSizeToNumber(regionSizeDefaults.asideWidth) ??
    200;
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
 * 同步区域尺寸状态
 * @param {Record<string, any> | undefined} props - 当前属性
 */
const syncRegionSizeState = (props) => {
  const safeProps = props || {};
  regionSizeState.value = {
    headerHeight: parseSize(
      safeProps.headerHeight ?? getRegionDefaultValue("headerHeight")
    ),
    asideWidth: parseSize(
      safeProps.asideWidth ?? getRegionDefaultValue("asideWidth")
    ),
    footerHeight: parseSize(
      safeProps.footerHeight ?? getRegionDefaultValue("footerHeight")
    ),
  };
};

watch(
  currentElement,
  (el) => {
    elementLabel.value = el?.label || "";
  },
  { immediate: true }
);

watch(
  () => currentElement.value?.id,
  () => {
    syncRegionSizeState(currentElement.value?.props);
  },
  { immediate: true }
);

/**
 * 获取组件 Manifest
 */
const manifest = computed(() => {
  const type = currentElement.value?.type;
  if (!type) return null;
  return getManifest(type);
});

const isElContainer = computed(() => elementType.value === "ElContainer");
const regionPropRows = computed(() => [
  {
    key: "header",
    label: "el-header",
    toggleProp: "showHeader",
    sizeProp: "headerHeight",
  },
  {
    key: "aside",
    label: "el-aside",
    toggleProp: "showAside",
    sizeProp: "asideWidth",
  },
  {
    key: "main",
    label: "el-main",
    toggleProp: "showMain",
    sizeProp: null,
  },
  {
    key: "footer",
    label: "el-footer",
    toggleProp: "showFooter",
    sizeProp: "footerHeight",
  },
]);

/**
 * 判断是否为区域属性分组
 * @param {{ props?: Array<{ name: string }> }} group - 属性分组
 * @returns {boolean}
 */
const isRegionGroup = (group) => {
  return Boolean(group?.props?.some((prop) => prop.name === "showHeader"));
};

/**
 * 按分组整理属性
 */
const groupedProps = computed(() => {
  if (!manifest.value) return [];

  const groups = new Map();
  for (const prop of manifest.value.props) {
    const groupName = prop.group || "基础";
    if (!groups.has(groupName)) {
      groups.set(groupName, { name: groupName, props: [] });
    }
    groups.get(groupName).props.push(prop);
  }
  const result = Array.from(groups.values());

  if (result.length > 0 && activeGroupNames.value.length === 0) {
    activeGroupNames.value = result.map((g) => g.name);
  }

  return result;
});

/**
 * 获取属性值
 * @param {string} propName - 属性名
 */
const getPropValue = (propName) => {
  const el = currentElement.value;
  if (!el?.props) return undefined;
  return el.props[propName];
};

/**
 * 获取区域尺寸临时状态
 * @param {string} propName - 属性名
 * @returns {{ value: string, unit: string } | undefined}
 */
const getRegionSizeState = (propName) => regionSizeState.value[propName];

/**
 * 更新区域尺寸临时状态
 * @param {string} propName - 属性名
 * @param {string} value - 数值
 * @param {string} unit - 单位
 */
const setRegionSizeState = (propName, value, unit) => {
  if (!propName) return;
  regionSizeState.value = {
    ...regionSizeState.value,
    [propName]: { value, unit },
  };
};


/**
 * 获取尺寸数值
 * @param {string} propName - 属性名
 * @returns {string}
 */
const getSizeValue = (propName) => {
  if (!propName) return "";
  const cached = getRegionSizeState(propName);
  if (cached) return cached.value;
  return "";
};

/**
 * 获取尺寸单位
 * @param {string} propName - 属性名
 * @returns {string}
 */
const getSizeUnit = (propName) => {
  if (!propName) return "auto";
  const cached = getRegionSizeState(propName);
  if (cached) return cached.unit;
  return "auto";
};

/**
 * 处理尺寸数值变更
 * @param {string} propName - 属性名
 * @param {string} value - 新值
 */
const handleSizeValueChange = (propName, value) => {
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
};

/**
 * 处理尺寸单位变更
 * @param {string} propName - 属性名
 * @param {string} unit - 单位
 */
const handleSizeUnitChange = (propName, unit) => {
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
};

/**
 * Props 格式化展示（无 Manifest 时使用）
 */
const formattedProps = computed(() => {
  const target = currentElement.value;
  if (!target) return "{}";
  return JSON.stringify(target.props || {}, null, 2);
});

/**
 * 是否展示样式面板
 */
const hasStyleSelection = computed(() => selectedNode.value !== null);

/**
 * 是否展示尺寸编辑器
 */
const showSizeEditor = computed(() => {
  const node = selectedNode.value;
  if (!node) return true;
  const regionTypes = ["ElHeader", "ElAside", "ElMain", "ElFooter"];
  if (!regionTypes.includes(node.type)) return true;
  const parentNode = doc.value?.getParent?.(node.id);
  return parentNode?.type !== "ElContainer";
});

/**
 * 容器最小尺寸
 */
const containerMinSize = computed(() =>
  resolveElContainerMinSize(selectedNode.value)
);

/**
 * 当前样式
 */
const currentStyle = computed(() => {
  if (!selectedNode.value) return {};
  return selectedNode.value.style || {};
});

/**
 * 处理样式变更
 * @param {Object} newStyle - 新样式
 */
const handleStyleChange = (newStyle) => {
  if (!selectedNode.value) return;
  editorStore.updateNode(selectedNode.value.id, { style: newStyle });
};

/**
 * 处理名称变更
 */
const handleLabelChange = () => {
  const el = currentElement.value;
  if (!el) return;

  const nextLabel = (elementLabel.value || "").trim();
  if (!nextLabel) {
    elementLabel.value = el.label || "";
    return;
  }
  if (!editorStore.isLabelUnique?.(nextLabel, el.id)) {
    ElMessage.warning("组件名称已存在，请更换");
    elementLabel.value = el.label || "";
    return;
  }

  if (selectedNode.value) {
    editorStore.updateNode(el.id, { label: nextLabel });
  } else if (selectedGraphic.value) {
    editorStore.updateGraphic(el.id, { label: nextLabel });
  }
};

/**
 * 处理属性变更
 * @param {string} propName - 属性名
 * @param {any} value - 新值
 */
const handlePropChange = (propName, value) => {
  const el = currentElement.value;
  if (!el) return;

  const newProps = { ...el.props, [propName]: value };
  if (regionSizeDefaults[propName]) {
    const parsed = parseSize(value);
    setRegionSizeState(propName, parsed.value, parsed.unit);
  }

  if (selectedNode.value) {
    editorStore.updateNode(el.id, { props: newProps });
  } else if (selectedGraphic.value) {
    editorStore.updateGraphic(el.id, { props: newProps });
  }
};

/**
 * 处理绑定按钮点击
 * @param {string} propName - 属性名
 */
const handleBindClick = (propName) => {
  // TODO: 通过事件或 store 通知父组件切换面板
  console.log("绑定属性", propName);
};
</script>

<style scoped>
.element-inspector {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.element-inspector :deep(.el-collapse) {
  border: none;
}

.element-inspector :deep(.el-collapse-item__header) {
  font-size: 13px;
  font-weight: 500;
  background: var(--el-fill-color-lighter);
  padding: 0 12px;
  border-radius: 4px;
}

.element-inspector :deep(.el-collapse-item__wrap) {
  border: none;
}

.element-inspector :deep(.el-collapse-item__content) {
  padding: 12px 8px;
}

.prop-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.prop-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.prop-label {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 12px;
  color: var(--el-text-color-regular);
}

.bind-btn {
  padding: 2px;
  height: auto;
  opacity: 0.5;
  transition: opacity 0.2s;
}

.bind-btn:hover {
  opacity: 1;
}

.region-prop-row .prop-label {
  margin-bottom: 4px;
}

.region-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.region-size-text {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.region-size-limit {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.region-prop-controls {
  display: flex;
  align-items: center;
  gap: 8px;
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
</style>

