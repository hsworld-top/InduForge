<!--
  脚本面板：系统启动/关闭脚本大编辑器（Monaco + 侧栏树）
-->
<script setup lang="ts">
import { ref, watch } from "vue";
import IconEpDocument from "~icons/ep/document";
import IconEpEditPen from "~icons/ep/edit-pen";
import IconEpFolder from "~icons/ep/folder";
import IconEpList from "~icons/ep/list";
import MonacoEditor from "@/ui/shared/widgets/base/monaco-editor-async";

interface SidebarNodeLike {
  id: string;
  label: string;
  type: string;
}

defineProps<{
  dialogTitle: string;
  metaTitle: string;
  customScriptSidebarTree?: SidebarNodeLike[];
  pageSidebarTree?: SidebarNodeLike[];
  jsCompletions?: unknown[];
  filterSidebarNode: (value: string, data: SidebarNodeLike) => boolean;
  beforeClose?: (...args: unknown[]) => void;
}>();
const emit = defineEmits(["openVariableEnum", "save", "customInsert", "pageInsert"]);
const visible = defineModel<boolean>({ default: false });
const systemCode = defineModel<string>("systemCode", { default: "" });
const scriptSearch = defineModel<string>("scriptSearch", { default: "" });
const pageSearch = defineModel<string>("pageSearch", { default: "" });

interface TreeFilterLike {
  filter?: (value: string) => void;
}

interface MonacoExposeLike {
  insertText?: (text: string) => void;
  format?: () => void;
}

const monacoRef = ref<MonacoExposeLike | null>(null);
const customTreeRef = ref<TreeFilterLike | null>(null);
const pageTreeInnerRef = ref<TreeFilterLike | null>(null);

watch(scriptSearch, (value: string) => {
  customTreeRef.value?.filter?.(value);
});

watch(pageSearch, (value: string) => {
  pageTreeInnerRef.value?.filter?.(value);
});

defineExpose({
  insertText: (text: string) => monacoRef.value?.insertText?.(text),
  format: () => monacoRef.value?.format?.(),
});

function handleCustomInsert(data: SidebarNodeLike) {
  emit("customInsert", data);
}

function handlePageInsert(data: SidebarNodeLike) {
  emit("pageInsert", data);
}
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
              @node-click="handleCustomInsert"
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
              @node-click="handlePageInsert"
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
