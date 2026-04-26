/**
 * 工程运行态身份主数据同步。
 *
 * 设计器只消费 dev_ide/dev_core 创建的运行态用户和角色，这里负责把接口数据规整成
 * 页面权限配置可以直接引用的稳定快照，避免继续使用默认占位角色污染保存结果。
 */

import type { Ref } from "vue";
import type { ProjectSchema, RuntimeRoleRef } from "@/editor-core/document/types";
import { unwrapApiData } from "@/types/api";

export interface RuntimeRoleRecord extends RuntimeRoleRef {
  id: string;
  code: string;
  name: string;
  description?: string | null;
  status: string;
}

export interface RuntimeUserRecord {
  id: string;
  username: string;
  displayName: string;
  status: string;
  roleIds: string[];
  roles: RuntimeRoleRecord[];
}

export interface RuntimeAccessApi {
  getRuntimeRoles: (projectId: string) => Promise<unknown>;
  getRuntimeUsers?: (projectId: string) => Promise<unknown>;
}

export interface RuntimeAccessStateRefs {
  activeProjectId: Ref<string>;
  runtimeRoleCodes: Ref<string[]>;
  runtimeRoles?: Ref<RuntimeRoleRecord[]>;
  runtimeUsers?: Ref<RuntimeUserRecord[]>;
  runtimeRoleLoadState: Ref<"idle" | "loading" | "ready" | "error">;
  runtimeRoleLoadError: Ref<string>;
  runtimeRoleRequestSerial: Ref<number>;
}

export interface RuntimeAccessLoadResult {
  accepted: boolean;
  runtimeRoleCodes: string[];
  runtimeRoles: RuntimeRoleRecord[];
  runtimeUsers: RuntimeUserRecord[];
}

const normalizeText = (value: unknown): string => String(value ?? "").trim();

function normalizeRoleCodes(codes: unknown[]): string[] {
  const deduped = new Set<string>();
  for (const code of codes) {
    const normalized = normalizeText(code);
    if (!normalized) continue;
    deduped.add(normalized);
  }
  return Array.from(deduped);
}

function normalizeStatus(value: unknown): string {
  const status = normalizeText(value);
  return status || "active";
}

function normalizeRuntimeRole(value: unknown): RuntimeRoleRecord | null {
  if (!value || typeof value !== "object") return null;
  const role = value as Record<string, unknown>;
  const id = normalizeText(role.id ?? role.roleId ?? role.code ?? role.roleCode);
  const code = normalizeText(role.code ?? role.roleCode);
  const name = normalizeText(role.name ?? role.roleName) || code || id;
  if (!id || !code) return null;
  return {
    id,
    code,
    name,
    roleId: id,
    roleCode: code,
    roleName: name,
    description: typeof role.description === "string" ? role.description : null,
    status: normalizeStatus(role.status),
  };
}

function uniqRoles(roles: RuntimeRoleRecord[]): RuntimeRoleRecord[] {
  const seen = new Set<string>();
  const result: RuntimeRoleRecord[] = [];
  for (const role of roles) {
    if (seen.has(role.id)) continue;
    seen.add(role.id);
    result.push(role);
  }
  return result;
}

export function toRuntimeRoleRef(role: RuntimeRoleRecord): RuntimeRoleRef {
  return {
    roleId: role.id,
    roleCode: role.code,
    roleName: role.name,
  };
}

export function normalizeProjectRuntimeRoles(payload: unknown): RuntimeRoleRecord[] {
  if (!payload || typeof payload !== "object") return [];
  const runtimeRoles = (payload as { runtimeRoles?: unknown }).runtimeRoles;
  if (!Array.isArray(runtimeRoles)) return [];
  return uniqRoles(runtimeRoles.map(normalizeRuntimeRole).filter(Boolean) as RuntimeRoleRecord[]);
}

export function normalizeProjectRuntimeRoleCodes(payload: unknown): string[] {
  return normalizeRoleCodes(normalizeProjectRuntimeRoles(payload).map((role) => role.code));
}

export function normalizeProjectRuntimeUsers(payload: unknown): RuntimeUserRecord[] {
  if (!payload || typeof payload !== "object") return [];
  const runtimeUsers = (payload as { runtimeUsers?: unknown }).runtimeUsers;
  if (!Array.isArray(runtimeUsers)) return [];
  return runtimeUsers
    .map((item) => {
      if (!item || typeof item !== "object") return null;
      const user = item as Record<string, unknown>;
      const id = normalizeText(user.id);
      const username = normalizeText(user.username);
      if (!id || !username) return null;
      const roles = Array.isArray(user.roles)
        ? uniqRoles(user.roles.map(normalizeRuntimeRole).filter(Boolean) as RuntimeRoleRecord[])
        : [];
      const roleIds = normalizeRoleCodes([
        ...(Array.isArray(user.roleIds) ? user.roleIds : []),
        ...roles.map((role) => role.id),
      ]);
      return {
        id,
        username,
        displayName: normalizeText(user.displayName) || username,
        status: normalizeStatus(user.status),
        roleIds,
        roles,
      };
    })
    .filter(Boolean) as RuntimeUserRecord[];
}

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

export async function loadProjectRuntimeAccessForStore(
  projectId: string,
  api: RuntimeAccessApi,
  out: RuntimeAccessStateRefs,
): Promise<RuntimeAccessLoadResult> {
  const requestSerial = out.runtimeRoleRequestSerial.value + 1;
  out.runtimeRoleRequestSerial.value = requestSerial;
  out.runtimeRoleCodes.value = [];
  if (out.runtimeRoles) out.runtimeRoles.value = [];
  if (out.runtimeUsers) out.runtimeUsers.value = [];
  out.runtimeRoleLoadError.value = "";

  if (!projectId) {
    out.runtimeRoleLoadState.value = "idle";
    return { accepted: false, runtimeRoleCodes: [], runtimeRoles: [], runtimeUsers: [] };
  }

  out.runtimeRoleLoadState.value = "loading";

  try {
    const [roleResponse, userResponse] = await Promise.all([
      api.getRuntimeRoles(projectId),
      api.getRuntimeUsers ? api.getRuntimeUsers(projectId) : Promise.resolve({ runtimeUsers: [] }),
    ]);
    const rolePayload = unwrapApiData(roleResponse);
    const userPayload = unwrapApiData(userResponse);
    const runtimeRoles = normalizeProjectRuntimeRoles(rolePayload);
    const runtimeUsers = normalizeProjectRuntimeUsers(userPayload);
    const runtimeRoleCodes = normalizeRoleCodes(runtimeRoles.map((role) => role.code));
    const isLatestRequest = out.runtimeRoleRequestSerial.value === requestSerial;
    const isActiveProject = out.activeProjectId.value === projectId;
    if (!isLatestRequest || !isActiveProject) {
      return { accepted: false, runtimeRoleCodes, runtimeRoles, runtimeUsers };
    }
    out.runtimeRoleCodes.value = runtimeRoleCodes;
    if (out.runtimeRoles) out.runtimeRoles.value = runtimeRoles;
    if (out.runtimeUsers) out.runtimeUsers.value = runtimeUsers;
    out.runtimeRoleLoadState.value = "ready";
    return { accepted: true, runtimeRoleCodes, runtimeRoles, runtimeUsers };
  } catch (error) {
    const isLatestRequest = out.runtimeRoleRequestSerial.value === requestSerial;
    const isActiveProject = out.activeProjectId.value === projectId;
    if (!isLatestRequest || !isActiveProject) {
      return { accepted: false, runtimeRoleCodes: [], runtimeRoles: [], runtimeUsers: [] };
    }
    out.runtimeRoleCodes.value = [];
    if (out.runtimeRoles) out.runtimeRoles.value = [];
    if (out.runtimeUsers) out.runtimeUsers.value = [];
    out.runtimeRoleLoadState.value = "error";
    out.runtimeRoleLoadError.value =
      error instanceof Error ? error.message : "加载运行态身份失败";
    return { accepted: true, runtimeRoleCodes: [], runtimeRoles: [], runtimeUsers: [] };
  }
}

export async function loadProjectRuntimeRoleCodesForStore(
  projectId: string,
  api: RuntimeAccessApi,
  out: RuntimeAccessStateRefs,
): Promise<{ accepted: boolean; runtimeRoleCodes: string[] }> {
  const result = await loadProjectRuntimeAccessForStore(projectId, api, out);
  return {
    accepted: result.accepted,
    runtimeRoleCodes: result.runtimeRoleCodes,
  };
}
