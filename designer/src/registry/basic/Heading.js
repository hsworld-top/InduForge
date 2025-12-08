/**
 * Heading - 标题组件
 *
 * 基础标题组件，支持 H1-H6 级别
 * 分类：基础组件
 */

export default {
    type: 'Heading',
    name: '标题',
    category: '基础组件',
    icon: 'document-text',
    thumbnail: null,
    tags: ['标题', 'heading', 'h1', 'h2', 'h3', 'title'],
    description: '标题组件，支持 H1-H6 级别',

    defaultProps: {
        content: '标题内容',
        level: 1,
        fontFamily: 'Arial, sans-serif',
        fontWeight: 'bold',
        color: '#303133',
        textAlign: 'left',
        lineHeight: 1.2,
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
        content: {
            type: 'text',
            label: '标题内容',
            group: '内容',
            default: '标题内容',
        },
        level: {
            type: 'select',
            label: '标题级别',
            group: '内容',
            options: [
                { label: 'H1', value: 1 },
                { label: 'H2', value: 2 },
                { label: 'H3', value: 3 },
                { label: 'H4', value: 4 },
                { label: 'H5', value: 5 },
                { label: 'H6', value: 6 },
            ],
            default: 1,
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
            default: 'bold',
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
            default: 1.2,
        },
    },

    eventsSchema: {
        click: { label: '点击' },
    },

    container: false,
    version: '1.0.0',
};
