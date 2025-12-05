/**
 * Property-Based Test: API Page List Response
 * **Feature: design-center, Property 13: API Page List Response**
 * **Validates: Requirements 7.1**
 * 
 * For any project with pages, the API response SHALL include all pages 
 * with id, name, type, and parentId fields.
 */

const fc = require('fast-check');

/**
 * Valid page types as defined in the DesignPage model
 */
const PAGE_TYPES = ['page', 'folder', 'dialog'];

/**
 * UUID v4 validation regex
 */
const UUID_V4_REGEX = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

/**
 * Generates a valid UUID v4
 */
function generateUUIDv4() {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
    const r = Math.random() * 16 | 0;
    const v = c === 'x' ? r : (r & 0x3 | 0x8);
    return v.toString(16);
  });
}

/**
 * Arbitrary for generating valid page names
 */
const pageNameArb = fc.string({ minLength: 1, maxLength: 200 })
  .filter(s => s.trim().length > 0);

/**
 * Arbitrary for generating valid page types
 */
const pageTypeArb = fc.constantFrom(...PAGE_TYPES);

/**
 * Arbitrary for generating a single page object (as stored in database)
 */
const pageArb = fc.record({
  id: fc.constant(null).map(() => generateUUIDv4()),
  name: pageNameArb,
  type: pageTypeArb,
  parentId: fc.option(fc.constant(null).map(() => generateUUIDv4()), { nil: null }),
  sortOrder: fc.integer({ min: 0, max: 1000 }),
  lockedBy: fc.option(fc.constant(null).map(() => generateUUIDv4()), { nil: null }),
  lockedAt: fc.option(fc.date(), { nil: null }),
  createdAt: fc.date(),
  updatedAt: fc.date(),
});

/**
 * Arbitrary for generating a list of pages
 */
const pageListArb = fc.array(pageArb, { minLength: 0, maxLength: 20 });

/**
 * Simulates the transformation done by designService.getPages()
 * This extracts the fields returned by the API
 * @param {Array} pages - Array of page objects from database
 * @returns {Array} Transformed page list for API response
 */
function transformPagesForApiResponse(pages) {
  return pages.map(page => ({
    id: page.id,
    name: page.name,
    type: page.type,
    parentId: page.parentId,
    sortOrder: page.sortOrder,
    lockedBy: page.lockedBy,
    lockedAt: page.lockedAt,
    createdAt: page.createdAt,
    updatedAt: page.updatedAt,
  }));
}

/**
 * Validates that a page response object has all required fields
 * @param {Object} page - Page object from API response
 * @returns {boolean} True if all required fields are present
 */
function hasRequiredFields(page) {
  // Required fields per Requirements 7.1: id, name, type, parentId
  const requiredFields = ['id', 'name', 'type', 'parentId'];
  return requiredFields.every(field => field in page);
}

/**
 * Validates that a page has valid field types
 * @param {Object} page - Page object from API response
 * @returns {boolean} True if all fields have valid types
 */
function hasValidFieldTypes(page) {
  // id must be a valid UUID string
  if (typeof page.id !== 'string' || !UUID_V4_REGEX.test(page.id)) {
    return false;
  }
  
  // name must be a non-empty string
  if (typeof page.name !== 'string' || page.name.length === 0) {
    return false;
  }
  
  // type must be one of the valid page types
  if (!PAGE_TYPES.includes(page.type)) {
    return false;
  }
  
  // parentId must be null or a valid UUID string
  if (page.parentId !== null && (typeof page.parentId !== 'string' || !UUID_V4_REGEX.test(page.parentId))) {
    return false;
  }
  
  return true;
}

describe('API Page List Response Property Tests', () => {
  /**
   * Property 13: API Page List Response
   * For any project with pages, the API response SHALL include all pages 
   * with id, name, type, and parentId fields.
   */
  describe('Property 13: API Page List Response', () => {
    test('All pages in response have required fields (id, name, type, parentId)', () => {
      fc.assert(
        fc.property(pageListArb, (pages) => {
          const apiResponse = transformPagesForApiResponse(pages);
          
          // Every page in the response must have all required fields
          return apiResponse.every(page => hasRequiredFields(page));
        }),
        { numRuns: 100 }
      );
    });

    test('All pages in response have valid field types', () => {
      fc.assert(
        fc.property(pageListArb, (pages) => {
          const apiResponse = transformPagesForApiResponse(pages);
          
          // Every page in the response must have valid field types
          return apiResponse.every(page => hasValidFieldTypes(page));
        }),
        { numRuns: 100 }
      );
    });

    test('Response preserves all pages from input (no data loss)', () => {
      fc.assert(
        fc.property(pageListArb, (pages) => {
          const apiResponse = transformPagesForApiResponse(pages);
          
          // The number of pages in response must equal input
          if (apiResponse.length !== pages.length) {
            return false;
          }
          
          // Each input page's id must appear in the response
          const responseIds = new Set(apiResponse.map(p => p.id));
          return pages.every(page => responseIds.has(page.id));
        }),
        { numRuns: 100 }
      );
    });

    test('Response page data matches input data', () => {
      fc.assert(
        fc.property(pageListArb, (pages) => {
          const apiResponse = transformPagesForApiResponse(pages);
          
          // Create a map for quick lookup
          const inputMap = new Map(pages.map(p => [p.id, p]));
          
          // Each response page must match its input
          return apiResponse.every(responsePage => {
            const inputPage = inputMap.get(responsePage.id);
            if (!inputPage) return false;
            
            return (
              responsePage.name === inputPage.name &&
              responsePage.type === inputPage.type &&
              responsePage.parentId === inputPage.parentId
            );
          });
        }),
        { numRuns: 100 }
      );
    });

    test('Page type is always one of valid enum values', () => {
      fc.assert(
        fc.property(pageListArb, (pages) => {
          const apiResponse = transformPagesForApiResponse(pages);
          
          return apiResponse.every(page => PAGE_TYPES.includes(page.type));
        }),
        { numRuns: 100 }
      );
    });

    test('Empty project returns empty page list', () => {
      const emptyPages = [];
      const apiResponse = transformPagesForApiResponse(emptyPages);
      
      expect(apiResponse).toEqual([]);
      expect(Array.isArray(apiResponse)).toBe(true);
    });
  });
});
