<!--
  属性面板：绑定编辑器内「变量枚举」弹窗。
  统一变量选择器只展示工程变量；页面变量由脚本/详细配置右侧页面上下文提供。
-->
<script setup lang="ts">
import { useI18n } from "vue-i18n";
import IconEpFolder from "~icons/ep/folder";

interface BindingEnumTreeNodeLike {
  id?: string | number;
  label?: string;
}

type BindingEnumRowClassName = string | string[] | Record<string, boolean> | undefined;

defineProps<{
  bindingProjectGroupTree?: BindingEnumTreeNodeLike[];
  bindingProjectVariableRows?: unknown[];
  filterBindingSidebarNode: (data: BindingEnumTreeNodeLike, node: unknown) => boolean;
  bindingEnumProjectRowClass: (...args: unknown[]) => BindingEnumRowClassName;
  canConfirmInsert?: boolean;
}>();

const emit = defineEmits<{
  (event: "projectGroupSelect", data: BindingEnumTreeNodeLike): void;
  (event: "projectRowClick", row: unknown): void;
  (event: "projectRowDblclick", row: unknown): void;
  (event: "confirmInsert"): void;
  (event: "cancel"): void;
}>();

const visible = defineModel<boolean>({ default: false });
const bindingProjectVarSearch = defineModel<string>("bindingProjectVarSearch", { default: "" });
const { t } = useI18n();

function handleProjectGroupSelect(data: BindingEnumTreeNodeLike) {
  emit("projectGroupSelect", data);
}

function handleProjectRowClick(row: unknown) {
  emit("projectRowClick", row);
}

function handleProjectRowDblclick(row: unknown) {
  emit("projectRowDblclick", row);
}
</script>

<template>
  <el-dialog
    v-model="visible"
    :title="t('propertyPanel.bindingVarEnum.title')"
    width="760px"
    :z-index="3100"
    append-to-body
    :modal-append-to-body="true"
    :close-on-click-modal="false"
    :lock-scroll="false"
  >
    <div class="enum-caption">{{ t("propertyPanel.bindingVarEnum.projectVars") }}</div>
    <div class="enum-layout">
      <div class="enum-left">
        <div class="sidebar-title">{{ t("propertyPanel.bindingVarEnum.groups") }}</div>
        <el-tree
          :data="bindingProjectGroupTree"
          node-key="id"
          :default-expand-all="true"
          :expand-on-click-node="false"
          :filter-node-method="filterBindingSidebarNode"
          @node-click="handleProjectGroupSelect"
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
          :placeholder="t('propertyPanel.bindingVarEnum.searchProjectVars')"
          clearable
        />
        <el-table
          :data="bindingProjectVariableRows"
          size="small"
          height="320"
          highlight-current-row
          :row-class-name="bindingEnumProjectRowClass"
          @row-click="handleProjectRowClick"
          @row-dblclick="handleProjectRowDblclick"
        >
          <el-table-column prop="name" :label="t('propertyPanel.bindingVarEnum.name')" min-width="150" />
          <el-table-column prop="type" :label="t('propertyPanel.bindingVarEnum.type')" width="90" />
          <el-table-column prop="description" :label="t('propertyPanel.bindingVarEnum.description')" min-width="140" />
          <el-table-column prop="sourceLabel" :label="t('propertyPanel.bindingVarEnum.source')" min-width="150" />
        </el-table>
      </div>
    </div>
    <template #footer>
      <el-button @click="emit('cancel')">{{ t("propertyPanel.bindingVarEnum.cancel") }}</el-button>
      <el-button type="primary" :disabled="!canConfirmInsert" @click="emit('confirmInsert')">
        {{ t("propertyPanel.bindingVarEnum.insert") }}
      </el-button>
    </template>
  </el-dialog>
</template>
