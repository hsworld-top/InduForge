<template>
  <DcDialog
    ref="dialogRef"
    :model-value="visible"
    :title="ui(`写权限：${datapoint?.name || '-'}`, `Write Permission: ${datapoint?.name || '-'}`)"
    width="520px"
    destroy-on-close
    :dirty="isDirty"
    @update:model-value="handleVisibleChange"
  >
    <div class="rp-dialog">
      <div class="rp-dialog__hint">
        {{ ui('运行态写权限只影响节点运行时是否允许写入该数据点，不改变开发态管理权限。', 'Runtime write permission controls whether nodes may write this data point; it does not change development management permissions.') }}
      </div>

      <el-form label-position="top">
        <el-form-item :label="ui('当前摘要', 'Current Summary')">
          <el-tag type="info">{{ grantSummary }}</el-tag>
        </el-form-item>

        <el-form-item :label="ui('继承工程默认规则', 'Inherit Project Defaults')">
          <el-switch v-model="formInherit" />
        </el-form-item>

        <el-form-item :label="ui('允许写入的角色', 'Roles Allowed to Write')">
          <el-input
            v-model="allowRolesInput"
            type="textarea"
            :rows="4"
            :placeholder="ui('每行一个角色，也可用逗号分隔', 'One role per line or comma-separated')"
          />
        </el-form-item>

        <el-form-item :label="ui('禁止写入的角色', 'Roles Denied Write Access')">
          <el-input
            v-model="denyRolesInput"
            type="textarea"
            :rows="4"
            :placeholder="ui('每行一个角色，也可用逗号分隔', 'One role per line or comma-separated')"
          />
        </el-form-item>
      </el-form>
    </div>

    <template #footer>
      <el-button @click="requestClose">{{ ui('取消', 'Cancel') }}</el-button>
      <el-button type="primary" :loading="saving" @click="handleSubmit">{{ ui('保存', 'Save') }}</el-button>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import DcDialog from '@/components/shared/DcDialog.vue'
import { datacenterLocale } from '@/i18n/runtime'
import {
  normalizeRuntimeGrantPayload,
  summarizeRuntimeGrant,
} from '@/utils/runtime-permission-grants'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)

interface RuntimeGrantPayload {
  allowRoles?: string[]
  denyRoles?: string[]
  inherit?: boolean
}

interface DataPointRow {
  id: string
  name?: string
  runtimePermissions?: Record<string, unknown>
  runtimePermissionGrants?: Record<string, unknown>
  writePermission?: Record<string, unknown>
  runtimeGrant?: RuntimeGrantPayload
}

const props = defineProps<{
  visible: boolean
  datapoint: DataPointRow | null
  projectId: string
  saving?: boolean
}>()

const emit = defineEmits<{
  /** 提交权限 payload */
  submit: [grant: RuntimeGrantPayload]
  cancel: []
}>()

const allowRolesInput = ref('')
const denyRolesInput = ref('')
const formInherit = ref(true)
const initialSnapshot = ref('')
const dialogRef = ref<InstanceType<typeof DcDialog> | null>(null)

function parseRoles(value: string): string[] {
  return String(value || '')
    .split(/[\n,，]/)
    .map((s) => s.trim())
    .filter(Boolean)
}

const draftGrant = computed(() =>
  normalizeRuntimeGrantPayload({
    allowRoles: parseRoles(allowRolesInput.value),
    denyRoles: parseRoles(denyRolesInput.value),
    inherit: formInherit.value,
  }),
)
const draftSnapshot = computed(() => JSON.stringify(draftGrant.value))
const isDirty = computed(() => props.visible && draftSnapshot.value !== initialSnapshot.value)

// 从数据点解析当前写权限 grant
function extractWriteGrant(row: DataPointRow | null): RuntimeGrantPayload {
  if (!row) return normalizeRuntimeGrantPayload()
  const rp = row.runtimePermissions as { write?: unknown } | undefined
  const rpg = row.runtimePermissionGrants as { write?: unknown } | undefined
  return normalizeRuntimeGrantPayload(
    rp?.write || rpg?.write || row.writePermission || row.runtimeGrant || {},
  )
}

// 初始化表单
watch(
  () => [props.visible, props.datapoint],
  () => {
    if (!props.visible) return
    const grant = extractWriteGrant(props.datapoint)
    formInherit.value = grant.inherit
    allowRolesInput.value = grant.allowRoles.join('\n')
    denyRolesInput.value = grant.denyRoles.join('\n')
    initialSnapshot.value = draftSnapshot.value
  },
  { immediate: true },
)

const grantSummary = computed(() => summarizeRuntimeGrant(draftGrant.value, datacenterLocale.value))

function handleSubmit() {
  initialSnapshot.value = draftSnapshot.value
  emit('submit', draftGrant.value)
}

function handleVisibleChange(val: boolean) {
  if (!val) emit('cancel')
}

function requestClose() {
  void dialogRef.value?.requestClose()
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
