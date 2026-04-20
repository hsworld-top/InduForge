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
    template: "<div><slot /><slot name=\"dropdown\" /></div>",
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
    template: "<select><slot /></select>",
  },
  "el-option": {
    template: "<option><slot /></option>",
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
    expect(wrapper.text()).toContain("主题");
    expect(wrapper.text()).toContain("语言");
    expect(wrapper.text()).toContain("预设尺寸");
    expect(wrapper.text()).toContain("自动保存");

    i18n.global.locale.value = "en";
    await nextTick();

    expect(wrapper.text()).toContain("Preview");
    expect(wrapper.text()).toContain("Save");
    expect(wrapper.text()).toContain("Undo");
    expect(wrapper.text()).toContain("Theme");
    expect(wrapper.text()).toContain("Language");
    expect(wrapper.text()).toContain("Preset Size");
    expect(wrapper.text()).toContain("Auto Save");
  });
});
