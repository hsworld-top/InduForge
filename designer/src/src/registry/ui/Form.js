/**
 * Form - 表单组件
 *
 * Element Plus 表单组件
 * 分类：Element 组件
 */

import FormComponent from './Form.vue';

export default {
    type: 'Form',
    name: '表单',
    category: 'Element 组件',
    icon: 'document',
    thumbnail: null,
    tags: ['表单', 'form', 'UI', 'element'],
    description: 'Element Plus 表单组件，用于数据收集和验证',
    component: FormComponent,

    defaultProps: {
        model: {},
        rules: {},
        labelWidth: '80px',
        labelPosition: 'right',
        inline: false,
        size: 'default',
        disabled: false,
        validateOnRuleChange: true,
        hideRequiredAsterisk: false,
        showMessage: true,
        inlineMessage: false,
        statusIcon: false,
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
        labelWidth: {
            type: 'string',
            label: '标签宽度',
            group: '外观',
            default: '80px',
        },
        labelPosition: {
            type: 'enum',
            label: '标签位置',
            group: '外观',
            options: [
                { value: 'left', label: '左侧' },
                { value: 'right', label: '右侧' },
                { value: 'top', label: '顶部' },
            ],
            default: 'right',
        },
        inline: {
            type: 'boolean',
            label: '行内表单',
            group: '外观',
            default: false,
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
        hideRequiredAsterisk: {
            type: 'boolean',
            label: '隐藏必填星号',
            group: '外观',
            default: false,
        },
        showMessage: {
            type: 'boolean',
            label: '显示校验信息',
            group: '验证',
            default: true,
        },
        inlineMessage: {
            type: 'boolean',
            label: '行内显示信息',
            group: '验证',
            default: false,
        },
        statusIcon: {
            type: 'boolean',
            label: '显示状态图标',
            group: '验证',
            default: false,
        },
    },

    eventsSchema: {
        validate: { label: '校验', description: '任一表单项被校验后触发' },
    },

    container: true,
    version: '1.0.0',
};
