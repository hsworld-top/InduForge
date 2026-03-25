import { createPinia, setActivePinia } from "pinia";
import { beforeEach, describe, expect, it } from "vitest";
import { useEditorStore } from "./editor-store";

describe("editor-store page tabs", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
  });

  it("setPageTabState normalizes tabs and activeId", () => {
    const store = useEditorStore();
    const tabs = [{ id: "p1", name: "首页", isDirty: true }];
    store.setPageTabState(tabs, "p1");
    expect(store.pageTabState.activeId).toBe("p1");
    expect(store.pageTabState.tabs).toHaveLength(1);
    expect(store.pageTabState.tabs[0]).toEqual(tabs[0]);
    expect(store.pageTabState.tabs[0]).not.toBe(tabs[0]);
  });

  it("setPageTabState coerces empty activeId to string", () => {
    const store = useEditorStore();
    store.setPageTabState([], "");
    expect(store.pageTabState.activeId).toBe("");
  });
});
