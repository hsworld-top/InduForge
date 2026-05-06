<!--
  脚本面板：系统启动/关闭脚本大编辑器（Monaco + 侧栏树）
-->
<script setup lang="ts">
import { ref, watch } from "vue";
import { useI18n } from "vue-i18n";
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

interface PageVariableRowLike {
  name: string;
  type?: string;
}

defineProps<{
  dialogTitle: string;
  metaTitle: string;
  customScriptSidebarTree?: SidebarNodeLike[];
  pageSidebarTree?: SidebarNodeLike[];
  pageVariableRows?: PageVariableRowLike[];
  jsCompletions?: unknown[];
  filterSidebarNode: (value: string, data: SidebarNodeLike) => boolean;
  beforeClose?: (...args: unknown[]) => void;
}>();
const emit = defineEmits([
  "openVariableEnum",
  "save",
  "customInsert",
  "pageInsert",
  "pageVariableInsert",
]);
const visible = defineModel<boolean>({ default: false });
const systemCode = defineModel<string>("systemCode", { default: "" });
const scriptSearch = defineModel<string>("scriptSearch", { default: "" });
const pageSearch = defineModel<string>("pageSearch", { default: "" });
const pageVariableSearch = defineModel<string>("pageVariableSearch", { default: "" });

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
const { t } = useI18n();

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

function handlePageVariableInsert(row: PageVariableRowLike) {
  emit("pageVariableInsert", row);
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
      <div class="meta-desc">{{ t("scriptPanel.editor.systemScript") }}</div>
      <div class="meta-actions">
        <el-tooltip :content="t('scriptPanel.editor.variableEnum')" placement="top">
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
          <div class="sidebar-title">{{ t("scriptPanel.editor.customScripts") }}</div>
          <el-input v-model="scriptSearch" size="small" :placeholder="t('scriptPanel.editor.searchScripts')" clearable />
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
          <div class="sidebar-title">{{ t("scriptPanel.editor.pageVariables") }}</div>
          <el-input
            v-model="pageVariableSearch"
            size="small"
            :placeholder="t('scriptPanel.editor.searchPageVariables')"
            clearable
          />
          <div class="sidebar-scroll page-var-list">
            <div
              v-for="row in pageVariableRows"
              :key="row.name"
              class="page-var-item"
              @click="handlePageVariableInsert(row)"
            >
              <el-icon class="node-icon icon-page">
                <IconEpList />
              </el-icon>
              <span class="node-label">{{ row.name }}</span>
              <span class="page-var-type">{{ row.type || 'string' }}</span>
            </div>
          </div>
        </div>
        <div class="sidebar-section">
          <div class="sidebar-title">{{ t("scriptPanel.editor.pages") }}</div>
          <el-input v-model="pageSearch" size="small" :placeholder="t('scriptPanel.editor.searchPages')" clearable />
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
      <el-button @click="visible = false">{{ t("scriptPanel.editor.cancel") }}</el-button>
      <el-button type="primary" @click="emit('save')">{{ t("scriptPanel.editor.saveShortcut") }}</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.page-var-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.page-var-item {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 28px;
  padding: 4px 6px;
  border-radius: var(--designer-radius-sm);
  cursor: pointer;
}

.page-var-item:hover {
  background: var(--designer-hover-surface);
}

.page-var-type {
  margin-left: auto;
  color: var(--designer-text-muted);
  font-size: 12px;
}
</style>
