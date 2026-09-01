<template>
  <el-dropdown trigger="click" placement="bottom-start" @command="handleCommand">
    <button
      type="button"
      class="ops-section-switcher"
      :aria-label="t('opsConsole.sections.switchAria', { current: currentSection.label })"
    >
      <el-icon><component :is="currentSection.icon" /></el-icon>
      <span>{{ currentSection.label }}</span>
      <el-icon class="ops-section-switcher__arrow"><ArrowDown /></el-icon>
    </button>
    <template #dropdown>
      <el-dropdown-menu class="ops-section-menu">
        <el-dropdown-item
          v-for="section in visibleSections"
          :key="section.value"
          :command="section.value"
          :class="{ 'is-current': section.value === modelValue }"
        >
          <el-icon><component :is="section.icon" /></el-icon>
          <span>{{ section.label }}</span>
        </el-dropdown-item>
      </el-dropdown-menu>
    </template>
  </el-dropdown>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowDown, Cpu, List, Monitor, UploadFilled } from '@element-plus/icons-vue'

export type OpsSection = 'environments' | 'nodes' | 'deployments' | 'tasks'

const props = defineProps<{
  modelValue: OpsSection
  canAdministerOperations: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: OpsSection]
}>()

const { t } = useI18n()

const sections = computed(() => [
  {
    value: 'environments' as const,
    label: t('opsConsole.sections.environments'),
    icon: Monitor,
    administratorOnly: true,
  },
  {
    value: 'nodes' as const,
    label: t('opsConsole.sections.nodes'),
    icon: Cpu,
    administratorOnly: true,
  },
  {
    value: 'deployments' as const,
    label: t('opsConsole.sections.deployments'),
    icon: UploadFilled,
    administratorOnly: false,
  },
  {
    value: 'tasks' as const,
    label: t('opsConsole.sections.tasks'),
    icon: List,
    administratorOnly: false,
  },
])

const visibleSections = computed(() =>
  sections.value.filter((section) => props.canAdministerOperations || !section.administratorOnly),
)
const currentSection = computed(
  () =>
    visibleSections.value.find((section) => section.value === props.modelValue) ||
    visibleSections.value[0],
)

function handleCommand(command: OpsSection) {
  emit('update:modelValue', command)
}
</script>

<style scoped>
.ops-section-switcher {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 118px;
  height: 34px;
  padding: 0 10px;
  border: 1px solid var(--ck-border, var(--el-border-color));
  border-radius: 9px;
  color: var(--ck-text-primary, var(--el-text-color-primary));
  background: var(--ck-bg-card, var(--el-bg-color));
  font: inherit;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition:
    border-color 0.18s ease,
    background 0.18s ease;
}
.ops-section-switcher:hover {
  border-color: var(--ck-primary, var(--el-color-primary));
  background: var(--ck-bg-hover, var(--el-fill-color-light));
}
.ops-section-switcher > span {
  flex: 1;
  text-align: left;
}
.ops-section-switcher :deep(.el-icon) {
  flex: 0 0 auto;
  font-size: 15px;
}
.ops-section-switcher__arrow {
  color: var(--ck-text-muted, var(--el-text-color-secondary));
  font-size: 12px !important;
}
</style>

<style>
.ops-section-menu {
  min-width: 152px;
}
.ops-section-menu .el-dropdown-menu__item {
  gap: 9px;
  margin: 2px 5px;
  border-radius: 7px;
}
.ops-section-menu .el-dropdown-menu__item.is-current {
  color: var(--ck-primary, var(--el-color-primary));
  background: var(--ck-primary-light, var(--el-color-primary-light-9));
  font-weight: 600;
}
</style>
