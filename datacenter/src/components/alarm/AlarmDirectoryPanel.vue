<template>
  <aside class="alarm-directories">
    <header>
      <strong>目录</strong>
      <button type="button" title="新建目录" @click="openCreate"><IconTablerFolderPlus /></button>
    </header>
    <button
      type="button"
      class="alarm-directories__item"
      :class="{ 'is-active': !selectedId }"
      @click="emit('select', '')"
    >
      <IconTablerFolders />
      <span>全部报警</span>
    </button>
    <div class="alarm-directories__list">
      <button
        v-for="group in groups"
        :key="group.id"
        type="button"
        class="alarm-directories__item"
        :class="{ 'is-active': selectedId === group.id }"
        @click="emit('select', group.id)"
      >
        <IconTablerFolder />
        <span :title="group.fullPath || group.name">{{ group.fullPath || group.name }}</span>
        <el-dropdown trigger="click" @command="(command: string) => handleCommand(command, group)">
          <span class="alarm-directories__more" @click.stop><IconTablerDots /></span>
          <template #dropdown
            ><el-dropdown-menu
              ><el-dropdown-item command="edit">编辑</el-dropdown-item
              ><el-dropdown-item command="delete">删除</el-dropdown-item></el-dropdown-menu
            ></template
          >
        </el-dropdown>
      </button>
    </div>

    <DcDialog v-model="dialogVisible" :title="editing ? '编辑目录' : '新建目录'" width="440px">
      <div class="alarm-directories__form">
        <label><span>名称</span><el-input v-model="draft.name" maxlength="100" /></label>
        <label
          ><span>上级目录</span
          ><el-select v-model="draft.parentId" clearable placeholder="根目录"
            ><el-option label="根目录" :value="null" /><el-option
              v-for="group in parentOptions"
              :key="group.id"
              :label="group.fullPath || group.name"
              :value="group.id" /></el-select
        ></label>
      </div>
      <template #footer
        ><button type="button" class="alarm-directories__cancel" @click="dialogVisible = false">
          取消</button
        ><button type="button" class="alarm-directories__save" :disabled="saving" @click="save">
          保存
        </button></template
      >
    </DcDialog>
  </aside>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import IconTablerDots from '~icons/tabler/dots'
import IconTablerFolder from '~icons/tabler/folder'
import IconTablerFolderPlus from '~icons/tabler/folder-plus'
import IconTablerFolders from '~icons/tabler/folders'
import DcDialog from '@/components/shared/DcDialog.vue'
import { createAlarmGroup, deleteAlarmGroup, updateAlarmGroup } from '@/api/alarm.api'
import type { AlarmGroup } from '@/api/schemas/alarm.schema'
import { getApiErrorMessage } from '@/utils/request'
import { useConfirm } from '@/composables/useConfirm'

const props = withDefaults(
  defineProps<{ projectId: string; groups?: AlarmGroup[]; selectedId?: string }>(),
  { groups: () => [], selectedId: '' },
)
const emit = defineEmits<{ select: [id: string]; changed: [] }>()
const { confirm } = useConfirm()
const dialogVisible = ref(false)
const editing = ref<AlarmGroup | null>(null)
const saving = ref(false)
const draft = reactive({
  name: '',
  parentId: null as string | null,
  description: null as string | null,
  sortOrder: 0,
})
const parentOptions = computed(() => props.groups.filter((item) => item.id !== editing.value?.id))

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
  if (!draft.name.trim()) return ElMessage.warning('请填写目录名称')
  saving.value = true
  try {
    const payload = { ...draft, name: draft.name.trim() }
    if (editing.value) await updateAlarmGroup(props.projectId, editing.value.id, payload)
    else await createAlarmGroup(props.projectId, payload)
    dialogVisible.value = false
    ElMessage.success('目录已保存')
    emit('changed')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '保存目录失败'))
  } finally {
    saving.value = false
  }
}
async function remove(group: AlarmGroup) {
  if (
    !(await confirm(`确认删除目录「${group.name}」？非空目录不能删除。`, {
      title: '删除目录',
      confirmText: '删除',
      type: 'warning',
    }))
  )
    return
  try {
    await deleteAlarmGroup(props.projectId, group.id)
    ElMessage.success('目录已删除')
    if (props.selectedId === group.id) emit('select', '')
    emit('changed')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '删除目录失败'))
  }
}
function handleCommand(command: string, group: AlarmGroup) {
  if (command === 'edit') openEdit(group)
  else void remove(group)
}
</script>

<style scoped>
.alarm-directories {
  width: 216px;
  min-width: 216px;
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding: 12px 10px;
  border-right: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle, var(--dc-surface-muted));
}
.alarm-directories header {
  height: 38px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 6px 8px;
}
.alarm-directories header strong {
  font-size: 13px;
  letter-spacing: 0;
}
.alarm-directories header button,
.alarm-directories__more {
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-muted);
  cursor: pointer;
}
.alarm-directories header button:hover,
.alarm-directories header button:focus-visible {
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}
.alarm-directories__list {
  min-height: 0;
  overflow: auto;
}
.alarm-directories__item {
  width: 100%;
  min-height: 36px;
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr) auto;
  align-items: center;
  gap: 7px;
  padding: 0 7px;
  border: 0;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  cursor: pointer;
  text-align: left;
  transition:
    background 0.16s ease,
    color 0.16s ease;
}
.alarm-directories__item:hover {
  background: var(--dc-surface-raised);
  color: var(--dc-text);
}
.alarm-directories__item:focus-visible {
  outline: none;
  box-shadow: 0 0 0 3px rgba(29, 78, 216, 0.12);
}
.alarm-directories__item > span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
}
.alarm-directories__item.is-active {
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
  font-weight: 700;
}
.alarm-directories__more {
  opacity: 0;
  transition: opacity 0.16s ease;
}
.alarm-directories__item:hover .alarm-directories__more,
.alarm-directories__item:focus-within .alarm-directories__more,
.alarm-directories__item.is-active .alarm-directories__more {
  opacity: 1;
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
.alarm-directories__cancel,
.alarm-directories__save {
  height: 32px;
  padding: 0 14px;
  border-radius: var(--dc-radius-sm);
  cursor: pointer;
}
.alarm-directories__cancel {
  border: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}
.alarm-directories__save {
  border: 1px solid var(--dc-primary);
  background: var(--dc-primary);
  color: white;
}
@container alarm-workspace (max-width: 760px) {
  .alarm-directories {
    width: 184px;
    min-width: 184px;
  }
}
@container alarm-workspace (max-width: 640px) {
  .alarm-directories {
    width: 100%;
    min-width: 0;
    max-height: 150px;
    flex: 0 0 auto;
    border-right: 0;
    border-bottom: 1px solid var(--dc-border);
  }
}
</style>
