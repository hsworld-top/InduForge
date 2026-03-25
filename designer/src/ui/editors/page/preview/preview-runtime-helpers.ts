/**
 * previewRuntime 纯辅助函数：排队键、数据点缓存键、组件调用队列回放。
 */

import type { PreviewComponentRefInfo } from "./preview-runtime.types";

export interface PendingComponentCall {
  method: string;
  args: unknown[];
}

/**
 * 组件等待队列键
 * @param {unknown} pageId - 页面 ID
 * @param {unknown} name - 组件名
 * @returns {string}
 */
export function buildPendingKey(pageId: unknown, name: unknown): string {
  return `${pageId || "global"}::${name}`;
}

/**
 * 数据点缓存键
 * @param {unknown} projectId - 工程 ID
 * @param {unknown} path - 数据点 path
 * @returns {string}
 */
export function buildDatapointCacheKey(projectId: unknown, path: unknown): string {
  return `${projectId || ""}::${path || ""}`;
}

/**
 * 组件别名（去掉结尾数字）
 * @param {string} name - 组件名
 * @returns {string}
 */
export function getComponentAlias(name: string): string {
  return String(name).replace(/\d+$/, "");
}

/**
 * 入队待执行组件调用
 * @param {Map<string, PendingComponentCall[]>} pendingCalls - 队列映射
 * @param {string | null | undefined} pageId - 页面 ID
 * @param {string | null | undefined} name - 组件名
 * @param {string} method - 方法名
 * @param {unknown[]} args - 参数
 * @returns {void}
 */
export function queueComponentCall(
  pendingCalls: Map<string, PendingComponentCall[]>,
  pageId: string | null | undefined,
  name: string | null | undefined,
  method: string,
  args: unknown[],
): void {
  if (!name) return;
  const key = buildPendingKey(pageId, name);
  if (!pendingCalls.has(key)) {
    pendingCalls.set(key, []);
  }
  pendingCalls.get(key)?.push({ method, args });
}

/**
 * 回放待执行组件调用
 * @param {Map<string, PendingComponentCall[]>} pendingCalls - 队列映射
 * @param {unknown} pageId - 页面 ID
 * @param {unknown} name - 组件名
 * @param {PreviewComponentRefInfo} refInfo - 组件 ref
 * @returns {void}
 */
export function applyPendingCalls(
  pendingCalls: Map<string, PendingComponentCall[]>,
  pageId: unknown,
  name: unknown,
  refInfo: PreviewComponentRefInfo,
): void {
  if (!name || !refInfo) return;
  const key = buildPendingKey(pageId, name);
  const calls = pendingCalls.get(key);
  if (!calls || calls.length === 0) return;
  calls.forEach((call) => {
    const fn = refInfo[call.method];
    if (typeof fn === "function") {
      (fn as (...args: unknown[]) => void)(...(call.args || []));
    }
  });
  pendingCalls.delete(key);
}

/** 组件脚本转发：stub 方法名与 queueComponentCall 第三参一致 */
export const COMPONENT_SCRIPT_QUEUE_METHODS = [
  "SetText",
  "SetType",
  "SetEllipsis",
  "SetTooltip",
  "SetLoading",
  "SetDisabled",
  "Click",
  "SetSrc",
  "Preview",
  "Reload",
  "InsertItem",
  "DeleteItem",
  "ClearAll",
  "ClearSelection",
  "AppendRow",
  "ToggleRowSelection",
  "ToggleAllSelection",
  "ToggleRowExpansion",
  "SetCurrentRow",
  "ClearSort",
  "ClearFilter",
  "Dolayout",
  "Sort",
  "SetData",
  "SetRadioEnable",
  "SetRadioVisible",
  "SetCheckState",
  "SetCheckEnable",
  "SetCheckVisible",
  "CheckAll",
  "UpdateKeyChildren",
  "SetCheckedNodes",
  "SetCheckedKeys",
  "SetChecked",
  "SetCurrentKey",
  "SetCurrentNode",
  "Remove",
  "Append",
  "InsertBefore",
  "InsertAfter",
  "Open",
  "Close",
  "Toggle",
  "SetActive",
  "Collapse",
  "Next",
  "Prev",
  "AddTab",
  "RemoveTab",
  "MoveToRight",
  "MoveToLeft",
  "Increase",
  "Decrease",
  "SetItems",
  "AppendItem",
  "Play",
  "Pause",
  "SetActiveItem",
  "Load",
  "PostMessage",
  "Back",
  "Forward",
  "Reset",
  "SetTitle",
  "SetTotal",
  "SetActiveNames",
  "Filter",
  "ExpandAll",
  "CollapseAll",
  "SetExpandedKeys",
  "ScrollToTop",
  "ScrollToRow",
  "DoLayoutSafe",
  "SetSelectionByKeys",
  "SetPage",
  "SetPageSize",
] as const;
