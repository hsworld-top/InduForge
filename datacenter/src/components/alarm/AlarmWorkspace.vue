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

    <AlarmEditorShell
      :draft="activeDraft"
      :active-tab="activeTab"
      :loading="alarmStore.detailLoading"
      :error="alarmStore.detailError"
      :saving="alarmStore.saving"
      :deleting="alarmStore.deleting"
      @update="updateActiveDraft"
      @save="saveActiveDraft"
      @toggle="toggleEnabled"
      @delete="deleteRule"
      @select-tab="selectTab"
    />

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
  draftToAlarmSavePayload,
  toAlarmDraft,
  type AlarmRuleDraft,
} from "@/components/alarm/alarmRuleModel";
import AlarmEditorShell, {
  type AlarmEditorTab,
} from "./AlarmEditorShell.vue";
import AlarmRuleList from "./AlarmRuleList.vue";
import CreateAlarmRuleDialog from "./CreateAlarmRuleDialog.vue";

const props = defineProps<{
  projectId: string;
}>();

type WorkspaceTab = AlarmEditorTab;

const tabs: WorkspaceTab[] = ["config", "test", "contract"];

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

const updateActiveDraft = (patch: Partial<AlarmRuleDraft>) => {
  if (activeDraft.value) {
    activeDraft.value = {
      ...activeDraft.value,
      ...patch,
      dirty: patch.dirty ?? true,
    };
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

@media (max-width: 920px) {
  .alarm-workspace {
    grid-template-columns: 1fr;
  }
}
</style>
