import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

describe("vite data_service proxy", () => {
  it("优先把 /api/v1/data 代理到 VITE_DATA_SERVICE_URL", async () => {
    const configSource = readFileSync(resolve(__dirname, "../vite.config.js"), "utf-8");
    const dataProxyIndex = configSource.indexOf('"/api/v1/data"');
    const apiProxyIndex = configSource.indexOf('"/api"');

    expect(dataProxyIndex).toBeGreaterThanOrEqual(0);
    expect(configSource).toContain("target: env.VITE_DATA_SERVICE_URL");
    expect(dataProxyIndex).toBeLessThan(apiProxyIndex);
  });
});
