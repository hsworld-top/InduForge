<template>
  <div class="alarm-workspace">
    <AlarmRuleList
      :rules="alarmStore.list"
      :total="alarmStore.total"
      :selected-id="selectedRuleId"
      :loading="alarmStore.loading"
      :error="alarmStore.listError"
      @refresh="reloadList"
      @create="openCreateDialog"
      @select="selectRule"
      @filter="filterList"
    />

    <section class="alarm-workspace__main">
      <header class="alarm-workspace__head">
        <div>
          <strong>{{ activeDraft?.name || "未选择报警规则" }}</strong>
          <span>{{ activeDraft?.targetPath || "从左侧选择规则" }}</span>
        </div>
        <div class="alarm-workspace__actions">
          <button
            type="button"
            :disabled="!activeDraft || alarmStore.saving"
            @click="saveActiveDraft"
          >
            保存
          </button>
          <button
            type="button"
            :disabled="!activeDraft || alarmStore.saving"
            @click="toggleEnabled"
          >
            {{ activeDraft?.isEnabled ? "停用" : "启用" }}
          </button>
          <button
            type="button"
            class="is-danger"
            :disabled="!activeDraft || alarmStore.deleting"
            @click="deleteRule"
          >
            删除
          </button>
        </div>
      </header>

      <div v-if="alarmStore.detailError" class="alarm-workspace__error">
        {{ alarmStore.detailError }}
      </div>
      <div v-else-if="alarmStore.detailLoading" class="alarm-workspace__empty">
        详情加载中
      </div>
      <div v-else-if="activeDraft" class="alarm-workspace__editor">
        <label>
          <span>规则名</span>
          <input v-model="activeDraft.name" type="text" @input="markDirty" />
        </label>
        <label>
          <span>目标路径</span>
          <input v-model="activeDraft.targetPath" type="text" @input="markDirty" />
        </label>
        <label>
          <span>规则类型</span>
          <select v-model="activeDraft.ruleType" @change="markDirty">
            <option
              v-for="item in alarmRuleTypeOptions"
              :key="item.value"
              :value="item.value"
            >
              {{ item.label }}
            </option>
          </select>
        </label>
        <label>
          <span>严重度</span>
          <select v-model="activeDraft.severity" @change="markDirty">
            <option
              v-for="item in alarmSeverityOptions"
              :key="item.value"
              :value="item.value"
            >
              {{ item.label }}
            </option>
          </select>
        </label>
        <label class="is-wide">
          <span>描述</span>
          <textarea v-model="activeDraft.description" rows="3" @input="markDirty" />
        </label>
      </div>
      <div v-else class="alarm-workspace__empty">暂无选中规则</div>

      <footer class="alarm-workspace__panel">
        <nav class="alarm-workspace__tabs">
          <button
            v-for="tab in tabs"
            :key="tab"
            type="button"
            :class="{ 'is-active': activeTab === tab }"
            @click="selectTab(tab)"
          >
            {{ tabLabels[tab] }}
          </button>
        </nav>

        <div v-if="activeTab === 'config'" class="alarm-workspace__panel-body">
          <pre>{{ draftPreview }}</pre>
        </div>
        <div v-else-if="activeTab === 'test'" class="alarm-workspace__panel-body">
          <button
            type="button"
            :disabled="!selectedRuleId || alarmStore.trial.running"
            @click="runTrial"
          >
            试算
          </button>
          <pre>{{ trialPreview }}</pre>
        </div>
        <div v-else class="alarm-workspace__panel-body">
          <button
            type="button"
            :disabled="!selectedRuleId || alarmStore.contract.loading"
            @click="loadContract"
          >
            读取契约
          </button>
          <pre>{{ contractPreview }}</pre>
        </div>
      </footer>
    </section>

    <CreateAlarmRuleDialog
      v-model="createDialogVisible"
      :project-id="projectId"
      :submitting="alarmStore.creating"
      :error="alarmStore.createError"
      @submit="createRule"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { ElMessage } from "element-plus";
import type { AlarmRuleSave } from "@/api/schemas/alarm.schema";
import { useAlarmStore } from "@/stores/alarm.store";
import {
  alarmRuleTypeOptions,
  alarmSeverityOptions,
  draftToAlarmSavePayload,
  toAlarmDraft,
  type AlarmRuleDraft,
} from "@/components/alarm/alarmRuleModel";
import AlarmRuleList from "./AlarmRuleList.vue";
import CreateAlarmRuleDialog from "./CreateAlarmRuleDialog.vue";

const props = defineProps<{
  projectId: string;
}>();

type WorkspaceTab = "config" | "test" | "contract";

const tabs: WorkspaceTab[] = ["config", "test", "contract"];
const tabLabels: Record<WorkspaceTab, string> = {
  config: "规则配置",
  test: "试算",
  contract: "契约",
};

const route = useRoute();
const router = useRouter();
const alarmStore = useAlarmStore();
const activeDraft = ref<AlarmRuleDraft | null>(null);
const createDialogVisible = ref(false);
const listParams = ref<Record<string, string>>({});

const selectedRuleId = computed(() => {
  const value = route.params.objectId;
  return typeof value === "string" ? value : "";
});

const activeTab = computed<WorkspaceTab>(() => {
  const value = route.params.tab;
  return tabs.includes(value as WorkspaceTab) ? (value as WorkspaceTab) : "config";
});

const draftPreview = computed(() =>
  JSON.stringify(activeDraft.value ? draftToAlarmSavePayload(activeDraft.value) : {}, null, 2),
);

const trialPreview = computed(() =>
  JSON.stringify(
    alarmStore.trial.error || alarmStore.trial.result || { state: "idle" },
    null,
    2,
  ),
);

const contractPreview = computed(() =>
  JSON.stringify(
    alarmStore.contract.error || alarmStore.contract.data || { state: "idle" },
    null,
    2,
  ),
);

const routeBase = computed(() => (route.path.startsWith("/debug/") ? "/debug" : ""));

const replaceAlarmRoute = (ruleId?: string, tab: WorkspaceTab = "config") => {
  const tabPath = tab === "config" ? "" : `/${tab}`;
  const rulePath = ruleId ? `/${ruleId}${tabPath}` : "";
  router.push(`${routeBase.value}/alarm${rulePath}`);
};

const cleanParams = (params: Record<string, string>) =>
  Object.fromEntries(
    Object.entries(params).filter(([, value]) => value.trim() !== ""),
  );

const reloadList = () => alarmStore.fetchList(props.projectId, listParams.value);

const filterList = (params: Record<string, string>) => {
  listParams.value = cleanParams(params);
  reloadList();
};

const openCreateDialog = () => {
  createDialogVisible.value = true;
};

const createRule = async (payload: AlarmRuleSave) => {
  const rule = await alarmStore.createRule(props.projectId, payload);
  activeDraft.value = toAlarmDraft(rule);
  createDialogVisible.value = false;
  replaceAlarmRoute(rule.id, "config");
  ElMessage.success("报警规则已创建");
};

const selectRule = (ruleId: string) => {
  replaceAlarmRoute(ruleId, "config");
};

const selectTab = (tab: WorkspaceTab) => {
  replaceAlarmRoute(selectedRuleId.value, tab);
};

const markDirty = () => {
  if (activeDraft.value) {
    activeDraft.value.dirty = true;
  }
};

const saveActiveDraft = async () => {
  if (!selectedRuleId.value || !activeDraft.value) {
    return;
  }
  const rule = await alarmStore.saveRule(
    props.projectId,
    selectedRuleId.value,
    draftToAlarmSavePayload(activeDraft.value),
  );
  activeDraft.value = toAlarmDraft(rule);
};

const toggleEnabled = async () => {
  if (!selectedRuleId.value || !activeDraft.value) {
    return;
  }
  const rule = await alarmStore.setRuleEnabled(
    props.projectId,
    selectedRuleId.value,
    !activeDraft.value.isEnabled,
  );
  activeDraft.value = toAlarmDraft(rule);
};

const deleteRule = async () => {
  if (!selectedRuleId.value || !window.confirm("确认删除当前报警规则？")) {
    return;
  }
  await alarmStore.removeRule(props.projectId, selectedRuleId.value);
  replaceAlarmRoute(alarmStore.list[0]?.id);
};

const runTrial = () => {
  if (selectedRuleId.value) {
    alarmStore.runTrial(props.projectId, selectedRuleId.value, {});
  }
};

const loadContract = () => {
  if (selectedRuleId.value) {
    alarmStore.fetchContract(props.projectId, selectedRuleId.value);
  }
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
  selectedRuleId,
  async (ruleId) => {
    if (!ruleId) {
      activeDraft.value = null;
      alarmStore.closeEdit();
      return;
    }
    const rule = await alarmStore.openForEdit(props.projectId, ruleId);
    activeDraft.value = toAlarmDraft(rule);
  },
  { immediate: true },
);

watch(activeTab, (tab) => {
  if (tab === "contract" && selectedRuleId.value) {
    loadContract();
  }
});

onMounted(() => {
  reloadList();
});
</script>

<style scoped>
.alarm-workspace {
  height: 100%;
  min-height: 0;
  display: grid;
  grid-template-columns: 320px minmax(0, 1fr);
  background: var(--dc-surface-subtle);
  color: var(--dc-text);
}

.alarm-workspace__main {
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-rows: auto minmax(0, 1fr) 280px;
}

.alarm-workspace__head {
  min-height: 56px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
}

.alarm-workspace__head strong,
.alarm-workspace__head span {
  display: block;
}

.alarm-workspace__head strong {
  font-size: 14px;
}

.alarm-workspace__head span {
  margin-top: 4px;
  color: var(--dc-text-secondary);
  font-size: 12px;
}

.alarm-workspace__actions {
  display: flex;
  gap: 8px;
}

.alarm-workspace button {
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-text);
  font-family: inherit;
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
}

.alarm-workspace__actions button,
.alarm-workspace__panel-body button {
  height: 32px;
  padding: 0 12px;
}

.alarm-workspace button:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.alarm-workspace__actions .is-danger {
  border-color: rgba(220, 38, 38, 0.28);
  color: var(--dc-danger, #b91c1c);
}

.alarm-workspace__editor {
  min-height: 0;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  align-content: start;
  gap: 14px;
  padding: 16px;
  overflow-y: auto;
}

.alarm-workspace__editor label {
  display: grid;
  gap: 6px;
}

.alarm-workspace__editor label.is-wide {
  grid-column: 1 / -1;
}

.alarm-workspace__editor span {
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.alarm-workspace__editor input,
.alarm-workspace__editor select,
.alarm-workspace__editor textarea {
  width: 100%;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text);
  font-family: inherit;
  font-size: 13px;
  outline: none;
}

.alarm-workspace__editor input,
.alarm-workspace__editor select {
  height: 36px;
  padding: 0 10px;
}

.alarm-workspace__editor textarea {
  padding: 10px;
  resize: vertical;
}

.alarm-workspace__panel {
  min-height: 0;
  border-top: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
}

.alarm-workspace__tabs {
  display: flex;
  gap: 6px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--dc-border);
}

.alarm-workspace__tabs button {
  height: 30px;
  padding: 0 12px;
}

.alarm-workspace__tabs button.is-active {
  border-color: var(--dc-primary);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.alarm-workspace__panel-body {
  height: calc(100% - 47px);
  padding: 12px;
  overflow: auto;
}

.alarm-workspace__panel-body pre {
  margin: 10px 0 0;
  color: var(--dc-text-secondary);
  font-size: 12px;
  line-height: 1.6;
}

.alarm-workspace__empty,
.alarm-workspace__error {
  margin: 12px;
  padding: 14px;
  border: 1px dashed var(--dc-border);
  border-radius: var(--dc-radius-sm);
  color: var(--dc-text-secondary);
  font-size: 13px;
}

.alarm-workspace__error {
  border-color: rgba(220, 38, 38, 0.32);
  color: var(--dc-danger, #b91c1c);
}

@media (max-width: 920px) {
  .alarm-workspace {
    grid-template-columns: 1fr;
  }

  .alarm-workspace__editor {
    grid-template-columns: 1fr;
  }
}
</style>
