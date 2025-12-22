/**
 * Link - 链接组件
 *
 * 基础链接组件，支持跳转和样式配置
 * 分类：基础组件
 */

export default {
    type: 'Link',
    name: '链接',
    category: '基础组件',
    icon: 'link',
    thumbnail: null,
    tags: ['链接', 'link', '超链接', 'a'],
    description: '链接组件，支持跳转和样式配置',

    defaultProps: {
        text: '链接文本',
        href: '#',
        target: '_self',
        underline: true,
        color: '#409EFF',
        hoverColor: '#66b1ff',
        fontSize: 14,
        fontWeight: 'normal',
    },

    defaultStyle: {
        position: 'absolute',
        left: 100,
        top: 100,
        width: 80,
        height: 24,
        zIndex: 1,
    },

    propsSchema: {
        text: {
            type: 'text',
            label: '链接文本',
            group: '内容',
            default: '链接文本',
        },
        href: {
            type: 'text',
            label: '链接地址',
            group: '内容',
            default: '#',
        },
        target: {
            type: 'select',
            label: '打开方式',
            group: '行为',
            options: [
                { label: '当前窗口', value: '_self' },
                { label: '新窗口', value: '_blank' },
                { label: '父窗口', value: '_parent' },
                { label: '顶层窗口', value: '_top' },
            ],
            default: '_self',
        },
        underline: {
            type: 'boolean',
            label: '显示下划线',
            group: '外观',
            default: true,
        },
        color: {
            type: 'color',
            label: '链接颜色',
            group: '外观',
            default: '#409EFF',
        },
        hoverColor: {
            type: 'color',
            label: '悬停颜色',
            group: '外观',
            default: '#66b1ff',
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
    },

    eventsSchema: {
        click: { label: '点击' },
    },

    container: false,
    version: '1.0.0',
};
