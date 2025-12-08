/**
 * Ellipse - 椭圆组件
 *
 * 基础图形组件，绘制椭圆
 * 分类：基础组件
 */

export default {
    type: 'Ellipse',
    name: '椭圆',
    category: '基础组件',
    icon: 'ellipse-outline',
    thumbnail: null,
    tags: ['图形', '椭圆', 'ellipse'],
    description: '椭圆图形，支持填充色、边框等属性',

    defaultProps: {
        fill: '#E6A23C',
        stroke: '#303133',
        strokeWidth: 1,
        opacity: 1,
    },

    defaultStyle: {
        position: 'absolute',
        left: 100,
        top: 100,
        width: 120,
        height: 80,
        zIndex: 1,
    },

    propsSchema: {
        fill: {
            type: 'color',
            label: '填充颜色',
            group: '外观',
            default: '#E6A23C',
        },
        stroke: {
            type: 'color',
            label: '边框颜色',
            group: '外观',
            default: '#303133',
        },
        strokeWidth: {
            type: 'number',
            label: '边框宽度',
            group: '外观',
            min: 0,
            max: 20,
            step: 1,
            default: 1,
            unit: 'px',
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
        dblclick: { label: '双击' },
        mouseenter: { label: '鼠标进入' },
        mouseleave: { label: '鼠标离开' },
    },

    container: false,
    version: '1.0.0',
};
