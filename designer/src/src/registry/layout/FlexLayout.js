/**
 * FlexLayout - 弹性布局组件
 *
 * 高级 Flex 布局容器，支持完整的 Flexbox 属性
 * 分类：布局组件
 *
 * Task 2.3: 重构布局组件注册
 */

import FlexComponent from './Flex.vue';

export default {
    type: 'FlexLayout',
    component: FlexComponent,
    name: '弹性容器',
    category: '布局组件',
    icon: 'grid',
    thumbnail: null,
    tags: ['布局', 'flex', '弹性', 'flexbox'],
    description: '弹性布局容器，支持完整的 Flexbox 属性配置',

    defaultProps: {
        flexDirection: 'row',
        justifyContent: 'start',
        alignItems: 'start',
        flexWrap: 'nowrap',
        gap: 10,
        padding: 16,
        backgroundColor: 'transparent',
        borderWidth: 1,
        borderStyle: 'dashed',
        borderColor: '#E6A23C',
        borderRadius: 4,
        boxShadow: 'none',
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
        flexDirection: {
            type: 'enum',
            label: '排列方向',
            group: '布局',
            options: [
                { value: 'row', label: '水平（从左到右）' },
                { value: 'row-reverse', label: '水平（从右到左）' },
                { value: 'column', label: '垂直（从上到下）' },
                { value: 'column-reverse', label: '垂直（从下到上）' },
            ],
            default: 'row',
        },
        justifyContent: {
            type: 'enum',
            label: '主轴对齐',
            group: '布局',
            options: [
                { value: 'start', label: '起始对齐' },
                { value: 'end', label: '结束对齐' },
                { value: 'center', label: '居中对齐' },
                { value: 'space-between', label: '两端对齐' },
                { value: 'space-around', label: '均匀分布（两侧留白）' },
                { value: 'space-evenly', label: '均匀分布（等间距）' },
            ],
            default: 'start',
        },
        alignItems: {
            type: 'enum',
            label: '交叉轴对齐',
            group: '布局',
            options: [
                { value: 'start', label: '起始对齐' },
                { value: 'end', label: '结束对齐' },
                { value: 'center', label: '居中对齐' },
                { value: 'stretch', label: '拉伸填充' },
                { value: 'baseline', label: '基线对齐' },
            ],
            default: 'start',
        },
        flexWrap: {
            type: 'enum',
            label: '换行方式',
            group: '布局',
            options: [
                { value: 'nowrap', label: '不换行' },
                { value: 'wrap', label: '换行' },
                { value: 'wrap-reverse', label: '反向换行' },
            ],
            default: 'nowrap',
        },
        gap: {
            type: 'number',
            label: '子元素间距',
            group: '布局',
            default: 10,
            min: 0,
            max: 100,
            unit: 'px',
        },
        padding: {
            type: 'number',
            label: '内边距',
            group: '布局',
            default: 16,
            min: 0,
            max: 100,
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
            default: '#E6A23C',
        },
        borderRadius: {
            type: 'number',
            label: '圆角',
            group: '外观',
            min: 0,
            max: 100,
            default: 4,
            unit: 'px',
        },
        boxShadow: {
            type: 'enum',
            label: '阴影',
            group: '外观',
            options: [
                { value: 'none', label: '无阴影' },
                { value: '0 2px 4px rgba(0,0,0,0.1)', label: '浅阴影' },
                { value: '0 4px 8px rgba(0,0,0,0.15)', label: '中阴影' },
                { value: '0 8px 16px rgba(0,0,0,0.2)', label: '深阴影' },
            ],
            default: 'none',
        },
    },

    eventsSchema: {
        click: { label: '点击', description: '鼠标点击时触发' },
    },

    container: true,
    version: '1.0.0',
};
