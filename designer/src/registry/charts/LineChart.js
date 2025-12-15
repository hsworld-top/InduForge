/**
 * LineChart - 折线图组件
 *
 * 基于 ECharts 的折线图
 * 分类：图表组件
 */

import LineChartComponent from './LineChart.vue';

export default {
    type: 'LineChart',
    name: '折线图',
    category: '图表组件',
    icon: 'trending-up',
    thumbnail: null,
    tags: ['图表', '折线图', 'line', 'chart', 'echarts'],
    description: 'ECharts 折线图，用于展示数据趋势',
    component: LineChartComponent,

    defaultProps: {
        title: '折线图',
        showTitle: true,
        showLegend: true,
        showGrid: true,
        smooth: false,
        xAxisData: ['周一', '周二', '周三', '周四', '周五', '周六', '周日'],
        series: [
            {
                name: '系列1',
                data: [120, 200, 150, 80, 70, 110, 130],
                color: '#409EFF',
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
