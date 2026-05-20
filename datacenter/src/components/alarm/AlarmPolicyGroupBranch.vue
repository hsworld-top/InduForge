<template>
  <section class="alarm-policy-branch">
    <button
      type="button"
      class="alarm-policy-branch__group"
      @click="expanded = !expanded"
      @contextmenu.prevent.stop="$emit('groupContextmenu', $event, node)"
    >
      <IconTablerChevronRight
        class="alarm-policy-branch__chevron"
        :class="{ 'is-open': expanded }"
      />
      <input
        type="checkbox"
        :checked="groupAllSelected(node.policyIds)"
        :indeterminate.prop="
          groupSomeSelected(node.policyIds) && !groupAllSelected(node.policyIds)
        "
        :disabled="!node.policyIds.length"
        @click.stop
        @change="$emit('selectGroup', node.policyIds, checked($event))"
      />
      <IconTablerFolderOpen
        v-if="expanded"
        class="alarm-policy-branch__folder-icon"
      />
      <IconTablerFolder v-else class="alarm-policy-branch__folder-icon" />
      <span class="alarm-policy-branch__group-name">{{ node.name }}</span>
      <span class="alarm-policy-branch__count">{{
        node.policyIds.length
      }}</span>
    </button>

    <div v-if="expanded" class="alarm-policy-branch__children">
      <button
        v-for="policy in node.policies"
        :key="policy.id"
        type="button"
        class="alarm-policy-branch__row"
        :class="{
          'is-active': policy.id === selectedId,
          'is-disabled': !policy.effectiveEnabled,
        }"
        @click="$emit('select', policy.id)"
        @contextmenu.prevent.stop="$emit('policyContextmenu', $event, policy)"
      >
        <input
          type="checkbox"
          :checked="isSelected(policy.id)"
          @click.stop
          @change="$emit('selectPolicy', policy.id, checked($event))"
        />
        <IconTablerBell class="alarm-policy-branch__row-icon" />
        <span class="alarm-policy-branch__row-main">
          <span class="alarm-policy-branch__row-name">{{ policy.name }}</span>
        </span>
        <StatusBadge
          class="alarm-policy-branch__row-status"
          :tone="policy.effectiveEnabled ? 'success' : 'muted'"
          :text="policy.isEnabled ? t('common.enabled') : t('alarm.stopped')"
        />
      </button>

      <AlarmPolicyGroupBranch
        v-for="child in node.children"
        :key="child.id"
        :node="child"
        :selection="selection"
        :selected-id="selectedId"
        @select="$emit('select', $event)"
        @select-policy="(id, selected) => $emit('selectPolicy', id, selected)"
        @select-group="(ids, selected) => $emit('selectGroup', ids, selected)"
        @policy-contextmenu="
          (mouseEvent, policy) => $emit('policyContextmenu', mouseEvent, policy)
        "
        @group-contextmenu="
          (mouseEvent, group) => $emit('groupContextmenu', mouseEvent, group)
        "
      />
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref } from "vue";
import IconTablerBell from "~icons/tabler/bell";
import IconTablerChevronRight from "~icons/tabler/chevron-right";
import IconTablerFolder from "~icons/tabler/folder";
import IconTablerFolderOpen from "~icons/tabler/folder-open";
import type {
  AlarmBulkSelection,
  AlarmPolicy,
} from "@/api/schemas/alarm.schema";
import StatusBadge from "@/components/shared/StatusBadge.vue";
import { t } from "@/i18n/runtime";
import type { AlarmPolicyGroupNode } from "./alarmPolicyTreeModel";

defineOptions({ name: "AlarmPolicyGroupBranch" });

const props = defineProps<{
  node: AlarmPolicyGroupNode;
  selection: AlarmBulkSelection;
  selectedId: string;
}>();

defineEmits<{
  select: [id: string];
  selectPolicy: [id: string, selected: boolean];
  selectGroup: [ids: string[], selected: boolean];
  policyContextmenu: [mouseEvent: MouseEvent, policy: AlarmPolicy];
  groupContextmenu: [mouseEvent: MouseEvent, group: AlarmPolicyGroupNode];
}>();

const expanded = ref(true);

const checked = (event: Event) => (event.target as HTMLInputElement).checked;

const isSelected = (id: string) => {
  if (props.selection.mode === "filtered") {
    return !props.selection.excludePolicyIds.includes(id);
  }
  return props.selection.policyIds.includes(id);
};

const groupAllSelected = (ids: string[]) =>
  ids.length > 0 && ids.every((id) => isSelected(id));
const groupSomeSelected = (ids: string[]) => ids.some((id) => isSelected(id));
</script>

<style scoped>
.alarm-policy-branch {
  display: grid;
  gap: 2px;
}

.alarm-policy-branch__group,
.alarm-policy-branch__row {
  width: 100%;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
}

.alarm-policy-branch__group {
  min-height: 28px;
  display: grid;
  grid-template-columns: 16px auto 18px minmax(0, 1fr) auto;
  align-items: center;
  gap: 5px;
  padding: 0 6px;
  font-size: 13px;
  font-weight: 700;
  text-align: left;
}

.alarm-policy-branch__group:hover,
.alarm-policy-branch__row:hover {
  border-color: var(--dc-border);
  background: var(--dc-surface-muted);
  color: var(--dc-text);
}

.alarm-policy-branch__chevron {
  width: 14px;
  height: 14px;
  transform: rotate(0deg);
  transition: transform 0.16s ease;
}

.alarm-policy-branch__chevron.is-open {
  transform: rotate(90deg);
}

.alarm-policy-branch__folder-icon,
.alarm-policy-branch__row-icon {
  width: 16px;
  height: 16px;
  color: var(--dc-primary);
}

.alarm-policy-branch__group-name,
.alarm-policy-branch__row-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.alarm-policy-branch__count {
  min-width: 20px;
  padding: 2px 6px;
  border-radius: 999px;
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
  font-size: 11px;
  text-align: center;
}

.alarm-policy-branch__children {
  display: grid;
  gap: 2px;
  margin-left: 10px;
  padding-left: 6px;
}

.alarm-policy-branch__row {
  min-height: 30px;
  display: grid;
  grid-template-columns: auto 18px minmax(0, 1fr) auto;
  align-items: center;
  gap: 6px;
  padding: 3px 6px;
  text-align: left;
}

.alarm-policy-branch__row.is-active {
  border-color: rgba(29, 78, 216, 0.28);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.alarm-policy-branch__row.is-disabled {
  opacity: 0.68;
}

.alarm-policy-branch__row-main {
  min-width: 0;
  display: block;
}

.alarm-policy-branch__row-name {
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 700;
}

.alarm-policy-branch__row-status {
  max-width: 76px;
}
</style>
