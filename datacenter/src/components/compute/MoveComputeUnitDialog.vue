<template>
  <DcDialog
    v-model="visible"
    :title="ui('移动到分组', 'Move to Folder')"
    width="460px"
    body-max-height="260px"
    :dirty="isDirty"
    :close-disabled="loading"
    @close="resetForm"
  >
    <el-form label-position="top" class="compute-move-dialog">
      <el-form-item :label="ui('目标分组', 'Target Folder')">
        <el-select
          v-model="folderId"
          class="compute-move-dialog__select"
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
    </el-form>

    <template #footer>
      <div class="compute-move-dialog__footer">
        <el-button @click="visible = false">{{ ui('取消', 'Cancel') }}</el-button>
        <el-button type="primary" :loading="loading" :disabled="!canSubmit" @click="submit">
          {{ ui('移动', 'Move') }}
        </el-button>
      </div>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { ComputeFolder, ComputeUnit } from '@/api/schemas/compute.schema'
import DcDialog from '@/components/shared/DcDialog.vue'
import { datacenterLocale } from '@/i18n/runtime'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    unit: ComputeUnit | null
    folders: ComputeFolder[]
    loading?: boolean
  }>(),
  {
    loading: false,
  },
)

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'submit', folderId: string | null): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const folderId = ref<string | null>(null)
const currentFolderId = computed(() => (props.unit?.folderId ? String(props.unit.folderId) : null))
const canSubmit = computed(
  () => Boolean(props.unit) && folderId.value !== currentFolderId.value && !props.loading,
)
const isDirty = computed(
  () => visible.value && Boolean(props.unit) && folderId.value !== currentFolderId.value,
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
  folderId.value = currentFolderId.value
}

function submit() {
  if (!canSubmit.value) return
  emit('submit', folderId.value || null)
}

watch(
  () => props.modelValue,
  (open) => {
    if (open) resetForm()
  },
)

watch(
  () => props.unit?.id,
  () => {
    if (props.modelValue) resetForm()
  },
)
</script>

<style scoped>
.compute-move-dialog {
  display: grid;
  gap: 2px;
}

.compute-move-dialog__select {
  width: 100%;
}

.compute-move-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
