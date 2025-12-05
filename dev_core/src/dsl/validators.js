/**
 * DSL Validators - Validation functions for DSL schemas
 * Requirements: 1.1, 2.1, 4.1, 5.1
 */

const {
  ScaleMode,
  Theme,
  ActionType,
  DataSourceType,
  DataSourceMode,
} = require('./constants');

/**
 * Creates a validation error object
 * @param {string} path - Path to the invalid field
 * @param {string} message - Error message
 * @param {string} [code] - Error code
 * @returns {Object} Validation error
 */
function createError(path, message, code = 'VALIDATION_ERROR') {
  return { path, message, code };
}

/**
 * Validates Page Schema structure
 * Requirements: 1.1
 * @param {Object} schema - Page schema to validate
 * @returns {{valid: boolean, errors: Array}} Validation result
 */
function validatePageSchema(schema) {
  const errors = [];

  if (!schema || typeof schema !== 'object') {
    return { valid: false, errors: [createError('', 'Schema must be an object')] };
  }

  // Check required top-level fields (Requirements 1.1)
  const requiredFields = ['version', 'meta', 'config', 'variables', 'dataSources', 'components', 'permissions'];
  for (const field of requiredFields) {
    if (!(field in schema)) {
      errors.push(createError(field, `Missing required field: ${field}`, 'MISSING_FIELD'));
    }
  }

  // Validate version
  if (schema.version !== undefined && typeof schema.version !== 'string') {
    errors.push(createError('version', 'version must be a string'));
  }

  // Validate meta
  if (schema.meta !== undefined) {
    if (typeof schema.meta !== 'object' || schema.meta === null) {
      errors.push(createError('meta', 'meta must be an object'));
    }
  }


  // Validate config
  if (schema.config !== undefined) {
    if (typeof schema.config !== 'object' || schema.config === null) {
      errors.push(createError('config', 'config must be an object'));
    } else {
      // Validate scaleMode enum
      if (schema.config.scaleMode !== undefined) {
        const validScaleModes = Object.values(ScaleMode);
        if (!validScaleModes.includes(schema.config.scaleMode)) {
          errors.push(createError('config.scaleMode', `scaleMode must be one of: ${validScaleModes.join(', ')}`));
        }
      }
      // Validate theme enum
      if (schema.config.theme !== undefined) {
        const validThemes = Object.values(Theme);
        if (!validThemes.includes(schema.config.theme)) {
          errors.push(createError('config.theme', `theme must be one of: ${validThemes.join(', ')}`));
        }
      }
    }
  }

  // Validate variables
  if (schema.variables !== undefined) {
    if (typeof schema.variables !== 'object' || schema.variables === null || Array.isArray(schema.variables)) {
      errors.push(createError('variables', 'variables must be an object'));
    }
  }

  // Validate dataSources
  if (schema.dataSources !== undefined) {
    if (!Array.isArray(schema.dataSources)) {
      errors.push(createError('dataSources', 'dataSources must be an array'));
    } else {
      schema.dataSources.forEach((ds, index) => {
        const dsResult = validateDataSourceSchema(ds);
        dsResult.errors.forEach(err => {
          errors.push(createError(`dataSources[${index}].${err.path}`, err.message, err.code));
        });
      });
    }
  }

  // Validate components
  if (schema.components !== undefined) {
    if (!Array.isArray(schema.components)) {
      errors.push(createError('components', 'components must be an array'));
    } else {
      schema.components.forEach((comp, index) => {
        const compResult = validateComponentSchema(comp);
        compResult.errors.forEach(err => {
          errors.push(createError(`components[${index}].${err.path}`, err.message, err.code));
        });
      });
    }
  }

  // Validate permissions
  if (schema.permissions !== undefined) {
    const permResult = validatePermissionsSchema(schema.permissions);
    permResult.errors.forEach(err => {
      errors.push(createError(`permissions.${err.path}`, err.message, err.code));
    });
  }

  return { valid: errors.length === 0, errors };
}


/**
 * Validates Component Schema structure
 * Requirements: 2.1
 * @param {Object} schema - Component schema to validate
 * @returns {{valid: boolean, errors: Array}} Validation result
 */
function validateComponentSchema(schema) {
  const errors = [];

  if (!schema || typeof schema !== 'object') {
    return { valid: false, errors: [createError('', 'Component schema must be an object')] };
  }

  // Check required fields (Requirements 2.1)
  const requiredFields = ['id', 'type', 'label', 'locked', 'visible', 'style', 'props', 'bindings', 'events', 'animations', 'children'];
  for (const field of requiredFields) {
    if (!(field in schema)) {
      errors.push(createError(field, `Missing required field: ${field}`, 'MISSING_FIELD'));
    }
  }

  // Validate id
  if (schema.id !== undefined && typeof schema.id !== 'string') {
    errors.push(createError('id', 'id must be a string'));
  }

  // Validate type
  if (schema.type !== undefined && typeof schema.type !== 'string') {
    errors.push(createError('type', 'type must be a string'));
  }

  // Validate label
  if (schema.label !== undefined && typeof schema.label !== 'string') {
    errors.push(createError('label', 'label must be a string'));
  }

  // Validate locked
  if (schema.locked !== undefined && typeof schema.locked !== 'boolean') {
    errors.push(createError('locked', 'locked must be a boolean'));
  }

  // Validate visible
  if (schema.visible !== undefined && typeof schema.visible !== 'boolean') {
    errors.push(createError('visible', 'visible must be a boolean'));
  }

  // Validate style
  if (schema.style !== undefined) {
    if (typeof schema.style !== 'object' || schema.style === null) {
      errors.push(createError('style', 'style must be an object'));
    }
  }

  // Validate props
  if (schema.props !== undefined) {
    if (typeof schema.props !== 'object' || schema.props === null) {
      errors.push(createError('props', 'props must be an object'));
    }
  }

  // Validate bindings
  if (schema.bindings !== undefined) {
    if (typeof schema.bindings !== 'object' || schema.bindings === null) {
      errors.push(createError('bindings', 'bindings must be an object'));
    }
  }

  // Validate events
  if (schema.events !== undefined) {
    if (typeof schema.events !== 'object' || schema.events === null) {
      errors.push(createError('events', 'events must be an object'));
    } else {
      // Validate each event's actions
      for (const [eventName, actions] of Object.entries(schema.events)) {
        if (!Array.isArray(actions)) {
          errors.push(createError(`events.${eventName}`, 'Event actions must be an array'));
        } else {
          actions.forEach((action, index) => {
            const actionResult = validateActionSchema(action);
            actionResult.errors.forEach(err => {
              errors.push(createError(`events.${eventName}[${index}].${err.path}`, err.message, err.code));
            });
          });
        }
      }
    }
  }

  // Validate animations
  if (schema.animations !== undefined) {
    if (!Array.isArray(schema.animations)) {
      errors.push(createError('animations', 'animations must be an array'));
    }
  }

  // Validate children recursively
  if (schema.children !== undefined) {
    if (!Array.isArray(schema.children)) {
      errors.push(createError('children', 'children must be an array'));
    } else {
      schema.children.forEach((child, index) => {
        const childResult = validateComponentSchema(child);
        childResult.errors.forEach(err => {
          errors.push(createError(`children[${index}].${err.path}`, err.message, err.code));
        });
      });
    }
  }

  return { valid: errors.length === 0, errors };
}


/**
 * Validates Action Schema structure
 * Requirements: 3.1
 * @param {Object} schema - Action schema to validate
 * @returns {{valid: boolean, errors: Array}} Validation result
 */
function validateActionSchema(schema) {
  const errors = [];

  if (!schema || typeof schema !== 'object') {
    return { valid: false, errors: [createError('', 'Action schema must be an object')] };
  }

  // Validate id
  if (schema.id !== undefined && typeof schema.id !== 'string') {
    errors.push(createError('id', 'id must be a string'));
  }

  // Validate action type (Requirements 3.1)
  if (schema.action === undefined) {
    errors.push(createError('action', 'Missing required field: action', 'MISSING_FIELD'));
  } else {
    const validActionTypes = Object.values(ActionType);
    if (!validActionTypes.includes(schema.action)) {
      errors.push(createError('action', `action must be one of: ${validActionTypes.join(', ')}`));
    }
  }

  // Validate payload
  if (schema.payload !== undefined && typeof schema.payload !== 'object') {
    errors.push(createError('payload', 'payload must be an object'));
  }

  // Validate condition
  if (schema.condition !== undefined && typeof schema.condition !== 'string') {
    errors.push(createError('condition', 'condition must be a string'));
  }

  return { valid: errors.length === 0, errors };
}

/**
 * Validates DataSource Schema structure
 * Requirements: 4.1, 4.2
 * @param {Object} schema - DataSource schema to validate
 * @returns {{valid: boolean, errors: Array}} Validation result
 */
function validateDataSourceSchema(schema) {
  const errors = [];

  if (!schema || typeof schema !== 'object') {
    return { valid: false, errors: [createError('', 'DataSource schema must be an object')] };
  }

  // Validate id
  if (schema.id === undefined) {
    errors.push(createError('id', 'Missing required field: id', 'MISSING_FIELD'));
  } else if (typeof schema.id !== 'string') {
    errors.push(createError('id', 'id must be a string'));
  }

  // Validate type (Requirements 4.1)
  if (schema.type === undefined) {
    errors.push(createError('type', 'Missing required field: type', 'MISSING_FIELD'));
  } else {
    const validTypes = Object.values(DataSourceType);
    if (!validTypes.includes(schema.type)) {
      errors.push(createError('type', `type must be one of: ${validTypes.join(', ')}`));
    }
  }

  // Validate mode (Requirements 4.2)
  if (schema.mode === undefined) {
    errors.push(createError('mode', 'Missing required field: mode', 'MISSING_FIELD'));
  } else {
    const validModes = Object.values(DataSourceMode);
    if (!validModes.includes(schema.mode)) {
      errors.push(createError('mode', `mode must be one of: ${validModes.join(', ')}`));
    }
  }

  // Validate pollingInterval when mode is 'poll' (Requirements 4.3)
  if (schema.mode === DataSourceMode.POLL) {
    if (schema.pollingInterval === undefined) {
      errors.push(createError('pollingInterval', 'pollingInterval is required when mode is poll', 'MISSING_FIELD'));
    } else if (typeof schema.pollingInterval !== 'number' || schema.pollingInterval <= 0) {
      errors.push(createError('pollingInterval', 'pollingInterval must be a positive number'));
    }
  }

  // Validate queryId for dataCenter type
  if (schema.type === DataSourceType.DATA_CENTER) {
    if (schema.queryId !== undefined && typeof schema.queryId !== 'string') {
      errors.push(createError('queryId', 'queryId must be a string'));
    }
  }

  // Validate dataHandler
  if (schema.dataHandler !== undefined && typeof schema.dataHandler !== 'string') {
    errors.push(createError('dataHandler', 'dataHandler must be a string'));
  }

  return { valid: errors.length === 0, errors };
}


/**
 * Validates Permissions Schema structure
 * Requirements: 5.1, 5.2
 * @param {Object} schema - Permissions schema to validate
 * @returns {{valid: boolean, errors: Array}} Validation result
 */
function validatePermissionsSchema(schema) {
  const errors = [];

  if (!schema || typeof schema !== 'object') {
    return { valid: false, errors: [createError('', 'Permissions schema must be an object')] };
  }

  // Validate roles (Requirements 5.1)
  if (schema.roles === undefined) {
    errors.push(createError('roles', 'Missing required field: roles', 'MISSING_FIELD'));
  } else if (!Array.isArray(schema.roles)) {
    errors.push(createError('roles', 'roles must be an array'));
  } else {
    schema.roles.forEach((role, index) => {
      if (typeof role !== 'string') {
        errors.push(createError(`roles[${index}]`, 'Each role must be a string'));
      }
    });
  }

  // Validate componentAcl (Requirements 5.2)
  if (schema.componentAcl !== undefined) {
    if (!Array.isArray(schema.componentAcl)) {
      errors.push(createError('componentAcl', 'componentAcl must be an array'));
    } else {
      schema.componentAcl.forEach((acl, index) => {
        const aclResult = validateComponentAcl(acl);
        aclResult.errors.forEach(err => {
          errors.push(createError(`componentAcl[${index}].${err.path}`, err.message, err.code));
        });
      });
    }
  }

  return { valid: errors.length === 0, errors };
}

/**
 * Validates Component ACL structure
 * Requirements: 5.2
 * @param {Object} acl - Component ACL to validate
 * @returns {{valid: boolean, errors: Array}} Validation result
 */
function validateComponentAcl(acl) {
  const errors = [];

  if (!acl || typeof acl !== 'object') {
    return { valid: false, errors: [createError('', 'Component ACL must be an object')] };
  }

  // Validate componentId
  if (acl.componentId === undefined) {
    errors.push(createError('componentId', 'Missing required field: componentId', 'MISSING_FIELD'));
  } else if (typeof acl.componentId !== 'string') {
    errors.push(createError('componentId', 'componentId must be a string'));
  }

  // Validate visibleFor
  if (acl.visibleFor === undefined) {
    errors.push(createError('visibleFor', 'Missing required field: visibleFor', 'MISSING_FIELD'));
  } else if (!Array.isArray(acl.visibleFor)) {
    errors.push(createError('visibleFor', 'visibleFor must be an array'));
  } else {
    acl.visibleFor.forEach((role, index) => {
      if (typeof role !== 'string') {
        errors.push(createError(`visibleFor[${index}]`, 'Each role must be a string'));
      }
    });
  }

  // Validate editableFor
  if (acl.editableFor === undefined) {
    errors.push(createError('editableFor', 'Missing required field: editableFor', 'MISSING_FIELD'));
  } else if (!Array.isArray(acl.editableFor)) {
    errors.push(createError('editableFor', 'editableFor must be an array'));
  } else {
    acl.editableFor.forEach((role, index) => {
      if (typeof role !== 'string') {
        errors.push(createError(`editableFor[${index}]`, 'Each role must be a string'));
      }
    });
  }

  return { valid: errors.length === 0, errors };
}

module.exports = {
  validatePageSchema,
  validateComponentSchema,
  validateActionSchema,
  validateDataSourceSchema,
  validatePermissionsSchema,
  validateComponentAcl,
  createError,
};
