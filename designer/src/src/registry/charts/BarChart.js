/**
 * BarChart - 柱状图组件
 *
 * 基于 ECharts 的柱状图
 * 分类：图表组件
 */

import BarChartComponent from './BarChart.vue';

export default {
    type: 'BarChart',
    name: '柱状图',
    category: '图表组件',
    icon: 'bar-chart',
    thumbnail: null,
    tags: ['图表', '柱状图', 'bar', 'chart', 'echarts'],
    description: 'ECharts 柱状图，用于展示数据对比',
    component: BarChartComponent,

    defaultProps: {
        title: '柱状图',
        showTitle: true,
        showLegend: true,
        xAxisData: ['一月', '二月', '三月', '四月', '五月'],
        series: [
            {
                name: '系列1',
                data: [120, 200, 150, 80, 70],
                color: '#67C23A',
            },
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
