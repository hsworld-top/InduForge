/**
 * Property Test: Dirty State Tracking
 * **Feature: design-center, Property 10: Dirty State Tracking**
 * **Validates: Requirements 6.1**
 *
 * *For any* schema modification operation, the isDirty flag SHALL be set to true.
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
 * Generates a valid component schema
 */
const componentSchemaArb = fc.record({
    id: componentIdArb,
    type: fc.constantFrom('Container', 'Text', 'Button', 'Image', 'Input'),
    label: componentLabelArb,
    locked: fc.boolean(),
    visible: fc.boolean(),
    style: fc.record({
        position: fc.constant('absolute'),
        left: fc.integer({ min: 0, max: 1000 }),
        top: fc.integer({ min: 0, max: 1000 }),
        width: fc.integer({ min: 10, max: 500 }),
        height: fc.integer({ min: 10, max: 500 }),
        zIndex: fc.integer({ min: 0, max: 100 }),
    }),
    props: fc.constant({}),
    children: fc.constant([]),
});

/**
 * Generates a valid page schema with at least one component
 */
const pageSchemaWithComponentsArb = fc.array(componentSchemaArb, { minLength: 1, maxLength: 5 }).map((components) => ({
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
    components,
    permissions: {},
}));

/**
 * Generates prop updates for a component
 */
const propUpdatesArb = fc.record({
    props: fc.record({
        text: fc.string({ minLength: 1, maxLength: 50 }),
    }),
});

/**
 * Generates style updates for a component
 */
const styleUpdatesArb = fc.record({
    style: fc.record({
        left: fc.integer({ min: 0, max: 1000 }),
        top: fc.integer({ min: 0, max: 1000 }),
    }),
});

/**
 * Helper to create a fresh store with a page loaded
 */
function createStoreWithPage(pageSchema) {
    const pinia = createPinia();
    setActivePinia(pinia);
    const store = useDesignStore();

    store.projectId = 'test-project';
    store.currentPageId = 'test-page-id';
    store.currentPage = JSON.parse(JSON.stringify(pageSchema)); // Deep clone
    store.isDirty = false; // Ensure clean state

    return store;
}

describe('Property 10: Dirty State Tracking', () => {
    beforeEach(() => {
        setActivePinia(createPinia());
    });

    /**
     * Property: For any component update with props, isDirty SHALL be set to true
     */
    it('should set isDirty to true when updating component props', () => {
        fc.assert(
            fc.property(pageSchemaWithComponentsArb, propUpdatesArb, (pageSchema, updates) => {
                const store = createStoreWithPage(pageSchema);
                const componentId = pageSchema.components[0].id;

                // Precondition: isDirty is false
                expect(store.isDirty).toBe(false);

                // Action: update component props
                store.updateComponent(componentId, updates);

                // Postcondition: isDirty is true
                expect(store.isDirty).toBe(true);

                return true;
            }),
            { numRuns: 100 },
        );
    });

    /**
     * Property: For any component update with style, isDirty SHALL be set to true
     */
    it('should set isDirty to true when updating component style', () => {
        fc.assert(
            fc.property(pageSchemaWithComponentsArb, styleUpdatesArb, (pageSchema, updates) => {
                const store = createStoreWithPage(pageSchema);
                const componentId = pageSchema.components[0].id;

                // Precondition: isDirty is false
                expect(store.isDirty).toBe(false);

                // Action: update component style
                store.updateComponent(componentId, updates);

                // Postcondition: isDirty is true
                expect(store.isDirty).toBe(true);

                return true;
            }),
            { numRuns: 100 },
        );
    });

    /**
     * Property: For any new component added, isDirty SHALL be set to true
     */
    it('should set isDirty to true when adding a component', () => {
        fc.assert(
            fc.property(pageSchemaWithComponentsArb, componentSchemaArb, (pageSchema, newComponent) => {
                const store = createStoreWithPage(pageSchema);

                // Precondition: isDirty is false
                expect(store.isDirty).toBe(false);

                // Action: add a new component
                store.addComponent(newComponent);

                // Postcondition: isDirty is true
                expect(store.isDirty).toBe(true);

                return true;
            }),
            { numRuns: 100 },
        );
    });

    /**
     * Property: For any component removed, isDirty SHALL be set to true
     */
    it('should set isDirty to true when removing a component', () => {
        fc.assert(
            fc.property(pageSchemaWithComponentsArb, (pageSchema) => {
                const store = createStoreWithPage(pageSchema);
                const componentId = pageSchema.components[0].id;

                // Precondition: isDirty is false
                expect(store.isDirty).toBe(false);

                // Action: remove a component
                store.removeComponent(componentId);

                // Postcondition: isDirty is true
                expect(store.isDirty).toBe(true);

                return true;
            }),
            { numRuns: 100 },
        );
    });

    /**
     * Property: For any component moved, isDirty SHALL be set to true
     */
    it('should set isDirty to true when moving a component', () => {
        fc.assert(
            fc.property(
                // Generate page with at least 2 components for meaningful move
                fc.array(componentSchemaArb, { minLength: 2, maxLength: 5 }).map((components) => ({
                    version: '2.0.0',
                    meta: { id: 'test-page-id', name: 'Test Page', description: '' },
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
                    components,
                    permissions: {},
                })),
                fc.integer({ min: 0, max: 10 }),
                (pageSchema, targetIndex) => {
                    const store = createStoreWithPage(pageSchema);
                    const componentId = pageSchema.components[0].id;
                    const safeIndex = Math.min(targetIndex, pageSchema.components.length);

                    // Precondition: isDirty is false
                    expect(store.isDirty).toBe(false);

                    // Action: move component to a new position (null = root level)
                    store.moveComponent(componentId, null, safeIndex);

                    // Postcondition: isDirty is true
                    expect(store.isDirty).toBe(true);

                    return true;
                },
            ),
            { numRuns: 100 },
        );
    });

    /**
     * Property: After saving, isDirty SHALL be reset to false
     * (This tests the complementary behavior - save clears dirty state)
     */
    it('should reset isDirty to false after successful save', async () => {
        await fc.assert(
            fc.asyncProperty(pageSchemaWithComponentsArb, async (pageSchema) => {
                const store = createStoreWithPage(pageSchema);
                const componentId = pageSchema.components[0].id;

                // Make a change to set isDirty
                store.updateComponent(componentId, { props: { text: 'changed' } });
                expect(store.isDirty).toBe(true);

                // Save the page
                await store.savePage();

                // isDirty should be reset to false
                expect(store.isDirty).toBe(false);

                return true;
            }),
            { numRuns: 100 },
        );
    });

    /**
     * Property: Multiple modifications should keep isDirty as true
     */
    it('should keep isDirty true after multiple modifications', () => {
        fc.assert(
            fc.property(pageSchemaWithComponentsArb, fc.array(propUpdatesArb, { minLength: 1, maxLength: 5 }), (pageSchema, updatesList) => {
                const store = createStoreWithPage(pageSchema);
                const componentId = pageSchema.components[0].id;

                // Precondition: isDirty is false
                expect(store.isDirty).toBe(false);

                // Action: apply multiple updates
                for (const updates of updatesList) {
                    store.updateComponent(componentId, updates);
                }

                // Postcondition: isDirty is still true
                expect(store.isDirty).toBe(true);

                return true;
            }),
            { numRuns: 100 },
        );
    });

    /**
     * Property: Adding component to a parent should set isDirty to true
     */
    it('should set isDirty to true when adding component to a parent', () => {
        fc.assert(
            fc.property(componentSchemaArb, componentSchemaArb, (parentComponent, childComponent) => {
                // Create a container parent
                const parent = {
                    ...parentComponent,
                    type: 'Container',
                    children: [],
                };

                const pageSchema = {
                    version: '2.0.0',
                    meta: { id: 'test-page-id', name: 'Test Page', description: '' },
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
                    components: [parent],
                    permissions: {},
                };

                const store = createStoreWithPage(pageSchema);

                // Precondition: isDirty is false
                expect(store.isDirty).toBe(false);

                // Action: add child component to parent
                store.addComponent(childComponent, parent.id);

                // Postcondition: isDirty is true
                expect(store.isDirty).toBe(true);

                return true;
            }),
            { numRuns: 100 },
        );
    });
});
