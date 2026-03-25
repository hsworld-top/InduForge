/** localStorage 键前缀：各工程页面树排序 */
export const PAGE_TREE_ORDER_PREFIX = "designer.pageTreeOrder";

/** 根级容器在排序表中的键 */
export const ROOT_CONTAINER_KEY = "__root__";

export function getPageTreeOrderStorageKey(projectId: string): string {
  return `${PAGE_TREE_ORDER_PREFIX}:${projectId || "default"}`;
}

export function validatePageName(value: string | undefined | null): {
  valid: boolean;
  message: string;
} {
  const name = String(value || "").trim();
  if (!name) {
    return { valid: false, message: "名称不能为空" };
  }
  if (/^\.+$/.test(name)) {
    return { valid: false, message: "页面名称不能仅包含点号" };
  }
  if (/[/?#\\%]/.test(name)) {
    return { valid: false, message: "页面名称不能包含 / ? # % \\" };
  }
  if (/[\u0000-\u001F\u007F]/.test(name)) {
    return { valid: false, message: "页面名称不能包含控制字符" };
  }
  return { valid: true, message: "" };
}

export function moveIdBefore(ids: string[], sourceId: string, targetId: string): string[] {
  const nextIds = ids.filter((id) => id !== sourceId);
  const targetIndex = nextIds.indexOf(targetId);
  if (targetIndex === -1) {
    nextIds.push(sourceId);
    return nextIds;
  }
  nextIds.splice(targetIndex, 0, sourceId);
  return nextIds;
}
