import { describe, expect, it } from "vitest";
import {
  CANVAS_COL_INSERT_EDGE_THRESHOLD,
  CANVAS_DEFAULT_PAGE_MARGIN_X,
  CANVAS_RULER_MAJOR_STEP,
  CANVAS_RULER_MINOR_STEP,
} from "./canvas-container-constants";

describe("canvas-container-constants", () => {
  it("exports positive ruler steps", () => {
    expect(CANVAS_RULER_MINOR_STEP).toBeGreaterThan(0);
    expect(CANVAS_RULER_MAJOR_STEP).toBeGreaterThan(CANVAS_RULER_MINOR_STEP);
    expect(CANVAS_DEFAULT_PAGE_MARGIN_X).toBeGreaterThan(0);
    expect(CANVAS_COL_INSERT_EDGE_THRESHOLD).toBeGreaterThan(0);
  });
});
