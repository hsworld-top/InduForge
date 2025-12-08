/**
 * PieChart - 饼图组件
 *
 * 基于 ECharts 的饼图
 * 分类：图表组件
 */

export default {
    type: 'PieChart',
    name: '饼图',
    category: '图表组件',
    icon: 'pie-chart',
    thumbnail: null,
    tags: ['图表', '饼图', 'pie', 'chart', 'echarts'],
    description: 'ECharts 饼图，用于展示数据占比',

    defaultProps: {
        title: '饼图',
        showTitle: true,
        showLegend: true,
        roseType: false,
        radius: ['0%', '75%'],
        data: [
            { name: '类目1', value: 335 },
            { name: '类目2', value: 310 },
            { name: '类目3', value: 234 },
            { name: '类目4', value: 135 },
            { name: '类目5', value: 148 },
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
            default: '饼图',
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
        roseType: {
            type: 'boolean',
            label: '南丁格尔图',
            group: '外观',
            default: false,
        },
        radius: {
            type: 'array',
            label: '半径',
            group: '外观',
            default: ['0%', '75%'],
        },
        data: {
            type: 'array',
            label: '数据',
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
