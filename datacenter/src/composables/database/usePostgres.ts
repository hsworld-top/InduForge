// @ts-nocheck
import { ref } from "vue";
import { ElMessage } from "element-plus";
import dataAPI from "@/api/data.api";

/**
 * PostgreSQL 数据库操作 Composable
 */
export function usePostgres(projectId, connectionId) {
  const tables = ref([]);
  const loading = ref(false);

  /**
   * 加载表列表
   */
  const loadTables = async () => {
    if (!projectId.value || !connectionId.value) {
      console.warn("projectId 或 connectionId 为空，无法加载表列表");
      return;
    }

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
      tables.value = [];
    } finally {
      loading.value = false;
    }
  };

  /**
   * 格式化 SQL（简单实现）
   */
  const formatSql = (sql) => {
    if (!sql) return "";

    // 简单的 SQL 格式化
    const formatted = sql
      .replace(/\s+/g, " ")
      .replace(/\s*,\s*/g, ",\n  ")
      .replace(
        /\s+(FROM|WHERE|GROUP BY|ORDER BY|LIMIT|OFFSET|HAVING|JOIN|LEFT JOIN|RIGHT JOIN|INNER JOIN)/gi,
        "\n$1",
      )
      .replace(/\s+(AND|OR)\s+/gi, "\n  $1 ")
      .trim();

    return formatted;
  };

  /**
   * 提取 SQL 参数（PostgreSQL 使用 $1, $2, ... 格式）
   */
  const extractSqlParameters = (sql) => {
    if (!sql) return [];

    // 匹配 $1, $2, $3 等参数占位符
    const paramRegex = /\$(\d+)/g;
    const matches = [...sql.matchAll(paramRegex)];

    if (matches.length === 0) return [];

    // 获取最大的参数编号
    const maxParamNum = Math.max(...matches.map((m) => parseInt(m[1])));

    // 创建参数数组
    const parameters = [];
    for (let i = 1; i <= maxParamNum; i++) {
      parameters.push({
        name: `param${i}`,
        type: "string",
        value: "",
      });
    }

    return parameters;
  };

  return {
    tables,
    loading,
    loadTables,
    formatSql,
    extractSqlParameters,
  };
}
