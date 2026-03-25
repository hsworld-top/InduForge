<!--
  EChart - ECharts 图表封装
  支持柱状、折线、饼图、雷达、散点、仪表盘等，支持 option 绑定
-->
<script setup>
import {
  BarChart,
  GaugeChart,
  LineChart,
  PieChart,
  RadarChart,
  ScatterChart,
} from "echarts/charts";
import {
  DatasetComponent,
  GridComponent,
  LegendComponent,
  RadarComponent,
  TitleComponent,
  TooltipComponent,
} from "echarts/components";
import { use } from "echarts/core";
import { CanvasRenderer } from "echarts/renderers";
import { computed, onMounted, ref, watch } from "vue";
import VChart from "vue-echarts";

const props = defineProps({
  option: {
    type: [Object, String],
    default: () => ({
      title: { text: "示例图表" },
      tooltip: {},
      xAxis: { type: "category", data: ["A", "B", "C", "D"] },
      yAxis: { type: "value" },
      series: [{ type: "bar", data: [12, 20, 15, 8] }],
    }),
  },
});

use([
  CanvasRenderer,
  BarChart,
  LineChart,
  PieChart,
  RadarChart,
  ScatterChart,
  GaugeChart,
  TitleComponent,
  TooltipComponent,
  LegendComponent,
  GridComponent,
  DatasetComponent,
  RadarComponent,
]);

const chartRef = ref(null);
const pendingOption = ref(null);
const updateOptions = computed(() => ({ notMerge: true }));
const getInstance = () => chartRef.value?.getEChartsInstance?.();

/** 规范化 setOption 参数（兼容多种调用形式） */
function normalizeSetOptionArgs(
  notMergeOrOpts,
  lazyUpdate,
  silent,

  replaceMerge,
) {
  if (notMergeOrOpts && typeof notMergeOrOpts === "object") {
    return { ...notMergeOrOpts };
  }
  const opts = {
    notMerge: typeof notMergeOrOpts === "boolean" ? notMergeOrOpts : notMergeOrOpts === undefined,
    lazyUpdate: Boolean(lazyUpdate),
    silent: Boolean(silent),
  };
  if (replaceMerge) opts.replaceMerge = replaceMerge;
  return opts;
}
function applyPendingOption() {
  const payload = pendingOption.value;
  if (!payload) return;
  const instance = getInstance();
  if (!instance) return;
  pendingOption.value = null;
  if (payload.opts?.notMerge) {
    instance.clear();
  }
  instance.setOption(payload.option || {}, payload.opts || {});
}
function setOption(option, notMergeOrOpts, lazyUpdate, silent, replaceMerge) {
  const opts = normalizeSetOptionArgs(notMergeOrOpts, lazyUpdate, silent, replaceMerge);
  if (opts.notMerge) {
    opts.lazyUpdate = false;
  }
  const instance = getInstance();
  if (!instance) {
    pendingOption.value = { option, opts };
    return;
  }
  if (opts.notMerge) {
    instance.clear();
  }
  instance.setOption(option || {}, opts);
}
function callECharts(method, ...args) {
  const instance = getInstance();
  if (!instance) return;
  const target = instance[method];
  if (typeof target !== "function") return;
  return target.apply(instance, args);
}
function parseOptionSource(source) {
  const code = String(source ?? "").trim();
  if (!code) return {};

  try {
    if (/^\s*[[{]/.test(code)) {
      return new Function(`"use strict"; return (${code});`)();
    }
    if (/\boption\s*=/.test(code)) {
      return new Function(`"use strict"; ${code}; return option;`)();
    }
    if (/\breturn\b/.test(code)) {
      return new Function(`"use strict"; ${code}`)();
    }
  } catch (error) {
    // ignore and fallback to JSON
  }

  try {
    return JSON.parse(code);
  } catch (error) {
    return {};
  }
}

onMounted(() => {
  applyPendingOption();
});

watch(chartRef, () => {
  applyPendingOption();
});

const chartOption = computed(() => {
  if (typeof props.option === "string") {
    return parseOptionSource(props.option);
  }
  return props.option || {};
});

defineExpose({ setOption, callECharts, getInstance });
</script>

<template>
  <VChart
    ref="chartRef"
    class="echart-canvas"
    :option="chartOption"
    :update-options="updateOptions"
    autoresize
  />
</template>

<style scoped>
.echart-canvas {
  width: 100%;
  height: 100%;
}
</style>
