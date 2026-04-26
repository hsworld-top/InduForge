<script setup lang="ts">
import type { PagePermissionScheme } from "@/editor-core/document/types";
import type { RuntimeRoleRecord } from "@/stores/editor/project-runtime-role-actions";
import type { PageInspectorFormState } from "./page-inspector-types";
import { computed } from "vue";

const props = defineProps<{
  form: PageInspectorFormState;
  runtimeRoles: RuntimeRoleRecord[];
}>();

const emit = defineEmits<{
  updateConfig: [];
}>();

const roleOptions = computed(() =>
  props.runtimeRoles
    .filter((role) => role.status !== "disabled")
    .map((role) => ({
      label: role.name || role.code || role.id,
      value: role.id,
    })),
);

function createSchemeId(): string {
  const random = globalThis.crypto?.randomUUID?.() || `${Date.now()}_${Math.random()}`;
  return `scheme_${random.replace(/-/g, "").slice(0, 12)}`;
}

function addScheme(): void {
  const schemes = props.form.runtimePermissionSchemes || [];
  props.form.runtimePermissionSchemes = [
    ...schemes,
    {
      id: createSchemeId(),
      name: `方案 ${schemes.length + 1}`,
      roleRefs: [],
    },
  ];
  emit("updateConfig");
}

function removeScheme(schemeId: string): void {
  props.form.runtimePermissionSchemes = (props.form.runtimePermissionSchemes || []).filter(
    (scheme) => scheme.id !== schemeId,
  );
  emit("updateConfig");
}

function updateSchemeRoleRefs(scheme: PagePermissionScheme, roleIds: unknown): void {
  const selected = new Set(Array.isArray(roleIds) ? roleIds.map((item) => String(item)) : []);
  scheme.roleRefs = props.runtimeRoles
    .filter((role) => selected.has(role.id))
    .map((role) => ({
      roleId: role.id,
      roleCode: role.code,
      roleName: role.name,
    }));
  emit("updateConfig");
}

function selectedRoleIds(scheme: PagePermissionScheme): string[] {
  return scheme.roleRefs.map((role) => role.roleId);
}
</script>

<template>
  <div class="scheme-section">
    <div v-if="(form.runtimePermissionSchemes || []).length === 0" class="scheme-empty">
      暂无权限方案
    </div>

    <div
      v-for="scheme in form.runtimePermissionSchemes || []"
      :key="scheme.id"
      class="scheme-item"
    >
      <div class="scheme-item__head">
        <el-input
          v-model="scheme.name"
          size="small"
          placeholder="方案名称"
          @change="$emit('updateConfig')"
        />
        <el-button size="small" text type="danger" @click="removeScheme(scheme.id)">
          删除
        </el-button>
      </div>
      <el-select
        :model-value="selectedRoleIds(scheme)"
        size="small"
        multiple
        collapse-tags
        collapse-tags-tooltip
        clearable
        placeholder="选择角色"
        @update:model-value="(value: unknown) => updateSchemeRoleRefs(scheme, value)"
      >
        <el-option
          v-for="item in roleOptions"
          :key="item.value"
          :label="item.label"
          :value="item.value"
        />
      </el-select>
    </div>

    <el-button class="scheme-add" size="small" @click="addScheme">
      新增方案
    </el-button>
  </div>
</template>

<style scoped>
.scheme-section {
  display: flex;
  flex-direction: column;
  gap: var(--designer-gap-sm);
}

.scheme-empty {
  padding: 8px 6px;
  color: var(--designer-text-secondary);
  font-size: var(--designer-font-sm);
}

.scheme-item {
  display: flex;
  flex-direction: column;
  gap: var(--designer-gap-xs);
  padding: 6px;
  border: 1px solid var(--designer-border-soft);
  border-radius: var(--designer-radius-sm);
  background: var(--designer-group-surface);
}

.scheme-item__head {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: var(--designer-gap-xs);
}

.scheme-add,
.scheme-item :deep(.el-select) {
  width: 100%;
}
</style>
