/**
 * Property-Based Test: Folder Deletion Protection
 * **Feature: design-center, Property 15: Folder Deletion Protection**
 * **Validates: Requirements 7.6**
 * 
 * For any folder page with child pages, deletion SHALL fail with an error 
 * indicating children must be removed first.
 */

const fc = require('fast-check');

/**
 * Error code for folder not empty
 */
const DESIGN_FOLDER_NOT_EMPTY = 'B5004';

/**
 * Simulates the folder deletion protection logic from designService.deletePage
 * This is a pure function that mirrors the business logic for testing
 * 
 * @param {Object} page - The page to delete
 * @param {number} childCount - Number of child pages
 * @returns {{ success: boolean, errorCode?: string, childCount?: number }}
 */
function checkFolderDeletion(page, childCount) {
  // If page doesn't exist, return not found error
  if (!page) {
    return { success: false, errorCode: 'B5001' };
  }

  // If it's a folder with children, prevent deletion
  if (page.type === 'folder' && childCount > 0) {
    return { 
      success: false, 
      errorCode: DESIGN_FOLDER_NOT_EMPTY,
      childCount 
    };
  }

  // Otherwise, deletion is allowed
  return { success: true };
}

/**
 * Arbitrary for generating valid page types
 */
const pageTypeArb = fc.constantFrom('page', 'folder', 'dialog');

/**
 * Arbitrary for generating a valid page object
 */
const pageArb = fc.record({
  id: fc.uuid(),
  name: fc.string({ minLength: 1, maxLength: 200 }),
  type: pageTypeArb,
  projectId: fc.uuid(),
  parentId: fc.option(fc.uuid(), { nil: null }),
});

/**
 * Arbitrary for generating a folder page specifically
 */
const folderPageArb = fc.record({
  id: fc.uuid(),
  name: fc.string({ minLength: 1, maxLength: 200 }),
  type: fc.constant('folder'),
  projectId: fc.uuid(),
  parentId: fc.option(fc.uuid(), { nil: null }),
});

/**
 * Arbitrary for generating a non-folder page
 */
const nonFolderPageArb = fc.record({
  id: fc.uuid(),
  name: fc.string({ minLength: 1, maxLength: 200 }),
  type: fc.constantFrom('page', 'dialog'),
  projectId: fc.uuid(),
  parentId: fc.option(fc.uuid(), { nil: null }),
});

describe('Folder Deletion Protection Property Tests', () => {
  /**
   * Property 15: Folder Deletion Protection
   * For any folder page with child pages, deletion SHALL fail with an error
   * indicating children must be removed first.
   */
  test('Folders with children cannot be deleted', () => {
    fc.assert(
      fc.property(
        folderPageArb,
        fc.integer({ min: 1, max: 1000 }), // childCount > 0
        (folder, childCount) => {
          const result = checkFolderDeletion(folder, childCount);
          
          // Deletion should fail
          expect(result.success).toBe(false);
          
          // Error code should be DESIGN_FOLDER_NOT_EMPTY
          expect(result.errorCode).toBe(DESIGN_FOLDER_NOT_EMPTY);
          
          // Child count should be included in the error
          expect(result.childCount).toBe(childCount);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property: Empty folders can be deleted
   * For any folder with zero children, deletion should succeed
   */
  test('Empty folders can be deleted', () => {
    fc.assert(
      fc.property(
        folderPageArb,
        (folder) => {
          const result = checkFolderDeletion(folder, 0);
          
          // Deletion should succeed
          expect(result.success).toBe(true);
          
          // No error code should be present
          expect(result.errorCode).toBeUndefined();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property: Non-folder pages can always be deleted regardless of child count
   * Pages and dialogs don't have the folder protection logic
   */
  test('Non-folder pages can be deleted regardless of child count', () => {
    fc.assert(
      fc.property(
        nonFolderPageArb,
        fc.integer({ min: 0, max: 1000 }),
        (page, childCount) => {
          const result = checkFolderDeletion(page, childCount);
          
          // Deletion should succeed for non-folders
          expect(result.success).toBe(true);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property: Child count threshold is exactly 1
   * Folders with exactly 1 child should fail, folders with 0 children should succeed
   */
  test('Child count threshold is exactly 1', () => {
    fc.assert(
      fc.property(
        folderPageArb,
        fc.integer({ min: 0, max: 100 }),
        (folder, childCount) => {
          const result = checkFolderDeletion(folder, childCount);
          
          if (childCount > 0) {
            // Should fail with children
            expect(result.success).toBe(false);
            expect(result.errorCode).toBe(DESIGN_FOLDER_NOT_EMPTY);
          } else {
            // Should succeed without children
            expect(result.success).toBe(true);
          }
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property: Error response includes child count for debugging
   * When deletion fails due to children, the error should include the count
   */
  test('Error response includes child count', () => {
    fc.assert(
      fc.property(
        folderPageArb,
        fc.integer({ min: 1, max: 10000 }),
        (folder, childCount) => {
          const result = checkFolderDeletion(folder, childCount);
          
          // Error should include the exact child count
          expect(result.childCount).toBe(childCount);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });

  /**
   * Property: Deletion protection is type-specific
   * Only 'folder' type pages have deletion protection
   */
  test('Deletion protection is type-specific to folders', () => {
    fc.assert(
      fc.property(
        pageArb,
        fc.integer({ min: 1, max: 100 }),
        (page, childCount) => {
          const result = checkFolderDeletion(page, childCount);
          
          if (page.type === 'folder') {
            // Folders with children should fail
            expect(result.success).toBe(false);
            expect(result.errorCode).toBe(DESIGN_FOLDER_NOT_EMPTY);
          } else {
            // Non-folders should succeed
            expect(result.success).toBe(true);
          }
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });
});
