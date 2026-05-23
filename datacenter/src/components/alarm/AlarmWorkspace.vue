<template>
  <div class="alarm-workspace">
    <AlarmPolicyManager
      :tree="alarmStore.tree"
      :selection="alarmStore.selection"
      :total="alarmStore.total"
      :selected-id="selectedPolicyId"
      :selected-count="alarmStore.selectedCount"
      :dirty-policy-ids="dirtyPolicyIds"
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
      @open-settings="openSettingsDialog"
    />

    <AlarmEditorShell
      :project-id="projectId"
      :file-tabs="editorTabs"
      :active-id="selectedPolicyId"
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
      :coverages="coverageByTargetId"
      :coverage-loading="alarmStore.coverage.loading"
      @update="updateActiveDraft"
      @save="saveActiveDraft"
      @toggle="toggleEnabled"
      @delete="deletePolicy"
      @activate-tab="activatePolicyTab"
      @close-tab="closePolicyTab"
      @select-tab="selectTab"
      @run-trial="runTrial"
      @refresh-contract="refreshContract"
      @check-current="checkCurrent"
      @select-target="openTargetPicker"
    />

    <DatapointPickerDialog
      v-model="targetPickerVisible"
      :project-id="projectId"
      title="数据点变量"
      confirm-text="选择"
      row-action-text="选择"
      @select="appendTargetDatapoint"
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

    <AlarmProjectSettingsDialog
      v-model="settingsDialogVisible"
      :settings="alarmStore.settings.data"
      :loading="alarmStore.settings.loading"
      :submitting="alarmStore.settings.saving"
      :error="alarmStore.settings.error"
      @submit="saveSettings"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { onBeforeRouteLeave, useRoute, useRouter } from "vue-router";
import { ElMessage, ElMessageBox } from "element-plus";
import type {
  AlarmCondition,
  AlarmPolicy,
  AlarmPolicyCoverage,
  AlarmPolicyGroup,
  AlarmPolicyGroupSave,
  AlarmPolicyTrialPayload,
} from "@/api/schemas/alarm.schema";
import type { Datapoint } from "@/api/schemas/datapoint.schema";
import { useAlarmStore } from "@/stores/alarm.store";
import DatapointPickerDialog from "@/components/shared/DatapointPickerDialog.vue";
import {
  createDefaultAlarmPolicyDraft,
  draftToAlarmPolicySavePayload,
  toAlarmPolicyDraft,
  type AlarmPolicyDraft,
} from "@/components/alarm/alarmPolicyModel";
import AlarmBulkConditionDialog from "./AlarmBulkConditionDialog.vue";
import AlarmEditorShell, { type AlarmEditorTab } from "./AlarmEditorShell.vue";
import AlarmPolicyManager from "./AlarmPolicyManager.vue";
import AlarmProjectSettingsDialog from "./AlarmProjectSettingsDialog.vue";
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
const drafts = ref<Record<string, AlarmPolicyDraft>>({});
const createDialogVisible = ref(false);
const createGroupDialogVisible = ref(false);
const bulkConditionDialogVisible = ref(false);
const bulkMoveDialogVisible = ref(false);
const renamePolicyDialogVisible = ref(false);
const movePolicyDialogVisible = ref(false);
const renameGroupDialogVisible = ref(false);
const moveGroupDialogVisible = ref(false);
const settingsDialogVisible = ref(false);
const targetPickerVisible = ref(false);
const listParams = ref<Record<string, string>>({});
const contextPolicy = ref<AlarmPolicy | null>(null);
const contextGroup = ref<AlarmPolicyGroup | null>(null);
const coverageByTargetId = ref<Record<string, AlarmPolicyCoverage>>({});

const selectedPolicyId = computed(() => {
  if (route.params.module !== "alarm") {
    return "";
  }
  const value = route.params.objectId;
  return typeof value === "string" ? value : "";
});

const editorTabs = computed(() =>
  Object.values(drafts.value).map((draft) => ({
    id: String(draft.id || ""),
    name: draft.name || "未命名报警策略",
    dirty: draft.dirty,
  })),
);

const activeDraft = computed(() => {
  if (!selectedPolicyId.value) {
    return null;
  }
  return drafts.value[selectedPolicyId.value] || null;
});

const activeTargetSignature = computed(() => {
  const draft = activeDraft.value;
  if (!draft || draft.mode !== "per_target") {
    return "";
  }
  return draft.targets
    .map((target) => `${target.datapointId || ""}:${target.path || ""}`)
    .join("|");
});

const hasDirtyTabs = computed(() =>
  Object.values(drafts.value).some((draft) => draft.dirty),
);
const dirtyPolicyIds = computed(() =>
  Object.values(drafts.value)
    .filter((draft) => draft.dirty)
    .map((draft) => String(draft.id || "")),
);

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

const openSettingsDialog = async () => {
  settingsDialogVisible.value = true;
  try {
    await alarmStore.fetchSettings(props.projectId);
  } catch {
    // 弹窗内会展示错误，这里不打断用户操作。
  }
};

const saveSettings = async (payload: {
  escalationIntervalSeconds: number;
  repeatNotificationIntervalSeconds: number;
}) => {
  try {
    await alarmStore.saveSettings(props.projectId, payload);
    settingsDialogVisible.value = false;
    ElMessage.success("报警设置已保存");
  } catch {
    // store 已记录错误，弹窗内展示。
  }
};

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
  drafts.value = {
    ...drafts.value,
    [policy.id]: toAlarmPolicyDraft(policy),
  };
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

const activatePolicyTab = (policyId: string) => {
  replaceAlarmRoute(policyId, activeTab.value);
};

const closePolicyTab = async (policyId: string) => {
  const draft = drafts.value[policyId];
  if (!draft) {
    return;
  }
  if (draft.dirty) {
    const action = await confirmDirtyClose(draft.name);
    if (action === "cancel") {
      return;
    }
    if (action === "save") {
      const saved = await savePolicyDraft(policyId);
      if (!saved) {
        return;
      }
    }
  }
  const nextDrafts = { ...drafts.value };
  delete nextDrafts[policyId];
  drafts.value = nextDrafts;
  if (selectedPolicyId.value === policyId) {
    const next = Object.keys(nextDrafts)[0] || "";
    replaceAlarmRoute(next || undefined, "config");
  }
};

async function confirmDirtyClose(name: string) {
  try {
    await ElMessageBox.confirm(
      `报警策略「${name}」有未保存修改。`,
      "关闭标签",
      {
        confirmButtonText: "保存",
        cancelButtonText: "丢弃",
        distinguishCancelAndClose: true,
        type: "warning",
        closeOnClickModal: false,
      },
    );
    return "save" as const;
  } catch (action) {
    if (action === "cancel") return "discard" as const;
    return "cancel" as const;
  }
}

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

const openTargetPicker = () => {
  if (!activeDraft.value || activeDraft.value.mode !== "per_target") {
    ElMessage.warning("当前模式不需要选择目标点");
    return;
  }
  targetPickerVisible.value = true;
};

const appendTargetDatapoint = (datapoint: Datapoint) => {
  const draft = activeDraft.value;
  if (!draft || draft.mode !== "per_target") {
    return;
  }
  if (draft.targets.some((target) => target.datapointId === String(datapoint.id))) {
    ElMessage.warning("该数据点已在目标点列表中");
    return;
  }
  updateActiveDraft({
    targets: [
      ...draft.targets,
      {
        datapointId: String(datapoint.id),
        path: datapoint.path,
        name: datapoint.name,
        dataType: datapoint.dataType || "",
      },
    ],
    dirty: true,
  });
};

const coverageKey = (target: { datapointId?: string; path?: string }) =>
  String(target.datapointId || target.path || "");

const hasCoverageConflicts = (draft: AlarmPolicyDraft) =>
  draft.mode === "per_target" &&
  draft.targets.some(
    (target) =>
      (coverageByTargetId.value[coverageKey(target)]?.policies.length || 0) >
      0,
  );

const confirmCoverageConflicts = async (draft: AlarmPolicyDraft) => {
  if (!hasCoverageConflicts(draft)) {
    return true;
  }
  return ElMessageBox.confirm(
    "部分目标点已被其他策略引用，保存后这些策略仍会同时生效。若要替代原策略，需要手动从原策略中移除目标点或停用原策略。",
    "目标点已被其他策略引用",
    {
      confirmButtonText: "继续保存",
      cancelButtonText: "取消",
      type: "warning",
      closeOnClickModal: false,
    },
  )
    .then(() => true)
    .catch(() => false);
};

const refreshTargetCoverages = async (policyId: string, draft: AlarmPolicyDraft) => {
  if (!policyId || draft.mode !== "per_target") {
    coverageByTargetId.value = {};
    return {};
  }
  const next: Record<string, AlarmPolicyCoverage> = {};
  for (const target of draft.targets) {
    const key = coverageKey(target);
    if (!key) {
      continue;
    }
    try {
      next[key] = await alarmStore.fetchCoverage(props.projectId, {
        datapointId: target.datapointId,
        path: target.path,
        excludePolicyId: policyId,
      });
    } catch {
      // 覆盖关系只用于编辑提醒，失败时不阻塞策略编辑。
    }
  }
  if (selectedPolicyId.value === policyId) {
    coverageByTargetId.value = next;
  }
  return next;
};

const updateActiveDraft = (patch: Partial<AlarmPolicyDraft>) => {
  const policyId = selectedPolicyId.value;
  const draft = policyId ? drafts.value[policyId] : null;
  if (draft) {
    drafts.value = {
      ...drafts.value,
      [policyId]: {
        ...draft,
        ...patch,
        dirty: patch.dirty ?? true,
      },
    };
  }
};

const setDraft = (policyId: string, draft: AlarmPolicyDraft) => {
  drafts.value = {
    ...drafts.value,
    [policyId]: draft,
  };
};

const validateDraft = (draft: AlarmPolicyDraft, requireReady = true) => {
  if (!draft) {
    return false;
  }
  if (!draft.name.trim()) {
    ElMessage.warning("请输入策略名");
    return false;
  }
  if (!requireReady) {
    return true;
  }
  if (
    draft.mode === "per_target" &&
    draft.targets.length === 0
  ) {
    ElMessage.warning("请选择目标点");
    return false;
  }
  if (draft.mode === "derived") {
    if (draft.inputs.length === 0) {
      ElMessage.warning("请选择输入点");
      return false;
    }
    if (!draft.derivedExpression.trim()) {
      ElMessage.warning("请输入计算表达式");
      return false;
    }
  }
  if (!draft.conditions.some((condition) => condition.isEnabled)) {
    ElMessage.warning("请至少启用一个报警条件");
    return false;
  }
  return true;
};

const savePolicyDraft = async (policyId: string) => {
  const draft = drafts.value[policyId];
  if (!draft) {
    return false;
  }
  if (!validateDraft(draft, draft.isEnabled)) {
    return false;
  }
  await refreshTargetCoverages(policyId, draft);
  if (!(await confirmCoverageConflicts(draft))) {
    return false;
  }
  const policy = await alarmStore.savePolicy(
    props.projectId,
    policyId,
    draftToAlarmPolicySavePayload(draft),
  );
  setDraft(policyId, toAlarmPolicyDraft(policy));
  await reloadList();
  return true;
};

const saveActiveDraft = async () => {
  if (!selectedPolicyId.value) {
    return;
  }
  await savePolicyDraft(selectedPolicyId.value);
};

const toggleEnabled = async () => {
  const policyId = selectedPolicyId.value;
  const draft = policyId ? drafts.value[policyId] : null;
  if (!policyId || !draft) {
    return;
  }
  const enable = !draft.isEnabled;
  if (enable) {
    if (!validateDraft(draft, true)) {
      return;
    }
    if (draft.dirty) {
      const saved = await alarmStore.savePolicy(
        props.projectId,
        policyId,
        draftToAlarmPolicySavePayload(draft),
      );
      setDraft(policyId, toAlarmPolicyDraft(saved));
    }
  }
  const policy = await alarmStore.setPolicyEnabled(
    props.projectId,
    policyId,
    enable,
  );
  setDraft(policyId, toAlarmPolicyDraft(policy));
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
  const nextDrafts = { ...drafts.value };
  delete nextDrafts[policyId];
  drafts.value = nextDrafts;
  if (selectedPolicyId.value === policyId) {
    replaceAlarmRoute(Object.keys(nextDrafts)[0] || alarmStore.tree.policies[0]?.id);
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
  if (removed.size) {
    const nextDrafts = { ...drafts.value };
    for (const policyId of removed) {
      delete nextDrafts[policyId];
    }
    drafts.value = nextDrafts;
  }
  if (removed.has(selectedPolicyId.value)) {
    replaceAlarmRoute(Object.keys(drafts.value)[0] || alarmStore.tree.policies[0]?.id);
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
    const nextDrafts = { ...drafts.value };
    for (const policyId of ids) {
      delete nextDrafts[policyId];
    }
    drafts.value = nextDrafts;
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
  const draft = drafts.value[policyId];
  if (draft) {
    setDraft(policyId, { ...draft, name, dirty: false });
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
  setDraft(policyId, toAlarmPolicyDraft(policy));
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
    drafts.value = {};
    alarmStore.closeEdit();
    await reloadList();
  },
);

watch(
  selectedPolicyId,
  async (policyId) => {
    if (!policyId) {
      alarmStore.closeEdit();
      return;
    }
    if (drafts.value[policyId]) {
      return;
    }
    const policy = await alarmStore.openPolicy(props.projectId, policyId);
    if (selectedPolicyId.value !== policyId) {
      return;
    }
    setDraft(policyId, toAlarmPolicyDraft(policy));
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

watch(
  [selectedPolicyId, activeTargetSignature],
  async ([policyId]) => {
    const draft = activeDraft.value;
    if (!policyId || !draft || draft.mode !== "per_target") {
      coverageByTargetId.value = {};
      return;
    }
    await refreshTargetCoverages(policyId, draft);
  },
  { immediate: true },
);

function handleBeforeUnload(event: BeforeUnloadEvent) {
  if (!hasDirtyTabs.value) return;
  event.preventDefault();
  event.returnValue = "";
}

onBeforeRouteLeave(async () => {
  if (!hasDirtyTabs.value) return true;
  return ElMessageBox.confirm(
    "当前存在未保存的报警策略修改，离开后这些修改不会保存。",
    "离开报警单元",
    {
      confirmButtonText: "离开",
      cancelButtonText: "取消",
      type: "warning",
    },
  )
    .then(() => true)
    .catch(() => false);
});

onMounted(() => {
  reloadList();
  window.addEventListener("beforeunload", handleBeforeUnload);
});

onBeforeUnmount(() => {
  window.removeEventListener("beforeunload", handleBeforeUnload);
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
