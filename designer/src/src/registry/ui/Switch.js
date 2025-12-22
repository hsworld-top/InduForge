/**
 * Switch - 开关组件
 *
 * Element Plus 开关组件
 * 分类：Element 组件
 */

import SwitchComponent from './Switch.vue';

export default {
    type: 'Switch',
    name: '开关',
    category: 'Element 组件',
    icon: 'toggle',
    thumbnail: null,
    tags: ['开关', 'switch', 'toggle', 'UI', 'element', '表单'],
    description: 'Element Plus 开关组件，用于切换状态',
    component: SwitchComponent,

    defaultProps: {
        value: false,
        disabled: false,
        activeText: '',
        inactiveText: '',
        activeColor: '#409EFF',
        inactiveColor: '#C0CCDA',
        activeValue: true,
        inactiveValue: false,
    },

    defaultStyle: {
        position: 'absolute',
        left: 100,
        top: 100,
        width: 50,
        height: 24,
        zIndex: 1,
    },

    propsSchema: {
        value: {
            type: 'boolean',
            label: '开关状态',
            group: '内容',
            default: false,
        },
        disabled: {
            type: 'boolean',
            label: '禁用',
            group: '状态',
            default: false,
        },
        activeText: {
            type: 'string',
            label: '打开时文字',
            group: '内容',
            default: '',
        },
        inactiveText: {
            type: 'string',
            label: '关闭时文字',
            group: '内容',
            default: '',
        },
        activeColor: {
            type: 'color',
            label: '打开时颜色',
            group: '外观',
            default: '#409EFF',
        },
        inactiveColor: {
            type: 'color',
            label: '关闭时颜色',
            group: '外观',
            default: '#C0CCDA',
        },
    },

    eventsSchema: {
        change: { label: '改变', description: '开关状态改变时触发' },
    },

    container: false,
    version: '1.0.0',
};
