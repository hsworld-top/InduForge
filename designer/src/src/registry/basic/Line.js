/**
 * Line - 直线组件
 *
 * 基础图形组件，绘制直线
 * 分类：基础组件
 */

export default {
    type: 'Line',
    name: '直线',
    category: '基础组件',
    icon: 'remove',
    thumbnail: null,
    tags: ['图形', '直线', 'line', '线条'],
    description: '直线图形，支持线条样式、颜色、粗细等属性',

    defaultProps: {
        stroke: '#303133',
        strokeWidth: 2,
        strokeDasharray: '',
        opacity: 1,
        startX: 0,
        startY: 0,
        endX: 100,
        endY: 0,
    },

    defaultStyle: {
        position: 'absolute',
        left: 100,
        top: 100,
        width: 100,
        height: 2,
        zIndex: 1,
    },

    propsSchema: {
        stroke: {
            type: 'color',
            label: '线条颜色',
            group: '外观',
            default: '#303133',
        },
        strokeWidth: {
            type: 'number',
            label: '线条宽度',
            group: '外观',
            min: 1,
            max: 20,
            step: 1,
            default: 2,
            unit: 'px',
        },
        strokeDasharray: {
            type: 'string',
            label: '虚线样式',
            group: '外观',
            default: '',
            placeholder: '例如: 5,5',
        },
        opacity: {
            type: 'number',
            label: '不透明度',
            group: '外观',
            min: 0,
            max: 1,
            step: 0.1,
            default: 1,
        },
    },

    eventsSchema: {
        click: { label: '点击' },
    },

    container: false,
    version: '1.0.0',
};
