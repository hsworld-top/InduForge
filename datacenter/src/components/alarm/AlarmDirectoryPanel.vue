<template>
  <TreePanel
    class="alarm-directories"
    :title="t('alarm.alarmDirectory')"
    :search-text="searchText"
    :search-placeholder="t('alarm.searchDirectory')"
    :all-label="t('alarm.allAlarms')"
    :all-active="!selectedId"
    :loading="loading"
    @update:search-text="updateSearch"
    @select-all="emit('select', '')"
  >
    <template #actions>
      <button type="button" class="is-primary" :title="t('alarm.createDirectory')" @click="openCreate">
        <IconTablerFolderPlus />
      </button>
      <button type="button" :title="t('alarm.refreshDirectory')" @click="load(true)">
        <IconTablerRefresh />
      </button>
    </template>

    <template v-if="searchText.trim()">
      <button
        v-for="group in groups"
        :key="group.id"
        type="button"
        class="alarm-directories__search-item"
        :class="{ 'is-active': selectedId === group.id }"
        @click="emit('select', group.id)"
      >
        <IconTablerFolder />
        <span :title="group.fullPath || group.name">{{ group.fullPath || group.name }}</span>
      </button>
    </template>
    <template v-else>
      <AlarmDirectoryNode
        v-for="group in groups"
        :key="group.id"
        :project-id="projectId"
        :group="group"
        :selected-id="selectedId"
        @select="emit('select', $event)"
        @edit="openEdit"
        @delete="remove"
      />
    </template>
    <el-empty v-if="!loading && groups.length === 0" :image-size="42" :description="t('alarm.emptyDirectory')" />
  </TreePanel>

  <DcDialog v-model="dialogVisible" :title="editing ? t('alarm.editDirectory') : t('alarm.createDirectory')" width="440px">
    <div class="alarm-directories__form">
      <label><span>{{ t('alarm.nameField') }}</span><el-input v-model="draft.name" maxlength="100" /></label>
      <label>
        <span>{{ t('alarm.parentDirectory') }}</span>
        <AlarmGroupSelect
          v-model="draft.parentId"
          :project-id="projectId"
          :initial-label="editing?.fullPath || ''"
        />
      </label>
    </div>
    <template #footer>
      <button type="button" class="dc-button" @click="dialogVisible = false">{{ t('alarm.cancel') }}</button>
      <button type="button" class="dc-button dc-button--primary" :disabled="saving" @click="save">
        {{ t('alarm.save') }}
      </button>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import IconTablerFolder from '~icons/tabler/folder'
import IconTablerFolderPlus from '~icons/tabler/folder-plus'
import IconTablerRefresh from '~icons/tabler/refresh'
import AlarmDirectoryNode from './AlarmDirectoryNode.vue'
import AlarmGroupSelect from './AlarmGroupSelect.vue'
import DcDialog from '@/components/shared/DcDialog.vue'
import TreePanel from '@/components/shared/TreePanel.vue'
import {
  createAlarmGroup,
  deleteAlarmGroup,
  listAlarmGroups,
  updateAlarmGroup,
} from '@/api/alarm.api'
import type { AlarmGroup } from '@/api/schemas/alarm.schema'
import { getApiErrorMessage } from '@/utils/request'
import { useConfirm } from '@/composables/useConfirm'
import { t } from '@/i18n/runtime'

const props = withDefaults(defineProps<{ projectId: string; selectedId?: string }>(), {
  selectedId: '',
})
const emit = defineEmits<{ select: [id: string] }>()
const { confirm } = useConfirm()
const groups = ref<AlarmGroup[]>([])
const loading = ref(false)
const searchText = ref('')
const dialogVisible = ref(false)
const editing = ref<AlarmGroup | null>(null)
const saving = ref(false)
const draft = reactive({
  name: '',
  parentId: null as string | null,
  description: null as string | null,
  sortOrder: 0,
})
let searchTimer = 0
let requestVersion = 0

async function load(reset = false) {
  if (!props.projectId || (loading.value && !reset)) return
  if (reset) {
    groups.value = []
  }
  const version = ++requestVersion
  loading.value = true
  try {
    const loaded: AlarmGroup[] = []
    let page = 1
    let totalPages = 1
    do {
      const result = await listAlarmGroups(props.projectId, {
        search: searchText.value.trim() || undefined,
        page,
        pageSize: 100,
      })
      if (version !== requestVersion) return
      loaded.push(...result.list)
      totalPages = result.pagination.totalPages
      page += 1
    } while (page <= totalPages)
    groups.value = Array.from(new Map(loaded.map((item) => [item.id, item])).values())
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, t('alarm.loadDirectoryFailed')))
  } finally {
    if (version === requestVersion) loading.value = false
  }
}

function queueSearch() {
  window.clearTimeout(searchTimer)
  searchTimer = window.setTimeout(() => void load(true), 250)
}
function updateSearch(value: string) {
  searchText.value = value
  queueSearch()
}
function openCreate() {
  editing.value = null
  Object.assign(draft, { name: '', parentId: null, description: null, sortOrder: 0 })
  dialogVisible.value = true
}
function openEdit(group: AlarmGroup) {
  editing.value = group
  Object.assign(draft, {
    name: group.name,
    parentId: group.parentId || null,
    description: group.description || null,
    sortOrder: group.sortOrder,
  })
  dialogVisible.value = true
}
async function save() {
  if (!draft.name.trim()) return ElMessage.warning(t('alarm.directoryNameRequired'))
  saving.value = true
  try {
    const payload = { ...draft, name: draft.name.trim() }
    if (editing.value) await updateAlarmGroup(props.projectId, editing.value.id, payload)
    else await createAlarmGroup(props.projectId, payload)
    dialogVisible.value = false
    ElMessage.success(t('alarm.directorySaved'))
    await load(true)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, t('alarm.saveDirectoryFailed')))
  } finally {
    saving.value = false
  }
}
async function remove(group: AlarmGroup) {
  const accepted = await confirm(t('alarm.deleteDirectoryConfirm', { name: group.name }), {
    title: t('alarm.deleteDirectoryTitle'),
    confirmText: t('alarm.delete'),
    type: 'warning',
  })
  if (!accepted) return
  try {
    await deleteAlarmGroup(props.projectId, group.id)
    if (props.selectedId === group.id) emit('select', '')
    ElMessage.success(t('alarm.directoryDeleted'))
    await load(true)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, t('alarm.deleteDirectoryFailed')))
  }
}

watch(
  () => props.projectId,
  () => void load(true),
)
onMounted(() => void load(true))
</script>

<style scoped>
.alarm-directories {
  width: 260px;
  min-width: 260px;
}
.alarm-directories__search-item {
  width: 100%;
  min-height: 32px;
  display: flex;
  align-items: center;
  gap: 7px;
  padding: 0 7px;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  cursor: pointer;
  text-align: left;
}
.alarm-directories__search-item.is-active,
.alarm-directories__search-item:hover {
  border-color: rgba(29, 78, 216, 0.24);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}
.alarm-directories__search-item span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
}
.alarm-directories__form {
  display: grid;
  gap: 14px;
}
.alarm-directories__form label {
  display: grid;
  gap: 6px;
  color: var(--dc-text-secondary);
  font-size: 12px;
}
</style>
