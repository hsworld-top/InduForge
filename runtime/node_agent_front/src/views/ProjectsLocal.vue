<template>
  <div class="projects">
    <div class="toolbar">
      <el-button type="primary" @click="showDeployDialog = true">
        <el-icon><Plus /></el-icon>
        部署本地工程
      </el-button>
    </div>

    <el-card class="box-card panel-card">
      <template #header>
        <span>本地工程列表</span>
      </template>
      <el-table v-loading="nodeStore.loading" :data="localProjects" style="width: 100%">
        <el-table-column prop="id" label="项目ID" min-width="170" />
        <el-table-column prop="currentVersion" label="当前版本" width="120" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="运行时状态" width="140">
          <template #default="{ row }">{{ row.runtimeStatus?.state || 'stopped' }}</template>
        </el-table-column>
        <el-table-column label="PID" width="100">
          <template #default="{ row }">{{ row.runtimeStatus?.pid || '-' }}</template>
        </el-table-column>
        <el-table-column label="最后启动" width="180">
          <template #default="{ row }">{{ formatTime(row.lastStartedAt) }}</template>
        </el-table-column>
        <el-table-column label="操作" min-width="320" fixed="right">
          <template #default="{ row }">
            <el-button v-if="row.status !== 'running'" type="primary" size="small" @click="startProject(row.id)">
              启动
            </el-button>
            <el-button v-else type="warning" size="small" @click="stopProject(row.id)">停止</el-button>
            <el-button type="info" size="small" @click="restartProject(row.id)">重启</el-button>
            <el-button type="success" size="small" @click="viewLogs(row.id)">日志</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-if="!localProjects.length && !nodeStore.loading" description="暂无本地工程" />
    </el-card>

    <el-dialog v-model="showDeployDialog" title="部署本地工程" width="600px">
      <el-form :model="deployForm" label-width="120px">
        <el-form-item label="IFP文件" required>
          <div class="ifp-picker">
            <el-input :model-value="deployForm.ifpPackage || '未选择文件'" readonly />
            <el-button @click="triggerFilePick">选择文件</el-button>
          </div>
          <input
            ref="ifpFileInput"
            class="hidden-file-input"
            type="file"
            accept=".ifp"
            @change="handleIfpFileChange"
          />
        </el-form-item>
        <el-form-item label="自动启动">
          <el-switch v-model="deployForm.autoStart" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showDeployDialog = false">取消</el-button>
        <el-button type="primary" :loading="deploying" @click="deployProject">部署</el-button>
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

const router = useRouter()
const nodeStore = useNodeStore()

const showDeployDialog = ref(false)
const deploying = ref(false)
const ifpFileInput = ref(null)
const selectedIfpFile = ref(null)

const deployForm = reactive({
  ifpPackage: '',
  autoStart: true,
})

const localProjects = computed(() => nodeStore.projects.filter((project) => project?.source !== 'center'))

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
    ElMessage.warning('仅支持选择 .ifp 文件')
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
    ElMessage.warning('请选择 IFP 文件')
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

const formatTime = (time) => (time ? new Date(time).toLocaleString() : '-')

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
