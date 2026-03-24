import { registerManifest } from "./manifest-registry";

// ============ Canvas 图形 Manifest ============

// Canvas.Rect 矩形
registerManifest({
  type: "Canvas.Rect",
  name: "矩形",
  category: "图形",
  props: [
    {
      name: "cornerRadius",
      type: "number",
      label: "圆角",
      group: "基础",
      defaultValue: 0,
      min: 0,
      max: 100,
    },
  ],
});

// Canvas.Circle 圆形
registerManifest({
  type: "Canvas.Circle",
  name: "圆形",
  category: "图形",
  props: [
    {
      name: "radius",
      type: "number",
      label: "半径",
      group: "基础",
      defaultValue: 50,
      min: 1,
      max: 1000,
    },
  ],
});

// Canvas.Line 线段
registerManifest({
  type: "Canvas.Line",
  name: "线段",
  category: "图形",
  props: [
    {
      name: "tension",
      type: "number",
      label: "张力",
      group: "基础",
      defaultValue: 0,
      min: 0,
      max: 1,
      step: 0.1,
    },
    {
      name: "closed",
      type: "boolean",
      label: "闭合",
      group: "基础",
      defaultValue: false,
    },
  ],
});

// Canvas.Pipe 管道
registerManifest({
  type: "Canvas.Pipe",
  name: "管道",
  category: "图形",
  props: [
    {
      name: "flowSpeed",
      type: "number",
      label: "流速",
      group: "动画",
      defaultValue: 0,
      min: 0,
      max: 100,
    },
    {
      name: "flowDirection",
      type: "enum",
      label: "流向",
      group: "动画",
      defaultValue: "forward",
      options: [
        { label: "正向", value: "forward" },
        { label: "反向", value: "backward" },
      ],
    },
    {
      name: "pipeWidth",
      type: "number",
      label: "管道宽度",
      group: "基础",
      defaultValue: 10,
      min: 1,
      max: 100,
    },
  ],
});

// Canvas.Text 文字标注
registerManifest({
  type: "Canvas.Text",
  name: "文字标注",
  category: "图形",
  props: [
    {
      name: "text",
      type: "string",
      label: "文本",
      group: "基础",
      defaultValue: "文字",
    },
    {
      name: "fontSize",
      type: "number",
      label: "字号",
      group: "样式",
      defaultValue: 14,
      min: 8,
      max: 200,
    },
    {
      name: "fontFamily",
      type: "enum",
      label: "字体",
      group: "样式",
      defaultValue: "system-ui",
      options: [
        { label: "系统默认", value: "system-ui" },
        { label: "思源黑体", value: "Source Han Sans SC" },
        { label: "等宽字体", value: "monospace" },
        { label: "数码字体", value: "DSEG7 Classic" },
      ],
    },
    {
      name: "align",
      type: "enum",
      label: "对齐",
      group: "样式",
      defaultValue: "left",
      options: [
        { label: "左对齐", value: "left" },
        { label: "居中", value: "center" },
        { label: "右对齐", value: "right" },
      ],
    },
  ],
});

// EChart 图表组件
registerManifest({
  type: "EChart",
  name: "图表",
  category: "图表",
  defaultSize: { width: 480, height: 300 },
  events: [
    { name: "click", label: "点击", description: "鼠标点击图表元素触发" },
    { name: "dblclick", label: "双击", description: "鼠标双击图表元素触发" },
    {
      name: "mouseover",
      label: "鼠标移入",
      description: "鼠标移入图表元素触发",
    },
    {
      name: "mouseout",
      label: "鼠标移出",
      description: "鼠标移出图表元素触发",
    },
    {
      name: "mousemove",
      label: "鼠标移动",
      description: "鼠标在图表内移动触发",
    },
    { name: "mousedown", label: "鼠标按下", description: "鼠标按下触发" },
    { name: "mouseup", label: "鼠标抬起", description: "鼠标抬起触发" },
    {
      name: "legendselectchanged",
      label: "图例切换",
      description: "图例选择变化触发",
    },
    { name: "datazoom", label: "缩放", description: "数据缩放触发" },
    { name: "brushselected", label: "刷选", description: "刷选触发" },
    { name: "finished", label: "渲染完成", description: "渲染完成触发" },
    { name: "rendered", label: "渲染中", description: "渲染过程中触发" },
  ],
  props: [
    {
      name: "option",
      type: "object",
      label: "详细配置",
      group: "详细配置",
      editor: "code",
      language: "javascript",
      height: "260px",
      defaultValue: `// 折线图模板
const option = {
  title: { text: "折线图" },
  tooltip: { trigger: "axis" },
  xAxis: { type: "category", data: ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"] },
  yAxis: { type: "value" },
  series: [
    { name: "访问量", type: "line", data: [120, 200, 150, 80, 70, 110, 130] },
  ],
};
return option;

// 柱状图模板
// const option = {
//   title: { text: "柱状图" },
//   tooltip: { trigger: "axis" },
//   xAxis: { type: "category", data: ["A", "B", "C", "D", "E"] },
//   yAxis: { type: "value" },
//   series: [{ type: "bar", data: [12, 20, 15, 8, 25] }],
// };
// return option;

// 饼图模板
// const option = {
//   title: { text: "饼图", left: "center" },
//   tooltip: { trigger: "item" },
//   legend: { bottom: 0 },
//   series: [
//     {
//       type: "pie",
//       radius: ["30%", "70%"],
//       data: [
//         { value: 1048, name: "A" },
//         { value: 735, name: "B" },
//         { value: 580, name: "C" },
//         { value: 484, name: "D" },
//         { value: 300, name: "E" },
//       ],
//     },
//   ],
// };
// return option;

// 面积图模板
// const option = {
//   title: { text: "面积图" },
//   tooltip: { trigger: "axis" },
//   xAxis: { type: "category", boundaryGap: false, data: ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"] },
//   yAxis: { type: "value" },
//   series: [{ type: "line", areaStyle: {}, data: [820, 932, 901, 934, 1290, 1330, 1320] }],
// };
// return option;`,
      // 雷达图模板
      // const option = {
      //   title: { text: "雷达图" },
      //   tooltip: {},
      //   legend: { data: ["预算分配", "实际开销"] },
      //   radar: {
      //     indicator: [
      //       { name: "销售", max: 6500 },
      //       { name: "管理", max: 16000 },
      //       { name: "技术", max: 30000 },
      //       { name: "客服", max: 38000 },
      //       { name: "研发", max: 52000 },
      //       { name: "市场", max: 25000 },
      //     ],
      //   },
      //   series: [
      //     {
      //       type: "radar",
      //       data: [
      //         { value: [4300, 10000, 28000, 35000, 50000, 19000], name: "预算分配" },
      //         { value: [5000, 14000, 28000, 31000, 42000, 21000], name: "实际开销" },
      //       ],
      //     },
      //   ],
      // };
      // return option;

      // 散点图模板
      // const option = {
      //   title: { text: "散点图" },
      //   tooltip: { trigger: "item" },
      //   xAxis: {},
      //   yAxis: {},
      //   series: [
      //     {
      //       type: "scatter",
      //       data: [
      //         [10, 8], [15, 12], [18, 16], [20, 6], [25, 18], [30, 14],
      //       ],
      //     },
      //   ],
      // };
      // return option;

      // 仪表盘模板
      // const option = {
      //   title: { text: "仪表盘" },
      //   series: [
      //     {
      //       type: "gauge",
      //       progress: { show: true },
      //       detail: { valueAnimation: true, formatter: "{value}%" },
      //       data: [{ value: 70, name: "完成率" }],
      //     },
      //   ],
      // };
      // return option;`',
    },
  ],
});
