<template>
  <div class="profile">
    <!-- 头部 -->
    <div class="header">
      <h1>连接配置管理</h1>
    </div>

    <el-card class="box-card">
      <el-form :model="profileForm" label-width="120px">
        <el-form-item label="项目ID" required>
          <el-input v-model="projectId" placeholder="输入项目ID" style="max-width: 300px;" />
          <el-button type="primary" @click="loadProfile" :loading="loading" style="margin-left: 10px;">
            加载配置
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card v-if="profileData" class="box-card">
      <template #header>
        <span>连接配置 - {{ projectId }}</span>
      </template>

      <el-descriptions :column="2" border>
        <el-descriptions-item label="名称">
          {{ profileData.name || '-' }}
        </el-descriptions-item>
        <el-descriptions-item label="端点">
          {{ profileData.endpoint || '-' }}
        </el-descriptions-item>
        <el-descriptions-item label="认证类型">
          <el-tag>{{ profileData.authType || '-' }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="元数据">
          <pre class="metadata">{{ JSON.stringify(profileData.metadata, null, 2) }}</pre>
        </el-descriptions-item>
      </el-descriptions>

      <div style="margin-top: 20px;">
        <el-button type="primary" @click="editMode = !editMode">
          {{ editMode ? '取消编辑' : '编辑配置' }}
        </el-button>
      </div>

      <el-form v-if="editMode" :model="editForm" label-width="120px" style="margin-top: 20px;">
        <el-form-item label="名称">
          <el-input v-model="editForm.name" />
        </el-form-item>
        <el-form-item label="端点">
          <el-input v-model="editForm.endpoint" />
        </el-form-item>
        <el-form-item label="认证类型">
          <el-select v-model="editForm.authType" placeholder="选择认证类型">
            <el-option label="Token" value="token" />
            <el-option label="Basic" value="basic" />
            <el-option label="OAuth" value="oauth" />
          </el-select>
        </el-form-item>
        <el-form-item label="认证数据">
          <el-input
            v-model="authDataStr"
            type="textarea"
            :rows="4"
            placeholder="JSON格式的认证数据"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="saveProfile" :loading="saving">
            保存
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-empty v-else-if="!loading" description="请输入项目ID并加载配置" />
  </div>
</template>

<script setup>
import { ref, reactive, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useNodeStore } from '@/store/nodeStore'
import { ElMessage } from 'element-plus'

const route = useRoute()
const nodeStore = useNodeStore()

const projectId = ref(route.query.projectId || '')
const loading = ref(false)
const saving = ref(false)
const editMode = ref(false)
const profileData = ref(null)

const editForm = reactive({
  name: '',
  endpoint: '',
  authType: '',
  metadata: {},
})

const authDataStr = ref('')

watch(editMode, (newVal) => {
  if (newVal && profileData.value) {
    editForm.name = profileData.value.name || ''
    editForm.endpoint = profileData.value.endpoint || ''
    editForm.authType = profileData.value.authType || ''
    editForm.metadata = profileData.value.metadata || {}
  }
})

const loadProfile = async () => {
  if (!projectId.value) {
    ElMessage.warning('请输入项目ID')
    return
  }

  loading.value = true
  try {
    const data = await nodeStore.getProfile(projectId.value)
    profileData.value = data
    ElMessage.success('配置加载成功')
  } catch (error) {
    console.error('加载配置失败:', error)
    ElMessage.error('加载配置失败')
    profileData.value = null
  } finally {
    loading.value = false
  }
}

const saveProfile = async () => {
  if (!projectId.value) {
    ElMessage.warning('项目ID不能为空')
    return
  }

  saving.value = true
  try {
    // 解析认证数据
    let authData = {}
    if (authDataStr.value) {
      try {
        authData = JSON.parse(authDataStr.value)
      } catch (e) {
        ElMessage.error('认证数据格式不正确，请输入有效的JSON')
        saving.value = false
        return
      }
    }

    const success = await nodeStore.saveProfile(projectId.value, {
      ...editForm,
      authData,
    })

    if (success) {
      editMode.value = false
      await loadProfile()
    }
  } catch (error) {
    console.error('保存配置失败:', error)
    ElMessage.error('保存配置失败')
  } finally {
    saving.value = false
  }
}

// 如果有项目ID参数，自动加载
if (projectId.value) {
  loadProfile()
}
</script>

<style scoped>
.profile {
  min-height: 100vh;
  padding: 20px;
  background-color: #f5f7fa;
}

.header {
  margin-bottom: 20px;
}

.header h1 {
  font-size: 24px;
  color: #303133;
}

.box-card {
  margin-bottom: 20px;
}

.metadata {
  background-color: #f5f7fa;
  padding: 10px;
  border-radius: 4px;
  font-size: 12px;
  max-height: 200px;
  overflow-y: auto;
}
</style>
