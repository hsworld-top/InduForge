/**
 * Polygon - 多边形组件
 *
 * 基础图形组件，绘制多边形
 * 分类：基础组件
 */

export default {
    type: 'Polygon',
    name: '多边形',
    category: '基础组件',
    icon: 'triangle-outline',
    thumbnail: null,
    tags: ['图形', '多边形', 'polygon', '三角形'],
    description: '多边形图形，支持自定义顶点坐标',

    defaultProps: {
        fill: '#F56C6C',
        stroke: '#303133',
        strokeWidth: 1,
        opacity: 1,
        points: '50,0 100,100 0,100', // 默认三角形
        sides: 3,
    },

    defaultStyle: {
        position: 'absolute',
        left: 100,
        top: 100,
        width: 100,
        height: 100,
        zIndex: 1,
    },

    propsSchema: {
        fill: {
            type: 'color',
            label: '填充颜色',
            group: '外观',
            default: '#F56C6C',
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
        sides: {
            type: 'number',
            label: '边数',
            group: '形状',
            min: 3,
            max: 12,
            step: 1,
            default: 3,
        },
        points: {
            type: 'string',
            label: '顶点坐标',
            group: '形状',
            default: '50,0 100,100 0,100',
            placeholder: 'x1,y1 x2,y2 x3,y3',
        },
    },

    eventsSchema: {
        click: { label: '点击' },
        dblclick: { label: '双击' },
    },

    container: false,
    version: '1.0.0',
};
