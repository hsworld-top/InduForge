import { describe, expect, it } from "vitest";
import { normalizeOptions, normalizeTableColumns, normalizeTableData } from "./shared";

describe("ElementPlusCore renderer normalizers", () => {
  it("normalizes option arrays from objects and primitive values", () => {
    expect(
      normalizeOptions([
        { label: "启用", value: true },
        { label: "禁用", value: "disabled", disabled: true },
        "其他",
      ]),
    ).toEqual([
      { label: "启用", value: true, disabled: false },
      { label: "禁用", value: "disabled", disabled: true },
      { label: "其他", value: "其他" },
    ]);
  });

  it("drops invalid table columns and preserves optional render fields", () => {
    expect(
      normalizeTableColumns([
        { prop: "name", label: "名称", width: 120 },
        { prop: "", label: "无效" },
        { prop: "value", label: "数值", align: "right" },
      ]),
    ).toEqual([
      { prop: "name", label: "名称", width: 120 },
      { prop: "value", label: "数值", align: "right" },
    ]);
  });

  it("keeps only object rows for table data", () => {
    expect(normalizeTableData([{ name: "A" }, null, "bad", { name: "B" }])).toEqual([
      { name: "A" },
      { name: "B" },
    ]);
  });
});
