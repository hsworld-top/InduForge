/**
 * Col - 栅格列组件
 *
 * 栅格列组件，配合 Row 使用，基于 24 栅格系统
 * 分类：布局组件
 *
 * Task 2.3: 重构布局组件注册
 */

import ColComponent from './Col.vue';

export default {
    type: 'Col',
    component: ColComponent,
    name: '栅格布局',
    category: '布局组件',
    icon: 'albums',
    thumbnail: null,
    tags: ['布局', '列', 'col', '栅格', 'grid'],
    description: '栅格列组件，配合栅格行使用，基于 24 栅格系统，宽度按比例自动计算',

    defaultProps: {
        span: 12,
        offset: 0,
        push: 0,
        pull: 0,
        backgroundColor: 'transparent',
        borderWidth: 1,
        borderStyle: 'dashed',
        borderColor: '#909399',
        borderRadius: 0,
        padding: 12,
        minHeight: 60,
    },

    defaultStyle: {
        position: 'relative',
        width: 'auto',
        height: 'auto',
        zIndex: 1,
    },

    propsSchema: {
        span: {
            type: 'number',
            label: '栅格占据列数',
            group: '布局',
            description: '栅格占据的列数（1-24），宽度 = span / 24 * 100%',
            default: 12,
            min: 1,
            max: 24,
            step: 1,
        },
        offset: {
            type: 'number',
            label: '左侧间隔',
            group: '布局',
            description: '栅格左侧的间隔格数',
            default: 0,
            min: 0,
            max: 24,
            step: 1,
        },
        push: {
            type: 'number',
            label: '向右移动',
            group: '布局',
            description: '栅格向右移动的格数',
            default: 0,
            min: 0,
            max: 24,
            step: 1,
        },
        pull: {
            type: 'number',
            label: '向左移动',
            group: '布局',
            description: '栅格向左移动的格数',
            default: 0,
            min: 0,
            max: 24,
            step: 1,
        },
        padding: {
            type: 'number',
            label: '内边距',
            group: '布局',
            default: 12,
            min: 0,
            max: 100,
            unit: 'px',
        },
        minHeight: {
            type: 'number',
            label: '最小高度',
            group: '布局',
            default: 60,
            min: 0,
            max: 1000,
            unit: 'px',
        },
        backgroundColor: {
            type: 'color',
            label: '背景颜色',
            group: '外观',
            default: 'transparent',
        },
        borderWidth: {
            type: 'number',
            label: '边框宽度',
            group: '外观',
            min: 0,
            max: 20,
            default: 1,
            unit: 'px',
        },
        borderStyle: {
            type: 'enum',
            label: '边框样式',
            group: '外观',
            options: [
                { value: 'solid', label: '实线' },
                { value: 'dashed', label: '虚线' },
                { value: 'dotted', label: '点线' },
                { value: 'none', label: '无边框' },
            ],
            default: 'dashed',
        },
        borderColor: {
            type: 'color',
            label: '边框颜色',
            group: '外观',
            default: '#909399',
        },
        borderRadius: {
            type: 'number',
            label: '圆角',
            group: '外观',
            min: 0,
            max: 100,
            default: 0,
            unit: 'px',
        },
    },

    eventsSchema: {
        click: { label: '点击', description: '鼠标点击时触发' },
    },

    container: true,
    gridCol: true, // 标记为栅格列
    // Col 组件不可在画布中自由拖拽，只能在 Row 内部使用
    draggableInCanvas: false,
    version: '1.0.0',
};
