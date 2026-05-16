import { defineStore } from "pinia";
import { ref } from "vue";
import { getAlarmRules, getAlarmRule } from "@/api/alarm.api";
import type { AlarmRule, AlarmTrialResult } from "@/api/schemas/alarm.schema";

// 试算上下文
interface TrialContext {
  ruleId: string | number;
  input: Record<string, unknown>;
  result: AlarmTrialResult | null;
  running: boolean;
}

export const useAlarmStore = defineStore("alarm", () => {
  const list = ref<AlarmRule[]>([]);
  const total = ref(0);
  const loading = ref(false);
  // 当前编辑的报警规则
  const editing = ref<AlarmRule | null>(null);
  // 试算上下文
  const trialContext = ref<TrialContext | null>(null);

  async function fetchList(projectId: string, params: Record<string, unknown> = {}) {
    loading.value = true;
    try {
      const res = await getAlarmRules(projectId, params);
      list.value = res.list;
      total.value = res.pagination?.total ?? res.list.length;
    } finally {
      loading.value = false;
    }
  }

  async function openForEdit(projectId: string, ruleId: string) {
    loading.value = true;
    try {
      editing.value = await getAlarmRule(projectId, ruleId);
    } finally {
      loading.value = false;
    }
  }

  /** 新建时直接设置一个空规则 */
  function newRule() {
    editing.value = {
      id: "",
      name: "",
      createdAt: null,
      updatedAt: null,
    };
  }

  function closeEdit() {
    editing.value = null;
  }

  /** 启动试算 */
  function startTrial(ruleId: string | number, input: Record<string, unknown> = {}) {
    trialContext.value = { ruleId, input, result: null, running: true };
  }

  /** 写入试算结果 */
  function finishTrial(result: AlarmTrialResult) {
    if (trialContext.value) {
      trialContext.value.result = result;
      trialContext.value.running = false;
    }
  }

  function clearTrial() {
    trialContext.value = null;
  }

  return {
    list,
    total,
    loading,
    editing,
    trialContext,
    fetchList,
    openForEdit,
    newRule,
    closeEdit,
    startTrial,
    finishTrial,
    clearTrial,
  };
});
