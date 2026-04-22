/**
 * 工程运行态角色主数据同步：
 * - 统一从后端运行态角色接口拉取真实角色编码
 * - 在 store 层把角色写回当前 schema，避免权限编辑继续读默认占位角色
 * - 接口失败或返回空列表时明确回落为 []，不能默默保留 admin/operator/viewer
 */

import type { Ref } from "vue";
import type { ProjectSchema } from "@/editor-core/document/types";
import { unwrapApiData } from "@/types/api";

export interface RuntimeRoleApi {
  getRuntimeRoles: (projectId: string) => Promise<unknown>;
}

export interface RuntimeRoleStateRefs {
  activeProjectId: Ref<string>;
  runtimeRoleCodes: Ref<string[]>;
  runtimeRoleLoadState: Ref<"idle" | "loading" | "ready" | "error">;
  runtimeRoleLoadError: Ref<string>;
  runtimeRoleRequestSerial: Ref<number>;
}

export interface RuntimeRoleLoadResult {
  accepted: boolean;
  runtimeRoleCodes: string[];
}

function normalizeRoleCodes(codes: unknown[]): string[] {
  const deduped = new Set<string>();
  for (const code of codes) {
    const normalized = String(code || "").trim();
    if (!normalized) continue;
    deduped.add(normalized);
  }
  return Array.from(deduped);
}

/**
 * 从接口业务载荷中提取运行态角色编码。
 * 这里只认 `runtimeRoles[].code`，不做任何默认角色补位。
 */
export function normalizeProjectRuntimeRoleCodes(payload: unknown): string[] {
  if (!payload || typeof payload !== "object") {
    return [];
  }

  const runtimeRoles = (payload as { runtimeRoles?: unknown }).runtimeRoles;
  if (!Array.isArray(runtimeRoles)) {
    return [];
  }

  return normalizeRoleCodes(
    runtimeRoles.map((role) => {
      if (!role || typeof role !== "object") return "";
      return (role as { code?: unknown }).code;
    }),
  );
}

/**
 * 用真实运行态角色覆盖 schema 中的安全声明。
 * 即使为空列表也必须覆盖，避免旧默认角色重新出现在编辑器里。
 */
export function applyRuntimeRoleCodesToSchema(
  schema: ProjectSchema,
  runtimeRoleCodes: string[],
): ProjectSchema {
  schema.securityDecl = {
    ...schema.securityDecl,
    roles: normalizeRoleCodes(runtimeRoleCodes),
  };
  return schema;
}

/**
 * 加载工程运行态角色。
 * 失败时不阻断工程打开，但会清空缓存并记录错误，确保不会继续使用旧角色源。
 */
export async function loadProjectRuntimeRoleCodesForStore(
  projectId: string,
  api: RuntimeRoleApi,
  out: RuntimeRoleStateRefs,
): Promise<RuntimeRoleLoadResult> {
  const requestSerial = out.runtimeRoleRequestSerial.value + 1;
  out.runtimeRoleRequestSerial.value = requestSerial;
  out.runtimeRoleCodes.value = [];
  out.runtimeRoleLoadError.value = "";

  if (!projectId) {
    out.runtimeRoleLoadState.value = "idle";
    return { accepted: false, runtimeRoleCodes: [] };
  }

  out.runtimeRoleLoadState.value = "loading";

  try {
    const response = await api.getRuntimeRoles(projectId);
    const payload = unwrapApiData(response);
    const runtimeRoleCodes = normalizeProjectRuntimeRoleCodes(payload);
    const isLatestRequest = out.runtimeRoleRequestSerial.value === requestSerial;
    const isActiveProject = out.activeProjectId.value === projectId;
    if (!isLatestRequest || !isActiveProject) {
      return { accepted: false, runtimeRoleCodes };
    }
    out.runtimeRoleCodes.value = runtimeRoleCodes;
    out.runtimeRoleLoadState.value = "ready";
    return { accepted: true, runtimeRoleCodes };
  } catch (error) {
    const isLatestRequest = out.runtimeRoleRequestSerial.value === requestSerial;
    const isActiveProject = out.activeProjectId.value === projectId;
    if (!isLatestRequest || !isActiveProject) {
      return { accepted: false, runtimeRoleCodes: [] };
    }
    out.runtimeRoleCodes.value = [];
    out.runtimeRoleLoadState.value = "error";
    out.runtimeRoleLoadError.value =
      error instanceof Error ? error.message : "加载运行态角色失败";
    return { accepted: true, runtimeRoleCodes: [] };
  }
}
