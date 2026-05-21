<template>
  <section class="compute-tree-branch">
    <button
      type="button"
      class="compute-tree-branch__folder"
      @click="expanded = !expanded"
      @contextmenu.prevent.stop="$emit('folderContextmenu', $event, node)"
    >
      <IconTablerChevronRight
        class="compute-tree-branch__chevron"
        :class="{ 'is-open': expanded }"
      />
      <IconTablerFolderOpen v-if="expanded" class="compute-tree-branch__icon" />
      <IconTablerFolder v-else class="compute-tree-branch__icon" />
      <span class="compute-tree-branch__folder-name">{{ node.name }}</span>
      <span class="compute-tree-branch__count">{{ totalCount }}</span>
    </button>

    <div v-if="expanded" class="compute-tree-branch__children">
      <button
        v-for="unit in node.units"
        :key="String(unit.id)"
        type="button"
        class="compute-tree-branch__unit"
        :class="{ 'is-active': String(unit.id) === selectedUnitId }"
        @click="$emit('selectUnit', String(unit.id))"
        @contextmenu.prevent.stop="$emit('unitContextmenu', $event, unit)"
      >
        <IconTablerFileCode class="compute-tree-branch__unit-icon" />
        <span class="compute-tree-branch__unit-main">
          <span class="compute-tree-branch__unit-name">{{ unit.name }}</span>
        </span>
        <span
          class="compute-tree-branch__status"
          :class="`is-${unitStatusTone(unit)}`"
          :title="unitStatusText(unit)"
          :aria-label="unitStatusText(unit)"
        ></span>
      </button>

      <ComputeTreeBranch
        v-for="child in node.children"
        :key="child.id"
        :node="child"
        :selected-unit-id="selectedUnitId"
        :dirty-unit-ids="dirtyUnitIds"
        @select-unit="$emit('selectUnit', $event)"
        @folder-contextmenu="
          (mouseEvent, folder) => $emit('folderContextmenu', mouseEvent, folder)
        "
        @unit-contextmenu="
          (mouseEvent, unit) => $emit('unitContextmenu', mouseEvent, unit)
        "
      />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import IconTablerChevronRight from "~icons/tabler/chevron-right";
import IconTablerFileCode from "~icons/tabler/file-code";
import IconTablerFolder from "~icons/tabler/folder";
import IconTablerFolderOpen from "~icons/tabler/folder-open";
import type { ComputeUnit } from "@/api/schemas/compute.schema";
import type { ComputeFolderTreeNode } from "./computeTreeModel";

defineOptions({ name: "ComputeTreeBranch" });

const props = defineProps<{
  node: ComputeFolderTreeNode;
  selectedUnitId?: string | null;
  dirtyUnitIds?: string[];
}>();

defineEmits<{
  (event: "selectUnit", id: string): void;
  (event: "unitContextmenu", mouseEvent: MouseEvent, unit: ComputeUnit): void;
  (
    event: "folderContextmenu",
    mouseEvent: MouseEvent,
    folder: ComputeFolderTreeNode,
  ): void;
}>();

const expanded = ref(true);
const dirtyUnitIdSet = computed(() => new Set((props.dirtyUnitIds || []).map(String)));

const countUnits = (node: ComputeFolderTreeNode): number =>
  node.units.length +
  node.children.reduce((sum, child) => sum + countUnits(child), 0);

const totalCount = computed(() => countUnits(props.node));

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

const isUnitDirty = (unit: ComputeUnit) => dirtyUnitIdSet.value.has(String(unit.id));

const unitStatusText = (unit: ComputeUnit) =>
  isUnitDirty(unit) ? "未保存" : statusText(unit.status);

const unitStatusTone = (unit: ComputeUnit) =>
  isUnitDirty(unit) ? "warning" : statusTone(unit.status);
</script>

<style scoped>
.compute-tree-branch {
  display: grid;
  gap: 2px;
}

.compute-tree-branch__folder,
.compute-tree-branch__unit {
  width: 100%;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
}

.compute-tree-branch__folder {
  min-height: 28px;
  display: grid;
  grid-template-columns: 16px 18px minmax(0, 1fr) auto;
  align-items: center;
  gap: 5px;
  padding: 0 6px;
  font-size: 13px;
  font-weight: 700;
  text-align: left;
}

.compute-tree-branch__folder:hover,
.compute-tree-branch__unit:hover {
  border-color: var(--dc-border);
  background: var(--dc-surface-muted);
  color: var(--dc-text);
}

.compute-tree-branch__chevron {
  width: 14px;
  height: 14px;
  transform: rotate(0deg);
  transition: transform 0.16s ease;
}

.compute-tree-branch__chevron.is-open {
  transform: rotate(90deg);
}

.compute-tree-branch__icon,
.compute-tree-branch__unit-icon {
  width: 16px;
  height: 16px;
  color: var(--dc-primary);
}

.compute-tree-branch__folder-name,
.compute-tree-branch__unit-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.compute-tree-branch__count {
  min-width: 20px;
  padding: 2px 6px;
  border-radius: 999px;
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
  font-size: 11px;
  text-align: center;
}

.compute-tree-branch__children {
  display: grid;
  gap: 2px;
  margin-left: 10px;
  padding-left: 6px;
}

.compute-tree-branch__unit {
  min-height: 30px;
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr) auto;
  align-items: center;
  gap: 6px;
  padding: 3px 6px;
  text-align: left;
}

.compute-tree-branch__unit.is-active {
  border-color: rgba(29, 78, 216, 0.28);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.compute-tree-branch__unit-main {
  min-width: 0;
  display: block;
}

.compute-tree-branch__unit-name {
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 700;
}

.compute-tree-branch__status {
  width: 8px;
  height: 8px;
  justify-self: end;
  border-radius: 999px;
  background: var(--dc-text-muted);
}

.compute-tree-branch__status.is-success {
  background: var(--dc-success);
}

.compute-tree-branch__status.is-danger {
  background: var(--dc-danger);
}

.compute-tree-branch__status.is-warning {
  background: var(--dc-warning);
}

.compute-tree-branch__status.is-muted {
  background: var(--dc-text-muted);
}

.compute-tree-branch__status.is-info {
  background: var(--dc-primary);
}
</style>
