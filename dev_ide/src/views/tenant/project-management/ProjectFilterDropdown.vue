<template>
  <el-popover
    v-model:visible="visible"
    placement="bottom-start"
    :width="360"
    trigger="click"
    :disabled="disabled"
    @update:visible="handleVisibleChange"
  >
    <template #reference>
      <el-button
        size="small"
        :disabled="disabled"
        class="!h-8 !px-3"
        data-testid="overview-composite-filter"
      >
        <el-icon><Filter /></el-icon>
        <span>{{ triggerLabel }}</span>
        <el-icon class="text-[10px] ml-1"><ArrowDown /></el-icon>
      </el-button>
    </template>

    <div class="space-y-4">
      <section class="space-y-2">
        <p class="text-xs font-semibold text-gray-700 dark:text-gray-300">
          {{ t('projectManagement.runtimeMode') }}
        </p>
        <el-checkbox-group v-model="draftFilters.runtimeModes" class="flex flex-wrap gap-2">
          <el-checkbox
            v-for="option in resolvedRuntimeModeOptions"
            :key="option.value"
            :label="option.value"
            border
            size="small"
          >
            {{ option.label }}
          </el-checkbox>
        </el-checkbox-group>
      </section>

      <section class="space-y-2">
        <p class="text-xs font-semibold text-gray-700 dark:text-gray-300">
          {{ t('projectManagement.deployStatus') }}
        </p>
        <el-checkbox-group v-model="draftFilters.deployStatuses" class="flex flex-wrap gap-2">
          <el-checkbox
            v-for="option in resolvedDeployStatusOptions"
            :key="option.value"
            :label="option.value"
            border
            size="small"
          >
            {{ option.label }}
          </el-checkbox>
        </el-checkbox-group>
      </section>

      <section class="space-y-2">
        <p class="text-xs font-semibold text-gray-700 dark:text-gray-300">
          {{ t('projectManagement.createdBy') }}
        </p>
        <el-input
          v-model="draftFilters.createdBy"
          size="small"
          clearable
          :placeholder="t('projectManagement.createdByPlaceholder')"
        />
      </section>

      <div class="flex items-center justify-between border-t border-gray-100 dark:border-gray-700 pt-3">
        <el-button text size="small" @click="handleReset">{{ t('common.reset') }}</el-button>
        <div class="flex items-center gap-2">
          <el-button size="small" @click="handleCancel">{{ t('common.cancel') }}</el-button>
          <el-button size="small" type="primary" @click="handleConfirm">
            {{ t('projectManagement.applyFilter') }}
          </el-button>
        </div>
      </div>
    </div>
  </el-popover>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { ArrowDown, Filter } from '@element-plus/icons-vue'
import { useI18n } from 'vue-i18n'
import type { ProjectDeployStatus, ProjectRuntimeMode } from '@/api/project.api'
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

const createEmptyCompositeFilters = (): ProjectOverviewCompositeFilters => ({
  runtimeModes: [],
  deployStatuses: [],
  createdBy: '',
})

const props = withDefaults(
  defineProps<{
    modelValue?: Partial<ProjectOverviewCompositeFilters>
    label?: string
    disabled?: boolean
    runtimeModeOptions?: RuntimeModeOption[]
    deployStatusOptions?: DeployStatusOption[]
  }>(),
  {
    modelValue: () => ({}),
    label: '',
    disabled: false,
    runtimeModeOptions: () => [],
    deployStatusOptions: () => [],
  },
)

const { t } = useI18n()

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

const emit = defineEmits<{
  (event: 'update:modelValue', value: ProjectOverviewCompositeFilters): void
  (event: 'change', value: ProjectOverviewCompositeFilters): void
}>()

const visible = ref(false)
const draftFilters = reactive<ProjectOverviewCompositeFilters>(createEmptyCompositeFilters())
const committedFilters = ref<ProjectOverviewCompositeFilters>(createEmptyCompositeFilters())

const normalizeCompositeFilters = (
  value: Partial<ProjectOverviewCompositeFilters>,
): ProjectOverviewCompositeFilters => {
  const seed = createEmptyCompositeFilters()
  return normalizeProjectOverviewFilters({
    composite: {
      runtimeModes: value.runtimeModes ?? seed.runtimeModes,
      deployStatuses: value.deployStatuses ?? seed.deployStatuses,
      createdBy: value.createdBy ?? seed.createdBy,
    },
  }).composite
}

const applyToDraft = (value: ProjectOverviewCompositeFilters) => {
  draftFilters.runtimeModes = [...value.runtimeModes]
  draftFilters.deployStatuses = [...value.deployStatuses]
  draftFilters.createdBy = value.createdBy
}

const syncCommittedFromModel = () => {
  committedFilters.value = normalizeCompositeFilters(props.modelValue || {})
}

const syncDraftFromCommitted = () => {
  applyToDraft(committedFilters.value)
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
  if (normalizedModelValue.value.createdBy) {
    count += 1
  }
  return count
})

const triggerLabel = computed(() => {
  const baseLabel = props.label || t('projectManagement.overviewCompositeFilter')
  if (activeFilterCount.value <= 0) {
    return baseLabel
  }
  return `${baseLabel} (${activeFilterCount.value})`
})

const emitFilterChange = (nextValue: ProjectOverviewCompositeFilters) => {
  const normalized = normalizeCompositeFilters(nextValue)
  emit('update:modelValue', normalized)
  emit('change', normalized)
}

const handleVisibleChange = (nextVisible: boolean) => {
  visible.value = nextVisible
  if (nextVisible) {
    syncCommittedFromModel()
    syncDraftFromCommitted()
    return
  }

  syncDraftFromCommitted()
}

const handleConfirm = () => {
  const normalized = normalizeCompositeFilters(draftFilters)
  committedFilters.value = normalized
  emitFilterChange(normalized)
  visible.value = false
}

const handleCancel = () => {
  syncDraftFromCommitted()
  visible.value = false
}

const handleReset = () => {
  applyToDraft(createEmptyCompositeFilters())
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
