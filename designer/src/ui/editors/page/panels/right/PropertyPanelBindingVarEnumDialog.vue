<!--
  属性面板：绑定编辑器内「变量枚举」弹窗（工程变量 / 页面变量）
-->
<script setup>
import IconEpFolder from "~icons/ep/folder";

defineProps({
  /** @type {unknown[]} */
  bindingProjectGroupTree: { type: Array, default: () => [] },
  /** @type {unknown[]} */
  bindingPageGroupTree: { type: Array, default: () => [] },
  /** @type {unknown[]} */
  bindingProjectVariableRows: { type: Array, default: () => [] },
  /** @type {unknown[]} */
  bindingPageVariableRows: { type: Array, default: () => [] },
  filterBindingSidebarNode: { type: Function, required: true },
  bindingEnumProjectRowClass: { type: Function, required: true },
  bindingEnumPageRowClass: { type: Function, required: true },
  canConfirmInsert: { type: Boolean, default: false },
});
const emit = defineEmits([
  "projectGroupSelect",
  "pageGroupSelect",
  "projectRowClick",
  "pageRowClick",
  "projectRowDblclick",
  "pageRowDblclick",
  "confirmInsert",
  "cancel",
]);
const visible = defineModel({ type: Boolean, default: false });
const bindingEnumTab = defineModel("bindingEnumTab", { type: String, default: "project" });
const bindingProjectVarSearch = defineModel("bindingProjectVarSearch", {
  type: String,
  default: "",
});
const bindingPageVarSearch = defineModel("bindingPageVarSearch", { type: String, default: "" });
</script>

<template>
  <el-dialog
    v-model="visible"
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
              :data="bindingProjectGroupTree"
              node-key="id"
              :default-expand-all="true"
              :expand-on-click-node="false"
              :filter-node-method="filterBindingSidebarNode"
              @node-click="(data) => emit('projectGroupSelect', data)"
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
              :row-class-name="bindingEnumProjectRowClass"
              @row-click="(row) => emit('projectRowClick', row)"
              @row-dblclick="(row) => emit('projectRowDblclick', row)"
            >
              <el-table-column prop="name" label="变量名" min-width="160" />
              <el-table-column prop="type" label="类型" width="90" />
              <el-table-column prop="description" label="描述" min-width="160" />
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
              :data="bindingPageGroupTree"
              node-key="id"
              :default-expand-all="true"
              :expand-on-click-node="false"
              :filter-node-method="filterBindingSidebarNode"
              @node-click="(data) => emit('pageGroupSelect', data)"
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
              :row-class-name="bindingEnumPageRowClass"
              @row-click="(row) => emit('pageRowClick', row)"
              @row-dblclick="(row) => emit('pageRowDblclick', row)"
            >
              <el-table-column prop="name" label="变量名" min-width="160" />
              <el-table-column prop="type" label="类型" width="90" />
              <el-table-column prop="description" label="描述" min-width="200" />
            </el-table>
          </div>
        </div>
      </el-tab-pane>
    </el-tabs>
    <template #footer>
      <el-button @click="emit('cancel')">取消</el-button>
      <el-button type="primary" :disabled="!canConfirmInsert" @click="emit('confirmInsert')">
        插入
      </el-button>
    </template>
  </el-dialog>
</template>
