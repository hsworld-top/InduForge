<template>
  <el-popover
    v-model:visible="visible"
    placement="bottom-start"
    :width="320"
    trigger="click"
    :disabled="disabled"
    @update:visible="handleVisibleChange"
  >
    <template #reference>
      <el-button
        size="small"
        :disabled="disabled"
        class="!h-8 !px-3"
        data-testid="overview-tag-filter"
      >
        <el-icon><CollectionTag /></el-icon>
        <span>{{ triggerLabel }}</span>
      </el-button>
    </template>

    <div class="space-y-3">
      <div class="flex items-center justify-between">
        <span class="text-xs font-semibold text-gray-700 dark:text-gray-300">{{ label }}</span>
        <el-button
          text
          size="small"
          :disabled="draftSelection.length === 0"
          @click="handleClear"
        >
          {{ t('projectManagement.clear') }}
        </el-button>
      </div>

      <el-input
        v-if="searchable"
        v-model="keyword"
        size="small"
        clearable
        :placeholder="resolvedPlaceholder"
      >
        <template #prefix>
          <el-icon><Search /></el-icon>
        </template>
      </el-input>

      <el-scrollbar max-height="220px">
        <el-checkbox-group v-model="draftSelection" class="flex flex-col gap-2">
          <el-checkbox
            v-for="tag in filteredOptions"
            :key="tag.id"
            :label="tag.id"
            class="!h-auto !mr-0"
          >
            <div class="flex flex-col">
              <span class="text-sm text-gray-700 dark:text-gray-200">{{ tag.name }}</span>
              <span
                v-if="tag.description"
                class="text-xs text-gray-400 dark:text-gray-500 leading-4 mt-0.5"
              >
                {{ tag.description }}
              </span>
            </div>
          </el-checkbox>
        </el-checkbox-group>
      </el-scrollbar>

      <p
        v-if="filteredOptions.length === 0"
        class="text-xs text-gray-400 dark:text-gray-500 text-center py-2"
      >
        {{ t('projectManagement.noTagOptions') }}
      </p>

      <div class="flex items-center justify-end gap-2 border-t border-gray-100 dark:border-gray-700 pt-3">
        <el-button size="small" @click="handleCancel">{{ t('common.cancel') }}</el-button>
        <el-button size="small" type="primary" @click="handleConfirm">{{ t('common.confirm') }}</el-button>
      </div>
    </div>
  </el-popover>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { CollectionTag, Search } from '@element-plus/icons-vue'
import { useI18n } from 'vue-i18n'
import type { ProjectOverviewTag } from './project-overview.types'

type TagOption = Pick<ProjectOverviewTag, 'id' | 'name'> &
  Partial<Pick<ProjectOverviewTag, 'description' | 'sortOrder'>>

const props = withDefaults(
  defineProps<{
    modelValue?: string[]
    options?: TagOption[]
    label?: string
    placeholder?: string
    searchable?: boolean
    disabled?: boolean
  }>(),
  {
    modelValue: () => [],
    options: () => [],
    label: '',
    placeholder: '',
    searchable: true,
    disabled: false,
  },
)

const { t } = useI18n()

const emit = defineEmits<{
  (event: 'update:modelValue', value: string[]): void
  (event: 'change', value: string[]): void
}>()

const visible = ref(false)
const keyword = ref('')
const draftSelection = ref<string[]>([])
const committedSelection = ref<string[]>([])

const sortedOptions = computed<TagOption[]>(() =>
  [...props.options].sort((left, right) => {
    const sortDelta = (left.sortOrder ?? 0) - (right.sortOrder ?? 0)
    if (sortDelta !== 0) {
      return sortDelta
    }
    return left.name.localeCompare(right.name, 'zh-Hans-CN')
  }),
)

const filteredOptions = computed<TagOption[]>(() => {
  const normalizedKeyword = keyword.value.trim().toLowerCase()
  if (!normalizedKeyword) {
    return sortedOptions.value
  }

  return sortedOptions.value.filter((tag) => {
    const target = `${tag.name} ${tag.description ?? ''}`.toLowerCase()
    return target.includes(normalizedKeyword)
  })
})

const resolvedPlaceholder = computed(
  () => props.placeholder || t('projectManagement.tagSearchPlaceholder'),
)

const triggerLabel = computed(() => {
  const baseLabel = props.label || t('projectManagement.tagFilter')
  if (props.modelValue.length <= 0) {
    return baseLabel
  }
  return `${baseLabel} (${props.modelValue.length})`
})

const normalizeSelection = (selection: string[]) => [
  ...new Set(selection.map((item) => item.trim()).filter(Boolean)),
]

const syncCommittedFromModel = () => {
  committedSelection.value = normalizeSelection(props.modelValue)
}

const syncDraftFromCommitted = () => {
  draftSelection.value = [...committedSelection.value]
}

const handleVisibleChange = (nextVisible: boolean) => {
  visible.value = nextVisible
  if (nextVisible) {
    syncCommittedFromModel()
    syncDraftFromCommitted()
    keyword.value = ''
    return
  }

  syncDraftFromCommitted()
  keyword.value = ''
}

const commitSelection = (selection: string[]) => {
  const normalized = normalizeSelection(selection)
  emit('update:modelValue', normalized)
  emit('change', normalized)
}

const handleConfirm = () => {
  const normalized = normalizeSelection(draftSelection.value)
  committedSelection.value = normalized
  commitSelection(normalized)
  visible.value = false
}

const handleCancel = () => {
  syncDraftFromCommitted()
  keyword.value = ''
  visible.value = false
}

const handleClear = () => {
  draftSelection.value = []
}

watch(
  () => props.modelValue,
  () => {
    if (!visible.value) {
      syncCommittedFromModel()
      syncDraftFromCommitted()
    }
  },
  { deep: true, immediate: true },
)
</script>
