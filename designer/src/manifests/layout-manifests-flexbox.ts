/** 布局 manifest：FlexContainer（flex 布局容器）。由 index 在 H/V 包 manifest 之前加载。 */
import { registerManifest } from "./manifest-registry";

// FlexContainer 布局容器
registerManifest({
  type: "FlexContainer",
  name: "Flex布局",
  category: "布局",
  isContainer: true,
  props: [
    {
      name: "direction",
      type: "enum",
      label: "方向",
      group: "布局",
      defaultValue: "row",
      options: [
        { label: "水平", value: "row" },
        { label: "垂直", value: "column" },
        { label: "水平反向", value: "row-reverse" },
        { label: "垂直反向", value: "column-reverse" },
      ],
    },
    {
      name: "wrap",
      type: "enum",
      label: "换行",
      group: "布局",
      defaultValue: "nowrap",
      options: [
        { label: "不换行", value: "nowrap" },
        { label: "换行", value: "wrap" },
        { label: "反向换行", value: "wrap-reverse" },
      ],
    },
    {
      name: "justify",
      type: "enum",
      label: "主轴对齐",
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
      label: "交叉轴对齐",
      group: "布局",
      defaultValue: "stretch",
      options: [
        { label: "拉伸", value: "stretch" },
        { label: "起始", value: "flex-start" },
        { label: "居中", value: "center" },
        { label: "末尾", value: "flex-end" },
        { label: "基线", value: "baseline" },
      ],
    },
    {
      name: "gap",
      type: "number",
      label: "间距",
      group: "布局",
      defaultValue: 0,
      min: 0,
      max: 100,
    },
  ],
});
