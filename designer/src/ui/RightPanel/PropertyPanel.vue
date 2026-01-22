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
                      <span
                        v-if="getRegionSizeText(item)"
                        class="region-size-text"
                      >
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
                      @change="
                        (val) => handleSizeUnitChange(item.sizeProp, val)
                      "
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
                        :class="{ 'is-active': hasPropBinding(propDef.name) }"
                        @click="handleBindClick(propDef)"
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
            <el-divider style="margin: 12px 0" />
            <div class="style-config-list">
              <div class="style-config-row">
                <div class="style-config-label">详细配置</div>
                <el-button
                  size="small"
                  class="config-btn"
                  :class="{ 'is-active': hasDetailConfig }"
                  @click="openConfigDialog('detail')"
                >
                  配置
                </el-button>
              </div>
              <div class="style-config-row">
                <div class="style-config-label">样式配置</div>
                <el-button
                  size="small"
                  class="config-btn"
                  :class="{ 'is-active': hasStyleConfig }"
                  @click="openConfigDialog('style')"
                >
                  配置
                </el-button>
              </div>
            </div>
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

  <el-dialog
    v-model="configDialogVisible"
    :title="configDialogTitle"
    width="980px"
    top="4vh"
    :close-on-click-modal="false"
    :lock-scroll="false"
  >
    <div class="config-toolbar">
      <div class="config-toolbar-item">
        <span class="config-label">样式模板：</span>
        <el-select
          v-model="selectedPresetId"
          size="small"
          class="config-select preset-select"
          placeholder="请选择"
          @change="handlePresetChange"
        >
          <el-option
            v-for="item in filteredPresetOptions"
            :key="item.id"
            :label="item.label"
            :value="item.id"
          />
        </el-select>
      </div>
      <div class="config-toolbar-item">
        <span class="config-label">筛选：</span>
        <el-input
          v-model="presetSearch"
          size="small"
          class="config-select"
          placeholder="搜索模板"
          clearable
        />
      </div>
    </div>
    <div class="config-editor">
      <MonacoEditor
        ref="configEditorRef"
        v-model="configDraft"
        :language="configEditorLanguage"
        height="520px"
        :completions="configEditorCompletions"
      />
    </div>
    <template #footer>
      <el-button @click="clearConfigDialog">清除</el-button>
      <el-button @click="configDialogVisible = false">取消</el-button>
      <el-button type="primary" @click="saveConfigDialog">保存</el-button>
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
        <el-tooltip content="变量枚举" placement="top">
          <el-button
            class="icon-button"
            size="small"
            circle
            @click.stop="openBindingVariableEnum"
          >
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
          <div class="sidebar-title">自定义脚本</div>
          <el-input
            v-model="bindingScriptSearch"
            size="small"
            placeholder="搜索脚本/分组"
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
                  <span
                    class="node-label"
                    :class="{ 'is-group': data.type === 'group' }"
                  >
                    {{ data.label }}
                  </span>
                </div>
              </template>
            </el-tree>
          </div>
        </div>
        <div class="sidebar-section">
          <div class="sidebar-title">页面组件</div>
          <el-input
            v-model="bindingComponentSearch"
            size="small"
            placeholder="搜索组件/分组"
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
                  <span
                    class="node-label"
                    :class="{ 'is-group': data.type === 'group' }"
                  >
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
      <el-button @click="bindingDialogVisible = false">取消</el-button>
      <el-button type="primary" @click="saveBinding">保存</el-button>
    </template>
  </el-dialog>

  <el-dialog
    v-model="bindingVariableEnumVisible"
    title="变量枚举"
    width="760px"
    :z-index="3100"
    append-to-body
    :modal-append-to-body="true"
    :close-on-click-modal="false"
    :lock-scroll="false"
  >
    <el-tabs v-model="bindingEnumTab">
      <el-tab-pane label="工程变量" name="project">
        <div class="enum-layout">
          <div class="enum-left">
            <div class="sidebar-title">分组</div>
            <el-tree
              ref="bindingEnumProjectTreeRef"
              :data="bindingProjectGroupTree"
              node-key="id"
              :default-expand-all="true"
              :expand-on-click-node="false"
              :filter-node-method="filterBindingSidebarNode"
              @node-click="handleBindingProjectGroupSelect"
            >
              <template #default="{ data }">
                <div class="tree-node node-group">
                  <el-icon class="node-icon icon-variable">
                    <IconEpFolder />
                  </el-icon>
                  <span class="node-label is-group">{{ data.label }}</span>
                </div>
              </template>
            </el-tree>
          </div>
          <div class="enum-right">
            <el-input
              v-model="bindingProjectVarSearch"
              size="small"
              placeholder="搜索工程变量"
              clearable
            />
            <el-table
              :data="bindingProjectVariableRows"
              size="small"
              height="320"
              highlight-current-row
              @row-click="handleBindingProjectRowClick"
              @row-dblclick="handleBindingProjectRowDblClick"
              :row-class-name="bindingEnumProjectRowClass"
            >
              <el-table-column prop="name" label="变量名" min-width="160" />
              <el-table-column prop="type" label="类型" width="90" />
              <el-table-column
                prop="description"
                label="描述"
                min-width="160"
              />
              <el-table-column prop="mapped" label="映射" width="70">
                <template #default="{ row }">
                  {{ row.mapped ? "是" : "" }}
                </template>
              </el-table-column>
            </el-table>
          </div>
        </div>
      </el-tab-pane>
      <el-tab-pane label="页面变量" name="page">
        <div class="enum-layout">
          <div class="enum-left">
            <div class="sidebar-title">分组</div>
            <el-tree
              ref="bindingEnumPageTreeRef"
              :data="bindingPageGroupTree"
              node-key="id"
              :default-expand-all="true"
              :expand-on-click-node="false"
              :filter-node-method="filterBindingSidebarNode"
              @node-click="handleBindingPageGroupSelect"
            >
              <template #default="{ data }">
                <div class="tree-node node-group">
                  <el-icon class="node-icon icon-variable">
                    <IconEpFolder />
                  </el-icon>
                  <span class="node-label is-group">{{ data.label }}</span>
                </div>
              </template>
            </el-tree>
          </div>
          <div class="enum-right">
            <el-input
              v-model="bindingPageVarSearch"
              size="small"
              placeholder="搜索页面变量"
              clearable
            />
            <el-table
              :data="bindingPageVariableRows"
              size="small"
              height="320"
              highlight-current-row
              @row-click="handleBindingPageRowClick"
              @row-dblclick="handleBindingPageRowDblClick"
              :row-class-name="bindingEnumPageRowClass"
            >
              <el-table-column prop="name" label="变量名" min-width="160" />
              <el-table-column prop="type" label="类型" width="90" />
              <el-table-column
                prop="description"
                label="描述"
                min-width="200"
              />
            </el-table>
          </div>
        </div>
      </el-tab-pane>
    </el-tabs>
    <template #footer>
      <el-button @click="bindingVariableEnumVisible = false">取消</el-button>
      <el-button
        type="primary"
        :disabled="
          !bindingEnumSelectedProjectVar && !bindingEnumSelectedPageVar
        "
        @click="confirmBindingEnumInsert"
      >
        插入
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup>
/**
 * ????
 * ?????????????
 * - ????????
 * - ???????
 * - ???????
 */

import { computed, ref, watch, nextTick } from "vue";
import { storeToRefs } from "pinia";
import PageInspectorPanel from "./PageInspectorPanel.vue";
import MultiInspectorPanel from "./MultiInspectorPanel.vue";
import PropEditor from "./PropEditor.vue";
import MonacoEditor from "@/components/common/MonacoEditor.vue";
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
import IconEpEditPen from "~icons/ep/edit-pen";
import IconEpFolder from "~icons/ep/folder";
import IconEpList from "~icons/ep/list";
import IconEpGrid from "~icons/ep/grid";

const editorStore = useEditorStore();
const {
  doc,
  docVersion,
  currentPage,
  currentPageId,
  projectVariables,
  projectVariableGroups,
  globalScripts,
} = storeToRefs(editorStore);

/** ??????? */
const activeGroupNames = ref([]);
const activeStyleNames = ref(["basic", "layout"]);

const { panelState, selectedElements, selectedNode, selectedGraphic } =
  usePanelState();

/** ??????? */
const currentElement = computed(
  () => selectedNode.value || selectedGraphic.value,
);

/** ?? ID */
const elementId = computed(() => currentElement.value?.id || "-");

/** ???? */
const elementType = computed(() => currentElement.value?.type || "-");

/** ????????? */
const elementLabel = ref("");
const regionSizeState = ref({});

const regionSizeDefaults = {
  headerHeight: "60px",
  asideWidth: "200px",
  footerHeight: "60px",
};

const configDialogVisible = ref(false);
const configDialogType = ref("style");
const configDraft = ref("");
const configEditorRef = ref(null);
const selectedPresetId = ref("");
const presetSearch = ref("");
const bindingDialogVisible = ref(false);
const bindingEditorCode = ref("");
const bindingEditorRef = ref(null);
const bindingProp = ref(null);
const bindingScriptSearch = ref("");
const bindingComponentSearch = ref("");
const bindingCustomTreeRef = ref(null);
const bindingComponentTreeRef = ref(null);
const bindingVariableEnumVisible = ref(false);
const bindingEnumTab = ref("project");
const bindingProjectVarSearch = ref("");
const bindingPageVarSearch = ref("");
const bindingEnumProjectTreeRef = ref(null);
const bindingEnumPageTreeRef = ref(null);
const bindingEnumSelectedProjectGroupId = ref(null);
const bindingEnumSelectedPageGroupId = ref("page-root");
const bindingEnumSelectedProjectVar = ref(null);
const bindingEnumSelectedPageVar = ref(null);

const stylePresets = [
  { id: "empty", label: "\u7a7a\u6a21\u677f", content: "" },
  {
    id: "center",
    label: "\u5c45\u4e2d\u5e03\u5c40",
    content: "display: flex;\nalign-items: center;\njustify-content: center;",
  },
];
const buttonStylePresets = [
  {
    id: "btn-bg",
    label: "\u6309\u94ae\u80cc\u666f\u8272",
    content: "background: var(--el-color-primary);\ncolor: #ffffff;",
  },
  {
    id: "btn-font",
    label: "\u5b57\u4f53\u989c\u8272\u548c\u5927\u5c0f",
    content: "color: #ffffff;\nfont-size: 14px;\nfont-weight: 500;",
  },
  {
    id: "btn-hover",
    label: "\u9f20\u6807\u79fb\u5165\u80cc\u666f\u8272",
    content:
      "#domId { transition: background 0.2s; }\n#domId:hover { background: var(--el-color-primary-light-3); }",
  },
  {
    id: "btn-border",
    label: "\u8fb9\u6846\u5bbd\u5ea6\u3001\u989c\u8272",
    content: "border: 1px solid var(--el-color-primary);",
  },
  {
    id: "btn-radius",
    label: "\u8fb9\u6846\u53d8\u6210\u76f4\u89d2",
    content: "border-radius: 0;",
  },
  {
    id: "btn-bg-image",
    label: "\u6dfb\u52a0\u80cc\u666f\u56fe\u7247",
    content: "background-image: url('');\nbackground-size: cover;",
  },
];
const detailPresets = [
  {
    id: "empty",
    label: "\u7a7a\u6a21\u677f",
    content:
      "// \u5728\u8fd9\u91cc\u7f16\u5199\u8be6\u7ec6\u914d\u7f6e\u811a\u672c\n",
  },
  {
    id: "comment",
    label: "\u6ce8\u91ca\u6a21\u677f",
    content: "/* id\u4e3a\u7ec4\u4ef6\u7684\u552f\u4e00\u6807\u8bc6 */\n",
  },
];
const buttonDetailPresets = [
  {
    id: "btn-text",
    label: "\u6309\u94ae\u6587\u5b57\u914d\u7f6e",
    content:
      "this.button({\n  id: \"saveBtn\",\n  text: \"\u9ed8\u8ba4\u6309\u94ae\",\n  textExpr: \"state.mode === 'edit' ? '\u4fdd\u5b58' : '\u63d0\u4ea4'\",\n});",
  },
  {
    id: "btn-style",
    label: "\u6309\u94ae\u6837\u5f0f\u914d\u7f6e",
    content:
      'this.button({\n  id: "saveBtn",\n  type: "primary",\n  size: "small",\n  plain: false,\n  round: false,\n  circle: false,\n  icon: "el-icon-check",\n  style: {\n    background: "#409eff",\n    color: "#ffffff",\n  },\n  className: "custom-button",\n});',
  },
  {
    id: "btn-action",
    label: "\u6309\u94ae\u884c\u4e3a\u914d\u7f6e",
    content:
      'this.button({\n  id: "saveBtn",\n  onClick: {\n    action: "save",\n    confirm: "\u786e\u8ba4\u63d0\u4ea4\u5417\uff1f",\n  },\n});',
  },
  {
    id: "btn-state",
    label: "\u6309\u94ae\u7981\u7528 / \u52a0\u8f7d\u72b6\u6001",
    content:
      'this.button({\n  id: "saveBtn",\n  disabled: true,\n  loading: false,\n});',
  },
  {
    id: "btn-visible",
    label: "\u6309\u94ae\u6743\u9650 / \u663e\u9690\u914d\u7f6e",
    content:
      'this.button({\n  id: "saveBtn",\n  permission: "form:save",\n  visible: "state.canSave === true",\n});',
  },
  {
    id: "btn-full-dsl",
    label: "\u5b8c\u6574 DSL \u6a21\u677f",
    content:
      'this.button({\n  /** \u57fa\u7840 */\n  id: "saveBtn",\n  text: "\u6309\u94ae",\n  textExpr: "state.mode === \'edit\' ? \'\u4fdd\u5b58\' : \'\u63d0\u4ea4\'",\n  visible: true,\n  permission: "form:save",\n\n  /** Element Plus Props */\n  type: "primary",\n  size: "small",\n  plain: false,\n  round: false,\n  circle: false,\n  disabled: false,\n  loading: false,\n  icon: "el-icon-check",\n\n  /** \u6837\u5f0f */\n  style: {\n    background: "#409eff",\n    color: "#ffffff",\n  },\n  className: "custom-button",\n\n  /** \u884c\u4e3a */\n  onClick: {\n    action: "save",\n    confirm: "\u786e\u8ba4\u63d0\u4ea4\u5417\uff1f",\n  },\n\n  /** \u6269\u5c55 */\n  plugin: null,\n});',
  },
];

const elementPlusTypes = new Set([
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
const isElementPlusType = (type) => {
  if (!type) return false;
  if (type.startsWith("El")) return true;
  return elementPlusTypes.has(type);
};

/**
 * 获取 DSL 方法名
 * @param {string} type - 组件类型
 * @returns {string}
 */
const getDslMethodName = (type) => {
  if (!type) return "component";
  const normalized = type.replace(/^El/, "el");
  return normalized.charAt(0).toLowerCase() + normalized.slice(1);
};

/**
 * 生成预设 ID
 * @param {string} type - 组件类型
 * @param {string} key - 预设标识
 * @returns {string}
 */
const buildPresetId = (type, key) => {
  return `${String(type || "component").toLowerCase()}-${key}`;
};

/**
 * 生成 DSL 模板内容
 * @param {string} methodName - 方法名
 * @param {string} body - 模板主体
 * @returns {string}
 */
const buildDslTemplate = (methodName, body) => {
  return `this.${methodName}({\n${body}\n});`;
};

/**
 * 提取 DSL 模板主体
 * @param {string} content - 模板内容
 * @returns {string}
 */
const extractDslBody = (content) => {
  const text = String(content || "");
  const start = text.indexOf("{");
  const end = text.lastIndexOf("}");
  if (start === -1 || end === -1 || end <= start) return text.trim();
  return text.slice(start + 1, end).trim();
};

const dslEventKeyMap = {
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
const splitTopLevelEntries = (body) => {
  const entries = [];
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
};

/**
 * 生成标准 DSL 模板
 * @param {string} type - 组件类型
 * @param {{ id: string, label: string, content: string }} preset - 预设项
 * @returns {{ id: string, label: string, content: string }}
 */
const normalizeDslPreset = (type, preset) => {
  const rawContent = preset?.content || "";
  if (rawContent.includes("props:") || rawContent.includes("events:")) {
    return preset;
  }
  const methodName = getDslMethodName(type);
  const body = extractDslBody(rawContent);
  const entries = splitTopLevelEntries(body);
  const dslId = `${methodName}Id`;
  const label = preset?.label || "";
  const props = [];
  const state = [];
  const events = [];
  let styleBody = "";
  let className = "";
  let idValue = "";
  let labelValue = "";
  let typeValue = "";

  const normalizeValue = (value) => value.replace(/,$/, "").trim();

  entries.forEach((entry) => {
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

  const toBlock = (items) =>
    items.length ? items.map((item) => item.replace(/,$/, "")).join(",\n") : "";

  const styleLines = styleBody
    ? styleBody
        .split("\n")
        .map((line) => line.trim())
        .filter(Boolean)
        .map((line) => `  ${line.replace(/,$/, "")}`)
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
    keywords:
      preset?.keywords || [preset?.label, type].filter(Boolean).join(" "),
    content: `this.${methodName}({\n${blocks}\n});`,
  };
};

const elementPlusPresetGroups = [
  {
    name: "input",
    types: ["Input", "InputNumber"],
    detail: (methodName, type) => [
      {
        id: buildPresetId(type, "basic"),
        label: "输入基础配置",
        content: buildDslTemplate(
          methodName,
          '  id: "inputField",\n  value: "",\n  placeholder: "请输入",\n  clearable: true,',
        ),
      },
      {
        id: buildPresetId(type, "status"),
        label: "输入状态配置",
        content: buildDslTemplate(
          methodName,
          "  disabled: false,\n  readonly: false,\n  maxlength: 20,\n  showWordLimit: true,",
        ),
      },
      {
        id: buildPresetId(type, "full-dsl"),
        label: "完整 DSL 模板",
        content: buildDslTemplate(
          methodName,
          '  /** 基础 */\n  id: "inputField",\n  value: "",\n  placeholder: "请输入",\n  clearable: true,\n  size: "small",\n\n  /** 状态 */\n  disabled: false,\n  readonly: false,\n\n  /** 样式 */\n  style: {\n    width: "220px",\n  },\n  className: "custom-input",\n\n  /** 事件 */\n  onChange: {\n    action: "setVar",\n    target: "$vars.inputValue",\n  },',
        ),
      },
    ],
    style: (type) => [
      {
        id: buildPresetId(type, "border"),
        label: "输入边框",
        content: "#domId .el-input__inner { border: 1px solid #409eff; }",
      },
      {
        id: buildPresetId(type, "bg"),
        label: "输入背景",
        content: "#domId .el-input__inner { background: #f5f7fa; }",
      },
      {
        id: buildPresetId(type, "radius"),
        label: "输入圆角",
        content: "#domId .el-input__inner { border-radius: 6px; }",
      },
    ],
  },
  {
    name: "select",
    types: ["Select", "Cascader"],
    detail: (methodName, type) => [
      {
        id: buildPresetId(type, "basic"),
        label: "选择基础配置",
        content: buildDslTemplate(
          methodName,
          '  id: "selectField",\n  value: "",\n  placeholder: "请选择",\n  clearable: true,\n  options: [\n    { label: "选项A", value: "A" },\n    { label: "选项B", value: "B" },\n  ],',
        ),
      },
      {
        id: buildPresetId(type, "multi"),
        label: "多选配置",
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
          '  /** 基础 */\n  id: "selectField",\n  value: "",\n  placeholder: "请选择",\n  clearable: true,\n  options: [\n    { label: "选项A", value: "A" },\n    { label: "选项B", value: "B" },\n  ],\n\n  /** 状态 */\n  disabled: false,\n\n  /** 样式 */\n  style: {\n    width: "220px",\n  },\n  className: "custom-select",\n\n  /** 事件 */\n  onChange: {\n    action: "setVar",\n    target: "$vars.selectValue",\n  },',
        ),
      },
    ],
    style: (type) => [
      {
        id: buildPresetId(type, "border"),
        label: "选择边框",
        content: "#domId .el-input__inner { border: 1px solid #409eff; }",
      },
      {
        id: buildPresetId(type, "bg"),
        label: "选择背景",
        content: "#domId .el-input__inner { background: #f5f7fa; }",
      },
      {
        id: buildPresetId(type, "radius"),
        label: "选择圆角",
        content: "#domId .el-input__inner { border-radius: 6px; }",
      },
    ],
  },
  {
    name: "choice",
    types: ["Switch", "Radio", "Checkbox"],
    detail: (methodName, type) => [
      {
        id: buildPresetId(type, "basic"),
        label: "选择基础配置",
        content: buildDslTemplate(
          methodName,
          '  id: "choiceField",\n  value: "",\n  options: [\n    { label: "选项A", value: "A" },\n    { label: "选项B", value: "B" },\n  ],',
        ),
      },
      {
        id: buildPresetId(type, "status"),
        label: "选择状态配置",
        content: buildDslTemplate(
          methodName,
          '  disabled: false,\n  size: "small",',
        ),
      },
      {
        id: buildPresetId(type, "full-dsl"),
        label: "完整 DSL 模板",
        content: buildDslTemplate(
          methodName,
          '  /** 基础 */\n  id: "choiceField",\n  value: "",\n  options: [\n    { label: "选项A", value: "A" },\n    { label: "选项B", value: "B" },\n  ],\n\n  /** 状态 */\n  disabled: false,\n\n  /** 样式 */\n  className: "custom-choice",\n\n  /** 事件 */\n  onChange: {\n    action: "setVar",\n    target: "$vars.choiceValue",\n  },',
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
        label: "边框颜色",
        content:
          "#domId .el-radio, #domId .el-checkbox { border-color: #dcdfe6; }",
      },
      {
        id: buildPresetId(type, "size"),
        label: "组件间距",
        content:
          "#domId .el-radio, #domId .el-checkbox { margin-right: 12px; }",
      },
    ],
  },
  {
    name: "table",
    types: ["Table", "BigDataTable"],
    detail: (methodName, type) => [
      {
        id: buildPresetId(type, "basic"),
        label: "表格基础配置",
        content: buildDslTemplate(
          methodName,
          '  id: "tableList",\n  data: [],\n  columns: [\n    { prop: "name", label: "名称" },\n    { prop: "value", label: "值" },\n  ],',
        ),
      },
      {
        id: buildPresetId(type, "layout"),
        label: "表格布局配置",
        content: buildDslTemplate(
          methodName,
          "  stripe: true,\n  border: true,\n  height: 360,",
        ),
      },
      {
        id: buildPresetId(type, "full-dsl"),
        label: "完整 DSL 模板",
        content: buildDslTemplate(
          methodName,
          '  /** 基础 */\n  id: "tableList",\n  data: [],\n  columns: [\n    { prop: "name", label: "名称" },\n    { prop: "value", label: "值" },\n  ],\n\n  /** 表格属性 */\n  stripe: true,\n  border: true,\n  rowKey: "id",\n  height: 360,\n\n  /** 样式 */\n  className: "custom-table",\n\n  /** 事件 */\n  onRowClick: {\n    action: "setVar",\n    target: "$vars.currentRow",\n  },',
        ),
      },
    ],
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
        label: "边框颜色",
        content: "#domId .el-table { border-color: #dcdfe6; }",
      },
    ],
  },
  {
    name: "tree",
    types: ["Tree"],
    detail: (methodName, type) => [
      {
        id: buildPresetId(type, "basic"),
        label: "树基础配置",
        content: buildDslTemplate(
          methodName,
          '  id: "treeData",\n  data: [],\n  props: { label: "label", children: "children" },',
        ),
      },
      {
        id: buildPresetId(type, "check"),
        label: "树勾选配置",
        content: buildDslTemplate(
          methodName,
          '  showCheckbox: true,\n  nodeKey: "id",\n  defaultExpandAll: true,',
        ),
      },
      {
        id: buildPresetId(type, "full-dsl"),
        label: "完整 DSL 模板",
        content: buildDslTemplate(
          methodName,
          '  /** 基础 */\n  id: "treeData",\n  data: [],\n  props: { label: "label", children: "children" },\n\n  /** 属性 */\n  showCheckbox: true,\n  nodeKey: "id",\n  defaultExpandAll: true,\n\n  /** 样式 */\n  className: "custom-tree",\n\n  /** 事件 */\n  onNodeClick: {\n    action: "setVar",\n    target: "$vars.activeNode",\n  },',
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
        label: "节点悬停色",
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
      {
        id: buildPresetId(type, "basic"),
        label: "菜单基础配置",
        content: buildDslTemplate(
          methodName,
          '  id: "menuNav",\n  items: [\n    { label: "编辑", command: "edit" },\n    { label: "删除", command: "delete" },\n  ],',
        ),
      },
      {
        id: buildPresetId(type, "trigger"),
        label: "菜单触发配置",
        content: buildDslTemplate(
          methodName,
          '  trigger: "click",\n  splitButton: false,',
        ),
      },
      {
        id: buildPresetId(type, "full-dsl"),
        label: "完整 DSL 模板",
        content: buildDslTemplate(
          methodName,
          '  /** 基础 */\n  id: "menuNav",\n  items: [\n    { label: "编辑", command: "edit" },\n    { label: "删除", command: "delete" },\n  ],\n\n  /** 状态 */\n  disabled: false,\n\n  /** 样式 */\n  className: "custom-menu",\n\n  /** 事件 */\n  onCommand: {\n    action: "setVar",\n    target: "$vars.menuCommand",\n  },',
        ),
      },
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
        label: "菜单悬停色",
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
        label: "标签/折叠基础配置",
        content: buildDslTemplate(
          methodName,
          '  id: "tabsPanel",\n  activeName: "tab1",\n  items: [\n    { name: "tab1", label: "标签一" },\n    { name: "tab2", label: "标签二" },\n  ],',
        ),
      },
      {
        id: buildPresetId(type, "type"),
        label: "样式类型配置",
        content: buildDslTemplate(
          methodName,
          '  type: "card",\n  stretch: true,',
        ),
      },
      {
        id: buildPresetId(type, "full-dsl"),
        label: "完整 DSL 模板",
        content: buildDslTemplate(
          methodName,
          '  /** 基础 */\n  id: "tabsPanel",\n  activeName: "tab1",\n  items: [\n    { name: "tab1", label: "标签一" },\n    { name: "tab2", label: "标签二" },\n  ],\n\n  /** 属性 */\n  type: "card",\n  stretch: true,\n\n  /** 样式 */\n  className: "custom-tabs",\n\n  /** 事件 */\n  onTabClick: {\n    action: "setVar",\n    target: "$vars.activeTab",\n  },',
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
        label: "滑块色",
        content: "#domId .el-tabs__active-bar { background: #409eff; }",
      },
      {
        id: buildPresetId(type, "card"),
        label: "卡片背景",
        content: "#domId .el-tabs__item { background: #f5f7fa; }",
      },
    ],
  },
  {
    name: "pagination",
    types: ["Pagination"],
    detail: (methodName, type) => [
      {
        id: buildPresetId(type, "basic"),
        label: "分页基础配置",
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
          '  /** 基础 */\n  id: "pagination",\n  total: 200,\n  pageSize: 20,\n  currentPage: 1,\n\n  /** 布局 */\n  layout: "total, sizes, prev, pager, next, jumper",\n  pageSizes: [10, 20, 50, 100],\n\n  /** 样式 */\n  className: "custom-pagination",\n\n  /** 事件 */\n  onChange: {\n    action: "setVar",\n    target: "$vars.pageIndex",\n  },',
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
        label: "标签基础配置",
        content: buildDslTemplate(
          methodName,
          '  id: "tagStatus",\n  text: "运行中",\n  type: "success",',
        ),
      },
      {
        id: buildPresetId(type, "closable"),
        label: "可关闭标签",
        content: buildDslTemplate(
          methodName,
          '  closable: true,\n  effect: "dark",',
        ),
      },
      {
        id: buildPresetId(type, "full-dsl"),
        label: "完整 DSL 模板",
        content: buildDslTemplate(
          methodName,
          '  /** 基础 */\n  id: "tagStatus",\n  text: "运行中",\n  type: "success",\n  effect: "light",\n\n  /** 状态 */\n  closable: true,\n\n  /** 样式 */\n  className: "custom-tag",\n\n  /** 事件 */\n  onClose: {\n    action: "setVar",\n    target: "$vars.tagClosed",\n  },',
        ),
      },
    ],
    style: (type) => [
      {
        id: buildPresetId(type, "bg"),
        label: "标签背景色",
        content: "#domId .el-tag { background: #ecf5ff; }",
      },
      {
        id: buildPresetId(type, "border"),
        label: "标签边框",
        content: "#domId .el-tag { border-color: #b3d8ff; }",
      },
      {
        id: buildPresetId(type, "radius"),
        label: "标签圆角",
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
        label: "滑块基础配置",
        content: buildDslTemplate(
          methodName,
          '  id: "sliderValue",\n  value: 30,\n  min: 0,\n  max: 100,',
        ),
      },
      {
        id: buildPresetId(type, "range"),
        label: "范围滑块配置",
        content: buildDslTemplate(
          methodName,
          "  range: true,\n  showStops: true,\n  step: 10,",
        ),
      },
      {
        id: buildPresetId(type, "full-dsl"),
        label: "完整 DSL 模板",
        content: buildDslTemplate(
          methodName,
          '  /** 基础 */\n  id: "sliderValue",\n  value: 30,\n  min: 0,\n  max: 100,\n  step: 1,\n\n  /** 属性 */\n  range: false,\n  showStops: false,\n\n  /** 样式 */\n  className: "custom-slider",\n\n  /** 事件 */\n  onChange: {\n    action: "setVar",\n    target: "$vars.sliderValue",\n  },',
        ),
      },
    ],
    style: (type) => [
      {
        id: buildPresetId(type, "bar"),
        label: "滑块色",
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
        label: "卡片基础配置",
        content: buildDslTemplate(
          methodName,
          '  id: "cardInfo",\n  header: "标题",\n  shadow: "hover",',
        ),
      },
      {
        id: buildPresetId(type, "simple"),
        label: "简洁卡片",
        content: buildDslTemplate(
          methodName,
          '  header: "概览",\n  shadow: "never",',
        ),
      },
      {
        id: buildPresetId(type, "full-dsl"),
        label: "完整 DSL 模板",
        content: buildDslTemplate(
          methodName,
          '  /** 基础 */\n  id: "cardInfo",\n  header: "标题",\n  shadow: "hover",\n\n  /** 样式 */\n  className: "custom-card",\n  style: {\n    padding: "12px",\n  },',
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
        label: "卡片圆角",
        content: "#domId .el-card { border-radius: 8px; }",
      },
      {
        id: buildPresetId(type, "border"),
        label: "卡片边框",
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
        label: "穿梭基础配置",
        content: buildDslTemplate(
          methodName,
          '  id: "transferData",\n  data: [\n    { key: "A", label: "选项A" },\n    { key: "B", label: "选项B" },\n  ],\n  value: [],',
        ),
      },
      {
        id: buildPresetId(type, "filter"),
        label: "穿梭筛选配置",
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
          '  /** 基础 */\n  id: "transferData",\n  data: [\n    { key: "A", label: "选项A" },\n    { key: "B", label: "选项B" },\n  ],\n  value: [],\n\n  /** 属性 */\n  filterable: true,\n  filterPlaceholder: "请输入关键字",\n\n  /** 样式 */\n  className: "custom-transfer",\n\n  /** 事件 */\n  onChange: {\n    action: "setVar",\n    target: "$vars.transferValue",\n  },',
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
        label: "穿梭面板边框",
        content: "#domId .el-transfer-panel { border-color: #e4e7ed; }",
      },
      {
        id: buildPresetId(type, "title"),
        label: "标题字体",
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
        content: buildDslTemplate(
          methodName,
          '  placement: "top",\n  reverse: false,',
        ),
      },
      {
        id: buildPresetId(type, "full-dsl"),
        label: "完整 DSL 模板",
        content: buildDslTemplate(
          methodName,
          '  /** 基础 */\n  id: "timeline",\n  items: [\n    { timestamp: "2024-01-01", content: "创建" },\n    { timestamp: "2024-01-02", content: "完成" },\n  ],\n\n  /** 属性 */\n  placement: "top",\n  reverse: false,\n\n  /** 样式 */\n  className: "custom-timeline",',
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
        label: "标题字体",
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
        label: "步骤基础配置",
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
          '  /** 基础 */\n  id: "steps",\n  active: 1,\n  items: [\n    { title: "开始" },\n    { title: "处理" },\n    { title: "完成" },\n  ],\n\n  /** 属性 */\n  direction: "horizontal",\n  finishStatus: "success",\n\n  /** 样式 */\n  className: "custom-steps",',
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
        label: "完成色",
        content: "#domId .is-finish .el-step__icon { border-color: #67c23a; }",
      },
      {
        id: buildPresetId(type, "title"),
        label: "标题字体",
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
        label: "轮播基础配置",
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
          '  /** 基础 */\n  id: "carousel",\n  height: "240px",\n  items: [\n    { src: "https://", title: "图片1" },\n    { src: "https://", title: "图片2" },\n  ],\n\n  /** 属性 */\n  autoplay: true,\n  interval: 3000,\n  arrow: "hover",\n\n  /** 样式 */\n  className: "custom-carousel",',
        ),
      },
    ],
    style: (type) => [
      {
        id: buildPresetId(type, "dot"),
        label: "指示器颜色",
        content:
          "#domId .el-carousel__indicator.is-active button { background: #409eff; }",
      },
      {
        id: buildPresetId(type, "arrow"),
        label: "箭头颜色",
        content: "#domId .el-carousel__arrow { background: rgba(0,0,0,0.4); }",
      },
      {
        id: buildPresetId(type, "radius"),
        label: "轮播圆角",
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
        label: "日历基础配置",
        content: buildDslTemplate(
          methodName,
          '  id: "calendar",\n  value: new Date(),',
        ),
      },
      {
        id: buildPresetId(type, "range"),
        label: "日历范围",
        content: buildDslTemplate(
          methodName,
          "  range: [new Date(), new Date()],",
        ),
      },
      {
        id: buildPresetId(type, "full-dsl"),
        label: "完整 DSL 模板",
        content: buildDslTemplate(
          methodName,
          '  /** 基础 */\n  id: "calendar",\n  value: new Date(),\n\n  /** 属性 */\n  range: null,\n\n  /** 样式 */\n  className: "custom-calendar",',
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
        label: "布局基础配置",
        content: buildDslTemplate(
          methodName,
          '  id: "layout",\n  direction: "horizontal",\n  gutter: 12,',
        ),
      },
      {
        id: buildPresetId(type, "size"),
        label: "布局尺寸配置",
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
          '  /** 基础 */\n  id: "layout",\n  direction: "horizontal",\n  gutter: 12,\n\n  /** 尺寸 */\n  span: 12,\n  offset: 0,\n  height: "100%",\n  width: "100%",\n\n  /** 样式 */\n  className: "custom-layout",',
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
        label: "容器边框",
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
const resolveElementPlusDetailPresets = (type) => {
  const group = elementPlusPresetGroups.find((item) =>
    item.types.includes(type),
  );
  const methodName = getDslMethodName(type);
  if (group?.detail) {
    return group
      .detail(methodName, type)
      .map((preset) => normalizeDslPreset(type, preset))
      .map((preset) => ({
        ...preset,
        tags: preset.tags || [type, "detail", group.name],
        keywords:
          preset.keywords ||
          [preset.label, type, group.name].filter(Boolean).join(" "),
      }));
  }
  return [
    normalizeDslPreset(type, {
      id: buildPresetId(type, "basic"),
      label: "基础 DSL 模板",
      content: buildDslTemplate(methodName, '  id: "component",\n  props: {},'),
    }),
  ];
};

/**
 * 获取 Element Plus 样式配置预设
 * @param {string} type - 组件类型
 * @returns {Array<{ id: string, label: string, content: string }>}
 */
const resolveElementPlusStylePresets = (type) => {
  const group = elementPlusPresetGroups.find((item) =>
    item.types.includes(type),
  );
  if (group?.style) {
    return group.style(type).map((preset) => ({
      ...preset,
      tags: preset.tags || [type, "style", group.name],
      keywords:
        preset.keywords ||
        [preset.label, type, group.name].filter(Boolean).join(" "),
    }));
  }
  return stylePresets;
};

/**
 * ?????????
 * @param {string} propName - ???
 * @returns {string | undefined}
 */
const getRegionDefaultValue = (propName) => regionSizeDefaults[propName];

/**
 * ????????
 * @param {string | number | undefined} value - ???
 * @returns {{ value: string, unit: string }}
 */
const parseSize = (value) => {
  if (!value || value === "auto") {
    return { value: "", unit: "auto" };
  }
  const str = String(value);
  const match = str.match(/^([0-9.]+)(px|%)?$/);
  if (match) {
    return { value: match[1], unit: match[2] || "px" };
  }
  return { value: "", unit: "auto" };
};

/**
 * ????????
 * @param {string | number | undefined} value - ???
 * @returns {number | undefined}
 */
const parseSizeToNumber = (value) => {
  const parsed = parseSize(value);
  if (!parsed.value || parsed.unit !== "px") return undefined;
  const num = Number.parseFloat(parsed.value);
  return Number.isFinite(num) ? num : undefined;
};

/**
 * ??????????
 * @param {{ key: string }} item - ????
 * @returns {string}
 */
const getRegionSizeText = (item) => {
  if (!item) return "";
  if (item.key === "aside") return "??";
  if (item.key === "header" || item.key === "footer") return "??";
  return "";
};

/**
 * ????????
 * @param {{ toggleProp: string }} item - ????
 * @returns {boolean}
 */
const isRegionEnabled = (item) => {
  if (!item?.toggleProp) return true;
  return Boolean(getPropValue(item.toggleProp));
};

/**
 * ??????????????????
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
 * ?????????????
 * @param {string} propName - ???
 * @returns {number | undefined}
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
 * ???????????
 * @param {string | null} propName - ???
 * @returns {string}
 */
const getRegionMaxLabel = (propName) => {
  if (!propName) return "";
  const maxValue = resolveRegionMaxValue(propName);
  if (!maxValue) return "";
  return `<= ${Math.round(maxValue)}px`;
};

/**
 * ?????????????
 * @param {string} propName - ???
 * @param {string} value - ??
 * @param {string} unit - ??
 * @returns {{ value: string, unit: string }}
 */
const clampRegionSize = (propName, value, unit) => {
  const maxSize = resolveContainerMaxSize();
  const maxValue = resolveRegionMaxValue(propName);
  if (unit === "%") {
    const num = Number.parseFloat(value || "0");
    if (!Number.isFinite(num)) return { value, unit };
    if (maxValue !== undefined && (maxSize.width || maxSize.height)) {
      const base = propName === "asideWidth" ? maxSize.width : maxSize.height;
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
 * ?????????????????
 * @param {import('@/editor-core').ComponentNode | null} containerNode - ????
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
 * ????????
 * @param {Record<string, any> | undefined} props - ????
 */
const syncRegionSizeState = (props) => {
  const safeProps = props || {};
  regionSizeState.value = {
    headerHeight: parseSize(
      safeProps.headerHeight ?? getRegionDefaultValue("headerHeight"),
    ),
    asideWidth: parseSize(
      safeProps.asideWidth ?? getRegionDefaultValue("asideWidth"),
    ),
    footerHeight: parseSize(
      safeProps.footerHeight ?? getRegionDefaultValue("footerHeight"),
    ),
  };
};

watch(
  currentElement,
  (el) => {
    elementLabel.value = el?.label || "";
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
 * 获取组件 Manifest
 */
const manifest = computed(() => {
  const type = currentElement.value?.type;
  if (!type) return null;
  return getManifest(type);
});

const configDialogTitle = computed(() =>
  configDialogType.value === "detail" ? "详细配置" : "样式配置",
);
const configEditorLanguage = computed(() =>
  configDialogType.value === "detail" ? "javascript" : "css",
);
const currentPresetOptions = computed(() => {
  const type = currentElement.value?.type;
  if (configDialogType.value === "detail") {
    if (type === "Button") {
      return buttonDetailPresets
        .map((preset) => normalizeDslPreset(type, preset))
        .map((preset) => ({
          ...preset,
          tags: preset.tags || [type, "detail", "button"],
          keywords:
            preset.keywords ||
            [preset.label, type, "button"].filter(Boolean).join(" "),
        }));
    }
    if (isElementPlusType(type)) return resolveElementPlusDetailPresets(type);
    return detailPresets;
  }
  if (type === "Button") {
    return buttonStylePresets.map((preset) => ({
      ...preset,
      tags: preset.tags || [type, "style", "button"],
      keywords:
        preset.keywords ||
        [preset.label, type, "button"].filter(Boolean).join(" "),
    }));
  }
  if (isElementPlusType(type)) return resolveElementPlusStylePresets(type);
  return stylePresets;
});

const filteredPresetOptions = computed(() => {
  const keyword = String(presetSearch.value || "")
    .trim()
    .toLowerCase();
  if (!keyword) return currentPresetOptions.value;
  return currentPresetOptions.value.filter((item) => {
    const haystack = `${item.label || ""} ${item.tags?.join(" ") || ""} ${item.keywords || ""}`;
    return haystack.toLowerCase().includes(keyword);
  });
});

const bindingDialogTitle = computed(() => {
  const label = bindingProp.value?.label || bindingProp.value?.name;
  return label ? `绑定数据 - ${label}` : "绑定数据";
});

const bindingDialogDescription = computed(() => {
  const elementName = elementLabel.value || elementType.value || "组件";
  const propLabel = bindingProp.value?.label || bindingProp.value?.name || "";
  return propLabel
    ? `${elementName} ${propLabel} 绑定脚本`
    : `${elementName} 绑定脚本`;
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
 * ???????????
 * @param {{ props?: Array<{ name: string }> }} group - ????
 * @returns {boolean}
 */
const isRegionGroup = (group) => {
  return Boolean(group?.props?.some((prop) => prop.name === "showHeader"));
};

/**
 * ???????
 */
const groupedProps = computed(() => {
  if (!manifest.value) return [];

  const groups = new Map();
  for (const prop of manifest.value.props) {
    const groupName = prop.group || "??";
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

const pageVars = computed(() => {
  docVersion.value;
  const pageId = currentPageId.value;
  if (!pageId || !doc.value) return {};
  const vars = doc.value.vars?.pages?.[pageId];
  return vars && typeof vars === "object" ? vars : {};
});

/**
 * ????????
 * @param {string} nodeId - ?? ID
 * @param {Set<string>} nameSet - ????
 */
const collectComponentNames = (nodeId, nameSet) => {
  if (!doc.value || !nodeId) return;
  const node = doc.value.getNode?.(nodeId);
  if (!node) return;
  const label = node.label || node.type;
  if (label) nameSet.add(label);
  (node.children || []).forEach((childId) =>
    collectComponentNames(childId, nameSet),
  );
};

const pageComponentNames = computed(() => {
  docVersion.value;
  const rootId = currentPage.value?.rootNodeId;
  if (!rootId || !doc.value) return [];
  const nameSet = new Set();
  collectComponentNames(rootId, nameSet);
  return Array.from(nameSet);
});

const bindingProjectGroupTree = computed(() => [
  {
    id: "all",
    label: "全部",
    type: "group",
    children: buildBindingGroupTree(projectVariableGroups.value || []),
  },
]);

const bindingProjectVariableRows = computed(() => {
  const keyword = String(bindingProjectVarSearch.value || "").toLowerCase();
  const items = Object.entries(projectVariables.value || {}).map(
    ([name, detail]) => ({
      name,
      groupId: detail?.groupId || null,
      type: detail?.type || "string",
      description: detail?.description || "",
      mapped: detail?.source?.type === "dataCenter" || detail?.mapped === true,
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

const bindingPageGroupTree = computed(() => [
  { id: "page-root", label: "页面变量", type: "group", children: [] },
]);

const bindingPageVariableRows = computed(() => {
  const keyword = String(bindingPageVarSearch.value || "").toLowerCase();
  return Object.entries(pageVars.value || {})
    .map(([name, detail]) => ({
      name,
      type: detail?.type || "string",
      description: detail?.description || "",
    }))
    .filter((item) => {
      if (!keyword) return true;
      return String(item.name || "")
        .toLowerCase()
        .includes(keyword);
    });
});

const bindingPageComponentTree = computed(() => {
  docVersion.value;
  const rootId = currentPage.value?.rootNodeId;
  if (!rootId || !doc.value) return [];
  const buildNode = (nodeId) => {
    const node = doc.value.getNode(nodeId);
    if (!node) return null;
    const children = (node.children || [])
      .map((childId) => buildNode(childId))
      .filter(Boolean);
    const label = node.label || node.type || "组件";
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

const bindingCustomScriptTree = computed(() => {
  const groups = globalScripts.value?.custom?.groups || [];
  const items = globalScripts.value?.custom?.items || [];
  const groupMap = new Map();
  const roots = [];

  groups.forEach((group) => {
    groupMap.set(group.id, {
      id: group.id,
      label: group.name,
      type: "group",
      children: [],
    });
  });

  groupMap.forEach((node, id) => {
    const group = groups.find((item) => item.id === id);
    if (group?.parentId && groupMap.has(group.parentId)) {
      groupMap.get(group.parentId).children.push(node);
    } else {
      roots.push(node);
    }
  });

  items.forEach((item) => {
    if (!item?.id) return;
    const node = {
      id: item.id,
      label: item.name || "未命名",
      type: "item",
      params: item.params || item.args || "",
    };
    if (item.groupId && groupMap.has(item.groupId)) {
      groupMap.get(item.groupId).children.push(node);
    } else {
      roots.push(node);
    }
  });

  return roots;
});

const bindingCompletions = computed(() => {
  const items = [
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
      detail: "工程变量",
      prefix: "$global.",
    });
  });

  (globalScripts.value?.custom?.items || []).forEach((script) => {
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
      detail: "自定义脚本",
      prefix: "customScripts.",
    });
  });

  Object.keys(pageVars.value || {}).forEach((name) => {
    items.push({
      label: name,
      insertText: name,
      kind: "Variable",
      detail: "页面变量",
      prefix: "$vars.",
    });
  });

  return items;
});

const detailCompletions = computed(() => {
  const items = [
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
      detail: "工程变量",
      prefix: "$global.",
    });
  });

  (globalScripts.value?.custom?.items || []).forEach((scriptItem) => {
    if (!scriptItem?.name) return;
    const params =
      typeof scriptItem.params === "string" && scriptItem.params.trim()
        ? scriptItem.params.trim()
        : typeof scriptItem.args === "string"
          ? scriptItem.args.trim()
          : "";
    const call = params
      ? `${scriptItem.name}(${params})`
      : `${scriptItem.name}()`;
    items.push({
      label: scriptItem.name,
      insertText: call,
      kind: "Function",
      detail: "自定义脚本",
      prefix: "customScripts.",
    });
  });

  Object.keys(pageVars.value || {}).forEach((name) => {
    items.push({
      label: name,
      insertText: name,
      kind: "Variable",
      detail: "页面变量",
      prefix: "$vars.",
    });
  });

  pageComponentNames.value.forEach((name) => {
    items.push({
      label: name,
      insertText: name,
      kind: "Variable",
      detail: "页面组件",
      prefix: "components.",
    });
  });

  return items;
});

const configEditorCompletions = computed(() => {
  return configDialogType.value === "detail" ? detailCompletions.value : [];
});

/**
 * 构建绑定面板分组树
 * @param {Array<{ id: string, name: string, parentId?: string }>} groups - 分组数据
 * @returns {Array<{ id: string, label: string, type: string, children: Array }>}
 */
const buildBindingGroupTree = (groups) => {
  const groupMap = new Map();
  const roots = [];
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
      groupMap.get(group.parentId).children.push(node);
    } else {
      roots.push(node);
    }
  });
  return roots;
};

/**
 * 过滤绑定侧边栏节点
 * @param {string} value - 关键字
 * @param {{ label?: string }} data - 节点数据
 * @returns {boolean}
 */
const filterBindingSidebarNode = (value, data) => {
  if (!value) return true;
  return String(data?.label || "")
    .toLowerCase()
    .includes(value.toLowerCase());
};

watch(bindingScriptSearch, (value) => {
  bindingCustomTreeRef.value?.filter?.(value);
});

watch(bindingComponentSearch, (value) => {
  bindingComponentTreeRef.value?.filter?.(value);
});

/**
 * 规范化样式配置输出
 * @param {string} content - 原始内容
 * @returns {string}
 */
const formatStyleConfigOutput = (content) => {
  const text = String(content || "");
  if (configDialogType.value !== "style") return text;
  if (text.includes("{")) return text;
  const trimmed = text.trim();
  if (!trimmed) return "";
  return `#domId {
${trimmed}
}`;
};

/**
 * 自动为选择器补全 #domId 作用域
 * @param {string} content - 样式内容
 * @returns {string}
 */
const prefixStyleConfigScope = (content) => {
  const text = String(content || "").trim();
  if (!text || !text.includes("{")) return text;
  const blocks = text.split("}");
  const rebuilt = blocks
    .map((block) => {
      const [selector, body] = block.split("{");
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
};

/**
 * 轻量校验 DSL 脚本
 * @param {string} content - DSL 内容
 * @returns {{ valid: boolean, message?: string }}
 */
const validateDetailConfig = (content) => {
  const text = String(content || "").trim();
  if (!text) return { valid: true };
  try {
    // 仅做语法检查，不执行
    new Function(text);
    return { valid: true };
  } catch (error) {
    return { valid: false, message: error?.message || "DSL 语法错误" };
  }
};

/**
 * 变量引用结构化
 * @param {string} content - DSL 内容
 * @returns {string}
 */
const normalizeDetailConfigBindings = (content) => {
  let text = String(content || "");
  text = text.replace(/:\s*(\$vars\.[A-Za-z0-9_]+)/g, ': { $var: "$1" }');
  text = text.replace(/:\s*(\$global\.[A-Za-z0-9_]+)/g, ': { $var: "$1" }');
  text = text.replace(
    /:\s*["'](\$vars\.[A-Za-z0-9_]+)["']/g,
    ': { $var: "$1" }',
  );
  text = text.replace(
    /:\s*["'](\$global\.[A-Za-z0-9_]+)["']/g,
    ': { $var: "$1" }',
  );
  return text;
};

/**
 * ?????
 * @param {string} propName - ???
 */
const getPropValue = (propName) => {
  const el = currentElement.value;
  if (!el?.props) return undefined;
  return el.props[propName];
};

/**
 * ??????????
 * @param {string} propName - ???
 * @returns {{ value: string, unit: string } | undefined}
 */
const getRegionSizeState = (propName) => regionSizeState.value[propName];

/**
 * ??????????
 * @param {string} propName - ???
 * @param {string} value - ??
 * @param {string} unit - ??
 */
const setRegionSizeState = (propName, value, unit) => {
  if (!propName) return;
  regionSizeState.value = {
    ...regionSizeState.value,
    [propName]: { value, unit },
  };
};

/**
 * ??????
 * @param {string} propName - ???
 * @returns {string}
 */
const getSizeValue = (propName) => {
  if (!propName) return "";
  const cached = getRegionSizeState(propName);
  if (cached) return cached.value;
  return "";
};

/**
 * ??????
 * @param {string} propName - ???
 * @returns {string}
 */
const getSizeUnit = (propName) => {
  if (!propName) return "auto";
  const cached = getRegionSizeState(propName);
  if (cached) return cached.unit;
  return "auto";
};

/**
 * ????????
 * @param {string} propName - ???
 * @param {string} value - ??
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
 * ????????
 * @param {string} propName - ???
 * @param {string} unit - ??
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
 * Props ??????? Manifest ????
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
const hasPropBinding = (propName) => {
  return Boolean(currentElement.value?.bindings?.[propName]);
};

/**
 * ????????
 */
const hasStyleSelection = computed(() => selectedNode.value !== null);

const hasDetailConfig = computed(() => {
  const value = currentElement.value?.detailConfig;
  return Boolean(String(value || "").trim());
});

const hasStyleConfig = computed(() => {
  const value = currentElement.value?.styleConfig;
  return Boolean(String(value || "").trim());
});

/**
 * ?????????
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
 * ??????
 */
const containerMinSize = computed(() =>
  resolveElContainerMinSize(selectedNode.value),
);

/**
 * ????
 */
const currentStyle = computed(() => {
  if (!selectedNode.value) return {};
  return selectedNode.value.style || {};
});

/**
 * ??????
 * @param {Object} newStyle - ???
 */
const handleStyleChange = (newStyle) => {
  if (!selectedNode.value) return;
  editorStore.updateNode(selectedNode.value.id, { style: newStyle });
};

/**
 * ??????
 * @param {"style" | "detail"} type - 配置类型
 */
const openConfigDialog = (type) => {
  if (!selectedNode.value) return;
  configDialogType.value = type === "detail" ? "detail" : "style";
  configDraft.value =
    configDialogType.value === "detail"
      ? selectedNode.value.detailConfig || ""
      : selectedNode.value.styleConfig || "";
  selectedPresetId.value = "";
  presetSearch.value = "";
  configDialogVisible.value = true;
};

/**
 * ??????
 */
const saveConfigDialog = () => {
  const node = selectedNode.value;
  if (!node) return;
  let content = String(configDraft.value || "");
  if (configDialogType.value === "style") {
    const normalized = formatStyleConfigOutput(content);
    content = prefixStyleConfigScope(normalized);
    configDraft.value = content;
  } else {
    content = normalizeDetailConfigBindings(content);
    configDraft.value = content;
    const validation = validateDetailConfig(content);
    if (!validation.valid) {
      ElMessage.warning(validation.message || "详细配置未通过校验");
      return;
    }
  }
  if (configDialogType.value === "detail") {
    editorStore.updateNode(node.id, { detailConfig: content });
  } else {
    editorStore.updateNode(node.id, { styleConfig: content });
  }
  void editorStore.saveCurrentPage?.();
  configDialogVisible.value = false;
};

/**
 * 清除配置内容
 */
const clearConfigDialog = () => {
  const node = selectedNode.value;
  if (!node) return;
  configDraft.value = "";
  if (configDialogType.value === "detail") {
    editorStore.updateNode(node.id, { detailConfig: "" });
  } else {
    editorStore.updateNode(node.id, { styleConfig: "" });
  }
  void editorStore.saveCurrentPage?.();
};

/**
 * ??????
 * @param {string} id - ?? ID
 */
const handlePresetChange = (id) => {
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
    configDraft.value = configDraft.value;
  });
};

/**
 * 获取绑定表达式
 * @param {string} propName - 属性名
 * @returns {string}
 */
const resolveBindingExpr = (propName) => {
  if (!propName) return "";
  const target = currentElement.value;
  const binding = target?.bindings?.[propName];
  if (!binding || typeof binding !== "object") return "";
  if (binding.kind === "expr") return binding.expr || "";
  if (binding.kind === "var" && binding.name) {
    return binding.scope === "global"
      ? `$global.${binding.name}`
      : `$vars.${binding.name}`;
  }
  return "";
};

/**
 * 结构化表达式/变量引用
 * @param {string} raw - 输入内容
 * @returns {string}
 */
const normalizeBindingReference = (raw) => {
  const trimmed = String(raw || "").trim();
  if (!trimmed) return "";
  if (trimmed.startsWith("{ $var:") || trimmed.startsWith("{ $expr:"))
    return trimmed;
  if (trimmed.startsWith("$vars.") || trimmed.startsWith("$global.")) {
    return `{ $var: "${trimmed}" }`;
  }
  return `{ $expr: "${trimmed.replace(/"/g, '\\"')}" }`;
};

/**
 * 解析绑定输入
 * @param {string} raw - 输入内容
 * @returns {{ kind: "expr" | "var", expr?: string, scope?: "page" | "global", name?: string } | null}
 */
const parseBindingInput = (raw) => {
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
};

/**
 * 打开绑定编辑器
 * @param {{ name: string, label?: string }} propDef - 属性定义
 */
const handleBindClick = (propDef) => {
  if (!propDef?.name || !currentElement.value) return;
  bindingProp.value = {
    name: propDef.name,
    label: propDef.label || propDef.name,
  };
  bindingEditorCode.value = resolveBindingExpr(propDef.name);
  bindingDialogVisible.value = true;
};

/**
 * 插入自定义脚本
 * @param {{ type?: string, label?: string, params?: string }} data - 节点数据
 */
const handleBindingCustomScriptInsert = (data) => {
  if (data?.type === "group") return;
  const params = data?.params ? data.params : "";
  const call = params
    ? `customScripts.${data.label}(${params})`
    : `customScripts.${data.label}()`;
  bindingEditorRef.value?.insertText?.(call);
};

/**
 * 插入页面组件引用
 * @param {{ componentName?: string }} data - 节点数据
 */
const handleBindingComponentInsert = (data) => {
  if (!data?.componentName) return;
  bindingEditorRef.value?.insertText?.(`components.${data.componentName}`);
};

/**
 * 打开变量枚举
 */
const openBindingVariableEnum = () => {
  bindingProjectVarSearch.value = "";
  bindingPageVarSearch.value = "";
  bindingEnumTab.value = "project";
  bindingEnumSelectedProjectGroupId.value = null;
  bindingEnumSelectedPageGroupId.value = "page-root";
  bindingEnumSelectedProjectVar.value = null;
  bindingEnumSelectedPageVar.value = null;
  bindingVariableEnumVisible.value = true;
};

const handleBindingProjectGroupSelect = (data) => {
  if (!data) {
    bindingEnumSelectedProjectGroupId.value = null;
    return;
  }
  bindingEnumSelectedProjectGroupId.value = data.id === "all" ? null : data.id;
};

const handleBindingPageGroupSelect = (data) => {
  bindingEnumSelectedPageGroupId.value = data?.id || "page-root";
};

const handleBindingProjectRowClick = (row) => {
  bindingEnumSelectedProjectVar.value = row || null;
};

const handleBindingPageRowClick = (row) => {
  bindingEnumSelectedPageVar.value = row || null;
};

const handleBindingProjectRowDblClick = (row) => {
  bindingEnumSelectedProjectVar.value = row || null;
  confirmBindingEnumInsert();
};

const handleBindingPageRowDblClick = (row) => {
  bindingEnumSelectedPageVar.value = row || null;
  confirmBindingEnumInsert();
};

const bindingEnumProjectRowClass = ({ row }) => {
  if (bindingEnumSelectedProjectVar.value?.name === row.name)
    return "is-selected";
  return "";
};

const bindingEnumPageRowClass = ({ row }) => {
  if (bindingEnumSelectedPageVar.value?.name === row.name) return "is-selected";
  return "";
};

const confirmBindingEnumInsert = () => {
  if (
    bindingEnumTab.value === "page" &&
    bindingEnumSelectedPageVar.value?.name
  ) {
    bindingEditorRef.value?.insertText?.(
      `$vars.${bindingEnumSelectedPageVar.value.name}`,
    );
    bindingVariableEnumVisible.value = false;
    return;
  }
  if (bindingEnumSelectedProjectVar.value?.name) {
    bindingEditorRef.value?.insertText?.(
      `$global.${bindingEnumSelectedProjectVar.value.name}`,
    );
    bindingVariableEnumVisible.value = false;
  }
};

/**
 * 保存绑定配置
 */
const saveBinding = () => {
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
  void editorStore.saveCurrentPage?.();
  bindingEditorCode.value = normalizeBindingReference(code);
  bindingDialogVisible.value = false;
};

/**
 * ??????
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
 * 更新按钮 DOM ID
 * @param {string} value - DOM ID
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

.bind-btn.is-active {
  opacity: 1;
  color: #409eff;
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

.style-config-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.style-config-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 12px;
  color: var(--el-text-color-regular);
}

.config-btn.is-active {
  color: #409eff;
  border-color: #b3d8ff;
  background: #ecf5ff;
}

.config-toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 12px 16px;
  align-items: center;
  margin-bottom: 12px;
  padding: 8px 10px;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  background: #fafafa;
}

.config-toolbar-item {
  display: flex;
  align-items: center;
  gap: 6px;
}

.config-label {
  font-size: 12px;
  color: #606266;
}

.config-select {
  width: 180px;
}

.config-editor {
  min-height: 520px;
}

.editor-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 10px 16px;
  padding: 10px 12px;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  background: #fafafa;
  margin-bottom: 10px;
  align-items: center;
}

.meta-title {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

.meta-desc {
  font-size: 12px;
  color: #606266;
}

.meta-actions {
  margin-left: auto;
  display: flex;
  align-items: center;
}

.icon-button {
  background: #eef2ff;
  border: none;
  color: #4f46e5;
}

.icon-button:hover {
  background: #e0e7ff;
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
  border-left: 1px solid #e4e7ed;
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
  font-size: 12px;
  font-weight: 600;
  color: #606266;
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
  border-radius: 6px;
  transition: background-color 0.2s;
}

.tree-node:hover {
  background: #f5f7fa;
}

.node-icon {
  color: #94a3b8;
  flex-shrink: 0;
}

.node-item .node-icon.icon-custom {
  color: #10b981;
}

.node-variable .node-icon.icon-variable {
  color: #0ea5e9;
}

.node-label {
  font-size: 13px;
  color: #303133;
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
  border-right: 1px solid #e4e7ed;
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
  background: #eef2ff;
}
</style>
