// @ts-nocheck
const asNonEmptyString = (value) =>
  typeof value === "string" && value.trim().length > 0 ? value.trim() : null;

/**
 * 解析数据中心 debug 路由的工程上下文。
 *
 * 优先级：
 * 1. URL 明确传入的 pid / id
 * 2. 默认调试工程（名称精确匹配 `test`）
 * 3. 当前本地缓存的工程上下文兜底
 */
export async function resolveDatacenterDebugProjectMeta({
  targetUrl,
  resolveDefaultDebugProject,
  getStoredProjectId,
  getStoredTenantId,
  setProjectId,
  setTenantId,
}) {
  const url = new URL(targetUrl);
  const projectIdFromUrl =
    asNonEmptyString(url.searchParams.get("pid")) ||
    asNonEmptyString(url.searchParams.get("id"));
  const tenantIdFromUrl = asNonEmptyString(url.searchParams.get("tenant"));

  if (projectIdFromUrl) {
    setProjectId(projectIdFromUrl);
    if (tenantIdFromUrl) {
      setTenantId(tenantIdFromUrl);
    }

    return {
      id: projectIdFromUrl,
      tenantId: tenantIdFromUrl || getStoredTenantId(),
    };
  }

  const defaultDebugProject = await resolveDefaultDebugProject();
  if (defaultDebugProject?.id) {
    setProjectId(defaultDebugProject.id);
    if (defaultDebugProject.tenantId) {
      setTenantId(defaultDebugProject.tenantId);
    }

    return {
      id: defaultDebugProject.id,
      tenantId: defaultDebugProject.tenantId || getStoredTenantId(),
    };
  }

  return {
    id: getStoredProjectId(),
    tenantId: getStoredTenantId(),
  };
}
