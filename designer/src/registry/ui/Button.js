/**
 * Button - 按钮组件
 *
 * Element Plus 按钮组件
 * 分类：Element 组件
 */

export default {
    type: 'Button',
    name: '按钮',
    category: 'Element 组件',
    icon: 'pointer',
    thumbnail: null,
    tags: ['按钮', 'button', 'UI', 'element'],
    description: 'Element Plus 按钮组件，支持多种类型和尺寸',

    defaultProps: {
        text: '按钮',
        type: 'primary',
        size: 'default',
        disabled: false,
        loading: false,
        plain: false,
        round: false,
        circle: false,
        icon: '',
    },

    defaultStyle: {
        position: 'absolute',
        left: 100,
        top: 100,
        width: 80,
        height: 32,
        zIndex: 1,
    },

    propsSchema: {
        text: {
            type: 'string',
            label: '按钮文字',
            group: '内容',
            default: '按钮',
        },
        type: {
            type: 'enum',
            label: '类型',
            group: '外观',
            options: [
                { value: 'primary', label: '主要' },
                { value: 'success', label: '成功' },
                { value: 'warning', label: '警告' },
                { value: 'danger', label: '危险' },
                { value: 'info', label: '信息' },
                { value: 'default', label: '默认' },
            ],
            default: 'primary',
        },
        size: {
            type: 'enum',
            label: '尺寸',
            group: '外观',
            options: [
                { value: 'large', label: '大' },
                { value: 'default', label: '默认' },
                { value: 'small', label: '小' },
            ],
            default: 'default',
        },
        disabled: {
            type: 'boolean',
            label: '禁用',
            group: '状态',
            default: false,
        },
        loading: {
            type: 'boolean',
            label: '加载中',
            group: '状态',
            default: false,
        },
        plain: {
            type: 'boolean',
            label: '朴素按钮',
            group: '外观',
            default: false,
        },
        round: {
            type: 'boolean',
            label: '圆角按钮',
            group: '外观',
            default: false,
        },
        circle: {
            type: 'boolean',
            label: '圆形按钮',
            group: '外观',
            default: false,
        },
        icon: {
            type: 'string',
            label: '图标',
            group: '内容',
            default: '',
        },
    },

    eventsSchema: {
        click: {
            label: '点击',
            description: '按钮被点击时触发',
        },
    },

    container: false,
    version: '1.0.0',
};
