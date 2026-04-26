<script setup lang="ts">
import type { PageInspectorFormState } from "./page-inspector-types";
import type { RuntimeRoleRecord } from "@/stores/editor/project-runtime-role-actions";
import { computed, ref } from "vue";
import PagePermissionSchemesSection from "./PagePermissionSchemesSection.vue";

const props = defineProps<{
  form: PageInspectorFormState;
  runtimeRoles: RuntimeRoleRecord[];
}>();

const emit = defineEmits<{
  updateConfig: [];
}>();

const schemeDialogVisible = ref(false);

const roleOptions = computed(() =>
  props.runtimeRoles
    .filter((role) => role.status !== "disabled")
    .map((role) => ({
      label: role.name || role.code || role.id,
      value: role.id,
    })),
);

const schemeCount = computed(() => (props.form.runtimePermissionSchemes || []).length);

function openSchemeDialog(): void {
  if (!props.form.runtimeAccessEnabled) return;
  schemeDialogVisible.value = true;
}

const selectedRoleIds = computed<string[]>({
  get() {
    return (props.form.runtimeAccessAllowedRoles || []).map((role) => role.roleId);
  },
  set(roleIds) {
    const selected = new Set(roleIds);
    props.form.runtimeAccessAllowedRoles = props.runtimeRoles
      .filter((role) => selected.has(role.id))
      .map((role) => ({
        roleId: role.id,
        roleCode: role.code,
        roleName: role.name,
      }));
    emit("updateConfig");
  },
});
</script>

<template>
  <div class="page-section-fields">
    <div class="page-prop-item page-prop-item--switch">
      <div class="page-prop-label">启用控制</div>
      <div class="page-prop-editor page-prop-editor-switch">
        <el-switch v-model="form.runtimeAccessEnabled" @change="$emit('updateConfig')" />
      </div>
    </div>

    <div class="page-prop-item">
      <div class="page-prop-label">访问角色</div>
      <div class="page-prop-editor">
        <el-select
          v-model="selectedRoleIds"
          size="small"
          multiple
          collapse-tags
          collapse-tags-tooltip
          clearable
          :disabled="!form.runtimeAccessEnabled"
          placeholder="不选则不限制"
        >
          <el-option
            v-for="item in roleOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </el-select>
      </div>
    </div>

    <div class="page-prop-item">
      <div class="page-prop-label">权限方案</div>
      <div class="page-prop-editor permission-scheme-entry">
        <span class="permission-scheme-entry__count">{{ schemeCount }} 个</span>
        <el-button
          size="small"
          :disabled="!form.runtimeAccessEnabled"
          @click="openSchemeDialog"
        >
          配置方案
        </el-button>
      </div>
    </div>

    <el-dialog
      v-model="schemeDialogVisible"
      title="权限方案"
      width="560px"
      append-to-body
      class="runtime-permission-scheme-dialog"
    >
      <PagePermissionSchemesSection
        :form="form"
        :runtime-roles="runtimeRoles"
        @update-config="$emit('updateConfig')"
      />
    </el-dialog>
  </div>
</template>

<style scoped>
.page-section-fields {
  display: flex;
  flex-direction: column;
  gap: var(--designer-gap-xs);
}

.page-prop-item {
  display: flex;
  align-items: center;
  gap: var(--designer-gap-sm);
  min-height: 32px;
  padding: 4px 6px;
  border-radius: var(--designer-radius-sm);
}

.page-prop-label {
  width: 92px;
  min-width: 92px;
  font-size: var(--designer-font-label);
  color: var(--designer-text-regular);
}

.page-prop-editor {
  flex: 1;
  min-width: 0;
}

.page-prop-editor-switch {
  display: flex;
  justify-content: flex-end;
}

.page-prop-item--switch {
  justify-content: space-between;
}

.page-prop-editor :deep(.el-select) {
  width: 100%;
}

.permission-scheme-entry {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: var(--designer-gap-sm);
}

.permission-scheme-entry__count {
  color: var(--designer-text-secondary);
  font-size: var(--designer-font-label);
}
</style>
