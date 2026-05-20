import { computed, ref } from "vue";
import { defineStore } from "pinia";
import {
  createAlarmRule,
  deleteAlarmRule,
  getAlarmRule,
  getAlarmRuleContract,
  getAlarmRules,
  testAlarmRule,
  toggleAlarmRule,
  updateAlarmRule,
  validateAlarmRuleDraft,
} from "@/api/alarm.api";
import type {
  AlarmContract,
  AlarmDraftValidation,
  AlarmRule,
  AlarmRuleSave,
  AlarmRuleUpdate,
  AlarmTrialPayload,
  AlarmTrialResult,
} from "@/api/schemas/alarm.schema";
import { getApiErrorMessage } from "@/utils/request";

type TrialState = {
  ruleId: string;
  payload: AlarmTrialPayload;
  result: AlarmTrialResult | null;
  running: boolean;
  error: string;
};

type ContractState = {
  ruleId: string;
  data: AlarmContract | null;
  loading: boolean;
  error: string;
};

export const useAlarmStore = defineStore("alarm", () => {
  const list = ref<AlarmRule[]>([]);
  const total = ref(0);
  const loading = ref(false);
  const listError = ref("");

  const editing = ref<AlarmRule | null>(null);
  const detailLoading = ref(false);
  const detailError = ref("");

  const creating = ref(false);
  const createError = ref("");
  const saving = ref(false);
  const deleting = ref(false);

  const trial = ref<TrialState>({
    ruleId: "",
    payload: {},
    result: null,
    running: false,
    error: "",
  });
  const contract = ref<ContractState>({
    ruleId: "",
    data: null,
    loading: false,
    error: "",
  });
  const validation = ref<AlarmDraftValidation | null>(null);
  const validationLoading = ref(false);
  const validationError = ref("");

  const hasEditing = computed(() => editing.value !== null);

  const mergeRuleIntoList = (rule: AlarmRule) => {
    const exists = list.value.some((item) => item.id === rule.id);
    list.value = exists
      ? list.value.map((item) => (item.id === rule.id ? { ...item, ...rule } : item))
      : [rule, ...list.value];
    total.value = Math.max(total.value, list.value.length);
  };

  async function fetchList(projectId: string, params: Record<string, unknown> = {}) {
    loading.value = true;
    listError.value = "";
    try {
      const res = await getAlarmRules(projectId, params);
      list.value = res.list;
      total.value = res.pagination?.total ?? res.list.length;
    } catch (error) {
      list.value = [];
      total.value = 0;
      listError.value = getApiErrorMessage(error, "报警规则列表不可用");
      throw error;
    } finally {
      loading.value = false;
    }
  }

  async function openForEdit(projectId: string, id: string) {
    detailLoading.value = true;
    detailError.value = "";
    try {
      editing.value = await getAlarmRule(projectId, id);
      return editing.value;
    } catch (error) {
      editing.value = null;
      detailError.value = getApiErrorMessage(error, "报警规则详情不可用");
      throw error;
    } finally {
      detailLoading.value = false;
    }
  }

  function closeEdit() {
    editing.value = null;
    detailError.value = "";
  }

  async function createRule(projectId: string, data: AlarmRuleSave) {
    creating.value = true;
    createError.value = "";
    try {
      const rule = await createAlarmRule(projectId, data);
      mergeRuleIntoList(rule);
      editing.value = rule;
      return rule;
    } catch (error) {
      createError.value = getApiErrorMessage(error, "新建报警规则失败");
      throw error;
    } finally {
      creating.value = false;
    }
  }

  async function saveRule(projectId: string, id: string, data: AlarmRuleUpdate) {
    saving.value = true;
    try {
      const rule = await updateAlarmRule(projectId, id, data);
      editing.value = rule;
      mergeRuleIntoList(rule);
      return rule;
    } finally {
      saving.value = false;
    }
  }

  async function removeRule(projectId: string, id: string) {
    deleting.value = true;
    try {
      await deleteAlarmRule(projectId, id);
      list.value = list.value.filter((item) => item.id !== id);
      total.value = Math.max(0, total.value - 1);
      if (editing.value?.id === id) {
        closeEdit();
      }
    } finally {
      deleting.value = false;
    }
  }

  async function setRuleEnabled(projectId: string, id: string, isEnabled: boolean) {
    saving.value = true;
    try {
      const rule = await toggleAlarmRule(projectId, id, isEnabled);
      editing.value = rule;
      mergeRuleIntoList(rule);
      return rule;
    } finally {
      saving.value = false;
    }
  }

  async function runTrial(
    projectId: string,
    id: string,
    payload: AlarmTrialPayload = {},
  ) {
    trial.value = {
      ruleId: id,
      payload,
      result: null,
      running: true,
      error: "",
    };
    try {
      const result = await testAlarmRule(projectId, id, payload);
      trial.value.result = result;
      return result;
    } catch (error) {
      trial.value.error = getApiErrorMessage(error, "报警规则试算失败");
      throw error;
    } finally {
      trial.value.running = false;
    }
  }

  function clearTrial() {
    trial.value = {
      ruleId: "",
      payload: {},
      result: null,
      running: false,
      error: "",
    };
  }

  async function fetchContract(projectId: string, id: string) {
    contract.value = {
      ruleId: id,
      data: null,
      loading: true,
      error: "",
    };
    try {
      const data = await getAlarmRuleContract(projectId, id);
      contract.value.data = data;
      return data;
    } catch (error) {
      contract.value.error = getApiErrorMessage(error, "报警规则契约不可用");
      throw error;
    } finally {
      contract.value.loading = false;
    }
  }

  async function validateDraft(projectId: string, payload: AlarmRuleSave) {
    validationLoading.value = true;
    validationError.value = "";
    try {
      validation.value = await validateAlarmRuleDraft(projectId, payload);
      return validation.value;
    } catch (error) {
      validation.value = null;
      validationError.value = getApiErrorMessage(error, "报警草稿校验失败");
      throw error;
    } finally {
      validationLoading.value = false;
    }
  }

  return {
    list,
    total,
    loading,
    listError,
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
    hasEditing,
    fetchList,
    openForEdit,
    closeEdit,
    createRule,
    saveRule,
    removeRule,
    setRuleEnabled,
    runTrial,
    clearTrial,
    fetchContract,
    validateDraft,
  };
});
