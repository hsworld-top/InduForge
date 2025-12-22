/**
 * Registry Refactor Tests
 *
 * 测试重构后的组件库功能
 */

import { describe, it, expect, beforeEach } from 'vitest';
import { registerComponent, getComponent, getAllComponents, getComponentsByCategory, getAllCategories, searchComponents, createComponentInstance, clearRegistry } from '../index';

// 导入组件注册函数
import { registerLayoutComponents, registerBasicComponents, registerUIComponents, registerChartComponents, registerAllComponents } from '../components';

describe('Registry Refactor', () => {
    beforeEach(() => {
        clearRegistry();
    });

    describe('Component Registration', () => {
        it('should register layout components', () => {
            registerLayoutComponents();
            const components = getAllComponents();
            expect(components.length).toBeGreaterThan(0);

            const container = getComponent('Container');
            expect(container).toBeDefined();
            expect(container.category).toBe('布局组件');
        });

        it('should register basic components', () => {
            registerBasicComponents();
            const components = getAllComponents();
            expect(components.length).toBeGreaterThan(0);

            const rectangle = getComponent('Rectangle');
            expect(rectangle).toBeDefined();
            expect(rectangle.category).toBe('基础组件');
        });

        it('should register UI components', () => {
            registerUIComponents();
            const components = getAllComponents();
            expect(components.length).toBeGreaterThan(0);

            const button = getComponent('Button');
            expect(button).toBeDefined();
            expect(button.category).toBe('Element 组件');
        });

        it('should register chart components', () => {
            registerChartComponents();
            const components = getAllComponents();
            expect(components.length).toBeGreaterThan(0);

            const lineChart = getComponent('LineChart');
            expect(lineChart).toBeDefined();
            expect(lineChart.category).toBe('图表组件');
        });

        it('should register all components', () => {
            registerAllComponents();
            const components = getAllComponents();

            // 应该有 28 个组件
            // 布局: 6, 基础: 7, Element: 10, 图表: 5
            expect(components.length).toBe(28);
        });
    });

    describe('Component Categories', () => {
        beforeEach(() => {
            registerAllComponents();
        });

        it('should have correct categories', () => {
            const categories = getAllCategories();

            expect(categories).toContain('布局组件');
            expect(categories).toContain('基础组件');
            expect(categories).toContain('Element 组件');
            expect(categories).toContain('图表组件');
            expect(categories.length).toBe(4);
        });

        it('should group components by category', () => {
            const byCategory = getComponentsByCategory();

            expect(byCategory['布局组件']).toBeDefined();
            expect(byCategory['布局组件'].length).toBe(6);

            expect(byCategory['基础组件']).toBeDefined();
            expect(byCategory['基础组件'].length).toBe(7);

            expect(byCategory['Element 组件']).toBeDefined();
            expect(byCategory['Element 组件'].length).toBe(10);

            expect(byCategory['图表组件']).toBeDefined();
            expect(byCategory['图表组件'].length).toBe(5);
        });
    });

    describe('Component Search', () => {
        beforeEach(() => {
            registerAllComponents();
        });

        it('should search by name', () => {
            const results = searchComponents('按钮');
            expect(results.length).toBeGreaterThan(0);
            expect(results[0].type).toBe('Button');
        });

        it('should search by tag', () => {
            const results = searchComponents('图表');
            expect(results.length).toBeGreaterThan(0);
            expect(results.every((c) => c.category === '图表组件')).toBe(true);
        });

        it('should search by type', () => {
            const results = searchComponents('LineChart');
            expect(results.length).toBe(1);
            expect(results[0].type).toBe('LineChart');
        });
    });

    describe('Component Instance Creation', () => {
        beforeEach(() => {
            registerAllComponents();
        });

        it('should create layout component instance', () => {
            const instance = createComponentInstance('Container');

            expect(instance).toBeDefined();
            expect(instance.type).toBe('Container');
            expect(instance.id).toBeDefined();
            expect(instance.props).toBeDefined();
            expect(instance.style).toBeDefined();
        });

        it('should create basic component instance', () => {
            const instance = createComponentInstance('Rectangle');

            expect(instance).toBeDefined();
            expect(instance.type).toBe('Rectangle');
            expect(instance.props.fill).toBe('#409EFF');
        });

        it('should create UI component instance', () => {
            const instance = createComponentInstance('Button', {
                props: {
                    text: '确定',
                    type: 'primary',
                },
            });

            expect(instance).toBeDefined();
            expect(instance.type).toBe('Button');
            expect(instance.props.text).toBe('确定');
            expect(instance.props.type).toBe('primary');
        });

        it('should create chart component instance', () => {
            const instance = createComponentInstance('LineChart');

            expect(instance).toBeDefined();
            expect(instance.type).toBe('LineChart');
            expect(instance.props.title).toBe('折线图');
        });
    });

    describe('Component Definition Structure', () => {
        beforeEach(() => {
            registerAllComponents();
        });

        it('should have required fields', () => {
            const button = getComponent('Button');

            expect(button.type).toBeDefined();
            expect(button.name).toBeDefined();
            expect(button.category).toBeDefined();
            expect(button.icon).toBeDefined();
            expect(button.defaultProps).toBeDefined();
            expect(button.defaultStyle).toBeDefined();
            expect(button.propsSchema).toBeDefined();
            expect(button.eventsSchema).toBeDefined();
            expect(button.container).toBeDefined();
            expect(button.version).toBeDefined();
        });

        it('should have new fields', () => {
            const button = getComponent('Button');

            expect(button.thumbnail).toBeDefined();
            expect(button.tags).toBeDefined();
            expect(button.description).toBeDefined();
            expect(Array.isArray(button.tags)).toBe(true);
        });

        it('should have grouped props schema', () => {
            const button = getComponent('Button');

            const textProp = button.propsSchema.text;
            expect(textProp.group).toBeDefined();
            expect(textProp.label).toBeDefined();
            expect(textProp.default).toBeDefined();
        });
    });

    describe('Specific Components', () => {
        beforeEach(() => {
            registerAllComponents();
        });

        describe('Layout Components', () => {
            it('should have Container', () => {
                const container = getComponent('Container');
                expect(container).toBeDefined();
                expect(container.container).toBe(true);
            });

            it('should have Row', () => {
                const row = getComponent('Row');
                expect(row).toBeDefined();
                expect(row.container).toBe(true);
            });

            it('should have Col', () => {
                const col = getComponent('Col');
                expect(col).toBeDefined();
                expect(col.container).toBe(true);
            });

            it('should have FlexLayout', () => {
                const flexLayout = getComponent('FlexLayout');
                expect(flexLayout).toBeDefined();
                expect(flexLayout.container).toBe(true);
            });

            it('should have Grid', () => {
                const grid = getComponent('Grid');
                expect(grid).toBeDefined();
                expect(grid.container).toBe(true);
            });

            it('should have CenterLayout', () => {
                const centerLayout = getComponent('CenterLayout');
                expect(centerLayout).toBeDefined();
                expect(centerLayout.container).toBe(true);
            });
        });

        describe('Basic Components', () => {
            it('should have Rectangle', () => {
                const rectangle = getComponent('Rectangle');
                expect(rectangle).toBeDefined();
                expect(rectangle.defaultProps.fill).toBeDefined();
            });

            it('should have Circle', () => {
                const circle = getComponent('Circle');
                expect(circle).toBeDefined();
            });

            it('should have Ellipse', () => {
                const ellipse = getComponent('Ellipse');
                expect(ellipse).toBeDefined();
            });

            it('should have Line', () => {
                const line = getComponent('Line');
                expect(line).toBeDefined();
            });

            it('should have Polygon', () => {
                const polygon = getComponent('Polygon');
                expect(polygon).toBeDefined();
            });

            it('should have Text', () => {
                const text = getComponent('Text');
                expect(text).toBeDefined();
            });

            it('should have Image', () => {
                const image = getComponent('Image');
                expect(image).toBeDefined();
            });
        });

        describe('Element Components', () => {
            it('should have Button', () => {
                const button = getComponent('Button');
                expect(button).toBeDefined();
            });

            it('should have Input', () => {
                const input = getComponent('Input');
                expect(input).toBeDefined();
            });

            it('should have Select', () => {
                const select = getComponent('Select');
                expect(select).toBeDefined();
            });

            it('should have Switch', () => {
                const switchComp = getComponent('Switch');
                expect(switchComp).toBeDefined();
            });

            it('should have Slider', () => {
                const slider = getComponent('Slider');
                expect(slider).toBeDefined();
            });

            it('should have Progress', () => {
                const progress = getComponent('Progress');
                expect(progress).toBeDefined();
            });

            it('should have Table', () => {
                const table = getComponent('Table');
                expect(table).toBeDefined();
            });
        });

        describe('Chart Components', () => {
            it('should have LineChart', () => {
                const lineChart = getComponent('LineChart');
                expect(lineChart).toBeDefined();
            });

            it('should have BarChart', () => {
                const barChart = getComponent('BarChart');
                expect(barChart).toBeDefined();
            });

            it('should have PieChart', () => {
                const pieChart = getComponent('PieChart');
                expect(pieChart).toBeDefined();
            });

            it('should have GaugeChart', () => {
                const gaugeChart = getComponent('GaugeChart');
                expect(gaugeChart).toBeDefined();
            });

            it('should have ScatterChart', () => {
                const scatterChart = getComponent('ScatterChart');
                expect(scatterChart).toBeDefined();
            });
        });
    });
});
