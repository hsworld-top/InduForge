<template>
  <DcDialog
    ref="dialogRef"
    v-model="visible"
    :title="ui('新建文件夹', 'New Folder')"
    width="460px"
    body-max-height="320px"
    :dirty="isDirty"
    :close-disabled="loading"
    @close="resetForm"
  >
    <el-form label-position="top" class="compute-folder-dialog">
      <el-form-item :label="ui('名称', 'Name')" required>
        <el-input v-model="form.name" maxlength="40" show-word-limit :placeholder="ui('输入文件夹名称', 'Enter a folder name')" />
      </el-form-item>

      <el-form-item :label="ui('上级文件夹', 'Parent Folder')">
        <el-select
          v-model="form.parentId"
          class="compute-folder-dialog__select"
          clearable
          :placeholder="ui('根目录', 'Root')"
        >
          <el-option :label="ui('根目录', 'Root')" :value="null" />
          <el-option
            v-for="folder in folderOptions"
            :key="folder.id"
            :label="folder.label"
            :value="folder.id"
          />
        </el-select>
      </el-form-item>

      <div v-if="error" class="compute-folder-dialog__error">{{ error }}</div>
    </el-form>

    <template #footer>
      <div class="compute-folder-dialog__footer">
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
import type { ComputeFolder, ComputeFolderSave } from '@/api/schemas/compute.schema'
import DcDialog from '@/components/shared/DcDialog.vue'
import { datacenterLocale } from '@/i18n/runtime'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    folders: ComputeFolder[]
    initialParentId?: string | null
    loading?: boolean
    error?: string
  }>(),
  {
    loading: false,
    error: '',
    initialParentId: null,
  },
)

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'submit', data: ComputeFolderSave): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})
const dialogRef = ref<InstanceType<typeof DcDialog> | null>(null)

const form = reactive<ComputeFolderSave>({
  name: '',
  parentId: null,
})

const canSubmit = computed(() => form.name.trim().length > 0 && !props.loading)
const isDirty = computed(
  () =>
    visible.value &&
    (form.name.trim().length > 0 || (form.parentId || null) !== props.initialParentId),
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
  form.parentId = props.initialParentId
}

function submit() {
  if (!canSubmit.value) return
  emit('submit', {
    name: form.name.trim(),
    parentId: form.parentId || null,
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
.compute-folder-dialog {
  display: grid;
  gap: 2px;
}

.compute-folder-dialog__select {
  width: 100%;
}

.compute-folder-dialog__error {
  padding: 9px 10px;
  border: 1px solid rgba(220, 38, 38, 0.2);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
  font-size: 13px;
  line-height: 1.5;
}

.compute-folder-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
