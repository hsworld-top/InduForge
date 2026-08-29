<template>
  <DcDialog
    v-model="visible"
    :title="ui('重命名分组', 'Rename Folder')"
    width="420px"
    body-max-height="220px"
    :dirty="isDirty"
    :close-disabled="loading"
    @close="resetForm"
  >
    <el-form label-position="top" class="compute-folder-rename-dialog">
      <el-form-item :label="ui('名称', 'Name')" required>
        <el-input
          v-model="name"
          maxlength="60"
          show-word-limit
          :placeholder="ui('输入分组名称', 'Enter a folder name')"
          @keyup.enter="submit"
        />
      </el-form-item>
    </el-form>

    <template #footer>
      <div class="compute-folder-rename-dialog__footer">
        <el-button @click="visible = false">{{ ui('取消', 'Cancel') }}</el-button>
        <el-button type="primary" :loading="loading" :disabled="!canSubmit" @click="submit">
          {{ ui('保存', 'Save') }}
        </el-button>
      </div>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import DcDialog from '@/components/shared/DcDialog.vue'
import { datacenterLocale } from '@/i18n/runtime'
import type { ComputeFolderTreeNode } from './computeTreeModel'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    folder: ComputeFolderTreeNode | null
    loading?: boolean
  }>(),
  {
    loading: false,
  },
)

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'submit', name: string): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const name = ref('')
const canSubmit = computed(
  () =>
    Boolean(props.folder) &&
    name.value.trim().length > 0 &&
    name.value.trim() !== props.folder?.name &&
    !props.loading,
)
const isDirty = computed(
  () => visible.value && Boolean(props.folder) && name.value.trim() !== (props.folder?.name || ''),
)

function resetForm() {
  name.value = props.folder?.name || ''
}

function submit() {
  if (!canSubmit.value) return
  emit('submit', name.value.trim())
}

watch(
  () => props.modelValue,
  (open) => {
    if (open) resetForm()
  },
)

watch(
  () => props.folder?.id,
  () => {
    if (props.modelValue) resetForm()
  },
)
</script>

<style scoped>
.compute-folder-rename-dialog {
  display: grid;
  gap: 2px;
}

.compute-folder-rename-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
