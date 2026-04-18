/**
 * 页面属性面板分区定义
 */

export type PageInspectorSectionKey = "basic" | "visual" | "runtime";

export interface PageInspectorSection {
  key: PageInspectorSectionKey;
  title: string;
  fields: string[];
}

const PAGE_INSPECTOR_SECTIONS: PageInspectorSection[] = [
  {
    key: "basic",
    title: "基本",
    fields: ["name", "description", "pageType", "path"],
  },
  {
    key: "visual",
    title: "视觉",
    fields: ["backgroundKind", "backgroundValue", "width", "height"],
  },
  {
    key: "runtime",
    title: "运行",
    fields: ["autoFit", "lockAspectRatio", "enableMinSize", "windowStyle", "permissionDesc"],
  },
];

/**
 * 获取页面属性面板分区定义
 * @returns {PageInspectorSection[]} 固定分区列表
 */
export function getPageInspectorSections(): PageInspectorSection[] {
  return PAGE_INSPECTOR_SECTIONS.map((section) => ({
    ...section,
    fields: [...section.fields],
  }));
}

/**
 * 判断运行区约束项是否可编辑
 * @param {boolean} autoFit - 是否启用自适应
 * @returns {boolean} 是否允许编辑约束项
 */
export function canEditRuntimeConstraint(autoFit: boolean): boolean {
  return Boolean(autoFit);
}
