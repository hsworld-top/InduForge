<template>
  <div ref="rootRef" class="project-filter">
    <button
      type="button"
      :disabled="disabled"
      class="project-filter__trigger"
      :class="{ 'is-active': visible || activeFilterCount > 0 }"
      data-testid="overview-composite-filter"
      @click.stop="toggleVisible"
    >
      <el-icon><Filter /></el-icon>
      <span>{{ triggerLabel }}</span>
    </button>

    <Transition name="project-filter">
      <div v-if="visible && !disabled" class="project-filter__panel" @click.stop>
        <button
          type="button"
          class="project-filter__section project-filter__section--all"
          :class="{ 'is-selected': activeFilterCount === 0 }"
          @click="selectAll"
        >
          <span>{{ t('projectManagement.filterAll') }}</span>
        </button>

        <section class="project-filter__section">
          <div class="project-filter__section-title">{{ t('projectManagement.runtimeMode') }}</div>
          <div class="project-filter__options">
            <label
              v-for="option in resolvedRuntimeModeOptions"
              :key="option.value"
              class="project-filter__option"
            >
              <input v-model="draftFilters.runtimeModes" type="checkbox" :value="option.value" />
              <span>{{ option.label }}</span>
            </label>
          </div>
        </section>

        <section class="project-filter__section">
          <div class="project-filter__section-title">{{ t('projectManagement.deployStatus') }}</div>
          <div class="project-filter__options">
            <label
              v-for="option in resolvedDeployStatusOptions"
              :key="option.value"
              class="project-filter__option"
            >
              <input v-model="draftFilters.deployStatuses" type="checkbox" :value="option.value" />
              <span>{{ option.label }}</span>
            </label>
          </div>
        </section>

        <section class="project-filter__section">
          <div class="project-filter__section-title">{{ t('projectManagement.visibility') }}</div>
          <div class="project-filter__options">
            <label
              v-for="option in resolvedVisibilityOptions"
              :key="option.value"
              class="project-filter__option"
            >
              <input v-model="draftFilters.visibility" type="checkbox" :value="option.value" />
              <span>{{ option.label }}</span>
            </label>
          </div>
        </section>

        <div class="project-filter__footer">
          <button type="button" class="project-filter__ghost" @click="handleReset">
            {{ t('common.reset') }}
          </button>
          <button type="button" class="project-filter__primary" @click="handleConfirm">
            {{ t('projectManagement.applyFilter') }}
          </button>
        </div>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { Filter } from '@element-plus/icons-vue'
import { useI18n } from 'vue-i18n'
import type { ProjectDeployStatus, ProjectRuntimeMode, ProjectVisibility } from '@/api/project.api'
import type { ProjectOverviewCompositeFilters } from './project-overview.types'
import { normalizeProjectOverviewFilters } from './use-project-filters'

type RuntimeModeOption = {
  label: string
  value: ProjectRuntimeMode
}

type DeployStatusOption = {
  label: string
  value: ProjectDeployStatus
}

type VisibilityOption = {
  label: string
  value: ProjectVisibility
}

const createEmptyCompositeFilters = (): ProjectOverviewCompositeFilters => ({
  runtimeModes: [],
  deployStatuses: [],
  visibility: [],
  createdBy: '',
})

const props = withDefaults(
  defineProps<{
    modelValue?: Partial<ProjectOverviewCompositeFilters>
    label?: string
    disabled?: boolean
    runtimeModeOptions?: RuntimeModeOption[]
    deployStatusOptions?: DeployStatusOption[]
    visibilityOptions?: VisibilityOption[]
  }>(),
  {
    modelValue: () => ({}),
    label: '',
    disabled: false,
    runtimeModeOptions: () => [],
    deployStatusOptions: () => [],
    visibilityOptions: () => [],
  },
)

const emit = defineEmits<{
  (event: 'update:modelValue', value: ProjectOverviewCompositeFilters): void
  (event: 'change', value: ProjectOverviewCompositeFilters): void
}>()

const { t } = useI18n()
const visible = ref(false)
const rootRef = ref<HTMLElement | null>(null)
const draftFilters = reactive<ProjectOverviewCompositeFilters>(createEmptyCompositeFilters())

const resolvedRuntimeModeOptions = computed<RuntimeModeOption[]>(() =>
  props.runtimeModeOptions.length > 0
    ? props.runtimeModeOptions
    : [
        { label: t('projectManagement.modeDisplayDev'), value: 'DEV' },
        { label: t('projectManagement.modeDisplayRelease'), value: 'RELEASE' },
      ],
)

const resolvedDeployStatusOptions = computed<DeployStatusOption[]>(() =>
  props.deployStatusOptions.length > 0
    ? props.deployStatusOptions
    : [
        { label: t('projectManagement.deployStatusPending'), value: 'pending' },
        { label: t('projectManagement.deployStatusDeploying'), value: 'deploying' },
        { label: t('projectManagement.deployStatusRunning'), value: 'running' },
        { label: t('projectManagement.deployStatusStopped'), value: 'stopped' },
        { label: t('projectManagement.deployStatusError'), value: 'error' },
        { label: t('projectManagement.deployStatusRollback'), value: 'rollback' },
      ],
)

const resolvedVisibilityOptions = computed<VisibilityOption[]>(() =>
  props.visibilityOptions.length > 0
    ? props.visibilityOptions
    : [
        { label: t('projectManagement.visibilityShared'), value: 'internal' },
        { label: t('projectManagement.visibilityPrivate'), value: 'private' },
      ],
)

const normalizeCompositeFilters = (
  value: Partial<ProjectOverviewCompositeFilters>,
): ProjectOverviewCompositeFilters => {
  const seed = createEmptyCompositeFilters()
  return normalizeProjectOverviewFilters({
    composite: {
      runtimeModes: value.runtimeModes ?? seed.runtimeModes,
      deployStatuses: value.deployStatuses ?? seed.deployStatuses,
      visibility: value.visibility ?? seed.visibility,
      createdBy: '',
    },
  }).composite
}

const normalizedModelValue = computed<ProjectOverviewCompositeFilters>(() =>
  normalizeCompositeFilters(props.modelValue || {}),
)

const activeFilterCount = computed(() => {
  let count = 0
  if (normalizedModelValue.value.runtimeModes.length > 0) {
    count += 1
  }
  if (normalizedModelValue.value.deployStatuses.length > 0) {
    count += 1
  }
  if (normalizedModelValue.value.visibility.length > 0) {
    count += 1
  }
  return count
})

const triggerLabel = computed(() => {
  const baseLabel = props.label || t('projectManagement.filterAll')
  if (activeFilterCount.value <= 0) {
    return baseLabel
  }
  return `${baseLabel} (${activeFilterCount.value})`
})

const applyToDraft = (value: ProjectOverviewCompositeFilters) => {
  draftFilters.runtimeModes = [...value.runtimeModes]
  draftFilters.deployStatuses = [...value.deployStatuses]
  draftFilters.visibility = [...value.visibility]
  draftFilters.createdBy = ''
}

const emitFilterChange = (nextValue: ProjectOverviewCompositeFilters) => {
  const normalized = normalizeCompositeFilters(nextValue)
  emit('update:modelValue', normalized)
  emit('change', normalized)
}

const toggleVisible = () => {
  if (props.disabled) {
    return
  }
  visible.value = !visible.value
  if (visible.value) {
    applyToDraft(normalizedModelValue.value)
  }
}

const selectAll = () => {
  applyToDraft(createEmptyCompositeFilters())
  emitFilterChange(createEmptyCompositeFilters())
  visible.value = false
}

const handleReset = () => {
  applyToDraft(createEmptyCompositeFilters())
}

const handleConfirm = () => {
  emitFilterChange(draftFilters)
  visible.value = false
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

watch(
  () => props.modelValue,
  () => {
    if (!visible.value) {
      applyToDraft(normalizedModelValue.value)
    }
  },
  { deep: true, immediate: true },
)

onBeforeUnmount(() => {
  document.removeEventListener('click', handleDocumentClick)
})
</script>

<style scoped>
.project-filter {
  position: relative;
  display: inline-flex;
}

.project-filter__trigger {
  display: inline-flex;
  height: 32px;
  align-items: center;
  gap: 8px;
  border: 1px solid transparent;
  border-radius: 10px;
  background: rgba(0, 0, 0, 0.04);
  padding: 0 12px;
  color: var(--ck-text-secondary);
  cursor: pointer;
  font-size: 13px;
  font-weight: 400;
  transition: all 0.2s ease;
}

.project-filter__trigger:hover,
.project-filter__trigger.is-active {
  border-color: transparent;
  background: rgba(0, 0, 0, 0.08);
  color: var(--ck-text-primary);
  box-shadow: none;
}

.project-filter__trigger:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.project-filter__panel {
  position: absolute;
  top: calc(100% + 10px);
  left: 0;
  z-index: 80;
  width: 300px;
  border: 1px solid #d9e0e8;
  border-radius: 14px;
  background: #fff;
  box-shadow: 0 16px 36px rgba(15, 23, 42, 0.16);
  padding: 10px 12px;
}

.project-filter__section {
  display: block;
  width: 100%;
  border: 0;
  border-bottom: 1px solid #edf2f7;
  background: transparent;
  padding: 10px 0;
  text-align: left;
}

.project-filter__section--all {
  min-height: 36px;
  border-radius: 8px;
  border-bottom: 0;
  padding: 0 10px;
  color: #334155;
  cursor: pointer;
  font-weight: 700;
}

.project-filter__section--all:hover,
.project-filter__section--all.is-selected {
  background: #f1f5f9;
  color: var(--el-color-primary);
}

.project-filter__section-title {
  margin-bottom: 8px;
  color: #334155;
  font-size: 13px;
  font-weight: 700;
}

.project-filter__options {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.project-filter__option {
  display: inline-flex;
  min-height: 28px;
  align-items: center;
  gap: 6px;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 0 9px;
  color: #475569;
  cursor: pointer;
  font-size: 12px;
}

.project-filter__option input {
  width: 13px;
  height: 13px;
  margin: 0;
  accent-color: var(--el-color-primary);
}

.project-filter__footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding-top: 10px;
}

.project-filter__ghost,
.project-filter__primary {
  height: 30px;
  border-radius: 8px;
  padding: 0 12px;
  cursor: pointer;
  font-size: 12px;
}

.project-filter__ghost {
  border: 1px solid #e2e8f0;
  background: #fff;
  color: #64748b;
}

.project-filter__primary {
  border: 1px solid var(--el-color-primary);
  background: var(--el-color-primary);
  color: #fff;
}

.project-filter-enter-active,
.project-filter-leave-active {
  transition:
    opacity 0.16s ease,
    transform 0.16s ease;
}

.project-filter-enter-from,
.project-filter-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}

html.dark .project-filter__trigger,
[data-theme='dark'] .project-filter__trigger {
  background: rgba(255, 255, 255, 0.06);
  color: var(--ck-text-secondary);
}

html.dark .project-filter__trigger:hover,
html.dark .project-filter__trigger.is-active,
[data-theme='dark'] .project-filter__trigger:hover,
[data-theme='dark'] .project-filter__trigger.is-active {
  background: rgba(255, 255, 255, 0.1);
  color: var(--ck-text-primary);
}

html.dark .project-filter__panel,
[data-theme='dark'] .project-filter__panel {
  border-color: rgba(148, 163, 184, 0.22);
  background: #1e293b;
}

html.dark .project-filter__section-title,
html.dark .project-filter__section--all,
html.dark .project-filter__option,
[data-theme='dark'] .project-filter__section-title,
[data-theme='dark'] .project-filter__section--all,
[data-theme='dark'] .project-filter__option {
  color: #cbd5e1;
}
</style>
