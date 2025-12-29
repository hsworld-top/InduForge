/**
 * Divider - 分割线组件
 *
 * 用于分隔内容的分割线
 * 分类：布局组件
 */

export default {
    type: 'Divider',
    name: '分割线',
    category: '布局组件',
    icon: 'remove',
    thumbnail: null,
    tags: ['布局', '分割线', 'divider', '分隔'],
    description: '用于分隔内容的分割线',

    defaultProps: {
        direction: 'horizontal',
        contentPosition: 'center',
        content: '',
        borderStyle: 'solid',
    },

    defaultStyle: {
        position: 'absolute',
        left: 100,
        top: 100,
        width: 300,
        height: 1,
        zIndex: 1,
    },

    propsSchema: {
        direction: {
            type: 'enum',
            label: '方向',
            group: '布局',
            options: [
                { value: 'horizontal', label: '水平' },
                { value: 'vertical', label: '垂直' },
            ],
            default: 'horizontal',
        },
        contentPosition: {
            type: 'enum',
            label: '文字位置',
            group: '布局',
            options: [
                { value: 'left', label: '左侧' },
                { value: 'center', label: '居中' },
                { value: 'right', label: '右侧' },
            ],
            default: 'center',
            visible: (props) => props.direction === 'horizontal' && props.content,
        },
        content: {
            type: 'string',
            label: '文字内容',
            group: '内容',
            default: '',
        },
        borderStyle: {
            type: 'enum',
            label: '线条样式',
            group: '外观',
            options: [
                { value: 'solid', label: '实线' },
                { value: 'dashed', label: '虚线' },
                { value: 'dotted', label: '点线' },
            ],
            default: 'solid',
        },
    },

    eventsSchema: {},

    container: false,
    version: '1.0.0',
};
