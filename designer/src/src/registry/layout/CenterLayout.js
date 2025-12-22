/**
 * CenterLayout - 全局居中布局组件
 *
 * 弹性布局的预设，子组件自动水平垂直居中
 * 分类：布局组件
 *
 * Task 2.3: 重构布局组件注册
 */

import CenterLayoutComponent from './CenterLayout.vue';

export default {
    type: 'CenterLayout',
    component: CenterLayoutComponent,
    name: '全宽居中',
    category: '布局组件',
    icon: 'apps',
    thumbnail: null,
    tags: ['布局', '居中', 'center', 'flex'],
    description: '居中布局容器，子组件自动水平垂直居中',

    defaultProps: {
        gap: 10,
        padding: 20,
        backgroundColor: 'transparent',
        borderWidth: 1,
        borderStyle: 'dashed',
        borderColor: '#67C23A',
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
            default: 20,
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
            default: '#67C23A',
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
    // 标记为居中布局，渲染时自动应用居中样式
    centerLayout: true,
    version: '1.0.0',
};
