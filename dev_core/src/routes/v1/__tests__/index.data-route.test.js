const express = require("express");
const request = require("supertest");

const buildMockRoute = (name) => {
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

describe("v1 router data route removal", () => {
  test("不再挂载 /data 历史数据域路由", async () => {
    const app = express();
    app.use(buildV1Router());

    const projectResponse = await request(app).get("/projects/ping");
    const dataResponse = await request(app).get("/data/ping");

    expect(projectResponse.status).toBe(200);
    expect(projectResponse.body).toEqual({ route: "project" });
    expect(dataResponse.status).toBe(404);
  });
});
