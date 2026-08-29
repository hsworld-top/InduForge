<template>
  <DcDialog
    v-model="visible"
    :title="ui('移动分组', 'Move Folder')"
    width="460px"
    body-max-height="260px"
    :dirty="isDirty"
    :close-disabled="loading"
    @close="resetForm"
  >
    <el-form label-position="top" class="compute-folder-move-dialog">
      <el-form-item :label="ui('目标分组', 'Target Folder')">
        <el-select
          v-model="parentId"
          class="compute-folder-move-dialog__select"
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
      <div class="compute-folder-move-dialog__footer">
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
import type { ComputeFolder } from '@/api/schemas/compute.schema'
import DcDialog from '@/components/shared/DcDialog.vue'
import { datacenterLocale } from '@/i18n/runtime'
import type { ComputeFolderTreeNode } from './computeTreeModel'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    folder: ComputeFolderTreeNode | null
    folders: ComputeFolder[]
    loading?: boolean
  }>(),
  {
    loading: false,
  },
)

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'submit', parentId: string | null): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const parentId = ref<string | null>(null)
const currentParentId = computed(() => props.folder?.parentId || null)
const blockedIds = computed(() => {
  const ids = new Set<string>()
  const collect = (folder?: ComputeFolderTreeNode | null) => {
    if (!folder) return
    ids.add(folder.id)
    folder.children.forEach(collect)
  }
  collect(props.folder)
  return ids
})
const canSubmit = computed(
  () => Boolean(props.folder) && parentId.value !== currentParentId.value && !props.loading,
)
const isDirty = computed(
  () => visible.value && Boolean(props.folder) && parentId.value !== currentParentId.value,
)

const flattenFolders = (
  folders: ComputeFolder[],
  depth = 0,
): Array<{ id: string; label: string }> =>
  folders.flatMap((folder) => {
    const id = String(folder.id)
    const children = flattenFolders(folder.children || [], depth + 1)
    if (blockedIds.value.has(id)) return children
    return [{ id, label: `${'　'.repeat(depth)}${folder.name}` }, ...children]
  })

const folderOptions = computed(() => flattenFolders(props.folders))

function resetForm() {
  parentId.value = currentParentId.value
}

function submit() {
  if (!canSubmit.value) return
  emit('submit', parentId.value || null)
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
.compute-folder-move-dialog {
  display: grid;
  gap: 2px;
}

.compute-folder-move-dialog__select {
  width: 100%;
}

.compute-folder-move-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
