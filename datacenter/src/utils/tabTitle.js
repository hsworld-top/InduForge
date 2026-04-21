/**
 * 统一解析数据中心标签标题。
 * 工程页切换语言后需要重算静态后缀，否则标签文字会停留在旧语言。
 *
 * @param {object|null|undefined} tab - 标签对象
 * @param {(key: string, params?: Record<string, unknown>) => string} translate - 翻译函数
 * @returns {string}
 */
export const resolveDatacenterTabLabel = (tab, translate) => {
  if (!tab) return "";

  const prefix =
    typeof tab.labelPrefix === "string" ? tab.labelPrefix.trim() : "";
  const labelKey = typeof tab.labelKey === "string" ? tab.labelKey : "";
  const labelParams =
    tab?.labelParams && typeof tab.labelParams === "object"
      ? tab.labelParams
      : undefined;
  const translated =
    labelKey && typeof translate === "function"
      ? translate(labelKey, labelParams)
      : tab.label || "";

  if (!translated) {
    return prefix;
  }

  return prefix ? `${prefix} - ${translated}` : translated;
};
