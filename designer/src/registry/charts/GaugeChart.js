/**
 * GaugeChart - 仪表盘组件
 *
 * 基于 ECharts 的仪表盘
 * 分类：图表组件
 */

export default {
    type: 'GaugeChart',
    name: '仪表盘',
    category: '图表组件',
    icon: 'speedometer',
    thumbnail: null,
    tags: ['图表', '仪表盘', 'gauge', 'chart', 'echarts'],
    description: 'ECharts 仪表盘，用于展示进度或指标',

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

    propsSchema: {
        title: {
            type: 'string',
            label: '标题',
            group: '基础',
            default: '仪表盘',
        },
        showTitle: {
            type: 'boolean',
            label: '显示标题',
            group: '基础',
            default: true,
        },
        value: {
            type: 'number',
            label: '当前值',
            group: '数据',
            default: 75,
        },
        min: {
            type: 'number',
            label: '最小值',
            group: '范围',
            default: 0,
        },
        max: {
            type: 'number',
            label: '最大值',
            group: '范围',
            default: 100,
        },
        unit: {
            type: 'string',
            label: '单位',
            group: '外观',
            default: '%',
        },
        splitNumber: {
            type: 'number',
            label: '刻度数量',
            group: '外观',
            default: 10,
            min: 1,
            max: 20,
        },
        startAngle: {
            type: 'number',
            label: '起始角度',
            group: '外观',
            default: 225,
            min: 0,
            max: 360,
        },
        endAngle: {
            type: 'number',
            label: '结束角度',
            group: '外观',
            default: -45,
            min: -360,
            max: 360,
        },
    },

    eventsSchema: {
        click: { label: '点击', description: '点击图表元素时触发' },
    },

    container: false,
    version: '1.0.0',
};
