/**
 * BarChart - 柱状图组件
 *
 * 基于 ECharts 的柱状图
 * 分类：图表组件
 */

export default {
    type: 'BarChart',
    name: '柱状图',
    category: '图表组件',
    icon: 'bar-chart',
    thumbnail: null,
    tags: ['图表', '柱状图', 'bar', 'chart', 'echarts'],
    description: 'ECharts 柱状图，用于展示数据对比',

    defaultProps: {
        title: '柱状图',
        showTitle: true,
        showLegend: true,
        showGrid: true,
        direction: 'vertical',
        xAxisData: ['类目1', '类目2', '类目3', '类目4', '类目5'],
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

    propsSchema: {
        title: {
            type: 'string',
            label: '标题',
            group: '基础',
            default: '柱状图',
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
        direction: {
            type: 'enum',
            label: '方向',
            group: '外观',
            options: [
                { value: 'vertical', label: '垂直' },
                { value: 'horizontal', label: '水平' },
            ],
            default: 'vertical',
        },
        xAxisData: {
            type: 'array',
            label: 'X轴数据',
            group: '数据',
            default: ['类目1', '类目2', '类目3', '类目4', '类目5'],
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
