import type { RolePermission } from "@/editor-core/document/types";

/**
 * 规范化角色列表，清理空值并去重，避免把编辑器输入噪音写入运行态。
 * @param {string[] | undefined} roles - 原始角色列表
 * @returns {string[]} 清洗后的角色列表
 */
export function normalizeRoleList(roles?: string[]): string[] {
  const deduped = new Set<string>();
  for (const role of roles || []) {
    const normalized = String(role || "").trim();
    if (!normalized) continue;
    deduped.add(normalized);
  }
  return Array.from(deduped);
}

/**
 * 规范化轻量角色授权结构，仅保留发布态需要的字段。
 * @param {RolePermission | null | undefined} grant - 原始授权配置
 * @returns {RolePermission | undefined} 清洗后的授权配置
 */
export function sanitizeRoleGrant(grant?: RolePermission | null): RolePermission | undefined {
  const allowRoles = normalizeRoleList(grant?.allowRoles);
  const denyRoles = normalizeRoleList(grant?.denyRoles);
  const inherit = grant?.inherit === false ? false : grant?.inherit === true ? true : undefined;

  if (allowRoles.length === 0 && denyRoles.length === 0 && inherit !== false) {
    return undefined;
  }

  const nextGrant: RolePermission = {};
  if (allowRoles.length > 0) {
    nextGrant.allowRoles = allowRoles;
  }
  if (denyRoles.length > 0) {
    nextGrant.denyRoles = denyRoles;
  }
  if (inherit !== undefined) {
    nextGrant.inherit = inherit;
  }
  return nextGrant;
}

/**
 * 生成轻量授权摘要文案，供页面和动作面板复用。
 * @param {RolePermission | null | undefined} grant - 原始授权配置
 * @returns {string} 摘要文案
 */
export function summarizeRoleGrant(grant?: RolePermission | null): string {
  const normalizedGrant = sanitizeRoleGrant(grant);
  if (!normalizedGrant) {
    return "继承上级";
  }

  const summaryParts: string[] = [];
  if (normalizedGrant.allowRoles?.length) {
    summaryParts.push(`允许 ${normalizedGrant.allowRoles.length}`);
  }
  if (normalizedGrant.denyRoles?.length) {
    summaryParts.push(`拒绝 ${normalizedGrant.denyRoles.length}`);
  }
  if (normalizedGrant.inherit === false) {
    summaryParts.push("不继承");
  }

  return summaryParts.length > 0 ? summaryParts.join(" / ") : "继承上级";
}
