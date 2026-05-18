import type { ComponentManifest } from "@/materials/manifests/manifest-registry";
import { registerManifest } from "@/materials/manifests/manifest-registry";

export const manifest: ComponentManifest = {
  type: "LanguageSwitcher",
  name: "语言切换",
  category: "功能",
  defaultStyle: { width: 160, height: 34 },
  props: [
    {
      name: "mode",
      type: "enum",
      label: "模式",
      group: "内容",
      defaultValue: "select",
      options: [
        { label: "下拉框", value: "select" },
        { label: "按钮组", value: "button-group" },
      ],
    },
    {
      name: "display",
      type: "enum",
      label: "显示内容",
      group: "内容",
      defaultValue: "name",
      options: [
        { label: "语言名称", value: "name" },
        { label: "语言编码", value: "code" },
        { label: "名称和编码", value: "name-code" },
      ],
    },
    {
      name: "size",
      type: "enum",
      label: "尺寸",
      group: "外观",
      defaultValue: "default",
      options: [
        { label: "大", value: "large" },
        { label: "默认", value: "default" },
        { label: "小", value: "small" },
      ],
    },
    {
      name: "disabled",
      type: "boolean",
      label: "禁用",
      group: "状态",
      defaultValue: false,
    },
  ],
};

registerManifest(manifest);

export default manifest;
