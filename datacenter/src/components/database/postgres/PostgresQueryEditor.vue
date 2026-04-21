<template>
  <div class="postgres-query-editor flex flex-col h-full">
    <!-- 工具栏 -->
    <div
      class="flex items-center justify-between mb-3 pb-3 border-b border-gray-200 dark:border-gray-700"
    >
      <div class="flex items-center space-x-3">
        <el-select
          v-model="currentConnectionId"
          :placeholder="t('queryEditor.selectDatabase')"
          size="small"
          class="w-56"
          filterable
          @change="handleConnectionChange"
        >
          <el-option
            v-for="conn in relationalConnections"
            :key="conn.id"
            :label="`${conn.name} (${conn.relationalConfig?.database || ''})`"
            :value="conn.id"
          >
            <div class="flex items-center justify-between w-full">
              <span class="truncate">{{ conn.name }}</span>
              <span class="text-xs text-gray-400 ml-2 flex-shrink-0">{{
                conn.relationalConfig?.database
              }}</span>
            </div>
          </el-option>
        </el-select>
        <el-select
          v-model="localTab.table"
          :placeholder="t('queryEditor.selectTable')"
          size="small"
          class="w-48"
          filterable
          @change="handleTableChange"
        >
          <el-option
            v-for="table in tables"
            :key="table.name"
            :label="table.name"
            :value="table.name"
          />
        </el-select>
      </div>
      <div class="flex items-center space-x-2">
        <el-button size="small" @click="handleFormat">
          <IconTablerFileCheck class="mr-1 w-4 h-4" />
          {{ t("actions.formatSql") }}
        </el-button>
        <el-button
          type="primary"
          size="small"
          @click="handleExecute"
          :loading="localTab.executing"
        >
          <IconTablerPlayerPlay class="mr-1 w-4 h-4" />
          {{ t("actions.run") }}
        </el-button>
        <el-button
          size="small"
          @click="handleSave"
          :loading="localTab.saving"
          >{{ t("actions.save") }}</el-button
        >
      </div>
    </div>

    <!-- 参数化查询提示 -->
    <div class="mb-2 flex items-center justify-between">
      <div
        class="flex items-center space-x-2 text-xs text-gray-500 dark:text-gray-400"
      >
        <IconTablerInfoCircle class="w-4 h-4" />
        <span>{{ t("queryEditor.parameterHintPostgres") }}</span>
      </div>
    </div>

    <!-- SQL编辑器 -->
    <div
      class="flex-shrink-0 border border-gray-200 dark:border-gray-700 rounded overflow-hidden mb-4"
      style="height: 300px"
    >
      <MonacoEditor
        v-model="localTab.sql"
        language="pgsql"
        :theme="isDark ? 'vs-dark' : 'vs'"
        height="300px"
        :options="{
          minimap: { enabled: true },
          fontSize: 14,
          wordWrap: 'on',
          formatOnPaste: true,
          formatOnType: true,
          suggestOnTriggerCharacters: true,
          quickSuggestions: {
            other: true,
            comments: false,
            strings: false,
          },
          acceptSuggestionOnCommitCharacter: true,
          acceptSuggestionOnEnter: 'on',
          tabCompletion: 'on',
        }"
        @change="handleSqlChange"
      />
    </div>

    <!-- 参数输入面板 -->
    <div
      v-if="localTab.parameters && localTab.parameters.length > 0"
      class="mb-4 border border-gray-200 dark:border-gray-700 rounded p-3 bg-gray-50 dark:bg-gray-800"
    >
      <div class="flex items-center justify-between mb-2">
        <div class="flex items-center space-x-2">
          <IconTablerList class="text-gray-500 w-4 h-4" />
          <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{{
            t("queryEditor.sqlParameters")
          }}</span>
          <el-tag size="small" type="info">{{
            t("queryEditor.parameterCount", {
              count: localTab.parameters.length,
            })
          }}</el-tag>
        </div>
        <el-button
          size="small"
          text
          @click="localTab.parametersExpanded = !localTab.parametersExpanded"
        >
          <component
            :is="
              localTab.parametersExpanded
                ? IconTablerChevronDown
                : IconTablerChevronRight
            "
            class="mr-1 w-4 h-4"
          />
          {{
            localTab.parametersExpanded
              ? t("actions.collapse")
              : t("actions.expand")
          }}
        </el-button>
      </div>

      <div v-show="localTab.parametersExpanded" class="space-y-2 mt-2">
        <div
          v-for="(param, index) in localTab.parameters"
          :key="index"
          class="flex items-center space-x-2"
        >
          <div
            class="w-24 text-sm text-gray-600 dark:text-gray-400 flex-shrink-0"
          >
            {{ t("queryEditor.parameterIndex", { index: index + 1 }) }}
          </div>
          <el-input
            v-model="param.value"
            :placeholder="
              t('queryEditor.parameterPlaceholder', { index: index + 1 })
            "
            size="small"
            class="flex-1"
          >
            <template #prepend>
              <span class="text-xs text-gray-500">${{ index + 1 }}</span>
            </template>
          </el-input>
        </div>
      </div>
    </div>

    <!-- 查询结果 -->
    <div
      v-if="localTab.result"
      class="flex-1 flex flex-col border-t border-gray-200 dark:border-gray-700 pt-4 min-h-0"
    >
      <div class="flex items-center justify-between mb-2 flex-shrink-0">
        <div class="text-sm text-gray-600 dark:text-gray-400">
          {{
            t("queryEditor.resultSummary", {
              rowCount: localTab.result.rowCount,
              executionTime: localTab.result.executionTime,
            })
          }}
        </div>
        <el-button size="small" @click="clearResult">{{
          t("actions.close")
        }}</el-button>
      </div>
      <div
        class="flex-1 flex flex-col border border-gray-200 dark:border-gray-700 rounded min-h-0"
      >
        <div class="flex-1 overflow-auto min-h-0">
          <el-table :data="paginatedRows" size="small" border stripe>
            <el-table-column
              v-for="(column, index) in localTab.result.columns"
              :key="index"
              :prop="index.toString()"
              :label="column"
              min-width="120"
              show-overflow-tooltip
            />
          </el-table>
        </div>
        <div
          class="flex items-center justify-between px-4 py-2 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 flex-shrink-0"
        >
          <div class="text-sm text-gray-600 dark:text-gray-400">
            {{
              t("queryEditor.paginationSummary", {
                start: (localTab.resultPage - 1) * localTab.resultPageSize + 1,
                end: Math.min(
                  localTab.resultPage * localTab.resultPageSize,
                  localTab.result.rowCount,
                ),
                total: localTab.result.rowCount,
              })
            }}
          </div>
          <el-pagination
            v-model:current-page="localTab.resultPage"
            v-model:page-size="localTab.resultPageSize"
            :page-sizes="[10, 20, 50, 100, 200]"
            :total="localTab.result.rowCount"
            layout="sizes, prev, pager, next"
            small
            @size-change="localTab.resultPage = 1"
          />
        </div>
      </div>
    </div>

    <DataPointInlineList
      v-if="localTab.queryId"
      :project-id="projectId"
      :query-id="localTab.queryId"
    />
  </div>
</template>

<script setup>
import { ref, computed, inject, onMounted, onUnmounted } from "vue";
import IconTablerFileCheck from "~icons/tabler/file-check";
import IconTablerPlayerPlay from "~icons/tabler/player-play";
import IconTablerList from "~icons/tabler/list";
import IconTablerChevronDown from "~icons/tabler/chevron-down";
import IconTablerChevronRight from "~icons/tabler/chevron-right";
import IconTablerInfoCircle from "~icons/tabler/info-circle";
import { t } from "@/i18n/runtime";
import MonacoEditor from "@/components/MonacoEditor.vue";
import { usePostgres } from "@/composables/database/usePostgres";
import { useConnection } from "@/composables/useConnection";
import {
  registerSqlCompletionProvider,
  unregisterSqlCompletionProvider,
} from "@/utils/sqlCompletion";
import * as monaco from "monaco-editor";
import DataPointInlineList from "@/components/datapoint/DataPointInlineList.vue";

const props = defineProps({
  tab: {
    type: Object,
    required: true,
  },
});

const emit = defineEmits([
  "execute",
  "save",
  "update:tab",
  "connection-change",
]);

const projectId = inject("projectId");
const isDark = computed(() =>
  document.documentElement.classList.contains("dark"),
);

const localTab = computed(() => props.tab);
const currentConnectionId = ref(props.tab.connectionId);

const { connections, loadConnections } = useConnection(projectId);

const relationalConnections = computed(() => {
  return connections.value.filter((conn) => conn.type === "relational");
});

const { tables, loadTables, formatSql, extractSqlParameters } = usePostgres(
  projectId,
  computed(() => currentConnectionId.value),
);

onMounted(async () => {
  await loadConnections();

  if (localTab.value.connectionId) {
    await loadTables();
  }

  try {
    registerSqlCompletionProvider(monaco);
    console.log("SQL 自动补全已注册");
  } catch (error) {
    console.error("注册 SQL 自动补全失败:", error);
  }
});

onUnmounted(() => {
  unregisterSqlCompletionProvider();
});

const paginatedRows = computed(() => {
  if (!localTab.value.result || !localTab.value.result.rows) return [];
  const start = (localTab.value.resultPage - 1) * localTab.value.resultPageSize;
  const end = start + localTab.value.resultPageSize;
  return localTab.value.result.rows.slice(start, end);
});

const handleConnectionChange = async (connectionId) => {
  props.tab.connectionId = connectionId;
  props.tab.modified = true;

  await loadTables();

  props.tab.table = "";

  emit("connection-change", connectionId);
};

const handleTableChange = (tableName) => {
  const sql = `SELECT * FROM "${tableName}" LIMIT 100`;
  props.tab.sql = sql;
  props.tab.modified = true;
  updateParameters();
};

const handleFormat = () => {
  if (!props.tab.sql.trim()) return;
  props.tab.sql = formatSql(props.tab.sql);
};

const handleExecute = () => {
  updateParameters();
  emit("execute", props.tab);
};

const handleSave = () => {
  emit("save", props.tab);
};

const handleSqlChange = (value) => {
  props.tab.sql = value;
  props.tab.modified = true;
  updateParameters();
};

const updateParameters = () => {
  const newParams = extractSqlParameters(props.tab.sql);
  const oldParams = props.tab.parameters || [];

  const mergedParams = newParams.map((newParam, index) => {
    const oldParam = oldParams[index];
    return {
      ...newParam,
      value: oldParam?.value || newParam.value || "",
    };
  });

  props.tab.parameters = mergedParams;
};

const clearResult = () => {
  props.tab.result = null;
  props.tab.resultPage = 1;
};
</script>
