/**
 * LineChart - 折线图组件
 *
 * 基于 ECharts 的折线图
 * 分类：图表组件
 */

export default {
    type: 'LineChart',
    name: '折线图',
    category: '图表组件',
    icon: 'trending-up',
    thumbnail: null,
    tags: ['图表', '折线图', 'line', 'chart', 'echarts'],
    description: 'ECharts 折线图，用于展示数据趋势',

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

    propsSchema: {
        title: {
            type: 'string',
            label: '标题',
            group: '基础',
            default: '折线图',
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
        smooth: {
            type: 'boolean',
            label: '平滑曲线',
            group: '外观',
            default: false,
        },
        xAxisData: {
            type: 'array',
            label: 'X轴数据',
            group: '数据',
            default: ['周一', '周二', '周三', '周四', '周五', '周六', '周日'],
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
