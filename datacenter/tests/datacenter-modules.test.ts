import { describe, expect, test } from "vitest";

import {
  datacenterModules,
  getDatacenterModule,
} from "../src/config/datacenterModules";

describe("datacenterModules", () => {
  test("按设计顺序提供四个数据中心模块", () => {
    expect(datacenterModules.map((item) => item.id)).toEqual([
      "datapoints",
      "access-sources",
      "compute-units",
      "alarm-units",
    ]);
  });

  test("getDatacenterModule 根据模块 ID 返回元数据", () => {
    expect(getDatacenterModule("datapoints")?.label).toBe("数据点");
    expect(getDatacenterModule("compute-units")?.label).toBe("计算单元");
    expect(getDatacenterModule("alarm-units")?.label).toBe("报警单元");
    expect(getDatacenterModule("missing")).toBeNull();
  });
});
