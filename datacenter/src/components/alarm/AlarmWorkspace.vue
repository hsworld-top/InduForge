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
      @rename-policy="renamePolicy"
      @move-policy="movePolicy"
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
      :submitting="alarmStore.saving"
      @submit="createGroup"
    />

    <AlarmBulkConditionDialog
      v-model="bulkConditionDialogVisible"
      :submitting="alarmStore.saving"
      @submit="batchApplyConditions"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { ElMessage } from "element-plus";
import type {
  AlarmCondition,
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
import AlarmEditorShell, {
  type AlarmEditorTab,
} from "./AlarmEditorShell.vue";
import AlarmPolicyManager from "./AlarmPolicyManager.vue";
import CreateAlarmGroupDialog from "./CreateAlarmGroupDialog.vue";
import CreateAlarmPolicyDialog from "./CreateAlarmPolicyDialog.vue";

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
const listParams = ref<Record<string, string>>({});

const selectedPolicyId = computed(() => {
  const value = route.params.objectId;
  return typeof value === "string" ? value : "";
});

const isCreatingDraft = computed(() => selectedPolicyId.value === "__new__");

const activeTab = computed<WorkspaceTab>(() => {
  const value = route.params.tab;
  return tabs.includes(value as WorkspaceTab) ? (value as WorkspaceTab) : "config";
});

const routeBase = computed(() => (route.path.startsWith("/debug/") ? "/debug" : ""));
const replaceAlarmRoute = (policyId?: string, tab: WorkspaceTab = "config") => {
  const tabPath = tab === "config" ? "" : `/${tab}`;
  const policyPath = policyId ? `/${policyId}${tabPath}` : "";
  router.push(`${routeBase.value}/alarm${policyPath}`);
};

const cleanParams = (params: Record<string, string>) =>
  Object.fromEntries(
    Object.entries(params).filter(([, value]) => value.trim() !== ""),
  );

const reloadList = () => alarmStore.fetchTree(props.projectId, listParams.value);

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
  activeDraft.value = {
    ...createDefaultAlarmPolicyDraft(),
    ...draft,
    id: undefined,
    dirty: true,
  };
  createDialogVisible.value = false;
  replaceAlarmRoute("__new__", "config");
};

const selectPolicy = (policyId: string) => {
  replaceAlarmRoute(policyId, "config");
};

const selectTab = (tab: WorkspaceTab) => {
  replaceAlarmRoute(selectedPolicyId.value, tab);
};

const refreshContract = async () => {
  if (!selectedPolicyId.value || isCreatingDraft.value) {
    return;
  }
  try {
    await alarmStore.fetchContract(props.projectId, selectedPolicyId.value);
  } catch {
    // 错误文案由 store 写入并传给契约面板展示。
  }
};

const runTrial = async (payload: AlarmPolicyTrialPayload) => {
  if (!selectedPolicyId.value || isCreatingDraft.value) {
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

const validateActiveDraft = () => {
  if (!activeDraft.value) {
    return false;
  }
  if (!activeDraft.value.name.trim()) {
    ElMessage.warning("请输入策略名");
    return false;
  }
  if (activeDraft.value.mode === "per_target" && activeDraft.value.targets.length === 0) {
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
  if (!validateActiveDraft()) {
    return;
  }
  if (isCreatingDraft.value) {
    const policy = await alarmStore.createPolicy(
      props.projectId,
      draftToAlarmPolicySavePayload(activeDraft.value),
    );
    activeDraft.value = toAlarmPolicyDraft(policy);
    await reloadList();
    replaceAlarmRoute(policy.id, "config");
    ElMessage.success("报警策略已创建");
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
  if (isCreatingDraft.value) {
    updateActiveDraft({ isEnabled: !activeDraft.value.isEnabled });
    return;
  }
  const policy = await alarmStore.setPolicyEnabled(
    props.projectId,
    selectedPolicyId.value,
    !activeDraft.value.isEnabled,
  );
  activeDraft.value = toAlarmPolicyDraft(policy);
  await reloadList();
};

const deletePolicy = async () => {
  if (isCreatingDraft.value) {
    activeDraft.value = null;
    replaceAlarmRoute();
    return;
  }
  if (!selectedPolicyId.value || !window.confirm("确认删除当前报警策略？")) {
    return;
  }
  await alarmStore.removePolicy(props.projectId, selectedPolicyId.value);
  await reloadList();
  replaceAlarmRoute(alarmStore.tree.policies[0]?.id);
};

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

const batchApplyConditions = async (conditions: AlarmCondition[]) => {
  await alarmStore.batchApplyConditions(props.projectId, conditions);
  bulkConditionDialogVisible.value = false;
  await reloadList();
};

const batchMove = async (groupId: string | null) => {
  await alarmStore.batchMove(props.projectId, groupId);
  await reloadList();
};

const renamePolicy = async (policyId: string, name: string) => {
  await alarmStore.savePolicy(props.projectId, policyId, { name });
  if (selectedPolicyId.value === policyId && activeDraft.value) {
    activeDraft.value = { ...activeDraft.value, name, dirty: false };
  }
  await reloadList();
};

const movePolicy = async (policyId: string, groupId: string | null) => {
  const policy = await alarmStore.savePolicy(props.projectId, policyId, { groupId });
  if (selectedPolicyId.value === policyId) {
    activeDraft.value = toAlarmPolicyDraft(policy);
  }
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
    if (policyId === "__new__") {
      return;
    }
    const policy = await alarmStore.openPolicy(props.projectId, policyId);
    activeDraft.value = toAlarmPolicyDraft(policy);
  },
  { immediate: true },
);

watch(
  [activeTab, selectedPolicyId],
  ([tab, policyId]) => {
    if (tab === "contract" && policyId && policyId !== "__new__") {
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
  display: grid;
  grid-template-columns: 400px minmax(0, 1fr);
  background: var(--dc-surface-subtle);
  color: var(--dc-text);
}

@media (max-width: 920px) {
  .alarm-workspace {
    grid-template-columns: 1fr;
  }
}
</style>
