import { mount } from "@vue/test-utils";
import { nextTick } from "vue";
import { afterEach, describe, expect, it } from "vitest";
import { i18n } from "@/i18n";
import PropertyPanelBasicSection from "./PropertyPanelBasicSection.vue";

const sharedStubs = {
  "el-input": {
    props: ["modelValue", "placeholder", "disabled"],
    template:
      "<div class=\"el-input-stub\" :data-placeholder=\"placeholder\" :data-disabled=\"disabled\"><slot />{{ modelValue }}</div>",
  },
};

describe("PropertyPanelBasicSection i18n", () => {
  afterEach(() => {
    i18n.global.locale.value = "zh";
  });

  it("会随 locale 切换更新基础属性文案", async () => {
    const wrapper = mount(PropertyPanelBasicSection, {
      global: {
        plugins: [i18n],
        stubs: sharedStubs,
      },
      props: {
        elementType: "Button",
        elementId: "node-1",
        currentStyle: {
          left: "12px",
          top: "24px",
        },
        label: "按钮",
        description: "描述",
      },
    });

    expect(wrapper.text()).toContain("基本");
    expect(wrapper.text()).toContain("名称");
    expect(wrapper.text()).toContain("描述");
    expect(wrapper.text()).toContain("类型");
    expect(wrapper.text()).toContain("位置");
    expect(wrapper.html()).toContain("未命名");
    expect(wrapper.html()).toContain("请输入描述");

    i18n.global.locale.value = "en";
    await nextTick();

    expect(wrapper.text()).toContain("Basic");
    expect(wrapper.text()).toContain("Name");
    expect(wrapper.text()).toContain("Description");
    expect(wrapper.text()).toContain("Type");
    expect(wrapper.text()).toContain("Position");
    expect(wrapper.html()).toContain("Untitled");
    expect(wrapper.html()).toContain("Enter description");
  });
});
