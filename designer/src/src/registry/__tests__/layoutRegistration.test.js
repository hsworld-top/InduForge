/**
 * Layout Component Registration Tests
 *
 * Task 2.3: 重构布局组件注册（Container, Row/Col, Flex, Grid, CenterLayout）
 *
 * Tests:
 * - All layout components are properly registered
 * - Component definitions have required fields
 * - Components can be retrieved by type
 * - Components can be instantiated
 */

import { describe, it, expect, beforeEach } from 'vitest';
import componentFactory from '../ComponentFactory.js';
import layoutComponents from '../layout/index.js';
import { registerAllComponentsSync } from '../index.js';

describe('Layout Component Registration', () => {
    beforeEach(() => {
        // Clear the factory before each test
        componentFactory.clear();
    });

    it('should export 6 layout components', () => {
        expect(layoutComponents).toHaveLength(6);
    });

    it('should have all required layout component types', () => {
        const types = layoutComponents.map((comp) => comp.type);

        expect(types).toContain('Container');
        expect(types).toContain('Row');
        expect(types).toContain('Col');
        expect(types).toContain('FlexLayout');
        expect(types).toContain('Grid');
        expect(types).toContain('CenterLayout');
    });

    it('should register all layout components successfully', () => {
        registerAllComponentsSync(layoutComponents);

        expect(componentFactory.getCount()).toBe(6);
    });

    it('should retrieve Container component definition', () => {
        registerAllComponentsSync(layoutComponents);

        const container = componentFactory.get('Container');

        expect(container).toBeDefined();
        expect(container.type).toBe('Container');
        expect(container.name).toBe('盒子容器');
        expect(container.category).toBe('布局组件');
        expect(container.container).toBe(true);
    });

    it('should retrieve Row component definition', () => {
        registerAllComponentsSync(layoutComponents);

        const row = componentFactory.get('Row');

        expect(row).toBeDefined();
        expect(row.type).toBe('Row');
        expect(row.name).toBe('行列容器');
        expect(row.category).toBe('布局组件');
        expect(row.container).toBe(true);
        expect(row.gridRow).toBe(true);
    });

    it('should retrieve Col component definition', () => {
        registerAllComponentsSync(layoutComponents);

        const col = componentFactory.get('Col');

        expect(col).toBeDefined();
        expect(col.type).toBe('Col');
        expect(col.name).toBe('栅格布局');
        expect(col.category).toBe('布局组件');
        expect(col.container).toBe(true);
        expect(col.gridCol).toBe(true);
    });

    it('should retrieve FlexLayout component definition', () => {
        registerAllComponentsSync(layoutComponents);

        const flex = componentFactory.get('FlexLayout');

        expect(flex).toBeDefined();
        expect(flex.type).toBe('FlexLayout');
        expect(flex.name).toBe('弹性容器');
        expect(flex.category).toBe('布局组件');
        expect(flex.container).toBe(true);
    });

    it('should retrieve Grid component definition', () => {
        registerAllComponentsSync(layoutComponents);

        const grid = componentFactory.get('Grid');

        expect(grid).toBeDefined();
        expect(grid.type).toBe('Grid');
        expect(grid.name).toBe('栅格布局');
        expect(grid.category).toBe('布局组件');
        expect(grid.container).toBe(true);
    });

    it('should retrieve CenterLayout component definition', () => {
        registerAllComponentsSync(layoutComponents);

        const center = componentFactory.get('CenterLayout');

        expect(center).toBeDefined();
        expect(center.type).toBe('CenterLayout');
        expect(center.name).toBe('全宽居中');
        expect(center.category).toBe('布局组件');
        expect(center.container).toBe(true);
        expect(center.centerLayout).toBe(true);
    });

    it('should have Vue component references', () => {
        registerAllComponentsSync(layoutComponents);

        const container = componentFactory.get('Container');
        const row = componentFactory.get('Row');
        const col = componentFactory.get('Col');
        const flex = componentFactory.get('FlexLayout');
        const grid = componentFactory.get('Grid');
        const center = componentFactory.get('CenterLayout');

        expect(container.component).toBeDefined();
        expect(row.component).toBeDefined();
        expect(col.component).toBeDefined();
        expect(flex.component).toBeDefined();
        expect(grid.component).toBeDefined();
        expect(center.component).toBeDefined();
    });

    it('should have default props for all components', () => {
        registerAllComponentsSync(layoutComponents);

        const types = ['Container', 'Row', 'Col', 'FlexLayout', 'Grid', 'CenterLayout'];

        types.forEach((type) => {
            const component = componentFactory.get(type);
            expect(component.defaultProps).toBeDefined();
            expect(typeof component.defaultProps).toBe('object');
        });
    });

    it('should have default style for all components', () => {
        registerAllComponentsSync(layoutComponents);

        const types = ['Container', 'Row', 'Col', 'FlexLayout', 'Grid', 'CenterLayout'];

        types.forEach((type) => {
            const component = componentFactory.get(type);
            expect(component.defaultStyle).toBeDefined();
            expect(typeof component.defaultStyle).toBe('object');
        });
    });

    it('should have props schema for all components', () => {
        registerAllComponentsSync(layoutComponents);

        const types = ['Container', 'Row', 'Col', 'FlexLayout', 'Grid', 'CenterLayout'];

        types.forEach((type) => {
            const component = componentFactory.get(type);
            expect(component.propsSchema).toBeDefined();
            expect(typeof component.propsSchema).toBe('object');
        });
    });

    it('should create Container instance with default props', () => {
        registerAllComponentsSync(layoutComponents);

        const instance = componentFactory.createInstance('Container');

        expect(instance).toBeDefined();
        expect(instance.id).toBeDefined();
        expect(instance.type).toBe('Container');
        expect(instance.props).toBeDefined();
        expect(instance.props.layout).toBe('flex');
        expect(instance.style).toBeDefined();
        expect(instance.children).toEqual([]);
    });

    it('should create Row instance with custom props', () => {
        registerAllComponentsSync(layoutComponents);

        const instance = componentFactory.createInstance('Row', {
            props: { gutter: 20 },
            style: { left: 200, top: 200 },
        });

        expect(instance).toBeDefined();
        expect(instance.type).toBe('Row');
        expect(instance.props.gutter).toBe(20);
        expect(instance.style.left).toBe(200);
        expect(instance.style.top).toBe(200);
    });

    it('should create Grid instance with grid properties', () => {
        registerAllComponentsSync(layoutComponents);

        const instance = componentFactory.createInstance('Grid', {
            props: {
                gridTemplateColumns: 'repeat(4, 1fr)',
                gap: 20,
            },
        });

        expect(instance).toBeDefined();
        expect(instance.type).toBe('Grid');
        expect(instance.props.gridTemplateColumns).toBe('repeat(4, 1fr)');
        expect(instance.props.gap).toBe(20);
    });

    it('should get all layout components by category', () => {
        registerAllComponentsSync(layoutComponents);

        const byCategory = componentFactory.getAllByCategory();

        expect(byCategory['布局组件']).toBeDefined();
        expect(byCategory['布局组件']).toHaveLength(6);
    });

    it('should search layout components by keyword', () => {
        registerAllComponentsSync(layoutComponents);

        const results = componentFactory.search('容器');

        expect(results.length).toBeGreaterThan(0);
        expect(results.some((comp) => comp.type === 'Container')).toBe(true);
    });

    it('should have Container with Flex/Grid/Block layout modes', () => {
        registerAllComponentsSync(layoutComponents);

        const container = componentFactory.get('Container');
        const layoutSchema = container.propsSchema.layout;

        expect(layoutSchema).toBeDefined();
        expect(layoutSchema.type).toBe('enum');
        expect(layoutSchema.options).toHaveLength(3);

        const layoutValues = layoutSchema.options.map((opt) => opt.value);
        expect(layoutValues).toContain('flex');
        expect(layoutValues).toContain('grid');
        expect(layoutValues).toContain('block');
    });

    it('should have Row with 24-grid system support', () => {
        registerAllComponentsSync(layoutComponents);

        const col = componentFactory.get('Col');
        const spanSchema = col.propsSchema.span;

        expect(spanSchema).toBeDefined();
        expect(spanSchema.type).toBe('number');
        expect(spanSchema.min).toBe(1);
        expect(spanSchema.max).toBe(24);
        expect(spanSchema.default).toBe(12);
    });

    it('should have Grid with CSS Grid properties', () => {
        registerAllComponentsSync(layoutComponents);

        const grid = componentFactory.get('Grid');

        expect(grid.propsSchema.gridTemplateColumns).toBeDefined();
        expect(grid.propsSchema.gridTemplateRows).toBeDefined();
        expect(grid.propsSchema.gridAutoFlow).toBeDefined();
        expect(grid.propsSchema.justifyItems).toBeDefined();
        expect(grid.propsSchema.alignItems).toBeDefined();
    });

    it('should have FlexLayout with Flexbox properties', () => {
        registerAllComponentsSync(layoutComponents);

        const flex = componentFactory.get('FlexLayout');

        expect(flex.propsSchema.flexDirection).toBeDefined();
        expect(flex.propsSchema.justifyContent).toBeDefined();
        expect(flex.propsSchema.alignItems).toBeDefined();
        expect(flex.propsSchema.flexWrap).toBeDefined();
    });

    it('should mark CenterLayout with centerLayout flag', () => {
        registerAllComponentsSync(layoutComponents);

        const center = componentFactory.get('CenterLayout');

        expect(center.centerLayout).toBe(true);
    });

    it('should mark all layout components as containers', () => {
        registerAllComponentsSync(layoutComponents);

        const types = ['Container', 'Row', 'Col', 'FlexLayout', 'Grid', 'CenterLayout'];

        types.forEach((type) => {
            const component = componentFactory.get(type);
            expect(component.container).toBe(true);
        });
    });
});
