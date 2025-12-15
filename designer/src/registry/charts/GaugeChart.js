/**
 * GaugeChart - 仪表盘组件
 *
 * 基于 ECharts 的仪表盘
 * 分类：图表组件
 */

import GaugeChartComponent from './GaugeChart.vue';

export default {
    type: 'GaugeChart',
    name: '仪表盘',
    category: '图表组件',
    icon: 'speedometer',
    thumbnail: null,
    tags: ['图表', '仪表盘', 'gauge', 'chart', 'echarts'],
    description: 'ECharts 仪表盘，用于展示进度或指标',
    component: GaugeChartComponent,

    defaultProps: {
        title: '仪表盘',
        showTitle: true,
        value: 75,
        min: 0,
        max: 100,
        unit: '%',
        splitNumber: 10,
        startAngle: 225,
        endAngle: -45,
    },

    defaultStyle: {
        position: 'absolute',
        left: 100,
        top: 100,
        width: 300,
        height: 300,
        zIndex: 1,
    },

    propsSchema: {},

    eventsSchema: {},

    container: false,
    version: '1.0.0',
};
