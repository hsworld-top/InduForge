<template>
  <el-dialog
    :model-value="visible"
    :title="dialogTitle"
    width="min(680px, 92vw)"
    append-to-body
    destroy-on-close
    :show-close="!operationLocked"
    :close-on-click-modal="!operationLocked"
    :close-on-press-escape="!operationLocked"
    class="project-publish-dialog"
    @update:model-value="handleVisibleChange"
  >
    <div v-if="activeView === 'versions'" v-loading="versionLoading" class="version-manager">
      <div class="version-manager-toolbar">
        <button type="button" class="version-manager-back" @click="activeView = 'publish'">
          <el-icon><ArrowLeft /></el-icon>
          <span>{{ t('projectManagement.backToPublish') }}</span>
        </button>
        <el-button
          type="primary"
          size="small"
          :loading="versionCreating"
          data-testid="create-project-version"
          @click="createManagedVersion"
        >
          <el-icon><Plus /></el-icon>
          {{ t('projectManagement.createReleaseVersion') }}
        </el-button>
      </div>

      <div class="version-list-header">
        <span>{{ t('projectManagement.version') }}</span>
        <span>{{ t('projectManagement.versionStatus') }}</span>
        <span>{{ t('projectManagement.versionRefCount') }}</span>
        <span>{{ t('projectManagement.createdAt') }}</span>
          <span>{{ t('projectManagement.actions') }}</span>
      </div>

      <div v-if="sortedVersions.length" class="version-list">
        <div v-for="version in sortedVersions" :key="version.id" class="version-list-row">
          <strong>v{{ version.version }}</strong>
          <el-tag size="small" effect="light" :type="versionStatusType(version.status)">
            {{ versionStatusLabel(version.status) }}
          </el-tag>
          <span>{{ version.nodeDeploymentRefCount || 0 }}</span>
          <time>{{ formatVersionTime(version.createdAt) }}</time>
          <div class="version-row-actions">
            <el-tooltip :content="versionRestoreReason(version)" :disabled="versionRestoreAvailability(version).allowed" placement="top">
              <span>
                <el-button
                  text
                  size="small"
                  class="version-restore-button"
                  :disabled="!versionRestoreAvailability(version).allowed || operationLocked"
                  @click="openRestoreConfirm(version)"
                >
                  {{ t('projectManagement.restoreDevelopment') }}
                </el-button>
              </span>
            </el-tooltip>
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
                :disabled="!canDeleteManagedVersion(version) || operationLocked"
                @click="deleteManagedVersion(version)"
              >
                <el-icon><Delete /></el-icon>
              </button>
            </el-tooltip>
          </div>
        </div>
      </div>
      <div v-else class="version-list-empty">
        {{ t('projectManagement.noReleaseVersion') }}
      </div>
    </div>

    <div v-else-if="activeView === 'restore-confirm'" class="restore-panel restore-confirm">
      <div class="restore-version-mark">v{{ selectedRestoreVersion?.version }}</div>
      <p>{{ t('projectManagement.restoreDevelopmentSummary') }}</p>
      <ul>
        <li>{{ t('projectManagement.restoreDeploymentUnaffected') }}</li>
        <li>{{ t('projectManagement.restoreEditorsReload') }}</li>
      </ul>
    </div>

    <div v-else-if="activeView === 'restore-progress'" class="restore-panel restore-progress" aria-live="polite">
      <div class="restore-stage-status">{{ t(currentRestoreStage.labelKey) }}</div>
      <div class="restore-stage-list">
        <div v-for="(label, index) in restoreStageLabels" :key="label" :class="['restore-stage', { 'is-active': index === currentRestoreStage.active, 'is-done': index < currentRestoreStage.active }]">
          <span class="restore-stage-dot" />
          <span>{{ t(label) }}</span>
        </div>
      </div>
      <p v-if="restoreTask?.taskId" class="restore-task-id">
        {{ t('projectManagement.restoreTaskId', { id: restoreTask.taskId }) }}
      </p>
    </div>

    <div v-else-if="activeView === 'restore-success'" class="restore-panel restore-result restore-result--success">
      <h3>{{ t('projectManagement.restoreSuccess', { version: selectedRestoreVersion?.version }) }}</h3>
      <dl>
        <div><dt>{{ t('projectManagement.restoreBackup') }}</dt><dd>{{ t('projectManagement.restoreBackupCreated') }}</dd></div>
        <div><dt>{{ t('projectManagement.restoreTime') }}</dt><dd>{{ formatVersionTime(restoreTask?.completedAt) }}</dd></div>
      </dl>
    </div>

    <div v-else-if="activeView === 'restore-failure'" class="restore-panel restore-result restore-result--failure">
      <h3>{{ t('projectManagement.restoreFailure') }}</h3>
      <p>{{ restoreTask?.errorMessage || restoreRequestError || t('projectManagement.restoreFailedFallback') }}</p>
      <p v-if="restoreTask?.rolledBack" class="restore-rollback">
        {{ t('projectManagement.restoreRolledBack') }}
      </p>
    </div>

    <div v-else v-loading="loading" class="publish-dialog-body">
      <p v-if="initialDeployment?.observedStatus === 'stopped'" class="publish-mode-hint">
        更新后将恢复并启动当前部署。
      </p>
      <div class="publish-mode-tabs" role="tablist">
        <button
          type="button"
          role="tab"
          data-testid="publish-mode-dev"
          :aria-selected="mode === 'DEV'"
          :disabled="submitting"
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
          :disabled="submitting"
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
          :disabled="submitting"
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
          :disabled="submitting"
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
            :disabled="nodeLoading || submitting"
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
        <el-button v-if="mode === 'RELEASE'" text :disabled="submitting" @click="activeView = 'versions'">
          {{ t('projectManagement.versionManage') }}
        </el-button>
        <span class="publish-dialog-footer__spacer" />
        <el-button :disabled="submitting" @click="emit('update:visible', false)">
          {{ t('projectManagement.cancel') }}
        </el-button>
        <el-button
          type="primary"
          class="publish-confirm-button"
          data-testid="publish-confirm"
          :disabled="!canConfirm"
          :loading="submitting"
          @click="submit"
        >
          {{ publishCta.label }}
        </el-button>
      </div>
      <div v-else-if="activeView === 'versions'" class="publish-dialog-footer">
        <span class="publish-dialog-footer__spacer" />
        <el-button :disabled="operationLocked" @click="requestClose">
          {{ t('projectManagement.close') }}
        </el-button>
      </div>
      <div v-else-if="activeView === 'restore-confirm'" class="publish-dialog-footer">
        <span class="publish-dialog-footer__spacer" />
        <el-button @click="activeView = 'versions'">{{ t('projectManagement.cancel') }}</el-button>
        <el-button type="primary" data-testid="restore-development-confirm" @click="submitRestore">{{ t('projectManagement.backupAndContinue') }}</el-button>
      </div>
      <div v-else-if="activeView === 'restore-success'" class="publish-dialog-footer restore-result-actions">
        <el-button @click="returnToVersions">{{ t('projectManagement.backToVersionManage') }}</el-button>
        <span class="publish-dialog-footer__spacer" />
        <el-button @click="openRestoredWorkspace('datacenter')">{{ t('projectManagement.enterDataCenter') }}</el-button>
        <el-button type="primary" @click="openRestoredWorkspace('designer')">{{ t('projectManagement.enterDesignCenter') }}</el-button>
      </div>
      <div v-else-if="activeView === 'restore-failure'" class="publish-dialog-footer">
        <span class="publish-dialog-footer__spacer" />
        <el-button @click="returnToVersions">{{ t('projectManagement.backToVersionManage') }}</el-button>
      </div>
      <div v-else class="publish-dialog-footer">
        <span class="publish-dialog-footer__spacer" />
        <el-button disabled>{{ t('projectManagement.restoreInProgress') }}</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { ArrowLeft, Delete, Plus } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'
import {
  opsAPI,
  type ApplicationVersion,
  type DevelopmentRestoreTask,
  type OpsNode,
  type ProjectDeployment,
  type RuntimeEnvironment,
} from '@/api/ops.api'
import request, { getApiErrorMessage } from '@/utils/request'
import { formatDateTime } from '@/utils'
import { publishActionContextMatches, type ProjectDeploymentPrimary } from './project-deployment-action'
import { projectPublishCta } from './project-publish-cta'
import {
  isRestoreTerminal,
  restoreStage,
  restoreView,
  versionRestoreAvailability,
} from './project-development-restore'

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
  initialDeployment?: ProjectDeploymentPrimary | null
  initialAction?: { kind: string; label: string } | null
  submitting?: boolean
}>()

const emit = defineEmits<{
  'update:visible': [visible: boolean]
  confirm: [payload: PublishPayload]
  'restore-complete': [payload: { projectId: string }]
  'open-restored-workspace': [payload: { projectId: string; target: 'designer' | 'datacenter' }]
}>()

const { t } = useI18n()
const mode = ref<PublishMode>('DEV')
type ActiveView =
  | 'publish'
  | 'versions'
  | 'restore-confirm'
  | 'restore-progress'
  | 'restore-success'
  | 'restore-failure'
const activeView = ref<ActiveView>('publish')
const loading = ref(false)
const versionLoading = ref(false)
const versionCreating = ref(false)
const environmentLoading = ref(false)
const nodeLoading = ref(false)
const loadError = ref('')
const environments = ref<RuntimeEnvironment[]>([])
const nodes = ref<OpsNode[]>([])
const versions = ref<ManagedApplicationVersion[]>([])
const deployments = ref<ProjectDeployment[]>([])
const developmentRequirements = ref<EngineKey[]>(['base'])
const developmentRequirementsReady = ref(false)
const environmentId = ref('')
const applicationVersionId = ref('')
const selectedRestoreVersion = ref<ManagedApplicationVersion | null>(null)
const restoreTask = ref<DevelopmentRestoreTask | null>(null)
const restoreRequestError = ref('')
let restorePollTimer: ReturnType<typeof setTimeout> | null = null
let restoreRequestRevision = 0
let restoreCompletionNotified = false
const placements = reactive<Record<EngineKey, string>>({
  base: '',
  compute: '',
  alarm: '',
  collector: '',
})

// 发布接口沿用 success 展示态，部署接口使用 ready 持久态；两者都代表制品已可部署。
const isDeployableVersion = (version: ManagedApplicationVersion) =>
  ['ready', 'success'].includes(String(version.status || ''))
const readyVersions = computed(() => versions.value.filter(isDeployableVersion))
const selectedCapabilities = computed(() => {
  if (mode.value === 'DEV') return new Set(developmentRequirements.value)
  const source = readyVersions.value.find((item) => item.id === applicationVersionId.value)
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
const publishCta = computed(() =>
  projectPublishCta(existingDeployment.value, mode.value, applicationVersionId.value),
)
const operationLocked = computed(
  () => Boolean(props.submitting) || activeView.value === 'restore-progress',
)
const currentRestoreStage = computed(() =>
  restoreStage(restoreTask.value?.state || 'queued'),
)
const restoreStageLabels = [
  'projectManagement.restoreBackupStage',
  'projectManagement.restoreContentStage',
  'projectManagement.restoreVerifyStage',
]

const initialContextMatches = computed(() =>
  publishActionContextMatches(props.initialDeployment, mode.value, environmentId.value),
)
const dialogTitle = computed(() =>
  activeView.value === 'restore-confirm'
    ? t('projectManagement.restoreDevelopmentTitle', { version: selectedRestoreVersion.value?.version || '' })
    : activeView.value === 'restore-progress'
      ? `${t('projectManagement.restoreProgressTitle')} · ${props.project?.name || ''}`
      : activeView.value === 'restore-success' || activeView.value === 'restore-failure'
        ? `${t('projectManagement.restoreResultTitle')} · ${props.project?.name || ''}`
  : initialContextMatches.value && props.initialAction?.label && activeView.value !== 'versions'
    ? `${props.initialAction.label} · ${props.project?.name || ''}`
    : initialContextMatches.value && activeView.value === 'versions' && props.initialAction?.label === '发布新版本'
      ? `${props.initialAction.label} · ${props.project?.name || ''}`
      : activeView.value === 'versions'
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
  if (status === 'success' || status === 'ready') return 'success'
  if (status === 'failed') return 'danger'
  if (status === 'building') return 'warning'
  return 'info'
}

const versionStatusLabel = (status: string | undefined) =>
  t(`projectManagement.versionStatus_${status || 'pending'}`)

const formatVersionTime = (value: string | undefined) => (value ? formatDateTime(value) : '--')

const canDeleteManagedVersion = (version: ManagedApplicationVersion) =>
  (version.nodeDeploymentRefCount || 0) === 0 &&
  ['ready', 'failed'].includes(String(version.status || ''))

const versionRestoreReason = (version: ManagedApplicationVersion) => {
  const availability = versionRestoreAvailability(version)
  if (availability.reason) return availability.reason
  return availability.reasonKey ? t(availability.reasonKey) : ''
}

const openRestoreConfirm = (version: ManagedApplicationVersion) => {
  if (!versionRestoreAvailability(version).allowed || operationLocked.value) return
  selectedRestoreVersion.value = version
  restoreTask.value = null
  restoreRequestError.value = ''
  activeView.value = 'restore-confirm'
}

const clearRestorePoll = () => {
  if (restorePollTimer) clearTimeout(restorePollTimer)
  restorePollTimer = null
}

const applyRestoreTask = (task: DevelopmentRestoreTask) => {
  restoreTask.value = task
  activeView.value = `restore-${restoreView(task)}` as ActiveView
  if (task.state === 'succeeded' && !restoreCompletionNotified && props.project?.id) {
    restoreCompletionNotified = true
    emit('restore-complete', { projectId: props.project.id })
  }
  if (isRestoreTerminal(task)) clearRestorePoll()
}

const pollRestoreTask = async (taskId: string, revision: number) => {
  try {
    const task = await opsAPI.getDevelopmentRestoreTask(taskId)
    if (revision !== restoreRequestRevision || !props.visible) return
    applyRestoreTask(task)
    if (!isRestoreTerminal(task)) {
      restorePollTimer = setTimeout(() => void pollRestoreTask(taskId, revision), 1500)
    }
  } catch (error) {
    if (revision !== restoreRequestRevision || !props.visible) return
    restoreRequestError.value = getApiErrorMessage(error, t('projectManagement.restoreStatusUnavailable'))
    restorePollTimer = setTimeout(() => void pollRestoreTask(taskId, revision), 3000)
  }
}

const submitRestore = async () => {
  const version = selectedRestoreVersion.value
  if (
    !version ||
    !versionRestoreAvailability(version).allowed ||
    operationLocked.value
  ) return
  const revision = ++restoreRequestRevision
  restoreRequestError.value = ''
  activeView.value = 'restore-progress'
  restoreTask.value = { taskId: '', state: 'queued', versionId: version.id, version: version.version }
  try {
    const task = await opsAPI.restoreProjectDevelopment(version.id)
    if (revision !== restoreRequestRevision) return
    applyRestoreTask(task)
    if (!isRestoreTerminal(task)) void pollRestoreTask(String(task.taskId), revision)
  } catch (error) {
    if (revision !== restoreRequestRevision) return
    restoreRequestError.value = getApiErrorMessage(error, t('projectManagement.restoreSubmitFailed'))
    restoreTask.value = {
      taskId: '',
      state: 'failed',
      versionId: version.id,
      version: version.version,
      errorMessage: restoreRequestError.value,
    }
    activeView.value = 'restore-failure'
  }
}

const returnToVersions = async () => {
  activeView.value = 'versions'
  await loadVersions()
}

const openRestoredWorkspace = (target: 'designer' | 'datacenter') => {
  if (!props.project?.id) return
  emit('open-restored-workspace', { projectId: props.project.id, target })
  emit('update:visible', false)
}

const requestClose = () => {
  if (!operationLocked.value) emit('update:visible', false)
}

const handleVisibleChange = (visible: boolean) => {
  if (visible || !operationLocked.value) emit('update:visible', visible)
}

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

const createManagedVersion = async () => {
  if (!props.project?.id || versionCreating.value) return
  versionCreating.value = true
  try {
    // 版本号和制品配置由服务端基于当前工程快照生成，页面不接受人工覆盖。
    const created = (await opsAPI.createProjectVersion(
      props.project.id,
    )) as ManagedApplicationVersion
    await loadVersions()
    applicationVersionId.value = created.id
    activeView.value = 'publish'
    syncPlacements()
    ElMessage.success(
      t('projectManagement.createReleaseVersionSuccess', { version: created.version }),
    )
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, t('projectManagement.createReleaseVersionFailed')))
  } finally {
    versionCreating.value = false
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
  if (!props.project?.id || !environmentId.value || nodeLoading.value || props.submitting || publishCta.value.disabled) return false
  if (mode.value === 'DEV' && !developmentRequirementsReady.value) return false
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
    const initial = props.initialDeployment
    mode.value = initial?.mode === 'production' ? 'RELEASE' : 'DEV'
    applicationVersionId.value = initial?.applicationVersionId || readyVersions.value[0]?.id || ''
    const defaultEnvironment =
      availableEnvironments.value.find((item) => item.id === initial?.environmentId) ||
      availableEnvironments.value.find((item) => item.isDefault) ||
      availableEnvironments.value[0]
    environmentId.value = defaultEnvironment?.id || ''
    for (const key of ['base', 'compute', 'alarm', 'collector'] as const) {
      placements[key] = initial?.placements?.[key] || ''
    }
    await loadNodes(environmentId.value)
    if (props.initialAction?.label === '发布新版本') activeView.value = 'versions'
  } catch (error) {
    loadError.value = getApiErrorMessage(error, t('projectManagement.publishContextLoadFailed'))
  } finally {
    environmentLoading.value = false
    loading.value = false
  }
}

const loadDevelopmentRequirements = async () => {
  if (!props.project?.id) return
  developmentRequirementsReady.value = false
  try {
    developmentRequirements.value = await opsAPI.getDevelopmentDeploymentRequirements(
      props.project.id,
    )
    setDefaultPlacements()
    developmentRequirementsReady.value = true
  } catch (error) {
    developmentRequirementsReady.value = false
    throw error
  }
}

const reset = () => {
  clearRestorePoll()
  restoreRequestRevision += 1
  mode.value = 'DEV'
  activeView.value = 'publish'
  environmentId.value = ''
  applicationVersionId.value = ''
  environments.value = []
  nodes.value = []
  versions.value = []
  deployments.value = []
  developmentRequirements.value = ['base']
  developmentRequirementsReady.value = false
  loadError.value = ''
  selectedRestoreVersion.value = null
  restoreTask.value = null
  restoreRequestError.value = ''
  restoreCompletionNotified = false
  placements.base = ''
  placements.compute = ''
  placements.alarm = ''
  placements.collector = ''
}

const syncPlacements = () => setDefaultPlacements()

watch(mode, (value) => {
  if (value === 'DEV')
    void loadDevelopmentRequirements().catch((error) => {
      loadError.value = getApiErrorMessage(error, t('projectManagement.publishContextLoadFailed'))
    })
})

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
    void loadDevelopmentRequirements().catch((error) => {
      loadError.value = getApiErrorMessage(error, t('projectManagement.publishContextLoadFailed'))
    })
  },
)

onBeforeUnmount(() => {
  clearRestorePoll()
  restoreRequestRevision += 1
})
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

.version-manager-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.version-manager-back {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 30px;
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
  grid-template-columns: minmax(86px, 0.7fr) 76px 54px minmax(120px, 1fr) minmax(190px, 1.2fr);
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

.version-row-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 2px;
  min-width: 0;
}

.version-restore-button {
  padding: 4px 6px;
  font-size: 12px;
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

.restore-panel {
  min-height: 300px;
  color: var(--ck-text-secondary);
  font-size: 13px;
}

.restore-panel h3 {
  margin: 12px 0 10px;
  color: var(--ck-text-primary);
  font-size: 15px;
  font-weight: 600;
}

.restore-confirm {
  padding: 32px 36px;
  border: 1px solid var(--ck-border-light);
  border-radius: var(--ck-radius-md);
  background: var(--ck-bg-card);
}

.restore-version-mark {
  display: inline-flex;
  padding: 4px 9px;
  border-radius: 999px;
  color: var(--ck-primary);
  background: var(--ck-primary-light);
  font-size: 12px;
  font-weight: 600;
}

.restore-confirm p,
.restore-confirm ul,
.restore-result p {
  margin: 8px 0;
  line-height: 1.7;
}

.restore-confirm ul {
  padding-left: 20px;
  color: var(--ck-text-muted);
}

.restore-progress {
  padding: 54px 48px;
}

.restore-stage-status {
  margin-bottom: 22px;
  color: var(--ck-text-primary);
  font-size: 14px;
  font-weight: 600;
}

.restore-stage-list {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
}

.restore-stage {
  display: flex;
  align-items: center;
  gap: 7px;
  min-height: 36px;
  color: var(--ck-text-muted);
  font-size: 12px;
}

.restore-stage-dot {
  width: 8px;
  height: 8px;
  flex: 0 0 auto;
  border-radius: 50%;
  background: var(--ck-border);
}

.restore-stage.is-active {
  color: var(--ck-primary);
  font-weight: 600;
}

.restore-stage.is-active .restore-stage-dot {
  background: var(--ck-primary);
  box-shadow: 0 0 0 4px var(--ck-primary-light);
}

.restore-stage.is-done .restore-stage-dot {
  background: var(--ck-success);
}

.restore-task-id {
  margin-top: 24px;
  color: var(--ck-text-muted);
  font-size: 11px;
}

.restore-result {
  padding: 52px 44px;
}

.restore-result--success h3 {
  color: var(--ck-success);
}

.restore-result--failure h3 {
  color: var(--ck-danger);
}

.restore-result dl {
  display: grid;
  gap: 8px;
  margin: 20px 0 0;
}

.restore-result dl div {
  display: grid;
  grid-template-columns: 74px minmax(0, 1fr);
}

.restore-result dt {
  color: var(--ck-text-muted);
}

.restore-result dd {
  margin: 0;
  color: var(--ck-text-primary);
}

.restore-rollback {
  color: var(--ck-success);
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
