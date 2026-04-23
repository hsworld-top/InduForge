import { afterEach, describe, expect, it, vi } from "vitest";
import { DataService } from "./DataService";
import { ApiRequestError } from "./types";

function buildJsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
  } as Response;
}

describe("DataService HTTP 响应兼容", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("fetchValue 遇到旧包络 success=false 时抛业务错误", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(buildJsonResponse({
      success: false,
      message: "读取失败",
    })));
    const service = new DataService({ baseUrl: "" });

    await expect(service.fetchValue("device.temp")).rejects.toMatchObject({
      message: "读取失败",
      isBusinessError: true,
    });
  });

  it("fetchValues/writeValue 遇到旧包络 success=false 时不会误判成功", async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(buildJsonResponse({
        success: false,
        message: "批量读取失败",
      }))
      .mockResolvedValueOnce(buildJsonResponse({
        success: false,
        message: "写入失败",
      }));
    vi.stubGlobal("fetch", fetchMock);
    const service = new DataService({ baseUrl: "" });

    await expect(service.fetchValues(["a", "b"])).rejects.toMatchObject({
      message: "批量读取失败",
      isBusinessError: true,
    });
    await expect(service.writeValue("a", 1)).rejects.toMatchObject({
      message: "写入失败",
      isBusinessError: true,
    });
  });

  it("2xx 且非 code/legacy 包络时按异常处理", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(buildJsonResponse({
      unexpected: true,
    })));
    const service = new DataService({ baseUrl: "" });

    await expect(service.fetchValue("device.temp")).rejects.toBeInstanceOf(ApiRequestError);
  });
});
