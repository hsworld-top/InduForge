<template>
  <el-dialog
    :model-value="visible"
    :title="dialogTitle"
    width="min(680px, 92vw)"
    append-to-body
    destroy-on-close
    class="project-publish-dialog"
    @update:model-value="emit('update:visible', $event)"
  >
    <div v-if="activeView === 'versions'" v-loading="versionLoading" class="version-manager">
      <button type="button" class="version-manager-back" @click="activeView = 'publish'">
        <el-icon><ArrowLeft /></el-icon>
        <span>{{ t('projectManagement.backToPublish') }}</span>
      </button>

      <div class="version-list-header">
        <span>{{ t('projectManagement.version') }}</span>
        <span>{{ t('projectManagement.versionStatus') }}</span>
        <span>{{ t('projectManagement.versionRefCount') }}</span>
        <span>{{ t('projectManagement.createdAt') }}</span>
        <span />
      </div>

      <div v-if="sortedVersions.length" class="version-list">
        <div v-for="version in sortedVersions" :key="version.id" class="version-list-row">
          <strong>v{{ version.version }}</strong>
          <el-tag size="small" effect="light" :type="versionStatusType(version.status)">
            {{ versionStatusLabel(version.status) }}
          </el-tag>
          <span>{{ version.nodeDeploymentRefCount || 0 }}</span>
          <time>{{ formatVersionTime(version.createdAt) }}</time>
          <el-tooltip
            :content="
              canDeleteManagedVersion(version)
                ? t('projectManagement.deleteVersion')
                : t('projectManagement.deleteVersionBlocked')
            "
            placement="top"
          >
            <button
              type="button"
              class="version-delete-button"
              :disabled="!canDeleteManagedVersion(version)"
              @click="deleteManagedVersion(version)"
            >
              <el-icon><Delete /></el-icon>
            </button>
          </el-tooltip>
        </div>
      </div>
      <div v-else class="version-list-empty">
        {{ t('projectManagement.noReleaseVersion') }}
      </div>
    </div>

    <div v-else v-loading="loading" class="publish-dialog-body">
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

      <p v-if="mode === 'DEV'" class="publish-mode-hint">
        开发模式使用服务端生成的开发快照，不需要选择版本。
      </p>

      <div v-if="mode === 'RELEASE'" class="publish-field">
        <label for="publish-release">{{ t('projectManagement.releasedVersion') }}</label>
        <el-select
          id="publish-release"
          v-model="applicationVersionId"
          size="small"
          :loading="versionLoading"
          :placeholder="t('projectManagement.chooseRelease')"
          @change="syncPlacements"
        >
          <el-option
            v-for="version in readyVersions"
            :key="version.id"
            :value="version.id"
            :label="version.name ? `${version.version} · ${version.name}` : version.version"
          />
        </el-select>
      </div>

      <div v-if="availableEnvironments.length > 1" class="publish-field">
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
            v-for="environment in availableEnvironments"
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
          </div>
          <el-select
            v-model="placements[engine.key]"
            size="small"
            :data-testid="`publish-engine-${engine.key}`"
            :placeholder="t('projectManagement.chooseNode')"
            :disabled="nodeLoading"
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
      <div v-if="activeView === 'publish'" class="publish-dialog-footer">
        <el-button v-if="mode === 'RELEASE'" text @click="activeView = 'versions'">
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
            existingDeployment
              ? '更新部署'
              : mode === 'RELEASE'
                ? t('projectManagement.publishAndDeploy')
                : t('projectManagement.deployDevMode')
          }}
        </el-button>
      </div>
      <div v-else class="publish-dialog-footer">
        <span class="publish-dialog-footer__spacer" />
        <el-button @click="emit('update:visible', false)">
          {{ t('projectManagement.close') }}
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { ArrowLeft, Delete } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'
import {
  opsAPI,
  type ApplicationVersion,
  type OpsNode,
  type ProjectDeployment,
  type RuntimeEnvironment,
} from '@/api/ops.api'
import request, { getApiErrorMessage } from '@/utils/request'
import { formatDateTime } from '@/utils'

type PublishMode = 'DEV' | 'RELEASE'
type EngineKey = 'base' | 'compute' | 'alarm' | 'collector'

interface ProjectSummary {
  id: string
  name: string
}

interface PublishPayload {
  projectId: string
  mode: 'development' | 'production'
  environmentId: string
  applicationVersionId: string | null
  placements: Record<EngineKey, string | null>
}

interface ManagedApplicationVersion extends ApplicationVersion {
  nodeDeploymentRefCount?: number
  mode?: string
}

const props = defineProps<{
  visible: boolean
  project: ProjectSummary | null
}>()

const emit = defineEmits<{
  'update:visible': [visible: boolean]
  confirm: [payload: PublishPayload]
}>()

const { t } = useI18n()
const mode = ref<PublishMode>('DEV')
const activeView = ref<'publish' | 'versions'>('publish')
const loading = ref(false)
const versionLoading = ref(false)
const environmentLoading = ref(false)
const nodeLoading = ref(false)
const loadError = ref('')
const environments = ref<RuntimeEnvironment[]>([])
const nodes = ref<OpsNode[]>([])
const versions = ref<ManagedApplicationVersion[]>([])
const deployments = ref<ProjectDeployment[]>([])
const environmentId = ref('')
const applicationVersionId = ref('')
const placements = reactive<Record<EngineKey, string>>({
  base: '',
  compute: '',
  alarm: '',
  collector: '',
})

const readyVersions = computed(() => versions.value.filter((item) => item.status === 'ready'))
const selectedCapabilities = computed(() => {
  const source =
    mode.value === 'RELEASE'
      ? readyVersions.value.find((item) => item.id === applicationVersionId.value)
      : versions.value.find((item) => item.mode === 'development' || item.version === '__DEV__')
  const capabilities = (source?.capabilities || source?.manifest?.capabilities || []) as unknown[]
  return new Set(capabilities.map((item) => String(item).toLowerCase()))
})
const engineRows = computed(() => {
  const rows: Array<{ key: EngineKey; label: string }> = [
    { key: 'base', label: t('projectManagement.baseEngine') },
  ]
  for (const [key, label] of [
    ['compute', 'computeEngine'],
    ['alarm', 'alarmEngine'],
    ['collector', 'collectionEngine'],
  ]) {
    if (selectedCapabilities.value.has(key))
      rows.push({ key: key as EngineKey, label: t(`projectManagement.${label}`) })
  }
  return rows
})

// 单一运行环境不是用户决策，隐藏选择器并直接使用它，避免两个部署入口语义不一致。
const availableEnvironments = computed(() =>
  environments.value.filter(
    (item) => item.desiredStatus !== 'deleting' && item.status !== 'uninitialized',
  ),
)
const existingDeployment = computed(() =>
  deployments.value.find(
    (item) => item.projectId === props.project?.id && item.environmentId === environmentId.value,
  ),
)

const dialogTitle = computed(() =>
  activeView.value === 'versions'
    ? `${t('projectManagement.versionManageDialog')} · ${props.project?.name || ''}`
    : t('projectManagement.deployDialog', { name: props.project?.name || '' }),
)

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

const versionStatusType = (status: string | undefined) => {
  if (status === 'success') return 'success'
  if (status === 'failed') return 'danger'
  if (status === 'building') return 'warning'
  return 'info'
}

const versionStatusLabel = (status: string | undefined) =>
  t(`projectManagement.versionStatus_${status || 'pending'}`)

const formatVersionTime = (value: string | undefined) => (value ? formatDateTime(value) : '--')

const canDeleteManagedVersion = (version: ManagedApplicationVersion) =>
  (version.nodeDeploymentRefCount || 0) === 0 &&
  ['success', 'failed'].includes(String(version.status || ''))

const loadVersions = async () => {
  if (!props.project?.id) return
  versionLoading.value = true
  try {
    const result = await opsAPI.listProjectVersions(props.project.id, {
      page: 1,
      pageSize: 200,
    })
    versions.value = result.items as ManagedApplicationVersion[]
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, t('projectManagement.versionManageLoadFailed')))
  } finally {
    versionLoading.value = false
  }
}

const deleteManagedVersion = async (version: ManagedApplicationVersion) => {
  if (!canDeleteManagedVersion(version)) return
  try {
    await ElMessageBox.confirm(
      t('projectManagement.deleteVersionConfirm', { version: version.version }),
      t('projectManagement.deleteConfirmTitle'),
      { type: 'warning' },
    )
  } catch {
    return
  }

  versionLoading.value = true
  try {
    await request.delete(`/publish/deployment/${version.id}`)
    ElMessage.success(t('projectManagement.deleteVersionSuccess'))
    await loadVersions()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, t('projectManagement.deleteVersionFailed')))
  } finally {
    versionLoading.value = false
  }
}

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
  if (mode.value === 'RELEASE' && !applicationVersionId.value) return false
  return engineRows.value.every((engine) => Boolean(placements[engine.key]))
})

const setDefaultPlacements = () => {
  const readyNodes = nodes.value.filter(isNodeReady)
  const firstNodeId = readyNodes[0]?.id || ''
  const readyNodeIds = new Set(readyNodes.map((node) => node.id))
  engineRows.value.forEach(({ key }) => {
    if (!readyNodeIds.has(placements[key])) placements[key] = firstNodeId
  })
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
    const [environmentResult, versionResult, deploymentResult] = await Promise.all([
      opsAPI.listRuntimeEnvironments({ page: 1, pageSize: 200 }),
      opsAPI.listProjectVersions(props.project.id, { page: 1, pageSize: 200 }),
      opsAPI.listProjectDeployments({ page: 1, pageSize: 200, projectId: props.project.id }),
    ])
    environments.value = environmentResult.items.filter((item) => item.desiredStatus !== 'deleting')
    versions.value = versionResult.items as ManagedApplicationVersion[]
    deployments.value = deploymentResult.items
    applicationVersionId.value = readyVersions.value[0]?.id || ''
    const defaultEnvironment =
      availableEnvironments.value.find((item) => item.isDefault) || availableEnvironments.value[0]
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
  activeView.value = 'publish'
  environmentId.value = ''
  applicationVersionId.value = ''
  environments.value = []
  nodes.value = []
  versions.value = []
  deployments.value = []
  loadError.value = ''
  placements.base = ''
  placements.compute = ''
  placements.alarm = ''
  placements.collector = ''
}

const syncPlacements = () => setDefaultPlacements()

const submit = () => {
  if (!props.project?.id || !canConfirm.value) return
  emit('confirm', {
    projectId: props.project.id,
    mode: mode.value === 'RELEASE' ? 'production' : 'development',
    environmentId: environmentId.value,
    applicationVersionId: mode.value === 'RELEASE' ? applicationVersionId.value : null,
    placements: {
      base: placements.base,
      compute: placements.compute,
      alarm: placements.alarm,
      collector: placements.collector,
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
  height: 372px;
  padding: 4px 20px 18px;
  overflow-y: auto;
}

:global(.project-publish-dialog .el-dialog__footer) {
  padding: 12px 20px 14px;
  border-top: 1px solid var(--ck-border-light);
}

.publish-dialog-body {
  min-height: 350px;
}

.publish-mode-hint {
  margin: 12px 0;
  color: var(--ck-text-muted);
  font-size: 12px;
}

.version-manager {
  min-height: 250px;
}

.version-manager-back {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 30px;
  margin-bottom: 12px;
  border: 0;
  border-radius: 8px;
  padding: 0 9px;
  color: var(--ck-text-secondary);
  background: var(--ck-bg-tertiary);
  cursor: pointer;
  font-size: 12px;
}

.version-manager-back:hover {
  color: var(--ck-primary);
  background: var(--ck-primary-light);
}

.version-list-header,
.version-list-row {
  display: grid;
  grid-template-columns: minmax(100px, 0.7fr) 92px 64px minmax(150px, 1.2fr) 32px;
  align-items: center;
  column-gap: 12px;
}

.version-list-header {
  min-height: 34px;
  border: 1px solid var(--ck-border-light);
  border-radius: var(--ck-radius-md) var(--ck-radius-md) 0 0;
  padding: 0 12px;
  color: var(--ck-text-muted);
  background: var(--ck-bg-tertiary);
  font-size: 11px;
}

.version-list {
  overflow: hidden;
  border: 1px solid var(--ck-border-light);
  border-top: 0;
  border-radius: 0 0 var(--ck-radius-md) var(--ck-radius-md);
}

.version-list-row {
  min-height: 48px;
  padding: 0 12px;
  color: var(--ck-text-secondary);
  background: var(--ck-bg-secondary);
  font-size: 12px;
}

.version-list-row + .version-list-row {
  border-top: 1px solid var(--ck-border-light);
}

.version-list-row strong {
  color: var(--ck-text-primary);
  font-size: 13px;
  font-weight: 600;
}

.version-list-row time {
  color: var(--ck-text-muted);
  font-size: 11px;
}

.version-delete-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: 0;
  border-radius: 8px;
  color: var(--ck-text-muted);
  background: transparent;
  cursor: pointer;
}

.version-delete-button:hover:not(:disabled) {
  color: var(--ck-danger);
  background: rgb(239 68 68 / 10%);
}

.version-delete-button:disabled {
  cursor: not-allowed;
  opacity: 0.35;
}

.version-list-empty {
  display: flex;
  min-height: 160px;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--ck-border-light);
  border-top: 0;
  border-radius: 0 0 var(--ck-radius-md) var(--ck-radius-md);
  color: var(--ck-text-muted);
  background: var(--ck-bg-secondary);
  font-size: 12px;
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
  .version-list-header,
  .version-list-row {
    grid-template-columns: minmax(82px, 1fr) 82px 42px 30px;
  }

  .version-list-header > :nth-child(4),
  .version-list-row > :nth-child(4) {
    display: none;
  }

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
