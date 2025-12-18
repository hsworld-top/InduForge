/**
 * Button - Element Plus 按钮组件
 */

import ButtonComponent from './Button.vue';

export default {
    type: 'Button',
    name: '按钮',
    category: 'Element 组件',
    icon: 'pointer',
    thumbnail: null,
    tags: ['按钮', 'button', 'UI', 'element'],
    description: 'Element Plus 按钮组件，支持多种类型和尺寸',
    component: ButtonComponent,

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
        domId: '',
        styleConfig: '',
        advancedConfig: '',
        events: {},
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
        domId: {
            type: 'string',
            label: 'DOM ID',
            group: '外观',
            default: '',
            description: '用于样式定制范围；也可在样式配置中使用 domId: xxx 设置',
        },
        styleConfig: {
            type: 'string',
            label: '样式配置',
            group: '外观',
            default: '',
        },
        advancedConfig: {
            type: 'string',
            label: '详细配置',
            group: '配置',
            default: '',
            description: 'JSON 或对象字面量，覆盖文本/类型/尺寸/禁用等按钮属性，支持 styleConfig',
        },
    },

    eventsSchema: {
        click: { label: '点击', description: '按钮被点击时触发' },
        mousedown: { label: '鼠标按下', description: '鼠标按下按钮时触发' },
        mouseup: { label: '鼠标抬起', description: '鼠标在按钮上抬起时触发' },
        mouseenter: { label: '鼠标进入', description: '鼠标移入按钮区域时触发' },
        mouseleave: { label: '鼠标离开', description: '鼠标移出按钮区域时触发' },
        focus: { label: '获得焦点', description: '按钮获得焦点时触发' },
        blur: { label: '失去焦点', description: '按钮失去焦点时触发' },
    },

    container: false,
    version: '1.0.0',
};
