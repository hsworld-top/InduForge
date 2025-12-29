/**
 * UI Component Registration Tests
 *
 * Task 2.3: 重构 UI 组件注册（ElButton, ElInput...）
 *
 * Tests:
 * - All UI components are properly registered
 * - Component definitions have required fields
 * - Components can be retrieved by type
 * - Components can be instantiated
 */

import { describe, it, expect, beforeEach } from 'vitest';
import componentFactory from '../ComponentFactory.js';
import uiComponents from '../ui/index.js';
import { registerAllComponentsSync } from '../index.js';

describe('UI Component Registration', () => {
    beforeEach(() => {
        // Clear the factory before each test
        componentFactory.clear();
    });

    it('should export 10 UI components', () => {
        expect(uiComponents).toHaveLength(10);
    });

    it('should have all required UI component types', () => {
        const types = uiComponents.map((comp) => comp.type);

        expect(types).toContain('Button');
        expect(types).toContain('Input');
        expect(types).toContain('Select');
        expect(types).toContain('Switch');
        expect(types).toContain('Slider');
        expect(types).toContain('Progress');
        expect(types).toContain('Table');
        expect(types).toContain('DatePicker');
        expect(types).toContain('Form');
        expect(types).toContain('Icon');
    });

    it('should register all UI components successfully', () => {
        registerAllComponentsSync([], [], uiComponents);

        expect(componentFactory.getCount()).toBe(10);
    });

    it('should retrieve Button component definition', () => {
        registerAllComponentsSync([], [], uiComponents);

        const button = componentFactory.get('Button');

        expect(button).toBeDefined();
        expect(button.type).toBe('Button');
        expect(button.name).toBe('按钮');
        expect(button.category).toBe('Element 组件');
        expect(button.container).toBe(false);
    });

    it('should retrieve Input component definition', () => {
        registerAllComponentsSync([], [], uiComponents);

        const input = componentFactory.get('Input');

        expect(input).toBeDefined();
        expect(input.type).toBe('Input');
        expect(input.name).toBe('输入框');
        expect(input.category).toBe('Element 组件');
        expect(input.container).toBe(false);
    });

    it('should retrieve Select component definition', () => {
        registerAllComponentsSync([], [], uiComponents);

        const select = componentFactory.get('Select');

        expect(select).toBeDefined();
        expect(select.type).toBe('Select');
        expect(select.name).toBe('选择器');
        expect(select.category).toBe('Element 组件');
    });

    it('should retrieve Switch component definition', () => {
        registerAllComponentsSync([], [], uiComponents);

        const switchComp = componentFactory.get('Switch');

        expect(switchComp).toBeDefined();
        expect(switchComp.type).toBe('Switch');
        expect(switchComp.name).toBe('开关');
        expect(switchComp.category).toBe('Element 组件');
    });

    it('should retrieve Slider component definition', () => {
        registerAllComponentsSync([], [], uiComponents);

        const slider = componentFactory.get('Slider');

        expect(slider).toBeDefined();
        expect(slider.type).toBe('Slider');
        expect(slider.name).toBe('滑块');
        expect(slider.category).toBe('Element 组件');
    });

    it('should retrieve Progress component definition', () => {
        registerAllComponentsSync([], [], uiComponents);

        const progress = componentFactory.get('Progress');

        expect(progress).toBeDefined();
        expect(progress.type).toBe('Progress');
        expect(progress.name).toBe('进度条');
        expect(progress.category).toBe('Element 组件');
    });

    it('should retrieve Table component definition', () => {
        registerAllComponentsSync([], [], uiComponents);

        const table = componentFactory.get('Table');

        expect(table).toBeDefined();
        expect(table.type).toBe('Table');
        expect(table.name).toBe('表格');
        expect(table.category).toBe('Element 组件');
    });

    it('should retrieve DatePicker component definition', () => {
        registerAllComponentsSync([], [], uiComponents);

        const datePicker = componentFactory.get('DatePicker');

        expect(datePicker).toBeDefined();
        expect(datePicker.type).toBe('DatePicker');
        expect(datePicker.name).toBe('日期选择器');
        expect(datePicker.category).toBe('Element 组件');
    });

    it('should retrieve Form component definition', () => {
        registerAllComponentsSync([], [], uiComponents);

        const form = componentFactory.get('Form');

        expect(form).toBeDefined();
        expect(form.type).toBe('Form');
        expect(form.name).toBe('表单');
        expect(form.category).toBe('Element 组件');
        expect(form.container).toBe(true);
    });

    it('should retrieve Icon component definition', () => {
        registerAllComponentsSync([], [], uiComponents);

        const icon = componentFactory.get('Icon');

        expect(icon).toBeDefined();
        expect(icon.type).toBe('Icon');
        expect(icon.name).toBe('图标');
        expect(icon.category).toBe('Element 组件');
    });

    it('should have default props for all components', () => {
        registerAllComponentsSync([], [], uiComponents);

        const types = ['Button', 'Input', 'Select', 'Switch', 'Slider', 'Progress', 'Table', 'DatePicker', 'Form', 'Icon'];

        types.forEach((type) => {
            const component = componentFactory.get(type);
            expect(component.defaultProps).toBeDefined();
            expect(typeof component.defaultProps).toBe('object');
        });
    });

    it('should have default style for all components', () => {
        registerAllComponentsSync([], [], uiComponents);

        const types = ['Button', 'Input', 'Select', 'Switch', 'Slider', 'Progress', 'Table', 'DatePicker', 'Form', 'Icon'];

        types.forEach((type) => {
            const component = componentFactory.get(type);
            expect(component.defaultStyle).toBeDefined();
            expect(typeof component.defaultStyle).toBe('object');
            expect(component.defaultStyle.position).toBe('absolute');
        });
    });

    it('should have props schema for all components', () => {
        registerAllComponentsSync([], [], uiComponents);

        const types = ['Button', 'Input', 'Select', 'Switch', 'Slider', 'Progress', 'Table', 'DatePicker', 'Form', 'Icon'];

        types.forEach((type) => {
            const component = componentFactory.get(type);
            expect(component.propsSchema).toBeDefined();
            expect(typeof component.propsSchema).toBe('object');
        });
    });

    it('should create Button instance with default props', () => {
        registerAllComponentsSync([], [], uiComponents);

        const instance = componentFactory.createInstance('Button');

        expect(instance).toBeDefined();
        expect(instance.id).toBeDefined();
        expect(instance.type).toBe('Button');
        expect(instance.props).toBeDefined();
        expect(instance.props.text).toBe('按钮');
        expect(instance.props.type).toBe('primary');
        expect(instance.style).toBeDefined();
        expect(instance.children).toEqual([]);
    });

    it('should create Input instance with custom props', () => {
        registerAllComponentsSync([], [], uiComponents);

        const instance = componentFactory.createInstance('Input', {
            props: { placeholder: '请输入用户名', clearable: true },
            style: { left: 200, top: 200, width: 300 },
        });

        expect(instance).toBeDefined();
        expect(instance.type).toBe('Input');
        expect(instance.props.placeholder).toBe('请输入用户名');
        expect(instance.props.clearable).toBe(true);
        expect(instance.style.left).toBe(200);
        expect(instance.style.width).toBe(300);
    });

    it('should create Table instance with data and columns', () => {
        registerAllComponentsSync([], [], uiComponents);

        const instance = componentFactory.createInstance('Table', {
            props: {
                stripe: true,
                border: true,
            },
        });

        expect(instance).toBeDefined();
        expect(instance.type).toBe('Table');
        expect(instance.props.stripe).toBe(true);
        expect(instance.props.border).toBe(true);
        expect(instance.props.data).toBeDefined();
        expect(instance.props.columns).toBeDefined();
    });

    it('should get all UI components by category', () => {
        registerAllComponentsSync([], [], uiComponents);

        const byCategory = componentFactory.getAllByCategory();

        expect(byCategory['Element 组件']).toBeDefined();
        expect(byCategory['Element 组件']).toHaveLength(10);
    });

    it('should search UI components by keyword', () => {
        registerAllComponentsSync([], [], uiComponents);

        const results = componentFactory.search('按钮');

        expect(results.length).toBeGreaterThan(0);
        expect(results.some((comp) => comp.type === 'Button')).toBe(true);
    });

    it('should have Button with type options', () => {
        registerAllComponentsSync([], [], uiComponents);

        const button = componentFactory.get('Button');
        const typeSchema = button.propsSchema.type;

        expect(typeSchema).toBeDefined();
        expect(typeSchema.type).toBe('enum');
        expect(typeSchema.options).toHaveLength(6);

        const typeValues = typeSchema.options.map((opt) => opt.value);
        expect(typeValues).toContain('primary');
        expect(typeValues).toContain('success');
        expect(typeValues).toContain('warning');
        expect(typeValues).toContain('danger');
    });

    it('should have Input with type options', () => {
        registerAllComponentsSync([], [], uiComponents);

        const input = componentFactory.get('Input');
        const typeSchema = input.propsSchema.type;

        expect(typeSchema).toBeDefined();
        expect(typeSchema.type).toBe('enum');

        const typeValues = typeSchema.options.map((opt) => opt.value);
        expect(typeValues).toContain('text');
        expect(typeValues).toContain('textarea');
        expect(typeValues).toContain('password');
        expect(typeValues).toContain('number');
    });

    it('should have Select with options array', () => {
        registerAllComponentsSync([], [], uiComponents);

        const select = componentFactory.get('Select');

        expect(select.propsSchema.options).toBeDefined();
        expect(select.propsSchema.options.type).toBe('array');
        expect(select.defaultProps.options).toBeDefined();
        expect(Array.isArray(select.defaultProps.options)).toBe(true);
    });

    it('should have Slider with min/max/step properties', () => {
        registerAllComponentsSync([], [], uiComponents);

        const slider = componentFactory.get('Slider');

        expect(slider.propsSchema.min).toBeDefined();
        expect(slider.propsSchema.max).toBeDefined();
        expect(slider.propsSchema.step).toBeDefined();
        expect(slider.defaultProps.min).toBe(0);
        expect(slider.defaultProps.max).toBe(100);
        expect(slider.defaultProps.step).toBe(1);
    });

    it('should have Progress with percentage property', () => {
        registerAllComponentsSync([], [], uiComponents);

        const progress = componentFactory.get('Progress');
        const percentageSchema = progress.propsSchema.percentage;

        expect(percentageSchema).toBeDefined();
        expect(percentageSchema.type).toBe('number');
        expect(percentageSchema.min).toBe(0);
        expect(percentageSchema.max).toBe(100);
    });

    it('should have DatePicker with type options', () => {
        registerAllComponentsSync([], [], uiComponents);

        const datePicker = componentFactory.get('DatePicker');
        const typeSchema = datePicker.propsSchema.type;

        expect(typeSchema).toBeDefined();
        expect(typeSchema.type).toBe('enum');

        const typeValues = typeSchema.options.map((opt) => opt.value);
        expect(typeValues).toContain('date');
        expect(typeValues).toContain('daterange');
        expect(typeValues).toContain('datetime');
    });

    it('should have Form with label position options', () => {
        registerAllComponentsSync([], [], uiComponents);

        const form = componentFactory.get('Form');
        const labelPositionSchema = form.propsSchema.labelPosition;

        expect(labelPositionSchema).toBeDefined();
        expect(labelPositionSchema.type).toBe('enum');

        const positionValues = labelPositionSchema.options.map((opt) => opt.value);
        expect(positionValues).toContain('left');
        expect(positionValues).toContain('right');
        expect(positionValues).toContain('top');
    });

    it('should mark Form as container', () => {
        registerAllComponentsSync([], [], uiComponents);

        const form = componentFactory.get('Form');

        expect(form.container).toBe(true);
    });

    it('should mark non-container components correctly', () => {
        registerAllComponentsSync([], [], uiComponents);

        const nonContainerTypes = ['Button', 'Input', 'Select', 'Switch', 'Slider', 'Progress', 'Table', 'DatePicker', 'Icon'];

        nonContainerTypes.forEach((type) => {
            const component = componentFactory.get(type);
            expect(component.container).toBe(false);
        });
    });

    it('should have events schema for interactive components', () => {
        registerAllComponentsSync([], [], uiComponents);

        const button = componentFactory.get('Button');
        const input = componentFactory.get('Input');
        const select = componentFactory.get('Select');

        expect(button.eventsSchema).toBeDefined();
        expect(button.eventsSchema.click).toBeDefined();

        expect(input.eventsSchema).toBeDefined();
        expect(input.eventsSchema.input).toBeDefined();
        expect(input.eventsSchema.change).toBeDefined();

        expect(select.eventsSchema).toBeDefined();
        expect(select.eventsSchema.change).toBeDefined();
    });

    it('should have size options for all form components', () => {
        registerAllComponentsSync([], [], uiComponents);

        const formComponents = ['Button', 'Input', 'Select', 'DatePicker', 'Form'];

        formComponents.forEach((type) => {
            const component = componentFactory.get(type);
            const sizeSchema = component.propsSchema.size;

            expect(sizeSchema).toBeDefined();
            expect(sizeSchema.type).toBe('enum');

            const sizeValues = sizeSchema.options.map((opt) => opt.value);
            expect(sizeValues).toContain('large');
            expect(sizeValues).toContain('default');
            expect(sizeValues).toContain('small');
        });
    });

    it('should have tags for all components', () => {
        registerAllComponentsSync([], [], uiComponents);

        const types = ['Button', 'Input', 'Select', 'Switch', 'Slider', 'Progress', 'Table', 'DatePicker', 'Form', 'Icon'];

        types.forEach((type) => {
            const component = componentFactory.get(type);
            expect(component.tags).toBeDefined();
            expect(Array.isArray(component.tags)).toBe(true);
            expect(component.tags.length).toBeGreaterThan(0);
            expect(component.tags).toContain('element');
        });
    });

    it('should have description for all components', () => {
        registerAllComponentsSync([], [], uiComponents);

        const types = ['Button', 'Input', 'Select', 'Switch', 'Slider', 'Progress', 'Table', 'DatePicker', 'Form', 'Icon'];

        types.forEach((type) => {
            const component = componentFactory.get(type);
            expect(component.description).toBeDefined();
            expect(typeof component.description).toBe('string');
            expect(component.description.length).toBeGreaterThan(0);
        });
    });
});
