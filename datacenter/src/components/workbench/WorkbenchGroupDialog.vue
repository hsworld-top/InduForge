<template>
  <DcDialog
    ref="dialogRef"
    v-model="visible"
    :title="title"
    width="460px"
    body-max-height="320px"
    :dirty="isDirty"
    :close-disabled="loading"
  >
    <el-form label-position="top" class="workbench-group-dialog" @submit.prevent>
      <el-form-item :label="ui('名称', 'Name')" required>
        <el-input
          v-model="form.name"
          :maxlength="maxNameLength"
          show-word-limit
          :placeholder="ui('输入分组名称', 'Enter a group name')"
          @keyup.enter="submit"
        />
      </el-form-item>

      <el-form-item :label="ui('上级分组', 'Parent Group')">
        <el-select
          v-model="form.parentId"
          class="workbench-group-dialog__select"
          clearable
          :placeholder="ui('根目录', 'Root')"
        >
          <el-option :label="ui('根目录', 'Root')" :value="null" />
          <el-option
            v-for="group in groupOptions"
            :key="group.id"
            :label="group.label"
            :value="group.id"
          />
        </el-select>
      </el-form-item>
    </el-form>

    <template #footer>
      <div class="workbench-group-dialog__footer">
        <el-button @click="requestClose">{{ ui('取消', 'Cancel') }}</el-button>
        <el-button type="primary" :loading="loading" :disabled="!canSubmit" @click="submit">
          {{ ui('保存', 'Save') }}
        </el-button>
      </div>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import DcDialog from '@/components/shared/DcDialog.vue'
import { datacenterLocale } from '@/i18n/runtime'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)

type WorkbenchGroupOption = {
  id: string
  label: string
}

type WorkbenchGroupValue = {
  id?: string | number | null
  name?: string | null
  parentId?: string | number | null
}

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    mode: 'create' | 'edit' | 'rename'
    title?: string
    group?: WorkbenchGroupValue | null
    groupOptions?: WorkbenchGroupOption[]
    loading?: boolean
    initialParentId?: string | null
    maxNameLength?: number
  }>(),
  {
    title: '',
    group: null,
    groupOptions: () => [],
    loading: false,
    initialParentId: null,
    maxNameLength: 100,
  },
)

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'submit', value: { name: string; parentId: string | null }): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const dialogRef = ref<InstanceType<typeof DcDialog> | null>(null)
const form = reactive({
  name: '',
  parentId: null as string | null,
})

const title = computed(() => {
  if (props.title) return props.title
  if (props.mode === 'create') return ui('新建分组', 'New Group')
  return ui('编辑分组', 'Edit Group')
})

const canSubmit = computed(() => {
  const name = form.name.trim()
  if (!name || props.loading) return false
  if (props.mode !== 'create') {
    const currentParentId = props.group?.parentId ? String(props.group.parentId) : null
    return name !== (props.group?.name || '') || form.parentId !== currentParentId
  }
  return true
})

const isDirty = computed(() => {
  if (!visible.value) return false
  if (props.mode !== 'create') {
    const currentParentId = props.group?.parentId ? String(props.group.parentId) : null
    return form.name.trim() !== (props.group?.name || '') || form.parentId !== currentParentId
  }
  return form.name.trim().length > 0 || Boolean(form.parentId)
})

function resetForm() {
  form.name = props.mode === 'create' ? '' : props.group?.name || ''
  form.parentId =
    props.mode === 'create'
      ? props.initialParentId || null
      : props.group?.parentId
        ? String(props.group.parentId)
        : null
}

function submit() {
  if (!canSubmit.value) return
  emit('submit', {
    name: form.name.trim(),
    parentId: form.parentId || null,
  })
}

function requestClose() {
  void dialogRef.value?.requestClose()
}

function closeSilently() {
  dialogRef.value?.closeSilently()
}

watch(
  () => [props.modelValue, props.group?.id, props.mode, props.initialParentId] as const,
  () => {
    if (props.modelValue) resetForm()
  },
  { immediate: true },
)

defineExpose({ closeSilently })
</script>

<style scoped>
.workbench-group-dialog {
  display: grid;
  gap: 2px;
}

.workbench-group-dialog__select {
  width: 100%;
}

.workbench-group-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
