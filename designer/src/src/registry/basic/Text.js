/**
 * Text - 文本组件
 *
 * 基础文本组件，支持字体、颜色、对齐等属性
 * 分类：基础组件
 */

export default {
    type: 'Text',
    name: '文本',
    category: '基础组件',
    icon: 'document-text',
    thumbnail: null,
    tags: ['文本', 'text', '标签', 'label'],
    description: '文本组件，支持字体、颜色、对齐等属性',

    defaultProps: {
        content: '文本内容',
        fontSize: 14,
        fontFamily: 'Arial, sans-serif',
        fontWeight: 'normal',
        fontStyle: 'normal',
        color: '#303133',
        textAlign: 'left',
        lineHeight: 1.5,
        opacity: 1,
    },

    defaultStyle: {
        position: 'absolute',
        left: 100,
        top: 100,
        width: 120,
        height: 30,
        zIndex: 1,
    },

    propsSchema: {
        content: {
            type: 'textarea',
            label: '文本内容',
            group: '内容',
            default: '文本内容',
        },
        fontSize: {
            type: 'number',
            label: '字体大小',
            group: '字体',
            min: 8,
            max: 72,
            step: 1,
            default: 14,
            unit: 'px',
        },
        fontFamily: {
            type: 'select',
            label: '字体',
            group: '字体',
            options: [
                { label: 'Arial', value: 'Arial, sans-serif' },
                { label: '微软雅黑', value: 'Microsoft YaHei, sans-serif' },
                { label: '宋体', value: 'SimSun, serif' },
                { label: '黑体', value: 'SimHei, sans-serif' },
                { label: 'Courier', value: 'Courier New, monospace' },
            ],
            default: 'Arial, sans-serif',
        },
        fontWeight: {
            type: 'select',
            label: '字重',
            group: '字体',
            options: [
                { label: '正常', value: 'normal' },
                { label: '粗体', value: 'bold' },
                { label: '细体', value: 'lighter' },
            ],
            default: 'normal',
        },
        fontStyle: {
            type: 'select',
            label: '样式',
            group: '字体',
            options: [
                { label: '正常', value: 'normal' },
                { label: '斜体', value: 'italic' },
            ],
            default: 'normal',
        },
        color: {
            type: 'color',
            label: '文字颜色',
            group: '外观',
            default: '#303133',
        },
        textAlign: {
            type: 'select',
            label: '对齐方式',
            group: '布局',
            options: [
                { label: '左对齐', value: 'left' },
                { label: '居中', value: 'center' },
                { label: '右对齐', value: 'right' },
            ],
            default: 'left',
        },
        lineHeight: {
            type: 'number',
            label: '行高',
            group: '布局',
            min: 1,
            max: 3,
            step: 0.1,
            default: 1.5,
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
        dblclick: { label: '双击' },
    },

    container: false,
    version: '1.0.0',
};
