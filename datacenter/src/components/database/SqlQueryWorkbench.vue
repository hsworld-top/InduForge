<template>
  <section class="sql-workbench">
    <aside class="sql-workbench__explorer">
      <WorkbenchSourceHeader
        :title="connection.name || '未命名接入源'"
        fallback-title="未命名接入源"
        :meta="sourceMetaRows"
        @back="$emit('back')"
      >
        <template #actions>
          <el-input
            v-model="filterText"
            class="sql-workbench__search"
            size="small"
            placeholder="筛选表或查询"
            clearable
          />
          <button
            type="button"
            class="workbench-source-header__icon-action is-primary"
            title="新建查询"
            aria-label="新建查询"
            @click="createQueryTab()"
          >
            <IconTablerPlus />
          </button>
          <button
            type="button"
            class="workbench-source-header__icon-action"
            title="刷新对象"
            aria-label="刷新对象"
            @click="reloadExplorer"
          >
            <IconTablerRefresh />
          </button>
        </template>
      </WorkbenchSourceHeader>

      <div class="sql-workbench__tree">
        <section class="sql-workbench__tree-section">
          <button
            type="button"
            class="sql-workbench__tree-category"
            :class="{ 'is-active': selectedExplorerNode === 'queries' }"
            @click="toggleExplorerCategory('queries')"
            @contextmenu.prevent.stop="openQueryCategoryMenu"
          >
            <IconTablerChevronDown v-if="queriesExpanded" />
            <IconTablerChevronRight v-else />
            <IconTablerFileText class="sql-workbench__category-icon" />
            <span>查询</span>
            <small>{{ filteredQueries.length }}</small>
          </button>

          <div v-if="queriesExpanded" class="sql-workbench__tree-body">
            <div v-if="queriesLoading" class="sql-workbench__loading">
              <IconTablerLoader2 />
              <span>加载查询...</span>
            </div>
            <template v-else>
              <div
                v-for="group in queryTreeGroups"
                :key="group.id"
                class="sql-workbench__tree-group-block"
              >
                <button
                  type="button"
                  class="sql-workbench__tree-group"
                  :class="{ 'is-active': selectedExplorerNode === `query-group:${group.id}` }"
                  @click="toggleObjectGroup('query', group.id)"
                  @contextmenu.prevent.stop="openObjectGroupMenu($event, 'query', group)"
                >
                  <IconTablerChevronDown v-if="isObjectGroupExpanded('query', group.id)" />
                  <IconTablerChevronRight v-else />
                  <span>{{ group.name }}</span>
                  <small>{{ group.items.length }}</small>
                </button>
                <div
                  v-if="isObjectGroupExpanded('query', group.id)"
                  class="sql-workbench__tree-group-body"
                >
                  <button
                    v-for="query in group.items"
                    :key="query.id"
                    type="button"
                    class="sql-workbench__tree-item"
                    :class="{ 'is-active': selectedExplorerNode === `query:${query.id}` }"
                    @click="selectedExplorerNode = `query:${query.id}`"
                    @dblclick="openSavedQuery(query)"
                    @contextmenu.prevent.stop="openQueryMenu($event, query)"
                  >
                    <IconTablerFileText />
                    <span>{{ query.name }}</span>
                    <small class="sql-workbench__datapoint-pill"> 数据点 </small>
                  </button>
                  <div v-if="group.items.length === 0" class="sql-workbench__empty is-compact">
                    暂无查询
                  </div>
                </div>
              </div>
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
            class="sql-workbench__tree-category"
            :class="{ 'is-active': selectedExplorerNode === 'tables' }"
            @click="toggleExplorerCategory('tables')"
            @contextmenu.prevent.stop="openTableCategoryMenu"
          >
            <IconTablerChevronDown v-if="tablesExpanded" />
            <IconTablerChevronRight v-else />
            <IconTablerTable class="sql-workbench__category-icon" />
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
                v-for="group in tableTreeGroups"
                :key="group.id"
                class="sql-workbench__tree-group-block"
              >
                <button
                  type="button"
                  class="sql-workbench__tree-group"
                  :class="{ 'is-active': selectedExplorerNode === `table-group:${group.id}` }"
                  @click="toggleObjectGroup('table', group.id)"
                  @contextmenu.prevent.stop="openObjectGroupMenu($event, 'table', group)"
                >
                  <IconTablerChevronDown v-if="isObjectGroupExpanded('table', group.id)" />
                  <IconTablerChevronRight v-else />
                  <span>{{ group.name }}</span>
                  <small>{{ group.items.length }}</small>
                </button>
                <div
                  v-if="isObjectGroupExpanded('table', group.id)"
                  class="sql-workbench__tree-group-body"
                >
                  <div
                    v-for="table in group.items"
                    :key="table.name"
                    role="button"
                    tabindex="0"
                    class="sql-workbench__tree-item"
                    :class="{ 'is-active': selectedExplorerNode === `table:${table.name}` }"
                    @click="selectTableNode(table)"
                    @dblclick="openTableData(table)"
                    @keydown.enter="openTableData(table)"
                    @contextmenu.prevent.stop="openTableMenu($event, table)"
                  >
                    <component :is="resolveTableIcon(table)" />
                    <span>{{ table.name }}</span>
                    <small v-if="table.kind === 'super_table'" class="sql-workbench__kind-pill">
                      超表
                    </small>
                    <small v-if="table.rows !== undefined">{{ table.rows }}</small>
                  </div>
                  <div v-if="group.items.length === 0" class="sql-workbench__empty is-compact">
                    暂无表
                  </div>
                </div>
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
          <IconTablerX class="sql-workbench__tab-close" @click.stop="closeTab(tab.id)" />
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

          <div class="sql-workbench__editor" :style="{ height: `${activeTab.editorHeight}px` }">
            <MonacoEditor
              ref="sqlEditorRef"
              v-model="activeTab.sql"
              :language="editorLanguage"
              :theme="isDark ? 'vs-dark' : 'vs'"
              height="100%"
              :options="editorOptions"
              @change="handleSqlChange"
            />
          </div>

          <div class="sql-workbench__params">
            <div class="sql-workbench__params-head">
              <div>
                <strong>模板参数</strong>
                <span v-if="activeTab.parameters.length > 0">
                  执行前按 ? 出现顺序任意替换；类型会随查询保存，测试值不会保存。
                </span>
                <span v-else>在 SQL 中插入 ? 后，可在这里填写本次执行使用的替换内容。</span>
              </div>
              <el-button size="small" @click="insertSqlParameter">插入参数</el-button>
            </div>
            <div v-if="activeTab.parameters.length > 0" class="sql-workbench__param-list">
              <label v-for="(parameter, index) in activeTab.parameters" :key="parameter.name">
                <span>参数 {{ index + 1 }}</span>
                <el-select v-model="parameter.type" size="small" placeholder="类型">
                  <el-option
                    v-for="option in sqlParameterTypeOptions"
                    :key="option.value"
                    :label="option.label"
                    :value="option.value"
                  />
                </el-select>
                <el-input
                  v-model="parameter.value"
                  size="small"
                  :placeholder="`对应第 ${index + 1} 个 ? 的测试值`"
                />
              </label>
            </div>
            <div v-else class="sql-workbench__param-empty">
              示例：SELECT * FROM ? WHERE id = ?
            </div>
          </div>

          <section class="sql-workbench__result">
            <div
              class="sql-workbench__result-resizer"
              role="separator"
              aria-orientation="horizontal"
              title="拖拽调整编辑器和结果区高度"
              @mousedown.prevent="startResultResize($event, activeTab)"
              @dblclick="resetEditorHeight(activeTab)"
            />
            <div class="sql-workbench__result-head">
              <strong>结果</strong>
              <span v-if="activeTab.result">
                {{
                  activeTab.result.columns.length > 0
                    ? `${activeTab.result.rowCount} 行，${activeTab.result.executionTime}ms`
                    : `命令执行成功，${activeTab.result.executionTime}ms`
                }}
              </span>
              <span v-else>尚未执行</span>
            </div>
            <div class="sql-workbench__result-table">
              <template v-if="activeTab.result && activeTab.result.columns.length > 0">
                <el-table
                  :data="activePagedRows"
                  class="sql-workbench__result-data"
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
                <div class="sql-workbench__result-pagination">
                  <el-pagination
                    v-model:current-page="activeTab.resultPage"
                    v-model:page-size="activeTab.resultPageSize"
                    :page-sizes="resultPageSizes"
                    :total="activeResultTotal"
                    :pager-count="5"
                    size="small"
                    layout="total, sizes, prev, pager, next, jumper"
                    @size-change="handleResultPageSizeChange(activeTab)"
                  />
                </div>
              </template>
              <div v-else-if="activeTab.result" class="sql-workbench__empty-result">
                命令执行成功，未返回结果集。
              </div>
              <div v-else class="sql-workbench__empty-result">
                执行 SQL 后在这里查看结果集、耗时和行数。
              </div>
            </div>
          </section>
        </template>

        <template v-else-if="activeTab.type === 'structure'">
          <div class="sql-workbench__structure">
            <div class="sql-workbench__structure-head">
              <div>
                <strong>{{ activeTab.table }}</strong>
                <span>表结构</span>
              </div>
              <el-button size="small" @click="openTableData({ name: activeTab.table })">
                查询数据
              </el-button>
            </div>
            <div v-if="activeTab.loading" class="sql-workbench__loading is-structure">
              <IconTablerLoader2 />
              <span>加载表结构...</span>
            </div>
            <el-tabs v-else v-model="activeTab.structureTab" class="sql-workbench__meta-tabs">
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
          </div>
        </template>
      </div>

      <div v-else class="sql-workbench__blank">
        <IconTablerFileSearch />
        <strong>打开表或保存查询</strong>
        <span>双击左侧表会生成 SELECT 查询，双击保存查询会打开 SQL 标签。</span>
      </div>
    </main>

    <aside class="sql-workbench__inspector">
      <div class="sql-workbench__inspector-main">
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
      </div>

      <section class="sql-workbench__history-panel">
        <div class="sql-workbench__panel-title">执行历史</div>
        <div v-if="executionHistory.length === 0" class="sql-workbench__empty">暂无执行记录</div>
        <div v-else class="sql-workbench__history-list">
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
        </div>
      </section>
    </aside>

    <TableDesignDialog
      v-model="tableDesignVisible"
      :project-id="projectId"
      :connection-id="connection.id"
      :db-type="dbType"
      :supports-super-table="supportsSuperTable"
      :mode="tableDesignMode"
      :table-name="editingTableName"
      :initial-structure="editingTableStructure"
      @created="handleTableCreated"
      @updated="handleTableUpdated"
    />

    <WorkbenchMoveGroupDialog
      ref="moveGroupDialogRef"
      v-model="moveGroupDialogVisible"
      :groups="moveGroupOptions"
      :current-group-id="moveGroupCurrentId"
      :loading="moveGroupSubmitting"
      @submit="submitMoveGroup"
    />

    <Teleport to="body">
      <div
        v-if="contextMenu.visible"
        class="sql-workbench__menu-mask"
        @click="closeContextMenu"
        @contextmenu.prevent="closeContextMenu"
      >
        <div
          class="sql-workbench__context-menu"
          :style="{ left: `${contextMenu.x}px`, top: `${contextMenu.y}px` }"
          @click.stop
        >
          <button
            v-if="contextMenu.type === 'query-category'"
            type="button"
            @click="openContextNewQuery"
          >
            <IconTablerPlus />
            <span>新建查询</span>
          </button>
          <button
            v-if="contextMenu.type === 'query-category'"
            type="button"
            @click="refreshContextQueries"
          >
            <IconTablerRefresh />
            <span>刷新查询</span>
          </button>
          <button v-if="contextMenu.type === 'query'" type="button" @click="openContextQuery">
            <IconTablerFileText />
            <span>打开查询</span>
          </button>
          <button v-if="contextMenu.type === 'query'" type="button" @click="renameContextQuery">
            <IconTablerEdit />
            <span>重命名查询</span>
          </button>
          <button v-if="contextMenu.type === 'query'" type="button" @click="moveContextQuery">
            <IconTablerFolder />
            <span>移动到分组</span>
          </button>
          <button v-if="contextMenu.type === 'query'" type="button" @click="deleteContextQuery">
            <IconTablerTrash />
            <span>删除查询</span>
          </button>
          <button
            v-if="contextMenu.type === 'table-category'"
            type="button"
            @click="openTableDesign"
          >
            <IconTablerPlus />
            <span>新建表</span>
          </button>
          <button
            v-if="contextMenu.type === 'table-category'"
            type="button"
            @click="refreshContextTables"
          >
            <IconTablerRefresh />
            <span>刷新表</span>
          </button>
          <button
            v-if="contextMenu.type === 'query-group' || contextMenu.type === 'table-group'"
            type="button"
            @click="createContextGroup"
          >
            <IconTablerPlus />
            <span>新建分组</span>
          </button>
          <button
            v-if="
              (contextMenu.type === 'query-group' || contextMenu.type === 'table-group') &&
              !contextMenu.group?.virtual
            "
            type="button"
            @click="renameContextGroup"
          >
            <IconTablerEdit />
            <span>重命名分组</span>
          </button>
          <button
            v-if="
              (contextMenu.type === 'query-group' || contextMenu.type === 'table-group') &&
              !contextMenu.group?.virtual
            "
            type="button"
            @click="deleteContextGroup"
          >
            <IconTablerTrash />
            <span>删除分组</span>
          </button>
          <button v-if="contextMenu.type === 'table'" type="button" @click="openContextStructure">
            <IconTablerColumns />
            <span>查看表结构</span>
          </button>
          <button
            v-if="contextMenu.type === 'table' && supportsTableStructureEdit"
            type="button"
            @click="editContextTableStructure"
          >
            <IconTablerEdit />
            <span>修改表结构</span>
          </button>
          <button v-if="contextMenu.type === 'table'" type="button" @click="openContextData">
            <IconTablerTable />
            <span>查询数据</span>
          </button>
          <button v-if="contextMenu.type === 'table'" type="button" @click="openContextInsert">
            <IconTablerPlus />
            <span>插入数据</span>
          </button>
          <button v-if="contextMenu.type === 'table'" type="button" @click="renameContextTable">
            <IconTablerEdit />
            <span>重命名表</span>
          </button>
          <button v-if="contextMenu.type === 'table'" type="button" @click="moveContextTable">
            <IconTablerFolder />
            <span>移动到分组</span>
          </button>
          <button v-if="contextMenu.type === 'table'" type="button" @click="deleteContextTable">
            <IconTablerTrash />
            <span>删除表</span>
          </button>
        </div>
      </div>
    </Teleport>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { format as formatSql } from 'sql-formatter'
import IconTablerChevronDown from '~icons/tabler/chevron-down'
import IconTablerChevronRight from '~icons/tabler/chevron-right'
import IconTablerColumns from '~icons/tabler/columns'
import IconTablerDatabase from '~icons/tabler/database'
import IconTablerEdit from '~icons/tabler/edit'
import IconTablerFileSearch from '~icons/tabler/file-search'
import IconTablerFileText from '~icons/tabler/file-text'
import IconTablerFolder from '~icons/tabler/folder'
import IconTablerLoader2 from '~icons/tabler/loader-2'
import IconTablerPlayerPlay from '~icons/tabler/player-play'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerStack2 from '~icons/tabler/stack-2'
import IconTablerTable from '~icons/tabler/table'
import IconTablerTrash from '~icons/tabler/trash'
import IconTablerX from '~icons/tabler/x'
import MonacoEditor from '@/components/MonacoEditor.vue'
import dataAPI from '@/api/data.api'
import { getApiErrorMessage } from '@/utils/request'
import WorkbenchSourceHeader from '@/components/workbench/WorkbenchSourceHeader.vue'
import TableDesignDialog from './TableDesignDialog.vue'
import WorkbenchMoveGroupDialog from './WorkbenchMoveGroupDialog.vue'

type SqlConnection = {
  id: string
  name?: string
  type?: string
  relationalConfig?: Record<string, any>
  config?: Record<string, any>
}

type WorkbenchScope = 'query' | 'table'
type WorkbenchGroup = {
  id: string
  name: string
  scope: WorkbenchScope
  virtual?: boolean
  items?: any[]
}
type SqlTemplateParameterType = 'text' | 'table' | 'column' | 'condition' | 'order' | 'value'
type SqlTemplateParameter = {
  name: string
  type: SqlTemplateParameterType
  value: string
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
const queryGroups = ref<WorkbenchGroup[]>([])
const tableGroups = ref<WorkbenchGroup[]>([])
const tableGroupMembers = ref<any[]>([])
const queryDataPoints = ref<any[]>([])
const tabs = ref<any[]>([])
const sqlEditorRef = ref<InstanceType<typeof MonacoEditor> | null>(null)
const activeTabId = ref('')
const filterText = ref('')
const selectedTableName = ref('')
const selectedExplorerNode = ref('')
const tablesLoading = ref(false)
const queriesLoading = ref(false)
const tablesExpanded = ref(true)
const queriesExpanded = ref(true)
const expandedObjectGroups = ref<Record<string, boolean>>({})
const executionHistory = ref<any[]>([])
const tableDesignVisible = ref(false)
const tableDesignMode = ref<'create' | 'edit'>('create')
const editingTableName = ref('')
const editingTableStructure = ref<Record<string, any> | null>(null)
const moveGroupDialogVisible = ref(false)
const moveGroupSubmitting = ref(false)
const moveGroupTarget = ref<{ scope: WorkbenchScope; item: any | null }>({ scope: 'query', item: null })
const moveGroupDialogRef = ref<InstanceType<typeof WorkbenchMoveGroupDialog> | null>(null)
const contextMenu = ref<{
  visible: boolean
  type:
    | 'query-category'
    | 'query-group'
    | 'query'
    | 'table-category'
    | 'table-group'
    | 'table'
    | null
  x: number
  y: number
  table: any | null
  query: any | null
  group: WorkbenchGroup | null
}>({
  visible: false,
  type: null,
  x: 0,
  y: 0,
  table: null,
  query: null,
  group: null,
})
let tabCounter = 0
const defaultEditorHeight = 270
const minEditorHeight = 120
const maxEditorHeight = 560
const resultPageSizes = [50, 100, 200, 500]
const historyLimit = 50
const historyStoragePrefix = 'datacenter:sql-workbench:history'
const sqlParameterTypeOptions: Array<{ label: string; value: SqlTemplateParameterType }> = [
  { label: 'SQL文本', value: 'text' },
  { label: '表名', value: 'table' },
  { label: '字段名', value: 'column' },
  { label: '条件片段', value: 'condition' },
  { label: '排序片段', value: 'order' },
  { label: '值', value: 'value' },
]
let resultResizeState: {
  tab: any
  startY: number
  startHeight: number
} | null = null

const dbConfig = computed(() => props.connection.relationalConfig || props.connection.config || {})
const dbType = computed(() => {
  if (props.connection.type === 'relational') {
    return dbConfig.value.dbType || 'mysql'
  }
  return props.connection.type || dbConfig.value.dbType || 'mysql'
})
const supportsSuperTable = computed(() =>
  ['builtin.timeseries', 'tdengine'].includes(props.connection.type || dbType.value),
)
const supportsTableStructureEdit = computed(() =>
  ['builtin.relation', 'builtin.timeseries'].includes(props.connection.type || ''),
)
const dbTypeLabel = computed(() => {
  if (props.connection.type === 'builtin.relation') return 'IF关系库'
  if (props.connection.type === 'builtin.timeseries') return 'IF时序库'
  const labels: Record<string, string> = {
    mysql: 'MySQL',
    postgresql: 'PostgreSQL',
    sqlserver: 'SQL Server',
    tdengine: 'TDengine',
  }
  return labels[dbType.value] || dbType.value
})
const databaseLabel = computed(() => {
  if (props.connection.type === 'builtin.relation') return '工程内置关系库'
  if (props.connection.type === 'builtin.timeseries') return '工程内置时序库'
  return dbConfig.value.database || dbConfig.value.schema || '已配置库'
})
const sourceMetaRows = computed(() => [
  { label: '类型', value: dbTypeLabel.value },
  {
    label: props.connection.type?.startsWith('builtin.') ? '标识' : '库',
    value:
      props.connection.type?.startsWith('builtin.') && dbConfig.value.schema
        ? String(dbConfig.value.schema)
        : databaseLabel.value,
  },
])
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
const activeResultTotal = computed(() => activeTab.value?.displayRows?.length || 0)
const activePagedRows = computed(() => {
  const tab = activeTab.value
  if (!tab?.displayRows?.length) return []
  const pageSize = Number(tab.resultPageSize || resultPageSizes[1])
  const currentPage = Number(tab.resultPage || 1)
  const start = (currentPage - 1) * pageSize
  return tab.displayRows.slice(start, start + pageSize)
})

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

const tableGroupByName = computed(() =>
  tableGroupMembers.value.reduce(
    (records, member) => {
      if (member.tableName && member.groupId) records[member.tableName] = member.groupId
      return records
    },
    {} as Record<string, string>,
  ),
)

const buildTreeGroups = (scope: WorkbenchScope, groups: WorkbenchGroup[], items: any[]) => {
  const ungrouped: WorkbenchGroup = {
    id: '__ungrouped__',
    name: '未分组',
    scope,
    virtual: true,
    items: [],
  }
  const groupMap = new Map<string, WorkbenchGroup>()
  const result = [
    ungrouped,
    ...groups.map((group) => {
      const next = { ...group, items: [] as any[] }
      groupMap.set(next.id, next)
      return next
    }),
  ]

  items.forEach((item) => {
    const groupID = scope === 'query' ? item.groupId : tableGroupByName.value[item.name]
    const target = groupID ? groupMap.get(groupID) : null
    ;(target || ungrouped).items?.push(item)
  })
  return result
}

const queryTreeGroups = computed(() =>
  buildTreeGroups('query', queryGroups.value, filteredQueries.value),
)
const tableTreeGroups = computed(() =>
  buildTreeGroups('table', tableGroups.value, filteredTables.value),
)

const resolveTableIcon = (table: any) => {
  if (table?.kind === 'super_table') return IconTablerStack2
  return IconTablerTable
}

const quoteTable = (tableName: string) => {
  if (dbType.value === 'mysql') return `\`${tableName}\``
  if (dbType.value === 'sqlserver') return `[${tableName}]`
  return `"${tableName}"`
}

const quoteColumn = (columnName: string) => quoteTable(columnName)

const buildSelectSql = (tableName: string) => {
  if (dbType.value === 'sqlserver') {
    return `SELECT * FROM ${quoteTable(tableName)} ORDER BY (SELECT NULL) OFFSET 0 ROWS FETCH NEXT 100 ROWS ONLY`
  }
  return `SELECT * FROM ${quoteTable(tableName)} LIMIT 100`
}

// 仅生成便于用户二次编辑的 INSERT 模板，不自动执行；自增字段默认排除，避免误导用户填写数据库生成值。
const columnPlaceholderValue = (column: Record<string, any>) => {
  const type = String(column.type || '').toLowerCase()
  const isNumberType =
    type.includes('int') ||
    type.includes('double') ||
    type.includes('numeric') ||
    type.includes('decimal') ||
    type.includes('float')
  const isDateTimeType =
    type.includes('timestamp') ||
    type.includes('date') ||
    column.name === 'create_time' ||
    column.name === 'created_at'

  if (isNumberType) {
    return '0'
  }
  if (type.includes('bool')) return 'false'
  if (isDateTimeType) {
    return dbType.value === 'mysql' ? 'NOW()' : 'CURRENT_TIMESTAMP'
  }
  if (type.includes('json')) return `'{}'`
  return `'<${column.name}>'`
}

const buildInsertSql = (tableName: string, columns: Array<Record<string, any>>) => {
  const writableColumns = columns.filter((column) => !column.autoIncrement)
  const targetColumns = writableColumns.length > 0 ? writableColumns : columns
  const columnList = targetColumns.map((column) => quoteColumn(column.name)).join(', ')
  const valueList = targetColumns.map((column) => columnPlaceholderValue(column)).join(', ')
  return `INSERT INTO ${quoteTable(tableName)} (${columnList})\nVALUES (${valueList});`
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

const normalizeParameterType = (type: any): SqlTemplateParameterType => {
  const value = String(type || '')
  return sqlParameterTypeOptions.some((option) => option.value === value)
    ? (value as SqlTemplateParameterType)
    : 'text'
}

const extractParameters = (sql: string, oldParameters: any[] = []): SqlTemplateParameter[] => {
  const count = (sql.match(/\?/g) || []).length
  return Array.from({ length: count }, (_, index) => ({
    name: `param${index + 1}`,
    type: normalizeParameterType(oldParameters[index]?.type),
    value: oldParameters[index]?.value || '',
  }))
}

// SQL 工作台的 ? 是执行前模板替换，允许用户替换表名、字段名和排序片段；后端仍只接收最终 SQL。
const buildSqlWithWorkbenchParameters = (sql: string, parameters: SqlTemplateParameter[]) => {
  let index = 0
  return sql.replace(/\?/g, () => String(parameters[index++]?.value ?? ''))
}

const buildSavedParameterDefinitions = (parameters: SqlTemplateParameter[] = []) =>
  parameters.map((parameter, index) => ({
    name: parameter.name || `param${index + 1}`,
    type: normalizeParameterType(parameter.type),
    index: index + 1,
  }))

const clampEditorHeight = (height: number) =>
  Math.min(maxEditorHeight, Math.max(minEditorHeight, Math.round(height)))

// 拖拽结果区分隔线时只调整当前查询 Tab 的编辑器高度，结果区由 flex 自动占用剩余空间。
const stopResultResize = () => {
  if (!resultResizeState) return
  resultResizeState = null
  document.body.classList.remove('sql-workbench--resizing-result')
  window.removeEventListener('mousemove', handleResultResize)
  window.removeEventListener('mouseup', stopResultResize)
}

const handleResultResize = (event: MouseEvent) => {
  if (!resultResizeState) return
  const nextHeight = resultResizeState.startHeight + (event.clientY - resultResizeState.startY)
  resultResizeState.tab.editorHeight = clampEditorHeight(nextHeight)
}

const startResultResize = (event: MouseEvent, tab: any) => {
  resultResizeState = {
    tab,
    startY: event.clientY,
    startHeight: Number(tab.editorHeight || defaultEditorHeight),
  }
  document.body.classList.add('sql-workbench--resizing-result')
  window.addEventListener('mousemove', handleResultResize)
  window.addEventListener('mouseup', stopResultResize)
}

const resetEditorHeight = (tab: any) => {
  tab.editorHeight = defaultEditorHeight
}

const handleResultPageSizeChange = (tab: any) => {
  tab.resultPage = 1
}

const executionHistoryStorageKey = () =>
  `${historyStoragePrefix}:${props.projectId}:${props.connection.id}`

const normalizeHistoryRecords = (records: any[]) =>
  (Array.isArray(records) ? records : [])
    .filter((record) => record && record.id && record.sql)
    .slice(0, historyLimit)

const loadExecutionHistory = () => {
  try {
    const raw = localStorage.getItem(executionHistoryStorageKey())
    executionHistory.value = raw ? normalizeHistoryRecords(JSON.parse(raw)) : []
  } catch {
    executionHistory.value = []
  }
}

const persistExecutionHistory = () => {
  try {
    localStorage.setItem(
      executionHistoryStorageKey(),
      JSON.stringify(normalizeHistoryRecords(executionHistory.value)),
    )
  } catch {
    // 历史记录只是辅助信息，本地存储失败不影响 SQL 执行主流程。
  }
}

const pushExecutionHistory = (record: Record<string, any>) => {
  executionHistory.value = normalizeHistoryRecords([record, ...executionHistory.value])
  persistExecutionHistory()
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

const loadWorkbenchGroups = async () => {
  try {
    const [queryResponse, tableResponse, memberResponse] = await Promise.all([
      dataAPI.getWorkbenchGroups(props.projectId, props.connection.id, 'query'),
      dataAPI.getWorkbenchGroups(props.projectId, props.connection.id, 'table'),
      dataAPI.getTableGroupMembers(props.projectId, props.connection.id),
    ])
    queryGroups.value = queryResponse.data?.groups || []
    tableGroups.value = tableResponse.data?.groups || []
    tableGroupMembers.value = memberResponse.data?.members || []
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载工作台分组失败'))
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
  await Promise.all([loadTables(), loadQueries(), loadWorkbenchGroups()])
}

const focusEditorEnd = async (retry = 0) => {
  await nextTick()
  if (sqlEditorRef.value?.revealEnd?.()) return
  if (retry >= 20) return
  window.setTimeout(() => {
    void focusEditorEnd(retry + 1)
  }, 80)
}

const createQueryTab = (initial: Record<string, any> = {}) => {
  tabCounter += 1
  const id = initial.id || `query-${props.connection.id}-${tabCounter}`
  const existing = tabs.value.find((tab) => tab.id === id)
  if (existing) {
    activeTabId.value = existing.id
    void focusEditorEnd()
    return existing
  }
  const initialSql = initial.sql || 'SELECT * FROM '

  const tab = {
    id,
    queryId: initial.queryId || '',
    type: 'query',
    title: initial.title || `查询 ${tabCounter}`,
    icon: IconTablerFileText,
    table: initial.table || '',
    sql: initialSql,
    parameters: extractParameters(initialSql, initial.parameters || []),
    result: null,
    displayRows: [],
    resultPage: 1,
    resultPageSize: 100,
    editorHeight: defaultEditorHeight,
    executing: false,
    saving: false,
    modified: false,
  }
  tabs.value.push(tab)
  activeTabId.value = tab.id
  void focusEditorEnd()
  return tab
}

const openTableData = async (table: any) => {
  selectedTableName.value = table.name
  selectedExplorerNode.value = `table:${table.name}`
  const tab = createQueryTab({
    title: table.name,
    table: table.name,
    sql: buildSelectSql(table.name),
  })
  await executeTab(tab)
}

const openSavedQuery = (query: any) => {
  selectedExplorerNode.value = `query:${query.id}`
  createQueryTab({
    id: `saved-query-${query.id}`,
    queryId: query.id,
    title: query.name,
    sql: query.config?.sql || '',
    parameters: query.config?.parameters || [],
  })
}

const selectTableNode = (table: any) => {
  selectedExplorerNode.value = `table:${table.name}`
  selectedTableName.value = table.name
}

const openTableStructure = async (table: any) => {
  selectedTableName.value = table.name
  selectedExplorerNode.value = `table:${table.name}`
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
    loading: true,
    modified: false,
  }
  tabs.value.push(tab)
  activeTabId.value = id

  const patchTab = (patch: Record<string, any>) => {
    tabs.value = tabs.value.map((item) => (item.id === id ? { ...item, ...patch } : item))
  }

  try {
    const response = await dataAPI.getTableStructure(
      props.projectId,
      props.connection.id,
      table.name,
    )
    patchTab({
      structure: response.data || {
        columns: [],
        indexes: [],
        foreignKeys: [],
      },
      loading: false,
    })
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载表结构失败'))
    patchTab({ loading: false })
  }
}

const openExplorerMenu = (
  event: MouseEvent,
  type: 'query-category' | 'query-group' | 'query' | 'table-category' | 'table-group' | 'table',
  payload: { table?: any; query?: any; group?: WorkbenchGroup; maxHeight?: number } = {},
) => {
  contextMenu.value = {
    visible: true,
    type,
    x: Math.min(event.clientX, window.innerWidth - 170),
    y: Math.min(event.clientY, window.innerHeight - (payload.maxHeight || 110)),
    table: payload.table || null,
    query: payload.query || null,
    group: payload.group || null,
  }
}

const toggleExplorerCategory = (category: 'queries' | 'tables') => {
  selectedExplorerNode.value = category
  if (category === 'queries') {
    queriesExpanded.value = !queriesExpanded.value
    return
  }
  tablesExpanded.value = !tablesExpanded.value
}

const openQueryCategoryMenu = (event: MouseEvent) => {
  selectedExplorerNode.value = 'queries'
  openExplorerMenu(event, 'query-category', { maxHeight: 100 })
}

const openObjectGroupMenu = (event: MouseEvent, scope: WorkbenchScope, group: WorkbenchGroup) => {
  selectedExplorerNode.value = `${scope}-group:${group.id}`
  openExplorerMenu(event, `${scope}-group` as 'query-group' | 'table-group', {
    group,
    maxHeight: group.virtual ? 80 : 140,
  })
}

const openQueryMenu = (event: MouseEvent, query: any) => {
  selectedExplorerNode.value = `query:${query.id}`
  openExplorerMenu(event, 'query', { query, maxHeight: 150 })
}

const openTableCategoryMenu = (event: MouseEvent) => {
  selectedExplorerNode.value = 'tables'
  openExplorerMenu(event, 'table-category', { maxHeight: 100 })
}

const openTableMenu = (event: MouseEvent, table: any) => {
  selectTableNode(table)
  openExplorerMenu(event, 'table', {
    table,
    maxHeight: supportsTableStructureEdit.value ? 220 : 190,
  })
}

const closeContextMenu = () => {
  contextMenu.value.visible = false
}

const openTableDesign = () => {
  closeContextMenu()
  tableDesignMode.value = 'create'
  editingTableName.value = ''
  editingTableStructure.value = null
  tableDesignVisible.value = true
}

const openContextNewQuery = () => {
  closeContextMenu()
  createQueryTab()
}

const refreshContextQueries = async () => {
  closeContextMenu()
  await Promise.all([loadQueries(), loadWorkbenchGroups()])
}

const refreshContextTables = async () => {
  closeContextMenu()
  await Promise.all([loadTables(), loadWorkbenchGroups()])
}

const openContextQuery = () => {
  const query = contextMenu.value.query
  closeContextMenu()
  if (query) openSavedQuery(query)
}

const renameContextQuery = async () => {
  const query = contextMenu.value.query
  closeContextMenu()
  if (!query) return
  try {
    const { value } = await ElMessageBox.prompt('请输入查询名称', '重命名查询', {
      inputValue: query.name,
      confirmButtonText: '保存',
      cancelButtonText: '取消',
      inputPattern: /^.{2,100}$/,
      inputErrorMessage: '名称长度需要在 2 到 100 个字符之间',
    })
    const nextName = String(value || '').trim()
    await dataAPI.updateQuery(query.id, { name: nextName })
    tabs.value.forEach((tab) => {
      if (tab.queryId === query.id) {
        tab.title = nextName
        tab.modified = false
      }
    })
    await loadQueries()
    ElMessage.success('查询已重命名')
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') {
      ElMessage.error(getApiErrorMessage(error, '重命名查询失败'))
    }
  }
}

const deleteContextQuery = async () => {
  const query = contextMenu.value.query
  closeContextMenu()
  if (!query) return
  try {
    await ElMessageBox.confirm(`确定删除查询“${query.name}”吗？`, '删除查询', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await dataAPI.deleteQuery(query.id)
    tabs.value.filter((tab) => tab.queryId === query.id).forEach((tab) => closeTab(tab.id))
    await loadQueries()
    ElMessage.success('查询已删除')
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') {
      ElMessage.error(getApiErrorMessage(error, '删除查询失败'))
    }
  }
}

const objectGroupKey = (scope: WorkbenchScope, groupID: string) => `${scope}:${groupID}`

const isObjectGroupExpanded = (scope: WorkbenchScope, groupID: string) => {
  return expandedObjectGroups.value[objectGroupKey(scope, groupID)] !== false
}

const toggleObjectGroup = (scope: WorkbenchScope, groupID: string) => {
  selectedExplorerNode.value = `${scope}-group:${groupID}`
  const key = objectGroupKey(scope, groupID)
  expandedObjectGroups.value = {
    ...expandedObjectGroups.value,
    [key]: !isObjectGroupExpanded(scope, groupID),
  }
}

const groupsForScope = (scope: WorkbenchScope) =>
  scope === 'query' ? queryGroups.value : tableGroups.value

const moveGroupOptions = computed(() => groupsForScope(moveGroupTarget.value.scope))
const moveGroupCurrentId = computed(() => {
  const target = moveGroupTarget.value
  if (!target.item) return null
  if (target.scope === 'query') return target.item.groupId || null
  return tableGroupByName.value[target.item.name] || null
})

const createContextGroup = async () => {
  const group = contextMenu.value.group
  const scope = group?.scope || (contextMenu.value.type === 'query-group' ? 'query' : 'table')
  closeContextMenu()
  try {
    const { value } = await ElMessageBox.prompt('请输入分组名称', '新建分组', {
      confirmButtonText: '创建',
      cancelButtonText: '取消',
      inputPattern: /\S+/,
      inputErrorMessage: '分组名称不能为空',
    })
    await dataAPI.createWorkbenchGroup(props.projectId, props.connection.id, {
      scope,
      name: String(value || '').trim(),
    })
    await loadWorkbenchGroups()
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') {
      ElMessage.error(getApiErrorMessage(error, '创建分组失败'))
    }
  }
}

const renameContextGroup = async () => {
  const group = contextMenu.value.group
  closeContextMenu()
  if (!group || group.virtual) return
  try {
    const { value } = await ElMessageBox.prompt('请输入分组名称', '重命名分组', {
      inputValue: group.name,
      confirmButtonText: '保存',
      cancelButtonText: '取消',
      inputPattern: /\S+/,
      inputErrorMessage: '分组名称不能为空',
    })
    await dataAPI.updateWorkbenchGroup(props.projectId, group.id, {
      name: String(value || '').trim(),
    })
    await loadWorkbenchGroups()
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') {
      ElMessage.error(getApiErrorMessage(error, '重命名分组失败'))
    }
  }
}

const deleteContextGroup = async () => {
  const group = contextMenu.value.group
  closeContextMenu()
  if (!group || group.virtual) return
  try {
    await ElMessageBox.confirm('删除分组不会删除其中的对象，对象会回到未分组。', '删除分组', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await dataAPI.deleteWorkbenchGroup(props.projectId, group.id)
    await Promise.all([loadWorkbenchGroups(), loadQueries()])
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') {
      ElMessage.error(getApiErrorMessage(error, '删除分组失败'))
    }
  }
}

const moveContextQuery = async () => {
  const query = contextMenu.value.query
  closeContextMenu()
  if (!query) return
  moveGroupTarget.value = { scope: 'query', item: query }
  moveGroupDialogVisible.value = true
}

const moveContextTable = async () => {
  const table = contextMenu.value.table
  closeContextMenu()
  if (!table) return
  moveGroupTarget.value = { scope: 'table', item: table }
  moveGroupDialogVisible.value = true
}

const submitMoveGroup = async (groupID: string | null) => {
  const target = moveGroupTarget.value
  if (!target.item) return
  moveGroupSubmitting.value = true
  try {
    if (target.scope === 'query') {
      await dataAPI.moveQueryToWorkbenchGroup(props.projectId, target.item.id, groupID)
      await loadQueries()
    } else {
      await dataAPI.moveTableToWorkbenchGroup(
        props.projectId,
        props.connection.id,
        target.item.name,
        groupID,
      )
      await loadWorkbenchGroups()
    }
    moveGroupDialogRef.value?.closeSilently()
    moveGroupTarget.value = { scope: 'query', item: null }
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, target.scope === 'query' ? '移动查询失败' : '移动表失败'))
  } finally {
    moveGroupSubmitting.value = false
  }
}

const renameContextTable = async () => {
  const table = contextMenu.value.table
  closeContextMenu()
  if (!table) return
  try {
    const { value } = await ElMessageBox.prompt('请输入新表名', '重命名表', {
      inputValue: table.name,
      confirmButtonText: '保存',
      cancelButtonText: '取消',
      inputPattern: /^[A-Za-z_][A-Za-z0-9_]*$/,
      inputErrorMessage: '表名只能包含字母、数字、下划线，且不能以数字开头',
    })
    const nextName = String(value || '').trim()
    await dataAPI.renameConnectionTable(props.projectId, props.connection.id, table.name, {
      name: nextName,
    })
    tabs.value.forEach((tab) => {
      if (tab.table === table.name) {
        tab.table = nextName
        tab.title = tab.title.replace(table.name, nextName)
      }
    })
    if (selectedTableName.value === table.name) selectedTableName.value = nextName
    selectedExplorerNode.value = `table:${nextName}`
    await Promise.all([loadTables(), loadWorkbenchGroups()])
    ElMessage.success('表已重命名')
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') {
      ElMessage.error(getApiErrorMessage(error, '重命名表失败'))
    }
  }
}

const deleteContextTable = async () => {
  const table = contextMenu.value.table
  closeContextMenu()
  if (!table) return
  try {
    await ElMessageBox.confirm(
      `删除表“${table.name}”会删除真实数据库表，且不会级联删除依赖对象。确认继续？`,
      '删除表',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning',
      },
    )
    await dataAPI.deleteConnectionTable(props.projectId, props.connection.id, table.name)
    tabs.value.filter((tab) => tab.table === table.name).forEach((tab) => closeTab(tab.id))
    if (selectedTableName.value === table.name) selectedTableName.value = ''
    if (selectedExplorerNode.value === `table:${table.name}`) selectedExplorerNode.value = 'tables'
    await Promise.all([loadTables(), loadWorkbenchGroups()])
    ElMessage.success('表已删除')
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') {
      ElMessage.error(getApiErrorMessage(error, '删除表失败'))
    }
  }
}

const editContextTableStructure = async () => {
  const table = contextMenu.value.table
  closeContextMenu()
  if (!table) return
  try {
    const response = await dataAPI.getTableStructure(
      props.projectId,
      props.connection.id,
      table.name,
    )
    tableDesignMode.value = 'edit'
    editingTableName.value = table.name
    editingTableStructure.value = response.data || { columns: [], indexes: [], foreignKeys: [] }
    tableDesignVisible.value = true
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载表结构失败'))
  }
}

const openContextStructure = () => {
  const table = contextMenu.value.table
  closeContextMenu()
  if (table) void openTableStructure(table)
}

const openContextData = () => {
  const table = contextMenu.value.table
  closeContextMenu()
  if (table) void openTableData(table)
}

const openContextInsert = async () => {
  const table = contextMenu.value.table
  closeContextMenu()
  if (!table) return
  try {
    const response = await dataAPI.getTableStructure(
      props.projectId,
      props.connection.id,
      table.name,
    )
    const columns = response.data?.columns || []
    if (columns.length === 0) {
      ElMessage.warning('未读取到表字段，无法生成插入模板')
      return
    }
    createQueryTab({
      title: `${table.name} / 插入`,
      table: table.name,
      sql: buildInsertSql(table.name, columns),
    })
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '生成插入模板失败'))
  }
}

const handleTableCreated = async (tableName: string) => {
  await loadTables()
  const table = tables.value.find((item) => item.name === tableName)
  if (table) selectedTableName.value = table.name
}

const handleTableUpdated = async (tableName: string) => {
  await loadTables()
  const structureTabId = `structure-${props.connection.id}-${tableName}`
  const existing = tabs.value.find((item) => item.id === structureTabId)
  if (existing) {
    tabs.value = tabs.value.filter((item) => item.id !== structureTabId)
    await openTableStructure({ name: tableName })
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

const insertSqlParameter = () => {
  sqlEditorRef.value?.insertText?.('?')
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
    const sqlText = buildSqlWithWorkbenchParameters(tab.sql, tab.parameters || [])
    const response = await dataAPI.executeSql(
      props.projectId,
      props.connection.id,
      sqlText,
      [],
    )
    const result = {
      columns: response.data?.columns || [],
      rows: response.data?.rows || [],
      rowCount: response.data?.rowCount || response.data?.rows?.length || 0,
      executionTime: response.data?.executionTime || 0,
    }
    tab.result = result
    tab.displayRows = normalizeRows(result.rows, result.columns)
    tab.resultPage = 1
    pushExecutionHistory({
      id: `${Date.now()}-${Math.random().toString(36).slice(2)}`,
      title: tab.title,
      sql: sqlText,
      rowCount: result.rowCount,
      executionTime: result.executionTime,
    })
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
        config: { sql: tab.sql, parameters: buildSavedParameterDefinitions(tab.parameters) },
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
      config: { sql: tab.sql, parameters: buildSavedParameterDefinitions(tab.parameters) },
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
    selectedExplorerNode.value = ''
    expandedObjectGroups.value = {}
    loadExecutionHistory()
    await reloadExplorer()
    createQueryTab()
  },
)

onMounted(async () => {
  loadExecutionHistory()
  await reloadExplorer()
  createQueryTab()
})

onBeforeUnmount(() => {
  stopResultResize()
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

:global(body.sql-workbench--resizing-result) {
  cursor: row-resize;
  user-select: none;
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
  border-left: 1px solid var(--dc-border);
}

.sql-workbench__inspector-main {
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  gap: 16px;
  padding: 14px 14px 10px;
}

.sql-workbench__history-panel {
  min-height: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 0 14px 14px;
}

.sql-workbench__history-list {
  min-height: 0;
  flex: 1;
  overflow-y: auto;
  padding-right: 2px;
}

.sql-workbench__search {
  min-width: 0;
}

.sql-workbench__tree {
  min-height: 0;
  flex: 1;
  overflow-y: auto;
  padding: 0 8px 12px;
}

.sql-workbench__tree-section + .sql-workbench__tree-section {
  margin-top: 4px;
}

.sql-workbench__tree-category,
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

.sql-workbench__tree-category {
  grid-template-columns: 16px 24px minmax(0, 1fr) auto;
  gap: 8px;
  min-height: 36px;
  padding: 5px 8px 5px 6px;
  border-radius: var(--dc-radius-sm);
  color: var(--dc-text);
  font-size: 14px;
  font-weight: 700;
}

.sql-workbench__tree-category svg,
.sql-workbench__tree-item svg {
  width: 15px;
  height: 15px;
}

.sql-workbench__tree-category small,
.sql-workbench__tree-item small {
  color: var(--dc-text-muted);
  font-size: 11px;
}

.sql-workbench__category-icon {
  width: 22px !important;
  height: 22px !important;
  color: var(--dc-primary);
}

.sql-workbench__tree-body {
  padding: 2px 0 2px 22px;
}

.sql-workbench__tree-group-block {
  display: grid;
  gap: 2px;
}

.sql-workbench__tree-group {
  width: 100%;
  min-height: 28px;
  display: grid;
  grid-template-columns: 16px minmax(0, 1fr) auto;
  align-items: center;
  gap: 6px;
  padding: 4px 6px;
  border: 0;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
  text-align: left;
}

.sql-workbench__tree-group svg {
  width: 14px;
  height: 14px;
}

.sql-workbench__tree-group small {
  color: var(--dc-text-muted);
  font-size: 11px;
}

.sql-workbench__tree-group-body {
  padding-left: 14px;
}

.sql-workbench__tree-item {
  position: relative;
  grid-template-columns: 20px minmax(0, 1fr) auto auto 30px;
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

.sql-workbench__kind-pill {
  padding: 2px 6px;
  border: 1px solid color-mix(in oklch, var(--dc-primary) 22%, var(--dc-border));
  border-radius: var(--dc-radius-xs);
  background: var(--dc-primary-soft);
  color: var(--dc-primary) !important;
  font-size: 11px;
  font-weight: 700;
}

.sql-workbench__tree-item span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sql-workbench__tree-category:hover,
.sql-workbench__tree-category.is-active,
.sql-workbench__tree-group:hover,
.sql-workbench__tree-group.is-active,
.sql-workbench__tree-item:hover,
.sql-workbench__tree-item.is-active,
.sql-workbench__history-item:hover {
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
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
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 12px;
  border-bottom: 1px solid var(--dc-border);
  background: color-mix(in oklch, var(--dc-primary) 5%, var(--dc-surface-muted));
}

.sql-workbench__params-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.sql-workbench__params-head div {
  min-width: 0;
  display: grid;
  gap: 3px;
}

.sql-workbench__params-head strong {
  color: var(--dc-text);
  font-size: 13px;
}

.sql-workbench__params-head span,
.sql-workbench__param-empty {
  color: var(--dc-text-muted);
  font-size: 12px;
}

.sql-workbench__param-list {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 10px;
}

.sql-workbench__param-list label {
  display: grid;
  grid-template-columns: 72px 116px minmax(0, 1fr);
  align-items: center;
  gap: 10px;
  padding: 8px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
}

.sql-workbench__param-list label > span {
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.sql-workbench__param-empty {
  padding: 9px 10px;
  border: 1px dashed var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
}

.sql-workbench__result {
  position: relative;
  min-height: 180px;
  flex: 1;
  display: flex;
  flex-direction: column;
  border-top: 1px solid var(--dc-border);
  overflow: hidden;
}

.sql-workbench__result-resizer {
  position: absolute;
  top: -4px;
  left: 0;
  z-index: 3;
  width: 100%;
  height: 8px;
  cursor: row-resize;
}

.sql-workbench__result-resizer::before {
  content: '';
  position: absolute;
  top: 3px;
  left: 50%;
  width: 52px;
  height: 2px;
  border-radius: 999px;
  background: color-mix(in oklch, var(--dc-text-muted) 42%, transparent);
  transform: translateX(-50%);
}

.sql-workbench__result-resizer:hover::before {
  background: var(--dc-primary);
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
  display: flex;
  flex-direction: column;
}

.sql-workbench__result-data {
  min-height: 0;
  flex: 1;
}

.sql-workbench__result-pagination {
  display: flex;
  justify-content: flex-end;
  min-height: 40px;
  padding: 6px 10px;
  border-top: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
}

.sql-workbench__structure {
  min-height: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
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

.sql-workbench__empty.is-compact {
  padding: 6px 8px;
}

.sql-workbench__loading {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px;
}

.sql-workbench__loading.is-structure {
  flex: 1;
  justify-content: center;
}

.sql-workbench__loading svg {
  width: 14px;
  height: 14px;
  animation: sql-workbench-spin 0.9s linear infinite;
}

.sql-workbench__menu-mask {
  position: fixed;
  inset: 0;
  z-index: 2100;
}

.sql-workbench__context-menu {
  position: fixed;
  min-width: 148px;
  padding: 4px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
}

.sql-workbench__context-menu button {
  width: 100%;
  height: 30px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 8px;
  border: 0;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  font-size: 13px;
  text-align: left;
}

.sql-workbench__context-menu button:hover {
  background: var(--dc-surface-muted);
  color: var(--dc-primary);
}

.sql-workbench__context-menu svg {
  width: 15px;
  height: 15px;
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
