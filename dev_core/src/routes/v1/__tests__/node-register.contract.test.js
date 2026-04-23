const ErrorCodes = require("../../../constants/errorCodes");

jest.mock("../../../services/nodeService", () => ({
  registerWithAuth: jest.fn(),
  getApprovalStatus: jest.fn(),
}));

const nodeService = require("../../../services/nodeService");
const nodeRegisterRouter = require("../node-register");

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

function getRouteHandler(path, method) {
  const routeLayer = nodeRegisterRouter.stack.find(
    (layer) => layer.route && layer.route.path === path && layer.route.methods[method]
  );

  if (!routeLayer) {
    throw new Error(`未找到路由: ${method.toUpperCase()} ${path}`);
  }

  return routeLayer.route.stack[routeLayer.route.stack.length - 1].handle;
}

describe("node-register 路由统一响应契约", () => {
  const registerHandler = getRouteHandler("/register-with-auth", "post");

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test("参数校验失败返回统一四字段且不再包含旧 success/error 字段", async () => {
    const req = {
      body: {
        password: "123456",
        nodeName: "node-demo",
      },
      language: "zh-CN",
      requestId: "req-node-register-validation",
      get: jest.fn(() => ""),
      ip: "127.0.0.1",
    };
    const { res, state } = createMockResponse(req);

    await registerHandler(req, res);

    expect(state.statusCode).toBe(200);
    expect(state.payload).toEqual({
      code: ErrorCodes.toPublicCode(ErrorCodes.VALIDATION_FAILED),
      msg: "用户名和密码不能为空",
      data: null,
      reqId: "req-node-register-validation",
    });
    expect(state.payload.success).toBeUndefined();
    expect(state.payload.error).toBeUndefined();
    expect(state.payload.errorCode).toBeUndefined();
    expect(state.payload.requestId).toBeUndefined();
    expect(nodeService.registerWithAuth).not.toHaveBeenCalled();
  });

  test("注册成功返回统一四字段契约", async () => {
    nodeService.registerWithAuth.mockResolvedValueOnce({
      nodeId: "node-1",
      approvalStatus: "pending",
    });

    const req = {
      body: {
        username: "operator",
        password: "123456",
        nodeName: "node-demo",
      },
      language: "zh-CN",
      requestId: "req-node-register-success",
      get: jest.fn((header) => {
        if (String(header).toLowerCase() === "user-agent") {
          return "jest-test";
        }
        return "";
      }),
      ip: "127.0.0.1",
    };
    const { res, state } = createMockResponse(req);

    await registerHandler(req, res);

    expect(state.statusCode).toBe(201);
    expect(state.payload).toEqual({
      code: ErrorCodes.toPublicCode(ErrorCodes.SUCCESS),
      msg: expect.any(String),
      data: {
        nodeId: "node-1",
        approvalStatus: "pending",
      },
      reqId: "req-node-register-success",
    });
    expect(state.payload.success).toBeUndefined();
    expect(state.payload.error).toBeUndefined();
    expect(state.payload.errorCode).toBeUndefined();
    expect(state.payload.requestId).toBeUndefined();
  });
});
