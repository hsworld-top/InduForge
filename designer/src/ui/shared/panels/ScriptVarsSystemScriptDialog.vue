<!--
  脚本面板：系统启动/关闭脚本大编辑器（Monaco + 侧栏树）
-->
<script setup>
import { ref, watch } from "vue";
import IconEpDocument from "~icons/ep/document";
import IconEpEditPen from "~icons/ep/edit-pen";
import IconEpFolder from "~icons/ep/folder";
import IconEpList from "~icons/ep/list";
import MonacoEditor from "@/components/common/monaco-editor-async";

defineProps({
  dialogTitle: { type: String, required: true },
  metaTitle: { type: String, required: true },
  /** @type {unknown[]} */
  customScriptSidebarTree: { type: Array, default: () => [] },
  /** @type {unknown[]} */
  pageSidebarTree: { type: Array, default: () => [] },
  /** @type {unknown[]} */
  jsCompletions: { type: Array, default: () => [] },
  filterSidebarNode: { type: Function, required: true },
  beforeClose: { type: Function, default: undefined },
});
const emit = defineEmits(["openVariableEnum", "save", "customInsert", "pageInsert"]);
const visible = defineModel({ type: Boolean, default: false });
const systemCode = defineModel("systemCode", { type: String, default: "" });
const scriptSearch = defineModel("scriptSearch", { type: String, default: "" });
const pageSearch = defineModel("pageSearch", { type: String, default: "" });

const monacoRef = ref(null);
const customTreeRef = ref(null);
const pageTreeInnerRef = ref(null);

watch(scriptSearch, (value) => {
  customTreeRef.value?.filter?.(value);
});

watch(pageSearch, (value) => {
  pageTreeInnerRef.value?.filter?.(value);
});

defineExpose({
  insertText: (text) => monacoRef.value?.insertText?.(text),
  format: () => monacoRef.value?.format?.(),
});
</script>

<template>
  <el-dialog
    v-model="visible"
    :title="dialogTitle"
    width="980px"
    top="3vh"
    :close-on-click-modal="false"
    :lock-scroll="false"
    :before-close="beforeClose"
  >
    <div class="editor-meta">
      <div class="meta-title">{{ metaTitle }}</div>
      <div class="meta-desc">系统脚本</div>
      <div class="meta-actions">
        <el-tooltip content="枚举工程变量" placement="top">
          <el-button class="icon-button" size="small" circle @click="emit('openVariableEnum')">
            <IconEpList />
          </el-button>
        </el-tooltip>
      </div>
    </div>
    <div class="editor-body">
      <div class="editor-main">
        <MonacoEditor
          ref="monacoRef"
          v-model="systemCode"
          language="javascript"
          height="520px"
          :completions="jsCompletions"
        />
      </div>
      <div class="editor-sidebar">
        <div class="sidebar-section">
          <div class="sidebar-title">自定义脚本</div>
          <el-input v-model="scriptSearch" size="small" placeholder="搜索脚本/分组" clearable />
          <div class="sidebar-scroll">
            <el-tree
              ref="customTreeRef"
              :data="customScriptSidebarTree"
              node-key="id"
              :default-expand-all="true"
              :expand-on-click-node="false"
              :filter-node-method="filterSidebarNode"
              @node-click="(data) => emit('customInsert', data)"
            >
              <template #default="{ data }">
                <div class="tree-node" :class="`node-${data.type}`">
                  <el-icon class="node-icon icon-custom">
                    <IconEpFolder v-if="data.type === 'group'" />
                    <IconEpEditPen v-else />
                  </el-icon>
                  <span
                    class="node-label"
                    :class="{
                      'is-group': data.type === 'group' || data.type === 'folder',
                    }"
                    >{{ data.label }}</span
                  >
                </div>
              </template>
            </el-tree>
          </div>
        </div>
        <div class="sidebar-section">
          <div class="sidebar-title">页面</div>
          <el-input v-model="pageSearch" size="small" placeholder="搜索页面/分组" clearable />
          <div class="sidebar-scroll">
            <el-tree
              ref="pageTreeInnerRef"
              :data="pageSidebarTree"
              node-key="id"
              :default-expand-all="true"
              :expand-on-click-node="false"
              :filter-node-method="filterSidebarNode"
              @node-click="(data) => emit('pageInsert', data)"
            >
              <template #default="{ data }">
                <div class="tree-node" :class="`node-${data.type}`">
                  <el-icon class="node-icon icon-page">
                    <IconEpFolder v-if="data.type === 'folder'" />
                    <IconEpDocument v-else />
                  </el-icon>
                  <span
                    class="node-label"
                    :class="{
                      'is-group': data.type === 'group' || data.type === 'folder',
                    }"
                    >{{ data.label }}</span
                  >
                </div>
              </template>
            </el-tree>
          </div>
        </div>
      </div>
    </div>
    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" @click="emit('save')">保存 (Ctrl+S)</el-button>
    </template>
  </el-dialog>
</template>
