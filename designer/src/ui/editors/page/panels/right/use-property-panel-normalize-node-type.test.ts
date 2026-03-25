import type { Ref } from "vue";
import type { ComponentNode } from "@/editor-core/document/types";
import { describe, expect, it } from "vitest";
import { ref } from "vue";
import { usePropertyPanelNormalizeNodeType } from "./use-property-panel-normalize-node-type";

type SelectedNodeSlice = Pick<ComponentNode, "id" | "type">;

describe("usePropertyPanelNormalizeNodeType", () => {
  it("calls updateNode when trimmed type differs from raw", async () => {
    const selectedNode: Ref<SelectedNodeSlice | null> = ref({
      id: "n1",
      type: "  ElButton  ",
    });
    const calls: [string, { type?: string }][] = [];
    usePropertyPanelNormalizeNodeType(selectedNode, (id, patch) => {
      calls.push([id, patch]);
    });
    await Promise.resolve();
    expect(calls.some(([id, patch]) => id === "n1" && patch.type === "ElButton")).toBe(true);
  });
});
