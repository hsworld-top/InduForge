/**
 * Row - 行布局组件（栅格系统）
 *
 * 栅格行容器，宽度 100%，配合 Col 组件使用
 * 分类：布局组件
 *
 * Task 2.3: 重构布局组件注册
 */

import RowComponent from './Row.vue';

export default {
    type: 'Row',
    component: RowComponent,
    name: '行列容器',
    category: '布局组件',
    icon: 'menu',
    thumbnail: null,
    tags: ['布局', '行', 'row', 'grid', '栅格', '水平'],
    description: '栅格行容器，宽度 100%，配合列组件使用，支持 24 栅格系统',

    defaultProps: {
        gutter: 0,
        gutterVertical: 0,
        justify: 'start',
        align: 'top',
        wrap: true,
        backgroundColor: 'transparent',
        borderWidth: 1,
        borderStyle: 'dashed',
        borderColor: '#F56C6C',
        borderRadius: 0,
        padding: 0,
    },

    defaultStyle: {
        position: 'absolute',
        left: 0,
        top: 100,
        width: '100%',
        height: 'auto',
        minHeight: 60,
        zIndex: 1,
    },

    propsSchema: {
        gutter: {
            type: 'number',
            label: '水平间距',
            group: '布局',
            description: '栅格列之间的水平间距（单位：px）',
            default: 0,
            min: 0,
            max: 100,
            unit: 'px',
        },
        gutterVertical: {
            type: 'number',
            label: '垂直间距',
            group: '布局',
            description: '栅格列之间的垂直间距（单位：px）',
            default: 0,
            min: 0,
            max: 100,
            unit: 'px',
        },
        justify: {
            type: 'enum',
            label: '水平排列',
            group: '布局',
            options: [
                { value: 'start', label: '左对齐' },
                { value: 'end', label: '右对齐' },
                { value: 'center', label: '居中' },
                { value: 'space-around', label: '均匀分布' },
                { value: 'space-between', label: '两端对齐' },
                { value: 'space-evenly', label: '等间距' },
            ],
            default: 'start',
        },
        align: {
            type: 'enum',
            label: '垂直对齐',
            group: '布局',
            options: [
                { value: 'top', label: '顶部' },
                { value: 'middle', label: '居中' },
                { value: 'bottom', label: '底部' },
                { value: 'stretch', label: '拉伸' },
            ],
            default: 'top',
        },
        wrap: {
            type: 'boolean',
            label: '自动换行',
            group: '布局',
            default: true,
        },
        padding: {
            type: 'number',
            label: '内边距',
            group: '布局',
            default: 0,
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
            default: '#F56C6C',
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
    gridRow: true, // 标记为栅格行
    version: '1.0.0',
};
