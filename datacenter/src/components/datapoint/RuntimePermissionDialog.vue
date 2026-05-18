<template>
  <DcDialog
    :model-value="visible"
    :title="`写权限：${datapoint?.name || '-'}`"
    width="520px"
    destroy-on-close
    @update:model-value="handleVisibleChange"
  >
    <div class="rp-dialog">
      <div class="rp-dialog__hint">
        运行态写权限只影响节点运行时是否允许写入该数据点，不改变开发态管理权限。
      </div>

      <el-form label-position="top">
        <el-form-item label="当前摘要">
          <el-tag type="info">{{ grantSummary }}</el-tag>
        </el-form-item>

        <el-form-item label="继承工程默认规则">
          <el-switch v-model="formInherit" />
        </el-form-item>

        <el-form-item label="允许写入的角色">
          <el-input
            v-model="allowRolesInput"
            type="textarea"
            :rows="4"
            placeholder="每行一个角色，也可用逗号分隔"
          />
        </el-form-item>

        <el-form-item label="禁止写入的角色">
          <el-input
            v-model="denyRolesInput"
            type="textarea"
            :rows="4"
            placeholder="每行一个角色，也可用逗号分隔"
          />
        </el-form-item>
      </el-form>
    </div>

    <template #footer>
      <el-button @click="$emit('cancel')">取消</el-button>
      <el-button type="primary" :loading="saving" @click="handleSubmit">
        保存
      </el-button>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from "vue";
import DcDialog from "@/components/shared/DcDialog.vue";
import {
  normalizeRuntimeGrantPayload,
  summarizeRuntimeGrant,
} from "@/utils/runtime-permission-grants";

interface RuntimeGrantPayload {
  allowRoles?: string[];
  denyRoles?: string[];
  inherit?: boolean;
}

interface DataPointRow {
  id: string;
  name?: string;
  runtimePermissions?: Record<string, unknown>;
  runtimePermissionGrants?: Record<string, unknown>;
  writePermission?: Record<string, unknown>;
  runtimeGrant?: RuntimeGrantPayload;
}

const props = defineProps<{
  visible: boolean;
  datapoint: DataPointRow | null;
  projectId: string;
  saving?: boolean;
}>();

const emit = defineEmits<{
  /** 提交权限 payload */
  submit: [grant: RuntimeGrantPayload];
  cancel: [];
}>();

const allowRolesInput = ref("");
const denyRolesInput = ref("");
const formInherit = ref(true);

// 从数据点解析当前写权限 grant
function extractWriteGrant(row: DataPointRow | null): RuntimeGrantPayload {
  if (!row) return normalizeRuntimeGrantPayload();
  const rp = row.runtimePermissions as { write?: unknown } | undefined;
  const rpg = row.runtimePermissionGrants as { write?: unknown } | undefined;
  return normalizeRuntimeGrantPayload(
    rp?.write || rpg?.write || row.writePermission || row.runtimeGrant || {},
  );
}

// 初始化表单
watch(
  () => [props.visible, props.datapoint],
  () => {
    if (!props.visible) return;
    const grant = extractWriteGrant(props.datapoint);
    formInherit.value = grant.inherit;
    allowRolesInput.value = grant.allowRoles.join("\n");
    denyRolesInput.value = grant.denyRoles.join("\n");
  },
  { immediate: true },
);

function parseRoles(value: string): string[] {
  return String(value || "")
    .split(/[\n,，]/)
    .map((s) => s.trim())
    .filter(Boolean);
}

const draftGrant = computed(() =>
  normalizeRuntimeGrantPayload({
    allowRoles: parseRoles(allowRolesInput.value),
    denyRoles: parseRoles(denyRolesInput.value),
    inherit: formInherit.value,
  }),
);

const grantSummary = computed(() => summarizeRuntimeGrant(draftGrant.value));

function handleSubmit() {
  emit("submit", draftGrant.value);
}

function handleVisibleChange(val: boolean) {
  if (!val) emit("cancel");
}
</script>

<style scoped>
.rp-dialog {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.rp-dialog__hint {
  color: var(--dc-text-secondary);
  font-size: 12px;
  line-height: 1.6;
}
</style>
