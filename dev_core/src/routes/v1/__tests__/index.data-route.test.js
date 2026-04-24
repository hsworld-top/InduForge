const buildMockRoute = (name) => {
  const express = require("express");
  const router = express.Router();
  router.get("/ping", (req, res) => {
    res.status(200).json({ route: name });
  });
  return router;
};

jest.mock("../auth", () => buildMockRoute("auth"));
jest.mock("../tenant", () => buildMockRoute("tenant"));
jest.mock("../user", () => buildMockRoute("user"));
jest.mock("../project", () => buildMockRoute("project"));
jest.mock("../log", () => buildMockRoute("log"));
jest.mock("../role", () => buildMockRoute("role"));
jest.mock("../design", () => buildMockRoute("design"));
jest.mock("../designAssets", () => buildMockRoute("designAssets"));
jest.mock("../pageLock", () => buildMockRoute("pageLock"));
jest.mock("../node", () => buildMockRoute("node"));
jest.mock("../node-register", () => buildMockRoute("node-register"));
jest.mock("../deployment", () => buildMockRoute("deployment"));
jest.mock("../publish", () => buildMockRoute("publish"));
jest.mock("../../../utils/logger", () => ({
  logger: {
    info: jest.fn(),
  },
}));
jest.mock("../../../middlewares/operationAuditLog", () => ({
  operationAuditLog: (req, res, next) => next(),
}));

const { buildV1Router } = require("../index");

function hasMountedPrefix(router, url) {
  return router.stack.some(
    (layer) =>
      Array.isArray(layer.matchers) &&
      layer.matchers.some((matcher) => Boolean(matcher(url)))
  );
}

describe("v1 router data route removal", () => {
  test("不再挂载 /data 历史数据域路由", () => {
    const router = buildV1Router();

    expect(hasMountedPrefix(router, "/projects/ping")).toBe(true);
    expect(hasMountedPrefix(router, "/data/ping")).toBe(false);
  });
});
