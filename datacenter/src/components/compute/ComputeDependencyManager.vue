<template>
  <DcDialog v-model="visible" :title="ui('工程依赖', 'Project Dependencies')" width="1080px" append-to-body>
    <section class="dependency-manager">
      <div class="dependency-manager__head">
        <p>{{ ui('工程内计算单元共用，代码中的导入语句会自动识别依赖。', 'Dependencies are shared by all compute units in the project and detected from import statements.') }}</p>
        <div class="dependency-manager__actions">
          <button type="button" :title="ui('刷新', 'Refresh')" :aria-label="ui('刷新依赖', 'Refresh dependencies')" @click="load">
            <IconTablerRefresh />
          </button>
          <button type="button" class="is-primary" @click="installDialogVisible = true">
            <IconTablerDownload />
            {{ ui('添加依赖', 'Add Dependency') }}
          </button>
        </div>
      </div>

      <div class="dependency-manager__filters">
        <el-input v-model="keyword" clearable :placeholder="ui('搜索包名或导入名称', 'Search package or import name')" />
        <el-segmented v-model="runtime" :options="runtimeOptions" />
        <span>{{ ui(`共 ${filteredDependencies.length} 项`, `${filteredDependencies.length} item${filteredDependencies.length === 1 ? '' : 's'}`) }}</span>
      </div>

      <div v-if="loading" class="dependency-manager__loading">
        <el-skeleton :rows="7" animated />
      </div>
      <EmptyState
        v-else-if="!filteredDependencies.length"
        icon-name="compute"
        :title="ui('暂无工程依赖', 'No Project Dependencies')"
        :description="ui('在线安装或导入离线包后，计算单元可直接引用。', 'Install online or import an offline package to use it in compute units.')"
      />
      <el-table v-else :data="filteredDependencies" height="460" class="dependency-manager__table">
        <el-table-column :label="ui('包名', 'Package')" min-width="190">
          <template #default="{ row }"
            ><strong>{{ row.name }}</strong></template
          >
        </el-table-column>
        <el-table-column :label="ui('导入名', 'Import Name')" min-width="170">
          <template #default="{ row }"
            ><code>{{ row.importName }}</code></template
          >
        </el-table-column>
        <el-table-column :label="ui('语言', 'Language')" width="120">
          <template #default="{ row }">{{ runtimeLabel(row.runtime) }}</template>
        </el-table-column>
        <el-table-column :label="ui('版本', 'Version')" width="150" prop="version" />
        <el-table-column :label="ui('引用', 'References')" width="120">
          <template #default="{ row }">{{ ui(`${row.referenceCount || 0} 个单元`, `${row.referenceCount || 0} unit${row.referenceCount === 1 ? '' : 's'}`) }}</template>
        </el-table-column>
        <el-table-column :label="ui('状态', 'Status')" width="110">
          <template #default><StatusBadge tone="success" :text="ui('已安装', 'Installed')" /></template>
        </el-table-column>
        <el-table-column :label="ui('操作', 'Actions')" width="90" align="center">
          <template #default="{ row }">
            <button
              type="button"
              class="dependency-manager__delete"
              :disabled="Boolean(row.referenceCount) || removingId === row.id"
              :title="row.referenceCount ? ui('仍有计算单元引用，不能卸载', 'Cannot uninstall while compute units reference it') : ui('卸载依赖', 'Uninstall dependency')"
              :aria-label="ui(`卸载 ${row.name}`, `Uninstall ${row.name}`)"
              @click="remove(row)"
            >
              <IconTablerTrash />
            </button>
          </template>
        </el-table-column>
      </el-table>
    </section>

    <template #footer>
      <el-button @click="visible = false">{{ ui('关闭', 'Close') }}</el-button>
    </template>
  </DcDialog>

  <DcDialog v-model="installDialogVisible" :title="ui('添加工程依赖', 'Add Project Dependency')" width="560px" append-to-body>
    <div class="dependency-install">
      <el-segmented v-model="installMode" :options="installModeOptions" />

      <form
        v-if="installMode === 'online'"
        class="dependency-install__form"
        @submit.prevent="install"
      >
        <label>
          <span>{{ ui('语言', 'Language') }}</span>
          <el-select v-model="onlineForm.language">
            <el-option label="JavaScript" value="js" />
            <el-option label="Python" value="python" />
          </el-select>
        </label>
        <label>
          <span>{{ ui('包名', 'Package') }}</span>
          <el-input
            v-model="onlineForm.packageName"
            :placeholder="onlineForm.language === 'js' ? ui('例如 dayjs', 'Example: dayjs') : ui('例如 humanize', 'Example: humanize')"
          />
        </label>
        <label>
          <span>{{ ui('版本', 'Version') }}</span>
          <el-input v-model="onlineForm.version" :placeholder="ui('可选，留空安装最新版本', 'Optional; leave blank for the latest version')" />
        </label>
      </form>

      <form v-else class="dependency-install__form" @submit.prevent="install">
        <label>
          <span>{{ ui('语言', 'Language') }}</span>
          <el-select v-model="offlineLanguage" @change="offlineFile = null">
            <el-option label="JavaScript" value="js" />
            <el-option label="Python" value="python" />
          </el-select>
        </label>
        <label class="dependency-install__file-row">
          <span>{{ ui('离线包', 'Offline Package') }}</span>
          <span class="dependency-install__file">
            <input
              :key="offlineLanguage"
              type="file"
              :accept="offlineLanguage === 'js' ? '.tgz' : '.whl'"
              :aria-label="ui('选择离线依赖文件', 'Select offline dependency file')"
              @change="selectOfflineFile"
            />
            <b>{{
              offlineFile?.name || (offlineLanguage === 'js' ? ui('选择 .tgz 文件', 'Select a .tgz file') : ui('选择 .whl 文件', 'Select a .whl file'))
            }}</b>
            <small>{{ ui('最大 64 MB；自动识别包信息，依赖的其他包需分别导入', 'Maximum 64 MB. Package metadata is detected automatically; import transitive packages separately.') }}</small>
          </span>
        </label>
      </form>
    </div>

    <template #footer>
      <el-button @click="installDialogVisible = false">{{ ui('取消', 'Cancel') }}</el-button>
      <el-button type="primary" :loading="installing" @click="install">
        {{ installMode === 'online' ? ui('在线安装', 'Install Online') : ui('导入并安装', 'Import and Install') }}
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
import { datacenterLocale } from '@/i18n/runtime'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)

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
const installModeOptions = computed(() => [
  { label: ui('在线安装', 'Online Install'), value: 'online' },
  { label: ui('离线导入', 'Offline Import'), value: 'offline' },
])
const runtimeOptions = computed(() => [
  { label: ui('全部', 'All'), value: 'all' },
  { label: 'JavaScript', value: 'javascript' },
  { label: 'Python', value: 'python' },
])
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
    ElMessage.error(getApiErrorMessage(error, ui('加载工程依赖失败', 'Failed to load project dependencies')))
  } finally {
    loading.value = false
  }
}

function selectOfflineFile(event: Event) {
  offlineFile.value = (event.target as HTMLInputElement).files?.[0] || null
}

async function install() {
  if (installMode.value === 'online' && !onlineForm.packageName.trim()) {
    ElMessage.warning(ui('请输入包名', 'Enter a package name'))
    return
  }
  if (installMode.value === 'offline' && !offlineFile.value) {
    ElMessage.warning(ui('请选择离线依赖文件', 'Select an offline dependency file'))
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
    ElMessage.success(installMode.value === 'online' ? ui('依赖已安装', 'Dependency installed') : ui('离线依赖已导入', 'Offline dependency imported'))
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, ui('添加依赖失败', 'Failed to add dependency')))
  } finally {
    installing.value = false
  }
}

async function remove(item: ComputeDependency) {
  await ElMessageBox.confirm(ui(`确定卸载 ${item.name}@${item.version || '-'}？`, `Uninstall ${item.name}@${item.version || '-'}?`), ui('卸载依赖', 'Uninstall Dependency'), {
    type: 'warning',
  })
  removingId.value = item.id
  try {
    await uninstallComputeDependency(props.projectId, item.id)
    await load()
    emit('changed')
    ElMessage.success(ui('依赖已卸载', 'Dependency uninstalled'))
  } catch (error) {
    if (error !== 'cancel') ElMessage.error(getApiErrorMessage(error, ui('卸载依赖失败', 'Failed to uninstall dependency')))
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
