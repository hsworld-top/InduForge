import { mount } from "@vue/test-utils";
import { describe, expect, it, vi } from "vitest";
import PageTreeBasicPagesSection from "./PageTreeBasicPagesSection.vue";

vi.mock("~icons/ep/document", () => ({
  default: {
    name: "IconEpDocument",
    template: `<span class="icon-ep-document" />`,
  },
}));

vi.mock("~icons/ep/more-filled", () => ({
  default: {
    name: "IconEpMoreFilled",
    template: `<span class="icon-ep-more-filled" />`,
  },
}));

const ElDropdownStub = {
  name: "ElDropdown",
  emits: ["command"],
  template: `<div class="el-dropdown-stub"><slot /><slot name="dropdown" /></div>`,
};

const ElDropdownMenuStub = {
  name: "ElDropdownMenu",
  template: `<div class="el-dropdown-menu-stub"><slot /></div>`,
};

const ElDropdownItemStub = {
  name: "ElDropdownItem",
  props: ["command"],
  template: `<button class="el-dropdown-item-stub" :data-command="command"><slot /></button>`,
};

const ElButtonStub = {
  name: "ElButton",
  template: `<button class="el-button-stub" @click="$emit('click', $event)"><slot /></button>`,
};

describe("pageTreeBasicPagesSection", () => {
  it("空槽位展示菜单创建提示，且单击卡片不直接触发创建", async () => {
    const basicSlot = { type: "login", label: "登录页", page: null };
    const wrapper = mount(PageTreeBasicPagesSection, {
      props: {
        basicSlots: [basicSlot],
        isPageActive: () => false,
        creatingBasicType: null,
      },
      global: {
        stubs: {
          ElDropdown: ElDropdownStub,
          ElDropdownMenu: ElDropdownMenuStub,
          ElDropdownItem: ElDropdownItemStub,
          ElButton: ElButtonStub,
          IconEpDocument: true,
          IconEpMoreFilled: true,
        },
      },
    });

    expect(wrapper.text()).toContain("通过右侧菜单创建固定入口页");
    expect(wrapper.text()).toContain("创建登录页");

    await wrapper.find(".page-node--basic").trigger("click");

    expect(wrapper.emitted("slotClick")).toHaveLength(1);
    expect(wrapper.emitted("rowAction")).toBeUndefined();
  });

  it("下拉菜单 command 事件会正确透传 rowAction", async () => {
    const basicSlot = { type: "logout", label: "登出页", page: null };
    const wrapper = mount(PageTreeBasicPagesSection, {
      props: {
        basicSlots: [basicSlot],
        isPageActive: () => false,
        creatingBasicType: null,
      },
      global: {
        stubs: {
          ElDropdown: ElDropdownStub,
          ElDropdownMenu: ElDropdownMenuStub,
          ElDropdownItem: ElDropdownItemStub,
          ElButton: ElButtonStub,
          IconEpDocument: true,
          IconEpMoreFilled: true,
        },
      },
    });

    const dropdown = wrapper.findComponent(ElDropdownStub as any);
    dropdown.vm.$emit("command", "create");
    await wrapper.vm.$nextTick();

    expect(wrapper.emitted("rowAction")).toEqual([["create", basicSlot]]);
  });
});
