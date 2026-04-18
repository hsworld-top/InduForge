import { describe, expect, it } from "vitest";
import {
  shouldClearSelectionOnMarqueeUp,
  shouldStartMarqueeFromCanvasPointerDown,
} from "./marquee-interaction";

describe("marquee-interaction", () => {
  it("命中根节点时应允许启动框选", () => {
    expect(
      shouldStartMarqueeFromCanvasPointerDown({
        hasNodeElement: true,
        isRootNode: true,
      }),
    ).toBe(true);
  });

  it("命中非根节点时不应启动框选", () => {
    expect(
      shouldStartMarqueeFromCanvasPointerDown({
        hasNodeElement: true,
        isRootNode: false,
      }),
    ).toBe(false);
  });

  it("命中非节点空白区时应允许启动框选", () => {
    expect(
      shouldStartMarqueeFromCanvasPointerDown({
        hasNodeElement: false,
        isRootNode: false,
      }),
    ).toBe(true);
  });

  it("页面外起手且未移动时应清空选中", () => {
    expect(
      shouldClearSelectionOnMarqueeUp({
        startSource: "outsideCanvas",
        moved: false,
      }),
    ).toBe(true);
  });

  it("页面外起手且已移动时不应强制清空选中", () => {
    expect(
      shouldClearSelectionOnMarqueeUp({
        startSource: "outsideCanvas",
        moved: true,
      }),
    ).toBe(false);
  });

  it("页面内起手且未移动时保持原有行为", () => {
    expect(
      shouldClearSelectionOnMarqueeUp({
        startSource: "insideCanvas",
        moved: false,
      }),
    ).toBe(false);
  });
});
