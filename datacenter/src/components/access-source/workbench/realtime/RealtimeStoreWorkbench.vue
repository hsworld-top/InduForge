<template>
  <section class="realtime-store">
    <aside class="realtime-store__sidebar">
      <WorkbenchSourceHeader
        :title="connection.name || titleFallback"
        :fallback-title="titleFallback"
        :meta="sourceMetaRows"
        @back="$emit('back')"
      >
        <template #actions>
          <el-input
            v-model="keyword"
            class="realtime-store__search"
            size="small"
            clearable
            placeholder="搜索 Key"
            @keyup.enter="loadKeys"
          />
          <button
            type="button"
            class="workbench-source-header__icon-action is-primary"
            title="新建 Key"
            aria-label="新建 Key"
            @click="createDraftKey"
          >
            <IconTablerPlus />
          </button>
          <button
            type="button"
            class="workbench-source-header__icon-action"
            title="刷新"
            aria-label="刷新"
            @click="loadKeys"
          >
            <IconTablerRefresh />
          </button>
        </template>
      </WorkbenchSourceHeader>

      <div class="realtime-store__groups">
        <button
          type="button"
          class="realtime-store__group"
          :class="{ 'is-active': activeGroup === '' }"
          @click="selectGroup('')"
        >
          <IconTablerDatabase />
          <span>全部 Key</span>
          <em>{{ keys.length }}</em>
        </button>
        <button
          v-for="group in groups"
          :key="group.name"
          type="button"
          class="realtime-store__group"
          :class="{ 'is-active': activeGroup === group.name }"
          @click="selectGroup(group.name)"
        >
          <IconTablerFolder />
          <span>{{ group.name }}</span>
          <em>{{ group.count }}</em>
        </button>
      </div>

      <div class="realtime-store__keys" v-loading="loadingKeys">
        <button
          v-for="item in visibleKeys"
          :key="item.key"
          type="button"
          class="realtime-store__key"
          :class="{ 'is-active': activeTabKey === item.key }"
          @click="openKey(item.key)"
        >
          <span :title="item.key">{{ item.key }}</span>
          <small>{{ item.type }}</small>
        </button>
        <el-empty
          v-if="!loadingKeys && visibleKeys.length === 0"
          description="暂无 Key"
          :image-size="72"
        />
      </div>
    </aside>

    <main class="realtime-store__main">
      <div class="realtime-store__tabs">
        <button
          v-for="tab in tabs"
          :key="tab.key"
          type="button"
          class="realtime-store__tab"
          :class="{ 'is-active': tab.key === activeTabKey }"
          @click="activeTabKey = tab.key"
        >
          <span>{{ tab.draft.key || '新建 Key' }}</span>
          <i v-if="tab.dirty" />
          <IconTablerX @click.stop="closeTab(tab.key)" />
        </button>
      </div>

      <section v-if="activeTab" class="realtime-store__editor">
        <div class="realtime-store__toolbar">
          <div class="realtime-store__identity">
            <el-input
              v-model="activeTab.draft.key"
              class="realtime-store__key-input"
              placeholder="device:line1:status"
              @input="markDirty"
            />
            <el-select
              v-model="activeTab.draft.type"
              class="realtime-store__type"
              :disabled="activeTab.draft.type === 'stream'"
              @change="handleTypeChange"
            >
              <el-option
                v-for="type in editableTypes"
                :key="type"
                :label="type"
                :value="type"
              />
              <el-option v-if="activeTab.draft.type === 'stream'" label="stream" value="stream" />
            </el-select>
            <el-input-number
              v-model="activeTab.draft.ttlSeconds"
              class="realtime-store__ttl"
              :min="-1"
              :max="86400"
              controls-position="right"
              @change="markDirty"
            />
          </div>
          <div class="realtime-store__actions">
            <el-button :loading="loadingValue" @click="reloadActive">
              <IconTablerRefresh />
            </el-button>
            <el-button
              type="primary"
              :disabled="activeTab.draft.type === 'stream'"
              :loading="saving"
              @click="saveActive"
            >
              保存
            </el-button>
            <el-dropdown trigger="click" @command="handleCommand">
              <el-button>
                <IconTablerDots />
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="rename">重命名</el-dropdown-item>
                  <el-dropdown-item command="datapoint">创建数据点</el-dropdown-item>
                  <el-dropdown-item command="delete" divided>删除</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </div>

        <div class="realtime-store__meta">
          <span>{{ providerLabel }}</span>
          <span>TTL {{ formatTTL(activeTab.draft.ttlSeconds) }}</span>
          <span v-if="activeTab.draft.dataPointPath">数据点 {{ activeTab.draft.dataPointPath }}</span>
          <span v-else>未创建数据点</span>
        </div>

        <div class="realtime-store__content">
          <el-input
            v-if="activeTab.draft.type === 'string'"
            v-model="activeTab.draft.stringValue"
            type="textarea"
            :rows="14"
            resize="none"
            placeholder="输入字符串或 JSON 文本"
            @input="markDirty"
          />

          <el-table
            v-else-if="activeTab.draft.type === 'hash'"
            :data="activeTab.draft.rows"
            height="100%"
            border
            size="small"
          >
            <el-table-column label="Key" min-width="160">
              <template #default="{ row }">
                <el-input v-model="row.key" size="small" @input="markDirty" />
              </template>
            </el-table-column>
            <el-table-column label="Value" min-width="220">
              <template #default="{ row }">
                <el-input v-model="row.value" size="small" @input="markDirty" />
              </template>
            </el-table-column>
            <el-table-column width="54">
              <template #header>
                <el-button link type="primary" @click="addRow">+</el-button>
              </template>
              <template #default="{ $index }">
                <el-button link type="danger" @click="removeRow($index)">删</el-button>
              </template>
            </el-table-column>
          </el-table>

          <el-table
            v-else-if="['list', 'set'].includes(activeTab.draft.type)"
            :data="activeTab.draft.rows"
            height="100%"
            border
            size="small"
          >
            <el-table-column type="index" width="58" />
            <el-table-column label="Value" min-width="260">
              <template #default="{ row }">
                <el-input v-model="row.value" size="small" @input="markDirty" />
              </template>
            </el-table-column>
            <el-table-column width="54">
              <template #header>
                <el-button link type="primary" @click="addRow">+</el-button>
              </template>
              <template #default="{ $index }">
                <el-button link type="danger" @click="removeRow($index)">删</el-button>
              </template>
            </el-table-column>
          </el-table>

          <el-table
            v-else-if="activeTab.draft.type === 'zset'"
            :data="activeTab.draft.rows"
            height="100%"
            border
            size="small"
          >
            <el-table-column label="Member" min-width="220">
              <template #default="{ row }">
                <el-input v-model="row.member" size="small" @input="markDirty" />
              </template>
            </el-table-column>
            <el-table-column label="Score" width="160">
              <template #default="{ row }">
                <el-input-number
                  v-model="row.score"
                  size="small"
                  controls-position="right"
                  @change="markDirty"
                />
              </template>
            </el-table-column>
            <el-table-column width="54">
              <template #header>
                <el-button link type="primary" @click="addRow">+</el-button>
              </template>
              <template #default="{ $index }">
                <el-button link type="danger" @click="removeRow($index)">删</el-button>
              </template>
            </el-table-column>
          </el-table>

          <pre v-else class="realtime-store__readonly">{{ formattedReadonlyValue }}</pre>
        </div>
      </section>

      <el-empty v-else class="realtime-store__empty-main" description="选择或新建一个 Key" />
    </main>

    <el-dialog v-model="renameDialog.visible" title="重命名 Key" width="420px">
      <el-input v-model="renameDialog.newKey" placeholder="新的 Key 名称" />
      <template #footer>
        <el-button @click="renameDialog.visible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="confirmRename">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="datapointDialog.visible" title="创建数据点" width="420px">
      <el-form label-width="86px">
        <el-form-item label="数据类型">
          <el-select v-model="datapointDialog.dataType">
            <el-option label="object" value="object" />
            <el-option label="array" value="array" />
            <el-option label="string" value="string" />
            <el-option label="number" value="number" />
            <el-option label="boolean" value="boolean" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="datapointDialog.visible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="confirmCreateDatapoint">
          创建
        </el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import dataAPI from '@/api/data.api'
import { getApiErrorMessage } from '@/utils/request'
import WorkbenchSourceHeader from '@/components/workbench/WorkbenchSourceHeader.vue'
import IconTablerDatabase from '~icons/tabler/database'
import IconTablerDots from '~icons/tabler/dots'
import IconTablerFolder from '~icons/tabler/folder'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerX from '~icons/tabler/x'

type AccessSourceConnection = {
  id: string
  name?: string
  type?: string
  config?: Record<string, any>
}

type KeySummary = {
  id?: string
  key: string
  type: string
  ttl: number
  size: number
  provider: string
  valueType?: string
  dataPointId?: string
  dataPointPath?: string
}

type KeyGroup = {
  name: string
  count: number
}

type EditorRow = {
  key?: string
  value?: string
  member?: string
  score?: number
}

type KeyDraft = {
  originalKey: string
  key: string
  type: string
  ttlSeconds: number
  valueType: string
  stringValue: string
  rows: EditorRow[]
  readonlyValue: unknown
  dataPointPath: string
}

type KeyTab = {
  key: string
  dirty: boolean
  draft: KeyDraft
}

const props = defineProps<{
  connection: AccessSourceConnection
  projectId: string
}>()

defineEmits<{
  (event: 'back'): void
}>()

const editableTypes = ['string', 'hash', 'list', 'set', 'zset']
const keyword = ref('')
const activeGroup = ref('')
const keys = ref<KeySummary[]>([])
const groups = ref<KeyGroup[]>([])
const tabs = ref<KeyTab[]>([])
const activeTabKey = ref('')
const loadingKeys = ref(false)
const loadingValue = ref(false)
const saving = ref(false)

const renameDialog = reactive({ visible: false, newKey: '' })
const datapointDialog = reactive({ visible: false, dataType: 'object' })

const config = computed(() => props.connection.config || {})
const isBuiltin = computed(() => props.connection.type === 'builtin.realtime')
const titleFallback = computed(() => (isBuiltin.value ? 'IF实时库' : 'Redis'))
const providerLabel = computed(() => (isBuiltin.value ? 'IF 实时库' : 'Redis'))
const sourceMetaRows = computed(() => {
  if (isBuiltin.value) {
    return [
      { label: '类型', value: 'IF实时库' },
      { label: 'RuntimeKey', value: String(config.value.runtimeKey || props.connection.id) },
    ]
  }
  return [
    { label: '类型', value: 'Redis' },
    { label: '地址', value: String(config.value.address || '-') },
    { label: 'DB', value: String(config.value.db ?? config.value.database ?? 0) },
  ]
})

const visibleKeys = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  return keys.value.filter((item) => {
    const groupMatched = !activeGroup.value || keyGroup(item.key) === activeGroup.value
    const keywordMatched = !q || item.key.toLowerCase().includes(q) || item.type.includes(q)
    return groupMatched && keywordMatched
  })
})

const activeTab = computed(() => tabs.value.find((tab) => tab.key === activeTabKey.value) || null)
const formattedReadonlyValue = computed(() => JSON.stringify(activeTab.value?.draft.readonlyValue ?? null, null, 2))

const loadKeys = async () => {
  loadingKeys.value = true
  try {
    const response = await dataAPI.getRealtimeStoreKeys(props.projectId, props.connection.id, {
      q: keyword.value || undefined,
      group: activeGroup.value || undefined,
    })
    keys.value = response?.data?.list || []
    groups.value = response?.data?.groups || []
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载实时库 Key 失败'))
  } finally {
    loadingKeys.value = false
  }
}

const selectGroup = (group: string) => {
  activeGroup.value = group
}

const createDraftKey = () => {
  const key = `__draft__${Date.now()}`
  tabs.value.push({
    key,
    dirty: true,
    draft: {
      originalKey: '',
      key: '',
      type: 'string',
      ttlSeconds: -1,
      valueType: 'object',
      stringValue: '',
      rows: [],
      readonlyValue: null,
      dataPointPath: '',
    },
  })
  activeTabKey.value = key
}

const openKey = async (key: string) => {
  const existing = tabs.value.find((tab) => tab.draft.originalKey === key || tab.draft.key === key)
  if (existing) {
    activeTabKey.value = existing.key
    return
  }
  loadingValue.value = true
  try {
    const response = await dataAPI.getRealtimeStoreKey(props.projectId, props.connection.id, key)
    const draft = createDraftFromResponse(response?.data || {})
    tabs.value.push({ key, dirty: false, draft })
    activeTabKey.value = key
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '读取实时库 Key 失败'))
  } finally {
    loadingValue.value = false
  }
}

const reloadActive = async () => {
  if (!activeTab.value?.draft.key) return
  const key = activeTab.value.draft.originalKey || activeTab.value.draft.key
  await openOrReplaceKey(key)
}

const openOrReplaceKey = async (key: string) => {
  loadingValue.value = true
  try {
    const response = await dataAPI.getRealtimeStoreKey(props.projectId, props.connection.id, key)
    const draft = createDraftFromResponse(response?.data || {})
    const tab = activeTab.value
    if (tab) {
      tab.draft = draft
      tab.key = draft.key
      tab.dirty = false
      activeTabKey.value = draft.key
    }
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '刷新实时库 Key 失败'))
  } finally {
    loadingValue.value = false
  }
}

const saveActive = async () => {
  const tab = activeTab.value
  if (!tab) return
  if (!tab.draft.key.trim()) {
    ElMessage.warning('Key 不能为空')
    return
  }
  saving.value = true
  try {
    const payload = {
      key: tab.draft.key.trim(),
      type: tab.draft.type,
      ttlSeconds: tab.draft.ttlSeconds,
      valueType: tab.draft.valueType || 'object',
      value: buildSaveValue(tab.draft),
    }
    const response = await dataAPI.saveRealtimeStoreKey(props.projectId, props.connection.id, payload)
    const draft = createDraftFromResponse(response?.data || {})
    tab.draft = draft
    tab.key = draft.key
    tab.dirty = false
    activeTabKey.value = draft.key
    await loadKeys()
    ElMessage.success('Key 已保存')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '保存实时库 Key 失败'))
  } finally {
    saving.value = false
  }
}

const closeTab = async (key: string) => {
  const tab = tabs.value.find((item) => item.key === key)
  if (tab?.dirty) {
    await ElMessageBox.confirm('当前 Key 有未保存改动，关闭后会丢失。', '关闭 Key', {
      type: 'warning',
    })
  }
  const index = tabs.value.findIndex((item) => item.key === key)
  if (index >= 0) tabs.value.splice(index, 1)
  if (activeTabKey.value === key) {
    activeTabKey.value = tabs.value[Math.max(0, index - 1)]?.key || ''
  }
}

const handleCommand = async (command: string) => {
  if (command === 'rename') {
    renameDialog.newKey = activeTab.value?.draft.key || ''
    renameDialog.visible = true
  } else if (command === 'datapoint') {
    datapointDialog.dataType = activeTab.value?.draft.valueType || 'object'
    datapointDialog.visible = true
  } else if (command === 'delete') {
    await deleteActive()
  }
}

const confirmRename = async () => {
  const tab = activeTab.value
  if (!tab) return
  saving.value = true
  try {
    const key = tab.draft.originalKey || tab.draft.key
    const response = await dataAPI.renameRealtimeStoreKey(props.projectId, props.connection.id, key, {
      newKey: renameDialog.newKey,
    })
    const draft = createDraftFromResponse(response?.data || {})
    tab.draft = draft
    tab.key = draft.key
    tab.dirty = false
    activeTabKey.value = draft.key
    renameDialog.visible = false
    await loadKeys()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '重命名 Key 失败'))
  } finally {
    saving.value = false
  }
}

const confirmCreateDatapoint = async () => {
  const tab = activeTab.value
  if (!tab?.draft.key) return
  saving.value = true
  try {
    const response = await dataAPI.createRealtimeStoreKeyDatapoint(
      props.projectId,
      props.connection.id,
      tab.draft.originalKey || tab.draft.key,
      { dataType: datapointDialog.dataType },
    )
    tab.draft.dataPointPath = response?.data?.path || ''
    datapointDialog.visible = false
    await loadKeys()
    ElMessage.success('数据点已创建')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '创建数据点失败'))
  } finally {
    saving.value = false
  }
}

const deleteActive = async () => {
  const tab = activeTab.value
  if (!tab?.draft.key) return
  await ElMessageBox.confirm('删除 Key 后，对应数据点会标记为失效。', '删除 Key', {
    type: 'warning',
  })
  saving.value = true
  try {
    await dataAPI.deleteRealtimeStoreKey(
      props.projectId,
      props.connection.id,
      tab.draft.originalKey || tab.draft.key,
    )
    tabs.value = tabs.value.filter((item) => item.key !== tab.key)
    activeTabKey.value = tabs.value[0]?.key || ''
    await loadKeys()
    ElMessage.success('Key 已删除')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '删除 Key 失败'))
  } finally {
    saving.value = false
  }
}

const markDirty = () => {
  if (activeTab.value) activeTab.value.dirty = true
}

const handleTypeChange = () => {
  const draft = activeTab.value?.draft
  if (!draft) return
  draft.rows = []
  draft.stringValue = ''
  markDirty()
}

const addRow = () => {
  const draft = activeTab.value?.draft
  if (!draft) return
  if (draft.type === 'zset') draft.rows.push({ member: '', score: 0 })
  else if (draft.type === 'hash') draft.rows.push({ key: '', value: '' })
  else draft.rows.push({ value: '' })
  markDirty()
}

const removeRow = (index: number) => {
  activeTab.value?.draft.rows.splice(index, 1)
  markDirty()
}

const createDraftFromResponse = (data: any): KeyDraft => {
  const type = data.type || 'string'
  return {
    originalKey: data.key || '',
    key: data.key || '',
    type,
    ttlSeconds: Number(data.ttl ?? -1),
    valueType: data.valueType || 'object',
    stringValue: type === 'string' ? stringifyEditorValue(data.value) : '',
    rows: rowsFromValue(type, data.value),
    readonlyValue: data.value,
    dataPointPath: data.dataPointPath || '',
  }
}

const rowsFromValue = (type: string, value: any): EditorRow[] => {
  if (type === 'hash' && value && typeof value === 'object' && !Array.isArray(value)) {
    return Object.entries(value).map(([key, rowValue]) => ({ key, value: stringifyEditorValue(rowValue) }))
  }
  if (['list', 'set'].includes(type) && Array.isArray(value)) {
    return value.map((item) => ({ value: stringifyEditorValue(item) }))
  }
  if (type === 'zset' && Array.isArray(value)) {
    return value.map((item) => ({
      member: stringifyEditorValue(item?.member),
      score: Number(item?.score || 0),
    }))
  }
  return []
}

const buildSaveValue = (draft: KeyDraft) => {
  if (draft.type === 'string') return draft.stringValue
  if (draft.type === 'hash') {
    return draft.rows
      .filter((row) => row.key)
      .map((row) => ({ key: row.key, value: row.value || '' }))
  }
  if (['list', 'set'].includes(draft.type)) return draft.rows.map((row) => row.value || '')
  if (draft.type === 'zset') {
    return draft.rows
      .filter((row) => row.member)
      .map((row) => ({ member: row.member, score: Number(row.score || 0) }))
  }
  return draft.readonlyValue
}

const stringifyEditorValue = (value: unknown) => {
  if (typeof value === 'string') return value
  if (value == null) return ''
  return JSON.stringify(value)
}

const keyGroup = (key: string) => {
  const index = key.indexOf(':')
  return index > 0 ? key.slice(0, index) : '未分组'
}

const formatTTL = (ttl: number) => {
  if (ttl < 0) return '不过期'
  if (ttl === 0) return '立即'
  return `${ttl}s`
}

onMounted(loadKeys)
</script>

<style scoped>
.realtime-store {
  height: 100%;
  min-height: 0;
  display: grid;
  grid-template-columns: 300px minmax(0, 1fr);
  background: #f6f8fb;
  color: #172033;
}

.realtime-store__sidebar {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  border-right: 1px solid #dfe5ee;
  background: #fff;
}

.realtime-store__search {
  width: 150px;
}

.realtime-store__groups {
  padding: 10px;
  border-bottom: 1px solid #edf0f5;
}

.realtime-store__group,
.realtime-store__key,
.realtime-store__tab {
  border: 0;
  background: transparent;
  font: inherit;
  color: inherit;
  cursor: pointer;
}

.realtime-store__group {
  width: 100%;
  height: 32px;
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
  padding: 0 8px;
  border-radius: 6px;
  color: #4c5b70;
}

.realtime-store__group svg,
.realtime-store__actions svg,
.realtime-store__tab svg {
  width: 16px;
  height: 16px;
}

.realtime-store__group span,
.realtime-store__key span,
.realtime-store__tab span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.realtime-store__group em {
  font-style: normal;
  color: #7f8ca3;
  font-size: 12px;
}

.realtime-store__group:hover,
.realtime-store__group.is-active {
  background: #eef5ff;
  color: #1d4ed8;
}

.realtime-store__keys {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 8px;
}

.realtime-store__key {
  width: 100%;
  min-height: 34px;
  display: grid;
  grid-template-columns: minmax(0, 1fr) 54px;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  border-radius: 6px;
  text-align: left;
}

.realtime-store__key small {
  justify-self: end;
  color: #6b7688;
  font-size: 12px;
}

.realtime-store__key:hover,
.realtime-store__key.is-active {
  background: #f0f6ff;
}

.realtime-store__main {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.realtime-store__tabs {
  height: 36px;
  display: flex;
  align-items: end;
  gap: 2px;
  padding: 0 8px;
  border-bottom: 1px solid #dfe5ee;
  background: #fff;
  overflow-x: auto;
}

.realtime-store__tab {
  height: 32px;
  max-width: 220px;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 0 10px;
  border: 1px solid transparent;
  border-bottom: 0;
  border-radius: 6px 6px 0 0;
  color: #4b5565;
}

.realtime-store__tab.is-active {
  background: #f6f8fb;
  border-color: #dfe5ee;
  color: #172033;
}

.realtime-store__tab i {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #f59e0b;
}

.realtime-store__editor {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding: 12px;
}

.realtime-store__toolbar {
  min-height: 44px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.realtime-store__identity {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
}

.realtime-store__key-input {
  max-width: 520px;
}

.realtime-store__type {
  width: 112px;
}

.realtime-store__ttl {
  width: 132px;
}

.realtime-store__actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.realtime-store__meta {
  height: 30px;
  display: flex;
  align-items: center;
  gap: 16px;
  color: #697589;
  font-size: 12px;
}

.realtime-store__content {
  flex: 1;
  min-height: 0;
}

.realtime-store__content :deep(.el-textarea),
.realtime-store__content :deep(.el-textarea__inner) {
  height: 100%;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
}

.realtime-store__readonly {
  height: 100%;
  overflow: auto;
  margin: 0;
  padding: 12px;
  border: 1px solid #dfe5ee;
  border-radius: 6px;
  background: #fff;
  color: #263244;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 13px;
}

.realtime-store__empty-main {
  flex: 1;
  min-height: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

@media (max-width: 960px) {
  .realtime-store {
    grid-template-columns: 260px minmax(0, 1fr);
  }

  .realtime-store__toolbar,
  .realtime-store__identity {
    align-items: stretch;
    flex-direction: column;
  }

  .realtime-store__actions {
    align-self: flex-start;
  }
}
</style>
