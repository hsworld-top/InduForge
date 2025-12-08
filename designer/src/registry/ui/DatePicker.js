/**
 * DatePicker - 日期选择器组件
 *
 * Element Plus 日期选择器组件
 * 分类：Element 组件
 */

export default {
    type: 'DatePicker',
    name: '日期选择器',
    category: 'Element 组件',
    icon: 'calendar',
    thumbnail: null,
    tags: ['日期', 'datepicker', 'calendar', 'UI', 'element', '表单'],
    description: 'Element Plus 日期选择器组件，支持日期、日期范围选择',

    defaultProps: {
        value: '',
        type: 'date',
        placeholder: '选择日期',
        size: 'default',
        disabled: false,
        clearable: true,
        format: 'YYYY-MM-DD',
        valueFormat: 'YYYY-MM-DD',
        rangeSeparator: '至',
        startPlaceholder: '开始日期',
        endPlaceholder: '结束日期',
    },

    defaultStyle: {
        position: 'absolute',
        left: 100,
        top: 100,
        width: 220,
        height: 32,
        zIndex: 1,
    },

    propsSchema: {
        value: {
            type: 'string',
            label: '选中值',
            group: '内容',
            default: '',
        },
        type: {
            type: 'enum',
            label: '类型',
            group: '外观',
            options: [
                { value: 'date', label: '日期' },
                { value: 'daterange', label: '日期范围' },
                { value: 'datetime', label: '日期时间' },
                { value: 'datetimerange', label: '日期时间范围' },
                { value: 'month', label: '月份' },
                { value: 'year', label: '年份' },
            ],
            default: 'date',
        },
        placeholder: {
            type: 'string',
            label: '占位文本',
            group: '内容',
            default: '选择日期',
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
        disabled: {
            type: 'boolean',
            label: '禁用',
            group: '状态',
            default: false,
        },
        clearable: {
            type: 'boolean',
            label: '可清空',
            group: '功能',
            default: true,
        },
        format: {
            type: 'string',
            label: '显示格式',
            group: '格式',
            default: 'YYYY-MM-DD',
        },
        valueFormat: {
            type: 'string',
            label: '值格式',
            group: '格式',
            default: 'YYYY-MM-DD',
        },
    },

    eventsSchema: {
        change: { label: '改变', description: '用户确认选定的值时触发' },
        blur: { label: '失去焦点' },
        focus: { label: '获得焦点' },
    },

    container: false,
    version: '1.0.0',
};
