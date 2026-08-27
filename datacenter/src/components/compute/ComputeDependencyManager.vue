<template>
  <DcDialog v-model="visible" title="工程依赖" width="1080px" append-to-body>
    <section class="dependency-manager">
      <div class="dependency-manager__head">
        <p>工程内计算单元共用，代码中的导入语句会自动识别依赖。</p>
        <div class="dependency-manager__actions">
          <button type="button" title="刷新" aria-label="刷新依赖" @click="load">
            <IconTablerRefresh />
          </button>
          <button type="button" class="is-primary" @click="installDialogVisible = true">
            <IconTablerDownload />
            添加依赖
          </button>
        </div>
      </div>

      <div class="dependency-manager__filters">
        <el-input v-model="keyword" clearable placeholder="搜索包名或导入名称" />
        <el-segmented v-model="runtime" :options="runtimeOptions" />
        <span>共 {{ filteredDependencies.length }} 项</span>
      </div>

      <div v-if="loading" class="dependency-manager__loading">
        <el-skeleton :rows="7" animated />
      </div>
      <EmptyState
        v-else-if="!filteredDependencies.length"
        icon-name="compute"
        title="暂无工程依赖"
        description="在线安装或导入离线包后，计算单元可直接引用。"
      />
      <el-table v-else :data="filteredDependencies" height="460" class="dependency-manager__table">
        <el-table-column label="包名" min-width="190">
          <template #default="{ row }"
            ><strong>{{ row.name }}</strong></template
          >
        </el-table-column>
        <el-table-column label="导入名" min-width="170">
          <template #default="{ row }"
            ><code>{{ row.importName }}</code></template
          >
        </el-table-column>
        <el-table-column label="语言" width="120">
          <template #default="{ row }">{{ runtimeLabel(row.runtime) }}</template>
        </el-table-column>
        <el-table-column label="版本" width="150" prop="version" />
        <el-table-column label="引用" width="110">
          <template #default="{ row }">{{ row.referenceCount || 0 }} 个单元</template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default><StatusBadge tone="success" text="已安装" /></template>
        </el-table-column>
        <el-table-column label="操作" width="90" align="center">
          <template #default="{ row }">
            <button
              type="button"
              class="dependency-manager__delete"
              :disabled="Boolean(row.referenceCount) || removingId === row.id"
              :title="row.referenceCount ? '仍有计算单元引用，不能卸载' : '卸载依赖'"
              :aria-label="`卸载 ${row.name}`"
              @click="remove(row)"
            >
              <IconTablerTrash />
            </button>
          </template>
        </el-table-column>
      </el-table>
    </section>

    <template #footer>
      <el-button @click="visible = false">关闭</el-button>
    </template>
  </DcDialog>

  <DcDialog v-model="installDialogVisible" title="添加工程依赖" width="560px" append-to-body>
    <div class="dependency-install">
      <el-segmented v-model="installMode" :options="installModeOptions" />

      <form
        v-if="installMode === 'online'"
        class="dependency-install__form"
        @submit.prevent="install"
      >
        <label>
          <span>语言</span>
          <el-select v-model="onlineForm.language">
            <el-option label="JavaScript" value="js" />
            <el-option label="Python" value="python" />
          </el-select>
        </label>
        <label>
          <span>包名</span>
          <el-input
            v-model="onlineForm.packageName"
            :placeholder="onlineForm.language === 'js' ? '例如 dayjs' : '例如 humanize'"
          />
        </label>
        <label>
          <span>版本</span>
          <el-input v-model="onlineForm.version" placeholder="可选，留空安装最新版本" />
        </label>
      </form>

      <form v-else class="dependency-install__form" @submit.prevent="install">
        <label>
          <span>语言</span>
          <el-select v-model="offlineLanguage" @change="offlineFile = null">
            <el-option label="JavaScript" value="js" />
            <el-option label="Python" value="python" />
          </el-select>
        </label>
        <label class="dependency-install__file-row">
          <span>离线包</span>
          <span class="dependency-install__file">
            <input
              :key="offlineLanguage"
              type="file"
              :accept="offlineLanguage === 'js' ? '.tgz' : '.whl'"
              aria-label="选择离线依赖文件"
              @change="selectOfflineFile"
            />
            <b>{{
              offlineFile?.name || (offlineLanguage === 'js' ? '选择 .tgz 文件' : '选择 .whl 文件')
            }}</b>
            <small>最大 64 MB；自动识别包信息，依赖的其他包需分别导入</small>
          </span>
        </label>
      </form>
    </div>

    <template #footer>
      <el-button @click="installDialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="installing" @click="install">
        {{ installMode === 'online' ? '在线安装' : '导入并安装' }}
      </el-button>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import IconTablerDownload from '~icons/tabler/download'
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerTrash from '~icons/tabler/trash'
import type { ComputeDependency } from '@/api/schemas/compute.schema'
import {
  getComputeDependencies,
  importComputeDependency,
  installComputeDependency,
  uninstallComputeDependency,
} from '@/api/compute.api'
import { getApiErrorMessage } from '@/utils/request'
import DcDialog from '@/components/shared/DcDialog.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import StatusBadge from '@/components/shared/StatusBadge.vue'

const props = defineProps<{ projectId: string }>()
const visible = defineModel<boolean>({ required: true })
const emit = defineEmits<{ (event: 'changed'): void }>()

const dependencies = ref<ComputeDependency[]>([])
const loading = ref(false)
const installing = ref(false)
const removingId = ref('')
const installDialogVisible = ref(false)
const installMode = ref<'online' | 'offline'>('online')
const offlineLanguage = ref<'js' | 'python'>('js')
const offlineFile = ref<File | null>(null)
const keyword = ref('')
const runtime = ref('all')
const installModeOptions = [
  { label: '在线安装', value: 'online' },
  { label: '离线导入', value: 'offline' },
]
const runtimeOptions = [
  { label: '全部', value: 'all' },
  { label: 'JavaScript', value: 'javascript' },
  { label: 'Python', value: 'python' },
]
const onlineForm = reactive({
  language: 'js' as 'js' | 'python',
  packageName: '',
  version: '',
})

const filteredDependencies = computed(() => {
  const search = keyword.value.trim().toLowerCase()
  return dependencies.value.filter((item) => {
    if (runtime.value !== 'all' && item.runtime !== runtime.value) return false
    return !search || `${item.name} ${item.importName || ''}`.toLowerCase().includes(search)
  })
})

function runtimeLabel(value: string) {
  return value === 'python' ? 'Python' : 'JavaScript'
}

async function load() {
  loading.value = true
  try {
    dependencies.value = await getComputeDependencies(props.projectId)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载工程依赖失败'))
  } finally {
    loading.value = false
  }
}

function selectOfflineFile(event: Event) {
  offlineFile.value = (event.target as HTMLInputElement).files?.[0] || null
}

async function install() {
  if (installMode.value === 'online' && !onlineForm.packageName.trim()) {
    ElMessage.warning('请输入包名')
    return
  }
  if (installMode.value === 'offline' && !offlineFile.value) {
    ElMessage.warning('请选择离线依赖文件')
    return
  }
  installing.value = true
  try {
    if (installMode.value === 'online') {
      await installComputeDependency(props.projectId, {
        language: onlineForm.language,
        packageName: onlineForm.packageName.trim(),
        version: onlineForm.version.trim() || undefined,
      })
      Object.assign(onlineForm, { packageName: '', version: '' })
    } else {
      await importComputeDependency(props.projectId, offlineLanguage.value, offlineFile.value!)
      offlineFile.value = null
    }
    installDialogVisible.value = false
    await load()
    emit('changed')
    ElMessage.success(installMode.value === 'online' ? '依赖已安装' : '离线依赖已导入')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '添加依赖失败'))
  } finally {
    installing.value = false
  }
}

async function remove(item: ComputeDependency) {
  await ElMessageBox.confirm(`确定卸载 ${item.name}@${item.version || '-'}？`, '卸载依赖', {
    type: 'warning',
  })
  removingId.value = item.id
  try {
    await uninstallComputeDependency(props.projectId, item.id)
    await load()
    emit('changed')
    ElMessage.success('依赖已卸载')
  } catch (error) {
    if (error !== 'cancel') ElMessage.error(getApiErrorMessage(error, '卸载依赖失败'))
  } finally {
    removingId.value = ''
  }
}

watch(visible, (value) => {
  if (value) void load()
})
</script>

<style scoped>
.dependency-manager {
  min-width: 0;
  min-height: 580px;
  display: grid;
  grid-template-rows: auto auto minmax(0, 1fr);
}
.dependency-manager__head,
.dependency-manager__actions,
.dependency-manager__filters {
  display: flex;
  align-items: center;
}
.dependency-manager__head {
  justify-content: space-between;
  gap: 16px;
  padding-bottom: 14px;
}
.dependency-manager__head p {
  margin: 0;
  color: var(--dc-text-muted);
  font-size: 12px;
}
.dependency-manager__actions {
  gap: 8px;
}
.dependency-manager__actions button {
  min-height: 34px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 0 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}
.dependency-manager__actions button.is-primary {
  border-color: var(--dc-primary);
  background: var(--dc-primary);
  color: white;
}
.dependency-manager__actions svg,
.dependency-manager__delete svg {
  width: 16px;
  height: 16px;
}
.dependency-manager__filters {
  gap: 10px;
  padding-bottom: 14px;
  border-bottom: 1px solid var(--dc-border);
}
.dependency-manager__filters .el-input {
  width: 280px;
}
.dependency-manager__filters > span {
  margin-left: auto;
  color: var(--dc-text-muted);
  font-size: 12px;
}
.dependency-manager__loading {
  padding: 20px;
}
.dependency-manager__table {
  padding-top: 8px;
}
.dependency-manager__table strong {
  color: var(--dc-text);
}
.dependency-manager__table code {
  color: var(--dc-primary);
}
.dependency-manager__delete {
  width: 30px;
  height: 30px;
  display: inline-grid;
  place-items: center;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--dc-danger);
}
.dependency-manager__delete:disabled {
  color: var(--dc-text-muted);
  opacity: 0.5;
}
.dependency-install {
  display: grid;
  gap: 18px;
}
.dependency-install > .el-segmented {
  justify-self: stretch;
}
.dependency-install__form {
  display: grid;
  gap: 14px;
}
.dependency-install__form label {
  display: grid;
  grid-template-columns: 80px minmax(0, 1fr);
  align-items: center;
  gap: 12px;
}
.dependency-install__form label > span:first-child {
  color: var(--dc-text-secondary);
  font-size: 13px;
}
.dependency-install__file {
  position: relative;
  min-height: 92px;
  display: grid;
  align-content: center;
  justify-items: center;
  gap: 6px;
  border: 1px dashed var(--dc-border-strong);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-text-secondary);
  overflow: hidden;
}
.dependency-install__file input {
  position: absolute;
  inset: 0;
  opacity: 0;
  cursor: pointer;
}
.dependency-install__file b {
  max-width: 360px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
}
.dependency-install__file small {
  color: var(--dc-text-muted);
  font-size: 11px;
}
</style>
