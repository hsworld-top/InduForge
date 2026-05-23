<template>
  <section class="sql-workbench">
    <aside class="sql-workbench__explorer">
      <div class="sql-workbench__source">
        <button type="button" class="sql-workbench__back" @click="$emit('back')">
          <IconTablerArrowLeft />
          <span>返回接入源</span>
        </button>
        <span>{{ dbTypeLabel }}</span>
        <strong>{{ connection.name || '未命名接入源' }}</strong>
        <small>{{ databaseLabel }}</small>
      </div>

      <div class="sql-workbench__toolbar">
        <button type="button" title="新建查询" @click="createQueryTab()">
          <IconTablerPlus />
        </button>
        <button type="button" title="刷新对象" @click="reloadExplorer">
          <IconTablerRefresh />
        </button>
      </div>

      <el-input
        v-model="filterText"
        class="sql-workbench__search"
        size="small"
        placeholder="筛选表或查询"
        clearable
      />

      <div class="sql-workbench__tree">
        <section class="sql-workbench__tree-section">
          <button
            type="button"
            class="sql-workbench__tree-head"
            @click="queriesExpanded = !queriesExpanded"
          >
            <IconTablerChevronDown v-if="queriesExpanded" />
            <IconTablerChevronRight v-else />
            <span>保存的查询</span>
            <small>{{ filteredQueries.length }}</small>
          </button>

          <div v-if="queriesExpanded" class="sql-workbench__tree-body">
            <div v-if="queriesLoading" class="sql-workbench__loading">
              <IconTablerLoader2 />
              <span>加载查询...</span>
            </div>
            <template v-else>
              <button
                v-for="query in filteredQueries"
                :key="query.id"
                type="button"
                class="sql-workbench__tree-item"
                @dblclick="openSavedQuery(query)"
              >
                <IconTablerFileText />
                <span>{{ query.name }}</span>
                <small
                  class="sql-workbench__datapoint-pill"
                  :class="{
                    'is-missing': !queryDataPointBySourceId[query.id],
                  }"
                >
                  {{
                    queryDataPointBySourceId[query.id]?.status === 'invalid'
                      ? '数据点失效'
                      : queryDataPointBySourceId[query.id]
                        ? '数据点'
                        : '未同步'
                  }}
                </small>
              </button>
            </template>
            <div
              v-if="!queriesLoading && filteredQueries.length === 0"
              class="sql-workbench__empty"
            >
              暂无保存查询
            </div>
          </div>
        </section>

        <section class="sql-workbench__tree-section">
          <button
            type="button"
            class="sql-workbench__tree-head"
            @click="tablesExpanded = !tablesExpanded"
          >
            <IconTablerChevronDown v-if="tablesExpanded" />
            <IconTablerChevronRight v-else />
            <span>表</span>
            <small>{{ filteredTables.length }}</small>
          </button>

          <div v-if="tablesExpanded" class="sql-workbench__tree-body">
            <div v-if="tablesLoading" class="sql-workbench__loading">
              <IconTablerLoader2 />
              <span>加载表结构...</span>
            </div>
            <template v-else>
              <div
                v-for="table in filteredTables"
                :key="table.name"
                role="button"
                tabindex="0"
                class="sql-workbench__tree-item"
                :class="{ 'is-active': selectedTableName === table.name }"
                @click="selectedTableName = table.name"
                @dblclick="openTableData(table)"
                @keydown.enter="openTableData(table)"
              >
                <IconTablerTable />
                <span>{{ table.name }}</span>
                <small v-if="table.rows !== undefined">{{ table.rows }}</small>
                <button
                  type="button"
                  class="sql-workbench__inline-action"
                  title="查看结构"
                  @click.stop="openTableStructure(table)"
                >
                  <IconTablerColumns />
                </button>
              </div>
            </template>
            <div v-if="!tablesLoading && filteredTables.length === 0" class="sql-workbench__empty">
              暂无表
            </div>
          </div>
        </section>
      </div>
    </aside>

    <main class="sql-workbench__main">
      <div class="sql-workbench__tabbar">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          type="button"
          class="sql-workbench__tab"
          :class="{ 'is-active': activeTabId === tab.id }"
          @click="activeTabId = tab.id"
        >
          <component :is="tab.icon" />
          <span>{{ tab.title }}</span>
          <i v-if="tab.modified" />
          <IconTablerX
            v-if="tabs.length > 1"
            class="sql-workbench__tab-close"
            @click.stop="closeTab(tab.id)"
          />
        </button>
      </div>

      <div v-if="activeTab" class="sql-workbench__content">
        <template v-if="activeTab.type === 'query'">
          <div class="sql-workbench__editor-toolbar">
            <div class="sql-workbench__context-line">
              <IconTablerDatabase />
              <span>{{ databaseLabel }}</span>
              <span v-if="activeTab.table">/ {{ activeTab.table }}</span>
            </div>
            <div class="sql-workbench__actions">
              <el-button size="small" @click="formatActiveSql"> 格式化 </el-button>
              <el-button size="small" :loading="activeTab.saving" @click="saveActiveQuery">
                保存
              </el-button>
              <el-button
                type="primary"
                size="small"
                :loading="activeTab.executing"
                @click="executeActiveQuery"
              >
                <IconTablerPlayerPlay class="sql-workbench__button-icon" />
                运行
              </el-button>
            </div>
          </div>

          <div class="sql-workbench__editor">
            <MonacoEditor
              v-model="activeTab.sql"
              :language="editorLanguage"
              :theme="isDark ? 'vs-dark' : 'vs'"
              height="100%"
              :options="editorOptions"
              @change="handleSqlChange"
            />
          </div>

          <div v-if="activeTab.parameters.length > 0" class="sql-workbench__params">
            <div v-for="(parameter, index) in activeTab.parameters" :key="parameter.name">
              <span>?{{ index + 1 }}</span>
              <el-input v-model="parameter.value" size="small" :placeholder="`参数 ${index + 1}`" />
            </div>
          </div>

          <section class="sql-workbench__result">
            <div class="sql-workbench__result-head">
              <strong>结果</strong>
              <span v-if="activeTab.result">
                {{ activeTab.result.rowCount }} 行，{{ activeTab.result.executionTime }}ms
              </span>
              <span v-else>尚未执行</span>
            </div>
            <div class="sql-workbench__result-table">
              <el-table
                v-if="activeTab.result"
                :data="activeTab.displayRows"
                size="small"
                border
                height="100%"
              >
                <el-table-column
                  v-for="column in activeTab.result.columns"
                  :key="column"
                  :prop="column"
                  :label="column"
                  min-width="140"
                  show-overflow-tooltip
                />
              </el-table>
              <div v-else class="sql-workbench__empty-result">
                执行 SQL 后在这里查看结果集、耗时和行数。
              </div>
            </div>
          </section>
        </template>

        <template v-else-if="activeTab.type === 'structure'">
          <div class="sql-workbench__structure-head">
            <div>
              <strong>{{ activeTab.table }}</strong>
              <span>表结构</span>
            </div>
            <el-button size="small" @click="openTableData({ name: activeTab.table })">
              查询数据
            </el-button>
          </div>
          <el-tabs v-model="activeTab.structureTab" class="sql-workbench__meta-tabs">
            <el-tab-pane label="字段" name="columns">
              <el-table :data="activeTab.structure.columns" size="small" border height="100%">
                <el-table-column prop="name" label="字段" min-width="160" />
                <el-table-column prop="type" label="类型" width="140" />
                <el-table-column label="可空" width="80" align="center">
                  <template #default="{ row }">
                    {{ row.nullable ? '是' : '否' }}
                  </template>
                </el-table-column>
                <el-table-column prop="defaultValue" label="默认值" width="140" />
                <el-table-column prop="comment" label="备注" min-width="180" />
              </el-table>
            </el-tab-pane>
            <el-tab-pane label="索引" name="indexes">
              <el-table :data="activeTab.structure.indexes" size="small" border height="100%">
                <el-table-column prop="name" label="索引" min-width="180" />
                <el-table-column prop="type" label="类型" width="120" />
                <el-table-column prop="method" label="方法" width="120" />
                <el-table-column label="字段" min-width="220">
                  <template #default="{ row }">
                    {{ (row.columns || []).join(', ') || '-' }}
                  </template>
                </el-table-column>
              </el-table>
            </el-tab-pane>
          </el-tabs>
        </template>
      </div>

      <div v-else class="sql-workbench__blank">
        <IconTablerFileSearch />
        <strong>打开表或保存查询</strong>
        <span>双击左侧表会生成 SELECT 查询，双击保存查询会打开 SQL 标签。</span>
      </div>
    </main>

    <aside class="sql-workbench__inspector">
      <section>
        <div class="sql-workbench__panel-title">当前上下文</div>
        <dl class="sql-workbench__facts">
          <div>
            <dt>类型</dt>
            <dd>{{ dbTypeLabel }}</dd>
          </div>
          <div>
            <dt>库</dt>
            <dd>{{ databaseLabel }}</dd>
          </div>
          <div>
            <dt>表</dt>
            <dd>{{ selectedTableName || activeTab?.table || '-' }}</dd>
          </div>
        </dl>
      </section>

      <section v-if="activeQueryDataPoint">
        <div class="sql-workbench__panel-title">自动数据点</div>
        <button
          type="button"
          class="sql-workbench__datapoint-card"
          @click="copyDataPointPath(activeQueryDataPoint.path)"
        >
          <span>{{ activeQueryDataPoint.name }}</span>
          <small>{{ activeQueryDataPoint.path }}</small>
        </button>
      </section>

      <section v-else-if="activeTab?.type === 'query'">
        <div class="sql-workbench__panel-title">自动数据点</div>
        <div class="sql-workbench__empty">保存查询后会自动生成 db.query 数据点。</div>
      </section>

      <section>
        <div class="sql-workbench__panel-title">执行历史</div>
        <div v-if="executionHistory.length === 0" class="sql-workbench__empty">暂无执行记录</div>
        <template v-else>
          <button
            v-for="record in executionHistory"
            :key="record.id"
            type="button"
            class="sql-workbench__history-item"
            @click="openHistory(record)"
          >
            <span>{{ record.title }}</span>
            <small>{{ record.rowCount }} 行 / {{ record.executionTime }}ms</small>
          </button>
        </template>
      </section>

      <section v-if="activeTab?.type === 'structure'">
        <div class="sql-workbench__panel-title">字段摘要</div>
        <div
          v-for="column in activeTab.structure.columns.slice(0, 12)"
          :key="column.name"
          class="sql-workbench__column-chip"
        >
          <span>{{ column.name }}</span>
          <small>{{ column.type }}</small>
        </div>
      </section>
    </aside>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { format as formatSql } from 'sql-formatter'
import IconTablerArrowLeft from '~icons/tabler/arrow-left'
import IconTablerChevronDown from '~icons/tabler/chevron-down'
import IconTablerChevronRight from '~icons/tabler/chevron-right'
import IconTablerColumns from '~icons/tabler/columns'
import IconTablerDatabase from '~icons/tabler/database'
import IconTablerFileSearch from '~icons/tabler/file-search'
import IconTablerFileText from '~icons/tabler/file-text'
import IconTablerLoader2 from '~icons/tabler/loader-2'
import IconTablerPlayerPlay from '~icons/tabler/player-play'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerTable from '~icons/tabler/table'
import IconTablerX from '~icons/tabler/x'
import MonacoEditor from '@/components/MonacoEditor.vue'
import dataAPI from '@/api/data.api'
import { getApiErrorMessage } from '@/utils/request'

type SqlConnection = {
  id: string
  name?: string
  type?: string
  relationalConfig?: Record<string, any>
  config?: Record<string, any>
}

const props = defineProps<{
  projectId: string | number
  connection: SqlConnection
}>()

defineEmits<{
  (event: 'back'): void
}>()

const tables = ref<any[]>([])
const queries = ref<any[]>([])
const queryDataPoints = ref<any[]>([])
const tabs = ref<any[]>([])
const activeTabId = ref('')
const filterText = ref('')
const selectedTableName = ref('')
const tablesLoading = ref(false)
const queriesLoading = ref(false)
const tablesExpanded = ref(true)
const queriesExpanded = ref(true)
const executionHistory = ref<any[]>([])
let tabCounter = 0

const dbConfig = computed(() => props.connection.relationalConfig || props.connection.config || {})
const dbType = computed(() => {
  if (props.connection.type === 'relational') {
    return dbConfig.value.dbType || 'mysql'
  }
  return props.connection.type || dbConfig.value.dbType || 'mysql'
})
const dbTypeLabel = computed(() => {
  const labels: Record<string, string> = {
    mysql: 'MySQL',
    postgresql: 'PostgreSQL',
    sqlserver: 'SQL Server',
    tdengine: 'TDengine',
  }
  return labels[dbType.value] || dbType.value
})
const databaseLabel = computed(() => dbConfig.value.database || dbConfig.value.schema || '已配置库')
const editorLanguage = computed(() => {
  if (dbType.value === 'postgresql') return 'pgsql'
  if (dbType.value === 'sqlserver') return 'sql'
  return 'mysql'
})
const formatterLanguage = computed(() => {
  if (dbType.value === 'postgresql') return 'postgresql'
  if (dbType.value === 'sqlserver') return 'transactsql'
  return 'mysql'
})
const isDark = computed(() => document.documentElement.classList.contains('dark'))
const activeTab = computed(() => tabs.value.find((tab) => tab.id === activeTabId.value) || null)

const queryDataPointBySourceId = computed(() =>
  queryDataPoints.value.reduce(
    (records, point) => {
      if (point.sourceId) records[point.sourceId] = point
      return records
    },
    {} as Record<string, any>,
  ),
)

const activeQueryDataPoint = computed(() => {
  const tab = activeTab.value
  if (!tab?.queryId) return null
  return queryDataPointBySourceId.value[tab.queryId] || null
})

const editorOptions = {
  minimap: { enabled: false },
  fontSize: 13,
  lineHeight: 20,
  wordWrap: 'on',
  formatOnPaste: true,
  suggestOnTriggerCharacters: true,
  quickSuggestions: { other: true, comments: false, strings: false },
}

const filteredTables = computed(() => {
  const keyword = filterText.value.trim().toLowerCase()
  if (!keyword) return tables.value
  return tables.value.filter((table) =>
    String(table.name || '')
      .toLowerCase()
      .includes(keyword),
  )
})

const filteredQueries = computed(() => {
  const keyword = filterText.value.trim().toLowerCase()
  if (!keyword) return queries.value
  return queries.value.filter((query) =>
    String(query.name || '')
      .toLowerCase()
      .includes(keyword),
  )
})

const quoteTable = (tableName: string) => {
  if (dbType.value === 'mysql') return `\`${tableName}\``
  if (dbType.value === 'sqlserver') return `[${tableName}]`
  return `"${tableName}"`
}

const buildSelectSql = (tableName: string) => {
  if (dbType.value === 'sqlserver') {
    return `SELECT * FROM ${quoteTable(tableName)} ORDER BY (SELECT NULL) OFFSET 0 ROWS FETCH NEXT 100 ROWS ONLY`
  }
  return `SELECT * FROM ${quoteTable(tableName)} LIMIT 100`
}

const normalizeRows = (rows: any[], columns: string[]) =>
  (rows || []).map((row) => {
    if (!Array.isArray(row)) return row
    return columns.reduce(
      (record, column, index) => {
        record[column] = row[index]
        return record
      },
      {} as Record<string, any>,
    )
  })

const extractParameters = (sql: string, oldParameters: any[] = []) => {
  const count = (sql.match(/\?/g) || []).length
  return Array.from({ length: count }, (_, index) => ({
    name: `param${index + 1}`,
    value: oldParameters[index]?.value || '',
  }))
}

const loadTables = async () => {
  tablesLoading.value = true
  try {
    const response = await dataAPI.getConnectionTables(props.projectId, props.connection.id)
    tables.value = response.data?.tables || []
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载表列表失败'))
  } finally {
    tablesLoading.value = false
  }
}

const loadQueries = async () => {
  queriesLoading.value = true
  try {
    const response = await dataAPI.getQueries(props.projectId, {
      connectionId: props.connection.id,
      queryType: 'sql',
    })
    queries.value = response.data?.queries || response.data || []
    await loadQueryDataPoints()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载保存查询失败'))
  } finally {
    queriesLoading.value = false
  }
}

const loadQueryDataPoints = async () => {
  const sourceIds = queries.value.map((query) => query.id).filter(Boolean)
  if (sourceIds.length === 0) {
    queryDataPoints.value = []
    return
  }
  try {
    const response = await dataAPI.getDataPoints(props.projectId, {
      type: 'db.query',
      sourceIds: sourceIds.join(','),
      page: 1,
      pageSize: 200,
    })
    queryDataPoints.value = response.data?.datapoints || []
  } catch {
    queryDataPoints.value = []
  }
}

const reloadExplorer = async () => {
  await Promise.all([loadTables(), loadQueries()])
}

const createQueryTab = (initial: Record<string, any> = {}) => {
  tabCounter += 1
  const id = initial.id || `query-${props.connection.id}-${tabCounter}`
  const existing = tabs.value.find((tab) => tab.id === id)
  if (existing) {
    activeTabId.value = existing.id
    return existing
  }

  const tab = {
    id,
    queryId: initial.queryId || '',
    type: 'query',
    title: initial.title || `查询 ${tabCounter}`,
    icon: IconTablerFileText,
    table: initial.table || '',
    sql: initial.sql || 'SELECT * FROM ',
    parameters: extractParameters(initial.sql || 'SELECT * FROM '),
    result: null,
    displayRows: [],
    executing: false,
    saving: false,
    modified: false,
  }
  tabs.value.push(tab)
  activeTabId.value = tab.id
  return tab
}

const openTableData = async (table: any) => {
  selectedTableName.value = table.name
  const tab = createQueryTab({
    title: table.name,
    table: table.name,
    sql: buildSelectSql(table.name),
  })
  await executeTab(tab)
}

const openSavedQuery = (query: any) => {
  createQueryTab({
    id: `saved-query-${query.id}`,
    queryId: query.id,
    title: query.name,
    sql: query.config?.sql || '',
  })
}

const openTableStructure = async (table: any) => {
  selectedTableName.value = table.name
  const id = `structure-${props.connection.id}-${table.name}`
  const existing = tabs.value.find((tab) => tab.id === id)
  if (existing) {
    activeTabId.value = id
    return
  }

  const tab = {
    id,
    type: 'structure',
    title: `${table.name} / 结构`,
    icon: IconTablerColumns,
    table: table.name,
    structureTab: 'columns',
    structure: { columns: [], indexes: [], foreignKeys: [] },
    modified: false,
  }
  tabs.value.push(tab)
  activeTabId.value = id

  try {
    const response = await dataAPI.getTableStructure(
      props.projectId,
      props.connection.id,
      table.name,
    )
    tab.structure = response.data || {
      columns: [],
      indexes: [],
      foreignKeys: [],
    }
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载表结构失败'))
  }
}

const closeTab = (tabId: string) => {
  const index = tabs.value.findIndex((tab) => tab.id === tabId)
  if (index < 0) return
  tabs.value.splice(index, 1)
  if (activeTabId.value === tabId) {
    activeTabId.value = tabs.value[index - 1]?.id || tabs.value[0]?.id || ''
  }
}

const handleSqlChange = () => {
  if (!activeTab.value || activeTab.value.type !== 'query') return
  activeTab.value.modified = true
  activeTab.value.parameters = extractParameters(activeTab.value.sql, activeTab.value.parameters)
}

const formatActiveSql = () => {
  if (!activeTab.value?.sql) return
  try {
    activeTab.value.sql = formatSql(activeTab.value.sql, {
      language: formatterLanguage.value as any,
    })
    activeTab.value.modified = true
  } catch {
    ElMessage.warning('当前 SQL 暂时无法格式化')
  }
}

const executeTab = async (tab: any) => {
  if (!tab?.sql?.trim()) {
    ElMessage.warning('请先输入 SQL')
    return
  }
  tab.executing = true
  try {
    const parameters = (tab.parameters || []).map((parameter) => parameter.value || '')
    const response = await dataAPI.executeSql(
      props.projectId,
      props.connection.id,
      tab.sql,
      parameters,
    )
    const result = {
      columns: response.data?.columns || [],
      rows: response.data?.rows || [],
      rowCount: response.data?.rowCount || response.data?.rows?.length || 0,
      executionTime: response.data?.executionTime || 0,
    }
    tab.result = result
    tab.displayRows = normalizeRows(result.rows, result.columns)
    executionHistory.value.unshift({
      id: `${Date.now()}-${Math.random().toString(36).slice(2)}`,
      title: tab.title,
      sql: tab.sql,
      rowCount: result.rowCount,
      executionTime: result.executionTime,
    })
    executionHistory.value = executionHistory.value.slice(0, 12)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '执行 SQL 失败'))
  } finally {
    tab.executing = false
  }
}

const executeActiveQuery = async () => {
  await executeTab(activeTab.value)
}

const saveActiveQuery = async () => {
  const tab = activeTab.value
  if (!tab || tab.type !== 'query') return

  if (tab.queryId) {
    tab.saving = true
    try {
      await dataAPI.updateQuery(tab.queryId, {
        name: tab.title,
        connectionId: props.connection.id,
        queryType: 'sql',
        config: { sql: tab.sql, parameters: [] },
      })
      tab.modified = false
      await loadQueries()
      ElMessage.success('查询已保存')
    } catch (error) {
      ElMessage.error(getApiErrorMessage(error, '保存查询失败'))
    } finally {
      tab.saving = false
    }
    return
  }

  try {
    const { value } = await ElMessageBox.prompt('请输入查询名称', '保存查询', {
      confirmButtonText: '保存',
      cancelButtonText: '取消',
      inputPattern: /^.{2,100}$/,
      inputErrorMessage: '名称长度需要在 2 到 100 个字符之间',
      inputValue: tab.table || tab.title,
    })

    tab.saving = true
    const response = await dataAPI.createQuery(props.projectId, {
      name: value,
      connectionId: props.connection.id,
      queryType: 'sql',
      config: { sql: tab.sql, parameters: [] },
    })
    tab.title = value
    tab.queryId = response.data?.id
    tab.modified = false
    await loadQueries()
    ElMessage.success('查询已保存')
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(getApiErrorMessage(error, '保存查询失败'))
    }
  } finally {
    tab.saving = false
  }
}

const openHistory = (record: any) => {
  createQueryTab({
    title: `${record.title} 副本`,
    sql: record.sql,
  })
}

const copyDataPointPath = async (path: string) => {
  if (!path) return
  try {
    await navigator.clipboard.writeText(path)
    ElMessage.success('数据点路径已复制')
  } catch {
    ElMessage.warning('复制失败，请手动复制路径')
  }
}

watch(
  () => props.connection.id,
  async () => {
    tabs.value = []
    activeTabId.value = ''
    selectedTableName.value = ''
    executionHistory.value = []
    await reloadExplorer()
    createQueryTab()
  },
)

onMounted(async () => {
  await reloadExplorer()
  createQueryTab()
})
</script>

<style scoped>
.sql-workbench {
  height: 100%;
  min-height: 0;
  display: grid;
  grid-template-columns: 260px minmax(0, 1fr) 260px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
  overflow: hidden;
}

.sql-workbench__explorer,
.sql-workbench__inspector {
  min-height: 0;
  overflow: hidden;
  background: var(--dc-surface-muted);
}

.sql-workbench__explorer {
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--dc-border);
}

.sql-workbench__inspector {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 14px;
  border-left: 1px solid var(--dc-border);
  overflow-y: auto;
}

.sql-workbench__source {
  padding: 14px;
  border-bottom: 1px solid var(--dc-border);
}

.sql-workbench__back {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 12px;
  padding: 6px 8px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.sql-workbench__back svg {
  width: 14px;
  height: 14px;
}

.sql-workbench__back:hover {
  border-color: color-mix(in oklch, var(--dc-primary) 28%, var(--dc-border));
  color: var(--dc-primary);
}

.sql-workbench__source span,
.sql-workbench__source small {
  display: block;
  color: var(--dc-text-muted);
  font-size: 11px;
}

.sql-workbench__source strong {
  display: block;
  margin: 6px 0 4px;
  overflow: hidden;
  color: var(--dc-text);
  font-size: 15px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sql-workbench__toolbar {
  display: flex;
  gap: 6px;
  padding: 10px 12px 6px;
}

.sql-workbench__toolbar button,
.sql-workbench__inline-action {
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}

.sql-workbench__toolbar svg,
.sql-workbench__inline-action svg {
  width: 15px;
  height: 15px;
}

.sql-workbench__toolbar button:hover,
.sql-workbench__inline-action:hover {
  color: var(--dc-primary);
  border-color: color-mix(in oklch, var(--dc-primary) 28%, var(--dc-border));
}

.sql-workbench__search {
  padding: 0 12px 10px;
}

.sql-workbench__tree {
  min-height: 0;
  flex: 1;
  overflow-y: auto;
  padding: 0 8px 12px;
}

.sql-workbench__tree-section + .sql-workbench__tree-section {
  margin-top: 10px;
}

.sql-workbench__tree-head,
.sql-workbench__tree-item,
.sql-workbench__history-item {
  width: 100%;
  display: grid;
  align-items: center;
  border: 0;
  background: transparent;
  color: var(--dc-text-secondary);
  text-align: left;
}

.sql-workbench__tree-head {
  grid-template-columns: 18px minmax(0, 1fr) auto;
  min-height: 30px;
  padding: 5px 8px;
  font-size: 12px;
  font-weight: 700;
}

.sql-workbench__tree-head svg,
.sql-workbench__tree-item svg {
  width: 15px;
  height: 15px;
}

.sql-workbench__tree-head small,
.sql-workbench__tree-item small {
  color: var(--dc-text-muted);
  font-size: 11px;
}

.sql-workbench__tree-body {
  padding-left: 8px;
}

.sql-workbench__tree-item {
  position: relative;
  grid-template-columns: 20px minmax(0, 1fr) auto 30px;
  gap: 6px;
  min-height: 30px;
  padding: 5px 4px 5px 8px;
  border-radius: var(--dc-radius-sm);
  font-size: 12px;
}

.sql-workbench__datapoint-pill {
  padding: 2px 6px;
  border: 1px solid color-mix(in oklch, var(--dc-success) 24%, transparent);
  border-radius: var(--dc-radius-xs);
  background: var(--dc-success-soft);
  color: var(--dc-success) !important;
  font-weight: 700;
}

.sql-workbench__datapoint-pill.is-missing {
  border-color: var(--dc-border);
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted) !important;
}

.sql-workbench__tree-item span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sql-workbench__tree-item:hover,
.sql-workbench__tree-item.is-active,
.sql-workbench__history-item:hover {
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.sql-workbench__inline-action {
  opacity: 0;
}

.sql-workbench__tree-item:hover .sql-workbench__inline-action {
  opacity: 1;
}

.sql-workbench__main {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.sql-workbench__tabbar {
  display: flex;
  min-height: 38px;
  overflow-x: auto;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
}

.sql-workbench__tab {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  min-width: 120px;
  max-width: 220px;
  padding: 0 10px;
  border: 0;
  border-right: 1px solid var(--dc-border);
  background: transparent;
  color: var(--dc-text-secondary);
  font-size: 12px;
}

.sql-workbench__tab.is-active {
  background: var(--dc-surface-raised);
  color: var(--dc-primary);
  font-weight: 700;
}

.sql-workbench__tab span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sql-workbench__tab svg {
  width: 15px;
  height: 15px;
  flex-shrink: 0;
}

.sql-workbench__tab i {
  width: 6px;
  height: 6px;
  flex-shrink: 0;
  border-radius: 50%;
  background: var(--dc-warning);
}

.sql-workbench__tab-close {
  margin-left: auto;
  color: var(--dc-text-muted);
}

.sql-workbench__content {
  min-height: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.sql-workbench__editor-toolbar,
.sql-workbench__structure-head,
.sql-workbench__result-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  border-bottom: 1px solid var(--dc-border);
}

.sql-workbench__editor-toolbar,
.sql-workbench__structure-head {
  min-height: 45px;
  padding: 8px 12px;
}

.sql-workbench__context-line {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  color: var(--dc-text-secondary);
  font-size: 12px;
}

.sql-workbench__context-line svg,
.sql-workbench__button-icon {
  width: 15px;
  height: 15px;
}

.sql-workbench__actions {
  display: flex;
  flex-shrink: 0;
  gap: 6px;
}

.sql-workbench__editor {
  height: 270px;
  flex-shrink: 0;
  border-bottom: 1px solid var(--dc-border);
}

.sql-workbench__params {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px 12px;
  padding: 10px 12px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-muted);
}

.sql-workbench__params div {
  display: grid;
  grid-template-columns: 34px minmax(0, 1fr);
  align-items: center;
  gap: 8px;
}

.sql-workbench__params span {
  color: var(--dc-text-muted);
  font-size: 12px;
}

.sql-workbench__result {
  min-height: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.sql-workbench__result-head {
  min-height: 38px;
  padding: 8px 12px;
  color: var(--dc-text-secondary);
  font-size: 12px;
}

.sql-workbench__result-head strong,
.sql-workbench__structure-head strong {
  color: var(--dc-text);
  font-size: 13px;
}

.sql-workbench__result-table {
  min-height: 0;
  flex: 1;
}

.sql-workbench__empty-result,
.sql-workbench__blank {
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--dc-text-secondary);
  font-size: 12px;
}

.sql-workbench__blank svg {
  width: 42px;
  height: 42px;
  color: var(--dc-text-muted);
}

.sql-workbench__meta-tabs {
  min-height: 0;
  flex: 1;
  padding: 0 12px 12px;
}

.sql-workbench__meta-tabs :deep(.el-tabs__content) {
  height: calc(100% - 40px);
}

.sql-workbench__meta-tabs :deep(.el-tab-pane) {
  height: 100%;
}

.sql-workbench__panel-title {
  margin-bottom: 8px;
  color: var(--dc-text);
  font-size: 12px;
  font-weight: 700;
}

.sql-workbench__facts {
  margin: 0;
}

.sql-workbench__facts div {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 0;
  border-bottom: 1px solid var(--dc-border);
}

.sql-workbench__facts dt {
  color: var(--dc-text-muted);
  font-size: 11px;
}

.sql-workbench__facts dd {
  min-width: 0;
  margin: 0;
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
  text-align: right;
  word-break: break-all;
}

.sql-workbench__history-item {
  grid-template-columns: minmax(0, 1fr);
  gap: 2px;
  min-height: 38px;
  margin-bottom: 6px;
  padding: 8px;
  border-radius: var(--dc-radius-sm);
}

.sql-workbench__datapoint-card {
  width: 100%;
  display: grid;
  gap: 4px;
  padding: 10px;
  border: 1px solid color-mix(in oklch, var(--dc-primary) 22%, var(--dc-border));
  border-radius: var(--dc-radius-sm);
  background: var(--dc-primary-soft);
  color: var(--dc-text);
  text-align: left;
}

.sql-workbench__datapoint-card span {
  overflow: hidden;
  font-size: 12px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sql-workbench__datapoint-card small {
  overflow: hidden;
  color: var(--dc-primary);
  font-family: var(--dc-font-mono);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sql-workbench__history-item span,
.sql-workbench__history-item small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sql-workbench__history-item span {
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.sql-workbench__history-item small,
.sql-workbench__empty,
.sql-workbench__loading {
  color: var(--dc-text-muted);
  font-size: 11px;
}

.sql-workbench__loading {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px;
}

.sql-workbench__loading svg {
  width: 14px;
  height: 14px;
  animation: sql-workbench-spin 0.9s linear infinite;
}

.sql-workbench__column-chip {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 7px 0;
  border-bottom: 1px solid var(--dc-border);
}

.sql-workbench__column-chip span {
  min-width: 0;
  overflow: hidden;
  color: var(--dc-text-secondary);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sql-workbench__column-chip small {
  color: var(--dc-text-muted);
  font-size: 11px;
}

:deep(.el-input__wrapper),
:deep(.el-table),
:deep(.el-button) {
  border-radius: var(--dc-radius-sm);
}

:deep(.el-table th.el-table__cell) {
  background: var(--dc-surface-muted);
  color: var(--dc-text-secondary);
  font-weight: 700;
}

@keyframes sql-workbench-spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 1100px) {
  .sql-workbench {
    grid-template-columns: 240px minmax(0, 1fr);
  }

  .sql-workbench__inspector {
    display: none;
  }
}

@media (max-width: 760px) {
  .sql-workbench {
    grid-template-columns: 1fr;
  }

  .sql-workbench__explorer {
    max-height: 320px;
    border-right: 0;
    border-bottom: 1px solid var(--dc-border);
  }
}
</style>
