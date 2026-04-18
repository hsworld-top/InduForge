import { describe, expect, it } from "vitest";
import {
  collectMarqueeNodeIds,
  rectsIntersect,
  type MarqueeNodeLike,
  type MarqueeRectLike,
} from "./marquee-selection";

const marqueeRect: MarqueeRectLike = {
  left: 0,
  top: 0,
  right: 200,
  bottom: 200,
};

describe("marquee-selection", () => {
  it("应支持同级多命中时仍下钻容器子节点", () => {
    const nodes = new Map<string, MarqueeNodeLike>([
      ["root", { id: "root", type: "PageRoot", children: ["layout", "standalone"] }],
      ["layout", { id: "layout", type: "VerticalLayout", children: ["innerA", "innerB"] }],
      ["innerA", { id: "innerA", type: "Button", children: [] }],
      ["innerB", { id: "innerB", type: "Button", children: [] }],
      ["standalone", { id: "standalone", type: "Button", children: [] }],
    ]);

    const rects = new Map<string, MarqueeRectLike>([
      ["layout", { left: 10, top: 10, right: 150, bottom: 150 }],
      ["innerA", { left: 20, top: 20, right: 60, bottom: 50 }],
      ["innerB", { left: 20, top: 70, right: 60, bottom: 100 }],
      ["standalone", { left: 160, top: 20, right: 190, bottom: 50 }],
    ]);

    const result = collectMarqueeNodeIds({
      rootId: "root",
      marqueeRect,
      getNode: (id) => nodes.get(id) || null,
      getRect: (id) => rects.get(id) || null,
      isContainer: (type) => type === "VerticalLayout",
    });

    expect(result).toEqual(["innerA", "innerB", "standalone"]);
  });

  it("容器命中但无子节点命中时应回退选中容器自身", () => {
    const nodes = new Map<string, MarqueeNodeLike>([
      ["root", { id: "root", type: "PageRoot", children: ["layout"] }],
      ["layout", { id: "layout", type: "VerticalLayout", children: ["innerA"] }],
      ["innerA", { id: "innerA", type: "Button", children: [] }],
    ]);

    const rects = new Map<string, MarqueeRectLike>([
      ["layout", { left: 10, top: 10, right: 150, bottom: 150 }],
      ["innerA", { left: 260, top: 260, right: 300, bottom: 300 }],
    ]);

    const result = collectMarqueeNodeIds({
      rootId: "root",
      marqueeRect,
      getNode: (id) => nodes.get(id) || null,
      getRect: (id) => rects.get(id) || null,
      isContainer: (type) => type === "VerticalLayout",
    });

    expect(result).toEqual(["layout"]);
  });

  it("相交判断应按边界闭合处理", () => {
    expect(
      rectsIntersect(
        { left: 0, top: 0, right: 100, bottom: 100 },
        { left: 100, top: 100, right: 160, bottom: 160 },
      ),
    ).toBe(true);
  });
});

