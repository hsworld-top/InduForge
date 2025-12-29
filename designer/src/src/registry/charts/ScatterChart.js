/**
 * ScatterChart - 散点图组件
 *
 * 基于 ECharts 的散点图
 * 分类：图表组件
 */

import ScatterChartComponent from './ScatterChart.vue';

export default {
    type: 'ScatterChart',
    name: '散点图',
    category: '图表组件',
    icon: 'ellipse',
    thumbnail: null,
    tags: ['图表', '散点图', 'scatter', 'chart', 'echarts'],
    description: 'ECharts 散点图，用于展示分布',
    component: ScatterChartComponent,

    defaultProps: {
        title: '散点图',
        showTitle: true,
        showLegend: true,
        series: [
            {
                name: '系列1',
                data: [
                    [10, 20],
                    [20, 40],
                    [30, 10],
                    [40, 60],
                    [50, 30],
                ],
                color: '#E6A23C',
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
