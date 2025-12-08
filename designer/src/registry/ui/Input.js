/**
 * Input - 输入框组件
 *
 * Element Plus 输入框组件
 * 分类：Element 组件
 */

export default {
    type: 'Input',
    name: '输入框',
    category: 'Element 组件',
    icon: 'create',
    thumbnail: null,
    tags: ['输入框', 'input', 'UI', 'element', '表单'],
    description: 'Element Plus 输入框组件，支持多种类型和验证',

    defaultProps: {
        value: '',
        placeholder: '请输入内容',
        type: 'text',
        size: 'default',
        disabled: false,
        clearable: false,
        showPassword: false,
        maxlength: null,
        showWordLimit: false,
        prefixIcon: '',
        suffixIcon: '',
    },

    defaultStyle: {
        position: 'absolute',
        left: 100,
        top: 100,
        width: 200,
        height: 32,
        zIndex: 1,
    },

    propsSchema: {
        value: {
            type: 'string',
            label: '输入值',
            group: '内容',
            default: '',
        },
        placeholder: {
            type: 'string',
            label: '占位文本',
            group: '内容',
            default: '请输入内容',
        },
        type: {
            type: 'enum',
            label: '类型',
            group: '外观',
            options: [
                { value: 'text', label: '文本' },
                { value: 'textarea', label: '文本域' },
                { value: 'password', label: '密码' },
                { value: 'number', label: '数字' },
            ],
            default: 'text',
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
            default: false,
        },
        showPassword: {
            type: 'boolean',
            label: '显示密码',
            group: '功能',
            default: false,
            visible: (props) => props.type === 'password',
        },
        maxlength: {
            type: 'number',
            label: '最大长度',
            group: '验证',
            default: null,
            min: 0,
        },
        showWordLimit: {
            type: 'boolean',
            label: '显示字数统计',
            group: '功能',
            default: false,
        },
        prefixIcon: {
            type: 'string',
            label: '前缀图标',
            group: '外观',
            default: '',
        },
        suffixIcon: {
            type: 'string',
            label: '后缀图标',
            group: '外观',
            default: '',
        },
    },

    eventsSchema: {
        input: { label: '输入', description: '输入值改变时触发' },
        change: { label: '改变', description: '输入值改变且失去焦点时触发' },
        focus: { label: '获得焦点' },
        blur: { label: '失去焦点' },
        clear: { label: '清空' },
    },

    container: false,
    version: '1.0.0',
};
