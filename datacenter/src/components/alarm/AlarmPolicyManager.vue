<template>
  <aside class="alarm-policy-manager" :class="{ 'is-collapsed': collapsed }">
    <header class="alarm-policy-manager__head">
      <div v-if="!collapsed">
        <h2>{{ t("alarm.policies") }}</h2>
        <p>{{ t("alarm.policyCount", { count: total }) }}</p>
      </div>
      <button
        type="button"
        class="alarm-policy-manager__icon-btn"
        :title="collapsed ? t('actions.expand') : t('actions.collapse')"
        :aria-label="collapsed ? t('actions.expand') : t('actions.collapse')"
        @click="collapsed = !collapsed"
      >
        <IconTablerLayoutSidebarLeftCollapse
          v-if="!collapsed"
          class="alarm-policy-manager__icon"
        />
        <IconTablerLayoutSidebarLeftExpand
          v-else
          class="alarm-policy-manager__icon"
        />
      </button>
    </header>

    <div v-if="collapsed" class="alarm-policy-manager__rail">
      <button
        type="button"
        class="alarm-policy-manager__rail-btn"
        :title="t('alarm.createPolicy')"
        :aria-label="t('alarm.createPolicy')"
        @click="emit('create')"
      >
        <IconTablerPlus class="alarm-policy-manager__button-icon" />
      </button>
      <button
        type="button"
        class="alarm-policy-manager__rail-btn"
        :title="t('alarm.createGroup')"
        :aria-label="t('alarm.createGroup')"
        @click="emit('createGroup')"
      >
        <IconTablerFolderPlus class="alarm-policy-manager__button-icon" />
      </button>
      <button
        type="button"
        class="alarm-policy-manager__rail-btn"
        :title="t('actions.refresh')"
        :aria-label="t('alarm.refreshPolicies')"
        :disabled="loading"
        @click="emit('refresh')"
      >
        <IconTablerRefresh class="alarm-policy-manager__button-icon" />
      </button>
    </div>

    <template v-else>
      <div class="alarm-policy-manager__toolbar">
        <el-input
          v-model="filters.search"
          class="alarm-policy-manager__search"
          clearable
          size="small"
          :placeholder="t('alarm.searchPlaceholder')"
          :prefix-icon="SearchIcon"
          @input="emitFilter"
        />
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

      <div class="alarm-policy-manager__filters">
        <label>
          <span>{{ t("alarm.status") }}</span>
          <select v-model="filters.enabled" @change="emitFilter">
            <option
              v-for="item in enabledOptions"
              :key="item.value"
              :value="item.value"
            >
              {{ item.label }}
            </option>
          </select>
        </label>
        <label>
          <span>{{ t("alarm.type") }}</span>
          <select v-model="filters.conditionType" @change="emitFilter">
            <option value="">{{ t("alarm.all") }}</option>
            <option
              v-for="item in conditionTypeOptions"
              :key="item.value"
              :value="item.value"
            >
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
        <button
          type="button"
          :disabled="!visiblePolicies.length"
          @click="selectFiltered"
        >
          {{ t("alarm.selectFiltered") }}
        </button>
        <button
          v-if="selectedCount"
          type="button"
          @click="emit('clearSelection')"
        >
          {{ t("actions.clean") }}
        </button>
      </div>

      <div v-if="selectedCount" class="alarm-policy-manager__bulk">
        <strong>{{
          t("alarm.selectedCount", { count: selectedCount })
        }}</strong>
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
          {{ t("alarm.batchConditions") }}
        </button>
        <select
          :disabled="!selectedCount"
          :aria-label="t('alarm.batchMoveGroup')"
          @change="moveSelected(value($event))"
        >
          <option value="">{{ t("alarm.moveTo") }}</option>
          <option value="__root__">{{ t("alarm.root") }}</option>
          <option
            v-for="group in tree.groups"
            :key="group.id"
            :value="group.id"
          >
            {{ group.name }}
          </option>
        </select>
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
        <template v-for="node in nodes" :key="node.key">
          <section
            v-if="node.type === 'group'"
            class="alarm-policy-manager__group"
          >
            <button
              type="button"
              class="alarm-policy-manager__group-head"
              @click="toggleGroup(node.group.id)"
              @contextmenu.prevent.stop="openGroupMenu($event, node)"
            >
              <IconTablerChevronRight
                class="alarm-policy-manager__chevron"
                :class="{ 'is-open': isGroupOpen(node.group.id) }"
              />
              <input
                type="checkbox"
                :checked="groupAllSelected(node.policyIds)"
                :indeterminate.prop="
                  groupSomeSelected(node.policyIds) &&
                  !groupAllSelected(node.policyIds)
                "
                :disabled="!node.policyIds.length"
                @click.stop
                @change="emit('selectGroup', node.policyIds, checked($event))"
              />
              <IconTablerFolderOpen
                v-if="isGroupOpen(node.group.id)"
                class="alarm-policy-manager__folder-icon"
              />
              <IconTablerFolder
                v-else
                class="alarm-policy-manager__folder-icon"
              />
              <span class="alarm-policy-manager__group-name">{{
                node.group.name
              }}</span>
              <span class="alarm-policy-manager__count">{{
                node.policies.length
              }}</span>
            </button>

            <div
              v-if="isGroupOpen(node.group.id)"
              class="alarm-policy-manager__children"
            >
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
                <IconTablerBell class="alarm-policy-manager__row-icon" />
                <span class="alarm-policy-manager__row-main">
                  <span class="alarm-policy-manager__row-name">{{
                    policy.name
                  }}</span>
                  <span class="alarm-policy-manager__row-path">
                    {{ policySummary(policy) }}
                  </span>
                </span>
                <StatusBadge
                  class="alarm-policy-manager__row-status"
                  :tone="policy.effectiveEnabled ? 'success' : 'muted'"
                  :text="
                    policy.isEnabled ? t('common.enabled') : t('alarm.stopped')
                  "
                />
              </button>
            </div>
          </section>

          <button
            v-else
            type="button"
            class="alarm-policy-manager__row"
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
            <IconTablerBell class="alarm-policy-manager__row-icon" />
            <span class="alarm-policy-manager__row-main">
              <span class="alarm-policy-manager__row-name">{{
                node.policy.name
              }}</span>
              <span class="alarm-policy-manager__row-path">
                {{ policySummary(node.policy) }}
              </span>
            </span>
            <StatusBadge
              class="alarm-policy-manager__row-status"
              :tone="node.policy.effectiveEnabled ? 'success' : 'muted'"
              :text="
                node.policy.isEnabled ? t('common.enabled') : t('alarm.stopped')
              "
            />
          </button>
        </template>

        <EmptyState
          v-if="nodes.length === 0"
          icon-name="alarm"
          :title="t('alarm.emptyPolicies')"
          :description="t('alarm.emptyPoliciesHint')"
        />
      </div>
    </template>

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
import IconTablerChevronRight from "~icons/tabler/chevron-right";
import IconTablerFolder from "~icons/tabler/folder";
import IconTablerFolderOpen from "~icons/tabler/folder-open";
import IconTablerFolderPlus from "~icons/tabler/folder-plus";
import IconTablerFolderSymlink from "~icons/tabler/folder-symlink";
import IconTablerLayoutSidebarLeftCollapse from "~icons/tabler/layout-sidebar-left-collapse";
import IconTablerLayoutSidebarLeftExpand from "~icons/tabler/layout-sidebar-left-expand";
import IconTablerPencil from "~icons/tabler/pencil";
import IconTablerPlus from "~icons/tabler/plus";
import IconTablerRefresh from "~icons/tabler/refresh";
import type {
  AlarmBulkSelection,
  AlarmPolicy,
  AlarmPolicyGroup,
  AlarmPolicyTree,
} from "@/api/schemas/alarm.schema";
import EmptyState from "@/components/shared/EmptyState.vue";
import StatusBadge from "@/components/shared/StatusBadge.vue";
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

type ContextMenuState = {
  visible: boolean;
  type: "policy" | "group" | null;
  x: number;
  y: number;
  policy: AlarmPolicy | null;
  group: GroupNode | null;
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
  renamePolicy: [policy: AlarmPolicy];
  movePolicy: [policy: AlarmPolicy];
  renameGroup: [group: AlarmPolicyGroup];
  moveGroupPolicies: [group: AlarmPolicyGroup, policyIds: string[]];
}>();

const collapsed = ref(false);
const openedGroupIds = ref<string[]>([]);
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

const visiblePolicies = computed(() => props.tree.policies);

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

  const result: Array<GroupNode | PolicyNode> = props.tree.groups.map(
    (group) => {
      const policies = byGroup.get(group.id) ?? [];
      return {
        type: "group" as const,
        key: `group-${group.id}`,
        group,
        policies,
        policyIds: policies.map((policy) => policy.id),
      };
    },
  );

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

const isGroupOpen = (id: string) => !openedGroupIds.value.includes(id);

const toggleGroup = (id: string) => {
  openedGroupIds.value = isGroupOpen(id)
    ? [...openedGroupIds.value, id]
    : openedGroupIds.value.filter((item) => item !== id);
};

const openPolicyMenu = (event: MouseEvent, policy: AlarmPolicy) => {
  contextMenu.value = {
    visible: true,
    type: "policy",
    x: Math.min(event.clientX, window.innerWidth - 180),
    y: Math.min(event.clientY, window.innerHeight - 96),
    policy,
    group: null,
  };
};

const openGroupMenu = (event: MouseEvent, group: GroupNode) => {
  contextMenu.value = {
    visible: true,
    type: "group",
    x: Math.min(event.clientX, window.innerWidth - 180),
    y: Math.min(event.clientY, window.innerHeight - 96),
    policy: null,
    group,
  };
};

const closeContextMenu = () => {
  contextMenu.value.visible = false;
};

const emitContextAction = (action: "rename" | "move") => {
  const current = contextMenu.value;
  closeContextMenu();
  if (current.type === "policy" && current.policy) {
    if (action === "rename") {
      emit("renamePolicy", current.policy);
      return;
    }
    emit("movePolicy", current.policy);
    return;
  }
  if (current.type === "group" && current.group) {
    if (action === "rename") {
      emit("renameGroup", current.group.group);
      return;
    }
    emit("moveGroupPolicies", current.group.group, current.group.policyIds);
  }
};

const policySummary = (policy: AlarmPolicy) => {
  if (policy.mode === "derived") {
    return policy.derivedExpression || "计算后判断";
  }
  return policy.targets.map((target) => target.path).join("、") || "逐点判断";
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

.alarm-policy-manager.is-collapsed .alarm-policy-manager__head {
  justify-content: center;
  padding: 8px;
}

.alarm-policy-manager__head h2 {
  margin: 0;
  color: var(--dc-text);
  font-size: 15px;
  font-weight: 800;
  line-height: 1.3;
}

.alarm-policy-manager__head p {
  margin: 2px 0 0;
  color: var(--dc-text-muted);
  font-size: 12px;
}

.alarm-policy-manager button,
.alarm-policy-manager input,
.alarm-policy-manager select {
  font-family: inherit;
}

.alarm-policy-manager__toolbar {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 30px 30px 30px;
  gap: 6px;
  padding: 8px;
  border-bottom: 1px solid var(--dc-border);
}

.alarm-policy-manager__primary,
.alarm-policy-manager__secondary,
.alarm-policy-manager__icon-btn {
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

.alarm-policy-manager__secondary,
.alarm-policy-manager__icon-btn {
  border: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}

.alarm-policy-manager__primary:hover,
.alarm-policy-manager__secondary:hover,
.alarm-policy-manager__icon-btn:hover {
  transform: translateY(-1px);
}

.alarm-policy-manager__secondary:hover,
.alarm-policy-manager__icon-btn:hover {
  border-color: rgba(29, 78, 216, 0.28);
  color: var(--dc-primary);
}

.alarm-policy-manager__icon-btn {
  width: 30px;
  height: 30px;
}

.alarm-policy-manager__rail {
  display: grid;
  justify-items: center;
  gap: 8px;
  padding: 10px 8px;
}

.alarm-policy-manager__rail-btn {
  width: 32px;
  height: 32px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}

.alarm-policy-manager__rail-btn:hover {
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.alarm-policy-manager__icon,
.alarm-policy-manager__button-icon,
.alarm-policy-manager__state-icon,
.alarm-policy-manager__row-icon,
.alarm-policy-manager__folder-icon {
  width: 16px;
  height: 16px;
}

.alarm-policy-manager__filters {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 8px;
  padding: 8px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
}

.alarm-policy-manager__filters label {
  min-width: 0;
  display: grid;
  gap: 4px;
}

.alarm-policy-manager__filters span {
  color: var(--dc-text-muted);
  font-size: 11px;
  font-weight: 700;
}

.alarm-policy-manager__filters select,
.alarm-policy-manager__bulk select {
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

.alarm-policy-manager__selectbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
  padding: 7px 8px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
}

.alarm-policy-manager__selectbar label {
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.alarm-policy-manager__selectbar button,
.alarm-policy-manager__bulk button,
.alarm-policy-manager__bulk select,
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

.alarm-policy-manager__bulk {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  padding: 8px;
  border-bottom: 1px solid var(--dc-border);
}

.alarm-policy-manager__bulk strong {
  color: var(--dc-text);
  font-size: 12px;
  margin-right: 2px;
}

.alarm-policy-manager__bulk select {
  width: auto;
  min-width: 112px;
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
  gap: 4px;
  padding: 8px;
  overflow-y: auto;
}

.alarm-policy-manager__loading {
  padding: 12px;
}

.alarm-policy-manager__group {
  display: grid;
  gap: 4px;
}

.alarm-policy-manager__group-head,
.alarm-policy-manager__row {
  width: 100%;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
}

.alarm-policy-manager__group-head {
  min-height: 32px;
  display: grid;
  grid-template-columns: 16px auto 18px minmax(0, 1fr) auto;
  align-items: center;
  gap: 6px;
  padding: 0 8px;
  font-size: 13px;
  font-weight: 700;
  text-align: left;
}

.alarm-policy-manager__group-head:hover,
.alarm-policy-manager__row:hover {
  border-color: var(--dc-border);
  background: var(--dc-surface-muted);
  color: var(--dc-text);
}

.alarm-policy-manager__chevron {
  width: 14px;
  height: 14px;
  transform: rotate(0deg);
  transition: transform 0.16s ease;
}

.alarm-policy-manager__chevron.is-open {
  transform: rotate(90deg);
}

.alarm-policy-manager__folder-icon,
.alarm-policy-manager__row-icon {
  color: var(--dc-primary);
}

.alarm-policy-manager__group-name,
.alarm-policy-manager__row-name,
.alarm-policy-manager__row-path {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.alarm-policy-manager__count {
  min-width: 20px;
  padding: 2px 6px;
  border-radius: 999px;
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
  font-size: 11px;
  text-align: center;
}

.alarm-policy-manager__children {
  display: grid;
  gap: 4px;
  margin-left: 12px;
  padding-left: 8px;
  border-left: 1px solid var(--dc-border);
}

.alarm-policy-manager__row {
  min-height: 44px;
  display: grid;
  grid-template-columns: auto 18px minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
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
  display: grid;
  gap: 3px;
}

.alarm-policy-manager__row-name {
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 700;
}

.alarm-policy-manager__row-path {
  color: var(--dc-text-muted);
  font-size: 11px;
}

.alarm-policy-manager__row-status {
  max-width: 76px;
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
