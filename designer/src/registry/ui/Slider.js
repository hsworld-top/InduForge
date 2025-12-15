/**
 * Slider - 滑块组件
 *
 * Element Plus 滑块组件
 * 分类：Element 组件
 */

import SliderComponent from './Slider.vue';

export default {
    type: 'Slider',
    name: '滑块',
    category: 'Element 组件',
    icon: 'options',
    thumbnail: null,
    tags: ['滑块', 'slider', 'range', 'UI', 'element', '表单'],
    description: 'Element Plus 滑块组件，用于数值选择',
    component: SliderComponent,

    defaultProps: {
        value: 0,
        min: 0,
        max: 100,
        step: 1,
        disabled: false,
        showStops: false,
        showTooltip: true,
        range: false,
    },

    defaultStyle: {
        position: 'absolute',
        left: 100,
        top: 100,
        width: 200,
        height: 40,
        zIndex: 1,
    },

    propsSchema: {
        value: {
            type: 'number',
            label: '当前值',
            group: '内容',
            default: 0,
        },
        min: {
            type: 'number',
            label: '最小值',
            group: '范围',
            default: 0,
        },
        max: {
            type: 'number',
            label: '最大值',
            group: '范围',
            default: 100,
        },
        step: {
            type: 'number',
            label: '步长',
            group: '范围',
            default: 1,
            min: 0,
        },
        disabled: {
            type: 'boolean',
            label: '禁用',
            group: '状态',
            default: false,
        },
        showStops: {
            type: 'boolean',
            label: '显示间断点',
            group: '外观',
            default: false,
        },
        showTooltip: {
            type: 'boolean',
            label: '显示提示',
            group: '外观',
            default: true,
        },
        range: {
            type: 'boolean',
            label: '范围选择',
            group: '功能',
            default: false,
        },
    },

    eventsSchema: {
        change: { label: '改变', description: '值改变时触发' },
        input: { label: '输入', description: '数据改变时触发' },
    },

    container: false,
    version: '1.0.0',
};
