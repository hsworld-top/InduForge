<template>
  <DcDialog
    ref="dialogRef"
    v-model="visible"
    :title="ui('新建计算单元', 'New Compute Unit')"
    width="520px"
    body-max-height="420px"
    :dirty="isDirty"
    :close-disabled="loading"
    @close="resetForm"
  >
    <el-form label-position="top" class="compute-create-dialog">
      <el-form-item :label="ui('名称', 'Name')" required>
        <el-input
          v-model="form.name"
          maxlength="60"
          show-word-limit
          :placeholder="ui('输入计算单元名称', 'Enter a compute unit name')"
        />
      </el-form-item>

      <el-form-item :label="ui('语言', 'Language')">
        <el-select v-model="form.lang" class="compute-create-dialog__select">
          <el-option label="JavaScript" value="javascript" />
          <el-option label="Python" value="python" />
        </el-select>
      </el-form-item>

      <el-form-item :label="ui('文件夹', 'Folder')">
        <el-select
          v-model="form.folderId"
          class="compute-create-dialog__select"
          clearable
          :placeholder="ui('根目录', 'Root')"
        >
          <el-option :label="ui('根目录', 'Root')" value="" />
          <el-option
            v-for="folder in folderOptions"
            :key="folder.id"
            :label="folder.label"
            :value="folder.id"
          />
        </el-select>
      </el-form-item>

      <el-form-item :label="ui('描述', 'Description')">
        <el-input
          v-model="form.description"
          type="textarea"
          :rows="3"
          maxlength="200"
          show-word-limit
          :placeholder="ui('可选', 'Optional')"
        />
      </el-form-item>

      <div v-if="error" class="compute-create-dialog__error">{{ error }}</div>
    </el-form>

    <template #footer>
      <div class="compute-create-dialog__footer">
        <el-button @click="visible = false">{{ ui('取消', 'Cancel') }}</el-button>
        <el-button type="primary" :loading="loading" :disabled="!canSubmit" @click="submit">
          {{ ui('创建', 'Create') }}
        </el-button>
      </div>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import type { ComputeFolder, ComputeUnitSave } from '@/api/schemas/compute.schema'
import DcDialog from '@/components/shared/DcDialog.vue'
import { datacenterLocale } from '@/i18n/runtime'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    folders: ComputeFolder[]
    initialFolderId?: string | null
    loading?: boolean
    error?: string
  }>(),
  {
    loading: false,
    error: '',
    initialFolderId: null,
  },
)

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'submit', data: ComputeUnitSave): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})
const dialogRef = ref<InstanceType<typeof DcDialog> | null>(null)

const form = reactive<ComputeUnitSave>({
  name: '',
  lang: 'javascript',
  folderId: null,
  description: '',
  code: '',
  outputs: [],
})

const canSubmit = computed(() => form.name.trim().length > 0 && !props.loading)
const isDirty = computed(
  () =>
    visible.value &&
    (form.name.trim().length > 0 ||
      form.description.trim().length > 0 ||
      form.lang !== 'javascript' ||
      (form.folderId || null) !== props.initialFolderId ||
      Boolean(form.code)),
)

const flattenFolders = (
  folders: ComputeFolder[],
  depth = 0,
): Array<{ id: string; label: string }> =>
  folders.flatMap((folder) => [
    { id: String(folder.id), label: `${'　'.repeat(depth)}${folder.name}` },
    ...flattenFolders(folder.children || [], depth + 1),
  ])

const folderOptions = computed(() => flattenFolders(props.folders))

function resetForm() {
  form.name = ''
  form.lang = 'javascript'
  form.folderId = props.initialFolderId
  form.description = ''
  form.code = ''
}

function submit() {
  if (!canSubmit.value) return
  emit('submit', {
    name: form.name.trim(),
    lang: form.lang,
    folderId: form.folderId || null,
    description: form.description || '',
    code: form.code || '',
    outputs: form.outputs,
  })
}

function closeSilently() {
  dialogRef.value?.closeSilently()
}

watch(
  () => props.modelValue,
  (open) => {
    if (open) resetForm()
  },
)

defineExpose({ closeSilently })
</script>

<style scoped>
.compute-create-dialog {
  display: grid;
  gap: 2px;
}

.compute-create-dialog__select {
  width: 100%;
}

.compute-create-dialog__error {
  padding: 9px 10px;
  border: 1px solid rgba(220, 38, 38, 0.2);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
  font-size: 13px;
  line-height: 1.5;
}

.compute-create-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
