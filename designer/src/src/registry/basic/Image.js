/**
 * Image - 图片组件
 *
 * 基础图片组件，支持本地图片和网络图片
 * 分类：基础组件
 */

import ImageComponent from './Image.vue';

export default {
    type: 'Image',
    name: '图片',
    category: '基础组件',
    icon: 'image',
    thumbnail: null,
    tags: ['图片', 'image', '图像'],
    description: '图片组件，支持本地图片和网络图片',
    component: ImageComponent,

    defaultProps: {
        src: 'https://via.placeholder.com/150',
        alt: '图片',
        fit: 'contain',
        opacity: 1,
    },

    defaultStyle: {
        position: 'absolute',
        left: 100,
        top: 100,
        width: 150,
        height: 150,
        zIndex: 1,
    },

    propsSchema: {
        src: {
            type: 'image',
            label: '图片地址',
            group: '内容',
            default: 'https://via.placeholder.com/150',
        },
        alt: {
            type: 'text',
            label: '替代文本',
            group: '内容',
            default: '图片',
        },
        fit: {
            type: 'select',
            label: '填充方式',
            group: '外观',
            options: [
                { label: '包含', value: 'contain' },
                { label: '覆盖', value: 'cover' },
                { label: '填充', value: 'fill' },
                { label: '原始', value: 'none' },
                { label: '缩小', value: 'scale-down' },
            ],
            default: 'contain',
        },
        opacity: {
            type: 'number',
            label: '不透明度',
            group: '外观',
            min: 0,
            max: 1,
            step: 0.1,
            default: 1,
        },
    },

    eventsSchema: {
        click: { label: '点击' },
        load: { label: '加载完成' },
        error: { label: '加载失败' },
    },

    container: false,
    version: '1.0.0',
};
