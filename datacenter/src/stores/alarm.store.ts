import { computed, ref } from "vue";
import { defineStore } from "pinia";
import {
  batchApplyAlarmConditions,
  batchDisableAlarmPolicies,
  batchEnableAlarmPolicies,
  batchMoveAlarmPolicies,
  createAlarmPolicyGroup,
  createAlarmPolicy,
  deleteAlarmPolicy,
  deleteAlarmPolicyGroup,
  getAlarmPolicy,
  getAlarmPolicyContract,
  getAlarmPolicyCoverage,
  getAlarmPolicyGroups,
  getAlarmPolicyTree,
  getAlarmPolicies,
  getAlarmProjectSettings,
  testAlarmPolicy,
  toggleAlarmPolicy,
  updateAlarmPolicy,
  updateAlarmPolicyGroup,
  updateAlarmProjectSettings,
  validateAlarmPolicyDraft,
} from "@/api/alarm.api";
import type {
  AlarmBulkSelection,
  AlarmCondition,
  AlarmDraftValidation,
  AlarmPolicy,
  AlarmPolicyContract,
  AlarmPolicyCoverage,
  AlarmPolicyGroup,
  AlarmPolicyGroupSave,
  AlarmProjectSettings,
  AlarmProjectSettingsSave,
  AlarmPolicySave,
  AlarmPolicyTree,
  AlarmPolicyTrialPayload,
  AlarmPolicyTrialResult,
  AlarmPolicyUpdate,
} from "@/api/schemas/alarm.schema";
import { getApiErrorMessage } from "@/utils/request";

type TrialState = {
  policyId: string;
  payload: AlarmPolicyTrialPayload;
  result: AlarmPolicyTrialResult | null;
  running: boolean;
  error: string;
};

type ContractState = {
  policyId: string;
  data: AlarmPolicyContract | null;
  loading: boolean;
  error: string;
};

type CoverageState = {
  key: string;
  data: AlarmPolicyCoverage | null;
  loading: boolean;
  error: string;
};

type SettingsState = {
  data: AlarmProjectSettings | null;
  loading: boolean;
  saving: boolean;
  error: string;
};

const emptyTree = (): AlarmPolicyTree => ({
  groups: [],
  rootPolicies: [],
  policies: [],
  matchedPolicyCount: 0,
  totalPolicyCount: 0,
});

export const useAlarmStore = defineStore("alarm", () => {
  const groups = ref<AlarmPolicyGroup[]>([]);
  const tree = ref<AlarmPolicyTree>(emptyTree());
  const list = ref<AlarmPolicy[]>([]);
  const total = ref(0);
  const loading = ref(false);
  const listError = ref("");
  const selection = ref<AlarmBulkSelection>({ mode: "ids", policyIds: [] });

  const editing = ref<AlarmPolicy | null>(null);
  const detailLoading = ref(false);
  const detailError = ref("");

  const creating = ref(false);
  const createError = ref("");
  const saving = ref(false);
  const deleting = ref(false);

  const trial = ref<TrialState>({
    policyId: "",
    payload: { context: {} },
    result: null,
    running: false,
    error: "",
  });
  const contract = ref<ContractState>({
    policyId: "",
    data: null,
    loading: false,
    error: "",
  });
  const validation = ref<AlarmDraftValidation | null>(null);
  const validationLoading = ref(false);
  const validationError = ref("");
  const coverage = ref<CoverageState>({
    key: "",
    data: null,
    loading: false,
    error: "",
  });
  const settings = ref<SettingsState>({
    data: null,
    loading: false,
    saving: false,
    error: "",
  });

  const hasEditing = computed(() => editing.value !== null);
  const selectedCount = computed(() => {
    if (selection.value.mode === "ids") {
      return selection.value.policyIds.length;
    }
    return Math.max(
      0,
      tree.value.matchedPolicyCount - selection.value.excludePolicyIds.length,
    );
  });
  const selectedPolicyIds = computed(() => {
    if (selection.value.mode === "ids") {
      return selection.value.policyIds;
    }
    const excluded = new Set(selection.value.excludePolicyIds);
    return tree.value.policies
      .filter((policy) => !excluded.has(policy.id))
      .map((policy) => policy.id);
  });

  const mergePolicyIntoList = (policy: AlarmPolicy) => {
    const exists = list.value.some((item) => item.id === policy.id);
    list.value = exists
      ? list.value.map((item) =>
          item.id === policy.id ? { ...item, ...policy } : item,
        )
      : [policy, ...list.value];
    tree.value = {
      ...tree.value,
      policies: tree.value.policies.map((item) =>
        item.id === policy.id ? { ...item, ...policy } : item,
      ),
      rootPolicies: tree.value.rootPolicies.map((item) =>
        item.id === policy.id ? { ...item, ...policy } : item,
      ),
    };
    total.value = Math.max(total.value, list.value.length);
  };

  async function fetchGroups(projectId: string) {
    groups.value = await getAlarmPolicyGroups(projectId);
    return groups.value;
  }

  async function fetchSettings(projectId: string) {
    settings.value = { ...settings.value, loading: true, error: "" };
    try {
      const data = await getAlarmProjectSettings(projectId);
      settings.value = { data, loading: false, saving: false, error: "" };
      return data;
    } catch (error) {
      settings.value = {
        ...settings.value,
        loading: false,
        error: getApiErrorMessage(error, "报警设置不可用"),
      };
      throw error;
    }
  }

  async function saveSettings(
    projectId: string,
    data: AlarmProjectSettingsSave,
  ) {
    settings.value = { ...settings.value, saving: true, error: "" };
    try {
      const saved = await updateAlarmProjectSettings(projectId, data);
      settings.value = { data: saved, loading: false, saving: false, error: "" };
      return saved;
    } catch (error) {
      settings.value = {
        ...settings.value,
        saving: false,
        error: getApiErrorMessage(error, "保存报警设置失败"),
      };
      throw error;
    }
  }

  async function createGroup(projectId: string, data: AlarmPolicyGroupSave) {
    const group = await createAlarmPolicyGroup(projectId, data);
    groups.value = [...groups.value, group];
    tree.value = { ...tree.value, groups: groups.value };
    return group;
  }

  async function updateGroup(
    projectId: string,
    groupId: string,
    data: Partial<AlarmPolicyGroupSave>,
  ) {
    saving.value = true;
    try {
      const group = await updateAlarmPolicyGroup(projectId, groupId, data);
      groups.value = groups.value.map((item) =>
        item.id === group.id ? group : item,
      );
      tree.value = {
        ...tree.value,
        groups: tree.value.groups.map((item) =>
          item.id === group.id ? group : item,
        ),
      };
      return group;
    } finally {
      saving.value = false;
    }
  }

  async function fetchTree(
    projectId: string,
    params: Record<string, unknown> = {},
  ) {
    loading.value = true;
    listError.value = "";
    try {
      tree.value = await getAlarmPolicyTree(projectId, params);
      groups.value = tree.value.groups;
      list.value = tree.value.policies.length
        ? tree.value.policies
        : tree.value.rootPolicies;
      total.value = tree.value.matchedPolicyCount;
      return tree.value;
    } catch (error) {
      tree.value = emptyTree();
      list.value = [];
      total.value = 0;
      listError.value = getApiErrorMessage(error, "报警策略树不可用");
      throw error;
    } finally {
      loading.value = false;
    }
  }

  async function fetchList(
    projectId: string,
    params: Record<string, unknown> = {},
  ) {
    loading.value = true;
    listError.value = "";
    try {
      const res = await getAlarmPolicies(projectId, params);
      list.value = res.list;
      total.value = res.pagination?.total ?? res.list.length;
    } catch (error) {
      list.value = [];
      total.value = 0;
      listError.value = getApiErrorMessage(error, "报警策略列表不可用");
      throw error;
    } finally {
      loading.value = false;
    }
  }

  async function openPolicy(projectId: string, id: string) {
    detailLoading.value = true;
    detailError.value = "";
    try {
      editing.value = await getAlarmPolicy(projectId, id);
      return editing.value;
    } catch (error) {
      editing.value = null;
      detailError.value = getApiErrorMessage(error, "报警策略详情不可用");
      throw error;
    } finally {
      detailLoading.value = false;
    }
  }

  function closeEdit() {
    editing.value = null;
    detailError.value = "";
  }

  async function createPolicy(projectId: string, data: AlarmPolicySave) {
    creating.value = true;
    createError.value = "";
    try {
      const policy = await createAlarmPolicy(projectId, data);
      mergePolicyIntoList(policy);
      editing.value = policy;
      return policy;
    } catch (error) {
      createError.value = getApiErrorMessage(error, "新建报警策略失败");
      throw error;
    } finally {
      creating.value = false;
    }
  }

  async function savePolicy(
    projectId: string,
    id: string,
    data: AlarmPolicyUpdate,
  ) {
    saving.value = true;
    try {
      const policy = await updateAlarmPolicy(projectId, id, data);
      editing.value = policy;
      mergePolicyIntoList(policy);
      return policy;
    } finally {
      saving.value = false;
    }
  }

  async function removePolicy(projectId: string, id: string) {
    deleting.value = true;
    try {
      await deleteAlarmPolicy(projectId, id);
      list.value = list.value.filter((item) => item.id !== id);
      tree.value = {
        ...tree.value,
        policies: tree.value.policies.filter((item) => item.id !== id),
        rootPolicies: tree.value.rootPolicies.filter((item) => item.id !== id),
        matchedPolicyCount: Math.max(0, tree.value.matchedPolicyCount - 1),
      };
      total.value = Math.max(0, total.value - 1);
      if (editing.value?.id === id) {
        closeEdit();
      }
    } finally {
      deleting.value = false;
    }
  }

  async function removeGroup(projectId: string, groupId: string) {
    deleting.value = true;
    try {
      await deleteAlarmPolicyGroup(projectId, groupId);
      await fetchTree(projectId);
    } finally {
      deleting.value = false;
    }
  }

  async function setPolicyEnabled(
    projectId: string,
    id: string,
    isEnabled: boolean,
  ) {
    saving.value = true;
    try {
      const policy = await toggleAlarmPolicy(projectId, id, isEnabled);
      editing.value = policy;
      mergePolicyIntoList(policy);
      return policy;
    } finally {
      saving.value = false;
    }
  }

  function selectPolicy(id: string, selected: boolean) {
    if (selection.value.mode === "filtered") {
      const excluded = new Set(selection.value.excludePolicyIds);
      if (selected) {
        excluded.delete(id);
      } else {
        excluded.add(id);
      }
      selection.value = {
        ...selection.value,
        excludePolicyIds: [...excluded],
      };
      return;
    }

    const ids = new Set(selection.value.policyIds);
    if (selected) {
      ids.add(id);
    } else {
      ids.delete(id);
    }
    selection.value = { mode: "ids", policyIds: [...ids] };
  }

  function selectGroup(policyIds: string[], selected: boolean) {
    for (const id of policyIds) {
      selectPolicy(id, selected);
    }
  }

  function selectFiltered(filters: Record<string, unknown>) {
    selection.value = { mode: "filtered", filters, excludePolicyIds: [] };
  }

  function clearSelection() {
    selection.value = { mode: "ids", policyIds: [] };
  }

  async function batchEnable(projectId: string) {
    saving.value = true;
    try {
      await batchEnableAlarmPolicies(projectId, selection.value);
    } finally {
      saving.value = false;
    }
  }

  async function batchDisable(projectId: string) {
    saving.value = true;
    try {
      await batchDisableAlarmPolicies(projectId, selection.value);
    } finally {
      saving.value = false;
    }
  }

  async function batchMove(projectId: string, groupId: string | null) {
    saving.value = true;
    try {
      await batchMoveAlarmPolicies(projectId, selection.value, groupId);
    } finally {
      saving.value = false;
    }
  }

  async function batchMovePolicies(
    projectId: string,
    policyIds: string[],
    groupId: string | null,
  ) {
    saving.value = true;
    try {
      await batchMoveAlarmPolicies(
        projectId,
        { mode: "ids", policyIds },
        groupId,
      );
    } finally {
      saving.value = false;
    }
  }

  async function batchDelete(projectId: string) {
    const ids = selectedPolicyIds.value;
    if (!ids.length) {
      return;
    }
    deleting.value = true;
    try {
      await Promise.all(ids.map((id) => deleteAlarmPolicy(projectId, id)));
      clearSelection();
    } finally {
      deleting.value = false;
    }
  }

  async function batchApplyConditions(
    projectId: string,
    conditions: AlarmCondition[],
  ) {
    saving.value = true;
    try {
      await batchApplyAlarmConditions(projectId, selection.value, conditions);
    } finally {
      saving.value = false;
    }
  }

  async function runTrial(
    projectId: string,
    id: string,
    payload: AlarmPolicyTrialPayload = { context: {} },
  ) {
    trial.value = {
      policyId: id,
      payload,
      result: null,
      running: true,
      error: "",
    };
    try {
      const result = await testAlarmPolicy(projectId, id, payload);
      trial.value.result = result;
      return result;
    } catch (error) {
      trial.value.error = getApiErrorMessage(error, "报警策略试算失败");
      throw error;
    } finally {
      trial.value.running = false;
    }
  }

  function clearTrial() {
    trial.value = {
      policyId: "",
      payload: { context: {} },
      result: null,
      running: false,
      error: "",
    };
  }

  async function fetchContract(projectId: string, id: string) {
    contract.value = {
      policyId: id,
      data: null,
      loading: true,
      error: "",
    };
    try {
      const data = await getAlarmPolicyContract(projectId, id);
      contract.value.data = data;
      return data;
    } catch (error) {
      contract.value.error = getApiErrorMessage(error, "报警策略契约不可用");
      throw error;
    } finally {
      contract.value.loading = false;
    }
  }

  async function validateDraft(projectId: string, payload: AlarmPolicySave) {
    validationLoading.value = true;
    validationError.value = "";
    try {
      validation.value = await validateAlarmPolicyDraft(projectId, payload);
      return validation.value;
    } catch (error) {
      validation.value = null;
      validationError.value = getApiErrorMessage(error, "报警策略草稿校验失败");
      throw error;
    } finally {
      validationLoading.value = false;
    }
  }

  async function fetchCoverage(
    projectId: string,
    params: {
      datapointId?: string;
      path?: string;
      excludePolicyId?: string;
    },
  ) {
    const key = JSON.stringify(params);
    coverage.value = { key, data: null, loading: true, error: "" };
    try {
      const data = await getAlarmPolicyCoverage(projectId, params);
      coverage.value = { key, data, loading: false, error: "" };
      return data;
    } catch (error) {
      coverage.value = {
        key,
        data: null,
        loading: false,
        error: getApiErrorMessage(error, "报警策略覆盖关系不可用"),
      };
      throw error;
    }
  }

  return {
    groups,
    tree,
    list,
    total,
    loading,
    listError,
    selection,
    selectedCount,
    editing,
    detailLoading,
    detailError,
    creating,
    createError,
    saving,
    deleting,
    trial,
    contract,
    validation,
    validationLoading,
    validationError,
    coverage,
    settings,
    fetchSettings,
    saveSettings,
    hasEditing,
    selectedPolicyIds,
    fetchGroups,
    createGroup,
    updateGroup,
    fetchTree,
    fetchList,
    openPolicy,
    openForEdit: openPolicy,
    closeEdit,
    createPolicy,
    createRule: createPolicy,
    savePolicy,
    saveRule: savePolicy,
    removePolicy,
    removeRule: removePolicy,
    removeGroup,
    setPolicyEnabled,
    setRuleEnabled: setPolicyEnabled,
    selectPolicy,
    selectGroup,
    selectFiltered,
    clearSelection,
    batchEnable,
    batchDisable,
    batchMove,
    batchDelete,
    batchMovePolicies,
    batchApplyConditions,
    runTrial,
    clearTrial,
    fetchContract,
    validateDraft,
    fetchCoverage,
  };
});
