import { mount } from "@vue/test-utils";
import { nextTick } from "vue";
import { afterEach, describe, expect, it } from "vitest";
import { i18n } from "@/i18n";
import TopToolbar from "./TopToolbar.vue";

const sharedStubs = {
  "el-tooltip": {
    props: ["content"],
    template: "<div><slot />{{ content }}</div>",
  },
  "el-button": {
    template: "<button type=\"button\"><slot /></button>",
  },
  "el-dropdown": {
    props: ["hideOnClick"],
    template: "<div class=\"dropdown-stub\" :data-hide-on-click=\"String(hideOnClick)\"><slot /><slot name=\"dropdown\" /></div>",
  },
  "el-dropdown-menu": {
    template: "<div><slot /></div>",
  },
  "el-dropdown-item": {
    template: "<div><slot /></div>",
  },
  "el-popover": {
    template: "<div><slot /><slot name=\"reference\" /></div>",
  },
  "el-input-number": {
    template: "<input />",
  },
  "el-select": {
    props: ["teleported"],
    template: "<select :data-teleported=\"String(teleported)\"><slot /></select>",
  },
  "el-option": {
    props: ["label", "value"],
    template: "<option :value=\"value\">{{ label }}<slot /></option>",
  },
  "el-checkbox": {
    template: "<input type=\"checkbox\" />",
  },
  "el-button-group": {
    template: "<div><slot /></div>",
  },
};

describe("TopToolbar i18n", () => {
  afterEach(() => {
    i18n.global.locale.value = "zh";
  });

  it("会随 locale 切换更新关键按钮文案", async () => {
    const wrapper = mount(TopToolbar, {
      global: {
        plugins: [i18n],
        stubs: sharedStubs,
      },
      props: {
        viewPresets: [
          { key: "pc", label: "PC", width: 1366, height: 768 },
        ],
        saveSettings: {
          autoSave: true,
          intervalMinutes: 5,
        },
      },
    });

    expect(wrapper.text()).toContain("预览");
    expect(wrapper.text()).toContain("保存");
    expect(wrapper.text()).toContain("撤销");
    expect(wrapper.text()).not.toContain("主题");
    expect(wrapper.text()).not.toContain("语言");
    expect(wrapper.text()).toContain("预设尺寸");
    expect(wrapper.text()).toContain("自动保存");

    i18n.global.locale.value = "en";
    await nextTick();

    expect(wrapper.text()).toContain("Preview");
    expect(wrapper.text()).toContain("Save");
    expect(wrapper.text()).toContain("Undo");
    expect(wrapper.text()).not.toContain("Theme");
    expect(wrapper.text()).not.toContain("Language");
    expect(wrapper.text()).toContain("Preset Size");
    expect(wrapper.text()).toContain("Auto Save");
  });

  it("预览身份只展示登录账号，并保持内层选择器不传送", () => {
    const wrapper = mount(TopToolbar, {
      global: {
        plugins: [i18n],
        stubs: sharedStubs,
      },
      props: {
        viewPresets: [
          { key: "pc", label: "PC", width: 1366, height: 768 },
        ],
        saveSettings: {
          autoSave: true,
          intervalMinutes: 5,
        },
        runtimeUsers: [
          {
            id: "runtime-user-1",
            username: "operator",
            displayName: "值班员",
            status: "active",
          },
        ],
      },
    });

    expect(wrapper.text()).toContain("operator");
    expect(wrapper.text()).not.toContain("值班员");
    expect(
      wrapper.findAll(".dropdown-stub").some((item) => item.attributes("data-hide-on-click") === "false"),
    ).toBe(true);
    expect(
      wrapper.findAll("select").some((item) => item.attributes("data-teleported") === "false"),
    ).toBe(true);
  });
});
