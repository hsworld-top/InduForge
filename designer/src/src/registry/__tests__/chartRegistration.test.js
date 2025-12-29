/**
 * Chart Component Registration Tests
 *
 * Task 2.3: 重构图表组件注册（LineChart, BarChart...）
 *
 * Tests:
 * - All chart components are properly registered
 * - Component definitions have required fields
 * - Components can be retrieved by type
 * - Components can be instantiated
 */

import { describe, it, expect, beforeEach } from 'vitest';
import componentFactory from '../ComponentFactory.js';
import chartComponents from '../charts/index.js';
import { registerAllComponentsSync } from '../index.js';

describe('Chart Component Registration', () => {
    beforeEach(() => {
        // Clear the factory before each test
        componentFactory.clear();
    });

    it('should export 5 chart components', () => {
        expect(chartComponents).toHaveLength(5);
    });

    it('should have all required chart component types', () => {
        const types = chartComponents.map((comp) => comp.type);

        expect(types).toContain('LineChart');
        expect(types).toContain('BarChart');
        expect(types).toContain('PieChart');
        expect(types).toContain('GaugeChart');
        expect(types).toContain('ScatterChart');
    });

    it('should register all chart components successfully', () => {
        registerAllComponentsSync([], [], [], chartComponents);

        expect(componentFactory.getCount()).toBe(5);
    });

    it('should retrieve LineChart component definition', () => {
        registerAllComponentsSync([], [], [], chartComponents);

        const lineChart = componentFactory.get('LineChart');

        expect(lineChart).toBeDefined();
        expect(lineChart.type).toBe('LineChart');
        expect(lineChart.name).toBe('折线图');
        expect(lineChart.category).toBe('图表组件');
        expect(lineChart.container).toBe(false);
    });

    it('should retrieve BarChart component definition', () => {
        registerAllComponentsSync([], [], [], chartComponents);

        const barChart = componentFactory.get('BarChart');

        expect(barChart).toBeDefined();
        expect(barChart.type).toBe('BarChart');
        expect(barChart.name).toBe('柱状图');
        expect(barChart.category).toBe('图表组件');
        expect(barChart.container).toBe(false);
    });

    it('should retrieve PieChart component definition', () => {
        registerAllComponentsSync([], [], [], chartComponents);

        const pieChart = componentFactory.get('PieChart');

        expect(pieChart).toBeDefined();
        expect(pieChart.type).toBe('PieChart');
        expect(pieChart.name).toBe('饼图');
        expect(pieChart.category).toBe('图表组件');
    });

    it('should retrieve GaugeChart component definition', () => {
        registerAllComponentsSync([], [], [], chartComponents);

        const gaugeChart = componentFactory.get('GaugeChart');

        expect(gaugeChart).toBeDefined();
        expect(gaugeChart.type).toBe('GaugeChart');
        expect(gaugeChart.name).toBe('仪表盘');
        expect(gaugeChart.category).toBe('图表组件');
    });

    it('should retrieve ScatterChart component definition', () => {
        registerAllComponentsSync([], [], [], chartComponents);

        const scatterChart = componentFactory.get('ScatterChart');

        expect(scatterChart).toBeDefined();
        expect(scatterChart.type).toBe('ScatterChart');
        expect(scatterChart.name).toBe('散点图');
        expect(scatterChart.category).toBe('图表组件');
    });

    it('should have default props for all chart components', () => {
        registerAllComponentsSync([], [], [], chartComponents);

        const types = ['LineChart', 'BarChart', 'PieChart', 'GaugeChart', 'ScatterChart'];

        types.forEach((type) => {
            const component = componentFactory.get(type);
            expect(component.defaultProps).toBeDefined();
            expect(typeof component.defaultProps).toBe('object');
        });
    });

    it('should have default style for all chart components', () => {
        registerAllComponentsSync([], [], [], chartComponents);

        const types = ['LineChart', 'BarChart', 'PieChart', 'GaugeChart', 'ScatterChart'];

        types.forEach((type) => {
            const component = componentFactory.get(type);
            expect(component.defaultStyle).toBeDefined();
            expect(typeof component.defaultStyle).toBe('object');
            expect(component.defaultStyle.position).toBe('absolute');
        });
    });

    it('should have props schema for all chart components', () => {
        registerAllComponentsSync([], [], [], chartComponents);

        const types = ['LineChart', 'BarChart', 'PieChart', 'GaugeChart', 'ScatterChart'];

        types.forEach((type) => {
            const component = componentFactory.get(type);
            expect(component.propsSchema).toBeDefined();
            expect(typeof component.propsSchema).toBe('object');
        });
    });

    it('should create LineChart instance with default props', () => {
        registerAllComponentsSync([], [], [], chartComponents);

        const instance = componentFactory.createInstance('LineChart');

        expect(instance).toBeDefined();
        expect(instance.id).toBeDefined();
        expect(instance.type).toBe('LineChart');
        expect(instance.props).toBeDefined();
        expect(instance.props.title).toBe('折线图');
        expect(instance.props.showTitle).toBe(true);
        expect(instance.props.showLegend).toBe(true);
        expect(instance.style).toBeDefined();
        expect(instance.children).toEqual([]);
    });

    it('should create BarChart instance with custom props', () => {
        registerAllComponentsSync([], [], [], chartComponents);

        const instance = componentFactory.createInstance('BarChart', {
            props: { title: '销售数据', direction: 'horizontal' },
            style: { left: 200, top: 200, width: 500, height: 400 },
        });

        expect(instance).toBeDefined();
        expect(instance.type).toBe('BarChart');
        expect(instance.props.title).toBe('销售数据');
        expect(instance.props.direction).toBe('horizontal');
        expect(instance.style.left).toBe(200);
        expect(instance.style.width).toBe(500);
    });

    it('should create PieChart instance with data', () => {
        registerAllComponentsSync([], [], [], chartComponents);

        const instance = componentFactory.createInstance('PieChart', {
            props: {
                title: '市场份额',
                roseType: true,
            },
        });

        expect(instance).toBeDefined();
        expect(instance.type).toBe('PieChart');
        expect(instance.props.title).toBe('市场份额');
        expect(instance.props.roseType).toBe(true);
        expect(instance.props.data).toBeDefined();
        expect(Array.isArray(instance.props.data)).toBe(true);
    });

    it('should get all chart components by category', () => {
        registerAllComponentsSync([], [], [], chartComponents);

        const byCategory = componentFactory.getAllByCategory();

        expect(byCategory['图表组件']).toBeDefined();
        expect(byCategory['图表组件']).toHaveLength(5);
    });

    it('should search chart components by keyword', () => {
        registerAllComponentsSync([], [], [], chartComponents);

        const results = componentFactory.search('折线图');

        expect(results.length).toBeGreaterThan(0);
        expect(results.some((comp) => comp.type === 'LineChart')).toBe(true);
    });

    it('should have LineChart with series data', () => {
        registerAllComponentsSync([], [], [], chartComponents);

        const lineChart = componentFactory.get('LineChart');

        expect(lineChart.defaultProps.series).toBeDefined();
        expect(Array.isArray(lineChart.defaultProps.series)).toBe(true);
        expect(lineChart.defaultProps.series.length).toBeGreaterThan(0);
        expect(lineChart.defaultProps.xAxisData).toBeDefined();
        expect(Array.isArray(lineChart.defaultProps.xAxisData)).toBe(true);
    });

    it('should have BarChart with direction options', () => {
        registerAllComponentsSync([], [], [], chartComponents);

        const barChart = componentFactory.get('BarChart');
        const directionSchema = barChart.propsSchema.direction;

        expect(directionSchema).toBeDefined();
        expect(directionSchema.type).toBe('enum');
        expect(directionSchema.options).toHaveLength(2);

        const directionValues = directionSchema.options.map((opt) => opt.value);
        expect(directionValues).toContain('vertical');
        expect(directionValues).toContain('horizontal');
    });

    it('should have PieChart with radius property', () => {
        registerAllComponentsSync([], [], [], chartComponents);

        const pieChart = componentFactory.get('PieChart');

        expect(pieChart.propsSchema.radius).toBeDefined();
        expect(pieChart.propsSchema.radius.type).toBe('array');
        expect(pieChart.defaultProps.radius).toBeDefined();
        expect(Array.isArray(pieChart.defaultProps.radius)).toBe(true);
        expect(pieChart.defaultProps.radius).toHaveLength(2);
    });

    it('should have GaugeChart with value range', () => {
        registerAllComponentsSync([], [], [], chartComponents);

        const gaugeChart = componentFactory.get('GaugeChart');

        expect(gaugeChart.propsSchema.min).toBeDefined();
        expect(gaugeChart.propsSchema.max).toBeDefined();
        expect(gaugeChart.propsSchema.value).toBeDefined();
        expect(gaugeChart.defaultProps.min).toBe(0);
        expect(gaugeChart.defaultProps.max).toBe(100);
        expect(gaugeChart.defaultProps.value).toBe(75);
    });

    it('should have ScatterChart with symbolSize property', () => {
        registerAllComponentsSync([], [], [], chartComponents);

        const scatterChart = componentFactory.get('ScatterChart');
        const symbolSizeSchema = scatterChart.propsSchema.symbolSize;

        expect(symbolSizeSchema).toBeDefined();
        expect(symbolSizeSchema.type).toBe('number');
        expect(symbolSizeSchema.min).toBe(1);
        expect(symbolSizeSchema.max).toBe(50);
        expect(scatterChart.defaultProps.symbolSize).toBe(10);
    });

    it('should mark all chart components as non-container', () => {
        registerAllComponentsSync([], [], [], chartComponents);

        const types = ['LineChart', 'BarChart', 'PieChart', 'GaugeChart', 'ScatterChart'];

        types.forEach((type) => {
            const component = componentFactory.get(type);
            expect(component.container).toBe(false);
        });
    });

    it('should have events schema for all chart components', () => {
        registerAllComponentsSync([], [], [], chartComponents);

        const lineChart = componentFactory.get('LineChart');
        const barChart = componentFactory.get('BarChart');
        const pieChart = componentFactory.get('PieChart');

        expect(lineChart.eventsSchema).toBeDefined();
        expect(lineChart.eventsSchema.click).toBeDefined();

        expect(barChart.eventsSchema).toBeDefined();
        expect(barChart.eventsSchema.click).toBeDefined();

        expect(pieChart.eventsSchema).toBeDefined();
        expect(pieChart.eventsSchema.click).toBeDefined();
    });

    it('should have title property for all chart components', () => {
        registerAllComponentsSync([], [], [], chartComponents);

        const types = ['LineChart', 'BarChart', 'PieChart', 'GaugeChart', 'ScatterChart'];

        types.forEach((type) => {
            const component = componentFactory.get(type);
            expect(component.propsSchema.title).toBeDefined();
            expect(component.propsSchema.title.type).toBe('string');
            expect(component.defaultProps.title).toBeDefined();
        });
    });

    it('should have showTitle property for all chart components', () => {
        registerAllComponentsSync([], [], [], chartComponents);

        const types = ['LineChart', 'BarChart', 'PieChart', 'GaugeChart', 'ScatterChart'];

        types.forEach((type) => {
            const component = componentFactory.get(type);
            expect(component.propsSchema.showTitle).toBeDefined();
            expect(component.propsSchema.showTitle.type).toBe('boolean');
            expect(component.defaultProps.showTitle).toBe(true);
        });
    });

    it('should have tags for all chart components', () => {
        registerAllComponentsSync([], [], [], chartComponents);

        const types = ['LineChart', 'BarChart', 'PieChart', 'GaugeChart', 'ScatterChart'];

        types.forEach((type) => {
            const component = componentFactory.get(type);
            expect(component.tags).toBeDefined();
            expect(Array.isArray(component.tags)).toBe(true);
            expect(component.tags.length).toBeGreaterThan(0);
            expect(component.tags).toContain('图表');
            expect(component.tags).toContain('echarts');
        });
    });

    it('should have description for all chart components', () => {
        registerAllComponentsSync([], [], [], chartComponents);

        const types = ['LineChart', 'BarChart', 'PieChart', 'GaugeChart', 'ScatterChart'];

        types.forEach((type) => {
            const component = componentFactory.get(type);
            expect(component.description).toBeDefined();
            expect(typeof component.description).toBe('string');
            expect(component.description.length).toBeGreaterThan(0);
            expect(component.description).toContain('ECharts');
        });
    });

    it('should have appropriate default dimensions for chart components', () => {
        registerAllComponentsSync([], [], [], chartComponents);

        const lineChart = componentFactory.get('LineChart');
        const barChart = componentFactory.get('BarChart');
        const pieChart = componentFactory.get('PieChart');
        const gaugeChart = componentFactory.get('GaugeChart');
        const scatterChart = componentFactory.get('ScatterChart');

        // Most charts should have 400x300 default size
        expect(lineChart.defaultStyle.width).toBe(400);
        expect(lineChart.defaultStyle.height).toBe(300);
        expect(barChart.defaultStyle.width).toBe(400);
        expect(barChart.defaultStyle.height).toBe(300);
        expect(pieChart.defaultStyle.width).toBe(400);
        expect(pieChart.defaultStyle.height).toBe(300);
        expect(scatterChart.defaultStyle.width).toBe(400);
        expect(scatterChart.defaultStyle.height).toBe(300);

        // Gauge chart should be square
        expect(gaugeChart.defaultStyle.width).toBe(300);
        expect(gaugeChart.defaultStyle.height).toBe(300);
    });

    it('should have version for all chart components', () => {
        registerAllComponentsSync([], [], [], chartComponents);

        const types = ['LineChart', 'BarChart', 'PieChart', 'GaugeChart', 'ScatterChart'];

        types.forEach((type) => {
            const component = componentFactory.get(type);
            expect(component.version).toBeDefined();
            expect(component.version).toBe('1.0.0');
        });
    });

    it('should have icon for all chart components', () => {
        registerAllComponentsSync([], [], [], chartComponents);

        const types = ['LineChart', 'BarChart', 'PieChart', 'GaugeChart', 'ScatterChart'];

        types.forEach((type) => {
            const component = componentFactory.get(type);
            expect(component.icon).toBeDefined();
            expect(typeof component.icon).toBe('string');
        });
    });

    it('should create multiple chart instances with unique IDs', () => {
        registerAllComponentsSync([], [], [], chartComponents);

        const instance1 = componentFactory.createInstance('LineChart');
        const instance2 = componentFactory.createInstance('LineChart');
        const instance3 = componentFactory.createInstance('BarChart');

        expect(instance1.id).not.toBe(instance2.id);
        expect(instance1.id).not.toBe(instance3.id);
        expect(instance2.id).not.toBe(instance3.id);
    });

    it('should have smooth property for LineChart', () => {
        registerAllComponentsSync([], [], [], chartComponents);

        const lineChart = componentFactory.get('LineChart');

        expect(lineChart.propsSchema.smooth).toBeDefined();
        expect(lineChart.propsSchema.smooth.type).toBe('boolean');
        expect(lineChart.defaultProps.smooth).toBe(false);
    });

    it('should have roseType property for PieChart', () => {
        registerAllComponentsSync([], [], [], chartComponents);

        const pieChart = componentFactory.get('PieChart');

        expect(pieChart.propsSchema.roseType).toBeDefined();
        expect(pieChart.propsSchema.roseType.type).toBe('boolean');
        expect(pieChart.defaultProps.roseType).toBe(false);
    });

    it('should have unit property for GaugeChart', () => {
        registerAllComponentsSync([], [], [], chartComponents);

        const gaugeChart = componentFactory.get('GaugeChart');

        expect(gaugeChart.propsSchema.unit).toBeDefined();
        expect(gaugeChart.propsSchema.unit.type).toBe('string');
        expect(gaugeChart.defaultProps.unit).toBe('%');
    });

    it('should have showGrid property for applicable charts', () => {
        registerAllComponentsSync([], [], [], chartComponents);

        const lineChart = componentFactory.get('LineChart');
        const barChart = componentFactory.get('BarChart');
        const scatterChart = componentFactory.get('ScatterChart');

        expect(lineChart.propsSchema.showGrid).toBeDefined();
        expect(lineChart.defaultProps.showGrid).toBe(true);

        expect(barChart.propsSchema.showGrid).toBeDefined();
        expect(barChart.defaultProps.showGrid).toBe(true);

        expect(scatterChart.propsSchema.showGrid).toBeDefined();
        expect(scatterChart.defaultProps.showGrid).toBe(true);
    });

    it('should have showLegend property for multi-series charts', () => {
        registerAllComponentsSync([], [], [], chartComponents);

        const lineChart = componentFactory.get('LineChart');
        const barChart = componentFactory.get('BarChart');
        const pieChart = componentFactory.get('PieChart');
        const scatterChart = componentFactory.get('ScatterChart');

        expect(lineChart.propsSchema.showLegend).toBeDefined();
        expect(lineChart.defaultProps.showLegend).toBe(true);

        expect(barChart.propsSchema.showLegend).toBeDefined();
        expect(barChart.defaultProps.showLegend).toBe(true);

        expect(pieChart.propsSchema.showLegend).toBeDefined();
        expect(pieChart.defaultProps.showLegend).toBe(true);

        expect(scatterChart.propsSchema.showLegend).toBeDefined();
        expect(scatterChart.defaultProps.showLegend).toBe(true);
    });
});
