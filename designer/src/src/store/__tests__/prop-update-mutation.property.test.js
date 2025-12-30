/**
 * Property Test: Prop Update Mutation
 * **Feature: design-center, Property 7: Prop Update Mutation**
 * **Validates: Requirements 4.2, 4.3**
 *
 * *For any* component and prop update (key, value), the resulting schema SHALL have
 * component.props[key] === value with all other props unchanged.
 *
 * *For any* component and style update (key, value), the resulting schema SHALL have
 * component.style[key] === value with all other styles unchanged.
 */
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';
import * as fc from 'fast-check';
import { useDesignStore } from '@/store/design';

// Mock the design API to prevent actual API calls
vi.mock('@/api/design.api', () => ({
    designAPI: {
        getPages: vi.fn(),
        getPage: vi.fn(),
        createPage: vi.fn(),
        updatePage: vi.fn(),
        deletePage: vi.fn(),
        renamePage: vi.fn().mockResolvedValue({}),
    },
}));

/**
 * Generates a valid component ID (UUID format)
 */
const componentIdArb = fc.uuid();

/**
 * Generates a valid component label
 */
const componentLabelArb = fc.stringMatching(/^[a-zA-Z][a-zA-Z0-9 ]{0,19}$/).filter((s) => s.trim().length > 0);

/**
 * Generates initial props for a component
 */
const initialPropsArb = fc.record({
    text: fc.string({ minLength: 0, maxLength: 50 }),
    disabled: fc.boolean(),
    placeholder: fc.string({ minLength: 0, maxLength: 30 }),
    value: fc.oneof(fc.string(), fc.integer()),
});

/**
 * Generates initial style for a component
 */
const initialStyleArb = fc.record({
    position: fc.constant('absolute'),
    left: fc.integer({ min: 0, max: 1000 }),
    top: fc.integer({ min: 0, max: 1000 }),
    width: fc.integer({ min: 10, max: 500 }),
    height: fc.integer({ min: 10, max: 500 }),
    zIndex: fc.integer({ min: 0, max: 100 }),
});

/**
 * Generates a valid component schema with props and style
 */
const componentSchemaArb = fc.tuple(componentIdArb, componentLabelArb, initialPropsArb, initialStyleArb).map(([id, label, props, style]) => ({
    id,
    type: 'Button',
    label,
    locked: false,
    visible: true,
    style,
    props,
    children: [],
}));

/**
 * Generates a prop key that exists in the component
 */
const propKeyArb = fc.constantFrom('text', 'disabled', 'placeholder', 'value');

/**
 * Generates a new prop value
 */
const propValueArb = fc.oneof(fc.string({ minLength: 1, maxLength: 50 }), fc.boolean(), fc.integer({ min: 0, max: 1000 }));

/**
 * Generates a style key that exists in the component
 */
const styleKeyArb = fc.constantFrom('left', 'top', 'width', 'height', 'zIndex');

/**
 * Generates a new style value (numeric for position/dimension properties)
 */
const styleValueArb = fc.integer({ min: 0, max: 2000 });

/**
 * Helper to create a fresh store with a page containing a component
 */
function createStoreWithComponent(component) {
    const pinia = createPinia();
    setActivePinia(pinia);
    const store = useDesignStore();

    const pageSchema = {
        version: '2.0.0',
        meta: {
            id: 'test-page-id',
            name: 'Test Page',
            description: '',
        },
        config: {
            width: 1920,
            height: 1080,
            scaleMode: 'fit',
            backgroundColor: '#ffffff',
            gridSize: 10,
            snapToGrid: true,
            theme: 'light',
        },
        variables: {},
        dataSources: [],
        components: [JSON.parse(JSON.stringify(component))], // Deep clone
        permissions: {},
    };

    store.projectId = 'test-project';
    store.currentPageId = 'test-page-id';
    store.currentPage = pageSchema;
    store.isDirty = false;

    return store;
}

describe('Property 7: Prop Update Mutation', () => {
    beforeEach(() => {
        setActivePinia(createPinia());
    });

    /**
     * Property: For any component and prop update (key, value), the resulting schema
     * SHALL have component.props[key] === value
     * **Validates: Requirements 4.2**
     */
    it('should update component.props[key] to the new value', () => {
        fc.assert(
            fc.property(componentSchemaArb, propKeyArb, propValueArb, (component, propKey, newValue) => {
                const store = createStoreWithComponent(component);
                const componentId = component.id;

                // Action: update a single prop
                store.updateComponent(componentId, { props: { [propKey]: newValue } });

                // Get the updated component
                const updatedComponent = store.currentPage.components[0];

                // Postcondition: the prop value is updated
                expect(updatedComponent.props[propKey]).toEqual(newValue);

                return true;
            }),
            { numRuns: 100 },
        );
    });

    /**
     * Property: For any prop update, all other props SHALL remain unchanged
     * **Validates: Requirements 4.2**
     */
    it('should preserve all other props when updating a single prop', () => {
        fc.assert(
            fc.property(componentSchemaArb, propKeyArb, propValueArb, (component, propKey, newValue) => {
                const store = createStoreWithComponent(component);
                const componentId = component.id;

                // Capture original props (excluding the one we're updating)
                const originalProps = { ...component.props };
                delete originalProps[propKey];

                // Action: update a single prop
                store.updateComponent(componentId, { props: { [propKey]: newValue } });

                // Get the updated component
                const updatedComponent = store.currentPage.components[0];

                // Postcondition: all other props remain unchanged
                Object.keys(originalProps).forEach((key) => {
                    expect(updatedComponent.props[key]).toEqual(originalProps[key]);
                });

                return true;
            }),
            { numRuns: 100 },
        );
    });

    /**
     * Property: For any component and style update (key, value), the resulting schema
     * SHALL have component.style[key] === value
     * **Validates: Requirements 4.3**
     */
    it('should update component.style[key] to the new value', () => {
        fc.assert(
            fc.property(componentSchemaArb, styleKeyArb, styleValueArb, (component, styleKey, newValue) => {
                const store = createStoreWithComponent(component);
                const componentId = component.id;

                // Action: update a single style property
                store.updateComponent(componentId, { style: { [styleKey]: newValue } });

                // Get the updated component
                const updatedComponent = store.currentPage.components[0];

                // Postcondition: the style value is updated
                expect(updatedComponent.style[styleKey]).toEqual(newValue);

                return true;
            }),
            { numRuns: 100 },
        );
    });

    /**
     * Property: For any style update, all other style properties SHALL remain unchanged
     * **Validates: Requirements 4.3**
     */
    it('should preserve all other styles when updating a single style property', () => {
        fc.assert(
            fc.property(componentSchemaArb, styleKeyArb, styleValueArb, (component, styleKey, newValue) => {
                const store = createStoreWithComponent(component);
                const componentId = component.id;

                // Capture original styles (excluding the one we're updating)
                const originalStyle = { ...component.style };
                delete originalStyle[styleKey];

                // Action: update a single style property
                store.updateComponent(componentId, { style: { [styleKey]: newValue } });

                // Get the updated component
                const updatedComponent = store.currentPage.components[0];

                // Postcondition: all other styles remain unchanged
                Object.keys(originalStyle).forEach((key) => {
                    expect(updatedComponent.style[key]).toEqual(originalStyle[key]);
                });

                return true;
            }),
            { numRuns: 100 },
        );
    });

    /**
     * Property: Multiple prop updates should each be applied correctly
     * **Validates: Requirements 4.2**
     */
    it('should correctly apply multiple prop updates in sequence', () => {
        fc.assert(
            fc.property(componentSchemaArb, fc.array(fc.tuple(propKeyArb, propValueArb), { minLength: 1, maxLength: 4 }), (component, updates) => {
                const store = createStoreWithComponent(component);
                const componentId = component.id;

                // Apply each update and track expected final state
                const expectedProps = { ...component.props };

                for (const [key, value] of updates) {
                    store.updateComponent(componentId, { props: { [key]: value } });
                    expectedProps[key] = value;
                }

                // Get the updated component
                const updatedComponent = store.currentPage.components[0];

                // Postcondition: all props match expected values
                Object.keys(expectedProps).forEach((key) => {
                    expect(updatedComponent.props[key]).toEqual(expectedProps[key]);
                });

                return true;
            }),
            { numRuns: 100 },
        );
    });

    /**
     * Property: Updating props should not affect style, and vice versa
     * **Validates: Requirements 4.2, 4.3**
     */
    it('should not affect style when updating props', () => {
        fc.assert(
            fc.property(componentSchemaArb, propKeyArb, propValueArb, (component, propKey, newValue) => {
                const store = createStoreWithComponent(component);
                const componentId = component.id;

                // Capture original style
                const originalStyle = { ...component.style };

                // Action: update a prop
                store.updateComponent(componentId, { props: { [propKey]: newValue } });

                // Get the updated component
                const updatedComponent = store.currentPage.components[0];

                // Postcondition: style remains unchanged
                Object.keys(originalStyle).forEach((key) => {
                    expect(updatedComponent.style[key]).toEqual(originalStyle[key]);
                });

                return true;
            }),
            { numRuns: 100 },
        );
    });

    /**
     * Property: Updating style should not affect props
     * **Validates: Requirements 4.2, 4.3**
     */
    it('should not affect props when updating style', () => {
        fc.assert(
            fc.property(componentSchemaArb, styleKeyArb, styleValueArb, (component, styleKey, newValue) => {
                const store = createStoreWithComponent(component);
                const componentId = component.id;

                // Capture original props
                const originalProps = { ...component.props };

                // Action: update a style property
                store.updateComponent(componentId, { style: { [styleKey]: newValue } });

                // Get the updated component
                const updatedComponent = store.currentPage.components[0];

                // Postcondition: props remain unchanged
                Object.keys(originalProps).forEach((key) => {
                    expect(updatedComponent.props[key]).toEqual(originalProps[key]);
                });

                return true;
            }),
            { numRuns: 100 },
        );
    });
});
