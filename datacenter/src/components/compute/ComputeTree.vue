<template>
  <aside class="compute-tree" :class="{ 'is-collapsed': collapsed }">
    <header class="compute-tree__head">
      <div v-if="!collapsed">
        <h2>计算单元</h2>
        <p>{{ total }} 个单元</p>
      </div>
      <button
        type="button"
        class="compute-tree__icon-btn"
        :title="collapsed ? '展开资源树' : '收起资源树'"
        :aria-label="collapsed ? '展开资源树' : '收起资源树'"
        @click="$emit('toggleCollapse')"
      >
        <IconTablerLayoutSidebarLeftCollapse
          v-if="!collapsed"
          class="compute-tree__icon"
        />
        <IconTablerLayoutSidebarLeftExpand v-else class="compute-tree__icon" />
      </button>
    </header>

    <div v-if="collapsed" class="compute-tree__rail">
      <button
        type="button"
        class="compute-tree__rail-btn"
        title="新建单元"
        aria-label="新建单元"
        @click="$emit('createUnit')"
      >
        <IconTablerPlus class="compute-tree__button-icon" />
      </button>
      <button
        type="button"
        class="compute-tree__rail-btn"
        title="新建文件夹"
        aria-label="新建文件夹"
        @click="$emit('createFolder')"
      >
        <IconTablerFolderPlus class="compute-tree__button-icon" />
      </button>
      <button
        type="button"
        class="compute-tree__rail-btn"
        title="刷新"
        aria-label="刷新"
        @click="$emit('refresh')"
      >
        <IconTablerRefresh class="compute-tree__button-icon" />
      </button>
    </div>

    <div v-else class="compute-tree__toolbar">
      <el-input
        v-model="keyword"
        class="compute-tree__search"
        clearable
        size="small"
        placeholder="搜索计算单元"
        :prefix-icon="SearchIcon"
      />
      <button
        type="button"
        class="compute-tree__primary"
        title="新建单元"
        aria-label="新建单元"
        @click="$emit('createUnit')"
      >
        <IconTablerPlus class="compute-tree__button-icon" />
      </button>
      <button
        type="button"
        class="compute-tree__secondary"
        title="新建文件夹"
        aria-label="新建文件夹"
        @click="$emit('createFolder')"
      >
        <IconTablerFolderPlus class="compute-tree__button-icon" />
      </button>
      <button
        type="button"
        class="compute-tree__secondary compute-tree__refresh"
        title="刷新"
        aria-label="刷新"
        @click="$emit('refresh')"
      >
        <IconTablerRefresh class="compute-tree__button-icon" />
      </button>
    </div>

    <div v-if="listError && !collapsed" class="compute-tree__error">
      <IconTablerAlertCircle class="compute-tree__error-icon" />
      <span>{{ listError }}</span>
    </div>
    <div v-else-if="foldersError && !collapsed" class="compute-tree__warn">
      <IconTablerAlertCircle class="compute-tree__error-icon" />
      <span>{{ foldersError }}</span>
    </div>

    <div v-if="loading && !collapsed" class="compute-tree__loading">
      <el-skeleton :rows="8" animated />
    </div>

    <div v-else-if="!collapsed" class="compute-tree__body">
      <ComputeTreeBranch
        v-for="folder in filteredFolders"
        :key="folder.id"
        :node="folder"
        :selected-unit-id="selectedUnitId"
        @select-unit="$emit('selectUnit', $event)"
        @folder-contextmenu="openFolderMenu"
        @unit-contextmenu="openUnitMenu"
      />
      <button
        v-for="unit in filteredRootUnits"
        :key="String(unit.id)"
        type="button"
        class="compute-tree__unit"
        :class="{ 'is-active': String(unit.id) === selectedUnitId }"
        @click="$emit('selectUnit', String(unit.id))"
        @contextmenu.prevent.stop="openUnitMenu($event, unit)"
      >
        <IconTablerFileCode class="compute-tree__unit-icon" />
        <span class="compute-tree__unit-main">
          <span class="compute-tree__unit-name">{{ unit.name }}</span>
          <span class="compute-tree__unit-path">
            {{ unit.outputPath || unit.path || "calc.*" }}
          </span>
        </span>
        <StatusBadge
          class="compute-tree__unit-status"
          :tone="statusTone(unit.status)"
          :text="statusText(unit.status)"
        />
      </button>

      <EmptyState
        v-if="filteredRootUnits.length === 0 && filteredFolders.length === 0"
        icon-name="compute"
        :title="keyword ? '没有匹配的计算单元' : '暂无计算单元'"
        :description="
          keyword ? '换个关键词再试。' : '新建计算单元后会显示在列表中。'
        "
      />
    </div>

    <Teleport to="body">
      <div
        v-if="contextMenu.visible"
        class="compute-tree__menu-mask"
        @click="closeContextMenu"
        @contextmenu.prevent="closeContextMenu"
      >
        <div
          class="compute-tree__context-menu"
          :style="{ left: `${contextMenu.x}px`, top: `${contextMenu.y}px` }"
          @click.stop
        >
          <button type="button" @click="emitContextAction('rename')">
            <IconTablerPencil class="compute-tree__menu-icon" />
            <span>重命名</span>
          </button>
          <button type="button" @click="emitContextAction('move')">
            <IconTablerFolderSymlink class="compute-tree__menu-icon" />
            <span>移动到分组</span>
          </button>
        </div>
      </div>
    </Teleport>
  </aside>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { Search } from "@element-plus/icons-vue";
import IconTablerAlertCircle from "~icons/tabler/alert-circle";
import IconTablerFileCode from "~icons/tabler/file-code";
import IconTablerFolderPlus from "~icons/tabler/folder-plus";
import IconTablerFolderSymlink from "~icons/tabler/folder-symlink";
import IconTablerLayoutSidebarLeftCollapse from "~icons/tabler/layout-sidebar-left-collapse";
import IconTablerLayoutSidebarLeftExpand from "~icons/tabler/layout-sidebar-left-expand";
import IconTablerPencil from "~icons/tabler/pencil";
import IconTablerPlus from "~icons/tabler/plus";
import IconTablerRefresh from "~icons/tabler/refresh";
import type {
  ComputeFolder,
  ComputeUnit,
} from "@/api/schemas/compute.schema";
import EmptyState from "@/components/shared/EmptyState.vue";
import StatusBadge from "@/components/shared/StatusBadge.vue";
import ComputeTreeBranch from "./ComputeTreeBranch.vue";
import {
  buildComputeFolderTree,
  filterComputeTree,
  type ComputeFolderTreeNode,
} from "./computeTreeModel";

const SearchIcon = Search;

const props = withDefaults(
  defineProps<{
    units: ComputeUnit[];
    folders: ComputeFolder[];
    selectedUnitId?: string | null;
    loading?: boolean;
    listError?: string;
    foldersError?: string;
    collapsed?: boolean;
  }>(),
  {
    selectedUnitId: null,
    loading: false,
    listError: "",
    foldersError: "",
    collapsed: false,
  },
);

const emit = defineEmits<{
  (event: "selectUnit", id: string): void;
  (event: "createUnit"): void;
  (event: "createFolder"): void;
  (event: "refresh"): void;
  (event: "toggleCollapse"): void;
  (event: "renameUnit", unit: ComputeUnit): void;
  (event: "moveUnit", unit: ComputeUnit): void;
  (event: "renameFolder", folder: ComputeFolderTreeNode): void;
  (event: "moveFolder", folder: ComputeFolderTreeNode): void;
}>();

const keyword = ref("");
const contextMenu = ref<{
  visible: boolean;
  type: "unit" | "folder" | null;
  x: number;
  y: number;
  unit: ComputeUnit | null;
  folder: ComputeFolderTreeNode | null;
}>({
  visible: false,
  type: null,
  x: 0,
  y: 0,
  unit: null,
  folder: null,
});

const total = computed(() => props.units.length);
const tree = computed(() => buildComputeFolderTree(props.folders, props.units));
const filteredTree = computed(() =>
  filterComputeTree(tree.value.rootFolders, tree.value.rootUnits, keyword.value),
);
const filteredFolders = computed(() => filteredTree.value.folders);
const filteredRootUnits = computed(() => filteredTree.value.units);

function openUnitMenu(event: MouseEvent, unit: ComputeUnit) {
  contextMenu.value = {
    visible: true,
    type: "unit",
    x: Math.min(event.clientX, window.innerWidth - 180),
    y: Math.min(event.clientY, window.innerHeight - 96),
    unit,
    folder: null,
  };
}

function openFolderMenu(event: MouseEvent, folder: ComputeFolderTreeNode) {
  contextMenu.value = {
    visible: true,
    type: "folder",
    x: Math.min(event.clientX, window.innerWidth - 180),
    y: Math.min(event.clientY, window.innerHeight - 96),
    unit: null,
    folder,
  };
}

function closeContextMenu() {
  contextMenu.value.visible = false;
}

function emitContextAction(action: "rename" | "move") {
  const { type, unit, folder } = contextMenu.value;
  closeContextMenu();
  if (type === "unit" && unit) {
    if (action === "rename") {
      emit("renameUnit", unit);
      return;
    }
    emit("moveUnit", unit);
    return;
  }
  if (type === "folder" && folder) {
    if (action === "rename") {
      emit("renameFolder", folder);
      return;
    }
    emit("moveFolder", folder);
    return;
  }
}

const statusText = (status?: string) => {
  const map: Record<string, string> = {
    enabled: "启用",
    idle: "空闲",
    running: "运行中",
    error: "异常",
    disabled: "停用",
  };
  return map[status || ""] || "未知";
};

const statusTone = (status?: string) => {
  if (status === "running" || status === "enabled" || status === "idle")
    return "success";
  if (status === "error") return "danger";
  if (status === "disabled") return "muted";
  return "info";
};
</script>

<style scoped>
.compute-tree {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border-right: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
}

.compute-tree__head {
  min-height: 46px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--dc-border);
}

.compute-tree.is-collapsed .compute-tree__head {
  justify-content: center;
  padding: 8px;
}

.compute-tree__head h2 {
  margin: 0;
  color: var(--dc-text);
  font-size: 15px;
  font-weight: 800;
  line-height: 1.3;
}

.compute-tree__head p {
  margin: 2px 0 0;
  color: var(--dc-text-muted);
  font-size: 12px;
}

.compute-tree__toolbar {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 30px 30px 30px;
  gap: 6px;
  padding: 8px;
  border-bottom: 1px solid var(--dc-border);
}

.compute-tree__search {
  grid-column: auto;
}

.compute-tree__primary,
.compute-tree__secondary,
.compute-tree__icon-btn {
  min-width: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border-radius: var(--dc-radius-sm);
  font-size: 13px;
  font-weight: 700;
  transition:
    border-color 0.18s ease,
    background-color 0.18s ease,
    color 0.18s ease,
    transform 0.18s ease;
}

.compute-tree__primary,
.compute-tree__secondary {
  width: 30px;
  height: 30px;
  padding: 0;
}

.compute-tree__primary {
  border: 1px solid var(--dc-primary);
  background: var(--dc-primary);
  color: var(--dc-surface-raised);
}

.compute-tree__secondary,
.compute-tree__icon-btn {
  border: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}

.compute-tree__primary:hover,
.compute-tree__secondary:hover,
.compute-tree__icon-btn:hover {
  transform: translateY(-1px);
}

.compute-tree__secondary:hover,
.compute-tree__icon-btn:hover {
  border-color: rgba(29, 78, 216, 0.28);
  color: var(--dc-primary);
}

.compute-tree__icon-btn {
  width: 30px;
  height: 30px;
}

.compute-tree__rail {
  display: grid;
  justify-items: center;
  gap: 8px;
  padding: 10px 8px;
}

.compute-tree__rail-btn {
  width: 32px;
  height: 32px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}

.compute-tree__rail-btn:hover {
  color: var(--dc-primary);
  background: var(--dc-primary-soft);
}

.compute-tree__icon,
.compute-tree__button-icon,
.compute-tree__error-icon,
.compute-tree__unit-icon {
  width: 16px;
  height: 16px;
}

.compute-tree__body {
  min-height: 0;
  flex: 1;
  display: grid;
  align-content: start;
  gap: 4px;
  padding: 8px;
  overflow-y: auto;
}

.compute-tree__loading {
  padding: 12px;
}

.compute-tree__error,
.compute-tree__warn {
  display: flex;
  align-items: flex-start;
  gap: 7px;
  margin: 10px 12px 0;
  padding: 9px 10px;
  border-radius: var(--dc-radius-sm);
  font-size: 12px;
  line-height: 1.5;
}

.compute-tree__error {
  border: 1px solid rgba(220, 38, 38, 0.2);
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
}

.compute-tree__warn {
  border: 1px solid rgba(217, 119, 6, 0.22);
  background: var(--dc-warning-soft);
  color: var(--dc-warning);
}

.compute-tree__unit {
  width: 100%;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  text-align: left;
}

.compute-tree__unit:hover {
  border-color: var(--dc-border);
  background: var(--dc-surface-muted);
  color: var(--dc-text);
}

.compute-tree__unit {
  min-height: 38px;
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
  padding: 5px 7px;
}

.compute-tree__unit.is-active {
  border-color: rgba(29, 78, 216, 0.28);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.compute-tree__unit-main {
  min-width: 0;
  display: grid;
  gap: 3px;
}

.compute-tree__unit-name,
.compute-tree__unit-path {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.compute-tree__unit-name {
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 700;
}

.compute-tree__unit-path {
  color: var(--dc-text-muted);
  font-size: 11px;
}

.compute-tree__unit-status {
  max-width: 76px;
}

.compute-tree__menu-mask {
  position: fixed;
  inset: 0;
  z-index: 2100;
}

.compute-tree__context-menu {
  position: fixed;
  min-width: 148px;
  padding: 4px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
}

.compute-tree__context-menu button {
  width: 100%;
  height: 30px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 8px;
  border: 0;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  font-size: 13px;
  text-align: left;
}

.compute-tree__context-menu button:hover {
  background: var(--dc-surface-muted);
  color: var(--dc-primary);
}

.compute-tree__menu-icon {
  width: 15px;
  height: 15px;
  flex: 0 0 auto;
}

@media (max-width: 900px) {
  .compute-tree {
    border-right: none;
    border-bottom: 1px solid var(--dc-border);
  }
}
</style>
