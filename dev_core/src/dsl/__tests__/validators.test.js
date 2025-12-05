/**
 * DSL Validators Unit Tests
 * Requirements: 1.1, 2.1, 4.1, 5.1
 */

const {
  validatePageSchema,
  validateComponentSchema,
  validateDataSourceSchema,
  validatePermissionsSchema,
  validateActionSchema,
  validateComponentAcl,
} = require('../validators');

describe('DSL Validators', () => {
  describe('validatePageSchema', () => {
    it('should return errors for missing required fields', () => {
      const result = validatePageSchema({});
      expect(result.valid).toBe(false);
      expect(result.errors.length).toBeGreaterThan(0);
      
      const missingFields = result.errors
        .filter(e => e.code === 'MISSING_FIELD')
        .map(e => e.path);
      
      expect(missingFields).toContain('version');
      expect(missingFields).toContain('meta');
      expect(missingFields).toContain('config');
      expect(missingFields).toContain('variables');
      expect(missingFields).toContain('dataSources');
      expect(missingFields).toContain('components');
      expect(missingFields).toContain('permissions');
    });

    it('should validate a complete valid page schema', () => {
      const validSchema = {
        version: '1.0.0',
        meta: { id: 'page_1', name: 'Test Page' },
        config: { width: 1920, height: 1080, scaleMode: 'fit', theme: 'dark' },
        variables: { isLoading: false },
        dataSources: [],
        components: [],
        permissions: { roles: ['admin'] },
      };
      
      const result = validatePageSchema(validSchema);
      expect(result.valid).toBe(true);
      expect(result.errors).toHaveLength(0);
    });

    it('should reject invalid scaleMode', () => {
      const schema = {
        version: '1.0.0',
        meta: {},
        config: { scaleMode: 'invalid' },
        variables: {},
        dataSources: [],
        components: [],
        permissions: { roles: [] },
      };
      
      const result = validatePageSchema(schema);
      expect(result.valid).toBe(false);
      expect(result.errors.some(e => e.path === 'config.scaleMode')).toBe(true);
    });
  });


  describe('validateComponentSchema', () => {
    it('should return errors for missing required fields', () => {
      const result = validateComponentSchema({});
      expect(result.valid).toBe(false);
      
      const missingFields = result.errors
        .filter(e => e.code === 'MISSING_FIELD')
        .map(e => e.path);
      
      expect(missingFields).toContain('id');
      expect(missingFields).toContain('type');
      expect(missingFields).toContain('label');
      expect(missingFields).toContain('locked');
      expect(missingFields).toContain('visible');
      expect(missingFields).toContain('style');
      expect(missingFields).toContain('props');
      expect(missingFields).toContain('bindings');
      expect(missingFields).toContain('events');
      expect(missingFields).toContain('animations');
      expect(missingFields).toContain('children');
    });

    it('should validate a complete valid component schema', () => {
      const validComponent = {
        id: 'comp_1',
        type: 'Button',
        label: 'Test Button',
        locked: false,
        visible: true,
        style: { left: 100, top: 200 },
        props: { text: 'Click me' },
        bindings: {},
        events: {},
        animations: [],
        children: [],
      };
      
      const result = validateComponentSchema(validComponent);
      expect(result.valid).toBe(true);
      expect(result.errors).toHaveLength(0);
    });
  });

  describe('validateDataSourceSchema', () => {
    it('should return errors for missing required fields', () => {
      const result = validateDataSourceSchema({});
      expect(result.valid).toBe(false);
      
      const missingFields = result.errors
        .filter(e => e.code === 'MISSING_FIELD')
        .map(e => e.path);
      
      expect(missingFields).toContain('id');
      expect(missingFields).toContain('type');
      expect(missingFields).toContain('mode');
    });

    it('should validate a valid dataCenter datasource', () => {
      const validDs = {
        id: 'ds_1',
        type: 'dataCenter',
        mode: 'subscription',
        queryId: 'query_123',
      };
      
      const result = validateDataSourceSchema(validDs);
      expect(result.valid).toBe(true);
    });

    it('should require pollingInterval when mode is poll', () => {
      const ds = {
        id: 'ds_1',
        type: 'http',
        mode: 'poll',
      };
      
      const result = validateDataSourceSchema(ds);
      expect(result.valid).toBe(false);
      expect(result.errors.some(e => e.path === 'pollingInterval')).toBe(true);
    });

    it('should reject invalid type', () => {
      const ds = {
        id: 'ds_1',
        type: 'invalid',
        mode: 'request',
      };
      
      const result = validateDataSourceSchema(ds);
      expect(result.valid).toBe(false);
      expect(result.errors.some(e => e.path === 'type')).toBe(true);
    });
  });

  describe('validatePermissionsSchema', () => {
    it('should validate a valid permissions schema', () => {
      const validPerms = {
        roles: ['admin', 'operator'],
        componentAcl: [
          {
            componentId: 'comp_1',
            visibleFor: ['admin'],
            editableFor: ['admin'],
          },
        ],
      };
      
      const result = validatePermissionsSchema(validPerms);
      expect(result.valid).toBe(true);
    });

    it('should require roles field', () => {
      const result = validatePermissionsSchema({});
      expect(result.valid).toBe(false);
      expect(result.errors.some(e => e.path === 'roles')).toBe(true);
    });
  });

  describe('validateActionSchema', () => {
    it('should validate valid action types', () => {
      const validActions = ['setVariable', 'executeQuery', 'navigate', 'openDialog', 'closeDialog', 'message', 'script'];
      
      validActions.forEach(actionType => {
        const result = validateActionSchema({ id: 'act_1', action: actionType });
        expect(result.valid).toBe(true);
      });
    });

    it('should reject invalid action type', () => {
      const result = validateActionSchema({ id: 'act_1', action: 'invalidAction' });
      expect(result.valid).toBe(false);
      expect(result.errors.some(e => e.path === 'action')).toBe(true);
    });
  });

  describe('validateComponentAcl', () => {
    it('should validate a valid component ACL', () => {
      const validAcl = {
        componentId: 'comp_1',
        visibleFor: ['admin', 'operator'],
        editableFor: ['admin'],
      };
      
      const result = validateComponentAcl(validAcl);
      expect(result.valid).toBe(true);
    });

    it('should require all three fields', () => {
      const result = validateComponentAcl({});
      expect(result.valid).toBe(false);
      
      const missingFields = result.errors
        .filter(e => e.code === 'MISSING_FIELD')
        .map(e => e.path);
      
      expect(missingFields).toContain('componentId');
      expect(missingFields).toContain('visibleFor');
      expect(missingFields).toContain('editableFor');
    });
  });
});
