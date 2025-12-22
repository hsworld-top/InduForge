/**
 * Property Test: Control Type Selection
 * **Feature: design-center, Property 6: Control Type Selection**
 * **Validates: Requirements 4.4, 4.5, 4.6, 4.7**
 *
 * *For any* prop definition with type T, the Property Panel SHALL render:
 * - switch for boolean
 * - number input for number
 * - text input for string
 * - select for enum
 */
import { describe, it, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import * as fc from 'fast-check';
import PropsEditor from '../PropsEditor.vue';
import ElementPlus from 'element-plus';

/**
 * Determines the expected control CSS class based on prop schema type
 * @param {Object} schema - The prop schema
 * @returns {string} Expected control CSS class
 */
function getExpectedControlClass(schema) {
    if (schema.type === 'boolean') {
        return 'el-switch';
    }
    if (schema.type === 'number') {
        return 'el-input-number';
    }
    if (schema.type === 'enum') {
        return 'el-select';
    }
    if (schema.type === 'string') {
        if (schema.format === 'color') {
            return 'el-color-picker';
        }
        return 'el-input';
    }
    return 'el-input'; // default fallback
}

/**
 * Arbitrary for generating boolean prop schemas
 */
const booleanPropSchemaArb = fc.record({
    type: fc.constant('boolean'),
    label: fc.string({ minLength: 1, maxLength: 20 }),
    default: fc.boolean(),
});

/**
 * Arbitrary for generating number prop schemas
 */
const numberPropSchemaArb = fc.record({
    type: fc.constant('number'),
    label: fc.string({ minLength: 1, maxLength: 20 }),
    default: fc.integer({ min: 0, max: 1000 }),
    min: fc.option(fc.integer({ min: 0, max: 100 }), { nil: undefined }),
    max: fc.option(fc.integer({ min: 100, max: 1000 }), { nil: undefined }),
    step: fc.option(fc.integer({ min: 1, max: 10 }), { nil: undefined }),
});

/**
 * Arbitrary for generating string prop schemas
 */
const stringPropSchemaArb = fc.record({
    type: fc.constant('string'),
    label: fc.string({ minLength: 1, maxLength: 20 }),
    default: fc.string({ minLength: 0, maxLength: 50 }),
    multiline: fc.constant(false),
    format: fc.constant(undefined),
});

/**
 * Arbitrary for generating enum prop schemas
 */
const enumPropSchemaArb = fc
    .record({
        type: fc.constant('enum'),
        label: fc.string({ minLength: 1, maxLength: 20 }),
        options: fc.array(
            fc.record({
                label: fc.string({ minLength: 1, maxLength: 20 }),
                value: fc.string({ minLength: 1, maxLength: 20 }),
            }),
            { minLength: 1, maxLength: 5 },
        ),
    })
    .map((schema) => ({
        ...schema,
        default: schema.options[0]?.value,
    }));

/**
 * Arbitrary for generating any valid prop schema
 */
const propSchemaArb = fc.oneof(booleanPropSchemaArb, numberPropSchemaArb, stringPropSchemaArb, enumPropSchemaArb);

/**
 * Arbitrary for generating a prop key
 */
const propKeyArb = fc.string({ minLength: 1, maxLength: 20 }).filter((s) => /^[a-zA-Z][a-zA-Z0-9]*$/.test(s));

describe('Property 6: Control Type Selection', () => {
    /**
     * Property: For any prop definition with type T, the correct control is rendered
     * - boolean -> el-switch
     * - number -> el-input-number
     * - string -> el-input
     * - enum -> el-select
     */
    it('should render the correct control type for any prop schema type', () => {
        fc.assert(
            fc.property(propKeyArb, propSchemaArb, (propKey, propSchema) => {
                const propsSchema = { [propKey]: propSchema };
                const propsValue = propSchema.default !== undefined ? { [propKey]: propSchema.default } : {};

                const wrapper = mount(PropsEditor, {
                    props: {
                        props: propsValue,
                        propsSchema,
                    },
                    global: {
                        plugins: [ElementPlus],
                    },
                });

                const expectedClass = getExpectedControlClass(propSchema);
                const html = wrapper.html();

                // Check that the expected control class exists in the rendered HTML
                expect(html).toContain(expectedClass);

                wrapper.unmount();
                return true;
            }),
            { numRuns: 100 },
        );
    });

    /**
     * Property: Boolean props always render switch controls
     * Requirements: 4.4
     */
    it('should render switch for boolean props', () => {
        fc.assert(
            fc.property(propKeyArb, booleanPropSchemaArb, (propKey, propSchema) => {
                const wrapper = mount(PropsEditor, {
                    props: {
                        props: { [propKey]: propSchema.default },
                        propsSchema: { [propKey]: propSchema },
                    },
                    global: {
                        plugins: [ElementPlus],
                    },
                });

                const html = wrapper.html();

                // Should have switch control
                expect(html).toContain('el-switch');

                // Should NOT have other control types
                expect(html).not.toContain('el-input-number');
                expect(html).not.toContain('el-select');

                wrapper.unmount();
                return true;
            }),
            { numRuns: 100 },
        );
    });

    /**
     * Property: Number props always render number input controls
     * Requirements: 4.5
     */
    it('should render number input for number props', () => {
        fc.assert(
            fc.property(propKeyArb, numberPropSchemaArb, (propKey, propSchema) => {
                const wrapper = mount(PropsEditor, {
                    props: {
                        props: { [propKey]: propSchema.default },
                        propsSchema: { [propKey]: propSchema },
                    },
                    global: {
                        plugins: [ElementPlus],
                    },
                });

                const html = wrapper.html();

                // Should have number input control
                expect(html).toContain('el-input-number');

                // Should NOT have other control types
                expect(html).not.toContain('el-switch');
                expect(html).not.toContain('el-select');

                wrapper.unmount();
                return true;
            }),
            { numRuns: 100 },
        );
    });

    /**
     * Property: String props always render text input controls
     * Requirements: 4.6
     */
    it('should render text input for string props', () => {
        fc.assert(
            fc.property(propKeyArb, stringPropSchemaArb, (propKey, propSchema) => {
                const wrapper = mount(PropsEditor, {
                    props: {
                        props: { [propKey]: propSchema.default },
                        propsSchema: { [propKey]: propSchema },
                    },
                    global: {
                        plugins: [ElementPlus],
                    },
                });

                const html = wrapper.html();

                // Should have text input control (el-input but not el-input-number)
                expect(html).toContain('el-input');
                expect(html).not.toContain('el-input-number');

                // Should NOT have other control types
                expect(html).not.toContain('el-switch');
                expect(html).not.toContain('el-select');

                wrapper.unmount();
                return true;
            }),
            { numRuns: 100 },
        );
    });

    /**
     * Property: Enum props always render select controls
     * Requirements: 4.7
     */
    it('should render select for enum props', () => {
        fc.assert(
            fc.property(propKeyArb, enumPropSchemaArb, (propKey, propSchema) => {
                const wrapper = mount(PropsEditor, {
                    props: {
                        props: { [propKey]: propSchema.default },
                        propsSchema: { [propKey]: propSchema },
                    },
                    global: {
                        plugins: [ElementPlus],
                    },
                });

                const html = wrapper.html();

                // Should have select control
                expect(html).toContain('el-select');

                // Should NOT have other control types
                expect(html).not.toContain('el-switch');
                expect(html).not.toContain('el-input-number');

                wrapper.unmount();
                return true;
            }),
            { numRuns: 100 },
        );
    });
});
