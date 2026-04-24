const Joi = require('joi');
const ErrorCodes = require('../constants/errorCodes');
const ApiResponse = require('../utils/response');

function validate(schema) {
  return (req, res, next) => {
    const toValidate = {
      params: req.params,
      query: req.query,
      body: req.body
    };
    const options = {
      abortEarly: false,
      allowUnknown: true,
      stripUnknown: true
    };
    const { error, value } = schema.validate(toValidate, options);
    if (error) {
      return ApiResponse.error(res, ErrorCodes.VALIDATION_FAILED, { message: 'Validation failed' }, 400);
    }
    req.params = value.params || req.params;
    req.query = value.query || req.query;
    req.body = value.body || req.body;
    next();
  };
}

module.exports = { validate };


