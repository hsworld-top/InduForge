<template>
  <el-dialog
    :model-value="visible"
    :title="t('projectManagement.deployDialog', { name: project?.name || '' })"
    width="min(680px, 92vw)"
    append-to-body
    destroy-on-close
    class="project-publish-dialog"
    @update:model-value="emit('update:visible', $event)"
  >
    <div v-loading="loading" class="publish-dialog-body">
      <div class="publish-mode-tabs" role="tablist">
        <button
          type="button"
          role="tab"
          data-testid="publish-mode-dev"
          :aria-selected="mode === 'DEV'"
          :class="['publish-mode-tab', { 'is-active': mode === 'DEV' }]"
          @click="mode = 'DEV'"
        >
          {{ t('projectManagement.publishModeDev') }}
        </button>
        <button
          type="button"
          role="tab"
          data-testid="publish-mode-release"
          :aria-selected="mode === 'RELEASE'"
          :class="['publish-mode-tab', { 'is-active': mode === 'RELEASE' }]"
          @click="mode = 'RELEASE'"
        >
          {{ t('projectManagement.publishModeRelease') }}
        </button>
      </div>

      <div v-if="mode === 'RELEASE'" class="release-version-bar">
        <div class="release-version-item">
          <span>{{ t('projectManagement.currentVersion') }}</span>
          <strong>{{ currentVersion ? `v${currentVersion}` : '-' }}</strong>
        </div>
        <span class="release-version-arrow">→</span>
        <div class="release-version-item release-version-item--next">
          <span>{{ t('projectManagement.nextVersion') }}</span>
          <strong>v{{ nextVersion }}</strong>
        </div>
        <el-tag size="small" type="primary" effect="light">
          {{ t('projectManagement.autoIncrement') }}
        </el-tag>
      </div>

      <div class="publish-field">
        <label for="publish-environment">{{ t('projectManagement.targetEnvironment') }}</label>
        <el-select
          id="publish-environment"
          v-model="environmentId"
          size="small"
          data-testid="publish-environment-select"
          :placeholder="t('projectManagement.chooseEnvironment')"
          :loading="environmentLoading"
          @change="handleEnvironmentChange"
        >
          <el-option
            v-for="environment in environments"
            :key="environment.id"
            :value="environment.id"
            :label="environment.name"
            :disabled="environment.status === 'uninitialized' || environment.status === 'deleting'"
          />
        </el-select>
      </div>

      <section class="engine-placement" aria-labelledby="engine-placement-title">
        <div class="engine-placement-header">
          <span id="engine-placement-title">{{ t('projectManagement.runtimeEngines') }}</span>
          <span>{{ t('projectManagement.deployNode') }}</span>
        </div>

        <div v-for="engine in engineRows" :key="engine.key" class="engine-placement-row">
          <div class="engine-name">
            <strong>{{ engine.label }}</strong>
            <el-switch
              v-if="engine.optional"
              v-model="collectorEnabled"
              size="small"
              :aria-label="t('projectManagement.collectionEngine')"
            />
          </div>
          <el-select
            v-model="placements[engine.key]"
            size="small"
            :data-testid="`publish-engine-${engine.key}`"
            :placeholder="t('projectManagement.chooseNode')"
            :disabled="nodeLoading || (engine.optional && !collectorEnabled)"
            filterable
          >
            <el-option
              v-for="node in nodes"
              :key="node.id"
              :value="node.id"
              :label="nodeOptionLabel(node)"
              :disabled="!isNodeReady(node)"
            />
          </el-select>
        </div>
      </section>

      <div v-if="loadError" class="publish-load-error">{{ loadError }}</div>
    </div>

    <template #footer>
      <div class="publish-dialog-footer">
        <el-button v-if="mode === 'RELEASE'" text @click="emit('manage-versions')">
          {{ t('projectManagement.versionManage') }}
        </el-button>
        <span class="publish-dialog-footer__spacer" />
        <el-button @click="emit('update:visible', false)">
          {{ t('projectManagement.cancel') }}
        </el-button>
        <el-button
          type="primary"
          class="publish-confirm-button"
          data-testid="publish-confirm"
          :disabled="!canConfirm"
          @click="submit"
        >
          {{
            mode === 'RELEASE'
              ? t('projectManagement.publishAndDeploy')
              : t('projectManagement.deployDevMode')
          }}
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  opsAPI,
  type ApplicationVersion,
  type OpsNode,
  type RuntimeEnvironment,
} from '@/api/ops.api'
import { getApiErrorMessage } from '@/utils/request'

type PublishMode = 'DEV' | 'RELEASE'
type EngineKey = 'runtime' | 'compute' | 'alarm' | 'collection'

interface ProjectSummary {
  id: string
  name: string
}

interface PublishPayload {
  projectId: string
  mode: PublishMode
  environmentId: string
  version: string | null
  placements: Record<EngineKey, string | null>
}

const props = defineProps<{
  visible: boolean
  project: ProjectSummary | null
}>()

const emit = defineEmits<{
  'update:visible': [visible: boolean]
  'manage-versions': []
  confirm: [payload: PublishPayload]
}>()

const { t } = useI18n()
const mode = ref<PublishMode>('DEV')
const loading = ref(false)
const environmentLoading = ref(false)
const nodeLoading = ref(false)
const loadError = ref('')
const environments = ref<RuntimeEnvironment[]>([])
const nodes = ref<OpsNode[]>([])
const versions = ref<ApplicationVersion[]>([])
const environmentId = ref('')
const collectorEnabled = ref(false)
const placements = reactive<Record<EngineKey, string>>({
  runtime: '',
  compute: '',
  alarm: '',
  collection: '',
})

const engineRows = computed(() => [
  {
    key: 'runtime' as const,
    label: t('projectManagement.runtimeEngine'),
    optional: false,
  },
  {
    key: 'compute' as const,
    label: t('projectManagement.computeEngine'),
    optional: false,
  },
  {
    key: 'alarm' as const,
    label: t('projectManagement.alarmEngine'),
    optional: false,
  },
  {
    key: 'collection' as const,
    label: t('projectManagement.collectionEngine'),
    optional: true,
  },
])

const versionParts = (version: string | undefined) => {
  const normalized = String(version || '')
    .trim()
    .replace(/^v/i, '')
  if (!/^\d+(\.\d+){1,2}$/.test(normalized)) return null
  return normalized.split('.').map(Number)
}

const compareVersions = (left: string | undefined, right: string | undefined) => {
  const leftParts = versionParts(left) || []
  const rightParts = versionParts(right) || []
  const length = Math.max(leftParts.length, rightParts.length)
  for (let index = 0; index < length; index += 1) {
    const difference = (leftParts[index] || 0) - (rightParts[index] || 0)
    if (difference !== 0) return difference
  }
  return 0
}

const sortedVersions = computed(() =>
  [...versions.value]
    .filter((item) => versionParts(item.version))
    .sort((left, right) => compareVersions(right.version, left.version)),
)

const currentVersion = computed(
  () => sortedVersions.value.find((item) => item.status === 'success')?.version || '',
)

const nextVersion = computed(() => {
  const latest = sortedVersions.value[0]?.version
  const parts = versionParts(latest)
  if (!parts) return '0.1'
  if (parts.length === 3) {
    parts[2] += 1
  } else {
    parts[1] += 1
  }
  return parts.join('.')
})

const isNodeReady = (node: OpsNode) =>
  node.platform === 'linux' &&
  node.observedStatus === 'online' &&
  (!node.clusterStatus || node.clusterStatus === 'ready')

const nodeOptionLabel = (node: OpsNode) => {
  const address = node.ipAddress ? ` · ${node.ipAddress}` : ''
  const state = isNodeReady(node) ? '' : ` · ${t('projectManagement.nodeUnavailable')}`
  return `${node.name}${address}${state}`
}

const canConfirm = computed(() => {
  if (!props.project?.id || !environmentId.value || nodeLoading.value) return false
  if (!placements.runtime || !placements.compute || !placements.alarm) return false
  return !collectorEnabled.value || Boolean(placements.collection)
})

const setDefaultPlacements = () => {
  const readyNodes = nodes.value.filter(isNodeReady)
  const firstNodeId = readyNodes[0]?.id || ''
  const readyNodeIds = new Set(readyNodes.map((node) => node.id))
  ;(['runtime', 'compute', 'alarm'] as EngineKey[]).forEach((key) => {
    if (!readyNodeIds.has(placements[key])) placements[key] = firstNodeId
  })
  if (!readyNodeIds.has(placements.collection)) placements.collection = firstNodeId
}

const loadNodes = async (selectedEnvironmentId: string) => {
  nodes.value = []
  if (!selectedEnvironmentId) {
    setDefaultPlacements()
    return
  }
  nodeLoading.value = true
  try {
    const result = await opsAPI.listRuntimeEnvironmentNodes(selectedEnvironmentId, {
      page: 1,
      pageSize: 200,
    })
    nodes.value = result.items
    setDefaultPlacements()
  } catch (error) {
    loadError.value = getApiErrorMessage(error, t('projectManagement.publishContextLoadFailed'))
  } finally {
    nodeLoading.value = false
  }
}

const handleEnvironmentChange = async (value: string) => {
  loadError.value = ''
  await loadNodes(value)
}

const loadPublishContext = async () => {
  if (!props.project?.id) return
  loading.value = true
  environmentLoading.value = true
  loadError.value = ''
  try {
    const [environmentResult, versionResult] = await Promise.all([
      opsAPI.listRuntimeEnvironments({ page: 1, pageSize: 200 }),
      opsAPI.listProjectVersions(props.project.id, { page: 1, pageSize: 200 }),
    ])
    environments.value = environmentResult.items.filter((item) => item.desiredStatus !== 'deleting')
    versions.value = versionResult.items
    const defaultEnvironment =
      environments.value.find((item) => item.isDefault && item.status !== 'uninitialized') ||
      environments.value.find((item) => item.status !== 'uninitialized') ||
      environments.value[0]
    environmentId.value = defaultEnvironment?.id || ''
    await loadNodes(environmentId.value)
  } catch (error) {
    loadError.value = getApiErrorMessage(error, t('projectManagement.publishContextLoadFailed'))
  } finally {
    environmentLoading.value = false
    loading.value = false
  }
}

const reset = () => {
  mode.value = 'DEV'
  collectorEnabled.value = false
  environmentId.value = ''
  environments.value = []
  nodes.value = []
  versions.value = []
  loadError.value = ''
  placements.runtime = ''
  placements.compute = ''
  placements.alarm = ''
  placements.collection = ''
}

const submit = () => {
  if (!props.project?.id || !canConfirm.value) return
  emit('confirm', {
    projectId: props.project.id,
    mode: mode.value,
    environmentId: environmentId.value,
    version: mode.value === 'RELEASE' ? nextVersion.value : null,
    placements: {
      runtime: placements.runtime,
      compute: placements.compute,
      alarm: placements.alarm,
      collection: collectorEnabled.value ? placements.collection : null,
    },
  })
}

watch(
  () => [props.visible, props.project?.id] as const,
  ([visible]) => {
    if (!visible) return
    reset()
    void loadPublishContext()
  },
)
</script>

<style scoped>
:global(.project-publish-dialog.el-dialog) {
  overflow: hidden;
  border: 1px solid var(--ck-border);
  border-radius: var(--ck-radius-lg);
  background: var(--ck-bg-secondary);
  box-shadow: var(--ck-shadow-lg);
}

:global(.project-publish-dialog .el-dialog__header) {
  margin: 0;
  padding: 16px 20px 12px;
}

:global(.project-publish-dialog .el-dialog__title) {
  color: var(--ck-text-primary);
  font-size: 15px;
  font-weight: 600;
}

:global(.project-publish-dialog .el-dialog__headerbtn) {
  top: 8px;
  right: 10px;
}

:global(.project-publish-dialog .el-dialog__body) {
  padding: 4px 20px 18px;
}

:global(.project-publish-dialog .el-dialog__footer) {
  padding: 12px 20px 14px;
  border-top: 1px solid var(--ck-border-light);
}

.publish-dialog-body {
  min-height: 322px;
}

.publish-mode-tabs {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 3px;
  padding: 3px;
  margin-bottom: 16px;
  border-radius: 10px;
  background: rgb(15 23 42 / 4%);
}

.publish-mode-tab {
  height: 32px;
  border: 0;
  border-radius: 8px;
  color: var(--ck-text-muted);
  font-size: 13px;
  font-weight: 400;
  background: transparent;
  cursor: pointer;
  transition:
    color 0.15s ease,
    background-color 0.15s ease,
    box-shadow 0.15s ease;
}

.publish-mode-tab:hover {
  color: var(--ck-text-primary);
  background: rgb(255 255 255 / 55%);
}

.publish-mode-tab:focus-visible {
  outline: none;
  box-shadow: inset 0 0 0 2px var(--ck-primary-light);
}

.publish-mode-tab.is-active {
  color: var(--ck-primary);
  background: #ffffff;
  box-shadow: 0 2px 8px rgb(15 23 42 / 8%);
}

.release-version-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 46px;
  padding: 7px 12px;
  margin-bottom: 12px;
  border: 1px solid rgb(15 23 42 / 5%);
  border-radius: var(--ck-radius-md);
  background: var(--ck-bg-tertiary);
}

.release-version-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
  color: var(--ck-text-muted);
  font-size: 11px;
}

.release-version-item strong {
  color: var(--ck-text-primary);
  font-size: 13px;
  font-weight: 600;
}

.release-version-item--next strong {
  color: var(--ck-primary);
}

.release-version-arrow {
  color: var(--ck-text-muted);
  font-size: 12px;
}

.publish-field {
  display: grid;
  grid-template-columns: 86px minmax(0, 1fr);
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.publish-field label,
.engine-placement-header {
  color: var(--ck-text-secondary);
  font-size: 12px;
  font-weight: 500;
}

.engine-placement {
  overflow: hidden;
  border: 1px solid rgb(15 23 42 / 6%);
  border-radius: var(--ck-radius-md);
  background: var(--ck-bg-tertiary);
}

.engine-placement-header,
.engine-placement-row {
  display: grid;
  grid-template-columns: minmax(170px, 0.72fr) minmax(240px, 1.28fr);
  align-items: center;
  column-gap: 12px;
}

.engine-placement-header {
  grid-template-columns: minmax(170px, 0.72fr) minmax(240px, 1.28fr);
  padding: 8px 12px;
  color: var(--ck-text-muted);
  font-size: 11px;
  background: transparent;
}

.engine-placement-row {
  min-height: 52px;
  padding: 7px 12px;
  border-top: 1px solid var(--ck-border-light);
  background: var(--ck-bg-card);
}

.engine-name {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.engine-name strong {
  color: var(--ck-text-secondary);
  font-size: 12px;
  font-weight: 500;
}

.engine-placement :deep(.el-select__wrapper),
.publish-field :deep(.el-select__wrapper) {
  min-height: 30px;
  border: 1px solid var(--ck-border);
  border-radius: 8px;
  background: var(--ck-bg-card);
  box-shadow: none;
}

.engine-placement :deep(.el-select__wrapper:hover),
.publish-field :deep(.el-select__wrapper:hover) {
  border-color: var(--ck-primary);
}

.engine-placement :deep(.el-switch) {
  --el-switch-on-color: var(--ck-primary);
}

.publish-load-error {
  margin-top: 12px;
  color: var(--ck-danger);
  font-size: 12px;
}

.publish-dialog-footer {
  display: flex;
  align-items: center;
  width: 100%;
}

.publish-dialog-footer__spacer {
  flex: 1;
}

.publish-dialog-footer :deep(.el-button) {
  min-width: 72px;
  height: 32px;
  border-radius: var(--ck-radius-md);
  font-size: 12px;
}

.publish-dialog-footer :deep(.publish-confirm-button) {
  border-color: transparent;
  background: var(--ck-gradient-primary);
  box-shadow: 0 8px 16px rgb(29 78 216 / 18%);
}

html.dark .publish-mode-tabs,
[data-theme='dark'] .publish-mode-tabs {
  background: rgb(255 255 255 / 6%);
}

html.dark .publish-mode-tab:hover,
[data-theme='dark'] .publish-mode-tab:hover {
  background: rgb(255 255 255 / 8%);
}

html.dark .publish-mode-tab.is-active,
[data-theme='dark'] .publish-mode-tab.is-active {
  color: #ffffff;
  background: #334155;
}

@media (max-width: 640px) {
  .engine-placement-header {
    display: none;
  }

  .engine-placement-row {
    grid-template-columns: minmax(112px, 0.78fr) minmax(145px, 1.22fr);
    column-gap: 8px;
    padding-inline: 10px;
  }

  .publish-field {
    grid-template-columns: 1fr;
    gap: 6px;
  }
}
</style>
