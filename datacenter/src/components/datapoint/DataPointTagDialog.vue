<template>
  <DcDialog
    ref="dialogRef"
    :model-value="visible"
    :title="dialogTitle"
    width="520px"
    class="dp-tag-dialog"
    destroy-on-close
    :show-close="false"
    :dirty="isDirty"
    @update:model-value="handleVisibleChange"
  >
    <template #header>
      <div class="dp-tag-dialog__head">
        <h3>{{ dialogTitle }}</h3>
        <button
          type="button"
          class="dp-tag-dialog__close"
          :aria-label="ui('关闭标签管理', 'Close tag management')"
          @click="requestClose"
        >
          <Close />
        </button>
      </div>
    </template>

    <div class="dp-tag-dialog__body">
      <!-- 批量模式说明 -->
      <p v-if="isBatch" class="dp-tag-dialog__batch-tip">
        {{ ui(`将为 ${batchDisplayCount} 个数据点合并以下标签（已有标签保留，不覆盖）。`, `The selected tags will be merged into ${batchDisplayCount} data points; existing tags are preserved.`) }}
      </p>

      <!-- 新建标签 -->
      <div class="dp-tag-dialog__row">
        <label>{{ ui('新建标签', 'New Tag') }}</label>
        <div class="dp-tag-dialog__create">
          <el-input
            v-model="tagCreateInput"
            :placeholder="ui('请输入标签名称', 'Enter a tag name')"
            @keyup.enter="appendTag"
          />
          <button type="button" class="dp-tag-dialog__create-btn" @click="appendTag">
            {{ ui('新建标签', 'Create Tag') }}
          </button>
        </div>
      </div>

      <!-- 标签选择 -->
      <div class="dp-tag-dialog__row">
        <label>{{ ui('标签选择', 'Select Tags') }}</label>
        <el-select
          v-model="tagDraft"
          multiple
          filterable
          allow-create
          default-first-option
          collapse-tags
          collapse-tags-tooltip
          :placeholder="ui('请选择或输入标签', 'Select or enter tags')"
        >
          <el-option
            v-for="item in tagOptions"
            :key="item.value"
            :label="item.name"
            :value="item.value"
          />
        </el-select>
      </div>
    </div>

    <template #footer>
      <div class="dp-tag-dialog__footer">
        <el-button @click="requestClose">{{ ui('取消', 'Cancel') }}</el-button>
        <el-button type="primary" :loading="saving" @click="handleSubmit">{{ ui('保存', 'Save') }}</el-button>
      </div>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Close } from '@element-plus/icons-vue'
import DcDialog from '@/components/shared/DcDialog.vue'
import { datacenterLocale } from '@/i18n/runtime'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)

interface DataPointRow {
  id: string
  name?: string
  tags?: unknown[]
}

const props = defineProps<{
  /** 是否显示 */
  visible: boolean
  /** 当前操作的数据点（单条），批量时为 null */
  datapoint: DataPointRow | null
  /** 项目 ID */
  projectId: string
  /** 已有标签选项（供选择器复用） */
  tagOptions?: Array<{ value: string; name: string }>
  /** 批量操作时选中的数据点列表 */
  batchRows?: DataPointRow[]
  /** 跨页批量选择时的真实数量 */
  batchCount?: number
  /** 外部传入的 saving 状态 */
  saving?: boolean
}>()

const emit = defineEmits<{
  /** 提交标签列表 */
  submit: [tags: string[], mode: 'single' | 'batch']
  cancel: []
}>()

const tagDraft = ref<string[]>([])
const tagCreateInput = ref('')
const initialSnapshot = ref('')
const dialogRef = ref<InstanceType<typeof DcDialog> | null>(null)

const isBatch = computed(() => !props.datapoint && (props.batchRows?.length ?? 0) > 0)
const batchDisplayCount = computed(() => props.batchCount ?? props.batchRows?.length ?? 0)
const draftSnapshot = computed(() =>
  JSON.stringify({
    tags: [...tagDraft.value].sort(),
    createInput: tagCreateInput.value,
  }),
)
const isDirty = computed(() => props.visible && draftSnapshot.value !== initialSnapshot.value)

const dialogTitle = computed(() =>
  isBatch.value
    ? ui(`标签管理（${batchDisplayCount.value} 项）`, `Manage Tags (${batchDisplayCount.value})`)
    : ui(`标签管理：${props.datapoint?.name || '-'}`, `Manage Tags: ${props.datapoint?.name || '-'}`),
)

// 初始化 draft
watch(
  () => [props.visible, props.datapoint],
  () => {
    if (!props.visible) return
    if (props.datapoint) {
      tagDraft.value = normalizeTags(props.datapoint.tags)
    } else {
      tagDraft.value = []
    }
    tagCreateInput.value = ''
    initialSnapshot.value = draftSnapshot.value
  },
  { immediate: true },
)

function normalizeTags(value: unknown): string[] {
  if (!Array.isArray(value)) return []
  const result: string[] = []
  for (const item of value) {
    let label = ''
    if (typeof item === 'string') {
      label = item
    } else if (item && typeof item === 'object') {
      const r = item as Record<string, unknown>
      label = String(r.label || r.name || r.value || '')
    }
    const n = label.trim()
    if (n && !result.includes(n)) result.push(n)
  }
  return result
}

function appendTag() {
  const tag = tagCreateInput.value.trim()
  if (!tag) return
  if (!tagDraft.value.includes(tag)) {
    tagDraft.value = [...tagDraft.value, tag]
  }
  tagCreateInput.value = ''
}

function handleSubmit() {
  initialSnapshot.value = draftSnapshot.value
  emit('submit', [...tagDraft.value], isBatch.value ? 'batch' : 'single')
}

function handleVisibleChange(val: boolean) {
  if (!val) emit('cancel')
}

function requestClose() {
  void dialogRef.value?.requestClose()
}
</script>

<style scoped>
:deep(.dp-tag-dialog) {
  border-radius: 14px;
}

:deep(.dp-tag-dialog .el-dialog__header) {
  margin: 0;
  padding: 0;
}

:deep(.dp-tag-dialog .el-dialog__body) {
  padding: 32px 36px 30px;
}

:deep(.dp-tag-dialog .el-dialog__footer) {
  padding: 0 36px 34px;
}

.dp-tag-dialog__head {
  min-height: 80px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 0 36px;
  border-radius: 14px 14px 0 0;
  background: rgba(248, 250, 252, 0.86);
}

.dp-tag-dialog__head h3 {
  margin: 0;
  color: var(--dc-text);
  font-size: 18px;
  font-weight: 500;
}

.dp-tag-dialog__close {
  width: 30px;
  height: 30px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--dc-text-muted);
  cursor: pointer;
}

.dp-tag-dialog__close:hover {
  background: rgba(0, 0, 0, 0.04);
  color: var(--dc-text);
}

.dp-tag-dialog__close svg {
  width: 18px;
  height: 18px;
}

.dp-tag-dialog__body {
  display: flex;
  flex-direction: column;
  gap: 22px;
  padding-bottom: 34px;
  border-bottom: 1px solid var(--dc-border);
}

.dp-tag-dialog__row {
  display: grid;
  grid-template-columns: 92px minmax(0, 1fr);
  align-items: center;
  gap: 14px;
}

.dp-tag-dialog__row > label {
  color: var(--dc-text-secondary);
  font-size: 14px;
  font-weight: 600;
  text-align: right;
  white-space: nowrap;
}

.dp-tag-dialog__create {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 112px;
  gap: 10px;
}

.dp-tag-dialog__create-btn {
  height: 32px;
  border: 1px solid rgba(64, 158, 255, 0.42);
  border-radius: var(--dc-radius-md);
  background: rgba(64, 158, 255, 0.08);
  color: #409eff;
  cursor: pointer;
  font-family: inherit;
  font-size: 14px;
  font-weight: 600;
}

.dp-tag-dialog__create-btn:hover {
  background: rgba(64, 158, 255, 0.14);
}

.dp-tag-dialog__row :deep(.el-select) {
  width: 100%;
}

.dp-tag-dialog__batch-tip {
  margin: 0 0 4px;
  padding: 8px 10px;
  border-radius: var(--dc-radius-md);
  background: rgba(29, 78, 216, 0.06);
  color: var(--dc-text-secondary);
  font-size: 13px;
  line-height: 1.6;
}

.dp-tag-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.dp-tag-dialog__footer :deep(.el-button) {
  min-width: 76px;
  height: 32px;
  border-radius: var(--dc-radius-md);
  font-size: 14px;
  font-weight: 600;
}
</style>
