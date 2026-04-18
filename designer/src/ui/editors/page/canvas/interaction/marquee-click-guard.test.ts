import { describe, expect, it } from "vitest";
import { createMarqueeClickGuard } from "./marquee-click-guard";

describe("marquee-click-guard", () => {
  it("默认不应抑制 click", () => {
    const guard = createMarqueeClickGuard();
    expect(guard.consumeShouldSuppressNextClick()).toBe(false);
  });

  it("标记后仅应抑制一次 click", () => {
    const guard = createMarqueeClickGuard();
    guard.markShouldSuppressNextClick();
    expect(guard.consumeShouldSuppressNextClick()).toBe(true);
    expect(guard.consumeShouldSuppressNextClick()).toBe(false);
  });

  it("reset 后应清空抑制状态", () => {
    const guard = createMarqueeClickGuard();
    guard.markShouldSuppressNextClick();
    guard.reset();
    expect(guard.consumeShouldSuppressNextClick()).toBe(false);
  });
});
