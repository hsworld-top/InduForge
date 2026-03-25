import { describe, expect, it } from "vitest";
import {
  applyTransforms,
  executeTransform,
  prefix,
  round,
  toFixed,
  transformRegistry,
} from "./transforms";

describe("transforms", () => {
  it("toFixed and round handle numeric strings", () => {
    expect(toFixed("3.14159", 2)).toBe("3.14");
    expect(round(3.6)).toBe(4);
  });

  it("prefix concatenates", () => {
    expect(prefix(42, "id:")).toBe("id:42");
  });

  it("executeTransform runs registry op with args", () => {
    expect(executeTransform(2, { op: "toFixed", args: [3] })).toBe("2.000");
  });

  it("applyTransforms chains ops", () => {
    const out = applyTransforms(5, [
      { op: "toFixed", args: [1] },
      { op: "prefix", args: ["v"] },
    ]);
    expect(out).toBe("v5.0");
  });

  it("transformRegistry exposes expected keys", () => {
    expect(typeof transformRegistry.clamp).toBe("function");
    expect(typeof transformRegistry.map).toBe("function");
  });
});
