<template>
  <div ref="rootRef" class="project-tag-filter">
    <button
      type="button"
      :disabled="disabled"
      class="project-tag-filter__trigger"
      :class="{ 'is-active': visible || selectedIdSet.size > 0 }"
      data-testid="overview-tag-filter"
      @click.stop="toggleVisible"
    >
      <el-icon><CollectionTag /></el-icon>
      <span>{{ triggerLabel }}</span>
    </button>

    <Transition name="project-tag-filter">
      <div
        v-if="visible && !disabled"
        class="project-tag-filter__panel"
        @click.stop
      >
        <div v-if="searchable" class="project-tag-filter__search">
          <el-icon><Search /></el-icon>
          <input
            v-model="keyword"
            type="text"
            :placeholder="resolvedPlaceholder"
          />
        </div>

        <div class="project-tag-filter__list">
          <label
            v-for="tag in filteredOptions"
            :key="tag.id"
            class="project-tag-filter__item"
          >
            <input
              type="checkbox"
              :checked="selectedIdSet.has(tag.id)"
              @change="toggleTag(tag.id)"
            />
            <span class="project-tag-filter__name">{{ tag.name }}</span>
            <button
              v-if="deletable"
              type="button"
              class="project-tag-filter__delete"
              :aria-label="`${t('projectManagement.deleteTag')} ${tag.name}`"
              @click.stop.prevent="requestDeleteTag(tag)"
            >
              <el-icon><Delete /></el-icon>
            </button>
          </label>
        </div>

        <p
          v-if="filteredOptions.length === 0"
          class="project-tag-filter__empty"
        >
          {{ t('projectManagement.noTagOptions') }}
        </p>

        <label class="project-tag-filter__group-row">
          <input v-model="groupByTags" type="checkbox" />
          <span>{{ t('projectManagement.groupByTag') }}</span>
        </label>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { CollectionTag, Delete, Search } from '@element-plus/icons-vue'
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
    deletable?: boolean
    maxSelected?: number
  }>(),
  {
    modelValue: () => [],
    options: () => [],
    label: '',
    placeholder: '',
    searchable: true,
    disabled: false,
    deletable: false,
    maxSelected: 10,
  },
)

const { t } = useI18n()

const emit = defineEmits<{
  (event: 'update:modelValue', value: string[]): void
  (event: 'change', value: string[]): void
  (event: 'delete-tag', value: TagOption): void
  (event: 'limit', value: number): void
}>()

const visible = ref(false)
const keyword = ref('')
const groupByTags = ref(false)
const rootRef = ref<HTMLElement | null>(null)

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

const selectedIdSet = computed(() => new Set(normalizeSelection(props.modelValue)))

const commitSelection = (selection: string[]) => {
  const normalized = normalizeSelection(selection)
  emit('update:modelValue', normalized)
  emit('change', normalized)
}

const toggleVisible = () => {
  if (props.disabled) {
    return
  }
  visible.value = !visible.value
  if (visible.value) {
    keyword.value = ''
  }
}

const toggleTag = (tagId: string) => {
  const next = new Set(selectedIdSet.value)
  if (next.has(tagId)) {
    next.delete(tagId)
  } else {
    if (props.maxSelected > 0 && next.size >= props.maxSelected) {
      emit('limit', props.maxSelected)
      return
    }
    next.add(tagId)
  }
  commitSelection([...next])
}

const requestDeleteTag = (tag: TagOption) => {
  emit('delete-tag', tag)
}

const handleDocumentClick = (event: MouseEvent) => {
  if (!visible.value) {
    return
  }
  const target = event.target as Node | null
  if (target && rootRef.value?.contains(target)) {
    return
  }
  visible.value = false
}

watch(
  () => visible.value,
  () => {
    if (visible.value) {
      document.addEventListener('click', handleDocumentClick)
    } else {
      document.removeEventListener('click', handleDocumentClick)
    }
  },
)

onBeforeUnmount(() => {
  document.removeEventListener('click', handleDocumentClick)
})
</script>

<style scoped>
.project-tag-filter {
  position: relative;
  display: inline-flex;
}

.project-tag-filter__trigger {
  display: inline-flex;
  height: 32px;
  align-items: center;
  gap: 8px;
  border: 1px solid transparent;
  border-radius: 10px;
  background: rgba(0, 0, 0, 0.04);
  padding: 0 12px;
  color: var(--ck-text-secondary);
  box-shadow: none;
  cursor: pointer;
  font-size: 13px;
  font-weight: 400;
  transition: all 0.2s ease;
}

.project-tag-filter__trigger:hover,
.project-tag-filter__trigger.is-active {
  border-color: transparent;
  background: rgba(0, 0, 0, 0.08);
  color: var(--ck-text-primary);
  box-shadow: none;
}

.project-tag-filter__trigger:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.project-tag-filter__panel {
  position: absolute;
  top: calc(100% + 10px);
  left: 0;
  z-index: 80;
  width: 280px;
  border: 1px solid #d9e0e8;
  border-radius: 14px;
  background: #fff;
  box-shadow: 0 16px 36px rgba(15, 23, 42, 0.16);
  padding: 12px 14px;
}

.project-tag-filter__search {
  display: flex;
  height: 32px;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 0 9px;
  color: #94a3b8;
}

.project-tag-filter__search input {
  width: 100%;
  min-width: 0;
  border: 0;
  outline: 0;
  color: #334155;
  font-size: 13px;
}

.project-tag-filter__list {
  display: flex;
  max-height: 220px;
  flex-direction: column;
  overflow-y: auto;
}

.project-tag-filter__item,
.project-tag-filter__group-row {
  display: flex;
  min-height: 42px;
  align-items: center;
  gap: 12px;
  color: #475569;
  cursor: pointer;
  font-size: 13px;
}

.project-tag-filter__item input,
.project-tag-filter__group-row input {
  width: 15px;
  height: 15px;
  margin: 0;
  accent-color: var(--el-color-primary);
}

.project-tag-filter__name {
  min-width: 0;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-tag-filter__delete {
  display: inline-flex;
  width: 26px;
  height: 26px;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 7px;
  background: transparent;
  color: #94a3b8;
  cursor: pointer;
  font-size: 14px;
}

.project-tag-filter__delete:hover {
  background: #fef2f2;
  color: #ef4444;
}

.project-tag-filter__empty {
  margin: 8px 0;
  color: #94a3b8;
  font-size: 12px;
  text-align: center;
}

.project-tag-filter__group-row {
  margin-top: 8px;
  border-top: 1px solid #e2e8f0;
  padding-top: 10px;
  font-size: 14px;
}

.project-tag-filter-enter-active,
.project-tag-filter-leave-active {
  transition: opacity 0.16s ease, transform 0.16s ease;
}

.project-tag-filter-enter-from,
.project-tag-filter-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}

html.dark .project-tag-filter__trigger,
[data-theme='dark'] .project-tag-filter__trigger {
  border-color: transparent;
  background: rgba(255, 255, 255, 0.06);
  color: var(--ck-text-secondary);
}

html.dark .project-tag-filter__trigger:hover,
html.dark .project-tag-filter__trigger.is-active,
[data-theme='dark'] .project-tag-filter__trigger:hover,
[data-theme='dark'] .project-tag-filter__trigger.is-active {
  background: rgba(255, 255, 255, 0.1);
  color: var(--ck-text-primary);
}

html.dark .project-tag-filter__panel,
[data-theme='dark'] .project-tag-filter__panel {
  border-color: rgba(148, 163, 184, 0.22);
  background: #1e293b;
}

html.dark .project-tag-filter__item,
html.dark .project-tag-filter__group-row,
[data-theme='dark'] .project-tag-filter__item,
[data-theme='dark'] .project-tag-filter__group-row {
  color: #cbd5e1;
}
</style>
