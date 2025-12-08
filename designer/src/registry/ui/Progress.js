/**
 * Progress - 进度条组件
 *
 * Element Plus 进度条组件
 * 分类：Element 组件
 */

export default {
    type: 'Progress',
    name: '进度条',
    category: 'Element 组件',
    icon: 'stats-chart',
    thumbnail: null,
    tags: ['进度条', 'progress', 'UI', 'element'],
    description: 'Element Plus 进度条组件，用于显示操作进度',

    defaultProps: {
        percentage: 0,
        type: 'line',
        strokeWidth: 6,
        status: '',
        color: '#409EFF',
        showText: true,
        textInside: false,
    },

    defaultStyle: {
        position: 'absolute',
        left: 100,
        top: 100,
        width: 200,
        height: 20,
        zIndex: 1,
    },

    propsSchema: {
        percentage: {
            type: 'number',
            label: '百分比',
            group: '内容',
            default: 0,
            min: 0,
            max: 100,
        },
        type: {
            type: 'enum',
            label: '类型',
            group: '外观',
            options: [
                { value: 'line', label: '线形' },
                { value: 'circle', label: '环形' },
                { value: 'dashboard', label: '仪表盘' },
            ],
            default: 'line',
        },
        strokeWidth: {
            type: 'number',
            label: '进度条宽度',
            group: '外观',
            default: 6,
            min: 1,
            max: 20,
            unit: 'px',
        },
        status: {
            type: 'enum',
            label: '状态',
            group: '外观',
            options: [
                { value: '', label: '默认' },
                { value: 'success', label: '成功' },
                { value: 'exception', label: '异常' },
                { value: 'warning', label: '警告' },
            ],
            default: '',
        },
        color: {
            type: 'color',
            label: '进度条颜色',
            group: '外观',
            default: '#409EFF',
        },
        showText: {
            type: 'boolean',
            label: '显示文字',
            group: '外观',
            default: true,
        },
        textInside: {
            type: 'boolean',
            label: '文字内显',
            group: '外观',
            default: false,
            visible: (props) => props.type === 'line',
        },
    },

    eventsSchema: {},

    container: false,
    version: '1.0.0',
};
