import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

describe("designer dark theme variables", () => {
  it("为暗色主题覆盖核心壳层变量，避免只有局部控件变暗", () => {
    const css = readFileSync(resolve(__dirname, "./main.css"), "utf-8");
    const darkBlock = css.match(/\.dark\s*\{[\s\S]*?\n\s*\}/)?.[0] || "";

    expect(css).toMatch(/\.dark\s*\{/);
    expect(darkBlock).toMatch(/--designer-shell-bg:/);
    expect(darkBlock).toMatch(/--designer-shell-surface:/);
    expect(darkBlock).toMatch(/--designer-group-surface:/);
    expect(darkBlock).toMatch(/--designer-border-color:/);
    expect(darkBlock).toMatch(/--designer-text-primary:/);
  });
});
