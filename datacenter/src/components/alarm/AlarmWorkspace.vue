<template>
  <div class="alarm-workspace">
    <AlarmPolicyManager
      :tree="alarmStore.tree"
      :selection="alarmStore.selection"
      :total="alarmStore.total"
      :selected-id="selectedPolicyId"
      :selected-count="alarmStore.selectedCount"
      :loading="alarmStore.loading"
      :error="alarmStore.listError"
      @refresh="reloadList"
      @create="openCreateDialog"
      @create-group="openCreateGroupDialog"
      @select="selectPolicy"
      @filter="filterList"
      @select-policy="selectPolicyForBatch"
      @select-group="selectGroupForBatch"
      @select-filtered="selectFilteredForBatch"
      @clear-selection="alarmStore.clearSelection"
      @batch-enable="batchEnable"
      @batch-disable="batchDisable"
      @batch-conditions="openBulkConditionDialog"
      @batch-move="batchMove"
      @batch-move-dialog="openBulkMoveDialog"
      @batch-delete="batchDelete"
      @rename-policy="openRenamePolicyDialog"
      @move-policy="openMovePolicyDialog"
      @delete-policy="deletePolicyFromTree"
      @rename-group="openRenameGroupDialog"
      @move-group="openMoveGroupDialog"
      @delete-group="deleteGroupFromTree"
    />

    <AlarmEditorShell
      :project-id="projectId"
      :groups="alarmStore.groups"
      :draft="activeDraft"
      :active-tab="activeTab"
      :loading="alarmStore.detailLoading"
      :error="alarmStore.detailError"
      :saving="alarmStore.saving"
      :deleting="alarmStore.deleting"
      :trial-result="alarmStore.trial.result"
      :trial-running="alarmStore.trial.running"
      :trial-error="alarmStore.trial.error"
      :contract="alarmStore.contract.data"
      :contract-loading="alarmStore.contract.loading"
      :contract-error="alarmStore.contract.error"
      @update="updateActiveDraft"
      @save="saveActiveDraft"
      @toggle="toggleEnabled"
      @delete="deletePolicy"
      @select-tab="selectTab"
      @run-trial="runTrial"
      @refresh-contract="refreshContract"
      @check-current="checkCurrent"
    />

    <CreateAlarmPolicyDialog
      v-model="createDialogVisible"
      :groups="alarmStore.groups"
      :submitting="alarmStore.creating"
      :error="alarmStore.createError"
      @submit="createPolicy"
    />

    <CreateAlarmGroupDialog
      v-model="createGroupDialogVisible"
      :groups="alarmStore.groups"
      :submitting="alarmStore.saving"
      @submit="createGroup"
    />

    <AlarmBulkConditionDialog
      v-model="bulkConditionDialogVisible"
      :submitting="alarmStore.saving"
      @submit="batchApplyConditions"
    />

    <MoveAlarmPoliciesDialog
      v-model="bulkMoveDialogVisible"
      :groups="alarmStore.groups"
      :selected-count="alarmStore.selectedCount"
      :loading="alarmStore.saving"
      @submit="batchMove"
    />

    <RenameAlarmPolicyDialog
      v-model="renamePolicyDialogVisible"
      :policy="contextPolicy"
      :loading="alarmStore.saving"
      @submit="renamePolicy"
    />

    <MoveAlarmPolicyDialog
      v-model="movePolicyDialogVisible"
      :policy="contextPolicy"
      :groups="alarmStore.groups"
      :loading="alarmStore.saving"
      @submit="movePolicy"
    />

    <RenameAlarmGroupDialog
      v-model="renameGroupDialogVisible"
      :group="contextGroup"
      :loading="alarmStore.saving"
      @submit="renameGroup"
    />

    <MoveAlarmGroupDialog
      v-model="moveGroupDialogVisible"
      :group="contextGroup"
      :groups="alarmStore.groups"
      :loading="alarmStore.saving"
      @submit="moveGroup"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { ElMessage, ElMessageBox } from "element-plus";
import type {
  AlarmCondition,
  AlarmPolicy,
  AlarmPolicyGroup,
  AlarmPolicyGroupSave,
  AlarmPolicyTrialPayload,
} from "@/api/schemas/alarm.schema";
import { useAlarmStore } from "@/stores/alarm.store";
import {
  createDefaultAlarmPolicyDraft,
  draftToAlarmPolicySavePayload,
  toAlarmPolicyDraft,
  type AlarmPolicyDraft,
} from "@/components/alarm/alarmPolicyModel";
import AlarmBulkConditionDialog from "./AlarmBulkConditionDialog.vue";
import AlarmEditorShell, { type AlarmEditorTab } from "./AlarmEditorShell.vue";
import AlarmPolicyManager from "./AlarmPolicyManager.vue";
import CreateAlarmGroupDialog from "./CreateAlarmGroupDialog.vue";
import CreateAlarmPolicyDialog from "./CreateAlarmPolicyDialog.vue";
import MoveAlarmGroupDialog from "./MoveAlarmGroupDialog.vue";
import MoveAlarmPolicyDialog from "./MoveAlarmPolicyDialog.vue";
import MoveAlarmPoliciesDialog from "./MoveAlarmPoliciesDialog.vue";
import RenameAlarmGroupDialog from "./RenameAlarmGroupDialog.vue";
import RenameAlarmPolicyDialog from "./RenameAlarmPolicyDialog.vue";
import type { AlarmPolicyGroupNode } from "./alarmPolicyTreeModel";

const props = defineProps<{
  projectId: string;
}>();

type WorkspaceTab = AlarmEditorTab;

const tabs: WorkspaceTab[] = ["config", "test", "contract"];

const route = useRoute();
const router = useRouter();
const alarmStore = useAlarmStore();
const activeDraft = ref<AlarmPolicyDraft | null>(null);
const createDialogVisible = ref(false);
const createGroupDialogVisible = ref(false);
const bulkConditionDialogVisible = ref(false);
const bulkMoveDialogVisible = ref(false);
const renamePolicyDialogVisible = ref(false);
const movePolicyDialogVisible = ref(false);
const renameGroupDialogVisible = ref(false);
const moveGroupDialogVisible = ref(false);
const listParams = ref<Record<string, string>>({});
const contextPolicy = ref<AlarmPolicy | null>(null);
const contextGroup = ref<AlarmPolicyGroup | null>(null);

const selectedPolicyId = computed(() => {
  if (route.params.module !== "alarm") {
    return "";
  }
  const value = route.params.objectId;
  return typeof value === "string" ? value : "";
});

const activeTab = computed<WorkspaceTab>(() => {
  const value = route.params.tab;
  return tabs.includes(value as WorkspaceTab)
    ? (value as WorkspaceTab)
    : "config";
});

const routeBase = computed(() =>
  route.path.startsWith("/debug/") ? "/debug" : "",
);
const replaceAlarmRoute = (policyId?: string, tab: WorkspaceTab = "config") => {
  const tabPath = tab === "config" ? "" : `/${tab}`;
  const policyPath = policyId ? `/${policyId}${tabPath}` : "";
  router.push(`${routeBase.value}/alarm${policyPath}`);
};

const cleanParams = (params: Record<string, string>) =>
  Object.fromEntries(
    Object.entries(params).filter(([, value]) => value.trim() !== ""),
  );

const reloadList = () =>
  alarmStore.fetchTree(props.projectId, listParams.value);

const filterList = (params: Record<string, string>) => {
  listParams.value = cleanParams(params);
  reloadList();
};

const openCreateDialog = () => {
  createDialogVisible.value = true;
};

const openCreateGroupDialog = () => {
  createGroupDialogVisible.value = true;
};

const createGroup = async (payload: AlarmPolicyGroupSave) => {
  await alarmStore.createGroup(props.projectId, payload);
  createGroupDialogVisible.value = false;
  await reloadList();
};

const createPolicy = async (draft: AlarmPolicyDraft) => {
  const policy = await alarmStore.createPolicy(
    props.projectId,
    draftToAlarmPolicySavePayload({
      ...createDefaultAlarmPolicyDraft(),
      ...draft,
      isEnabled: false,
    }),
  );
  activeDraft.value = toAlarmPolicyDraft(policy);
  createDialogVisible.value = false;
  await reloadList();
  replaceAlarmRoute(policy.id, "config");
  ElMessage.success("报警策略已创建");
};

const selectPolicy = (policyId: string) => {
  replaceAlarmRoute(policyId, "config");
};

const selectTab = (tab: WorkspaceTab) => {
  replaceAlarmRoute(selectedPolicyId.value, tab);
};

const refreshContract = async () => {
  if (!selectedPolicyId.value) {
    return;
  }
  try {
    await alarmStore.fetchContract(props.projectId, selectedPolicyId.value);
  } catch {
    // 错误文案由 store 写入并传给契约面板展示。
  }
};

const runTrial = async (payload: AlarmPolicyTrialPayload) => {
  if (!selectedPolicyId.value) {
    return;
  }
  try {
    await alarmStore.runTrial(props.projectId, selectedPolicyId.value, payload);
  } catch {
    // 错误文案由 store 写入并传给试算面板展示。
  }
};

const checkCurrent = async () => {
  if (!activeDraft.value) {
    return;
  }
  try {
    const result = await alarmStore.validateDraft(
      props.projectId,
      draftToAlarmPolicySavePayload(activeDraft.value),
    );
    if (result.valid) {
      ElMessage.success("策略检查通过");
    } else {
      ElMessage.warning("策略检查未通过");
    }
  } catch {
    ElMessage.error(alarmStore.validationError || "报警草稿校验失败");
  }
};

const updateActiveDraft = (patch: Partial<AlarmPolicyDraft>) => {
  if (activeDraft.value) {
    activeDraft.value = {
      ...activeDraft.value,
      ...patch,
      dirty: patch.dirty ?? true,
    };
  }
};

const validateActiveDraft = (requireReady = true) => {
  if (!activeDraft.value) {
    return false;
  }
  if (!activeDraft.value.name.trim()) {
    ElMessage.warning("请输入策略名");
    return false;
  }
  if (!requireReady) {
    return true;
  }
  if (
    activeDraft.value.mode === "per_target" &&
    activeDraft.value.targets.length === 0
  ) {
    ElMessage.warning("请选择目标点");
    return false;
  }
  if (activeDraft.value.mode === "derived") {
    if (activeDraft.value.inputs.length === 0) {
      ElMessage.warning("请选择输入点");
      return false;
    }
    if (!activeDraft.value.derivedExpression.trim()) {
      ElMessage.warning("请输入计算表达式");
      return false;
    }
  }
  if (!activeDraft.value.conditions.some((condition) => condition.isEnabled)) {
    ElMessage.warning("请至少启用一个报警条件");
    return false;
  }
  return true;
};

const saveActiveDraft = async () => {
  if (!activeDraft.value) {
    return;
  }
  if (!validateActiveDraft(activeDraft.value.isEnabled)) {
    return;
  }
  if (!selectedPolicyId.value) {
    return;
  }
  const policy = await alarmStore.savePolicy(
    props.projectId,
    selectedPolicyId.value,
    draftToAlarmPolicySavePayload(activeDraft.value),
  );
  activeDraft.value = toAlarmPolicyDraft(policy);
  await reloadList();
};

const toggleEnabled = async () => {
  if (!selectedPolicyId.value || !activeDraft.value) {
    return;
  }
  const enable = !activeDraft.value.isEnabled;
  if (enable) {
    if (!validateActiveDraft(true)) {
      return;
    }
    if (activeDraft.value.dirty) {
      const saved = await alarmStore.savePolicy(
        props.projectId,
        selectedPolicyId.value,
        draftToAlarmPolicySavePayload(activeDraft.value),
      );
      activeDraft.value = toAlarmPolicyDraft(saved);
    }
  }
  const policy = await alarmStore.setPolicyEnabled(
    props.projectId,
    selectedPolicyId.value,
    enable,
  );
  activeDraft.value = toAlarmPolicyDraft(policy);
  await reloadList();
};

const deletePolicy = async () => {
  if (!selectedPolicyId.value) {
    return;
  }
  await confirmAndDeletePolicy(selectedPolicyId.value, activeDraft.value?.name);
};

const deletePolicyFromTree = async (policy: AlarmPolicy) => {
  await confirmAndDeletePolicy(policy.id, policy.name);
};

const confirmAndDeletePolicy = async (policyId: string, name?: string) => {
  const ok = await ElMessageBox.confirm(
    `确认删除报警策略「${name || policyId}」？此操作不可恢复。`,
    "删除报警策略",
    {
      confirmButtonText: "删除",
      cancelButtonText: "取消",
      type: "warning",
    },
  )
    .then(() => true)
    .catch(() => false);
  if (!ok) {
    return;
  }
  await alarmStore.removePolicy(props.projectId, policyId);
  await reloadList();
  if (selectedPolicyId.value === policyId) {
    activeDraft.value = null;
    replaceAlarmRoute(alarmStore.tree.policies[0]?.id);
  }
  ElMessage.success("报警策略已删除");
};

const deleteGroupFromTree = async (group: AlarmPolicyGroupNode) => {
  const childGroupCount = countAlarmChildGroups(group);
  const policyCount = group.policyIds.length;
  const detail =
    childGroupCount > 0 || policyCount > 0
      ? `该分组包含 ${childGroupCount} 个子分组、${policyCount} 条报警策略。确认后会一起删除。`
      : "该分组为空。确认后会删除该分组。";
  const ok = await ElMessageBox.confirm(
    `确认删除分组「${group.name}」？${detail}此操作不可恢复。`,
    "删除分组",
    {
      confirmButtonText: "删除",
      cancelButtonText: "取消",
      type: "warning",
    },
  )
    .then(() => true)
    .catch(() => false);
  if (!ok) {
    return;
  }
  const removed = new Set(group.policyIds);
  await alarmStore.removeGroup(props.projectId, group.id);
  await reloadList();
  if (removed.has(selectedPolicyId.value)) {
    activeDraft.value = null;
    replaceAlarmRoute(alarmStore.tree.policies[0]?.id);
  }
  ElMessage.success("分组已删除");
};

const countAlarmChildGroups = (group: AlarmPolicyGroupNode): number =>
  group.children.reduce(
    (total, child) => total + 1 + countAlarmChildGroups(child),
    0,
  );

const selectPolicyForBatch = (id: string, selected: boolean) => {
  alarmStore.selectPolicy(id, selected);
};

const selectGroupForBatch = (ids: string[], selected: boolean) => {
  alarmStore.selectGroup(ids, selected);
};

const selectFilteredForBatch = (filters: Record<string, string>) => {
  alarmStore.selectFiltered(filters);
};

const batchEnable = async () => {
  await alarmStore.batchEnable(props.projectId);
  await reloadList();
};

const batchDisable = async () => {
  await alarmStore.batchDisable(props.projectId);
  await reloadList();
};

const openBulkConditionDialog = () => {
  bulkConditionDialogVisible.value = true;
};

const openBulkMoveDialog = () => {
  bulkMoveDialogVisible.value = true;
};

const batchApplyConditions = async (conditions: AlarmCondition[]) => {
  await alarmStore.batchApplyConditions(props.projectId, conditions);
  bulkConditionDialogVisible.value = false;
  await reloadList();
};

const batchMove = async (groupId: string | null) => {
  await alarmStore.batchMove(props.projectId, groupId);
  bulkMoveDialogVisible.value = false;
  await reloadList();
};

const batchDelete = async () => {
  const ids = alarmStore.selectedPolicyIds;
  if (!ids.length) {
    return;
  }
  const ok = await ElMessageBox.confirm(
    `确认删除已选 ${ids.length} 条报警策略？此操作不可恢复。`,
    "批量删除报警策略",
    {
      confirmButtonText: "删除",
      cancelButtonText: "取消",
      type: "warning",
    },
  )
    .then(() => true)
    .catch(() => false);
  if (!ok) {
    return;
  }
  await alarmStore.batchDelete(props.projectId);
  await reloadList();
  if (selectedPolicyId.value && !alarmStore.tree.policies.some((item) => item.id === selectedPolicyId.value)) {
    activeDraft.value = null;
    replaceAlarmRoute(alarmStore.tree.policies[0]?.id);
  }
  ElMessage.success("已删除已选报警策略");
};

const openRenamePolicyDialog = (policy: AlarmPolicy) => {
  contextPolicy.value = policy;
  renamePolicyDialogVisible.value = true;
};

const openMovePolicyDialog = (policy: AlarmPolicy) => {
  contextPolicy.value = policy;
  movePolicyDialogVisible.value = true;
};

const openRenameGroupDialog = (group: AlarmPolicyGroup) => {
  contextGroup.value = group;
  renameGroupDialogVisible.value = true;
};

const openMoveGroupDialog = (group: AlarmPolicyGroup) => {
  contextGroup.value = group;
  moveGroupDialogVisible.value = true;
};

const renamePolicy = async (name: string) => {
  if (!contextPolicy.value) {
    return;
  }
  const policyId = contextPolicy.value.id;
  await alarmStore.savePolicy(props.projectId, policyId, { name });
  if (selectedPolicyId.value === policyId && activeDraft.value) {
    activeDraft.value = { ...activeDraft.value, name, dirty: false };
  }
  renamePolicyDialogVisible.value = false;
  await reloadList();
};

const movePolicy = async (groupId: string | null) => {
  if (!contextPolicy.value) {
    return;
  }
  const policyId = contextPolicy.value.id;
  const policy = await alarmStore.savePolicy(props.projectId, policyId, {
    groupId,
  });
  if (selectedPolicyId.value === policyId) {
    activeDraft.value = toAlarmPolicyDraft(policy);
  }
  movePolicyDialogVisible.value = false;
  await reloadList();
};

const renameGroup = async (name: string) => {
  if (!contextGroup.value) {
    return;
  }
  await alarmStore.updateGroup(props.projectId, contextGroup.value.id, {
    name,
  });
  renameGroupDialogVisible.value = false;
  await reloadList();
};

const moveGroup = async (parentId: string | null) => {
  if (!contextGroup.value) {
    return;
  }
  await alarmStore.updateGroup(props.projectId, contextGroup.value.id, {
    parentId,
  });
  moveGroupDialogVisible.value = false;
  await reloadList();
};

watch(
  () => props.projectId,
  async () => {
    activeDraft.value = null;
    alarmStore.closeEdit();
    await reloadList();
  },
);

watch(
  selectedPolicyId,
  async (policyId) => {
    if (!policyId) {
      activeDraft.value = null;
      alarmStore.closeEdit();
      return;
    }
    const policy = await alarmStore.openPolicy(props.projectId, policyId);
    if (selectedPolicyId.value !== policyId) {
      return;
    }
    activeDraft.value = toAlarmPolicyDraft(policy);
  },
  { immediate: true },
);

watch(
  [activeTab, selectedPolicyId],
  ([tab, policyId]) => {
    if (tab === "contract" && policyId) {
      void refreshContract();
    }
  },
  { immediate: true },
);

onMounted(() => {
  reloadList();
});
</script>

<style scoped>
.alarm-workspace {
  height: 100%;
  min-height: 0;
  display: flex;
  overflow: hidden;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-subtle);
  box-shadow: var(--dc-shadow-surface);
  color: var(--dc-text);
}

.alarm-workspace :deep(.alarm-policy-manager) {
  flex: 0 0 260px;
}

.alarm-workspace :deep(.alarm-editor-shell) {
  min-width: 0;
  flex: 1 1 auto;
}

@media (max-width: 920px) {
  .alarm-workspace {
    flex-direction: column;
  }

  .alarm-workspace :deep(.alarm-policy-manager),
  .alarm-workspace :deep(.alarm-policy-manager) {
    flex: 0 0 auto;
  }
}
</style>
