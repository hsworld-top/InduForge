/** 布局 manifest：区域布局与表单布局。 */
import { registerManifest } from "./manifest-registry";

// FormLayout 表单布局
registerManifest({
  type: "FormLayout",
  name: "表单布局",
  category: "布局",
  isContainer: true,
  defaultSize: { width: 360, height: 280 },
  props: [
    {
      name: "itemGap",
      type: "number",
      label: "表单项间距",
      group: "布局",
      defaultValue: 12,
      min: 0,
      max: 100,
    },
  ],
});

// ElContainer 区域布局
registerManifest({
  type: "ElContainer",
  name: "区域布局",
  category: "布局",
  isContainer: true,
  props: [
    {
      name: "regionPreset",
      type: "enum",
      label: "区域预设",
      group: "布局",
      defaultValue: "aside-between",
      options: [
        { label: "上下", value: "top-main" },
        { label: "上左下（左单独一列）", value: "aside-full-height" },
        { label: "上左下（左被上下夹着）", value: "aside-between" },
      ],
    },
    {
      name: "showHeader",
      type: "boolean",
      label: "Header区域",
      group: "显示",
      defaultValue: true,
    },
    {
      name: "showAside",
      type: "boolean",
      label: "Aside区域",
      group: "显示",
      defaultValue: true,
    },
    {
      name: "showMain",
      type: "boolean",
      label: "Main区域",
      group: "显示",
      defaultValue: true,
    },
    {
      name: "showFooter",
      type: "boolean",
      label: "Footer区域",
      group: "显示",
      defaultValue: true,
    },
    {
      name: "headerHeight",
      type: "string",
      label: "Header高度",
      group: "布局",
      defaultValue: "60px",
    },
    {
      name: "asideWidth",
      type: "string",
      label: "Aside宽度",
      group: "布局",
      defaultValue: "200px",
    },
    {
      name: "footerHeight",
      type: "string",
      label: "Footer高度",
      group: "布局",
      defaultValue: "60px",
    },
  ],
});
