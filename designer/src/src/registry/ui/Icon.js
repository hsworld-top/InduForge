/**
 * Icon - 图标组件
 *
 * Element Plus 图标组件
 * 分类：Element 组件
 */

import IconComponent from './Icon.vue';

export default {
    type: 'Icon',
    name: '图标',
    category: 'Element 组件',
    icon: 'star',
    thumbnail: null,
    tags: ['图标', 'icon', 'UI', 'element'],
    description: 'Element Plus 图标组件，支持 Element Plus Icons',
    component: IconComponent,

    defaultProps: {
        name: 'star',
        size: 16,
        color: '#606266',
    },

    defaultStyle: {
        position: 'absolute',
        left: 100,
        top: 100,
        width: 32,
        height: 32,
        zIndex: 1,
    },

    propsSchema: {
        name: {
            type: 'string',
            label: '图标名称',
            group: '内容',
            default: 'star',
            description: 'Element Plus Icons 图标名称',
        },
        size: {
            type: 'number',
            label: '图标大小',
            group: '外观',
            default: 16,
            min: 8,
            max: 128,
            unit: 'px',
        },
        color: {
            type: 'color',
            label: '图标颜色',
            group: '外观',
            default: '#606266',
        },
    },

    eventsSchema: {
        click: { label: '点击', description: '图标被点击时触发' },
    },

    container: false,
    version: '1.0.0',
};
