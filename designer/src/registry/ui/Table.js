/**
 * Table - 表格组件
 *
 * Element Plus 表格组件
 * 分类：Element 组件
 */

export default {
    type: 'Table',
    name: '表格',
    category: 'Element 组件',
    icon: 'grid',
    thumbnail: null,
    tags: ['表格', 'table', 'grid', 'UI', 'element', '数据'],
    description: 'Element Plus 表格组件，用于展示结构化数据',

    defaultProps: {
        data: [
            { id: 1, name: '张三', age: 18, address: '北京' },
            { id: 2, name: '李四', age: 20, address: '上海' },
            { id: 3, name: '王五', age: 22, address: '广州' },
        ],
        columns: [
            { prop: 'id', label: 'ID', width: 80 },
            { prop: 'name', label: '姓名', width: 120 },
            { prop: 'age', label: '年龄', width: 80 },
            { prop: 'address', label: '地址', width: 200 },
        ],
        stripe: false,
        border: false,
        size: 'default',
        showHeader: true,
        highlightCurrentRow: false,
    },

    defaultStyle: {
        position: 'absolute',
        left: 100,
        top: 100,
        width: 500,
        height: 300,
        zIndex: 1,
    },

    propsSchema: {
        data: {
            type: 'array',
            label: '表格数据',
            group: '数据',
            default: [],
        },
        columns: {
            type: 'array',
            label: '列配置',
            group: '数据',
            default: [],
        },
        stripe: {
            type: 'boolean',
            label: '斑马纹',
            group: '外观',
            default: false,
        },
        border: {
            type: 'boolean',
            label: '边框',
            group: '外观',
            default: false,
        },
        size: {
            type: 'enum',
            label: '尺寸',
            group: '外观',
            options: [
                { value: 'large', label: '大' },
                { value: 'default', label: '默认' },
                { value: 'small', label: '小' },
            ],
            default: 'default',
        },
        showHeader: {
            type: 'boolean',
            label: '显示表头',
            group: '外观',
            default: true,
        },
        highlightCurrentRow: {
            type: 'boolean',
            label: '高亮当前行',
            group: '功能',
            default: false,
        },
    },

    eventsSchema: {
        rowClick: { label: '行点击', description: '点击行时触发' },
        rowDblclick: { label: '行双击', description: '双击行时触发' },
        selectionChange: { label: '选择改变', description: '选择项改变时触发' },
    },

    container: false,
    version: '1.0.0',
};
