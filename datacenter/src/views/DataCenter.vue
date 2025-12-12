<template>
  <div class="data-center h-full flex overflow-hidden">
    <!-- 左侧连接面板 -->
    <div
      class="w-64 bg-white dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700 flex flex-col overflow-hidden">
      <div class="p-4 flex-1 overflow-y-auto min-h-0">
        <div class="space-y-2">
          <!-- 标题和操作按钮 -->
          <div class="flex items-center justify-between mb-3">
            <h3 class="text-sm font-medium text-gray-900 dark:text-white">数据连接</h3>
            <div class="flex items-center space-x-2">
              <el-button type="primary" size="small" @click="openCreateConnectionDialog">
                <el-icon class="mr-1">
                  <Plus />
                </el-icon>
                新建
              </el-button>
              <el-button size="small" circle @click="refreshConnections"
                class="!border-gray-300 dark:!border-gray-600 hover:!bg-gray-50 dark:hover:!bg-gray-700">
                <el-icon>
                  <Refresh />
                </el-icon>
              </el-button>
            </div>
          </div>

          <div class="connection-list">
            <div v-for="connection in connections" :key="connection.id" :class="[
              'connection-item p-3 rounded-lg cursor-pointer transition-colors',
              selectedConnectionId === connection.id
                ? 'bg-blue-100 dark:bg-blue-900/30 border-blue-300 dark:border-blue-600'
                : 'hover:bg-gray-100 dark:hover:bg-gray-700'
            ]" @click="selectConnection(connection)" @dblclick.stop="handleConnectionDblClick(connection)"
              @contextmenu.prevent="handleConnectionContextMenu($event, connection)">
              <!-- 连接图标 -->
              <div class="flex items-center justify-between">
                <div class="flex items-center flex-1">
                  <div class="w-6 h-6 mr-3 flex-shrink-0">
                    <svg v-if="connection.type === 'relational'" class="w-6 h-6 text-blue-500" fill="none"
                      viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                        d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4" />
                    </svg>
                  </div>

                  <!-- 连接信息 -->
                  <div class="flex-1 min-w-0">
                    <div class="text-sm font-medium text-gray-900 dark:text-white truncate">
                      {{ connection.name }}
                    </div>
                    <div class="text-xs text-gray-500 dark:text-gray-400">
                      {{ getConnectionTypeLabel(connection.type) }}
                    </div>
                  </div>
                </div>

                <!-- 状态指示器 -->
                <div class="flex-shrink-0 flex items-center space-x-1">
                  <span :class="[
                    'w-2 h-2 rounded-full',
                    connection.status === 'connected' ? 'bg-green-500' : 
                    connection.status === 'error' ? 'bg-red-500' : 
                    connection.status === 'disconnected' ? 'bg-gray-400' : 'bg-gray-400'
                  ]" :title="
                    connection.status === 'connected' ? '已连接' : 
                    connection.status === 'error' ? '连接错误' : 
                    connection.status === 'disconnected' ? '已断开' : '未知状态'
                  "></span>
                  <el-icon class="text-gray-400" v-if="connection.type === 'relational'">
                    <component :is="getConnectionState(connection.id).expanded ? 'CaretBottom' : 'CaretRight'" />
                  </el-icon>
                </div>
              </div>

              <div v-if="connection.type === 'relational' && getConnectionState(connection.id).expanded"
                class="mt-2 space-y-3">
                <div class="space-y-1">
                  <div
                    class="flex items-center justify-between text-xs font-medium text-gray-600 dark:text-gray-300 cursor-pointer"
                    @click="toggleConnectionSection(connection.id, 'tables')">
                    <div class="flex items-center space-x-2">
                      <el-icon class="text-gray-400">
                        <component
                          :is="getConnectionState(connection.id).tablesExpanded ? 'CaretBottom' : 'CaretRight'" />
                      </el-icon>
                      <span>表</span>
                    </div>
                    <span v-if="!getConnectionState(connection.id).loading" class="text-gray-400">
                      {{ getConnectionState(connection.id).tables.length }}
                    </span>
                  </div>
                  <div v-if="getConnectionState(connection.id).loading"
                    class="pl-6 text-xs text-gray-500 dark:text-gray-400">
                    加载表列表...
                  </div>
                  <template v-else-if="getConnectionState(connection.id).tablesExpanded">
                    <div v-if="getConnectionState(connection.id).tables.length === 0"
                      class="pl-6 text-xs text-gray-500 dark:text-gray-400">
                      暂无表
                    </div>
                    <ul v-else class="space-y-1 pl-6 border-l border-gray-200 dark:border-gray-700">
                      <li v-for="table in getConnectionState(connection.id).tables" :key="table.name"
                        class="flex items-center justify-between text-xs text-gray-700 dark:text-gray-300 hover:text-blue-600 dark:hover:text-blue-400 cursor-pointer"
                        @dblclick.stop="handleLeftTableDblClick(connection, table)">
                        <span class="truncate">{{ table.name }}</span>
                        <span class="text-gray-400">{{ table.rows }}</span>
                      </li>
                    </ul>
                  </template>
                </div>

                <div class="space-y-1">
                  <div
                    class="flex items-center justify-between text-xs font-medium text-gray-600 dark:text-gray-300 cursor-pointer"
                    @click="toggleConnectionSection(connection.id, 'queries')">
                    <div class="flex items-center space-x-2">
                      <el-icon class="text-gray-400">
                        <component
                          :is="getConnectionState(connection.id).queriesExpanded ? 'CaretBottom' : 'CaretRight'" />
                      </el-icon>
                      <span>查询</span>
                    </div>
                    <span class="text-gray-400">{{ getConnectionQueries(connection.id).length }}</span>
                  </div>
                  <template v-if="getConnectionState(connection.id).queriesExpanded">
                    <div v-if="getConnectionQueries(connection.id).length === 0"
                      class="pl-6 text-xs text-gray-500 dark:text-gray-400">
                      暂无查询
                    </div>
                    <ul v-else class="space-y-1 pl-6 border-l border-gray-200 dark:border-gray-700">
                      <li v-for="query in getConnectionQueries(connection.id)" :key="query.id"
                        class="flex items-center justify-between text-xs text-gray-700 dark:text-gray-300 hover:text-blue-600 dark:hover:text-blue-400 cursor-pointer"
                        @dblclick.stop="handleLeftQueryDblClick(connection, query)">
                        <span class="truncate">{{ query.name }}</span>
                        <el-tag size="small" class="flex-shrink-0" v-if="query.queryType">
                          {{ getQueryTypeLabel(query.queryType) }}
                        </el-tag>
                      </li>
                    </ul>
                  </template>
                </div>
              </div>

              <!-- 连接详情 -->
              <div v-if="connection.type === 'relational' && connection.relationalConfig"
                class="mt-2 text-xs text-gray-500 dark:text-gray-400">
                {{ connection.relationalConfig.dbType }} - {{ connection.relationalConfig.host }}:{{
                  connection.relationalConfig.port
                }}
              </div>
            </div>
          </div>

          <div v-if="connections.length === 0" class="text-center py-8 text-gray-500 dark:text-gray-400">
            <div class="text-sm">暂无数据连接</div>
            <el-button type="primary" size="small" @click="() => showCreateConnectionDialog = true" class="mt-2">
              创建连接
            </el-button>
          </div>
        </div>
      </div>
    </div>

    <!-- 右侧内容区域 -->
    <div class="flex-1 flex flex-col bg-gray-50 dark:bg-gray-900 overflow-hidden">
      <!-- 主体内容区 -->
      <div class="flex-1 overflow-hidden min-h-0">
        <!-- 标签页区域 -->
        <div v-if="queryTabs.length > 0" class="flex-1 flex flex-col overflow-hidden">
          <el-tabs v-model="activeQueryTab" type="card" closable class="query-tabs flex-1 flex flex-col overflow-hidden"
            @tab-remove="removeQueryTab" @tab-change="onQueryTabChange">
            <el-tab-pane v-for="tab in queryTabs" :key="tab.id" :label="tab.name" :name="tab.id" :lazy="false"
              class="flex-1 flex flex-col overflow-hidden">
              <template #label>
                <span class="flex items-center">
                  <span>{{ tab.name }}</span>
                  <el-icon v-if="tab.modified" class="ml-1 text-orange-500" style="font-size: 12px;">
                    <Warning />
                  </el-icon>
                </span>
              </template>

              <!-- 查询编辑器内容 -->
              <div class="query-editor-tab-content flex-1 flex flex-col overflow-y-auto p-4">
                <!-- 工具栏 -->
                <div class="flex items-center justify-between mb-3 pb-3 border-b border-gray-200 dark:border-gray-700">
                  <div class="flex items-center space-x-3">
                    <el-select v-model="tab.connectionId" placeholder="请选择数据连接" size="small" class="w-48"
                      @change="(val) => onQueryTabConnectionChange(tab.id, val)">
                      <el-option v-for="conn in relationalConnections" :key="conn.id" :label="conn.name"
                        :value="conn.id" />
                    </el-select>
                    <el-select v-model="tab.table" :placeholder="tab.connectionId ? '选择表' : '请先选择数据连接'" size="small"
                      class="w-48" :disabled="!tab.connectionId" filterable
                      @change="(val) => onQueryTabTableChange(tab.id, val)">
                      <el-option v-for="table in tab.tables" :key="table.name" :label="table.name"
                        :value="table.name" />
                    </el-select>
                  </div>
                  <div class="flex items-center space-x-2">
                    <el-button size="small" @click="() => formatSqlInTab(tab.id)">
                      <el-icon class="mr-1">
                        <DocumentChecked />
                      </el-icon>
                      美化SQL
                    </el-button>
                    <el-button type="primary" size="small" @click="() => executeSqlInTab(tab.id)"
                      :loading="tab.executing">
                      <el-icon class="mr-1">
                        <VideoPlay />
                      </el-icon>
                      运行
                    </el-button>
                    <el-button size="small" @click="() => saveQueryTab(tab.id)" :loading="tab.saving">
                      保存
                    </el-button>
                  </div>
                </div>

                <!-- SQL编辑器 -->
                <div class="flex-shrink-0 border border-gray-200 dark:border-gray-700 rounded overflow-hidden mb-4"
                  style="min-height: 280px; height: 300px;">
                  <MonacoEditor v-if="activeQueryTab === tab.id" :ref="el => setMonacoEditorRef(el, tab.id)"
                    v-model="tab.sql" language="sql" :theme="isDark ? 'vs-dark' : 'vs'" height="300px" :options="{
                      minimap: { enabled: true },
                      fontSize: 14,
                      wordWrap: 'on',
                      formatOnPaste: true,
                      formatOnType: true,
                      readOnly: false
                    }" @change="(value) => { tab.sql = value; tab.modified = true; updateParametersFromSql(tab.id) }" />
                  <div v-else class="w-full h-full flex items-center justify-center">
                    <span class="text-gray-400">切换到该标签页以加载编辑器</span>
                  </div>
                </div>

                <!-- 参数输入面板 -->
                <div v-if="tab.parameters && tab.parameters.length > 0" 
                  class="mb-4 border border-gray-200 dark:border-gray-700 rounded p-3 bg-gray-50 dark:bg-gray-800">
                  <div class="flex items-center justify-between mb-2">
                    <div class="flex items-center space-x-2">
                      <el-icon class="text-gray-500"><List /></el-icon>
                      <span class="text-sm font-medium text-gray-700 dark:text-gray-300">SQL 参数</span>
                      <el-tag size="small" type="info">{{ tab.parameters.length }} 个参数</el-tag>
                    </div>
                    <el-button 
                      size="small" 
                      text 
                      @click="tab.parametersExpanded = !tab.parametersExpanded"
                    >
                      <el-icon class="mr-1">
                        <component :is="tab.parametersExpanded ? 'CaretBottom' : 'CaretRight'" />
                      </el-icon>
                      {{ tab.parametersExpanded ? '收起' : '展开' }}
                    </el-button>
                  </div>
                  
                  <div v-show="tab.parametersExpanded" class="space-y-2 mt-2">
                    <div 
                      v-for="(param, index) in tab.parameters" 
                      :key="index"
                      class="flex items-center space-x-2"
                    >
                      <div class="w-24 text-sm text-gray-600 dark:text-gray-400 flex-shrink-0">
                        参数 {{ index + 1 }}:
                      </div>
                      <el-input 
                        v-model="param.value" 
                        :placeholder="`请输入参数 ${index + 1} 的值`"
                        size="small"
                        class="flex-1"
                        @input="tab.modified = true"
                      >
                        <template #prepend>
                          <span class="text-xs text-gray-500">?</span>
                        </template>
                      </el-input>
                      <el-button 
                        size="small" 
                        text 
                        type="danger"
                        @click="clearParameter(tab.id, index)"
                        title="清空"
                      >
                        <el-icon><Delete /></el-icon>
                      </el-button>
                    </div>
                  </div>
                </div>

                <!-- 查询结果 -->
                <div v-if="tab.result" class="flex-shrink-0 flex flex-col border-t border-gray-200 dark:border-gray-700 pt-4 h-full"
                  style="min-height: 380px;">
                  <div class="flex items-center justify-between mb-2 flex-shrink-0">
                    <div class="text-sm text-gray-600 dark:text-gray-400">
                      查询结果 ({{ tab.result.rowCount }} 行，耗时 {{ tab.result.executionTime }}ms)
                    </div>
                    <el-button size="small" @click="() => clearQueryResult(tab.id)">关闭</el-button>
                  </div>
                  <div class="flex-1 flex flex-col border border-gray-200 dark:border-gray-700 rounded min-h-0">
                    <div class="flex-1 overflow-auto min-h-0">
                      <el-table :data="getPaginatedResultRows(tab)" size="small" border stripe>
                        <el-table-column v-for="(column, index) in tab.result.columns" :key="index"
                          :prop="index.toString()" :label="column" min-width="120" show-overflow-tooltip />
                      </el-table>
                    </div>
                    <div
                      class="flex items-center justify-between px-4 py-2 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 flex-shrink-0">
                      <div class="text-sm text-gray-600 dark:text-gray-400">
                        显示第 {{ (getResultPage(tab) - 1) * getResultPageSize(tab) + 1 }} - {{ Math.min(getResultPage(tab)
                          *
                        getResultPageSize(tab), tab.result.rowCount) }} 条，共 {{ tab.result.rowCount }} 条
                      </div>
                      <el-pagination v-model:current-page="tab.resultPage" v-model:page-size="tab.resultPageSize"
                        :page-sizes="[10, 20, 50, 100, 200]" :total="tab.result.rowCount"
                        layout="sizes, prev, pager, next" small @size-change="() => { tab.resultPage = 1 }" />
                    </div>
                  </div>
                </div>
              </div>
            </el-tab-pane>
          </el-tabs>
        </div>

        <!-- 默认视图：表列表或表数据 -->
        <div v-else class="h-full flex flex-col overflow-hidden">
          <div
            class="table-toolbar flex items-center justify-between px-4 py-3 border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800">
            <div class="text-sm text-gray-700 dark:text-gray-300">
              {{ formatActiveTableTitle }}
            </div>
            <div class="space-x-2">
              <el-button type="primary" size="small" @click="createNewQuery">
                <el-icon class="mr-1">
                  <Plus />
                </el-icon>
                新建查询
              </el-button>
            </div>
          </div>
          <div class="flex-1 overflow-y-auto p-4 min-h-0">
            <div v-if="tableLoading" class="h-full flex items-center justify-center text-gray-500 dark:text-gray-400">
              <el-icon class="mr-2">
                <Loading />
              </el-icon>数据加载中...
            </div>
            <div v-else-if="!currentTableListConnectionId"
              class="h-full flex items-center justify-center text-gray-500 dark:text-gray-400">
              请选择数据连接以查看表和数据
            </div>
            <div v-else-if="!isShowingTableData">
              <div v-if="isTableListLoading"
                class="h-full flex items-center justify-center text-gray-500 dark:text-gray-400">
                <el-icon class="mr-2">
                  <Loading />
                </el-icon>表列表加载中...
              </div>
              <div v-else>
                <el-table v-if="currentTables.length > 0" :data="currentTables" size="small" border class="table-list"
                  @row-dblclick="handleTableRowDblClick">
                  <el-table-column prop="name" label="表名" min-width="160" show-overflow-tooltip />
                  <el-table-column prop="comment" label="备注" min-width="200" show-overflow-tooltip />
                  <el-table-column prop="rows" label="记录数" width="120" align="right" />
                </el-table>
                <div v-else class="text-center py-12 text-gray-500 dark:text-gray-400">
                  暂无表信息
                </div>
              </div>
            </div>
            <div v-else>
              <div v-if="tableColumns.length === 0" class="text-center py-12 text-gray-500 dark:text-gray-400">
                未查询到数据
              </div>
              <div v-else class="flex flex-col h-full">
                <el-table :data="tableData" size="small" border class="flex-1 overflow-auto">
                  <el-table-column v-for="column in tableColumns" :key="column" :prop="column" :label="column"
                    min-width="140" show-overflow-tooltip />
                </el-table>
                <div class="flex justify-end mt-4">
                  <el-pagination background layout="prev, pager, next, jumper" :current-page="tablePagination.page"
                    :page-size="tablePagination.limit" :total="tablePagination.total"
                    @current-change="handleTablePageChange" />
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 右键菜单 -->
    <div v-show="showContextMenu" 
      :style="{ position: 'fixed', left: contextMenuPosition.x + 'px', top: contextMenuPosition.y + 'px', zIndex: 9999 }"
      class="bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 py-1 min-w-[150px]">
      <div @click="openConnection" 
        class="px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer flex items-center">
        <el-icon class="mr-2">
          <Connection />
        </el-icon>
        打开连接
      </div>
      <div @click="disconnectConnection" 
        class="px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer flex items-center">
        <el-icon class="mr-2">
          <Close />
        </el-icon>
        断开连接
      </div>
      <div class="border-t border-gray-200 dark:border-gray-700 my-1"></div>
      <div @click="viewConnectionDetails" 
        class="px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer flex items-center">
        <el-icon class="mr-2">
          <View />
        </el-icon>
        查看详情
      </div>
      <div @click="editConnection" 
        class="px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer flex items-center">
        <el-icon class="mr-2">
          <Edit />
        </el-icon>
        编辑连接
      </div>
      <div class="border-t border-gray-200 dark:border-gray-700 my-1"></div>
      <div @click="deleteConnection" 
        class="px-4 py-2 text-sm text-red-600 dark:text-red-400 hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer flex items-center">
        <el-icon class="mr-2">
          <Delete />
        </el-icon>
        删除连接
      </div>
    </div>

    <!-- 新建/编辑连接对话框 -->
    <el-dialog v-model="showConnectionDialog" :title="connectionDialogMode === 'create' ? '新建数据连接' : '编辑数据连接'"
      width="600px" :close-on-click-modal="false">
      <el-form ref="connectionFormRef" :model="connectionForm" :rules="connectionFormRules" label-width="120px">
        <el-form-item label="连接名称" prop="name">
          <el-input v-model="connectionForm.name" placeholder="请输入连接名称" />
        </el-form-item>

        <el-form-item label="连接类型" prop="type">
          <el-select v-model="connectionForm.type" placeholder="选择连接类型" class="w-full">
            <el-option label="关系数据库" value="relational" />
          </el-select>
        </el-form-item>

        <el-form-item label="数据库类型" prop="dbType">
          <el-select v-model="connectionForm.dbType" placeholder="选择数据库类型" class="w-full">
            <el-option label="MySQL" value="mysql" />
            <el-option label="PostgreSQL" value="postgresql" />
            <el-option label="SQL Server" value="sqlserver" />
          </el-select>
        </el-form-item>

        <el-form-item label="主机地址" prop="host">
          <el-input v-model="connectionForm.host" placeholder="localhost" />
        </el-form-item>

        <el-form-item label="端口" prop="port">
          <el-input-number v-model="connectionForm.port" :min="1" :max="65535" class="w-full" />
        </el-form-item>

        <el-form-item label="数据库名" prop="database">
          <el-input v-model="connectionForm.database" placeholder="请输入数据库名" />
        </el-form-item>

        <el-form-item label="用户名" prop="username">
          <el-input v-model="connectionForm.username" placeholder="root" />
        </el-form-item>

        <el-form-item label="密码" prop="password">
          <el-input v-model="connectionForm.password" type="password" placeholder="请输入密码" />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="showConnectionDialog = false">取消</el-button>
        <el-button @click="testConnection" :loading="testingConnection">测试连接</el-button>
        <el-button type="primary" @click="saveConnection" :loading="creatingConnection">
          {{ connectionDialogMode === 'create' ? '创建连接' : '保存修改' }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 查看连接详情对话框 -->
    <el-dialog v-model="showViewConnectionDialog" title="连接详情" width="600px">
      <el-descriptions :column="1" border v-if="currentConnection">
        <el-descriptions-item label="连接名称">{{ currentConnection.name }}</el-descriptions-item>
        <el-descriptions-item label="连接类型">{{ getConnectionTypeLabel(currentConnection.type) }}</el-descriptions-item>
        <el-descriptions-item label="连接状态">
          <el-tag :type="
            currentConnection.status === 'connected' ? 'success' : 
            currentConnection.status === 'error' ? 'danger' : 
            currentConnection.status === 'disconnected' ? 'warning' : 'info'
          ">
            {{ 
              currentConnection.status === 'connected' ? '已连接' : 
              currentConnection.status === 'error' ? '连接错误' : 
              currentConnection.status === 'disconnected' ? '已断开' : '未知状态'
            }}
          </el-tag>
        </el-descriptions-item>
        <template v-if="currentConnection.type === 'relational' && currentConnection.relationalConfig">
          <el-descriptions-item label="数据库类型">{{ currentConnection.relationalConfig.dbType }}</el-descriptions-item>
          <el-descriptions-item label="主机地址">{{ currentConnection.relationalConfig.host }}</el-descriptions-item>
          <el-descriptions-item label="端口">{{ currentConnection.relationalConfig.port }}</el-descriptions-item>
          <el-descriptions-item label="数据库名">{{ currentConnection.relationalConfig.database }}</el-descriptions-item>
          <el-descriptions-item label="用户名">{{ currentConnection.relationalConfig.username }}</el-descriptions-item>
        </template>
        <el-descriptions-item label="创建时间">{{ formatDate(currentConnection.createdAt) }}</el-descriptions-item>
        <el-descriptions-item label="更新时间">{{ formatDate(currentConnection.updatedAt) }}</el-descriptions-item>
      </el-descriptions>
      <template #footer>
        <el-button @click="showViewConnectionDialog = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed, watch, nextTick, onBeforeUnmount } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Refresh, VideoPlay, DocumentChecked, CaretRight, CaretBottom, Loading, Warning, List, Delete, View, Edit, Close, Connection } from '@element-plus/icons-vue'
import { format } from 'sql-formatter'
import MonacoEditor from '@/components/MonacoEditor.vue'
import dataAPI from '@/api/data.api'

// 从路由获取 project 信息
const route = useRoute()
const project = computed(() => {
  // 优先从 URL 参数获取（支持 pid 和 id，兼容旧版本）
  const projectId = route.query.pid || route.query.id || route.meta?.project?.id
  const tenantId = route.query.tenant || route.meta?.project?.tenantId
  if (projectId) {
    return { id: projectId, tenantId: tenantId }
  }
  return null
})

// 状态管理
const connections = ref([])
const tablesByConnection = ref({})
const queries = ref([])
const selectedConnectionId = ref('')
const currentTableListConnectionId = ref(null)
const activeTable = ref(null)
const tableColumns = ref([])
const tableData = ref([])
const tableLoading = ref(false)
const tablePagination = reactive({
  page: 1,
  limit: 100,
  total: 0,
  totalPages: 0
})
const activeTab = ref('new-query')
const showConnectionDialog = ref(false)
const showViewConnectionDialog = ref(false)
const connectionDialogMode = ref('create') // 'create' | 'edit'
const currentConnection = ref(null)
const creatingConnection = ref(false)
const testingConnection = ref(false)

// 右键菜单相关
const showContextMenu = ref(false)
const contextMenuPosition = reactive({ x: 0, y: 0 })
const contextMenuConnection = ref(null)
const contextMenuRef = ref(null)
const executingQuery = ref(null)
const queryResults = ref({})
const queryParameters = ref({})

// 查询标签页相关
const queryTabs = ref([])
const activeQueryTab = ref('')
const monacoEditors = ref({}) // 存储每个标签页的 Monaco Editor 实例引用
let queryTabCounter = 0

// 检查是否为暗色主题
const isDark = computed(() => {
  return document.documentElement.classList.contains('dark')
})

// 连接表单
const connectionForm = reactive({
  name: '',
  type: 'relational',
  dbType: 'mysql',
  host: 'localhost',
  port: 3306,
  database: '',
  username: 'root',
  password: ''
})

const connectionFormRules = {
  name: [
    { required: true, message: '请输入连接名称', trigger: 'blur' },
    { min: 2, max: 100, message: '连接名称长度在 2 到 100 个字符', trigger: 'blur' }
  ],
  type: [
    { required: true, message: '请选择连接类型', trigger: 'change' }
  ],
  dbType: [
    { required: true, message: '请选择数据库类型', trigger: 'change' }
  ],
  host: [
    { required: true, message: '请输入主机地址', trigger: 'blur' }
  ],
  port: [
    { required: true, message: '请输入端口号', trigger: 'blur' }
  ],
  database: [
    { required: true, message: '请输入数据库名', trigger: 'blur' }
  ],
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' }
  ]
}

const connectionFormRef = ref(null)

// 获取连接类型标签
const getConnectionTypeLabel = (type) => {
  const labels = {
    relational: '关系数据库',
    mqtt: 'MQTT',
    websocket: 'WebSocket',
    opcua: 'OPC UA',
    http: 'HTTP'
  }
  return labels[type] || type
}

// 获取连接名称
const getConnectionName = (connectionId) => {
  const connection = connections.value.find(c => c.id === connectionId)
  return connection ? connection.name : '未知连接'
}

const getConnectionState = (connectionId) => {
  if (!tablesByConnection.value[connectionId]) {
    tablesByConnection.value[connectionId] = {
      expanded: false,
      loading: false,
      tables: [],
      tablesExpanded: true,
      queriesExpanded: true
    }
  }
  return tablesByConnection.value[connectionId]
}

const currentConnectionState = computed(() => {
  if (!currentTableListConnectionId.value) return null
  return getConnectionState(currentTableListConnectionId.value)
})

const currentTables = computed(() => currentConnectionState.value?.tables || [])
const isTableListLoading = computed(() => currentConnectionState.value?.loading || false)
const isShowingTableData = computed(() => !!activeTable.value)

const queriesByConnection = computed(() => {
  return queries.value.reduce((acc, query) => {
    if (!query?.connectionId) return acc
    if (!acc[query.connectionId]) {
      acc[query.connectionId] = []
    }
    acc[query.connectionId].push(query)
    return acc
  }, {})
})

const getConnectionQueries = (connectionId) => queriesByConnection.value[connectionId] || []

const toggleConnectionSection = (connectionId, section) => {
  const state = getConnectionState(connectionId)
  if (section === 'tables') {
    state.tablesExpanded = !state.tablesExpanded
  } else if (section === 'queries') {
    state.queriesExpanded = !state.queriesExpanded
  }
}

const getQueryTypeLabel = (type) => {
  const labels = {
    sql: 'SQL 查询',
    http_get: 'HTTP GET',
    http_post: 'HTTP POST',
    http_put: 'HTTP PUT',
    http_delete: 'HTTP DELETE',
    http_patch: 'HTTP PATCH',
    mqtt_publish: 'MQTT 发布',
    mqtt_subscribe: 'MQTT 订阅',
    websocket: 'WebSocket',
    opcua_read: 'OPC UA 读取',
    opcua_write: 'OPC UA 写入',
    opcua_browse: 'OPC UA 浏览'
  }
  return labels[type] || type
}

const resetTableView = () => {
  activeTable.value = null
  tableColumns.value = []
  tableData.value = []
  tablePagination.page = 1
  tablePagination.total = 0
  tablePagination.totalPages = 0
}

const getConnectionById = (connectionId) => connections.value.find((item) => item.id === connectionId)

const formatActiveTableTitle = computed(() => {
  if (!activeTable.value) {
    const connection = getConnectionById(currentTableListConnectionId.value || '')
    return connection ? `${connection.name} - 表列表` : '请选择数据连接'
  }
  const connection = getConnectionById(activeTable.value.connectionId)
  const table = activeTable.value.tableName
  const total = tablePagination.total
  const connectionName = connection ? connection.name : '未知连接'
  return `${connectionName} / ${table} （共 ${total} 行）`
})

const toggleConnectionTables = async (connection) => {
  const state = getConnectionState(connection.id)
  if (!state.expanded && state.tables.length === 0) {
    state.loading = true
    try {
      const response = await dataAPI.getConnectionTables(project.value.id, connection.id)
      if (response.success) {
        state.tables = response.data.tables || []
      } else {
        ElMessage.error(response.message || '获取表列表失败')
      }
    } catch (error) {
      ElMessage.error('获取表列表失败：' + (error.response?.data?.message || error.message))
    } finally {
      state.loading = false
    }
  }
  state.expanded = !state.expanded
  if (state.expanded) {
    currentTableListConnectionId.value = connection.id
    state.tablesExpanded = true
  } else if (currentTableListConnectionId.value === connection.id) {
    currentTableListConnectionId.value = null
  }
  resetTableView()
}

const handleTableSelect = (connectionId, tableName) => {
  loadTableData(connectionId, tableName, 1)
}

const handleTableRowDblClick = (row) => {
  if (!currentTableListConnectionId.value || !row?.name) return
  handleTableSelect(currentTableListConnectionId.value, row.name)
}

const loadTableData = async (connectionId, tableName, page = 1) => {
  tableLoading.value = true
  activeTable.value = { connectionId, tableName }
  try {
    const response = await dataAPI.getTableData(project.value.id, connectionId, tableName, {
      page,
      limit: tablePagination.limit
    })
    if (response.success) {
      tableColumns.value = response.data.columns || []
      tableData.value = response.data.rows || []
      const pagination = response.data.pagination || {}
      tablePagination.page = pagination.page || 1
      tablePagination.limit = pagination.limit || 100
      tablePagination.total = pagination.total || 0
      tablePagination.totalPages = pagination.totalPages || 0
    } else {
      ElMessage.error(response.message || '获取表数据失败')
    }
  } catch (error) {
    ElMessage.error('获取表数据失败：' + (error.response?.data?.message || error.message))
  } finally {
    tableLoading.value = false
  }
}

const handleTablePageChange = (page) => {
  if (!activeTable.value) return
  loadTableData(activeTable.value.connectionId, activeTable.value.tableName, page)
}

// 加载数据连接
const loadConnections = async () => {
  try {
    if (!project.value?.id) {
      return
    }

    const response = await dataAPI.getConnections(project.value.id)
    if (response.success) {
      connections.value = response.data.connections || []
    }
  } catch (error) {
    console.error('DataCenter: Load connections error:', error)
    ElMessage.error('加载数据连接失败：' + (error.response?.data?.message || error.message))
  }
}

// 加载数据查询
const loadQueries = async () => {
  try {
    if (!project.value?.id) return

    const response = await dataAPI.getQueries(project.value.id)
    if (response.success) {
      queries.value = response.data.queries
    }
  } catch (error) {
    ElMessage.error('加载数据查询失败：' + (error.response?.data?.message || error.message))
  }
}

// 选择连接
const selectConnection = (connection) => {
  selectedConnectionId.value = connection.id
  currentTableListConnectionId.value = connection.id
  resetTableView()
}

// 连接变化处理
const onConnectionChange = (connectionId) => {
  const connection = connections.value.find(c => c.id === connectionId)
  if (connection) {
    selectConnection(connection)
  }
}

const refreshConnections = () => {
  loadConnections()
}

const handleConnectionDblClick = async (connection) => {
  // 测试连接并更新状态
  try {
    if (connection.type === 'relational' && connection.relationalConfig) {
      const config = connection.relationalConfig
      const response = await dataAPI.testConnection(project.value.id, {
        type: connection.type,
        config: {
          dbType: config.dbType,
          host: config.host,
          port: config.port,
          database: config.database,
          username: config.username,
          password: config.password
        }
      })

      if (response.success) {
        // 更新连接状态为已连接
        await dataAPI.updateConnectionStatus(project.value.id, connection.id, 'connected')
        ElMessage.success('连接测试成功')
        // 刷新连接列表以显示最新状态
        await loadConnections()
      }
    }
  } catch (error) {
    // 更新连接状态为错误
    await dataAPI.updateConnectionStatus(project.value.id, connection.id, 'error')
    ElMessage.error('连接测试失败：' + (error.response?.data?.message || error.message))
    // 刷新连接列表以显示最新状态
    await loadConnections()
    return
  }

  // 测试成功后，展开表列表
  toggleConnectionTables(connection)
}

const handleLeftTableDblClick = async (connection, table) => {
  // 如果有查询标签页，创建新标签页并自动查询表数据
  if (relationalConnections.value.length > 0) {
    queryTabCounter++
    const tabId = `query-${queryTabCounter}`
    const tableName = table.name
    const sql = `SELECT * FROM \`${tableName}\` LIMIT 100`

    const newTab = {
      id: tabId,
      name: `${connection.name} - ${tableName}`,
      connectionId: connection.id,
      table: tableName,
      tables: [],
      sql: sql,
      result: null,
      executing: false,
      saving: false,
      modified: false,
      resultPage: 1,
      resultPageSize: 20,
      parameters: [],
      parametersExpanded: true,
      autoExecute: true // 标记需要自动执行
    }

    queryTabs.value.push(newTab)
    activeQueryTab.value = tabId

    // 加载表列表
    try {
      const response = await dataAPI.getConnectionTables(project.value.id, connection.id)
      if (response.success) {
        newTab.tables = response.data.tables || []
      }
    } catch (error) {
      console.error('加载表列表失败:', error)
    }

    // 等待 Monaco Editor 初始化后自动执行 SQL
    nextTick(() => {
      const checkEditor = () => {
        const editor = monacoEditors.value[tabId]
        if (editor) {
          // 更新参数列表
          updateParametersFromSql(tabId)
          // 编辑器已初始化，执行 SQL
          executeSqlInTab(tabId)
          delete newTab.autoExecute
        } else {
          // 如果编辑器还未初始化，等待一下再检查
          setTimeout(checkEditor, 100)
        }
      }
      checkEditor()
    })
  } else {
    // 如果没有关系型连接，使用原来的方式
    currentTableListConnectionId.value = connection.id
    handleTableSelect(connection.id, table.name)
  }
}

const handleLeftQueryDblClick = async (connection, query) => {
  // 创建新的查询标签页
  queryTabCounter++
  const tabId = `query-${queryTabCounter}`
  
  // 获取查询的SQL语句
  let sql = ''
  if (query.sqlConfig && query.sqlConfig.sql) {
    sql = query.sqlConfig.sql
  } else {
    ElMessage.warning('该查询没有SQL语句')
    return
  }
  
  const newTab = {
    id: tabId,
    name: query.name,
    connectionId: query.connectionId || connection.id,
    table: '',
    tables: [],
    sql: sql,
    result: null,
    executing: false,
    saving: false,
    modified: false,
    resultPage: 1,
    resultPageSize: 20,
    parameters: [],
    parametersExpanded: true,
    autoExecute: true // 标记需要自动执行
  }

  queryTabs.value.push(newTab)
  activeQueryTab.value = tabId

  // 加载表列表
  try {
    const response = await dataAPI.getConnectionTables(project.value.id, newTab.connectionId)
    if (response.success) {
      newTab.tables = response.data.tables || []
    }
  } catch (error) {
    console.error('加载表列表失败:', error)
  }

  // 等待 Monaco Editor 初始化后自动执行 SQL
  nextTick(() => {
    const checkEditor = () => {
      const editor = monacoEditors.value[tabId]
      if (editor) {
        // 更新参数列表
        updateParametersFromSql(tabId)
        // 编辑器已初始化，执行 SQL
        executeSqlInTab(tabId)
        delete newTab.autoExecute
      } else {
        // 如果编辑器还未初始化，等待一下再检查
        setTimeout(checkEditor, 100)
      }
    }
    checkEditor()
  })
}

// 创建连接
const createConnection = async () => {
  if (!connectionFormRef.value) return

  try {
    await connectionFormRef.value.validate()
  } catch (error) {
    return
  }

  creatingConnection.value = true
  try {
    const config = {
      dbType: connectionForm.dbType,
      host: connectionForm.host,
      port: connectionForm.port,
      database: connectionForm.database,
      username: connectionForm.username,
      password: connectionForm.password
    }

    const response = await dataAPI.createConnection(project.value.id, {
      name: connectionForm.name,
      type: connectionForm.type,
      // category 由后端根据 type 自动推断
      config
    })

    if (response.success) {
      ElMessage.success('数据连接创建成功')
      showCreateConnectionDialog.value = false
      resetConnectionForm()
      loadConnections()
    }
  } catch (error) {
    ElMessage.error('创建连接失败：' + (error.response?.data?.message || error.message))
  } finally {
    creatingConnection.value = false
  }
}

const testConnection = async () => {
  if (!connectionFormRef.value) return

  try {
    await connectionFormRef.value.validate()
  } catch (error) {
    return
  }

  testingConnection.value = true
  try {
    const config = {
      dbType: connectionForm.dbType,
      host: connectionForm.host,
      port: connectionForm.port,
      database: connectionForm.database,
      username: connectionForm.username,
      password: connectionForm.password
    }

    const response = await dataAPI.testConnection(project.value.id, {
      type: connectionForm.type,
      config
    })

    if (response.success) {
      ElMessage.success(response.message || '数据连接可用')
    } else {
      ElMessage.error(response.message || '数据连接不可用')
    }
  } catch (error) {
    ElMessage.error('测试连接失败：' + (error.response?.data?.message || error.message))
  } finally {
    testingConnection.value = false
  }
}

// 重置连接表单
const resetConnectionForm = () => {
  Object.keys(connectionForm).forEach(key => {
    if (key === 'type') {
      connectionForm[key] = 'relational'
    } else if (key === 'dbType') {
      connectionForm[key] = 'mysql'
    } else if (key === 'port') {
      connectionForm[key] = 3306
    } else {
      connectionForm[key] = ''
    }
  })
  if (connectionFormRef.value) {
    connectionFormRef.value?.clearValidate()
  }
}

// 打开创建连接对话框
const openCreateConnectionDialog = () => {
  connectionDialogMode.value = 'create'
  currentConnection.value = null
  resetConnectionForm()
  showConnectionDialog.value = true
}

// 保存连接（创建或更新）
const saveConnection = async () => {
  if (connectionDialogMode.value === 'create') {
    await createConnection()
  } else {
    await updateConnection()
  }
}

// 更新连接
const updateConnection = async () => {
  if (!connectionFormRef.value || !currentConnection.value) return

  try {
    await connectionFormRef.value.validate()
  } catch (error) {
    return
  }

  creatingConnection.value = true
  try {
    const config = {
      dbType: connectionForm.dbType,
      host: connectionForm.host,
      port: connectionForm.port,
      database: connectionForm.database,
      username: connectionForm.username,
      password: connectionForm.password
    }

    const response = await dataAPI.updateConnection(project.value.id, currentConnection.value.id, {
      name: connectionForm.name,
      type: connectionForm.type,
      config
    })

    if (response.success) {
      ElMessage.success('数据连接更新成功')
      showConnectionDialog.value = false
      resetConnectionForm()
      loadConnections()
    }
  } catch (error) {
    ElMessage.error('更新连接失败：' + (error.response?.data?.message || error.message))
  } finally {
    creatingConnection.value = false
  }
}

// 处理右键菜单
const handleConnectionContextMenu = (event, connection) => {
  event.preventDefault()
  contextMenuConnection.value = connection
  contextMenuPosition.x = event.clientX
  contextMenuPosition.y = event.clientY
  showContextMenu.value = true
  
  // 点击其他地方关闭菜单
  const closeMenu = () => {
    showContextMenu.value = false
    document.removeEventListener('click', closeMenu)
  }
  setTimeout(() => {
    document.addEventListener('click', closeMenu)
  }, 100)
}

// 查看连接详情
const viewConnectionDetails = () => {
  if (!contextMenuConnection.value) return
  currentConnection.value = contextMenuConnection.value
  showViewConnectionDialog.value = true
  showContextMenu.value = false
}

// 编辑连接
const editConnection = () => {
  if (!contextMenuConnection.value) return
  
  connectionDialogMode.value = 'edit'
  currentConnection.value = contextMenuConnection.value
  
  // 填充表单数据
  connectionForm.name = currentConnection.value.name
  connectionForm.type = currentConnection.value.type
  
  if (currentConnection.value.type === 'relational' && currentConnection.value.relationalConfig) {
    const config = currentConnection.value.relationalConfig
    connectionForm.dbType = config.dbType
    connectionForm.host = config.host
    connectionForm.port = config.port
    connectionForm.database = config.database
    connectionForm.username = config.username
    connectionForm.password = config.password || ''
  }
  
  showConnectionDialog.value = true
  showContextMenu.value = false
}

// 打开连接（与双击效果一致）
const openConnection = async () => {
  if (!contextMenuConnection.value) return
  showContextMenu.value = false
  await handleConnectionDblClick(contextMenuConnection.value)
}

// 断开连接
const disconnectConnection = async () => {
  if (!contextMenuConnection.value) return
  
  const connectionId = contextMenuConnection.value.id
  
  try {
    const response = await dataAPI.updateConnectionStatus(
      project.value.id, 
      connectionId, 
      'disconnected'
    )
    
    if (response.success) {
      ElMessage.success('连接已断开')
      
      // 折叠左侧树
      const state = getConnectionState(connectionId)
      state.expanded = false
      state.tablesExpanded = false
      state.queriesExpanded = false
      
      // 如果当前显示的是该连接的表列表，清空右侧内容
      if (currentTableListConnectionId.value === connectionId) {
        currentTableListConnectionId.value = null
        resetTableView()
      }
      
      // 刷新连接列表以显示最新状态
      await loadConnections()
    }
  } catch (error) {
    ElMessage.error('断开连接失败：' + (error.response?.data?.message || error.message))
  } finally {
    showContextMenu.value = false
  }
}

// 删除连接
const deleteConnection = async () => {
  if (!contextMenuConnection.value) return
  
  try {
    await ElMessageBox.confirm(
      `确定要删除连接 "${contextMenuConnection.value.name}" 吗？此操作不可恢复。`,
      '删除确认',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    const response = await dataAPI.deleteConnection(project.value.id, contextMenuConnection.value.id)
    
    if (response.success) {
      ElMessage.success('连接删除成功')
      loadConnections()
      
      // 如果删除的是当前选中的连接，清空选择
      if (selectedConnectionId.value === contextMenuConnection.value.id) {
        selectedConnectionId.value = null
      }
    }
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除连接失败：' + (error.response?.data?.message || error.message))
    }
  } finally {
    showContextMenu.value = false
  }
}

// 格式化日期
const formatDate = (dateString) => {
  if (!dateString) return '-'
  const date = new Date(dateString)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

// 关系型连接列表
const relationalConnections = computed(() => {
  return connections.value.filter(c => c.type === 'relational')
})

// 创建新查询标签页
const createNewQuery = () => {
  if (relationalConnections.value.length === 0) {
    ElMessage.warning('请先创建关系型数据连接')
    return
  }

  queryTabCounter++
  const tabId = `query-${queryTabCounter}`
  const newTab = {
    id: tabId,
    name: `查询 ${queryTabCounter}`,
    connectionId: '',
    table: '',
    tables: [],
    sql: 'SELECT * FROM ',
    result: null,
    executing: false,
    saving: false,
    modified: false,
    resultPage: 1,
    resultPageSize: 20,
    parameters: [],
    parametersExpanded: true
  }

  queryTabs.value.push(newTab)
  activeQueryTab.value = tabId
  // 编辑器会在 ref 设置时自动初始化
}

// 设置 Monaco Editor ref
const setMonacoEditorRef = (el, tabId) => {
  if (el) {
    monacoEditors.value[tabId] = el
    // 编辑器初始化后，更新参数列表
    nextTick(() => {
      updateParametersFromSql(tabId)
    })
  } else {
    delete monacoEditors.value[tabId]
  }
}

// 从SQL中检测参数占位符并更新参数列表
const updateParametersFromSql = (tabId) => {
  const tab = queryTabs.value.find(t => t.id === tabId)
  if (!tab) return
  
  const sql = tab.sql || ''
  // 检测 SQL 中的 ? 占位符
  const matches = sql.match(/\?/g)
  const paramCount = matches ? matches.length : 0
  
  // 如果参数数量变化，更新参数数组
  if (tab.parameters.length !== paramCount) {
    const newParameters = []
    for (let i = 0; i < paramCount; i++) {
      // 如果已有参数值，保留；否则创建新的参数对象
      if (tab.parameters[i]) {
        newParameters.push({
          ...tab.parameters[i],
          index: i
        })
      } else {
        newParameters.push({
          index: i,
          value: '',
          type: 'string' // 默认类型
        })
      }
    }
    tab.parameters = newParameters
  }
}

// 清空参数值
const clearParameter = (tabId, index) => {
  const tab = queryTabs.value.find(t => t.id === tabId)
  if (tab && tab.parameters[index]) {
    tab.parameters[index].value = ''
    tab.modified = true
  }
}

// 格式化指定标签页的 SQL
const formatSqlInTab = (tabId) => {
  const editor = monacoEditors.value[tabId]
  if (!editor) {
    ElMessage.warning('SQL编辑器未初始化')
    return
  }

  try {
    const currentSql = editor.getValue()
    if (!currentSql.trim()) {
      ElMessage.warning('SQL内容为空')
      return
    }

    const formatted = format(currentSql, {
      language: 'mysql',
      tabWidth: 2,
      keywordCase: 'upper'
    })

    editor.setValue(formatted)
    ElMessage.success('SQL格式化成功')
  } catch (error) {
    ElMessage.error('SQL格式化失败：' + error.message)
  }
}

// 加载指定标签页的表列表
const loadTablesForTab = async (tabId) => {
  const tab = queryTabs.value.find(t => t.id === tabId)
  if (!tab || !tab.connectionId) return

  try {
    const response = await dataAPI.getConnectionTables(project.value.id, tab.connectionId)
    if (response.success) {
      tab.tables = response.data.tables || []
    }
  } catch (error) {
    ElMessage.error('加载表列表失败：' + error.message)
  }
}

// 标签页连接变化
const onQueryTabConnectionChange = (tabId, connectionId) => {
  const tab = queryTabs.value.find(t => t.id === tabId)
  if (!tab) return

  tab.connectionId = connectionId
  tab.table = ''
  tab.tables = []
  loadTablesForTab(tabId)
}

// 标签页表变化
const onQueryTabTableChange = (tabId, tableName) => {
  const editor = monacoEditors.value[tabId]
  const tab = queryTabs.value.find(t => t.id === tabId)
  if (!editor || !tab || !tableName) return

  // Monaco Editor 的插入文本功能可以通过组件方法实现
  // 这里简化处理，直接追加到 SQL 末尾
  const currentSql = editor.getValue()
  const newSql = currentSql + (currentSql.trim() ? ' ' : '') + tableName
  editor.setValue(newSql)
  editor.focus()
}

// 执行指定标签页的 SQL 查询
const executeSqlInTab = async (tabId) => {
  const editor = monacoEditors.value[tabId]
  const tab = queryTabs.value.find(t => t.id === tabId)
  if (!editor || !tab) {
    ElMessage.warning('SQL编辑器未初始化')
    return
  }

  const sqlText = editor.getValue().trim()
  if (!sqlText) {
    ElMessage.warning('SQL语句不能为空')
    return
  }

  if (!tab.connectionId) {
    ElMessage.warning('请先选择数据连接')
    return
  }

  // 更新参数列表
  updateParametersFromSql(tabId)

  // 检查必填参数（如果有参数但值为空，给出提示但不阻止执行）
  const emptyParams = []
  tab.parameters.forEach((param, index) => {
    if (!param.value || param.value.trim() === '') {
      emptyParams.push(index + 1)
    }
  })

  if (emptyParams.length > 0 && tab.parameters.length > 0) {
    const shouldContinue = await ElMessageBox.confirm(
      `参数 ${emptyParams.join(', ')} 未填写，将使用空值执行。是否继续？`,
      '参数提示',
      {
        confirmButtonText: '继续执行',
        cancelButtonText: '取消',
        type: 'warning'
      }
    ).catch(() => false)
    
    if (!shouldContinue) {
      tab.parametersExpanded = true
      return
    }
  }

  tab.executing = true
  tab.result = null

  try {
    // 构建参数数组（按顺序）
    const parameters = tab.parameters.map(param => {
      const value = param.value ? param.value.trim() : ''
      // 根据类型转换值
      if (param.type === 'number') {
        return value === '' ? null : Number(value)
      } else if (param.type === 'boolean') {
        return value === 'true' || value === '1'
      } else {
        return value
      }
    })

    const response = await dataAPI.executeSql(
      project.value.id,
      tab.connectionId,
      sqlText,
      parameters
    )

    if (response.success) {
      tab.result = {
        columns: response.data.columns || [],
        rows: response.data.rows || [],
        rowCount: response.data.rowCount || 0,
        executionTime: response.executionTime || 0
      }
      // 重置分页到第一页
      tab.resultPage = 1
      if (!tab.resultPageSize) {
        tab.resultPageSize = 20
      }
      ElMessage.success(`查询执行成功，返回 ${tab.result.rowCount} 行`)
    } else {
      ElMessage.error('查询执行失败：' + (response.error || response.message))
    }
  } catch (error) {
    ElMessage.error('执行SQL失败：' + (error.response?.data?.message || error.message))
  } finally {
    tab.executing = false
  }
}

// 保存指定标签页的查询
const saveQueryTab = async (tabId) => {
  const editor = monacoEditors.value[tabId]
  const tab = queryTabs.value.find(t => t.id === tabId)
  if (!editor || !tab) {
    ElMessage.warning('SQL编辑器未初始化')
    return
  }

  const sqlText = editor.getValue().trim()
  if (!sqlText) {
    ElMessage.warning('SQL语句不能为空')
    return
  }

  if (!tab.connectionId) {
    ElMessage.warning('请先选择数据连接')
    return
  }

  // 更新参数列表
  updateParametersFromSql(tabId)

  try {
    const { value: queryName } = await ElMessageBox.prompt('请输入查询名称', '保存查询', {
      confirmButtonText: '保存',
      cancelButtonText: '取消',
      inputPattern: /^.{2,100}$/,
      inputErrorMessage: '查询名称长度在 2 到 100 个字符',
      inputValue: tab.name.startsWith('查询') ? '' : tab.name
    })

    // 如果有参数，构建参数定义数组
    let parameters = []
    if (tab.parameters.length > 0) {
      parameters = tab.parameters.map((param, index) => ({
        name: `param${index + 1}`,
        type: param.type || 'string',
        required: false,
        default: param.value || null
      }))
    }

    tab.saving = true

    const response = await dataAPI.createQuery(project.value.id, {
      name: queryName,
      connectionId: tab.connectionId,
      queryType: 'sql',
      config: {
        sql: sqlText,
        parameters: parameters
      }
    })

    if (response.success) {
      ElMessage.success('查询保存成功')
      tab.name = queryName
      tab.modified = false
      await loadQueries()
    }
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('保存查询失败：' + (error.response?.data?.message || error.message))
    }
  } finally {
    tab.saving = false
  }
}

// 获取分页后的结果行数据
const getPaginatedResultRows = (tab) => {
  if (!tab.result || !tab.result.rows) {
    return []
  }
  const page = tab.resultPage || 1
  const pageSize = tab.resultPageSize || 20
  const start = (page - 1) * pageSize
  const end = start + pageSize
  return tab.result.rows.slice(start, end)
}

// 获取结果分页页码
const getResultPage = (tab) => {
  return tab.resultPage || 1
}

// 获取结果分页大小
const getResultPageSize = (tab) => {
  return tab.resultPageSize || 20
}

// 清除查询结果
const clearQueryResult = (tabId) => {
  const tab = queryTabs.value.find(t => t.id === tabId)
  if (tab) {
    tab.result = null
    tab.resultPage = 1
    tab.resultPageSize = 20
  }
}

// 移除查询标签页
const removeQueryTab = (tabId) => {
  const index = queryTabs.value.findIndex(t => t.id === tabId)
  if (index > -1) {
    // 销毁编辑器
    const editor = monacoEditors.value[tabId]
    if (editor) {
      editor.dispose()
      delete monacoEditors.value[tabId]
    }

    queryTabs.value.splice(index, 1)

    // 如果关闭的是当前活动标签页，切换到其他标签页
    if (activeQueryTab.value === tabId) {
      if (queryTabs.value.length > 0) {
        activeQueryTab.value = queryTabs.value[queryTabs.value.length - 1].id
      } else {
        activeQueryTab.value = ''
      }
    }
  }
}

// 标签页切换
const onQueryTabChange = (tabId) => {
  activeQueryTab.value = tabId
  // Monaco Editor 组件会自动处理初始化，无需手动处理
}

// 监听数据库类型变化，自动更新默认端口
watch(() => connectionForm.dbType, (newDbType) => {
  // 只在创建模式下自动更新端口，编辑模式保留原端口
  if (connectionDialogMode.value === 'create') {
    const defaultPorts = {
      'mysql': 3306,
      'postgresql': 5432,
      'sqlserver': 1433
    }
    connectionForm.port = defaultPorts[newDbType] || 3306
  }
})

// 组件挂载时加载数据
onMounted(() => {
  if (project.value?.id) {
    loadConnections()
    loadQueries()
  }
})

// 组件卸载时清理所有编辑器
onBeforeUnmount(() => {
  Object.values(monacoEditors.value).forEach(editor => {
    if (editor && editor.dispose) {
      editor.dispose()
    }
  })
  monacoEditors.value = {}
})

</script>

<style scoped>
.data-center {
  min-height: 400px;
}

/* 连接项样式 */
.connection-item {
  border-radius: 8px;
  transition: all 0.2s ease;
}

.connection-item:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

/* SQL编辑器样式 */
.sql-editor :deep(.el-textarea__inner) {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 14px;
  line-height: 1.5;
  background-color: #f8f9fa;
  border-radius: 6px;
}

.dark .sql-editor :deep(.el-textarea__inner) {
  background-color: #1f2937;
  color: #e5e7eb;
  border-color: #374151;
}

/* 查询标签页样式 */
.query-tabs {
  display: flex;
  flex-direction: column;
  height: 100%;
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

.query-tabs :deep(.el-tabs__nav-wrap::after) {
  display: none;
}

.query-tabs :deep(.el-tabs__item) {
  border-radius: 6px 6px 0 0;
}

/* 结果表格样式 */
.result-table :deep(.el-table__header th) {
  background-color: #f9fafb;
  color: #374151;
  font-weight: 600;
}

.dark .result-table :deep(.el-table__header th) {
  background-color: #1f2937;
  color: #e5e7eb;
}

/* 功能卡片样式 */
.function-card {
  transition: all 0.3s ease;
}

.function-card:hover {
  transform: scale(1.02);
}

/* 查询标签页内容样式 */
.query-editor-tab-content {
  min-height: 400px;
}

.sql-editor-wrapper {
  min-height: 300px;
  height: 100%;
  width: 100%;
}

.sql-editor-wrapper :deep(.cm-editor) {
  height: 100%;
  width: 100%;
}

.sql-editor-wrapper :deep(.cm-scroller) {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', 'Consolas', monospace;
  font-size: 14px;
  line-height: 1.6;
  height: 100%;
  overflow: auto;
}

.sql-editor-wrapper :deep(.cm-content) {
  min-height: 200px;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .data-center {
    flex-direction: column;
  }

  .connection-list {
    max-height: 200px;
  }

}
</style>

