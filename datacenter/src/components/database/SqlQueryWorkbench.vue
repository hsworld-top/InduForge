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
            <small>{{ queryPagination.total }}</small>
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
                    @click="handleQueryTreeClick(query)"
                    @keydown.enter.stop.prevent="openSavedQuery(query)"
                    @contextmenu.prevent.stop="openQueryMenu($event, query)"
                  >
                    <IconTablerFileText />
                    <span>{{ query.name }}</span>
                    <small
                      v-if="queryHasGeneratedDatapoint(query)"
                      class="sql-workbench__datapoint-pill"
                    >
                      数据点
                    </small>
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
            <button
              v-if="!queriesLoading && queries.length < queryPagination.total"
              type="button"
              class="sql-workbench__load-more"
              :disabled="queryLoadingMore"
              @click="loadMoreQueries"
            >
              {{
                queryLoadingMore
                  ? '加载中...'
                  : `加载更多（${queries.length}/${queryPagination.total}）`
              }}
            </button>
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
            <small>{{ isReadOnlySource ? tablePagination.total : filteredTables.length }}</small>
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
                    @click="handleTableTreeClick(table)"
                    @keydown.enter="openTableData(table)"
                    @contextmenu.prevent.stop="openTableMenu($event, table)"
                  >
                    <component :is="resolveTableIcon(table)" />
                    <span>{{ table.name }}</span>
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
            <button
              v-if="isReadOnlySource && !tablesLoading && tables.length < tablePagination.total"
              type="button"
              class="sql-workbench__load-more"
              :disabled="tableLoadingMore"
              @click="loadTables(tablePagination.page + 1, true)"
            >
              {{
                tableLoadingMore
                  ? '加载中...'
                  : `加载更多（${tables.length}/${tablePagination.total}）`
              }}
            </button>
          </div>
        </section>
      </div>
    </aside>

    <main class="sql-workbench__main">
      <div ref="tabbarRef" class="sql-workbench__tabbar" @wheel="handleTabbarWheel">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          type="button"
          class="sql-workbench__tab"
          :class="{ 'is-active': activeTabId === tab.id }"
          :data-tab-id="tab.id"
          @click="activeTabId = tab.id"
          @contextmenu.prevent="openTabMenu($event, tab)"
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
              <el-button
                class="sql-workbench__inspector-toggle"
                size="small"
                @click="inspectorOpen = true"
              >
                <IconTablerLayoutSidebarRight class="sql-workbench__button-icon" />
                输出配置
              </el-button>
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
                <strong>值参数</strong>
                <span v-if="activeTab.parameters.length > 0">
                  {{ parameterHelpText }}；只允许绑定值，不能替代表名、列名或 SQL 片段。
                </span>
                <span v-else>{{ parameterHelpText }}；测试值不会随查询保存。</span>
              </div>
              <el-button size="small" @click="insertSqlParameter">插入参数</el-button>
            </div>
            <div v-if="activeTab.parameters.length > 0" class="sql-workbench__param-list">
              <label v-for="(parameter, index) in activeTab.parameters" :key="parameter.name">
                <span>参数 {{ index + 1 }}</span>
                <span class="sql-workbench__param-kind">值</span>
                <el-input
                  v-model="parameter.value"
                  size="small"
                  :placeholder="`对应 ${parameterToken(index)} 的测试值`"
                />
              </label>
            </div>
            <div v-else class="sql-workbench__param-empty">{{ parameterExample }}</div>
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
              <span v-else>{{ activeTab.executionError ? '执行失败' : '尚未执行' }}</span>
            </div>
            <el-alert
              v-if="activeTab.executionError"
              class="sql-workbench__result-warning"
              type="error"
              :closable="false"
              show-icon
              :title="activeTab.executionError"
            />
            <el-alert
              v-if="activeTab.result?.truncated"
              class="sql-workbench__result-warning"
              type="warning"
              :closable="false"
              show-icon
              :title="
                activeTab.result.truncatedBy === 'bytes'
                  ? `结果超过 ${Math.round(activeTab.result.limits.maxBytes / 1024 / 1024)} MiB，仅显示已安全读取的 ${activeTab.result.rowCount} 行`
                  : `结果超过 ${activeTab.result.limits.maxRows} 行，仅显示前 ${activeTab.result.rowCount} 行`
              "
              description="当前结果不完整，不会作为全量查询结果使用。需要全量数据请使用后续导出任务。"
            />
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
                  >
                    <template #default="{ row }">
                      {{ formatResultCellValue(row[column]) }}
                    </template>
                  </el-table-column>
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
              <el-tab-pane v-if="activeTab.structure.timeseries" label="时序信息" name="timeseries">
                <dl class="sql-workbench__timeseries-facts">
                  <div>
                    <dt>类型</dt>
                    <dd>TimescaleDB Hypertable</dd>
                  </div>
                  <div>
                    <dt>时间分区字段</dt>
                    <dd>{{ activeTab.structure.timeseries.timeColumn }}</dd>
                  </div>
                  <div>
                    <dt>维度字段</dt>
                    <dd>
                      {{
                        (activeTab.structure.timeseries.dimensionColumns || []).join(', ') || '-'
                      }}
                    </dd>
                  </div>
                  <div>
                    <dt>Chunk 间隔</dt>
                    <dd>{{ activeTab.structure.timeseries.chunkInterval || '-' }}</dd>
                  </div>
                  <div>
                    <dt>保留策略</dt>
                    <dd>
                      {{
                        activeTab.structure.timeseries.retentionDays
                          ? `${activeTab.structure.timeseries.retentionDays} 天`
                          : '未启用'
                      }}
                    </dd>
                  </div>
                </dl>
              </el-tab-pane>
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

    <aside class="sql-workbench__inspector" :class="{ 'is-open': inspectorOpen }">
      <div class="sql-workbench__inspector-mobile-head">
        <strong>{{ activeTab?.type === 'structure' ? '表结构' : '查询输出' }}</strong>
        <button type="button" aria-label="关闭右侧面板" @click="inspectorOpen = false">
          <IconTablerX />
        </button>
      </div>
      <div class="sql-workbench__inspector-main">
        <div v-if="activeTab?.type === 'query'" class="sql-workbench__inspector-switch">
          <button
            type="button"
            :class="{ 'is-active': inspectorView === 'outputs' }"
            @click="inspectorView = 'outputs'"
          >
            输出配置
          </button>
          <button
            type="button"
            :class="{ 'is-active': inspectorView === 'preview' }"
            :disabled="!activeTab.result && !activeTab.outputPreviewItems.length"
            @click="inspectorView = 'preview'"
          >
            数据预览
          </button>
        </div>

        <section v-if="activeTab?.type === 'query' && inspectorView === 'outputs'">
          <SourceOutputEditor
            v-model="activeTab.outputs"
            title="查询输出"
            :columns="activeTab.result?.columns || []"
            :column-types="activeTab.result?.columnTypes || {}"
            :sample="activeTab.displayRows?.[0]"
            whole-data-type="object"
            progressive
            default-output-name="查询结果"
            :datapoints-generated="activeTab.datapointsGenerated"
            :output-modified="activeTab.outputModified"
            :generating="activeTab.generatingDatapoints"
            :can-generate="Boolean(activeTab.queryId || activeTab.result)"
            @change="handleOutputChange(activeTab)"
            @generate="generateActiveQueryDatapoints"
          />
        </section>

        <section
          v-if="
            activeTab?.type === 'query' &&
            inspectorView === 'preview' &&
            (activeTab.result || activeTab.outputPreviewItems.length)
          "
          class="sql-workbench__output-preview"
        >
          <div class="sql-workbench__output-preview-head">
            <div>
              <strong>结果预览</strong>
              <span>{{ outputPreviewDescription(activeTab) }}</span>
            </div>
            <div class="sql-workbench__output-preview-actions">
              <el-button
                v-if="activeTab.datapointsGenerated"
                text
                type="primary"
                size="small"
                :loading="activeTab.outputPreviewLoading"
                @click="refreshOutputPreview(activeTab)"
              >
                刷新 GET
              </el-button>
              <el-button
                type="primary"
                size="small"
                :plain="activeTab.datapointsGenerated"
                :loading="activeTab.generatingDatapoints"
                :disabled="
                  (!activeTab.queryId && !activeTab.result) ||
                  (activeTab.datapointsGenerated && !activeTab.outputModified)
                "
                :title="datapointGenerationButtonTitle(activeTab)"
                @click="generateActiveQueryDatapoints"
              >
                {{ datapointGenerationButtonText(activeTab) }}
              </el-button>
            </div>
          </div>
          <el-alert
            v-if="activeTab.outputPreviewError"
            :title="activeTab.outputPreviewError"
            type="warning"
            :closable="false"
            show-icon
          />
          <div
            v-for="item in activeTab.outputPreviewItems"
            :key="item.key"
            class="sql-workbench__output-preview-item"
          >
            <div class="sql-workbench__output-preview-meta">
              <strong>{{ item.displayName }}</strong>
              <el-tag v-if="item.quality" size="small" :type="previewQualityType(item.quality)">
                {{ item.quality === 'preview' ? '配置预览' : item.quality }}
              </el-tag>
            </div>
            <code v-if="item.path">GET {{ item.path }}</code>
            <small v-if="item.error">{{ item.error }}</small>
            <pre v-else>{{ formatOutputPreview(item.value) }}</pre>
          </div>
        </section>

        <section v-if="activeTab?.type === 'structure'">
          <template v-if="activeTab.structure.timeseries">
            <div class="sql-workbench__panel-title">时序策略</div>
            <dl class="sql-workbench__facts">
              <div>
                <dt>时间字段</dt>
                <dd>{{ activeTab.structure.timeseries.timeColumn }}</dd>
              </div>
              <div>
                <dt>Chunk</dt>
                <dd>{{ activeTab.structure.timeseries.chunkInterval || '-' }}</dd>
              </div>
            </dl>
          </template>
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
      v-if="!isReadOnlySource"
      v-model="tableDesignVisible"
      :project-id="projectId"
      :connection-id="connection.id"
      :db-type="dbType"
      :supports-hypertable="supportsHypertable"
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
          <button v-if="contextMenu.type === 'tab'" type="button" @click="closeContextTab">
            <IconTablerX />
            <span>关闭当前</span>
          </button>
          <button
            v-if="contextMenu.type === 'tab'"
            type="button"
            :disabled="!hasContextTabsRight"
            @click="closeContextRightTabs"
          >
            <IconTablerChevronsRight />
            <span>关闭右侧标签</span>
          </button>
          <button v-if="contextMenu.type === 'tab'" type="button" @click="closeContextAllTabs">
            <IconTablerStack2 />
            <span>关闭所有标签</span>
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
            v-if="contextMenu.type === 'table-category' && !isReadOnlySource"
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
          <button
            v-if="contextMenu.type === 'table' && !isReadOnlySource"
            type="button"
            @click="openContextInsert"
          >
            <IconTablerPlus />
            <span>插入数据</span>
          </button>
          <button
            v-if="contextMenu.type === 'table' && contextMenu.table?.kind === 'hypertable'"
            type="button"
            @click="openContextRecentTimeseries"
          >
            <IconTablerClock />
            <span>查询最近时序</span>
          </button>
          <button
            v-if="contextMenu.type === 'table' && contextMenu.table?.kind === 'hypertable'"
            type="button"
            @click="openContextTimeseriesAggregate"
          >
            <IconTablerChartLine />
            <span>按时间聚合</span>
          </button>
          <button
            v-if="contextMenu.type === 'table' && contextMenu.table?.kind === 'hypertable'"
            type="button"
            @click="openContextTimeseriesQuality"
          >
            <IconTablerAlertTriangle />
            <span>查询异常质量</span>
          </button>
          <button
            v-if="contextMenu.type === 'table' && !isReadOnlySource"
            type="button"
            @click="renameContextTable"
          >
            <IconTablerEdit />
            <span>重命名表</span>
          </button>
          <button v-if="contextMenu.type === 'table'" type="button" @click="moveContextTable">
            <IconTablerFolder />
            <span>移动到分组</span>
          </button>
          <button
            v-if="contextMenu.type === 'table' && !isReadOnlySource"
            type="button"
            @click="deleteContextTable"
          >
            <IconTablerTrash />
            <span>删除表</span>
          </button>
        </div>
      </div>
    </Teleport>
  </section>
</template>

<script setup lang="ts">
import { computed, inject, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { format as formatSql } from 'sql-formatter'
import IconTablerAlertTriangle from '~icons/tabler/alert-triangle'
import IconTablerChevronDown from '~icons/tabler/chevron-down'
import IconTablerChevronRight from '~icons/tabler/chevron-right'
import IconTablerChevronsRight from '~icons/tabler/chevrons-right'
import IconTablerChartLine from '~icons/tabler/chart-line'
import IconTablerClock from '~icons/tabler/clock'
import IconTablerColumns from '~icons/tabler/columns'
import IconTablerDatabase from '~icons/tabler/database'
import IconTablerEdit from '~icons/tabler/edit'
import IconTablerFileSearch from '~icons/tabler/file-search'
import IconTablerFileText from '~icons/tabler/file-text'
import IconTablerFolder from '~icons/tabler/folder'
import IconTablerLoader2 from '~icons/tabler/loader-2'
import IconTablerLayoutSidebarRight from '~icons/tabler/layout-sidebar-right'
import IconTablerPlayerPlay from '~icons/tabler/player-play'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerStack2 from '~icons/tabler/stack-2'
import IconTablerTable from '~icons/tabler/table'
import IconTablerTrash from '~icons/tabler/trash'
import IconTablerX from '~icons/tabler/x'
import MonacoEditor from '@/components/MonacoEditor.vue'
import dataAPI, { getDataPointValue } from '@/api/data.api'
import { SqlWorkbenchResultSchema } from '@/api/schemas/sql-workbench.schema'
import { createWholeSourceOutput } from '@/api/schemas/source-output.schema'
import { getApiErrorMessage } from '@/utils/request'
import { sqlCodeForParameterDetection } from '@/utils/sqlParameterDetection'
import SourceOutputEditor from '@/components/shared/SourceOutputEditor.vue'
import WorkbenchSourceHeader from '@/components/workbench/WorkbenchSourceHeader.vue'
import { generatedQueryCopyName, nextGeneratedQueryName } from './sqlQueryNaming'
import {
  createSqlTreeActivationTracker,
  sqlWorkbenchTablePreviewTabId,
  sqlWorkbenchTabIdsToClose,
} from './sqlWorkbenchTabs'
import { buildTablePreviewSql, quoteSqlIdentifier } from './sqlWorkbenchSql'
import { buildQueryOutputPreviews } from './sourceOutputPreview'
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
type SqlTemplateParameterType = 'value'
type SqlTemplateParameter = {
  name: string
  type: SqlTemplateParameterType
  value: string
}

const isValidWorkbenchDisplayName = (value: string) => {
  const length = Array.from(value).length
  if (length < 1 || length > 100 || value !== value.trim()) return false
  return !Array.from(value).some((character) => {
    const code = character.codePointAt(0) ?? 0
    return code < 32 || code === 127
  })
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
const tabs = ref<any[]>([])
const tabbarRef = ref<HTMLElement | null>(null)
const inspectorOpen = ref(false)
const inspectorView = ref<'outputs' | 'preview'>('outputs')
const registerDraftChecker =
  inject<
    (guard: {
      isDirty: () => boolean
      save: () => Promise<boolean>
      discard: () => void
    }) => () => void
  >('registerDraftChecker')
let unregisterDraftChecker: (() => void) | undefined
const sqlEditorRef = ref<InstanceType<typeof MonacoEditor> | null>(null)
const activeTabId = ref('')
const filterText = ref('')
const selectedTableName = ref('')
const selectedExplorerNode = ref('')
const tablesLoading = ref(false)
const tableLoadingMore = ref(false)
const queriesLoading = ref(false)
const queryLoadingMore = ref(false)
const queryPagination = ref({ page: 1, pageSize: 50, total: 0 })
const tablePagination = ref({ page: 1, pageSize: 50, total: 0 })
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
const moveGroupTarget = ref<{ scope: WorkbenchScope; item: any | null }>({
  scope: 'query',
  item: null,
})
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
    | 'tab'
    | null
  x: number
  y: number
  table: any | null
  query: any | null
  group: WorkbenchGroup | null
  tab: any | null
}>({
  visible: false,
  type: null,
  x: 0,
  y: 0,
  table: null,
  query: null,
  group: null,
  tab: null,
})
let tabCounter = 0
let querySearchTimer: number | undefined
let queryRequestSequence = 0
const defaultEditorHeight = 270
const minEditorHeight = 120
const maxEditorHeight = 560
const resultPageSizes = [50, 100, 200, 500]
const historyLimit = 50
const historyStoragePrefix = 'datacenter:sql-workbench:history'
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
const parameterDialect = computed(() => {
  if (dbType.value === 'sqlserver') return 'sqlserver'
  if (dbType.value === 'postgresql' || props.connection.type?.startsWith('builtin.'))
    return 'postgresql'
  return 'mysql'
})
const parameterToken = (index: number) => {
  if (parameterDialect.value === 'postgresql') return `$${index + 1}`
  if (parameterDialect.value === 'sqlserver') return `@p${index + 1}`
  return '?'
}
const parameterHelpText = computed(() => {
  if (parameterDialect.value === 'postgresql') return 'PostgreSQL 使用 $1、$2… 占位'
  if (parameterDialect.value === 'sqlserver') return 'SQL Server 使用 @p1、@p2… 占位'
  if (dbType.value === 'tdengine') return 'TDengine 使用 ? 占位，按出现顺序绑定'
  return 'MySQL 使用 ? 占位，按出现顺序绑定'
})
const parameterExample = computed(
  () => `示例：SELECT * FROM device_data WHERE id = ${parameterToken(0)}`,
)
const supportsHypertable = computed(() => props.connection.type === 'builtin.timeseries')
const isReadOnlySource = computed(() => props.connection.type === 'tdengine')
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
  return (
    dbConfig.value.databaseName || dbConfig.value.database || dbConfig.value.schema || '已配置库'
  )
})
const sourceMetaRows = computed(() => {
  const rows = [
    { label: '类型', value: dbTypeLabel.value },
    {
      label: props.connection.type?.startsWith('builtin.') ? '标识' : '库',
      value:
        props.connection.type?.startsWith('builtin.') && dbConfig.value.schema
          ? String(dbConfig.value.schema)
          : databaseLabel.value,
    },
  ]
  if (isReadOnlySource.value) {
    rows.push({
      label: '地址',
      value: `${dbConfig.value.protocol || 'ws'}://${dbConfig.value.host || '-'}:${dbConfig.value.port || 6041}`,
    })
    rows.push({ label: '时区', value: dbConfig.value.timezone || '未设置' })
  }
  return rows
})
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
const tableTreeGroups = computed(() => {
  if (!isReadOnlySource.value) {
    return buildTreeGroups('table', tableGroups.value, filteredTables.value)
  }
  const definitions = [
    { id: '__tdengine_stables__', name: '超级表', kinds: ['supertable'] },
    { id: '__tdengine_tables__', name: '普通表', kinds: ['table'] },
    { id: '__tdengine_children__', name: '子表', kinds: ['child_table'] },
  ]
  return definitions.map((definition) => ({
    id: definition.id,
    name: definition.name,
    scope: 'table' as WorkbenchScope,
    virtual: true,
    items: filteredTables.value.filter((table) => definition.kinds.includes(String(table.kind))),
  }))
})

const resolveTableIcon = (table: any) => {
  if (['hypertable', 'supertable'].includes(table?.kind)) return IconTablerStack2
  return IconTablerTable
}

const quoteTable = (tableName: string) => quoteSqlIdentifier(dbType.value, tableName)

const quoteColumn = (columnName: string) => quoteTable(columnName)

const buildSelectSql = (tableName: string) => buildTablePreviewSql(dbType.value, tableName)

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

const firstNumericColumn = (columns: Array<Record<string, any>>, excluded = new Set<string>()) =>
  columns.find((column) => {
    const type = String(column.type || '').toLowerCase()
    return (
      !excluded.has(column.name) &&
      (type.includes('int') ||
        type.includes('double') ||
        type.includes('numeric') ||
        type.includes('decimal') ||
        type.includes('float'))
    )
  })

const buildRecentTimeseriesSql = (tableName: string, structure: Record<string, any>) => {
  const timeColumn = structure.timeseries?.timeColumn || 'ts'
  return `SELECT *\nFROM ${quoteTable(tableName)}\nWHERE ${quoteColumn(timeColumn)} >= now() - INTERVAL '1 hour'\nORDER BY ${quoteColumn(timeColumn)} DESC\nLIMIT 100;`
}

const buildTimeseriesAggregateSql = (
  tableName: string,
  structure: Record<string, any>,
  columns: Array<Record<string, any>>,
) => {
  const timeseries = structure.timeseries || {}
  const timeColumn = timeseries.timeColumn || 'ts'
  const dimensions = Array.isArray(timeseries.dimensionColumns) ? timeseries.dimensionColumns : []
  const excluded = new Set([timeColumn, ...dimensions])
  const valueColumn =
    firstNumericColumn(columns, excluded)?.name ||
    columns.find((column) => !excluded.has(column.name))?.name ||
    'value'
  const dimensionSelect =
    dimensions.length > 0 ? `,\n  ${dimensions.map(quoteColumn).join(', ')}` : ''
  const dimensionGroup = dimensions.length > 0 ? `, ${dimensions.map(quoteColumn).join(', ')}` : ''
  return `SELECT\n  time_bucket('5 minutes', ${quoteColumn(timeColumn)}) AS bucket${dimensionSelect},\n  avg(${quoteColumn(valueColumn)}) AS avg_${valueColumn}\nFROM ${quoteTable(tableName)}\nWHERE ${quoteColumn(timeColumn)} >= now() - INTERVAL '1 day'\nGROUP BY bucket${dimensionGroup}\nORDER BY bucket DESC;`
}

const buildTimeseriesQualitySql = (tableName: string, structure: Record<string, any>) => {
  const timeColumn = structure.timeseries?.timeColumn || 'ts'
  return `SELECT *\nFROM ${quoteTable(tableName)}\nWHERE ${quoteColumn('quality')} <> 192\n  AND ${quoteColumn(timeColumn)} >= now() - INTERVAL '1 day'\nORDER BY ${quoteColumn(timeColumn)} DESC\nLIMIT 100;`
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

// SQL 结果中的 JSON、数组不能交给表格隐式转字符串，否则只会显示 [object Object]。
const formatResultCellValue = (value: unknown) => {
  if (value === null) return 'null'
  if (value === undefined) return ''
  if (typeof value !== 'object') return String(value)
  try {
    return JSON.stringify(value)
  } catch {
    return String(value)
  }
}

const normalizeParameterType = (type: any): SqlTemplateParameterType => {
  void type
  return 'value'
}

const extractParameters = (sql: string, oldParameters: any[] = []): SqlTemplateParameter[] => {
  const parameterCode = sqlCodeForParameterDetection(sql)
  const count = (() => {
    if (parameterDialect.value === 'postgresql') {
      const indexes = [...parameterCode.matchAll(/\$(\d+)/g)].map((match) => Number(match[1]))
      return indexes.length ? Math.max(...indexes) : 0
    }
    if (parameterDialect.value === 'sqlserver')
      return new Set(
        [...parameterCode.matchAll(/@([A-Za-z_][A-Za-z0-9_]*)/g)].map((match) =>
          match[1].toLowerCase(),
        ),
      ).size
    return (parameterCode.match(/\?/g) || []).length
  })()
  return Array.from({ length: count }, (_, index) => ({
    name: `param${index + 1}`,
    type: normalizeParameterType(oldParameters[index]?.type),
    value: oldParameters[index]?.value || '',
  }))
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

const loadTables = async (page = 1, append = false) => {
  if (append) tableLoadingMore.value = true
  else tablesLoading.value = true
  try {
    const params = isReadOnlySource.value
      ? {
          page,
          pageSize: tablePagination.value.pageSize,
          search: filterText.value.trim() || undefined,
        }
      : undefined
    const response = await dataAPI.getConnectionTables(props.projectId, props.connection.id, params)
    const nextTables = response.data?.tables || []
    tables.value = append ? [...tables.value, ...nextTables] : nextTables
    if (isReadOnlySource.value) {
      const pagination = response.data?.pagination || {}
      tablePagination.value = {
        page: Number(pagination.page || page),
        pageSize: Number(pagination.limit || tablePagination.value.pageSize),
        total: Number(pagination.total || 0),
      }
    } else {
      tablePagination.value = {
        page: 1,
        pageSize: tables.value.length || 50,
        total: tables.value.length,
      }
    }
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载表列表失败'))
  } finally {
    tablesLoading.value = false
    tableLoadingMore.value = false
  }
}

const loadQueries = async (page = 1, append = false) => {
  const requestSequence = ++queryRequestSequence
  const projectId = props.projectId
  const connectionId = props.connection.id
  if (append) {
    queryLoadingMore.value = true
  } else {
    queriesLoading.value = true
    queryLoadingMore.value = false
  }
  try {
    const response = await dataAPI.getQueries(projectId, {
      connectionId,
      queryType: 'sql',
      search: filterText.value.trim() || undefined,
      page,
      pageSize: queryPagination.value.pageSize,
    })
    if (
      requestSequence !== queryRequestSequence ||
      projectId !== props.projectId ||
      connectionId !== props.connection.id
    ) {
      return
    }
    const next = response.data?.queries || response.data || []
    if (append) {
      const byId = new Map(queries.value.map((query) => [String(query.id), query]))
      next.forEach((query: any) => byId.set(String(query.id), query))
      queries.value = Array.from(byId.values())
    } else {
      queries.value = next
    }
    const pageInfo = response.data?.pagination || {}
    queryPagination.value = {
      page: Number(pageInfo.page || page),
      pageSize: Number(pageInfo.pageSize || queryPagination.value.pageSize),
      total: Number(pageInfo.total ?? queries.value.length),
    }
  } catch (error) {
    if (
      requestSequence !== queryRequestSequence ||
      projectId !== props.projectId ||
      connectionId !== props.connection.id
    ) {
      return
    }
    ElMessage.error(getApiErrorMessage(error, '加载保存查询失败'))
  } finally {
    if (requestSequence === queryRequestSequence) {
      if (append) queryLoadingMore.value = false
      else queriesLoading.value = false
    }
  }
}

const loadMoreQueries = () => {
  if (queryLoadingMore.value || queries.value.length >= queryPagination.value.total) return
  void loadQueries(queryPagination.value.page + 1, true)
}

const queryHasGeneratedDatapoint = (query: any) =>
  Array.isArray(query?.outputs) &&
  query.outputs.some(
    (output: any) => Boolean(output?.datapointId) && Boolean(output?.datapointPath),
  )

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

const normalizeQueryEditorOutput = (output: any) => {
  const selector = { ...(output.selector || { kind: 'whole' }) }
  if (selector.kind === 'path') {
    const rootColumn = Array.isArray(selector.segments)
      ? selector.segments.find((segment: unknown) => typeof segment === 'string')
      : ''
    selector.kind = rootColumn ? 'column' : 'whole'
    if (rootColumn) selector.column = rootColumn
    delete selector.segments
  }
  return { ...output, selector }
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
  const generatedTitle = nextGeneratedQueryName([
    ...tabs.value.map((tab) => String(tab.title || '')),
    ...queries.value.map((query) => String(query.name || '')),
  ])

  const tab = {
    id,
    queryId: initial.queryId || '',
    type: 'query',
    title: initial.title || generatedTitle,
    icon: IconTablerFileText,
    table: initial.table || '',
    sql: initialSql,
    parameters: extractParameters(initialSql, initial.parameters || []),
    outputs: initial.outputs?.length
      ? initial.outputs.slice(0, 1).map((output: any) => normalizeQueryEditorOutput(output))
      : [createWholeSourceOutput('result', initial.title || '查询结果', 'object')],
    datapointsGenerated: Boolean(initial.outputs?.length),
    outputModified: false,
    generatingDatapoints: false,
    outputPreviewItems: [],
    outputPreviewLoading: false,
    outputPreviewError: '',
    outputPreviewSource: '',
    outputPreviewRequestSequence: 0,
    executionError: '',
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
    id: sqlWorkbenchTablePreviewTabId(props.connection.id, table.name),
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
    outputs: query.outputs || [],
  })
}

const registerTreeActivation = createSqlTreeActivationTracker()

const handleQueryTreeClick = (query: any) => {
  const key = `query:${query.id}`
  selectedExplorerNode.value = key
  if (registerTreeActivation(key)) openSavedQuery(query)
}

const selectTableNode = (table: any) => {
  selectedExplorerNode.value = `table:${table.name}`
  selectedTableName.value = table.name
}

const handleTableTreeClick = (table: any) => {
  const key = `table:${table.name}`
  selectTableNode(table)
  if (registerTreeActivation(key)) void openTableData(table)
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
  type:
    | 'query-category'
    | 'query-group'
    | 'query'
    | 'table-category'
    | 'table-group'
    | 'table'
    | 'tab',
  payload: {
    table?: any
    query?: any
    group?: WorkbenchGroup
    tab?: any
    maxHeight?: number
  } = {},
) => {
  contextMenu.value = {
    visible: true,
    type,
    x: Math.min(event.clientX, window.innerWidth - 170),
    y: Math.min(event.clientY, window.innerHeight - (payload.maxHeight || 110)),
    table: payload.table || null,
    query: payload.query || null,
    group: payload.group || null,
    tab: payload.tab || null,
  }
}

const openTabMenu = (event: MouseEvent, tab: any) => {
  openExplorerMenu(event, 'tab', { tab, maxHeight: 118 })
}

const hasContextTabsRight = computed(() => {
  const tabId = contextMenu.value.tab?.id
  const index = tabs.value.findIndex((tab) => tab.id === tabId)
  return index >= 0 && index < tabs.value.length - 1
})

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
      inputValidator: isValidWorkbenchDisplayName,
      inputErrorMessage: '名称需为 1 到 100 个字符，且不能包含首尾空格或控制字符',
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
    tabs.value.filter((tab) => tab.queryId === query.id).forEach((tab) => removeTab(tab.id))
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
      inputValidator: isValidWorkbenchDisplayName,
      inputErrorMessage: '分组名称需为 1 到 100 个字符，且不能包含首尾空格或控制字符',
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
      inputValidator: isValidWorkbenchDisplayName,
      inputErrorMessage: '分组名称需为 1 到 100 个字符，且不能包含首尾空格或控制字符',
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
    ElMessage.error(
      getApiErrorMessage(error, target.scope === 'query' ? '移动查询失败' : '移动表失败'),
    )
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
      inputPattern: /^[A-Za-z_][A-Za-z0-9_]{0,62}$/,
      inputErrorMessage: '表名需为 1 到 63 个字符，只能包含字母、数字、下划线，且不能以数字开头',
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

const openContextRecentTimeseries = async () => {
  const table = contextMenu.value.table
  closeContextMenu()
  if (!table) return
  try {
    const response = await dataAPI.getTableStructure(
      props.projectId,
      props.connection.id,
      table.name,
    )
    createQueryTab({
      title: `${table.name} / 最近时序`,
      table: table.name,
      sql: buildRecentTimeseriesSql(table.name, response.data || {}),
    })
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '生成时序查询模板失败'))
  }
}

const openContextTimeseriesAggregate = async () => {
  const table = contextMenu.value.table
  closeContextMenu()
  if (!table) return
  try {
    const response = await dataAPI.getTableStructure(
      props.projectId,
      props.connection.id,
      table.name,
    )
    const structure = response.data || {}
    createQueryTab({
      title: `${table.name} / 时间聚合`,
      table: table.name,
      sql: buildTimeseriesAggregateSql(table.name, structure, structure.columns || []),
    })
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '生成时序聚合模板失败'))
  }
}

const openContextTimeseriesQuality = async () => {
  const table = contextMenu.value.table
  closeContextMenu()
  if (!table) return
  try {
    const response = await dataAPI.getTableStructure(
      props.projectId,
      props.connection.id,
      table.name,
    )
    createQueryTab({
      title: `${table.name} / 异常质量`,
      table: table.name,
      sql: buildTimeseriesQualitySql(table.name, response.data || {}),
    })
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '生成异常质量查询模板失败'))
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

const removeTab = (tabId: string) => {
  const index = tabs.value.findIndex((tab) => tab.id === tabId)
  if (index < 0) return
  tabs.value.splice(index, 1)
  if (activeTabId.value === tabId) {
    activeTabId.value = tabs.value[index - 1]?.id || tabs.value[0]?.id || ''
  }
}

const closeTab = async (tabId: string) => {
  const tab = tabs.value.find((item) => item.id === tabId)
  if (!tab) return
  if (tab.type === 'query' && tab.modified) {
    try {
      await ElMessageBox.confirm(`查询“${tab.title}”有未保存修改，关闭前是否保存？`, '关闭查询', {
        confirmButtonText: '保存后关闭',
        cancelButtonText: '放弃修改',
        distinguishCancelAndClose: true,
        type: 'warning',
      })
      if (!(await saveQueryTab(tab))) return
    } catch (action) {
      if (action !== 'cancel') return
    }
  }
  removeTab(tabId)
}

const closeTabSet = async (tabIds: string[], fallbackTabId = '') => {
  const idSet = new Set(tabIds)
  const targetTabs = tabs.value.filter((tab) => idSet.has(tab.id))
  if (targetTabs.length === 0) return
  const modifiedTabs = targetTabs.filter((tab) => tab.type === 'query' && tab.modified)
  if (modifiedTabs.length > 0) {
    try {
      await ElMessageBox.confirm(
        `将关闭 ${targetTabs.length} 个标签，其中 ${modifiedTabs.length} 个有未保存修改。`,
        '批量关闭标签',
        {
          confirmButtonText: '保存后关闭',
          cancelButtonText: '放弃修改并关闭',
          distinguishCancelAndClose: true,
          type: 'warning',
        },
      )
      for (const tab of modifiedTabs) {
        if (!(await saveQueryTab(tab))) return
      }
    } catch (action) {
      if (action !== 'cancel') return
    }
  }

  const activeWillClose = idSet.has(activeTabId.value)
  const firstTargetIndex = tabs.value.findIndex((tab) => idSet.has(tab.id))
  tabs.value = tabs.value.filter((tab) => !idSet.has(tab.id))
  if (!activeWillClose) return
  activeTabId.value =
    tabs.value.find((tab) => tab.id === fallbackTabId)?.id ||
    tabs.value[Math.min(Math.max(firstTargetIndex - 1, 0), tabs.value.length - 1)]?.id ||
    ''
}

const closeContextTab = async () => {
  const tabId = contextMenu.value.tab?.id || ''
  closeContextMenu()
  if (tabId) await closeTab(tabId)
}

const closeContextRightTabs = async () => {
  const tabId = contextMenu.value.tab?.id || ''
  const tabIds = sqlWorkbenchTabIdsToClose(tabs.value, tabId, 'right')
  closeContextMenu()
  await closeTabSet(tabIds, tabId)
}

const closeContextAllTabs = async () => {
  const tabId = contextMenu.value.tab?.id || ''
  const tabIds = sqlWorkbenchTabIdsToClose(tabs.value, tabId, 'all')
  closeContextMenu()
  await closeTabSet(tabIds)
}

const handleTabbarWheel = (event: WheelEvent) => {
  const tabbar = event.currentTarget as HTMLElement | null
  if (!tabbar || tabbar.scrollWidth <= tabbar.clientWidth) return
  if (Math.abs(event.deltaY) <= Math.abs(event.deltaX) || event.deltaY === 0) return
  tabbar.scrollLeft += event.deltaY
  event.preventDefault()
}

const handleSqlChange = () => {
  if (!activeTab.value || activeTab.value.type !== 'query') return
  activeTab.value.modified = true
  activeTab.value.parameters = extractParameters(activeTab.value.sql, activeTab.value.parameters)
}

const insertSqlParameter = () => {
  sqlEditorRef.value?.insertText?.(parameterToken(activeTab.value?.parameters?.length || 0))
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
  tab.outputPreviewItems = []
  tab.outputPreviewError = ''
  tab.outputPreviewSource = ''
  tab.executionError = ''
  tab.executing = true
  try {
    const sqlText = tab.sql
    const values = (tab.parameters || []).map((parameter: SqlTemplateParameter) => parameter.value)
    const response = await dataAPI.executeSql(props.projectId, props.connection.id, sqlText, values)
    const result = SqlWorkbenchResultSchema.parse(response.data || {})
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
    void refreshOutputPreview(tab)
    inspectorView.value = 'preview'
  } catch (error) {
    tab.result = null
    tab.displayRows = []
    tab.executionError = getApiErrorMessage(error, '执行 SQL 失败')
    ElMessage.error(tab.executionError)
  } finally {
    tab.executing = false
  }
}

const executeActiveQuery = async () => {
  await executeTab(activeTab.value)
}

const saveQueryTab = async (tab: any): Promise<boolean> => {
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
      return true
    } catch (error) {
      ElMessage.error(getApiErrorMessage(error, '保存查询失败'))
      return false
    } finally {
      tab.saving = false
    }
  }

  try {
    const { value } = await ElMessageBox.prompt('请输入查询名称', '保存查询', {
      confirmButtonText: '保存',
      cancelButtonText: '取消',
      inputValidator: isValidWorkbenchDisplayName,
      inputErrorMessage: '名称需为 1 到 100 个字符，且不能包含首尾空格或控制字符',
      inputValue: tab.table || tab.title,
    })

    tab.saving = true
    const response = await dataAPI.createQuery(props.projectId, {
      name: String(value || '').trim(),
      connectionId: props.connection.id,
      queryType: 'sql',
      config: { sql: tab.sql, parameters: buildSavedParameterDefinitions(tab.parameters) },
    })
    tab.title = String(value || '').trim()
    tab.queryId = response.data?.id
    tab.modified = false
    await loadQueries()
    ElMessage.success('查询已保存')
    return true
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') {
      ElMessage.error(getApiErrorMessage(error, '保存查询失败'))
    }
    return false
  } finally {
    tab.saving = false
  }
}

const saveActiveQuery = async () => {
  const tab = activeTab.value
  if (!tab || tab.type !== 'query') return
  await saveQueryTab(tab)
}

const datapointGenerationButtonText = (tab: any) => {
  if (!tab.datapointsGenerated) return '保存并生成数据点'
  if (tab.outputModified) return '保存并更新数据点'
  return '数据点已生成'
}

const datapointGenerationButtonTitle = (tab: any) => {
  if (!tab.queryId && !tab.result) return '请先运行 SQL，确认结果后再生成数据点'
  if (!tab.queryId) return '输入查询名称，保存后直接生成数据点'
  if (tab.datapointsGenerated && !tab.outputModified) return '提取配置和数据点已经同步保存'
  return tab.datapointsGenerated ? '保存当前提取配置并更新数据点' : '保存当前配置并生成数据点'
}

const generateActiveQueryDatapoints = async () => {
  const tab = activeTab.value
  if (!tab || tab.type !== 'query') return
  if (!tab.queryId && !tab.result) {
    ElMessage.warning('请先运行 SQL，确认结果后再生成数据点')
    return
  }
  tab.generatingDatapoints = true
  try {
    // 临时查询已有成功结果时，生成动作负责先命名保存，再继续创建数据点。
    if (!tab.queryId && !(await saveQueryTab(tab))) return
    if (!tab.queryId) return
    const response = await dataAPI.updateQuery(tab.queryId, {
      outputs: (tab.outputs || []).slice(0, 1),
    })
    const generatedOutputs = (response.data?.outputs || []).slice(0, 1)
    tab.outputs = generatedOutputs.map((output: any) => ({
      ...output,
      selector: { ...output.selector },
    }))
    tab.datapointsGenerated = generatedOutputs.length > 0
    tab.outputModified = false
    await loadQueries()
    if (tab.result) {
      await refreshOutputPreview(tab)
      inspectorView.value = 'preview'
    } else {
      inspectorView.value = 'outputs'
    }
    ElMessage.success('数据点已生成')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '生成数据点失败'))
  } finally {
    tab.generatingDatapoints = false
  }
}

const handleOutputChange = (tab: any) => {
  tab.outputModified = true
  if (tab.result) void refreshOutputPreview(tab)
}

const refreshOutputPreview = async (tab: any) => {
  if (!tab?.result) return
  const requestSequence = Number(tab.outputPreviewRequestSequence || 0) + 1
  tab.outputPreviewRequestSequence = requestSequence
  const generatedOutputs = (tab.outputs || []).filter((output: any) => output.datapointPath)
  if (!tab.datapointsGenerated || generatedOutputs.length === 0 || tab.outputModified) {
    tab.outputPreviewLoading = false
    tab.outputPreviewSource = 'config'
    tab.outputPreviewError = ''
    tab.outputPreviewItems = buildQueryOutputPreviews(
      tab.outputs || [],
      tab.result,
      tab.displayRows || [],
    )
    return
  }

  tab.outputPreviewLoading = true
  tab.outputPreviewError = ''
  tab.outputPreviewSource = 'get'
  tab.outputPreviewItems = generatedOutputs.map((output: any) => ({
    key: output.id || output.key,
    displayName: output.displayName || output.key,
    path: output.datapointPath,
    quality: 'unknown',
    value: null,
  }))
  try {
    const results = await Promise.allSettled(
      generatedOutputs.map((output: any) =>
        getDataPointValue(
          props.projectId,
          output.datapointPath,
          Object.fromEntries(
            (tab.parameters || []).map((parameter: SqlTemplateParameter) => [
              parameter.name,
              parameter.value,
            ]),
          ),
        ),
      ),
    )
    if (requestSequence !== tab.outputPreviewRequestSequence) return
    tab.outputPreviewItems = results.map((result, index) => {
      const output = generatedOutputs[index]
      if (result.status === 'rejected') {
        return {
          key: output.id || output.key,
          displayName: output.displayName || output.key,
          path: output.datapointPath,
          quality: 'bad',
          value: null,
          error: getApiErrorMessage(result.reason, '读取数据点失败'),
        }
      }
      const value = result.value?.data || result.value || {}
      return {
        key: output.id || output.key,
        displayName: output.displayName || output.key,
        path: output.datapointPath,
        quality: value.quality || 'unknown',
        value: value.value,
      }
    })
    const failedCount = tab.outputPreviewItems.filter((item: any) => item.error).length
    if (failedCount > 0) tab.outputPreviewError = `${failedCount} 个数据点读取失败`
  } catch (error) {
    if (requestSequence === tab.outputPreviewRequestSequence) {
      tab.outputPreviewError = getApiErrorMessage(error, '读取数据点预览失败')
      tab.outputPreviewItems = generatedOutputs.map((output: any) => ({
        key: output.id || output.key,
        displayName: output.displayName || output.key,
        path: output.datapointPath,
        quality: 'bad',
        value: null,
        error: tab.outputPreviewError,
      }))
    }
  } finally {
    if (requestSequence === tab.outputPreviewRequestSequence) tab.outputPreviewLoading = false
  }
}

const outputPreviewDescription = (tab: any) => {
  if (tab.outputPreviewLoading) return '正在读取数据点 GET 结果…'
  if (tab.outputPreviewSource !== 'get') {
    if (tab.outputModified)
      return '当前按本次 SQL 结果即时演算；点击“更新数据点”后自动切换为真实 GET 值。'
    return '当前按本次 SQL 结果预览；生成数据点后自动切换为真实 GET 值。'
  }
  if (tab.modified) return '当前 GET 使用已保存的 SQL，未保存的 SQL 修改尚未生效。'
  return '以下内容来自数据点 GET 接口，可用于核对脚本实际读取结果。'
}

const previewQualityType = (quality: string) => {
  if (quality === 'good') return 'success'
  if (quality === 'bad') return 'danger'
  if (quality === 'preview') return 'info'
  return 'warning'
}

const formatOutputPreview = (value: unknown) => {
  let text = ''
  try {
    text = JSON.stringify(value, null, 2)
  } catch {
    text = String(value)
  }
  if (text === undefined) text = String(value)
  const limit = 4000
  return text.length > limit ? `${text.slice(0, limit)}\n…预览内容已截断` : text
}

const saveDirtyQueries = async (): Promise<boolean> => {
  for (const tab of tabs.value.filter((item) => item.type === 'query' && item.modified)) {
    if (!(await saveQueryTab(tab))) return false
  }
  return true
}

const openHistory = (record: any) => {
  createQueryTab({
    title: generatedQueryCopyName(record.title),
    sql: record.sql,
  })
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

watch(filterText, () => {
  window.clearTimeout(querySearchTimer)
  querySearchTimer = window.setTimeout(() => {
    void loadQueries(1)
    if (isReadOnlySource.value) void loadTables(1)
  }, 250)
})

onMounted(async () => {
  loadExecutionHistory()
  await reloadExplorer()
  createQueryTab()
  unregisterDraftChecker = registerDraftChecker?.({
    isDirty: () => tabs.value.some((tab) => tab.type === 'query' && tab.modified),
    save: saveDirtyQueries,
    discard: () => tabs.value.forEach((tab) => (tab.modified = false)),
  })
})

onBeforeUnmount(() => {
  unregisterDraftChecker?.()
  window.clearTimeout(querySearchTimer)
  stopResultResize()
})
</script>

<style scoped>
.sql-workbench {
  position: relative;
  height: 100%;
  min-height: 0;
  display: grid;
  grid-template-columns: 260px minmax(0, 1fr) minmax(300px, 320px);
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
  min-width: 0;
  min-height: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 16px;
  overflow-x: hidden;
  overflow-y: auto;
  padding: 14px 14px 10px;
}

.sql-workbench__inspector-switch {
  flex-shrink: 0;
  display: grid;
  grid-template-columns: 1fr 1fr;
  padding: 3px;
  border-radius: 7px;
  background: var(--dc-surface);
}

.sql-workbench__inspector-switch button {
  min-width: 0;
  height: 30px;
  border: 0;
  border-radius: 5px;
  background: transparent;
  color: var(--dc-text-secondary);
  font-size: 12px;
  cursor: pointer;
}

.sql-workbench__inspector-switch button.is-active {
  background: var(--dc-surface-raised);
  color: var(--dc-primary);
  font-weight: 700;
  box-shadow: var(--dc-shadow-surface);
}

.sql-workbench__inspector-switch button:disabled {
  cursor: not-allowed;
  opacity: 0.45;
}

.sql-workbench__output-preview {
  display: grid;
  gap: 10px;
}

.sql-workbench__output-preview-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
}

.sql-workbench__output-preview-head > div {
  min-width: 0;
  display: grid;
  gap: 3px;
}

.sql-workbench__output-preview-head > .sql-workbench__output-preview-actions {
  flex: none;
  display: flex;
  align-items: center;
  gap: 4px;
}

.sql-workbench__output-preview-head strong,
.sql-workbench__output-preview-item strong {
  color: var(--dc-text);
  font-size: 13px;
}

.sql-workbench__output-preview-head span {
  color: var(--dc-text-secondary);
  font-size: 11px;
  line-height: 1.5;
}

.sql-workbench__output-preview-item {
  min-width: 0;
  display: grid;
  gap: 7px;
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
}

.sql-workbench__output-preview-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.sql-workbench__output-preview-item code {
  overflow: hidden;
  color: var(--dc-primary);
  font-family: var(--dc-font-mono);
  font-size: 10px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sql-workbench__output-preview-item small {
  color: var(--el-color-danger);
  font-size: 11px;
}

.sql-workbench__output-preview-item pre {
  max-height: 260px;
  margin: 0;
  overflow: auto;
  padding: 9px;
  border-radius: 5px;
  background: var(--dc-surface-muted);
  color: var(--dc-text);
  font-family: var(--dc-font-mono);
  font-size: 11px;
  line-height: 1.55;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.sql-workbench__history-panel {
  min-height: 0;
  flex: 0 0 min(220px, 34%);
  display: flex;
  flex-direction: column;
  padding: 12px 14px 14px;
  border-top: 1px solid var(--dc-border);
}

.sql-workbench__inspector-mobile-head,
.sql-workbench__inspector-toggle {
  display: none;
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

.sql-workbench__load-more {
  width: calc(100% - 8px);
  min-height: 30px;
  margin: 6px 4px 2px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
  color: var(--dc-primary);
  cursor: pointer;
  font: inherit;
  font-size: 12px;
}

.sql-workbench__load-more:hover:not(:disabled) {
  border-color: var(--dc-primary);
  background: var(--dc-primary-soft);
}

.sql-workbench__load-more:disabled {
  cursor: wait;
  opacity: 0.65;
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

.sql-workbench__result-warning {
  margin: 0 12px 8px;
  flex: none;
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

.sql-workbench__timeseries-facts {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  margin: 0;
}

.sql-workbench__timeseries-facts div {
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
}

.sql-workbench__timeseries-facts dt {
  margin-bottom: 4px;
  color: var(--dc-text-muted);
  font-size: 11px;
}

.sql-workbench__timeseries-facts dd {
  margin: 0;
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 700;
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

.sql-workbench__context-menu button:disabled {
  cursor: not-allowed;
  opacity: 0.45;
}

.sql-workbench__context-menu button:disabled:hover {
  background: transparent;
  color: var(--dc-text-secondary);
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
    position: absolute;
    top: 0;
    right: 0;
    bottom: 0;
    z-index: 8;
    width: min(360px, calc(100% - 24px));
    border-left: 1px solid var(--dc-border);
    box-shadow: var(--dc-shadow-popover);
    transform: translateX(102%);
    transition: transform 160ms ease;
  }

  .sql-workbench__inspector.is-open {
    transform: translateX(0);
  }

  .sql-workbench__inspector-mobile-head {
    min-height: 44px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 14px;
    border-bottom: 1px solid var(--dc-border);
    background: var(--dc-surface-raised);
  }

  .sql-workbench__inspector-mobile-head strong {
    color: var(--dc-text);
    font-size: 13px;
  }

  .sql-workbench__inspector-mobile-head button {
    width: 28px;
    height: 28px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding: 0;
    border: 0;
    border-radius: var(--dc-radius-sm);
    background: transparent;
    color: var(--dc-text-secondary);
    cursor: pointer;
  }

  .sql-workbench__inspector-mobile-head button:hover {
    background: var(--dc-primary-soft);
    color: var(--dc-primary);
  }

  .sql-workbench__inspector-mobile-head svg {
    width: 16px;
    height: 16px;
  }

  .sql-workbench__inspector-toggle {
    display: inline-flex;
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
