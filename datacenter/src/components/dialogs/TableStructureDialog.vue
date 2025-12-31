<template>
  <el-dialog
    v-model="visible"
    :title="`表结构 - ${tableName}`"
    width="900px"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <div v-loading="loading" class="table-structure-content">
      <!-- 标签页 -->
      <el-tabs v-model="activeTab" class="structure-tabs">
        <!-- 字段信息 -->
        <el-tab-pane label="字段" name="columns">
          <el-table :data="structure.columns" border stripe max-height="500">
            <el-table-column prop="name" label="字段名" width="150" />
            <el-table-column prop="type" label="类型" width="120" />
            <el-table-column label="长度/精度" width="100">
              <template #default="{ row }">
                <span v-if="row.maxLength">{{ row.maxLength }}</span>
                <span v-else-if="row.numericPrecision">
                  {{ row.numericPrecision }}
                  <span v-if="row.numericScale">,{{ row.numericScale }}</span>
                </span>
                <span v-else>-</span>
              </template>
            </el-table-column>
            <el-table-column label="允许为空" width="90" align="center">
              <template #default="{ row }">
                <el-tag :type="row.nullable ? 'info' : 'success'" size="small">
                  {{ row.nullable ? "是" : "否" }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="defaultValue" label="默认值" width="100">
              <template #default="{ row }">
                <span v-if="row.defaultValue !== null">{{
                  row.defaultValue
                }}</span>
                <span v-else class="text-gray-400">NULL</span>
              </template>
            </el-table-column>
            <el-table-column label="标识" width="120">
              <template #default="{ row }">
                <div class="flex gap-1">
                  <el-tag v-if="row.isPrimary" type="danger" size="small"
                    >主键</el-tag
                  >
                  <el-tag v-if="row.isUnique" type="warning" size="small"
                    >唯一</el-tag
                  >
                  <el-tag v-if="row.autoIncrement" type="info" size="small"
                    >自增</el-tag
                  >
                </div>
              </template>
            </el-table-column>
            <el-table-column
              prop="comment"
              label="注释"
              min-width="150"
              show-overflow-tooltip
            />
          </el-table>
        </el-tab-pane>

        <!-- 索引信息 -->
        <el-tab-pane label="索引" name="indexes">
          <el-table :data="structure.indexes" border stripe max-height="500">
            <el-table-column prop="name" label="索引名" width="200" />
            <el-table-column label="类型" width="120">
              <template #default="{ row }">
                <el-tag
                  :type="
                    row.type === 'PRIMARY'
                      ? 'danger'
                      : row.type === 'UNIQUE'
                        ? 'warning'
                        : 'info'
                  "
                  size="small"
                >
                  {{ row.type }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="method" label="索引方法" width="120" />
            <el-table-column label="包含字段" min-width="300">
              <template #default="{ row }">
                <el-tag
                  v-for="(col, index) in row.columns"
                  :key="index"
                  size="small"
                  class="mr-1"
                >
                  {{ col }}
                </el-tag>
              </template>
            </el-table-column>
          </el-table>
          <div
            v-if="structure.indexes.length === 0"
            class="text-center text-gray-400 py-8"
          >
            暂无索引信息
          </div>
        </el-tab-pane>

        <!-- 外键信息 -->
        <el-tab-pane label="外键" name="foreignKeys">
          <el-table
            :data="structure.foreignKeys"
            border
            stripe
            max-height="500"
          >
            <el-table-column prop="name" label="外键名" width="200" />
            <el-table-column prop="columnName" label="本表字段" width="150" />
            <el-table-column
              prop="referencedTable"
              label="引用表"
              width="150"
            />
            <el-table-column
              prop="referencedColumn"
              label="引用字段"
              width="150"
            />
            <el-table-column prop="updateRule" label="ON UPDATE" width="120" />
            <el-table-column prop="deleteRule" label="ON DELETE" width="120" />
          </el-table>
          <div
            v-if="structure.foreignKeys.length === 0"
            class="text-center text-gray-400 py-8"
          >
            暂无外键信息
          </div>
        </el-tab-pane>
      </el-tabs>
    </div>

    <template #footer>
      <el-button @click="handleClose">关闭</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, watch, computed } from "vue";
import { ElMessage } from "element-plus";
import dataAPI from "@/api/data.api";

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false,
  },
  projectId: {
    type: String,
    required: true,
  },
  connectionId: {
    type: String,
    required: true,
  },
  tableName: {
    type: String,
    required: true,
  },
});

const emit = defineEmits(["update:modelValue"]);

const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit("update:modelValue", val),
});

const loading = ref(false);
const activeTab = ref("columns");
const structure = ref({
  columns: [],
  indexes: [],
  foreignKeys: [],
});

/**
 * 加载表结构
 */
const loadTableStructure = async () => {
  if (!props.projectId || !props.connectionId || !props.tableName) return;

  loading.value = true;
  try {
    const response = await dataAPI.getTableStructure(
      props.projectId,
      props.connectionId,
      props.tableName,
    );

    if (response.success) {
      structure.value = response.data;
    }
  } catch (error) {
    ElMessage({
      type: "error",
      message:
        "加载表结构失败：" + (error.response?.data?.message || error.message),
      offset: 60,
      duration: 5000,
      showClose: true,
    });
  } finally {
    loading.value = false;
  }
};

/**
 * 关闭对话框
 */
const handleClose = () => {
  visible.value = false;
  activeTab.value = "columns";
};

// 监听对话框打开
watch(
  () => props.modelValue,
  (newVal) => {
    if (newVal) {
      loadTableStructure();
    }
  },
);
</script>

<style scoped>
.table-structure-content {
  min-height: 400px;
}

.structure-tabs {
  margin-top: -10px;
}

:deep(.el-tabs__content) {
  padding: 0;
}

:deep(.el-table) {
  font-size: 13px;
}

:deep(.el-table th) {
  background-color: #f5f7fa;
  font-weight: 600;
}
</style>
