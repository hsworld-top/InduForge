/**
 * Select - 选择器组件
 *
 * Element Plus 选择器组件
 * 分类：Element 组件
 */

import SelectComponent from './Select.vue';

export default {
    type: 'Select',
    name: '选择器',
    category: 'Element 组件',
    icon: 'chevron-down',
    thumbnail: null,
    tags: ['选择器', 'select', 'dropdown', 'UI', 'element', '表单'],
    description: 'Element Plus 选择器组件，支持单选和多选',
    component: SelectComponent,

    defaultProps: {
        value: '',
        placeholder: '请选择',
        size: 'default',
        disabled: false,
        clearable: false,
        multiple: false,
        filterable: false,
        options: [
            { label: '选项1', value: '1' },
            { label: '选项2', value: '2' },
            { label: '选项3', value: '3' },
        ],
    },

    defaultStyle: {
        position: 'absolute',
        left: 100,
        top: 100,
        width: 200,
        height: 32,
        zIndex: 1,
    },

    propsSchema: {
        value: {
            type: 'string',
            label: '选中值',
            group: '内容',
            default: '',
        },
        placeholder: {
            type: 'string',
            label: '占位文本',
            group: '内容',
            default: '请选择',
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
        clearable: {
            type: 'boolean',
            label: '可清空',
            group: '功能',
            default: false,
        },
        multiple: {
            type: 'boolean',
            label: '多选',
            group: '功能',
            default: false,
        },
        filterable: {
            type: 'boolean',
            label: '可搜索',
            group: '功能',
            default: false,
        },
        options: {
            type: 'array',
            label: '选项列表',
            group: '数据',
            default: [
                { label: '选项1', value: '1' },
                { label: '选项2', value: '2' },
                { label: '选项3', value: '3' },
            ],
        },
    },

    eventsSchema: {
        change: { label: '改变', description: '选中值改变时触发' },
        visibleChange: { label: '下拉框显隐', description: '下拉框出现/隐藏时触发' },
        clear: { label: '清空' },
    },

    container: false,
    version: '1.0.0',
};
