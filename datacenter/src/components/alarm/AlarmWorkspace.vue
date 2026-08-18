<template>
  <div class="alarm-workspace">
    <section class="alarm-workspace__panel">
      <header
        class="alarm-workspace__head"
        :class="{ 'is-notifications': activeTab === 'notifications' }"
      >
        <el-segmented v-model="activeTab" :options="tabOptions" size="small" />

        <div v-if="activeTab === 'policies'" class="alarm-workspace__toolbar">
          <el-input
            v-model="search"
            class="alarm-workspace__search"
            size="small"
            clearable
            placeholder="搜索报警名称或数据点"
            :prefix-icon="Search"
          />
          <PillButton :active="directoryVisible" @click="toggleDirectory"
            ><template #icon><IconTablerFolders /></template>目录</PillButton
          >
          <el-popover trigger="click" placement="bottom-start" :width="150"
            ><template #reference
              ><PillButton :active="modeFilter !== ''">{{ modeLabel }}</PillButton></template
            >
            <div class="alarm-workspace__filter-menu">
              <button
                v-for="item in modeOptions"
                :key="item.value"
                type="button"
                :class="{ 'is-active': modeFilter === item.value }"
                @click="modeFilter = item.value"
              >
                {{ item.label }}
              </button>
            </div></el-popover
          >
          <el-popover trigger="click" placement="bottom-start" :width="150"
            ><template #reference
              ><PillButton :active="severityFilter !== ''">{{
                severityLabel
              }}</PillButton></template
            >
            <div class="alarm-workspace__filter-menu">
              <button
                v-for="item in severityOptions"
                :key="item.value"
                type="button"
                :class="{ 'is-active': severityFilter === item.value }"
                @click="severityFilter = item.value"
              >
                {{ item.label }}
              </button>
            </div></el-popover
          >
          <el-popover trigger="click" placement="bottom-start" :width="140"
            ><template #reference
              ><PillButton :active="enabledFilter !== ''">{{ enabledLabel }}</PillButton></template
            >
            <div class="alarm-workspace__filter-menu">
              <button
                v-for="item in enabledOptions"
                :key="item.value"
                type="button"
                :class="{ 'is-active': enabledFilter === item.value }"
                @click="enabledFilter = item.value"
              >
                {{ item.label }}
              </button>
            </div></el-popover
          >
          <button
            type="button"
            class="alarm-workspace__icon"
            title="刷新报警配置"
            @click="loadPolicies"
          >
            <IconTablerRefresh />
          </button>
          <span class="alarm-workspace__total">共 {{ pagination.total }}</span>
          <div class="alarm-workspace__create-actions">
            <button type="button" class="alarm-workspace__secondary" @click="openCreate('derived')">
              <IconTablerFunction />组合报警
            </button>
            <button
              type="button"
              class="alarm-workspace__primary"
              @click="openCreate('per_target')"
            >
              <IconTablerPlus />新建报警
            </button>
          </div>
        </div>
      </header>

      <section class="alarm-workspace__content-panel">
        <template v-if="activeTab === 'policies'">
          <AlarmDirectoryPanel
            v-if="directoryVisible"
            :project-id="projectId"
            :groups="groups"
            :selected-id="groupFilter"
            @select="selectGroup"
            @changed="loadGroups"
          />

          <div class="alarm-workspace__table-panel">
            <div v-loading="loading" class="alarm-workspace__table-wrap">
              <el-table
                :data="policies"
                row-key="id"
                height="100%"
                class="alarm-workspace__table"
                @row-dblclick="openEdit"
              >
                <el-table-column label="名称" width="142">
                  <template #default="{ row }"
                    ><div class="alarm-workspace__name">
                      <strong>{{ row.name }}</strong
                      ><small>{{ row.mode === 'derived' ? '组合报警' : '普通报警' }}</small>
                    </div></template
                  >
                </el-table-column>
                <el-table-column
                  label="数据点"
                  width="110"
                  class-name="alarm-workspace__compact-hide"
                  label-class-name="alarm-workspace__compact-hide"
                  ><template #default="{ row }"
                    ><span
                      class="alarm-workspace__ellipsis"
                      :title="alarmPolicyPointSummary(row)"
                      >{{ alarmPolicyPointSummary(row) }}</span
                    ></template
                  ></el-table-column
                >
                <el-table-column label="报警条件" width="90"
                  ><template #default="{ row }">{{
                    alarmPolicyConditionSummary(row)
                  }}</template></el-table-column
                >
                <el-table-column label="最高等级" width="78"
                  ><template #default="{ row }"
                    ><StatusBadge
                      :tone="severityTone(alarmPolicyHighestSeverity(row))"
                      :text="alarmSeverityLabels[alarmPolicyHighestSeverity(row)]" /></template
                ></el-table-column>
                <el-table-column label="状态" width="62"
                  ><template #default="{ row }"
                    ><el-switch
                      :model-value="row.isEnabled"
                      :loading="togglingId === row.id"
                      @change="togglePolicy(row, Boolean($event))" /></template
                ></el-table-column>
                <el-table-column
                  label="目录"
                  width="80"
                  class-name="alarm-workspace__compact-hide"
                  label-class-name="alarm-workspace__compact-hide"
                  ><template #default="{ row }"
                    ><span class="alarm-workspace__ellipsis">{{
                      row.groupName || '根目录'
                    }}</span></template
                  ></el-table-column
                >
                <el-table-column label="操作" width="70" align="right" fixed="right"
                  ><template #default="{ row }"
                    ><button
                      type="button"
                      class="alarm-workspace__row-action"
                      title="编辑报警"
                      @click="openEdit(row)"
                    >
                      <IconTablerEdit /></button
                    ><button
                      type="button"
                      class="alarm-workspace__row-action is-danger"
                      title="删除报警"
                      @click="removePolicy(row)"
                    >
                      <IconTablerTrash /></button></template
                ></el-table-column>
                <template #empty>
                  <el-empty :image-size="54" description="暂无报警配置" />
                </template>
              </el-table>
            </div>
            <DataCenterPagination
              :page="pagination.page"
              :page-size="pagination.pageSize"
              :total="pagination.total"
              :total-pages="pagination.totalPages"
              @change="changePage"
            />
          </div>
        </template>

        <AlarmNotificationSettings v-else :project-id="projectId" @changed="channels = $event" />
      </section>
    </section>

    <AlarmPolicyDrawer
      v-model="drawerVisible"
      :project-id="projectId"
      :mode="creatingMode"
      :policy="editingPolicy"
      :groups="groups"
      :channels="channels"
      :saving="saving"
      @save="savePolicy"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Search } from '@element-plus/icons-vue'
import IconTablerEdit from '~icons/tabler/edit'
import IconTablerFolders from '~icons/tabler/folders'
import IconTablerFunction from '~icons/tabler/function'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerTrash from '~icons/tabler/trash'
import AlarmDirectoryPanel from './AlarmDirectoryPanel.vue'
import AlarmNotificationSettings from './AlarmNotificationSettings.vue'
import AlarmPolicyDrawer from './AlarmPolicyDrawer.vue'
import DataCenterPagination from '@/components/shared/DataCenterPagination.vue'
import PillButton from '@/components/shared/PillButton.vue'
import StatusBadge from '@/components/shared/StatusBadge.vue'
import {
  createAlarmPolicy,
  deleteAlarmPolicy,
  listAlarmChannels,
  listAlarmGroupTree,
  listAlarmPolicies,
  setAlarmPolicyEnabled,
  updateAlarmPolicy,
} from '@/api/alarm.api'
import type {
  AlarmNotificationChannel,
  AlarmPolicy,
  AlarmPolicyGroup,
  AlarmPolicyMode,
  AlarmPolicySave,
  AlarmSeverity,
} from '@/api/schemas/alarm.schema'
import {
  alarmPolicyConditionSummary,
  alarmPolicyHighestSeverity,
  alarmPolicyPointSummary,
  alarmSeverityLabels,
  ensureBuiltinAlarmNotificationChannel,
} from '@/models/alarm-policy'
import { getApiErrorMessage } from '@/utils/request'
import { useConfirm } from '@/composables/useConfirm'

const props = defineProps<{ projectId: string }>()
const { confirm } = useConfirm()
const activeTab = ref<'policies' | 'notifications'>('policies')
const tabOptions = [
  { label: '报警配置', value: 'policies' },
  { label: '通知设置', value: 'notifications' },
]
const policies = ref<AlarmPolicy[]>([])
const groups = ref<AlarmPolicyGroup[]>([])
const channels = ref<AlarmNotificationChannel[]>([])
const loading = ref(false)
const saving = ref(false)
const togglingId = ref('')
const search = ref('')
const modeFilter = ref<AlarmPolicyMode | ''>('')
const severityFilter = ref<AlarmSeverity | ''>('')
const enabledFilter = ref<'true' | 'false' | ''>('')
const groupFilter = ref('')
const directoryVisible = ref(localStorage.getItem('datacenter.alarm.directory-visible') === 'true')
const drawerVisible = ref(false)
const creatingMode = ref<AlarmPolicyMode>('per_target')
const editingPolicy = ref<AlarmPolicy | null>(null)
const pagination = reactive({ page: 1, pageSize: 20, total: 0, totalPages: 0 })
let searchTimer: ReturnType<typeof setTimeout> | null = null

const modeOptions = [
  { label: '全部类型', value: '' as const },
  { label: '普通报警', value: 'per_target' as const },
  { label: '组合报警', value: 'derived' as const },
]
const severityOptions = [
  { label: '全部等级', value: '' as const },
  ...(['info', 'warning', 'major', 'critical'] as AlarmSeverity[]).map((value) => ({
    label: alarmSeverityLabels[value],
    value,
  })),
]
const enabledOptions = [
  { label: '全部状态', value: '' as const },
  { label: '已启用', value: 'true' as const },
  { label: '已停用', value: 'false' as const },
]
const modeLabel = computed(
  () => modeOptions.find((item) => item.value === modeFilter.value)?.label || '类型',
)
const severityLabel = computed(
  () => severityOptions.find((item) => item.value === severityFilter.value)?.label || '等级',
)
const enabledLabel = computed(
  () => enabledOptions.find((item) => item.value === enabledFilter.value)?.label || '状态',
)

onMounted(async () => {
  await Promise.all([loadGroups(), loadChannels(), loadPolicies()])
})

async function loadPolicies() {
  loading.value = true
  try {
    const result = await listAlarmPolicies(props.projectId, {
      page: pagination.page,
      pageSize: pagination.pageSize,
      search: search.value.trim(),
      mode: modeFilter.value,
      severity: severityFilter.value,
      enabled: enabledFilter.value,
      groupId: groupFilter.value,
    })
    policies.value = result.list
    pagination.total = result.pagination.total || 0
    pagination.page = result.pagination.page || pagination.page
    pagination.pageSize = result.pagination.pageSize || pagination.pageSize
    pagination.totalPages = Math.ceil(pagination.total / pagination.pageSize)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载报警配置失败'))
  } finally {
    loading.value = false
  }
}

async function loadGroups() {
  try {
    groups.value = await listAlarmGroupTree(props.projectId)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载报警目录失败'))
  }
}
async function loadChannels() {
  try {
    channels.value = ensureBuiltinAlarmNotificationChannel(
      props.projectId,
      await listAlarmChannels(props.projectId),
    )
  } catch {
    channels.value = ensureBuiltinAlarmNotificationChannel(props.projectId, [])
  }
}
function toggleDirectory() {
  directoryVisible.value = !directoryVisible.value
  localStorage.setItem('datacenter.alarm.directory-visible', String(directoryVisible.value))
}
function selectGroup(id: string) {
  groupFilter.value = id
  pagination.page = 1
  void loadPolicies()
}
function openCreate(mode: AlarmPolicyMode) {
  creatingMode.value = mode
  editingPolicy.value = null
  drawerVisible.value = true
}
function openEdit(policy: AlarmPolicy) {
  creatingMode.value = policy.mode
  editingPolicy.value = policy
  drawerVisible.value = true
}

async function savePolicy(payload: AlarmPolicySave) {
  saving.value = true
  try {
    if (editingPolicy.value)
      await updateAlarmPolicy(props.projectId, editingPolicy.value.id, payload)
    else await createAlarmPolicy(props.projectId, payload)
    drawerVisible.value = false
    ElMessage.success('报警配置已保存')
    await loadPolicies()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '保存报警配置失败'))
  } finally {
    saving.value = false
  }
}

async function togglePolicy(policy: AlarmPolicy, enabled: boolean) {
  togglingId.value = policy.id
  try {
    const saved = await setAlarmPolicyEnabled(props.projectId, policy.id, enabled)
    Object.assign(policy, saved)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '更新报警状态失败'))
  } finally {
    togglingId.value = ''
  }
}

async function removePolicy(policy: AlarmPolicy) {
  if (
    !(await confirm(`确认删除报警「${policy.name}」？`, {
      title: '删除报警',
      confirmText: '删除',
      type: 'warning',
    }))
  )
    return
  try {
    await deleteAlarmPolicy(props.projectId, policy.id)
    ElMessage.success('报警配置已删除')
    await loadPolicies()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '删除报警配置失败'))
  }
}

function changePage(value: { page: number; pageSize: number }) {
  pagination.page = value.page
  pagination.pageSize = value.pageSize
  void loadPolicies()
}
function severityTone(value: AlarmSeverity): 'muted' | 'warning' | 'danger' | 'success' {
  return value === 'critical'
    ? 'danger'
    : value === 'major' || value === 'warning'
      ? 'warning'
      : 'muted'
}

watch([modeFilter, severityFilter, enabledFilter], () => {
  pagination.page = 1
  void loadPolicies()
})
watch(search, () => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    pagination.page = 1
    void loadPolicies()
  }, 300)
})
</script>

<style scoped>
.alarm-workspace {
  height: 100%;
  min-height: 0;
  display: flex;
  color: var(--dc-text);
}
.alarm-workspace__panel {
  min-width: 0;
  min-height: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.alarm-workspace__head {
  min-height: 56px;
  flex: 0 0 56px;
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 0 20px;
  border: 1px solid var(--dc-border);
  border-radius: 14px;
  background: var(--dc-surface-raised);
  box-shadow:
    0 8px 24px rgba(15, 23, 42, 0.06),
    0 1px 2px rgba(15, 23, 42, 0.04);
}
.alarm-workspace__head :deep(.el-segmented) {
  flex: 0 0 auto;
}
.alarm-workspace__toolbar {
  min-width: 0;
  flex: 1;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.alarm-workspace__search {
  width: 230px;
}
.alarm-workspace__create-actions {
  display: flex;
  gap: 8px;
}
.alarm-workspace__primary,
.alarm-workspace__secondary,
.alarm-workspace__icon,
.alarm-workspace__row-action {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border-radius: var(--dc-radius-sm);
  cursor: pointer;
  font-family: inherit;
}
.alarm-workspace__primary,
.alarm-workspace__secondary {
  height: 32px;
  padding: 0 11px;
}
.alarm-workspace__primary {
  border: 1px solid var(--dc-primary);
  background: var(--dc-primary);
  color: white;
}
.alarm-workspace__secondary {
  border: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}
.alarm-workspace__secondary:hover,
.alarm-workspace__icon:hover {
  border-color: color-mix(in oklch, var(--dc-primary) 24%, var(--dc-border));
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}
.alarm-workspace__primary:hover {
  background: color-mix(in srgb, var(--dc-primary) 88%, black);
}
.alarm-workspace__primary svg,
.alarm-workspace__secondary svg {
  width: 15px;
}
.alarm-workspace__icon {
  width: 32px;
  height: 32px;
  border: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text-muted);
}
.alarm-workspace__primary:focus-visible,
.alarm-workspace__secondary:focus-visible,
.alarm-workspace__icon:focus-visible,
.alarm-workspace__row-action:focus-visible {
  outline: none;
  border-color: var(--dc-primary);
  box-shadow: 0 0 0 3px rgba(29, 78, 216, 0.12);
}
.alarm-workspace__total {
  margin-left: auto;
  color: var(--dc-text-muted);
  font-size: 12px;
  white-space: nowrap;
}
.alarm-workspace__content-panel {
  min-width: 0;
  min-height: 0;
  flex: 1;
  display: flex;
  overflow: hidden;
  border: 1px solid var(--dc-border);
  border-radius: 14px;
  background: var(--dc-surface-raised);
  box-shadow:
    0 8px 24px rgba(15, 23, 42, 0.055),
    0 1px 2px rgba(15, 23, 42, 0.04);
}
.alarm-workspace__table-panel {
  min-width: 0;
  min-height: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.alarm-workspace__table-wrap {
  min-height: 0;
  flex: 1;
  overflow: hidden;
  padding: 12px 20px 0;
}
.alarm-workspace__table {
  width: 100%;
  height: 100%;
}
.alarm-workspace__table :deep(.el-table__inner-wrapper::before) {
  display: none;
}
.alarm-workspace__table :deep(.el-table__header th) {
  height: 50px;
  background: var(--dc-surface-raised);
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0;
}
.alarm-workspace__table :deep(.el-table__cell) {
  padding: 8px 0;
  border-color: var(--dc-border);
  color: var(--dc-text-secondary);
  font-size: 13px;
}
.alarm-workspace__table :deep(.el-table__row) {
  height: 60px;
}
.alarm-workspace__table :deep(.el-table__row:hover > td.el-table__cell) {
  background: var(--dc-surface-muted);
}
.alarm-workspace__name {
  min-width: 0;
  display: grid;
  gap: 5px;
}
.alarm-workspace__name strong {
  overflow: hidden;
  font-size: 13px;
  letter-spacing: 0;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.alarm-workspace__name small {
  color: var(--dc-text-muted);
  font-size: 11px;
}
.alarm-workspace__ellipsis {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.alarm-workspace__row-action {
  width: 30px;
  height: 30px;
  border: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}
.alarm-workspace__row-action:hover {
  border-color: color-mix(in oklch, var(--dc-primary) 24%, var(--dc-border));
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}
.alarm-workspace__row-action.is-danger {
  margin-left: 4px;
}
.alarm-workspace__row-action.is-danger:hover {
  border-color: color-mix(in oklch, var(--el-color-danger) 28%, var(--dc-border));
  background: color-mix(in srgb, var(--el-color-danger) 8%, white);
  color: var(--el-color-danger);
}
.alarm-workspace__row-action svg {
  width: 15px;
}
.alarm-workspace__filter-menu {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin: -6px -8px;
}
.alarm-workspace__filter-menu button {
  width: 100%;
  padding: 7px 12px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--dc-text-secondary);
  font-size: 13px;
  text-align: left;
  cursor: pointer;
}
.alarm-workspace__filter-menu button:hover {
  background: var(--dc-surface-muted);
  color: var(--dc-text);
}
.alarm-workspace__filter-menu button.is-active {
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
  font-weight: 700;
}
@media (max-width: 1050px) {
  .alarm-workspace__head {
    min-height: 112px;
    align-items: flex-start;
    flex-direction: column;
    justify-content: center;
    padding: 10px 14px;
  }
  .alarm-workspace__toolbar {
    width: 100%;
    flex: 0 0 auto;
  }
  .alarm-workspace__head.is-notifications {
    min-height: 56px;
    align-items: center;
    flex-direction: row;
    justify-content: flex-start;
  }
}
@media (max-width: 760px) {
  .alarm-workspace__head {
    min-height: 154px;
    align-items: stretch;
  }
  .alarm-workspace__head.is-notifications {
    min-height: 56px;
  }
  .alarm-workspace__toolbar {
    width: 100%;
  }
  .alarm-workspace__search {
    width: 100%;
  }
  .alarm-workspace__total {
    display: none;
  }
  .alarm-workspace__create-actions {
    margin-left: auto;
  }
  .alarm-workspace__table-wrap {
    padding-inline: 12px;
  }
  .alarm-workspace__table :deep(.alarm-workspace__compact-hide) {
    display: none;
  }
}
</style>
