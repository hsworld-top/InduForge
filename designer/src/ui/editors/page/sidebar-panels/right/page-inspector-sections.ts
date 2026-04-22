/**
 * 页面属性面板分区定义
 */

import { i18n } from "@/i18n";

export type PageInspectorSectionKey = "basic" | "visual" | "runtime";

export interface PageInspectorSection {
  key: PageInspectorSectionKey;
  title: string;
  fields: string[];
}

function buildPageInspectorSections(): PageInspectorSection[] {
  return [
    {
      key: "basic",
      title: i18n.global.t("pageInspector.sections.basic"),
      fields: ["name", "description", "pageType", "path"],
    },
    {
      key: "visual",
      title: i18n.global.t("pageInspector.sections.visual"),
      fields: ["backgroundKind", "backgroundValue", "width", "height"],
    },
    {
      key: "runtime",
      title: i18n.global.t("pageInspector.sections.runtime"),
      fields: ["autoFit", "lockAspectRatio", "enableMinSize", "windowStyle", "pageViewPermission"],
    },
  ];
}

/**
 * 获取页面属性面板分区定义
 * @returns {PageInspectorSection[]} 固定分区列表
 */
export function getPageInspectorSections(): PageInspectorSection[] {
  return buildPageInspectorSections().map((section) => ({
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
