<template>
  <div class="projects">
    <!-- 头部 -->
    <div class="header">
      <h1>项目管理</h1>
      <el-button type="primary" @click="showDeployDialog = true">
        <el-icon><Plus /></el-icon>
        部署项目
      </el-button>
    </div>

    <!-- 项目列表 -->
    <el-card class="box-card">
      <el-table
        v-loading="nodeStore.loading"
        :data="nodeStore.projects"
        style="width: 100%"
      >
        <el-table-column prop="id" label="项目ID" />
        <el-table-column prop="currentVersion" label="当前版本" width="120" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="运行时状态" width="120">
          <template #default="{ row }">
            {{ row.runtimeStatus?.state || 'stopped' }}
          </template>
        </el-table-column>
        <el-table-column label="PID" width="100">
          <template #default="{ row }">
            {{ row.runtimeStatus?.pid || '-' }}
          </template>
        </el-table-column>
        <el-table-column label="最后启动" width="180">
          <template #default="{ row }">
            {{ formatTime(row.lastStartedAt) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="300" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="row.status !== 'running'"
              type="primary"
              size="small"
              @click="startProject(row.id)"
            >
              启动
            </el-button>
            <el-button
              v-else
              type="warning"
              size="small"
              @click="stopProject(row.id)"
            >
              停止
            </el-button>
            <el-button
              type="info"
              size="small"
              @click="restartProject(row.id)"
            >
              重启
            </el-button>
            <el-button
              type="success"
              size="small"
              @click="viewLogs(row.id)"
            >
              日志
            </el-button>
            <el-dropdown>
              <el-button size="small">
                更多<el-icon class="el-icon--right"><ArrowDown /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item @click="openRollbackDialog(row)">
                    回滚版本
                  </el-dropdown-item>
                  <el-dropdown-item @click="openProfileDialog(row)">
                    连接配置
                  </el-dropdown-item>
                  <el-dropdown-item divided @click="deleteProject(row)">
                    删除项目
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 部署对话框 -->
    <el-dialog
      v-model="showDeployDialog"
      title="部署项目"
      width="600px"
    >
      <el-form :model="deployForm" label-width="120px">
        <el-form-item label="项目ID" required>
          <el-input v-model="deployForm.projectId" placeholder="输入项目ID" />
        </el-form-item>
        <el-form-item label="版本" required>
          <el-input v-model="deployForm.version" placeholder="输入版本号" />
        </el-form-item>
        <el-form-item label="IFP包路径" required>
          <el-input v-model="deployForm.ifpPackage" placeholder="输入IFP包路径" />
        </el-form-item>
        <el-form-item label="自动启动">
          <el-switch v-model="deployForm.autoStart" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showDeployDialog = false">取消</el-button>
        <el-button type="primary" @click="deployProject" :loading="deploying">
          部署
        </el-button>
      </template>
    </el-dialog>

    <!-- 回滚对话框 -->
    <el-dialog
      v-model="showRollbackDialog"
      title="回滚版本"
      width="400px"
    >
      <el-form>
        <el-form-item label="选择版本">
          <el-select v-model="rollbackVersion" placeholder="选择要回滚的版本">
            <el-option
              v-for="version in availableVersions"
              :key="version"
              :label="version"
              :value="version"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showRollbackDialog = false">取消</el-button>
        <el-button
          type="primary"
          @click="confirmRollback"
          :disabled="!rollbackVersion"
        >
          确认回滚
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { useNodeStore } from '@/store/nodeStore'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, ArrowDown } from '@element-plus/icons-vue'

const router = useRouter()
const nodeStore = useNodeStore()

const showDeployDialog = ref(false)
const showRollbackDialog = ref(false)
const showProfileDialog = ref(false)
const deploying = ref(false)
const rollbackVersion = ref('')
const availableVersions = ref([])
const currentProject = ref(null)

const deployForm = reactive({
  projectId: '',
  version: '',
  ifpPackage: '',
  autoStart: true,
})

const deployProject = async () => {
  if (!deployForm.projectId || !deployForm.version) {
    ElMessage.warning('请填写项目ID和版本')
    return
  }

  deploying.value = true
  const success = await nodeStore.deployProject(deployForm.projectId, {
    version: deployForm.version,
    ifpPackage: deployForm.ifpPackage,
    autoStart: deployForm.autoStart,
  })

  if (success) {
    showDeployDialog.value = false
    Object.assign(deployForm, {
      projectId: '',
      version: '',
      ifpPackage: '',
      autoStart: true,
    })
  }

  deploying.value = false
}

const startProject = async (projectId) => {
  await nodeStore.startProject(projectId)
}

const stopProject = async (projectId) => {
  await nodeStore.stopProject(projectId)
}

const restartProject = async (projectId) => {
  await nodeStore.restartProject(projectId)
}

const viewLogs = (projectId) => {
  router.push(`/logs/${projectId}`)
}

const openRollbackDialog = (project) => {
  currentProject.value = project
  availableVersions.value = project.versions || []
  showRollbackDialog.value = true
}

const confirmRollback = async () => {
  if (!rollbackVersion.value) return

  await nodeStore.rollbackProject(currentProject.value.id, rollbackVersion.value)
  showRollbackDialog.value = false
  rollbackVersion.value = ''
}

const openProfileDialog = (project) => {
  router.push('/profile?projectId=' + project.id)
}

const deleteProject = async (project) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除项目 ${project.id} 吗？`,
      '警告',
      {
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )
    ElMessage.success('项目已删除')
  } catch {
    // 用户取消
  }
}

const getStatusType = (status) => {
  const map = {
    running: 'success',
    stopped: 'info',
    error: 'danger',
    deploying: 'warning',
  }
  return map[status] || 'info'
}

const formatTime = (time) => {
  if (!time) return '-'
  return new Date(time).toLocaleString()
}
</script>

<style scoped>
.projects {
  min-height: 100vh;
  padding: 20px;
  background-color: #f5f7fa;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.header h1 {
  font-size: 24px;
  color: #303133;
}

.box-card {
  background-color: #fff;
}
</style>
