// @ts-nocheck
import request from "@/utils/request";
import { Storage } from "@/utils/storage";

const DEFAULT_DEBUG_PROJECT_NAME = "test";
const DEBUG_PROJECT_QUERY_LIMIT = 50;

const asNonEmptyString = (value) =>
  typeof value === "string" && value.trim().length > 0 ? value.trim() : null;

const normalizeProjectList = (payload) => {
  const projects = Array.isArray(payload?.data?.projects)
    ? payload.data.projects
    : [];

  return projects
    .map((item) => {
      if (!item || typeof item !== "object") {
        return null;
      }

      const id = asNonEmptyString(item.id);
      const name = asNonEmptyString(item.name);
      if (!id || !name) {
        return null;
      }

      return {
        id,
        name,
        tenantId: asNonEmptyString(item.tenantId),
      };
    })
    .filter(Boolean);
};

/**
 * `/datacenter/debug` 默认工程解析。
 * 仅在开发态独立调试链路使用，按工程名精确匹配名称为 `test` 的工程。
 */
export const debugProjectAPI = {
  async resolveDefaultProjectByName(projectName = DEFAULT_DEBUG_PROJECT_NAME) {
    if (!Storage.getToken()) {
      return null;
    }

    const response = await request.get("/projects", {
      params: {
        page: 1,
        limit: DEBUG_PROJECT_QUERY_LIMIT,
        name: projectName,
      },
    });
    return (
      normalizeProjectList(response).find(
        (item) => item.name === projectName,
      ) || null
    );
  },
};

export default debugProjectAPI;
