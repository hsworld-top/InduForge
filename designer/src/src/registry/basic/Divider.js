/**
 * Divider - 分割线组件
 *
 * 基础分割线组件，用于内容分隔
 * 分类：基础组件
 */

export default {
    type: 'Divider',
    name: '分割线',
    category: '基础组件',
    icon: 'minus',
    thumbnail: null,
    tags: ['分割线', 'divider', '分隔符', 'hr'],
    description: '分割线组件，用于内容分隔',

    defaultProps: {
        direction: 'horizontal',
        borderStyle: 'solid',
        borderWidth: 1,
        borderColor: '#DCDFE6',
        contentPosition: 'center',
        content: '',
    },

    defaultStyle: {
        position: 'absolute',
        left: 100,
        top: 100,
        width: 200,
        height: 1,
        zIndex: 1,
    },

    propsSchema: {
        direction: {
            type: 'select',
            label: '方向',
            group: '布局',
            options: [
                { label: '水平', value: 'horizontal' },
                { label: '垂直', value: 'vertical' },
            ],
            default: 'horizontal',
        },
        borderStyle: {
            type: 'select',
            label: '线条样式',
            group: '外观',
            options: [
                { label: '实线', value: 'solid' },
                { label: '虚线', value: 'dashed' },
                { label: '点线', value: 'dotted' },
            ],
            default: 'solid',
        },
        borderWidth: {
            type: 'number',
            label: '线条宽度',
            group: '外观',
            min: 1,
            max: 10,
            step: 1,
            default: 1,
            unit: 'px',
        },
        borderColor: {
            type: 'color',
            label: '线条颜色',
            group: '外观',
            default: '#DCDFE6',
        },
        contentPosition: {
            type: 'select',
            label: '文字位置',
            group: '内容',
            options: [
                { label: '左侧', value: 'left' },
                { label: '居中', value: 'center' },
                { label: '右侧', value: 'right' },
            ],
            default: 'center',
        },
        content: {
            type: 'text',
            label: '文字内容',
            group: '内容',
            default: '',
        },
    },

    eventsSchema: {},

    container: false,
    version: '1.0.0',
};
