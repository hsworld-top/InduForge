jest.mock("../../models", () => ({
  DesignPage: {
    findAll: jest.fn(),
    findByPk: jest.fn(),
  },
  Project: {
    findByPk: jest.fn(),
    sequelize: {},
  },
}));

jest.mock("../../config/database", () => ({
  sequelize: {
    transaction: jest.fn(),
  },
}));

jest.mock("../../dsl/validators", () => ({
  validatePageSchema: jest.fn(),
}));

const { DesignPage, Project } = require("../../models");
const designService = require("../designService");

describe("DesignService.getPages PostgreSQL 排序兼容", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  test("排序表达式应显式引用 parentId，避免 PostgreSQL 将其降为 parentid", async () => {
    Project.findByPk.mockResolvedValue({
      id: "project-1",
      entryConfig: {},
    });
    DesignPage.findAll.mockResolvedValue([]);

    await designService.getPages("project-1");

    expect(DesignPage.findAll).toHaveBeenCalledWith(
      expect.objectContaining({
        order: expect.arrayContaining([
          [expect.objectContaining({ val: "\"parentId\" IS NOT NULL" }), "ASC"],
        ]),
      }),
    );
  });
});
