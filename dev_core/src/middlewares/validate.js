const Joi = require('joi');

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
      const details = error.details.map(d => ({ message: d.message, path: d.path }));
      return res.status(400).json({
        success: false,
        error: { message: 'Validation failed', details },
        requestId: req.requestId
      });
    }
    req.params = value.params || req.params;
    req.query = value.query || req.query;
    req.body = value.body || req.body;
    next();
  };
}

module.exports = { validate };


