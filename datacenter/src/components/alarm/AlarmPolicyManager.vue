<template>
  <aside class="alarm-policy-manager" @click="closeContextMenu">
    <header class="alarm-policy-manager__head">
      <div>
        <strong>{{ t("alarm.policies") }}</strong>
        <span>{{ t("alarm.policyCount", { count: total }) }}</span>
      </div>
      <div class="alarm-policy-manager__actions">
        <el-tooltip :content="t('alarm.createGroup')" placement="bottom">
          <button
            type="button"
            :aria-label="t('alarm.createGroup')"
            @click="emit('createGroup')"
          >
            <FolderAdd />
          </button>
        </el-tooltip>
        <el-tooltip :content="t('actions.refresh')" placement="bottom">
          <button
            type="button"
            :aria-label="t('alarm.refreshPolicies')"
            :disabled="loading"
            @click="emit('refresh')"
          >
            <Refresh />
          </button>
        </el-tooltip>
        <button type="button" class="is-primary" @click="emit('create')">
          <Plus />
          <span>{{ t("alarm.createPolicy") }}</span>
        </button>
      </div>
    </header>

    <div class="alarm-policy-manager__filters">
      <label class="is-wide">
        <input
          v-model="filters.search"
          type="search"
          :placeholder="t('alarm.searchPlaceholder')"
          @input="emitFilter"
        />
      </label>
      <div class="alarm-policy-manager__pills" :aria-label="t('alarm.statusFilter')">
        <button
          v-for="item in enabledOptions"
          :key="item.value"
          type="button"
          :class="{ 'is-active': filters.enabled === item.value }"
          @click="setEnabled(item.value)"
        >
          {{ item.label }}
        </button>
      </div>
      <label>
        <span>{{ t("alarm.type") }}</span>
        <select v-model="filters.conditionType" @change="emitFilter">
          <option value="">{{ t("alarm.all") }}</option>
          <option v-for="item in conditionTypeOptions" :key="item.value" :value="item.value">
            {{ item.label }}
          </option>
        </select>
      </label>
    </div>

    <div class="alarm-policy-manager__selectbar">
      <label>
        <input
          type="checkbox"
          :checked="allVisibleSelected"
          :indeterminate.prop="someVisibleSelected && !allVisibleSelected"
          :disabled="!visiblePolicies.length"
          @change="toggleVisible(checked($event))"
        />
        <span>{{ t("alarm.currentResult") }}</span>
      </label>
      <button type="button" :disabled="!visiblePolicies.length" @click="selectFiltered">
        {{ t("alarm.selectFiltered") }}
      </button>
      <button v-if="selectedCount" type="button" @click="emit('clearSelection')">
        {{ t("actions.clean") }}
      </button>
    </div>

    <div v-if="selectedCount" class="alarm-policy-manager__bulk">
      <div class="alarm-policy-manager__bulk-row">
        <strong>{{ t("alarm.selectedCount", { count: selectedCount }) }}</strong>
        <button type="button" :disabled="!selectedCount" @click="emit('batchEnable')">
          {{ t("common.enabled") }}
        </button>
        <button type="button" :disabled="!selectedCount" @click="emit('batchDisable')">
          {{ t("alarm.stopped") }}
        </button>
        <button type="button" :disabled="!selectedCount" @click="emit('batchConditions')">
          {{ t("alarm.batchConditions") }}
        </button>
        <select
          :disabled="!selectedCount"
          :aria-label="t('alarm.batchMoveGroup')"
          @change="moveSelected(value($event))"
        >
          <option value="">{{ t("alarm.moveTo") }}</option>
          <option value="__root__">{{ t("alarm.root") }}</option>
          <option v-for="group in tree.groups" :key="group.id" :value="group.id">
            {{ group.name }}
          </option>
        </select>
      </div>
    </div>

    <div v-if="error" class="alarm-policy-manager__state is-error">
      <strong>{{ t("alarm.policyTreeUnavailable") }}</strong>
      <span>{{ error }}</span>
      <button type="button" @click="emit('refresh')">{{ t("alarm.retry") }}</button>
    </div>
    <div v-else-if="loading" class="alarm-policy-manager__state">
      <strong>{{ t("alarm.loadingTitle") }}</strong>
      <span>{{ t("alarm.loadingTree") }}</span>
    </div>
    <div v-else-if="!visiblePolicies.length" class="alarm-policy-manager__state">
      <strong>{{ t("alarm.emptyPolicies") }}</strong>
      <span>{{ t("alarm.emptyPoliciesHint") }}</span>
      <button type="button" @click="emit('create')">{{ t("alarm.createPolicy") }}</button>
    </div>

    <div v-else class="alarm-policy-manager__tree">
      <template v-for="node in nodes" :key="node.key">
        <section v-if="node.type === 'group'" class="alarm-policy-manager__group">
          <header>
            <label>
              <input
                type="checkbox"
                :checked="groupAllSelected(node.policyIds)"
                :indeterminate.prop="groupSomeSelected(node.policyIds) && !groupAllSelected(node.policyIds)"
                :disabled="!node.policyIds.length"
                @change="emit('selectGroup', node.policyIds, checked($event))"
              />
              <strong>{{ node.group.name }}</strong>
            </label>
            <span>{{ node.policies.length }}</span>
          </header>
          <div class="alarm-policy-manager__rows">
            <button
              v-for="policy in node.policies"
              :key="policy.id"
              type="button"
              class="alarm-policy-manager__row"
              :class="{
                'is-active': policy.id === selectedId,
                'is-disabled': !policy.effectiveEnabled,
              }"
              @click="emit('select', policy.id)"
              @contextmenu.prevent.stop="openPolicyMenu($event, policy)"
            >
              <input
                type="checkbox"
                :checked="isSelected(policy.id)"
                @click.stop
                @change="emit('selectPolicy', policy.id, checked($event))"
              />
              <i class="alarm-policy-manager__mark" aria-hidden="true" />
              <span>
                <strong>{{ policy.name }}</strong>
                <code>{{ policySummary(policy) }}</code>
              </span>
              <em :class="{ 'is-off': !policy.isEnabled }">{{ policy.isEnabled ? t("common.enabled") : t("alarm.stopped") }}</em>
            </button>
          </div>
        </section>
        <button
          v-else
          type="button"
          class="alarm-policy-manager__row is-root"
          :class="{
            'is-active': node.policy.id === selectedId,
            'is-disabled': !node.policy.effectiveEnabled,
          }"
          @click="emit('select', node.policy.id)"
          @contextmenu.prevent.stop="openPolicyMenu($event, node.policy)"
        >
          <input
            type="checkbox"
            :checked="isSelected(node.policy.id)"
            @click.stop
            @change="emit('selectPolicy', node.policy.id, checked($event))"
          />
          <i class="alarm-policy-manager__mark" aria-hidden="true" />
          <span>
            <strong>{{ node.policy.name }}</strong>
            <code>{{ policySummary(node.policy) }}</code>
          </span>
          <em :class="{ 'is-off': !node.policy.isEnabled }">{{ node.policy.isEnabled ? t("common.enabled") : t("alarm.stopped") }}</em>
        </button>
      </template>
    </div>

    <div
      v-if="contextMenu.visible && contextPolicy"
      class="alarm-policy-manager__menu"
      :style="{ left: `${contextMenu.x}px`, top: `${contextMenu.y}px` }"
      @click.stop
    >
      <strong>{{ contextPolicy.name }}</strong>
      <button type="button" @click="renameContextPolicy">{{ t("alarm.rename") }}</button>
      <label>
        <span>{{ t("alarm.moveToGroup") }}</span>
        <select :value="contextPolicy.groupId ?? ''" @change="moveContextPolicy(value($event))">
          <option value="">{{ t("alarm.root") }}</option>
          <option v-for="group in tree.groups" :key="group.id" :value="group.id">
            {{ group.name }}
          </option>
        </select>
      </label>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive } from "vue";
import { FolderAdd, Plus, Refresh } from "@element-plus/icons-vue";
import type {
  AlarmBulkSelection,
  AlarmPolicy,
  AlarmPolicyGroup,
  AlarmPolicyTree,
} from "@/api/schemas/alarm.schema";
import { t } from "@/i18n/runtime";

type GroupNode = {
  type: "group";
  key: string;
  group: AlarmPolicyGroup;
  policies: AlarmPolicy[];
  policyIds: string[];
};

type PolicyNode = {
  type: "policy";
  key: string;
  policy: AlarmPolicy;
};

const props = defineProps<{
  tree: AlarmPolicyTree;
  selection: AlarmBulkSelection;
  selectedId: string;
  selectedCount: number;
  total: number;
  loading: boolean;
  error: string;
}>();

const emit = defineEmits<{
  refresh: [];
  create: [];
  createGroup: [];
  select: [id: string];
  filter: [params: Record<string, string>];
  selectPolicy: [id: string, selected: boolean];
  selectGroup: [ids: string[], selected: boolean];
  selectFiltered: [filters: Record<string, string>];
  clearSelection: [];
  batchEnable: [];
  batchDisable: [];
  batchConditions: [];
  batchMove: [groupId: string | null];
  renamePolicy: [id: string, name: string];
  movePolicy: [id: string, groupId: string | null];
}>();

const filters = reactive({
  search: "",
  enabled: "",
  conditionType: "",
});

const enabledOptions = computed(() => [
  { value: "", label: t("alarm.all") },
  { value: "true", label: t("common.enabled") },
  { value: "false", label: t("alarm.stopped") },
]);

const conditionTypeOptions = computed(() => [
  { value: "HH", label: t("alarm.conditionTypes.HH") },
  { value: "H", label: t("alarm.conditionTypes.H") },
  { value: "L", label: t("alarm.conditionTypes.L") },
  { value: "LL", label: t("alarm.conditionTypes.LL") },
  { value: "deviation_high", label: t("alarm.conditionTypes.deviationHigh") },
  { value: "deviation_low", label: t("alarm.conditionTypes.deviationLow") },
  { value: "rate_of_change", label: t("alarm.conditionTypes.rateOfChange") },
  { value: "cel", label: t("alarm.conditionTypes.cel") },
]);

const contextMenu = reactive({
  visible: false,
  x: 0,
  y: 0,
  policyId: "",
});

const visiblePolicies = computed(() => props.tree.policies);

const contextPolicy = computed(
  () => props.tree.policies.find((policy) => policy.id === contextMenu.policyId) ?? null,
);

const nodes = computed<Array<GroupNode | PolicyNode>>(() => {
  const byGroup = new Map<string, AlarmPolicy[]>();
  for (const policy of props.tree.policies) {
    if (!policy.groupId) {
      continue;
    }
    const current = byGroup.get(policy.groupId) ?? [];
    current.push(policy);
    byGroup.set(policy.groupId, current);
  }

  const result: Array<GroupNode | PolicyNode> = props.tree.groups
    .map((group) => {
      const policies = byGroup.get(group.id) ?? [];
      return {
        type: "group" as const,
        key: `group-${group.id}`,
        group,
        policies,
        policyIds: policies.map((policy) => policy.id),
      };
    })
    .filter((node) => node.policies.length > 0);

  for (const policy of props.tree.rootPolicies) {
    result.push({ type: "policy", key: `policy-${policy.id}`, policy });
  }
  return result;
});

const checked = (event: Event) => (event.target as HTMLInputElement).checked;
const value = (event: Event) =>
  (event.target as HTMLInputElement | HTMLSelectElement).value;

const cleanFilters = () =>
  Object.fromEntries(
    Object.entries(filters).filter(([, value]) => value.trim() !== ""),
  );

const emitFilter = () => {
  emit("filter", cleanFilters());
};

const setEnabled = (value: string) => {
  filters.enabled = value;
  emitFilter();
};

const isSelected = (id: string) => {
  if (props.selection.mode === "filtered") {
    return !props.selection.excludePolicyIds.includes(id);
  }
  return props.selection.policyIds.includes(id);
};

const allVisibleSelected = computed(
  () => visiblePolicies.value.length > 0 && visiblePolicies.value.every((item) => isSelected(item.id)),
);
const someVisibleSelected = computed(() =>
  visiblePolicies.value.some((item) => isSelected(item.id)),
);

const groupAllSelected = (ids: string[]) =>
  ids.length > 0 && ids.every((id) => isSelected(id));
const groupSomeSelected = (ids: string[]) => ids.some((id) => isSelected(id));

const toggleVisible = (selected: boolean) => {
  emit(
    "selectGroup",
    visiblePolicies.value.map((policy) => policy.id),
    selected,
  );
};

const selectFiltered = () => {
  emit("selectFiltered", cleanFilters());
};

const moveSelected = (raw: string) => {
  if (!raw) {
    return;
  }
  emit("batchMove", raw === "__root__" ? null : raw);
};

const openPolicyMenu = (event: MouseEvent, policy: AlarmPolicy) => {
  contextMenu.visible = true;
  contextMenu.policyId = policy.id;
  contextMenu.x = event.clientX;
  contextMenu.y = event.clientY;
};

const closeContextMenu = () => {
  contextMenu.visible = false;
};

const renameContextPolicy = () => {
  if (!contextPolicy.value) {
    return;
  }
  const next = window.prompt(t("alarm.policyName"), contextPolicy.value.name);
  const name = next?.trim();
  if (!name || name === contextPolicy.value.name) {
    closeContextMenu();
    return;
  }
  emit("renamePolicy", contextPolicy.value.id, name);
  closeContextMenu();
};

const moveContextPolicy = (raw: string) => {
  if (!contextPolicy.value) {
    return;
  }
  emit("movePolicy", contextPolicy.value.id, raw || null);
  closeContextMenu();
};

const policySummary = (policy: AlarmPolicy) => {
  if (policy.mode === "derived") {
    return policy.derivedExpression || "计算后判断";
  }
  return policy.targets.map((target) => target.path).join("、") || "逐点判断";
};

onMounted(() => {
  window.addEventListener("scroll", closeContextMenu, true);
});

onBeforeUnmount(() => {
  window.removeEventListener("scroll", closeContextMenu, true);
});
</script>

<style scoped>
.alarm-policy-manager {
  min-height: 0;
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text);
}

.alarm-policy-manager__head {
  min-height: 58px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 10px 12px;
  border-bottom: 1px solid var(--dc-border);
}

.alarm-policy-manager__head strong,
.alarm-policy-manager__head span {
  display: block;
}

.alarm-policy-manager__head strong {
  font-size: 14px;
}

.alarm-policy-manager__head span {
  margin-top: 3px;
  color: var(--dc-text-secondary);
  font-size: 12px;
}

.alarm-policy-manager__actions {
  display: flex;
  gap: 8px;
}

.alarm-policy-manager button,
.alarm-policy-manager input,
.alarm-policy-manager select {
  font-family: inherit;
}

.alarm-policy-manager__actions button,
.alarm-policy-manager__selectbar button,
.alarm-policy-manager__bulk button,
.alarm-policy-manager__bulk select {
  height: 30px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-text-secondary);
  cursor: pointer;
  font-size: 12px;
  font-weight: 700;
  padding: 0 9px;
}

.alarm-policy-manager__bulk select {
  justify-content: flex-start;
  min-width: 104px;
  outline: none;
}

.alarm-policy-manager__actions button:not(.is-primary) {
  width: 30px;
  padding: 0;
}

.alarm-policy-manager__actions .is-primary {
  color: var(--dc-primary);
}

.alarm-policy-manager svg {
  width: 15px;
  height: 15px;
}

.alarm-policy-manager__filters {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 8px;
  padding: 10px 12px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
}

.alarm-policy-manager__filters label {
  min-width: 0;
  display: grid;
  gap: 5px;
}

.alarm-policy-manager__filters span {
  color: var(--dc-text-muted);
  font-size: 11px;
  font-weight: 700;
}

.alarm-policy-manager__filters input,
.alarm-policy-manager__filters select {
  width: 100%;
  height: 30px;
  min-width: 0;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text);
  font-size: 12px;
  outline: none;
  padding: 0 8px;
}

.alarm-policy-manager__pills {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 6px;
}

.alarm-policy-manager__pills button {
  height: 28px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
  cursor: pointer;
  font-family: inherit;
  font-size: 12px;
  font-weight: 700;
}

.alarm-policy-manager__pills button.is-active {
  border-color: rgba(37, 99, 235, 0.34);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.alarm-policy-manager__selectbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
}

.alarm-policy-manager__selectbar label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.alarm-policy-manager__bulk {
  display: grid;
  gap: 6px;
  padding: 9px 12px;
  border-bottom: 1px solid var(--dc-border);
}

.alarm-policy-manager__bulk-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
}

.alarm-policy-manager__bulk-row strong {
  color: var(--dc-text);
  font-size: 12px;
  margin-right: 2px;
}

.alarm-policy-manager__bulk label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.alarm-policy-manager__bulk button:disabled,
.alarm-policy-manager__actions button:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.alarm-policy-manager__tree {
  min-height: 0;
  display: grid;
  align-content: start;
  gap: 8px;
  padding: 10px;
  overflow-y: auto;
}

.alarm-policy-manager__group {
  display: grid;
  gap: 6px;
}

.alarm-policy-manager__group > header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 6px 4px;
  color: var(--dc-text-secondary);
}

.alarm-policy-manager__group label {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 7px;
}

.alarm-policy-manager__group strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
}

.alarm-policy-manager__group header span {
  color: var(--dc-text-muted);
  font-size: 12px;
}

.alarm-policy-manager__rows {
  display: grid;
  gap: 6px;
}

.alarm-policy-manager__row {
  width: 100%;
  min-height: 54px;
  display: grid;
  grid-template-columns: auto 24px minmax(0, 1fr) auto;
  align-items: center;
  gap: 9px;
  padding: 8px 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-text);
  cursor: pointer;
  text-align: left;
}

.alarm-policy-manager__row.is-root {
  margin-bottom: 0;
}

.alarm-policy-manager__row:hover,
.alarm-policy-manager__row.is-active {
  border-color: rgba(37, 99, 235, 0.32);
  background: var(--dc-primary-soft);
}

.alarm-policy-manager__row.is-disabled {
  opacity: 0.68;
}

.alarm-policy-manager__mark {
  width: 24px;
  height: 24px;
  display: inline-block;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background:
    linear-gradient(135deg, rgba(37, 99, 235, 0.16), transparent),
    var(--dc-surface-raised);
}

.alarm-policy-manager__row strong,
.alarm-policy-manager__row code {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.alarm-policy-manager__row strong {
  font-size: 13px;
}

.alarm-policy-manager__row code {
  margin-top: 4px;
  color: var(--dc-text-secondary);
  font-family: var(--dc-font-mono);
  font-size: 12px;
}

.alarm-policy-manager__row em {
  height: 22px;
  display: inline-flex;
  align-items: center;
  border-radius: 999px;
  background: rgba(22, 163, 74, 0.1);
  color: #15803d;
  font-size: 12px;
  font-style: normal;
  font-weight: 700;
  padding: 0 8px;
  white-space: nowrap;
}

.alarm-policy-manager__row em.is-off {
  background: var(--dc-surface);
  color: var(--dc-text-muted);
}

.alarm-policy-manager__menu {
  position: fixed;
  z-index: 30;
  width: 190px;
  display: grid;
  gap: 6px;
  padding: 8px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  box-shadow: 0 12px 28px rgba(15, 23, 42, 0.14);
}

.alarm-policy-manager__menu strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--dc-text);
  font-size: 12px;
  padding: 2px 4px 5px;
}

.alarm-policy-manager__menu button,
.alarm-policy-manager__menu select {
  width: 100%;
  height: 30px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-text);
  cursor: pointer;
  font-family: inherit;
  font-size: 12px;
  padding: 0 8px;
  text-align: left;
}

.alarm-policy-manager__menu label {
  display: grid;
  gap: 4px;
}

.alarm-policy-manager__menu label span {
  color: var(--dc-text-muted);
  font-size: 11px;
  font-weight: 700;
}

.alarm-policy-manager__state {
  display: grid;
  gap: 8px;
  margin: 12px;
  padding: 16px;
  border: 1px dashed var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background:
    linear-gradient(135deg, rgba(37, 99, 235, 0.05), transparent 58%),
    var(--dc-surface-muted);
  color: var(--dc-text-secondary);
  font-size: 13px;
}

.alarm-policy-manager__state strong {
  color: var(--dc-text);
  font-size: 13px;
}

.alarm-policy-manager__state button {
  justify-self: start;
  height: 28px;
  padding: 0 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-primary);
  cursor: pointer;
  font-size: 12px;
  font-weight: 700;
}

.alarm-policy-manager__state.is-error {
  border-color: rgba(220, 38, 38, 0.32);
}

.alarm-policy-manager__state.is-error strong,
.alarm-policy-manager__state.is-error span {
  color: var(--dc-danger, #b91c1c);
}

@media (max-width: 920px) {
  .alarm-policy-manager {
    max-height: 380px;
    border-right: 0;
    border-bottom: 1px solid var(--dc-border);
  }
}
</style>
