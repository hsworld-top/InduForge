import type { ComponentManifest } from "@/materials/manifests/manifest-registry";
import { registerManifest } from "@/materials/manifests/manifest-registry";

export const defaultUserAvatarMenuItems = [
  {
    id: "profile",
    label: "个人资料",
    icon: "User",
    type: "script",
    script: "",
  },
  {
    id: "locale",
    label: "语言切换",
    icon: "Switch",
    type: "builtin",
    builtinAction: "locale",
  },
  {
    id: "theme",
    label: "主题切换",
    icon: "Moon",
    type: "builtin",
    builtinAction: "theme",
  },
  {
    id: "logout",
    label: "退出登录",
    icon: "SwitchButton",
    type: "builtin",
    builtinAction: "logout",
    danger: true,
  },
];

export const manifest: ComponentManifest = {
  type: "UserAvatarMenu",
  name: "用户头像",
  category: "功能",
  defaultStyle: { width: 180, height: 44 },
  props: [
    {
      name: "showName",
      type: "boolean",
      label: "显示用户名",
      group: "内容",
      defaultValue: true,
    },
    {
      name: "showSubtitle",
      type: "boolean",
      label: "显示副标题",
      group: "内容",
      defaultValue: true,
    },
    {
      name: "avatarSize",
      type: "number",
      label: "头像尺寸",
      group: "外观",
      defaultValue: 32,
      min: 24,
      max: 56,
      step: 1,
    },
    {
      name: "variant",
      type: "enum",
      label: "样式",
      group: "外观",
      defaultValue: "standard",
      options: [
        { label: "标准", value: "standard" },
        { label: "紧凑", value: "compact" },
      ],
    },
    {
      name: "menuItems",
      type: "array",
      label: "菜单项",
      group: "菜单",
      defaultValue: defaultUserAvatarMenuItems,
      editor: "code",
      language: "json",
      height: "220px",
    },
  ],
};

registerManifest(manifest);

export default manifest;
