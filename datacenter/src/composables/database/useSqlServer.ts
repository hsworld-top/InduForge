// @ts-nocheck
import { ref } from "vue";
import { ElMessage } from "element-plus";
import dataAPI from "@/api/data.api";

/**
 * SQL Server 数据库操作 Composable
 */
export function useSqlServer(projectId, connectionId) {
  const tables = ref([]);
  const loading = ref(false);

  /**
   * 加载表列表
   */
  const loadTables = async () => {
    if (!connectionId.value) return;

    loading.value = true;
    try {
      const response = await dataAPI.getConnectionTables(
        projectId.value,
        connectionId.value,
      );
      if (response.success) {
        tables.value = response.data.tables || [];
      }
    } catch (error) {
      ElMessage({
        type: "error",
        message:
          "加载表列表失败：" + (error.response?.data?.message || error.message),
        offset: 60,
        duration: 5000,
        showClose: true,
      });
    } finally {
      loading.value = false;
    }
  };

  /**
   * 格式化 SQL
   */
  const formatSql = (sql) => {
    if (!sql) return "";

    // 简单的 SQL 格式化
    return sql
      .replace(/\s+/g, " ")
      .replace(/\s*,\s*/g, ", ")
      .replace(/\s*\(\s*/g, " (")
      .replace(/\s*\)\s*/g, ") ")
      .replace(/\bSELECT\b/gi, "SELECT")
      .replace(/\bFROM\b/gi, "\nFROM")
      .replace(/\bWHERE\b/gi, "\nWHERE")
      .replace(/\bAND\b/gi, "\n  AND")
      .replace(/\bOR\b/gi, "\n  OR")
      .replace(/\bORDER BY\b/gi, "\nORDER BY")
      .replace(/\bGROUP BY\b/gi, "\nGROUP BY")
      .replace(/\bHAVING\b/gi, "\nHAVING")
      .replace(/\bLIMIT\b/gi, "\nLIMIT")
      .replace(/\bOFFSET\b/gi, "\nOFFSET")
      .replace(/\bJOIN\b/gi, "\nJOIN")
      .replace(/\bLEFT JOIN\b/gi, "\nLEFT JOIN")
      .replace(/\bRIGHT JOIN\b/gi, "\nRIGHT JOIN")
      .replace(/\bINNER JOIN\b/gi, "\nINNER JOIN")
      .trim();
  };

  /**
   * 提取 SQL 参数
   * SQL Server 使用 @param1, @param2 格式
   */
  const extractSqlParameters = (sql) => {
    if (!sql) return [];

    const paramRegex = /@(\w+)/g;
    const matches = [...sql.matchAll(paramRegex)];
    const uniqueParams = [...new Set(matches.map((m) => m[1]))];

    return uniqueParams.map((name) => ({
      name: name,
      type: "string",
      value: "",
    }));
  };

  return {
    tables,
    loading,
    loadTables,
    formatSql,
    extractSqlParameters,
  };
}
