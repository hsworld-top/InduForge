<template>
  <aside class="alarm-policy-manager">
    <header class="alarm-policy-manager__head">
      <div class="alarm-policy-manager__title">
        <h2>{{ t("alarm.policies") }}</h2>
      </div>
      <div class="alarm-policy-manager__actions">
        <button
          type="button"
          class="alarm-policy-manager__primary"
          :title="t('alarm.createPolicy')"
          :aria-label="t('alarm.createPolicy')"
          @click="emit('create')"
        >
          <IconTablerPlus class="alarm-policy-manager__button-icon" />
        </button>
        <button
          type="button"
          class="alarm-policy-manager__secondary"
          :title="t('alarm.createGroup')"
          :aria-label="t('alarm.createGroup')"
          @click="emit('createGroup')"
        >
          <IconTablerFolderPlus class="alarm-policy-manager__button-icon" />
        </button>
        <button
          type="button"
          class="alarm-policy-manager__secondary"
          :title="t('actions.refresh')"
          :aria-label="t('alarm.refreshPolicies')"
          :disabled="loading"
          @click="emit('refresh')"
        >
          <IconTablerRefresh class="alarm-policy-manager__button-icon" />
        </button>
      </div>
    </header>

    <div class="alarm-policy-manager__filterbar">
      <el-input
        v-model="filters.search"
        class="alarm-policy-manager__search"
        clearable
        size="small"
        :placeholder="t('alarm.searchPlaceholder')"
        :prefix-icon="SearchIcon"
        @input="emitFilter"
      />
      <el-popover
        trigger="click"
        placement="bottom-start"
        :width="130"
        transition=""
        :hide-after="0"
        popper-class="alarm-policy-manager__popover"
      >
        <template #reference>
          <PillButton
            class="alarm-policy-manager__filter-pill"
            :active="filters.enabled !== ''"
          >
            {{ enabledLabel }}
          </PillButton>
        </template>
        <div class="alarm-policy-manager__pop-list">
          <button
            v-for="item in enabledOptions"
            :key="item.value"
            type="button"
            class="alarm-policy-manager__pop-item"
            :class="{ 'is-active': filters.enabled === item.value }"
            @click="selectEnabled(item.value)"
          >
            {{ item.label }}
          </button>
        </div>
      </el-popover>

      <el-popover
        trigger="click"
        placement="bottom-start"
        :width="170"
        transition=""
        :hide-after="0"
        popper-class="alarm-policy-manager__popover"
      >
        <template #reference>
          <PillButton
            class="alarm-policy-manager__filter-pill"
            :active="filters.conditionType !== ''"
          >
            {{ conditionTypeLabel }}
          </PillButton>
        </template>
        <div class="alarm-policy-manager__pop-list">
          <button
            v-for="item in conditionFilterOptions"
            :key="item.value"
            type="button"
            class="alarm-policy-manager__pop-item"
            :class="{ 'is-active': filters.conditionType === item.value }"
            @click="selectConditionType(item.value)"
          >
            {{ item.label }}
          </button>
        </div>
      </el-popover>
    </div>

    <div class="alarm-policy-manager__selectbar">
      <label class="alarm-policy-manager__select-control">
        <input
          type="checkbox"
          :checked="allVisibleSelected"
          :indeterminate.prop="someVisibleSelected && !allVisibleSelected"
          :disabled="!visiblePolicies.length"
          @change="toggleVisible(checked($event))"
        />
      </label>
      <div class="alarm-policy-manager__select-actions">
        <button
          v-if="hasFilters"
          type="button"
          :disabled="!visiblePolicies.length"
          @click="selectFiltered"
        >
          {{ t("alarm.selectMatched") }}
        </button>
        <button
          v-if="selectedCount"
          type="button"
          @click="emit('clearSelection')"
        >
          {{ t("alarm.cancelSelection") }}
        </button>
      </div>
    </div>

    <div v-if="error" class="alarm-policy-manager__state is-error">
      <IconTablerAlertCircle class="alarm-policy-manager__state-icon" />
      <span>{{ error }}</span>
      <button type="button" @click="emit('refresh')">
        {{ t("alarm.retry") }}
      </button>
    </div>
    <div v-else-if="loading" class="alarm-policy-manager__loading">
      <el-skeleton :rows="8" animated />
    </div>

    <div v-else class="alarm-policy-manager__body">
      <AlarmPolicyGroupBranch
        v-for="group in policyTree.groups"
        :key="group.id"
        :node="group"
        :selection="selection"
        :selected-id="selectedId"
        @select="emit('select', $event)"
        @select-policy="(id, selected) => emit('selectPolicy', id, selected)"
        @select-group="(ids, selected) => emit('selectGroup', ids, selected)"
        @policy-contextmenu="
          (mouseEvent, policy) => openPolicyMenu(mouseEvent, policy)
        "
        @group-contextmenu="
          (mouseEvent, group) => openGroupMenu(mouseEvent, group)
        "
      />

      <button
        v-for="policy in policyTree.rootPolicies"
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
        <IconTablerBell class="alarm-policy-manager__row-icon" />
        <span class="alarm-policy-manager__row-main">
          <span class="alarm-policy-manager__row-name">{{ policy.name }}</span>
        </span>
        <span
          class="alarm-policy-manager__row-status"
          :class="policy.effectiveEnabled ? 'is-success' : 'is-muted'"
          :title="policy.isEnabled ? t('common.enabled') : t('alarm.stopped')"
          :aria-label="
            policy.isEnabled ? t('common.enabled') : t('alarm.stopped')
          "
        ></span>
      </button>

      <EmptyState
        v-if="
          policyTree.groups.length === 0 && policyTree.rootPolicies.length === 0
        "
        icon-name="alarm"
        :title="t('alarm.emptyPolicies')"
        :description="t('alarm.emptyPoliciesHint')"
      />
    </div>

    <div v-if="selectedCount" class="alarm-policy-manager__bulk">
      <button
        type="button"
        :disabled="!selectedCount"
        @click="emit('batchEnable')"
      >
        {{ t("common.enabled") }}
      </button>
      <button
        type="button"
        :disabled="!selectedCount"
        @click="emit('batchDisable')"
      >
        {{ t("alarm.stopped") }}
      </button>
      <button
        type="button"
        :disabled="!selectedCount"
        @click="emit('batchConditions')"
      >
        {{ t("alarm.conditions") }}
      </button>
      <button
        type="button"
        :disabled="!selectedCount"
        @click="emit('batchMoveDialog')"
      >
        {{ t("alarm.move") }}
      </button>
      <button
        type="button"
        class="alarm-policy-manager__bulk-danger"
        :disabled="!selectedCount"
        @click="emit('batchDelete')"
      >
        {{ t("actions.delete") }}
      </button>
    </div>

    <footer class="alarm-policy-manager__foot">
      <span>{{ t("alarm.policyCount", { count: total }) }}</span>
      <span v-if="selectedCount">
        {{ t("alarm.selectedItemCount", { count: selectedCount }) }}
      </span>
    </footer>

    <Teleport to="body">
      <div
        v-if="contextMenu.visible"
        class="alarm-policy-manager__menu-mask"
        @click="closeContextMenu"
        @contextmenu.prevent="closeContextMenu"
      >
        <div
          class="alarm-policy-manager__context-menu"
          :style="{ left: `${contextMenu.x}px`, top: `${contextMenu.y}px` }"
          @click.stop
        >
          <button type="button" @click="emitContextAction('rename')">
            <IconTablerPencil class="alarm-policy-manager__menu-icon" />
            <span>{{ t("alarm.rename") }}</span>
          </button>
          <button type="button" @click="emitContextAction('move')">
            <IconTablerFolderSymlink class="alarm-policy-manager__menu-icon" />
            <span>{{ t("alarm.moveToGroup") }}</span>
          </button>
          <button
            type="button"
            class="is-danger"
            @click="emitContextAction('delete')"
          >
            <IconTablerTrash class="alarm-policy-manager__menu-icon" />
            <span>删除</span>
          </button>
        </div>
      </div>
    </Teleport>
  </aside>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { Search } from "@element-plus/icons-vue";
import IconTablerAlertCircle from "~icons/tabler/alert-circle";
import IconTablerBell from "~icons/tabler/bell";
import IconTablerFolderPlus from "~icons/tabler/folder-plus";
import IconTablerFolderSymlink from "~icons/tabler/folder-symlink";
import IconTablerPencil from "~icons/tabler/pencil";
import IconTablerPlus from "~icons/tabler/plus";
import IconTablerRefresh from "~icons/tabler/refresh";
import IconTablerTrash from "~icons/tabler/trash";
import type {
  AlarmBulkSelection,
  AlarmPolicy,
  AlarmPolicyGroup,
  AlarmPolicyTree,
} from "@/api/schemas/alarm.schema";
import EmptyState from "@/components/shared/EmptyState.vue";
import PillButton from "@/components/shared/PillButton.vue";
import { t } from "@/i18n/runtime";
import AlarmPolicyGroupBranch from "./AlarmPolicyGroupBranch.vue";
import {
  buildAlarmPolicyGroupTree,
  type AlarmPolicyGroupNode,
} from "./alarmPolicyTreeModel";

type ContextMenuState = {
  visible: boolean;
  type: "policy" | "group" | null;
  x: number;
  y: number;
  policy: AlarmPolicy | null;
  group: AlarmPolicyGroupNode | null;
};

const SearchIcon = Search;

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
  batchMoveDialog: [];
  batchDelete: [];
  renamePolicy: [policy: AlarmPolicy];
  movePolicy: [policy: AlarmPolicy];
  deletePolicy: [policy: AlarmPolicy];
  renameGroup: [group: AlarmPolicyGroup];
  moveGroup: [group: AlarmPolicyGroup];
  deleteGroup: [group: AlarmPolicyGroupNode];
}>();

const filters = reactive({
  search: "",
  enabled: "",
  conditionType: "",
});

const contextMenu = ref<ContextMenuState>({
  visible: false,
  type: null,
  x: 0,
  y: 0,
  policy: null,
  group: null,
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
const conditionFilterOptions = computed(() => [
  { value: "", label: t("alarm.all") },
  ...conditionTypeOptions.value,
]);
const enabledLabel = computed(() => {
  const found = enabledOptions.value.find(
    (item) => item.value === filters.enabled,
  );
  return found && filters.enabled ? found.label : t("alarm.status");
});
const conditionTypeLabel = computed(() => {
  const found = conditionFilterOptions.value.find(
    (item) => item.value === filters.conditionType,
  );
  return found && filters.conditionType ? found.label : t("alarm.type");
});

const visiblePolicies = computed(() => props.tree.policies);

const policyTree = computed(() =>
  buildAlarmPolicyGroupTree(props.tree.groups, props.tree.policies),
);

const checked = (event: Event) => (event.target as HTMLInputElement).checked;

const cleanFilters = () =>
  Object.fromEntries(
    Object.entries(filters).filter(([, value]) => value.trim() !== ""),
  );
const hasFilters = computed(() => Object.keys(cleanFilters()).length > 0);

const emitFilter = () => {
  emit("filter", cleanFilters());
};

const selectEnabled = (value: string) => {
  filters.enabled = value;
  emitFilter();
};

const selectConditionType = (value: string) => {
  filters.conditionType = value;
  emitFilter();
};

const isSelected = (id: string) => {
  if (props.selection.mode === "filtered") {
    return !props.selection.excludePolicyIds.includes(id);
  }
  return props.selection.policyIds.includes(id);
};

const allVisibleSelected = computed(
  () =>
    visiblePolicies.value.length > 0 &&
    visiblePolicies.value.every((item) => isSelected(item.id)),
);
const someVisibleSelected = computed(() =>
  visiblePolicies.value.some((item) => isSelected(item.id)),
);

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

const openPolicyMenu = (event: MouseEvent, policy: AlarmPolicy) => {
  contextMenu.value = {
    visible: true,
    type: "policy",
    x: Math.min(event.clientX, window.innerWidth - 180),
    y: Math.min(event.clientY, window.innerHeight - 126),
    policy,
    group: null,
  };
};

const openGroupMenu = (event: MouseEvent, group: AlarmPolicyGroupNode) => {
  contextMenu.value = {
    visible: true,
    type: "group",
    x: Math.min(event.clientX, window.innerWidth - 180),
    y: Math.min(event.clientY, window.innerHeight - 126),
    policy: null,
    group,
  };
};

const closeContextMenu = () => {
  contextMenu.value.visible = false;
};

const emitContextAction = (action: "rename" | "move" | "delete") => {
  const current = contextMenu.value;
  closeContextMenu();
  if (current.type === "policy" && current.policy) {
    if (action === "rename") {
      emit("renamePolicy", current.policy);
      return;
    }
    if (action === "delete") {
      emit("deletePolicy", current.policy);
      return;
    }
    emit("movePolicy", current.policy);
    return;
  }
  if (current.type === "group" && current.group) {
    if (action === "rename") {
      emit("renameGroup", current.group);
      return;
    }
    if (action === "delete") {
      emit("deleteGroup", current.group);
      return;
    }
    emit("moveGroup", current.group);
  }
};
</script>

<style scoped>
.alarm-policy-manager {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border-right: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text);
}

.alarm-policy-manager__head {
  min-height: 46px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--dc-border);
}

.alarm-policy-manager__head h2 {
  margin: 0;
  color: var(--dc-text);
  font-size: 15px;
  font-weight: 800;
  line-height: 1.3;
}

.alarm-policy-manager button,
.alarm-policy-manager input,
.alarm-policy-manager select {
  font-family: inherit;
}

.alarm-policy-manager__title {
  min-width: 0;
}

.alarm-policy-manager__actions {
  display: flex;
  align-items: center;
  gap: 6px;
  flex: 0 0 auto;
}

.alarm-policy-manager__primary,
.alarm-policy-manager__secondary {
  min-width: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--dc-radius-sm);
  font-size: 13px;
  font-weight: 700;
  transition:
    border-color 0.18s ease,
    background-color 0.18s ease,
    color 0.18s ease,
    transform 0.18s ease;
}

.alarm-policy-manager__primary,
.alarm-policy-manager__secondary {
  width: 30px;
  height: 30px;
  padding: 0;
}

.alarm-policy-manager__primary {
  border: 1px solid var(--dc-primary);
  background: var(--dc-primary);
  color: var(--dc-surface-raised);
}

.alarm-policy-manager__secondary {
  border: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}

.alarm-policy-manager__primary:hover,
.alarm-policy-manager__secondary:hover {
  transform: translateY(-1px);
}

.alarm-policy-manager__secondary:hover {
  border-color: rgba(29, 78, 216, 0.28);
  color: var(--dc-primary);
}

.alarm-policy-manager__button-icon,
.alarm-policy-manager__state-icon,
.alarm-policy-manager__row-icon {
  width: 16px;
  height: 16px;
}

.alarm-policy-manager__filterbar {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px;
  background: var(--dc-surface-raised);
}

.alarm-policy-manager__search {
  min-width: 0;
  flex: 1 1 auto;
}

.alarm-policy-manager__filter-pill {
  flex: 0 0 auto;
  height: 24px;
  max-width: 72px;
  padding: 0 7px;
  border-radius: 9px;
  font-size: 12px;
  font-weight: 600;
}

.alarm-policy-manager__pop-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 4px 0;
}

.alarm-policy-manager__pop-item {
  width: 100%;
  padding: 7px 12px;
  border: none;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  cursor: pointer;
  font-size: 13px;
  text-align: left;
  transition:
    background-color 0.14s ease,
    color 0.14s ease;
}

.alarm-policy-manager__pop-item:hover {
  background: var(--dc-surface-muted);
  color: var(--dc-text);
}

.alarm-policy-manager__pop-item.is-active {
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
  font-weight: 700;
}

.alarm-policy-manager__selectbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-height: 28px;
  padding: 3px 8px 3px 12px;
  background: var(--dc-surface-raised);
}

.alarm-policy-manager__select-control {
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
}

.alarm-policy-manager__select-control input {
  margin: 0;
}

.alarm-policy-manager__select-actions {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex: 0 0 auto;
}

.alarm-policy-manager__selectbar button,
.alarm-policy-manager__bulk button,
.alarm-policy-manager__state button {
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-text-secondary);
  cursor: pointer;
  font-size: 12px;
  font-weight: 700;
  padding: 0 8px;
  white-space: nowrap;
}

.alarm-policy-manager__selectbar button {
  height: auto;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--dc-primary);
  font-weight: 700;
}

.alarm-policy-manager__selectbar button:hover:not(:disabled) {
  color: var(--dc-primary);
}

.alarm-policy-manager__bulk {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  align-items: center;
  gap: 5px;
  flex: 0 0 auto;
  padding: 6px 8px;
  background: var(--dc-surface-raised);
}

.alarm-policy-manager__bulk button {
  width: 100%;
  height: 26px;
  min-width: 0;
  padding: 0 4px;
}

.alarm-policy-manager__bulk-danger {
  color: var(--dc-danger);
}

.alarm-policy-manager button:disabled,
.alarm-policy-manager select:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.alarm-policy-manager__body {
  min-height: 0;
  flex: 1;
  display: grid;
  align-content: start;
  gap: 2px;
  padding: 6px;
  overflow-y: auto;
}

.alarm-policy-manager__loading {
  padding: 12px;
}

.alarm-policy-manager__row {
  width: 100%;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
}

.alarm-policy-manager__row:hover {
  border-color: var(--dc-border);
  background: var(--dc-surface-muted);
  color: var(--dc-text);
}

.alarm-policy-manager__row-icon {
  color: var(--dc-primary);
}

.alarm-policy-manager__row-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.alarm-policy-manager__row {
  min-height: 30px;
  display: grid;
  grid-template-columns: auto 18px minmax(0, 1fr) auto;
  align-items: center;
  gap: 6px;
  padding: 3px 6px;
  text-align: left;
}

.alarm-policy-manager__row.is-active {
  border-color: rgba(29, 78, 216, 0.28);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.alarm-policy-manager__row.is-disabled {
  opacity: 0.68;
}

.alarm-policy-manager__row-main {
  min-width: 0;
  display: block;
}

.alarm-policy-manager__row-name {
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 700;
}

.alarm-policy-manager__row-status {
  width: 8px;
  height: 8px;
  justify-self: end;
  border-radius: 999px;
  background: var(--dc-text-muted);
}

.alarm-policy-manager__row-status.is-success {
  background: var(--dc-success);
}

.alarm-policy-manager__row-status.is-muted {
  background: var(--dc-text-muted);
}

.alarm-policy-manager__foot {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 7px 10px;
  color: var(--dc-text-muted);
  font-size: 12px;
  line-height: 1.4;
}

.alarm-policy-manager__menu-mask {
  position: fixed;
  inset: 0;
  z-index: 2100;
}

.alarm-policy-manager__context-menu {
  position: fixed;
  min-width: 148px;
  padding: 4px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
}

.alarm-policy-manager__context-menu button {
  width: 100%;
  height: 30px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 8px;
  border: 0;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  font-size: 13px;
  text-align: left;
}

.alarm-policy-manager__context-menu button:hover {
  background: var(--dc-surface-muted);
  color: var(--dc-primary);
}

.alarm-policy-manager__context-menu button.is-danger:hover {
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
}

.alarm-policy-manager__menu-icon {
  width: 15px;
  height: 15px;
  flex: 0 0 auto;
}

.alarm-policy-manager__state {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  align-items: start;
  gap: 8px;
  margin: 10px 12px 0;
  padding: 9px 10px;
  border: 1px solid rgba(220, 38, 38, 0.2);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
  font-size: 12px;
  line-height: 1.5;
}

.alarm-policy-manager__state button {
  grid-column: 2;
  justify-self: start;
  color: var(--dc-danger);
}

@media (max-width: 920px) {
  .alarm-policy-manager {
    max-height: 380px;
    border-right: 0;
    border-bottom: 1px solid var(--dc-border);
  }
}
</style>
