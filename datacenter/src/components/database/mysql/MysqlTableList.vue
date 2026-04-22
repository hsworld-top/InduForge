<template>
  <div class="mysql-table-list">
    <div
      v-if="loading"
      class="h-full flex items-center justify-center text-gray-500"
    >
      <IconTablerLoader class="mr-2 w-5 h-5 animate-spin" />
      表列表加载中...
    </div>
    <el-table
      v-else-if="tables.length > 0"
      :data="tables"
      size="small"
      border
      class="table-list"
      @row-dblclick="handleRowDblClick"
    >
      <el-table-column
        prop="name"
        label="表名"
        min-width="160"
        show-overflow-tooltip
      />
      <el-table-column
        prop="comment"
        label="备注"
        min-width="200"
        show-overflow-tooltip
      />
      <el-table-column prop="rows" label="记录数" width="120" align="right" />
    </el-table>
    <div v-else class="text-center py-12 text-gray-500">暂无表信息</div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, inject, watch } from "vue";
import IconTablerLoader from "~icons/tabler/loader";
import { useMysql } from "@/composables/database/useMysql";

const props = defineProps({
  connectionId: {
    type: String,
    required: true,
  },
});

const emit = defineEmits(["table-select"]);

const projectId = inject("projectId");
const { tables, loading, loadTables } = useMysql(
  projectId,
  ref(props.connectionId),
);

onMounted(() => {
  if (props.connectionId) {
    loadTables();
  }
});

watch(
  () => props.connectionId,
  (newId) => {
    if (newId) {
      loadTables();
    }
  },
);

const handleRowDblClick = (row) => {
  emit("table-select", row.name);
};
</script>
