/**
 * 运行态权限授权结构工具。
 * 这里保持前端编辑模型足够轻量，只处理 allowRoles / denyRoles / inherit，
 * 不承担角色解析、继承链计算或任何运行时鉴权逻辑。
 */

type RuntimeGrantPayload = {
  allowRoles?: string[];
  denyRoles?: string[];
  inherit?: boolean;
};

const normalizeRoleList = (value: string[] | unknown) => {
  if (!Array.isArray(value)) {
    return [];
  }

  const result = [];
  const seen = new Set();

  value.forEach((item) => {
    const role = String(item ?? "").trim();
    if (!role || seen.has(role)) {
      return;
    }
    seen.add(role);
    result.push(role);
  });

  return result;
};

/**
 * 归一化运行态授权 payload。
 * - 边界条件：非数组输入会退化为空数组，避免把表单脏值直接提交给接口。
 * - 冲突处理：同一角色若同时出现在 allow/deny 中，固定以 deny 优先，避免生成自相矛盾的 payload。
 * - 异常分支：inherit 只有显式传入 false 时才关闭，缺省一律按继承处理。
 * @param {object} input - 原始授权输入
 * @returns {{allowRoles: string[], denyRoles: string[], inherit: boolean}}
 */
export const normalizeRuntimeGrantPayload = (
  input: RuntimeGrantPayload = {},
) => {
  const denyRoles = normalizeRoleList(input?.denyRoles);
  const denyRoleSet = new Set(denyRoles);

  return {
    allowRoles: normalizeRoleList(input?.allowRoles).filter(
      (role) => !denyRoleSet.has(role),
    ),
    denyRoles,
    inherit: input?.inherit !== false,
  };
};

/**
 * 输出数据点运行态权限摘要。
 * 摘要只用于列表就近展示，固定返回计数和继承状态，不在这里展开角色明细。
 * @param {object} input - 原始授权输入
 * @returns {string}
 */
export const summarizeRuntimeGrant = (input: RuntimeGrantPayload = {}) => {
  const value = normalizeRuntimeGrantPayload(input);
  return `允许 ${value.allowRoles.length} / 拒绝 ${value.denyRoles.length} / ${value.inherit ? "继承" : "不继承"}`;
};

export default {
  normalizeRuntimeGrantPayload,
  summarizeRuntimeGrant,
};
