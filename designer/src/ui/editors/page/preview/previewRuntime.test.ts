import { afterEach, describe, expect, it } from "vitest";
import { clearPreviewRuntime, getPreviewRuntime, initPreviewRuntime } from "./previewRuntime";

describe("previewRuntime", () => {
  afterEach(() => {
    clearPreviewRuntime();
  });

  it("getPreviewRuntime returns null after clear", () => {
    clearPreviewRuntime();
    expect(getPreviewRuntime()).toBeNull();
  });

  it("init then get returns handle; clear resets singleton", () => {
    clearPreviewRuntime();
    const rt = initPreviewRuntime({
      projectId: "test-proj",
      projectVariables: {},
      globalScripts: {},
    });
    expect(rt).toBeTruthy();
    expect(getPreviewRuntime()).toBe(rt);
    clearPreviewRuntime();
    expect(getPreviewRuntime()).toBeNull();
  });

  it("second init without clear replaces singleton", () => {
    clearPreviewRuntime();
    const first = initPreviewRuntime({
      projectId: "a",
      projectVariables: {},
      globalScripts: {},
    });
    const second = initPreviewRuntime({
      projectId: "b",
      projectVariables: {},
      globalScripts: {},
    });
    expect(getPreviewRuntime()).toBe(second);
    expect(second).not.toBe(first);
  });

  it("init returns a handle with script runner API", () => {
    clearPreviewRuntime();
    const rt = initPreviewRuntime({
      projectId: "api-test",
      projectVariables: {},
      globalScripts: {},
    });
    expect(rt).toBeTruthy();
    expect(typeof rt.runCode).toBe("function");
    expect(typeof rt.start).toBe("function");
    expect(typeof rt.stop).toBe("function");
  });
});
