<template>
  <div class="projects">
    <div class="toolbar">
      <el-button type="primary" @click="showDeployDialog = true">
        <el-icon><Plus /></el-icon>
        {{ t('local.deploy') }}
      </el-button>
    </div>

    <el-card class="box-card panel-card">
      <template #header>
        <span>{{ t('local.listTitle') }}</span>
      </template>
      <el-table v-loading="nodeStore.loading" :data="localProjects" style="width: 100%">
        <el-table-column prop="id" :label="t('local.projectId')" min-width="170" />
        <el-table-column prop="currentVersion" :label="t('local.version')" width="120" />
        <el-table-column :label="t('local.status')" width="110">
          <template #default="{ row }">
            <el-tag class="status-tag" :type="getStatusType(row.status)">{{
              formatStatus(row.status)
            }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('local.runtimeStatus')" width="140">
          <template #default="{ row }">{{
            formatStatus(row.runtimeStatus?.state || 'stopped')
          }}</template>
        </el-table-column>
        <el-table-column :label="t('local.pid')" width="100">
          <template #default="{ row }">{{ row.runtimeStatus?.pid || t('common.na') }}</template>
        </el-table-column>
        <el-table-column :label="t('local.lastStartedAt')" width="180">
          <template #default="{ row }">{{ formatTime(row.lastStartedAt) }}</template>
        </el-table-column>
        <el-table-column :label="t('local.actions')" min-width="320" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="row.status !== 'running'"
              type="primary"
              size="small"
              @click="startProject(row.id)"
            >
              {{ t('local.start') }}
            </el-button>
            <el-button v-else type="warning" size="small" @click="stopProject(row.id)">{{
              t('local.stop')
            }}</el-button>
            <el-button type="info" size="small" @click="restartProject(row.id)">{{
              t('local.restart')
            }}</el-button>
            <el-button type="success" size="small" @click="viewLogs(row.id)">{{
              t('local.logs')
            }}</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-empty
        v-if="!localProjects.length && !nodeStore.loading"
        :description="t('local.empty')"
      />
    </el-card>

    <el-dialog v-model="showDeployDialog" :title="t('local.deployDialogTitle')" width="600px">
      <el-form :model="deployForm" label-width="120px">
        <el-form-item :label="t('local.ifpFile')" required>
          <div class="ifp-picker">
            <el-input :model-value="deployForm.ifpPackage || t('local.fileNotSelected')" readonly />
            <el-button @click="triggerFilePick">{{ t('local.selectFile') }}</el-button>
          </div>
          <input
            ref="ifpFileInput"
            class="hidden-file-input"
            type="file"
            accept=".ifp"
            @change="handleIfpFileChange"
          />
        </el-form-item>
        <el-form-item :label="t('local.autoStart')">
          <el-switch v-model="deployForm.autoStart" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showDeployDialog = false">{{ t('local.cancel') }}</el-button>
        <el-button type="primary" :loading="deploying" @click="deployProject">{{
          t('local.confirmDeploy')
        }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { useNodeStore } from '@/store/nodeStore'
import { useI18nText } from '@/composables/useI18nText'

const router = useRouter()
const nodeStore = useNodeStore()
const { locale, t } = useI18nText()

const showDeployDialog = ref(false)
const deploying = ref(false)
const ifpFileInput = ref(null)
const selectedIfpFile = ref(null)

const deployForm = reactive({
  ifpPackage: '',
  autoStart: true,
})

const localProjects = computed(() =>
  nodeStore.projects.filter((project) => project?.source !== 'center'),
)

const triggerFilePick = () => {
  ifpFileInput.value?.click()
}

const handleIfpFileChange = (event) => {
  const file = event?.target?.files?.[0]
  if (!file) {
    selectedIfpFile.value = null
    deployForm.ifpPackage = ''
    return
  }

  const lowerName = file.name.toLowerCase()
  if (!lowerName.endsWith('.ifp')) {
    ElMessage.warning(t('local.onlyIfp'))
    selectedIfpFile.value = null
    deployForm.ifpPackage = ''
    event.target.value = ''
    return
  }

  selectedIfpFile.value = file
  deployForm.ifpPackage = file.name
}

const deployProject = async () => {
  if (!selectedIfpFile.value || !deployForm.ifpPackage) {
    ElMessage.warning(t('local.pleaseSelectIfp'))
    return
  }
  deploying.value = true
  const success = await nodeStore.deployProjectByIfp(selectedIfpFile.value, deployForm.autoStart)
  if (success) {
    showDeployDialog.value = false
    selectedIfpFile.value = null
    Object.assign(deployForm, { ifpPackage: '', autoStart: true })
    if (ifpFileInput.value) {
      ifpFileInput.value.value = ''
    }
  }
  deploying.value = false
}

const startProject = async (projectId) => nodeStore.startProject(projectId)
const stopProject = async (projectId) => nodeStore.stopProject(projectId)
const restartProject = async (projectId) => nodeStore.restartProject(projectId)
const viewLogs = (projectId) => router.push(`/logs/${projectId}`)

const getStatusType = (status) => {
  const map = { running: 'success', stopped: 'info', error: 'danger', deploying: 'warning' }
  return map[status] || 'info'
}

const formatStatus = (status) => t(`status.${status || 'unknown'}`)
const formatTime = (time) => (time ? new Date(time).toLocaleString(locale.value) : t('common.na'))

onMounted(async () => {
  await nodeStore.fetchProjects()
})
</script>

<style scoped>
.projects {
  display: grid;
  gap: 16px;
}

.toolbar {
  display: flex;
  justify-content: flex-end;
}

.box-card {
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-sm);
}

.ifp-picker {
  width: 100%;
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 8px;
}

.hidden-file-input {
  display: none;
}
</style>
