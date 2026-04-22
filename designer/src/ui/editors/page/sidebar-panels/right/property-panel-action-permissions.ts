import type { Action, RolePermission } from "@/editor-core/document/types";
import { sanitizeRoleGrant } from "@/ui/shared/permissions/role-grant-summary";

/**
 * 以纯函数方式更新节点动作权限，便于测试“设置”和“清空”两条写回链路。
 * @param {Record<string, Action[]> | undefined} events - 节点事件映射
 * @param {string} eventName - 目标事件名
 * @param {number} actionIndex - 目标动作索引
 * @param {RolePermission | undefined} permission - 新权限；空值表示删除权限字段
 * @returns {Record<string, Action[]>} 更新后的事件映射
 */
export function updateNodeActionPermission(
  events: Record<string, Action[]> | undefined,
  eventName: string,
  actionIndex: number,
  permission: RolePermission | undefined,
): Record<string, Action[]> {
  const normalizedPermission = sanitizeRoleGrant(permission);
  return Object.fromEntries(
    Object.entries(events || {}).map(([name, actions]) => [
      name,
      (Array.isArray(actions) ? actions : []).map((action, index) => {
        if (name !== eventName || index !== actionIndex) {
          return action;
        }

        const nextAction: Action = { ...action };
        if (normalizedPermission) {
          nextAction.permissions = normalizedPermission;
        } else {
          delete nextAction.permissions;
        }
        return nextAction;
      }),
    ]),
  );
}
