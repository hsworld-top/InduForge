/**
 * 页面属性面板分区定义
 */

import { i18n } from "@/i18n";

export type PageInspectorSectionKey = "identity" | "route" | "viewport" | "visual" | "runtime";

export interface PageInspectorSection {
  key: PageInspectorSectionKey;
  title: string;
  fields: string[];
}

function buildPageInspectorSections(): PageInspectorSection[] {
  return [
    {
      key: "identity",
      title: i18n.global.t("pageInspector.sections.identity"),
      fields: ["name", "title", "description", "pageId"],
    },
    {
      key: "route",
      title: i18n.global.t("pageInspector.sections.route"),
      fields: ["role", "routeMode", "routePath", "routeSlug", "parentRoutePath"],
    },
    {
      key: "viewport",
      title: i18n.global.t("pageInspector.sections.viewport"),
      fields: [
        "viewportPreset",
        "width",
        "height",
        "autoFit",
        "lockAspectRatio",
        "minWidth",
        "minHeight",
        "overflowMode",
      ],
    },
    {
      key: "visual",
      title: i18n.global.t("pageInspector.sections.visual"),
      fields: [
        "backgroundType",
        "backgroundValue",
        "backgroundSize",
        "backgroundPosition",
        "backgroundRepeat",
        "transitionType",
      ],
    },
    {
      key: "runtime",
      title: i18n.global.t("pageInspector.sections.runtime"),
      fields: ["openMode", "popup", "permission", "cacheMode", "preloadMode"],
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
