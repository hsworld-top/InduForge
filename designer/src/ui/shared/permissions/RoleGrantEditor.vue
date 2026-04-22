<script setup lang="ts">
import { computed } from "vue";
import type { RolePermission } from "@/editor-core/document/types";
import { normalizeRoleList, sanitizeRoleGrant, summarizeRoleGrant } from "./role-grant-summary";

const props = withDefaults(
  defineProps<{
    modelValue: RolePermission | undefined;
    roleOptions?: string[];
  }>(),
  {
    roleOptions: () => [],
  },
);

const emit = defineEmits<{
  (event: "update:modelValue", value: RolePermission | undefined): void;
}>();

/**
 * 解析角色输入框内容，允许逗号、顿号、分号和换行混输。
 * @param {string} value - 原始输入
 * @returns {string[]} 清洗后的角色列表
 */
function parseRoleInput(value: string): string[] {
  return normalizeRoleList(
    String(value || "")
      .split(/[\n,，;；、]+/g)
      .map((role) => role.trim()),
  );
}

/**
 * 以补丁方式更新授权配置，统一走清洗逻辑，避免泄漏空字段。
 * @param {Partial<RolePermission>} patch - 局部补丁
 */
function updateGrant(patch: Partial<RolePermission>): void {
  emit("update:modelValue", sanitizeRoleGrant({ ...(props.modelValue || {}), ...patch }));
}

const allowRolesText = computed<string>({
  get: () => normalizeRoleList(props.modelValue?.allowRoles).join(", "),
  set: (value) => updateGrant({ allowRoles: parseRoleInput(value) }),
});

const denyRolesText = computed<string>({
  get: () => normalizeRoleList(props.modelValue?.denyRoles).join(", "),
  set: (value) => updateGrant({ denyRoles: parseRoleInput(value) }),
});

const inheritEnabled = computed<boolean>({
  get: () => props.modelValue?.inherit !== false,
  set: (value) => updateGrant({ inherit: value ? true : false }),
});

const summaryText = computed(() => summarizeRoleGrant(props.modelValue));
const normalizedRoleOptions = computed(() => normalizeRoleList(props.roleOptions));
</script>

<template>
  <div class="role-grant-editor">
    <div class="role-grant-editor__summary">{{ summaryText }}</div>
    <div class="role-grant-editor__row">
      <span class="role-grant-editor__label">继承上级</span>
      <el-switch v-model="inheritEnabled" />
    </div>
    <div class="role-grant-editor__row role-grant-editor__row--stacked">
      <span class="role-grant-editor__label">允许角色</span>
      <el-input
        v-model="allowRolesText"
        size="small"
        placeholder="多个角色用逗号分隔"
      />
    </div>
    <div class="role-grant-editor__row role-grant-editor__row--stacked">
      <span class="role-grant-editor__label">拒绝角色</span>
      <el-input
        v-model="denyRolesText"
        size="small"
        placeholder="多个角色用逗号分隔"
      />
    </div>
    <div v-if="normalizedRoleOptions.length > 0" class="role-grant-editor__options">
      <span class="role-grant-editor__label">可选角色</span>
      <div class="role-grant-editor__tags">
        <el-tag
          v-for="role in normalizedRoleOptions"
          :key="role"
          size="small"
          effect="plain"
        >
          {{ role }}
        </el-tag>
      </div>
    </div>
  </div>
</template>

<style scoped>
.role-grant-editor {
  display: flex;
  flex-direction: column;
  gap: var(--designer-gap-xs);
  width: 100%;
}

.role-grant-editor__summary {
  font-size: 12px;
  color: var(--designer-text-secondary);
}

.role-grant-editor__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--designer-gap-sm);
}

.role-grant-editor__row--stacked,
.role-grant-editor__options {
  flex-direction: column;
  align-items: stretch;
}

.role-grant-editor__label {
  font-size: 12px;
  color: var(--designer-text-regular);
}

.role-grant-editor__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
</style>
