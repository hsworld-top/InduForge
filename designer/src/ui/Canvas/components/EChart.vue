<template>
  <VChart class="echart-canvas" :option="chartOption" autoresize />
</template>

<script setup>
import { computed } from "vue";
import { use } from "echarts/core";
import { CanvasRenderer } from "echarts/renderers";
import {
  BarChart,
  LineChart,
  PieChart,
  RadarChart,
  ScatterChart,
  GaugeChart,
} from "echarts/charts";
import {
  TitleComponent,
  TooltipComponent,
  LegendComponent,
  GridComponent,
  DatasetComponent,
  RadarComponent,
} from "echarts/components";
import VChart from "vue-echarts";

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

const parseOptionSource = (source) => {
  const code = String(source ?? "").trim();
  if (!code) return {};

  try {
    if (/^\s*[\[{]/.test(code)) {
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
};

const chartOption = computed(() => {
  if (typeof props.option === "string") {
    return parseOptionSource(props.option);
  }
  return props.option || {};
});
</script>

<style scoped>
.echart-canvas {
  width: 100%;
  height: 100%;
}
</style>