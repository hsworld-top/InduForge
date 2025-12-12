<template>
  <div class="mysql-content h-full flex flex-col overflow-hidden">
    <!-- 标签页系统 -->
    <div class="flex-1 flex flex-col overflow-hidden">
      <el-tabs
        v-model="activeTab"
        type="card"
        closable
        class="query-tabs flex-1 flex flex-col overflow-hidden"
        @tab-remove="handleCloseTab"
      >
        <!-- 表列表标签页（不可关闭） -->
        <el-tab-pane
          name="table-list"
          :closable="false"
          class="flex-1 flex flex-col overflow-hidden"
        >
          <template #label>
            <span class="flex items-center">
              <IconTablerTable class="mr-1 w-4 h-4" />
              <span>表列表</span>
            </span>
          </template>

          <div class="table-list-content flex-1 flex flex-col overflow-hidden">
            <!-- 工具栏 -->
            <div class="table-toolbar flex items-center justify-between px-4 py-3 border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800">
              <div class="text-sm text-gray-700 dark:text-gray-300">
                {{ connection.name }} - 表列表
              </div>
              <div class="space-x-2">
                <el-button type="primary" size="small" @click="createNewQuery">
                  <IconTablerPlus class="mr-1 w-4 h-4" />
                  新建查询
                </el-button>
              </div>
            </div>

            <!-- 表列表 -->
            <div class="flex-1 overflow-y-auto p-4 min-h-0">
              <MysqlTableList
                :connection-id="connectionId"
                :loading="loading"
                @table-select="handleTableSelect"
              />
            </div>
          </div>
        </el-tab-pane>

        <!-- 查询标签页 -->
        <el-tab-pane
          v-for="tab in queryTabs"
          :key="tab.id"
          :name="tab.id"
          :lazy="false"
          class="flex-1 flex flex-col overflow-hidden"
        >
          <template #label>
            <span class="flex items-center">
              <span>{{ tab.name }}</span>
              <IconTablerAlertCircle v-if="tab.modified" class="ml-1 text-orange-500 w-3 h-3" />
            </span>
          </template>

          <div class="query-editor-tab-content flex-1 flex flex-col overflow-y-auto p-4">
            <MysqlQueryEditor
              v-if="activeTab === tab.id"
              :tab="tab"
              @execute="handleExecuteQuery"
              @save="handleSaveQuery"
            />
          </div>
        </el-tab-pane>
      </el-tabs>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, provide } from 'vue'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerTable from '~icons/tabler/table'
import IconTablerAlertCircle from '~icons/tabler/alert-circle'
import MysqlTableList from './MysqlTableList.vue'
import MysqlQueryEditor from './MysqlQueryEditor.vue'

const props = defineProps({
  connectionId: {
    type: String,
    required: true
  },
  connection: {
    type: Object,
    required: true
  }
})

const emit = defineEmits(['query-execute', 'query-save'])

const queryTabs = ref([])
const activeTab = ref('table-list')
const loading = ref(false)
let queryTabCounter = 0

// 提供给子组件使用
provide('connectionId', computed(() => props.connectionId))
provide('connection', computed(() => props.connection))

/**
 * 创建新查询
 */
const createNewQuery = () => {
  queryTabCounter++
  const tabId = `query-${queryTabCounter}`
  
  const newTab = {
    id: tabId,
    name: `查询 ${queryTabCounter}`,
    connectionId: props.connectionId,
    table: '',
    sql: 'SELECT * FROM ',
    result: null,
    executing: false,
    saving: false,
    modified: false,
    resultPage: 1,
    resultPageSize: 20,
    parameters: [],
    parametersExpanded: true,
  }

  queryTabs.value.push(newTab)
  activeTab.value = tabId
}

/**
 * 选择表（双击表）
 */
const handleTableSelect = (tableName) => {
  // 双击表时不再切换视图，而是创建查询标签页
  createQueryFromTable(tableName)
}

/**
 * 执行查询
 */
const handleExecuteQuery = (data) => {
  emit('query-execute', data)
}

/**
 * 保存查询
 */
const handleSaveQuery = (data) => {
  emit('query-save', data)
}

/**
 * 关闭标签页
 */
const handleCloseTab = (tabId) => {
  // 不允许关闭表列表标签页
  if (tabId === 'table-list') return
  
  const index = queryTabs.value.findIndex(t => t.id === tabId)
  if (index > -1) {
    queryTabs.value.splice(index, 1)

    // 如果关闭的是当前活动标签页，切换到其他标签页
    if (activeTab.value === tabId) {
      if (queryTabs.value.length > 0) {
        activeTab.value = queryTabs.value[queryTabs.value.length - 1].id
      } else {
        activeTab.value = 'table-list'
      }
    }
  }
}

/**
 * 从表双击创建查询
 */
const createQueryFromTable = (tableName) => {
  queryTabCounter++
  const tabId = `query-${queryTabCounter}`
  const sql = `SELECT * FROM \`${tableName}\` LIMIT 100`

  const newTab = {
    id: tabId,
    name: `${tableName}`,
    connectionId: props.connectionId,
    table: tableName,
    sql: sql,
    result: null,
    executing: false,
    saving: false,
    modified: false,
    resultPage: 1,
    resultPageSize: 20,
    parameters: [],
    parametersExpanded: true,
    autoExecute: true,
  }

  queryTabs.value.push(newTab)
  activeTab.value = tabId
}

/**
 * 显示表列表标签页
 */
const showTableList = () => {
  activeTab.value = 'table-list'
}

/**
 * 打开已保存的查询
 */
const openQuery = (query) => {
  // 检查是否已经打开
  const existingTab = queryTabs.value.find(t => t.queryId === query.id)
  if (existingTab) {
    activeTab.value = existingTab.id
    return
  }

  // 创建新标签页
  queryTabCounter++
  const tabId = `query-${queryTabCounter}`
  
  const newTab = {
    id: tabId,
    queryId: query.id,
    name: query.name,
    connectionId: props.connectionId,
    table: '',
    sql: query.config?.sql || '',
    result: null,
    executing: false,
    saving: false,
    modified: false,
    resultPage: 1,
    resultPageSize: 20,
    parameters: (query.config?.parameters || []).map(p => ({
      name: p.name,
      type: p.type || 'string',
      value: p.default || ''
    })),
    parametersExpanded: true,
  }

  queryTabs.value.push(newTab)
  activeTab.value = tabId
}

/**
 * 关闭查询标签页（通过查询ID）
 */
const closeQueryTab = (queryId) => {
  const index = queryTabs.value.findIndex(t => t.queryId === queryId)
  if (index > -1) {
    const tabId = queryTabs.value[index].id
    handleCloseTab(tabId)
  }
}

// 暴露方法给父组件
defineExpose({
  createQueryFromTable,
  createNewQuery,
  showTableList,
  openQuery,
  closeQueryTab,
})
</script>

<style scoped>
.mysql-content {
  background-color: #f9fafb;
}

.dark .mysql-content {
  background-color: #111827;
}

.query-tabs :deep(.el-tabs__header) {
  margin: 0;
  flex-shrink: 0;
}

.query-tabs :deep(.el-tabs__content) {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.query-tabs :deep(.el-tab-pane) {
  height: 100%;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  min-height: 0;
}
</style>
