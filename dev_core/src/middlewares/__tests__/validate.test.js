const Joi = require('joi');
const ErrorCodes = require('../../constants/errorCodes');
const { validate } = require('../validate');

function createMockResponse(req = {}) {
  const res = {
    req,
    locals: {
      requestId: req.requestId,
      language: req.language,
    },
    statusCode: 200,
    body: null,
    headers: {},
    status: jest.fn(function status(code) {
      res.statusCode = code;
      return res;
    }),
    json: jest.fn(function json(payload) {
      res.body = payload;
      return res;
    }),
    setHeader: jest.fn(function setHeader(key, value) {
      res.headers[key] = value;
    }),
  };
  return res;
}

describe('validate middleware', () => {
  test('参数校验失败时返回统一 JSON 响应契约', () => {
    const middleware = validate(Joi.object({
      query: Joi.object({
        limit: Joi.number().integer().max(100).required(),
      }).required(),
    }));
    const req = {
      requestId: 'req-validate-test',
      language: 'zh-CN',
      params: {},
      query: { limit: '1000' },
      body: {},
    };
    const res = createMockResponse(req);
    const next = jest.fn();

    middleware(req, res, next);

    expect(next).not.toHaveBeenCalled();
    expect(res.status).toHaveBeenCalledWith(400);
    expect(res.body).toEqual({
      code: ErrorCodes.toPublicCode(ErrorCodes.VALIDATION_FAILED),
      msg: 'Validation failed',
      data: null,
      reqId: 'req-validate-test',
    });
    expect(res.body).not.toHaveProperty('success');
    expect(res.body).not.toHaveProperty('error');
    expect(res.body).not.toHaveProperty('requestId');
  });
});
