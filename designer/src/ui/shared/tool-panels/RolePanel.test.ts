import { describe, expect, it } from "vitest";
import { mount } from "@vue/test-utils";
import RolePanel from "./RolePanel.vue";
import * as toolPanels from "./index";

describe("RolePanel", () => {
  it("renders the degraded maintenance guidance", () => {
    const wrapper = mount(RolePanel);

    expect(wrapper.text()).toContain("角色维护已迁移");
    expect(wrapper.text()).toContain("成员与权限");
    expect(wrapper.text()).toContain("页面与动作的就近授权配置");
  });

  it("remains exported from shared tool panels", () => {
    expect(toolPanels.RolePanel).toBe(RolePanel);
  });
});
