/**
 * ScatterChart - 散点图组件
 *
 * 基于 ECharts 的散点图
 * 分类：图表组件
 */

export default {
    type: 'ScatterChart',
    name: '散点图',
    category: '图表组件',
    icon: 'ellipse',
    thumbnail: null,
    tags: ['图表', '散点图', 'scatter', 'chart', 'echarts'],
    description: 'ECharts 散点图，用于展示数据分布',

    defaultProps: {
        title: '散点图',
        showTitle: true,
        showLegend: true,
        showGrid: true,
        symbolSize: 10,
        series: [
            {
                name: '系列1',
                data: [
                    [10.0, 8.04],
                    [8.0, 6.95],
                    [13.0, 7.58],
                    [9.0, 8.81],
                    [11.0, 8.33],
                    [14.0, 9.96],
                    [6.0, 7.24],
                    [4.0, 4.26],
                    [12.0, 10.84],
                    [7.0, 4.82],
                    [5.0, 5.68],
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

    propsSchema: {
        title: {
            type: 'string',
            label: '标题',
            group: '基础',
            default: '散点图',
        },
        showTitle: {
            type: 'boolean',
            label: '显示标题',
            group: '基础',
            default: true,
        },
        showLegend: {
            type: 'boolean',
            label: '显示图例',
            group: '基础',
            default: true,
        },
        showGrid: {
            type: 'boolean',
            label: '显示网格',
            group: '外观',
            default: true,
        },
        symbolSize: {
            type: 'number',
            label: '标记大小',
            group: '外观',
            default: 10,
            min: 1,
            max: 50,
        },
        series: {
            type: 'array',
            label: '系列数据',
            group: '数据',
            default: [],
        },
    },

    eventsSchema: {
        click: { label: '点击', description: '点击图表元素时触发' },
        legendselectchanged: { label: '图例选择', description: '图例选择改变时触发' },
    },

    container: false,
    version: '1.0.0',
};
