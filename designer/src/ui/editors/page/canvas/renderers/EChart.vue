<!--
  EChart - ECharts 图表封装
  支持柱状、折线、饼图、雷达、散点、仪表盘等，支持 option 绑定
-->
<script setup lang="ts">
import { BarChart, GaugeChart, LineChart, PieChart, RadarChart, ScatterChart } from 'echarts/charts'
import {
  DatasetComponent,
  GridComponent,
  LegendComponent,
  RadarComponent,
  TitleComponent,
  TooltipComponent,
} from 'echarts/components'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { computed, onMounted, ref, watch } from 'vue'
import VChart from 'vue-echarts'

type EChartOptionLike = Record<string, unknown> | string

const props = withDefaults(defineProps<{ option?: EChartOptionLike }>(), {
  option: () => ({
    title: { text: '示例图表' },
    tooltip: {},
    xAxis: { type: 'category', data: ['A', 'B', 'C', 'D'] },
    yAxis: { type: 'value' },
    series: [{ type: 'bar', data: [12, 20, 15, 8] }],
  }),
})

const OPTION_LITERAL_RE = /^\s*[[{]/
const OPTION_ASSIGN_RE = /\boption\s*=/
const OPTION_RETURN_RE = /\breturn\b/

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
])

interface EChartPendingPayload {
  option: Record<string, unknown>
  opts: {
    notMerge?: boolean
    lazyUpdate?: boolean
    silent?: boolean
    replaceMerge?: unknown
  }
}

interface EChartInstanceLike {
  clear?: () => void
  setOption?: (option: Record<string, unknown>, opts?: Record<string, unknown>) => void
  [key: string]: unknown
}

const chartRef = ref<{ getEChartsInstance?: () => EChartInstanceLike | undefined } | null>(null)
const pendingOption = ref<EChartPendingPayload | null>(null)
const updateOptions = computed(() => ({ notMerge: true }))
const getInstance = (): EChartInstanceLike | undefined => chartRef.value?.getEChartsInstance?.()

/** 规范化 setOption 参数（兼容多种调用形式） */
function normalizeSetOptionArgs(
  notMergeOrOpts: boolean | Record<string, unknown> | undefined,
  lazyUpdate?: boolean,
  silent?: boolean,
  replaceMerge?: unknown,
) {
  if (notMergeOrOpts && typeof notMergeOrOpts === 'object') {
    return {
      notMerge: Boolean((notMergeOrOpts as { notMerge?: boolean }).notMerge),
      lazyUpdate: Boolean((notMergeOrOpts as { lazyUpdate?: boolean }).lazyUpdate),
      silent: Boolean((notMergeOrOpts as { silent?: boolean }).silent),
      replaceMerge,
    }
  }
  const opts: {
    notMerge: boolean
    lazyUpdate: boolean
    silent: boolean
    replaceMerge?: unknown
  } = {
    notMerge: typeof notMergeOrOpts === 'boolean' ? notMergeOrOpts : notMergeOrOpts === undefined,
    lazyUpdate: Boolean(lazyUpdate),
    silent: Boolean(silent),
  }
  if (replaceMerge !== undefined) opts.replaceMerge = replaceMerge
  return opts
}
function applyPendingOption() {
  const payload = pendingOption.value
  if (!payload) return
  const instance = getInstance()
  if (!instance) return
  pendingOption.value = null
  if (payload.opts?.notMerge) {
    instance.clear?.()
  }
  instance.setOption?.(payload.option || {}, payload.opts || {})
}
function setOption(
  option: Record<string, unknown>,
  notMergeOrOpts?: boolean | Record<string, unknown>,
  lazyUpdate?: boolean,
  silent?: boolean,
  replaceMerge?: unknown,
) {
  const opts = normalizeSetOptionArgs(notMergeOrOpts, lazyUpdate, silent, replaceMerge)
  if (opts.notMerge) {
    opts.lazyUpdate = false
  }
  const instance = getInstance()
  if (!instance) {
    pendingOption.value = {
      option: (option || {}) as Record<string, unknown>,
      opts,
    }
    return
  }
  if (opts.notMerge) {
    instance.clear?.()
  }
  instance.setOption?.(option || {}, opts)
}
function callECharts(method: string, ...args: unknown[]) {
  const instance = getInstance()
  if (!instance) return
  const target = instance[method]
  if (typeof target !== 'function') return
  return (target as (...callArgs: unknown[]) => unknown).apply(instance, args)
}
function parseOptionSource(source: string) {
  const code = String(source ?? '').trim()
  if (!code) return {}

  try {
    if (OPTION_LITERAL_RE.test(code)) {
      return new Function(`"use strict"; return (${code});`)()
    }
    if (OPTION_ASSIGN_RE.test(code)) {
      return new Function(`"use strict"; ${code}; return option;`)()
    }
    if (OPTION_RETURN_RE.test(code)) {
      return new Function(`"use strict"; ${code}`)()
    }
  } catch {
    // ignore and fallback to JSON
  }

  try {
    return JSON.parse(code)
  } catch {
    return {}
  }
}

onMounted(() => {
  applyPendingOption()
})

watch(chartRef, () => {
  applyPendingOption()
})

const chartOption = computed<Record<string, unknown>>(() => {
  if (typeof props.option === 'string') {
    return parseOptionSource(props.option)
  }
  return props.option || {}
})

defineExpose({ setOption, callECharts, getInstance })
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
