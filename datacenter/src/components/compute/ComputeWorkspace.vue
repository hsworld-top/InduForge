<template>
  <div
    class="compute-workspace"
    :class="{ 'is-tree-collapsed': treeCollapsed }"
  >
    <ComputeTree
      :units="computeStore.list"
      :folders="computeStore.folders"
      :selected-unit-id="selectedUnitId"
      :loading="computeStore.loading || computeStore.foldersLoading"
      :list-error="computeStore.listError"
      :folders-error="computeStore.foldersError"
      :collapsed="treeCollapsed"
      @select-unit="selectUnit"
      @create-unit="showCreateUnitDialog = true"
      @create-folder="showCreateFolderDialog = true"
      @refresh="loadWorkspace"
      @toggle-collapse="treeCollapsed = !treeCollapsed"
    />

    <ComputeEditorShell
      :project-id="String(projectId)"
      :tabs="tabs"
      :active-id="activeTabId"
      :active-draft="activeDraft"
      :loading="computeStore.detailLoading"
      :saving="computeStore.saving"
      :deleting="computeStore.deleting"
      :error="computeStore.detailError"
      :dependencies="computeStore.dependencies"
      :dependencies-loading="computeStore.dependenciesLoading"
      :dependencies-error="computeStore.dependenciesError"
      @activate-tab="activateTab"
      @close-tab="closeTab"
      @save="saveTab"
      @toggle-enabled="toggleEnabled"
      @delete-unit="deleteUnit"
      @mark-dirty="markDirty"
      @refresh-dependencies="loadDependencies"
    />

    <CreateComputeUnitDialog
      v-model="showCreateUnitDialog"
      :folders="computeStore.folders"
      :loading="computeStore.creating"
      :error="computeStore.createError"
      @submit="handleCreateUnit"
    />

    <CreateComputeFolderDialog
      v-model="showCreateFolderDialog"
      :folders="computeStore.folders"
      :loading="computeStore.creating"
      :error="computeStore.createError"
      @submit="handleCreateFolder"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { onBeforeRouteLeave, useRoute, useRouter } from "vue-router";
import { ElMessage, ElMessageBox } from "element-plus";
import type {
  ComputeFolderSave,
  ComputeUnitSave,
} from "@/api/schemas/compute.schema";
import { useComputeStore } from "@/stores/compute.store";
import { getApiErrorMessage } from "@/utils/request";
import ComputeEditorShell from "./ComputeEditorShell.vue";
import ComputeTree from "./ComputeTree.vue";
import CreateComputeFolderDialog from "./CreateComputeFolderDialog.vue";
import CreateComputeUnitDialog from "./CreateComputeUnitDialog.vue";
import {
  draftToSavePayload,
  toComputeDraft,
  type ComputeDraft,
  type ComputeEditorTab,
} from "./computeEditorModel";

const props = defineProps<{
  projectId: string;
  selectedUnitId?: string | null;
}>();

const route = useRoute();
const router = useRouter();
const computeStore = useComputeStore();

const showCreateUnitDialog = ref(false);
const showCreateFolderDialog = ref(false);
const treeCollapsed = ref(false);
const drafts = ref<Record<string, ComputeDraft>>({});
const activeTabId = ref<string | null>(null);

const selectedUnitId = computed(() => props.selectedUnitId || null);

const computeBasePath = computed(() => {
  const prefix = route.path.startsWith("/debug/") ? "/debug" : "";
  return `${prefix}/compute`;
});

const tabs = computed<ComputeEditorTab[]>(() =>
  Object.values(drafts.value).map((draft) => ({
    id: draft.id,
    name: draft.name,
    dirty: draft.dirty,
  })),
);

const activeDraft = computed(() => {
  if (!activeTabId.value) return null;
  return drafts.value[activeTabId.value] || null;
});

const hasDirtyTabs = computed(() =>
  Object.values(drafts.value).some((draft) => draft.dirty),
);

async function loadWorkspace() {
  const projectId = String(props.projectId);
  const results = await Promise.allSettled([
    computeStore.fetchList(projectId),
    computeStore.fetchFolders(projectId),
  ]);
  const failed = results.find((result) => result.status === "rejected");
  if (failed) {
    ElMessage.warning("计算单元部分能力未启用");
  }
}

function loadDependencies() {
  computeStore.fetchDependencies(String(props.projectId)).catch(() => undefined);
}

function selectUnit(id: string) {
  void router.push({
    path: `${computeBasePath.value}/${id}`,
    query: route.query,
  });
}

async function openSelectedUnit(id: string | null) {
  if (!id) {
    activeTabId.value = null;
    return;
  }
  if (drafts.value[id]) {
    activeTabId.value = id;
    return;
  }
  try {
    const unit = await computeStore.openForEdit(String(props.projectId), id);
    drafts.value = {
      ...drafts.value,
      [id]: toComputeDraft(unit),
    };
    activeTabId.value = id;
  } catch {
    // 错误文案已写入 store，由编辑区展示。
  }
}

function activateTab(id: string) {
  activeTabId.value = id;
  selectUnit(id);
}

async function closeTab(id: string) {
  const draft = drafts.value[id];
  if (!draft) return;
  if (draft.dirty) {
    const action = await confirmDirtyClose(draft.name);
    if (action === "cancel") return;
    if (action === "save") {
      const saved = await saveTab(id);
      if (!saved) return;
    }
  }
  const nextDrafts = { ...drafts.value };
  delete nextDrafts[id];
  drafts.value = nextDrafts;
  if (activeTabId.value === id) {
    const next = Object.keys(nextDrafts)[0] || null;
    activeTabId.value = next;
    if (next) {
      selectUnit(next);
    } else {
      void router.push({ path: computeBasePath.value, query: route.query });
    }
  }
}

function markDirty(id: string) {
  const draft = drafts.value[id];
  if (!draft) return;
  draft.dirty = true;
}

async function saveTab(id: string): Promise<boolean> {
  const draft = drafts.value[id];
  if (!draft) return false;
  try {
    const unit = await computeStore.saveUnit(
      String(props.projectId),
      id,
      draftToSavePayload(draft),
    );
    drafts.value = {
      ...drafts.value,
      [id]: toComputeDraft(unit),
    };
    ElMessage.success("计算单元已保存");
    await computeStore.fetchList(String(props.projectId)).catch(() => undefined);
    return true;
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, "保存计算单元失败"));
    return false;
  }
}

async function toggleEnabled(id: string, enabled: boolean) {
  try {
    const unit = await computeStore.setUnitEnabled(
      String(props.projectId),
      id,
      enabled,
    );
    drafts.value = {
      ...drafts.value,
      [id]: toComputeDraft(unit),
    };
    ElMessage.success(enabled ? "计算单元已启用" : "计算单元已停用");
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, "更新计算单元状态失败"));
  }
}

async function deleteUnit(id: string) {
  const draft = drafts.value[id];
  const ok = await ElMessageBox.confirm(
    `确认删除计算单元「${draft?.name || id}」？此操作不可恢复。`,
    "删除计算单元",
    {
      confirmButtonText: "删除",
      cancelButtonText: "取消",
      type: "warning",
    },
  )
    .then(() => true)
    .catch(() => false);
  if (!ok) return;
  try {
    await computeStore.removeUnit(String(props.projectId), id);
    const nextDrafts = { ...drafts.value };
    delete nextDrafts[id];
    drafts.value = nextDrafts;
    const next = Object.keys(nextDrafts)[0] || null;
    activeTabId.value = next;
    if (next) {
      selectUnit(next);
    } else {
      void router.push({ path: computeBasePath.value, query: route.query });
    }
    ElMessage.success("计算单元已删除");
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, "删除计算单元失败"));
  }
}

async function confirmDirtyClose(name: string) {
  try {
    await ElMessageBox.confirm(`计算单元「${name}」有未保存修改。`, "关闭标签", {
      confirmButtonText: "保存",
      cancelButtonText: "丢弃",
      distinguishCancelAndClose: true,
      type: "warning",
      closeOnClickModal: false,
    });
    return "save" as const;
  } catch (action) {
    if (action === "cancel") return "discard" as const;
    return "cancel" as const;
  }
}

function handleBeforeUnload(event: BeforeUnloadEvent) {
  if (!hasDirtyTabs.value) return;
  event.preventDefault();
  event.returnValue = "";
}

onBeforeRouteLeave(async () => {
  if (!hasDirtyTabs.value) return true;
  return ElMessageBox.confirm(
    "当前存在未保存的计算单元修改，离开后这些修改不会保存。",
    "离开计算单元",
    {
      confirmButtonText: "离开",
      cancelButtonText: "取消",
      type: "warning",
    },
  )
    .then(() => true)
    .catch(() => false);
});

async function handleCreateUnit(data: ComputeUnitSave) {
  try {
    const unit = await computeStore.createUnit(String(props.projectId), data);
    showCreateUnitDialog.value = false;
    drafts.value = {
      ...drafts.value,
      [String(unit.id)]: toComputeDraft(unit),
    };
    selectUnit(String(unit.id));
    ElMessage.success("计算单元已创建");
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, "新建计算单元失败"));
  }
}

async function handleCreateFolder(data: ComputeFolderSave) {
  try {
    await computeStore.createFolder(String(props.projectId), data);
    showCreateFolderDialog.value = false;
    ElMessage.success("文件夹已创建");
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, "新建文件夹失败"));
  }
}

watch(
  () => props.projectId,
  () => {
    drafts.value = {};
    activeTabId.value = null;
    void loadWorkspace();
    loadDependencies();
  },
);

watch(
  selectedUnitId,
  (id) => {
    void openSelectedUnit(id);
  },
  { immediate: true },
);

onMounted(() => {
  void loadWorkspace();
  loadDependencies();
  window.addEventListener("beforeunload", handleBeforeUnload);
});

onBeforeUnmount(() => {
  window.removeEventListener("beforeunload", handleBeforeUnload);
});
</script>

<style scoped>
.compute-workspace {
  height: 100%;
  min-height: 0;
  display: grid;
  grid-template-columns: 260px minmax(0, 1fr);
  overflow: hidden;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
}

.compute-workspace.is-tree-collapsed {
  grid-template-columns: 48px minmax(0, 1fr);
}

@media (max-width: 900px) {
  .compute-workspace {
    grid-template-columns: 1fr;
    grid-template-rows: minmax(260px, 42vh) minmax(0, 1fr);
  }
}
</style>
