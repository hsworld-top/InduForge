/**
 * Component Definition Template
 * Task 2.2: 定义组件定义规范
 *
 * 组件定义schema结构示例
 */

export const componentDefinitionSchema = {
    // 组件类型（唯一标识，必填）
    type: 'ComponentType',

    // 组件显示名称（必填）
    name: 'Component Name',

    // 组件分类（必填）
    // 可选值: 'layout', 'basic', 'form', 'data', 'chart', 'media', 'other'
    category: 'basic',

    // 组件图标（可选）
    icon: 'icon-name',

    // 组件描述（可选）
    description: 'Component description',

    // 默认属性（必填）
    defaultProps: {
        // 组件特定的属性
    },

    // 默认样式（必填）
    defaultStyle: {
        x: 0,
        y: 0,
        width: 100,
        height: 100,
        backgroundColor: '#ffffff',
        borderColor: '#cccccc',
        borderWidth: 1,
        borderRadius: 0,
        opacity: 1,
        rotation: 0,
        padding: 0,
        margin: 0,
    },

    // 属性schema定义（可选）
    propsSchema: {
        // 属性名: {
        //   type: 'string' | 'number' | 'boolean' | 'array' | 'object',
        //   label: '属性显示名称',
        //   default: 默认值,
        //   options: [], // 如果是枚举类型
        //   required: true | false,
        //   description: '属性描述',
        // }
    },

    // Vue组件（必填）
    component: null, // Vue component object or function
};

/**
 * 示例：Text组件定义
 */
export const textComponentExample = {
    type: 'Text',
    name: '文本',
    category: 'basic',
    icon: 'icon-text',
    description: '显示文本内容',

    defaultProps: {
        text: 'Text',
        fontSize: 14,
        fontWeight: 'normal',
        fontFamily: 'Arial',
        color: '#000000',
        textAlign: 'left',
        lineHeight: 1.5,
    },

    defaultStyle: {
        x: 0,
        y: 0,
        width: 200,
        height: 50,
        backgroundColor: 'transparent',
        borderColor: 'transparent',
        borderWidth: 0,
        borderRadius: 0,
        opacity: 1,
        rotation: 0,
        padding: 10,
        margin: 0,
    },

    propsSchema: {
        text: {
            type: 'string',
            label: '文本内容',
            default: 'Text',
            required: true,
            description: '要显示的文本内容',
        },
        fontSize: {
            type: 'number',
            label: '字体大小',
            default: 14,
            required: false,
            description: '文本字体大小（像素）',
        },
        fontWeight: {
            type: 'string',
            label: '字体粗细',
            default: 'normal',
            options: ['normal', 'bold', '100', '200', '300', '400', '500', '600', '700', '800', '900'],
            required: false,
            description: '文本字体粗细',
        },
        color: {
            type: 'string',
            label: '文本颜色',
            default: '#000000',
            required: false,
            description: '文本颜色（十六进制）',
        },
        textAlign: {
            type: 'string',
            label: '文本对齐',
            default: 'left',
            options: ['left', 'center', 'right', 'justify'],
            required: false,
            description: '文本水平对齐方式',
        },
    },

    component: null, // Will be set to actual Vue component
};

/**
 * 示例：Container组件定义
 */
export const containerComponentExample = {
    type: 'Container',
    name: '容器',
    category: 'layout',
    icon: 'icon-container',
    description: '布局容器，可包含子组件',

    defaultProps: {
        layoutMode: 'flex', // 'flex' | 'grid' | 'block'
        flexDirection: 'row',
        justifyContent: 'flex-start',
        alignItems: 'flex-start',
        gap: 10,
    },

    defaultStyle: {
        x: 0,
        y: 0,
        width: 400,
        height: 300,
        backgroundColor: '#f5f5f5',
        borderColor: '#cccccc',
        borderWidth: 1,
        borderRadius: 4,
        opacity: 1,
        rotation: 0,
        padding: 20,
        margin: 0,
    },

    propsSchema: {
        layoutMode: {
            type: 'string',
            label: '布局模式',
            default: 'flex',
            options: ['flex', 'grid', 'block'],
            required: true,
            description: '容器的布局模式',
        },
        flexDirection: {
            type: 'string',
            label: 'Flex方向',
            default: 'row',
            options: ['row', 'column', 'row-reverse', 'column-reverse'],
            required: false,
            description: 'Flexbox布局方向',
        },
        justifyContent: {
            type: 'string',
            label: '主轴对齐',
            default: 'flex-start',
            options: ['flex-start', 'center', 'flex-end', 'space-between', 'space-around', 'space-evenly'],
            required: false,
            description: 'Flexbox主轴对齐方式',
        },
        alignItems: {
            type: 'string',
            label: '交叉轴对齐',
            default: 'flex-start',
            options: ['flex-start', 'center', 'flex-end', 'stretch', 'baseline'],
            required: false,
            description: 'Flexbox交叉轴对齐方式',
        },
        gap: {
            type: 'number',
            label: '间距',
            default: 10,
            required: false,
            description: '子组件之间的间距（像素）',
        },
    },

    component: null, // Will be set to actual Vue component
};
