jest.mock('../routes/register', () => ({
  registerVersionedRoutes: jest.fn(),
}));

const ErrorCodes = require('../constants/errorCodes');
const { buildApp } = require('../app');

/**
 * 构造最小化响应对象，用于直接调用错误中间件并验证契约字段。
 */
function createMockResponse(req) {
  const state = {
    statusCode: 200,
    payload: null,
    headers: {},
  };

  const res = {
    req,
    locals: {},
    setHeader: jest.fn((key, value) => {
      state.headers[key] = value;
    }),
    status: jest.fn((code) => {
      state.statusCode = code;
      return res;
    }),
    json: jest.fn((payload) => {
      state.payload = payload;
      return res;
    }),
  };

  return { res, state };
}

describe('app 全局兜底错误契约', () => {
  test('非 AppError/Joi 错误也仅返回 code/msg/data/reqId', () => {
    process.env.ENABLE_SWAGGER = 'false';
    process.env.NODE_ENV = 'test';

    const app = buildApp();
    const errorLayer = app.router.stack.find((layer) => typeof layer.handle === 'function' && layer.handle.length === 4);
    expect(errorLayer).toBeTruthy();

    const req = {
      language: 'zh-CN',
      requestId: 'req-unhandled',
    };
    const { res, state } = createMockResponse(req);
    const next = jest.fn();

    errorLayer.handle(new Error('forced unhandled error'), req, res, next);

    expect(state.statusCode).toBe(500);
    expect(state.payload).toEqual({
      code: ErrorCodes.toPublicCode(ErrorCodes.INTERNAL_SERVER_ERROR),
      msg: expect.any(String),
      data: null,
      reqId: 'req-unhandled',
    });
    expect(Object.keys(state.payload).sort()).toEqual(['code', 'data', 'msg', 'reqId']);
    expect(state.payload.success).toBeUndefined();
    expect(state.payload.errorCode).toBeUndefined();
    expect(state.payload.message).toBeUndefined();
    expect(state.payload.requestId).toBeUndefined();
    expect(state.payload.stack).toBeUndefined();
    expect(next).not.toHaveBeenCalled();
  });
});
