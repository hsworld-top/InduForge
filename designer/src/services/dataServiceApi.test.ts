import { beforeEach, describe, expect, it, vi } from "vitest";
import { dataServiceApi } from "./dataServiceApi";
import request from "@/utils/request";

vi.mock("@/utils/request", () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
  },
}));

const requestMock = request as unknown as {
  get: ReturnType<typeof vi.fn>;
  post: ReturnType<typeof vi.fn>;
};

describe("dataServiceApi", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("按 data_service 路径分页查询数据点", async () => {
    requestMock.get.mockResolvedValueOnce({
      datapoints: [{ id: "dp-1", path: "a.b" }],
      pagination: { page: 2, pageSize: 20, total: 1 },
    });

    const result = await dataServiceApi.listDataPoints("project-1", {
      page: 2,
      pageSize: 20,
      search: "temp",
      status: "active",
      type: "mqtt",
      sourceId: "source-1",
    });

    expect(requestMock.get).toHaveBeenCalledWith("/data/projects/project-1/datapoints", {
      params: {
        page: 2,
        pageSize: 20,
        search: "temp",
        status: "active",
        type: "mqtt",
        sourceId: "source-1",
      },
    });
    expect(result.datapoints).toEqual([{ id: "dp-1", path: "a.b" }]);
    expect(result.pagination.total).toBe(1);
  });

  it("支持按 path 批量读取数据点值", async () => {
    requestMock.post.mockResolvedValueOnce({ values: { "a.b": 1 } });

    const result = await dataServiceApi.getDataPointValues("project-1", { paths: ["a.b"] });

    expect(requestMock.post).toHaveBeenCalledWith("/data/projects/project-1/datapoints/values", {
      paths: ["a.b"],
    });
    expect(result.values).toEqual({ "a.b": 1 });
  });
});
