/**
 * VerticalLayout 垂直布局组件 Manifest
 */

/** @type {import('../../manifests/index.js').ComponentManifest} */
export const manifest = {
  type: "VerticalLayout",
  name: "垂直布局",
  category: "布局",
  isContainer: true,
  defaultSize: { width: 240, height: 240 },
  props: [
    {
      name: "justify",
      type: "enum",
      label: "垂直排列",
      group: "布局",
      defaultValue: "flex-start",
      options: [
        { label: "起始", value: "flex-start" },
        { label: "居中", value: "center" },
        { label: "末尾", value: "flex-end" },
        { label: "两端", value: "space-between" },
        { label: "环绕", value: "space-around" },
        { label: "均匀", value: "space-evenly" },
      ],
    },
    {
      name: "align",
      type: "enum",
      label: "水平对齐",
      group: "布局",
      defaultValue: "stretch",
      options: [
        { label: "拉伸", value: "stretch" },
        { label: "左侧", value: "flex-start" },
        { label: "居中", value: "center" },
        { label: "右侧", value: "flex-end" },
        { label: "基线", value: "baseline" },
      ],
    },
    {
      name: "gap",
      type: "number",
      label: "间距",
      group: "布局",
      defaultValue: 10,
      min: 0,
      max: 100,
    },
  ],
};

export default manifest;
