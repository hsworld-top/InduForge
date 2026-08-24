<template>
  <DcDialog
    ref="dialogRef"
    :model-value="visible"
    :title="`自定义属性：${datapointName || '-'}`"
    width="min(680px, calc(100vw - 32px))"
    body-max-height="520px"
    destroy-on-close
    :dirty="isDirty"
    :close-disabled="saving"
    @update:model-value="handleVisibleChange"
  >
    <div class="attribute-dialog">
      <div class="attribute-dialog__toolbar">
        <div class="attribute-dialog__label">属性默认值</div>
        <el-button :icon="Plus" @click="appendRow">添加属性</el-button>
      </div>

      <div v-if="rows.length" class="attribute-dialog__table">
        <div class="attribute-dialog__head">
          <span>属性 Key</span>
          <span>默认值</span>
          <span aria-hidden="true"></span>
        </div>
        <div v-for="row in rows" :key="row.id" class="attribute-dialog__row">
          <el-input
            v-model="row.key"
            class="attribute-dialog__key"
            maxlength="64"
            placeholder="例如 asset_code"
            @input="validationVisible = false"
          />
          <el-input v-model="row.value" class="attribute-dialog__value" placeholder="可留空" />
          <el-button
            text
            circle
            :icon="Delete"
            title="删除属性"
            aria-label="删除属性"
            @click="removeRow(row.id)"
          />
        </div>
      </div>

      <el-empty v-else :image-size="56" description="暂无自定义属性" />

      <div v-if="validationVisible && validationError" class="attribute-dialog__error">
        {{ validationError }}
      </div>
    </div>

    <template #footer>
      <el-button :disabled="saving" @click="requestClose">取消</el-button>
      <el-button type="primary" :loading="saving" @click="handleSubmit">保存</el-button>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Delete, Plus } from '@element-plus/icons-vue'
import DcDialog from '@/components/shared/DcDialog.vue'
import {
  buildDatapointAttributeDefaults,
  createDatapointAttributeRows,
  validateDatapointAttributeRows,
  type DatapointAttributeRow,
} from '@/models/datapoint-custom-attributes'

const props = withDefaults(
  defineProps<{
    visible: boolean
    datapointName?: string
    attributes?: Record<string, string>
    saving?: boolean
  }>(),
  { datapointName: '', attributes: () => ({}), saving: false },
)

const emit = defineEmits<{
  submit: [attributes: Record<string, string>]
  cancel: []
}>()

const dialogRef = ref<InstanceType<typeof DcDialog> | null>(null)
const rows = ref<DatapointAttributeRow[]>([])
const nextRowId = ref(1)
const initialSnapshot = ref('{}')
const validationVisible = ref(false)

const payload = computed(() => buildDatapointAttributeDefaults(rows.value))
const currentSnapshot = computed(() => JSON.stringify(payload.value))
const isDirty = computed(() => props.visible && currentSnapshot.value !== initialSnapshot.value)
const validationError = computed(() => validateDatapointAttributeRows(rows.value))

watch(
  () => props.visible,
  (visible) => {
    if (!visible) return
    rows.value = createDatapointAttributeRows(props.attributes)
    nextRowId.value = rows.value.reduce((max, row) => Math.max(max, row.id), 0) + 1
    validationVisible.value = false
    initialSnapshot.value = JSON.stringify(buildDatapointAttributeDefaults(rows.value))
  },
  { immediate: true },
)

function appendRow() {
  rows.value.push({ id: nextRowId.value++, key: '', value: '' })
  validationVisible.value = false
}

function removeRow(id: number) {
  rows.value = rows.value.filter((row) => row.id !== id)
  validationVisible.value = false
}

function handleSubmit() {
  validationVisible.value = true
  if (validationError.value) return
  emit('submit', payload.value)
}

function handleVisibleChange(value: boolean) {
  if (!value) emit('cancel')
}

function requestClose() {
  void dialogRef.value?.requestClose()
}

/** 仅在接口保存成功后调用：清除脏状态并跳过未保存确认。 */
function closeAfterSave() {
  initialSnapshot.value = currentSnapshot.value
  dialogRef.value?.closeSilently()
}

defineExpose({ closeAfterSave })
</script>

<style scoped>
.attribute-dialog {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.attribute-dialog__toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.attribute-dialog__label {
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 600;
}

.attribute-dialog__table {
  overflow: hidden;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
}

.attribute-dialog__head,
.attribute-dialog__row {
  display: grid;
  grid-template-columns: minmax(180px, 0.8fr) minmax(220px, 1.2fr) 36px;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
}

.attribute-dialog__head {
  background: var(--dc-surface-muted, #f8fafc);
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 600;
}

.attribute-dialog__row + .attribute-dialog__row {
  border-top: 1px solid var(--dc-border);
}

.attribute-dialog__error {
  color: var(--dc-danger);
  font-size: 12px;
  line-height: 1.5;
}

:deep(.el-empty) {
  padding: 24px 0;
}

@media (max-width: 640px) {
  .attribute-dialog__head {
    display: none;
  }

  .attribute-dialog__row {
    grid-template-columns: minmax(0, 1fr) 36px;
  }

  .attribute-dialog__value {
    grid-column: 1;
  }
}
</style>
