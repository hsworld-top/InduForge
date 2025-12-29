/**
 * PieChart - 饼图组件
 *
 * 基于 ECharts 的饼图
 * 分类：图表组件
 */

import PieChartComponent from './PieChart.vue';

export default {
    type: 'PieChart',
    name: '饼图',
    category: '图表组件',
    icon: 'pie-chart',
    thumbnail: null,
    tags: ['图表', '饼图', 'pie', 'chart', 'echarts'],
    description: 'ECharts 饼图，用于展示占比',
    component: PieChartComponent,

    defaultProps: {
        title: '饼图',
        showTitle: true,
        showLegend: true,
        roseType: false,
        radius: ['0%', '75%'],
        data: [
            { name: '类别1', value: 335 },
            { name: '类别2', value: 310 },
            { name: '类别3', value: 234 },
            { name: '类别4', value: 135 },
            { name: '类别5', value: 148 },
        ],
    },

    defaultStyle: {
        position: 'absolute',
        left: 100,
        top: 100,
        width: 400,
        height: 300,
        zIndex: 1,
    },

    propsSchema: {},

    eventsSchema: {},

    container: false,
    version: '1.0.0',
};
